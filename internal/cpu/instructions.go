package cpu

import (
	"fmt"
)

// executeNOP executes a NOP instruction
func (c *CPU) executeNOP() error {
	// No operation, 1 cycle already counted
	return nil
}

// executeMOV executes MOV instructions
func (c *CPU) executeMOV(mode, reg1, reg2 uint8) error {
	switch mode {
	case 0: // MOV R1, R2 - Register to register
		value := c.GetRegister(reg2)
		c.SetRegister(reg1, value)
		c.UpdateFlags(value)
		c.State.Cycles++
		return nil

	case 1: // MOV R1, #imm - Immediate to register
		imm := c.FetchImmediate()
		c.SetRegister(reg1, imm)
		c.UpdateFlags(imm)
		c.State.Cycles++
		return nil

	case 2: // MOV R1, [R2] - Load from memory
		addr := c.GetRegister(reg2)
		bank := c.State.DBR

		// Bank 0 high addresses are I/O; higher banks use the normal LoROM/data-bank window.
		if addr >= 0x8000 && bank == 0 {
			value := uint16(c.Mem.Read8(0, addr))
			c.SetRegister(reg1, value)
			c.UpdateFlags(value)
			c.State.Cycles += 2 // Memory access
			return nil
		}

		// Normal memory (WRAM, Extended WRAM, or ROM space): 16-bit read
		value := c.Mem.Read16(bank, addr)
		c.SetRegister(reg1, value)
		c.UpdateFlags(value)
		c.State.Cycles += 2 // Memory access
		return nil

	case 3: // MOV [R1], R2 - Store 16-bit to memory
		addr := c.GetRegister(reg1)
		value := c.GetRegister(reg2)
		bank := c.State.DBR

		// Bank 0 high addresses are I/O; higher banks use the normal LoROM/data-bank window.
		if addr >= 0x8000 && bank == 0 {
			c.Mem.Write8(0, addr, uint8(value&0xFF))
			c.State.Cycles += 2 // Memory access
			return nil
		}

		// Normal memory (WRAM, Extended WRAM, or ROM space): 16-bit write
		// Note: ROM writes are ignored by memory system, but we still do 16-bit write
		c.Mem.Write16(bank, addr, value)
		c.State.Cycles += 2 // Memory access
		return nil

	case 4: // PUSH R1 - Push to stack
		value := c.GetRegister(reg1)
		c.Push16(value)
		c.State.Cycles += 2
		return nil

	case 5: // POP R1 - Pop from stack
		value, err := c.Pop16()
		if err != nil {
			return fmt.Errorf("POP failed: %w", err)
		}
		c.SetRegister(reg1, value)
		c.UpdateFlags(value)
		c.State.Cycles += 2
		return nil

	case 6: // MOV R1, [R2] - Load 8-bit from memory (zero-extended)
		addr := c.GetRegister(reg2)
		value := uint16(c.Mem.Read8(c.State.DBR, addr))
		c.SetRegister(reg1, value)
		c.UpdateFlags(value)
		c.State.Cycles += 2 // Memory access
		return nil

	case 7: // MOV [R1], R2 - Store 8-bit to memory
		addr := c.GetRegister(reg1)
		value := c.GetRegister(reg2)
		bank := c.State.DBR
		// Bank 0 high addresses are I/O; higher banks use the normal LoROM/data-bank window.
		if addr >= 0x8000 && bank == 0 {
			c.Mem.Write8(0, addr, uint8(value&0xFF))
			c.State.Cycles += 2 // Memory access
			return nil
		}
		c.Mem.Write8(bank, addr, uint8(value&0xFF))
		c.State.Cycles += 2 // Memory access
		return nil

	case 8: // MOV DBR, R1 - Update data bank register from low byte of register
		c.State.DBR = uint8(c.GetRegister(reg1) & 0x00FF)
		c.State.Cycles++
		return nil

	case 9: // MOV R1, [R2+imm] - Indexed load
		disp := int16(c.FetchImmediate())
		base := int32(c.GetRegister(reg2))
		addr := uint16(base + int32(disp))
		bank := c.State.DBR

		// I/O addresses (offset 0x8000+ in bank 0) are 8-bit only
		// For I/O addresses, read 8-bit and zero-extend to 16-bit
		var value uint16
		if addr >= 0x8000 && bank == 0 {
			value = uint16(c.Mem.Read8(0, addr))
		} else {
			value = c.Mem.Read16(bank, addr)
		}
		c.SetRegister(reg1, value)
		c.UpdateFlags(value)
		c.State.Cycles += 3 // indexed address calc + memory access
		return nil

	case 10: // MOV [R1+imm], R2 - Indexed store
		disp := int16(c.FetchImmediate())
		base := int32(c.GetRegister(reg1))
		addr := uint16(base + int32(disp))
		value := c.GetRegister(reg2)
		bank := c.State.DBR

		// I/O addresses (offset 0x8000+ in bank 0) are 8-bit only
		// For I/O addresses, always use bank 0 and write only the low byte
		if addr >= 0x8000 && bank == 0 {
			c.Mem.Write8(0, addr, uint8(value&0xFF))
		} else {
			c.Mem.Write16(bank, addr, value)
		}
		c.State.Cycles += 3 // indexed address calc + memory access
		return nil

	case 11: // MOV R1, [R2]+ - Load with post-increment (word stride)
		addr := c.GetRegister(reg2)
		bank := c.State.DBR

		var value uint16
		if addr >= 0x8000 && bank == 0 {
			value = uint16(c.Mem.Read8(0, addr))
		} else {
			value = c.Mem.Read16(bank, addr)
		}
		c.SetRegister(reg1, value)
		c.UpdateFlags(value)
		c.SetRegister(reg2, addr+2)
		c.State.Cycles += 3 // memory access + pointer update
		return nil

	case 12: // MOV [R1]-, R2 - Pre-decrement store (word stride)
		addr := c.GetRegister(reg1) - 2
		c.SetRegister(reg1, addr)
		value := c.GetRegister(reg2)
		bank := c.State.DBR

		if addr >= 0x8000 && bank == 0 {
			c.Mem.Write8(0, addr, uint8(value&0xFF))
		} else {
			c.Mem.Write16(bank, addr, value)
		}
		c.State.Cycles += 3 // pointer update + memory access
		return nil

	// MOV mode 13/14 are optional byte variants for the indexed [R+imm] addressing.
	// Implemented per `docs/specifications/CPU_AMPED_EXTENSION_DESIGN.md`.
	case 13: // MOV R1, [R2+imm] - Load 8-bit from indexed memory, zero-extended
		disp := int16(c.FetchImmediate())
		base := int32(c.GetRegister(reg2))
		addr := uint16(base + int32(disp))
		bank := c.State.DBR

		// Bank 0 high addresses are I/O; higher banks use the normal LoROM/data-bank window.
		if addr >= 0x8000 && bank == 0 {
			value := uint16(c.Mem.Read8(0, addr))
			c.SetRegister(reg1, value)
			c.UpdateFlags(value)
			c.State.Cycles += 3 // indexed addr calc + memory access
			return nil
		}

		value := uint16(c.Mem.Read8(bank, addr))
		c.SetRegister(reg1, value)
		c.UpdateFlags(value)
		c.State.Cycles += 3 // indexed addr calc + memory access
		return nil

	case 14: // MOV [R1+imm], R2 - Store 8-bit to indexed memory, low-byte only
		disp := int16(c.FetchImmediate())
		base := int32(c.GetRegister(reg1))
		addr := uint16(base + int32(disp))
		value := c.GetRegister(reg2)
		bank := c.State.DBR

		// Bank 0 high addresses are I/O; higher banks use the normal LoROM/data-bank window.
		if addr >= 0x8000 && bank == 0 {
			c.Mem.Write8(0, addr, uint8(value&0xFF))
		} else {
			c.Mem.Write8(bank, addr, uint8(value&0xFF))
		}
		c.State.Cycles += 3 // indexed addr calc + memory access
		return nil

	default:
		return fmt.Errorf("unknown MOV mode: %d (valid: 0-12; reserved: 13-15)", mode)
	}
}

