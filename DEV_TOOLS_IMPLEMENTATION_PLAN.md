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

#### 2.3 Tile Viewer ⏳ PLANNED

**File**: `internal/ui/panels/tile_viewer.go`

**Features**:
- Visual grid of tiles from VRAM
- Palette selector
- Tile size selector (8x8 or 16x16)
- Click to select tile
- Export tile as image
- Real-time updates as VRAM changes

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

## File Structure

```
internal/ui/
├── fyne_ui.go              # Main Fyne UI (external rendering)
├── panels/
│   ├── register_viewer.go  # ✅ CPU register display
│   ├── memory_viewer.go    # ⏳ Memory hex dump
│   ├── tile_viewer.go      # ⏳ VRAM tile viewer
│   ├── log_viewer.go       # ✅ Log viewer (existing)
│   └── log_controls.go     # ✅ Log controls (existing)

cmd/
├── emulator/
│   └── main.go            # Main emulator (uses FyneUI)
└── sprite_editor/
    └── main.go            # ✅ Sprite editor tool

test/roms/
├── build_animated_sprite.go # ⏳ Animated sprite ROM builder
└── build_character_sprite.go # ⏳ Character sprite ROM builder
```

---

## Next Steps

1. **Integrate Register Viewer** into FyneUI
2. **Create Memory Viewer** panel
3. **Create Tile Viewer** panel
4. **Complete Sprite Editor** (pixel editing, export)
5. **Create better test ROMs** (animated sprites, characters)

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
