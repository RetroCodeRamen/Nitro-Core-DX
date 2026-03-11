# Nitro-Core-DX Programming Guide

**Last Updated**: 2026-03-06  
**Purpose**: A practical CoreLX-first guide that starts with a working demo, explains program structure, walks through sprites/tilemaps with real-world rules, templates and Dev Kit labs, and builds a small Pong-style game.

> This guide is intentionally hands-on and aligned to the current Dev Kit + CoreLX workflow.
> For broader platform context and reference material, also see `PROGRAMMING_MANUAL.md` and `docs/CORELX.md`.

---

## Table of Contents

1. [Start Here: The Current Demo Program](#start-here-the-current-demo-program)
2. [How a CoreLX Program Is Structured](#how-a-corelx-program-is-structured)
3. [Run the Demo (Dev Kit and CLI)](#run-the-demo-dev-kit-and-cli)
4. [A Reusable CoreLX Game Template](#a-reusable-corelx-game-template)
5. [Working with Sprites (Real-World Guide)](#working-with-sprites-real-world-guide)
6. [Build a Small Pong-Style Game (Mini Pong)](#build-a-small-pong-style-game-mini-pong)
7. [Test the Pong Example Before Publishing](#test-the-pong-example-before-publishing)
8. [Using the Sprite Lab](#using-the-sprite-lab)
9. [Using the Tilemap Lab](#using-the-tilemap-lab)
10. [Project Asset Manifest (`corelx.assets.json`)](#project-asset-manifest-corelxassetsjson)
11. [Where to Go Next](#where-to-go-next)

---

## Start Here: The Current Demo Program

This is the current CoreLX demo used for Dev Kit validation: `test/roms/devkit_moving_box_test.corelx`.

It is a great starting point because it already demonstrates:
- asset declaration
- palette setup
- tile upload
- sprite creation
- input
- frame-timed game loop updates
- OAM writes / flush

```corelx
-- Dev Kit Moving Box Test (CoreLX)
-- Purpose:
--   - Validate CoreLX compile -> ROM package -> Dev Kit Build+Run flow
--   - Validate embedded emulator rendering/input path in the Dev Kit
-- Controls:
--   - Arrow Keys / WASD: move the box
--   - Z: cycle palette color

asset BoxTile: tiles16
    hex
        -- Top half: color index 1 (0x11)
        11 11 11 11 11 11 11 11
        11 11 11 11 11 11 11 11
        11 11 11 11 11 11 11 11
        11 11 11 11 11 11 11 11
        11 11 11 11 11 11 11 11
        11 11 11 11 11 11 11 11
        11 11 11 11 11 11 11 11
        11 11 11 11 11 11 11 11
        -- Bottom half: color index 2 (0x22)
        22 22 22 22 22 22 22 22
        22 22 22 22 22 22 22 22
        22 22 22 22 22 22 22 22
        22 22 22 22 22 22 22 22
        22 22 22 22 22 22 22 22
        22 22 22 22 22 22 22 22
        22 22 22 22 22 22 22 22
        22 22 22 22 22 22 22 22

function Start()
    ppu.enable_display()

    -- Palette 1 defaults for visible box colors (RGB555)
    gfx.set_palette(1, 1, 0x7FFF)  -- White
    gfx.set_palette(1, 2, 0x03FF)  -- Cyan
    gfx.set_palette(1, 3, 0x7C1F)  -- Magenta
    gfx.set_palette(1, 4, 0x7FE0)  -- Yellow

    tile_base := gfx.load_tiles(ASSET_BoxTile, 0)

    box_x := 152
    box_y := 92
    color_mode := 0
    prev_buttons := 0
    last_frame := frame_counter()

    box := Sprite()
    sprite.set_pos(&box, box_x, box_y)
    box.tile = tile_base
    box.attr = SPR_PAL(1) | SPR_PRI(0)
    box.ctrl = SPR_ENABLE() | SPR_SIZE_16()

    wait_vblank()
    oam.write(0, &box)
    oam.flush()

    while true
        -- wait for next frame edge (wait_vblank() is level-based today and can
        -- advance multiple logic ticks during one visible frame)
        while frame_counter() == last_frame
            wait_vblank()
        last_frame = frame_counter()

        buttons := input.read(0)

        if (buttons & 0x01) != 0
            if box_y > 4
                box_y = box_y - 1

        if (buttons & 0x02) != 0
            if box_y < 180
                box_y = box_y + 1

        if (buttons & 0x04) != 0
            if box_x > 4
                box_x = box_x - 1

        if (buttons & 0x08) != 0
            if box_x < 300
                box_x = box_x + 1

        -- A button edge-trigger (Z / bit 4) cycles palette colors for the tile
        if (buttons & 0x10) != 0 and (prev_buttons & 0x10) == 0
            color_mode = color_mode + 1
            if color_mode >= 4
                color_mode = 0

            if color_mode == 0
                gfx.set_palette(1, 1, 0x7FFF)
                gfx.set_palette(1, 2, 0x03FF)
            if color_mode == 1
                gfx.set_palette(1, 1, 0x7C00)
                gfx.set_palette(1, 2, 0x03E0)
            if color_mode == 2
                gfx.set_palette(1, 1, 0x001F)
                gfx.set_palette(1, 2, 0x7FE0)
            if color_mode == 3
                gfx.set_palette(1, 1, 0x7C1F)
                gfx.set_palette(1, 2, 0x4210)

        prev_buttons = buttons

        sprite.set_pos(&box, box_x, box_y)
        oam.write(0, &box)
        oam.flush()
```

---

## How a CoreLX Program Is Structured

Using the demo above, here is the practical structure you should follow for most programs today.

### 1. Asset declarations (optional, but common)

Assets are declared at the top level:

```corelx
asset BoxTile: tiles16
    hex
        ...
```

This keeps art/data close to the code for small projects and tutorials.

### 2. Entry point: `Start()`

Most CoreLX programs start at:

```corelx
function Start()
```

That is where you do setup, then enter your main loop.

### 3. Initialization phase (run once)

The demo does all one-time setup first:
- enable display (`ppu.enable_display()`)
- set palette colors (`gfx.set_palette(...)`)
- upload tile graphics (`gfx.load_tiles(...)`)
- initialize game state variables
- create/configure sprite structs

This makes the main loop smaller and easier to reason about.

### 4. Sprite setup and first OAM write

The demo creates a sprite struct and fills it out once:

```corelx
box := Sprite()
sprite.set_pos(&box, box_x, box_y)
box.tile = tile_base
box.attr = SPR_PAL(1) | SPR_PRI(0)
box.ctrl = SPR_ENABLE() | SPR_SIZE_16()
```

Then it writes the sprite to OAM and flushes:

```corelx
wait_vblank()
oam.write(0, &box)
oam.flush()
```

`oam.flush()` is what commits the queued OAM writes. For multiple sprites and real-world rules (tile indices, palettes, 16×16 assets), see [Working with Sprites (Real-World Guide)](#working-with-sprites-real-world-guide).

### 5. Main loop

Your program usually spends the rest of its life in:

```corelx
while true
    ...
```

Inside the loop you typically do:
1. frame sync
2. input read
3. game logic update
4. sprite/OAM writes
5. `oam.flush()`

### 6. Frame timing (important right now)

Use the frame-edge pattern shown in the demo:

```corelx
while frame_counter() == last_frame
    wait_vblank()
last_frame = frame_counter()
```

Why: `wait_vblank()` is currently level-based, so this pattern avoids running logic multiple times within the same visible frame.

### 7. Input handling

The demo reads controller 1 and checks bitmasks:

```corelx
buttons := input.read(0)
```

Common low bits used in current examples:
- `0x01` = Up
- `0x02` = Down
- `0x04` = Left
- `0x08` = Right
- `0x10` = A button (mapped to `Z` in common test ROMs)

### 8. Edge-triggered buttons (press once)

For actions like color cycling, use current vs previous button state:

```corelx
if (buttons & 0x10) != 0 and (prev_buttons & 0x10) == 0
    -- do action once
```

This prevents repeating every frame while the button is held.

---

## Run the Demo (Dev Kit and CLI)

### Option A: Nitro-Core-DX App (Recommended)

```bash
go run ./cmd/corelx_devkit
```

Then:
1. Click **New** to create a project from a template, or **Open** to load `test/roms/devkit_moving_box_test.corelx`
2. Click **Build + Run**
3. Click the emulator pane and enable **Capture Game Input** if needed
4. Use **Split View**, **Emulator Focus**, or **Code Only** to arrange your workspace

### Option B: CLI Compile + Standalone Emulator

Compile:

```bash
go run ./cmd/corelx test/roms/devkit_moving_box_test.corelx /tmp/devkit_moving_box_test.rom
```

Run:

```bash
go run -tags no_sdl_ttf ./cmd/emulator -rom /tmp/devkit_moving_box_test.rom
```

---

## A Reusable CoreLX Game Template

Use this shape for small games and prototypes:

```corelx
function Start()
    ppu.enable_display()

    -- palettes
    -- gfx.load_tiles(...)
    -- state variables
    -- sprite setup

    last_frame := frame_counter()

    wait_vblank()
    -- initial oam writes
    oam.flush()

    while true
        while frame_counter() == last_frame
            wait_vblank()
        last_frame = frame_counter()

        buttons := input.read(0)

        -- update logic
        -- update sprites

        -- oam writes
        oam.flush()
```

This pattern is intentionally boring, and that is a good thing. It is easy to debug and easy to extend.

---

## Working with Sprites (Real-World Guide)

This section summarizes how sprites work in practice, based on real projects like `Games/SpriteProbe/ship.corelx`. Use it when you add characters, enemies, or UI elements.

### Sprite sizes and asset types

- **8×8** sprites use **32 bytes** of tile data (one 8×8 tile). Use asset type `tiles8` for a single 8×8 tile.
- **16×16** sprites use **128 bytes** (16 rows × 8 bytes). Use asset type `tiles16` for a single 16×16 tile, or `tileset` for a 128-byte block (e.g. one 16×16 “tile” stored as a tileset).

The compiler uploads tile data to VRAM using the correct stride: 32 bytes per 8×8 tile index, 128 bytes per 16×16 tile index. For a 128-byte `tileset`, loading at base index **N** writes to VRAM at **N × 128**, which matches how the PPU looks up 16×16 sprite tiles.

### Tile index and the background

- The **background** often uses **tile index 0** for empty or repeated cells. If you load your sprite at tile index **0**, that same art can appear everywhere the background draws tile 0 (a repeating pattern).
- **Best practice:** Load sprites at **non-zero** tile indices (e.g. 16, 17, 18). That keeps tile 0 for the BG and puts each sprite in its own VRAM region.
- Example: `ship_base := gfx.load_tiles(ASSET_Ship, 16)` and `ufo_base := gfx.load_tiles(ASSET_UFO, 17)`.

### Palettes

- Each sprite uses one **palette bank** (0–3). Set colors with `gfx.set_palette(bank, color_index, rgb555)`.
- **Color index 0** is always **transparent** for sprites; use indices 1–15 for visible colors.
- Call `gfx.init_default_palettes()` once if you want a sane default backdrop; then set the palette entries your sprites use (e.g. bank 1 for ship, bank 2 for UFO).

### One-time setup order

1. `ppu.enable_display()`
2. `gfx.init_default_palettes()` (optional)
3. Set palette colors for each bank you use: `gfx.set_palette(bank, index, rgb555)`
4. Load tile assets into VRAM at chosen tile indices: `base := gfx.load_tiles(ASSET_Name, tile_index)`
5. Create sprite structs and set position, tile, attr, ctrl (see below).

### Sprite struct: position, tile, attr, ctrl

- **Position:** `sprite.set_pos(&spr, x, y)` — screen coordinates (0–319 x, 0–199 y).
- **Tile:** `spr.tile = base` — the tile index returned by `gfx.load_tiles` (e.g. 16 or 17).
- **Attributes:** `spr.attr = SPR_PAL(bank) | SPR_PRI(priority)` — palette bank 0–3, priority 0–3.
- **Control:** `spr.ctrl = SPR_ENABLE() | SPR_SIZE_16()` or `SPR_SIZE_8()` — enable the sprite and choose 8×8 or 16×16.

Combine with optional flip/blend helpers as needed: `SPR_HFLIP()`, `SPR_VFLIP()`, etc.

### Multiple sprites

- Each sprite goes in a different **OAM slot**. Write them in order, then flush once per frame:
  ```corelx
  oam.write(0, &ship_spr)
  oam.write(1, &ufo_spr)
  oam.flush()
  ```
- Use different **tile bases** (different `gfx.load_tiles(..., 16)`, `..., 17)`) and different **palette banks** (`SPR_PAL(1)`, `SPR_PAL(2)`) so each sprite has its own art and colors.

### Per-frame loop

Each frame:

1. Sync (e.g. `wait_vblank()` and/or the frame-edge pattern with `frame_counter()`).
2. Update game state (positions, etc.).
3. Update sprite positions: `sprite.set_pos(&spr, x, y)`.
4. Write every visible sprite: `oam.write(0, &spr1)`, `oam.write(1, &spr2)`, …
5. `oam.flush()` so the PPU sees the new OAM data.

### Working example

A minimal two-sprite example (ship + UFO) that matches real behavior is in the repo:

- **Source:** `Games/SpriteProbe/ship.corelx`
- **Automated test:** `go test ./Games/SpriteProbe/... -run TestShip -v` (checks that the ship region is not black and writes an ASCII dump to `testdata/ship_actual.txt`).

That file shows: two 16×16 `tileset` assets, loading at tile indices 16 and 17, two palette banks, two sprites in OAM slots 0 and 1, and a simple vblank loop with `oam.write` + `oam.flush()`.

---

## Build a Small Pong-Style Game (Mini Pong)

This section walks through a small Pong-style game and points to a tested source file:

- Source: `test/roms/tutorial_pong.corelx`
- Smoke test: `internal/corelx/tutorial_pong_smoke_test.go`

### What the example includes

- left paddle controlled by player (Up/Down or W/S)
- right paddle controlled by simple AI
- moving ball with wall and paddle collisions
- score pips (5 per side) using sprites
- backdrop flash on score/reset
- reset scores/serve with `Z` (A button)

### Why this is a good tutorial project

It uses the same core building blocks as many arcade-style games:
- sprites
- per-frame state updates
- input
- collision checks
- simple AI
- UI feedback (score pips / screen flash)

### Step 1: Define your assets

The example uses three `tiles16` assets:
- `PaddleTile`
- `BallTile`
- `ScorePipTile`

Keeping all three as `tiles16` avoids mixed-size VRAM placement math in the tutorial.

### Step 2: Set palettes and load tiles

In `Start()`:

```corelx
ppu.enable_display()
gfx.set_palette(0, 0, 0x0000)  -- backdrop black
...
paddle_base := gfx.load_tiles(ASSET_PaddleTile, 0)
ball_base := gfx.load_tiles(ASSET_BallTile, 1)
pip_base := gfx.load_tiles(ASSET_ScorePipTile, 2)
```

This gives you tile base IDs for later sprite writes.

### Step 3: Set up game state variables

The example stores everything as plain variables:
- paddle positions
- ball position and velocity
- scores
- flash timer/color
- `prev_buttons` for edge-triggered reset
- `last_frame` for frame-edge timing

This is the simplest reliable approach with current CoreLX compiler behavior.

### Step 4: Update on frame edges

Just like the demo, the Pong example uses:

```corelx
while frame_counter() == last_frame
    wait_vblank()
last_frame = frame_counter()
```

That keeps movement stable and predictable.

### Step 5: Handle input and AI

Player paddle:

```corelx
if (buttons & 0x01) != 0
    if left_y > 4
        left_y = left_y - 2

if (buttons & 0x02) != 0
    if left_y < 180
        left_y = left_y + 2
```

AI paddle (very simple):

```corelx
if ball_y + 8 < right_y + 8
    if right_y > 4
        right_y = right_y - 1
if ball_y + 8 > right_y + 8
    if right_y < 180
        right_y = right_y + 1
```

### Step 6: Ball movement and collision

The game moves the ball each frame and bounces it:
- off top/bottom walls
- off either paddle when bounding boxes overlap

It also tweaks `ball_vy` based on where the ball hits the paddle to make rallies feel less flat.

### Step 7: Score and feedback

When the ball goes off-screen:
- score increments for the correct side
- ball resets to center
- serve direction flips
- backdrop flashes briefly

Because text/UI APIs are still evolving, the tutorial uses **sprite score pips** instead of text rendering.

### Step 8: Render with OAM writes + flush

The example writes:
- paddle sprites (slots `0` and `1`)
- ball sprite (slot `2`)
- score pip sprites (slots `3..12`)

Then commits with:

```corelx
oam.flush()
```

### Full Source (Tested)

Use this file directly:

- `test/roms/tutorial_pong.corelx`

Compile manually:

```bash
go run ./cmd/corelx test/roms/tutorial_pong.corelx /tmp/tutorial_pong.rom
```

Run in integrated app:
- Open `test/roms/tutorial_pong.corelx`
- Click `Build + Run`

Or run in standalone emulator:

```bash
go run -tags no_sdl_ttf ./cmd/emulator -rom /tmp/tutorial_pong.rom
```

---

## Test the Pong Example Before Publishing

Before publishing docs/tutorial changes, run the automated smoke test for the Pong example:

```bash
go test ./internal/corelx -run TestTutorialPongCompileAndRunSmoke -v
```

What this test checks (`internal/corelx/tutorial_pong_smoke_test.go`):
- `tutorial_pong.corelx` compiles successfully through the production compiler API
- ROM bytes load in the emulator
- the game renders visible pixels (not a black screen)
- the ball sprite moves across frames
- scripted input moves the left paddle up and back down

This is not a full gameplay correctness proof, but it is a strong publish gate for a tutorial example.

---

## Using Project Templates

The Dev Kit includes built-in project templates to accelerate getting started. Click **New** in the toolbar to open the template dialog.

### Available Templates

| Template | Description |
|----------|-------------|
| **Blank Game** | Clean project with display enabled and a vblank loop |
| **Minimal Loop** | Frame-cadence scaffold for gameplay update loops |
| **Sprite Demo** | Sprite placement and OAM flush testing |
| **Tilemap Demo** | Tilemap upload and camera experiments |
| **Shmup Starter** | Vertical shoot-em-up scaffold with player movement and bullet firing |
| **Matrix Mode Demo** | Matrix transform experiments |

Templates generate valid CoreLX code that compiles and runs immediately with **Build + Run**. They are a good way to learn common patterns (game loops, input, sprites, bullets) without writing boilerplate from scratch.

---

## Using the Sprite Lab

The Sprite Lab is a built-in pixel-art editor accessible from the workbench tabs in the Dev Kit. It lets you design sprites visually and export them directly into your CoreLX code.

### Quick Workflow

1. Switch to the **Sprite Lab** tab in the workbench area
2. Choose a canvas size (8x8 to 32x32 in steps of 8)
3. Select a color from the palette and paint with the Pencil tool
4. Adjust colors using RGB555 channel editors, full hex entry, or the palette value slider
5. Click **Apply To Manifest** (recommended) to write the asset into `corelx.assets.json`, or use **Insert CoreLX Asset** / **Apply To Project** for in-source workflows

### Key Features

- **16 palette banks** with 16 colors each (RGB555 format)
- **Mirror X** painting for symmetric sprites
- **Grid overlay** with hover highlighting
- **Wrap-shift controls** to move sprite pixels up/down/left/right with edge wrapping
- **Undo/Redo** history (up to 128 states)
- **Import/Export** `.clxsprite` files for saving and sharing sprite assets
- **Preview pane** showing actual-size sprite and packed 4bpp hex (aspect-preserving scale)
- **Apply To Manifest** for compiler-ingested asset upserts
- **Apply To Project** upsert flow that updates existing matching asset/palette blocks
- **Transparent index 0** checkerboard mode for no-color pixels

### Sprite Lab to Game Workflow

1. Design your sprite in the Sprite Lab
2. Click **Apply To Manifest** (recommended), or **Insert CoreLX Asset** / **Apply To Project**
3. Switch back to the Code tab — the asset declaration and palette init code are appended
4. Reference the asset in your game code: `tile_base := gfx.load_tiles(ASSET_YourSprite, 0)`
5. Click **Build + Run** to see it in action

---

## Using the Tilemap Lab

The Tilemap Lab is the map-authoring companion to Sprite Lab in the workbench tabs.

### Quick Workflow

1. Open the **Tilemap Lab** tab
2. Choose map size (8x8 to 32x32)
3. Set brush tile index/palette/flip attributes and paint/fill
4. Use **Tile Source** refresh to parse tiles from current source asset declarations
5. Pick tiles directly from the atlas view
6. Use **Apply To Manifest** (recommended), or **Insert CoreLX Asset** / **Apply To Project**
7. In CoreLX, point a BG layer at a tilemap base and load the asset with `bg.load_tilemap(ASSET_YourMap, layer)`

### Current Behavior Notes

- Tilemap entries are stored as packed `(tile, attr)` pairs
- `attr` includes palette and flip flags
- Tile atlas rendering uses parsed in-source tile assets and current palette assignments when available
- Import/export uses `.clxtilemap`
- Runtime BG rendering currently consumes **32x32** tilemaps. The lab can still author larger canvases, but larger-than-32x32 content is not yet a first-class runtime path until large-world tilemap support lands.

---

## Project Asset Manifest (`corelx.assets.json`)

In the Dev Kit build path, the compiler automatically checks for `corelx.assets.json` next to the active `.corelx` source file.

- Manifest assets are merged with in-source `asset` declarations
- The compiler remains the source of truth for final emitted manifest/ROM mapping
- Editor/lab flows are editing helpers; validate state with **Build** or **Build + Run** and the top-bar **Build State** indicator

---

## Where to Go Next

After you understand the demo and Mini Pong, good next projects are:

1. Add sound effects (paddle hit / score beep) using the legacy APU built-ins (current stable path before YM2608 migration)
2. Add two-player Pong (map controller 2 or alternate keyboard scheme)
3. Add a title screen and game-over state
4. Replace score pips with a tilemap/text-based score display once your UI path is ready
5. Build a Breakout-style game using the same paddle/ball collision logic

Related docs:
- `PROGRAMMING_MANUAL.md` (broader programming guide)
- `docs/CORELX.md` (language reference, under active alignment)
- `docs/testing/README.md` (current test entry points)
- `test/roms/devkit_moving_box_test.corelx` (starter demo)
- `test/roms/tutorial_pong.corelx` (small game example)
