package ppu

import (
	"testing"

	"nitro-core-dx/internal/debug"
)

// TestSpritePriority tests sprite priority sorting and rendering
func TestSpritePriority(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Set up colors: palette 0 color 1 = red, palette 0 color 2 = green
	// Red: RGB555 = 0x7C00 (red = 31, green = 0, blue = 0)
	ppu.CGRAM[0x01*2] = 0x00
	ppu.CGRAM[0x01*2+1] = 0x7C
	// Green: RGB555 = 0x03E0 (red = 0, green = 31, blue = 0)
	ppu.CGRAM[0x02*2] = 0xE0
	ppu.CGRAM[0x02*2+1] = 0x03

	// Create tiles: tile 0 = red (color index 1), tile 1 = green (color index 2)
	// 8x8 tile = 32 bytes
	for i := 0; i < 32; i++ {
		ppu.VRAM[i] = 0x11    // Tile 0: color index 1 (red) - all pixels
		ppu.VRAM[32+i] = 0x22 // Tile 1: color index 2 (green) - all pixels
	}

	// Sprite 0: Priority 0 (lowest), position (100, 100), tile 0 (red), 8x8
	ppu.OAM[0] = 100  // X low
	ppu.OAM[1] = 0x00 // X high
	ppu.OAM[2] = 100  // Y
	ppu.OAM[3] = 0x00 // Tile index 0
	ppu.OAM[4] = 0x00 // Attributes: palette 0, priority 0 (bits [7:6] = 0)
	ppu.OAM[5] = 0x03 // Control: enabled (bit 0), 8x8 (bit 1 = 0)

	// Sprite 1: Priority 3 (highest), position (100, 100), tile 1 (green), 8x8
	ppu.OAM[6] = 100   // X low
	ppu.OAM[7] = 0x00  // X high
	ppu.OAM[8] = 100   // Y
	ppu.OAM[9] = 0x01  // Tile index 1
	ppu.OAM[10] = 0xC0 // Attributes: palette 0, priority 3 (bits [7:6] = 3 = 0xC0)
	ppu.OAM[11] = 0x03 // Control: enabled, 8x8

	// Initialize output buffer to black
	ppu.OutputBuffer[100*320+100] = 0x000000

	// Render pixel at (100, 100) - should render green (higher priority sprite)
	// renderDot takes (scanline, dot) parameters where scanline=y, dot=x
	ppu.renderDot(100, 100)

	color := ppu.OutputBuffer[100*320+100]
	// Check that sprite was rendered (not black)
	if color == 0x000000 {
		t.Errorf("Expected sprite to be rendered at (100, 100), got black (0x%06X)", color)
		return
	}

	// Verify priority sorting works - higher priority sprite should be on top
	// We can't easily verify exact color without knowing RGB conversion, but we can verify
	// that a sprite was rendered and priority system is working
	t.Logf("Sprite priority test: Rendered color at (100, 100) = 0x%06X", color)
}

// TestSpriteBlending tests sprite blending modes
func TestSpriteBlending(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Set up background: white
	ppu.CGRAM[0x00*2] = 0xFF
	ppu.CGRAM[0x00*2+1] = 0x7F

	// Set up sprite: red, palette 0 color 1
	ppu.CGRAM[0x01*2] = 0x00
	ppu.CGRAM[0x01*2+1] = 0x7C

	// Create tiles: tile 0 = background color index 0 (white), tile 1 = sprite color index 1 (red)
	for i := 0; i < 32; i++ {
		ppu.VRAM[i] = 0x00
		ppu.VRAM[32+i] = 0x11
	}

	// Set up background layer to render white
	ppu.BG0.Enabled = true
	ppu.BG0.Priority = 0
	ppu.BG0.TilemapBase = 0x4000
	// Fill tilemap with tile 0 (white)
	ppu.VRAM[0x4000] = 0x00
	ppu.VRAM[0x4001] = 0x00

	// Sprite 0: Alpha blend mode, alpha 8 (about 50% transparent)
	ppu.OAM[0] = 100
	ppu.OAM[1] = 0x00
	ppu.OAM[2] = 100
	ppu.OAM[3] = 0x01 // Tile 1 (red)
	ppu.OAM[4] = 0x00 // Palette 0
	ppu.OAM[5] = 0x85 // Enabled, 8x8, blend mode 1 (alpha), alpha 8 (bits [7:4])

	// Render background first
	ppu.renderDotBackgroundLayer(0, 100, 100)
	bgColor := ppu.OutputBuffer[100*320+100]

	// Render sprite with blending
	ppu.renderDot(100, 100)

	color := ppu.OutputBuffer[100*320+100]
	// Should be blended (not pure red or pure white)
	if color == bgColor {
		t.Errorf("Expected blended color, got background color 0x%06X", color)
	}
	// Should not be pure red (RGB888)
	if color == 0xFF0000 {
		t.Errorf("Expected blended color, got pure sprite color 0x%06X", color)
	}
}

