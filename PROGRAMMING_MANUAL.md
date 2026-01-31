# Nitro Core DX Programming Manual

**Version 2.0**  
**Last Updated: January 27, 2026**

> **✅ Architecture Stable**: The core hardware architecture is complete and stable. All hardware features are implemented and tested. You can develop ROMs with confidence—the API is stable and future enhancements will maintain backward compatibility.

---

## Table of Contents

1. [Design Philosophy](#design-philosophy)
2. [Getting Started](#getting-started)
3. [CoreLX: The Primary Language](#corelx-the-primary-language)
4. [Assembly: The Low-Level Complement](#assembly-the-low-level-complement)
5. [Mixing CoreLX and Assembly](#mixing-corelx-and-assembly)
6. [Hardware Reference](#hardware-reference)
7. [Complete Examples](#complete-examples)
8. [Reference Tables](#reference-tables)

---

## Design Philosophy

### The Vision: Two Languages, One Goal

Nitro Core DX provides **two complementary programming approaches** that work together seamlessly:

- **CoreLX**: A high-level, expressive language designed for productivity and clarity
- **Assembly**: Direct hardware control for maximum performance and precision

Both compile to the same machine code. Both run on the same hardware. Both can be mixed in a single file.

### Why Two Languages?

**CoreLX exists because:**
- Most game logic doesn't need assembly-level control
- High-level abstractions make code readable and maintainable
- Type safety catches bugs at compile time
- Built-in functions provide clean hardware APIs
- Indentation-based syntax reduces boilerplate

**Assembly exists because:**
- Sometimes you need cycle-perfect timing
- Some hardware features aren't yet wrapped in CoreLX functions
- Performance-critical code benefits from direct control
- Learning assembly helps you understand the hardware
- Legacy code or examples may be written in assembly

### The Philosophy: Start High, Go Low When Needed

**Recommended approach:**
1. **Start with CoreLX** - Write your game logic in CoreLX
2. **Use built-in functions** - Leverage `ppu.*`, `sprite.*`, `apu.*` APIs
3. **Drop to assembly when needed** - For unimplemented features or performance-critical sections
4. **Mix seamlessly** - Use inline assembly blocks within CoreLX functions

**Example workflow:**
```corelx
-- CoreLX for game logic
function Start()
    ppu.enable_display()
    hero := Sprite()
    sprite.set_pos(&hero, 120, 80)
    
    -- Assembly for precise timing
    asm {
        MOV R1, #0x803E  -- VBLANK_FLAG
        MOV R2, [R1]
        CMP R2, #0
        BEQ wait_vblank
    }
    
    -- Back to CoreLX
    oam.write(0, &hero)
end
```

### Design Principles

**1. Hardware-First**
- Both languages provide direct hardware access
- No virtual machines, no interpreters, no runtime overhead
- Compiled to native machine code

**2. Zero-Cost Abstractions**
- CoreLX functions compile to efficient assembly
- High-level code doesn't sacrifice performance
- Assembly is available when you need it

**3. Learnable Progression**
- Start simple with CoreLX
- Learn assembly gradually as needed
- Both languages use the same hardware concepts

**4. Practical Flexibility**
- Use the right tool for each task
- Mix languages naturally
- No artificial boundaries

---

## Getting Started

### Your First Program: CoreLX

Every Nitro Core DX program **must** define a `Start()` function. Here's the simplest possible program:

```corelx
function Start()
    ppu.enable_display()
    
    while true
        wait_vblank()
```

**Compile and run:**
```bash
corelx game.corelx game.rom
./nitro-core-dx -rom game.rom
```

### Your First Program: Assembly

The same program in assembly:

```assembly
; Entry point
MOV R1, #0x8008        ; BG0_CONTROL
MOV R2, #0x01          ; Enable display
MOV [R1], R2

main_loop:
    MOV R1, #0x803E    ; VBLANK_FLAG
wait_vblank:
    MOV R2, [R1]       ; Read flag
    CMP R2, #0
    BEQ wait_vblank    ; Loop until VBlank
    
    JMP main_loop
```

**Compile and run:**
```bash
# Assembly compilation (using rombuilder or similar tool)
./rombuilder game.asm game.rom
./nitro-core-dx -rom game.rom
```

### Choosing Your Path

**Use CoreLX if:**
- You're new to Nitro Core DX
- You want readable, maintainable code
- You're building game logic, not hardware drivers
- You prefer type safety and compile-time checks

**Use Assembly if:**
- You need precise cycle timing
- You're implementing a feature not yet in CoreLX
- You're porting existing assembly code
- You want to understand the hardware deeply

**Mix both if:**
- You want the best of both worlds (recommended!)
- You're building a complex project
- You need assembly for specific sections

---

## CoreLX: The Primary Language

> **CoreLX** (pronounced *core elix*) is the native compiled programming language for Nitro Core DX.  
> CoreLX is **compiled-only**, **hardware-first** with **no interpreter**, **no virtual machine**, and **no runtime scripting layer**.  
> Each CoreLX source file produces **one ROM image** that runs directly on Nitro Core DX.

> **📖 Quick Reference**: For a focused CoreLX language reference with detailed syntax, examples, and built-in function documentation, see [docs/CORELX.md](../docs/CORELX.md). This section provides an overview; the full reference has comprehensive details.

### Implementation Status

**Current Status**: CoreLX is **functional for basic programs** but has some incomplete features.

**✅ Fully Working:**
- Basic syntax and control flow
- Sprite positioning (`sprite.set_pos()`)
- OAM operations (`oam.write()`, `oam.flush()`)
- VBlank synchronization (`wait_vblank()`)
- Display control (`ppu.enable_display()`)
- All sprite helpers (`SPR_PAL()`, `SPR_PRI()`, `SPR_HFLIP()`, `SPR_VFLIP()`, `SPR_ENABLE()`, `SPR_SIZE_8()`, `SPR_SIZE_16()`, `SPR_BLEND()`, `SPR_ALPHA()`)
- Frame counter (`frame_counter()`)
- All APU functions (`apu.enable()`, `apu.set_channel_wave()`, `apu.set_channel_freq()`, `apu.set_channel_volume()`, `apu.note_on()`, `apu.note_off()`)

**⚠️ Partially Implemented:**
- `gfx.load_tiles()` - Returns base parameter, doesn't load asset data yet
- Asset system - Assets can be declared but not embedded into ROM

**❌ Not Yet Implemented:**
- User-defined functions (only `Start()` works)
- Proper variable storage and register allocation (simplified implementation exists)

**Workaround**: For unimplemented features, use assembly (see [Assembly: The Low-Level Complement](#assembly-the-low-level-complement)).

### Language Overview

#### Core Principles

CoreLX is designed to feel:
- **Magical on the surface** - Simple, expressive syntax
- **Strict underneath** - Type-safe, compile-time checked
- **Powerful without being vague** - Direct hardware access
- **Learnable without dumbing anything down** - Clear, precise documentation

#### Key Features

- **Indentation-based syntax** - No braces, no semicolons
- **Compiled to machine code** - Direct Nitro Core DX execution
- **Hardware-accurate** - Direct access to PPU, APU, OAM, VRAM
- **Single-file compilation** - One `.corelx` file = one ROM
- **Inline assets** - Embed graphics and data directly in source

### Syntax Basics

#### Indentation Rules

CoreLX uses **indentation** to define blocks. Indentation is **mandatory** and **authoritative**.

- **Increased indentation** → enter a block
- **Decreased indentation** → exit a block
- **Same indentation** → same block level
- **Tabs OR spaces** are allowed, but **must not be mixed** in the same file

```corelx
function Start()
    -- This is indented (inside Start)
    if x == 5
        -- This is more indented (inside if)
        y = 10
    -- Back to Start() level
    z = 20
```

#### Comments

```corelx
-- This is a comment until end of line
```

#### No Braces, No Semicolons

CoreLX has **no** `{ }` braces and **no** `;` semicolons.  
Newlines end statements. Indentation defines blocks.

### Types

#### Built-in Types

| Type | Size | Description |
|------|------|-------------|
| `i8` | 8-bit | Signed integer (-128 to 127) |
| `u8` | 8-bit | Unsigned integer (0 to 255) |
| `i16` | 16-bit | Signed integer (-32768 to 32767) |
| `u16` | 16-bit | Unsigned integer (0 to 65535) |
| `i32` | 32-bit | Signed integer |
| `u32` | 32-bit | Unsigned integer |
| `bool` | 1-bit | Boolean (true/false) |
| `fx8_8` | 16-bit | Fixed-point (8.8 format) |
| `fx16_16` | 32-bit | Fixed-point (16.16 format) |

#### Pointer Types

```corelx
x: *u8      -- Pointer to u8
sprite: *Sprite  -- Pointer to Sprite struct
```

#### Type Inference

You can declare variables without explicit types:

```corelx
x := 10        -- Inferred as i16 (smallest signed type)
y: u8 = 5      -- Explicitly u8
```

### Variables and Assignment

```corelx
x := 10              -- Declare with inferred type
y: u8 = 5            -- Declare with explicit type
z: i16               -- Declare without initializer (zero-initialized)

x = x + 1             -- Assignment
y = 42
```

### Control Flow

#### If Statements

```corelx
if x == 5
    y = 10
elseif x == 6
    y = 20
else
    y = 30
```

#### While Loops

```corelx
while x < 100
    x = x + 1
    do_something()
```

#### For Loops

```corelx
for i := 0; i < 10; i = i + 1
    print(i)
```

### Functions

#### Function Declaration

```corelx
function add(a: i16, b: i16) -> i16
    return a + b
```

#### Void Functions

```corelx
function do_something()
    -- No return type = void
    x = 10
```

#### Entry Point: Start()

Every CoreLX program **must** define:

```corelx
function Start()
    -- Your game code here
```

`Start()` is a **reserved system entry point**:
- Exactly one `Start()` must exist
- `Start()` may not be overloaded, aliased, shadowed, or reassigned
- Any violation is a compile-time error

### Structs

#### Struct Declaration

```corelx
type Vec2 = struct
    x: i16
    y: i16
```

#### Struct Usage

```corelx
pos := Vec2()
pos.x = 10
pos.y = 20
```

#### The Sprite Struct

CoreLX provides a canonical `Sprite` struct that maps 1:1 to OAM:

```corelx
type Sprite = struct
    x_lo: u8     -- OAM byte 0
    x_hi: u8     -- OAM byte 1
    y: u8        -- OAM byte 2
    tile: u8     -- OAM byte 3
    attr: u8     -- OAM byte 4
    ctrl: u8     -- OAM byte 5
```

This struct **must match hardware layout exactly**.

### Assets

#### Inline Asset Blocks

CoreLX supports **inline asset blocks** at the top level:

```corelx
asset HeroTiles: tiles8
    hex
        00 00 11 11 22 22 33 33
        44 44 55 55 66 66 77 77
```

#### Asset Types

- `tiles8` - 8×8 pixel tiles
- `tiles16` - 16×16 pixel tiles

#### Asset Encodings

- `hex` - Hexadecimal data
- `b64` - Base64 encoded data

#### Using Assets

> **⚠️ Status**: Asset embedding into ROM is **not yet fully implemented**. Assets can be declared, but `gfx.load_tiles()` doesn't actually load asset data yet.

The compiler emits constants for each asset:

```corelx
base := gfx.load_tiles(ASSET_HeroTiles, 0)
```

`ASSET_<Name>` is a `u16` constant representing the asset.

**Current Limitation**: Asset data is parsed but not embedded into the ROM. The `gfx.load_tiles()` function currently returns the `base` parameter without loading actual asset data.

### Sprites and OAM

#### Sprite Helper Functions

```corelx
function sprite.set_pos(s: *Sprite, x: i16, y: u8)
```

#### Sprite Attribute Helpers

**All Implemented:**

```corelx
SPR_PAL(p: u8) -> u8        -- Set palette (0-15) ✅
SPR_PRI(p: u8) -> u8        -- Set priority (shifts to bits [7:6]) ✅
SPR_HFLIP() -> u8           -- Horizontal flip (returns 0x10) ✅
SPR_VFLIP() -> u8           -- Vertical flip (returns 0x20) ✅
SPR_ENABLE() -> u8          -- Enable sprite (returns 0x01) ✅
SPR_SIZE_8() -> u8          -- 8×8 size (returns 0x00) ✅
SPR_SIZE_16() -> u8         -- 16×16 size (returns 0x02) ✅
SPR_BLEND(mode: u8) -> u8   -- Blending mode (shifts to bits [3:2]) ✅
SPR_ALPHA(a: u8) -> u8      -- Alpha value (shifts to bits [7:4]) ✅
```

#### OAM API

```corelx
function oam.write(id: u8, s: *Sprite)
function oam.flush()
```

**Important**: OAM writes must occur during VBlank. Always call `wait_vblank()` before updating sprites.

### Audio (APU)

> **⚠️ Status**: APU functions are **not yet implemented** in the CoreLX compiler. Use assembly for audio programming until these are added.

Audio is **hardware-driven**, not software-mixed.

#### APU API (Future Feature)

**Note**: These functions are registered but not yet implemented in code generation. Programs using them will fail to compile.

```corelx
function apu.enable()                                    -- ⚠️ Not implemented
function apu.set_channel_wave(ch: u8, wave: u8)         -- ⚠️ Not implemented
function apu.set_channel_freq(ch: u8, freq: u16)        -- ⚠️ Not implemented
function apu.set_channel_volume(ch: u8, vol: u8)         -- ⚠️ Not implemented
function apu.note_on(ch: u8)                            -- ⚠️ Not implemented
function apu.note_off(ch: u8)                           -- ⚠️ Not implemented
```

**Workaround**: Use assembly for audio programming (see [Assembly: The Low-Level Complement](#assembly-the-low-level-complement)).

**No PCM streaming**. **No software synthesis**.  
Audio is generated by hardware channels.

### Built-in Functions

#### Frame Synchronization

```corelx
function wait_vblank()                    -- ✅ Fully implemented
function frame_counter() -> u32           -- ✅ Fully implemented (reads actual frame counter)
```

#### Graphics

```corelx
function ppu.enable_display()             -- ✅ Fully implemented
function gfx.load_tiles(asset: u16, base: u16) -> u16  -- ⚠️ Returns base parameter, doesn't load asset data yet
```

**Note**: `gfx.load_tiles()` currently returns the `base` parameter. Asset embedding into ROM is not yet implemented, so assets can be declared but not fully utilized.

### Complete CoreLX Example

Here's a complete CoreLX program demonstrating all major features:

```corelx
asset HeroTiles: tiles8
    hex
        00 00 11 11 22 22 33 33
        44 44 55 55 66 66 77 77

function Start()
    ppu.enable_display()

    base := gfx.load_tiles(ASSET_HeroTiles, 0)

    hero := Sprite()
    sprite.set_pos(&hero, 120, 80)
    hero.tile = base
    hero.attr = SPR_PAL(1) | SPR_PRI(2)
    hero.ctrl = SPR_ENABLE() | SPR_SIZE_16()

    while true
        wait_vblank()
        oam.write(0, &hero)
        oam.flush()
```

---

## Assembly: The Low-Level Complement

> **Assembly** provides direct, precise control over the Nitro Core DX hardware.  
> Use assembly when you need cycle-perfect timing, unimplemented features, or maximum performance.

### When to Use Assembly

**Use assembly for:**
- **Precise timing** - Cycle-perfect synchronization
- **Unimplemented features** - Hardware features not yet wrapped in CoreLX
- **Performance-critical code** - Hot loops that need optimization
- **Learning** - Understanding how the hardware actually works
- **Legacy code** - Porting existing assembly examples

**Don't use assembly for:**
- General game logic (use CoreLX)
- Code that doesn't need precise timing
- When CoreLX functions already exist

### CPU Architecture

#### Registers

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
- **D (Division by Zero)**: Set when division by zero occurs

### Instruction Set

#### Arithmetic Instructions

**ADD - Add**
```assembly
ADD R1, R2        -- R1 = R1 + R2
ADD R1, #imm      -- R1 = R1 + immediate
```

**SUB - Subtract**
```assembly
SUB R1, R2        -- R1 = R1 - R2
SUB R1, #imm      -- R1 = R1 - immediate
```

**MUL - Multiply**
```assembly
MUL R1, R2        -- R1 = (R1 * R2) & 0xFFFF
MUL R1, #imm      -- R1 = (R1 * immediate) & 0xFFFF
```

**DIV - Divide**
```assembly
DIV R1, R2        -- R1 = R1 / R2
DIV R1, #imm      -- R1 = R1 / immediate
-- Sets D flag if divisor is 0
```

#### Logical Instructions

**AND - Bitwise AND**
```assembly
AND R1, R2        -- R1 = R1 & R2
AND R1, #imm      -- R1 = R1 & immediate
```

**OR - Bitwise OR**
```assembly
OR R1, R2         -- R1 = R1 | R2
OR R1, #imm       -- R1 = R1 | immediate
```

**XOR - Bitwise XOR**
```assembly
XOR R1, R2        -- R1 = R1 ^ R2
XOR R1, #imm      -- R1 = R1 ^ immediate
```

**NOT - Bitwise NOT**
```assembly
NOT R1            -- R1 = ~R1
```

#### Shift Instructions

**SHL - Shift Left**
```assembly
SHL R1, R2        -- R1 = R1 << R2
SHL R1, #imm      -- R1 = R1 << immediate
```

**SHR - Shift Right**
```assembly
SHR R1, R2        -- R1 = R1 >> R2
SHR R1, #imm      -- R1 = R1 >> immediate
```

#### Data Movement Instructions

**MOV - Move/Load/Store**
```assembly
MOV R1, R2        -- Register to register
MOV R1, #imm      -- Immediate to register
MOV R1, [R2]      -- Load from memory (auto-detects 8/16-bit for I/O)
MOV [R1], R2      -- Store to memory (auto-detects 8/16-bit for I/O)
PUSH R1           -- Push to stack
POP R1            -- Pop from stack
```

**I/O Register Access:**
- Reading from I/O addresses (bank 0, offset >= 0x8000) automatically reads 8-bit and zero-extends
- Writing to I/O addresses automatically writes only low 8 bits
- This makes I/O access seamless - no special modes needed

#### Comparison and Branching

**CMP - Compare**
```assembly
CMP R1, R2        -- Sets flags based on (R1 - R2)
CMP R1, #imm      -- Sets flags based on (R1 - immediate)
```

**Branch Instructions:**
```assembly
BEQ offset        -- Branch if Equal (Z flag set)
BNE offset        -- Branch if Not Equal (Z flag clear)
BGT offset        -- Branch if Greater Than (signed)
BLT offset        -- Branch if Less Than (signed)
BGE offset        -- Branch if Greater or Equal (signed)
BLE offset        -- Branch if Less or Equal (signed)
```

#### Jump and Call Instructions

```assembly
JMP offset        -- Unconditional jump (relative)
CALL offset       -- Subroutine call (pushes return address)
RET               -- Return from subroutine
```

### Memory Map

#### Memory Layout

- **Bank 0**: WRAM (Work RAM) - 64KB
  - `0x0000-0x7FFF`: Work RAM (32KB)
  - `0x8000-0xFFFF`: I/O Registers
- **Banks 1-125**: ROM space (LoROM-like mapping)
  - ROM appears at `0x8000-0xFFFF` in each bank
- **Banks 126-127**: Extended WRAM (128KB)

#### I/O Register Access

**Important:** All I/O registers are 8-bit only. The CPU automatically handles this:
- **Mode 2 (`MOV R1, [R2]`)**: When reading from I/O addresses (bank 0, offset >= 0x8000), automatically reads 8-bit and zero-extends to 16-bit
- **Mode 3 (`MOV [R1], R2`)**: When writing to I/O addresses, automatically writes only the low byte

**Example:**
```assembly
MOV R4, #0x803E        ; VBLANK_FLAG address
MOV R5, [R4]           ; Automatically reads 8-bit, zero-extends
CMP R5, #0
BEQ wait_vblank        ; Loop if flag is 0
```

### Complete Assembly Example

Here's a complete assembly program equivalent to the CoreLX example:

```assembly
; Enable display
MOV R1, #0x8008        ; BG0_CONTROL
MOV R2, #0x01          ; Enable BG0
MOV [R1], R2

; Set up sprite 0
MOV R1, #0x8014        ; OAM_ADDR
MOV R2, #0x00          ; Sprite 0
MOV [R1], R2

MOV R1, #0x8015        ; OAM_DATA
MOV R2, #120           ; X low = 120
MOV [R1], R2
MOV R2, #0x00          ; X high = 0
MOV [R1], R2
MOV R2, #80            ; Y = 80
MOV [R1], R2
MOV R2, #0x00          ; Tile = 0
MOV [R1], R2
MOV R2, #0x21          ; Palette 1, Priority 2
MOV [R1], R2
MOV R2, #0x03          ; Enable, 16x16
MOV [R1], R2

main_loop:
    ; Wait for VBlank
    MOV R1, #0x803E    ; VBLANK_FLAG
wait_vblank:
    MOV R2, [R1]
    CMP R2, #0
    BEQ wait_vblank
    
    ; Update sprite (OAM already set up)
    ; In a real program, you'd update position here
    
    JMP main_loop
```

---

## Mixing CoreLX and Assembly

> **The Power of Both Worlds**: Use CoreLX for clarity, drop to assembly when needed.

### Inline Assembly Blocks

**Future Feature**: The compiler will support inline assembly blocks within CoreLX functions:

```corelx
function Start()
    ppu.enable_display()
    
    -- CoreLX code
    hero := Sprite()
    sprite.set_pos(&hero, 120, 80)
    
    -- Inline assembly for precise timing
    asm {
        MOV R1, #0x803E  -- VBLANK_FLAG
        MOV R2, [R1]
        CMP R2, #0
        BEQ wait_vblank
    }
    
    -- Back to CoreLX
    oam.write(0, &hero)
end
```

### Variable Access in Assembly

**Future Feature**: Assembly blocks will be able to access CoreLX variables:

```corelx
function update_sprite(x: i16, y: u8)
    -- Assembly can use function parameters
    asm {
        MOV R1, #0x8014  -- OAM_ADDR
        MOV R2, #0x00    -- Sprite 0
        MOV [R1], R2
        
        MOV R1, #0x8015  -- OAM_DATA
        MOV R2, x        -- Use CoreLX variable
        MOV [R1], R2
        MOV R2, y        -- Use CoreLX variable
        MOV [R1], R2
    }
end
```

### Calling Assembly from CoreLX

**Future Feature**: Assembly functions can be called from CoreLX:

```corelx
-- Assembly function (defined separately or inline)
asm function wait_vblank_asm()
    MOV R1, #0x803E
wait_loop:
    MOV R2, [R1]
    CMP R2, #0
    BEQ wait_loop
end

-- CoreLX code calls assembly function
function Start()
    ppu.enable_display()
    
    while true
        wait_vblank_asm()  -- Call assembly function
        -- CoreLX game logic here
end
```

### Current Workaround: Separate Files

**Current approach** (until inline assembly is implemented):

1. **Write CoreLX code** in `.corelx` files
2. **Write assembly code** in `.asm` files
3. **Link them together** during ROM building

**Example structure:**
```
game.corelx      -- Main game logic
timing.asm       -- Assembly timing functions
```

The ROM builder combines both into a single ROM.

### Best Practices for Mixing

**1. Use CoreLX by Default**
- Write most code in CoreLX
- Use built-in functions when available
- Keep code readable and maintainable

**2. Use Assembly Selectively**
- Only when CoreLX doesn't have the feature
- For performance-critical sections
- For precise timing requirements

**3. Document the Boundary**
- Comment why assembly is needed
- Explain what the assembly does
- Show the CoreLX equivalent if possible

**4. Keep Assembly Small**
- Prefer small, focused assembly functions
- Don't rewrite entire programs in assembly
- Use assembly as a tool, not a requirement

---

## Hardware Reference

This section provides detailed hardware reference for both CoreLX and assembly programmers.

### PPU (Graphics System)

#### Display

- **Resolution**: 320×200 pixels (landscape) / 200×320 (portrait)
- **Color Depth**: 256 colors (8-bit indexed)
- **Palette**: 256-color CGRAM (RGB555 format)

#### Background Layers

Four tile-based background layers (BG0, BG1, BG2, BG3):

- **Tile Size**: 8×8 or 16×16 pixels (configurable per layer)
- **Tile Format**: 4bpp (4 bits per pixel, 16 colors per tile)
- **Tilemap**: 64×64 tiles (512×512 pixels for 8×8 tiles)
- **Scrolling**: Independent X/Y scroll per layer
- **Priority**: BG3 (highest) → BG2 → BG1 → BG0 (lowest)

#### Sprites

- **Max Sprites**: 128 sprites
- **Size**: 8×8 or 16×16 pixels (per sprite)
- **Attributes**: X/Y position, tile index, palette, priority, flip X/Y, blend mode, alpha
- **Color Limit**: 15 visible colors per sprite (color index 0 is transparent)

#### Matrix Mode (Mode 7-Style Effects)

Matrix Mode enables advanced perspective and rotation effects on **any background layer** (BG0-BG3).

**Features:**
- Per-layer matrix transformations
- Rotation, scaling, perspective
- Per-scanline HDMA updates
- Large world map support

**CoreLX API** (future):
```corelx
ppu.set_matrix_mode(layer: u8, enable: bool)
ppu.set_matrix(layer: u8, a: fx8_8, b: fx8_8, c: fx8_8, d: fx8_8)
ppu.set_matrix_center(layer: u8, x: i16, y: i16)
```

**Assembly Registers:**
- `MATRIX_CONTROL` (0x8018): Enable Matrix Mode
- `MATRIX_A/B/C/D` (0x8019-0x8020): Transformation matrix (8.8 fixed point)
- `MATRIX_CENTER_X/Y` (0x8027-0x802A): Center point

### APU (Audio System)

#### Audio Channels

4 independent audio channels:
- **Channels 0-2**: Sine, Square, or Saw waveform
- **Channel 3**: Square or Noise waveform

#### CoreLX API

```corelx
function apu.enable()
function apu.set_channel_wave(ch: u8, wave: u8)
function apu.set_channel_freq(ch: u8, freq: u16)
function apu.set_channel_volume(ch: u8, vol: u8)
function apu.note_on(ch: u8)
function apu.note_off(ch: u8)
```

#### Assembly Registers

**Per-Channel Registers** (8 bytes per channel):
- Channel 0: 0x9000-0x9007
- Channel 1: 0x9008-0x900F
- Channel 2: 0x9010-0x9017
- Channel 3: 0x9018-0x901F

**Register Layout:**
- +0: FREQ_LOW (8-bit)
- +1: FREQ_HIGH (8-bit)
- +2: VOLUME (8-bit, 0-255)
- +3: CONTROL (8-bit, bit 0=enable, bits 1-2=waveform)
- +4: DURATION_LOW (8-bit)
- +5: DURATION_HIGH (8-bit)
- +6: DURATION_MODE (8-bit)

**Waveform Types:**
- `0` = Sine
- `1` = Square
- `2` = Saw
- `3` = Noise

### Input System

#### CoreLX API (future)

```corelx
function input.read(controller: u8) -> u16
function input.pressed(controller: u8, button: u8) -> bool
```

#### Assembly Registers

- `CONTROLLER1` (0xA000): Controller 1 button state (low byte)
- `CONTROLLER1_LATCH` (0xA001): Controller 1 latch / high byte
- `CONTROLLER2` (0xA002): Controller 2 button state (low byte)
- `CONTROLLER2_LATCH` (0xA003): Controller 2 latch / high byte

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
- Bit 11: Z (high byte)

### Memory Map

#### CoreLX Memory Access (future)

```corelx
function mem.read8(addr: u24) -> u8
function mem.read16(addr: u24) -> u16
function mem.write8(addr: u24, value: u8)
function mem.write16(addr: u24, value: u16)
```

#### Assembly Memory Access

```assembly
MOV R1, #address       -- Load address
MOV R2, [R1]           -- Read 16-bit (or 8-bit for I/O)
MOV [R1], R2           -- Write 16-bit (or 8-bit for I/O)
```

---

## Complete Examples

### Example 1: Moving Sprite (CoreLX)

```corelx
asset HeroTiles: tiles8
    hex
        00 00 11 11 22 22 33 33
        44 44 55 55 66 66 77 77

function Start()
    ppu.enable_display()
    
    base := gfx.load_tiles(ASSET_HeroTiles, 0)
    
    hero := Sprite()
    sprite.set_pos(&hero, 120, 80)
    hero.tile = base
    hero.attr = SPR_PAL(1) | SPR_PRI(2)
    hero.ctrl = SPR_ENABLE() | SPR_SIZE_16()
    
    x: i16 = 120
    
    while true
        wait_vblank()
        
        -- Move sprite
        x = x + 1
        if x > 336
            x = -16
        sprite.set_pos(&hero, x, 80)
        
        oam.write(0, &hero)
        oam.flush()
```

### Example 2: Moving Sprite (Assembly)

```assembly
; Enable display
MOV R1, #0x8008        ; BG0_CONTROL
MOV R2, #0x01
MOV [R1], R2

; Initialize sprite position
MOV R3, #120           ; X position (stored in R3)

main_loop:
    ; Wait for VBlank
    MOV R1, #0x803E    ; VBLANK_FLAG
wait_vblank:
    MOV R2, [R1]
    CMP R2, #0
    BEQ wait_vblank
    
    ; Update sprite position
    ADD R3, #1         ; Increment X
    MOV R4, #336      ; Check if off-screen
    CMP R3, R4
    BLT no_wrap
    MOV R3, #0xF0     ; Wrap to -16 (0xF0 = 240, sign-extends)
    MOV R4, #0x01     ; Set sign bit
    MOV R5, R4        ; Store X high
    JMP update_sprite
no_wrap:
    MOV R5, #0x00     ; X high = 0
update_sprite:
    ; Write sprite to OAM
    MOV R1, #0x8014    ; OAM_ADDR
    MOV R2, #0x00      ; Sprite 0
    MOV [R1], R2
    
    MOV R1, #0x8015    ; OAM_DATA
    MOV R2, R3         ; X low
    MOV [R1], R2
    MOV R2, R5         ; X high
    MOV [R1], R2
    MOV R2, #80        ; Y
    MOV [R1], R2
    MOV R2, #0x00      ; Tile
    MOV [R1], R2
    MOV R2, #0x21      ; Palette 1, Priority 2
    MOV [R1], R2
    MOV R2, #0x03      ; Enable, 16x16
    MOV [R1], R2
    
    JMP main_loop
```

### Example 3: Audio Scale (CoreLX)

```corelx
function Start()
    apu.enable()
    
    notes: [u16] = [262, 294, 330, 349, 392, 440, 494, 523]  -- C major scale
    note_index: u8 = 0
    
    -- Play first note
    apu.set_channel_freq(0, notes[note_index])
    apu.set_channel_volume(0, 128)
    apu.set_channel_wave(0, 0)  -- Sine
    apu.note_on(0)
    
    while true
        wait_vblank()
        
        -- Check if note finished (future: completion status API)
        -- For now, use frame counter
        frame := frame_counter()
        if (frame % 60) == 0  -- Every 60 frames (1 second)
            apu.note_off(0)
            note_index = (note_index + 1) % 8
            apu.set_channel_freq(0, notes[note_index])
            apu.note_on(0)
```

### Example 4: Audio Scale (Assembly)

```assembly
; Initialize note index
MOV R4, #0x0000         ; Note index = 0

; Play first note (C4 = 262 Hz)
MOV R7, #0x9000         ; CH0_FREQ_LOW
MOV R0, #0x06           ; Low byte (262 & 0xFF)
MOV [R7], R0
MOV R7, #0x9001         ; CH0_FREQ_HIGH
MOV R0, #0x01           ; High byte (262 >> 8)
MOV [R7], R0
MOV R7, #0x9002         ; CH0_VOLUME
MOV R0, #0x80           ; Volume = 128
MOV [R7], R0
MOV R7, #0x9003         ; CH0_CONTROL
MOV R0, #0x01           ; Enable, sine wave
MOV [R7], R0

main_loop:
    ; Wait for VBlank
    MOV R1, #0x803E      ; VBLANK_FLAG
wait_vblank:
    MOV R2, [R1]
    CMP R2, #0
    BEQ wait_vblank
    
    ; Check frame counter (every 60 frames = 1 second)
    MOV R1, #0x803F      ; FRAME_COUNTER_LOW
    MOV R2, [R1]         ; Read frame counter
    AND R2, #0x3F        ; Mask to 6 bits (check if divisible by 64, approximate)
    CMP R2, #0x00
    BNE skip_note
    
    ; Note finished - play next note
    ADD R4, #1           ; Increment note index
    AND R4, #0x07        ; Keep in range 0-7
    
    ; Calculate frequency: 262 + (note_index * 32) (approximate)
    MOV R7, R4
    SHL R7, #5           ; R7 = note_index * 32
    ADD R7, #262         ; R7 = 262 + (note_index * 32)
    
    ; Set frequency
    MOV R0, R7
    AND R0, #0xFF        ; Low byte
    MOV R7, #0x9000      ; CH0_FREQ_LOW
    MOV [R7], R0
    MOV R0, #0x01        ; High byte (simplified)
    MOV R7, #0x9001      ; CH0_FREQ_HIGH
    MOV [R7], R0
    
    ; Re-enable channel
    MOV R7, #0x9003      ; CH0_CONTROL
    MOV R0, #0x01        ; Enable, sine
    MOV [R7], R0
    
skip_note:
    JMP main_loop
```

---

## Reference Tables

### CoreLX Built-in Functions

| Function | Description |
|----------|-------------|
| `wait_vblank()` | Wait for vertical blanking period |
| `frame_counter() -> u32` | Get current frame counter |
| `ppu.enable_display()` | Enable PPU display |
| `gfx.load_tiles(asset, base) -> u16` | Load tiles from asset to VRAM |
| `sprite.set_pos(s, x, y)` | Set sprite position |
| `oam.write(id, sprite)` | Write sprite to OAM |
| `oam.flush()` | Flush OAM updates |
| `apu.enable()` | Enable APU |
| `apu.set_channel_wave(ch, wave)` | Set channel waveform |
| `apu.set_channel_freq(ch, freq)` | Set channel frequency |
| `apu.set_channel_volume(ch, vol)` | Set channel volume |
| `apu.note_on(ch)` | Start note on channel |
| `apu.note_off(ch)` | Stop note on channel |

### Assembly Instruction Quick Reference

| Instruction | Opcode | Description |
|------------|--------|-------------|
| NOP | 0x0000 | No operation |
| MOV | 0x1000 | Move/load/store |
| ADD | 0x2000 | Add |
| SUB | 0x3000 | Subtract |
| MUL | 0x4000 | Multiply |
| DIV | 0x5000 | Divide |
| AND | 0x6000 | Bitwise AND |
| OR | 0x7000 | Bitwise OR |
| XOR | 0x8000 | Bitwise XOR |
| NOT | 0x9000 | Bitwise NOT |
| SHL | 0xA000 | Shift left |
| SHR | 0xB000 | Shift right |
| CMP | 0xC000 | Compare |
| BEQ | 0xC100 | Branch if equal |
| BNE | 0xC200 | Branch if not equal |
| BGT | 0xC300 | Branch if greater |
| BLT | 0xC400 | Branch if less |
| BGE | 0xC500 | Branch if >= |
| BLE | 0xC600 | Branch if <= |
| JMP | 0xD000 | Jump |
| CALL | 0xE000 | Call subroutine |
| RET | 0xF000 | Return |

### PPU Register Quick Reference

| Register | Address | Description |
|----------|---------|-------------|
| BG0_SCROLLX | 0x8000-0x8001 | BG0 scroll X (16-bit) |
| BG0_SCROLLY | 0x8002-0x8003 | BG0 scroll Y (16-bit) |
| BG0_CONTROL | 0x8008 | BG0 enable/tile size |
| VRAM_ADDR | 0x800E-0x800F | VRAM address (16-bit) |
| VRAM_DATA | 0x8010 | VRAM data (8-bit, auto-increment) |
| CGRAM_ADDR | 0x8012 | Palette address (8-bit) |
| CGRAM_DATA | 0x8013 | Palette data (RGB555, two writes) |
| OAM_ADDR | 0x8014 | OAM address (8-bit) |
| OAM_DATA | 0x8015 | OAM data (8-bit, auto-increment) |
| VBLANK_FLAG | 0x803E | VBlank flag (one-shot) |
| FRAME_COUNTER | 0x803F-0x8040 | Frame counter (16-bit) |

### APU Register Quick Reference

| Register | Address | Description |
|----------|---------|-------------|
| CH0_FREQ | 0x9000-0x9001 | Channel 0 frequency (16-bit) |
| CH0_VOLUME | 0x9002 | Channel 0 volume (8-bit) |
| CH0_CONTROL | 0x9003 | Channel 0 control (8-bit) |
| CH0_DURATION | 0x9004-0x9005 | Channel 0 duration (16-bit) |
| CH1_FREQ | 0x9008-0x9009 | Channel 1 frequency |
| CH2_FREQ | 0x9010-0x9011 | Channel 2 frequency |
| CH3_FREQ | 0x9018-0x9019 | Channel 3 frequency |
| MASTER_VOLUME | 0x9020 | Master volume (8-bit) |
| CHANNEL_COMPLETION_STATUS | 0x9021 | Completion flags (one-shot) |

### Input Register Quick Reference

| Register | Address | Description |
|----------|---------|-------------|
| CONTROLLER1 | 0xA000 | Controller 1 buttons (low byte) |
| CONTROLLER1_LATCH | 0xA001 | Controller 1 latch / high byte |
| CONTROLLER2 | 0xA002 | Controller 2 buttons (low byte) |
| CONTROLLER2_LATCH | 0xA003 | Controller 2 latch / high byte |

---

## Language Constraints

### CoreLX Absolute Rules

- ❌ **No braces** `{ }`
- ❌ **No semicolons** `;`
- ❌ **No runtime interpretation**
- ❌ **No garbage collection**
- ❌ **No implicit heap allocation**
- ✅ **Indentation is authoritative**
- ✅ **One file = one ROM**
- ✅ **Compiled to machine code**

### Assembly Best Practices

- ✅ **Always wait for VBlank** before updating graphics
- ✅ **Preserve registers** when calling functions
- ✅ **Use I/O auto-detection** - don't manually use mode 6/7 for I/O
- ✅ **Check flags** after arithmetic operations
- ✅ **Use relative branches** for portability

---

## Troubleshooting

### CoreLX Issues

**"Missing required function: Start()"**
- Every CoreLX program must define `Start()`
- Check function name is exactly `Start` (case-sensitive)
- Function must have no parameters
- Function must be at top level

**"Indentation mismatch"**
- Check you're using tabs OR spaces (not both)
- All lines in a block must have same indentation
- Don't mix indentation styles

**"Undefined identifier"**
- Variable must be declared before use
- Function name must be spelled correctly
- Built-in functions must be called correctly

### Assembly Issues

**"Sprite flickers or disappears"**
- Always wait for VBlank before updating OAM
- Reset `OAM_ADDR` before each sprite update
- Write all 6 bytes of sprite entry
- Always write Control byte last

**"Audio doesn't play"**
- Set frequency, volume, and duration BEFORE enabling channel
- Check completion status once per frame
- Ensure duration > 0 for timed notes

**"Input doesn't work"**
- Latch controller before reading
- Read low byte first, then high byte
- Check button bits correctly

---

## Best Practices

### CoreLX Best Practices

1. **Always Wait for VBlank**
   ```corelx
   while true
       wait_vblank()  -- Always wait before updating graphics
       -- Update sprites, backgrounds, etc.
   ```

2. **Use Explicit Types for Hardware**
   ```corelx
   sprite: Sprite     -- Use struct types for hardware structures
   x: u8 = 0         -- Use explicit types for hardware registers
   ```

3. **Keep Functions Small**
   - CoreLX compiles to machine code
   - Large functions may produce large code
   - Keep functions focused and small

4. **Use Assets for Graphics**
   ```corelx
   asset MyTiles: tiles8
       hex
           -- Your tile data here
   ```
   Don't try to generate graphics at runtime. Use assets.

### Assembly Best Practices

1. **Always Wait for VBlank**
   ```assembly
   wait_vblank:
       MOV R1, #0x803E  ; VBLANK_FLAG
       MOV R2, [R1]
       CMP R2, #0
       BEQ wait_vblank
   ```

2. **Preserve Registers in Functions**
   - Push registers you'll modify
   - Pop them before returning
   - Document which registers are used

3. **Use I/O Auto-Detection**
   - Don't manually use mode 6/7 for I/O registers
   - Let the CPU handle 8-bit I/O automatically
   - Use mode 6/7 only for normal memory

4. **Complete Sprite Updates**
   - Always reset `OAM_ADDR` before updating
   - Write all 6 bytes of sprite entry
   - Write Control byte last

---

**End of Nitro Core DX Programming Manual**

---

*This manual demonstrates the philosophy of Nitro Core DX: Start with CoreLX for clarity and productivity, drop to assembly when you need precise control. Both languages compile to the same machine code, run on the same hardware, and can be mixed seamlessly. Choose the right tool for each task, and build amazing games.*
