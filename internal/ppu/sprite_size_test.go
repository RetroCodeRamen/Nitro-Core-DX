package ppu

import (
	"testing"

	"nitro-core-dx/internal/debug"
)

// setSpriteOAM writes one 6-byte OAM entry directly, with sizeCode folded
// into the X-high byte's bits [3:1] (see spriteSizeTable's doc comment).
func setSpriteOAM(ppu *PPU, index int, x int, y, tile, palette, priority, sizeCode uint8) {
	oamAddr := index * 6
	ppu.OAM[oamAddr] = uint8(x & 0xFF)
	xHigh := uint8(0)
	if x < 0 {
		xHigh |= 0x01
	}
	xHigh |= (sizeCode & 0x07) << 1
	ppu.OAM[oamAddr+1] = xHigh
	ppu.OAM[oamAddr+2] = y
	ppu.OAM[oamAddr+3] = tile
	ppu.OAM[oamAddr+4] = (palette & 0x0F) | (priority << 6)
	ppu.OAM[oamAddr+5] = 0x01 // enable
}

// setPaletteColor sets CGRAM entry (palette, colorIndex) to a distinguishable
// non-black RGB555 color derived from the two indices, so different tiles
// can be told apart by their rendered color.
func setPaletteColor(ppu *PPU, palette, colorIndex uint8, rgb555 uint16) {
	off := (int(palette)*16 + int(colorIndex)) * 2
	ppu.CGRAM[off] = uint8(rgb555 & 0xFF)
	ppu.CGRAM[off+1] = uint8(rgb555 >> 8)
}

// fill8x8Tile fills one 32-byte 8x8 4bpp tile at VRAM tile index tileIndex
// with a single color index (both nibbles of every byte).
func fill8x8Tile(ppu *PPU, tileIndex int, colorIndex uint8) {
	base := tileIndex * 32
	b := (colorIndex << 4) | colorIndex
	for i := 0; i < 32; i++ {
		ppu.VRAM[base+i] = b
	}
}

// TestSprite32x32TileGridAddressing proves the new tile-grid addressing
// (not the legacy contiguous-blob scheme) is actually used for sizes above
// 16x16: a 32x32 sprite (size code 3) built from 4 distinctly-colored 8x8
// tiles in its top row, at base tileIndex 10, must show each tile's color
// in the correct screen quadrant -- proving tiles are fetched
// sequentially/row-major from the base index, not read as one contiguous
// blob (which would produce nonsense given only 4 of the 16 grid cells
// have real data). Also proves the Y-range hit-test uses the size table's
// height (32), not a hardcoded 8/16.
func TestSprite32x32TileGridAddressing(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	setPaletteColor(ppu, 0, 1, 0x001F) // red-ish
	setPaletteColor(ppu, 0, 2, 0x03E0) // green-ish
	setPaletteColor(ppu, 0, 3, 0x7C00) // blue-ish
	setPaletteColor(ppu, 0, 4, 0x7FFF) // white

	baseTile := 10
	fill8x8Tile(ppu, baseTile+0, 1) // grid cell (0,0): top-left
	fill8x8Tile(ppu, baseTile+1, 2) // grid cell (1,0): top, second column
	fill8x8Tile(ppu, baseTile+2, 3) // grid cell (2,0)
	fill8x8Tile(ppu, baseTile+3, 4) // grid cell (3,0)
	// Cells in rows 1-3 (tileIndex+4..+15) are left as zero (transparent).

	setSpriteOAM(ppu, 0, 50, 50, uint8(baseTile), 0, 1, 3) // size code 3 = 32x32

	cases := []struct {
		name          string
		x, y          int
		wantTransparent bool
		wantColorIndex  uint8
	}{
		{"cell(0,0) top-left", 50 + 3, 50 + 3, false, 1},
		{"cell(1,0) second column", 50 + 11, 50 + 3, false, 2},
		{"cell(2,0) third column", 50 + 19, 50 + 3, false, 3},
		{"cell(3,0) fourth column", 50 + 27, 50 + 3, false, 4},
		{"cell(0,1) second row (unloaded, transparent)", 50 + 3, 50 + 11, true, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ppu.OutputBuffer[c.y*320+c.x] = 0 // reset
			ppu.renderDot(c.y, c.x) // renderDot(scanline, dot) == (y, x)
			got := ppu.OutputBuffer[c.y*320+c.x]
			if c.wantTransparent {
				if got != 0 {
					t.Errorf("expected transparent (black) at (%d,%d), got 0x%06X", c.x, c.y, got)
				}
				return
			}
			want := ppu.getColorFromCGRAM(0, c.wantColorIndex)
			if got != want {
				t.Errorf("at (%d,%d): got 0x%06X, want 0x%06X (color index %d)", c.x, c.y, got, want, c.wantColorIndex)
			}
		})
	}

	// Hit-test: a 32x32 sprite at Y=50 must still be considered active at
	// Y=81 (the last row of its 32-tall bounding box, 50..81 inclusive)
	// and NOT at Y=82 (one past it) -- proving the scanline evaluation
	// uses the size table's height, not a hardcoded 8/16.
	ppu.evaluateSpritesForScanline(81)
	if ppu.activeScanlineSpriteCount != 1 {
		t.Errorf("expected sprite still active at y=81 (bottom row of a 32-tall sprite), got count=%d", ppu.activeScanlineSpriteCount)
	}
	ppu.evaluateSpritesForScanline(82)
	if ppu.activeScanlineSpriteCount != 0 {
		t.Errorf("expected sprite NOT active at y=82 (one past a 32-tall sprite), got count=%d", ppu.activeScanlineSpriteCount)
	}
}

