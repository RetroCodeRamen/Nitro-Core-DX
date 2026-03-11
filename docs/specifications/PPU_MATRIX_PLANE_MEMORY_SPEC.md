# Nitro-Core-DX Matrix Plane Memory Specification

**Version 0.1**  
**Date:** March 10, 2026  
**Scope:** Emulator-first architecture specification for Matrix Mode source sizing, dedicated matrix-plane memory, wrapping behavior, and the minimum per-plane capability target.

## Purpose

This document defines the baseline capability target for each Nitro-Core-DX matrix plane.

The target is not "some rotating tilemap." The target is:

- each matrix plane must meet or exceed **SNES Mode 7-class source capacity**
- each matrix plane must support affine transform behavior equivalent to SNES-style rotation/scaling
- each matrix plane must provide defined outside-area behavior
- each matrix plane must have enough source area that transformed floors/backgrounds do not obviously reset because the source is too small

For Nitro-Core-DX, this means **each matrix plane** should be treated as at least a:

- **1024x1024 pixel transformable source area**

This is the minimum baseline, not a pooled shared budget across all matrix planes.

## Current Implementation Review

Current matrix rendering path:

- [renderDotMatrixMode()](../../internal/ppu/scanline.go)
- [BackgroundLayer](../../internal/ppu/ppu.go)

Current source-size behavior:

- `TileSize = 8x8` or `16x16`
- `TilemapSize = 32x32` or `64x64`

Current maximum source area:

- `32x32 @ 8x8` = `256x256`
- `32x32 @ 16x16` = `512x512`
- `64x64 @ 8x8` = `512x512`
- `64x64 @ 16x16` = `1024x1024`

Current outside behavior:

- wrap
- backdrop
- character-0 fallback

Current renderer limitation:

- the matrix plane still reads from normal VRAM-backed tilemap/tile data
- there is no dedicated per-plane source memory model
- there is no explicit `128x128 @ 8x8` path

## Gap Against Target

### 1. Source Map Dimensions

Current implementation initially fell short because:

- it only reached `1024x1024` in the special case of `64x64` tiles with `16x16` cells
- it did **not** support a true SNES-class `128x128` tile source at `8x8`
- it did **not** define a scalable matrix-plane source model beyond the current BG tilemap assumptions

Current emulator progress:

- explicit size modes now support `32x32`, `64x64`, and `128x128`
- dedicated matrix-plane tilemap backing now exists per transform channel
- a matrix-plane upload aperture exists in PPU MMIO
- dedicated matrix-plane pattern memory now exists per plane with its own upload aperture

Still not complete:

- no polished image-import/tooling path for matrix-plane bitmap content yet
- no CoreLX bitmap-plane surface yet
- no FPGA-side implementation yet

Required minimum target per matrix plane:

- `128x128 tiles @ 8x8 = 1024x1024`

Current and future target per matrix plane:

- `128x128 @ 8x8` tile-backed mode
- larger tile-backed modes where practical
- bitmap-backed matrix sources for selected use cases

### 2. Effective Visible Transform Range

Current visible transform range is bounded directly by:

- source width
- source height
- outside behavior

So even when the affine matrix is correct, the visible floor/background may appear to reset early because the source itself is too small or too repetitive.

The target architecture must ensure:

- the transform range is large enough that motion reads as continuous
- the plane does not "obviously restart" after a small rotation span

### 3. Wrapping Behavior

Current outside behavior is not enough for a mature matrix-plane contract.

Required contract:

- `wrap`
- `backdrop`
- `clamp`
- optional `tile0/border` fallback

Current emulator progress:

- `wrap`
- `backdrop`
- `tile0`
- `clamp`

Notes:

- `wrap` is required for classic Mode 7-style repeating floors
- `clamp` is required for large authored planes and less repetitive pseudo-3D scenes
- `backdrop` is required for defined outside-space behavior

### 4. Memory / Layout Constraints

Current matrix planes still consume ordinary VRAM-style tilemap/tile storage.

