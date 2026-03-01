package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: testrom_input <output.rom>")
		os.Exit(1)
	}

	outputPath := os.Args[1]
	code := []uint16{}

	// Helper to add instruction
	add := func(inst uint16) {
		code = append(code, inst)
	}

	// Helper to add immediate value
	addImmVal := func(val uint16) {
		code = append(code, val)
	}

	// Helper for MOV R, #imm (16-bit)
	movImm16 := func(reg uint8, val uint16) {
		add(0x1000 | (1 << 8) | (uint16(reg) << 4))
		addImmVal(val)
	}

	// Helper for MOV R, #imm (8-bit)
	movImm := func(reg uint8, val uint8) {
		add(0x1000 | (1 << 8) | (uint16(reg) << 4))
		addImmVal(uint16(val))
	}

	// Helper for MOV R1, R2
	movReg := func(reg1, reg2 uint8) {
		add(0x1000 | (uint16(reg1) << 4) | uint16(reg2))
	}

	// Helper for MOV [R1], R2 (store to memory)
	movMem := func(reg1, reg2 uint8) {
		add(0x1000 | (3 << 8) | (uint16(reg1) << 4) | uint16(reg2))
	}

	// Helper for ADD R, #imm
	addImm := func(reg uint8, val uint16) {
		add(0x2000 | (1 << 8) | (uint16(reg) << 4))
		addImmVal(val)
	}

	// Helper for SUB R, #imm
	subImm := func(reg uint8, val uint16) {
		add(0x3000 | (1 << 8) | (uint16(reg) << 4))
		addImmVal(val)
	}

	// Helper for AND R, #imm
	andImm := func(reg uint8, val uint16) {
		add(0x6000 | (1 << 8) | (uint16(reg) << 4))
		addImmVal(val)
	}

	// Helper for CMP R, #imm
	cmpImm := func(reg uint8, val uint16) {
		add(0xC000 | (1 << 8) | (uint16(reg) << 4))
		addImmVal(val)
	}

	// Helper for CMP R1, R2
	cmpReg := func(reg1, reg2 uint8) {
		add(0xC000 | (uint16(reg1) << 4) | uint16(reg2))
	}

	// Helper for BEQ offset
	beq := func(offset int16) {
		add(0xC100)
		addImmVal(uint16(offset))
	}

	// Helper for BNE offset
	bne := func(offset int16) {
		add(0xC200)
		addImmVal(uint16(offset))
	}

	// Helper for JMP offset
	jmp := func(offset int16) {
		add(0xD000)
		addImmVal(uint16(offset))
	}

	// Calculate offset helper
	calcOffset := func(fromPC, toPC int) int16 {
		offset := int32(toPC) - int32(fromPC) - 4
		if offset < -32768 || offset > 32767 {
			panic(fmt.Sprintf("branch offset out of range: %d (from %d to %d)", offset, fromPC, toPC))
		}
		return int16(offset)
	}

	fmt.Println("Building input test ROM...")
	fmt.Println("Feature: Display a white 8x8 sprite that moves with arrow keys/WASD")

	// ============================================
	// FEATURE 1: Enable BG0 (required for display)
	// ============================================
	fmt.Println("  [1] Enabling BG0...")
	movImm16(7, 0x8008) // R7 = BG0_CONTROL address
	movImm(0, 0x01)     // R0 = enable BG0
	movMem(7, 0)

	// ============================================
	// FEATURE 2: Initialize palette (white color for sprite)
	// ============================================
	fmt.Println("  [2] Initializing palette 1, color 1 (white)...")
	// Set CGRAM address to palette 1, color 1
	movImm16(7, 0x8012) // R7 = CGRAM_ADDR
	movImm(0, 0x11)     // R0 = palette 1, color 1
	movMem(7, 0)

	// Write white color: RGB555 = 0x7FFF (low=0xFF, high=0x7F)
	movImm16(7, 0x8013) // R7 = CGRAM_DATA
	movImm(0, 0xFF)     // R0 = low byte (0xFF)
	movMem(7, 0)
	movImm(0, 0x7F)     // R0 = high byte (0x7F)
	movMem(7, 0)

	// ============================================
	// FEATURE 3: Load tile data to VRAM
	// ============================================
	fmt.Println("  [3] Loading tile data to VRAM...")
	// Set VRAM address to 0 (tile 0)
	movImm16(7, 0x800E) // R7 = VRAM_ADDR_L
	movImm(0, 0x00)     // R0 = address low byte
	movMem(7, 0)

	movImm16(7, 0x800F) // R7 = VRAM_ADDR_H
	movImm(0, 0x00)     // R0 = address high byte
	movMem(7, 0)

	// Set VRAM_DATA register address once (outside loop)
	movImm16(7, 0x8010) // R7 = VRAM_DATA register address
	// Set tile data value (0x11 = two pixels, both color index 1)
	movImm(0, 0x11)     // R0 = 0x11 (solid tile, color index 1)

	// Write 32 bytes of 0x11 (solid tile, color index 1)
	// Use R6 as counter
	movImm16(6, 32) // R6 = 32 (counter)

	tileLoopStart := len(code)
	// Write to VRAM_DATA (auto-increments VRAM address)
	movMem(7, 0) // MOV [R7], R0 (write 0x11 to VRAM_DATA)
	// Decrement counter
	subImm(6, 1) // SUB R6, #1
	// Check if done
	cmpImm(6, 0) // CMP R6, #0
	tileLoopEnd := len(code)
	bne(calcOffset(tileLoopEnd*2, tileLoopStart*2))

	// ============================================
	// FEATURE 4: Initialize sprite position (R0 = X, R1 = Y)
	// ============================================
	fmt.Println("  [4] Initializing sprite position...")
	movImm16(0, 160) // R0 = X position (160)
	movImm16(1, 100) // R1 = Y position (100)

	// ============================================
	// FEATURE 5: Wait for VBlank and write initial sprite
	// ============================================
	fmt.Println("  [5] Waiting for VBlank and writing initial sprite...")
	waitVBlankInitStart := len(code)
	movImm16(4, 0x803E) // R4 = VBLANK_FLAG address
	add(0x1000 | (2 << 8) | (5 << 4) | 4) // MOV R5, [R4] (read VBlank flag)
	movImm(7, 0)        // R7 = 0 (for comparison)
	cmpReg(5, 7)        // CMP R5, R7
	beqPC := len(code) * 2
	beq(calcOffset(beqPC, waitVBlankInitStart*2))

	// Write initial sprite to OAM
	// Set OAM address to sprite 0
	movImm16(7, 0x8014) // R7 = OAM_ADDR
	movImm(2, 0x00)     // R2 = Sprite 0
	movMem(7, 2)

	movImm16(7, 0x8015) // R7 = OAM_DATA

	// Write X position (low byte) from R0
	movReg(2, 0)        // R2 = R0 (X low)
	movMem(7, 2)

	// Write X position (high byte) = 0
	movImm(2, 0x00)     // R2 = 0 (X high)
	movMem(7, 2)

	// Write Y position from R1
	movReg(2, 1)        // R2 = R1 (Y)
	movMem(7, 2)

	// Write tile index = 0
	movImm(2, 0x00)     // R2 = 0 (Tile)
	movMem(7, 2)

	// Write attributes = palette 1 (0x01)
	movImm(2, 0x01)     // R2 = palette 1
	movMem(7, 2)

	// Write control = enable, 8x8 (0x01)
	movImm(2, 0x01)     // R2 = Enable, 8x8
	movMem(7, 2)

	// ============================================
	// FEATURE 6: Main loop (read input, update sprite)
	// ============================================
	fmt.Println("  [6] Setting up main loop with input handling...")
	mainLoop := len(code)

	// Wait for VBlank
	waitVBlankLoopStart := len(code)
	movImm16(4, 0x803E) // R4 = VBLANK_FLAG address
	add(0x1000 | (2 << 8) | (5 << 4) | 4) // MOV R5, [R4] (read VBlank flag)
	movImm(7, 0)        // R7 = 0
	cmpReg(5, 7)        // CMP R5, R7
	beqPC = len(code) * 2
	beq(calcOffset(beqPC, waitVBlankLoopStart*2))

	// ============================================
	// Read Input
	// ============================================
	// Latch controller
	movImm16(7, 0xA001) // R7 = CONTROLLER1_LATCH
	movImm(2, 0x01)     // R2 = latch
	movMem(7, 2)

	// Read controller state (16-bit)
	movImm16(7, 0xA000) // R7 = CONTROLLER1
	add(0x1000 | (2 << 8) | (2 << 4) | 7) // MOV R2, [R7] (16-bit read)
	// R2 now contains button state

	// Release latch
	movImm16(7, 0xA001) // R7 = CONTROLLER1_LATCH
	movImm(3, 0x00)     // R3 = release
	movMem(7, 3)

	// ============================================
	// Check UP button (bit 0) - decrement Y
	// ============================================
	movReg(3, 2)        // R3 = buttons
	andImm(3, 0x01)      // R3 = buttons & UP
	cmpImm(3, 0x01)      // CMP R3, #0x01
	skipUpPC := len(code) * 2
	bne(0)               // BNE skip_up (placeholder)
	subImm(1, 1)         // SUB R1, #1 (move up)
	skipUpTarget := len(code) * 2
	// Fix branch offset: PC of immediate is skipUpPC + 2
	code[(skipUpPC/2)+1] = uint16(calcOffset(skipUpPC+2, skipUpTarget))

	// ============================================
	// Check DOWN button (bit 1) - increment Y
	// ============================================
	movReg(3, 2)         // R3 = buttons
	andImm(3, 0x02)      // R3 = buttons & DOWN
	cmpImm(3, 0x02)      // CMP R3, #0x02
	skipDownPC := len(code) * 2
	bne(0)               // BNE skip_down (placeholder)
	addImm(1, 1)         // ADD R1, #1 (move down)
	skipDownTarget := len(code) * 2
	// Fix branch offset: PC of immediate is skipDownPC + 2
	code[(skipDownPC/2)+1] = uint16(calcOffset(skipDownPC+2, skipDownTarget))

	// ============================================
	// Check LEFT button (bit 2) - decrement X
	// ============================================
	movReg(3, 2)         // R3 = buttons
	andImm(3, 0x04)      // R3 = buttons & LEFT
	cmpImm(3, 0x04)      // CMP R3, #0x04
	skipLeftPC := len(code) * 2
	bne(0)               // BNE skip_left (placeholder)
	subImm(0, 1)         // SUB R0, #1 (move left)
	skipLeftTarget := len(code) * 2
	// Fix branch offset: PC of immediate is skipLeftPC + 2
	code[(skipLeftPC/2)+1] = uint16(calcOffset(skipLeftPC+2, skipLeftTarget))

	// ============================================
	// Check RIGHT button (bit 3) - increment X
	// ============================================
	movReg(3, 2)         // R3 = buttons
	andImm(3, 0x08)      // R3 = buttons & RIGHT
	cmpImm(3, 0x08)      // CMP R3, #0x08
	skipRightPC := len(code) * 2
	bne(0)               // BNE skip_right (placeholder)
	addImm(0, 1)         // ADD R0, #1 (move right)
	skipRightTarget := len(code) * 2
	// Fix branch offset: PC of immediate is skipRightPC + 2
	code[(skipRightPC/2)+1] = uint16(calcOffset(skipRightPC+2, skipRightTarget))

	// ============================================
	// Update sprite position in OAM
	// ============================================
	// Set OAM address to sprite 0
	movImm16(7, 0x8014) // R7 = OAM_ADDR
	movImm(2, 0x00)     // R2 = Sprite 0
	movMem(7, 2)

	movImm16(7, 0x8015) // R7 = OAM_DATA

	// Write X position (low byte) from R0
	movReg(2, 0)        // R2 = R0 (X low)
	movMem(7, 2)

	// Write X position (high byte) = 0 (for now, assume positive X)
	movImm(2, 0x00)     // R2 = 0 (X high)
	movMem(7, 2)

	// Write Y position from R1
	movReg(2, 1)        // R2 = R1 (Y)
	movMem(7, 2)

	// Write tile index = 0
	movImm(2, 0x00)     // R2 = 0 (Tile)
	movMem(7, 2)

	// Write attributes = palette 1 (0x01)
	movImm(2, 0x01)     // R2 = palette 1
	movMem(7, 2)

	// Write control = enable, 8x8 (0x01)
	movImm(2, 0x01)     // R2 = Enable, 8x8
	movMem(7, 2)

	// Jump back to main loop
	jmpPC := len(code) * 2
	jmp(calcOffset(jmpPC, mainLoop*2))

	// Build ROM file
	romSize := uint32(len(code) * 2)
	romData := make([]byte, 32+romSize)

	// Header
	binary.LittleEndian.PutUint32(romData[0:4], 0x46434D52) // "RMCF"
	binary.LittleEndian.PutUint16(romData[4:6], 1)          // Version
	binary.LittleEndian.PutUint32(romData[6:10], romSize)  // ROM size
	binary.LittleEndian.PutUint16(romData[10:12], 1)       // Entry bank
	binary.LittleEndian.PutUint16(romData[12:14], 0x8000)  // Entry offset
	binary.LittleEndian.PutUint16(romData[14:16], 0)       // Mapper flags

	// Code
	for i, word := range code {
		offset := 32 + (i * 2)
		binary.LittleEndian.PutUint16(romData[offset:offset+2], word)
	}

	// Write file
	if err := os.WriteFile(outputPath, romData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing ROM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Input test ROM created: %s\n", outputPath)
	fmt.Printf("  ROM size: %d bytes (%d instructions)\n", len(romData), len(code))
	fmt.Println("\nFeatures included:")
	fmt.Println("  [1] BG0 enabled")
	fmt.Println("  [2] Palette 1, color 1 = white")
	fmt.Println("  [3] Tile 0 = solid white (32 bytes of 0x11)")
	fmt.Println("  [4] Sprite position initialized (R0=X=160, R1=Y=100)")
	fmt.Println("  [5] Initial sprite written to OAM")
	fmt.Println("  [6] Main loop: Read input, update position, write to OAM")
	fmt.Println("\nControls:")
	fmt.Println("  Arrow Keys / WASD - Move sprite")
	fmt.Println("\nExpected result: White 8x8 block that moves with arrow keys/WASD")
}
