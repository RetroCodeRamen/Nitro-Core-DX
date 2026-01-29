# Debugging Quick Start

## Interactive Debugger

The easiest way to debug your ROMs is using the interactive debugger:

```bash
./debugger test/roms/sprite_eater_game.rom
```

### Quick Commands

- `break 1:0x8000` - Set breakpoint at entry point
- `continue` - Run until breakpoint
- `step 5` - Step 5 instructions
- `registers` - Show CPU state
- `oam` - Show sprite data
- `memory 0:0x1FFF 32` - Show stack
- `help` - Full command list

### Example: Debugging Sprite Issues

```bash
# Start debugger
./debugger test/roms/sprite_eater_game.rom

# Set breakpoint after sprite initialization
(debugger) break 1:0x8100

# Run until breakpoint
(debugger) continue

# Check OAM (sprite data)
(debugger) oam

# Check registers
(debugger) registers

# Step through a few instructions
(debugger) step 10

# Check memory (stack area where sprites are stored)
(debugger) memory 0:0x1FF0 32

# Continue execution
(debugger) continue
```

## Component Logging

Enable logging for specific components:

```bash
# In your test program
logger := debug.NewLogger(10000)
logger.SetComponentEnabled(debug.ComponentCPU, true)
logger.SetComponentEnabled(debug.ComponentPPU, true)
logger.SetMinLevel(debug.LogLevelDebug)
```

## Cycle Logger

For detailed timing analysis:

```bash
./nitro-core-dx -cyclelog=debug.log -maxcycles=1000 test/roms/sprite_eater_game.rom
```

## UI Debug Panels

In the emulator UI:
- View → Registers
- View → Memory Viewer
- View → Tile Viewer
- Debug → Toggle Cycle Logging

## Common Debugging Scenarios

### Sprite Not Appearing

1. Check OAM data: `(debugger) oam`
2. Verify sprite is enabled (Ctrl bit 0 = 1)
3. Check PPU state: `(debugger) ppu`
4. Verify VBlank sync is working

### Variable Has Wrong Value

1. Check stack area: `(debugger) memory 0:0x1FF0 64`
2. Check registers: `(debugger) registers`
3. Step through assignment: `(debugger) step 1`

### Infinite Loop

1. Set breakpoint at loop start
2. Use `step` to see loop condition
3. Check flags: `(debugger) registers` (look at Z flag)

### Memory Corruption

1. Watch memory area: `(debugger) memory 0:0x1000 256`
2. Step through writes
3. Check stack pointer: `(debugger) stack`

For more details, see [DEBUGGING_GUIDE.md](DEBUGGING_GUIDE.md).
