package ui

import (
	"fmt"
	"image/color"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"weztermconfigurator/internal/catalog"
	"weztermconfigurator/internal/luagen"
)

// row is one mounted option editor.
type row struct {
	opt      *catalog.Option
	validate func() error
	obj      fyne.CanvasObject
}

var (
	intRe    = regexp.MustCompile(`^-?\d+$`)
	floatRe  = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
	dimRe    = regexp.MustCompile(`^-?\d+(\.\d+)?(px|pt|cell|%)?$`)
	colorHex = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{9}|[0-9a-fA-F]{12})$`)
	colorFn  = regexp.MustCompile(`^(rgb|rgba|hsl|hsla|hsv|hwb)\(.+\)$`)
	colorCol = regexp.MustCompile(`^(rgb|rgba|hsl):.+$`)
	colorNam = regexp.MustCompile(`^[A-Za-z]+$`)
	weightRe = regexp.MustCompile(`^([A-Za-z]+|\d+)$`)
)

// isSet reports whether the option currently has a value.
func formatOptionTooltip(o *catalog.Option, target string) string {
	var b strings.Builder
	b.WriteString(o.Name)
	if o.Since != "" {
		b.WriteString(" (since " + o.Since + ")")
	}
	b.WriteString("\n\n")
	if o.Doc != "" {
		b.WriteString(o.Doc)
		b.WriteString("\n")
	}
	if o.Deprecated != "" {
		b.WriteString("\n⚠️ DEPRECATED: " + o.Deprecated + "\n")
	}
	def := catalog.DefaultFor(o, target)
	if def == nil && o.DefaultNote != "" {
		b.WriteString("\nDefault: " + o.DefaultNote)
	} else if def != nil {
		b.WriteString("\nDefault: " + literal(def))
	}
	if len(o.Enum) > 0 {
		b.WriteString("\n\nAvailable options / values:\n")
		for _, optVal := range o.Enum {
			b.WriteString(" • " + optVal + "\n")
		}
	}
	return strings.TrimSpace(b.String())
}

type hoverHelpButton struct {
	widget.Button
	infoText   string
	title      string  // popup header; empty = generic documentation title
	minH       float32 // popup body height; 0 = 220
	win        fyne.Window
	overlay    *fyne.Container // popup on the canvas; nil while closed
	pinned     bool            // opened by a click, so the pointer may leave
	wasFocused fyne.Focusable
}

func (b *hoverHelpButton) height() float32 {
	if b.minH > 0 {
		return b.minH
	}
	return 220
}

// choiceCell puts w (a radio or check for one choice) next to a "?" that explains it.
func (a *appState) choiceCell(w fyne.CanvasObject, field, value string) fyne.CanvasObject {
	doc := catalog.ValueDoc(field, value)
	if doc == "" {
		return w
	}
	h := newHoverHelpButton(doc, a.win)
	h.title = value
	h.minH = 70
	return container.NewHBox(w, h)
}

// choiceGrid lays cells out in two columns, or one when a label is too long to share a row.
func choiceGrid(values []string, cells []fyne.CanvasObject) fyne.CanvasObject {
	cols := 2
	for _, v := range values {
		if len(v) > 24 {
			cols = 1
		}
	}
	return container.NewGridWithColumns(cols, cells...)
}

// choiceListHelp is one "?" that lists every choice of a dropdown with its explanation.
func (a *appState) choiceListHelp(field string, values []string) fyne.CanvasObject {
	var b strings.Builder
	for _, v := range values {
		if doc := catalog.ValueDoc(field, v); doc != "" {
			b.WriteString(v + "\n    " + doc + "\n\n")
		}
	}
	if b.Len() == 0 {
		return widget.NewLabel("")
	}
	h := newHoverHelpButton(strings.TrimSpace(b.String()), a.win)
	h.title = "What each choice means"
	return h
}

// The help popup is a canvas overlay of our own rather than a widget.PopUp
// because that one covers the window with a transparent overlay which swallows
// every mouse event: showing it made the button under the pointer fire MouseOut,
// hide the popup, fire MouseIn again and show it once more, forever. Here the
// popup is opened after a short hover, and closes a moment after the pointer has
// left both the button and the popup itself, which it can be moved onto to read
// and scroll. A click pins it open until it is clicked again, closed or Escape.
const (
	helpShowDelay = 250 * time.Millisecond
	helpHideDelay = 200 * time.Millisecond
	helpGap       = 8 // keep the popup clear of the button
)

var (
	activeHelp *hoverHelpButton // the one help popup shown, app-wide
	helpShow   *time.Timer
	helpHide   *time.Timer
)

// at replaces a pending timer; fn runs on the UI thread, never on the goroutine
// the timer fires on.
func at(t **time.Timer, d time.Duration, fn func()) {
	stopTimer(t)
	*t = time.AfterFunc(d, func() { fyne.Do(fn) })
}

func stopTimer(t **time.Timer) {
	if *t != nil {
		(*t).Stop()
		*t = nil
	}
}

// helpPlacement returns where a popup of the given size goes for a button at
// btn: beside and below it, above it when the window is too short for that,
// never over the button and always inside the window.
func helpPlacement(btn fyne.Position, btnSize, pop, win fyne.Size) fyne.Position {
	x := btn.X + btnSize.Width + helpGap
	y := btn.Y + btnSize.Height + helpGap
	if y+pop.Height > win.Height-helpGap {
		if above := btn.Y - helpGap - pop.Height; above >= 0 {
			y = above
		}
	}
	if x+pop.Width > win.Width-helpGap {
		x = win.Width - helpGap - pop.Width
	}
	if x < helpGap {
		x = helpGap
	}
	if y+pop.Height > win.Height-helpGap {
		y = win.Height - helpGap - pop.Height
	}
	if y < helpGap {
		y = helpGap
	}
	return fyne.NewPos(x, y)
}

// helpPad is an invisible hit area. Fyne hands mouse events to the last object
// in the tree that matches the pointer, so the pad drawn last wins the whole
// popup: without it the label under the cursor would swallow the pointer and
// the popup would close as soon as the text was reached.
type helpPad struct {
	canvas.Rectangle
	b *hoverHelpButton
}

func (p *helpPad) MouseIn(*desktop.MouseEvent)    { stopTimer(&helpHide) }
func (p *helpPad) MouseMoved(*desktop.MouseEvent) {}
func (p *helpPad) MouseOut()                      { p.b.scheduleHide() }

// helpBackdrop is the pad behind the popup, covering the rest of the window: a
// pointer leaving it closes the help, and so does a tap, because the overlay
// swallows every tap that is not on the popup itself.
type helpBackdrop struct{ helpPad }

func (p *helpBackdrop) Tapped(*fyne.PointEvent)          { p.b.hide() }
func (p *helpBackdrop) TappedSecondary(*fyne.PointEvent) { p.b.hide() }
func (p *helpBackdrop) FocusGained()                     {}
func (p *helpBackdrop) FocusLost()                       {}

func (p *helpBackdrop) TypedKey(e *fyne.KeyEvent) {
	if e.Name == fyne.KeyEscape {
		p.b.hide()
		return
	}
	if f := p.b.win.Canvas().OnTypedKey(); f != nil { // keep the app's shortcuts alive
		f(e)
	}
}

func (p *helpBackdrop) TypedRune(rune) {}

func newHoverHelpButton(infoText string, win fyne.Window) *hoverHelpButton {
	b := &hoverHelpButton{
		infoText: infoText,
		win:      win,
	}
	b.Text = "?"
	b.Importance = widget.LowImportance
	b.OnTapped = func() {
		if b.overlay != nil && b.pinned { // pinned: a second click closes
			b.hide()
			return
		}
		b.hide() // hover-opened: a click keeps it open instead of closing it under the cursor
		b.pinned = true
		b.show()
	}
	b.ExtendBaseWidget(b)
	return b
}

// show puts the popup on the canvas, closing whichever one was open.
func (b *hoverHelpButton) show() {
	if b.win == nil || b.overlay != nil {
		return
	}
	stopTimer(&helpShow)
	if activeHelp != nil && activeHelp != b {
		activeHelp.hide()
	}

	title := b.title
	if title == "" {
		title = "Documentation & Details"
	}
	closeBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), b.hide)
	closeBtn.Importance = widget.LowImportance
	lbl := widget.NewLabel(b.infoText)
	lbl.Wrapping = fyne.TextWrapWord
	body := container.NewVScroll(lbl)
	body.SetMinSize(fyne.NewSize(380, b.height()))
	header := container.NewBorder(nil, nil, nil, closeBtn,
		widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	content := container.NewPadded(container.NewBorder(header, nil, nil, nil, body))

	th := fyne.CurrentApp().Settings().Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	bg := canvas.NewRectangle(th.Color(theme.ColorNameOverlayBackground, v))
	// the same shadow a widget.PopUp would have drawn
	bg.Shadow = canvas.Shadow{Color: th.Color(theme.ColorNameShadow, v), BlurRadius: 14, Offset: fyne.NewPos(0, 4)}
	bg.CornerRadius = th.Size(theme.SizeNamePopupRadius)

	back := &helpBackdrop{helpPad{b: b}}
	back.FillColor = color.Transparent
	pad := &helpPad{b: b}
	pad.FillColor = color.Transparent
	overlay := container.NewWithoutLayout(back, bg, content, pad)
	b.overlay, activeHelp = overlay, b

	cv := b.win.Canvas()
	cv.Overlays().Add(overlay)
	origin := fyne.CurrentApp().Driver().AbsolutePositionForObject(overlay)
	area := fyne.NewSize(cv.Size().Width-origin.X, cv.Size().Height-origin.Y)
	size := content.MinSize()
	if size.Width > area.Width {
		size.Width = area.Width
	}
	if size.Height > area.Height {
		size.Height = area.Height
	}
	btn := fyne.CurrentApp().Driver().AbsolutePositionForObject(b).Subtract(origin)
	pos := helpPlacement(btn, b.Size(), size, area)
	back.Resize(area)
	bg.Move(pos)
	bg.Resize(size)
	content.Move(pos)
	content.Resize(size)
	pad.Move(pos)
	pad.Resize(size)

	if b.pinned { // only then may the popup take the keyboard
		b.wasFocused = cv.Focused()
		cv.Focus(back)
	}
}

func (b *hoverHelpButton) hide() {
	stopTimer(&helpShow)
	stopTimer(&helpHide)
	b.pinned = false
	if b.overlay != nil {
		b.win.Canvas().Overlays().Remove(b.overlay)
		if b.wasFocused != nil {
			b.win.Canvas().Focus(b.wasFocused)
		}
		b.wasFocused, b.overlay = nil, nil
	}
	if activeHelp == b {
		activeHelp = nil
	}
}

func (b *hoverHelpButton) scheduleHide() {
	if b.pinned || b.overlay == nil {
		return
	}
	at(&helpHide, helpHideDelay, b.hide)
}

func (b *hoverHelpButton) MouseIn(*desktop.MouseEvent) {
	if b.overlay != nil {
		return
	}
	at(&helpShow, helpShowDelay, b.show)
}

func (b *hoverHelpButton) MouseMoved(*desktop.MouseEvent) {}

func (b *hoverHelpButton) MouseOut() {
	if b.overlay != nil {
		return // the popup tracks the pointer from here on
	}
	stopTimer(&helpShow)
}
func (a *appState) isSet(o *catalog.Option) bool {
	if a.st.Raw[o.Name] != "" {
		return true
	}
	_, ok := a.st.Values[o.Name]
	return ok
}
func (a *appState) isOptionChangedFromDefault(o *catalog.Option) bool {
	if a.st.Raw[o.Name] != "" {
		return true
	}
	v, ok := a.st.Values[o.Name]
	if !ok {
		return false
	}
	def := catalog.DefaultFor(o, a.target)
	if def == nil {
		return true
	}
	return !valuesEqual(v, def)
}

func valuesEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == b
	}
	// Numbers in JSON / config may be float64 or int
	toF64 := func(v any) (float64, bool) {
		switch x := v.(type) {
		case float64:
			return x, true
		case int:
			return float64(x), true
		case int64:
			return float64(x), true
		default:
			return 0, false
		}
	}
	if na, oka := toF64(a); oka {
		if nb, okb := toF64(b); okb {
			return na == nb
		}
	}
	switch va := a.(type) {
	case string:
		vb, ok := b.(string)
		return ok && va == vb
	case bool:
		vb, ok := b.(bool)
		return ok && va == vb
	case []any:
		vb, ok := b.([]any)
		if !ok || len(va) != len(vb) {
			return false
		}
		for i := range va {
			if !valuesEqual(va[i], vb[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		vb, ok := b.(map[string]any)
		if !ok || len(va) != len(vb) {
			return false
		}
		for k, v1 := range va {
			v2, ok2 := vb[k]
			if !ok2 || !valuesEqual(v1, v2) {
				return false
			}
		}
		return true
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// pill renders a small rounded badge: text in the importance colour on a faint tint of tintHex.
func pill(text string, imp widget.Importance, tintHex string) fyne.CanvasObject {
	l := widget.NewLabelWithStyle(text, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	l.Importance = imp
	l.SizeName = theme.SizeNameCaptionText
	c := mustHex(tintHex).(color.NRGBA)
	c.A = 0x2e
	bg := canvas.NewRectangle(c)
	bg.CornerRadius = 10
	return container.NewStack(bg, l)
}

// sinceDate trims a WezTerm release id (20210203-095643-70a364eb) to its date.
func sinceDate(s string) string {
	if i := strings.IndexByte(s, '-'); i > 0 {
		return s[:i]
	}
	return s
}

// newOptionRow builds the full row for one option.
func (a *appState) newOptionRow(o *catalog.Option) *row {
	r := &row{opt: o}

	nameLabel := widget.NewLabelWithStyle(o.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Monospace: true})
	nameLabel.Truncation = fyne.TextTruncateEllipsis

	changedBadge := pill("CHANGED", widget.WarningImportance, colWarning)
	changedBadge.Hide()

	var badges []fyne.CanvasObject
	for _, t := range o.Tags {
		badges = append(badges, pill(t, widget.HighImportance, colPrimary))
	}
	if o.Deprecated != "" {
		badges = append(badges, pill("DEPRECATED", widget.DangerImportance, colDanger))
	}
	if o.Since != "" {
		badges = append(badges, pill("since "+sinceDate(o.Since), widget.LowImportance, colTextMuted))
	}
	badges = append(badges, changedBadge)
	helpBtn := newHoverHelpButton(formatOptionTooltip(o, a.target), a.win)
	badges = append(badges, helpBtn)
	badgesBox := container.NewHBox(badges...)
	helpText := o.Doc
	if o.Deprecated != "" {
		helpText += " DEPRECATED: " + o.Deprecated
	}
	helpLabel := widget.NewLabel(helpText)
	helpLabel.Wrapping = fyne.TextWrapWord
	helpLabel.Importance = widget.LowImportance

	defText := ""
	if def := catalog.DefaultFor(o, a.target); def == nil && o.DefaultNote != "" {
		defText = o.DefaultNote
	} else if def != nil {
		defText = literal(def)
	}
	var defLabel fyne.CanvasObject
	if defText != "" {
		dl := widget.NewLabelWithStyle("Default: "+defText, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
		dl.Importance = widget.SuccessImportance
		dl.Wrapping = fyne.TextWrapWord
		defLabel = dl
	}

	docs := widget.NewHyperlink("Docs ↗", nil)
	docs.SizeName = theme.SizeNameCaptionText
	docs.OnTapped = func() {
		u, err := url.Parse(catalog.DocURL(o.Name))
		if err != nil {
			dialog.ShowError(err, a.win)
			return
		}
		a.app.OpenURL(u)
	}

	star := &starButton{Button: widget.NewButton(unpinStar, nil), name: o.Name}
	star.pinned = a.st.IsPinned(o.Name)
	star.sync()
	star.OnTapped = func() {
		star.pinned = a.st.TogglePin(o.Name)
		star.sync()
		a.pinToggled()
	}

	resetBtn := widget.NewButtonWithIcon("", theme.ContentUndoIcon(), nil) // icon only: keeps the header from truncating long names
	luaToggle := widget.NewButton("Lua", nil)
	buttons := container.NewHBox(docs, star, luaToggle, resetBtn)

	bar := canvas.NewRectangle(mustHex(colBorder))
	bar.SetMinSize(fyne.NewSize(4, 4))

	refresh := func() {
		isChanged := a.isOptionChangedFromDefault(o)
		if isChanged {
			changedBadge.Show()
		} else {
			changedBadge.Hide()
		}

		if a.isSet(o) {
			nameLabel.Importance = widget.HighImportance
			resetBtn.Show()
			if isChanged {
				bar.FillColor = mustHex(colWarning)
			} else {
				bar.FillColor = mustHex(colPrimary)
			}
		} else if o.Deprecated != "" {
			nameLabel.Importance = widget.MediumImportance
			resetBtn.Hide()
			bar.FillColor = mustHex(colDanger)
		} else {
			nameLabel.Importance = widget.MediumImportance
			resetBtn.Hide()
			bar.FillColor = mustHex(colBorder)
		}
		bar.Refresh()
		nameLabel.Refresh()
		a.refreshNav()
	}

	editorBox := container.NewStack()
	buildForm := func() {
		editorBox.Objects = []fyne.CanvasObject{a.buildEditor(&o.Field,
			func() any { return a.st.Values[o.Name] },
			func(v any) {
				if v == nil {
					delete(a.st.Values, o.Name)
				} else {
					a.st.Values[o.Name] = v
				}
				a.markDirty()
				refresh()
			})}
		editorBox.Refresh()
	}
	buildRaw := func() {
		e := widget.NewMultiLineEntry()
		e.TextStyle = fyne.TextStyle{Monospace: true}
		e.SetMinRowsVisible(4)
		if raw := a.st.Raw[o.Name]; raw != "" {
			e.SetText(raw)
		} else if v, ok := a.st.Values[o.Name]; ok {
			if s, err := luagen.RenderValue(v, &o.Field, 0); err == nil {
				e.SetText(s)
			}
		}
		e.OnChanged = func(s string) {
			if s == "" {
				delete(a.st.Raw, o.Name)
			} else {
				a.st.Raw[o.Name] = s
			}
			a.markDirty()
			refresh()
		}
		editorBox.Objects = []fyne.CanvasObject{e}
		editorBox.Refresh()
	}

	rawMode := a.st.Raw[o.Name] != ""
	applyMode := func() {
		if rawMode {
			buildRaw()
			luaToggle.SetText("Form")
		} else {
			buildForm()
			luaToggle.SetText("Lua")
		}
	}
	applyMode()

	luaToggle.OnTapped = func() {
		rawMode = !rawMode
		if !rawMode {
			delete(a.st.Raw, o.Name)
		}
		applyMode()
		a.markDirty()
	}
	resetBtn.OnTapped = func() {
		delete(a.st.Values, o.Name)
		delete(a.st.Raw, o.Name)
		rawMode = false
		a.markDirty()
		refresh()
		applyMode()
	}

	r.validate = func() error {
		if a.st.Raw[o.Name] != "" || !a.isSet(o) {
			return nil
		}
		v, ok := a.st.Values[o.Name]
		if !ok {
			return nil
		}
		return validateField(v, &o.Field, o.Name)
	}
	refresh()

	nameRow := container.New(headerLayout{}, nameLabel, badgesBox, buttons)
	parts := []fyne.CanvasObject{nameRow, helpLabel}
	if defLabel != nil {
		parts = append(parts, defLabel)
	}
	contentVBox := container.NewVBox(append(parts, editorBox)...)
	card := container.NewPadded(contentVBox)
	bg := canvas.NewRectangle(mustHex(colSurface))
	bg.CornerRadius = 6
	bg.StrokeColor = mustHex(colBorder)
	bg.StrokeWidth = 1
	rowWithBg := container.NewStack(bg, card)
	r.obj = container.NewBorder(nil, nil, bar, nil, rowWithBg)
	return r
}

// starButton is the ☆/★ pin toggle. A plain button would announce the glyph
// itself, so the accessible label spells out what pressing it does.
type starButton struct {
	*widget.Button
	name   string
	pinned bool
}

const (
	pinStar   = "★"
	unpinStar = "☆"
)

func (s *starButton) sync() {
	if s.pinned {
		s.Text = pinStar
		s.Importance = widget.HighImportance
	} else {
		s.Text = unpinStar
		s.Importance = widget.LowImportance
	}
	s.Refresh()
}

func (s *starButton) AccessibilityLabel() string {
	if s.pinned {
		return "Unpin " + s.name + " from Quick Settings"
	}
	return "Pin " + s.name + " to Quick Settings"
}

// headerLayout lays out the header of an option row: the name takes whatever
// is left between the badges and the buttons, truncated with an ellipsis, so a
// long option name cannot push the Docs link, the star or Reset off the card.
// The objects are the name, the badges and the buttons.
type headerLayout struct{}

const (
	headerGap     = 6
	headerMinName = 110 // the name matters more than the platform and since tags
)

func (h headerLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	if len(objs) < 3 {
		return
	}
	name, badges, btns := objs[0], objs[1], objs[2]
	bm, dm := btns.MinSize(), badges.MinSize()

	place := func(o fyne.CanvasObject, x, width float32) {
		hgt := o.MinSize().Height
		o.Move(fyne.NewPos(x, (size.Height-hgt)/2))
		o.Resize(fyne.NewSize(width, hgt))
	}
	place(btns, size.Width-bm.Width, bm.Width)
	place(badges, size.Width-bm.Width-dm.Width-headerGap, dm.Width)
	// a truncating label reports the width of its ellipsis as its minimum, so
	// rather than measuring the name it gets everything that is left and cuts
	// its own text to fit
	nameW := size.Width - bm.Width - dm.Width - 2*headerGap
	if nameW < headerMinName {
		nameW = headerMinName
	}
	place(name, 0, nameW)
}

func (h headerLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	if len(objs) < 3 {
		return fyne.NewSize(0, 0)
	}
	nm, dm, bm := objs[0].MinSize(), objs[1].MinSize(), objs[2].MinSize()
	return fyne.NewSize(
		headerMinName+dm.Width+bm.Width+2*headerGap,
		max(nm.Height, max(dm.Height, bm.Height)))
}

// literal renders a JSON-shaped default for help text.
func literal(v any) string {
	switch x := v.(type) {
	case string:
		if x == "" {
			return `""`
		}
		return x
	case bool:
		return strconv.FormatBool(x)
	case float64:
		return numStr(x)
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = literal(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		if len(x) == 0 {
			return "{}"
		}
		parts := []string{}
		for k, e := range x {
			parts = append(parts, k+"="+literal(e))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func numStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// validateField recursively validates a stored JSON value against its schema.
func validateField(v any, f *catalog.Field, path string) error {
	switch f.Kind {
	case catalog.Int, catalog.Float, catalog.Float01:
		n, ok := v.(float64)
		if !ok {
			return fmt.Errorf("%s: expected a number", path)
		}
		return checkRange(n, f, path)
	case catalog.StringList:
		arr, ok := v.([]any)
		if !ok {
			return fmt.Errorf("%s: expected a list", path)
		}
		for _, item := range arr {
			s, ok := item.(string)
			if !ok {
				return fmt.Errorf("%s: expected string items", path)
			}
			if len(f.Enum) > 0 && !contains(f.Enum, s) {
				return fmt.Errorf("%s: %q is not one of %s", path, s, strings.Join(f.Enum, "|"))
			}
		}
	case catalog.IntList:
		arr, ok := v.([]any)
		if !ok {
			return fmt.Errorf("%s: expected a list", path)
		}
		for _, item := range arr {
			if _, ok := item.(float64); !ok {
				return fmt.Errorf("%s: expected number items", path)
			}
		}
	case catalog.StringMap, catalog.FloatMap:
		if _, ok := v.(map[string]any); !ok {
			return fmt.Errorf("%s: expected a table", path)
		}
	case catalog.Struct:
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: expected a table", path)
		}
		if len(f.Fields) == 3 && f.Fields[0].Name == "kind" {
			return nil // mouse event shape; checked by the emitter
		}
		for i := range f.Fields {
			mf := &f.Fields[i]
			mv, ok := m[mf.Name]
			if !ok || mv == nil {
				if mf.Required {
					return fmt.Errorf("%s: field %s is required", path, mf.Name)
				}
				continue
			}
			if err := validateField(mv, mf, path+"."+mf.Name); err != nil {
				return err
			}
		}
	case catalog.List:
		arr, ok := v.([]any)
		if !ok {
			return fmt.Errorf("%s: expected a list", path)
		}
		if f.FixedLen > 0 && len(arr) != f.FixedLen {
			return fmt.Errorf("%s: needs exactly %d items", path, f.FixedLen)
		}
		for i, item := range arr {
			ip := fmt.Sprintf("%s[%d]", path, i)
			if f.ItemKind != 0 {
				if err := validateField(item, &catalog.Field{Kind: f.ItemKind}, ip); err != nil {
					return err
				}
				continue
			}
			if err := validateField(item, &catalog.Field{Kind: catalog.Struct, Fields: f.Fields}, ip); err != nil {
				return err
			}
		}
	case catalog.Union:
		switch x := v.(type) {
		case string:
			if len(f.Enum) > 0 && !contains(f.Enum, x) {
				return fmt.Errorf("%s: %q is not one of %s", path, x, strings.Join(f.Enum, "|"))
			}
		case map[string]any:
			if len(x) != 1 {
				return fmt.Errorf("%s: union must have exactly one variant", path)
			}
			for variant, payload := range x {
				if !contains(f.Enum, variant) {
					return fmt.Errorf("%s: unknown variant %q", path, variant)
				}
				var vf *catalog.Field
				for i := range f.Fields {
					if f.Fields[i].Name == variant {
						vf = &f.Fields[i]
					}
				}
				if vf == nil {
					continue
				}
				if vf.Scalar {
					if err := validateField(payload, &vf.Fields[0], path); err != nil {
						return err
					}
				} else if err := validateField(payload, vf, path); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("%s: union must be a string or object", path)
		}
	case catalog.Font:
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: expected a table", path)
		}
		fontList, _ := m["font"].([]any)
		if len(fontList) == 0 {
			return fmt.Errorf("%s: at least one font is required", path)
		}
		for i, fa := range fontList {
			am, ok := fa.(map[string]any)
			if !ok {
				return fmt.Errorf("%s: font entries must be tables", path)
			}
			fam, _ := am["family"].(string)
			if fam == "" {
				return fmt.Errorf("%s.font[%d]: family is required", path, i)
			}
			if w, ok := am["weight"].(string); ok && w != "" && !weightRe.MatchString(w) {
				return fmt.Errorf("%s.font[%d]: invalid weight %q", path, i, w)
			}
		}
	case catalog.Color:
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%s: expected a color string", path)
		}
		return validateColor(s, path)
	case catalog.Dimension:
		s, ok := v.(string)
		if !ok {
			return nil // numeric dimensions are always valid
		}
		if !dimRe.MatchString(s) && !contains(f.Enum, s) {
			return fmt.Errorf("%s: invalid dimension %q", path, s)
		}
	case catalog.Action:
		switch v.(type) {
		case map[string]any, string:
			return nil
		default:
			return fmt.Errorf("%s: action must be a name or object", path)
		}
	case catalog.Leader:
		return validateField(v, &catalog.Field{Kind: catalog.Struct, Fields: catalog.LeaderFields}, path)
	case catalog.Keys:
		return validateField(v, &catalog.Field{Kind: catalog.List, Fields: catalog.KeyBindingFields}, path)
	case catalog.Mouse:
		return validateField(v, &catalog.Field{Kind: catalog.List, Fields: catalog.MouseBindingFields}, path)
	case catalog.KeyTables:
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: expected a table", path)
		}
		for name, tv := range m {
			if err := validateField(tv, &catalog.Field{Kind: catalog.List, Fields: catalog.KeyBindingFields}, path+"."+name); err != nil {
				return err
			}
		}
	case catalog.Palette:
		return validateField(v, &catalog.Field{Kind: catalog.Struct, Fields: catalog.PaletteFields}, path)
	case catalog.NamedPalettes:
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: expected a table", path)
		}
		for name, pv := range m {
			if err := validateField(pv, &catalog.Field{Kind: catalog.Palette, Fields: catalog.PaletteFields}, path+"."+name); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkRange(n float64, f *catalog.Field, path string) error {
	if f.Min != nil {
		if f.MinExclusive && n <= *f.Min {
			return fmt.Errorf("%s: must be greater than %v", path, *f.Min)
		}
		if !f.MinExclusive && n < *f.Min {
			return fmt.Errorf("%s: must be at least %v", path, *f.Min)
		}
	}
	if f.Max != nil && n > *f.Max {
		return fmt.Errorf("%s: must be at most %v", path, *f.Max)
	}
	return nil
}

func validateColor(s, path string) error {
	switch {
	case colorHex.MatchString(s), colorFn.MatchString(s), colorCol.MatchString(s), colorNam.MatchString(s):
		return nil
	case strings.EqualFold(s, "auto"), strings.EqualFold(s, "none"):
		return nil
	default:
		return fmt.Errorf("%s: invalid color %q", path, s)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// buildEditor builds the editor widget for one field value.
func (a *appState) buildEditor(f *catalog.Field, get func() any, set func(any)) fyne.CanvasObject {
	switch f.Kind {
	case catalog.Bool:
		c := widget.NewCheck("Enabled", nil)
		// Set Checked directly: SetChecked would fire OnChanged and write the
		// default into state just by rendering the row.
		if v, ok := get().(bool); ok {
			c.Checked = v
		} else if d, ok := f.Default.(bool); ok {
			c.Checked = d
		}
		c.OnChanged = func(on bool) { set(on) }
		return c
	case catalog.Int, catalog.Float:
		e := widget.NewEntry()
		if f.Kind == catalog.Int {
			e.Validator = func(s string) error {
				if !intRe.MatchString(s) {
					return fmt.Errorf("must be an integer")
				}
				n, _ := strconv.ParseFloat(s, 64)
				return checkRange(n, f, "value")
			}
		} else {
			e.Validator = func(s string) error {
				if !floatRe.MatchString(s) {
					return fmt.Errorf("must be a number")
				}
				n, _ := strconv.ParseFloat(s, 64)
				return checkRange(n, f, "value")
			}
		}
		if d := numStrOf(f.Default); d != "" {
			e.PlaceHolder = d
		}
		if v, ok := get().(float64); ok {
			e.SetText(numStr(v))
		}
		e.OnChanged = func(s string) {
			if s == "" {
				set(nil)
				return
			}
			if n, err := strconv.ParseFloat(s, 64); err == nil {
				set(n)
			}
		}
		return e

	case catalog.Float01:
		val := 0.0
		if v, ok := get().(float64); ok {
			val = v
		} else if d, ok := f.Default.(float64); ok {
			val = d
		}
		lbl := widget.NewLabel(numStr(val))
		s := widget.NewSlider(0, 1)
		s.Step = 0.01
		s.Value = val
		s.OnChangeEnded = func(v float64) {
			lbl.SetText(numStr(v))
			set(v)
		}
		return container.NewHBox(s, lbl)

	case catalog.String:
		e := widget.NewEntry()
		if d, ok := f.Default.(string); ok && d != "" {
			e.PlaceHolder = d
		} else if f.DefaultNote != "" {
			e.PlaceHolder = f.DefaultNote
		}
		if v, ok := get().(string); ok {
			e.SetText(v)
		}
		e.OnChanged = func(s string) {
			if s == "" {
				set(nil)
			} else {
				set(s)
			}
		}
		return e

	case catalog.SchemeName:
		return a.schemeEditor(get, set)

	case catalog.Enum:
		opts := append([]string{}, f.Enum...)
		if f.Name == "integrated_title_button_style" && a.target != "macos" {
			opts = dropToken(opts, "MacOsNative")
		}
		if len(opts) <= 4 { // few choices: show them all at once, tap the selected one again to unset
			defV, _ := f.Default.(string)
			const tag = " (default)"
			label := func(v string) string {
				if v == defV {
					return v + tag
				}
				return v
			}
			// One single-item radio per choice so each can carry its own "?".
			groups := make([]*widget.RadioGroup, len(opts))
			cells := make([]fyne.CanvasObject, len(opts))
			for i, v := range opts {
				i, v := i, v
				rg := widget.NewRadioGroup([]string{label(v)}, nil)
				if cur, ok := get().(string); ok && cur == v {
					rg.Selected = label(v)
				}
				rg.OnChanged = func(l string) {
					if l == "" {
						set(nil)
						return
					}
					for j, g := range groups { // exclusive: clear the others without firing callbacks
						if j != i && g.Selected != "" {
							g.Selected = ""
							g.Refresh()
						}
					}
					set(v)
				}
				groups[i] = rg
				cells[i] = a.choiceCell(rg, f.Name, v)
			}
			return choiceGrid(opts, cells)
		}
		sel := widget.NewSelect(opts, func(v string) { set(v) })
		sel.PlaceHolder = "(default: " + literal(f.Default) + ")"
		if v, ok := get().(string); ok {
			sel.Selected = v
		}
		return container.NewBorder(nil, nil, nil, a.choiceListHelp(f.Name, opts), sel)

	case catalog.Flags:
		checks := make([]*widget.Check, len(f.Enum))
		cells := make([]fyne.CanvasObject, len(f.Enum))
		var current []string
		if v, ok := get().(string); ok && v != "" && v != f.EmptyToken {
			current = strings.Split(v, "|")
		}
		changed := func() {
			var ordered []string
			for i, tok := range f.Enum {
				if checks[i].Checked {
					ordered = append(ordered, tok)
				}
			}
			if len(ordered) == 0 {
				if get() != nil {
					set(f.EmptyToken)
				}
				return
			}
			set(strings.Join(ordered, "|"))
		}
		for i, tok := range f.Enum {
			c := widget.NewCheck(tok, nil)
			c.Checked = contains(current, tok) // direct: SetChecked would fire the callback and mark the page dirty
			c.OnChanged = func(bool) { changed() }
			checks[i] = c
			cells[i] = a.choiceCell(c, f.Name, tok)
		}
		return choiceGrid(f.Enum, cells)

	case catalog.Color:
		return a.colorEditor(f, get, set)

	case catalog.Path, catalog.Dir:
		e := widget.NewEntry()
		if v, ok := get().(string); ok {
			e.SetText(v)
		}
		e.OnChanged = func(s string) {
			if s == "" {
				set(nil)
			} else {
				set(s)
			}
		}
		browse := widget.NewButtonWithIcon("Browse…", theme.FolderOpenIcon(), func() {
			if f.Kind == catalog.Dir {
				dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
					if err != nil || lu == nil {
						return
					}
					e.SetText(lu.Path())
				}, a.win)
				return
			}
			dialog.ShowFileOpen(func(r fyne.URIReadCloser, err error) {
				if err != nil || r == nil {
					return
				}
				e.SetText(r.URI().Path())
				r.Close()
			}, a.win)
		})
		return container.NewBorder(nil, nil, nil, browse, e)

	case catalog.Dimension:
		e := widget.NewEntry()
		e.Validator = func(s string) error {
			if !dimRe.MatchString(s) {
				if len(f.Enum) > 0 && contains(f.Enum, s) {
					return nil
				}
				return fmt.Errorf("use a number with optional px|pt|cell|%% unit")
			}
			return nil
		}
		if d, ok := f.Default.(string); ok {
			e.PlaceHolder = d
		}
		if v, ok := get().(string); ok {
			e.SetText(v)
		} else if v, ok := get().(float64); ok {
			e.SetText(numStr(v))
		}
		e.OnChanged = func(s string) {
			if s == "" {
				set(nil)
				return
			}
			if floatRe.MatchString(s) {
				n, _ := strconv.ParseFloat(s, 64)
				set(n)
			} else {
				set(s)
			}
		}
		return e

	case catalog.StringList, catalog.IntList:
		return a.listTextEditor(f, get, set)

	case catalog.StringMap, catalog.FloatMap:
		e := widget.NewMultiLineEntry()
		e.SetMinRowsVisible(3)
		e.TextStyle = fyne.TextStyle{Monospace: true}
		e.Validator = func(s string) error {
			for _, line := range strings.Split(s, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				k, v, ok := strings.Cut(line, "=")
				if !ok || strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
					return fmt.Errorf("use KEY=VALUE per line")
				}
				if f.Kind == catalog.FloatMap {
					if _, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err != nil {
						return fmt.Errorf("%s: value must be a number", strings.TrimSpace(k))
					}
				}
			}
			return nil
		}
		e.SetText(mapToText(get()))
		e.OnChanged = func(s string) {
			m := map[string]any{}
			for _, line := range strings.Split(s, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				k, v, _ := strings.Cut(line, "=")
				k, v = strings.TrimSpace(k), strings.TrimSpace(v)
				if k == "" || v == "" {
					continue
				}
				if f.Kind == catalog.FloatMap {
					if n, err := strconv.ParseFloat(v, 64); err == nil {
						m[k] = n
						continue
					}
				}
				m[k] = v
			}
			if len(m) == 0 {
				set(nil)
			} else {
				set(m)
			}
		}
		return e

	case catalog.Union:
		return a.unionEditor(f, get, set)

	case catalog.Lua:
		e := widget.NewMultiLineEntry()
		e.TextStyle = fyne.TextStyle{Monospace: true}
		e.SetMinRowsVisible(6)
		if v, ok := get().(string); ok {
			e.SetText(v)
		} else if d, ok := f.Default.(string); ok && d != "" {
			e.PlaceHolder = d
		}
		e.OnChanged = func(s string) {
			if s == "" {
				set(nil)
			} else {
				set(s)
			}
		}
		return e

	case catalog.Font:
		return a.fontEditor(f, get, set)

	case catalog.Palette:
		return a.paletteEditor(f, get, set)

	case catalog.NamedPalettes:
		return a.namedPalettesEditor(f, get, set)

	case catalog.Keys:
		return a.keysEditor(get, set)

	case catalog.KeyTables:
		return a.keyTablesEditor(get, set)

	case catalog.Leader, catalog.Struct:
		return a.structEditor(f, get, set)

	case catalog.Mouse:
		return a.mouseEditor(get, set)

	case catalog.List:
		return a.genericListEditor(f, get, set)

	default:
		return widget.NewLabel("(unsupported editor)")
	}
}

func numStrOf(v any) string {
	if f, ok := v.(float64); ok {
		return numStr(f)
	}
	return ""
}

func dropToken(list []string, tok string) []string {
	out := make([]string, 0, len(list))
	for _, t := range list {
		if t != tok {
			out = append(out, t)
		}
	}
	return out
}

func mapToText(v any) string {
	m, ok := v.(map[string]any)
	if !ok || len(m) == 0 {
		return ""
	}
	lines := []string{}
	for k, val := range m {
		switch x := val.(type) {
		case string:
			lines = append(lines, k+"="+x)
		case float64:
			lines = append(lines, k+"="+numStr(x))
		}
	}
	return strings.Join(lines, "\n")
}

// schemeEditor: a filterable list of built-in and user schemes, every row
// showing its colours, with the current choice in a larger strip below.
// Rows are built lazily by widget.List, so the ~1000 built-ins stay cheap.
func (a *appState) schemeEditor(get func() any, set func(any)) fyne.CanvasObject {
	schemes := a.allSchemes()
	byName := make(map[string]catalog.Scheme, len(schemes))
	for _, s := range schemes {
		byName[strings.ToLower(s.Name)] = s
	}

	// The strip previews the current choice; hovering a row replaces it and
	// leaving the row restores the choice.
	stripName := widget.NewLabel("")
	stripName.Truncation = fyne.TextTruncateEllipsis
	strip := newSchemeStrip(schemeStripPx*12, 22)
	showStrip := func(name string) {
		colors := map[string]any(nil)
		if s, ok := byName[strings.ToLower(name)]; ok {
			colors = s.Colors
		}
		setSchemeStrip(strip, colors)
		if name == "" {
			stripName.SetText("WezTerm default palette")
		} else {
			stripName.SetText(name)
		}
	}

	current, _ := get().(string)
	clear := widget.NewButton("Clear", func() {
		set(nil)
		current = ""
		showStrip("")
	})
	previewRow := container.NewBorder(nil, nil, nil, strip,
		container.NewBorder(nil, nil, nil, clear, stripName))

	// shown holds indexes into schemes; customName is offered as the first row
	// when the filter matches no built-in, so arbitrary names still work.
	var shown []int
	customName := ""
	filter := widget.NewEntry()
	filter.PlaceHolder = "Filter " + strconv.Itoa(len(schemes)) + " schemes…"

	list := widget.NewList(
		func() int { return len(shown) },
		func() fyne.CanvasObject {
			return newSchemeRow(showStrip, func() { showStrip(current) })
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			row := o.(*schemeRow)
			if id >= len(shown) {
				return
			}
			if i := shown[id]; i < 0 {
				row.set(customName, nil)
			} else {
				s := schemes[i]
				row.set(s.Name, s.Colors)
			}
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(shown) {
			return
		}
		name := customName
		if i := shown[id]; i >= 0 {
			name = schemes[i].Name
		}
		current = name
		set(name)
		showStrip(name)
	}

	refilter := func() {
		q := strings.ToLower(strings.TrimSpace(filter.Text))
		shown = shown[:0]
		customName = ""
		if q != "" {
			if _, known := byName[q]; !known {
				shown = append(shown, -1)
				customName = strings.TrimSpace(filter.Text)
			}
		}
		for i, s := range schemes {
			if q == "" || schemeMatches(s, q) {
				shown = append(shown, i)
			}
		}
		list.Refresh()
	}
	filter.OnChanged = func(string) { refilter() }
	refilter()

	showStrip(current)
	scroll := container.NewVScroll(list)
	scroll.SetMinSize(fyne.NewSize(200, 240))
	return container.NewVBox(filter, scroll, previewRow)
}

// allSchemes returns the built-in schemes plus the user's color_schemes.
func (a *appState) allSchemes() []catalog.Scheme {
	out := append([]catalog.Scheme{}, catalog.Schemes()...)
	m, _ := a.st.Values["color_schemes"].(map[string]any)
	if len(m) == 0 {
		return out
	}
	seen := make(map[string]bool, len(out)+len(m))
	for _, s := range out {
		seen[strings.ToLower(s.Name)] = true
	}
	extra := make([]string, 0, len(m))
	for k := range m {
		extra = append(extra, k)
	}
	sort.Strings(extra)
	for _, k := range extra {
		if seen[strings.ToLower(k)] {
			continue
		}
		pal, _ := m[k].(map[string]any)
		out = append(out, catalog.Scheme{Name: k, Colors: pal})
	}
	return out
}

func schemeMatches(s catalog.Scheme, lowerQuery string) bool {
	if strings.Contains(strings.ToLower(s.Name), lowerQuery) {
		return true
	}
	for _, alias := range s.Aliases {
		if strings.Contains(strings.ToLower(alias), lowerQuery) {
			return true
		}
	}
	return false
}

// schemeRow is one list row: the scheme name, its colour strip, and hover
// preview. widget.List reuses one instance per visible row, so the content and
// the hover callbacks are set together.
type schemeRow struct {
	widget.BaseWidget
	name    *widget.Label
	strip   *canvas.Image
	hovered string
	onHover func(string)
	onLeave func()
}

func newSchemeRow(onHover func(string), onLeave func()) *schemeRow {
	r := &schemeRow{onHover: onHover, onLeave: onLeave}
	r.name = widget.NewLabel("")
	r.name.Truncation = fyne.TextTruncateEllipsis
	r.strip = newSchemeStrip(schemeStripPx*7, 16)
	r.ExtendBaseWidget(r)
	return r
}

func (r *schemeRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(nil, nil, nil, r.strip, r.name))
}

// set fills the row in; nil colors means a name with no known palette.
func (r *schemeRow) set(name string, colors map[string]any) {
	r.name.SetText(name)
	setSchemeStrip(r.strip, colors)
	r.hovered = name
}

func (r *schemeRow) MouseIn(*desktop.MouseEvent) {
	if r.onHover != nil {
		r.onHover(r.hovered)
	}
}

func (r *schemeRow) MouseOut() {
	if r.onLeave != nil {
		r.onLeave()
	}
}

func newSchemeStrip(w, h float32) *canvas.Image {
	img := canvas.NewImageFromImage(schemeStrip(nil))
	img.ScaleMode = canvas.ImageScalePixels
	img.FillMode = canvas.ImageFillStretch
	img.SetMinSize(fyne.NewSize(w, h))
	return img
}

// colorEditor: entry + swatch + picker button.
func (a *appState) colorEditor(f *catalog.Field, get func() any, set func(any)) fyne.CanvasObject {
	e := widget.NewEntry()
	e.Validator = func(s string) error {
		if s == "" {
			return nil
		}
		return validateColor(s, "color")
	}
	if d, ok := f.Default.(string); ok {
		e.PlaceHolder = d
	}
	current := ""
	if v, ok := get().(string); ok {
		current = v
		e.SetText(v)
	}
	swatch := canvas.NewRectangle(parseColor(current))
	swatch.SetMinSize(fyne.NewSize(24, 24))
	e.OnChanged = func(s string) {
		current = s
		swatch.FillColor = parseColor(s)
		swatch.Refresh()
		if s == "" {
			set(nil)
		} else {
			set(s)
		}
	}
	pick := widget.NewButton("Pick", func() {
		picker := dialog.NewColorPicker("Pick a color", "", func(c color.Color) {
			if c == nil {
				return
			}
			r32, g32, b32, a32 := c.RGBA()
			if a32 == 0xFFFF {
				e.SetText(fmt.Sprintf("#%02x%02x%02x", r32>>8, g32>>8, b32>>8))
			} else {
				alpha := float64(a32) / 0xFFFF
				e.SetText(fmt.Sprintf("rgba(%d,%d,%d,%.2f)", r32>>8, g32>>8, b32>>8, alpha))
			}
		}, a.win)
		picker.Advanced = true
		if cur := parseColor(current); cur != nil {
			picker.SetColor(cur)
		}
		picker.Show()
	})
	return container.NewBorder(nil, nil, swatch, pick, e)
}

// parseColor parses #rgb/#rrggbb into a color; other syntaxes return nil.
func parseColor(s string) color.Color {
	if !strings.HasPrefix(s, "#") {
		return nil
	}
	hex := s[1:]
	switch len(hex) {
	case 3:
		r, _ := strconv.ParseUint(hex[:1]+hex[:1], 16, 8)
		g, _ := strconv.ParseUint(hex[1:2]+hex[1:2], 16, 8)
		b, _ := strconv.ParseUint(hex[2:3]+hex[2:3], 16, 8)
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xFF}
	case 6:
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xFF}
	default:
		return nil
	}
}

// listTextEditor edits StringList/IntList as one item per line.
func (a *appState) listTextEditor(f *catalog.Field, get func() any, set func(any)) fyne.CanvasObject {
	e := widget.NewMultiLineEntry()
	e.SetMinRowsVisible(3)
	e.TextStyle = fyne.TextStyle{Monospace: true}
	if f.Kind == catalog.IntList {
		e.Validator = func(s string) error {
			for _, line := range strings.Split(s, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if !intRe.MatchString(line) {
					return fmt.Errorf("%q is not an integer", line)
				}
			}
			return nil
		}
	} else if len(f.Enum) > 0 {
		e.Validator = func(s string) error {
			for _, line := range strings.Split(s, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if !contains(f.Enum, line) {
					return fmt.Errorf("%q is not one of %s", line, strings.Join(f.Enum, "|"))
				}
			}
			return nil
		}
	}
	if arr, ok := get().([]any); ok {
		var lines []string
		for _, item := range arr {
			switch x := item.(type) {
			case string:
				lines = append(lines, x)
			case float64:
				lines = append(lines, numStr(x))
			}
		}
		e.SetText(strings.Join(lines, "\n"))
	}
	e.OnChanged = func(s string) {
		var arr []any
		for _, line := range strings.Split(s, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if f.Kind == catalog.IntList {
				if n, err := strconv.ParseFloat(line, 64); err == nil {
					arr = append(arr, n)
					continue
				}
			}
			arr = append(arr, line)
		}
		if len(arr) == 0 {
			set(nil)
		} else {
			set(arr)
		}
	}

	var extra fyne.CanvasObject
	if o := a.rowOptionFor(f); o != nil && (o.Name == "font_dirs" || o.Name == "color_scheme_dirs") {
		addBtn := widget.NewButtonWithIcon("Add folder…", theme.FolderNewIcon(), func() {
			dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
				if err != nil || lu == nil {
					return
				}
				cur := strings.TrimRight(e.Text, "\n")
				if cur == "" {
					e.SetText(lu.Path())
				} else {
					e.SetText(cur + "\n" + lu.Path())
				}
			}, a.win)
		})
		extra = addBtn
	}
	if extra != nil {
		return container.NewVBox(e, extra)
	}
	return e
}

// rowOptionFor finds the mounted option owning field f (top-level rows only).
func (a *appState) rowOptionFor(f *catalog.Field) *catalog.Option {
	for i := range catalog.Options {
		if &catalog.Options[i].Field == f {
			return &catalog.Options[i]
		}
	}
	return nil
}
