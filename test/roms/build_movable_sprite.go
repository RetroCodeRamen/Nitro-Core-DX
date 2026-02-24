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
		fmt.Println("Usage: go run build_movable_sprite.go <output.rom>")
		os.Exit(1)
	}

	outputPath := os.Args[1]
	builder := rom.NewROMBuilder()

	// Movable sprite test ROM
	// Arrow keys/WASD move the sprite
	entryBank := uint8(1)
	entryOffset := uint16(0x8000)

	fmt.Println("Building movable sprite test ROM...")

	// ============================================
	// STEP 1: Set up CGRAM colors
	// ============================================
	fmt.Println("  [1] Setting up CGRAM colors...")
	
	// Palette 0, Color 0 (backdrop) - Blue
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012 (CGRAM_ADDR)
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013 (CGRAM_DATA)
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x1F (blue low)
	builder.AddImmediate(0x1F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (blue high)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Palette 1, Color 0 (transparent)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x20 (palette 1, color 0)
	builder.AddImmediate(0x20)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Palette 1, Color 1 (sprite) - Red
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x11 (palette 1, color 1)
	builder.AddImmediate(0x11)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (red low)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7C (red high)
	builder.AddImmediate(0x7C)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// ============================================
	// STEP 2: Set up VRAM tile data
	// ============================================
	fmt.Println("  [2] Setting up VRAM tile data...")
	
	// Tile 0: Color index 0 (for backdrop)
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

	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #32
	builder.AddImmediate(32)
	initTile0Start := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010 (VRAM_DATA)
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

	// Tile 1: Color index 1 (for sprite)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E
	builder.AddImmediate(0x800E)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x20 (tile 1 = 32 bytes)
	builder.AddImmediate(0x20)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F
	builder.AddImmediate(0x800F)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

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

	// ============================================
	// STEP 3: Set up BG0 with backdrop
	// ============================================
	fmt.Println("  [3] Setting up BG0...")
	
	// Enable BG0
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8008 (BG0_CONTROL)
	builder.AddImmediate(0x8008)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Set up tilemap at 0x4000 (all tile 0 = backdrop)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E
	builder.AddImmediate(0x800E)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F
	builder.AddImmediate(0x800F)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x40 (address = 0x4000)
	builder.AddImmediate(0x40)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write one tilemap entry (tile 0, palette 0)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010
	builder.AddImmediate(0x8010)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (tile 0)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (palette 0)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// ============================================
	// STEP 4: Initialize sprite position (R0 = X, R1 = Y)
	// ============================================
	fmt.Println("  [4] Initializing sprite position...")
	builder.AddInstruction(rom.EncodeMOV(1, 0, 0)) // MOV R0, #160 (center X)
	builder.AddImmediate(160)
	builder.AddInstruction(rom.EncodeMOV(1, 1, 0)) // MOV R1, #100 (center Y)
	builder.AddImmediate(100)

	// ============================================
	// STEP 5: Main loop
	// ============================================
	fmt.Println("  [5] Setting up main loop with input handling...")
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
	// STEP 6: Read input
	// ============================================
	// Latch controller
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001 (CONTROLLER1_LATCH)
	builder.AddImmediate(0xA001)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Read controller state (low byte)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA000 (CONTROLLER1)
	builder.AddImmediate(0xA000)
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (8-bit read, zero-extended)

	// Release latch
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001
	builder.AddImmediate(0xA001)
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 6)) // MOV [R4], R6

	// R5 now contains button state (bits: UP=0, DOWN=1, LEFT=2, RIGHT=3)

	// ============================================
	// STEP 7: Handle UP button (bit 0) - decrement Y
	// ============================================
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5 (copy buttons)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7 (mask UP bit)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE skip_up
	skipUpPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	builder.AddInstruction(rom.EncodeSUB(1, 1, 0)) // SUB R1, #1 (move up)
	builder.AddImmediate(1)
	skipUpTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(skipUpPC/2), uint16(rom.CalculateBranchOffset(skipUpPC+2, skipUpTarget)))

	// ============================================
	// STEP 8: Handle DOWN button (bit 1) - increment Y
	// ============================================
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x02
	builder.AddImmediate(0x02)
	builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE skip_down
	skipDownPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	builder.AddInstruction(rom.EncodeADD(1, 1, 0)) // ADD R1, #1 (move down)
	builder.AddImmediate(1)
	skipDownTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(skipDownPC/2), uint16(rom.CalculateBranchOffset(skipDownPC+2, skipDownTarget)))

	// ============================================
	// STEP 9: Handle LEFT button (bit 2) - decrement X
	// ============================================
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x04
	builder.AddImmediate(0x04)
	builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE skip_left
	skipLeftPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	builder.AddInstruction(rom.EncodeSUB(1, 0, 0)) // SUB R0, #1 (move left)
	builder.AddImmediate(1)
	skipLeftTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(skipLeftPC/2), uint16(rom.CalculateBranchOffset(skipLeftPC+2, skipLeftTarget)))

	// ============================================
	// STEP 10: Handle RIGHT button (bit 3) - increment X
	// ============================================
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x08
	builder.AddImmediate(0x08)
	builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE skip_right
	skipRightPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	builder.AddInstruction(rom.EncodeADD(1, 0, 0)) // ADD R0, #1 (move right)
	builder.AddImmediate(1)
	skipRightTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(skipRightPC/2), uint16(rom.CalculateBranchOffset(skipRightPC+2, skipRightTarget)))

	// ============================================
	// STEP 11: Update sprite position in OAM
	// ============================================
	// Set OAM address to sprite 0
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014 (OAM_ADDR)
	builder.AddImmediate(0x8014)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write sprite data via OAM_DATA
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015 (OAM_DATA)
	builder.AddImmediate(0x8015)

	// Byte 0: X low byte (from R0)
	builder.AddInstruction(rom.EncodeMOV(0, 5, 0)) // MOV R5, R0
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Byte 1: X high byte (check if X >= 256)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x0100
	builder.AddImmediate(0x0100)
	builder.AddInstruction(rom.EncodeCMP(0, 0, 7)) // CMP R0, R7
	builder.AddInstruction(rom.EncodeBLT())         // BLT x_low
	xHighPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01 (high bit set)
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeJMP())         // JMP x_done
	xDonePC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)

	xHighTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(xHighPC/2), uint16(rom.CalculateBranchOffset(xHighPC+2, xHighTarget)))

	// X < 256
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	xDoneTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(xDonePC/2), uint16(rom.CalculateBranchOffset(xDonePC+2, xDoneTarget)))

	// Byte 2: Y position (from R1)
	builder.AddInstruction(rom.EncodeMOV(0, 5, 1)) // MOV R5, R1
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

	// ============================================
	// STEP 12: Loop back to main loop
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

	fmt.Printf("Movable sprite test ROM built: %s\n", outputPath)
	fmt.Println("\nControls:")
	fmt.Println("  Arrow Keys / WASD: Move sprite")
	fmt.Println("\nExpected result:")
	fmt.Println("  - Blue background")
	fmt.Println("  - Red sprite that moves with arrow keys")
}
