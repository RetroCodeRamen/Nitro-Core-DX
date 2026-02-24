# Nitro-Core-DX

**A Fantasy Console Emulator Combining SNES Graphics with Genesis Power**

A custom 16-bit fantasy console emulator inspired by classic 8/16-bit consoles, designed to combine the best features of the SNES and Sega Genesis into a single, powerful platform.

> **✅ Architecture Stable**: The core hardware architecture is complete and stable. The emulator and tooling are actively maintained, and tests/documentation continue to be refined as clock-driven behavior and debug tooling evolve.

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

A fantasy console that gives you SNES-quality visuals running at Genesis-level performance. Target is smooth 60 FPS (now achieving steady 60 FPS on the current desktop emulator build) with complex graphics, advanced parallax scrolling, and Matrix Mode effects that can handle 3D landscapes and racing games.

**My Philosophy:**
I'm not in a rush. This is a long-term project where doing it right matters more than doing it fast. Every component gets the attention it deserves—from cycle-accurate CPU emulation to hardware-accurate synchronization signals. I'm building something that'll last.

---

## Why Go?

I didn't just pick Go because it's trendy. I evaluated multiple languages and Go won because it hits the sweet spot between "fast enough" and "actually maintainable."

Here's why Go works so well for Nitro-Core-DX:

- **Performance**: Target is 60 FPS (now achieving steady 60 FPS on the current desktop emulator build, with optimization ongoing for more headroom)
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

Nitro-Core-DX is a complete fantasy console system built from scratch, consisting of five major pieces that work together:

1. **Hardware Architecture** - Custom 16-bit CPU, memory map, PPU (graphics), APU (audio), and I/O systems
2. **Emulator** - Hardware-model-focused CPU/PPU/APU emulation with deterministic frame stepping and debug tooling
3. **CoreLX Compiler** - Custom compiled language with Lua-like syntax for hardware-first programming
4. **Nitro-Core-DX App (Dev Kit)** - Integrated editor/build/run environment with embedded emulator pane
5. **Assembler v1** - Text assembly (`.asm`) -> ROM pipeline for lower-level workflows

For detailed information about the development process and challenges, see [Development Notes](docs/DEVELOPMENT_NOTES.md).

---

## Project Status

**Validation Snapshot (2026-02-24):**
- `go build -tags no_sdl_ttf -o nitro-core-dx ./cmd/emulator` passes locally.
- `go test -tags no_sdl_ttf ./... -timeout 180s` passes in a local environment without SDL2_ttf development libraries.
- Some ROM generator helper programs are intentionally gated behind the `testrom_tools` build tag to avoid multiple-`main` conflicts during normal test runs.

### ✅ Currently Implemented

- **Core Emulation**: CPU, Memory, PPU, APU, Input systems implemented and under active validation
- **Synchronization**: One-shot completion status, frame counter, VBlank flag
- **Graphics System**: Complete PPU with all features
  - Sprite system with priority, blending, and alpha transparency
  - 4 background layers with per-layer Matrix Mode transformations
  - Matrix Mode with outside-screen handling and direct color mode
  - Mosaic effect, DMA transfers, sprite-to-background priority
- **Audio System**: 4-channel legacy audio synthesis with PCM playback support
  - Waveform generation (sine, square, saw, noise)
  - PCM sample playback with loop and one-shot modes
  - Volume control and duration management
  - FM/OPM extension host interface + audible OPM-lite synthesis path (in progress)
- **Interrupt System**: Complete IRQ/NMI handling with vector table
- **ROM Loading**: Complete ROM header parsing and execution
- **Developer Tooling Baseline**: Nitro-Core-DX integrated app (editor + embedded emulator), optional standalone emulator UI, structured compiler diagnostics, manifest output, register/memory viewers, cycle logger
- **Test Suite**: Broad regression coverage across CPU/PPU/APU/emulator paths (includes some long-running timing tests)
- **Assembly Toolchain v1**: text assembler (`.asm` -> `.rom`) for advanced low-level workflows
- **Nitro-Core-DX App MVP**: integrated CoreLX editor + Build/Build+Run + diagnostics + embedded emulator

