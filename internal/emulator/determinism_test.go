package emulator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"nitro-core-dx/internal/cpu"
	"nitro-core-dx/internal/debug"
)

// FrameState represents the complete state of the emulator at a frame boundary
type FrameState struct {
	// CPU state
	CPURegisters [8]uint16 // R0-R7
	PCBank       uint8
	PCOffset     uint16
	PBR          uint8
	DBR          uint8
	SP           uint16
	Flags        uint8
	CPUCycles    uint32

	// Selected WRAM ranges (or full WRAM)
	WRAMHash string // SHA256 hash of WRAM

	// Framebuffer checksum
	FramebufferHash string // SHA256 hash of framebuffer
}

// ComputeFrameState computes the state hash for the current frame
func (e *Emulator) ComputeFrameState() FrameState {
	state := FrameState{
		CPURegisters: [8]uint16{
			e.CPU.State.R0,
			e.CPU.State.R1,
			e.CPU.State.R2,
			e.CPU.State.R3,
			e.CPU.State.R4,
			e.CPU.State.R5,
			e.CPU.State.R6,
			e.CPU.State.R7,
		},
		PCBank:    e.CPU.State.PCBank,
		PCOffset:  e.CPU.State.PCOffset,
		PBR:       e.CPU.State.PBR,
		DBR:       e.CPU.State.DBR,
		SP:        e.CPU.State.SP,
		Flags:     e.CPU.State.Flags,
		CPUCycles: e.CPU.State.Cycles,
	}

	// Compute WRAM hash
	wramHash := sha256.Sum256(e.Bus.WRAM[:])
	state.WRAMHash = hex.EncodeToString(wramHash[:])

	// Compute framebuffer hash (convert uint32 to byte slice)
	fbBytes := make([]byte, len(e.PPU.OutputBuffer)*4)
	for i, pixel := range e.PPU.OutputBuffer {
		fbBytes[i*4] = byte(pixel & 0xFF)
		fbBytes[i*4+1] = byte((pixel >> 8) & 0xFF)
		fbBytes[i*4+2] = byte((pixel >> 16) & 0xFF)
		fbBytes[i*4+3] = byte((pixel >> 24) & 0xFF)
	}
	fbHash := sha256.Sum256(fbBytes)
	state.FramebufferHash = hex.EncodeToString(fbHash[:])

	return state
}

// DeterminismHarness runs a ROM for N frames with scripted input and captures per-frame hashes
type DeterminismHarness struct {
	Emulator      *Emulator
	FrameStates   []FrameState
	InputSequence []uint16 // Input button states per frame
	cycleLogPath  string
}

// NewDeterminismHarness creates a new determinism harness
func NewDeterminismHarness(useDebugMode bool) (*DeterminismHarness, error) {
	logger := debug.NewLogger(10000)

	emu := NewEmulatorWithLogger(logger)
	h := &DeterminismHarness{
		Emulator:      emu,
		FrameStates:   make([]FrameState, 0),
		InputSequence: make([]uint16, 0),
	}

	if useDebugMode {
		// Force RunFrame() into cycle-by-cycle mode via CycleLogger.IsEnabled(),
		// but avoid large log output by deferring logging start far beyond test runtime.
		tmp, err := os.CreateTemp("", "nitro-determinism-*.log")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp cycle log: %w", err)
		}
		h.cycleLogPath = tmp.Name()
		_ = tmp.Close()

		const deferredStartCycle = uint64(1 << 60)
		cycleLogger, err := debug.NewCycleLogger(
			h.cycleLogPath,
			1,
			deferredStartCycle,
			emu.Bus,
			NewOAMAdapter(emu.PPU),
			NewPPUAdapter(emu.PPU),
			NewAPUAdapter(emu.APU),
		)
		if err != nil {
			_ = os.Remove(h.cycleLogPath)
			h.cycleLogPath = ""
			return nil, fmt.Errorf("failed to create cycle logger: %w", err)
		}
		emu.CycleLogger = cycleLogger
		emu.CycleLogger.SetEnabled(true)
	}

	return h, nil
}

