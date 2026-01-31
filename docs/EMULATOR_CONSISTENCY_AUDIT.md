# Emulator Consistency & ROM Compatibility Audit

**Date:** January 30, 2026  
**Purpose:** Identify inconsistencies, bugs, and ROM compatibility issues in the emulator implementation

---

## Executive Summary

This audit examines the emulator source code for:
1. Internal inconsistencies within the emulator
2. Mismatches between specification v2.0 and emulator implementation
3. Potential ROM compatibility issues
4. Undefined or ambiguous behaviors

**Overall Assessment:** The emulator is largely consistent and well-implemented, but several specification mismatches and potential issues were identified.

---

## Critical Issues

### 1. CPU Speed Mismatch (Spec vs Implementation)

**Severity:** Critical  
**Confidence:** High

**Issue:** Specification v2.0 states CPU speed is 10 MHz, but emulator uses ~7.67 MHz.

**Evidence:**
- Spec v2.0: "CPU Speed: 10 MHz (166,667 cycles per frame at 60 FPS)"
- Emulator: `emulator.go:114` - `cpuSpeed := uint32(7670000) // ~7.67 MHz`

**Impact:**
- ROMs that rely on cycle-accurate timing will run at wrong speed
- Frame timing calculations are incorrect in spec
- FPGA implementation would be wrong if following spec

**Fix:** Update specification to match emulator (7.67 MHz, 127,820 cycles per frame).

---

### 2. APU Register Layout Mismatch

**Severity:** Major  
**Confidence:** High

**Issue:** Specification v2.0 states channels are 4 bytes each, but emulator uses 8 bytes.

**Evidence:**
- Spec v2.0: "Channel Registers (per channel, offset 0x10 per channel)" - implies 4 bytes
- Emulator: `apu.go:130-131` - `channel := int((offset / 8) & 0x3)` - 8 bytes per channel

**Impact:**
- ROMs using APU registers would access wrong addresses
- Channel 1 base would be 0x9004 instead of 0x9010
- Master volume address would be wrong

**Fix:** Specification corrected in v2.1 (8 bytes per channel, master volume at 0x9020).

---

### 3. Input Latch Register Address Mismatch

**Severity:** Major  
**Confidence:** High

**Issue:** Specification v2.0 lists INPUT_LATCH at 0xA004, but emulator uses separate latches per controller.

**Evidence:**
- Spec v2.0: "0xA004 | INPUT_LATCH | 8-bit | Latch control"
- Emulator: `input.go:41-52` - Controller 1 latch at 0xA001, Controller 2 at 0xA003

**Impact:**
- ROMs trying to latch controllers would write to wrong address
- Controller 2 latch would not work

**Fix:** Specification corrected in v2.1 (separate latch registers per controller).

---

## Major Issues

### 4. PPU Timing Constants Mismatch

**Severity:** Major  
**Confidence:** High

**Issue:** Specification v2.0 does not specify exact PPU timing, but emulator uses 581 dots per scanline.

**Evidence:**
- Emulator: `scanline.go:17` - `DotsPerScanline = 581`
- Emulator: `scanline.go:18-19` - `VisibleDots = 320`, `HBlankDots = 261`

**Impact:**
- FPGA implementation would need correct timing
- ROMs relying on scanline timing would be affected

**Fix:** Specification updated in v2.1 with exact timing constants.

---

### 5. OAM Write Protection Not Documented

**Severity:** Major  
**Confidence:** High

**Issue:** Specification v2.0 mentions OAM write protection but doesn't specify exact timing.

**Evidence:**
- Emulator: `ppu.go:356-360, 373-377` - Writes blocked during scanlines 0-199
- Spec v2.0: Only mentions "writes blocked during visible rendering"

**Impact:**
- ROMs might try to update sprites during visible rendering
- Behavior would be inconsistent if not properly documented

**Fix:** Specification updated in v2.1 with exact scanline ranges.

---

### 6. VBlank Flag Re-set Behavior

**Severity:** Medium  
**Confidence:** High

**Issue:** VBlank flag is re-set if still in VBlank period after being read.

**Evidence:**
- Emulator: `ppu.go:204-219` - Flag re-set if `inVBlank` is true after read
- This allows multiple reads during VBlank

**Impact:**
- ROMs that read flag multiple times would see it set each time
- This is actually correct behavior (matches NES/SNES), but should be documented

**Fix:** Specification updated in v2.1 to document re-set behavior.

---

## Minor Issues

### 7. Division by Zero Result

**Severity:** Minor  
**Confidence:** High

**Issue:** Division by zero returns 0xFFFF and sets D flag.

**Evidence:**
- Emulator: `instructions.go:190-197` - Returns 0xFFFF, sets FlagD

**Impact:**
- ROMs should check D flag after division
- Result value (0xFFFF) is reasonable but should be documented

**Fix:** Documented in v2.1 specification.

---

### 8. MOV Mode 8 Reserved Behavior

**Severity:** Minor  
**Confidence:** High

**Issue:** MOV mode 8 is reserved and treated as NOP.

**Evidence:**
- Emulator: `instructions.go:107-111` - Mode 8 treated as NOP

**Impact:**
- ROMs using mode 8 would execute NOP instead of error
- Should probably return error for reserved modes

**Fix:** Documented in v2.1. Consider changing to error in future.

---

### 9. I/O Register 16-bit Access Behavior

