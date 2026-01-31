package memory

import (
	"fmt"
	"nitro-core-dx/internal/debug"
)

// MemorySystem represents the complete memory system
type MemorySystem struct {
	// WRAM (Work RAM) - Bank 0, 0x0000-0x7FFF (32KB)
	WRAM [32768]uint8

	// Extended WRAM - Banks 126-127 (128KB)
	WRAMExtended [131072]uint8

	// ROM data
	ROMData  []uint8
	ROMSize  uint32
	ROMBanks uint8

	// ROM header (for entry point)
	ROMHeader [32]uint8

	// I/O handlers
	PPUHandler   IOHandler
	APUHandler   IOHandler
	InputHandler IOHandler

	// Logger for debug logging
	logger *debug.Logger
}

// NewMemorySystem creates a new memory system
func NewMemorySystem() *MemorySystem {
	return &MemorySystem{
		ROMData: make([]uint8, 0),
	}
}

// SetLogger sets the logger for debug logging
func (m *MemorySystem) SetLogger(logger *debug.Logger) {
	m.logger = logger
}

// LoadROM loads ROM data into memory
func (m *MemorySystem) LoadROM(data []uint8) error {
	if len(data) < 32 {
		return fmt.Errorf("ROM too small: %d bytes", len(data))
	}

	// Parse header
	magic := uint32(data[0]) | (uint32(data[1]) << 8) | (uint32(data[2]) << 16) | (uint32(data[3]) << 24)
	if magic != 0x46434D52 { // "RMCF"
		return fmt.Errorf("invalid ROM magic: 0x%08X", magic)
	}

	version := uint16(data[4]) | (uint16(data[5]) << 8)
	if version > 1 {
		return fmt.Errorf("unsupported ROM version: %d", version)
	}

	romSize := uint32(data[6]) | (uint32(data[7]) << 8) | (uint32(data[8]) << 16) | (uint32(data[9]) << 24)

	// Save header for entry point lookup
	copy(m.ROMHeader[:], data[0:32])

	// Load ROM data (skip 32-byte header)
	if len(data) < int(romSize)+32 {
		return fmt.Errorf("ROM data too small: expected %d bytes, got %d", romSize+32, len(data))
	}

	m.ROMData = make([]uint8, romSize)
	copy(m.ROMData, data[32:32+romSize])
	m.ROMSize = romSize
	m.ROMBanks = uint8((romSize + 65535) / 65536) // Round up to nearest bank

	return nil
}

// Read8 reads an 8-bit value from memory
func (m *MemorySystem) Read8(bank uint8, offset uint16) uint8 {
	// Bank 0: WRAM (0x0000-0x7FFF) or I/O (0x8000+)
	if bank == 0 {
		if offset < 0x8000 {
			// WRAM
			return m.WRAM[offset]
		} else {
			// I/O registers
			return m.readIO8(offset)
		}
	}

	// Banks 1-125: ROM space (LoROM mapping, appears at 0x8000+)
	if bank >= 1 && bank <= 125 {
		if offset < 0x8000 {
			return 0 // Unmapped
		}
		romOffset := (uint32(bank-1) * 32768) + uint32(offset-0x8000)
		if romOffset < m.ROMSize {
			return m.ROMData[romOffset]
		}
		return 0
	}

	// Banks 126-127: Extended WRAM
	if bank == 126 || bank == 127 {
		extOffset := (uint32(bank-126) * 65536) + uint32(offset)
		if extOffset < 131072 {
			return m.WRAMExtended[extOffset]
		}
		return 0
	}

	return 0
}

// Write8 writes an 8-bit value to memory
func (m *MemorySystem) Write8(bank uint8, offset uint16, value uint8) {
	// Bank 0: WRAM (0x0000-0x7FFF) or I/O (0x8000+)
	if bank == 0 {
		if offset < 0x8000 {
			// WRAM
			m.WRAM[offset] = value
		} else {
			// I/O registers
			m.writeIO8(offset, value)
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
			m.WRAMExtended[extOffset] = value
		}
		return
	}
}

// Read16 reads a 16-bit value from memory (little-endian)
func (m *MemorySystem) Read16(bank uint8, offset uint16) uint16 {
	low := m.Read8(bank, offset)
	high := m.Read8(bank, offset+1)
	result := uint16(low) | (uint16(high) << 8)
	return result
}

// Write16 writes a 16-bit value to memory (little-endian)
// For I/O registers that need special handling (like CGRAM_DATA),
// write both bytes to the same address
func (m *MemorySystem) Write16(bank uint8, offset uint16, value uint16) {
	// Special case: CGRAM_DATA (0x8013) needs two 8-bit writes to the same address
	if bank == 0 && offset == 0x8013 {
		// Write low byte first, then high byte (both to same address)
		m.Write8(bank, offset, uint8(value&0xFF))
		m.Write8(bank, offset, uint8(value>>8))
		return
	}
	// Special case: CGRAM_ADDR (0x8012) - only write low byte, don't write to CGRAM_DATA
	if bank == 0 && offset == 0x8012 {
		// Only write the low byte to CGRAM_ADDR, ignore high byte
		m.Write8(bank, offset, uint8(value&0xFF))
		return
	}
	// Normal case: write low byte, then high byte to consecutive addresses
	m.Write8(bank, offset, uint8(value&0xFF))
	m.Write8(bank, offset+1, uint8(value>>8))
}

