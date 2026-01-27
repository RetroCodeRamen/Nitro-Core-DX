# Nitro-Core-DX Programming Manual

**Version 1.2**  
**Last Updated: January 27, 2026**

> **⚠️ ARCHITECTURE IN DESIGN PHASE**: This system is currently in active development. The architecture is not yet finalized, and changes may break compatibility with existing ROMs. If you're developing software for Nitro-Core-DX, be aware that future changes may require code updates. See [System Manual](SYSTEM_MANUAL.md) for current implementation status and development plans.

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

The **Nitro-Core-DX** is a custom 16-bit fantasy console inspired by classic 8/16-bit consoles like the SNES and Sega Genesis. It combines the best features of both systems:

- **16-bit CPU** with banked 24-bit addressing
- **320x200 pixel display** (portrait mode: 200x320)
- **Tile-based graphics** with 4 background layers (BG0-BG3), sprites, windowing, and per-scanline scroll
- **Matrix Mode (Mode 7-style)** with large world support and vertical sprites for pseudo-3D games
- **4-channel audio synthesizer** (sine, square, saw, noise)
- **SNES-like input** with 12 buttons
- **60 FPS** target frame rate
- **10 MHz CPU** (166,667 CPU cycles per frame)

### System Specifications

| Feature | Specification |
|---------|--------------|
| Display Resolution | 320x200 (landscape) / 200x320 (portrait) |
| Color Depth | 256 colors (8-bit indexed) |
| Tile Size | 8x8 or 16x16 pixels |
| Max Sprites | 128 (including vertical sprites for Matrix Mode) |
| Matrix Mode | Mode 7-style effects with large world support |
| Audio Channels | 4 (sine, square, saw, noise) |
| Audio Sample Rate | 44,100 Hz |
| CPU Speed | 10 MHz (166,667 cycles/frame) |
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
- **D (Division by Zero)**: Set when division by zero occurs (DIV instruction with divisor = 0)

### Addressing Modes

1. **Register Direct** (Mode 0): `MOV R1, R2`
2. **Immediate** (Mode 1): `MOV R1, #0x1234`
3. **Direct Address** (Mode 2): `MOV R1, [R2]` (load from address in R2)
   - **I/O Registers** (bank 0, offset >= 0x8000): Automatically reads 8-bit and zero-extends to 16-bit
   - **Normal Memory** (WRAM, Extended WRAM, ROM): Reads 16-bit
   - This automatic detection makes it easy to read I/O registers without needing mode 6
4. **Indirect** (Mode 3): `MOV [R1], R2` (store to address in R1)
   - **I/O Registers** (bank 0, offset >= 0x8000): Automatically writes only low 8 bits
   - **Normal Memory**: Writes 16-bit
5. **Stack Operations** (Mode 4/5): `PUSH R1` / `POP R1`
6. **8-bit Load** (Mode 6): `MOV R1, [R2]` (explicitly load 8-bit from address in R2, zero-extended)
   - Use this when you want to explicitly read 8-bit from normal memory
7. **8-bit Store** (Mode 7): `MOV [R1], R2` (explicitly store low 8 bits of R2 to address in R1)
   - Use this when you want to explicitly write 8-bit to normal memory

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
- **Flags**: Sets Z, N, D (division by zero flag)
- **Division by Zero**: If divisor is 0, result is set to 0xFFFF and the D flag is set. The D flag can be checked to detect division by zero errors.
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
    - I/O registers (bank 0, offset >= 0x8000): Automatically reads 8-bit, zero-extends to 16-bit
    - Normal memory: Reads 16-bit
  - **Mode 3**: `MOV [R1], R2` - Store to memory at address in R1
    - I/O registers (bank 0, offset >= 0x8000): Automatically writes only low 8 bits
    - Normal memory: Writes 16-bit
  - **Mode 4**: `PUSH R1` - Push register to stack
  - **Mode 5**: `POP R1` - Pop stack to register
  - **Mode 6**: `MOV R1, [R2]` - Load 8-bit from memory at address in R2 (zero-extended to 16-bit)
  - **Mode 7**: `MOV [R1], R2` - Store 8-bit to memory at address in R1 (stores low byte of R2)
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
- **Stack Underflow**: If the stack is empty (SP >= 0x1FFF) or corrupted (SP < 0x0100), the instruction will return an error. Always ensure there is a matching PUSH for each POP.

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

**Important:** All I/O registers are 8-bit only. The CPU automatically handles this:
- **Mode 2 (`MOV R1, [R2]`)**: When reading from I/O addresses (bank 0, offset >= 0x8000), automatically reads 8-bit and zero-extends to 16-bit. For normal memory, reads 16-bit.
- **Mode 3 (`MOV [R1], R2`)**: When writing to I/O addresses, automatically writes only the low byte. The high byte is ignored. For normal memory, writes 16-bit.
- This automatic detection makes I/O register access seamless - you don't need to use mode 6/7 for I/O registers.

**Example:**
```assembly
MOV R4, #0x803E        ; VBlank flag address
MOV R5, [R4]           ; Mode 2: Automatically reads 8-bit from I/O, zero-extends to 16-bit
CMP R5, #0             ; Compare with 0
BEQ wait_vblank        ; Loop if flag is 0
```

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
| 0x8008 | BG0_CONTROL | 8-bit | BG0 control: bit 0=enable, bit 1=tile size (0=8x8, 1=16x16) |
| 0x8009 | BG1_CONTROL | 8-bit | BG1 control: bit 0=enable, bit 1=tile size |
| 0x800A | BG2_SCROLLX_L | 8-bit | Background 2 scroll X (low byte) |
| 0x800B | BG2_SCROLLX_H | 8-bit | Background 2 scroll X (high byte) |
| 0x800C | BG2_SCROLLY_L | 8-bit | Background 2 scroll Y (low byte) |
| 0x800D | BG2_SCROLLY_H | 8-bit | Background 2 scroll Y (high byte) |
| 0x8021 | BG2_CONTROL | 8-bit | BG2 control: bit 0=enable, bit 1=tile size |
| 0x8022 | BG3_SCROLLX_L | 8-bit | Background 3 scroll X (low byte) |
| 0x8023 | BG3_SCROLLX_H | 8-bit | Background 3 scroll X (high byte) |
| 0x8024 | BG3_SCROLLY_L | 8-bit | Background 3 scroll Y (low byte) |
| 0x8025 | BG3_SCROLLY_H | 8-bit | Background 3 scroll Y (high byte) |
| 0x8026 | BG3_CONTROL | 8-bit | BG3 control: bit 0=enable, bit 1=tile size (can be affine layer) |
| 0x800E | VRAM_ADDR_L | 8-bit | VRAM address (low byte) |
| 0x800F | VRAM_ADDR_H | 8-bit | VRAM address (high byte) |
| 0x8010 | VRAM_DATA | 8-bit | VRAM data (auto-increments address) |
| 0x8012 | CGRAM_ADDR | 8-bit | CGRAM (palette) address (0-255) |
| 0x8013 | CGRAM_DATA | 8-bit | CGRAM data (RGB555, requires two 8-bit writes: low byte first, then high byte) |
| 0x8014 | OAM_ADDR | 8-bit | OAM (sprite) address |
| 0x8015 | OAM_DATA | 8-bit | OAM data |
| 0x8016 | FRAMEBUFFER_ENABLE | 8-bit | Framebuffer enable (0=off, 1=on) |
| 0x8017 | DISPLAY_MODE | 8-bit | Display mode (0=landscape, 1=portrait) |
| 0x8018 | MATRIX_CONTROL | 8-bit | Matrix Mode control (bit 0=enable, bit 1=mirror_h, bit 2=mirror_v) |
| 0x8019 | MATRIX_A_L | 8-bit | Transformation matrix A (low byte, 8.8 fixed point) |
| 0x801A | MATRIX_A_H | 8-bit | Transformation matrix A (high byte) |
| 0x801B | MATRIX_B_L | 8-bit | Transformation matrix B (low byte, 8.8 fixed point) |
| 0x801C | MATRIX_B_H | 8-bit | Transformation matrix B (high byte) |
| 0x801D | MATRIX_C_L | 8-bit | Transformation matrix C (low byte, 8.8 fixed point) |
| 0x801E | MATRIX_C_H | 8-bit | Transformation matrix C (high byte) |
| 0x801F | MATRIX_D_L | 8-bit | Transformation matrix D (low byte, 8.8 fixed point) |
| 0x8020 | MATRIX_D_H | 8-bit | Transformation matrix D (high byte) |
| 0x8027 | MATRIX_CENTER_X_L | 8-bit | Center point X (low byte) |
| 0x8028 | MATRIX_CENTER_X_H | 8-bit | Center point X (high byte) |
| 0x8029 | MATRIX_CENTER_Y_L | 8-bit | Center point Y (low byte) |
| 0x802A | MATRIX_CENTER_Y_H | 8-bit | Center point Y (high byte) |
| 0x802B | WINDOW0_LEFT | 8-bit | Window 0 left edge (0-319) |
| 0x802C | WINDOW0_RIGHT | 8-bit | Window 0 right edge (0-319) |
| 0x802D | WINDOW0_TOP | 8-bit | Window 0 top edge (0-199) |
| 0x802E | WINDOW0_BOTTOM | 8-bit | Window 0 bottom edge (0-199) |
| 0x802F | WINDOW1_LEFT | 8-bit | Window 1 left edge (0-319) |
| 0x8030 | WINDOW1_RIGHT | 8-bit | Window 1 right edge (0-319) |
| 0x8031 | WINDOW1_TOP | 8-bit | Window 1 top edge (0-199) |
| 0x8032 | WINDOW1_BOTTOM | 8-bit | Window 1 bottom edge (0-199) |
| 0x8033 | WINDOW_CONTROL | 8-bit | Window control: bit 0=Window0 enable, bit 1=Window1 enable, bits 2-3=logic (0=OR, 1=AND, 2=XOR, 3=XNOR) |
| 0x8034 | WINDOW_MAIN_ENABLE | 8-bit | Main window enable per layer: bit 0=BG0, 1=BG1, 2=BG2, 3=BG3, 4=sprites |
| 0x8035 | WINDOW_SUB_ENABLE | 8-bit | Sub window enable (for color math, future use) |
| 0x8036 | HDMA_CONTROL | 8-bit | HDMA control: bit 0=enable, bits 1-4=layer enable |
| 0x8037 | HDMA_TABLE_BASE_L | 8-bit | HDMA table base address (low byte, in WRAM) |
| 0x8038 | HDMA_TABLE_BASE_H | 8-bit | HDMA table base address (high byte) |
| 0x8039 | HDMA_SCANLINE | 8-bit | Current scanline for HDMA write (0-199) |
| 0x803A | HDMA_BG0_SCROLLX_L | 8-bit | HDMA: BG0 scroll X for current scanline (low byte) |
| 0x803B | HDMA_BG0_SCROLLX_H | 8-bit | HDMA: BG0 scroll X (high byte) |
| 0x803C | HDMA_BG0_SCROLLY_L | 8-bit | HDMA: BG0 scroll Y (low byte) |
| 0x803D | HDMA_BG0_SCROLLY_H | 8-bit | HDMA: BG0 scroll Y (high byte) |
| 0x803E | VBLANK_FLAG | 8-bit | VBlank flag (bit 0 = VBlank active, one-shot, cleared when read) - hardware-accurate frame synchronization |
| 0x803F | FRAME_COUNTER_LOW | 8-bit | Frame counter low byte (increments once per frame, wraps at 65536) |
| 0x8040 | FRAME_COUNTER_HIGH | 8-bit | Frame counter high byte |

