# Nitro-Core-DX

**The "Dream Console" - Combining the Best of SNES and Genesis**

Nitro-Core-DX is a fantasy console emulator that takes the best features from the Super Nintendo Entertainment System (SNES) and Sega Genesis (Mega Drive) to create the ultimate 16-bit console experience. It's a "what if" scenario: what if you could combine SNES's advanced graphics capabilities with Genesis's raw processing power and FM synthesis?

A fantasy console emulator written in Python, featuring a custom 16-bit CPU, banked memory system, advanced tile-based graphics, hybrid audio system, and SNES-like input handling.

## The Vision: A "Dream Console"

Nitro-Core-DX is designed as a **hybrid console that never existed** - combining the best features from both the SNES and Genesis:

### From SNES (Super Nintendo):
- ✅ **Advanced Graphics**: 4 background layers with windowing and per-scanline scroll (HDMA)
- ✅ **Mode 7-style Effects**: Matrix Mode for perspective and rotation (affine transformations)
- ✅ **15-bit RGB555 Color**: Rich color palette (32,768 colors)
- ✅ **Sophisticated PPU**: Windowing system, sprite priorities, blending modes
- ✅ **Banked Memory Architecture**: Flexible 24-bit addressing

### From Genesis (Mega Drive):
- ✅ **Raw Processing Power**: 10-12 MHz CPU (vs SNES's 2.68 MHz)
- ✅ **FM Synthesis**: YM2612-style FM audio (planned - Phase 3)
- ✅ **Fast DMA**: High-speed memory transfers (planned - Phase 1)
- ✅ **Arcade-Friendly**: High sprite throughput, fast rendering

### Unique Features:
- ✅ **Built-in 3D Assist**: SuperFX-style co-processor for 3D graphics (planned - Phase 4)
- ✅ **Hybrid Audio**: FM synthesis + PCM sample playback
- ✅ **Modern Development**: Python-based for easy debugging and collaboration

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

**Why Python for Performance-Critical Code?**

While Python isn't traditionally used for emulators due to performance concerns, we're committed to making it work through:
- **Strategic Optimization**: Using NumPy for array operations, PyPy for JIT compilation (optional)
- **Cython Integration**: Critical paths can be compiled to C for near-native speed
- **Smart Architecture**: Minimizing Python overhead in hot paths, using efficient data structures
- **Profile-Driven Development**: Identifying bottlenecks and optimizing systematically

See [Performance Optimization](#performance-optimization) below for our strategy.

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

## Design Philosophy

Nitro-Core-DX is designed as a **"what if"** console that combines:

- **SNES's Graphics Prowess**: Advanced PPU with multiple layers, windowing, HDMA, and Mode 7-style effects
- **Genesis's Power**: Faster CPU, FM synthesis, arcade-friendly performance
- **Modern Features**: Built-in 3D assist (SuperFX-style), hybrid audio, and developer-friendly tools

This creates a console that could have existed in the early 90s but with capabilities beyond either system alone.

See **[DESIGN_ANALYSIS.md](DESIGN_ANALYSIS.md)** for a detailed breakdown of current implementation vs. target design.

## Documentation

- **[PROGRAMMING_MANUAL.md](PROGRAMMING_MANUAL.md)**: Complete programming reference for the system
  - CPU architecture and instruction set
  - Memory map and I/O registers
  - PPU (graphics) system with Matrix Mode, windowing, HDMA
  - APU (audio) system
  - Input system
  - ROM format
  - Programming examples

- **[DESIGN_ANALYSIS.md](DESIGN_ANALYSIS.md)**: Detailed analysis of current vs. target architecture
  - Gap analysis for each subsystem
  - Implementation roadmap
  - Feature comparison (SNES vs. Genesis vs. Nitro-Core-DX)

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

## Performance Optimization

**Current Status**: The emulator runs at 5-30 FPS depending on features enabled. Our goal is to reach 60 FPS consistently while keeping Python as the base language.

### Why Python?

We chose Python for its excellent developer experience, but we're committed to making it performant through strategic optimization:

- ✅ **Easy Debugging**: Clear error messages, built-in debugger, VS Code integration
- ✅ **Rapid Development**: Fast iteration, easy to test and modify
- ✅ **Cross-Platform**: Runs everywhere without recompilation
- ✅ **Maintainable**: Readable code, easy to collaborate

### Current Performance Breakdown

**Frame Rate**: 5-30 FPS (target: 60 FPS)

The emulator's performance varies based on:
- **Features Enabled**: Windowing, HDMA, sprite blending add overhead
- **Logging Level**: Detailed logging significantly impacts performance
- **ROM Complexity**: More active layers and sprites = lower FPS

**Component Performance**:
- CPU Execution: ~5-10 FPS (166,667 cycles/frame at 10 MHz = biggest bottleneck)
- PPU Rendering: ~10-30 FPS (4 layers + windowing + sprites)
- Audio Generation: Real-time (not a bottleneck)

### Optimization Strategy

#### 1. **NumPy for Array Operations** ✅ Partially Implemented
- VRAM, CGRAM, and framebuffer can use NumPy arrays
- Vectorized operations for palette lookups and pixel manipulation
- **Status**: NumPy imported but not fully utilized yet - **optimization opportunity**

#### 2. **Profile-Driven Optimization** 🔄 In Progress
- Using Python's `cProfile` to identify bottlenecks
- Focusing optimization efforts on hot paths:
  - CPU execution loop (166,667 cycles/frame = biggest bottleneck)
  - PPU rendering (4 layers + windowing + sprites)
  - Memory access patterns

#### 3. **Cython for Critical Paths** 📋 Planned
- Compile CPU execution loop to C for near-native speed
- Keep Python interface for debugging and flexibility
- **Expected Speedup**: 10-50x for CPU execution
- **Implementation**: Create `.pyx` files for hot paths, compile to `.so`/`.pyd`

#### 4. **PyPy JIT Compilation** 📋 Optional
- PyPy can provide 2-10x speedup for pure Python code
- Requires ensuring compatibility (no C extensions that don't work with PyPy)
- Good for development and testing
- **Trade-off**: Some libraries may not work with PyPy

#### 5. **Architectural Optimizations** ✅ Implemented
- **Palette Caching**: Pre-compute RGB555 → display color conversions
- **Reduced Logging Overhead**: Conditional logging, string formatting only when enabled
- **Efficient Data Structures**: Using lists and arrays optimized for access patterns
- **Minimize Function Calls**: Inline hot paths, reduce indirection

#### 6. **Future Optimizations** 📋 Planned
- **Multithreading**: Separate threads for audio, rendering, and CPU (with proper synchronization)
- **SIMD Operations**: Using NumPy's SIMD capabilities for pixel operations
- **Memory Pooling**: Reuse buffers instead of allocating new ones each frame
- **JIT Compilation**: Consider Numba for numeric-heavy operations
- **Batch Operations**: Process multiple pixels/cycles at once

### Performance Targets

| Component | Current | Target | Strategy |
|-----------|---------|--------|----------|
| CPU Execution | ~5-10 FPS | 60 FPS | Cython compilation, reduce overhead |
| PPU Rendering | ~10-30 FPS | 60 FPS | NumPy optimization, reduce windowing checks |
| Audio Generation | Real-time | Real-time | Already optimized |
| Overall | 5-30 FPS | 60 FPS | Combined optimizations |

### Measuring Performance

```bash
# Profile the emulator
python3 -m cProfile -o profile.stats src_python/main.py roms/graphics.rom

# Analyze profile (interactive)
python3 -m pstats profile.stats
# Then type: sort cumulative, stats 20

# Or use snakeviz for visual profiling
pip install snakeviz
snakeviz profile.stats
```

### Known Bottlenecks

1. **CPU Execution Loop** (Highest Priority)
   - 166,667 cycles per frame = massive Python overhead
   - Each cycle: instruction fetch, decode, execute = ~3-5 Python function calls
   - **Solution**: Cython compilation of `cpu_execute_instruction()`

2. **PPU Rendering** (Second Priority)
   - 4 background layers × 320×200 pixels = 256,000 pixel operations
   - Windowing checks per pixel
   - Sprite sorting and blending
   - **Solution**: NumPy vectorization, reduce redundant checks

3. **Memory Access** (Third Priority)
   - Bounds checking on every VRAM/CGRAM access
   - Bank switching overhead
   - **Solution**: Optimize hot paths, cache frequently accessed data

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

This project is licensed under the **Nitro-Core-DX License** - a custom license that:

- ✅ **Allows commercial game development** - You can sell games made with this emulator
- ❌ **Prohibits commercial hardware/console manufacturing** - Cannot be used to build/sell hardware
- ✅ **Allows free non-commercial use** - Personal projects, learning, education
- ✅ **Allows modification and tinkering** - Fork, modify, experiment freely
- 🔒 **Protects project identity** - Cannot release as your own project

See [LICENSE](LICENSE) file for full details.

## Contributing

Contributions welcome! Please:
1. Follow the existing code style
2. Add tests for new features
3. Update documentation
4. Test on multiple platforms if possible

---

**Note**: This is a work in progress. Some features may be incomplete or subject to change.
