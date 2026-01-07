package emulator

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"nitro-core-dx/internal/apu"
	"nitro-core-dx/internal/cpu"
	"nitro-core-dx/internal/ppu"
)

func init() {
	// Register types with gob for serialization
	gob.Register(PPUState{})
	gob.Register(APUState{})
	gob.Register(MemoryState{})
	gob.Register(InputState{})
	gob.Register(SaveState{})
	gob.Register(cpu.CPUState{})
	gob.Register(ppu.BackgroundLayer{})
	gob.Register(ppu.Window{})
	gob.Register(apu.AudioChannel{})
}

// SaveState represents a complete emulator state snapshot
type SaveState struct {
	// Version for compatibility checking
	Version uint16

	// CPU state
	CPUState cpu.CPUState

	// PPU state (excluding output buffer and debug fields)
	PPUState PPUState

	// APU state
	APUState APUState

	// Memory state (WRAM and Extended WRAM, but not ROM)
	MemoryState MemoryState

	// Input state
	InputState InputState

	// Emulator state
	Running bool
	Paused  bool
}

// PPUState represents PPU state for save/load
type PPUState struct {
	VRAM            [65536]uint8
	CGRAM           [512]uint8
	OAM             [768]uint8
	BG0, BG1, BG2, BG3 ppu.BackgroundLayer
	MatrixEnabled   bool
	MatrixA, MatrixB, MatrixC, MatrixD int16
	MatrixCenterX, MatrixCenterY int16
	MatrixMirrorH, MatrixMirrorV bool
	Window0, Window1 ppu.Window
	WindowControl   uint8
	WindowMainEnable uint8
	WindowSubEnable  uint8
	HDMAEnabled     bool
	HDMATableBase   uint16
	FrameCounter    uint16
	VBlankFlag      bool
	VRAMAddr        uint16
	CGRAMAddr       uint8
	CGRAMWriteLatch bool
	CGRAMWriteValue uint16
	OAMAddr         uint8
	OAMByteIndex    uint8
}

// APUState represents APU state for save/load
type APUState struct {
	Channels                [4]apu.AudioChannel
	MasterVolume            uint8
	ChannelCompletionStatus uint8
}

// MemoryState represents Memory state for save/load
type MemoryState struct {
	WRAM         [32768]uint8
	WRAMExtended [131072]uint8
}

// InputState represents Input state for save/load
type InputState struct {
	Controller1Buttons uint16
	Controller2Buttons uint16
	LatchActive        bool
	Controller2LatchActive bool
}

// SaveState saves the current emulator state to a byte slice
func (e *Emulator) SaveState() ([]byte, error) {
	// Create save state structure
	state := SaveState{
		Version: 1, // Version 1 of save state format
		CPUState: e.CPU.State,
		PPUState: e.savePPUState(),
		APUState: e.saveAPUState(),
		MemoryState: e.saveMemoryState(),
		InputState: e.saveInputState(),
		Running: e.Running,
		Paused:  e.Paused,
	}

	// Serialize using gob
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(state); err != nil {
		return nil, fmt.Errorf("failed to encode save state: %w", err)
	}

	return buf.Bytes(), nil
}

// LoadState loads an emulator state from a byte slice
func (e *Emulator) LoadState(data []byte) error {
	// Deserialize using gob
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	
	var state SaveState
	if err := decoder.Decode(&state); err != nil {
		return fmt.Errorf("failed to decode save state: %w", err)
	}

	// Check version compatibility
	if state.Version != 1 {
		return fmt.Errorf("unsupported save state version: %d (expected 1)", state.Version)
	}

	// Restore state
	e.CPU.State = state.CPUState
	e.loadPPUState(state.PPUState)
	e.loadAPUState(state.APUState)
	e.loadMemoryState(state.MemoryState)
	e.loadInputState(state.InputState)
	e.Running = state.Running
	e.Paused = state.Paused

	return nil
}

// savePPUState extracts PPU state for saving
func (e *Emulator) savePPUState() PPUState {
	return PPUState{
		VRAM:            e.PPU.VRAM,
		CGRAM:           e.PPU.CGRAM,
		OAM:             e.PPU.OAM,
		BG0:             e.PPU.BG0,
		BG1:             e.PPU.BG1,
		BG2:             e.PPU.BG2,
		BG3:             e.PPU.BG3,
		MatrixEnabled:   e.PPU.MatrixEnabled,
		MatrixA:         e.PPU.MatrixA,
		MatrixB:         e.PPU.MatrixB,
		MatrixC:         e.PPU.MatrixC,
		MatrixD:         e.PPU.MatrixD,
		MatrixCenterX:   e.PPU.MatrixCenterX,
		MatrixCenterY:   e.PPU.MatrixCenterY,
		MatrixMirrorH:   e.PPU.MatrixMirrorH,
		MatrixMirrorV:   e.PPU.MatrixMirrorV,
		Window0:         e.PPU.Window0,
		Window1:         e.PPU.Window1,
		WindowControl:   e.PPU.WindowControl,
		WindowMainEnable: e.PPU.WindowMainEnable,
		WindowSubEnable: e.PPU.WindowSubEnable,
		HDMAEnabled:     e.PPU.HDMAEnabled,
		HDMATableBase:   e.PPU.HDMATableBase,
		FrameCounter:    e.PPU.FrameCounter,
		VBlankFlag:      e.PPU.VBlankFlag,
		VRAMAddr:        e.PPU.VRAMAddr,
		CGRAMAddr:       e.PPU.CGRAMAddr,
		CGRAMWriteLatch: e.PPU.CGRAMWriteLatch,
		CGRAMWriteValue: e.PPU.CGRAMWriteValue,
		OAMAddr:         e.PPU.OAMAddr,
		OAMByteIndex:    e.PPU.OAMByteIndex,
	}
}