#### APU Registers (0x9000-0x9FFF)

**Channel Registers (8 bytes per channel):**

Each channel has 8 bytes of registers, starting at:
- Channel 0: 0x9000-0x9007
- Channel 1: 0x9008-0x900F
- Channel 2: 0x9010-0x9017
- Channel 3: 0x9018-0x901F

**Per-Channel Register Layout:**

| Offset | Name | Size | Description |
|--------|------|------|-------------|
| +0 | FREQ_LOW | 8-bit | Frequency low byte (Hz) |
| +1 | FREQ_HIGH | 8-bit | Frequency high byte (Hz) - triggers phase reset on change |
| +2 | VOLUME | 8-bit | Volume (0-255, 0=silent, 255=max) |
| +3 | CONTROL | 8-bit | Control register: bit 0=enable, bits 1-2=waveform |
| +4 | DURATION_LOW | 8-bit | Note duration low byte (frames, 0=infinite) |
| +5 | DURATION_HIGH | 8-bit | Note duration high byte (frames) |
| +6 | DURATION_MODE | 8-bit | Duration mode: bit 0=0 (stop when done), bit 0=1 (loop/restart) |
| +7 | Reserved | 8-bit | Reserved for future use |

**Global APU Registers:**

| Address | Name | Size | Description |
|---------|------|------|-------------|
| 0x9020 | MASTER_VOLUME | 8-bit | Master volume (0-255) |
| 0x9021 | CHANNEL_COMPLETION_STATUS | 8-bit | Channel completion flags (bits 0-3 = channels 0-3, one-shot, cleared immediately after read) |

**Waveform Types:**
- `0` = Sine
- `1` = Square
- `2` = Saw
- `3` = Noise

#### Input Registers (0xA000-0xAFFF)

| Address | Name | Size | Description |
|---------|------|------|-------------|
| 0xA000 | CONTROLLER1 | 8-bit | Controller 1 button state (read, low byte) |
| 0xA001 | CONTROLLER1_LATCH | 8-bit | Controller 1 latch (write 1 to latch) / Controller 1 high byte (read) |
| 0xA002 | CONTROLLER2 | 8-bit | Controller 2 button state (read, low byte) |
| 0xA003 | CONTROLLER2_LATCH | 8-bit | Controller 2 latch (write 1 to latch) / Controller 2 high byte (read) |

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

Four tile-based background layers (BG0, BG1, BG2, BG3) for advanced parallax and layering effects:

- **Tile Size**: 8x8 or 16x16 pixels (configurable per layer)
- **Tile Format**: 4bpp (4 bits per pixel, 16 colors per tile)
- **Tilemap**: 64x64 tiles (512x512 pixels for 8x8 tiles)
- **Scrolling**: Independent X/Y scroll per layer
- **Priority**: BG3 (highest) → BG2 → BG1 → BG0 (lowest)
- **BG3**: Can be used as dedicated affine layer (Matrix Mode alternative)

### Matrix Mode (Mode 7-Style Effects)

> **⚠️ IMPLEMENTATION STATUS**: Matrix Mode is currently not fully implemented. The transformation matrix registers are available and can be written to, but the rendering pipeline does not yet apply the transformation. Currently, enabling Matrix Mode will render BG0 normally without transformation. This feature is planned for a future release.

Matrix Mode enables advanced perspective and rotation effects on BG0, perfect for creating 3D-style landscapes, racing game tracks, and **pseudo-3D world maps**. This is the console's implementation of SNES Mode 7-style effects, enhanced for larger worlds.

**Features:**
- **Rotation**: Rotate the entire background
- **Scaling**: Zoom in/out with perspective
- **Perspective**: Create pseudo-3D "looking down a road" effects
- **Mirroring**: Horizontal and vertical mirroring support
- **Large World Maps**: Support for large tilemaps through tile stitching and extended VRAM
- **Vertical Sprites**: Sprites rendered in 3D space (buildings, people, objects) that scale and position based on Matrix Mode transformation

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
- `MATRIX_CONTROL` (0x8018): Enable Matrix Mode and set mirroring flags
  - Bit 0: Enable Matrix Mode (1=enabled, 0=normal BG0)
  - Bit 1: Mirror horizontally
  - Bit 2: Mirror vertically
- `MATRIX_A/B/C/D` (0x8019-0x8020): Transformation matrix coefficients (16-bit, 8.8 fixed point)
- `MATRIX_CENTER_X/Y` (0x8027-0x802A): Center point for transformation (16-bit)

**8.8 Fixed Point Format:**
Values are stored as 16-bit signed integers where:
- Integer part: bits 15-8
- Fractional part: bits 7-0
- Example: 1.0 = 0x0100, 0.5 = 0x0080, -1.0 = 0xFF00

**Large World Maps:**

Matrix Mode supports **large world maps** through:
- **Extended Tilemaps**: Normal tilemaps are 32×32 tiles, but Matrix Mode can use larger tilemaps by accessing extended VRAM regions
- **Tile Stitching**: Multiple tilemaps can be arranged to create seamless large worlds
- **World Coordinate System**: World coordinates can span multiple tilemap boundaries, with seamless transitions

**Vertical Sprites:**

Vertical sprites are sprites that exist in world space and are rendered with 3D perspective effects when Matrix Mode is enabled. They enable pseudo-3D worlds with buildings, people, and objects.

**Vertical Sprite Properties:**
- **World Coordinates**: X/Y position in tilemap/world space (16-bit signed)
- **Base Size**: Sprite size at reference distance (8×8 or 16×16)
- **Scaling**: Automatic scaling based on distance from camera
- **Depth Sorting**: Rendered back-to-front for correct occlusion

**Setting Up Matrix Mode:**

1. **Enable BG0 and Matrix Mode**:
   ```
   MOV R1, #0x8008        ; BG0_CONTROL
   MOV R2, #0x01          ; Enable BG0
   MOV [R1], R2
   MOV R1, #0x8018        ; MATRIX_CONTROL
   MOV R2, #0x01          ; Enable Matrix Mode
   MOV [R1], R2
   ```

2. **Set Transformation Matrix** (example: 45° rotation):
   ```
   ; Rotation matrix: cos(45°) = sin(45°) ≈ 0.707 = 0x00B5
   MOV R1, #0x8019        ; MATRIX_A_L
   MOV R2, #0xB5          ; A = 0.707 (low byte)
   MOV [R1], R2
   MOV R1, #0x801A        ; MATRIX_A_H
   MOV R2, #0x00          ; A (high byte)
   MOV [R1], R2
   
   MOV R1, #0x801B        ; MATRIX_B_L
   MOV R2, #0x4B          ; B = -0.707 (low byte of -0.707)
   MOV [R1], R2
   MOV R1, #0x801C        ; MATRIX_B_H
   MOV R2, #0xFF          ; B (high byte, negative)
   MOV [R1], R2
   
   ; C = 0.707, D = 0.707 (similar pattern for 0x801D-0x8020)
   ```

