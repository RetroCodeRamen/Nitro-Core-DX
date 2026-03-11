# Nitro-Core-DX PPU Transform Channel Architecture

**Version 0.1 (Stage 1 Architectural Baseline)**  
**Date:** March 10, 2026  
**Purpose:** Define the target PPU architecture for multi-plane affine rendering, raster-time reassignment, and future FPGA parity work.

> **Stage 1 Scope:** This document is an architecture/specification pass only. It does not claim that the current Go emulator or in-tree FPGA RTL already implements the full model below.

## Why This Exists

The current Go PPU already supports:
- 4 independent background layers
- 4 runtime transform channels bound to those layers
- per-scanline command-table updates for scroll, transform, rebinding, priority, and tilemap base
- sprite/background priority and blending

That is enough for useful Mode-7-style visuals today, but it is not yet the full long-term architecture target.

The main limitation is no longer transform ownership. The current runtime now uses dedicated transform channels, but the public register contract and FPGA RTL still trail the intended long-term architecture.

This document defines that target architecture so future emulator and FPGA work can converge on one contract.

For the per-plane matrix source-size and dedicated-memory baseline, see:

- [PPU_MATRIX_PLANE_MEMORY_SPEC.md](./PPU_MATRIX_PLANE_MEMORY_SPEC.md)

## Current Implementation Snapshot

Current Go PPU behavior:
- Background layers `BG0`-`BG3` bind to dedicated runtime transform channels.
- Scanline command tables can update scroll, transform coefficients, channel binding, explicit priority, and tilemap base at scanline boundaries.
- Composition is done per pixel by priority-sorting active backgrounds and sprites.
- Raster-time behavior is table-driven at scanline boundaries, not interrupt-driven beyond VBlank.

Current FPGA RTL behavior:
- Single matrix path in `ppu_core.v`
- Fixed VRAM/tilemap assumptions
- Not yet at parity with Go PPU for per-layer transform behavior

This means the emulator is the stronger implementation today, but both emulator and RTL need a cleaner shared architecture model before larger pseudo-3D features are added.

## Target Architecture

### 1. Visible Layers

The PPU should continue to expose **4 visible background layers**.

Each visible layer should define:
- enable
- source mode
- tilemap/bitmap configuration
- tile size or source-format metadata
- transform channel binding
- layer priority
- blend/transparency behavior
- windowing enable/mask behavior

Visible layers are presentation surfaces. They should not permanently own transform engines.

### 2. Transform Channels

The hardware should provide at least **4 transform channels**.

Each transform channel should contain:
- affine matrix `A/B/C/D` (8.8 fixed point unless later revised)
- pivot/origin `CenterX/CenterY`
- scroll offsets `ScrollX/ScrollY`
- outside behavior:
  - wrap
  - backdrop
  - character-0 fallback
  - optional future clamp mode
- source interpretation flags
- optional direct-color/source-format flags

Transform channels are independent of visible layer identity.

### 3. Layer-to-Channel Binding

Each visible layer should bind to one transform channel or to no transform channel.

This enables:
- normal tilemap layers
- transformed tilemap layers
- future reassignment of transform channels between layers

### 4. Source Model

Stage 1 architectural contract supports two source classes:
- `tilemap`
- `bitmap` (future-capable; not required in current runtime)

Current implementation only uses tilemap-backed transformed layers. Bitmap source support is part of the target architecture and must be included in the register contract before implementation.

### Tooling / Asset Pipeline Requirements

The graphics toolchain must eventually support import from common image formats such as:
- PNG
- JPG/JPEG

Expected pipeline behavior:
- quantize or reduce fidelity into console-native indexed/tile formats
- emit Sprite Lab / tile / tilemap friendly assets instead of raw image blobs
- preserve a clear mapping between imported source art and runtime tile/sprite assets

This is not only a Dev Kit feature request. It affects the long-term graphics contract because bitmap-source and transformed-source workflows need a stable authored asset path.

### Large Transform Source Requirement

For pseudo-3D use cases such as kart-racer floor planes, transformed source imagery may need to be materially larger than the currently visible map region.

Implications:
- transformed source size must not be assumed equal to current 32x32 tilemap visibility
- future bitmap/tilemap source work needs explicit limits for:
  - source width/height
  - wrapping/clamping policy
  - streaming/cache strategy
- large-source support is an architecture concern, not just a content/tooling concern

## Raster-Time Update Model

### Scanline Command / Parameter Tables

The preferred raster model is **table-driven scanline updates**, not arbitrary mid-dot register mutation.

