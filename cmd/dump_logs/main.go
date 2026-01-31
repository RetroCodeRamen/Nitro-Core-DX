package main

import (
	"flag"
	"fmt"
	"os"

	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/emulator"
)

func main() {
	romPath := flag.String("rom", "", "Path to ROM file")
	logFile := flag.String("out", "logs.txt", "Output log file")
	maxFrames := flag.Int("frames", 60, "Run for N frames then dump logs")
	flag.Parse()

	if *romPath == "" {
		fmt.Println("Usage: dump_logs -rom <rom> [-out <file>] [-frames <N>]")
		os.Exit(1)
	}

	// Read ROM
	romData, err := os.ReadFile(*romPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading ROM: %v\n", err)
		os.Exit(1)
	}

	// Create emulator with logging enabled
	logger := debug.NewLogger(50000)
	logger.SetComponentEnabled(debug.ComponentPPU, true)
	logger.SetMinLevel(debug.LogLevelDebug)
	emu := emulator.NewEmulatorWithLogger(logger)

	// Load ROM
	if err := emu.LoadROM(romData); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ROM: %v\n", err)
		os.Exit(1)
	}

	// Start emulator
	emu.Start()

	// Run for specified number of frames
	fmt.Printf("Running ROM for %d frames...\n", *maxFrames)
	for i := 0; i < *maxFrames; i++ {
		if err := emu.RunFrame(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running frame: %v\n", err)
			break
		}
	}

	// Get all PPU logs
	entries := logger.GetEntries()
	ppuEntries := []debug.LogEntry{}
	for _, entry := range entries {
		if entry.Component == debug.ComponentPPU {
			ppuEntries = append(ppuEntries, entry)
		}
	}

	// Write to file
	file, err := os.Create(*logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating log file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	fmt.Fprintf(file, "PPU Logs from %s (%d entries)\n", *romPath, len(ppuEntries))
	fmt.Fprintf(file, "===========================================\n\n")

	for _, entry := range ppuEntries {
		fmt.Fprintf(file, "%s\n", entry.Format())
	}

	fmt.Printf("Dumped %d PPU log entries to %s\n", len(ppuEntries), *logFile)
}
