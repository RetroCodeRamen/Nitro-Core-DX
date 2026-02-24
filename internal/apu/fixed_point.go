package apu

import (
	"nitro-core-dx/internal/debug"
)

// Fixed-point audio generation
// All internal calculations use fixed-point arithmetic
// Only the host adapter converts to float32

// Phase accumulator: 32-bit unsigned integer
// Phase wraps at 2^32 (represents 0 to 2π)
// Phase increment is calculated as: (frequency * 2^32) / sampleRate
const (
	// Phase is represented as 32-bit unsigned integer
	// 0 = 0 radians, 2^32 = 2π radians
	PhaseMax = uint32(0xFFFFFFFF) // 2^32 - 1

	// Fixed-point scale for volume (8-bit volume -> 16-bit fixed point)
	VolumeScale = 256 // Volume is 0-255, scale to 0-65535 for 16-bit fixed point
)

// updatePhaseIncrementFixed updates the phase increment using fixed-point
func (a *APU) updatePhaseIncrementFixed(channel int) {
	ch := &a.Channels[channel]
	freq := uint32(ch.Frequency)

	if a.SampleRate == 0 {
		if a.Logger != nil {
			a.Logger.LogAPUf(debug.LogLevelError, "SampleRate is 0!")
		}
		return
	}

	// Calculate phase increment: (frequency * 2^32) / sampleRate
	// This gives us the phase advance per sample in fixed-point
	// Using 64-bit intermediate to avoid overflow
	// Note: PhaseMax+1 = 2^32, which overflows uint32, so we use 0x100000000 directly
	phaseIncrement64 := (uint64(freq) * 0x100000000) / uint64(a.SampleRate)
	ch.PhaseIncrementFixed = uint32(phaseIncrement64)
}

// GenerateSampleFixed generates a single audio sample using fixed-point arithmetic
// Returns a 16-bit signed integer sample (-32768 to 32767)
// Host adapter converts this to float32
// This is the preferred method for clock-driven operation
func (a *APU) GenerateSampleFixed() int16 {
	var sample int32 = 0

	for i := 0; i < 4; i++ {
		ch := &a.Channels[i]
		if !ch.Enabled {
			continue
		}

		var channelSample int32

		switch ch.Waveform {
		case 0: // Sine wave
			// Convert phase (0-2^32) to sine value using lookup table or approximation
			// For now, use a simple approximation: sin(x) ≈ x for small x, or use lookup
			// Using a simple polynomial approximation for sine
			phaseNormalized := uint16(ch.PhaseFixed >> 16) // Get upper 16 bits (0-65535 represents 0-2π)
			channelSample = int32(a.sineFixed(phaseNormalized))

		case 1: // Square wave
			// 50% duty cycle: output 1 if phase < π, else -1
			if ch.PhaseFixed < (PhaseMax / 2) {
				channelSample = 32767 // Max positive
			} else {
				channelSample = -32768 // Max negative
			}

		case 2: // Sawtooth wave
			// Linear ramp from -1 to 1
			// Phase 0-2^32 maps to -32768 to 32767
			// Use the upper 16 bits so the output is in the same amplitude range
			// as the other waveforms before mixing/clamping.
			channelSample = int32(int64(ch.PhaseFixed>>16) - 32768)

		case 3: // Noise (LFSR-based)
			// 15-bit Linear Feedback Shift Register
			feedback := (ch.NoiseLFSR & 1) ^ ((ch.NoiseLFSR >> 14) & 1)
			ch.NoiseLFSR = (ch.NoiseLFSR >> 1) | (feedback << 14)
			if ch.NoiseLFSR == 0 {
				ch.NoiseLFSR = 1 // Prevent stuck at 0
			}
			// Output: MSB determines output value
			if (ch.NoiseLFSR & 1) != 0 {
				channelSample = 32767
			} else {
				channelSample = -32768
			}
		}

		// Apply channel volume (0-255 -> multiply by volume, divide by 255)
		// Using fixed-point: (sample * volume) / 255
		channelSample = (channelSample * int32(ch.Volume)) / 255

		// Add to mix
		sample += channelSample

		// Update phase accumulator for next sample
		ch.PhaseFixed += ch.PhaseIncrementFixed
		// Phase wraps automatically (uint32 overflow)
	}

	if a.FM != nil {
		sample += int32(a.FM.GenerateSampleFixed())
	}

	// Apply master volume
	sample = (sample * int32(a.MasterVolume)) / 255

	// Clamp to valid range [-32768, 32767]
	if sample > 32767 {
		sample = 32767
	} else if sample < -32768 {
		sample = -32768
	}

	return int16(sample)
}

// sineFixed approximates sine using fixed-point arithmetic
// Input: phase (0-65535 represents 0-2π)
// Output: fixed-point sine value (-32768 to 32767 represents -1.0 to 1.0)
func (a *APU) sineFixed(phase uint16) int16 {
	// Simple approximation: use a lookup table or polynomial
	// For now, use a simple polynomial approximation
	// sin(x) ≈ x - x^3/6 + x^5/120 (normalized to 0-2π range)

	// Normalize phase to 0-1 range (fixed point: 0-65535 -> 0-1)
	// Then map to -π to π for better approximation
	phaseNormalized := int32(phase)
	if phaseNormalized >= 32768 {
		// Second half of cycle: map to -π to 0
		phaseNormalized = phaseNormalized - 65536
	}

	// Use polynomial approximation: sin(x) ≈ x - x^3/6
	// Scale appropriately for fixed-point
	x := phaseNormalized >> 8 // Scale down for calculation
	x3 := (x * x * x) >> 16   // x^3 with scaling
	result := x - (x3 / 6)

	// Scale to output range
	result = result << 7 // Scale up to -32768 to 32767 range

	// Clamp
	if result > 32767 {
		result = 32767
	} else if result < -32768 {
		result = -32768
	}

	return int16(result)
}

// ConvertFixedToFloat converts a fixed-point sample to float32
// This is the only place where float conversion happens (host adapter)
func ConvertFixedToFloat(sample int16) float32 {
	// Convert 16-bit signed integer (-32768 to 32767) to float32 (-1.0 to 1.0)
	return float32(sample) / 32768.0
}
