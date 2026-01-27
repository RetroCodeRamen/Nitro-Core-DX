# Development Tools Implementation Plan

**Created:** January 27, 2026  
**Status:** In Progress

## Overview

This document outlines the implementation plan for development tools and UI improvements for Nitro-Core-DX. The goal is to create a professional development environment with proper tooling.

---

## Current Issues

1. **UI Architecture**: Mixed rendering (some buttons rendered internally by emulator, some externally)
2. **Button Functionality**: Toolbar buttons exist but don't update state properly
3. **Debugging Tools**: Limited live debugging capabilities
4. **Sprite Editor**: No tool for creating/editing sprites
5. **ROM Testing**: Only simple box sprites available for testing

---

## Implementation Plan

### Phase 1: UI Consolidation (External Rendering Only) ✅ IN PROGRESS

**Goal**: All UI rendered externally using Fyne, nothing rendered by emulator internals

**Tasks**:
1. ✅ Make Fyne toolbar buttons functional with state updates
2. ⏳ Remove SDL2-based UI rendering (keep only for emulator screen)
3. ⏳ Ensure all panels are Fyne widgets
4. ⏳ Update menu items to toggle panels

**Files Modified**:
- `internal/ui/fyne_ui.go` - Main UI implementation
- `internal/ui/toolbar.go` - Toolbar (deprecated, using Fyne widgets)
- `internal/ui/menu.go` - Menu (deprecated, using Fyne menus)

**Status**: Toolbar buttons now update state correctly

---

### Phase 2: Debug Panels

#### 2.1 Register Viewer ✅ CREATED

**File**: `internal/ui/panels/register_viewer.go`

**Features**:
- Real-time CPU register display (R0-R7)
- Program Counter (PC) display (bank:offset)
- Stack Pointer (SP) display
- Bank registers (PBR, DBR)
- Flags register (Z, N, C, V, I, D)
- Updates at 60 FPS

**Status**: Panel created, needs integration into FyneUI

#### 2.2 Memory Viewer ⏳ PLANNED

**File**: `internal/ui/panels/memory_viewer.go`

**Features**:
- Hex dump view of memory
- Bank selector (0-255)
- Offset selector (0x0000-0xFFFF)
- Real-time updates
- Search functionality
- Bookmark addresses

#### 2.3 Tile Viewer ✅ CREATED

**File**: `internal/ui/panels/tile_viewer.go`

**Features**:
- ✅ Visual grid of tiles from VRAM
- ✅ Palette selector (0-15)
- ✅ Tile size selector (8x8 or 16x16)
- ✅ Tile offset selector (start tile)
- ✅ Grid size selector (tiles per row: 8, 16, 32)
- ✅ Real-time updates as VRAM/CGRAM changes
- ⏳ Click to select tile (future enhancement)
- ⏳ Export tile as image (future enhancement)

---

### Phase 3: Sprite Editor Tool

#### 3.1 Basic Sprite Editor ✅ CREATED

**File**: `cmd/sprite_editor/main.go`

**Features**:
- Pixel-level editing (8x8 or 16x16 tiles)
- Palette selector (16 colors)
- Clear/Export/Import buttons
- Grid display

**Status**: Basic structure created, needs:
- Mouse click handling for pixel editing
- Color picker
- Export to VRAM format
- Import from images

#### 3.2 Enhanced Sprite Editor ⏳ PLANNED

**Features**:
- Multi-tile sprite editing
- Animation support
- Sprite sheet management
- Export to ROM format
- Preview with different palettes

---

### Phase 4: Better Test ROMs

#### 4.1 Animated Sprite ROM ⏳ PLANNED

**File**: `test/roms/build_animated_sprite.go`

**Features**:
- Multiple animation frames
- Sprite movement
- Collision detection
- Sound effects

#### 4.2 Character Sprite ROM ⏳ PLANNED

**File**: `test/roms/build_character_sprite.go`

**Features**:
- Character sprite (not just a box)
- Walking animation
- Multiple directions
- Background scrolling

---

### Phase 5: NitroLang Language & Compiler 🚀 NEW PRIORITY

