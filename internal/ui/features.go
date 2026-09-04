package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"weztermconfigurator/internal/catalog"
)

// featuresEditor renders one card per catalog feature: an enable checkbox
// plus a parameter form (shown when enabled).
func (a *appState) featuresEditor() []fyne.CanvasObject {
	var out []fyne.CanvasObject

	for i := range catalog.Features {
		f := &catalog.Features[i]
		params := a.st.FeatureParams(f.ID)
		on := f.IsOn(params)

		cardBody := container.NewVBox()
		doc := widget.NewLabel(f.Doc)
		doc.Wrapping = fyne.TextWrapWord
		doc.Importance = widget.LowImportance
		cardBody.Add(doc)
		src := widget.NewLabelWithStyle("source: "+f.Source, fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
		src.Importance = widget.LowImportance
		cardBody.Add(src)

		// Param widgets are rebuilt on toggle so the form reflects the value map.
		rebuildPage := func() { a.rebuildPage() }

		enable := widget.NewCheck("Enabled", func(on bool) {
			if on {
				// Materialize defaults so IsOn() sees the feature as enabled.
				for _, prm := range f.Params {
					if prm.Default != "" && params[prm.Name] == "" {
						params[prm.Name] = prm.Default
					}
				}
				params["__on"] = "1"
			} else {
				a.st.DeleteFeature(f.ID)
			}
			a.markDirty()
			rebuildPage()
		})
		enable.Checked = on

		if on {
			for _, prm := range f.Params {
				p := prm
				switch p.Kind {
				case "bool":
					c := widget.NewCheck(p.Label, func(v bool) {
						if v {
							params[p.Name] = "1"
						} else {
							params[p.Name] = ""
						}
						a.markDirty()
					})
					c.Checked = params[p.Name] != ""
					cardBody.Add(c)
				case "select":
					sel := widget.NewSelect(p.Options, func(v string) {
						params[p.Name] = v
						a.markDirty()
					})
					sel.PlaceHolder = "(default: " + p.Default + ")"
					if params[p.Name] != "" {
						sel.Selected = params[p.Name]
					}
					cardBody.Add(container.NewBorder(nil, nil, widget.NewLabel(p.Label), nil, sel))
				case "text":
					e := widget.NewMultiLineEntry()
					e.TextStyle = fyne.TextStyle{Monospace: true}
					e.SetMinRowsVisible(3)
					e.SetPlaceHolder(p.Label)
					e.SetText(params[p.Name])
					e.OnChanged = func(v string) { params[p.Name] = v; a.markDirty() }
					cardBody.Add(container.NewVBox(widget.NewLabel(p.Label), e))
				default: // string
					e := widget.NewEntry()
					e.SetPlaceHolder(p.Label + " (default: " + p.Default + ")")
					e.SetText(params[p.Name])
					e.OnChanged = func(v string) { params[p.Name] = strings.TrimSpace(v); a.markDirty() }
					cardBody.Add(container.NewBorder(nil, nil, widget.NewLabel(p.Label), nil, e))
				}
			}
		}
		cardBody.Add(enable)
		card := widget.NewCard("", f.Name+" ["+onT(on)+"]", cardBody)
		out = append(out, card)
	}
	return out
}

func onT(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