// Cleanup releases resources created by the harness.
func (h *DeterminismHarness) Cleanup() {
	if h == nil || h.Emulator == nil {
		return
	}
	if h.Emulator.CycleLogger != nil {
		_ = h.Emulator.CycleLogger.Close()
		h.Emulator.CycleLogger = nil
	}
	if h.Emulator.Logger != nil {
		h.Emulator.Logger.Shutdown()
	}
	if h.cycleLogPath != "" {
		_ = os.Remove(h.cycleLogPath)
		h.cycleLogPath = ""
	}
}

// RunFrames runs the emulator for N frames with scripted input
func (h *DeterminismHarness) RunFrames(numFrames int, inputSequence []uint16) error {
	h.Emulator.Start()

	for i := 0; i < numFrames; i++ {
		// Set input for this frame
		if i < len(inputSequence) {
			h.Emulator.SetInputButtons(inputSequence[i])
		} else {
			h.Emulator.SetInputButtons(0) // No input
		}

		// Run one frame
		if err := h.Emulator.RunFrame(); err != nil {
			return fmt.Errorf("frame %d error: %w", i, err)
		}

		// Capture state
		state := h.Emulator.ComputeFrameState()
		h.FrameStates = append(h.FrameStates, state)
	}

	return nil
}

// CompareStates compares two frame states and returns differences
func CompareStates(a, b FrameState) []string {
	var diffs []string

	// Compare CPU registers
	for i := 0; i < 8; i++ {
		if a.CPURegisters[i] != b.CPURegisters[i] {
			diffs = append(diffs, fmt.Sprintf("R%d: 0x%04X != 0x%04X", i, a.CPURegisters[i], b.CPURegisters[i]))
		}
	}

	// Compare PC
	if a.PCBank != b.PCBank || a.PCOffset != b.PCOffset {
		diffs = append(diffs, fmt.Sprintf("PC: %02X:%04X != %02X:%04X", a.PCBank, a.PCOffset, b.PCBank, b.PCOffset))
	}

	// Compare bank registers
	if a.PBR != b.PBR {
		diffs = append(diffs, fmt.Sprintf("PBR: 0x%02X != 0x%02X", a.PBR, b.PBR))
	}
	if a.DBR != b.DBR {
		diffs = append(diffs, fmt.Sprintf("DBR: 0x%02X != 0x%02X", a.DBR, b.DBR))
	}

	// Compare SP
	if a.SP != b.SP {
		diffs = append(diffs, fmt.Sprintf("SP: 0x%04X != 0x%04X", a.SP, b.SP))
	}

	// Compare flags
	if a.Flags != b.Flags {
		diffs = append(diffs, fmt.Sprintf("Flags: 0x%02X != 0x%02X", a.Flags, b.Flags))
	}

	// Compare WRAM hash
	if a.WRAMHash != b.WRAMHash {
		diffs = append(diffs, fmt.Sprintf("WRAM hash differs: %s != %s", a.WRAMHash[:16], b.WRAMHash[:16]))
	}

	// Compare framebuffer hash
	if a.FramebufferHash != b.FramebufferHash {
		diffs = append(diffs, fmt.Sprintf("Framebuffer hash differs: %s != %s", a.FramebufferHash[:16], b.FramebufferHash[:16]))
	}

	return diffs
}

