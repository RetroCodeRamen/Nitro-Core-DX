# Test ROM Feature-by-Feature Build Plan

## Strategy

Build the test ROM incrementally, testing each feature before adding the next.

## Feature List

### ✅ Phase 1: Minimal Display (test_minimal.rom)
**Goal**: Display a white sprite on screen

1. **Feature 1**: Enable BG0
2. **Feature 2**: Initialize palette (white color)
3. **Feature 3**: Load tile data to VRAM
4. **Feature 4**: Wait for VBlank
5. **Feature 5**: Write sprite to OAM
6. **Feature 6**: Main loop (VBlank wait)

**Test**: Run ROM, verify white block appears at (160, 100)

---

### 🔄 Phase 2: Add Input (test_input.rom)
**Goal**: Add input reading

7. **Feature 7**: Read controller input
8. **Feature 8**: Handle arrow keys (move sprite)

**Test**: Verify sprite moves with arrow keys

---

### 🔄 Phase 3: Add Color Changes (test_colors.rom)
**Goal**: Add color/palette changes

9. **Feature 9**: A button changes sprite color
10. **Feature 10**: B button changes background color

**Test**: Verify colors change when buttons pressed

---

### 🔄 Phase 4: Add Audio (test_audio.rom)
**Goal**: Add audio playback

11. **Feature 11**: Audio system initialization
12. **Feature 12**: Play C major scale
13. **Feature 13**: X button toggles audio

**Test**: Verify audio plays and can be toggled

---

### 🔄 Phase 5: Complete (test.rom)
**Goal**: All features working together

14. **Feature 14**: Background scroll
15. **Feature 15**: All features integrated

**Test**: Full functionality test

---

## Testing Procedure

For each phase:
1. Build the ROM
2. Run: `./nitro-core-dx -rom test/roms/test_<phase>.rom`
3. Verify the feature works
4. If it works, proceed to next phase
5. If it doesn't, debug and fix before proceeding

## Current Status

- ✅ **Phase 1**: Minimal ROM created (`test_minimal.rom`)
- ⏳ **Next**: Test Phase 1, then add Phase 2
