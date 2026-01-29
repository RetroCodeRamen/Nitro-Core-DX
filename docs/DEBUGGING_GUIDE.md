# Nitro Core DX Debugging Guide

This guide covers the debugging tools available for developing and debugging Nitro Core DX ROMs and CoreLX programs.

## Overview

The Nitro Core DX development environment includes several debugging tools:

1. **Interactive Debugger** (`cmd/debugger`) - Command-line debugger with breakpoints, step-through, and inspection
2. **Cycle Logger** - Cycle-by-cycle execution logging
3. **Component Logging** - Filterable logging by component (CPU, PPU, APU, etc.)
4. **Debug Panels** - Visual debugging panels in the emulator UI
5. **Tracing Tools** - Specialized tracing for CPU, OAM, VRAM, etc.

## Interactive Debugger

The interactive debugger (`./debugger <rom.rom>`) provides a command-line interface for debugging ROMs.

### Starting the Debugger

```bash
./debugger test/roms/sprite_eater_game.rom
```

The debugger starts with the emulator in a paused state, ready for commands.

### Basic Commands

#### Execution Control

- `continue` or `c` - Continue execution until next breakpoint
- `step [count]` or `s [count]` - Step N instructions (default: 1)
- `pause` or `p` - Pause execution
- `frame` or `f` - Run one complete frame
- `run` or `r` - Start emulator (runs continuously)

#### Breakpoints

- `break <bank>:<offset>` or `b <bank>:<offset>` - Set breakpoint
  - Example: `break 1:0x8000`
- `breakpoints` or `bp` - List all breakpoints
- `delete <key>` or `d <key>` - Delete breakpoint
- `enable <key>` - Enable breakpoint
- `disable <key>` - Disable breakpoint
- `clear breakpoints` - Clear all breakpoints

#### Inspection

- `registers` or `regs` or `r` - Show CPU registers
- `memory <bank>:<offset> [count]` or `mem <bank>:<offset> [count]` - Show memory
  - Example: `memory 0:0x1000 32`
- `stack` or `st` - Show stack contents
- `oam` - Show OAM (sprite) data
- `ppu` - Show PPU state
- `status` or `st` - Show emulator status

#### Watch Expressions

- `watch <expr>` or `w <expr>` - Add watch expression
  - Example: `watch R0`
- `watches` - Show all watch expressions
- `clear watches` - Clear all watches

#### Variables

- `variables` or `vars` or `v` - Show tracked variables
- Variables are automatically tracked when debugging CoreLX programs

#### Call Stack

- `callstack` or `cs` - Show function call stack

#### Help

- `help` or `h` - Show help message
- `quit` or `q` or `exit` - Exit debugger

### Example Debug Session

```
(debugger) break 1:0x8000
Breakpoint set at 01:8000 (key: 01:8000)

(debugger) continue
Breakpoint hit at 01:8000

(debugger) registers
CPU Registers:
  R0: 0x0000  R1: 0x0000  R2: 0x0000  R3: 0x0000
  R4: 0x0000  R5: 0x0000  R6: 0x0000  R7: 0x0000
  PC: 01:8000  PBR: 01  DBR: 00  SP: 0x1FFF
  Flags: 0x00 (Z:0 N:0 C:0 V:0 I:0 D:0)
  Cycles: 0

(debugger) step 5
[Stepped 5 instructions]

(debugger) memory 0:0x1FFF 16
Memory at 00:1FFF:
  1FFF: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00

(debugger) oam
OAM (Object Attribute Memory):
  Sprite 0: X=0 Y=0 Tile=0x00 Attr=0x00 Ctrl=0x00 (enabled=false)
  Sprite 1: X=0 Y=0 Tile=0x00 Attr=0x00 Ctrl=0x00 (enabled=false)
  ...

(debugger) quit
Exiting debugger...
```

## Cycle Logger

The cycle logger provides cycle-by-cycle execution traces for detailed timing analysis.

### Using Cycle Logger

Enable cycle logging with the `-cyclelog` flag:

