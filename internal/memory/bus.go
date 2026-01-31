package memory

import (
	"fmt"
	"nitro-core-dx/internal/debug"
)

// Bus represents the memory bus that routes memory accesses
// It connects CPU to WRAM, Extended WRAM, Cartridge, and I/O devices
type Bus struct {
	// WRAM (Work RAM) - Bank 0, 0x0000-0x7FFF (32KB)
	WRAM [32768]uint8

	// Extended WRAM - Banks 126-127 (128KB)
	WRAMExtended [131072]uint8

	// Cartridge (ROM)
	Cartridge *Cartridge

	// I/O handlers
	PPUHandler   IOHandler
	APUHandler   IOHandler
	InputHandler IOHandler

	// Logger for debug logging
	logger *debug.Logger
}

// IOHandler defines the interface for I/O register handlers
type IOHandler interface {
	Read8(offset uint16) uint8
	Write8(offset uint16, value uint8)
	Read16(offset uint16) uint16
	Write16(offset uint16, value uint16)
}

// NewBus creates a new memory bus
func NewBus(cartridge *Cartridge) *Bus {
	return &Bus{
		Cartridge: cartridge,
	}
}

// SetLogger sets the logger for debug logging
func (b *Bus) SetLogger(logger *debug.Logger) {
	b.logger = logger
}

// Read8 reads an 8-bit value from memory
func (b *Bus) Read8(bank uint8, offset uint16) uint8 {
	// Bank 0: WRAM (0x0000-0x7FFF) or I/O (0x8000+)
	if bank == 0 {
		if offset < 0x8000 {
			// WRAM
			return b.WRAM[offset]
		} else {
			// I/O registers
			return b.readIO8(offset)
		}
	}

	// Banks 1-125: ROM space (routed to cartridge)
	if bank >= 1 && bank <= 125 {
		if b.Cartridge != nil {
			return b.Cartridge.Read8(bank, offset)
		}
		// Unmapped: return 0 (could implement open bus behavior in future if needed)
		// Open bus would return the previous data bus value, but most ROMs don't rely on this
		return 0
	}

	// Banks 126-127: Extended WRAM
	if bank == 126 || bank == 127 {
		extOffset := (uint32(bank-126) * 65536) + uint32(offset)
		if extOffset < 131072 {
			return b.WRAMExtended[extOffset]
		}
		// Unmapped: return 0 (could implement open bus behavior in future if needed)
		return 0
	}

	// Unmapped bank: return 0
	// NOTE: Real hardware might have "open bus" behavior (returning previous data bus value),
	// but most ROMs don't rely on this. If ROM compatibility issues arise, open bus can be
	// implemented by tracking the last data bus value and returning it for unmapped addresses.
	return 0
}

// Write8 writes an 8-bit value to memory
func (b *Bus) Write8(bank uint8, offset uint16, value uint8) {
	// Bank 0: WRAM (0x0000-0x7FFF) or I/O (0x8000+)
	if bank == 0 {
		if offset < 0x8000 {
			// WRAM
			b.WRAM[offset] = value
		} else {
			// I/O registers
			b.writeIO8(offset, value)
		}
		return
	}

	// Banks 1-125: ROM (read-only, writes ignored)
	if bank >= 1 && bank <= 125 {
		return
	}

	// Banks 126-127: Extended WRAM
	if bank == 126 || bank == 127 {
		extOffset := (uint32(bank-126) * 65536) + uint32(offset)
		if extOffset < 131072 {
			b.WRAMExtended[extOffset] = value
		}
		return
	}
}

// Read16 reads a 16-bit value from memory (little-endian)
func (b *Bus) Read16(bank uint8, offset uint16) uint16 {
	low := b.Read8(bank, offset)
	high := b.Read8(bank, offset+1)
	result := uint16(low) | (uint16(high) << 8)
	return result
}

// Write16 writes a 16-bit value to memory (little-endian)
// For I/O registers that need special handling (like CGRAM_DATA),
// write both bytes to the same address
func (b *Bus) Write16(bank uint8, offset uint16, value uint16) {
	// Special case: CGRAM_DATA (0x8013) needs two 8-bit writes to the same address
	if bank == 0 && offset == 0x8013 {
		// Write low byte first, then high byte (both to same address)
		b.Write8(bank, offset, uint8(value&0xFF))
		b.Write8(bank, offset, uint8(value>>8))
		return
	}
	// Special case: CGRAM_ADDR (0x8012) - only write low byte, don't write to CGRAM_DATA
	if bank == 0 && offset == 0x8012 {
		// Only write the low byte to CGRAM_ADDR, ignore high byte
		b.Write8(bank, offset, uint8(value&0xFF))
		return
	}
	// Normal case: write low byte, then high byte to consecutive addresses
	b.Write8(bank, offset, uint8(value&0xFF))
	b.Write8(bank, offset+1, uint8(value>>8))
}

// readIO8 reads from I/O registers
func (b *Bus) readIO8(offset uint16) uint8 {
	// PPU registers: 0x8000-0x8FFF
	if offset >= 0x8000 && offset < 0x9000 {
		if b.PPUHandler != nil {
			return b.PPUHandler.Read8(offset - 0x8000)
		}
		return 0
	}

	// APU registers: 0x9000-0x9FFF
	if offset >= 0x9000 && offset < 0xA000 {
		if b.APUHandler != nil {
			return b.APUHandler.Read8(offset - 0x9000)
		}
		return 0
	}

	// Input registers: 0xA000-0xAFFF
	if offset >= 0xA000 && offset < 0xB000 {
		if b.InputHandler != nil {
			inputOffset := offset - 0xA000
			value := b.InputHandler.Read8(inputOffset)
			// Debug logging for input reads (if logger is available and input logging is enabled)
			if b.logger != nil && b.logger.IsComponentEnabled(debug.ComponentInput) {
				b.logger.LogInput(debug.LogLevelDebug, fmt.Sprintf("Input read: offset=0x%04X (0xA000+0x%02X), value=0x%02X", offset, inputOffset, value), nil)
			}
			return value
		}
		return 0
	}

	return 0
}

// writeIO8 writes to I/O registers
func (b *Bus) writeIO8(offset uint16, value uint8) {
	// PPU registers: 0x8000-0x8FFF
	if offset >= 0x8000 && offset < 0x9000 {
		if b.PPUHandler != nil {
			ppuOffset := offset - 0x8000
			// Log OAM writes for debugging
			if ppuOffset == 0x14 || ppuOffset == 0x15 {
				// OAM_ADDR or OAM_DATA - will be logged by PPU
			}
			b.PPUHandler.Write8(ppuOffset, value)
		}
		return
	}

	// APU registers: 0x9000-0x9FFF
	if offset >= 0x9000 && offset < 0xA000 {
		if b.APUHandler != nil {
			b.APUHandler.Write8(offset-0x9000, value)
		}
		return
	}

	// Input registers: 0xA000-0xAFFF
	if offset >= 0xA000 && offset < 0xB000 {
		if b.InputHandler != nil {
			inputOffset := offset - 0xA000
			b.InputHandler.Write8(inputOffset, value)
			// Debug logging for input writes (if logger is available and input logging is enabled)
			if b.logger != nil && b.logger.IsComponentEnabled(debug.ComponentInput) {
				b.logger.LogInput(debug.LogLevelDebug, fmt.Sprintf("Input write: offset=0x%04X (0xA000+0x%02X), value=0x%02X", offset, inputOffset, value), nil)
			}
		}
		return
	}
}