// executeADD executes ADD instructions
func (c *CPU) executeADD(mode, reg1, reg2 uint8) error {
	switch mode {
	case 0: // ADD R1, R2 (16-bit)
		value := c.GetRegister(reg2)
		a := c.GetRegister(reg1)
		result := a + value
		c.SetRegister(reg1, result)
		c.UpdateFlagsWithOverflow(a, value, result, false)
		c.State.Cycles++
		return nil

	case 1: // ADD R1, #imm (16-bit)
		value := c.FetchImmediate()
		c.State.Cycles++
		a := c.GetRegister(reg1)
		result := a + value
		c.SetRegister(reg1, result)
		c.UpdateFlagsWithOverflow(a, value, result, false)
		c.State.Cycles++
		return nil

	case 2: // ADD.B R1, R2 (8-bit low-byte, result zero-extended)
		a8 := uint8(c.GetRegister(reg1) & 0x00FF)
		b8 := uint8(c.GetRegister(reg2) & 0x00FF)
		result8 := a8 + b8
		c.SetRegister(reg1, uint16(result8))
		c.UpdateFlagsWithOverflow8(a8, b8, result8, false)
		c.State.Cycles++
		return nil

	case 3: // ADD.B R1, #imm (8-bit low-byte immediate, result zero-extended)
		imm := c.FetchImmediate()
		c.State.Cycles++
		a8 := uint8(c.GetRegister(reg1) & 0x00FF)
		b8 := uint8(imm & 0x00FF)
		result8 := a8 + b8
		c.SetRegister(reg1, uint16(result8))
		c.UpdateFlagsWithOverflow8(a8, b8, result8, false)
		c.State.Cycles++
		return nil

	default:
		return fmt.Errorf("unknown ADD mode: %d", mode)
	}
}

