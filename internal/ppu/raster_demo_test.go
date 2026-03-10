package ppu

import (
	"testing"

	"nitro-core-dx/internal/debug"
)

// TestScanlineCommandProgramRendersPerScanlineTilemapEffect is an end-to-end
// raster demo test. It uses the authored scanline table API to switch tilemap
// bases per scanline and verifies the rendered frame visibly changes across
// scanlines, not just internal register state.
func TestScanlineCommandProgramRendersPerScanlineTilemapEffect(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.HDMATableBase = 0x3800
	ppu.ApplyScanlineCommandLayout(ScanlineCommandLayout{
		Enabled:            true,
		LayerMask:          0x01, // BG0 only
		IncludeTilemapBase: true,
	})

	// BG0 visible, direct-color matrix path disabled. We want normal tilemap
	// rendering whose source base changes per scanline.
	ppu.BG0.Enabled = true
	ppu.BG0.Priority = 0
	ppu.BG0.TilemapBase = 0x1000
	ppu.TransformChannels[0].Enabled = false

	// Palette entries:
	// palette 0, color 1 = red
	ppu.CGRAM[0x01*2] = 0x00
	ppu.CGRAM[0x01*2+1] = 0x7C
	// palette 0, color 2 = green
	ppu.CGRAM[0x02*2] = 0xE0
	ppu.CGRAM[0x02*2+1] = 0x03

	// Tile 0 = solid palette index 1 (red)
	for i := 0; i < 32; i++ {
		ppu.VRAM[0x0000+uint16(i)] = 0x11
	}
	// Tile 1 = solid palette index 2 (green)
	for i := 0; i < 32; i++ {
		ppu.VRAM[0x0020+uint16(i)] = 0x22
	}

	// Fill both tilemaps so every sampled scanline resolves to the intended tile.
	for entry := 0; entry < 32*32; entry++ {
		addrA := uint16(0x1000 + entry*2)
		addrB := uint16(0x1800 + entry*2)
		ppu.VRAM[addrA] = 0x00
		ppu.VRAM[addrA+1] = 0x00
		ppu.VRAM[addrB] = 0x01
		ppu.VRAM[addrB+1] = 0x00
	}

	// Program every visible scanline. Top half uses tilemap A, bottom half uses
	// tilemap B. Keep the rest of the payload neutral.
	for scanline := 0; scanline < VisibleScanlines; scanline++ {
		tilemapBase := uint16(0x1000)
		if scanline >= VisibleScanlines/2 {
			tilemapBase = 0x1800
		}
		err := ppu.WriteScanlineCommandProgram(scanline, ScanlineCommandProgram{
			Layers: [4]ScanlineLayerProgram{
				{
					ScrollX:        0,
					ScrollY:        0,
					TransformA:     0x0100,
					TransformB:     0,
					TransformC:     0,
					TransformD:     0x0100,
					CenterX:        0,
					CenterY:        0,
					HasTilemapBase: true,
					TilemapBase:    tilemapBase,
				},
			},
		})
		if err != nil {
			t.Fatalf("WriteScanlineCommandProgram(%d) failed: %v", scanline, err)
		}
	}

	// Render a full frame through the normal scanline path.
	if err := ppu.StepPPU(uint64(TotalScanlines * DotsPerScanline)); err != nil {
		t.Fatalf("StepPPU failed: %v", err)
	}

	topColor := ppu.OutputBuffer[10*ScreenWidth+10]
	bottomColor := ppu.OutputBuffer[(VisibleScanlines-10)*ScreenWidth+10]

	if topColor == 0 {
		t.Fatal("top-half raster demo pixel rendered black; expected visible color")
	}
	if bottomColor == 0 {
		t.Fatal("bottom-half raster demo pixel rendered black; expected visible color")
	}
	if topColor == bottomColor {
		t.Fatalf("raster demo did not visibly change across scanlines: top=0x%06X bottom=0x%06X", topColor, bottomColor)
	}
}

