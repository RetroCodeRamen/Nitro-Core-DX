# Input System Debugging

## Issue Reported
"Box moves from right to left and freezes on the left edge"

## Possible Causes

### 1. Input Latch Timing Issue
The ROM latches input every frame, but if there's a timing issue where:
- ROM latches before UI updates input
- Or latched state persists incorrectly

**Fix Applied**: Edge-triggered latch that captures on rising edge (0→1 transition)

### 2. Wrapping Logic Issue
The test ROM's wrapping logic only checks for X == 65535 (underflow case), but doesn't properly handle X == 0 case.

**Current wrapping code**:
- Checks if X == 320 (wrap to 0)
- Checks if X == 65535 (wrap to 319)

**Problem**: When X reaches 0 and you try to move left, it becomes 65535, which should be caught. But if the wrap check happens before movement, or if there's a logic error, it might not work.

### 3. Input Always Reading LEFT
If LEFT (bit 2 = 0x04) is always being read as pressed, the box will continuously move left.

**To Debug**:
1. Check if input is actually 0 when no keys are pressed
2. Check if latch is capturing correctly
3. Check if read is returning latched state correctly

## Testing Steps

1. **Run with logging**:
   ```bash
   ./nitro-core-dx -rom test/roms/input_test.rom -log
   ```

2. **Check input state**:
   - Open register viewer in emulator
   - Check R2 after input read (should be 0 when no keys pressed)
   - Check R0 (X position) - should not continuously decrease

3. **Verify latch behavior**:
   - The ROM should latch every frame (write 1 to 0xA001)
   - Then read from 0xA000
   - If input is 0, read should return 0

## Quick Fix to Try

If the issue is that input is stuck, try:
1. Rebuild the test ROM: `./testrom_input test/roms/input_test.rom`
2. Rebuild the emulator: `go build -tags "no_sdl_ttf" -o nitro-core-dx ./cmd/emulator`
3. Run again: `./nitro-core-dx -rom test/roms/input_test.rom`

If the issue persists, the problem might be in:
- The test ROM's wrapping logic
- Or a deeper issue with input timing

## Next Steps

If the issue continues:
1. Add debug logging to input system to see what values are being latched/read
2. Check if the ROM is reading input correctly
3. Verify the wrapping logic in the test ROM is correct
