# Changelog

All notable changes to the Nitro-Core-DX project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

**Note:** This changelog was created on January 27, 2026. Previous changes have been reconstructed from project documentation and commit history.

---

## [Unreleased]

### Planned
- Tile Viewer panel for visual VRAM inspection
- Advanced debugging tools (breakpoints, watchpoints)
- Interrupt system implementation
- Matrix Mode (Mode 7-style) transformation
- SDK asset pipeline
- IDE integration

---

## [0.2.0] - 2026-01-27

### Added
- **Cycle-by-Cycle Debug Logger** - Comprehensive logging system that records CPU registers, PPU state (scanline, dot, VBlank flag, frame counter), APU state (all 4 channels), and key memory locations for each clock cycle
  - Command-line flags: `-cyclelog <file>`, `-maxcycles <N>`, `-cyclestart <N>`
  - UI toggle: Debug → Toggle Cycle Logging
  - Supports start cycle offset to skip initialization
  - Single file output with all state information
- **Register Viewer Panel** - Real-time CPU register display with:
  - All registers (R0-R7, PC, SP, PBR, DBR, Flags)
  - Scrollable display (fixes off-screen issue)
  - Copy All button for clipboard access
  - Save State button to save register state to timestamped file
  - Binary representation for registers
- **Memory Viewer Panel** - Hex dump viewer with:
  - Bank selector (0-255)
  - Offset selector (0x0000-0xFFFF)
  - 16 bytes per line display
  - ASCII representation
  - Real-time updates
- **PPU State Getters** - Added `GetScanline()` and `GetDot()` methods for debugging
- **OAM/PPU/APU Adapters** - Interface adapters to avoid import cycles in debug logger

### Fixed
- **MOV Mode 2 I/O Register Bug** (Critical) - Fixed sprite movement issue
  - **Problem**: Mode 2 was reading 16 bits from I/O registers (which are 8-bit only)
  - **Impact**: ROMs reading VBlank flag got 0x0100 instead of 0x0001, causing infinite wait loops
  - **Solution**: Automatic detection of I/O addresses (bank 0, offset >= 0x8000) - reads 8-bit and zero-extends to 16-bit
  - **Location**: `internal/cpu/instructions.go:30-49`
  - **Hardware Compatibility**: ✅ FPGA-implementable using standard address decoding logic
- **VBlank Flag Timing** - Improved flag persistence through entire VBlank period
  - Flag now correctly persists through scanlines 200-219
  - Fixed flag re-set logic to ensure it's available throughout VBlank

### Changed
- **MOV Mode 2 Behavior** - Now automatically detects I/O vs normal memory:
  - I/O registers: Reads 8-bit, zero-extends to 16-bit
  - Normal memory: Reads 16-bit as before
- **MOV Mode 3 Behavior** - Already had I/O detection, now consistent with Mode 2
- **Programming Manual** - Updated to document automatic I/O register detection
- **Documentation** - Consolidated and updated all documentation

### Documentation
- Updated `NITRO_CORE_DX_PROGRAMMING_MANUAL.md` to version 1.1
- Documented automatic I/O register detection in MOV instructions
- Updated `MASTER_PLAN.md` with current status
- Updated `README.md` with latest features

---

## [0.1.0] - 2026-01-06 to 2026-01-26

### Added
- **Clock-Driven Architecture** - Complete refactor to cycle-accurate, FPGA-ready design
  - Master clock scheduler coordinating CPU, PPU, and APU
  - PPU scanline/dot stepping for pixel-perfect rendering
  - APU fixed-point audio synthesis
- **Memory System Split** - Separated Bus and Cartridge for better organization
- **Save State System** - Complete save/load state implementation using encoding/gob
- **Logging System** - Centralized logging with component filtering
- **Basic Debugging Tools** - Initial debugger infrastructure
- **Fyne UI Framework** - External UI using Fyne with SDL2 for rendering
- **ROM Builder** - Tools for building test ROMs

### Fixed
- **CPU Reset() Corruption Bug** (Critical)
  - Issue: Reset() set PCBank=0, causing crashes after ROM load
  - Fix: Reset() no longer resets PCBank/PCOffset/PBR
  - Location: `internal/cpu/cpu.go:74-92`
