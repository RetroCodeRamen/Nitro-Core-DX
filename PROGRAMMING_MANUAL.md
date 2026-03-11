# Nitro Core DX Programming Manual

**Version 3.1 (Pre-Alpha)**  
**Last Updated: March 6, 2026**

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

Nitro-Core-DX (the integrated app / Dev Kit) is a professional IDE for day-to-day development.

### Run Nitro-Core-DX

```bash
go run ./cmd/corelx_devkit
```

### IDE Structure

The app uses a traditional IDE layout with a menu bar and domain-grouped toolbar:

**Menu Bar:** File, Edit, View, Build, Debug, Tools, Help

**Toolbar Groups (left to right):**
- **Project:** New, Open, Save, Load ROM
- **Build:** Build, Build + Run (primary action)
- **Run/Debug:** Run, Pause, Stop, Step Frame, Step CPU
- **View:** Split View, Emulator Focus, Code Only

### View Modes

- **Split View** — Editor + emulator side by side (default)
- **Emulator Focus** — Emulator fills the workspace for play/test workflows
- **Code Only** — Editor fills the workspace, emulator hidden for focused coding

Use your OS window controls (title-bar maximize/restore) for window state.

### Dev Kit Features

- **Project Templates:** Create new projects from templates (Blank Game, Minimal Loop, Sprite Demo, Tilemap Demo, Shmup Starter, Matrix Mode Demo)
- **CoreLX Editor:** Inline syntax highlighting (in the active editor), line numbers, active-line emphasis, and diagnostics jump
- **Sprite Lab:** Pixel-art sprite editor integrated as a workbench tab (see below)
- **Tilemap Lab:** Tilemap paint/edit tool integrated as a workbench tab (see below)
- **Diagnostics Panel:** Compiler errors/warnings with severity filtering
- **Build State:** `Draft`, `Validating...`, `Validated`, and `Error` state indicator in the top bar
- **Build Output / Manifest / Debug Panels:** Build logs, memory summary, debugger output
- **Autosave:** Automatic crash recovery for unsaved work
- **Settings Persistence:** View mode, split positions, recent files, and UI density are saved between sessions
- **UI Density:** Switch between Compact and Standard spacing via **Tools > UI Density**
- **Load ROM:** Test prebuilt `.rom` files directly without recompilation

### Sprite Lab

The Sprite Lab is a built-in pixel-art editor accessible from the workbench tabs. It supports:

- Canvas sizes from 8x8 to 64x64 (step of 8)
- 16 palette banks with 16 colors each (RGB555 format)
- Pencil and Erase tools with optional Mirror X painting
- Wrapped sprite shifting controls (Shift Up/Down/Left/Right) for quick pixel block moves
- Grid overlay and hover highlighting
- Undo/Redo history (up to 128 states)
- Import/Export `.clxsprite` asset files
- **Apply To Manifest** to upsert asset data into `corelx.assets.json` (recommended compiler-ingested path)
- **Insert CoreLX Asset** to append generated asset + palette snippet
- **Apply To Project** to upsert/update matching asset blocks without duplicating them
- Preview pane with packed 4bpp hex output
- Transparent index-0 checkerboard preview mode for no-color pixels
- Palette editor with full-width RGB555 hex entry and slider-based value control

### Tilemap Lab

The Tilemap Lab is a built-in map editor in the same workbench area. It supports:

- Map sizes from 8x8 to 64x64 (step of 8)
- Tile-entry editing as packed `(tile, attr)` values
- Brush/fill/erase editing, undo/redo history
- Palette/flip attribute editing (`pal`, `flipX`, `flipY`)
- Parsing tile assets from current source (`tiles8`, `tileset`, `sprite`) into a selectable tile atlas
- Import/Export `.clxtilemap` files
- **Apply To Manifest** to upsert tilemap data into `corelx.assets.json`
- **Insert CoreLX Asset** and **Apply To Project** flows (same model as Sprite Lab)

### Workflow (Typical)

1. Click **New** and choose a project template, or **Open** an existing `.corelx` file
2. Edit code in the CoreLX editor
3. Click **Build + Run** to compile and run in the embedded emulator
4. Use **Sprite Lab** and **Tilemap Lab** to create assets, then apply them to source
5. Use **Load ROM** to test prebuilt ROMs without recompilation
6. Confirm the build status indicator returns to **Validated** after edits/builds

### Project Asset Manifest (`corelx.assets.json`)

The current compiler service path (used by Dev Kit Build/Build+Run) automatically checks for `corelx.assets.json` next to your `.corelx` file.

- If present, manifest assets are loaded and merged with in-source `asset` declarations.
- This keeps final build manifest/binary mapping compiler-owned.
- Editor tools should be treated as proposal/edit helpers; compiler outputs remain the source of truth for emitted artifacts.

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

