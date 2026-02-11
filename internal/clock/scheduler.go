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
	CPUSpeed  uint32 // ~7.67 MHz = 7,670,000 cycles/sec (Genesis-like)
	PPUSpeed  uint32 // Same as CPU (unified clock)
	APUSpeed  uint32 // 44,100 Hz sample rate

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
		Cycle:                   0,
		CPUSpeed:                cpuSpeed,
		PPUSpeed:                ppuSpeed,
		APUSpeed:                apuSpeed,
		CPUNextCycle:            0,
		PPUNextCycle:            0,
		APUNextCycle:            0,
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

	// Check APU (runs at sample rate: 44,100 Hz = every ~174 cycles at ~7.67 MHz)
	// Use fractional accumulator for accurate timing
	if c.APUStep != nil && c.Cycle >= c.APUNextCycle {
		cyclesToRun := c.Cycle - c.APUNextCycle + 1
		if err := c.APUStep(cyclesToRun); err != nil {
			return 0, fmt.Errorf("APU step error: %w", err)
		}
		
		// Calculate exact cycles per sample using fixed-point arithmetic
		// CPUSpeed / APUSpeed = cycles per sample (may be fractional)
		// Use 32-bit fractional part for precision
		// Formula: (CPUSpeed * 2^32) / APUSpeed gives us fixed-point cycles per sample
		exactCyclesPerSampleFixed := (uint64(c.CPUSpeed) << 32) / uint64(c.APUSpeed)
		
		// Add fractional cycles to accumulator
		c.APUFractionalAccumulator += exactCyclesPerSampleFixed
		
		// Extract integer part (cycles until next sample)
		integerCycles := c.APUFractionalAccumulator >> 32
		
		// Keep fractional part in accumulator
		c.APUFractionalAccumulator &= 0xFFFFFFFF
		
		// Schedule next APU step
		c.APUNextCycle = c.Cycle + integerCycles
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
	
	// APU runs at sample rate - step it for each sample that occurs in this batch
	// Use fractional accumulator for accurate timing
	if c.APUStep != nil {
		cycleAtStart := c.Cycle
		cycleAtEnd := c.Cycle + cycles
		
		// Calculate exact cycles per sample using fixed-point arithmetic
		exactCyclesPerSampleFixed := (uint64(c.CPUSpeed) << 32) / uint64(c.APUSpeed)
		
		// Step APU for each sample that occurs during this batch
		for c.APUNextCycle < cycleAtEnd {
			if c.APUNextCycle >= cycleAtStart {
				// APU sample occurs during this batch - step it
				cyclesToRun := c.APUNextCycle - cycleAtStart + 1
				if cyclesToRun > cycles {
					cyclesToRun = cycles
				}
				if err := c.APUStep(cyclesToRun); err != nil {
					return fmt.Errorf("APU step error: %w", err)
				}
				
				// Add fractional cycles to accumulator
				c.APUFractionalAccumulator += exactCyclesPerSampleFixed
				
				// Extract integer part (cycles until next sample)
				integerCycles := c.APUFractionalAccumulator >> 32
				
				// Keep fractional part in accumulator
				c.APUFractionalAccumulator &= 0xFFFFFFFF
				
				// Schedule next APU step
				c.APUNextCycle += integerCycles
			} else {
				// APU sample was before this batch, advance to next
				// Add fractional cycles to accumulator
				c.APUFractionalAccumulator += exactCyclesPerSampleFixed
				
				// Extract integer part
				integerCycles := c.APUFractionalAccumulator >> 32
				
				// Keep fractional part
				c.APUFractionalAccumulator &= 0xFFFFFFFF
				
				// Advance to next sample
				c.APUNextCycle += integerCycles
			}
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