### 🚧 In Progress

- **Nitro-Core-DX App Expansion**: Sprite Lab, Tilemap Editor, Sound Studio, richer editor UX
- **CoreLX Toolchain**: unified asset model, packaging flow, structured assets, banked linker integration
- **FM Audio**: moving from OPM-lite subset toward fuller YM2151-style behavior

### ❌ Optional Enhancements (Not Required)

- **Vertical Sprites**: 3D sprite scaling for Matrix Mode (can be added later)
- **FM Synthesis**: Experimental FM/OPM extension is in progress (not final YM2151-accurate yet)

For detailed status and documentation navigation, see [docs/README.md](docs/README.md) and [docs/HARDWARE_FEATURES_STATUS.md](docs/HARDWARE_FEATURES_STATUS.md).

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
| **Matrix Mode** | Mode 7-style effects with large world support |
| **Audio Channels** | 4 legacy channels + experimental FM extension (OPM-lite/YM2151-style MMIO) |
| **Audio Sample Rate** | 44,100 Hz |
| **CPU Speed** | ~7.67 MHz (127,820 cycles per frame at 60 FPS, Genesis-like) |
| **Memory** | 64KB per bank, 256 banks (16MB total address space) |
| **ROM Size** | Up to 7.8MB (125 banks × 64KB) |
| **Frame Rate** | Target: 60 FPS (Currently: steady 60 FPS on current desktop build) |

### Performance Targets

- **Target: 60 FPS** - Goal is steady frame rate with no drops
- **Current: Steady 60 FPS** - Currently holding 60 FPS on the current desktop emulator build; optimization work continues for headroom and consistency across heavier scenes/platforms
- **Frame Time Target**: < 16.67ms per frame (including rendering)
- **CPU Usage**: Reasonable CPU usage (not 100% on one core)
- **Memory Usage**: Efficient memory usage

---

## Quick Start

### Prerequisites

