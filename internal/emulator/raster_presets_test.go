package emulator

import "testing"

func TestBuildSplitTilemapRasterProgramRendersVisibleSplit(t *testing.T) {
	emu := NewEmulator()

	ppu := emu.PPU
	ppu.BG0.Enabled = true
	ppu.BG0.Priority = 0
	ppu.BG0.TilemapBase = 0x1000
	ppu.TransformChannels[0].Enabled = false

	ppu.CGRAM[0x01*2] = 0x00
	ppu.CGRAM[0x01*2+1] = 0x7C
	ppu.CGRAM[0x02*2] = 0xE0
	ppu.CGRAM[0x02*2+1] = 0x03

	for i := 0; i < 32; i++ {
		ppu.VRAM[0x0000+uint16(i)] = 0x11
		ppu.VRAM[0x0020+uint16(i)] = 0x22
	}
	for entry := 0; entry < 32*32; entry++ {
		addrA := uint16(0x1000 + entry*2)
		addrB := uint16(0x1800 + entry*2)
		ppu.VRAM[addrA] = 0x00
		ppu.VRAM[addrA+1] = 0x00
		ppu.VRAM[addrB] = 0x01
		ppu.VRAM[addrB+1] = 0x00
	}

	program, err := BuildSplitTilemapRasterProgram(0x3800, 0, 100, 0x1000, 0x1800)
	if err != nil {
		t.Fatalf("BuildSplitTilemapRasterProgram failed: %v", err)
	}
	if err := emu.InstallRasterProgram(program); err != nil {
		t.Fatalf("InstallRasterProgram failed: %v", err)
	}
	if err := ppu.StepPPU(uint64(220 * 581)); err != nil {
		t.Fatalf("StepPPU failed: %v", err)
	}

	topColor := ppu.OutputBuffer[10*320+10]
	bottomColor := ppu.OutputBuffer[(200-10)*320+10]
	if topColor == 0 || bottomColor == 0 {
		t.Fatalf("expected visible colors, got top=0x%06X bottom=0x%06X", topColor, bottomColor)
	}
	if topColor == bottomColor {
		t.Fatalf("expected raster split colors to differ, got 0x%06X", topColor)
	}
}

func TestBuildRebindPriorityRasterProgramRendersVisibleSwap(t *testing.T) {
	emu := NewEmulator()

	ppu := emu.PPU
	ppu.BG0.Enabled = true
	ppu.BG0.Priority = 0
	ppu.BG0.TileSize = true
	ppu.BG0.TilemapBase = 0x1000

	ppu.BG1.Enabled = true
	ppu.BG1.Priority = 1
	ppu.BG1.TilemapBase = 0x1800
	ppu.BG1.TransformChannel = 2

	ppu.TransformChannels[0].Enabled = true
	ppu.TransformChannels[1].Enabled = true
	ppu.TransformChannels[2].Enabled = false

	ppu.CGRAM[0x01*2] = 0x00
	ppu.CGRAM[0x01*2+1] = 0x7C
	ppu.CGRAM[0x02*2] = 0xE0
	ppu.CGRAM[0x02*2+1] = 0x03
	ppu.CGRAM[0x03*2] = 0x1F
	ppu.CGRAM[0x03*2+1] = 0x00

	for row := 0; row < 16; row++ {
		rowBase := row * 8
		for colByte := 0; colByte < 4; colByte++ {
			ppu.VRAM[uint16(rowBase+colByte)] = 0x11
		}
		for colByte := 4; colByte < 8; colByte++ {
			ppu.VRAM[uint16(rowBase+colByte)] = 0x22
		}
	}
	for i := 0; i < 32; i++ {
		ppu.VRAM[0x0080+uint16(i)] = 0x33
	}
	for entry := 0; entry < 32*32; entry++ {
		addrBG0 := uint16(0x1000 + entry*2)
		addrBG1 := uint16(0x1800 + entry*2)
		ppu.VRAM[addrBG0] = 0x00
		ppu.VRAM[addrBG0+1] = 0x00
		ppu.VRAM[addrBG1] = 0x04
		ppu.VRAM[addrBG1+1] = 0x00
	}

	top := IdentityRasterTransform()
	bottom := IdentityRasterTransform()
	bottom.CenterX = -8

	program, err := BuildRebindPriorityRasterProgram(0x3C00, 0, 100, 0, 1, 0, 3, top, bottom)
	if err != nil {
		t.Fatalf("BuildRebindPriorityRasterProgram failed: %v", err)
	}
	if err := emu.InstallRasterProgram(program); err != nil {
		t.Fatalf("InstallRasterProgram failed: %v", err)
	}
	if err := ppu.StepPPU(uint64(220 * 581)); err != nil {
		t.Fatalf("StepPPU failed: %v", err)
	}

	topColor := ppu.OutputBuffer[10*320+4]
	bottomColor := ppu.OutputBuffer[(200-10)*320+4]
	if topColor == 0 || bottomColor == 0 {
		t.Fatalf("expected visible colors, got top=0x%06X bottom=0x%06X", topColor, bottomColor)
	}
	if topColor == bottomColor {
		t.Fatalf("expected rebind/priority raster swap to change visible color, got 0x%06X", topColor)
	}
}

