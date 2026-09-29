package state

import (
	"encoding/json"
	"slices"
)

// IsPinned reports whether the option is pinned to Quick Settings.
func (s *State) IsPinned(name string) bool { return slices.Contains(s.Pinned, name) }

// TogglePin pins or unpins the option and returns the new pinned state.
func (s *State) TogglePin(name string) bool {
	if i := slices.Index(s.Pinned, name); i >= 0 {
		s.Pinned = slices.Delete(s.Pinned, i, i+1)
		return false
	}
	s.Pinned = append(s.Pinned, name)
	return true
}

// Clone returns a deep copy (via JSON, the state's own serialization).
func (s *State) Clone() *State {
	b, err := json.Marshal(s)
	if err != nil {
		panic("state: marshal: " + err.Error()) // State only holds JSON-safe values
	}
	c := New(s.TargetOS)
	if err := json.Unmarshal(b, c); err != nil {
		panic("state: unmarshal: " + err.Error())
	}
	if c.Values == nil {
		c.Values = map[string]any{}
	}
	if c.Raw == nil {
		c.Raw = map[string]string{}
	}
	if c.Features == nil {
		c.Features = map[string]map[string]string{}
	}
	return c
}
