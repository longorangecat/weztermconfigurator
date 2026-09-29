package ui

import (
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"weztermconfigurator/internal/catalog"
	"weztermconfigurator/internal/state"
)

// The preview draws a fake WezTerm window from the current state. Everything
// numeric about it lives in previewModel so it can be unit tested without a
// canvas; the widget below only turns that model into rectangles and text.

// Nominal geometry constants used to convert state values into a model that is
// independent of the pane's pixel size (the real metrics are applied at draw
// time).
const (
	previewMargin   = 6.0
	previewCols     = 52.0 // columns the fake window is sized to show
	previewCharAsp  = 0.58 // monospace advance width / font size
	nominalCellPx   = 8.0  // cell size assumed when converting "px"/"pt" padding
	nominalCols     = 40.0 // columns assumed when converting "%" padding
	previewDebounce = 100 * time.Millisecond
)

// previewModel is everything the preview needs from the state, resolved.
type previewModel struct {
	Scheme      string
	Foreground  color.NRGBA
	Background  color.NRGBA
	CursorBG    color.NRGBA
	CursorFG    color.NRGBA
	SelectionBG color.NRGBA
	SelectionFG color.NRGBA
	ANSI        [16]color.NRGBA // 8 ansi then 8 brights
	Opacity     float32         // window_background_opacity, 0..1
	FontSize    float32
	LineHeight  float32
	Padding     [4]float32 // left, right, top, bottom, in cells
	CursorStyle string
	FontFamily  string
	TitleBar    bool
	TabBar      bool
	TabBottom   bool
	FancyTabs   bool
	HideOneTab  bool
}

func rgb(r, g, b uint8) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: 0xff} }

// defaultPreviewModel is WezTerm's out-of-the-box dark scheme.
func defaultPreviewModel() previewModel {
	ansi := [8]color.NRGBA{
		rgb(0x14, 0x14, 0x18), rgb(0xd7, 0x3a, 0x49), rgb(0x2e, 0xa0, 0x43), rgb(0xb5, 0x8e, 0x0b),
		rgb(0x25, 0x6c, 0xbb), rgb(0xa3, 0x4c, 0xa3), rgb(0x18, 0x8a, 0x8a), rgb(0xd0, 0xd0, 0xd0),
	}
	m := previewModel{
		Foreground:  rgb(0xe6, 0xe6, 0xe6),
		Background:  rgb(0x14, 0x14, 0x18),
		CursorBG:    rgb(0xe6, 0xe6, 0xe6),
		CursorFG:    rgb(0x14, 0x14, 0x18),
		SelectionBG: rgb(0x3a, 0x4a, 0x6b),
		SelectionFG: rgb(0xff, 0xff, 0xff),
		Opacity:     1,
		FontSize:    12,
		LineHeight:  1,
		Padding:     [4]float32{1, 1, 0.5, 0.5},
		CursorStyle: "SteadyBlock",
		TitleBar:    true,
		TabBar:      true,
		FancyTabs:   true,
	}
	copy(m.ANSI[:8], ansi[:])
	for i := 0; i < 8; i++ {
		m.ANSI[8+i] = lighten(m.ANSI[i], 0.28)
	}
	return m
}

