package ppu

import (
	"nitro-core-dx/internal/debug"
)

// PPU represents the Picture Processing Unit
// It implements the memory.IOHandler interface
type PPU struct {
	// VRAM (64KB)
	VRAM [65536]uint8

	// CGRAM (512 bytes, 256 colors × 2 bytes)
	CGRAM [512]uint8

	// OAM (768 bytes, 128 sprites × 6 bytes)
	OAM [768]uint8

	// Background layers (each has its own matrix transformation)
	BG0, BG1, BG2, BG3 BackgroundLayer

	// Legacy Matrix Mode (deprecated - use per-layer matrix instead)
	// Kept for backward compatibility, maps to BG0 matrix
	MatrixEnabled    bool
	MatrixA, MatrixB int16 // 8.8 fixed point
	MatrixC, MatrixD int16 // 8.8 fixed point
	MatrixCenterX    int16
	MatrixCenterY    int16
	MatrixMirrorH    bool
	MatrixMirrorV    bool

	// Windowing
	Window0, Window1 Window
	WindowControl    uint8
	WindowMainEnable uint8
	WindowSubEnable  uint8

	// HDMA
	HDMAEnabled   bool
	HDMATableBase uint16
	HDMAControl   uint8 // Bit 0=enable, bits 1-4=layer enable (BG0-BG3), bits 5-7=matrix update enable per layer

	// Debug
	debugFrameCount int
	HDMAScrollX     [4][200]int16
	HDMAScrollY     [4][200]int16
	HDMAMatrixA     [4][200]int16 // Per-scanline matrix A updates
	HDMAMatrixB     [4][200]int16 // Per-scanline matrix B updates
	HDMAMatrixC     [4][200]int16 // Per-scanline matrix C updates
	HDMAMatrixD     [4][200]int16 // Per-scanline matrix D updates
	HDMAMatrixCX    [4][200]int16 // Per-scanline center X updates
	HDMAMatrixCY    [4][200]int16 // Per-scanline center Y updates

	// Frame counter (for ROM timing) - increments once per frame
	FrameCounter uint16

	// VBlank flag (hardware-accurate synchronization signal)
	// Set at start of VBlank period (scanline 200), cleared when read (one-shot)
	// This matches real hardware behavior (NES, SNES, etc.)
	// FPGA-implementable: Simple D flip-flop with read-clear logic
	VBlankFlag bool

	// Logger for centralized logging
	Logger *debug.Logger

	// Interrupt callback (called when VBlank occurs)
	// This allows PPU to trigger CPU interrupts
	InterruptCallback func(interruptType uint8)

	// Memory reader for DMA (reads from ROM/RAM)
	// Set by emulator to allow DMA transfers
	MemoryReader func(bank uint8, offset uint16) uint8

	// VRAM/CGRAM/OAM access registers
	VRAMAddr        uint16
	CGRAMAddr       uint8
	CGRAMWriteLatch bool // For 16-bit RGB555 writes

	// DMA (Direct Memory Access)
	DMAEnabled      bool
	DMASourceBank   uint8
	DMASourceOffset uint16
	DMADestType     uint8 // 0=VRAM, 1=CGRAM, 2=OAM
	DMADestAddr     uint16
	DMALength       uint16
	DMAMode         uint8  // 0=copy, 1=fill
	DMACycles       uint16 // Cycles remaining for DMA transfer (deprecated, use DMAProgress)
	// Cycle-accurate DMA state
	DMAProgress     uint16 // Current byte position in transfer (0 = start, DMALength = complete)
	DMACurrentSrc   uint16 // Current source offset
	DMACurrentDest  uint16 // Current destination address
	DMAFillValue    uint8  // Fill value for fill mode (read once at start)
	CGRAMWriteValue uint16
	OAMAddr         uint8
	OAMByteIndex    uint8 // Current byte index within sprite (0-5)

	// Output buffer (320×200, RGB888)
	OutputBuffer [320 * 200]uint32

	// Scanline/dot stepping state (for clock-driven operation)
	currentScanline     int
	currentDot          int
	scanlineInitialized bool
	frameStarted        bool
	FrameComplete       bool // Set to true when frame rendering is complete (safe to read buffer)
}

// GetScanline returns the current scanline (for debugging)
func (p *PPU) GetScanline() int {
	return p.currentScanline
}

// GetDot returns the current dot (for debugging)
func (p *PPU) GetDot() int {
	return p.currentDot
}

// GetOAMByteIndex returns the current OAM byte index (for debugging)
func (p *PPU) GetOAMByteIndex() uint8 {
	return p.OAMByteIndex
}

// BackgroundLayer represents a background layer
type BackgroundLayer struct {
	ScrollX     int16
	ScrollY     int16
	Enabled     bool
	TileSize    bool // false = 8×8, true = 16×16
	TilemapBase uint16

	// Matrix Mode (per-layer transformation)
	MatrixEnabled     bool
	MatrixA, MatrixB  int16 // 8.8 fixed point
	MatrixC, MatrixD  int16 // 8.8 fixed point
	MatrixCenterX     int16
	MatrixCenterY     int16
	MatrixMirrorH     bool
	MatrixMirrorV     bool
	MatrixOutsideMode uint8 // 0=repeat/wrap, 1=backdrop, 2=character #0
	MatrixDirectColor bool  // Direct color mode (bypass CGRAM, use direct RGB)
	// Mosaic effect
	MosaicEnabled bool
	MosaicSize    uint8 // 1-15 (1 = no effect, 15 = max block size)
}

// Window represents a window
type Window struct {
	Left, Right, Top, Bottom uint8
}

// NewPPU creates a new PPU instance
func NewPPU(logger *debug.Logger) *PPU {
	return &PPU{
		BG0:     BackgroundLayer{},
		BG1:     BackgroundLayer{},
		BG2:     BackgroundLayer{},
		BG3:     BackgroundLayer{},
		Window0: Window{},
		Window1: Window{},
		Logger:  logger,
	}
}

