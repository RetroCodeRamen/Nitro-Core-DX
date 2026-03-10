package ppu

import "fmt"

// ScanlineCommandLayout describes which scanline command tables are active.
// LayerMask uses bit 0=BG0 through bit 3=BG3.
type ScanlineCommandLayout struct {
	Enabled            bool
	LayerMask          uint8
	IncludeRebind      bool
	IncludePriority    bool
	IncludeTilemapBase bool
	IncludeSourceMode  bool
}

// ScanlineLayerProgram is the authored scanline state for one visible layer.
// The base payload (scroll + transform + center) is always written.
// Optional fields are only emitted when their corresponding layout table is enabled.
type ScanlineLayerProgram struct {
	ScrollX, ScrollY int16
	TransformA       int16
	TransformB       int16
	TransformC       int16
	TransformD       int16
	CenterX          int16
	CenterY          int16

	HasRebind      bool
	Rebind         uint8
	HasPriority    bool
	Priority       uint8
	HasTilemapBase bool
	TilemapBase    uint16
	HasSourceMode  bool
	SourceMode     uint8
}

// ScanlineCommandProgram contains the authored state for one scanline.
type ScanlineCommandProgram struct {
	Layers [4]ScanlineLayerProgram
}

// ApplyScanlineCommandLayout programs the active scanline command-table layout.
func (p *PPU) ApplyScanlineCommandLayout(layout ScanlineCommandLayout) {
	var control uint8
	if layout.Enabled {
		control |= 0x01
	}
	control |= (layout.LayerMask & 0x0F) << 1
	if layout.IncludeRebind {
		control |= hdmaRebindTablePresent
	}
	if layout.IncludePriority {
		control |= hdmaPriorityTablePresent
	}
	if layout.IncludeTilemapBase {
		control |= hdmaTilemapTablePresent
	}
	p.HDMAEnabled = layout.Enabled
	p.HDMAControl = control

	var ext uint8
	if layout.IncludeSourceMode {
		ext |= hdmaExtSourceModePresent
	}
	p.HDMAExtControl = ext
}

// WriteScanlineCommandProgram serializes a scanline command program into VRAM
// using the currently configured scanline command-table layout.
func (p *PPU) WriteScanlineCommandProgram(scanline int, program ScanlineCommandProgram) error {
	if scanline < 0 || scanline >= VisibleScanlines {
		return fmt.Errorf("scanline %d out of range", scanline)
	}

	stride := p.hdmaScanlineStride()
	base := uint32(p.HDMATableBase) + uint32(scanline)*stride
	if base+stride > uint32(len(p.VRAM)) {
		return fmt.Errorf("scanline command table exceeds VRAM: base=0x%04X stride=%d", base, stride)
	}

	for layerNum := 0; layerNum < 4; layerNum++ {
		addr := uint16(base + uint32(layerNum*hdmaLayerPayloadBytes))
		layer := program.Layers[layerNum]
		writeUint16LE(p.VRAM[:], addr+0, uint16(layer.ScrollX))
		writeUint16LE(p.VRAM[:], addr+2, uint16(layer.ScrollY))
		writeUint16LE(p.VRAM[:], addr+4, uint16(layer.TransformA))
		writeUint16LE(p.VRAM[:], addr+6, uint16(layer.TransformB))
		writeUint16LE(p.VRAM[:], addr+8, uint16(layer.TransformC))
		writeUint16LE(p.VRAM[:], addr+10, uint16(layer.TransformD))
		writeUint16LE(p.VRAM[:], addr+12, uint16(layer.CenterX))
		writeUint16LE(p.VRAM[:], addr+14, uint16(layer.CenterY))
	}

	rebindBase := base + hdmaBaseScanlineBytes
	priorityBase := rebindBase
	if (p.HDMAControl & hdmaRebindTablePresent) != 0 {
		for layerNum := 0; layerNum < 4; layerNum++ {
			value := uint8(hdmaRebindSentinelKeep)
			if program.Layers[layerNum].HasRebind {
				value = program.Layers[layerNum].Rebind & 0x03
			}
			p.VRAM[uint16(rebindBase+uint32(layerNum))] = value
		}
		priorityBase += 4
	}

	tilemapBase := priorityBase
	if (p.HDMAControl & hdmaPriorityTablePresent) != 0 {
		for layerNum := 0; layerNum < 4; layerNum++ {
			value := uint8(hdmaPrioritySentinelKeep)
			if program.Layers[layerNum].HasPriority {
				value = program.Layers[layerNum].Priority & 0x03
			}
			p.VRAM[uint16(priorityBase+uint32(layerNum))] = value
		}
		tilemapBase += 4
	}

	sourceModeBase := tilemapBase
	if (p.HDMAControl & hdmaTilemapTablePresent) != 0 {
		for layerNum := 0; layerNum < 4; layerNum++ {
			value := uint16(hdmaTilemapSentinelKeep)
			if program.Layers[layerNum].HasTilemapBase {
				value = program.Layers[layerNum].TilemapBase
			}
			writeUint16LE(p.VRAM[:], uint16(tilemapBase+uint32(layerNum*2)), value)
		}
		sourceModeBase += 8
	}

	if (p.HDMAExtControl & hdmaExtSourceModePresent) != 0 {
		for layerNum := 0; layerNum < 4; layerNum++ {
			value := uint8(hdmaSourceModeSentinelKeep)
			if program.Layers[layerNum].HasSourceMode {
				value = program.Layers[layerNum].SourceMode & 0x01
			}
			p.VRAM[uint16(sourceModeBase+uint32(layerNum))] = value
		}
	}

	return nil
}

func writeUint16LE(buf []uint8, addr uint16, value uint16) {
	buf[addr] = uint8(value & 0xFF)
	buf[addr+1] = uint8(value >> 8)
}