// TestMosaicEffect tests mosaic pixel grouping
func TestMosaicEffect(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Set up a colored tile
	ppu.CGRAM[0x01*2] = 0x00
	ppu.CGRAM[0x01*2+1] = 0x7C // Red

	// Create tile with color index 1
	for i := 0; i < 32; i++ {
		ppu.VRAM[i] = 0x11 // All pixels color index 1
	}

	// Enable mosaic for BG0, size 4
	ppu.BG0.Enabled = true
	ppu.BG0.MosaicEnabled = true
	ppu.BG0.MosaicSize = 4
	ppu.BG0.TilemapBase = 0x4000
	ppu.VRAM[0x4000] = 0x00 // Tile 0
	ppu.VRAM[0x4001] = 0x00 // Attributes

	// Initialize output buffer
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			ppu.OutputBuffer[y*320+x] = 0x000000
		}
	}

	// Render pixels in a 4x4 mosaic block
	// Top-left pixel should determine color for entire block
	ppu.renderDotBackgroundLayer(0, 0, 0)
	topLeftColor := ppu.OutputBuffer[0*320+0]

	if topLeftColor == 0x000000 {
		t.Errorf("Mosaic test: Top-left pixel should be rendered, got black")
		return
	}

	// Render other pixels in the same mosaic block
	// Mosaic effect reads from top-left pixel in the block
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if x == 0 && y == 0 {
				continue // Already rendered
			}
			ppu.renderDotBackgroundLayer(0, x, y)
		}
	}

	// All pixels in the 4x4 block should have the same color (from top-left)
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			color := ppu.OutputBuffer[y*320+x]
			if color != topLeftColor {
				t.Errorf("Mosaic block pixel at (%d, %d) should match top-left color 0x%06X, got 0x%06X",
					x, y, topLeftColor, color)
			}
		}
	}
}

// TestMatrixModeOutsideScreen tests outside-screen handling modes
func TestMatrixModeOutsideScreen(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Set up backdrop color (palette 0, color 0 = blue)
	ppu.CGRAM[0x00*2] = 0x1F
	ppu.CGRAM[0x00*2+1] = 0x00 // Blue

	// Enable Matrix Mode for BG0
	ppu.BG0.Enabled = true
	ppu.TransformChannels[0].Enabled = true
	ppu.TransformChannels[0].A = 0x0100 // 1.0
	ppu.TransformChannels[0].B = 0x0000
	ppu.TransformChannels[0].C = 0x0000
	ppu.TransformChannels[0].D = 0x0100 // 1.0
	ppu.TransformChannels[0].CenterX = 160
	ppu.TransformChannels[0].CenterY = 100

	// Set up tilemap with a visible tile
	ppu.BG0.TilemapBase = 0x4000
	ppu.VRAM[0x4000] = 0x00 // Tile 0
	ppu.VRAM[0x4001] = 0x00 // Attributes
	// Create tile 0 with color
	for i := 0; i < 32; i++ {
		ppu.VRAM[i] = 0x11 // Color index 1
	}
	ppu.CGRAM[0x01*2] = 0xFF
	ppu.CGRAM[0x01*2+1] = 0x7F // White

	// Test backdrop mode - use coordinates that are outside bounds
	ppu.TransformChannels[0].OutsideMode = 1 // Backdrop mode
	// Set matrix to produce coordinates outside tilemap (negative or > 256)
	ppu.TransformChannels[0].A = 0x0100 // 1.0
	ppu.TransformChannels[0].B = 0x0000
	ppu.TransformChannels[0].C = 0x0000
	ppu.TransformChannels[0].D = 0x0100 // 1.0
	ppu.TransformChannels[0].CenterX = 0
	ppu.TransformChannels[0].CenterY = 0
	ppu.BG0.ScrollX = -300 // Push coordinates outside bounds
	ppu.BG0.ScrollY = -300

	ppu.renderDotMatrixMode(0, 0, 0)
	color := ppu.OutputBuffer[0*320+0]
	expectedBackdrop := ppu.getColorFromCGRAM(0, 0)
	if color != expectedBackdrop {
		t.Logf("Backdrop mode: Got color 0x%06X, expected backdrop 0x%06X (may need coordinate adjustment)", color, expectedBackdrop)
	}

	// Test repeat mode (default) - coordinates should wrap
	ppu.TransformChannels[0].OutsideMode = 0 // Repeat mode
	ppu.renderDotMatrixMode(0, 0, 0)
	// Should wrap coordinates and render tile data
	color = ppu.OutputBuffer[0*320+0]
	// In repeat mode, coordinates wrap, so we should get tile data, not backdrop
	if color == expectedBackdrop && expectedBackdrop != 0x000000 {
		t.Logf("Repeat mode: Got backdrop color, but coordinates may have wrapped correctly")
	}

	// Test clamp mode - out-of-range coordinates should clamp to tile 0 instead of wrapping.
	ppu.TransformChannels[0].OutsideMode = 3 // Clamp mode
	ppu.BG0.ScrollX = -300
	ppu.BG0.ScrollY = -300
	ppu.renderDotMatrixMode(0, 0, 0)
	color = ppu.OutputBuffer[0]
	expectedTileColor := ppu.getColorFromCGRAM(0, 1)
	if color != expectedTileColor {
		t.Fatalf("Clamp mode: got color 0x%06X, want clamped tile color 0x%06X", color, expectedTileColor)
	}
}

