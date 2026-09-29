package luaimport

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"weztermconfigurator/internal/luagen"
	"weztermconfigurator/internal/state"
)

// goldenState mirrors the state behind internal/luagen/testdata/golden.lua so an
// emitted config can be read back in and compared value by value.
func goldenState() *state.State {
	s := state.New("linux")
	s.Values["font_size"] = 13.5
	s.Values["scrollback_lines"] = 5000.0
	s.Values["window_decorations"] = "TITLE|RESIZE|INTEGRATED_BUTTONS"
	s.Values["freetype_interpreter_version"] = "40"
	s.Values["enable_scroll_bar"] = true
	s.Values["color_scheme"] = "Batman"
	s.Values["window_background_opacity"] = 0.85
	s.Values["window_background_image"] = "/tmp/bg.png"
	s.Values["default_cwd"] = "/home/wadhah"
	s.Values["min_scroll_bar_height"] = "0.5cell"
	s.Values["default_gui_startup_args"] = []any{"start"}
	s.Values["clean_exit_codes"] = []any{0.0, 130.0}
	s.Values["dpi_by_screen"] = map[string]any{"DP-1": 96.0}
	s.Values["set_environment_variables"] = map[string]any{"TERM_PROGRAM": "wezterm"}
	s.Values["selection_word_boundary"] = " \t\n{[}]()\"'`"
	s.Values["font"] = map[string]any{
		"font": []any{
			map[string]any{"family": "JetBrains Mono", "weight": "Bold"},
			map[string]any{"family": "Noto Color Emoji"},
		},
	}
	s.Values["font_rules"] = []any{
		map[string]any{
			"intensity": "Bold",
			"font":      map[string]any{"font": []any{map[string]any{"family": "Iosevka", "weight": "500"}}, "foreground": "tomato"},
		},
	}
	s.Values["colors"] = map[string]any{
		"ansi":                  []any{"#000000", "#d70000", "#5faf00", "#d7af00", "#0087d7", "#af00af", "#00afaf", "#bcbcbc"},
		"indexed":               map[string]any{"16": "#af8700"},
		"tab_bar":               map[string]any{"active_tab": map[string]any{"bg_color": "#2b2042", "fg_color": "#c0c0c0"}},
		"quick_select_label_fg": map[string]any{"AnsiColor": "Black"},
	}
	s.Values["color_schemes"] = map[string]any{
		"My Scheme": map[string]any{"foreground": "#fff"},
	}
	s.Values["keys"] = []any{
		map[string]any{"key": "t", "mods": "CTRL|SHIFT", "action": map[string]any{"SpawnTab": "CurrentPaneDomain"}},
		map[string]any{"key": "h", "mods": "LEADER", "action": map[string]any{"AdjustPaneSize": []any{"Left", 5.0}}},
		map[string]any{"key": "F11", "action": "ToggleFullScreen"},
		map[string]any{"key": "x", "mods": "CTRL", "action": map[string]any{"Multiple": []any{"ScrollToBottom", map[string]any{"CopyMode": "Close"}}}},
		map[string]any{"key": "p", "mods": "CTRL", "action": map[string]any{"SplitPane": map[string]any{"direction": "Right", "size": map[string]any{"Percent": 30.0}}}},
		map[string]any{"key": "i", "mods": "CTRL", "action": map[string]any{"Lua": "wezterm.action_callback(function(window, pane) end)"}},
	}
	s.Values["key_tables"] = map[string]any{
		"resize_pane": []any{map[string]any{"key": "LeftArrow", "action": map[string]any{"AdjustPaneSize": []any{"Left", 1.0}}}},
	}
	s.Values["leader"] = map[string]any{"key": "a", "mods": "CTRL", "timeout_milliseconds": 1000.0}
	s.Values["mouse_bindings"] = []any{
		map[string]any{
			"event":      map[string]any{"Down": map[string]any{"streak": 1.0, "button": map[string]any{"WheelUp": 1.0}}},
			"mods":       "CTRL",
			"alt_screen": "false",
			"action":     "IncreaseFontSize",
		},
	}
	s.Values["background"] = []any{
		map[string]any{
			"source":     map[string]any{"Gradient": map[string]any{"orientation": map[string]any{"Linear": []any{-45.0}}, "preset": "Warm"}},
			"attachment": map[string]any{"Parallax": 0.1},
			"opacity":    0.9,
		},
	}
	s.Values["cursor_blink_ease_in"] = map[string]any{"CubicBezier": []any{0.0, 0.0, 0.58, 1.0}}
	s.Values["exec_domains"] = []any{
		map[string]any{"name": "docker", "fixup": "function(cmd)\n  return cmd\nend", "label": "Docker"},
	}
	s.Raw["hyperlink_rules"] = "wezterm.default_hyperlink_rules()"
	return s
}

