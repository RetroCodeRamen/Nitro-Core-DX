package main

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Visual system for the Dev Kit.
//
// Nitro Industrial is the default: gunmetal surfaces, hazard-yellow
// interaction accents, and a recessed editor. Nitro Classic is the light
// companion: warm gray panels, charcoal text, and the same yellow accent.
// Both ignore the operating-system light/dark preference so the chosen
// theme stays stable.
//
// Layout convention: the toolbar owns project, build, run, debug, and view
// groups. One hazard bar sits under that toolbar. Tool panes use a single
// uppercase header. Syntax colors stay inside the editor surface.

const (
	themeIndustrial = "industrial"
	themeClassic    = "classic"
)

type nitroPalette struct {
	background      color.NRGBA
	panel           color.NRGBA
	raised          color.NRGBA
	recessed        color.NRGBA
	hazard          color.NRGBA
	hazardMuted     color.NRGBA
	edge            color.NRGBA
	warning         color.NRGBA
	ok              color.NRGBA
	err             color.NRGBA
	info            color.NRGBA
	text            color.NRGBA
	secondary       color.NRGBA
	disabled        color.NRGBA
	onHazard        color.NRGBA
	hover           color.NRGBA
	pressed         color.NRGBA
	editorBG        color.NRGBA
	editorLine      color.NRGBA
	editorCaret     color.NRGBA
	editorSelect    color.NRGBA
	syntaxString    color.NRGBA
	syntaxNumber    color.NRGBA
	syntaxIdent     color.NRGBA
	syntaxFunction  color.NRGBA
	syntaxNamespace color.NRGBA
	syntaxType      color.NRGBA
	syntaxComment   color.NRGBA
	syntaxKeyword   color.NRGBA
	stripeDark      color.NRGBA
}

func industrialPalette() nitroPalette {
	return nitroPalette{
		background:      color.NRGBA{R: 0x20, G: 0x23, B: 0x26, A: 0xff},
		panel:           color.NRGBA{R: 0x2C, G: 0x30, B: 0x34, A: 0xff},
		raised:          color.NRGBA{R: 0x38, G: 0x3D, B: 0x42, A: 0xff},
		recessed:        color.NRGBA{R: 0x18, G: 0x1B, B: 0x1E, A: 0xff},
		hazard:          color.NRGBA{R: 0xF2, G: 0xC2, B: 0x30, A: 0xff},
		hazardMuted:     color.NRGBA{R: 0xB8, G: 0x95, B: 0x32, A: 0xff},
		edge:            color.NRGBA{R: 0x5C, G: 0x63, B: 0x68, A: 0xff},
		warning:         color.NRGBA{R: 0xD8, G: 0x84, B: 0x32, A: 0xff},
		ok:              color.NRGBA{R: 0x68, G: 0xB8, B: 0x6B, A: 0xff},
		err:             color.NRGBA{R: 0xCE, G: 0x51, B: 0x48, A: 0xff},
		info:            color.NRGBA{R: 0x6B, G: 0x9F, B: 0xC4, A: 0xff},
		text:            color.NRGBA{R: 0xE4, G: 0xE4, B: 0xDE, A: 0xff},
		secondary:       color.NRGBA{R: 0xA7, G: 0xAA, B: 0xA9, A: 0xff},
		disabled:        color.NRGBA{R: 0x73, G: 0x78, B: 0x7B, A: 0xff},
		onHazard:        color.NRGBA{R: 0x18, G: 0x1B, B: 0x1E, A: 0xff},
		hover:           color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x16},
		pressed:         color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x28},
		editorBG:        color.NRGBA{R: 0x18, G: 0x1B, B: 0x1E, A: 0xff},
		editorLine:      color.NRGBA{R: 0x2A, G: 0x2E, B: 0x28, A: 0xff},
		editorCaret:     color.NRGBA{R: 0x4A, G: 0x40, B: 0x18, A: 0xff},
		editorSelect:    color.NRGBA{R: 0x3E, G: 0x36, B: 0x14, A: 0xff},
		syntaxString:    color.NRGBA{R: 0xE7, G: 0xA1, B: 0x5A, A: 0xff},
		syntaxNumber:    color.NRGBA{R: 0xF0, G: 0xE2, B: 0xA8, A: 0xff},
		syntaxIdent:     color.NRGBA{R: 0xD5, G: 0xDD, B: 0xD0, A: 0xff},
		syntaxFunction:  color.NRGBA{R: 0x8F, G: 0xB8, B: 0xD8, A: 0xff},
		syntaxNamespace: color.NRGBA{R: 0xE0, G: 0xC5, B: 0x6A, A: 0xff},
		syntaxType:      color.NRGBA{R: 0xD2, G: 0xC4, B: 0xA2, A: 0xff},
		syntaxComment:   color.NRGBA{R: 0xB4, G: 0xBB, B: 0xB4, A: 0xff},
		syntaxKeyword:   color.NRGBA{R: 0xF2, G: 0xC2, B: 0x30, A: 0xff},
		stripeDark:      color.NRGBA{R: 0x18, G: 0x1B, B: 0x1E, A: 0xff},
	}
}