// Read8 reads an 8-bit value from PPU registers
func (p *PPU) Read8(offset uint16) uint8 {
	switch offset {
	case 0x10: // VRAM_DATA
		value := p.VRAM[p.VRAMAddr]
		p.VRAMAddr++
		if p.VRAMAddr > 0xFFFF {
			p.VRAMAddr = 0
		}
		return value
	case 0x13: // CGRAM_DATA
		// CGRAM is write-only, return 0
		return 0
	case 0x15: // OAM_DATA
		if p.OAMAddr < 128 {
			addr := uint16(p.OAMAddr)*6 + uint16(p.OAMByteIndex)
			if addr < 768 {
				value := p.OAM[addr]
				// Increment byte index (like write does)
				p.OAMByteIndex++
				if p.OAMByteIndex >= 6 {
					// Move to next sprite after reading 6 bytes
					p.OAMByteIndex = 0
					p.OAMAddr++
					if p.OAMAddr > 127 {
						p.OAMAddr = 0
					}
				}
				return value
			}
		}
		return 0
	case 0x3E: // VBLANK_FLAG (one-shot: cleared when read)
		// VBlank flag: hardware-accurate synchronization signal
		// Set at start of VBlank period (scanline 200), cleared when read (one-shot)
		// Bit 0 = VBlank active (1 = VBlank period, 0 = not VBlank)
		// This matches real hardware behavior (NES, SNES, etc.)
		//
		// IMPORTANT: The flag persists through the entire VBlank period (scanlines 200-219).
		// If ROM reads it during VBlank and clears it, we re-set it so it's available
		// for the rest of VBlank. This allows ROM to read the flag multiple times during
		// VBlank if needed (though typically only once).
		//
		// CRITICAL FIX: Check if we're in VBlank BEFORE reading the flag value.
		// This ensures the flag is set correctly even if it was cleared by a previous read.
		inVBlank := p.currentScanline >= VisibleScanlines && p.currentScanline < TotalScanlines

		flag := p.VBlankFlag

		// If we're in VBlank period, the flag should always be true
		// This fixes the issue where ROM reads flag multiple times during VBlank
		if inVBlank {
			flag = true
		}

		// Clear flag after read (one-shot behavior)
		// But immediately re-set if still in VBlank period
		p.VBlankFlag = false
		if inVBlank {
			p.VBlankFlag = true
		}

		if p.Logger != nil {
			p.Logger.LogPPUf(debug.LogLevelDebug,
				"VBlank flag read: scanline=%d, dot=%d, inVBlank=%v, flag=%v, returning=0x%02X",
				p.currentScanline, p.currentDot, inVBlank, flag, map[bool]uint8{true: 0x01, false: 0x00}[flag])
		}
		if flag {
			return 0x01
		}
		return 0x00
	case 0x3F: // FRAME_COUNTER_LOW
		return uint8(p.FrameCounter & 0xFF)
	case 0x40: // FRAME_COUNTER_HIGH
		return uint8(p.FrameCounter >> 8)
	case 0x60: // DMA_STATUS
		// Bit 0: DMA active (1=transferring, 0=idle)
		if p.DMAEnabled && p.DMAProgress < p.DMALength {
			return 0x01
		}
		return 0x00
	case 0x61: // DMA_LENGTH_L
		return uint8(p.DMALength & 0xFF)
	case 0x62: // DMA_LENGTH_H
		return uint8(p.DMALength >> 8)
	default:
		return 0
	}
}

