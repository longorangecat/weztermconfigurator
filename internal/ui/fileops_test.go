package ui

import (
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"weztermconfigurator/internal/lint"
	"weztermconfigurator/internal/luaimport"
	"weztermconfigurator/internal/state"
)

// newFileOpsApp builds a headless appState with just enough wiring for the
// state-replacing actions to run.
func newFileOpsApp(t *testing.T) *appState {
	t.Helper()
	dir := t.TempDir()
	st := state.New("linux")
	a := &appState{
		app:    test.NewTempApp(t),
		win:    test.NewWindow(nil),
		paths:  state.Paths{ConfigDir: dir, Config: filepath.Join(dir, "wezterm.lua"), State: filepath.Join(dir, "wezterm_configurator.json")},
		st:     st,
		target: "linux",
		page:   container.NewVBox(),
		status: widget.NewLabel(""),
	}
	a.hist = newHistory(a.st)
	return a
}

// seedHistory gives the test a known undo stack. history coalesces edits made
// within 800ms into a single step, which tests would otherwise race against.
func seedHistory(a *appState, cur, prev *state.State) {
	a.st = cur
	a.hist = &history{undo: []*state.State{prev}}
	a.hist.setLast(cur)
}

func TestUndoRedoRestoresState(t *testing.T) {
	a := newFileOpsApp(t)

	a.st.Values["font_size"] = 12.0
	a.markDirty()
	a.st.Values["font_size"] = 20.0

	a.undo()
	if got, ok := a.st.Values["font_size"]; ok {
		t.Fatalf("undo kept font_size = %v, want it gone", got)
	}
	if !a.dirty {
		t.Error("undo must leave the app dirty, the on-disk config no longer matches")
	}
	if got := a.status.Text; got != "↶  Undid last change" {
		t.Errorf("undo status = %q, want it to report the undo", got)
	}

	a.redo()
	if got := a.st.Values["font_size"]; got != 20.0 {
		t.Fatalf("redo restored font_size = %v, want 20", got)
	}
	if !a.dirty {
		t.Error("redo must leave the app dirty")
	}
}

func TestUndoWithEmptyHistory(t *testing.T) {
	a := newFileOpsApp(t)
	a.undo()
	if a.dirty {
		t.Error("a no-op undo must not mark the app dirty")
	}
	if a.status.Text != "Nothing to undo" {
		t.Errorf("status = %q, want the no-op message", a.status.Text)
	}
	a.redo()
	if a.status.Text != "Nothing to redo" {
		t.Errorf("status = %q, want the no-op message", a.status.Text)
	}
}

func TestRestoreBackupReloadsStateAndIsUndoable(t *testing.T) {
	a := newFileOpsApp(t)

	saved := state.New("linux")
	saved.Values["font_size"] = 11.0
	if err := saved.Save(a.paths.State); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := writeAtomic(a.paths.Config, "-- version one\n"); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := state.Backup(a.paths); err != nil {
		t.Fatalf("Backup: %v", err)
	}

	// The files on disk move on, and so does what the window thinks is current.
	live := state.New("linux")
	live.Values["font_size"] = 22.0
	if err := live.Save(a.paths.State); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := writeAtomic(a.paths.Config, "-- version two\n"); err != nil {
		t.Fatalf("write config: %v", err)
	}
	seedHistory(a, live, live)

	backups, err := state.ListBackups(a.paths)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("got %d backups, want the one taken above", len(backups))
	}

	a.restoreBackup(backups[0])
	after, err := state.ListBackups(a.paths)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(after) != 2 {
		t.Errorf("got %d backups after a restore, want 2 (the restore snapshots the files it replaced)", len(after))
	}
	if got := a.st.Values["font_size"]; got != 11.0 {
		t.Fatalf("restored font_size = %v, want the value from the backup (11)", got)
	}
	if a.dirty {
		t.Error("after a restore the files on disk match the state again")
	}

	a.undo()
	if got := a.st.Values["font_size"]; got != 22.0 {
		t.Errorf("undo after restore gave font_size = %v, want the pre-restore value (22)", got)
	}
}

func TestLoadProfileReplacesStateAndIsUndoable(t *testing.T) {
	a := newFileOpsApp(t)

	profile := state.New("linux")
	profile.Values["font_size"] = 11.0
	profile.CustomLua = "-- from the profile"
	profile.Pinned = []string{"font_size"}
	if err := state.SaveProfile(a.paths, "work", profile); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	live := state.New("linux")
	live.Values["font_size"] = 22.0
	live.CustomLua = "-- current"
	live.Pinned = []string{"color_scheme"}
	seedHistory(a, live, live)

	if err := a.loadProfile("work"); err != nil {
		t.Fatalf("loadProfile: %v", err)
	}
	if got := a.st.Values["font_size"]; got != 11.0 {
		t.Errorf("loaded font_size = %v, want the profile value (11)", got)
	}
	if a.st.CustomLua != "-- from the profile" {
		t.Errorf("custom Lua = %q, want the profile's", a.st.CustomLua)
	}
	if len(a.st.Pinned) != 1 || a.st.Pinned[0] != "color_scheme" {
		t.Errorf("pinned = %v, want this window's pin list kept", a.st.Pinned)
	}
	if !a.dirty {
		t.Error("loading a profile must leave the app dirty so it can be saved")
	}

	a.undo()
	if got := a.st.Values["font_size"]; got != 22.0 {
		t.Errorf("undo after load gave font_size = %v, want the pre-load value (22)", got)
	}
}