// executeSUB executes SUB instructions
func (c *CPU) executeSUB(mode, reg1, reg2 uint8) error {
	switch mode {
	case 0: // SUB R1, R2 (16-bit)
		value := c.GetRegister(reg2)
		a := c.GetRegister(reg1)
		result := a - value
		c.SetRegister(reg1, result)
		c.UpdateFlagsWithOverflow(a, value, result, true)
		c.State.Cycles++
		return nil

	case 1: // SUB R1, #imm (16-bit)
		value := c.FetchImmediate()
		c.State.Cycles++
		a := c.GetRegister(reg1)
		result := a - value
		c.SetRegister(reg1, result)
		c.UpdateFlagsWithOverflow(a, value, result, true)
		c.State.Cycles++
		return nil

	case 2: // SUB.B R1, R2 (8-bit low-byte, result zero-extended)
		a8 := uint8(c.GetRegister(reg1) & 0x00FF)
		b8 := uint8(c.GetRegister(reg2) & 0x00FF)
		result8 := a8 - b8
		c.SetRegister(reg1, uint16(result8))
		c.UpdateFlagsWithOverflow8(a8, b8, result8, true)
		c.State.Cycles++
		return nil

	case 3: // SUB.B R1, #imm (8-bit low-byte immediate, result zero-extended)
		imm := c.FetchImmediate()
		c.State.Cycles++
		a8 := uint8(c.GetRegister(reg1) & 0x00FF)
		b8 := uint8(imm & 0x00FF)
		result8 := a8 - b8
		c.SetRegister(reg1, uint16(result8))
		c.UpdateFlagsWithOverflow8(a8, b8, result8, true)
		c.State.Cycles++
		return nil

	default:
		return fmt.Errorf("unknown SUB mode: %d", mode)
	}
}

