// Package luagen renders app state into a complete wezterm.lua file.
package luagen

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"weztermconfigurator/internal/catalog"
	"weztermconfigurator/internal/state"
)

var luaKeywords = map[string]bool{
	"and": true, "break": true, "do": true, "else": true, "elseif": true,
	"end": true, "false": true, "for": true, "function": true, "goto": true,
	"if": true, "in": true, "local": true, "nil": true, "not": true, "or": true,
	"repeat": true, "return": true, "then": true, "true": true, "until": true,
	"while": true,
}

var identRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var digitsRe = regexp.MustCompile(`^[0-9]+$`)

// Emit renders the whole wezterm.lua for s.
func Emit(s *state.State) (string, error) {
	var b strings.Builder
	b.WriteString(state.Marker + "\n")
	b.WriteString("-- Target platform: " + s.TargetOS + "\n")
	b.WriteString("local wezterm = require 'wezterm'\n")
	b.WriteString("local act = wezterm.action\n")
	b.WriteString("local config = wezterm.config_builder()\n")
	b.WriteString("local function with_foreground(style, color) style.foreground = color return style end\n")

	for i := range catalog.Options {
		o := &catalog.Options[i]
		if raw := s.Raw[o.Name]; raw != "" {
			b.WriteString("config." + o.Name + " = " + raw + "\n")
			continue
		}
		v, ok := s.Values[o.Name]
		if !ok || v == nil {
			continue
		}
		line, err := renderTop(v, o)
		if err != nil {
			return "", err
		}
		b.WriteString(line)
	}

	if len(s.Plugins) > 0 {
		b.WriteString("\n-- Plugins\n")
		for _, p := range s.Plugins {
			if p.URL == "" || !identRe.MatchString(p.Var) {
				continue
			}
			b.WriteString("local " + p.Var + " = wezterm.plugin.require " + q(p.URL) + "\n")
			if p.Apply {
				if opts := strings.TrimSpace(p.Opts); opts != "" {
					b.WriteString(p.Var + ".apply_to_config(config, " + opts + ")\n")
				} else {
					b.WriteString(p.Var + ".apply_to_config(config)\n")
				}
			}
		}
		b.WriteString("\n")
	}

	if len(s.Features) > 0 {
		var feat strings.Builder
		for i := range catalog.Features {
			f := &catalog.Features[i]
			params, ok := s.Features[f.ID]
			if !ok || !f.IsOn(params) {
				continue
			}
			code := f.Emit(params)
			if strings.TrimSpace(code) != "" {
				feat.WriteString("-- " + f.Name + " (" + f.Source + ")\n")
				feat.WriteString(code)
				feat.WriteString("\n")
			}
		}
		if feat.Len() > 0 {
			b.WriteString("\n-- Features\n")
			b.WriteString(feat.String())
		}
	}

 	out := b.String()
	if s.CustomLua != "" {
		out += "\n-- Custom Lua\n" + s.CustomLua + "\n\n"
	}
	out += "return config\n"
	return out, nil
}

// RenderValue renders one value the way Emit would (used for raw-Lua prefill).
func RenderValue(v any, f *catalog.Field, indent int) (string, error) {
	return renderValue(v, f, "option "+f.Name)
}

