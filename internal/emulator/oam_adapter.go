package emulator

import (
	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/ppu"
)

// OAMAdapter adapts PPU OAM to the debug.OAMReader interface
type OAMAdapter struct {
	ppu *ppu.PPU
}

// ReadOAM reads a byte from OAM at the given offset
func (a *OAMAdapter) ReadOAM(offset uint8) uint8 {
	if a.ppu == nil {
		return 0
	}
	if int(offset) < len(a.ppu.OAM) {
		return a.ppu.OAM[offset]
	}
	return 0
}

// NewOAMAdapter creates a new OAM adapter
func NewOAMAdapter(ppu *ppu.PPU) debug.OAMReader {
	return &OAMAdapter{ppu: ppu}
}
