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
		fmt.Println("Usage: go run build_simple_input_test.go <output.rom>")
		os.Exit(1)
	}

	outputPath := os.Args[1]
	builder := rom.NewROMBuilder()

	// Very simple input test - just reads input and sets backdrop color
	entryBank := uint8(1)
	entryOffset := uint16(0x8000)

	fmt.Println("Building simple input test ROM...")

	// ============================================
	// STEP 1: Set up CGRAM colors
	// ============================================
	// Palette 0, Color 0 = Black
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

	// Palette 0, Color 1 = Red (for UP button)
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

	// ============================================
	// STEP 3: Enable BG0
	// ============================================
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8008
	builder.AddImmediate(0x8008)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Set up tilemap
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
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010
	builder.AddImmediate(0x8010)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// ============================================
	// STEP 4: Main loop - very simple
	// ============================================
	mainLoopStart := uint16(builder.GetCodeLength() * 2)

	// Read input
	// Latch
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001
	builder.AddImmediate(0xA001)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Read
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA000
	builder.AddImmediate(0xA000)
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4]

	// Release latch
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001
	builder.AddImmediate(0xA001)
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 6)) // MOV [R4], R6

	// Check if UP button (bit 0) is pressed
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE no_up
	noUpPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)

	// UP pressed - set backdrop to red
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
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7C
	builder.AddImmediate(0x7C)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeJMP())         // JMP loop_back
	loopBackPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)

	noUpTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(noUpPC/2), uint16(rom.CalculateBranchOffset(noUpPC+2, noUpTarget)))

	// UP not pressed - set backdrop to black
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

	loopBackTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(loopBackPC/2), uint16(rom.CalculateBranchOffset(loopBackPC+2, loopBackTarget)))

	// Simple delay (don't loop too fast)
	builder.AddInstruction(rom.EncodeMOV(1, 2, 0)) // MOV R2, #10000
	builder.AddImmediate(10000)
	delayStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeSUB(1, 2, 0)) // SUB R2, #1
	builder.AddImmediate(1)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 2, 7)) // CMP R2, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE delay_start
	delayPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	builder.SetImmediateAt(int(delayPC/2), uint16(rom.CalculateBranchOffset(delayPC+2, delayStart)))

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

	fmt.Printf("Simple input test ROM built: %s\n", outputPath)
	fmt.Println("\nExpected behavior:")
	fmt.Println("  - Black background = UP not pressed")
	fmt.Println("  - Red background = UP pressed (W or Up Arrow)")
}
