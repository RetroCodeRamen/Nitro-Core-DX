package ppu

import (
	"fmt"
	"testing"

	"nitro-core-dx/internal/debug"
)

// DebugSpriteRendering helps debug sprite rendering issues
func TestDebugSpriteRendering(t *testing.T) {
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

	// Debug: Print sprite data
	fmt.Printf("Sprite 0 OAM data:\n")
	fmt.Printf("  X low: %d\n", ppu.OAM[0])
	fmt.Printf("  X high: %d\n", ppu.OAM[1])
	fmt.Printf("  Y: %d\n", ppu.OAM[2])
	fmt.Printf("  Tile: %d\n", ppu.OAM[3])
	fmt.Printf("  Attributes: 0x%02X\n", ppu.OAM[4])
	fmt.Printf("  Control: 0x%02X\n", ppu.OAM[5])

	// Calculate sprite X position
	spriteX := int(ppu.OAM[0])
	if (ppu.OAM[1] & 0x01) != 0 {
		spriteX |= 0xFFFFFF00
	}
	spriteY := int(ppu.OAM[2])
	fmt.Printf("Sprite position: (%d, %d)\n", spriteX, spriteY)

	// Check if sprite is enabled
	enabled := (ppu.OAM[5] & 0x01) != 0
	tileSize16 := (ppu.OAM[5] & 0x02) != 0
	fmt.Printf("Sprite enabled: %v, 16x16: %v\n", enabled, tileSize16)

	spriteSize := 8
	if tileSize16 {
		spriteSize = 16
	}
	fmt.Printf("Sprite size: %d\n", spriteSize)

	// Test rendering at sprite position
	testX, testY := 100, 100
	fmt.Printf("\nTesting renderDot(%d, %d):\n", testY, testX)
	
	// Check bounds manually
	fmt.Printf("  Sprite bounds: X=[%d, %d), Y=[%d, %d)\n", 
		spriteX, spriteX+spriteSize, spriteY, spriteY+spriteSize)
	fmt.Printf("  Test pixel (%d, %d) in bounds: %v\n", 
		testX, testY, 
		testX >= spriteX && testX < spriteX+spriteSize && testY >= spriteY && testY < spriteY+spriteSize)

	ppu.renderDot(testY, testX)
	color := ppu.OutputBuffer[testY*320+testX]
	fmt.Printf("  Result color: 0x%06X\n", color)

	// Test rendering sprite pixel by pixel
	fmt.Printf("\nRendering sprite area:\n")
	for y := spriteY; y < spriteY+spriteSize && y < 200; y++ {
		for x := spriteX; x < spriteX+spriteSize && x < 320; x++ {
			ppu.renderDot(y, x)
			color := ppu.OutputBuffer[y*320+x]
			if color != 0x000000 {
				fmt.Printf("  Pixel (%d, %d): 0x%06X\n", x, y, color)
			}
		}
	}
}