// Write8 writes an 8-bit value to PPU registers
func (p *PPU) Write8(offset uint16, value uint8) {
	switch offset {
	// BG0 scroll
	case 0x00: // BG0_SCROLLX_L
		p.BG0.ScrollX = int16((uint16(p.BG0.ScrollX) & 0xFF00) | uint16(value))
	case 0x01: // BG0_SCROLLX_H
		p.BG0.ScrollX = int16((uint16(p.BG0.ScrollX) & 0x00FF) | (uint16(value) << 8))
	case 0x02: // BG0_SCROLLY_L
		p.BG0.ScrollY = int16((uint16(p.BG0.ScrollY) & 0xFF00) | uint16(value))
	case 0x03: // BG0_SCROLLY_H
		p.BG0.ScrollY = int16((uint16(p.BG0.ScrollY) & 0x00FF) | (uint16(value) << 8))

	// BG1 scroll
	case 0x04: // BG1_SCROLLX_L
		p.BG1.ScrollX = int16((uint16(p.BG1.ScrollX) & 0xFF00) | uint16(value))
	case 0x05: // BG1_SCROLLX_H
		p.BG1.ScrollX = int16((uint16(p.BG1.ScrollX) & 0x00FF) | (uint16(value) << 8))
	case 0x06: // BG1_SCROLLY_L
		p.BG1.ScrollY = int16((uint16(p.BG1.ScrollY) & 0xFF00) | uint16(value))
	case 0x07: // BG1_SCROLLY_H
		p.BG1.ScrollY = int16((uint16(p.BG1.ScrollY) & 0x00FF) | (uint16(value) << 8))

	// BG0/BG1 control
	case 0x08: // BG0_CONTROL
		p.BG0.Enabled = (value & 0x01) != 0
		p.BG0.TileSize = (value & 0x02) != 0
	case 0x09: // BG1_CONTROL
		p.BG1.Enabled = (value & 0x01) != 0
		p.BG1.TileSize = (value & 0x02) != 0

	// BG2 scroll
	case 0x0A: // BG2_SCROLLX_L
		p.BG2.ScrollX = int16((uint16(p.BG2.ScrollX) & 0xFF00) | uint16(value))
	case 0x0B: // BG2_SCROLLX_H
		p.BG2.ScrollX = int16((uint16(p.BG2.ScrollX) & 0x00FF) | (uint16(value) << 8))
	case 0x0C: // BG2_SCROLLY_L
		p.BG2.ScrollY = int16((uint16(p.BG2.ScrollY) & 0xFF00) | uint16(value))
	case 0x0D: // BG2_SCROLLY_H
		p.BG2.ScrollY = int16((uint16(p.BG2.ScrollY) & 0x00FF) | (uint16(value) << 8))

	// VRAM access
	case 0x0E: // VRAM_ADDR_L
		p.VRAMAddr = (p.VRAMAddr & 0xFF00) | uint16(value)
	case 0x0F: // VRAM_ADDR_H
		p.VRAMAddr = (p.VRAMAddr & 0x00FF) | (uint16(value) << 8)
	case 0x10: // VRAM_DATA
		// Only log VRAM writes during initialization (first frame) and only first 32 bytes
		if p.Logger != nil && p.FrameCounter == 0 && p.VRAMAddr < 32 {
			p.Logger.LogPPUf(debug.LogLevelDebug, "VRAM_DATA write: addr=0x%04X, value=0x%02X", p.VRAMAddr, value)
		}
		p.VRAM[p.VRAMAddr] = value
		p.VRAMAddr++
		if p.VRAMAddr > 0xFFFF {
			p.VRAMAddr = 0
		}

	// CGRAM access
	case 0x12: // CGRAM_ADDR
		// Only log CGRAM_ADDR during initialization (first frame)
		if p.Logger != nil && p.FrameCounter == 0 && value < 64 {
			paletteIndex := value / 32
			colorIndex := (value / 2) % 16
			p.Logger.LogPPUf(debug.LogLevelDebug, "CGRAM_ADDR write: 0x%02X (palette %d, color %d)", value, paletteIndex, colorIndex)
		}
		p.CGRAMAddr = value
		p.CGRAMWriteLatch = false
	case 0x13: // CGRAM_DATA
		if !p.CGRAMWriteLatch {
			// First write: low byte
			p.CGRAMWriteValue = uint16(value)
			p.CGRAMWriteLatch = true
		} else {
			// Second write: high byte (RGB555 format)
			p.CGRAMWriteValue |= (uint16(value) << 8)
			// Write to CGRAM
			addr := uint16(p.CGRAMAddr) * 2
			if addr < 512 {
				// Only log CGRAM_DATA during initialization (first frame) and only first 20 colors
				if p.Logger != nil && p.FrameCounter == 0 && addr < 40 {
					paletteIndex := p.CGRAMAddr / 32
					colorIndex := (p.CGRAMAddr / 2) % 16
					p.Logger.LogPPUf(debug.LogLevelDebug, "CGRAM_DATA write complete: addr=0x%02X (palette %d, color %d), RGB555=0x%04X",
						p.CGRAMAddr, paletteIndex, colorIndex, p.CGRAMWriteValue)
				}
				// Store in little-endian order: low byte first, high byte second
				p.CGRAM[addr] = uint8(p.CGRAMWriteValue & 0xFF) // Low byte
				p.CGRAM[addr+1] = uint8(p.CGRAMWriteValue >> 8) // High byte
				p.CGRAMAddr++
				if p.CGRAMAddr > 255 {
					p.CGRAMAddr = 0
				}
			}
			p.CGRAMWriteLatch = false
		}

	// OAM access
	case 0x14: // OAM_ADDR
		// Only log OAM_ADDR writes occasionally (every 60 frames) to reduce performance impact
		if p.Logger != nil && p.FrameCounter%60 == 0 && value < 4 {
			p.Logger.LogPPUf(debug.LogLevelDebug, "OAM_ADDR write: 0x%02X (sprite %d), byte index was %d, resetting to 0",
				value, value, p.OAMByteIndex)
		}
		// OAM writes are only allowed during VBlank period (hardware-accurate)
		// During visible rendering (scanlines 0-199), OAM is locked
		// Allow writes if: VBlank period (scanline >= 200) OR frame hasn't started yet OR first frame (initialization)
		// Note: ROM should wait for VBlank before updating sprites to avoid wavy artifacts
		if p.currentScanline < 200 && p.frameStarted && p.FrameCounter > 1 {
			if p.Logger != nil {
				p.Logger.LogPPUf(debug.LogLevelWarning, "OAM_ADDR write ignored during visible rendering (scanline %d)", p.currentScanline)
			}
			return
		}
		p.OAMAddr = value
		if p.OAMAddr > 127 {
			p.OAMAddr = 127
		}
		p.OAMByteIndex = 0 // Reset byte index when setting sprite address
		// Removed frequent logging - only log occasionally above
	case 0x15: // OAM_DATA
		// OAM writes are only allowed during VBlank period (hardware-accurate)
		// During visible rendering (scanlines 0-199), OAM is locked
		// Allow writes if: VBlank period (scanline >= 200) OR frame hasn't started yet OR first frame (initialization)
		// Note: ROM should wait for VBlank before updating sprites to avoid wavy artifacts
		if p.currentScanline < 200 && p.frameStarted && p.FrameCounter > 1 {
			if p.Logger != nil {
				p.Logger.LogPPUf(debug.LogLevelWarning, "OAM_DATA write ignored during visible rendering (scanline %d)", p.currentScanline)
			}
			return
		}
		// Only log OAM_DATA writes occasionally (every 60 frames) and only for first few sprites
		// Log only when completing a sprite (byte 5 = Ctrl) to reduce verbosity
		if p.Logger != nil && p.FrameCounter%60 == 0 && p.OAMByteIndex == 5 && p.OAMAddr < 4 {
			spriteID := p.OAMAddr
			p.Logger.LogPPUf(debug.LogLevelDebug, "OAM_DATA: sprite=%d complete (Ctrl=0x%02X), addr=%d",
				spriteID, value, uint16(p.OAMAddr)*6+uint16(p.OAMByteIndex))
		}
		addr := uint16(p.OAMAddr)*6 + uint16(p.OAMByteIndex)
		if addr < 768 {
			p.OAM[addr] = value
			// Removed frequent logging - only log occasionally above
			p.OAMByteIndex++
			if p.OAMByteIndex >= 6 {
				// Move to next sprite after writing 6 bytes
				p.OAMByteIndex = 0
				p.OAMAddr++
				if p.OAMAddr > 127 {
					p.OAMAddr = 0
				}
			}
			// Debug logging removed for performance - use -log flag to enable PPU logging if needed
		} else {
			if p.Logger != nil {
				p.Logger.LogPPUf(debug.LogLevelWarning, "OAM_DATA write out of bounds: addr=%d (max 767)", addr)
			}
		}

	// Matrix Mode (Legacy - maps to BG0 for backward compatibility)
	case 0x18: // MATRIX_CONTROL (BG0)
		p.MatrixEnabled = (value & 0x01) != 0
		p.MatrixMirrorH = (value & 0x02) != 0
		p.MatrixMirrorV = (value & 0x04) != 0
		p.BG0.MatrixOutsideMode = (value >> 3) & 0x3  // Bits [4:3]
		p.BG0.MatrixDirectColor = (value & 0x20) != 0 // Bit 5
		// Also update BG0 matrix
		p.BG0.MatrixEnabled = (value & 0x01) != 0
		p.BG0.MatrixMirrorH = (value & 0x02) != 0
		p.BG0.MatrixMirrorV = (value & 0x04) != 0
	case 0x19: // MATRIX_A_L (BG0)
		p.MatrixA = int16((uint16(p.MatrixA) & 0xFF00) | uint16(value))
		p.BG0.MatrixA = p.MatrixA
	case 0x1A: // MATRIX_A_H (BG0)
		p.MatrixA = int16((uint16(p.MatrixA) & 0x00FF) | (uint16(value) << 8))
		p.BG0.MatrixA = p.MatrixA
	case 0x1B: // MATRIX_B_L (BG0)
		p.MatrixB = int16((uint16(p.MatrixB) & 0xFF00) | uint16(value))
		p.BG0.MatrixB = p.MatrixB
	case 0x1C: // MATRIX_B_H (BG0)
		p.MatrixB = int16((uint16(p.MatrixB) & 0x00FF) | (uint16(value) << 8))
		p.BG0.MatrixB = p.MatrixB
	case 0x1D: // MATRIX_C_L (BG0)
		p.MatrixC = int16((uint16(p.MatrixC) & 0xFF00) | uint16(value))
		p.BG0.MatrixC = p.MatrixC
	case 0x1E: // MATRIX_C_H (BG0)
		p.MatrixC = int16((uint16(p.MatrixC) & 0x00FF) | (uint16(value) << 8))
		p.BG0.MatrixC = p.MatrixC
	case 0x1F: // MATRIX_D_L (BG0)
		p.MatrixD = int16((uint16(p.MatrixD) & 0xFF00) | uint16(value))
		p.BG0.MatrixD = p.MatrixD
	case 0x20: // MATRIX_D_H (BG0)
		p.MatrixD = int16((uint16(p.MatrixD) & 0x00FF) | (uint16(value) << 8))
		p.BG0.MatrixD = p.MatrixD

	// BG2/BG3 control
	case 0x21: // BG2_CONTROL
		p.BG2.Enabled = (value & 0x01) != 0
		p.BG2.TileSize = (value & 0x02) != 0
	case 0x22: // BG3_SCROLLX_L
		p.BG3.ScrollX = int16((uint16(p.BG3.ScrollX) & 0xFF00) | uint16(value))
	case 0x23: // BG3_SCROLLX_H
		p.BG3.ScrollX = int16((uint16(p.BG3.ScrollX) & 0x00FF) | (uint16(value) << 8))
	case 0x24: // BG3_SCROLLY_L
		p.BG3.ScrollY = int16((uint16(p.BG3.ScrollY) & 0xFF00) | uint16(value))
	case 0x25: // BG3_SCROLLY_H
		p.BG3.ScrollY = int16((uint16(p.BG3.ScrollY) & 0x00FF) | (uint16(value) << 8))
	case 0x26: // BG3_CONTROL
		p.BG3.Enabled = (value & 0x01) != 0
		p.BG3.TileSize = (value & 0x02) != 0

	// Matrix center (BG0)
	case 0x27: // MATRIX_CENTER_X_L (BG0)
		p.MatrixCenterX = int16((uint16(p.MatrixCenterX) & 0xFF00) | uint16(value))
		p.BG0.MatrixCenterX = p.MatrixCenterX
	case 0x28: // MATRIX_CENTER_X_H (BG0)
		p.MatrixCenterX = int16((uint16(p.MatrixCenterX) & 0x00FF) | (uint16(value) << 8))
		p.BG0.MatrixCenterX = p.MatrixCenterX
	case 0x29: // MATRIX_CENTER_Y_L (BG0)
		p.MatrixCenterY = int16((uint16(p.MatrixCenterY) & 0xFF00) | uint16(value))
		p.BG0.MatrixCenterY = p.MatrixCenterY
	case 0x2A: // MATRIX_CENTER_Y_H (BG0)
		p.MatrixCenterY = int16((uint16(p.MatrixCenterY) & 0x00FF) | (uint16(value) << 8))
		p.BG0.MatrixCenterY = p.MatrixCenterY

	// BG1 Matrix Mode (per-layer transformation)
	case 0x2B: // BG1_MATRIX_CONTROL
		p.BG1.MatrixEnabled = (value & 0x01) != 0
		p.BG1.MatrixMirrorH = (value & 0x02) != 0
		p.BG1.MatrixMirrorV = (value & 0x04) != 0
		p.BG1.MatrixOutsideMode = (value >> 3) & 0x3  // Bits [4:3]
		p.BG1.MatrixDirectColor = (value & 0x20) != 0 // Bit 5
	case 0x2C: // BG1_MATRIX_A_L
		p.BG1.MatrixA = int16((uint16(p.BG1.MatrixA) & 0xFF00) | uint16(value))
	case 0x2D: // BG1_MATRIX_A_H
		p.BG1.MatrixA = int16((uint16(p.BG1.MatrixA) & 0x00FF) | (uint16(value) << 8))
	case 0x2E: // BG1_MATRIX_B_L
		p.BG1.MatrixB = int16((uint16(p.BG1.MatrixB) & 0xFF00) | uint16(value))
	case 0x2F: // BG1_MATRIX_B_H
		p.BG1.MatrixB = int16((uint16(p.BG1.MatrixB) & 0x00FF) | (uint16(value) << 8))
	case 0x30: // BG1_MATRIX_C_L
		p.BG1.MatrixC = int16((uint16(p.BG1.MatrixC) & 0xFF00) | uint16(value))
	case 0x31: // BG1_MATRIX_C_H
		p.BG1.MatrixC = int16((uint16(p.BG1.MatrixC) & 0x00FF) | (uint16(value) << 8))
	case 0x32: // BG1_MATRIX_D_L
		p.BG1.MatrixD = int16((uint16(p.BG1.MatrixD) & 0xFF00) | uint16(value))
	case 0x33: // BG1_MATRIX_D_H
		p.BG1.MatrixD = int16((uint16(p.BG1.MatrixD) & 0x00FF) | (uint16(value) << 8))
	case 0x34: // BG1_MATRIX_CENTER_X_L
		p.BG1.MatrixCenterX = int16((uint16(p.BG1.MatrixCenterX) & 0xFF00) | uint16(value))
	case 0x35: // BG1_MATRIX_CENTER_X_H
		p.BG1.MatrixCenterX = int16((uint16(p.BG1.MatrixCenterX) & 0x00FF) | (uint16(value) << 8))
	case 0x36: // BG1_MATRIX_CENTER_Y_L
		p.BG1.MatrixCenterY = int16((uint16(p.BG1.MatrixCenterY) & 0xFF00) | uint16(value))
	case 0x37: // BG1_MATRIX_CENTER_Y_H
		p.BG1.MatrixCenterY = int16((uint16(p.BG1.MatrixCenterY) & 0x00FF) | (uint16(value) << 8))

	// BG2 Matrix Mode
	case 0x38: // BG2_MATRIX_CONTROL
		p.BG2.MatrixEnabled = (value & 0x01) != 0
		p.BG2.MatrixMirrorH = (value & 0x02) != 0
		p.BG2.MatrixMirrorV = (value & 0x04) != 0
		p.BG2.MatrixOutsideMode = (value >> 3) & 0x3  // Bits [4:3]
		p.BG2.MatrixDirectColor = (value & 0x20) != 0 // Bit 5
	case 0x39: // BG2_MATRIX_A_L
		p.BG2.MatrixA = int16((uint16(p.BG2.MatrixA) & 0xFF00) | uint16(value))
	case 0x3A: // BG2_MATRIX_A_H
		p.BG2.MatrixA = int16((uint16(p.BG2.MatrixA) & 0x00FF) | (uint16(value) << 8))
	case 0x3B: // BG2_MATRIX_B_L
		p.BG2.MatrixB = int16((uint16(p.BG2.MatrixB) & 0xFF00) | uint16(value))
	case 0x3C: // BG2_MATRIX_B_H
		p.BG2.MatrixB = int16((uint16(p.BG2.MatrixB) & 0x00FF) | (uint16(value) << 8))
	case 0x3D: // BG2_MATRIX_C_L
		p.BG2.MatrixC = int16((uint16(p.BG2.MatrixC) & 0xFF00) | uint16(value))
	case 0x3E: // BG2_MATRIX_C_H
		p.BG2.MatrixC = int16((uint16(p.BG2.MatrixC) & 0x00FF) | (uint16(value) << 8))
	case 0x3F: // BG2_MATRIX_D_L
		p.BG2.MatrixD = int16((uint16(p.BG2.MatrixD) & 0xFF00) | uint16(value))
	case 0x40: // BG2_MATRIX_D_H
		p.BG2.MatrixD = int16((uint16(p.BG2.MatrixD) & 0x00FF) | (uint16(value) << 8))
	case 0x41: // BG2_MATRIX_CENTER_X_L
		p.BG2.MatrixCenterX = int16((uint16(p.BG2.MatrixCenterX) & 0xFF00) | uint16(value))
	case 0x42: // BG2_MATRIX_CENTER_X_H
		p.BG2.MatrixCenterX = int16((uint16(p.BG2.MatrixCenterX) & 0x00FF) | (uint16(value) << 8))
	case 0x43: // BG2_MATRIX_CENTER_Y_L
		p.BG2.MatrixCenterY = int16((uint16(p.BG2.MatrixCenterY) & 0xFF00) | uint16(value))
	case 0x44: // BG2_MATRIX_CENTER_Y_H
		p.BG2.MatrixCenterY = int16((uint16(p.BG2.MatrixCenterY) & 0x00FF) | (uint16(value) << 8))

	// BG3 Matrix Mode
	case 0x45: // BG3_MATRIX_CONTROL
		p.BG3.MatrixEnabled = (value & 0x01) != 0
		p.BG3.MatrixMirrorH = (value & 0x02) != 0
		p.BG3.MatrixMirrorV = (value & 0x04) != 0
		p.BG3.MatrixOutsideMode = (value >> 3) & 0x3  // Bits [4:3]
		p.BG3.MatrixDirectColor = (value & 0x20) != 0 // Bit 5
	case 0x46: // BG3_MATRIX_A_L
		p.BG3.MatrixA = int16((uint16(p.BG3.MatrixA) & 0xFF00) | uint16(value))
	case 0x47: // BG3_MATRIX_A_H
		p.BG3.MatrixA = int16((uint16(p.BG3.MatrixA) & 0x00FF) | (uint16(value) << 8))
	case 0x48: // BG3_MATRIX_B_L
		p.BG3.MatrixB = int16((uint16(p.BG3.MatrixB) & 0xFF00) | uint16(value))
	case 0x49: // BG3_MATRIX_B_H
		p.BG3.MatrixB = int16((uint16(p.BG3.MatrixB) & 0x00FF) | (uint16(value) << 8))
	case 0x4A: // BG3_MATRIX_C_L
		p.BG3.MatrixC = int16((uint16(p.BG3.MatrixC) & 0xFF00) | uint16(value))
	case 0x4B: // BG3_MATRIX_C_H
		p.BG3.MatrixC = int16((uint16(p.BG3.MatrixC) & 0x00FF) | (uint16(value) << 8))
	case 0x4C: // BG3_MATRIX_D_L
		p.BG3.MatrixD = int16((uint16(p.BG3.MatrixD) & 0xFF00) | uint16(value))
	case 0x4D: // BG3_MATRIX_D_H
		p.BG3.MatrixD = int16((uint16(p.BG3.MatrixD) & 0x00FF) | (uint16(value) << 8))
	case 0x4E: // BG3_MATRIX_CENTER_X_L
		p.BG3.MatrixCenterX = int16((uint16(p.BG3.MatrixCenterX) & 0xFF00) | uint16(value))
	case 0x4F: // BG3_MATRIX_CENTER_X_H
		p.BG3.MatrixCenterX = int16((uint16(p.BG3.MatrixCenterX) & 0x00FF) | (uint16(value) << 8))
	case 0x50: // BG3_MATRIX_CENTER_Y_L
		p.BG3.MatrixCenterY = int16((uint16(p.BG3.MatrixCenterY) & 0xFF00) | uint16(value))
	case 0x51: // BG3_MATRIX_CENTER_Y_H
		p.BG3.MatrixCenterY = int16((uint16(p.BG3.MatrixCenterY) & 0x00FF) | (uint16(value) << 8))

	// Windowing (0x52-0x5C)
	case 0x52: // WINDOW0_LEFT
		p.Window0.Left = value
	case 0x53: // WINDOW0_RIGHT
		p.Window0.Right = value
	case 0x54: // WINDOW0_TOP
		p.Window0.Top = value
	case 0x55: // WINDOW0_BOTTOM
		p.Window0.Bottom = value
	case 0x56: // WINDOW1_LEFT
		p.Window1.Left = value
	case 0x57: // WINDOW1_RIGHT
		p.Window1.Right = value
	case 0x58: // WINDOW1_TOP
		p.Window1.Top = value
	case 0x59: // WINDOW1_BOTTOM
		p.Window1.Bottom = value
	case 0x5A: // WINDOW_CONTROL
		p.WindowControl = value
	case 0x5B: // WINDOW_MAIN_ENABLE
		p.WindowMainEnable = value
	case 0x5C: // WINDOW_SUB_ENABLE
		p.WindowSubEnable = value

	// HDMA (0x5D-0x5F)
	case 0x5D: // HDMA_CONTROL
		// Bit 0: HDMA enable
		// Bits 1-4: Layer enable for scroll HDMA (BG0-BG3)
		// Bits 5-7: Reserved for future use (matrix HDMA is always enabled if layer has matrix enabled)
		p.HDMAEnabled = (value & 0x01) != 0
		p.HDMAControl = value
	case 0x5E: // HDMA_TABLE_BASE_L
		p.HDMATableBase = (p.HDMATableBase & 0xFF00) | uint16(value)
	case 0x5F: // HDMA_TABLE_BASE_H
		p.HDMATableBase = (p.HDMATableBase & 0x00FF) | (uint16(value) << 8)

	// DMA registers (0x8060-0x8067, but offset is relative to 0x8000, so 0x60-0x67)
	case 0x60: // DMA_CONTROL
		// Bit 0: Enable DMA (1=start transfer, 0=disable)
		// Bit 1: Mode (0=copy, 1=fill)
		// Bits [3:2]: Destination type (0=VRAM, 1=CGRAM, 2=OAM)
		if (value & 0x01) != 0 {
			// Start DMA transfer
			p.DMAEnabled = true
			p.DMAMode = (value >> 1) & 0x01
			p.DMADestType = (value >> 2) & 0x3
			// Initialize cycle-accurate DMA state
			p.DMAProgress = 0
			p.DMACurrentSrc = p.DMASourceOffset
			p.DMACurrentDest = p.DMADestAddr
			// For fill mode, read fill value once at start
			if p.DMAMode == 1 && p.MemoryReader != nil {
				p.DMAFillValue = p.MemoryReader(p.DMASourceBank, p.DMASourceOffset)
			}
			// Note: DMA will execute incrementally during StepPPU (cycle-accurate)
		} else {
			// Disable DMA (abort current transfer)
			p.DMAEnabled = false
			p.DMAProgress = 0
		}
	case 0x61: // DMA_SOURCE_BANK
		p.DMASourceBank = value
	case 0x62: // DMA_SOURCE_OFFSET_L
		p.DMASourceOffset = (p.DMASourceOffset & 0xFF00) | uint16(value)
	case 0x63: // DMA_SOURCE_OFFSET_H
		p.DMASourceOffset = (p.DMASourceOffset & 0x00FF) | (uint16(value) << 8)
	case 0x64: // DMA_DEST_ADDR_L
		p.DMADestAddr = (p.DMADestAddr & 0xFF00) | uint16(value)
	case 0x65: // DMA_DEST_ADDR_H
		p.DMADestAddr = (p.DMADestAddr & 0x00FF) | (uint16(value) << 8)
	case 0x66: // DMA_LENGTH_L
		p.DMALength = (p.DMALength & 0xFF00) | uint16(value)
	case 0x67: // DMA_LENGTH_H
		p.DMALength = (p.DMALength & 0x00FF) | (uint16(value) << 8)
	default:
		// Unknown register, ignore
	}
}

