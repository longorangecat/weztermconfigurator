package ui

import (
	"fyne.io/fyne/v2/test"
	"strings"
	"testing"

	"weztermconfigurator/internal/catalog"
	"weztermconfigurator/internal/luagen"
	"weztermconfigurator/internal/state"
)

func TestValidateAll(t *testing.T) {
	st := state.New("linux")
	app := &appState{st: st}

	// Clean state should validate cleanly
	if err := app.validateAll(); err != nil {
		t.Fatalf("expected nil err for clean state, got %v", err)
	}

	// State with conflicting feature groups should fail validation
	st.Features["tab_title_powerline"] = map[string]string{"__on": "1"}
	st.Features["tab_cwd_title"] = map[string]string{"__on": "1"}
	if err := app.validateAll(); err == nil {
		t.Fatalf("expected validation error on feature group conflict, got nil")
	} else if !strings.Contains(err.Error(), "both set the tab title") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestCatalogActionLookup(t *testing.T) {
	// Verify action lookup works for known actions
	act := catalog.FindAction("SpawnTab")
	if act == nil {
		t.Fatalf("FindAction('SpawnTab') returned nil")
	}
	if act.Arg == nil {
		t.Fatalf("expected SpawnTab to have an Arg definition")
	}
}

func TestExportAndOpenRoundTrip(t *testing.T) {
	st := state.New("linux")
	st.Values["font_size"] = 14.5
	st.CustomLua = "-- testing export/open\n"

	// 1. Emit Lua with embedded state
	out, err := luagen.Emit(st)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	// 2. Extract state as openFile would from Lua
	extracted, err := state.Extract(out)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}
	if extracted.Values["font_size"] != 14.5 || extracted.CustomLua != st.CustomLua {
		t.Fatalf("extracted state mismatch: %+v", extracted)
	}
}

func TestQuickCategoryAndOptionChanges(t *testing.T) {
	if len(catalog.QuickOptionNames) == 0 {
		t.Fatal("expected QuickOptionNames to have entries")
	}
	for _, name := range catalog.QuickOptionNames {
		if catalog.Find(name) == nil {
			t.Fatalf("quick option %s not found in catalog", name)
		}
	}

	a := &appState{
		st:     state.New("linux"),
		target: "linux",
	}

	tabOpt := catalog.Find("enable_tab_bar")
	if tabOpt == nil {
		t.Fatal("enable_tab_bar not found")
	}
	// Default is true, not set in state => not changed
	if a.isOptionChangedFromDefault(tabOpt) {
		t.Fatal("expected unchanged when not set")
	}

	// Set to false (different from default true) => changed
	a.st.Values["enable_tab_bar"] = false
	if !a.isOptionChangedFromDefault(tabOpt) {
		t.Fatal("expected changed when set to false")
	}

	// Set to true (same as default true) => not changed
	a.st.Values["enable_tab_bar"] = true
	if a.isOptionChangedFromDefault(tabOpt) {
		t.Fatal("expected unchanged when explicitly set to default true")
	}
}

func TestOptionTooltipFormatting(t *testing.T) {
	opt := catalog.Find("color_scheme")
	if opt == nil {
		t.Fatal("color_scheme not found")
	}
	tip := formatOptionTooltip(opt, "linux")
	if !strings.Contains(tip, "color_scheme") {
		t.Fatalf("expected tooltip to contain option name: %s", tip)
	}
	if !strings.Contains(tip, opt.Doc) {
		t.Fatalf("expected tooltip to contain doc: %s", tip)
	}

	enumOpt := catalog.Find("window_decorations")
	if enumOpt != nil && len(enumOpt.Enum) > 0 {
		enumTip := formatOptionTooltip(enumOpt, "linux")
		if !strings.Contains(enumTip, "Available options / values:") {
			t.Fatalf("expected enum options in tooltip: %s", enumTip)
		}
		if !strings.Contains(enumTip, enumOpt.Enum[0]) {
			t.Fatalf("expected enum value %s in tooltip: %s", enumOpt.Enum[0], enumTip)
		}
	}
}

// Rendering an editor must not write the shown default into state.
func TestEditorsDoNotFireOnRender(t *testing.T) {
	test.NewTempApp(t)
	a := &appState{st: state.New("linux")}
	for _, f := range []*catalog.Field{
		{Name: "b", Kind: catalog.Bool, Default: true},
		{Name: "e", Kind: catalog.Enum, Enum: []string{"A", "B"}, Default: "A"},
		{Name: "fl", Kind: catalog.Flags, Enum: []string{"X", "Y"}},
	} {
		fired := false
		get := func() any {
			switch f.Kind {
			case catalog.Bool:
				return true
			case catalog.Enum:
				return "B"
			}
			return "X|Y"
		}
		a.buildEditor(f, get, func(any) { fired = true })
		if fired {
			t.Errorf("%s: editor wrote state while rendering", f.Name)
		}
	}
}
