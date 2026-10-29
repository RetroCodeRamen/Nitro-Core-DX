# Input Debugging Guide

**Layer**: [IF] → [EMU] → [ROM]

This document helps debug input issues using the layer naming convention.

---

## Input Flow

```
[IF] Keyboard Press
  ↓
[IF] SDL Keyboard State (sdl.GetKeyboardState())
  ↓
[IF] updateInputFromKeys() → SetInputButtons(buttons)
  ↓
[EMU] Input.Controller1Buttons = buttons
  ↓
[ROM] Latch (write 1 to 0xA001)
  ↓
[EMU] Input.Write8(0x01) → Controller1Latched = Controller1Buttons
  ↓
[ROM] Read (read from 0xA000)
  ↓
[EMU] Input.Read8(0x00) → returns Controller1Latched
  ↓
[ROM] Process input
```

---

## Common Issues

### [IF] SDL Keyboard State Not Updating

**Symptoms**: No input detected, even when keys are pressed

**Debug Steps**:
1. Check if SDL is initialized: `sdl.Init(sdl.INIT_EVENTS)`
2. Check if `sdl.PumpEvents()` is being called regularly
3. Check if window has focus (SDL may require focus)
4. Verify `sdl.GetKeyboardState()` returns non-nil

**Fix**: Ensure `sdl.PumpEvents()` is called before reading keyboard state

---

### [IF] Keyboard Mapping Incorrect

**Symptoms**: Wrong buttons trigger wrong actions

**Debug Steps**:
1. Check button bit mappings in `updateInputFromKeys()`
2. Verify SDL scancode constants match expected keys
3. Test with known-good keyboard mapping

**Fix**: Update button mappings in `internal/ui/fyne_ui.go`

---

### [EMU] Input Not Being Set

**Symptoms**: Input set but ROM doesn't see it

**Debug Steps**:
1. Check `SetInputButtons()` is being called
2. Verify `Input.Controller1Buttons` is being set correctly
3. Check timing: input should be set BEFORE `RunFrame()`

**Fix**: Ensure input is set before frame execution

---

### [EMU] Latch Mechanism Not Working

**Symptoms**: ROM reads stale or incorrect input

**Debug Steps**:
1. Check `Input.Write8(0x01)` captures button state correctly
2. Verify `Controller1Latched` is set on latch write
3. Check edge detection logic (rising edge only)

**Fix**: Verify latch edge detection in `internal/input/input.go`

---

### [ROM] Input Reading Incorrect

**Symptoms**: ROM reads input but processes it wrong

**Debug Steps**:
1. Check ROM latch sequence (write 1, read, write 0)
2. Verify ROM reads correct register (0xA000 for low byte)
3. Check button bit masks match expected values

**Fix**: Update ROM code to match input register layout

---

## Diagnostic Test

Create a simple ROM that:
1. Latches input
2. Reads input
3. Displays button state as sprite colors or background color
4. This isolates [ROM] from [IF] and [EMU]

---

## Testing Checklist

- [ ] [IF] SDL keyboard state updates when keys pressed
- [ ] [IF] Button bits set correctly (0x01=UP, 0x02=DOWN, etc.)
- [ ] [EMU] `SetInputButtons()` called with correct value
- [ ] [EMU] `Input.Controller1Buttons` matches expected value
- [ ] [EMU] Latch write (0xA001 = 1) captures button state
- [ ] [EMU] Input read (0xA000) returns latched state
- [ ] [ROM] Latch sequence correct (write 1, read, write 0)
- [ ] [ROM] Button bit masks correct (0x01, 0x02, 0x04, 0x08)

---

**Last Updated**: 2025-02-11
