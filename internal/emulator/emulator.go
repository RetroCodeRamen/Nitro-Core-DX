package emulator

import (
	"fmt"
	"time"

	"nitro-core-dx/internal/apu"
	"nitro-core-dx/internal/clock"
	"nitro-core-dx/internal/cpu"
	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/input"
	"nitro-core-dx/internal/memory"
	"nitro-core-dx/internal/ppu"
)

// Emulator represents the clock-driven emulator
// This is the FPGA-ready implementation using cycle-accurate clock scheduling
type Emulator struct {
	// Components
	CPU       *cpu.CPU
	Bus       *memory.Bus
	Cartridge *memory.Cartridge
	PPU       *ppu.PPU
	APU       *apu.APU
	Input     *input.InputSystem
	Logger    *debug.Logger

	// Clock scheduler (core of FPGA-ready design)
	Clock *clock.MasterClock

	// Frame timing (for compatibility with host systems)
	FrameLimitEnabled bool
	TargetFPS         float64
	FrameTime         time.Duration
	LastFrameTime     time.Time

	// Performance tracking
	FPS                float64
	FrameCount         uint64
	FPSUpdateTime      time.Time
	CPUCyclesPerFrame  uint32
	LastCPUCycles      uint32
	CyclesPerFrame     uint64 // 79,200 cycles per frame (220 scanlines × 360 dots)

	// State
	Running bool
	Paused  bool

	// Audio samples buffer (for host adapter)
	AudioSampleBuffer []int16
	AudioSampleIndex  int
}

// NewEmulator creates a new clock-driven emulator instance
func NewEmulator() *Emulator {
	logger := debug.NewLogger(10000)
	return NewEmulatorWithLogger(logger)
}

// NewEmulatorWithLogger creates a new clock-driven emulator with a logger
func NewEmulatorWithLogger(logger *debug.Logger) *Emulator {
	// Create cartridge
	cartridge := memory.NewCartridge()

	// Create bus
	bus := memory.NewBus(cartridge)

	// Create components
	ppu := ppu.NewPPU(logger)
	apu := apu.NewAPU(44100, logger)
	input := input.NewInputSystem()

	// Connect I/O handlers to bus
	bus.PPUHandler = ppu
	bus.APUHandler = apu
	bus.InputHandler = input

	// Create CPU logger adapter
	cpuLogger := cpu.NewCPULoggerAdapter(logger, cpu.CPULogNone)

	// Create CPU with bus (not MemorySystem)
	cpu := cpu.NewCPU(bus, cpuLogger)

	// Create clock scheduler (10 MHz CPU, 10 MHz PPU, 44,100 Hz APU)
	masterClock := clock.NewMasterClock(10000000, 10000000, 44100)

	// Register component step functions
	masterClock.CPUStep = func(cycles uint64) error {
		return cpu.StepCPU(cycles)
	}
	masterClock.PPUStep = func(cycles uint64) error {
		return ppu.StepPPU(cycles)
	}
	masterClock.APUStep = func(cycles uint64) error {
		return apu.StepAPU(cycles)
	}

	emu := &Emulator{
		CPU:                cpu,
		Bus:                bus,
		Cartridge:          cartridge,
		PPU:                ppu,
		APU:                apu,
		Input:              input,
		Logger:             logger,
		Clock:              masterClock,
		FrameLimitEnabled:  true,
		TargetFPS:          60.0,
		FrameTime:          time.Duration(1000000000 / 60),
		LastFrameTime:      time.Now(),
		FPS:                0.0,
		FrameCount:         0,
		FPSUpdateTime:      time.Now(),
		CPUCyclesPerFrame:  0,
		LastCPUCycles:      0,
		CyclesPerFrame:     79200, // PPU frame timing: 220 scanlines × 360 dots = 79,200 cycles
		Running:            false,
		Paused:              false,
		AudioSampleBuffer:   make([]int16, 735), // 735 samples per frame
		AudioSampleIndex:   0,
	}

	return emu
}

// LoadROM loads a ROM file
func (e *Emulator) LoadROM(data []uint8) error {
	if err := e.Cartridge.LoadROM(data); err != nil {
		return fmt.Errorf("failed to load ROM: %w", err)
	}

	// Set CPU entry point
	bank, offset, err := e.Cartridge.GetROMEntryPoint()
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

// RunFrame runs a single frame using clock-driven execution
// This is cycle-accurate and FPGA-ready
func (e *Emulator) RunFrame() error {
	if !e.Running || e.Paused {
		return nil
	}

	// Track CPU cycles before frame
	cyclesBefore := e.CPU.State.Cycles

	// Step clock for one frame (79,200 cycles = 220 scanlines × 360 dots per scanline)
	// The clock scheduler coordinates CPU, PPU, and APU at cycle boundaries
	// This is the core of FPGA-ready design - all components run cycle-accurately
	// PPU renders dot-by-dot, scanline-by-scanline, matching hardware timing exactly
	
	// Generate audio samples during frame execution
	// At 44,100 Hz sample rate and 60 FPS, we need 735 samples per frame
	// APU runs every ~227 cycles (10,000,000 / 44,100 ≈ 226.76)
	apuCyclesPerSample := uint64(10000000 / 44100) // ~227 cycles per sample
	samplesGenerated := 0
	
	// Step clock cycle by cycle, collecting audio samples
	for cyclesStepped := uint64(0); cyclesStepped < e.CyclesPerFrame; cyclesStepped++ {
		_, err := e.Clock.Step()
		if err != nil {
			return fmt.Errorf("clock step error: %w", err)
		}
		
		// Generate audio sample when it's time (every ~227 cycles)
		if cyclesStepped%apuCyclesPerSample == 0 && samplesGenerated < 735 {
			sampleFixed := e.APU.GenerateSampleFixed()
			if samplesGenerated < len(e.AudioSampleBuffer) {
				e.AudioSampleBuffer[samplesGenerated] = sampleFixed
			}
			samplesGenerated++
		}
	}

	// Calculate CPU cycles used this frame
	cyclesAfter := e.CPU.State.Cycles
	e.CPUCyclesPerFrame = cyclesAfter - cyclesBefore

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
	e.Clock.Reset()
	if e.Cartridge.HasROM() {
		bank, offset, err := e.Cartridge.GetROMEntryPoint()
		if err != nil {
			if e.Logger != nil {
				e.Logger.LogSystem(debug.LogLevelError, fmt.Sprintf("Failed to get ROM entry point during reset: %v", err), nil)
			}
			return
		}
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
	// Convert buffered fixed-point samples to float32
	samples := make([]float32, len(e.AudioSampleBuffer))
	for i, sampleFixed := range e.AudioSampleBuffer {
		samples[i] = apu.ConvertFixedToFloat(sampleFixed)
	}
	return samples
}
