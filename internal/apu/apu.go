package apu

import (
	"math"
	"time"

	"nitro-core-dx/internal/debug"
)

// APU represents the Audio Processing Unit
// It implements the memory.IOHandler interface
//
// Design Philosophy:
// - Developer-friendly: Easy to use, intuitive API
// - Hardware-accurate: Behaves like real retro hardware
// - Clean audio: No artifacts, clicks, or warbling
// - Flexible: Supports various audio use cases
type APU struct {
	Channels        [4]AudioChannel
	MasterVolume    uint8
	SampleRate      uint32
	debugFrameCount int

	// Logger for centralized logging
	Logger *debug.Logger

	// Debug: Track frequency changes
	lastFrequencyChange  [4]time.Time
	frequencyChangeCount [4]int

	// Debug logging control
	debugLoggingEnabled bool
	debugLogStartTime   time.Time

	// Channel completion status (developer-friendly!)
	// Bits 0-3 indicate which channels just finished this frame
	// ROM can read this once per frame to check for completion
	// This register is automatically cleared after being read
	ChannelCompletionStatus uint8
}

// AudioChannel represents an audio channel
type AudioChannel struct {
	// Current state
	Frequency uint16 // Current frequency (Hz)
	Volume    uint8  // Volume (0-255)
	Enabled   bool   // Channel enabled
	Waveform  uint8  // 0=sine, 1=square, 2=saw, 3=noise

	// Note duration (in frames) - developer-friendly timing!
	// When set to non-zero, counts down each frame
	// When reaches 0, channel auto-disables (if autoStop is true)
	Duration        uint16 // Remaining frames (0 = play indefinitely)
	DurationMode    uint8  // 0=countdown and stop, 1=countdown and loop (restart note)
	InitialDuration uint16 // Initial duration value (for loop mode - stored when channel enabled)

	// Phase accumulator for waveform generation
	// Fixed-point: 32-bit unsigned integer (0-2^32 represents 0-2π)
	PhaseFixed          uint32 // Current phase (fixed-point)
	PhaseIncrementFixed uint32 // Phase increment per sample (fixed-point)

	// Legacy floating-point fields (deprecated, kept for compatibility during transition)
	Phase          float64 // Current phase (0 to 2π) - DEPRECATED: use PhaseFixed
	PhaseIncrement float64 // Phase increment per sample - DEPRECATED: use PhaseIncrementFixed

	// Noise generator state
	NoiseLFSR uint16 // LFSR state for noise waveform

	// Internal: Track frequency updates to prevent unnecessary phase resets
	// This prevents warbling when the same frequency is written multiple times
	lastCompleteFrequency  uint16 // Last complete frequency that was set
	pendingFrequencyLow    uint8  // Pending low byte (not yet committed)
	frequencyUpdatePending bool   // True if low byte written but high byte not yet written
}

// NewAPU creates a new APU instance
func NewAPU(sampleRate uint32, logger *debug.Logger) *APU {
	apu := &APU{
		SampleRate:          sampleRate,
		MasterVolume:        255,
		Logger:              logger,
		debugLoggingEnabled: true, // Enable debug logging for first 5 seconds
		debugLogStartTime:   time.Now(),
	}
	return apu
}