func classicPalette() nitroPalette {
	return nitroPalette{
		background:      color.NRGBA{R: 0xE4, G: 0xE1, B: 0xD8, A: 0xff},
		panel:           color.NRGBA{R: 0xD5, G: 0xD1, B: 0xC6, A: 0xff},
		raised:          color.NRGBA{R: 0xEE, G: 0xEB, B: 0xE3, A: 0xff},
		recessed:        color.NRGBA{R: 0xF7, G: 0xF4, B: 0xEE, A: 0xff},
		hazard:          color.NRGBA{R: 0xC6, G: 0xA2, B: 0x27, A: 0xff},
		hazardMuted:     color.NRGBA{R: 0x8C, G: 0x72, B: 0x16, A: 0xff},
		edge:            color.NRGBA{R: 0x7A, G: 0x75, B: 0x6C, A: 0xff},
		warning:         color.NRGBA{R: 0xB8, G: 0x6A, B: 0x20, A: 0xff},
		ok:              color.NRGBA{R: 0x2E, G: 0x7D, B: 0x46, A: 0xff},
		err:             color.NRGBA{R: 0xA3, G: 0x3C, B: 0x36, A: 0xff},
		info:            color.NRGBA{R: 0x3D, G: 0x6E, B: 0x93, A: 0xff},
		text:            color.NRGBA{R: 0x1C, G: 0x1E, B: 0x20, A: 0xff},
		secondary:       color.NRGBA{R: 0x4E, G: 0x55, B: 0x54, A: 0xff},
		disabled:        color.NRGBA{R: 0x8A, G: 0x8E, B: 0x8C, A: 0xff},
		onHazard:        color.NRGBA{R: 0x1C, G: 0x1E, B: 0x20, A: 0xff},
		hover:           color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x10},
		pressed:         color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x1C},
		editorBG:        color.NRGBA{R: 0xF7, G: 0xF4, B: 0xEE, A: 0xff},
		editorLine:      color.NRGBA{R: 0xE7, G: 0xE0, B: 0xC8, A: 0xff},
		editorCaret:     color.NRGBA{R: 0xF2, G: 0xC2, B: 0x30, A: 0xff},
		editorSelect:    color.NRGBA{R: 0xE4, G: 0xD4, B: 0x8A, A: 0xff},
		syntaxString:    color.NRGBA{R: 0x8A, G: 0x3C, B: 0x0C, A: 0xff},
		syntaxNumber:    color.NRGBA{R: 0x5C, G: 0x48, B: 0x10, A: 0xff},
		syntaxIdent:     color.NRGBA{R: 0x1E, G: 0x3A, B: 0x2C, A: 0xff},
		syntaxFunction:  color.NRGBA{R: 0x1E, G: 0x4E, B: 0x78, A: 0xff},
		syntaxNamespace: color.NRGBA{R: 0x6E, G: 0x5A, B: 0x10, A: 0xff},
		syntaxType:      color.NRGBA{R: 0x5A, G: 0x40, B: 0x20, A: 0xff},
		syntaxComment:   color.NRGBA{R: 0x3F, G: 0x4C, B: 0x46, A: 0xff},
		syntaxKeyword:   color.NRGBA{R: 0x6A, G: 0x52, B: 0x08, A: 0xff},
		stripeDark:      color.NRGBA{R: 0x2C, G: 0x30, B: 0x34, A: 0xff},
	}
}

