# Design Analysis: Current Implementation vs. Target Design

## Executive Summary

Your current implementation is a solid **SNES-inspired fantasy console** with basic 2D graphics, simple audio, and a custom 16-bit CPU. To reach your ambitious **"Nitro-Core-DX"** vision (a hybrid SNES/Genesis with built-in 3D assist), you'll need significant architectural changes. This document breaks down what exists, what's missing, and what needs to change.

---

## 1. CPU Architecture

### Target Design
- **Main CPU**: 68000-class @ 10-12 MHz
- **I/O + Sound CPU**: Z80 (or compatible) dedicated to audio + peripherals
- **Fast DMA + bus arbitration**

### Current Implementation
- **Main CPU**: Custom 16-bit CPU (8 registers R0-R7, banked addressing)
- **CPU Speed**: 2.68 MHz (SNES-accurate timing: 44,667 cycles/frame)
- **No secondary CPU**: Single CPU handles everything
- **No DMA system**: All memory transfers are CPU-driven

### Gap Analysis
| Feature | Status | Gap |
|---------|--------|-----|
| CPU Type | ❌ Custom 16-bit | Need 68000-class architecture |
| CPU Speed | ❌ 2.68 MHz | Need 10-12 MHz (4-5x faster) |
| Secondary CPU | ❌ None | Need Z80 for I/O/audio |
| DMA | ❌ None | Need DMA controller + bus arbitration |

### Required Changes
1. **Replace custom CPU with 68000-class emulation** (or extend current CPU to 68000-like instruction set)
2. **Increase CPU speed** to 10-12 MHz (adjust `CPU_CYCLES_PER_FRAME` from 44,667 to ~166,667-200,000 cycles/frame)
3. **Add Z80 CPU emulation** for I/O and audio processing
4. **Implement DMA controller** with bus arbitration (CPU, Z80, PPU, DMA compete for memory bus)

---

## 2. Video System

### Target Design
- **4 background layers** + windowing + per-scanline scroll (parallax)
- **High sprite throughput** (arcade-friendly)
- **Priorities, blending tricks**
- **Affine layer** (Mode 7 but modernized) - **1 dedicated affine BG plane**
- **15-bit color** (SNES-style)

### Current Implementation
- **2 background layers** (BG0, BG1)
- **128 sprites max** (basic rendering, no priority/blending)
- **Matrix Mode** (affine transformation) - ✅ **EXISTS!** (on BG0)
- **15-bit color** (RGB555) - ✅ **EXISTS!**
- **No windowing system**
- **No per-scanline scroll**
- **No sprite priorities/blending**

### Gap Analysis
| Feature | Status | Gap |
|---------|--------|-----|
| Background Layers | ⚠️ 2 layers | Need 4 layers (BG0-BG3) |
| Windowing | ❌ None | Need windowing system |
| Per-Scanline Scroll | ❌ None | Need HDMA-style per-line scroll |
| Sprite Priority | ⚠️ Basic | Need proper priority system |
| Sprite Blending | ❌ None | Need alpha blending |
| Affine Layer | ✅ Matrix Mode | **EXISTS** - but only on BG0 |
| Color Depth | ✅ 15-bit RGB555 | **EXISTS** |

### Required Changes
1. **Add BG2 and BG3 layers** to `PPUState` and rendering pipeline
2. **Implement windowing system** (window masks, window enable per layer)
3. **Add per-scanline scroll** (HDMA-style registers for scroll X/Y per scanline)
4. **Enhance sprite system** with priority levels (0-3) and blending modes
5. **Keep Matrix Mode** - it's already implemented! Just needs to be on a dedicated affine layer (BG3?)

---

## 3. Audio System

### Target Design
- **FM synth core** (Genesis-style)
- **Sample playback** DSP-ish path (SNES-style)
- **Dedicated audio RAM** 64-128KB
- **Cartridge streaming support** (for bigger voices/drums)

### Current Implementation
- **4-channel synthesizer** (sine, square, saw, noise waveforms)
- **No FM synthesis** (no operators, no FM algorithms)
- **No sample playback** (no PCM channels)
- **No dedicated audio RAM** (audio state in APUState only)
- **No cartridge streaming** (no ROM-to-audio-RAM DMA)

### Gap Analysis
| Feature | Status | Gap |
|---------|--------|-----|
| FM Synthesis | ❌ None | Need YM2612-style FM (6 operators, algorithms) |
| Sample Playback | ❌ None | Need PCM channels (8-16 channels) |
| Audio RAM | ❌ None | Need 64-128KB dedicated RAM |
| Cartridge Streaming | ❌ None | Need DMA from ROM to audio RAM |

### Required Changes
1. **Implement FM synthesis** (YM2612-like: 6 operators, 4 algorithms, ADSR)
2. **Add PCM sample playback** (8-16 channels, 8-bit/16-bit samples)
3. **Add dedicated audio RAM** (64-128KB, accessible by Z80 and DMA)
4. **Implement cartridge streaming** (DMA from ROM to audio RAM during gameplay)

---

## 4. 3D Assist Block (SuperFX-style)

### Target Design
- **Fixed-point geometry + raster co-processor**
- **Option B: Chunky framebuffer + blit into tiles** (recommended)
- **8-32KB internal fast RAM**
- **Job queue (FIFO)**, interrupt when done, DMA-based transfers
- **Must not stall main CPU**

### Current Implementation
- **❌ No 3D assist block exists**

