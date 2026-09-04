package state

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Paths are the locations the app reads and writes.
type Paths struct {
	ConfigDir string // $XDG_CONFIG_HOME/wezterm, else <home>/.config/wezterm
	Config    string // active wezterm.lua (WezTerm's lookup order)
	State     string // ConfigDir/wezterm_configurator.json
}

// Resolve determines the WezTerm config paths, mirroring WezTerm's own
// lookup order (config/src/config.rs): --config-file is not handled here;
// $WEZTERM_CONFIG_FILE → (Windows) wezterm.lua beside wezterm.exe →
// <home>/.wezterm.lua → ConfigDir/wezterm.lua → XDG_CONFIG_DIRS entries.
func Resolve() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}

	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(home, ".config")
	}
	configDir = filepath.Join(configDir, "wezterm")

	p := Paths{ConfigDir: configDir, State: filepath.Join(configDir, "wezterm_configurator.json")}

	// 1. $WEZTERM_CONFIG_FILE wins even if the file does not exist yet.
	if v := os.Getenv("WEZTERM_CONFIG_FILE"); v != "" {
		p.Config = v
		return p, nil
	}

	var candidates []string

	// 2. (Windows only) wezterm.lua beside the wezterm.exe on PATH.
	if RuntimeTarget() == "windows" {
		if exe, err := exec.LookPath("wezterm"); err == nil {
			candidates = append(candidates, filepath.Join(filepath.Dir(exe), "wezterm.lua"))
		}
	}

	// 3. ~/.wezterm.lua
	candidates = append(candidates, filepath.Join(home, ".wezterm.lua"))

	// 4. ConfigDir/wezterm.lua
	candidates = append(candidates, filepath.Join(configDir, "wezterm.lua"))

	// 5. (non-Windows) each $XDG_CONFIG_DIRS/wezterm/wezterm.lua
	if RuntimeTarget() != "windows" {
		for _, dir := range xdgConfigDirs() {
			candidates = append(candidates, filepath.Join(dir, "wezterm", "wezterm.lua"))
		}
	}

	for _, c := range candidates {
		if fileExists(c) {
			p.Config = c
			return p, nil
		}
	}

	// None exists: the default location will be created on first save.
	p.Config = filepath.Join(configDir, "wezterm.lua")
	return p, nil
}

func xdgConfigDirs() []string {
	var dirs []string
	for _, d := range strings.Split(os.Getenv("XDG_CONFIG_DIRS"), string(os.PathListSeparator)) {
		if d != "" {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
