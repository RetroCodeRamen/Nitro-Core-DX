# Nitro-Core-DX Programming Guide

**Last Updated**: 2025-02-11  
**Purpose**: Practical guide for developing ROMs/games for Nitro-Core-DX, based on lessons learned from building test ROMs.

---

## Table of Contents

1. [Basic Setup](#basic-setup)
2. [Display Initialization](#display-initialization)
3. [Color Management](#color-management)
4. [Background Layers](#background-layers)
5. [Sprites](#sprites)
6. [Input Handling](#input-handling)
7. [Main Loop Structure](#main-loop-structure)
8. [Common Patterns](#common-patterns)
9. [Troubleshooting](#troubleshooting)

---

## Basic Setup

### ROM Entry Point

All ROMs must start at bank 1, offset 0x8000 (LoROM mapping):

```go
entryBank := uint8(1)
entryOffset := uint16(0x8000)
```

### Initialization Order

**CRITICAL**: Follow this order for reliable display setup:

1. Set up CGRAM colors
2. Set up VRAM tile data
3. Set up tilemap
4. Enable background layers
5. Initialize sprites (wait for VBlank first)
6. Enter main loop

---

## Display Initialization

### Setting Up a Solid Color Background

To display a solid color background (backdrop), you need:

1. **Set CGRAM palette 0, color 0** to your desired color
2. **Create tile 0** with color index 0 (all 0x00 bytes)
3. **Set up tilemap entry** at 0x4000 referencing tile 0, palette 0
4. **Enable BG0**

**Example: Blue backdrop**

```go
// Step 1: Set CGRAM palette 0, color 0 to blue
// CGRAM address = (palette × 16 + color) = (0 × 16 + 0) = 0x00
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012 (CGRAM_ADDR)
builder.AddImmediate(0x8012)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Write blue color: RGB555 = 0x001F
// Low byte first, then high byte
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013 (CGRAM_DATA)
builder.AddImmediate(0x8013)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x1F (low byte)
builder.AddImmediate(0x1F)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (high byte)
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Step 2: Create tile 0 with color index 0 (all 0x00 bytes)
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E (VRAM_ADDR_L)
builder.AddImmediate(0x800E)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F (VRAM_ADDR_H)
builder.AddImmediate(0x800F)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Write 32 bytes of 0x00 (8×8 tile = 32 bytes for 4bpp)
builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #32
builder.AddImmediate(32)
tileLoopStart := uint16(builder.GetCodeLength() * 2)
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010 (VRAM_DATA)
builder.AddImmediate(0x8010)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
builder.AddInstruction(rom.EncodeSUB(1, 6, 0)) // SUB R6, #1
builder.AddImmediate(1)
builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
builder.AddImmediate(0)
builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
builder.AddInstruction(rom.EncodeBNE())         // BNE tile_loop_start
currentPC := uint16(builder.GetCodeLength() * 2)
offset := rom.CalculateBranchOffset(currentPC, tileLoopStart)
builder.AddImmediate(uint16(offset))

// Step 3: Set up tilemap entry at 0x4000
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E
builder.AddImmediate(0x800E)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F
builder.AddImmediate(0x800F)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x40 (address = 0x4000)
builder.AddImmediate(0x40)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Write tilemap entry: tile 0, palette 0
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010 (VRAM_DATA)
builder.AddImmediate(0x8010)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (tile 0)
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (palette 0)
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Step 4: Enable BG0
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8008 (BG0_CONTROL)
builder.AddImmediate(0x8008)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01 (enable)
builder.AddImmediate(0x01)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
```

**Key Points:**
- The tilemap wraps, so writing ONE entry at 0x4000 is sufficient for a solid color
- Tile 0 must exist and have color index 0 to show the backdrop color
- CGRAM palette 0, color 0 is the backdrop color

---

## Color Management

### CGRAM Address Calculation

CGRAM address = `(palette × 16 + color)`

- **Palette**: 0-15 (16 palettes)
- **Color**: 0-15 (16 colors per palette)
- **Total**: 256 colors (16 × 16)

**Example:**
- Palette 1, Color 1 = `(1 × 16 + 1) = 17 = 0x11`
- Palette 2, Color 3 = `(2 × 16 + 3) = 35 = 0x23`

### RGB555 Color Format

Colors are stored as RGB555 (5 bits per channel):
- **Format**: `RRRRRGGGGGBBBBB` (15 bits)
- **Low byte**: `GGGBBBBB` (bits 0-7)
- **High byte**: `0RRRRRGG` (bits 8-14, bit 15 = 0)

**Write Order**: Low byte first, then high byte

**Common Colors:**
- Black: `0x0000` (low=0x00, high=0x00)
- Red: `0x7C00` (low=0x00, high=0x7C)
- Green: `0x03E0` (low=0xE0, high=0x03)
- Blue: `0x001F` (low=0x1F, high=0x00)
- White: `0x7FFF` (low=0xFF, high=0x7F)

**Example: Setting red color**

```go
// Set CGRAM address to palette 1, color 1
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
builder.AddImmediate(0x8012)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x11
builder.AddImmediate(0x11)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Write red: RGB555 = 0x7C00
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
builder.AddImmediate(0x8013)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (low byte)
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x7C (high byte)
builder.AddImmediate(0x7C)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
```

---

## Background Layers

### Enabling BG0

```go
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8008 (BG0_CONTROL)
builder.AddImmediate(0x8008)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01 (enable, 8×8 tiles)
builder.AddImmediate(0x01)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
```

**BG0_CONTROL (0x8008) bits:**
- Bit 0: Enable (1 = enabled)
- Bit 1: Tile size (0 = 8×8, 1 = 16×16)

### Tilemap Setup

Tilemap entries are 2 bytes:
- **Byte 0**: Tile index (0-255)
- **Byte 1**: Attributes
  - Bits [3:0]: Palette index (0-15)
  - Bit 4: Flip X
  - Bit 5: Flip Y
  - Bits [7:6]: Priority

**Tilemap Base Address**: Default is 0x4000

**Example: Tilemap entry for tile 1, palette 1**

```go
// Set VRAM address to tilemap location
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E
builder.AddImmediate(0x800E)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F
builder.AddImmediate(0x800F)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x40 (0x4000)
builder.AddImmediate(0x40)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Write tilemap entry
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010 (VRAM_DATA)
builder.AddImmediate(0x8010)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01 (tile 1)
builder.AddImmediate(0x01)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01 (palette 1)
builder.AddImmediate(0x01)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
```

---

## Sprites

### Sprite Format

Each sprite is 6 bytes in OAM:
- **Byte 0**: X low byte (bits 0-7)
- **Byte 1**: X high byte (bit 0 = sign bit for X > 255)
- **Byte 2**: Y position (0-199)
- **Byte 3**: Tile index
- **Byte 4**: Attributes
  - Bits [3:0]: Palette index
  - Bit 4: Flip X
  - Bit 5: Flip Y
  - Bits [7:6]: Priority
- **Byte 5**: Control
  - Bit 0: Enable (1 = enabled)
  - Bit 1: Size (0 = 8×8, 1 = 16×16)

### Writing a Sprite

**CRITICAL**: Wait for VBlank before writing OAM!

```go
// Wait for VBlank
waitVBlankStart := uint16(builder.GetCodeLength() * 2)
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x803E (VBLANK_FLAG)
builder.AddImmediate(0x803E)
builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4]
builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
builder.AddImmediate(0)
builder.AddInstruction(rom.EncodeCMP(0, 5, 7)) // CMP R5, R7
builder.AddInstruction(rom.EncodeBEQ())         // BEQ wait_vblank_start
currentPC := uint16(builder.GetCodeLength() * 2)
offset := rom.CalculateBranchOffset(currentPC, waitVBlankStart)
builder.AddImmediate(uint16(offset))

// Set OAM address to sprite 0
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014 (OAM_ADDR)
builder.AddImmediate(0x8014)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (sprite 0)
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Write sprite data via OAM_DATA (0x8015)
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015 (OAM_DATA)
builder.AddImmediate(0x8015)

// Byte 0: X low = 160
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #160
builder.AddImmediate(160)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Byte 1: X high = 0
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Byte 2: Y = 100
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #100
builder.AddImmediate(100)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Byte 3: Tile index = 0
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Byte 4: Attributes = palette 1
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
builder.AddImmediate(0x01)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Byte 5: Control = enable, 8×8
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
builder.AddImmediate(0x01)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
```

---

## Input Handling

### Reading Controller Input

**CRITICAL**: Follow this sequence exactly:

1. **Latch** (write 1 to 0xA001)
2. **Read** (read from 0xA000)
3. **Release latch** (write 0 to 0xA001)

**Button Bit Mappings:**
- Bit 0: UP
- Bit 1: DOWN
- Bit 2: LEFT
- Bit 3: RIGHT
- Bit 4: A
- Bit 5: B
- Bit 6: X
- Bit 7: Y
- Bit 8: L
- Bit 9: R
- Bit 10: SELECT
- Bit 11: START

**Example: Read UP button**

```go
// Latch controller
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001 (CONTROLLER1_LATCH)
builder.AddImmediate(0xA001)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
builder.AddImmediate(0x01)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Read controller state (low byte)
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA000 (CONTROLLER1)
builder.AddImmediate(0xA000)
builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4]

// Release latch
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001
builder.AddImmediate(0xA001)
builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0x00
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 6)) // MOV [R4], R6

// Check UP button (bit 0)
builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5 (copy buttons)
builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x01
builder.AddImmediate(0x01)
builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7 (mask UP bit)
builder.AddInstruction(rom.EncodeCMP(0, 6, 7)) // CMP R6, R7
builder.AddInstruction(rom.EncodeBNE())         // BNE skip_up
// UP is pressed - handle it here
```

---

## Main Loop Structure

### Standard Main Loop Pattern

```go
mainLoopStart := uint16(builder.GetCodeLength() * 2)

// Wait for VBlank
waitVBlankStart := uint16(builder.GetCodeLength() * 2)
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x803E (VBLANK_FLAG)
builder.AddImmediate(0x803E)
builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4]
builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
builder.AddImmediate(0)
builder.AddInstruction(rom.EncodeCMP(0, 5, 7)) // CMP R5, R7
builder.AddInstruction(rom.EncodeBEQ())         // BEQ wait_vblank_start
currentPC := uint16(builder.GetCodeLength() * 2)
offset := rom.CalculateBranchOffset(currentPC, waitVBlankStart)
builder.AddImmediate(uint16(offset))

// Read input
// ... (input reading code)

// Update game state
// ... (game logic)

// Update display (OAM, CGRAM, etc.)
// ... (display updates)

// Loop back
builder.AddInstruction(rom.EncodeJMP())
currentPC = uint16(builder.GetCodeLength() * 2)
offset = rom.CalculateBranchOffset(currentPC, mainLoopStart)
builder.AddImmediate(uint16(offset))
```

**Key Points:**
- Always wait for VBlank before updating OAM
- Read input once per frame
- Update display during VBlank
- Loop back to start

---

## Common Patterns

### Changing Backdrop Color Dynamically

To change the backdrop color based on input or game state:

```go
// Set CGRAM address to palette 0, color 0
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
builder.AddImmediate(0x8012)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Write new color (based on game state in R0)
// Use jump table or conditional logic to select color
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
builder.AddImmediate(0x8013)
// ... write color based on R0 value
```

### Moving a Sprite

```go
// Update sprite X position (stored in R0)
builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014 (OAM_ADDR)
builder.AddImmediate(0x8014)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00 (sprite 0)
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015 (OAM_DATA)
builder.AddImmediate(0x8015)

// Write X low byte (from R0)
builder.AddInstruction(rom.EncodeMOV(0, 5, 0)) // MOV R5, R0
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Write X high byte (0 if X < 256)
builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x00
builder.AddImmediate(0x00)
builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

// Skip Y, Tile, Attributes (OAM auto-increments)
// ... or set OAM_ADDR to byte 2 to write Y directly
```

---

## Troubleshooting

### Black Screen / Nothing Shows

**Checklist:**
1. ✅ CGRAM palette 0, color 0 is set to a visible color (not black)
2. ✅ Tile 0 exists in VRAM with color index 0
3. ✅ Tilemap entry at 0x4000 references tile 0, palette 0
4. ✅ BG0 is enabled (`BG0_CONTROL = 0x01`)
5. ✅ ROM is executing (check CPU cycles)

**Common Issues:**
- Forgot to enable BG0
- Tile 0 doesn't exist or has wrong color index
- CGRAM color is black (0x0000)
- Tilemap entry points to wrong tile/palette

### Sprites Not Showing

**Checklist:**
1. ✅ Sprite is enabled (control byte bit 0 = 1)
2. ✅ Sprite tile exists in VRAM
3. ✅ Sprite palette is set correctly
4. ✅ Sprite X/Y coordinates are within screen bounds (0-319 for X, 0-199 for Y)
5. ✅ OAM was written during VBlank

**Common Issues:**
- Sprite disabled (control byte = 0x00)
- Sprite tile doesn't exist
- Sprite palette not set up in CGRAM
- Sprite off-screen (X/Y out of bounds)

### Input Not Working

**Checklist:**
1. ✅ Latch sequence is correct (write 1, read, write 0)
2. ✅ Reading from correct address (0xA000 for controller 1)
3. ✅ Button bit masks are correct
4. ✅ Input is being read each frame

**Common Issues:**
- Forgot to latch (write 1 to 0xA001)
- Reading before latching
- Wrong button bit mask
- Input read only once, not each frame

### Colors Wrong

**Checklist:**
1. ✅ CGRAM address calculation is correct: `(palette × 16 + color)`
2. ✅ RGB555 format: low byte first, then high byte
3. ✅ Color values are correct for RGB555

**Common Issues:**
- Wrong CGRAM address (palette/color calculation error)
- RGB555 byte order wrong (high byte first)
- Color values not in RGB555 format

---

## Best Practices

1. **Always wait for VBlank** before writing OAM
2. **Initialize display before main loop** (CGRAM, VRAM, tilemap)
3. **Read input once per frame** during main loop
4. **Use consistent register conventions**:
   - R0-R3: Temporary/scratch
   - R4-R5: I/O address/value pairs
   - R6-R7: Comparison values
5. **Test incrementally**: Get display working first, then add input, then add game logic
6. **Use delay loops** instead of tight loops if VBlank timing is uncertain
7. **Fill tilemap entries** if using multiple tiles (or rely on wrapping for single tile)

---

**Last Updated**: 2025-02-11  
**Version**: 1.0
