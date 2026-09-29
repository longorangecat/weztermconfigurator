package ui

import (
	"testing"
	"time"

	"weztermconfigurator/internal/state"
)

func TestHistoryUndoRedo(t *testing.T) {
	st := state.New("linux")
	h := newHistory(st)

	st.Values["font_size"] = 12.0
	h.Changed(st)
	h.lastChange = time.Time{} // end the burst
	st.Values["font_size"] = 14.0
	h.Changed(st)

	got := h.Undo(st)
	if got == nil || got.Values["font_size"] != 12.0 {
		t.Fatalf("undo → %v, want font_size 12", got)
	}
	got2 := h.Undo(got)
	if got2 == nil || len(got2.Values) != 0 {
		t.Fatalf("second undo → %v, want empty state", got2)
	}
	if h.Undo(got2) != nil {
		t.Fatal("undo past the beginning must return nil")
	}
	if r := h.Redo(got2); r == nil || r.Values["font_size"] != 12.0 {
		t.Fatalf("redo → %v, want font_size 12", r)
	}
}

func TestHistoryCoalescesBurstsAndClearsRedo(t *testing.T) {
	st := state.New("linux")
	h := newHistory(st)
	for _, s := range []string{"a", "ab", "abc"} { // typing: one undo step
		st.Values["term"] = s
		h.Changed(st)
	}
	if len(h.undo) != 1 {
		t.Fatalf("burst made %d undo steps, want 1", len(h.undo))
	}
	back := h.Undo(st)
	if _, ok := back.Values["term"]; ok {
		t.Fatalf("undo of a burst should reach the state before it, got %v", back.Values)
	}
	back.Values["term"] = "new"
	h.Changed(back)
	if h.CanRedo() {
		t.Fatal("a new edit must clear the redo stack")
	}
}
