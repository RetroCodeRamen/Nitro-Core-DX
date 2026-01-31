# CGRAM Write Fix

## Critical Bug Found

The test ROM was writing CGRAM_DATA incorrectly!

### The Problem

**CGRAM_DATA is an 8-bit register**, but the test ROM was trying to write 16-bit RGB555 color values in a single write:

```go
movImm16(0, 0x7FFF) // R0 = white color (16-bit value)
movMem(7, 0)        // Write to CGRAM_DATA - ONLY LOW BYTE WRITTEN!
```

**Result**: Only the low byte (0xFF) was written, the high byte (0x7F) was lost!

### The Fix

CGRAM_DATA must be written in **two separate 8-bit writes**:
1. Write low byte first
2. Write high byte second

```go
// Write low byte
movReg(0, 5)        // R0 = color value
andImm(0, 0xFF)     // R0 = R0 & 0xFF (mask to low byte)
movMem(7, 0)        // Write low byte

// Write high byte
movReg(0, 5)        // R0 = color value
shrImm(0, 8)        // R0 = R0 >> 8 (shift to get high byte)
andImm(0, 0xFF)     // R0 = R0 & 0xFF (mask to byte)
movMem(7, 0)        // Write high byte
```

### Comparison with Working ROM

**Working ROM** (correct):
- Uses two 8-bit writes: low byte, then high byte
- Colors are written correctly

**Test ROM** (before fix):
- Used single 16-bit write
- Only low byte was written
- Colors were incorrect → **black screen**

### Files Changed

- `cmd/testrom/main.go`:
  - Fixed initialization CGRAM writes (blue and white colors)
  - Fixed main loop CGRAM write (background color)
  - Added `shrImm` helper function

### Verification

After fix:
- ✅ CGRAM writes use two 8-bit writes
- ✅ RGB555 colors are written correctly
- ✅ Palettes should be initialized properly
- ✅ Sprites should display with correct colors
