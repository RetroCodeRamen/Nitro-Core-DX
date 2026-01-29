package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/emulator"
)

// Interactive debugger for Nitro Core DX ROMs
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debugger <rom.rom>")
		fmt.Println("Interactive debugger for Nitro Core DX ROMs")
		os.Exit(1)
	}

	romPath := os.Args[1]
	romData, err := os.ReadFile(romPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading ROM: %v\n", err)
		os.Exit(1)
	}

	// Create logger and emulator
	logger := debug.NewLogger(10000)
	logger.SetComponentEnabled(debug.ComponentCPU, true)
	logger.SetComponentEnabled(debug.ComponentPPU, true)
	logger.SetComponentEnabled(debug.ComponentSystem, true)
	logger.SetMinLevel(debug.LogLevelDebug)

	emu := emulator.NewEmulatorWithLogger(logger)
	dbg := debug.NewDebugger()

	// Load ROM
	if err := emu.LoadROM(romData); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ROM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("=== Nitro Core DX Debugger ===\n")
	fmt.Printf("ROM loaded: %s (%d bytes)\n", romPath, len(romData))
	fmt.Printf("Entry point: Bank %d, Offset 0x%04X\n\n", emu.CPU.State.PCBank, emu.CPU.State.PCOffset)
	fmt.Printf("Type 'help' for commands\n\n")

	// Start emulator in paused state
	emu.Pause()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("(debugger) ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		cmd := strings.ToLower(parts[0])
		args := parts[1:]

		switch cmd {
		case "help", "h":
			printHelp()

		case "break", "b":
			if len(args) < 1 {
				fmt.Println("Usage: break <bank>:<offset>")
				fmt.Println("Example: break 1:0x8000")
				continue
			}
			handleBreakpoint(dbg, args[0])

		case "delete", "d":
			if len(args) < 1 {
				fmt.Println("Usage: delete <breakpoint-key>")
				fmt.Println("Use 'breakpoints' to list breakpoint keys")
				continue
			}
			if dbg.RemoveBreakpoint(args[0]) {
				fmt.Printf("Breakpoint %s removed\n", args[0])
			} else {
				fmt.Printf("Breakpoint %s not found\n", args[0])
			}

		case "breakpoints", "bp":
			printBreakpoints(dbg)

		case "enable":
			if len(args) < 1 {
				fmt.Println("Usage: enable <breakpoint-key>")
				continue
			}
			if dbg.EnableBreakpoint(args[0]) {
				fmt.Printf("Breakpoint %s enabled\n", args[0])
			} else {
				fmt.Printf("Breakpoint %s not found\n", args[0])
			}

		case "disable":
			if len(args) < 1 {
				fmt.Println("Usage: disable <breakpoint-key>")
				continue
			}
			if dbg.DisableBreakpoint(args[0]) {
				fmt.Printf("Breakpoint %s disabled\n", args[0])
			} else {
				fmt.Printf("Breakpoint %s not found\n", args[0])
			}

		case "continue", "c":
			dbg.Resume()
			emu.Resume()
			runUntilBreakpoint(emu, dbg)

		case "step", "s":
			count := 1
			if len(args) > 0 {
				if n, err := strconv.Atoi(args[0]); err == nil {
					count = n
				}
			}
			dbg.Step(count)
			emu.Resume()
			runUntilBreakpoint(emu, dbg)

		case "pause", "p":
			dbg.Pause()
			emu.Pause()
			fmt.Println("Execution paused")

		case "registers", "regs":
			printRegisters(emu)

		case "memory", "mem", "m":
			if len(args) < 1 {
				fmt.Println("Usage: memory <bank>:<offset> [count]")
				fmt.Println("Example: memory 0:0x1000 16")
				continue
			}
			handleMemory(emu, args)

		case "stack":
			printStack(emu)

		case "oam":
			printOAM(emu)

		case "ppu":
			printPPU(emu)

		case "watch", "w":
			if len(args) < 1 {
				fmt.Println("Usage: watch <expression>")
				fmt.Println("Example: watch R0")
				continue
			}
			dbg.AddWatch(strings.Join(args, " "))
			fmt.Printf("Added watch: %s\n", strings.Join(args, " "))

		case "watches":
			printWatches(dbg, emu)

		case "variables", "vars", "v":
			printVariables(dbg)

		case "callstack", "cs":
			printCallStack(dbg)

		case "run":
			emu.Start()
			fmt.Println("Emulator running (press Ctrl+C to pause)")

		case "frame", "f":
			emu.RunFrame()
			printStatus(emu)

		case "status":
			printStatus(emu)

		case "clear":
			if len(args) > 0 && args[0] == "breakpoints" {
				dbg.ClearBreakpoints()
				fmt.Println("All breakpoints cleared")
			} else if len(args) > 0 && args[0] == "watches" {
				dbg.ClearWatches()
				fmt.Println("All watches cleared")
			} else {
				fmt.Println("Usage: clear <breakpoints|watches>")
			}

		case "quit", "q", "exit":
			fmt.Println("Exiting debugger...")
			return

		default:
			fmt.Printf("Unknown command: %s\n", cmd)
			fmt.Println("Type 'help' for available commands")
		}
	}
}

func printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  break <bank>:<offset>     - Set breakpoint (e.g., break 1:0x8000)")
	fmt.Println("  delete <key>              - Delete breakpoint")
	fmt.Println("  breakpoints               - List all breakpoints")
	fmt.Println("  enable <key>              - Enable breakpoint")
	fmt.Println("  disable <key>             - Disable breakpoint")
	fmt.Println("  continue                 - Continue execution")
	fmt.Println("  step [count]              - Step N instructions (default: 1)")
	fmt.Println("  pause                    - Pause execution")
	fmt.Println("  registers                - Show CPU registers")
	fmt.Println("  memory <bank>:<offset>   - Show memory contents")
	fmt.Println("  stack                    - Show stack contents")
	fmt.Println("  oam                      - Show OAM (sprite) data")
	fmt.Println("  ppu                      - Show PPU state")
	fmt.Println("  watch <expr>              - Add watch expression")
	fmt.Println("  watches                  - Show watch expressions")
	fmt.Println("  variables                - Show tracked variables")
	fmt.Println("  callstack                - Show call stack")
	fmt.Println("  frame                    - Run one frame")
	fmt.Println("  status                   - Show emulator status")
	fmt.Println("  clear <bp|watches>        - Clear breakpoints or watches")
	fmt.Println("  quit                     - Exit debugger")
}

func handleBreakpoint(dbg *debug.Debugger, addrStr string) {
	parts := strings.Split(addrStr, ":")
	if len(parts) != 2 {
		fmt.Println("Invalid address format. Use: bank:offset")
		return
	}

	bank, err := strconv.ParseUint(parts[0], 0, 8)
	if err != nil {
		fmt.Printf("Invalid bank: %v\n", err)
		return
	}

	offset, err := strconv.ParseUint(parts[1], 0, 16)
	if err != nil {
		fmt.Printf("Invalid offset: %v\n", err)
		return
	}

	key := dbg.SetBreakpoint(uint8(bank), uint16(offset))
	fmt.Printf("Breakpoint set at %02X:%04X (key: %s)\n", uint8(bank), uint16(offset), key)
}

func printBreakpoints(dbg *debug.Debugger) {
	bps := dbg.GetAllBreakpoints()
	if len(bps) == 0 {
		fmt.Println("No breakpoints set")
		return
	}

	fmt.Println("Breakpoints:")
	for key, bp := range bps {
		status := "disabled"
		if bp.Enabled {
			status = "enabled"
		}
		fmt.Printf("  %s: %02X:%04X (%s, hit %d times)\n", key, bp.Bank, bp.Offset, status, bp.HitCount)
	}
}

func runUntilBreakpoint(emu *emulator.Emulator, dbg *debug.Debugger) {
	// Run until breakpoint or step count reached
	for {
		if err := emu.CPU.ExecuteInstruction(); err != nil {
			fmt.Printf("Execution error: %v\n", err)
			emu.Pause()
			return
		}

		// Check if we should break
		if dbg.ShouldBreak(emu.CPU.State.PCBank, emu.CPU.State.PCOffset) {
			emu.Pause()
			fmt.Printf("\nBreakpoint hit at %02X:%04X\n", emu.CPU.State.PCBank, emu.CPU.State.PCOffset)
			printStatus(emu)
			return
		}

		// Check if paused
		if dbg.IsPaused() {
			return
		}
	}
}

func printRegisters(emu *emulator.Emulator) {
	state := emu.CPU.State
	fmt.Printf("CPU Registers:\n")
	fmt.Printf("  R0: 0x%04X  R1: 0x%04X  R2: 0x%04X  R3: 0x%04X\n", state.R0, state.R1, state.R2, state.R3)
	fmt.Printf("  R4: 0x%04X  R5: 0x%04X  R6: 0x%04X  R7: 0x%04X\n", state.R4, state.R5, state.R6, state.R7)
	fmt.Printf("  PC: %02X:%04X  PBR: %02X  DBR: %02X  SP: 0x%04X\n", state.PCBank, state.PCOffset, state.PBR, state.DBR, state.SP)
	fmt.Printf("  Flags: 0x%02X (Z:%d N:%d C:%d V:%d I:%d D:%d)\n", state.Flags,
		(state.Flags>>0)&1, (state.Flags>>1)&1, (state.Flags>>2)&1,
		(state.Flags>>3)&1, (state.Flags>>4)&1, (state.Flags>>5)&1)
	fmt.Printf("  Cycles: %d\n", state.Cycles)
}

