package ui

import (
	"slices"
	"strings"
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

// Fuzzy search must find options by letters in order, rank a real name hit
// above scattered ones, find them by choice values, and stay quiet on junk.
func TestFuzzySearch(t *testing.T) {
	all := make([]*catalog.Option, len(catalog.Options))
	for i := range catalog.Options {
		all[i] = &catalog.Options[i]
	}
	first := func(q string) string {
		got := rankOptions(all, q)
		if len(got) == 0 {
			return ""
		}
		return got[0].Name
	}
	for q, want := range map[string]string{
		"winbgopac":   "window_background_opacity", // abbreviation across words
		"fontsize":    "font_size",
		"font size":   "font_size", // terms may be separated
		"FONT_SIZE":   "font_size", // case-insensitive
		"scrollback":  "scrollback_lines",
		"scrolbak":    "scrollback_lines",      // missing letters
		"mica":        "win32_system_backdrop", // a choice value, not in the name
		"tabbaratbot": "tab_bar_at_bottom",
	} {
		if got := first(q); got != want {
			t.Errorf("query %q: best match %q, want %q", q, got, want)
		}
	}
	// acronyms are ambiguous, so the wanted option only has to be near the top
	top := rankOptions(all, "wbo")
	if len(top) > 8 {
		top = top[:8]
	}
	if !slices.Contains(optNames(top), "window_background_opacity") {
		t.Errorf("wbo: window_background_opacity not in top 8: %v", optNames(top))
	}
	for _, q := range []string{"zzzzq", "qxjv", "fz", "mouseshape zzz"} {
		if got := rankOptions(all, q); len(got) != 0 {
			t.Errorf("junk query %q matched %v", q, optNames(got))
		}
	}
	if got := rankOptions(all, ""); len(got) != 0 {
		t.Errorf("empty query must match nothing, got %d", len(got))
	}
	// a single letter only matches names/choices with a word starting with it
	for _, o := range rankOptions(all, "z") {
		ok := false
		for _, w := range strings.FieldsFunc(o.Name, isSep) {
			ok = ok || strings.HasPrefix(w, "z")
		}
		for _, e := range o.Enum {
			for _, w := range strings.FieldsFunc(strings.ToLower(e), isSep) {
				ok = ok || strings.HasPrefix(w, "z")
			}
		}
		if !ok {
			t.Errorf("single-letter query matched %q without a word starting with z", o.Name)
		}
	}
	// a strong hit hides weak scattered ones
	for _, o := range rankOptions(all, "opac") {
		if o.Name == "mux_output_parser_coalesce_delay_ms" {
			t.Error("opac listed a scattered acronym hit next to real opacity options")
		}
	}
	// exact name outranks longer scattered hits
	got := rankOptions(all, "scrollback_lines")
	if len(got) == 0 || got[0].Name != "scrollback_lines" {
		t.Errorf("exact name must rank first, got %v", optNames(got))
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
