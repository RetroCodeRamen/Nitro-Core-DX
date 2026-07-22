# Nitro-Core-DX Hardware Features Status

**Last Updated:** July 22, 2026

This document tracks the implementation status of hardware-facing emulator
features for the console itself. It does not track CoreLX language completion
or Dev Kit UI/tool readiness; use `README.md`, `docs/README.md`, and
`docs/planning/NEXT_STEPS_PLAN.md` for product/tooling status.

---

## ✅ Fully Implemented

### CPU
- ✅ Complete instruction set (arithmetic, logical, branching, jumps, stack)
- ✅ 8 general-purpose registers (R0-R7)
- ✅ 24-bit banked addressing
- ✅ Cycle counting
- ✅ Flag management (Z, N, C, V, I, D)
- ✅ Stack operations

### Memory System
- ✅ Banked memory architecture (256 banks × 64KB = 16MB)
- ✅ WRAM (32KB in bank 0)
- ✅ Extended WRAM (128KB in banks 126-127)
- ✅ ROM loading and mapping
- ✅ I/O register routing (PPU, APU, Input)

### PPU (Graphics)
- ✅ VRAM, CGRAM, OAM management
- ✅ 4 background layers (BG0-BG3)
- ✅ Basic tile rendering (4bpp)
- ✅ Hardware sprite rendering with size codes from 8×8 through 128×128
- ✅ Sprite flip (X/Y)
- ✅ Sprite transparency (color index 0)
- ✅ Larger-sprite tile-grid addressing for 32×16 and above
- ✅ Per-scanline sprite pixel-fetch budget with priority-ordered dropping
- ✅ Windowing system structure
- ✅ **Matrix Mode (per-layer)** - ✅ NEWLY COMPLETED
  - ✅ Per-layer matrix transformations (BG0-BG3)
  - ✅ Affine transformation (rotation, scaling, perspective)
  - ✅ Matrix registers for all layers
  - ✅ Per-scanline HDMA scroll updates
  - ✅ Per-scanline HDMA matrix updates
- ✅ HDMA per-layer scroll/matrix table processing in emulator runtime

### APU (Audio) — YM2608 / OPNA (final audio subsystem)
- ✅ YM2608/OPNA MMIO host interface (`0x9100-0x91FF`) — FM, SSG, rhythm, ADPCM
- ✅ Sample generation at 44,100 Hz
- ✅ YM2608 audio operational through the YMFM-backed runtime
- ✅ Bus-side YM burst streamer (`0x9110-0x9115`) for efficient register-stream playback
- ✅ Compact `.ncdxmusic` stream playback path exists above the hardware runtime
- 🚧 YM2608 hardware conformance under active refinement (not yet fully verified)

**Legacy 4-channel synth — temporary migration scaffolding (not final hardware):**
- ⚠️ 4 audio channels, waveforms (sine/square/saw/noise), per-channel
  frequency/volume/duration, master volume, PCM playback — retained only to keep
  existing CoreLX/ROMs working during the YM2608 migration; slated for removal.

### Input System
- ✅ Controller 1 & 2 support
- ✅ 12-button support (UP, DOWN, LEFT, RIGHT, A, B, X, Y, L, R, START, Z)
- ✅ Latch mechanism

### Synchronization
- ✅ VBlank flag (hardware-accurate)
- ✅ Frame counter
- ✅ Channel completion status
- ✅ Clock-driven architecture

---

## 🚧 Partially Implemented

### PPU Rendering
- ✅ **Sprite Priority** - ✅ NEWLY COMPLETED
  - ✅ Priority bits read from attributes (bits [7:6])
  - ✅ Priority-based sprite sorting and rendering order
  - ✅ Sprites sorted by priority, then by index
  - ✅ Scanline bandwidth pressure keeps higher-priority sprites first

- ✅ **Sprite-to-Background Priority** - ✅ NEWLY COMPLETED
  - ✅ Proper priority interaction between sprites and backgrounds
  - ✅ Unified priority system (BG3=3, BG2=2, BG1=1, BG0=0, Sprites=0-3)
  - ✅ Sprites can render behind backgrounds based on priority