// runDeterminismTest runs a ROM in both debug and optimized mode and compares results
// This is a helper function, not a test function itself
func runDeterminismTest(t *testing.T, romData []byte, numFrames int, inputSequence []uint16) (bool, []string) {
	// Run in debug mode (scheduler-driven)
	harnessDebug, err := NewDeterminismHarness(true)
	if err != nil {
		t.Fatalf("Failed to create debug determinism harness: %v", err)
		return false, []string{fmt.Sprintf("Failed to create debug harness: %v", err)}
	}
	defer harnessDebug.Cleanup()
	if err := harnessDebug.Emulator.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM in debug mode: %v", err)
		return false, []string{fmt.Sprintf("Failed to load ROM: %v", err)}
	}
	if err := harnessDebug.RunFrames(numFrames, inputSequence); err != nil {
		t.Fatalf("Failed to run frames in debug mode: %v", err)
		return false, []string{fmt.Sprintf("Failed to run frames: %v", err)}
	}

	// Run in optimized mode (non-logging)
	harnessOpt, err := NewDeterminismHarness(false)
	if err != nil {
		t.Fatalf("Failed to create optimized determinism harness: %v", err)
		return false, []string{fmt.Sprintf("Failed to create optimized harness: %v", err)}
	}
	defer harnessOpt.Cleanup()
	if err := harnessOpt.Emulator.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM in optimized mode: %v", err)
		return false, []string{fmt.Sprintf("Failed to load ROM: %v", err)}
	}
	if err := harnessOpt.RunFrames(numFrames, inputSequence); err != nil {
		t.Fatalf("Failed to run frames in optimized mode: %v", err)
		return false, []string{fmt.Sprintf("Failed to run frames: %v", err)}
	}

	// Compare frame-by-frame
	var allDiffs []string
	for i := 0; i < numFrames; i++ {
		if i >= len(harnessDebug.FrameStates) || i >= len(harnessOpt.FrameStates) {
			allDiffs = append(allDiffs, fmt.Sprintf("Frame %d: missing state", i))
			continue
		}

		diffs := CompareStates(harnessDebug.FrameStates[i], harnessOpt.FrameStates[i])
		if len(diffs) > 0 {
			allDiffs = append(allDiffs, fmt.Sprintf("Frame %d:", i))
			for _, diff := range diffs {
				allDiffs = append(allDiffs, fmt.Sprintf("  %s", diff))
			}
		}
	}

	return len(allDiffs) == 0, allDiffs
}

