package corelx

import (
	"fmt"

	"nitro-core-dx/internal/rom"
)

// Part of generateBuiltinCall. See the register contract on that function.
func (cg *CodeGenerator) builtinInput(name string, args []Expr, destReg uint8) error {
	switch name {
	case "input.read":
		// input.read() -> u16
		// Read controller 1 buttons (16-bit)
		// Latch buttons first, then read
		// Latch: write 1 to 0xA001
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001
		cg.builder.AddImmediate(0xA001)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (latch)
		// Read low byte from 0xA000
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA000
		cg.builder.AddImmediate(0xA000)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read low byte)
		// Read high byte from 0xA001
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001
		cg.builder.AddImmediate(0xA001)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 6, 4)) // MOV R6, [R4] (read high byte)
		// Combine: R5 (low) | (R6 << 8) (high)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #8
		cg.builder.AddImmediate(8)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 6, 7)) // SHL R6, R7 -> R6 = high << 8
		cg.builder.AddInstruction(rom.EncodeOR(0, 5, 6))  // OR R5, R6 -> R5 = low | (high << 8)
		// Release latch: write 0 to 0xA001
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001
		cg.builder.AddImmediate(0xA001)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 6)) // MOV [R4], R6 (release latch)
		// Return value in destReg
		cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 5)) // MOV R{destReg}, R5
		return nil

	// APU Functions.
	//
	// LEGACY (scaffolding): the apu.* builtins below target the legacy 4-channel
	// synth (registers 0x9000-0x9021) and are transitional only. The final audio
	// subsystem is YM2608/OPNA; these are pending replacement by the future
	// music.* YM2608 API.
	case "input.poll":
		// input.poll() latches and reads the controller, saving the previous
		// frame's state first so input.pressed/released can detect edges. The
		// curr/prev state lives in the compiler-reserved runtime block.
		if len(args) != 0 {
			return fmt.Errorf("input.poll takes no arguments")
		}
		// prev = curr
		cg.hLoad16(0, inputCurrSlot)
		cg.hStore16(inputPrevSlot, 0)
		// latch: write 1 to 0xA001
		cg.hMovImm(4, 0xA001)
		cg.hMovImm(5, 1)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // [0xA001] = 1
		// read low (0xA000) and high (0xA001) bytes, combine
		cg.hMovImm(4, 0xA000)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // R5 = [0xA000] low
		cg.hMovImm(4, 0xA001)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 6, 4)) // R6 = [0xA001] high
		cg.hMovImm(7, 8)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 6, 7)) // R6 = high << 8
		cg.builder.AddInstruction(rom.EncodeOR(0, 5, 6))  // R5 = low | (high<<8)
		// release latch: write 0 to 0xA001
		cg.hMovImm(4, 0xA001)
		cg.hMovImm(6, 0)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 6)) // [0xA001] = 0
		// curr = combined state
		cg.hStore16(inputCurrSlot, 5)
		return nil

	case "input.held":
		// input.held(button) -> nonzero if the button is down this frame.
		if len(args) != 1 {
			return fmt.Errorf("input.held requires 1 argument (a button, e.g. UP)")
		}
		if err := cg.generateExpr(args[0], 1); err != nil { // R1 = mask
			return err
		}
		cg.hLoad16(0, inputCurrSlot)                      // R0 = curr
		cg.builder.AddInstruction(rom.EncodeAND(0, 0, 1)) // R0 = curr & mask
		if destReg != 0 {
			cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 0))
		}
		return nil

	case "input.pressed":
		// input.pressed(button) -> nonzero on the rising edge (down now, up last
		// frame): (curr & mask) & ~(prev & mask).
		if len(args) != 1 {
			return fmt.Errorf("input.pressed requires 1 argument (a button, e.g. A)")
		}
		if err := cg.generateExpr(args[0], 1); err != nil { // R1 = mask
			return err
		}
		cg.hLoad16(0, inputCurrSlot)
		cg.builder.AddInstruction(rom.EncodeAND(0, 0, 1)) // R0 = curr & mask
		cg.hLoad16(2, inputPrevSlot)
		cg.builder.AddInstruction(rom.EncodeAND(0, 2, 1)) // R2 = prev & mask
		cg.hMovImm(7, 0xFFFF)
		cg.builder.AddInstruction(rom.EncodeXOR(0, 2, 7)) // R2 = ~(prev & mask)
		cg.builder.AddInstruction(rom.EncodeAND(0, 0, 2)) // R0 = (curr&mask) & ~(prev&mask)
		if destReg != 0 {
			cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 0))
		}
		return nil

	case "input.released":
		// input.released(button) -> nonzero on the falling edge:
		// (prev & mask) & ~(curr & mask).
		if len(args) != 1 {
			return fmt.Errorf("input.released requires 1 argument (a button)")
		}
		if err := cg.generateExpr(args[0], 1); err != nil { // R1 = mask
			return err
		}
		cg.hLoad16(0, inputPrevSlot)
		cg.builder.AddInstruction(rom.EncodeAND(0, 0, 1)) // R0 = prev & mask
		cg.hLoad16(2, inputCurrSlot)
		cg.builder.AddInstruction(rom.EncodeAND(0, 2, 1)) // R2 = curr & mask
		cg.hMovImm(7, 0xFFFF)
		cg.builder.AddInstruction(rom.EncodeXOR(0, 2, 7)) // R2 = ~(curr & mask)
		cg.builder.AddInstruction(rom.EncodeAND(0, 0, 2)) // R0 = (prev&mask) & ~(curr&mask)
		if destReg != 0 {
			cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 0))
		}
		return nil

	default:
		return fmt.Errorf("%w: %s", errUnknownBuiltin, name)
	}
}
