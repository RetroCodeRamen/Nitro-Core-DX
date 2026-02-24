//go:build testrom_tools
// +build testrom_tools

package main

import (
	"fmt"
	"os"

	"nitro-core-dx/internal/rom"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run build_sprite_with_backdrop.go <output.rom>")
		os.Exit(1)
	}

	outputPath := os.Args[1]
	builder := rom.NewROMBuilder()

	// Sprite test with colored backdrop
	// This will help us see if sprites are rendering
	entryBank := uint8(1)
	entryOffset := uint16(0x8000)

	fmt.Println("Building sprite test with colored backdrop...")

	// ============================================
	// STEP 1: Set up CGRAM colors
	// ============================================
	fmt.Println("  [1] Setting up CGRAM colors...")
	
	// Palette 0, Color 0 (backdrop) - Blue
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012 (CGRAM_ADDR)
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (palette 0, color 0)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013 (CGRAM_DATA)
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x1F (blue low byte)
	builder.AddImmediate(0x1F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (blue high byte)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Palette 1, Color 0 (transparent for sprite)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x20 (palette 1, color 0)
	builder.AddImmediate(0x20)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (black/transparent)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Palette 1, Color 1 (sprite color) - Red
	// CGRAM address = (palette * 16 + color) = (1 * 16 + 1) = 17 = 0x11
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x11 (palette 1, color 1)
	builder.AddImmediate(0x11)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (red low byte)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7C (red high byte)
	builder.AddImmediate(0x7C)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// ============================================
	// STEP 2: Set up VRAM tile data
	// ============================================
	fmt.Println("  [2] Setting up VRAM tile data...")
	
	// Set VRAM address to 0 (tile 0)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E (VRAM_ADDR_L)
	builder.AddImmediate(0x800E)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F (VRAM_ADDR_H)
	builder.AddImmediate(0x800F)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write 32 bytes of 0x11 (solid tile, color index 1)
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #32
	builder.AddImmediate(32)

	initVRAMStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010 (VRAM_DATA)
	builder.AddImmediate(0x8010)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x11
	builder.AddImmediate(0x11)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeSUB(1, 6, 0)) // SUB R6, #1
	builder.AddImmediate(1)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE init_vram_start
	currentPC := uint16(builder.GetCodeLength() * 2)
	offset := rom.CalculateBranchOffset(currentPC, initVRAMStart)
	builder.AddImmediate(uint16(offset))

	// ============================================
	// STEP 3: Set up BG0 with backdrop color
	// ============================================
	fmt.Println("  [3] Setting up BG0 with backdrop...")
	
	// Enable BG0
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8008 (BG0_CONTROL)
	builder.AddImmediate(0x8008)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01 (enable)
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Set up a simple tilemap at 0x4000 that uses palette 0, color 0 (backdrop)
	// This will fill the screen with the backdrop color
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E (VRAM_ADDR_L)
	builder.AddImmediate(0x800E)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (address low = 0x00)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F (VRAM_ADDR_H)
	builder.AddImmediate(0x800F)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x40 (address high = 0x40, so address = 0x4000)
	builder.AddImmediate(0x40)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write tilemap entry: tile 0, palette 0, color 0
	// Tile index = 0 (byte 0)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010 (VRAM_DATA)
	builder.AddImmediate(0x8010)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (tile 0)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	// Attributes = palette 0 (byte 1)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (palette 0)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// But wait - tile 0 needs to have color index 0 to show backdrop color
	// Actually, we need to set up tile 0 with color index 0
	// Let me set VRAM address back to 0 and write tile 0 with color index 0
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E
	builder.AddImmediate(0x800E)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F
	builder.AddImmediate(0x800F)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write 32 bytes of 0x00 (solid tile, color index 0 = backdrop)
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #32
	builder.AddImmediate(32)

	initTile0Start := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010
	builder.AddImmediate(0x8010)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeSUB(1, 6, 0)) // SUB R6, #1
	builder.AddImmediate(1)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE init_tile0_start
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, initTile0Start)
	builder.AddImmediate(uint16(offset))

	// ============================================
	// STEP 4: Main loop
	// ============================================
	fmt.Println("  [4] Setting up main loop...")
	mainLoopStart := uint16(builder.GetCodeLength() * 2)

	// Wait for VBlank
	waitVBlankStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x803E (VBLANK_FLAG)
	builder.AddImmediate(0x803E)
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4]
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 5, 7)) // CMP R5, R7
	builder.AddInstruction(rom.EncodeBEQ())         // BEQ wait_vblank_start
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, waitVBlankStart)
	builder.AddImmediate(uint16(offset))

	// ============================================
	// STEP 5: Write sprite to OAM
	// ============================================
	fmt.Println("  [5] Writing sprite 0 to OAM...")
	
	// Set OAM address to sprite 0
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014 (OAM_ADDR)
	builder.AddImmediate(0x8014)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write sprite data
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015 (OAM_DATA)
	builder.AddImmediate(0x8015)

	// Byte 0: X low = 160
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #160
	builder.AddImmediate(160)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Byte 1: X high = 0
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Byte 2: Y = 100
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #100
	builder.AddImmediate(100)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Byte 3: Tile index = 0 (but this tile has color index 1, not 0!)
	// Actually, we need tile 1 for the sprite, or we need to use a different tile
	// Let me use tile 0 but the sprite will use palette 1, color 1
	// Wait, tile 0 now has color index 0. We need tile 1 with color index 1
	// Let me set up tile 1
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E
	builder.AddImmediate(0x800E)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x20 (tile 1 starts at 0x20 = 32 bytes)
	builder.AddImmediate(0x20)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F
	builder.AddImmediate(0x800F)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write 32 bytes of 0x11 for tile 1
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #32
	builder.AddImmediate(32)

	initTile1Start := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010
	builder.AddImmediate(0x8010)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x11
	builder.AddImmediate(0x11)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeSUB(1, 6, 0)) // SUB R6, #1
	builder.AddImmediate(1)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE init_tile1_start
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, initTile1Start)
	builder.AddImmediate(uint16(offset))

	// Now write sprite with tile index 1
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014 (OAM_ADDR)
	builder.AddImmediate(0x8014)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015
	builder.AddImmediate(0x8015)

	// Byte 0: X low = 160
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #160
	builder.AddImmediate(160)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Byte 1: X high = 0
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Byte 2: Y = 100
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #100
	builder.AddImmediate(100)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Byte 3: Tile index = 1
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Byte 4: Attributes = palette 1
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Byte 5: Control = enable, 8×8
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Loop back
	builder.AddInstruction(rom.EncodeJMP())
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, mainLoopStart)
	builder.AddImmediate(uint16(offset))

	// Build ROM
	if err := builder.BuildROM(entryBank, entryOffset, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error building ROM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sprite test ROM with backdrop built: %s\n", outputPath)
	fmt.Println("\nExpected result:")
	fmt.Println("  - Blue background (backdrop color)")
	fmt.Println("  - Red 8×8 sprite at position (160, 100)")
}
