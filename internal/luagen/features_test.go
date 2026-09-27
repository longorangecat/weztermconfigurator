package luagen

import (
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"

	"weztermconfigurator/internal/catalog"
	"weztermconfigurator/internal/state"
)

// TestFeaturesCompileAndRun tests that each catalog feature emits valid Lua
// both with default params and with custom non-empty params, and runs cleanly
// through gopher-lua with simulated WezTerm globals.
func TestFeaturesCompileAndRun(t *testing.T) {
	for i := range catalog.Features {
		f := &catalog.Features[i]
		t.Run(f.ID+"/defaults", func(t *testing.T) {
			params := make(map[string]string)
			for _, p := range f.Params {
				if p.Default != "" {
					params[p.Name] = p.Default
				}
			}
			params["__on"] = "1"
			checkFeatureLua(t, f, params)
		})

		t.Run(f.ID+"/custom", func(t *testing.T) {
			params := make(map[string]string)
			for _, p := range f.Params {
				switch p.Kind {
				case "bool":
					if p.Default == "1" {
						params[p.Name] = ""
					} else {
						params[p.Name] = "1"
					}
				case "select":
					if len(p.Options) > 1 {
						params[p.Name] = p.Options[1]
					} else if len(p.Options) > 0 {
						params[p.Name] = p.Options[0]
					}
				default:
					params[p.Name] = p.Default + "_custom"
				}
			}
			// Special handling for backdrop_cycler custom dir to ensure emit is not a comment
			if f.ID == "backdrop_cycler" {
				params["dir"] = "/tmp/images/"
			}
			params["__on"] = "1"
			checkFeatureLua(t, f, params)
		})
	}
}

func checkFeatureLua(t *testing.T, f *catalog.Feature, params map[string]string) {
	t.Helper()
	code := f.Emit(params)
	if strings.TrimSpace(code) == "" || strings.HasPrefix(strings.TrimSpace(code), "--") {
		// Valid skip/comment (e.g. backdrop_cycler when dir is empty)
		return
	}

	L := lua.NewState()
	defer L.Close()

	// Inject wezterm mock harness
	harness := `
local wezterm = {
  events = {},
  action = {
    EmitEvent = function(name) return { type = 'EmitEvent', name = name } end,
  },
  color = {
    get_builtin_schemes = function() return { ["Default"] = {}, ["Nord"] = {} } end,
  },
  gui = {
    enumerate_gpus = function() return { { name = "Mock GPU" } } end,
  },
  mux = {
    set_active_workspace = function(ws) end,
    spawn_window = function(opt) return {}, {}, {} end,
  },
  enumerate_ssh_hosts = function() return { ["host1"] = {}, ["host2"] = {} } end,
  battery_info = function() return { { state = 'Charging', state_of_charge = 0.85 } } end,
  strftime = function(fmt) return "2026-09-27" end,
  truncate_right = function(s, max) return s:sub(1, max) end,
  column_width = function(s) return #s end,
  format = function(cells) return "formatted" end,
  glob = function(pattern) return { "img1.jpg", "img2.png" } end,
}
function wezterm.on(event, fn)
  wezterm.events[event] = fn
end

local config = { keys = {} }
local mux = wezterm.mux
_G.wezterm = wezterm
_G.config = config
_G.mux = mux
`
	if err := L.DoString(harness); err != nil {
		t.Fatalf("harness init failed: %v", err)
	}

	if err := L.DoString(code); err != nil {
		t.Fatalf("feature %s emitted invalid Lua: %v\nCode:\n%s", f.ID, err, code)
	}
}

func TestFeatureGroupConflict(t *testing.T) {
	st := state.New("")
	st.Features = map[string]map[string]string{
		"tab_title_powerline": {"__on": "1"},
		"tab_cwd_title":       {"__on": "1"},
	}

	_, err := Emit(st)
	if err == nil {
		t.Fatalf("expected conflict error when two features in 'tab title' group are on, got nil")
	}
	if !strings.Contains(err.Error(), "both set the tab title; turn one off") {
		t.Errorf("unexpected error message: %v", err)
	}
}
