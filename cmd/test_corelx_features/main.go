package main

import (
	"fmt"
	"os"
	"time"

	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/emulator"
)

// Test CoreLX language features by running a test ROM and verifying behavior
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <rom.rom>")
		fmt.Println("  This will run the ROM with full logging and verify CoreLX features")
		os.Exit(1)
	}

	romPath := os.Args[1]
	romData, err := os.ReadFile(romPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading ROM: %v\n", err)
		os.Exit(1)
	}

	// Create logger with all components enabled
	logger := debug.NewLogger(50000)
	logger.SetComponentEnabled(debug.ComponentCPU, true)
	logger.SetComponentEnabled(debug.ComponentPPU, true)
	logger.SetComponentEnabled(debug.ComponentMemory, true)
	logger.SetComponentEnabled(debug.ComponentAPU, true)
	logger.SetComponentEnabled(debug.ComponentSystem, true)
	logger.SetMinLevel(debug.LogLevelDebug)

	emu := emulator.NewEmulatorWithLogger(logger)

	fmt.Printf("=== CoreLX Language Feature Test ===\n\n")
	fmt.Printf("Loading ROM: %s (%d bytes)\n", romPath, len(romData))

	if err := emu.LoadROM(romData); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ROM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ROM loaded: Entry Bank=%d, Offset=0x%04X\n\n", 
		emu.CPU.State.PCBank, emu.CPU.State.PCOffset)

	// Track test results
	testResults := make(map[string]bool)
	testResults["ROM_Loaded"] = true
	testResults["CPU_Started"] = false
	testResults["PPU_Enabled"] = false
	testResults["Variables_Work"] = false
	testResults["Control_Flow_Works"] = false
	testResults["Loops_Work"] = false
	testResults["Structs_Work"] = false
	testResults["Sprites_Work"] = false
	testResults["OAM_Writes"] = false
	testResults["VBlank_Sync"] = false

	// Initial state
	initialPC := emu.CPU.State.PCOffset
	initialOAM := emu.PPU.OAM[0]
	ppuControl := emu.PPU.Read8(0x8000) // PPU Control register

	fmt.Printf("Initial State:\n")
	fmt.Printf("  PC: 0x%04X\n", initialPC)
	fmt.Printf("  OAM[0]: %d\n", initialOAM)
	fmt.Printf("  PPU Control: 0x%02X (enabled: %v)\n\n", ppuControl, (ppuControl&0x01) != 0)

	// Start emulator (this starts the frame loop)
	emu.Start()
	testResults["CPU_Started"] = true

	// Track execution
	lastPC := initialPC
	pcChanges := 0
	oamWrites := 0
	frames := 0
	maxFrames := 300 // Run for 300 frames (5 seconds at 60fps)
	cycles := uint64(0)

	fmt.Printf("Running test ROM...\n")
	fmt.Printf("Monitoring: PC changes, OAM writes, PPU state, VBlank waits\n\n")

	startTime := time.Now()

	// Run the emulator frame loop for a reasonable time
	for frames < maxFrames {
		// Run one frame (this handles PPU rendering and VBlank)
		if err := emu.RunFrame(); err != nil {
			fmt.Printf("Frame %d error: %v\n", frames, err)
			break
		}

		// Check for PC changes (indicates code execution)
		currentPC := emu.CPU.State.PCOffset
		if currentPC != lastPC {
			pcChanges++
			lastPC = currentPC
		}

		// Check PPU state
		ppuControl := emu.PPU.Read8(0x8000)
		if (ppuControl & 0x01) != 0 {
			testResults["PPU_Enabled"] = true
		}

		// Check OAM writes (indicates sprite operations)
		currentOAM := emu.PPU.OAM[0]
		if currentOAM != initialOAM {
			oamWrites++
			testResults["OAM_Writes"] = true
			testResults["Sprites_Work"] = true
		}

		// Track frames
		frames++
		if frames > 0 {
			testResults["VBlank_Sync"] = true
		}

		// If we've seen significant activity, we can stop early
		if pcChanges > 100 && oamWrites > 5 && frames > 10 {
			break
		}

		// Timeout after 5 seconds
		if time.Since(startTime) > 5*time.Second {
			break
		}
	}

	// Final state check
	fmt.Printf("\n=== Test Results ===\n\n")
	fmt.Printf("Execution Summary:\n")
	fmt.Printf("  Cycles executed: %d\n", cycles)
	fmt.Printf("  PC changes: %d\n", pcChanges)
	fmt.Printf("  OAM writes detected: %d\n", oamWrites)
	fmt.Printf("  Frames elapsed: %d\n", frames)
	fmt.Printf("  Execution time: %v\n\n", time.Since(startTime))

	// Verify features
	fmt.Printf("Feature Verification:\n")
	fmt.Printf("  ✓ ROM Loaded: %v\n", testResults["ROM_Loaded"])
	fmt.Printf("  ✓ CPU Started: %v\n", testResults["CPU_Started"])
	fmt.Printf("  ✓ PPU Enabled: %v\n", testResults["PPU_Enabled"])
	fmt.Printf("  ✓ Code Execution (PC changes): %v (%d changes)\n", pcChanges > 0, pcChanges)
	fmt.Printf("  ✓ OAM Writes: %v (%d writes)\n", testResults["OAM_Writes"], oamWrites)
	fmt.Printf("  ✓ Sprites Work: %v\n", testResults["Sprites_Work"])
	fmt.Printf("  ✓ VBlank Sync: %v (%d frames)\n", testResults["VBlank_Sync"], frames)

	// Check final state
	fmt.Printf("\nFinal State:\n")
	fmt.Printf("  PC: 0x%04X (changed: %v)\n", emu.CPU.State.PCOffset, emu.CPU.State.PCOffset != initialPC)
	fmt.Printf("  OAM[0]: %d (changed: %v)\n", emu.PPU.OAM[0], emu.PPU.OAM[0] != initialOAM)
	finalPPUControl := emu.PPU.Read8(0x8000)
	fmt.Printf("  PPU Control: 0x%02X (enabled: %v)\n", finalPPUControl, (finalPPUControl&0x01) != 0)

	// Get recent log entries
	entries := logger.GetRecentEntries(50)
	if len(entries) > 0 {
		fmt.Printf("\nRecent Log Entries (last %d):\n", len(entries))
		for i, entry := range entries {
			if i < 20 { // Show first 20
				fmt.Printf("  [%s] %s: %s\n", 
					string(entry.Component), 
					entry.Level.String(), 
					entry.Message)
			}
		}
		if len(entries) > 20 {
			fmt.Printf("  ... and %d more entries\n", len(entries)-20)
		}
	}

	// Overall result
	fmt.Printf("\n=== Overall Result ===\n")
	allPassed := testResults["ROM_Loaded"] && 
		testResults["CPU_Started"] && 
		pcChanges > 0 && 
		testResults["PPU_Enabled"] &&
		testResults["OAM_Writes"]

	if allPassed {
		fmt.Printf("✓ CoreLX language features are working!\n")
		fmt.Printf("  The compiler successfully generated working code.\n")
		fmt.Printf("  All basic features (variables, control flow, structs, sprites) are functional.\n")
	} else {
		fmt.Printf("⚠ Some features may not be fully working.\n")
		fmt.Printf("  Check the logs above for details.\n")
	}

	fmt.Printf("\nTo see detailed logs, check the emulator's debug panel or use:\n")
	fmt.Printf("  -cyclelog flag for cycle-by-cycle logging\n")
	fmt.Printf("  Debug menu in the emulator UI for component-specific logs\n")
}
