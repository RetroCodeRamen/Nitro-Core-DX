package emulator

import "testing"

func TestRasterProgramBuilderAndInstall(t *testing.T) {
	emu := NewEmulator()

	builder := NewRasterProgramBuilder(0x3400, RasterProgramLayout{
		Enabled:            true,
		LayerMask:          0x01,
		IncludePriority:    true,
		IncludeTilemapBase: true,
		IncludeSourceMode:  true,
	})

	topHalf := RasterScanlineProgram{
		Layers: [4]RasterLayerProgram{
			{
				ScrollX:        0x0010,
				ScrollY:        0x0020,
				TransformA:     0x0100,
				TransformD:     0x0100,
				HasPriority:    true,
				Priority:       1,
				HasTilemapBase: true,
				TilemapBase:    0x1000,
				HasSourceMode:  true,
				SourceMode:     0,
			},
		},
	}
	bottomHalf := RasterScanlineProgram{
		Layers: [4]RasterLayerProgram{
			{
				ScrollX:        0x0030,
				ScrollY:        0x0040,
				TransformA:     0x0200,
				TransformD:     0x0200,
				HasPriority:    true,
				Priority:       3,
				HasTilemapBase: true,
				TilemapBase:    0x1800,
				HasSourceMode:  true,
				SourceMode:     1,
			},
		},
	}

	if err := builder.FillScanlineRange(0, 99, topHalf); err != nil {
		t.Fatalf("FillScanlineRange top: %v", err)
	}
	if err := builder.FillScanlineRange(100, 199, bottomHalf); err != nil {
		t.Fatalf("FillScanlineRange bottom: %v", err)
	}

	if err := emu.InstallRasterProgram(builder.Build()); err != nil {
		t.Fatalf("InstallRasterProgram failed: %v", err)
	}

	if emu.PPU.HDMATableBase != 0x3400 {
		t.Fatalf("HDMATableBase = 0x%04X, want 0x3400", emu.PPU.HDMATableBase)
	}
	if !emu.PPU.HDMAEnabled {
		t.Fatal("expected HDMA enabled after raster install")
	}
	if emu.PPU.HDMAControl != 0xC3 {
		t.Fatalf("HDMAControl = 0x%02X, want 0xC3", emu.PPU.HDMAControl)
	}
	if emu.PPU.HDMAExtControl != 0x01 {
		t.Fatalf("HDMAExtControl = 0x%02X, want 0x01", emu.PPU.HDMAExtControl)
	}

	base := int(emu.PPU.HDMATableBase)
	if got := emu.PPU.VRAM[base]; got != 0x10 {
		t.Fatalf("top scanline scrollX low byte = 0x%02X, want 0x10", got)
	}

	stride := int(rasterProgramStride(RasterProgramLayout{
		Enabled:            true,
		LayerMask:          0x01,
		IncludePriority:    true,
		IncludeTilemapBase: true,
		IncludeSourceMode:  true,
	}))
	bottomBase := base + 100*stride
	if got := emu.PPU.VRAM[bottomBase]; got != 0x30 {
		t.Fatalf("bottom scanline scrollX low byte = 0x%02X, want 0x30", got)
	}
}

func TestRasterProgramBuilderRejectsOutOfRangeRange(t *testing.T) {
	builder := NewRasterProgramBuilder(0x2000, RasterProgramLayout{})
	if err := builder.FillScanlineRange(0, 200, RasterScanlineProgram{}); err == nil {
		t.Fatal("expected out-of-range fill to fail")
	}
}

func TestClearRasterProgramDisablesHDMA(t *testing.T) {
	emu := NewEmulator()
	emu.PPU.HDMAEnabled = true
	emu.PPU.HDMAControl = 0xFF
	emu.PPU.HDMAExtControl = 0x01

	if err := emu.ClearRasterProgram(); err != nil {
		t.Fatalf("ClearRasterProgram failed: %v", err)
	}
	if emu.PPU.HDMAEnabled || emu.PPU.HDMAControl != 0 || emu.PPU.HDMAExtControl != 0 {
		t.Fatalf("expected raster state cleared, got enabled=%v control=0x%02X ext=0x%02X",
			emu.PPU.HDMAEnabled, emu.PPU.HDMAControl, emu.PPU.HDMAExtControl)
	}
}