// stepDMA executes one cycle of DMA transfer (transfers one byte per cycle)
// This is called from StepPPU to make DMA cycle-accurate
func (p *PPU) stepDMA() {
	if !p.DMAEnabled || p.DMAProgress >= p.DMALength {
		// DMA not active or already complete
		if p.DMAProgress >= p.DMALength {
			// DMA just completed
			p.DMAEnabled = false
			p.DMAProgress = 0
		}
		return
	}

	if p.MemoryReader == nil {
		// No memory reader, abort DMA
		p.DMAEnabled = false
		p.DMAProgress = 0
		return
	}

	// Transfer one byte
	var data uint8
	if p.DMAMode == 1 {
		// Fill mode: use fill value for all bytes
		data = p.DMAFillValue
	} else {
		// Copy mode: read from source
		data = p.MemoryReader(p.DMASourceBank, p.DMACurrentSrc)
		p.DMACurrentSrc++
	}

	// Write to destination
	switch p.DMADestType {
	case 0: // VRAM
		destAddr := uint32(p.DMACurrentDest)
		if destAddr < 65536 {
			p.VRAM[destAddr] = data
		}
		p.DMACurrentDest++
	case 1: // CGRAM
		// CGRAM is 16-bit (RGB555), so we need to handle it specially
		// For simplicity, write as 8-bit (low byte only)
		addr := p.DMACurrentDest & 0x1FF // Wrap at 512 bytes
		p.CGRAM[addr] = data
		p.DMACurrentDest++
	case 2: // OAM
		addr := p.DMACurrentDest & 0x2FF // Wrap at 768 bytes
		p.OAM[addr] = data
		p.DMACurrentDest++
	}

	// Advance progress
	p.DMAProgress++

	// Check if DMA is complete
	if p.DMAProgress >= p.DMALength {
		p.DMAEnabled = false
		p.DMAProgress = 0
	}
}