func TestBuildScrollAffineRasterProgramRendersVisibleWarp(t *testing.T) {
	emu := NewEmulator()

	ppu := emu.PPU
	ppu.BG0.Enabled = true
	ppu.BG0.Priority = 0
	ppu.BG0.TileSize = true
	ppu.BG0.TilemapBase = 0x1000
	ppu.BG0.TransformChannel = 0
	ppu.TransformChannels[0].Enabled = true

	ppu.CGRAM[0x01*2] = 0x00
	ppu.CGRAM[0x01*2+1] = 0x7C
	ppu.CGRAM[0x02*2] = 0xE0
	ppu.CGRAM[0x02*2+1] = 0x03

	for row := 0; row < 16; row++ {
		rowBase := row * 8
		for colByte := 0; colByte < 4; colByte++ {
			ppu.VRAM[uint16(rowBase+colByte)] = 0x11
		}
		for colByte := 4; colByte < 8; colByte++ {
			ppu.VRAM[uint16(rowBase+colByte)] = 0x22
		}
	}
	for entry := 0; entry < 32*32; entry++ {
		addr := uint16(0x1000 + entry*2)
		ppu.VRAM[addr] = 0x00
		ppu.VRAM[addr+1] = 0x00
	}

	top := IdentityRasterTransform()
	bottom := IdentityRasterTransform()
	bottom.A = 0x0200
	bottom.ScrollX = 2

	program, err := BuildScrollAffineRasterProgram(0x3400, 0, 100, top, bottom)
	if err != nil {
		t.Fatalf("BuildScrollAffineRasterProgram failed: %v", err)
	}
	if err := emu.InstallRasterProgram(program); err != nil {
		t.Fatalf("InstallRasterProgram failed: %v", err)
	}
	if err := ppu.StepPPU(uint64(220 * 581)); err != nil {
		t.Fatalf("StepPPU failed: %v", err)
	}

	topColor := ppu.OutputBuffer[10*320+3]
	bottomColor := ppu.OutputBuffer[(200-10)*320+3]
	if topColor == 0 || bottomColor == 0 {
		t.Fatalf("expected visible colors, got top=0x%06X bottom=0x%06X", topColor, bottomColor)
	}
	if topColor == bottomColor {
		t.Fatalf("expected scroll/affine raster warp to change visible color, got 0x%06X", topColor)
	}
}

func TestBuildRasterPresetsRejectOutOfRangeInputs(t *testing.T) {
	if _, err := BuildSplitTilemapRasterProgram(0x3000, 4, 100, 0x1000, 0x1800); err == nil {
		t.Fatal("expected split tilemap preset to reject invalid layer")
	}
	if _, err := BuildRebindPriorityRasterProgram(0x3000, 0, 201, 0, 1, 0, 3, IdentityRasterTransform(), IdentityRasterTransform()); err == nil {
		t.Fatal("expected rebind/priority preset to reject invalid split scanline")
	}
	if _, err := BuildScrollAffineRasterProgram(0x3000, 0, 201, IdentityRasterTransform(), IdentityRasterTransform()); err == nil {
		t.Fatal("expected scroll/affine preset to reject invalid split scanline")
	}
}
