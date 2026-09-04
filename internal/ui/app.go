// Package ui implements the WezTerm Configurator application shell and
// option editors.
package ui

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"weztermconfigurator/internal/catalog"
	"weztermconfigurator/internal/luagen"
	"weztermconfigurator/internal/state"
	"weztermconfigurator/internal/wezcli"
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

	page       *fyne.Container
	pageScroll *container.Scroll
	nav        *widget.List
	status     *widget.Label
	pathLabel  *widget.Label

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
	w := a.NewWindow("WezTerm Configurator")
	w.Resize(fyne.NewSize(
		float32(a.Preferences().FloatWithFallback("win.w", 1280)),
		float32(a.Preferences().FloatWithFallback("win.h", 820)),
	))

	paths, err := state.Resolve()
	if err != nil {
		dialog.ShowError(fmt.Errorf("resolving WezTerm paths: %w", err), w)
	}
	st, err := state.Load(paths.State)
	if err != nil {
		dialog.ShowError(fmt.Errorf("loading state: %w", err), w)
		st = state.New(state.RuntimeTarget())
	}

	a2 := &appState{
		app:    a,
		win:    w,
		paths:  paths,
		st:     st,
		target: st.TargetOS,
	}
	a2.target = st.TargetOS

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
	a2.rebuildPage()
	a2.refreshNav()

	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl},
		func(fyne.Shortcut) { a2.save() })

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
				if a2.save() {
					a2.persistWindowSize()
					w.Close()
				}
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

func (a *appState) buildUI() fyne.CanvasObject {
	// Toolbar
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.DocumentSaveIcon(), func() { a.save() }),
		widget.NewToolbarAction(theme.DocumentIcon(), a.previewLua),
		widget.NewToolbarAction(theme.FolderOpenIcon(), a.openConfigDir),
	)
	if a.wezterm != "" {
		toolbar.Items = append(toolbar.Items, widget.NewToolbarAction(theme.ConfirmIcon(), a.checkWithWezterm))
	}
	toolbar.Items = append(toolbar.Items,
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.DeleteIcon(), a.resetAll),
	)

	platformSelect := widget.NewSelect(platformLabels, func(label string) {
		if label == "" {
			return
		}
		a.setTarget(platformToTarget(label))
	})
	platformSelect.Selected = targetToPlatform(a.target)
	showAllCheck := widget.NewCheck("Show all platforms", func(on bool) {
		a.showAll = on
		a.rebuildPage()
	})
	rightControls := container.NewHBox(widget.NewLabel("Target platform"), platformSelect, showAllCheck)

	topBar := container.NewBorder(nil, nil, toolbar, rightControls)

	// Nav: search + category list
	searchEntry := widget.NewEntry()
	searchEntry.PlaceHolder = "Search options…"
	searchEntry.OnChanged = func(q string) {
		a.searchQuery = strings.ToLower(q)
		a.rebuildPage()
	}

	cats := append(append([]string{}, catalog.Categories...), catalog.CustomLuaCategory)
	a.nav = widget.NewList(
		func() int { return len(cats) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, o fyne.CanvasObject) {
			c := cats[id]
			n := 0
			if c != catalog.CustomLuaCategory {
				for i := range catalog.Options {
					o := &catalog.Options[i]
					if o.Category == c && (a.setByName(o.Name) || a.st.Raw[o.Name] != "") {
						n++
					}
				}
			} else if a.st.CustomLua != "" {
				n = 1
			}
			o.(*widget.Label).SetText(fmt.Sprintf("%s (%d)", c, n))
		},
	)
	a.nav.OnSelected = func(id widget.ListItemID) {
		a.currentCat = cats[id]
		a.rebuildPage()
	}

	nav := container.NewBorder(searchEntry, nil, nil, nil, a.nav)

	// Content scroll
	a.page = container.NewVBox()
	a.pageScroll = container.NewVScroll(a.page)

	// Status bar
	a.pathLabel = widget.NewLabel("Config: " + a.paths.Config)
	a.status = widget.NewLabel("")
	statusBar := container.NewBorder(nil, nil, a.pathLabel, a.status)

	split := container.NewHSplit(nav, a.pageScroll)
	split.SetOffset(0.22)

	return container.NewBorder(topBar, statusBar, nil, nil, split)
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
}

