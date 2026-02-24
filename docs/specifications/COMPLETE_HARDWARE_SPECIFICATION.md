# Nitro-Core-DX Complete Hardware Specification

**Version 2.0**  
**Last Updated: January 30, 2026**  
**Purpose: Complete hardware specification for FPGA implementation including all physical interfaces**

> **📌 FPGA-Ready**: This specification is designed for FPGA implementation. All timing, signals, and register layouts are hardware-accurate and can be directly translated to Verilog/VHDL.
>
> **⚠️ Historical Note (APU/FM):** This `v2.0` file predates later APU corrections and FM extension implementation work. For current APU register/audio details, use `docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md` and `docs/specifications/APU_FM_OPM_EXTENSION_SPEC.md`.

---

## Table of Contents

1. [System Overview](#system-overview)
2. [FPGA Architecture Recommendation](#fpga-architecture-recommendation)
3. [Physical Connectors](#physical-connectors)
   - [Cartridge Connector](#cartridge-connector)
   - [Controller Connectors](#controller-connectors)
   - [Expansion Port](#expansion-port)
4. [System Architecture](#system-architecture)
   - [CPU Architecture](#cpu-architecture)
   - [Memory Map](#memory-map)
   - [PPU (Graphics) Specification](#ppu-graphics-specification)
   - [APU (Audio) Specification](#apu-audio-specification)
   - [Input System Specification](#input-system-specification)
5. [I/O Register Map](#io-register-map)
6. [Timing and Synchronization](#timing-and-synchronization)
7. [ROM Format](#rom-format)
8. [FPGA Implementation Guidelines](#fpga-implementation-guidelines)

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
| **Matrix Mode** | Mode 7-style effects with large world support |
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
│  ├─ Controller 1 & 2 (DB-9 connectors)                 │
│  └─ 12-button Support                                 │
└─────────────────────────────────────────────────────────┘
```

---

## FPGA Architecture Recommendation

### Executive Summary

**Recommendation: Single FPGA Solution**

For cost-effective production, use **one medium-sized FPGA**. This provides the best balance of cost, complexity, and performance.

**Recommended FPGA**: **Xilinx Spartan-7 XC7S50**
- **LUTs**: 52,000 (plenty of headroom)
- **BRAM**: 180 blocks (more than enough)
- **DSP Slices**: 120 (useful for matrix math)
- **Cost**: ~$20-25 per chip (volume pricing)
- **Toolchain**: Vivado (free WebPack edition)

**Estimated Production Cost:**
- **Prototype**: ~$100-150 (development board)
- **Small batch (10-100)**: ~$50-80 per console
- **Medium batch (1,000)**: ~$40-60 per console
- **Large batch (10,000+)**: ~$30-50 per console

### System Resource Requirements

| Component | Estimated LUTs | Estimated BRAM | Notes |
|-----------|----------------|----------------|-------|
| **CPU Core** | 5,000-10,000 | 2-4 blocks | Custom 10 MHz CPU, simple instruction set |
| **PPU (Graphics)** | 15,000-30,000 | 20-40 blocks | Most complex: 4 layers, 128 sprites, matrix mode |
| **APU (Audio)** | 2,000-5,000 | 4-8 blocks | 4 channels, waveform generation |
| **Memory Controllers** | 3,000-5,000 | 10-20 blocks | WRAM, VRAM, CGRAM, OAM, cartridge |
| **I/O Interfaces** | 1,000-2,000 | 0-2 blocks | Controllers, expansion port |
| **System Logic** | 2,000-3,000 | 2-4 blocks | Clocking, reset, synchronization |
| **TOTAL** | **28,000-55,000 LUTs** | **40-78 BRAM blocks** | Need 30-50% headroom |

**Spartan-7 XC7S50 provides 52,000 LUTs and 180 BRAM blocks** - sufficient with good headroom.

### Why Single FPGA?

**Advantages:**
- ✅ **Lowest cost** (~$20-25 per FPGA vs $30-60 for two, $45-75 for three)
- ✅ **Simplest PCB** (4-layer vs 6-8 layer for multi-FPGA)
- ✅ **No inter-FPGA communication** (no latency, no sync issues)
- ✅ **Lower power consumption** (0.8-2.3W vs 1.3-3.1W for two FPGAs)
- ✅ **Easier debugging** (all logic in one place)
- ✅ **Lower manufacturing cost** (simpler assembly, higher yield)
- ✅ **Better performance** (no communication overhead)

**Alternatives Considered:**
- **Two FPGAs**: Higher cost ($60-90 per console), complex PCB, synchronization challenges
- **Three FPGAs**: Highest cost ($84-125 per console), very complex PCB, severe synchronization issues

**Conclusion**: Single FPGA is the clear winner for cost-effectiveness and simplicity.

### Alternative FPGA Options

1. **Intel Cyclone IV EP4CE22** (~$15-20)
   - 22,000 LEs, 66 M9K blocks
   - Lower cost, but might be tight on resources
   - Well-documented, many retro console examples

2. **Lattice ECP5 LFE5U-25F** (~$18-22)
   - 24,000 LUTs, 84 EBR blocks
   - Open-source toolchain (Project Trellis)
   - Good for hobbyist projects

---

## Physical Connectors

### Cartridge Connector

**Connector Type**: 50-pin Edge Connector or Pin Header (2×25, 0.1" pitch)  
**Recommended**: 50-pin connector (includes expansion port)

#### Pin Allocation

| Category | Pins | Signals |
|----------|------|---------|
| **Address Bus** | 24 | A[23:0] - Bank number (8 bits) + Offset (16 bits) |
| **Data Bus** | 8 | D[7:0] - 8-bit data bus |
| **Control Signals** | 5 | /CE, /OE, /RD, CLK, /RST |
| **Expansion Port** | 8 | EXT_D[3:0], EXT_/EN, EXT_/RD, EXT_/WR |
| **Power/Ground** | 5 | VCC×2, GND×3 |
| **TOTAL** | **50** | |

#### Pin Layout (50-pin, 2×25)

```
Pin Layout (viewed from cartridge side):

    1   2   3   4   5   6   7   8   9  10  11  12  13  14  15  16  17  18  19  20  21  22  23  24  25
    ┌───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┐
    │GND│VCC│A23│A22│A21│A20│A19│A18│A17│A16│A15│A14│A13│A12│A11│A10│ A9│ A8│ A7│ A6│ A5│ A4│ A3│ A2│ A1│
    ├───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┤
    │ A0│ D7│ D6│ D5│ D4│ D3│ D2│ D1│ D0│/CE│/OE│/RD│CLK│/RST│GND│EXT│EXT│EXT│EXT│EXT│EXT│EXT│EXT│GND│VCC│
    │   │   │   │   │   │   │   │   │   │   │   │   │   │    │   │D0 │D1 │D2 │D3 │/EN│/RD│/WR│   │   │   │
    └───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┘
    26  27  28  29  30  31  32  33  34  35  36  37  38  39  40  41  42  43  44  45  46  47  48  49  50

Cartridge Interface: Pins 1-40 (40 pins)
Expansion Port: Pins 41-48 (8 pins)
Power/Ground: Pins 1, 2, 39, 49, 50 (5 pins)
```

#### Signal Descriptions

| Signal | Type | Description |
|--------|------|-------------|
| A[23:0] | Output | 24-bit address bus (8-bit bank + 16-bit offset) |
| D[7:0] | Bidirectional | 8-bit data bus (ROM is read-only) |
| /CE | Output | Chip Enable (active low) - Enables cartridge when bank 1-125 accessed |
| /OE | Output | Output Enable (active low) - Enables data output from cartridge |
| /RD | Output | Read Strobe (active low) - Indicates read operation |
| CLK | Output | System Clock (10 MHz) - Optional, for synchronous ROM |
| /RST | Output | Reset (active low) - Optional, for cartridge initialization |
| VCC | Power | Power supply (3.3V or 5V, depending on FPGA I/O standard) |
| GND | Ground | Ground reference (multiple pins for signal integrity) |

#### ROM Mapping (LoROM)

- **ROM Banks**: 1-125 (0x01-0x7D)
- **ROM Window**: 0x8000-0xFFFF (32KB per bank)
- **Total ROM Space**: 7.8MB (125 banks × 32KB)
- **Address Translation**: `romOffset = (bank-1) × 32768 + (offset - 0x8000)`

#### ROM Chip Recommendations

1. **Parallel NOR Flash** (most common)
   - Examples: SST39VF series, AT29C series
   - Speed: 70-150ns access time
   - Capacity: Up to 8MB easily available

2. **Parallel EEPROM**
   - Examples: AT28C series
   - Speed: 150-200ns access time
   - Capacity: Up to 1MB common

---

### Controller Connectors

**Connector Type**: DB-9 (DE-9) Female on Console  
**Total Pins**: 9 pins (6 active, 3 reserved)  
**Interface**: Serial Shift Register (SNES-style)  
**Controllers**: 2 controllers supported

#### DB-9 Pinout

```
DB-9 Connector (Female on Console, Male on Controller Cable):

    5  4  3  2  1
     \  \  \  \  \
      \  \  \  \  \
       \  \  \  \  \
        \  \  \  \  \
         \  \  \  \  \
          └──┴──┴──┴──┘
             9  8  7  6

Pin Layout (viewed from front of connector):

Pin 1: +5V (Power)           [Red wire recommended]
Pin 2: GND (Ground)          [Black wire recommended]
Pin 3: DATA (Serial Data In) [Data from controller]
Pin 4: LATCH (Latch Signal)  [Output from console]
Pin 5: CLK (Clock Signal)    [Output from console]
Pin 6: GND (Ground)           [Additional ground for signal integrity]
Pin 7: Reserved               [Future use: Analog stick X]
Pin 8: Reserved               [Future use: Analog stick Y]
Pin 9: Reserved               [Future use: Rumble/Additional features]
```

#### Signal Descriptions

| Pin | Signal | Type | Description |
|-----|--------|------|-------------|
| 1 | +5V | Power | 5V power supply (100mA for shift register, 500mA for ESP32) |
| 2 | GND | Ground | Ground reference |
| 3 | DATA | Input | Serial data input from controller (active high) |
| 4 | LATCH | Output | Latch signal (active high, pulses to capture button state) |
| 5 | CLK | Output | Clock signal (active high, shifts data on rising edge) |
| 6 | GND | Ground | Additional ground for signal integrity |
| 7-9 | Reserved | - | Reserved for future expansion |

#### Button Mapping

| Button | Bit Position | Description |
|--------|--------------|-------------|
| UP | 0 | D-pad up |
| DOWN | 1 | D-pad down |
| LEFT | 2 | D-pad left |
| RIGHT | 3 | D-pad right |
| A | 4 | Primary action button |
| B | 5 | Secondary action button |
| X | 6 | Tertiary action button |
| Y | 7 | Quaternary action button |
| L | 8 | Left shoulder button |
| R | 9 | Right shoulder button |
| START | 10 | Start/pause button |
| Z | 11 | Additional shoulder button |

**Total: 12 buttons = 12 bits (fits in 16-bit word)**

#### Controller Hardware Design

**Recommended: ESP32-Based Controller**

**Why ESP32?**
- ✅ Very cheap (~$3-5 per module)
- ✅ Easy to program (Arduino IDE)
- ✅ Built-in WiFi/Bluetooth (wireless support later!)
- ✅ Built-in ADC (analog sticks in future)
- ✅ Software debouncing (no hardware needed)
- ✅ More flexible than shift register

**ESP32 Pin Assignment:**
- GPIO 0-11: Button inputs (12 buttons)
- GPIO 12: DATA output (Pin 3)
- GPIO 13: LATCH input (Pin 4)
- GPIO 14: CLK input (Pin 5)
- GPIO 15: Status LED (optional)
- GPIO 16: Rumble PWM (optional)

**Power Requirements:**
- **Option A (Recommended)**: Console provides 3.3V @ 500mA per controller
- **Option B**: Console provides 5V @ 300mA per controller (with on-board regulator)

**Alternative: Shift Register Design**
- Use 74HC165 shift register chips (~$1 total)
- Very low power (~10-30mA)
- Simple, reliable, no programming needed

#### Communication Protocol

**Serial Shift Register Interface (SNES-style):**

1. **Latch Phase**: Console pulses LATCH high to capture button states
2. **Shift Phase**: Console pulses CLK 16 times to shift data out serially
3. **Data Phase**: Controller outputs button states on DATA line, one bit per clock

**Timing Specifications:**
- tLATCH_HIGH: 12-100μs (latch pulse width)
- tCLK_HIGH: 6-50μs (clock high time)
- tCLK_LOW: 6-50μs (clock low time)
- **Recommended Clock Frequency**: 100 kHz (10μs period)

---

### Expansion Port

**Location**: Pins 41-48 of cartridge connector (8 pins)  
**Purpose**: General-purpose I/O interface for external devices

#### Expansion Port Pinout

| Pin | Signal | Type | Description |
|-----|--------|------|-------------|
| EXT_D[3:0] | 4 | Bidirectional | 4-bit data bus |
| EXT_/EN | 1 | Input | Enable signal (active low) |
| EXT_/RD | 1 | Input | Read strobe (active low) |
| EXT_/WR | 1 | Input | Write strobe (active low) |

#### Use Cases

1. **Memory Expansion** - Additional SRAM for save games
2. **Co-Processor** - Math co-processor, graphics accelerator
3. **Communication** - Serial (UART), network interface, wireless module
4. **Debug Interface** - JTAG connection, logic analyzer interface

#### Address Mapping

**Recommended Mapping:**
- **I/O Space**: 0xB000-0xBFFF (bank 0, offset 0xB000+)
- **Memory Space**: Banks 0x80-0xFF (reserved area)

---

## System Architecture

### CPU Architecture

#### Register Set

- **8 General Purpose Registers**: R0-R7 (all 16-bit)
- **Program Counter (PC)**: 24-bit (8-bit bank + 16-bit offset)
- **Stack Pointer (SP)**: 16-bit (points to WRAM in bank 0)
- **Flags Register**: 6 flags (Z, N, C, V, I, D)

#### Instruction Format

All instructions are 16-bit words:
```
[15:12] = Opcode family (0x0-0xF)
[11:8]  = Mode/subop
[7:4]   = Register 1 (destination)
[3:0]   = Register 2 (source)
```

#### Instruction Set Summary

| Opcode | Instruction | Description |
|--------|-------------|-------------|
| 0x1000 | MOV | Move/Load/Store (7 modes) |
| 0x2000 | ADD | Add |
| 0x3000 | SUB | Subtract |
| 0x4000 | MUL | Multiply |
| 0x5000 | DIV | Divide |
| 0x6000 | AND | Bitwise AND |
| 0x7000 | OR | Bitwise OR |
| 0x8000 | XOR | Bitwise XOR |
| 0x9000 | NOT | Bitwise NOT |
| 0xA000 | SHL | Shift Left |
| 0xB000 | SHR | Shift Right |
| 0xC000 | CMP | Compare |
| 0xC100 | BEQ | Branch if Equal |
| 0xC200 | BNE | Branch if Not Equal |
| 0xC300 | BGT | Branch if Greater Than |
| 0xC400 | BLT | Branch if Less Than |
| 0xC500 | BGE | Branch if Greater or Equal |
| 0xC600 | BLE | Branch if Less or Equal |
| 0xD000 | JMP | Jump (absolute) |
| 0xE000 | CALL | Call subroutine |
| 0xF000 | RET | Return from subroutine |

#### Flags

- **Z (Zero)**: Set when result is zero
- **N (Negative)**: Set when result is negative (bit 15 set)
- **C (Carry)**: Set on unsigned overflow/underflow
- **V (Overflow)**: Set on signed overflow
- **I (Interrupt Enable)**: When clear, interrupts are disabled
- **D (Division by Zero)**: Set when division by zero occurs

#### Interrupt System

- **Interrupt Vector Table**: Bank 0, addresses 0xFFE0-0xFFE3
  - 0xFFE0-0xFFE1: IRQ handler (16-bit offset)
  - 0xFFE2-0xFFE3: NMI handler (16-bit offset)
- **VBlank Interrupt**: IRQ triggered at start of each frame
- **Interrupt State Saving**: PC and flags pushed to stack automatically

---

### Memory Map

#### Banked Memory Architecture

- **Total Address Space**: 16MB (256 banks × 64KB)
- **Bank Size**: 64KB (16-bit addressing)
- **Bank Number**: 8-bit (0-255)
- **Full Address**: 24-bit (bank:offset)

#### Memory Layout

| Bank Range | Type | Size | Description |
|------------|------|------|-------------|
| 0x00 | WRAM + I/O | 64KB | 32KB WRAM + 32KB I/O registers |
| 0x01-0x7D | ROM | 7.8MB | ROM space (LoROM mapping) |
| 0x7E-0x7F | Extended WRAM | 128KB | Additional RAM |
| 0x80-0xFF | Reserved | - | Reserved for future use |

#### WRAM (Bank 0, 0x0000-0x7FFF)

- **Size**: 32KB
- **Access**: Read/Write, 16-bit
- **Stack**: Grows downward from 0x7FFF
- **Usage**: General purpose RAM, stack, variables

#### I/O Registers (Bank 0, 0x8000-0x8FFF)

- **PPU Registers**: 0x8000-0x8FFF
- **APU Registers**: 0x9000-0x9FFF
- **Input Registers**: 0xA000-0xAFFF
- **System Registers**: 0x8000-0x80FF
- **Expansion Port**: 0xB000-0xBFFF (optional)

---

### PPU (Graphics) Specification

#### VRAM (Video RAM)

- **Size**: 64KB
- **Organization**: Tiles, tilemaps, sprite data
- **Access**: 8-bit via VRAM_DATA register
- **Addressing**: 16-bit address via VRAM_ADDR_L/H

#### CGRAM (Color Graphics RAM)

- **Size**: 512 bytes (256 colors × 2 bytes)
- **Format**: RGB555 (5 bits per channel)
- **Organization**: 16 palettes × 16 colors each
- **Access**: 16-bit via CGRAM_DATA (low byte, then high byte)

#### OAM (Object Attribute Memory)

- **Size**: 768 bytes (128 sprites × 6 bytes)
- **Access**: 8-bit via OAM_DATA register
- **Write Protection**: Writes blocked during visible rendering (scanlines 0-199)

#### Background Layers

- **4 Independent Layers**: BG0, BG1, BG2, BG3
- **Tile Sizes**: 8×8 or 16×16 pixels (configurable per layer)
- **Priority**: BG3 > BG2 > BG1 > BG0 (higher number = higher priority)
- **Scroll**: Per-layer X/Y scroll registers

#### Sprites

- **Max Sprites**: 128
- **Sizes**: 8×8 or 16×16 pixels
- **Priority**: 4 levels (0-3, from sprite attributes)
- **Transparency**: Color index 0 is transparent
- **Flip**: Horizontal and/or vertical flip

#### Matrix Mode

- **Mode 7-style Effects**: Per-layer affine transformations
- **Transformations**: Rotation, scaling, perspective
- **Large World Support**: Extended tilemap coordinates
- **HDMA Updates**: Per-scanline matrix updates

#### Rendering Pipeline

1. **Background Rendering**: BG3 → BG2 → BG1 → BG0
2. **Sprite Rendering**: Sorted by priority, then by index
3. **Blending**: Alpha blending, additive, subtractive modes
4. **Output**: 320×200 pixel frame buffer

---

### APU (Audio) Specification

#### Audio Channels

- **4 Independent Channels**: Each with frequency, volume, waveform, duration
- **Waveforms**: Sine, Square, Saw, Noise
- **Sample Rate**: 44,100 Hz
- **Output**: Stereo (left/right)

#### Channel Parameters

- **Frequency**: 16-bit (0-65535, controls pitch)
- **Volume**: 4-bit (0-15, per-channel)
- **Master Volume**: 4-bit (0-15, global)
- **Duration**: 16-bit (frames, 0 = infinite)
- **Loop**: Boolean (repeat when duration expires)

#### Audio Generation

- **Samples per Frame**: 735 (44,100 Hz / 60 FPS)
- **Generation**: Continuous during frame execution
- **Completion**: Flags set when duration expires (if not looping)

---

### Input System Specification

#### Controllers

- **2 Controllers**: Controller 1 and Controller 2
- **12 Buttons per Controller**: UP, DOWN, LEFT, RIGHT, A, B, X, Y, L, R, START, Z
- **Latch Mechanism**: Write 1 to INPUT_LATCH to capture state, 0 to release

#### Button Reading

1. Write 1 to INPUT_LATCH (captures current button state)
2. Read INPUT_DATA_L/H to get button states
3. Write 0 to INPUT_LATCH (releases latch)

**Button States**: 1 = pressed, 0 = released

---

## I/O Register Map

### System Registers (0x8000-0x80FF)

| Address | Name | Size | Description |
|---------|------|------|-------------|
| 0x803E | VBLANK_FLAG | 8-bit | VBlank flag (bit 0), cleared when read |
| 0x803F | FRAME_COUNTER_LOW | 8-bit | Frame counter low byte |
| 0x8040 | FRAME_COUNTER_HIGH | 8-bit | Frame counter high byte |

### PPU Registers (0x8000-0x8FFF)

#### Background Control Registers

| Address | Name | Size | Description |
|---------|------|------|-------------|
| 0x8008 | BG0_CONTROL | 8-bit | BG0 enable (bit 0) |
| 0x8009 | BG1_CONTROL | 8-bit | BG1 enable (bit 0) |
| 0x800A | BG2_CONTROL | 8-bit | BG2 enable (bit 0) |
| 0x800B | BG3_CONTROL | 8-bit | BG3 enable (bit 0) |

#### VRAM Access Registers

| Address | Name | Size | Description |
|---------|------|------|-------------|
| 0x800E | VRAM_ADDR_L | 8-bit | VRAM address low byte |
| 0x800F | VRAM_ADDR_H | 8-bit | VRAM address high byte |
| 0x8010 | VRAM_DATA | 8-bit | VRAM data (auto-increments address) |

#### CGRAM Access Registers

| Address | Name | Size | Description |
|---------|------|------|-------------|
| 0x8012 | CGRAM_ADDR | 8-bit | CGRAM address (palette + color index) |
| 0x8013 | CGRAM_DATA | 16-bit | CGRAM data (RGB555, low byte then high byte) |

#### OAM Access Registers

| Address | Name | Size | Description |
|---------|------|------|-------------|
| 0x8014 | OAM_ADDR | 8-bit | OAM address (sprite ID, 0-127) |
| 0x8015 | OAM_DATA | 8-bit | OAM data (auto-increments byte index) |

**OAM Sprite Format (6 bytes per sprite):**
- Byte 0: X position low byte
- Byte 1: X position high byte (bit 0 = sign bit)
- Byte 2: Y position
- Byte 3: Tile index
- Byte 4: Attributes (bits [7:6] = priority, bits [4:0] = palette)
- Byte 5: Control (bit 0 = enable, bit 1 = 16x16 size)

#### DMA Registers

| Address | Name | Size | Description |
|---------|------|------|-------------|
| 0x8060 | DMA_CONTROL | 8-bit | DMA enable (bit 0), mode, destination type |
| 0x8061 | DMA_SOURCE_BANK | 8-bit | Source bank (1-125 for ROM) |
| 0x8062 | DMA_SOURCE_OFFSET_L | 8-bit | Source offset low byte |
| 0x8063 | DMA_SOURCE_OFFSET_H | 8-bit | Source offset high byte |
| 0x8064 | DMA_DEST_ADDR_L | 8-bit | Destination address low byte |
| 0x8065 | DMA_DEST_ADDR_H | 8-bit | Destination address high byte |
| 0x8066 | DMA_LENGTH_L | 8-bit | Transfer length low byte |
| 0x8067 | DMA_LENGTH_H | 8-bit | Transfer length high byte |

### APU Registers (0x9000-0x9FFF)

#### Channel Registers (per channel, offset 0x10 per channel)

| Offset | Name | Size | Description |
|--------|------|------|-------------|
| +0x00 | CHN_FREQ_L | 8-bit | Frequency low byte |
| +0x01 | CHN_FREQ_H | 8-bit | Frequency high byte |
| +0x02 | CHN_VOLUME | 8-bit | Volume (0-15) |
| +0x03 | CHN_WAVEFORM | 8-bit | Waveform type (0=sine, 1=square, 2=saw, 3=noise) |
| +0x04 | CHN_DURATION_L | 8-bit | Duration low byte |
| +0x05 | CHN_DURATION_H | 8-bit | Duration high byte |
| +0x06 | CHN_CONTROL | 8-bit | Enable (bit 0), loop (bit 1) |

**Channel Base Addresses:**
- Channel 0: 0x9000
- Channel 1: 0x9010
- Channel 2: 0x9020
- Channel 3: 0x9030

#### Master Control Registers

| Address | Name | Size | Description |
|---------|------|------|-------------|
| 0x9040 | MASTER_VOLUME | 8-bit | Master volume (0-15) |
| 0x9021 | COMPLETION_STATUS | 8-bit | Channel completion flags (bits [3:0], cleared when read) |

### Input Registers (0xA000-0xAFFF)

| Address | Name | Size | Description |
|---------|------|------|-------------|
| 0xA000 | INPUT_DATA_L | 8-bit | Controller 1 buttons (low byte) |
| 0xA001 | INPUT_DATA_H | 8-bit | Controller 1 buttons (high byte) |
| 0xA002 | INPUT_DATA2_L | 8-bit | Controller 2 buttons (low byte) |
| 0xA003 | INPUT_DATA2_H | 8-bit | Controller 2 buttons (high byte) |
| 0xA004 | INPUT_LATCH | 8-bit | Latch control (write 1 to latch, 0 to release) |

**Button Mapping (Controller 1):**
- Low byte: UP (bit 0), DOWN (bit 1), LEFT (bit 2), RIGHT (bit 3), A (bit 4), B (bit 5), X (bit 6), Y (bit 7)
- High byte: L (bit 0), R (bit 1), START (bit 2), Z (bit 3)

---

## Timing and Synchronization

### Frame Timing

- **Frame Rate**: 60 FPS
- **Frame Duration**: 16.67 ms
- **CPU Cycles per Frame**: 166,667 (at 10 MHz)

### Frame Execution Order

1. **APU Update** (start of frame)
   - Decrement channel durations
   - Set completion flags
   - Clear completion status (if not already cleared)

2. **PPU Rendering** (start of frame)
   - Set VBlank flag = 1
   - Increment frame counter
   - Render frame using state from previous frame

3. **CPU Execution** (during frame)
   - CPU runs for 166,667 cycles
   - Can read VBlank flag (will see 1, then cleared)
   - Can read frame counter
   - Can read completion status

4. **Audio Generation** (continuous)
   - Generate 735 samples per frame
   - Independent of frame timing

### VBlank Flag (0x803E)

- **Set**: At start of frame (scanline 200)
- **Cleared**: When read (one-shot latch)
- **Behavior**: Hardware-accurate, matches NES/SNES pattern

**FPGA Implementation:**
```verilog
always @(posedge clk) begin
    if (frame_start) begin
        vblank_flag <= 1'b1;
    end else if (read_vblank) begin
        vblank_flag <= 1'b0;  // Clear when read
    end
end
```

### Frame Counter (0x803F/0x8040)

- **16-bit Counter**: Increments once per frame
- **Low Byte**: 0x803F
- **High Byte**: 0x8040

**FPGA Implementation:**
```verilog
reg [15:0] frame_counter;
always @(posedge clk) begin
    if (frame_start) begin
        frame_counter <= frame_counter + 1;
    end
end
```

---

## ROM Format

### ROM Header (32 bytes)

| Offset | Size | Name | Description |
|--------|------|------|-------------|
| 0x00 | 4 | Magic | "RMCF" (0x46434D52) |
| 0x04 | 2 | Version | ROM format version (1) |
| 0x06 | 4 | ROM Size | Total ROM size in bytes |
| 0x0A | 2 | Entry Bank | Entry point bank (1-125) |
| 0x0C | 2 | Entry Offset | Entry point offset (0x8000+) |
| 0x0E | 2 | Mapper Flags | Mapper type (0 = LoROM) |
| 0x10 | 4 | Checksum | ROM checksum (currently 0) |
| 0x14 | 12 | Reserved | Reserved for future use |

### ROM Data

- **Code Section**: Variable size, little-endian 16-bit words
- **Asset Section**: Optional, appended after code
- **Total Size**: Up to 7.8MB (125 banks × 64KB)

---

## FPGA Implementation Guidelines

### Clock Domains

- **CPU Clock**: 10 MHz (main system clock)
- **Audio Clock**: 44.1 kHz (audio sample generation)
- **Video Clock**: 60 Hz (frame timing, VBlank)

### Synchronization

- **VBlank Signal**: Synchronized to video clock
- **Frame Counter**: Synchronized to video clock
- **Completion Status**: Synchronized to CPU clock
- **Cross-Clock Domain**: Use proper synchronization (FIFOs, handshaking)

### State Machines

- **APU**: Simple state machine for channel control
- **PPU**: Rendering pipeline with clear stages
- **CPU**: Instruction fetch/decode/execute pipeline

### Register Access

- **I/O Registers**: Byte-addressable, simple address decoding
- **Address Decoding**: 0x8000-0x8FFF = PPU, 0x9000-0x9FFF = APU, 0xA000-0xAFFF = Input
- **Read/Write**: Standard memory-mapped I/O

### Memory Interfaces

- **WRAM**: Dual-port RAM (CPU read/write, PPU read)
- **VRAM**: Dual-port RAM (CPU write, PPU read)
- **CGRAM**: Dual-port RAM (CPU write, PPU read)
- **OAM**: Dual-port RAM (CPU write, PPU read)

### Testing Strategy

1. **Test ROMs**: Use same test ROMs as emulator
2. **Signal Verification**: Verify VBlank, frame counter, completion status match emulator
3. **Timing Verification**: Ensure frame boundaries align correctly
4. **Audio Verification**: Verify audio samples match emulator output

---

## Summary

### Key Specifications

- **FPGA**: Xilinx Spartan-7 XC7S50 (recommended)
- **Cartridge Connector**: 50-pin (40 pins cartridge + 8 pins expansion)
- **Controller Connectors**: 2× DB-9 (6 active pins each, 3 reserved)
- **Controller Design**: ESP32-based (recommended) or shift register
- **Power Consumption**: 0.8-2.3W (single FPGA)
- **Production Cost**: $30-50 per console (volume pricing)

### Physical Layout

```
Console Physical Layout:

┌─────────────────────────────────────┐
│         Nitro-Core-DX Console       │
│                                     │
│  [Cartridge Slot] (50-pin)          │
│                                     │
│  [Controller 1] [Controller 2]     │
│     (DB-9)        (DB-9)            │
│                                     │
│  [Expansion Port] (8 pins)          │
│                                     │
│  [Power Connector]                  │
│  [Video Output]                     │
│  [Audio Output]                     │
└─────────────────────────────────────┘
```

---

## References

- **Programming Manual**: `NITRO_CORE_DX_PROGRAMMING_MANUAL.md` - Complete instruction set and programming guide
- **System Manual**: `SYSTEM_MANUAL.md` - System architecture and design details
- **FPGA Compatibility**: `docs/archive/FPGA_COMPATIBILITY.md` - FPGA-specific notes
- **Hardware Status**: `HARDWARE_FEATURES_STATUS.md` - Implementation status

---

## Version History

- **v2.0** (2026-01-30): Complete hardware specification merged from all component specs
- **v1.0** (2026-01-30): Initial hardware specification document

---

**End of Complete Hardware Specification**