- **Frame Execution Order Bug** (Critical)
  - Issue: VBlank flag set AFTER CPU execution
  - Fix: PPU.RenderFrame() moved before CPU execution (now clock-driven)
  - Location: `internal/emulator/emulator.go`
- **MOV Mode 3 I/O Write Bug** (Critical)
  - Issue: Always wrote 8-bit to I/O, breaking 16-bit writes
  - Fix: Write 16-bit to non-I/O addresses, 8-bit to I/O
  - Location: `internal/cpu/instructions.go:38-53`
- **Logger Goroutine Leak**
  - Issue: Logger goroutine never shut down
  - Fix: Added logger.Shutdown() calls in UI cleanup
  - Location: `internal/ui/ui.go`, `internal/ui/fyne_ui.go`
- **Division by Zero**
  - Issue: Returned 0xFFFF silently
  - Fix: Added FlagD (division by zero flag)
  - Location: `internal/cpu/instructions.go:171-188`
- **Stack Underflow**
  - Issue: Returned 0 without error
  - Fix: Pop16() now returns error on underflow
  - Location: `internal/cpu/instructions.go:501-517`
- **APU Duration Loop Mode**
  - Issue: Didn't reload initial duration
  - Fix: Store InitialDuration, reload on loop
  - Location: `internal/apu/apu.go`

### Changed
- **Architecture**: Migrated from frame-based to clock-driven execution
- **PPU Rendering**: Changed from frame-based to scanline/dot stepping
- **APU Audio**: Migrated to fixed-point arithmetic for FPGA compatibility
- **Memory System**: Split into Bus (routing) and Cartridge (ROM storage)

### Documentation
- Created `SYSTEM_MANUAL.md` - Complete system architecture documentation
- Created `NITRO_CORE_DX_PROGRAMMING_MANUAL.md` - Programming guide for ROM developers
- Created `MASTER_PLAN.md` - Consolidated planning and review document
- Consolidated documentation from multiple files into main documents
- Archived historical documentation to `docs/archive/`

---

## [0.0.1] - Initial Development

### Added
- **Core CPU Emulation** - 16-bit CPU with banked 24-bit addressing
  - 8 general-purpose registers (R0-R7)
  - Complete instruction set (arithmetic, logical, branching, jumps)
  - Cycle-accurate execution
- **Memory System** - Banked memory architecture
  - Bank 0: WRAM (32KB) + I/O registers
  - Banks 1-125: ROM space
  - Banks 126-127: Extended WRAM (128KB)
- **PPU (Graphics)** - Picture Processing Unit
  - 4 background layers (BG0-BG3)
  - Sprite system (128 sprites)
  - VRAM, CGRAM, OAM management
  - 320x200 pixel display
- **APU (Audio)** - Audio Processing Unit
  - 4 audio channels
  - Waveforms: sine, square, saw, noise
  - Duration control with loop mode
  - 44.1 kHz sample rate
- **Input System** - Dual controller support with 12 buttons
- **ROM Loading** - ROM header parsing and execution
- **Basic UI** - Initial user interface

### Known Issues
- Various bugs documented in MASTER_PLAN.md (all fixed in later versions)

---

## Version History Notes

**Version 0.2.0** marks a significant milestone:
- ✅ Sprite movement working correctly
- ✅ Comprehensive debugging tools
- ✅ FPGA-ready architecture complete
- ✅ All critical bugs fixed

**Version 0.1.0** represents the clock-driven refactor:
- Complete architecture overhaul for FPGA compatibility
- Cycle-accurate execution
- Hardware-accurate synchronization signals

**Version 0.0.1** was the initial development phase:
- Core emulation systems implemented
- Basic functionality working
- Foundation for future improvements

---

## Format Notes

- **Added** - New features
- **Changed** - Changes to existing functionality
- **Deprecated** - Features that will be removed
- **Removed** - Removed features
- **Fixed** - Bug fixes
- **Security** - Security fixes

---

## Links

- [README.md](README.md) - Project overview
- [SYSTEM_MANUAL.md](SYSTEM_MANUAL.md) - System architecture
- [NITRO_CORE_DX_PROGRAMMING_MANUAL.md](NITRO_CORE_DX_PROGRAMMING_MANUAL.md) - Programming guide
- [MASTER_PLAN.md](MASTER_PLAN.md) - Development planning