// executeMUL executes MUL instructions
func (c *CPU) executeMUL(mode, reg1, reg2 uint8) error {
	var value uint16
	if mode == 0 {
		// MUL R1, R2
		value = c.GetRegister(reg2)
	} else {
		// MUL R1, #imm
		value = c.FetchImmediate()
		c.State.Cycles++
	}

	a := c.GetRegister(reg1)
	result := uint32(a) * uint32(value)
	c.SetRegister(reg1, uint16(result&0xFFFF)) // Store low 16 bits
	c.UpdateFlags(uint16(result & 0xFFFF))
	c.State.Cycles += 2
	return nil
}

// executeDIV executes DIV instructions
func (c *CPU) executeDIV(mode, reg1, reg2 uint8) error {
	var value uint16
	if mode == 0 {
		// DIV R1, R2
		value = c.GetRegister(reg2)
	} else {
		// DIV R1, #imm
		value = c.FetchImmediate()
		c.State.Cycles++
	}

	if value == 0 {
		// Division by zero: set result to maximum value (0xFFFF)
		// Set division by zero flag so ROM can detect the error
		c.SetRegister(reg1, 0xFFFF)
		c.UpdateFlags(0xFFFF)
		c.SetFlag(FlagD, true) // Set division by zero flag
		c.State.Cycles += 4
		return nil
	}

	// Clear division by zero flag on successful division
	c.SetFlag(FlagD, false)

	a := c.GetRegister(reg1)
	result := a / value
	c.SetRegister(reg1, result)
	c.UpdateFlags(result)
	c.State.Cycles += 4
	return nil
}

// executeAND executes AND instructions
func (c *CPU) executeAND(mode, reg1, reg2 uint8) error {
	var value uint16
	if mode == 0 {
		// AND R1, R2
		value = c.GetRegister(reg2)
	} else {
		// AND R1, #imm
		value = c.FetchImmediate()
		c.State.Cycles++
	}

	a := c.GetRegister(reg1)
	result := a & value
	c.SetRegister(reg1, result)
	c.UpdateFlags(result)
	c.State.Cycles++
	return nil
}

// executeOR executes OR instructions
func (c *CPU) executeOR(mode, reg1, reg2 uint8) error {
	var value uint16
	if mode == 0 {
		// OR R1, R2
		value = c.GetRegister(reg2)
	} else {
		// OR R1, #imm
		value = c.FetchImmediate()
		c.State.Cycles++
	}

	a := c.GetRegister(reg1)
	result := a | value
	c.SetRegister(reg1, result)
	c.UpdateFlags(result)
	c.State.Cycles++
	return nil
}

// executeXOR executes XOR instructions
func (c *CPU) executeXOR(mode, reg1, reg2 uint8) error {
	var value uint16
	if mode == 0 {
		// XOR R1, R2
		value = c.GetRegister(reg2)
	} else {
		// XOR R1, #imm
		value = c.FetchImmediate()
		c.State.Cycles++
	}

	a := c.GetRegister(reg1)
	result := a ^ value
	c.SetRegister(reg1, result)
	c.UpdateFlags(result)
	c.State.Cycles++
	return nil
}

// executeNOT executes NOT instruction
func (c *CPU) executeNOT(reg1 uint8) error {
	a := c.GetRegister(reg1)
	result := ^a
	c.SetRegister(reg1, result)
	c.UpdateFlags(result)
	c.State.Cycles++
	return nil
}

// executeSHL executes SHL instructions
func (c *CPU) executeSHL(mode, reg1, reg2 uint8) error {
	var shift uint8
	if mode == 0 {
		// SHL R1, R2
		shift = uint8(c.GetRegister(reg2) & 0xF) // Limit to 0-15
	} else {
		// SHL R1, #imm
		imm := c.FetchImmediate()
		shift = uint8(imm & 0xF)
		c.State.Cycles++
	}

	a := c.GetRegister(reg1)
	var carry bool
	if shift > 0 {
		carry = (a & (1 << (16 - shift))) != 0
		result := a << shift
		c.SetRegister(reg1, result)
		c.UpdateFlagsWithCarry(result, carry)
	} else {
		c.UpdateFlags(a)
	}
	c.State.Cycles++
	return nil
}