// TestSpriteLegacySizesUnchanged confirms 8x8 (size code 0) and 16x16 (size
// code 1) still use the original contiguous-blob VRAM addressing after the
// size-code field moved into the X-high byte -- this has to stay exactly as
// it was, since gfx.load_tiles' write-side addressing (internal/corelx/
// codegen.go) depends on it for every already-shipped sprite.
func TestSpriteLegacySizesUnchanged(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)
	setPaletteColor(ppu, 0, 5, 0x1234)

	// 16x16 sprite: one contiguous 128-byte blob at tileIndex*128, matching
	// gfx.load_tiles' base*128 write-side addressing exactly (not 4
	// separate 32-byte tiles at sequential indices).
	tileIndex := 2
	base := tileIndex * 128
	for i := 0; i < 128; i++ {
		ppu.VRAM[base+i] = 0x55 // color index 5 in both nibbles
	}
	setSpriteOAM(ppu, 0, 60, 60, uint8(tileIndex), 0, 1, 1) // size code 1 = 16x16

	ppu.renderDot(60+15, 60+15) // bottom-right corner, within the 16x16 box
	got := ppu.OutputBuffer[(60+15)*320+(60+15)]
	want := ppu.getColorFromCGRAM(0, 5)
	if got != want {
		t.Errorf("16x16 legacy addressing: got 0x%06X, want 0x%06X", got, want)
	}

	// Hit-test still 16 tall/wide, not 32 (which sizeCode=1 would mean
	// under the new table if misdecoded).
	ppu.evaluateSpritesForScanline(60 + 15)
	if ppu.activeScanlineSpriteCount != 1 {
		t.Fatalf("expected 16x16 sprite active at its last row, got count=%d", ppu.activeScanlineSpriteCount)
	}
	ppu.evaluateSpritesForScanline(60 + 16)
	if ppu.activeScanlineSpriteCount != 0 {
		t.Errorf("expected 16x16 sprite NOT active one row past its bottom, got count=%d", ppu.activeScanlineSpriteCount)
	}
}