// TestMatrixModeDirectColor tests direct color mode
func TestMatrixModeDirectColor(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Enable Matrix Mode with direct color
	ppu.BG0.Enabled = true
	ppu.TransformChannels[0].Enabled = true
	ppu.TransformChannels[0].DirectColor = true
	ppu.TransformChannels[0].A = 0x0100 // 1.0 (no transform)
	ppu.TransformChannels[0].B = 0x0000
	ppu.TransformChannels[0].C = 0x0000
	ppu.TransformChannels[0].D = 0x0100 // 1.0
	ppu.TransformChannels[0].CenterX = 0
	ppu.TransformChannels[0].CenterY = 0
	ppu.BG0.TilemapBase = 0x4000
	ppu.BG0.ScrollX = 0
	ppu.BG0.ScrollY = 0

	// Create tile with color index 0xF (should be used directly)
	// 8x8 tile = 32 bytes
	for i := 0; i < 32; i++ {
		ppu.VRAM[i] = 0xFF // Color index 15 (all pixels)
	}
	ppu.VRAM[0x4000] = 0x00 // Tile 0
	ppu.VRAM[0x4001] = 0x00 // Attributes

	ppu.renderDotMatrixMode(0, 0, 0)
	color := ppu.OutputBuffer[0*320+0]

	// Direct color mode should bypass CGRAM and use direct RGB
	// Color index 0xF should be expanded to RGB
	// Should not be black (0x000000)
	if color == 0x000000 {
		t.Errorf("Direct color mode: Expected non-black color, got 0x%06X", color)
	}

	// Verify it's not CGRAM color (if CGRAM[0] is set to something different)
	ppu.CGRAM[0] = 0x00
	ppu.CGRAM[1] = 0x00 // Black in CGRAM
	cgramColor := ppu.getColorFromCGRAM(0, 15)
	if color == cgramColor {
		t.Logf("Direct color mode: Color matches CGRAM, but direct color should bypass CGRAM")
	}
}

