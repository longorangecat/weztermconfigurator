package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProfileRoundTrip(t *testing.T) {
	p := testPaths(t)
	if names, err := ListProfiles(p); err != nil || len(names) != 0 {
		t.Fatalf("ListProfiles on missing dir = %v, %v; want empty, nil", names, err)
	}

	work := New("linux")
	work.Values["font_size"] = 14
	work.CustomLua = "-- extra"
	if err := SaveProfile(p, "work laptop", work); err != nil {
		t.Fatal(err)
	}
	home := New("windows")
	if err := SaveProfile(p, "home", home); err != nil {
		t.Fatal(err)
	}
	// A non-profile file must not show up in the listing.
	if err := os.WriteFile(filepath.Join(profilesDir(p), "scratch.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	names, err := ListProfiles(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "home" || names[1] != "work laptop" {
		t.Fatalf("ListProfiles = %v, want sorted [home work laptop]", names)
	}

	got, err := LoadProfile(p, "work laptop")
	if err != nil {
		t.Fatal(err)
	}
	if got.TargetOS != "linux" || got.Values["font_size"] != float64(14) || got.CustomLua != "-- extra" {
		t.Errorf("loaded profile = %+v", got)
	}
	if got.Values == nil || got.Raw == nil || got.Features == nil {
		t.Error("loaded profile has nil maps; Load normalisation was not applied")
	}

	// Overwriting replaces the previous content.
	work.Values["font_size"] = 20
	if err := SaveProfile(p, "work laptop", work); err != nil {
		t.Fatal(err)
	}
	got, err = LoadProfile(p, "work laptop")
	if err != nil {
		t.Fatal(err)
	}
	if got.Values["font_size"] != float64(20) {
		t.Errorf("after overwrite font_size = %v, want 20", got.Values["font_size"])
	}

	if err := DeleteProfile(p, "home"); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadProfile(p, "home"); err == nil {
		t.Error("LoadProfile of deleted profile returned no error")
	}
	if err := DeleteProfile(p, "home"); err == nil {
		t.Error("DeleteProfile of missing profile returned no error")
	}
	names, err = ListProfiles(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "work laptop" {
		t.Fatalf("after delete ListProfiles = %v", names)
	}
}

func TestLoadProfileNormalisesEmptyState(t *testing.T) {
	p := testPaths(t)
	if err := os.MkdirAll(profilesDir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profilesDir(p), "bare.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadProfile(p, "bare")
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != Version || got.TargetOS != RuntimeTarget() {
		t.Errorf("defaults not applied: %+v", got)
	}
	if got.Values == nil || got.Raw == nil || got.Features == nil {
		t.Error("nil maps not normalised")
	}
}

func TestProfileNameValidation(t *testing.T) {
	valid := []string{"a", "work", "work laptop", "v1.0-rc_2", "9lives"}
	for _, name := range valid {
		if err := ValidProfileName(name); err != nil {
			t.Errorf("ValidProfileName(%q) = %v, want nil", name, err)
		}
	}
	invalid := []string{
		"", "   ", ".", "..", ".hidden", "../x", "a/b", `a\b`, "a\x00b",
		strings.Repeat("n", maxProfileNameLen+1), "naïve", "tab\there", "a:b", "*",
	}
	for _, name := range invalid {
		if err := ValidProfileName(name); err == nil {
			t.Errorf("ValidProfileName(%q) = nil, want error", name)
		}
	}
	if err := ValidProfileName(strings.Repeat("n", maxProfileNameLen)); err != nil {
		t.Errorf("64-character name rejected: %v", err)
	}
}

func TestProfilesRejectTraversal(t *testing.T) {
	p := testPaths(t)
	for _, name := range []string{"../escaped", "..", "a/b"} {
		if err := SaveProfile(p, name, New("linux")); err == nil {
			t.Errorf("SaveProfile(%q) succeeded", name)
		}
		if _, err := LoadProfile(p, name); err == nil {
			t.Errorf("LoadProfile(%q) succeeded", name)
		}
		if err := DeleteProfile(p, name); err == nil {
			t.Errorf("DeleteProfile(%q) succeeded", name)
		}
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(p.ConfigDir), "escaped.json")); err == nil {
		t.Error("profile written outside the profiles directory")
	}
	if names, err := ListProfiles(p); err != nil || len(names) != 0 {
		t.Fatalf("ListProfiles = %v, %v; want empty, nil", names, err)
	}
}
