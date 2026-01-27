package main

import (
	"fmt"
	"os"

	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/emulator"
)

// Trace VRAM loop execution to see why it doesn't exit
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

	logger := debug.NewLogger(10000)
	logger.SetComponentEnabled(debug.ComponentCPU, true)
	logger.SetMinLevel(debug.LogLevelDebug)

	emu := emulator.NewEmulatorWithLogger(logger)
	
	fmt.Printf("=== VRAM Loop Execution Trace ===\n\n")
	fmt.Printf("Loading ROM: %s\n", romPath)
	
	if err := emu.LoadROM(romData); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ROM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ROM loaded: Entry Bank=%d, Offset=0x%04X\n\n", emu.CPU.State.PCBank, emu.CPU.State.PCOffset)

	emu.Start()

	// Track loop iterations
	loopIterations := 0
	lastR6 := uint16(0xFFFF)
	lastPC := uint16(0xFFFF)
	lastZFlag := false
	
	// VRAM loop should be around PC 0x802C-0x8046
	// OAM setup starts at 0x804C
	vramLoopStart := uint16(0x802C)
	vramLoopEnd := uint16(0x8046)
	oamSetupStart := uint16(0x804C)
	
	fmt.Printf("VRAM loop range: 0x%04X - 0x%04X\n", vramLoopStart, vramLoopEnd)
	fmt.Printf("OAM setup starts at: 0x%04X\n\n", oamSetupStart)
	fmt.Printf("Tracing execution...\n\n")

	// Execute instructions one at a time and track the loop
	for i := 0; i < 200; i++ {
		// Execute one instruction
		if err := emu.CPU.ExecuteInstruction(); err != nil {
			fmt.Printf("Error at iteration %d: %v\n", i, err)
			break
		}
		
		currentPC := emu.CPU.State.PCOffset
		currentR6 := emu.CPU.State.R6
		currentZFlag := emu.CPU.GetFlag(0) // FlagZ = 0
		
		// Check if we're in the VRAM loop
		inLoop := currentPC >= vramLoopStart && currentPC <= vramLoopEnd
		reachedOAM := currentPC >= oamSetupStart
		
		// Detect loop iteration (PC returns to loop start)
		if inLoop && lastPC >= vramLoopEnd && currentPC == vramLoopStart {
			loopIterations++
			fmt.Printf("--- Loop iteration #%d ---\n", loopIterations)
		}
		
		// Log significant events
		if i < 50 || inLoop || reachedOAM || currentR6 != lastR6 || currentZFlag != lastZFlag {
			fmt.Printf("  [%3d] PC=0x%04X, R6=%d, Z=%v", i, currentPC, currentR6, currentZFlag)
			
			if inLoop {
				fmt.Printf(" [IN LOOP]")
			}
			if reachedOAM {
				fmt.Printf(" [REACHED OAM SETUP!]")
			}
			if currentR6 != lastR6 {
				fmt.Printf(" [R6 CHANGED: %d -> %d]", lastR6, currentR6)
			}
			if currentZFlag != lastZFlag {
				fmt.Printf(" [Z FLAG CHANGED: %v -> %v]", lastZFlag, currentZFlag)
			}
			fmt.Printf("\n")
		}
		
		// Stop if we reach OAM setup
		if reachedOAM {
			fmt.Printf("\n✅ Successfully exited VRAM loop and reached OAM setup!\n")
			fmt.Printf("   Loop iterations: %d\n", loopIterations)
			fmt.Printf("   Final R6: %d\n", currentR6)
			fmt.Printf("   Final Z flag: %v\n", currentZFlag)
			break
		}
		
		// Stop if we've done too many iterations without progress
		if i > 100 && loopIterations > 20 {
			fmt.Printf("\n⚠️  Loop appears to be infinite (20+ iterations)\n")
			fmt.Printf("   Current PC: 0x%04X\n", currentPC)
			fmt.Printf("   Current R6: %d\n", currentR6)
			fmt.Printf("   Current Z flag: %v\n", currentZFlag)
			break
		}
		
		lastPC = currentPC
		lastR6 = currentR6
		lastZFlag = currentZFlag
	}
	
	fmt.Printf("\n=== Final State ===\n")
	fmt.Printf("PC: Bank=%d, Offset=0x%04X\n", emu.CPU.State.PCBank, emu.CPU.State.PCOffset)
	fmt.Printf("R6: %d\n", emu.CPU.State.R6)
	fmt.Printf("R7: %d\n", emu.CPU.State.R7)
	fmt.Printf("Z flag: %v\n", emu.CPU.GetFlag(0))
	fmt.Printf("Loop iterations: %d\n", loopIterations)
}