> Current compiler behavior: the first argument to `gfx.load_tiles` can be either an `ASSET_*` literal (for example `ASSET_BoxTile`) or a runtime `u16` variable that holds one of the IDs of assets declared in the same source file.

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

## 5. Matrix Planes (Dedicated Matrix Sources)

Nitro-Core-DX now has a dedicated matrix-plane model in the emulator/runtime.

This is separate from ordinary BG tilemap usage.

Each matrix-capable layer can still bind to a transform channel, but that
transform channel can now source from its own:

- tilemap memory
- pattern/tile graphics memory
- size mode (`32x32`, `64x64`, `128x128`)

This is the current path for building large affine planes that do not just
borrow the ordinary BG tilemap.

### Why This Exists

The older approach was enough for small matrix demos, but not enough for the
larger per-plane target we want from Nitro-Core-DX.

The dedicated matrix-plane path is the first architecture that can scale toward:

- SNES-class Mode 7 sized planes
- larger pseudo-3D floors
- dedicated transformed backgrounds

### Current Matrix Plane Capabilities

Today, each dedicated matrix plane supports:

- independent tilemap backing
- independent pattern memory backing
- size mode:
  - `32x32`
  - `64x64`
  - `128x128`
- affine transform through the bound transform channel
- outside behavior:
  - wrap
  - backdrop
  - tile0
  - clamp

### Current Limits

- Matrix planes are available at the PPU/MMIO level, through emulator/Dev Kit helper APIs, and through a baseline CoreLX surface.
- Current CoreLX matrix-plane built-ins:
  - `matrix_plane.enable(channel, size)`
  - `matrix_plane.disable(channel)`
  - `matrix_plane.load_tiles(asset, channel, base)`
  - `matrix_plane.load_tilemap(asset, channel)`
  - `matrix_plane.set_tile(channel, x, y, tile, attr)`
  - `matrix_plane.fill_rect(channel, x, y, w, h, tile, attr)`
  - `matrix_plane.clear(channel, tile, attr)`
- This is now enough to author and upload useful dedicated planes directly from CoreLX without dropping to raw MMIO.
- Reference ROMs:
  - `test/roms/matrix_plane_showcase.corelx`
  - `test/roms/matrix_plane_pipeline_showcase.corelx`

### Recommended High-Level Programming Path (Current)

For ROM-side content, use the CoreLX matrix-plane built-ins first.

For tools, emulator-side tests, and Dev Kit experiments, use the matrix-plane builder when you need to generate or upload large planes efficiently.

Go-side example:

```go
builder, _ := emulator.NewMatrixPlaneBuilder(0, ppu.TilemapSize128x128)

builder.SetPatternTile8x8(0, redTileBytes)
builder.SetPatternTile8x8(1, greenTileBytes)
builder.FillRect(0, 0, 64, 128, 0, 0)
builder.FillRect(64, 0, 64, 128, 1, 0)

program := builder.Build()
svc.InstallMatrixPlaneProgram(program)
```

The emulator-side builder remains useful because it:

- validates the real matrix-plane model
- avoids raw byte packing by hand
- still compiles down to the real PPU programming surface

### Low-Level MMIO Programming Path

If you are writing low-level ROM code, the dedicated matrix-plane aperture is:

- `0x8080` `MATRIX_PLANE_SELECT`
- `0x8081` `MATRIX_PLANE_CONTROL`
- `0x8082` `MATRIX_PLANE_ADDR_L`
- `0x8083` `MATRIX_PLANE_ADDR_H`
- `0x8084` `MATRIX_PLANE_DATA`
- `0x8085` `MATRIX_PLANE_PATTERN_ADDR_L`
- `0x8086` `MATRIX_PLANE_PATTERN_ADDR_H`
- `0x8087` `MATRIX_PLANE_PATTERN_DATA`
- `0x8088` `MATRIX_PLANE_BITMAP_ADDR_L`
- `0x8089` `MATRIX_PLANE_BITMAP_ADDR_M`
- `0x808A` `MATRIX_PLANE_BITMAP_ADDR_H`
- `0x808B` `MATRIX_PLANE_BITMAP_DATA`

#### Register Meanings

- `MATRIX_PLANE_SELECT`
  - selects plane `0-3`

- `MATRIX_PLANE_CONTROL`
  - bit `0` = enable
  - bits `[2:1]` = size mode
    - `0 = 32x32`
    - `1 = 64x64`
    - `2 = 128x128`
  - bit `3` = source mode
    - `0 = tilemap/pattern-backed plane`
    - `1 = bitmap-backed plane`
  - bits `[7:4]` = bitmap palette bank

- `MATRIX_PLANE_ADDR_L/H`
  - tilemap upload address

