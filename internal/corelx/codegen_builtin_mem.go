package corelx

import (
	"fmt"

	"nitro-core-dx/internal/rom"
)

// Part of generateBuiltinCall. See the register contract on that function.
func (cg *CodeGenerator) builtinMem(name string, args []Expr, destReg uint8) error {
	switch name {
	case "mem.write":
		// mem.write(addr: u16, value: u8)
		// Args: R0 = address, R1 = value
		// Writes an 8-bit value to any memory-mapped address.
		if len(args) != 2 {
			return fmt.Errorf("mem.write requires 2 arguments (addr, value)")
		}
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0)) // MOV R4, R0 (addr)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 1)) // MOV R5, R1 (value)
		cg.builder.AddInstruction(rom.EncodeMOV(7, 4, 5)) // MOV [R4], R5 (8-bit store)
		return nil

	case "mem.read":
		// mem.read(addr: u16) -> u8
		// Args: R0 = address
		// Reads an 8-bit value from any memory-mapped address.
		if len(args) != 1 {
			return fmt.Errorf("mem.read requires 1 argument (addr)")
		}
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0))       // MOV R4, R0 (addr)
		cg.builder.AddInstruction(rom.EncodeMOV(6, destReg, 4)) // MOV R{dest}, [R4] (8-bit read)
		return nil

	case "mem.write16":
		// mem.write16(addr: u16, value: u16)
		// 16-bit store. In WRAM this is true little-endian 16-bit; on I/O
		// addresses (bank 0, >= 0x8000) the memory system writes only the low
		// byte (8-bit MMIO) -- the same hardware-accurate routing the FPGA must
		// implement. Args: R0 = address, R1 = value.
		if len(args) != 2 {
			return fmt.Errorf("mem.write16 requires 2 arguments (addr, value)")
		}
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0)) // MOV R4, R0 (addr)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 1)) // MOV R5, R1 (value)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (16-bit store)
		return nil

	case "mem.read16":
		// mem.read16(addr: u16) -> u16
		// 16-bit load. In WRAM this is true little-endian 16-bit; on I/O
		// addresses the memory system zero-extends an 8-bit read. Args: R0 = addr.
		if len(args) != 1 {
			return fmt.Errorf("mem.read16 requires 1 argument (addr)")
		}
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0))       // MOV R4, R0 (addr)
		cg.builder.AddInstruction(rom.EncodeMOV(2, destReg, 4)) // MOV R{dest}, [R4] (16-bit read)
		return nil

	default:
		return fmt.Errorf("%w: %s", errUnknownBuiltin, name)
	}
}
