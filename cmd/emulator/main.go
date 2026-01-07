package main

import (
	"flag"
	"fmt"
	"os"

	"nitro-core-dx/internal/cpu"
	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/emulator"
	"nitro-core-dx/internal/ui"
)

func main() {
	romPath := flag.String("rom", "", "Path to ROM file")
	unlimited := flag.Bool("unlimited", false, "Run at unlimited speed (no frame limit)")
	scale := flag.Int("scale", 3, "Display scale (1-6)")
	enableLogging := flag.Bool("log", false, "Enable logging (disabled by default)")
	flag.Parse()

	if *romPath == "" {
		fmt.Println("Usage: nitro-core-dx -rom <path-to-rom>")
		fmt.Println("  -rom <path>      Path to ROM file (.rom)")
		fmt.Println("  -unlimited       Run at unlimited speed")
		fmt.Println("  -scale <1-6>     Display scale (default: 3)")
		fmt.Println("  -log             Enable logging (disabled by default)")
		os.Exit(1)
	}

	// Validate scale
	if *scale < 1 || *scale > 6 {
		fmt.Fprintf(os.Stderr, "Error: scale must be between 1 and 6\n")
		os.Exit(1)
	}

	// Read ROM file
	romData, err := os.ReadFile(*romPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading ROM file: %v\n", err)
		os.Exit(1)
	}

	// Create emulator with optional logging
	var emu *emulator.Emulator
	if *enableLogging {
		// Enable logging - create logger and enable all components
		logger := debug.NewLogger(10000)
		// Enable all components when -log flag is used
		logger.SetComponentEnabled(debug.ComponentCPU, true)
		logger.SetComponentEnabled(debug.ComponentPPU, true)
		logger.SetComponentEnabled(debug.ComponentAPU, true)
		logger.SetComponentEnabled(debug.ComponentMemory, true)
		logger.SetComponentEnabled(debug.ComponentInput, true)
		logger.SetComponentEnabled(debug.ComponentUI, true)
		logger.SetComponentEnabled(debug.ComponentSystem, true)
		// Set CPU log level to Instructions when logging is enabled
		emu = emulator.NewEmulatorWithLogger(logger)
		// Get CPU logger and set level
		if emu.CPU != nil && emu.CPU.Log != nil {
			if adapter, ok := emu.CPU.Log.(*cpu.CPULoggerAdapter); ok {
				adapter.SetLevel(cpu.CPULogInstructions)
			}
		}
	} else {
		// No logging - use default (all disabled)
		emu = emulator.NewEmulator()
	}

	// Load ROM
	if err := emu.LoadROM(romData); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ROM: %v\n", err)
		os.Exit(1)
	}

	// Set frame limit
	emu.SetFrameLimit(!*unlimited)

	fmt.Println("Nitro-Core-DX Emulator")
	fmt.Println("====================")
	fmt.Printf("ROM loaded: %s\n", *romPath)
	fmt.Printf("Frame limit: %v\n", !*unlimited)
	fmt.Printf("Display scale: %dx\n", *scale)
	fmt.Println("\nStarting emulation...")
	fmt.Println("\nControls:")
	fmt.Println("  Arrow Keys / WASD - Move block")
	fmt.Println("  Z / W - A button (change block color)")
	fmt.Println("  X - B button (change background color)")
	fmt.Println("  Space - Pause/Resume")
	fmt.Println("  Ctrl+R - Reset")
	fmt.Println("  Alt+F - Toggle fullscreen")
	fmt.Println("  ESC - Quit")

	// Create Fyne UI (with SDL2 for emulator rendering)
	uiInstance, err := ui.NewFyneUI(emu, *scale)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating UI: %v\n", err)
		os.Exit(1)
	}

	// Run UI (blocks until window is closed)
	if err := uiInstance.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "UI error: %v\n", err)
		os.Exit(1)
	}
}
