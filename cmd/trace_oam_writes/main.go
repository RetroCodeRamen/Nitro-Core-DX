package main

import (
	"fmt"
	"os"

	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/emulator"
)

// Trace OAM writes from ROM execution
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

	// Create logger with PPU logging enabled
	logger := debug.NewLogger(10000)
	logger.SetComponentEnabled(debug.ComponentPPU, true)
	logger.SetComponentEnabled(debug.ComponentCPU, true)
	logger.SetComponentEnabled(debug.ComponentMemory, true)
	logger.SetMinLevel(debug.LogLevelDebug)

	emu := emulator.NewEmulatorWithLogger(logger)

	fmt.Printf("=== OAM Write Tracing ===\n\n")
	fmt.Printf("Loading ROM: %s (%d bytes)\n", romPath, len(romData))

	if err := emu.LoadROM(romData); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ROM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ROM loaded: Entry Bank=%d, Offset=0x%04X\n\n", emu.CPU.State.PCBank, emu.CPU.State.PCOffset)

	// Add custom logging for OAM writes
	// We'll hook into the PPU Write8 function to log OAM writes
	fmt.Printf("Initial PPU state:\n")
	fmt.Printf("  OAM[0-5]: [%d, %d, %d, %d, 0x%02X, 0x%02X]\n",
		emu.PPU.OAM[0], emu.PPU.OAM[1], emu.PPU.OAM[2],
		emu.PPU.OAM[3], emu.PPU.OAM[4], emu.PPU.OAM[5])
	fmt.Printf("  OAMAddr: %d, OAMByteIndex: %d\n", emu.PPU.OAMAddr, emu.PPU.OAMByteIndex)

	// Start emulator
	emu.Start()

	// Track OAM writes
	oamWrites := 0

	// Run a few frames and monitor OAM writes
	fmt.Printf("\nRunning 3 frames and monitoring OAM writes...\n\n")

	for frame := 0; frame < 3; frame++ {
		fmt.Printf("--- Frame %d ---\n", frame)

		// Check OAM before frame
		fmt.Printf("Before frame: OAM[0-5] = [%d, %d, %d, %d, 0x%02X, 0x%02X]\n",
			emu.PPU.OAM[0], emu.PPU.OAM[1], emu.PPU.OAM[2],
			emu.PPU.OAM[3], emu.PPU.OAM[4], emu.PPU.OAM[5])

		// Count OAM writes before frame
		_ = oamWrites // Track if needed

		// Run frame
		if err := emu.RunFrame(); err != nil {
			fmt.Printf("Frame %d error: %v\n", frame, err)
			break
		}

		// Check OAM after frame
		fmt.Printf("After frame: OAM[0-5] = [%d, %d, %d, %d, 0x%02X, 0x%02X]\n",
			emu.PPU.OAM[0], emu.PPU.OAM[1], emu.PPU.OAM[2],
			emu.PPU.OAM[3], emu.PPU.OAM[4], emu.PPU.OAM[5])
		fmt.Printf("  OAMAddr: %d, OAMByteIndex: %d\n", emu.PPU.OAMAddr, emu.PPU.OAMByteIndex)
		fmt.Printf("  CPU cycles: %d\n", emu.GetCPUCyclesPerFrame())
		fmt.Printf("  CPU PC: Bank=%d, Offset=0x%04X\n", emu.CPU.State.PCBank, emu.CPU.State.PCOffset)

		// Check if OAM changed
		if emu.PPU.OAM[0] != 0 || emu.PPU.OAM[5] != 0 {
			fmt.Printf("  ✅ OAM was written!\n")
		} else {
			fmt.Printf("  ❌ OAM still all zeros\n")
		}
		fmt.Printf("\n")
	}

	fmt.Printf("=== Analysis ===\n")
	fmt.Printf("If OAM stays all zeros, the ROM code that writes to OAM is either:\n")
	fmt.Printf("  1. Not executing (code path issue)\n")
	fmt.Printf("  2. Writing to wrong address\n")
	fmt.Printf("  3. Being overwritten/cleared\n")
	fmt.Printf("  4. Timing issue (writes happen but get cleared)\n")
}
