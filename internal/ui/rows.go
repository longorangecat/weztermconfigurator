package ui

import (
	"fmt"
	"image/color"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"weztermconfigurator/internal/catalog"
	"weztermconfigurator/internal/luagen"
)

// row is one mounted option editor.
type row struct {
	opt      *catalog.Option
	validate func() error
	obj      fyne.CanvasObject
}

var (
	intRe    = regexp.MustCompile(`^-?\d+$`)
	floatRe  = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
	dimRe    = regexp.MustCompile(`^-?\d+(\.\d+)?(px|pt|cell|%)?$`)
	colorHex = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{9}|[0-9a-fA-F]{12})$`)
	colorFn  = regexp.MustCompile(`^(rgb|rgba|hsl|hsla|hsv|hwb)\(.+\)$`)
	colorCol = regexp.MustCompile(`^(rgb|rgba|hsl):.+$`)
	colorNam = regexp.MustCompile(`^[A-Za-z]+$`)
	weightRe = regexp.MustCompile(`^([A-Za-z]+|\d+)$`)
)

// isSet reports whether the option currently has a value.
func (a *appState) isSet(o *catalog.Option) bool {
	if a.st.Raw[o.Name] != "" {
		return true
	}
	_, ok := a.st.Values[o.Name]
	return ok
}

// newOptionRow builds the full row for one option.
func (a *appState) newOptionRow(o *catalog.Option) *row {
	r := &row{opt: o}

	nameLabel := widget.NewLabelWithStyle(o.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Monospace: true})

	var badges []fyne.CanvasObject
	for _, t := range o.Tags {
		b := widget.NewLabel(t)
		b.Importance = widget.WarningImportance
		b.TextStyle = fyne.TextStyle{Monospace: true}
		badges = append(badges, b)
	}
	if o.Deprecated != "" {
		b := widget.NewLabel("Deprecated")
		b.Importance = widget.DangerImportance
		badges = append(badges, b)
	}
	if o.Since != "" {
		b := widget.NewLabel("since " + o.Since)
		b.Importance = widget.LowImportance
		b.TextStyle = fyne.TextStyle{Monospace: true}
		badges = append(badges, b)
	}
	nameAndBadges := container.NewHBox(append([]fyne.CanvasObject{nameLabel}, badges...)...)

	helpText := o.Doc
	if o.Deprecated != "" {
		helpText += " DEPRECATED: " + o.Deprecated
	}
	def := catalog.DefaultFor(o, a.target)
	if def == nil && o.DefaultNote != "" {
		helpText += " Default: " + o.DefaultNote + "."
	} else if def != nil {
		helpText += " Default: " + literal(def) + "."
	}
	helpLabel := widget.NewLabel(helpText)
	helpLabel.Wrapping = fyne.TextWrapWord
	helpLabel.Importance = widget.LowImportance

	resetBtn := widget.NewButtonWithIcon("Reset", theme.ContentUndoIcon(), nil)
	luaToggle := widget.NewButton("Lua", nil)
	buttons := container.NewHBox(luaToggle, resetBtn)

	bar := canvas.NewRectangle(mustHex(colBorder))
	bar.SetMinSize(fyne.NewSize(4, 4))

	refresh := func() {
		if a.isSet(o) {
			nameLabel.Importance = widget.HighImportance
			resetBtn.Show()
			bar.FillColor = mustHex(colPrimary)
		} else if o.Deprecated != "" {
			nameLabel.Importance = widget.MediumImportance
			resetBtn.Hide()
			bar.FillColor = mustHex(colDanger)
		} else {
			nameLabel.Importance = widget.MediumImportance
			resetBtn.Hide()
			bar.FillColor = mustHex(colBorder)
		}
		bar.Refresh()
		nameLabel.Refresh()
		a.refreshNav()
	}

	editorBox := container.NewStack()
	buildForm := func() {
		editorBox.Objects = []fyne.CanvasObject{a.buildEditor(&o.Field,
			func() any { return a.st.Values[o.Name] },
			func(v any) {
				if v == nil {
					delete(a.st.Values, o.Name)
				} else {
					a.st.Values[o.Name] = v
				}
				a.markDirty()
				refresh()
			})}
		editorBox.Refresh()
	}
	buildRaw := func() {
		e := widget.NewMultiLineEntry()
		e.TextStyle = fyne.TextStyle{Monospace: true}
		e.SetMinRowsVisible(4)
		if raw := a.st.Raw[o.Name]; raw != "" {
			e.SetText(raw)
		} else if v, ok := a.st.Values[o.Name]; ok {
			if s, err := luagen.RenderValue(v, &o.Field, 0); err == nil {
				e.SetText(s)
			}
		}
		e.OnChanged = func(s string) {
			if s == "" {
				delete(a.st.Raw, o.Name)
			} else {
				a.st.Raw[o.Name] = s
			}
			a.markDirty()
			refresh()
		}
		editorBox.Objects = []fyne.CanvasObject{e}
		editorBox.Refresh()
	}

	rawMode := a.st.Raw[o.Name] != ""
	applyMode := func() {
		if rawMode {
			buildRaw()
			luaToggle.SetText("Form")
		} else {
			buildForm()
			luaToggle.SetText("Lua")
		}
	}
	applyMode()

	luaToggle.OnTapped = func() {
		rawMode = !rawMode
		if !rawMode {
			delete(a.st.Raw, o.Name)
		}
		applyMode()
		a.markDirty()
	}
	resetBtn.OnTapped = func() {
		delete(a.st.Values, o.Name)
		delete(a.st.Raw, o.Name)
		rawMode = false
		a.markDirty()
		refresh()
		applyMode()
	}

	r.validate = func() error {
		if a.st.Raw[o.Name] != "" || !a.isSet(o) {
			return nil
		}
		v, ok := a.st.Values[o.Name]
		if !ok {
			return nil
		}
		return validateField(v, &o.Field, o.Name)
	}
	refresh()

	nameRow := container.NewBorder(nil, nil, nameAndBadges, buttons)
	r.obj = container.NewBorder(nil, nil, bar, nil,
		container.NewVBox(nameRow, helpLabel, editorBox),
	)
	return r
}

// literal renders a JSON-shaped default for help text.
func literal(v any) string {
	switch x := v.(type) {
	case string:
		if x == "" {
			return `""`
		}
		return x
	case bool:
		return strconv.FormatBool(x)
	case float64:
		return numStr(x)
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = literal(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		if len(x) == 0 {
			return "{}"
		}
		parts := []string{}
		for k, e := range x {
			parts = append(parts, k+"="+literal(e))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func numStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// validateField recursively validates a stored JSON value against its schema.
func validateField(v any, f *catalog.Field, path string) error {
	switch f.Kind {
	case catalog.Int, catalog.Float, catalog.Float01:
		n, ok := v.(float64)
		if !ok {
			return fmt.Errorf("%s: expected a number", path)
		}
		return checkRange(n, f, path)
	case catalog.StringList:
		arr, ok := v.([]any)
		if !ok {
			return fmt.Errorf("%s: expected a list", path)
		}
		for _, item := range arr {
			s, ok := item.(string)
			if !ok {
				return fmt.Errorf("%s: expected string items", path)
			}
			if len(f.Enum) > 0 && !contains(f.Enum, s) {
				return fmt.Errorf("%s: %q is not one of %s", path, s, strings.Join(f.Enum, "|"))
			}
		}
	case catalog.IntList:
		arr, ok := v.([]any)
		if !ok {
			return fmt.Errorf("%s: expected a list", path)
		}
		for _, item := range arr {
			if _, ok := item.(float64); !ok {
				return fmt.Errorf("%s: expected number items", path)
			}
		}
	case catalog.StringMap, catalog.FloatMap:
		if _, ok := v.(map[string]any); !ok {
			return fmt.Errorf("%s: expected a table", path)
		}
	case catalog.Struct:
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: expected a table", path)
		}
		if len(f.Fields) == 3 && f.Fields[0].Name == "kind" {
			return nil // mouse event shape; checked by the emitter
		}
		for i := range f.Fields {
			mf := &f.Fields[i]
			mv, ok := m[mf.Name]
			if !ok || mv == nil {
				if mf.Required {
					return fmt.Errorf("%s: field %s is required", path, mf.Name)
				}
				continue
			}
			if err := validateField(mv, mf, path+"."+mf.Name); err != nil {
				return err
			}
		}
	case catalog.List:
		arr, ok := v.([]any)
		if !ok {
			return fmt.Errorf("%s: expected a list", path)
		}
		if f.FixedLen > 0 && len(arr) != f.FixedLen {
			return fmt.Errorf("%s: needs exactly %d items", path, f.FixedLen)
		}
		for i, item := range arr {
			ip := fmt.Sprintf("%s[%d]", path, i)
			if f.ItemKind != 0 {
				if err := validateField(item, &catalog.Field{Kind: f.ItemKind}, ip); err != nil {
					return err
				}
				continue
			}
			if err := validateField(item, &catalog.Field{Kind: catalog.Struct, Fields: f.Fields}, ip); err != nil {
				return err
			}
		}
	case catalog.Union:
		switch x := v.(type) {
		case string:
			if len(f.Enum) > 0 && !contains(f.Enum, x) {
				return fmt.Errorf("%s: %q is not one of %s", path, x, strings.Join(f.Enum, "|"))
			}
		case map[string]any:
			if len(x) != 1 {
				return fmt.Errorf("%s: union must have exactly one variant", path)
			}
			for variant, payload := range x {
				if !contains(f.Enum, variant) {
					return fmt.Errorf("%s: unknown variant %q", path, variant)
				}
				var vf *catalog.Field
				for i := range f.Fields {
					if f.Fields[i].Name == variant {
						vf = &f.Fields[i]
					}
				}
				if vf == nil {
					continue
				}
				if vf.Scalar {
					if err := validateField(payload, &vf.Fields[0], path); err != nil {
						return err
					}
				} else if err := validateField(payload, vf, path); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("%s: union must be a string or object", path)
		}
	case catalog.Font:
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: expected a table", path)
		}
		fontList, _ := m["font"].([]any)
		if len(fontList) == 0 {
			return fmt.Errorf("%s: at least one font is required", path)
		}
		for i, fa := range fontList {
			am, ok := fa.(map[string]any)
			if !ok {
				return fmt.Errorf("%s: font entries must be tables", path)
			}
			fam, _ := am["family"].(string)
			if fam == "" {
				return fmt.Errorf("%s.font[%d]: family is required", path, i)
			}
			if w, ok := am["weight"].(string); ok && w != "" && !weightRe.MatchString(w) {
				return fmt.Errorf("%s.font[%d]: invalid weight %q", path, i, w)
			}
		}
	case catalog.Color:
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%s: expected a color string", path)
		}
		return validateColor(s, path)
	case catalog.Dimension:
		s, ok := v.(string)
		if !ok {
			return nil // numeric dimensions are always valid
		}
		if !dimRe.MatchString(s) && !contains(f.Enum, s) {
			return fmt.Errorf("%s: invalid dimension %q", path, s)
		}
	case catalog.Action:
		switch v.(type) {
		case map[string]any, string:
			return nil
		default:
			return fmt.Errorf("%s: action must be a name or object", path)
		}
	case catalog.Leader:
		return validateField(v, &catalog.Field{Kind: catalog.Struct, Fields: catalog.LeaderFields}, path)
	case catalog.Keys:
		return validateField(v, &catalog.Field{Kind: catalog.List, Fields: catalog.KeyBindingFields}, path)
	case catalog.Mouse:
		return validateField(v, &catalog.Field{Kind: catalog.List, Fields: catalog.MouseBindingFields}, path)
	case catalog.KeyTables:
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: expected a table", path)
		}
		for name, tv := range m {
			if err := validateField(tv, &catalog.Field{Kind: catalog.List, Fields: catalog.KeyBindingFields}, path+"."+name); err != nil {
				return err
			}
		}
	case catalog.Palette:
		return validateField(v, &catalog.Field{Kind: catalog.Struct, Fields: catalog.PaletteFields}, path)
	case catalog.NamedPalettes:
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: expected a table", path)
		}
		for name, pv := range m {
			if err := validateField(pv, &catalog.Field{Kind: catalog.Palette, Fields: catalog.PaletteFields}, path+"."+name); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkRange(n float64, f *catalog.Field, path string) error {
	if f.Min != nil {
		if f.MinExclusive && n <= *f.Min {
			return fmt.Errorf("%s: must be greater than %v", path, *f.Min)
		}
		if !f.MinExclusive && n < *f.Min {
			return fmt.Errorf("%s: must be at least %v", path, *f.Min)
		}
	}
	if f.Max != nil && n > *f.Max {
		return fmt.Errorf("%s: must be at most %v", path, *f.Max)
	}
	return nil
}

func validateColor(s, path string) error {
	switch {
	case colorHex.MatchString(s), colorFn.MatchString(s), colorCol.MatchString(s), colorNam.MatchString(s):
		return nil
	case strings.EqualFold(s, "auto"), strings.EqualFold(s, "none"):
		return nil
	default:
		return fmt.Errorf("%s: invalid color %q", path, s)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// buildEditor builds the editor widget for one field value.
func (a *appState) buildEditor(f *catalog.Field, get func() any, set func(any)) fyne.CanvasObject {
	switch f.Kind {
	case catalog.Bool:
		c := widget.NewCheck("Enabled", func(on bool) { set(on) })
		if v, ok := get().(bool); ok {
			c.SetChecked(v)
		}
		return c

	case catalog.Int, catalog.Float:
		e := widget.NewEntry()
		if f.Kind == catalog.Int {
			e.Validator = func(s string) error {
				if !intRe.MatchString(s) {
					return fmt.Errorf("must be an integer")
				}
				n, _ := strconv.ParseFloat(s, 64)
				return checkRange(n, f, "value")
			}
		} else {
			e.Validator = func(s string) error {
				if !floatRe.MatchString(s) {
					return fmt.Errorf("must be a number")
				}
				n, _ := strconv.ParseFloat(s, 64)
				return checkRange(n, f, "value")
			}
		}
		if d := numStrOf(f.Default); d != "" {
			e.PlaceHolder = d
		}
		if v, ok := get().(float64); ok {
			e.SetText(numStr(v))
		}
		e.OnChanged = func(s string) {
			if s == "" {
				set(nil)
				return
			}
			if n, err := strconv.ParseFloat(s, 64); err == nil {
				set(n)
			}
		}
		return e

	case catalog.Float01:
		val := 0.0
		if v, ok := get().(float64); ok {
			val = v
		} else if d, ok := f.Default.(float64); ok {
			val = d
		}
		lbl := widget.NewLabel(numStr(val))
		s := widget.NewSlider(0, 1)
		s.Step = 0.01
		s.Value = val
		s.OnChangeEnded = func(v float64) {
			lbl.SetText(numStr(v))
			set(v)
		}
		return container.NewHBox(s, lbl)

	case catalog.String:
		e := widget.NewEntry()
		if d, ok := f.Default.(string); ok && d != "" {
			e.PlaceHolder = d
		} else if f.DefaultNote != "" {
			e.PlaceHolder = f.DefaultNote
		}
		if v, ok := get().(string); ok {
			e.SetText(v)
		}
		e.OnChanged = func(s string) {
			if s == "" {
				set(nil)
			} else {
				set(s)
			}
		}
		return e

	case catalog.SchemeName:
		return a.schemeEditor(get, set)

	case catalog.Enum:
		opts := append([]string{}, f.Enum...)
		if f.Name == "integrated_title_button_style" && a.target != "macos" {
			opts = dropToken(opts, "MacOsNative")
		}
		sel := widget.NewSelect(opts, func(v string) { set(v) })
		sel.PlaceHolder = "(default: " + literal(f.Default) + ")"
		if v, ok := get().(string); ok {
			sel.Selected = v
		}
		return sel

	case catalog.Flags:
		g := widget.NewCheckGroup(f.Enum, func(selected []string) {
			if len(selected) == 0 {
				if get() != nil {
					set(f.EmptyToken)
				}
				return
			}
			var ordered []string
			for _, tok := range f.Enum {
				for _, s := range selected {
					if s == tok {
						ordered = append(ordered, tok)
					}
				}
			}
			set(strings.Join(ordered, "|"))
		})
		g.Horizontal = true
		if v, ok := get().(string); ok && v != "" && v != f.EmptyToken {
			g.SetSelected(strings.Split(v, "|"))
		}
		return g

	case catalog.Color:
		return a.colorEditor(f, get, set)

	case catalog.Path, catalog.Dir:
		e := widget.NewEntry()
		if v, ok := get().(string); ok {
			e.SetText(v)
		}
		e.OnChanged = func(s string) {
			if s == "" {
				set(nil)
			} else {
				set(s)
			}
		}
		browse := widget.NewButtonWithIcon("Browse…", theme.FolderOpenIcon(), func() {
			if f.Kind == catalog.Dir {
				dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
					if err != nil || lu == nil {
						return
					}
					e.SetText(lu.Path())
				}, a.win)
				return
			}
			dialog.ShowFileOpen(func(r fyne.URIReadCloser, err error) {
				if err != nil || r == nil {
					return
				}
				e.SetText(r.URI().Path())
				r.Close()
			}, a.win)
		})
		return container.NewBorder(nil, nil, nil, browse, e)

	case catalog.Dimension:
		e := widget.NewEntry()
		e.Validator = func(s string) error {
			if !dimRe.MatchString(s) {
				if len(f.Enum) > 0 && contains(f.Enum, s) {
					return nil
				}
				return fmt.Errorf("use a number with optional px|pt|cell|%% unit")
			}
			return nil
		}
		if d, ok := f.Default.(string); ok {
			e.PlaceHolder = d
		}
		if v, ok := get().(string); ok {
			e.SetText(v)
		} else if v, ok := get().(float64); ok {
			e.SetText(numStr(v))
		}
		e.OnChanged = func(s string) {
			if s == "" {
				set(nil)
				return
			}
			if floatRe.MatchString(s) {
				n, _ := strconv.ParseFloat(s, 64)
				set(n)
			} else {
				set(s)
			}
		}
		return e

	case catalog.StringList, catalog.IntList:
		return a.listTextEditor(f, get, set)

	case catalog.StringMap, catalog.FloatMap:
		e := widget.NewMultiLineEntry()
		e.SetMinRowsVisible(3)
		e.TextStyle = fyne.TextStyle{Monospace: true}
		e.Validator = func(s string) error {
			for _, line := range strings.Split(s, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				k, v, ok := strings.Cut(line, "=")
				if !ok || strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
					return fmt.Errorf("use KEY=VALUE per line")
				}
				if f.Kind == catalog.FloatMap {
					if _, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err != nil {
						return fmt.Errorf("%s: value must be a number", strings.TrimSpace(k))
					}
				}
			}
			return nil
		}
		e.SetText(mapToText(get()))
		e.OnChanged = func(s string) {
			m := map[string]any{}
			for _, line := range strings.Split(s, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				k, v, _ := strings.Cut(line, "=")
				k, v = strings.TrimSpace(k), strings.TrimSpace(v)
				if k == "" || v == "" {
					continue
				}
				if f.Kind == catalog.FloatMap {
					if n, err := strconv.ParseFloat(v, 64); err == nil {
						m[k] = n
						continue
					}
				}
				m[k] = v
			}
			if len(m) == 0 {
				set(nil)
			} else {
				set(m)
			}
		}
		return e

	case catalog.Union:
		return a.unionEditor(f, get, set)

	case catalog.Lua:
		e := widget.NewMultiLineEntry()
		e.TextStyle = fyne.TextStyle{Monospace: true}
		e.SetMinRowsVisible(6)
		if v, ok := get().(string); ok {
			e.SetText(v)
		} else if d, ok := f.Default.(string); ok && d != "" {
			e.PlaceHolder = d
		}
		e.OnChanged = func(s string) {
			if s == "" {
				set(nil)
			} else {
				set(s)
			}
		}
		return e

	case catalog.Font:
		return a.fontEditor(f, get, set)

	case catalog.Palette:
		return a.paletteEditor(f, get, set)

	case catalog.NamedPalettes:
		return a.namedPalettesEditor(f, get, set)

	case catalog.Keys:
		return a.keysEditor(get, set)

	case catalog.KeyTables:
		return a.keyTablesEditor(get, set)

	case catalog.Leader, catalog.Struct:
		return a.structEditor(f, get, set)

	case catalog.Mouse:
		return a.mouseEditor(get, set)

	case catalog.List:
		return a.genericListEditor(f, get, set)

	default:
		return widget.NewLabel("(unsupported editor)")
	}
}

func numStrOf(v any) string {
	if f, ok := v.(float64); ok {
		return numStr(f)
	}
	return ""
}

func dropToken(list []string, tok string) []string {
	out := make([]string, 0, len(list))
	for _, t := range list {
		if t != tok {
			out = append(out, t)
		}
	}
	return out
}

func mapToText(v any) string {
	m, ok := v.(map[string]any)
	if !ok || len(m) == 0 {
		return ""
	}
	lines := []string{}
	for k, val := range m {
		switch x := val.(type) {
		case string:
			lines = append(lines, k+"="+x)
		case float64:
			lines = append(lines, k+"="+numStr(x))
		}
	}
	return strings.Join(lines, "\n")
}

// schemeEditor: SelectEntry over built-ins + user schemes + swatch strip.
func (a *appState) schemeEditor(get func() any, set func(any)) fyne.CanvasObject {
	names := catalog.SchemeNames()
	if m, ok := a.st.Values["color_schemes"].(map[string]any); ok {
		for k := range m {
			names = append(names, k)
		}
	}
	e := widget.NewSelectEntry(names)
	e.PlaceHolder = "(default palette)"
	if v, ok := get().(string); ok {
		e.SetText(v)
	}
	swatches := container.NewHBox()
	find := func(name string) *catalog.Scheme {
		for i := range catalog.Schemes() {
			if catalog.Schemes()[i].Name == name {
				return &catalog.Schemes()[i]
			}
		}
		return nil
	}
	updateSwatches := func(name string) {
		swatches.Objects = nil
		if s := find(name); s != nil {
			if ansi, ok := s.Colors["ansi"].([]any); ok {
				for _, c := range ansi {
					r := canvas.NewRectangle(parseColor(fmt.Sprint(c)))
					r.SetMinSize(fyne.NewSize(14, 14))
					swatches.Objects = append(swatches.Objects, r)
				}
			}
			if b, ok := s.Colors["brights"].([]any); ok {
				for _, c := range b {
					r := canvas.NewRectangle(parseColor(fmt.Sprint(c)))
					r.SetMinSize(fyne.NewSize(14, 14))
					swatches.Objects = append(swatches.Objects, r)
				}
			}
		}
		swatches.Refresh()
	}
	e.OnChanged = func(s string) {
		set(s)
		updateSwatches(s)
	}
	updateSwatches(e.Text)
	return container.NewVBox(e, swatches)
}

// colorEditor: entry + swatch + picker button.
func (a *appState) colorEditor(f *catalog.Field, get func() any, set func(any)) fyne.CanvasObject {
	e := widget.NewEntry()
	e.Validator = func(s string) error {
		if s == "" {
			return nil
		}
		return validateColor(s, "color")
	}
	if d, ok := f.Default.(string); ok {
		e.PlaceHolder = d
	}
	current := ""
	if v, ok := get().(string); ok {
		current = v
		e.SetText(v)
	}
	swatch := canvas.NewRectangle(parseColor(current))
	swatch.SetMinSize(fyne.NewSize(24, 24))
	e.OnChanged = func(s string) {
		current = s
		swatch.FillColor = parseColor(s)
		swatch.Refresh()
		if s == "" {
			set(nil)
		} else {
			set(s)
		}
	}
	pick := widget.NewButton("Pick", func() {
		picker := dialog.NewColorPicker("Pick a color", "", func(c color.Color) {
			if c == nil {
				return
			}
			r32, g32, b32, a32 := c.RGBA()
			if a32 == 0xFFFF {
				e.SetText(fmt.Sprintf("#%02x%02x%02x", r32>>8, g32>>8, b32>>8))
			} else {
				alpha := float64(a32) / 0xFFFF
				e.SetText(fmt.Sprintf("rgba(%d,%d,%d,%.2f)", r32>>8, g32>>8, b32>>8, alpha))
			}
		}, a.win)
		picker.Advanced = true
		if cur := parseColor(current); cur != nil {
			picker.SetColor(cur)
		}
		picker.Show()
	})
	return container.NewBorder(nil, nil, swatch, pick, e)
}

// parseColor parses #rgb/#rrggbb into a color; other syntaxes return nil.
func parseColor(s string) color.Color {
	if !strings.HasPrefix(s, "#") {
		return nil
	}
	hex := s[1:]
	switch len(hex) {
	case 3:
		r, _ := strconv.ParseUint(hex[:1]+hex[:1], 16, 8)
		g, _ := strconv.ParseUint(hex[1:2]+hex[1:2], 16, 8)
		b, _ := strconv.ParseUint(hex[2:3]+hex[2:3], 16, 8)
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xFF}
	case 6:
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xFF}
	default:
		return nil
	}
}

// listTextEditor edits StringList/IntList as one item per line.
func (a *appState) listTextEditor(f *catalog.Field, get func() any, set func(any)) fyne.CanvasObject {
	e := widget.NewMultiLineEntry()
	e.SetMinRowsVisible(3)
	e.TextStyle = fyne.TextStyle{Monospace: true}
	if f.Kind == catalog.IntList {
		e.Validator = func(s string) error {
			for _, line := range strings.Split(s, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if !intRe.MatchString(line) {
					return fmt.Errorf("%q is not an integer", line)
				}
			}
			return nil
		}
	} else if len(f.Enum) > 0 {
		e.Validator = func(s string) error {
			for _, line := range strings.Split(s, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if !contains(f.Enum, line) {
					return fmt.Errorf("%q is not one of %s", line, strings.Join(f.Enum, "|"))
				}
			}
			return nil
		}
	}
	if arr, ok := get().([]any); ok {
		var lines []string
		for _, item := range arr {
			switch x := item.(type) {
			case string:
				lines = append(lines, x)
			case float64:
				lines = append(lines, numStr(x))
			}
		}
		e.SetText(strings.Join(lines, "\n"))
	}
	e.OnChanged = func(s string) {
		var arr []any
		for _, line := range strings.Split(s, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if f.Kind == catalog.IntList {
				if n, err := strconv.ParseFloat(line, 64); err == nil {
					arr = append(arr, n)
					continue
				}
			}
			arr = append(arr, line)
		}
		if len(arr) == 0 {
			set(nil)
		} else {
			set(arr)
		}
	}

	var extra fyne.CanvasObject
	if o := a.rowOptionFor(f); o != nil && (o.Name == "font_dirs" || o.Name == "color_scheme_dirs") {
		addBtn := widget.NewButtonWithIcon("Add folder…", theme.FolderNewIcon(), func() {
			dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
				if err != nil || lu == nil {
					return
				}
				cur := strings.TrimRight(e.Text, "\n")
				if cur == "" {
					e.SetText(lu.Path())
				} else {
					e.SetText(cur + "\n" + lu.Path())
				}
			}, a.win)
		})
		extra = addBtn
	}
	if extra != nil {
		return container.NewVBox(e, extra)
	}
	return e
}

// rowOptionFor finds the mounted option owning field f (top-level rows only).
func (a *appState) rowOptionFor(f *catalog.Field) *catalog.Option {
	for i := range catalog.Options {
		if &catalog.Options[i].Field == f {
			return &catalog.Options[i]
		}
	}
	return nil
}
