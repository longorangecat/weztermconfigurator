package luaimport

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
	"weztermconfigurator/internal/catalog"
)

const (
	// maxNodes bounds the conversion walk so a file that builds a huge table
	// cannot turn an import into an allocation storm.
	// ponytail: far above any real config; lower it if this walk ever shows up.
	maxNodes = 200000
	// maxDepth bounds nesting so a table that refers to itself stops.
	maxDepth = 24
)

// skipErr marks a value with no JSON shape: a Lua function, or a wezterm call
// whose result only exists while WezTerm is running. The enclosing list drops
// just that entry and the reason reaches the user as a Skip.
type skipErr struct{ path, reason string }

func (e *skipErr) Error() string { return e.path + ": " + e.reason }

func errf(path, format string, args ...any) error {
	return fmt.Errorf(path+": "+format, args...)
}

// entry is one Lua table member with its key normalised to text.
type entry struct {
	key string
	num float64 // numeric key; NaN when the key is not a number
	val lua.LValue
}

// index reports whether the key is a list position (a whole number >= 1).
func (e entry) index() bool {
	return !math.IsNaN(e.num) && e.num >= 1 && e.num == math.Trunc(e.num)
}

// fontFields is the JSON shape of the catalog's Font kind: a required list of
// font attribute objects plus the optional foreground colour with_foreground adds.
var fontFields = func() []catalog.Field {
	f := []catalog.Field{{Name: "font", Kind: catalog.List, Fields: catalog.FontAttributesFields, Required: true}}
	f = append(f, catalog.FontAttributesFields...)
	return append(f, catalog.Field{Name: "foreground", Kind: catalog.Color})
}()

