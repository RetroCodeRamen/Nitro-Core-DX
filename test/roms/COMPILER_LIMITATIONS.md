# CoreLX Compiler Current Limitations

The CoreLX compiler is still in early development. The pellet game was written but encounters these limitations:

## Current Issues

1. **Variable Storage Not Implemented**
   - Variables are declared but values aren't stored or retrieved
   - All variable reads return 0 (placeholder)
   - Variables can't persist across statements

2. **Struct Member Access Not Implemented**
   - Assignments like `player.tile = base` are discarded
   - Struct members can't be read or written
   - `sprite.set_pos()` is a placeholder (does nothing)

3. **Sprite Data Not Passed to OAM**
   - `oam.write()` sets OAM_ADDR but writes hardcoded sprite data
   - Can't pass tile, attr, ctrl values from variables
   - Currently writes: x=160, y=96, tile=0, attr=1, ctrl=1

4. **Tile Loading Not Implemented**
   - `gfx.load_tiles()` returns a placeholder value
   - Tiles aren't actually loaded to VRAM
   - No tile data means sprites won't render

5. **Palette Setup Missing**
   - CGRAM isn't initialized with colors
   - Even if sprites render, they'll be invisible without palettes

## What Works

- ✅ Basic syntax parsing
- ✅ Function calls (built-ins)
- ✅ Control flow (if/while/for)
- ✅ PPU display enable
- ✅ VBlank waiting
- ✅ OAM_ADDR setting

## What's Needed

To make the game work, the compiler needs:

1. **Variable Storage System**
   - Track variables in WRAM or registers
   - Store and load variable values
   - Handle variable scoping

2. **Struct Support**
   - Store struct instances in memory
   - Access struct members (read/write)
   - Pass struct data to functions

3. **Tile Loading**
   - Load asset data to VRAM
   - Set up VRAM addresses correctly
   - Initialize tile data

4. **Palette Initialization**
   - Set up CGRAM with colors
   - Map palettes to sprites correctly

5. **Complete OAM Writing**
   - Read sprite data from structs
   - Write all 6 bytes correctly
   - Support multiple sprites

## Workaround

For now, you can test sprite rendering by:
1. Using a Go test ROM that directly writes to OAM/VRAM/CGRAM
2. Waiting for compiler improvements
3. Manually writing assembly-style code

See `test/roms/build_simple_sprite.go` for a working example using the ROM builder directly.
