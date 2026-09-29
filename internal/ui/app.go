// Package ui implements the WezTerm Configurator application shell and
// option editors.
package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"weztermconfigurator/internal/catalog"
	"weztermconfigurator/internal/lint"
	"weztermconfigurator/internal/luagen"
	"weztermconfigurator/internal/state"
	"weztermconfigurator/internal/wezcli"
)

// appVersion is the release this build belongs to; shown in the About dialog.
const appVersion = "1.6.0"

// Preference keys for the UI state that survives a restart.
const (
	prefNavCat  = "nav.cat"
	prefShowAll = "view.all"
	prefChanged = "view.changed"
	prefPreview = "preview.on"
)

// appState is the single application instance shared by all editors.
type appState struct {
	app         fyne.App
	win         fyne.Window
	paths       state.Paths
	st          *state.State
	dirty       bool
	wezterm     string
	systemFonts []string
	target      string
	showAll     bool
	uiScale     float32
	changedOnly bool // hide options that have no value set
	hist        *history
	undoBtn     *widget.Button
	redoBtn     *widget.Button
	page        *fyne.Container
	pageScroll  *fastScroll
	nav         *widget.List
	search      *widget.Entry
	previewOn   bool
	preview     *previewPane      // live config preview column (nil when unavailable)
	previewCol  fyne.CanvasObject // wrapper for the column, hidden when previewOn is false
	previewBtn  *widget.Button    // top-bar toggle
	status      *widget.Label
	pathLabel   *widget.Label

	rows        []*row // currently mounted option rows
	wroteOK     bool
	currentCat  string
	searchQuery string
	savedFirst  bool // first save this session (ownership prompt)
}

var platformLabels = []string{"Linux", "Windows", "macOS"}

func platformToTarget(label string) string {
	switch label {
	case "Windows":
		return "windows"
	case "macOS":
		return "macos"
	default:
		return "linux"
	}
}

func targetToPlatform(t string) string {
	switch t {
	case "windows":
		return "Windows"
	case "macos":
		return "macOS"
	default:
		return "Linux"
	}
}

