# Clock-Driven Architecture Cleanup

## Summary

Verified and cleaned up all remnants of the old frame-based system to ensure full clock-driven operation.

## Changes Made

### 1. Updated Comments
- ✅ `internal/emulator/emulator.go`: Updated comment from "166,667 cycles" to "79,200 cycles"
- ✅ Added clarification that PPU renders dot-by-dot, scanline-by-scanline

### 2. Deprecated Old Functions
- ✅ `internal/ppu/ppu.go`: Marked `RenderFrame()` as DEPRECATED
- ✅ Added comment explaining it's not used in clock-driven mode
- ✅ Clock-driven mode uses `StepPPU()` → `stepDot()` → `renderDot()`

### 3. Updated Tests
- ✅ `internal/emulator/frame_order_test.go`: Updated to test clock-driven execution
- ✅ Test now uses `RunFrame()` instead of calling `RenderFrame()` directly
- ✅ Verifies `FrameComplete` flag instead of old frame-based checks

### 4. Fixed FrameComplete Flag
- ✅ `internal/ppu/scanline.go`: Set `FrameComplete = false` at start of frame
- ✅ Ensures flag is properly managed throughout frame rendering

### 5. UI Updates
- ✅ `internal/ui/fyne_ui.go`: Removed unnecessary `FrameComplete` check
- ✅ `RunFrame()` guarantees frame completion, so check is redundant

## Current Architecture

### Clock-Driven Execution Flow

```
RunFrame() (79,200 cycles)
  ↓
MasterClock.Step() (called 79,200 times)
  ↓
  ├─→ CPU.StepCPU(1) - Every cycle
  ├─→ PPU.StepPPU(1) - Every cycle
  │     └─→ stepDot()
  │         ├─→ startFrame() (scanline 0, dot 0)
  │         ├─→ renderDot() (visible pixels)
  │         └─→ endFrame() (after 220 scanlines)
  └─→ APU.StepAPU(cycles) - Every ~227 cycles
```

### Component Timing

- **CPU**: Runs every cycle (10 MHz)
- **PPU**: Runs every cycle (dot-by-dot rendering)
- **APU**: Runs every ~227 cycles (44.1 kHz sample rate)

### Frame Completion

- **Frame Start**: `startFrame()` called at scanline 0, dot 0
  - Sets `VBlankFlag = true`
  - Increments `FrameCounter`
  - Clears output buffer
  - Sets `FrameComplete = false`

- **Frame End**: `endFrame()` called after scanline 219
  - Sets `FrameComplete = true`
  - Buffer is safe to read

## Remaining Legacy Code

### Functions Kept for Compatibility (Not Used)

1. **`PPU.RenderFrame()`**: DEPRECATED
   - Kept for compatibility
   - Not called in clock-driven mode
   - Marked with deprecation comment

2. **`APU.UpdateFrame()`**: May still exist
   - Check if still used or can be removed
   - Clock-driven mode uses `StepAPU()` instead

3. **`APU.GenerateSamples()`**: May still exist
   - Check if still used or can be removed
   - Clock-driven mode uses `GenerateSampleFixed()` instead

## Verification

✅ All tests pass
✅ Emulator builds successfully
✅ Clock-driven execution verified
✅ Frame timing matches PPU timing (79,200 cycles)

## Next Steps

1. Check if `APU.UpdateFrame()` and `APU.GenerateSamples()` are still needed
2. Remove or mark as deprecated if not used
3. Verify OAM write issue is not related to timing
4. Add logging to trace OAM writes from ROM execution
