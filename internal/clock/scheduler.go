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
	CPUSpeed  uint32 // 10 MHz = 10,000,000 cycles/sec
	PPUSpeed  uint32 // Same as CPU for now
	APUSpeed  uint32 // 44,100 Hz sample rate

	// Component cycle counters (when each component should run next)
	CPUNextCycle uint64
	PPUNextCycle uint64
	APUNextCycle uint64

	// Component step functions
	CPUStep func(cycles uint64) error
	PPUStep func(cycles uint64) error
	APUStep func(cycles uint64) error
}

// NewMasterClock creates a new master clock scheduler
func NewMasterClock(cpuSpeed, ppuSpeed, apuSpeed uint32) *MasterClock {
	return &MasterClock{
		Cycle:        0,
		CPUSpeed:     cpuSpeed,
		PPUSpeed:     ppuSpeed,
		APUSpeed:     apuSpeed,
		CPUNextCycle: 0,
		PPUNextCycle: 0,
		APUNextCycle: 0,
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
		// CPU runs every cycle (10 MHz)
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

	// Check APU (runs at sample rate: 44,100 Hz = every ~227 cycles at 10 MHz)
	if c.APUStep != nil && c.Cycle >= c.APUNextCycle {
		cyclesToRun := c.Cycle - c.APUNextCycle + 1
		if err := c.APUStep(cyclesToRun); err != nil {
			return 0, fmt.Errorf("APU step error: %w", err)
		}
		// APU runs every ~227 cycles (10,000,000 / 44,100 ≈ 226.76)
		apuCyclesPerSample := uint64(c.CPUSpeed / c.APUSpeed)
		c.APUNextCycle = c.Cycle + apuCyclesPerSample
	}

	// Advance master clock
	c.Cycle++
	return 1, nil
}

// StepCycles advances the clock by a specific number of cycles
func (c *MasterClock) StepCycles(cycles uint64) error {
	for i := uint64(0); i < cycles; i++ {
		if _, err := c.Step(); err != nil {
			return err
		}
	}
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
}
