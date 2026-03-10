package emulator

import (
	"math"
	"testing"

	"nitro-core-dx/internal/apu"
	"nitro-core-dx/internal/cpu"
)

// TestResetReloadsEntryPoint tests that emulator.Reset() reloads entry point correctly
func TestResetReloadsEntryPoint(t *testing.T) {
	emu := NewEmulator()

	// Create a minimal ROM with entry point at bank 1, offset 0x8000
	romData := make([]uint8, 64)
	// Magic: "RMCF"
	romData[0] = 0x52
	romData[1] = 0x4D
	romData[2] = 0x43
	romData[3] = 0x46
	// Version: 1
	romData[4] = 0x01
	romData[5] = 0x00
	// ROM Size: 32 bytes
	romData[6] = 0x20
	romData[7] = 0x00
	romData[8] = 0x00
	romData[9] = 0x00
	// Entry Bank: 1
	romData[10] = 0x01
	romData[11] = 0x00
	// Entry Offset: 0x8000
	romData[12] = 0x00
	romData[13] = 0x80

	// Load ROM
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// Verify entry point is set
	if emu.CPU.State.PCBank != 1 {
		t.Errorf("Expected PCBank=1 after LoadROM, got %d", emu.CPU.State.PCBank)
	}
	if emu.CPU.State.PCOffset != 0x8000 {
		t.Errorf("Expected PCOffset=0x8000 after LoadROM, got 0x%04X", emu.CPU.State.PCOffset)
	}

	// Modify PC to simulate execution
	emu.CPU.State.PCBank = 2
	emu.CPU.State.PCOffset = 0x9000

	// Call Reset() - should reload entry point
	emu.Reset()

	// Verify entry point is reloaded correctly
	if emu.CPU.State.PCBank != 1 {
		t.Errorf("After Reset(): Expected PCBank=1, got %d (entry point should be reloaded)", emu.CPU.State.PCBank)
	}
	if emu.CPU.State.PCOffset != 0x8000 {
		t.Errorf("After Reset(): Expected PCOffset=0x8000, got 0x%04X (entry point should be reloaded)", emu.CPU.State.PCOffset)
	}
}

func TestFMTimerIRQWiresToCPUInterrupt(t *testing.T) {
	t.Setenv("NCDX_YM_BACKEND", "legacy")
	emu := NewEmulator()

	// Enable FM extension and program Timer A to a short phase-1 period.
	emu.APU.Write8(apu.FMExtensionOffsetBase+apu.FMRegControl, 0x01) // enable FM
	emu.APU.Write8(apu.FMExtensionOffsetBase+apu.FMRegAddr, 0x10)
	emu.APU.Write8(apu.FMExtensionOffsetBase+apu.FMRegData, 0xFF)
	emu.APU.Write8(apu.FMExtensionOffsetBase+apu.FMRegAddr, 0x11)
	emu.APU.Write8(apu.FMExtensionOffsetBase+apu.FMRegData, 0x03)
	emu.APU.Write8(apu.FMExtensionOffsetBase+apu.FMRegAddr, 0x14)
	emu.APU.Write8(apu.FMExtensionOffsetBase+apu.FMRegData, 0x11) // start A + IRQ enable A

	// Clear any prior pending interrupt and step the APU until expiry.
	emu.CPU.State.InterruptPending = cpu.INT_NONE
	if err := emu.APU.StepAPU(64); err != nil {
		t.Fatalf("APU step failed: %v", err)
	}

	if emu.CPU.State.InterruptPending != cpu.INT_TIMER {
		t.Fatalf("expected CPU INT_TIMER pending, got %d", emu.CPU.State.InterruptPending)
	}
}

func TestRunFrameUsesFixedPointAPUPath(t *testing.T) {
	emu := NewEmulator()

	romData := make([]uint8, 64)
	romData[0] = 0x52
	romData[1] = 0x4D
	romData[2] = 0x43
	romData[3] = 0x46
	romData[4] = 0x01
	romData[6] = 0x20
	romData[10] = 0x01
	romData[12] = 0x00
	romData[13] = 0x80
	romData[32] = 0x00 // NOP low
	romData[33] = 0x00 // NOP high
	romData[34] = 0xD0 // JMP opcode
	romData[35] = 0x00
	romData[36] = 0xFD // relative offset low
	romData[37] = 0xFF // relative offset high

	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()
	emu.Clock.CPUStep = func(cycles uint64) error {
		emu.CPU.State.Cycles += uint32(cycles)
		return nil
	}

	emu.APU.Write8(0x00, 0xB8) // freq low
	emu.APU.Write8(0x01, 0x01) // freq high => 440 Hz
	emu.APU.Write8(0x02, 0xFF) // volume
	emu.APU.Write8(0x03, 0x03) // enable + square wave

	// Corrupt the deprecated float-phase state. RunFrame should still use the
	// fixed-point path and generate audio samples from PhaseFixed/PhaseIncrementFixed.
	emu.APU.Channels[0].Phase = math.NaN()
	emu.APU.Channels[0].PhaseIncrement = 0

	if err := emu.RunFrame(); err != nil {
		t.Fatalf("RunFrame failed: %v", err)
	}

	nonZero := false
	for _, sample := range emu.AudioSampleBuffer {
		if sample != 0 {
			nonZero = true
			break
		}
	}
	if !nonZero {
		t.Fatal("expected non-zero audio from fixed-point APU path during RunFrame")
	}
}
