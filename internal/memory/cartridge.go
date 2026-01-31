package memory

import (
	"fmt"
)

// Cartridge represents the ROM cartridge
// It holds ROM data and provides read-only access
type Cartridge struct {
	// ROM data (without header)
	ROMData []uint8

	// ROM metadata
	ROMSize  uint32
	ROMBanks uint8

	// ROM header (for entry point lookup)
	ROMHeader [32]uint8
}

// NewCartridge creates a new cartridge instance
func NewCartridge() *Cartridge {
	return &Cartridge{
		ROMData: make([]uint8, 0),
	}
}

// LoadROM loads ROM data into the cartridge
func (c *Cartridge) LoadROM(data []uint8) error {
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
	copy(c.ROMHeader[:], data[0:32])

	// Load ROM data (skip 32-byte header)
	if len(data) < int(romSize)+32 {
		return fmt.Errorf("ROM data too small: expected %d bytes, got %d", romSize+32, len(data))
	}

	c.ROMData = make([]uint8, romSize)
	copy(c.ROMData, data[32:32+romSize])
	c.ROMSize = romSize
	c.ROMBanks = uint8((romSize + 65535) / 65536) // Round up to nearest bank

	return nil
}

// Read8 reads an 8-bit value from ROM
// bank: 1-125 (ROM banks)
// offset: 0x8000-0xFFFF (ROM appears at 0x8000+)
func (c *Cartridge) Read8(bank uint8, offset uint16) uint8 {
	// Banks 1-125: ROM space (LoROM mapping, appears at 0x8000+)
	if bank >= 1 && bank <= 125 {
		if offset < 0x8000 {
			return 0 // Unmapped
		}
		romOffset := (uint32(bank-1) * 32768) + uint32(offset-0x8000)
		if romOffset < c.ROMSize {
			return c.ROMData[romOffset]
		}
		return 0
	}
	return 0
}

// Read16 reads a 16-bit value from ROM (little-endian)
func (c *Cartridge) Read16(bank uint8, offset uint16) uint16 {
	low := c.Read8(bank, offset)
	high := c.Read8(bank, offset+1)
	return uint16(low) | (uint16(high) << 8)
}

// GetROMEntryPoint returns the ROM entry point from the header
func (c *Cartridge) GetROMEntryPoint() (bank uint8, offset uint16, err error) {
	if c.ROMSize == 0 {
		return 0, 0, fmt.Errorf("ROM not loaded")
	}

	// Entry point is in header (offsets 0x0A-0x0D)
	// Little-endian: low byte first, then high byte
	entryBank := uint16(c.ROMHeader[10]) | (uint16(c.ROMHeader[11]) << 8)
	entryOffset := uint16(c.ROMHeader[12]) | (uint16(c.ROMHeader[13]) << 8)

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

// HasROM returns true if a ROM is loaded
func (c *Cartridge) HasROM() bool {
	return c.ROMSize > 0
}
