package emulator

import (
	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/apu"
)

// APUAdapter adapts APU to the debug.APUStateReader interface
type APUAdapter struct {
	apu *apu.APU
}

// GetChannelState returns the state of a channel
func (a *APUAdapter) GetChannelState(channel int) (enabled bool, frequency uint16, volume uint8, waveform uint8, duration uint16) {
	if a.apu == nil || channel < 0 || channel >= 4 {
		return false, 0, 0, 0, 0
	}
	ch := &a.apu.Channels[channel]
	return ch.Enabled, ch.Frequency, ch.Volume, ch.Waveform, ch.Duration
}

// GetMasterVolume returns the master volume
func (a *APUAdapter) GetMasterVolume() uint8 {
	if a.apu == nil {
		return 0
	}
	return a.apu.MasterVolume
}

// NewAPUAdapter creates a new APU adapter
func NewAPUAdapter(apu *apu.APU) debug.APUStateReader {
	return &APUAdapter{apu: apu}
}
