# Development Roadmap

## Current Status (Updated: 2025-12-30)

✅ **Completed:**
- Project structure and architecture
- All module frameworks in place
- Memory system (read/write working, I/O mapping, LoROM-style ROM mapping)
- ROM loading infrastructure (32-byte header, entry point, checksum)
- ROM builder tool (`create_graphics_rom.py`, `create_test_rom.py`)
- **CPU Instruction Set (Mostly Complete):**
  - ✅ Instruction decoder (proper opcode extraction)
  - ✅ MOV instruction (register-to-register, immediate, load/store)
  - ✅ ADD/SUB instructions
  - ✅ MUL/DIV instructions
  - ✅ Logical operations (AND, OR, XOR, NOT)
  - ✅ Shift operations (SHL, SHR)
  - ✅ Comparison (CMP)
  - ✅ Branch instructions (BEQ, BNE, BGT, BLT, BGE, BLE)
  - ✅ JMP instruction (absolute, relative)
  - ✅ Stack operations (PUSH, POP, CALL, RET)
  - ✅ Cycle counting per instruction
  - ✅ CPU bootstrap/BIOS (auto-recovery from invalid PC states)
- **PPU Rendering:**
  - ✅ Basic tile rendering (8x8 and 16x16 tiles, 4bpp)
  - ✅ Tilemap rendering with scrolling
  - ✅ Sprite rendering (8x8 and 16x16)
  - ✅ Palette system (256 colors, RGB555)
  - ✅ Background layer rendering (BG0, BG1)
  - ✅ Framebuffer compositing (stub)
  - ✅ Portrait mode rotation (stub)
- **UI System:**
  - ✅ Menu bar with dropdown menus
  - ✅ Settings window (render scale, logging controls)
  - ✅ Hex debugger window (live memory view, bank selection, copy/paste, dump to file)
  - ✅ Status bar (frame count, PC, FPS, running/paused status)
  - ✅ Draggable windows with close buttons
  - ✅ File dialogs (non-blocking, threaded)
- **Logging System:**
  - ✅ Configurable log levels (ERROR, WARNING, INFO, DEBUG, TRACE)
  - ✅ Category filtering (CPU, Memory, PPU, ROM)
  - ✅ Log file saving
  - ✅ Auto-save boot log (first 100,000 messages)
  - ✅ Global logging disable toggle
- **Input System:**
  - ✅ Pygame key mapping
  - ✅ Controller input handling
  - ✅ I/O register reads/writes
- **Audio System:**
  - ✅ APU initialization
  - ✅ 4-channel synthesizer (sine, square, saw, noise)
  - ✅ Sample buffer generation
  - ⚠️ Audio output to pygame (generates samples but not fully connected)
- **Performance:**
  - ✅ Optimized rendering (pygame.image.fromstring instead of set_at)
  - ✅ SNES-period-accurate CPU timing (44,667 cycles/frame)
  - ✅ Frame rate targeting (60 FPS)
- **Testing:**
  - ✅ Comprehensive test suite (CPU instructions, branches, stack, etc.)
  - ✅ Interactive debugging tools
  - ✅ Test ROMs (simple loop, animated graphics)

🚧 **In Progress / Known Issues:**
- ⚠️ PPU tile rendering: Two boxes appearing instead of one (tilemap wrapping issue)
- ⚠️ PPU performance: Currently 30-60 FPS (acceptable but could be better)
- ⚠️ APU audio output: Generates samples but not fully connected to pygame mixer
- ⚠️ Sprite priority and layering: Basic implementation, needs refinement
- ⚠️ Interrupt handling: Stub only

## Priority 1: Get Basic Emulator Running ✅ COMPLETE

### 1.1 Implement Core CPU Instructions ✅ COMPLETE
**Goal:** Make the CPU actually execute code

**Tasks:**
- [x] Implement instruction decoder (proper opcode extraction)
- [x] Implement MOV instruction (register-to-register, immediate)
- [x] Implement ADD/SUB instructions
- [x] Implement JMP instruction (absolute, relative)
- [x] Implement basic addressing modes
- [x] Add cycle counting per instruction

**Status:** All core instructions implemented and tested.

### 1.2 Create a Test ROM ✅ COMPLETE
**Goal:** Have something to actually run

**Tasks:**
- [x] Design simple ROM format
- [x] Create ROM builder tool (Python script)
- [x] Build a minimal test ROM that:
  - Sets up PPU
  - Draws something simple
  - Loops forever

**Status:** ROM builder created, test ROMs working (simple loop, animated bouncing box).

### 1.3 Fix PPU Rendering 🚧 MOSTLY COMPLETE
**Goal:** Show something on screen

**Tasks:**
- [x] Implement basic tile rendering
- [x] Implement sprite rendering
- [x] Fix framebuffer compositing (stub implemented)
- [x] Test with simple patterns
- [ ] **Fix tilemap wrapping issue (two boxes appearing)**

**Status:** Rendering works, but tilemap wrapping causes duplicate tiles. This is the current issue.

## Priority 2: Make It Functional

