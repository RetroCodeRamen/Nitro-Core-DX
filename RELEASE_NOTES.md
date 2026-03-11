# Nitro-Core-DX v0.1.9

## What Changed (Plain-English)

This release pushes the emulator baseline forward in the areas that actually matter for software development: audio, graphics, CoreLX control surface, and test content.

### 1) YM2608 Is Now The Real Runtime Audio Path

The legacy FM fallback path is gone. The emulator and Dev Kit now run against the YMFM-backed YM2608 path directly.

What that means for you:
- audio testing is now happening against the intended sound path
- bundled demo ROMs exercise the current YM2608 runtime directly
- release users are no longer testing a fallback audio stack by accident

### 2) Matrix / Mode-7-Style Graphics Took A Big Step Forward

The PPU now has a much more serious matrix-plane implementation than before:

- dedicated matrix-plane tilemap memory
- dedicated matrix-plane pattern memory
- bitmap-backed matrix planes in the emulator
- explicit outside behavior, including clamp
- larger source sizes aimed at SNES-class `1024x1024` floor/background use

What that means for you:
- matrix planes are no longer just “small rotating tilemaps”
- large floor/background experiments are now practical
- the graphics pipeline is much closer to a real pseudo-3D baseline

### 3) CoreLX Can Drive Matrix Planes Directly

CoreLX now has first-class helpers for matrix-plane setup and authored content:

- `matrix_plane.enable(...)`
- `matrix_plane.disable(...)`
- `matrix_plane.load_tiles(...)`
- `matrix_plane.load_tilemap(...)`
- `matrix_plane.set_tile(...)`
- `matrix_plane.fill_rect(...)`
- `matrix_plane.clear(...)`

What that means for you:
- you can author and load dedicated matrix-plane content without dropping to raw MMIO
- the programming manual now documents the supported matrix-plane workflow

### 4) The Release Package Now Includes Test ROMs

Both release archives now include two ROMs in `roms/`:

- `pong_ym2608_demo.rom`
  - gameplay + YM2608 audio validation
- `matrix_floor_only_kart.rom`
  - dedicated matrix-floor validation using the kart image path

What that means for you:
- users can test the current audio/runtime path immediately
- users can test the current matrix-floor path immediately

## Why v0.1.9 Matters

This release is about moving the emulator from “feature experiments” toward a usable software platform:

- the audio path is cleaner and more intentional
- the matrix-plane architecture is much stronger
- the language surface is better aligned with the graphics hardware model
- the release downloads now include real validation content, not just the app binary

## Downloads

- **Linux:** `nitrocoredx-v0.1.9-linux-amd64.tar.gz`
- **Windows:** `nitrocoredx-v0.1.9-windows-amd64.zip`

Both downloads include:
- the integrated Nitro-Core-DX app
- `roms/pong_ym2608_demo.rom`
- `roms/matrix_floor_only_kart.rom`
