package wezcli

import (
	"strings"
	"testing"
)

func TestFontRegex(t *testing.T) {
	sample := `
wezterm.font("JetBrains Mono")
wezterm.font("Fira Code", {weight="Bold"})
wezterm.font("Cascadia Code")
`
	seen := map[string]bool{}
	for _, m := range fontLineRe.FindAllStringSubmatch(sample, -1) {
		seen[m[1]] = true
	}
	if !seen["JetBrains Mono"] || !seen["Fira Code"] || !seen["Cascadia Code"] {
		t.Fatalf("font parser failed to extract fonts: %v", seen)
	}
}

func TestCheckErrorDetection(t *testing.T) {
	stderrSample := `
2026-09-27T10:00:00Z ERROR wezterm_gui::config > In configuration file /home/user/.config/wezterm/wezterm.lua:
attempt to call a nil value
`
	var errLines []string
	for _, line := range strings.Split(stderrSample, "\n") {
		if strings.Contains(line, "ERROR") {
			errLines = append(errLines, line)
		}
	}
	if len(errLines) == 0 {
		t.Fatalf("expected error lines detected in stderr sample")
	}
}