// TestSpriteScanlineByteBudgetDropsLowestPriorityFirst proves the new
// per-scanline sprite pixel-fetch budget (spriteScanlineByteBudget) is
// enforced, and that it drops lowest-priority sprites first, consistently
// -- not the highest-priority ones, and not randomly -- when a scanline's
// combined sprite width exceeds the budget.
func TestSpriteScanlineByteBudgetDropsLowestPriorityFirst(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Three 128-wide sprites (size code 7 = 128x128) all on the same
	// scanline: cost 64 bytes each (128/2, 4bpp). Budget is 128 bytes, so
	// only 2 of the 3 can fit. Priorities: sprite 0 = priority 3 (highest),
	// sprite 1 = priority 2, sprite 2 = priority 1 (lowest) -- sprite 2
	// should be the one dropped.
	setSpriteOAM(ppu, 0, 0, 20, 0, 0, 3, 7)
	setSpriteOAM(ppu, 1, 0, 20, 0, 0, 2, 7)
	setSpriteOAM(ppu, 2, 0, 20, 0, 0, 1, 7)

	ppu.evaluateSpritesForScanline(20)
	if ppu.activeScanlineSpriteCount != 2 {
		t.Fatalf("expected exactly 2 of 3 128-wide sprites to survive the %d-byte scanline budget, got %d",
			spriteScanlineByteBudget, ppu.activeScanlineSpriteCount)
	}
	survivorIndices := map[int]bool{}
	for i := 0; i < ppu.activeScanlineSpriteCount; i++ {
		survivorIndices[ppu.activeScanlineSprites[i].index] = true
	}
	if !survivorIndices[0] || !survivorIndices[1] {
		t.Errorf("expected the two highest-priority sprites (OAM index 0, 1) to survive, got survivors=%v", survivorIndices)
	}
	if survivorIndices[2] {
		t.Errorf("expected the lowest-priority sprite (OAM index 2) to be dropped, but it survived")
	}
}

// TestSpriteLegacyCtrlFieldAssignmentFallback proves the pre-existing idiom
// of setting a Sprite struct's size via a plain field assignment
// (`box.ctrl = SPR_ENABLE() | SPR_SIZE_16()`, never touching X-high at all)
// still decodes correctly, via spriteSizeCodeFromOAM's fallback to the
// legacy control-byte bit. This is exactly the pattern
// Games/NitroPackInDemo/corelx/overworld.corelx's hero sprite uses (and the
// pattern PROGRAMMING_MANUAL.md's original Chapter 12 example predates this
// session's sprite-size work with) -- without the fallback, this idiom
// silently decodes as size code 0 (8x8) with 8x8 VRAM addressing, even
// though the art was loaded as a 16x16 tile: wrong size and garbled pixels.
func TestSpriteLegacyCtrlFieldAssignmentFallback(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)
	setPaletteColor(ppu, 0, 5, 0x1234)

	tileIndex := 2
	base := tileIndex * 128
	for i := 0; i < 128; i++ {
		ppu.VRAM[base+i] = 0x55
	}

	// OAM written the way a plain `.ctrl = SPR_ENABLE()|SPR_SIZE_16()`
	// field assignment would leave it: X-high entirely untouched (0), size
	// info only in the control byte's legacy bit 1.
	oamAddr := 0
	ppu.OAM[oamAddr] = 60   // X low
	ppu.OAM[oamAddr+1] = 0  // X high -- untouched, no size code here
	ppu.OAM[oamAddr+2] = 60 // Y
	ppu.OAM[oamAddr+3] = uint8(tileIndex)
	ppu.OAM[oamAddr+4] = 0 << 6 // palette 0, priority 0
	ppu.OAM[oamAddr+5] = 0x03  // enable | legacy 16x16 bit, control byte only

	ppu.renderDot(60+15, 60+15) // bottom-right corner of a 16x16 box
	got := ppu.OutputBuffer[(60+15)*320+(60+15)]
	want := ppu.getColorFromCGRAM(0, 5)
	if got != want {
		t.Errorf("legacy ctrl-only 16x16 sprite: got 0x%06X, want 0x%06X (tile data/size decode broken)", got, want)
	}

	ppu.evaluateSpritesForScanline(60 + 15)
	if ppu.activeScanlineSpriteCount != 1 {
		t.Fatalf("expected legacy 16x16 sprite active at its last row, got count=%d", ppu.activeScanlineSpriteCount)
	}
	ppu.evaluateSpritesForScanline(60 + 16)
	if ppu.activeScanlineSpriteCount != 0 {
		t.Errorf("expected legacy 16x16 sprite NOT active one row past its bottom (would be true if misdecoded as 8x8's height=8, or wrongly kept at some other size)")
	}
}
