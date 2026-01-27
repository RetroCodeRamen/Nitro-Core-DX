package main

import (
	"nitro-core-dx/internal/rom"
)

func main() {
	builder := rom.NewROMBuilder()

	// Simple test ROM that just does basic operations
	// Entry point will be at bank 1, offset 0x8000

	// MOV R0, #0x1234 - Load immediate value
	builder.AddInstruction(rom.EncodeMOV(1, 0, 0)) // MOV R0, #imm
	builder.AddImmediate(0x1234)

	// MOV R1, #0x5678 - Load another immediate value
	builder.AddInstruction(rom.EncodeMOV(1, 1, 0)) // MOV R1, #imm
	builder.AddImmediate(0x5678)

	// ADD R0, R1 - Add R1 to R0
	builder.AddInstruction(rom.EncodeADD(0, 0, 1)) // ADD R0, R1

	// MOV [0x8000], R0 - Write result to PPU register (just to do something)
	builder.AddInstruction(rom.EncodeMOV(3, 0, 0)) // MOV [R0], R0
	// But wait, we need the address in a register first
	// MOV R2, #0x8000
	builder.AddInstruction(rom.EncodeMOV(1, 2, 0)) // MOV R2, #imm
	builder.AddImmediate(0x8000)
	// MOV [R2], R0 - Write R0 to address in R2
	builder.AddInstruction(rom.EncodeMOV(3, 2, 0)) // MOV [R2], R0

	// Infinite loop with NOPs
	// JMP to self (relative jump of 0)
	builder.AddInstruction(rom.EncodeJMP())
	builder.AddImmediate(0xFFFE) // Jump back 2 bytes (to the JMP instruction itself)

	// Build ROM with entry point at bank 1, offset 0x8000
	if err := builder.BuildROM(1, 0x8000, "test/roms/simple_test.rom"); err != nil {
		panic(err)
	}
}