That is not a good long-term model if each matrix plane is expected to independently meet or exceed SNES-class capacity.

Example cost for one 1024x1024 tile-backed plane:

- tilemap at `128x128 * 2 bytes = 32KB`
- tile graphics are additional

That already consumes too much of the current ordinary VRAM-style budget if treated casually, especially with multiple matrix planes.

So the target architecture needs **dedicated matrix-plane source memory**, not just "bigger BG tilemaps."

## Target Matrix Plane Memory Model

Each matrix plane should have its own source configuration, independent of ordinary BG assumptions.

Per plane:

- `SourceType`
  - `tilemap`
  - `bitmap`
- `SourceWidth`
- `SourceHeight`
- `TileSize`
  - `8x8`
  - `16x16`
- `SourceBase`
- `OutsideMode`
  - `wrap`
  - `backdrop`
  - `clamp`
  - optional `tile0`
- `Palette/DirectColor flags`

The plane should then sample from its own configured source space before composition into the regular PPU output.

## Dedicated Matrix Memory

Nitro-Core-DX may give matrix planes their own memory backing.

This is acceptable and aligns with the goal.

Recommended model:

- matrix planes use **dedicated matrix-source memory regions**
- normal BG layers continue to use ordinary tilemap/tile paths
- visible layers can bind to matrix channels that sample from matrix-source memory

Benefits:

- avoids overloading ordinary BG memory assumptions
- allows large transformable sources without distorting the rest of the PPU model
- creates room for future advanced graphics workflows
- supports imported floor/background content more naturally

## Minimum Per-Plane Baseline

The minimum baseline for each matrix plane should be:

- `1024x1024`
- affine transform with `A/B/C/D`
- center/pivot
- scroll offsets
- wrap/backdrop/clamp outside behavior

This is the minimum acceptable target.

Anything below that is below the intended Nitro-Core-DX matrix-plane baseline.

## Recommended Register Contract

The current single `TilemapSize` bit is not sufficient long-term.

Replace it with an explicit source-size configuration for matrix-capable layers/planes.

Recommended per-plane fields:

- `SOURCE_MODE`
- `SOURCE_BASE`
- `SOURCE_WIDTH`
- `SOURCE_HEIGHT`
- `TILE_SIZE`
- `OUTSIDE_MODE`
- `DIRECT_COLOR`

Recommended size encodings:

- `32x32`
- `64x64`
- `128x128`
- future bitmap sizes

## Emulator-First Implementation Plan

### Stage 1: Matrix Source Model

Add explicit matrix-plane source sizing:

- replace `TilemapSize bool` with a size enum
- support:
  - `32x32`
  - `64x64`
  - `128x128`

This should apply to the matrix path first.

### Stage 2: 128x128 Tilemap Support

Implement `128x128 @ 8x8` sampling in the emulator matrix renderer.

Requirements:

- explicit source width/height in renderer
- correct wrap behavior at 1024x1024
- correct outside behavior selection

### Stage 3: Dedicated Matrix Memory Backing

Move matrix-source storage out of the implicit ordinary BG assumptions.

Options:

- dedicated matrix-plane memory arrays in emulator
- matrix-source asset pages
- banked source loading for large planes

### Stage 4: Tooling / Asset Pipeline

Once matrix-plane memory exists, the content pipeline should support:

- large tile-backed matrix sources
- imported PNG/JPG conversion into matrix-plane-compatible assets
- bitmap-backed matrix-plane sources for image-style validation and content

## Impact on Current Pong Demo

The Pong demo exposed the architectural limit correctly.

The current floor reset is not just a content problem. It is evidence that:

- current source span is still too limited
- current wrap behavior is still tied to a relatively small repeated source

So Pong should be treated as a validation symptom, not the root problem.

## Decision

Nitro-Core-DX matrix planes will be designed around:

- **dedicated matrix-source memory**
- **minimum 1024x1024 source area per plane**
- **affine transform behavior at least matching SNES Mode 7 baseline**
- **multiple planes each individually meeting that baseline**

This is now the target contract for future emulator-first PPU work.
