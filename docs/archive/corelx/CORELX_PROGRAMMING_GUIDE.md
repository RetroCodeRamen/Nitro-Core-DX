# CoreLX Programming Guide

**Version 1.0**  
**For Nitro Core DX**

> **CoreLX** (pronounced *core elix*) is the native compiled programming language for the **Nitro Core DX** console.  
> CoreLX is a **compiled-only**, **hardware-first** language with **no interpreter**, **no virtual machine**, and **no runtime scripting layer**.  
> Each CoreLX source file produces **one ROM image** that runs directly on the Nitro Core DX emulator or future hardware.

---

## Table of Contents

1. [Language Overview](#language-overview)
2. [Getting Started](#getting-started)
3. [Syntax Basics](#syntax-basics)
4. [Types](#types)
5. [Variables and Assignment](#variables-and-assignment)
6. [Control Flow](#control-flow)
7. [Functions](#functions)
8. [Structs](#structs)
9. [Assets](#assets)
10. [Sprites and OAM](#sprites-and-oam)
11. [Audio (APU)](#audio-apu)
12. [Built-in Functions](#built-in-functions)
13. [Complete Example](#complete-example)

---

## Language Overview

### Core Principles

CoreLX is designed to feel:
- **Magical on the surface** - Simple, expressive syntax
- **Strict underneath** - Type-safe, compile-time checked
- **Powerful without being vague** - Direct hardware access
- **Learnable without dumbing anything down** - Clear, precise documentation

### Key Features

- **Indentation-based syntax** - No braces, no semicolons
- **Compiled to machine code** - Direct Nitro Core DX execution
- **Hardware-accurate** - Direct access to PPU, APU, OAM, VRAM
- **Single-file compilation** - One `.corelx` file = one ROM
- **Inline assets** - Embed graphics and data directly in source

---

## Getting Started

### Your First CoreLX Program

Every CoreLX program **must** define a `Start()` function:

```lua
function Start()
    ppu.enable_display()
    
    while true
        wait_vblank()
```

### Compiling

```bash
corelx game.corelx game.rom
```

This produces a `.rom` file that can be run in the Nitro Core DX emulator.

---

## Syntax Basics

### Indentation Rules

CoreLX uses **indentation** to define blocks. Indentation is **mandatory** and **authoritative**.

- **Increased indentation** → enter a block
- **Decreased indentation** → exit a block
- **Same indentation** → same block level
- **Tabs OR spaces** are allowed, but **must not be mixed** in the same file

```lua
function Start()
    -- This is indented (inside Start)
    if x == 5
        -- This is more indented (inside if)
        y = 10
    -- Back to Start() level
    z = 20
```

### Comments

```lua
-- This is a comment until end of line
```

### No Braces, No Semicolons

CoreLX has **no** `{ }` braces and **no** `;` semicolons.  
Newlines end statements. Indentation defines blocks.

---

## Types

### Built-in Types

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

### Pointer Types

```lua
x: *u8      -- Pointer to u8
sprite: *Sprite  -- Pointer to Sprite struct
```

### Type Inference

You can declare variables without explicit types:

```lua
x := 10        -- Inferred as i16 (smallest signed type)
y: u8 = 5      -- Explicitly u8
```

---

## Variables and Assignment

### Variable Declaration

```lua
x := 10              -- Declare with inferred type
y: u8 = 5            -- Declare with explicit type
z: i16               -- Declare without initializer (zero-initialized)
```

### Assignment

```lua
x = x + 1
y = 42
```

---

## Control Flow

### If Statements

```lua
if x == 5
    y = 10
elseif x == 6
    y = 20
else
    y = 30
```

### While Loops

```lua
while x < 100
    x = x + 1
    do_something()
```

### For Loops

```lua
for i := 0; i < 10; i = i + 1
    print(i)
```

---

## Functions

### Function Declaration

```lua
function add(a: i16, b: i16) -> i16
    return a + b
```

### Void Functions

```lua
function do_something()
    -- No return type = void
    x = 10
```

### Entry Point: Start()

Every CoreLX program **must** define:

```lua
function Start()
    -- Your game code here
```

`Start()` is a **reserved system entry point**:
- Exactly one `Start()` must exist
- `Start()` may not be overloaded, aliased, shadowed, or reassigned
- Any violation is a compile-time error

---

## Structs

### Struct Declaration

```lua
type Vec2 = struct
    x: i16
    y: i16
```

### Struct Usage

```lua
pos := Vec2()
pos.x = 10
pos.y = 20
```

### The Sprite Struct

CoreLX provides a canonical `Sprite` struct that maps 1:1 to OAM:

```lua
type Sprite = struct
    x_lo: u8     -- OAM byte 0
    x_hi: u8     -- OAM byte 1
    y: u8        -- OAM byte 2
    tile: u8     -- OAM byte 3
    attr: u8     -- OAM byte 4
    ctrl: u8     -- OAM byte 5
```

This struct **must match hardware layout exactly**.

---

## Assets

### Inline Asset Blocks

CoreLX supports **inline asset blocks** at the top level:

```lua
asset HeroTiles: tiles8
    hex
        00 00 11 11 22 22 33 33
        44 44 55 55 66 66 77 77
```

### Asset Types

- `tiles8` - 8×8 pixel tiles
- `tiles16` - 16×16 pixel tiles

### Asset Encodings

- `hex` - Hexadecimal data
- `b64` - Base64 encoded data

### Using Assets

The compiler emits constants for each asset:

```lua
base := gfx.load_tiles(ASSET_HeroTiles, 0)
```

`ASSET_<Name>` is a `u16` constant representing the asset.

---

## Sprites and OAM

### Sprite Sizes

Supported sizes: **8×8** and **16×16**  
Controlled via OAM control byte.

### Sprite Helper Functions

```lua
function sprite.set_pos(s: *Sprite, x: i16, y: u8)
```

### Sprite Attribute Helpers

```lua
SPR_PAL(p: u8) -> u8        -- Set palette (0-15)
SPR_HFLIP() -> u8           -- Horizontal flip
SPR_VFLIP() -> u8           -- Vertical flip
SPR_PRI(p: u8) -> u8        -- Set priority
SPR_ENABLE() -> u8          -- Enable sprite
SPR_SIZE_8() -> u8          -- 8×8 size
SPR_SIZE_16() -> u8         -- 16×16 size
SPR_BLEND(mode: u8) -> u8   -- Blending mode
SPR_ALPHA(a: u8) -> u8      -- Alpha value
```

### OAM API

```lua
function oam.write(id: u8, s: *Sprite)
function oam.flush()
```

**Important**: OAM writes must occur during VBlank. Always call `wait_vblank()` before updating sprites.

---

## Audio (APU)

Audio is **hardware-driven**, not software-mixed.

### APU API

```lua
function apu.enable()
function apu.set_channel_wave(ch: u8, wave: u8)
function apu.set_channel_freq(ch: u8, freq: u16)
function apu.set_channel_volume(ch: u8, vol: u8)
function apu.note_on(ch: u8)
function apu.note_off(ch: u8)
```

**No PCM streaming**. **No software synthesis**.  
Audio is generated by hardware channels.

---

## Built-in Functions

### Frame Synchronization

```lua
function wait_vblank()
function frame_counter() -> u32
```

### Graphics

```lua
function ppu.enable_display()
function gfx.load_tiles(asset: u16, base: u16) -> u16
```

---

## Complete Example

Here's a complete CoreLX program demonstrating all major features:

```lua
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

This example demonstrates:
- Inline asset declaration
- Sprite struct usage
- 16×16 sprite rendering
- Hardware-accurate OAM writes
- Indentation-based CoreLX syntax

---

## Language Constraints

### Absolute Rules

- ❌ **No braces** `{ }`
- ❌ **No semicolons** `;`
- ❌ **No runtime interpretation**
- ❌ **No garbage collection**
- ❌ **No implicit heap allocation**
- ✅ **Indentation is authoritative**

### Compilation Model

- **One file = one ROM**
- **Compiled to machine code** (not bytecode)
- **No runtime dependencies**
- **Direct hardware access**

---

## Best Practices

### 1. Always Wait for VBlank

```lua
while true
    wait_vblank()  -- Always wait before updating graphics
    -- Update sprites, backgrounds, etc.
```

### 2. Use Explicit Types for Hardware

```lua
sprite: Sprite     -- Use struct types for hardware structures
x: u8 = 0         -- Use explicit types for hardware registers
```

### 3. Keep Functions Small

CoreLX compiles to machine code. Large functions may produce large code.  
Keep functions focused and small.

### 4. Use Assets for Graphics

```lua
asset MyTiles: tiles8
    hex
        -- Your tile data here
```

Don't try to generate graphics at runtime. Use assets.

---

## Troubleshooting

### "Missing required function: Start()"

Every CoreLX program must define `Start()`. Check:
- Function name is exactly `Start` (case-sensitive)
- Function has no parameters
- Function is at top level (not nested)

### "Indentation mismatch"

Check:
- You're using tabs OR spaces (not both)
- All lines in a block have the same indentation
- You're not mixing indentation styles

### "Undefined identifier"

Check:
- Variable is declared before use
- Function name is spelled correctly
- Built-in functions are called correctly (e.g., `wait_vblank`, not `waitVBlank`)

---

## Appendix: Operator Precedence

From highest to lowest:

1. Member access: `.`
2. Function call: `()`
3. Unary: `-`, `!`, `~`, `&`
4. Multiplicative: `*`, `/`, `%`
5. Additive: `+`, `-`
6. Shift: `<<`, `>>`
7. Bitwise AND: `&`
8. Bitwise XOR: `^`
9. Bitwise OR: `|`
10. Comparison: `<`, `<=`, `>`, `>=`
11. Equality: `==`, `!=`
12. Logical AND: `and`
13. Logical OR: `or`
14. Assignment: `=`

---

## Appendix: Type Conversion

CoreLX performs **implicit conversions** in safe cases:

- Smaller to larger types: `u8` → `u16` → `u32`
- Signed to unsigned (when value is non-negative)
- Integer to fixed-point (when appropriate)

**Explicit conversions** are required for:
- Larger to smaller types
- Unsigned to signed (when value might be negative)
- Fixed-point to integer

---

**End of CoreLX Programming Guide**

---

*CoreLX is designed to feel like learning a spellbook that directly controls a machine.*