**Goal**: Create a custom compiled language with Lua-like syntax that compiles to Nitro-Core-DX bytecode, making development a dream.

**Design Document**: See `docs/LANGUAGE_DESIGN.md`

**Note**: This is a COMPILED language (not interpreted/scripted). It uses Lua-like syntax for readability but compiles to efficient bytecode.

#### 5.1 Language Design ✅ COMPLETED

**File**: `docs/LANGUAGE_DESIGN.md`

**Features**:
- ✅ Lua-like syntax (familiar, clean, readable)
- ✅ Dynamic typing (no type declarations needed)
- ✅ Compiled to bytecode (NOT interpreted - true compilation)
- ✅ Assembly integration (direct CPU access)
- ✅ Standard library wrappers (PPU, memory, input, audio)

**Status**: Language design document complete

**Note**: This is a COMPILED language, not a scripting language. It uses Lua-like syntax for readability but compiles to efficient bytecode.

#### 5.2 Lexer Implementation ⏳ PLANNED

**File**: `cmd/nitrolang/lexer.go`

**Features**:
- Tokenize NitroScript source code
- Handle comments (`--` style)
- Parse strings, numbers, keywords
- Support both Lua-style and assembly syntax
- Error reporting with line numbers

**Tasks**:
1. Define token types
2. Implement tokenizer
3. Handle string escaping
4. Handle number parsing (decimal, hex, binary)
5. Handle operator tokens

#### 5.3 Parser Implementation ⏳ PLANNED

**File**: `cmd/nitrolang/parser.go`

**Features**:
- Build AST (Abstract Syntax Tree)
- Parse Lua-like syntax
- Parse inline assembly blocks (`asm { ... }`)
- Handle type hints (optional)
- Variable scope resolution
- Function parsing

**Tasks**:
1. Expression parsing
2. Statement parsing
3. Function parsing
4. Control flow parsing (if/else, while, for)
5. Table/object parsing
6. Assembly block parsing

#### 5.4 Code Generator ⏳ PLANNED

**File**: `cmd/nitrolang/codegen.go`

**Features**:
- Generate Nitro-Core-DX bytecode
- Register allocation
- Function call code generation
- Control flow code generation
- Assembly block integration
- Standard library function calls

**Tasks**:
1. Basic expression code generation
2. Variable assignment code generation
3. Function call code generation
4. Control flow code generation
5. Register allocation
6. Memory access code generation

#### 5.5 Standard Library ⏳ PLANNED

**File**: `cmd/nitrolang/stdlib.nl`

**Features**:
- PPU wrapper functions (`ppu.set_sprite_pos`, etc.)
- Memory access functions (`mem.read8`, `mem.write8`, etc.)
- Input functions (`input.pressed`, `input.update`, etc.)
- Audio functions (`audio.play_sound`, etc.)
- Background functions (`bg.set_scroll`, etc.)
- Utility functions (`math`, `string`, etc.)

**Tasks**:
1. PPU function wrappers
2. Memory access wrappers
3. Input wrappers
4. Audio wrappers
5. Math utilities
6. String utilities

#### 5.6 Assembly Integration ⏳ PLANNED

**File**: `cmd/nitrolang/asm.go`

**Features**:
- Parse inline assembly blocks
- Variable substitution in assembly
- Register access from high-level code
- Memory access from high-level code
- Assembly function definitions

**Tasks**:
1. Assembly block parser
2. Variable substitution
3. Register access API
4. Memory access API
5. Assembly function integration

#### 5.7 Optimizer ⏳ PLANNED

**File**: `cmd/nitrolang/optimizer.go`

**Features**:
- Dead code elimination
- Constant folding
- Register allocation optimization
- Function inlining (optional)
- Loop optimization (optional)

**Tasks**:
1. Dead code elimination
2. Constant folding
3. Register allocation optimization
4. Function inlining
5. Loop unrolling (optional)

#### 5.8 Build Tools ⏳ PLANNED

**File**: `cmd/nitrolang/main.go`

