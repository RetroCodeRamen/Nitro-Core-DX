# CoreLX Compiler Complete Implementation Plan

**Date**: January 27, 2026  
**Status**: Planning Phase

## Overview

This document outlines a comprehensive plan to complete the CoreLX compiler implementation. The compiler is currently ~60% complete and functional for basic programs, but needs completion work for production readiness.

---

## Phase 1: Complete Built-in Functions (Priority: High)

### 1.1 APU Functions Implementation
**Status**: ❌ Not Implemented  
**Priority**: High (documented but missing)  
**Estimated Effort**: 4-6 hours

**Functions to Implement:**
- `apu.enable()` - Enable APU master volume
- `apu.set_channel_wave(ch: u8, wave: u8)` - Set waveform type
- `apu.set_channel_freq(ch: u8, freq: u16)` - Set frequency (16-bit)
- `apu.set_channel_volume(ch: u8, vol: u8)` - Set volume
- `apu.note_on(ch: u8)` - Start note playback
- `apu.note_off(ch: u8)` - Stop note playback

**Implementation Details:**
- APU registers: 0x9000-0x9FFF
- Channel base addresses: CH0=0x9000, CH1=0x9008, CH2=0x9010, CH3=0x9018
- Each channel: 8 bytes (FREQ_LOW, FREQ_HIGH, VOLUME, CONTROL, DURATION_LOW, DURATION_HIGH, DURATION_MODE, Reserved)
- Master volume: 0x9020

**Tasks:**
1. Implement `apu.enable()` - Write 0xFF to 0x9020
2. Implement `apu.set_channel_wave()` - Write to CONTROL register (bits 1-2)
3. Implement `apu.set_channel_freq()` - Write low byte, then high byte (triggers phase reset)
4. Implement `apu.set_channel_volume()` - Write to VOLUME register
5. Implement `apu.note_on()` - Set CONTROL bit 0 to 1
6. Implement `apu.note_off()` - Set CONTROL bit 0 to 0
7. Add error checking for channel number (0-3)
8. Test with example program

**Files to Modify:**
- `internal/corelx/codegen.go` - Add cases to `generateBuiltinCall()`

---

### 1.2 Complete Sprite Helper Functions
**Status**: ⚠️ Partially Implemented  
**Priority**: Medium  
**Estimated Effort**: 2-3 hours

**Functions to Implement:**
- `SPR_HFLIP() -> u8` - Return 0x10 (horizontal flip bit)
- `SPR_VFLIP() -> u8` - Return 0x20 (vertical flip bit)
- `SPR_SIZE_8() -> u8` - Return 0x00 (8x8 size, bit 1 = 0)
- `SPR_BLEND(mode: u8) -> u8` - Return mode shifted to bits [3:2]
- `SPR_ALPHA(a: u8) -> u8` - Return alpha value shifted to bits [7:4]

**Fix Existing:**
- `SPR_PRI(p: u8) -> u8` - Shift priority to bits [7:6] of attr byte

**Tasks:**
1. Implement missing sprite helpers
2. Fix `SPR_PRI()` to shift bits correctly
3. Test sprite attribute combinations

**Files to Modify:**
- `internal/corelx/codegen.go` - Add/fix cases in `generateBuiltinCall()`

---

### 1.3 Fix Frame Counter
**Status**: ⚠️ Returns Placeholder  
**Priority**: Medium  
**Estimated Effort**: 1 hour

**Implementation:**
- Read FRAME_COUNTER_LOW (0x803F) and FRAME_COUNTER_HIGH (0x8040)
- Combine into 16-bit value
- Return in destReg

**Tasks:**
1. Read low byte from 0x803F
2. Read high byte from 0x8040
3. Combine: (high << 8) | low
4. Return in destReg

**Files to Modify:**
- `internal/corelx/codegen.go` - Fix `frame_counter` case

---

## Phase 2: Asset System (Priority: High)

### 2.1 Asset Embedding
**Status**: ❌ Not Implemented  
**Priority**: High  
**Estimated Effort**: 6-8 hours

**Current State:**
- Assets are parsed and stored in AST
- `ASSET_<Name>` constants are generated but point to placeholder values
- `gfx.load_tiles()` doesn't actually load asset data

**Implementation Plan:**

**Step 1: ROM Layout Planning**
- Reserve space in ROM for asset data
- Calculate asset offsets
- Store asset metadata (offset, size, type)

**Step 2: Asset Embedding**
- Embed asset data into ROM after code section
- Update `ASSET_<Name>` constants to point to actual offsets
- Handle both hex and b64 encodings

**Step 3: Implement `gfx.load_tiles()`**
- Read asset data from ROM
- Write to VRAM starting at base address
- Return base address (or next available address)

**Tasks:**
1. Add asset storage to ROMBuilder
2. Calculate asset offsets
3. Embed asset data into ROM
4. Update asset constant generation
5. Implement VRAM DMA or byte-by-byte copy in `gfx.load_tiles()`
6. Test with example asset

**Files to Modify:**
- `internal/corelx/codegen.go` - Asset handling
- `internal/rom/builder.go` - Asset embedding
- `internal/corelx/ast.go` - Asset metadata

---

## Phase 3: Struct System (Priority: Medium)

### 3.1 Fix Struct Member Access
**Status**: ❌ Broken  
**Priority**: Medium  
**Estimated Effort**: 4-6 hours

**Current State:**
- Struct types are parsed
- Struct initialization works (`Sprite()`)
- Member access (`hero.tile = base`) generates error

