# Nitro-Core-DX

**A Fantasy Console Emulator Combining SNES Graphics with Genesis Power**

A custom 16-bit fantasy console emulator inspired by classic 8/16-bit consoles, designed to combine the best features of the SNES and Sega Genesis into a single, powerful platform.

> **✅ Architecture Stable**: The core hardware architecture is complete and stable. All hardware features are implemented and tested. The system is ready for game development. Optional enhancements may be added in the future, but they won't break compatibility with existing ROMs.

---

## Meet Nitro-Core-DX

Ever wonder what would happen if you took the SNES's gorgeous graphics and mixed them with the Genesis's raw horsepower? That's exactly what Nitro-Core-DX is all about. It's a fantasy console that doesn't just emulate the classics—it creates something entirely new by combining the best of both worlds.

Think of it as the console that could have existed in an alternate timeline where Nintendo and Sega decided to collaborate instead of compete. I'm building this from the ground up with modern tools, but with the soul of the 16-bit era.

---

## The Vision: Best of Both Worlds

Nitro-Core-DX started with a simple question: *"What if?"* What if you could take the SNES's beautiful graphics and combine them with the Genesis's raw speed? What if you didn't have to choose between Mode 7 effects and smooth 60 FPS gameplay?

This isn't just another emulator—it's a passion project that's building something genuinely new. I'm not trying to recreate history; I'm trying to create the console that *should have* existed. And I'm doing it the right way: cycle-accurate emulation, proper architecture, comprehensive testing, and documentation that actually makes sense.

### What I'm Stealing (Politely) from SNES

The SNES brought some incredible graphics tech, and Nitro-Core-DX brings all of it:

- **4 Background Layers** - Parallax scrolling that'll make your eyes happy
- **Matrix Mode** - Mode 7-style perspective and rotation (but better, because it can do it on multiple layers simultaneously)
- **32,768 Colors** - That gorgeous 15-bit RGB555 palette
- **Sprite Magic** - Priorities, blending modes, alpha transparency—the works
- **Smart Memory** - Banked architecture that gives you flexibility without headaches

### What I'm Borrowing from Genesis

The Genesis was fast, and I like fast:

- **~7.67 MHz CPU** - Nearly 3× faster than the SNES's 2.68 MHz
- **DMA That Actually Works** - Fast memory transfers that don't slow you down
- **Arcade Performance** - The kind of speed that makes racing games and shooters feel *right*

### The Result?

A fantasy console that gives you SNES-quality visuals running at Genesis-level performance. Target is smooth 60 FPS (currently ~30 FPS) with complex graphics, advanced parallax scrolling, and Matrix Mode effects that can handle 3D landscapes and racing games.

**My Philosophy:**
I'm not in a rush. This is a long-term project where doing it right matters more than doing it fast. Every component gets the attention it deserves—from cycle-accurate CPU emulation to hardware-accurate synchronization signals. I'm building something that'll last.

---

## Why Go?

I didn't just pick Go because it's trendy. I evaluated multiple languages and Go won because it hits the sweet spot between "fast enough" and "actually maintainable."

Here's why Go works so well for Nitro-Core-DX:

- **Performance**: Target is 60 FPS (currently achieving ~30 FPS, optimization ongoing)
- **Developer Experience**: Clean syntax that doesn't make you want to throw your keyboard
- **Concurrency**: Built-in goroutines that make audio/rendering threading actually pleasant
- **Cross-Platform**: One binary, runs everywhere (Linux, macOS, Windows—you name it)
- **Memory Safety**: Garbage collected, but not in a "pause the world for 5 seconds" kind of way
- **Maintainability**: Code that you can actually read and understand six months later

The best part? When I eventually port this to FPGA hardware, the architecture I've built in Go will translate cleanly. That's not an accident—it's by design.

---

## Console Design

Here's what the console will look like when I build the first prototype:

<div align="center">

![Console Isometric View](Images/Console%20isometric.jpg)

*Isometric view of the Nitro-Core-DX console*

