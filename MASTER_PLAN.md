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
11. [Hardware Design & Physical Components](#hardware-design--physical-components)
12. [Stretch Goals](#stretch-goals)

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

### Phase 4: Visual Tools ⏳ PARTIAL

- ✅ Tile Viewer - Visual grid of tiles from VRAM with palette selection
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

### ✅ Issue: Sprite Not Moving (VBlank Wait Loop) - FIXED

**Problem:** ROM was stuck waiting for VBlank flag. Sprite was visible but not moving.

**Root Cause:**
The bug was in `MOV` mode 2 instruction. When reading from I/O registers (like the VBlank flag at 0x803E), mode 2 was reading 16 bits instead of 8 bits. Since I/O registers are 8-bit only, this caused:
- Reading 0x803E (VBlank flag = 0x01) and 0x803F (next register) together
- Getting 0x0100 instead of 0x0001
- Comparison `CMP R5, #0` failed because 0x0100 ≠ 0x0000
- ROM never exited the VBlank wait loop

**Solution Implemented:**
Modified `MOV` mode 2 to automatically detect I/O addresses (bank 0, offset >= 0x8000):
- **I/O registers**: Reads 8-bit and zero-extends to 16-bit
- **Normal memory**: Reads 16-bit as before

**Fix Location:** `internal/cpu/instructions.go:30-49`

**Status:** ✅ **FIXED** - Sprite movement now works correctly. ROM can properly read VBlank flag and synchronize with frame boundaries.

**Hardware Compatibility:** ✅ This fix is FPGA-implementable using standard address decoding logic (combinational comparators + data path multiplexer).

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
- **MOV Mode 2 I/O Register Fix** - Automatic 8-bit/16-bit detection for I/O vs memory
- **VBlank Flag Timing Fix** - Flag persists correctly through VBlank period
- **Cycle-by-Cycle Debug Logger** - Comprehensive logging with PPU/APU state
- **Register Viewer** - Real-time CPU register display with copy/save functionality
- **Memory Viewer** - Hex dump viewer with bank/offset navigation
- **Tile Viewer** - Visual tile grid viewer with palette selection and tile size options
- **Sprite Movement** - Working sprite animation with VBlank synchronization

### In Progress ⏳

- Advanced debugging tools (breakpoints, watchpoints)
- Sprite viewer panel
- Palette viewer panel
- Layer viewer panel
- SDK build tools
- Test coverage expansion
- TestEmulatorFrameExecution test fix

### Planned 📋

- Interrupt system
- Matrix Mode implementation
- Cycle count verification
- Performance optimizations
- SDK asset pipeline
- IDE integration

### Hardware Design & Physical Components 📋

- **3D Printable Console Casing** - Design hardware casing as 3D printable file
  - Create STL/3MF files for people to print their own console
  - Include mounting points for internal components
  - Design for standard 3D printer bed sizes
  - Documentation for assembly and component placement

- **3D Printable Controller** - Design controller as 3D printable file
  - Create STL/3MF files matching the controller mockup
  - Include button placement and PCB mounting points
  - Design for comfortable ergonomics
  - Documentation for assembly and wiring

### Stretch Goals 🎯

**After Dev Kit Completion:**

- **FPGA Implementation** - Port emulator to FPGA hardware
  - Translate Go architecture to Verilog/VHDL
  - Target FPGA platform (TBD - likely Lattice or Xilinx)
  - Hardware verification and testing
  - Performance optimization for FPGA constraints

- **Production Run** - Physical console manufacturing
  - **Prerequisites:**
    - ✅ Dev kit must be complete
    - ✅ At least 3 good games must be available
    - ✅ FPGA implementation must be verified
    - ✅ Funding secured (crowdfunding, investors, etc.)
  - Manufacturing considerations:
    - PCB design and assembly
    - Component sourcing
    - Quality assurance
    - Packaging and distribution

**Note:** Production run is a long-term goal and will only proceed if:
1. Dev kit is fully functional and polished
2. At least 3 quality games demonstrate the console's capabilities
3. FPGA implementation is stable and verified
4. Funding is available to support manufacturing

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