At minimum, a scanline update format should support:
- transform channel parameter updates
- layer scroll updates
- layer-to-channel rebinding
- layer priority updates
- layer tilemap-base updates
- optional source-mode changes

This model is preferred because it is:
- deterministic in the emulator
- practical for FPGA control logic
- cheaper than free-form CPU-timed raster writes

Current runtime baseline:
- base scanline payload: `64` bytes (`4` layers × `16` bytes)
- optional rebind table: `4` bytes
- optional priority table: `4` bytes
- optional tilemap-base table: `8` bytes
- optional source-mode table: `4` bytes via extension control register

### Raster Interrupts

Raster interrupts should be considered **optional support**, not the primary control model.

If added, they should be limited to:
- scanline compare interrupt
- predictable HBlank/scanline-safe update points

The primary architecture should remain command-table based.

## Pipeline Model

The target hardware-minded render pipeline is:

1. Frame start/reset of line-local state
2. Scanline setup
3. Raster command/parameter fetch
4. Sprite evaluation for current scanline
5. Background source address generation
6. Transform address generation (for bound transform channels)
7. Tile/pattern or source fetch
8. Palette/direct-color resolve
9. Priority/blend composition
10. Pixel output

This is the conceptual contract future FPGA work should follow, even if the emulator continues to use a software-oriented implementation internally.

## Register Design Direction

### Layer Registers

Each visible layer should eventually expose registers for:
- `ENABLE`
- `SOURCE_MODE`
- `TILE_SIZE` / source-format mode
- `SOURCE_BASE` / tilemap base
- `TRANSFORM_BIND`
- `PRIORITY`
- `BLEND_CONTROL`
- `WINDOW_MASK`

### Transform Channel Registers

Each transform channel should eventually expose:
- `A/B/C/D`
- `CENTER_X/CENTER_Y`
- `SCROLL_X/SCROLL_Y`
- `OUTSIDE_MODE`
- `DIRECT_COLOR` / source flags

### Raster Control Registers

Raster control should eventually expose:
- command-table base
- command-table enable
- scanline command mode/config
- optional raster IRQ compare / status / ack

## Memory Bandwidth Implications

This architecture increases pressure on memory bandwidth compared to the current implicit per-layer model.

Key costs:
- scanline command fetch
- transformed tilemap/source fetch
- sprite fetch/evaluation
- palette/direct-color resolve

Implications for future work:
- transformed tilemap source remains practical
- bitmap-source planes must not be added casually
- FPGA implementation likely needs:
  - explicit fetch scheduling
  - line-local caches or staged fetch
  - clear arbitration policy between CPU/PPU memory activity

Bandwidth/resource planning must be treated as part of the architecture, not as an afterthought.

## Sprite Integration Contract

Sprites should remain a separate rendering subsystem that coexists with transform planes.

Stage 1 target contract:
- sprites share the final priority/blend composition path
- sprites are not required to live in transform space
- sprite/background coexistence must support pseudo-3D presentation patterns

Future-capable additions:
- vertical sprite scaling
- pseudo-depth ordering rules
- optional world-position conventions for racing/floor-plane scenes

These are not required for Stage 1, but the architecture must not block them.

## Emulator and FPGA Feasibility

### Emulator

This target model is feasible in the emulator with moderate refactoring:
- move transform state into dedicated channel objects
- bind layers to channels
- extend HDMA into a clearer scanline command format

### FPGA

This target model is feasible in FPGA, but only if the current RTL evolves from:
- one matrix path

to:
- 4 transform-capable channels or a documented equivalent
- explicit scanline setup/control path
- explicit fetch/composition stages

The target is feasible; the current RTL is not yet close enough to claim parity.

## Stage Roadmap

### Stage 1: Architecture / Spec
- define transform channels
- define layer/channel binding model
- define raster command-table direction
- define register families

### Stage 2: Emulator Refactor
- separate layer state from transform-channel state
- preserve current visible behavior where possible
- keep tests green while moving to the new ownership model

### Stage 3: Capability Expansion
- channel rebinding
- explicit programmable priority
- optional raster IRQ support
- vertical-sprite / pseudo-3D integration work

### Stage 4: FPGA Parity
- implement transform-channel architecture in RTL
- align tilemap/fetch behavior with emulator contract
- close raster/path parity gaps

## Current Decision

For Nitro-Core-DX, the canonical long-term direction is:

- **4 visible layers**
- **at least 4 transform channels**
- **scanline command-table updates as the primary raster control model**
- **transform channels independent of visible-layer ownership**

That is the architecture future PPU implementation work should follow.