3. **Set Center Point**:
   ```
   MOV R1, #0x8027        ; MATRIX_CENTER_X_L
   MOV R2, #0xA0          ; Center X = 160 (screen center)
   MOV [R1], R2
   MOV R1, #0x8028        ; MATRIX_CENTER_X_H
   MOV R2, #0x00
   MOV [R1], R2
   
   MOV R1, #0x8029        ; MATRIX_CENTER_Y_L
   MOV R2, #0x64          ; Center Y = 100 (screen center)
   MOV [R1], R2
   MOV R1, #0x802A        ; MATRIX_CENTER_Y_H
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

The Nitro-Core-DX sprite system supports up to **128 sprites** on screen simultaneously, with flexible palette selection and advanced features like priority, flipping, and blending.

#### Sprite System Overview

- **Max Sprites**: 128 sprites
- **Size**: 8×8 or 16×16 pixels (per sprite)
- **Attributes**: X/Y position, tile index, palette, priority, flip X/Y, blend mode, alpha
- **Rendering Order**: Sprites are sorted by priority, then by index (lower index = higher priority if same priority level)

#### Color Limit Per Sprite

**Each sprite can use up to 16 colors**, but with an important caveat:

- **Format**: 4bpp (4 bits per pixel)
- **Color indices**: 0-15 per pixel
- **Color index 0**: **Transparent** (not rendered)
- **Visible colors**: **15 colors** (indices 1-15) per sprite

**Palette System:**

Each sprite selects one of **16 palettes** (palette index 0-15) from CGRAM:

- **CGRAM structure**: 256 total colors organized as 16 palettes × 16 colors each
- **Palette selection**: Stored in sprite attributes (bits [3:0] of byte 4)
- **Color lookup**: `CGRAM address = (paletteIndex × 16 + colorIndex) × 2`

**Example:**
- Sprite uses palette 1, color index 5
- CGRAM address = (1 × 16 + 5) × 2 = 42
- Reads RGB555 color from CGRAM[42] and CGRAM[43]

#### Sprite Format (OAM Entry)

Each sprite occupies **6 bytes** in OAM (Object Attribute Memory):

```
Byte 0: X position (low byte, signed)
Byte 1: X position (high byte, bit 0 only, sign extends)
Byte 2: Y position (8-bit, 0-255)
Byte 3: Tile index (8-bit, 0-255)
Byte 4: Attributes
  - Bits [3:0]: Palette index (0-15)
  - Bit 4: Flip X
  - Bit 5: Flip Y
  - Bits [7:6]: Priority (0-3)
Byte 5: Control
  - Bit 0: Enable (1=enabled, 0=disabled)
  - Bit 1: Size (0=8×8, 1=16×16)
  - Bits [3:2]: Blend mode (0=normal, 1=alpha, 2=additive, 3=subtractive)
  - Bits [7:4]: Alpha value (0-15, 0=transparent, 15=opaque)
