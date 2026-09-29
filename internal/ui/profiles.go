package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"weztermconfigurator/internal/state"
)

// showProfilesDialog manages named snapshots of the option values. Profiles
// live beside the config and are only applied when loaded; nothing here writes
// wezterm.lua.
func (a *appState) showProfilesDialog() {
	names, err := state.ListProfiles(a.paths)
	if err != nil {
		dialog.ShowError(fmt.Errorf("listing profiles: %w", err), a.win)
		return
	}

	sel := 0
	list := widget.NewList(
		func() int { return len(names) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(names[id])
		},
	)
	list.OnSelected = func(id widget.ListItemID) { sel = int(id) }
	// A VBox gives a child only its MinSize, so the list is stacked over a
	// spacer to make it tall enough to be useful.
	listBox := container.NewStack(spacer(160), list)

	hint := emptyHint("No saved profiles yet — name the current settings below to save one.")
	if len(names) > 0 {
		hint.Hide()
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Profile name")
	nameErr := widget.NewLabel("")
	nameErr.Importance = widget.DangerImportance
	nameErr.Hide()

	refresh := func() {
		n, err := state.ListProfiles(a.paths)
		if err != nil {
			dialog.ShowError(fmt.Errorf("listing profiles: %w", err), a.win)
			return
		}
		names = n
		if sel >= len(names) {
			sel = 0
		}
		if len(names) == 0 {
			hint.Show()
		} else {
			hint.Hide()
		}
		list.Refresh()
	}

	saveName := func(name string) {
		if err := state.SaveProfile(a.paths, name, a.st); err != nil {
			dialog.ShowError(fmt.Errorf("saving profile: %w", err), a.win)
			return
		}
		nameErr.Hide()
		nameEntry.SetText("")
		refresh()
		a.flashStatus(widget.SuccessImportance, "✓  Saved profile “%s”", name)
	}

	var d *dialog.CustomDialog
	saveBtn := widget.NewButton("Save current settings as…", func() {
		name := strings.TrimSpace(nameEntry.Text)
		if err := state.ValidProfileName(name); err != nil {
			nameErr.SetText(err.Error())
			nameErr.Show()
			return
		}
		overwrite := func() { saveName(name) }
		if contains(names, name) {
			dialog.ShowConfirm("Overwrite profile?",
				fmt.Sprintf("A profile named “%s” already exists. Replace it with the current settings?", name),
				func(ok bool) {
					if ok {
						overwrite()
					}
				}, a.win)
			return
		}
		overwrite()
	})

	loadBtn := widget.NewButton("Load", func() {
		if sel < 0 || sel >= len(names) {
			return
		}
		name := names[sel]
		apply := func() {
			if err := a.loadProfile(name); err != nil {
				dialog.ShowError(err, a.win)
				return
			}
			d.Hide()
		}
		if a.dirty {
			dialog.ShowConfirm("Discard unsaved changes?",
				fmt.Sprintf("Loading “%s” replaces the current settings. You can undo this.", name),
				func(ok bool) {
					if ok {
						apply()
					}
				}, a.win)
			return
		}
		apply()
	})

	delBtn := widget.NewButton("Delete", func() {
		if sel < 0 || sel >= len(names) {
			return
		}
		name := names[sel]
		dialog.ShowConfirm("Delete profile?",
			fmt.Sprintf("Delete the profile “%s”? This cannot be undone.", name),
			func(ok bool) {
				if !ok {
					return
				}
				if err := state.DeleteProfile(a.paths, name); err != nil {
					dialog.ShowError(fmt.Errorf("deleting profile: %w", err), a.win)
					return
				}
				refresh()
			}, a.win)
	})

	body := container.NewVBox(
		widget.NewLabel("Profiles are snapshots of your option values. They are stored next to your config and applied only when you load one."),
		spacer(6),
		hint,
		listBox,
		spacer(6),
		widget.NewLabel("Save the current settings as a new profile:"),
		container.NewBorder(nil, nil, nil, saveBtn, nameEntry),
		nameErr,
	)

	d = dialog.NewCustomWithoutButtons("Profiles", body, a.win)
	d.SetButtons([]fyne.CanvasObject{loadBtn, delBtn, widget.NewButton("Close", func() { d.Hide() })})
	d.Resize(fyne.NewSize(560, 460))
	d.Show()
}

// loadProfile replaces the option values with a saved profile. The pinned list
// belongs to this window, not to the snapshot, so it is carried over. The load
// goes through markDirty, so it is undoable.
func (a *appState) loadProfile(name string) error {
	st, err := state.LoadProfile(a.paths, name)
	if err != nil {
		return fmt.Errorf("loading profile: %w", err)
	}
	pinned := a.st.Pinned
	a.adopt(st)
	a.st.Pinned = pinned
	a.markDirty()
	a.stateReplaced()
	a.flashStatus(widget.SuccessImportance, "✓  Loaded profile “%s” — press Ctrl+S to apply", name)
	return nil
}
