package ppu

import (
	"fmt"
	"testing"

	"nitro-core-dx/internal/debug"
)

// TestFullFrameRendering tests if a complete frame renders correctly
func TestFullFrameRendering(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Set up a white color in palette 1, color 1
	ppu.CGRAM[0x11*2] = 0xFF     // Low byte (RGB555)
	ppu.CGRAM[0x11*2+1] = 0x7F    // High byte

	// Initialize VRAM with white tile (tile 0)
	for i := 0; i < 128; i++ {
		ppu.VRAM[i] = 0x11
	}

	// Set up sprite 0: position (100, 100), tile 0, palette 1, enabled, 16x16
	ppu.OAM[0] = 100   // X low
	ppu.OAM[1] = 0x00  // X high
	ppu.OAM[2] = 100   // Y
	ppu.OAM[3] = 0x00  // Tile index
	ppu.OAM[4] = 0x01  // Attributes (palette 1)
	ppu.OAM[5] = 0x03  // Control (enable + 16x16)

	// Step PPU for one full frame
	// 220 scanlines × 360 dots = 79,200 cycles
	cyclesPerFrame := uint64(220 * 360)
	fmt.Printf("Stepping PPU for %d cycles (one frame)...\n", cyclesPerFrame)
	
	err := ppu.StepPPU(cyclesPerFrame)
	if err != nil {
		t.Fatalf("StepPPU error: %v", err)
	}

	// Check if sprite was rendered
	whitePixels := 0
	for y := 100; y < 116 && y < 200; y++ {
		for x := 100; x < 116 && x < 320; x++ {
			color := ppu.OutputBuffer[y*320+x]
			if color != 0x000000 {
				whitePixels++
				if whitePixels == 1 {
					fmt.Printf("Found non-black pixel at (%d, %d): 0x%06X\n", x, y, color)
				}
			}
		}
	}

	fmt.Printf("White pixels found in sprite area: %d (expected ~256 for 16x16 sprite)\n", whitePixels)
	
	if whitePixels == 0 {
		t.Errorf("No sprite pixels rendered! Sprite should be visible at (100, 100)")
		// Debug: Check a few specific pixels
		fmt.Printf("Debug: Checking specific pixels:\n")
		for _, pos := range []struct{ x, y int }{{100, 100}, {105, 100}, {100, 105}, {115, 115}} {
			color := ppu.OutputBuffer[pos.y*320+pos.x]
			fmt.Printf("  Pixel (%d, %d): 0x%06X\n", pos.x, pos.y, color)
		}
	}
}

// TestPPUStepCounts verifies PPU stepping counts correctly
func TestPPUStepCounts(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Step a few cycles
	err := ppu.StepPPU(10)
	if err != nil {
		t.Fatalf("StepPPU error: %v", err)
	}

	// Check that we've advanced
	if ppu.currentDot == 0 && ppu.currentScanline == 0 {
		t.Errorf("PPU didn't advance after 10 cycles")
	}

	fmt.Printf("After 10 cycles: scanline=%d, dot=%d\n", ppu.currentScanline, ppu.currentDot)
}
