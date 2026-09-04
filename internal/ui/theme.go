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
	colBackground   = "#14161b"
	colSurface      = "#1c1f27"
	colSurfaceHigh  = "#242833"
	colBorder       = "#2e3340"
	colPrimary      = "#7aa2f7" // calm blue
	colPrimaryDim   = "#5a7bc4"
	colAccent       = "#bb9af7" // violet
	colSuccess      = "#9ece6a"
	colWarning      = "#e0af68"
	colDanger       = "#f7768e"
	colTextMain     = "#c8d0e0"
	colTextDisabled = "#565f76"
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

// appTheme implements fyne.Theme with the app's palette.
type appTheme struct {
	fyne.Theme
}

func newAppTheme() fyne.Theme {
	return &appTheme{Theme: theme.DarkTheme()}
}

func (t *appTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return mustHex(colBackground)
	case theme.ColorNameInputBackground:
		return mustHex(colSurface)
	case theme.ColorNameMenuBackground:
		return mustHex(colSurface)
	case theme.ColorNameOverlayBackground:
		return mustHex(colSurfaceHigh)
	case theme.ColorNameDisabled:
		return mustHex(colTextDisabled)
	case theme.ColorNameDisabledButton:
		return mustHex(colSurfaceHigh)
	case theme.ColorNameSeparator:
		return mustHex(colBorder)
	case theme.ColorNameInputBorder:
		return mustHex(colBorder)
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
	switch name {
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameInnerPadding:
		return 10
	case theme.SizeNameLineSpacing:
		return 4
	case theme.SizeNameScrollBar:
		return 10
	case theme.SizeNameScrollBarSmall:
		return 4
	case theme.SizeNameInputRadius:
		return 6
	case theme.SizeNameButtonRadius:
		return 6
	case theme.SizeNameSelectionRadius:
		return 8
	case theme.SizeNameCardRadius:
		return 10
	}
	return t.Theme.Size(name)
}
