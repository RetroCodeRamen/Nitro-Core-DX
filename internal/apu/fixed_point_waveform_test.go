package apu

import "testing"

func TestGenerateSampleFixedSawScaling(t *testing.T) {
	apu := NewAPU(44100, nil)
	apu.MasterVolume = 255
	ch := &apu.Channels[0]
	ch.Enabled = true
	ch.Waveform = 2 // saw
	ch.Volume = 255
	ch.PhaseIncrementFixed = 0

	// Quarter-cycle should be roughly mid-negative (around -16384), not hard-clipped.
	ch.PhaseFixed = 0x40000000
	s := apu.GenerateSampleFixed()
	if s < -20000 || s > -12000 {
		t.Fatalf("unexpected saw sample at quarter-cycle: got %d, want around -16384", s)
	}

	// Three-quarter cycle should be roughly mid-positive (around +16384).
	ch.PhaseFixed = 0xC0000000
	s = apu.GenerateSampleFixed()
	if s < 12000 || s > 20000 {
		t.Fatalf("unexpected saw sample at three-quarter-cycle: got %d, want around +16384", s)
	}
}