```

#### Tile Data Format

Sprites use the same tile format as backgrounds:

- **4bpp**: 4 bits per pixel (2 pixels per byte)
- **8×8 tile**: 64 pixels = 32 bytes
- **16×16 tile**: 256 pixels = 128 bytes
- **Pixel order**: Row-major, 2 pixels per byte
  - Even pixels (0, 2, 4, ...): Upper 4 bits of byte
  - Odd pixels (1, 3, 5, ...): Lower 4 bits of byte

**Example tile data byte:**
```
Byte = 0x1A
- Upper 4 bits (0x1): Color index 1 for even pixel
- Lower 4 bits (0xA): Color index 10 for odd pixel
```

#### Sprite Properties

**Size:**
- **8×8 pixels**: Standard size, 32 bytes of tile data
- **16×16 pixels**: Large sprites, 128 bytes of tile data
- Configurable per sprite via control byte bit 1

**Position:**
- **X position**: 16-bit signed (-32768 to 32767)
  - Low byte: Byte 0
  - High byte: Byte 1 (only bit 0 used, sign extends)
- **Y position**: 8-bit unsigned (0-255)

**Priority:**
- **4 priority levels**: 0 (lowest) to 3 (highest)
- Sprites with higher priority render in front
- Sprites render in front of backgrounds (BG0-BG3)
- Rendering order: Sprites sorted by priority, then by index

**Flipping:**
- **Flip X**: Mirror sprite horizontally
- **Flip Y**: Mirror sprite vertically
- Useful for animation and reducing tile data

**Blending Modes:**
- **0 (Normal)**: Opaque, no blending
- **1 (Alpha)**: Alpha blending with background
- **2 (Additive)**: Add sprite color to background
- **3 (Subtractive)**: Subtract sprite color from background

**Transparency:**
- **Color index 0**: Always transparent (not rendered)
- **Alpha value**: Controls transparency for alpha blending mode
  - 0 = fully transparent
  - 15 = fully opaque

#### Memory Layout

**OAM (Object Attribute Memory):**
- **Size**: 768 bytes (128 sprites × 6 bytes)
- **Location**: VRAM offset 0x9000-0x92FF (typical)
- **Access**: Via OAM_ADDR (0x8014) and OAM_DATA (0x8015) registers

#### OAM Access Patterns and Byte Index Behavior

**⚠️ CRITICAL**: Understanding OAM byte index behavior is essential for correct sprite updates. Incorrect usage can corrupt sprite data.

**OAM Byte Index Auto-Increment:**
- When you write to `OAM_DATA` (0x8015), the PPU automatically increments an internal byte index
- When you read from `OAM_DATA`, the byte index also increments (same as write)
- After 6 bytes (one complete sprite entry), the byte index wraps to 0 and `OAM_ADDR` advances to the next sprite
- Writing to `OAM_ADDR` (0x8014) **resets the byte index to 0** for that sprite

**OAM Write Protection:**
- OAM writes are **only allowed during VBlank** (scanlines 200-262) or before the first frame starts
- Attempts to write during visible rendering (scanlines 0-199) are ignored
- **Always wait for VBlank** before updating sprite data to avoid corruption

**Common Pitfalls:**

1. **Partial Updates Without Reset**: If you read some bytes from a sprite (e.g., to check X position), the byte index advances. If you then try to write without resetting `OAM_ADDR`, your writes will go to the wrong bytes.

   ```assembly
   ; ❌ WRONG: Reading then writing without reset
   MOV R1, #0x8014  ; OAM_ADDR
   MOV R2, #0x00    ; Sprite 0
   MOV [R1], R2
   MOV R1, #0x8015  ; OAM_DATA
   MOV R2, [R1]     ; Read X low (byte index now = 1)
   ; ... do calculations ...
   MOV [R1], R2     ; Write goes to byte 1 (X high) instead of byte 0!
   ```

2. **Forgetting to Write Control Byte**: After writing Attributes (byte 4), the byte index increments to 5 (Control byte). If you don't write the Control byte, the next write will wrap to byte 0 and corrupt X position.

   ```assembly
   ; ❌ WRONG: Writing Attributes without preserving Control byte
   MOV R1, #0x8014  ; OAM_ADDR
   MOV R2, #0x00    ; Sprite 0
   MOV [R1], R2
   MOV R1, #0x8015  ; OAM_DATA
   ; Skip to Attributes (byte 4)
   MOV R2, [R1]     ; Skip X low
   MOV R2, [R1]     ; Skip X high
   MOV R2, [R1]     ; Skip Y
   MOV R2, [R1]     ; Skip Tile
   MOV R2, #0x01    ; New palette
   MOV [R1], R2     ; Write Attributes (byte index now = 5)
   ; Byte index wraps to 0, next write corrupts X position!
   ```

3. **Reading OAM_DATA Increments Index**: Reading from `OAM_DATA` increments the byte index just like writing. This can cause issues if you read for debugging or checking values.

**Best Practices:**

1. **Always Reset OAM_ADDR Before Updates**: When updating a sprite, always write to `OAM_ADDR` first to reset the byte index to 0.

   ```assembly
   ; ✅ CORRECT: Reset OAM_ADDR before updating
   MOV R1, #0x8014  ; OAM_ADDR
   MOV R2, #0x00    ; Sprite 0
   MOV [R1], R2     ; Reset byte index to 0
   MOV R1, #0x8015  ; OAM_DATA
   ; Now write sprite data starting from byte 0
   ```

2. **Write Complete Sprite Entries**: When updating a sprite, write all 6 bytes (or at least write the Control byte last to preserve sprite state).

   ```assembly
   ; ✅ CORRECT: Write complete sprite entry
   MOV R1, #0x8014  ; OAM_ADDR
   MOV R2, #0x00    ; Sprite 0
   MOV [R1], R2
   MOV R1, #0x8015  ; OAM_DATA
   MOV R2, #50      ; X low
   MOV [R1], R2
   MOV R2, #0x00    ; X high
   MOV [R1], R2
   MOV R2, #50      ; Y
   MOV [R1], R2
   MOV R2, #0x00    ; Tile index
   MOV [R1], R2
   MOV R2, #0x01    ; Attributes (palette 1)
   MOV [R1], R2
   MOV R2, #0x03    ; Control (enabled, 16x16)
   MOV [R1], R2     ; Always write Control byte last!
   ```

3. **Partial Updates with Care**: If you only need to update some bytes (e.g., just X position), reset `OAM_ADDR`, skip to the byte you want, write it, then write any remaining bytes that need to be preserved.

   ```assembly
   ; ✅ CORRECT: Partial update preserving Control byte
   ; Update X position only
   MOV R1, #0x8014  ; OAM_ADDR
   MOV R2, #0x00    ; Sprite 0
   MOV [R1], R2     ; Reset byte index to 0
   MOV R1, #0x8015  ; OAM_DATA
   MOV R2, #100     ; New X low
   MOV [R1], R2     ; Write X low (byte index now = 1)
   MOV R2, #0x00    ; X high
   MOV [R1], R2     ; Write X high (byte index now = 2)
   ; Skip Y, Tile, Attributes (read to advance byte index)
   MOV R2, [R1]     ; Skip Y (byte index now = 3)
   MOV R2, [R1]     ; Skip Tile (byte index now = 4)
   MOV R2, [R1]     ; Skip Attributes (byte index now = 5)
   MOV R2, #0x03    ; Control byte (preserve enabled, 16x16)
   MOV [R1], R2     ; Write Control byte
   ```

4. **Wait for VBlank**: Always wait for VBlank before updating sprites.

   ```assembly
   ; ✅ CORRECT: Wait for VBlank before sprite updates
   wait_vblank:
   MOV R1, #0x803E  ; VBLANK_FLAG
   MOV R2, [R1]     ; Read VBlank flag
   MOV R3, #0x00
   CMP R2, R3       ; Compare with 0
   BEQ wait_vblank  ; Loop if flag is 0
   ; Now safe to update sprites
   ```

#### Sprite Update Best Practices: Lessons from Real-World Debugging

**⚠️ CRITICAL**: These practices are based on actual debugging experience. Following them will prevent common sprite corruption issues.

**1. Register Preservation When Updating Position**

When updating a sprite's X or Y position, you must read and preserve all other sprite fields (Y, Tile, Attributes, Control) before writing the new position. Otherwise, the auto-increment behavior will cause writes to corrupt other fields.

   ```assembly
   ; ✅ CORRECT: Read all fields before updating X position
   MOV R1, #0x8014  ; OAM_ADDR
   MOV R2, #0x00    ; Sprite 0
   MOV [R1], R2     ; Reset byte index to 0
   MOV R1, #0x8015  ; OAM_DATA
   
   ; Read current sprite data (preserve everything)
   MOV R2, [R1]     ; Read X low (byte 0) - save to R6
   MOV R6, R2
   MOV R2, [R1]     ; Read X high (byte 1) - save to R5
   MOV R5, R2
   MOV R2, [R1]     ; Read Y (byte 2) - save to R0
   MOV R0, R2
   MOV R2, [R1]     ; Read Tile (byte 3) - save to R3
   MOV R3, R2
   MOV R2, [R1]     ; Read Attributes (byte 4) - save to R2
   MOV R2, R2       ; (R2 already has it)
   MOV R4, [R1]     ; Read Control (byte 5) - save to R4
   
   ; Now calculate new X position
   ADD R6, #1       ; Increment X low
   ; ... handle wrapping, sign bit, etc. ...
   
   ; Write all 6 bytes back to preserve sprite state
   MOV R1, #0x8014  ; OAM_ADDR
   MOV R2, #0x00    ; Sprite 0
   MOV [R1], R2     ; Reset byte index to 0
   MOV R1, #0x8015  ; OAM_DATA
   MOV R2, R6       ; New X low
   MOV [R1], R2
   MOV R2, R5       ; New X high
   MOV [R1], R2
   MOV R2, R0       ; Preserved Y
   MOV [R1], R2
   MOV R2, R3       ; Preserved Tile
   MOV [R1], R2
   MOV R2, R2       ; Preserved Attributes (already in R2)
   MOV [R1], R2
   MOV R2, #0x03    ; Control byte (enabled, 16x16)
   MOV [R1], R2
   ```

   ```assembly
   ; ❌ WRONG: Updating X without preserving other fields
   MOV R1, #0x8014  ; OAM_ADDR
   MOV R2, #0x00    ; Sprite 0
   MOV [R1], R2
   MOV R1, #0x8015  ; OAM_DATA
   MOV R2, [R1]     ; Read X low
   ADD R2, #1       ; Increment X
   MOV [R1], R2     ; Write X low
   ; Problem: Byte index is now 1, but we haven't preserved Y, Tile, Attributes, Control!
   ; Next write will corrupt X high, and subsequent operations will corrupt other fields
   ```

**2. Complete Sprite Entry Writes**

Always write all 6 bytes of a sprite entry when updating it. If you only update some fields, you must still write the remaining bytes to preserve their values.

   ```assembly
   ; ✅ CORRECT: Update palette but preserve all other fields
   MOV R1, #0x8014  ; OAM_ADDR
   MOV R2, #0x00    ; Sprite 0
   MOV [R1], R2
   MOV R1, #0x8015  ; OAM_DATA
   
   ; Skip to Attributes (byte 4)
   MOV R2, [R1]     ; Skip X low
   MOV R2, [R1]     ; Skip X high
   MOV R2, [R1]     ; Skip Y
   MOV R2, [R1]     ; Skip Tile
   MOV R2, #0x02    ; New palette (palette 2)
   MOV [R1], R2     ; Write Attributes (byte index now = 5)
   MOV R2, #0x03    ; Control byte (MUST write this!)
   MOV [R1], R2     ; Write Control byte
   ; If you don't write Control byte, the next OAM operation will wrap and corrupt X
   ```

**3. Conditional Code Execution (Wrap Blocks)**

When you have code that should only execute under certain conditions (e.g., wrapping when sprite goes off-screen), ensure your branch logic correctly skips the conditional code. If the branch check is incorrect, the conditional code will execute every frame, causing corruption.

   ```assembly
   ; ✅ CORRECT: Proper wrap check with branch
   MOV R1, #0x8014  ; OAM_ADDR
   MOV R2, #0x00    ; Sprite 0
   MOV [R1], R2
   MOV R1, #0x8015  ; OAM_DATA
   MOV R2, [R1]     ; Read X low
   ADD R2, #1       ; Increment X
   MOV R6, R2       ; Save incremented X
   
   ; Check if X >= 336 (off-screen)
   MOV R7, #336
   CMP R6, R7       ; Compare X with 336
   BLT no_wrap      ; Branch if X < 336 (skip wrap code)
   ; Wrap code only executes if X >= 336
   MOV R6, #0xF0    ; Set X to -16 (wrapped)
   ; ... wrap handling code ...
   
   no_wrap:
   ; Continue with normal sprite update
   ; ... write sprite data ...
   ```

   ```assembly
   ; ❌ WRONG: Wrap code executes every frame
   MOV R1, #0x8014  ; OAM_ADDR
   MOV R2, #0x00    ; Sprite 0
   MOV [R1], R2
   MOV R1, #0x8015  ; OAM_DATA
   MOV R2, [R1]     ; Read X low
   ADD R2, #1       ; Increment X
   
   ; Problem: No branch check - wrap code always executes!
   MOV R2, #0xF0    ; Set X to -16 (WRONG - executes every frame!)
   ; This causes sprite to wrap immediately, corrupting position
   ```

**4. Palette Cycling Best Practices**

When cycling through palettes, always read the current Attributes value, modify it, and write it back. Don't forget to write the Control byte after Attributes to preserve sprite state.

   ```assembly
   ; ✅ CORRECT: Palette cycling with proper preservation
   MOV R1, #0x8014  ; OAM_ADDR
   MOV R2, #0x00    ; Sprite 0
   MOV [R1], R2
   MOV R1, #0x8015  ; OAM_DATA
   
   ; Skip to Attributes (byte 4)
   MOV R2, [R1]     ; Skip X low
   MOV R2, [R1]     ; Skip X high
   MOV R2, [R1]     ; Skip Y
   MOV R2, [R1]     ; Skip Tile
   MOV R2, [R1]     ; Read Attributes into R2
   
   ; Increment palette (bits [3:0])
   ADD R2, #1       ; Increment palette
   MOV R7, #0x0F
   AND R2, R7       ; Mask to 4 bits (wrap at 16)
   MOV R7, #0x05
   CMP R2, R7       ; Compare with 5
   BLT palette_ok   ; If palette < 5, continue
   MOV R2, #0x01    ; Wrap to palette 1
   
   palette_ok:
   ; Write Attributes back
   MOV R1, #0x8014  ; OAM_ADDR (reset byte index)
   MOV R3, #0x00    ; Sprite 0
   MOV [R1], R3
   MOV R1, #0x8015  ; OAM_DATA
   MOV R3, [R1]     ; Skip X low
   MOV R3, [R1]     ; Skip X high
   MOV R3, [R1]     ; Skip Y
   MOV R3, [R1]     ; Skip Tile
   MOV R3, R2       ; Copy new palette to R3
   MOV [R1], R3     ; Write Attributes (byte index now = 5)
   MOV R3, #0x03    ; Control byte (MUST write this!)
   MOV [R1], R3     ; Write Control byte
   ```

**5. Debugging Sprite Issues**

If sprites are flickering, disappearing, or showing incorrect colors:

1. **Check OAM byte index**: Ensure you're resetting `OAM_ADDR` before each sprite update
2. **Verify complete writes**: Make sure you're writing all 6 bytes (or at least the Control byte)
3. **Check register preservation**: Verify that you're preserving Y, Tile, Attributes, and Control when updating position
4. **Verify conditional logic**: Ensure wrap blocks and other conditional code only execute when intended
5. **Check VBlank timing**: Ensure sprite updates happen during VBlank (scanlines 200-262)

**Common Symptoms and Causes:**

- **Sprite flickers**: Palette or Control byte being overwritten every frame
- **Sprite disappears**: Control byte set to 0x00 (disabled) or corrupted
- **Sprite shows wrong colors**: Attributes byte corrupted or palette cycling incorrectly
- **Sprite wraps immediately**: Wrap check logic incorrect, wrap code executing every frame
- **Sprite position jumps**: X/Y values corrupted due to incomplete writes

**Tile Data:**
- **Location**: VRAM (anywhere in 64KB space)
- **Format**: 4bpp, row-major order
- **Tile index**: 0-255 for 8×8 tiles, 0-63 for 16×16 tiles

#### Setting Up a Sprite

```
; Set up sprite 0:
; 1. Set OAM address
MOV R1, #0x8014  ; OAM_ADDR
MOV R2, #0x00    ; Sprite 0
MOV [R1], R2

