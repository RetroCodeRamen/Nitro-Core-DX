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
		
		// I/O addresses (offset 0x8000+ in bank 0) are 8-bit only
		// For I/O addresses, read 8-bit and zero-extend to 16-bit
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

		// I/O addresses (offset 0x8000+ in bank 0) are 8-bit only
		// For I/O addresses, always use bank 0 and write only the low byte
		if addr >= 0x8000 && bank == 0 {
			// I/O registers are 8-bit, so only write the low byte
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
		// I/O addresses (0x8000+) always use bank 0
		bank := c.State.DBR
		if addr >= 0x8000 {
			bank = 0
		}
		c.Mem.Write8(bank, addr, uint8(value&0xFF))
		c.State.Cycles += 2 // Memory access
		return nil

	case 8: // Reserved - return error
		// Mode 8 is reserved and not currently implemented
		// Returning error to catch ROM bugs early
		return fmt.Errorf("MOV mode 8 is reserved and not implemented (this indicates a ROM bug or invalid instruction encoding)")

	default:
		return fmt.Errorf("unknown MOV mode: %d (modes 0-7 are valid, 8-15 are reserved)", mode)
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
	// Mode field encoding:
	// - Mode 0: CMP R1, R2 (register mode)
	// - Mode 1: CMP R1, #imm (immediate mode) OR BEQ (branch)
	// - Mode 2-6: Branch instructions (BNE, BGT, BLT, BGE, BLE)
	// 
	// To distinguish CMP immediate from BEQ, we check if reg2 is used:
	// - CMP uses reg1 and reg2 (or immediate)
	// - Branches don't use reg1/reg2 meaningfully (they're part of instruction encoding)
	// Actually, looking at encoding: CMP immediate is mode=1 with immediate following
	// BEQ is mode=1 but it's a branch instruction, not CMP
	//
	// The real distinction: CMP mode 0-1, branches mode 1-6
	// But mode 1 overlaps! Let's check the instruction format:
	// - 0xC000 = CMP reg (mode=0)
	// - 0xC100 = BEQ (mode=1, but this is a branch, not CMP!)
	// - 0xC1XX where XX has reg1/reg2 could be CMP immediate?
	//
	// Actually, based on ROM builder: EncodeCMP(mode=1, reg1, reg2) = 0xC100 | (reg1<<4) | reg2
	// So 0xC100 with reg1=0, reg2=0 would be BEQ, but 0xC100 with reg1=1, reg2=0 would be CMP R1, #imm
	//
	// The issue: we can't distinguish CMP immediate from BEQ by mode alone!
	// Solution: CMP immediate must use a different mode value, OR the instruction set doesn't support CMP immediate
	//
	// Let me check the spec: According to docs, CMP supports mode 0-1, where mode 1 is immediate
	// But branches also use mode 1-6. This is a conflict!
	//
	// For now, let's assume: if mode == 0, it's CMP reg. If mode >= 1 and < 7, check if it's a branch first.
	// If mode == 1 and it's not BEQ (reg1==0 && reg2==0), treat as CMP immediate.
	// Actually, that's too complex. Let's use a simpler rule:
	// - Mode 0: CMP reg
	// - Mode 1: Check if reg1==0 && reg2==0 -> BEQ, else CMP immediate
	// - Mode 2-6: Branch instructions
	
	if mode == 0 {
		// CMP R1, R2 (register mode)
		value := c.GetRegister(reg2)
		a := c.GetRegister(reg1)
		result := a - value
		c.UpdateFlagsWithOverflow(a, value, result, true)
		c.State.Cycles++
		return nil
	}
	
	// For mode >= 1, check if it's a branch instruction
	// Branch instructions: mode 1-6 map to BEQ, BNE, BGT, BLT, BGE, BLE
	// But CMP immediate also uses mode 1!
	// 
	// Looking at the instruction encoding more carefully:
	// The instruction word is: 0xC[mode][reg1][reg2]
	// For branches, the standard encoding seems to be with reg1=0, reg2=0
	// But CMP immediate would have reg1 set to the register being compared
	//
	// Let's use a heuristic: if mode >= 1 && mode <= 6, and the next word looks like a branch offset
	// (signed 16-bit), treat as branch. Otherwise, if mode == 1, treat as CMP immediate.
	//
	// Actually, simpler: check the opcode pattern. Branches are 0xC1, 0xC2, etc. with reg1=reg2=0 typically.
	// But CMP immediate is 0xC1 with reg1 set.
	//
	// For now, let's fix the immediate issue: if mode == 1 and we're not in branch context, it's CMP immediate
	// But we need to distinguish somehow. Let's check: branches always fetch an offset. CMP immediate fetches an immediate value.
	// They're the same! So we can't distinguish by what follows.
	//
	// The real solution: CMP immediate probably doesn't exist, OR it uses a different encoding.
	// Let me check the test - it uses 0xC100 which would decode as mode=1, reg1=0, reg2=0 = BEQ!
	//
	// Wait, the test writes: 0xC100 = 0xC1, 0x00 = opcode=0xC, mode=0x1, reg1=0x0, reg2=0x0
	// This is BEQ, not CMP immediate!
	//
	// So the test is wrong, OR CMP immediate uses a different encoding. Let me check if there's a CMP immediate instruction at all.
	//
	// Based on the ROM builder and docs, CMP should support mode 1 for immediate. But the encoding conflicts with BEQ.
	// Solution: CMP immediate might not be supported, OR it uses a reserved mode value, OR the instruction set needs clarification.
	//
	// For now, let's implement what makes sense: if mode == 0, CMP reg. If mode == 1 and we want CMP immediate,
	// we need a way to distinguish. Let's assume CMP immediate is not supported for now, and fix the test.
	//
	// Actually, re-reading the code comment: "CMP R1, #imm" - this suggests it should exist.
	// Let me check: maybe CMP immediate uses mode 7 or higher? Or maybe it's encoded differently?
	//
	// For the purpose of this fix, let's assume: CMP immediate uses mode 1, but we distinguish it from BEQ
	// by checking if it's actually a comparison (has a register operand). But BEQ doesn't use registers...
	//
	// I think the issue is that the instruction encoding is ambiguous. Let's use a simple rule:
	// - Mode 0: CMP reg
	// - Mode 1: Try CMP immediate first (fetch immediate), if that fails or doesn't make sense, treat as BEQ
	// But that's not deterministic.
	//
	// Better solution: Check the instruction word more carefully. If mode=1 and the instruction looks like
	// it has register operands (reg1 != 0 or reg2 != 0), it might be CMP immediate. But BEQ is 0xC100 which has reg1=0, reg2=0.
	//
	// So: if mode == 1 && (reg1 != 0 || reg2 != 0), treat as CMP immediate. Otherwise, BEQ.
	
	if mode == 1 && (reg1 != 0 || reg2 != 0) {
		// CMP R1, #imm (immediate mode) - distinguished from BEQ by having register operands
		value := c.FetchImmediate()
		c.State.Cycles++
		a := c.GetRegister(reg1)
		result := a - value
		c.UpdateFlagsWithOverflow(a, value, result, true)
		c.State.Cycles++
		return nil
	}

	// Branch instructions (mode 1-6, where mode 1 = BEQ when reg1=reg2=0)
	// Extract branch opcode: mode 1=BEQ, 2=BNE, 3=BGT, 4=BLT, 5=BGE, 6=BLE
	branchOpcode := mode
	if mode == 1 && reg1 == 0 && reg2 == 0 {
		branchOpcode = 1 // BEQ
	} else if mode >= 2 && mode <= 6 {
		branchOpcode = mode
	} else {
		// Invalid - treat as error or CMP immediate fallback
		// For now, if mode == 1 with registers, we already handled it above
		// So this shouldn't happen, but let's handle it
		return fmt.Errorf("invalid CMP/branch mode: %d (reg1=%d, reg2=%d)", mode, reg1, reg2)
	}
	
	offset := int16(c.FetchImmediate())
	c.State.Cycles++

	var shouldBranch bool
	switch branchOpcode {
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
		return fmt.Errorf("unknown branch opcode: 0x%X", branchOpcode)
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

// executeJMP executes JMP instruction
func (c *CPU) executeJMP() error {
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
}

// executeCALL executes CALL instruction
func (c *CPU) executeCALL() error {
	offset := int16(c.FetchImmediate())
	c.State.Cycles++

	// Push return address (PBR:PC) - use PCBank as authoritative
	c.Push16(uint16(c.State.PCBank))
	c.Push16(c.State.PCOffset)

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
		return fmt.Errorf("stack underflow: RET called with empty stack (SP=0x%04X)", c.State.SP)
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