// value converts one Lua value to the JSON shape f describes. Entries dropped
// along the way come back as Skips; any other error costs the whole option.
func (r *rt) value(v lua.LValue, f *catalog.Field, path string, d int) (any, []Skip, error) {
	if d > maxDepth {
		return nil, nil, errf(path, "nested too deeply to import")
	}
	if r.nodes <= 0 {
		return nil, nil, errf(path, "the file builds too many values to import")
	}
	r.nodes--
	if t, ok := v.(*lua.LTable); ok {
		if p, ok := r.proxies[t]; ok {
			return nil, nil, &skipErr{path, p + " has no value until WezTerm itself runs, so it was not imported"}
		}
		if name, ok := r.actions[t]; ok {
			return name, nil, nil // a bare act.Name reference
		}
	}
	switch v.(type) {
	case *lua.LFunction, *lua.LUserData:
		return nil, nil, &skipErr{path, "a Lua function has no importable value; put it in Custom Lua"}
	}

	switch f.Kind {
	case catalog.Bool:
		b, ok := v.(lua.LBool)
		if !ok {
			return nil, nil, errf(path, "expected true or false")
		}
		return bool(b), nil, nil
	case catalog.Int, catalog.Float, catalog.Float01:
		n, ok := v.(lua.LNumber)
		if !ok {
			return nil, nil, errf(path, "expected a number")
		}
		x := float64(n)
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return nil, nil, errf(path, "must be a finite number")
		}
		if f.Kind == catalog.Int && x != math.Trunc(x) {
			return nil, nil, errf(path, "must be a whole number")
		}
		return x, nil, nil
	case catalog.String:
		switch x := v.(type) {
		case lua.LString:
			return string(x), nil, nil
		case lua.LNumber:
			// An all-digit string is emitted bare, so a number here is that
			// string written without quotes.
			if f.NumericString {
				return numText(float64(x)), nil, nil
			}
		}
		return nil, nil, errf(path, "expected a string")
	case catalog.Enum:
		// Enum tokens are emitted bare when they are all digits, true or
		// false, so those Lua literals name the same token.
		switch x := v.(type) {
		case lua.LString:
			return string(x), nil, nil
		case lua.LNumber:
			return numText(float64(x)), nil, nil
		case lua.LBool:
			return strconv.FormatBool(bool(x)), nil, nil
		}
		return nil, nil, errf(path, "expected one of %s", enumList(f))
	case catalog.Flags, catalog.Color, catalog.Path, catalog.Dir, catalog.SchemeName, catalog.Lua:
		s, ok := v.(lua.LString)
		if !ok {
			return nil, nil, errf(path, "expected a string")
		}
		return string(s), nil, nil
	case catalog.Dimension:
		switch x := v.(type) {
		case lua.LString:
			return string(x), nil, nil
		case lua.LNumber:
			return numText(float64(x)), nil, nil
		}
		return nil, nil, errf(path, `expected a size such as "1cell"`)
	case catalog.StringList:
		return r.list(v, &catalog.Field{Kind: catalog.List, ItemKind: catalog.String}, path, d)
	case catalog.IntList:
		return r.list(v, &catalog.Field{Kind: catalog.List, ItemKind: catalog.Int}, path, d)
	case catalog.StringMap:
		return r.mapOf(v, &catalog.Field{Kind: catalog.String}, path, d)
	case catalog.FloatMap:
		return r.mapOf(v, &catalog.Field{Kind: catalog.Float}, path, d)
	case catalog.Struct:
		t, err := r.table(v, path)
		if err != nil {
			return nil, nil, err
		}
		if f.Tuple {
			return r.tuple(t, f, path, d)
		}
		if isMouseEvent(f) {
			return r.mouseEvent(t, f, path, d)
		}
		return r.structOf(t, f.Fields, path, d)
	case catalog.List:
		return r.list(v, f, path, d)
	case catalog.Union:
		return r.union(v, f, path, d)
	case catalog.Action:
		return r.action(v, f, path, d)
	case catalog.Font:
		t, err := r.table(v, path)
		if err != nil {
			return nil, nil, err
		}
		return r.structOf(t, fontFields, path, d)
	case catalog.Palette:
		return r.value(v, &catalog.Field{Kind: catalog.Struct, Fields: catalog.PaletteFields}, path, d)
	case catalog.Keys:
		return r.value(v, &catalog.Field{Kind: catalog.List, Fields: catalog.KeyBindingFields}, path, d)
	case catalog.Mouse:
		return r.value(v, &catalog.Field{Kind: catalog.List, Fields: catalog.MouseBindingFields}, path, d)
	case catalog.Leader:
		return r.value(v, &catalog.Field{Kind: catalog.Struct, Fields: catalog.LeaderFields}, path, d)
	case catalog.KeyTables:
		return r.mapOf(v, &catalog.Field{Kind: catalog.List, Fields: catalog.KeyBindingFields}, path, d)
	case catalog.NamedPalettes:
		return r.mapOf(v, &catalog.Field{Kind: catalog.Struct, Fields: catalog.PaletteFields}, path, d)
	}
	return nil, nil, errf(path, "cannot import a value of this kind")
}

// list converts a Lua array; entries with no importable value are dropped and
// reported so one bad key binding does not cost the whole key map.
func (r *rt) list(v lua.LValue, f *catalog.Field, path string, d int) (any, []Skip, error) {
	t, ok := v.(*lua.LTable)
	if !ok {
		return nil, nil, errf(path, "expected a list")
	}
	es, err := r.entries(t, path)
	if err != nil {
		return nil, nil, err
	}
	var items, extra []entry
	for _, e := range es {
		if e.index() {
			items = append(items, e)
		} else {
			extra = append(extra, e)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].num < items[j].num })

	item := &catalog.Field{Kind: catalog.Struct, Fields: f.Fields}
	if f.ItemKind != 0 {
		item = &catalog.Field{Kind: f.ItemKind}
	}
	out := make([]any, 0, len(items))
	var skips []Skip
	for i, e := range items {
		if e.num != float64(i+1) {
			return nil, nil, errf(path, "list entries must be numbered 1 to %d", len(items))
		}
		ip := fmt.Sprintf("%s[%d]", path, i+1)
		v, s, err := r.value(e.val, item, ip, d+1)
		if se, ok := err.(*skipErr); ok {
			skips = append(skips, Skip{se.path, se.reason})
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		skips = append(skips, s...)
		out = append(out, v)
	}
	for _, e := range extra {
		skips = append(skips, Skip{path + "." + e.key, "ignored: not a numbered list entry"})
	}
	if len(out) == 0 && len(items) > 0 {
		return nil, skips, errf(path, "none of its %d entries could be imported", len(items))
	}
	return out, skips, nil
}

