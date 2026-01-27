package main

import (
	"fmt"
	"os"

	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/emulator"
)

// Test ROM execution and PPU writes
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <rom.rom>")
		os.Exit(1)
	}

	romPath := os.Args[1]
	romData, err := os.ReadFile(romPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading ROM: %v\n", err)
		os.Exit(1)
	}

	logger := debug.NewLogger(1000)
	logger.SetComponentEnabled(debug.ComponentCPU, true)
	logger.SetComponentEnabled(debug.ComponentPPU, true)
	logger.SetComponentEnabled(debug.ComponentSystem, true)

	emu := emulator.NewEmulatorWithLogger(logger)
	
	fmt.Printf("=== ROM Execution Test ===\n\n")
	fmt.Printf("Loading ROM: %s (%d bytes)\n", romPath, len(romData))
	
	if err := emu.LoadROM(romData); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ROM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ROM loaded: Entry Bank=%d, Offset=0x%04X\n", emu.CPU.State.PCBank, emu.CPU.State.PCOffset)
	fmt.Printf("CPU state: PCBank=%d, PCOffset=0x%04X, DBR=%d\n", 
		emu.CPU.State.PCBank, emu.CPU.State.PCOffset, emu.CPU.State.DBR)

	// Check PPU state before execution
	fmt.Printf("\nPPU state before execution:\n")
	fmt.Printf("  OAM[0-5]: [%d, %d, %d, %d, 0x%02X, 0x%02X]\n",
		emu.PPU.OAM[0], emu.PPU.OAM[1], emu.PPU.OAM[2], 
		emu.PPU.OAM[3], emu.PPU.OAM[4], emu.PPU.OAM[5])
	fmt.Printf("  VRAM[0]: 0x%02X\n", emu.PPU.VRAM[0])
	fmt.Printf("  CGRAM[0x22]: 0x%02X, CGRAM[0x23]: 0x%02X\n", 
		emu.PPU.CGRAM[0x22], emu.PPU.CGRAM[0x23])

	// Start emulator
	emu.Start()

	// Run a few frames and check if ROM executes
	fmt.Printf("\nRunning 5 frames...\n")
	for frame := 0; frame < 5; frame++ {
		if err := emu.RunFrame(); err != nil {
			fmt.Printf("Frame %d error: %v\n", frame, err)
			break
		}
		
		// Check PPU state after frame
		fmt.Printf("\nAfter frame %d:\n", frame)
		fmt.Printf("  CPU cycles: %d\n", emu.GetCPUCyclesPerFrame())
		fmt.Printf("  CPU PCBank: %d, PCOffset: 0x%04X\n", 
			emu.CPU.State.PCBank, emu.CPU.State.PCOffset)
		fmt.Printf("  PPU OAM[0-5]: [%d, %d, %d, %d, 0x%02X, 0x%02X]\n",
			emu.PPU.OAM[0], emu.PPU.OAM[1], emu.PPU.OAM[2], 
			emu.PPU.OAM[3], emu.PPU.OAM[4], emu.PPU.OAM[5])
		fmt.Printf("  PPU VRAM[0]: 0x%02X\n", emu.PPU.VRAM[0])
		fmt.Printf("  PPU CGRAM[0x22]: 0x%02X, CGRAM[0x23]: 0x%02X\n", 
			emu.PPU.CGRAM[0x22], emu.PPU.CGRAM[0x23])
		fmt.Printf("  PPU FrameComplete: %v\n", emu.PPU.FrameComplete)
		
		// Check output buffer
		buffer := emu.GetOutputBuffer()
		whitePixels := 0
		for y := 100; y < 116 && y < 200; y++ {
			for x := 100; x < 116 && x < 320; x++ {
				if buffer[y*320+x] != 0x000000 {
					whitePixels++
				}
			}
		}
		fmt.Printf("  White pixels in sprite area: %d\n", whitePixels)
		
		// If CPU didn't execute, stop
		if emu.GetCPUCyclesPerFrame() == 0 {
			fmt.Printf("  ⚠️  CPU did not execute any cycles!\n")
		}
	}
	
	fmt.Printf("\n=== Test Complete ===\n")
	fmt.Printf("If OAM/VRAM/CGRAM changed, ROM is executing.\n")
	fmt.Printf("If white pixels > 0, sprite rendering works.\n")
}
