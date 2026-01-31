# VBlank Flag Analysis

## Current Implementation

### PPU VBlank Flag Logic

1. **Flag is SET**: At end of scanline 199 (before scanline 200 starts)
   - Location: `scanline.go:158-159` and `scanline.go:97`
   - Sets `VBlankFlag = true`

2. **Flag is CLEARED**: 
   - At start of new frame: `startFrame()` sets `VBlankFlag = false`
   - When read: `Read8(0x3E)` clears flag, but immediately re-sets if still in VBlank period

3. **Flag Read Behavior**:
   ```go
   inVBlank := p.currentScanline >= VisibleScanlines && p.currentScanline < TotalScanlines
   flag := p.VBlankFlag
   if inVBlank {
       flag = true  // Always return 1 during VBlank
   }
   p.VBlankFlag = false  // Clear after read
   if inVBlank {
       p.VBlankFlag = true  // Re-set if still in VBlank
   }
   ```

### Test ROM VBlank Wait Logic

At end of main loop:
```go
waitVBlankClearStart:
    MOV R4, #0x803E        // VBLANK_FLAG address
    MOV R5, [R4]           // Read flag (8-bit, zero-extended)
    MOV R7, #0            // R7 = 0
    CMP R5, R7            // Compare flag with 0
    BNE waitVBlankClearStart  // If flag != 0, keep waiting
```

**This waits for flag to be 0** (i.e., wait for VBlank to END).

## The Problem

### Issue 1: PPU May Not Be Running
- If PPU hasn't started (`StepPPU` not called), `currentScanline` might be 0
- `inVBlank` check: `currentScanline >= VisibleScanlines` (200)
- If `currentScanline = 0`, `inVBlank = false`
- Flag read returns 0 (not in VBlank)
- But flag might never be set if PPU isn't running!

### Issue 2: Flag Never Clears During VBlank
- During VBlank period (scanlines 200-219), flag is always 1
- ROM waits for flag to be 0
- If ROM is stuck in VBlank wait loop, and PPU is in VBlank period, flag will always be 1
- Wait will never complete!

### Issue 3: Timing Issue
- ROM might read flag at wrong time
- If read happens during VBlank, flag is 1
- ROM waits for 0, but flag stays 1 during entire VBlank period
- Only clears when new frame starts (scanline 0)

## Root Cause

The test ROM's VBlank wait logic is **waiting for VBlank to END**, but:
1. If PPU hasn't started, flag is 0 (but should wait for it to be 1 first!)
2. If PPU is in VBlank, flag is always 1, so wait never completes
3. The wait should be: wait for VBlank to START (flag = 1), then wait for it to END (flag = 0)

## Solution

The test ROM should:
1. **First wait for VBlank to START** (flag = 1) - this ensures we're in VBlank period
2. **Then wait for VBlank to END** (flag = 0) - this ensures we only update once per frame

OR, simpler:
- Just wait for VBlank to START (flag = 1), then continue
- Don't wait for it to clear - the next frame will handle it

## Recommended Fix

Change the end-of-loop wait to:
```go
// Wait for VBlank to START (not clear)
waitVBlankStart:
    MOV R4, #0x803E
    MOV R5, [R4]
    MOV R7, #0
    CMP R5, R7
    BEQ waitVBlankStart  // If flag == 0, keep waiting (wait for flag to be 1)
```

This ensures:
- We wait for VBlank period to start
- We do our updates during VBlank
- We don't wait for it to clear (which might never happen if timing is off)