**Severity:** Minor  
**Confidence:** High

**Issue:** 16-bit reads from I/O are zero-extended 8-bit reads, writes only affect low byte.

**Evidence:**
- Emulator: `instructions.go:34-49, 56-69` - Special handling for I/O addresses

**Impact:**
- ROMs doing 16-bit I/O access would see unexpected behavior
- Should be documented

**Fix:** Documented in v2.1 specification.

---

## Potential ROM Compatibility Issues

### 10. ROM Entry Point Validation

**Severity:** Medium  
**Confidence:** High

**Issue:** Emulator validates entry point but doesn't enforce strict rules.

**Evidence:**
- Emulator: `cartridge.go:99-105` - Validates bank != 0 and offset >= 0x8000
- CPU: `cpu.go:254-260` - Safety checks for invalid PC addresses

**Impact:**
- ROMs with invalid entry points would fail to load
- This is correct behavior, but error messages could be clearer

**Recommendation:** Keep validation, improve error messages.

---

### 11. Stack Underflow Protection

**Severity:** Medium  
**Confidence:** High

**Issue:** Emulator checks for stack underflow in RET instruction.

**Evidence:**
- Emulator: `instructions.go:466-477` - RET checks for empty/corrupted stack

**Impact:**
- ROMs with stack bugs would get errors instead of undefined behavior
- This is helpful for debugging but may mask ROM bugs

**Recommendation:** Keep checks, but consider making them optional for production.

---

### 12. PC Alignment Enforcement

**Severity:** Low  
**Confidence:** High

**Issue:** Emulator enforces 16-bit alignment for PC (instructions are 16-bit).

**Evidence:**
- Emulator: `cpu.go:239-240, 277-278` - PC offset masked to ensure alignment

**Impact:**
- ROMs with misaligned jumps would have PC corrected
- This is correct behavior, but should be documented

**Fix:** Documented in v2.1.

---

## Undefined Behaviors

### 13. Open Bus Behavior

**Severity:** Low  
**Confidence:** Low

**Issue:** Emulator returns 0 for unmapped addresses, but real hardware might have open bus.

**Evidence:**
- Emulator: `bus.go:66, 94, 102` - Returns 0 for unmapped addresses
- No open bus implementation

**Impact:**
- ROMs relying on open bus behavior would see 0 instead of previous data
- This is unlikely to be an issue for most ROMs

**Recommendation:** Document as "returns 0" for now. Consider open bus implementation if needed.

---

### 14. DMA Cycle Accuracy

**Severity:** Medium  
**Confidence:** High

**Issue:** DMA executes immediately, not cycle-accurate.

**Evidence:**
- Emulator: `ppu.go:616` - `p.executeDMA()` called immediately
- Comment: "Execute DMA transfer immediately (for simplicity, can be made cycle-accurate later)"

**Impact:**
- ROMs relying on DMA timing might see different behavior
- Could cause issues with ROMs that read DMA status during transfer

**Recommendation:** Implement cycle-accurate DMA if ROMs require it.

---

### 15. APU Phase Reset on Frequency Change

**Severity:** Low  
**Confidence:** High

**Issue:** Phase resets only if frequency actually changes.

**Evidence:**
- Emulator: `apu.go:241` - `if newFreq != oldFreq && newFreq != 0`

**Impact:**
- Redundant frequency writes don't reset phase (prevents warbling)
- This is correct behavior, but should be documented

**Fix:** Documented in v2.1.

---

## Internal Consistency Issues

### 16. PBR and PCBank Synchronization

**Severity:** Low  
**Confidence:** High

**Issue:** PBR and PCBank can get out of sync, but code syncs them.

**Evidence:**
- Emulator: `cpu.go:244-248` - Syncs PBR to PCBank if out of sync

**Impact:**
- This is defensive programming, but indicates potential for bugs
- Should ensure they stay in sync at all times

**Recommendation:** Review all PC update paths to ensure PBR stays in sync.

---

### 17. CGRAM Write Latch Behavior

**Severity:** Low  
**Confidence:** High

**Issue:** CGRAM write requires two 8-bit writes, but 16-bit write is handled specially.

**Evidence:**
- Emulator: `memory/bus.go:110-115` - Special case for CGRAM_DATA 16-bit write
- PPU: `ppu.go:316-343` - Write latch mechanism

**Impact:**
- Both 8-bit and 16-bit writes work, but behavior is complex
- Should be well-documented

**Fix:** Documented in v2.1.

---

## Summary

### Issues by Severity

- **Critical:** 1 issue (CPU speed mismatch)
- **Major:** 4 issues (APU registers, input latch, PPU timing, OAM protection)
- **Medium:** 4 issues (VBlank flag, ROM validation, stack protection, DMA timing)
- **Minor:** 4 issues (Division by zero, MOV mode 8, I/O access, PC alignment)

### Issues by Category

- **Specification Mismatches:** 6 issues
- **Undefined Behaviors:** 3 issues
- **Internal Consistency:** 2 issues
- **ROM Compatibility:** 5 issues

### Recommended Actions

1. **Immediate:** Update specification to match emulator (CPU speed, APU registers, input latch)
2. **High Priority:** Document all timing constants and edge cases
3. **Medium Priority:** Consider cycle-accurate DMA implementation
4. **Low Priority:** Review PBR/PCBank synchronization, consider open bus implementation

---

**End of Emulator Consistency Audit**
