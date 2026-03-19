package emulator

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"

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
	gob.Register(ppu.MatrixPlane{})
	gob.Register(ppu.MatrixPlaneRowParams{})
	gob.Register(ppu.TransformChannel{})
	gob.Register(ppu.Window{})
	gob.Register(apu.AudioChannel{})
}

const (
	saveStateVersion1 uint16 = 1
	saveStateVersion2 uint16 = 2
)

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
	VRAM                   [65536]uint8
	CGRAM                  [512]uint8
	OAM                    [768]uint8
	BG0, BG1, BG2, BG3     ppu.BackgroundLayer
	TransformChannels      [4]ppu.TransformChannel
	MatrixPlanes           [ppu.NumTransformChannels]ppu.MatrixPlane
	Window0, Window1       ppu.Window
	WindowControl          uint8
	WindowMainEnable       uint8
	WindowSubEnable        uint8
	HDMAEnabled            bool
	HDMATableBase          uint16
	HDMAControl            uint8
	HDMAExtControl         uint8
	FrameCounter           uint16
	VBlankFlag             bool
	VRAMAddr               uint16
	CGRAMAddr              uint8
	CGRAMWriteLatch        bool
	CGRAMWriteValue        uint16
	MatrixPlaneSelect      uint8
	MatrixPlaneAddr        uint16
	MatrixPlanePatternAddr uint16
	MatrixPlaneBitmapAddr  uint32
	MatrixPlaneRowAddr     uint16
	DMAEnabled             bool
	DMASourceBank          uint8
	DMASourceOffset        uint16
	DMADestType            uint8
	DMADestAddr            uint16
	DMALength              uint16
	DMAMode                uint8
	DMACycles              uint16
	DMAProgress            uint16
	DMACurrentSrc          uint16
	DMACurrentDest         uint16
	DMAFillValue           uint8
	OAMAddr                uint8
	OAMByteIndex           uint8
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
	Controller1Buttons    uint16
	Controller2Buttons    uint16
	Controller1Latched    uint16
	Controller2Latched    uint16
	Controller1LatchState bool
	Controller2LatchState bool
}

