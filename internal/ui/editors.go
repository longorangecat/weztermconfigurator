package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"weztermconfigurator/internal/catalog"
)

func (a *appState) structEditor(f *catalog.Field, get func() any, set func(any)) fyne.CanvasObject {
	var rebuild func()
	box := container.NewVBox()
	rebuild = func() {
		box.Objects = nil
		m, _ := get().(map[string]any)
		card := widget.NewCard("", "", container.NewVBox())
		content := card.Content.(*fyne.Container)
		for i := range f.Fields {
			mf := f.Fields[i]
			i, mf := i, mf
			memberGet := func() any {
				if m == nil {
					return nil
				}
				return m[mf.Name]
			}
			memberSet := func(v any) {
				cur, _ := get().(map[string]any)
				if cur == nil {
					cur = map[string]any{}
				}
				if v == nil {
					delete(cur, mf.Name)
				} else {
					cur[mf.Name] = v
				}
				m = cur
				if len(cur) == 0 {
					set(nil)
				} else {
					set(cur)
				}
				a.markDirty()
			}
			label := widget.NewLabelWithStyle(mf.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Monospace: true})
			if mf.Required {
				req := widget.NewLabel("required")
				req.Importance = widget.DangerImportance
				content.Add(container.NewHBox(label, req))
			} else {
				content.Add(label)
			}
			content.Add(a.buildEditor(&f.Fields[i], memberGet, memberSet))
			reset := widget.NewButtonWithIcon("Reset "+mf.Name, theme.ContentUndoIcon(), nil)
			reset.OnTapped = func() {
				memberSet(nil)
				rebuild()
			}
			content.Add(reset)
		}
		box.Objects = append(box.Objects, card)
		box.Refresh()
	}
	rebuild()
	return box
}

// unionEditor: variant select + payload sub-form.
func (a *appState) unionEditor(f *catalog.Field, get func() any, set func(any)) fyne.CanvasObject {
	variantFor := func(v any) string {
		switch x := v.(type) {
		case string:
			return x
		case map[string]any:
			if len(x) == 1 {
				for k := range x {
					return k
				}
			}
		}
		return ""
	}

	sel := widget.NewSelect(f.Enum, nil)
	payloadBox := container.NewStack()

	var vf *catalog.Field
	payloadFor := func(variant string) *catalog.Field {
		for i := range f.Fields {
			if f.Fields[i].Name == variant {
				return &f.Fields[i]
			}
		}
		return nil
	}

	buildPayload := func(variant string) {
		payloadBox.Objects = nil
		vf = payloadFor(variant)
		if vf == nil || len(f.Fields) == 0 && !hasPayloadVariants(f) {
			payloadBox.Refresh()
			return
		}
		if vf == nil || (vf.Scalar && len(vf.Fields) == 0) || isUnitVariant(f, variant) {
			payloadBox.Refresh()
			return
		}
		cur := get()
		var payloadGet func() any
		if m, ok := cur.(map[string]any); ok {
			payloadGet = func() any { return m[variant] }
		} else {
			payloadGet = func() any { return nil }
		}
		payloadSet := func(v any) {
			if v == nil {
				set(variant) // bare variant name
				return
			}
			set(map[string]any{variant: v})
		}
		if vf.Scalar {
			// single scalar payload field rendered directly
			inner := vf.Fields[0]
			payloadBox.Objects = []fyne.CanvasObject{a.buildEditor(&inner,
				func() any { return payloadGet() },
				func(v any) { payloadSet(v) })}
		} else if vf.Tuple {
			payloadBox.Objects = []fyne.CanvasObject{a.tupleEditor(vf, payloadGet, payloadSet)}
		} else {
			payloadBox.Objects = []fyne.CanvasObject{a.structEditor(vf, payloadGet, payloadSet)}
		}
		payloadBox.Refresh()
	}

	sel.OnChanged = func(variant string) {
		if variant == "" {
			return
		}
		buildPayload(variant)
		set(variant)
	}

	sel.Selected = variantFor(get())
	if sel.Selected != "" {
		buildPayload(sel.Selected)
	} else if f.Default != nil {
		if d, ok := f.Default.(string); ok {
			sel.Selected = d
			buildPayload(d)
		}
	}

	return container.NewVBox(sel, payloadBox)
}