// executeSHR executes SHR instructions
func (c *CPU) executeSHR(mode, reg1, reg2 uint8) error {
	a := c.GetRegister(reg1)
	var count uint8

	switch mode {
	case 0: // SHR R1, R2
		count = uint8(c.GetRegister(reg2) & 0xF) // Limit to 0-15
		if count > 0 {
			carry := (a & (1 << (count - 1))) != 0
			result := a >> count
			c.SetRegister(reg1, result)
			c.UpdateFlagsWithCarry(result, carry)
		} else {
			c.UpdateFlags(a)
		}

	case 1: // SHR R1, #imm
		imm := c.FetchImmediate()
		count = uint8(imm & 0xF)
		c.State.Cycles++
		if count > 0 {
			carry := (a & (1 << (count - 1))) != 0
			result := a >> count
			c.SetRegister(reg1, result)
			c.UpdateFlagsWithCarry(result, carry)
		} else {
			c.UpdateFlags(a)
		}

	case 2: // SAR R1, R2
		count = uint8(c.GetRegister(reg2) & 0xF)
		if count > 0 {
			carry := (a & (1 << (count - 1))) != 0
			result := uint16(int16(a) >> count)
			c.SetRegister(reg1, result)
			c.UpdateFlagsWithCarry(result, carry)
		} else {
			c.UpdateFlags(a)
		}

	case 3: // SAR R1, #imm
		imm := c.FetchImmediate()
		count = uint8(imm & 0xF)
		c.State.Cycles++
		if count > 0 {
			carry := (a & (1 << (count - 1))) != 0
			result := uint16(int16(a) >> count)
			c.SetRegister(reg1, result)
			c.UpdateFlagsWithCarry(result, carry)
		} else {
			c.UpdateFlags(a)
		}

	case 4: // ROL R1, R2 (through carry)
		count = uint8(c.GetRegister(reg2) & 0xF)
		if count > 0 {
			extended := (uint32(c.State.Flags>>FlagC)&0x1)<<16 | uint32(a)
			for i := uint8(0); i < count; i++ {
				out := (extended >> 16) & 0x1
				extended = ((extended << 1) & 0x1FFFF) | out
			}
			result := uint16(extended & 0xFFFF)
			carry := ((extended >> 16) & 0x1) != 0
			c.SetRegister(reg1, result)
			c.UpdateFlagsWithCarry(result, carry)
		} else {
			c.UpdateFlags(a)
		}

	case 5: // ROR R1, R2 (through carry)
		count = uint8(c.GetRegister(reg2) & 0xF)
		if count > 0 {
			extended := (uint32(c.State.Flags>>FlagC)&0x1)<<16 | uint32(a)
			for i := uint8(0); i < count; i++ {
				out := extended & 0x1
				extended = (extended >> 1) | (out << 16)
			}
			result := uint16(extended & 0xFFFF)
			carry := ((extended >> 16) & 0x1) != 0
			c.SetRegister(reg1, result)
			c.UpdateFlagsWithCarry(result, carry)
		} else {
			c.UpdateFlags(a)
		}

	default:
		return fmt.Errorf("unknown shift/rotate mode: %d", mode)
	}

	c.State.Cycles++
	return nil
}

