package ppu

import (
	"testing"

	"nitro-core-dx/internal/debug"
)

// TestSpriteRendering tests basic sprite rendering
func TestSpriteRendering(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Set up a white color in palette 1, color 1
	ppu.CGRAM[0x11*2] = 0xFF     // Low byte (RGB555)
	ppu.CGRAM[0x11*2+1] = 0x7F    // High byte

	// Initialize VRAM with white tile (tile 0)
	// 16x16 tile = 128 bytes, fill with 0x11 (color index 1)
	for i := 0; i < 128; i++ {
		ppu.VRAM[i] = 0x11
	}

	// Set up sprite 0: position (100, 100), tile 0, palette 1, enabled, 16x16
	ppu.OAM[0] = 100   // X low
	ppu.OAM[1] = 0x00  // X high
	ppu.OAM[2] = 100   // Y
	ppu.OAM[3] = 0x00  // Tile index
	ppu.OAM[4] = 0x01  // Attributes (palette 1 = bits [3:0] = 0x01)
	ppu.OAM[5] = 0x03  // Control (enable + 16x16)

	// Render a single dot where the sprite should be
	ppu.renderDot(100, 100)

	// Check if sprite was rendered (should be white, not black)
	color := ppu.OutputBuffer[100*320+100]
	if color == 0x000000 {
		t.Errorf("Sprite not rendered at (100, 100), got color 0x%06X (black)", color)
	}

	// Check a pixel outside sprite bounds (should be black)
	ppu.renderDot(50, 50)
	color = ppu.OutputBuffer[50*320+50]
	if color != 0x000000 {
		t.Errorf("Expected black at (50, 50), got color 0x%06X", color)
	}
}

// TestOAMWrite tests OAM write functionality
func TestOAMWrite(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Set OAM address to sprite 0
	ppu.Write8(0x14, 0x00) // OAM_ADDR = 0

	// Write sprite data
	ppu.Write8(0x15, 100) // X low
	ppu.Write8(0x15, 0x00) // X high
	ppu.Write8(0x15, 100) // Y
	ppu.Write8(0x15, 0x00) // Tile
	ppu.Write8(0x15, 0x10) // Attributes
	ppu.Write8(0x15, 0x03) // Control

	// Verify sprite data was written correctly
	if ppu.OAM[0] != 100 {
		t.Errorf("OAM[0] (X low) = %d, expected 100", ppu.OAM[0])
	}
	if ppu.OAM[2] != 100 {
		t.Errorf("OAM[2] (Y) = %d, expected 100", ppu.OAM[2])
	}
	if ppu.OAM[5] != 0x03 {
		t.Errorf("OAM[5] (Control) = 0x%02X, expected 0x03", ppu.OAM[5])
	}
}

// TestVRAMWrite tests VRAM write functionality
func TestVRAMWrite(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Set VRAM address to 0
	ppu.Write8(0x0E, 0x00) // VRAM_ADDR_L
	ppu.Write8(0x0F, 0x00) // VRAM_ADDR_H

	// Write test data
	ppu.Write8(0x10, 0x11) // VRAM_DATA

	// Verify VRAM was written
	if ppu.VRAM[0] != 0x11 {
		t.Errorf("VRAM[0] = 0x%02X, expected 0x11", ppu.VRAM[0])
	}

	// Verify address auto-incremented
	if ppu.VRAMAddr != 1 {
		t.Errorf("VRAMAddr = %d, expected 1", ppu.VRAMAddr)
	}
}

// TestCGRAMWrite tests CGRAM write functionality
func TestCGRAMWrite(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Set CGRAM address to palette 1, color 1 (0x11)
	ppu.Write8(0x12, 0x11) // CGRAM_ADDR

	// Write RGB555 color (low byte first, then high byte)
	ppu.Write8(0x13, 0xFF) // Low byte
	ppu.Write8(0x13, 0x7F) // High byte

	// Verify CGRAM was written correctly
	addr := 0x11 * 2
	if ppu.CGRAM[addr] != 0xFF {
		t.Errorf("CGRAM[%d] (low) = 0x%02X, expected 0xFF", addr, ppu.CGRAM[addr])
	}
	if ppu.CGRAM[addr+1] != 0x7F {
		t.Errorf("CGRAM[%d] (high) = 0x%02X, expected 0x7F", addr+1, ppu.CGRAM[addr+1])
	}
}

// TestFrameTiming tests PPU frame timing
func TestFrameTiming(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Step PPU for one full frame
	// At 10 MHz, 60 FPS = 166,667 cycles per frame
	// But PPU expects: 220 scanlines × 360 dots = 79,200 cycles per frame
	// This is a timing mismatch that needs to be fixed!
	
	cyclesPerFrame := uint64(220 * 360) // 79,200 cycles
	err := ppu.StepPPU(cyclesPerFrame)
	if err != nil {
		t.Fatalf("StepPPU error: %v", err)
	}

	// Check that frame completed
	if ppu.currentScanline != 0 {
		t.Errorf("After full frame, currentScanline = %d, expected 0", ppu.currentScanline)
	}
	if ppu.currentDot != 0 {
		t.Errorf("After full frame, currentDot = %d, expected 0", ppu.currentDot)
	}
}
