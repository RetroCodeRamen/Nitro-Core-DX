# Actionable Fix List

**Date:** January 30, 2026  
**Purpose:** Prioritized list of fixes for emulator and specification

---

## Priority 1: Critical Specification Corrections

### Fix 1.1: Update CPU Speed in Specification
**Status:** ✅ Fixed in v2.1  
**Priority:** Critical  
**Effort:** Low (documentation only)

**Issue:** Specification v2.0 states 10 MHz, emulator uses 7.67 MHz.

**Action:**
- ✅ Updated specification v2.1 to reflect 7.67 MHz CPU speed
- ✅ Updated cycles per frame from 166,667 to 127,820

**Evidence:** `emulator.go:114`, `emulator.go:147`

---

### Fix 1.2: Correct APU Register Layout
**Status:** ✅ Fixed in v2.1  
**Priority:** Critical  
**Effort:** Low (documentation only)

**Issue:** Specification states 4 bytes per channel, emulator uses 8 bytes.

**Action:**
- ✅ Updated specification v2.1 to reflect 8 bytes per channel
- ✅ Corrected master volume address from 0x9040 to 0x9020
- ✅ Documented all channel registers (FREQ, VOLUME, CONTROL, DURATION, DURATION_MODE)

**Evidence:** `apu.go:130-131`, `apu.go:331`

---

### Fix 1.3: Correct Input Latch Register Addresses
**Status:** ✅ Fixed in v2.1  
**Priority:** Critical  
**Effort:** Low (documentation only)

**Issue:** Specification lists single latch at 0xA004, emulator uses separate latches.

**Action:**
- ✅ Updated specification v2.1 to reflect separate latch registers
- ✅ Controller 1 latch: 0xA001
- ✅ Controller 2 latch: 0xA003

**Evidence:** `input.go:41-52`

---

## Priority 2: High Priority Documentation

### Fix 2.1: Document PPU Timing Constants
**Status:** ✅ Fixed in v2.1  
**Priority:** High  
**Effort:** Low (documentation only)

**Issue:** Specification doesn't specify exact PPU timing.

**Action:**
- ✅ Documented exact timing constants in v2.1:
  - Dots per scanline: 581
  - Visible dots: 320
  - HBlank dots: 261
  - Visible scanlines: 200
  - VBlank scanlines: 20
  - Total scanlines: 220

**Evidence:** `scanline.go:6-27`

---

### Fix 2.2: Document OAM Write Protection Timing
**Status:** ✅ Fixed in v2.1  
**Priority:** High  
**Effort:** Low (documentation only)

**Issue:** Specification mentions OAM protection but doesn't specify exact timing.

**Action:**
- ✅ Documented exact scanline ranges in v2.1
- ✅ Writes blocked during scanlines 0-199 (visible rendering)
- ✅ Writes allowed during scanlines 200-219 (VBlank)

**Evidence:** `ppu.go:356-360, 373-377`

---

### Fix 2.3: Document VBlank Flag Re-set Behavior
**Status:** ✅ Fixed in v2.1  
**Priority:** High  
**Effort:** Low (documentation only)

**Issue:** VBlank flag re-sets if still in VBlank after read, not documented.

**Action:**
- ✅ Documented re-set behavior in v2.1
- ✅ Allows multiple reads during VBlank period
- ✅ Matches NES/SNES hardware behavior

**Evidence:** `ppu.go:204-219`

---

## Priority 3: Medium Priority Improvements

### Fix 3.1: Implement Cycle-Accurate DMA
**Status:** ⚠️ TODO  
**Priority:** Medium  
**Effort:** Medium (code changes)

**Issue:** DMA executes immediately, not cycle-accurate.

**Current Behavior:**
- DMA executes immediately when enabled (`ppu.go:616`)
- No cycle counting or timing

**Recommended Action:**
1. Add DMA cycle counter
2. Execute DMA over multiple cycles (similar to CPU instruction execution)
3. Update DMA_STATUS register to reflect active transfers
4. Allow CPU to continue during DMA (or block if needed)

**Evidence:** `ppu.go:639-683`, comment at line 615

**Impact:** ROMs relying on DMA timing might see different behavior.

---

### Fix 3.2: Improve Error Messages for Invalid ROM Entry Points
**Status:** ⚠️ TODO  
**Priority:** Medium  
**Effort:** Low (code changes)

**Issue:** Error messages for invalid entry points could be clearer.

**Current Behavior:**
- Validates entry point in `cartridge.go:99-105`
- Error messages are functional but could be more helpful