; 2. Write sprite data
MOV R1, #0x8015  ; OAM_DATA

; X position (low byte)
MOV R2, #0x50    ; X = 80
MOV [R1], R2

; X position (high byte)
MOV R2, #0x00    ; High byte = 0
MOV [R1], R2

; Y position
MOV R2, #0x64    ; Y = 100
MOV [R1], R2

; Tile index
MOV R2, #0x00    ; Use tile 0
MOV [R1], R2

; Attributes (palette 1, no flip, priority 3)
MOV R2, #0xC1    ; 0b11000001 = priority 3, palette 1
MOV [R1], R2

; Control (enabled, 8×8, normal blend)
MOV R2, #0x01    ; Enable, 8×8
MOV [R1], R2
```

#### Color Limitations Summary

- **Per sprite**: 15 visible colors (color indices 1-15)
- **Color index 0**: Always transparent
- **Palette selection**: One of 16 palettes (0-15)
- **Total colors available**: 256 colors in CGRAM (16 palettes × 16 colors)
- **Multiple sprites**: Can share the same palette or use different palettes

#### Sprite Tips

1. **Color index 0 is transparent**: Use color indices 1-15 for visible pixels
2. **Palette sharing**: Multiple sprites can use the same palette to save CGRAM space
3. **Animation**: Use tile index changes for sprite animation
4. **Flipping**: Use flip X/Y to mirror sprites, reducing tile data needed
5. **Priority**: Use priority levels to control sprite layering
6. **Blending**: Use alpha blending for semi-transparent effects

### Vertical Sprites (Matrix Mode)

Vertical sprites are special sprites for Matrix Mode that render in 3D space with perspective effects. They enable pseudo-3D worlds with buildings, people, and objects.

**Vertical Sprite Properties:**
- **World Coordinates**: X/Y position in tilemap/world space (16-bit signed)
- **Base Size**: Sprite size at reference distance (8×8 or 16×16)
- **Scaling**: Automatic scaling based on distance from camera (world Y position)
- **Depth Sorting**: Rendered back-to-front by world Y coordinate

**Use Cases:**
- **Buildings**: Large sprites that scale based on distance
- **People/Characters**: Smaller sprites that move in 3D space
- **Objects**: Items, vehicles, decorations in 3D world
- **Parallax Layers**: Multiple sprite layers at different depths

**Implementation Notes:**
- Vertical sprites use extended OAM entries or WRAM for world coordinates
- Scale calculation: `sprite_scale = (camera_height / sprite_world_y) × base_scale`
- Sprites outside screen bounds can be culled for performance
- Recommended limit: 32-64 active vertical sprites for optimal performance

### VRAM Layout

VRAM (64KB) is organized as:

- **Tile Data**: 4bpp tile patterns (32 bytes per 8×8 tile, 128 bytes per 16×16 tile)
- **Tilemap**: Tile indices and attributes (2 bytes per tile entry)
  - Normal tilemaps: 32×32 tiles (2048 bytes per tilemap)
  - **Large Worlds**: Multiple tilemaps can be stitched together for seamless large worlds
  - Extended tilemaps (64×64, 128×128) can be stored across multiple VRAM regions
- **OAM**: Sprite attribute table (768 bytes for 128 sprites)
  - **Vertical Sprites**: Extended attributes (world coordinates) can be stored in WRAM or extended OAM

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
   MOV R1, #0x800E        ; VRAM_ADDR_L
   MOV R2, #0x00          ; Address = 0x0000
   MOV [R1], R2           ; Write low byte
   MOV R1, #0x800F        ; VRAM_ADDR_H
   MOV R2, #0x00          ; High byte = 0
   MOV [R1], R2           ; Write high byte
   MOV R1, #0x8010        ; VRAM_DATA
   MOV R2, #0x11          ; Tile data byte
   MOV [R1], R2           ; Write data (auto-increments address)
   ```

4. **Set Palette Color**:
   ```
   MOV R1, #0x8012        ; CGRAM_ADDR
   MOV R2, #0x00          ; Palette 0, color 0
   MOV [R1], R2           ; Set address
   MOV R1, #0x8013        ; CGRAM_DATA
   ; CGRAM_DATA requires two 8-bit writes to the same address
   ; First write: low byte (RGB555 format)
   MOV R2, #0x1F          ; Low byte = 0x1F (blue in RGB555)
   MOV [R1], R2           ; Write low byte
   ; Second write: high byte (auto-increments CGRAM address)
   MOV R2, #0x00          ; High byte = 0x00
   MOV [R1], R2           ; Write high byte (completes RGB555 value 0x001F)
   ```

---

## APU (Audio System)

### Audio Channels

4 independent audio channels:

- **Channels 0-2**: Sine, Square, or Saw waveform
- **Channel 3**: Square or Noise waveform

### Execution Order

**Important:** The emulator runs in this order each frame:
1. **APU.UpdateFrame()** - Decrements durations, sets completion flags
2. **CPU execution** - Your ROM code runs (can check completion status)
3. **PPU rendering** - Frame is rendered
4. **Audio generation** - Audio samples generated

This ensures completion status is available **before** your ROM code runs, so you can check it reliably.

### Setting Up Audio

1. **Set Frequency** (440 Hz example):
   ```
   ; Frequency = 440 Hz (direct value, not phase increment)
   ; Write low byte first, then high byte
   MOV R7, #0x9000        ; CH0_FREQ_LOW
   MOV R0, #0xB8          ; Low byte = 0xB8 (440 & 0xFF)
   MOV [R7], R0
   MOV R7, #0x9001        ; CH0_FREQ_HIGH
   MOV R0, #0x01          ; High byte = 0x01 (440 >> 8)
   MOV [R7], R0
   ```

2. **Set Volume**:
   ```
   MOV R7, #0x9002        ; CH0_VOLUME
   MOV R0, #0x80          ; Volume = 128 (50%)
   MOV [R7], R0
   ```

3. **Enable Channel**:
   ```
   MOV R7, #0x9003        ; CH0_CONTROL
   MOV R0, #0x01          ; Bit 0 = enable, bits 1-2 = 0 (sine)
   MOV [R7], R0
   ```

### Waveform Types

- **0 (Sine)**: Smooth sine wave
- **1 (Square)**: Square wave (50% duty cycle)
- **2 (Saw)**: Sawtooth wave
- **3 (Noise)**: White noise (LFSR-based, channel 3 only)

### Audio Timing and Note Durations

The APU now supports **automatic note duration control** - a developer-friendly feature that makes timing much easier!

#### Automatic Duration Control (Recommended)

Each channel has **duration registers** that automatically count down each frame. When duration reaches 0, the channel can auto-disable (or loop, depending on mode).

**Key Features:**
- **Duration in frames**: Set duration directly in frames (60 frames = 1 second at 60 FPS)
- **Automatic countdown**: APU decrements duration each frame automatically
- **Auto-disable**: Channel automatically disables when duration expires (if stop mode)
- **Completion status**: Read `CHANNEL_COMPLETION_STATUS` (0x9021) to detect when channels finish

**Example: Play a note for 1 second (60 frames)**

```
; Set up channel 0 for a note with 60-frame duration
MOV R7, #0x9000        ; CH0_FREQ_LOW
MOV R0, #0x06          ; Low byte (262 Hz = C4)
MOV [R7], R0

MOV R7, #0x9001        ; CH0_FREQ_HIGH
MOV R0, #0x01          ; High byte
MOV [R7], R0

MOV R7, #0x9002        ; CH0_VOLUME
MOV R0, #0x80          ; Volume = 128
MOV [R7], R0

; Set duration to 60 frames (1 second)
MOV R7, #0x9004        ; CH0_DURATION_LOW
MOV R0, #60            ; Low byte = 60
MOV [R7], R0

MOV R7, #0x9005        ; CH0_DURATION_HIGH
MOV R0, #0             ; High byte = 0
MOV [R7], R0

; Set duration mode: stop when done (mode 0)
MOV R7, #0x9006        ; CH0_DURATION_MODE
MOV R0, #0             ; Mode 0 = stop when done
MOV [R7], R0

; NOW enable the channel (do this LAST!)
MOV R7, #0x9003        ; CH0_CONTROL
MOV R0, #0x01          ; Enable, sine wave
MOV [R7], R0
```