func handleMemory(emu *emulator.Emulator, args []string) {
	parts := strings.Split(args[0], ":")
	if len(parts) != 2 {
		fmt.Println("Invalid address format. Use: bank:offset")
		return
	}

	bank, err := strconv.ParseUint(parts[0], 0, 8)
	if err != nil {
		fmt.Printf("Invalid bank: %v\n", err)
		return
	}

	offset, err := strconv.ParseUint(parts[1], 0, 16)
	if err != nil {
		fmt.Printf("Invalid offset: %v\n", err)
		return
	}

	count := 16
	if len(args) > 1 {
		if n, err := strconv.Atoi(args[1]); err == nil {
			count = n
		}
	}

	fmt.Printf("Memory at %02X:%04X:\n", uint8(bank), uint16(offset))
	for i := 0; i < count; i += 16 {
		fmt.Printf("  %04X: ", uint16(offset)+uint16(i))
		for j := 0; j < 16 && i+j < count; j++ {
			val := emu.Bus.Read8(uint8(bank), uint16(offset)+uint16(i+j))
			fmt.Printf("%02X ", val)
		}
		fmt.Println()
	}
}

func printStack(emu *emulator.Emulator) {
	sp := emu.CPU.State.SP
	fmt.Printf("Stack (SP: 0x%04X):\n", sp)
	for i := 0; i < 16 && sp+uint16(i*2) < 0x2000; i++ {
		addr := sp + uint16(i*2)
		low := emu.Bus.Read8(0, addr)
		high := emu.Bus.Read8(0, addr+1)
		value := uint16(low) | (uint16(high) << 8)
		fmt.Printf("  [0x%04X]: 0x%04X\n", addr, value)
	}
}

func printOAM(emu *emulator.Emulator) {
	fmt.Println("OAM (Object Attribute Memory):")
	for i := 0; i < 8; i++ {
		offset := i * 6
		xLo := emu.PPU.OAM[offset]
		xHi := emu.PPU.OAM[offset+1]
		y := emu.PPU.OAM[offset+2]
		tile := emu.PPU.OAM[offset+3]
		attr := emu.PPU.OAM[offset+4]
		ctrl := emu.PPU.OAM[offset+5]
		
		x := uint16(xLo) | (uint16(xHi&0x01) << 8)
		if (xHi & 0x80) != 0 {
			x |= 0xFE00 // Sign extend
		}
		
		enabled := (ctrl & 0x01) != 0
		fmt.Printf("  Sprite %d: X=%d Y=%d Tile=0x%02X Attr=0x%02X Ctrl=0x%02X (enabled=%v)\n",
			i, int16(x), y, tile, attr, ctrl, enabled)
	}
}

func printPPU(emu *emulator.Emulator) {
	fmt.Printf("PPU State:\n")
	fmt.Printf("  Scanline: %d\n", emu.PPU.GetScanline())
	fmt.Printf("  Dot: %d\n", emu.PPU.GetDot())
	fmt.Printf("  VBlank: %v\n", emu.PPU.VBlankFlag)
	fmt.Printf("  Frame Counter: %d\n", emu.PPU.FrameCounter)
	fmt.Printf("  OAM Addr: %d\n", emu.PPU.OAMAddr)
	fmt.Printf("  OAM Byte Index: %d\n", emu.PPU.OAMByteIndex)
}

func printWatches(dbg *debug.Debugger, emu *emulator.Emulator) {
	watches := dbg.GetWatches()
	if len(watches) == 0 {
		fmt.Println("No watch expressions set")
		return
	}

	fmt.Println("Watch expressions:")
	for i, watch := range watches {
		// Simple evaluation (in a full implementation, this would parse and evaluate)
		fmt.Printf("  [%d] %s = (not yet evaluated)\n", i, watch.Expression)
	}
}

func printVariables(dbg *debug.Debugger) {
	vars := dbg.GetAllVariables()
	if len(vars) == 0 {
		fmt.Println("No variables tracked")
		return
	}

	fmt.Println("Tracked variables:")
	for name, info := range vars {
		fmt.Printf("  %s: %v (type: %s, location: %s)\n", name, info.Value, info.Type, info.Location)
	}
}

func printCallStack(dbg *debug.Debugger) {
	stack := dbg.GetCallStack()
	if len(stack) == 0 {
		fmt.Println("Call stack is empty")
		return
	}

	fmt.Println("Call stack:")
	for i := len(stack) - 1; i >= 0; i-- {
		frame := stack[i]
		fmt.Printf("  #%d: %s at %02X:%04X\n", len(stack)-i-1, frame.FunctionName, frame.Bank, frame.Offset)
	}
}

func printStatus(emu *emulator.Emulator) {
	fmt.Printf("Emulator Status:\n")
	fmt.Printf("  Running: %v\n", emu.Running)
	fmt.Printf("  Paused: %v\n", emu.Paused)
	fmt.Printf("  FPS: %.2f\n", emu.FPS)
	fmt.Printf("  Frame: %d\n", emu.FrameCount)
	fmt.Printf("  CPU Cycles/Frame: %d\n", emu.CPUCyclesPerFrame)
	printRegisters(emu)
}
