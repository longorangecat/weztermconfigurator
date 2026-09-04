// Package wezcli wraps the wezterm binary: detection, font listing and
// config validation.
package wezcli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Find locates the wezterm executable; on Windows it also checks the common
// install locations.
func Find() (path string, ok bool) {
	if p, err := exec.LookPath("wezterm"); err == nil {
		return p, true
	}
	if runtime.GOOS == "windows" {
		for _, base := range []string{os.Getenv("ProgramFiles"), os.Getenv("LOCALAPPDATA")} {
			if base == "" {
				continue
			}
			c := filepath.Join(base, "WezTerm", "wezterm.exe")
			if _, err := os.Stat(c); err == nil {
				return c, true
			}
		}
	}
	return "", false
}

var fontLineRe = regexp.MustCompile(`wezterm\.font\("([^"]+)"`)

// ListSystemFonts returns the unique system font families, sorted.
func ListSystemFonts(bin string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "ls-fonts", "--list-system").Output()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, m := range fontLineRe.FindAllStringSubmatch(string(out), -1) {
		seen[m[1]] = true
	}
	families := make([]string, 0, len(seen))
	for f := range seen {
		families = append(families, f)
	}
	sort.Strings(families)
	return families, nil
}

// Check loads configPath with WezTerm and reports whether it parsed cleanly.
// WezTerm logs config errors to stderr while still exiting 0, so the exit
// code alone is not sufficient.
func Check(bin, configPath string) (ok bool, output string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--config-file", configPath, "ls-fonts")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	stdout, runErr := cmd.Output()

	errLines := []string{}
	for _, line := range strings.Split(stderr.String(), "\n") {
		if strings.Contains(line, "ERROR") {
			errLines = append(errLines, line)
		}
	}
	ok = runErr == nil && len(errLines) == 0

	stdoutLines := strings.Split(strings.TrimRight(string(stdout), "\n"), "\n")
	if len(stdoutLines) > 40 {
		stdoutLines = stdoutLines[:40]
	}
	var b strings.Builder
	b.WriteString(stderr.String())
	b.WriteString(strings.Join(stdoutLines, "\n"))
	return ok, b.String(), nil
}