// Read8 reads an 8-bit value from APU registers
func (a *APU) Read8(offset uint16) uint8 {
	// Global APU registers (not channel-specific)
	// Check these BEFORE channel-specific registers
	if offset == 0x20 { // MASTER_VOLUME
		return a.MasterVolume
	}
	if offset == 0x21 { // CHANNEL_COMPLETION_STATUS
		// Read completion status (ONE-SHOT: cleared immediately after read)
		// Bits 0-3 indicate which channels finished this frame
		// Status is cleared immediately after being read to prevent multiple updates per frame
		// This ensures the ROM only sees the completion status once per frame
		status := a.ChannelCompletionStatus
		a.ChannelCompletionStatus = 0 // Clear immediately after read (one-shot behavior)
		if a.shouldLog() && status != 0 && a.Logger != nil {
			channels := []int{}
			for i := 0; i < 4; i++ {
				if (status & (1 << i)) != 0 {
					channels = append(channels, i)
				}
			}
			a.Logger.LogAPUf(debug.LogLevelInfo,
				"CHANNEL_COMPLETION_STATUS read: 0x%02X (channels finished: %v) - cleared after read",
				status, channels)
		}
		return status
	}

	// Channel-specific registers (8 bytes per channel)
	channel := int((offset / 8) & 0x3) // Changed from /4 to /8
	reg := offset & 0x7                // Changed from &0x3 to &0x7

	ch := &a.Channels[channel]

	switch reg {
	case 0: // FREQ_LOW
		return uint8(ch.Frequency & 0xFF)
	case 1: // FREQ_HIGH
		return uint8(ch.Frequency >> 8)
	case 2: // VOLUME
		return ch.Volume
	case 3: // CONTROL - enable bit + waveform
		control := uint8(0)
		if ch.Enabled {
			control |= 0x01
		}
		if channel < 3 {
			control |= (ch.Waveform & 0x3) << 1
		} else {
			if ch.Waveform == 3 {
				control |= 0x02
			}
		}
		return control
	case 4: // DURATION_LOW
		return uint8(ch.Duration & 0xFF)
	case 5: // DURATION_HIGH
		return uint8(ch.Duration >> 8)
	case 6: // DURATION_MODE
		return ch.DurationMode
	case 7: // Reserved (could be used for channel-specific status)
		return 0
	default:
		// Reserved
		return 0
	}
}