// SaveState saves the current emulator state to a byte slice
func (e *Emulator) SaveState() ([]byte, error) {
	// Create save state structure
	state := SaveState{
		Version:     saveStateVersion2,
		CPUState:    e.CPU.State,
		PPUState:    e.savePPUState(),
		APUState:    e.saveAPUState(),
		MemoryState: e.saveMemoryState(),
		InputState:  e.saveInputState(),
		Running:     e.Running,
		Paused:      e.Paused,
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
	if state.Version != saveStateVersion1 && state.Version != saveStateVersion2 {
		return fmt.Errorf("unsupported save state version: %d (expected %d or %d)", state.Version, saveStateVersion1, saveStateVersion2)
	}

	// Restore state
	e.CPU.State = state.CPUState
	e.loadPPUState(state.PPUState, state.Version)
	e.loadAPUState(state.APUState)
	e.loadMemoryState(state.MemoryState)
	e.loadInputState(state.InputState)
	e.Running = state.Running
	e.Paused = state.Paused

	return nil
}

// savePPUState extracts PPU state for saving
func (e *Emulator) savePPUState() PPUState {
	var transformChannels [4]ppu.TransformChannel
	for i := 0; i < ppu.NumTransformChannels && i < 4; i++ {
		transformChannels[i] = e.PPU.TransformChannels[i]
	}
	return PPUState{
		VRAM:                   e.PPU.VRAM,
		CGRAM:                  e.PPU.CGRAM,
		OAM:                    e.PPU.OAM,
		BG0:                    e.PPU.BG0,
		BG1:                    e.PPU.BG1,
		BG2:                    e.PPU.BG2,
		BG3:                    e.PPU.BG3,
		TransformChannels:      transformChannels,
		MatrixPlanes:           e.PPU.MatrixPlanes,
		Window0:                e.PPU.Window0,
		Window1:                e.PPU.Window1,
		WindowControl:          e.PPU.WindowControl,
		WindowMainEnable:       e.PPU.WindowMainEnable,
		WindowSubEnable:        e.PPU.WindowSubEnable,
		HDMAEnabled:            e.PPU.HDMAEnabled,
		HDMATableBase:          e.PPU.HDMATableBase,
		HDMAControl:            e.PPU.HDMAControl,
		HDMAExtControl:         e.PPU.HDMAExtControl,
		FrameCounter:           e.PPU.FrameCounter,
		VBlankFlag:             e.PPU.VBlankFlag,
		VRAMAddr:               e.PPU.VRAMAddr,
		CGRAMAddr:              e.PPU.CGRAMAddr,
		CGRAMWriteLatch:        e.PPU.CGRAMWriteLatch,
		CGRAMWriteValue:        e.PPU.CGRAMWriteValue,
		MatrixPlaneSelect:      e.PPU.MatrixPlaneSelect,
		MatrixPlaneAddr:        e.PPU.MatrixPlaneAddr,
		MatrixPlanePatternAddr: e.PPU.MatrixPlanePatternAddr,
		MatrixPlaneBitmapAddr:  e.PPU.MatrixPlaneBitmapAddr,
		MatrixPlaneRowAddr:     e.PPU.MatrixPlaneRowAddr,
		DMAEnabled:             e.PPU.DMAEnabled,
		DMASourceBank:          e.PPU.DMASourceBank,
		DMASourceOffset:        e.PPU.DMASourceOffset,
		DMADestType:            e.PPU.DMADestType,
		DMADestAddr:            e.PPU.DMADestAddr,
		DMALength:              e.PPU.DMALength,
		DMAMode:                e.PPU.DMAMode,
		DMACycles:              e.PPU.DMACycles,
		DMAProgress:            e.PPU.DMAProgress,
		DMACurrentSrc:          e.PPU.DMACurrentSrc,
		DMACurrentDest:         e.PPU.DMACurrentDest,
		DMAFillValue:           e.PPU.DMAFillValue,
		OAMAddr:                e.PPU.OAMAddr,
		OAMByteIndex:           e.PPU.OAMByteIndex,
	}
}

// loadPPUState restores PPU state from saved state
func (e *Emulator) loadPPUState(state PPUState, version uint16) {
	e.PPU.VRAM = state.VRAM
	e.PPU.CGRAM = state.CGRAM
	e.PPU.OAM = state.OAM
	e.PPU.BG0 = state.BG0
	e.PPU.BG1 = state.BG1
	e.PPU.BG2 = state.BG2
	e.PPU.BG3 = state.BG3
	for i := 0; i < ppu.NumTransformChannels && i < 4; i++ {
		e.PPU.TransformChannels[i] = state.TransformChannels[i]
	}
	e.PPU.Window0 = state.Window0
	e.PPU.Window1 = state.Window1
	e.PPU.WindowControl = state.WindowControl
	e.PPU.WindowMainEnable = state.WindowMainEnable
	e.PPU.WindowSubEnable = state.WindowSubEnable
	e.PPU.HDMAEnabled = state.HDMAEnabled
	e.PPU.HDMATableBase = state.HDMATableBase
	if version >= saveStateVersion2 {
		e.PPU.MatrixPlanes = state.MatrixPlanes
		e.PPU.HDMAControl = state.HDMAControl
		e.PPU.HDMAExtControl = state.HDMAExtControl
		e.PPU.MatrixPlaneSelect = state.MatrixPlaneSelect
		e.PPU.MatrixPlaneAddr = state.MatrixPlaneAddr
		e.PPU.MatrixPlanePatternAddr = state.MatrixPlanePatternAddr
		e.PPU.MatrixPlaneBitmapAddr = state.MatrixPlaneBitmapAddr
		e.PPU.MatrixPlaneRowAddr = state.MatrixPlaneRowAddr
		e.PPU.DMAEnabled = state.DMAEnabled
		e.PPU.DMASourceBank = state.DMASourceBank
		e.PPU.DMASourceOffset = state.DMASourceOffset
		e.PPU.DMADestType = state.DMADestType
		e.PPU.DMADestAddr = state.DMADestAddr
		e.PPU.DMALength = state.DMALength
		e.PPU.DMAMode = state.DMAMode
		e.PPU.DMACycles = state.DMACycles
		e.PPU.DMAProgress = state.DMAProgress
		e.PPU.DMACurrentSrc = state.DMACurrentSrc
		e.PPU.DMACurrentDest = state.DMACurrentDest
		e.PPU.DMAFillValue = state.DMAFillValue
	} else {
		// Version 1 save states predate matrix-plane serialization. Reset the
		// newer plane/MMIO state so we don't leak stale runtime data across loads.
		e.PPU.MatrixPlanes = [ppu.NumTransformChannels]ppu.MatrixPlane{}
		for i := range e.PPU.MatrixPlanes {
			e.PPU.MatrixPlanes[i].Size = ppu.TilemapSize32x32
			e.PPU.MatrixPlanes[i].BaseDistance = 0x0100
			e.PPU.MatrixPlanes[i].FocalLength = 0x3000
			e.PPU.MatrixPlanes[i].WidthScale = 0x0100
			e.PPU.MatrixPlanes[i].HeightScale = 0x0200
			e.PPU.MatrixPlanes[i].HeadingY = -0x0100
			e.PPU.MatrixPlanes[i].FacingY = -0x0100
		}
		e.PPU.HDMAControl = 0
		e.PPU.HDMAExtControl = 0
		e.PPU.MatrixPlaneSelect = 0
		e.PPU.MatrixPlaneAddr = 0
		e.PPU.MatrixPlanePatternAddr = 0
		e.PPU.MatrixPlaneBitmapAddr = 0
		e.PPU.MatrixPlaneRowAddr = 0
		e.PPU.DMAEnabled = false
		e.PPU.DMASourceBank = 0
		e.PPU.DMASourceOffset = 0
		e.PPU.DMADestType = 0
		e.PPU.DMADestAddr = 0
		e.PPU.DMALength = 0
		e.PPU.DMAMode = 0
		e.PPU.DMACycles = 0
		e.PPU.DMAProgress = 0
		e.PPU.DMACurrentSrc = 0
		e.PPU.DMACurrentDest = 0
		e.PPU.DMAFillValue = 0
	}
	e.PPU.FrameCounter = state.FrameCounter
	e.PPU.VBlankFlag = state.VBlankFlag
	e.PPU.VRAMAddr = state.VRAMAddr
	e.PPU.CGRAMAddr = state.CGRAMAddr
	e.PPU.CGRAMWriteLatch = state.CGRAMWriteLatch
	e.PPU.CGRAMWriteValue = state.CGRAMWriteValue
	e.PPU.OAMAddr = state.OAMAddr
	e.PPU.OAMByteIndex = state.OAMByteIndex
	e.PPU.ResetDerivedRuntimeCachesForStateLoad()
	e.PPU.SyncTransformBindingsForStateLoad()
}

// saveAPUState extracts APU state for saving
func (e *Emulator) saveAPUState() APUState {
	return APUState{
		Channels:                e.APU.Channels,
		MasterVolume:            e.APU.MasterVolume,
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
		WRAM:         e.Bus.WRAM,
		WRAMExtended: e.Bus.WRAMExtended,
	}
}

// loadMemoryState restores Memory state from saved state
func (e *Emulator) loadMemoryState(state MemoryState) {
	e.Bus.WRAM = state.WRAM
	e.Bus.WRAMExtended = state.WRAMExtended
}

// saveInputState extracts Input state for saving
func (e *Emulator) saveInputState() InputState {
	return InputState{
		Controller1Buttons:    e.Input.Controller1Buttons,
		Controller2Buttons:    e.Input.Controller2Buttons,
		Controller1Latched:    e.Input.Controller1Latched,
		Controller2Latched:    e.Input.Controller2Latched,
		Controller1LatchState: e.Input.Controller1LatchState,
		Controller2LatchState: e.Input.Controller2LatchState,
	}
}

// loadInputState restores Input state from saved state
func (e *Emulator) loadInputState(state InputState) {
	e.Input.Controller1Buttons = state.Controller1Buttons
	e.Input.Controller2Buttons = state.Controller2Buttons
	e.Input.Controller1Latched = state.Controller1Latched
	e.Input.Controller2Latched = state.Controller2Latched
	e.Input.Controller1LatchState = state.Controller1LatchState
	e.Input.Controller2LatchState = state.Controller2LatchState
}

// SaveStateToFile saves the current emulator state to a file.
func (e *Emulator) SaveStateToFile(filename string) error {
	data, err := e.SaveState()
	if err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	if err := os.WriteFile(filename, data, 0o644); err != nil {
		return fmt.Errorf("failed to write save state file %q: %w", filename, err)
	}

	return nil
}

// LoadStateFromFile loads an emulator state from a file.
func (e *Emulator) LoadStateFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read save state file %q: %w", filename, err)
	}

	if err := e.LoadState(data); err != nil {
		return fmt.Errorf("failed to load save state from %q: %w", filename, err)
	}

	return nil
}
