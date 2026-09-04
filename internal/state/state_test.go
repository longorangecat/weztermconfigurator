package state

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveXDGDefault(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp+"/.config")
	t.Setenv("WEZTERM_CONFIG_FILE", "")
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	p, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if p.ConfigDir != filepath.Join(tmp, ".config", "wezterm") {
		t.Fatalf("ConfigDir = %q", p.ConfigDir)
	}
	if p.Config != filepath.Join(tmp, ".config", "wezterm", "wezterm.lua") {
		t.Fatalf("Config = %q", p.Config)
	}
}

func TestResolveHomeDotLuaWins(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp+"/.config")
	t.Setenv("WEZTERM_CONFIG_FILE", "")
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	os.MkdirAll(filepath.Join(tmp, ".config", "wezterm"), 0o755)
	os.WriteFile(filepath.Join(tmp, ".wezterm.lua"), []byte("return {}"), 0o644)
	os.WriteFile(filepath.Join(tmp, ".config", "wezterm", "wezterm.lua"), []byte("return {}"), 0o644)
	p, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if p.Config != filepath.Join(tmp, ".wezterm.lua") {
		t.Fatalf("Config = %q, want ~/.wezterm.lua (source order)", p.Config)
	}
}

func TestResolveEnvVarWins(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp+"/.config")
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	envCfg := filepath.Join(tmp, "custom", "wezterm.lua")
	t.Setenv("WEZTERM_CONFIG_FILE", envCfg)
	p, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if p.Config != envCfg {
		t.Fatalf("Config = %q, want env override", p.Config)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s := New("linux")
	s.Values["font_size"] = 13.5
	s.Values["keys"] = []any{map[string]any{"key": "t", "mods": "CTRL|SHIFT"}}
	s.Raw["hyperlink_rules"] = "wezterm.default_hyperlink_rules()"
	s.CustomLua = "wezterm.on('x', function() end)"
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.TargetOS != "linux" || got.Values["font_size"] != 13.5 || got.Raw["hyperlink_rules"] != "wezterm.default_hyperlink_rules()" || got.CustomLua != s.CustomLua {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestLoadMissingFile(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got.TargetOS != RuntimeTarget() || got.Values == nil || got.Raw == nil {
		t.Fatalf("expected fresh runtime state, got %+v", got)
	}
}

func TestOwned(t *testing.T) {
	dir := t.TempDir()
	with := filepath.Join(dir, "with.lua")
	os.WriteFile(with, []byte(Marker+"\nrest"), 0o644)
	without := filepath.Join(dir, "without.lua")
	os.WriteFile(without, []byte("return {}"), 0o644)

	e, o, err := Owned(with)
	if err != nil || !e || !o {
		t.Fatalf("with marker: e=%v o=%v err=%v", e, o, err)
	}
	e, o, err = Owned(without)
	if err != nil || !e || o {
		t.Fatalf("without marker: e=%v o=%v err=%v", e, o, err)
	}
	e, o, err = Owned(filepath.Join(dir, "missing.lua"))
	if err != nil || e || o {
		t.Fatalf("missing: e=%v o=%v err=%v", e, o, err)
	}
}

func TestBackupUnowned(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "wezterm.lua")
	os.WriteFile(cfg, []byte("return {}"), 0o644)
	backup, err := BackupUnowned(cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(backup)
	if err != nil || string(b) != "return {}" {
		t.Fatalf("backup content: %q err=%v", b, err)
	}
}

func TestRuntimeTarget(t *testing.T) {
	want := "linux"
	switch runtime.GOOS {
	case "darwin":
		want = "macos"
	case "windows":
		want = "windows"
	}
	if RuntimeTarget() != want {
		t.Fatalf("RuntimeTarget() = %q, want %q", RuntimeTarget(), want)
	}
}
