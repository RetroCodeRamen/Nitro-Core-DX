package clock

import (
	"fmt"
)

// MasterClock represents the master clock scheduler
// It coordinates all subsystems (CPU, PPU, APU) based on clock cycles
type MasterClock struct {
	// Current master clock cycle (64-bit to avoid overflow)
	Cycle uint64

	// Clock speeds (cycles per second)
	CPUSpeed uint32 // ~7.67 MHz = 7,670,000 cycles/sec (Genesis-like)
	PPUSpeed uint32 // Same as CPU (unified clock)
	APUSpeed uint32 // 44,100 Hz sample rate

	// Component cycle counters (when each component should run next)
	CPUNextCycle uint64
	PPUNextCycle uint64
	APUNextCycle uint64

	// APU fractional accumulator for accurate timing
	// Fixed-point: 32-bit fractional part (0-2^32 represents 0-1.0 cycles)
	// Tracks fractional cycles to avoid drift from integer division
	APUFractionalAccumulator uint64 // Fixed-point fractional cycles (32-bit fractional part)

	// Component step functions
	CPUStep func(cycles uint64) error
	PPUStep func(cycles uint64) error
	APUStep func(cycles uint64) error
}

// NewMasterClock creates a new master clock scheduler
func NewMasterClock(cpuSpeed, ppuSpeed, apuSpeed uint32) *MasterClock {
	return &MasterClock{
		Cycle:                    0,
		CPUSpeed:                 cpuSpeed,
		PPUSpeed:                 ppuSpeed,
		APUSpeed:                 apuSpeed,
		CPUNextCycle:             0,
		PPUNextCycle:             0,
		APUNextCycle:             0,
		APUFractionalAccumulator: 0,
	}
}

// Step advances the clock by the minimum step needed to trigger the next component
// Returns the number of cycles advanced
func (c *MasterClock) Step() (uint64, error) {
	// Check CPU
	if c.CPUStep != nil && c.Cycle >= c.CPUNextCycle {
		cyclesToRun := c.Cycle - c.CPUNextCycle + 1
		if err := c.CPUStep(cyclesToRun); err != nil {
			return 0, fmt.Errorf("CPU step error: %w", err)
		}
		// CPU runs every cycle (~7.67 MHz, unified clock)
		c.CPUNextCycle = c.Cycle + 1
	}

	// Check PPU (runs at same speed as CPU for now)
	if c.PPUStep != nil && c.Cycle >= c.PPUNextCycle {
		cyclesToRun := c.Cycle - c.PPUNextCycle + 1
		if err := c.PPUStep(cyclesToRun); err != nil {
			return 0, fmt.Errorf("PPU step error: %w", err)
		}
		// PPU runs every cycle (for dot-by-dot rendering)
		c.PPUNextCycle = c.Cycle + 1
	}

	// APU is now stepped as a cycle-domain component.
	// Sample-rate pacing is handled by the host audio sampling path,
	// while chip/timer/envelope/phase evolution remains cycle-driven.
	if c.APUStep != nil {
		if err := c.APUStep(1); err != nil {
			return 0, fmt.Errorf("APU step error: %w", err)
		}
	}

	// Advance master clock
	c.Cycle++
	return 1, nil
}

// StepCycles advances the clock by a specific number of cycles
// Optimized version: batches CPU/PPU steps since they run every cycle
func (c *MasterClock) StepCycles(cycles uint64) error {
	if cycles == 0 {
		return nil
	}

	// CPU and PPU run every cycle, so we can batch them
	// Step them for the full batch
	if c.CPUStep != nil {
		if err := c.CPUStep(cycles); err != nil {
			return fmt.Errorf("CPU step error: %w", err)
		}
		c.CPUNextCycle += cycles
	}

	if c.PPUStep != nil {
		if err := c.PPUStep(cycles); err != nil {
			return fmt.Errorf("PPU step error: %w", err)
		}
		c.PPUNextCycle += cycles
	}

	// APU is stepped for the full cycle batch so chip state evolves
	// continuously in the same clock domain as CPU/PPU.
	if c.APUStep != nil {
		if err := c.APUStep(cycles); err != nil {
			return fmt.Errorf("APU step error: %w", err)
		}
	}

	// Advance master clock
	c.Cycle += cycles
	return nil
}

// GetCycle returns the current master clock cycle
func (c *MasterClock) GetCycle() uint64 {
	return c.Cycle
}

// Reset resets the clock scheduler
func (c *MasterClock) Reset() {
	c.Cycle = 0
	c.CPUNextCycle = 0
	c.PPUNextCycle = 0
	c.APUNextCycle = 0
	c.APUFractionalAccumulator = 0
}
