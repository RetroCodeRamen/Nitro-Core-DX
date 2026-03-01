package emulator

import (
	"fmt"
	"testing"

	"nitro-core-dx/internal/debug"
)

// TestCPUToPPUCommunication tests if CPU can write to PPU through the bus
func TestCPUToPPUCommunication(t *testing.T) {
	logger := debug.NewLogger(1000)
	emu := NewEmulatorWithLogger(logger)

	fmt.Printf("=== CPU → Bus → PPU Communication Test ===\n\n")

	// Test 1: Write to PPU CGRAM_ADDR through CPU
	fmt.Printf("1. Testing CPU write to PPU CGRAM_ADDR...\n")
	
	// Simulate: MOV R0, #0x8012  (CGRAM_ADDR address)
	// Then: MOV [R0], #0x11      (Write 0x11 to CGRAM_ADDR)
	
	// Set up CPU state manually
	emu.CPU.SetRegister(0, 0x8012) // R0 = CGRAM_ADDR address
	emu.CPU.State.DBR = 0           // Data bank = 0 (for I/O)
	
	// Write through bus (simulating MOV [R0], #0x11)
	emu.Bus.Write8(0, 0x8012, 0x11)
	
	// Verify PPU received the write
	if emu.PPU.CGRAMAddr != 0x11 {
		t.Errorf("PPU CGRAM_ADDR not set: got 0x%02X, expected 0x11", emu.PPU.CGRAMAddr)
	} else {
		fmt.Printf("   ✅ PPU CGRAM_ADDR = 0x%02X\n", emu.PPU.CGRAMAddr)
	}

	// Test 2: Write to PPU OAM_ADDR
	fmt.Printf("\n2. Testing CPU write to PPU OAM_ADDR...\n")
	emu.Bus.Write8(0, 0x8014, 0x00) // OAM_ADDR = sprite 0
	if emu.PPU.OAMAddr != 0x00 {
		t.Errorf("PPU OAM_ADDR not set: got %d, expected 0", emu.PPU.OAMAddr)
	} else {
		fmt.Printf("   ✅ PPU OAM_ADDR = %d\n", emu.PPU.OAMAddr)
	}

	// Test 3: Write sprite data to OAM
	fmt.Printf("\n3. Testing sprite data write to OAM...\n")
	emu.Bus.Write8(0, 0x8015, 100)  // X low
	emu.Bus.Write8(0, 0x8015, 0x00) // X high
	emu.Bus.Write8(0, 0x8015, 100)  // Y
	emu.Bus.Write8(0, 0x8015, 0x00) // Tile
	emu.Bus.Write8(0, 0x8015, 0x01) // Attributes
	emu.Bus.Write8(0, 0x8015, 0x03) // Control
	
	if emu.PPU.OAM[0] != 100 || emu.PPU.OAM[5] != 0x03 {
		t.Errorf("OAM not written correctly: OAM[0]=%d, OAM[5]=0x%02X", emu.PPU.OAM[0], emu.PPU.OAM[5])
	} else {
		fmt.Printf("   ✅ Sprite data written: OAM[0]=%d, OAM[5]=0x%02X\n", emu.PPU.OAM[0], emu.PPU.OAM[5])
	}

	// Test 4: Write to VRAM
	fmt.Printf("\n4. Testing VRAM write...\n")
	emu.Bus.Write8(0, 0x800E, 0x00) // VRAM_ADDR_L
	emu.Bus.Write8(0, 0x800F, 0x00) // VRAM_ADDR_H
	emu.Bus.Write8(0, 0x8010, 0x11) // VRAM_DATA
	
	if emu.PPU.VRAM[0] != 0x11 {
		t.Errorf("VRAM not written: VRAM[0]=0x%02X, expected 0x11", emu.PPU.VRAM[0])
	} else {
		fmt.Printf("   ✅ VRAM[0] = 0x%02X\n", emu.PPU.VRAM[0])
	}
}

