# Changelog

All notable changes to the Nitro-Core-DX project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

**Note:** This changelog was created on January 27, 2026. Previous changes have been reconstructed from project documentation and commit history.

---

## [Unreleased]

### Changed
- **Documentation Reorganization** - Reorganized and cleaned up documentation structure (2026-01-31)
  - Moved narrative/bloggy sections from README to `docs/DEVELOPMENT_NOTES.md`
  - Organized fix/issue documents into `docs/issues/` directory
  - Organized testing documents into `docs/testing/` directory
  - Organized specification documents into `docs/specifications/` directory
  - Simplified README to be concise and reference-based
  - Created README files in each docs subdirectory for navigation
  - Location: Multiple files reorganized

### Added
- **GUI Logging Controls** - Added logging component controls in Debug menu
  - Enable/disable logging for CPU, PPU, APU, Memory, Input, UI, System components
  - "Enable All Logging" and "Disable All Logging" options
  - Location: `internal/ui/fyne_ui.go` - Debug menu
- **Input Debug Logging** - Added debug logging for input system
  - Logs all input reads and writes with offset and value
  - Helps diagnose input issues and verify latch behavior
  - Location: `internal/memory/bus.go` - input I/O logging
- **Input System Unit Tests** - Created comprehensive unit tests for input system
  - Tests latch behavior, edge-triggered latching, multiple buttons, controller 2
  - Location: `internal/input/input_test.go`
- **Test ROM Input Generator** - Created test ROM generator for input testing
  - Generates ROM that displays sprite moved by arrow keys/WASD
  - Tests input latching, button reading, and sprite movement
  - Location: `cmd/testrom_input/main.go`
- **Input Testing Guide** - Created testing documentation
  - Manual and automated testing procedures
  - Expected behavior and controls
  - Location: `INPUT_TESTING_GUIDE.md`

### Changed
- **CoreLX Debugging Documentation** - Created `CORELX_DEBUGGING_ISSUES.md` to track compiler bugs and debugging progress
  - Documents fixed compiler bugs (VRAM address calculation, binary operations)
  - Tracks ongoing blank screen issue with CoreLX-compiled ROMs
  - Provides test ROMs and debugging checklist
  - Location: `CORELX_DEBUGGING_ISSUES.md`
- **Fyne Log Viewer Panel** - Implemented log viewer panel for Fyne UI
  - Text selection and copy functionality (Ctrl+C)
  - Component and log level filtering
  - Auto-scroll and save logs functionality
  - Location: `internal/ui/panels/log_viewer_fyne.go`
- **CoreLX Test ROMs** - Created test ROMs for debugging compiler issues
  - `moving_sprite_colored.corelx` - Recreation of working Go ROM in CoreLX
  - `moving_sprite_colored_simple.corelx` - Simplified version with hardcoded values
  - Location: `test/roms/`

### Changed
- **Input System Refactor** - Refactored input system to match FPGA latch-based behavior (2026-01-31)
  - Changed from direct button state reading to latch-based serial shift register interface
  - Implements edge-triggered latching (captures on 0->1 transition)
  - Latched state persists until next latch, matching FPGA behavior
  - Location: `internal/input/input.go`
- **Test ROM Wrapping Logic** - Improved sprite position wrapping in test ROM (2026-01-31)
  - Uses BGT (Branch if Greater Than) for proper X > 319 check
  - Handles unsigned wrap (65535) and signed comparison (X >= 320)
  - Location: `cmd/testrom_input/main.go`

### Fixed
- **Input System FPGA Compatibility** - Fixed input system to match FPGA specification (2026-01-31)
  - Input now uses latch mechanism: write 1 to latch register captures button state
  - Reading input returns latched state, not current state
  - Edge-triggered latching ensures proper button capture timing
  - Location: `internal/input/input.go` - `Write8` and `Read8` methods
- **Savestate Input Fields** - Fixed savestate to use new input system field names (2026-01-31)
  - Updated from `LatchActive`/`Controller2LatchActive` to `Controller1Latched`/`Controller2Latched`
  - Added `Controller1LatchState` and `Controller2LatchState` fields
  - Location: `internal/emulator/savestate.go`