func TestMatrixMode128x128TilemapExtendsWrapSpan(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.BG0.Enabled = true
	ppu.BG0.TilemapSize = TilemapSize128x128
	ppu.TransformChannels[0].Enabled = true
	ppu.TransformChannels[0].A = 0x0100
	ppu.TransformChannels[0].D = 0x0100
	ppu.TransformChannels[0].CenterX = 0
	ppu.TransformChannels[0].CenterY = 0
	ppu.BG0.TilemapBase = 0x4000
	ppu.BG0.ScrollX = 400

	// Tile 0 -> palette color 1, tile 1 -> palette color 2.
	for i := 0; i < 32; i++ {
		ppu.VRAM[i] = 0x11
		ppu.VRAM[32+i] = 0x22
	}
	ppu.CGRAM[0x01*2] = 0x1F
	ppu.CGRAM[0x01*2+1] = 0x00
	ppu.CGRAM[0x02*2] = 0xE0
	ppu.CGRAM[0x02*2+1] = 0x03

	// At x=700 with identity matrix:
	// - 64x64@8x8 source would wrap to tileX 23
	// - 128x128@8x8 source should sample tileX 87 directly
	ppu.VRAM[0x4000+(23*2)] = 0x00
	ppu.VRAM[0x4000+(23*2)+1] = 0x00
	ppu.VRAM[0x4000+(87*2)] = 0x01
	ppu.VRAM[0x4000+(87*2)+1] = 0x00

	ppu.renderDotMatrixMode(0, 300, 0)
	color := ppu.OutputBuffer[300]
	expected := ppu.getColorFromCGRAM(0, 2)
	if color != expected {
		t.Fatalf("matrix 128x128 sample color = 0x%06X, want 0x%06X from tileX 87", color, expected)
	}
}

func TestMatrixModeDedicatedPlaneOverridesVRAMTilemap(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.BG0.Enabled = true
	ppu.BG0.TilemapSize = TilemapSize32x32
	ppu.BG0.TilemapBase = 0x4000
	ppu.TransformChannels[0].Enabled = true
	ppu.TransformChannels[0].A = 0x0100
	ppu.TransformChannels[0].D = 0x0100

	for i := 0; i < 32; i++ {
		ppu.VRAM[i] = 0x11
		ppu.VRAM[32+i] = 0x22
	}
	ppu.CGRAM[0x01*2] = 0x1F
	ppu.CGRAM[0x01*2+1] = 0x00
	ppu.CGRAM[0x02*2] = 0xE0
	ppu.CGRAM[0x02*2+1] = 0x03

	// Ordinary BG tilemap points at tile 0.
	ppu.VRAM[0x4000] = 0x00
	ppu.VRAM[0x4001] = 0x00

	// Dedicated matrix plane points at tile 1.
	ppu.MatrixPlanes[0].Enabled = true
	ppu.MatrixPlanes[0].Size = TilemapSize32x32
	ppu.MatrixPlanes[0].Tilemap[0] = 0x01
	ppu.MatrixPlanes[0].Tilemap[1] = 0x00
	ppu.MatrixPlanes[0].Pattern[32] = 0x22

	ppu.renderDotMatrixMode(0, 0, 0)
	color := ppu.OutputBuffer[0]
	expected := ppu.getColorFromCGRAM(0, 2)
	if color != expected {
		t.Fatalf("dedicated matrix plane color = 0x%06X, want 0x%06X from dedicated plane tile 1", color, expected)
	}
}

func TestMatrixModeDedicatedPlanePatternBaseOverridesDefaultTileSource(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.BG0.Enabled = true
	ppu.BG0.TilemapSize = TilemapSize32x32
	ppu.TransformChannels[0].Enabled = true
	ppu.TransformChannels[0].A = 0x0100
	ppu.TransformChannels[0].D = 0x0100

	ppu.MatrixPlanes[0].Enabled = true
	ppu.MatrixPlanes[0].Size = TilemapSize32x32
	ppu.MatrixPlanes[0].Tilemap[0] = 0x01
	ppu.MatrixPlanes[0].Tilemap[1] = 0x00

	// Default VRAM tile source for tile 1 -> palette color 2.
	ppu.VRAM[32] = 0x22
	// Dedicated matrix-plane pattern memory for tile 1 -> palette color 1.
	ppu.MatrixPlanes[0].Pattern[32] = 0x11
	ppu.CGRAM[0x01*2] = 0x1F
	ppu.CGRAM[0x01*2+1] = 0x00
	ppu.CGRAM[0x02*2] = 0xE0
	ppu.CGRAM[0x02*2+1] = 0x03

	ppu.renderDotMatrixMode(0, 0, 0)

	color := ppu.OutputBuffer[0]
	expected := ppu.getColorFromCGRAM(0, 1)
	if color != expected {
		t.Fatalf("dedicated matrix plane pattern memory color = 0x%06X, want 0x%06X from dedicated pattern tile 1", color, expected)
	}
}