// buildPreviewModel resolves the current state into everything the preview
// draws. Unknown names, wrong types and out-of-range numbers degrade to the
// catalog default or the built-in default; it never panics.
func buildPreviewModel(st *state.State, target string) previewModel {
	m := defaultPreviewModel()
	if st == nil {
		return m
	}
	if name := stringOpt(st, target, "color_scheme"); name != "" {
		if s, ok := findScheme(st, name); ok {
			m.Scheme = name
			applyPalette(&m, s.Colors)
		}
	}
	if pal, ok := st.Values["colors"].(map[string]any); ok {
		applyPalette(&m, pal)
	}
	if v, ok := numberOpt(st, target, "window_background_opacity"); ok {
		m.Opacity = clamp01(float32(v))
	}
	if v, ok := numberOpt(st, target, "font_size"); ok && v > 0 {
		m.FontSize = clampF(float32(v), 4, 200)
	}
	if v, ok := numberOpt(st, target, "line_height"); ok && v > 0 {
		m.LineHeight = clampF(float32(v), 0.5, 5)
	}
	if pad, ok := st.Values["window_padding"].(map[string]any); ok {
		for i, key := range [4]string{"left", "right", "top", "bottom"} {
			if c, ok := dimCells(pad[key]); ok {
				m.Padding[i] = clampF(c, 0, 20)
			}
		}
	}
	if v := stringOpt(st, target, "default_cursor_style"); v != "" {
		m.CursorStyle = v
	}
	if v, ok := boolOpt(st, target, "enable_tab_bar"); ok {
		m.TabBar = v
	}
	if v, ok := boolOpt(st, target, "tab_bar_at_bottom"); ok {
		m.TabBottom = v
	}
	if v, ok := boolOpt(st, target, "use_fancy_tab_bar"); ok {
		m.FancyTabs = v
	}
	if v, ok := boolOpt(st, target, "hide_tab_bar_if_only_one_tab"); ok {
		m.HideOneTab = v
	}
	if v := stringOpt(st, target, "window_decorations"); v != "" {
		m.TitleBar = strings.Contains(strings.ToUpper(v), "TITLE")
	}
	m.FontFamily = fontFamily(st)
	return m
}

// ---- state access helpers (value, else catalog default, else nothing) ----

func numberOpt(st *state.State, target, name string) (float64, bool) {
	if st != nil {
		if v, ok := asNumber(st.Values[name]); ok {
			return v, true
		}
	}
	if o := catalog.Find(name); o != nil {
		if v, ok := asNumber(catalog.DefaultFor(o, target)); ok {
			return v, true
		}
	}
	return 0, false
}

func boolOpt(st *state.State, target, name string) (bool, bool) {
	if st != nil {
		if v, ok := st.Values[name].(bool); ok {
			return v, true
		}
	}
	if o := catalog.Find(name); o != nil {
		if v, ok := catalog.DefaultFor(o, target).(bool); ok {
			return v, true
		}
	}
	return false, false
}

func stringOpt(st *state.State, target, name string) string {
	if st != nil {
		if v, ok := st.Values[name].(string); ok {
			return v
		}
	}
	if o := catalog.Find(name); o != nil {
		if v, ok := catalog.DefaultFor(o, target).(string); ok {
			return v
		}
	}
	return ""
}

func asNumber(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	}
	return 0, false
}

// findScheme looks a name up in the user's color_schemes first, then the
// built-ins.
func findScheme(st *state.State, name string) (catalog.Scheme, bool) {
	if m, ok := st.Values["color_schemes"].(map[string]any); ok {
		if pal, ok := m[name].(map[string]any); ok {
			return catalog.Scheme{Name: name, Colors: pal}, true
		}
	}
	ss := catalog.Schemes()
	for i := range ss {
		if ss[i].Name == name {
			return ss[i], true
		}
	}
	return catalog.Scheme{}, false
}

func applyPalette(m *previewModel, c map[string]any) {
	if c == nil {
		return
	}
	set := func(key string, dst *color.NRGBA) {
		if s, ok := c[key].(string); ok {
			if col, ok := hexColor(s); ok {
				*dst = col
			}
		}
	}
	set("foreground", &m.Foreground)
	set("background", &m.Background)
	set("cursor_bg", &m.CursorBG)
	set("cursor_fg", &m.CursorFG)
	set("selection_bg", &m.SelectionBG)
	set("selection_fg", &m.SelectionFG)
	if list, ok := c["ansi"].([]any); ok {
		setAnsi(m.ANSI[0:8], list)
	}
	if list, ok := c["brights"].([]any); ok {
		setAnsi(m.ANSI[8:16], list)
	}
}

