package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

// Musical note frequencies (in Hz) - C major scale
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
		fmt.Println("Usage: audiotest <output.rom>")
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

	// Helper for MOV R1, R2
	movReg := func(reg1, reg2 uint8) {
		add(0x1000 | (0 << 8) | (uint16(reg1) << 4) | uint16(reg2))
	}

	// Helper for MOV [R1], R2 (8-bit write)
	movMem := func(reg1, reg2 uint8) {
		add(0x1000 | (3 << 8) | (uint16(reg1) << 4) | uint16(reg2))
	}

	// Helper for MOV R1, [R2] (16-bit read)
	movMemRead := func(reg1, reg2 uint8) {
		add(0x1000 | (2 << 8) | (uint16(reg1) << 4) | uint16(reg2))
	}

	// Helper for ADD R, #imm
	addImm := func(reg uint8, val uint16) {
		add(0x2000 | (1 << 8) | (uint16(reg) << 4))
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

	// Helper for BEQ offset (branch if equal)
	beq := func(offset int16) {
		// BEQ: opcode 0xC, mode field has opcode 0x1 (BEQ)
		// Format: 0xC[1][0][0][0] = 0xC100
		add(0xC100)
		addImmVal(uint16(offset))
	}

	// Helper for BLT offset (branch if less than)
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
	// Offset is relative to PC after instruction and offset word
	// So if instruction is at index i, PC after instruction+offset = (i+2)*2
	calcOffset := func(fromPC, toPC int) int16 {
		// fromPC is the PC after instruction and offset word (already accounts for both)
		// toPC is the target PC
		// offset = toPC - fromPC (no need to subtract 4, fromPC already accounts for it)
		offset := int32(toPC) - int32(fromPC)
		if offset < -32768 || offset > 32767 {
			panic(fmt.Sprintf("branch offset out of range: %d (from %d to %d)", offset, fromPC, toPC))
		}
		return int16(offset)
	}

	// ============================================
	// INITIALIZATION
	// ============================================
	
	// R4 = arpeggio note index (0-7)
	// R5 = chord index (0-2: C, F, G)
	// R7 = temporary register for addresses/calculations
	movImm16(4, 0) // Arpeggio note index
	movImm16(5, 0) // Chord index

	// Set master volume to maximum
	movImm16(7, 0x9020) // MASTER_VOLUME
	movImm16(0, 0xFF)    // R0 = volume 255 (max)
	movMem(7, 0)
	movImm16(0, 0)

	// ============================================
	// CHANNEL 0: Arpeggio (sine wave)
	// ============================================
	
	// Set volume
	movImm16(7, 0x9002) // CH0_VOLUME
	movImm16(0, 0xFF)    // R0 = volume 255 (max)
	movMem(7, 0)
	movImm16(0, 0)

	// Set initial frequency (C4 = 262 Hz)
	movImm16(7, 0x9000) // CH0_FREQ_LOW
	movImm16(0, 0x06)    // R0 = 262 & 0xFF = 0x06
	movMem(7, 0)
	movImm16(0, 0)

	movImm16(7, 0x9001) // CH0_FREQ_HIGH
	movImm16(0, 0x01)    // R0 = 262 >> 8 = 0x01
	movMem(7, 0)
	movImm16(0, 0)

	// Set duration to 60 frames (1 second)
	movImm16(7, 0x9004) // CH0_DURATION_LOW
	movImm16(0, 60)      // R0 = 60
	movMem(7, 0)
	movImm16(0, 0)

	movImm16(7, 0x9005) // CH0_DURATION_HIGH
	movImm16(0, 0)       // R0 = 0
	movMem(7, 0)
	movImm16(0, 0)

	// Set duration mode: stop when done
	movImm16(7, 0x9006) // CH0_DURATION_MODE
	movImm16(0, 0)       // R0 = 0 (stop mode)
	movMem(7, 0)
	movImm16(0, 0)

	// NOW enable the channel
	movImm16(7, 0x9003) // CH0_CONTROL
	movImm16(0, 0x01)    // R0 = enable, sine wave
	movMem(7, 0)
	movImm16(0, 0)

	// ============================================
	// CHANNEL 1: Chord root (square wave)
	// ============================================
	
	movImm16(7, 0x900A) // CH1_VOLUME (0x9000 + 1*8 + 2)
	movImm16(0, 0x60)    // R0 = volume 96
	movMem(7, 0)
	movImm16(0, 0)

	// Set initial frequency (C = 262 Hz)
	movImm16(7, 0x9008) // CH1_FREQ_LOW (0x9000 + 1*8 + 0)
	movImm16(0, 0x06)    // R0 = 262 & 0xFF
	movMem(7, 0)
	movImm16(0, 0)

	movImm16(7, 0x9009) // CH1_FREQ_HIGH (0x9000 + 1*8 + 1)
	movImm16(0, 0x01)    // R0 = 262 >> 8
	movMem(7, 0)
	movImm16(0, 0)

	// Set duration to 0 (play indefinitely for chords)
	movImm16(7, 0x900C) // CH1_DURATION_LOW (0x9000 + 1*8 + 4)
	movImm16(0, 0)       // R0 = 0 (infinite)
	movMem(7, 0)
	movImm16(0, 0)

	movImm16(7, 0x900D) // CH1_DURATION_HIGH (0x9000 + 1*8 + 5)
	movImm16(0, 0)       // R0 = 0
	movMem(7, 0)
	movImm16(0, 0)

	// NOW enable
	movImm16(7, 0x900B) // CH1_CONTROL (0x9000 + 1*8 + 3)
	movImm16(0, 0x03)    // R0 = enable, square wave
	movMem(7, 0)
	movImm16(0, 0)

	// ============================================
	// CHANNEL 2: Chord third (square wave)
	// ============================================
	
	movImm16(7, 0x9012) // CH2_VOLUME (0x9000 + 2*8 + 2)
	movImm16(0, 0x60)    // R0 = volume 96
	movMem(7, 0)
	movImm16(0, 0)

	// Set initial frequency (E = 330 Hz)
	movImm16(7, 0x9010) // CH2_FREQ_LOW (0x9000 + 2*8 + 0)
	movImm16(0, 0x4A)    // R0 = 330 & 0xFF = 0x4A
	movMem(7, 0)
	movImm16(0, 0)

	movImm16(7, 0x9011) // CH2_FREQ_HIGH (0x9000 + 2*8 + 1)
	movImm16(0, 0x01)    // R0 = 330 >> 8
	movMem(7, 0)
	movImm16(0, 0)

	// Set duration to 0 (play indefinitely for chords)
	movImm16(7, 0x9014) // CH2_DURATION_LOW (0x9000 + 2*8 + 4)
	movImm16(0, 0)       // R0 = 0 (infinite)
	movMem(7, 0)
	movImm16(0, 0)

	movImm16(7, 0x9015) // CH2_DURATION_HIGH (0x9000 + 2*8 + 5)
	movImm16(0, 0)       // R0 = 0
	movMem(7, 0)
	movImm16(0, 0)

	// NOW enable
	movImm16(7, 0x9013) // CH2_CONTROL (0x9000 + 2*8 + 3)
	movImm16(0, 0x03)    // R0 = enable, square wave
	movMem(7, 0)
	movImm16(0, 0)

	// ============================================
	// MAIN LOOP
	// ============================================
	
	mainLoop := len(code)

	// Check if channel 0 just finished (using completion status register)
	// Read 8-bit completion status (0x9021)
	movImm16(7, 0x9021) // CHANNEL_COMPLETION_STATUS
	movMemRead(6, 7)     // R6 = 16-bit read: (Read8(0x9022) << 8) | Read8(0x9021)
	                     // Since 0x9022 doesn't exist, high byte is 0, so R6 = completion status
	andImm(6, 0x01)      // R6 = bit 0 (channel 0 completion flag)
	cmpImm(6, 0)         // Compare with 0 (channel 0 didn't finish?)
	skipArpPC := len(code)
	beq(0) // Placeholder offset - will be patched later
	// BEQ branches if Zero flag is set (R6 == 0), so if R6 != 0, we fall through to note update

	// ============================================
	// CHANNEL 0 FINISHED - START NEXT NOTE
	// ============================================
	
	// Increment arpeggio note index
	addImm(4, 1)
	andImm(4, 0x07) // Keep in range 0-7

	// Calculate arpeggio frequency: approximate 262 + (note_index * 32)
	movReg(7, 4)    // R7 = note index
	shlImm(7, 5)    // R7 = note index * 32
	addImm(7, 262)  // R7 = 262 + (note index * 32)

	// Set Channel 0 frequency (low byte first)
	movReg(0, 7)    // R0 = frequency
	andImm(0, 0xFF) // R0 = low byte
	movImm16(7, 0x9000) // CH0_FREQ_LOW
	movMem(7, 0)
	movImm16(0, 0)

	// High byte: check if note index == 7 (523 Hz needs 0x02)
	cmpImm(4, 7)
	skipHighPC := len(code)
	blt(0) // Placeholder - will patch
	movImm16(0, 0x02) // High byte 0x02 for note 7
	skipHighTarget := len(code)
	jmp(calcOffset(len(code)*2, skipHighTarget*2))
	skipHighTarget2 := len(code)
	code[skipHighPC+1] = uint16(calcOffset((skipHighPC+2)*2, skipHighTarget2*2))
	movImm16(0, 0x01) // High byte 0x01 for notes 0-6
	skipHighTarget = len(code)
	code[skipHighTarget-1] = uint16(calcOffset((skipHighTarget-1)*2, skipHighTarget*2))

	movImm16(7, 0x9001) // CH0_FREQ_HIGH
	movMem(7, 0)
	movImm16(0, 0)

	// Set duration to 60 frames (1 second at 60 FPS)
	movImm16(7, 0x9004) // CH0_DURATION_LOW
	movImm16(0, 60)      // R0 = 60 (low byte)
	movMem(7, 0)
	movImm16(0, 0)

	movImm16(7, 0x9005) // CH0_DURATION_HIGH
	movImm16(0, 0)       // R0 = 0 (high byte)
	movMem(7, 0)
	movImm16(0, 0)

	// Set duration mode: stop when done (mode 0)
	movImm16(7, 0x9006) // CH0_DURATION_MODE
	movImm16(0, 0)       // R0 = 0 (stop mode)
	movMem(7, 0)
	movImm16(0, 0)

	// Re-enable channel to start the note
	movImm16(7, 0x9003) // CH0_CONTROL
	movImm16(0, 0x01)    // R0 = enable, sine wave
	movMem(7, 0)
	movImm16(0, 0)

	// ============================================
	// UPDATE CHORD EVERY 8 NOTES (when arpeggio wraps)
	// ============================================
	
	// Check if note index wrapped to 0
	cmpImm(4, 0)         // Check if note index wrapped to 0
	skipChordPC := len(code)
	blt(0) // Placeholder - will patch

	// Update chord
	addImm(5, 1)
	andImm(5, 0x03) // Keep in range 0-3
	cmpImm(5, 3)
	skipResetChordPC := len(code)
	blt(0) // Placeholder - will patch
	movImm16(5, 0) // Reset to 0 if it was 3
	skipResetChordTarget := len(code)
	code[skipResetChordPC+1] = uint16(calcOffset((skipResetChordPC+2)*2, skipResetChordTarget*2))

	// Set chord frequencies based on R5 (0=C, 1=F, 2=G)
	// Chord 0: C major - C(262), E(330)
	// Chord 1: F major - F(349), A(440)
	// Chord 2: G major - G(392), B(494)

	// Channel 1 (root note)
	cmpImm(5, 0)
	skipChord0PC := len(code)
	blt(0) // Placeholder - will patch
	// C = 262 Hz
	movImm16(0, 0x06) // Low byte
	movImm16(1, 0x01) // High byte (stored in R1)
	skipChord0Target := len(code)
	jmp(calcOffset(len(code)*2, skipChord0Target*2))
	skipChord0Target2 := len(code)
	code[skipChord0PC+1] = uint16(calcOffset((skipChord0PC+2)*2, skipChord0Target2*2))

	cmpImm(5, 1)
	skipChord1PC := len(code)
	blt(0) // Placeholder - will patch
	// F = 349 Hz
	movImm16(0, 0x5D) // Low byte
	movImm16(1, 0x01) // High byte
	skipChord1Target := len(code)
	jmp(calcOffset(len(code)*2, skipChord1Target*2))
	skipChord1Target2 := len(code)
	code[skipChord1PC+1] = uint16(calcOffset((skipChord1PC+2)*2, skipChord1Target2*2))

	// Chord 2: G = 392 Hz
	movImm16(0, 0x88) // Low byte
	movImm16(1, 0x01) // High byte

	skipChord1Target2 = len(code)
	code[skipChord1Target-1] = uint16(calcOffset((skipChord1Target-1)*2, skipChord1Target2*2))
	skipChord0Target2 = len(code)
	code[skipChord0Target-1] = uint16(calcOffset((skipChord0Target-1)*2, skipChord0Target2*2))

	// Write Channel 1 frequency
	movImm16(7, 0x9008) // CH1_FREQ_LOW (0x9000 + 1*8 + 0)
	movMem(7, 0)
	movImm16(0, 0)

	movImm16(7, 0x9009) // CH1_FREQ_HIGH (0x9000 + 1*8 + 1)
	movReg(0, 1)         // R0 = high byte from R1
	movMem(7, 0)
	movImm16(0, 0)
	movImm16(1, 0) // Restore R1

	// Channel 2 (third note)
	cmpImm(5, 0)
	skipChord0ThirdPC := len(code)
	blt(0) // Placeholder - will patch
	// E = 330 Hz
	movImm16(0, 0x4A) // Low byte
	movImm16(1, 0x01) // High byte
	skipChord0ThirdTarget := len(code)
	jmp(calcOffset(len(code)*2, skipChord0ThirdTarget*2))
	skipChord0ThirdTarget2 := len(code)
	code[skipChord0ThirdPC+1] = uint16(calcOffset((skipChord0ThirdPC+2)*2, skipChord0ThirdTarget2*2))

	cmpImm(5, 1)
	skipChord1ThirdPC := len(code)
	blt(0) // Placeholder - will patch
	// A = 440 Hz
	movImm16(0, 0xB8) // Low byte
	movImm16(1, 0x01) // High byte
	skipChord1ThirdTarget := len(code)
	jmp(calcOffset(len(code)*2, skipChord1ThirdTarget*2))
	skipChord1ThirdTarget2 := len(code)
	code[skipChord1ThirdPC+1] = uint16(calcOffset((skipChord1ThirdPC+2)*2, skipChord1ThirdTarget2*2))

	// Chord 2: B = 494 Hz
	movImm16(0, 0xEE) // Low byte
	movImm16(1, 0x01) // High byte

	skipChord1ThirdTarget2 = len(code)
	code[skipChord1ThirdTarget-1] = uint16(calcOffset((skipChord1ThirdTarget-1)*2, skipChord1ThirdTarget2*2))
	skipChord0ThirdTarget2 = len(code)
	code[skipChord0ThirdTarget-1] = uint16(calcOffset((skipChord0ThirdTarget-1)*2, skipChord0ThirdTarget2*2))

	// Write Channel 2 frequency
	movImm16(7, 0x9010) // CH2_FREQ_LOW (0x9000 + 2*8 + 0)
	movMem(7, 0)
	movImm16(0, 0)

	movImm16(7, 0x9011) // CH2_FREQ_HIGH (0x9000 + 2*8 + 1)
	movReg(0, 1)         // R0 = high byte from R1
	movMem(7, 0)
	movImm16(0, 0)
	movImm16(1, 0) // Restore R1

	// Patch branch offsets
	skipChordTarget := len(code)
	code[skipChordPC+1] = uint16(calcOffset((skipChordPC+2)*2, skipChordTarget*2))

	// Patch the BEQ offset for channel 0 completion check
	// If channel 0 didn't finish (R6 == 0), skip the note update code
	// BEQ instruction is at skipArpPC, offset word is at skipArpPC+1
	// PC after BEQ instruction + offset = (skipArpPC + 2) * 2
	// Target PC = skipArpTarget * 2
	skipArpTarget := len(code)
	code[skipArpPC+1] = uint16(calcOffset((skipArpPC+2)*2, skipArpTarget*2))

	// Loop forever
	jmp(calcOffset(len(code)*2, mainLoop*2))

	// ============================================
	// BUILD ROM FILE
	// ============================================
	
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

	fmt.Printf("Audio test ROM created: %s\n", outputPath)
	fmt.Printf("ROM size: %d bytes (%d instructions)\n", len(romData), len(code))
	fmt.Printf("\nThis ROM uses the NEW APU architecture:\n")
	fmt.Printf("  - Channel 0: Arpeggio (sine wave) with 60-frame duration per note\n")
	fmt.Printf("  - Channel 1: Chord root (square wave) - cycles C, F, G\n")
	fmt.Printf("  - Channel 2: Chord third (square wave) - cycles E, A, B\n")
	fmt.Printf("\nAPU Register Layout (NEW - 8 bytes per channel):\n")
	fmt.Printf("  +0: FREQ_LOW, +1: FREQ_HIGH, +2: VOLUME, +3: CONTROL\n")
	fmt.Printf("  +4: DURATION_LOW, +5: DURATION_HIGH, +6: DURATION_MODE\n")
	fmt.Printf("\nCompletion Status Register: 0x9021 (bits 0-3 = channels 0-3)\n")
	fmt.Printf("\nRun with: ./nitro-core-dx -rom %s -scale 3\n", outputPath)
}
