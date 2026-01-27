# Nitro-Core-DX

**A Fantasy Console Emulator Combining SNES Graphics with Genesis Power**

A custom 16-bit fantasy console emulator inspired by classic 8/16-bit consoles, designed to combine the best features of the SNES and Sega Genesis into a single, powerful platform.

> **⚠️ ARCHITECTURE IN DESIGN PHASE**: This system is currently in active development. The architecture is not yet finalized, and changes may break compatibility with existing ROMs. If you're using the current iteration, be aware that future changes may require ROM updates. See [System Manual](SYSTEM_MANUAL.md) for current implementation status and development plans.

---

## Project Status

### ✅ Currently Implemented

- **Core Emulation**: CPU, Memory, PPU, APU, Input systems
- **Synchronization**: One-shot completion status, frame counter, VBlank flag
- **Basic Rendering**: Sprite system, background layers, tile rendering
- **Audio System**: 4-channel audio synthesis with duration control
- **ROM Loading**: Complete ROM header parsing and execution
- **Debugging Tools**: Register viewer, memory viewer, cycle-by-cycle logger
- **Sprite Animation**: Working sprite movement with VBlank synchronization

### 🚧 In Progress

- **PPU Rendering**: Full tilemap implementation, Matrix Mode transformation
- **Development Tools**: Tile viewer panel, advanced debugging features

### ❌ Planned

- **UI Framework**: SDL2-based interface with dockable panels
- **Advanced Features**: HDMA, large world support, vertical sprites
- **Development Tools**: CPU debugger, PPU viewer, APU analyzer

For detailed status, see the [System Manual](SYSTEM_MANUAL.md).

---

## Documentation

The project documentation is organized into four main documents:

- **[README.md](README.md)**: Project overview, quick start, build instructions, and contributing guide
- **[SYSTEM_MANUAL.md](SYSTEM_MANUAL.md)**: Complete system architecture, FPGA compatibility, testing framework, and development tools
- **[NITRO_CORE_DX_PROGRAMMING_MANUAL.md](NITRO_CORE_DX_PROGRAMMING_MANUAL.md)**: Complete programming guide for ROM developers
- **[CHANGELOG.md](CHANGELOG.md)**: Version history and change log
- **[END_OF_DAY_PROCEDURE.md](END_OF_DAY_PROCEDURE.md)**: End-of-day cleanup and documentation procedure

---

## Project Vision & Approach

Nitro-Core-DX is a passion project that represents a "what if" scenario: what if we could combine the best features of the Super Nintendo Entertainment System (SNES) and the Sega Genesis (Mega Drive) into one unified fantasy console?

This project is about doing things right from the ground up. While the emulator is still in early development, the focus is on building a solid foundation: accurate emulation, proper architecture, comprehensive testing, and thorough documentation. Every component is being implemented with attention to detail, from cycle-accurate CPU emulation to hardware-accurate synchronization signals.

**From SNES:**
- Advanced graphics capabilities (4 background layers, windowing, per-scanline scroll)
- Mode 7-style perspective and rotation effects (Matrix Mode)
- Rich 15-bit RGB555 color palette (32,768 colors)
- Sophisticated PPU with sprite priorities and blending modes
- Banked memory architecture for flexible addressing