// Run creates the window and starts the app loop.
func Run() {
	a := app.NewWithID("io.github.wadhah.weztermconfigurator")
	savedScale := float32(a.Preferences().FloatWithFallback("ui.scale", 1.0))
	if savedScale < 0.7 || savedScale > 2.0 {
		savedScale = 1.0
	}
	a.Settings().SetTheme(newAppTheme(savedScale))
	w := a.NewWindow("WezTerm Configurator")
	initW := float32(a.Preferences().FloatWithFallback("win.w", 1240))
	initH := float32(a.Preferences().FloatWithFallback("win.h", 780))
	w.Resize(fyne.NewSize(initW, initH))
	w.CenterOnScreen()
	paths, err := state.Resolve()
	if err != nil {
		dialog.ShowError(fmt.Errorf("resolving WezTerm paths: %w", err), w)
	}
	st, err := state.LoadOrRecover(paths.State, paths.Config)
	if err != nil {
		dialog.ShowError(fmt.Errorf("loading state: %w", err), w)
		st = state.New(state.RuntimeTarget())
	}

	prefs := a.Preferences()
	a2 := &appState{
		hist:        newHistory(st),
		app:         a,
		win:         w,
		paths:       paths,
		st:          st,
		target:      st.TargetOS,
		uiScale:     savedScale,
		showAll:     prefs.BoolWithFallback(prefShowAll, false),
		changedOnly: prefs.BoolWithFallback(prefChanged, false),
		previewOn:   prefs.BoolWithFallback(prefPreview, true),
	}

	if bin, ok := wezcli.Find(); ok {
		a2.wezterm = bin
		go func() {
			fonts, ferr := wezcli.ListSystemFonts(bin)
			if ferr == nil {
				fyne.Do(func() { a2.systemFonts = fonts })
			}
		}()
	}

	w.SetContent(a2.buildUI())
	a2.currentCat = catalog.QuickCategory
	if cat := prefs.StringWithFallback(prefNavCat, catalog.QuickCategory); validCategory(cat) {
		a2.currentCat = cat
	}
	a2.rebuildPage()
	a2.refreshNav()
	a2.selectNav()
	a2.updatePreview()
	a2.setTitle()

	saveShortcut := &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl}
	saveShortcutSuper := &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierSuper}
	w.Canvas().AddShortcut(saveShortcut, func(fyne.Shortcut) { a2.save() })
	w.Canvas().AddShortcut(saveShortcutSuper, func(fyne.Shortcut) { a2.save() })

	openShortcut := &desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: fyne.KeyModifierControl}
	openShortcutSuper := &desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: fyne.KeyModifierSuper}
	w.Canvas().AddShortcut(openShortcut, func(fyne.Shortcut) { a2.openFile() })
	w.Canvas().AddShortcut(openShortcutSuper, func(fyne.Shortcut) { a2.openFile() })
	zoomIn := func() { a2.adjustScale(0.1) }
	zoomOut := func() { a2.adjustScale(-0.1) }
	zoomReset := func() { a2.setScale(1.0) }

	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyEqual, Modifier: fyne.KeyModifierControl}, func(fyne.Shortcut) { zoomIn() })
	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyEqual, Modifier: fyne.KeyModifierSuper}, func(fyne.Shortcut) { zoomIn() })
	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyPlus, Modifier: fyne.KeyModifierControl}, func(fyne.Shortcut) { zoomIn() })
	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyPlus, Modifier: fyne.KeyModifierSuper}, func(fyne.Shortcut) { zoomIn() })
	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyMinus, Modifier: fyne.KeyModifierControl}, func(fyne.Shortcut) { zoomOut() })
	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyMinus, Modifier: fyne.KeyModifierSuper}, func(fyne.Shortcut) { zoomOut() })
	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.Key0, Modifier: fyne.KeyModifierControl}, func(fyne.Shortcut) { zoomReset() })
	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.Key0, Modifier: fyne.KeyModifierSuper}, func(fyne.Shortcut) { zoomReset() })

	findShortcut := &desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierControl}
	findShortcutSuper := &desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierSuper}
	w.Canvas().AddShortcut(findShortcut, func(fyne.Shortcut) { a2.focusSearch() })
	w.Canvas().AddShortcut(findShortcutSuper, func(fyne.Shortcut) { a2.focusSearch() })
	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyEscape}, func(fyne.Shortcut) { a2.escapeSearch() })

	helpShortcut := &desktop.CustomShortcut{KeyName: fyne.KeyF1}
	helpSlash := &desktop.CustomShortcut{KeyName: fyne.KeySlash, Modifier: fyne.KeyModifierControl}
	helpSlashSuper := &desktop.CustomShortcut{KeyName: fyne.KeySlash, Modifier: fyne.KeyModifierSuper}
	for _, s := range []*desktop.CustomShortcut{helpShortcut, helpSlash, helpSlashSuper} {
		sc := s
		w.Canvas().AddShortcut(sc, func(fyne.Shortcut) { a2.showShortcuts() })
	}

	for _, s := range []*desktop.CustomShortcut{
		{KeyName: fyne.KeyZ, Modifier: fyne.KeyModifierControl},
		{KeyName: fyne.KeyZ, Modifier: fyne.KeyModifierSuper},
	} {
		w.Canvas().AddShortcut(s, func(fyne.Shortcut) { a2.undo() })
	}
	for _, s := range []*desktop.CustomShortcut{
		{KeyName: fyne.KeyY, Modifier: fyne.KeyModifierControl},
		{KeyName: fyne.KeyY, Modifier: fyne.KeyModifierSuper},
		{KeyName: fyne.KeyZ, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift},
		{KeyName: fyne.KeyZ, Modifier: fyne.KeyModifierSuper | fyne.KeyModifierShift},
	} {
		w.Canvas().AddShortcut(s, func(fyne.Shortcut) { a2.redo() })
	}

	quitItem := fyne.NewMenuItem("Quit", func() {
		a2.persistWindowSize()
		w.Close()
	})
	quitItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyQ, Modifier: fyne.KeyModifierControl}
	saveItem := fyne.NewMenuItem("Save & Apply", func() { a2.save() })
	saveItem.Shortcut = saveShortcut
	openItem := fyne.NewMenuItem("Open…", a2.openFile)
	openItem.Shortcut = openShortcut
	fileMenu := fyne.NewMenu("File",
		saveItem,
		openItem,
		fyne.NewMenuItem("Export Lua…", a2.exportLua),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Import existing wezterm.lua…", a2.importLua),
		fyne.NewMenuItem("Profiles…", a2.showProfilesDialog),
		fyne.NewMenuItem("Restore backup…", a2.showRestoreDialog),
		fyne.NewMenuItemSeparator(),
		quitItem,
	)
	undoItem := fyne.NewMenuItem("Undo", a2.undo)
	undoItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyZ, Modifier: fyne.KeyModifierControl}
	redoItem := fyne.NewMenuItem("Redo", a2.redo)
	redoItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyY, Modifier: fyne.KeyModifierControl}
	editMenu := fyne.NewMenu("Edit",
		undoItem,
		redoItem,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Review changes…", a2.showReview),
	)
	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("Keyboard shortcuts", a2.showShortcuts),
		fyne.NewMenuItem("About", a2.showAbout),
	)
	helpMenu.Items[0].Shortcut = helpShortcut
	mainMenu := fyne.NewMainMenu(fileMenu, editMenu, helpMenu)
	w.SetMainMenu(mainMenu)
	w.SetCloseIntercept(func() {
		if !a2.dirty {
			a2.persistWindowSize()
			w.Close()
			return
		}
		d := dialog.NewCustom("Unsaved changes", "Cancel",
			widget.NewLabel("Save changes before closing?"), w)
		d.SetButtons([]fyne.CanvasObject{
			widget.NewButton("Save", func() {
				d.Hide()
				a2.saveThen(func() {
					a2.persistWindowSize()
					w.Close()
				})
			}),
			widget.NewButton("Discard", func() {
				d.Hide()
				a2.persistWindowSize()
				w.Close()
			}),
			widget.NewButton("Cancel", d.Hide),
		})
		d.Show()
	})

	w.ShowAndRun()
}

func (a *appState) persistWindowSize() {
	a.app.Preferences().SetFloat("win.w", float64(a.win.Canvas().Size().Width))
	a.app.Preferences().SetFloat("win.h", float64(a.win.Canvas().Size().Height))
}
func (a *appState) setScale(s float32) {
	if s < 0.7 {
		s = 0.7
	}
	if s > 2.0 {
		s = 2.0
	}
	a.uiScale = s
	a.app.Preferences().SetFloat("ui.scale", float64(s))
	a.app.Settings().SetTheme(newAppTheme(s))
}