func hasPayloadVariants(f *catalog.Field) bool {
	for i := range f.Fields {
		if !f.Fields[i].Scalar && len(f.Fields[i].Fields) > 0 {
			return true
		}
	}
	return false
}

func isUnitVariant(f *catalog.Field, variant string) bool {
	for i := range f.Fields {
		if f.Fields[i].Name == variant {
			vf := &f.Fields[i]
			return vf.Scalar && len(vf.Fields) == 0
		}
	}
	return true // no entry = unit variant
}

// tupleEditor edits a positional (array) payload field by field.
func (a *appState) tupleEditor(f *catalog.Field, get func() any, set func(any)) fyne.CanvasObject {
	box := container.NewVBox()
	rebuild := func() {
		box.Objects = nil
		arr, _ := get().([]any)
		for i := range f.Fields {
			i := i
			mf := f.Fields[i]
			idxGet := func() any {
				if i < len(arr) {
					return arr[i]
				}
				return nil
			}
			idxSet := func(v any) {
				cur, _ := get().([]any)
				if cur == nil {
					cur = make([]any, len(f.Fields))
				}
				if len(cur) <= i {
					grown := make([]any, i+1)
					copy(grown, cur)
					cur = grown
				}
				cur[i] = v
				arr = cur
				set(cur)
				a.markDirty()
			}
			box.Objects = append(box.Objects,
				widget.NewLabelWithStyle(mf.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Monospace: true}),
				a.buildEditor(&f.Fields[i], idxGet, idxSet))
		}
		box.Refresh()
	}
	rebuild()
	return box
}

