package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// maxProfileNameLen bounds profile names so they stay usable as filenames.
const maxProfileNameLen = 64

var errBadProfileName = errors.New("invalid profile name")

func profilesDir(p Paths) string { return filepath.Join(p.ConfigDir, "profiles") }

// ValidProfileName rejects names that are empty, over-long, hidden, or that
// could escape the profiles directory.
func ValidProfileName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: empty", errBadProfileName)
	}
	if len(name) > maxProfileNameLen {
		return fmt.Errorf("%w: longer than %d characters", errBadProfileName, maxProfileNameLen)
	}
	if name[0] == '.' {
		return fmt.Errorf("%w: must not start with %q", errBadProfileName, ".")
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == ' ' || r == '_' || r == '-' || r == '.':
		default:
			return fmt.Errorf("%w: %q", errBadProfileName, r)
		}
	}
	return nil
}

func profilePath(p Paths, name string) (string, error) {
	if err := ValidProfileName(name); err != nil {
		return "", err
	}
	return filepath.Join(profilesDir(p), name+".json"), nil
}

// ListProfiles returns the saved profile names, sorted. A missing profiles
// directory yields an empty list.
func ListProfiles(p Paths) ([]string, error) {
	dir := profilesDir(p)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("profiles: %s: %w", dir, err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		if ValidProfileName(name) != nil {
			continue
		}
		out = append(out, name)
	}
	slices.Sort(out)
	return out, nil
}

// SaveProfile writes st to <ConfigDir>/profiles/<name>.json.
func SaveProfile(p Paths, name string, st *State) error {
	path, err := profilePath(p, name)
	if err != nil {
		return err
	}
	return st.Save(path)
}

// LoadProfile reads a saved profile; it errors when the profile is missing.
func LoadProfile(p Paths, name string) (*State, error) {
	path, err := profilePath(p, name)
	if err != nil {
		return nil, err
	}
	if !fileExists(path) {
		return nil, fmt.Errorf("profile %q: not found", name)
	}
	return Load(path)
}

// DeleteProfile removes a saved profile.
func DeleteProfile(p Paths, name string) error {
	path, err := profilePath(p, name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("profile %q: %w", name, err)
	}
	return nil
}
