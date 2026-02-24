# Nitro-Core-DX System Manual

**Version 1.0 (Under Revision)**  
**Last Updated: January 6, 2026**

> **⚠️ Under Revision / Historical Snapshot:** This manual contains useful architectural context but includes stale values/status claims (for example CPU timing and audio/FM status). Verify against `README.md`, `docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md`, and `docs/specifications/APU_FM_OPM_EXTENSION_SPEC.md` before using as current source-of-truth.

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

- **`test/roms/simple_sprite.rom`**: Simple static sprite test
- **`test/roms/bouncing_ball.rom`**: Complex test with movement, collision, and sound

### Running Tests

**Run All Tests:**
```bash
go test ./...
```

**Run Specific Test Suite:**
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
   - 16x16 tile = 128 bytes
   - Color index 1 = `0x11` (two pixels per byte)

4. **OAM Data**: Verify sprite is enabled and positioned correctly
   - Control byte bit 0 = enable
   - Control byte bit 1 = 16x16 size

### Test Coverage Goals

- [x] PPU sprite rendering
- [x] PPU OAM/VRAM/CGRAM writes
- [ ] PPU background rendering
- [ ] PPU scanline timing
- [ ] CPU instruction execution
- [ ] CPU cycle accuracy
- [ ] APU sound generation
- [ ] Clock scheduler coordination
- [ ] Memory bus routing
- [ ] Save/load state

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

#### Phase 1: UI Consolidation ✅ IN PROGRESS

**Goal**: All UI rendered externally using Fyne, nothing rendered by emulator internals

**Status**:
- ✅ Fyne toolbar buttons functional with state updates
- ⏳ SDL2-based UI rendering removal (keep only for emulator screen)
- ⏳ All panels are Fyne widgets
- ⏳ Menu items toggle panels

#### Phase 2: Debug Panels

**2.1 Register Viewer ✅ CREATED**

- Real-time CPU register display (R0-R7)
- Program Counter (PC) display (bank:offset)
- Stack Pointer (SP) display
- Bank registers (PBR, DBR)
- Flags register (Z, N, C, V, I, D)
- Updates at 60 FPS
- **Status**: Panel created and integrated into FyneUI

**2.2 Memory Viewer ⏳ PLANNED**

- Hex dump view of memory
- Bank selector (0-255)
- Offset selector (0x0000-0xFFFF)
- Real-time updates
- Search functionality
- Bookmark addresses

**2.3 Tile Viewer ⏳ PLANNED**

- Visual grid of tiles from VRAM
- Palette selector
- Tile size selector (8x8 or 16x16)
- Click to select tile
- Export tile as image
- Real-time updates as VRAM changes

#### Phase 3: Sprite Editor Tool

**3.1 Basic Sprite Editor ✅ CREATED**

- Pixel-level editing (8x8 or 16x16 tiles)
- Palette selector (16 colors)
- Clear/Export/Import buttons
- Grid display
- **Status**: Basic structure created, needs pixel editing and export functionality

**3.2 Enhanced Sprite Editor ⏳ PLANNED**

- Multi-tile sprite editing
- Animation support
- Sprite sheet management
- Export to ROM format
- Preview with different palettes

#### Phase 4: Better Test ROMs

**4.1 Animated Sprite ROM ⏳ PLANNED**

- Multiple animation frames
- Sprite movement
- Collision detection
- Sound effects

**4.2 Character Sprite ROM ⏳ PLANNED**

- Character sprite (not just a box)
- Walking animation
- Multiple directions
- Background scrolling

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
