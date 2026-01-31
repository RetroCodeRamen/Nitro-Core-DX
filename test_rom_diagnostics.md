# Test ROM Diagnostics Report

**Date**: January 30, 2026  
**ROM**: `test/roms/test.rom`  
**ROM Size**: 626 bytes (297 instructions)  
**Entry Point**: Bank 1, Offset 0x8000

---

## ✅ ROM Loading - PASS

- ROM file exists and loads successfully
- Header is valid
- Entry point is correct

---

## 🔍 Code Analysis

### 1. Initialization Sequence ✅

**Palette Setup:**
- ✅ Sets CGRAM_ADDR (0x8012) correctly
- ✅ Writes background color (palette 0, color 0 = blue, 0x001F)
- ✅ Writes block color (palette 1, color 1 = white, 0x7FFF)

**VRAM Setup:**
- ✅ Sets VRAM_ADDR (0x800E-0x800F) to 0x0000
- ✅ Writes 32 bytes of 0x11 (solid tile, color index 1)
- ✅ Uses loop to write tile data

**Background Setup:**
- ✅ Enables BG0 (0x8008 = 0x01)

**Status**: ✅ All initialization looks correct

---

### 2. Input System ⚠️ POTENTIAL ISSUE

**Latch Address:**
- Code uses: 0xA001 for latch
- Input system: Latch is at offset 0x01 (0xA001 absolute) ✅ CORRECT

**Reading Input:**
- Code reads 16-bit from 0xA000 using `MOV R0, [R7]` (mode 2)
- Input system: Read16() combines low and high bytes ✅ Should work

**Button Mapping:**
- UP = bit 0 ✅ Correct
- DOWN = bit 1 ✅ Correct
- LEFT = bit 2 ✅ Correct
- RIGHT = bit 3 ✅ Correct
- A = bit 4 ✅ Correct
- B = bit 5 ✅ Correct
- X = bit 6 ✅ Correct

**Issue Found:**
- ⚠️ **No VBlank wait before reading input** - Input is read immediately in main loop
- ⚠️ **Delay loop is too short** - Only 4096 iterations, may not be enough for 60 FPS timing

**Status**: ⚠️ Input code looks correct but timing may be off

---

### 3. Sprite Rendering ⚠️ CRITICAL ISSUE

**OAM Write Sequence:**
- ✅ Sets OAM_ADDR to sprite 0 (0x8014 = 0x00)
- ✅ Writes X low, X high, Y, Tile, Attr, Ctrl to OAM_DATA
- ✅ All 6 bytes written correctly

**CRITICAL ISSUE:**
- ⚠️ **OAM writes happen during main loop, not during VBlank**
- ⚠️ **PPU blocks OAM writes during visible rendering (scanlines 0-199)**
- ⚠️ **Test ROM uses delay loop instead of VBlank flag wait**

**Impact:**
- OAM writes may be silently ignored if they happen during visible frame
- Sprite may not update position correctly
- Visual glitches possible

**Fix Needed:**
- Replace delay loop with proper VBlank wait:
  ```assembly
  wait_vblank:
      MOV R7, #0x803E    ; VBlank flag
      MOV R6, [R7]       ; Read (clears flag)
      AND R6, #0x01
      BEQ wait_vblank    ; Wait for VBlank
  ```

**Status**: ⚠️ **CRITICAL - OAM writes may not work correctly**

---

### 4. Audio System ✅

**Channel Setup:**
- ✅ Uses channel 0 (0x9000-0x9003)
- ✅ Sets frequency (low and high bytes)
- ✅ Sets volume (0x80 = 128)
- ✅ Sets control (0x01 = enable, sine wave)

**Note Timing:**
- ✅ Timer increments each frame
- ✅ Changes note every 90 frames (1.5 seconds)
- ✅ Plays note for 60 frames (1 second), silence for 30 frames (0.5 seconds)

**Sound Toggle:**
- ✅ X button toggles sound flag (R6)
- ✅ Disables channel when sound is off

**Status**: ✅ Audio code looks correct

---

### 5. Background Scroll ✅

**Scroll Registers:**
- ✅ BG0_SCROLLX_L (0x8000) = X position
- ✅ BG0_SCROLLY_L (0x8002) = Y position
- ✅ Updates each frame

**Status**: ✅ Background scroll code looks correct

---

### 6. Color Cycling ✅

**Block Color (A button):**
- ✅ Increments palette (R2)
- ✅ Masks to 0-15 range
- ✅ Writes to OAM attributes (palette bits)

**Background Color (B button):**
- ✅ Increments palette (R3)
- ✅ Masks to 0-15 range
- ⚠️ **Issue**: Code doesn't update CGRAM - only increments R3 but never writes it back to CGRAM

**Status**: ⚠️ Block color works, background color may not update

---

## 🐛 Issues Found

### Issue 1: No VBlank Wait (CRITICAL)
- **Problem**: Uses delay loop instead of VBlank flag
- **Impact**: Timing may be wrong, OAM writes may be blocked
- **Fix**: Add VBlank wait before OAM writes

### Issue 2: OAM Write Timing (CRITICAL)
- **Problem**: OAM writes happen during main loop, may be during visible rendering
- **Impact**: Sprite updates may be ignored
- **Fix**: Ensure OAM writes happen during VBlank

### Issue 3: Background Color Not Updated
- **Problem**: R3 increments but never written to CGRAM
- **Impact**: Background color won't change when B button pressed
- **Fix**: Add CGRAM write to update background color

### Issue 4: Delay Loop Too Short
- **Problem**: Only 4096 iterations, may not be enough for 60 FPS
- **Impact**: Frame timing may be too fast
- **Fix**: Use VBlank wait instead

---

## ✅ What Works

1. ✅ ROM loads correctly
2. ✅ Initialization (palettes, VRAM, BG0) is correct
3. ✅ Input reading code structure is correct
4. ✅ Audio channel setup is correct
5. ✅ Sprite OAM write sequence is correct
6. ✅ Background scroll update is correct

---

## 🔧 Recommended Fixes

### Fix 1: Add VBlank Wait
Replace delay loop with proper VBlank wait before OAM writes.

### Fix 2: Update Background Color
Add CGRAM write to update background color when B button pressed.

### Fix 3: Ensure OAM Writes During VBlank
Move OAM writes to happen right after VBlank wait.

---

## 📊 Test Results Summary

| Feature | Code Correct | Timing Correct | Status |
|---------|--------------|----------------|--------|
| ROM Loading | ✅ | ✅ | ✅ PASS |
| Initialization | ✅ | ✅ | ✅ PASS |
| Input Reading | ✅ | ⚠️ | ⚠️ NEEDS FIX |
| Sprite Rendering | ✅ | ❌ | ❌ CRITICAL |
| Audio | ✅ | ✅ | ✅ PASS |
| Background Scroll | ✅ | ⚠️ | ⚠️ NEEDS FIX |
| Color Cycling (Block) | ✅ | ⚠️ | ⚠️ NEEDS FIX |
| Color Cycling (BG) | ❌ | N/A | ❌ BROKEN |

---

## Next Steps

1. **Fix VBlank wait** - Replace delay loop with VBlank flag check
2. **Fix OAM timing** - Ensure writes happen during VBlank
3. **Fix background color** - Add CGRAM write when B button pressed
4. **Test manually** - Run emulator and test each feature
5. **Verify visually** - Check sprite appears and moves correctly

---

**Overall Status**: ⚠️ **NEEDS FIXES** - Core functionality is there but timing and some features need correction.
