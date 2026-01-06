package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

// Musical scale frequencies (in Hz) - C major scale
var scaleFrequencies = []uint16{
	262, // C4
	294, // D4
	330, // E4
	349, // F4
	392, // G4
	440, // A4
	494, // B4
	523, // C5
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: demorom <output.rom>")
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

	// Helper for MOV R, #imm
	movImm := func(reg, val uint8) {
		add(0x1000 | (1 << 8) | (uint16(reg) << 4))
		addImmVal(uint16(val))
	}

	// Helper for MOV R, #imm (16-bit)
	movImm16 := func(reg uint8, val uint16) {
		add(0x1000 | (1 << 8) | (uint16(reg) << 4))
		addImmVal(val)
	}

	// Helper for MOV R1, R2
	movReg := func(reg1, reg2 uint8) {
		add(0x1000 | (0 << 8) | (uint16(reg1) << 4) | uint16(reg2))
	}

	// Helper for MOV [R1], R2 (16-bit write)
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

	// Helper for CMP R, #imm
	cmpImm := func(reg uint8, val uint16) {
		add(0xC000 | (1 << 8) | (uint16(reg) << 4))
		addImmVal(val)
	}

	// Helper for AND R, #imm
	andImm := func(reg uint8, val uint16) {
		add(0x6000 | (1 << 8) | (uint16(reg) << 4))
		addImmVal(val)
	}

	// Helper for SHL R, #imm
	shlImm := func(reg uint8, val uint16) {
		add(0xA000 | (1 << 8) | (uint16(reg) << 4))
		addImmVal(val)
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

	// Initialize: Sprite position (R0 = X: 160, R1 = Y: 100)
	movImm16(0, 160) // R0 = X position
	movImm16(1, 100) // R1 = Y position

	// Sprite color palette (R2 = 1, palette 1)
	movImm16(2, 1)

	// Background color palette (R3 = 0, palette 0)
	movImm16(3, 0)

	// Current note index (R4 = 0)
	movImm16(4, 0)

	// Note timer (R5 = 0, counts frames)
	movImm16(5, 0)

	// Enable BG0
	movImm16(7, 0x8008) // R7 = BG0_CONTROL address
	movImm(0, 0x01)      // R0 = enable BG0 (temporarily use R0)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Set up palette colors
	// Background color (palette 0, color 0 = blue)
	movImm16(7, 0x8012) // R7 = CGRAM_ADDR
	movImm(0, 0x00)      // R0 = address 0 (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	movImm16(7, 0x8013) // R7 = CGRAM_DATA
	movImm16(0, 0x001F) // R0 = blue color (0x001F: low=0x1F, high=0x00)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Sprite color (palette 1, color 1 = white)
	// Note: Color index 0 is transparent for sprites, so we use color 1
	movImm16(7, 0x8012) // R7 = CGRAM_ADDR
	movImm(0, 0x11)     // R0 = palette 1, color 1 (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	movImm16(7, 0x8013) // R7 = CGRAM_DATA
	movImm16(0, 0x7FFF) // R0 = white color (0x7FFF: low=0xFF, high=0x7F)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Write tile data to VRAM (8x8 tile = 32 bytes)
	// Create a solid tile using color index 1 (white from palette 1)
	movImm16(7, 0x800E) // R7 = VRAM_ADDR_L
	movImm(0, 0x00)      // R0 = address low byte (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	movImm16(7, 0x800F) // R7 = VRAM_ADDR_H
	movImm(0, 0x00)      // R0 = address high byte (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	movImm16(7, 0x8010) // R7 = VRAM_DATA
	// Write 32 bytes of 0x11 (solid tile, color index 1)
	movImm16(6, 32) // R6 = 32 (counter)

	tileLoopStart := len(code)
	// Write 0x11 to VRAM_DATA
	movImm(0, 0x11) // R0 = 0x11 (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0
	// Decrement counter
	subImm(6, 1)
	// Check if done
	cmpImm(6, 0)
	tileLoopEnd := len(code)
	bne(calcOffset(tileLoopEnd*2, tileLoopStart*2))

	// Main loop
	mainLoop := len(code) // Word index

	// Latch controller
	movImm16(7, 0xA001) // R7 = CONTROLLER1_LATCH
	movImm(0, 0x01)     // R0 = latch (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Read controller state (16-bit)
	movImm16(7, 0xA000) // R7 = CONTROLLER1
	add(0x1000 | (2 << 8) | (0 << 4) | 7) // MOV R0, [R7] (16-bit read)
	// R0 now contains button state

	// Release latch
	movImm16(7, 0xA001) // R7 = CONTROLLER1_LATCH
	movImm(1, 0x00)     // R1 = release (temporarily use R1)
	movMem(7, 1)
	movImm16(1, 100) // Restore R1

	// Check UP button (bit 0)
	movReg(7, 0)    // R7 = buttons
	andImm(7, 0x01) // R7 = buttons & UP
	cmpImm(7, 0x01)
	skipUpPC := len(code)
	bne(0)
	subImm(1, 1) // SUB R1, #1 (move up)
	skipUpTarget := len(code)
	code[skipUpPC+1] = uint16(calcOffset((skipUpPC+1)*2, skipUpTarget*2))

	// Check DOWN button (bit 1)
	movReg(7, 0)    // R7 = buttons
	andImm(7, 0x02) // R7 = buttons & DOWN
	cmpImm(7, 0x02)
	skipDownPC := len(code)
	bne(0)
	addImm(1, 1) // ADD R1, #1 (move down)
	skipDownTarget := len(code)
	code[skipDownPC+1] = uint16(calcOffset((skipDownPC+1)*2, skipDownTarget*2))

	// Check LEFT button (bit 2)
	movReg(7, 0)    // R7 = buttons
	andImm(7, 0x04) // R7 = buttons & LEFT
	cmpImm(7, 0x04)
	skipLeftPC := len(code)
	bne(0)
	subImm(0, 1) // SUB R0, #1 (move left)
	skipLeftTarget := len(code)
	code[skipLeftPC+1] = uint16(calcOffset((skipLeftPC+1)*2, skipLeftTarget*2))

	// Check RIGHT button (bit 3)
	movReg(7, 0)    // R7 = buttons
	andImm(7, 0x08) // R7 = buttons & RIGHT
	cmpImm(7, 0x08)
	skipRightPC := len(code)
	bne(0)
	addImm(0, 1) // ADD R0, #1 (move right)
	skipRightTarget := len(code)
	code[skipRightPC+1] = uint16(calcOffset((skipRightPC+1)*2, skipRightTarget*2))

	// Check A button (bit 4) - change sprite color
	movReg(7, 0)    // R7 = buttons
	andImm(7, 0x10) // R7 = buttons & A
	cmpImm(7, 0x10)
	skipAPC := len(code)
	bne(0)
	addImm(2, 1)    // ADD R2, #1 (increment sprite color palette)
	andImm(2, 0x0F) // AND R2, #0x0F (keep in range 0-15)
	skipATarget := len(code)
	code[skipAPC+1] = uint16(calcOffset((skipAPC+1)*2, skipATarget*2))

	// Check B button (bit 5) - change background color
	movReg(7, 0)    // R7 = buttons
	andImm(7, 0x20) // R7 = buttons & B
	cmpImm(7, 0x20)
	skipBPC := len(code)
	bne(0)
	addImm(3, 1)    // ADD R3, #1 (increment background color palette)
	andImm(3, 0x0F) // AND R3, #0x0F (keep in range 0-15)
	skipBTarget := len(code)
	code[skipBPC+1] = uint16(calcOffset((skipBPC+1)*2, skipBTarget*2))

	// Update audio (play scale)
	// Increment note timer (R5)
	addImm(5, 1)

	// Check if 60 frames (1 second) have passed - time to move to next note
	cmpImm(5, 60)
	skipNextNotePC := len(code)
	bne(0) // If timer < 60, not time for next note yet

	// Reset timer
	movImm16(5, 0)

	// Increment note index (R4), cycle 0-7
	addImm(4, 1)
	andImm(4, 0x07) // Keep in range 0-7

	skipNextNoteTarget := len(code)
	code[skipNextNotePC+1] = uint16(calcOffset((skipNextNotePC+1)*2, skipNextNoteTarget*2))

	// Play current note based on R4
	// We'll use a lookup table approach - calculate frequency from note index
	// For simplicity, use approximate frequencies: 262 + (R4 * 35)
	movReg(7, 4)    // R7 = note index
	shlImm(7, 5)    // R7 = note index * 32
	addImm(7, 30)   // Add 30 more for better approximation
	addImm(7, 232)  // R7 = 262 + (note index * 32) + 30 (approximate)

	// Set frequency low byte
	movImm16(6, 0x9000) // R6 = CH0_FREQ_LOW
	movReg(0, 7)         // R0 = frequency low byte (temporarily)
	movMem(6, 0)
	movImm16(0, 160) // Restore R0

	// Set frequency high byte (most frequencies are < 512, so high byte is 0 or 1)
	movImm16(6, 0x9001) // R6 = CH0_FREQ_HIGH
	movImm(0, 0x01)     // R0 = high byte (most need 0x01)
	movMem(6, 0)
	movImm16(0, 160) // Restore R0

	movImm16(6, 0x9002) // R6 = CH0_VOLUME
	movImm(0, 0x80)     // R0 = volume 128
	movMem(6, 0)
	movImm16(0, 160) // Restore R0

	movImm16(6, 0x9003) // R6 = CH0_CONTROL
	movImm(0, 0x01)     // R0 = enable, sine wave
	movMem(6, 0)
	movImm16(0, 160) // Restore R0

	// Update sprite position (write to OAM)
	movImm16(7, 0x8014) // R7 = OAM_ADDR
	movImm(1, 0x00)      // R1 = Sprite 0 (temporarily)
	movMem(7, 1)
	movImm16(1, 100) // Restore R1

	movImm16(7, 0x8015) // R7 = OAM_DATA

	// Write X position (low byte)
	movReg(1, 0) // R1 = X position (temporarily)
	movMem(7, 1)
	movImm16(1, 100) // Restore R1

	// Write X position (high byte, sign extend)
	movImm(1, 0x00) // R1 = 0 (temporarily)
	movMem(7, 1)
	movImm16(1, 100) // Restore R1

	// Write Y position
	movReg(1, 1) // R1 = Y position (temporarily)
	movMem(7, 1)
	movImm16(1, 100) // Restore R1

	// Write tile index (simple block tile)
	movImm(1, 0x00) // R1 = 0 (temporarily)
	movMem(7, 1)
	movImm16(1, 100) // Restore R1

	// Write attributes (palette from R2)
	movReg(1, 2) // R1 = palette (temporarily)
	shlImm(1, 4) // R1 = palette << 4
	movMem(7, 1)
	movImm16(1, 100) // Restore R1

	// Write control (enable, 8x8)
	movImm(1, 0x01) // R1 = Enable, 8x8 (temporarily)
	movMem(7, 1)
	movImm16(1, 100) // Restore R1

	// Update background scroll (for testing)
	movImm16(7, 0x8000) // R7 = BG0_SCROLLX_L
	movReg(1, 0)         // R1 = X position (temporarily)
	movMem(7, 1)
	movImm16(1, 100) // Restore R1

	movImm16(7, 0x8002) // R7 = BG0_SCROLLY_L
	movReg(1, 1)         // R1 = Y position (temporarily)
	movMem(7, 1)
	movImm16(1, 100) // Restore R1

	// Delay loop (wait for next frame)
	movImm16(7, 0) // R7 = counter

	delayLoopStart := len(code)
	addImm(7, 1)
	cmpImm(7, 0x1000) // Delay ~4096 iterations
	delayLoopEnd := len(code)
	bne(calcOffset(delayLoopEnd*2, delayLoopStart*2))

	// Jump back to main loop
	jmp(calcOffset(len(code)*2, mainLoop*2))

	// Build ROM file
	romSize := uint32(len(code) * 2)
	romData := make([]byte, 32+romSize)

	// Header
	binary.LittleEndian.PutUint32(romData[0:4], 0x46434D52) // "RMCF"
	binary.LittleEndian.PutUint16(romData[4:6], 1)          // Version
	binary.LittleEndian.PutUint32(romData[6:10], romSize)   // ROM size
	binary.LittleEndian.PutUint16(romData[10:12], 1)        // Entry bank
	binary.LittleEndian.PutUint16(romData[12:14], 0x8000)   // Entry offset
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

	fmt.Printf("Demo ROM created: %s\n", outputPath)
	fmt.Printf("ROM size: %d bytes (%d instructions)\n", len(romData), len(code))
}