- `MATRIX_PLANE_DATA`
  - writes one byte into matrix-plane tilemap memory
  - auto-increments address

- `MATRIX_PLANE_PATTERN_ADDR_L/H`
  - pattern upload address

- `MATRIX_PLANE_PATTERN_DATA`
  - writes one byte into matrix-plane pattern memory
  - auto-increments address

- `MATRIX_PLANE_BITMAP_ADDR_L/M/H`
  - bitmap upload address for bitmap-backed matrix planes

- `MATRIX_PLANE_BITMAP_DATA`
  - writes one byte into dedicated matrix-plane bitmap memory
  - auto-increments address

### Typical Upload Sequence

1. select the matrix plane
2. configure its size and enable bit
3. choose source type
4. if tile-backed:
   - upload tilemap bytes through `0x8082-0x8084`
   - upload pattern bytes through `0x8085-0x8087`
5. if bitmap-backed:
   - upload bitmap bytes through `0x8088-0x808B`
6. bind a visible layer to that transform channel
7. enable matrix mode on that channel

Pseudo-code:

```text
write8(0x8080, 0)      ; plane 0
write8(0x8081, 0x05)   ; enable + 128x128

write8(0x8082, 0x00)
write8(0x8083, 0x00)
for each tilemap byte
    write8(0x8084, byte)

write8(0x8085, 0x00)
write8(0x8086, 0x00)
for each pattern byte
    write8(0x8087, byte)
```

Bitmap-backed upload:

```text
write8(0x8080, 0)      ; plane 0
write8(0x8081, 0x1D)   ; enable + 128x128 + bitmap source + palette bank 1

write8(0x8088, 0x00)
write8(0x8089, 0x00)
write8(0x808A, 0x00)
for each packed bitmap byte
    write8(0x808B, byte)
```

### Pattern Memory Format

Pattern memory currently uses the same packed 4bpp tile format as normal tile data.

- `8x8` tile = `32 bytes`
- `16x16` tile = `128 bytes`

The layer's tile-size setting still controls how the matrix renderer interprets the pattern data.

So if BG0 is using `8x8`, your dedicated matrix plane patterns must be authored as `8x8` tiles.

### Bitmap Plane Memory Format

Bitmap-backed matrix planes use packed indexed 4bpp pixels:

- two pixels per byte
- high nibble = even pixel
- low nibble = odd pixel
- palette bank comes from `MATRIX_PLANE_CONTROL[7:4]`

Bitmap-backed planes are now the direct validation path for large imported images that do not fit cleanly into the tile-backed plane's 256-tile index ceiling.

### Tilemap Entry Format

Each tilemap entry is still:

- byte `0` = tile index
- byte `1` = attributes

Current attributes:

- palette low bits
- flip bits

### Outside Behavior

The transform channel's matrix-control register still defines outside behavior:

- `0` = wrap
- `1` = backdrop
- `2` = tile0
- `3` = clamp

Use `clamp` when you want the edge of the plane to hold instead of repeating.

Use `wrap` when you want classic repeating floor behavior.

### Practical Advice

- Use dedicated matrix planes for large rotated/scaled backgrounds.
- Keep ordinary BG tilemaps for conventional HUD/background work.
- Start with `128x128 @ 8x8` when you want a true `1024x1024` source plane.
- Do not hand-pack tilemap/pattern uploads unless you are explicitly doing low-level hardware work.

> **Watch Out:** CoreLX now exposes the practical dedicated matrix-plane path, including `fill_rect`, `clear`, and direct tilemap-asset upload. For very large or procedurally generated planes, the emulator/Dev Kit helper path is still the more efficient authoring route.

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

### FM Extension (Current Runtime) and YM2608 Plan

The emulator currently includes an in-progress **FM extension block** at the hardware/MMIO level (`0x9100-0x91FF`) with:

- host MMIO interface
- timer/status/IRQ behavior (deterministic placeholder timing model)
- audible OPM-lite FM synthesis path (software emulation, transitional)

Current constraints:

- `fm.*` CoreLX APIs are **not finalized yet**
- advanced FM programming is currently best done through assembly or low-level ROM code
- the V1 release audio target is now **YM2608**, so the current OPM-lite path should be treated as a bridge state while Sound Studio and audio migration gates are completed

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
- better diagnostics and editor integration (syntax highlighting, squiggles, find/replace)
- more complete runtime APIs
- eventual mixed CoreLX + assembly support

### Nitro-Core-DX App

- Sound Studio
- Debug overlays / memory viewers
- stronger code editing experience (native single-widget editor engine stabilization in progress)

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
- `docs/specifications/APU_FM_OPM_EXTENSION_SPEC.md` - current FM extension runtime architecture/status (transitional)
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
