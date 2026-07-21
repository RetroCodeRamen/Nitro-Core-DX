package apu

import "testing"

// programAudibleVoice sets up the same channel-0 "audible subset" patch used
// by TestFMOPMAudibleSubsetGeneratesSamples, so two FMOPM instances can be
// compared sample-for-sample under different FMRegVolume settings.
func programAudibleVoice(fm *FMOPM) {
	fm.Write8(FMRegControl, 0x01) // enable
	writeOPMReg(fm, 0x20, 0xC0|0x08|0x01)
	writeOPMReg(fm, 0x28, 36)
	writeOPMReg(fm, 0x30, 0x00)
	writeOPMReg(fm, 0x38, 0x50)
	writeOPMReg(fm, 0x40, 0x02)
	writeOPMReg(fm, 0x58, 0x01)
	writeOPMReg(fm, 0x60, 0x40)
	writeOPMReg(fm, 0x78, 0x10)
	writeOPMReg(fm, fmOPMRegKeyOn, 0x78)
}

// TestFMRegVolumeScalesOutput verifies FMRegVolume (music.set_volume's
// target register) scales every generated sample by volume/255, applied
// uniformly regardless of which FM synthesis path produced the raw sample.
func TestFMRegVolumeScalesOutput(t *testing.T) {
	full := NewFMOPM(nil)
	full.SampleRate = 44100
	programAudibleVoice(full)

	half := NewFMOPM(nil)
	half.SampleRate = 44100
	programAudibleVoice(half)
	half.Write8(FMRegVolume, 0x80)

	sawNonZero := false
	for i := 0; i < 64; i++ {
		fullSample := full.GenerateSampleFixed()
		halfSample := half.GenerateSampleFixed()
		want := int16((int32(fullSample) * 0x80) / 255)
		if halfSample != want {
			t.Fatalf("sample %d: volume=0x80 want %d (full=%d scaled), got %d", i, want, fullSample, halfSample)
		}
		if fullSample != 0 {
			sawNonZero = true
		}
	}
	if !sawNonZero {
		t.Fatal("test voice generated only zeros; scaling comparison is not meaningful")
	}
}

// TestFMRegVolumeZeroSilences verifies FMRegVolume=0 (music.stop's
// eventual target, and the low end of a fade) produces silence.
func TestFMRegVolumeZeroSilences(t *testing.T) {
	fm := NewFMOPM(nil)
	fm.SampleRate = 44100
	programAudibleVoice(fm)
	fm.Write8(FMRegVolume, 0x00)

	for i := 0; i < 64; i++ {
		if s := fm.GenerateSampleFixed(); s != 0 {
			t.Fatalf("sample %d: want 0 at volume=0x00, got %d", i, s)
		}
	}
}

// TestFMRegVolumeReadback verifies the register round-trips through the
// host interface (Write8 then Read8), matching the convention every other
// FM host register follows.
func TestFMRegVolumeReadback(t *testing.T) {
	fm := NewFMOPM(nil)
	fm.Write8(FMRegVolume, 0x42)
	if got := fm.Read8(FMRegVolume); got != 0x42 {
		t.Fatalf("FMRegVolume readback: want 0x42, got 0x%02X", got)
	}
}