// TestROMExecution tests if ROM actually executes
func TestROMExecution(t *testing.T) {
	logger := debug.NewLogger(1000)
	emu := NewEmulatorWithLogger(logger)

	// Create minimal ROM data
	romData := make([]uint8, 64)
		// Create minimal ROM data
		romData = make([]uint8, 64)
		// ROM header
		romData[0] = 0x52 // 'R'
		romData[1] = 0x4D // 'M'
		romData[2] = 0x43 // 'C'
		romData[3] = 0x46 // 'F'
		romData[4] = 0x01 // Version
		romData[5] = 0x00
		romData[6] = 0x20 // Size = 32 bytes
		romData[7] = 0x00
		romData[8] = 0x00
		romData[9] = 0x00
		romData[10] = 0x01 // Entry bank = 1
		romData[11] = 0x00
		romData[12] = 0x00 // Entry offset = 0x8000
		romData[13] = 0x80
		
		// Minimal ROM code: just NOP loop
		// NOP = 0x0000
		// JMP back = 0xD000 + offset
		romData[32] = 0x00 // NOP low
		romData[33] = 0x00 // NOP high
		romData[34] = 0xD0 // JMP
		romData[35] = 0x00
	romData[36] = 0xFD // Offset = -3 (jump back)
	romData[37] = 0xFF

	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	fmt.Printf("=== ROM Execution Test ===\n\n")
	fmt.Printf("ROM loaded: Entry Bank=%d, Offset=0x%04X\n", emu.CPU.State.PCBank, emu.CPU.State.PCOffset)
	fmt.Printf("CPU PCBank: %d, PCOffset: 0x%04X\n", emu.CPU.State.PCBank, emu.CPU.State.PCOffset)

	// Start emulator
	emu.Start()

	// Execute a few instructions
	fmt.Printf("\nExecuting CPU instructions...\n")
	initialCycles := emu.CPU.State.Cycles
	
	// Step CPU for a few cycles
	for i := 0; i < 10; i++ {
		if err := emu.CPU.StepCPU(1); err != nil {
			fmt.Printf("  CPU step error: %v\n", err)
			break
		}
	}
	
	cyclesExecuted := emu.CPU.State.Cycles - initialCycles
	fmt.Printf("  Cycles executed: %d\n", cyclesExecuted)
	fmt.Printf("  CPU PCBank: %d, PCOffset: 0x%04X\n", emu.CPU.State.PCBank, emu.CPU.State.PCOffset)
	
	if cyclesExecuted == 0 {
		t.Errorf("CPU did not execute any instructions!")
	}
}

// TestFrameTimingIssue tests if PPU clears buffer at wrong time
func TestFrameTimingIssue(t *testing.T) {
	logger := debug.NewLogger(1000)
	emu := NewEmulatorWithLogger(logger)

	// Set up sprite directly
	ppu := emu.PPU
	ppu.CGRAM[0x11*2] = 0xFF
	ppu.CGRAM[0x11*2+1] = 0x7F
	for i := 0; i < 128; i++ {
		ppu.VRAM[i] = 0x11
	}
	ppu.OAM[0] = 100
	ppu.OAM[1] = 0x00
	ppu.OAM[2] = 100
	ppu.OAM[3] = 0x00
	ppu.OAM[4] = 0x01
	ppu.OAM[5] = 0x03

	fmt.Printf("=== Frame Timing Issue Test ===\n\n")

	// Don't start emulator (no ROM loaded, would cause error)
	// Instead, directly step PPU
	
	// Run one frame by stepping PPU directly
	fmt.Printf("Running one PPU frame directly...\n")
	cyclesPerFrame := uint64(220 * 360)
	if err := emu.PPU.StepPPU(cyclesPerFrame); err != nil {
		t.Fatalf("StepPPU error: %v", err)
	}

	// Immediately check buffer
	buffer1 := emu.GetOutputBuffer()
	whitePixels1 := 0
	for y := 100; y < 116 && y < 200; y++ {
		for x := 100; x < 116 && x < 320; x++ {
			if buffer1[y*320+x] != 0x000000 {
				whitePixels1++
			}
		}
	}
	fmt.Printf("After frame 1: %d white pixels\n", whitePixels1)

	// Run another frame
	if err := emu.RunFrame(); err != nil {
		t.Fatalf("RunFrame error: %v", err)
	}

	// Check buffer again
	buffer2 := emu.GetOutputBuffer()
	whitePixels2 := 0
	for y := 100; y < 116 && y < 200; y++ {
		for x := 100; x < 116 && x < 320; x++ {
			if buffer2[y*320+x] != 0x000000 {
				whitePixels2++
			}
		}
	}
	fmt.Printf("After frame 2: %d white pixels\n", whitePixels2)

	if whitePixels1 == 0 && whitePixels2 == 0 {
		t.Errorf("Sprite not rendered in either frame!")
	} else if whitePixels1 > 0 && whitePixels2 == 0 {
		t.Errorf("Sprite rendered in frame 1 but not frame 2 - timing issue!")
	} else if whitePixels1 == 0 && whitePixels2 > 0 {
		fmt.Printf("⚠️  Sprite rendered in frame 2 but not frame 1 - first frame issue\n")
	} else {
		fmt.Printf("✅ Sprite rendered consistently\n")
	}
}
