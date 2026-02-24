# Nitro-Core-DX Hardware Implementation on Tang Mega 60K

## Comprehensive Hardware Design Document

**Version:** 1.0  
**Date:** February 2026  
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

| Address | Register | Description |
|---------|----------|-------------|
| 0x8000 - 0x80FF | PPU_CTRL | PPU Control registers |
| 0x8100 - 0x81FF | PPU_SCROLL | Scroll registers (4 layers) |
| 0x8200 - 0x82FF | PPU_MATRIX | Matrix mode registers |
| 0x8300 - 0x83FF | SPRITE_OAM | Sprite attribute memory |
| 0x8400 - 0x84FF | CGRAM | Color palette RAM |
| 0x9000 - 0x900F | APU_CTRL | Audio channel control |
| 0x9010 - 0x901F | APU_FREQ | Audio frequency registers |
| 0x9020 - 0x902F | APU_VOL | Audio volume registers |
| 0x9030 | APU_MASTER | Master volume control |
| 0x9031 | CHANNEL_COMPLETION_STATUS | Audio completion flags |
| 0xA000 - 0xA00F | INPUT_CTRL | Controller input registers |
| 0x803E | VBLANK_FLAG | VBlank synchronization flag |
| 0x803F | FRAME_COUNTER_LOW | Frame counter (low byte) |
| 0x8040 | FRAME_COUNTER_HIGH | Frame counter (high byte) |

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

| Parameter | Value |
|-----------|-------|
| Channels | 4 independent |
| Sample Rate | 44,100 Hz |
| Bit Depth | 16-bit |
| Waveforms | Sine, Square, Saw, Noise |
| Output | Stereo (2× 3W speakers + 3.5mm headphone) |

### Channel Features

Each audio channel supports:

| Feature | Range |
|---------|-------|
| Frequency | 20 Hz - 20 kHz |
| Volume | 0 - 255 (8-bit) |
| Panning | Left - Center - Right |

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