// executeDMA executes a DMA transfer immediately (legacy function, kept for compatibility)
// This is now a wrapper that calls stepDMA until complete
// Note: This is used by tests. For cycle-accurate operation, DMA should be stepped
// incrementally via stepDMA() during StepPPU.
func (p *PPU) executeDMA() {
	if !p.DMAEnabled || p.MemoryReader == nil {
		return
	}

	// Initialize DMA state if not already initialized (check if we need to reset)
	// If DMAProgress is 0 and we haven't started, initialize
	if p.DMAProgress == 0 {
		p.DMACurrentSrc = p.DMASourceOffset
		p.DMACurrentDest = p.DMADestAddr
		// For fill mode, read fill value once at start
		if p.DMAMode == 1 {
			p.DMAFillValue = p.MemoryReader(p.DMASourceBank, p.DMASourceOffset)
		}
	}

	// Execute all remaining bytes (for compatibility with tests)
	for p.DMAProgress < p.DMALength {
		p.stepDMA()
	}
}

// Read16 reads a 16-bit value from PPU registers
func (p *PPU) Read16(offset uint16) uint16 {
	low := p.Read8(offset)
	high := p.Read8(offset + 1)
	return uint16(low) | (uint16(high) << 8)
}

// Write16 writes a 16-bit value to PPU registers
func (p *PPU) Write16(offset uint16, value uint16) {
	p.Write8(offset, uint8(value&0xFF))
	p.Write8(offset+1, uint8(value>>8))
}

