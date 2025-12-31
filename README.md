# Nitro-Core-DX

A fantasy console emulator written in Python, featuring a custom 16-bit CPU, banked memory system, tile-based graphics, 4-channel audio, and SNES-like input handling.

A fantasy console emulator written in Python, featuring a custom 16-bit CPU, banked memory system, tile-based graphics, 4-channel audio, and SNES-like input handling.

## Project History

This project originally started in **QB64 (QuickBASIC)** - chosen as an anachronistic choice for a modern emulator project. The goal was to create something that could run on everything, and using BASIC for a modern emulator felt like a fun anachronism.

QB64 was suggested as the best modern BASIC implementation, but we encountered significant issues getting it to work:

- **Compilation problems**: QB64's command-line compiler had issues with multi-file projects
- **Poor error messages**: "Syntax error on line 111" with no context made debugging nearly impossible
- **Difficult troubleshooting**: Limited tooling and unclear error reporting made it hard to fix issues
- **Collaboration barriers**: Hard to work together when errors were cryptic

We converted the entire project to **Python** to solve these issues while maintaining the same simple, straightforward ethos. Python provides:
- ✅ **Clear error messages**: "IndexError: list index out of range at line 45" instead of "Syntax error"
- ✅ **Easy debugging**: Built-in debugger, `print()` statements, VS Code integration
- ✅ **Better tooling**: Auto-complete, type checking, linting, excellent IDE support
- ✅ **Cross-platform**: Runs on everything (Windows, Linux, macOS)
- ✅ **Collaborative development**: Much easier to work together and get help

The emulator logic remains **identical** - we just changed the language for better developer experience and cross-platform compatibility. All the core architecture, algorithms, and design decisions are preserved.

## Project Structure

```
.
├── src_python/          # Core emulator code
│   ├── main.py         # Main entry point
│   ├── cpu.py          # CPU implementation
│   ├── memory.py       # Memory management
│   ├── ppu.py          # Graphics (PPU)
│   ├── apu.py          # Audio (APU)
│   ├── input.py        # Input handling
│   ├── rom.py          # ROM loading
│   ├── ui.py           # User interface
│   └── config.py       # Configuration constants
├── tests/              # Test files
│   ├── test_basic.py
│   ├── test_cpu_instructions.py
│   └── ...
├── scripts/            # ROM builders and utilities
│   ├── create_graphics_rom.py
│   ├── create_test_rom.py
│   └── ...
├── roms/               # ROM files
│   ├── graphics.rom
│   └── test.rom
├── logs/               # Log files
│   └── emulator_log_*.txt
├── PROGRAMMING_MANUAL.md  # Complete programming reference
├── ROADMAP.md          # Development roadmap
└── README.md           # This file
```

## Features

- **16-bit CPU**: Custom CPU with 8 registers, 24-bit addressing, and interrupt support (10 MHz)
- **Banked Memory**: 24-bit address space with 64KB banks (256 banks total, 16MB)
- **Graphics (PPU)**: 320x200 display with 4 background layers (BG0-BG3), sprites, windowing, and per-scanline scroll
- **Matrix Mode**: 90's retro-futuristic perspective/rotation effects for 3D-style landscapes
- **Windowing System**: SNES-style windowing with 2 windows, per-layer control, and logic modes (OR/AND/XOR/XNOR)
- **HDMA**: Per-scanline scroll for parallax and perspective effects
- **Sprite System**: 128 sprites with priority levels (0-3) and blending modes (alpha, additive, subtractive)
- **Audio (APU)**: 4-channel synthesizer with sine, square, saw, and noise waveforms
- **Input**: SNES-like 12-button controller support
- **ROM Format**: Custom ROM format with header and mapper support

## Quick Start

### Installation

```bash
# Install dependencies
pip install pygame

# Or use requirements.txt
pip install -r requirements.txt
```

### Running

```bash
# Run without ROM (test mode)
python3 src_python/main.py

# Run with a ROM file
python3 src_python/main.py roms/graphics.rom
```

### Creating ROMs

```bash
# Create a test ROM with animated graphics
python3 scripts/create_graphics_rom.py roms/my_game.rom

# Create a simple test ROM
python3 scripts/create_test_rom.py roms/simple.rom
```

### Running Tests

```bash
# Run all tests
python3 tests/test_basic.py
python3 tests/test_cpu_instructions.py
python3 tests/test_branches.py
```

## Documentation

- **[PROGRAMMING_MANUAL.md](PROGRAMMING_MANUAL.md)**: Complete programming reference for the system
  - CPU architecture and instruction set
  - Memory map and I/O registers
  - PPU (graphics) system with Matrix Mode
  - APU (audio) system
  - Input system
  - ROM format
  - Programming examples

- **[ROADMAP.md](ROADMAP.md)**: Development roadmap and progress tracking

## Controls

- **P**: Pause/Resume emulation
- **D**: Toggle debug mode
- **S**: Toggle step mode
- **ESC**: Exit emulator

## Keyboard Mapping

- **Arrow Keys**: D-Pad
- **Z**: A button
- **X**: B button
- **A**: X button
- **S**: Y button
- **Q**: L button
- **W**: R button
- **Enter**: START
- **Shift**: SELECT

## System Specifications

| Feature | Specification |
|---------|--------------|
| Display Resolution | 320x200 (landscape) / 200x320 (portrait) |
| Color Depth | 256 colors (8-bit indexed) |
| Tile Size | 8x8 or 16x16 pixels |
| Max Sprites | 128 |
| Audio Channels | 4 (sine, square, saw, noise) |
| Audio Sample Rate | 44,100 Hz |
| CPU Speed | 10 MHz (166,667 cycles/frame) |
| Memory | 64KB per bank, 256 banks (16MB total) |
| ROM Size | Up to 7.8MB (125 banks × 64KB) |
| Background Layers | 4 (BG0, BG1, BG2, BG3) |
| Windowing | 2 windows with OR/AND/XOR/XNOR logic |
| HDMA | Per-scanline scroll support |
| Sprite Priority | 4 levels (0-3) with blending modes |

## Development

### Adding New Instructions

1. Add opcode constant to `src_python/config.py`
2. Add decoding logic to `cpu_decode_instruction()` in `src_python/cpu.py`
3. Add execution logic to `cpu_execute_instruction()` in `src_python/cpu.py`
4. Write tests in `tests/`
5. Update `PROGRAMMING_MANUAL.md`

### Debugging

The emulator includes a built-in debugger accessible via the UI:
- Hex memory viewer
- CPU register display
- Step mode execution
- Detailed logging (configurable)

Enable detailed logging in Settings > Logging to see CPU execution traces.

## License

This project is open source. See LICENSE file for details.

## Contributing

Contributions welcome! Please:
1. Follow the existing code style
2. Add tests for new features
3. Update documentation
4. Test on multiple platforms if possible

---

**Note**: This is a work in progress. Some features may be incomplete or subject to change.
