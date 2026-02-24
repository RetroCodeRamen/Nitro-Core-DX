# Timing Synchronization Fix Summary

**Date:** January 27, 2026  
**Status:** ✅ Implemented

> **Historical Snapshot:** Summary of a timing-fix milestone. Use current hardware spec and test results for present timing behavior/status.

## Changes Made

### 1. CPU Speed: Genesis-Like (~7.67 MHz)
- **Changed from**: 10 MHz (166,667 cycles/frame)
- **Changed to**: ~7.67 MHz (127,820 cycles/frame)
- **Rationale**: User wants Genesis-like speed, unified clock synchronization

### 2. PPU Timing: Synchronized with CPU
- **Changed from**: 220 scanlines × 360 dots = 79,200 cycles/frame
- **Changed to**: 220 scanlines × 581 dots = 127,820 cycles/frame
- **Rationale**: Match CPU speed for unified clock

### 3. Clock Speed Updates
- **CPU Speed**: 7,670,000 Hz (was 10,000,000 Hz)
- **PPU Speed**: 7,670,000 Hz (same as CPU, unified clock)
- **APU Speed**: 44,100 Hz (unchanged)
- **APU Cycles Per Sample**: ~174 cycles (was ~227 cycles)

### 4. Performance Optimization
- **Batch Stepping**: Optimized StepCycles() to batch CPU/PPU steps
- **Cycle Logging**: Only step cycle-by-cycle when cycle logging enabled
- **Expected Result**: Faster frame execution, should reach 60 FPS

## Files Modified

1. `internal/ppu/scanline.go`
   - DotsPerScanline: 360 → 581
   - HBlankDots: 40 → 261
   - Updated comments

2. `internal/emulator/emulator.go`
   - CPU speed: 10,000,000 → 7,670,000 Hz
   - CyclesPerFrame: 79,200 → 127,820
   - APU cycles per sample: ~227 → ~174
   - Optimized batch stepping logic

3. `internal/clock/scheduler.go`
   - CPU speed comments updated
   - APU timing comments updated
   - Optimized StepCycles() for batch processing

4. `internal/ui/fyne_ui.go`
   - Updated comment about PPU frame cycles

5. `internal/apu/apu.go`
   - Updated comment about APU timing

## Expected Results

- **FPS**: Should reach 60 FPS (was 40 FPS)
- **CPU Cycles/Frame**: ~127,820 cycles (instruction cycles may vary)
- **Timing**: CPU and PPU synchronized via unified clock
- **Performance**: Faster due to batch stepping optimization

## Testing Needed

1. Verify FPS reaches 60 FPS
2. Verify sprite movement timing is correct
3. Verify audio timing is correct
4. Verify no timing-related bugs

---

**End of Summary**
