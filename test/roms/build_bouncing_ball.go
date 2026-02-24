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
		fmt.Println("Usage: go run build_bouncing_ball.go <output.rom>")
		os.Exit(1)
	}

	outputPath := os.Args[1]
	builder := rom.NewROMBuilder()

	// Bouncing Ball ROM
	// Tests clock-driven architecture with:
	// - CPU: Movement calculations
	// - PPU: Sprite rendering
	// - APU: Sound effects on bounce

	entryBank := uint8(1)
	entryOffset := uint16(0x8000)

	// ============================================
	// INITIALIZATION
	// ============================================

	// R0 = Ball X position (initial: 160, center)
	// R1 = Ball Y position (initial: 100, center)
	// R2 = Ball X velocity (initial: 2)
	// R3 = Ball Y velocity (initial: 2)
	// R4 = Temporary register for I/O addresses
	// R5 = Temporary register for values
	// R6 = Frame counter / delay counter
	// R7 = Temporary

	// Initialize ball position: X = 160
	builder.AddInstruction(rom.EncodeMOV(1, 0, 0)) // MOV R0, #imm
	builder.AddImmediate(160)

	// Initialize ball position: Y = 100
	builder.AddInstruction(rom.EncodeMOV(1, 1, 0)) // MOV R1, #imm
	builder.AddImmediate(100)

	// Initialize X velocity: 2
	builder.AddInstruction(rom.EncodeMOV(1, 2, 0)) // MOV R2, #imm
	builder.AddImmediate(2)

	// Initialize Y velocity: 2
	builder.AddInstruction(rom.EncodeMOV(1, 3, 0)) // MOV R3, #imm
	builder.AddImmediate(2)

	// ============================================
	// SET UP PPU
	// ============================================

	// Enable BG0
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8008 (BG0_CONTROL)
	builder.AddImmediate(0x8008)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01 (enable BG0)
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Set up palette: Background = dark blue, Ball = white
	// Set CGRAM address to 0 (background color, palette 0)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012 (CGRAM_ADDR)
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write background color: RGB555 = 0x001F (dark blue)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013 (CGRAM_DATA)
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x1F (low byte)
	builder.AddImmediate(0x1F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (high byte)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Set ball color: palette 1, color 1 = white
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012 (CGRAM_ADDR)
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x11 (palette 1, color 1)
	builder.AddImmediate(0x11)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write white color: RGB555 = 0x7FFF
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013 (CGRAM_DATA)
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xFF (low byte)
	builder.AddImmediate(0xFF)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7F (high byte)
	builder.AddImmediate(0x7F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// ============================================
	// SET UP APU (for bounce sound)
	// ============================================

	// Channel 0: Bounce sound (square wave, 440 Hz)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x9000 (CH0_FREQ_LOW)
	builder.AddImmediate(0x9000)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xB4 (440 Hz low byte)
	builder.AddImmediate(0xB4)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x9001 (CH0_FREQ_HIGH)
	builder.AddImmediate(0x9001)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01 (440 Hz high byte)
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x9002 (CH0_VOLUME)
	builder.AddImmediate(0x9002)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xC0 (volume = 192)
	builder.AddImmediate(0xC0)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	// Note: We'll enable the channel when bounce happens

	// ============================================
	// MAIN LOOP
	// ============================================

	mainLoopStart := uint16(builder.GetCodeLength() * 2)

	// Wait for VBlank (read VBlank flag)
	waitVBlankStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x803E (VBLANK_FLAG)
	builder.AddImmediate(0x803E)
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read flag)
	builder.AddInstruction(rom.EncodeCMP(0, 5, 0)) // CMP R5, #0
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 5, 7)) // CMP R5, R7
	builder.AddInstruction(rom.EncodeBEQ())       // BEQ wait_vblank (if flag is 0, keep waiting)
	currentPC := uint16(builder.GetCodeLength()*2 + 2)
	offset := rom.CalculateBranchOffset(currentPC, waitVBlankStart)
	builder.AddImmediate(uint16(offset))

	// Update ball position: X = X + velocity_x
	builder.AddInstruction(rom.EncodeADD(0, 0, 2)) // ADD R0, R2 (X += velocity_x)

	// Update ball position: Y = Y + velocity_y
	builder.AddInstruction(rom.EncodeADD(0, 1, 3)) // ADD R1, R3 (Y += velocity_y)

	// Check X bounds (0 <= X <= 320-16, ball is 16x16)
	// If X < 0, bounce (reverse velocity)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 0, 7)) // CMP R0, R7
	builder.AddInstruction(rom.EncodeBLT())       // BLT bounce_x
	bounceXLabel := uint16(builder.GetCodeLength() * 2)
	currentPC = uint16(builder.GetCodeLength()*2 + 2)
	offset = rom.CalculateBranchOffset(currentPC, bounceXLabel)
	builder.AddImmediate(uint16(offset))

	// If X >= 304 (320-16), bounce
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #304
	builder.AddImmediate(304)
	builder.AddInstruction(rom.EncodeCMP(0, 0, 7)) // CMP R0, R7
	builder.AddInstruction(rom.EncodeBGE())       // BGE bounce_x
	currentPC = uint16(builder.GetCodeLength()*2 + 2)
	offset = rom.CalculateBranchOffset(currentPC, bounceXLabel)
	builder.AddImmediate(uint16(offset))

	// Skip bounce_x if no collision
	builder.AddInstruction(rom.EncodeJMP())
	skipBounceXLabel := uint16(builder.GetCodeLength() * 2)
	currentPC = uint16(builder.GetCodeLength()*2 + 2)
	offset = rom.CalculateBranchOffset(currentPC, skipBounceXLabel)
	builder.AddImmediate(uint16(offset))

	// bounce_x: Reverse X velocity and play sound
	bounceXLabel = uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeSUB(0, 7, 2))  // SUB R7, R2 (R7 = 0 - velocity_x)
	builder.AddInstruction(rom.EncodeMOV(0, 2, 7)) // MOV R2, R7 (velocity_x = -velocity_x)

	// Play bounce sound
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x9003 (CH0_CONTROL)
	builder.AddImmediate(0x9003)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x03 (enable, square wave)
	builder.AddImmediate(0x03)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// skip_bounce_x:
	skipBounceXLabel = uint16(builder.GetCodeLength() * 2)

	// Check Y bounds (0 <= Y <= 200-16)
	// If Y < 0, bounce
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 1, 7)) // CMP R1, R7
	builder.AddInstruction(rom.EncodeBLT())       // BLT bounce_y
	bounceYLabel := uint16(builder.GetCodeLength() * 2)
	currentPC = uint16(builder.GetCodeLength()*2 + 2)
	offset = rom.CalculateBranchOffset(currentPC, bounceYLabel)
	builder.AddImmediate(uint16(offset))

	// If Y >= 184 (200-16), bounce
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #184
	builder.AddImmediate(184)
	builder.AddInstruction(rom.EncodeCMP(0, 1, 7)) // CMP R1, R7
	builder.AddInstruction(rom.EncodeBGE())       // BGE bounce_y
	currentPC = uint16(builder.GetCodeLength()*2 + 2)
	offset = rom.CalculateBranchOffset(currentPC, bounceYLabel)
	builder.AddImmediate(uint16(offset))

	// Skip bounce_y if no collision
	builder.AddInstruction(rom.EncodeJMP())
	skipBounceYLabel := uint16(builder.GetCodeLength() * 2)
	currentPC = uint16(builder.GetCodeLength()*2 + 2)
	offset = rom.CalculateBranchOffset(currentPC, skipBounceYLabel)
	builder.AddImmediate(uint16(offset))

	// bounce_y: Reverse Y velocity and play sound
	bounceYLabel = uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeSUB(0, 7, 3))  // SUB R7, R3 (R7 = 0 - velocity_y)
	builder.AddInstruction(rom.EncodeMOV(0, 3, 7)) // MOV R3, R7 (velocity_y = -velocity_y)

	// Play bounce sound
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x9003 (CH0_CONTROL)
	builder.AddImmediate(0x9003)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x03 (enable, square wave)
	builder.AddImmediate(0x03)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// skip_bounce_y:
	skipBounceYLabel = uint16(builder.GetCodeLength() * 2)

	// Update sprite position (Sprite 0)
	// OAM address: 0x8015 (OAM_DATA), but we need to set OAM_ADDR first
	// Sprite 0 is at OAM offset 0
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014 (OAM_ADDR)
	builder.AddImmediate(0x8014)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (sprite 0)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write sprite X position (low byte)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015 (OAM_DATA)
	builder.AddImmediate(0x8015)
	builder.AddInstruction(rom.EncodeMOV(0, 5, 0)) // MOV R5, R0 (X position)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write sprite X position (high byte - sign bit)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (positive)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write sprite Y position
	builder.AddInstruction(rom.EncodeMOV(0, 5, 1)) // MOV R5, R1 (Y position)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write sprite tile index (tile 0, which we'll fill with a circle)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (tile 0)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write sprite attributes (palette 1)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x10 (palette 1)
	builder.AddImmediate(0x10)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write sprite control (enable, 16x16)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x03 (enable + 16x16)
	builder.AddImmediate(0x03)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Jump back to main loop
	builder.AddInstruction(rom.EncodeJMP())
	currentPC = uint16(builder.GetCodeLength()*2 + 2)
	offset = rom.CalculateBranchOffset(currentPC, mainLoopStart)
	builder.AddImmediate(uint16(offset))

	// Build ROM
	if err := builder.BuildROM(entryBank, entryOffset, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error building ROM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Bouncing Ball ROM built successfully: %s\n", outputPath)
	fmt.Printf("ROM size: %d bytes\n", builder.GetCodeLength()*2)
	fmt.Printf("Entry point: Bank %d, Offset 0x%04X\n", entryBank, entryOffset)
}
