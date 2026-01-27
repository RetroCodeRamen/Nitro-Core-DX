# Nitro-Core-DX Master Planning & Review Document

**Last Updated:** January 27, 2026  
**Purpose:** Consolidated planning, reviews, and implementation status

---

## Table of Contents

1. [Code Review Findings](#code-review-findings)
2. [Architecture Reviews](#architecture-reviews)
3. [Clock-Driven Refactor Status](#clock-driven-refactor-status)
4. [Hardware Accuracy Review](#hardware-accuracy-review)
5. [CPU Comparison](#cpu-comparison)
6. [Week 1 & 2 Fixes](#week-1--2-fixes)
7. [Development Tools Plan](#development-tools-plan)
8. [Testing Framework](#testing-framework)
9. [SDK Features Plan](#sdk-features-plan)
10. [Current Issues & Solutions](#current-issues--solutions)

---

## Code Review Findings

### Critical Bugs (Fixed)

1. **CPU Reset() Corruption Bug** ✅ FIXED
   - Issue: Reset() set PCBank=0, causing crashes after ROM load
   - Fix: Reset() no longer resets PCBank/PCOffset/PBR
   - Location: `internal/cpu/cpu.go:74-92`

2. **Frame Execution Order Bug** ✅ FIXED
   - Issue: VBlank flag set AFTER CPU execution
   - Fix: PPU.RenderFrame() moved before CPU execution (now clock-driven)
   - Location: `internal/emulator/emulator.go`

3. **MOV Mode 3 I/O Write Bug** ✅ FIXED
   - Issue: Always wrote 8-bit to I/O, breaking 16-bit writes
   - Fix: Write 16-bit to non-I/O addresses, 8-bit to I/O
   - Location: `internal/cpu/instructions.go:38-53`

4. **Logger Goroutine Leak** ✅ FIXED
   - Issue: Logger goroutine never shut down
   - Fix: Added logger.Shutdown() calls in UI cleanup
   - Location: `internal/ui/ui.go`, `internal/ui/fyne_ui.go`

5. **Save State Implementation** ✅ IMPLEMENTED
   - Issue: Documented but not implemented
   - Fix: Complete save/load state system using encoding/gob
   - Location: `internal/emulator/savestate.go`

### High Priority Fixes (Fixed)

1. **Division by Zero** ✅ FIXED
   - Issue: Returned 0xFFFF silently
   - Fix: Added FlagD (division by zero flag)
   - Location: `internal/cpu/instructions.go:171-188`

2. **Stack Underflow** ✅ FIXED
   - Issue: Returned 0 without error
   - Fix: Pop16() now returns error on underflow
   - Location: `internal/cpu/instructions.go:501-517`

3. **APU Duration Loop Mode** ✅ FIXED
   - Issue: Didn't reload initial duration
   - Fix: Store InitialDuration, reload on loop
   - Location: `internal/apu/apu.go`

### Technical Debt

1. **Interrupt System** ⏳ NOT IMPLEMENTED
   - Status: Stubbed with TODO
   - Priority: Medium (not critical for basic ROMs)

2. **Matrix Mode** ⏳ NOT IMPLEMENTED
   - Status: Calls renderBackgroundLayer(0) instead
   - Priority: Medium (documented as not implemented)

3. **Cycle Counting Accuracy** ⚠️ NEEDS VERIFICATION
   - Status: Cycle costs exist but not verified
   - Priority: Medium (important for timing-sensitive ROMs)

---

## Architecture Reviews

### Architecture Review Summary

**Status:** ✅ PPU component works correctly  
**Issue:** Data flow from ROM → CPU → Bus → PPU needs verification

**Findings:**
- ✅ Direct PPU writes work correctly
- ✅ Sprite rendering works (256 white pixels found)
- ✅ CPU → Bus → PPU communication works
- ⚠️ ROM execution: VRAM/CGRAM writes work, OAM writes need verification
- ⚠️ Frame timing: Fixed (79,200 cycles per frame)

### Clock-Driven Architecture Review

**Status:** ✅ COMPLETE

**Completed:**
- ✅ Master clock scheduler (`internal/clock/scheduler.go`)
- ✅ Memory system split (Bus + Cartridge)
- ✅ PPU scanline/dot stepping (`internal/ppu/scanline.go`)
- ✅ APU fixed-point audio (`internal/apu/fixed_point.go`)
- ✅ Emulator integration (clock-driven is now default)
- ✅ Old frame-driven emulator removed

**Benefits:**
- Cycle-accurate timing
- Hardware-accurate PPU rendering
- FPGA-compatible design
- Better synchronization

---

## Hardware Accuracy Review

**Date:** January 27, 2026  
**Status:** ✅ All changes are hardware-accurate and FPGA-implementable

### VBlank Flag Timing ✅

- Set at scanline 200 (start of VBlank)
- Persists through scanlines 200-219
- Cleared when read (one-shot latch)
- Cleared at start of frame

**FPGA Implementation:** Simple D flip-flop with read-clear logic

### OAM Write Protection ✅

- Blocks writes during visible rendering (scanlines 0-199)
- Allows writes during VBlank (scanlines 200-219)
- Allows writes during first frame (initialization)

**FPGA Implementation:** Simple combinational logic based on scanline counter

### OAM_DATA Read Auto-Increment ✅

- Increments byte index on read
- Wraps to next sprite after 6 bytes
- Matches SNES/NES behavior

**FPGA Implementation:** Simple counter logic

### Dead Code Removed ✅

- Removed unused `oamInitFrame` field

---

## CPU Comparison

### Register Architecture

**Nitro-Core-DX:**
- 8 general-purpose registers (R0-R7), all 16-bit
- All registers equal - no special-purpose restrictions
- 24-bit banked addressing (16MB space)

**SNES:**
- Accumulator-based (A register is special)
- X, Y index registers
- 8/16-bit mode switching
- 24-bit banked addressing

**Genesis:**
- 8 data registers (D0-D7), 32-bit
- 7 address registers (A0-A6), 32-bit
- Separate data/address registers
- 32-bit flat addressing (24-bit externally)

### Processing Power

| System | Clock Speed | Cycles/Frame (60 FPS) |
|--------|-------------|----------------------|
| **Nitro-Core-DX** | **10 MHz** | **166,667** |
| Genesis | 7.67 MHz | ~127,833 |
| SNES | 2.68 MHz | ~44,667 |

**Result:** Nitro-Core-DX has the highest CPU performance of the three systems.

---

## Week 1 & 2 Fixes

### Week 1 Fixes ✅ ALL COMPLETE

1. ✅ CPU Reset() bug fixed
2. ✅ Frame execution order fixed
3. ✅ MOV mode 3 I/O write bug fixed
4. ✅ Logger goroutine leak fixed
5. ✅ Documentation updates

### Week 2 Fixes ✅ ALL COMPLETE

1. ✅ Save states implemented
2. ✅ Division by zero flag added
3. ✅ Stack underflow error handling
4. ✅ APU duration loop mode fixed

---

## Development Tools Plan

### Phase 1: Foundation ✅ COMPLETE

- ✅ Centralized logging system
- ✅ Log viewer panel
- ✅ Component-based logging toggles
- ✅ CPU log levels

### Phase 2: Emulation Control ✅ COMPLETE

- ✅ Start/Stop/Reset buttons
- ✅ Frame stepping
- ✅ Keyboard shortcuts

### Phase 3: Real-Time Debugging ⏳ PARTIAL

- ✅ Register viewer (basic)
- ✅ Memory viewer (basic)
- ⏳ Advanced features (breakpoints, watchpoints)

### Phase 4: Visual Tools ⏳ PLANNED

- ⏳ Tile map viewer
- ⏳ Sprite viewer
- ⏳ Palette viewer
- ⏳ Layer viewer

---

## Testing Framework

### Unit Tests ✅

- ✅ PPU component tests
- ✅ CPU instruction tests
- ✅ Emulator core tests
- ✅ Save/load state tests

### ROM-Based Tests ✅

- ✅ Simple sprite ROM
- ✅ Moving sprite ROM
- ✅ Bouncing ball ROM

### Test Coverage Goals

- [x] PPU sprite rendering
- [x] PPU OAM/VRAM/CGRAM writes
- [ ] PPU background rendering
- [ ] PPU scanline timing
- [ ] CPU cycle accuracy
- [ ] APU sound generation
- [ ] Clock scheduler coordination

---

## SDK Features Plan

### Core Tools ⏳ PLANNED

- ⏳ Enhanced ROM builder (macros, includes)
- ⏳ ROM validator
- ⏳ ROM analyzer
- ⏳ Standard library

### Asset Pipeline ⏳ PLANNED

- ⏳ Image to tile converter
- ⏳ Palette converter
- ⏳ Sprite sheet packer
- ⏳ Audio converter

### Asset Editors ⏳ PLANNED

- ⏳ Tile editor
- ⏳ Sprite editor
- ⏳ Map editor

---

## Current Issues & Solutions

### Issue: Sprite Not Moving (VBlank Wait Loop)

**Problem:** ROM is stuck waiting for VBlank flag. Sprite is visible but not moving.

**Root Cause Analysis:**
1. ROM reads VBlank flag at 0x803E
2. Flag is one-shot (cleared when read)
3. ROM reads flag during scanlines 0-199 → flag is false → loops
4. When scanline 200 hits → flag is set to true
5. ROM reads flag → gets 1 → clears flag → exits loop
6. ROM updates sprite position
7. ROM loops back to wait for VBlank
8. **PROBLEM:** Flag was already cleared, so ROM reads 0 → loops forever

**Solution:**
The flag should persist through the ENTIRE VBlank period (scanlines 200-219), not just be cleared on first read. However, the current implementation already does this - the flag is set at scanline 200 and persists until read or cleared at start of frame.

**Actual Issue:**
The flag is being cleared at `startFrame()` which happens at scanline 0, but the ROM might be reading it during scanlines 0-199 when it's false. The flag needs to be set BEFORE the ROM can read it during VBlank.

**Fix Required:**
Ensure the flag is set at scanline 200 and persists through scanlines 200-219. The flag should only be cleared:
1. When read (one-shot behavior)
2. At start of next frame (scanline 0)

**Implementation:**
The current implementation is correct. The issue might be that the ROM is reading the flag too quickly or there's a timing issue. Need to verify:
1. Flag is set correctly at scanline 200
2. Flag persists through scanlines 200-219
3. ROM reads flag at correct time
4. Flag is cleared correctly when read

**Next Steps:**
1. Add debug logging to trace VBlank flag state
2. Verify flag timing matches PPU scanline timing
3. Check if ROM is reading flag multiple times per cycle
4. Ensure flag is set before ROM can read it during VBlank

---

## Implementation Status Summary

### Completed ✅

- Clock-driven architecture
- Master clock scheduler
- PPU scanline/dot stepping
- APU fixed-point audio
- Memory system split (Bus + Cartridge)
- Save states
- Week 1 & 2 fixes
- Logging system
- Basic debugging tools

### In Progress ⏳

- Advanced debugging tools
- Visual debugging tools
- SDK build tools
- Test coverage expansion

### Planned 📋

- Interrupt system
- Matrix Mode implementation
- Cycle count verification
- Performance optimizations
- SDK asset pipeline
- IDE integration

---

## Key Design Principles

1. **Hardware Accuracy:** All timing and behavior matches real hardware
2. **FPGA Compatibility:** Architecture is FPGA-implementable
3. **Cycle Accuracy:** Components run at correct clock speeds
4. **Determinism:** Same inputs produce same outputs
5. **Testability:** Comprehensive test coverage
6. **Documentation:** Complete and accurate documentation

---

**End of Master Plan**
