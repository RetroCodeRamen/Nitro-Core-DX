package emulator

import (
	"fmt"
	"testing"

	"nitro-core-dx/internal/cpu"
	"nitro-core-dx/internal/debug"
)

// TestEmulatorFrameExecution tests if the emulator runs frames correctly
func TestEmulatorFrameExecution(t *testing.T) {
	logger := debug.NewLogger(1000)
	emu := NewEmulatorWithLogger(logger)

	// Create a minimal ROM that loops safely within valid ROM address space.
	// ROM header (32 bytes) + minimal code (128 bytes = 0x80)
	romSize := uint32(128) // 128 bytes of ROM data
	romData := make([]byte, 32+romSize)
	// Magic number "RMCF"
	romData[0] = 'R'
	romData[1] = 'M'
	romData[2] = 'C'
	romData[3] = 'F'
	// Version: 1
	romData[4] = 0x01
	romData[5] = 0x00
	// ROM size (32-bit, little-endian): 128 bytes
	romData[6] = byte(romSize & 0xFF)
	romData[7] = byte((romSize >> 8) & 0xFF)
	romData[8] = byte((romSize >> 16) & 0xFF)
	romData[9] = byte((romSize >> 24) & 0xFF)
	// Entry point: bank 1, offset 0x8000 (16-bit little-endian)
	romData[10] = 0x01 // Entry bank
	romData[11] = 0x00
	romData[12] = 0x00 // Entry offset low byte
	romData[13] = 0x80 // Entry offset high byte
	// Minimal loop at 0x8000:
	// 0x8000: NOP
	// 0x8002: JMP -6  (relative to PC after immediate -> back to 0x8000)
	romData[32] = 0x00 // NOP low
	romData[33] = 0x00 // NOP high
	romData[34] = 0x00 // JMP opcode low (0xD000)
	romData[35] = 0xD0 // JMP opcode high
	romData[36] = 0xFA // -6 low byte
	romData[37] = 0xFF // -6 high byte

	// Load ROM
	err := emu.LoadROM(romData)
	if err != nil {
		t.Fatalf("LoadROM error: %v", err)
	}

	// Set up PPU with a sprite
	ppu := emu.PPU

	// Set up a white color in palette 1, color 1
	ppu.CGRAM[0x11*2] = 0xFF   // Low byte (RGB555)
	ppu.CGRAM[0x11*2+1] = 0x7F // High byte

	// Initialize VRAM with white tile (tile 0)
	for i := 0; i < 128; i++ {
		ppu.VRAM[i] = 0x11
	}

	// Set up sprite 0: position (100, 100), tile 0, palette 1, enabled, 16x16
	ppu.OAM[0] = 100  // X low
	ppu.OAM[1] = 0x00 // X high
	ppu.OAM[2] = 100  // Y
	ppu.OAM[3] = 0x00 // Tile index
	ppu.OAM[4] = 0x01 // Attributes (palette 1)
	ppu.OAM[5] = 0x03 // Control (enable + 16x16)

	// Start emulator
	emu.Start()
	// This test is validating frame execution/rendering, not IRQ vector handling.
	// Mask IRQs so VBlank interrupts don't require vector setup in the synthetic ROM.
	emu.CPU.SetFlag(cpu.FlagI, true)

	// Run one frame
	fmt.Printf("Running one frame...\n")
	err = emu.RunFrame()
	if err != nil {
		t.Fatalf("RunFrame error: %v", err)
	}

	// Check output buffer
	buffer := emu.GetOutputBuffer()
	if len(buffer) != 320*200 {
		t.Fatalf("Output buffer size: %d, expected %d", len(buffer), 320*200)
	}

	// Check if sprite was rendered
	whitePixels := 0
	for y := 100; y < 116 && y < 200; y++ {
		for x := 100; x < 116 && x < 320; x++ {
			color := buffer[y*320+x]
			if color != 0x000000 {
				whitePixels++
			}
		}
	}

	fmt.Printf("White pixels found in sprite area: %d\n", whitePixels)
	fmt.Printf("Cycles per frame: %d\n", emu.CyclesPerFrame)
	fmt.Printf("CPU cycles this frame: %d\n", emu.GetCPUCyclesPerFrame())

	if whitePixels == 0 {
		t.Errorf("No sprite pixels rendered! Check if PPU is being stepped correctly.")

		// Debug: Check a few specific pixels
		fmt.Printf("Debug: Checking specific pixels:\n")
		for _, pos := range []struct{ x, y int }{{100, 100}, {105, 100}, {100, 105}, {115, 115}} {
			color := buffer[pos.y*320+pos.x]
			fmt.Printf("  Pixel (%d, %d): 0x%06X\n", pos.x, pos.y, color)
		}

		// Check if buffer is all black
		allBlack := true
		for i := 0; i < 320*200; i++ {
			if buffer[i] != 0x000000 {
				allBlack = false
				fmt.Printf("Found non-black pixel at index %d: 0x%06X\n", i, buffer[i])
				break
			}
		}
		if allBlack {
			t.Errorf("Output buffer is completely black - PPU may not be rendering")
		}
	}
}
