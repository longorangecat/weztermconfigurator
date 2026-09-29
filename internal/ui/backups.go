package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"weztermconfigurator/internal/state"
)

const backupTimeLayout = "2006-01-02 15:04:05"

// backupLabel is one row of the restore list: when the copy was taken and what
// it actually holds, so a config-only backup is not mistaken for a full one.
func backupLabel(b state.BackupInfo) string {
	var parts []string
	if b.HasConfig {
		parts = append(parts, "wezterm.lua")
	}
	if b.HasState {
		parts = append(parts, "settings")
	}
	if len(parts) == 0 {
		parts = append(parts, "empty")
	}
	return b.Time.Format(backupTimeLayout) + "   " + strings.Join(parts, " + ")
}

// showRestoreDialog lists the automatic pre-save backups, newest first, and
// restores the selected one.
func (a *appState) showRestoreDialog() {
	backups, err := state.ListBackups(a.paths)
	if err != nil {
		dialog.ShowError(fmt.Errorf("listing backups: %w", err), a.win)
		return
	}
	if len(backups) == 0 {
		dialog.ShowInformation("No backups yet",
			fmt.Sprintf("wezterm.lua and your settings are copied into a timestamped backup automatically before every save.\n\nThe last %d backups are kept.", state.KeepBackups),
			a.win)
		return
	}

	sel := 0
	list := widget.NewList(
		func() int { return len(backups) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(backupLabel(backups[id]))
		},
	)
	list.OnSelected = func(id widget.ListItemID) { sel = int(id) }
	var d *dialog.CustomDialog
	restore := widget.NewButton("Restore", func() {
		if sel < 0 || sel >= len(backups) {
			return
		}
		b := backups[sel]
		dialog.ShowConfirm("Restore this backup?",
			fmt.Sprintf("wezterm.lua and your settings will be replaced by the copies from %s.\n\nThe files on disk right now are backed up first, so this can be undone from the next backup.",
				b.Time.Format(backupTimeLayout)),
			func(ok bool) {
				if !ok {
					return
				}
				d.Hide()
				a.restoreBackup(b)
			}, a.win)
	})

	d = dialog.NewCustomWithoutButtons("Restore a backup", list, a.win)
	d.SetButtons([]fyne.CanvasObject{restore, widget.NewButton("Close", func() { d.Hide() })})
	d.Resize(fyne.NewSize(520, 420))
	d.Show()
}

// restoreBackup copies a backup over the live files and reloads the app state
// from disk. The reload is recorded in history, so it is undoable.
func (a *appState) restoreBackup(b state.BackupInfo) {
	if err := state.RestoreBackup(a.paths, b); err != nil {
		dialog.ShowError(fmt.Errorf("restoring backup: %w", err), a.win)
		return
	}
	loaded, err := state.Load(a.paths.State)
	if err != nil {
		dialog.ShowError(fmt.Errorf("reloading settings: %w", err), a.win)
		return
	}
	a.adopt(loaded)
	a.hist.Changed(a.st)
	a.dirty = false
	a.flashStatus(widget.SuccessImportance, "✓  Restored backup from %s", b.Time.Format(backupTimeLayout))
	a.stateReplaced()
}