// structOf converts a named-field object. Members the schema does not know are
// reported rather than dropped silently.
func (r *rt) structOf(t *lua.LTable, fields []catalog.Field, path string, d int) (any, []Skip, error) {
	es, err := r.entries(t, path)
	if err != nil {
		return nil, nil, err
	}
	byName := make(map[string]entry, len(es))
	for _, e := range es {
		if e.index() {
			return nil, nil, errf(path, "expected named fields, not a list")
		}
		byName[e.key] = e
	}
	out := map[string]any{}
	var skips []Skip
	for i := range fields {
		f := &fields[i]
		e, ok := byName[f.Name]
		if !ok {
			continue
		}
		delete(byName, f.Name)
		v, s, err := r.value(e.val, f, path+"."+f.Name, d+1)
		if se, ok := err.(*skipErr); ok {
			// Without a field the schema requires the object is not itself
			// importable, so the item that holds it goes too.
			if f.Required {
				return nil, nil, se
			}
			skips = append(skips, Skip{se.path, se.reason})
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		skips = append(skips, s...)
		out[f.Name] = v
	}
	for _, e := range es {
		if _, known := byName[e.key]; known {
			skips = append(skips, Skip{path + "." + e.key, "not a field this option accepts"})
		}
	}
	return out, skips, nil
}

// tuple converts a positional object, such as the argument of an action whose
// schema is a tuple.
func (r *rt) tuple(t *lua.LTable, f *catalog.Field, path string, d int) (any, []Skip, error) {
	es, err := r.entries(t, path)
	if err != nil {
		return nil, nil, err
	}
	if len(es) > len(f.Fields) {
		return nil, nil, errf(path, "takes at most %d values, got %d", len(f.Fields), len(es))
	}
	out := make([]any, 0, len(es))
	var skips []Skip
	for i, e := range es {
		if e.num != float64(i+1) {
			return nil, nil, errf(path, "expected a list of values")
		}
		v, s, err := r.value(e.val, &f.Fields[i], fmt.Sprintf("%s[%d]", path, i+1), d+1)
		if err != nil {
			return nil, nil, err
		}
		skips = append(skips, s...)
		out = append(out, v)
	}
	return out, skips, nil
}

// union converts a value that is either a variant name or {Variant = payload}.
func (r *rt) union(v lua.LValue, f *catalog.Field, path string, d int) (any, []Skip, error) {
	switch x := v.(type) {
	case lua.LString:
		return string(x), nil, nil
	case *lua.LTable:
		es, err := r.entries(x, path)
		if err != nil {
			return nil, nil, err
		}
		if len(es) != 1 || es[0].index() {
			return nil, nil, errf(path, "expected one of %s", enumList(f))
		}
		e := es[0]
		if variant(f, e.key) == nil {
			return nil, nil, errf(path, "expected one of %s, got %q", enumList(f), e.key)
		}
		payload, skips, err := r.unionPayload(e.val, variant(f, e.key), path, d)
		if err != nil {
			return nil, nil, err
		}
		return map[string]any{e.key: payload}, skips, nil
	}
	return nil, nil, errf(path, "expected a string or an object with one of %s", enumList(f))
}

// unionPayload converts the value carried by a union variant, which is a bare
// scalar, a positional list or a named-field object depending on the schema.
func (r *rt) unionPayload(v lua.LValue, vf *catalog.Field, path string, d int) (any, []Skip, error) {
	switch {
	case vf.Scalar:
		if len(vf.Fields) == 0 {
			return nil, nil, errf(path, "variant %q takes no value", vf.Name)
		}
		return r.value(v, &vf.Fields[0], path, d+1)
	case vf.Tuple:
		t, ok := v.(*lua.LTable)
		if !ok {
			return nil, nil, errf(path, "expected a list of values")
		}
		return r.tuple(t, vf, path, d+1)
	default:
		t, ok := v.(*lua.LTable)
		if !ok {
			return nil, nil, errf(path, "expected an object")
		}
		return r.structOf(t, vf.Fields, path, d+1)
	}
}

// action converts a bare action reference or the {Name = argument} table that
// act.Name(arg) builds.
func (r *rt) action(v lua.LValue, f *catalog.Field, path string, d int) (any, []Skip, error) {
	switch x := v.(type) {
	case lua.LString:
		return string(x), nil, nil
	case *lua.LTable:
		es, err := r.entries(x, path)
		if err != nil {
			return nil, nil, err
		}
		if len(es) != 1 || es[0].index() {
			return nil, nil, errf(path, "an action must name exactly one action")
		}
		e := es[0]
		def := catalog.FindAction(e.key)
		if def == nil {
			return nil, nil, errf(path, "unknown action %q", e.key)
		}
		out := map[string]any{e.key: nil}
		if def.Arg == nil {
			return out, nil, nil
		}
		arg, skips, err := r.value(e.val, def.Arg, path+"."+e.key, d+1)
		if err != nil {
			return nil, nil, err
		}
		out[e.key] = arg
		return out, skips, nil
	}
	return nil, nil, errf(path, "an action must be act.Name or act.Name(...)")
}

// mapOf converts an object whose members all have the same shape.
func (r *rt) mapOf(v lua.LValue, inner *catalog.Field, path string, d int) (any, []Skip, error) {
	t, ok := v.(*lua.LTable)
	if !ok {
		return nil, nil, errf(path, "expected an object")
	}
	es, err := r.entries(t, path)
	if err != nil {
		return nil, nil, err
	}
	out := make(map[string]any, len(es))
	var skips []Skip
	for _, e := range es {
		x, s, err := r.value(e.val, inner, path+"."+e.key, d+1)
		if err != nil {
			return nil, nil, err
		}
		skips = append(skips, s...)
		out[e.key] = x
	}
	return out, skips, nil
}

// mouseEvent converts the mouse binding event struct, whose JSON shape is
// {<kind> = {streak, button}}.
func (r *rt) mouseEvent(t *lua.LTable, f *catalog.Field, path string, d int) (any, []Skip, error) {
	es, err := r.entries(t, path)
	if err != nil {
		return nil, nil, err
	}
	kinds := f.Fields[0].Enum
	if len(es) != 1 {
		return nil, nil, errf(path, "expected exactly one of %s", strings.Join(kinds, ", "))
	}
	e := es[0]
	if !has(kinds, e.key) {
		return nil, nil, errf(path, "expected one of %s, got %q", strings.Join(kinds, ", "), e.key)
	}
	payload, ok := e.val.(*lua.LTable)
	if !ok {
		return nil, nil, errf(path, "event payload must be an object")
	}
	inner, skips, err := r.structOf(payload, f.Fields[1:], path, d+1)
	if err != nil {
		return nil, nil, err
	}
	return map[string]any{e.key: inner}, skips, nil
}

func (r *rt) table(v lua.LValue, path string) (*lua.LTable, error) {
	t, ok := v.(*lua.LTable)
	if !ok {
		return nil, errf(path, "expected an object")
	}
	return t, nil
}

func (r *rt) entries(t *lua.LTable, path string) ([]entry, error) {
	if r.nodes <= 0 {
		return nil, errf(path, "the file builds too many values to import")
	}
	var es []entry
	t.ForEach(func(k, v lua.LValue) {
		e := entry{key: k.String(), num: math.NaN(), val: v}
		if n, ok := k.(lua.LNumber); ok {
			e.num = float64(n)
		}
		es = append(es, e)
	})
	r.nodes -= len(es) + 1
	return es, nil
}

// isMouseEvent detects the mouse binding event struct, the same way the
// emitter does.
func isMouseEvent(f *catalog.Field) bool {
	return len(f.Fields) == 3 && f.Fields[0].Name == "kind" && f.Fields[0].Kind == catalog.Enum
}

func variant(f *catalog.Field, name string) *catalog.Field {
	for i := range f.Fields {
		if f.Fields[i].Name == name {
			return &f.Fields[i]
		}
	}
	return nil
}

func has(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

func enumList(f *catalog.Field) string {
	if len(f.Enum) == 0 {
		return "the known values"
	}
	return strings.Join(f.Enum, ", ")
}

// numText formats a number the way the emitter prints one.
func numText(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}