// TestDeterminism runs a simple determinism test with a minimal ROM
func TestDeterminism(t *testing.T) {
	// Create a minimal ROM: MOV R0, #0x1234; NOP; infinite loop
	// Also includes interrupt handler that just returns immediately
	romBuilder := func() []byte {
		// ROM header: 32 bytes
		// Make ROM larger: 32 bytes header + 32 bytes code (16 words)
		romData := make([]byte, 32+32)
		
		// Header
		romData[0] = 'R'
		romData[1] = 'M'
		romData[2] = 'C'
		romData[3] = 'F'
		romData[4] = 0x01 // Version
		romData[5] = 0x00
		romData[6] = 0x20 // ROM size: 32 bytes
		romData[7] = 0x00
		romData[8] = 0x00
		romData[9] = 0x00
		romData[10] = 0x01 // Entry bank: 1
		romData[11] = 0x00
		romData[12] = 0x00 // Entry offset: 0x8000
		romData[13] = 0x80
		romData[14] = 0x00 // Mapper flags
		romData[15] = 0x00
		
		// Code at 0x8000: Main loop
		// MOV R0, #0x1234 (mode 1 = immediate, reg1=0, reg2=0)
		// MOV instruction: 0x1000 | (mode<<8) | (reg1<<4) | reg2
		// = 0x1000 | (1<<8) | (0<<4) | 0 = 0x1100
		romData[32] = 0x00 // Little-endian: 0x1100 = [0x00, 0x11]
		romData[33] = 0x11
		romData[34] = 0x34 // Immediate: 0x1234 (little-endian)
		romData[35] = 0x12
		
		// NOP: 0x0000
		romData[36] = 0x00
		romData[37] = 0x00
		
		// JMP back: 0xD000 (little-endian)
		romData[38] = 0x00
		romData[39] = 0xD0
		// Offset: -4 bytes = -2 words = 0xFFFE (little-endian)
		romData[40] = 0xFE
		romData[41] = 0xFF
		
		// Main code at 0x8000: Simple infinite loop
		// MOV R0, #0x1234
		romData[32] = 0x00 // MOV instruction: 0x1100
		romData[33] = 0x11
		romData[34] = 0x34 // Immediate: 0x1234
		romData[35] = 0x12
		
		// NOP: 0x0000
		romData[36] = 0x00
		romData[37] = 0x00
		
		// NOP: 0x0000 (another NOP to pad)
		romData[38] = 0x00
		romData[39] = 0x00
		
		// JMP back to start: 0xD000 (little-endian)
		romData[40] = 0x00
		romData[41] = 0xD0
		// Offset: -6 bytes = -3 words = 0xFFFD (little-endian)
		romData[42] = 0xFD
		romData[43] = 0xFF
		
		// Fill rest with NOPs to prevent reading garbage
		for i := 44; i < 64; i += 2 {
			romData[i] = 0x00
			romData[i+1] = 0x00
		}
		
		return romData
	}()
	
	// Create harnesses and set interrupt vector to RET instruction
	harnessDebug, err := NewDeterminismHarness(true)
	if err != nil {
		t.Fatalf("Failed to create debug determinism harness: %v", err)
	}
	defer harnessDebug.Cleanup()
	if err := harnessDebug.Emulator.LoadROM(romBuilder); err != nil {
		t.Fatalf("Failed to load ROM in debug mode: %v", err)
	}
	// Set interrupt vector to point to RET instruction at 0x8008
	// Vector format: bank (1 byte) + offset_high (1 byte), offset low is always 0x00
	// So 0x8008 = bank 1, offset_high 0x80 (but that gives 0x8000, not 0x8008)
	// Actually, the vector format doesn't support arbitrary offsets - it's always 0x8000+
	// So we can't point to 0x8008 directly. Instead, let's disable interrupts by
	// setting the I flag, or we can put RET at 0x8000 and change entry point
	// For simplicity, let's disable interrupts by setting I flag in CPU
	harnessDebug.Emulator.CPU.SetFlag(cpu.FlagI, true) // Disable interrupts
	
	harnessOpt, err := NewDeterminismHarness(false)
	if err != nil {
		t.Fatalf("Failed to create optimized determinism harness: %v", err)
	}
	defer harnessOpt.Cleanup()
	if err := harnessOpt.Emulator.LoadROM(romBuilder); err != nil {
		t.Fatalf("Failed to load ROM in optimized mode: %v", err)
	}
	harnessOpt.Emulator.CPU.SetFlag(cpu.FlagI, true) // Disable interrupts
	
	// Run determinism test: 5 frames, no input
	numFrames := 5
	inputSequence := []uint16{0, 0, 0, 0, 0}
	
	// Run frames
	if err := harnessDebug.RunFrames(numFrames, inputSequence); err != nil {
		t.Fatalf("Failed to run frames in debug mode: %v", err)
	}
	if err := harnessOpt.RunFrames(numFrames, inputSequence); err != nil {
		t.Fatalf("Failed to run frames in optimized mode: %v", err)
	}
	
	// Compare results
	var allDiffs []string
	for i := 0; i < numFrames; i++ {
		if i >= len(harnessDebug.FrameStates) || i >= len(harnessOpt.FrameStates) {
			allDiffs = append(allDiffs, fmt.Sprintf("Frame %d: missing state", i))
			continue
		}
		diffs := CompareStates(harnessDebug.FrameStates[i], harnessOpt.FrameStates[i])
		if len(diffs) > 0 {
			allDiffs = append(allDiffs, fmt.Sprintf("Frame %d:", i))
			for _, diff := range diffs {
				allDiffs = append(allDiffs, fmt.Sprintf("  %s", diff))
			}
		}
	}
	
	if len(allDiffs) > 0 {
		t.Errorf("Debug and optimized modes produced different results:")
		for _, diff := range allDiffs {
			t.Errorf("  %s", diff)
		}
	}
}
