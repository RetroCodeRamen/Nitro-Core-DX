# Nitro-Core-DX Hardware Features Status

**Last Updated:** February 28, 2026

This document tracks the implementation status of all hardware features for the emulated console itself (not emulator UI or dev tools). For Dev Kit and tooling status, see the README.md project status section.

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
- ✅ Basic sprite rendering (8×8, 16×16)
- ✅ Sprite flip (X/Y)
- ✅ Sprite transparency (color index 0)
- ✅ Windowing system structure
- ✅ **Matrix Mode (per-layer)** - ✅ NEWLY COMPLETED
  - ✅ Per-layer matrix transformations (BG0-BG3)
  - ✅ Affine transformation (rotation, scaling, perspective)
  - ✅ Matrix registers for all layers
  - ✅ Per-scanline HDMA matrix updates
- ✅ HDMA structure (scroll updates)

### APU (Audio)
- ✅ 4 audio channels
- ✅ All waveform types (sine, square, saw, noise)
- ✅ Frequency control
- ✅ Volume control (per channel + master)
- ✅ Duration control with loop mode
- ✅ Sample generation at 44,100 Hz
- ✅ PCM playback support (per-channel)
- 🚧 FM extension MMIO host interface (`0x9100-0x91FF`) and OPM-lite audible synthesis path (in progress)

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

- ✅ **Sprite-to-Background Priority** - ✅ NEWLY COMPLETED
  - ✅ Proper priority interaction between sprites and backgrounds
  - ✅ Unified priority system (BG3=3, BG2=2, BG1=1, BG0=0, Sprites=0-3)
  - ✅ Sprites can render behind backgrounds based on priority

- ✅ **Sprite Blending/Alpha** - ✅ NEWLY COMPLETED
  - ✅ Blend modes (normal, alpha, additive, subtractive)
  - ✅ Alpha transparency (0-15 levels)
  - ✅ Blending with backgrounds

- 🚧 **HDMA Scroll Updates** - Structure exists
  - HDMA table reading implemented
  - **Needs:** Full per-layer scroll HDMA support

---

## 🚧 Remaining / In Progress

### PPU Features
- ❌ **Vertical Sprites for Matrix Mode**
  - Documented but not implemented
  - **Needs:** Sprites that scale/position based on Matrix Mode transformation
  - **Needs:** Depth sorting for 3D sprites
  - **Needs:** World coordinate system for sprites

### APU Features
- 🚧 **FM Synthesis Extension** (In Progress)
  - ✅ FM host interface (`FM_ADDR`, `FM_DATA`, `FM_STATUS`, `FM_CONTROL`, `FM_MIX_L/R`)
  - ✅ Timer/status/IRQ bridge behavior (deterministic placeholder timing)
  - ✅ Audible OPM-lite subset (software-first, hardware-oriented)
  - ❌ Full YM2151/OPM behavior accuracy (future work)
  - ❌ Final patch/envelope/timbre polish (future work)

### Advanced Features
- ❌ **Large World Tilemap Support**
  - Basic tilemap rendering works
  - **Needs:** Extended tilemap support, tile stitching, seamless large worlds

---

## Priority Recommendations

### High Priority (Active Implementation)
1. **HDMA Full Implementation** - Per-layer scroll updates beyond current structure/table read path
2. **FM Synthesis Extension Accuracy/Polish** - Continue from OPM-lite subset toward fuller YM2151 behavior

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
- APU: ✅ Complete (legacy audio + PCM) with 🚧 FM extension in progress
- Input: ✅ Complete
- Matrix Mode: ✅ Per-layer transforms complete (outside-screen handling and direct color implemented)
- DMA: ✅ Complete (VRAM/CGRAM/OAM transfers)

**Optional Enhancements (Not Required for Core System):**
- Vertical sprites for Matrix Mode (advanced 3D feature - can be added later)
- FM extension accuracy/polish (already started; continue incrementally)
- Large world tilemap workflows

**System Status:** ✅ **READY FOR SOFTWARE DEVELOPMENT**

All core hardware features are implemented and functional for game/application development. The legacy audio path is complete; the FM extension is now actively implemented but still evolving toward fuller OPM/YM2151 compatibility.