```bash
./nitro-core-dx -cyclelog=debug.log -maxcycles=1000 -startcycle=0 test/roms/sprite_eater_game.rom
```

Options:
- `-cyclelog=<file>` - Log file path
- `-maxcycles=<n>` - Maximum cycles to log (0 = unlimited)
- `-startcycle=<n>` - Start logging after N cycles

### Cycle Log Format

Each line contains:
- Cycle number
- PC (Program Counter)
- All registers (R0-R7)
- Stack pointer, PBR, DBR
- CPU flags
- PPU state (scanline, dot, VBlank, frame counter)
- APU state (channels, volume, etc.)
- Key memory locations (VBlank flag, OAM registers, etc.)

## Component Logging

The emulator supports component-based logging that can be filtered and controlled.

### Enabling Component Logging

In code:
```go
logger := debug.NewLogger(10000)
logger.SetComponentEnabled(debug.ComponentCPU, true)
logger.SetComponentEnabled(debug.ComponentPPU, true)
logger.SetMinLevel(debug.LogLevelDebug)
```

### Available Components

- `ComponentCPU` - CPU execution logs
- `ComponentPPU` - PPU rendering logs
- `ComponentAPU` - APU audio logs
- `ComponentMemory` - Memory access logs
- `ComponentInput` - Input system logs
- `ComponentUI` - UI logs
- `ComponentSystem` - System-level logs

### Log Levels

- `LogLevelDebug` - Detailed debugging information
- `LogLevelInfo` - General information
- `LogLevelWarning` - Warnings
- `LogLevelError` - Errors

## Debug Panels (UI)

The emulator UI includes several debug panels accessible from the View/Debug menu:

- **Registers** - CPU register viewer
- **Memory Viewer** - Memory inspection tool
- **Tile Viewer** - Tile and sprite visualization
- **Log Viewer** - Component log viewer
- **Log Controls** - Log filtering controls

Toggle panels with:
- View menu → [Panel Name]
- Debug menu → [Panel Name]

## Tracing Tools

Specialized tracing tools are available in `cmd/`:

- `trace_cpu_execution` - Trace CPU instruction execution
- `trace_oam_writes` - Trace OAM (sprite) writes
- `trace_vram_loop` - Trace VRAM access patterns

## Debugging CoreLX Programs

When debugging CoreLX programs:

1. **Compile with debug info** (future feature):
   ```bash
   ./corelx -debug sprite_eater_game.corelx sprite_eater_game.rom
   ```

2. **Set breakpoints at key locations**:
   ```bash
   (debugger) break 1:0x8000  # Start function
   (debugger) break 1:0x8100  # Game loop
   ```

3. **Inspect variables**:
   ```bash
   (debugger) variables
   (debugger) memory 0:0x1FF0 32  # Stack area
   ```

4. **Watch OAM writes**:
   ```bash
   (debugger) break 1:0x8200  # After oam.write call
   (debugger) oam
   ```

## Tips and Best Practices

1. **Start with breakpoints** - Set breakpoints at function entry points
2. **Use step-through** - Step through code to understand execution flow
3. **Inspect memory** - Check stack and memory for variable values
4. **Watch OAM/PPU** - Monitor sprite and rendering state
5. **Use cycle logging sparingly** - It generates large files quickly
6. **Filter logs** - Enable only needed component logs
7. **Use watch expressions** - Track specific values over time

## Troubleshooting

### Debugger not breaking at breakpoints
- Check that breakpoint address is correct (use `breakpoints` command)
- Ensure breakpoint is enabled (`enable <key>`)
- Verify ROM is executing at that address

### Variables not showing
- Variables are tracked automatically during execution
- Use `memory` command to inspect stack area directly
- Check that variable addresses are correct

### OAM not updating
- Check PPU state with `ppu` command
- Verify OAM writes with `oam` command
- Ensure VBlank synchronization is working

## Future Enhancements

Planned debugging features:
- Source-level debugging (map ROM addresses to CoreLX source lines)
- Expression evaluation in watch expressions
- Conditional breakpoints
- Memory watchpoints
- Performance profiling
- Visual debugger UI integration
