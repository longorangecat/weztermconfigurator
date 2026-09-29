package lint

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"weztermconfigurator/internal/state"
)

// modOrder is the canonical modifier order used to normalise mods strings.
var modOrder = []string{"SHIFT", "CTRL", "ALT", "SUPER", "LEADER"}

// KeyConflicts reports key and mouse bindings that can never fire because a
// later entry shadows them. WezTerm builds every key table into a map keyed by
// (key, mods) and inserts the entries in list order, so a duplicate keeps only
// the last one; mouse bindings are keyed by trigger and mods and behave the
// same.
//
// The comparison is deliberately simple: keys are lowercased and mods are
// compared as a set, so SHIFT+a and SHIFT+A count as one binding, and a
// binding only competes with others carrying LEADER. WezTerm additionally
// folds SHIFT|CTRL+a into CTRL+A, so bindings that collide only after that
// shift folding are not reported here.
func KeyConflicts(st *state.State) []string {
	if st == nil {
		return nil
	}
	// Options replaced by verbatim Lua are opaque to this check.
	val := func(name string) (any, bool) {
		if st.Raw[name] != "" {
			return nil, false
		}
		v, ok := st.Values[name]
		return v, ok
	}

	var out []string
	keys, _ := val("keys")
	top := bindings(keys)
	out = append(out, dupWarnings("keys", top)...)
	if tables, ok := val("key_tables"); ok {
		if m, ok := tables.(map[string]any); ok {
			names := make([]string, 0, len(m))
			for name := range m {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				out = append(out, dupWarnings("key_tables."+name, bindings(m[name]))...)
			}
		}
	}
	out = append(out, dupWarnings("mouse_bindings", mouseBindings(val))...)
	out = append(out, leaderWarning(val, top)...)
	return out
}

// binding is one assignment reduced to the identity WezTerm keys it by.
type binding struct {
	label string // e.g. "CTRL|SHIFT+t"
	id    string
}

// bindings extracts the valid key bindings of one list, keeping input order.
// Entries that are not objects or have no key are skipped; validation reports
// those.
func bindings(v any) []binding {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]binding, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(strm(m["key"])))
		if key == "" {
			continue
		}
		mods := normTokens(strm(m["mods"]))
		out = append(out, binding{label: joinLabel(mods, key), id: mods + "|" + key})
	}
	return out
}

// dupWarnings reports bindings used more than once in the same list, in the
// order the duplicates are first seen.
func dupWarnings(scope string, bs []binding) []string {
	first := make(map[string]int, len(bs))
	var out []string
	for i, b := range bs {
		if j, dup := first[b.id]; dup {
			out = append(out, fmt.Sprintf("%s: %s is bound 2 times (entries %d, %d); only the last one wins", scope, b.label, j+1, i+1))
			continue
		}
		first[b.id] = i
	}
	return out
}

// leaderWarning reports top-level bindings that can never fire because they
// repeat the leader key itself: WezTerm consumes that press as the prefix.
func leaderWarning(val func(string) (any, bool), top []binding) []string {
	v, ok := val("leader")
	if !ok {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(strm(m["key"])))
	if key == "" {
		return nil
	}
	mods := normTokens(strm(m["mods"]))
	id := mods + "|" + key
	var out []string
	for i, b := range top {
		if b.id == id {
			out = append(out, fmt.Sprintf("leader: %s matches keys entry %d; that binding never fires on its own", joinLabel(mods, key), i+1))
		}
	}
	return out
}

// mouseBindings reduces mouse bindings to their trigger identity, keeping
// input order.
func mouseBindings(val func(string) (any, bool)) []binding {
	v, ok := val("mouse_bindings")
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]binding, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ev, label, ok := mouseEvent(m["event"])
		if !ok {
			continue
		}
		mods, alt := normTokens(strm(m["mods"])), normAlt(m["alt_screen"])
		out = append(out, binding{
			label: joinLabel(mods, label) + " alt_screen=" + alt,
			id:    ev + "|" + mods + "|" + alt,
		})
	}
	return out
}

// mouseEvent returns the identity and a readable label for a mouse trigger.
// Both the nested {Down = {streak, button}} form emitted into wezterm.lua and
// the flat {kind, streak, button} form held in state are accepted.
func mouseEvent(v any) (id, label string, ok bool) {
	m, isMap := v.(map[string]any)
	if !isMap {
		return "", "", false
	}
	kind, streak, button := strm(m["kind"]), m["streak"], m["button"]
	if kind == "" {
		if len(m) != 1 {
			return "", "", false
		}
		for k, payload := range m {
			p, isMap := payload.(map[string]any)
			if !isMap {
				return "", "", false
			}
			kind, streak, button = k, p["streak"], p["button"]
		}
	}
	s := num(streak)
	if s == "" {
		s = "1"
	}
	btn, btnCount := strm(button), ""
	if bm, isMap := button.(map[string]any); isMap {
		for name, count := range bm {
			btn, btnCount = name, num(count)
		}
	}
	if btnCount == "1" {
		btnCount = ""
	}
	btn = strings.ToLower(btn)
	return normTokens(kind) + "|" + s + "|" + btn + "|" + btnCount,
		fmt.Sprintf("%s streak=%s %s", strings.ToLower(kind), s, btn), true
}

// normTokens canonicalises a "|"-separated WezTerm token list: order, case and
// duplicates do not matter, and NONE means no modifier.
func normTokens(mods string) string {
	seen := make(map[string]bool)
	for _, tok := range strings.Split(mods, "|") {
		if t := strings.ToUpper(strings.TrimSpace(tok)); t != "" && t != "NONE" {
			seen[t] = true
		}
	}
	out := make([]string, 0, len(seen))
	for _, m := range modOrder {
		if seen[m] {
			out = append(out, m)
			delete(seen, m)
		}
	}
	extra := make([]string, 0, len(seen))
	for m := range seen {
		extra = append(extra, m)
	}
	sort.Strings(extra)
	return strings.Join(append(out, extra...), "|")
}

// normAlt normalises alt_screen, whose default is "any"; the state may hold the
// enum string or the boolean it renders to.
func normAlt(v any) string {
	switch x := v.(type) {
	case bool:
		return strconv.FormatBool(x)
	case string:
		if s := strings.ToLower(strings.TrimSpace(x)); s != "" {
			return s
		}
	}
	return "any"
}

func joinLabel(mods, key string) string {
	if mods == "" {
		return key
	}
	return mods + "+" + key
}

func strm(v any) string {
	s, _ := v.(string)
	return s
}

func num(v any) string {
	switch x := v.(type) {
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case string:
		return strings.TrimSpace(x)
	}
	return ""
}
