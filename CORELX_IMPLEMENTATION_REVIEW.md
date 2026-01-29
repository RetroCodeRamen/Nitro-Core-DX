# CoreLX Implementation Review

**Date**: January 27, 2026  
**Status**: Functional but Incomplete

## Executive Summary

The CoreLX compiler is **functional for basic use cases** but has **significant gaps** between what's documented and what's implemented. The compiler successfully compiles simple programs to valid ROMs, but many documented features are either missing or only partially implemented.

---

## ✅ Fully Implemented Features

### Lexer & Parser
- ✅ Indentation-based tokenization
- ✅ Comments (`--`)
- ✅ Numbers (decimal and hex)
- ✅ Strings
- ✅ All operators
- ✅ Variable declarations (`:=` and typed)
- ✅ Control flow (if, while, for)
- ✅ Function declarations
- ✅ Struct type declarations
- ✅ Asset declarations

### Semantic Analysis
- ✅ Type checking
- ✅ Symbol resolution
- ✅ Built-in type registration
- ✅ Built-in function registration
- ✅ `Start()` function requirement validation
- ✅ Namespace handling

### Code Generation
- ✅ Basic machine code generation
- ✅ Control flow (if, while, for)
- ✅ Expression evaluation
- ✅ Binary operators
- ✅ Unary operators
- ✅ Function calls (built-in functions)
- ✅ Struct initialization
- ✅ Asset constant handling

### Implemented Built-in Functions

**Frame Synchronization:**
- ✅ `wait_vblank()` - Fully implemented

