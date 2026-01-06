# Nitro-Core-DX - A Love Letter to 1990's Core Gaming

**A Fantasy Console Emulator Combining SNES Graphics with Genesis Power**

---

## The Vision: A Love Letter to 1990's Gaming

Nitro-Core-DX is more than just an emulator—it's a **passion project**, a **love letter to the golden age of 16-bit gaming**. This project represents the "what if" scenario: what if we could combine the best features of the Super Nintendo Entertainment System (SNES) and the Sega Genesis (Mega Drive) into one ultimate fantasy console?

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

**The Result:**
A fantasy console that delivers **SNES-quality graphics** with **Genesis-level performance**, enabling smooth 60 FPS gameplay with complex graphics, advanced parallax scrolling, and stunning Matrix Mode effects for 3D landscapes and racing games.

This isn't just about nostalgia—it's about **respect**. Respect for the original hardware, respect for the games that will run on it, and respect for the developers who will use this emulator to create amazing games.

---

## Project Evolution: The Journey to Go

The path to building Nitro-Core-DX has been a journey of discovery, learning, and finding the right tool for the job. Each language choice taught us valuable lessons about what we needed.

### Basic (QB64): The Anachronistic Start

We started with **QB64**—an anachronistic choice for a modern emulator, but one that could run everywhere. The idea was simple: use something that could compile to native code on any platform, with minimal dependencies.

**What we learned:**
- QB64 could indeed run everywhere
- But compilation problems were frequent and difficult to troubleshoot
- Error messages were cryptic and unhelpful
- The development experience was frustrating
- We needed better tooling, better error messages, better debugging

**The realization:** We needed a language with modern tooling and clear error messages.

### Python: Developer Experience First

We switched to **Python** for the developer experience. Python offered:
- Clear, readable error messages
- Easy debugging with excellent tooling
- Cross-platform compatibility
- Collaborative development made simple
- A rich ecosystem of libraries

**What we learned:**
- Python's developer experience was exactly what we needed
- Development was fast and enjoyable
- But performance became the bottleneck
- We couldn't reach 60 FPS consistently
- CPU emulation was too slow
- PPU rendering couldn't keep up
- The emulator struggled with complex graphics

**The realization:** We needed Python's developer experience, but with C-like performance.

### Cython: The Speed Compromise

We attempted to speed up Python with **Cython**—compiling critical paths to C for near-native speed. The idea was to keep Python's flexibility while getting the performance we needed.

**What we learned:**
- Cython could indeed speed up critical paths
- But complexity grew exponentially
- Python's limitations were still present
- The hybrid approach was difficult to maintain
- We needed something that was fast from the ground up

**The realization:** We needed a language that was fast by design, not fast through compilation tricks.

### Go: The Right Tool for the Job

**Go** is where this project belongs. Go combines Python's developer experience with C-like performance:

- **Performance**: Near-native speed, perfect for 60 FPS emulation
- **Developer Experience**: Clean syntax, excellent tooling, great standard library
- **Concurrency**: Built-in goroutines for audio/rendering threads
- **Cross-Platform**: Single binary, runs everywhere
- **Memory Safety**: Garbage collected but efficient
- **Flexibility**: Easy to extend and maintain

**Why Go?**
- Go gives us the speed we need (that Python couldn't provide)
- Go gives us the flexibility and maintainability that C/C++ lacks
- Go's tooling is excellent (`go fmt`, `go test`, `go build`)
- Go's error handling is explicit and clear
- Go's standard library is comprehensive
- Go's concurrency model is perfect for emulator architecture

**This is where the project belongs.** Go gives us both speed AND maintainability. It makes development a joy while delivering the performance we need.

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

### Performance: The 60 FPS Promise

**This is critical.** The emulator **MUST** run at a steady 60 FPS. This is non-negotiable.

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

### Development Toolkit: A Complete Debugging Environment

This emulator isn't just for playing games—it's for **developing** games. The development tools are as important as the emulation itself.

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

- **Go 1.21 or later** ([Download Go](https://golang.org/dl/))
- **Git** (for cloning the repository)

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/yourusername/nitro-core-dx.git
   cd nitro-core-dx
   ```

2. **Build the project:**
   ```bash
   go build -o nitro-core-dx ./cmd/emulator
   ```

3. **Run the emulator:**
   ```bash
   ./nitro-core-dx
   ```

### Loading ROMs

1. Click **File → Load ROM** in the menu
2. Select a `.ncx` ROM file
3. The emulator will automatically start execution

### Building from Source

```bash
# Build the emulator
go build -o nitro-core-dx ./cmd/emulator

# Run tests
go test ./...

# Format code
go fmt ./...

# Run linter
golangci-lint run
```

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

## Development

### Project Structure

```
nitro-core-dx/
├── cmd/
│   └── emulator/          # Main emulator application
├── internal/
│   ├── cpu/               # CPU emulation
│   ├── memory/            # Memory system
│   ├── ppu/               # Graphics system
│   ├── apu/               # Audio system
│   ├── input/             # Input system
│   ├── rom/               # ROM loading
│   ├── ui/                # User interface
│   └── debug/             # Debugging tools
├── docs/                  # Documentation
├── test/                  # Test ROMs
├── go.mod                 # Go module definition
├── go.sum                 # Go module checksums
└── README.md             # This file
```

### Contributing

Contributions are welcome! This is a passion project, and we'd love to have you contribute.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

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

## The Passion Project Vision

This isn't just an emulator—it's a **love letter to retro gaming**. Every pixel, every cycle, every sound wave matters. This is about preserving the magic of 16-bit gaming while making it accessible and debuggable for modern developers.

The development tools aren't an afterthought—they're **essential**. They're what will allow developers to create amazing games for this fantasy console. The logging system, the memory viewer, the register displays—these are the tools that will help developers understand, debug, and optimize their games.

The 60 FPS requirement isn't just about performance—it's about **respect**. Respect for the original hardware, respect for the games that will run on it, and respect for the developers who will use this emulator.

The video scaling isn't just about convenience—it's about **accessibility**. Making sure that even on a 4K monitor, the games look great and are playable.

The future CRT shader support isn't just a feature—it's about **nostalgia**. Bringing back that warm, fuzzy feeling of playing on a CRT TV.

**This is a passion project. Take pride in every line of code. Make it beautiful. Make it fast. Make it accurate. Make it the best fantasy console emulator it can possibly be.**

---

**Let's build something amazing in Go.** 🎮✨