- **Memory Bus Logger Support** - Added logger support to memory bus for input debugging (2026-01-31)
  - Bus now has logger field and SetLogger method
  - Enables input debug logging through bus
  - Location: `internal/memory/bus.go`
- **CoreLX Compiler: VRAM Address Calculation** - Fixed `tiles16` VRAM address calculation
  - Changed from `base * 32` to `base * 128` for 16x16 tiles
  - Impact: 16x16 tiles now load to correct VRAM addresses
  - Location: `internal/corelx/codegen.go` - `generateInlineTileLoad()`
- **CoreLX Compiler: Binary OR Operation** - Fixed register usage in binary OR expressions
  - Left result was saved to R1 but operation used destReg (R0)
  - Now correctly uses R1 for OR operation then moves result to destReg
  - Impact: Expressions like `SPR_PAL(1) | SPR_PRI(0)` now work correctly
  - Location: `internal/corelx/codegen.go` - `BinaryExpr` case `TOKEN_PIPE`
- **CoreLX Compiler: Binary AND Operation** - Fixed register usage in binary AND expressions
  - Same fix as OR operation
  - Impact: Bitwise AND operations now work correctly
  - Location: `internal/corelx/codegen.go` - `BinaryExpr` case `TOKEN_AMPERSAND`
- **CoreLX Compiler: Binary ADD/SUB Operations** - Fixed to restore left result before operation
  - Left result saved to R1, but operations used destReg directly
  - Now restores left result from R1 to destReg before performing operation
  - Impact: Addition and subtraction expressions now work correctly
  - Location: `internal/corelx/codegen.go` - `BinaryExpr` cases `TOKEN_PLUS` and `TOKEN_MINUS`
- **PPU Logging Performance** - Optimized PPU logging to reduce performance impact
  - OAM logging limited to first 4 sprites, every 60 frames
  - VRAM logging limited to first 32 bytes, first frame only
  - CGRAM logging limited to first 20 colors, first frame only
  - Impact: Reduced logging overhead from 30 FPS to 7 FPS back to ~30 FPS
  - Location: `internal/ppu/ppu.go`

### Changed
- **UI Consolidation** - Removed all SDL2-based UI code, using Fyne exclusively
  - Deleted: `internal/ui/ui.go`, `ui_render.go`, `menu.go`, `toolbar.go`, `statusbar.go`, `font.go`
  - Deleted: `internal/ui/panels/log_viewer.go`, `log_controls.go`
  - Removed redundant toolbar (controls now only in menu)
  - Location: `internal/ui/`
- **Fyne UI Layout** - Improved resizable log viewer and dynamic panel visibility
  - Log viewer and debug panels now hide when disabled
  - Splitter adjusts automatically based on panel visibility
  - Location: `internal/ui/fyne_ui.go`
- **CoreLX Compiler Entry Point** - Enhanced `__Boot()` function support
  - Compiler now ensures `__Boot()` is generated first if present
  - Sets entry point to 0x8000 for `__Boot()` function
  - Location: `cmd/corelx/main.go`, `internal/corelx/codegen.go`

### Removed
- **Old CoreLX Documentation Files** - Cleaned up redundant CoreLX documentation
  - Removed 10 old CoreLX status/implementation/guide files
  - Consolidated into `docs/CORELX.md` and `CORELX_DEBUGGING_ISSUES.md`
  - Files: `CORELX_APU_IMPLEMENTATION.md`, `CORELX_COMPILER_STATUS.md`, etc.

### Known Issues
- **CoreLX Blank Screen Issue** - CoreLX-compiled ROMs show blank screen
  - Test ROMs: `moving_sprite_colored_corelx.rom`, `moving_sprite_colored_simple.rom`
  - Possible causes: Variable persistence, `wait_vblank()` loop, OAM writes, tile loading
  - Status: In progress - see `CORELX_DEBUGGING_ISSUES.md` for details
  - Date: 2026-01-30

### Added
- **CoreLX Compiler** - Complete compiler implementation for CoreLX language
  - Lexer, parser, semantic analyzer, and code generator
  - Lua-like syntax compiled to Nitro-Core-DX bytecode
  - Documentation: `CORELX_PROGRAMMING_GUIDE.md`, `CORELX_COMPILER_STATUS.md`
  - Test ROMs and examples in `test/roms/`
  - Location: `cmd/corelx/`, `internal/corelx/`
