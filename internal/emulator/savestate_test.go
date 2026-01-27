package emulator

import (
	"testing"
)

// TestSaveLoadState tests that save/load state works correctly
func TestSaveLoadState(t *testing.T) {
	emu := NewEmulator()
	
	// Create minimal ROM
	romData := make([]uint8, 64)
	romData[0] = 0x52 // "RMCF"
	romData[1] = 0x4D
	romData[2] = 0x43
	romData[3] = 0x46
	romData[4] = 0x01 // Version 1
	romData[6] = 0x20 // ROM size 32
	romData[10] = 0x01 // Entry bank 1
	romData[12] = 0x00 // Entry offset 0x8000
	romData[13] = 0x80
	
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	
	// Modify some state
	emu.CPU.State.R0 = 0x1234
	emu.CPU.State.R1 = 0x5678
	emu.CPU.State.PCBank = 2
	emu.CPU.State.PCOffset = 0x9000
	emu.Bus.WRAM[0x1000] = 0xAB
	emu.Bus.WRAM[0x1001] = 0xCD
	emu.PPU.VRAM[0x2000] = 0xEF
	emu.PPU.CGRAM[0] = 0x12
	emu.PPU.FrameCounter = 42
	emu.APU.MasterVolume = 128
	emu.APU.Channels[0].Frequency = 440
	emu.APU.Channels[0].Volume = 200
	emu.Input.Controller1Buttons = 0x1234
	
	// Save state
	savedData, err := emu.SaveState()
	if err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}
	
	if len(savedData) == 0 {
		t.Fatal("SaveState returned empty data")
	}
	
	// Modify state to verify it changes
	emu.CPU.State.R0 = 0x9999
	emu.CPU.State.R1 = 0x8888
	emu.Bus.WRAM[0x1000] = 0xFF
	emu.PPU.VRAM[0x2000] = 0x00
	emu.PPU.FrameCounter = 999
	emu.APU.MasterVolume = 255
	emu.APU.Channels[0].Frequency = 880
	
	// Load state
	if err := emu.LoadState(savedData); err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}
	
	// Verify state was restored
	if emu.CPU.State.R0 != 0x1234 {
		t.Errorf("R0 not restored: expected 0x1234, got 0x%04X", emu.CPU.State.R0)
	}
	if emu.CPU.State.R1 != 0x5678 {
		t.Errorf("R1 not restored: expected 0x5678, got 0x%04X", emu.CPU.State.R1)
	}
	if emu.CPU.State.PCBank != 2 {
		t.Errorf("PCBank not restored: expected 2, got %d", emu.CPU.State.PCBank)
	}
	if emu.CPU.State.PCOffset != 0x9000 {
		t.Errorf("PCOffset not restored: expected 0x9000, got 0x%04X", emu.CPU.State.PCOffset)
	}
	if emu.Bus.WRAM[0x1000] != 0xAB {
		t.Errorf("WRAM[0x1000] not restored: expected 0xAB, got 0x%02X", emu.Bus.WRAM[0x1000])
	}
	if emu.Bus.WRAM[0x1001] != 0xCD {
		t.Errorf("WRAM[0x1001] not restored: expected 0xCD, got 0x%02X", emu.Bus.WRAM[0x1001])
	}
	if emu.PPU.VRAM[0x2000] != 0xEF {
		t.Errorf("VRAM[0x2000] not restored: expected 0xEF, got 0x%02X", emu.PPU.VRAM[0x2000])
	}
	if emu.PPU.CGRAM[0] != 0x12 {
		t.Errorf("CGRAM[0] not restored: expected 0x12, got 0x%02X", emu.PPU.CGRAM[0])
	}
	if emu.PPU.FrameCounter != 42 {
		t.Errorf("FrameCounter not restored: expected 42, got %d", emu.PPU.FrameCounter)
	}
	if emu.APU.MasterVolume != 128 {
		t.Errorf("MasterVolume not restored: expected 128, got %d", emu.APU.MasterVolume)
	}
	if emu.APU.Channels[0].Frequency != 440 {
		t.Errorf("Channel 0 Frequency not restored: expected 440, got %d", emu.APU.Channels[0].Frequency)
	}
	if emu.APU.Channels[0].Volume != 200 {
		t.Errorf("Channel 0 Volume not restored: expected 200, got %d", emu.APU.Channels[0].Volume)
	}
	if emu.Input.Controller1Buttons != 0x1234 {
		t.Errorf("Controller1Buttons not restored: expected 0x1234, got 0x%04X", emu.Input.Controller1Buttons)
	}
}

