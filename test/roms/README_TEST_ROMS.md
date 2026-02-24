# CoreLX Test ROMs

This directory contains test ROMs that verify CoreLX language features work correctly.

## Assembly / Diagnostic ROM Generators

Several Go-based ROM generator utilities also live in this directory. They are excluded from default builds/tests using the `testrom_tools` build tag (to avoid multiple `main()` conflicts).

### `build_input_visual_diagnostic.go`
**Purpose**: Manual emulator input diagnostics with obvious on-screen feedback

**Features**:
- Arrow keys / WASD move a white 8x8 sprite
- Movement is rate-limited (1 pixel every 2 frames) for easier visual testing at lower FPS
- `Z` (A button): background color toggle (gray <-> cyan)
- `X` (B button): sprite color toggle (white <-> green)
- `C` (Y button): reset sprite to start position
- `Q` / `E` (L/R): select low/high test note frequency
- `Enter` (START): start tone on APU channel 0
- `Backspace` (Z button): stop tone on APU channel 0

**Usage**:
```bash
# Build ROM
go run -tags testrom_tools ./test/roms/build_input_visual_diagnostic.go ./test/roms/input_visual_diagnostic.rom

# Run in emulator (no SDL_ttf build)
go run -tags no_sdl_ttf ./cmd/emulator -rom ./test/roms/input_visual_diagnostic.rom
```

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