- **Interactive Debugger** - Debugger tool for ROM development
  - Breakpoints, step execution, register viewing
  - Documentation: `docs/DEBUGGING_GUIDE.md`, `docs/DEBUGGING_QUICK_START.md`
  - Location: `cmd/debugger/`, `internal/debug/debugger.go`
- **ROM Builder Enhancements** - Added `EncodeXOR` function for XOR instruction encoding
  - Location: `internal/rom/builder.go`
- **Input System Update** - Changed ButtonSELECT to ButtonZ for better controller mapping
  - Z key now maps to ButtonZ instead of SELECT
  - Location: `internal/input/input.go`, `internal/ui/ui.go`

### Changed
- **Code Refactoring Rollback** - Rolled back refactoring changes that caused performance degradation
  - Restored original CPU, memory, and emulator implementations
  - Performance restored from ~18 FPS back to ~60 FPS
  - Location: `internal/cpu/`, `internal/memory/`, `internal/emulator/`
  - Date: 2026-01-28
- **Console Mockup Images** - Added prototype design images to README
  - Console isometric view
  - Console top view
  - Controller design
  - Images showcase what the physical console will look like
- **Test Suite** - Comprehensive test coverage for all new features
  - Sprite priority, blending, and mosaic effect tests
  - Matrix Mode outside-screen and direct color tests
  - DMA transfer tests
  - PCM playback tests (basic, loop, one-shot, volume)
  - Interrupt system tests (IRQ, NMI, masking)
  - Test documentation (TEST_SUMMARY.md, TEST_RESULTS.md, TEST_FIXES.md)
- **ROM Compatibility Fix** - Fixed sprite blending backward compatibility
  - Normal mode (blendMode=0) now ignores alpha value
  - Maintains compatibility with ROMs using control byte 0x03
  - Fixed sprite transparency issue in normal mode
- **Matrix Mode Outside-Screen Handling** - Complete outside-screen coordinate handling
  - Repeat/wrap mode (default)
  - Backdrop mode (render backdrop color when outside bounds)
  - Character #0 mode (render tile 0 when outside bounds)
- **Matrix Mode Direct Color Mode** - Direct RGB color rendering
  - Bypass CGRAM palette lookup
  - Direct 4-bit per channel color expansion
  - Per-layer direct color control
- **PCM Playback** - Complete PCM audio playback system
  - PCM channel support (one per audio channel)
  - 8-bit signed PCM sample playback
  - Loop and one-shot playback modes
  - PCM volume control
  - Integrated with existing audio channel system
- **Sprite Blending/Alpha** - Complete sprite blending system
  - Normal, alpha, additive, and subtractive blend modes
  - Alpha transparency (0-15 levels)
  - Sprite-to-background blending
- **Mosaic Effect** - Per-layer mosaic support
  - Configurable mosaic size (1-15 pixels)
  - Pixel grouping for retro/pixelated effects
- **DMA System** - Direct Memory Access for fast transfers
  - Memory to VRAM/CGRAM/OAM transfers
  - Copy and fill modes
  - DMA registers and control
- **Sprite Priority System** - Complete sprite priority sorting and rendering
  - Sprites sorted by priority (bits [7:6] of attributes)
  - Proper sprite-to-background priority interaction
  - Unified priority system (BG3=3, BG2=2, BG1=1, BG0=0, Sprites=0-3)
  - Sprites can render behind backgrounds based on priority
- **Interrupt System** - Complete interrupt handling implementation
  - IRQ/NMI handlers with vector table
  - Interrupt vector table (bank 0, addresses 0xFFE0-0xFFE3)
  - VBlank interrupt (IRQ) automatically triggered
  - Interrupt state saving (PC, PBR, Flags to stack)
  - Non-maskable interrupt (NMI) support
  - Interrupt enable/disable via I flag
- **Per-Layer Matrix Mode** - Each background layer (BG0-BG3) now supports independent Matrix Mode transformations
  - BG1, BG2, BG3 matrix registers (0x802B-0x8051)
  - Multiple simultaneous 3D objects (roads, buildings, boxes, etc.)
  - Per-layer matrix control, center points, and mirroring
