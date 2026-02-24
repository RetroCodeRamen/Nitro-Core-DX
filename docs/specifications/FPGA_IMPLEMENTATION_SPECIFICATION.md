# Nitro-Core-DX FPGA Implementation Specification

**Version 1.0**  
**Date:** January 30, 2026  
**Purpose:** Hardware-level specification for FPGA implementation  
**Based on:** Complete Hardware Specification v2.1

> **📌 FPGA-Level Detail**: This specification provides hardware-level details needed for FPGA implementation, including state machines, timing diagrams, resource estimates, and interface specifications.

---

## Table of Contents

1. [System Architecture](#system-architecture)
2. [Clock Domains & Synchronization](#clock-domains--synchronization)
3. [CPU Implementation](#cpu-implementation)
4. [Memory System Implementation](#memory-system-implementation)
5. [PPU Implementation](#ppu-implementation)
6. [APU Implementation](#apu-implementation)
7. [Input System Implementation](#input-system-implementation)
8. [Interconnect & Bus Architecture](#interconnect--bus-architecture)
9. [Resource Estimates](#resource-estimates)
10. [Interface Specifications](#interface-specifications)
11. [Timing Constraints](#timing-constraints)
12. [Implementation Notes](#implementation-notes)

---

## System Architecture

### Top-Level Block Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    Nitro-Core-DX FPGA System                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐    │
│  │     CPU      │    │     PPU      │    │     APU      │    │
│  │  (~7.67 MHz) │    │  (~7.67 MHz) │    │  (44.1 kHz)  │    │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘    │
│         │                    │                    │            │
│         └──────────┬──────────┴──────────┬─────────┘            │
│                   │                     │                      │
│         ┌─────────▼──────────┐  ┌──────▼──────────┐          │
│         │  Memory Controller  │  │  Clock Scheduler│          │
│         │   & Bus Arbiter    │  │  & Synchronizer │          │
│         └─────────┬───────────┘  └─────────────────┘          │
│                   │                                           │
│         ┌─────────▼───────────────────────────────────┐     │
│         │         Memory Subsystem                     │     │
│         │  ┌────────┐  ┌────────┐  ┌────────┐         │     │
│         │  │ WRAM   │  │ VRAM   │  │ CGRAM  │         │     │
│         │  │ 32KB   │  │ 64KB   │  │ 512B   │         │     │
│         │  └────────┘  └────────┘  └────────┘         │     │
│         │  ┌────────┐  ┌────────┐                      │     │
│         │  │ OAM    │  │ ExtRAM │                     │     │
│         │  │ 768B   │  │ 128KB  │                     │     │
│         │  └────────┘  └────────┘                      │     │
│         └──────────────────────────────────────────────┘     │
│                                                               │
│  ┌──────────────┐    ┌──────────────┐                       │
│  │   Input      │    │   ROM        │                       │
│  │  Controller  │    │  Interface   │                       │
│  └──────────────┘    └──────────────┘                       │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

### System Interconnect

- **CPU Bus**: 24-bit address (8-bit bank + 16-bit offset), 16-bit data
- **PPU Bus**: 16-bit address (VRAM/CGRAM/OAM), 8-bit data
- **APU Bus**: Register-based (no direct memory access)
- **Memory Controller**: Routes CPU/PPU accesses, handles arbitration

---

## Clock Domains & Synchronization

### Clock Domains

| Domain | Frequency | Source | Usage |
|--------|-----------|--------|-------|
| **CPU_CLK** | ~7.67 MHz | External crystal/PLL | CPU, PPU, Memory Controller |
| **APU_CLK** | 44.1 kHz | Derived from CPU_CLK (÷174) | Audio sample generation |
| **VIDEO_CLK** | ~7.67 MHz | Same as CPU_CLK | Video output timing |

### Clock Generation

```
CPU_CLK (7.67 MHz)
    │
    ├───► CPU (direct)
    ├───► PPU (direct)
    ├───► Memory Controller (direct)
    │
    └───► APU Clock Divider (÷174)
            │
            └───► APU_CLK (44.1 kHz)
```

### Synchronization Primitives

**Cross-Domain Signals:**
- VBlank signal (PPU → CPU): Use 2-stage synchronizer
- Interrupt signals (PPU → CPU): Use 2-stage synchronizer
- APU completion status (APU → CPU): Use 2-stage synchronizer

**Synchronizer Implementation:**
```verilog
// 2-stage synchronizer for async signals
reg sync_stage1, sync_stage2;
always @(posedge dest_clk) begin
    sync_stage1 <= async_signal;
    sync_stage2 <= sync_stage1;
end
```

---

## CPU Implementation

### CPU Pipeline

**3-Stage Pipeline:**
1. **Fetch**: Read instruction from memory
2. **Decode**: Decode opcode, mode, registers
3. **Execute**: Execute instruction, write results

**Pipeline Timing:**
- Fetch: 1 cycle (if cache hit)
- Decode: 1 cycle
- Execute: 1-5 cycles (depending on instruction)

### CPU State Machine

```
CPU_STATE_IDLE
    │
    ├──► CPU_STATE_FETCH
    │         │
    │         ├──► Read instruction from memory
    │         └──► CPU_STATE_DECODE
    │                   │
    │                   ├──► Decode opcode/mode/registers
    │                   └──► CPU_STATE_EXECUTE
    │                             │
    │                             ├──► Execute instruction
    │                             ├──► Update flags
    │                             ├──► Write results
    │                             └──► CPU_STATE_FETCH (next instruction)
    │
    └──► CPU_STATE_INTERRUPT (if interrupt pending)
              │
              ├──► Save PC, PBR, Flags to stack
              ├──► Jump to interrupt vector
              └──► CPU_STATE_FETCH
```

### Register File

**Implementation:**
- **Type**: Dual-port RAM (read 2 registers, write 1 register per cycle)
- **Size**: 8 registers × 16 bits = 128 bits
- **Ports**: 
  - Port A: Read R1 (destination)
  - Port B: Read R2 (source)
  - Port A: Write R1 (result)

**Resource Estimate:**
- 8 × 16-bit registers = 128 flip-flops
- Dual-port access = minimal additional logic

### ALU Implementation

**ALU Operations:**
- ADD, SUB: 16-bit ripple-carry adder/subtractor
- MUL: 16×16 → 32-bit multiplier (use DSP blocks)
- DIV: Iterative divider (16 cycles for 16-bit division)
- AND, OR, XOR, NOT: Bitwise logic gates
- SHL, SHR: Barrel shifter (16-bit)

**ALU Resource Estimate:**
- Adder/Subtractor: ~32 LUTs
- Multiplier: 1 DSP block (if available)
- Divider: ~200 LUTs (iterative)
- Logic gates: ~16 LUTs
- Barrel shifter: ~64 LUTs

### Instruction Fetch Unit

**Implementation:**
- **PC Register**: 24-bit (8-bit bank + 16-bit offset)
- **Instruction Cache**: Optional 4-entry cache (reduces memory access)
- **Bank Register**: PBR (Program Bank Register)

**Fetch State Machine:**
```
FETCH_IDLE
    │
    ├──► Calculate memory address (PBR:PC)
    ├──► FETCH_READ_MEMORY
    │         │
    │         ├──► Request memory read
    │         └──► FETCH_WAIT_MEMORY
    │                   │
    │                   ├──► Wait for memory ready
    │                   └──► FETCH_COMPLETE
    │                             │
    │                             └──► Instruction ready
```

---

## Memory System Implementation

### Memory Controller State Machine

```
MEM_IDLE
    │
    ├──► CPU Request?
    │         │
    │         └──► MEM_CPU_ACCESS
    │                   │
    │                   ├──► Decode address (bank + offset)
    │                   ├──► Route to WRAM/ROM/I/O
    │                   └──► MEM_COMPLETE
    │
    └──► PPU Request?
              │
              └──► MEM_PPU_ACCESS
                        │
                        ├──► Decode address (VRAM/CGRAM/OAM)
                        ├──► Check write protection (OAM)
                        └──► MEM_COMPLETE
```

### Memory Arbitration

**Priority:**
1. PPU reads (during rendering, can't wait)
2. CPU reads/writes
3. PPU writes (during VBlank)

**Arbiter Implementation:**
```verilog
always @(*) begin
    if (ppu_read_request && ppu_rendering) begin
        mem_grant = PPU;
    end else if (cpu_request) begin
        mem_grant = CPU;
    end else begin
        mem_grant = PPU;
    end
end
```

### Memory Blocks

**WRAM (32KB):**
- **Type**: Block RAM (BRAM)
- **Ports**: Dual-port (CPU read/write, PPU read)
- **Resource**: 16 × 2KB BRAM blocks (Xilinx) or equivalent

**VRAM (64KB):**
- **Type**: Block RAM (BRAM)
- **Ports**: Dual-port (CPU write, PPU read)
- **Resource**: 32 × 2KB BRAM blocks

**CGRAM (512 bytes):**
- **Type**: Block RAM (BRAM)
- **Ports**: Dual-port (CPU write, PPU read)
- **Resource**: 1 × 512-byte BRAM

**OAM (768 bytes):**
- **Type**: Block RAM (BRAM)
- **Ports**: Dual-port (CPU write, PPU read, write-protected during rendering)
- **Resource**: 1 × 1KB BRAM

**Extended WRAM (128KB):**
- **Type**: Block RAM (BRAM)
- **Ports**: Dual-port (CPU read/write)
- **Resource**: 64 × 2KB BRAM blocks

### Address Decoding

**CPU Address Decoding:**
```verilog
// Bank 0: WRAM (0x0000-0x7FFF) or I/O (0x8000+)
if (cpu_bank == 0) begin
    if (cpu_offset < 16'h8000) begin
        // WRAM access
        wram_addr = cpu_offset[14:0];
        wram_enable = 1;
    end else begin
        // I/O access
        io_addr = cpu_offset - 16'h8000;
        io_enable = 1;
    end
end
// Banks 1-125: ROM
else if (cpu_bank >= 1 && cpu_bank <= 125) begin
    if (cpu_offset >= 16'h8000) begin
        rom_addr = (cpu_bank - 1) * 32768 + (cpu_offset - 16'h8000);
        rom_enable = 1;
    end
end
// Banks 126-127: Extended WRAM
else if (cpu_bank == 126 || cpu_bank == 127) begin
    extwram_addr = (cpu_bank - 126) * 65536 + cpu_offset;
    extwram_enable = 1;
end
```

---

## PPU Implementation

### PPU Rendering Pipeline

**Per-Pixel Pipeline (320×200 = 64,000 pixels per frame):**

```
PIXEL_PIPELINE:
1. Calculate screen coordinates (scanline, dot)
2. Fetch background layers (BG0-BG3)
   - Calculate tile coordinates
   - Fetch tile data from VRAM
   - Fetch palette from CGRAM
3. Fetch sprites (up to 128 sprites)
   - Sprite evaluation (per-scanline)
   - Fetch sprite data from OAM
   - Fetch tile data from VRAM
4. Priority resolution
   - Sort backgrounds by priority (BG0=0, BG1=1, BG2=2, BG3=3)
   - Sort sprites by priority (from OAM attributes)
5. Blending
   - Alpha blending (if enabled)
   - Additive/subtractive blending (if enabled)
6. Output to frame buffer
```

### PPU State Machine

```
PPU_IDLE
    │
    ├──► PPU_STATE_FRAME_START
    │         │
    │         ├──► Clear VBlank flag
    │         ├──► Increment frame counter
    │         └──► PPU_STATE_SCANLINE_START
    │                   │
    │                   ├──► Scanline < 200? (Visible)
    │                   │         │
    │                   │         └──► PPU_STATE_RENDER_SCANLINE
    │                   │                   │
    │                   │                   ├──► For each dot (0-319):
    │                   │                   │     - Execute DMA (if enabled)
    │                   │                   │     - Render pixel
    │                   │                   │
    │                   │                   └──► PPU_STATE_HBLANK
    │                   │                             │
    │                   │                             ├──► Advance dot counter (261 cycles)
    │                   │                             └──► PPU_STATE_SCANLINE_START (next scanline)
    │                   │
    │                   └──► Scanline >= 200? (VBlank)
    │                             │
    │                             └──► PPU_STATE_VBLANK
    │                                       │
    │                                       ├──► Set VBlank flag
    │                                       ├──► Trigger VBlank interrupt
    │                                       ├──► Allow OAM writes
    │                                       └──► PPU_STATE_SCANLINE_START (next scanline)
    │
    └──► After scanline 219: PPU_STATE_FRAME_START (next frame)
```

### Tile Fetch State Machine

```
TILE_FETCH_IDLE
    │
    ├──► Calculate tile coordinates (x, y)
    ├──► TILE_FETCH_READ_TILEMAP
    │         │
    │         ├──► Read tilemap entry from VRAM
    │         │     Address = tilemap_base + (ty * 32 + tx) * 2
    │         └──► TILE_FETCH_READ_TILE
    │                   │
    │                   ├──► Read tile data from VRAM
    │                   │     Address = tile_base + tile_index * tile_size
    │                   └──► TILE_FETCH_READ_PALETTE
    │                             │
    │                             ├──► Read palette from CGRAM
    │                             │     Address = palette_base + palette_index * 32
    │                             └──► TILE_FETCH_COMPLETE
```

### Sprite Evaluation

**Per-Scanline Sprite List:**
- Evaluate all 128 sprites
- Check if sprite is on current scanline
- Build list of active sprites (max 64 per scanline)
- Sort by priority

**Sprite Evaluation State Machine:**
```
SPRITE_EVAL_IDLE
    │
    ├──► SPRITE_EVAL_LOOP (for sprite 0-127)
    │         │
    │         ├──► Read sprite Y position from OAM
    │         ├──► Check if sprite on current scanline
    │         ├──► If yes: Add to active sprite list
    │         └──► SPRITE_EVAL_LOOP (next sprite)
    │
    └──► SPRITE_EVAL_SORT
              │
              ├──► Sort active sprites by priority
              └──► SPRITE_EVAL_COMPLETE
```

### Priority Resolver

**Implementation:**
- **Inputs**: Up to 4 background pixels, up to 64 sprite pixels
- **Logic**: Compare priority values, select highest priority pixel
- **Output**: Single pixel with color and alpha

**Priority Resolution Logic:**
```verilog
// Priority comparison (higher number = higher priority)
if (sprite_priority > bg3_priority) pixel_out = sprite_pixel;
else if (bg3_enabled) pixel_out = bg3_pixel;
else if (bg2_enabled) pixel_out = bg2_pixel;
else if (bg1_enabled) pixel_out = bg1_pixel;
else if (bg0_enabled) pixel_out = bg0_pixel;
else pixel_out = backdrop_color;
```

### Blending Unit

**Blend Modes:**
- **Normal**: No blending
- **Alpha**: `result = (src * alpha + dst * (15-alpha)) / 15`
- **Additive**: `result = min(255, src + dst)`
- **Subtractive**: `result = max(0, src - dst)`

**Blending Implementation:**
```verilog
case (blend_mode)
    2'b00: blended = src;  // Normal
    2'b01: blended = (src * alpha + dst * (15 - alpha)) / 15;  // Alpha
    2'b10: blended = (src + dst > 255) ? 255 : (src + dst);  // Additive
    2'b11: blended = (src > dst) ? (src - dst) : 0;  // Subtractive
endcase
```

### Matrix Mode (Mode 7-style)

**Matrix Multiplication:**
- **Format**: 8.8 fixed-point (int16, 1.0 = 0x0100)
- **Matrix**: 2×2 transformation matrix [A B; C D]
- **Calculation**: 
  ```
  x' = A*(x-cx) + B*(y-cy) + cx
  y' = C*(x-cx) + D*(y-cy) + cy
  ```

**Matrix Math Unit:**
- **Multipliers**: 4 × 16×16 → 32-bit multipliers (use DSP blocks)
- **Adders**: 2 × 32-bit adders
- **Fixed-point**: Shift right by 8 bits after multiplication

**Resource Estimate:**
- 4 DSP blocks (for multipliers)
- ~100 LUTs (for adders and control)

### DMA State Machine

```
DMA_IDLE
    │
    ├──► DMA_ENABLED? (DMA_CONTROL bit 0)
    │         │
    │         └──► DMA_STATE_INIT
    │                   │
    │                   ├──► Initialize source/dest addresses
    │                   ├──► Read fill value (if fill mode)
    │                   └──► DMA_STATE_TRANSFER
    │                             │
    │                             ├──► Read byte from source
    │                             ├──► Write byte to destination
    │                             ├──► Increment addresses
    │                             ├──► Increment progress counter
    │                             │
    │                             ├──► Progress < Length?
    │                             │     │
    │                             │     ├──► Yes: DMA_STATE_TRANSFER (next byte)
    │                             │     │
    │                             │     └──► No: DMA_STATE_COMPLETE
    │                             │               │
    │                             │               ├──► Clear DMA enabled flag
    │                             │               └──► DMA_IDLE
    │
    └──► DMA_IDLE (waiting for enable)
```

**DMA Timing:**
- **Transfer Rate**: 1 byte per CPU cycle (~7.67 MHz)
- **Maximum Transfer**: 65535 bytes = ~8.5 ms at 7.67 MHz
- **Blocking**: Non-blocking (CPU continues during DMA)

**DMA Register Access:**
- **DMA_CONTROL (0x8060)**: Write-only (enable, mode, dest type)
- **DMA_STATUS (0x8060)**: Read-only (bit 0 = active)
- **DMA_SOURCE_BANK (0x8061)**: Write-only
- **DMA_SOURCE_OFFSET (0x8062-0x8063)**: Write-only
- **DMA_DEST_ADDR (0x8064-0x8065)**: Write-only
- **DMA_LENGTH (0x8066-0x8067)**: Read/Write

**Note:** DMA register read addresses need verification - current implementation may have inconsistencies.

---

## APU Implementation

### APU Channel State Machine

```
CHANNEL_IDLE
    │
    ├──► CHANNEL_ENABLED? (CONTROL bit 0)
    │         │
    │         └──► CHANNEL_GENERATE
    │                   │
    │                   ├──► Update phase accumulator
    │                   │     phase = phase + phase_increment
    │                   │
    │                   ├──► Generate waveform sample
    │                   │     case (waveform):
    │                   │       0: sample = sin(phase)
    │                   │       1: sample = square(phase)
    │                   │       2: sample = sawtooth(phase)
    │                   │       3: sample = noise(phase)
    │                   │
    │                   ├──► Apply volume
    │                   │     sample = sample * volume / 255
    │                   │
    │                   ├──► Update duration counter
    │                   │     duration = duration - 1
    │                   │
    │                   ├──► Duration > 0?
    │                   │     │
    │                   │     ├──► Yes: CHANNEL_GENERATE (next sample)
    │                   │     │
    │                   │     └──► No: CHANNEL_COMPLETE
    │                   │               │
    │                   │               ├──► Set completion flag
    │                   │               ├──► Loop mode?
    │                   │               │     │
    │                   │               │     ├──► Yes: Reset duration, CHANNEL_GENERATE
    │                   │               │     │
    │                   │               │     └──► No: CHANNEL_IDLE
    │
    └──► CHANNEL_IDLE (disabled)
```

### Phase Accumulator

**Implementation:**
- **Size**: 32-bit accumulator
- **Format**: Fixed-point (0-2^32 represents 0-2π)
- **Update**: `phase = phase + phase_increment` (every sample)
- **Phase Increment Calculation**: 
  ```
  phase_increment = (frequency * 2^32) / sample_rate
  phase_increment = (frequency * 2^32) / 44100
  ```

**Resource Estimate:**
- 32-bit accumulator: 32 flip-flops
- 32-bit adder: ~64 LUTs

### Waveform Generation

**Sine Wave:**
- **Method**: Lookup table (LUT) or CORDIC
- **LUT Size**: 256 entries × 16 bits = 4KB
- **Resource**: 1 BRAM block

**Square Wave:**
- **Method**: Compare phase with π
- **Logic**: `sample = (phase < π) ? 1.0 : -1.0`
- **Resource**: ~10 LUTs

**Sawtooth Wave:**
- **Method**: Linear ramp
- **Logic**: `sample = (phase / 2π) * 2 - 1`
- **Resource**: ~20 LUTs

**Noise:**
- **Method**: 15-bit LFSR (Linear Feedback Shift Register)
- **Polynomial**: x^15 + x^14 + 1
- **Update**: Every sample
- **Resource**: 15 flip-flops + ~10 LUTs

### Audio Mixer

**Implementation:**
- **Inputs**: 4 channel outputs (16-bit each)
- **Mix**: Sum all channels, apply master volume
- **Output**: 16-bit sample

**Mixing Logic:**
```verilog
mixed = (ch0 + ch1 + ch2 + ch3) * master_volume / 255;
// Clamp to 16-bit range
if (mixed > 32767) mixed = 32767;
if (mixed < -32768) mixed = -32768;
```

**Resource Estimate:**
- 4 × 16-bit adders: ~64 LUTs
- 16-bit multiplier: 1 DSP block (or ~200 LUTs)
- Clamping logic: ~20 LUTs

---

## Input System Implementation

### Input Controller State Machine

```
INPUT_IDLE
    │
    ├──► LATCH signal asserted? (write 1 to latch register)
    │         │
    │         └──► INPUT_LATCH_CAPTURE
    │                   │
    │                   ├──► Capture button states into shift register
    │                   └──► INPUT_IDLE
    │
    └──► CLK signal asserted? (read data)
              │
              └──► INPUT_SHIFT
                        │
                        ├──► Shift out one bit from shift register
                        └──► INPUT_IDLE
```

### Serial Shift Register Interface

**Timing:**
- **LATCH pulse**: 12-100μs (capture button states)
- **CLK frequency**: 100 kHz (10μs period)
- **Data bits**: 12 bits (shifted out serially)

**Implementation:**
```verilog
// Latch capture (edge-triggered)
always @(posedge latch_signal) begin
    shift_register <= button_states;  // Capture all 12 buttons
end

// Serial shift (on CLK edge)
always @(posedge clk_signal) begin
    data_out <= shift_register[0];
    shift_register <= {1'b0, shift_register[11:1]};  // Shift right
end
```

---

## Interconnect & Bus Architecture

### System Bus

**CPU Bus:**
- **Address**: 24-bit (8-bit bank + 16-bit offset)
- **Data**: 16-bit
- **Control**: Read/Write, Byte Enable

**PPU Bus:**
- **Address**: 16-bit (VRAM/CGRAM/OAM)
- **Data**: 8-bit
- **Control**: Read/Write

### Bus Arbiter

**Arbitration Logic:**
```verilog
always @(*) begin
    if (ppu_read_request && ppu_rendering) begin
        bus_grant = PPU;
        cpu_wait = 1;
    end else if (cpu_request) begin
        bus_grant = CPU;
        ppu_wait = 1;
    end else begin
        bus_grant = NONE;
        cpu_wait = 0;
        ppu_wait = 0;
    end
end
```

---

## Resource Estimates

### Overall Resource Estimates (Xilinx Artix-7)

| Resource | Estimated Usage | Notes |
|----------|----------------|-------|
| **LUTs** | ~15,000-20,000 | Logic implementation |
| **Flip-Flops** | ~8,000-12,000 | State registers |
| **BRAM** | ~120 blocks | Memory (WRAM, VRAM, CGRAM, OAM) |
| **DSP Blocks** | ~10 blocks | Multipliers (CPU MUL, PPU matrix, APU mixer) |
| **Clock Domains** | 2 | CPU/PPU (7.67 MHz), APU (44.1 kHz) |

### Subsystem Resource Breakdown

**CPU:**
- LUTs: ~3,000
- FFs: ~500
- DSP: 1 (multiplier)

**PPU:**
- LUTs: ~8,000
- FFs: ~2,000
- BRAM: ~50 blocks (VRAM, CGRAM, OAM)
- DSP: 4 (matrix multipliers)

**APU:**
- LUTs: ~2,000
- FFs: ~200
- BRAM: 1 (sine LUT)
- DSP: 1 (mixer)

**Memory System:**
- BRAM: ~70 blocks (WRAM, Extended WRAM)

**Input System:**
- LUTs: ~100
- FFs: ~50

---

## Interface Specifications

### Video Output Interface

**Format Options:**

**Option 1: VGA (640×400, scaled)**
- **Resolution**: 640×400 (2× scaling from 320×200)
- **Pixel Clock**: ~25.2 MHz
- **Timing**: Standard VGA timing
- **Color Depth**: 8-bit indexed → RGB888 via CGRAM

**Option 2: HDMI (320×200, native)**
- **Resolution**: 320×200
- **Pixel Clock**: ~7.67 MHz
- **Timing**: Custom timing (non-standard)
- **Color Depth**: RGB888

**Option 3: DVI/HDMI (640×400, scaled)**
- **Resolution**: 640×400 (2× scaling)
- **Pixel Clock**: ~25.2 MHz
- **Timing**: Standard HDMI timing
- **Color Depth**: RGB888

**Recommended:** VGA or HDMI with 2× scaling for compatibility.

### Audio Output Interface

**Format:**
- **Sample Rate**: 44.1 kHz
- **Bit Depth**: 16-bit
- **Channels**: Mono (can be duplicated to stereo)
- **Interface**: I2S or PWM

**I2S Interface:**
- **Clock**: 44.1 kHz × 32 = 1.4112 MHz (BCLK)
- **Word Select**: 44.1 kHz (LRCK)
- **Data**: 16-bit samples, MSB first

**PWM Interface:**
- **PWM Frequency**: ~352.8 kHz (8× oversampling)
- **Resolution**: 16-bit
- **Output**: Single PWM signal (low-pass filtered)

### ROM Interface

**Interface Type:** SPI Flash or Parallel ROM

**SPI Flash (Recommended):**
- **Clock**: Up to 50 MHz
- **Protocol**: Standard SPI (Mode 0)
- **Capacity**: 8MB+ (for 7.8MB ROM)
- **Pins**: CS, SCK, MOSI, MISO

**Parallel ROM (Alternative):**
- **Address**: 23-bit (for 8MB)
- **Data**: 16-bit
- **Control**: /OE, /CE
- **Pins**: 23 address + 16 data + 2 control = 41 pins

### Controller Interface

**Serial Interface:**
- **Protocol**: SNES-style serial shift register
- **Signals**: DATA, LATCH, CLK
- **Timing**: 100 kHz clock, 12-100μs latch pulse
- **Pins**: 3 per controller (6 total for 2 controllers)

---

## Timing Constraints

### Setup/Hold Times

**CPU Clock Domain (7.67 MHz):**
- **Clock Period**: ~130.4 ns
- **Setup Time**: 5 ns (typical)
- **Hold Time**: 2 ns (typical)
- **Clock-to-Output**: 10 ns (typical)

**APU Clock Domain (44.1 kHz):**
- **Clock Period**: ~22.7 μs
- **Setup Time**: 5 ns
- **Hold Time**: 2 ns

### Critical Paths

**CPU Critical Path:**
- Instruction fetch → Decode → Execute → Write back
- **Target**: < 130 ns (1 CPU cycle)

**PPU Critical Path:**
- Tile fetch → Priority resolve → Blend → Output
- **Target**: < 130 ns (1 pixel per cycle)

**Memory Access:**
- Address decode → Memory read → Data ready
- **Target**: < 130 ns (1 cycle access)

---

## Implementation Notes

### Fixed-Point Arithmetic

**Matrix Mode (8.8 fixed-point):**
- **Format**: int16, 1.0 = 0x0100
- **Range**: -128.0 to +127.996
- **Precision**: 1/256 = 0.0039

**APU Phase (32-bit fixed-point):**
- **Format**: uint32, 2π = 2^32
- **Range**: 0 to 2π
- **Precision**: 2π/2^32 ≈ 1.46×10^-9 radians

### Rounding Modes

**Default**: Truncate (round toward zero)
- Matrix calculations: Truncate after multiplication
- Audio mixing: Truncate after division

### Saturation Behavior

**Audio Mixing:**
- **Overflow**: Clamp to ±32767
- **Underflow**: Clamp to -32768

**Color Blending:**
- **Overflow**: Clamp to 255
- **Underflow**: Clamp to 0

### Reset Sequence

**Power-On Reset:**
1. Assert reset signal (100+ clock cycles)
2. Initialize all registers to reset state
3. Clear all memory (optional)
4. Set PC to ROM entry point
5. Deassert reset signal
6. Begin instruction fetch

**Reset Timing:**
- **Reset Duration**: 100 CPU cycles (~13 μs)
- **Initialization**: 1000+ cycles (~130 μs)

---

## Known Issues & TODO

### DMA Register Addresses
- **Issue**: DMA_LENGTH read addresses may be incorrect
- **Status**: Needs verification during implementation
- **Action**: Verify read addresses match write addresses

### Cycle-Accurate Timing
- **Status**: Most subsystems cycle-accurate
- **DMA**: Cycle-accurate (1 byte per cycle)
- **PPU**: Cycle-accurate (1 pixel per cycle)
- **CPU**: Cycle-accurate (instruction-level)

### Testing Requirements
- **Unit Tests**: Each subsystem independently
- **Integration Tests**: CPU + Memory + PPU + APU
- **Timing Tests**: Verify cycle-accurate behavior
- **ROM Compatibility**: Test with existing ROMs

---

## Appendix: Verilog Code Templates

### CPU Instruction Fetch
```verilog
always @(posedge clk) begin
    if (reset) begin
        pc_bank <= 0;
        pc_offset <= 0;
        instruction <= 0;
    end else if (cpu_state == FETCH) begin
        // Calculate memory address
        mem_addr = {pc_bank, pc_offset};
        // Request memory read
        mem_read_req <= 1;
    end else if (mem_read_ready) begin
        instruction <= mem_read_data;
        pc_offset <= pc_offset + 2;  // 16-bit instructions
        cpu_state <= DECODE;
    end
end
```

### PPU Pixel Pipeline
```verilog
always @(posedge clk) begin
    if (ppu_state == RENDER_PIXEL) begin
        // Fetch backgrounds
        bg0_pixel <= fetch_background(0, x, y);
        bg1_pixel <= fetch_background(1, x, y);
        bg2_pixel <= fetch_background(2, x, y);
        bg3_pixel <= fetch_background(3, x, y);
        
        // Fetch sprites
        sprite_pixel <= fetch_sprite(x, y);
        
        // Priority resolve
        pixel_out <= priority_resolve(bg0, bg1, bg2, bg3, sprite);
        
        // Blend
        pixel_final <= blend(pixel_out, alpha);
        
        // Output
        frame_buffer[scanline * 320 + x] <= pixel_final;
    end
end
```

---

**End of FPGA Implementation Specification v1.0**