// readIO8 reads from I/O registers
func (m *MemorySystem) readIO8(offset uint16) uint8 {
	// PPU registers: 0x8000-0x8FFF
	if offset >= 0x8000 && offset < 0x9000 {
		if m.PPUHandler != nil {
			return m.PPUHandler.Read8(offset - 0x8000)
		}
		return 0
	}

	// APU registers: 0x9000-0x9FFF
	if offset >= 0x9000 && offset < 0xA000 {
		if m.APUHandler != nil {
			return m.APUHandler.Read8(offset - 0x9000)
		}
		return 0
	}

	// Input registers: 0xA000-0xAFFF
	if offset >= 0xA000 && offset < 0xB000 {
		if m.InputHandler != nil {
			inputOffset := offset - 0xA000
			value := m.InputHandler.Read8(inputOffset)
			// Debug logging for input reads (if logger is available and input logging is enabled)
			if m.logger != nil && m.logger.IsComponentEnabled(debug.ComponentInput) {
				m.logger.LogInput(debug.LogLevelDebug, fmt.Sprintf("Input read: offset=0x%04X (0xA000+0x%02X), value=0x%02X", offset, inputOffset, value), nil)
			}
			return value
		}
		return 0
	}

	return 0
}

// writeIO8 writes to I/O registers
func (m *MemorySystem) writeIO8(offset uint16, value uint8) {
	// PPU registers: 0x8000-0x8FFF
	if offset >= 0x8000 && offset < 0x9000 {
		if m.PPUHandler != nil {
			m.PPUHandler.Write8(offset-0x8000, value)
		}
		return
	}

	// APU registers: 0x9000-0x9FFF
	if offset >= 0x9000 && offset < 0xA000 {
		if m.APUHandler != nil {
			m.APUHandler.Write8(offset-0x9000, value)
		}
		return
	}

	// Input registers: 0xA000-0xAFFF
	if offset >= 0xA000 && offset < 0xB000 {
		if m.InputHandler != nil {
			inputOffset := offset - 0xA000
			m.InputHandler.Write8(inputOffset, value)
			// Debug logging for input writes (if logger is available and input logging is enabled)
			if m.logger != nil && m.logger.IsComponentEnabled(debug.ComponentInput) {
				m.logger.LogInput(debug.LogLevelDebug, fmt.Sprintf("Input write: offset=0x%04X (0xA000+0x%02X), value=0x%02X", offset, inputOffset, value), nil)
			}
		}
		return
	}
}

// GetROMEntryPoint returns the ROM entry point from the header
func (m *MemorySystem) GetROMEntryPoint() (bank uint8, offset uint16, err error) {
	if m.ROMSize == 0 {
		return 0, 0, fmt.Errorf("ROM not loaded")
	}

	// Entry point is in header (offsets 0x0A-0x0D)
	// Little-endian: low byte first, then high byte
	entryBank := uint16(m.ROMHeader[10]) | (uint16(m.ROMHeader[11]) << 8)
	entryOffset := uint16(m.ROMHeader[12]) | (uint16(m.ROMHeader[13]) << 8)

	// Validate entry point
	if entryBank == 0 {
		return 0, 0, fmt.Errorf("invalid ROM entry point: bank is 0 (expected bank 1-125, got 0). "+
			"ROM code must be located in bank 1 or higher. Bank 0 is reserved for WRAM and I/O registers. "+
			"Please check your ROM header entry point (offsets 0x0A-0x0B) and ensure it specifies a valid ROM bank (1-125).")
	}
	if entryBank > 125 {
		return 0, 0, fmt.Errorf("invalid ROM entry point: bank %d (expected bank 1-125, got %d). "+
			"ROM banks are limited to 1-125. Banks 126-127 are reserved for extended WRAM. "+
			"Please check your ROM header entry point (offsets 0x0A-0x0B) and ensure it specifies a valid ROM bank (1-125).",
			entryBank, entryBank)
	}
	if entryOffset < 0x8000 {
		return 0, 0, fmt.Errorf("invalid ROM entry point: offset 0x%04X (expected offset 0x8000-0xFFFF, got 0x%04X). "+
			"ROM code must start at offset 0x8000 or higher within the bank (LoROM mapping). "+
			"Please check your ROM header entry point (offsets 0x0C-0x0D) and ensure it specifies an offset >= 0x8000.",
			entryOffset, entryOffset)
	}

	return uint8(entryBank), entryOffset, nil
}
