package state

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func testPaths(t *testing.T) Paths {
	t.Helper()
	dir := t.TempDir()
	return Paths{
		ConfigDir: dir,
		Config:    filepath.Join(dir, "wezterm.lua"),
		State:     filepath.Join(dir, "wezterm_configurator.json"),
	}
}

func writeFiles(t *testing.T, p Paths, cfg, st string) {
	t.Helper()
	if err := os.WriteFile(p.Config, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.State, []byte(st), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBackupCopiesBytesExactly(t *testing.T) {
	p := testPaths(t)
	cfg := Marker + "\nreturn { font_size = 12 }\n\x00\xff binary tail"
	st := "{\"version\":1}\n"
	writeFiles(t, p, cfg, st)

	dir, err := Backup(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := parseBackupName(filepath.Base(dir)); !ok {
		t.Fatalf("backup dir %q does not match the naming pattern", dir)
	}
	if got := readFile(t, filepath.Join(dir, backupConfigName)); !bytes.Equal(got, []byte(cfg)) {
		t.Errorf("config copy = %q, want %q", got, cfg)
	}
	if got := readFile(t, filepath.Join(dir, backupStateName)); !bytes.Equal(got, []byte(st)) {
		t.Errorf("state copy = %q, want %q", got, st)
	}

	list, err := ListBackups(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("ListBackups = %d entries, want 1", len(list))
	}
	if list[0].Dir != dir || !list[0].HasConfig || !list[0].HasState {
		t.Errorf("BackupInfo = %+v", list[0])
	}
	if list[0].Time.IsZero() {
		t.Error("BackupInfo.Time is zero")
	}
}

func TestBackupNoFilesCreatesNothing(t *testing.T) {
	p := testPaths(t)
	dir, err := Backup(p)
	if err != nil {
		t.Fatal(err)
	}
	if dir != "" {
		t.Fatalf("Backup = %q, want %q", dir, "")
	}
	if entries, err := os.ReadDir(backupRoot(p)); err == nil && len(entries) != 0 {
		t.Errorf("backups dir has %d entries, want none", len(entries))
	} else if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestBackupPrunesToKeepNewest(t *testing.T) {
	p := testPaths(t)
	writeFiles(t, p, "v0", "s0")

	var made []string
	for i := range 8 {
		writeFiles(t, p, fmt.Sprintf("v%d", i), "s")
		dir, err := Backup(p)
		if err != nil {
			t.Fatal(err)
		}
		made = append(made, dir)
	}

	list, err := ListBackups(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != KeepBackups {
		t.Fatalf("kept %d backups, want %d", len(list), KeepBackups)
	}
	want := make(map[string]bool, KeepBackups)
	for _, d := range made[len(made)-KeepBackups:] {
		want[d] = true
	}
	for _, b := range list {
		if !want[b.Dir] {
			t.Errorf("kept unexpected backup %q (newest %q)", b.Dir, made[len(made)-1])
		}
	}
	for _, d := range made[:len(made)-KeepBackups] {
		if _, err := os.Stat(d); !os.IsNotExist(err) {
			t.Errorf("oldest backup %q was not pruned (err %v)", d, err)
		}
	}
	// Newest first.
	for i := 1; i < len(list); i++ {
		if list[i].Time.After(list[i-1].Time) {
			t.Fatalf("ListBackups not newest-first: %v then %v", list[i-1].Time, list[i].Time)
		}
	}
}

func TestBackupLeavesForeignEntriesAlone(t *testing.T) {
	p := testPaths(t)
	root := backupRoot(p)
	foreign := []string{"notes.txt", "keep-me", "20240101-0000.000", "20240101-000000.0000", "20240230-000000.000", "old"}
	for _, name := range foreign {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// A plain file that happens to carry a backup name is not a snapshot.
	if err := os.WriteFile(filepath.Join(root, "20230101-000000.000"), []byte("decoy"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, p, "cfg", "st")
	for range KeepBackups + 3 {
		if _, err := Backup(p); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range foreign {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Errorf("foreign entry %q removed: %v", name, err)
		}
	}
	if got := string(readFile(t, filepath.Join(root, "20230101-000000.000"))); got != "decoy" {
		t.Errorf("backup-named file content = %q, want it untouched", got)
	}

	list, err := ListBackups(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != KeepBackups {
		t.Fatalf("ListBackups = %d, want %d (foreign entries must be ignored)", len(list), KeepBackups)
	}
}

func TestRestoreOldestBackup(t *testing.T) {
	p := testPaths(t)
	writeFiles(t, p, "cfg-0", "state-0")
	first, err := Backup(p)
	if err != nil {
		t.Fatal(err)
	}
	for i := range KeepBackups - 1 {
		writeFiles(t, p, fmt.Sprintf("cfg-%d", i+1), fmt.Sprintf("state-%d", i+1))
		if _, err := Backup(p); err != nil {
			t.Fatal(err)
		}
	}
	writeFiles(t, p, "cfg-final", "state-final")

	var oldest BackupInfo
	list, err := ListBackups(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range list {
		if b.Dir == first {
			oldest = b
		}
	}
	if oldest.Dir == "" {
		t.Fatalf("oldest backup %q missing from list", first)
	}

	// Restoring the oldest is itself undoable, even though the prune inside
	// Backup drops the very directory being restored from.
	if err := RestoreBackup(p, oldest); err != nil {
		t.Fatal(err)
	}
	if got := string(readFile(t, p.Config)); got != "cfg-0" {
		t.Errorf("config = %q, want %q", got, "cfg-0")
	}
	if got := string(readFile(t, p.State)); got != "state-0" {
		t.Errorf("state = %q, want %q", got, "state-0")
	}

	list, err = ListBackups(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != KeepBackups {
		t.Fatalf("after restore: %d backups, want %d", len(list), KeepBackups)
	}
	var undone bool
	for _, b := range list {
		if got := string(readFile(t, filepath.Join(b.Dir, backupConfigName))); got == "cfg-final" {
			if s := string(readFile(t, filepath.Join(b.Dir, backupStateName))); s != "state-final" {
				t.Errorf("pre-restore backup state = %q", s)
			}
			undone = true
		}
	}
	if !undone {
		t.Error("pre-restore state was not preserved as a new backup")
	}
}

func TestRestoreLeavesMissingFilesUntouched(t *testing.T) {
	p := testPaths(t)
	// The snapshot holds a config only.
	if err := os.WriteFile(p.Config, []byte("only-config"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir, err := Backup(p)
	if err != nil {
		t.Fatal(err)
	}
	// A state file written after the snapshot must survive the restore.
	writeFiles(t, p, "changed-config", "newer-state")
	if err := RestoreBackup(p, BackupInfo{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	if got := string(readFile(t, p.Config)); got != "only-config" {
		t.Errorf("config = %q, want %q", got, "only-config")
	}
	if got := string(readFile(t, p.State)); got != "newer-state" {
		t.Errorf("state = %q, want it left untouched", got)
	}
}