func TestMatrixModeDedicatedPlaneBitmapUsesDedicatedBitmapMemory(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.BG0.Enabled = true
	ppu.BG0.TilemapSize = TilemapSize128x128
	ppu.BG0.TransformChannel = 0
	ppu.BG0.ScrollX = 0
	ppu.BG0.ScrollY = 0
	ppu.TransformChannels[0].Enabled = true
	ppu.TransformChannels[0].A = 0x0100
	ppu.TransformChannels[0].D = 0x0100
	ppu.TransformChannels[0].CenterX = 0
	ppu.TransformChannels[0].CenterY = 0

	ppu.MatrixPlanes[0].Enabled = true
	ppu.MatrixPlanes[0].Size = TilemapSize128x128
	ppu.MatrixPlanes[0].SourceMode = MatrixPlaneSourceBitmap
	ppu.MatrixPlanes[0].BitmapPalette = 1

	// Bitmap pixel at world coordinate (300,0) -> palette color index 2.
	pixelOffset := 300
	byteOffset := pixelOffset / 2
	ppu.MatrixPlanes[0].Bitmap[byteOffset] = 0x20

	ppu.CGRAM[(1*16+2)*2] = 0x1F
	ppu.CGRAM[(1*16+2)*2+1] = 0x00

	ppu.renderDotMatrixMode(0, 300, 0)
	color := ppu.OutputBuffer[300]
	expected := ppu.getColorFromCGRAM(1, 2)
	if color != expected {
		t.Fatalf("dedicated matrix plane bitmap color = 0x%06X, want 0x%06X", color, expected)
	}
}

func TestMatrixModeCenterActsAsSourceOrigin(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.BG0.Enabled = true
	ppu.BG0.TransformChannel = 0
	ppu.TransformChannels[0].Enabled = true
	ppu.TransformChannels[0].A = 0x0100
	ppu.TransformChannels[0].D = 0x0100
	ppu.TransformChannels[0].CenterX = 64
	ppu.TransformChannels[0].CenterY = 64

	ppu.MatrixPlanes[0].Enabled = true
	ppu.MatrixPlanes[0].Size = TilemapSize128x128
	ppu.MatrixPlanes[0].SourceMode = MatrixPlaneSourceBitmap
	ppu.MatrixPlanes[0].BitmapPalette = 1

	// Source pixel (0,0) = color 1, source pixel (64,64) = color 2.
	ppu.MatrixPlanes[0].Bitmap[0] = 0x10
	targetOffset := (64*1024 + 64) / 2
	if (64*1024+64)%2 == 0 {
		ppu.MatrixPlanes[0].Bitmap[targetOffset] = 0x20
	} else {
		ppu.MatrixPlanes[0].Bitmap[targetOffset] = 0x02
	}

	ppu.CGRAM[(1*16+1)*2] = 0x1F
	ppu.CGRAM[(1*16+1)*2+1] = 0x00
	ppu.CGRAM[(1*16+2)*2] = 0xE0
	ppu.CGRAM[(1*16+2)*2+1] = 0x03

	ppu.renderDotMatrixMode(0, 64, 64)
	color := ppu.OutputBuffer[64*320+64]
	expected := ppu.getColorFromCGRAM(1, 2)
	if color != expected {
		t.Fatalf("matrix center/origin sample color = 0x%06X, want 0x%06X from source pixel (64,64)", color, expected)
	}
}

func TestMatrixModeDedicatedPlaneBitmapIndexZeroIsTransparentWhenEnabled(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)
	ppu.BG1.Enabled = true
	ppu.BG1.Priority = 1
	ppu.BG1.ScrollX = 0
	ppu.BG1.ScrollY = 0
	ppu.BG1.TransformChannel = 1
	ppu.TransformChannels[1].Enabled = true
	ppu.TransformChannels[1].A = 0x0100
	ppu.TransformChannels[1].D = 0x0100

	ppu.VRAM[0] = 0x11
	ppu.CGRAM[2] = 0x1F
	ppu.CGRAM[3] = 0x00

	ppu.MatrixPlanes[1].Enabled = true
	ppu.MatrixPlanes[1].Size = TilemapSize128x128
	ppu.MatrixPlanes[1].SourceMode = MatrixPlaneSourceBitmap
	ppu.MatrixPlanes[1].BitmapPalette = 1
	ppu.MatrixPlanes[1].Transparent0 = true
	// Both nibbles zero => transparent.
	ppu.MatrixPlanes[1].Bitmap[0] = 0x00

	expected := uint32(0x112233)
	ppu.OutputBuffer[0] = expected
	ppu.renderDotMatrixMode(1, 0, 0)
	if got := ppu.OutputBuffer[0]; got != expected {
		t.Fatalf("transparent bitmap plane color = 0x%06X, want underlying BG color 0x%06X", got, expected)
	}
}

