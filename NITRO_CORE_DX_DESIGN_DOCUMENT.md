# Nitro-Core-DX Fantasy Console Design Document

**Version 1.0**  
**Last Updated: December 2024**

---

## Table of Contents

1. [Vision and Goals](#vision-and-goals)
2. [System Overview](#system-overview)
3. [CPU Architecture](#cpu-architecture)
4. [Memory System](#memory-system)
5. [PPU (Graphics System)](#ppu-graphics-system)
6. [APU (Audio System)](#apu-audio-system)
7. [Input System](#input-system)
8. [ROM Format](#rom-format)
9. [Emulator Implementation Guide](#emulator-implementation-guide)
10. [Reference Tables](#reference-tables)

---

## Vision and Goals

### The Dream Console

Nitro-Core-DX is a **fantasy console** that represents the "perfect 1990s console" - combining the best features from the Super Nintendo Entertainment System (SNES) and Sega Genesis (Mega Drive) to create an ultimate 16-bit gaming experience.

### Design Philosophy

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
- FM synthesis audio capabilities (planned)

**Unique Features:**
- Built-in 3D assist co-processor (SuperFX-style, planned)
- Hybrid audio system (FM synthesis + PCM)
- Modern development-friendly architecture
- **Enhanced Matrix Mode**: Large world maps with tile stitching, vertical sprites for pseudo-3D worlds
- **Integrated Development Toolkit**: Hex editor, component logging, debugger, performance profiler

### Target Experience

Nitro-Core-DX aims to deliver:
- **SNES-quality graphics** with **Genesis-level performance**
- **Mode 7-style effects** for 3D landscapes and racing games
- **Rich parallax scrolling** with 4 independent background layers
- **Smooth 60 FPS** gameplay with complex graphics
- **Developer-friendly** architecture for easy game development

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
| **Development Toolkit** | Hex editor, component logging, debugger, profiler |

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
│  ├─ Banks 1-125: ROM Space (7.8MB)                     │
│  └─ Banks 126-127: Extended WRAM (128KB)                │
├─────────────────────────────────────────────────────────┤
│  PPU (Picture Processing Unit)                         │
│  ├─ 4 Background Layers (BG0-BG3)                      │
│  ├─ 128 Sprites                                         │
│  ├─ Matrix Mode (Mode 7-style effects)                 │
│  ├─ Windowing System (2 windows, OR/AND/XOR/XNOR)      │
│  ├─ HDMA (Per-scanline scroll)                          │
│  ├─ VRAM: 64KB                                          │
│  └─ CGRAM: 512 bytes (256 colors × 2 bytes)            │
├─────────────────────────────────────────────────────────┤
│  APU (Audio Processing Unit)                            │
│  ├─ 4 Audio Channels                                    │
│  ├─ Waveforms: Sine, Square, Saw, Noise                 │
│  └─ Master Volume Control                               │
├─────────────────────────────────────────────────────────┤
│  Input System                                           │
│  └─ SNES-style 12-button Controller                    │
└─────────────────────────────────────────────────────────┘
```

---

## CPU Architecture

### CPU Overview

The Nitro-Core-DX CPU is a custom 16-bit processor with banked 24-bit addressing, designed for high performance while maintaining simplicity.

**Key Features:**
- 8 general-purpose 16-bit registers
- 24-bit logical addressing (bank:offset)
- Simple, orthogonal instruction set
- Fast execution (10 MHz)
- Interrupt support (VBlank, Timer, NMI)

### Registers

#### General Purpose Registers

- **R0-R7**: 8 general-purpose 16-bit registers
  - All registers are equal (no special-purpose restrictions)
  - Used for arithmetic, data movement, addressing

#### Special Registers

- **PC (Program Counter)**: 24-bit logical address
  - `pc_bank`: Bank number (0-255)
  - `pc_offset`: 16-bit offset within bank (0x0000-0xFFFF)
  - Used for instruction fetching

- **SP (Stack Pointer)**: 16-bit offset in bank 0
  - Starts at 0x1FFF (top of WRAM)
  - Grows downward (decrements on push)
  - Wraps around if underflow

- **PBR (Program Bank Register)**: Current program bank
  - Used for instruction fetches
  - Typically matches `pc_bank`

- **DBR (Data Bank Register)**: Current data bank
  - Used for memory load/store operations
  - Can be set independently from PBR

#### Flags Register

The CPU maintains a flags register with the following bits:

| Bit | Flag | Name | Description |
|-----|------|------|-------------|
| 0 | Z | Zero | Set when result equals zero |
| 1 | N | Negative | Set when result bit 15 is set (signed negative) |
| 2 | C | Carry | Set on unsigned overflow (carry out) |
| 3 | V | Overflow | Set on signed overflow |
| 4 | I | Interrupt | Interrupt mask (1 = interrupts disabled) |

**Flag Usage:**
- **Z**: Set by arithmetic/logical operations, used by BEQ/BNE
- **N**: Set by arithmetic/logical operations, used by BLT/BGE
- **C**: Set by ADD/SUB/SHL/SHR, indicates unsigned overflow
- **V**: Set by ADD/SUB, indicates signed overflow
- **I**: Controls interrupt handling

### Addressing Modes

The CPU supports the following addressing modes:

1. **Register Direct** (Mode 0)
   - Format: `MOV R1, R2`
   - Operand is a register value

2. **Immediate** (Mode 1)
   - Format: `MOV R1, #0x1234`
   - Operand is a 16-bit immediate value (next instruction word)
   - Instruction is 2 words (instruction + immediate)

3. **Direct Address** (Mode 2)
   - Format: `MOV R1, [R2]`
   - Load from memory at address in R2
   - Uses DBR as bank register
   - Address = DBR:R2

4. **Indirect** (Mode 3)
   - Format: `MOV [R1], R2`
   - Store to memory at address in R1
   - Uses DBR as bank register
   - Address = DBR:R1
   - **Special**: For I/O addresses (0x8000+), always uses bank 0

5. **Stack** (Mode 4/5)
   - Format: `PUSH R1` / `POP R1`
   - Uses SP in bank 0
   - Stack grows downward

### Instruction Set

#### Instruction Encoding

All instructions are 16-bit words with the following format:

```
Bits [15:12]: Opcode family (0x0-0xF)
Bits [11:8]:  Mode/subop
Bits [7:4]:   Register 1 (destination)
Bits [3:0]:   Register 2 (source)
```

Some instructions require an additional 16-bit immediate value (Mode 1 instructions).

#### Instruction Categories

**1. Data Movement**

| Instruction | Opcode | Modes | Description |
|------------|--------|-------|-------------|
| MOV | 0x1000 | 0-5 | Move/load/store |
| PUSH | 0x1000 | 4 | Push to stack |
| POP | 0x1000 | 5 | Pop from stack |

**MOV Modes:**
- Mode 0: `MOV R1, R2` - Register to register
- Mode 1: `MOV R1, #imm` - Immediate to register (2 words)
- Mode 2: `MOV R1, [R2]` - Load from memory [DBR:R2]
- Mode 3: `MOV [R1], R2` - Store to memory [DBR:R1]
- Mode 4: `PUSH R1` - Push R1 to stack
- Mode 5: `POP R1` - Pop stack to R1

**2. Arithmetic**

| Instruction | Opcode | Modes | Description | Flags |
|------------|--------|-------|-------------|-------|
| ADD | 0x2000 | 0-1 | Add | Z, N, C, V |
| SUB | 0x3000 | 0-1 | Subtract | Z, N, C, V |
| MUL | 0x4000 | 0-1 | Multiply (low 16 bits) | Z, N |
| DIV | 0x5000 | 0-1 | Divide | Z, N |

**Modes:**
- Mode 0: `ADD R1, R2` - Register to register
- Mode 1: `ADD R1, #imm` - Immediate (2 words)

**3. Logical**

| Instruction | Opcode | Modes | Description | Flags |
|------------|--------|-------|-------------|-------|
| AND | 0x6000 | 0-1 | Bitwise AND | Z, N |
| OR | 0x7000 | 0-1 | Bitwise OR | Z, N |
| XOR | 0x8000 | 0-1 | Bitwise XOR | Z, N |
| NOT | 0x9000 | 0 | Bitwise NOT | Z, N |

**4. Shift**

| Instruction | Opcode | Modes | Description | Flags |
|------------|--------|-------|-------------|-------|
| SHL | 0xA000 | 0-1 | Shift left | Z, N, C |
| SHR | 0xB000 | 0-1 | Shift right | Z, N, C |

**C flag**: Set to the last bit shifted out

**5. Comparison and Branching**

| Instruction | Opcode | Mode | Description | Condition |
|------------|--------|------|-------------|-----------|
| CMP | 0xC000 | 0-1 | Compare (sets flags) | - |
| BEQ | 0xC100 | 1 | Branch if equal | Z = 1 |
| BNE | 0xC200 | 1 | Branch if not equal | Z = 0 |
| BGT | 0xC300 | 1 | Branch if greater (signed) | Z = 0 AND N = 0 |
| BLT | 0xC400 | 1 | Branch if less (signed) | N = 1 |
| BGE | 0xC500 | 1 | Branch if >= (signed) | N = 0 |
| BLE | 0xC600 | 1 | Branch if <= (signed) | Z = 1 OR N = 1 |

**Branch Instructions:**
- Format: `BEQ offset` (2 words: instruction + signed 16-bit offset)
- Offset is relative to PC **after** the instruction and offset word are fetched
- Offset = target_address - (PC + 4)
- Range: -32768 to +32767 bytes

**6. Jump and Call**

| Instruction | Opcode | Mode | Description |
|------------|--------|------|-------------|
| JMP | 0xD000 | 1 | Jump (relative) |
| CALL | 0xE000 | 1 | Call subroutine |
| RET | 0xF000 | 0 | Return from subroutine |

**CALL:**
- Pushes return address (PBR:PC) to stack
- Pushes PBR first, then PC (both 16-bit)
- Jumps to target address

**RET:**
- Pops PC from stack, then PBR
- Returns to caller

**7. Other**

| Instruction | Opcode | Description |
|------------|--------|-------------|
| NOP | 0x0000 | No operation (1 cycle) |

### Instruction Execution

**Cycle Counts:**
- Most instructions: 1-2 cycles
- Memory access: +1 cycle
- Branch taken: +1 cycle
- CALL/RET: 2-3 cycles

**Instruction Fetch:**
1. Read instruction word from [PBR:PC]
2. Increment PC by 2
3. Decode opcode, mode, registers
4. If Mode 1 (immediate), fetch next word and increment PC by 2
5. Execute instruction
6. Update flags if applicable

**Memory Access:**
- Load: `MOV R1, [R2]` reads from [DBR:R2]
- Store: `MOV [R1], R2` writes to [DBR:R1]
- **Exception**: I/O addresses (0x8000+) always use bank 0, regardless of DBR

### Interrupts

**Interrupt Types:**
- **VBlank** (INT_VBLANK = 1): Triggered at start of vertical blanking period
- **Timer** (INT_TIMER = 2): Timer interrupt (future)
- **NMI** (INT_NMI = 3): Non-maskable interrupt

**Interrupt Handling:**
1. If I flag is set, interrupts are disabled
2. When interrupt occurs, set `interrupt_pending`
3. At end of current instruction, if interrupt pending and I=0:
   - Push PBR and PC to stack
   - Jump to interrupt handler (address TBD)
   - Set I flag (disable further interrupts)

**Interrupt Return:**
- Use RTI instruction (future)
- Pops PC, PBR, and flags from stack
- Clears I flag

---

## Memory System

### Memory Architecture

Nitro-Core-DX uses a **banked memory architecture** with 24-bit logical addressing:

- **Logical Address**: `bank:offset` (24 bits total)
  - Bank: 8 bits (0-255)
  - Offset: 16 bits (0x0000-0xFFFF)
- **Total Address Space**: 16MB (256 banks × 64KB)

### Memory Map

| Bank Range | Address Range | Description | Size |
|------------|---------------|-------------|------|
| **Bank 0** | 0x0000-0x7FFF | Work RAM (WRAM) | 32KB |
| **Bank 0** | 0x8000-0xFFFF | I/O Registers | 32KB |
| **Banks 1-125** | 0x8000-0xFFFF | ROM Space (LoROM mapping) | 7.8MB |
| **Banks 126-127** | 0x0000-0xFFFF | Extended WRAM | 128KB |

**Notes:**
- ROM appears at offset 0x8000+ in each ROM bank (SNES LoROM-style)
- Extended WRAM is contiguous across banks 126-127
- I/O registers are always in bank 0, offset 0x8000+

### Memory Access Rules

**Read Access:**
- WRAM (Bank 0, 0x0000-0x7FFF): Read from WRAM array
- I/O (Bank 0, 0x8000+): Route to I/O handler (PPU/APU/Input)
- ROM (Banks 1-125, 0x8000+): Read from ROM data
- Extended WRAM (Banks 126-127): Read from extended WRAM array

**Write Access:**
- WRAM (Bank 0, 0x0000-0x7FFF): Write to WRAM array
- I/O (Bank 0, 0x8000+): Route to I/O handler (write-only for most registers)
- ROM: Read-only, writes ignored
- Extended WRAM (Banks 126-127): Write to extended WRAM array

**I/O Address Special Handling:**
- When writing to I/O addresses (0x8000+), **always use bank 0**, regardless of DBR
- This ensures I/O registers are always accessible

### I/O Register Map

#### PPU Registers (0x8000-0x8FFF)

**Base Address**: 0x8000

| Offset | Name | Size | Description |
|--------|------|------|-------------|
| 0x00 | BG0_SCROLLX_L | 8-bit | Background 0 scroll X (low byte) |
| 0x01 | BG0_SCROLLX_H | 8-bit | Background 0 scroll X (high byte) |
| 0x02 | BG0_SCROLLY_L | 8-bit | Background 0 scroll Y (low byte) |
| 0x03 | BG0_SCROLLY_H | 8-bit | Background 0 scroll Y (high byte) |
| 0x04 | BG1_SCROLLX_L | 8-bit | Background 1 scroll X (low byte) |
| 0x05 | BG1_SCROLLX_H | 8-bit | Background 1 scroll X (high byte) |
| 0x06 | BG1_SCROLLY_L | 8-bit | Background 1 scroll Y (low byte) |
| 0x07 | BG1_SCROLLY_H | 8-bit | Background 1 scroll Y (high byte) |
| 0x08 | BG0_CONTROL | 8-bit | BG0 control: bit 0=enable, bit 1=tile size (0=8×8, 1=16×16) |
| 0x09 | BG1_CONTROL | 8-bit | BG1 control: bit 0=enable, bit 1=tile size |
| 0x0A | BG2_SCROLLX_L | 8-bit | Background 2 scroll X (low byte) |
| 0x0B | BG2_SCROLLX_H | 8-bit | Background 2 scroll X (high byte) |
| 0x0C | BG2_SCROLLY_L | 8-bit | Background 2 scroll Y (low byte) |
| 0x0D | BG2_SCROLLY_H | 8-bit | Background 2 scroll Y (high byte) |
| 0x0E | VRAM_ADDR_L | 8-bit | VRAM address (low byte) |
| 0x0F | VRAM_ADDR_H | 8-bit | VRAM address (high byte) |
| 0x10 | VRAM_DATA | 8-bit | VRAM data (auto-increments address) |
| 0x12 | CGRAM_ADDR | 8-bit | CGRAM (palette) address (0-255) |
| 0x13 | CGRAM_DATA | 8-bit | CGRAM data (RGB555, 16-bit write: low byte, then high byte) |
| 0x14 | OAM_ADDR | 8-bit | OAM (sprite) address (0-127) |
| 0x15 | OAM_DATA | 8-bit | OAM data (multiple bytes per sprite) |
| 0x16 | FRAMEBUFFER_ENABLE | 8-bit | Framebuffer enable (0=off, 1=on) |
| 0x17 | DISPLAY_MODE | 8-bit | Display mode (0=landscape, 1=portrait) |
| 0x18 | MATRIX_CONTROL | 8-bit | Matrix Mode: bit 0=enable, bit 1=mirror_h, bit 2=mirror_v |
| 0x19 | MATRIX_A_L | 8-bit | Transformation matrix A (low byte, 8.8 fixed point) |
| 0x1A | MATRIX_A_H | 8-bit | Transformation matrix A (high byte) |
| 0x1B | MATRIX_B_L | 8-bit | Transformation matrix B (low byte) |
| 0x1C | MATRIX_B_H | 8-bit | Transformation matrix B (high byte) |
| 0x1D | MATRIX_C_L | 8-bit | Transformation matrix C (low byte) |
| 0x1E | MATRIX_C_H | 8-bit | Transformation matrix C (high byte) |
| 0x1F | MATRIX_D_L | 8-bit | Transformation matrix D (low byte) |
| 0x20 | MATRIX_D_H | 8-bit | Transformation matrix D (high byte) |
| 0x21 | BG2_CONTROL | 8-bit | BG2 control: bit 0=enable, bit 1=tile size |
| 0x22 | BG3_SCROLLX_L | 8-bit | Background 3 scroll X (low byte) |
| 0x23 | BG3_SCROLLX_H | 8-bit | Background 3 scroll X (high byte) |
| 0x24 | BG3_SCROLLY_L | 8-bit | Background 3 scroll Y (low byte) |
| 0x25 | BG3_SCROLLY_H | 8-bit | Background 3 scroll Y (high byte) |
| 0x26 | BG3_CONTROL | 8-bit | BG3 control: bit 0=enable, bit 1=tile size (can be affine layer) |
| 0x27 | MATRIX_CENTER_X_L | 8-bit | Matrix center point X (low byte) |
| 0x28 | MATRIX_CENTER_X_H | 8-bit | Matrix center point X (high byte) |
| 0x29 | MATRIX_CENTER_Y_L | 8-bit | Matrix center point Y (low byte) |
| 0x2A | MATRIX_CENTER_Y_H | 8-bit | Matrix center point Y (high byte) |
| 0x2B | WINDOW0_LEFT | 8-bit | Window 0 left edge (0-319) |
| 0x2C | WINDOW0_RIGHT | 8-bit | Window 0 right edge (0-319) |
| 0x2D | WINDOW0_TOP | 8-bit | Window 0 top edge (0-199) |
| 0x2E | WINDOW0_BOTTOM | 8-bit | Window 0 bottom edge (0-199) |
| 0x2F | WINDOW1_LEFT | 8-bit | Window 1 left edge (0-319) |
| 0x30 | WINDOW1_RIGHT | 8-bit | Window 1 right edge (0-319) |
| 0x31 | WINDOW1_TOP | 8-bit | Window 1 top edge (0-199) |
| 0x32 | WINDOW1_BOTTOM | 8-bit | Window 1 bottom edge (0-199) |
| 0x33 | WINDOW_CONTROL | 8-bit | Window control: bit 0=Window0 enable, bit 1=Window1 enable, bits 2-3=logic (0=OR, 1=AND, 2=XOR, 3=XNOR) |
| 0x34 | WINDOW_MAIN_ENABLE | 8-bit | Main window enable per layer: bit 0=BG0, 1=BG1, 2=BG2, 3=BG3, 4=sprites |
| 0x35 | WINDOW_SUB_ENABLE | 8-bit | Sub window enable (for color math, future use) |
| 0x36 | HDMA_CONTROL | 8-bit | HDMA control: bit 0=enable, bits 1-4=layer enable (bit 1=BG0, 2=BG1, 3=BG2, 4=BG3) |
| 0x37 | HDMA_TABLE_BASE_L | 8-bit | HDMA table base address in WRAM (low byte) |
| 0x38 | HDMA_TABLE_BASE_H | 8-bit | HDMA table base address (high byte) |
| 0x39 | HDMA_SCANLINE | 8-bit | Current scanline for HDMA write (0-199) |
| 0x3A | HDMA_BG0_SCROLLX_L | 8-bit | HDMA: BG0 scroll X for current scanline (low byte) |
| 0x3B | HDMA_BG0_SCROLLX_H | 8-bit | HDMA: BG0 scroll X (high byte) |
| 0x3C | HDMA_BG0_SCROLLY_L | 8-bit | HDMA: BG0 scroll Y (low byte) |
| 0x3D | HDMA_BG0_SCROLLY_H | 8-bit | HDMA: BG0 scroll Y (high byte) |
| 0x3E-0x4D | HDMA_BG1/BG2/BG3 | 8-bit | Similar registers for BG1, BG2, BG3 |

**VRAM Access:**
- Set VRAM address via VRAM_ADDR_L/H (0x0E/0x0F)
- Read/write data via VRAM_DATA (0x10)
- Address auto-increments after each VRAM_DATA access
- VRAM size: 64KB (0x0000-0xFFFF)

**CGRAM Access:**
- Set CGRAM address via CGRAM_ADDR (0x12)
- Write color via CGRAM_DATA (0x13) - **16-bit write**:
  - First write: Low byte (RGB555 bits 0-7)
  - Second write: High byte (RGB555 bits 8-14)
- Address auto-increments after each 16-bit color write
- CGRAM size: 512 bytes (256 colors × 2 bytes)

**RGB555 Format:**
- 15-bit color: `RRRRR GGGGG BBBBB`
- Low byte: `GGGBBBBB` (bits 0-7)
- High byte: `0RRRRRGG` (bits 8-14)
- Example: Red (R=31, G=0, B=0) = 0x001F (low=0x1F, high=0x00)

#### APU Registers (0x9000-0x9FFF)

**Base Address**: 0x9000

| Offset | Name | Size | Description |
|--------|------|------|-------------|
| 0x00 | CH0_FREQ_LOW | 8-bit | Channel 0 frequency (low byte) |
| 0x01 | CH0_FREQ_HIGH | 8-bit | Channel 0 frequency (high byte) |
| 0x02 | CH0_VOLUME | 8-bit | Channel 0 volume (0-255) |
| 0x03 | CH0_CONTROL | 8-bit | Channel 0 control: bit 0=enable, bits 1-2=waveform (0=sine, 1=square, 2=saw, 3=noise) |
| 0x04-0x07 | CH1_* | 8-bit | Channel 1 (same pattern) |
| 0x08-0x0B | CH2_* | 8-bit | Channel 2 (same pattern) |
| 0x0C-0x0F | CH3_* | 8-bit | Channel 3 (same pattern, bit 1 of CONTROL selects noise vs square) |
| 0x10 | MASTER_VOLUME | 8-bit | Master volume (0-255) |

**Frequency Encoding:**
- 16-bit frequency value (FREQ_LOW | (FREQ_HIGH << 8))
- Frequency in Hz (direct value)
- Phase increment per sample = (frequency / 44100) × 2π

**Waveform Types:**
- 0 = Sine wave
- 1 = Square wave (50% duty cycle)
- 2 = Sawtooth wave
- 3 = Noise (LFSR-based, channel 3 only)

#### Input Registers (0xA000-0xAFFF)

**Base Address**: 0xA000

| Offset | Name | Size | Description |
|--------|------|------|-------------|
| 0x00 | CONTROLLER1 | 8-bit | Controller 1 button state (read, low byte) |
| 0x01 | CONTROLLER1_LATCH | 8-bit | Controller 1 latch (write 1 to latch, 0 to release) / Controller 1 high byte (read) |
| 0x02 | CONTROLLER2 | 8-bit | Controller 2 button state (read, low byte) |
| 0x03 | CONTROLLER2_LATCH | 8-bit | Controller 2 latch (write 1 to latch, 0 to release) / Controller 2 high byte (read) |

**Button Bit Mapping:**
- Bit 0: UP
- Bit 1: DOWN
- Bit 2: LEFT
- Bit 3: RIGHT
- Bit 4: A
- Bit 5: B
- Bit 6: X
- Bit 7: Y
- Bit 8: L (high byte)
- Bit 9: R (high byte)
- Bit 10: START (high byte)
- Bit 11: SELECT (high byte)

**Input Reading:**
1. Write 1 to CONTROLLER1_LATCH to latch button states
2. Read CONTROLLER1 (16-bit) to get button state
3. Write 0 to CONTROLLER1_LATCH to release latch

---

## PPU (Graphics System)

### Display Specifications

- **Resolution**: 320×200 pixels (landscape mode) or 200×320 (portrait mode)
- **Color Depth**: 256 colors (8-bit indexed)
- **Palette**: 256-color CGRAM (RGB555 format, 32,768 possible colors)
- **Frame Rate**: 60 FPS
- **Pixel Format**: 8-bit palette index → RGB555 lookup → RGB output

### Background Layers

Nitro-Core-DX features **4 independent background layers** (BG0, BG1, BG2, BG3) for advanced parallax and layering effects.

#### Layer Properties

Each layer has:
- **Independent scroll** (X and Y, 16-bit each)
- **Tile size**: 8×8 or 16×16 pixels (configurable per layer)
- **Tile format**: 4bpp (4 bits per pixel, 16 colors per tile)
- **Tilemap**: 32×32 tiles (256×256 pixels for 8×8 tiles, 512×512 for 16×16)
- **Priority**: BG3 (highest) → BG2 → BG1 → BG0 (lowest)
- **Enable/disable**: Per-layer control

#### Tile Format

**4bpp (4 bits per pixel):**
- Each pixel uses 4 bits (0-15) to index into a 16-color palette
- 2 pixels per byte
- Tile size: 8×8 = 64 pixels = 32 bytes per tile
- Tile size: 16×16 = 256 pixels = 128 bytes per tile

**Tile Data Layout:**
- Tiles stored sequentially in VRAM
- Each tile: 32 bytes (8×8) or 128 bytes (16×16)
- Pixel order: Row-major, 2 pixels per byte
  - Even pixels (0, 2, 4, ...): Upper 4 bits of byte
  - Odd pixels (1, 3, 5, ...): Lower 4 bits of byte

#### Tilemap Format

**Tilemap Entry** (2 bytes per tile):
- **Byte 0** (Low): Tile index (0-255 for 8×8, 0-63 for 16×16)
- **Byte 1** (High): Attributes
  - Bits [3:0]: Palette index (0-15)
  - Bit 4: Flip X
  - Bit 5: Flip Y
  - Bits [7:6]: Reserved

**Tilemap Layout:**
- 32×32 tilemap = 1024 entries = 2048 bytes
- Row-major order: Entry at (x, y) = y × 32 + x
- Address = `tile_map_base + (y × 32 + x) × 2`

#### Layer Control Register

**BGx_CONTROL** (8-bit):
- Bit 0: Enable (1=enabled, 0=disabled)
- Bit 1: Tile size (0=8×8, 1=16×16)
- Bits [7:2]: Reserved

### Matrix Mode (Mode 7-Style Effects)

Matrix Mode enables advanced perspective and rotation effects on BG0, perfect for creating 3D-style landscapes, racing game tracks, and **pseudo-3D world maps**. This is the console's implementation of SNES Mode 7-style effects, enhanced for larger worlds.

#### Features

- **Rotation**: Rotate the entire background
- **Scaling**: Zoom in/out with perspective
- **Perspective**: Create pseudo-3D "looking down a road" effects
- **Mirroring**: Horizontal and vertical mirroring support
- **Large World Maps**: Support for large tilemaps through tile stitching and extended VRAM
- **Vertical Sprites**: Sprites rendered in 3D space (buildings, people, objects) that scale and position based on Matrix Mode transformation

#### Transformation Matrix

The transformation uses a 2×2 affine matrix:

```
[x']   [A B]   [x - CX]
[y'] = [C D] × [y - CY]
```

Where:
- `A, B, C, D`: 8.8 fixed-point values (1.0 = 0x0100)
- `CX, CY`: Center point of transformation (16-bit)
- `x, y`: Screen coordinates (0-319, 0-199)
- `x', y'`: Transformed tilemap coordinates

#### Large World Maps

Matrix Mode supports **large world maps** through:

1. **Extended Tilemaps**: 
   - Normal tilemaps: 32×32 tiles (256×256 pixels for 8×8 tiles)
   - Matrix Mode can use larger tilemaps by accessing extended VRAM regions
   - Tilemaps can be stitched together to create seamless large worlds
   - Support for tilemaps up to 128×128 tiles (1024×1024 pixels) or larger

2. **Tile Stitching**:
   - Multiple tilemaps can be arranged to create larger worlds
   - Seamless transitions between tilemap regions
   - World coordinates can span multiple tilemap boundaries

3. **World Coordinate System**:
   - Matrix Mode transforms screen coordinates to world coordinates
   - World coordinates can extend beyond single tilemap boundaries
   - Tilemap wrapping or seamless stitching for infinite worlds

**Implementation Notes:**
- For large worlds, use multiple tilemaps in VRAM
- Calculate which tilemap region contains the transformed coordinates
- Seamlessly render tiles from adjacent tilemaps when coordinates cross boundaries
- Consider using extended VRAM or multiple VRAM banks for very large worlds

#### 8.8 Fixed Point Format

Values stored as 16-bit signed integers:
- Integer part: bits 15-8
- Fractional part: bits 7-0
- Examples:
  - 1.0 = 0x0100
  - 0.5 = 0x0080
  - -1.0 = 0xFF00
  - 0.707 ≈ 0x00B5

#### Matrix Mode Registers

- **MATRIX_CONTROL** (0x8018): Enable and mirroring
  - Bit 0: Enable Matrix Mode (1=enabled, 0=normal BG0)
  - Bit 1: Mirror horizontally
  - Bit 2: Mirror vertically

- **MATRIX_A/B/C/D** (0x8019-0x8020): Transformation matrix coefficients
  - Each is 16-bit (low byte, then high byte)
  - 8.8 fixed point format

- **MATRIX_CENTER_X/Y** (0x8027-0x802A): Center point
  - 16-bit values (low byte, then high byte)

#### Common Matrix Values

- **Identity (no transform)**: A=0x0100, B=0x0000, C=0x0000, D=0x0100
- **90° rotation**: A=0x0000, B=0xFF00, C=0x0100, D=0x0000
- **2× zoom**: A=0x0200, B=0x0000, C=0x0000, D=0x0200
- **0.5× zoom**: A=0x0080, B=0x0000, C=0x0000, D=0x0080
- **45° rotation**: A=0x00B5, B=0xFF4B, C=0x00B5, D=0x00B5

#### Vertical Sprites in Matrix Mode

Matrix Mode supports **vertical sprites** - sprites that are rendered in 3D space and scale/position based on the transformation matrix. This enables pseudo-3D worlds with buildings, people, and objects that appear to have depth.

**Vertical Sprite Rendering:**

1. **Sprite World Coordinates**: Sprites have world X/Y coordinates (in tilemap space)
2. **3D Transformation**: Transform sprite world coordinates to screen coordinates using inverse matrix
3. **Scaling**: Sprites scale based on distance from camera (Z coordinate or Y position in world)
4. **Depth Sorting**: Sprites are sorted by depth (Y coordinate in world space)

**Sprite Attributes for Matrix Mode:**

- **World X/Y**: Sprite position in world/tilemap coordinates (16-bit each)
- **Base Size**: Sprite size at distance 0 (8×8 or 16×16)
- **Scale Factor**: Calculated from world Y position (farther = smaller)
- **Screen Position**: Calculated from world X/Y using inverse matrix transformation

**Rendering Pipeline for Vertical Sprites:**

1. For each sprite:
   - Read sprite world coordinates (X, Y in tilemap space)
   - Transform to screen coordinates: `screen_x = f(world_x, world_y)`, `screen_y = f(world_x, world_y)`
   - Calculate scale: `scale = base_scale × (reference_distance / world_y)`
   - Render sprite at screen position with calculated scale
   - Apply depth sorting (render back-to-front)

**Use Cases:**
- **Buildings**: Large sprites that scale based on distance
- **People/Characters**: Smaller sprites that move in 3D space
- **Objects**: Items, vehicles, decorations in 3D world
- **Parallax Layers**: Multiple sprite layers at different depths

**Implementation Notes:**
- Sprites can use extended OAM entries for world coordinates
- Scale calculation: `sprite_scale = (camera_height / sprite_world_y) × base_scale`
- Clipping: Sprites outside screen bounds can be culled
- Performance: Limit active vertical sprites for performance (recommend 32-64 max)

### Sprites

#### Sprite System

- **Max Sprites**: 128 sprites
- **Size**: 8×8 or 16×16 pixels (per sprite)
- **Attributes**: X/Y position, tile index, palette, priority, flip X/Y, blend mode, alpha
- **Vertical Sprites**: Special sprites for Matrix Mode that render in 3D space with scaling

#### Vertical Sprites (Matrix Mode)

Vertical sprites are sprites that exist in world space and are rendered with 3D perspective effects when Matrix Mode is enabled. They enable pseudo-3D worlds with buildings, people, and objects.

**Vertical Sprite Properties:**
- **World Coordinates**: X/Y position in tilemap/world space (16-bit signed)
- **Base Size**: Sprite size at reference distance (8×8 or 16×16)
- **Scaling**: Automatic scaling based on distance from camera
- **Depth Sorting**: Rendered back-to-front for correct occlusion

**Vertical Sprite Rendering:**
1. Transform world coordinates to screen coordinates using inverse matrix
2. Calculate scale factor: `scale = (camera_height / world_y) × base_scale`
3. Render sprite at screen position with calculated scale
4. Apply depth sorting (sprites with larger world_y render first)

**Use Cases:**
- **Buildings**: Large structures that scale with distance
- **People/Characters**: Animated sprites moving in 3D space
- **Objects**: Items, vehicles, decorations in pseudo-3D world
- **Parallax Layers**: Multiple sprite layers at different depths for depth effect

#### Sprite Attributes (OAM Entry)

Each sprite requires multiple bytes in OAM:

**OAM Entry Format:**
- **Byte 0**: X position (low byte, signed)
- **Byte 1**: X position (high byte, bit 0 only, sign extends)
- **Byte 2**: Y position (8-bit, 0-255)
- **Byte 3**: Tile index (8-bit)
- **Byte 4**: Attributes
  - Bits [3:0]: Palette index (0-15)
  - Bit 4: Flip X
  - Bit 5: Flip Y
  - Bits [7:6]: Priority (0-3)
- **Byte 5**: Control
  - Bit 0: Enable
  - Bit 1: Size (0=8×8, 1=16×16)
  - Bits [3:2]: Blend mode (0=normal, 1=alpha, 2=additive, 3=subtractive)
  - Bits [7:4]: Alpha value (0-15, 0=transparent, 15=opaque)

#### Sprite Priority

- **Priority Levels**: 0 (lowest, behind all BGs) to 3 (highest, in front of all BGs)
- **Rendering Order**: Sprites sorted by priority, then by index (lower index = higher priority if same priority level)

#### Sprite Blending Modes

- **0 (Normal)**: Opaque, no blending
- **1 (Alpha)**: Alpha blending with background
- **2 (Additive)**: Add sprite color to background
- **3 (Subtractive)**: Subtract sprite color from background

### Windowing System

SNES-style windowing with 2 windows and per-layer control.

#### Window Properties

- **2 Windows**: Window 0 and Window 1
- **Window Coordinates**: Left, Right, Top, Bottom (8-bit each, 0-319/0-199)
- **Window Logic**: OR, AND, XOR, XNOR (configurable)
- **Per-Layer Control**: Each layer and sprites can be individually windowed

#### Window Registers

- **WINDOW0_LEFT/RIGHT/TOP/BOTTOM** (0x802B-0x802E): Window 0 coordinates
- **WINDOW1_LEFT/RIGHT/TOP/BOTTOM** (0x802F-0x8032): Window 1 coordinates
- **WINDOW_CONTROL** (0x8033): Window enable and logic
  - Bit 0: Window 0 enable
  - Bit 1: Window 1 enable
  - Bits [3:2]: Logic mode (0=OR, 1=AND, 2=XOR, 3=XNOR)
- **WINDOW_MAIN_ENABLE** (0x8034): Per-layer window enable
  - Bit 0: BG0 window enable
  - Bit 1: BG1 window enable
  - Bit 2: BG2 window enable
  - Bit 3: BG3 window enable
  - Bit 4: Sprites window enable

#### Window Logic

For each pixel:
1. Check if pixel is inside Window 0
2. Check if pixel is inside Window 1
3. Apply logic:
   - **OR**: Pixel is inside if in Window 0 OR Window 1
   - **AND**: Pixel is inside if in Window 0 AND Window 1
   - **XOR**: Pixel is inside if in Window 0 XOR Window 1
   - **XNOR**: Pixel is inside if NOT (Window 0 XOR Window 1)
4. If pixel is inside window and layer has window enabled, render pixel
5. If pixel is outside window, skip pixel (transparent)

### HDMA (Per-Scanline Scroll)

HDMA enables per-scanline scroll changes for parallax and perspective effects.

#### HDMA System

- **HDMA Table**: Stored in WRAM, contains scroll values for each scanline
- **Per-Scanline Control**: Each scanline can have different scroll values
- **Layer Control**: Each layer can independently use HDMA

#### HDMA Registers

- **HDMA_CONTROL** (0x8036): Enable and layer control
  - Bit 0: HDMA enable
  - Bit 1: BG0 HDMA enable
  - Bit 2: BG1 HDMA enable
  - Bit 3: BG2 HDMA enable
  - Bit 4: BG3 HDMA enable

- **HDMA_TABLE_BASE** (0x8037-0x8038): Base address of HDMA table in WRAM (16-bit)

- **HDMA_SCANLINE** (0x8039): Current scanline (0-199)

- **HDMA_BGx_SCROLLX/Y** (0x803A+): Scroll values for current scanline

#### HDMA Table Format

HDMA table in WRAM contains scroll values for each scanline:

```
For each scanline (0-199):
  - 2 bytes: BG0 scroll X (if BG0 HDMA enabled)
  - 2 bytes: BG0 scroll Y (if BG0 HDMA enabled)
  - 2 bytes: BG1 scroll X (if BG1 HDMA enabled)
  - ... (similar for BG2, BG3)
```

**HDMA Processing:**
1. At start of each scanline, read HDMA table entry for current scanline
2. Update layer scroll registers with HDMA values
3. Render scanline with updated scroll values

### VRAM Layout

**VRAM (64KB) Organization:**
- **Tile Data**: 4bpp tile patterns
  - 8×8 tiles: 32 bytes each
  - 16×16 tiles: 128 bytes each
  - Can store up to 2048 8×8 tiles or 512 16×16 tiles
- **Tilemap**: Tile indices and attributes
  - 32×32 tilemap = 1024 entries = 2048 bytes per tilemap
  - Can store multiple tilemaps
- **OAM**: Sprite attribute table
  - 128 sprites × 6 bytes = 768 bytes

**Typical VRAM Layout:**
- 0x0000-0x3FFF: Tile data (16KB, ~512 8×8 tiles)
- 0x4000-0x7FFF: Tilemap 0 (8KB, 4 tilemaps)
- 0x8000-0x8FFF: Tilemap 1 (4KB, 2 tilemaps)
- 0x9000-0x92FF: OAM (768 bytes, 128 sprites)
- 0x9300-0xFFFF: Free space

### Rendering Pipeline

**Frame Rendering (per scanline):**
1. **HDMA Update**: Update scroll values from HDMA table (if enabled)
2. **Background Rendering**: Render BG3 → BG2 → BG1 → BG0 (back to front)
   - For each layer:
     - If Matrix Mode enabled (BG0):
       - Transform screen coordinates to world coordinates using inverse matrix
       - For large worlds: Calculate which tilemap region contains coordinates
       - Seamlessly stitch tiles from adjacent tilemap regions if needed
       - Read tilemap entries from calculated world coordinates
     - Else (normal mode):
       - Calculate visible tile range based on scroll
       - Read tilemap entries
     - Render tiles with windowing applied
3. **Sprite Rendering**: 
   - **Normal Sprites**: Render sprites (sorted by priority)
     - Sort sprites by priority and index
     - Render each sprite with blending applied
   - **Vertical Sprites** (if Matrix Mode enabled):
     - Transform sprite world coordinates to screen coordinates
     - Calculate sprite scale based on world Y position
     - Depth sort sprites (back-to-front by world Y)
     - Render scaled sprites at calculated screen positions
4. **Output**: Composite all layers to output buffer

**Pixel Rendering:**
1. Check windowing (if enabled for layer)
2. Read tilemap entry for pixel position
3. Read tile data for pixel
4. Look up palette color (CGRAM)
5. Apply blending (if sprite)
6. Write to output buffer

---

## APU (Audio System)

### Audio Overview

Nitro-Core-DX features a 4-channel audio synthesizer with multiple waveform types.

### Audio Channels

**4 Independent Channels:**
- **Channels 0-2**: Sine, Square, or Saw waveform
- **Channel 3**: Square or Noise waveform

### Channel Registers

Each channel has 4 registers:

| Register | Description |
|----------|-------------|
| FREQ_LOW | Frequency low byte (16-bit frequency) |
| FREQ_HIGH | Frequency high byte |
| VOLUME | Volume (0-255) |
| CONTROL | Control: bit 0=enable, bits 1-2=waveform (channels 0-2), bit 1=noise mode (channel 3) |

### Waveform Types

**0 (Sine)**: Smooth sine wave
- `sample = sin(phase) × volume`

**1 (Square)**: Square wave (50% duty cycle)
- `sample = (phase < π) ? volume : -volume`

**2 (Saw)**: Sawtooth wave
- `sample = ((phase / 2π) × 2 - 1) × volume`

**3 (Noise)**: White noise (LFSR-based, channel 3 only)
- 15-bit LFSR (Linear Feedback Shift Register)
- Polynomial: x^15 + x^14 + 1
- Output: MSB of LFSR

### Frequency Encoding

**Frequency Value:**
- 16-bit value: `FREQ_LOW | (FREQ_HIGH << 8)`
- Frequency in Hz (direct value)
- Range: 0-65535 Hz

**Phase Increment:**
- Calculated per sample: `phase_increment = (frequency / 44100) × 2π`
- Phase accumulator: `phase += phase_increment` (wraps at 2π)

### Audio Generation

**Per Sample:**
1. For each channel:
   - If enabled, generate sample from waveform
   - Apply volume
   - Add to output buffer
2. Apply master volume
3. Clamp to valid range (-1.0 to 1.0)
4. Convert to output format (16-bit PCM)

**Sample Rate**: 44,100 Hz (CD quality)

### Master Volume

- **MASTER_VOLUME** (0x9010): Master volume control (0-255)
- Applied to all channels after individual channel volume
- Final volume = `channel_volume × master_volume / 255`

---

## Input System

### Controller

SNES-style 12-button controller support.

### Button Mapping

| Button | Bit | Value |
|--------|-----|-------|
| UP | 0 | 0x01 |
| DOWN | 1 | 0x02 |
| LEFT | 2 | 0x04 |
| RIGHT | 3 | 0x08 |
| A | 4 | 0x10 |
| B | 5 | 0x20 |
| X | 6 | 0x40 |
| Y | 7 | 0x80 |
| L | 8 | 0x100 |
| R | 9 | 0x200 |
| START | 10 | 0x400 |
| SELECT | 11 | 0x800 |

### Input Reading

#### Controller 1
1. **Latch Controller**: Write 1 to CONTROLLER1_LATCH (0xA001)
2. **Read Button State**: Read CONTROLLER1 (0xA000) - low byte, then 0xA001 - high byte
3. **Release Latch**: Write 0 to CONTROLLER1_LATCH

#### Controller 2
1. **Latch Controller**: Write 1 to CONTROLLER2_LATCH (0xA003)
2. **Read Button State**: Read CONTROLLER2 (0xA002) - low byte, then 0xA003 - high byte
3. **Release Latch**: Write 0 to CONTROLLER2_LATCH

**Button State Format:**
- Low byte (bits 0-7): UP, DOWN, LEFT, RIGHT, A, B, X, Y
- High byte (bits 8-15): L, R, START, SELECT, reserved

---

## ROM Format

### ROM Header

**Header Size**: 32 bytes

| Offset | Size | Field | Description |
|--------|------|-------|-------------|
| 0x00 | 4 | Magic | `0x46434D52` ("RMCF" - Fantasy Console ROM) |
| 0x04 | 2 | Version | ROM format version (currently 1) |
| 0x06 | 4 | ROM Size | Size of code data in bytes (little-endian) |
| 0x0A | 2 | Entry Bank | Entry point bank (typically 1) |
| 0x0C | 2 | Entry Offset | Entry point offset (typically 0x8000) |
| 0x0E | 2 | Mapper Flags | Mapper type and flags (bits 0-3: mapper type) |
| 0x10 | 4 | Checksum | Optional checksum (currently unused) |
| 0x14 | 16 | Reserved | Reserved for future use |

### ROM Structure

```
[32-byte Header]
[Code Data...]
```

### Entry Point

The CPU starts execution at:
- **Bank**: `entry_point_bank` (from header, typically 1)
- **Offset**: `entry_point_offset` (from header, typically 0x8000)

### Mapper Types

- **0 (LoROM)**: SNES-like LoROM mapping
  - ROM appears at offset 0x8000+ in each bank
  - Banks 1-125 contain ROM data

### ROM Loading

1. Read 32-byte header
2. Validate magic number (0x46434D52)
3. Validate version (must be ≤ supported version)
4. Read ROM data (size from header)
5. Load ROM into memory mapper
6. Set CPU entry point to header values

---

## Emulator Implementation Guide

### Core Components

An emulator for Nitro-Core-DX requires the following components:

1. **CPU Emulator**
   - Instruction fetch, decode, execute
   - Register management
   - Flag handling
   - Interrupt handling
   - Cycle counting

2. **Memory System**
   - Banked memory access
   - WRAM, Extended WRAM, ROM mapping
   - I/O register routing

3. **PPU (Graphics)**
   - VRAM, CGRAM, OAM management
   - Background layer rendering
   - Sprite rendering
   - Matrix Mode transformation
   - Windowing system
   - HDMA processing

4. **APU (Audio)**
   - Channel management
   - Waveform generation
   - Audio mixing
   - Sample output

5. **Input System**
   - Controller state management
   - Latch handling

6. **ROM Loader**
   - Header parsing
   - ROM data loading
   - Entry point setup

### Implementation Steps

#### Step 1: CPU Core

1. **Define CPU State Structure:**
   ```go
   type CPUState struct {
       R0, R1, R2, R3, R4, R5, R6, R7 uint16
       PCBank uint8
       PCOffset uint16
       PBR, DBR uint8
       SP uint16
       Flags uint8  // Z, N, C, V, I
       Cycles uint32
       InterruptMask uint8
       InterruptPending uint8
   }
   ```

   **Note**: This example uses Go syntax, but can be adapted to C/C++ or Rust.

2. **Implement Instruction Fetch:**
   - Read from [PBR:PC]
   - Increment PC
   - Handle immediate values

3. **Implement Instruction Decode:**
   - Extract opcode family, mode, registers
   - Map to instruction type

4. **Implement Instruction Execution:**
   - Handle each instruction type
   - Update flags
   - Handle memory access
   - Handle branches/jumps

5. **Implement Cycle Counting:**
   - Track cycles per instruction
   - Run until target cycles reached

#### Step 2: Memory System

1. **Define Memory State:**
   ```go
   type MemorySystem struct {
       WRAM [32768]uint8
       WRAMExtended [131072]uint8
       ROMData []uint8
       ROMSize uint32
       ROMBanks uint8
   }
   ```

2. **Implement Memory Access:**
   - `Read8(bank, offset uint16) uint8`
   - `Write8(bank, offset uint16, value uint8)`
   - `Read16(bank, offset uint16) uint16`
   - `Write16(bank, offset uint16, value uint16)`

3. **Implement Bank Mapping:**
   - Bank 0: WRAM (0x0000-0x7FFF) + I/O (0x8000+)
   - Banks 1-125: ROM (0x8000+)
   - Banks 126-127: Extended WRAM

4. **Implement I/O Routing:**
   - Route 0x8000+ to PPU/APU/Input handlers
   - Always use bank 0 for I/O

#### Step 3: PPU Implementation

1. **Define PPU State:**
   ```go
   type PPUState struct {
       VRAM [65536]uint8
       CGRAM [512]uint8
       OAM [768]uint8  // 128 sprites × 6 bytes
       
       BG0, BG1, BG2, BG3 TileLayer
       
       MatrixEnabled bool
       MatrixA, MatrixB, MatrixC, MatrixD int16  // 8.8 fixed point
       MatrixCenterX, MatrixCenterY int16
       
       Window0, Window1 Window
       WindowControl uint8
       
       HDMAEnabled bool
       HDMATableBase uint16
       HDMAScrollX [4][200]int16  // Per scanline scroll
       HDMAScrollY [4][200]int16
       
       OutputBuffer [320 * 200]uint32  // RGB output
       
       // Matrix Mode enhancements
       LargeWorldEnabled bool
       WorldTilemapBase uint16  // Base address for large world tilemap
       WorldTilemapWidth uint16  // World width in tiles
       WorldTilemapHeight uint16 // World height in tiles
       
       // Vertical sprites
       VerticalSpritesEnabled bool
       VerticalSpriteCount uint8
       VerticalSpriteWorldX [128]int16  // World X coordinates
       VerticalSpriteWorldY [128]int16  // World Y coordinates
   }
   ```

2. **Implement Register Writes:**
   - Handle scroll, control, VRAM, CGRAM, OAM writes
   - Update PPU state accordingly

3. **Implement Rendering:**
   - Background layer rendering (BG3 → BG0)
   - Sprite rendering (sorted by priority)
   - Matrix Mode transformation
   - **Large world tilemap rendering** (tile stitching for seamless worlds)
   - **Vertical sprite rendering** (3D sprites with scaling)
   - Windowing application
   - HDMA per-scanline updates

4. **Implement Matrix Mode Enhancements:**
   - **Large World Support**: Handle tilemaps larger than 32×32 tiles
     - Calculate which tilemap region contains transformed coordinates
     - Seamlessly render tiles from adjacent tilemap regions
     - Support for world coordinates beyond single tilemap boundaries
   - **Vertical Sprite Rendering**:
     - Transform sprite world coordinates to screen coordinates using inverse matrix
     - Calculate sprite scale based on world Y position (distance)
     - Depth sort sprites (render back-to-front)
     - Render scaled sprites at calculated screen positions

4. **Implement Tile Rendering:**
   - Read tilemap entry
   - Read tile data
   - Look up palette color
   - Apply transformations (flip, palette)

#### Step 4: APU Implementation

1. **Define APU State:**
   ```go
   type APUState struct {
       Channels [4]AudioChannel
       MasterVolume uint8
       SampleBuffer []float32
   }
   
   type AudioChannel struct {
       Frequency uint16
       Volume uint8
       Enabled bool
       Waveform uint8
       Phase float64
       PhaseIncrement float64
       NoiseLFSR uint16
   }
   ```

2. **Implement Waveform Generation:**
   - Sine: `math.Sin(phase)`
   - Square: `phase < π ? 1.0 : -1.0`
   - Saw: `(phase / 2π) * 2 - 1`
   - Noise: LFSR update

3. **Implement Audio Mixing:**
   - Generate sample for each channel
   - Mix channels together
   - Apply master volume
   - Output to audio buffer

#### Step 5: Input System

1. **Define Input State:**
   ```go
   type InputState struct {
       Controller1Buttons uint16
       Controller2Buttons uint16
       LatchActive bool
       Controller2LatchActive bool
   }
   ```

2. **Implement Input Reading:**
   - Handle latch write
   - Return button state on read

#### Step 6: ROM Loading

1. **Parse ROM Header:**
   ```go
   type ROMHeader struct {
       Magic uint32
       Version uint16
       ROMSize uint32
       EntryBank uint16
       EntryOffset uint16
       MapperFlags uint16
       Checksum uint32
       Reserved [16]uint8
   }
   ```

2. **Load ROM Data:**
   - Read header (32 bytes)
   - Validate magic and version
   - Read ROM data
   - Set up memory mapper
   - Set CPU entry point

#### Step 7: Development Toolkit

1. **Hex Editor / Memory Viewer:**
   ```go
   type MemoryViewer struct {
       Regions []MemoryRegion
       CurrentAddress uint32
       SearchPattern []uint8
   }
   
   func ViewMemory(region MemoryRegion, address uint32) []uint8
   func EditMemory(region MemoryRegion, address uint32, value uint8)
   func SearchMemory(region MemoryRegion, pattern []uint8) []uint32
   ```

2. **Component Logging:**
   ```go
   type Logger struct {
       CPUEnabled bool
       MemoryEnabled bool
       PPUEnabled bool
       APUEnabled bool
       InputEnabled bool
       LogLevel LogLevel
   }
   
   func LogCPU(instruction uint16, regs CPUState)
   func LogMemory(accessType string, bank uint8, offset uint16, value uint8)
   func LogPPU(register uint16, value uint8)
   ```

3. **Debugger Interface:**
   ```go
   type Debugger struct {
       Breakpoints []Breakpoint
       Watchpoints []Watchpoint
       Stepping bool
   }
   
   func SetBreakpoint(address uint32)
   func StepInstruction()
   func GetRegisterState() CPUState
   ```

#### Step 8: Main Emulation Loop

```go
func RunFrame() {
    targetCycles := cpu.Cycles + 166667  // 10 MHz @ 60 FPS
    
    // Run CPU
    for cpu.Cycles < targetCycles {
        // Check breakpoints
        if debugger.IsBreakpoint(cpu.PCBank, cpu.PCOffset) {
            debugger.Pause()
            return
        }
        
        instruction := FetchInstruction()
        
        // Log CPU execution
        if logger.CPUEnabled {
            logger.LogCPU(instruction, cpu)
        }
        
        ExecuteInstruction(instruction)
        
        // Check interrupts
        if interruptPending && !interruptMasked {
            HandleInterrupt()
        }
        
        // Step mode
        if debugger.Stepping {
            debugger.Pause()
            return
        }
    }
    
    // Render frame
    PPURenderFrame()
    
    // Generate audio
    APUGenerateSamples()
    
    // Update input
    UpdateInput()
}
```

### Performance Considerations

**Optimization Strategies:**
1. **Hot Paths**: Optimize CPU instruction execution loop
2. **Memory Access**: Cache frequently accessed memory regions
3. **Rendering**: Use efficient tile rendering algorithms
4. **Matrix Mode**: Optimize matrix calculations, use SIMD if available
5. **Large Worlds**: Use spatial partitioning for tilemap lookups
6. **Vertical Sprites**: Limit active sprites, use efficient depth sorting
7. **Audio**: Pre-calculate phase increments, use lookup tables for waveforms
8. **Branch Prediction**: Use branch prediction for conditional branches
9. **Logging**: Make logging zero-cost when disabled (compile-time or runtime checks)

**Target Performance:**
- 60 FPS emulation
- CPU: 166,667 cycles per frame
- PPU: 320×200 = 64,000 pixels per frame
- Matrix Mode: Efficient matrix math (consider SIMD/vectorization)
- Large Worlds: Optimized tilemap lookups and tile stitching
- Vertical Sprites: Efficient 3D transformation and depth sorting
- Audio: 44,100 samples per second
- Debugging: Minimal overhead when logging disabled (< 1% performance impact)

### Testing

**Test ROMs:**
1. **CPU Test ROM**: Test all instructions, flags, branches
2. **PPU Test ROM**: Test backgrounds, sprites, Matrix Mode
3. **APU Test ROM**: Test all waveforms, channels
4. **Input Test ROM**: Test controller reading

**Validation:**
- Compare emulator output with reference implementation
- Test edge cases (overflow, underflow, boundary conditions)
- Performance profiling

---

## Reference Tables

### Instruction Opcode Quick Reference

| Instruction | Opcode | Mode | Cycles | Description |
|------------|--------|------|--------|-------------|
| NOP | 0x0000 | - | 1 | No operation |
| MOV R1, R2 | 0x1000 | 0 | 1 | Register to register |
| MOV R1, #imm | 0x1000 | 1 | 2 | Immediate (2 words) |
| MOV R1, [R2] | 0x1000 | 2 | 2 | Load from memory |
| MOV [R1], R2 | 0x1000 | 3 | 2 | Store to memory |
| PUSH R1 | 0x1000 | 4 | 2 | Push to stack |
| POP R1 | 0x1000 | 5 | 2 | Pop from stack |
| ADD R1, R2 | 0x2000 | 0 | 1 | Add registers |
| ADD R1, #imm | 0x2000 | 1 | 2 | Add immediate |
| SUB R1, R2 | 0x3000 | 0 | 1 | Subtract registers |
| SUB R1, #imm | 0x3000 | 1 | 2 | Subtract immediate |
| MUL R1, R2 | 0x4000 | 0 | 2 | Multiply |
| DIV R1, R2 | 0x5000 | 0 | 4 | Divide |
| AND R1, R2 | 0x6000 | 0 | 1 | Bitwise AND |
| OR R1, R2 | 0x7000 | 0 | 1 | Bitwise OR |
| XOR R1, R2 | 0x8000 | 0 | 1 | Bitwise XOR |
| NOT R1 | 0x9000 | 0 | 1 | Bitwise NOT |
| SHL R1, R2 | 0xA000 | 0 | 1 | Shift left |
| SHR R1, R2 | 0xB000 | 0 | 1 | Shift right |
| CMP R1, R2 | 0xC000 | 0 | 1 | Compare |
| BEQ offset | 0xC100 | 1 | 2/3 | Branch if equal |
| BNE offset | 0xC200 | 1 | 2/3 | Branch if not equal |
| BGT offset | 0xC300 | 1 | 2/3 | Branch if greater |
| BLT offset | 0xC400 | 1 | 2/3 | Branch if less |
| BGE offset | 0xC500 | 1 | 2/3 | Branch if >= |
| BLE offset | 0xC600 | 1 | 2/3 | Branch if <= |
| JMP offset | 0xD000 | 1 | 2 | Jump |
| CALL offset | 0xE000 | 1 | 3 | Call subroutine |
| RET | 0xF000 | 0 | 2 | Return |

### PPU Register Quick Reference

| Register | Address | Description |
|----------|---------|-------------|
| BG0_SCROLLX | 0x8000-0x8001 | BG0 scroll X (16-bit) |
| BG0_SCROLLY | 0x8002-0x8003 | BG0 scroll Y (16-bit) |
| BG0_CONTROL | 0x8008 | BG0 enable/tile size |
| VRAM_ADDR | 0x800E-0x800F | VRAM address (16-bit) |
| VRAM_DATA | 0x8010 | VRAM data (auto-increment) |
| CGRAM_ADDR | 0x8012 | Palette address |
| CGRAM_DATA | 0x8013 | Palette data (16-bit RGB555) |
| OAM_ADDR | 0x8014 | Sprite address |
| OAM_DATA | 0x8015 | Sprite data |
| MATRIX_CONTROL | 0x8018 | Matrix Mode enable |
| MATRIX_A | 0x8019-0x801A | Matrix A (16-bit) |
| MATRIX_B | 0x801B-0x801C | Matrix B (16-bit) |
| MATRIX_C | 0x801D-0x801E | Matrix C (16-bit) |
| MATRIX_D | 0x801F-0x8020 | Matrix D (16-bit) |
| WINDOW0_LEFT | 0x802B | Window 0 left |
| WINDOW0_RIGHT | 0x802C | Window 0 right |
| WINDOW0_TOP | 0x802D | Window 0 top |
| WINDOW0_BOTTOM | 0x802E | Window 0 bottom |
| WINDOW_CONTROL | 0x8033 | Window enable/logic |
| HDMA_CONTROL | 0x8036 | HDMA enable |

### APU Register Quick Reference

| Register | Address | Description |
|----------|---------|-------------|
| CH0_FREQ_LOW | 0x9000 | Channel 0 frequency (low) |
| CH0_FREQ_HIGH | 0x9001 | Channel 0 frequency (high) |
| CH0_VOLUME | 0x9002 | Channel 0 volume |
| CH0_CONTROL | 0x9003 | Channel 0 control |
| CH1_* | 0x9004-0x9007 | Channel 1 |
| CH2_* | 0x9008-0x900B | Channel 2 |
| CH3_* | 0x900C-0x900F | Channel 3 |
| MASTER_VOLUME | 0x9010 | Master volume |

### Memory Map Quick Reference

| Bank | Address Range | Description |
|------|---------------|-------------|
| 0 | 0x0000-0x7FFF | Work RAM (32KB) |
| 0 | 0x8000-0xFFFF | I/O Registers |
| 1-125 | 0x8000-0xFFFF | ROM (per bank) |
| 126-127 | 0x0000-0xFFFF | Extended WRAM |

### Flag Bits Quick Reference

| Bit | Flag | Set When | Used By |
|-----|------|----------|---------|
| 0 | Z | Result = 0 | BEQ, BNE |
| 1 | N | Result bit 15 = 1 | BLT, BGE |
| 2 | C | Unsigned overflow | - |
| 3 | V | Signed overflow | - |
| 4 | I | Interrupt mask | Interrupt handling |

---

## Appendix: Programming Examples

### Example 1: Enable Background and Display Tile

```assembly
; Enable BG0
MOV R0, #0x8008        ; BG0_CONTROL address
MOV R1, #0x01          ; Enable BG0, 8x8 tiles
MOV [R0], R1           ; Write control

; Set palette color 1 to white
MOV R0, #0x8012        ; CGRAM_ADDR
MOV R1, #0x02          ; Palette 0, color 1 = index 2
MOV [R0], R1
MOV R0, #0x8013        ; CGRAM_DATA
MOV R1, #0xFF          ; White low byte
MOV [R0], R1
MOV R1, #0x7F          ; White high byte
MOV [R0], R1

; Create tile 1 (solid white)
MOV R0, #0x800E        ; VRAM_ADDR_L
MOV R1, #0x20          ; Tile 1 = offset 32
MOV [R0], R1
MOV R0, #0x800F        ; VRAM_ADDR_H
MOV R1, #0x00
MOV [R0], R1
MOV R0, #0x8010        ; VRAM_DATA
MOV R1, #0x11          ; Both pixels = color 1
; Write 32 bytes (loop or unroll)
MOV [R0], R1           ; (repeat 32 times)

; Write tilemap entry (tile 1 at position 20, 12)
MOV R0, #0x800E        ; VRAM_ADDR_L
MOV R1, #0x28          ; Tilemap address = (12*32+20)*2 = 0x0328
MOV [R0], R1
MOV R0, #0x800F        ; VRAM_ADDR_H
MOV R1, #0x03
MOV [R0], R1
MOV R0, #0x8010        ; VRAM_DATA
MOV R1, #0x01          ; Tile index 1
MOV [R0], R1
MOV R1, #0x10          ; Palette 1 << 4
MOV [R0], R1
```

### Example 2: Simple Game Loop

```assembly
; Initialize
MOV R0, #0x0000        ; X position
MOV R1, #0x0000        ; Y position

game_loop:
    ; Read input
    MOV R2, #0xA001     ; CONTROLLER1_LATCH
    MOV R3, #0x01
    MOV [R2], R3        ; Latch
    MOV R2, #0xA000     ; CONTROLLER1
    MOV R3, [R2]        ; Read buttons
    
    ; Check UP button
    MOV R4, #0x01       ; UP mask
    AND R4, R3          ; R4 = buttons & UP
    CMP R4, #0x01
    BNE check_down
    SUB R1, #0x0001     ; Move up
    
check_down:
    MOV R4, #0x02       ; DOWN mask
    AND R4, R3
    CMP R4, #0x02
    BNE update_scroll
    ADD R1, #0x0001     ; Move down
    
update_scroll:
    ; Write scroll to PPU
    MOV R2, #0x8000     ; BG0_SCROLLX_L
    MOV [R2], R0        ; Write X
    MOV R2, #0x8002     ; BG0_SCROLLY_L
    MOV [R2], R1        ; Write Y
    
    ; Delay
    MOV R5, #0x0000
delay_loop:
    ADD R5, #0x0001
    CMP R5, #0x0100
    BNE delay_loop
    
    JMP game_loop       ; Loop forever
```

### Example 3: Play Sound

```assembly
; Play 440 Hz sine wave on channel 0
MOV R0, #0x9000         ; CH0_FREQ_LOW
MOV R1, #0xB4           ; 440 Hz low byte
MOV [R0], R1
MOV R0, #0x9001         ; CH0_FREQ_HIGH
MOV R1, #0x01           ; 440 Hz high byte
MOV [R0], R1

MOV R0, #0x9002         ; CH0_VOLUME
MOV R1, #0x80           ; Volume = 128 (50%)
MOV [R0], R1

MOV R0, #0x9003         ; CH0_CONTROL
MOV R1, #0x01           ; Enable, sine wave
MOV [R0], R1
```

### Example 4: Matrix Mode with Large World

```assembly
; Enable Matrix Mode for large world map
MOV R0, #0x8008         ; BG0_CONTROL
MOV R1, #0x01           ; Enable BG0
MOV [R0], R1

MOV R0, #0x8018         ; MATRIX_CONTROL
MOV R1, #0x01           ; Enable Matrix Mode
MOV [R0], R1

; Set up transformation matrix for perspective
; (Example: looking down at 45° angle)
MOV R0, #0x8019         ; MATRIX_A_L
MOV R1, #0xB5           ; A = 0.707 (low byte)
MOV [R0], R1
MOV R0, #0x801A         ; MATRIX_A_H
MOV R1, #0x00
MOV [R0], R1

; Set center point
MOV R0, #0x8027         ; MATRIX_CENTER_X_L
MOV R1, #0xA0           ; Center X = 160
MOV [R0], R1
MOV R0, #0x8029         ; MATRIX_CENTER_Y_L
MOV R1, #0x64           ; Center Y = 100
MOV [R0], R1

; Large world: Use multiple tilemaps stitched together
; World coordinates can span multiple tilemap regions
; Emulator handles seamless tile stitching automatically
```

### Example 5: Vertical Sprite (Building in 3D World)

```assembly
; Set up vertical sprite for building
; Sprite world coordinates: X=100, Y=200 (in tilemap space)
; Sprite will scale and position based on Matrix Mode transformation

; Store sprite world coordinates in OAM extended attributes
; (Implementation-specific: may use extended OAM or WRAM)
MOV R0, #0x0000         ; WRAM address for sprite world data
MOV R1, #0x0064         ; World X = 100
MOV [R0], R1
ADD R0, #0x0002
MOV R1, #0x00C8         ; World Y = 200
MOV [R0], R1

; Set sprite base attributes (normal sprite entry)
MOV R0, #0x8014         ; OAM_ADDR
MOV R1, #0x00           ; Sprite 0
MOV [R0], R1
MOV R0, #0x8015         ; OAM_DATA
MOV R1, #0x00           ; X position (will be calculated from world X)
MOV [R0], R1
MOV R1, #0x64           ; Y position (will be calculated from world Y)
MOV [R0], R1
MOV R1, #0x10           ; Tile index for building
MOV [R0], R1
MOV R1, #0x10           ; Palette 1
MOV [R0], R1
MOV R1, #0x81           ; Enable, 16x16 size, vertical sprite flag
MOV [R0], R1

; Emulator will:
; 1. Read world coordinates from WRAM or extended OAM
; 2. Transform to screen coordinates using inverse matrix
; 3. Calculate scale based on world Y
; 4. Render sprite at calculated position with scale
```

---

## Notes

- All addresses are in hexadecimal (0x notation)
- All values are 16-bit unless otherwise specified
- The CPU is little-endian (low byte first)
- ROM code must start at offset 0x8000 in a ROM bank
- Stack grows downward (SP decreases on push)
- Branch offsets are signed 16-bit relative to PC after instruction
- I/O addresses (0x8000+) always use bank 0, regardless of DBR
- Matrix Mode uses 8.8 fixed point (1.0 = 0x0100)
- Audio frequency is in Hz (direct value, not phase increment)

---

**End of Design Document**

This document provides complete specifications for implementing a Nitro-Core-DX emulator. All information is language-agnostic and can be used to build an emulator in any programming language, including Go.

