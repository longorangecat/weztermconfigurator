package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func helpTestWindow(t *testing.T, buttons ...*hoverHelpButton) fyne.Window {
	t.Helper()
	test.NewTempApp(t)
	w := test.NewWindow(nil)
	t.Cleanup(w.Close)
	w.Resize(fyne.NewSize(600, 700))
	objs := make([]fyne.CanvasObject, len(buttons))
	for i, b := range buttons {
		b.win = w
		objs[i] = b
	}
	w.SetContent(container.NewVBox(objs...))
	return w
}

func TestHelpPlacement(t *testing.T) {
	win := fyne.NewSize(600, 700)
	small := fyne.NewSize(380, 220)
	huge := fyne.NewSize(380, 600) // does not fit: only then may covering the button pass

	for _, tc := range []struct {
		name    string
		btn     fyne.Position
		pop     fyne.Size
		wantX   float32
		wantY   float32
		fitsWin bool
	}{
		{"beside and below", fyne.NewPos(50, 300), small, 82, 332, true},
		{"above when the window is short", fyne.NewPos(50, 600), small, 82, 372, true},
		{"clamped to the right edge", fyne.NewPos(560, 100), small, 212, 132, true},
		{"clamped to the left edge", fyne.NewPos(0, 0), small, 32, 32, true},
		{"tall popup pulled back inside", fyne.NewPos(50, 350), huge, 82, 92, true},
		{"popup wider than the window", fyne.NewPos(50, 100), fyne.NewSize(900, 220), 8, 132, false},
	} {
		pos := helpPlacement(tc.btn, fyne.NewSize(24, 24), tc.pop, win)
		if pos.X != tc.wantX || pos.Y != tc.wantY {
			t.Errorf("%s: got %v, want (%v, %v)", tc.name, pos, tc.wantX, tc.wantY)
		}
		if tc.fitsWin && (pos.X < helpGap || pos.Y < helpGap ||
			pos.X+tc.pop.Width > win.Width-helpGap || pos.Y+tc.pop.Height > win.Height-helpGap) {
			t.Errorf("%s: %v is not inside the window", tc.name, pos)
		}
		if tc.fitsWin && overlaps(pos, tc.pop, tc.btn, fyne.NewSize(24, 24)) {
			t.Errorf("%s: popup at %v covers the button at %v", tc.name, pos, tc.btn)
		}
	}
}

func overlaps(a fyne.Position, as fyne.Size, b fyne.Position, bs fyne.Size) bool {
	return a.X < b.X+bs.Width && b.X < a.X+as.Width && a.Y < b.Y+bs.Height && b.Y < a.Y+as.Height
}

// TestHelpPopupHover is the flicker regression: the button must not close the
// popup it just opened, or the two re-arm each other on every mouse move.
func TestHelpPopupHover(t *testing.T) {
	btn := newHoverHelpButton("some help", nil)
	other := newHoverHelpButton("other help", nil)
	win := helpTestWindow(t, btn, other)
	overlays := func() int { return len(win.Canvas().Overlays().List()) }

	btn.MouseIn(&desktop.MouseEvent{})
	if overlays() != 0 {
		t.Fatal("popup opened before the hover delay")
	}

	btn.show()
	if overlays() != 1 {
		t.Fatalf("expected the popup on the canvas, got %d overlays", overlays())
	}
	btn.MouseOut()
	if overlays() != 1 {
		t.Fatal("the button closed its own popup; hovering it flickers")
	}
	(&helpPad{b: btn}).MouseOut() // pointer left the popup
	if overlays() != 1 {
		t.Fatal("popup closed without the grace period")
	}
	btn.hide()
	if overlays() != 0 {
		t.Fatal("popup did not close")
	}

	// only one popup at a time
	btn.show()
	other.show()
	if overlays() != 1 || btn.overlay != nil || other.overlay == nil {
		t.Fatal("expected the second popup to replace the first")
	}
	other.hide()

	// a click pins it: leaving it, and leaving the popup, keep it open
	btn.OnTapped()
	if !btn.pinned || overlays() != 1 {
		t.Fatal("a click did not pin the popup open")
	}
	(&helpPad{b: btn}).MouseOut()
	btn.MouseOut()
	if overlays() != 1 || helpHide != nil {
		t.Fatal("pinned popup closed when the pointer left")
	}
	(&helpBackdrop{helpPad{b: btn}}).TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if overlays() != 0 {
		t.Fatal("Escape did not close the pinned popup")
	}

	// a second click closes it again, and leaves nothing running
	btn.OnTapped()
	btn.OnTapped()
	if btn.pinned || overlays() != 0 {
		t.Fatal("a second click did not close the popup")
	}
	stopTimer(&helpShow)
	stopTimer(&helpHide)
}

