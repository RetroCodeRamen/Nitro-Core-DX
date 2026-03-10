package devkit

import (
	"fmt"

	"nitro-core-dx/internal/emulator"
)

const (
	RasterDemoSplitTilemap   = "split-tilemap"
	RasterDemoRebindPriority = "rebind-priority"
	RasterDemoScrollAffine   = "scroll-affine"
)

func (s *Service) InstallRasterDemo(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.emu == nil {
		return fmt.Errorf("no ROM loaded")
	}

	switch name {
	case RasterDemoSplitTilemap:
		return installSplitTilemapRasterDemoLocked(s.emu)
	case RasterDemoRebindPriority:
		return installRebindPriorityRasterDemoLocked(s.emu)
	case RasterDemoScrollAffine:
		return installScrollAffineRasterDemoLocked(s.emu)
	default:
		return fmt.Errorf("unknown raster demo %q", name)
	}
}

func installSplitTilemapRasterDemoLocked(emu *emulator.Emulator) error {
	ppu := emu.PPU
	ppu.BG0.Enabled = true
	ppu.BG0.Priority = 0
	ppu.BG0.TileSize = false
	ppu.BG0.TilemapBase = 0x1000
	ppu.BG0.TransformChannel = 0
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

	program, err := emulator.BuildSplitTilemapRasterProgram(0x3800, 0, 100, 0x1000, 0x1800)
	if err != nil {
		return err
	}
	return emu.InstallRasterProgram(program)
}

func installRebindPriorityRasterDemoLocked(emu *emulator.Emulator) error {
	ppu := emu.PPU
	ppu.BG0.Enabled = true
	ppu.BG0.Priority = 0
	ppu.BG0.TileSize = true
	ppu.BG0.TilemapBase = 0x1000
	ppu.BG0.TransformChannel = 0

	ppu.BG1.Enabled = true
	ppu.BG1.Priority = 1
	ppu.BG1.TileSize = false
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

	top := emulator.IdentityRasterTransform()
	bottom := emulator.IdentityRasterTransform()
	bottom.CenterX = -8

	program, err := emulator.BuildRebindPriorityRasterProgram(0x3C00, 0, 100, 0, 1, 0, 3, top, bottom)
	if err != nil {
		return err
	}
	return emu.InstallRasterProgram(program)
}

func installScrollAffineRasterDemoLocked(emu *emulator.Emulator) error {
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

	top := emulator.IdentityRasterTransform()
	bottom := emulator.IdentityRasterTransform()
	bottom.A = 0x0200
	bottom.ScrollX = 2

	program, err := emulator.BuildScrollAffineRasterProgram(0x3400, 0, 100, top, bottom)
	if err != nil {
		return err
	}
	return emu.InstallRasterProgram(program)
}
