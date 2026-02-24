# Specification Audit - Discrepancies Found

**Date:** January 30, 2026  
**Auditor:** AI Assistant  
**Scope:** Complete section-by-section audit of COMPLETE_HARDWARE_SPECIFICATION_V2.1.md

---

## Summary

This document lists all discrepancies found between the specification and the actual implementation. For each discrepancy, a recommendation is provided on whether to update the spec or the emulator.

---

## 1. CPU Architecture

### ✅ Section Status: **MATCHES** (No discrepancies found)

All CPU details match the implementation:
- Register set (R0-R7, PC, SP, Flags) ✓
- Instruction format (16-bit, opcode layout) ✓
- Instruction set (all opcodes match) ✓
- Flags (Z, N, C, V, I, D) ✓
- Interrupt system (vectors, handling) ✓

---

## 2. Memory System

### ✅ Section Status: **MATCHES** (No discrepancies found)

All memory system details match:
- Bank layout (0x00, 0x01-0x7D, 0x7E-0x7F) ✓
- WRAM size and location ✓
- I/O register ranges ✓
- ROM mapping formula ✓
- Extended WRAM calculation ✓

---

## 3. PPU (Graphics) Specification

### ⚠️ **DISCREPANCY #1: DMA Register Address Conflict**

**Issue:**
- **Spec says:** DMA_LENGTH_L/H are at 0x8066/0x8067 (write-only)
- **Implementation Read8():** DMA_LENGTH_L/H are read from 0x8061/0x8062
- **Implementation Write8():** DMA_LENGTH_L/H are written to 0x8066/0x8067 (correct)
- **Implementation Write8():** DMA_SOURCE_BANK/OFFSET_L are written to 0x8061/0x8062

**Details:**
- Reading from 0x8061/0x8062 returns DMA_LENGTH (incorrect - should return DMA_SOURCE values)
- Writing to 0x8061/0x8062 sets DMA_SOURCE_BANK/OFFSET_L (correct)
- Writing to 0x8066/0x8067 sets DMA_LENGTH (correct)

**Evidence:**
- `ppu.go:245-248` - Read8() reads DMA_LENGTH from 0x61/0x62
- `ppu.go:633-646` - Write8() writes DMA_SOURCE to 0x61/0x62, DMA_LENGTH to 0x66/0x67

**Recommendation:** **UPDATE EMULATOR**
- **Reason:** The spec is correct - DMA_LENGTH should be at 0x8066/0x8067 for both read and write
- **Fix:** Change Read8() to read DMA_LENGTH from 0x66/0x67 instead of 0x61/0x62
- **Impact:** Low - DMA registers are rarely read, mostly write-only

---

### ⚠️ **DISCREPANCY #2: Register Read/Write Conflicts Not Documented**

**Issue:**
Several registers have different read vs write behavior that isn't explicitly documented:

1. **VBLANK_FLAG (0x803E):**
   - Read: Returns VBlank flag (0x01 if VBlank, 0x00 otherwise)
   - Write: Sets BG2_MATRIX_C_H (different register!)
   - **Spec:** Only documents read behavior

2. **FRAME_COUNTER (0x803F/0x8040):**
   - Read: Returns frame counter low/high bytes
   - Write: Sets BG2_MATRIX_D_L/H (different registers!)
   - **Spec:** Only documents read behavior

**Evidence:**
- `ppu.go:196-238` - Read8() handles VBLANK_FLAG and FRAME_COUNTER
- `ppu.go:526-531` - Write8() handles BG2_MATRIX_C_H, BG2_MATRIX_D_L/H at same offsets

**Recommendation:** **UPDATE SPEC**
- **Reason:** The spec should document that these registers are read-only, and that writes to these addresses affect different registers (BG2 matrix registers)
- **Fix:** Add note that 0x803E-0x8040 are read-only for system registers, writes affect BG2 matrix registers
- **Impact:** Low - These are typically read-only registers

---

## 4. APU (Audio) Specification

### ✅ Section Status: **MATCHES** (No discrepancies found)

All APU details match:
- Channel layout (8 bytes per channel) ✓
- Channel base addresses (0x9000, 0x9008, 0x9010, 0x9018) ✓
- Master volume address (0x9020) ✓
- Completion status (0x9021) ✓
- Waveforms and parameters ✓

---

## 5. Input System Specification

### ✅ Section Status: **MATCHES** (No discrepancies found)

All input details match:
- Controller addresses (0xA000-0xA003) ✓
- Latch addresses (0xA001, 0xA003) ✓
- Button mapping ✓

---

## 6. Timing and Synchronization

### ✅ Section Status: **MATCHES** (No discrepancies found)

All timing details match:
- CPU speed (~7.67 MHz) ✓
- PPU timing (581 dots per scanline) ✓
- Frame timing (127,820 cycles per frame) ✓
- VBlank flag timing ✓
- Frame counter ✓

---

## 7. ROM Format

### ✅ Section Status: **MATCHES** (No discrepancies found)

All ROM format details match:
- Header format (32 bytes) ✓
- Magic number ("RMCF") ✓
- Entry point fields ✓
- ROM size calculation ✓

---

## 8. Reset & Power-On State

### ✅ Section Status: **MATCHES** (No discrepancies found)

All reset state details match:
- CPU reset state ✓
- Memory reset state ✓
- PPU reset state ✓
- APU reset state ✓
- Input reset state ✓

---

## Summary of Discrepancies

| # | Section | Issue | Severity | Recommendation |
|---|---------|-------|----------|----------------|
| 1 | PPU | DMA register read addresses incorrect | Medium | Update Emulator |
| 2 | PPU | Register read/write conflicts not documented | Low | Update Spec |

---

## Action Items

### Priority 1: Fix DMA Register Read Addresses
- **File:** `internal/ppu/ppu.go`
- **Lines:** 245-248
- **Change:** Read DMA_LENGTH from 0x66/0x67 instead of 0x61/0x62
- **Reason:** Matches spec and write behavior
- **Status:** ⚠️ **INCOMPLETE AREA** - Related to recent cycle-accurate DMA work
- **Note:** This is a bug that needs fixing, but DMA implementation is still being refined

### Priority 2: Document Register Read/Write Conflicts
- **File:** `docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md`
- **Section:** PPU Registers (around line 567-570)
- **Change:** Add note that 0x803E-0x8040 are read-only system registers, writes affect BG2 matrix registers
- **Reason:** Clarifies hardware behavior

---

## Notes

- Most of the specification is accurate and matches the implementation
- The DMA register issue is a bug in the implementation (read addresses don't match write addresses)
- The register conflict documentation is a minor clarification needed in the spec
- No critical discrepancies found that would break ROM compatibility