![Console Top View](Images/Console%20Top%20view.png)

*Top-down view showing the console design*

![Controller](Images/Controller.jpg)

*The Nitro-Core-DX controller design*

</div>

---

## Project Components

Nitro-Core-DX is a complete fantasy console system built from scratch, consisting of three major components:

1. **Hardware Architecture** - Custom 16-bit CPU, memory map, PPU (graphics), APU (audio), and I/O systems
2. **Emulator** - Cycle-accurate CPU emulation, pixel-perfect PPU rendering, sample-accurate audio synthesis
3. **CoreLX Compiler** - Custom compiled language with Lua-like syntax for hardware-first programming

For detailed information about the development process and challenges, see [Development Notes](docs/DEVELOPMENT_NOTES.md).

---

## Project Status

### ✅ Currently Implemented

- **Core Emulation**: CPU, Memory, PPU, APU, Input systems (100% complete)
- **Synchronization**: One-shot completion status, frame counter, VBlank flag
- **Graphics System**: Complete PPU with all features
  - Sprite system with priority, blending, and alpha transparency
  - 4 background layers with per-layer Matrix Mode transformations
  - Matrix Mode with outside-screen handling and direct color mode
  - Mosaic effect, DMA transfers, sprite-to-background priority
- **Audio System**: 4-channel audio synthesis with PCM playback support
  - Waveform generation (sine, square, saw, noise)
  - PCM sample playback with loop and one-shot modes
  - Volume control and duration management
- **Interrupt System**: Complete IRQ/NMI handling with vector table
- **ROM Loading**: Complete ROM header parsing and execution
- **Debugging Tools**: Register viewer, memory viewer, cycle-by-cycle logger, GUI logging controls
- **Test Suite**: Comprehensive tests for all hardware features

### 🚧 In Progress

- **Development Tools**: Advanced debugging features, tile viewer panel
- **Language Design**: NitroLang compiler design (documentation phase)

### ❌ Optional Enhancements (Not Required)

- **Vertical Sprites**: 3D sprite scaling for Matrix Mode (can be added later)
- **FM Synthesis**: Advanced audio synthesis (can be added later)

For detailed status, see the [System Manual](SYSTEM_MANUAL.md).

---

## System Specifications

| Feature | Specification |
|---------|--------------|
| **Display Resolution** | 320×200 pixels (landscape) / 200×320 (portrait) |
| **Color Depth** | 256 colors (8-bit indexed) |
| **Color Palette** | 256-color CGRAM (RGB555 format, 32,768 possible colors) |
| **Tile Size** | 8×8 or 16×16 pixels (configurable per layer) |
| **Max Sprites** | 128 sprites |
| **Background Layers** | 4 independent layers (BG0, BG1, BG2, BG3) |
| **Matrix Mode** | Mode 7-style effects with large world support, vertical sprites |
| **Audio Channels** | 4 channels (sine, square, saw, noise waveforms) |
| **Audio Sample Rate** | 44,100 Hz |
| **CPU Speed** | ~7.67 MHz (127,820 cycles per frame at 60 FPS, Genesis-like) |
| **Memory** | 64KB per bank, 256 banks (16MB total address space) |
| **ROM Size** | Up to 7.8MB (125 banks × 64KB) |
| **Frame Rate** | Target: 60 FPS (Currently: ~30 FPS) |

### Performance Targets

- **Target: 60 FPS** - Goal is steady frame rate with no drops
- **Current: ~30 FPS** - Currently achieving approximately 30 FPS, optimization work in progress
- **Frame Time Target**: < 16.67ms per frame (including rendering)
- **CPU Usage**: Reasonable CPU usage (not 100% on one core)
- **Memory Usage**: Efficient memory usage

---

## Quick Start

### Prerequisites