// executeCMPAndBranches executes CMP and branch instructions.
// Encoding contract:
// - Mode 0: CMP R1, R2
// - Mode 1: BEQ rel16
// - Mode 2: BNE rel16
// - Mode 3: BGT rel16
// - Mode 4: BLT rel16
// - Mode 5: BGE rel16
// - Mode 6: BLE rel16
// - Mode 7: CMP R1, #imm16
func (c *CPU) executeCMPAndBranches(mode, reg1, reg2 uint8) error {
	if mode == 0 {
		// CMP R1, R2 (register mode)
		value := c.GetRegister(reg2)
		a := c.GetRegister(reg1)
		result := a - value
		c.UpdateFlagsWithOverflow(a, value, result, true)
		c.State.Cycles++
		return nil
	}

	if mode == 7 {
		// CMP R1, #imm16
		value := c.FetchImmediate()
		c.State.Cycles++
		a := c.GetRegister(reg1)
		result := a - value
		c.UpdateFlagsWithOverflow(a, value, result, true)
		c.State.Cycles++
		return nil
	}

	if mode < 1 || mode > 6 {
		return fmt.Errorf("invalid CMP/branch mode: %d", mode)
	}

	offset := int16(c.FetchImmediate())
	c.State.Cycles++

	var shouldBranch bool
	switch mode {
	case 0x1: // BEQ - Branch if equal
		shouldBranch = c.GetFlag(FlagZ)
	case 0x2: // BNE - Branch if not equal
		shouldBranch = !c.GetFlag(FlagZ)
	case 0x3: // BGT - Branch if greater (signed)
		// BGT: !Z && (N == V)
		shouldBranch = !c.GetFlag(FlagZ) && (c.GetFlag(FlagN) == c.GetFlag(FlagV))
	case 0x4: // BLT - Branch if less (signed)
		// BLT: N != V
		shouldBranch = c.GetFlag(FlagN) != c.GetFlag(FlagV)
	case 0x5: // BGE - Branch if >= (signed)
		// BGE: N == V
		shouldBranch = c.GetFlag(FlagN) == c.GetFlag(FlagV)
	case 0x6: // BLE - Branch if <= (signed)
		// BLE: Z || (N != V)
		shouldBranch = c.GetFlag(FlagZ) || (c.GetFlag(FlagN) != c.GetFlag(FlagV))
	default:
		return fmt.Errorf("unknown branch opcode: 0x%X", mode)
	}

	if shouldBranch {
		// Offset is relative to PC after instruction and offset word
		newOffset := int32(c.State.PCOffset) + int32(offset)

		// Validate that the new offset is valid for ROM execution
		// ROM code must be at offset 0x8000+ within a bank
		if newOffset < 0x8000 {
			return fmt.Errorf("CRITICAL: Branch to invalid address 0x%04X (ROM code must be at 0x8000+). This indicates a bug in the ROM or invalid branch offset", newOffset)
		}
		if newOffset > 0xFFFF {
			c.State.PCOffset = 0xFFFF
		} else {
			c.State.PCOffset = uint16(newOffset)
		}
		// Ensure PC stays aligned (instructions are 16-bit)
		c.State.PCOffset &^= 1
		c.State.Cycles++ // Branch taken penalty
	}

	return nil
}

func (c *CPU) resolveAbsoluteROMTarget(bankReg, offsetReg uint8) (uint8, uint16, error) {
	targetBank := uint8(c.GetRegister(bankReg) & 0x00FF)
	if targetBank < 1 || targetBank > 125 {
		return 0, 0, fmt.Errorf("CRITICAL: absolute jump/call to invalid bank %d (valid ROM banks are 1..125)", targetBank)
	}

	targetOffset := c.GetRegister(offsetReg) &^ 1
	if targetOffset < 0x8000 {
		return 0, 0, fmt.Errorf("CRITICAL: absolute jump/call to invalid offset 0x%04X (ROM code must be at 0x8000+)", targetOffset)
	}

	return targetBank, targetOffset, nil
}

// executeJMP executes JMP instruction
func (c *CPU) executeJMP(mode, reg1, reg2 uint8) error {
	switch mode {
	case 0: // JMP #rel16
		offset := int16(c.FetchImmediate())
		c.State.Cycles++

		// Offset is relative to PC after instruction and offset word
		newOffset := int32(c.State.PCOffset) + int32(offset)

		// Validate that the new offset is valid for ROM execution
		// ROM code must be at offset 0x8000+ within a bank
		if newOffset < 0x8000 {
			return fmt.Errorf("CRITICAL: JMP to invalid address 0x%04X (ROM code must be at 0x8000+). This indicates a bug in the ROM or invalid jump offset", newOffset)
		}
		if newOffset > 0xFFFF {
			c.State.PCOffset = 0xFFFF
		} else {
			c.State.PCOffset = uint16(newOffset)
		}
		// Ensure PC stays aligned (instructions are 16-bit)
		c.State.PCOffset &^= 1
		c.State.Cycles++
		return nil

	case 1: // JMP [bankReg:offsetReg]
		targetBank, targetOffset, err := c.resolveAbsoluteROMTarget(reg1, reg2)
		if err != nil {
			return err
		}
		c.State.PBR = targetBank
		c.State.PCBank = targetBank
		c.State.PCOffset = targetOffset
		c.State.Cycles++
		return nil

	default:
		return fmt.Errorf("unknown JMP mode: %d", mode)
	}
}

