package apu

import (
	"testing"
)

func writeOPMReg(fm *FMOPM, addr, value uint8) {
	fm.Write8(FMRegAddr, addr)
	fm.Write8(FMRegData, value)
}

func TestFMOPMHostRegisterFile(t *testing.T) {
	t.Setenv("NCDX_YM_BACKEND", "ymfm")
	fm := NewFMOPM(nil)

	fm.Write8(FMRegAddr, 0x34)
	fm.Write8(FMRegData, 0x56)
	if got := fm.Read8(FMRegStatus); got&FMStatusBusy == 0 {
		t.Fatalf("busy flag not set after FM register write, status=0x%02X", got)
	}

	if got := fm.Read8(FMRegAddr); got != 0x34 {
		t.Fatalf("FMRegAddr: got 0x%02X, want 0x34", got)
	}

	// Data port reads from the currently selected OPM register.
	fm.Write8(FMRegAddr, 0x34)
	if got := fm.Read8(FMRegData); got != 0x56 {
		t.Fatalf("FMRegData readback: got 0x%02X, want 0x56", got)
	}
}

func TestFMOPMControlAndReset(t *testing.T) {
	t.Setenv("NCDX_YM_BACKEND", "ymfm")
	fm := NewFMOPM(nil)

	fm.Write8(FMRegAddr, 0x10)
	fm.Write8(FMRegData, 0xAA)
	fm.Write8(FMRegControl, 0x03) // enable + mute

	if !fm.Enabled {
		t.Fatalf("Enabled = false, want true")
	}
	if !fm.Muted {
		t.Fatalf("Muted = false, want true")
	}
	if got := fm.Read8(FMRegControl); got != 0x03 {
		t.Fatalf("FMRegControl: got 0x%02X, want 0x03", got)
	}

	// Reset is write-one-shot via bit 7 and should clear the register shadow.
	fm.Write8(FMRegControl, 0x83) // reset request + enable + mute

	if got := fm.Read8(FMRegData); got != 0x00 {
		t.Fatalf("FM register shadow not cleared on reset, got 0x%02X", got)
	}
	if got := fm.Read8(FMRegControl); got != 0x03 {
		t.Fatalf("reset bit should not latch: got 0x%02X, want 0x03", got)
	}
}

func TestAPURoutesFMExtensionOffsets(t *testing.T) {
	t.Setenv("NCDX_YM_BACKEND", "ymfm")
	apu := NewAPU(44100, nil)

	// CPU-visible 0x9100 maps to APU offset 0x0100.
	apu.Write8(FMExtensionOffsetBase+FMRegAddr, 0x22)
	apu.Write8(FMExtensionOffsetBase+FMRegData, 0x99)

	apu.Write8(FMExtensionOffsetBase+FMRegAddr, 0x22)
	if got := apu.Read8(FMExtensionOffsetBase + FMRegData); got != 0x99 {
		t.Fatalf("APU FM data readback: got 0x%02X, want 0x99", got)
	}

	// Legacy APU registers must still work unchanged.
	apu.Write8(0x20, 0x7F) // MASTER_VOLUME
	if got := apu.Read8(0x20); got != 0x7F {
		t.Fatalf("legacy MASTER_VOLUME broke: got 0x%02X, want 0x7F", got)
	}
}

func TestFMOPMTimerRegisterProgrammingSmoke(t *testing.T) {
	t.Setenv("NCDX_YM_BACKEND", "ymfm")
	fm := NewFMOPM(nil)
	fm.Write8(FMRegControl, 0x01) // enable extension

	writeOPMReg(fm, fmOPMRegTimerAHi, 0xFF)
	writeOPMReg(fm, fmOPMRegTimerALo, 0x03)
	writeOPMReg(fm, fmOPMRegTimerCtrl, 0x11)
	writeOPMReg(fm, fmOPMRegTimerB, 0xFF)
	writeOPMReg(fm, fmOPMRegTimerCtrl, 0x13) // start A+B with Timer A IRQ enable

	fm.Step(2048)
	_ = fm.Read8(FMRegStatus)
}

