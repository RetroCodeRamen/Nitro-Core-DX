# Comprehensive Architecture Review

## Executive Summary

After comprehensive testing, the PPU component works correctly when written to directly. The issue appears to be in the **data flow from ROM → CPU → Bus → PPU** or **timing synchronization between UI and PPU rendering**.

## Component Status

### ✅ PPU Component (Working)
- **Direct PPU writes**: ✅ PASS
- **Sprite rendering**: ✅ PASS (256 white pixels found)
- **OAM writes**: ✅ PASS
- **VRAM writes**: ✅ PASS
- **CGRAM writes**: ✅ PASS
- **Frame timing**: ✅ PASS (79,200 cycles per frame)

### ⚠️ CPU → Bus → PPU Communication (Needs Testing)
- CPU writes to I/O addresses through Bus
- Bus routes to PPU handler
- **Status**: Tests created, need to verify

### ⚠️ Timing Issues (Critical)

#### Issue 1: Frame Rate Mismatch
- **PPU frame rate**: ~126 FPS (79,200 cycles per frame at 10 MHz)
- **UI update rate**: 60 FPS (1/60 second)
- **Problem**: UI reads buffer every 1/60 second, but PPU renders every 1/126 second
- **Impact**: UI might read buffer mid-frame or after buffer was cleared

#### Issue 2: Buffer Clearing Timing
- PPU clears output buffer at **start** of each frame (`startFrame()`)
- If UI reads buffer during frame rendering, it might see:
  - Partially rendered frame
  - Cleared buffer (if read right after `startFrame()`)
  - Complete frame (if read at end of frame)

#### Issue 3: Race Condition
- UI reads buffer in `renderEmulatorScreen()` (goroutine)
- PPU writes buffer during `StepPPU()` (clock-driven)
- **Current fix**: Buffer copy added, but timing issue remains

## Data Flow Analysis

### ROM → CPU → Bus → PPU Path

1. **ROM Execution**:
   ```
   ROM Code → CPU Fetch → CPU Execute → Memory Write
   ```

2. **Memory Write**:
   ```
   CPU.Mem.Write8(0, 0x8014, value)
   ↓
   Bus.Write8(0, 0x8014, value)
   ↓
   Bus.writeIO8(0x8014, value)
   ↓
   PPU.Write8(0x14, value)  // offset - 0x8000
   ```

3. **PPU Register Write**:
   ```
   PPU.Write8(0x14, value)
   ↓
   PPU.OAMAddr = value
   PPU.OAMByteIndex = 0
   ```

### Potential Issues

1. **CPU Not Executing ROM**: 
   - ROM might not be executing at all
   - CPU might be stuck in infinite loop
   - Entry point might be wrong

2. **CPU Not Writing to PPU**:
   - CPU might not be executing the ROM code that sets up sprites
   - MOV instructions might not be reaching PPU registers

3. **Timing Issue**:
   - PPU clears buffer before UI reads it
   - UI reads buffer during frame rendering
   - Multiple PPU frames per UI frame

## Recommendations

### Immediate Fixes

1. **Add Frame Synchronization**:
   - Use VBlank flag to synchronize UI reads
   - Only read buffer when frame is complete
   - Add frame completion flag to PPU

2. **Fix Frame Rate**:
   - Either: Slow down PPU to 60 FPS (increase cycles per frame)
   - Or: Speed up UI to match PPU frame rate
   - Or: Add frame buffering (double buffering)

3. **Add Debug Logging**:
   - Log when CPU writes to PPU registers
   - Log when PPU clears buffer
   - Log when UI reads buffer
   - Track frame completion

### Testing Strategy

1. **Direct PPU Test**: ✅ PASS
2. **CPU → Bus → PPU Test**: ⏳ IN PROGRESS
3. **ROM Execution Test**: ⏳ IN PROGRESS
4. **Frame Timing Test**: ⏳ IN PROGRESS
5. **UI Buffer Read Test**: ⏳ TODO

### Next Steps

1. Verify CPU is executing ROM
2. Verify CPU writes reach PPU
3. Fix frame timing synchronization
4. Add frame completion flag
5. Test with actual ROM execution
