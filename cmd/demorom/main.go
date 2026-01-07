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

	// Helper for BLT offset (branch if less than, signed)
	blt := func(offset int16) {
		add(0xC400)
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

	// Note timer - use single counter approach for simplicity
	// R6 = loop iteration counter (accumulates across all iterations)
	// R7 is used for addresses, so we need to save/restore R6 when using R7
	movImm16(6, 0) // Loop iteration counter (single counter for note timing)

	// Disable BG0 (we only want sprites for now)
	movImm16(7, 0x8008) // R7 = BG0_CONTROL address
	movImm(0, 0x00)      // R0 = disable BG0 (temporarily use R0)
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

	// Write tile data to VRAM
	// Tile 0: Keep blank (all zeros) - reserved as transparent/blank tile
	// Tile 1: Sprite tile (8x8 tile = 32 bytes)
	// Write tile 1 at VRAM address 0x0020 (32 bytes = tile 1)
	movImm16(7, 0x800E) // R7 = VRAM_ADDR_L
	movImm(0, 0x20)      // R0 = address low byte 0x20 (tile 1 starts at 32) (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	movImm16(7, 0x800F) // R7 = VRAM_ADDR_H
	movImm(0, 0x00)      // R0 = address high byte 0x00 (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	movImm16(7, 0x8010) // R7 = VRAM_DATA
	// Write 32 bytes of 0x11 (solid tile, color index 1)
	// Temporarily use R3 for tile loop counter (save current R3 value first)
	// Actually, R3 starts at 0 (background palette), so we can use it directly
	movImm16(3, 32) // R3 = 32 (counter) - temporarily

	tileLoopStart := len(code)
	// Write 0x11 to VRAM_DATA
	movImm(0, 0x11) // R0 = 0x11 (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0
	// Decrement counter
	subImm(3, 1)
	// Check if done
	cmpImm(3, 0)
	tileLoopEnd := len(code)
	bne(calcOffset(tileLoopEnd*2, tileLoopStart*2))
	// Restore R3 (background palette)
	movImm16(3, 0) // Restore R3 to 0 (background palette)

	// Initialize audio - play first note immediately
	// Use lookup table approach: store frequencies in memory or use direct values
	// For now, use direct frequency value for note 0 (262 Hz = 0x0106)
	movImm16(7, 262) // R7 = frequency for note 0 (C4 = 262 Hz)

	// Set frequency low byte (use R7 for address, preserve R6!)
	movImm16(7, 0x9000) // R7 = CH0_FREQ_LOW
	movReg(0, 7)         // R0 = frequency (temporarily, from R7 calculation above)
	andImm(0, 0xFF)      // R0 = low byte only
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Set frequency high byte
	movImm16(7, 0x9001) // R7 = CH0_FREQ_HIGH
	movImm(0, 0x01)     // R0 = high byte
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	movImm16(7, 0x9002) // R7 = CH0_VOLUME
	movImm(0, 0x80)     // R0 = volume 128
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	movImm16(7, 0x9003) // R7 = CH0_CONTROL
	movImm(0, 0x01)     // R0 = enable, sine wave
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Main loop
	mainLoop := len(code) // Word index
	// NOTE: R6 is NOT reset here - it accumulates across loop iterations
	// It only resets when the threshold is reached (in the audio update section)

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

	// Update audio (play scale) - single counter approach
	// Increment loop iteration counter (R6) - accumulates across all loop iterations
	addImm(6, 1)

	// Check if enough iterations have passed for one note (approximately 1 second)
	// The loop runs many times per frame (~2000-3000 iterations per frame)
	// For 1 second = 60 frames, we need ~120,000-180,000 total iterations
	// Start with 30000 and tune based on actual behavior
	// This is simpler and more reliable than trying to count "frames"
	cmpImm(6, 30000)
	skipNextNotePC := len(code)
	blt(0) // If R6 < 30000, skip note update

	// Reset counter FIRST before updating frequency
	// This ensures we only update notes when we've counted enough iterations
	movImm16(6, 0)

	// Increment note index (R4), cycle 0-7
	addImm(4, 1)
	andImm(4, 0x07) // Keep in range 0-7

	// Play current note based on R4 (only when timer resets)
	// Use lookup table: calculate address offset for frequency lookup
	// Store frequencies in WRAM starting at address 0x0000
	// Frequencies: 262, 294, 330, 349, 392, 440, 494, 523
	// For simplicity, use a switch-like structure with direct values
	// R4 = note index (0-7), we'll use a series of comparisons
	
	// For now, use approximate calculation but with better accuracy
	// Note frequencies: 262, 294, 330, 349, 392, 440, 494, 523
	// Better approximation: base + (note * step) where step varies
	// Actually, let's use direct lookup via memory or better calculation
	
	// Simplified: use note index to calculate approximate frequency
	// For better accuracy, we'll use: 262 + note_index * 32 (close enough for demo)
	// But actually, let's use a more accurate method:
	// For notes 0-1: exact (262, 294)
	// For notes 2-7: use better approximation
	
	// Use direct frequency lookup - store in R7 based on note index
	// We'll use a series of comparisons to set the correct frequency
	// For simplicity in demo, use: 262 + (note_index * 32) for now
	movReg(7, 4)    // R7 = note index
	shlImm(7, 5)    // R7 = note index * 32
	addImm(7, 262)  // R7 = 262 + (note index * 32) - approximate

	// Set frequency LOW byte FIRST
	// R7 already has the frequency from calculation above
	movReg(0, 7)    // R0 = frequency (from R7)
	andImm(0, 0xFF) // R0 = low byte only
	movImm16(7, 0x9000) // R7 = CH0_FREQ_LOW address (R6 preserved!)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Set frequency HIGH byte SECOND (this completes the frequency update)
	// Writing the high byte triggers phase reset in APU for clean note start
	// Calculate high byte: check if note index == 7 (523 Hz needs 0x02, others need 0x01)
	cmpImm(4, 7)    // Compare note index with 7
	skipHighByte2PC := len(code)
	bne(0)          // If not equal, skip to set 0x01
	movImm(0, 0x02) // R0 = high byte 0x02 (for note 7 = 523 Hz)
	skipHighByteTarget := len(code)
	jmp(calcOffset(len(code)*2, skipHighByteTarget*2))
	skipHighByte2Target := len(code)
	code[skipHighByte2PC+1] = uint16(calcOffset((skipHighByte2PC+1)*2, skipHighByte2Target*2))
	movImm(0, 0x01) // R0 = high byte 0x01 (for notes 0-6, frequencies 256-511)
	skipHighByteTarget = len(code)
	code[skipHighByteTarget-1] = uint16(calcOffset((skipHighByteTarget-1)*2, skipHighByteTarget*2))
	
	movImm16(7, 0x9001) // R7 = CH0_FREQ_HIGH (R6 preserved!)
	movMem(7, 0)         // Write high byte (triggers phase reset in APU)
	movImm16(0, 160) // Restore R0
	// NOTE: Channel stays enabled - phase resets automatically on FREQ_HIGH write
	// R6 (loop counter) is preserved since we used R7 for addresses

	skipNextNoteTarget := len(code)
	code[skipNextNotePC+1] = uint16(calcOffset((skipNextNotePC+1)*2, skipNextNoteTarget*2))

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

	// Write tile index (tile 1 - tile 0 is reserved as blank)
	movImm(1, 0x01) // R1 = 1 (temporarily)
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

	// Jump back to main loop (emulator handles frame timing)
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

