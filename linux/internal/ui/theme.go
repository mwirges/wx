package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Theme is the desk palette: dark field, cyan marks, steady hazard colors.
type Theme struct{ fyne.Theme }

// NewTheme returns the desk theme.
func NewTheme() fyne.Theme {
	return &Theme{Theme: theme.DefaultTheme()}
}

// Color overrides the tactical palette and leaves the rest of the default theme.
func (t *Theme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground, theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground:
		return color.NRGBA{R: 0x06, G: 0x0A, B: 0x14, A: 0xFF}
	case theme.ColorNameHeaderBackground, theme.ColorNameInputBackground:
		return color.NRGBA{R: 0x0A, G: 0x12, B: 0x22, A: 0xFF}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0x0E, G: 0x1A, B: 0x2D, A: 0xFF}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0xF0, G: 0xF6, B: 0xFF, A: 0xFF}
	case theme.ColorNamePrimary, theme.ColorNameFocus, theme.ColorNameHyperlink:
		return color.NRGBA{R: 0x38, G: 0xE1, B: 0xFF, A: 0xFF}
	case theme.ColorNameDisabled, theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0x94, G: 0xA7, B: 0xC5, A: 0xFF}
	case theme.ColorNameError:
		return color.NRGBA{R: 0xFF, G: 0x3B, B: 0x56, A: 0xFF}
	case theme.ColorNameWarning:
		return color.NRGBA{R: 0xFF, G: 0x9F, B: 0x0A, A: 0xFF}
	case theme.ColorNameSuccess:
		return color.NRGBA{R: 0x30, G: 0xD1, B: 0x58, A: 0xFF}
	case theme.ColorNameSeparator, theme.ColorNameInputBorder:
		return color.NRGBA{R: 0x38, G: 0xE1, B: 0xFF, A: 0x59}
	default:
		return t.Theme.Color(name, theme.VariantDark)
	}
}
