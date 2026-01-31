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
	FPS               float64
	FrameCount        uint64
	FPSUpdateTime     time.Time
	CPUCyclesPerFrame uint32
	LastCPUCycles     uint32
	CyclesPerFrame    uint64 // 79,200 cycles per frame (220 scanlines × 360 dots)

	// State
	Running bool
	Paused  bool

	// Audio samples buffer (for host adapter)
	AudioSampleBuffer []int16
	AudioSampleIndex  int

	// Cycle logger (for debugging)
	CycleLogger *debug.CycleLogger

	// Debugger (for interactive debugging)
	Debugger *debug.Debugger
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
	
	// Set logger on bus for input debug logging
	bus.SetLogger(logger)

	// Create CPU logger adapter
	cpuLogger := cpu.NewCPULoggerAdapter(logger, cpu.CPULogNone)

	// Create CPU with bus (not MemorySystem)
	cpu := cpu.NewCPU(bus, cpuLogger)

	// Set up PPU interrupt callback to trigger CPU interrupts
	ppu.InterruptCallback = func(interruptType uint8) {
		cpu.TriggerInterrupt(interruptType)
	}

	// Set up PPU memory reader for DMA
	ppu.MemoryReader = func(bank uint8, offset uint16) uint8 {
		return bus.Read8(bank, offset)
	}

	// Initialize interrupt vectors in memory (bank 0, addresses 0xFFE0-0xFFE3)
	// Default vectors point to ROM entry point (can be overridden by ROM)
	// Vector format: 2 bytes per vector (bank, offset_high)
	// Offset low byte is always 0x00 (ROM addresses start at 0x8000+)
	// IRQ vector (0xFFE0-0xFFE1): bank, offset_high
	bus.Write8(0, 0xFFE0, 0x01) // Default bank 1
	bus.Write8(0, 0xFFE1, 0x80) // Default offset high byte (0x8000)
	// NMI vector (0xFFE2-0xFFE3): bank, offset_high
	bus.Write8(0, 0xFFE2, 0x01) // Default bank 1
	bus.Write8(0, 0xFFE3, 0x80) // Default offset high byte (0x8000)

	// Create clock scheduler (~7.67 MHz CPU, ~7.67 MHz PPU, 44,100 Hz APU)
	// Genesis-like speed: 7,670,000 Hz
	// At 60 FPS: 127,833 cycles per frame (220 scanlines × 581 dots = 127,820 cycles)
	cpuSpeed := uint32(7670000) // ~7.67 MHz (Genesis-like)
	ppuSpeed := uint32(7670000) // Same as CPU (unified clock)
	masterClock := clock.NewMasterClock(cpuSpeed, ppuSpeed, 44100)

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
		CPU:               cpu,
		Bus:               bus,
		Cartridge:         cartridge,
		PPU:               ppu,
		APU:               apu,
		Input:             input,
		Logger:            logger,
		Clock:             masterClock,
		FrameLimitEnabled: true,
		TargetFPS:         60.0,
		FrameTime:         time.Duration(1000000000 / 60),
		LastFrameTime:     time.Now(),
		FPS:               0.0,
		FrameCount:        0,
		FPSUpdateTime:     time.Now(),
		CPUCyclesPerFrame: 0,
		LastCPUCycles:     0,
		CyclesPerFrame:    127820, // PPU frame timing: 220 scanlines × 581 dots = 127,820 cycles (~7.67 MHz at 60 FPS)
		Running:           false,
		Paused:            false,
		AudioSampleBuffer: make([]int16, 735), // 735 samples per frame
		AudioSampleIndex:  0,
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

	// Additional validation (entry point should already be validated by GetROMEntryPoint,
	// but we double-check here for safety)
	if bank == 0 {
		return fmt.Errorf("invalid ROM entry point: bank is 0 (expected bank 1-125, got 0). "+
			"ROM code must be located in bank 1 or higher. Bank 0 is reserved for WRAM and I/O registers.")
	}
	if bank > 125 {
		return fmt.Errorf("invalid ROM entry point: bank %d (expected bank 1-125, got %d). "+
			"ROM banks are limited to 1-125. Banks 126-127 are reserved for extended WRAM.",
			bank, bank)
	}
	if offset < 0x8000 {
		return fmt.Errorf("invalid ROM entry point: offset 0x%04X (expected offset 0x8000-0xFFFF, got 0x%04X). "+
			"ROM code must start at offset 0x8000 or higher within the bank (LoROM mapping).",
			offset, offset)
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

	// Step clock for one frame (127,820 cycles = 220 scanlines × 581 dots per scanline)
	// The clock scheduler coordinates CPU, PPU, and APU at cycle boundaries
	// This is the core of FPGA-ready design - all components run cycle-accurately
	// PPU renders dot-by-dot, scanline-by-scanline, matching hardware timing exactly

	// Generate audio samples during frame execution
	// At 44,100 Hz sample rate and 60 FPS, we need 735 samples per frame
	// APU runs every ~174 cycles (7,670,000 / 44,100 ≈ 173.92)
	apuCyclesPerSample := uint64(7670000 / 44100) // ~174 cycles per sample
	samplesGenerated := 0

	// Optimized clock stepping: step CPU/PPU directly for full frame when cycle logging disabled
	// Since CPU and PPU run at same speed (unified clock), we can step them for the entire frame
	// APU needs fine-grained timing (every ~174 cycles), handle separately

	// Step in batches for performance (only cycle-by-cycle if cycle logging enabled)
	if e.CycleLogger != nil && e.CycleLogger.IsEnabled() {
		// Cycle logging enabled: step cycle-by-cycle for accuracy
		for cyclesStepped := uint64(0); cyclesStepped < e.CyclesPerFrame; cyclesStepped++ {
			_, err := e.Clock.Step()
			if err != nil {
				return fmt.Errorf("clock step error: %w", err)
			}

			// Log cycle state
			snapshot := &debug.CPUStateSnapshot{
				R0:       e.CPU.State.R0,
				R1:       e.CPU.State.R1,
				R2:       e.CPU.State.R2,
				R3:       e.CPU.State.R3,
				R4:       e.CPU.State.R4,
				R5:       e.CPU.State.R5,
				R6:       e.CPU.State.R6,
				R7:       e.CPU.State.R7,
				PCBank:   e.CPU.State.PCBank,
				PCOffset: e.CPU.State.PCOffset,
				PBR:      e.CPU.State.PBR,
				DBR:      e.CPU.State.DBR,
				SP:       e.CPU.State.SP,
				Flags:    e.CPU.State.Flags,
				Cycles:   e.CPU.State.Cycles,
			}
			e.CycleLogger.LogCycle(snapshot)

			// Generate audio sample when it's time
			if cyclesStepped%apuCyclesPerSample == 0 && samplesGenerated < 735 {
				sampleFixed := e.APU.GenerateSampleFixed()
				if samplesGenerated < len(e.AudioSampleBuffer) {
					e.AudioSampleBuffer[samplesGenerated] = sampleFixed
				}
				samplesGenerated++
			}
		}
	} else {
		// No cycle logging: optimize by stepping CPU/PPU for full frame directly
		// This bypasses clock scheduler overhead for better performance
		// Step CPU for entire frame
		if err := e.CPU.StepCPU(e.CyclesPerFrame); err != nil {
			return fmt.Errorf("CPU step error: %w", err)
		}

		// Step PPU for entire frame
		if err := e.PPU.StepPPU(e.CyclesPerFrame); err != nil {
			return fmt.Errorf("PPU step error: %w", err)
		}

		// Step APU for each sample in the frame (735 samples per frame at 44.1kHz, 60 FPS)
		for samplesGenerated < 735 {
			if err := e.APU.StepAPU(apuCyclesPerSample); err != nil {
				return fmt.Errorf("APU step error: %w", err)
			}
			sampleFixed := e.APU.GenerateSampleFixed()
			if samplesGenerated < len(e.AudioSampleBuffer) {
				e.AudioSampleBuffer[samplesGenerated] = sampleFixed
			}
			samplesGenerated++
		}

		// Update clock cycle counters to keep them in sync
		e.Clock.Cycle += e.CyclesPerFrame
		e.Clock.CPUNextCycle += e.CyclesPerFrame
		e.Clock.PPUNextCycle += e.CyclesPerFrame
		e.Clock.APUNextCycle += uint64(735 * apuCyclesPerSample)
	}

	// Calculate CPU cycles used this frame
	// Note: This is CPU instruction cycles (each instruction takes multiple cycles),
	// not clock cycles. Clock cycles per frame = 127,820 (220 scanlines × 581 dots)
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
				e.Logger.LogSystem(debug.LogLevelError, "Invalid ROM entry point: bank is 0 (expected bank 1-125, got 0). ROM code must be located in bank 1 or higher.", nil)
			}
			return
		}
		if bank > 125 {
			if e.Logger != nil {
				e.Logger.LogSystem(debug.LogLevelError, fmt.Sprintf("Invalid ROM entry point: bank %d (expected bank 1-125, got %d). ROM banks are limited to 1-125.", bank, bank), nil)
			}
			return
		}
		if offset < 0x8000 {
			if e.Logger != nil {
				e.Logger.LogSystem(debug.LogLevelError, fmt.Sprintf("Invalid ROM entry point: offset 0x%04X (expected offset 0x8000-0xFFFF, got 0x%04X). ROM code must start at offset 0x8000 or higher.", offset, offset), nil)
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
