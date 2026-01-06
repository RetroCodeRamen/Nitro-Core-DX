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
		value := c.Mem.Read16(c.State.DBR, addr)
		c.SetRegister(reg1, value)
		c.UpdateFlags(value)
		c.State.Cycles += 2 // Memory access
		return nil
		
	case 3: // MOV [R1], R2 - Store to memory
		addr := c.GetRegister(reg1)
		value := c.GetRegister(reg2)
		// I/O addresses (0x8000+) always use bank 0
		bank := c.State.DBR
		if addr >= 0x8000 {
			bank = 0
		}
		c.Mem.Write16(bank, addr, value)
		c.State.Cycles += 2 // Memory access
		return nil
		
	case 4: // PUSH R1 - Push to stack
		value := c.GetRegister(reg1)
		c.Push16(value)
		c.State.Cycles += 2
		return nil
		
	case 5: // POP R1 - Pop from stack
		value := c.Pop16()
		c.SetRegister(reg1, value)
		c.UpdateFlags(value)
		c.State.Cycles += 2
		return nil
		
	default:
		return fmt.Errorf("unknown MOV mode: %d", mode)
	}
}

// executeADD executes ADD instructions
func (c *CPU) executeADD(mode, reg1, reg2 uint8) error {
	var value uint16
	if mode == 0 {
		// ADD R1, R2
		value = c.GetRegister(reg2)
	} else {
		// ADD R1, #imm
		value = c.FetchImmediate()
		c.State.Cycles++
	}
	
	a := c.GetRegister(reg1)
	result := a + value
	c.SetRegister(reg1, result)
	c.UpdateFlagsWithOverflow(a, value, result, false)
	c.State.Cycles++
	return nil
}

// executeSUB executes SUB instructions
func (c *CPU) executeSUB(mode, reg1, reg2 uint8) error {
	var value uint16
	if mode == 0 {
		// SUB R1, R2
		value = c.GetRegister(reg2)
	} else {
		// SUB R1, #imm
		value = c.FetchImmediate()
		c.State.Cycles++
	}
	
	a := c.GetRegister(reg1)
	result := a - value
	c.SetRegister(reg1, result)
	c.UpdateFlagsWithOverflow(a, value, result, true)
	c.State.Cycles++
	return nil
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
		// This is more graceful than crashing
		c.SetRegister(reg1, 0xFFFF)
		c.UpdateFlags(0xFFFF)
		c.State.Cycles += 4
		return nil
	}
	
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
	var shift uint8
	if mode == 0 {
		// SHR R1, R2
		shift = uint8(c.GetRegister(reg2) & 0xF) // Limit to 0-15
	} else {
		// SHR R1, #imm
		imm := c.FetchImmediate()
		shift = uint8(imm & 0xF)
		c.State.Cycles++
	}
	
	a := c.GetRegister(reg1)
	var carry bool
	if shift > 0 {
		carry = (a & (1 << (shift - 1))) != 0
		result := a >> shift
		c.SetRegister(reg1, result)
		c.UpdateFlagsWithCarry(result, carry)
	} else {
		c.UpdateFlags(a)
	}
	c.State.Cycles++
	return nil
}

// executeCMPAndBranches executes CMP and branch instructions
func (c *CPU) executeCMPAndBranches(mode, reg1, reg2 uint8) error {
	opcode := mode & 0xF
	
	if opcode == 0 { // CMP
		var value uint16
		if mode == 0 {
			// CMP R1, R2
			value = c.GetRegister(reg2)
		} else {
			// CMP R1, #imm
			value = c.FetchImmediate()
			c.State.Cycles++
		}
		
		a := c.GetRegister(reg1)
		result := a - value
		c.UpdateFlagsWithOverflow(a, value, result, true)
		c.State.Cycles++
		return nil
	}
	
	// Branch instructions (opcode 0xC1-0xC6)
	offset := int16(c.FetchImmediate())
	c.State.Cycles++
	
	var shouldBranch bool
	switch opcode {
	case 0x1: // BEQ - Branch if equal
		shouldBranch = c.GetFlag(FlagZ)
	case 0x2: // BNE - Branch if not equal
		shouldBranch = !c.GetFlag(FlagZ)
	case 0x3: // BGT - Branch if greater (signed)
		shouldBranch = !c.GetFlag(FlagZ) && !c.GetFlag(FlagN)
	case 0x4: // BLT - Branch if less (signed)
		shouldBranch = c.GetFlag(FlagN)
	case 0x5: // BGE - Branch if >= (signed)
		shouldBranch = !c.GetFlag(FlagN)
	case 0x6: // BLE - Branch if <= (signed)
		shouldBranch = c.GetFlag(FlagZ) || c.GetFlag(FlagN)
	default:
		return fmt.Errorf("unknown branch opcode: 0x%X", opcode)
	}
	
	if shouldBranch {
		// Offset is relative to PC after instruction and offset word
		newOffset := int32(c.State.PCOffset) + int32(offset)
		if newOffset < 0 {
			c.State.PCOffset = 0
		} else if newOffset > 0xFFFF {
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

// executeJMP executes JMP instruction
func (c *CPU) executeJMP() error {
	offset := int16(c.FetchImmediate())
	c.State.Cycles++
	
	// Offset is relative to PC after instruction and offset word
	newOffset := int32(c.State.PCOffset) + int32(offset)
	if newOffset < 0 {
		c.State.PCOffset = 0
	} else if newOffset > 0xFFFF {
		c.State.PCOffset = 0xFFFF
	} else {
		c.State.PCOffset = uint16(newOffset)
	}
	// Ensure PC stays aligned (instructions are 16-bit)
	c.State.PCOffset &^= 1
	c.State.Cycles++
	return nil
}

// executeCALL executes CALL instruction
func (c *CPU) executeCALL() error {
	offset := int16(c.FetchImmediate())
	c.State.Cycles++
	
	// Push return address (PBR:PC)
	c.Push16(uint16(c.State.PBR))
	c.Push16(c.State.PCOffset)
	
	// Jump to target
	newOffset := int32(c.State.PCOffset) + int32(offset)
	if newOffset < 0 {
		c.State.PCOffset = 0
	} else if newOffset > 0xFFFF {
		c.State.PCOffset = 0xFFFF
	} else {
		c.State.PCOffset = uint16(newOffset)
	}
	// Ensure PC stays aligned (instructions are 16-bit)
	c.State.PCOffset &^= 1
	c.State.Cycles += 3
	return nil
}

// executeRET executes RET instruction
func (c *CPU) executeRET() error {
	// Pop return address
	c.State.PCOffset = c.Pop16()
	// Ensure PC stays aligned (instructions are 16-bit)
	c.State.PCOffset &^= 1
	c.State.PBR = uint8(c.Pop16())
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
func (c *CPU) Pop16() uint16 {
	// Stack grows downward
	c.State.SP++
	high := uint16(c.Mem.Read8(0, c.State.SP))
	c.State.SP++
	low := uint16(c.Mem.Read8(0, c.State.SP))
	
	// Wrap around if overflow
	if c.State.SP > 0x1FFF {
		c.State.SP = 0x0000
	}
	
	return (high << 8) | low
}


