//go:build testrom_tools
// +build testrom_tools

package main

import (
	"fmt"
	"os"

	"nitro-core-dx/internal/rom"
)

// Build a ROM with a moving sprite that has a colorful pattern to help understand tile structure
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <output.rom>\n", os.Args[0])
		os.Exit(1)
	}

	outputPath := os.Args[1]
	builder := rom.NewROMBuilder()

	entryBank := uint8(1)
	entryOffset := uint16(0x8000)

	// Initialize CGRAM with multiple colors for palette 1
	// Palette 1 will have:
	//   Color 0 = transparent (black) - not used for sprites
	//   Color 1 = Red
	//   Color 2 = Green
	//   Color 3 = Blue
	//   Color 4 = Yellow
	//   Color 5 = Cyan
	//   Color 6 = Magenta
	//   Color 7 = White

	// Set CGRAM address to palette 1, color 1 (Red)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012 (CGRAM_ADDR)
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x11 (palette 1, color 1)
	builder.AddImmediate(0x11)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write Red: RGB555 = 0x7C00 (R=31, G=0, B=0)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013 (CGRAM_DATA)
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (low byte: 0x00)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7C (high byte: 0x7C)
	builder.AddImmediate(0x7C)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Set CGRAM address to palette 1, color 2 (Green)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012 (CGRAM_ADDR)
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x12 (palette 1, color 2)
	builder.AddImmediate(0x12)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write Green: RGB555 = 0x03E0 (R=0, G=31, B=0)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013 (CGRAM_DATA)
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xE0 (low byte: 0xE0)
	builder.AddImmediate(0xE0)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x03 (high byte: 0x03)
	builder.AddImmediate(0x03)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Set CGRAM address to palette 1, color 3 (Blue)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012 (CGRAM_ADDR)
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x13 (palette 1, color 3)
	builder.AddImmediate(0x13)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write Blue: RGB555 = 0x001F (R=0, G=0, B=31)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013 (CGRAM_DATA)
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x1F (low byte: 0x1F)
	builder.AddImmediate(0x1F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (high byte: 0x00)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Set CGRAM address to palette 1, color 4 (Yellow = Red + Green)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012 (CGRAM_ADDR)
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x14 (palette 1, color 4)
	builder.AddImmediate(0x14)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write Yellow: RGB555 = 0x7FE0 (R=31, G=31, B=0)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013 (CGRAM_DATA)
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xE0 (low byte: 0xE0)
	builder.AddImmediate(0xE0)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7F (high byte: 0x7F)
	builder.AddImmediate(0x7F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Initialize VRAM: create a 16x16 tile with a colorful pattern
	// Pattern: 4 colored 8x8 blocks
	// Top-left 8x8: Red (color 1)
	// Top-right 8x8: Green (color 2)
	// Bottom-left 8x8: Blue (color 3)
	// Bottom-right 8x8: Yellow (color 4)

	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E (VRAM_ADDR_L)
	builder.AddImmediate(0x800E)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F (VRAM_ADDR_H)
	builder.AddImmediate(0x800F)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Top-left 8x8 block (32 bytes of 0x11 = Red)
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #32
	builder.AddImmediate(32)
	initVRAMRedStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010 (VRAM_DATA)
	builder.AddImmediate(0x8010)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x11 (two red pixels)
	builder.AddImmediate(0x11)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeSUB(1, 6, 0)) // SUB R6, #1
	builder.AddImmediate(1)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())        // BNE initVRAMRedStart
	currentPC := uint16(builder.GetCodeLength() * 2)
	offset := rom.CalculateBranchOffset(currentPC, initVRAMRedStart)
	builder.AddImmediate(uint16(offset))

	// Top-right 8x8 block (32 bytes of 0x22 = Green)
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #32
	builder.AddImmediate(32)
	initVRAMGreenStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010 (VRAM_DATA)
	builder.AddImmediate(0x8010)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x22 (two green pixels)
	builder.AddImmediate(0x22)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeSUB(1, 6, 0)) // SUB R6, #1
	builder.AddImmediate(1)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())        // BNE initVRAMGreenStart
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, initVRAMGreenStart)
	builder.AddImmediate(uint16(offset))

	// Bottom-left 8x8 block (32 bytes of 0x33 = Blue)
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #32
	builder.AddImmediate(32)
	initVRAMBlueStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010 (VRAM_DATA)
	builder.AddImmediate(0x8010)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x33 (two blue pixels)
	builder.AddImmediate(0x33)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeSUB(1, 6, 0)) // SUB R6, #1
	builder.AddImmediate(1)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())        // BNE initVRAMBlueStart
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, initVRAMBlueStart)
	builder.AddImmediate(uint16(offset))

	// Bottom-right 8x8 block (32 bytes of 0x44 = Yellow)
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #32
	builder.AddImmediate(32)
	initVRAMYellowStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010 (VRAM_DATA)
	builder.AddImmediate(0x8010)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x44 (two yellow pixels)
	builder.AddImmediate(0x44)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeSUB(1, 6, 0)) // SUB R6, #1
	builder.AddImmediate(1)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())        // BNE initVRAMYellowStart
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, initVRAMYellowStart)
	builder.AddImmediate(uint16(offset))

	// Enable BG0 and set up a solid color background
	// Use tile 0 for background (tile index is 8-bit, so max 255)
	// Tile 0 is already initialized with sprite data, but we'll overwrite it
	// Actually, let's use a different tile - tile 1 to avoid conflicts
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E (VRAM_ADDR_L)
	builder.AddImmediate(0x800E)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x20 (tile 1 = 0x0020, low byte: 32 bytes per 8x8 tile)
	builder.AddImmediate(0x20)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F (VRAM_ADDR_H)
	builder.AddImmediate(0x800F)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (tile 1 = 0x0020, high byte)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Fill tile 1 with color index 1 (32 bytes for 8x8 tile)
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #32
	builder.AddImmediate(32)
	initBGTileStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010 (VRAM_DATA)
	builder.AddImmediate(0x8010)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x11 (two pixels of color 1)
	builder.AddImmediate(0x11)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeSUB(1, 6, 0)) // SUB R6, #1
	builder.AddImmediate(1)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())        // BNE init_bg_tile_start
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, initBGTileStart)
	builder.AddImmediate(uint16(offset))

	// Set up tilemap (fill with tile 1)
	// Screen is 320x200, tiles are 8x8, so we need 40x25 = 1000 tiles
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E (VRAM_ADDR_L)
	builder.AddImmediate(0x800E)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (tilemap base 0x4000, low byte)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F (VRAM_ADDR_H)
	builder.AddImmediate(0x800F)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x40 (tilemap base 0x4000, high byte)
	builder.AddImmediate(0x40)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Fill tilemap: 40 tiles wide x 25 tiles tall = 1000 tiles = 2000 bytes
	// Tilemap entry: byte 0 = tile index (8-bit), byte 1 = attributes (palette in low 4 bits)
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #2000 (1000 tiles * 2 bytes)
	builder.AddImmediate(2000)
	initTilemapStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010 (VRAM_DATA)
	builder.AddImmediate(0x8010)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01 (tile index = 1)
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (attributes: palette 0)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeSUB(1, 6, 0)) // SUB R6, #2 (we wrote 2 bytes)
	builder.AddImmediate(2)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())        // BNE init_tilemap_start
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, initTilemapStart)
	builder.AddImmediate(uint16(offset))

	// Set background color in CGRAM palette 0, color 1 (start with blue)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012 (CGRAM_ADDR)
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01 (palette 0, color 1)
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013 (CGRAM_DATA)
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x1F (Blue: RGB555 = 0x001F, low byte)
	builder.AddImmediate(0x1F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (high byte)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Enable BG0
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8008 (BG0_CONTROL)
	builder.AddImmediate(0x8008)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01 (enable)
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Initialize R1 (background color counter) to 0
	builder.AddInstruction(rom.EncodeMOV(1, 1, 0)) // MOV R1, #0x00
	builder.AddImmediate(0x00)

	// Wait for VBlank before initializing sprite (OAM writes need VBlank)
	initVBlankWaitStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x803E (VBLANK_FLAG)
	builder.AddImmediate(0x803E)
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read VBlank flag)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 5, 7)) // CMP R5, R7
	builder.AddInstruction(rom.EncodeBEQ())        // BEQ init_vblank_wait_start
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, initVBlankWaitStart)
	builder.AddImmediate(uint16(offset))

	// Initialize sprite 0: position (50, 50), tile 0, palette 1, enabled, 16x16
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014 (OAM_ADDR)
	builder.AddImmediate(0x8014)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (sprite 0)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write sprite data: X low = 50
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015 (OAM_DATA)
	builder.AddImmediate(0x8015)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #50
	builder.AddImmediate(50)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// X high = 0
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Y = 50 (more visible, not too close to edge)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #50
	builder.AddImmediate(50)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Tile = 0
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Attributes: palette 1, no flip
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01 (palette 1)
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Control: enabled, 16x16
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x03 (enabled=1, 16x16=1)
	builder.AddImmediate(0x03)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Main loop: wait for VBlank, update sprite position, check boundaries
	mainLoopStart := uint16(builder.GetCodeLength() * 2)

	// Initialize R1 (background color counter) to 0 if not already set
	// R1 will be used to track background color cycles (0-3)
	// We only initialize it once at the start, then it persists across frames
	// Check if this is the first iteration by checking if R1 is uninitialized
	// Actually, simpler: just initialize R1 to 0 at the start of main loop
	// But we don't want to reset it every frame, so we'll only initialize if it's 0
	// Actually, let's just initialize it once before the main loop starts
	// Wait, R1 might already be set from a previous wrap. Let's not reset it.
	// Actually, the issue is that R1 might not be initialized at all, causing undefined behavior.
	// Let's initialize R1 to 0 before the main loop, but only if it's the first time.
	// Simplest: Initialize R1 to 0 before entering main loop (before sprite init)
	// But we already did sprite init... so R1 should be initialized there.
	// Actually, let's just make sure R1 is initialized to 0 at the very start of the program.

	// Wait for VBlank
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x803E (VBLANK_FLAG)
	builder.AddImmediate(0x803E)
	waitVBlankStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read VBlank flag)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 5, 7)) // CMP R5, R7
	builder.AddInstruction(rom.EncodeBEQ())        // BEQ wait_vblank_start
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, waitVBlankStart)
	builder.AddImmediate(uint16(offset))

	// Read current sprite X position from OAM
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014 (OAM_ADDR)
	builder.AddImmediate(0x8014)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (sprite 0)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015 (OAM_DATA)
	builder.AddImmediate(0x8015)
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read X low)
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5 (save X low)

	// Increment X
	builder.AddInstruction(rom.EncodeADD(1, 5, 0)) // ADD R5, #1
	builder.AddImmediate(1)

	// Save incremented X to R6 before we overwrite R5
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5 (save incremented X)

	// Check if X >= 336 (fully off-screen on the right)
	// Sprite width is 16, so X=320 means right edge is at 336, which is off-screen
	// We want the sprite to be completely off-screen before wrapping
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #336 (320 + 16)
	builder.AddImmediate(336)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7 (compare saved X)
	builder.AddInstruction(rom.EncodeBLT())        // BLT no_wrap_x (if X < 336, continue)
	currentPC = uint16(builder.GetCodeLength() * 2)
	noWrapXOffsetWordIndex := builder.GetCodeLength() // Word index of placeholder
	builder.AddImmediate(0)                           // Placeholder

	// Wrap X: set to -16 (off-screen on the left, will come in from left)
	// X low = 0xF0 (240), X high = 0x01 (sign bit) = -16 in two's complement
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0xF0 (240 = -16 when sign-extended)
	builder.AddImmediate(0xF0)

	// TEMPORARILY DISABLED: Wrap block code (background color cycling and palette cycling)
	// This code is executing every frame instead of only when wrapping, causing sprite corruption
	// TODO: Fix wrap check logic so wrap block only executes when X >= 336
	// For now, skip all wrap block code and just wrap X position
	/*
		// Before palette cycling, save Y and Tile to preserve them (R7 will be used for palette cycling)
		// Read current Y and Tile from sprite 0 OAM
		builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014 (OAM_ADDR)
		builder.AddImmediate(0x8014)
		builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (sprite 0)
		builder.AddImmediate(0x00)
		builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (reset OAM_ADDR and byte index)
		builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015 (OAM_DATA)
		builder.AddImmediate(0x8015)
		// Skip X low and X high (bytes 0-1)
		builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (skip X low)
		builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (skip X high)
		// Read Y (byte 2) - save to a temporary location (we'll use R3's high byte or save to memory)
		// Actually, we can't easily save to memory, so let's use R3 temporarily and restore it
		// But R3 is used for Tile... let's save Y to R0 temporarily (R0 is used for comparisons but we can restore it)
		builder.AddInstruction(rom.EncodeMOV(2, 0, 4)) // MOV R0, [R4] (read Y into R0 temporarily, advance to byte 3)
		// Read Tile (byte 3) - save to R3 (this is fine, R3 is used for Tile)
		builder.AddInstruction(rom.EncodeMOV(2, 3, 4)) // MOV R3, [R4] (read Tile into R3, advance to byte 4)
		// Now R0 has Y, R3 has Tile - we'll restore them after palette cycling

		// Cycle background color and sprite palette on wrap
		// Use R3 as background color counter (0-3: Blue, Green, Red, Yellow)
		// First, check if R3 exists (we'll use R3 for this)
		// Load current background color index (stored in WRAM or use a register)
		// For simplicity, we'll use R1 as background color counter (0-3)
		// Increment R1 (wrap at 4)
		builder.AddInstruction(rom.EncodeADD(1, 1, 0)) // ADD R1, #1
		builder.AddImmediate(1)
		builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #4
		builder.AddImmediate(4)
		builder.AddInstruction(rom.EncodeCMP(0, 1, 7)) // CMP R1, R7
		builder.AddInstruction(rom.EncodeBLT())        // BLT no_reset_bg_color
		currentPC = uint16(builder.GetCodeLength() * 2)
		noResetBGColorOffsetWordIndex := builder.GetCodeLength()
		builder.AddImmediate(0)                        // Placeholder
		builder.AddInstruction(rom.EncodeMOV(1, 1, 0)) // MOV R1, #0 (reset to 0)
		builder.AddImmediate(0)
		noResetBGColorTargetPC := uint16(builder.GetCodeLength() * 2)
		offset = rom.CalculateBranchOffset(currentPC, noResetBGColorTargetPC)
		builder.SetImmediateAt(noResetBGColorOffsetWordIndex, uint16(offset))

		// Set background color based on R1 (0=Blue, 1=Green, 2=Red, 3=Yellow)
		// Set CGRAM palette 0, color 1
		builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012 (CGRAM_ADDR)
		builder.AddImmediate(0x8012)
		builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01 (palette 0, color 1)
		builder.AddImmediate(0x01)
		builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013 (CGRAM_DATA)
		builder.AddImmediate(0x8013)

		// Jump table based on R1 value (0-3) - use simple if/else chain
		builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
		builder.AddImmediate(0)
		builder.AddInstruction(rom.EncodeCMP(0, 1, 7)) // CMP R1, R7
		builder.AddInstruction(rom.EncodeBNE())        // BNE not_blue
		currentPC = uint16(builder.GetCodeLength() * 2)
		notBlueOffsetWordIndex := builder.GetCodeLength()
		builder.AddImmediate(0) // Placeholder
		// Blue: RGB555 = 0x001F
		builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x1F
		builder.AddImmediate(0x1F)
		builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
		builder.AddImmediate(0x00)
		builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		builder.AddInstruction(rom.EncodeJMP())        // JMP done_bg_color
		currentPC = uint16(builder.GetCodeLength() * 2)
		blueJMPOffsetWordIndex := builder.GetCodeLength()          // Word index of JMP offset
		doneBGColorTargetPC := uint16(builder.GetCodeLength() * 2) // Target will be calculated later
		builder.AddImmediate(0)                                    // Placeholder - will calculate later

		notBlueTargetPC := uint16(builder.GetCodeLength() * 2)
		offset = rom.CalculateBranchOffset(currentPC, notBlueTargetPC)
		builder.SetImmediateAt(notBlueOffsetWordIndex, uint16(offset))

		builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #1
		builder.AddImmediate(1)
		builder.AddInstruction(rom.EncodeCMP(0, 1, 7)) // CMP R1, R7
		builder.AddInstruction(rom.EncodeBNE())        // BNE not_green
		currentPC = uint16(builder.GetCodeLength() * 2)
		notGreenOffsetWordIndex := builder.GetCodeLength()
		builder.AddImmediate(0) // Placeholder
		// Green: RGB555 = 0x03E0
		builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xE0
		builder.AddImmediate(0xE0)
		builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x03
		builder.AddImmediate(0x03)
		builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		builder.AddInstruction(rom.EncodeJMP())        // JMP done_bg_color
		currentPC = uint16(builder.GetCodeLength() * 2)
		greenJMPOffsetWordIndex := builder.GetCodeLength() // Word index of JMP offset
		builder.AddImmediate(0)                            // Placeholder - will calculate later

		notGreenTargetPC := uint16(builder.GetCodeLength() * 2)
		offset = rom.CalculateBranchOffset(currentPC, notGreenTargetPC)
		builder.SetImmediateAt(notGreenOffsetWordIndex, uint16(offset))

		builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #2
		builder.AddImmediate(2)
		builder.AddInstruction(rom.EncodeCMP(0, 1, 7)) // CMP R1, R7
		builder.AddInstruction(rom.EncodeBNE())        // BNE not_red
		currentPC = uint16(builder.GetCodeLength() * 2)
		notRedOffsetWordIndex := builder.GetCodeLength()
		builder.AddImmediate(0) // Placeholder
		// Red: RGB555 = 0x7C00
		builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
		builder.AddImmediate(0x00)
		builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7C
		builder.AddImmediate(0x7C)
		builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		builder.AddInstruction(rom.EncodeJMP())        // JMP done_bg_color
		currentPC = uint16(builder.GetCodeLength() * 2)
		redJMPOffsetWordIndex := builder.GetCodeLength() // Word index of JMP offset
		builder.AddImmediate(0)                          // Placeholder - will calculate later

		notRedTargetPC := uint16(builder.GetCodeLength() * 2)
		offset = rom.CalculateBranchOffset(currentPC, notRedTargetPC)
		builder.SetImmediateAt(notRedOffsetWordIndex, uint16(offset))

		// Yellow: RGB555 = 0x7FE0
		builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xE0
		builder.AddImmediate(0xE0)
		builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7F
		builder.AddImmediate(0x7F)
		builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

		// done_bg_color: (all color cases converge here)
		doneBGColorTargetPC = uint16(builder.GetCodeLength() * 2)
		// Update all JMP offsets
		// Blue JMP
		blueJMPPC := uint16(blueJMPOffsetWordIndex * 2)
		offset = rom.CalculateBranchOffset(blueJMPPC, doneBGColorTargetPC)
		builder.SetImmediateAt(blueJMPOffsetWordIndex, uint16(offset))
		// Green JMP
		greenJMPPC := uint16(greenJMPOffsetWordIndex * 2)
		offset = rom.CalculateBranchOffset(greenJMPPC, doneBGColorTargetPC)
		builder.SetImmediateAt(greenJMPOffsetWordIndex, uint16(offset))
		// Red JMP
		redJMPPC := uint16(redJMPOffsetWordIndex * 2)
		offset = rom.CalculateBranchOffset(redJMPPC, doneBGColorTargetPC)
		builder.SetImmediateAt(redJMPOffsetWordIndex, uint16(offset))

		// Change sprite palette (cycle through palettes 1-4) - DISABLED FOR DEBUGGING
		// TEMPORARILY DISABLED: Palette cycling is causing flickering
		// The palette cycling code should only run when wrapping, but it seems to run every frame
		// TODO: Fix wrap detection logic
		// For now, keep palette fixed at 1 to stop flickering
		builder.AddInstruction(rom.EncodeMOV(1, 2, 0)) // MOV R2, #0x01 (keep palette fixed at 1)
		builder.AddImmediate(0x01)
		// Skip the palette cycling code - just set R2 to palette 1
		// Original palette cycling code commented out below:
		/*
			builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014 (OAM_ADDR)
			builder.AddImmediate(0x8014)
			builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (sprite 0)
			builder.AddImmediate(0x00)
			builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
			builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015 (OAM_DATA)
			builder.AddImmediate(0x8015)
			builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (skip X low)
			builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (skip X high)
			builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (skip Y)
			builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (skip Tile)
			builder.AddInstruction(rom.EncodeMOV(2, 2, 4)) // MOV R2, [R4] (read Attributes into R2)
			builder.AddInstruction(rom.EncodeADD(1, 2, 0)) // ADD R2, #1
			builder.AddImmediate(1)
			builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x0F
			builder.AddImmediate(0x0F)
			builder.AddInstruction(rom.EncodeAND(0, 2, 7)) // AND R2, R7 (mask to 4 bits, wrap at 16)
			builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x05
			builder.AddImmediate(0x05)
			builder.AddInstruction(rom.EncodeCMP(0, 2, 7)) // CMP R2, R7
			builder.AddInstruction(rom.EncodeBLT())        // BLT palette_ok
			currentPC = uint16(builder.GetCodeLength() * 2)
			paletteOKOffsetWordIndex := builder.GetCodeLength()
			builder.AddImmediate(0)                        // Placeholder
			builder.AddInstruction(rom.EncodeMOV(1, 2, 0)) // MOV R2, #0x01 (wrap to palette 1)
			builder.AddImmediate(0x01)
			paletteOKTargetPC := uint16(builder.GetCodeLength() * 2)
			offset = rom.CalculateBranchOffset(currentPC, paletteOKTargetPC)
			builder.SetImmediateAt(paletteOKOffsetWordIndex, uint16(offset))
			// Still need to write Attributes and Control byte to OAM
			builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014 (OAM_ADDR)
			builder.AddImmediate(0x8014)
			builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (sprite 0)
			builder.AddImmediate(0x00)
			builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (reset OAM_ADDR and byte index)
			builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015 (OAM_DATA)
			builder.AddImmediate(0x8015)
			// Skip X low, X high, Y, Tile (4 bytes) to get to Attributes (byte 4)
			builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (skip X low)
			builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (skip X high)
			builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (skip Y)
			builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (skip Tile)
			builder.AddInstruction(rom.EncodeMOV(0, 5, 2)) // MOV R5, R2 (copy palette to R5)
			builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write Attributes)
			// After writing Attributes, byte index increments to 5 (Control byte)
			// We need to write Control byte (0x03 = enabled, 16x16) to preserve sprite state
			builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x03 (Control: enabled, 16x16)
			builder.AddImmediate(0x03)
			builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write Control byte)
			// R2 still contains the new palette value - preserve it by saving to R1 temporarily
		// (R1 is background color counter, but we'll save it, use R1 for Attributes, then restore)
		// Actually, R1 is already being used. Let's save R2 to memory or use a different approach.
		// Actually, simplest: After palette cycling, the new Attributes is already written to OAM.
		// When we update X position later, if we wrapped, we'll read Attributes from OAM (which will have the new value).
		// So we don't need to save R2 - we can just read Attributes normally and it will be the new value.
		// But wait, that won't work if we read Attributes before palette cycling writes it.
		// Let's save R2 to R1 temporarily, but we need to preserve R1 (background color counter) first.
		// Actually, let's save background color counter to memory temporarily... but we don't have easy memory access.
		// Simplest: Save R2 (new Attributes) to R1, but mark that R1 now contains Attributes (not background counter).
		// We can check if R1 < 16 to see if it's Attributes (palette values are 0-15).
		// But R1 is background color counter (0-3), so R1 < 16 is always true... that won't work.
		// Let's use a different approach: After palette cycling, don't save anything.
		// When updating X position, if we wrapped, read Attributes from OAM (it will have the new value from palette cycling).
		// If we didn't wrap, read Attributes normally.
		// But we need to know if we wrapped... we can check if R0 has Y (if R0 != 0 and R0 != 50, it might be from wrap, but that's unreliable).
		// Actually, simplest fix: Don't save Attributes at all. After palette cycling writes Attributes to OAM,
		// when we later update X position, we'll read Attributes from OAM and it will be the new value.
		// But we need to make sure we read Attributes AFTER palette cycling, not before.
		// The code flow is: wrap -> palette cycling -> update X position. So when we update X position,
		// if we wrapped, Attributes was already updated by palette cycling. So we can just read Attributes normally.
		// But the issue is: if we didn't wrap, we read Attributes before updating X. If we wrapped, palette cycling already updated Attributes.
		// So the Attributes in OAM is always correct - we just need to read it.
		// So we don't need to save R2 at all! Just remove this line.
	*/

	// no_wrap_x:
	noWrapXTargetPC := uint16(builder.GetCodeLength() * 2)
	// Calculate offset from placeholder position to target
	// currentPC points to placeholder word, so PC after placeholder is currentPC + 2
	offset = rom.CalculateBranchOffset(currentPC, noWrapXTargetPC)
	// Update the placeholder offset
	builder.SetImmediateAt(noWrapXOffsetWordIndex, uint16(offset))

	// Before updating X position, read current Y, Tile, Attributes, Control to preserve them
	// Reset OAM_ADDR to sprite 0 and read all bytes we need to preserve
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014 (OAM_ADDR)
	builder.AddImmediate(0x8014)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (sprite 0)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (reset OAM_ADDR and byte index)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015 (OAM_DATA)
	builder.AddImmediate(0x8015)

	// Read X low (byte 0) - skip it, we'll write new value
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (skip X low, advance to byte 1)
	// Read X high (byte 1) - skip it, we'll write new value
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (skip X high, advance to byte 2)
	// Read Y (byte 2) - preserve this in R0
	builder.AddInstruction(rom.EncodeMOV(2, 0, 4)) // MOV R0, [R4] (read Y into R0, advance to byte 3)
	// Read Tile (byte 3) - preserve this in R3
	builder.AddInstruction(rom.EncodeMOV(2, 3, 4)) // MOV R3, [R4] (read Tile into R3, advance to byte 4)
	// Read Attributes (byte 4) - preserve this in R2
	// If we wrapped, palette cycling already wrote the new Attributes to OAM, so reading it will get the new value.
	builder.AddInstruction(rom.EncodeMOV(2, 2, 4)) // MOV R2, [R4] (read Attributes into R2, advance to byte 5)
	// Read Control (byte 5) - we'll preserve this as 0x03
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read Control, byte index wraps to 0, OAM_ADDR increments)

	// Now reset OAM_ADDR back to sprite 0 to write all 6 bytes
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014 (OAM_ADDR)
	builder.AddImmediate(0x8014)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (sprite 0)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (reset OAM_ADDR and byte index)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015 (OAM_DATA)
	builder.AddImmediate(0x8015)

	// Write X low byte (from R6)
	builder.AddInstruction(rom.EncodeMOV(0, 5, 6)) // MOV R5, R6 (restore incremented X)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write X low)

	// Write X high byte (sign bit for negative values)
	// X high byte is 0x00 for positive X (0-255), or 0x01 for negative X (sign bit)
	// Only set sign bit if R6 == 0xF0 (wrapped to -16)
	// IMPORTANT: R0 contains Y, R3 contains Tile, R2 contains Attributes - don't overwrite them!
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (default: no sign bit)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0xF0 (use R7 for comparison)
	builder.AddImmediate(0xF0)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7 (compare X with 0xF0)
	builder.AddInstruction(rom.EncodeBNE())        // BNE no_sign_bit (if X != 0xF0, no sign bit)
	currentPC = uint16(builder.GetCodeLength() * 2)
	noSignBitOffsetWordIndex := builder.GetCodeLength()
	builder.AddImmediate(0) // Placeholder

	// X == 0xF0 (wrapped to -16), set sign bit
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01 (set sign bit)
	builder.AddImmediate(0x01)

	// no_sign_bit: (X != 0xF0, no sign bit needed - R5 already has 0x00)
	noSignBitTargetPC := uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, noSignBitTargetPC)
	builder.SetImmediateAt(noSignBitOffsetWordIndex, uint16(offset))

	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write X high)

	// Write Y (byte 2) - preserved from R0 (either from wrap code or from reading)
	builder.AddInstruction(rom.EncodeMOV(0, 5, 0)) // MOV R5, R0 (restore Y from R0)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write Y)
	// Write Tile (byte 3) - preserved from earlier read
	builder.AddInstruction(rom.EncodeMOV(0, 5, 3)) // MOV R5, R3 (restore Tile)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write Tile)
	// Write Attributes (byte 4) - preserved from earlier read
	builder.AddInstruction(rom.EncodeMOV(0, 5, 2)) // MOV R5, R2 (restore Attributes)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write Attributes)
	// Write Control byte (byte 5) - always 0x03 (enabled, 16x16)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x03 (Control: enabled, 16x16)
	builder.AddImmediate(0x03)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write Control byte)

	// Wait for VBlank to clear (so we only update once per VBlank period)
	waitVBlankClearStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x803E (VBLANK_FLAG)
	builder.AddImmediate(0x803E)
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read VBlank flag)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 5, 7)) // CMP R5, R7
	builder.AddInstruction(rom.EncodeBNE())        // BNE wait_vblank_clear_start (if flag is 1, keep waiting)
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, waitVBlankClearStart)
	builder.AddImmediate(uint16(offset))

	// Jump back to main loop
	builder.AddInstruction(rom.EncodeJMP())
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, mainLoopStart)
	builder.AddImmediate(uint16(offset))

	// Build ROM
	if err := builder.BuildROM(entryBank, entryOffset, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error building ROM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Colored moving sprite ROM built: %s\n", outputPath)
	fmt.Println("This ROM creates a 16x16 sprite with 4 colored quadrants:")
	fmt.Println("  Top-left: Red (color 1)")
	fmt.Println("  Top-right: Green (color 2)")
	fmt.Println("  Bottom-left: Blue (color 3)")
	fmt.Println("  Bottom-right: Yellow (color 4)")
	fmt.Println("Use palette 1 in the tile viewer to see the pattern!")
}
