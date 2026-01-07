package cpu

import (
	"fmt"
	"nitro-core-dx/internal/debug"
)

// CPULogLevel represents granular logging levels for CPU
type CPULogLevel int

const (
	CPULogNone CPULogLevel = iota // No CPU logging
	CPULogErrors                  // Only errors
	CPULogBranches                // Branches and jumps
	CPULogMemory                  // Memory access (reads/writes)
	CPULogRegisters               // Register changes
	CPULogInstructions            // All instructions
	CPULogTrace                   // Full trace (every cycle)
)

// CPULoggerAdapter adapts the debug.Logger to the CPU's LoggerInterface
type CPULoggerAdapter struct {
	logger     *debug.Logger
	level      CPULogLevel
	enabled    bool
	lastState  CPUState // Track state changes for register logging
}

// NewCPULoggerAdapter creates a new CPU logger adapter
func NewCPULoggerAdapter(logger *debug.Logger, level CPULogLevel) *CPULoggerAdapter {
	return &CPULoggerAdapter{
		logger:  logger,
		level:   level,
		enabled: true,
	}
}

// SetLevel sets the CPU logging level
func (a *CPULoggerAdapter) SetLevel(level CPULogLevel) {
	a.level = level
}

// SetEnabled enables or disables CPU logging
func (a *CPULoggerAdapter) SetEnabled(enabled bool) {
	a.enabled = enabled
}

// LogCPU implements LoggerInterface.LogCPU
func (a *CPULoggerAdapter) LogCPU(instruction uint16, state CPUState, cycles uint32) {
	if !a.enabled || a.logger == nil || a.level == CPULogNone {
		return
	}

	// Decode instruction for logging
	opcode := uint8((instruction >> 12) & 0xF)
	mode := uint8((instruction >> 8) & 0xF)
	reg1 := uint8((instruction >> 4) & 0xF)
	reg2 := uint8(instruction & 0xF)

	// Determine log level based on CPU log level
	var logLevel debug.LogLevel
	var message string
	var data map[string]interface{}

	switch a.level {
	case CPULogErrors:
		// Only log errors (handled separately)
		return

	case CPULogBranches:
		// Only log branches and jumps
		if opcode == 0xC || opcode == 0xD || opcode == 0xE || opcode == 0xF {
			logLevel = debug.LogLevelInfo
			message = a.formatInstruction(instruction, opcode, mode, reg1, reg2)
			data = a.getStateData(state, cycles)
		} else {
			return
		}

	case CPULogMemory:
		// Log memory access and branches
		if opcode == 0x1 && (mode == 2 || mode == 3) { // MOV with memory
			logLevel = debug.LogLevelInfo
			message = a.formatInstruction(instruction, opcode, mode, reg1, reg2)
			data = a.getStateData(state, cycles)
			if mode == 2 {
				data["memory_op"] = "read"
				data["address"] = fmt.Sprintf("%02X:%04X", state.DBR, state.R0) // Approximate
			} else {
				data["memory_op"] = "write"
				data["address"] = fmt.Sprintf("%02X:%04X", state.DBR, state.R0) // Approximate
			}
		} else if opcode == 0xC || opcode == 0xD || opcode == 0xE || opcode == 0xF {
			logLevel = debug.LogLevelInfo
			message = a.formatInstruction(instruction, opcode, mode, reg1, reg2)
			data = a.getStateData(state, cycles)
		} else {
			return
		}

	case CPULogRegisters:
		// Log register changes and branches
		regChanged := a.detectRegisterChange(state)
		if regChanged || opcode == 0xC || opcode == 0xD || opcode == 0xE || opcode == 0xF {
			logLevel = debug.LogLevelInfo
			message = a.formatInstruction(instruction, opcode, mode, reg1, reg2)
			data = a.getStateData(state, cycles)
			if regChanged {
				data["registers_changed"] = true
			}
		} else {
			return
		}

	case CPULogInstructions:
		// Log all instructions
		logLevel = debug.LogLevelDebug
		message = a.formatInstruction(instruction, opcode, mode, reg1, reg2)
		data = a.getStateData(state, cycles)

	case CPULogTrace:
		// Full trace
		logLevel = debug.LogLevelTrace
		message = a.formatInstruction(instruction, opcode, mode, reg1, reg2)
		data = a.getStateData(state, cycles)
		data["trace"] = true
	}

	// Update last state for register change detection
	a.lastState = state

	// Log the message
	a.logger.LogCPU(logLevel, message, data)
}

// formatInstruction formats an instruction for logging
func (a *CPULoggerAdapter) formatInstruction(instruction uint16, opcode, mode, reg1, reg2 uint8) string {
	opcodeNames := map[uint8]string{
		0x0: "NOP",
		0x1: "MOV",
		0x2: "ADD",
		0x3: "SUB",
		0x4: "MUL",
		0x5: "DIV",
		0x6: "AND",
		0x7: "OR",
		0x8: "XOR",
		0x9: "NOT",
		0xA: "SHL",
		0xB: "SHR",
		0xC: "CMP/BR",
		0xD: "JMP",
		0xE: "CALL",
		0xF: "RET",
	}

	opName := opcodeNames[opcode]
	if opName == "" {
		opName = fmt.Sprintf("OP%X", opcode)
	}

	pc := fmt.Sprintf("%02X:%04X", a.lastState.PCBank, a.lastState.PCOffset)
	return fmt.Sprintf("%s %s (0x%04X) @ %s", opName, a.formatOperands(opcode, mode, reg1, reg2), instruction, pc)
}

// formatOperands formats instruction operands
func (a *CPULoggerAdapter) formatOperands(opcode, mode, reg1, reg2 uint8) string {
	switch opcode {
	case 0x1: // MOV
		modeNames := map[uint8]string{
			0: fmt.Sprintf("R%d, R%d", reg1, reg2),
			1: fmt.Sprintf("R%d, #imm", reg1),
			2: fmt.Sprintf("R%d, [R%d]", reg1, reg2),
			3: fmt.Sprintf("[R%d], R%d", reg1, reg2),
			4: fmt.Sprintf("PUSH R%d", reg1),
			5: fmt.Sprintf("POP R%d", reg1),
		}
		if name, ok := modeNames[mode]; ok {
			return name
		}
		return fmt.Sprintf("mode %d", mode)
	default:
		return fmt.Sprintf("R%d, R%d", reg1, reg2)
	}
}

// getStateData extracts state data for logging
func (a *CPULoggerAdapter) getStateData(state CPUState, cycles uint32) map[string]interface{} {
	return map[string]interface{}{
		"pc":      fmt.Sprintf("%02X:%04X", state.PCBank, state.PCOffset),
		"cycles":  cycles,
		"r0":      state.R0,
		"r1":      state.R1,
		"r2":      state.R2,
		"r3":      state.R3,
		"r4":      state.R4,
		"r5":      state.R5,
		"r6":      state.R6,
		"r7":      state.R7,
		"sp":      state.SP,
		"flags":   fmt.Sprintf("%08b", state.Flags),
	}
}

// detectRegisterChange detects if any register changed
func (a *CPULoggerAdapter) detectRegisterChange(state CPUState) bool {
	return state.R0 != a.lastState.R0 ||
		state.R1 != a.lastState.R1 ||
		state.R2 != a.lastState.R2 ||
		state.R3 != a.lastState.R3 ||
		state.R4 != a.lastState.R4 ||
		state.R5 != a.lastState.R5 ||
		state.R6 != a.lastState.R6 ||
		state.R7 != a.lastState.R7 ||
		state.SP != a.lastState.SP ||
		state.Flags != a.lastState.Flags
}



