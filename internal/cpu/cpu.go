package cpu

import (
	"fmt"
)

// CPUState represents the complete state of the CPU
type CPUState struct {
	// General purpose registers
	R0, R1, R2, R3, R4, R5, R6, R7 uint16

	// Program Counter (24-bit logical address)
	PCBank   uint8
	PCOffset uint16

	// Bank registers
	PBR uint8 // Program Bank Register
	DBR uint8 // Data Bank Register

	// Stack Pointer
	SP uint16 // Stack Pointer (offset in bank 0)

	// Flags register (Z, N, C, V, I)
	Flags uint8

	// Cycle counter
	Cycles uint32

	// Interrupt state
	InterruptMask   uint8
	InterruptPending uint8
}

// Flag bits
const (
	FlagZ = 0 // Zero
	FlagN = 1 // Negative
	FlagC = 2 // Carry
	FlagV = 3 // Overflow
	FlagI = 4 // Interrupt mask
)

// CPU represents the emulated CPU
type CPU struct {
	State CPUState
	Mem   MemoryInterface
	Log   LoggerInterface
}

// MemoryInterface defines the interface for memory access
type MemoryInterface interface {
	Read8(bank uint8, offset uint16) uint8
	Write8(bank uint8, offset uint16, value uint8)
	Read16(bank uint8, offset uint16) uint16
	Write16(bank uint8, offset uint16, value uint16)
}

// LoggerInterface defines the interface for logging
type LoggerInterface interface {
	LogCPU(instruction uint16, state CPUState, cycles uint32)
}

// NewCPU creates a new CPU instance
func NewCPU(mem MemoryInterface, log LoggerInterface) *CPU {
	cpu := &CPU{
		Mem: mem,
		Log: log,
	}
	cpu.Reset()
	return cpu
}

// Reset resets the CPU to initial state
func (c *CPU) Reset() {
	c.State.R0 = 0
	c.State.R1 = 0
	c.State.R2 = 0
	c.State.R3 = 0
	c.State.R4 = 0
	c.State.R5 = 0
	c.State.R6 = 0
	c.State.R7 = 0
	c.State.PCBank = 0
	c.State.PCOffset = 0
	c.State.PBR = 0
	c.State.DBR = 0
	c.State.SP = 0x1FFF // Stack starts at top of WRAM
	c.State.Flags = 0
	c.State.Cycles = 0
	c.State.InterruptMask = 0
	c.State.InterruptPending = 0
}

// SetEntryPoint sets the CPU entry point
func (c *CPU) SetEntryPoint(bank uint8, offset uint16) {
	c.State.PCBank = bank
	c.State.PCOffset = offset
	c.State.PBR = bank
}

// GetRegister returns the value of a general-purpose register
func (c *CPU) GetRegister(reg uint8) uint16 {
	switch reg {
	case 0:
		return c.State.R0
	case 1:
		return c.State.R1
	case 2:
		return c.State.R2
	case 3:
		return c.State.R3
	case 4:
		return c.State.R4
	case 5:
		return c.State.R5
	case 6:
		return c.State.R6
	case 7:
		return c.State.R7
	default:
		return 0
	}
}

// SetRegister sets the value of a general-purpose register
func (c *CPU) SetRegister(reg uint8, value uint16) {
	switch reg {
	case 0:
		c.State.R0 = value
	case 1:
		c.State.R1 = value
	case 2:
		c.State.R2 = value
	case 3:
		c.State.R3 = value
	case 4:
		c.State.R4 = value
	case 5:
		c.State.R5 = value
	case 6:
		c.State.R6 = value
	case 7:
		c.State.R7 = value
	}
}

// GetFlag returns the value of a flag
func (c *CPU) GetFlag(flag uint8) bool {
	return (c.State.Flags & (1 << flag)) != 0
}

// SetFlag sets a flag
func (c *CPU) SetFlag(flag uint8, value bool) {
	if value {
		c.State.Flags |= (1 << flag)
	} else {
		c.State.Flags &^= (1 << flag)
	}
}

// UpdateFlags updates Z and N flags based on a 16-bit result
func (c *CPU) UpdateFlags(result uint16) {
	c.SetFlag(FlagZ, result == 0)
	c.SetFlag(FlagN, (result&0x8000) != 0)
}

// UpdateFlagsWithCarry updates Z, N, and C flags
func (c *CPU) UpdateFlagsWithCarry(result uint16, carry bool) {
	c.UpdateFlags(result)
	c.SetFlag(FlagC, carry)
}

