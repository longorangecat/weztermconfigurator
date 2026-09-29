package state

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// KeepBackups is how many snapshots are retained under <ConfigDir>/backups.
const KeepBackups = 5

const (
	backupConfigName = "wezterm.lua"
	backupStateName  = "wezterm_configurator.json"
	// backupLayout is a fixed-width timestamp, so lexicographic name order
	// matches chronological order.
	backupLayout  = "20060102-150405.000"
	backupNameLen = len(backupLayout)
)

// BackupInfo describes one snapshot directory.
type BackupInfo struct {
	Dir       string
	Time      time.Time
	HasConfig bool
	HasState  bool
}

func backupRoot(p Paths) string { return filepath.Join(p.ConfigDir, "backups") }

// parseBackupName reports whether name is exactly a backup directory name.
func parseBackupName(name string) (time.Time, bool) {
	if len(name) != backupNameLen {
		return time.Time{}, false
	}
	t, err := time.Parse(backupLayout, name)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func readIfExists(path string) (b []byte, ok bool, err error) {
	b, err = os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return b, true, nil
}

// Backup snapshots the config and state files into a fresh timestamped
// directory and prunes old ones. It returns "" with no error when neither
// file exists yet.
func Backup(p Paths) (string, error) {
	cfg, hasCfg, err := readIfExists(p.Config)
	if err != nil {
		return "", fmt.Errorf("backup: %s: %w", p.Config, err)
	}
	st, hasSt, err := readIfExists(p.State)
	if err != nil {
		return "", fmt.Errorf("backup: %s: %w", p.State, err)
	}
	if !hasCfg && !hasSt {
		return "", nil
	}

	root := backupRoot(p)
	existing, err := ListBackups(p)
	if err != nil {
		return "", err
	}
	// Names must keep increasing even when the wall clock does not (or two
	// saves land in the same millisecond): pruning keeps the newest, and
	// "newest" is only well defined by creation order.
	base := timeNow().Truncate(time.Millisecond)
	if len(existing) > 0 && !base.After(existing[0].Time) {
		base = existing[0].Time
	}
	var dir string
	for i := 0; ; i++ {
		dir = filepath.Join(root, base.Add(time.Duration(i)*time.Millisecond).Format(backupLayout))
		if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
			break
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("backup: %s: %w", dir, err)
	}
	if hasCfg {
		if err := WriteAtomic(filepath.Join(dir, backupConfigName), cfg); err != nil {
			return "", err
		}
	}
	if hasSt {
		if err := WriteAtomic(filepath.Join(dir, backupStateName), st); err != nil {
			return "", err
		}
	}
	if err := pruneBackups(p); err != nil {
		return "", err
	}
	return dir, nil
}

// ListBackups returns the backup snapshots, newest first. Entries that do
// not match the backup naming pattern are ignored, and a missing backups
// directory yields an empty list.
func ListBackups(p Paths) ([]BackupInfo, error) {
	root := backupRoot(p)
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("backups: %s: %w", root, err)
	}
	var out []BackupInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		t, ok := parseBackupName(e.Name())
		if !ok {
			continue
		}
		dir := filepath.Join(root, e.Name())
		out = append(out, BackupInfo{
			Dir:       dir,
			Time:      t,
			HasConfig: fileExists(filepath.Join(dir, backupConfigName)),
			HasState:  fileExists(filepath.Join(dir, backupStateName)),
		})
	}
	slices.SortFunc(out, func(a, b BackupInfo) int {
		if a.Time.Equal(b.Time) {
			return strings.Compare(b.Dir, a.Dir)
		}
		return b.Time.Compare(a.Time)
	})
	return out, nil
}

// pruneBackups removes all but the newest KeepBackups matching directories.
func pruneBackups(p Paths) error {
	all, err := ListBackups(p)
	if err != nil {
		return err
	}
	for _, b := range all[min(KeepBackups, len(all)):] {
		if err := os.RemoveAll(b.Dir); err != nil {
			return fmt.Errorf("prune backup: %s: %w", b.Dir, err)
		}
	}
	return nil
}

// current files are backed up first, so a restore can itself be undone.
func RestoreBackup(p Paths, b BackupInfo) error {
	if b.Dir == "" {
		return errors.New("restore backup: empty backup directory")
	}
	// Read before snapshotting: the snapshot prune can drop this very dir.
	cfg, hasCfg, err := readIfExists(filepath.Join(b.Dir, backupConfigName))
	if err != nil {
		return fmt.Errorf("restore backup: %s: %w", b.Dir, err)
	}
	st, hasSt, err := readIfExists(filepath.Join(b.Dir, backupStateName))
	if err != nil {
		return fmt.Errorf("restore backup: %s: %w", b.Dir, err)
	}
	if _, err := Backup(p); err != nil {
		return err
	}
	if hasCfg {
		if err := WriteAtomic(p.Config, cfg); err != nil {
			return err
		}
	}
	if hasSt {
		if err := WriteAtomic(p.State, st); err != nil {
			return err
		}
	}
	return nil
}
