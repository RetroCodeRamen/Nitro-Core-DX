package apu

import (
	"math"
)

// APU represents the Audio Processing Unit
// It implements the memory.IOHandler interface
type APU struct {
	Channels    [4]AudioChannel
	MasterVolume uint8
	SampleRate   uint32
}

// AudioChannel represents an audio channel
type AudioChannel struct {
	Frequency      uint16
	Volume         uint8
	Enabled        bool
	Waveform       uint8 // 0=sine, 1=square, 2=saw, 3=noise
	Phase          float64
	PhaseIncrement float64
	NoiseLFSR      uint16 // For noise waveform
}

// NewAPU creates a new APU instance
func NewAPU(sampleRate uint32) *APU {
	return &APU{
		SampleRate:    sampleRate,
		MasterVolume:  255,
	}
}

// Read8 reads an 8-bit value from APU registers
func (a *APU) Read8(offset uint16) uint8 {
	// APU registers are mostly write-only
	return 0
}

// Write8 writes an 8-bit value to APU registers
func (a *APU) Write8(offset uint16, value uint8) {
	channel := int((offset / 4) & 0x3)
	reg := offset & 0x3

	switch reg {
	case 0: // FREQ_LOW
		a.Channels[channel].Frequency = (a.Channels[channel].Frequency & 0xFF00) | uint16(value)
		a.updatePhaseIncrement(channel)
	case 1: // FREQ_HIGH
		a.Channels[channel].Frequency = (a.Channels[channel].Frequency & 0x00FF) | (uint16(value) << 8)
		a.updatePhaseIncrement(channel)
	case 2: // VOLUME
		a.Channels[channel].Volume = value
	case 3: // CONTROL
		a.Channels[channel].Enabled = (value & 0x01) != 0
		if channel < 3 {
			a.Channels[channel].Waveform = (value >> 1) & 0x3
		} else {
			// Channel 3: noise mode
			if (value & 0x02) != 0 {
				a.Channels[channel].Waveform = 3 // Noise
			} else {
				a.Channels[channel].Waveform = 1 // Square
			}
		}
	}

	if offset == 0x10 { // MASTER_VOLUME
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
func (a *APU) Write16(offset uint16, value uint16) {
	a.Write8(offset, uint8(value&0xFF))
	a.Write8(offset+1, uint8(value>>8))
}

// updatePhaseIncrement updates the phase increment for a channel
func (a *APU) updatePhaseIncrement(channel int) {
	freq := float64(a.Channels[channel].Frequency)
	phaseIncrement := (freq / float64(a.SampleRate)) * 2.0 * math.Pi
	a.Channels[channel].PhaseIncrement = phaseIncrement
}

// GenerateSample generates a single audio sample
func (a *APU) GenerateSample() float32 {
	var sample float32 = 0.0

	for i := 0; i < 4; i++ {
		ch := &a.Channels[i]
		if !ch.Enabled {
			continue
		}

		var channelSample float32

		switch ch.Waveform {
		case 0: // Sine
			channelSample = float32(math.Sin(ch.Phase))
		case 1: // Square
			if ch.Phase < math.Pi {
				channelSample = 1.0
			} else {
				channelSample = -1.0
			}
		case 2: // Saw
			channelSample = float32((ch.Phase / (2.0 * math.Pi)) * 2.0 - 1.0)
		case 3: // Noise
			// LFSR-based noise
			feedback := (ch.NoiseLFSR & 1) ^ ((ch.NoiseLFSR >> 14) & 1)
			ch.NoiseLFSR = (ch.NoiseLFSR >> 1) | (feedback << 14)
			if ch.NoiseLFSR == 0 {
				ch.NoiseLFSR = 1
			}
			if (ch.NoiseLFSR & 1) != 0 {
				channelSample = 1.0
			} else {
				channelSample = -1.0
			}
		}

		// Apply volume
		volume := float32(ch.Volume) / 255.0
		channelSample *= volume

		// Add to mix
		sample += channelSample

		// Update phase
		ch.Phase += ch.PhaseIncrement
		if ch.Phase >= 2.0*math.Pi {
			ch.Phase -= 2.0 * math.Pi
		}
	}

	// Apply master volume
	masterVol := float32(a.MasterVolume) / 255.0
	sample *= masterVol

	// Clamp to [-1.0, 1.0]
	if sample > 1.0 {
		sample = 1.0
	} else if sample < -1.0 {
		sample = -1.0
	}

	return sample
}

// GenerateSamples generates multiple audio samples
func (a *APU) GenerateSamples(count int) []float32 {
	samples := make([]float32, count)
	for i := 0; i < count; i++ {
		samples[i] = a.GenerateSample()
	}
	return samples
}