**Important:** Always set frequency, volume, and duration **BEFORE** enabling the channel!

#### Frame Synchronization Options

The emulator provides **three synchronization mechanisms** for different use cases:

**1. Completion Status Register (0x9021) - One-Shot**
- **Best for**: Simple audio timing, detecting when channels finish
- **Behavior**: One-shot flag, cleared immediately after read
- **Use when**: You only need to know if a channel finished, don't need frame boundaries

**2. Frame Counter (0x803F/0x8040) - Continuous**
- **Best for**: Precise timing, frame-perfect synchronization, debugging
- **Behavior**: 16-bit counter that increments once per frame (wraps at 65536)
- **Use when**: You need to measure time in frames, or wait for specific frame numbers

**3. VBlank Flag (0x803E) - Hardware-Accurate**
- **Best for**: Hardware-accurate synchronization, FPGA implementation compatibility
- **Behavior**: One-shot flag set at start of each frame, cleared when read
- **Use when**: You want to match real hardware behavior (NES, SNES pattern), or plan FPGA implementation

#### Using Completion Status Register

The `CHANNEL_COMPLETION_STATUS` register (0x9021) is a **one-shot flag** that indicates which channels just finished this frame:

- **Bits 0-3**: Channels 0-3 completion flags (1 = channel finished this frame)
- **One-shot behavior**: Flag is cleared immediately after being read
- **Read once per frame**: Check this register once per frame to detect channel completion

**Example: Detect when channel 0 finishes and start next note**

```
main_loop:
    ; ... your game logic ...
    
    ; Check if channel 0 just finished
    MOV R7, #0x9021        ; CHANNEL_COMPLETION_STATUS
    MOV R6, [R7]           ; Read completion status (16-bit read)
    AND R6, #0x01          ; Mask to bit 0 (channel 0 flag)
    CMP R6, #0             ; Compare with 0
    BEQ skip_note_update   ; If channel 0 didn't finish, skip update
    
    ; Channel 0 finished - start next note!
    ; ... update frequency, duration, re-enable channel ...
    
skip_note_update:
    JMP main_loop
```

#### Using Frame Counter

The `FRAME_COUNTER` register (0x803F/0x8040) is a **16-bit counter** that increments once per frame:

- **Low byte (0x803F)**: Frame counter bits 0-7
- **High byte (0x8040)**: Frame counter bits 8-15
- **Behavior**: Increments at start of each frame, wraps at 65536
- **Use for**: Precise timing, frame-perfect synchronization, measuring elapsed time

**Example: Wait for next frame using frame counter**

```
main_loop:
    ; Read current frame counter
    MOV R7, #0x803F        ; FRAME_COUNTER_LOW
    MOV R3, [R7]           ; Read 16-bit frame counter into R3
    
    ; ... your game logic ...
    
    ; Wait for frame counter to change (next frame)
    wait_frame:
        MOV R7, #0x803F    ; FRAME_COUNTER_LOW
        MOV R6, [R7]       ; Read current frame counter
        CMP R6, R3         ; Compare with previous frame
        BEQ wait_frame     ; If same, keep waiting
    
    ; Now we're in a new frame!
    ; Update frame counter for next iteration
    MOV R3, R6
    
    ; Check completion status (now in new frame)
    MOV R7, #0x9021        ; CHANNEL_COMPLETION_STATUS
    MOV R6, [R7]           ; Read completion status
    ; ... process completion ...
    
    JMP main_loop
```

#### Using VBlank Flag (Hardware-Accurate)

The `VBLANK_FLAG` register (0x803E) is a **one-shot flag** that indicates the start of vertical blanking period:

- **Bit 0**: VBlank active (1 = VBlank period, 0 = not VBlank)
- **One-shot behavior**: Flag is cleared immediately after being read
- **Hardware-accurate**: Matches real hardware behavior (NES, SNES, etc.)
- **Best for**: FPGA implementation, hardware-accurate synchronization

**Example: Wait for VBlank (hardware-accurate pattern)**

```
main_loop:
    ; Wait for VBlank signal
    wait_vblank:
        MOV R7, #0x803E    ; VBLANK_FLAG
        MOV R6, [R7]       ; Read VBlank flag
        AND R6, #0x01      ; Mask to bit 0
        CMP R6, #0         ; Compare with 0
        BEQ wait_vblank    ; If not VBlank, keep waiting
    
    ; Now we're at start of frame (VBlank period)
    ; This is the standard pattern for retro game development
    
    ; Check completion status
    MOV R7, #0x9021        ; CHANNEL_COMPLETION_STATUS
    MOV R6, [R7]           ; Read completion status
    ; ... process completion ...
    
    ; ... your game logic ...
    
    JMP main_loop
```

**Execution Order:**
The emulator runs in this order each frame:
1. **APU.UpdateFrame()** - Decrements durations, sets completion flags
2. **CPU execution** - Your ROM code runs (can check completion status, VBlank, frame counter)
3. **PPU.RenderFrame()** - Frame is rendered, VBlank flag set, frame counter incremented
4. **Audio generation** - Audio samples generated

This ensures all synchronization signals are available **before** your ROM code runs, so you can check them reliably.

#### Duration Modes

**Mode 0 (Stop when done):**
- Channel plays for specified duration
- When duration reaches 0, channel automatically disables
- Completion flag is set when channel disables
- Use for: Single notes, sound effects, one-shot sounds

**Mode 1 (Loop/Restart):**
- Channel plays for specified duration
- When duration reaches 0, the initial duration is automatically reloaded and the channel continues playing
- The initial duration is stored when the channel is enabled (if duration > 0)
- Use for: Looping background music, sustained notes

**Setting Duration:**
- **Duration = 0**: Channel plays indefinitely (no auto-disable)
- **Duration > 0**: Channel counts down each frame, auto-disables when reaches 0 (if mode 0)

### Musical Note Frequencies

Common musical note frequencies (in Hz):

| Note | Frequency (Hz) | Low Byte | High Byte |
|------|----------------|----------|-----------|
| C4 | 262 | 0x06 | 0x01 |
| D4 | 294 | 0x26 | 0x01 |
| E4 | 330 | 0x4A | 0x01 |
| F4 | 349 | 0x5D | 0x01 |
| G4 | 392 | 0x88 | 0x01 |
| A4 | 440 | 0xB8 | 0x01 |
| B4 | 494 | 0xEE | 0x01 |
| C5 | 523 | 0x0B | 0x02 |

**C Major Scale (Octave 4):**
- C4: 262 Hz
- D4: 294 Hz
- E4: 330 Hz
- F4: 349 Hz
- G4: 392 Hz
- A4: 440 Hz
- B4: 494 Hz
- C5: 523 Hz

### Complete Audio Example: Playing a Scale with Duration Control

This example uses the new duration-based timing system:

```
; Initialize note index (R4)
MOV R4, #0x0000           ; Start with note 0 (C4)

; Play first note immediately
; Set frequency (C4 = 262 Hz)
MOV R7, #0x9000           ; CH0_FREQ_LOW
MOV R0, #0x06             ; Low byte (262 & 0xFF)
MOV [R7], R0

MOV R7, #0x9001           ; CH0_FREQ_HIGH
MOV R0, #0x01             ; High byte (262 >> 8)
MOV [R7], R0

; Set volume
MOV R7, #0x9002           ; CH0_VOLUME
MOV R0, #0x80             ; Volume = 128
MOV [R7], R0

; Set duration to 60 frames (1 second)
MOV R7, #0x9004           ; CH0_DURATION_LOW
MOV R0, #60               ; Duration = 60 frames
MOV [R7], R0

MOV R7, #0x9005           ; CH0_DURATION_HIGH
MOV R0, #0                ; High byte = 0
MOV [R7], R0

; Set duration mode: stop when done
MOV R7, #0x9006           ; CH0_DURATION_MODE
MOV R0, #0                ; Mode 0 = stop when done
MOV [R7], R0

; NOW enable the channel
MOV R7, #0x9003           ; CH0_CONTROL
MOV R0, #0x01             ; Enable, sine wave
MOV [R7], R0

main_loop:
    ; ... your game logic ...
    
    ; Check if channel 0 just finished (using completion status)
    MOV R7, #0x9021        ; CHANNEL_COMPLETION_STATUS
    MOV R6, [R7]           ; Read completion status (16-bit read)
    AND R6, #0x01          ; Mask to bit 0 (channel 0 flag)
    CMP R6, #0             ; Compare with 0
    BEQ skip_note          ; If channel 0 didn't finish, skip update
    
    ; Channel 0 finished - start next note!
    ; Increment note index (cycle 0-7)
    ADD R4, #1
    AND R4, #0x07          ; Keep in range 0-7
    
    ; Calculate frequency: approximate scale
    ; For simplicity, use: 262 + (note_index * 32)
    MOV R7, R4             ; R7 = note index
    SHL R7, #5             ; R7 = note index * 32
    ADD R7, #262           ; R7 = 262 + (note_index * 32)
    
    ; Set frequency (low byte first, then high byte)
    MOV R0, R7             ; R0 = frequency
    AND R0, #0xFF          ; R0 = low byte
    MOV R7, #0x9000        ; CH0_FREQ_LOW
    MOV [R7], R0
    
    ; High byte: check if note index == 7 (523 Hz needs 0x02)
    CMP R4, #7
    BLT note_low
    MOV R0, #0x02          ; High byte 0x02 for note 7
    JMP note_high_done
note_low:
    MOV R0, #0x01          ; High byte 0x01 for notes 0-6
note_high_done:
    MOV R7, #0x9001        ; CH0_FREQ_HIGH
    MOV [R7], R0
    
    ; Set duration to 60 frames again
    MOV R7, #0x9004        ; CH0_DURATION_LOW
    MOV R0, #60            ; Duration = 60 frames
    MOV [R7], R0
    
    MOV R7, #0x9005        ; CH0_DURATION_HIGH
    MOV R0, #0             ; High byte = 0
    MOV [R7], R0
    
    ; Re-enable channel to start the note
    MOV R7, #0x9003        ; CH0_CONTROL
    MOV R0, #0x01          ; Enable, sine wave
    MOV [R7], R0
    
skip_note:
    JMP main_loop
```

