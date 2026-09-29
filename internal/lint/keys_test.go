package lint

import (
	"strings"
	"testing"

	"weztermconfigurator/internal/state"
)

func kb(key, mods string) map[string]any {
	m := map[string]any{"key": key, "action": "Nop"}
	if mods != "" {
		m["mods"] = mods
	}
	return m
}

// mouseBinding builds a flat (state) mouse binding.
func mouseBinding(kind string, streak float64, button any, mods, alt any) map[string]any {
	m := map[string]any{
		"event":  map[string]any{"kind": kind, "streak": streak, "button": button},
		"action": "Nop",
	}
	if mods != nil {
		m["mods"] = mods
	}
	if alt != nil {
		m["alt_screen"] = alt
	}
	return m
}

// nestedMouseBinding builds the {Down = {...}} form emitted into wezterm.lua.
func nestedMouseBinding(kind string, streak float64, button any, mods, alt any) map[string]any {
	m := map[string]any{
		"event":  map[string]any{kind: map[string]any{"streak": streak, "button": button}},
		"action": "Nop",
	}
	if mods != nil {
		m["mods"] = mods
	}
	if alt != nil {
		m["alt_screen"] = alt
	}
	return m
}

func stWith(values map[string]any) *state.State {
	return &state.State{Values: values, Raw: map[string]string{}}
}