var activePalette = industrialPalette()

func normalizeThemeName(name string) string {
	if name == themeClassic {
		return themeClassic
	}
	return themeIndustrial
}

func paletteFor(name string) nitroPalette {
	if normalizeThemeName(name) == themeClassic {
		return classicPalette()
	}
	return industrialPalette()
}

type nitroTheme struct {
	base    fyne.Theme
	pal     nitroPalette
	compact bool
}

func newNitroTheme(choice, density string) fyne.Theme {
	pal := paletteFor(choice)
	activePalette = pal
	return &nitroTheme{
		base:    theme.DefaultTheme(),
		pal:     pal,
		compact: density != "standard",
	}
}

func (t *nitroTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return t.pal.background
	case theme.ColorNameButton, theme.ColorNameDisabledButton:
		return t.pal.raised
	case theme.ColorNameDisabled:
		return t.pal.disabled
	case theme.ColorNameError:
		return t.pal.err
	case theme.ColorNameForeground:
		return t.pal.text
	case theme.ColorNameForegroundOnError, theme.ColorNameForegroundOnSuccess, theme.ColorNameForegroundOnWarning, theme.ColorNameForegroundOnPrimary:
		return t.pal.onHazard
	case theme.ColorNameHeaderBackground, theme.ColorNameMenuBackground:
		return t.pal.panel
	case theme.ColorNameHover:
		return t.pal.hover
	case theme.ColorNameInputBackground, theme.ColorNameOverlayBackground:
		return t.pal.recessed
	case theme.ColorNameInputBorder, theme.ColorNameSeparator:
		return t.pal.edge
	case theme.ColorNamePlaceHolder:
		return t.pal.secondary
	case theme.ColorNamePressed:
		return t.pal.pressed
	case theme.ColorNamePrimary, theme.ColorNameFocus, theme.ColorNameHyperlink:
		return t.pal.hazard
	case theme.ColorNameScrollBar:
		return t.pal.edge
	case theme.ColorNameScrollBarBackground:
		return t.pal.recessed
	case theme.ColorNameSelection:
		return t.pal.editorSelect
	case theme.ColorNameShadow:
		return color.NRGBA{A: 0x66}
	case theme.ColorNameSuccess:
		return t.pal.ok
	case theme.ColorNameWarning:
		return t.pal.warning
	default:
		return t.base.Color(name, variant)
	}
}

func (t *nitroTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}

func (t *nitroTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

func (t *nitroTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameInputRadius, theme.SizeNameSelectionRadius, theme.SizeNameScrollBarRadius, theme.SizeNameWindowButtonRadius:
		return 0
	case theme.SizeNameSeparatorThickness:
		return 1
	}
	if !t.compact {
		switch name {
		case theme.SizeNameText:
			return 13
		case theme.SizeNameCaptionText:
			return 11
		case theme.SizeNamePadding:
			return 3
		case theme.SizeNameInnerPadding:
			return 4
		}
		return t.base.Size(name)
	}
	switch name {
	case theme.SizeNameText:
		return 12
	case theme.SizeNameCaptionText:
		return 10
	case theme.SizeNameHeadingText:
		return 16
	case theme.SizeNameSubHeadingText:
		return 13
	case theme.SizeNamePadding:
		return 2
	case theme.SizeNameInnerPadding:
		return 3
	case theme.SizeNameLineSpacing:
		return 2
	case theme.SizeNameInlineIcon:
		return 14
	case theme.SizeNameScrollBar:
		return 8
	}
	return t.base.Size(name)
}