- **Go 1.22 or later** ([Download Go](https://golang.org/dl/))
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

### Option A: Download a Release (Recommended)

Download the latest prebuilt package for your platform:

- Releases: https://github.com/RetroCodeRamen/Nitro-Core-DX/releases
- Latest release: https://github.com/RetroCodeRamen/Nitro-Core-DX/releases/latest

Package names:
- **Linux**: `nitrocoredx-<version>-linux-amd64.tar.gz`
- **Windows**: `nitrocoredx-<version>-windows-amd64.zip`

After extracting:
- **Linux**: run `./nitrocoredx`
- **Windows**: run `nitrocoredx.exe`

Use **Emulator Only** view inside the app if you just want to load and play/test ROMs.

### Option B: Build from Source (Developer Workflow)

1. **Clone the repository:**
   ```bash
   git clone https://github.com/RetroCodeRamen/Nitro-Core-DX.git
   cd Nitro-Core-DX
   ```

2. **Run Nitro-Core-DX (recommended integrated app):**

   ```bash
   go run ./cmd/corelx_devkit
   ```

3. **Optional: build the standalone emulator UI:**
   
   **Without SDL2_ttf (recommended if SDL2_ttf is not installed):**
   ```bash
   go build -tags "no_sdl_ttf" -o nitro-core-dx ./cmd/emulator
   ```
   
   **With SDL2_ttf (if you have SDL2_ttf installed):**
   ```bash
   go build -o nitro-core-dx ./cmd/emulator
   ```

4. **Build a test ROM (optional):**
   ```bash
   go build -o testrom ./cmd/testrom
   ./testrom test.rom
   ```

5. **Optional: run the standalone emulator UI directly:**
   ```bash
   ./nitro-core-dx -rom test.rom
   ```

### Nitro-Core-DX App Quick Start (Recommended)

```bash
go run ./cmd/corelx_devkit
```

- Use `Load ROM` to run a known-good `.rom` directly in the embedded emulator
- Use `Open` + `Build + Run` for CoreLX files (`.corelx`)
- Use `Emulator Only` view when you want a play/test-focused layout in the same app
- Use `Capture Game Input` when you want keyboard input routed to the embedded emulator while editing
- Example CoreLX validation file: `test/roms/devkit_moving_box_test.corelx`

Known-good ROMs for embedded emulator testing (after generating them locally):
- `test/roms/input_visual_diagnostic.rom`
- `test/roms/fm_opmlite_showcase.rom`
- `test/roms/apu_fm_showcase.rom`

See `test/roms/README_TEST_ROMS.md` for generator commands (`testrom_tools` build tag).

### Release Downloads (GitHub Releases)

Users can download the prebuilt archive for their platform from the **Releases** page:

- Releases: https://github.com/RetroCodeRamen/Nitro-Core-DX/releases
- Latest release: https://github.com/RetroCodeRamen/Nitro-Core-DX/releases/latest

- Linux: `nitrocoredx-<version>-linux-amd64.tar.gz`
- Windows: `nitrocoredx-<version>-windows-amd64.zip`

See `docs/guides/RELEASE_BINARIES.md` for the release build workflow and packaging details.

### Standalone Emulator Command Line Options (Optional)

- `-rom <path>`: Path to ROM file (required)
- `-unlimited`: Run at unlimited speed (no frame limit)
- `-scale <1-6>`: Display scale multiplier (default: 3)
- `-log`: Enable logging (disabled by default)
- `-cyclelog <file>`: Enable cycle-by-cycle logging to a file
- `-maxcycles <N>`: Maximum cycles to log (default `100000`, `0` = unlimited)
- `-cyclestart <N>`: Start cycle logging after cycle `N`

### Standalone Emulator Example Usage

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

### Emulator Input Mapping (Integrated App and Standalone Emulator)

- **Arrow Keys / WASD**: D-pad (move/control)
- **Z / X / V / C**: A / B / X / Y buttons
- **Q / E**: L / R buttons
- **Enter**: Start
- **Backspace**: Extra diagnostic/test button (used by some test ROMs)

Note: Test ROMs can map controls differently. Use the ROM-specific docs/comments for expected behavior.

### Troubleshooting

**SDL2 Not Found:**
1. Install SDL2 development libraries (see Prerequisites above)
2. Make sure `pkg-config` can find SDL2: `pkg-config --modversion sdl2`
3. If using a custom SDL2 installation, set `PKG_CONFIG_PATH` environment variable

**Build Errors:**
- Make sure Go is properly installed: `go version` (should show 1.22 or later)
- Make sure all dependencies are downloaded: `go mod download`
- If SDL2_ttf is not installed, use `-tags no_sdl_ttf` for emulator/UI builds and tests
- Clean and rebuild (no SDL2_ttf): `go clean -cache && go build -tags no_sdl_ttf ./...`
- Fast regression suite: `make test-fast` (recommended before longer test runs)

**ROM Generator Utilities (multiple mains):**
- Some helper generators in `cmd/testrom` and `test/roms` are excluded from default builds/tests with the `testrom_tools` build tag
- Run explicitly with `go run -tags testrom_tools <path-to-generator.go>`

**Runtime Errors:**
- Check that the ROM file exists and is readable
- Verify the ROM file is a valid Nitro-Core-DX ROM (magic number "RMCF")
- Check console output for specific error messages

**Nitro-Core-DX App Input Seems Ignored:**
- Click the embedded emulator pane to give it keyboard focus
- Make sure `Capture Game Input` is enabled in the Dev Kit when testing controls

For more detailed troubleshooting, see [docs/issues/](docs/issues/) for known issues and fixes.

---

## Documentation

The project documentation is organized into several main documents:

### Core Documentation (Start Here)
- **[docs/README.md](docs/README.md)**: Documentation map (current vs historical docs)
- **[SYSTEM_MANUAL.md](SYSTEM_MANUAL.md)**: System architecture manual (under revision; verify against current specs)
- **[PROGRAMMING_MANUAL.md](PROGRAMMING_MANUAL.md)**: Programming manual (under revision; pre-alpha APIs may change)
- **[docs/CORELX.md](docs/CORELX.md)**: Current CoreLX language reference (implementation-aware)
- **[docs/CORELX_DATA_MODEL_PLAN.md](docs/CORELX_DATA_MODEL_PLAN.md)**: Current CoreLX / Nitro-Core-DX app Phase 1 plan
- **[docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md](docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md)**: Current evidence-based hardware specification
- **[docs/specifications/APU_FM_OPM_EXTENSION_SPEC.md](docs/specifications/APU_FM_OPM_EXTENSION_SPEC.md)**: FM extension design + implementation status

### Additional Documentation
- **[CHANGELOG.md](CHANGELOG.md)**: Version history and change log
- **[docs/DEVELOPMENT_NOTES.md](docs/DEVELOPMENT_NOTES.md)**: Development process, challenges, and philosophy
- **[docs/DEVKIT_ARCHITECTURE.md](docs/DEVKIT_ARCHITECTURE.md)**: Dev Kit backend/frontend split and invariants
- **[docs/issues/](docs/issues/)**: Known issues and fixes
- **[docs/testing/](docs/testing/)**: Testing guides and results
- **[docs/specifications/](docs/specifications/)**: Hardware specs, pin definitions, FPGA docs (with current-vs-historical notes)
- **[docs/guides/](docs/guides/)**: Setup guides, build instructions, and procedures
- **[docs/planning/](docs/planning/)**: Development plans and roadmaps

---

## Features

### Core Emulation

- **Cycle-Counted CPU Emulation**
  - Custom 16-bit CPU with banked 24-bit addressing
  - 8 general-purpose registers (R0-R7)
  - Complete instruction set with precise cycle counting

- **Feature-Complete PPU Rendering (under continued timing/perf validation)**
  - 4 independent background layers (BG0-BG3)
  - 128 sprites with priorities and blending modes
  - Matrix Mode (Mode 7-style effects on multiple layers)
  - Windowing system with proper logic
  - HDMA for per-scanline effects

- **Legacy APU + FM Extension Path**
  - 4 legacy audio channels with waveforms (sine, square, saw, noise)
  - 44,100 Hz sample rate
  - PCM playback support
  - Master volume control
  - FM extension MMIO + timer/IRQ path with audible OPM-lite subset (in progress)

- **Precise Memory Mapping**
  - Banked memory architecture (256 banks × 64KB = 16MB)
  - WRAM (32KB), Extended WRAM (128KB), ROM (up to 7.8MB)
  - I/O register routing

- **ROM Loading and Execution**
  - Proper header parsing (32-byte header)
  - Entry point handling
  - LoROM-style memory mapping

### Tooling

- **Nitro-Core-DX App (MVP)**: CoreLX editor, Build/Build+Run, structured diagnostics panel, manifest/build output panels, embedded emulator
- **CoreLX Compiler**: structured diagnostics, manifest JSON, compile bundle JSON, asset normalization groundwork
- **Assembler v1 (`cmd/asm`)**: text assembly to ROM for low-level workflows
- **Logging & Debug Support**: component logging, cycle logger, register/memory viewers, debugger CLI components
- **ROM Test Tooling**: Go-based ROM builders and test generators (some gated by `testrom_tools`)

For detailed information about debugging tools, see [SYSTEM_MANUAL.md](SYSTEM_MANUAL.md) and [docs/README.md](docs/README.md) for current doc routing.

---

## Project Structure

```
nitro-core-dx/
├── cmd/
│   ├── emulator/          # Standalone emulator application
│   ├── corelx/            # CoreLX compiler CLI
│   ├── corelx_devkit/     # Integrated Dev Kit (editor + embedded emulator)
│   ├── asm/               # v1 text assembler CLI
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
│   ├── asm/               # Assembler implementation
│   ├── devkit/            # UI-agnostic Dev Kit backend
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
2. Read [docs/README.md](docs/README.md) for the current documentation map
3. Read the [CoreLX Documentation](docs/CORELX.md) for current language behavior
4. Use [PROGRAMMING_MANUAL.md](PROGRAMMING_MANUAL.md) and [SYSTEM_MANUAL.md](SYSTEM_MANUAL.md) as manuals under revision, verifying details against current specs/tests

**Development Status:**
✅ **Architecture Stable**: Core hardware architecture is stable; active work continues on tooling, tests, and documentation alignment.

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
