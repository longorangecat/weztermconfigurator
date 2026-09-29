package ui

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"weztermconfigurator/internal/luaimport"
)

// maxImportBytes guards the file picker: a real wezterm.lua is a few KB, so
// anything past 5 MB is the wrong file.
const maxImportBytes = 5 << 20

// importLua reads an existing wezterm.lua and offers to adopt the options it
// recognises.
func (a *appState) importLua() {
	d := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
		if err != nil || rc == nil {
			return
		}
		defer rc.Close()
		name := filepath.Base(rc.URI().Path())

		content, err := io.ReadAll(io.LimitReader(rc, maxImportBytes+1))
		if err != nil {
			dialog.ShowError(fmt.Errorf("reading %s: %w", name, err), a.win)
			return
		}
		if len(content) > maxImportBytes {
			dialog.ShowError(fmt.Errorf("%s is larger than 5 MB — that is not a WezTerm config", name), a.win)
			return
		}

		res, err := luaimport.Import(string(content), a.target)
		if err != nil {
			dialog.ShowError(fmt.Errorf("importing %s: %w", name, err), a.win)
			return
		}
		a.confirmImport(name, res)
	}, a.win)
	d.SetTitleText("Import a wezterm.lua")
	d.SetFilter(storage.NewExtensionFileFilter([]string{".lua"}))
	d.Show()
}

// confirmImport summarises what the import found and asks before replacing the
// current option values.
func (a *appState) confirmImport(name string, res *luaimport.Result) {
	if res == nil || res.State == nil {
		dialog.ShowInformation("Nothing to import",
			fmt.Sprintf("No options were recognised in %s.", name), a.win)
		return
	}

	head := widget.NewLabel(fmt.Sprintf("%d options imported from %s · %d skipped",
		len(res.Imported), name, len(res.Skipped)))
	head.Importance = widget.HighImportance

	warn := widget.NewLabel("The current option values and raw-Lua overrides are replaced. Plugins, features, your custom Lua and pinned options are kept. Undo puts the old values back.")
	warn.Importance = widget.WarningImportance
	warn.Wrapping = fyne.TextWrapWord

	body := []fyne.CanvasObject{head, warn}

	if len(res.Notes) > 0 {
		notes := widget.NewLabel("Notes:\n• " + strings.Join(res.Notes, "\n• "))
		notes.Wrapping = fyne.TextWrapWord
		body = append(body, notes)
	}

	if len(res.Skipped) > 0 {
		var b strings.Builder
		for _, s := range res.Skipped {
			fmt.Fprintf(&b, "• %s — %s\n", s.Name, s.Reason)
		}
		skipLbl := widget.NewLabel(b.String())
		skipLbl.Wrapping = fyne.TextWrapOff
		scroll := container.NewVScroll(skipLbl)
		scroll.SetMinSize(fyne.NewSize(520, 220))
		body = append(body, widget.NewLabel("Skipped:"), scroll)
	}

	var d *dialog.CustomDialog
	replace := widget.NewButton("Replace current settings", func() {
		a.applyImport(res, name)
		d.Hide()
	})

	d = dialog.NewCustomWithoutButtons("Import from "+name, container.NewVBox(body...), a.win)
	d.SetButtons([]fyne.CanvasObject{replace, widget.NewButton("Cancel", func() { d.Hide() })})
	d.Resize(fyne.NewSize(600, 520))
	d.Show()
}

// applyImport adopts the imported option values. Only Values and Raw are
// replaced: plugins, features, custom Lua, pinned options and the target are
// this window's own settings. It goes through markDirty, so it is undoable.
func (a *appState) applyImport(res *luaimport.Result, name string) {
	if res == nil || res.State == nil {
		return
	}
	vals, raw := res.State.Values, res.State.Raw
	if vals == nil {
		vals = map[string]any{}
	}
	if raw == nil {
		raw = map[string]string{}
	}
	a.st.Values = vals
	a.st.Raw = raw
	a.markDirty()
	a.flashStatus(widget.SuccessImportance, "✓  Imported %d options from %s", len(res.Imported), name)
	a.stateReplaced()
}