- **Go 1.18 or later** ([Download Go](https://golang.org/dl/))
- **SDL2 Development Libraries**
  - **Ubuntu/Debian**: `sudo apt-get install libsdl2-dev`
  - **Fedora/RHEL**: `sudo dnf install SDL2-devel`
  - **macOS**: `brew install sdl2`
  - **Windows**: Download from [SDL2 website](https://www.libsdl.org/download-2.0.php)

**Optional - SDL2_ttf** (for system fonts):
  - **Ubuntu/Debian**: `sudo apt-get install libsdl2-ttf-dev`
  - **macOS**: `brew install sdl2_ttf`
  - **Windows**: Download from [SDL2_ttf website](https://www.libsdl.org/projects/SDL_ttf/)
  
  *Note: The emulator works fine without SDL2_ttf—it has a built-in bitmap font.*

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/RetroCodeRamen/Nitro-Core-DX.git
   cd Nitro-Core-DX
   ```

2. **Build the emulator:**
   
   **Without SDL2_ttf (recommended if SDL2_ttf is not installed):**
   ```bash
   go build -tags "no_sdl_ttf" -o nitro-core-dx ./cmd/emulator
   ```
   
   **With SDL2_ttf (if you have SDL2_ttf installed):**
   ```bash
   go build -o nitro-core-dx ./cmd/emulator
   ```

3. **Build a test ROM (optional):**
   ```bash
   go build -o testrom ./cmd/testrom
   ./testrom test.rom
   ```

4. **Run the emulator:**
   ```bash
   ./nitro-core-dx -rom test.rom
   ```

### Command Line Options

- `-rom <path>`: Path to ROM file (required)
- `-unlimited`: Run at unlimited speed (no frame limit)
- `-scale <1-6>`: Display scale multiplier (default: 3)
- `-log`: Enable logging (disabled by default)

### Example Usage

```bash
# Run with default 3x scale
./nitro-core-dx -rom test.rom

# Run at unlimited speed with 4x scale
./nitro-core-dx -rom test.rom -unlimited -scale 4

# Run with 1x scale (native resolution)
./nitro-core-dx -rom test.rom -scale 1

# Run with logging enabled
./nitro-core-dx -rom test.rom -log
```

### Controls

- **Arrow Keys / WASD**: Move/control
- **Z**: A button
- **X**: B button
- **Space**: Pause/Resume
- **Ctrl+R**: Reset emulator
- **Alt+F**: Toggle fullscreen
- **ESC**: Quit

### Troubleshooting

**SDL2 Not Found:**
1. Install SDL2 development libraries (see Prerequisites above)
2. Make sure `pkg-config` can find SDL2: `pkg-config --modversion sdl2`
3. If using a custom SDL2 installation, set `PKG_CONFIG_PATH` environment variable

**Build Errors:**
- Make sure Go is properly installed: `go version` (should show 1.18 or later)
- Make sure all dependencies are downloaded: `go mod download`
- Clean and rebuild: `go clean -cache && go build ./...`

**Runtime Errors:**
- Check that the ROM file exists and is readable
- Verify the ROM file is a valid Nitro-Core-DX ROM (magic number "RMCF")
- Check console output for specific error messages

For more detailed troubleshooting, see [docs/issues/](docs/issues/) for known issues and fixes.

---

## Documentation

The project documentation is organized into several main documents:

### Core Documentation
- **[SYSTEM_MANUAL.md](SYSTEM_MANUAL.md)**: Complete system architecture, FPGA compatibility, testing framework, and development tools
- **[PROGRAMMING_MANUAL.md](PROGRAMMING_MANUAL.md)**: Complete programming guide covering both CoreLX and assembly languages
- **[docs/CORELX.md](docs/CORELX.md)**: Complete CoreLX language documentation
- **[docs/specifications/HARDWARE_SPECIFICATION.md](docs/specifications/HARDWARE_SPECIFICATION.md)**: Complete hardware specification for FPGA implementation

### Additional Documentation
- **[CHANGELOG.md](CHANGELOG.md)**: Version history and change log
- **[docs/DEVELOPMENT_NOTES.md](docs/DEVELOPMENT_NOTES.md)**: Development process, challenges, and philosophy
- **[docs/issues/](docs/issues/)**: Known issues and fixes
- **[docs/testing/](docs/testing/)**: Testing guides and results
- **[docs/specifications/](docs/specifications/)**: Hardware specifications and pin definitions
- **[docs/guides/](docs/guides/)**: Setup guides, build instructions, and procedures
- **[docs/planning/](docs/planning/)**: Development plans and roadmaps

---

## Features

### Core Emulation

- **100% Cycle-Accurate CPU Emulation**
  - Custom 16-bit CPU with banked 24-bit addressing
  - 8 general-purpose registers (R0-R7)
  - Complete instruction set with precise cycle counting

- **Pixel-Perfect PPU Rendering**
  - 4 independent background layers (BG0-BG3)
  - 128 sprites with priorities and blending modes
  - Matrix Mode (Mode 7-style effects on multiple layers)
  - Windowing system with proper logic
  - HDMA for per-scanline effects

- **Sample-Accurate APU**
  - 4 audio channels with waveforms (sine, square, saw, noise)
  - 44,100 Hz sample rate
  - PCM playback support
  - Master volume control

- **Precise Memory Mapping**
  - Banked memory architecture (256 banks × 64KB = 16MB)
  - WRAM (32KB), Extended WRAM (128KB), ROM (up to 7.8MB)
  - I/O register routing

- **ROM Loading and Execution**
  - Proper header parsing (32-byte header)
  - Entry point handling
  - LoROM-style memory mapping

### Development Tools

- **Logging System**: Component-based logging with filtering and export
- **CPU Debugging**: Register viewer, instruction tracer, breakpoints, watchpoints
- **PPU Debugging**: Layer viewer, sprite viewer, tile viewer, palette viewer
- **Memory Tools**: Hex editor, memory map, memory dump
- **GUI Logging Controls**: Enable/disable logging per component from Debug menu

For detailed information about debugging tools, see [SYSTEM_MANUAL.md](SYSTEM_MANUAL.md).

---

## Project Structure

```
nitro-core-dx/
├── cmd/
│   ├── emulator/          # Main emulator application
│   ├── corelx/            # CoreLX compiler
│   ├── testrom/           # Test ROM generator
│   ├── testrom_input/     # Input test ROM generator
│   └── ...
├── internal/
│   ├── cpu/               # CPU emulation
│   ├── memory/            # Memory system
│   ├── ppu/               # Graphics system
│   ├── apu/               # Audio system
│   ├── input/             # Input system
│   ├── ui/                # User interface
│   ├── emulator/          # Emulator orchestration
│   ├── corelx/            # CoreLX compiler
│   └── debug/             # Debugging tools
├── test/
│   └── roms/              # Test ROMs
├── docs/
│   ├── issues/            # Known issues and fixes
│   ├── testing/           # Testing guides
│   ├── specifications/    # Hardware specifications
│   └── ...
├── README.md              # This file
├── SYSTEM_MANUAL.md       # System architecture
├── PROGRAMMING_MANUAL.md  # Programming guide
└── CHANGELOG.md           # Version history
```

---

## Contributing

Contributions are welcome! This project is in active development.

**Getting Started:**
1. Read the [README.md](README.md) for project overview
2. Read the [SYSTEM_MANUAL.md](SYSTEM_MANUAL.md) for architecture details
3. Read the [CoreLX Documentation](docs/CORELX.md) for CoreLX language guide
4. Read the [PROGRAMMING_MANUAL.md](PROGRAMMING_MANUAL.md) for complete programming guide

**Development Status:**
✅ **Architecture Stable**: Core hardware is 100% complete. The system is ready for game development.

**Code Style:**
- Follow Go conventions and best practices
- Use `go fmt` to format code
- Write clear, commented code
- Add tests where appropriate

**Pull Request Process:**
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request with a clear description

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- **SNES**: For showing what beautiful 16-bit graphics could look like
- **Sega Genesis**: For proving that speed matters just as much as looks
- **The Retro Gaming Community**: For keeping the spirit of 16-bit gaming alive
