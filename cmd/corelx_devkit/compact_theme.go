package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type compactTheme struct {
	base fyne.Theme
}

func newCompactTheme() fyne.Theme {
	return &compactTheme{base: theme.DefaultTheme()}
}

func (t *compactTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return t.base.Color(name, variant)
}

func (t *compactTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}

func (t *compactTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

func (t *compactTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 12
	case theme.SizeNameCaptionText:
		return 10
	case theme.SizeNameHeadingText:
		return 20
	case theme.SizeNameSubHeadingText:
		return 15
	case theme.SizeNamePadding:
		return 2
	case theme.SizeNameInnerPadding:
		return 5
	case theme.SizeNameLineSpacing:
		return 2
	case theme.SizeNameInlineIcon:
		return 16
	case theme.SizeNameScrollBar:
		return 10
	}
	return t.base.Size(name)
}

func newStandardTheme() fyne.Theme {
	return theme.DefaultTheme()
}
