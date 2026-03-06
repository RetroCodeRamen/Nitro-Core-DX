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
		fmt.Println("Usage: go run build_checkerboard_sprite.go <output.rom>")
		os.Exit(1)
	}

	outputPath := os.Args[1]
	builder := rom.NewROMBuilder()

	entryBank := uint8(1)
	entryOffset := uint16(0x8000)

	fmt.Println("Building checkerboard sprite test ROM (white/blue 8x8)...")

	// ============================================
	// STEP 1: CGRAM - palette 1 color 1 (white), color 2 (blue)
	// ============================================
	fmt.Println("  [1] Setting up CGRAM palette 1 (white + blue)...")

	// CGRAM_ADDR = 0x8012, CGRAM_DATA = 0x8013
	// Palette 1, color 1 = 0x11 -> white 0x7FFF
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x11)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0xFF)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x7F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	// Palette 1, color 2 = 0x12 -> blue 0x001F
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x12)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x1F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	// ============================================
	// STEP 2: VRAM tile 0 = 8x8 checkerboard (4bpp: 0x12 = pixel 1,2; 0x21 = pixel 2,1)
	// ============================================
	fmt.Println("  [2] Setting up VRAM tile 0 (checkerboard)...")

	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x800E)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x800F)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	// R4 = VRAM_DATA (0x8010) for all following writes
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x8010)

	// 32 bytes: rows 0,2,4,6 = 0x12 0x12 0x12 0x12; rows 1,3,5,7 = 0x21 0x21 0x21 0x21
	checkerboard := []uint16{
		0x12, 0x12, 0x12, 0x12, 0x21, 0x21, 0x21, 0x21,
		0x12, 0x12, 0x12, 0x12, 0x21, 0x21, 0x21, 0x21,
		0x12, 0x12, 0x12, 0x12, 0x21, 0x21, 0x21, 0x21,
		0x12, 0x12, 0x12, 0x12, 0x21, 0x21, 0x21, 0x21,
	}
	for _, b := range checkerboard {
		builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
		builder.AddImmediate(b)
		builder.AddInstruction(rom.EncodeMOV(3, 4, 5))
	}

	// ============================================
	// STEP 3: Disable BG0
	// ============================================
	fmt.Println("  [3] Disabling BG0...")
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x8008)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	// ============================================
	// STEP 4 & 5: Main loop - wait VBlank, write sprite 0 OAM
	// ============================================
	fmt.Println("  [4] Main loop + OAM sprite 0 at (160, 100)...")
	mainLoopStart := uint16(builder.GetCodeLength() * 2)

	waitVBlankStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x803E)
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4))
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 5, 7))
	builder.AddInstruction(rom.EncodeBEQ())
	currentPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(uint16(rom.CalculateBranchOffset(currentPC, waitVBlankStart)))

	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x8014)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	builder.AddImmediate(0x8015)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(160)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(100)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	builder.AddInstruction(rom.EncodeJMP())
	currentPC = uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(uint16(rom.CalculateBranchOffset(currentPC, mainLoopStart)))

	if err := builder.BuildROM(entryBank, entryOffset, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error building ROM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Checkerboard sprite ROM built: %s\n", outputPath)
	fmt.Println("Expected: 8×8 sprite at (160, 100) with alternating white and blue pixels (checkerboard).")
}