// Write8 writes an 8-bit value to APU registers
//
// Register Layout (per channel, 8 bytes each - expanded for developer convenience!):
//
//	Offset 0: FREQ_LOW      - Frequency low byte
//	Offset 1: FREQ_HIGH     - Frequency high byte (triggers update)
//	Offset 2: VOLUME        - Volume (0-255)
//	Offset 3: CONTROL       - Enable + waveform
//	Offset 4: DURATION_LOW  - Note duration low byte (frames)
//	Offset 5: DURATION_HIGH - Note duration high byte (frames)
//	Offset 6: DURATION_MODE - Duration mode (0=stop when done, 1=loop/restart)
//	Offset 7: Reserved
//
// Design Philosophy: Developer-friendly!
// - Set frequency, volume, duration, and enable - the APU handles timing automatically
// - No need to manually count frames or loop iterations
// - Duration in frames (60 frames = 1 second at 60 FPS)
func (a *APU) Write8(offset uint16, value uint8) {
	channel := int((offset / 8) & 0x3) // Changed from /4 to /8
	reg := offset & 0x7                // Changed from &0x3 to &0x7

	// Debug: Log APU writes (only first few to verify it's working)
	// fmt.Fprintf(os.Stderr, "[APU Write8] offset=0x%04X, channel=%d, reg=%d, value=0x%02X\n", offset, channel, reg, value)

	switch reg {
	case 0: // FREQ_LOW
		// Store low byte but don't update yet
		// This allows atomic frequency updates (low + high together)
		ch := &a.Channels[channel]
		ch.pendingFrequencyLow = value
		ch.frequencyUpdatePending = true

		// Update the frequency value (but it's incomplete until high byte is written)
		ch.Frequency = (ch.Frequency & 0xFF00) | uint16(value)

	case 1: // FREQ_HIGH
		// Complete the frequency update
		ch := &a.Channels[channel]

		// Reconstruct the complete frequency from pending low byte and new high byte
		// This ensures we get the correct value even if low byte was written earlier
		var newFreq uint16
		if ch.frequencyUpdatePending {
			// Use the pending low byte (the one that was written before this high byte)
			newFreq = uint16(ch.pendingFrequencyLow) | (uint16(value) << 8)
		} else {
			// High byte written without low byte first - use current low byte
			newFreq = (ch.Frequency & 0x00FF) | (uint16(value) << 8)
		}

		// Get the old complete frequency for comparison
		oldFreq := ch.lastCompleteFrequency

		// Update frequency and mark as complete
		ch.Frequency = newFreq
		ch.lastCompleteFrequency = newFreq
		ch.frequencyUpdatePending = false

		// Update phase increment with new frequency (fixed-point)
		a.updatePhaseIncrementFixed(channel)
		// Also update legacy floating-point for compatibility
		a.updatePhaseIncrement(channel)
		// fmt.Printf("[APU] Channel %d: Frequency updated to %d Hz (0x%04X), PhaseIncrement=%f\n",
		// 	channel, newFreq, newFreq, ch.PhaseIncrement)

		// CRITICAL: Only reset phase if frequency ACTUALLY changed
		// This prevents warbling from redundant writes while ensuring clean note starts
		// Real hardware (NES, SNES) resets phase when frequency changes, not on every write
		if newFreq != oldFreq && newFreq != 0 {
			// Frequency changed - reset phase for clean note start
			// This matches real hardware behavior and prevents phase discontinuities
			ch.PhaseFixed = 0
			ch.Phase = 0.0 // Legacy

			// Debug logging
			now := time.Now()
			a.frequencyChangeCount[channel]++

			if a.shouldLog() && a.Logger != nil {
				timeSinceLastChange := time.Duration(0)
				if !a.lastFrequencyChange[channel].IsZero() {
					timeSinceLastChange = now.Sub(a.lastFrequencyChange[channel])
				}
				a.Logger.LogAPUf(debug.LogLevelDebug,
					"Channel %d: Frequency changed %d Hz (0x%04X) -> %d Hz (0x%04X) | Time since last: %v",
					channel, oldFreq, oldFreq, newFreq, newFreq, timeSinceLastChange)
			}

			a.lastFrequencyChange[channel] = now
		}
		// If frequency didn't change, phase continues naturally (no reset)
		// This allows smooth playback without artifacts

	case 2: // VOLUME
		a.Channels[channel].Volume = value

	case 3: // CONTROL
		ch := &a.Channels[channel]
		wasEnabled := ch.Enabled
		ch.Enabled = (value & 0x01) != 0

		if channel < 3 {
			// Channels 0-2: waveform in bits 1-2
			ch.Waveform = (value >> 1) & 0x3
		} else {
			// Channel 3: bit 1 selects noise vs square
			if (value & 0x02) != 0 {
				ch.Waveform = 3 // Noise
			} else {
				ch.Waveform = 1 // Square
			}
		}

		// When enabling a channel, store initial duration for loop mode
		if !wasEnabled && ch.Enabled && ch.Duration > 0 {
			ch.InitialDuration = ch.Duration
		}

		// Debug: Log channel enable/disable
		if a.shouldLog() && a.Logger != nil {
			if !wasEnabled && ch.Enabled {
				a.Logger.LogAPUf(debug.LogLevelInfo,
					"Channel %d: ENABLED - Freq=%d Hz, Volume=%d, Waveform=%d, Duration=%d frames (InitialDuration=%d)",
					channel, ch.Frequency, ch.Volume, ch.Waveform, ch.Duration, ch.InitialDuration)
			} else if wasEnabled && !ch.Enabled {
				a.Logger.LogAPUf(debug.LogLevelInfo, "Channel %d: DISABLED", channel)
			}
		}

		// When enabling a channel, if duration is set, start the timer
		// When disabling, duration continues (so re-enabling resumes timing)

	case 4: // DURATION_LOW
		ch := &a.Channels[channel]
		ch.Duration = (ch.Duration & 0xFF00) | uint16(value)
		// Update InitialDuration if channel is enabled (for loop mode)
		if ch.Enabled && ch.Duration > 0 {
			ch.InitialDuration = ch.Duration
		}

	case 5: // DURATION_HIGH
		ch := &a.Channels[channel]
		ch.Duration = (ch.Duration & 0x00FF) | (uint16(value) << 8)
		// When duration is set, channel will count down each frame
		// Update InitialDuration if channel is enabled (for loop mode)
		if ch.Enabled && ch.Duration > 0 {
			ch.InitialDuration = ch.Duration
		}

	case 6: // DURATION_MODE
		ch := &a.Channels[channel]
		ch.DurationMode = value & 0x01 // Bit 0: 0=stop when done, 1=loop
		// Bit 1+: reserved for future use

	case 7: // Reserved
		// Reserved for future expansion
	}

	if offset == 0x20 { // MASTER_VOLUME (moved to 0x20 since channels are now 8 bytes)
		a.MasterVolume = value
	}
}

