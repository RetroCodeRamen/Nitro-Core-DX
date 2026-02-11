package emulator

import (
	"testing"
)

// TestAudioTimingLongRun tests that audio timing remains accurate over long runs
// This verifies the fractional accumulator prevents drift from integer division
func TestAudioTimingLongRun(t *testing.T) {
	// Create emulator
	emu := NewEmulator()
	
	// Create minimal ROM (same as determinism test)
	romData := func() []byte {
		romData := make([]byte, 32+32)
		
		// Header
		romData[0] = 'R'
		romData[1] = 'M'
		romData[2] = 'C'
		romData[3] = 'F'
		romData[4] = 0x01
		romData[5] = 0x00
		romData[6] = 0x20 // ROM size: 32 bytes
		romData[7] = 0x00
		romData[10] = 0x01 // Entry bank: 1
		romData[11] = 0x00
		romData[12] = 0x00 // Entry offset: 0x8000
		romData[13] = 0x80
		
		// Main code at 0x8000: MOV R0, #0x1234
		romData[32] = 0x00 // MOV: 0x1100
		romData[33] = 0x11
		romData[34] = 0x34 // Immediate: 0x1234
		romData[35] = 0x12
		
		// NOP
		romData[36] = 0x00
		romData[37] = 0x00
		
		// NOP
		romData[38] = 0x00
		romData[39] = 0x00
		
		// JMP back
		romData[40] = 0x00
		romData[41] = 0xD0
		romData[42] = 0xFD // Offset: -6
		romData[43] = 0xFF
		
		// Fill rest with NOPs
		for i := 44; i < 64; i += 2 {
			romData[i] = 0x00
			romData[i+1] = 0x00
		}
		
		return romData
	}()
	
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	
	// Disable interrupts
	emu.CPU.SetFlag(4, true) // FlagI
	
	emu.Start()
	
	// Run for 60 frames (1 second at 60 FPS)
	numFrames := 60
	
	// Expected samples: 44,100 samples per second = 735 samples per frame
	// Over 60 frames: 60 × 735 = 44,100 samples
	expectedSamplesTotal := 60 * 735
	
	totalSamplesGenerated := 0
	
	for i := 0; i < numFrames; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("Frame %d error: %v", i, err)
		}
		
		// Count samples generated this frame
		samples := emu.GetAudioSamples()
		totalSamplesGenerated += len(samples)
	}
	
	// Verify we generated exactly the expected number of samples
	// Allow small tolerance (±1 sample) for rounding
	if totalSamplesGenerated < expectedSamplesTotal-1 || totalSamplesGenerated > expectedSamplesTotal+1 {
		t.Errorf("Audio timing drift detected: expected %d samples, got %d (drift: %d samples)",
			expectedSamplesTotal, totalSamplesGenerated, totalSamplesGenerated-expectedSamplesTotal)
	}
	
	// Also verify per-frame sample count is correct
	// Each frame should generate exactly 735 samples (±1 for rounding)
	expectedSamplesPerFrame := 735
	for i := 0; i < numFrames; i++ {
		// Re-run to check per-frame counts
		emu.Reset()
		emu.LoadROM(romData)
		emu.CPU.SetFlag(4, true) // Disable interrupts
		emu.Start()
		
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("Frame %d error: %v", i, err)
		}
		
		samples := emu.GetAudioSamples()
		if len(samples) < expectedSamplesPerFrame-1 || len(samples) > expectedSamplesPerFrame+1 {
			t.Errorf("Frame %d: expected %d samples, got %d", i, expectedSamplesPerFrame, len(samples))
		}
	}
}

// TestAudioTimingFractionalAccumulator tests the fractional accumulator directly
func TestAudioTimingFractionalAccumulator(t *testing.T) {
	// Test that fractional accumulator prevents drift
	// Run for 1000 frames and verify sample count is accurate
	
	emu := NewEmulator()
	
	// Create minimal ROM (same as determinism test)
	romData := func() []byte {
		romData := make([]byte, 32+32)
		romData[0] = 'R'
		romData[1] = 'M'
		romData[2] = 'C'
		romData[3] = 'F'
		romData[4] = 0x01
		romData[6] = 0x20 // ROM size: 32 bytes
		romData[10] = 0x01
		romData[13] = 0x80
		romData[32] = 0x00 // MOV: 0x1100
		romData[33] = 0x11
		romData[34] = 0x34 // Immediate: 0x1234
		romData[35] = 0x12
		romData[36] = 0x00 // NOP
		romData[37] = 0x00
		romData[38] = 0x00 // NOP
		romData[39] = 0x00
		romData[40] = 0x00 // JMP back
		romData[41] = 0xD0
		romData[42] = 0xFD // Offset: -6
		romData[43] = 0xFF
		// Fill rest with NOPs
		for i := 44; i < 64; i += 2 {
			romData[i] = 0x00
			romData[i+1] = 0x00
		}
		return romData
	}()
	
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	
	// Disable interrupts before starting
	emu.CPU.SetFlag(4, true) // FlagI = interrupt disable
	emu.Start()
	
	// Run for 1000 frames
	numFrames := 1000
	expectedSamplesTotal := 1000 * 735 // 735 samples per frame
	
	totalSamplesGenerated := 0
	
	for i := 0; i < numFrames; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("Frame %d error: %v", i, err)
		}
		
		samples := emu.GetAudioSamples()
		totalSamplesGenerated += len(samples)
	}
	
	// Verify total sample count (allow ±10 samples tolerance for 1000 frames)
	tolerance := 10
	if totalSamplesGenerated < expectedSamplesTotal-tolerance || totalSamplesGenerated > expectedSamplesTotal+tolerance {
		t.Errorf("Long-run audio timing drift: expected %d samples (±%d), got %d (drift: %d samples)",
			expectedSamplesTotal, tolerance, totalSamplesGenerated, totalSamplesGenerated-expectedSamplesTotal)
	}
}