// executeCALL executes CALL instruction
func (c *CPU) executeCALL(mode, reg1, reg2 uint8) error {
	// Push return address and flags (matching RET pop order: Flags, PCOffset, PBR).
	// RET pops 3 words: Flags first, then PCOffset, then PBR.
	// Push in the reverse order so LIFO matches.
	c.Push16(uint16(c.State.PCBank)) // PBR  (popped last by RET)
	c.Push16(c.State.PCOffset)       // PCOffset (popped second by RET)
	c.Push16(uint16(c.State.Flags))  // Flags (popped first by RET)

	switch mode {
	case 0: // CALL #rel16
		offset := int16(c.FetchImmediate())
		c.State.Cycles++

		// Jump to target
		newOffset := int32(c.State.PCOffset) + int32(offset)

		// Validate that the new offset is valid for ROM execution
		// ROM code must be at offset 0x8000+ within a bank
		if newOffset < 0x8000 {
			return fmt.Errorf("CRITICAL: CALL to invalid address 0x%04X (ROM code must be at 0x8000+). This indicates a bug in the ROM or invalid call offset", newOffset)
		}
		if newOffset > 0xFFFF {
			c.State.PCOffset = 0xFFFF
		} else {
			c.State.PCOffset = uint16(newOffset)
		}
		// Ensure PC stays aligned (instructions are 16-bit)
		c.State.PCOffset &^= 1
		c.State.Cycles += 3
		return nil

	case 1: // CALL [bankReg:offsetReg]
		targetBank, targetOffset, err := c.resolveAbsoluteROMTarget(reg1, reg2)
		if err != nil {
			return err
		}
		c.State.PBR = targetBank
		c.State.PCBank = targetBank
		c.State.PCOffset = targetOffset
		c.State.Cycles += 3
		return nil

	default:
		return fmt.Errorf("unknown CALL mode: %d", mode)
	}
}

// executeRET executes RET instruction