// renderTop renders one top-level assignment line (or multi-line block).
func renderTop(v any, o *catalog.Option) (string, error) {
	path := "option " + o.Name
	multiline := false
	switch o.Kind {
	case catalog.List, catalog.Keys, catalog.Mouse, catalog.KeyTables, catalog.NamedPalettes, catalog.Palette:
		multiline = true
	}
	if !multiline {
		s, err := renderValue(v, &o.Field, path)
		if err != nil {
			return "", err
		}
		return "config." + o.Name + " = " + s + "\n", nil
	}

	var b strings.Builder
	b.WriteString("config." + o.Name + " = {\n")
	var firstErr error
	fail := func(err error) bool {
		if err != nil && firstErr == nil {
			firstErr = err
			return true
		}
		return false
	}
	switch o.Kind {
	case catalog.Palette:
		m, _ := v.(map[string]any)
		for _, mf := range catalog.PaletteFields {
			mv, ok := m[mf.Name]
			if !ok || mv == nil {
				continue
			}
			s, err := renderValue(mv, &mf, path+"."+mf.Name)
			if fail(err) {
				break
			}
			b.WriteString("  " + keyExpr(mf.Name) + " = " + s + ",\n")
		}
	case catalog.NamedPalettes:
		m, _ := v.(map[string]any)
		for _, k := range sortedNames(m, true) {
			s, err := renderValue(m[k], &catalog.Field{Kind: catalog.Palette, Fields: catalog.PaletteFields}, path+"."+k)
			if fail(err) {
				break
			}
			b.WriteString("  " + keyExpr(k) + " = " + s + ",\n")
		}
	case catalog.KeyTables:
		m, _ := v.(map[string]any)
		for _, k := range sortedNames(m, false) {
			itemF := catalog.Field{Kind: catalog.List, Fields: catalog.KeyBindingFields}
			s, err := renderValue(m[k], &itemF, path+"."+k)
			if fail(err) {
				break
			}
			b.WriteString("  " + keyExpr(k) + " = " + s + ",\n")
		}
	case catalog.Keys:
		arr, _ := v.([]any)
		for _, item := range arr {
			s, err := renderValue(item, &catalog.Field{Kind: catalog.Struct, Fields: catalog.KeyBindingFields}, path)
			if fail(err) {
				break
			}
			b.WriteString("  " + s + ",\n")
		}
	case catalog.Mouse:
		arr, _ := v.([]any)
		for _, item := range arr {
			s, err := renderValue(item, &catalog.Field{Kind: catalog.Struct, Fields: catalog.MouseBindingFields}, path)
			if fail(err) {
				break
			}
			b.WriteString("  " + s + ",\n")
		}
	case catalog.List:
		arr, _ := v.([]any)
		for _, item := range arr {
			var s string
			var err error
			if o.Name == "exec_domains" {
				s, err = renderExecDomain(item, path)
			} else {
				s, err = renderValue(item, &catalog.Field{Kind: catalog.Struct, Fields: o.Fields}, path)
			}
			if fail(err) {
				break
			}
			b.WriteString("  " + s + ",\n")
		}
	}
	if firstErr != nil {
		return "", firstErr
	}
	b.WriteString("}\n")
	return b.String(), nil
}

func renderExecDomain(item any, path string) (string, error) {
	m, ok := item.(map[string]any)
	if !ok {
		return "", fmt.Errorf("%s: expected an object", path)
	}
	name, _ := m["name"].(string)
	fixup, _ := m["fixup"].(string)
	label, _ := m["label"].(string)
	if name == "" {
		return "", fmt.Errorf("%s: field name is required", path)
	}
	if fixup == "" {
		return "", fmt.Errorf("%s: field fixup is required", path)
	}
	s := "wezterm.exec_domain(" + q(name) + ", " + fixup
	if label != "" {
		s += ", " + q(label)
	}
	return s + ")", nil
}

func renderValue(v any, f *catalog.Field, path string) (string, error) {
	switch f.Kind {
	case catalog.Bool, catalog.Int, catalog.Float, catalog.Float01,
		catalog.String, catalog.Enum, catalog.Flags, catalog.Color,
		catalog.Path, catalog.Dir, catalog.Dimension, catalog.SchemeName, catalog.Lua:
		return renderScalar(v, f, path)
	case catalog.StringList:
		return renderStringList(v, path)
	case catalog.IntList:
		return renderNumberList(v, path)
	case catalog.StringMap, catalog.FloatMap:
		return renderMap(v, f, path)
	case catalog.Struct:
		m, ok := v.(map[string]any)
		if !ok {
			return "", fmt.Errorf("%s: expected an object", path)
		}
		if isMouseEventField(f) {
			return renderMouseEvent(m, f, path)
		}
		return renderStruct(m, f.Fields, path)
	case catalog.List:
		return renderList(v, f, path)
	case catalog.Union:
		return renderUnion(v, f, path)
	case catalog.Font:
		return renderFont(v, f, path)
	case catalog.Palette:
		m, ok := v.(map[string]any)
		if !ok {
			return "", fmt.Errorf("%s: expected an object", path)
		}
		return renderStruct(m, catalog.PaletteFields, path)
	case catalog.NamedPalettes:
		m, ok := v.(map[string]any)
		if !ok {
			return "", fmt.Errorf("%s: expected an object", path)
		}
		var parts []string
		for _, k := range sortedNames(m, true) {
			s, err := renderValue(m[k], &catalog.Field{Kind: catalog.Palette, Fields: catalog.PaletteFields}, path+"."+k)
			if err != nil {
				return "", err
			}
			parts = append(parts, keyExpr(k)+" = "+s)
		}
		return "{ " + strings.Join(parts, ", ") + " }", nil
	case catalog.Keys:
		return renderList(v, &catalog.Field{Kind: catalog.List, Fields: catalog.KeyBindingFields}, path)
	case catalog.KeyTables:
		m, ok := v.(map[string]any)
		if !ok {
			return "", fmt.Errorf("%s: expected an object", path)
		}
		var parts []string
		for _, k := range sortedNames(m, false) {
			s, err := renderList(m[k], &catalog.Field{Kind: catalog.List, Fields: catalog.KeyBindingFields}, path+"."+k)
			if err != nil {
				return "", err
			}
			parts = append(parts, keyExpr(k)+" = "+s)
		}
		return "{ " + strings.Join(parts, ", ") + " }", nil
	case catalog.Mouse:
		return renderList(v, &catalog.Field{Kind: catalog.List, Fields: catalog.MouseBindingFields}, path)
	case catalog.Leader:
		m, ok := v.(map[string]any)
		if !ok {
			return "", fmt.Errorf("%s: expected an object", path)
		}
		return renderStruct(m, catalog.LeaderFields, path)
	case catalog.Action:
		return renderAction(v, path)
	default:
		return "", fmt.Errorf("%s: unsupported kind %d", path, f.Kind)
	}
}

