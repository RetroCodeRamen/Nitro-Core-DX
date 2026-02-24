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
		fmt.Println("Usage: go run build_input_test.go <output.rom>")
		os.Exit(1)
	}

	outputPath := os.Args[1]
	builder := rom.NewROMBuilder()

	// Input test ROM - displays button state as background color
	// This helps debug input issues by visualizing what the ROM sees
	entryBank := uint8(1)
	entryOffset := uint16(0x8000)

	fmt.Println("Building input test ROM...")
	fmt.Println("This ROM displays button state as background color:")
	fmt.Println("  UP (bit 0) = Red")
	fmt.Println("  DOWN (bit 1) = Green")
	fmt.Println("  LEFT (bit 2) = Blue")
	fmt.Println("  RIGHT (bit 3) = White")

	// ============================================
	// STEP 1: Set up CGRAM colors
	// ============================================
	// Palette 0, Color 0 = Black (no buttons)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Palette 0, Color 1 = Red (UP button)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7C
	builder.AddImmediate(0x7C)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Palette 0, Color 2 = Green (DOWN button)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x02
	builder.AddImmediate(0x02)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xE0
	builder.AddImmediate(0xE0)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x03
	builder.AddImmediate(0x03)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Palette 0, Color 3 = Blue (LEFT button)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x03
	builder.AddImmediate(0x03)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x1F
	builder.AddImmediate(0x1F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Palette 0, Color 4 = White (RIGHT button)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x04
	builder.AddImmediate(0x04)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xFF
	builder.AddImmediate(0xFF)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7F
	builder.AddImmediate(0x7F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// ============================================
	// STEP 2: Set up VRAM tile data
	// ============================================
	// Tile 0: Color index 0 (black)
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
	currentPC := uint16(builder.GetCodeLength() * 2)
	offset := rom.CalculateBranchOffset(currentPC, initTile0Start)
	builder.AddImmediate(uint16(offset))

	// Tiles 1-4: Color indices 1-4 (for button colors)
	for tile := 1; tile <= 4; tile++ {
		colorIndex := uint8(tile)
		tileDataByte := (colorIndex << 4) | colorIndex // 0x11, 0x22, 0x33, 0x44
		
		builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E
		builder.AddImmediate(0x800E)
		builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #(tile * 32)
		builder.AddImmediate(uint16(tile * 32))
		builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F
		builder.AddImmediate(0x800F)
		builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
		builder.AddImmediate(0x00)
		builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

		builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #32
		builder.AddImmediate(32)
		initTileStart := uint16(builder.GetCodeLength() * 2)
		builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010
		builder.AddImmediate(0x8010)
		builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #tileDataByte
		builder.AddImmediate(uint16(tileDataByte))
		builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		builder.AddInstruction(rom.EncodeSUB(1, 6, 0)) // SUB R6, #1
		builder.AddImmediate(1)
		builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
		builder.AddImmediate(0)
		builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
		builder.AddInstruction(rom.EncodeBNE())         // BNE init_tile_start
		currentPC = uint16(builder.GetCodeLength() * 2)
		offset = rom.CalculateBranchOffset(currentPC, initTileStart)
		builder.AddImmediate(uint16(offset))
	}

	// ============================================
	// STEP 3: Set up BG0 tilemap
	// ============================================
	// Enable BG0
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8008
	builder.AddImmediate(0x8008)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Set up tilemap at 0x4000 (all tile 0 initially)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E
	builder.AddImmediate(0x800E)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F
	builder.AddImmediate(0x800F)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x40
	builder.AddImmediate(0x40)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write one tilemap entry (tile 0, palette 0)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010
	builder.AddImmediate(0x8010)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// ============================================
	// STEP 4: Main loop - read input and update backdrop
	// ============================================
	mainLoopStart := uint16(builder.GetCodeLength() * 2)

	// Wait for VBlank
	waitVBlankStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x803E
	builder.AddImmediate(0x803E)
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4]
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 5, 7)) // CMP R5, R7
	builder.AddInstruction(rom.EncodeBEQ())         // BEQ wait_vblank_start
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, waitVBlankStart)
	builder.AddImmediate(uint16(offset))

	// Read input
	// Latch
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001
	builder.AddImmediate(0xA001)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Read low byte
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA000
	builder.AddImmediate(0xA000)
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4]

	// Release latch
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001
	builder.AddImmediate(0xA001)
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 6)) // MOV [R4], R6

	// R5 now contains button state
	// Map button bits to color indices:
	// Bit 0 (UP) = color 1 (red)
	// Bit 1 (DOWN) = color 2 (green)
	// Bit 2 (LEFT) = color 3 (blue)
	// Bit 3 (RIGHT) = color 4 (white)
	// If multiple buttons, use highest priority (RIGHT > LEFT > DOWN > UP)

	// Check RIGHT (bit 3) - highest priority
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x08
	builder.AddImmediate(0x08)
	builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE check_left
	checkRightPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	builder.AddInstruction(rom.EncodeMOV(1, 0, 0)) // MOV R0, #4 (color index for RIGHT)
	builder.AddImmediate(4)
	builder.AddInstruction(rom.EncodeJMP())         // JMP update_color
	colorUpdatePC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)

	checkRightTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(checkRightPC/2), uint16(rom.CalculateBranchOffset(checkRightPC+2, checkRightTarget)))

	// Check LEFT (bit 2)
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x04
	builder.AddImmediate(0x04)
	builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE check_down
	checkLeftPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	builder.AddInstruction(rom.EncodeMOV(1, 0, 0)) // MOV R0, #3 (color index for LEFT)
	builder.AddImmediate(3)
	builder.AddInstruction(rom.EncodeJMP())         // JMP update_color
	builder.AddImmediate(uint16(rom.CalculateBranchOffset(uint16(builder.GetCodeLength()*2), colorUpdatePC)))

	checkLeftTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(checkLeftPC/2), uint16(rom.CalculateBranchOffset(checkLeftPC+2, checkLeftTarget)))

	// Check DOWN (bit 1)
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x02
	builder.AddImmediate(0x02)
	builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE check_up
	checkDownPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	builder.AddInstruction(rom.EncodeMOV(1, 0, 0)) // MOV R0, #2 (color index for DOWN)
	builder.AddImmediate(2)
	builder.AddInstruction(rom.EncodeJMP())         // JMP update_color
	builder.AddImmediate(uint16(rom.CalculateBranchOffset(uint16(builder.GetCodeLength()*2), colorUpdatePC)))

	checkDownTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(checkDownPC/2), uint16(rom.CalculateBranchOffset(checkDownPC+2, checkDownTarget)))

	// Check UP (bit 0)
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE no_input
	checkUpPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	builder.AddInstruction(rom.EncodeMOV(1, 0, 0)) // MOV R0, #1 (color index for UP)
	builder.AddImmediate(1)
	builder.AddInstruction(rom.EncodeJMP())         // JMP update_color
	builder.AddImmediate(uint16(rom.CalculateBranchOffset(uint16(builder.GetCodeLength()*2), colorUpdatePC)))

	checkUpTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(checkUpPC/2), uint16(rom.CalculateBranchOffset(checkUpPC+2, checkUpTarget)))

	// No input - use color 0 (black)
	builder.AddInstruction(rom.EncodeMOV(1, 0, 0)) // MOV R0, #0
	builder.AddImmediate(0)

	// Update backdrop color (palette 0, color 0)
	colorUpdateTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(colorUpdatePC/2), uint16(rom.CalculateBranchOffset(colorUpdatePC+2, colorUpdateTarget)))

	// Set CGRAM address to palette 0, color 0
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write color based on R0 (color index)
	// Use a jump table - check R0 value
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
	builder.AddImmediate(0x8013)

	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 0, 7)) // CMP R0, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE not_0
	color0PC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	// Color 0: Black
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeJMP())         // JMP color_done
	colorDonePC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)

	color0Target := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(color0PC/2), uint16(rom.CalculateBranchOffset(color0PC+2, color0Target)))

	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #1
	builder.AddImmediate(1)
	builder.AddInstruction(rom.EncodeCMP(0, 0, 7)) // CMP R0, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE not_1
	color1PC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	// Color 1: Red
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7C
	builder.AddImmediate(0x7C)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeJMP())         // JMP color_done
	builder.AddImmediate(uint16(rom.CalculateBranchOffset(uint16(builder.GetCodeLength()*2), colorDonePC)))

	color1Target := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(color1PC/2), uint16(rom.CalculateBranchOffset(color1PC+2, color1Target)))

	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #2
	builder.AddImmediate(2)
	builder.AddInstruction(rom.EncodeCMP(0, 0, 7)) // CMP R0, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE not_2
	color2PC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	// Color 2: Green
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xE0
	builder.AddImmediate(0xE0)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x03
	builder.AddImmediate(0x03)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeJMP())         // JMP color_done
	builder.AddImmediate(uint16(rom.CalculateBranchOffset(uint16(builder.GetCodeLength()*2), colorDonePC)))

	color2Target := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(color2PC/2), uint16(rom.CalculateBranchOffset(color2PC+2, color2Target)))

	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #3
	builder.AddImmediate(3)
	builder.AddInstruction(rom.EncodeCMP(0, 0, 7)) // CMP R0, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE not_3
	color3PC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	// Color 3: Blue
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x1F
	builder.AddImmediate(0x1F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeJMP())         // JMP color_done
	builder.AddImmediate(uint16(rom.CalculateBranchOffset(uint16(builder.GetCodeLength()*2), colorDonePC)))

	color3Target := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(color3PC/2), uint16(rom.CalculateBranchOffset(color3PC+2, color3Target)))

	// Color 4: White (default)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xFF
	builder.AddImmediate(0xFF)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7F
	builder.AddImmediate(0x7F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	colorDoneTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(colorDonePC/2), uint16(rom.CalculateBranchOffset(colorDonePC+2, colorDoneTarget)))

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

	fmt.Printf("Input test ROM built: %s\n", outputPath)
	fmt.Println("\nExpected behavior:")
	fmt.Println("  - Black background = no input")
	fmt.Println("  - Red background = UP pressed")
	fmt.Println("  - Green background = DOWN pressed")
	fmt.Println("  - Blue background = LEFT pressed")
	fmt.Println("  - White background = RIGHT pressed")
}
