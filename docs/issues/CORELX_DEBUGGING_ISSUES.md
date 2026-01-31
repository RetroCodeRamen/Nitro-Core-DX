# CoreLX Compiler Debugging Issues

**Created:** January 30, 2026  
**Status:** In Progress  
**Goal:** Debug why CoreLX-compiled ROMs show blank screen while Go-compiled ROMs work correctly

## Context

We're recreating the working `moving_sprite_colored.rom` in CoreLX to identify compiler bugs. The original ROM works perfectly, but the CoreLX version shows a blank screen, indicating compiler issues rather than just code problems.

## Compiler Bugs Fixed ✅

### 1. VRAM Address Calculation for tiles16 ✅ FIXED
- **Issue**: `generateInlineTileLoad()` always calculated VRAM address as `base * 32` (for 8x8 tiles)
- **Problem**: For `tiles16` assets, it should be `base * 128` (16x16 tiles are 128 bytes at 4bpp)
- **Impact**: 16x16 tiles were being loaded to wrong VRAM addresses
- **Fix**: Added check for `asset.Type == "tiles16"` to use `base << 7` instead of `base << 5`
- **File**: `internal/corelx/codegen.go` - `generateInlineTileLoad()`

### 2. Binary OR Operation ✅ FIXED
- **Issue**: Binary OR (`|`) operation used wrong register
- **Problem**: Left result saved to R1, but OR used destReg (R0) which may have been overwritten
- **Impact**: Expressions like `SPR_PAL(1) | SPR_PRI(0)` and `SPR_ENABLE() | SPR_SIZE_16()` produced incorrect results
- **Fix**: Changed to `OR R1, R2` then move result to destReg
- **File**: `internal/corelx/codegen.go` - `BinaryExpr` case `TOKEN_PIPE`

### 3. Binary AND Operation ✅ FIXED
- **Issue**: Same register issue as OR
- **Fix**: Changed to `AND R1, R2` then move result to destReg
- **File**: `internal/corelx/codegen.go` - `BinaryExpr` case `TOKEN_AMPERSAND`

### 4. Binary ADD/SUB Operations ✅ FIXED
- **Issue**: Left result saved to R1, but operations used destReg (R0) directly
- **Fix**: Restore left result from R1 to destReg before performing operation
- **File**: `internal/corelx/codegen.go` - `BinaryExpr` cases `TOKEN_PLUS` and `TOKEN_MINUS`

## Current Issues (Still Debugging) 🔍

### Issue #1: Blank Screen - Sprite Not Rendering
- **Symptom**: CoreLX-compiled ROM shows blank screen, even with hardcoded values
- **Test ROM**: `test/roms/moving_sprite_colored_corelx.rom`
- **Simplified Test**: `test/roms/moving_sprite_colored_simple.corelx` (hardcoded values, no variables)
- **Status**: Still blank screen after fixing binary operations

**Possible Causes:**
1. Variable storage/loading not working correctly across loop iterations
2. `wait_vblank()` not working correctly (infinite loop?)
3. Sprite not being written to OAM correctly
4. Tile data not loading to VRAM correctly (despite VRAM address fix)
5. Display not being enabled properly
6. Palette initialization not working
7. OAM writes happening at wrong time (during visible rendering instead of VBlank)

**Next Steps:**
- Add detailed logging to trace execution flow
- Compare generated machine code with working Go version
- Test each function individually (palette, tiles, sprite writing)
- Check if `wait_vblank()` is causing infinite loop
- Verify OAM writes are happening during VBlank

### Issue #2: Variable Persistence
- **Symptom**: Variables like `sprite_x`, `sprite_y` may not persist across loop iterations
- **Test**: Variables are stored in registers or stack, but may be overwritten
- **Status**: Needs investigation

### Issue #3: Negative Number Handling
- **Symptom**: `sprite_x = -16` (negative value) may not be handled correctly
- **Test**: Need to verify two's complement handling in assignments
- **Status**: Needs investigation

## Test ROMs

### Working Reference
- `test/roms/moving_sprite_colored.rom` - Original Go-compiled version (works perfectly)
- Source: `test/roms/build_moving_sprite_colored.go`

### CoreLX Versions (Debugging)
- `test/roms/moving_sprite_colored_corelx.rom` - Full CoreLX version with variables
- `test/roms/moving_sprite_colored_simple.rom` - Simplified with hardcoded values
- Source: `test/roms/moving_sprite_colored.corelx` and `test/roms/moving_sprite_colored_simple.corelx`

## Tile Pattern Correction

**Original Pattern**: 4 horizontal color lines
- Rows 0-3: Red (0x11)
- Rows 4-7: Red (0x11)  
- Rows 8-11: Blue (0x33)
- Rows 12-15: Yellow (0x44)

**Initial Mistake**: Created 4 quadrants instead of 4 horizontal lines
**Status**: ✅ Corrected

## Files Modified

- `internal/corelx/codegen.go`:
  - Fixed `generateInlineTileLoad()` for tiles16 VRAM address
  - Fixed binary OR operation register usage
  - Fixed binary AND operation register usage
  - Fixed binary ADD/SUB operations to restore left result

## Success Criteria

- [ ] CoreLX-compiled ROM shows sprite (not blank screen)
- [ ] Sprite displays with correct colors (Red, Red, Blue, Yellow lines)
- [ ] Sprite moves horizontally across screen
- [ ] Sprite wraps correctly when reaching edge
- [ ] Behavior matches original Go-compiled ROM

## Notes

- The compiler has variable storage infrastructure but may have bugs in how variables are loaded/stored
- Binary operations were fundamentally broken (using wrong registers)
- VRAM address calculation was wrong for 16x16 tiles
- Need systematic debugging approach: test each component individually
