package ui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"weztermconfigurator/internal/state"
)

func TestBuildPreviewModelDefaults(t *testing.T) {
	m := buildPreviewModel(state.New("linux"), "linux")
	d := defaultPreviewModel()
	if m.Background != d.Background || m.Foreground != d.Foreground {
		t.Fatalf("empty state should use the built-in dark scheme, got bg=%v fg=%v", m.Background, m.Foreground)
	}
	if m.Opacity != 1 || m.FontSize != 12 || m.LineHeight != 1 {
		t.Fatalf("catalog defaults not applied: opacity=%v font=%v line=%v", m.Opacity, m.FontSize, m.LineHeight)
	}
	if !m.TitleBar || !m.TabBar || !m.FancyTabs || m.TabBottom || m.HideOneTab {
		t.Fatalf("window/tab defaults wrong: %+v", m)
	}
	if m.CursorStyle != "SteadyBlock" {
		t.Fatalf("cursor default = %q", m.CursorStyle)
	}
	// window_padding default is 1cell sides, 0.5cell top/bottom.
	if m.Padding != [4]float32{1, 1, 0.5, 0.5} {
		t.Fatalf("padding default = %v", m.Padding)
	}
}

func TestBuildPreviewModelNilState(t *testing.T) {
	if m := buildPreviewModel(nil, "linux"); m.Opacity != 1 || m.Foreground != defaultPreviewModel().Foreground {
		t.Fatal("nil state must fall back to defaults")
	}
}

func TestBuildPreviewModelSchemeLookup(t *testing.T) {
	st := state.New("linux")
	st.Values["color_scheme"] = "Solarized Dark - Patched"
	m := buildPreviewModel(st, "linux")
	if m.Scheme != "Solarized Dark - Patched" {
		t.Fatalf("scheme = %q", m.Scheme)
	}
	if m.Background == defaultPreviewModel().Background {
		t.Fatal("scheme background was not applied")
	}
	if m.ANSI[0] == m.ANSI[1] {
		t.Fatal("ansi colours were not taken from the scheme")
	}
}

func TestBuildPreviewModelUserSchemeWins(t *testing.T) {
	st := state.New("linux")
	st.Values["color_scheme"] = "mine"
	st.Values["color_schemes"] = map[string]any{
		"mine": map[string]any{
			"foreground": "#111111",
			"background": "#222222",
			"ansi":       []any{"#010101", "#020202", "#030303", "#040404", "#050505", "#060606", "#070707", "#080808"},
		},
	}
	m := buildPreviewModel(st, "linux")
	if m.Foreground != (color.NRGBA{0x11, 0x11, 0x11, 0xff}) || m.Background != (color.NRGBA{0x22, 0x22, 0x22, 0xff}) {
		t.Fatalf("user scheme not used: fg=%v bg=%v", m.Foreground, m.Background)
	}
	if m.ANSI[1] != (color.NRGBA{0x02, 0x02, 0x02, 0xff}) {
		t.Fatalf("user ansi not used: %v", m.ANSI)
	}
}

func TestBuildPreviewModelUnknownSchemeFallsBack(t *testing.T) {
	st := state.New("linux")
	st.Values["color_scheme"] = "no-such-scheme"
	m := buildPreviewModel(st, "linux")
	if m.Scheme != "" {
		t.Fatalf("unknown scheme should not be recorded, got %q", m.Scheme)
	}
	if m.Background != defaultPreviewModel().Background {
		t.Fatal("unknown scheme must leave the fallback colours")
	}
}

func TestBuildPreviewModelColorsOverrideScheme(t *testing.T) {
	st := state.New("linux")
	st.Values["color_scheme"] = "Solarized Dark - Patched"
	base := buildPreviewModel(st, "linux")
	st.Values["colors"] = map[string]any{
		"background": "#010203",
		"ansi":       []any{"#0a0a0a"},
	}
	m := buildPreviewModel(st, "linux")
	if m.Background != (color.NRGBA{0x01, 0x02, 0x03, 0xff}) {
		t.Fatalf("colors must override the scheme background, got %v", m.Background)
	}
	if m.ANSI[0] != (color.NRGBA{0x0a, 0x0a, 0x0a, 0xff}) {
		t.Fatalf("colors must override ansi[0], got %v", m.ANSI[0])
	}
	// Untouched entries keep the scheme's values.
	if m.ANSI[1] != base.ANSI[1] || m.Foreground != base.Foreground {
		t.Fatal("colors override must not reset untouched entries")
	}
}

func TestBuildPreviewModelBadTypesIgnored(t *testing.T) {
	d := defaultPreviewModel()
	cases := map[string]map[string]any{
		"colors":      {"colors": "not a palette"},
		"font_size":   {"font_size": "big"},
		"line_height": {"line_height": true},
		"opacity":     {"window_background_opacity": "clear"},
		"tab bar":     {"enable_tab_bar": 1},
		"cursor":      {"default_cursor_style": 7},
		"decorations": {"window_decorations": []any{"TITLE"}},
		"padding":     {"window_padding": "1cell"},
		"font":        {"font": "Comic Sans"},
	}
	for name, vals := range cases {
		st := state.New("linux")
		for k, v := range vals {
			st.Values[k] = v
		}
		if m := buildPreviewModel(st, "linux"); m != d {
			t.Errorf("%s: bad types changed the model:\n got %+v\nwant %+v", name, m, d)
		}
	}
}

