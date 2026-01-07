# Nitro-Core-DX System Manual

**Version 1.0**  
**Last Updated: January 6, 2026**

> **⚠️ ARCHITECTURE IN DESIGN PHASE**: This system is currently in active development. The architecture is not yet finalized, and changes may break compatibility with existing ROMs. See [Development Status](#development-status) for current implementation status.

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
| **Background Layers** | 4 independent layers (BG0, BG1, BG2, BG3) |
| **Matrix Mode** | Mode 7-style effects with large world support, vertical sprites |
| **Audio Channels** | 4 channels (sine, square, saw, noise waveforms) |
| **Audio Sample Rate** | 44,100 Hz |
| **CPU Speed** | 10 MHz (166,667 cycles per frame at 60 FPS) |
| **Memory** | 64KB per bank, 256 banks (16MB total address space) |
| **ROM Size** | Up to 7.8MB (125 banks × 64KB) |
| **Frame Rate** | 60 FPS target |

### System Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Nitro-Core-DX                        │
├─────────────────────────────────────────────────────────┤
│  CPU (10 MHz)                                           │
│  ├─ 8 General Purpose Registers (R0-R7)                │
│  ├─ 24-bit Banked Addressing                            │
│  └─ Custom Instruction Set                              │
├─────────────────────────────────────────────────────────┤
│  Memory System                                          │
│  ├─ Bank 0: WRAM (32KB) + I/O (32KB)                   │
│  ├─ Banks 1-125: ROM Space (7.8MB)                    │
│  └─ Banks 126-127: Extended WRAM (128KB)               │
├─────────────────────────────────────────────────────────┤
│  PPU (Picture Processing Unit)                         │
│  ├─ 4 Background Layers (BG0-BG3)                     │
│  ├─ 128 Sprites                                        │
│  ├─ Matrix Mode (Mode 7-style)                         │
│  ├─ Windowing System                                   │
│  └─ HDMA (per-scanline scroll)                         │
├─────────────────────────────────────────────────────────┤
│  APU (Audio Processing Unit)                           │
│  ├─ 4 Audio Channels                                   │
│  ├─ Waveforms: Sine, Square, Saw, Noise               │
│  └─ Master Volume Control                              │
├─────────────────────────────────────────────────────────┤
│  Input System                                          │
│  ├─ Controller 1 & 2                                  │
│  └─ 12-button Support                                 │
└─────────────────────────────────────────────────────────┘
```

---

## Architecture Analysis

### Execution Model

#### Frame Execution Order (Synchronized)

```
Frame Start:
  1. APU.UpdateFrame()
     - Decrements channel durations
     - Sets completion flags (if channels finished)
  
  2. PPU.RenderFrame()
     - Sets VBlank flag = true (at START of frame, before CPU runs)
     - Increments FrameCounter
     - Renders frame using state from previous frame's CPU execution
  
  3. CPU.ExecuteCycles(166667)
     - ROM can read:
       * VBlank flag (0x803E) - will see 1, then cleared
       * Frame counter (0x803F/0x8040) - current frame number
       * Completion status (0x9021) - will see flags, then cleared
  
  4. APU.GenerateSamples(735)
     - Generate audio for this frame
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
- **CPU Clock**: 10 MHz (main system clock)
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

### ✅ Completed Components

#### Core Emulation
- ✅ **CPU Core**: Complete instruction set implementation
- ✅ **Memory System**: Complete banked memory architecture
- ✅ **PPU (Graphics)**: Basic rendering pipeline, sprite system, background layers
- ✅ **APU (Audio)**: Complete audio synthesis with 4 channels
- ✅ **Input System**: Complete input handling with dual controllers
- ✅ **ROM Loader**: Complete ROM loading with header parsing

#### Synchronization
- ✅ **One-Shot Completion Status**: Implemented at 0x9021
- ✅ **Frame Counter**: Implemented at 0x803F/0x8040
- ✅ **VBlank Flag**: Implemented at 0x803E
- ✅ **Execution Order**: Synchronized frame execution

### 🚧 Partially Implemented

#### PPU Rendering
- 🚧 **Background Layer Rendering**: Basic structure, needs full tilemap implementation
- 🚧 **Sprite Rendering**: Structure in place, needs full implementation
- 🚧 **Matrix Mode**: Structure in place, needs transformation matrix implementation
- 🚧 **Tile Rendering**: Placeholder implementation, needs full 4bpp tile decoding

### ❌ Not Yet Implemented

#### UI Framework
- ❌ Main window with SDL2 or similar
- ❌ Menu bar (File, Emulation, View, Debug, Settings, Help)
- ❌ Toolbar with quick actions
- ❌ Status bar (FPS counter, cycle count, frame time)
- ❌ Dockable panels system

#### Development Tools
- ❌ **Logging System**: Component logging (CPU, Memory, PPU, APU, Input)
- ❌ **CPU Debugger**: Register viewer, instruction tracer, breakpoints, watchpoints
- ❌ **PPU Debugger**: Layer viewer, sprite viewer, tile viewer, palette viewer
- ❌ **Memory Viewer**: Hex editor, memory map, memory dump
- ❌ **APU Debugger**: Channel viewer, waveform display

#### Advanced Features
- ❌ Full tilemap rendering with scrolling
- ❌ Complete sprite rendering with priorities and blending
- ❌ Matrix Mode transformation calculations
- ❌ Large world tilemap support
- ❌ Vertical sprite rendering for Matrix Mode
- ❌ HDMA per-scanline scroll updates

---

## Design Philosophy

### Vision: A Love Letter to 1990's Gaming

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
- **Banks 1-125**: ROM Space (7.8MB)
- **Banks 126-127**: Extended WRAM (128KB)
- **I/O Routing**: PPU (0x8000-0x8FFF), APU (0x9000-0x9FFF), Input (0xA000-0xAFFF)

### PPU (Graphics System)

- **4 Background Layers**: BG0-BG3 with independent scrolling
- **128 Sprites**: 8×8 or 16×16 pixels, priorities, blending modes
- **Matrix Mode**: Mode 7-style effects with large world support
- **Windowing System**: 2 windows with OR/AND/XOR/XNOR logic
- **HDMA**: Per-scanline scroll updates for parallax effects

### APU (Audio System)

- **4 Audio Channels**: Sine, square, saw, noise waveforms
- **44,100 Hz Sample Rate**: CD quality audio
- **Master Volume Control**: Global volume adjustment
- **Duration Control**: Automatic note duration with completion status

### Input System

- **Dual Controllers**: Controller 1 and Controller 2
- **12-Button Support**: UP, DOWN, LEFT, RIGHT, A, B, X, Y, L, R, START, SELECT
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

## Reference

For detailed programming information, see the [Programming Manual](NITRO_CORE_DX_PROGRAMMING_MANUAL.md).

For project overview and quick start, see the [README](README.md).

---

**End of System Manual**