**Key Points:**
- Duration is set **before** enabling the channel
- Completion status is checked **once per frame** in the main loop
- When channel finishes, update frequency/duration and re-enable
- No manual frame counting needed - APU handles it automatically!

### Frequency Calculation Tips

**Direct Frequency Value:**
- The APU uses direct frequency values in Hz (0-65535)
- No conversion needed - just write the frequency value directly

**Calculating Frequency from Note Index:**
- For a simple scale: `frequency = base_freq + (note_index * step)`
- Example: C major scale starting at C4 (262 Hz)
  - `frequency = 262 + (note_index * 32)` (approximate)
  - More accurate: use a lookup table or precise calculation

**Frequency Update Order:**
1. Always write **low byte first** (FREQ_LOW)
2. Then write **high byte** (FREQ_HIGH)
3. The APU updates the phase increment when the high byte is written
4. **Phase is automatically reset to 0** when frequency changes (for clean note starts)

### Retro Game Audio Best Practices

#### Phase Reset on Note Change

The APU automatically resets the phase to 0 when the frequency changes. This ensures:
- **Clean note transitions**: Each note starts from the beginning of the waveform cycle
- **No warbling**: Prevents phase discontinuities that cause warbling sounds
- **Authentic retro sound**: Matches behavior of real hardware (NES, SNES)

**Why this matters:**
- Without phase reset, changing frequency mid-cycle causes the waveform to jump to a different point
- This creates warbling, clicks, and other artifacts
- Retro game audio relies on clean note starts for musical clarity

#### Volume Envelope and Note Timing

**Proper Note Timing Sequence:**
1. **Set frequency** (low byte, then high byte)
2. **Set volume** (0-255)
3. **Enable channel** (CONTROL register, bit 0 = 1)

**Note Duration Control:**
- Use APU duration registers for automatic timing (recommended)
- Set duration in frames (60 frames = 1 second at 60 FPS)
- Check completion status register to detect when notes finish
- No manual frame counting needed!

**Volume Envelope Example:**
```
; Start note with full volume
MOV R7, #0x9002        ; CH0_VOLUME
MOV R0, #0xFF          ; Full volume
MOV [R7], R0

; ... play note for duration ...

; Fade out (optional)
MOV R7, #0x9002
MOV R0, #0x80          ; Half volume
MOV [R7], R0

; ... continue ...

; End note
MOV R7, #0x9003        ; CH0_CONTROL
MOV R0, #0x00          ; Disable channel
MOV [R7], R0
```

#### Waveform Selection for Different Sounds

**Sine Wave (0)**: 
- Smooth, pure tone
- Best for: melodies, background music, soft sounds
- Example: `CONTROL = 0x01` (enable, sine)

**Square Wave (1)**:
- Harsh, buzzy tone
- Best for: bass lines, percussion, retro game sound effects
- Example: `CONTROL = 0x03` (enable, square)

**Saw Wave (2)**:
- Bright, buzzy tone
- Best for: leads, synth sounds, special effects
- Example: `CONTROL = 0x05` (enable, saw)

**Noise (3, Channel 3 only)**:
- White noise
- Best for: percussion, explosion sounds, wind effects
- Example: `CONTROL = 0x07` (enable, noise)

#### Multi-Channel Audio

Use multiple channels for:
- **Melody + Bass**: Channel 0 for melody, Channel 1 for bass
- **Harmony**: Multiple channels playing different notes simultaneously
- **Sound Effects**: Reserve Channel 3 for sound effects (noise)

**Example: Two-Channel Harmony**
```
; Channel 0: Play C4 (262 Hz)
MOV R7, #0x9000        ; CH0_FREQ_LOW
MOV R0, #0x06          ; Low byte
MOV [R7], R0
MOV R7, #0x9001        ; CH0_FREQ_HIGH
MOV R0, #0x01          ; High byte
MOV [R7], R0
MOV R7, #0x9002        ; CH0_VOLUME
MOV R0, #0x80
MOV [R7], R0
MOV R7, #0x9004        ; CH0_DURATION_LOW
MOV R0, #0             ; Duration = 0 (play indefinitely)
MOV [R7], R0
MOV R7, #0x9005        ; CH0_DURATION_HIGH
MOV R0, #0
MOV [R7], R0
MOV R7, #0x9003        ; CH0_CONTROL
MOV R0, #0x01          ; Enable, sine
MOV [R7], R0

; Channel 1: Play E4 (330 Hz) - harmony
MOV R7, #0x9008        ; CH1_FREQ_LOW (0x9000 + 1*8 + 0)
MOV R0, #0x4A          ; Low byte
MOV [R7], R0
MOV R7, #0x9009        ; CH1_FREQ_HIGH (0x9000 + 1*8 + 1)
MOV R0, #0x01          ; High byte
MOV [R7], R0
MOV R7, #0x900A        ; CH1_VOLUME (0x9000 + 1*8 + 2)
MOV R0, #0x80
MOV [R7], R0
MOV R7, #0x900C        ; CH1_DURATION_LOW (0x9000 + 1*8 + 4)
MOV R0, #0             ; Duration = 0 (play indefinitely)
MOV [R7], R0
MOV R7, #0x900D        ; CH1_DURATION_HIGH (0x9000 + 1*8 + 5)
MOV R0, #0
MOV [R7], R0
MOV R7, #0x900B        ; CH1_CONTROL (0x9000 + 1*8 + 3)
MOV R0, #0x01          ; Enable, sine
MOV [R7], R0
```

#### Common Pitfalls and Solutions

**Problem: Channel disables immediately after enabling**
- **Cause**: Duration was set to 0, or channel was enabled before duration was set
- **Solution**: Always set frequency, volume, and duration **BEFORE** enabling the channel. Set duration > 0 for timed notes.

**Problem: Completion status always reads 0**
- **Cause**: Reading completion status after it's been cleared (it's one-shot), or channel never finishes
- **Solution**: Completion status is cleared immediately after being read (one-shot). Read it once per frame in your main loop. Ensure duration > 0 and mode = 0 (stop when done). Use VBlank or frame counter to ensure you only check once per frame.