### 2.1 Complete CPU Instruction Set ✅ MOSTLY COMPLETE
**Tasks:**
- [x] Implement all arithmetic operations (MUL, DIV, AND, OR, XOR)
- [x] Implement comparison operations (CMP)
- [x] Implement branch instructions (BEQ, BNE, BGT, BLT, BGE, BLE)
- [x] Implement stack operations (PUSH, POP, CALL, RET)
- [x] Implement load/store instructions (via MOV)
- [ ] Add interrupt handling (stub only)

**Status:** All major instructions implemented. Interrupt handling needs work.

### 2.2 Audio Output 🚧 PARTIAL
**Tasks:**
- [x] Initialize pygame audio
- [x] Connect APU sample buffer to pygame
- [ ] Test with simple tones
- [ ] Verify audio playback works correctly

**Status:** Audio system generates samples but needs testing and verification.

### 2.3 Input Integration ✅ COMPLETE
**Tasks:**
- [x] Complete pygame key mapping
- [x] Test controller input
- [x] Verify I/O register reads/writes

**Status:** Input system fully functional.

## Priority 3: Polish & Features

### 3.1 Advanced PPU Features 🚧 IN PROGRESS
- [x] Proper tilemap rendering (working, but has wrapping issue)
- [x] Sprite rendering (basic implementation)
- [ ] **Fix tilemap coordinate wrapping** ⚠️ **CURRENT ISSUE**
- [ ] Sprite priority and layering (basic, needs refinement)
- [ ] Palette effects
- [ ] Rotation/scaling (if supported)

**Current Issue:** Two boxes appearing instead of one. Likely caused by:
- Tilemap wrapping logic checking the same tile twice
- Tile coordinate calculation when scroll values wrap around
- Need to ensure each tile is only rendered once per frame

**Potential Fixes:**
- Track which tiles have been rendered in current frame
- Improve tile range calculation to prevent duplicate checks
- Verify tilemap index calculation with wrapping

### 3.2 Debugging Tools ✅ MOSTLY COMPLETE
- [x] Debug overlay on screen (status bar)
- [x] Memory viewer (hex debugger)
- [ ] Breakpoints (not yet implemented)
- [ ] Disassembler (not yet implemented)

**Status:** Basic debugging tools in place. Advanced features (breakpoints, disassembler) would be useful.

### 3.3 ROM Development Tools 🚧 PARTIAL
- [x] ROM builder (Python script)
- [ ] Assembler for the CPU (manual encoding currently)
- [ ] ROM packer (basic format exists)
- [ ] Sprite/tile editor
- [ ] Documentation for ROM format (basic, needs expansion)

**Status:** Can create ROMs manually. Assembler would make development easier.

## Known Issues & Next Steps

### Current Priority: Fix Two Boxes Issue

**Problem:** Two boxes appearing instead of one when rendering tiles.

**Possible Causes:**
1. Tilemap wrapping causing same tile to be checked twice
2. Tile coordinate calculation with scroll wrapping
3. Tile range calculation checking overlapping regions

**Investigation Steps:**
1. Add debug logging to track which tiles are being rendered
2. Verify tilemap index calculation with modulo wrapping
3. Check if tile range overlaps when scroll wraps around
4. Ensure each screen pixel is only written once per frame

**Potential Solutions:**
- Track rendered tiles per frame to prevent duplicates
- Improve tile range calculation to account for wrapping
- Add bounds checking to prevent rendering same tile twice
- Verify scroll value handling when it wraps

### Performance Optimization
- ✅ Replaced slow `set_at()` with `pygame.image.fromstring()` (10-100x speedup)
- Current: 30-60 FPS (acceptable)
- Could improve further with:
  - Optimized tile rendering (cache tile data)
  - Reduce logging overhead when disabled
  - Profile and optimize hot paths

### Future Enhancements
- Interrupt handling (VBlank, timer, etc.)
- More advanced PPU features (palette effects, rotation)
- CPU disassembler for debugging
- Breakpoint system
- Save states
- ROM development tools (assembler, sprite editor)

## Testing Strategy

1. **Unit tests** ✅ - Test each instruction in isolation
2. **Integration tests** ✅ - Test ROM loading → CPU → PPU pipeline
3. **Visual tests** ✅ - Run ROMs and verify output
4. **Interactive debugging** ✅ - Use test_interactive.py to inspect state
5. **Performance tests** - Verify frame rate and responsiveness

## Recommended Next Steps

1. **Fix tilemap wrapping issue** (1-2 hours)
   - Debug why two boxes appear
   - Fix tile coordinate calculation
   - Test with various scroll values

2. **Improve PPU performance** (optional, 1-2 hours)
   - Profile rendering code
   - Optimize tile data access
   - Cache frequently used data

3. **Complete audio output** (1 hour)
   - Test audio playback
   - Verify sample generation
   - Fix any audio issues

4. **Add breakpoint system** (2-3 hours)
   - Implement breakpoint checking in CPU
   - Add UI for setting breakpoints
   - Add step-over/step-into functionality

5. **Create assembler** (4-6 hours)
   - Design assembly syntax
   - Implement parser
   - Generate ROM files
   - Would greatly improve ROM development experience