**Graphics:**
- ✅ `ppu.enable_display()` - Fully implemented
- ✅ `gfx.load_tiles()` - Partially implemented (returns base, doesn't actually load tiles)

**Sprites:**
- ✅ `sprite.set_pos()` - Fully implemented
- ✅ `oam.write()` - Fully implemented
- ✅ `oam.flush()` - Implemented (no-op)
- ✅ `SPR_PAL()` - Fully implemented
- ✅ `SPR_PRI()` - Partially implemented (returns arg, doesn't shift bits)
- ✅ `SPR_ENABLE()` - Fully implemented
- ✅ `SPR_SIZE_16()` - Fully implemented

**Input:**
- ✅ `input.read()` - Fully implemented (not documented in manual)

---

## ⚠️ Partially Implemented Features

### Built-in Functions

**`frame_counter() -> u32`**
- **Status**: Registered but not implemented
- **Code**: Returns placeholder value (0)
- **Issue**: Manual documents this as returning `u32`, but implementation just returns 0
- **Location**: `codegen.go:1025-1029`

**`gfx.load_tiles(asset, base) -> u16`**
- **Status**: Partially implemented
- **Code**: Returns `base` parameter, doesn't actually load tiles from asset
- **Issue**: Asset data structure is parsed but not embedded in ROM
- **Location**: `codegen.go:1185-1190`

**`SPR_PRI(p: u8) -> u8`**
- **Status**: Partially implemented
- **Code**: Returns argument directly, doesn't shift to priority bits
- **Issue**: Should shift priority to upper bits of attr byte
- **Location**: `codegen.go:1150-1156`

---

## ❌ Missing Features (Documented but Not Implemented)

### APU Functions

**All APU functions are registered in semantic analyzer but NOT implemented in codegen:**

- ❌ `apu.enable()`
- ❌ `apu.set_channel_wave(ch, wave)`
- ❌ `apu.set_channel_freq(ch, freq)`
- ❌ `apu.set_channel_volume(ch, vol)`
- ❌ `apu.note_on(ch)`
- ❌ `apu.note_off(ch)`

**Impact**: Any program using APU functions will fail at code generation with "unknown builtin" error.

**Location**: 
- Registered: `semantic.go:87-88`
- Missing: `codegen.go` (no cases for `apu.*`)

### Sprite Helper Functions

**Missing sprite helper functions:**

- ❌ `SPR_HFLIP() -> u8` - Documented but not implemented
- ❌ `SPR_VFLIP() -> u8` - Documented but not implemented
- ❌ `SPR_SIZE_8() -> u8` - Documented but not implemented
- ❌ `SPR_BLEND(mode: u8) -> u8` - Documented but not implemented
- ❌ `SPR_ALPHA(a: u8) -> u8` - Documented but not implemented

**Location**: 
- Registered: `semantic.go:90-92`
- Missing: `codegen.go` (no cases for these functions)

### Asset System

**Asset embedding:**
- ❌ Asset data is parsed but not embedded into ROM
- ❌ `ASSET_<Name>` constants are generated but point to placeholder values
- ❌ `gfx.load_tiles()` doesn't actually load asset data

**Impact**: Assets can be declared but can't be used effectively.

**Location**: `codegen.go` - Asset handling is incomplete

---

## 🔧 Implementation Limitations

### Variable Storage

**Status**: Simplified implementation
- Variables use placeholder values
- No proper register allocation or memory tracking
- Variables aren't properly tracked across scopes

**Impact**: Complex programs may not work correctly.

**Location**: `codegen.go:20-41` (VariableInfo structure exists but not fully used)

### Struct Member Access

**Status**: Simplified implementation
- Struct member access is simplified
- Field offsets aren't calculated properly
- Member access generates error: "member access not fully implemented"

**Impact**: Struct member assignment (`hero.tile = base`) may not work correctly.

**Location**: `codegen.go:1230-1237`

### Function Calls

**Status**: User-defined functions not fully implemented
- Only built-in functions work
- No calling convention
- No stack management
- No parameter passing for user functions

**Impact**: Can't define custom functions beyond `Start()`.

**Location**: `codegen.go` - Function call handling is built-in only

### Register Allocation

**Status**: Simplified scheme
- Basic register allocation exists
- No proper register spilling
- No register lifetime tracking

**Impact**: Complex expressions may fail or use registers inefficiently.

**Location**: `codegen.go:43-47` (RegisterAllocator exists but is basic)

---

## 📋 Documentation Accuracy

### Manual Claims vs Implementation

| Feature | Manual Says | Actually Implemented | Status |
|---------|-------------|---------------------|--------|
| `wait_vblank()` | ✅ Available | ✅ Fully implemented | ✅ Accurate |
| `frame_counter() -> u32` | ✅ Available | ⚠️ Returns placeholder | ⚠️ Inaccurate |
| `ppu.enable_display()` | ✅ Available | ✅ Fully implemented | ✅ Accurate |
| `gfx.load_tiles()` | ✅ Available | ⚠️ Doesn't load tiles | ⚠️ Inaccurate |
| `sprite.set_pos()` | ✅ Available | ✅ Fully implemented | ✅ Accurate |
| `oam.write()` | ✅ Available | ✅ Fully implemented | ✅ Accurate |
| `oam.flush()` | ✅ Available | ✅ Implemented (no-op) | ✅ Accurate |
| `SPR_PAL()` | ✅ Available | ✅ Fully implemented | ✅ Accurate |
| `SPR_PRI()` | ✅ Available | ⚠️ Partially implemented | ⚠️ Inaccurate |
| `SPR_ENABLE()` | ✅ Available | ✅ Fully implemented | ✅ Accurate |
| `SPR_SIZE_16()` | ✅ Available | ✅ Fully implemented | ✅ Accurate |
| `SPR_HFLIP()` | ✅ Available | ❌ Not implemented | ❌ Inaccurate |
| `SPR_VFLIP()` | ✅ Available | ❌ Not implemented | ❌ Inaccurate |
| `SPR_SIZE_8()` | ✅ Available | ❌ Not implemented | ❌ Inaccurate |
| `SPR_BLEND()` | ✅ Available | ❌ Not implemented | ❌ Inaccurate |
| `SPR_ALPHA()` | ✅ Available | ❌ Not implemented | ❌ Inaccurate |
| `apu.enable()` | ✅ Available | ❌ Not implemented | ❌ Inaccurate |
| `apu.set_channel_wave()` | ✅ Available | ❌ Not implemented | ❌ Inaccurate |
| `apu.set_channel_freq()` | ✅ Available | ❌ Not implemented | ❌ Inaccurate |
| `apu.set_channel_volume()` | ✅ Available | ❌ Not implemented | ❌ Inaccurate |
| `apu.note_on()` | ✅ Available | ❌ Not implemented | ❌ Inaccurate |
| `apu.note_off()` | ✅ Available | ❌ Not implemented | ❌ Inaccurate |
| `input.read()` | ❌ Not documented | ✅ Fully implemented | ⚠️ Missing from docs |

### Manual Sections That Need Updates

1. **Audio (APU) Section** - Documents functions that don't exist
   - **Location**: `PROGRAMMING_MANUAL.md:427-443`
   - **Fix**: Mark as "Future Feature" or remove until implemented

2. **Sprite Helper Functions** - Documents functions that don't exist
   - **Location**: `PROGRAMMING_MANUAL.md:406-416`
   - **Fix**: Only document implemented functions (`SPR_PAL`, `SPR_PRI`, `SPR_ENABLE`, `SPR_SIZE_16`)

3. **Frame Counter** - Documents return value that doesn't work
   - **Location**: `PROGRAMMING_MANUAL.md:449-452`
   - **Fix**: Mark as "Placeholder - returns 0" or remove until implemented

4. **Asset System** - Implies assets work but they don't
   - **Location**: `PROGRAMMING_MANUAL.md:360-394`
   - **Fix**: Document that asset embedding is not yet implemented

5. **Input System** - Missing from manual
   - **Fix**: Add `input.read()` to manual since it's implemented

---

## 🎯 Recommendations

### Priority 1: Fix Documentation

1. **Mark unimplemented features clearly**
   - Add "⚠️ Not Yet Implemented" badges to APU functions
   - Add "⚠️ Partially Implemented" to `frame_counter()` and `gfx.load_tiles()`
   - Remove or mark missing sprite helpers

2. **Document what actually works**
   - Create a "Currently Implemented Features" section
   - List only functions that are fully functional
   - Separate "Future Features" section

3. **Add `input.read()` to manual**
   - Document the implemented input function
   - Show usage examples

### Priority 2: Complete Critical Features

1. **Implement APU functions** (if audio is important)
   - At minimum, mark as "not implemented" in codegen
   - Return proper error messages

2. **Fix `frame_counter()`**
   - Either implement properly or remove from manual
   - Current placeholder is misleading

3. **Complete asset embedding**
   - Embed asset data into ROM
   - Make `gfx.load_tiles()` actually work

### Priority 3: Improve Core Features

1. **Fix struct member access**
   - Properly calculate field offsets
   - Make `hero.tile = base` work correctly

2. **Improve variable storage**
   - Track variables across scopes
   - Proper register allocation

3. **Add user-defined function support**
   - Implement calling convention
   - Stack management
   - Parameter passing

---

## 📊 Implementation Completeness Score

**Overall**: ~60% Complete

**Breakdown:**
- **Lexer/Parser**: 95% ✅
- **Semantic Analysis**: 90% ✅
- **Code Generation**: 50% ⚠️
- **Built-in Functions**: 40% ⚠️
- **Documentation Accuracy**: 50% ⚠️

---

## ✅ What Works Well

1. **Basic compilation pipeline** - End-to-end compilation works
2. **Control flow** - If/while/for generate correct code
3. **Core sprite functions** - `sprite.set_pos()` and `oam.write()` work
4. **VBlank synchronization** - `wait_vblank()` is correct
5. **Type system** - Type checking and inference work

---

## 🚨 Critical Issues

1. **APU functions documented but not implemented** - Will cause compile errors
2. **Asset system incomplete** - Assets can't be used effectively
3. **Documentation mismatch** - Manual promises features that don't exist
4. **Struct member access broken** - `hero.tile = base` may not work
5. **User functions not supported** - Can only use `Start()` function

---

## 📝 Conclusion

The CoreLX compiler is **functional for simple programs** but has significant gaps. The biggest issue is **documentation that promises features that don't exist**. 

**Recommendation**: 
1. Update the manual to accurately reflect what's implemented
2. Mark unimplemented features clearly
3. Prioritize completing critical features (APU, assets, struct access)
4. Add proper error messages for unimplemented features

The compiler foundation is solid, but it needs completion work before it can be considered "production-ready" for the full feature set.
