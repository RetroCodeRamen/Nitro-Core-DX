package ppu

import (
	"fmt"
	"testing"

	"nitro-core-dx/internal/debug"
)

// TestPPUDirectWrite tests PPU by directly writing to registers (no CPU)
func TestPPUDirectWrite(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	fmt.Printf("=== Direct PPU Write Test ===\n\n")

	// Step 1: Set up palette (palette 1, color 1 = white)
	fmt.Printf("1. Setting up palette...\n")
	ppu.Write8(0x12, 0x11) // CGRAM_ADDR = palette 1, color 1
	ppu.Write8(0x13, 0xFF) // CGRAM_DATA low byte
	ppu.Write8(0x13, 0x7F) // CGRAM_DATA high byte
	
	// Verify palette
	addr := 0x11 * 2
	if ppu.CGRAM[addr] != 0xFF || ppu.CGRAM[addr+1] != 0x7F {
		t.Errorf("Palette not set correctly: CGRAM[%d]=0x%02X, CGRAM[%d]=0x%02X", 
			addr, ppu.CGRAM[addr], addr+1, ppu.CGRAM[addr+1])
	}
	fmt.Printf("   ✅ Palette set: CGRAM[0x%02X]=0x%02X, CGRAM[0x%02X]=0x%02X\n", 
		addr, ppu.CGRAM[addr], addr+1, ppu.CGRAM[addr+1])

	// Step 2: Initialize VRAM with white tile (tile 0)
	fmt.Printf("\n2. Initializing VRAM tile 0...\n")
	ppu.Write8(0x0E, 0x00) // VRAM_ADDR_L = 0x00
	ppu.Write8(0x0F, 0x00) // VRAM_ADDR_H = 0x00
	
	// Write 128 bytes of 0x11 (16x16 tile = 128 bytes)
	for i := 0; i < 128; i++ {
		ppu.Write8(0x10, 0x11) // VRAM_DATA = 0x11 (color index 1)
	}
	
	// Verify VRAM
	if ppu.VRAM[0] != 0x11 || ppu.VRAM[127] != 0x11 {
		t.Errorf("VRAM not initialized correctly: VRAM[0]=0x%02X, VRAM[127]=0x%02X", 
			ppu.VRAM[0], ppu.VRAM[127])
	}
	fmt.Printf("   ✅ VRAM initialized: VRAM[0]=0x%02X, VRAM[127]=0x%02X\n", 
		ppu.VRAM[0], ppu.VRAM[127])

	// Step 3: Set up sprite 0
	fmt.Printf("\n3. Setting up sprite 0...\n")
	ppu.Write8(0x14, 0x00) // OAM_ADDR = sprite 0
	ppu.Write8(0x15, 100)  // X low = 100
	ppu.Write8(0x15, 0x00) // X high = 0
	ppu.Write8(0x15, 100)  // Y = 100
	ppu.Write8(0x15, 0x00) // Tile = 0
	ppu.Write8(0x15, 0x01) // Attributes = palette 1
	ppu.Write8(0x15, 0x03) // Control = enable + 16x16
	
	// Verify OAM
	if ppu.OAM[0] != 100 || ppu.OAM[2] != 100 || ppu.OAM[4] != 0x01 || ppu.OAM[5] != 0x03 {
		t.Errorf("OAM not set correctly: OAM[0]=%d, OAM[2]=%d, OAM[4]=0x%02X, OAM[5]=0x%02X",
			ppu.OAM[0], ppu.OAM[2], ppu.OAM[4], ppu.OAM[5])
	}
	fmt.Printf("   ✅ Sprite set: X=%d, Y=%d, Tile=%d, Attr=0x%02X, Ctrl=0x%02X\n",
		ppu.OAM[0], ppu.OAM[2], ppu.OAM[3], ppu.OAM[4], ppu.OAM[5])

	// Step 4: Render a full frame
	fmt.Printf("\n4. Rendering full frame...\n")
	cyclesPerFrame := uint64(220 * 360) // 79,200 cycles
	err := ppu.StepPPU(cyclesPerFrame)
	if err != nil {
		t.Fatalf("StepPPU error: %v", err)
	}
	fmt.Printf("   ✅ Frame rendered\n")

	// Step 5: Check output buffer
	fmt.Printf("\n5. Checking output buffer...\n")
	whitePixels := 0
	firstWhitePixel := struct{ x, y int; color uint32 }{-1, -1, 0}
	
	for y := 100; y < 116 && y < 200; y++ {
		for x := 100; x < 116 && x < 320; x++ {
			color := ppu.OutputBuffer[y*320+x]
			if color != 0x000000 {
				whitePixels++
				if firstWhitePixel.x == -1 {
					firstWhitePixel.x = x
					firstWhitePixel.y = y
					firstWhitePixel.color = color
				}
			}
		}
	}
	
	fmt.Printf("   White pixels in sprite area: %d (expected ~256)\n", whitePixels)
	if firstWhitePixel.x != -1 {
		fmt.Printf("   First white pixel at (%d, %d): 0x%06X\n", 
			firstWhitePixel.x, firstWhitePixel.y, firstWhitePixel.color)
	}
	
	if whitePixels == 0 {
		t.Errorf("No sprite pixels rendered!")
		
		// Debug: Check specific pixels
		fmt.Printf("\n   Debug: Checking specific pixels:\n")
		for _, pos := range []struct{ x, y int }{{100, 100}, {105, 100}, {100, 105}, {115, 115}} {
			color := ppu.OutputBuffer[pos.y*320+pos.x]
			fmt.Printf("     Pixel (%d, %d): 0x%06X\n", pos.x, pos.y, color)
		}
		
		// Check if sprite rendering function works
		fmt.Printf("\n   Testing renderDot directly:\n")
		ppu.OutputBuffer[100*320+100] = 0x000000 // Clear first
		ppu.renderDot(100, 100)
		directColor := ppu.OutputBuffer[100*320+100]
		fmt.Printf("     renderDot(100, 100) result: 0x%06X\n", directColor)
	} else {
		fmt.Printf("   ✅ Sprite rendering works!\n")
	}
}

