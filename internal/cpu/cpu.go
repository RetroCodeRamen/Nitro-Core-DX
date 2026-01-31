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
	FlagD = 5 // Division by zero (set when division by zero occurs)
)

// Interrupt types
const (
	INT_NONE   = 0 // No interrupt
	INT_VBLANK = 1 // VBlank interrupt (IRQ)
	INT_TIMER  = 2 // Timer interrupt (IRQ) - future
	INT_NMI    = 3 // Non-maskable interrupt
)

// Interrupt vector addresses (in bank 0)
const (
	VectorIRQ  = 0xFFE0 // IRQ vector (2 bytes: bank, offset)
	VectorNMI  = 0xFFE2 // NMI vector (2 bytes: bank, offset)
	VectorRESET = 0xFFE4 // Reset vector (2 bytes: bank, offset) - for future use
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
// NOTE: PCBank, PCOffset, and PBR are NOT reset here - they should be set
// by SetEntryPoint() in the emulator. This prevents corruption if Reset()
// is called after a ROM is loaded.
func (c *CPU) Reset() {
	c.State.R0 = 0
	c.State.R1 = 0
	c.State.R2 = 0
	c.State.R3 = 0
	c.State.R4 = 0
	c.State.R5 = 0
	c.State.R6 = 0
	c.State.R7 = 0
	// DO NOT reset PCBank, PCOffset, PBR - these are set by SetEntryPoint()
	// c.State.PCBank = 0
	// c.State.PCOffset = 0
	// c.State.PBR = 0
	c.State.DBR = 0
	c.State.SP = 0x1FFF // Stack starts at top of WRAM
	c.State.Flags = 0
	c.State.Cycles = 0
	c.State.InterruptMask = 0
	c.State.InterruptPending = 0
}

// SetEntryPoint sets the CPU entry point
func (c *CPU) SetEntryPoint(bank uint8, offset uint16) {
	// Validate entry point
	if bank == 0 {
		// This is an error - ROM should never be in bank 0
		// Bank 0 is WRAM/I/O space, ROM should be in bank 1+
		// But we'll allow it for now and let the safety check catch it
	}
	if offset < 0x8000 {
		// ROM code should start at 0x8000+ in the bank
		// But we'll allow it and let the safety check catch it
	}
	
	c.State.PCBank = bank
	c.State.PCOffset = offset &^ 1 // Ensure 16-bit alignment
	c.State.PBR = bank
	
	// Verify it was set correctly
	if c.State.PCBank != bank {
		panic(fmt.Sprintf("SetEntryPoint failed: PCBank is %d, expected %d", c.State.PCBank, bank))
	}
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
	// Ensure PC is aligned (instructions are 16-bit)
	c.State.PCOffset &^= 1
	
	// Use PCBank for instruction fetch (PBR should match, but PCBank is authoritative)
	// PCBank and PBR should always be in sync. They are updated together in:
	// - SetEntryPoint: sets both to entry bank
	// - Interrupt handling: sets both to vector bank
	// - RET instruction: pops PBR from stack, then syncs PCBank to PBR
	// The defensive sync below is a safety measure in case they somehow get out of sync
	// (e.g., due to a bug or future instruction that modifies one but not the other)
	bank := c.State.PCBank
	if c.State.PBR != c.State.PCBank {
		// Sync PBR to PCBank if they're out of sync (defensive safety measure)
		c.State.PBR = c.State.PCBank
	}
	
	// Safety check: If PCBank is 0 and offset is >= 0x8000, this is I/O space, not ROM!
	// This indicates a serious bug - PC should never be in I/O space for instruction fetch
	// This likely means PCBank was incorrectly set to 0, or there's a bug in bank switching
	// ROM code should always be in bank 1+ (banks 1-125 are ROM space)
	if bank == 0 && c.State.PCOffset >= 0x8000 {
		// This is a critical error - trying to execute from I/O space
		// This should never happen - ROM code should be in bank 1+
		// The instruction word we're about to read will be garbage from I/O registers
		// Return a NOP to prevent crash, but this indicates PCBank is wrong
		// The real issue is that PCBank should not be 0 when executing ROM code
		return 0x0000 // NOP - but this is wrong, PCBank should be 1+
	}
	
	// Read instruction from [bank:PCOffset]
	low := c.Mem.Read8(bank, c.State.PCOffset)
	high := c.Mem.Read8(bank, c.State.PCOffset+1)
	
	// Construct instruction word (little-endian: low byte first, then high byte)
	instruction := uint16(low) | (uint16(high) << 8)
	
	c.State.PCOffset += 2
	c.State.Cycles++
	return instruction
}

// FetchImmediate fetches a 16-bit immediate value
func (c *CPU) FetchImmediate() uint16 {
	// Ensure PC is aligned (immediates are 16-bit)
	c.State.PCOffset &^= 1
	
	// Use PCBank for immediate fetch (same as instruction fetch)
	// PCBank and PBR should always be in sync (see FetchInstruction for details)
	bank := c.State.PCBank
	if c.State.PBR != c.State.PCBank {
		// Sync PBR to PCBank if they're out of sync (defensive safety measure)
		c.State.PBR = c.State.PCBank
	}
	
	low := c.Mem.Read8(bank, c.State.PCOffset)
	high := c.Mem.Read8(bank, c.State.PCOffset+1)
	c.State.PCOffset += 2
	c.State.Cycles++
	return uint16(low) | (uint16(high) << 8)
}

// ExecuteInstruction executes a single instruction
func (c *CPU) ExecuteInstruction() error {
	// Safety check: If PCBank is 0 and we're in I/O space, this is a critical error
	// This should never happen - ROM code should be in bank 1+
	if c.State.PCBank == 0 && c.State.PCOffset >= 0x8000 {
		return fmt.Errorf("CRITICAL: Attempting to execute from I/O space (bank 0, offset 0x%04X). PCBank should be 1+ for ROM execution. Current state: PCBank=%d, PCOffset=0x%04X, PBR=%d. This indicates PCBank was incorrectly set to 0 or Reset() was called after LoadROM", 
			c.State.PCOffset, c.State.PCBank, c.State.PCOffset, c.State.PBR)
	}
	
	// Safety check: If PCBank is 1+ but PCOffset is < 0x8000, this is invalid
	// ROM code must be at offset 0x8000+ within a bank
	if c.State.PCBank >= 1 && c.State.PCBank <= 125 && c.State.PCOffset < 0x8000 {
		return fmt.Errorf("CRITICAL: Attempting to execute from invalid ROM address (bank %d, offset 0x%04X). ROM code must be at offset 0x8000+ within a bank. This indicates PC was corrupted or an invalid jump occurred", 
			c.State.PCBank, c.State.PCOffset)
	}
	
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
		if err := c.executeMOV(mode, reg1, reg2); err != nil {
			// Calculate the PC where this instruction was fetched from
			fetchPC := c.State.PCOffset - 2
			return fmt.Errorf("%s (instruction: 0x%04X, mode: %d, reg1: %d, reg2: %d, PC: %02X:%04X, PBR: %02X)", 
				err, instruction, mode, reg1, reg2, c.State.PCBank, fetchPC, c.State.PBR)
		}
		return nil
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
		
		// Check for interrupts (at end of instruction)
		if c.State.InterruptPending != 0 {
			// NMI is non-maskable (always handled)
			// IRQ is maskable (only if I flag is clear)
			if c.State.InterruptPending == INT_NMI || !c.GetFlag(FlagI) {
				if err := c.handleInterrupt(c.State.InterruptPending); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// StepCPU steps the CPU by a number of cycles (for clock-driven operation)
// This is called by the clock scheduler
func (c *CPU) StepCPU(cycles uint64) error {
	targetCycles := c.State.Cycles + uint32(cycles)
	return c.ExecuteCycles(targetCycles)
}

// handleInterrupt handles an interrupt
// Saves current state to stack and jumps to interrupt vector
func (c *CPU) handleInterrupt(interruptType uint8) error {
	// Determine vector address based on interrupt type
	var vectorAddr uint16
	switch interruptType {
	case INT_VBLANK, INT_TIMER:
		vectorAddr = VectorIRQ
	case INT_NMI:
		vectorAddr = VectorNMI
	default:
		return fmt.Errorf("unknown interrupt type: %d", interruptType)
	}

	// Save current PC to stack (PBR first, then PC)
	c.Push16(uint16(c.State.PBR))
	c.Push16(c.State.PCOffset)

	// Save flags to stack
	c.Push16(uint16(c.State.Flags))

	// Set I flag (disable interrupts)
	c.SetFlag(FlagI, true)

	// Read interrupt vector from memory (bank 0)
	// Vector is 2 bytes: bank (1 byte), offset_high (1 byte)
	// Offset low byte is always 0x00 (ROM addresses start at 0x8000+)
	vectorBank := uint8(c.Mem.Read8(0, vectorAddr))
	vectorOffsetHigh := uint8(c.Mem.Read8(0, vectorAddr+1))
	vectorOffset := uint16(vectorOffsetHigh) << 8 // Offset low byte is 0x00

	// Validate vector
	if vectorBank == 0 {
		// Invalid vector - don't jump (prevents crashes)
		return fmt.Errorf("invalid interrupt vector: bank is 0 (vector at 0x%04X)", vectorAddr)
	}
	if vectorOffset < 0x8000 {
		// Invalid vector - don't jump (prevents crashes)
		return fmt.Errorf("invalid interrupt vector: offset 0x%04X < 0x8000 (vector at 0x%04X)", vectorOffset, vectorAddr)
	}

	// Jump to interrupt vector
	c.State.PBR = vectorBank
	c.State.PCOffset = vectorOffset
	c.State.PCBank = vectorBank

	// Clear interrupt pending flag
	c.State.InterruptPending = INT_NONE

	// Interrupt handling takes cycles
	c.State.Cycles += 7 // Interrupt overhead (save state + jump)

	return nil
}

// TriggerInterrupt triggers an interrupt
// interruptType: INT_VBLANK, INT_TIMER, or INT_NMI
func (c *CPU) TriggerInterrupt(interruptType uint8) {
	if interruptType == INT_NONE {
		return
	}
	// Set interrupt pending (will be handled at end of current instruction)
	c.State.InterruptPending = interruptType
}

// GetPC returns the current PC as a string (bank:offset)
func (c *CPU) GetPC() string {
	return fmt.Sprintf("%02X:%04X", c.State.PCBank, c.State.PCOffset)
}


