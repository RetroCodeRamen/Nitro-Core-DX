# Fantasy Console Programming Manual

**Version 1.0**  
**Last Updated: December 2024**

---

## Table of Contents

1. [System Overview](#system-overview)
2. [CPU Architecture](#cpu-architecture)
3. [Instruction Set](#instruction-set)
4. [Memory Map](#memory-map)
5. [PPU (Graphics System)](#ppu-graphics-system)
6. [APU (Audio System)](#apu-audio-system)
7. [Input System](#input-system)
8. [ROM Format](#rom-format)
9. [Programming Examples](#programming-examples)
10. [Reference Tables](#reference-tables)

---

## System Overview

The Fantasy Console is a custom 16-bit system inspired by classic 8/16-bit consoles like the SNES. It features:

- **16-bit CPU** with banked 24-bit addressing
- **320x200 pixel display** (portrait mode: 200x320)
- **Tile-based graphics** with 2 background layers and sprites
- **4-channel audio synthesizer** (sine, square, saw, noise)
- **SNES-like input** with 12 buttons
- **60 FPS** target frame rate
- **SNES-period-accurate timing** (44,667 CPU cycles per frame)

### System Specifications

| Feature | Specification |
|---------|--------------|
| Display Resolution | 320x200 (landscape) / 200x320 (portrait) |
| Color Depth | 256 colors (8-bit indexed) |
| Tile Size | 8x8 or 16x16 pixels |
| Max Sprites | 128 |
| Audio Channels | 4 (sine, square, saw, noise) |
| Audio Sample Rate | 44,100 Hz |
| CPU Speed | 2.68 MHz (SNES-accurate) |
| Memory | 64KB per bank, 256 banks (16MB total) |
| ROM Size | Up to 7.8MB (125 banks × 64KB) |

---

## CPU Architecture

### Registers

The CPU has 8 general-purpose 16-bit registers:

- **R0-R7**: General-purpose registers (16-bit)

**Special Registers:**

- **PC (Program Counter)**: 24-bit logical address (bank:offset)
  - `pc_bank`: Bank number (0-255)
  - `pc_offset`: 16-bit offset within bank (0x0000-0xFFFF)
- **SP (Stack Pointer)**: 16-bit offset in stack bank (starts at 0x1FFF)
- **PBR (Program Bank Register)**: Current program bank
- **DBR (Data Bank Register)**: Current data bank

**Flags Register:**

- **Z (Zero)**: Set when result is zero
- **N (Negative)**: Set when result is negative (bit 15 set)
- **C (Carry)**: Set on unsigned overflow
- **V (Overflow)**: Set on signed overflow
- **I (Interrupt)**: Interrupt mask flag

### Addressing Modes

1. **Register Direct** (Mode 0): `MOV R1, R2`
2. **Immediate** (Mode 1): `MOV R1, #0x1234`
3. **Direct Address** (Mode 2): `MOV R1, [R2]` (load from address in R2)
4. **Indirect** (Mode 3): `MOV [R1], R2` (store to address in R1)
5. **Indexed** (Mode 4): Register + offset (future)

---

## Instruction Set

### Instruction Encoding

Instructions are 16-bit words with the following format:

```
[15:12] = Opcode family (0x0-0xF)
[11:8]  = Mode/subop
[7:4]   = Register 1 (destination)
[3:0]   = Register 2 (source)
```

Some instructions require an additional 16-bit immediate value.

### Arithmetic Instructions

#### ADD - Add
- **Format**: `ADD R1, R2` or `ADD R1, #imm`
- **Opcode**: `0x2000`
- **Description**: Adds R2 (or immediate) to R1, stores result in R1
- **Flags**: Sets Z, N, C, V
- **Example**: `ADD R3, R4` → R3 = R3 + R4

#### SUB - Subtract
- **Format**: `SUB R1, R2` or `SUB R1, #imm`
- **Opcode**: `0x3000`
- **Description**: Subtracts R2 (or immediate) from R1
- **Flags**: Sets Z, N, C, V
- **Example**: `SUB R3, #5` → R3 = R3 - 5

#### MUL - Multiply
- **Format**: `MUL R1, R2` or `MUL R1, #imm`
- **Opcode**: `0x4000`
- **Description**: Multiplies R1 by R2 (or immediate), stores 32-bit result (low 16 bits in R1)
- **Flags**: Sets Z, N
- **Example**: `MUL R3, R4` → R3 = (R3 * R4) & 0xFFFF

#### DIV - Divide
- **Format**: `DIV R1, R2` or `DIV R1, #imm`
- **Opcode**: `0x5000`
- **Description**: Divides R1 by R2 (or immediate), stores quotient in R1
- **Flags**: Sets Z, N
- **Example**: `DIV R3, R4` → R3 = R3 / R4

### Logical Instructions

#### AND - Bitwise AND
- **Format**: `AND R1, R2` or `AND R1, #imm`
- **Opcode**: `0x6000`
- **Description**: Bitwise AND operation
- **Flags**: Sets Z, N
- **Example**: `AND R3, #0xFF` → R3 = R3 & 0xFF

#### OR - Bitwise OR
- **Format**: `OR R1, R2` or `OR R1, #imm`
- **Opcode**: `0x7000`
- **Description**: Bitwise OR operation
- **Flags**: Sets Z, N
- **Example**: `OR R3, #0x80` → R3 = R3 | 0x80

#### XOR - Bitwise XOR
- **Format**: `XOR R1, R2` or `XOR R1, #imm`
- **Opcode**: `0x8000`
- **Description**: Bitwise XOR operation
- **Flags**: Sets Z, N
- **Example**: `XOR R3, R4` → R3 = R3 ^ R4

#### NOT - Bitwise NOT
- **Format**: `NOT R1`
- **Opcode**: `0x9000`
- **Description**: Bitwise NOT (one's complement)
- **Flags**: Sets Z, N
- **Example**: `NOT R3` → R3 = ~R3

### Shift Instructions

#### SHL - Shift Left
- **Format**: `SHL R1, R2` or `SHL R1, #imm`
- **Opcode**: `0xA000`
- **Description**: Logical shift left by R2 (or immediate) bits
- **Flags**: Sets Z, N, C (last bit shifted out)
- **Example**: `SHL R3, #2` → R3 = R3 << 2

#### SHR - Shift Right
- **Format**: `SHR R1, R2` or `SHR R1, #imm`
- **Opcode**: `0xB000`
- **Description**: Logical shift right by R2 (or immediate) bits
- **Flags**: Sets Z, N, C (last bit shifted out)
- **Example**: `SHR R3, #1` → R3 = R3 >> 1

### Data Movement Instructions

#### MOV - Move/Load/Store
- **Opcode**: `0x1000`
- **Modes**:
  - **Mode 0**: `MOV R1, R2` - Register to register
  - **Mode 1**: `MOV R1, #imm` - Immediate to register (next word is immediate)
  - **Mode 2**: `MOV R1, [R2]` - Load from memory at address in R2
  - **Mode 3**: `MOV [R1], R2` - Store to memory at address in R1
  - **Mode 4**: `PUSH R1` - Push register to stack
  - **Mode 5**: `POP R1` - Pop stack to register
- **Example**: `MOV R3, #0x1234` → R3 = 0x1234

### Comparison and Branching

#### CMP - Compare
- **Format**: `CMP R1, R2` or `CMP R1, #imm`
- **Opcode**: `0xC000`
- **Description**: Subtracts R2 (or immediate) from R1, sets flags but doesn't store result
- **Flags**: Sets Z, N, C, V
- **Example**: `CMP R3, #10` → Sets flags based on (R3 - 10)

#### BEQ - Branch if Equal
- **Format**: `BEQ offset` (16-bit signed relative offset)
- **Opcode**: `0xC100`
- **Description**: Branches if Z flag is set
- **Example**: `BEQ label` → Jump if previous comparison was equal

#### BNE - Branch if Not Equal
- **Format**: `BNE offset`
- **Opcode**: `0xC200`
- **Description**: Branches if Z flag is clear
- **Example**: `BNE loop` → Jump if previous comparison was not equal

#### BGT - Branch if Greater Than
- **Format**: `BGT offset`
- **Opcode**: `0xC300`
- **Description**: Branches if Z=0 and N=0 (signed greater than)
- **Example**: `BGT skip` → Jump if R1 > R2 (signed)

#### BLT - Branch if Less Than
- **Format**: `BLT offset`
- **Opcode**: `0xC400`
- **Description**: Branches if N flag is set (signed less than)
- **Example**: `BLT bounce` → Jump if R1 < R2 (signed)

#### BGE - Branch if Greater or Equal
- **Format**: `BGE offset`
- **Opcode**: `0xC500`
- **Description**: Branches if N=0 (signed greater or equal)
- **Example**: `BGE continue` → Jump if R1 >= R2 (signed)

#### BLE - Branch if Less or Equal
- **Format**: `BLE offset`
- **Opcode**: `0xC600`
- **Description**: Branches if Z=1 or N=1 (signed less or equal)
- **Example**: `BLE done` → Jump if R1 <= R2 (signed)

### Jump and Call Instructions

#### JMP - Jump
- **Format**: `JMP rel offset` (16-bit signed relative offset)
- **Opcode**: `0xD000`
- **Description**: Unconditional jump (relative)
- **Example**: `JMP loop` → Jump to label

#### CALL - Subroutine Call
- **Format**: `CALL rel offset`
- **Opcode**: `0xE000`
- **Description**: Calls subroutine, pushes return address to stack
- **Example**: `CALL delay` → Call subroutine

#### RET - Return
- **Format**: `RET`
- **Opcode**: `0xF000`
- **Description**: Returns from subroutine, pops return address from stack
- **Example**: `RET` → Return to caller

### Stack Instructions

#### PUSH - Push to Stack
- **Format**: `PUSH R1` (MOV mode 4)
- **Description**: Pushes register value to stack
- **Example**: `PUSH R3` → Push R3 to stack

#### POP - Pop from Stack
- **Format**: `POP R1` (MOV mode 5)
- **Description**: Pops value from stack to register
- **Example**: `POP R3` → Pop stack to R3

### Other Instructions

#### NOP - No Operation
- **Format**: `NOP`
- **Opcode**: `0x0000`
- **Description**: Does nothing, takes 1 cycle
- **Example**: `NOP` → No operation

---

## Memory Map

### Memory Layout

The system uses a banked memory architecture with 24-bit addressing (bank:offset):

- **Bank 0**: WRAM (Work RAM) - 64KB
  - `0x0000-0x7FFF`: Work RAM (32KB)
  - `0x8000-0xFFFF`: I/O Registers (see below)
- **Banks 1-125**: ROM space (LoROM-like mapping)
  - ROM appears at `0x8000-0xFFFF` in each bank
- **Banks 126-127**: Extended WRAM (128KB)

### I/O Register Map (Bank 0, 0x8000-0xFFFF)

#### PPU Registers (0x8000-0x8FFF)

| Address | Name | Size | Description |
|---------|------|------|-------------|
| 0x8000 | BG0_SCROLLX_L | 8-bit | Background 0 scroll X (low byte) |
| 0x8001 | BG0_SCROLLX_H | 8-bit | Background 0 scroll X (high byte) |
| 0x8002 | BG0_SCROLLY_L | 8-bit | Background 0 scroll Y (low byte) |
| 0x8003 | BG0_SCROLLY_H | 8-bit | Background 0 scroll Y (high byte) |
| 0x8004 | BG1_SCROLLX_L | 8-bit | Background 1 scroll X (low byte) |
| 0x8005 | BG1_SCROLLX_H | 8-bit | Background 1 scroll X (high byte) |
| 0x8006 | BG1_SCROLLY_L | 8-bit | Background 1 scroll Y (low byte) |
| 0x8007 | BG1_SCROLLY_H | 8-bit | Background 1 scroll Y (high byte) |
| 0x8008 | BG0_CONTROL | 8-bit | BG0 control (bit 0=enable, bit 1=tile size) |
| 0x8009 | BG1_CONTROL | 8-bit | BG1 control |
| 0x800A | VRAM_ADDR_L | 8-bit | VRAM address (low byte) |
| 0x800B | VRAM_ADDR_H | 8-bit | VRAM address (high byte) |
| 0x800C | VRAM_DATA | 8-bit | VRAM data (auto-increments address) |
| 0x800E | CGRAM_ADDR | 8-bit | CGRAM (palette) address |
| 0x800F | CGRAM_DATA_L | 8-bit | CGRAM data (low byte, RGB555) |
| 0x8010 | CGRAM_DATA_H | 8-bit | CGRAM data (high byte) |
| 0x8011 | OAM_ADDR | 8-bit | OAM (sprite) address |
| 0x8012 | OAM_DATA | 8-bit | OAM data |
| 0x8013 | FRAMEBUFFER_ENABLE | 8-bit | Framebuffer enable (0=off, 1=on) |
| 0x8013 | DISPLAY_MODE | 8-bit | Display mode (0=landscape, 1=portrait) |
| 0x8014 | MATRIX_CONTROL | 8-bit | Matrix Mode control (bit 0=enable, bit 1=mirror_h, bit 2=mirror_v) |
| 0x8015 | MATRIX_A_L | 8-bit | Transformation matrix A (low byte, 8.8 fixed point) |
| 0x8016 | MATRIX_A_H | 8-bit | Transformation matrix A (high byte) |
| 0x8017 | MATRIX_B_L | 8-bit | Transformation matrix B (low byte, 8.8 fixed point) |
| 0x8018 | MATRIX_B_H | 8-bit | Transformation matrix B (high byte) |
| 0x8019 | MATRIX_C_L | 8-bit | Transformation matrix C (low byte, 8.8 fixed point) |
| 0x801A | MATRIX_C_H | 8-bit | Transformation matrix C (high byte) |
| 0x801B | MATRIX_D_L | 8-bit | Transformation matrix D (low byte, 8.8 fixed point) |
| 0x801C | MATRIX_D_H | 8-bit | Transformation matrix D (high byte) |
| 0x801D | MATRIX_CENTER_X_L | 8-bit | Center point X (low byte) |
| 0x801E | MATRIX_CENTER_X_H | 8-bit | Center point X (high byte) |
| 0x801F | MATRIX_CENTER_Y_L | 8-bit | Center point Y (low byte) |
| 0x8020 | MATRIX_CENTER_Y_H | 8-bit | Center point Y (high byte) |

#### APU Registers (0x9000-0x9FFF)

| Address | Name | Size | Description |
|---------|------|------|-------------|
| 0x9000 | CH0_FREQ_LOW | 8-bit | Channel 0 frequency (low byte) |
| 0x9001 | CH0_FREQ_HIGH | 8-bit | Channel 0 frequency (high byte) |
| 0x9002 | CH0_VOLUME | 8-bit | Channel 0 volume (0-255) |
| 0x9003 | CH0_CONTROL | 8-bit | Channel 0 control (bit 0=enable, bits 2-3=waveform) |
| 0x9004-0x9007 | CH1_* | 8-bit | Channel 1 (same pattern) |
| 0x9008-0x900B | CH2_* | 8-bit | Channel 2 (same pattern) |
| 0x900C-0x900F | CH3_* | 8-bit | Channel 3 (noise/square) |
| 0x9010 | MASTER_VOLUME | 8-bit | Master volume (0-255) |

**Waveform Types:**
- `0` = Sine
- `1` = Square
- `2` = Saw
- `3` = Noise

#### Input Registers (0xA000-0xAFFF)

| Address | Name | Size | Description |
|---------|------|------|-------------|
| 0xA000 | CONTROLLER1 | 8-bit | Controller 1 button state (read) |
| 0xA001 | CONTROLLER1_LATCH | 8-bit | Controller 1 latch (write 1 to latch) |

**Button Bits:**
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

---

## PPU (Graphics System)

### Display

- **Resolution**: 320x200 pixels (landscape) or 200x320 (portrait)
- **Color Depth**: 256 colors (8-bit indexed)
- **Palette**: 256-color CGRAM (RGB555 format)

### Background Layers

Two tile-based background layers (BG0 and BG1):

- **Tile Size**: 8x8 or 16x16 pixels (configurable per layer)
- **Tile Format**: 4bpp (4 bits per pixel, 16 colors per tile)
- **Tilemap**: 64x64 tiles (512x512 pixels for 8x8 tiles)
- **Scrolling**: Independent X/Y scroll per layer

### Matrix Mode (90's Retro-Futuristic Perspective Effects)

Matrix Mode enables advanced perspective and rotation effects on BG0, perfect for creating 3D-style landscapes and racing game tracks. When enabled, BG0 uses a 128x128 tilemap with real-time affine transformation.

**Features:**
- **Rotation**: Rotate the entire background
- **Scaling**: Zoom in/out with perspective
- **Perspective**: Create pseudo-3D "looking down a road" effects
- **Mirroring**: Horizontal and vertical mirroring support

**Transformation Matrix:**
The transformation uses a 2x2 matrix:
```
[x']   [A B]   [x - CX]
[y'] = [C D] × [y - CY]
```

Where:
- `A, B, C, D` are 8.8 fixed-point values (1.0 = 0x0100)
- `CX, CY` is the center point of transformation
- `x, y` are screen coordinates
- `x', y'` are transformed tilemap coordinates

**Matrix Mode Registers:**
- `MATRIX_CONTROL` (0x8014): Enable Matrix Mode and set mirroring flags
  - Bit 0: Enable Matrix Mode (1=enabled, 0=normal BG0)
  - Bit 1: Mirror horizontally
  - Bit 2: Mirror vertically
- `MATRIX_A/B/C/D` (0x8015-0x801C): Transformation matrix coefficients (16-bit, 8.8 fixed point)
- `MATRIX_CENTER_X/Y` (0x801D-0x8020): Center point for transformation (16-bit)

**8.8 Fixed Point Format:**
Values are stored as 16-bit signed integers where:
- Integer part: bits 15-8
- Fractional part: bits 7-0
- Example: 1.0 = 0x0100, 0.5 = 0x0080, -1.0 = 0xFF00

**Setting Up Matrix Mode:**

1. **Enable BG0 and Matrix Mode**:
   ```
   MOV R1, #0x8008        ; BG0_CONTROL
   MOV R2, #0x01          ; Enable BG0
   MOV [R1], R2
   MOV R1, #0x8014        ; MATRIX_CONTROL
   MOV R2, #0x01          ; Enable Matrix Mode
   MOV [R1], R2
   ```

2. **Set Transformation Matrix** (example: 45° rotation):
   ```
   ; Rotation matrix: cos(45°) = sin(45°) ≈ 0.707 = 0x00B5
   MOV R1, #0x8015        ; MATRIX_A_L
   MOV R2, #0xB5          ; A = 0.707 (low byte)
   MOV [R1], R2
   MOV R1, #0x8016        ; MATRIX_A_H
   MOV R2, #0x00          ; A (high byte)
   MOV [R1], R2
   
   MOV R1, #0x8017        ; MATRIX_B_L
   MOV R2, #0x4B          ; B = -0.707 (low byte of -0.707)
   MOV [R1], R2
   MOV R1, #0x8018        ; MATRIX_B_H
   MOV R2, #0xFF          ; B (high byte, negative)
   MOV [R1], R2
   
   ; C = 0.707, D = 0.707 (similar pattern)
   ```

3. **Set Center Point**:
   ```
   MOV R1, #0x801D        ; MATRIX_CENTER_X_L
   MOV R2, #0xA0          ; Center X = 160 (screen center)
   MOV [R1], R2
   MOV R1, #0x801E        ; MATRIX_CENTER_X_H
   MOV R2, #0x00
   MOV [R1], R2
   
   MOV R1, #0x801F        ; MATRIX_CENTER_Y_L
   MOV R2, #0x64          ; Center Y = 100 (screen center)
   MOV [R1], R2
   MOV R1, #0x8020        ; MATRIX_CENTER_Y_H
   MOV R2, #0x00
   MOV [R1], R2
   ```

**Creating Perspective Effects:**
For a "looking down a road" effect, vary the matrix values per scanline:
- Top of screen: Scale down (smaller values)
- Bottom of screen: Scale up (larger values)
- This creates the illusion of depth

**Common Matrix Values:**
- **Identity (no transform)**: A=0x0100, B=0x0000, C=0x0000, D=0x0100
- **90° rotation**: A=0x0000, B=0xFF00, C=0x0100, D=0x0000
- **2x zoom**: A=0x0200, B=0x0000, C=0x0000, D=0x0200
- **0.5x zoom**: A=0x0080, B=0x0000, C=0x0000, D=0x0080

### Sprites

- **Max Sprites**: 128
- **Size**: 8x8 or 16x16 pixels
- **Attributes**: X/Y position, tile index, palette, priority, flip X/Y

### VRAM Layout

VRAM (64KB) is organized as:

- **Tile Data**: 4bpp tile patterns (8 bytes per 8x8 tile)
- **Tilemap**: Tile indices and attributes (2 bytes per tile entry)
- **OAM**: Sprite attribute table

### Setting Up Graphics

1. **Enable Background Layer**:
   ```
   MOV R1, #0x8008        ; BG0_CONTROL address
   MOV R2, #0x01          ; Enable BG0, 8x8 tiles
   MOV [R1], R2           ; Write control register
   ```

2. **Set Scroll Position**:
   ```
   MOV R1, #0x8000        ; BG0_SCROLLX_L
   MOV R2, #0x50          ; Scroll X = 0x0050
   MOV [R1], R2           ; Write low byte
   MOV R1, #0x8001        ; BG0_SCROLLX_H
   MOV R2, #0x00          ; High byte = 0
   MOV [R1], R2           ; Write high byte
   ```

3. **Write to VRAM**:
   ```
   MOV R1, #0x800A        ; VRAM_ADDR_L
   MOV R2, #0x00          ; Address = 0x0000
   MOV [R1], R2           ; Write low byte
   MOV R1, #0x800B        ; VRAM_ADDR_H
   MOV R2, #0x00          ; High byte = 0
   MOV [R1], R2           ; Write high byte
   MOV R1, #0x800C        ; VRAM_DATA
   MOV R2, #0x11          ; Tile data byte
   MOV [R1], R2           ; Write data (auto-increments address)
   ```

4. **Set Palette Color**:
   ```
   MOV R1, #0x800E        ; CGRAM_ADDR
   MOV R2, #0x01          ; Palette entry 1
   MOV [R1], R2           ; Set address
   MOV R1, #0x800F        ; CGRAM_DATA_L
   MOV R2, #0x1F          ; Red = 0x1F (RGB555)
   MOV [R1], R2           ; Write low byte
   MOV R1, #0x8010        ; CGRAM_DATA_H
   MOV R2, #0x00          ; Green/Blue = 0
   MOV [R1], R2           ; Write high byte
   ```

---

## APU (Audio System)

### Audio Channels

4 independent audio channels:

- **Channels 0-2**: Sine, Square, or Saw waveform
- **Channel 3**: Square or Noise waveform

### Setting Up Audio

1. **Set Frequency** (440 Hz example):
   ```
   ; Frequency = 440 Hz
   ; Sample rate = 44100 Hz
   ; Phase increment = (440 / 44100) * 65536 ≈ 654
   MOV R1, #0x9000        ; CH0_FREQ_LOW
   MOV R2, #0x8E          ; Low byte = 0x8E (654 & 0xFF)
   MOV [R1], R2
   MOV R1, #0x9001        ; CH0_FREQ_HIGH
   MOV R2, #0x02          ; High byte = 0x02 (654 >> 8)
   MOV [R1], R2
   ```

2. **Set Volume**:
   ```
   MOV R1, #0x9002        ; CH0_VOLUME
   MOV R2, #0x80          ; Volume = 128 (50%)
   MOV [R1], R2
   ```

3. **Enable Channel**:
   ```
   MOV R1, #0x9003        ; CH0_CONTROL
   MOV R2, #0x01          ; Bit 0 = enable, bits 2-3 = 0 (sine)
   MOV [R1], R2
   ```

### Waveform Types

- **0 (Sine)**: Smooth sine wave
- **1 (Square)**: Square wave (50% duty cycle)
- **2 (Saw)**: Sawtooth wave
- **3 (Noise)**: White noise (LFSR-based, channel 3 only)

---

## Input System

### Reading Input

1. **Latch Controller**:
   ```
   MOV R1, #0xA001        ; CONTROLLER1_LATCH
   MOV R2, #0x01          ; Latch = 1
   MOV [R1], R2           ; Latch button states
   ```

2. **Read Button State**:
   ```
   MOV R1, #0xA000        ; CONTROLLER1
   MOV R2, [R1]            ; Read button state
   ; R2 now contains button bitfield
   ```

3. **Check Button**:
   ```
   MOV R1, #0xA000
   MOV R2, [R1]            ; Read buttons
   AND R2, #0x10           ; Check A button (bit 4)
   CMP R2, #0x10
   BEQ button_pressed      ; Branch if A pressed
   ```

### Button Constants

- `0x01` = UP
- `0x02` = DOWN
- `0x04` = LEFT
- `0x08` = RIGHT
- `0x10` = A
- `0x20` = B
- `0x40` = X
- `0x80` = Y
- `0x100` = L (read from high byte)
- `0x200` = R (read from high byte)
- `0x400` = START (read from high byte)
- `0x800` = SELECT (read from high byte)

---

## ROM Format

### ROM Header (32 bytes)

| Offset | Size | Field | Description |
|--------|------|-------|-------------|
| 0x00 | 4 | Magic | `0x46434D52` ("RMCF") |
| 0x04 | 2 | Version | ROM format version (currently 1) |
| 0x06 | 4 | ROM Size | Size of code data in bytes |
| 0x0A | 2 | Entry Bank | Entry point bank (typically 1) |
| 0x0C | 2 | Entry Offset | Entry point offset (typically 0x8000) |
| 0x0E | 2 | Mapper Flags | Mapper type and flags |
| 0x10 | 4 | Checksum | Optional checksum |
| 0x14 | 12 | Reserved | Reserved for future use |

### ROM Structure

```
[32-byte Header]
[Code Data...]
```

### Entry Point

The CPU starts execution at the address specified in the ROM header:
- **Bank**: `entry_point_bank` (typically 1)
- **Offset**: `entry_point_offset` (typically 0x8000)

### Creating a ROM

See `create_graphics_rom.py` for an example ROM builder. The basic steps:

1. Create instruction encoding functions
2. Build code array with instructions
3. Calculate branch offsets
4. Pack code into bytes
5. Create 32-byte header
6. Write header + code to file

---

## Programming Examples

### Example 1: Simple Loop

```python
# Initialize counter
MOV R0, #0x0000          ; R0 = 0

loop:
    ADD R0, #1            ; R0 = R0 + 1
    CMP R0, #100          ; Compare R0 with 100
    BLT loop              ; Branch if R0 < 100
    ; Loop continues until R0 >= 100
```

### Example 2: Enable Background and Set Scroll

```python
# Enable BG0
MOV R1, #0x8008           ; BG0_CONTROL address
MOV R2, #0x01             ; Enable BG0, 8x8 tiles
MOV [R1], R2              ; Write control

# Set scroll X = 50
MOV R1, #0x8000           ; BG0_SCROLLX_L
MOV R2, #0x32             ; 50 decimal = 0x32
MOV [R1], R2              ; Write low byte
MOV R1, #0x8001           ; BG0_SCROLLX_H
MOV R2, #0x00             ; High byte = 0
MOV [R1], R2              ; Write high byte

# Set scroll Y = 100
MOV R1, #0x8002           ; BG0_SCROLLY_L
MOV R2, #0x64             ; 100 decimal = 0x64
MOV [R1], R2              ; Write low byte
MOV R1, #0x8003           ; BG0_SCROLLY_H
MOV R2, #0x00             ; High byte = 0
MOV [R1], R2              ; Write high byte
```

### Example 3: Delay Loop

```python
delay:
    MOV R7, #0x0000       ; Initialize counter
    
delay_loop:
    ADD R7, #1            ; Increment counter
    CMP R7, #0x0100       ; Compare with 256
    BNE delay_loop        ; Loop if not equal
    ; Delay complete
```

### Example 4: Bouncing Box Animation

```python
# Initialize position and direction
MOV R3, #0x0000           ; Scroll X = 0
MOV R4, #0x0000           ; Scroll Y = 0
MOV R5, #0x0000           ; X direction (0=right, 1=left)
MOV R6, #0x0000           ; Y direction (0=down, 1=up)

animation_loop:
    # Update X position
    CMP R5, #0             ; Check X direction
    BNE move_left_x
    ADD R3, #5             ; Move right 5 pixels
    CMP R3, #0x0134        ; Check if at right edge
    BLT skip_x_bounce
    MOV R5, #0x0001        ; Reverse direction
    MOV R3, #0x0138        ; Clamp to max
skip_x_bounce:
    JMP skip_x_dec
move_left_x:
    CMP R3, #5             ; Check if at left edge
    BLT bounce_x
    SUB R3, #5             ; Move left 5 pixels
    JMP skip_x_dec
bounce_x:
    MOV R5, #0x0000        ; Reverse direction
    MOV R3, #0x0000        ; Clamp to 0
skip_x_dec:
    
    # Update Y position (similar logic)
    # ... (Y movement code) ...
    
    # Write scroll to PPU
    MOV R1, #0x8000        ; BG0_SCROLLX_L
    MOV [R1], R3            ; Write scroll X
    MOV R1, #0x8002        ; BG0_SCROLLY_L
    MOV [R1], R4            ; Write scroll Y
    
    # Delay
    MOV R7, #0x0000
delay_loop:
    ADD R7, #1
    CMP R7, #0x0100
    BNE delay_loop
    
    JMP animation_loop      ; Loop forever
```

---

## Reference Tables

### Opcode Quick Reference

| Instruction | Opcode | Mode | Description |
|------------|--------|------|-------------|
| NOP | 0x0000 | - | No operation |
| MOV | 0x1000 | 0-5 | Move/load/store |
| ADD | 0x2000 | 0-1 | Add |
| SUB | 0x3000 | 0-1 | Subtract |
| MUL | 0x4000 | 0-1 | Multiply |
| DIV | 0x5000 | 0-1 | Divide |
| AND | 0x6000 | 0-1 | Bitwise AND |
| OR | 0x7000 | 0-1 | Bitwise OR |
| XOR | 0x8000 | 0-1 | Bitwise XOR |
| NOT | 0x9000 | 0 | Bitwise NOT |
| SHL | 0xA000 | 0-1 | Shift left |
| SHR | 0xB000 | 0-1 | Shift right |
| CMP | 0xC000 | 0-1 | Compare |
| BEQ | 0xC100 | 1 | Branch if equal |
| BNE | 0xC200 | 1 | Branch if not equal |
| BGT | 0xC300 | 1 | Branch if greater |
| BLT | 0xC400 | 1 | Branch if less |
| BGE | 0xC500 | 1 | Branch if >= |
| BLE | 0xC600 | 1 | Branch if <= |
| JMP | 0xD000 | 1 | Jump |
| CALL | 0xE000 | 1 | Call subroutine |
| RET | 0xF000 | 0 | Return |

### PPU Register Quick Reference

| Register | Address | Description |
|----------|---------|-------------|
| BG0_SCROLLX | 0x8000-0x8001 | BG0 scroll X (16-bit) |
| BG0_SCROLLY | 0x8002-0x8003 | BG0 scroll Y (16-bit) |
| BG0_CONTROL | 0x8008 | BG0 enable/tile size |
| VRAM_ADDR | 0x800A-0x800B | VRAM address (16-bit) |
| VRAM_DATA | 0x800C | VRAM data (8-bit, auto-increment) |
| CGRAM_ADDR | 0x800E | Palette address (8-bit) |
| CGRAM_DATA | 0x800F-0x8010 | Palette data (16-bit RGB555) |

### Memory Map Quick Reference

| Bank | Address Range | Description |
|------|---------------|-------------|
| 0 | 0x0000-0x7FFF | Work RAM (32KB) |
| 0 | 0x8000-0xFFFF | I/O Registers |
| 1-125 | 0x8000-0xFFFF | ROM (per bank) |
| 126-127 | 0x0000-0xFFFF | Extended WRAM |

---

## Appendix: Instruction Encoding Examples

### Encoding MOV R3, #0x1234

```python
# Opcode: 0x1000 (MOV)
# Mode: 0x1 (immediate)
# Reg1: 3 (destination)
# Reg2: 0 (not used)
# Instruction word: 0x1130
# Immediate word: 0x1234
```

### Encoding ADD R3, R4

```python
# Opcode: 0x2000 (ADD)
# Mode: 0x0 (register)
# Reg1: 3 (destination)
# Reg2: 4 (source)
# Instruction word: 0x2034
```

### Encoding BNE offset

```python
# Opcode: 0xC200 (BNE)
# Mode: 0x1 (relative)
# Reg1: 0 (not used)
# Reg2: 0 (not used)
# Instruction word: 0xC100
# Offset word: signed 16-bit offset
```

---

## Notes

- All addresses are in hexadecimal (0x notation)
- All values are 16-bit unless otherwise specified
- The CPU is little-endian (low byte first)
- ROM code must start at offset 0x8000 in a ROM bank
- Stack grows downward (SP decreases on push)
- Branch offsets are signed 16-bit relative to PC after instruction

---

**End of Programming Manual**

