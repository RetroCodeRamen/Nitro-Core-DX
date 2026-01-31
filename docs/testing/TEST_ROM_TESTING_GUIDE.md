# Test ROM Testing Guide

## Quick Start

### 1. Run the Test ROM

```bash
./nitro-core-dx -rom test/roms/test.rom
```

### 2. What You Should See

- **White 8×8 block sprite** at the center of the screen (position 160, 100)
- **Blue background** initially
- **Emulator window** opens with the test ROM running

---

## Testing Each Feature

### ✅ Test 1: Sprite Rendering
**Expected**: White block appears on screen  
**Status**: Should work immediately when ROM loads

### ✅ Test 2: Input - Arrow Keys
**Controls**:
- **Arrow Keys** or **WASD** - Move the block
- **UP** / **W** - Move block up
- **DOWN** / **S** - Move block down
- **LEFT** / **A** - Move block left
- **RIGHT** / **D** - Move block right

**Expected**: Block moves smoothly in the direction pressed

### ✅ Test 3: Input - A Button (Block Color)
**Controls**:
- **Z** or **W** - A button (change block color)

**Expected**: 
- Block color cycles through different palettes (0-15)
- Each press changes the block's color
- Color should change immediately

### ✅ Test 4: Input - B Button (Background Color)
**Controls**:
- **X** - B button (change background color)

**Expected**: 
- Background color cycles: **Blue → Green → Red → Yellow → Blue** (repeats)
- Each press changes the background color
- Color should change immediately

### ✅ Test 5: Input - X Button (Sound Toggle)
**Controls**:
- **X** - X button (toggle sound on/off)

**Expected**: 
- First press: Sound turns OFF (if it was on)
- Second press: Sound turns ON (if it was off)
- Toggles each time you press X

### ✅ Test 6: Audio System
**Expected** (when sound is enabled):
- Plays **C major scale** automatically
- Each note plays for **1 second**, then **0.5 seconds silence**
- Notes cycle: **C, D, E, F, G, A, B, C** (repeats)
- You should hear audio if your system has audio enabled

**Note**: Audio may be quiet or inaudible depending on your system's audio settings.

### ✅ Test 7: Background Scroll
**Expected**: 
- Background scroll position follows the block's position
- As you move the block, the background scrolls with it
- This creates a parallax-like effect

---

## Troubleshooting

### Issue: Blank Screen
**Possible Causes**:
- ROM didn't load correctly
- PPU not initialized properly

**Check**:
```bash
# Verify ROM exists and has correct size
ls -lh test/roms/test.rom
# Should show: 726 bytes
```

### Issue: Sprite Doesn't Move
**Possible Causes**:
- Input not working
- OAM writes blocked

**Check**:
- Try pressing arrow keys
- Check if other buttons work (A, B, X)
- If nothing works, input system may have issues

### Issue: Background Color Doesn't Change
**Possible Causes**:
- CGRAM write not working
- B button not detected

**Check**:
- Press X button (should toggle sound) - if this works, input is working
- Press B button (X key) - background should change color
- If A button works but B doesn't, CGRAM write may have issues

### Issue: Audio Doesn't Play
**Possible Causes**:
- Audio disabled in system
- APU not initialized
- Sound toggled off

**Check**:
- Press X button to toggle sound on
- Check system audio settings
- Audio may be very quiet

---

## Advanced Testing

### Test with Logging

To see detailed logs of what's happening:

```bash
./nitro-core-dx -rom test/roms/test.rom -log
```

This will show:
- PPU operations (OAM, VRAM, CGRAM writes)
- Input reads
- APU updates
- Frame timing

**Note**: Logging significantly impacts performance (reduces FPS from 30 to ~7).

### Test Frame Timing

The ROM should run at **60 FPS** (or 30 FPS if frame-limited).

**Check**:
- Look at the emulator window title or status bar
- Should show stable frame rate
- No stuttering or freezing

---

## Expected Test Results

| Feature | Expected Result | Status |
|---------|----------------|--------|
| ROM Loading | Loads without errors | ✅ |
| Sprite Rendering | White block visible | ✅ |
| Arrow Keys | Block moves smoothly | ✅ |
| A Button | Block color changes | ✅ |
| B Button | Background color cycles | ✅ |
| X Button | Sound toggles | ✅ |
| Audio | Plays C major scale | ✅ |
| Background Scroll | Follows block position | ✅ |
| Frame Rate | Stable 60 FPS | ✅ |
| No Glitches | Smooth rendering | ✅ |

---

## Quick Test Checklist

Run through these quickly:

1. [ ] ROM loads (no errors)
2. [ ] White block appears on screen
3. [ ] Arrow keys move block
4. [ ] Z key changes block color
5. [ ] X key changes background color
6. [ ] X key toggles sound
7. [ ] Audio plays (if enabled)
8. [ ] Background scrolls with block
9. [ ] No visual glitches
10. [ ] Frame rate is stable

---

## Success Criteria

✅ **All features work correctly**  
✅ **No visual glitches**  
✅ **Stable frame rate**  
✅ **Input responds immediately**  
✅ **Audio plays when enabled**

If all of these pass, the fixes are working correctly!
