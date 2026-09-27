package state

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteAtomic writes content to path atomically via a sibling temp file + rename,
// ensuring parent directory exists with 0755 and file with 0644.
func WriteAtomic(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	f, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	tmpName := f.Name()
	defer func() {
		// Clean up temp file on failure
		_ = os.Remove(tmpName)
	}()

	if _, err := f.Write(content); err != nil {
		_ = f.Close()
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// BackupUnowned copies path to path.bak-<timestamp> using WriteAtomic and returns the backup path.
func BackupUnowned(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	backup := path + ".bak-" + timeNow().Format("20060102-150405")
	if err := WriteAtomic(backup, b); err != nil {
		return "", err
	}
	return backup, nil
}