// RenderFrame renders a complete frame (DEPRECATED - use StepPPU for clock-driven operation)
// This function is kept for compatibility but is NOT used in clock-driven mode
// Clock-driven mode uses StepPPU() which calls startFrame() and endFrame() automatically
func (p *PPU) RenderFrame() {
	// DEPRECATED: This is the old frame-based rendering function
	// In clock-driven mode, PPU rendering happens via StepPPU() -> stepDot() -> renderDot()
	// This function should not be called in clock-driven mode

	// Set VBlank flag at start of frame (hardware-accurate synchronization)
	// This signal indicates the start of vertical blanking period
	// ROMs can wait for this signal to synchronize with frame boundaries
	p.VBlankFlag = true

	// Increment frame counter at start of frame (for ROM timing)
	p.FrameCounter++

	// Clear output buffer
	for i := range p.OutputBuffer {
		p.OutputBuffer[i] = 0x000000 // Black
	}

	// Debug: Print CGRAM contents once per 60 frames
	p.debugFrameCount++
	if p.debugFrameCount == 60 && p.Logger != nil {
		// Log CGRAM debug info
		for i := 0; i < 4; i++ {
			addr := i * 2
			low := p.CGRAM[addr]
			high := p.CGRAM[addr+1]
			color := p.getColorFromCGRAM(0, uint8(i))
			r := (color >> 16) & 0xFF
			g := (color >> 8) & 0xFF
			b := color & 0xFF
			p.Logger.LogPPUf(debug.LogLevelDebug,
				"CGRAM palette 0, color %d: CGRAM[%d]=0x%02X, CGRAM[%d]=0x%02X -> RGB(%d,%d,%d) = 0x%06X",
				i, addr, low, addr+1, high, r, g, b, color)
		}
		// Log first 10 pixels
		for i := 0; i < 10; i++ {
			color := p.OutputBuffer[i]
			r := (color >> 16) & 0xFF
			g := (color >> 8) & 0xFF
			b := color & 0xFF
			p.Logger.LogPPUf(debug.LogLevelDebug,
				"Output buffer pixel %d: 0x%06X (RGB %d,%d,%d)",
				i, color, r, g, b)
		}
		// Log BG0 state
		p.Logger.LogPPUf(debug.LogLevelDebug,
			"BG0 state: Enabled=%v, ScrollX=%d, ScrollY=%d",
			p.BG0.Enabled, p.BG0.ScrollX, p.BG0.ScrollY)
		// Log VRAM entries
		p.Logger.LogPPUf(debug.LogLevelDebug,
			"VRAM[0x4000-0x4003] (first tilemap entry): 0x%02X 0x%02X 0x%02X 0x%02X",
			p.VRAM[0x4000], p.VRAM[0x4001], p.VRAM[0x4002], p.VRAM[0x4003])
		p.Logger.LogPPUf(debug.LogLevelDebug,
			"VRAM[0x0000-0x0003] (first tile data): 0x%02X 0x%02X 0x%02X 0x%02X",
			p.VRAM[0x0000], p.VRAM[0x0001], p.VRAM[0x0002], p.VRAM[0x0003])
		p.debugFrameCount = 0
	}

	// Render background layers (BG3 → BG0, back to front)
	if p.BG3.Enabled {
		p.renderBackgroundLayer(3)
	}
	if p.BG2.Enabled {
		p.renderBackgroundLayer(2)
	}
	if p.BG1.Enabled {
		p.renderBackgroundLayer(1)
	}
	if p.BG0.Enabled {
		if p.MatrixEnabled {
			p.renderMatrixMode()
		} else {
			p.renderBackgroundLayer(0)
		}
	}

	// Render sprites
	p.renderSprites()

	// Debug: Log sprite 0 OAM data
	if p.debugFrameCount%60 == 0 && p.Logger != nil {
		p.Logger.LogPPUf(debug.LogLevelDebug,
			"Sprite 0 OAM: OAM[0-5]=0x%02X 0x%02X 0x%02X 0x%02X 0x%02X 0x%02X",
			p.OAM[0], p.OAM[1], p.OAM[2], p.OAM[3], p.OAM[4], p.OAM[5])
		spriteX := int(p.OAM[0])
		if (p.OAM[1] & 0x01) != 0 {
			spriteX |= 0xFFFFFF00
		}
		spriteY := int(p.OAM[2])
		tileIndex := uint8(p.OAM[3])
		attributes := uint8(p.OAM[4])
		control := uint8(p.OAM[5])
		paletteIndex := attributes & 0x0F
		enabled := (control & 0x01) != 0
		p.Logger.LogPPUf(debug.LogLevelDebug,
			"Sprite 0: X=%d, Y=%d, Tile=%d, Palette=%d, Enabled=%v",
			spriteX, spriteY, tileIndex, paletteIndex, enabled)
		tileAddr := uint16(tileIndex) * 32
		p.Logger.LogPPUf(debug.LogLevelDebug,
			"Sprite 0 tile data at VRAM[%d]: 0x%02X 0x%02X 0x%02X 0x%02X",
			tileAddr, p.VRAM[tileAddr], p.VRAM[tileAddr+1],
			p.VRAM[tileAddr+2], p.VRAM[tileAddr+3])
		if uint16(paletteIndex)*16+1 < 256 {
			addr := (uint16(paletteIndex)*16 + 1) * 2
			if addr < 512 {
				low := p.CGRAM[addr]
				high := p.CGRAM[addr+1]
				color := p.getColorFromCGRAM(paletteIndex, 1)
				p.Logger.LogPPUf(debug.LogLevelDebug,
					"Sprite 0 CGRAM palette %d, color 1: CGRAM[%d]=0x%02X, CGRAM[%d]=0x%02X -> RGB(0x%06X)",
					paletteIndex, addr, low, addr+1, high, color)
			}
		}
	}
}