**Features**:
- Command-line compiler (`nitrolang build`)
- Error reporting with source locations
- Debug symbol generation
- ROM output generation
- Watch mode (auto-rebuild on changes)

**Usage**:
```bash
# Compile NitroLang to ROM
nitrolang build sprite_demo.nl -o sprite_demo.rom

# Debug mode (include debug symbols)
nitrolang build sprite_demo.nl -o sprite_demo.rom --debug

# Watch mode (auto-rebuild)
nitrolang watch sprite_demo.nl -o sprite_demo.rom
```

**Tasks**:
1. Command-line interface
2. File I/O
3. Error reporting
4. Debug symbol generation
5. Watch mode implementation

#### 5.9 IDE Integration ⏳ PLANNED

**File**: `cmd/nitrolang/lsp/` (Language Server Protocol)

**Features**:
- Syntax highlighting
- Auto-completion
- Error checking
- Go-to definition
- Hover documentation
- VS Code extension

**Tasks**:
1. LSP server implementation
2. VS Code extension
3. Syntax highlighting
4. Auto-completion
5. Error checking
6. Documentation generation

---

## File Structure

```
internal/ui/
├── fyne_ui.go              # Main Fyne UI (external rendering)
├── panels/
│   ├── register_viewer.go  # ✅ CPU register display
│   ├── memory_viewer.go    # ⏳ Memory hex dump
│   ├── tile_viewer.go      # ✅ VRAM tile viewer
│   ├── log_viewer.go       # ✅ Log viewer (existing)
│   └── log_controls.go     # ✅ Log controls (existing)

cmd/
├── emulator/
│   └── main.go            # Main emulator (uses FyneUI)
├── sprite_editor/
│   └── main.go            # ✅ Sprite editor tool
└── nitrolang/             # 🚀 NEW: NitroLang compiler
    ├── main.go            # ⏳ CLI entry point
    ├── lexer.go           # ⏳ Tokenizer
    ├── parser.go          # ⏳ AST generator
    ├── codegen.go         # ⏳ Bytecode generator
    ├── optimizer.go       # ⏳ Code optimizer
    ├── asm.go             # ⏳ Assembly integration
    └── lsp/               # ⏳ Language Server Protocol
        └── server.go      # ⏳ LSP server

docs/
└── LANGUAGE_DESIGN.md     # ✅ NitroLang language design

test/roms/
├── build_animated_sprite.go # ⏳ Animated sprite ROM builder
└── build_character_sprite.go # ⏳ Character sprite ROM builder
```

---

## Next Steps

### Immediate Priority: Finish Processor/PPU Features First ⚠️

**User Request**: Complete processor/PPU features and design before starting language work.

**Current Status**:
- ✅ Timing synchronized (CPU/PPU unified clock at ~7.67 MHz)
- ✅ Frame timing fixed (127,820 cycles/frame)
- ⏳ Performance optimization (batch stepping implemented)
- ⏳ Verify 60 FPS achieved

### Secondary Priority: NitroLang Language 🚀

1. **Implement Lexer** - Tokenize NitroLang source code
2. **Implement Parser** - Build AST from tokens
3. **Implement Code Generator** - Generate Nitro-Core-DX bytecode
4. **Create Standard Library** - PPU, memory, input wrappers
5. **Implement Assembly Integration** - Inline assembly support

### Secondary Priority: Development Tools

1. **Integrate Register Viewer** into FyneUI
2. **Create Memory Viewer** panel
3. **Complete Sprite Editor** (pixel editing, export)
4. **Create better test ROMs** (animated sprites, characters)

---

## UI Architecture

### External Rendering (Fyne)
- ✅ Menu bar (Fyne native menus)
- ✅ Toolbar buttons (Fyne widgets)
- ✅ Status bar (Fyne label)
- ✅ Debug panels (Fyne containers)
- ✅ Emulator screen (Fyne canvas with SDL2 rendering)

### Internal Rendering (SDL2)
- ✅ Emulator output buffer (320x200 pixels)
- ❌ NO UI buttons or menus rendered by SDL2
- ❌ NO UI elements rendered by emulator internals

---

**End of Plan**
