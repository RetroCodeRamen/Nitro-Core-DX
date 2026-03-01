# Nitro-Core-DX Programming Guide

**Last Updated**: 2026-02-28  
**Purpose**: A practical CoreLX-first guide that starts with a working demo, explains program structure, walks through templates and the Sprite Lab, and builds a small Pong-style game.

> This guide is intentionally hands-on and aligned to the current Dev Kit + CoreLX workflow.
> For broader platform context and reference material, also see `PROGRAMMING_MANUAL.md` and `docs/CORELX.md`.

---

## Table of Contents

1. [Start Here: The Current Demo Program](#start-here-the-current-demo-program)
2. [How a CoreLX Program Is Structured](#how-a-corelx-program-is-structured)
3. [Run the Demo (Dev Kit and CLI)](#run-the-demo-dev-kit-and-cli)
4. [A Reusable CoreLX Game Template](#a-reusable-corelx-game-template)
5. [Build a Small Pong-Style Game (Mini Pong)](#build-a-small-pong-style-game-mini-pong)
6. [Test the Pong Example Before Publishing](#test-the-pong-example-before-publishing)
7. [Where to Go Next](#where-to-go-next)

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

`oam.flush()` is what commits the queued OAM writes.

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
2. Choose a canvas size (8x8 to 64x64 in steps of 8)
3. Select a color from the palette and paint with the Pencil tool
4. Adjust colors using the RGB555 channel editors or hex input
5. Click **Insert CoreLX Asset** to generate tile hex data + palette setup code into the active editor

### Key Features

- **16 palette banks** with 16 colors each (RGB555 format)
- **Mirror X** painting for symmetric sprites
- **Grid overlay** with hover highlighting
- **Undo/Redo** history (up to 128 states)
- **Import/Export** `.clxsprite` files for saving and sharing sprite assets
- **Preview pane** showing actual-size sprite and packed 4bpp hex

### Sprite Lab to Game Workflow

1. Design your sprite in the Sprite Lab
2. Click **Insert CoreLX Asset** (optionally with palette setup code)
3. Switch back to the Code tab — the asset declaration and palette init code are appended
4. Reference the asset in your game code: `tile_base := gfx.load_tiles(ASSET_YourSprite, 0)`
5. Click **Build + Run** to see it in action

---

## Where to Go Next

After you understand the demo and Mini Pong, good next projects are:

1. Add sound effects (paddle hit / score beep) using the legacy APU built-ins
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