func TestAPUFMTimerProgrammingSmoke(t *testing.T) {
	t.Setenv("NCDX_YM_BACKEND", "ymfm")
	apu := NewAPU(44100, nil)
	apu.Write8(FMExtensionOffsetBase+FMRegControl, 0x01) // enable FM extension

	apu.Write8(FMExtensionOffsetBase+FMRegAddr, fmOPMRegTimerAHi)
	apu.Write8(FMExtensionOffsetBase+FMRegData, 0xFF)
	apu.Write8(FMExtensionOffsetBase+FMRegAddr, fmOPMRegTimerALo)
	apu.Write8(FMExtensionOffsetBase+FMRegData, 0x03)
	apu.Write8(FMExtensionOffsetBase+FMRegAddr, fmOPMRegTimerCtrl)
	apu.Write8(FMExtensionOffsetBase+FMRegData, 0x11)

	if err := apu.StepAPU(4096); err != nil {
		t.Fatalf("StepAPU failed: %v", err)
	}
	_ = apu.Read8(FMExtensionOffsetBase + FMRegStatus)
}

func TestFMOPMAudibleSubsetGeneratesSamples(t *testing.T) {
	t.Setenv("NCDX_YM_BACKEND", "ymfm")
	fm := NewFMOPM(nil)
	fm.SampleRate = 44100
	fm.Write8(FMRegControl, 0x01) // enable

	// Program channel 0 using the phase-2 OPM-lite subset:
	// 0x20: pan+alg/feedback, 0x28: keycode, 0x30: keyfrac, 0x38: PMS
	// 0x40/0x58: mod/carrier MUL, 0x60/0x78: mod/carrier TL
	writeOPMReg(fm, 0x20, 0xC0|0x08|0x01) // pan both + light feedback + alt algo
	writeOPMReg(fm, 0x28, 36)             // C2-ish in phase-2 mapping
	writeOPMReg(fm, 0x30, 0x00)
	writeOPMReg(fm, 0x38, 0x50)          // moderate PMS
	writeOPMReg(fm, 0x40, 0x02)          // mod MUL
	writeOPMReg(fm, 0x58, 0x01)          // carrier MUL
	writeOPMReg(fm, 0x60, 0x40)          // mod TL (quieter)
	writeOPMReg(fm, 0x78, 0x10)          // carrier TL (louder)
	writeOPMReg(fm, fmOPMRegKeyOn, 0x78) // channel 0 + nonzero op mask => key on

	nonZero := false
	for i := 0; i < 128; i++ {
		if s := fm.GenerateSampleFixed(); s != 0 {
			nonZero = true
			break
		}
	}
	if !nonZero {
		t.Fatalf("FM audible subset generated only zeros after key-on")
	}
}

func TestFMOPMAudibleSubsetKeyOffStopsOutput(t *testing.T) {
	t.Setenv("NCDX_YM_BACKEND", "ymfm")
	fm := NewFMOPM(nil)
	fm.SampleRate = 44100
	fm.Write8(FMRegControl, 0x01) // enable

	writeOPMReg(fm, 0x20, 0xC0)
	writeOPMReg(fm, 0x28, 48)
	writeOPMReg(fm, 0x58, 0x01)
	writeOPMReg(fm, 0x78, 0x00) // max carrier level
	writeOPMReg(fm, fmOPMRegKeyOn, 0x78)

	_ = fm.GenerateSampleFixed()         // advance once while on
	writeOPMReg(fm, fmOPMRegKeyOn, 0x00) // channel 0, zero opmask => key off

	for i := 0; i < 8; i++ {
		if got := fm.GenerateSampleFixed(); got != 0 {
			t.Fatalf("expected silence after key-off, got sample %d on iteration %d", got, i)
		}
	}
}
