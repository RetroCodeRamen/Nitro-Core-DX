# Galaxy Force

**Flagship game for Nitro Core DX**

A vertical shmup with Tyrian-level bullet density, cinematic transitions, and a story-driven campaign.

## Quick Start

### Build

```bash
# From project root:
go run ./cmd/corelx Games/GalaxyForce/main.corelx Games/GalaxyForce/galaxy_force.rom
```

### Run in Emulator

```bash
go run -tags no_sdl_ttf ./cmd/emulator -rom Games/GalaxyForce/galaxy_force.rom
```

### Run in Dev Kit

```bash
go run ./cmd/corelx_devkit -file Games/GalaxyForce/main.corelx
```

## Controls

| Key        | Action      |
|------------|-------------|
| Arrow Keys | Move ship   |
| Z (A)      | Fire        |
| Enter      | Start       |

## Phase A: Vertical Slice

The current implementation includes:

- Title screen with flashing start prompt
- Level 1 vertical scrolling gameplay
  - Player ship with 4-directional movement
  - 8-slot bullet pool
  - 12-slot enemy wave spawner
  - Pickup drops (HP restore, score bonus)
  - Parallax background scrolling
- Boss fight with 3 phases
  - Phase 1: Slow horizontal movement, slow fire
  - Phase 2: Faster with vertical bounce, rapid fire
  - Phase 3: Erratic movement, aggressive fire
- Results screen with score/HP display
- Full game loop (title -> play -> boss -> results -> title)
- Sound effects (shoot, hit, pickup) and background music
- HP system (5 max, pickups restore)

## Architecture

Since CoreLX compiles single files, `main.corelx` contains all game logic organized via user-defined functions. Game state (bullets, enemies, pickups) is stored in WRAM via `mem.write`/`mem.read`.

### Sprite Slot Allocation

| Slots   | Purpose           |
|---------|-------------------|
| 0       | Player ship       |
| 1-5     | HP indicator      |
| 16-23   | Player bullets    |
| 32-43   | Enemies / Boss projectiles |
| 48-51   | Pickups           |
| 56-59   | Boss (4 tiles)    |
| 64-79   | Title text        |
| 80-95   | Results / UI      |

### WRAM Layout

| Address Range | Purpose              |
|---------------|----------------------|
| 0x0100-0x011F | Bullet pool (8 x 4B) |
| 0x0140-0x016F | Enemy pool (12 x 4B) |
| 0x0180-0x018F | Pickup pool (4 x 4B) |

## Automated testing (record/replay)

The game has a **record/replay harness** so behavior can be checked automatically:

- **Replay test** (default): Loads a golden recording from `testdata/galaxy_force_golden.json`, runs the same input script, and compares framebuffer (and audio) hashes each frame. Fails on first mismatch and writes the actual frame to `testdata/diff/`.

```bash
cd Games/GalaxyForce && go test -v -run TestGalaxyForceRecordReplay
```

- **Record a new golden**: After changing game behavior (or the first time), create/update the golden file:

```bash
cd Games/GalaxyForce && RECORD_GALAXY_FORCE=1 go test -v -run TestGalaxyForceRecordReplay
```

- **Export replay as PNGs for review**: Replay the golden and save every frame as an image plus `frames.json` (frame index + input per frame):

```bash
cd Games/GalaxyForce && go test -v -run TestGalaxyForceExportReplay
```

Output: `testdata/replay_frames/frame_00000.png` … `frame_00299.png`, and `frames.json`. Use this to visually confirm that the script does what you expect (title → Start → gameplay → one shot at frame 120).

The harness lives in `internal/harness` and can be reused for other games: script input per frame, record hashes (or optionally full pixels), then replay and compare.

## Roadmap

- **Phase B:** Branching missions, save/load, weapon families, dialog choices
- **Phase C:** 8-10 levels, horizontal segments, tunnel stages, UI polish