**Recommended Action:**
1. Add more context to error messages (expected vs actual values)
2. Suggest common fixes (e.g., "Entry point must be in bank 1-125, offset 0x8000+")
3. Include ROM file name in error message if available

**Evidence:** `cartridge.go:99-105`, `emulator.go:170-175`

---

### Fix 3.3: Document I/O Register 16-bit Access Behavior
**Status:** ✅ Fixed in v2.1  
**Priority:** Medium  
**Effort:** Low (documentation only)

**Issue:** 16-bit I/O access behavior not documented.

**Action:**
- ✅ Documented in v2.1 specification
- ✅ 16-bit reads: zero-extended 8-bit read
- ✅ 16-bit writes: only low byte written

**Evidence:** `instructions.go:34-49, 56-69`

---

## Priority 4: Low Priority Enhancements

### Fix 4.1: Consider Open Bus Implementation
**Status:** ⚠️ TODO (Optional)  
**Priority:** Low  
**Effort:** Medium (code changes)

**Issue:** Emulator returns 0 for unmapped addresses, real hardware might have open bus.

**Current Behavior:**
- Unmapped addresses return 0 (`bus.go:66, 94, 102`)

**Recommended Action:**
1. Research if open bus behavior is needed for ROM compatibility
2. If needed, implement open bus (return previous data bus value)
3. Document behavior in specification

**Evidence:** `bus.go:66, 94, 102`

**Impact:** Low - most ROMs don't rely on open bus behavior.

---

### Fix 4.2: Review PBR/PCBank Synchronization
**Status:** ⚠️ TODO (Code Review)  
**Priority:** Low  
**Effort:** Low (code review)

**Issue:** PBR and PCBank can get out of sync, code syncs them defensively.

**Current Behavior:**
- Code syncs PBR to PCBank if out of sync (`cpu.go:244-248`)

**Recommended Action:**
1. Review all PC update paths
2. Ensure PBR is updated whenever PCBank changes
3. Remove defensive sync if not needed (or keep if it's a safety measure)

**Evidence:** `cpu.go:244-248`, `cpu.go:512`

---

### Fix 4.3: Change MOV Mode 8 to Error Instead of NOP
**Status:** ⚠️ TODO (Optional)  
**Priority:** Low  
**Effort:** Low (code changes)

**Issue:** MOV mode 8 is reserved but treated as NOP.

**Current Behavior:**
- Mode 8 treated as NOP (`instructions.go:107-111`)

**Recommended Action:**
1. Change to return error for reserved modes
2. This would catch ROM bugs earlier
3. Document in specification

**Evidence:** `instructions.go:107-111`

**Impact:** Low - mode 8 is reserved, unlikely to be used.

---

## Priority 5: Future Enhancements

### Fix 5.1: Add Cycle-Accurate Timing Tests
**Status:** ⚠️ TODO (Future)  
**Priority:** Low  
**Effort:** High (test development)

**Issue:** No cycle-accurate timing tests to verify timing constants.

**Recommended Action:**
1. Create test ROMs that verify timing
2. Test frame timing, scanline timing, dot timing
3. Verify against specification

---

### Fix 5.2: Add ROM Compatibility Test Suite
**Status:** ⚠️ TODO (Future)  
**Priority:** Low  
**Effort:** High (test development)

**Issue:** No comprehensive ROM compatibility test suite.

**Recommended Action:**
1. Create test ROMs for edge cases
2. Test undefined behaviors
3. Verify against real hardware (when available)

---

## Summary

### Completed Fixes (✅)
- Fix 1.1: CPU Speed (v2.1)
- Fix 1.2: APU Register Layout (v2.1)
- Fix 1.3: Input Latch Addresses (v2.1)
- Fix 2.1: PPU Timing Constants (v2.1)
- Fix 2.2: OAM Write Protection (v2.1)
- Fix 2.3: VBlank Flag Re-set (v2.1)
- Fix 3.3: I/O 16-bit Access (v2.1)

### Pending Fixes (⚠️)
- Fix 3.1: Cycle-Accurate DMA (Medium priority, Medium effort)
- Fix 3.2: Error Messages (Medium priority, Low effort)
- Fix 4.1: Open Bus (Low priority, Medium effort)
- Fix 4.2: PBR/PCBank Sync (Low priority, Low effort)
- Fix 4.3: MOV Mode 8 Error (Low priority, Low effort)

### Future Enhancements
- Fix 5.1: Cycle-Accurate Timing Tests
- Fix 5.2: ROM Compatibility Test Suite

---

**End of Actionable Fix List**
