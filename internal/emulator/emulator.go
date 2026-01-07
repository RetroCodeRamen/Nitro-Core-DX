package emulator

import (
	"fmt"
	"time"

	"nitro-core-dx/internal/apu"
	"nitro-core-dx/internal/cpu"
	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/input"
	"nitro-core-dx/internal/memory"
	"nitro-core-dx/internal/ppu"
)

// Emulator represents the complete emulator
type Emulator struct {
	CPU    *cpu.CPU
	Memory *memory.MemorySystem
	PPU    *ppu.PPU
	APU    *apu.APU
	Input  *input.InputSystem
	Logger *debug.Logger // Centralized logger

	// Frame timing
	FrameLimitEnabled bool
	TargetFPS         float64
	FrameTime         time.Duration
	LastFrameTime     time.Time

	// Performance tracking
	FPS               float64
	FrameCount        uint64
	FPSUpdateTime     time.Time
	CPUCyclesPerFrame uint32
	LastCPUCycles     uint32

	// State
	Running bool
	Paused  bool

	// Audio samples from last frame
	LastAudioSamples []float32
}

// NewEmulator creates a new emulator instance
func NewEmulator() *Emulator {
	// Create centralized logger (10,000 entry buffer)
	// Logging is disabled by default - components are disabled in NewLogger
	logger := debug.NewLogger(10000)
	return NewEmulatorWithLogger(logger)
}

// NewEmulatorWithLogger creates a new emulator instance with a specific logger
// This allows enabling logging via command-line flag
func NewEmulatorWithLogger(logger *debug.Logger) *Emulator {

	mem := memory.NewMemorySystem()
	ppu := ppu.NewPPU(logger)
	apu := apu.NewAPU(44100, logger)
	input := input.NewInputSystem()

	// Connect I/O handlers
	mem.PPUHandler = ppu
	mem.APUHandler = apu
	mem.InputHandler = input

	// Create CPU logger adapter with default level (None - logging disabled by default)
	cpuLogger := cpu.NewCPULoggerAdapter(logger, cpu.CPULogNone)

	// Create CPU with logger adapter
	cpu := cpu.NewCPU(mem, cpuLogger)

	return &Emulator{
		CPU:               cpu,
		Memory:            mem,
		PPU:               ppu,
		APU:               apu,
		Input:             input,
		Logger:            logger,
		FrameLimitEnabled: true,
		TargetFPS:         60.0,
		FrameTime:         time.Duration(1000000000 / 60), // 16.666... ms
		LastFrameTime:     time.Now(),
		FPS:               0.0,
		FrameCount:        0,
		FPSUpdateTime:     time.Now(),
		CPUCyclesPerFrame: 0,
		LastCPUCycles:     0,
		Running:           false,
		Paused:            false,
	}
}

// LoadROM loads a ROM file
func (e *Emulator) LoadROM(data []uint8) error {
	if err := e.Memory.LoadROM(data); err != nil {
		return fmt.Errorf("failed to load ROM: %w", err)
	}

	// Set CPU entry point
	bank, offset, err := e.Memory.GetROMEntryPoint()
	if err != nil {
		return fmt.Errorf("failed to get ROM entry point: %w", err)
	}

	// Verify entry point is valid
	if bank == 0 {
		return fmt.Errorf("invalid ROM entry point: bank is 0 (should be 1+)")
	}
	if offset < 0x8000 {
		return fmt.Errorf("invalid ROM entry point: offset 0x%04X (should be >= 0x8000)", offset)
	}

	e.CPU.SetEntryPoint(bank, offset)

	// Verify entry point was set correctly
	if e.CPU.State.PCBank != bank {
		return fmt.Errorf("failed to set entry point: PCBank is %d, expected %d", e.CPU.State.PCBank, bank)
	}

	return nil
}