// TestRoundTripGolden is the sharp end: every option the emitter can write must
// survive a trip back through the importer with the same value.
func TestRoundTripGolden(t *testing.T) {
	src, err := luagen.Emit(goldenState())
	if err != nil {
		t.Fatal(err)
	}
	res, err := Import(src, "linux")
	if err != nil {
		t.Fatal(err)
	}
	want := goldenState()

	// exec_domains and hyperlink_rules cannot come back: one holds a Lua
	// function, the other is a wezterm call. Everything else must.
	wantNames := make([]string, 0, len(want.Values))
	for name := range want.Values {
		if name == "exec_domains" {
			continue
		}
		wantNames = append(wantNames, name)
	}
	slices.Sort(wantNames)
	if !slices.Equal(res.Imported, wantNames) {
		t.Errorf("imported options:\n got %v\nwant %v", res.Imported, wantNames)
	}

	wantSkips := []string{"config.keys[6].action", "exec_domains", "hyperlink_rules"}
	var gotSkips []string
	for _, s := range res.Skipped {
		gotSkips = append(gotSkips, s.Name)
	}
	if !slices.Equal(gotSkips, wantSkips) {
		t.Errorf("skipped entries:\n got %+v\nwant %v", res.Skipped, wantSkips)
	}

	for _, name := range wantNames {
		got := res.State.Values[name]
		if name == "keys" {
			// The action_callback binding has no value to import; the other
			// five must come back untouched.
			list, ok := got.([]any)
			if !ok || len(list) != 5 {
				t.Errorf("keys = %#v, want the five importable bindings", got)
				continue
			}
			for i, e := range want.Values[name].([]any)[:5] {
				if !reflect.DeepEqual(list[i], e) {
					t.Errorf("keys[%d]:\n got %#v\nwant %#v", i, list[i], e)
				}
			}
			continue
		}
		if !reflect.DeepEqual(got, want.Values[name]) {
			t.Errorf("%s:\n got %#v\nwant %#v", name, got, want.Values[name])
		}
	}
}

