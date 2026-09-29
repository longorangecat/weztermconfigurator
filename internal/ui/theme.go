package ui

import (
	"image/color"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Theme: a custom Fyne theme giving the app a distinct dark "terminal"
// identity instead of the default flat look.

const (
	colBackground  = "#12141a"
	colSurface     = "#1a1e28" // option cards
	colSurfaceHigh = "#272c3d"
	colInput       = "#0d0f14" // inputs are darker than the card so they read as fields
	colBorder      = "#3b4260"
	colInputBorder = "#5d6890"
	colPrimary     = "#82aaff" // calm blue
	colPrimaryDim  = "#4a68b0"
	colAccent      = "#c3a6ff" // violet
	colSuccess     = "#a6e06f"
	colWarning     = "#ffc66d"
	colDanger      = "#ff7a93"
	colTextMain    = "#eef1fa" // typed values, names
	colTextMuted   = "#a3adc8" // help text, defaults (>=6:1 on card)
	colTextHint    = "#7d87a6" // placeholders: dimmer than values, still readable
)

func mustHex(s string) color.Color {
	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return color.NRGBA{R: 0xff, G: 0, B: 0xff, A: 0xff}
	}
	return color.NRGBA{
		R: uint8(v >> 16 & 0xff),
		G: uint8(v >> 8 & 0xff),
		B: uint8(v & 0xff),
		A: 0xff,
	}
}

// appTheme implements fyne.Theme with the app's palette and scale support.
type appTheme struct {
	fyne.Theme
	scale float32
}

func newAppTheme(scale float32) *appTheme {
	if scale <= 0 {
		scale = 1.0
	}
	return &appTheme{Theme: theme.DarkTheme(), scale: scale}
}
func (t *appTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return mustHex(colBackground)
	case theme.ColorNameForeground:
		return mustHex(colTextMain)
	case theme.ColorNameInputBackground:
		return mustHex(colInput)
	case theme.ColorNameMenuBackground:
		return mustHex(colSurface)
	case theme.ColorNameOverlayBackground:
		return mustHex(colSurfaceHigh)
	case theme.ColorNameButton:
		return mustHex(colSurfaceHigh)
	case theme.ColorNameDisabled: // Fyne uses this for LowImportance labels
		return mustHex(colTextMuted)
	case theme.ColorNamePlaceHolder:
		return mustHex(colTextHint)
	case theme.ColorNameDisabledButton:
		return mustHex(colSurface)
	case theme.ColorNameSeparator:
		return mustHex(colBorder)
	case theme.ColorNameInputBorder:
		return mustHex(colInputBorder)
	case theme.ColorNameFocus:
		return mustHex(colPrimaryDim)
	case theme.ColorNameHover:
		return mustHex(colSurfaceHigh)
	case theme.ColorNameSelection:
		return mustHex(colPrimaryDim)
	case theme.ColorNamePrimary:
		return mustHex(colPrimary)
	case theme.ColorNameError:
		return mustHex(colDanger)
	case theme.ColorNameSuccess:
		return mustHex(colSuccess)
	case theme.ColorNameWarning:
		return mustHex(colWarning)
	}
	return t.Theme.Color(name, theme.VariantDark)
}

func (t *appTheme) Size(name fyne.ThemeSizeName) float32 {
	var base float32
	switch name {
	case theme.SizeNamePadding:
		base = 6
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameInnerPadding:
		base = 10
	case theme.SizeNameLineSpacing:
		base = 4
	case theme.SizeNameScrollBar:
		base = 10
	case theme.SizeNameScrollBarSmall:
		base = 4
	case theme.SizeNameInputRadius:
		return 6
	case theme.SizeNameButtonRadius:
		return 6
	case theme.SizeNameSelectionRadius:
		return 8
	case theme.SizeNameCardRadius:
		return 10
	case theme.SizeNameText:
		base = t.Theme.Size(name)
	case theme.SizeNameHeadingText:
		base = t.Theme.Size(name)
	case theme.SizeNameSubHeadingText:
		base = t.Theme.Size(name)
	case theme.SizeNameCaptionText:
		base = t.Theme.Size(name)
	default:
		return t.Theme.Size(name)
	}
	if t.scale > 0 && t.scale != 1.0 {
		return base * t.scale
	}
	return base
}
