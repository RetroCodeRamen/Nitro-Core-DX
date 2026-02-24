# Nitro Core DX Programming Manual

**Version 3.0 (Pre-Alpha Rewrite)**  
**Last Updated: February 24, 2026**

> **Pre-Alpha Note:** Nitro Core DX is moving fast. This manual is intentionally practical and current, but some language features and tools will still change before alpha.
>
> If something in the manual and the code disagree, trust the code and tests first.

---

## Table of Contents

1. [Welcome](#welcome)
2. [What You Can Build Today](#what-you-can-build-today)
3. [Two Ways to Program Nitro Core DX](#two-ways-to-program-nitro-core-dx)
4. [Nitro-Core-DX App (Recommended Starting Point)](#nitro-core-dx-app-recommended-starting-point)
5. [CoreLX Quick Start](#corelx-quick-start)
6. [CoreLX Basics (Beginner-Friendly)](#corelx-basics-beginner-friendly)
7. [Game Loop Basics (Frames, VBlank, Input)](#game-loop-basics-frames-vblank-input)
8. [Graphics: Tiles, Palettes, Sprites, OAM](#graphics-tiles-palettes-sprites-oam)
9. [Audio: Legacy APU First, FM Extension Next](#audio-legacy-apu-first-fm-extension-next)
10. [Assets and Data in CoreLX](#assets-and-data-in-corelx)
11. [Build and Run Workflows](#build-and-run-workflows)
12. [Assembly (Advanced Users)](#assembly-advanced-users)
13. [CoreLX vs Assembly: When to Use Which](#corelx-vs-assembly-when-to-use-which)
14. [Troubleshooting Guide](#troubleshooting-guide)
15. [What Is Planned (So You Can Plan Ahead)](#what-is-planned-so-you-can-plan-ahead)
16. [Reference Links](#reference-links)

---

## Welcome

Nitro Core DX is a hardware-first fantasy console platform with a software emulator today and an FPGA hardware target in the future.

This manual is written for two groups at the same time:

- **Brand new programmers** who want a clear path to making something visible and interactive.
- **Experienced programmers** who are new to game development, retro-style constraints, or direct hardware-style APIs.

### The Tone of This Manual

You asked for a friendly, practical manual in the spirit of an approachable "learn by doing" guide. That is the goal here.

You will see short side notes like these:

> **Quick Note:** A small detail that saves you time.

> **Watch Out:** A current pre-alpha behavior or gotcha.

> **Why This Matters:** The hardware/game-dev reason behind a design choice.

---

## What You Can Build Today

You can already build and run:

- CoreLX ROMs (`.corelx`) in the Nitro-Core-DX app and emulator
- low-level ROMs written directly in Go using ROM builders (machine-code emitters)
- text assembly ROMs (`.asm`) using the new assembler v1 (separate from CoreLX)

You can already test:

- sprites and movement
- palette changes
- input
- legacy APU audio
- FM extension behavior/audio experiments (currently via hardware/MMIO path and test ROMs)

---

## Two Ways to Program Nitro Core DX

Nitro Core DX now has **two practical programming paths**.

### 1. CoreLX (Recommended for Most Projects)

CoreLX is the main language for game and app development.

- indentation-based syntax
- hardware-oriented built-ins (`ppu.*`, `gfx.*`, `sprite.*`, `oam.*`, `input.*`, `apu.*`)
- compiles directly to machine code ROMs
- integrated into the Nitro-Core-DX app `Build` / `Build + Run` flow

### 2. Assembly (Advanced / Low-Level)

You now have a real v1 text assembler (`.asm` -> `.rom`) for advanced users.

Use assembly when you want:

- exact instruction-level behavior
- hardware bring-up tests
- low-level experiments
- to learn the CPU and machine model directly

> **Important:** CoreLX and Assembly are currently **separate build paths**. Inline mixed-mode `asm { ... }` inside CoreLX is **not implemented yet**.

---

## Nitro-Core-DX App (Recommended Starting Point)

Nitro-Core-DX (the integrated app / Dev Kit) is the current best way to work day-to-day.

### What Nitro-Core-DX Does Today

- CoreLX editor pane
- integrated emulator (embedded in the same app)
- diagnostics panel (compiler errors/warnings)
- build output panel
- manifest/memory summary panel
- `Build`
- `Build + Run`
- `Load ROM` (for prebuilt `.rom` files, including assembly ROMs)

### Run Nitro-Core-DX

```bash
go run ./cmd/corelx_devkit
```

### Layout (Current)

- **Top-left:** embedded emulator
- **Bottom-left:** diagnostics / output / manifest tabs
- **Right side:** editor/workbench tabs (Code + placeholders for future tools)

> **Quick Note:** If game input seems unresponsive, make sure **Capture Game Input** is enabled and click the emulator pane once.

---

## CoreLX Quick Start

### Your Entry Point: `Start()`

Every normal CoreLX program begins with:

```corelx
function Start()
    -- your game/app code here
```

CoreLX programs currently use a default startup path and then call `Start()`.

> **Quick Note:** A startup logo/boot presentation is planned, but not fully enabled yet.

### Smallest Useful Program

```corelx
function Start()
    ppu.enable_display()

    while true
        wait_vblank()
```

This turns on display output and waits forever.

### Compile from the CLI

```bash
go run ./cmd/corelx hello.corelx hello.rom
```

### Run in Emulator

```bash
go run -tags no_sdl_ttf ./cmd/emulator -rom hello.rom
```

(Use `-tags no_sdl_ttf` if your machine does not have SDL2_ttf dev libraries installed.)

---

## CoreLX Basics (Beginner-Friendly)

## Comments

Use `--` for comments:

```corelx
-- This is a comment
x := 10  -- Inline comment
```

## Indentation Defines Blocks

CoreLX uses indentation (like Python).

```corelx
if x > 10
    y = 1
else
    y = 0
```

No braces. No semicolons.

## Variables

### Declare with `:=`

```corelx
x := 100
y := 42
```

### Assign with `=`

```corelx
x = x + 1
y = y - 1
```

## Conditions and Loops

```corelx
while true
    if x < 10
        x = x + 1
```

> **Why This Matters:** Most game code is just “read input -> update variables -> draw state” repeated every frame.

## Structs (You Will Use `Sprite` a Lot)

CoreLX supports a `Sprite()` struct helper used for OAM sprite writes.

Typical pattern:

```corelx
hero := Sprite()
sprite.set_pos(&hero, 120, 80)
hero.tile = 0
hero.attr = SPR_PAL(1)
hero.ctrl = SPR_ENABLE() | SPR_SIZE_16()
```

---

## Game Loop Basics (Frames, VBlank, Input)

This section is the most important one for smooth movement.

## The Frame Concept (Beginner Version)

A game usually updates once per screen frame.

Each frame you typically:

1. wait for a safe update moment (VBlank / frame boundary)
2. read input
3. update positions/state
4. write sprite/background/audio changes

## `wait_vblank()` (Current Behavior)

`wait_vblank()` exists and is useful, but in the current CoreLX implementation it behaves as a **level wait**, not a strict edge wait.

That means a fast loop can sometimes run multiple logic updates during the same visible frame if you only use `wait_vblank()`.

> **Watch Out:** If movement looks too fast, this is often why.

## Recommended Frame-Step Pattern (Current Best Practice)

Use `frame_counter()` to wait for a new frame edge:

```corelx
last_frame := frame_counter()

while true
    while frame_counter() == last_frame
        wait_vblank()
    last_frame = frame_counter()

    -- exactly one logic update per frame
```

This is the pattern used in the current `devkit_moving_box_test.corelx` test.

## Reading Input

```corelx
buttons := input.read(0)
```

Controller `0` = player 1.

### Common Button Bits

These are the common button bits returned by `input.read(0)`:

- `0x01` = Up
- `0x02` = Down
- `0x04` = Left
- `0x08` = Right
- `0x10` = A
- `0x20` = B
- `0x40` = X
- `0x80` = Y
- `0x100` = L
- `0x200` = R
- `0x400` = Start
- `0x800` = Aux/extended button (used by some test mappings)

### Example Input Check

```corelx
if (buttons & 0x08) != 0
    x = x + 1
```

### Edge Trigger (Press Once, Not Hold)

Use previous state tracking:

```corelx
if (buttons & 0x10) != 0 and (prev_buttons & 0x10) == 0
    -- A button just pressed this frame
```

---

## Graphics: Tiles, Palettes, Sprites, OAM

This is the practical graphics workflow you will use most often in CoreLX.

## 1. Enable Display

```corelx
ppu.enable_display()
```

## 2. Set Palette Colors

```corelx
gfx.set_palette(1, 1, 0x7FFF)  -- palette 1, color 1 = white (RGB555)
```

### RGB555 Reminder

Colors are 15-bit values:

- `0x7FFF` = white
- `0x7C00` = red
- `0x03E0` = green
- `0x001F` = blue

> **Quick Note:** Palette edits are a great way to test rendering without changing tile data.

## 3. Load Tiles from an Asset

```corelx
tile_base := gfx.load_tiles(ASSET_BoxTile, 0)
```

This loads a tile asset into VRAM and returns the base tile index.

### Current Reliable Asset Types for `gfx.load_tiles`

- `tiles8`
- `tiles16`

## 4. Build a Sprite and Write It to OAM

```corelx
box := Sprite()
sprite.set_pos(&box, 152, 92)
box.tile = tile_base
box.attr = SPR_PAL(1) | SPR_PRI(0)
box.ctrl = SPR_ENABLE() | SPR_SIZE_16()

oam.write(0, &box)
oam.flush()
```

## Sprite Helper Functions (Common)

- `SPR_PAL(n)` - sprite palette select bits
- `SPR_PRI(n)` - sprite priority bits
- `SPR_ENABLE()` - enable sprite
- `SPR_SIZE_8()` - 8x8 sprite
- `SPR_SIZE_16()` - 16x16 sprite
- `SPR_HFLIP()` - horizontal flip
- `SPR_VFLIP()` - vertical flip
- `SPR_BLEND(mode)` - blend mode bits
- `SPR_ALPHA(a)` - alpha bits

## OAM Notes

- `oam.write(index, &sprite)` writes one sprite record
- `oam.flush()` currently exists as the write-finalization call in CoreLX code patterns

---

## Audio: Legacy APU First, FM Extension Next

Audio is in transition (in a good way).

## The Current Audio Architecture

### Legacy APU (Stable Path for CoreLX)

CoreLX currently exposes the original APU built-ins (the easy path).

This is the recommended path for now when writing CoreLX gameplay code.

Typical examples include:

- `apu.enable()`
- `apu.set_channel_wave(...)`
- `apu.set_channel_freq(...)`
- `apu.set_channel_volume(...)`
- `apu.note_on(...)`
- `apu.note_off(...)`

### FM Extension (OPM/YM2151-Style Direction)

The emulator now includes an **FM extension block** at the hardware/MMIO level (`0x9100-0x91FF`) with:

- host MMIO interface
- timer/status/IRQ behavior (deterministic placeholder timing model)
- audible OPM-lite FM synthesis path (software emulation)

But:

- `fm.*` CoreLX APIs are **not finalized yet**
- advanced FM programming is currently best done through assembly or low-level ROM code

> **Why This Matters:** This lets us keep CoreLX simple for beginners while still building toward richer, FPGA-friendly FM audio.

## Recommended Audio Channel Convention (Current Default Guidance)

For the legacy APU path, a practical default is:

- `CH0-CH1` = music voices (lead/harmony)
- `CH2` = bass or ambience layer
- `CH3` = sound effects / noise / accents

Tradeoff:

- Reserving one channel for SFX makes gameplay audio more responsive
- Using all channels for music sounds fuller but can make SFX harder to mix in

> **Quick Note:** For beginner projects, reserve at least one channel for SFX. It makes your game feel more alive immediately.

---

## Assets and Data in CoreLX

CoreLX supports inline asset declarations. The compiler pipeline is being expanded, but some parts are already useful today.

## Current Practical Asset Workflow

### Tile Assets (Most Mature Path)

```corelx
asset BoxTile: tiles16
    hex
        11 11 11 11 11 11 11 11
        ...
```

### Supported Encodings (Current Compiler)

- `hex` (most common for examples/tests)
- `b64` (supported in compiler pipeline)
- `text` (for broader asset normalization/planning; runtime APIs vary)

## Asset Naming

Assets are referenced in code using generated constants:

```corelx
ASSET_BoxTile
```

## What Is Still Evolving

The compiler now has a normalized asset pipeline and manifest reporting, but the full unified asset authoring model (tilemaps/music/gamedata tooling flow) is still being expanded.

> **Watch Out:** The compiler understands more asset kinds than the runtime APIs currently expose in CoreLX.

---

## Build and Run Workflows

## Workflow A: Nitro-Core-DX App (Recommended)

### CoreLX Build + Run

1. Open Nitro-Core-DX
2. Open a `.corelx` file
3. Click `Build + Run`
4. The ROM is compiled and loaded into the embedded emulator

### Load a Prebuilt ROM

Use `Load ROM` to test:

- ROMs built by CoreLX CLI
- ROMs built by the assembler CLI
- ROMs built by Go test ROM generators

This is especially useful for validating emulator behavior separately from compiler behavior.

## Workflow B: CoreLX CLI

```bash
go run ./cmd/corelx mygame.corelx mygame.rom
```

Then run:

```bash
go run -tags no_sdl_ttf ./cmd/emulator -rom mygame.rom
```

## Workflow C: Assembly CLI (New v1)

```bash
go run ./cmd/asm mygame.asm mygame.rom
```

Then either:

- use Nitro-Core-DX `Load ROM` (recommended), or
- use the standalone emulator CLI (optional)

---

## Assembly (Advanced Users)

You asked for an explicit assembly path, and v1 is now available.

## Current Status (v1 Assembler)

- text `.asm` input -> `.rom` output ✅
- labels ✅
- branches/jumps/calls ✅
- full current CPU opcode coverage ✅
- simple directives (`.entry`, `.word`) ✅

## What Assembly Is Good For Right Now

- hardware tests
- timing experiments
- low-level demos
- FM MMIO experiments before CoreLX `fm.*` APIs are finalized
- debugging compiler output assumptions

## Assembly Syntax (v1)

### Registers

- `R0` to `R7`

### Immediate Values

Prefix immediates with `#`:

```asm
MOV R0, #1
MOV R1, #0x8008
MOV R2, #$FF
```

### Memory Access

Use `[Rn]` syntax:

```asm
MOV R0, [R1]      ; 16-bit load (MMIO-safe for IO because CPU handles IO reads as 8-bit zero-extended)
MOV [R1], R0      ; 16-bit store (IO writes become low-byte writes)
MOV.B R2, [R1]    ; explicit 8-bit load
MOV.B [R1], R2    ; explicit 8-bit store
```

### Comments

Both styles are accepted:

```asm
; comment
-- comment
```

### Labels

```asm
start:
    MOV R0, #0
loop:
    ADD R0, #1
    CMP R0, #10
    BLT loop
    RET
```

## Directives (v1)

### `.entry`

Set entry bank and offset (optional; defaults are bank `1`, offset `0x8000`):

```asm
.entry 1, 0x8000
```

### `.word`

Emit one raw 16-bit word:

```asm
.word 0x1234
```

## Supported Instructions (v1)

### Data / Memory

- `NOP`
- `MOV`
- `MOV.B`
- `PUSH`
- `POP`

### Arithmetic / Logic

- `ADD`
- `SUB`
- `MUL`
- `DIV`
- `AND`
- `OR`
- `XOR`
- `NOT`
- `SHL`
- `SHR`
- `CMP`

### Control Flow

- `BEQ`
- `BNE`
- `BGT`
- `BLT`
- `BGE`
- `BLE`
- `JMP`
- `CALL`
- `RET`

## Example: Tiny Assembly ROM

```asm
.entry 1, 0x8000

start:
    MOV R4, #0x8008      ; BG0_CONTROL
    MOV R5, #0x01        ; enable display
    MOV [R4], R5

main_loop:
    JMP main_loop
```

Build it:

```bash
go run ./cmd/asm tiny.asm tiny.rom
```

> **Watch Out:** v1 assembler is same-bank / relative-control-flow oriented. Far-call/banked-assembler workflows are future work.

---

## CoreLX vs Assembly: When to Use Which

## Use CoreLX When You Want...

- fast iteration
- readable gameplay logic
- easier onboarding
- Nitro-Core-DX `Build + Run`
- fewer hardware details in your face

## Use Assembly When You Want...

- exact instruction behavior
- hardware validation
- custom low-level routines
- MMIO experiments not wrapped by CoreLX yet

## Current Best Combined Workflow (Pre-Mixed-Mode)

Since mixed-mode inline assembly is not implemented yet, the practical workflow is:

- write gameplay in CoreLX (`.corelx`)
- write low-level experiments/tests in assembly (`.asm`)
- load both types of ROMs in Nitro-Core-DX (CoreLX via `Build + Run`, assembly via `Load ROM`)

---

## Troubleshooting Guide

## "My ROM compiles but the screen is black"

Check these first:

- Did you call `ppu.enable_display()`?
- Did you load tile data (`gfx.load_tiles`) before using the tile index?
- Did you set visible palette colors with `gfx.set_palette(...)`?
- Did you write your sprite to OAM and call `oam.flush()`?

## "Movement is way too fast"

This is a common pre-alpha CoreLX pattern issue.

Use the frame-edge loop pattern with `frame_counter()` (see [Game Loop Basics](#game-loop-basics-frames-vblank-input)).

## "Input does nothing in Nitro-Core-DX"

- Make sure `Capture Game Input` is enabled
- Click the emulator pane once
- Confirm input works with a known-good ROM via `Load ROM` (this separates emulator issues from compiler issues)

## "Audio is silent"

- Make sure your system audio output is active
- For standalone emulator runs, use the right build flags (`no_sdl_ttf` affects fonts, not audio)
- Test with a known-good audio ROM first (for example, the APU/FM showcase test ROMs)

## "Assembly ROM runs strangely"

- Check label placement and branch targets
- Prefer labels over manual branch offsets
- Remember branches/jumps are PC-relative
- Start with a tiny loop and add one feature at a time

---

## What Is Planned (So You Can Plan Ahead)

These are active directions, not promises of exact syntax.

### CoreLX / Compiler

- richer asset model (tilemaps, palettes, music, gamedata, packaging integration)
- better diagnostics and editor integration
- more complete runtime APIs
- eventual mixed CoreLX + assembly support

### Nitro-Core-DX App

- Sprite Lab
- Tilemap Editor
- Sound Studio
- Debug overlays / memory viewers
- stronger code editing experience (current editor is intentionally simple)

### Audio

- better CoreLX audio ergonomics
- stronger FM tooling support
- continued FM accuracy and performance improvements

### Startup Experience

- startup/logo behavior is planned to evolve before alpha

---

## Reference Links

Use these for deeper details after you finish this manual.

- `docs/README.md` - docs map / source-of-truth guide
- `docs/CORELX.md` - language reference and examples (still being updated alongside compiler changes)
- `docs/CORELX_DATA_MODEL_PLAN.md` - compiler/data model plan for the SDK/dev kit
- `docs/DEVKIT_ARCHITECTURE.md` - backend/frontend split for Nitro-Core-DX (Dev Kit architecture)
- `docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md` - current hardware spec reference
- `docs/specifications/APU_FM_OPM_EXTENSION_SPEC.md` - FM extension architecture/status
- `test/roms/devkit_moving_box_test.corelx` - current CoreLX Nitro-Core-DX input/sprite validation example

---

## Final Advice (Especially If You Are New)

Start small.

A great first Nitro Core DX project is:

1. draw one sprite
2. move it with input
3. change a color on button press
4. add one sound effect
5. then add a second object

That path teaches the whole system without overwhelming you.

And if you are an experienced programmer: treat Nitro Core DX like a hardware platform, not a desktop app. Frame timing, memory layout, and simple pipelines matter here in a good way.
