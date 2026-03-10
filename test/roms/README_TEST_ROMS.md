# CoreLX Test ROMs

This directory contains test ROMs that verify CoreLX language features work correctly.

## Current vs Archived ROM Binaries

To keep this folder usable during active development, older prebuilt `.rom` binaries have been moved to:

- `test/roms/archive/legacy_roms/`

Current/manual diagnostics and frequently referenced ROMs stay in `test/roms/` (for quick loading in Nitro-Core-DX and docs examples).
Sources (`.corelx`) and ROM generator programs (`build_*.go`) remain in `test/roms/` so archived ROMs can be rebuilt when needed.

## Assembly / Diagnostic ROM Generators

Several Go-based ROM generator utilities also live in this directory. They are excluded from default builds/tests using the `testrom_tools` build tag (to avoid multiple `main()` conflicts).

**Important:** Each generator is a **single-file utility**. Run one at a time with `go run -tags testrom_tools ./test/roms/<filename>.go <args>`. Do **not** run `go test -tags testrom_tools ./test/roms` or build the whole package with the tag—that would try to compile multiple `main()` in one package and fail.

### `build_input_visual_diagnostic.go`
**Purpose**: Manual emulator input diagnostics with obvious on-screen feedback

**Features**:
- Arrow keys / WASD move a white 8x8 sprite
- Movement uses acceleration/friction for a more game-like feel
- `Z` (A button): background color toggle (gray <-> cyan)
- `X` (B button): sprite color toggle (white <-> green)
- `C` (Y button): reset sprite to start position
- `Q` (L): trigger a short one-shot "bewp" jump SFX (pitch sweep)
- `E` (R): toggle a layered music loop while moving (melody + bass + percussion, plus FM MMIO/timer traffic)
- Music loop also exercises the FM MMIO/timer path in the background (diagnostic traffic)
- `Enter` (START): enable legacy CH0 tone
- `Backspace` (high-byte Z / mapped to Backspace): stop all test tones

**Usage**:
```bash
# Build ROM
go run -tags testrom_tools ./test/roms/build_input_visual_diagnostic.go ./roms/input_visual_diagnostic.rom

# Run in emulator (no SDL_ttf build)
go run -tags no_sdl_ttf ./cmd/emulator -rom ./roms/input_visual_diagnostic.rom
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
go run -tags testrom_tools ./test/roms/build_apu_fm_showcase.go ./roms/apu_fm_showcase.rom

# Run in emulator (no SDL_ttf build)
go run -tags no_sdl_ttf ./cmd/emulator -rom ./roms/apu_fm_showcase.rom
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
go run -tags testrom_tools ./test/roms/build_fm_opmlite_showcase.go ./roms/fm_opmlite_showcase.rom

# Run in emulator (no SDL_ttf build)
go run -tags no_sdl_ttf ./cmd/emulator -rom ./roms/fm_opmlite_showcase.rom
```

### `build_ym2608_demo_song.go`
**Purpose**: Build a YM2608 MMIO replay ROM from `Resources/Demo.vgz` for quick A/B diagnostics.

**Notes**:
- This generator emits a banked replay stream (YM writes + waits), so full-song playback is supported.
- IRQ-safe trampoline is installed at bank `01:8000`; normal song entry starts at `01:8002`.
- `-max-frames` remains available for short diagnostic clips.

**Usage**:
```bash
# Build ROM from Demo.vgz
go run -tags testrom_tools ./test/roms/build_ym2608_demo_song.go \
  -in Resources/Demo.vgz \
  -out roms/ym2608_demo_song.rom \
  -frames-per-bank 70

# Run ROM using YMFM backend
go run -tags ymfm_cgo,no_sdl_ttf ./cmd/emulator \
  -rom roms/ym2608_demo_song.rom \
  -audio-backend ymfm

# Optional: headless capture + compare against Resources/Demo.wav
go run -tags ymfm_cgo ./cmd/rom_audio_capture \
  -rom roms/ym2608_demo_song.rom \
  -out /tmp/ym2608_demo_song_capture.wav \
  -frames 4500 \
  -audio-backend ymfm
go run ./cmd/wav_compare \
  -ref Resources/Demo.wav \
  -got /tmp/ym2608_demo_song_capture.wav \
  -seconds 30
```

### `build_pong_ym2608.go`
**Purpose**: Build a simple Pong clone with YM2608 background music replayed from `Resources/Demo.vgz`.

**Gameplay**:
- Left paddle: `Up` / `Down` (or mapped equivalents)
- Press `START` to begin the match (and start music)
- Right paddle: simple AI
- First to 3 points wins
- Music starts when the match begins and continues during the match

**Notes**:
- This ROM inlines per-frame game logic plus per-frame YM writes, so full-song playback can exceed bank budget.
- Default settings target a stable playable build: about `25s` of BGM (`1500` VGM frames).
- The BGM segment loops continuously during gameplay.

**Usage**:
```bash
# Build Pong + YM2608 BGM ROM
go run -tags testrom_tools ./test/roms/build_pong_ym2608.go \
  -in Resources/Demo.vgz \
  -out roms/pong_ym2608_demo.rom

# Run with YMFM backend
go run -tags ymfm_cgo,no_sdl_ttf ./cmd/emulator \
  -rom roms/pong_ym2608_demo.rom \
  -audio-backend ymfm
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
./corelx test/roms/corelx_comprehensive_test.corelx roms/corelx_comprehensive_test.rom

# Test with harness
go build ./cmd/test_corelx_features
./test_corelx_features roms/corelx_comprehensive_test.rom

# Or run in emulator
./nitro-core-dx roms/corelx_comprehensive_test.rom
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
