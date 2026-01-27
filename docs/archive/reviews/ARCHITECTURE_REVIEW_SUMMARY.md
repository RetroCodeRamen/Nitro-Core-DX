# Architecture Review Summary

## Critical Finding: OAM Writes Not Reaching PPU

### Test Results

**✅ PPU Component (Direct Writes)**: PASS
- Direct PPU writes work correctly
- Sprite rendering works (256 white pixels found)
- All PPU registers respond correctly

**✅ CPU → Bus → PPU Communication**: PASS  
- CPU writes to PPU registers through Bus work correctly
- Test shows CGRAM_ADDR, OAM_ADDR, OAM_DATA, VRAM_DATA all route correctly

**✅ ROM Execution**: PARTIAL
- ROM is executing (CPU cycles: 247,498-247,500 per frame)
- VRAM writes work: VRAM[0] = 0x11 ✅
- CGRAM writes work: CGRAM[0x22] = 0xFF, CGRAM[0x23] = 0x7F ✅
- **OAM writes FAIL: OAM[0-5] = [0, 0, 0, 0, 0x00, 0x00]** ❌

### Root Cause Analysis

The ROM code includes OAM setup:
```go
// ROM writes:
MOV R4, #0x8014  // OAM_ADDR
MOV R5, #0x00    // sprite 0
MOV [R4], R5     // Write to OAM_ADDR

MOV R4, #0x8015  // OAM_DATA
MOV R5, #100     // X low
MOV [R4], R5     // Write to OAM_DATA
// ... more OAM_DATA writes
```

But after execution, OAM remains all zeros. This suggests:

1. **ROM code path issue**: The OAM setup code might not be executing
2. **OAM reset**: OAM might be getting cleared somewhere
3. **Write timing**: OAM writes might be happening but getting overwritten

### Architecture Issues Found

1. **Frame Timing**: Fixed ✅
   - Changed from 166,667 to 79,200 cycles per frame
   - Matches PPU frame timing exactly

2. **Frame Completion Flag**: Added ✅
   - `FrameComplete` flag added to PPU
   - Set to `true` at end of frame rendering
   - UI can check before reading buffer

3. **Fyne Threading**: Fixed ✅
   - Wrapped UI updates in `fyne.Do()`
   - Prevents threading errors

4. **Buffer Race Condition**: Fixed ✅
   - Added buffer copy in UI render function
   - Prevents reading buffer while PPU is writing

### Remaining Issue: OAM Writes

**Problem**: ROM executes and writes VRAM/CGRAM, but OAM writes don't appear to reach PPU.

**Next Steps**:
1. Add logging to trace OAM writes from CPU → Bus → PPU
2. Check if ROM code path skips OAM setup
3. Verify OAM isn't being cleared after writes
4. Check if there's a timing issue with OAM writes

### Test Files Created

1. `internal/ppu/ppu_direct_test.go` - Direct PPU write tests ✅
2. `internal/emulator/architecture_test.go` - CPU→Bus→PPU communication tests ✅
3. `cmd/test_rom_execution/main.go` - ROM execution tracing ✅

### Recommendations

1. **Add OAM Write Logging**: Log every OAM write to trace the issue
2. **Verify ROM Code Path**: Check if OAM setup code is actually executing
3. **Check OAM Reset**: Verify OAM isn't being cleared unexpectedly
4. **Add OAM Read Test**: Test reading OAM to verify writes persist
