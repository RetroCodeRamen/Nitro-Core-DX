package corelx

import (
	"fmt"

	"nitro-core-dx/internal/rom"
)

// Part of generateBuiltinCall. See the register contract on that function.
func (cg *CodeGenerator) builtinApu(name string, args []Expr, destReg uint8) error {
	switch name {
	case "apu.enable":
		// apu.enable() - Enable APU master volume
		// Write 0xFF to MASTER_VOLUME (0x9020)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x9020
		cg.builder.AddImmediate(0x9020)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xFF
		cg.builder.AddImmediate(0xFF)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write master volume)
		return nil

	case "apu.set_channel_wave":
		// apu.set_channel_wave(ch: u8, wave: u8)
		// Args: R0 = channel (0-3), R1 = waveform (0-3)
		// Write to CONTROL register (offset +3) with bits [1:2] = waveform
		// Channel base: CH0=0x9000, CH1=0x9008, CH2=0x9010, CH3=0x9018
		// CONTROL = channel_base + 3

		// Calculate channel base address: 0x9000 + (ch * 8)
		// ch * 8 = ch << 3
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0)) // MOV R4, R0 (save channel)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #3
		cg.builder.AddImmediate(3)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 4, 5)) // SHL R4, R5 -> R4 = ch << 3 = ch * 8
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x9000
		cg.builder.AddImmediate(0x9000)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = 0x9000 + (ch * 8)

		// Add offset 3 for CONTROL register
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #3
		cg.builder.AddImmediate(3)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = channel_base + 3

		// Prepare waveform value: shift to bits [1:2]
		// Waveform is in R1, need to shift left by 1 bit
		cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 1)) // MOV R5, R1 (waveform)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 5, 6)) // SHL R5, R6 -> R5 = wave << 1

		// Read current CONTROL value, OR with waveform bits, write back
		// For simplicity, just write waveform bits (assumes enable bit will be set separately)
		// In practice, we'd read, mask, OR, write - but for now just write waveform
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write CONTROL)
		return nil

	case "apu.set_channel_freq":
		// apu.set_channel_freq(ch: u8, freq: u16)
		// Args: R0 = channel (0-3), R1 = frequency (16-bit)
		// Write low byte to FREQ_LOW (offset +0), then high byte to FREQ_HIGH (offset +1)
		// Writing high byte triggers phase reset

		// Calculate channel base address: 0x9000 + (ch * 8)
		// ch * 8 = ch << 3
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0)) // MOV R4, R0 (save channel)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #3
		cg.builder.AddImmediate(3)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 4, 5)) // SHL R4, R5 -> R4 = ch << 3 = ch * 8
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x9000
		cg.builder.AddImmediate(0x9000)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = 0x9000 + (ch * 8)

		// Save frequency value
		cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 1)) // MOV R5, R1 (frequency)

		// Write low byte to FREQ_LOW (offset +0)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5 (copy freq)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0xFF
		cg.builder.AddImmediate(0xFF)
		cg.builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7 -> R6 = freq & 0xFF (low byte)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 6)) // MOV [R4], R6 (write FREQ_LOW)

		// Write high byte to FREQ_HIGH (offset +1)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 6)) // ADD R4, R6 -> R4 = channel_base + 1
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5 (copy freq)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #8
		cg.builder.AddImmediate(8)
		cg.builder.AddInstruction(rom.EncodeSHR(0, 6, 7)) // SHR R6, R7 -> R6 = freq >> 8 (high byte)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 6)) // MOV [R4], R6 (write FREQ_HIGH, triggers phase reset)
		return nil

	case "apu.set_channel_volume":
		// apu.set_channel_volume(ch: u8, vol: u8)
		// Args: R0 = channel (0-3), R1 = volume (0-255)
		// Write to VOLUME register (offset +2)

		// Calculate channel base address: 0x9000 + (ch * 8)
		// ch * 8 = ch << 3
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0)) // MOV R4, R0 (save channel)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #3
		cg.builder.AddImmediate(3)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 4, 5)) // SHL R4, R5 -> R4 = ch << 3 = ch * 8
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x9000
		cg.builder.AddImmediate(0x9000)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = 0x9000 + (ch * 8)

		// Add offset 2 for VOLUME register
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #2
		cg.builder.AddImmediate(2)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = channel_base + 2

		// Write volume (R1) to VOLUME register
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 1)) // MOV [R4], R1 (write VOLUME)
		return nil

	case "apu.note_on":
		// apu.note_on(ch: u8)
		// Args: R0 = channel (0-3)
		// Set CONTROL register (offset +3) bit 0 to 1 (enable)

		// Calculate channel base address: 0x9000 + (ch * 8)
		// ch * 8 = ch << 3
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0)) // MOV R4, R0 (save channel)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #3
		cg.builder.AddImmediate(3)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 4, 5)) // SHL R4, R5 -> R4 = ch << 3 = ch * 8
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x9000
		cg.builder.AddImmediate(0x9000)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = 0x9000 + (ch * 8)

		// Add offset 3 for CONTROL register
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #3
		cg.builder.AddImmediate(3)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = channel_base + 3

		// Read current CONTROL value, OR with 0x01 (enable bit)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read current CONTROL)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0x01
		cg.builder.AddImmediate(0x01)
		cg.builder.AddInstruction(rom.EncodeOR(0, 5, 6))  // OR R5, R6 -> R5 = CONTROL | 0x01
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write CONTROL with enable bit)
		return nil

	case "apu.note_off":
		// apu.note_off(ch: u8)
		// Args: R0 = channel (0-3)
		// Clear CONTROL register (offset +3) bit 0 (disable)

		// Calculate channel base address: 0x9000 + (ch * 8)
		// ch * 8 = ch << 3
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0)) // MOV R4, R0 (save channel)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #3
		cg.builder.AddImmediate(3)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 4, 5)) // SHL R4, R5 -> R4 = ch << 3 = ch * 8
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x9000
		cg.builder.AddImmediate(0x9000)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = 0x9000 + (ch * 8)

		// Add offset 3 for CONTROL register
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #3
		cg.builder.AddImmediate(3)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = channel_base + 3

		// Read current CONTROL value, AND with 0xFE (clear enable bit)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read current CONTROL)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0xFE
		cg.builder.AddImmediate(0xFE)
		cg.builder.AddInstruction(rom.EncodeAND(0, 5, 6)) // AND R5, R6 -> R5 = CONTROL & 0xFE (clear bit 0)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write CONTROL without enable bit)
		return nil

	default:
		return fmt.Errorf("%w: %s", errUnknownBuiltin, name)
	}
}