// Read16 reads a 16-bit value from APU registers
func (a *APU) Read16(offset uint16) uint16 {
	low := a.Read8(offset)
	high := a.Read8(offset + 1)
	return uint16(low) | (uint16(high) << 8)
}

// Write16 writes a 16-bit value to APU registers
// This is a convenience function that writes low byte then high byte
func (a *APU) Write16(offset uint16, value uint16) {
	a.Write8(offset, uint8(value&0xFF))
	a.Write8(offset+1, uint8(value>>8))
}

// updatePhaseIncrement updates the phase increment for a channel
// Called whenever the frequency is updated
func (a *APU) updatePhaseIncrement(channel int) {
	ch := &a.Channels[channel]
	freq := float64(ch.Frequency)

	// Calculate phase increment: how much to advance phase per sample
	// Formula: (frequency / sampleRate) * 2π
	// This gives us the phase advance per sample for the desired frequency
	if a.SampleRate == 0 {
		if a.Logger != nil {
			a.Logger.LogAPUf(debug.LogLevelError, "SampleRate is 0!")
		}
		return
	}
	ch.PhaseIncrement = (freq / float64(a.SampleRate)) * 2.0 * math.Pi
	if freq > 0 && ch.PhaseIncrement == 0 {
		if a.Logger != nil {
			a.Logger.LogAPUf(debug.LogLevelError, "PhaseIncrement is 0 for frequency %f Hz!", freq)
		}
	}
}

// GenerateSample generates a single audio sample
// This is called 44,100 times per second (once per sample)
func (a *APU) GenerateSample() float32 {
	var sample float32 = 0.0

	for i := 0; i < 4; i++ {
		ch := &a.Channels[i]
		if !ch.Enabled {
			continue
		}

		// Debug: Log if channel is enabled but has no phase increment
		if ch.PhaseIncrement == 0 && ch.Frequency > 0 {
			// This shouldn't happen, but log it once
			if a.debugFrameCount < 5 && a.Logger != nil {
				a.Logger.LogAPUf(debug.LogLevelWarning,
					"Channel %d enabled with frequency %d Hz but PhaseIncrement is 0! (SampleRate=%d)",
					i, ch.Frequency, a.SampleRate)
			}
		}

		// Debug logging removed - audio generation is working

		var channelSample float32

		switch ch.Waveform {
		case 0: // Sine wave
			// Smooth sine wave: sin(phase) gives -1.0 to 1.0
			channelSample = float32(math.Sin(ch.Phase))

		case 1: // Square wave
			// 50% duty cycle square wave
			if ch.Phase < math.Pi {
				channelSample = 1.0
			} else {
				channelSample = -1.0
			}

		case 2: // Sawtooth wave
			// Linear ramp from -1.0 to 1.0
			channelSample = float32((ch.Phase/(2.0*math.Pi))*2.0 - 1.0)

		case 3: // Noise (LFSR-based)
			// 15-bit Linear Feedback Shift Register
			// Polynomial: x^15 + x^14 + 1
			feedback := (ch.NoiseLFSR & 1) ^ ((ch.NoiseLFSR >> 14) & 1)
			ch.NoiseLFSR = (ch.NoiseLFSR >> 1) | (feedback << 14)
			if ch.NoiseLFSR == 0 {
				ch.NoiseLFSR = 1 // Prevent stuck at 0
			}
			// Output: MSB determines output value
			if (ch.NoiseLFSR & 1) != 0 {
				channelSample = 1.0
			} else {
				channelSample = -1.0
			}
		}

		// Apply channel volume (0-255 -> 0.0-1.0)
		volume := float32(ch.Volume) / 255.0
		channelSample *= volume

		// Debug logging removed - audio generation is working

		// Add to mix
		sample += channelSample

		// Update phase accumulator for next sample
		// Phase wraps at 2π to maintain continuity
		ch.Phase += ch.PhaseIncrement
		if ch.Phase >= 2.0*math.Pi {
			ch.Phase -= 2.0 * math.Pi
		}
		// Note: We use subtraction instead of modulo for better floating-point precision
	}

	// Apply master volume
	masterVol := float32(a.MasterVolume) / 255.0
	sample *= masterVol

	// Clamp to valid range [-1.0, 1.0]
	// This prevents clipping artifacts
	if sample > 1.0 {
		sample = 1.0
	} else if sample < -1.0 {
		sample = -1.0
	}

	return sample
}

