package catalog

import (
	"bufio"
	"os"
	"testing"
)

func expectedNames(t *testing.T) []string {
	t.Helper()
	f, err := os.Open("testdata/config_fields.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var names []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if s := sc.Text(); s != "" {
			names = append(names, s)
		}
	}
	return names
}

func TestCatalogMatchesOptionMatrix(t *testing.T) {
	want := expectedNames(t)
	if len(Options) != len(want) {
		t.Fatalf("Options has %d entries, matrix has %d", len(Options), len(want))
	}
	seen := map[string]bool{}
	for i, o := range Options {
		if o.Name != want[i] {
			t.Fatalf("option %d: got %q, want %q (order must match the matrix)", i, o.Name, want[i])
		}
		if o.Doc == "" {
			t.Fatalf("option %q: missing Doc", o.Name)
		}
		if seen[o.Name] {
			t.Fatalf("duplicate option %q", o.Name)
		}
		seen[o.Name] = true
		validCat := o.Category == CustomLuaCategory
		for _, c := range Categories {
			if o.Category == c {
				validCat = true
			}
		}
		if !validCat {
			t.Fatalf("option %q: unknown category %q", o.Name, o.Category)
		}
		switch o.Kind {
		case Enum, Union:
			if len(o.Enum) < 2 {
				t.Fatalf("option %q: Enum/Union needs >=2 variants", o.Name)
			}
		case Struct, List:
			if len(o.Fields) == 0 && o.ItemKind == 0 {
				t.Fatalf("option %q: Struct/List needs Fields or ItemKind", o.Name)
			}
		case Flags:
			if o.EmptyToken == "" {
				t.Fatalf("option %q: Flags needs EmptyToken", o.Name)
			}
		}
	}
}

func TestSchemesCatalog(t *testing.T) {
	ss := Schemes()
	if len(ss) == 0 {
		t.Fatal("no schemes loaded")
	}
	seen := map[string]bool{}
	var hasBatman bool
	for _, s := range ss {
		if s.Name == "" {
			t.Fatal("scheme with empty name")
		}
		if seen[s.Name] {
			t.Fatalf("duplicate scheme name: %q", s.Name)
		}
		seen[s.Name] = true
		if len(s.Colors) == 0 {
			t.Fatalf("scheme %q has no colors", s.Name)
		}
		if s.Name == "Batman" {
			hasBatman = true
		}
	}
	if !hasBatman {
		t.Fatal("Batman scheme missing")
	}
	if Find("color_scheme") == nil {
		t.Fatal("color_scheme option missing")
	}
}

func TestActionsCatalog(t *testing.T) {
	if len(Actions) != 84 {
		t.Fatalf("got %d actions, want 84", len(Actions))
	}
	seen := map[string]bool{}
	for _, a := range Actions {
		if a.Name == "" {
			t.Fatal("action with empty name")
		}
		if a.Doc == "" {
			t.Fatalf("action %q: missing Doc", a.Name)
		}
		if seen[a.Name] {
			t.Fatalf("duplicate action %q", a.Name)
		}
		seen[a.Name] = true
	}
	if FindAction("SpawnTab") == nil || FindAction("Nonexistent") != nil {
		t.Fatal("FindAction broken")
	}
}

func TestRelevantTo(t *testing.T) {
	if !RelevantTo(nil, "linux") {
		t.Fatal("empty tags must apply everywhere")
	}
	if !RelevantTo([]string{"Linux", "Wayland"}, "linux") {
		t.Fatal("Linux tags must match linux target")
	}
	if RelevantTo([]string{"Windows"}, "linux") {
		t.Fatal("Windows tag must not match linux target")
	}
	if !RelevantTo([]string{"Windows", "WSL"}, "windows") {
		t.Fatal("Windows tags must match windows target")
	}
	if !RelevantTo([]string{"macOS", "Unix"}, "macos") {
		t.Fatal("macOS tags must match macos target")
	}
	if !RelevantTo([]string{"Unix"}, "linux") {
		t.Fatal("Unix tag must match linux target")
	}
}

func TestDefaultFor(t *testing.T) {
	o := &Option{Field: Field{Name: "x", Default: "base"}, DefaultByOS: map[string]any{"macos": "mac"}}
	if DefaultFor(o, "macos") != "mac" || DefaultFor(o, "linux") != "base" {
		t.Fatal("DefaultFor broken")
	}
}

func TestNumericDefaultsAreFloat(t *testing.T) {
	var checkField func(f *Field, optName string)
	checkField = func(f *Field, optName string) {
		if f == nil {
			return
		}
		if f.Kind == Int || f.Kind == Float || f.Kind == Float01 {
			if f.Default != nil {
				if _, ok := f.Default.(float64); !ok {
					t.Errorf("option %q field %q: default is %T (%v), want float64", optName, f.Name, f.Default, f.Default)
				}
			}
		}
		for i := range f.Fields {
			checkField(&f.Fields[i], optName)
		}
	}

	for _, o := range Options {
		checkField(&o.Field, o.Name)
		for osName, def := range o.DefaultByOS {
			if o.Kind == Int || o.Kind == Float || o.Kind == Float01 {
				if def != nil {
					if _, ok := def.(float64); !ok {
						t.Errorf("option %q default for %s: is %T (%v), want float64", o.Name, osName, def, def)
					}
				}
			}
		}
	}
}
