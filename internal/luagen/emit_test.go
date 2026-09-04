package luagen

import (
	"os"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
	"weztermconfigurator/internal/catalog"
	"weztermconfigurator/internal/state"
)

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
	s.CustomLua = "wezterm.on('update-status', function(window, pane) end)"
	return s
}

func TestGolden(t *testing.T) {
	got, err := Emit(goldenState())
	if err != nil {
		t.Fatal(err)
	}
	const path = "testdata/golden.lua"
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestEmitExecutesAsLua(t *testing.T) {
	golden, err := Emit(goldenState())
	if err != nil {
		t.Fatal(err)
	}
	L := lua.NewState()
	defer L.Close()
	L.PreloadModule("wezterm", weztermStub)
	script := `
local config = (function()
` + golden + `
end)()
assert(config.font_size == 13.5, "font_size")
assert(config.window_decorations == 'TITLE|RESIZE|INTEGRATED_BUTTONS', "decorations")
assert(config.keys[1].action.SpawnTab == 'CurrentPaneDomain', "SpawnTab")
assert(config.scrollback_lines == 5000, "scrollback")
`
	if err := L.DoString(script); err != nil {
		t.Fatalf("generated Lua failed: %v", err)
	}
}

// weztermStub preloads a minimal wezterm module for the executable test.
func weztermStub(L *lua.LState) int {
	mod := L.NewTable()
	L.SetField(mod, "config_builder", L.NewFunction(func(L *lua.LState) int {
		L.Push(L.NewTable())
		return 1
	}))
	L.SetField(mod, "font_with_fallback", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		L.SetField(t, "font", L.Get(1))
		L.Push(t)
		return 1
	}))
	L.SetField(mod, "action_callback", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("callback"))
		return 1
	}))
	L.SetField(mod, "on", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNil)
		return 1
	}))
	L.SetField(mod, "default_hyperlink_rules", L.NewFunction(func(L *lua.LState) int {
		L.Push(L.NewTable())
		return 1
	}))
	L.SetField(mod, "exec_domain", L.NewFunction(func(L *lua.LState) int {
		t := L.NewTable()
		L.SetField(t, "name", L.Get(1))
		L.SetField(t, "fixup", L.Get(2))
		if L.GetTop() >= 3 {
			L.SetField(t, "label", L.Get(3))
		}
		L.Push(t)
		return 1
	}))

	actionMeta := L.NewTable()
	L.SetField(actionMeta, "__index", L.NewFunction(func(L *lua.LState) int {
		name, _ := L.Get(2).(lua.LString)
		callable := L.NewTable()
		callMeta := L.NewTable()
		L.SetField(callMeta, "__call", L.NewFunction(func(L *lua.LState) int {
			result := L.NewTable()
			if L.GetTop() >= 2 {
				L.SetField(result, string(name), L.Get(2))
			}
			L.Push(result)
			return 1
		}))
		L.SetMetatable(callable, callMeta)
		L.Push(callable)
		return 1
	}))
	action := L.NewTable()
	L.SetMetatable(action, actionMeta)
	L.SetField(mod, "action", action)

	L.Push(mod)
	return 1
}

func TestRequiredFieldError(t *testing.T) {
	s := state.New("linux")
	s.Values["window_content_alignment"] = map[string]any{"horizontal": "Left"}
	_, err := Emit(s)
	if err == nil || !strings.Contains(err.Error(), "vertical is required") {
		t.Fatalf("expected required-field error naming vertical, got: %v", err)
	}
}

func TestRenderValuePrefill(t *testing.T) {
	o := catalog.Find("font_size")
	got, err := RenderValue(13.5, &o.Field, 0)
	if err != nil || got != "13.5" {
		t.Fatalf("RenderValue = %q err=%v", got, err)
	}
}

func TestEmitPlugins(t *testing.T) {
	s := state.New("linux")
	s.Values["font_size"] = 13.5
	s.Plugins = []state.Plugin{
		{URL: "https://github.com/mrjones2014/smart-splits.nvim", Var: "plugin_smartsplitsnvim", Apply: true,
			Opts: "{ direction_keys = { 'h', 'j', 'k', 'l' } }"},
		{URL: "https://github.com/adriankarlen/bar.wezterm", Var: "plugin_bar", Apply: false},
		{URL: "bad url", Var: "has space"}, // invalid Var+URL → skipped silently
	}
	out, err := Emit(s)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"\n-- Plugins\n",
		"local plugin_smartsplitsnvim = wezterm.plugin.require 'https://github.com/mrjones2014/smart-splits.nvim'\n",
		"plugin_smartsplitsnvim.apply_to_config(config, { direction_keys = { 'h', 'j', 'k', 'l' } })\n",
		"local plugin_bar = wezterm.plugin.require 'https://github.com/adriankarlen/bar.wezterm'\n",
		"return config\n",
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Fatalf("emitted config missing %q:\n%s", w, out)
		}
	}
	if strings.Contains(out, "has space") {
		t.Fatalf("invalid plugin var leaked into output:\n%s", out)
	}
	// Plugins emitted after options, before custom lua.
	if strings.Index(out, "plugin_smartsplitsnvim") < strings.Index(out, "font_size") {
		t.Fatalf("plugins must come after option assignments:\n%s", out)
	}
}

func TestEmitFeatures(t *testing.T) {
	s := state.New("linux")
	s.Values["font_size"] = 13.5
	s.Features = map[string]map[string]string{
		"opacity_toggle":     {"__on": "1", "levels": "1.0,0.75,0.5", "key": "CTRL|SHIFT|O"},
		"quick_select_pack":  {"__on": "1", "git_sha": "1", "ipv4": "1"},
		"gpu_adapter_select": {"__on": "1", "backend": "Vulkan"},
	}
	out, err := Emit(s)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"\n-- Features\n",
		"-- Opacity runtime toggle (",
		"local opacity_levels = { 1.0, 0.75, 0.5, }",
		"wezterm.action.EmitEvent('toggle-opacity')",
		"config.quick_select_patterns",
		"'[0-9a-f]{7,40}'",
		"config.front_end = 'WebGpu'",
		"wezterm.gui.enumerate_gpus()",
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Fatalf("missing %q in:\n%s", w, out)
		}
	}
	// Features come after options; custom lua still last.
	if strings.Index(out, "-- Features") < strings.Index(out, "font_size") {
		t.Fatalf("features must come after options")
	}
	s.CustomLua = "wezterm.on('x', function() end)"
	out2, _ := Emit(s)
	if strings.Index(out2, "-- Features") > strings.Index(out2, "-- Custom Lua") {
		t.Fatalf("features must precede custom lua")
	}
	// Disabled feature emits nothing.
	s2 := state.New("linux")
	s2.Features = map[string]map[string]string{"theme_rotator": {}}
	out3, _ := Emit(s2)
	if strings.Contains(out3, "Theme rotator") {
		t.Fatalf("disabled feature leaked:\n%s", out3)
	}
}
