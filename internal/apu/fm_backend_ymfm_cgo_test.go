//go:build ymfm_cgo

package apu

import "testing"

func TestYMFMBackendAutoDefault(t *testing.T) {
	t.Setenv("NCDX_YM_BACKEND", "")
	fm := NewFMOPM(nil)
	if fm.backend == nil {
		t.Fatalf("expected YMFM backend when using default auto mode under ymfm_cgo build")
	}
}

func TestYMFMBackendSelectionViaEnv(t *testing.T) {
	t.Setenv("NCDX_YM_BACKEND", "ymfm")
	fm := NewFMOPM(nil)
	if fm.backend == nil {
		t.Fatalf("expected YMFM backend when NCDX_YM_BACKEND=ymfm")
	}

	fm.Write8(FMRegControl, 0x01)
	fm.Write8(FMRegAddr, 0x22) // LFO control
	fm.Write8(FMRegData, 0x00)
	_ = fm.Read8(FMRegStatus)
	fm.Step(256)
	_ = fm.GenerateSampleFixed()
}

func TestYMFMBackendUpperPortViaMixRegisters(t *testing.T) {
	t.Setenv("NCDX_YM_BACKEND", "ymfm")
	fm := NewFMOPM(nil)
	if fm.backend == nil {
		t.Fatalf("expected YMFM backend when NCDX_YM_BACKEND=ymfm")
	}
	fm.Write8(FMRegControl, 0x01)

	// Map FMRegMixL/FMRegMixR to YM2608 upper address/data ports.
	fm.Write8(FMRegMixL, 0x10) // ADPCM-B control reg select
	fm.Write8(FMRegMixR, 0x00)

	if got := fm.Read8(FMRegMixL); got != 0x10 {
		t.Fatalf("FMRegMixL readback mismatch: got=0x%02X want=0x10", got)
	}
}

func TestYMFMBackendLegacyOverride(t *testing.T) {
	t.Setenv("NCDX_YM_BACKEND", "legacy")
	fm := NewFMOPM(nil)
	if fm.backend != nil {
		t.Fatalf("expected legacy in-tree backend when NCDX_YM_BACKEND=legacy")
	}
}
