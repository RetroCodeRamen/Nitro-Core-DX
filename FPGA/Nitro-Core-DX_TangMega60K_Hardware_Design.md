# Nitro-Core-DX Hardware Implementation on Tang Mega 60K

## Comprehensive Hardware Design Document

**Version:** 1.1  
**Date:** June 2026 (CPU ISA, I/O register map, and audio sections reconciled against the current emulator implementation — the authoritative hardware contract)  
**Target Platform:** Sipeed Tang Mega 60K (GW5AT-LV60PG484A)

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [System Overview](#system-overview)
3. [FPGA Resource Analysis](#fpga-resource-analysis)
4. [Memory Architecture](#memory-architecture)
5. [Video Subsystem](#video-subsystem)
6. [Audio Subsystem](#audio-subsystem)
7. [Input/Controller Interface](#inputcontroller-interface)
8. [Pin Assignment](#pin-assignment)
9. [Clocking Strategy](#clocking-strategy)
10. [Power Requirements](#power-requirements)
11. [Bill of Materials](#bill-of-materials)
12. [Implementation Guidelines](#implementation-guidelines)

---

## Executive Summary

This document provides a complete hardware design for implementing the **Nitro-Core-DX** fantasy console on the **Sipeed Tang Mega 60K** FPGA development board. The design leverages the GW5AT-LV60PG484A FPGA's abundant resources (59,904 LUTs, 118 DSPs, 2,124Kb BSRAM) to create a fully functional 16-bit retro gaming system.

### Key Features Implemented

| Feature | Specification |
|---------|--------------|
| **Resolution** | 320×200 pixels @ 60 FPS |
| **Color Depth** | 256 colors (8-bit indexed), 32,768 color palette (RGB555) |
| **Background Layers** | 4 independent scrolling layers |
| **Sprites** | 128 sprites with priority and blending |
| **Matrix Mode** | Mode 7-style perspective/rotation effects |
| **Audio** | 4 channels, 44.1kHz sample rate |
| **CPU** | Custom 16-bit @ 10 MHz |
| **Memory** | 16MB banked address space (512MB DDR3) |
| **Input** | 2× game controllers (SNES/Genesis compatible) |

---

## System Overview

### Architecture Block Diagram

The Nitro-Core-DX system consists of four main components implemented within the FPGA:

```
┌─────────────────────────────────────────────────────────────┐
│                    Nitro-Core-DX System                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐  │
│  │   CPU   │◄──►│  PPU    │◄──►│  APU    │◄──►│  MEM    │  │
│  │ 10 MHz  │    │4 Layers │    │ 4CH     │    │16MB     │  │
│  │16-bit   │    │128 Sprt │    │44.1kHz  │    │Banked   │  │
│  └────┬────┘    └────┬────┘    └────┬────┘    └────┬────┘  │
│       │              │              │              │       │
│       └──────────────┴──────────────┴──────────────┘       │
│                      │                                       │
│              ┌───────┴───────┐                              │
│              │  System Bus   │                              │
│              │  Controller   │                              │
│              └───────┬───────┘                              │
│                      │                                       │
│       ┌──────────────┼──────────────┐                      │
│       ▼              ▼              ▼                      │
│  ┌─────────┐   ┌─────────┐   ┌─────────┐                  │
│  │  HDMI   │   │  Audio  │   │  Input  │                  │
│  │ Output  │   │  Output │   │Controllers                  │
│  └─────────┘   └─────────┘   └─────────┘                  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## FPGA Resource Analysis

### Tang Mega 60K Specifications

| Resource | Available | Used by Nitro-Core-DX | Utilization |
|----------|-----------|----------------------|-------------|
| **LUT4** | 59,904 | ~35,000 | 58% |
| **Registers (FF)** | 59,904 | ~28,000 | 47% |
| **BSRAM (Kb)** | 2,124 | ~1,500 | 71% |
| **BSRAM Blocks** | 118 | ~80 | 68% |
| **DSP (27×18)** | 118 | ~40 | 34% |
| **PLLs** | 8 | 3 | 38% |

### Resource Allocation by Component

#### CPU Core (~5,000 LUTs)
- 16-bit ALU with multiplier (uses 4 DSPs)
- 8 general-purpose registers
- Banked memory addressing unit
- Interrupt controller
- Instruction decoder

**Authoritative instruction set** (opcode in bits [15:12]; mode in [11:8];
reg1 in [7:4]; reg2 in [3:0]). This is the encoding the emulator executes
(`internal/cpu/cpu.go` dispatch, `internal/cpu/instructions.go`) and is the
contract the FPGA core must match:

| Opcode | Instruction | Notes |
|--------|-------------|-------|
| 0x0 | NOP | |
| 0x1 | MOV | modes: 0 reg, 1 #imm, 2 [R] load16, 3 [R] store16, 4 PUSH, 5 POP, 6 [R] load8 (zero-ext), 7 [R] store8, 9 [R+imm] load, 10 [R+imm] store |
| 0x2 | ADD | mode 0 reg, 1 #imm; modes 2/3 = 8-bit (.B) |
| 0x3 | SUB | mode 0 reg, 1 #imm; modes 2/3 = 8-bit (.B); two's-complement wrap |
| 0x4 | MUL | 16×16 → low 16 bits |
| 0x5 | DIV | unsigned; div-by-zero → 0xFFFF and sets flag D |
| 0x6 | AND | mode 0 reg, 1 #imm |
| 0x7 | OR | mode 0 reg, 1 #imm |
| 0x8 | XOR | mode 0 reg, 1 #imm |
| 0x9 | NOT | |
| 0xA | SHL | mode 0 reg, 1 #imm |
| 0xB | SHR | modes 0/1 logical right; modes 2/3 SAR (arithmetic, sign-extending); mode 4 ROL, mode 5 ROR (through carry) |
| 0xC | CMP + branches | CMP mode 0 reg / mode 7 #imm; +0x100 BEQ, +0x200 BNE, +0x300 BGT, +0x400 BLT, +0x500 BGE, +0x600 BLE |
| 0xD | JMP | PC-relative |
| 0xE | CALL | |
| 0xF | RET | |

Flags: Z (zero), N (negative, bit 15), C (carry), D (division by zero).
SP initializes to 0x1FFF and grows downward; PC is 24-bit (8-bit bank +
16-bit offset).

> **⚠️ Verilog reconciliation required before bring-up.** The current
> `src/cpu/cpu_core.v` opcode map has drifted from this authoritative
> emulator encoding (it uses a unified shift opcode at 0x8 and places
> MUL/DIV at 0x9/0xA). The Verilog must be brought back to the table above
> so a ROM that runs on the emulator runs identically on hardware. Verify by
> executing the same ROMs through both and comparing machine state.

#### PPU - Picture Processing Unit (~15,000 LUTs)
- 4 background layer renderers
- Sprite engine (128 sprites)
- Matrix mode transformation unit (uses 16 DSPs)
- Color palette lookup (RGB555 → RGB888)
- Priority and blending logic
- HDMA (Horizontal DMA) controller

#### APU - Audio Processing Unit (~3,000 LUTs)
- 4 channel synthesizers
- Waveform generators (sine, square, saw, noise)
- Sample rate converter (uses 2 DSPs)
- Master volume control
- I2S output interface

#### Memory Controller (~8,000 LUTs)
- DDR3 interface controller
- Banked address translation
- DMA engine
- Cache controller (optional)

#### Video Output (~4,000 LUTs)
- HDMI TMDS encoder
- RGB LCD interface
- Scan doubler (320×200 → 640×400)

---

## Memory Architecture

### Address Space Mapping

The Nitro-Core-DX uses a 24-bit banked addressing scheme:

```
┌─────────────────────────────────────────────────────────┐
│                  16MB Address Space                     │
├─────────────────────────────────────────────────────────┤
│  Bank 0    │  0x000000 - 0x007FFF  │  WRAM (32KB)       │
│            │  0x008000 - 0x00FFFF  │  I/O Registers     │
├────────────┼───────────────────────┼────────────────────┤
│  Bank 1    │  0x010000 - 0x017FFF  │  ROM (32KB)        │
│  Bank 2    │  0x020000 - 0x027FFF  │  ROM (32KB)        │
│  ...       │  ...                  │  ...               │
│  Bank 125  │  0x7D0000 - 0x7D7FFF  │  ROM (32KB)        │
├────────────┼───────────────────────┼────────────────────┤
│  Bank 126  │  0x7E0000 - 0x7E7FFF  │  Extended WRAM     │
│  Bank 127  │  0x7F0000 - 0x7F7FFF  │  Extended WRAM     │
└────────────┴───────────────────────┴────────────────────┘
```

### DDR3 Memory Mapping

The 512MB DDR3 is organized as follows:

| Region | Size | Purpose |
|--------|------|---------|
| 0x00000000 - 0x00007FFF | 32KB | System WRAM (mirrored to Bank 0) |
| 0x00008000 - 0x001FFFFF | ~2MB | PPU VRAM (tilemaps, patterns, sprites) |
| 0x00200000 - 0x007FFFFF | 6MB | Game ROM storage |
| 0x00800000 - 0x1FFFFFFF | ~384MB | Extended storage (multiple games) |

### I/O Register Map

> **Authoritative source:** these addresses are taken from the current
> emulator implementation (`internal/memory/bus.go`, `internal/ppu/ppu.go`,
> `internal/apu/apu.go`, `internal/input/input.go`), which the FPGA core must
> match bit-for-bit. The full per-register table with code citations lives in
> `docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md`; the summary
> below is the high-level routing plus the highest-traffic registers.

**Top-level MMIO routing** (bank 0; `bus.go:131-146`):

| Range | Block | Notes |
|-------|-------|-------|
| 0x8000 - 0x8FFF | PPU | All video registers (single block, not sub-paged) |
| 0x9000 - 0x9FFF | APU | Audio registers |
| 0xA000 - 0xAFFF | Input | Controller registers |

I/O is **8-bit only**: 16-bit reads zero-extend an 8-bit read; 16-bit writes
write only the low byte.

**Key PPU registers (0x8000-0x8FFF):**

| Address | Register | Description |
|---------|----------|-------------|
| 0x8008 / 0x8009 / 0x8021 / 0x8026 | BGn_CONTROL | BG0/1/2/3 enable, tile size, priority, tilemap size |
| 0x8000-0x800D, 0x8022-0x8025 | BGn_SCROLL | Per-layer scroll X/Y (low/high) |
| 0x800E / 0x800F | VRAM_ADDR_L / _H | VRAM address |
| 0x8010 | VRAM_DATA | VRAM data (auto-increment) |
| 0x8012 / 0x8013 | CGRAM_ADDR / CGRAM_DATA | Palette index / RGB555 data (low byte then high byte) |
| 0x8014 / 0x8015 | OAM_ADDR / OAM_DATA | Sprite ID / sprite data (auto-increment) |
| 0x8018 | MATRIX_CONTROL | Affine enable, mirror H/V, outside mode, direct color |
| 0x806C-0x806F | BGn_TRANSFORM_BIND | Bind a layer to a transform channel (0-3) |
| 0x8070-0x8076 | TEXT port | X(16-bit), Y, R, G, B, char (debug/HUD text) |
| 0x8080-0x80A9 | MATRIX_PLANE | Dedicated matrix-plane select/control, tilemap/pattern/bitmap data, projection (mode, horizon, camera, heading, focal length, scale) |
| 0x803E | VBLANK_FLAG | VBlank flag (bit 0), cleared on read |
| 0x803F | FRAME_COUNTER_LOW | Frame counter low byte |

**APU registers (0x9000-0x9FFF):** per-channel control/frequency/volume,
`0x9020` MASTER_VOLUME, `0x9021` COMPLETION_STATUS. V1 audio target is a
YM2608/OPNA FM path (see Audio Subsystem) layered on this MMIO shell.

**Input registers (0xA000-0xAFFF):** `0xA000`/`0xA001` controller 1 (low/high;
`0xA001` also drives latch control), `0xA002`/`0xA003` controller 2.
Latch-then-read serial model matching real controller hardware
(`input.go:7-52`).

---

## Video Subsystem

### Display Specifications

| Parameter | Value |
|-----------|-------|
| Native Resolution | 320×200 pixels |
| Output Resolution | 1280×720 (720p) via HDMI |
| Refresh Rate | 60 Hz |
| Color Depth | 8-bit indexed (256 colors) |
| Palette | 32,768 colors (RGB555) |

### Background Layers

Each of the 4 background layers supports:

| Feature | Specification |
|---------|--------------|
| Tile Size | 8×8 or 16×16 pixels (configurable) |
| Tilemap Size | Up to 64×64 tiles (4096×4096 pixels) |
| Colors per layer | 16 or 256 (configurable) |
| Scrolling | Horizontal and vertical per-layer |
| Priority | 4 priority levels |

### Sprite Engine

| Feature | Specification |
|---------|--------------|
| Max Sprites | 128 |
| Sprite Size | 8×8 to 64×64 pixels |
| Colors per sprite | 16 or 256 |
| Attributes | X, Y, Pattern, Palette, Priority, Flip H/V |

### Matrix Mode (Mode 7-style)

Transformation matrix for perspective/rotation effects:

```
[ X']   [ A  B  C ]   [ X ]
[ Y'] = [ D  E  F ] × [ Y ]
[ 1 ]   [ 0  0  1 ]   [ 1 ]
```

- Supports full affine transformations
- Large world support (up to 4096×4096)
- Vertical sprite mode for 3D effects

### HDMI Output

The Tang Mega 60K's HDMI interface uses TMDS signaling:

| Signal | FPGA Pin | Description |
|--------|----------|-------------|
| HDMI_CLK_P | Bank 5 | TMDS Clock+ |
| HDMI_CLK_N | Bank 5 | TMDS Clock- |
| HDMI_D0_P | Bank 5 | TMDS Data 0+ (Blue) |
| HDMI_D0_N | Bank 5 | TMDS Data 0- (Blue) |
| HDMI_D1_P | Bank 5 | TMDS Data 1+ (Green) |
| HDMI_D1_N | Bank 5 | TMDS Data 1- (Green) |
| HDMI_D2_P | Bank 5 | TMDS Data 2+ (Red) |
| HDMI_D2_N | Bank 5 | TMDS Data 2- (Red) |

**Pixel Clock:** 74.25 MHz (720p60)

### RGB LCD Interface (Alternative)

| Signal | Count | Description |
|--------|-------|-------------|
| LCD_R[7:0] | 8 | Red data |
| LCD_G[7:0] | 8 | Green data |
| LCD_B[7:0] | 8 | Blue data |
| LCD_HSYNC | 1 | Horizontal sync |
| LCD_VSYNC | 1 | Vertical sync |
| LCD_PCLK | 1 | Pixel clock |
| LCD_DE | 1 | Data enable |

---

## Audio Subsystem

### Audio Specifications

> **V1 audio target is YM2608/OPNA FM** (`docs/specifications/APU_FM_OPM_EXTENSION_SPEC.md`).
> The legacy PSG-style channel path is preserved underneath and runtime FM is
> driven through a YM2608/OPNA backend layered on the same MMIO shell
> (`0x9000-0x9FFF`). The FPGA FM implementation must match the emulator's
> backend behavioral profile.

| Parameter | Value |
|-----------|-------|
| FM channels | YM2608/OPNA FM (4-operator) |
| Legacy channels | PSG-style waveform channels (preserved compatibility path) |
| Sample Rate | 44,100 Hz |
| Bit Depth | 16-bit |
| Output | Stereo (2× 3W speakers + 3.5mm headphone) |

### Channel Features

| Feature | Range |
|---------|-------|
| Master volume | 0x9020, 0 - 255 (8-bit) |
| Completion status | 0x9021, per-channel flags (cleared on read) |
| Frequency / volume | Per-channel registers in the 0x9000-0x9FFF block |

### I2S Audio Interface

| Signal | FPGA Pin | Description |
|--------|----------|-------------|
| I2S_BCLK | Bank 7 | Bit clock (2.8224 MHz) |
| I2S_LRCLK | Bank 7 | Left/Right clock (44.1 kHz) |
| I2S_DATA | Bank 7 | Serial audio data |
| I2S_MCLK | Bank 7 | Master clock (11.2896 MHz) |

---

## Input/Controller Interface

### Controller Support

The system supports 2 game controllers via PMOD connectors:

| Controller Type | Interface | Pins Required |
|-----------------|-----------|---------------|
| SNES Controller | Parallel (PMOD) | 6 pins |
| Genesis Controller | Parallel (PMOD) | 9 pins |
| USB HID Gamepad | USB 2.0 | 2 pins (D+/D-) |

### SNES Controller Pinout (PMOD A)

| PMOD Pin | Signal | Description |
|----------|--------|-------------|
| 1 | VCC | +3.3V |
| 2 | GND | Ground |
| 3 | DATA1 | Controller 1 data |
| 4 | DATA2 | Controller 2 data |
| 5 | LATCH | Latch signal |
| 6 | CLK | Clock signal |
| 7 | NC | Not connected |
| 8 | NC | Not connected |

### Button Mapping

| Bit | SNES Button | Genesis Button |
|-----|-------------|----------------|
| 0 | B | A |
| 1 | Y | B |
| 2 | SELECT | - |
| 3 | START | START |
| 4 | UP | UP |
| 5 | DOWN | DOWN |
| 6 | LEFT | LEFT |
| 7 | RIGHT | RIGHT |
| 8 | A | C |
| 9 | X | X |
| 10 | L | Y |
| 11 | R | Z |

---

## Pin Assignment

### Complete Pinout Table

#### DDR3 Memory Interface (Bank 2-4)

| Signal | FPGA Bank | Pin Count | Voltage |
|--------|-----------|-----------|---------|
| DDR3_DQ[15:0] | Bank 2 | 16 | 1.5V |
| DDR3_DM[1:0] | Bank 2 | 2 | 1.5V |
| DDR3_DQS_P[1:0] | Bank 2 | 2 | 1.5V |
| DDR3_DQS_N[1:0] | Bank 2 | 2 | 1.5V |
| DDR3_ADDR[13:0] | Bank 3 | 14 | 1.5V |
| DDR3_BA[2:0] | Bank 3 | 3 | 1.5V |
| DDR3_RAS_N | Bank 4 | 1 | 1.5V |
| DDR3_CAS_N | Bank 4 | 1 | 1.5V |
| DDR3_WE_N | Bank 4 | 1 | 1.5V |
| DDR3_CK_P | Bank 4 | 1 | 1.5V |
| DDR3_CK_N | Bank 4 | 1 | 1.5V |
| DDR3_CKE | Bank 4 | 1 | 1.5V |
| DDR3_ODT | Bank 4 | 1 | 1.5V |
| DDR3_RESET_N | Bank 4 | 1 | 1.5V |
| **Subtotal** | | **47** | |

#### HDMI Interface (Bank 5)

| Signal | FPGA Bank | Pin Count | Voltage |
|--------|-----------|-----------|---------|
| HDMI_CLK_P | Bank 5 | 1 | 3.3V |
| HDMI_CLK_N | Bank 5 | 1 | 3.3V |
| HDMI_D0_P | Bank 5 | 1 | 3.3V |
| HDMI_D0_N | Bank 5 | 1 | 3.3V |
| HDMI_D1_P | Bank 5 | 1 | 3.3V |
| HDMI_D1_N | Bank 5 | 1 | 3.3V |
| HDMI_D2_P | Bank 5 | 1 | 3.3V |
| HDMI_D2_N | Bank 5 | 1 | 3.3V |
| HDMI_HPD | Bank 5 | 1 | 3.3V |
| HDMI_SCL | Bank 5 | 1 | 3.3V |
| HDMI_SDA | Bank 5 | 1 | 3.3V |
| **Subtotal** | | **11** | |

#### RGB LCD Interface (Bank 6)

| Signal | FPGA Bank | Pin Count | Voltage |
|--------|-----------|-----------|---------|
| LCD_R[7:0] | Bank 6 | 8 | 3.3V |
| LCD_G[7:0] | Bank 6 | 8 | 3.3V |
| LCD_B[7:0] | Bank 6 | 8 | 3.3V |
| LCD_PCLK | Bank 6 | 1 | 3.3V |
| LCD_HSYNC | Bank 6 | 1 | 3.3V |
| LCD_VSYNC | Bank 6 | 1 | 3.3V |
| LCD_DE | Bank 6 | 1 | 3.3V |
| **Subtotal** | | **28** | |

#### Audio Interface (Bank 7)

| Signal | FPGA Bank | Pin Count | Voltage |
|--------|-----------|-----------|---------|
| I2S_BCLK | Bank 7 | 1 | 3.3V |
| I2S_LRCLK | Bank 7 | 1 | 3.3V |
| I2S_DATA | Bank 7 | 1 | 3.3V |
| I2S_MCLK | Bank 7 | 1 | 3.3V |
| **Subtotal** | | **4** | |

#### SD Card Interface (Bank 7)

| Signal | FPGA Bank | Pin Count | Voltage |
|--------|-----------|-----------|---------|
| SD_CLK | Bank 7 | 1 | 3.3V |
| SD_CMD | Bank 7 | 1 | 3.3V |
| SD_DAT0 | Bank 7 | 1 | 3.3V |
| SD_DAT1 | Bank 7 | 1 | 3.3V |
| SD_DAT2 | Bank 7 | 1 | 3.3V |
| SD_DAT3 | Bank 7 | 1 | 3.3V |
| **Subtotal** | | **6** | |

#### SPI Flash Interface (Bank 4)

| Signal | FPGA Bank | Pin Count | Voltage |
|--------|-----------|-----------|---------|
| FLASH_CS_N | Bank 4 | 1 | 3.3V |
| FLASH_CLK | Bank 4 | 1 | 3.3V |
| FLASH_DI | Bank 4 | 1 | 3.3V |
| FLASH_DO | Bank 4 | 1 | 3.3V |
| **Subtotal** | | **4** | |

#### PMOD Controller Interface (Bank 7)

| Signal | FPGA Bank | Pin Count | Voltage |
|--------|-----------|-----------|---------|
| PMOD_A[7:0] | Bank 7 | 8 | 3.3V |
| PMOD_B[7:0] | Bank 7 | 8 | 3.3V |
| **Subtotal** | | **16** | |

#### Clock and Reset

| Signal | FPGA Bank | Pin Count | Voltage |
|--------|-----------|-----------|---------|
| CLK_50M | Dedicated | 1 | 3.3V |
| RESET_N | Bank 7 | 1 | 3.3V |
| **Subtotal** | | **2** | |

### Pin Usage Summary

| Interface | Pin Count |
|-----------|-----------|
| DDR3 Memory | 47 |
| HDMI | 11 |
| RGB LCD | 28 |
| Audio (I2S) | 4 |
| SD Card | 6 |
| SPI Flash | 4 |
| PMOD (2×) | 16 |
| Clock/Reset | 2 |
| **Total** | **118** |

---

## Clocking Strategy

### Clock Domains

| Clock Domain | Frequency | Source | Purpose |
|--------------|-----------|--------|---------|
| sys_clk | 50 MHz | On-board oscillator | System reference |
| cpu_clk | 10 MHz | PLL | CPU core clock |
| ppu_clk | 25 MHz | PLL | PPU rendering clock |
| pixel_clk | 74.25 MHz | PLL | HDMI 720p pixel clock |
| audio_clk | 11.2896 MHz | PLL | Audio master clock |
| ddr3_clk | 333 MHz | PLL | DDR3 interface (666 MT/s) |

### PLL Configuration

**PLL1 - System Clocks:**
- Input: 50 MHz
- Output 0: 10 MHz (CPU)
- Output 1: 25 MHz (PPU)
- Output 2: 74.25 MHz (HDMI pixel clock)

**PLL2 - Audio Clock:**
- Input: 50 MHz
- Output 0: 11.2896 MHz (Audio MCLK)
- Output 1: 2.8224 MHz (Audio BCLK)
- Output 2: 44.1 kHz (Audio LRCLK)

**PLL3 - DDR3 Clock:**
- Input: 50 MHz
- Output 0: 333 MHz (DDR3 clock)

---

## Power Requirements

### Power Consumption Estimate

| Component | Current (mA) | Voltage | Power (mW) |
|-----------|--------------|---------|------------|
| FPGA Core | 500 | 1.0V | 500 |
| FPGA I/O | 200 | 3.3V | 660 |
| DDR3 | 150 | 1.5V | 225 |
| HDMI | 50 | 3.3V | 165 |
| Audio Amp | 100 | 5V | 500 |
| Other | 100 | 3.3V | 330 |
| **Total** | | | **~2.4W** |

### Power Rails

| Rail | Voltage | Current | Provided By |
|------|---------|---------|-------------|
| VCC | 1.0V | 500mA | On-board regulator |
| VCCX | 3.3V | 500mA | On-board regulator |
| VCC_DDR | 1.5V | 200mA | On-board regulator |
| VCC_AUDIO | 5V | 200mA | External (for amp) |

---

## Bill of Materials

### Required Components

| Item | Description | Quantity | Notes |
|------|-------------|----------|-------|
| 1 | Tang Mega 60K Dock | 1 | Main FPGA board |
| 2 | DDR3 SODIMM | 1 | 512MB (included with board) |
| 3 | HDMI Cable | 1 | Standard HDMI |
| 4 | USB Power Supply | 1 | 5V/2A minimum |
| 5 | SD Card | 1 | 8GB+ for game storage |
| 6 | SNES Controllers | 2 | Or Genesis controllers |
| 7 | PMOD Adapter | 1 | For controller connection |
| 8 | Speakers | 2 | 3W, 4Ω (optional) |
| 9 | RGB LCD Display | 1 | 4.3" or 5" (optional) |

### Optional Components

| Item | Description | Purpose |
|------|-------------|---------|
| 1 | Audio Amplifier Module | PAM8403-based 3W stereo amp |
| 2 | 3.5mm Audio Jack | Headphone output |
| 3 | Enclosure | Custom 3D printed case |
| 4 | Cooling Fan | 40mm 5V fan (optional) |

---

## Implementation Guidelines

### Development Environment

1. **Gowin IDE:** Version 1.9.11.03 or later (Educational)
2. **Device Selection:** GW5AT-LV60PG484A, Version B
3. **Package:** PBG484A

### Project Structure

> **Status:** No Verilog implementation currently exists in the repository.
> An earlier exploratory Verilog skeleton was removed to avoid it drifting
> from the console design while CoreLX and the emulator (the authoritative
> hardware contract) are still evolving. When FPGA bring-up begins, the core
> is to be built **fresh from the current emulator implementation** and the
> register/ISA tables in this document — not resurrected from the old
> skeleton. The layout below is the recommended target structure.

```
nitro_core_dx_fpga/
├── src/
│   ├── cpu/
│   │   ├── cpu_core.v
│   │   ├── alu.v
│   │   └── registers.v
│   ├── ppu/
│   │   ├── ppu_core.v
│   │   ├── bg_renderer.v
│   │   ├── sprite_engine.v
│   │   └── matrix_transform.v
│   ├── apu/
│   │   ├── apu_core.v
│   │   ├── channel.v
│   │   └── i2s_interface.v
│   ├── memory/
│   │   ├── mem_controller.v
│   │   ├── ddr3_interface.v
│   │   └── dma_engine.v
│   ├── video/
│   │   ├── hdmi_encoder.v
│   │   ├── rgb_interface.v
│   │   └── scan_doubler.v
│   ├── io/
│   │   ├── input_controller.v
│   │   └── spi_interface.v
│   └── top/
│       ├── nitro_core_dx_top.v
│       └── clock_gen.v
├── constraints/
│   └── tang_mega_60k.cst
├── sim/
│   └── testbenches/
└── docs/
    └── hardware_design.md
```

### Build Process

1. Create new project in Gowin IDE
2. Add source files
3. Import pin constraints (.cst file)
4. Synthesize design
5. Place and route
6. Generate bitstream
7. Program FPGA via USB

### Programming Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| SRAM | Download to FPGA RAM | Development, testing |
| Flash | Program SPI Flash | Production, persistent |

Recommended programming command:
```
# Flash programming (persistent)
programmer_cli --device GW5AT-LV60PG484A --operation "exFlash Erase, Program through GAO-Bridge 5A"
```

---

## Appendix A: Reference Documents

1. [GW5AT Series Datasheet](https://cdn.gowinsemi.com.cn/DS981E.pdf)
2. [Tang Mega 60K Wiki](https://wiki.sipeed.com/hardware/en/tang/tang-mega-60k/mega-60k.html)
3. [Nitro-Core-DX GitHub](https://github.com/RetroCodeRamen/Nitro-Core-DX)
4. [Gowin Software User Guide](https://www.gowinsemi.com/upload/database_doc/352/document/66d0341c5d2b4.pdf)

---

## Appendix B: Troubleshooting

| Issue | Possible Cause | Solution |
|-------|---------------|----------|
| No HDMI output | Incorrect pixel clock | Verify PLL configuration |
| DDR3 errors | Signal integrity | Check termination resistors |
| Audio noise | Ground loop | Use separate audio ground |
| Controller not working | Wrong pin mapping | Verify PMOD pinout |
| FPGA not programming | Mode setting | Set MODE pins to "001" for Flash |

---

*Document generated for Nitro-Core-DX Hardware Implementation on Tang Mega 60K*