func panelHeader(title string) fyne.CanvasObject {
	label := widget.NewLabel(title)
	label.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	return label
}

type hazardStripe struct {
	widget.BaseWidget
}

func newHazardStripe() *hazardStripe {
	h := &hazardStripe{}
	h.ExtendBaseWidget(h)
	return h
}

func (h *hazardStripe) CreateRenderer() fyne.WidgetRenderer {
	rect := canvas.NewRectangle(activePalette.hazard)
	return &hazardStripeRenderer{rect: rect, objs: []fyne.CanvasObject{rect}}
}

type hazardStripeRenderer struct {
	rect *canvas.Rectangle
	objs []fyne.CanvasObject
}

func (r *hazardStripeRenderer) Layout(size fyne.Size) {
	r.rect.Resize(size)
	r.rect.Move(fyne.NewPos(0, 0))
}

func (r *hazardStripeRenderer) MinSize() fyne.Size {
	return fyne.NewSize(8, 2)
}

func (r *hazardStripeRenderer) Refresh() {
	r.rect.FillColor = activePalette.hazard
	r.rect.Refresh()
}

func (r *hazardStripeRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *hazardStripeRenderer) Destroy()                     {}

type statusLamp struct {
	widget.BaseWidget
	Fill color.Color
}

func newStatusLamp(fill color.Color) *statusLamp {
	l := &statusLamp{Fill: fill}
	l.ExtendBaseWidget(l)
	return l
}

func (l *statusLamp) SetFill(c color.Color) {
	l.Fill = c
	l.Refresh()
}

func (l *statusLamp) CreateRenderer() fyne.WidgetRenderer {
	circ := canvas.NewCircle(l.Fill)
	return &statusLampRenderer{lamp: l, circ: circ, objs: []fyne.CanvasObject{circ}}
}

type statusLampRenderer struct {
	lamp *statusLamp
	circ *canvas.Circle
	objs []fyne.CanvasObject
}

func (r *statusLampRenderer) Layout(size fyne.Size) {
	inset := float32(2)
	if size.Width < inset*2 || size.Height < inset*2 {
		r.circ.Resize(size)
		r.circ.Move(fyne.NewPos(0, 0))
		return
	}
	r.circ.Resize(fyne.NewSize(size.Width-inset*2, size.Height-inset*2))
	r.circ.Move(fyne.NewPos(inset, inset))
}

func (r *statusLampRenderer) MinSize() fyne.Size { return fyne.NewSize(14, 14) }

func (r *statusLampRenderer) Refresh() {
	r.circ.FillColor = r.lamp.Fill
	r.circ.Refresh()
}

func (r *statusLampRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *statusLampRenderer) Destroy()                     {}

func contrastRatio(a, b color.NRGBA) float64 {
	l1 := relativeLuminance(a)
	l2 := relativeLuminance(b)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

func relativeLuminance(c color.NRGBA) float64 {
	return 0.2126*linearChannel(c.R) + 0.7152*linearChannel(c.G) + 0.0722*linearChannel(c.B)
}

func linearChannel(v uint8) float64 {
	c := float64(v) / 255
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

// readableInk returns a light or dark ink that stays readable on bg.
func mixNRGBA(a, b color.NRGBA, aWeight float64) color.NRGBA {
	bw := 1 - aWeight
	return color.NRGBA{
		R: uint8(float64(a.R)*aWeight + float64(b.R)*bw),
		G: uint8(float64(a.G)*aWeight + float64(b.G)*bw),
		B: uint8(float64(a.B)*aWeight + float64(b.B)*bw),
		A: 0xff,
	}
}

func readableInk(bg color.NRGBA) color.NRGBA {
	dark := color.NRGBA{R: 0x14, G: 0x16, B: 0x18, A: 0xff}
	light := color.NRGBA{R: 0xF6, G: 0xF3, B: 0xEA, A: 0xff}
	if contrastRatio(dark, bg) >= contrastRatio(light, bg) {
		return dark
	}
	return light
}