// loadPPUState restores PPU state from saved state
func (e *Emulator) loadPPUState(state PPUState) {
	e.PPU.VRAM = state.VRAM
	e.PPU.CGRAM = state.CGRAM
	e.PPU.OAM = state.OAM
	e.PPU.BG0 = state.BG0
	e.PPU.BG1 = state.BG1
	e.PPU.BG2 = state.BG2
	e.PPU.BG3 = state.BG3
	e.PPU.MatrixEnabled = state.MatrixEnabled
	e.PPU.MatrixA = state.MatrixA
	e.PPU.MatrixB = state.MatrixB
	e.PPU.MatrixC = state.MatrixC
	e.PPU.MatrixD = state.MatrixD
	e.PPU.MatrixCenterX = state.MatrixCenterX
	e.PPU.MatrixCenterY = state.MatrixCenterY
	e.PPU.MatrixMirrorH = state.MatrixMirrorH
	e.PPU.MatrixMirrorV = state.MatrixMirrorV
	e.PPU.Window0 = state.Window0
	e.PPU.Window1 = state.Window1
	e.PPU.WindowControl = state.WindowControl
	e.PPU.WindowMainEnable = state.WindowMainEnable
	e.PPU.WindowSubEnable = state.WindowSubEnable
	e.PPU.HDMAEnabled = state.HDMAEnabled
	e.PPU.HDMATableBase = state.HDMATableBase
	e.PPU.FrameCounter = state.FrameCounter
	e.PPU.VBlankFlag = state.VBlankFlag
	e.PPU.VRAMAddr = state.VRAMAddr
	e.PPU.CGRAMAddr = state.CGRAMAddr
	e.PPU.CGRAMWriteLatch = state.CGRAMWriteLatch
	e.PPU.CGRAMWriteValue = state.CGRAMWriteValue
	e.PPU.OAMAddr = state.OAMAddr
	e.PPU.OAMByteIndex = state.OAMByteIndex
}

// saveAPUState extracts APU state for saving
func (e *Emulator) saveAPUState() APUState {
	return APUState{
		Channels:                e.APU.Channels,
		MasterVolume:           e.APU.MasterVolume,
		ChannelCompletionStatus: e.APU.ChannelCompletionStatus,
	}
}

// loadAPUState restores APU state from saved state
func (e *Emulator) loadAPUState(state APUState) {
	e.APU.Channels = state.Channels
	e.APU.MasterVolume = state.MasterVolume
	e.APU.ChannelCompletionStatus = state.ChannelCompletionStatus
}

// saveMemoryState extracts Memory state for saving
func (e *Emulator) saveMemoryState() MemoryState {
	return MemoryState{
		WRAM:         e.Memory.WRAM,
		WRAMExtended: e.Memory.WRAMExtended,
	}
}

// loadMemoryState restores Memory state from saved state
func (e *Emulator) loadMemoryState(state MemoryState) {
	e.Memory.WRAM = state.WRAM
	e.Memory.WRAMExtended = state.WRAMExtended
}

// saveInputState extracts Input state for saving
func (e *Emulator) saveInputState() InputState {
	return InputState{
		Controller1Buttons:     e.Input.Controller1Buttons,
		Controller2Buttons:     e.Input.Controller2Buttons,
		LatchActive:             e.Input.LatchActive,
		Controller2LatchActive: e.Input.Controller2LatchActive,
	}
}

// loadInputState restores Input state from saved state
func (e *Emulator) loadInputState(state InputState) {
	e.Input.Controller1Buttons = state.Controller1Buttons
	e.Input.Controller2Buttons = state.Controller2Buttons
	e.Input.LatchActive = state.LatchActive
	e.Input.Controller2LatchActive = state.Controller2LatchActive
}

// SaveStateToFile saves the current emulator state to a file
// TODO: Implement file writing - for now, use SaveState() and write to file manually
func (e *Emulator) SaveStateToFile(filename string) error {
	_, err := e.SaveState()
	if err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// TODO: Write data to file using os.WriteFile
	_ = filename
	return fmt.Errorf("SaveStateToFile not yet implemented - use SaveState() and write to file manually")
}

// LoadStateFromFile loads an emulator state from a file
func (e *Emulator) LoadStateFromFile(filename string) error {
	// Read from file (we'll need to import os)
	// For now, return error - caller should read file and call LoadState()
	_ = filename // TODO: Implement file reading when needed
	return fmt.Errorf("LoadStateFromFile not yet implemented - read file and use LoadState() manually")
}

