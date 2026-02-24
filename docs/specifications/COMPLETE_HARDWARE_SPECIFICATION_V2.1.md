# Nitro-Core-DX Complete Hardware Specification

**Version 2.1**  
**Last Updated: January 30, 2026**  
**Purpose: FPGA-implementable hardware specification based on emulator source code evidence**

> **📌 Evidence-Based**: This specification is derived directly from the emulator source code. Every behavior, register, and timing specification is backed by code evidence. Unverified or inferred behaviors are explicitly marked.

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Evidence Map](#evidence-map)
3. [CPU Architecture](#cpu-architecture)
4. [Memory System](#memory-system)
5. [PPU (Graphics) Specification](#ppu-graphics-specification)
6. [APU (Audio) Specification](#apu-audio-specification)
7. [Input System Specification](#input-system-specification)
8. [I/O Register Map](#io-register-map)
9. [Timing and Synchronization](#timing-and-synchronization)
10. [ROM Format](#rom-format)
11. [Reset & Power-On State](#reset--power-on-state)
12. [Open Bus & Undefined Behavior](#open-bus--undefined-behavior)
13. [Spec Corrections from v2.0](#spec-corrections-from-v20)
14. [Implementation Confidence Notes](#implementation-confidence-notes)
15. [FPGA Implementation Guidelines](#fpga-implementation-guidelines)

---

## System Overview

### System Specifications

| Feature | Specification | Evidence |
|---------|--------------|----------|
| **Display Resolution** | 320×200 pixels | `internal/ppu/scanline.go:9-10` |
| **Color Depth** | 256 colors (8-bit indexed) | `internal/ppu/ppu.go:1105-1133` (CGRAM lookup) |
| **Color Palette** | 256-color CGRAM (RGB555 format) | `internal/ppu/ppu.go:14, 1105-1133` |
| **Tile Size** | 8×8 or 16×16 pixels (configurable per layer) | `internal/ppu/ppu.go:123, 839-842` |
| **Max Sprites** | 128 sprites | `internal/ppu/ppu.go:17, 328` |
| **Background Layers** | 4 independent layers (BG0, BG1, BG2, BG3) | `internal/ppu/ppu.go:20` |
| **Matrix Mode** | Mode 7-style effects with per-layer support | `internal/ppu/ppu.go:127-135, 647-827` |
| **Audio Channels** | 4 channels (sine, square, saw, noise waveforms) | `internal/apu/apu.go:19, 56-87` |
| **Audio Sample Rate** | 44,100 Hz | `internal/emulator/emulator.go:76, 116` |
| **CPU Speed** | ~7.67 MHz (127,820 cycles per frame at 60 FPS) | `internal/emulator/emulator.go:114, 147` |
| **Memory** | 64KB per bank, 256 banks (16MB total address space) | `internal/memory/bus.go:7, 10` |
| **ROM Size** | Up to 7.8MB (125 banks × 32KB) | `internal/memory/cartridge.go:72` |
| **Frame Rate** | 60 FPS target | `internal/emulator/emulator.go:139` |

### System Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Nitro-Core-DX                        │
├─────────────────────────────────────────────────────────┤
│  CPU (~7.67 MHz)                                        │
│  ├─ 8 General Purpose Registers (R0-R7, 16-bit)        │
│  ├─ 24-bit Banked Addressing                            │
│  ├─ Custom Instruction Set (16-bit instructions)       │
│  └─ Flags: Z, N, C, V, I, D                            │
├─────────────────────────────────────────────────────────┤
│  Memory System                                          │
│  ├─ Bank 0: WRAM (32KB) + I/O (32KB)                   │
│  ├─ Banks 1-125: ROM Space (7.8MB)                     │
│  └─ Banks 126-127: Extended WRAM (128KB)                │
├─────────────────────────────────────────────────────────┤
│  PPU (Picture Processing Unit)                         │
│  ├─ VRAM (64KB)                                         │
│  ├─ CGRAM (512 bytes, 256 colors)                      │
│  ├─ OAM (768 bytes, 128 sprites)                       │
│  ├─ 4 Background Layers (BG0-BG3)                     │
│  ├─ 128 Sprites                                        │
│  ├─ Matrix Mode (Mode 7-style, per-layer)             │
│  ├─ Windowing System                                   │
│  └─ HDMA (per-scanline scroll/matrix)                 │
├─────────────────────────────────────────────────────────┤
│  APU (Audio Processing Unit)                           │
│  ├─ 4 Audio Channels                                   │
│  ├─ Waveforms: Sine, Square, Saw, Noise               │
│  └─ Master Volume Control                              │
├─────────────────────────────────────────────────────────┤
│  Input System                                          │
│  ├─ Controller 1 & 2 (12-button support)               │
│  └─ Latch mechanism                                    │
└─────────────────────────────────────────────────────────┘
```

---

## Evidence Map

### CPU Subsystem

**Files:**
- `internal/cpu/cpu.go` - CPU state, instruction fetch, interrupt handling
- `internal/cpu/instructions.go` - Instruction execution implementations

**Key Structs:**
- `CPUState` - Complete CPU state (registers, PC, flags, cycles)
- `CPU` - CPU instance with memory interface

**Key Functions:**
- `ExecuteInstruction()` - Main instruction execution
- `FetchInstruction()` - Instruction fetch with bank handling
- `handleInterrupt()` - Interrupt handling
- Instruction executors: `executeMOV()`, `executeADD()`, etc.

**Constants:**
- Flags: `FlagZ`, `FlagN`, `FlagC`, `FlagV`, `FlagI`, `FlagD` (`cpu.go:35-41`)
- Interrupt vectors: `VectorIRQ=0xFFE0`, `VectorNMI=0xFFE2` (`cpu.go:53-56`)
- Stack start: `SP=0x1FFF` (`cpu.go:107`)

**Evidence Confidence: High** - Complete implementation with all instructions verified.

### Memory Subsystem

**Files:**
- `internal/memory/bus.go` - Memory bus routing
- `internal/memory/memory.go` - Memory system (legacy, not used in emulator)
- `internal/memory/cartridge.go` - ROM cartridge handling

**Key Structs:**
- `Bus` - Memory bus with WRAM, Extended WRAM, Cartridge, I/O handlers
- `Cartridge` - ROM data and header parsing

**Key Functions:**
- `Read8()`, `Write8()`, `Read16()`, `Write16()` - Memory access
- `readIO8()`, `writeIO8()` - I/O register routing
- `LoadROM()` - ROM loading and header parsing

**Memory Layout (Evidence):**
- Bank 0: WRAM 0x0000-0x7FFF, I/O 0x8000+ (`bus.go:38-46`)
- Banks 1-125: ROM space (`bus.go:49-55`)
- Banks 126-127: Extended WRAM (`bus.go:57-64`)
- LoROM mapping: `romOffset = (bank-1) × 32768 + (offset - 0x8000)` (`cartridge.go:72`)

**Evidence Confidence: High** - Complete memory map implementation.

### PPU Subsystem

**Files:**
- `internal/ppu/ppu.go` - PPU registers, rendering
- `internal/ppu/scanline.go` - Scanline-by-scanline rendering

**Key Structs:**
- `PPU` - Complete PPU state (VRAM, CGRAM, OAM, layers, HDMA)
- `BackgroundLayer` - Per-layer state (scroll, matrix, mosaic)
- `Window` - Windowing configuration

**Key Functions:**
- `Read8()`, `Write8()` - Register access
- `StepPPU()` - Clock-driven rendering
- `renderDot()` - Per-pixel rendering with priority
- `updateHDMA()` - HDMA table processing

**Timing Constants (Evidence):**
- `DotsPerScanline = 581` (`scanline.go:17`)
- `VisibleDots = 320` (`scanline.go:18`)
- `HBlankDots = 261` (`scanline.go:19`)
- `VisibleScanlines = 200` (`scanline.go:24`)
- `VBlankScanlines = 20` (`scanline.go:25`)
- `TotalScanlines = 220` (`scanline.go:26`)

**Evidence Confidence: High** - Complete PPU implementation with clock-driven rendering.

### APU Subsystem

**Files:**
- `internal/apu/apu.go` - APU channels, waveform generation

**Key Structs:**
- `APU` - APU state (channels, master volume, completion status)
- `AudioChannel` - Per-channel state (frequency, volume, waveform, duration)

**Key Functions:**
- `Read8()`, `Write8()` - Register access
- `GenerateSample()` - Audio sample generation
- `UpdateFrame()` - Frame-based duration countdown

**Register Layout (Evidence):**
- Channels: 8 bytes per channel (`apu.go:130-131`)
- Channel 0: 0x9000, Channel 1: 0x9008, Channel 2: 0x9010, Channel 3: 0x9018 (8 bytes per channel)
- Master Volume: 0x9020 (`apu.go:331`)
- Completion Status: 0x9021 (`apu.go:108`)

**Evidence Confidence: High** - Complete APU implementation with all waveforms.

### Input Subsystem

**Files:**
- `internal/input/input.go` - Input system

**Key Structs:**
- `InputSystem` - Controller button states, latch flags

**Key Functions:**
- `Read8()`, `Write8()` - Register access
- `SetButton()`, `SetButton2()` - Button state setting

**Register Layout (Evidence):**
- Controller 1: 0xA000 (low), 0xA001 (high) (`input.go:25-28`)
- Controller 2: 0xA002 (low), 0xA003 (high) (`input.go:29-32`)
- Latch: 0xA001 (Controller 1), 0xA003 (Controller 2) (`input.go:41-52`)

**Evidence Confidence: High** - Simple input system fully implemented.

---

## CPU Architecture

### Register Set

**Evidence:** `internal/cpu/cpu.go:8-32`

- **8 General Purpose Registers**: R0-R7 (all 16-bit)
- **Program Counter (PC)**: 24-bit (8-bit bank + 16-bit offset)
  - `PCBank` (uint8) - Bank number
  - `PCOffset` (uint16) - Offset within bank
- **Bank Registers**:
  - `PBR` (uint8) - Program Bank Register (should match PCBank)
  - `DBR` (uint8) - Data Bank Register
- **Stack Pointer (SP)**: 16-bit (points to WRAM in bank 0)
  - Initial value: `0x1FFF` (top of WRAM) (`cpu.go:107`)
  - Grows downward (`cpu.go:519-523`)
- **Flags Register**: 6 flags (Z, N, C, V, I, D)
  - Bit 0: Z (Zero)
  - Bit 1: N (Negative)
  - Bit 2: C (Carry)
  - Bit 3: V (Overflow)
  - Bit 4: I (Interrupt Enable)
  - Bit 5: D (Division by Zero)

### Instruction Format

**Evidence:** `internal/cpu/cpu.go:313-316`

All instructions are 16-bit words:
```
[15:12] = Opcode family (0x0-0xF)
[11:8]  = Mode/subop
[7:4]   = Register 1 (destination)
[3:0]   = Register 2 (source)
```

### Instruction Set

**Evidence:** `internal/cpu/instructions.go` (all instruction executors)

| Opcode | Instruction | Description | Cycles | Evidence |
|--------|-------------|-------------|--------|----------|
| 0x0000 | NOP | No operation | 1 | `instructions.go:8-11` |
| 0x1000 | MOV | Move/Load/Store (7 modes) | 1-3 | `instructions.go:14-116` |
| 0x2000 | ADD | Add | 2-3 | `instructions.go:118-136` |
| 0x3000 | SUB | Subtract | 2-3 | `instructions.go:138-156` |
| 0x4000 | MUL | Multiply | 3-4 | `instructions.go:158-176` |
| 0x5000 | DIV | Divide | 4-5 | `instructions.go:178-209` |
| 0x6000 | AND | Bitwise AND | 2-3 | `instructions.go:211-229` |
| 0x7000 | OR | Bitwise OR | 2-3 | `instructions.go:231-249` |
| 0x8000 | XOR | Bitwise XOR | 2-3 | `instructions.go:251-269` |
| 0x9000 | NOT | Bitwise NOT | 2 | `instructions.go:271-279` |
| 0xA000 | SHL | Shift Left | 2-3 | `instructions.go:281-306` |
| 0xB000 | SHR | Shift Right | 2-3 | `instructions.go:308-333` |
| 0xC000 | CMP | Compare | 2-3 | `instructions.go:335-360` |
| 0xC100 | BEQ | Branch if Equal | 2-3 | `instructions.go:362-404` |
| 0xC200 | BNE | Branch if Not Equal | 2-3 | `instructions.go:362-404` |
| 0xC300 | BGT | Branch if Greater Than | 2-3 | `instructions.go:362-404` |
| 0xC400 | BLT | Branch if Less Than | 2-3 | `instructions.go:362-404` |
| 0xC500 | BGE | Branch if >= | 2-3 | `instructions.go:362-404` |
| 0xC600 | BLE | Branch if <= | 2-3 | `instructions.go:362-404` |
| 0xD000 | JMP | Jump (relative) | 2-3 | `instructions.go:406-428` |
| 0xE000 | CALL | Call subroutine | 4-5 | `instructions.go:430-456` |
| 0xF000 | RET | Return from subroutine | 3-4 | `instructions.go:458-515` |

**MOV Modes (Evidence: `instructions.go:14-116`):**
- Mode 0: MOV R1, R2 (register to register)
- Mode 1: MOV R1, #imm (immediate to register)
- Mode 2: MOV R1, [R2] (load 16-bit from memory)
- Mode 3: MOV [R1], R2 (store 16-bit to memory)
- Mode 4: PUSH R1
- Mode 5: POP R1
- Mode 6: MOV R1, [R2] (load 8-bit, zero-extended)
- Mode 7: MOV [R1], R2 (store 8-bit)
- Mode 8: Reserved (treated as NOP)

**I/O Access Behavior (Evidence: `instructions.go:34-49, 56-69`):**
- I/O addresses (bank 0, offset 0x8000+) are 8-bit only
- 16-bit reads from I/O: zero-extended 8-bit read
- 16-bit writes to I/O: only low byte written

### Flags

**Evidence:** `internal/cpu/cpu.go:197-235`

- **Z (Zero)**: Set when result is zero (`UpdateFlags()`)
- **N (Negative)**: Set when result bit 15 is set (`UpdateFlags()`)
- **C (Carry)**: Set on unsigned overflow/underflow (`UpdateFlagsWithOverflow()`)
- **V (Overflow)**: Set on signed overflow (`UpdateFlagsWithOverflow()`)
- **I (Interrupt Enable)**: When set, interrupts are disabled (`cpu.go:418`)
- **D (Division by Zero)**: Set when division by zero occurs (`instructions.go:195`)

### Interrupt System

**Evidence:** `internal/cpu/cpu.go:396-449`

- **Interrupt Vector Table**: Bank 0, addresses 0xFFE0-0xFFE3
  - 0xFFE0: IRQ handler bank (1 byte)
  - 0xFFE1: IRQ handler offset high byte (low byte always 0x00)
  - 0xFFE2: NMI handler bank (1 byte)
  - 0xFFE3: NMI handler offset high byte (low byte always 0x00)
- **VBlank Interrupt**: IRQ triggered at start of VBlank (`ppu/scanline.go:99-101`)
- **Interrupt State Saving**: PBR, PC, and flags pushed to stack automatically (`cpu.go:411-415`)
- **Interrupt Overhead**: 7 cycles (`cpu.go:446`)
- **NMI**: Non-maskable (always handled)
- **IRQ**: Maskable (only if I flag is clear)

---

## Memory System

### Banked Memory Architecture

**Evidence:** `internal/memory/bus.go:36-96`

- **Total Address Space**: 16MB (256 banks × 64KB)
- **Bank Size**: 64KB (16-bit addressing)
- **Bank Number**: 8-bit (0-255)
- **Full Address**: 24-bit (bank:offset)

### Memory Layout

**Evidence:** `internal/memory/bus.go:36-96`

| Bank Range | Type | Size | Description | Evidence |
|------------|------|------|-------------|----------|
| 0x00 | WRAM + I/O | 64KB | 32KB WRAM + 32KB I/O registers | `bus.go:38-46` |
| 0x01-0x7D | ROM | 7.8MB | ROM space (LoROM mapping) | `bus.go:49-55` |
| 0x7E-0x7F | Extended WRAM | 128KB | Additional RAM | `bus.go:57-64` |
| 0x80-0xFF | Reserved | - | Reserved for future use | Not implemented |

### WRAM (Bank 0, 0x0000-0x7FFF)

**Evidence:** `internal/memory/bus.go:7, 40-42`

- **Size**: 32KB
- **Access**: Read/Write, 8-bit and 16-bit
- **Stack**: Grows downward from 0x1FFF (`cpu.go:107, 519-523`)
- **Usage**: General purpose RAM, stack, variables

### I/O Registers (Bank 0, 0x8000-0x8FFF)

**Evidence:** `internal/memory/bus.go:128-187`

- **PPU Registers**: 0x8000-0x8FFF (`bus.go:131-136`)
- **APU Registers**: 0x9000-0x9FFF (`bus.go:138-144`)
- **Input Registers**: 0xA000-0xAFFF (`bus.go:146-152`)
- **System Registers**: 0x8000-0x80FF (within PPU space)

### ROM Space (Banks 0x01-0x7D)

**Evidence:** `internal/memory/cartridge.go:66-78`

- **LoROM Mapping**: ROM appears at offset 0x8000+
- **Formula**: `romOffset = (bank-1) × 32768 + (offset - 0x8000)` (`cartridge.go:72`)
- **Read-Only**: Writes are ignored (`bus.go:83-85`)
- **Max Size**: 7.8MB (125 banks × 32KB)
- **Unmapped Space**: Offset 0x0000-0x7FFF in ROM banks returns 0 (`cartridge.go:69-70`)

### Extended WRAM (Banks 0x7E-0x7F)

**Evidence:** `internal/memory/bus.go:10, 57-64`

- **Size**: 128KB (2 banks × 64KB)
- **Access**: Read/Write, 8-bit and 16-bit
- **Address Calculation**: `extOffset = (bank-126) × 65536 + offset` (`bus.go:59`)

---

## PPU (Graphics) Specification

### VRAM (Video RAM)

**Evidence:** `internal/ppu/ppu.go:11`

- **Size**: 64KB
- **Organization**: Tiles, tilemaps, sprite data
- **Access**: 8-bit via VRAM_DATA register (0x8010)
- **Addressing**: 16-bit address via VRAM_ADDR_L/H (0x800E-0x800F)
- **Auto-increment**: Address increments after each read/write (`ppu.go:164, 301`)

### CGRAM (Color Graphics RAM)

**Evidence:** `internal/ppu/ppu.go:14, 306-343, 1105-1133`

- **Size**: 512 bytes (256 colors × 2 bytes)
- **Format**: RGB555 (5 bits per channel)
- **Organization**: 16 palettes × 16 colors each
- **Access**: 16-bit via CGRAM_DATA (0x8013) with write latch
  - First write: low byte stored in latch
  - Second write: high byte completes RGB555 value
- **Storage**: Little-endian (low byte first, high byte second) (`ppu.go:335-336`)
- **RGB555 Decoding** (`ppu.go:1118-1130`):
  - R: bits 10-14 from high byte (bits 2-6)
  - G: bits 5-9, split between high (bits 0-1) and low (bits 5-7)
  - B: bits 0-4 from low byte

### OAM (Object Attribute Memory)

**Evidence:** `internal/ppu/ppu.go:17, 346-404`

- **Size**: 768 bytes (128 sprites × 6 bytes)
- **Access**: 8-bit via OAM_DATA register (0x8015)
- **Addressing**: OAM_ADDR (0x8014) sets sprite ID (0-127)
- **Auto-increment**: Byte index increments, wraps to next sprite after 6 bytes
- **Write Protection**: Writes blocked during visible rendering (scanlines 0-199) (`ppu.go:356-360, 373-377`)

**OAM Sprite Format (6 bytes per sprite) - Evidence: `ppu.go:963-991`**
- Byte 0: X position low byte (unsigned)
- Byte 1: X position high byte (bit 0 = sign bit, extends to 9-bit signed)
- Byte 2: Y position (8-bit, 0-255)
- Byte 3: Tile index
- Byte 4: Attributes
  - Bits [3:0]: Palette index
  - Bit 4: Flip X
  - Bit 5: Flip Y
  - Bits [7:6]: Priority (0-3)
- Byte 5: Control
  - Bit 0: Enable
  - Bit 1: 16×16 size (0=8×8, 1=16×16)
  - Bits [3:2]: Blend mode (0=normal, 1=alpha, 2=additive, 3=subtractive)
  - Bits [7:4]: Alpha value (0-15)

### Background Layers

**Evidence:** `internal/ppu/ppu.go:20, 119-139, 817-944`

- **4 Independent Layers**: BG0, BG1, BG2, BG3
- **Tile Sizes**: 8×8 or 16×16 pixels (configurable per layer via BG_CONTROL bit 1)
- **Priority**: BG3=3, BG2=2, BG1=1, BG0=0 (higher number = higher priority) (`scanline.go:232-260`)
- **Scroll**: Per-layer X/Y scroll registers (16-bit signed)
- **Tilemap Base**: Configurable per layer (default 0x4000) (`ppu.go:580-583`)
- **Tilemap Size**: 32×32 tiles (`ppu.go:844-846`)

### Sprites

**Evidence:** `internal/ppu/ppu.go:17, 324-377, 955-1063`

- **Max Sprites**: 128
- **Sizes**: 8×8 or 16×16 pixels (per-sprite via control bit 1)
- **Priority**: 4 levels (0-3, from sprite attributes bits [7:6])
- **Transparency**: Color index 0 is transparent (`ppu.go:497, 1052`)
- **Flip**: Horizontal and/or vertical flip (attributes bits 4-5)
- **Blending**: Alpha blending, additive, subtractive modes (control bits [3:2], [7:4])

### Matrix Mode

**Evidence:** `internal/ppu/ppu.go:127-135, 647-827`

- **Mode 7-style Effects**: Per-layer affine transformations
- **Transformations**: Rotation, scaling, perspective
- **Matrix Format**: 8.8 fixed point (int16, 1.0 = 0x0100)
- **Matrix Parameters**: A, B, C, D (transformation matrix), CenterX, CenterY
- **Mirroring**: Horizontal and/or vertical mirroring
- **Outside Mode**: 0=repeat/wrap, 1=backdrop, 2=character #0
- **Direct Color**: Bypass CGRAM, use direct RGB (bit 5 of matrix control)

### Rendering Pipeline

**Evidence:** `internal/ppu/scanline.go:214-322`

1. **Priority Sorting**: Collect backgrounds and sprites, sort by priority
2. **Background Rendering**: BG0 → BG1 → BG2 → BG3 (lower priority first)
3. **Sprite Rendering**: Sorted by priority, then by index
4. **Blending**: Alpha blending, additive, subtractive modes
5. **Output**: 320×200 pixel frame buffer

### HDMA (Horizontal DMA)

**Evidence:** `internal/ppu/ppu.go:38-42, 955-1047`

- **Per-scanline Updates**: Scroll and matrix parameters updated per scanline
- **HDMA Table**: Stored in VRAM, 64 bytes per scanline (max 4 layers × 16 bytes)
- **Table Format**: Per layer, 16 bytes (scroll + matrix if enabled)
- **Control**: HDMA_CONTROL (0x805D) - bit 0=enable, bits 1-4=layer enable

---

## APU (Audio) Specification

### Audio Channels

**Evidence:** `internal/apu/apu.go:19, 56-87`

- **4 Independent Channels**: Each with frequency, volume, waveform, duration
- **Waveforms**: Sine, Square, Saw, Noise (`apu.go:437-468`)
- **Sample Rate**: 44,100 Hz (`emulator.go:76`)
- **Output**: Mono (single channel, can be mixed to stereo by host)

### Channel Parameters

**Evidence:** `internal/apu/apu.go:56-87, 173-334`

- **Frequency**: 16-bit (0-65535, controls pitch)
  - Low byte: 0x9000+channel*8+0
  - High byte: 0x9000+channel*8+1 (triggers phase reset if changed)
- **Volume**: 8-bit (0-255, per-channel) (`apu.go:266-267`)
- **Master Volume**: 8-bit (0-255, global) (`apu.go:331-333`)
- **Duration**: 16-bit (frames, 0 = infinite) (`apu.go:305-320`)
- **Duration Mode**: 0=stop when done, 1=loop/restart (`apu.go:322-325`)
- **Waveform**: 0=sine, 1=square, 2=saw, 3=noise (`apu.go:274-284`)

### Audio Generation

**Evidence:** `internal/apu/apu.go:373-502`

- **Samples per Frame**: 735 (44,100 Hz / 60 FPS) (`emulator.go:150, 204`)
- **Generation**: Continuous during frame execution
- **Phase Accumulator**: 32-bit fixed-point (0-2^32 represents 0-2π)
- **Phase Increment**: Calculated from frequency and sample rate
- **Completion Status**: Flags set when duration expires (if not looping) (`apu.go:108-127, 509-561`)

### Waveform Details

**Evidence:** `internal/apu/apu.go:437-468`

- **Sine**: `sin(phase)` gives -1.0 to 1.0
- **Square**: 50% duty cycle (1.0 if phase < π, -1.0 otherwise)
- **Sawtooth**: Linear ramp from -1.0 to 1.0
- **Noise**: 15-bit LFSR (polynomial: x^15 + x^14 + 1)

---

## Input System Specification

### Controllers

**Evidence:** `internal/input/input.go:5-10`

- **2 Controllers**: Controller 1 and Controller 2
- **12 Buttons per Controller**: UP, DOWN, LEFT, RIGHT, A, B, X, Y, L, R, START, Z
- **Latch Mechanism**: Write 1 to latch register to capture state, 0 to release

### Button Reading

**Evidence:** `internal/input/input.go:23-54`

1. Write 1 to INPUT_LATCH (Controller 1: 0xA001, Controller 2: 0xA003) - captures current button state
2. Read INPUT_DATA_L/H to get button states
3. Write 0 to INPUT_LATCH (releases latch)

**Button States**: 1 = pressed, 0 = released

**Button Mapping (Evidence: `input.go:88-101`):**
- Low byte: UP (bit 0), DOWN (bit 1), LEFT (bit 2), RIGHT (bit 3), A (bit 4), B (bit 5), X (bit 6), Y (bit 7)
- High byte: L (bit 0), R (bit 1), START (bit 2), Z (bit 3)

---

## I/O Register Map

### System Registers (0x8000-0x80FF)

**Evidence:** `internal/ppu/ppu.go:191-233`

| Address | Name | Size | Description | Evidence |
|---------|------|------|-------------|----------|
| 0x803E | VBLANK_FLAG | 8-bit | VBlank flag (bit 0), cleared when read | `ppu.go:191-229` |
| 0x803F | FRAME_COUNTER_LOW | 8-bit | Frame counter low byte | `ppu.go:230-231` |
| 0x8040 | FRAME_COUNTER_HIGH | 8-bit | Frame counter high byte | `ppu.go:232-233` |

**VBLANK_FLAG Behavior (Evidence: `ppu.go:191-229`):**
- Set at end of scanline 199 (before scanline 200 starts) (`scanline.go:96-102`)
- Cleared when read (one-shot latch)
- Re-set if still in VBlank period (allows multiple reads during VBlank)

### PPU Registers (0x8000-0x8FFF)

**Evidence:** `internal/ppu/ppu.go:159-637`

#### Background Scroll Registers

| Address | Name | Size | Description | Evidence |
|---------|------|------|-------------|----------|
| 0x8000 | BG0_SCROLLX_L | 8-bit | BG0 scroll X low byte | `ppu.go:253-254` |
| 0x8001 | BG0_SCROLLX_H | 8-bit | BG0 scroll X high byte | `ppu.go:255-256` |
| 0x8002 | BG0_SCROLLY_L | 8-bit | BG0 scroll Y low byte | `ppu.go:257-258` |
| 0x8003 | BG0_SCROLLY_H | 8-bit | BG0 scroll Y high byte | `ppu.go:259-260` |
| 0x8004 | BG1_SCROLLX_L | 8-bit | BG1 scroll X low byte | `ppu.go:263-264` |
| 0x8005 | BG1_SCROLLX_H | 8-bit | BG1 scroll X high byte | `ppu.go:265-266` |
| 0x8006 | BG1_SCROLLY_L | 8-bit | BG1 scroll Y low byte | `ppu.go:267-268` |
| 0x8007 | BG1_SCROLLY_H | 8-bit | BG1 scroll Y high byte | `ppu.go:269-270` |
| 0x800A | BG2_SCROLLX_L | 8-bit | BG2 scroll X low byte | `ppu.go:281-282` |
| 0x800B | BG2_SCROLLX_H | 8-bit | BG2 scroll X high byte | `ppu.go:283-284` |
| 0x800C | BG2_SCROLLY_L | 8-bit | BG2 scroll Y low byte | `ppu.go:285-286` |
| 0x800D | BG2_SCROLLY_H | 8-bit | BG2 scroll Y high byte | `ppu.go:287-288` |
| 0x8022 | BG3_SCROLLX_L | 8-bit | BG3 scroll X low byte | `ppu.go:446-447` |
| 0x8023 | BG3_SCROLLX_H | 8-bit | BG3 scroll X high byte | `ppu.go:448-449` |
| 0x8024 | BG3_SCROLLY_L | 8-bit | BG3 scroll Y low byte | `ppu.go:450-451` |
| 0x8025 | BG3_SCROLLY_H | 8-bit | BG3 scroll Y high byte | `ppu.go:452-453` |

#### Background Control Registers

| Address | Name | Size | Description | Evidence |
|---------|------|------|-------------|----------|
| 0x8008 | BG0_CONTROL | 8-bit | Bit 0=enable, bit 1=tile size | `ppu.go:273-275` |
| 0x8009 | BG1_CONTROL | 8-bit | Bit 0=enable, bit 1=tile size | `ppu.go:276-278` |
| 0x8021 | BG2_CONTROL | 8-bit | Bit 0=enable, bit 1=tile size | `ppu.go:443-445` |
| 0x8026 | BG3_CONTROL | 8-bit | Bit 0=enable, bit 1=tile size | `ppu.go:454-456` |

#### VRAM Access Registers

| Address | Name | Size | Description | Evidence |
|---------|------|------|-------------|----------|
| 0x800E | VRAM_ADDR_L | 8-bit | VRAM address low byte | `ppu.go:291-292` |
| 0x800F | VRAM_ADDR_H | 8-bit | VRAM address high byte | `ppu.go:293-294` |
| 0x8010 | VRAM_DATA | 8-bit | VRAM data (auto-increments address) | `ppu.go:295-304` |

#### CGRAM Access Registers

| Address | Name | Size | Description | Evidence |
|---------|------|------|-------------|----------|
| 0x8012 | CGRAM_ADDR | 8-bit | CGRAM address (palette + color index) | `ppu.go:307-315` |
| 0x8013 | CGRAM_DATA | 16-bit | CGRAM data (RGB555, low byte then high byte) | `ppu.go:316-343` |

#### OAM Access Registers

| Address | Name | Size | Description | Evidence |
|---------|------|------|-------------|----------|
| 0x8014 | OAM_ADDR | 8-bit | OAM address (sprite ID, 0-127) | `ppu.go:346-366` |
| 0x8015 | OAM_DATA | 8-bit | OAM data (auto-increments byte index) | `ppu.go:368-404` |

#### Matrix Mode Registers (BG0 - Legacy)

| Address | Name | Size | Description | Evidence |
|---------|------|------|-------------|----------|
| 0x8018 | MATRIX_CONTROL | 8-bit | Bit 0=enable, bit 1=mirror H, bit 2=mirror V, bits [4:3]=outside mode, bit 5=direct color | `ppu.go:407-416` |
| 0x8019 | MATRIX_A_L | 8-bit | Matrix A low byte | `ppu.go:417-419` |
| 0x801A | MATRIX_A_H | 8-bit | Matrix A high byte | `ppu.go:420-422` |
| 0x801B | MATRIX_B_L | 8-bit | Matrix B low byte | `ppu.go:423-425` |
| 0x801C | MATRIX_B_H | 8-bit | Matrix B high byte | `ppu.go:426-428` |
| 0x801D | MATRIX_C_L | 8-bit | Matrix C low byte | `ppu.go:429-431` |
| 0x801E | MATRIX_C_H | 8-bit | Matrix C high byte | `ppu.go:432-434` |
| 0x801F | MATRIX_D_L | 8-bit | Matrix D low byte | `ppu.go:435-437` |
| 0x8020 | MATRIX_D_H | 8-bit | Matrix D high byte | `ppu.go:438-440` |
| 0x8027 | MATRIX_CENTER_X_L | 8-bit | Matrix center X low byte | `ppu.go:459-461` |
| 0x8028 | MATRIX_CENTER_X_H | 8-bit | Matrix center X high byte | `ppu.go:462-464` |
| 0x8029 | MATRIX_CENTER_Y_L | 8-bit | Matrix center Y low byte | `ppu.go:465-467` |
| 0x802A | MATRIX_CENTER_Y_H | 8-bit | Matrix center Y high byte | `ppu.go:468-470` |

#### Matrix Mode Registers (BG1-BG3 - Per-Layer)

**BG1 (Evidence: `ppu.go:472-502`):**
- 0x802B-0x8037: BG1 matrix registers (same layout as BG0)

**BG2 (Evidence: `ppu.go:504-534`):**
- 0x8038-0x8044: BG2 matrix registers (same layout as BG0)

**BG3 (Evidence: `ppu.go:536-566`):**
- 0x8045-0x8051: BG3 matrix registers (same layout as BG0)

#### Windowing Registers

| Address | Name | Size | Description | Evidence |
|---------|------|------|-------------|----------|
| 0x8052 | WINDOW0_LEFT | 8-bit | Window 0 left | `ppu.go:569-570` |
| 0x8053 | WINDOW0_RIGHT | 8-bit | Window 0 right | `ppu.go:571-572` |
| 0x8054 | WINDOW0_TOP | 8-bit | Window 0 top | `ppu.go:573-574` |
| 0x8055 | WINDOW0_BOTTOM | 8-bit | Window 0 bottom | `ppu.go:575-576` |
| 0x8056 | WINDOW1_LEFT | 8-bit | Window 1 left | `ppu.go:577-578` |
| 0x8057 | WINDOW1_RIGHT | 8-bit | Window 1 right | `ppu.go:579-580` |
| 0x8058 | WINDOW1_TOP | 8-bit | Window 1 top | `ppu.go:581-582` |
| 0x8059 | WINDOW1_BOTTOM | 8-bit | Window 1 bottom | `ppu.go:583-584` |
| 0x805A | WINDOW_CONTROL | 8-bit | Window control/logic | `ppu.go:585-586` |
| 0x805B | WINDOW_MAIN_ENABLE | 8-bit | Window main enable | `ppu.go:587-588` |
| 0x805C | WINDOW_SUB_ENABLE | 8-bit | Window sub enable | `ppu.go:589-590` |

#### HDMA Registers

| Address | Name | Size | Description | Evidence |
|---------|------|------|-------------|----------|
| 0x805D | HDMA_CONTROL | 8-bit | Bit 0=enable, bits 1-4=layer enable | `ppu.go:593-598` |
| 0x805E | HDMA_TABLE_BASE_L | 8-bit | HDMA table base low byte | `ppu.go:599-600` |
| 0x805F | HDMA_TABLE_BASE_H | 8-bit | HDMA table base high byte | `ppu.go:601-602` |

#### DMA Registers

| Address | Name | Size | Description | Evidence |
|---------|------|------|-------------|----------|
| 0x8060 | DMA_CONTROL | 8-bit | Bit 0=enable, bit 1=mode, bits [3:2]=dest type (write-only) | `ppu.go:610-632` |
| 0x8060 | DMA_STATUS | 8-bit | Bit 0=DMA active (read-only) | `ppu.go:239-244` |
| 0x8061 | DMA_SOURCE_BANK | 8-bit | Source bank (1-125 for ROM) (write-only) | `ppu.go:633-634` |
| 0x8062 | DMA_SOURCE_OFFSET_L | 8-bit | Source offset low byte (write-only) | `ppu.go:635-636` |
| 0x8063 | DMA_SOURCE_OFFSET_H | 8-bit | Source offset high byte (write-only) | `ppu.go:637-638` |
| 0x8064 | DMA_DEST_ADDR_L | 8-bit | Destination address low byte (write-only) | `ppu.go:639-640` |
| 0x8065 | DMA_DEST_ADDR_H | 8-bit | Destination address high byte (write-only) | `ppu.go:641-642` |
| 0x8066 | DMA_LENGTH_L | 8-bit | Transfer length low byte (read/write) | `ppu.go:643-644` |
| 0x8067 | DMA_LENGTH_H | 8-bit | Transfer length high byte (read/write) | `ppu.go:645-646` |

**DMA Behavior (Evidence: `ppu.go:652-717, scanline.go:66-93`):**
- Mode 0: Copy (read from source, write to destination)
- Mode 1: Fill (read fill value once, write to all destinations)
- Destination types: 0=VRAM, 1=CGRAM, 2=OAM
- **Cycle-Accurate**: DMA executes one byte per cycle via `stepDMA()` during `StepPPU()` (`scanline.go:67, 81, 93`)
- **Timing**: One byte transferred per CPU/PPU cycle (~7.67 MHz)
- **Status**: DMA_STATUS (0x8060 read) returns 0x01 when active, 0x00 when idle
- **Progress**: DMA completes when DMAProgress >= DMALength

### APU Registers (0x9000-0x9FFF)

**Evidence:** `internal/apu/apu.go:101-334`

#### Channel Registers (per channel, 8 bytes each)

| Offset | Name | Size | Description | Evidence |
|--------|------|------|-------------|----------|
| +0x00 | FREQ_LOW | 8-bit | Frequency low byte | `apu.go:136-137` |
| +0x01 | FREQ_HIGH | 8-bit | Frequency high byte (triggers update) | `apu.go:138-139, 208-262` |
| +0x02 | VOLUME | 8-bit | Volume (0-255) | `apu.go:140-141, 266-267` |
| +0x03 | CONTROL | 8-bit | Enable (bit 0), waveform (bits 1-2), PCM mode (bit 4) | `apu.go:142-158, 269-300` |
| +0x04 | DURATION_LOW | 8-bit | Duration low byte (frames) | `apu.go:159-161, 305-311` |
| +0x05 | DURATION_HIGH | 8-bit | Duration high byte (frames) | `apu.go:162, 313-320` |
| +0x06 | DURATION_MODE | 8-bit | Duration mode (bit 0: 0=stop, 1=loop) | `apu.go:163-164, 322-325` |
| +0x07 | Reserved | 8-bit | Reserved | `apu.go:166-169` |

**Channel Base Addresses (Evidence: `apu.go:130-131`):**
- Channel 0: 0x9000 (offsets 0x00-0x07)
- Channel 1: 0x9008 (offsets 0x08-0x0F)
- Channel 2: 0x9010 (offsets 0x10-0x17)
- Channel 3: 0x9018 (offsets 0x18-0x1F)

#### Master Control Registers

| Address | Name | Size | Description | Evidence |
|---------|------|------|-------------|----------|
| 0x9020 | MASTER_VOLUME | 8-bit | Master volume (0-255) | `apu.go:331-333` |
| 0x9021 | COMPLETION_STATUS | 8-bit | Channel completion flags (bits [3:0], cleared when read) | `apu.go:108-127` |

**Note:** Spec v2.0 incorrectly listed MASTER_VOLUME at 0x9040. Actual address is 0x9020.

### Input Registers (0xA000-0xAFFF)

**Evidence:** `internal/input/input.go:23-54`

| Address | Name | Size | Description | Evidence |
|---------|------|------|-------------|----------|
| 0xA000 | INPUT_DATA_L | 8-bit | Controller 1 buttons (low byte) | `input.go:25-26` |
| 0xA001 | INPUT_DATA_H | 8-bit | Controller 1 buttons (high byte) / Latch control | `input.go:27-28, 41-46` |
| 0xA002 | INPUT_DATA2_L | 8-bit | Controller 2 buttons (low byte) | `input.go:29-30` |
| 0xA003 | INPUT_DATA2_H | 8-bit | Controller 2 buttons (high byte) / Latch control | `input.go:31-32, 47-52` |

**Note:** Spec v2.0 incorrectly listed INPUT_LATCH at 0xA004. Actual implementation uses 0xA001 for Controller 1 latch and 0xA003 for Controller 2 latch.

---

## Timing and Synchronization

### Frame Timing

**Evidence:** `internal/ppu/scanline.go:6-27, internal/emulator/emulator.go:114-147`

- **Frame Rate**: 60 FPS
- **Frame Duration**: 16.67 ms
- **CPU Clock**: ~7.67 MHz (7,670,000 Hz)
- **Cycles per Frame**: 127,820 cycles (220 scanlines × 581 dots per scanline)
- **PPU Clock**: Same as CPU (unified clock)
- **APU Clock**: 44,100 Hz (every ~174 CPU cycles)

### Frame Execution Order

**Evidence:** `internal/emulator/emulator.go:187-309, internal/ppu/scanline.go:29-117`

1. **Frame Start** (`ppu/scanline.go:179-196`):
   - Clear VBlank flag
   - Increment frame counter
   - Clear output buffer

2. **PPU Rendering** (during frame):
   - Render scanlines 0-199 (visible period)
   - Each scanline: 320 visible dots + 261 HBlank dots = 581 dots
   - HDMA updates at start of each visible scanline

3. **VBlank Period** (scanlines 200-219):
   - VBlank flag set at end of scanline 199
   - VBlank interrupt triggered
   - OAM writes allowed

4. **CPU Execution** (during frame):
   - CPU runs for 127,820 cycles per frame
   - Can read VBlank flag, frame counter, completion status

5. **Audio Generation** (continuous):
   - Generate 735 samples per frame (44,100 Hz / 60 FPS)
   - Independent of frame timing

### VBlank Flag (0x803E)

**Evidence:** `internal/ppu/ppu.go:191-229, internal/ppu/scanline.go:96-102`

- **Set**: At end of scanline 199 (before scanline 200 starts)
- **Cleared**: When read (one-shot latch)
- **Re-set**: If still in VBlank period (allows multiple reads during VBlank)
- **Behavior**: Hardware-accurate, matches NES/SNES pattern

**FPGA Implementation:**
```verilog
always @(posedge clk) begin
    if (scanline_end && scanline == 199) begin
        vblank_flag <= 1'b1;
    end else if (read_vblank) begin
        vblank_flag <= 1'b0;  // Clear when read
        if (in_vblank) begin
            vblank_flag <= 1'b1;  // Re-set if still in VBlank
        end
    end
end
```

### Frame Counter (0x803F/0x8040)

**Evidence:** `internal/ppu/ppu.go:54, 230-233, 187`

- **16-bit Counter**: Increments once per frame at frame start
- **Low Byte**: 0x803F
- **High Byte**: 0x8040

---

## ROM Format

### ROM Header (32 bytes)

**Evidence:** `internal/memory/cartridge.go:28-60`

| Offset | Size | Name | Description | Evidence |
|--------|------|------|-------------|----------|
| 0x00 | 4 | Magic | "RMCF" (0x46434D52) | `cartridge.go:35-37` |
| 0x04 | 2 | Version | ROM format version (1) | `cartridge.go:40-42` |
| 0x06 | 4 | ROM Size | Total ROM size in bytes | `cartridge.go:45` |
| 0x0A | 2 | Entry Bank | Entry point bank (1-125) | `cartridge.go:96` |
| 0x0C | 2 | Entry Offset | Entry point offset (0x8000+) | `cartridge.go:97` |
| 0x0E | 2 | Mapper Flags | Mapper type (0 = LoROM) | Not used in emulator |
| 0x10 | 4 | Checksum | ROM checksum (currently 0) | Not verified in emulator |
| 0x14 | 12 | Reserved | Reserved for future use | Not used |

### ROM Data

- **Code Section**: Variable size, little-endian 16-bit words
- **Asset Section**: Optional, appended after code
- **Total Size**: Up to 7.8MB (125 banks × 32KB)

---

## Reset & Power-On State

### CPU Reset State

**Evidence:** `internal/cpu/cpu.go:89-112`

- **Registers R0-R7**: 0
- **PCBank, PCOffset, PBR**: NOT reset (set by SetEntryPoint() after ROM load)
- **DBR**: 0
- **SP**: 0x1FFF (top of WRAM)
- **Flags**: 0 (all flags clear)
- **Cycles**: 0
- **InterruptMask**: 0
- **InterruptPending**: 0

**Note:** PC is NOT reset to prevent corruption if Reset() is called after ROM load.

### Memory Reset State

**Evidence:** `internal/memory/bus.go:7, 10` (arrays initialized to zero)

- **WRAM**: All zeros
- **Extended WRAM**: All zeros
- **ROM**: Loaded from file (not reset)

### PPU Reset State

**Evidence:** `internal/ppu/ppu.go:147-157` (NewPPU initializes to zero)

- **VRAM**: All zeros
- **CGRAM**: All zeros
- **OAM**: All zeros
- **Background Layers**: All disabled, scroll=0
- **Matrix Mode**: Disabled
- **VBlank Flag**: false
- **Frame Counter**: 0

### APU Reset State

**Evidence:** `internal/apu/apu.go:90-98` (NewAPU initializes)

- **Channels**: All disabled, frequency=0, volume=0
- **Master Volume**: 255 (maximum)
- **Completion Status**: 0

### Input Reset State

**Evidence:** `internal/input/input.go:13-19` (NewInputSystem initializes)

- **Controller Buttons**: 0 (all buttons released)
- **Latch**: false (not latched)

---

## Open Bus & Undefined Behavior

### Open Bus Behavior

**Unknown in Emulator** - The emulator does not implement open bus behavior. Reads from unmapped addresses return 0.

**Undefined Behaviors:**
1. **Reading from unmapped ROM space** (offset 0x0000-0x7FFF in ROM banks): Returns 0 (`cartridge.go:69-70`)
2. **Writing to ROM space**: Ignored (`bus.go:83-85`)
3. **Writing to I/O registers during rendering**: Some registers may have undefined behavior
4. **OAM writes during visible rendering**: Blocked (`ppu.go:356-360, 373-377`)

### Undefined Instruction Encodings

**Evidence:** `internal/cpu/instructions.go:107-111`

- **MOV Mode 8-15**: Reserved, treated as NOP
- **Unknown opcodes**: Return error (`cpu.go:364`)

### Division by Zero

**Evidence:** `internal/cpu/instructions.go:190-197`

- **Result**: 0xFFFF (maximum value)
- **Flag D**: Set (division by zero flag)
- **Cycles**: 4

---

## Spec Corrections from v2.0

1. **CPU Speed**: Changed from 10 MHz to ~7.67 MHz (Genesis-like speed)
   - Evidence: `emulator.go:114`
   - Impact: Timing calculations, cycle counts

2. **Cycles per Frame**: Changed from 166,667 to 127,820
   - Evidence: `emulator.go:147, scanline.go:17`
   - Calculation: 220 scanlines × 581 dots = 127,820 cycles

3. **PPU Timing**: Changed from 360 dots per scanline to 581 dots
   - Evidence: `scanline.go:17-19`
   - Visible: 320 dots, HBlank: 261 dots

4. **APU Channel Registers**: Changed from 4 bytes to 8 bytes per channel
   - Evidence: `apu.go:130-131`
   - Added: DURATION_LOW, DURATION_HIGH, DURATION_MODE registers

5. **APU Channel Base Addresses**: Corrected to 8-byte spacing (not 16-byte)
   - Evidence: `apu.go:130-131` (channel = (offset / 8) & 0x3)
   - Correct addresses: Channel 0: 0x9000, Channel 1: 0x9008, Channel 2: 0x9010, Channel 3: 0x9018
   - Previous spec incorrectly listed: 0x9000, 0x9010, 0x9020, 0x9030 (16-byte spacing)

6. **APU Master Volume Address**: Changed from 0x9040 to 0x9020
   - Evidence: `apu.go:331`
   - Impact: Register map correction

7. **Input Latch Address**: Changed from 0xA004 to 0xA001 (Controller 1) and 0xA003 (Controller 2)
   - Evidence: `input.go:41-52`
   - Impact: Separate latch registers per controller

8. **VBlank Flag Timing**: Clarified to set at end of scanline 199, not start of scanline 200
   - Evidence: `scanline.go:96-102`
   - Impact: Timing accuracy

9. **OAM Write Protection**: Added explicit protection during visible rendering (scanlines 0-199)
   - Evidence: `ppu.go:356-360, 373-377`
   - Impact: Hardware-accurate behavior

10. **HDMA Table Format**: Documented 16 bytes per layer (scroll + matrix if enabled)
   - Evidence: `ppu.go:955-1047`
   - Impact: HDMA implementation

11. **Sprite Blending**: Added blend modes and alpha support
    - Evidence: `ppu.go:379-437, 505-519`
    - Impact: Enhanced sprite rendering

---

## Implementation Confidence Notes

### High Confidence (Directly Verified in Code)

- CPU instruction set and encoding
- Memory map and bank switching
- PPU register addresses and bit layouts
- APU channel parameters and waveforms
- Input button mapping
- ROM format and loading
- Frame timing and VBlank behavior

### Medium Confidence (Inferred from Code Structure)

- Open bus behavior (assumed to return 0)
- Reset state for some registers (initialized to zero)
- Timing edge cases (some may need hardware verification)

### Low Confidence / Unknown

- Physical connector pinouts (not in emulator code)
- FPGA resource requirements (estimated, not measured)
- Power consumption (estimated, not measured)
- Exact timing of register side effects (some may be cycle-accurate in hardware)

---

## FPGA Implementation Guidelines

### Clock Domains

- **CPU Clock**: ~7.67 MHz (main system clock)
- **PPU Clock**: Same as CPU (unified clock)
- **Audio Clock**: 44.1 kHz (audio sample generation)

### Synchronization

- **VBlank Signal**: Synchronized to PPU clock
- **Frame Counter**: Synchronized to PPU clock
- **Completion Status**: Synchronized to CPU clock
- **Cross-Clock Domain**: Use proper synchronization (FIFOs, handshaking)

### State Machines

- **APU**: Simple state machine for channel control
- **PPU**: Rendering pipeline with clear stages (scanline/dot stepping)
- **CPU**: Instruction fetch/decode/execute pipeline

### Register Access

- **I/O Registers**: Byte-addressable, simple address decoding
- **Address Decoding**: 0x8000-0x8FFF = PPU, 0x9000-0x9FFF = APU, 0xA000-0xAFFF = Input
- **Read/Write**: Standard memory-mapped I/O

### Memory Interfaces

- **WRAM**: Dual-port RAM (CPU read/write, PPU read)
- **VRAM**: Dual-port RAM (CPU write, PPU read)
- **CGRAM**: Dual-port RAM (CPU write, PPU read)
- **OAM**: Dual-port RAM (CPU write, PPU read, write-protected during visible rendering)

---

**End of Complete Hardware Specification v2.1**
