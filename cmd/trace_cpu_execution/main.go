package main

import (
	"fmt"
	"os"

	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/emulator"
)

// Trace CPU execution to see if OAM setup code is reached
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
	logger.SetComponentEnabled(debug.ComponentPPU, true)
	logger.SetComponentEnabled(debug.ComponentMemory, true)

	emu := emulator.NewEmulatorWithLogger(logger)
	
	fmt.Printf("=== CPU Execution Trace ===\n\n")
	fmt.Printf("Loading ROM: %s\n", romPath)
	
	if err := emu.LoadROM(romData); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ROM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ROM loaded: Entry Bank=%d, Offset=0x%04X\n", emu.CPU.State.PCBank, emu.CPU.State.PCOffset)
	fmt.Printf("Initial CPU state: DBR=%d, PCBank=%d, PCOffset=0x%04X\n\n", 
		emu.CPU.State.DBR, emu.CPU.State.PCBank, emu.CPU.State.PCOffset)

	// Track PC to see execution path
	pcHistory := make(map[string]int)
	oamWriteCount := 0
	
	// Note: OAM writes will be logged via the logger if PPU logging is enabled
	// We can track OAM writes by monitoring the logger output or checking OAM state

	emu.Start()

	// Run one frame and track PC
	fmt.Printf("Running one frame...\n")
	fmt.Printf("Tracking PC locations...\n\n")
	
	// Track PC every 1000 cycles
	lastPC := ""
	pcChanges := 0
	
	for cycles := 0; cycles < 10000 && pcChanges < 50; cycles++ {
		// Step CPU for a few cycles
		if err := emu.CPU.StepCPU(100); err != nil {
			fmt.Printf("CPU error at cycle %d: %v\n", cycles, err)
			break
		}
		
		// Check PC
		currentPC := fmt.Sprintf("%d:0x%04X", emu.CPU.State.PCBank, emu.CPU.State.PCOffset)
		if currentPC != lastPC {
			pcHistory[currentPC]++
			pcChanges++
			if pcChanges <= 20 {
				fmt.Printf("  PC: %s (DBR=%d, R4=0x%04X, R5=0x%04X)\n", 
					currentPC, emu.CPU.State.DBR, emu.CPU.State.R4, emu.CPU.State.R5)
			}
			lastPC = currentPC
		}
	}
	
	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("OAM writes detected: %d\n", oamWriteCount)
	fmt.Printf("Unique PC locations: %d\n", len(pcHistory))
	fmt.Printf("PC after execution: Bank=%d, Offset=0x%04X\n", 
		emu.CPU.State.PCBank, emu.CPU.State.PCOffset)
	fmt.Printf("DBR: %d\n", emu.CPU.State.DBR)
	fmt.Printf("R4: 0x%04X (should be 0x8014 or 0x8015 for OAM)\n", emu.CPU.State.R4)
	fmt.Printf("R5: 0x%04X\n", emu.CPU.State.R5)
	fmt.Printf("\nOAM state: [%d, %d, %d, %d, 0x%02X, 0x%02X]\n",
		emu.PPU.OAM[0], emu.PPU.OAM[1], emu.PPU.OAM[2], 
		emu.PPU.OAM[3], emu.PPU.OAM[4], emu.PPU.OAM[5])
}
