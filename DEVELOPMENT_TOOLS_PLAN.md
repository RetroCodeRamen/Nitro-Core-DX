# Development Tools & UI Enhancement Plan

**Created: January 6, 2026**

## Overview

This document outlines the plan for implementing comprehensive development tools and UI enhancements for the Nitro-Core-DX emulator. The goal is to create a professional debugging environment that makes ROM development efficient and enjoyable.

---

## Current State Analysis

### What We Have
- ✅ Basic SDL2 UI with emulator screen rendering
- ✅ Audio output via SDL
- ✅ Basic input handling
- ✅ Frame limiting and performance tracking
- ✅ Scattered `fmt.Printf` logging throughout codebase

### What We Need
- ❌ Centralized logging system with UI integration
- ❌ Real-time register viewer
- ❌ Real-time memory viewer/hex editor
- ❌ Visual tile map viewer
- ❌ Proper start/stop/reset with full memory clearing
- ❌ Component-based logging toggles
- ❌ Better UI organization (panels, dockable windows)

---

## Phase 1: Foundation - Logging System & UI Structure

### 1.1 Centralized Logging System

**Goal**: Replace scattered `fmt.Printf` with a structured logging system that can be displayed in the UI.

**Components**:
- Create `internal/debug/logger.go` with structured logging
- Log levels: None, Error, Warning, Info, Debug, Trace
- Component-based logging (CPU, PPU, APU, Memory, Input)
- Log entries include: timestamp, component, level, message, optional data
- Thread-safe logging (goroutine-safe)
- Circular buffer for log history (configurable size, e.g., 10,000 entries)

**Log Entry Structure**:
```go
type LogEntry struct {
    Timestamp time.Time
    Component string  // "CPU", "PPU", "APU", "Memory", "Input"
    Level     LogLevel
    Message   string
    Data      map[string]interface{} // Optional structured data
}
```

**Implementation**:
- Remove all `fmt.Printf` calls from core emulator code
- Replace with `logger.Log(component, level, message, data)`
- Add component-specific helpers: `logger.LogCPU()`, `logger.LogPPU()`, etc.
- Zero-cost when logging is disabled (use build tags or runtime checks)

### 1.2 UI Structure Improvements

**Goal**: Create a proper windowed UI with panels and dockable windows.

**Components**:
- Main window with menu bar (File, Emulation, View, Debug, Help)
- Toolbar with quick actions (Play, Pause, Stop, Reset, Step Frame)
- Status bar (FPS, cycles, frame time)
- Dockable panels system (using SDL2 or ImGui if we add it)
- Panel management (show/hide, resize, dock/undock)

**Window Layout**:
```
┌─────────────────────────────────────────────────────────┐
│ File  Emulation  View  Debug  Help                      │
├─────────────────────────────────────────────────────────┤
│ [▶] [⏸] [⏹] [↻] [⏭]  [Settings]                        │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────┐  ┌────────────────────────────────┐ │
│  │              │  │                                │ │
│  │  Emulator    │  │  Debug Panels (dockable)      │ │
│  │  Screen      │  │  - Registers                   │ │
│  │  (320x200)   │  │  - Memory Viewer               │ │
│  │              │  │  - Log Viewer                  │ │
│  │              │  │  - Tile Viewer                │ │
│  └──────────────┘  └────────────────────────────────┘ │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ FPS: 60.0 | Cycles: 166,667 | Frame: 1,234            │
└─────────────────────────────────────────────────────────┘
```

**Implementation Options**:
- **Option A**: Pure SDL2 with custom UI (more control, more work)
- **Option B**: Add ImGui (Dear ImGui) via Go bindings (faster development, professional UI)
- **Recommendation**: Start with Option A (SDL2) for simplicity, consider ImGui later if needed

---

## Phase 2: Emulation Control

### 2.1 Start/Stop/Reset with Full Memory Clearing

**Goal**: Proper emulation control that mimics real hardware power-on/off.