// UpdateFlagsWithOverflow updates Z, N, C, and V flags for signed arithmetic
func (c *CPU) UpdateFlagsWithOverflow(a, b, result uint16, isSubtract bool) {
	c.UpdateFlags(result)
	
	// Calculate carry (unsigned overflow)
	var carry bool
	if isSubtract {
		carry = a >= b
	} else {
		carry = result < a || result < b
	}
	c.SetFlag(FlagC, carry)
	
	// Calculate overflow (signed overflow)
	aSigned := int16(a)
	bSigned := int16(b)
	resultSigned := int16(result)
	var overflow bool
	if isSubtract {
		overflow = (aSigned < 0 && bSigned > 0 && resultSigned >= 0) ||
			(aSigned >= 0 && bSigned < 0 && resultSigned < 0)
	} else {
		overflow = (aSigned >= 0 && bSigned >= 0 && resultSigned < 0) ||
			(aSigned < 0 && bSigned < 0 && resultSigned >= 0)
	}
	c.SetFlag(FlagV, overflow)
}

// FetchInstruction fetches the next instruction from memory
func (c *CPU) FetchInstruction() uint16 {
	// Read instruction from [PBR:PC]
	low := c.Mem.Read8(c.State.PBR, c.State.PCOffset)
	high := c.Mem.Read8(c.State.PBR, c.State.PCOffset+1)
	c.State.PCOffset += 2
	c.State.Cycles++
	return uint16(low) | (uint16(high) << 8)
}

// FetchImmediate fetches a 16-bit immediate value
func (c *CPU) FetchImmediate() uint16 {
	low := c.Mem.Read8(c.State.PBR, c.State.PCOffset)
	high := c.Mem.Read8(c.State.PBR, c.State.PCOffset+1)
	c.State.PCOffset += 2
	c.State.Cycles++
	return uint16(low) | (uint16(high) << 8)
}

// ExecuteInstruction executes a single instruction
func (c *CPU) ExecuteInstruction() error {
	// Fetch instruction
	instruction := c.FetchInstruction()
	
	// Decode instruction
	opcode := uint8((instruction >> 12) & 0xF)
	mode := uint8((instruction >> 8) & 0xF)
	reg1 := uint8((instruction >> 4) & 0xF)
	reg2 := uint8(instruction & 0xF)
	
	// Log instruction if logger is available
	if c.Log != nil {
		c.Log.LogCPU(instruction, c.State, 1)
	}
	
	// Execute based on opcode
	switch opcode {
	case 0x0: // NOP
		return c.executeNOP()
	case 0x1: // MOV
		return c.executeMOV(mode, reg1, reg2)
	case 0x2: // ADD
		return c.executeADD(mode, reg1, reg2)
	case 0x3: // SUB
		return c.executeSUB(mode, reg1, reg2)
	case 0x4: // MUL
		return c.executeMUL(mode, reg1, reg2)
	case 0x5: // DIV
		return c.executeDIV(mode, reg1, reg2)
	case 0x6: // AND
		return c.executeAND(mode, reg1, reg2)
	case 0x7: // OR
		return c.executeOR(mode, reg1, reg2)
	case 0x8: // XOR
		return c.executeXOR(mode, reg1, reg2)
	case 0x9: // NOT
		return c.executeNOT(reg1)
	case 0xA: // SHL
		return c.executeSHL(mode, reg1, reg2)
	case 0xB: // SHR
		return c.executeSHR(mode, reg1, reg2)
	case 0xC: // CMP and branches
		return c.executeCMPAndBranches(mode, reg1, reg2)
	case 0xD: // JMP
		return c.executeJMP()
	case 0xE: // CALL
		return c.executeCALL()
	case 0xF: // RET
		return c.executeRET()
	default:
		return fmt.Errorf("unknown opcode: 0x%X", opcode)
	}
}

// ExecuteCycles executes CPU cycles until target cycles are reached
func (c *CPU) ExecuteCycles(targetCycles uint32) error {
	for c.State.Cycles < targetCycles {
		if err := c.ExecuteInstruction(); err != nil {
			return err
		}
		
		// Check for interrupts
		if c.State.InterruptPending != 0 && !c.GetFlag(FlagI) {
			// TODO: Handle interrupt
		}
	}
	return nil
}

// GetPC returns the current PC as a string (bank:offset)
func (c *CPU) GetPC() string {
	return fmt.Sprintf("%02X:%04X", c.State.PCBank, c.State.PCOffset)
}