// fontEditor: primary font + fallbacks + optional foreground color.
func (a *appState) fontEditor(f *catalog.Field, get func() any, set func(any)) fyne.CanvasObject {
	box := container.NewVBox()

	fontValue := func() map[string]any {
		m, _ := get().(map[string]any)
		if m == nil {
			m = map[string]any{}
		}
		if _, ok := m["font"]; !ok {
			m["font"] = []any{map[string]any{"family": ""}}
		}
		return m
	}
	sync := func(m map[string]any) {
		has := false
		if arr, ok := m["font"].([]any); ok {
			for _, e := range arr {
				if em, _ := e.(map[string]any); em != nil {
					if fam, _ := em["family"].(string); fam != "" {
						has = true
					}
				}
			}
		}
		if !has {
			set(nil)
		} else {
			set(m)
		}
		a.markDirty()
	}

	// Primary font attributes.
	primaryGet := func() any {
		m := fontValue()
		arr, _ := m["font"].([]any)
		if len(arr) > 0 {
			return arr[0]
		}
		return nil
	}
	primarySet := func(v any) {
		m := fontValue()
		arr, _ := m["font"].([]any)
		rest := []any{}
		if len(arr) > 1 {
			rest = arr[1:]
		}
		if v == nil {
			m["font"] = rest
		} else {
			m["font"] = append([]any{v}, rest...)
		}
		sync(m)
	}

	// Fallback list.
	fallbackGet := func() any {
		m := fontValue()
		arr, _ := m["font"].([]any)
		if len(arr) > 1 {
			return arr[1:]
		}
		return nil
	}
	fallbackSet := func(v any) {
		m := fontValue()
		arr, _ := m["font"].([]any)
		var head []any
		if len(arr) > 0 {
			head = arr[:1]
		}
		if v == nil {
			m["font"] = head
		} else {
			tail, _ := v.([]any)
			m["font"] = append(head, tail...)
		}
		sync(m)
	}

	content := container.NewVBox(
		widget.NewLabelWithStyle("family / weight / style", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		a.fontAttributesEditor(primaryGet, primarySet),
		widget.NewLabelWithStyle("Fallback fonts", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		a.fallbackListEditor(fallbackGet, fallbackSet),
	)

	if f.FontForeground {
		fgGet := func() any {
			m, _ := get().(map[string]any)
			if m == nil {
				return nil
			}
			return m["foreground"]
		}
		fgSet := func(v any) {
			m, _ := get().(map[string]any)
			if m == nil {
				m = map[string]any{}
			}
			if v == nil {
				delete(m, "foreground")
			} else {
				m["foreground"] = v
			}
			if v != nil {
				set(m)
			}
			a.markDirty()
		}
		colorField := catalog.Field{Name: "foreground", Kind: catalog.Color}
		content.Add(widget.NewLabelWithStyle("foreground", fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Monospace: true}))
		content.Add(a.buildEditor(&colorField, fgGet, fgSet))
	}

	box.Add(content)
	return box
}

// fontAttributesEditor edits one FontAttributes object.
func (a *appState) fontAttributesEditor(get func() any, set func(any)) fyne.CanvasObject {
	fields := make([]catalog.Field, len(catalog.FontAttributesFields))
	copy(fields, catalog.FontAttributesFields)
	stF := catalog.Field{Name: "font", Kind: catalog.Struct, Fields: fields}
	editor := a.structEditor(&stF, get, set)

	// Wrap family entry into a SelectEntry with system fonts when available.
	return editor
}

// fallbackListEditor is a List of FontAttributes objects.
func (a *appState) fallbackListEditor(get func() any, set func(any)) fyne.CanvasObject {
	f := catalog.Field{Name: "fallbacks", Kind: catalog.List, Fields: catalog.FontAttributesFields}
	return a.genericListEditor(&f, get, set)
}

// paletteEditor edits a Palette with presentation helpers.
func (a *appState) paletteEditor(f *catalog.Field, get func() any, set func(any)) fyne.CanvasObject {
	box := container.NewVBox()

	schemeBtn := widget.NewButtonWithIcon("Start from built-in scheme…", theme.ColorPaletteIcon(), func() {
		a.pickScheme(func(name string, colors map[string]any) {
			set(colors)
			a.markDirty()
			box.Refresh()
			_ = name
		})
	})
	box.Add(schemeBtn)

	fields := make([]catalog.Field, len(catalog.PaletteFields))
	copy(fields, catalog.PaletteFields)
	stF := catalog.Field{Name: "colors", Kind: catalog.Struct, Fields: fields}
	box.Add(a.structEditor(&stF, get, set))
	return box
}

// pickScheme shows the scheme picker and calls back with the colors object.
func (a *appState) pickScheme(cb func(name string, colors map[string]any)) {
	filter := widget.NewEntry()
	filter.PlaceHolder = "Filter schemes…"
	list := widget.NewList(
		func() int { return len(catalog.Schemes()) },
		func() fyne.CanvasObject {
			return container.NewHBox(widget.NewLabel(""), container.NewHBox())
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			hb := o.(*fyne.Container).Objects[0].(*fyne.Container)
			nameLbl := o.(*fyne.Container).Objects[0].(*widget.Label)
			sw := hb
			s := catalog.Schemes()[id]
			nameLbl.SetText(s.Name)
			sw.Objects = nil
			if ansi, ok := s.Colors["ansi"].([]any); ok {
				for _, c := range ansi {
					r := canvas.NewRectangle(parseColor(fmt.Sprint(c)))
					r.SetMinSize(fyne.NewSize(10, 10))
					sw.Objects = append(sw.Objects, r)
				}
			}
			sw.Refresh()
		},
	)
	// Filtering via a wrapper list would need index mapping; simplest correct
	// approach: re-scan on filter change using SetOptions-like rebuild.
	var shown []int
	rebuild := func(q string) {
		shown = shown[:0]
		q = strings.ToLower(q)
		for i, s := range catalog.Schemes() {
			if q == "" || strings.Contains(strings.ToLower(s.Name), q) {
				shown = append(shown, i)
			}
		}
		list.Length = func() int { return len(shown) }
		list.UpdateItem = func(id widget.ListItemID, o fyne.CanvasObject) {
			s := catalog.Schemes()[shown[id]]
			nameLbl := o.(*fyne.Container).Objects[0].(*widget.Label)
			sw := o.(*fyne.Container).Objects[1].(*fyne.Container)
			nameLbl.SetText(s.Name)
			sw.Objects = nil
			if ansi, ok := s.Colors["ansi"].([]any); ok {
				for _, c := range ansi {
					r := canvas.NewRectangle(parseColor(fmt.Sprint(c)))
					r.SetMinSize(fyne.NewSize(10, 10))
					sw.Objects = append(sw.Objects, r)
				}
			}
			sw.Refresh()
		}
		list.Refresh()
	}
	filter.OnChanged = rebuild
	rebuild("")

	list.OnSelected = func(id widget.ListItemID) {
		s := catalog.Schemes()[shown[id]]
		cb(s.Name, s.Colors)
	}
	content := container.NewBorder(filter, nil, nil, nil, list)
	d := dialog.NewCustom("Start from built-in scheme", "Close", content, a.win)
	d.Resize(fyne.NewSize(560, 640))
	d.Show()
}

// namedPalettesEditor edits color_schemes: {name: palette}.
func (a *appState) namedPalettesEditor(f *catalog.Field, get func() any, set func(any)) fyne.CanvasObject {
	box := container.NewVBox()
	var rebuild func()
	rebuild = func() {
		box.Objects = nil
		m, _ := get().(map[string]any)
		names := make([]string, 0, len(m))
		for k := range m {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, name := range names {
			name := name
			rowBox := container.NewVBox()
			nameLbl := widget.NewLabelWithStyle(name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Monospace: true})
			palGet := func() any { return m[name] }
			palSet := func(v any) {
				cur, _ := get().(map[string]any)
				if cur == nil {
					cur = map[string]any{}
				}
				if v == nil {
					delete(cur, name)
				} else {
					cur[name] = v
				}
				m = cur
				if len(cur) == 0 {
					set(nil)
				} else {
					set(cur)
				}
				a.markDirty()
			}
			rm := widget.NewButtonWithIcon("Remove scheme", theme.DeleteIcon(), func() {
				palSet(nil)
				rebuild()
			})
			rm.Importance = widget.DangerImportance
			rowBox.Add(container.NewHBox(nameLbl, rm))
			pf := catalog.Field{Name: name, Kind: catalog.Palette, Fields: catalog.PaletteFields}
			rowBox.Add(a.paletteEditor(&pf, palGet, palSet))
			rowBox.Add(widget.NewSeparator())
			box.Objects = append(box.Objects, rowBox)
		}
		add := widget.NewButtonWithIcon("Add scheme", theme.ContentAddIcon(), func() {
			dialog.NewCustom("New scheme name", "Cancel", newNameEntry(func(name string) {
				cur, _ := get().(map[string]any)
				if cur == nil {
					cur = map[string]any{}
				}
				if _, exists := cur[name]; exists {
					return
				}
				cur[name] = map[string]any{}
				set(cur)
				a.markDirty()
				rebuild()
			}, a.win), a.win).Show()
		})
		box.Objects = append(box.Objects, add)
		box.Refresh()
	}
	rebuild()
	return box
}

func newNameEntry(cb func(string), w fyne.Window) fyne.CanvasObject {
	e := widget.NewEntry()
	e.PlaceHolder = "Scheme name"
	btn := widget.NewButton("Create", func() {
		if strings.TrimSpace(e.Text) != "" {
			cb(strings.TrimSpace(e.Text))
		}
	})
	return container.NewVBox(e, btn)
}

// keysEditor edits config.keys: a list of key bindings.
func (a *appState) keysEditor(get func() any, set func(any)) fyne.CanvasObject {
	f := catalog.Field{Name: "keys", Kind: catalog.List, Fields: catalog.KeyBindingFields}
	return a.bindingListEditor(&f, get, set, nil)
}

// keyTablesEditor edits key_tables: {name: [bindings]}.
func (a *appState) keyTablesEditor(get func() any, set func(any)) fyne.CanvasObject {
	box := container.NewVBox()
	var rebuild func()
	rebuild = func() {
		box.Objects = nil
		m, _ := get().(map[string]any)
		names := make([]string, 0, len(m))
		for k := range m {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, name := range names {
			name := name
			rowBox := container.NewVBox()
			nameLbl := widget.NewLabelWithStyle(name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Monospace: true})
			tblGet := func() any { return m[name] }
			tblSet := func(v any) {
				cur, _ := get().(map[string]any)
				if cur == nil {
					cur = map[string]any{}
				}
				if v == nil {
					delete(cur, name)
				} else {
					cur[name] = v
				}
				m = cur
				if len(cur) == 0 {
					set(nil)
				} else {
					set(cur)
				}
				a.markDirty()
			}
			rm := widget.NewButtonWithIcon("Remove table", theme.DeleteIcon(), func() {
				tblSet(nil)
				rebuild()
			})
			rm.Importance = widget.DangerImportance
			rowBox.Add(container.NewHBox(nameLbl, rm))
			rowBox.Add(a.bindingListEditor(
				&catalog.Field{Kind: catalog.List, Fields: catalog.KeyBindingFields},
				tblGet, tblSet, nil))
			rowBox.Add(widget.NewSeparator())
			box.Objects = append(box.Objects, rowBox)
		}
		add := widget.NewButtonWithIcon("Add key table", theme.ContentAddIcon(), func() {
			dialog.NewCustom("New key table name", "Cancel", newNameEntry(func(name string) {
				cur, _ := get().(map[string]any)
				if cur == nil {
					cur = map[string]any{}
				}
				if _, exists := cur[name]; exists {
					return
				}
				cur[name] = []any{}
				set(cur)
				a.markDirty()
				rebuild()
			}, a.win), a.win).Show()
		})
		box.Objects = append(box.Objects, add)
		box.Refresh()
	}
	rebuild()
	return box
}

// mouseEditor edits mouse_bindings.
func (a *appState) mouseEditor(get func() any, set func(any)) fyne.CanvasObject {
	f := catalog.Field{Name: "mouse_bindings", Kind: catalog.List, Fields: catalog.MouseBindingFields}
	return a.bindingListEditor(&f, get, set, nil)
}

// bindingListEditor renders a List of binding objects with add/remove cards.
// itemLabel optionally titles each card.
func (a *appState) bindingListEditor(f *catalog.Field, get func() any, set func(any), itemLabel func(any) string) fyne.CanvasObject {
	box := container.NewVBox()
	var rebuild func()
	rebuild = func() {
		box.Objects = nil
		arr, _ := get().([]any)
		for i, item := range arr {
			i, item := i, item
			card := widget.NewCard("", fmt.Sprintf("Binding %d", i+1), nil)
			content := card.Content.(*fyne.Container)
			itemGet := func() any { return item }
			itemSet := func(v any) {
				cur, _ := get().([]any)
				if cur == nil {
					cur = []any{}
				}
				if v == nil {
					cur = append(cur[:i], cur[i+1:]...)
				} else {
					cur[i] = v
				}
				item = v
				if len(cur) == 0 {
					set(nil)
				} else {
					set(cur)
				}
				a.markDirty()
				rebuild()
			}
			rm := widget.NewButtonWithIcon("Remove", theme.DeleteIcon(), func() { itemSet(nil) })
			rm.Importance = widget.DangerImportance
			content.Add(rm)
			content.Add(a.buildEditor(&catalog.Field{Kind: catalog.Struct, Fields: f.Fields}, itemGet, itemSet))
			box.Objects = append(box.Objects, card)
		}
		add := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() {
			cur, _ := get().([]any)
			if cur == nil {
				cur = []any{}
			}
			newItem := map[string]any{}
			// pre-fill defaults of required members where they exist
			for _, mf := range f.Fields {
				if mf.Required && mf.Default != nil {
					newItem[mf.Name] = mf.Default
				}
			}
			set(append(cur, newItem))
			a.markDirty()
			rebuild()
		})
		box.Objects = append(box.Objects, add)
		box.Refresh()
	}
	rebuild()
	return box
}

