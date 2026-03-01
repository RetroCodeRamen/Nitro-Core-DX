# CoreLX Language Documentation

**Version 1.0**  
**For Nitro Core DX**

> **CoreLX** (pronounced *core elix*) is the native compiled programming language for the **Nitro Core DX** console.  
> CoreLX is a **compiled-only**, **hardware-first** language with **no interpreter**, **no virtual machine**, and **no runtime scripting layer**.  
> Each CoreLX source file produces **one ROM image** that runs directly on the Nitro Core DX emulator or future hardware.

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Language Overview](#language-overview)
3. [Syntax Basics](#syntax-basics)
4. [Types](#types)
5. [Variables and Assignment](#variables-and-assignment)
6. [Control Flow](#control-flow)
7. [Functions](#functions)
8. [Structs](#structs)
9. [Assets](#assets)
10. [Sprites and OAM](#sprites-and-oam)
11. [Audio (APU)](#audio-apu)
12. [Built-in Functions Reference](#built-in-functions-reference)
13. [Examples](#examples)
14. [Test ROMs](#test-roms)
15. [Compiler Status](#compiler-status)
16. [Testing](#testing)

---

## Quick Start

### Installation

The CoreLX compiler is built with the project:

```bash
go build ./cmd/corelx
```

### Compile Your First Program

Create `hello.corelx`:

```corelx
function Start()
    ppu.enable_display()
    
    while true
        wait_vblank()
```

Compile it:

```bash
./corelx hello.corelx hello.rom
```

Run it:

```bash
./nitro-core-dx hello.rom
```

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

## Syntax Basics

### Indentation

CoreLX uses indentation to define blocks (like Python):

```corelx
if x > 5
    y = 10
    z = 20
else
    y = 0
```

### Comments

Single-line comments start with `--`:

```corelx
-- This is a comment
x := 10  -- Inline comment
```

### Identifiers

- Start with letter or underscore
- Can contain letters, numbers, underscores
- Case-sensitive

---

## Types

### Built-in Types

- **Integers**: `i8`, `u8`, `i16`, `u16`, `i32`, `u32`
- **Boolean**: `bool`
- **Fixed Point**: `fx8_8`, `fx16_16`
- **Pointers**: `*T` (e.g., `*Sprite`, `*u8`)

### Type Inference

Use `:=` for type inference:

```corelx
x := 10        -- Inferred as i32
y := 0x1234    -- Inferred as i32
flag := true   -- Inferred as bool
```

### Explicit Types

Use `:` for explicit typing:

```corelx
x: u8 = 10
y: i16 = -100
ptr: *Sprite = &sprite
```

---

## Variables and Assignment

### Variable Declaration

```corelx
-- Inferred type
x := 10
name := "Hello"

-- Explicit type
count: u8 = 0
position: i16 = 100
```

### Assignment

```corelx
x = 20
sprite.tile = 5
position = position + 1
```

---

## Control Flow

### If Statements

```corelx
if x > 5
    y = 10
else
    y = 0
```

### Else If

```corelx
if x < 5
    y = 1
else
    if x < 10
        y = 2
    else
        y = 3
```

### While Loops

```corelx
i := 0
while i < 10
    -- Do something
    i = i + 1
```

### For Loops

```corelx
-- For loops are syntactic sugar for while loops
for i := 0; i < 10; i = i + 1
    -- Do something
```

---

## Functions

### Function Declaration

Currently, only the `Start()` function is required:

```corelx
function Start()
    -- Your game code here
```

User-defined functions are planned for a future release.

---

## Structs

### Struct Definition

```corelx
type Vec2 = struct
    x: i16
    y: i16

type Sprite = struct
    x_lo: u8
    x_hi: u8
    y: u8
    tile: u8
    attr: u8
    ctrl: u8
```

### Struct Initialization

```corelx
pos := Vec2()
pos.x = 100
pos.y = 200

hero := Sprite()
hero.tile = 0
hero.attr = SPR_PAL(1)
hero.ctrl = SPR_ENABLE()
```

---

## Assets

### Asset Declaration

```corelx
asset MyTiles: tiles8
    hex
        FF FF FF FF FF FF FF FF
        00 00 00 00 00 00 00 00
```

### Using Assets

```corelx
base := gfx.load_tiles(ASSET_MyTiles, 0)
```

**Note**: `gfx.load_tiles` accepts either an `ASSET_*` constant or a runtime `u16` variable that contains one of the compiler-assigned asset IDs for assets declared in the same source file.
When the first argument is an `ASSET_*` literal, tile writes are inlined at compile time. Runtime IDs are dispatched across declared assets at runtime.

---

## Sprites and OAM

### Creating Sprites

```corelx
hero := Sprite()
sprite.set_pos(&hero, 160, 100)
hero.tile = base
hero.attr = SPR_PAL(0)
hero.ctrl = SPR_ENABLE()
```

### Writing to OAM

```corelx
oam.write(0, &hero)
oam.flush()
```

### Sprite Helpers

```corelx
palette := SPR_PAL(1)      -- Palette 0-3
priority := SPR_PRI(2)      -- Priority 0-3
hflip := SPR_HFLIP()        -- Horizontal flip
vflip := SPR_VFLIP()        -- Vertical flip
enabled := SPR_ENABLE()     -- Enable sprite
size8 := SPR_SIZE_8()       -- 8×8 size
size16 := SPR_SIZE_16()     -- 16×16 size
blend := SPR_BLEND(1)       -- Blend mode
alpha := SPR_ALPHA(15)      -- Alpha value
```

---

## Audio (APU)

Note: CoreLX currently exposes the legacy 4-channel APU built-ins documented below. The newer FM/OPM extension exists in the emulator/APU (`0x9100-0x91FF`) but does not yet have stable CoreLX language-level APIs.

### Enabling APU

```corelx
apu.enable()
```

### Configuring Channels

```corelx
-- Set waveform (0=Sine, 1=Square, 2=Saw, 3=Noise)
apu.set_channel_wave(0, 1)

-- Set frequency (Hz)
apu.set_channel_freq(0, 440)

-- Set volume (0-255)
apu.set_channel_volume(0, 128)

-- Start playback
apu.note_on(0)

-- Stop playback
apu.note_off(0)
```

---

## Built-in Functions Reference

### Frame Synchronization

- `wait_vblank()` - Wait for VBlank period
- `frame_counter() -> u32` - Get current frame number

### Graphics

- `ppu.enable_display()` - Enable PPU display
- `gfx.load_tiles(asset, base) -> u16` - Load declared tile asset into VRAM (`asset` may be an `ASSET_*` literal or a runtime asset-ID value for a declared asset)

### Sprites

- `sprite.set_pos(sprite, x, y)` - Set sprite position
- `oam.write(index, sprite)` - Write sprite to OAM
- `oam.flush()` - Flush OAM writes

### Audio

- `apu.enable()` - Enable APU
- `apu.set_channel_wave(ch, wave)` - Set waveform
- `apu.set_channel_freq(ch, freq)` - Set frequency
- `apu.set_channel_volume(ch, vol)` - Set volume
- `apu.note_on(ch)` - Start note
- `apu.note_off(ch)` - Stop note
- `fm.*` APIs - Planned (FM extension exists at hardware/MMIO level; CoreLX API integration is not finalized yet)

### Input

- `input.read(controller) -> u16` - Read controller state

### Sprite Helpers

- `SPR_PAL(p) -> u8` - Palette bits
- `SPR_PRI(p) -> u8` - Priority bits
- `SPR_HFLIP() -> u8` - Horizontal flip
- `SPR_VFLIP() -> u8` - Vertical flip
- `SPR_ENABLE() -> u8` - Enable bit
- `SPR_SIZE_8() -> u8` - 8×8 size
- `SPR_SIZE_16() -> u8` - 16×16 size
- `SPR_BLEND(mode) -> u8` - Blend mode
- `SPR_ALPHA(a) -> u8` - Alpha value

---

## Compiler Status

### ✅ Fully Implemented

- Lexer (tokenization)
- Parser (AST construction)
- Semantic analyzer (type checking)
- Code generator (machine code)
- ROM builder
- All built-in functions
- Control flow (if, while, for)
- Struct initialization
- Variable declarations
- Expression evaluation

### 🚧 In Progress

- Asset embedding (assets can be declared but not yet embedded into ROM)
- Variable storage optimization (basic implementation works, but can be improved)

### 📋 Planned

- User-defined functions
- Array support
- Enhanced expression optimization

---

## Testing

### Running Tests

```bash
go test ./internal/corelx/...
```

### Test ROMs

Test ROMs are in `test/roms/`:
- `simple_test.corelx` - Basic features
- `example.corelx` - Simple game loop
- `full_example.corelx` - Complete sprite example
- `corelx_comprehensive_test.corelx` - All language features

---

## Examples

### Example 1: Simple Game Loop

Basic frame synchronization:

```corelx
function Start()
    ppu.enable_display()
    
    frame := 0
    while true
        wait_vblank()
        frame = frame + 1
```

**Source**: `test/roms/example.corelx`

---

### Example 2: Sprite with Asset

Complete sprite example with asset loading:

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

**Source**: `test/roms/full_example.corelx`

**Features demonstrated**:
- Asset declaration
- Sprite initialization
- Struct member access
- Sprite helpers (`SPR_PAL`, `SPR_PRI`, `SPR_ENABLE`, `SPR_SIZE_16`)
- OAM operations

---

### Example 3: Audio Playback

Simple audio example:

```corelx
function Start()
    apu.enable()
    apu.set_channel_wave(0, 1)  -- Square wave
    apu.set_channel_freq(0, 440)  -- A4 note
    apu.set_channel_volume(0, 128)
    apu.note_on(0)
    
    while true
        wait_vblank()
```

**Features demonstrated**:
- APU initialization
- Channel configuration
- Waveform selection
- Frequency and volume control
- Note playback

---

### Example 4: Input and Movement

Reading input and moving a sprite:

```corelx
function Start()
    ppu.enable_display()
    
    player := Sprite()
    player_x := 160
    player_y := 100
    sprite.set_pos(&player, player_x, player_y)
    player.tile = 0
    player.attr = SPR_PAL(0)
    player.ctrl = SPR_ENABLE()
    
    while true
        wait_vblank()
        
        -- Read input
        buttons := input.read(0)
        
        -- Move player
        if (buttons & 0x01) != 0  -- UP
            if player_y > 8
                player_y = player_y - 2
        if (buttons & 0x02) != 0  -- DOWN
            if player_y < 192
                player_y = player_y + 2
        if (buttons & 0x04) != 0  -- LEFT
            if player_x > 8
                player_x = player_x - 2
        if (buttons & 0x08) != 0  -- RIGHT
            if player_x < 312
                player_x = player_x + 2
        
        sprite.set_pos(&player, player_x, player_y)
        oam.write(0, &player)
        oam.flush()
```

**Features demonstrated**:
- Input reading
- Bitwise operations
- Conditional movement
- Boundary checking
- Sprite position updates

**Source**: Based on `test/roms/sprite_eater_game.corelx`

---

### Example 5: Comprehensive Feature Test

This example demonstrates all CoreLX language features:

```corelx
-- See test/roms/corelx_comprehensive_test.corelx for complete example

function Start()
    -- Variable declarations
    x := 10
    y: u8 = 20
    
    -- Control flow
    if x > 5
        y = y + 1
    else
        y = y - 1
    
    -- Loops
    i := 0
    while i < 10
        i = i + 1
    
    -- Structs
    hero := Sprite()
    hero.tile = 0
    hero.attr = SPR_PAL(1)
    
    -- Expressions
    sum := x + y
    flag := x == 10
    bitwise := 0x0F & 0xF0
```

**Source**: `test/roms/corelx_comprehensive_test.corelx`

---

## Test ROMs

The `test/roms/` directory contains several example ROMs:

- **`example.corelx`** - Simple game loop
- **`full_example.corelx`** - Complete sprite example
- **`sprite_eater_game.corelx`** - Full game with input, collision, and multiple sprites
- **`corelx_comprehensive_test.corelx`** - Tests all language features
- **`apu_test.corelx`** - Audio function tests

To compile and run:

```bash
# Compile
./corelx test/roms/example.corelx test/roms/example.rom

# Run
./nitro-core-dx test/roms/example.rom
```

See [test/roms/README_TEST_ROMS.md](../test/roms/README_TEST_ROMS.md) for more details.

---

## Additional Resources

- [Debugging Guide](DEBUGGING_GUIDE.md) - How to debug CoreLX programs
- [Programming Manual](../PROGRAMMING_MANUAL.md) - Complete guide covering both CoreLX and Assembly
- [Language Design](LANGUAGE_DESIGN.md) - Design decisions and rationale
- [Compiler Implementation](archive/corelx/) - Historical implementation notes

---

## See Also

- **New to Nitro Core DX?** Start with the [Programming Manual](../PROGRAMMING_MANUAL.md) for a comprehensive introduction
- **Need Assembly?** See the [Programming Manual](../PROGRAMMING_MANUAL.md) for assembly language details
- **Debugging Issues?** Check the [Debugging Guide](DEBUGGING_GUIDE.md)
- **Want to understand design decisions?** Read [Language Design](LANGUAGE_DESIGN.md)

---

**Last Updated**: January 29, 2026
