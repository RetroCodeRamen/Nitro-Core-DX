package corelx

import (
	"fmt"

	"nitro-core-dx/internal/rom"
)

// Part of generateBuiltinCall. See the register contract on that function.
func (cg *CodeGenerator) builtinFlow(name string, args []Expr, destReg uint8) error {
	switch name {
	case "wait_vblank":
		// Wait for VBlank flag (0x803E, bit 0 = 1 means VBlank)
		// Pattern from manual: Read flag, AND with 0x01, CMP with 0, BEQ if 0 (keep waiting)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x803E
		cg.builder.AddImmediate(0x803E)
		waitPos := cg.builder.GetCodeLength()
		cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read flag)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x01
		cg.builder.AddImmediate(0x01)
		cg.builder.AddInstruction(rom.EncodeAND(0, 5, 7)) // AND R5, R7 (mask to bit 0)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeCMP(0, 5, 7)) // CMP R5, R7 (compare with 0)
		cg.builder.AddInstruction(rom.EncodeBEQ())        // BEQ waitPos (if equal to 0, keep waiting)
		currentPC := uint16(cg.builder.GetCodeLength() * 2)
		offset := cg.calcBranchOffset(currentPC, uint16(waitPos*2))
		cg.builder.AddImmediate(uint16(offset))
		// Advance any playing music once per VBlank (only emitted/called when
		// the program declares a music asset — see emitMusicAdvanceHelper).
		if len(cg.musicAssets) > 0 {
			cg.emitHelperCall("__musicadvance")
		}
		return nil

	case "frame_counter":
		// frame_counter() -> u32 (returns 16-bit frame counter)
		// Read FRAME_COUNTER_LOW (0x803F) and FRAME_COUNTER_HIGH (0x8040)
		// Combine into 16-bit value: (high << 8) | low

		// Read low byte from 0x803F
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x803F
		cg.builder.AddImmediate(0x803F)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read low byte)

		// Read high byte from 0x8040
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8040
		cg.builder.AddImmediate(0x8040)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 6, 4)) // MOV R6, [R4] (read high byte)

		// Combine: (high << 8) | low
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 6)) // MOV R7, R6 (copy high byte)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #8
		cg.builder.AddImmediate(8)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 7, 4)) // SHL R7, R4 -> R7 = high << 8
		cg.builder.AddInstruction(rom.EncodeOR(0, 5, 7))  // OR R5, R7 -> R5 = (high << 8) | low

		// Return value in destReg
		cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 5)) // MOV R{destReg}, R5
		return nil

	default:
		return fmt.Errorf("%w: %s", errUnknownBuiltin, name)
	}
}