// genericListEditor renders a List of struct items (domains, rules, layers).
func (a *appState) genericListEditor(f *catalog.Field, get func() any, set func(any)) fyne.CanvasObject {
	if f.ItemKind == catalog.Action {
		return a.multipleEditor(get, set)
	}
	if len(f.Fields) == 3 && f.Fields[0].Name == "kind" {
		// BackgroundLayer-like; the generic path is fine.
		_ = f.Fields
	}
	if len(f.Fields) > 0 && f.Fields[0].Name == "name" && f.Fields[0].Required {
		return a.namedItemListEditor(f, get, set)
	}
	return a.bindingListEditor(f, get, set, nil)
}

// namedItemListEditor: List items whose first member is a required "name".
func (a *appState) namedItemListEditor(f *catalog.Field, get func() any, set func(any)) fyne.CanvasObject {
	box := container.NewVBox()
	var rebuild func()
	rebuild = func() {
		box.Objects = nil
		arr, _ := get().([]any)
		for i, item := range arr {
			i := i
			card := widget.NewCard("", fmt.Sprintf("Item %d", i+1), nil)
			content := card.Content.(*fyne.Container)
			itemGet := func() any { return item }
			itemSet := func(v any) {
				cur, _ := get().([]any)
				if cur == nil {
					cur = []any{}
				}
				if v == nil {
					cur = append(cur[:i], cur[i+1:]...)
				} else {
					cur[i] = v
				}
				if len(cur) == 0 {
					set(nil)
				} else {
					set(cur)
				}
				a.markDirty()
				rebuild()
			}
			rm := widget.NewButtonWithIcon("Remove", theme.DeleteIcon(), func() { itemSet(nil) })
			rm.Importance = widget.DangerImportance
			content.Add(rm)
			content.Add(a.buildEditor(&catalog.Field{Kind: catalog.Struct, Fields: f.Fields}, itemGet, itemSet))
			box.Objects = append(box.Objects, card)
		}
		add := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() {
			cur, _ := get().([]any)
			if cur == nil {
				cur = []any{}
			}
			newItem := map[string]any{}
			for _, mf := range f.Fields {
				if mf.Required && mf.Default != nil {
					newItem[mf.Name] = mf.Default
				}
			}
			set(append(cur, newItem))
			a.markDirty()
			rebuild()
		})
		box.Objects = append(box.Objects, add)
		box.Refresh()
	}
	rebuild()
	return box
}

