package ui

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"weztermconfigurator/internal/lint"
)

// reviewSkipPref, when set, saves without showing the diff. It is only honoured
// for a save (proceed != nil); the menu entry always shows the review.
const reviewSkipPref = "review.skip"

func (a *appState) reviewSkip() bool {
	return a.app != nil && a.app.Preferences().Bool(reviewSkipPref)
}

func (a *appState) setReviewSkip(v bool) {
	if a.app != nil {
		a.app.Preferences().SetBool(reviewSkipPref, v)
	}
}

// diffLineText prefixes each diff line with its marker. Fold markers already
// read as "… 12 unchanged lines" and are shown centred, so they pass through.
func diffLineText(ln lint.DiffLine) string {
	switch ln.Op {
	case '+':
		return "+ " + ln.Text
	case '-':
		return "- " + ln.Text
	case '~':
		return ln.Text
	}
	return "  " + ln.Text
}

// reviewChanges shows what saving newLua would change. With proceed != nil the
// user confirms before the write; with proceed == nil it is a read-only
// "Review changes…" from the menu.
func (a *appState) reviewChanges(newLua string, proceed func()) {
	// A missing or unreadable config reads as empty: the diff then shows the
	// whole file as new, which is the honest picture.
	old, _ := os.ReadFile(a.paths.Config)
	oldText := string(old)

	if oldText == newLua {
		if proceed != nil {
			proceed()
			return
		}
		dialog.ShowInformation("No changes", "The generated wezterm.lua is identical to the file on disk.", a.win)
		return
	}
	if proceed != nil && a.reviewSkip() {
		proceed()
		return
	}

	full := lint.Diff(oldText, newLua)
	added, removed := lint.Stats(full)
	lines := lint.Collapse(full, 3)

	list := widget.NewList(
		func() int { return len(lines) },
		func() fyne.CanvasObject {
			t := canvas.NewText(" ", mustHex(colTextMuted))
			t.TextSize = theme.TextSize()
			t.TextStyle = fyne.TextStyle{Monospace: true}
			return t
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			t := o.(*canvas.Text)
			ln := lines[id]
			t.TextSize = theme.TextSize()
			t.TextStyle = fyne.TextStyle{Monospace: true}
			t.Alignment = fyne.TextAlignLeading
			switch ln.Op {
			case '+':
				t.Color = mustHex(colSuccess)
			case '-':
				t.Color = mustHex(colDanger)
			case '~':
				t.Color = mustHex(colTextHint)
				t.Alignment = fyne.TextAlignCenter
			default:
				t.Color = mustHex(colTextMuted)
			}
			t.Text = diffLineText(ln)
		},
	)
	// A VBox gives a child only its MinSize, so the diff list is stacked over a
	// spacer; without it the diff shows two lines and a scrollbar.
	listBox := container.NewStack(spacer(300), list)

	header := widget.NewLabel(fmt.Sprintf("+%d / −%d lines", added, removed))
	header.Importance = widget.HighImportance
	sub := widget.NewLabel(a.paths.Config)
	sub.Importance = widget.LowImportance

	body := []fyne.CanvasObject{header, sub}

	var skip *widget.Check
	if proceed != nil {
		skip = widget.NewCheck("Don't ask again", nil)
		body = append(body, skip)
	}

	content := container.NewVBox(append(body, listBox)...)
	d := dialog.NewCustomWithoutButtons("Review changes", content, a.win)
	if proceed != nil {
		d.SetButtons([]fyne.CanvasObject{
			widget.NewButton("Save", func() {
				if skip.Checked {
					a.setReviewSkip(true)
				}
				d.Hide()
				proceed()
			}),
			widget.NewButton("Cancel", func() { d.Hide() }),
		})
	} else {
		d.SetButtons([]fyne.CanvasObject{widget.NewButton("Close", func() { d.Hide() })})
	}
	d.Resize(fyne.NewSize(720, 480))
	d.Show()
}
