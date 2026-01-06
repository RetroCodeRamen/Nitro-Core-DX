package main

import (
	"fmt"
	"os"

	"nitro-core-dx/internal/rom"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: rombuilder <output.rom>")
		os.Exit(1)
	}

	outputPath := os.Args[1]
	builder := rom.NewROMBuilder()

	// Test ROM: Movable block with color changes and sound
	// Entry point: Bank 1, Offset 0x8000
	entryBank := uint8(1)
	entryOffset := uint16(0x8000)

	// We'll build the ROM programmatically
	// This is a simplified version - in a real assembler, we'd parse assembly

	// Initialize: Set up block position (R0 = X, R1 = Y)
	// MOV R0, #160  (center X)
	builder.AddInstruction(rom.EncodeMOV(1, 0, 0)) // MOV R0, #imm
	builder.AddImmediate(160)

	// MOV R1, #100  (center Y)
	builder.AddInstruction(rom.EncodeMOV(1, 1, 0)) // MOV R1, #imm
	builder.AddImmediate(100)

	// MOV R2, #1    (block color palette index)
	builder.AddInstruction(rom.EncodeMOV(1, 2, 0))
	builder.AddImmediate(1)

	// MOV R3, #0    (background color palette index)
	builder.AddInstruction(rom.EncodeMOV(1, 3, 0))
	builder.AddImmediate(0)

	// Enable BG0
	// MOV R4, #0x8008  (BG0_CONTROL address)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x8008)

	// MOV R5, #0x01  (enable BG0, 8x8 tiles)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x01)

	// MOV [R4], R5  (write control)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	// Set up palette colors
	// Set CGRAM address to 0 (background color)
	// MOV R4, #0x8012  (CGRAM_ADDR)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x8012)

	// MOV R5, #0x00  (address 0)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x00)

	// MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	// Write background color (blue: RGB555 = 0x001F)
	// MOV R4, #0x8013  (CGRAM_DATA)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x8013)

	// MOV R5, #0x1F  (low byte: blue)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x1F)

	// MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	// MOV R5, #0x00  (high byte)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x00)

	// MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	// Set block color (palette 1, color 0 = white)
	// MOV R4, #0x8012  (CGRAM_ADDR)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x8012)

	// MOV R5, #0x10  (palette 1, color 0 = index 16)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x10)

	// MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	// Write white color (RGB555 = 0x7FFF)
	// MOV R4, #0x8013  (CGRAM_DATA)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x8013)

	// MOV R5, #0xFF  (low byte)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0xFF)

	// MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	// MOV R5, #0x7F  (high byte)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x7F)

	// MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	// Initialize audio: Set up channel 0 for scale
	// We'll set frequency to C4 (261.63 Hz) initially
	// MOV R4, #0x9000  (CH0_FREQ_LOW)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x9000)

	// MOV R5, #0x05  (261 Hz low byte = 0x0105)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x05)

	// MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	// MOV R4, #0x9001  (CH0_FREQ_HIGH)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x9001)

	// MOV R5, #0x01  (261 Hz high byte)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x01)

	// MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	// MOV R4, #0x9002  (CH0_VOLUME)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x9002)

	// MOV R5, #0x80  (volume = 128)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x80)

	// MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	// MOV R4, #0x9003  (CH0_CONTROL)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x9003)

	// MOV R5, #0x01  (enable, sine wave)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x01)

	// MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	// Main loop
	// Note: This is a simplified version. A full implementation would need:
	// - Proper tile rendering
	// - Sprite rendering for the block
	// - Input reading and movement
	// - Color changing logic
	// - Scale playing logic with timing

	// For now, we'll create a minimal loop that at least runs
	// JMP main_loop (relative offset will be calculated)
	mainLoopStart := uint16(len(builder.code) * 2) // Current position in bytes

	// Simple delay loop
	// MOV R6, #0  (counter)
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
	builder.AddImmediate(0)

	// delay_loop:
	delayLoopStart := uint16(len(builder.code) * 2)

	// ADD R6, #1
	builder.AddInstruction(rom.EncodeADD(1, 6, 0))
	builder.AddImmediate(1)

	// CMP R6, #0x1000  (compare with 4096)
	builder.AddInstruction(rom.EncodeCMP(1, 6, 0))
	builder.AddImmediate(0x1000)

	// BNE delay_loop
	builder.AddInstruction(rom.EncodeBNE())
	currentPC := uint16(len(builder.code)*2 + 2) // PC after instruction + offset
	offset := rom.CalculateBranchOffset(currentPC, delayLoopStart)
	builder.AddImmediate(uint16(offset))

	// JMP main_loop
	builder.AddInstruction(rom.EncodeJMP())
	currentPC = uint16(len(builder.code)*2 + 2)
	offset = rom.CalculateBranchOffset(currentPC, mainLoopStart)
	builder.AddImmediate(uint16(offset))

	// Build ROM
	if err := builder.BuildROM(entryBank, entryOffset, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error building ROM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ROM built successfully: %s\n", outputPath)
	fmt.Printf("ROM size: %d bytes\n", len(builder.code)*2)
}