**Problem: Notes not changing**
- **Cause**: Not checking completion status, or branch logic is wrong
- **Solution**: Read `CHANNEL_COMPLETION_STATUS` (0x9021) once per frame. Use `BEQ` to skip note update if status is 0 (channel didn't finish).

**Problem: Warbling/clicking sounds**
- **Cause**: Phase not resetting on frequency change (now fixed in APU)
- **Solution**: Ensure frequency is updated correctly (low byte first, then high byte). APU automatically resets phase when frequency changes.

**Problem: Audio sounds distorted**
- **Cause**: Volume too high, or multiple channels mixing incorrectly
- **Solution**: Lower individual channel volumes, check master volume

**Problem: Timing is inconsistent**
- **Cause**: Using old loop-counter approach instead of duration registers
- **Solution**: Use APU duration registers for automatic timing. Set duration in frames (60 frames = 1 second).

---

## Input System

### Reading Input

#### Controller 1

1. **Latch Controller**:
   ```
   MOV R1, #0xA001        ; CONTROLLER1_LATCH
   MOV R2, #0x01          ; Latch = 1
   MOV [R1], R2           ; Latch button states
   ```

2. **Read Button State**:
   ```
   MOV R1, #0xA000        ; CONTROLLER1
   MOV R2, [R1]            ; Read button state (low byte)
   ; Read high byte for L, R, START, SELECT buttons
   ADD R1, #0x0001         ; CONTROLLER1 high byte
   MOV R3, [R1]            ; Read high byte
   ; R2 = low byte, R3 = high byte
   ```

3. **Check Button**:
   ```
   MOV R1, #0xA000
   MOV R2, [R1]            ; Read buttons (low byte)
   AND R2, #0x10           ; Check A button (bit 4)
   CMP R2, #0x10
   BEQ button_pressed      ; Branch if A pressed
   ```

#### Controller 2

1. **Latch Controller 2**:
   ```
   MOV R1, #0xA003        ; CONTROLLER2_LATCH
   MOV R2, #0x01          ; Latch = 1
   MOV [R1], R2           ; Latch button states
   ```

2. **Read Controller 2 Button State**:
   ```
   MOV R1, #0xA002        ; CONTROLLER2
   MOV R2, [R1]            ; Read button state (low byte)
   ; Read high byte for L, R, START, SELECT buttons
   ADD R1, #0x0001         ; CONTROLLER2 high byte
   MOV R3, [R1]            ; Read high byte
   ; R2 = low byte, R3 = high byte
   ```

3. **Check Controller 2 Button**:
   ```
   MOV R1, #0xA002
   MOV R2, [R1]            ; Read Controller 2 buttons (low byte)
   AND R2, #0x10           ; Check A button (bit 4)
   CMP R2, #0x10
   BEQ player2_pressed     ; Branch if A pressed
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

The ROM format consists of a 32-byte header followed by executable code. The basic steps:

1. Create instruction encoding functions
2. Build code array with instructions
3. Calculate branch offsets (signed 16-bit relative offsets)
4. Pack code into bytes (little-endian)
5. Create 32-byte header with magic, version, size, entry point
6. Write header + code to file

**ROM Header Fields:**
- Magic: `0x46434D52` ("RMCF")
- Version: `1` (current format version)
- ROM Size: Size of executable code (excluding header)
- Entry Bank: Initial program bank (typically 1)
- Entry Offset: Initial program offset (typically 0x8000)
- Mapper Flags: Mapper type (0 = LoROM)
- Checksum: Optional 32-bit checksum

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

### Example 4: Matrix Mode with Large World

```
; Enable Matrix Mode for large world map
MOV R1, #0x8008         ; BG0_CONTROL
MOV R2, #0x01           ; Enable BG0
MOV [R1], R2

MOV R1, #0x8018         ; MATRIX_CONTROL
MOV R2, #0x01           ; Enable Matrix Mode
MOV [R1], R2

; Set up transformation matrix for perspective
; (Example: looking down at 45° angle)
MOV R1, #0x8019         ; MATRIX_A_L
MOV R2, #0xB5           ; A = 0.707 (low byte)
MOV [R1], R2
MOV R1, #0x801A         ; MATRIX_A_H
MOV R2, #0x00
MOV [R1], R2

; Set center point
MOV R1, #0x8027         ; MATRIX_CENTER_X_L
MOV R2, #0xA0           ; Center X = 160
MOV [R1], R2
MOV R1, #0x8029         ; MATRIX_CENTER_Y_L
MOV R2, #0x64           ; Center Y = 100
MOV [R1], R2

; Large world: Use multiple tilemaps stitched together
; World coordinates can span multiple tilemap regions
; Emulator handles seamless tile stitching automatically
```

### Example 5: Animated Moving Sprite (Best Practices)

```
; This example demonstrates proper sprite position updates with field preservation
; Key principles: Read all fields, preserve them, update position, write all fields back

main_loop:
    ; Wait for VBlank before updating sprites
    MOV R1, #0x803E      ; VBLANK_FLAG
wait_vblank:
    MOV R2, [R1]         ; Read VBlank flag
    MOV R7, #0x00
    CMP R2, R7           ; Compare with 0
    BEQ wait_vblank      ; Loop if flag is 0
    
    ; Read current sprite 0 data (preserve all fields)
    MOV R1, #0x8014      ; OAM_ADDR
    MOV R2, #0x00        ; Sprite 0
    MOV [R1], R2         ; Reset byte index to 0
    MOV R1, #0x8015      ; OAM_DATA
    
    ; Read all 6 bytes and preserve them
    MOV R2, [R1]         ; Read X low (byte 0)
    MOV R6, R2           ; Save X low to R6
    MOV R2, [R1]         ; Read X high (byte 1)
    MOV R5, R2           ; Save X high to R5
    MOV R2, [R1]         ; Read Y (byte 2)
    MOV R0, R2           ; Save Y to R0
    MOV R2, [R1]         ; Read Tile (byte 3)
    MOV R3, R2           ; Save Tile to R3
    MOV R2, [R1]         ; Read Attributes (byte 4)
    MOV R2, R2           ; Save Attributes to R2 (already in R2)
    MOV R4, [R1]         ; Read Control (byte 5)
    MOV R4, R4           ; Save Control to R4 (already in R4)
    
    ; Calculate new X position
    ADD R6, #1           ; Increment X low
    ; Check for overflow (if X low wraps to 0, increment X high)
    MOV R7, #0x00
    CMP R6, R7
    BNE no_overflow
    ADD R5, #1           ; Increment X high on overflow
no_overflow:
    
    ; Check if sprite should wrap (X >= 336, fully off-screen)
    MOV R7, #336         ; Screen width + sprite width
    CMP R6, R7           ; Compare X low with 336
    BLT no_wrap          ; Branch if X < 336 (skip wrap)
    
    ; Wrap sprite to left side (X = -16)
    MOV R6, #0xF0        ; X low = 240 (-16 in two's complement)
    MOV R5, #0x01        ; X high = 1 (sign bit set)
    
no_wrap:
    ; Write all 6 bytes back to OAM (preserve sprite state)
    MOV R1, #0x8014      ; OAM_ADDR
    MOV R2, #0x00        ; Sprite 0
    MOV [R1], R2         ; Reset byte index to 0
    MOV R1, #0x8015      ; OAM_DATA
    
    MOV R2, R6           ; X low (updated)
    MOV [R1], R2
    MOV R2, R5           ; X high (updated)
    MOV [R1], R2
    MOV R2, R0           ; Y (preserved)
    MOV [R1], R2
    MOV R2, R3           ; Tile (preserved)
    MOV [R1], R2
    MOV R2, R2           ; Attributes (preserved, already in R2)
    MOV [R1], R2
    MOV R2, #0x03        ; Control (preserved: enabled, 16x16)
    MOV [R1], R2
    
    ; Wait for VBlank to clear (so we only update once per VBlank period)
wait_vblank_clear:
    MOV R1, #0x803E      ; VBLANK_FLAG
    MOV R2, [R1]         ; Read VBlank flag
    MOV R7, #0x00
    CMP R2, R7           ; Compare with 0
    BNE wait_vblank_clear ; Loop if flag is 1 (still in VBlank)
    
    ; Jump back to main loop
    JMP main_loop
```

**Key Points from This Example:**

1. **Always wait for VBlank** before reading/writing sprite data
2. **Read all 6 bytes** before making any changes
3. **Preserve all fields** in registers (R0=Y, R3=Tile, R2=Attributes, R4=Control)
4. **Write all 6 bytes back** after updating position
5. **Always write Control byte last** to preserve sprite state
6. **Proper wrap check** with branch to skip wrap code when not needed

### Example 6: Vertical Sprite (Building in 3D World)

```
; Set up vertical sprite for building
; Sprite world coordinates: X=100, Y=200 (in tilemap space)
; Sprite will scale and position based on Matrix Mode transformation

; Store sprite world coordinates in WRAM or extended OAM
MOV R1, #0x0000          ; WRAM address for sprite world data
MOV R2, #0x0064         ; World X = 100
MOV [R1], R2
ADD R1, #0x0002
MOV R2, #0x00C8         ; World Y = 200
MOV [R1], R2

; Set sprite base attributes (normal sprite entry)
MOV R1, #0x8014         ; OAM_ADDR
MOV R2, #0x00           ; Sprite 0
MOV [R1], R2
MOV R1, #0x8015         ; OAM_DATA
MOV R2, #0x00           ; X position (calculated from world X)
MOV [R1], R2
MOV R2, #0x64           ; Y position (calculated from world Y)
MOV [R1], R2
MOV R2, #0x10           ; Tile index for building
MOV [R1], R2
MOV R2, #0x10           ; Palette 1
MOV [R1], R2
MOV R2, #0x81           ; Enable, 16x16 size, vertical sprite flag
MOV [R1], R2

; Emulator will:
; 1. Read world coordinates from WRAM or extended OAM
; 2. Transform to screen coordinates using inverse matrix
; 3. Calculate scale based on world Y
; 4. Render sprite at calculated position with scale
```

### Example 7: Bouncing Box Animation

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
| MOV | 0x1000 | 0-7 | Move/load/store |
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
| VRAM_ADDR | 0x800E-0x800F | VRAM address (16-bit) |
| VRAM_DATA | 0x8010 | VRAM data (8-bit, auto-increment) |
| CGRAM_ADDR | 0x8012 | Palette address (8-bit, 0-255) |
| CGRAM_DATA | 0x8013 | Palette data (RGB555, two 8-bit writes: low byte first, then high byte) |
| MATRIX_CONTROL | 0x8018 | Matrix Mode enable and mirroring |
| MATRIX_A/B/C/D | 0x8019-0x8020 | Transformation matrix (16-bit each) |
| MATRIX_CENTER_X/Y | 0x8027-0x802A | Center point (16-bit each) |
| VBLANK_FLAG | 0x803E | VBlank flag (bit 0 = VBlank active, one-shot, cleared when read) |
| FRAME_COUNTER | 0x803F-0x8040 | Frame counter (16-bit, increments once per frame) |

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
- I/O addresses (0x8000+) always use bank 0, regardless of DBR
- Matrix Mode uses 8.8 fixed point (1.0 = 0x0100)
- **Large Worlds**: Matrix Mode supports tilemaps larger than 32×32 tiles through seamless tile stitching
- **Vertical Sprites**: Sprites can have world coordinates and are rendered with 3D perspective in Matrix Mode

---

**End of Nitro-Core-DX Programming Manual**

