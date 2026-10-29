package main

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

func TestNitroIndustrialUsesHazardPrimary(t *testing.T) {
	th := newNitroTheme(themeIndustrial, "compact")
	got := nrgba(th.Color(theme.ColorNamePrimary, theme.VariantLight))
	if got.R < 0xE0 || got.G < 0xB0 || got.B > 0x60 {
		t.Fatalf("industrial primary = #%02X%02X%02X, want hazard yellow", got.R, got.G, got.B)
	}
	bg := nrgba(th.Color(theme.ColorNameBackground, theme.VariantLight))
	if bg.R > 0x40 {
		t.Fatalf("industrial background stayed light: #%02X%02X%02X", bg.R, bg.G, bg.B)
	}
}

func TestNitroClassicStaysLightRegardlessOfVariant(t *testing.T) {
	th := newNitroTheme(themeClassic, "standard")
	bg := nrgba(th.Color(theme.ColorNameBackground, theme.VariantDark))
	if bg.R < 0xD0 || bg.G < 0xD0 {
		t.Fatalf("classic background = #%02X%02X%02X, want warm light gray", bg.R, bg.G, bg.B)
	}
	text := nrgba(th.Color(theme.ColorNameForeground, theme.VariantDark))
	if text.R > 0x40 {
		t.Fatalf("classic text = #%02X%02X%02X, want charcoal", text.R, text.G, text.B)
	}
}

func TestNormalizeThemeName(t *testing.T) {
	if normalizeThemeName("classic") != themeClassic {
		t.Fatal("classic should be preserved")
	}
	if normalizeThemeName("") != themeIndustrial || normalizeThemeName("light") != themeIndustrial {
		t.Fatal("unknown themes should fall back to industrial")
	}
}

func nrgba(c color.Color) color.NRGBA {
	r, g, b, a := c.RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

func TestEditorSyntaxMeetsReadingContrast(t *testing.T) {
	check := func(name string, pal nitroPalette) {
		pairs := []struct {
			label string
			fg    color.NRGBA
		}{
			{"comment", pal.syntaxComment},
			{"keyword", pal.syntaxKeyword},
			{"string", pal.syntaxString},
			{"number", pal.syntaxNumber},
			{"ident", pal.syntaxIdent},
			{"function", pal.syntaxFunction},
		}
		for _, pair := range pairs {
			if ratio := contrastRatio(pair.fg, pal.editorBG); ratio < 4.5 {
				t.Errorf("%s %s contrast %.2f on editor background", name, pair.label, ratio)
			}
			if ratio := contrastRatio(pair.fg, pal.editorLine); ratio < 4.5 {
				t.Errorf("%s %s contrast %.2f on active line", name, pair.label, ratio)
			}
		}
	}
	check("industrial", industrialPalette())
	check("classic", classicPalette())
}

func TestPaletteChipInkStaysReadable(t *testing.T) {
	samples := []color.NRGBA{
		{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		{R: 0xFF, G: 0x00, B: 0x00, A: 0xff},
		{R: 0x00, G: 0x00, B: 0xFF, A: 0xff},
		{R: 0xF2, G: 0xC2, B: 0x30, A: 0xff},
		{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xff},
	}
	for _, pal := range []nitroPalette{industrialPalette(), classicPalette()} {
		for _, sample := range samples {
			chip := mixNRGBA(sample, pal.editorBG, 0.82)
			if contrastRatio(readableInk(chip), chip) < 4.5 {
				t.Errorf("chip ink contrast %.2f for #%02X%02X%02X", contrastRatio(readableInk(chip), chip), sample.R, sample.G, sample.B)
			}
		}
	}
}

func TestCompactThemeStillBuilds(t *testing.T) {
	th := newCompactTheme()
	if th.Size(theme.SizeNameText) != 12 {
		t.Fatalf("compact text size = %v", th.Size(theme.SizeNameText))
	}
	var _ fyne.Theme = th
}
