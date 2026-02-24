# Testing Documentation

This directory contains testing guides, test results, and testing procedures.

## Quick Commands

Recommended local test targets (repo root):

```bash
make test-fast    # Fast core regression checks
make test-emulator
make test-full    # Full local baseline (uses -tags no_sdl_ttf)
make test-long    # Expensive emulator audio timing tests
```

These targets default to `-tags no_sdl_ttf` for environments without SDL2_ttf development libraries.

## Files

- **TEST_ROM_FIXES.md** - Test ROM fixes and improvements
- **TEST_RESULTS.md** - Test execution results
- **TEST_SUMMARY.md** - Summary of test results
- **TEST_ROM_TESTING_GUIDE.md** - Guide for testing ROMs
- **TEST_ROM_FEATURE_BUILD.md** - Test ROM feature build documentation
- **TESTING_PLAN.md** - Overall testing plan
- **test_rom_diagnostics.md** - Test ROM diagnostic information
- **INPUT_TESTING_GUIDE.md** - Input system testing guide