// renderBackgroundLayer renders a background layer
func (p *PPU) renderBackgroundLayer(layerNum int) {
	// Get layer
	var layer *BackgroundLayer
	switch layerNum {
	case 0:
		layer = &p.BG0
	case 1:
		layer = &p.BG1
	case 2:
		layer = &p.BG2
	case 3:
		layer = &p.BG3
	default:
		return
	}

	if !layer.Enabled {
		return
	}

	// Tile size: 8x8 or 16x16
	tileSize := 8
	if layer.TileSize {
		tileSize = 16
	}

	// Tilemap is 32x32 tiles
	tilemapWidth := 32
	tilemapHeight := 32

	// Render each pixel
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			// Check windowing
			if !p.isPixelInWindow(x, y, layerNum) {
				continue
			}

			// Calculate tilemap coordinates with scroll
			// Screen pixel (x, y) -> world pixel (worldX, worldY)
			worldX := int(x) + int(layer.ScrollX)
			worldY := int(y) + int(layer.ScrollY)

			// Wrap coordinates (tilemap repeats)
			tilemapPixelWidth := tilemapWidth * tileSize
			tilemapPixelHeight := tilemapHeight * tileSize
			worldX = worldX % tilemapPixelWidth
			if worldX < 0 {
				worldX += tilemapPixelWidth
			}
			worldY = worldY % tilemapPixelHeight
			if worldY < 0 {
				worldY += tilemapPixelHeight
			}

			// Calculate which tile this pixel is in
			tileX := worldX / tileSize
			tileY := worldY / tileSize

			// Calculate pixel position within tile
			pixelXInTile := worldX % tileSize
			pixelYInTile := worldY % tileSize

			// Read tilemap entry (2 bytes per tile)
			// Tilemap entry at (tileX, tileY) = tilemapBase + (tileY * 32 + tileX) * 2
			// Default tilemap base: 0x4000 for BG0 (can be configured later)
			tilemapBase := uint16(0x4000) // Default tilemap base
			if layer.TilemapBase != 0 {
				tilemapBase = layer.TilemapBase
			}
			tilemapOffset := uint16((tileY*tilemapWidth + tileX) * 2)
			if uint32(tilemapBase)+uint32(tilemapOffset) >= 65536 {
				// Out of bounds, render black
				p.OutputBuffer[y*320+x] = 0x000000
				continue
			}
			tilemapEntryAddr := tilemapBase + tilemapOffset

			// Read tile index and attributes
			tileIndex := uint8(p.VRAM[tilemapEntryAddr])
			attributes := uint8(p.VRAM[tilemapEntryAddr+1])
			paletteIndex := attributes & 0x0F
			flipX := (attributes & 0x10) != 0
			flipY := (attributes & 0x20) != 0

			// Apply flip
			if flipX {
				pixelXInTile = tileSize - 1 - pixelXInTile
			}
			if flipY {
				pixelYInTile = tileSize - 1 - pixelYInTile
			}

			// Read tile data (4bpp = 2 pixels per byte)
			// Tile data starts at VRAM offset = tileIndex * (tileSize * tileSize / 2)
			tileDataOffset := uint16(tileIndex) * uint16(tileSize*tileSize/2)
			// Pixel position in tile: pixelYInTile * tileSize + pixelXInTile
			pixelOffsetInTile := pixelYInTile*tileSize + pixelXInTile
			// Byte offset in tile data
			byteOffsetInTile := pixelOffsetInTile / 2
			// Which pixel in the byte (0 = upper 4 bits, 1 = lower 4 bits)
			pixelInByte := pixelOffsetInTile % 2

			if uint32(tileDataOffset)+uint32(byteOffsetInTile) >= 65536 {
				// Out of bounds, render black
				p.OutputBuffer[y*320+x] = 0x000000
				continue
			}
			tileDataAddr := tileDataOffset + uint16(byteOffsetInTile)

			// Read pixel color index (4 bits = 0-15)
			tileByte := p.VRAM[tileDataAddr]
			var colorIndex uint8
			if pixelInByte == 0 {
				colorIndex = (tileByte >> 4) & 0x0F // Upper 4 bits
			} else {
				colorIndex = tileByte & 0x0F // Lower 4 bits
			}

			// Look up color in CGRAM
			// Note: Color index 0 is NOT transparent for backgrounds (only for sprites)
			// Backgrounds always render, even if color index is 0
			color := p.getColorFromCGRAM(paletteIndex, colorIndex)
			p.OutputBuffer[y*320+x] = color
		}
	}
}