func renderScalar(v any, f *catalog.Field, path string) (string, error) {
	switch x := v.(type) {
	case bool:
		return strconv.FormatBool(x), nil
	case float64:
		return num(x), nil
	case string:
		switch f.Kind {
		case catalog.Bool, catalog.Int, catalog.Float, catalog.Float01:
			return "", fmt.Errorf("%s: expected a %v value", path, f.Kind)
		case catalog.Enum:
			if digitsRe.MatchString(x) || x == "true" || x == "false" {
				return x, nil
			}
			return q(x), nil
		case catalog.Flags:
			if x == "" {
				return q(f.EmptyToken), nil
			}
			return q(x), nil
		case catalog.Dimension:
			return q(x), nil
		case catalog.String:
			if f.NumericString && digitsRe.MatchString(x) {
				return x, nil
			}
			return q(x), nil
		default: // Color, Path, Dir, SchemeName, Lua
			if f.Kind == catalog.Lua {
				return x, nil
			}
			return q(x), nil
		}
	default:
		return "", fmt.Errorf("%s: unexpected value %#v for kind %d", path, v, int(f.Kind))
	}
}

func renderStringList(v any, path string) (string, error) {
	arr, ok := v.([]any)
	if !ok {
		return "", fmt.Errorf("%s: expected an array", path)
	}
	var parts []string
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			return "", fmt.Errorf("%s: expected string items", path)
		}
		parts = append(parts, q(s))
	}
	return "{ " + strings.Join(parts, ", ") + " }", nil
}

func renderNumberList(v any, path string) (string, error) {
	arr, ok := v.([]any)
	if !ok {
		return "", fmt.Errorf("%s: expected an array", path)
	}
	var parts []string
	for _, item := range arr {
		f, ok := item.(float64)
		if !ok {
			return "", fmt.Errorf("%s: expected number items", path)
		}
		parts = append(parts, num(f))
	}
	return "{ " + strings.Join(parts, ", ") + " }", nil
}

func renderMap(v any, f *catalog.Field, path string) (string, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", fmt.Errorf("%s: expected an object", path)
	}
	var parts []string
	for _, k := range sortedNames(m, false) {
		var s string
		switch x := m[k].(type) {
		case string:
			if f.Kind == catalog.FloatMap {
				return "", fmt.Errorf("%s: key %s expects a number", path, k)
			}
			s = q(x)
		case float64:
			s = num(x)
		default:
			return "", fmt.Errorf("%s: key %s: unsupported value", path, k)
		}
		parts = append(parts, keyExpr(k)+" = "+s)
	}
	return "{ " + strings.Join(parts, ", ") + " }", nil
}

// isMouseEventField detects the mouse binding event struct whose JSON shape
// is {<kind> = {streak, button}}.
func isMouseEventField(f *catalog.Field) bool {
	return len(f.Fields) == 3 && f.Fields[0].Name == "kind" && f.Fields[0].Kind == catalog.Enum
}