**From Genesis:**
- Raw processing power (10-12 MHz CPU vs SNES's 2.68 MHz)
- Fast DMA and high sprite throughput
- Arcade-friendly performance characteristics

**The Goal:**
A fantasy console that delivers SNES-quality graphics with Genesis-level performance, enabling smooth 60 FPS gameplay with complex graphics, advanced parallax scrolling, and Matrix Mode effects for 3D landscapes and racing games.

**Current Focus:**
The project is currently in active development, with core systems implemented and working toward full functionality. The approach emphasizes correctness over speed, proper error handling, comprehensive testing, and maintaining clean, maintainable code. This is a long-term project where doing it right matters more than doing it fast.

---

## Technology Stack

The project is built in Go, chosen after evaluating multiple languages for the right balance of performance and developer experience.

### Why Go?

Go provides the optimal balance of performance and developer experience:

- **Performance**: Near-native speed, perfect for 60 FPS emulation
- **Developer Experience**: Clean syntax, excellent tooling, great standard library
- **Concurrency**: Built-in goroutines for audio/rendering threads
- **Cross-Platform**: Single binary, runs everywhere
- **Memory Safety**: Garbage collected but efficient
- **Maintainability**: Easy to extend and maintain with clear error handling

---

## Features

### Core Emulation

- **100% Cycle-Accurate CPU Emulation**
  - Custom 16-bit CPU with banked 24-bit addressing
  - 8 general-purpose registers (R0-R7)
  - Complete instruction set (arithmetic, logical, branching, jumps)
  - Precise cycle counting for accurate timing

- **Pixel-Perfect PPU Rendering**
  - 4 independent background layers (BG0-BG3)
  - 128 sprites with priorities and blending modes
  - Matrix Mode (Mode 7-style effects) with large world support
  - Windowing system (2 windows, OR/AND/XOR/XNOR logic)
  - HDMA (per-scanline scroll) for parallax effects
  - Vertical sprites for pseudo-3D worlds

- **Sample-Accurate APU**
  - 4 audio channels (sine, square, saw, noise waveforms)
  - 44,100 Hz sample rate (CD quality)
  - Master volume control
  - Low-latency audio output

- **Precise Memory Mapping**
  - Banked memory architecture (256 banks × 64KB = 16MB)
  - WRAM (32KB), Extended WRAM (128KB), ROM (up to 7.8MB)
  - I/O register routing (PPU, APU, Input)

- **ROM Loading and Execution**
  - Proper header parsing (32-byte header)
  - Entry point handling
  - LoROM-style memory mapping

### Performance

The emulator targets a steady 60 FPS for accurate emulation and smooth gameplay.

- **Frame Limiting**: Automatic 60 FPS frame limiting
  - High-resolution timers (nanosecond precision)
  - Smooth frame pacing (exactly 16.666... milliseconds per frame)
  - No stuttering, no frame drops, no frame skips

- **Unlimited Mode**: Optional "Run at Full Speed" mode
  - Remove all frame limiting for testing and speedruns
  - Toggle in settings menu

- **Performance Optimization**:
  - Optimized hot paths (CPU instruction execution, PPU rendering)
  - Efficient algorithms for tile rendering, sprite sorting, Matrix Mode
  - SIMD/vectorization for Matrix Mode and large world rendering
  - Zero-cost logging when disabled

### Development Toolkit

The emulator includes a comprehensive debugging environment designed for game development and ROM creation.

#### Logging System

- **Component Logging**:
  - CPU Logger: Every instruction execution, register changes, flag updates
  - Memory Logger: All memory reads/writes, bank switches, I/O access
  - PPU Logger: VRAM/CGRAM/OAM writes, layer rendering, Matrix Mode calculations
  - APU Logger: Channel updates, waveform generation, frequency changes
  - Input Logger: Controller state changes, button presses

- **Log Features**:
  - Filterable by component, address range, instruction type
  - Searchable with full-text search
  - Exportable to file (text, CSV, JSON)
  - Cycle-accurate or frame-accurate timestamps
  - Color-coded by component
  - Scrollable with auto-scroll toggle
  - Log levels: None, Errors, Info, Debug, Trace

#### CPU Debugging Tools

- **Register Viewer**: Real-time display of all registers (R0-R7, PC, SP, PBR, DBR, Flags)
- **Instruction Tracer**: Current instruction, disassembly, address, cycles
- **Breakpoints**: Address breakpoints, instruction type breakpoints, conditional breakpoints
- **Watchpoints**: Monitor memory addresses and register values
- **Call Stack**: Function call stack with return addresses

#### PPU Debugging Tools

- **Layer Viewer**: Toggle individual background layers, show scroll positions
- **Sprite Viewer**: List all 128 sprites with attributes, highlight active sprites
- **Tile Viewer**: Browse VRAM tiles visually, export tiles as images
- **Palette Viewer**: Visual palette editor (CGRAM), edit colors in real-time
- **Tilemap Viewer**: Visual tilemap editor, navigate through multiple tilemaps
- **Matrix Mode Visualizer**: Visualize transformation matrix effects

#### APU Debugging Tools

- **Channel Viewer**: Show all 4 channels with current state
- **Waveform Display**: Real-time waveform visualization (oscilloscope-style)

#### Memory Tools

- **Memory Viewer (Hex Editor)**: View and edit all memory regions
  - WRAM, Extended WRAM, VRAM, CGRAM, OAM, ROM
  - Search functionality (find bytes, patterns, strings)
  - Bookmark frequently accessed addresses
  - Memory dump/export capabilities
  - Real-time memory monitoring

- **Live Memory Map**: Visual representation of entire memory space
  - Color-coded regions
  - Memory usage heat map
  - Click to jump to memory viewer

- **Memory Dump**: Export memory regions to file (Binary, Hex, C array, JSON)

### User Interface: Professional and Intuitive

- **Main Window**:
  - Emulator screen (320×200, scaled)
  - Menu bar (File, Emulation, View, Debug, Settings, Help)
  - Toolbar with quick actions (Play, Pause, Reset, Step)
  - Status bar (FPS counter, cycle count, frame time)

- **Dockable Panels**:
  - CPU Registers panel
  - Memory Viewer panel
  - Logs panel
  - Debugger panel
  - All panels dockable, resizable, and hideable

- **Settings Menu**:
  - Emulation Settings: Frame limit (60 FPS / Unlimited), audio settings, input settings
  - Display Settings: Video scaling (1×-6×), fullscreen mode, VSync toggle
  - Debug Settings: Log levels per component, performance profiling

### Video Scaling: Multiple Resolution Support

The console has a native resolution of 320×200 pixels. To make it usable on modern displays:

- **Scaling Options**: 1×, 2×, 3×, 4×, 5×, 6× (native resolution multipliers)
- **Scaling Quality**: High-quality scaling algorithms (bilinear, bicubic, or nearest-neighbor for pixel-perfect)
- **Aspect Ratio**: Maintain aspect ratio
- **Integer Scaling**: Option for pixel-perfect scaling (no blur)

**Future: CRT Shaders**
- Plan for CRT-style shader support (scanlines, phosphor glow, curvature, chromatic aberration)
- Shader selection in settings menu (when implemented)

---

## Quick Start

### Prerequisites

- **Go 1.18 or later** ([Download Go](https://golang.org/dl/))
- **SDL2 Development Libraries** (for UI)
  - **Ubuntu/Debian**: `sudo apt-get install libsdl2-dev`
  - **Fedora/RHEL**: `sudo dnf install SDL2-devel`
  - **macOS**: `brew install sdl2`
  - **Windows**: Download from [SDL2 website](https://www.libsdl.org/download-2.0.php)

**Optional - SDL2_ttf** (for system fonts instead of bitmap fonts):
  - **Ubuntu/Debian**: `sudo apt-get install libsdl2-ttf-dev`
  - **macOS**: `brew install sdl2_ttf`
  - **Windows**: Download from [SDL2_ttf website](https://www.libsdl.org/projects/SDL_ttf/)

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
   
   **Note:** The emulator binary is named `nitro-core-dx` (not `emulator`). This is the FPGA-ready, clock-driven emulator.

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

- **Arrow Keys / WASD**: Move block
- **Z / W**: A button (change block color)
- **X**: B button (change background color)
- **Space**: Pause/Resume
- **Ctrl+R**: Reset emulator
- **Alt+F**: Toggle fullscreen
- **ESC**: Quit

### Building from Source

```bash
# Build the emulator
go build -tags "no_sdl_ttf" -o nitro-core-dx ./cmd/emulator

# Build test ROM generators
go build -o testrom ./cmd/testrom
go build -o demorom ./cmd/demorom
go build -o audiotest ./cmd/audiotest

# Run tests
go test ./...

# Format code
go fmt ./...
```

### Troubleshooting

**SDL2 Not Found:**
1. Install SDL2 development libraries (see Prerequisites)
2. Make sure `pkg-config` can find SDL2: `pkg-config --modversion sdl2`
3. If using a custom SDL2 installation, set `PKG_CONFIG_PATH` environment variable

**Build Errors:**
- Make sure Go is properly installed: `go version`
- Make sure all dependencies are downloaded: `go mod download`
- Clean and rebuild: `go clean -cache && go build ./...`

**Runtime Errors:**
- Check that the ROM file exists and is readable
- Verify the ROM file is a valid Nitro-Core-DX ROM (magic number "RMCF")
- Check console output for specific error messages

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
| **CPU Speed** | 10 MHz (166,667 cycles per frame at 60 FPS) |
| **Memory** | 64KB per bank, 256 banks (16MB total address space) |
| **ROM Size** | Up to 7.8MB (125 banks × 64KB) |
| **Frame Rate** | 60 FPS target |

### Performance Targets

- **60 FPS**: Steady frame rate, no drops
- **Frame Time**: < 16.67ms per frame (including rendering)
- **CPU Usage**: Reasonable CPU usage (not 100% on one core)
- **Memory Usage**: Efficient memory usage

---

## Recent Changes (January 6, 2026)

### Synchronization System

Implemented three complementary synchronization mechanisms:

1. **One-Shot Completion Status (0x9021)**: Audio channel completion detection
2. **Frame Counter (0x803F/0x8040)**: Precise frame-based timing
3. **VBlank Flag (0x803E)**: Hardware-accurate frame synchronization (FPGA-ready)

All three mechanisms work together to provide flexible, hardware-accurate timing. See [System Manual](SYSTEM_MANUAL.md) for details.

### Architecture Improvements

- ✅ Synchronized execution order (APU → CPU → PPU → Audio)
- ✅ One-shot flags prevent race conditions
- ✅ Hardware-accurate signals for FPGA compatibility
- ✅ Clear timing guarantees

---

## Development

### Project Structure

```
nitro-core-dx/
├── cmd/
│   ├── emulator/          # Main emulator application
│   ├── demorom/           # Demo ROM generator
│   ├── audiotest/         # Audio test ROM generator
│   └── testrom/           # Test ROM generator
├── internal/
│   ├── cpu/               # CPU emulation
│   ├── memory/            # Memory system
│   ├── ppu/               # Graphics system
│   ├── apu/               # Audio system
│   ├── input/             # Input system
│   ├── rom/               # ROM loading
│   ├── ui/                # User interface (SDL2)
│   ├── emulator/          # Emulator orchestration
│   └── debug/             # Debugging tools (planned)
├── docs/                  # Documentation (planned)
├── test/                  # Test ROMs
├── go.mod                 # Go module definition
├── go.sum                 # Go module checksums
├── README.md              # This file
├── SYSTEM_MANUAL.md       # Architecture and design documentation
└── NITRO_CORE_DX_PROGRAMMING_MANUAL.md  # Programming guide
```

### Contributing

Contributions are welcome! This project is in active development, and we appreciate any help.

**Getting Started:**
1. Read the [README.md](README.md) for project overview
2. Read the [SYSTEM_MANUAL.md](SYSTEM_MANUAL.md) for architecture details
3. Read the [NITRO_CORE_DX_PROGRAMMING_MANUAL.md](NITRO_CORE_DX_PROGRAMMING_MANUAL.md) for programming guide

**Development Status:**
⚠️ **ARCHITECTURE IN DESIGN PHASE**: This system is currently in active development. The architecture is not yet finalized, and changes may break compatibility with existing ROMs.

**Code Style:**
- Follow Go conventions and best practices
- Use `go fmt` to format code
- Write clear, commented code
- Add tests where appropriate

**Documentation:**
- Update relevant documentation when making changes
- Keep the [SYSTEM_MANUAL.md](SYSTEM_MANUAL.md) up to date with architecture changes
- Update the [NITRO_CORE_DX_PROGRAMMING_MANUAL.md](NITRO_CORE_DX_PROGRAMMING_MANUAL.md) for API changes

**Pull Request Process:**
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Test thoroughly
5. Submit a pull request with a clear description

**Questions?**
Feel free to open an issue for questions or discussions.

### Code Quality

- **Clean Code**: Readable, well-commented, well-structured Go code
- **Go Best Practices**: Follow Go idioms and conventions
  - Use `gofmt` for formatting
  - Follow Go naming conventions
  - Use interfaces where appropriate
  - Keep functions small and focused
  - Use channels and goroutines for concurrency
- **Error Handling**: Proper Go error handling (`error` return values)
- **Logging**: Comprehensive logging (but zero-cost when disabled)
- **Documentation**: Code comments explaining complex logic
- **Testing**: Write tests using Go's `testing` package

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- **SNES**: For inspiring the graphics capabilities and Matrix Mode
- **Sega Genesis**: For inspiring the raw processing power
- **The Retro Gaming Community**: For keeping the spirit of 16-bit gaming alive

---

## Technical Highlights

- **Cycle-accurate CPU emulation** with custom 16-bit instruction set
- **Hardware-accurate synchronization** with VBlank, frame counter, and completion status signals
- **Complete save/load state** functionality for debugging and testing
- **Comprehensive test suite** with automated verification
- **FPGA-ready architecture** designed for potential hardware implementation
- **Professional debugging tools** including logging, memory viewer, and register displays

Built with Go for performance and maintainability, featuring clean architecture, comprehensive error handling, and extensive documentation.