// TestPPUFrameTiming tests PPU frame timing accuracy
func TestPPUFrameTiming(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Render multiple frames and verify timing
	framesToTest := 5
	cyclesPerFrame := uint64(220 * 360)
	
	fmt.Printf("=== PPU Frame Timing Test ===\n")
	fmt.Printf("Testing %d frames at %d cycles per frame...\n\n", framesToTest, cyclesPerFrame)
	
	for frame := 0; frame < framesToTest; frame++ {
		// Clear buffer manually to track frames
		for i := range ppu.OutputBuffer {
			ppu.OutputBuffer[i] = 0x000000
		}
		
		// Set a test pixel
		ppu.OutputBuffer[0] = 0x123456
		
		// Render frame
		err := ppu.StepPPU(cyclesPerFrame)
		if err != nil {
			t.Fatalf("Frame %d error: %v", frame, err)
		}
		
		// Check if buffer was cleared (startFrame clears it)
		if ppu.OutputBuffer[0] == 0x000000 {
			fmt.Printf("Frame %d: ✅ Buffer cleared at start (correct)\n", frame)
		} else {
			fmt.Printf("Frame %d: ⚠️  Buffer not cleared (test pixel still there)\n", frame)
		}
		
		// Check frame counter
		fmt.Printf("  Frame counter: %d\n", ppu.FrameCounter)
	}
}

// TestPPUBufferRaceCondition tests if buffer can be read while rendering
func TestPPUBufferRaceCondition(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	// Set up sprite
	ppu.CGRAM[0x11*2] = 0xFF
	ppu.CGRAM[0x11*2+1] = 0x7F
	for i := 0; i < 128; i++ {
		ppu.VRAM[i] = 0x11
	}
	ppu.OAM[0] = 100
	ppu.OAM[1] = 0x00
	ppu.OAM[2] = 100
	ppu.OAM[3] = 0x00
	ppu.OAM[4] = 0x01
	ppu.OAM[5] = 0x03

	fmt.Printf("=== PPU Buffer Race Condition Test ===\n\n")
	
	// Simulate reading buffer while rendering
	cyclesPerFrame := uint64(220 * 360)
	
	// Start rendering (in a simulated way)
	halfFrame := cyclesPerFrame / 2
	
	// Render half frame
	ppu.StepPPU(halfFrame)
	
	// Read buffer mid-frame
	bufferMidFrame := make([]uint32, len(ppu.OutputBuffer))
	copy(bufferMidFrame, ppu.OutputBuffer[:])
	
	// Count non-black pixels mid-frame
	midFramePixels := 0
	for _, color := range bufferMidFrame {
		if color != 0x000000 {
			midFramePixels++
		}
	}
	fmt.Printf("Mid-frame (after %d cycles): %d non-black pixels\n", halfFrame, midFramePixels)
	
	// Finish frame
	ppu.StepPPU(halfFrame)
	
	// Read buffer after frame
	bufferAfterFrame := make([]uint32, len(ppu.OutputBuffer))
	copy(bufferAfterFrame, ppu.OutputBuffer[:])
	
	// Count non-black pixels after frame
	afterFramePixels := 0
	for _, color := range bufferAfterFrame {
		if color != 0x000000 {
			afterFramePixels++
		}
	}
	fmt.Printf("After frame: %d non-black pixels\n", afterFramePixels)
	
	// Check sprite area
	whitePixels := 0
	for y := 100; y < 116 && y < 200; y++ {
		for x := 100; x < 116 && x < 320; x++ {
			if bufferAfterFrame[y*320+x] != 0x000000 {
				whitePixels++
			}
		}
	}
	fmt.Printf("White pixels in sprite area: %d\n", whitePixels)
	
	if whitePixels == 0 {
		t.Errorf("Sprite not rendered after full frame!")
	} else {
		fmt.Printf("✅ Sprite rendered correctly\n")
	}
}