// TestHeaderLayoutKeepsButtonsOnTheCard: a long option name must not push the
// Docs link, the star or Reset off the card, nor grow the row.
func TestHeaderLayoutKeepsButtonsOnTheCard(t *testing.T) {
	test.NewTempApp(t)
	badges := container.NewHBox(widget.NewLabel("linux"), widget.NewButton("?", nil))
	btns := container.NewHBox(widget.NewButton("Docs", nil), widget.NewButton("☆", nil),
		widget.NewButton("Lua", nil), widget.NewButtonWithIcon("Reset", theme.ContentUndoIcon(), nil))
	h := headerLayout{}
	row := container.New(h, widget.NewLabel("skip_close_confirmation_for_processes_named"), badges, btns)
	row.Resize(fyne.NewSize(560, 40))
	h.Layout(row.Objects, row.Size())

	if got := btns.Position().X + btns.Size().Width; got > 560 {
		t.Errorf("buttons end at %v, past the card", got)
	}
	if got := badges.Position().X; got < row.Objects[0].Size().Width {
		t.Errorf("name (%v wide) runs into the badges at %v", row.Objects[0].Size().Width, got)
	}
	if got := row.Objects[0].Size().Width; got < headerMinName {
		t.Errorf("name squeezed to %v, below the readable minimum", got)
	}
	short := container.New(h, widget.NewLabel("font"), badges, btns)
	if short.MinSize().Width != row.MinSize().Width {
		t.Errorf("a long name widened the row: %v vs %v", row.MinSize(), short.MinSize())
	}
}

// TestHeaderLayoutGivesTheNameRoom: a short name next to badges and buttons
// must be shown in full, and a long one truncated rather than pushing the
// buttons off the card.
func TestHeaderLayoutGivesTheNameRoom(t *testing.T) {
	test.NewTempApp(t)
	badges := container.NewHBox(widget.NewLabel("linux"), widget.NewLabel("since 2024-01-01"), widget.NewButton("?", nil))
	btns := container.NewHBox(widget.NewButton("Docs", nil), widget.NewButton("☆", nil),
		widget.NewButton("Lua", nil), widget.NewButtonWithIcon("Reset", theme.ContentUndoIcon(), nil))
	h := headerLayout{}

	short := widget.NewLabel("font")
	short.Truncation = fyne.TextTruncateEllipsis
	row := container.New(h, short, badges, btns)
	row.Resize(fyne.NewSize(700, 40))
	h.Layout(row.Objects, row.Size())
	if short.Size().Width < headerMinName+100 {
		t.Errorf("short name only got %v of the %v row", short.Size().Width, 700)
	}

	long := widget.NewLabel("skip_close_confirmation_for_processes_named")
	long.Truncation = fyne.TextTruncateEllipsis
	row = container.New(h, long, badges, btns)
	row.Resize(fyne.NewSize(600, 40))
	h.Layout(row.Objects, row.Size())
	if long.Size().Width < headerMinName {
		t.Errorf("long name squeezed to %v, below the readable minimum", long.Size().Width)
	}
	if got := btns.Position().X; got <= long.Size().Width {
		t.Errorf("name at %v runs into the buttons at %v", long.Size().Width, got)
	}
}