**Implementation Plan:**

**Step 1: Struct Layout Calculation**
- Calculate field offsets for each struct type
- Store struct layout in symbol table
- Handle alignment (if needed)

**Step 2: Member Access Code Generation**
- Generate code to access struct fields
- Handle both read and write access
- Support nested member access (`obj.field.subfield`)

**Step 3: Assignment to Members**
- Generate code for `obj.field = value`
- Handle pointer dereferencing (`&hero.tile`)
- Preserve other struct fields

**Tasks:**
1. Add struct layout calculation to semantic analyzer
2. Store field offsets in symbol table
3. Implement `generateMember()` properly
4. Handle member assignment
5. Test with sprite struct

**Files to Modify:**
- `internal/corelx/semantic.go` - Struct layout calculation
- `internal/corelx/codegen.go` - Member access generation
- `internal/corelx/ast.go` - Struct metadata

---

## Phase 4: Variable Storage (Priority: Medium)

### 4.1 Proper Variable Tracking
**Status**: ⚠️ Simplified  
**Priority**: Medium  
**Estimated Effort**: 6-8 hours

**Current State:**
- Variables use placeholder values
- No proper register allocation
- Variables aren't tracked across scopes

**Implementation Plan:**

**Step 1: Variable Location Tracking**
- Track where variables are stored (register, stack, memory)
- Handle variable scoping
- Support variable shadowing

**Step 2: Register Allocation**
- Implement proper register allocation algorithm
- Handle register spilling to stack
- Track register lifetimes

**Step 3: Stack Management**
- Allocate stack space for spilled variables
- Generate PUSH/POP for function calls
- Handle stack frame management

**Tasks:**
1. Implement variable location tracking
2. Add register allocation algorithm (simplified graph coloring or linear scan)
3. Implement register spilling
4. Generate stack management code
5. Test with complex expressions

**Files to Modify:**
- `internal/corelx/codegen.go` - Variable tracking and register allocation
- `internal/corelx/semantic.go` - Scope tracking

---

## Phase 5: User-Defined Functions (Priority: Low)

### 5.1 Function Call Support
**Status**: ❌ Not Implemented  
**Priority**: Low  
**Estimated Effort**: 8-10 hours

**Current State:**
- Function declarations are parsed
- Only `Start()` function works
- No calling convention

**Implementation Plan:**

**Step 1: Calling Convention**
- Define parameter passing (registers vs stack)
- Define return value handling
- Define register preservation rules

**Step 2: Function Call Code Generation**
- Generate CALL instruction
- Set up parameters
- Handle return value

**Step 3: Function Prologue/Epilogue**
- Generate function entry code
- Save registers
- Allocate stack frame
- Generate RET instruction

**Tasks:**
1. Design calling convention
2. Implement function call code generation
3. Implement function prologue/epilogue
4. Handle return values
5. Test with recursive functions

**Files to Modify:**
- `internal/corelx/codegen.go` - Function call generation
- `internal/corelx/semantic.go` - Function signature checking

---

## Phase 6: Expression System Improvements (Priority: Low)

### 6.1 Complex Expressions
**Status**: ⚠️ Basic Support  
**Priority**: Low  
**Estimated Effort**: 4-6 hours

**Improvements:**
- Better register allocation for complex expressions
- Optimize common patterns
- Handle nested function calls
- Support array indexing

**Tasks:**
1. Improve expression evaluation
2. Optimize register usage
3. Handle nested calls
4. Add array support

---

## Implementation Order

**Recommended Sequence:**

1. **Phase 1.1: APU Functions** (Start here - well-defined, self-contained)
2. **Phase 1.2: Sprite Helpers** (Quick wins)
3. **Phase 1.3: Frame Counter** (Simple fix)
4. **Phase 2.1: Asset Embedding** (High value, enables graphics)
5. **Phase 3.1: Struct Member Access** (Needed for sprite code)
6. **Phase 4.1: Variable Storage** (Foundation for functions)
7. **Phase 5.1: User Functions** (Complex, can wait)
8. **Phase 6.1: Expression Improvements** (Polish)

---

## Testing Strategy

For each phase:

1. **Unit Tests**
   - Test individual function code generation
   - Verify register usage
   - Check instruction correctness

2. **Integration Tests**
   - Compile example programs
   - Run in emulator
   - Verify behavior matches expected

3. **Example Programs**
   - Create test programs for each feature
   - Document expected behavior
   - Add to test suite

---

## Success Criteria

**Phase 1 Complete:**
- ✅ All documented built-in functions work
- ✅ No "unknown builtin" errors
- ✅ Example programs compile and run

**Phase 2 Complete:**
- ✅ Assets embedded in ROM
- ✅ `gfx.load_tiles()` loads actual data
- ✅ Sprite programs can use assets

**Phase 3 Complete:**
- ✅ Struct member access works
- ✅ `hero.tile = base` compiles correctly
- ✅ Nested member access works

**Phase 4 Complete:**
- ✅ Variables tracked properly
- ✅ Register allocation efficient
- ✅ Complex expressions work

**Phase 5 Complete:**
- ✅ User-defined functions work
- ✅ Recursive functions work
- ✅ Function calls preserve state

---

## Notes

- Each phase should be tested independently
- Keep backward compatibility with existing code
- Update documentation as features are added
- Add example programs for each feature
- Consider performance implications of register allocation

---

**Next Steps**: Start with Phase 1.1 (APU Functions) - well-defined, self-contained, and high priority.