- ✅ **Sprite Blending/Alpha** - ✅ NEWLY COMPLETED
  - ✅ Blend modes (normal, alpha, additive, subtractive)
  - ✅ Alpha transparency (0-15 levels)
  - ✅ Blending with backgrounds

---

## 🚧 Remaining / In Progress

### PPU Features
- ❌ **Vertical Sprites for Matrix Mode**
  - Documented but not implemented
  - This is separate from the now-implemented larger flat hardware sprites
  - **Needs:** Sprites that scale/position based on Matrix Mode transformation
  - **Needs:** Depth sorting for 3D sprites
  - **Needs:** World coordinate system for sprites

### APU Features
- 🚧 **YM2608 Conformance Refinement** (In Progress)
  - ✅ FM host interface (`FM_ADDR`, `FM_DATA`, `FM_STATUS`, `FM_CONTROL`, `FM_MIX_L/R`)
  - ✅ Timer/status/IRQ bridge behavior (deterministic runtime model)
  - ✅ Song replay and gameplay BGM playback working via YM write streams
  - ✅ `.ncdxmusic` compact stream assets can be replayed through the runtime path
  - ✅ Fixed-point sample generation is the canonical emulator runtime path
  - ✅ Legacy floating-point phase fields remain only as compatibility/savestate support
  - ❌ Final timbre/pitch parity tuning against expanded external references
  - ❌ Full subsystem parity coverage (SSG/ADPCM/rhythm edge behavior) in integration tests

### Advanced Features
- ❌ **Large World Tilemap Support**
  - Basic tilemap rendering works
  - **Needs:** Extended tilemap support, tile stitching, seamless large worlds

---

## Priority Recommendations

### High Priority (Active Implementation)
1. **YM2608 Conformance Accuracy/Polish** - Continue parity tuning and broader behavioral validation
2. **Sound Studio Runtime Integration Support** - Keep the emulator/YM stream path stable while the Dev Kit adds import/preview/export UI
3. **FPGA Parity Documentation/Planning** - Keep target hardware gaps explicit before new FPGA implementation pushes

### Medium Priority (Enhanced Features)
1. **Vertical Sprites for Matrix Mode** - Enables 3D sprite rendering
2. **Large World Tilemap Support** - Extended tilemap stitching/seamless world workflows

### Low Priority (Nice to Have)
1. **Additional FM Preset/Content Tooling** - Improve timbre authoring workflows once accuracy stabilizes
2. **Matrix-Mode-Oriented Visual Extras** - Advanced rendering extensions beyond current core target

---

## Summary

**Core Hardware:** ✅ **Stable and Software-Ready**
- CPU: ✅ Complete (including interrupts)
- Memory: ✅ Complete
- PPU: ✅ Core-complete (priority, blending, mosaic, DMA, Matrix Mode enhancements)
- APU: ✅ YM2608/OPNA audio subsystem operational with 🚧 conformance refinement in progress (legacy 4-channel synth retained only as temporary migration scaffolding)
- Input: ✅ Complete
- Matrix Mode: ✅ Per-layer transforms complete (outside-screen handling and direct color implemented)
- DMA/HDMA: ✅ Complete in emulator runtime (VRAM/CGRAM/OAM DMA plus per-scanline HDMA scroll/matrix updates)

**Optional Enhancements (Not Required for Core System):**
- Vertical sprites for Matrix Mode (advanced 3D feature - can be added later)
- Large world tilemap workflows

(YM2608 conformance accuracy/polish is core audio work, not an optional
enhancement — tracked under APU Features above.)

**System Status:** ✅ **READY FOR SOFTWARE DEVELOPMENT**

All core hardware features are implemented and functional for game/application
development. The YM2608/OPNA audio subsystem is operational through the
YMFM-backed runtime and has a working register-stream playback path; current
remaining work is conformance/polish, not basic functionality. A legacy
4-channel synth remains in the tree only as temporary migration scaffolding.
