# Black Screen Fix

## Root Cause Identified

The black screen was caused by **OAM writes being ignored**!

### The Problem

1. **OAM writes are only allowed during VBlank** (scanlines 200-219)
2. The ROM removed the VBlank wait to fix the infinite loop
3. Without VBlank wait, ROM writes to OAM during visible rendering (scanlines 0-199)
4. PPU **ignores OAM writes during visible rendering** (hardware-accurate behavior)
5. Result: Sprite data never gets written → **black screen**

### The Fix

Added back VBlank wait, but this time:
- **Wait for VBlank to START** (flag = 1), not wait for it to clear
- This ensures OAM writes happen during VBlank period
- Prevents infinite loop (we wait for flag to be 1, which happens every frame)

### Code Change

```go
// Wait for VBlank to START (flag = 1)
waitVBlankStart:
    MOV R4, #0x803E        // VBLANK_FLAG address
    MOV R5, [R4]           // Read flag
    MOV R7, #0            // R7 = 0
    CMP R5, R7            // Compare flag with 0
    BEQ waitVBlankStart   // If flag == 0, keep waiting (wait for flag to be 1)
```

This ensures:
- We wait for VBlank period to start
- OAM writes happen during VBlank (not ignored)
- Sprite data gets written correctly
- Screen displays properly

## Verification

- ✅ OAM write sequence is complete (6 bytes per sprite)
- ✅ VBlank wait added (waits for flag = 1)
- ✅ OAM writes will happen during VBlank period
- ✅ Sprite should now be visible
