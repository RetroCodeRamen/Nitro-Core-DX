package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: testrom_minimal <output.rom>")
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

	// Helper for MOV [R1], R2 (store to memory)
	movMem := func(reg1, reg2 uint8) {
		add(0x1000 | (3 << 8) | (uint16(reg1) << 4) | uint16(reg2))
	}

	// Helper for BEQ offset
	beq := func(offset int16) {
		add(0xC100)
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

	fmt.Println("Building minimal test ROM...")
	fmt.Println("Feature: Display a white 8x8 sprite at position (160, 100)")

	// ============================================
	// FEATURE 1: Enable BG0 (required for display)
	// ============================================
	fmt.Println("  [1] Enabling BG0...")
	movImm16(7, 0x8008) // R7 = BG0_CONTROL address
	movImm(0, 0x01)     // R0 = enable BG0 (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// ============================================
	// FEATURE 2: Initialize palette (white color for sprite)
	// ============================================
	fmt.Println("  [2] Initializing palette 1, color 1 (white)...")
	// Set CGRAM address to palette 1, color 1
	movImm16(7, 0x8012) // R7 = CGRAM_ADDR
	movImm(0, 0x11)     // R0 = palette 1, color 1 (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Write white color: RGB555 = 0x7FFF (low=0xFF, high=0x7F)
	movImm16(7, 0x8013) // R7 = CGRAM_DATA
	movImm(0, 0xFF)    // R0 = low byte (0xFF)
	movMem(7, 0)
	movImm(0, 0x7F)    // R0 = high byte (0x7F)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

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
	add(0x3000 | (1 << 8) | (6 << 4)) // SUB R6, #1
	addImmVal(1)
	// Check if done
	add(0xC000 | (1 << 8) | (6 << 4)) // CMP R6, #0
	addImmVal(0)
	tileLoopEnd := len(code)
	add(0xC200) // BNE tileLoopStart
	addImmVal(uint16(calcOffset(tileLoopEnd*2, tileLoopStart*2)))

	// ============================================
	// FEATURE 4: Wait for VBlank (required for OAM writes)
	// ============================================
	fmt.Println("  [4] Waiting for VBlank...")
	waitVBlankStart := len(code)
	movImm16(4, 0x803E) // R4 = VBLANK_FLAG address
	add(0x1000 | (2 << 8) | (5 << 4) | 4) // MOV R5, [R4] (mode 2: read 8-bit from I/O, zero-extend)
	movImm(7, 0)        // R7 = 0 (for comparison)
	add(0xC000 | (5 << 4) | 7) // CMP R5, R7 (compare flag with 0)
	beqPC := len(code) * 2
	beq(calcOffset(beqPC, waitVBlankStart*2)) // BEQ wait_vblank_start (if flag is 0, keep waiting)

	// ============================================
	// FEATURE 5: Write sprite to OAM
	// ============================================
	fmt.Println("  [5] Writing sprite to OAM...")
	// Set OAM address to sprite 0
	movImm16(7, 0x8014) // R7 = OAM_ADDR
	movImm(0, 0x00)     // R0 = Sprite 0 (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	movImm16(7, 0x8015) // R7 = OAM_DATA

	// Write X position (low byte) = 160
	movImm(0, 160)      // R0 = 160 (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Write X position (high byte) = 0
	movImm(0, 0x00)     // R0 = 0 (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Write Y position = 100
	movImm(0, 100)      // R0 = 100 (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Write tile index = 0
	movImm(0, 0x00)     // R0 = 0 (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Write attributes = palette 1 (0x01)
	movImm(0, 0x01)     // R0 = palette 1 (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Write control = enable, 8x8 (0x01)
	movImm(0, 0x01)     // R0 = Enable, 8x8 (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// ============================================
	// FEATURE 6: Main loop (just wait for VBlank and update)
	// ============================================
	fmt.Println("  [6] Setting up main loop...")
	mainLoop := len(code)

	// Wait for VBlank
	waitVBlankLoopStart := len(code)
	movImm16(4, 0x803E) // R4 = VBLANK_FLAG address
	add(0x1000 | (2 << 8) | (5 << 4) | 4) // MOV R5, [R4]
	movImm(7, 0)        // R7 = 0
	add(0xC000 | (5 << 4) | 7) // CMP R5, R7
	beqPC = len(code) * 2
	beq(calcOffset(beqPC, waitVBlankLoopStart*2))

	// Jump back to main loop
	jmpPC := len(code) * 2
	jmp(calcOffset(jmpPC, mainLoop*2))

	// Build ROM file
	romSize := uint32(len(code) * 2)
	romData := make([]byte, 32+romSize)

	// Header
	binary.LittleEndian.PutUint32(romData[0:4], 0x46434D52) // "RMCF"
	binary.LittleEndian.PutUint16(romData[4:6], 1)          // Version
	binary.LittleEndian.PutUint32(romData[6:10], romSize)   // ROM size
	binary.LittleEndian.PutUint16(romData[10:12], 1)        // Entry bank
	binary.LittleEndian.PutUint16(romData[12:14], 0x8000)   // Entry offset
	binary.LittleEndian.PutUint16(romData[14:16], 0)        // Mapper flags

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

	fmt.Printf("\n✓ Minimal test ROM created: %s\n", outputPath)
	fmt.Printf("  ROM size: %d bytes (%d instructions)\n", len(romData), len(code))
	fmt.Println("\nFeatures included:")
	fmt.Println("  [1] BG0 enabled")
	fmt.Println("  [2] Palette 1, color 1 = white")
	fmt.Println("  [3] Tile 0 = solid white (32 bytes of 0x11)")
	fmt.Println("  [4] VBlank wait")
	fmt.Println("  [5] Sprite 0 at (160, 100), tile 0, palette 1, enabled")
	fmt.Println("  [6] Main loop (waits for VBlank)")
	fmt.Println("\nExpected result: White 8x8 block at center of screen")
}
