# Test ROM Feature-by-Feature Testing Plan

## Phase 1: Minimal Display ✅

**ROM**: `test/roms/test_minimal.rom`  
**Features**: 
- BG0 enabled
- White palette color
- Tile data loaded
- Sprite displayed

**Test Command**:
```bash
./nitro-core-dx -rom test/roms/test_minimal.rom
```

**Expected Result**:
- ✅ White 8×8 block appears at center of screen (160, 100)
- ✅ Block stays in place (no movement)
- ✅ No input response (expected)

**If it works**: Proceed to Phase 2  
**If it doesn't**: Debug initialization sequence

---

## Phase 2: Add Input (Next)

**ROM**: `test/roms/test_input.rom`  
**Features**: 
- All Phase 1 features
- Controller input reading
- Arrow key movement

**Test Command**:
```bash
./nitro-core-dx -rom test/roms/test_input.rom
```

**Expected Result**:
- ✅ White block appears
- ✅ Arrow keys move the block
- ✅ Block position updates smoothly

**If it works**: Proceed to Phase 3  
**If it doesn't**: Debug input reading and OAM updates

---

## Phase 3: Add Colors (Next)

**ROM**: `test/roms/test_colors.rom`  
**Features**: 
- All Phase 2 features
- A button changes sprite color
- B button changes background color

**Test Command**:
```bash
./nitro-core-dx -rom test/roms/test_colors.rom
```

**Expected Result**:
- ✅ All Phase 2 features work
- ✅ A button cycles sprite colors
- ✅ B button cycles background colors

**If it works**: Proceed to Phase 4  
**If it doesn't**: Debug CGRAM writes and button handling

---

## Phase 4: Add Audio (Next)

**ROM**: `test/roms/test_audio.rom`  
**Features**: 
- All Phase 3 features
- Audio playback (C major scale)
- X button toggles audio

**Test Command**:
```bash
./nitro-core-dx -rom test/roms/test_audio.rom
```

**Expected Result**:
- ✅ All Phase 3 features work
- ✅ Audio plays automatically
- ✅ X button toggles audio on/off

**If it works**: Proceed to Phase 5  
**If it doesn't**: Debug APU initialization and control

---

## Phase 5: Complete (Final)

**ROM**: `test/roms/test.rom`  
**Features**: 
- All Phase 4 features
- Background scroll
- All features integrated

**Test Command**:
```bash
./nitro-core-dx -rom test/roms/test.rom
```

**Expected Result**:
- ✅ All features work together
- ✅ Smooth operation
- ✅ No glitches or crashes

---

## Current Status

- ✅ **Phase 1**: Minimal ROM created - **READY TO TEST**
- ⏳ **Phase 2-5**: To be implemented after Phase 1 is verified

## Next Steps

1. **Test Phase 1**: Run `./nitro-core-dx -rom test/roms/test_minimal.rom`
2. **Verify**: White block appears on screen
3. **If successful**: Build Phase 2 (add input)
4. **If failed**: Debug Phase 1 issues before proceeding