// UpdateFrame is called once per frame to update timers
// This handles note duration countdown automatically
// Note: Duration=0 means "play indefinitely" (no auto-disable)
// IMPORTANT: This runs BEFORE the CPU executes, so channel status is updated
// before the ROM can check it. This makes the completion status register work correctly.
func (a *APU) UpdateFrame() {
	// Note: Completion status is NOT cleared here anymore
	// It's cleared immediately after being read (one-shot behavior)
	// This prevents the ROM from seeing it multiple times per frame
	// Only set new completion flags if channels finish this frame

	for i := 0; i < 4; i++ {
		ch := &a.Channels[i]
		if ch.Duration > 0 {
			oldDuration := ch.Duration
			ch.Duration--
			if a.shouldLog() && i == 0 && a.Logger != nil {
				a.Logger.LogAPUf(debug.LogLevelDebug,
					"Channel %d: Duration %d -> %d (frame %d)",
					i, oldDuration, ch.Duration, a.debugFrameCount)
			}
			if ch.Duration == 0 {
				// Duration expired
				if ch.DurationMode == 1 {
					// Loop mode: reload initial duration and continue playing
					if ch.InitialDuration > 0 {
						ch.Duration = ch.InitialDuration
						if a.shouldLog() && a.Logger != nil {
							a.Logger.LogAPUf(debug.LogLevelDebug,
								"Channel %d: Duration expired, looping (reloaded InitialDuration=%d)",
								i, ch.InitialDuration)
						}
					} else {
						// No initial duration stored - play indefinitely (duration stays at 0)
						if a.shouldLog() && a.Logger != nil {
							a.Logger.LogAPUf(debug.LogLevelDebug,
								"Channel %d: Duration expired, but no InitialDuration stored - playing indefinitely",
								i)
						}
					}
				} else {
					// Stop mode: disable channel when duration expires
					ch.Enabled = false
					// Set completion status bit so ROM can detect it
					// This flag persists for the entire frame (until next UpdateFrame clears it)
					a.ChannelCompletionStatus |= (1 << i)
					// Debug: Log when channel auto-disables
					if a.shouldLog() && a.Logger != nil {
						a.Logger.LogAPUf(debug.LogLevelInfo,
							"Channel %d: Duration expired, auto-disabled (frame %d, completion status=0x%02X)",
							i, a.debugFrameCount, a.ChannelCompletionStatus)
					}
				}
			}
		}
		// If Duration == 0, channel plays indefinitely (no countdown, no auto-disable)
	}
}

// shouldLog returns true if debug logging should be enabled
func (a *APU) shouldLog() bool {
	if !a.debugLoggingEnabled {
		return false
	}
	// Log for first 5 seconds
	elapsed := time.Since(a.debugLogStartTime)
	return elapsed < 5*time.Second
}

// GenerateSamples generates multiple audio samples
// Typically called once per frame (735 samples at 60 FPS)
func (a *APU) GenerateSamples(count int) []float32 {
	samples := make([]float32, count)
	for i := 0; i < count; i++ {
		samples[i] = a.GenerateSample()
	}
	a.debugFrameCount++
	return samples
}

// StepAPU steps the APU by a number of cycles (for clock-driven operation)
// This is called by the clock scheduler
// At ~7.67 MHz CPU and 44,100 Hz sample rate, APU runs every ~174 cycles
func (a *APU) StepAPU(cycles uint64) error {
	// Generate one sample per APU step
	// The clock scheduler calls this at the correct rate (44,100 Hz)
	// For now, we'll generate samples on-demand
	// In a full implementation, we'd buffer samples and output them
	return nil
}