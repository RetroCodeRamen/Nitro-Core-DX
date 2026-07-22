package corelx

import (
	"os"
	"path/filepath"
	"testing"

	"nitro-core-dx/internal/emulator"
)

// TestSpriteSize32x32EndToEnd proves the new SPR_SIZE_32X32 builtin (and
// the codegen bit-twiddling behind oam.write_sprite_data's X-high byte --
// see emitSizeCodeBits) round-trips correctly through the real compiler,
// not just hand-constructed OAM bytes (see internal/ppu/sprite_size_test.go
// for the PPU-side decode/tile-grid-addressing proof). A single 8x8 tile is
// loaded and declared as a 32x32 sprite: only its top-left cell has real
// pixel data, so a visible pixel there plus the raw OAM byte both have to
// check out for this to pass.
func TestSpriteSize32x32EndToEnd(t *testing.T) {
	source := `asset TileA: tiles8 hex
    11 11 11 11 11 11 11 11 11 11 11 11 11 11 11 11
    11 11 11 11 11 11 11 11 11 11 11 11 11 11 11 11

function Start()
    gfx.set_palette_color(1, 0x7C00)
    tile := gfx.load_tiles(ASSET_TileA, 0)
    -- ctrl must be precomputed into a local, not passed as an inline
    -- SPR_ENABLE()|SPR_SIZE_32X32() expression -- a computed expression as
    -- a call argument beyond the first corrupts earlier-evaluated argument
    -- registers in this compiler (see corelx-nested-expr-register-bug);
    -- this is the same established safe pattern the demo's own
    -- draw_object_sprite_lod already uses for its ctrl argument.
    ctrl := SPR_ENABLE() | SPR_SIZE_32X32()
    oam.write_sprite_data(0, 60, 60, tile, SPR_PAL(0), ctrl)
    ppu.enable_display()
    while true
        wait_vblank()
`
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sprite_size.corelx")
	if err := os.WriteFile(srcPath, []byte(source), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	result, err := CompileProject(srcPath, nil)
	if err != nil {
		t.Fatalf("compile: %v (diagnostics: %+v)", err, result.Diagnostics)
	}
	if HasErrors(result.Diagnostics) {
		t.Fatalf("compile produced error diagnostics: %+v", result.Diagnostics)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(result.ROMBytes); err != nil {
		t.Fatalf("LoadROM: %v", err)
	}
	emu.Start()
	for i := 0; i < 5; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame: %v", err)
		}
	}

	// Raw OAM check: byte 1 (X-high) bit 0 = sign (0, positive x), bits
	// [3:1] = size code 3 (32x32) -> byte value 0x06.
	xHigh := emu.PPU.OAM[1]
	if xHigh != 0x06 {
		t.Fatalf("OAM byte 1 (X-high): got 0x%02X, want 0x06 (sign=0, size code 3 = 32x32)", xHigh)
	}

	// A pixel inside the loaded top-left 8x8 cell must show the tile's
	// color (proves the sprite actually renders via tile-grid addressing,
	// not just that the OAM byte looks right). Tile-grid addressing and
	// hit-test bounds for the far/unloaded grid cells are already proven
	// with controlled VRAM content at the PPU level
	// (internal/ppu/sprite_size_test.go) -- this test's job is narrower:
	// prove the real compiler/codegen round-trips a SPR_SIZE_* value into
	// the OAM format correctly, which the byte check above and this
	// visible-pixel check together establish.
	color := emu.PPU.OutputBuffer[63*320+63]
	if color == 0 {
		t.Fatalf("expected a visible sprite pixel at (63,63), got black/transparent")
	}
}