// multipleEditor edits the Multiple action arg: a list of nested actions.
func (a *appState) multipleEditor(get func() any, set func(any)) fyne.CanvasObject {
	box := container.NewVBox()
	var rebuild func()
	rebuild = func() {
		box.Objects = nil
		arr, _ := get().([]any)
		for i, item := range arr {
			i, item := i, item
			rowBox := container.NewHBox()
			lbl := widget.NewLabel(fmt.Sprintf("%d", i+1))
			rm := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
				cur, _ := get().([]any)
				cur = append(cur[:i], cur[i+1:]...)
				if len(cur) == 0 {
					set(nil)
				} else {
					set(cur)
				}
				a.markDirty()
				rebuild()
			})
			actGet := func() any { return item }
			actSet := func(v any) {
				cur, _ := get().([]any)
				if cur == nil {
					cur = []any{}
				}
				if i < len(cur) {
					cur[i] = v
				} else {
					cur = append(cur, v)
				}
				set(cur)
				a.markDirty()
			}
			actF := catalog.Field{Name: "action", Kind: catalog.Action}
			rowBox.Add(lbl)
			rowBox.Add(a.buildEditor(&actF, actGet, actSet))
			rowBox.Add(rm)
			box.Objects = append(box.Objects, rowBox)
		}
		add := widget.NewButtonWithIcon("Add action", theme.ContentAddIcon(), func() {
			cur, _ := get().([]any)
			if cur == nil {
				cur = []any{}
			}
			set(append(cur, "Nop"))
			a.markDirty()
			rebuild()
		})
		box.Objects = append(box.Objects, add)
		box.Refresh()
	}
	rebuild()
	return box
}