func renderMouseEvent(m map[string]any, f *catalog.Field, path string) (string, error) {
	if len(m) != 1 {
		return "", fmt.Errorf("%s: expected exactly one of Down, Up, Drag", path)
	}
	for kind, payload := range m {
		p, ok := payload.(map[string]any)
		if !ok {
			return "", fmt.Errorf("%s: event payload must be an object", path)
		}
		inner, err := renderStruct(p, f.Fields[1:], path)
		if err != nil {
			return "", err
		}
		return "{ " + keyExpr(kind) + " = " + inner + " }", nil
	}
	return "", nil
}

func renderStruct(m map[string]any, fields []catalog.Field, path string) (string, error) {
	var parts []string
	for i := range fields {
		mf := &fields[i]
		mv, ok := m[mf.Name]
		if !ok || mv == nil {
			if mf.Required {
				return "", fmt.Errorf("%s: field %s is required", path, mf.Name)
			}
			continue
		}
		if mf.Kind == catalog.Flags {
			if s, _ := mv.(string); s == "" || s == mf.EmptyToken {
				continue
			}
		}
		s, err := renderValue(mv, mf, path+"."+mf.Name)
		if err != nil {
			return "", err
		}
		parts = append(parts, keyExpr(mf.Name)+" = "+s)
	}
	return "{ " + strings.Join(parts, ", ") + " }", nil
}

func renderList(v any, f *catalog.Field, path string) (string, error) {
	arr, ok := v.([]any)
	if !ok {
		return "", fmt.Errorf("%s: expected an array", path)
	}
	var parts []string
	for i, item := range arr {
		ipath := fmt.Sprintf("%s[%d]", path, i)
		switch {
		case f.ItemKind == catalog.Action:
			s, err := renderAction(item, ipath)
			if err != nil {
				return "", err
			}
			parts = append(parts, s)
		case f.ItemKind != 0:
			s, err := renderValue(item, &catalog.Field{Kind: f.ItemKind}, ipath)
			if err != nil {
				return "", err
			}
			parts = append(parts, s)
		default:
			s, err := renderValue(item, &catalog.Field{Kind: catalog.Struct, Fields: f.Fields}, ipath)
			if err != nil {
				return "", err
			}
			parts = append(parts, s)
		}
	}
	return "{ " + strings.Join(parts, ", ") + " }", nil
}

func renderUnion(v any, f *catalog.Field, path string) (string, error) {
	switch x := v.(type) {
	case string:
		return q(x), nil
	case map[string]any:
		if len(x) != 1 {
			return "", fmt.Errorf("%s: union value must have exactly one variant", path)
		}
		variant, payload := "", any(nil)
		for k, p := range x {
			variant, payload = k, p
		}
		var vf *catalog.Field
		for i := range f.Fields {
			if f.Fields[i].Name == variant {
				vf = &f.Fields[i]
				break
			}
		}
		if vf == nil {
			return "", fmt.Errorf("%s: unknown variant %q", path, variant)
		}
		var inner string
		var err error
		switch {
		case vf.Scalar:
			if len(vf.Fields) == 0 {
				return "", fmt.Errorf("%s: variant %q takes no payload", path, variant)
			}
			inner, err = renderValue(payload, &vf.Fields[0], path)
		case vf.Tuple:
			arr, ok := payload.([]any)
			if !ok {
				return "", fmt.Errorf("%s: variant %q expects an array payload", path, variant)
			}
			var parts []string
			for i := range vf.Fields {
				if i >= len(arr) {
					if vf.Fields[i].Required {
						return "", fmt.Errorf("%s: variant %q: field %s is required", path, variant, vf.Fields[i].Name)
					}
					continue
				}
				s, e := renderValue(arr[i], &vf.Fields[i], path)
				if e != nil {
					return "", e
				}
				parts = append(parts, s)
			}
			inner = "{ " + strings.Join(parts, ", ") + " }"
		default:
			m, ok := payload.(map[string]any)
			if !ok {
				return "", fmt.Errorf("%s: variant %q expects an object payload", path, variant)
			}
			inner, err = renderStruct(m, vf.Fields, path)
		}
		if err != nil {
			return "", err
		}
		return "{ " + variant + " = " + inner + " }", nil
	default:
		return "", fmt.Errorf("%s: union value must be a string or object", path)
	}
}

