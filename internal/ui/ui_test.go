package ui

import (
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