func setAnsi(dst []color.NRGBA, list []any) {
	for i := 0; i < len(dst) && i < len(list); i++ {
		if s, ok := list[i].(string); ok {
			if col, ok := hexColor(s); ok {
				dst[i] = col
			}
		}
	}
}

// fontFamily pulls the first family out of the FontAttributes list.
func fontFamily(st *state.State) string {
	if st == nil {
		return ""
	}
	obj, ok := st.Values["font"].(map[string]any)
	if !ok {
		return ""
	}
	list, ok := obj["font"].([]any)
	if !ok {
		return ""
	}
	for _, item := range list {
		if attrs, ok := item.(map[string]any); ok {
			if fam, ok := attrs["family"].(string); ok && fam != "" {
				return fam
			}
		}
	}
	return ""
}

// dimCells converts a window_padding dimension into cells. px/pt/% are
// resolved against nominal metrics, since the real cell size is only known at
// draw time.
func dimCells(v any) (float32, bool) {
	var s string
	switch x := v.(type) {
	case string:
		s = strings.TrimSpace(x)
	case float64:
		s = strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		s = strconv.Itoa(x)
	default:
		return 0, false
	}
	if s == "" {
		return 0, false
	}
	unit := "cell"
	for _, u := range []string{"px", "pt", "cell", "%"} {
		if strings.HasSuffix(s, u) {
			unit = u
			s = strings.TrimSpace(strings.TrimSuffix(s, u))
			break
		}
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	switch unit {
	case "px":
		return float32(n / nominalCellPx), true
	case "pt":
		return float32(n * (4.0 / 3.0) / nominalCellPx), true
	case "%":
		return float32(n / 100.0 * nominalCols), true
	}
	return float32(n), true
}

// hexColor parses #rgb, #rgba, #rrggbb and #rrggbbaa.
func hexColor(s string) (color.NRGBA, bool) {
	if !strings.HasPrefix(s, "#") {
		return color.NRGBA{}, false
	}
	h := s[1:]
	expand := func(c string) uint8 {
		n, _ := strconv.ParseUint(c, 16, 8)
		return uint8(n)
	}
	// The short forms use one digit per channel, so each digit is doubled.
	digit := func(c string) uint8 { return expand(c + c) }
	rgba := func(r, g, b, a uint8) (color.NRGBA, bool) { return color.NRGBA{r, g, b, a}, true }
	switch len(h) {
	case 3:
		return rgba(digit(h[0:1]), digit(h[1:2]), digit(h[2:3]), 0xff)
	case 4:
		return rgba(digit(h[0:1]), digit(h[1:2]), digit(h[2:3]), digit(h[3:4]))
	case 6:
		return rgba(expand(h[0:2]), expand(h[2:4]), expand(h[4:6]), 0xff)
	case 8:
		return rgba(expand(h[0:2]), expand(h[2:4]), expand(h[4:6]), expand(h[6:8]))
	}
	return color.NRGBA{}, false
}

func clamp01(v float32) float32 { return clampF(v, 0, 1) }

func clampF(v, lo, hi float32) float32 {
	return float32(math.Min(float64(hi), math.Max(float64(lo), float64(v))))
}

// lighten blends c towards white by amount.
func lighten(c color.NRGBA, amount float32) color.NRGBA {
	blend := func(a, b uint8) uint8 { return uint8(float32(a) + (float32(b)-float32(a))*amount) }
	return color.NRGBA{blend(c.R, 0xff), blend(c.G, 0xff), blend(c.B, 0xff), c.A}
}

// shade blends c towards other by amount.
func shade(c, other color.NRGBA, amount float32) color.NRGBA {
	blend := func(a, b uint8) uint8 { return uint8(float32(a) + (float32(b)-float32(a))*amount) }
	return color.NRGBA{blend(c.R, other.R), blend(c.G, other.G), blend(c.B, other.B), c.A}
}

func withAlpha(c color.NRGBA, alpha float32) color.NRGBA {
	c.A = uint8(clampF(alpha, 0, 1) * 255)
	return c
}

// ---- widget ----

// previewPane renders a fake terminal window that follows the current state.
type previewPane struct {
	widget.BaseWidget
	app   *appState
	model previewModel
	timer *time.Timer
}

func newPreviewPane(a *appState) *previewPane {
	p := &previewPane{app: a}
	p.ExtendBaseWidget(p)
	p.model = p.build()
	return p
}

func (p *previewPane) build() previewModel {
	if p.app == nil {
		return defaultPreviewModel()
	}
	return buildPreviewModel(p.app.st, p.app.target)
}

// CreateRenderer implements fyne.Widget.
func (p *previewPane) CreateRenderer() fyne.WidgetRenderer { return &previewPaneRenderer{pane: p} }

// Update re-reads the state and redraws. It coalesces bursts (typing in an
// entry) into one redraw and may be called from any goroutine.
func (p *previewPane) Update() {
	if p.timer != nil {
		p.timer.Stop()
	}
	p.timer = time.AfterFunc(previewDebounce, func() {
		if fyne.CurrentApp() == nil {
			p.redraw()
			return
		}
		fyne.Do(p.redraw)
	})
}

func (p *previewPane) redraw() {
	p.model = p.build()
	p.Refresh()
}

type previewPaneRenderer struct {
	pane    *previewPane
	objects []fyne.CanvasObject
	size    fyne.Size
	built   bool
}

func (r *previewPaneRenderer) Layout(size fyne.Size) {
	if r.built && r.size == size {
		return
	}
	r.size = size
	r.built = true
	r.rebuild()
}

func (r *previewPaneRenderer) rebuild() {
	r.objects = previewObjects(r.pane.model, r.size)
}

func (r *previewPaneRenderer) MinSize() fyne.Size { return fyne.NewSize(300, 260) }
func (r *previewPaneRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}
func (r *previewPaneRenderer) Destroy() {}
func (r *previewPaneRenderer) Refresh() { r.rebuild() }

// ---- drawing ----

// previewMetrics are the pixel metrics derived from the model and the pane size.
type previewMetrics struct {
	font, cell, line float32
	titleH, tabH     float32
}

func (m previewModel) metrics(w, h float32) previewMetrics {
	font := clampF(w/(previewCols*previewCharAsp)*(m.FontSize/12), 5, 34)
	mt := previewMetrics{}
	// Shrink until at least a few body lines fit, so a tall font does not
	// squeeze the terminal away entirely.
	for i := 0; i < 8; i++ {
		mt.font = font
		mt.cell = font * previewCharAsp
		mt.line = maxF(font*1.25*m.LineHeight, font*0.95)
		mt.titleH = 0
		if m.TitleBar {
			mt.titleH = maxF(font*2.0, 18)
		}
		mt.tabH = 0
		if m.showTabs() {
			mt.tabH = maxF(font*1.9, 20)
		}
		body := h - 2*previewMargin - mt.titleH - mt.tabH
		if body/mt.line >= 4 || font <= 5 {
			break
		}
		font *= 0.9
	}
	return mt
}

// showTabs reports whether the tab bar is drawn; the preview shows two tabs
// unless hide_tab_bar_if_only_one_tab is on, in which case it shows one.
func (m previewModel) showTabs() bool { return m.TabBar }

func (m previewModel) tabs() []string {
	if m.HideOneTab {
		return []string{"zsh"}
	}
	return []string{"zsh", "neovim"}
}

func previewObjects(m previewModel, size fyne.Size) []fyne.CanvasObject {
	w, h := size.Width, size.Height
	if w < previewMargin*2+40 || h < 90 {
		return nil
	}
	mt := m.metrics(w, h)

	cap1, cap2 := m.captions()
	capH := float32(0)
	if cap1 != "" {
		capH += 13
	}
	if cap2 != "" {
		capH += 13
	}

	winX, winY := float32(previewMargin), float32(previewMargin)
	winW, winH := w-2*previewMargin, h-2*previewMargin-capH-2
	if winW < 20 || winH < 20 {
		return nil
	}

	objs := make([]fyne.CanvasObject, 0, 48)
	add := func(o fyne.CanvasObject, x, y float32, w, h float32) {
		o.Move(fyne.NewPos(x, y))
		o.Resize(fyne.NewSize(w, h))
		objs = append(objs, o)
	}
	rect := func(c color.NRGBA, x, y, w, h float32) {
		add(canvas.NewRectangle(c), x, y, w, h)
	}
	text := func(s string, c color.NRGBA, size, x, y float32) {
		if s == "" {
			return
		}
		t := canvas.NewText(s, c)
		t.TextSize = size
		t.TextStyle = fyne.TextStyle{Monospace: true}
		add(t, x, y, textWidth(s, size), size*1.4)
	}

	y := winY
	if m.TitleBar {
		bar := shade(m.Background, m.Foreground, 0.10)
		rect(bar, winX, y, winW, mt.titleH)
		dots := [3]color.NRGBA{m.ANSI[9], m.ANSI[11], m.ANSI[10]} // red, yellow, green
		d := maxF(mt.titleH*0.22, 3)
		for i, c := range dots {
			dot := canvas.NewCircle(c)
			add(dot, winX+winW-12-float32(2-i)*16, y+mt.titleH/2-d/2, d, d)
		}
		title := "wezterm — ~/src/weztermconfigurator"
		t := canvas.NewText(title, shade(m.Foreground, m.Background, 0.2))
		t.TextSize = maxF(mt.font*0.95, 9)
		t.TextStyle = fyne.TextStyle{Monospace: true}
		t.Alignment = fyne.TextAlignCenter
		add(t, winX, y+mt.titleH/2-t.TextSize*0.6, winW, mt.titleH)
		y += mt.titleH
	}

	bodyY := y
	bodyH := winH - mt.titleH
	if m.showTabs() {
		if m.TabBottom {
			bodyH -= mt.tabH
		} else {
			bodyY = y + mt.tabH
			bodyH -= mt.tabH
		}
		tabY := y
		if m.TabBottom {
			tabY = y + winH - mt.titleH - mt.tabH
		}
		objs = append(objs, m.tabBarObjects(winX, tabY, winW, mt)...)
	}

	// Terminal background: checkerboard so window_background_opacity is visible.
	check := canvas.NewImageFromImage(checkerboard())
	check.ScaleMode = canvas.ImageScalePixels
	check.FillMode = canvas.ImageFillStretch
	add(check, winX, bodyY, winW, bodyH)
	rect(withAlpha(m.Background, m.Opacity), winX, bodyY, winW, bodyH)

	// Body text.
	padL, padR := m.Padding[0]*mt.cell, m.Padding[1]*mt.cell
	padT, padB := m.Padding[2]*mt.line, m.Padding[3]*mt.line
	tx := winX + padL
	ty := bodyY + padT
	cols := int((winW - padL - padR) / mt.cell)
	rows := int((bodyH - padT - padB) / mt.line)
	lines := m.sampleLines()
	used := 0
	var lastX, lastY float32
	for i := 0; i < len(lines) && used < rows-1; i++ {
		lx := tx
		col := 0
		for _, sg := range lines[i] {
			if col >= cols {
				break
			}
			s := sg.text
			if col+len(s) > cols {
				s = s[:cols-col]
			}
			text(s, sg.color, mt.font, lx, ty)
			lx += float32(len(s)) * mt.cell
			col += len(s)
		}
		lastX, lastY = lx, ty
		ty += mt.line
		used++
	}
	// ANSI palette strip on its own line.
	if used < rows {
		sw := mt.cell * 0.9
		for i, c := range m.ANSI {
			rect(c, tx+float32(i)*sw, ty+mt.line*0.15, sw, mt.line*0.7)
		}
	}

	// Cursor at the end of the last line.
	if used > 0 {
		cw, ch := mt.cell, mt.line
		var cx, cy float32 = lastX, lastY
		switch {
		case strings.HasSuffix(strings.ToLower(m.CursorStyle), "underline"):
			ch = maxF(mt.line*0.12, 1)
		case strings.HasSuffix(strings.ToLower(m.CursorStyle), "bar"):
			cw = maxF(mt.cell*0.16, 1)
		}
		if cx+cw > winX+winW-padR {
			cx = tx
			cy = lastY + mt.line
		}
		rect(m.CursorBG, cx, cy+mt.line*0.08, cw, ch)
	}

	// Window border on top of everything.
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = colBorderOf(m.Background)
	border.StrokeWidth = 1
	add(border, winX, winY, winW, winH)

	// Captions.
	cy := winY + winH + 4
	if cap1 != "" {
		text(fitText(cap1, int(winW/(11*0.72))), mustHex(colTextMuted).(color.NRGBA), 11, winX, cy)
		cy += 13
	}
	if cap2 != "" {
		text(fitText(cap2, int(winW/(11*0.72))), mustHex(colTextMuted).(color.NRGBA), 11, winX, cy)
	}
	return objs
}

func (m previewModel) captions() (string, string) {
	opacity := "opaque"
	if m.Opacity < 0.999 {
		opacity = strconv.Itoa(int(m.Opacity*100+0.5)) + "% opacity"
	}
	c1 := m.CursorStyle + " cursor · " + trimNum(m.FontSize) + "pt · " + trimNum(m.LineHeight) + " line height · " + opacity
	c2 := ""
	if m.FontFamily != "" {
		c2 = "Font: " + m.FontFamily + " (preview uses a generic monospace)"
	}
	return c1, c2
}

func trimNum(v float32) string {
	return strconv.FormatFloat(float64(v), 'f', -1, 32)
}

// tabBarObjects draws the tab strip; the y argument is its top edge.
func (m previewModel) tabBarObjects(x, y, w float32, mt previewMetrics) []fyne.CanvasObject {
	objs := make([]fyne.CanvasObject, 0, 8+2*len(m.tabs()))
	add := func(o fyne.CanvasObject, ox, oy, ow, oh float32) {
		o.Move(fyne.NewPos(x+ox, y+oy))
		o.Resize(fyne.NewSize(ow, oh))
		objs = append(objs, o)
	}
	bar := shade(m.Background, m.Foreground, 0.06)
	add(canvas.NewRectangle(bar), 0, 0, w, mt.tabH)

	tabs := m.tabs()
	gap := float32(0)
	radius := float32(0)
	if m.FancyTabs {
		gap = 4
		radius = 4
	}
	tw := minF(w/float32(len(tabs)+1), mt.font*9)
	for i, name := range tabs {
		tx := float32(4) + float32(i)*(tw+gap)
		active := i == 0
		fill := bar
		if active {
			fill = shade(m.Background, m.Foreground, 0.18)
		}
		r := canvas.NewRectangle(fill)
		r.CornerRadius = radius
		add(r, tx, 3, tw, mt.tabH-3)
		fg := shade(m.Foreground, m.Background, 0.45)
		if active {
			fg = m.Foreground
		}
		label := name
		if i == 1 {
			label = name + " *"
		}
		t := canvas.NewText(label, fg)
		t.TextSize = maxF(mt.font*0.9, 8)
		t.TextStyle = fyne.TextStyle{Monospace: true}
		add(t, tx+6, mt.tabH/2-t.TextSize*0.62, textWidth(label, t.TextSize), t.TextSize*1.4)
		if active && m.FancyTabs {
			// The fancy tab bar marks the active tab with a colour stripe.
			add(canvas.NewRectangle(m.ANSI[5]), tx, 3, 2, mt.tabH-3)
		}
	}
	return objs
}

// sampleLines is the fake terminal content: a prompt, a coloured listing, a
// git diff and its stat line.
func (m previewModel) sampleLines() [][]previewSeg {
	f, d := m.Foreground, shade(m.Foreground, m.Background, 0.35) // dim
	dir, exe, plain := m.ANSI[12], m.ANSI[10], m.Foreground
	return [][]previewSeg{
		{{text: "❯ ", color: m.ANSI[6]}, {text: "ls --color", color: f}, {text: " src", color: dir}},
		{{text: "  internal/", color: dir}, {text: "  ui/", color: dir},
			{text: "  main.go", color: exe}, {text: "  go.mod", color: plain}},
		{{text: "❯ ", color: m.ANSI[6]}, {text: "git diff --stat", color: f}},
		{{text: " internal/ui/preview.go", color: plain}, {text: " | ", color: d},
			{text: "128 ", color: m.ANSI[10]}, {text: "++++", color: m.ANSI[2]},
			{text: "-----", color: m.ANSI[1]}},
		{{text: "-", color: m.ANSI[1]}, {text: "\tfmt.Println(\"hi\")", color: d}},
		{{text: "+", color: m.ANSI[2]}, {text: "\tfmt.Println(hello())", color: m.ANSI[10]}},
	}
}

type previewSeg struct {
	text  string
	color color.NRGBA
}

func colBorderOf(bg color.NRGBA) color.NRGBA { return shade(bg, rgb(0xff, 0xff, 0xff), 0.28) }

// textWidth is the drawn width of s in a monospace face at the given size. The
// box is a deliberate over-estimate: canvas.Text wraps at its width, so extra
// room costs nothing while a tight box would break lines.
func textWidth(s string, size float32) float32 {
	return float32(utf8.RuneCountInString(s))*size*0.72 + 2
}

// fitText truncates s to maxChars cells so it cannot wrap out of its box.
func fitText(s string, maxChars int) string {
	if maxChars < 2 || utf8.RuneCountInString(s) <= maxChars {
		return s
	}
	r := []rune(s)
	return string(r[:maxChars-1]) + "…"
}

// schemeStripPx is the width of one colour block in a scheme swatch strip.
const schemeStripPx = 8

// schemeStrip renders a palette as a 19 pixel wide image: background,
// foreground, cursor, then the 8 ANSI and 8 bright colours. One image per row
// keeps a list of a thousand schemes cheap.
func schemeStrip(colors map[string]any) *image.NRGBA {
	const n = 19
	img := image.NewNRGBA(image.Rect(0, 0, n, 1))
	for x := range n {
		img.SetNRGBA(x, 0, rgb(0x33, 0x36, 0x40))
	}
	put := func(i int, s string) {
		if c, ok := hexColor(s); ok {
			img.SetNRGBA(i, 0, c)
		}
	}
	for i, key := range [3]string{"background", "foreground", "cursor_bg"} {
		if s, ok := colors[key].(string); ok {
			put(i, s)
		}
	}
	for _, group := range []struct {
		key string
		at  int
	}{{"ansi", 3}, {"brights", 11}} {
		list, _ := colors[group.key].([]any)
		for i := range 8 {
			if i >= len(list) {
				break
			}
			if s, ok := list[i].(string); ok {
				put(group.at+i, s)
			}
		}
	}
	return img
}

// setSchemeStrip points an existing strip image at a new palette.
func setSchemeStrip(img *canvas.Image, colors map[string]any) {
	img.Image = schemeStrip(colors)
	img.Refresh()
}

// checkerboard is an 8x8 two-tone tile shown behind transparent windows.
func checkerboard() image.Image {
	const n = 8
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for y := range n {
		for x := range n {
			c := rgb(0x2c, 0x2f, 0x38)
			if (x/4+y/4)%2 == 0 {
				c = rgb(0x55, 0x59, 0x66)
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func minF(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