func TestRealisticConfig(t *testing.T) {
	src := `
local wezterm = require 'wezterm'
local act = wezterm.action
local config = wezterm.config_builder()
config:set_strict_mode(false)

local pad = 8
local function padder(n) return n end

config.color_scheme = 'Catppuccin Mocha'
config.font = wezterm.font_with_fallback({
  { family = 'Berkeley Mono', weight = 'Regular' },
  { family = 'Noto Color Emoji' },
})
config.font_size = 12.0
config.window_padding = { left = pad, right = pad, top = 4, bottom = 4 }
config.window_background_opacity = 0.96
config.window_decorations = 'INTEGRATED_BUTTONS|RESIZE'
config.term = 'wezterm-256color'
config.check_for_updates = false
config.enable_tab_bar = true
config.scrollback_lines = 10000
config.leader = { key = 'a', mods = 'CTRL', timeout_milliseconds = 1500 }
config.keys = {
  { key = 't', mods = 'CTRL', action = act.SpawnTab('CurrentPaneDomain') },
  { key = 'h', mods = 'LEADER', action = act.ActivateTabRelative(1) },
  { key = 'z', mods = 'CTRL', action = wezterm.action_callback(function(window, pane) end) },
}
config.mouse_bindings = {
  { event = { Up = { streak = 1, button = { WheelUp = 1 } } }, mods = 'SHIFT', action = act.ScrollByLine(3) },
}
wezterm.on('format-tab-title', function(tab, tabs, cfg) return 'x' end)
wezterm.on('update-status', function(window, pane) end)
config.not_a_real_option = 1
config.scrollback_lines = padder(10000)
return config
`
	res, err := Import(src, "linux")
	if err != nil {
		t.Fatal(err)
	}
	v := res.State.Values
	for name, want := range map[string]any{
		"color_scheme":              "Catppuccin Mocha",
		"font_size":                 12.0,
		"window_background_opacity": 0.96,
		"window_decorations":        "INTEGRATED_BUTTONS|RESIZE",
		"term":                      "wezterm-256color",
		"check_for_updates":         false,
		"enable_tab_bar":            true,
		"scrollback_lines":          10000.0,
		"leader":                    map[string]any{"key": "a", "mods": "CTRL", "timeout_milliseconds": 1500.0},
		"window_padding":            map[string]any{"left": "8", "right": "8", "top": "4", "bottom": "4"},
		"font": map[string]any{"font": []any{
			map[string]any{"family": "Berkeley Mono", "weight": "Regular"},
			map[string]any{"family": "Noto Color Emoji"},
		}},
	} {
		if !reflect.DeepEqual(v[name], want) {
			t.Errorf("%s:\n got %#v\nwant %#v", name, v[name], want)
		}
	}
	keys, ok := v["keys"].([]any)
	if !ok || len(keys) != 2 {
		t.Fatalf("keys = %#v, want the two importable bindings (skipped: %+v)", v["keys"], res.Skipped)
	}
	if want := map[string]any{"key": "h", "mods": "LEADER", "action": map[string]any{"ActivateTabRelative": 1.0}}; !reflect.DeepEqual(keys[1], want) {
		t.Errorf("keys[1]:\n got %#v\nwant %#v", keys[1], want)
	}
	mouse, ok := v["mouse_bindings"].([]any)
	if !ok || len(mouse) != 1 {
		t.Fatalf("mouse_bindings = %#v", v["mouse_bindings"])
	}
	want := map[string]any{
		"event":  map[string]any{"Up": map[string]any{"streak": 1.0, "button": map[string]any{"WheelUp": 1.0}}},
		"mods":   "SHIFT",
		"action": map[string]any{"ScrollByLine": 3.0},
	}
	if !reflect.DeepEqual(mouse[0], want) {
		t.Errorf("mouse_bindings[0]:\n got %#v\nwant %#v", mouse[0], want)
	}

	var names []string
	for _, s := range res.Skipped {
		names = append(names, s.Name)
	}
	if want := []string{"config.keys[3].action", "not_a_real_option"}; !slices.Equal(names, want) {
		t.Errorf("skipped:\n got %v\nwant %v", names, want)
	}
	if len(res.Notes) != 1 || !strings.Contains(res.Notes[0], "2 wezterm.on() event handlers") {
		t.Errorf("notes = %v, want one note about both event handlers", res.Notes)
	}
}

func TestImportErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"syntax error", "local config = { \nreturn config\n", []string{"could not read wezterm.lua", "wezterm.lua line:"}},
		{"no return", "local wezterm = require 'wezterm'\nlocal config = wezterm.config_builder()\n", []string{"must end with"}},
		{"number return", "return 42\n", []string{"number, not a configuration table"}},
		{"function return", "return function() end\n", []string{"function, not a configuration table"}},
		{"runtime error", "return wezterm.config_builder().missing.deep\n", []string{"could not run wezterm.lua"}},
		{"empty", "", []string{"must end with"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Import(tc.src, "linux")
			if err == nil {
				t.Fatal("want an error")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not mention %q", err, w)
				}
			}
		})
	}
}