func (a *appState) adjustScale(delta float32) {
	a.setScale(a.uiScale + delta)
}
func (a *appState) buildUI() fyne.CanvasObject {
	// ---- Top bar: brand + primary action, then quick actions and view filters
	brand := widget.NewLabelWithStyle("WezTerm Configurator", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	brand.Importance = widget.HighImportance

	saveBtn := widget.NewButtonWithIcon("Save & Apply", theme.DocumentSaveIcon(), func() { a.save() })
	saveBtn.Importance = widget.HighImportance

	action := func(label string, icon fyne.Resource, fn func()) *widget.Button {
		b := widget.NewButtonWithIcon(label, icon, fn)
		b.Importance = widget.LowImportance
		return b
	}
	actions := container.NewHBox(
		action("Show Lua", theme.DocumentIcon(), a.previewLua),
		action("Folder", theme.FolderOpenIcon(), a.openConfigDir),
	)
	if a.wezterm != "" {
		actions.Add(action("Check", theme.ConfirmIcon(), a.checkWithWezterm))
	}
	actions.Add(action("Reset all", theme.DeleteIcon(), a.resetAll))
	actions.Add(action("", theme.ZoomOutIcon(), func() { a.adjustScale(-0.1) }))
	actions.Add(action("", theme.ZoomInIcon(), func() { a.adjustScale(0.1) }))
	previewBtn := widget.NewButtonWithIcon("Preview", theme.VisibilityOffIcon(), func() { a.setPreview(!a.previewOn) })
	previewBtn.Importance = widget.LowImportance
	actions.Add(previewBtn)
	a.previewBtn = previewBtn
	a.undoBtn = action("", theme.ContentUndoIcon(), a.undo)
	a.redoBtn = action("", theme.ContentRedoIcon(), a.redo)
	actions.Add(a.undoBtn)
	actions.Add(a.redoBtn)
	a.syncUndo()

	platformSelect := widget.NewSelect(platformLabels, func(label string) {
		if label == "" {
			return
		}
		a.setTarget(platformToTarget(label))
	})
	platformSelect.PlaceHolder = "Target"
	platformSelect.Selected = targetToPlatform(a.target)
	// Checked is set before the handler so restoring the preference does not
	// rebuild the page that is about to be built.
	showAllCheck := widget.NewCheck("All platforms", nil)
	showAllCheck.Checked = a.showAll
	showAllCheck.OnChanged = func(on bool) {
		a.showAll = on
		a.app.Preferences().SetBool(prefShowAll, on)
		a.rebuildPage()
	}
	changedCheck := widget.NewCheck("Changed only", nil)
	changedCheck.Checked = a.changedOnly
	changedCheck.OnChanged = func(on bool) {
		a.changedOnly = on
		a.app.Preferences().SetBool(prefChanged, on)
		a.rebuildPage()
	}
	filters := container.NewHBox(platformSelect, showAllCheck, changedCheck)

	topBar := container.NewVBox(
		container.NewBorder(nil, nil, container.NewHBox(widget.NewIcon(theme.ComputerIcon()), brand), saveBtn),
		container.NewBorder(nil, nil, actions, filters),
		widget.NewSeparator(),
	)

	// ---- Nav: search + grouped categories
	searchEntry := widget.NewEntry()
	a.search = searchEntry
	searchEntry.PlaceHolder = "🔍  Search options…"
	searchEntry.OnChanged = func(q string) {
		a.searchQuery = strings.ToLower(strings.TrimSpace(q))
		if a.searchQuery != "" && a.nav != nil {
			a.nav.UnselectAll()
		}
		a.rebuildPage()
	}

	cats := navCats()
	a.nav = widget.NewList(
		func() int { return len(cats) },
		func() fyne.CanvasObject {
			// Objects order is [name, count]; the count sits flush right.
			return container.NewBorder(nil, nil, nil, widget.NewLabel(""), widget.NewLabel(""))
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			c := cats[id]
			n := 0
			switch {
			case c == catalog.QuickCategory:
				for _, name := range a.quickOptionNames() {
					if a.setByName(name) || a.st.Raw[name] != "" {
						n++
					}
				}
			case c == catalog.PluginsCategory:
				n = len(a.st.Plugins)
			case c == catalog.FeaturesCategory:
				for i := range catalog.Features {
					if catalog.Features[i].IsOn(a.st.Features[catalog.Features[i].ID]) {
						n++
					}
				}
			case c != catalog.CustomLuaCategory:
				for i := range catalog.Options {
					o := &catalog.Options[i]
					if o.Category == c && (a.setByName(o.Name) || a.st.Raw[o.Name] != "") {
						n++
					}
				}
			case a.st.CustomLua != "":
				n = 1
			}
			box := o.(*fyne.Container).Objects
			name := box[0].(*widget.Label)
			count := box[1].(*widget.Label)
			name.SetText(c)
			if c == catalog.QuickCategory || c == catalog.PluginsCategory || c == catalog.FeaturesCategory || c == catalog.CustomLuaCategory {
				name.TextStyle = fyne.TextStyle{Bold: true}
			} else {
				name.TextStyle = fyne.TextStyle{}
			}
			count.TextStyle = fyne.TextStyle{Monospace: true}
			if n > 0 {
				count.Importance = widget.HighImportance
				count.SetText(" " + fmt.Sprintf("%d ●", n))
			} else {
				count.Importance = widget.LowImportance
				count.SetText("")
			}
		},
	)
	a.nav.OnSelected = func(id widget.ListItemID) { a.selectCategory(cats[id]) }
	navMin := canvas.NewRectangle(color.Transparent)
	navMin.SetMinSize(fyne.NewSize(220, 0))
	nav := container.NewStack(navMin, container.NewBorder(searchEntry, nil, nil, nil, a.nav))

	// ---- Content with readable max-width centering
	a.page = container.NewVBox()
	pageContent := container.NewPadded(a.page)
	maxContent := &readableWidthContainer{content: pageContent, maxWidth: 840}
	maxContent.ExtendBaseWidget(maxContent)
	a.pageScroll = newFastScroll(maxContent)
	// ---- Live preview column (optional, toggled from the top bar)
	a.preview = newPreviewPane(a)
	// ---- Status bar
	a.pathLabel = widget.NewLabel("⚙  " + a.paths.Config)
	a.pathLabel.TextStyle = fyne.TextStyle{Monospace: true}
	a.pathLabel.Importance = widget.LowImportance
	a.status = widget.NewLabel("")
	statusBar := container.NewBorder(nil, nil, a.pathLabel, a.status)

	split := container.NewHSplit(nav, a.contentSplit())
	split.SetOffset(0.20)
	return container.NewBorder(topBar, statusBar, nil, nil, split)
}

// navCats lists the nav entries in order: Quick Settings, the real categories,
// then the Plugins/Features/Custom Lua pages.
func navCats() []string {
	cats := append([]string{catalog.QuickCategory}, catalog.Categories...)
	return append(cats, catalog.PluginsCategory, catalog.FeaturesCategory, catalog.CustomLuaCategory)
}

// validCategory reports whether c is a nav entry; anything else (a category
// dropped from the catalog, a hand-edited preference) falls back to Quick Settings.
func validCategory(c string) bool {
	return slices.Contains(navCats(), c)
}

// selectCategory switches the page, drops any active search and remembers the
// choice for the next launch. Selecting the category already shown is a no-op
// so restoring the nav selection at startup does not rebuild twice.
func (a *appState) selectCategory(c string) {
	changed := c != a.currentCat
	a.currentCat = c
	if a.app != nil {
		a.app.Preferences().SetString(prefNavCat, c)
	}
	if a.searchQuery != "" {
		if a.search != nil {
			a.search.SetText("") // OnChanged clears the query and rebuilds
		}
		return
	}
	if changed {
		a.rebuildPage()
	}
}

// selectNav highlights the current category in the nav list.
func (a *appState) selectNav() {
	if a.nav == nil {
		return
	}
	for i, c := range navCats() {
		if c == a.currentCat {
			a.nav.Select(widget.ListItemID(i))
			return
		}
	}
}

// focusSearch puts the caret in the nav search box.
func (a *appState) focusSearch() {
	if a.search != nil {
		a.win.Canvas().Focus(a.search)
	}
}

// escapeSearch clears the search box when it holds the focus. Esc is left to
// whatever else has focus (dialogs, entries) so it keeps closing them.
func (a *appState) escapeSearch() {
	if a.search == nil || a.searchQuery == "" || a.win.Canvas().Focused() != a.search {
		return
	}
	a.search.SetText("")
}

// quickOptionNames lists what the Quick Settings page shows: the user's pins in
// pinning order, then the defaults, without duplicates. Unknown pins are skipped.
func (a *appState) quickOptionNames() []string {
	seen := make(map[string]bool, len(a.st.Pinned)+len(catalog.QuickOptionNames))
	out := make([]string, 0, len(catalog.QuickOptionNames)+len(a.st.Pinned))
	for _, name := range slices.Concat(a.st.Pinned, catalog.QuickOptionNames) {
		if seen[name] || catalog.Find(name) == nil {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

// optionMatchesQuery reports whether an option matches a search query by name,
// documentation or one of its top-level choice values. Matching ignores case.
func optionMatchesQuery(o *catalog.Option, q string) bool {
	q = strings.ToLower(q)
	if strings.Contains(strings.ToLower(o.Name), q) || strings.Contains(strings.ToLower(o.Doc), q) {
		return true
	}
	for _, e := range o.Enum {
		if strings.Contains(strings.ToLower(e), q) {
			return true
		}
	}
	return false
}

// searchResults lists the options a search query hits, honouring both view
// filters the same way the category pages do.
func (a *appState) searchResults(q string) []*catalog.Option {
	var out []*catalog.Option
	for i := range catalog.Options {
		o := &catalog.Options[i]
		if !optionMatchesQuery(o, q) {
			continue
		}
		if a.changedOnly && !a.isSet(o) {
			continue
		}
		if !a.showAll && !catalog.RelevantTo(o.Tags, a.target) {
			continue
		}
		out = append(out, o)
	}
	return out
}

// matchesSomePlatform reports whether the query hits anything once the target
// platform filter is dropped, so the empty search page can offer the right hint.
func (a *appState) matchesSomePlatform(q string) bool {
	for i := range catalog.Options {
		o := &catalog.Options[i]
		if optionMatchesQuery(o, q) && (!a.changedOnly || a.isSet(o)) {
			return true
		}
	}
	return false
}

// contentSplit lays out the page next to the optional preview column. The
// column is hidden rather than rebuilt so toggling it keeps the mounted rows.
func (a *appState) contentSplit() fyne.CanvasObject {
	if a.preview == nil {
		return a.pageScroll
	}
	previewMin := canvas.NewRectangle(color.Transparent)
	previewMin.SetMinSize(fyne.NewSize(300, 0))
	a.previewCol = container.NewStack(previewMin, a.preview)
	right := container.NewHSplit(a.pageScroll, a.previewCol)
	right.SetOffset(0.7)
	a.applyPreviewVisibility()
	return right
}

// setPreview shows or hides the live preview column and remembers the choice.
func (a *appState) setPreview(on bool) {
	a.previewOn = on
	if a.app != nil {
		a.app.Preferences().SetBool(prefPreview, on)
	}
	if a.previewBtn != nil {
		icon := theme.VisibilityIcon()
		if on {
			icon = theme.VisibilityOffIcon()
		}
		a.previewBtn.Icon = icon
		if on {
			a.previewBtn.Importance = widget.HighImportance
		} else {
			a.previewBtn.Importance = widget.LowImportance
		}
		a.previewBtn.Refresh()
	}
	a.applyPreviewVisibility()
	a.updatePreview()
}

func (a *appState) applyPreviewVisibility() {
	if a.previewCol == nil {
		return
	}
	if a.previewOn {
		a.previewCol.Show()
	} else {
		a.previewCol.Hide()
	}
}

func (a *appState) updatePreview() {
	if a.preview != nil {
		a.preview.Update()
	}
}

// setTitle marks unsaved changes in the window title.
func (a *appState) setTitle() {
	if a.win == nil {
		return
	}
	if a.dirty {
		a.win.SetTitle("● WezTerm Configurator")
	} else {
		a.win.SetTitle("WezTerm Configurator")
	}
}

// changedCount counts options that differ from their default, for the status bar.
func (a *appState) changedCount() int {
	n := 0
	for i := range catalog.Options {
		if a.isOptionChangedFromDefault(&catalog.Options[i]) {
			n++
		}
	}
	return n
}

// shortcuts lists the key bindings shown in the help dialog.
var shortcuts = [][2]string{
	{"Ctrl+S", "Save & apply"},
	{"Ctrl+O", "Open a state file or wezterm.lua"},
	{"Ctrl+F", "Focus the search box"},
	{"Ctrl+Z", "Undo"},
	{"Ctrl+Y / Ctrl+Shift+Z", "Redo"},
	{"Ctrl+= / Ctrl+- / Ctrl+0", "Zoom in / out / reset"},
	{"F1 or Ctrl+/", "This dialog"},
	{"Ctrl+Q", "Quit"},
	{"Esc", "Clear the search box"},
}

// showShortcuts opens the keyboard reference.
func (a *appState) showShortcuts() {
	rows := make([]fyne.CanvasObject, 0, len(shortcuts)*2)
	for _, s := range shortcuts {
		key := widget.NewLabelWithStyle(s[0], fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
		desc := widget.NewLabel(s[1])
		desc.Importance = widget.LowImportance
		rows = append(rows, key, desc)
	}
	note := widget.NewLabel("On macOS, Ctrl is the Super/⌘ key.")
	note.Importance = widget.LowImportance
	note.Wrapping = fyne.TextWrapWord
	rows = append(rows, container.NewHBox(), container.NewHBox(), spacer(8), note)
	grid := container.NewGridWithColumns(2, rows...)
	d := dialog.NewCustom("Keyboard shortcuts", "Close", container.NewVScroll(container.NewPadded(grid)), a.win)
	d.Resize(fyne.NewSize(520, 460))
	d.Show()
}

// showAbout reports the version and the files this app owns.
func (a *appState) showAbout() {
	l := widget.NewLabel(fmt.Sprintf(
		"WezTerm Configurator %s\n\nA visual editor for wezterm.lua.\n\nConfiguration: %s\nState file:    %s",
		appVersion, a.paths.Config, a.paths.State))
	l.Wrapping = fyne.TextWrapBreak
	d := dialog.NewCustom("About WezTerm Configurator", "Close", container.NewPadded(l), a.win)
	d.Resize(fyne.NewSize(520, 260))
	d.Show()
}

type readableWidthContainer struct {
	widget.BaseWidget
	content  fyne.CanvasObject
	maxWidth float32
}

func (c *readableWidthContainer) CreateRenderer() fyne.WidgetRenderer {
	return &readableWidthRenderer{container: c}
}

type readableWidthRenderer struct {
	container *readableWidthContainer
}

func (r *readableWidthRenderer) Layout(size fyne.Size) {
	w := size.Width
	if r.container.maxWidth > 0 && w > r.container.maxWidth {
		w = r.container.maxWidth
	}
	x := float32(0) // left-aligned with readable max width
	r.container.content.Move(fyne.NewPos(x, 0))
	r.container.content.Resize(fyne.NewSize(w, r.container.content.MinSize().Height))
}

func (r *readableWidthRenderer) MinSize() fyne.Size {
	contentMin := r.container.content.MinSize()
	w := contentMin.Width
	if r.container.maxWidth > 0 && w > r.container.maxWidth {
		w = r.container.maxWidth
	}
	return fyne.NewSize(w, contentMin.Height)
}

func (r *readableWidthRenderer) Refresh() {
	r.container.content.Refresh()
}

func (r *readableWidthRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.container.content}
}

func (r *readableWidthRenderer) Destroy() {}

// fastScroll is a vertical scroll with a boosted wheel. It embeds Scroll by
// value and extends it in place: the object handed to the layout must be the
// fastScroll itself, otherwise the widget and its renderer get two identities.
type fastScroll struct {
	container.Scroll
}

func newFastScroll(content fyne.CanvasObject) *fastScroll {
	s := &fastScroll{}
	s.Direction = container.ScrollVerticalOnly
	s.Content = content
	s.ExtendBaseWidget(s)
	return s
}

func (s *fastScroll) Scrolled(ev *fyne.ScrollEvent) {
	s.Scroll.Scrolled(&fyne.ScrollEvent{
		PointEvent: ev.PointEvent,
		Scrolled:   fyne.Delta{DX: ev.Scrolled.DX * 8, DY: ev.Scrolled.DY * 8},
	})
}

func (a *appState) setByName(name string) bool {
	_, ok := a.st.Values[name]
	return ok
}

func (a *appState) setTarget(t string) {
	if t == a.target {
		return
	}
	a.target = t
	a.st.TargetOS = t
	a.dirty = true
	// MacOsNative is rejected by WezTerm on non-macOS builds.
	if t != "macos" {
		if v, ok := a.st.Values["integrated_title_button_style"].(string); ok && v == "MacOsNative" {
			delete(a.st.Values, "integrated_title_button_style")
			dialog.ShowInformation("macOS-only option",
				"integrated_title_button_style = MacOsNative is only accepted on macOS and was removed from your settings.", a.win)
		}
	}
	a.refreshNav()
	a.rebuildPage()
	a.updatePreview()
	a.setTitle()
}

func (a *appState) refreshNav() {
	if a.nav != nil {
		a.nav.Refresh()
	}
}

func (a *appState) markDirty() {
	a.dirty = true
	a.hist.Changed(a.st)
	if a.status != nil { // headless callers (profiles, import) have no status bar
		a.status.Importance = widget.WarningImportance
		a.status.SetText(fmt.Sprintf("●  Unsaved changes: %s changed — press Ctrl+S or “Save & Apply”", plural(a.changedCount(), "option")))
	}
	a.refreshNav()
	a.updatePreview()
	a.syncUndo()
	a.setTitle()
}

// plural renders "3 options" / "1 option".
func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// stateReplaced refreshes the UI after a.st's contents were swapped wholesale
// (undo, restore, profile load, import). Callers set a.dirty/status themselves.
func (a *appState) stateReplaced() {
	a.rebuildPage()
	a.refreshNav()
	a.updatePreview()
	a.setTitle()
	a.syncUndo()
}

// syncUndo enables the undo/redo buttons only when there is something to do.
func (a *appState) syncUndo() {
	if a.undoBtn == nil || a.hist == nil {
		return
	}
	if a.hist.CanUndo() {
		a.undoBtn.Enable()
	} else {
		a.undoBtn.Disable()
	}
	if a.hist.CanRedo() {
		a.redoBtn.Enable()
	} else {
		a.redoBtn.Disable()
	}
}

// pinToggled refreshes what depends on the pinned list after a star was toggled.
func (a *appState) pinToggled() {
	a.markDirty()
	if a.currentCat == catalog.QuickCategory {
		a.rebuildPage()
	}
}

// visible reports whether an option passes the target-platform and "Changed only" filters.
func (a *appState) visible(o *catalog.Option) bool {
	if a.changedOnly && !a.isSet(o) {
		return false
	}
	return a.showAll || catalog.RelevantTo(o.Tags, a.target)
}

// noneMsg picks the empty-page message, explaining the active filter when there is one.
func (a *appState) noneMsg(def string) string {
	if a.changedOnly {
		return "Nothing changed here yet. Turn off “Changed only” to see every option."
	}
	return def
}

func (a *appState) rebuildPage() {
	if a.page == nil { // headless callers never build a page container
		return
	}
	rows := []fyne.CanvasObject{}
	a.rows = nil

	pageHeader := func(title, sub string) {
		rows = append(rows, heading(title))
		if sub != "" {
			h := widget.NewLabel(sub)
			h.Wrapping = fyne.TextWrapWord
			h.Importance = widget.LowImportance
			rows = append(rows, h)
		}
		rows = append(rows, spacer(6))
	}

	switch {
	// Search wins over the selected page: the box is always visible in the nav
	// and picks a category clears it, so the two are mutually exclusive.
	case a.searchQuery != "":
		rows = append(rows, heading("Search results"))
		found := a.searchResults(a.searchQuery)
		for _, o := range found {
			rows = append(rows, a.makeRow(o))
		}
		if len(found) == 0 {
			msg := "Nothing matches “" + a.searchQuery + "”. Try a shorter query."
			if !a.showAll && a.matchesSomePlatform(a.searchQuery) {
				msg += " Some hits belong to other platforms — enable “All platforms” to see them."
			}
			rows = append(rows, emptyHint(msg))
		}
	case a.currentCat == catalog.PluginsCategory:
		pageHeader("Plugins", "Plugins are git repos loaded with wezterm.plugin.require (WezTerm 20230320 or newer). URLs must be https:// or file://. Updates: run wezterm.plugin.update_all() in the debug overlay, then reload the config. Clones live in ~/.local/share/wezterm/plugins.")
		rows = append(rows, a.pluginsEditor()...)
	case a.currentCat == catalog.FeaturesCategory:
		rows = append(rows, a.featuresEditor()...)
	case a.currentCat == catalog.CustomLuaCategory:
		pageHeader("Custom Lua", "Arbitrary Lua code appended at the end of wezterm.lua. Runs inside an isolated 'do ... end' block. Return statement is added automatically by the emitter.")
		luaEntry := widget.NewMultiLineEntry()
		luaEntry.TextStyle = fyne.TextStyle{Monospace: true}
		luaEntry.SetMinRowsVisible(16)
		luaEntry.PlaceHolder = "-- Write your custom Lua here, e.g.:\n-- wezterm.on('format-tab-title', function(tab, tabs, panes, config, hover, max_width)\n--   return tab.active_pane.title\n-- end)"
		luaEntry.SetText(a.st.CustomLua)
		luaEntry.OnChanged = func(s string) {
			a.st.CustomLua = s
			a.markDirty()
		}
		rows = append(rows, luaEntry)
	case a.currentCat == catalog.QuickCategory:
		pageHeader("Quick Settings", "Frequently used and recommended settings, plus everything you pinned. ★ pin any option from its row to keep it here.")
		shown := 0
		for _, name := range a.quickOptionNames() {
			o := catalog.Find(name)
			if o == nil || !a.visible(o) {
				continue
			}
			rows = append(rows, a.makeRow(o))
			shown++
		}
		if shown == 0 {
			rows = append(rows, emptyHint(a.noneMsg("No quick settings available for "+targetToPlatform(a.target)+". Enable “All platforms” to see them.")))
		}
	default:
		rows = append(rows, heading(a.currentCat))
		shown := 0
		for i := range catalog.Options {
			o := &catalog.Options[i]
			if o.Category != a.currentCat {
				continue
			}
			if !a.visible(o) {
				continue
			}
			rows = append(rows, a.makeRow(o))
			shown++
		}
		if shown == 0 {
			rows = append(rows, emptyHint(a.noneMsg("No options in this category for "+targetToPlatform(a.target)+". Enable “All platforms” to see them.")))
		}
	}

	rows = append(rows, spacer(24))
	a.page.Objects = rows
	a.page.Refresh()
}

// spacer returns a fixed-height transparent filler.
func spacer(h int) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(0, float32(h)))
	return r
}

// emptyHint renders a friendly message when a page has no content.
func emptyHint(msg string) fyne.CanvasObject {
	l := widget.NewLabel("•  " + msg)
	l.Wrapping = fyne.TextWrapWord
	l.Importance = widget.WarningImportance
	return l
}

func heading(text string) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	title.Importance = widget.HighImportance
	underline := canvas.NewRectangle(mustHex(colPrimary))
	underline.SetMinSize(fyne.NewSize(48, 3))
	return container.NewVBox(title, underline, spacer(4))
}

func (a *appState) makeRow(o *catalog.Option) fyne.CanvasObject {
	r := a.newOptionRow(o)
	a.rows = append(a.rows, r)
	return r.obj
}

// validateAll runs validation over mounted rows and the emitter.
func (a *appState) validateAll() error {
	var errs []error
	for _, r := range a.rows {
		if err := r.validate(); err != nil {
			errs = append(errs, err)
		}
	}
	if _, err := luagen.Emit(a.st); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// save runs the save pipeline; see saveThen.
func (a *appState) save() { a.saveThen(nil) }

// saveThen validates, emits, then walks the user through the checks that can
// stop a save: WezTerm's own config check, key-binding conflicts, the diff
// review, and the one-time "replace a foreign wezterm.lua" prompt. Each of
// those may ask asynchronously, so done (may be nil) runs only after the
// files were written.
func (a *appState) saveThen(done func()) {
	if err := a.validateAll(); err != nil {
		dialog.ShowError(err, a.win)
		return
	}
	out, err := luagen.Emit(a.st)
	if err != nil {
		dialog.ShowError(err, a.win)
		return
	}
	confirm := func(title, msg string, next func()) {
		dialog.ShowConfirm(title, msg, func(ok bool) {
			if ok {
				next()
			}
		}, a.win)
	}

	write := func() {
		a.writeFiles(out)
		if a.wroteOK && done != nil {
			done()
		}
	}
	ownership := func() {
		exists, owned, err := state.Owned(a.paths.Config)
		if err != nil {
			dialog.ShowError(err, a.win)
			return
		}
		if exists && !owned && !a.savedFirst {
			confirm("Replace existing wezterm.lua?",
				fmt.Sprintf("%s was not generated by this app. It will be backed up to %s.bak-<timestamp> and replaced. Continue?", a.paths.Config, a.paths.Config),
				func() {
					if _, err := state.BackupUnowned(a.paths.Config); err != nil {
						dialog.ShowError(err, a.win)
						return
					}
					a.savedFirst = true
					write()
				})
			return
		}
		write()
	}
	review := func() { a.reviewChanges(out, ownership) }
	conflicts := func() {
		if c := lint.KeyConflicts(a.st); len(c) > 0 {
			confirm("Key binding conflicts", strings.Join(c, "\n")+"\n\nSave anyway?", review)
			return
		}
		review()
	}

	if a.wezterm != "" {
		if tmpFile, err := os.CreateTemp("", "wezterm-check-*.lua"); err == nil {
			tmpPath := tmpFile.Name()
			_, _ = tmpFile.WriteString(out)
			_ = tmpFile.Close()
			defer os.Remove(tmpPath)

			if ok, checkOut, _ := wezcli.Check(a.wezterm, tmpPath); !ok {
				confirm("Configuration Check Failed",
					fmt.Sprintf("WezTerm reported configuration errors:\n\n%s\nSave anyway?", strings.TrimSpace(checkOut)),
					conflicts)
				return
			}
		}
	}
	conflicts()
}

// showReview shows what saving would change without saving.
func (a *appState) showReview() {
	out, err := luagen.Emit(a.st)
	if err != nil {
		dialog.ShowError(err, a.win)
		return
	}
	a.reviewChanges(out, nil)
}

func (a *appState) writeFiles(out string) {
	a.wroteOK = false
	// Keep the previous version (last state.KeepBackups) before overwriting it.
	if _, err := state.Backup(a.paths); err != nil {
		dialog.ShowError(fmt.Errorf("backing up the current config failed, nothing was saved: %w", err), a.win)
		return
	}
	if err := a.st.Save(a.paths.State); err != nil {
		dialog.ShowError(fmt.Errorf("saving state: %w", err), a.win)
		return
	}
	if err := writeAtomic(a.paths.Config, out); err != nil {
		dialog.ShowError(fmt.Errorf("writing %s: %w", a.paths.Config, err), a.win)
		return
	}
	a.status.Importance = widget.SuccessImportance
	a.status.SetText("✓  Saved & applied " + time.Now().Format("15:04:05"))
	a.refreshNav()
	a.dirty = false
	a.wroteOK = true
	a.setTitle()
}
func (a *appState) previewLua() {
	out, err := luagen.Emit(a.st)
	text := out
	if err != nil {
		text = "Error: " + err.Error()
	}
	e := widget.NewMultiLineEntry()
	e.TextStyle = fyne.TextStyle{Monospace: true}
	e.Wrapping = fyne.TextWrapBreak
	e.SetMinRowsVisible(30)
	e.SetText(text)
	scroll := container.NewVScroll(e)
	d := dialog.NewCustom("Generated wezterm.lua", "Close", scroll, a.win)
	d.Resize(fyne.NewSize(900, 700))
	d.Show()
}

func (a *appState) openConfigDir() {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", a.paths.ConfigDir)
	case "darwin":
		cmd = exec.Command("open", a.paths.ConfigDir)
	default:
		cmd = exec.Command("xdg-open", a.paths.ConfigDir)
	}
	if err := cmd.Start(); err != nil {
		dialog.ShowError(err, a.win)
	}
}

func (a *appState) checkWithWezterm() {
	if a.wezterm == "" {
		return
	}
	if a.dirty {
		dialog.ShowConfirm("Save first?", "Save the current changes before asking WezTerm to load the config?", func(ok bool) {
			if !ok {
				return
			}
			a.saveThen(a.runCheck)
		}, a.win)
		return
	}
	a.runCheck()
}

func (a *appState) runCheck() {
	bin, cfg := a.wezterm, a.paths.Config
	go func() {
		ok, output, _ := wezcli.Check(bin, cfg)
		title := "WezTerm loaded the config"
		if !ok {
			title = "WezTerm reported problems"
		}
		output = strings.TrimSpace(output)
		fyne.Do(func() {
			l := widget.NewLabel(output)
			l.TextStyle = fyne.TextStyle{Monospace: true}
			l.Wrapping = fyne.TextWrapBreak
			dialog.ShowCustom(title, "Close", container.NewVScroll(l), a.win)
		})
	}()
}

func (a *appState) resetAll() {
	dialog.ShowConfirm("Reset all options?", "Clear every configured option, raw-Lua override and the custom Lua block?", func(ok bool) {
		if !ok {
			return
		}
		a.st.Values = map[string]any{}
		a.st.Raw = map[string]string{}
		a.st.CustomLua = ""
		a.markDirty()
		a.rebuildPage()
	}, a.win)
}

func (a *appState) exportLua() {
	out, err := luagen.Emit(a.st)
	if err != nil {
		dialog.ShowError(fmt.Errorf("generating lua: %w", err), a.win)
		return
	}
	d := dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
		if err != nil || uc == nil {
			return
		}
		defer uc.Close()
		if _, werr := uc.Write([]byte(out)); werr != nil {
			dialog.ShowError(fmt.Errorf("exporting lua: %w", werr), a.win)
		}
	}, a.win)
	d.SetFileName("wezterm.lua")
	d.Show()
}

func (a *appState) openFile() {
	d := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
		if err != nil || rc == nil {
			return
		}
		defer rc.Close()
		content, err := io.ReadAll(rc)
		if err != nil {
			dialog.ShowError(fmt.Errorf("reading file: %w", err), a.win)
			return
		}
		uriPath := rc.URI().Path()
		var newSt *state.State
		// Try JSON unmarshal first
		var st state.State
		if jerr := json.Unmarshal(content, &st); jerr == nil && st.Version > 0 {
			newSt = &st
		} else {
			// Try extracting embedded state from Lua
			extracted, xerr := state.Extract(string(content))
			if xerr != nil {
				dialog.ShowError(fmt.Errorf("could not load state from %s (neither valid state JSON nor embedded state Lua: %w)", filepath.Base(uriPath), xerr), a.win)
				return
			}
			newSt = extracted
		}

		a.st = newSt
		if a.st.TargetOS != "" {
			a.target = a.st.TargetOS
		}
		a.dirty = false
		a.stateReplaced()
	}, a.win)
	d.Show()
}
func writeAtomic(path, content string) error {
	return osWriteFileAtomic(path, content)
}
