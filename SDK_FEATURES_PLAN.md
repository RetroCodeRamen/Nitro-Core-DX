# Nitro-Core-DX SDK Features Plan

**Created: January 6, 2026**

## Overview

This document outlines essential features for a complete Software Development Kit (SDK) for Nitro-Core-DX. The goal is to provide everything a developer needs to create games efficiently, from asset creation to debugging to final ROM distribution.

---

## Core SDK Components

### 1. ROM Builder & Toolchain

#### 1.1 Assembler/Compiler
**Status**: ✅ Basic ROM builder exists (`cmd/rombuilder`)

**Needs Enhancement**:
- **Macro Support**: Define reusable code blocks
- **Include Files**: Modular code organization
- **Labels & Symbols**: Better symbol management
- **Linker**: Support for multiple source files
- **Preprocessor**: Conditional compilation, defines
- **Error Messages**: Clear, helpful error messages with line numbers
- **Warnings**: Non-fatal issues (unused labels, etc.)

**Example**:
```assembly
; macros.inc
.macro SET_PALETTE_COLOR palette, index, r, g, b
    MOV R7, #0x8012        ; CGRAM_ADDR
    MOV R0, #(palette * 16 + index)
    MOV [R7], R0
    ; ... set color ...
.endmacro

; main.asm
.include "macros.inc"
.include "constants.inc"

main:
    SET_PALETTE_COLOR 0, 1, 255, 0, 0  ; Red
```

#### 1.2 ROM Validator
**Purpose**: Validate ROMs before distribution

**Features**:
- Check ROM header validity
- Verify entry point is valid
- Check for common errors (unaligned jumps, invalid addresses)
- Size validation (warn if ROM is too large)
- Checksum calculation
- ROM info display (size, entry point, mapper flags)

**Usage**:
```bash
nitro-rom-validator game.rom
# Output:
# ✓ ROM header valid
# ✓ Entry point: 01:8000
# ✓ ROM size: 64KB
# ⚠ Warning: Unused memory at end of ROM
```

#### 1.3 ROM Analyzer
**Purpose**: Analyze ROM contents for debugging

**Features**:
- Disassemble ROM code
- List all symbols/labels
- Show memory usage breakdown
- Identify unused code/data
- Export symbol table
- Find all jump/call targets

---

### 2. Asset Pipeline Tools

#### 2.1 Image to Tile Converter
**Purpose**: Convert PNG/other images to VRAM tile format

**Features**:
- Convert image to 4bpp tiles
- Automatic palette extraction
- Support for 8x8 and 16x16 tiles
- Tile optimization (detect duplicate tiles)
- Export to VRAM binary format
- Preview tiles with palette

**Usage**:
```bash
nitro-img2tiles sprite.png --size 8x8 --palette auto --output tiles.bin
# Creates: tiles.bin, palette.bin, tilemap.bin
```

#### 2.2 Palette Converter
**Purpose**: Convert RGB images to console palette format

**Features**:
- RGB888 → RGB555 conversion
- Palette optimization (reduce to 16 colors per palette)
- Dithering options
- Preview with console palette
- Export CGRAM format

**Usage**:
```bash
nitro-palette-convert image.png --palettes 4 --output palette.cgram
```

#### 2.3 Sprite Sheet Packer
**Purpose**: Organize sprites into efficient VRAM layout

**Features**:
- Pack multiple sprite images into VRAM
- Generate OAM data
- Optimize tile usage (shared tiles)
- Export sprite metadata (X, Y, tile index, palette)
- Preview sprite sheet

#### 2.4 Audio Converter
**Purpose**: Convert audio files to APU format

**Features**:
- WAV/MP3 → Frequency/note conversion
- Generate musical scale data
- Export frequency tables
- Preview audio with console limitations
- Generate APU initialization code

**Usage**:
```bash
nitro-audio-convert melody.wav --format notes --output notes.inc
# Generates frequency tables and note data
```

#### 2.5 Tilemap Editor
**Purpose**: Visual tilemap creation

**Features**:
- Visual tilemap editor (like Tiled but for Nitro-Core-DX)
- Paint tiles onto map
- Multiple layers support
- Export to VRAM format
- Import from images
- Collision data support

---

### 3. Standard Library & APIs

#### 3.1 System Library
**Purpose**: Common system functions

