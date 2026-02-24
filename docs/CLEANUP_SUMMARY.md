# Project Cleanup Summary

**Date:** January 27, 2026  
**Last Updated:** January 27, 2026

> **Historical Snapshot:** Cleanup summary for a prior reorganization pass. Current documentation structure/source-of-truth guidance is in `docs/README.md`.

## Overview

This document summarizes the cleanup performed to organize the Nitro-Core-DX project, ensuring only the correct FPGA-ready emulator binary is used and documentation is properly consolidated into four main documents.

## Emulator Binary

### Correct Binary: `nitro-core-dx`

The **correct emulator binary** is `nitro-core-dx`, built from `cmd/emulator/main.go`. This is the FPGA-ready, clock-driven emulator with cycle-accurate execution.

**Build Command:**
```bash
go build -tags "no_sdl_ttf" -o nitro-core-dx ./cmd/emulator
```

### Removed Binaries

The following old/test binaries were removed from the project root:
- `./emulator` - Old/outdated binary (use `nitro-core-dx` instead)
- `./audiotest` - Test utility (build on-demand: `go build -o audiotest ./cmd/audiotest`)
- `./demorom` - Demo ROM generator (build on-demand: `go build -o demorom ./cmd/demorom`)
- `./testrom` - Test ROM generator (build on-demand: `go build -o testrom ./cmd/testrom`)

**Note:** All binaries are now in `.gitignore` to prevent committing built binaries.

## Documentation Organization

### Essential Documentation (Root Level)

All documentation has been consolidated into **four main documents**:

- **README.md** - Project overview, quick start, build instructions, troubleshooting, and contributing guide
- **SYSTEM_MANUAL.md** - Complete system architecture, FPGA compatibility, testing framework, and development tools
- **NITRO_CORE_DX_PROGRAMMING_MANUAL.md** - Complete programming guide for ROM developers

**Consolidated Content:**
- `BUILD_INSTRUCTIONS.md` → Merged into `README.md` (Quick Start section)
- `INSTALL_SDL_TTF.md` → Merged into `README.md` (Prerequisites section)
- `CONTRIBUTING.md` → Merged into `README.md` (Contributing section)
- `TESTING_FRAMEWORK.md` → Merged into `SYSTEM_MANUAL.md` (Testing Framework section)
- `DEV_TOOLS_IMPLEMENTATION_PLAN.md` → Merged into `SYSTEM_MANUAL.md` (Development Tools section)

### Archived Documentation

Historical and completed documentation has been moved to `docs/archive/`:

#### `docs/archive/test_results/`
- WEEK1_FIXES_TEST_RESULTS.md - Historical test results from Week 1
- WEEK2_FIXES_TEST_RESULTS.md - Historical test results from Week 2

#### `docs/archive/reviews/`
- ARCHITECTURE_REVIEW.md - Old architecture review (superseded by SYSTEM_MANUAL.md)
- ARCHITECTURE_REVIEW_SUMMARY.md - Summary of old architecture review
- CODE_REVIEW_REPORT.md - Historical code review report
- HARDWARE_ACCURACY_REVIEW.md - Historical hardware accuracy review

#### `docs/archive/plans/`
- CLOCK_DRIVEN_CLEANUP.md - Completed cleanup notes (clock-driven refactor complete)
- CLOCK_DRIVEN_REFACTOR.md - Completed refactoring notes (now documented in SYSTEM_MANUAL.md)
- DEVELOPMENT_TOOLS_PLAN.md - Superseded by DEV_TOOLS_IMPLEMENTATION_PLAN.md
- TESTING_LOGGING.md - Specific testing guide (merged into TESTING_FRAMEWORK.md)
- CPU_COMPARISON.md - CPU architecture comparison (reference material)
- MASTER_PLAN.md - Historical master plan
- SDK_FEATURES_PLAN.md - Historical SDK features plan

## Project Structure

```
Nitro-Core-DX/
├── cmd/
│   ├── emulator/          # Main FPGA-ready emulator (builds to nitro-core-dx)
│   ├── audiotest/         # Audio test ROM generator
│   ├── demorom/           # Demo ROM generator
│   ├── testrom/           # Test ROM generator
│   └── ...                # Other utilities
├── internal/              # Core emulator code
├── test/                  # Test ROMs
├── docs/
│   ├── archive/           # Historical/completed documentation
│   └── README.md          # Documentation index
├── README.md              # Main project overview
├── BUILD_INSTRUCTIONS.md  # Build instructions
├── SYSTEM_MANUAL.md       # System architecture
└── ...                    # Other essential docs
```

## Key Points

1. **Always use `nitro-core-dx`** - This is the FPGA-ready, clock-driven emulator
2. **Build binaries on-demand** - Don't commit built binaries (they're in `.gitignore`)
3. **Check essential docs first** - Start with README.md, BUILD_INSTRUCTIONS.md, and SYSTEM_MANUAL.md
4. **Historical docs archived** - Old reviews, test results, and completed plans are in `docs/archive/`

## Next Steps

- Continue development using `nitro-core-dx` as the main emulator
- Build test utilities on-demand as needed
- Refer to essential documentation in the project root
- Historical documentation available in `docs/archive/` for reference