- **HDMA Matrix Updates** - Per-scanline matrix parameter updates via HDMA
  - Matrix A, B, C, D, Center X, Center Y can be updated every scanline
  - Enables advanced perspective effects (roads, buildings, 3D landscapes)
  - HDMA table format: 64 bytes per scanline (4 layers × 16 bytes)
- **Enhanced Matrix Mode Capabilities**:
  - Multiple simultaneous transformations (SNES Mode 7 could only do one layer)
  - Per-scanline perspective effects
  - 3D town scenes with multiple transformed objects
  - Independent transformations per layer
- **Timing Synchronization** - Unified clock system with CPU and PPU synchronized at ~7.67 MHz (Genesis-like speed)
  - CPU speed: 7,670,000 Hz (changed from 10 MHz)
  - PPU timing: 220 scanlines × 581 dots = 127,820 cycles per frame
  - APU timing: ~174 cycles per sample (adjusted for new CPU speed)
- **Performance Optimizations** - Improved frame rendering performance
  - Optimized PPU `StepPPU()` to process scanlines in batches instead of dot-by-dot
  - Removed debug logging overhead (SPRITE0_STATE logging removed)
  - Batch stepping for CPU/PPU when cycle logging disabled
  - Performance improved from ~27 FPS to ~35 FPS
- **NitroLang Language Design** - Designed new compiled language with Lua-like syntax
  - Documentation: `docs/LANGUAGE_DESIGN.md`
  - Renamed from "NitroScript" to "NitroLang" (compiled, not interpreted)
  - Features: Lua-like syntax, compiled to bytecode, inline assembly support
- **Timing Analysis Documentation** - Added timing analysis and fix summary documents
  - `docs/TIMING_ANALYSIS.md` - Detailed timing analysis and design decisions
  - `docs/TIMING_FIX_SUMMARY.md` - Summary of timing synchronization changes

### Changed
- **Matrix Mode Implementation** - Fully implemented with per-layer support
  - Matrix Mode status updated from "not fully implemented" to "fully implemented"
  - Legacy Matrix Mode registers (0x8018-0x802A) now map to BG0 for backward compatibility
- **Register Map Updates**:
  - Window registers moved to 0x8052-0x805C (was 0x802B-0x8035)
  - HDMA registers moved to 0x805D-0x805F (was 0x8036-0x8038)
  - New per-layer matrix registers added (0x802B-0x8051)
- **CPU Clock Speed** - Reduced from 10 MHz to ~7.67 MHz (Genesis-like)
  - Target: 127,820 cycles per frame at 60 FPS
  - Better matches Genesis console speed
- **PPU Timing** - Adjusted to match CPU speed
  - Dots per scanline: 360 → 581
  - HBlank dots: 40 → 261
  - Total cycles per frame: 79,200 → 127,820
- **Logging System** - All logging disabled by default
  - Console logging removed unless `-log` flag is used
  - Removed SPRITE0_STATE debug output that was causing performance issues
  - Cycle logging only enabled with `-cyclelog` flag
- **Build System** - Added `-tags no_sdl_ttf` build option
  - Allows building without SDL2_ttf dependency
  - Uses simple bitmap font renderer as fallback
- **Documentation** - Updated Programming Manual with new features
  - Added per-layer matrix register documentation
  - Added HDMA matrix update documentation
  - Added interrupt system documentation (new section)
  - Updated register map with new matrix registers
  - Added examples for multiple simultaneous matrix transformations
  - Added interrupt handler examples

### Fixed
- **Performance Issues** - Removed excessive logging overhead
  - Removed `fmt.Printf` statements that were printing to console every frame
  - Optimized PPU rendering loop
  - FPS improved from ~27 to ~35

### Planned
- Tile Viewer panel for visual VRAM inspection
- Advanced debugging tools (breakpoints, watchpoints)
- Interrupt system implementation
- Matrix Mode (Mode 7-style) transformation
- SDK asset pipeline
- IDE integration
- Further PPU rendering optimizations to reach 60 FPS

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
