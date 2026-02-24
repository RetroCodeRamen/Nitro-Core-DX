# Test Suite Summary

This document provides an overview of the test suite for Nitro-Core-DX.

## Test Files

### PPU Tests
- **`internal/ppu/ppu_test.go`**: Basic PPU functionality (sprite rendering, OAM access)
- **`internal/ppu/features_test.go`**: New feature tests
  - `TestSpritePriority`: Sprite priority sorting
  - `TestSpriteBlending`: Sprite blending modes (alpha, additive, subtractive)
  - `TestMosaicEffect`: Mosaic pixel grouping
  - `TestMatrixModeOutsideScreen`: Matrix Mode outside-screen handling (repeat/backdrop/character #0)
  - `TestMatrixModeDirectColor`: Direct color mode
  - `TestDMATransfer`: DMA copy and fill modes
  - `TestSpriteToBackgroundPriority`: Sprite-to-background priority interaction

### CPU Tests
- **`internal/cpu/cpu_test.go`**: Basic CPU functionality
- **`internal/cpu/interrupt_test.go`**: Interrupt system tests
  - `TestInterruptSystem`: IRQ interrupt handling
  - `TestNMIInterrupt`: Non-maskable interrupt
  - `TestIRQMasked`: IRQ masking with I flag

### APU Tests
- **`internal/apu/pcm_test.go`**: PCM playback tests
  - `TestPCMPlayback`: Basic PCM playback
  - `TestPCMPlaybackLoop`: PCM looping
  - `TestPCMPlaybackOneShot`: One-shot PCM playback
  - `TestPCMVolume`: PCM volume control

## Running Tests

Run all tests:
```bash
go test -tags no_sdl_ttf ./...
```

Run specific package tests:
```bash
go test ./internal/ppu -v
go test ./internal/cpu -v
go test ./internal/apu -v
go test -tags no_sdl_ttf ./internal/emulator -v
```

Run specific test:
```bash
go test ./internal/ppu -v -run TestSpritePriority
```

## Test Coverage

The test suite covers:
- ✅ Sprite priority system
- ✅ Sprite blending modes
- ✅ Mosaic effect
- ✅ Matrix Mode outside-screen handling
- ✅ Matrix Mode direct color
- ✅ DMA transfers (copy and fill)
- ✅ Sprite-to-background priority
- ✅ Interrupt system (IRQ/NMI)
- ✅ PCM playback (loop and one-shot)

## Notes

Some tests are intentionally long-running (especially emulator audio timing tests) and may require higher timeouts in local runs/CI.

If SDL2_ttf development libraries are not installed locally, use the `no_sdl_ttf` build tag for emulator/UI-related builds and tests.

Generator utilities under `cmd/testrom` and `test/roms` are gated behind the `testrom_tools` build tag to avoid multiple `main()` conflicts during default `go test ./...` runs.
