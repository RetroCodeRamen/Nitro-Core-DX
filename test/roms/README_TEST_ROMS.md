# CoreLX Test ROMs

This directory contains test ROMs that verify CoreLX language features work correctly.

## Test ROMs

### `corelx_comprehensive_test.corelx`
**Purpose**: Comprehensive test of all CoreLX language features

**Tests**:
- Variable declarations (inferred and typed)
- Control flow (if/else, while loops)
- Expressions (arithmetic, comparison, logical, bitwise)
- Structs (declaration, initialization, member access, assignment)
- Built-in functions (PPU, sprites, OAM, graphics)
- Address-of operator
- Function calls with arguments
- Main game loop with VBlank sync

**Usage**:
```bash
# Compile
./corelx test/roms/corelx_comprehensive_test.corelx test/roms/corelx_comprehensive_test.rom

# Test with harness
go build ./cmd/test_corelx_features
./test_corelx_features test/roms/corelx_comprehensive_test.rom

# Or run in emulator
./nitro-core-dx test/roms/corelx_comprehensive_test.rom
```

### `simple_test.corelx`
Basic variable and arithmetic test.

### `example.corelx`
Simple while loop with VBlank wait.

### `full_example.corelx`
Complete sprite example from CoreLX spec.

### `comprehensive_test.corelx`
All language features (older version).

## Test Harness

The `test_corelx_features` tool runs ROMs with full logging and verifies:
- ROM loading
- CPU execution
- PPU state changes
- OAM writes
- VBlank synchronization

## Verification

All test ROMs compile successfully, confirming:
- ✅ Lexer handles all tokens correctly
- ✅ Parser builds correct AST
- ✅ Semantic analyzer validates code
- ✅ Code generator produces valid machine code
- ✅ ROM builder creates valid ROM files

Runtime verification requires running in the emulator and checking:
- Visual output (sprites appear)
- Debug logs (CPU/PPU/Memory)
- Register state (variables have correct values)