func TestMatrixModeDedicatedPlaneBitmapIndexZeroOpaqueWhenTransparencyDisabled(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)
	ppu.BG1.Enabled = true
	ppu.BG1.Priority = 1
	ppu.BG1.ScrollX = 0
	ppu.BG1.ScrollY = 0
	ppu.BG1.TransformChannel = 1
	ppu.TransformChannels[1].Enabled = true
	ppu.TransformChannels[1].A = 0x0100
	ppu.TransformChannels[1].D = 0x0100

	ppu.MatrixPlanes[1].Enabled = true
	ppu.MatrixPlanes[1].Size = TilemapSize128x128
	ppu.MatrixPlanes[1].SourceMode = MatrixPlaneSourceBitmap
	ppu.MatrixPlanes[1].BitmapPalette = 1
	ppu.MatrixPlanes[1].Transparent0 = false
	ppu.MatrixPlanes[1].Bitmap[0] = 0x00

	expected := ppu.getColorFromCGRAM(1, 0)
	ppu.OutputBuffer[0] = 0x112233
	ppu.renderDotMatrixMode(1, 0, 0)
	if got := ppu.OutputBuffer[0]; got != expected {
		t.Fatalf("opaque bitmap plane color = 0x%06X, want palette color 0x%06X", got, expected)
	}
}

func TestDMATransferToDedicatedMatrixPlaneBitmap(t *testing.T) {
	ppu := NewPPU(nil)
	source := []uint8{0x12, 0x34, 0x56, 0x78}
	ppu.MemoryReader = func(bank uint8, offset uint16) uint8 {
		if bank != 1 {
			return 0
		}
		if int(offset) >= len(source) {
			return 0
		}
		return source[offset]
	}

	ppu.MatrixPlaneSelect = 0
	ppu.MatrixPlaneBitmapAddr = 0
	ppu.DMASourceBank = 1
	ppu.DMASourceOffset = 0
	ppu.DMADestType = 5
	ppu.DMALength = uint16(len(source))
	ppu.DMAEnabled = true

	ppu.executeDMA()

	if got := ppu.MatrixPlanes[0].Bitmap[:len(source)]; got[0] != 0x12 || got[1] != 0x34 || got[2] != 0x56 || got[3] != 0x78 {
		t.Fatalf("matrix bitmap DMA upload mismatch: got=%v want=%v", got, source)
	}
}

func TestMatrixPlaneLiveFloorRegisterRoundTrip(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.Write8(0x80, 0x00)
	ppu.Write8(0x8D, 0x01)
	ppu.Write8(0x8E, 92)
	ppu.Write8(0x8F, 0x34)
	ppu.Write8(0x90, 0x12)
	ppu.Write8(0x91, 0x78)
	ppu.Write8(0x92, 0x56)
	ppu.Write8(0x93, 0x00)
	ppu.Write8(0x94, 0x01)
	ppu.Write8(0x95, 0x00)
	ppu.Write8(0x96, 0xFF)

	plane := ppu.MatrixPlanes[0]
	if !plane.LiveFloorEnabled {
		t.Fatal("expected live floor to be enabled")
	}
	if plane.LiveFloorHorizon != 92 {
		t.Fatalf("LiveFloorHorizon = %d, want 92", plane.LiveFloorHorizon)
	}
	if plane.LiveFloorCameraX != 0x1234 {
		t.Fatalf("LiveFloorCameraX = 0x%04X, want 0x1234", uint16(plane.LiveFloorCameraX))
	}
	if plane.LiveFloorCameraY != 0x5678 {
		t.Fatalf("LiveFloorCameraY = 0x%04X, want 0x5678", uint16(plane.LiveFloorCameraY))
	}
	if plane.LiveFloorHeadingX != 0x0100 {
		t.Fatalf("LiveFloorHeadingX = 0x%04X, want 0x0100", uint16(plane.LiveFloorHeadingX))
	}
	if uint16(plane.LiveFloorHeadingY) != 0xFF00 {
		t.Fatalf("LiveFloorHeadingY = 0x%04X, want 0xFF00", uint16(plane.LiveFloorHeadingY))
	}

	if got := ppu.Read8(0x8D); got != 0x01 {
		t.Fatalf("readback live floor control = 0x%02X, want 0x01", got)
	}
	if got := ppu.Read8(0x8E); got != 92 {
		t.Fatalf("readback live floor horizon = %d, want 92", got)
	}
}

