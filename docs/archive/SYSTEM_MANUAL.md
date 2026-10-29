# Nitro-Core-DX System Manual

**Version 1.0 (Under Revision)**  
**Last Updated: July 22, 2026**

> **⚠️ Under Revision / System Context:** This manual contains useful
> architectural context, but the authoritative current sources are `README.md`,
> `docs/HARDWARE_FEATURES_STATUS.md`,
> `docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md`,
> `docs/specifications/APU_FM_OPM_EXTENSION_SPEC.md`, and
> `docs/planning/NEXT_STEPS_PLAN.md`. CoreLX language/manual details are
> intentionally excluded from this pass and will be handled separately.

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Architecture Analysis](#architecture-analysis)
3. [Synchronization Mechanisms](#synchronization-mechanisms)
4. [FPGA Compatibility](#fpga-compatibility)
5. [Development Status](#development-status)
6. [Design Philosophy](#design-philosophy)
7. [Implementation Details](#implementation-details)
8. [Migration Notes](#migration-notes)

---

## System Overview

### System Specifications

| Feature | Specification |
|---------|--------------|
| **Display Resolution** | 320×200 pixels (landscape) / 200×320 (portrait) |
| **Color Depth** | 256 colors (8-bit indexed) |
| **Color Palette** | 256-color CGRAM (RGB555 format, 32,768 possible colors) |
| **Tile Size** | 8×8 or 16×16 pixels (configurable per layer) |
| **Max Sprites** | 128 sprites |
| **Sprite Sizes** | 8×8, 16×16, 32×16, 32×32, 64×32, 64×64, 128×64, 128×128 |
| **Background Layers** | 4 independent layers (BG0, BG1, BG2, BG3) |
| **Matrix Mode** | Mode 7-style per-layer transforms; vertical sprites remain future work |
| **Audio** | YM2608/OPNA audio subsystem (FM + SSG + rhythm + ADPCM) — final audio hardware; a legacy 4-channel synth remains as temporary migration scaffolding |
| **Audio Sample Rate** | 44,100 Hz |
| **CPU Speed** | ~7.67 MHz (127,820 cycles per frame at 60 FPS) |
| **Memory** | 64KB per bank, 256 banks (16MB total address space) |
| **ROM Size** | Up to 3.9MB (125 banks × 32KB LoROM windows) |
| **Frame Rate** | 60 FPS target |

### System Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Nitro-Core-DX                        │
├─────────────────────────────────────────────────────────┤
│  CPU (~7.67 MHz)                                        │
│  ├─ 8 General Purpose Registers (R0-R7)                │
│  ├─ 24-bit Banked Addressing                            │
│  └─ Custom Instruction Set                              │
├─────────────────────────────────────────────────────────┤
│  Memory System                                          │
│  ├─ Bank 0: WRAM (32KB) + I/O (32KB)                   │
│  ├─ Banks 1-125: ROM Space (3.9MB)                    │
│  └─ Banks 126-127: Extended WRAM (128KB)               │
├─────────────────────────────────────────────────────────┤
│  PPU (Picture Processing Unit)                         │
│  ├─ 4 Background Layers (BG0-BG3)                     │
│  ├─ 128 Sprites (8×8 through 128×128)                 │
│  ├─ Matrix Mode (Mode 7-style, per-layer)              │
│  ├─ Windowing System                                   │
│  └─ HDMA (per-scanline scroll/transform/control)       │
├─────────────────────────────────────────────────────────┤
│  APU (Audio Processing Unit) — YM2608 / OPNA           │
│  ├─ FM, SSG, Rhythm, ADPCM (YM2608)                    │
│  ├─ YM2608 Host Interface + Timer/IRQ                  │
│  ├─ (legacy 4-ch synth: temporary scaffolding)        │
│  └─ Master Volume Control                               │
├─────────────────────────────────────────────────────────┤
│  Input System                                          │
│  ├─ Controller 1 & 2                                  │
│  └─ 12-button Support                                 │
└─────────────────────────────────────────────────────────┘
```

---

## Architecture Analysis

### Execution Model

#### Clock-Driven Scheduler Architecture

The emulator uses a **master clock scheduler** that coordinates all subsystems (CPU, PPU, APU) on a unified cycle timeline. This ensures cycle-accurate synchronization and FPGA-ready design.

**Execution Modes:**

1. **Debug Mode** (Cycle-by-Cycle):
   - Steps scheduler one cycle at a time
   - Enables cycle-by-cycle logging and debugging
   - Maximum accuracy for debugging and verification

2. **Optimized Mode** (Chunk-Based):
   - Steps scheduler in chunks of 1000 cycles
   - CPU and PPU advance together on the same timeline
   - APU steps at its sample rate (~174 cycles per sample)
   - Maintains cycle-accurate synchronization while improving performance

**Key Properties:**
- Both modes produce identical results (verified via determinism tests)
- CPU always executes before PPU within each chunk, ensuring writes are visible
- All components advance on the same master clock cycle timeline
- Synchronization is maintained at chunk boundaries

#### Frame Execution Order (Synchronized)

```
Frame Start (127,820 cycles per frame):
  Master Clock coordinates all components:
  
  1. CPU steps for N cycles (via scheduler)
     - Executes ROM instructions
     - Can write to PPU/APU registers (immediate, synchronous)
  
  2. PPU steps for N cycles (via scheduler, same N as CPU)
     - Renders scanlines dot-by-dot
     - Reads PPU registers (sees CPU writes from step 1)
     - Sets VBlank flag at scanline 200
     - Triggers VBlank interrupt
  
  3. APU steps at sample rate (~174 cycles per sample)
     - Generates audio samples
     - Updates channel state
  
  All components advance together on unified cycle timeline
  CPU writes are immediately visible to PPU/APU
```

### Component Synchronization

**Before**: Components operated independently, no synchronization  
**After**: All components synchronized via clear execution order and shared signals

**State Management**:
- ✅ APU completion status synchronized with CPU reads
- ✅ PPU frame counter synchronized with CPU reads
- ✅ VBlank signal provides hardware-accurate frame boundary
- ✅ All signals are one-shot or stable during frame

**Timing Guarantees**:
- ✅ Frame counter increments exactly once per frame
- ✅ VBlank flag set exactly once per frame
- ✅ Completion status set when channels finish, cleared when read
- ✅ No race conditions between CPU and APU/PPU

### Key Architectural Decisions

1. **One-Shot Flags**: All status flags (completion status, VBlank) are cleared immediately after being read, preventing multiple updates per frame
2. **Frame-First Updates**: APU and PPU update their state at the start of the frame, before CPU execution
3. **Hardware-Accurate Signals**: VBlank signal matches real hardware behavior (NES, SNES pattern)
4. **FPGA-Ready Design**: All synchronization signals are simple and easy to implement in FPGA

---

## Synchronization Mechanisms

The emulator provides **three complementary synchronization mechanisms**:

### 1. One-Shot Completion Status (0x9021)

- **Purpose**: Detect when audio channels finish playing
- **Behavior**: Cleared immediately after being read (one-shot)
- **Use Case**: Simple audio timing, prevent multiple updates per frame
- **Status**: ✅ Implemented

**Register**: `CHANNEL_COMPLETION_STATUS` at `0x9021`
- Bits 0-3: Channels 0-3 completion flags (1 = channel finished this frame)
- One-shot: Cleared immediately after read

### 2. Frame Counter (0x803F/0x8040)

- **Purpose**: Precise frame-based timing
- **Behavior**: 16-bit counter increments once per frame
- **Use Case**: Frame-perfect synchronization, measuring elapsed time
- **Status**: ✅ Implemented

**Registers**:
- `FRAME_COUNTER_LOW` at `0x803F`: Frame counter low byte
- `FRAME_COUNTER_HIGH` at `0x8040`: Frame counter high byte

### 3. VBlank Flag (0x803E)

- **Purpose**: Hardware-accurate frame synchronization
- **Behavior**: One-shot flag set at start of frame, cleared when read
- **Use Case**: FPGA compatibility, hardware-accurate synchronization (matches NES/SNES pattern)
- **Status**: ✅ Implemented

**Register**: `VBLANK_FLAG` at `0x803E`
- Bit 0: VBlank active (1 = VBlank period, 0 = not VBlank)
- One-shot: Cleared immediately after read

### Usage Patterns

**Pattern 1: Simple Audio Timing (Completion Status)**
```assembly
main_loop:
    MOV R7, #0x9021        ; Completion status
    MOV R6, [R7]           ; Read (clears flag)
    AND R6, #0x01          ; Check channel 0
    BEQ skip_update        ; Skip if not finished
    ; Update note
skip_update:
    JMP main_loop
```

**Pattern 2: Frame-Perfect Timing (Frame Counter)**
```assembly
main_loop:
    MOV R7, #0x803F        ; Frame counter low
    MOV R3, [R7]           ; Store current frame
    ; ... do work ...
    wait_frame:
        MOV R7, #0x803F
        MOV R6, [R7]
        CMP R6, R3         ; Wait for frame to change
        BEQ wait_frame
    JMP main_loop
```

**Pattern 3: Hardware-Accurate (VBlank)**
```assembly
main_loop:
    wait_vblank:
        MOV R7, #0x803E    ; VBlank flag
        MOV R6, [R7]       ; Read (clears flag)
        AND R6, #0x01
        BEQ wait_vblank    ; Wait for VBlank
    ; Now at start of frame
    ; ... do work ...
    JMP main_loop
```

---

## FPGA Compatibility

### Design Philosophy

The Nitro-Core-DX architecture is designed with **FPGA implementation in mind**. All synchronization signals and timing mechanisms are hardware-accurate and can be directly translated to FPGA logic.

### Hardware-Accurate Signals

#### VBlank Flag (0x803E)
- **Behavior**: One-shot flag set at start of frame, cleared when read
- **FPGA Implementation**: Simple D flip-flop with read-clear logic
- **Hardware Pattern**: Matches NES, SNES, and other retro consoles

**FPGA Logic**:
```verilog
// VBlank flag generation
always @(posedge clk) begin
    if (frame_start) begin
        vblank_flag <= 1'b1;
    end else if (read_vblank) begin
        vblank_flag <= 1'b0;  // Clear when read
    end
end
```

#### Frame Counter (0x803F/0x8040)
- **Behavior**: 16-bit counter increments once per frame
- **FPGA Implementation**: Simple 16-bit counter with frame_start increment

**FPGA Logic**:
```verilog
// Frame counter
reg [15:0] frame_counter;
always @(posedge clk) begin
    if (frame_start) begin
        frame_counter <= frame_counter + 1;
    end
end
```

#### Completion Status (0x9021)
- **Behavior**: One-shot flag, cleared immediately after read
- **FPGA Implementation**: Register with read-clear logic

**FPGA Logic**:
```verilog
// Completion status (one-shot)
reg [3:0] completion_status;
always @(posedge clk) begin
    if (channel_finished) begin
        completion_status[channel] <= 1'b1;
    end else if (read_completion) begin
        completion_status <= 4'b0000;  // Clear when read
    end
end
```

### Recommended FPGA Architecture

#### Clock Domains
- **CPU Clock**: ~7.67 MHz (main system clock in current emulator timing model)
- **Audio Clock**: 44.1 kHz (audio sample generation)
- **Video Clock**: 60 Hz (frame timing, VBlank)

#### Synchronization
- VBlank signal synchronized to video clock
- Frame counter synchronized to video clock
- Completion status synchronized to CPU clock
- Cross-clock domain signals use proper synchronization

### Migration Path

When migrating to FPGA:

1. **Keep Register Layout**: All I/O addresses remain the same
2. **Keep Signal Behavior**: VBlank, frame counter, completion status work identically
3. **Replace Emulation Logic**: Replace Go emulation code with Verilog/VHDL
4. **Add Hardware Peripherals**: Real audio DAC, video DAC, etc.

---

## Development Status

### Current Snapshot (2026-07-22)

The emulator hardware foundation is software-ready. Active work is concentrated
in Dev Kit workflow completion, Sound Studio, CoreLX demo stabilization, YM2608
conformance polish, editor ergonomics, debugger UX, and documentation
alignment.

### ✅ Completed / Operational Components

#### Core Emulation
- ✅ **CPU Core**: Complete instruction set implementation
- ✅ **Memory System**: Complete banked memory architecture
- ✅ **PPU (Graphics)**: background layers, native larger sprites, priority/blending, DMA/HDMA, Matrix Mode, matrix-plane rendering, and scanline effects operational in the emulator
- ✅ **APU (Audio)**: YM2608/OPNA runtime operational; `.ncdxmusic` stream playback and bus-side YM burst streaming exist; conformance refinement remains in progress
- ✅ **Input System**: Complete input handling with dual controllers
- ✅ **ROM Loader**: Complete ROM loading with header parsing

#### Synchronization
- ✅ **One-Shot Completion Status**: Implemented at 0x9021
- ✅ **Frame Counter**: Implemented at 0x803F/0x8040
- ✅ **VBlank Flag**: Implemented at 0x803E
- ✅ **Execution Order**: Synchronized frame execution

#### Dev Kit Tooling
- ✅ **Build/Build+Run**: integrated compiler/emulator path
- ✅ **Embedded Emulator**: framebuffer presentation and SDL audio queue
- ✅ **Diagnostics/Output/Manifest Panes**: build feedback and artifact visibility
- ✅ **Sprite Lab**: strongest art tool; edit/import/export/manifest flows exist
- 🚧 **Tilemap Lab**: usable but needs production round-trip hardening
- 🚧 **Sound Studio**: placeholder only; runtime support exists, UI workflow missing

### 🚧 Remaining / In Progress

#### Product / Tooling
- 🚧 **NitroPackInDemo CoreLX rebuild**: active acceptance target; currently
  exposing a large-program codegen/banking stress case
- 🚧 **Dev Kit generated-code alignment**: templates and tool snippets must
  compile against current language rules
- 🚧 **Sound Studio MVP**: VGM/VGZ import, `.ncdxmusic` inspection/export,
  emulator-backed preview, and source/manifest insertion
- 🚧 **Editor Essentials**: find/replace, go-to-line, symbol navigation, updated
  namespace/builtin highlighting, diagnostics squiggle polish
- 🚧 **Debugger UX**: pause/resume, frame step, CPU step, register/PC panels,
  memory watch workflow

#### Hardware / Rendering Enhancements
- 🚧 **YM2608 Conformance**: runtime is operational; reference-quality
  timbre/pitch/subsystem parity evidence remains
- 🚧 **Large World Tilemap Workflows**: advanced streaming/stitching support
- 🚧 **Vertical Sprites for Matrix Mode**: advanced pseudo-3D sprite scaling and
  depth handling

---

## Design Philosophy

### Vision: A Love Letter to 1990's Gaming

Nitro-Core-DX is more than just an emulator—it's a **passion project**, a **love letter to the golden age of 16-bit gaming**. This project represents the "what if" scenario: what if we could combine the best features of the Super Nintendo Entertainment System (SNES) and the Sega Genesis (Mega Drive) into one ultimate fantasy console?

**From SNES:**
- Advanced graphics capabilities (4 background layers, windowing, per-scanline scroll)
- Mode 7-style perspective and rotation effects (Matrix Mode)
- Rich 15-bit RGB555 color palette (32,768 colors)
- Sophisticated PPU with large hardware sprites, priorities, and blending modes
- Banked memory architecture for flexible addressing

**From Genesis:**
- Raw processing power (10-12 MHz CPU vs SNES's 2.68 MHz)
- Fast DMA and high sprite throughput
- Arcade-friendly performance characteristics

**The Result:**
A fantasy console that delivers **SNES-quality graphics** with **Genesis-level performance**, enabling smooth 60 FPS gameplay with complex graphics, advanced parallax scrolling, and Matrix Mode effects across multiple layers.

### Development Principles

1. **Hardware Accuracy**: Emulate real hardware behavior, not just functionality
2. **Developer-Friendly**: Make development easy with good tools and documentation
3. **FPGA-Ready**: Design with FPGA implementation in mind
4. **Performance First**: 60 FPS is non-negotiable
5. **Respect for Retro**: Honor the original hardware and games

---

## Implementation Details

### CPU Architecture

- **8 General Purpose Registers**: R0-R7 (16-bit)
- **24-bit Banked Addressing**: 256 banks × 64KB = 16MB address space
- **Instruction Set**: Arithmetic, logical, branching, jumps, stack operations
- **Cycle Counting**: Precise cycle counting for accurate timing
- **Flag Management**: Z, N, C, V, I flags

### Memory System

- **Bank 0**: WRAM (32KB) + I/O Registers (32KB)
- **Banks 1-125**: ROM Space (3.9MB)
- **Banks 126-127**: Extended WRAM (128KB)
- **I/O Routing**: PPU (0x8000-0x8FFF), APU (0x9000-0x9FFF), Input (0xA000-0xAFFF)

### PPU (Graphics System)

- **4 Background Layers**: BG0-BG3 with independent scrolling
- **128 Sprites**: native size codes for 8×8, 16×16, 32×16, 32×32, 64×32, 64×64, 128×64, and 128×128 sprites; priorities and blending modes
- **Matrix Mode**: Mode 7-style effects with per-layer transforms; large-world workflows remain future work
- **Windowing System**: 2 windows with OR/AND/XOR/XNOR logic
- **HDMA**: Per-scanline scroll, transform, rebind, priority, tilemap-base, and source-mode updates

### APU (Audio System) — YM2608 / OPNA

- **YM2608/OPNA**: FM, SSG, rhythm, and ADPCM — the final audio subsystem
- **YM2608 Host Interface + Timer/IRQ**: register path, played through the YMFM-backed runtime
- **44,100 Hz Sample Rate**: CD quality audio
- **Conformance**: operational and under active refinement
- **Legacy 4-channel synth** (sine/square/saw/noise, master volume, duration/completion): retained only as temporary migration scaffolding — not final hardware

### Input System

- **Dual Controllers**: Controller 1 and Controller 2
- **12-Button Support**: UP, DOWN, LEFT, RIGHT, A, B, X, Y, L, R, START, Z
- **Latch Mechanism**: Button state latching for reliable input

---

## Migration Notes

### From Python to Go

This repository has been completely rewritten in Go to replace the previous Python implementation.

**What Changed:**
- **Language**: Python → Go
- **Performance**: 5-30 FPS → 60 FPS target
- **Architecture**: Complete rewrite with proper CPU instruction alignment
- **Tooling**: Go's excellent standard library and tooling

**New Go Version Features:**
- Cycle-accurate CPU emulation
- Pixel-perfect PPU rendering
- Working sprite system
- Demo ROM with controllable box
- Comprehensive documentation
- Proper instruction alignment fixes

---

## Testing Framework

The emulator uses a comprehensive testing approach designed for the clock-driven, FPGA-ready architecture.

### Testing Strategy

#### 1. Unit Tests (`*_test.go`)

Unit tests verify individual components work correctly in isolation:

- **`internal/ppu/ppu_test.go`**: Tests PPU components
  - `TestSpriteRendering`: Verifies sprite rendering logic
  - `TestOAMWrite`: Tests OAM (Object Attribute Memory) write functionality
  - `TestVRAMWrite`: Tests VRAM write functionality
  - `TestCGRAMWrite`: Tests CGRAM (palette) write functionality
  - `TestFrameTiming`: Tests PPU frame timing

- **`internal/cpu/cpu_test.go`**: Tests CPU instruction execution
- **`internal/emulator/emulator_test.go`**: Tests emulator core functionality

#### 2. Integration Tests

Integration tests verify components work together:

- **`internal/emulator/savestate_test.go`**: Tests save/load state functionality
- **`internal/emulator/frame_order_test.go`**: Tests frame execution order

#### 3. ROM-Based Tests

ROM-based tests verify the complete system:

- **`roms/simple_sprite.rom`**: Simple static sprite test
- **`roms/bouncing_ball.rom`**: Complex test with movement, collision, and sound

### Running Tests

Use the Makefile targets documented in `docs/testing/README.md` for the current
baseline. `go test ./...` is not the preferred top-level command because the
repository contains vendored/reference resources and standalone generator
packages with their own build assumptions.

**Run Specific Test Suites:**
```bash
go test ./internal/ppu -v
go test ./internal/cpu -v
go test ./internal/emulator -v
```

**Run Specific Test:**
```bash
go test ./internal/ppu -v -run TestSpriteRendering
```

### Debugging Failed Tests

#### PPU Sprite Rendering Issues

If sprite rendering fails, check:

1. **Palette Index**: Must be in bits [3:0] of attributes byte
   - Palette 0 = `0x00`
   - Palette 1 = `0x01`
   - Palette 2 = `0x02`
   - etc.

2. **CGRAM Setup**: Ensure color is written to correct palette
   - Palette 1, Color 1 = CGRAM address `0x11 * 2 = 0x22`

3. **VRAM Tile Data**: Ensure tile data is initialized
   - 16x16 legacy sprite tile = 128 contiguous bytes
   - Larger sprites use 8x8 tile-grid addressing from the base tile index
   - Color index 1 = `0x11` (two pixels per byte)

4. **OAM Data**: Verify sprite is enabled and positioned correctly
   - Control byte bit 0 = enable
   - X-high byte bits [3:1] hold the sprite size code
   - Control byte bit 1 remains only as the legacy 16x16 fallback

### Test Coverage Goals

- [x] PPU sprite rendering
- [x] PPU OAM/VRAM/CGRAM writes
- [x] PPU background/matrix rendering regression coverage
- [x] PPU scanline timing and frame behavior coverage
- [x] CPU instruction execution coverage
- [x] CPU cycle/counting regression coverage
- [x] APU/YM2608 runtime coverage
- [x] Clock scheduler coordination coverage
- [x] Memory bus routing coverage
- [x] Save/load state coverage
- [ ] Expanded YM2608 reference-audio/timbre conformance coverage
- [ ] Tool-generated asset render/audio acceptance coverage

### Performance Testing

For performance-critical components:

```go
func BenchmarkFeature(b *testing.B) {
    // Set up
    // ...
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Test code
    }
}
```

Run benchmarks:
```bash
go test -bench=. ./internal/ppu
```

### Future Testing Improvements

1. **Visual Regression Tests**: Compare rendered frames
2. **Cycle Accuracy Tests**: Verify exact cycle counts
3. **FPGA Compatibility Tests**: Test against FPGA implementation
4. **ROM Compatibility Tests**: Test against known-good ROMs
5. **Stress Tests**: Long-running tests for memory leaks
6. **Concurrency Tests**: Test thread safety (if applicable)

---

## Development Tools

The emulator includes a comprehensive development toolkit for debugging and ROM creation.

### Current Implementation Status

The active Dev Kit is `cmd/corelx_devkit` with a Fyne frontend over
`internal/devkit/service.go`.

Implemented:

- Menu bar, toolbar, status/build state, layout presets, and view modes
- Core editor surface with syntax highlighting and diagnostic navigation
- Diagnostics, output, manifest, and debugger text panes
- Embedded emulator pane with input capture and SDL audio output
- Sprite Lab with project insertion and manifest application
- Tilemap Lab with import/export and source insertion
- Autosave/recovery and persisted settings

Open:

- Sound Studio tab implementation
- Image/plane import UI over the existing CLI importer
- Find/replace, go-to-line, and symbol navigation
- Full debugger controls and structured state panels
- More compile/render acceptance tests for tool-generated assets

### UI Architecture

**External Rendering (Fyne)**:
- ✅ Menu bar (Fyne native menus)
- ✅ Toolbar buttons (Fyne widgets)
- ✅ Status bar (Fyne label)
- ✅ Debug panels (Fyne containers)
- ✅ Emulator screen (Fyne canvas with SDL2 rendering)

**Internal Rendering (SDL2)**:
- ✅ Emulator output buffer (320x200 pixels)
- ❌ NO UI buttons or menus rendered by SDL2
- ❌ NO UI elements rendered by emulator internals

---

## Reference

For detailed programming information, see the [Programming Manual](PROGRAMMING_MANUAL.md).

For project overview and quick start, see the [README](README.md).

---

**End of System Manual**
