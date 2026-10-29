package corelx

import (
	"fmt"

	"nitro-core-dx/internal/rom"
)

// Part of generateBuiltinCall. See the register contract on that function.
func (cg *CodeGenerator) builtinRaster(name string, args []Expr, destReg uint8) error {
	switch name {
	case "raster.enable":
		// raster.enable(table_base: u16, layer_mask: u8, rebind: bool, priority: bool, tilemap_base: bool, source_mode: bool)
		// Args:
		//   R0 = scanline table base in VRAM
		//   R1 = layer mask bits [3:0] for BG0-BG3
		//   R2 = include rebind table
		//   R3 = include priority table
		//   R4 = include tilemap-base table
		//   R5 = include source-mode table
		if len(args) != 6 {
			return fmt.Errorf("raster.enable requires 6 arguments (table_base, layer_mask, rebind, priority, tilemap_base, source_mode)")
		}

		// Program table base.
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		cg.builder.AddImmediate(0x805E)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 0))
		cg.builder.AddInstruction(rom.EncodeAND(1, 7, 0))
		cg.builder.AddImmediate(0x00FF)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7))

		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		cg.builder.AddImmediate(0x805F)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 0))
		cg.builder.AddInstruction(rom.EncodeSHR(1, 7, 0))
		cg.builder.AddImmediate(8)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7))

		// Build HDMA control byte in R7.
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(0x01) // enable bit

		// Layer mask -> bits [4:1]
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 1))
		cg.builder.AddInstruction(rom.EncodeAND(1, 6, 0))
		cg.builder.AddImmediate(0x0F)
		cg.builder.AddInstruction(rom.EncodeSHL(1, 6, 0))
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeOR(0, 7, 6))

		// rebind -> bit 5
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 2))
		cg.builder.AddInstruction(rom.EncodeAND(1, 6, 0))
		cg.builder.AddImmediate(0x01)
		cg.builder.AddInstruction(rom.EncodeSHL(1, 6, 0))
		cg.builder.AddImmediate(5)
		cg.builder.AddInstruction(rom.EncodeOR(0, 7, 6))

		// priority -> bit 6
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 3))
		cg.builder.AddInstruction(rom.EncodeAND(1, 6, 0))
		cg.builder.AddImmediate(0x01)
		cg.builder.AddInstruction(rom.EncodeSHL(1, 6, 0))
		cg.builder.AddImmediate(6)
		cg.builder.AddInstruction(rom.EncodeOR(0, 7, 6))

		// tilemap_base -> bit 7
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 4))
		cg.builder.AddInstruction(rom.EncodeAND(1, 6, 0))
		cg.builder.AddImmediate(0x01)
		cg.builder.AddInstruction(rom.EncodeSHL(1, 6, 0))
		cg.builder.AddImmediate(7)
		cg.builder.AddInstruction(rom.EncodeOR(0, 7, 6))

		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		cg.builder.AddImmediate(0x805D)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7))

		// Extension control: source-mode table bit 0.
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		cg.builder.AddImmediate(0x807F)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 5))
		cg.builder.AddInstruction(rom.EncodeAND(1, 7, 0))
		cg.builder.AddImmediate(0x01)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7))
		return nil

	case "raster.disable":
		// raster.disable()
		// Clears scanline-command enablement and extension flags.
		if len(args) != 0 {
			return fmt.Errorf("raster.disable requires 0 arguments")
		}
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
		cg.builder.AddImmediate(0x805D)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
		cg.builder.AddImmediate(0x807F)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))
		return nil

	case "raster.set_scanline_scroll":
		// raster.set_scanline_scroll(scanline: u16, layer: u8, scroll_x: i16, scroll_y: i16)
		// Args: R0 = scanline, R1 = layer, R2 = scroll_x, R3 = scroll_y
		if len(args) != 4 {
			return fmt.Errorf("raster.set_scanline_scroll requires 4 arguments (scanline, layer, scroll_x, scroll_y)")
		}
		cg.emitComputeRasterScanlineBase(0, 6, 7)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 1))
		cg.builder.AddInstruction(rom.EncodeSHL(1, 7, 0))
		cg.builder.AddImmediate(4)
		cg.builder.AddInstruction(rom.EncodeADD(0, 6, 7))
		cg.emitWriteVRAM16AtAddrReg(6, 2, 7, 0)
		cg.builder.AddInstruction(rom.EncodeADD(1, 6, 0))
		cg.builder.AddImmediate(2)
		cg.emitWriteVRAM16AtAddrReg(6, 3, 7, 0)
		return nil

	case "raster.set_scanline_matrix":
		// raster.set_scanline_matrix(scanline: u16, layer: u8, a: i16, b: i16, c: i16, d: i16)
		// Args: R0 = scanline, R1 = layer, R2 = A, R3 = B, R4 = C, R5 = D
		if len(args) != 6 {
			return fmt.Errorf("raster.set_scanline_matrix requires 6 arguments (scanline, layer, a, b, c, d)")
		}
		cg.emitComputeRasterScanlineBase(0, 6, 7)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 1))
		cg.builder.AddInstruction(rom.EncodeSHL(1, 7, 0))
		cg.builder.AddImmediate(4)
		cg.builder.AddInstruction(rom.EncodeADD(0, 6, 7))
		cg.builder.AddInstruction(rom.EncodeADD(1, 6, 0))
		cg.builder.AddImmediate(4)
		cg.emitWriteVRAM16AtAddrReg(6, 2, 7, 0)
		cg.builder.AddInstruction(rom.EncodeADD(1, 6, 0))
		cg.builder.AddImmediate(2)
		cg.emitWriteVRAM16AtAddrReg(6, 3, 7, 0)
		cg.builder.AddInstruction(rom.EncodeADD(1, 6, 0))
		cg.builder.AddImmediate(2)
		cg.emitWriteVRAM16AtAddrReg(6, 4, 7, 0)
		cg.builder.AddInstruction(rom.EncodeADD(1, 6, 0))
		cg.builder.AddImmediate(2)
		cg.emitWriteVRAM16AtAddrReg(6, 5, 7, 0)
		return nil

	case "raster.set_scanline_center":
		// raster.set_scanline_center(scanline: u16, layer: u8, center_x: i16, center_y: i16)
		// Args: R0 = scanline, R1 = layer, R2 = center_x, R3 = center_y
		if len(args) != 4 {
			return fmt.Errorf("raster.set_scanline_center requires 4 arguments (scanline, layer, center_x, center_y)")
		}
		cg.emitComputeRasterScanlineBase(0, 6, 7)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 1))
		cg.builder.AddInstruction(rom.EncodeSHL(1, 7, 0))
		cg.builder.AddImmediate(4)
		cg.builder.AddInstruction(rom.EncodeADD(0, 6, 7))
		cg.builder.AddInstruction(rom.EncodeADD(1, 6, 0))
		cg.builder.AddImmediate(12)
		cg.emitWriteVRAM16AtAddrReg(6, 2, 7, 0)
		cg.builder.AddInstruction(rom.EncodeADD(1, 6, 0))
		cg.builder.AddImmediate(2)
		cg.emitWriteVRAM16AtAddrReg(6, 3, 7, 0)
		return nil

	case "raster.set_scanline_tilemap_base":
		// raster.set_scanline_tilemap_base(scanline: u16, layer: u8, tilemap_base: u16)
		// Args: R0 = scanline, R1 = layer, R2 = tilemap base
		if len(args) != 3 {
			return fmt.Errorf("raster.set_scanline_tilemap_base requires 3 arguments (scanline, layer, tilemap_base)")
		}
		cg.emitComputeRasterScanlineBase(0, 6, 7)
		cg.builder.AddInstruction(rom.EncodeADD(1, 6, 0))
		cg.builder.AddImmediate(64)
		cg.emitMaybeAddImmediateIfMMIOFlag(0x805D, 0x20, 4, 6, 7)
		cg.emitMaybeAddImmediateIfMMIOFlag(0x805D, 0x40, 4, 6, 7)

		cg.emitLoadMMIO8(7, 0x805D)
		cg.builder.AddInstruction(rom.EncodeAND(1, 7, 0))
		cg.builder.AddImmediate(0x80)
		cg.builder.AddInstruction(rom.EncodeBEQ())
		skipPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)

		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 1))
		cg.builder.AddInstruction(rom.EncodeSHL(1, 7, 0))
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 6, 7))
		cg.emitWriteVRAM16AtAddrReg(6, 2, 7, 0)

		skipPC := uint16(skipPos * 2)
		nextPC := uint16(cg.builder.GetCodeLength() * 2)
		cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		return nil

	case "raster.set_scanline_rebind":
		// raster.set_scanline_rebind(scanline: u16, layer: u8, channel: u8)
		// Args: R0 = scanline, R1 = layer, R2 = channel
		if len(args) != 3 {
			return fmt.Errorf("raster.set_scanline_rebind requires 3 arguments (scanline, layer, channel)")
		}
		cg.emitComputeRasterScanlineBase(0, 6, 7)
		cg.builder.AddInstruction(rom.EncodeADD(1, 6, 0))
		cg.builder.AddImmediate(64)

		cg.emitLoadMMIO8(7, 0x805D)
		cg.builder.AddInstruction(rom.EncodeAND(1, 7, 0))
		cg.builder.AddImmediate(0x20)
		cg.builder.AddInstruction(rom.EncodeBEQ())
		skipPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)

		cg.builder.AddInstruction(rom.EncodeADD(0, 6, 1))
		cg.emitWriteVRAM8AtAddrReg(6, 2, 7, 0)

		skipPC := uint16(skipPos * 2)
		nextPC := uint16(cg.builder.GetCodeLength() * 2)
		cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		return nil

	case "raster.set_scanline_priority":
		// raster.set_scanline_priority(scanline: u16, layer: u8, priority: u8)
		// Args: R0 = scanline, R1 = layer, R2 = priority
		if len(args) != 3 {
			return fmt.Errorf("raster.set_scanline_priority requires 3 arguments (scanline, layer, priority)")
		}
		cg.emitComputeRasterScanlineBase(0, 6, 7)
		cg.builder.AddInstruction(rom.EncodeADD(1, 6, 0))
		cg.builder.AddImmediate(64)
		cg.emitMaybeAddImmediateIfMMIOFlag(0x805D, 0x20, 4, 6, 7)

		cg.emitLoadMMIO8(7, 0x805D)
		cg.builder.AddInstruction(rom.EncodeAND(1, 7, 0))
		cg.builder.AddImmediate(0x40)
		cg.builder.AddInstruction(rom.EncodeBEQ())
		skipPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)

		cg.builder.AddInstruction(rom.EncodeADD(0, 6, 1))
		cg.emitWriteVRAM8AtAddrReg(6, 2, 7, 0)

		skipPC := uint16(skipPos * 2)
		nextPC := uint16(cg.builder.GetCodeLength() * 2)
		cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		return nil

	case "raster.set_scanline_source_mode":
		// raster.set_scanline_source_mode(scanline: u16, layer: u8, mode: u8)
		// Args: R0 = scanline, R1 = layer, R2 = mode
		if len(args) != 3 {
			return fmt.Errorf("raster.set_scanline_source_mode requires 3 arguments (scanline, layer, mode)")
		}
		cg.emitComputeRasterScanlineBase(0, 6, 7)
		cg.builder.AddInstruction(rom.EncodeADD(1, 6, 0))
		cg.builder.AddImmediate(64)
		cg.emitMaybeAddImmediateIfMMIOFlag(0x805D, 0x20, 4, 6, 7)
		cg.emitMaybeAddImmediateIfMMIOFlag(0x805D, 0x40, 4, 6, 7)
		cg.emitMaybeAddImmediateIfMMIOFlag(0x805D, 0x80, 8, 6, 7)

		cg.emitLoadMMIO8(7, 0x807F)
		cg.builder.AddInstruction(rom.EncodeAND(1, 7, 0))
		cg.builder.AddImmediate(0x01)
		cg.builder.AddInstruction(rom.EncodeBEQ())
		skipPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)

		cg.builder.AddInstruction(rom.EncodeADD(0, 6, 1))
		cg.emitWriteVRAM8AtAddrReg(6, 2, 7, 0)

		skipPC := uint16(skipPos * 2)
		nextPC := uint16(cg.builder.GetCodeLength() * 2)
		cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		return nil

	default:
		return fmt.Errorf("%w: %s", errUnknownBuiltin, name)
	}
}
