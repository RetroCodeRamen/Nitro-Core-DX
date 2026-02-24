# CoreLX Test ROMs

This directory contains test ROMs that verify CoreLX language features work correctly.

## Assembly / Diagnostic ROM Generators

Several Go-based ROM generator utilities also live in this directory. They are excluded from default builds/tests using the `testrom_tools` build tag (to avoid multiple `main()` conflicts).

### `build_input_visual_diagnostic.go`
**Purpose**: Manual emulator input diagnostics with obvious on-screen feedback

**Features**:
- Arrow keys / WASD move a white 8x8 sprite
- Movement uses acceleration/friction for a more game-like feel
- `Z` (A button): background color toggle (gray <-> cyan)
- `X` (B button): sprite color toggle (white <-> green)
- `C` (Y button): reset sprite to start position
- `Q` (L): trigger a short one-shot SFX (CH3 noise burst)
- `Q` (L): trigger a short one-shot "bewp" jump SFX (pitch sweep)
- `E` (R): toggle a layered music loop while moving (melody + bass + percussion, plus FM MMIO/timer traffic)
- Music loop also exercises the FM MMIO/timer path in the background (diagnostic traffic)
- `Enter` (START): enable legacy CH0 tone
- `Backspace` (high-byte Z / mapped to Backspace): stop all test tones

**Usage**:
```bash
# Build ROM
go run -tags testrom_tools ./test/roms/build_input_visual_diagnostic.go ./test/roms/input_visual_diagnostic.rom

# Run in emulator (no SDL_ttf build)
go run -tags no_sdl_ttf ./cmd/emulator -rom ./test/roms/input_visual_diagnostic.rom
```

### `build_apu_fm_showcase.go`
**Purpose**: Manual audio diagnostic/showcase for legacy APU + new FM extension MMIO/timer path

**Features**:
- `Z` (A button): plays a legacy APU scale (channel 0)
- `X` (B button): exercises FM extension MMIO/timer status path and plays an audible legacy proxy scale
- `C` (Y button): plays a short simplified Bach excerpt (multi-channel legacy APU, duration + completion status usage)
- Visual background color changes indicate which demo is active
- FM extension register/timer writes are mirrored during `B`/`C` for future FM audio validation

**Note**:
- This ROM remains a legacy-APU-focused showcase with FM MMIO/timer diagnostics; the `B` demo behavior is intentionally a legacy proxy path.

**Usage**:
```bash
# Build ROM
go run -tags testrom_tools ./test/roms/build_apu_fm_showcase.go ./test/roms/apu_fm_showcase.rom

# Run in emulator (no SDL_ttf build)
go run -tags no_sdl_ttf ./cmd/emulator -rom ./test/roms/apu_fm_showcase.rom
```

### `build_fm_opmlite_showcase.go`
**Purpose**: Manual audio diagnostic/showcase for the new **FM OPM-lite audible subset** (new FM extension register layout)

**Features**:
- `Z` (A button): FM lead scale (single FM voice)
- `X` (B button): FM layered chord + arpeggio demo (3 FM voices)
- `C` (Y button): FM multi-voice "Hall of the Mountain King"-style phrase (melody + bass + pad)
- Uses `FM_ADDR`/`FM_DATA` (`0x9100/0x9101`) to program the new OPM-lite register subset
- Uses FM Timer A status pacing (`FM_STATUS`) during phrase steps, so the MMIO/timer path is exercised while audio plays
- Legacy APU channels are disabled; audible output is from the FM extension path

**Usage**:
```bash
# Build ROM
go run -tags testrom_tools ./test/roms/build_fm_opmlite_showcase.go ./test/roms/fm_opmlite_showcase.rom

# Run in emulator (no SDL_ttf build)
go run -tags no_sdl_ttf ./cmd/emulator -rom ./test/roms/fm_opmlite_showcase.rom
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
