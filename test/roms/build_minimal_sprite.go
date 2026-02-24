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
		fmt.Println("Usage: go run build_minimal_sprite.go <output.rom>")
		os.Exit(1)
	}

	outputPath := os.Args[1]
	builder := rom.NewROMBuilder()

	// Minimal sprite test ROM - displays a single white sprite
	// Based on system design specification
	entryBank := uint8(1)
	entryOffset := uint16(0x8000)

	fmt.Println("Building minimal sprite test ROM from design spec...")

	// ============================================
	// STEP 1: Set up CGRAM palette
	// ============================================
	// CGRAM: 512 bytes, 256 colors × 2 bytes
	// Format: RGB555, 16 palettes × 16 colors each
	// CGRAM_ADDR (0x8012): Sets address (palette + color index)
	// CGRAM_DATA (0x8013): Write low byte first, then high byte (16-bit write with latch)
	
	fmt.Println("  [1] Setting up CGRAM palette 1, color 1 (white)...")
	
	// Set CGRAM address to palette 1, color 1
	// Address = (palette × 16 + color) = (1 × 16 + 1) = 17 = 0x11
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012 (CGRAM_ADDR)
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x11 (palette 1, color 1)
	builder.AddImmediate(0x11)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write CGRAM_ADDR)

	// Write white color: RGB555 = 0x7FFF
	// Low byte = 0xFF, High byte = 0x7F
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013 (CGRAM_DATA)
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xFF (low byte)
	builder.AddImmediate(0xFF)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write low byte)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7F (high byte)
	builder.AddImmediate(0x7F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write high byte)

	// ============================================
	// STEP 2: Set up VRAM tile data
	// ============================================
	// VRAM: 64KB, 8-bit access via VRAM_DATA (0x8010)
	// VRAM_ADDR_L (0x800E) and VRAM_ADDR_H (0x800F) set address
	// Tile format: 4bpp, 2 pixels per byte
	// 8×8 tile = 32 bytes, 16×16 tile = 128 bytes
	// For a solid tile with color index 1: write 0x11 (two pixels, both color 1)
	
	fmt.Println("  [2] Setting up VRAM tile data (tile 0)...")
	
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
	// 8×8 tile = 64 pixels = 32 bytes (2 pixels per byte)
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #32 (counter)
	builder.AddImmediate(32)

	initVRAMStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010 (VRAM_DATA)
	builder.AddImmediate(0x8010)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x11 (two pixels, color 1)
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
	// STEP 3: Disable BG0 (so sprite is visible on black background)
	// ============================================
	fmt.Println("  [3] Disabling BG0 (sprite will show on black background)...")
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8008 (BG0_CONTROL)
	builder.AddImmediate(0x8008)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (disable)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// ============================================
	// STEP 4: Main loop - wait for VBlank and update sprite
	// ============================================
	fmt.Println("  [4] Setting up main loop...")
	mainLoopStart := uint16(builder.GetCodeLength() * 2)

	// Wait for VBlank
	// VBLANK_FLAG (0x803E): Set at start of VBlank, cleared when read
	waitVBlankStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x803E (VBLANK_FLAG)
	builder.AddImmediate(0x803E)
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read VBlank flag, mode 2 = register indirect)
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
	// OAM: 768 bytes (128 sprites × 6 bytes)
	// OAM_ADDR (0x8014): Sets sprite ID (0-127)
	// OAM_DATA (0x8015): 8-bit writes, auto-increments byte index
	// Sprite format:
	//   Byte 0: X low byte
	//   Byte 1: X high byte (bit 0 = sign bit)
	//   Byte 2: Y position
	//   Byte 3: Tile index
	//   Byte 4: Attributes (bits [3:0] = palette, bit 4 = flip X, bit 5 = flip Y, bits [7:6] = priority)
	//   Byte 5: Control (bit 0 = enable, bit 1 = 16×16 size)
	
	fmt.Println("  [5] Writing sprite 0 to OAM...")
	
	// Set OAM address to sprite 0
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014 (OAM_ADDR)
	builder.AddImmediate(0x8014)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (sprite 0)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write sprite data via OAM_DATA (0x8015)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015 (OAM_DATA)
	builder.AddImmediate(0x8015)

	// Byte 0: X low byte = 160 (center of 320px screen)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #160
	builder.AddImmediate(160)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Byte 1: X high byte = 0 (X < 256, so bit 0 = 0)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Byte 2: Y position = 100
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #100
	builder.AddImmediate(100)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Byte 3: Tile index = 0
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Byte 4: Attributes = palette 1 (bits [3:0] = 0x01), no flip, priority 0
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Byte 5: Control = enable (bit 0 = 1), 8×8 size (bit 1 = 0)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// ============================================
	// STEP 6: Loop back to main loop
	// ============================================
	builder.AddInstruction(rom.EncodeJMP())
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, mainLoopStart)
	builder.AddImmediate(uint16(offset))

	// Build ROM
	if err := builder.BuildROM(entryBank, entryOffset, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error building ROM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Minimal sprite test ROM built: %s\n", outputPath)
	fmt.Println("\nExpected result: White 8×8 sprite at position (160, 100)")
}
