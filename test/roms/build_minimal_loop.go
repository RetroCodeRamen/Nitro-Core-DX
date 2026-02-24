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
		fmt.Println("Usage: go run build_minimal_loop.go <output.rom>")
		os.Exit(1)
	}

	outputPath := os.Args[1]
	builder := rom.NewROMBuilder()

	// Minimal ROM - just loops forever doing NOPs
	// This tests if basic execution works
	entryBank := uint8(1)
	entryOffset := uint16(0x8000)

	fmt.Println("Building minimal loop ROM...")

	// Set up backdrop color (black)
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

	// Enable BG0
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8008
	builder.AddImmediate(0x8008)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Simple loop - just NOP forever
	loopStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(0x0000) // NOP
	builder.AddInstruction(rom.EncodeJMP()) // JMP loop_start
	currentPC := uint16(builder.GetCodeLength() * 2)
	offset := rom.CalculateBranchOffset(currentPC, loopStart)
	builder.AddImmediate(uint16(offset))

	// Build ROM
	if err := builder.BuildROM(entryBank, entryOffset, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error building ROM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Minimal loop ROM built: %s\n", outputPath)
	fmt.Println("This ROM just loops forever doing NOPs.")
	fmt.Println("If CPU cycles are still 0, there's an issue with ROM loading or execution.")
}