func TestApplyImportReplacesOnlyValuesAndRaw(t *testing.T) {
	a := newFileOpsApp(t)
	live := state.New("linux")
	live.Values["font_size"] = 22.0
	live.Raw["cursor_shape"] = "'beam'"
	live.CustomLua = "-- mine"
	live.Plugins = []state.Plugin{{URL: "https://example.com/p", Var: "p"}}
	live.Pinned = []string{"color_scheme"}
	seedHistory(a, live, live)

	res := &luaimport.Result{State: state.New("macos"), Imported: []string{"font_size", "color_scheme"}}
	res.State.Values["font_size"] = 18.0
	res.State.Raw["cursor_blink_rate"] = "0.5"

	a.applyImport(res, "old.lua")

	if got := a.st.Values["font_size"]; got != 18.0 {
		t.Errorf("font_size = %v, want the imported 18", got)
	}
	if len(a.st.Values) != 1 {
		t.Errorf("values = %v, want the imported set only", a.st.Values)
	}
	if len(a.st.Raw) != 1 || a.st.Raw["cursor_blink_rate"] != "0.5" {
		t.Errorf("raw = %v, want the imported set only", a.st.Raw)
	}
	if a.st.CustomLua != "-- mine" {
		t.Errorf("custom Lua = %q, import must not touch it", a.st.CustomLua)
	}
	if len(a.st.Plugins) != 1 {
		t.Errorf("plugins = %v, import must not touch them", a.st.Plugins)
	}
	if a.st.TargetOS != "linux" {
		t.Errorf("target = %q, import must not switch platform", a.st.TargetOS)
	}
	if len(a.st.Pinned) != 1 || a.st.Pinned[0] != "color_scheme" {
		t.Errorf("pinned = %v, import must not touch it", a.st.Pinned)
	}
	if !a.dirty {
		t.Error("an import must leave the app dirty so it can be saved")
	}
	if got := a.status.Text; got == "" {
		t.Error("import should report what happened")
	}

	a.undo()
	if got := a.st.Values["font_size"]; got != 22.0 {
		t.Errorf("undo after import gave font_size = %v, want the pre-import 22", got)
	}
}

func TestReviewChangesIdenticalTextProceedsWithoutDialog(t *testing.T) {
	a := newFileOpsApp(t)
	if err := writeAtomic(a.paths.Config, "-- identical\n"); err != nil {
		t.Fatalf("write config: %v", err)
	}

	calls := 0
	a.reviewChanges("-- identical\n", func() { calls++ })

	if calls != 1 {
		t.Fatalf("proceed called %d times, want exactly 1", calls)
	}
	if o := a.win.Canvas().Overlays().Top(); o != nil {
		t.Error("no dialog should open when there is nothing to review")
	}
}

func TestReviewChangesShowsDialogForRealChanges(t *testing.T) {
	a := newFileOpsApp(t) // no wezterm.lua on disk yet: everything is new

	calls := 0
	a.reviewChanges("return {}\n", func() { calls++ })

	if calls != 0 {
		t.Fatalf("proceed ran %d times, want 0 until the user confirms", calls)
	}
	if a.win.Canvas().Overlays().Top() == nil {
		t.Fatal("a changed config should open the review dialog")
	}

	// Drive the live dialog: only Save goes through.
	cancel := findButton(a.win.Canvas().Overlays().Top(), "Cancel")
	if cancel == nil {
		t.Fatal("the review dialog has no Cancel button")
	}
	test.Tap(cancel)
	if calls != 0 {
		t.Errorf("Cancel ran proceed %d times, want 0", calls)
	}

	// The diff is rebuilt from scratch on every call.
	a.reviewChanges("return { font_size = 12 }\n", func() { calls++ })
	save := findButton(a.win.Canvas().Overlays().Top(), "Save")
	if save == nil {
		t.Fatal("the reopened review dialog has no Save button")
	}
	test.Tap(save)
	if calls != 1 {
		t.Errorf("Save ran proceed %d times, want exactly 1", calls)
	}
}

// findButton walks an object tree (widgets are asked for their rendered
// children) looking for a button with the given label.
func findButton(o fyne.CanvasObject, text string) *widget.Button {
	if b, ok := o.(*widget.Button); ok && b.Text == text {
		return b
	}
	switch t := o.(type) {
	case *fyne.Container:
		for _, c := range t.Objects {
			if b := findButton(c, text); b != nil {
				return b
			}
		}
	case fyne.Widget:
		for _, c := range test.WidgetRenderer(t).Objects() {
			if b := findButton(c, text); b != nil {
				return b
			}
		}
	}
	return nil
}

func TestReviewChangesSkipPrefBypassesDialog(t *testing.T) {
	a := newFileOpsApp(t)
	if err := writeAtomic(a.paths.Config, "-- old\n"); err != nil {
		t.Fatalf("write config: %v", err)
	}
	a.setReviewSkip(true)

	calls := 0
	a.reviewChanges("-- new\n", func() { calls++ })

	if calls != 1 {
		t.Fatalf("proceed called %d times, want 1", calls)
	}
	if o := a.win.Canvas().Overlays().Top(); o != nil {
		t.Error("the skip preference must suppress the review dialog")
	}
	if !a.reviewSkip() {
		t.Error("skip preference did not persist")
	}
}

func TestDiffLineText(t *testing.T) {
	cases := []struct {
		op   byte
		text string
		want string
	}{
		{'+', "config.font_size = 12", "+ config.font_size = 12"},
		{'-', "config.font_size = 10", "- config.font_size = 10"},
		{' ', "local wezterm = {}", "  local wezterm = {}"},
		{'~', "… 9 unchanged lines", "… 9 unchanged lines"},
	}
	for _, c := range cases {
		if got := diffLineText(lint.DiffLine{Op: c.op, Text: c.text}); got != c.want {
			t.Errorf("op %q: got %q, want %q", string(c.op), got, c.want)
		}
	}
}
