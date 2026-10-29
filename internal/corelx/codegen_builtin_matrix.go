package corelx

import (
	"fmt"

	"nitro-core-dx/internal/rom"
)

// Part of generateBuiltinCall. See the register contract on that function.
func (cg *CodeGenerator) builtinMatrix(name string, args []Expr, destReg uint8) error {
	switch name {
	case "matrix.enable", "matrix.disable":
		// matrix.enable(layer: u8)
		// matrix.disable(layer: u8)
		// Args: R0 = layer
		if len(args) != 1 {
			return fmt.Errorf("%s requires 1 argument (layer)", name)
		}
		matrixControlAddrs := []uint16{0x8018, 0x802B, 0x8038, 0x8045}
		jumpToEnd := make([]int, 0, 4)
		for i, addr := range matrixControlAddrs {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(uint16(i))
			cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
			cg.builder.AddImmediate(addr)
			cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // current control
			if name == "matrix.enable" {
				cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
				cg.builder.AddImmediate(0x01)
				cg.builder.AddInstruction(rom.EncodeOR(0, 5, 6))
			} else {
				cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
				cg.builder.AddImmediate(0xFE)
				cg.builder.AddInstruction(rom.EncodeAND(0, 5, 6))
			}
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

			cg.builder.AddInstruction(rom.EncodeJMP())
			jumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			jumpToEnd = append(jumpToEnd, jumpPos)

			nextPC := uint16(cg.builder.GetCodeLength() * 2)
			skipPC := uint16(skipPos * 2)
			cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		}
		endPC := uint16(cg.builder.GetCodeLength() * 2)
		for _, jp := range jumpToEnd {
			jpPC := uint16(jp * 2)
			cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC)))
		}
		return nil

	case "matrix.set_matrix":
		// matrix.set_matrix(layer: u8, a: i16, b: i16, c: i16, d: i16)
		// Args: R0 = layer, R1 = A, R2 = B, R3 = C, R4 = D
		if len(args) != 5 {
			return fmt.Errorf("matrix.set_matrix requires 5 arguments (layer, a, b, c, d)")
		}
		matrixAAddrs := []uint16{0x8019, 0x802C, 0x8039, 0x8046}
		jumpToEnd := make([]int, 0, 4)
		for i, addr := range matrixAAddrs {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(uint16(i))
			cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			matrixRegs := []struct {
				base uint16
				reg  uint8
			}{
				{addr, 1},
				{addr + 2, 2},
				{addr + 4, 3},
				{addr + 6, 4},
			}
			for _, m := range matrixRegs {
				cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
				cg.builder.AddImmediate(m.base)
				cg.builder.AddInstruction(rom.EncodeMOV(0, 6, m.reg))
				cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
				cg.builder.AddImmediate(0x00FF)
				cg.builder.AddInstruction(rom.EncodeAND(0, 6, 7))
				cg.builder.AddInstruction(rom.EncodeMOV(3, 5, 6))

				cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
				cg.builder.AddImmediate(m.base + 1)
				cg.builder.AddInstruction(rom.EncodeMOV(0, 6, m.reg))
				cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
				cg.builder.AddImmediate(8)
				cg.builder.AddInstruction(rom.EncodeSHR(0, 6, 7))
				cg.builder.AddInstruction(rom.EncodeMOV(3, 5, 6))
			}

			cg.builder.AddInstruction(rom.EncodeJMP())
			jumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			jumpToEnd = append(jumpToEnd, jumpPos)

			nextPC := uint16(cg.builder.GetCodeLength() * 2)
			skipPC := uint16(skipPos * 2)
			cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		}
		endPC := uint16(cg.builder.GetCodeLength() * 2)
		for _, jp := range jumpToEnd {
			jpPC := uint16(jp * 2)
			cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC)))
		}
		return nil

	case "matrix.set_center":
		// matrix.set_center(layer: u8, x: i16, y: i16)
		// Args: R0 = layer, R1 = centerX, R2 = centerY
		if len(args) != 3 {
			return fmt.Errorf("matrix.set_center requires 3 arguments (layer, x, y)")
		}
		centerAddrs := []uint16{0x8027, 0x8034, 0x8041, 0x804E}
		jumpToEnd := make([]int, 0, 4)
		for i, addr := range centerAddrs {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(uint16(i))
			cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			centerRegs := []struct {
				base uint16
				reg  uint8
			}{
				{addr, 1},
				{addr + 2, 2},
			}
			for _, c := range centerRegs {
				cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
				cg.builder.AddImmediate(c.base)
				cg.builder.AddInstruction(rom.EncodeMOV(0, 5, c.reg))
				cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
				cg.builder.AddImmediate(0x00FF)
				cg.builder.AddInstruction(rom.EncodeAND(0, 5, 6))
				cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

				cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
				cg.builder.AddImmediate(c.base + 1)
				cg.builder.AddInstruction(rom.EncodeMOV(0, 5, c.reg))
				cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
				cg.builder.AddImmediate(8)
				cg.builder.AddInstruction(rom.EncodeSHR(0, 5, 6))
				cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))
			}

			cg.builder.AddInstruction(rom.EncodeJMP())
			jumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			jumpToEnd = append(jumpToEnd, jumpPos)

			nextPC := uint16(cg.builder.GetCodeLength() * 2)
			skipPC := uint16(skipPos * 2)
			cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		}
		endPC := uint16(cg.builder.GetCodeLength() * 2)
		for _, jp := range jumpToEnd {
			jpPC := uint16(jp * 2)
			cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC)))
		}
		return nil

	case "matrix.identity":
		// matrix.identity(layer: u8)
		// Args: R0 = layer
		if len(args) != 1 {
			return fmt.Errorf("matrix.identity requires 1 argument (layer)")
		}
		matrixAAddrs := []uint16{0x8019, 0x802C, 0x8039, 0x8046}
		jumpToEnd := make([]int, 0, 4)
		for i, addr := range matrixAAddrs {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(uint16(i))
			cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			values := []struct {
				base  uint16
				value uint16
			}{
				{addr, 0x0100},     // A
				{addr + 2, 0x0000}, // B
				{addr + 4, 0x0000}, // C
				{addr + 6, 0x0100}, // D
			}
			for _, v := range values {
				cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
				cg.builder.AddImmediate(v.base)
				cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
				cg.builder.AddImmediate(v.value & 0x00FF)
				cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

				cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
				cg.builder.AddImmediate(v.base + 1)
				cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
				cg.builder.AddImmediate(v.value >> 8)
				cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))
			}

			cg.builder.AddInstruction(rom.EncodeJMP())
			jumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			jumpToEnd = append(jumpToEnd, jumpPos)

			nextPC := uint16(cg.builder.GetCodeLength() * 2)
			skipPC := uint16(skipPos * 2)
			cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		}
		endPC := uint16(cg.builder.GetCodeLength() * 2)
		for _, jp := range jumpToEnd {
			jpPC := uint16(jp * 2)
			cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC)))
		}
		return nil

	case "matrix.set_flags":
		// matrix.set_flags(layer: u8, mirror_h: bool, mirror_v: bool, outside_mode: u8, direct_color: bool)
		// Args: R0 = layer, R1 = mirror_h, R2 = mirror_v, R3 = outside_mode, R4 = direct_color
		// Preserves the enable bit while rewriting control bits [5:1].
		if len(args) != 5 {
			return fmt.Errorf("matrix.set_flags requires 5 arguments (layer, mirror_h, mirror_v, outside_mode, direct_color)")
		}
		matrixControlAddrs := []uint16{0x8018, 0x802B, 0x8038, 0x8045}
		jumpToEnd := make([]int, 0, 4)
		for i, addr := range matrixControlAddrs {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(uint16(i))
			cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
			cg.builder.AddImmediate(addr)
			cg.builder.AddInstruction(rom.EncodeMOV(2, 6, 5)) // R6 = current control
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
			cg.builder.AddImmediate(0x01)
			cg.builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // R6 = enabled bit only

			cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 1)) // mirror_h
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
			cg.builder.AddImmediate(0x01)
			cg.builder.AddInstruction(rom.EncodeAND(0, 7, 5))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeSHL(0, 7, 5))
			cg.builder.AddInstruction(rom.EncodeOR(0, 6, 7))

			cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 2)) // mirror_v
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
			cg.builder.AddImmediate(0x01)
			cg.builder.AddInstruction(rom.EncodeAND(0, 7, 5))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
			cg.builder.AddImmediate(2)
			cg.builder.AddInstruction(rom.EncodeSHL(0, 7, 5))
			cg.builder.AddInstruction(rom.EncodeOR(0, 6, 7))

			cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 3)) // outside_mode
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
			cg.builder.AddImmediate(0x03)
			cg.builder.AddInstruction(rom.EncodeAND(0, 7, 5))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
			cg.builder.AddImmediate(3)
			cg.builder.AddInstruction(rom.EncodeSHL(0, 7, 5))
			cg.builder.AddInstruction(rom.EncodeOR(0, 6, 7))

			cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 4)) // direct_color
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
			cg.builder.AddImmediate(0x01)
			cg.builder.AddInstruction(rom.EncodeAND(0, 7, 5))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
			cg.builder.AddImmediate(5)
			cg.builder.AddInstruction(rom.EncodeSHL(0, 7, 5))
			cg.builder.AddInstruction(rom.EncodeOR(0, 6, 7))

			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
			cg.builder.AddImmediate(addr)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 5, 6))

			cg.builder.AddInstruction(rom.EncodeJMP())
			jumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			jumpToEnd = append(jumpToEnd, jumpPos)

			nextPC := uint16(cg.builder.GetCodeLength() * 2)
			skipPC := uint16(skipPos * 2)
			cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		}
		endPC := uint16(cg.builder.GetCodeLength() * 2)
		for _, jp := range jumpToEnd {
			jpPC := uint16(jp * 2)
			cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC)))
		}
		return nil

	default:
		return fmt.Errorf("%w: %s", errUnknownBuiltin, name)
	}
}
