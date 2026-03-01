package emulator

import (
	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/ppu"
)

// PPUAdapter adapts PPU to the debug.PPUStateReader interface
type PPUAdapter struct {
	ppu *ppu.PPU
}

// GetScanline returns the current scanline
func (a *PPUAdapter) GetScanline() int {
	if a.ppu == nil {
		return -1
	}
	return a.ppu.GetScanline()
}

// GetDot returns the current dot
func (a *PPUAdapter) GetDot() int {
	if a.ppu == nil {
		return -1
	}
	return a.ppu.GetDot()
}

// GetVBlankFlag returns the VBlank flag state
func (a *PPUAdapter) GetVBlankFlag() bool {
	if a.ppu == nil {
		return false
	}
	return a.ppu.VBlankFlag
}

// GetFrameCounter returns the frame counter
func (a *PPUAdapter) GetFrameCounter() uint16 {
	if a.ppu == nil {
		return 0
	}
	return a.ppu.FrameCounter
}

// GetOAMByteIndex returns the current OAM byte index
func (a *PPUAdapter) GetOAMByteIndex() uint8 {
	if a.ppu == nil {
		return 0
	}
	return a.ppu.GetOAMByteIndex()
}

// NewPPUAdapter creates a new PPU adapter
func NewPPUAdapter(ppu *ppu.PPU) debug.PPUStateReader {
	return &PPUAdapter{ppu: ppu}
}