func TestBuildPreviewModelPartialPadding(t *testing.T) {
	st := state.New("linux")
	st.Values["window_padding"] = map[string]any{"left": "wat", "right": 2.0, "top": "1cell"}
	m := buildPreviewModel(st, "linux")
	if m.Padding[0] != 1 {
		t.Errorf("unparseable left padding kept %v, want default 1", m.Padding[0])
	}
	if m.Padding[1] != 2 || m.Padding[2] != 1 {
		t.Errorf("valid padding entries not applied: %v", m.Padding)
	}
}

func TestBuildPreviewModelOpacityClamped(t *testing.T) {
	for _, c := range []struct{ in, want float64 }{{2.5, 1}, {-1, 0}, {0.6, 0.6}} {
		st := state.New("linux")
		st.Values["window_background_opacity"] = c.in
		if got := buildPreviewModel(st, "linux").Opacity; got != float32(c.want) {
			t.Errorf("opacity %v -> %v, want %v", c.in, got, c.want)
		}
	}
}

func TestBuildPreviewModelWindowFlags(t *testing.T) {
	st := state.New("linux")
	st.Values["window_decorations"] = "RESIZE|INTEGRATED_BUTTONS"
	st.Values["enable_tab_bar"] = false
	st.Values["tab_bar_at_bottom"] = true
	st.Values["use_fancy_tab_bar"] = false
	st.Values["hide_tab_bar_if_only_one_tab"] = true
	st.Values["default_cursor_style"] = "BlinkingBar"
	st.Values["font"] = map[string]any{"font": []any{map[string]any{"family": "Iosevka", "weight": "500"}}}
	m := buildPreviewModel(st, "linux")
	if m.TitleBar {
		t.Error("TITLE missing from decorations must drop the title bar")
	}
	if m.TabBar || !m.TabBottom || m.FancyTabs || !m.HideOneTab {
		t.Errorf("tab flags wrong: %+v", m)
	}
	if m.showTabs() {
		t.Error("disabled tab bar must not be drawn")
	}
	if m.CursorStyle != "BlinkingBar" {
		t.Errorf("cursor = %q", m.CursorStyle)
	}
	if m.FontFamily != "Iosevka" {
		t.Errorf("font family = %q", m.FontFamily)
	}
	if len(m.tabs()) != 1 {
		t.Errorf("hide_tab_bar_if_only_one_tab should show one tab, got %v", m.tabs())
	}
}

func TestDimCells(t *testing.T) {
	cases := []struct {
		in   any
		want float32
		ok   bool
	}{
		{"2cell", 2, true},
		{"0.5cell", 0.5, true},
		{"12px", 1.5, true}, // 12 / nominalCellPx
		{"12pt", 2, true},   // 16px / 8
		{"50%", 20, true},   // 50% of nominalCols
		{3.0, 3, true},
		{"", 0, false},
		{"1 em", 0, false},
		{true, 0, false},
		{nil, 0, false},
	}
	for _, c := range cases {
		got, ok := dimCells(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("dimCells(%#v) = %v,%v want %v,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestHexColor(t *testing.T) {
	cases := []struct {
		in   string
		want color.NRGBA
		ok   bool
	}{
		{"#abc", color.NRGBA{0xaa, 0xbb, 0xcc, 0xff}, true},
		{"#010203", color.NRGBA{1, 2, 3, 0xff}, true},
		{"#01020380", color.NRGBA{1, 2, 3, 0x80}, true},
		{"#abcd", color.NRGBA{0xaa, 0xbb, 0xcc, 0xdd}, true},
		{"tomato", color.NRGBA{}, false},
		{"#12345", color.NRGBA{}, false},
		{"", color.NRGBA{}, false},
	}
	for _, c := range cases {
		got, ok := hexColor(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("hexColor(%q) = %v,%v want %v,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestPreviewObjectsFitAnySize(t *testing.T) {
	m := defaultPreviewModel()
	if got := previewObjects(m, fyne.NewSize(10, 10)); got != nil {
		t.Errorf("tiny pane drew %d objects, want none", len(got))
	}
	objs := previewObjects(m, fyne.NewSize(320, 480))
	if len(objs) < 20 {
		t.Fatalf("a normal pane drew only %d objects", len(objs))
	}
	for _, o := range objs {
		if o.Position().X < 0 || o.Position().Y < 0 {
			t.Fatalf("object at negative position %v", o.Position())
		}
	}
}

// A state full of junk must still render, and Update must be safe to call.
func TestPreviewPaneUpdateDoesNotPanic(t *testing.T) {
	test.NewTempApp(t)
	st := state.New("linux")
	st.Values["font_size"] = "huge"
	st.Values["window_padding"] = 42
	st.Values["color_scheme"] = "nope"
	p := newPreviewPane(&appState{st: st, target: "linux"})
	w := test.NewWindow(p)
	defer w.Close()
	w.Resize(fyne.NewSize(320, 480))
	p.Update()
}
