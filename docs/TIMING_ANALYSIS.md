# Nitro-Core-DX Timing Analysis & Design Decision

**Created:** January 27, 2026  
**Status:** Design Review Needed

> **Historical Snapshot:** This document predates later timing fixes. Use `docs/TIMING_FIX_SUMMARY.md` and current hardware spec/test evidence for current timing values.

## Current Situation

### PPU Timing (Current Implementation)
- **220 scanlines** × **360 dots per scanline** = **79,200 cycles per frame**
- This is hardcoded in the PPU implementation
- Each dot = 1 CPU cycle

### CPU Speed Options

| CPU Speed | Cycles/Second | Cycles/Frame (60 FPS) | Status |
|-----------|---------------|----------------------|--------|
| **4.752 MHz** | 4,752,000 | **79,200** | ✅ Matches PPU timing |
| **7 MHz** | 7,000,000 | **116,667** | ⚠️ User's preference (~116k) |
| **10 MHz** | 10,000,000 | **166,667** | ❌ Current documentation says this |

### The Problem

1. **Documentation says 10 MHz** (166,667 cycles/frame)
2. **PPU uses 79,200 cycles/frame** (matches 4.752 MHz)
3. **User wants ~116,000 cycles/frame** (matches 7 MHz)
4. **Actual CPU cycles per frame** may be showing 237,594 (which would be ~14.26 MHz)

## Analysis

### Option 1: Keep PPU at 79,200 cycles, adjust CPU to match
- **CPU Speed**: 4.752 MHz
- **Cycles/Frame**: 79,200
- **Pros**: PPU timing is already correct
- **Cons**: Slower than desired, doesn't match user's ~116k preference

### Option 2: Adjust PPU to match 7 MHz CPU
- **CPU Speed**: 7 MHz
- **Target Cycles/Frame**: 116,667
- **PPU Adjustment Needed**: 
  - Current: 220 scanlines × 360 dots = 79,200
  - New: Need to calculate scanline/dot timing for 116,667 cycles
  - Options:
    - **Option 2a**: 324 scanlines × 360 dots = 116,640 cycles (close!)
    - **Option 2b**: 220 scanlines × 530 dots = 116,600 cycles (close!)
    - **Option 2c**: Keep 220 scanlines, adjust dots: 116,667 / 220 = 530.3 dots/scanline

### Option 3: Adjust PPU to match 10 MHz CPU
- **CPU Speed**: 10 MHz
- **Target Cycles/Frame**: 166,667
- **PPU Adjustment Needed**:
  - Current: 220 scanlines × 360 dots = 79,200
  - New: Need to calculate scanline/dot timing for 166,667 cycles
  - Options:
    - **Option 3a**: 463 scanlines × 360 dots = 166,680 cycles (close!)
    - **Option 3b**: 220 scanlines × 757 dots = 166,540 cycles (close!)
    - **Option 3c**: Keep 220 scanlines, adjust dots: 166,667 / 220 = 757.6 dots/scanline

## Recommendation: 7 MHz CPU (~116k cycles/frame)

**Rationale:**
1. User originally wanted ~116,000 cycles per frame
2. 7 MHz is a reasonable speed (faster than SNES 2.68 MHz, slower than 10 MHz)
3. Good balance between performance and power consumption
4. Still significantly faster than SNES/Genesis

### Implementation Plan for 7 MHz

**PPU Timing Adjustment:**
- Keep **220 scanlines** (200 visible + 20 VBlank)
- Adjust **dots per scanline**: 116,667 / 220 = **530.3 dots/scanline**
- Round to **530 dots per scanline**
- Actual cycles: 220 × 530 = **116,600 cycles per frame** (very close to target)

**Or Alternative:**
- Keep **360 dots per scanline** (current)
- Adjust **scanlines**: 116,667 / 360 = **324 scanlines**
- Visible: 200 scanlines
- VBlank: 124 scanlines (longer VBlank period)

**Recommendation: Adjust dots per scanline to 530**
- Keeps familiar 220 scanlines
- Maintains 200 visible scanlines
- Just extends HBlank period (40 → 170 dots)

## Code Changes Needed

### 1. Update PPU Timing Constants
```go
// internal/ppu/scanline.go
const (
    DotsPerScanline = 530  // Changed from 360
    VisibleDots     = 320  // Keep same
    HBlankDots      = 210  // Changed from 40 (530 - 320)
    
    VisibleScanlines = 200  // Keep same
    VBlankScanlines  = 20   // Keep same
    TotalScanlines   = 220  // Keep same
)
```

### 2. Update Emulator Frame Timing
```go
// internal/emulator/emulator.go
CyclesPerFrame: 116600, // 220 scanlines × 530 dots = 116,600 cycles
```

### 3. Update Clock Speed
```go
// internal/clock/scheduler.go
CPUSpeed: 7000000, // 7 MHz (changed from 10,000,000)
```

### 4. Update All Documentation
- Programming Manual
- System Manual
- README
- All references to 10 MHz → 7 MHz
- All references to 166,667 cycles → 116,600 cycles

## Where Did 237,594 Come From?

This number suggests:
- 237,594 × 60 = 14,255,640 cycles/second = **14.26 MHz**

This might be:
1. A bug in cycle counting
2. CPU running faster than intended
3. Multiple frames being counted
4. Cycle logger counting incorrectly

**Need to investigate:** Check `GetCPUCyclesPerFrame()` implementation and verify it's counting correctly.

---

## ✅ IMPLEMENTED: Genesis-Like Speed (~7.67 MHz)

**Decision Made:**
- **CPU Speed**: ~7.67 MHz (Genesis-like)
- **Cycles/Frame**: 127,820 (220 scanlines × 581 dots)
- **Unified Clock**: CPU and PPU synchronized to same clock
- **Target**: 60 FPS

**Changes Made:**
1. ✅ Updated PPU timing: 581 dots per scanline (was 360)
2. ✅ Updated CPU speed: 7,670,000 Hz (was 10,000,000 Hz)
3. ✅ Updated frame cycles: 127,820 cycles per frame (was 79,200)
4. ✅ Updated APU timing: ~174 cycles per sample (was ~227)
5. ✅ Updated clock scheduler comments

**Next Steps:**
- Test FPS performance (should be 60 FPS now)
- Update all documentation to reflect new timing
- Verify sprite movement and timing are correct

---

**End of Analysis**