// executeRET executes RET instruction
func (c *CPU) executeRET() error {
	// Safety check: If PCBank is 0, we're in bank 0 (WRAM/I/O)
	// RET should only be called from ROM (bank 1+)
	if c.State.PCBank == 0 {
		return fmt.Errorf("CRITICAL: RET called from bank 0 (PCOffset=0x%04X). RET should only be called from ROM (bank 1+). This indicates PCBank was incorrectly set or Reset() was called after LoadROM",
			c.State.PCOffset)
	}

	// Safety check: If PCBank is 1+ but PCOffset is < 0x8000, this is invalid
	// ROM code must be at offset 0x8000+ within a bank
	if c.State.PCBank >= 1 && c.State.PCBank <= 125 && c.State.PCOffset < 0x8000 {
		return fmt.Errorf("CRITICAL: RET called from invalid address 0x%04X (ROM code must be at 0x8000+). This indicates PC was corrupted before RET",
			c.State.PCOffset)
	}

	// Check if stack is empty
	if c.State.SP >= 0x1FFF {
		return fmt.Errorf("stack underflow: RET called with empty stack (PC=%02X:%04X, SP=0x%04X)", c.State.PCBank, c.State.PCOffset, c.State.SP)
	}

	// Check if stack is corrupted
	if c.State.SP < 0x0100 {
		return fmt.Errorf("stack underflow: RET called with corrupted stack (SP=0x%04X)", c.State.SP)
	}

	// Pop return address
	// For interrupts, stack contains: PBR, PCOffset, Flags (pushed in that order)
	// For CALL, stack contains: PBR, PCOffset (pushed in that order)
	// RET pops in reverse order (LIFO): Flags (if present), PCOffset, PBR

	// Try to pop flags first (for interrupt returns)
	flagsValue, err := c.Pop16()
	isInterruptReturn := err == nil

	if isInterruptReturn {
		// Flags were on stack (interrupt return) - restore them
		c.State.Flags = uint8(flagsValue & 0xFF)
	}

	// Pop PCOffset (always present for both CALL and interrupt returns)
	pcOffset, err := c.Pop16()
	if err != nil {
		return fmt.Errorf("RET failed to pop PCOffset: %w", err)
	}

	c.State.PCOffset = pcOffset
	// Ensure PC stays aligned (instructions are 16-bit)
	c.State.PCOffset &^= 1

	// Pop PBR (always present for both CALL and interrupt returns)
	pbrValue, err := c.Pop16()
	if err != nil {
		return fmt.Errorf("RET failed to pop PBR: %w", err)
	}

	c.State.PBR = uint8(pbrValue)

	// Validate that PBR is not 0 (ROM code should be in bank 1+)
	if c.State.PBR == 0 {
		return fmt.Errorf("RET popped PBR=0 from stack (PCOffset=0x%04X, SP=0x%04X). This indicates stack corruption",
			c.State.PCOffset, c.State.SP)
	}

	// Validate that PCOffset is in valid ROM range (>= 0x8000)
	if c.State.PCOffset < 0x8000 {
		return fmt.Errorf("RET popped invalid PCOffset=0x%04X (should be >= 0x8000 for ROM). This indicates stack corruption",
			c.State.PCOffset)
	}

	// Keep PCBank in sync with PBR
	c.State.PCBank = c.State.PBR

	c.State.Cycles += 2
	return nil
}

// Push16 pushes a 16-bit value to the stack
func (c *CPU) Push16(value uint16) {
	// Stack grows downward
	c.Mem.Write8(0, c.State.SP, uint8(value&0xFF))
	c.State.SP--
	c.Mem.Write8(0, c.State.SP, uint8(value>>8))
	c.State.SP--

	// Wrap around if underflow
	if c.State.SP > 0x1FFF {
		c.State.SP = 0x1FFF
	}
}

// Pop16 pops a 16-bit value from the stack
// Returns an error if stack underflow occurs
func (c *CPU) Pop16() (uint16, error) {
	// Stack grows downward
	// Stack starts at 0x1FFF and grows downward, so valid SP range is 0x0000-0x1FFF
	// If SP is at or above 0x1FFF, the stack is empty
	if c.State.SP >= 0x1FFF {
		return 0, fmt.Errorf("stack underflow: SP=0x%04X (stack is empty)", c.State.SP)
	}

	// Check if stack is corrupted (SP too low indicates underflow)
	if c.State.SP < 0x0100 {
		return 0, fmt.Errorf("stack underflow: SP=0x%04X (too low - indicates stack corruption)", c.State.SP)
	}

	spBefore := c.State.SP
	c.State.SP++
	addrHigh := c.State.SP
	high := uint16(c.Mem.Read8(0, c.State.SP))
	c.State.SP++
	addrLow := c.State.SP
	low := uint16(c.Mem.Read8(0, c.State.SP))
	spAfter := c.State.SP

	// Debug: Verify SP was actually modified
	if spAfter == spBefore {
		panic(fmt.Sprintf("CRITICAL: Pop16 didn't modify SP! Before: 0x%04X, After: 0x%04X", spBefore, spAfter))
	}

	// Wrap around if overflow
	if c.State.SP > 0x1FFF {
		c.State.SP = 0x0000
		spAfter = 0x0000
	}

	result := (high << 8) | low

	// Debug logging (remove after fixing)
	_ = spBefore
	_ = addrHigh
	_ = addrLow
	_ = spAfter

	return result, nil
}