func TestTimeout(t *testing.T) {
	old := importTimeout
	importTimeout = 200 * time.Millisecond
	t.Cleanup(func() { importTimeout = old })

	start := time.Now()
	_, err := Import("while true do end\nreturn {}\n", "linux")
	if err == nil {
		t.Fatal("want a timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q, want it to mention the timeout", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Errorf("took %s, the timeout did not cut it short", time.Since(start))
	}
}

func TestSourceTooLarge(t *testing.T) {
	_, err := Import(strings.Repeat("x", maxSrcBytes+1), "linux")
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Errorf("error = %v, want a size limit error", err)
	}
}

func TestNoFileAccess(t *testing.T) {
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("untouched"), 0o600); err != nil {
		t.Fatal(err)
	}
	q := "'" + sentinel + "'"
	tests := []string{
		"return { f = io.open(" + q + ", 'w') }",
		"return { f = io.lines(" + q + ") }",
		"return { f = os.execute('touch " + filepath.Join(dir, "pwned") + "') }",
		"return { f = os.remove(" + q + ") }",
		"return { f = os.rename(" + q + ", " + q + ") }",
		"return { f = dofile(" + q + ") }",
		"return { f = loadfile(" + q + ") }",
		"return { f = load('return 1')() }",
		"return { f = require 'wezterm.gui' }",
		"return { f = require 'lfs' }",
	}
	for _, body := range tests {
		src := "local config = wezterm.config_builder()\n" + body + "\nreturn config\n"
		if _, err := Import(src, "linux"); err == nil {
			t.Errorf("%s ran without error, so it reached something it must not", body)
		}
	}
	if b, err := os.ReadFile(sentinel); err != nil || string(b) != "untouched" {
		t.Errorf("sentinel = %q, %v", b, err)
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 1 {
		t.Errorf("the sandbox wrote to the working directory: %v", ents)
	}
}

func TestSandboxHasNoGlobals(t *testing.T) {
	for _, name := range []string{"io", "debug", "dofile", "load", "loadfile", "collectgarbage"} {
		src := "if " + name + " ~= nil then error('" + name + " is reachable') end\nreturn {}\n"
		if _, err := Import(src, "linux"); err != nil {
			t.Errorf("%s should be gone but the import failed: %v", name, err)
		}
	}
	// require must answer from preload only.
	if _, err := Import("local m = require 'wezterm'\nreturn { f = m.config_builder() }\n", "linux"); err != nil {
		t.Errorf("require 'wezterm' must work: %v", err)
	}
}

func TestNodeBudget(t *testing.T) {
	// A small file that expands into far more values than an import may walk.
	src := `local schemes = {}
for i = 1, 60000 do schemes['s'..i] = { foreground = '#fff' } end
return { color_scheme = 'Batman', color_schemes = schemes }
`
	res, err := Import(src, "linux")
	if err != nil {
		t.Fatal(err)
	}
	// The budget is spent on whichever option the walk reaches first, so what
	// matters is that the runaway table is reported rather than half-imported.
	if len(res.Imported)+len(res.Skipped) != 2 {
		t.Errorf("imported %v and skipped %v, want every option accounted for", res.Imported, res.Skipped)
	}
	if !slices.ContainsFunc(res.Skipped, func(s Skip) bool {
		return s.Name == "color_schemes" && strings.Contains(s.Reason, "too many")
	}) {
		t.Errorf("skipped = %v, want color_schemes reported as too large", res.Skipped)
	}
}

func TestStrftime(t *testing.T) {
	at := time.Date(2024, 3, 7, 9, 5, 3, 0, time.UTC)
	tests := []struct{ in, want string }{
		{"%Y-%m-%d", "2024-03-07"},
		{"%H:%M:%S", "09:05:03"},
		{"%d%%", "07%"},
		{"100% sure", "100% sure"},
		{"%Q", "%Q"},
	}
	for _, tc := range tests {
		if got := strftime(tc.in, at); got != tc.want {
			t.Errorf("strftime(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
