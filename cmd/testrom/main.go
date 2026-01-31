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
		fmt.Println("Usage: testrom <output.rom>")
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
		add(0x1000 | (uint16(reg1) << 4) | uint16(reg2))
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

	// Helper for CMP R1, R2 (register to register)
	cmpReg := func(reg1, reg2 uint8) {
		add(0xC000 | (uint16(reg1) << 4) | uint16(reg2))
	}

	// Helper for AND R, #imm
	andImm := func(reg uint8, val uint16) {
		add(0x6000 | (1 << 8) | (uint16(reg) << 4))
		addImmVal(val)
	}

	// Helper for XOR R, #imm
	xorImm := func(reg uint8, val uint16) {
		add(0x8000 | (1 << 8) | (uint16(reg) << 4))
		addImmVal(val)
	}

	// Helper for SHL R, #imm
	shlImm := func(reg uint8, val uint16) {
		add(0xA000 | (1 << 8) | (uint16(reg) << 4))
		addImmVal(val)
	}

	// Helper for SHR R, #imm
	shrImm := func(reg uint8, val uint16) {
		add(0xB000 | (1 << 8) | (uint16(reg) << 4))
		addImmVal(val)
	}

	// Helper for BNE offset
	bne := func(offset int16) {
		add(0xC200)
		addImmVal(uint16(offset))
	}

	// Helper for BGE offset
	bge := func(offset int16) {
		add(0xC500)
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

	// Initialize: Block position (R0 = X: 160, R1 = Y: 100)
	movImm16(0, 160) // R0 = X position
	movImm16(1, 100) // R1 = Y position

	// Block color palette (R2 = 1)
	movImm16(2, 1)

	// Background color palette (R3 = 0)
	movImm16(3, 0)

	// Current note index (R4 = 0)
	movImm16(4, 0)

	// Note timer (R5 = 0, counts frames)
	movImm16(5, 0)

	// Sound enabled flag (R6 = 1, enabled by default)
	movImm16(6, 1)

	// Enable BG0
	movImm16(7, 0x8008) // R7 = BG0_CONTROL address
	movImm(0, 0x01)     // R0 = enable BG0 (temporarily use R0)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Set up initial palette colors
	// Background color (palette 0, color 0 = blue)
	movImm16(7, 0x8012) // R7 = CGRAM_ADDR
	movImm(0, 0x00)     // R0 = address 0 (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	movImm16(7, 0x8013) // R7 = CGRAM_DATA
	// Write blue color: RGB555 = 0x001F (low=0x1F, high=0x00)
	// CGRAM_DATA is 8-bit, so we need TWO writes: low byte, then high byte
	movImm(0, 0x1F) // R0 = low byte (0x1F)
	movMem(7, 0)
	movImm(0, 0x00) // R0 = high byte (0x00)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Block color (palette 1, color 1 = white)
	// Note: Color index 0 is transparent for sprites, so we use color 1
	movImm16(7, 0x8012) // R7 = CGRAM_ADDR
	movImm(0, 0x11)     // R0 = palette 1, color 1 (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	movImm16(7, 0x8013) // R7 = CGRAM_DATA
	// Write white color: RGB555 = 0x7FFF (low=0xFF, high=0x7F)
	// CGRAM_DATA is 8-bit, so we need TWO writes: low byte, then high byte
	movImm(0, 0xFF) // R0 = low byte (0xFF)
	movMem(7, 0)
	movImm(0, 0x7F) // R0 = high byte (0x7F)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Write tile data to VRAM (8x8 tile = 32 bytes)
	// Create a solid tile using color index 1 (white from palette 1)
	// 4bpp format: 2 pixels per byte, even pixels in upper 4 bits, odd in lower 4 bits
	// For a solid tile, all pixels = 0x11 (color index 1 in both upper and lower 4 bits)
	movImm16(7, 0x800E) // R7 = VRAM_ADDR_L
	movImm(0, 0x00)     // R0 = address low byte (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	movImm16(7, 0x800F) // R7 = VRAM_ADDR_H
	movImm(0, 0x00)     // R0 = address high byte (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	movImm16(7, 0x8010) // R7 = VRAM_DATA
	// Write 32 bytes of 0x11 (solid tile, color index 1)
	// Use R6 as counter (temporarily, it's already set to 1 for sound)
	movReg(6, 6)    // R6 = R6 (preserve sound flag, but we'll use it as counter)
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

	// Restore R6 (sound flag)
	movImm16(6, 1)

	// ============================================
	// Main loop starts here
	// ============================================
	mainLoop := len(code) // Word index - capture AFTER initialization, BEFORE main loop code

	// ============================================
	// CRITICAL: Wait for VBlank BEFORE doing any updates
	// OAM writes are only allowed during VBlank period (scanlines 200-219)
	// If we write during visible rendering, the writes are IGNORED!
	// Pattern: MOV R4, #0x803E; MOV R5, [R4]; MOV R7, #0; CMP R5, R7; BEQ wait_start
	// We wait for flag to be 1 (VBlank started), not 0
	// ============================================
	waitVBlankStart := len(code)
	movImm16(4, 0x803E)                   // R4 = VBLANK_FLAG address
	add(0x1000 | (2 << 8) | (5 << 4) | 4) // MOV R5, [R4] (mode 2: read 8-bit from I/O, zero-extend)
	movImm(7, 0)                          // R7 = 0 (for comparison)
	cmpReg(5, 7)                          // CMP R5, R7 (compare flag with 0)
	// BEQ = Branch if Equal, so if flag == 0, keep waiting (wait for flag to be 1)
	beqPC := len(code) * 2 // Current position (before BEQ)
	beq := func(offset int16) {
		add(0xC100)
		addImmVal(uint16(offset))
	}
	beq(calcOffset(beqPC, waitVBlankStart*2)) // BEQ wait_vblank_start (if flag is 0, keep waiting)

	// ============================================
	// Input Reading
	// ============================================
	// Latch controller
	movImm16(7, 0xA001) // R7 = CONTROLLER1_LATCH
	movImm(0, 0x01)     // R0 = latch (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	// Read controller state (16-bit)
	movImm16(7, 0xA000)                   // R7 = CONTROLLER1
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

	// Check A button (bit 4) - change block color
	movReg(7, 0)    // R7 = buttons
	andImm(7, 0x10) // R7 = buttons & A
	cmpImm(7, 0x10)
	skipAPC := len(code)
	bne(0)
	addImm(2, 1)    // ADD R2, #1 (increment block color palette)
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

	// ============================================
	// FIX 3: Update CGRAM with new background color
	// ============================================
	// Write new background color to CGRAM (palette 0, color 0)
	// Use R3 (palette value) to select color
	// Simple approach: use palette value to shift into different color components
	// For simplicity, cycle through: blue, green, red, yellow, cyan, magenta, white
	// Use R5 to calculate color value
	movReg(5, 3) // R5 = palette value (R3)

	// Simple color selection: use palette & 3 to get 4 colors, then cycle
	// palette & 1 determines blue/green, palette & 2 determines red
	andImm(5, 0x03) // R5 = palette & 3 (0-3)
	cmpImm(5, 0)
	bgColor0PC := len(code)
	bne(0)
	movImm16(5, 0x001F) // R5 = blue
	bgColor0Target := len(code)
	jmp(calcOffset(len(code)*2, bgColor0Target*2))
	code[bgColor0PC+1] = uint16(calcOffset((bgColor0PC+1)*2, bgColor0Target*2))

	cmpImm(5, 1)
	bgColor1PC := len(code)
	bne(0)
	movImm16(5, 0x03E0) // R5 = green
	bgColor1Target := len(code)
	jmp(calcOffset(len(code)*2, bgColor1Target*2))
	code[bgColor1PC+1] = uint16(calcOffset((bgColor1PC+1)*2, bgColor1Target*2))

	cmpImm(5, 2)
	bgColor2PC := len(code)
	bne(0)
	movImm16(5, 0x7C00) // R5 = red
	bgColor2Target := len(code)
	jmp(calcOffset(len(code)*2, bgColor2Target*2))
	code[bgColor2PC+1] = uint16(calcOffset((bgColor2PC+1)*2, bgColor2Target*2))

	// Default: yellow (for palette & 3 == 3)
	movImm16(5, 0x7FE0) // R5 = yellow

	bgColorEnd := len(code)
	// Patch all jump targets to bgColorEnd
	code[bgColor0Target+1] = uint16(calcOffset((bgColor0Target+1)*2, bgColorEnd*2))
	code[bgColor1Target+1] = uint16(calcOffset((bgColor1Target+1)*2, bgColorEnd*2))
	code[bgColor2Target+1] = uint16(calcOffset((bgColor2PC+1)*2, bgColorEnd*2))

	// Now write color to CGRAM (palette 0, color 0)
	movImm16(7, 0x8012) // R7 = CGRAM_ADDR
	movImm(0, 0x00)     // R0 = address 0 (palette 0, color 0) (temporarily)
	movMem(7, 0)
	movImm16(0, 160) // Restore R0

	movImm16(7, 0x8013) // R7 = CGRAM_DATA
	// CGRAM_DATA is 8-bit, so we need TWO writes: low byte, then high byte
	// Write low byte (R5 & 0xFF)
	movReg(0, 5)    // R0 = color value (temporarily)
	andImm(0, 0xFF) // R0 = R0 & 0xFF (mask to low byte)
	movMem(7, 0)    // Write low byte
	// Write high byte ((R5 >> 8) & 0xFF)
	movReg(0, 5)     // R0 = color value
	shrImm(0, 8)     // R0 = R0 >> 8 (shift to get high byte)
	andImm(0, 0xFF)  // R0 = R0 & 0xFF (mask to byte)
	movMem(7, 0)     // Write high byte
	movImm16(0, 160) // Restore R0

	skipBTarget := len(code)
	code[skipBPC+1] = uint16(calcOffset((skipBPC+1)*2, skipBTarget*2))

	// Check X button (bit 6) - toggle sound
	movReg(7, 0)    // R7 = buttons
	andImm(7, 0x40) // R7 = buttons & X
	cmpImm(7, 0x40)
	skipXPC := len(code)
	bne(0)
	// Toggle sound flag (XOR with 1)
	xorImm(6, 0x01) // R6 = R6 XOR 1
	skipXTarget := len(code)
	code[skipXPC+1] = uint16(calcOffset((skipXPC+1)*2, skipXTarget*2))

	// Update audio (play scale)
	// Increment note timer (R5)
	addImm(5, 1)

	// Check if sound is enabled (R6)
	cmpImm(6, 0x01)
	skipSoundCheckPC := len(code)
	bne(0) // If not enabled, skip audio update and disable channel
	// Sound is disabled - disable channel
	movImm16(7, 0x9003) // R7 = CH0_CONTROL
	movImm(1, 0x00)     // R1 = disable
	movMem(7, 1)
	movImm16(1, 100) // Restore R1
	skipSoundDisableTarget := len(code)
	jmp(calcOffset(len(code)*2, skipSoundDisableTarget*2))

	// Fix sound check branch
	code[skipSoundCheckPC+1] = uint16(calcOffset((skipSoundCheckPC+1)*2, skipSoundDisableTarget*2))

	// Sound is enabled - continue with audio logic
	skipSoundCheckTarget := len(code)

	// Check if 90 frames (1.5 seconds) have passed - time to move to next note
	cmpImm(5, 90)
	skipNextNotePC := len(code)
	bne(0) // If timer < 90, not time for next note yet

	// Reset timer
	movImm16(5, 0)

	// Increment note index (R4), cycle 0-7
	addImm(4, 1)
	andImm(4, 0x07) // Keep in range 0-7

	skipNextNoteTarget := len(code)
	code[skipNextNotePC+1] = uint16(calcOffset((skipNextNotePC+1)*2, skipNextNoteTarget*2))

	// Check if we're in note phase (first 60 frames = 1 second)
	cmpImm(5, 60)
	skipNotePhasePC := len(code)
	bge(0) // If timer >= 60, skip to silence phase
	// We're in note phase (timer < 60) - play current note
	// Use note index (R4) to select frequency from scale
	// For simplicity, we'll use hardcoded frequencies based on R4 value
	// We'll need to do a switch-like structure, but for now use approximation
	// freq ≈ 262 + (R4 * 35) gives us a rough scale
	movReg(7, 4)   // R7 = note index
	shlImm(7, 5)   // R7 = note index * 32
	addImm(7, 30)  // Add 30 more for better approximation
	addImm(7, 232) // R7 = 262 + (note index * 32) + 30 (approximate)

	// Set frequency low byte
	movImm16(1, 0x9000) // R1 = CH0_FREQ_LOW
	movReg(0, 7)        // R0 = frequency low byte (temporarily)
	movMem(1, 0)
	movImm16(0, 160) // Restore R0

	// Set frequency high byte (most frequencies are < 512, so high byte is 0 or 1)
	// For frequencies 262-523, high byte is 0x01 for most
	movImm16(1, 0x9001) // R1 = CH0_FREQ_HIGH
	movImm(0, 0x01)     // R0 = high byte (most need 0x01)
	movMem(1, 0)
	movImm16(0, 160) // Restore R0

	movImm16(1, 0x9002) // R1 = CH0_VOLUME
	movImm(0, 0x80)     // R0 = volume 128
	movMem(1, 0)
	movImm16(0, 160) // Restore R0

	movImm16(1, 0x9003) // R1 = CH0_CONTROL
	movImm(0, 0x01)     // R0 = enable, sine wave
	movMem(1, 0)
	movImm16(0, 160) // Restore R0

	skipNotePhaseTarget := len(code)
	jmp(calcOffset(len(code)*2, skipSoundCheckTarget*2))

	// Fix note phase branch
	code[skipNotePhasePC+1] = uint16(calcOffset((skipNotePhasePC+1)*2, skipNotePhaseTarget*2))

	// We're in silence phase (timer >= 60 and < 90) - disable channel
	movImm16(7, 0x9003) // R7 = CH0_CONTROL
	movImm(1, 0x00)     // R1 = disable
	movMem(7, 1)
	movImm16(1, 100) // Restore R1

	// ============================================
	// FIX 2: OAM writes happen AFTER VBlank wait
	// (We already waited for VBlank at the start)
	// ============================================
	// Update sprite position (write to OAM)
	movImm16(7, 0x8014) // R7 = OAM_ADDR
	movImm(1, 0x00)     // R1 = Sprite 0 (temporarily)
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

	// Write attributes (palette from R2, but use color index 1, not 0)
	// Attributes: bits [3:0] = palette index, but we want to use color 1 in the tile
	// Actually, the palette index in attributes selects which 16-color palette to use
	// The color index in tile data selects which color within that palette
	// So we keep palette from R2, and the tile data uses color index 1
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
	movReg(1, 0)        // R1 = X position (temporarily)
	movMem(7, 1)
	movImm16(1, 100) // Restore R1

	movImm16(7, 0x8002) // R7 = BG0_SCROLLY_L
	movReg(1, 1)        // R1 = Y position (temporarily)
	movMem(7, 1)
	movImm16(1, 100) // Restore R1

	// ============================================
	// FIX: Remove VBlank clear wait - it causes infinite loops
	// The flag stays 1 during entire VBlank period, so waiting for it to clear
	// will never complete if we're in VBlank. Instead, we just continue and
	// let the next frame handle synchronization naturally.
	// ============================================
	// Removed: wait for VBlank to clear (was causing infinite loop)

	// Jump back to main loop
	// Calculate offset: JMP adds 2 words (instruction + offset), so PC after JMP will be len(code)*2 + 4
	jmpPC := len(code) * 2 // Current position (before JMP)
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

	fmt.Printf("Test ROM created: %s\n", outputPath)
	fmt.Printf("ROM size: %d bytes (%d instructions)\n", len(romData), len(code))
}
