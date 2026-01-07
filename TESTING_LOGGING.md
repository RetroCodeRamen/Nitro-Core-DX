# Testing the Logging System

## Quick Start

1. **Build the emulator:**
   ```bash
   go build -o nitro-core-dx ./cmd/emulator
   ```

2. **Run with a ROM:**
   ```bash
   ./nitro-core-dx -rom test/roms/demo.rom -scale 3
   ```

## Testing the Log Viewer

1. **Open the log viewer:**
   - Press `Ctrl+L` to toggle the log viewer panel
   - The panel appears on the right side of the window
   - You should see log entries scrolling in real-time

2. **What to look for:**
   - Log entries from different components (CPU, PPU, APU, etc.)
   - Timestamps for each entry
   - Color-coded log levels (Error=Red, Warning=Yellow, Info=Blue, Debug=Green, Trace=Gray)

## Testing Component Toggles

1. **Open log controls:**
   - Press `Ctrl+K` to toggle the log controls panel
   - The panel appears on the left side of the window

2. **Toggle components:**
   - Click the colored checkboxes next to each component name
   - Green = logging enabled, Red = logging disabled
   - Try disabling CPU logging and watch the log viewer - CPU entries should stop appearing
   - Re-enable it and CPU logs should resume

3. **Test each component:**
   - Disable APU logging - you should see fewer audio-related logs
   - Disable PPU logging - you should see fewer graphics-related logs
   - Toggle them back on to verify they resume

## Testing CPU Log Levels

1. **Change CPU log level with keyboard:**
   - `Ctrl+1` = None (no CPU logging)
   - `Ctrl+2` = Errors only
   - `Ctrl+3` = Branches and jumps only
   - `Ctrl+4` = Memory access + branches
   - `Ctrl+5` = Register changes + branches
   - `Ctrl+6` = All instructions (default)
   - `Ctrl+7` = Full trace (every cycle)

2. **What to expect:**
   - At level 1 (None): No CPU logs appear
   - At level 2 (Errors): Only CPU errors (if any)
   - At level 3 (Branches): Only branch/jump instructions
   - At level 4 (Memory): Memory reads/writes + branches
   - At level 5 (Registers): Register changes + branches
   - At level 6 (Instructions): All CPU instructions (most verbose)
   - At level 7 (Trace): Full trace with all details

3. **Test with the demo ROM:**
   - Start with level 6 (Instructions) - you should see many CPU logs
   - Switch to level 3 (Branches) - you should see far fewer logs, only branches
   - Switch to level 1 (None) - CPU logs should disappear completely

## Testing with Different ROMs

1. **Demo ROM (graphics + audio):**
   ```bash
   ./nitro-core-dx -rom test/roms/demo.rom -scale 3
   ```
   - Should generate logs from CPU, PPU, and APU
   - Good for testing all components

2. **Audio Test ROM (audio only):**
   ```bash
   ./nitro-core-dx -rom test/roms/audiotest.rom -scale 3
   ```
   - Should generate heavy APU logging
   - Good for testing APU component toggle

## Expected Behavior

### Log Viewer Panel
- Shows log entries in chronological order
- Auto-scrolls to newest entries
- Color-coded by log level
- Shows component name, timestamp, and message

### Log Controls Panel
- Component checkboxes toggle logging on/off immediately
- CPU log level changes take effect immediately
- Visual feedback (green/red checkboxes)
- CPU level indicator shows current level

### Performance
- Logging should not significantly impact emulator performance
- With all components enabled at highest level, you may see many logs per frame
- Disabling components reduces log volume and improves performance

## Troubleshooting

1. **No logs appearing:**
   - Check that components are enabled (green checkboxes)
   - Check CPU log level is not set to "None" (Ctrl+1)
   - Verify ROM is running (check emulator screen)

2. **Too many logs:**
   - Disable some components (click checkboxes)
   - Lower CPU log level (Ctrl+1-5)
   - Use level 3 (Branches) for minimal CPU logging

3. **Panel not appearing:**
   - Press `Ctrl+K` to show log controls
   - Press `Ctrl+L` to show log viewer
   - Check that window is large enough to show panels

4. **Clicks not working:**
   - Make sure log controls panel is visible (`Ctrl+K`)
   - Click directly on the colored checkbox areas
   - Try clicking on the CPU level selector area

## Keyboard Shortcuts Summary

- `Ctrl+L` - Toggle log viewer
- `Ctrl+K` - Toggle log controls
- `Ctrl+1` - CPU log level: None
- `Ctrl+2` - CPU log level: Errors
- `Ctrl+3` - CPU log level: Branches
- `Ctrl+4` - CPU log level: Memory
- `Ctrl+5` - CPU log level: Registers
- `Ctrl+6` - CPU log level: Instructions (default)
- `Ctrl+7` - CPU log level: Trace