// RunFrame runs a single frame
func (e *Emulator) RunFrame() error {
	if !e.Running || e.Paused {
		return nil
	}

	// Update APU frame FIRST (handles note duration timers)
	// This ensures channel status is updated BEFORE the CPU runs,
	// so ROMs can check channel completion status during the frame
	e.APU.UpdateFrame()

	// Render frame (sets VBlank flag at start, increments frame counter)
	// This must happen BEFORE CPU execution so CPU can see VBlank flag
	// The frame rendered uses state from the previous frame's CPU execution
	e.PPU.RenderFrame()

	// Track CPU cycles before frame
	cyclesBefore := e.CPU.State.Cycles

	// Calculate target cycles for this frame (10 MHz @ 60 FPS = 166,667 cycles)
	targetCycles := e.CPU.State.Cycles + 166667

	// Run CPU until target cycles
	// CPU can now see VBlank flag and frame counter that were set in RenderFrame()
	if err := e.CPU.ExecuteCycles(targetCycles); err != nil {
		return fmt.Errorf("CPU error at %s: %w", e.CPU.GetPC(), err)
	}

	// Calculate CPU cycles used this frame
	cyclesAfter := e.CPU.State.Cycles
	e.CPUCyclesPerFrame = cyclesAfter - cyclesBefore

	// Generate audio samples (44100 Hz / 60 FPS = 735 samples per frame)
	audioSamples := e.APU.GenerateSamples(735)
	e.LastAudioSamples = audioSamples

	// Update FPS counter
	e.FrameCount++
	now := time.Now()
	if now.Sub(e.FPSUpdateTime) >= time.Second {
		e.FPS = float64(e.FrameCount) / now.Sub(e.FPSUpdateTime).Seconds()
		e.FrameCount = 0
		e.FPSUpdateTime = now
	}

	// Frame limiting
	if e.FrameLimitEnabled {
		elapsed := now.Sub(e.LastFrameTime)
		if elapsed < e.FrameTime {
			time.Sleep(e.FrameTime - elapsed)
		}
		e.LastFrameTime = time.Now()
	} else {
		e.LastFrameTime = time.Now()
	}

	return nil
}

// Start starts the emulator
func (e *Emulator) Start() {
	e.Running = true
	e.Paused = false
}

// Stop stops the emulator
func (e *Emulator) Stop() {
	e.Running = false
}

// Pause pauses the emulator
func (e *Emulator) Pause() {
	e.Paused = true
}

// Resume resumes the emulator
func (e *Emulator) Resume() {
	e.Paused = false
}

// Reset resets the emulator
func (e *Emulator) Reset() {
	e.CPU.Reset()
	if e.Memory.ROMData != nil {
		bank, offset, err := e.Memory.GetROMEntryPoint()
		if err != nil {
			// If we can't get the entry point, something is wrong
			// But don't crash - just log it
			if e.Logger != nil {
				e.Logger.LogSystem(debug.LogLevelError, fmt.Sprintf("Failed to get ROM entry point during reset: %v", err), nil)
			}
			return
		}
		// Validate entry point before setting
		if bank == 0 {
			if e.Logger != nil {
				e.Logger.LogSystem(debug.LogLevelError, "Invalid ROM entry point: bank is 0 (should be 1+)", nil)
			}
			return
		}
		if offset < 0x8000 {
			if e.Logger != nil {
				e.Logger.LogSystem(debug.LogLevelError, fmt.Sprintf("Invalid ROM entry point: offset 0x%04X (should be >= 0x8000)", offset), nil)
			}
			return
		}
		e.CPU.SetEntryPoint(bank, offset)
	}
}

// SetFrameLimit sets the frame limit mode
func (e *Emulator) SetFrameLimit(enabled bool) {
	e.FrameLimitEnabled = enabled
}

// GetFPS returns the current FPS
func (e *Emulator) GetFPS() float64 {
	return e.FPS
}

// GetCPUCyclesPerFrame returns CPU cycles used in the last frame
func (e *Emulator) GetCPUCyclesPerFrame() uint32 {
	return e.CPUCyclesPerFrame
}

// GetOutputBuffer returns the PPU output buffer
func (e *Emulator) GetOutputBuffer() []uint32 {
	return e.PPU.OutputBuffer[:]
}

// SetInputButtons sets the controller button state
func (e *Emulator) SetInputButtons(buttons uint16) {
	e.Input.Controller1Buttons = buttons
}

// GetAudioSamples returns the audio samples from the last frame
func (e *Emulator) GetAudioSamples() []float32 {
	return e.LastAudioSamples
}
