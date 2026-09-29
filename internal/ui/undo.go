package ui

import (
	"fmt"

	"fyne.io/fyne/v2/widget"

	"weztermconfigurator/internal/state"
)

// flashStatus writes a one-off status line. Headless tests build an appState
// without a status label, hence the nil check.
func (a *appState) flashStatus(imp widget.Importance, format string, args ...any) {
	if a.status == nil {
		return
	}
	a.status.Importance = imp
	a.status.SetText(fmt.Sprintf(format, args...))
}

// adopt swaps the live state for a snapshot loaded from elsewhere (undo, redo,
// backup restore, profile load). The platform filter follows the snapshot so
// the page shows the options that snapshot targets.
func (a *appState) adopt(s *state.State) {
	*a.st = *s
	if a.st.TargetOS != "" {
		a.target = a.st.TargetOS
	}
}

// undo restores the previous state snapshot. hist.Undo already does the
// history bookkeeping, so markDirty must not be called here.
func (a *appState) undo() {
	s := a.hist.Undo(a.st)
	if s == nil {
		a.flashStatus(widget.HighImportance, "Nothing to undo")
		return
	}
	a.adopt(s)
	a.dirty = true
	a.flashStatus(widget.WarningImportance, "↶  Undid last change")
	a.stateReplaced()
}

// redo re-applies the state undone most recently.
func (a *appState) redo() {
	s := a.hist.Redo(a.st)
	if s == nil {
		a.flashStatus(widget.HighImportance, "Nothing to redo")
		return
	}
	a.adopt(s)
	a.dirty = true
	a.flashStatus(widget.WarningImportance, "↷  Redid last change")
	a.stateReplaced()
}