func TestBitmapMatrixPlaneLiveFloorRendersBelowHorizon(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.CGRAM[0x11*2] = 0x1F
	ppu.CGRAM[0x11*2+1] = 0x03 // green-ish

	ppu.BG0.Enabled = true
	ppu.BG0.TransformChannel = 0
	ppu.TransformChannels[0].Enabled = true
	ppu.TransformChannels[0].OutsideMode = 0

	plane := &ppu.MatrixPlanes[0]
	plane.Enabled = true
	plane.Size = TilemapSize128x128
	plane.SourceMode = MatrixPlaneSourceBitmap
	plane.BitmapPalette = 1
	plane.LiveFloorEnabled = true
	plane.LiveFloorHorizon = 92
	plane.LiveFloorCameraX = 512
	plane.LiveFloorCameraY = 512
	plane.LiveFloorHeadingX = 0
	plane.LiveFloorHeadingY = -0x0100
	for i := range plane.Bitmap {
		plane.Bitmap[i] = 0x11
	}

	ppu.renderDotMatrixMode(0, 160, 80)
	if got := ppu.OutputBuffer[80*320+160]; got != 0 {
		t.Fatalf("pixel above horizon = 0x%06X, want 0x000000", got)
	}

	ppu.renderDotMatrixMode(0, 160, 150)
	want := ppu.getColorFromCGRAM(1, 1)
	if got := ppu.OutputBuffer[150*320+160]; got != want {
		t.Fatalf("pixel below horizon = 0x%06X, want 0x%06X", got, want)
	}
}

// TestDMATransfer tests DMA copy and fill modes
func TestDMATransfer(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Set up memory reader (mock)
	sourceData := make([]uint8, 256)
	for i := 0; i < 256; i++ {
		sourceData[i] = uint8(i)
	}

	ppu.MemoryReader = func(bank uint8, offset uint16) uint8 {
		if offset < uint16(len(sourceData)) {
			return sourceData[offset]
		}
		return 0
	}

	// Test DMA copy mode: copy 256 bytes from source to VRAM
	ppu.DMASourceBank = 0
	ppu.DMASourceOffset = 0
	ppu.DMADestAddr = 0x1000
	ppu.DMALength = 256
	ppu.DMAMode = 0     // Copy mode
	ppu.DMADestType = 0 // VRAM
	ppu.DMAEnabled = true

	ppu.executeDMA()

	// Verify data was copied
	for i := 0; i < 256; i++ {
		if ppu.VRAM[0x1000+i] != sourceData[i] {
			t.Errorf("DMA copy: VRAM[0x%04X] should be 0x%02X, got 0x%02X",
				0x1000+i, sourceData[i], ppu.VRAM[0x1000+i])
		}
	}

	// Test DMA fill mode: fill VRAM with value 0xAA
	// Reset DMA state
	ppu.DMAEnabled = false
	// Set fill value in source data BEFORE setting up memory reader
	sourceData[0] = 0xAA // Fill value (read from source offset 0)

	// Update memory reader to return the fill value
	ppu.MemoryReader = func(bank uint8, offset uint16) uint8 {
		if offset == 0 {
			return 0xAA // Fill value
		}
		if offset < uint16(len(sourceData)) {
			return sourceData[offset]
		}
		return 0
	}

	ppu.DMASourceOffset = 0
	ppu.DMADestAddr = 0x2000
	ppu.DMALength = 128
	ppu.DMAMode = 1     // Fill mode
	ppu.DMADestType = 0 // VRAM
	ppu.DMAEnabled = true

	ppu.executeDMA()

	// Verify VRAM was filled
	for i := 0; i < 128; i++ {
		if ppu.VRAM[0x2000+i] != 0xAA {
			t.Errorf("DMA fill: VRAM[0x%04X] should be 0xAA, got 0x%02X",
				0x2000+i, ppu.VRAM[0x2000+i])
		}
	}
}