**Components**:
- **Start**: Begin emulation from current state
- **Stop**: Halt emulation and clear ALL memory (WRAM, VRAM, CGRAM, OAM, registers)
- **Reset**: Reset CPU state, clear memory, reload ROM entry point
- **Pause/Resume**: Toggle emulation without clearing state

**Implementation**:
- Add `FullReset()` method to `Emulator` that:
  - Clears all WRAM (bank 0, extended WRAM)
  - Clears all VRAM
  - Clears all CGRAM (palette)
  - Clears all OAM (sprites)
  - Resets CPU state (registers, flags, PC, SP)
  - Resets PPU state (scroll, layers, etc.)
  - Resets APU state (channels, volume)
  - Reloads ROM entry point
- Add UI buttons for Start/Stop/Reset
- Add keyboard shortcuts (Space = Pause, Ctrl+R = Reset, Ctrl+S = Stop)

### 2.2 Frame Stepping

**Goal**: Step through emulation one frame at a time for debugging.

**Components**:
- **Step Frame**: Execute exactly one frame, then pause
- **Step Instruction**: Execute one CPU instruction, then pause (advanced)
- UI button for "Step Frame"
- Keyboard shortcut (F5 = Step Frame)

**Implementation**:
- Add `StepFrame()` method to `Emulator`
- Add `stepping` flag to control step mode
- When stepping, execute one frame then set `Paused = true`

---

## Phase 3: Real-Time Debugging Tools

### 3.1 Register Viewer

**Goal**: Display all CPU registers in real-time with updates.

**Components**:
- Panel showing:
  - General Purpose Registers: R0-R7 (hex and decimal)
  - Program Counter: Bank:Offset (hex)
  - Stack Pointer: SP (hex)
  - Bank Registers: PBR, DBR (hex)
  - Flags: Z, N, C, V, I (checkboxes or colored indicators)
  - Cycle Counter: Total cycles (decimal)
- Auto-refresh at frame rate (60 FPS)
- Highlight registers that changed since last frame
- Optional: Edit registers (advanced feature)

**UI Layout**:
```
┌─ CPU Registers ─────────────────┐
│ R0: 0x0000 (0)                  │
│ R1: 0x1234 (4660)                │
│ R2: 0x5678 (22136)               │
│ ...                              │
│ PC: 01:8000                      │
│ SP: 0x7FFF                       │
│ PBR: 0x01  DBR: 0x00            │
│ Flags: [Z] [N] [ ] [ ] [I]      │
│ Cycles: 1,234,567                │
└──────────────────────────────────┘
```

### 3.2 Memory Viewer / Hex Editor

**Goal**: View and edit memory in real-time.

**Components**:
- Hex editor panel showing:
  - Address range selector (bank + offset)
  - Hex dump view (address | hex bytes | ASCII)
  - Scrollable view
  - Search functionality
  - Bookmark addresses
  - Memory regions highlighted (WRAM, VRAM, CGRAM, OAM, ROM, I/O)
- Real-time updates (refresh at frame rate)
- Optional: Edit memory values (advanced feature)
- Memory map view (visual representation of memory regions)

**UI Layout**:
```
┌─ Memory Viewer ─────────────────┐
│ Bank: [01] Offset: [8000]       │
│                                  │
│ 0x018000: 12 34 56 78 9A BC ... │
│ 0x018010: DE F0 12 34 56 78 ... │
│ ...                              │
│                                  │
│ [Search] [Bookmark] [Edit]       │
└──────────────────────────────────┘
```

### 3.3 Log Viewer

**Goal**: Display logs in the UI with filtering and search.

**Components**:
- Log panel showing:
  - Scrollable log entries
  - Color-coded by component (CPU=blue, PPU=green, APU=yellow, etc.)
  - Filter by component (checkboxes)
  - Filter by log level (dropdown)
  - Search functionality
  - Auto-scroll toggle
  - Clear log button
  - Export to file button