**Features**:
- **Memory Functions**: `memcpy`, `memset`, `memcmp`
- **String Functions**: `strlen`, `strcpy`, `strcmp`
- **Math Functions**: Fixed-point math (since we don't have FPU)
- **Delay Functions**: Frame-accurate delays
- **Input Functions**: Button state checking, input polling
- **PPU Functions**: Sprite setup, background setup, palette loading
- **APU Functions**: Note playing, sound effects

**Example**:
```assembly
.include "stdlib.inc"

main:
    ; Load palette
    CALL LoadPalette, #0, #palette_data
    
    ; Setup sprite
    CALL SetupSprite, #0, #100, #50, #tile_index, #palette_index
    
    ; Wait for button press
wait_input:
    CALL PollInput
    CMP R0, #BUTTON_A
    BNE wait_input
```

#### 3.2 Graphics Library
**Purpose**: High-level graphics functions

**Features**:
- **Sprite Management**: Easy sprite creation, movement, animation
- **Background Management**: Layer setup, scrolling
- **Tile Management**: Tile loading, tilemap rendering
- **Matrix Mode Helpers**: Transformation matrix setup
- **HDMA Helpers**: Per-scanline effects

#### 3.3 Audio Library
**Purpose**: High-level audio functions

**Features**:
- **Music Player**: Play songs, handle channels
- **Sound Effects**: Play SFX, manage priority
- **Note Helpers**: Musical note constants, frequency tables
- **Volume Control**: Fade in/out, master volume

---

### 4. Development Tools (In Emulator)

#### 4.1 Save States

> **⚠️ STATUS**: Save states are planned but not yet implemented. This feature will be added in a future release.

**Purpose**: Save/load emulator state for testing

**Features**:
- Save current emulator state to file
- Load saved state
- Multiple save slots
- Quick save/load (F5/F7)
- State browser (list all saves)

**Use Cases**:
- Test specific game states repeatedly
- Debug hard-to-reproduce bugs
- Share bug reports with exact state

#### 4.2 ROM Hot Reloading
**Purpose**: Reload ROM without restarting emulator

**Features**:
- Reload ROM while emulator is running
- Preserve debugger state (breakpoints, etc.)
- Fast iteration during development
- Keyboard shortcut (Ctrl+R to reload)

#### 4.3 Performance Profiler
**Purpose**: Identify performance bottlenecks

**Features**:
- Cycle counting per function
- Frame time breakdown
- Hot spot identification
- Performance graph over time
- CPU usage per component (CPU, PPU, APU)

#### 4.4 Memory Profiler
**Purpose**: Track memory usage

**Features**:
- Memory usage over time
- Identify memory leaks
- Track WRAM/VRAM usage
- Memory allocation tracking (if we add malloc-like functions)

---

### 5. Asset Editors (Standalone Tools)

#### 5.1 Tile Editor
**Purpose**: Create/edit individual tiles

**Features**:
- Pixel-level tile editing
- Palette selection
- Preview with different palettes
- Export single tile or tile set
- Import from images

#### 5.2 Sprite Editor
**Purpose**: Create/edit sprites

**Features**:
- Multi-tile sprite editing
- Animation support (frame editor)
- Collision box editing
- Export sprite data + OAM format
- Preview sprite in different palettes

#### 5.3 Palette Editor
**Purpose**: Create/edit color palettes

**Features**:
- Visual color picker
- RGB555 color editing
- Palette preview
- Import from images
- Export CGRAM format
- Color ramping tools

#### 5.4 Map Editor
**Purpose**: Create tilemaps visually

**Features**:
- Visual tilemap painting
- Multiple layers
- Tileset browser
- Collision layer
- Export to VRAM format
- Import/export to common formats

---

### 6. Documentation & Examples

#### 6.1 API Documentation
**Purpose**: Complete API reference

**Features**:
- Function reference (all standard library functions)
- Register reference (all I/O registers)
- Instruction set reference
- Memory map documentation
- Code examples for each function
- Searchable documentation

#### 6.2 Tutorial Series
**Purpose**: Learn game development step-by-step

**Topics**:
1. Getting Started (Hello World)
2. Displaying Sprites
3. Input Handling
4. Background Layers
5. Audio Basics
6. Animation
7. Game Loop Structure
8. Advanced Graphics (Matrix Mode)
9. Sound Effects & Music
10. Complete Game Example

#### 6.3 Sample Projects
**Purpose**: Working examples to learn from

**Projects**:
- **Hello World**: Basic ROM that displays text
- **Sprite Demo**: Moving sprites, animation
- **Platformer**: Simple platformer game
- **Music Player**: Audio demonstration
- **Matrix Mode Demo**: 3D effects showcase
- **Full Game**: Complete game (Pong, Breakout, etc.)

#### 6.4 Best Practices Guide
**Purpose**: Learn from common mistakes

**Topics**:
- Memory management
- Performance optimization
- Code organization
- Asset optimization
- Audio programming tips
- Graphics programming tips

---

### 7. Testing & Quality Assurance Tools

#### 7.1 ROM Test Suite
**Purpose**: Automated testing framework

**Features**:
- Unit test framework for ROM code
- Test ROM execution
- Assertion macros
- Test reporting
- Integration with CI/CD

**Example**:
```assembly
.test "Sprite rendering"
    CALL SetupSprite, #0, #100, #50, #0, #0
    CALL RenderFrame
    ASSERT SpriteVisible, #0
.endtest
```

#### 7.2 Performance Benchmarking
**Purpose**: Measure ROM performance

**Features**:
- Frame rate measurement
- Cycle counting
- Memory usage tracking
- Performance regression detection
- Benchmark reports

#### 7.3 Compatibility Checker
**Purpose**: Ensure ROM works across emulator versions

**Features**:
- Test ROM on different emulator builds
- Check for deprecated features
- Validate against specification
- Compatibility report

---

### 8. Distribution Tools

#### 8.1 ROM Packager
**Purpose**: Prepare ROM for distribution

**Features**:
- Add metadata (title, author, version)
- Optimize ROM size
- Generate checksum
- Create distribution package
- Add splash screen (optional)

#### 8.2 ROM Installer
**Purpose**: Install ROMs to emulator

**Features**:
- ROM library management
- Categorization (games, demos, tools)
- Screenshot generation
- Metadata editing
- ROM launcher integration

---

### 9. Modern Development Conveniences

#### 9.1 IDE Integration
**Purpose**: Better development experience

**Features**:
- **VS Code Extension**: Syntax highlighting, code completion, debugging
- **Language Server**: Go-to definition, find references, hover info
- **Build Integration**: One-click ROM building
- **Debug Integration**: Launch emulator with debugger attached

#### 9.2 Source Control Integration
**Purpose**: Version control friendly

**Features**:
- Text-based asset formats (JSON, YAML)
- Human-readable ROM format (optional)
- Diff-friendly formats
- Git hooks for validation

#### 9.3 Continuous Integration
**Purpose**: Automated testing

**Features**:
- GitHub Actions integration
- Automated ROM building
- Automated testing
- Performance regression detection
- Release automation

---

### 10. Community & Sharing Tools

#### 10.1 ROM Sharing Platform
**Purpose**: Share ROMs with community

**Features**:
- Upload/download ROMs
- Rating/review system
- Screenshot gallery
- Source code sharing (optional)
- Categorization and tags

#### 10.2 Asset Library
**Purpose**: Share reusable assets

**Features**:
- Tile sets
- Sprite sheets
- Palettes
- Audio samples
- Code snippets

#### 10.3 Documentation Wiki
**Purpose**: Community-maintained documentation

**Features**:
- Wiki for tips and tricks
- Community examples
- FAQ
- Troubleshooting guide

---

## Implementation Priority

### Phase 1: Essential Tools (MVP)
1. ✅ Enhanced ROM builder (macros, includes)
2. ✅ ROM validator
3. ✅ Image to tile converter
4. ✅ Basic standard library
5. ✅ Save states in emulator
6. ✅ Basic documentation

### Phase 2: Asset Pipeline
1. ✅ Palette converter
2. ✅ Sprite sheet packer
3. ✅ Audio converter
4. ✅ Tile editor (basic)

### Phase 3: Advanced Tools
1. ✅ Tilemap editor
2. ✅ Sprite editor
3. ✅ Performance profiler
4. ✅ Hot reloading

### Phase 4: Polish & Distribution
1. ✅ IDE integration
2. ✅ ROM packager
3. ✅ Complete documentation
4. ✅ Sample projects

---

## File Structure

```
nitro-core-dx-sdk/
├── tools/
│   ├── rom-builder/          # Enhanced ROM builder
│   ├── img2tiles/            # Image to tile converter
│   ├── palette-converter/    # Palette tools
│   ├── audio-converter/      # Audio tools
│   ├── tile-editor/          # Tile editor
│   ├── sprite-editor/        # Sprite editor
│   └── map-editor/           # Tilemap editor
├── lib/
│   ├── stdlib.inc            # Standard library
│   ├── graphics.inc          # Graphics library
│   └── audio.inc             # Audio library
├── examples/
│   ├── hello-world/
│   ├── sprite-demo/
│   ├── platformer/
│   └── full-game/
├── docs/
│   ├── api-reference/
│   ├── tutorials/
│   └── best-practices/
└── templates/
    └── project-template/     # Starter project template
```

---

## Success Criteria

- ✅ Developer can create a simple game in < 1 hour
- ✅ All assets can be created/imported with SDK tools
- ✅ Debugging is efficient and pleasant
- ✅ Documentation is complete and helpful
- ✅ Standard library covers common tasks
- ✅ Tools are easy to use and well-documented

---

## Modern vs. Retro Balance

**Retro (Authentic)**:
- Assembly language programming
- Manual memory management
- Fixed-point math
- Cycle-accurate timing
- Hardware limitations

**Modern (Convenient)**:
- Good tooling and error messages
- Asset pipeline automation
- Hot reloading during development
- Save states for testing
- IDE integration
- Version control friendly

**Philosophy**: Keep the authentic 1990s programming experience (assembly, manual control) but make it as pleasant as possible with modern tooling.

---

**End of SDK Features Plan**