// renderMatrixMode renders Matrix Mode (Mode 7-style)
// NOTE: This is the old frame-based rendering function (deprecated)
// Clock-driven mode uses renderDotMatrixMode() instead
func (p *PPU) renderMatrixMode() {
	// Clock-driven mode uses renderDotMatrixMode() per-pixel
	// This function is kept for compatibility but should not be called in clock-driven mode
	p.renderBackgroundLayer(0)
}

// renderSprites renders all sprites
func (p *PPU) renderSprites() {
	// Render sprites (128 max)
	for spriteIndex := 0; spriteIndex < 128; spriteIndex++ {
		// OAM entry is 6 bytes per sprite
		oamAddr := spriteIndex * 6

		// Read sprite data
		// Byte 0: X position (low byte, unsigned)
		xLow := uint8(p.OAM[oamAddr])
		// Byte 1: X position (high byte, bit 0 only, sign extends)
		xHigh := uint8(p.OAM[oamAddr+1])
		// Combine X position: 9-bit signed value
		// Low 8 bits from byte 0, sign bit from bit 0 of byte 1
		spriteX := int(xLow)
		if (xHigh & 0x01) != 0 {
			// Sign extend (negative value)
			spriteX |= 0xFFFFFF00
		}

		// Byte 2: Y position (8-bit, 0-255)
		spriteY := int(p.OAM[oamAddr+2])

		// Byte 3: Tile index
		tileIndex := uint8(p.OAM[oamAddr+3])

		// Byte 4: Attributes
		attributes := uint8(p.OAM[oamAddr+4])
		paletteIndex := attributes & 0x0F
		flipX := (attributes & 0x10) != 0
		flipY := (attributes & 0x20) != 0
		_ = (attributes >> 6) & 0x3 // priority (not used yet)

		// Byte 5: Control
		control := uint8(p.OAM[oamAddr+5])
		enabled := (control & 0x01) != 0
		tileSize16 := (control & 0x02) != 0

		// Log sprite 0 rendering state (for debugging blinking)
		if spriteIndex == 0 && p.Logger != nil && p.currentScanline == 0 && p.currentDot == 0 {
			p.Logger.LogPPUf(debug.LogLevelDebug,
				"SPRITE0_RENDER: Enabled=%v X=%d Y=%d Tile=%d Palette=%d Control=0x%02X",
				enabled, spriteX, spriteY, tileIndex, paletteIndex, control)
		}

		if !enabled {
			continue
		}

		// Sprite size
		spriteSize := 8
		if tileSize16 {
			spriteSize = 16
		}

		// Render sprite pixels
		for py := 0; py < spriteSize; py++ {
			for px := 0; px < spriteSize; px++ {
				// Calculate screen position
				screenX := spriteX + px
				screenY := spriteY + py

				// Check bounds
				if screenX < 0 || screenX >= 320 || screenY < 0 || screenY >= 200 {
					continue
				}

				// Apply flip
				tileX := px
				tileY := py
				if flipX {
					tileX = spriteSize - 1 - tileX
				}
				if flipY {
					tileY = spriteSize - 1 - tileY
				}

				// Read tile data (4bpp = 2 pixels per byte)
				tileDataOffset := uint16(tileIndex) * uint16(spriteSize*spriteSize/2)
				pixelOffsetInTile := tileY*spriteSize + tileX
				byteOffsetInTile := pixelOffsetInTile / 2
				pixelInByte := pixelOffsetInTile % 2

				if uint32(tileDataOffset)+uint32(byteOffsetInTile) >= 65536 {
					continue
				}
				tileDataAddr := tileDataOffset + uint16(byteOffsetInTile)

				// Read pixel color index
				tileByte := p.VRAM[tileDataAddr]
				var colorIndex uint8
				if pixelInByte == 0 {
					colorIndex = (tileByte >> 4) & 0x0F
				} else {
					colorIndex = tileByte & 0x0F
				}

				// Color index 0 is transparent for sprites
				if colorIndex == 0 {
					continue
				}

				// Look up color and render
				color := p.getColorFromCGRAM(paletteIndex, colorIndex)
				p.OutputBuffer[screenY*320+screenX] = color
			}
		}
	}
}

// isPixelInWindow checks if a pixel is inside the window
func (p *PPU) isPixelInWindow(x, y, layerNum int) bool {
	// Check if windowing is enabled for this layer
	if (p.WindowMainEnable & (1 << layerNum)) == 0 {
		return true // No windowing
	}

	// Check window logic
	// Window bounds: Left/Right/Top/Bottom are 8-bit values (0-255)
	// If windowing is enabled, check if pixel is inside window bounds
	// If Right is 0 and Left is 0, assume window is not active
	win0Inside := true // Default to inside if window not configured
	if p.Window0.Right > 0 || p.Window0.Left > 0 {
		// Window is configured, check bounds
		win0Inside = x >= int(p.Window0.Left) && x <= int(p.Window0.Right) &&
			y >= int(p.Window0.Top) && y <= int(p.Window0.Bottom)
	}

	win1Inside := true // Default to inside if window not configured
	if p.Window1.Right > 0 || p.Window1.Left > 0 {
		// Window is configured, check bounds
		win1Inside = x >= int(p.Window1.Left) && x <= int(p.Window1.Right) &&
			y >= int(p.Window1.Top) && y <= int(p.Window1.Bottom)
	}

	logic := (p.WindowControl >> 2) & 0x3
	switch logic {
	case 0: // OR
		return win0Inside || win1Inside
	case 1: // AND
		return win0Inside && win1Inside
	case 2: // XOR
		return win0Inside != win1Inside
	case 3: // XNOR
		return win0Inside == win1Inside
	}

	return true
}

// getColorFromCGRAM gets a color from CGRAM
func (p *PPU) getColorFromCGRAM(paletteIndex, colorIndex uint8) uint32 {
	addr := (uint16(paletteIndex)*16 + uint16(colorIndex)) * 2
	if addr >= 512 {
		return 0x000000
	}

	// Read RGB555 color
	// CGRAM stores colors in little-endian order: low byte first, high byte second
	low := p.CGRAM[addr]    // Low byte is stored first
	high := p.CGRAM[addr+1] // High byte is stored second

	// Convert RGB555 to RGB888
	// RGB555 format: Low byte = GGGGG BBBBB, High byte = 0 RRRRR GG
	// Extract components
	// R: bits 10-14 from high byte (bits 2-6)
	r := uint32((high & 0x7C) >> 2)
	// G: bits 5-9, split between high (bits 0-1) and low (bits 5-7)
	g := uint32(((high & 0x03) << 3) | ((low & 0xE0) >> 5))
	// B: bits 0-4 from low byte
	b := uint32(low & 0x1F)

	// Scale to 8 bits
	r = (r * 255) / 31
	g = (g * 255) / 31
	b = (b * 255) / 31

	return (r << 16) | (g << 8) | b
}
