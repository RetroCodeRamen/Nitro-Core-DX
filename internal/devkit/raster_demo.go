package devkit

import (
	"fmt"

	"nitro-core-dx/internal/emulator"
	ppucore "nitro-core-dx/internal/ppu"
)

const (
	RasterDemoSplitTilemap   = "split-tilemap"
	RasterDemoRebindPriority = "rebind-priority"
	RasterDemoScrollAffine   = "scroll-affine"
	RasterDemoMatrixPlane    = "matrix-plane"
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
	case RasterDemoMatrixPlane:
		return installMatrixPlaneRasterDemoLocked(s.emu)
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

func installMatrixPlaneRasterDemoLocked(emu *emulator.Emulator) error {
	ppu := emu.PPU
	ppu.BG0.Enabled = true
	ppu.BG0.Priority = 0
	ppu.BG0.TileSize = false
	ppu.BG0.TilemapSize = ppucore.TilemapSize128x128
	ppu.BG0.TransformChannel = 0
	ppu.BG0.ScrollX = 256
	ppu.BG0.ScrollY = 192

	ppu.TransformChannels[0].Enabled = true
	ppu.TransformChannels[0].A = 0x0100
	ppu.TransformChannels[0].B = 0x0040
	ppu.TransformChannels[0].C = -0x0040
	ppu.TransformChannels[0].D = 0x0100
	ppu.TransformChannels[0].CenterX = 160
	ppu.TransformChannels[0].CenterY = 100
	ppu.TransformChannels[0].OutsideMode = 0

	ppu.CGRAM[0x01*2] = 0x00
	ppu.CGRAM[0x01*2+1] = 0x7C // red
	ppu.CGRAM[0x02*2] = 0xE0
	ppu.CGRAM[0x02*2+1] = 0x03 // green
	ppu.CGRAM[0x03*2] = 0x1F
	ppu.CGRAM[0x03*2+1] = 0x00 // blue
	ppu.CGRAM[0x04*2] = 0xFF
	ppu.CGRAM[0x04*2+1] = 0x03 // yellow

	pattern := make([]byte, 32*4)
	for i := 0; i < 32; i++ {
		pattern[0x00+uint16(i)] = 0x11
		pattern[0x20+uint16(i)] = 0x22
		pattern[0x40+uint16(i)] = 0x33
		pattern[0x60+uint16(i)] = 0x44
	}

	tilemap := make([]byte, 128*128*2)
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			entry := (y*128 + x) * 2
			tile := uint8(((x / 8) + (y / 8)) & 0x01)
			if x >= 64 {
				tile += 2
			}
			if (x+y)%29 == 0 {
				tile = 3
			}
			tilemap[entry] = tile
			tilemap[entry+1] = 0x00
		}
	}

	return emu.InstallMatrixPlaneProgram(emulator.MatrixPlaneProgram{
		Channel: 0,
		Enabled: true,
		Size:    ppucore.TilemapSize128x128,
		Tilemap: tilemap,
		Pattern: pattern,
	})
}
