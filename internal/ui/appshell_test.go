package ui

import (
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"weztermconfigurator/internal/catalog"
	"weztermconfigurator/internal/state"
)

func optNames(opts []*catalog.Option) []string {
	out := make([]string, len(opts))
	for i, o := range opts {
		out[i] = o.Name
	}
	return out
}

// A search must find an option by one of its choice values, not only by name
// and doc: "acrylic" appears nowhere in the win32_system_backdrop name or doc.
func TestOptionMatchesQuery(t *testing.T) {
	o := catalog.Find("win32_system_backdrop")
	if o == nil {
		t.Fatal("win32_system_backdrop missing from catalog")
	}
	for _, q := range []string{"acrylic", "ACRYLIC", "win32_system", "backdrop effect"} {
		if !optionMatchesQuery(o, q) {
			t.Errorf("query %q did not match win32_system_backdrop", q)
		}
	}
	if optionMatchesQuery(o, "mouseshape") {
		t.Error("unrelated query matched win32_system_backdrop")
	}
}

func TestSearchResultsHonourFilters(t *testing.T) {
	a := &appState{st: state.New("linux"), target: "linux"}

	if got := a.searchResults("acrylic"); len(got) != 0 {
		t.Fatalf("Windows-only option leaked into a Linux search: %v", optNames(got))
	}
	if !a.matchesSomePlatform("acrylic") {
		t.Fatal("matchesSomePlatform should ignore the platform filter")
	}

	a.showAll = true
	if len(a.searchResults("acrylic")) == 0 {
		t.Fatal("expected a hit once All platforms is on")
	}

	a.changedOnly = true
	if got := a.searchResults("acrylic"); len(got) != 0 {
		t.Fatalf("unset option survived “Changed only”: %v", optNames(got))
	}
	a.st.Values["win32_system_backdrop"] = "Mica"
	if got := a.searchResults("acrylic"); len(got) != 1 {
		t.Fatalf("expected only win32_system_backdrop once set, got %v", optNames(got))
	}
}

func TestQuickOptionNamesPinsFirstWithoutDuplicates(t *testing.T) {
	a := &appState{st: state.New("linux")}
	a.st.Pinned = []string{"win32_system_backdrop", "scrollback_lines", "not_an_option"}

	names := a.quickOptionNames()
	pins := []string{"win32_system_backdrop", "scrollback_lines"}
	if len(names) < len(pins) || names[0] != pins[0] || names[1] != pins[1] {
		t.Fatalf("pins must lead in pinning order, got %v", names)
	}
	// scrollback_lines is a default quick option too, so the tail must not repeat it.
	if len(names) != len(catalog.QuickOptionNames)+1 {
		t.Fatalf("expected the pins plus every default, got %d names", len(names))
	}
	seen := map[string]bool{}
	for _, n := range names {
		if seen[n] {
			t.Fatalf("duplicate %q in quick settings", n)
		}
		seen[n] = true
		if catalog.Find(n) == nil {
			t.Fatalf("unknown option %q shown in quick settings", n)
		}
	}
}

func TestValidCategory(t *testing.T) {
	if !validCategory(catalog.QuickCategory) || !validCategory(catalog.CustomLuaCategory) ||
		!validCategory(catalog.PluginsCategory) || !validCategory(catalog.FeaturesCategory) ||
		!validCategory(catalog.Categories[0]) {
		t.Fatal("real nav entries must validate")
	}
	if !validCategory("Fonts") {
		t.Fatal("Fonts must validate")
	}
	if validCategory("") || validCategory("Removed Category") {
		t.Fatal("unknown category must not validate")
	}
}

// The Preview button and the remembered preference must agree with the column's
// visibility, otherwise a restart shows a column the user switched off.
func TestSetPreviewTogglesColumn(t *testing.T) {
	testApp := test.NewTempApp(t)
	a := &appState{app: testApp, st: state.New("linux"), previewOn: true}
	a.previewCol = container.NewHBox()

	a.setPreview(false)
	if a.previewOn || a.previewCol.Visible() {
		t.Fatal("setPreview(false) left the preview on")
	}
	if got := a.app.Preferences().Bool(prefPreview); got {
		t.Fatal("preview preference not persisted as off")
	}
	a.setPreview(true)
	if !a.previewOn || !a.previewCol.Visible() {
		t.Fatal("setPreview(true) did not show the preview column")
	}
}