- Real-time updates (new logs appear as they're generated)
- Virtual scrolling for performance (only render visible entries)

**UI Layout**:
```
┌─ Log Viewer ────────────────────┐
│ [CPU] [PPU] [APU] [Memory] [All]│
│ Level: [All ▼] [Search: ____]    │
│                                  │
│ [12:34:56] [CPU] PC: 01:8000    │
│ [12:34:56] [PPU] VRAM write...   │
│ [12:34:56] [APU] Channel 0...    │
│ ...                              │
│                                  │
│ [Clear] [Export] [Auto-scroll]  │
└──────────────────────────────────┘
```

---

## Phase 4: Visual Debugging Tools

### 4.1 Tile Map Viewer

**Goal**: Visual representation of all tiles in VRAM, not just hex data.

**Components**:
- Tile viewer panel showing:
  - Grid of tiles (e.g., 16x16 tiles per row)
  - Each tile rendered as 8x8 or 16x16 pixel image
  - Tile index displayed on hover
  - Click to select tile (shows details)
  - Palette selector (view tiles with different palettes)
  - Scrollable view (for large VRAM)
  - Real-time updates (tiles update as VRAM changes)
- Tile details panel:
  - Selected tile index
  - Tile data (hex dump)
  - Palette used
  - Color indices
  - Export tile as image

**UI Layout**:
```
┌─ Tile Viewer ───────────────────┐
│ Palette: [0 ▼] Size: [8x8 ▼]     │
│                                  │
│ [Tile 0] [Tile 1] [Tile 2] ...  │
│ [Tile 16][Tile 17][Tile 18] ... │
│ ...                              │
│                                  │
│ Selected: Tile 42                │
│ Data: 12 34 56 78 ...            │
│ [Export]                          │
└──────────────────────────────────┘
```

**Implementation Details**:
- Render tiles by reading VRAM tile data
- Use current CGRAM palette (or selected palette)
- Convert 4bpp tile data to RGB pixels
- Cache rendered tiles for performance (invalidate on VRAM/CGRAM changes)

### 4.2 Sprite Viewer

**Goal**: Visual representation of all sprites with their attributes.

**Components**:
- Sprite viewer panel showing:
  - List of all 128 sprites
  - Each sprite rendered as thumbnail
  - Sprite attributes (X, Y, tile, palette, enabled, size)
  - Highlight active sprites (enabled)
  - Filter by enabled/disabled
  - Click to select sprite (shows details)
- Real-time updates (sprites update as OAM changes)

**UI Layout**:
```
┌─ Sprite Viewer ──────────────────┐
│ [Show Enabled Only]              │
│                                  │
│ Sprite 0: [8x8] X:100 Y:50      │
│ [Thumbnail] Tile:42 Palette:1   │
│                                  │
│ Sprite 1: [16x16] X:200 Y:100   │
│ [Thumbnail] Tile:43 Palette:2   │
│ ...                              │
└──────────────────────────────────┘
```

### 4.3 Palette Viewer

**Goal**: Visual representation of CGRAM palettes.

**Components**:
- Palette viewer panel showing:
  - All 16 palettes (16 colors each)
  - Color swatches for each palette
  - RGB values displayed
  - Click to edit color (advanced feature)
- Real-time updates (palettes update as CGRAM changes)

**UI Layout**:
```
┌─ Palette Viewer ─────────────────┐
│ Palette 0:                       │
│ [Color 0][Color 1][Color 2]...   │
│                                  │
│ Palette 1:                       │
│ [Color 0][Color 1][Color 2]...   │
│ ...                              │
└──────────────────────────────────┘
```

### 4.4 Layer Viewer

**Goal**: Toggle individual background layers for debugging.

**Components**:
- Layer control panel:
  - Checkboxes for BG0, BG1, BG2, BG3
  - Checkbox for Sprites
  - Real-time layer toggling (affects main screen)
- Visual feedback (disabled layers show as black or checkerboard)

---

## Phase 5: Advanced Features (Future)

### 5.1 Breakpoints & Watchpoints

**Goal**: Pause emulation at specific conditions.

**Components**:
- Address breakpoints (pause when PC reaches address)
- Conditional breakpoints (pause when condition is met)
- Watchpoints (pause when memory address is read/written)
- Breakpoint management panel

### 5.2 Instruction Tracer

**Goal**: Step through CPU instructions one at a time.

**Components**:
- Current instruction display (disassembly)
- Step forward/backward through instructions
- Instruction history (last N instructions)
- Call stack viewer

### 5.3 Performance Profiler

**Goal**: Identify performance bottlenecks.

**Components**:
- Cycle counting per component
- Frame time breakdown
- Hot spot identification
- Performance graph

---

## Implementation Plan

### Phase 1: Foundation (Week 1-2)
1. ✅ Create centralized logging system
2. ✅ Remove all `fmt.Printf` calls
3. ✅ Create basic UI structure (menu bar, toolbar, status bar)
4. ✅ Implement log viewer panel
5. ✅ Add component-based logging toggles

### Phase 2: Emulation Control (Week 2-3)
1. ✅ Implement full memory clearing on stop
2. ✅ Add Start/Stop/Reset buttons
3. ✅ Implement frame stepping
4. ✅ Add keyboard shortcuts

### Phase 3: Real-Time Debugging (Week 3-4)
1. ✅ Implement register viewer
2. ✅ Implement memory viewer/hex editor
3. ✅ Integrate with logging system

### Phase 4: Visual Tools (Week 4-5)
1. ✅ Implement tile map viewer
2. ✅ Implement sprite viewer
3. ✅ Implement palette viewer
4. ✅ Implement layer viewer

### Phase 5: Polish (Week 5-6)
1. ✅ UI polish and styling
2. ✅ Performance optimization
3. ✅ Documentation
4. ✅ Testing

---

## Technical Considerations

### UI Library Choice

**Option A: Pure SDL2**
- Pros: No external dependencies, full control, lightweight
- Cons: More code to write, manual UI rendering

**Option B: ImGui (Dear ImGui)**
- Pros: Professional UI, fast development, many widgets
- Cons: External dependency, C++ bindings needed

**Recommendation**: Start with SDL2 for core UI, consider ImGui for debug panels if needed.

### Performance

- Use virtual scrolling for large lists (log viewer, memory viewer)
- Cache rendered tiles/sprites (invalidate on data changes)
- Update panels at frame rate (60 FPS) or lower (30 FPS for non-critical panels)
- Use goroutines for log collection (non-blocking)

### Thread Safety

- Logging system must be thread-safe (use channels or mutexes)
- UI updates from main thread only (SDL requirement)
- Use channels to pass data from emulator to UI

---

## File Structure

```
internal/
├── debug/
│   ├── logger.go          # Centralized logging system
│   ├── log_entry.go       # Log entry structure
│   └── log_filter.go      # Log filtering logic
├── ui/
│   ├── ui.go              # Main UI structure
│   ├── menu.go            # Menu bar
│   ├── toolbar.go         # Toolbar
│   ├── statusbar.go       # Status bar
│   ├── panels/
│   │   ├── register_viewer.go
│   │   ├── memory_viewer.go
│   │   ├── log_viewer.go
│   │   ├── tile_viewer.go
│   │   ├── sprite_viewer.go
│   │   ├── palette_viewer.go
│   │   └── layer_viewer.go
│   └── render_fixed.go    # Existing rendering code
└── emulator/
    └── emulator.go        # Add FullReset(), StepFrame()
```

---

## Success Criteria

- ✅ All `fmt.Printf` calls removed from core emulator code
- ✅ Logging system with UI integration working
- ✅ Register viewer showing real-time CPU state
- ✅ Memory viewer showing real-time memory contents
- ✅ Tile viewer showing visual representation of tiles
- ✅ Start/Stop/Reset working with full memory clearing
- ✅ Component-based logging toggles working
- ✅ UI is responsive and professional-looking

---

## Phase 6: SDK Build Tools

### 6.1 Enhanced ROM Builder

**Goal**: Professional assembler with modern features while maintaining retro authenticity.

**Components**:
- **Macro Support**: Define reusable code blocks
  ```assembly
  .macro SET_PALETTE_COLOR palette, index, r, g, b
      MOV R7, #0x8012        ; CGRAM_ADDR
      MOV R0, #(palette * 16 + index)
      MOV [R7], R0
      ; ... set color ...
  .endmacro
  ```
- **Include Files**: Modular code organization
  ```assembly
  .include "macros.inc"
  .include "constants.inc"
  ```
- **Labels & Symbols**: Better symbol management with error checking
- **Linker**: Support for multiple source files
- **Preprocessor**: Conditional compilation, defines
- **Error Messages**: Clear, helpful errors with line numbers and context
- **Warnings**: Non-fatal issues (unused labels, potential bugs)

**Implementation**:
- Enhance existing `cmd/rombuilder` or create new `cmd/assembler`
- Parse assembly syntax
- Two-pass assembly (first pass: collect labels, second pass: resolve)
- Symbol table generation
- Export symbol table for debugging

### 6.2 ROM Validator

**Goal**: Validate ROMs before distribution to catch errors early.

**Components**:
- Check ROM header validity (magic, version, size)
- Verify entry point is valid and reachable
- Check for common errors:
  - Unaligned jumps (PC must be even)
  - Invalid memory addresses
  - Unreachable code
  - Missing entry point
- Size validation (warn if ROM is too large)
- Checksum calculation
- ROM info display (size, entry point, mapper flags, symbol count)

**UI Integration**:
- Add "Validate ROM" option to File menu
- Display validation results in log viewer
- Highlight errors/warnings
- Export validation report

**Usage**:
```bash
nitro-rom-validator game.rom
# Output:
# ✓ ROM header valid
# ✓ Entry point: 01:8000
# ✓ ROM size: 64KB
# ⚠ Warning: Unused memory at end of ROM
# ⚠ Warning: Unreachable code at 01:8500
```

### 6.3 ROM Analyzer

**Goal**: Analyze ROM contents for debugging and optimization.

**Components**:
- Disassemble ROM code (instruction-by-instruction)
- List all symbols/labels with addresses
- Show memory usage breakdown (code, data, unused)
- Identify unused code/data
- Export symbol table (for debugger integration)
- Find all jump/call targets
- Code flow analysis
- Export disassembly to file

**UI Integration**:
- Add "Analyze ROM" option to Debug menu
- Display analysis results in new panel
- Click symbols to jump to address in memory viewer
- Export analysis report

### 6.4 Asset Pipeline Tools

**Goal**: Convert modern assets (images, audio) to console format.

#### 6.4.1 Image to Tile Converter
**Purpose**: Convert PNG/other images to VRAM tile format

**Features**:
- Convert image to 4bpp tiles (8x8 or 16x16)
- Automatic palette extraction (quantize to 16 colors per palette)
- Tile optimization (detect and merge duplicate tiles)
- Export to VRAM binary format
- Preview tiles with palette
- Batch processing (multiple images)

**UI Integration**:
- Add "Tools → Convert Image to Tiles" menu option
- File dialog for image selection
- Preview panel showing tiles
- Export options dialog

**Usage**:
```bash
nitro-img2tiles sprite.png --size 8x8 --palette auto --output tiles.bin
# Creates: tiles.bin, palette.bin, tilemap.bin
```

#### 6.4.2 Palette Converter
**Purpose**: Convert RGB images to console palette format

**Features**:
- RGB888 → RGB555 conversion
- Palette optimization (reduce to 16 colors per palette)
- Dithering options (Floyd-Steinberg, ordered)
- Preview with console palette
- Export CGRAM format
- Import existing palettes

**UI Integration**:
- Add "Tools → Convert Palette" menu option
- Visual palette editor
- Real-time preview

#### 6.4.3 Sprite Sheet Packer
**Purpose**: Organize sprites into efficient VRAM layout

**Features**:
- Pack multiple sprite images into VRAM
- Generate OAM data automatically
- Optimize tile usage (detect shared tiles)
- Export sprite metadata (X, Y, tile index, palette)
- Preview sprite sheet
- Animation frame support

**UI Integration**:
- Add "Tools → Pack Sprite Sheet" menu option
- Drag-and-drop sprite images
- Visual sprite sheet preview
- OAM data export

#### 6.4.4 Audio Converter
**Purpose**: Convert audio files to APU format

**Features**:
- WAV/MP3 → Frequency/note conversion
- Generate musical scale data
- Export frequency tables
- Preview audio with console limitations
- Generate APU initialization code
- Note-to-frequency mapping

**UI Integration**:
- Add "Tools → Convert Audio" menu option
- Audio waveform preview
- Frequency table display
- Code generation

### 6.5 Standard Library

**Goal**: Provide common functions to make development easier.

**Components**:
- **System Library** (`stdlib.inc`):
  - Memory functions: `memcpy`, `memset`, `memcmp`
  - String functions: `strlen`, `strcpy`, `strcmp`
  - Math functions: Fixed-point math (add, sub, mul, div)
  - Delay functions: Frame-accurate delays
- **Graphics Library** (`graphics.inc`):
  - Sprite management: `SetupSprite`, `MoveSprite`, `AnimateSprite`
  - Background management: `SetupBackground`, `ScrollBackground`
  - Tile management: `LoadTiles`, `RenderTilemap`
  - Matrix Mode helpers: `SetupMatrix`, `TransformPoint`
- **Audio Library** (`audio.inc`):
  - Music player: `PlaySong`, `StopSong`, `PauseSong`
  - Sound effects: `PlaySFX`, `StopSFX`
  - Note helpers: Musical note constants, frequency tables
  - Volume control: `FadeIn`, `FadeOut`, `SetMasterVolume`
- **Input Library** (`input.inc`):
  - Button polling: `PollButtons`, `IsButtonPressed`, `IsButtonHeld`
  - Input state: `GetInputState`

**UI Integration**:
- Add "Help → Standard Library Reference" menu option
- Display library documentation in help panel
- Code completion hints (if we add IDE integration)

**Example Usage**:
```assembly
.include "stdlib.inc"
.include "graphics.inc"
.include "audio.inc"

main:
    ; Load palette
    CALL LoadPalette, #0, #palette_data
    
    ; Setup sprite
    CALL SetupSprite, #0, #100, #50, #tile_index, #palette_index
    
    ; Play sound effect
    CALL PlaySFX, #SFX_JUMP
    
    ; Wait for button press
wait_input:
    CALL PollButtons
    CMP R0, #BUTTON_A
    BNE wait_input
```

### 6.6 Asset Editors (Standalone Tools)

**Goal**: Visual tools for creating game assets.

#### 6.6.1 Tile Editor
**Purpose**: Create/edit individual tiles

**Features**:
- Pixel-level tile editing (8x8 or 16x16)
- Palette selection and preview
- Export single tile or tile set
- Import from images
- Tile animation support (frame editor)

**Integration**: Can be launched from emulator "Tools" menu or standalone

#### 6.6.2 Sprite Editor
**Purpose**: Create/edit sprites

**Features**:
- Multi-tile sprite editing
- Animation support (frame editor, timeline)
- Collision box editing
- Export sprite data + OAM format
- Preview sprite in different palettes
- Sprite sheet management

**Integration**: Can be launched from emulator "Tools" menu or standalone

#### 6.6.3 Palette Editor
**Purpose**: Create/edit color palettes

**Features**:
- Visual color picker (RGB555)
- Palette preview with tiles
- Import from images
- Export CGRAM format
- Color ramping tools
- Palette animation (color cycling)

**Integration**: Integrated into emulator as a debug panel (see Phase 4.3)

#### 6.6.4 Map Editor
**Purpose**: Create tilemaps visually

**Features**:
- Visual tilemap painting
- Multiple layers (BG0-BG3)
- Tileset browser
- Collision layer editing
- Export to VRAM format
- Import/export to common formats (Tiled compatibility)

**Integration**: Standalone tool, can export directly to ROM

---

## Updated Implementation Plan

### Phase 1: Foundation (Week 1-2)
1. ✅ Create centralized logging system
2. ✅ Remove all `fmt.Printf` calls
3. ✅ Create basic UI structure (menu bar, toolbar, status bar)
4. ✅ Implement log viewer panel
5. ✅ Add component-based logging toggles

### Phase 2: Emulation Control (Week 2-3)
1. ✅ Implement full memory clearing on stop
2. ✅ Add Start/Stop/Reset buttons
3. ✅ Implement frame stepping
4. ✅ Add keyboard shortcuts

### Phase 3: Real-Time Debugging (Week 3-4)
1. ✅ Implement register viewer
2. ✅ Implement memory viewer/hex editor
3. ✅ Integrate with logging system

### Phase 4: Visual Tools (Week 4-5)
1. ✅ Implement tile map viewer
2. ✅ Implement sprite viewer
3. ✅ Implement palette viewer
4. ✅ Implement layer viewer

### Phase 5: SDK Build Tools - Core (Week 5-6)
1. ✅ Enhance ROM builder (macros, includes, better errors)
2. ✅ Implement ROM validator
3. ✅ Implement ROM analyzer
4. ✅ Create basic standard library (stdlib.inc)

### Phase 6: SDK Build Tools - Asset Pipeline (Week 6-7)
1. ✅ Implement image to tile converter
2. ✅ Implement palette converter
3. ✅ Implement sprite sheet packer
4. ✅ Implement audio converter

### Phase 7: SDK Build Tools - Editors (Week 7-8)
1. ✅ Implement tile editor (basic)
2. ✅ Implement sprite editor (basic)
3. ✅ Implement map editor (basic)

### Phase 8: Polish & Integration (Week 8-9)
1. ✅ UI polish and styling
2. ✅ Performance optimization
3. ✅ Complete documentation
4. ✅ Create sample projects
5. ✅ Testing and bug fixes

---

## SDK Tools UI Integration

### Menu Structure

```
File
  ├─ Load ROM...
  ├─ Reload ROM (Ctrl+R)
  ├─ Validate ROM...
  ├─ Analyze ROM...
  └─ Exit

Emulation
  ├─ Start (F5)
  ├─ Pause/Resume (Space)
  ├─ Stop (Ctrl+S)
  ├─ Reset (Ctrl+R)
  ├─ Step Frame (F6)
  └─ Settings...

View
  ├─ Show/Hide Panels
  │   ├─ Register Viewer
  │   ├─ Memory Viewer
  │   ├─ Log Viewer
  │   ├─ Tile Viewer
  │   ├─ Sprite Viewer
  │   ├─ Palette Viewer
  │   └─ Layer Viewer
  └─ Fullscreen (Alt+F)

Tools
  ├─ Convert Image to Tiles...
  ├─ Convert Palette...
  ├─ Pack Sprite Sheet...
  ├─ Convert Audio...
  ├─ Tile Editor...
  ├─ Sprite Editor...
  └─ Map Editor...

Debug
  ├─ Breakpoints...
  ├─ Watchpoints...
  ├─ Performance Profiler...
  └─ Memory Profiler...

Help
  ├─ Standard Library Reference
  ├─ API Documentation
  ├─ Tutorials
  └─ About
```

### Toolbar Buttons

```
[▶ Start] [⏸ Pause] [⏹ Stop] [↻ Reset] [⏭ Step] [📁 Load ROM] [🔍 Validate]
```

---

## File Structure (Updated)

```
internal/
├── debug/
│   ├── logger.go          # Centralized logging system
│   ├── log_entry.go       # Log entry structure
│   └── log_filter.go      # Log filtering logic
├── ui/
│   ├── ui.go              # Main UI structure
│   ├── menu.go            # Menu bar
│   ├── toolbar.go         # Toolbar
│   ├── statusbar.go       # Status bar
│   ├── panels/
│   │   ├── register_viewer.go
│   │   ├── memory_viewer.go
│   │   ├── log_viewer.go
│   │   ├── tile_viewer.go
│   │   ├── sprite_viewer.go
│   │   ├── palette_viewer.go
│   │   └── layer_viewer.go
│   └── render_fixed.go    # Existing rendering code
├── emulator/
│   └── emulator.go        # Add FullReset(), StepFrame()
└── sdk/                    # NEW: SDK tools
    ├── assembler/          # Enhanced ROM builder
    │   ├── parser.go       # Assembly parser
    │   ├── macro.go        # Macro processor
    │   ├── linker.go       # Linker
    │   └── validator.go    # ROM validator
    ├── assets/             # Asset conversion tools
    │   ├── img2tiles.go    # Image to tile converter
    │   ├── palette.go      # Palette converter
    │   ├── sprites.go      # Sprite packer
    │   └── audio.go        # Audio converter
    └── library/            # Standard library
        ├── stdlib.inc      # System library
        ├── graphics.inc    # Graphics library
        └── audio.inc       # Audio library

cmd/
├── emulator/               # Main emulator
├── assembler/              # NEW: Enhanced assembler
├── validator/              # NEW: ROM validator
├── analyzer/               # NEW: ROM analyzer
├── img2tiles/              # NEW: Image converter
├── palette-converter/      # NEW: Palette converter
├── sprite-packer/          # NEW: Sprite packer
├── audio-converter/        # NEW: Audio converter
├── tile-editor/            # NEW: Tile editor
├── sprite-editor/          # NEW: Sprite editor
└── map-editor/             # NEW: Map editor

sdk/                        # NEW: SDK distribution
├── lib/                    # Standard libraries
│   ├── stdlib.inc
│   ├── graphics.inc
│   └── audio.inc
├── examples/               # Sample projects
│   ├── hello-world/
│   ├── sprite-demo/
│   └── platformer/
└── templates/              # Project templates
    └── starter-project/
```

---

## Success Criteria (Updated)

### Emulator Debugging Tools
- ✅ All `fmt.Printf` calls removed from core emulator code
- ✅ Logging system with UI integration working
- ✅ Register viewer showing real-time CPU state
- ✅ Memory viewer showing real-time memory contents
- ✅ Tile viewer showing visual representation of tiles
- ✅ Start/Stop/Reset working with full memory clearing
- ✅ Component-based logging toggles working

### SDK Build Tools
- ✅ Enhanced assembler with macros and includes
- ✅ ROM validator catches common errors
- ✅ ROM analyzer provides useful insights
- ✅ Image to tile converter works seamlessly
- ✅ Standard library covers common tasks
- ✅ Asset editors are usable and intuitive

### Overall
- ✅ Developer can create a simple game in < 1 hour
- ✅ All assets can be created/imported with SDK tools
- ✅ Debugging is efficient and pleasant
- ✅ UI is responsive and professional-looking
- ✅ Documentation is complete and helpful

---

## Next Steps

1. Review and approve this updated plan
2. Start with Phase 1: Foundation (logging system)
3. Iterate and refine based on feedback
4. Move through phases as foundation is solid
5. Integrate SDK tools into emulator UI as they're completed

---

**End of Plan**