### Gap Analysis
| Feature | Status | Gap |
|---------|--------|-----|
| 3D Co-processor | ❌ None | Need entire 3D block |
| Geometry Engine | ❌ None | Need matrix transforms, clipping |
| Raster Engine | ❌ None | Need triangle rasterization |
| Framebuffer | ❌ None | Need 256x160 or 320x200 chunky buffer |
| Job Queue | ❌ None | Need FIFO command buffer |
| Interrupt System | ⚠️ Basic | Need 3D completion interrupt |

### Required Changes
1. **Create new `gau.py` (GFX-Assist Unit)** module:
   - Fixed-point math (matrix 3x3, 4x4)
   - Perspective divide assist (lookup table)
   - Clipping (near-plane + screen bounds)
   - Vector ops (dot products, normalization)
2. **Implement chunky framebuffer** (256x160 or 320x200, 8-bit indexed)
3. **Add rasterization** (flat-shaded triangles, optional Gouraud)
4. **Implement job queue** (FIFO command buffer, interrupt on completion)
5. **Add DMA support** for framebuffer → tilemap blit
6. **Map as I/O registers** (0xC000-0xCFFF range?)

---

## 5. Cartridge / Expansion Philosophy

### Target Design
- **Optional enhancement carts** (extra RAM, mapper chips, rare "3D boost" cart)
- **Base hardware already does 3D** (no cart required)

### Current Implementation
- **Basic LoROM mapper** (banks 1-125 for ROM)
- **No enhancement cart support**

### Gap Analysis
| Feature | Status | Gap |
|---------|--------|-----|
| Base Mapper | ✅ LoROM | **EXISTS** |
| Enhancement Carts | ❌ None | Need mapper detection + extra RAM support |
| 3D Boost Cart | ❌ N/A | Not needed if base has 3D |

### Required Changes
1. **Extend ROM header** to detect enhancement carts (mapper flags)
2. **Add mapper chip emulation** (for large ROMs)
3. **Support extra RAM on cart** (banks 128-255?)

---

## 6. Dev Experience (APIs/Libraries)

### Target Design
- **Official "microcode" command set** (transform, project, draw tri, draw quad)
- **Reference libraries**: sprite/tile engine, affine layer helper, polygon helper

### Current Implementation
- **No official APIs/libraries**
- **ROM builders are Python scripts** (not runtime libraries)

### Gap Analysis
| Feature | Status | Gap |
|---------|--------|-----|
| Microcode API | ❌ None | Need command set documentation |
| Reference Libraries | ❌ None | Need helper libraries (C/assembly?) |
| Dev Tools | ⚠️ Basic | Need better tooling |

### Required Changes
1. **Document 3D command set** (register-based API)
2. **Create reference libraries** (sprite engine, tile engine, polygon helper)
3. **Improve dev tools** (better ROM builder, debugger enhancements)

---

## Implementation Priority Roadmap

### Phase 1: Foundation (Critical Path)
1. **Upgrade CPU to 68000-class** (or extend current CPU)
2. **Increase CPU speed** to 10-12 MHz
3. **Add Z80 secondary CPU** for I/O/audio
4. **Implement DMA system** with bus arbitration

### Phase 2: Video Enhancements
1. **Add BG2 and BG3 layers**
2. **Implement windowing system**
3. **Add per-scanline scroll** (HDMA)
4. **Enhance sprite priorities/blending**

### Phase 3: Audio Overhaul
1. **Implement FM synthesis** (YM2612-like)
2. **Add PCM sample playback**
3. **Add dedicated audio RAM** (64-128KB)
4. **Implement cartridge streaming**

### Phase 4: 3D Assist Block (The Big One)
1. **Create GAU module** (geometry + raster)
2. **Implement chunky framebuffer**
3. **Add job queue + interrupt system**
4. **Implement DMA blit** (framebuffer → tilemap)

### Phase 5: Polish
1. **Enhancement cart support**
2. **Reference libraries**
3. **Dev tools improvements**

---

## What You Already Have (Keep These!)

✅ **Matrix Mode** - Your affine transformation system is solid!  
✅ **15-bit RGB555 color** - Perfect for SNES-style graphics  
✅ **Basic tile/sprite system** - Good foundation  
✅ **Banked memory architecture** - Flexible and extensible  
✅ **ROM format** - Clean header structure  

---

## Estimated Effort

- **Phase 1 (CPU/DMA)**: 2-3 weeks (major refactor)
- **Phase 2 (Video)**: 1-2 weeks (additive, less risky)
- **Phase 3 (Audio)**: 2-3 weeks (complex FM synthesis)
- **Phase 4 (3D Block)**: 3-4 weeks (largest feature)
- **Phase 5 (Polish)**: 1-2 weeks

**Total: ~9-14 weeks** of focused development

---

## Recommendations

1. **Start with Phase 1** - CPU/DMA is the foundation. Everything else depends on it.

2. **Keep Matrix Mode** - It's already great! Just make it a dedicated affine layer (BG3).

3. **3D Block is the killer feature** - This is what makes your console unique. Prioritize it after foundation.

4. **Consider incremental approach** - You could ship with 2 layers + Matrix Mode + basic 3D, then add more layers later.

5. **Z80 can wait** - You could start with main CPU handling audio, then offload to Z80 later.

---

## Questions to Consider

1. **Do you want full 68000 emulation, or extend your custom CPU?** (68000 is more authentic, custom is more flexible)
2. **How important is Z80 vs. main CPU audio?** (Z80 adds complexity but is more authentic)
3. **Should 3D block be optional?** (Base hardware vs. enhancement cart)
4. **What's your target performance?** (10-20 FPS for 3D scenes is realistic for 16-bit era)

---

**Bottom Line**: You're about **30-40% of the way** to your target design. The good news: Matrix Mode and color system are done! The bad news: CPU, audio, and 3D block need major work. But it's all achievable with focused effort.