func (a *appState) refreshNav() {
	if a.nav != nil {
		a.nav.Refresh()
	}
}

func (a *appState) markDirty() {
	a.dirty = true
	a.status.SetText("Unsaved changes")
	a.refreshNav()
}

func (a *appState) rebuildPage() {
	rows := []fyne.CanvasObject{}
	a.rows = nil

	switch {
	case a.searchQuery != "":
		rows = append(rows, heading("Search: "+a.searchQuery))
		for i := range catalog.Options {
			o := &catalog.Options[i]
			if strings.Contains(strings.ToLower(o.Name), a.searchQuery) ||
				strings.Contains(strings.ToLower(o.Doc), a.searchQuery) {
				rows = append(rows, a.makeRow(o))
			}
		}
	case a.currentCat == catalog.CustomLuaCategory || a.currentCat == "":
		rows = append(rows, heading(catalog.CustomLuaCategory))
		help := widget.NewLabel("Lua appended verbatim before `return config`; use it for event handlers and anything the form cannot express.")
		help.Wrapping = fyne.TextWrapWord
		rows = append(rows, help)
		e := widget.NewMultiLineEntry()
		e.TextStyle = fyne.TextStyle{Monospace: true}
		e.SetMinRowsVisible(20)
		e.SetText(a.st.CustomLua)
		e.OnChanged = func(s string) {
			a.st.CustomLua = s
			a.markDirty()
		}
		rows = append(rows, e)
	default:
		rows = append(rows, heading(a.currentCat))
		for i := range catalog.Options {
			o := &catalog.Options[i]
			if o.Category != a.currentCat {
				continue
			}
			if !catalog.RelevantTo(o.Tags, a.target) && !a.showAll {
				continue
			}
			rows = append(rows, a.makeRow(o))
		}
	}

	a.page.Objects = rows
	a.page.Refresh()
}

func heading(text string) fyne.CanvasObject {
	return widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
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

// save validates, backs up an unowned config once, writes state + Lua.
// Returns true when saved.
func (a *appState) save() bool {
	if err := a.validateAll(); err != nil {
		dialog.ShowError(err, a.win)
		return false
	}
	a.wroteOK = false

	doWrite := func() {
		exists, owned, err := state.Owned(a.paths.Config)
		if err != nil {
			dialog.ShowError(err, a.win)
			return
		}
		if exists && !owned && !a.savedFirst {
			dialog.ShowConfirm("Replace existing wezterm.lua?",
				fmt.Sprintf("%s was not generated by this app. It will be backed up to %s.bak-<timestamp> and replaced. Continue?", a.paths.Config, a.paths.Config),
				func(ok bool) {
					if !ok {
						return
					}
					if _, err := state.BackupUnowned(a.paths.Config); err != nil {
						dialog.ShowError(err, a.win)
						return
					}
					a.savedFirst = true
					a.writeFiles()
				}, a.win)
			return
		}
		a.writeFiles()
	}
	doWrite()
	return a.wroteOK
}

func (a *appState) writeFiles() {
	a.wroteOK = false
	if err := a.st.Save(a.paths.State); err != nil {
		dialog.ShowError(fmt.Errorf("saving state: %w", err), a.win)
		return
	}
	out, err := luagen.Emit(a.st)
	if err != nil {
		dialog.ShowError(err, a.win)
		return
	}
	if err := writeAtomic(a.paths.Config, out); err != nil {
		dialog.ShowError(fmt.Errorf("writing %s: %w", a.paths.Config, err), a.win)
		return
	}
	a.dirty = false
	a.status.SetText("Saved " + time.Now().Format("15:04:05"))
	a.refreshNav()
	a.wroteOK = true
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
			a.save()
			a.runCheck()
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

func writeAtomic(path, content string) error {
	return osWriteFileAtomic(path, content)
}
