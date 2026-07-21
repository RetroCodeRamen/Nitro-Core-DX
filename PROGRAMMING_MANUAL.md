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
9. [Audio: The YM2608 / OPNA Subsystem (with Legacy Scaffolding During Migration)](#audio-the-ym2608--opna-subsystem-with-legacy-scaffolding-during-migration)
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
- YM2608/OPNA audio (via the hardware/MMIO host interface and test ROMs)
- legacy `apu.*` built-ins (temporary migration scaffolding)

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
go run ./cmd/emulator -rom hello.rom
```

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

`break` exits the innermost `while` or `for` loop immediately; `continue`
skips the rest of the current iteration and moves on to the next one (in a
`for` loop, the loop variable still advances — `continue` never skips that):

```corelx
for i = 0 to 9
    if i == skip_value
        continue          -- skip just this iteration
    if i == stop_value
        break              -- stop the loop entirely
    total = total + i
```

> **Why This Matters:** Most game code is just “read input -> update variables -> draw state” repeated every frame.

## Structs (You Will Use `Sprite` a Lot)

CoreLX supports a `Sprite()` struct helper used for OAM sprite writes.

Typical pattern:

```corelx
hero := Sprite()
sprite.set_pos(hero, 120, 80)
hero.tile = 0
hero.attr = SPR_PAL(1)
hero.ctrl = SPR_ENABLE() | SPR_SIZE_16()
```

Structs are reference types — passing `hero` to a function shares it, so the
callee's edits are visible to the caller (no `&`, no pointers). You aren't
limited to `Sprite()` — declare your own with `struct Name:` followed by an
indented list of `field: type` lines:

```corelx
struct Player:
    x: fixed
    y: fixed
    lives: int

function Start()
    player := Player()
    player.lives = 3
```

Every field created this way works exactly like `Sprite`'s built-in fields —
read, write, and pass the whole struct by name to functions.

---

## Modules (`--!` Directives)

`--!` lines are directives, not comments — they're only legal at the very top
of the file, before any code, and declare things about the file itself:

```corelx
--! corelx 1.0
--! modules: walker, dialog
```

- `--! corelx <version>` records which CoreLX version the file targets.
- `--! modules: name, name, ...` pulls in one or more modules — plain
  `.corelx` files that live in a `modules/` folder next to your project.
  Functions inside a module are called the same way as builtins, namespaced
  by the module's name:

```corelx
--! modules: walker

function Start()
    walker.update(1)
    while true
        wait_vblank()
```

A module is just a normal CoreLX file — functions, and any `const`/`var`
declarations those functions need — nothing special about its own syntax. If
`walker.corelx` isn't found in the `modules/` folder, you get a clear error
(`module 'walker' not installed`) rather than a confusing "unknown function"
at the call site. An unrecognized directive (a typo, or a directive keyword
that doesn't exist yet) is also a compile error, not a silently-ignored line.

### The `anim` Module (Sprite Animation)

`anim` ships with the project (`modules/anim.corelx`) and handles the two
genuinely reusable parts of sprite animation: frame timing and mirroring.
Frame lists themselves stay as plain array constants in your own code — a
module can only index arrays declared in the same file, so this keeps things
working with today's language rather than waiting on new syntax:

```corelx
--! modules: anim

const WALK_FRAME_COUNT = 4
var walk_frames: int[4] = [1, 2, 3, 4]

function Start()
    hero := Sprite()
    while true
        wait_vblank()
        idx := anim.frame_index(WALK_FRAME_COUNT, 8)  -- new frame every 8 ticks
        hero.tile = walk_frames[idx]
        anim.set_mirror(hero, 0)                      -- 1 to flip horizontally
```

`anim.frame_index(frame_count, ticks_per_frame)` returns which frame (0 to
`frame_count - 1`) should be showing right now, looping back to 0 after the
last one. `anim.set_mirror(sprite, mirror)` sets or clears horizontal flip —
useful for getting a second direction (e.g. "walk right") out of frames you
only drew once (e.g. "walk left").

### The `sfx` Module (Sound Effect Triggers)

`sfx` ships with the project (`modules/sfx.corelx`) and handles the one part
every FM note needs and the one place a wrong register bit is easy to get
subtly wrong: the key-on/key-off sequence. It doesn't set pitch or
instrument sound — configure those yourself with `ym.write`/`ym.write_port1`
(see the register map in `docs/specifications/`), then trigger and release
the note:

```corelx
--! modules: sfx

function Start()
    -- (configure channel 0's pitch/instrument once via ym.write)
    sfx.play(0)
    -- ... later ...
    sfx.stop(0)
```

Channel numbers are 0, 1, 2 for FM channels 1-3, and 4, 5, 6 for FM channels
4-6 (channel 3 doesn't exist in this encoding).

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

`gfx.set_palette_color(cgram_index, color)` is the same write with a flat
0-255 CGRAM index instead of separate palette/color-index arguments — useful
when you already have a combined index (e.g. writing a color table in a loop):

```corelx
gfx.set_palette_color(17, 0x7FFF)  -- CGRAM index 17 (palette 1, color 1) = white
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
- independent bitmap backing
- independent per-scanline row parameters
- generic per-plane projection modes
  - perspective row projection
  - vertical projected quad
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
  - `test/roms/build_matrix_rowmode_showcase.go`

### Row Mode (Generic Per-Scanline Projection)

The PPU now exposes a generic row-mode path for dedicated matrix planes.

This is the reusable hardware primitive for:

- floor-style perspective views
- road-style scanline projection
- flat row-stepped views
- ROM-side fisheye or custom distortion

Each visible scanline stores four `16.16` fixed-point values:

- `StartX`
- `StartY`
- `StepX`
- `StepY`

For each screen pixel on that row, the plane samples:

- `worldX = StartX + StepX * screenX`
- `worldY = StartY + StepY * screenX`

This keeps the PPU generic. ROM/CoreLX/tooling owns the camera and projection
math; the PPU just executes the row parameters.

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
- `0x808C` `MATRIX_PLANE_FLAGS`
- `0x808D` `MATRIX_PLANE_ROW_CONTROL`
- `0x808E` `MATRIX_PLANE_ROW_ADDR_L`
- `0x808F` `MATRIX_PLANE_ROW_ADDR_H`
- `0x8090` `MATRIX_PLANE_ROW_DATA`
- `0x8091` `MATRIX_PLANE_PROJECTION_CONTROL`
- `0x8092` `MATRIX_PLANE_HORIZON`
- `0x8093` `MATRIX_PLANE_CAMERA_X_L`
- `0x8094` `MATRIX_PLANE_CAMERA_X_H`
- `0x8095` `MATRIX_PLANE_CAMERA_Y_L`
- `0x8096` `MATRIX_PLANE_CAMERA_Y_H`
- `0x8097` `MATRIX_PLANE_HEADING_X_L`
- `0x8098` `MATRIX_PLANE_HEADING_X_H`
- `0x8099` `MATRIX_PLANE_HEADING_Y_L`
- `0x809A` `MATRIX_PLANE_HEADING_Y_H`
- `0x809B` `MATRIX_PLANE_BASE_DISTANCE_L`
- `0x809C` `MATRIX_PLANE_BASE_DISTANCE_H`
- `0x809D` `MATRIX_PLANE_FOCAL_LENGTH_L`
- `0x809E` `MATRIX_PLANE_FOCAL_LENGTH_H`
- `0x809F` `MATRIX_PLANE_WIDTH_SCALE_L`
- `0x80A0` `MATRIX_PLANE_WIDTH_SCALE_H`
- `0x80A1` `MATRIX_PLANE_ORIGIN_X_L`
- `0x80A2` `MATRIX_PLANE_ORIGIN_X_H`
- `0x80A3` `MATRIX_PLANE_ORIGIN_Y_L`
- `0x80A4` `MATRIX_PLANE_ORIGIN_Y_H`
- `0x80A5` `MATRIX_PLANE_FACING_X_L`
- `0x80A6` `MATRIX_PLANE_FACING_X_H`
- `0x80A7` `MATRIX_PLANE_FACING_Y_L`
- `0x80A8` `MATRIX_PLANE_FACING_Y_H`
- `0x80A9` `MATRIX_PLANE_HEIGHT_SCALE_L`
- `0x80AA` `MATRIX_PLANE_HEIGHT_SCALE_H`

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

- `MATRIX_PLANE_FLAGS`
  - bit `0` = bitmap palette index `0` is transparent
  - bit `1` = projection stays visible from both sides

- `MATRIX_PLANE_ROW_CONTROL`
  - bit `0` = row mode enabled

- `MATRIX_PLANE_ROW_ADDR_L/H`
  - selects a byte address inside the per-plane row table

- `MATRIX_PLANE_ROW_DATA`
  - writes one byte into row-parameter memory
  - auto-increments address

- `MATRIX_PLANE_PROJECTION_CONTROL`
  - generic per-plane projection primitive selector
  - `0 = none/manual rows`
  - `1 = perspective row projection`
  - `2 = vertical projected quad`

- `MATRIX_PLANE_HORIZON`
  - screen-space horizon line for generic perspective projection

- `MATRIX_PLANE_CAMERA_*`
  - source-space camera position in pixels

- `MATRIX_PLANE_HEADING_*`
  - `8.8` fixed-point forward vector

- `MATRIX_PLANE_BASE_DISTANCE`
  - base depth term for the projection

- `MATRIX_PLANE_FOCAL_LENGTH`
  - projection focal-length term

- `MATRIX_PLANE_WIDTH_SCALE`
  - horizontal span scale

- `MATRIX_PLANE_ORIGIN_*`
  - world/source anchor used by vertical projected quads

- `MATRIX_PLANE_FACING_*`
  - `8.8` fixed-point facing/normal vector for vertical projected quads

- `MATRIX_PLANE_HEIGHT_SCALE`
  - vertical size term for projected quads

Vertical projected quads are now treated as real world-space planes instead of
screen-facing billboards.

That means:

- the renderer intersects the camera ray with the plane defined by
  `ORIGIN_*` and `FACING_*`
- the bottom of the quad is anchored to the ground/world position represented
  by `ORIGIN_*`
- off-angle views should narrow and foreshorten instead of behaving like a
  Doom-style sprite that keeps facing the camera
- if a facade should stay visually locked to the floor, it should normally
  share the same camera/horizon/focal model as the floor plane that scene uses

#### Pitfall: Every Plane In A Scene Must Share One Camera-Eye

If a scene combines a perspective floor with one or more vertical-billboard
planes (a building, an NPC, any object standing "on" that floor), every one of
those planes' `matrix_plane.set_camera(channel, x, y, heading_x, heading_y)`
calls must be fed the **exact same** `x, y` position for a given frame — not
just the same *player* position, but the same fully-computed camera-eye
position, including any camera trick layered on top of it.

This bit the NitroPackInDemo rebuild directly. A common technique for a
walking-around game is a "feet pivot": instead of rendering the floor from the
player's raw position, the floor's camera trails the player by a fixed
world-unit offset in the direction opposite of facing —

```corelx
eye_x = cam_x - pivot_x[heading_index]
eye_y = cam_y - pivot_y[heading_index]
matrix_plane.set_camera(0, eye_x, eye_y, heading_x[heading_index], heading_y[heading_index])
```

— so that turning visually pivots around the character's own feet/screen
position instead of around the floor's raw coordinate. The mistake is
assuming a billboard object (a building, say) should track the player's *raw*
position instead, on the reasoning that "the billboard should scale off the
real player position, not some camera trick." That reasoning sounds
principled, but it's wrong: it silently makes the billboard render from a
**different eye position than the floor**, and since the pivot offset itself
rotates with heading, the mismatch between the two eyes also rotates —
visually, the billboard appears to drift/slip across the floor as the camera
turns, even though the object's own world position never changed. It looks
exactly like a physics bug (the building "isn't anchored to the ground
right"), but the actual cause is a camera inconsistency between planes, not
anything wrong with the billboard's own placement math.

**The fix**: compute the camera-eye position once per frame, and pass that
same value to `matrix_plane.set_camera` for every plane in the scene that's
meant to share one coherent 3D space — the floor, every billboard standing on
it, and, if applicable, the audio/gameplay logic that also cares about "where
the camera is." Never let one plane use a raw position while a sibling plane
in the same scene uses an adjusted one, even if the adjustment seems purely
cosmetic (like a feet-pivot offset). If you have multiple independent scenes
(say, an outdoor overworld and an interior room), each scene needs its own
consistently-shared eye — don't mix eyes from different scenes either.

```corelx
-- WRONG: billboard uses a different eye than the floor it stands on.
matrix_plane.set_camera(0, cam_x - pivot_x[h], cam_y - pivot_y[h], heading_x[h], heading_y[h])
matrix_plane.set_camera(1, cam_x, cam_y, heading_x[h], heading_y[h])

-- RIGHT: compute the eye once, feed it to every plane in the scene.
eye_x = cam_x - pivot_x[h]
eye_y = cam_y - pivot_y[h]
matrix_plane.set_camera(0, eye_x, eye_y, heading_x[h], heading_y[h])
matrix_plane.set_camera(1, eye_x, eye_y, heading_x[h], heading_y[h])
```

If you actually *want* an object to visually behave differently from the
floor (e.g. a HUD-anchored object that should never move relative to the
screen), that's a real design choice — but it means the object isn't meant to
occupy the same 3D space as the floor at all, and shouldn't be using
`matrix_plane.set_camera`'s world-position semantics for it. That's different
from silently drifting an object that's supposed to be standing still on the
ground.

#### Row Table Layout

Each visible scanline has a 16-byte row record:

- bytes `0-3` = `StartX`
- bytes `4-7` = `StartY`
- bytes `8-11` = `StepX`
- bytes `12-15` = `StepY`

There are `200` visible scanlines, so one plane's row table is `3200` bytes.

### Typical Upload Sequence

1. select the matrix plane
2. configure its size and enable bit
3. choose source type
4. if tile-backed:
   - upload tilemap bytes through `0x8082-0x8084`
   - upload pattern bytes through `0x8085-0x8087`
5. if bitmap-backed:
   - upload bitmap bytes through `0x8088-0x808B`
6. if row-driven:
   - upload row parameters through `0x808E-0x8090`
7. bind a visible layer to that transform channel
8. enable matrix mode on that channel

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

Row-mode upload:

```text
write8(0x8080, 0)      ; plane 0
write8(0x808D, 0x01)   ; row mode enabled

write8(0x808E, 0x00)
write8(0x808F, 0x00)
for each row-table byte
    write8(0x8090, byte)
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

### ROM-Side Responsibility

The PPU now owns only the generic plane, source, transform, and row-parameter
features.

Generic per-plane projection is also part of the PPU contract when used through
the `0x8091-0x80AA` register block. This is a reusable hardware feature, not a
demo-specific mode.

ROM/CoreLX/tooling should own:

- camera movement
- track following
- floor-table generation
- billboard/object placement
- fisheye or other custom projection styles

The rule is:

- the PPU provides generic per-plane primitives
- ROM/CoreLX decides how to drive them for a particular game

That keeps the PPU reusable and hardware-shaped instead of baking one demo into
the chip.

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

## Audio: The YM2608 / OPNA Subsystem (with Legacy Scaffolding During Migration)

Nitro-Core-DX has **one final audio subsystem: YM2608 / OPNA** (FM, SSG,
rhythm, and ADPCM). The old 4-channel "fantasy APU" is **not final hardware** —
it remains in the tree only as temporary migration scaffolding while the YM2608
CoreLX surface is built, and it will be removed.

## The Audio Architecture

### YM2608 / OPNA (the final audio subsystem)

The emulator runs YM2608/OPNA audio through a hardware/MMIO host interface
(`0x9100-0x91FF`):

- host MMIO interface + register path
- timer/status/IRQ behavior
- YM2608/OPNA playback through the YMFM-backed runtime

Current status (honest):

- YM2608 conformance is **operational and under active refinement** — not yet
  fully verified against hardware.
- The CoreLX **music playback API (`music.*`) is built and emulator-tested**
  — see below. Advanced FM sound design (instrument definition, pitch,
  per-voice register tweaking) is still done through low-level ROM code / the
  host registers, or the `sfx` module's key-on/key-off convenience wrapper.

### The `music.*` Built-ins (Song Playback)

A music asset is a compiled `.ncdxmusic` stream (built with
`cmd/vgm_to_ncdxmusic` or written directly) declared like any other asset:

```corelx
asset Theme: music "theme.ncdxmusic"
asset Fanfare: music "fanfare.ncdxmusic"

function Start()
    music.play_loop(Theme)
    while true
        wait_vblank()
```

- `music.play(asset)` plays once and stops (silences the chip and clears
  playback state when the song ends).
- `music.play_loop(asset)` plays and wraps back to the start forever.
- `music.play_jingle(asset)` stashes whatever's currently playing (including
  "nothing"), plays the given song once, then restores exactly what was
  playing before — frame index and all, not restarted from the top. Use it
  for a one-off sting (level-clear fanfare, item pickup) over a looping BGM
  track.
- `music.stop()` silences the chip immediately and clears playback state.
- `music.set_volume(level)` sets output volume (0-255) immediately.
- `music.fade_to(level, frames)` ramps volume to `level` over `frames` real
  frames — call it once, the ramp runs on its own each `wait_vblank()`.

All of these drive off `wait_vblank()` — the per-frame advance (loading the
next frame's register writes, checking for loop/end) only happens when your
code calls `wait_vblank()`, the same place every other per-frame system
(input, animation) already expects to run.

### Legacy `apu.*` built-ins (temporary scaffolding)

Until the YM2608 music API lands, CoreLX still exposes the older 4-channel
built-ins. **Treat these as transitional** — they target the legacy synth, not
the final YM2608 subsystem, and will be replaced:

- `apu.enable()`
- `apu.set_channel_wave(...)`
- `apu.set_channel_freq(...)`
- `apu.set_channel_volume(...)`
- `apu.note_on(...)`
- `apu.note_off(...)`

> **Why This Matters:** keeping the old built-ins working during the migration
> means existing CoreLX code keeps compiling while the YM2608 audio surface is
> designed and built — then audio converges fully on YM2608.

## Recommended Audio Channel Convention (Legacy Scaffolding Only)

For the temporary legacy `apu.*` path, a practical default is:

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
go run ./cmd/emulator -rom mygame.rom
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
- `docs/specifications/CORELX_CARTRIDGE_FORMAT.md` - CoreLX cartridge/asset format (v1 draft)
- `docs/DEVKIT_ARCHITECTURE.md` - backend/frontend split for Nitro-Core-DX (Dev Kit architecture)
- `docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md` - current hardware spec reference
- `docs/specifications/APU_FM_OPM_EXTENSION_SPEC.md` - YM2608 audio subsystem runtime architecture/status (file to be renamed in a later step)
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
