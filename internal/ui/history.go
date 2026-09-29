package ui

import (
	"bytes"
	"encoding/json"
	"time"

	"weztermconfigurator/internal/state"
)

const (
	historyLimit    = 100
	historyCoalesce = 800 * time.Millisecond // edits closer together than this form one undo step
)

// history is an undo/redo stack of whole-state snapshots. Changed is called
// after every mutation (markDirty); a burst of edits (typing in an entry)
// becomes a single step.
type history struct {
	undo, redo []*state.State
	last       *state.State // state after the most recent change
	lastJSON   []byte
	lastChange time.Time
}

func newHistory(cur *state.State) *history {
	h := &history{}
	h.setLast(cur)
	return h
}

func (h *history) setLast(cur *state.State) {
	h.last = cur.Clone()
	h.lastJSON, _ = json.Marshal(cur)
	h.lastChange = time.Time{}
}

// Changed records that cur differs from the previous snapshot, if it does.
func (h *history) Changed(cur *state.State) {
	j, _ := json.Marshal(cur)
	if bytes.Equal(j, h.lastJSON) {
		return
	}
	now := time.Now()
	if now.Sub(h.lastChange) > historyCoalesce || len(h.undo) == 0 {
		h.undo = append(h.undo, h.last)
		if len(h.undo) > historyLimit {
			h.undo = h.undo[1:]
		}
	}
	h.redo = nil
	h.last = cur.Clone()
	h.lastJSON = j
	h.lastChange = now
}

// Undo returns the state to restore, or nil when there is nothing to undo.
func (h *history) Undo(cur *state.State) *state.State {
	if len(h.undo) == 0 {
		return nil
	}
	s := h.undo[len(h.undo)-1]
	h.undo = h.undo[:len(h.undo)-1]
	h.redo = append(h.redo, cur.Clone())
	h.setLast(s)
	return s.Clone()
}

// Redo returns the state to restore, or nil when there is nothing to redo.
func (h *history) Redo(cur *state.State) *state.State {
	if len(h.redo) == 0 {
		return nil
	}
	s := h.redo[len(h.redo)-1]
	h.redo = h.redo[:len(h.redo)-1]
	h.undo = append(h.undo, cur.Clone())
	h.setLast(s)
	return s.Clone()
}

func (h *history) CanUndo() bool { return len(h.undo) > 0 }
func (h *history) CanRedo() bool { return len(h.redo) > 0 }
