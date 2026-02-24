# Testing Documentation

This directory contains testing guides, test results, and testing procedures.

## Current Entry Points

- `TEST_SUMMARY.md`
  - Current recommended commands and baseline test expectations
- `TEST_ROM_TESTING_GUIDE.md`
  - Manual ROM testing workflow
- `INPUT_TESTING_GUIDE.md`
  - Input-specific testing guide

## Historical / Snapshot Test Docs

Older milestone-specific testing plans/results/fix logs have been moved to `docs/archive/test_results/` to keep this folder focused on current workflows.

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

- **TEST_SUMMARY.md** - Summary of test results
- **TEST_ROM_TESTING_GUIDE.md** - Guide for testing ROMs
- **INPUT_TESTING_GUIDE.md** - Input system testing guide
- **(Archived)** Historical testing plans/results/fix logs - `docs/archive/test_results/`
