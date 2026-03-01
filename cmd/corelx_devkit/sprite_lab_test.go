package main

import (
	"image/color"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

func TestPackSpriteLabPixelsNibbleOrder(t *testing.T) {
	w, h := 8, 8
	pixels := make([]uint8, w*h)
	for i := range pixels {
		pixels[i] = uint8(i % 16)
	}
	packed, err := packSpriteLabPixels(pixels, w, h)
	if err != nil {
		t.Fatalf("pack pixels: %v", err)
	}
	if len(packed) != 32 {
		t.Fatalf("expected 32 packed bytes, got %d", len(packed))
	}
	if packed[0] != 0x10 {
		t.Fatalf("expected first packed byte 0x10, got 0x%02X", packed[0])
	}
	if packed[1] != 0x32 {
		t.Fatalf("expected second packed byte 0x32, got 0x%02X", packed[1])
	}
}

func TestSpriteLabAssetRoundTripWithPalettes(t *testing.T) {
	w, h := 24, 24
	pixels := make([]uint8, w*h)
	for i := range pixels {
		pixels[i] = uint8((i / 2) % 16)
	}
	palettes := defaultSpriteLabPaletteData()
	palettes[1*spriteLabColorsPerBank+2] = 0x7FFF

	data, err := marshalSpriteLabAsset("Player-Ship", pixels, w, h, 1, palettes)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	asset, err := unmarshalSpriteLabAsset(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if asset.Format != spriteLabFormatV1 {
		t.Fatalf("unexpected format: %q", asset.Format)
	}
	if asset.Name != "Player_Ship" {
		t.Fatalf("expected sanitized name Player_Ship, got %q", asset.Name)
	}
	if asset.Width != w || asset.Height != h {
		t.Fatalf("expected %dx%d, got %dx%d", w, h, asset.Width, asset.Height)
	}
	if asset.PaletteBank != 1 {
		t.Fatalf("expected palette bank 1, got %d", asset.PaletteBank)
	}
	if len(asset.Pixels) != len(pixels) {
		t.Fatalf("pixel length mismatch: %d vs %d", len(asset.Pixels), len(pixels))
	}
	for i := range pixels {
		if asset.Pixels[i] != pixels[i] {
			t.Fatalf("pixel mismatch at %d: %d vs %d", i, asset.Pixels[i], pixels[i])
		}
	}
	if len(asset.Palettes) != spriteLabPaletteCount {
		t.Fatalf("expected %d palettes, got %d", spriteLabPaletteCount, len(asset.Palettes))
	}
	if asset.Palettes[1*spriteLabColorsPerBank+2] != 0x7FFF {
		t.Fatalf("expected edited palette entry 0x7FFF, got 0x%04X", asset.Palettes[1*spriteLabColorsPerBank+2])
	}
}

func TestSpriteLabAssetLegacyDefaults(t *testing.T) {
	legacy := `{
  "format": "clxsprite-v1",
  "name": "Legacy",
  "width": 8,
  "height": 8,
  "pixels": [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]
}`
	asset, err := unmarshalSpriteLabAsset([]byte(legacy))
	if err != nil {
		t.Fatalf("unmarshal legacy: %v", err)
	}
	if asset.PaletteBank != 0 {
		t.Fatalf("expected default palette bank 0, got %d", asset.PaletteBank)
	}
	if len(asset.Palettes) != spriteLabPaletteCount {
		t.Fatalf("expected default palette length %d, got %d", spriteLabPaletteCount, len(asset.Palettes))
	}
}

func TestSpriteLabCoreLXSnippetTypeSelection(t *testing.T) {
	pixels8 := make([]uint8, 8*8)
	s8, err := spriteLabCoreLXAssetSnippet("ship8", pixels8, 8, 8)
	if err != nil {
		t.Fatalf("snippet 8x8: %v", err)
	}
	if !strings.Contains(s8, ": tiles8 hex") {
		t.Fatalf("expected tiles8 snippet, got: %q", s8)
	}

	pixels16 := make([]uint8, 16*16)
	s16, err := spriteLabCoreLXAssetSnippet("ship16", pixels16, 16, 16)
	if err != nil {
		t.Fatalf("snippet 16x16: %v", err)
	}
	if !strings.Contains(s16, ": tiles16 hex") {
		t.Fatalf("expected tiles16 snippet, got: %q", s16)
	}

	pixels24 := make([]uint8, 24*24)
	s24, err := spriteLabCoreLXAssetSnippet("ship24", pixels24, 24, 24)
	if err != nil {
		t.Fatalf("snippet 24x24: %v", err)
	}
	if !strings.Contains(s24, ": tileset hex") {
		t.Fatalf("expected tileset snippet, got: %q", s24)
	}
}

func TestSpriteLabCoreLXSnippetShape(t *testing.T) {
	w, h := 8, 8
	pixels := make([]uint8, w*h)
	for i := range pixels {
		pixels[i] = 0x0F
	}
	snippet, err := spriteLabCoreLXAssetSnippet("9ship", pixels, w, h)
	if err != nil {
		t.Fatalf("snippet: %v", err)
	}
	if !strings.Contains(snippet, "asset A_9ship: tiles8 hex\n") {
		t.Fatalf("unexpected header: %q", snippet)
	}
	lines := strings.Split(snippet, "\n")
	if len(lines) != 10 {
		t.Fatalf("expected 10 lines (comment + header + 8 rows), got %d", len(lines))
	}
	for i := 2; i < len(lines); i++ {
		fields := strings.Fields(lines[i])
		if len(fields) != 4 {
			t.Fatalf("line %d expected 4 hex bytes, got %d (%q)", i, len(fields), lines[i])
		}
		for _, f := range fields {
			if f != "FF" {
				t.Fatalf("expected FF bytes, got %q", f)
			}
		}
	}
}

func TestSpriteLabPaletteInitSnippetShape(t *testing.T) {
	palettes := defaultSpriteLabPaletteData()
	snippet, err := spriteLabPaletteInitSnippet(2, palettes)
	if err != nil {
		t.Fatalf("palette snippet: %v", err)
	}
	lines := strings.Split(snippet, "\n")
	if len(lines) != 17 {
		t.Fatalf("expected 17 lines (header + 16 writes), got %d", len(lines))
	}
	if !strings.HasPrefix(lines[0], "// Sprite Lab palette bank 2") {
		t.Fatalf("unexpected header: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "gfx.set_palette(2, 0, 0x") {
		t.Fatalf("unexpected first write: %q", lines[1])
	}
}

func TestRGB555EncodeDecodeRoundTrip(t *testing.T) {
	cases := []struct {
		r uint8
		g uint8
		b uint8
	}{
		{0, 0, 0},
		{31, 31, 31},
		{3, 15, 27},
		{30, 1, 17},
	}
	for _, tc := range cases {
		v := encodeRGB555(tc.r, tc.g, tc.b)
		r, g, b := decodeRGB555(v)
		if r != tc.r || g != tc.g || b != tc.b {
			t.Fatalf("round-trip mismatch for (%d,%d,%d): got (%d,%d,%d)", tc.r, tc.g, tc.b, r, g, b)
		}
	}
}

func TestRenderSpriteLabImageTransparentIndexPattern(t *testing.T) {
	w, h := 8, 8
	pixels := make([]uint8, w*h) // all index 0
	palettes := defaultSpriteLabPaletteData()

	imgSolid := renderSpriteLabImage(pixels, palettes, 0, w, h, 8, -1, -1, false, false)
	imgTransparent := renderSpriteLabImage(pixels, palettes, 0, w, h, 8, -1, -1, false, true)

	solid := color.NRGBAModel.Convert(imgSolid.At(1, 1)).(color.NRGBA)
	trans := color.NRGBAModel.Convert(imgTransparent.At(1, 1)).(color.NRGBA)

	if solid == trans {
		t.Fatalf("expected transparent-index preview to differ from solid color preview")
	}
}

func TestSpriteLabDimensionValidation(t *testing.T) {
	if !isValidSpriteDimension(24) {
		t.Fatalf("24 should be a valid sprite dimension")
	}
	if isValidSpriteDimension(12) {
		t.Fatalf("12 should not be valid (must be step of 8)")
	}
}

func TestSpriteLabDisplaySizeUniformPixels(t *testing.T) {
	cases := []struct {
		name   string
		w, h   int
		fn     func(int, int) fyne.Size
		expectW, expectH float32
	}{
		{"8x8 editor", 8, 8, spriteLabEditorDisplaySize, 384, 384},
		{"16x16 editor", 16, 16, spriteLabEditorDisplaySize, 384, 384},
		{"24x24 editor", 24, 24, spriteLabEditorDisplaySize, 384, 384},
		{"64x64 editor", 64, 64, spriteLabEditorDisplaySize, 384, 384},
		{"64x8 editor", 64, 8, spriteLabEditorDisplaySize, 384, 48},
		{"8x8 preview", 8, 8, spriteLabPreviewDisplaySize, 192, 192},
		{"64x8 preview", 64, 8, spriteLabPreviewDisplaySize, 192, 24},
	}
	for _, tc := range cases {
		sz := tc.fn(tc.w, tc.h)
		if sz.Width != tc.expectW || sz.Height != tc.expectH {
			t.Errorf("%s: expected %.0fx%.0f, got %.0fx%.0f", tc.name, tc.expectW, tc.expectH, sz.Width, sz.Height)
		}
	}
}