// actionEditor: select an action + build its arg form.
func (a *appState) actionEditor(get func() any, set func(any)) fyne.CanvasObject {
	names := make([]string, 0, len(catalog.Actions)+1)
	for _, def := range catalog.Actions {
		names = append(names, def.Name)
	}
	names = append(names, "Custom Lua…")

	box := container.NewVBox()
	argBox := container.NewStack()

	currentName := func() string {
		switch x := get().(type) {
		case string:
			return x
		case map[string]any:
			if len(x) == 1 {
				for k := range x {
					return k
				}
			}
		}
		return ""
	}

	buildArg := func(name string) {
		argBox.Objects = nil
		if name == "" {
			argBox.Refresh()
			return
		}
		if name == "Custom Lua…" {
			e := widget.NewMultiLineEntry()
			e.TextStyle = fyne.TextStyle{Monospace: true}
			e.SetMinRowsVisible(6)
			e.PlaceHolder = "wezterm.action_callback(function(window, pane)\n  -- your code here\nend)"
			if m, ok := get().(map[string]any); ok {
				if s, ok := m["Lua"].(string); ok {
					e.SetText(s)
				}
			}
			e.OnChanged = func(s string) {
				if s == "" {
					set(nil)
				} else {
					set(map[string]any{"Lua": s})
				}
			}
			argBox.Objects = []fyne.CanvasObject{e}
			argBox.Refresh()
			return
		}
		def := catalog.FindAction(name)
		if def == nil || def.Arg == nil {
			return
		}
		argGet := func() any {
			if m, ok := get().(map[string]any); ok {
				return m[name]
			}
			return nil
		}
		argSet := func(v any) {
			if v == nil {
				set(name)
				return
			}
			set(map[string]any{name: v})
		}
		af := def.Arg
		switch {
		case af.Kind == catalog.Struct && af.Tuple:
			argBox.Objects = []fyne.CanvasObject{a.tupleEditor(af, argGet, argSet)}
		case af.Kind == catalog.List && af.ItemKind == catalog.Action:
			argBox.Objects = []fyne.CanvasObject{a.multipleEditor(argGet, argSet)}
		case af.Kind == catalog.Union:
			argBox.Objects = []fyne.CanvasObject{a.unionEditor(af, argGet, argSet)}
		case af.Kind == catalog.Struct:
			argBox.Objects = []fyne.CanvasObject{a.structEditor(af, argGet, argSet)}
		default:
			argBox.Objects = []fyne.CanvasObject{a.buildEditor(af, argGet, argSet)}
		}
		argBox.Refresh()
	}

	// fill missing struct arg objects so nested editors can write
	sel := widget.NewSelect(names, func(name string) {
		if name == "" {
			return
		}
		if name != "Custom Lua…" {
			def := catalog.FindAction(name)
			if def != nil && def.Arg != nil && def.Arg.Kind == catalog.Struct && !def.Arg.Tuple {
				if _, ok := get().(map[string]any); !ok {
					set(map[string]any{name: map[string]any{}})
				}
			}
		}
		buildArg(name)
		a.markDirty()
	})
	sel.PlaceHolder = "(choose an action)"
	if n := currentName(); n != "" {
		sel.Selected = n
		buildArg(n)
	}

	box.Add(sel)
	box.Add(argBox)
	return box
}

var _ = strconv.Itoa // keep strconv if unused on some paths
