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
		fmt.Println("Usage: go run build_interactive_box.go <output.rom>")
		os.Exit(1)
	}

	outputPath := os.Args[1]
	builder := rom.NewROMBuilder()

	// Interactive box test ROM
	// - Box moves with arrow keys (UP/DOWN/LEFT/RIGHT)
	// - Button A changes background color
	// - Button B changes box color
	entryBank := uint8(1)
	entryOffset := uint16(0x8000)

	fmt.Println("Building interactive box test ROM...")
	fmt.Println("Features:")
	fmt.Println("  - Arrow keys/WASD: Move box")
	fmt.Println("  - Button A (Z key): Change background color")
	fmt.Println("  - Button B (X key): Change box color")

	// ============================================
	// INITIALIZATION: Set up palettes and tile
	// ============================================
	fmt.Println("  [1] Setting up palettes...")

	// Palette 0, Color 0 (background/backdrop) - Start with blue
	// CGRAM address: palette 0, color 0 = 0x00
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012 (CGRAM_ADDR)
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (palette 0, color 0)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write blue color: RGB555 = 0x001F (blue)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013 (CGRAM_DATA)
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x1F (low byte)
	builder.AddImmediate(0x1F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (high byte)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Palette 1, Color 0 (transparent for sprite)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012 (CGRAM_ADDR)
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x20 (palette 1, color 0)
	builder.AddImmediate(0x20)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write transparent/black: RGB555 = 0x0000
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Palette 1, Color 1 (box color) - Start with red
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x21 (palette 1, color 1)
	builder.AddImmediate(0x21)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write red color: RGB555 = 0x7C00 (red)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
	builder.AddImmediate(0x8013)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (low byte)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7C (high byte)
	builder.AddImmediate(0x7C)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// ============================================
	// INITIALIZATION: Set up tile data
	// ============================================
	fmt.Println("  [2] Setting up tile data...")

	// Set VRAM address to 0 (tile 0)
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

	// Write 128 bytes of 0x11 (solid tile, color index 1)
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #128
	builder.AddImmediate(128)

	initVRAMStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010 (VRAM_DATA)
	builder.AddImmediate(0x8010)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x11
	builder.AddImmediate(0x11)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeSUB(1, 6, 0)) // SUB R6, #1
	builder.AddImmediate(1)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE init_vram_start
	currentPC := uint16(builder.GetCodeLength() * 2)
	offset := rom.CalculateBranchOffset(currentPC, initVRAMStart)
	builder.AddImmediate(uint16(offset))

	// ============================================
	// INITIALIZATION: Enable BG0
	// ============================================
	fmt.Println("  [3] Enabling BG0...")
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8008 (BG0_CONTROL)
	builder.AddImmediate(0x8008)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01 (enable)
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// ============================================
	// INITIALIZATION: Set up sprite 0
	// ============================================
	fmt.Println("  [4] Setting up sprite 0...")
	// Initialize sprite position (R0 = X, R1 = Y)
	builder.AddInstruction(rom.EncodeMOV(1, 0, 0)) // MOV R0, #160 (center X)
	builder.AddImmediate(160)
	builder.AddInstruction(rom.EncodeMOV(1, 1, 0)) // MOV R1, #100 (center Y)
	builder.AddImmediate(100)

	// Initialize color indices (R2 = bg color index, R3 = sprite color index)
	builder.AddInstruction(rom.EncodeMOV(1, 2, 0)) // MOV R2, #0 (bg color: blue)
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeMOV(1, 3, 0)) // MOV R3, #0 (sprite color: red)
	builder.AddImmediate(0)

	// ============================================
	// MAIN LOOP
	// ============================================
	fmt.Println("  [5] Setting up main loop...")
	mainLoopStart := uint16(builder.GetCodeLength() * 2)

	// Wait for VBlank
	waitVBlankStart := uint16(builder.GetCodeLength() * 2)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x803E (VBLANK_FLAG)
	builder.AddImmediate(0x803E)
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read VBlank flag, mode 2 = register indirect)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 5, 7)) // CMP R5, R7
	builder.AddInstruction(rom.EncodeBEQ())         // BEQ wait_vblank_start
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, waitVBlankStart)
	builder.AddImmediate(uint16(offset))

	// ============================================
	// Read Input
	// ============================================
	// Latch controller
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001 (CONTROLLER1_LATCH)
	builder.AddImmediate(0xA001)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Read controller state (low byte)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA000 (CONTROLLER1)
	builder.AddImmediate(0xA000)
	builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (8-bit read, zero-extended, mode 2 = register indirect)

	// Release latch
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001
	builder.AddImmediate(0xA001)
	builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 6)) // MOV [R4], R6

	// R5 now contains button state (bits: UP=0, DOWN=1, LEFT=2, RIGHT=3, A=4, B=5)

	// ============================================
	// Handle UP button (bit 0)
	// ============================================
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5 (copy buttons, mode 0 = register)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7 (mask UP bit, mode 0 = register)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE skip_up
	skipUpPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000) // Placeholder
	builder.AddInstruction(rom.EncodeSUB(1, 1, 0)) // SUB R1, #1 (move up)
	builder.AddImmediate(1)
	skipUpTarget := uint16(builder.GetCodeLength() * 2)
	// Fix branch offset
	builder.SetImmediateAt(int(skipUpPC/2), uint16(rom.CalculateBranchOffset(skipUpPC+2, skipUpTarget)))

	// ============================================
	// Handle DOWN button (bit 1)
	// ============================================
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5 (mode 0 = register)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x02
	builder.AddImmediate(0x02)
	builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7 (mode 0 = register)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE skip_down
	skipDownPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	builder.AddInstruction(rom.EncodeADD(1, 1, 0)) // ADD R1, #1 (move down)
	builder.AddImmediate(1)
	skipDownTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(skipDownPC/2), uint16(rom.CalculateBranchOffset(skipDownPC+2, skipDownTarget)))

	// ============================================
	// Handle LEFT button (bit 2)
	// ============================================
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5 (mode 0 = register)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x04
	builder.AddImmediate(0x04)
	builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7 (mode 0 = register)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE skip_left
	skipLeftPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	builder.AddInstruction(rom.EncodeSUB(1, 0, 0)) // SUB R0, #1 (move left)
	builder.AddImmediate(1)
	skipLeftTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(skipLeftPC/2), uint16(rom.CalculateBranchOffset(skipLeftPC+2, skipLeftTarget)))

	// ============================================
	// Handle RIGHT button (bit 3)
	// ============================================
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5 (mode 0 = register)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x08
	builder.AddImmediate(0x08)
	builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7 (mode 0 = register)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE skip_right
	skipRightPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	builder.AddInstruction(rom.EncodeADD(1, 0, 0)) // ADD R0, #1 (move right)
	builder.AddImmediate(1)
	skipRightTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(skipRightPC/2), uint16(rom.CalculateBranchOffset(skipRightPC+2, skipRightTarget)))

	// ============================================
	// Handle A button (bit 4) - Change background color
	// ============================================
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5 (mode 0 = register)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x10
	builder.AddImmediate(0x10)
	builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7 (mode 0 = register)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE skip_a
	skipAPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)

	// Increment background color index (R2)
	builder.AddInstruction(rom.EncodeADD(1, 2, 0)) // ADD R2, #1
	builder.AddImmediate(1)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #4 (wrap at 4 colors)
	builder.AddImmediate(4)
	builder.AddInstruction(rom.EncodeCMP(0, 2, 7)) // CMP R2, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE no_wrap_bg
	wrapBgPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	builder.AddInstruction(rom.EncodeMOV(1, 2, 0)) // MOV R2, #0 (wrap)
	builder.AddImmediate(0)
	wrapBgTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(wrapBgPC/2), uint16(rom.CalculateBranchOffset(wrapBgPC+2, wrapBgTarget)))

	// Set CGRAM address to palette 0, color 0
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write color based on R2 index
	// Color 0: Blue (0x001F), Color 1: Green (0x03E0), Color 2: Red (0x7C00), Color 3: White (0x7FFF)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
	builder.AddImmediate(0x8013)

	// Use a jump table approach - check R2 value
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 2, 7)) // CMP R2, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE bg_not_0
	bgColor0PC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	// Color 0: Blue
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x1F
	builder.AddImmediate(0x1F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeJMP())         // JMP bg_color_done
	bgColorDonePC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000) // Placeholder

	bgColor0Target := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(bgColor0PC/2), uint16(rom.CalculateBranchOffset(bgColor0PC+2, bgColor0Target)))

	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #1
	builder.AddImmediate(1)
	builder.AddInstruction(rom.EncodeCMP(0, 2, 7)) // CMP R2, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE bg_not_1
	bgColor1PC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	// Color 1: Green
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xE0
	builder.AddImmediate(0xE0)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x03
	builder.AddImmediate(0x03)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeJMP())         // JMP bg_color_done
	builder.AddImmediate(uint16(rom.CalculateBranchOffset(uint16(builder.GetCodeLength()*2), bgColorDonePC)))

	bgColor1Target := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(bgColor1PC/2), uint16(rom.CalculateBranchOffset(bgColor1PC+2, bgColor1Target)))

	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #2
	builder.AddImmediate(2)
	builder.AddInstruction(rom.EncodeCMP(0, 2, 7)) // CMP R2, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE bg_not_2
	bgColor2PC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	// Color 2: Red
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7C
	builder.AddImmediate(0x7C)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeJMP())         // JMP bg_color_done
	builder.AddImmediate(uint16(rom.CalculateBranchOffset(uint16(builder.GetCodeLength()*2), bgColorDonePC)))

	bgColor2Target := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(bgColor2PC/2), uint16(rom.CalculateBranchOffset(bgColor2PC+2, bgColor2Target)))

	// Color 3: White (default)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xFF
	builder.AddImmediate(0xFF)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7F
	builder.AddImmediate(0x7F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	bgColorDoneTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(bgColorDonePC/2), uint16(rom.CalculateBranchOffset(bgColorDonePC+2, bgColorDoneTarget)))

	skipATarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(skipAPC/2), uint16(rom.CalculateBranchOffset(skipAPC+2, skipATarget)))

	// ============================================
	// Handle B button (bit 5) - Change box color
	// ============================================
	builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5 (mode 0 = register)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x20
	builder.AddImmediate(0x20)
	builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7 (mode 0 = register)
	builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE skip_b
	skipBPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)

	// Increment sprite color index (R3)
	builder.AddInstruction(rom.EncodeADD(1, 3, 0)) // ADD R3, #1
	builder.AddImmediate(1)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #4
	builder.AddImmediate(4)
	builder.AddInstruction(rom.EncodeCMP(0, 3, 7)) // CMP R3, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE no_wrap_sprite
	wrapSpritePC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	builder.AddInstruction(rom.EncodeMOV(1, 3, 0)) // MOV R3, #0
	builder.AddImmediate(0)
	wrapSpriteTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(wrapSpritePC/2), uint16(rom.CalculateBranchOffset(wrapSpritePC+2, wrapSpriteTarget)))

	// Set CGRAM address to palette 1, color 1
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
	builder.AddImmediate(0x8012)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x21
	builder.AddImmediate(0x21)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write color based on R3 index (same colors as background)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
	builder.AddImmediate(0x8013)

	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	builder.AddImmediate(0)
	builder.AddInstruction(rom.EncodeCMP(0, 3, 7)) // CMP R3, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE sprite_not_0
	spriteColor0PC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	// Color 0: Blue
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x1F
	builder.AddImmediate(0x1F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeJMP())         // JMP sprite_color_done
	spriteColorDonePC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)

	spriteColor0Target := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(spriteColor0PC/2), uint16(rom.CalculateBranchOffset(spriteColor0PC+2, spriteColor0Target)))

	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #1
	builder.AddImmediate(1)
	builder.AddInstruction(rom.EncodeCMP(0, 3, 7)) // CMP R3, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE sprite_not_1
	spriteColor1PC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	// Color 1: Green
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xE0
	builder.AddImmediate(0xE0)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x03
	builder.AddImmediate(0x03)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeJMP())         // JMP sprite_color_done
	builder.AddImmediate(uint16(rom.CalculateBranchOffset(uint16(builder.GetCodeLength()*2), spriteColorDonePC)))

	spriteColor1Target := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(spriteColor1PC/2), uint16(rom.CalculateBranchOffset(spriteColor1PC+2, spriteColor1Target)))

	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #2
	builder.AddImmediate(2)
	builder.AddInstruction(rom.EncodeCMP(0, 3, 7)) // CMP R3, R7
	builder.AddInstruction(rom.EncodeBNE())         // BNE sprite_not_2
	spriteColor2PC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	// Color 2: Red
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7C
	builder.AddImmediate(0x7C)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeJMP())         // JMP sprite_color_done
	builder.AddImmediate(uint16(rom.CalculateBranchOffset(uint16(builder.GetCodeLength()*2), spriteColorDonePC)))

	spriteColor2Target := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(spriteColor2PC/2), uint16(rom.CalculateBranchOffset(spriteColor2PC+2, spriteColor2Target)))

	// Color 3: White (default)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xFF
	builder.AddImmediate(0xFF)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7F
	builder.AddImmediate(0x7F)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	spriteColorDoneTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(spriteColorDonePC/2), uint16(rom.CalculateBranchOffset(spriteColorDonePC+2, spriteColorDoneTarget)))

	skipBTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(skipBPC/2), uint16(rom.CalculateBranchOffset(skipBPC+2, skipBTarget)))

	// ============================================
	// Update sprite position in OAM
	// ============================================
	// Set OAM address to sprite 0
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014 (OAM_ADDR)
	builder.AddImmediate(0x8014)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (sprite 0)
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write X position (low byte)
	builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015 (OAM_DATA)
	builder.AddImmediate(0x8015)
	builder.AddInstruction(rom.EncodeMOV(0, 5, 0)) // MOV R5, R0 (X position, mode 0 = register)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (mode 3 = register indirect write)

	// Write X position (high bit)
	builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x0100
	builder.AddImmediate(0x0100)
	builder.AddInstruction(rom.EncodeCMP(0, 0, 7)) // CMP R0, R7 (check if X >= 256)
	builder.AddInstruction(rom.EncodeBLT())         // BLT x_low
	xHighPC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01 (high bit set)
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	builder.AddInstruction(rom.EncodeJMP())         // JMP x_done
	xDonePC := uint16(builder.GetCodeLength() * 2)
	builder.AddImmediate(0x0000)

	xHighTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(xHighPC/2), uint16(rom.CalculateBranchOffset(xHighPC+2, xHighTarget)))

	// X < 256
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	xDoneTarget := uint16(builder.GetCodeLength() * 2)
	builder.SetImmediateAt(int(xDonePC/2), uint16(rom.CalculateBranchOffset(xDonePC+2, xDoneTarget)))

	// Write Y position
	builder.AddInstruction(rom.EncodeMOV(0, 5, 1)) // MOV R5, R1 (Y position, mode 0 = register)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (mode 3 = register indirect write)

	// Write tile index (0)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
	builder.AddImmediate(0x00)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write attributes (palette 1)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
	builder.AddImmediate(0x01)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// Write control (enable, 16x16)
	builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x03
	builder.AddImmediate(0x03)
	builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

	// ============================================
	// Loop back to main loop
	// ============================================
	builder.AddInstruction(rom.EncodeJMP())
	currentPC = uint16(builder.GetCodeLength() * 2)
	offset = rom.CalculateBranchOffset(currentPC, mainLoopStart)
	builder.AddImmediate(uint16(offset))

	// Build ROM
	if err := builder.BuildROM(entryBank, entryOffset, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error building ROM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Interactive box test ROM built: %s\n", outputPath)
	fmt.Println("\nControls:")
	fmt.Println("  Arrow Keys / WASD: Move box")
	fmt.Println("  Button A (Z key): Change background color (Blue → Green → Red → White)")
	fmt.Println("  Button B (X key): Change box color (Red → Blue → Green → White)")
}