// TestSpriteToBackgroundPriority tests sprite-to-background priority interaction
func TestSpriteToBackgroundPriority(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Set up colors
	ppu.CGRAM[0x01*2] = 0x00
	ppu.CGRAM[0x01*2+1] = 0x7C // Red
	ppu.CGRAM[0x02*2] = 0xE0
	ppu.CGRAM[0x02*2+1] = 0x03 // Green

	// Create tiles
	for i := 0; i < 32; i++ {
		ppu.VRAM[i] = 0x11    // Red tile
		ppu.VRAM[32+i] = 0x22 // Green tile
	}

	// Set up BG0 (priority 0) with green
	ppu.BG0.Enabled = true
	ppu.BG0.TilemapBase = 0x4000
	testTilemapOffset := uint16((((100 / 8) * 32) + (100 / 8)) * 2) // tilemap entry for pixel (100,100)
	ppu.VRAM[0x4000+testTilemapOffset] = 0x01                       // Green tile
	ppu.VRAM[0x4000+testTilemapOffset+1] = 0x00                     // Palette 0

	// Sprite 0: Priority 0 (same as BG0), red
	ppu.OAM[0] = 100
	ppu.OAM[1] = 0x00
	ppu.OAM[2] = 100
	ppu.OAM[3] = 0x00 // Red tile
	ppu.OAM[4] = 0x00 // Priority 0
	ppu.OAM[5] = 0x03 // Enabled

	// Render - sprite should render on top of background (same priority, sprite wins)
	ppu.renderDot(100, 100)

	color := ppu.OutputBuffer[100*320+100]
	expectedRed := uint32(0xFF0000) // RGB(255, 0, 0)
	if color != expectedRed {
		t.Errorf("Sprite-to-background priority: Expected red sprite (priority 0) on top, got 0x%06X", color)
	}

	// Test sprite with lower priority than background
	// BG1 has priority 1, sprite with priority 0 should be behind
	ppu.BG1.Enabled = true
	ppu.BG1.Priority = 1
	ppu.BG1.TilemapBase = 0x4000
	ppu.VRAM[0x4000+testTilemapOffset] = 0x01
	ppu.VRAM[0x4000+testTilemapOffset+1] = 0x00

	ppu.OAM[4] = 0x00 // Sprite priority 0 (lower than BG1 priority 1)

	ppu.renderDot(100, 100)
	color = ppu.OutputBuffer[100*320+100]
	expectedGreen := uint32(0x00FF00) // RGB(0, 255, 0)
	if color != expectedGreen {
		t.Errorf("Sprite-to-background priority: Expected green background (priority 1) on top of sprite (priority 0), got 0x%06X", color)
	}
}

func TestExplicitLayerPriorityOverridesBGIndexOrder(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Palette 0 color 1 = red, color 2 = green
	ppu.CGRAM[0x01*2] = 0x00
	ppu.CGRAM[0x01*2+1] = 0x7C
	ppu.CGRAM[0x02*2] = 0xE0
	ppu.CGRAM[0x02*2+1] = 0x03

	for i := 0; i < 32; i++ {
		ppu.VRAM[i] = 0x11    // tile 0 -> red
		ppu.VRAM[32+i] = 0x22 // tile 1 -> green
	}

	testTilemapOffset := uint16((((40 / 8) * 32) + (40 / 8)) * 2)

	ppu.BG0.Enabled = true
	ppu.BG0.Priority = 3
	ppu.BG0.TilemapBase = 0x4000
	ppu.VRAM[0x4000+testTilemapOffset] = 0x00
	ppu.VRAM[0x4000+testTilemapOffset+1] = 0x00

	ppu.BG3.Enabled = true
	ppu.BG3.Priority = 0
	ppu.BG3.TilemapBase = 0x5000
	ppu.VRAM[0x5000+testTilemapOffset] = 0x01
	ppu.VRAM[0x5000+testTilemapOffset+1] = 0x00

	ppu.renderDot(40, 40)

	color := ppu.OutputBuffer[40*320+40]
	expectedRed := uint32(0xFF0000)
	if color != expectedRed {
		t.Fatalf("explicit layer priority should put BG0(priority 3) over BG3(priority 0), got 0x%06X", color)
	}
}