// TestScanlineCommandProgramRendersRebindAndPriorityEffect verifies a visible
// raster effect that depends on both transform-channel rebinding and explicit
// layer priority changes across scanlines.
func TestScanlineCommandProgramRendersRebindAndPriorityEffect(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.HDMATableBase = 0x3C00
	ppu.ApplyScanlineCommandLayout(ScanlineCommandLayout{
		Enabled:         true,
		LayerMask:       0x03, // BG0 + BG1
		IncludeRebind:   true,
		IncludePriority: true,
	})

	ppu.BG0.Enabled = true
	ppu.BG0.Priority = 0
	ppu.BG0.TileSize = true
	ppu.BG0.TilemapBase = 0x1000

	ppu.BG1.Enabled = true
	ppu.BG1.Priority = 1
	ppu.BG1.TileSize = false
	ppu.BG1.TilemapBase = 0x1800
	ppu.BG1.TransformChannel = 2 // Keep BG1 on a disabled channel so it renders normally.

	ppu.TransformChannels[0].Enabled = true
	ppu.TransformChannels[1].Enabled = true
	ppu.TransformChannels[2].Enabled = false

	// Palette 0 colors:
	// color 1 = red
	ppu.CGRAM[0x01*2] = 0x00
	ppu.CGRAM[0x01*2+1] = 0x7C
	// color 2 = green
	ppu.CGRAM[0x02*2] = 0xE0
	ppu.CGRAM[0x02*2+1] = 0x03
	// color 3 = blue
	ppu.CGRAM[0x03*2] = 0x1F
	ppu.CGRAM[0x03*2+1] = 0x00

	// BG0 tile 0 (16x16): left half red, right half green.
	for row := 0; row < 16; row++ {
		rowBase := row * 8
		for colByte := 0; colByte < 4; colByte++ {
			ppu.VRAM[uint16(rowBase+colByte)] = 0x11
		}
		for colByte := 4; colByte < 8; colByte++ {
			ppu.VRAM[uint16(rowBase+colByte)] = 0x22
		}
	}

	// BG1 tile 4 (8x8): solid blue. Tile index 4 starts at 4*32 = 128.
	for i := 0; i < 32; i++ {
		ppu.VRAM[0x0080+uint16(i)] = 0x33
	}

	// Fill tilemaps.
	for entry := 0; entry < 32*32; entry++ {
		addrBG0 := uint16(0x1000 + entry*2)
		addrBG1 := uint16(0x1800 + entry*2)
		ppu.VRAM[addrBG0] = 0x00
		ppu.VRAM[addrBG0+1] = 0x00
		ppu.VRAM[addrBG1] = 0x04
		ppu.VRAM[addrBG1+1] = 0x00
	}

	for scanline := 0; scanline < VisibleScanlines; scanline++ {
		bg0 := ScanlineLayerProgram{
			ScrollX:    0,
			ScrollY:    0,
			TransformA: 0x0100,
			TransformB: 0,
			TransformC: 0,
			TransformD: 0x0100,
			CenterX:    0,
			CenterY:    0,
		}

		if scanline >= VisibleScanlines/2 {
			bg0.HasRebind = true
			bg0.Rebind = 1
			bg0.HasPriority = true
			bg0.Priority = 3
			bg0.CenterX = -8 // shift sample right within the 16x16 tile => green half
		}

		err := ppu.WriteScanlineCommandProgram(scanline, ScanlineCommandProgram{
			Layers: [4]ScanlineLayerProgram{
				bg0,
				{
					ScrollX:    0,
					ScrollY:    0,
					TransformA: 0x0100,
					TransformB: 0,
					TransformC: 0,
					TransformD: 0x0100,
					CenterX:    0,
					CenterY:    0,
				},
			},
		})
		if err != nil {
			t.Fatalf("WriteScanlineCommandProgram(%d) failed: %v", scanline, err)
		}
	}

	if err := ppu.StepPPU(uint64(TotalScanlines * DotsPerScanline)); err != nil {
		t.Fatalf("StepPPU failed: %v", err)
	}

	topColor := ppu.OutputBuffer[10*ScreenWidth+4]
	bottomColor := ppu.OutputBuffer[(VisibleScanlines-10)*ScreenWidth+4]
	expectedTop := ppu.getColorFromCGRAM(0, 3)    // BG1 blue wins in top half
	expectedBottom := ppu.getColorFromCGRAM(0, 2) // BG0 rebound green wins in bottom half

	if topColor != expectedTop {
		t.Fatalf("top-half pixel = 0x%06X, want blue 0x%06X", topColor, expectedTop)
	}
	if bottomColor != expectedBottom {
		t.Fatalf("bottom-half pixel = 0x%06X, want green 0x%06X", bottomColor, expectedBottom)
	}
}