func TestKeyConflicts(t *testing.T) {
	tests := []struct {
		name string
		st   *state.State
		want []string
	}{
		{
			name: "no bindings",
			st:   stWith(map[string]any{"font_size": 12.0}),
		},
		{
			name: "distinct bindings",
			st: stWith(map[string]any{"keys": []any{
				kb("a", "CTRL"), kb("b", "CTRL"), kb("c", "SHIFT|CTRL"),
			}}),
		},
		{
			name: "duplicate in keys",
			st: stWith(map[string]any{"keys": []any{
				kb("t", "CTRL"), kb("b", "ALT"), kb("t", "CTRL"),
			}}),
			want: []string{"keys: CTRL+t is bound 2 times (entries 1, 3); only the last one wins"},
		},
		{
			name: "mod order and case do not matter",
			st: stWith(map[string]any{"keys": []any{
				kb("t", "shift|ctrl"), kb("T", "CTRL|SHIFT"),
			}}),
			want: []string{"keys: SHIFT|CTRL+t is bound 2 times (entries 1, 2); only the last one wins"},
		},
		{
			name: "NONE equals empty mods",
			st: stWith(map[string]any{"keys": []any{
				kb("q", "NONE"), kb("Q", ""),
			}}),
			want: []string{"keys: q is bound 2 times (entries 1, 2); only the last one wins"},
		},
		{
			name: "LEADER only competes with LEADER",
			st: stWith(map[string]any{"keys": []any{
				kb("a", "CTRL"), kb("b", "LEADER|CTRL"),
				kb("c", "LEADER"), kb("d", "CTRL"), kb("c", "LEADER"),
			}}),
			want: []string{"keys: LEADER+c is bound 2 times (entries 3, 5); only the last one wins"},
		},
		{
			name: "key tables are scoped separately",
			st: stWith(map[string]any{"key_tables": map[string]any{
				"resize_pane": []any{kb("h", "SHIFT"), kb("h", "SHIFT")},
				"move_tab":    []any{kb("h", "SHIFT")},
			}}),
			want: []string{"key_tables.resize_pane: SHIFT+h is bound 2 times (entries 1, 2); only the last one wins"},
		},
		{
			name: "key table order is deterministic",
			st: stWith(map[string]any{"key_tables": map[string]any{
				"b_table": []any{kb("x", "ALT"), kb("x", "ALT")},
				"a_table": []any{kb("y", "ALT"), kb("y", "ALT")},
			}}),
			want: []string{
				"key_tables.a_table: ALT+y is bound 2 times (entries 1, 2); only the last one wins",
				"key_tables.b_table: ALT+x is bound 2 times (entries 1, 2); only the last one wins",
			},
		},
		{
			name: "top level and key table do not collide",
			st: stWith(map[string]any{
				"keys":       []any{kb("h", "SHIFT")},
				"key_tables": map[string]any{"resize_pane": []any{kb("h", "SHIFT")}},
			}),
		},
		{
			name: "mouse duplicates",
			st: stWith(map[string]any{"mouse_bindings": []any{
				mouseBinding("Down", 1, map[string]any{"Left": 1.0}, "CTRL", nil),
				mouseBinding("Down", 1, "Left", "ctrl", nil),
				mouseBinding("Down", 2, map[string]any{"Left": 1.0}, "CTRL", nil),
			}}),
			want: []string{"mouse_bindings: CTRL+down streak=1 left alt_screen=any is bound 2 times (entries 1, 2); only the last one wins"},
		},
		{
			name: "nested and flat mouse forms agree",
			st: stWith(map[string]any{"mouse_bindings": []any{
				nestedMouseBinding("Down", 1, map[string]any{"WheelUp": 1.0}, "CTRL", "false"),
				mouseBinding("Down", 1, "WheelUp", "CTRL", "false"),
			}}),
			want: []string{"mouse_bindings: CTRL+down streak=1 wheelup alt_screen=false is bound 2 times (entries 1, 2); only the last one wins"},
		},
		{
			name: "mods, streak and alt_screen separate mouse triggers",
			st: stWith(map[string]any{"mouse_bindings": []any{
				nestedMouseBinding("Down", 1, map[string]any{"WheelUp": 1.0}, "CTRL", "any"),
				nestedMouseBinding("Down", 1, map[string]any{"WheelUp": 1.0}, "CTRL", "false"),
				nestedMouseBinding("Down", 2, map[string]any{"WheelUp": 1.0}, "CTRL", "any"),
				nestedMouseBinding("Drag", 1, map[string]any{"WheelUp": 1.0}, "CTRL", "any"),
				nestedMouseBinding("Down", 1, map[string]any{"WheelUp": 1.0}, "CTRL|SHIFT", "any"),
			}}),
		},
		{
			name: "leader matches a key",
			st: stWith(map[string]any{
				"keys":   []any{kb("a", "CTRL"), kb("b", "CTRL|SHIFT")},
				"leader": map[string]any{"key": "A", "mods": "CTRL"},
			}),
			want: []string{"leader: CTRL+a matches keys entry 1; that binding never fires on its own"},
		},
		{
			name: "leader mod mismatch is fine",
			st: stWith(map[string]any{
				"keys":   []any{kb("a", "CTRL")},
				"leader": map[string]any{"key": "a", "mods": "CTRL|SHIFT"},
			}),
		},
		{
			name: "malformed entries are ignored",
			st: stWith(map[string]any{
				"keys": []any{
					"not a binding", 42, nil,
					map[string]any{"mods": "CTRL"},
					map[string]any{"key": "  "},
					kb("a", "CTRL"),
				},
				"mouse_bindings": []any{
					map[string]any{"event": "Down"},
					map[string]any{"event": map[string]any{"Down": "nope"}},
					map[string]any{"event": map[string]any{"Down": map[string]any{"streak": 1.0, "button": map[string]any{"Left": 1.0}}, "Up": map[string]any{}}},
					mouseBinding("Down", 1, map[string]any{"Left": 1.0}, nil, nil),
				},
				"leader":     "a",
				"key_tables": map[string]any{"resize_pane": "not a list"},
			}),
		},
		{
			name: "raw lua overrides are opaque",
			st: &state.State{
				Values: map[string]any{"keys": []any{kb("a", "CTRL"), kb("a", "CTRL")}},
				Raw:    map[string]string{"keys": "{ { key = 'a' } }"},
			},
		},
		{
			name: "nil state",
			st:   nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := KeyConflicts(tc.st)
			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Errorf("KeyConflicts =\n%v\nwant:\n%v", got, tc.want)
			}
		})
	}
}