func renderFont(v any, f *catalog.Field, path string) (string, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", fmt.Errorf("%s: expected an object", path)
	}
	fontList, _ := m["font"].([]any)
	if len(fontList) == 0 {
		return "", fmt.Errorf("%s: at least one font is required", path)
	}
	var parts []string
	for i, a := range fontList {
		am, ok := a.(map[string]any)
		if !ok {
			return "", fmt.Errorf("%s: font entries must be objects", path)
		}
		s, err := renderStruct(am, catalog.FontAttributesFields, fmt.Sprintf("%s.font[%d]", path, i))
		if err != nil {
			return "", err
		}
		parts = append(parts, s)
	}
	call := "wezterm.font_with_fallback({ " + strings.Join(parts, ", ") + " })"
	if f.FontForeground {
		if fg, ok := m["foreground"]; ok && fg != nil {
			fgs, _ := fg.(string)
			call = "with_foreground(" + call + ", " + q(fgs) + ")"
		}
	}
	return call, nil
}

func renderAction(v any, path string) (string, error) {
	switch x := v.(type) {
	case string:
		return "act." + x, nil
	case map[string]any:
		if len(x) != 1 {
			return "", fmt.Errorf("%s: action must have exactly one name", path)
		}
		name, arg := "", any(nil)
		for k, p := range x {
			name, arg = k, p
		}
		if name == "Lua" {
			s, _ := arg.(string)
			if s == "" {
				return "", fmt.Errorf("%s: Lua action needs a string", path)
			}
			return s, nil
		}
		def := catalog.FindAction(name)
		if def == nil {
			return "", fmt.Errorf("%s: unknown action %q", path, name)
		}
		if arg == nil || def.Arg == nil {
			return "act." + name, nil
		}
		if m, ok := arg.(map[string]any); ok && len(m) == 0 && def.BareOK {
			return "act." + name, nil
		}
		s, err := renderActionArg(arg, def.Arg, path)
		if err != nil {
			return "", err
		}
		return "act." + name + "(" + s + ")", nil
	default:
		return "", fmt.Errorf("%s: action must be a string or object", path)
	}
}

func renderActionArg(arg any, f *catalog.Field, path string) (string, error) {
	if f.Kind == catalog.Struct && f.Tuple {
		arr, ok := arg.([]any)
		if !ok {
			return "", fmt.Errorf("%s: expected an array argument", path)
		}
		var parts []string
		for i := range f.Fields {
			if i >= len(arr) {
				if f.Fields[i].Required {
					return "", fmt.Errorf("%s: field %s is required", path, f.Fields[i].Name)
				}
				continue
			}
			s, err := renderValue(arr[i], &f.Fields[i], path)
			if err != nil {
				return "", err
			}
			parts = append(parts, s)
		}
		return "{ " + strings.Join(parts, ", ") + " }", nil
	}
	if f.Kind == catalog.List && f.ItemKind == catalog.Action {
		return renderList(arg, f, path)
	}
	return renderValue(arg, f, path)
}

// q quotes s as a single-quoted Lua string.
func q(s string) string {
	var b strings.Builder
	b.WriteByte('\'')
	for i := range s {
		c := s[i]
		switch {
		case c == '\\':
			b.WriteString(`\\`)
		case c == '\'':
			b.WriteString(`\'`)
		case c == '\n':
			b.WriteString(`\n`)
		case c == '\r':
			b.WriteString(`\r`)
		case c == '\t':
			b.WriteString(`\t`)
		case c < 0x20:
			b.WriteString(fmt.Sprintf(`\%03d`, c))
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('\'')
	return b.String()
}

// num formats a JSON number: integral values without a decimal point.
func num(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// keyExpr renders a Lua table key token: [16], name, or ['name'].
// Callers append " = value".
func keyExpr(k string) string {
	switch {
	case digitsRe.MatchString(k):
		return "[" + k + "]"
	case identRe.MatchString(k) && !luaKeywords[k]:
		return k
	default:
		return "[" + q(k) + "]"
	}
}

// sortedNames sorts object keys; numeric keys sort numerically first, and
// nameSort (case-insensitive) applies for NamedPalettes.
func sortedNames(m map[string]any, nameSort bool) []string {
	keys := make([]string, 0, len(m))
	allDigits := true
	for k := range m {
		keys = append(keys, k)
		if !digitsRe.MatchString(k) {
			allDigits = false
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if allDigits {
			a, _ := strconv.Atoi(keys[i])
			b, _ := strconv.Atoi(keys[j])
			return a < b
		}
		if nameSort {
			return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
		}
		return keys[i] < keys[j]
	})
	return keys
}
