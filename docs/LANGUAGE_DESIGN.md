# NitroLang: Custom Compiled Language for Nitro-Core-DX

**Created:** January 27, 2026  
**Status:** Design Phase

## Overview

NitroLang is a custom compiled language designed specifically for Nitro-Core-DX. It uses Lua-like syntax for readability and ease of use, but compiles to efficient bytecode (not interpreted). It provides direct assembly access for unimplemented features and low-level control when needed.

**Note:** This is NOT a scripting language - it's a compiled language that happens to use Lua-like syntax for readability. All code is compiled to bytecode before execution.

## Design Goals

1. **Lua-like Syntax**: Familiar, clean, and easy to learn
2. **Dynamic Typing**: No type declarations needed
3. **Compiled to Bytecode**: Fast execution, not interpreted
4. **Assembly Integration**: Direct access to CPU instructions for unimplemented features
5. **Type Safety**: Optional type hints for better tooling
6. **Zero-cost Abstractions**: High-level features compile to efficient code

---

## Language Syntax

### Basic Syntax (Lua-inspired)

```lua
-- Comments use -- (like Lua)

-- Variables are dynamically typed
x = 10
y = 20.5
name = "Hello"
is_active = true

-- Functions
function add(a, b)
    return a + b
end

-- Tables (like Lua)
player = {
    x = 50,
    y = 100,
    speed = 5
}

-- Access table members
player.x = player.x + player.speed
```

### Type System

**Dynamic Types:**
- `number` - 16-bit signed integer (matches CPU registers)
- `float` - 32-bit floating point (if needed)
- `string` - String literals
- `bool` - Boolean (true/false)
- `table` - Associative arrays/objects
- `function` - First-class functions
- `nil` - Null/none value

**Optional Type Hints:**
```lua
-- Type hints for better tooling (compile-time checked, runtime ignored)
function move_sprite(x: number, y: number): void
    -- Implementation
end

-- Type annotations for variables (optional)
local sprite_x: number = 50
local sprite_y: number = 100
```

### Assembly Integration

**Direct Assembly Access:**
```lua
-- Inline assembly for direct CPU control
asm {
    MOV R1, #0x8014  -- OAM_ADDR
    MOV R2, #0x00    -- Sprite 0
    MOV [R1], R2
}

-- Assembly blocks can use variables
local sprite_id = 0
asm {
    MOV R1, #0x8014
    MOV R2, sprite_id  -- Variable substitution
    MOV [R1], R2
}

-- Named assembly functions (reusable)
function reset_oam_addr(sprite_id: number)
    asm {
        MOV R1, #0x8014
        MOV R2, sprite_id
        MOV [R1], R2
    }
end
```

**Register Access:**
```lua
-- Direct register access (when needed)
local reg_r0 = cpu.reg[0]  -- Read R0
cpu.reg[0] = 100           -- Write R0

-- Memory access
local value = mem.read8(0x8014)  -- Read byte from address
mem.write8(0x8014, 0x00)         -- Write byte to address
mem.read16(0x8000)               -- Read 16-bit word
mem.write16(0x8000, 0x1234)      -- Write 16-bit word
```

### Control Flow

```lua
-- If/else
if x > 100 then
    print("X is large")
elseif x > 50 then
    print("X is medium")
else
    print("X is small")
end

-- While loops
while x < 100 do
    x = x + 1
end

-- For loops
for i = 1, 10 do
    print(i)
end

-- For loops with step
for i = 0, 100, 5 do
    print(i)
end

-- Repeat-until (like Lua)
repeat
    x = x + 1
until x >= 100
```

### Functions

```lua
-- Function definition
function update_sprite(x, y)
    -- Set sprite position
    set_sprite_pos(0, x, y)
end

-- Multiple return values (like Lua)
function get_position()
    return sprite_x, sprite_y
end

-- Call with multiple returns
local x, y = get_position()

-- Anonymous functions
local callback = function(value)
    print(value)
end

-- Higher-order functions
function map(array, func)
    local result = {}
    for i, v in ipairs(array) do
        result[i] = func(v)
    end
    return result
end
```

### Standard Library Integration

```lua
-- PPU functions (high-level wrappers)
ppu.set_sprite_pos(sprite_id, x, y)
ppu.set_sprite_tile(sprite_id, tile_index)
ppu.set_sprite_palette(sprite_id, palette)
ppu.set_sprite_size(sprite_id, size)  -- 8x8 or 16x16
ppu.enable_sprite(sprite_id, enabled)

-- Background functions
bg.set_scroll(layer, x, y)
bg.set_tilemap(layer, tilemap_addr)
bg.enable_layer(layer, enabled)

-- Audio functions
audio.play_sound(channel, frequency, duration)
audio.set_volume(channel, volume)
audio.stop_channel(channel)

-- Input functions
input.update()  -- Latch controllers
local button_a = input.pressed(1, "A")
local button_b = input.pressed(1, "B")

-- Memory functions
mem.read8(addr)
mem.write8(addr, value)
mem.read16(addr)
mem.write16(addr, value)
```

### Object-Oriented Features (Optional)

```lua
-- Simple object-oriented style (using tables)
Sprite = {}
function Sprite:new(x, y)
    local obj = {
        x = x,
        y = y,
        tile = 0,
        palette = 1
    }
    setmetatable(obj, { __index = Sprite })
    return obj
end

function Sprite:update()
    self.x = self.x + 1
    ppu.set_sprite_pos(0, self.x, self.y)
end

-- Usage
local sprite = Sprite:new(50, 100)
sprite:update()
```

### Compiler Directives

```lua
-- Include other files
#include "stdlib.ns"
#include "sprite_utils.ns"

-- Define constants
#define MAX_SPRITES 128
#define SCREEN_WIDTH 320
#define SCREEN_HEIGHT 200

-- Conditional compilation
#ifdef DEBUG
    print("Debug mode")
#endif

-- Memory layout hints (for optimizer)
@align(2)  -- Align to 2-byte boundary
local sprite_data = {}
```

---

## Compiler Architecture

### Phase 1: Lexer
- Tokenize source code
- Handle comments, strings, numbers, keywords
- Support both Lua-style and assembly syntax

### Phase 2: Parser
- Build AST (Abstract Syntax Tree)
- Parse Lua-like syntax
- Parse inline assembly blocks
- Handle type hints (optional)

### Phase 3: Semantic Analysis
- Type checking (optional, for hints)
- Variable scope resolution
- Function call validation
- Register allocation planning

### Phase 4: Code Generation
- Generate Nitro-Core-DX bytecode
- Optimize register usage
- Inline function calls where beneficial
- Convert high-level constructs to efficient assembly

### Phase 5: Optimization
- Dead code elimination
- Constant folding
- Register allocation optimization
- Loop unrolling (optional)

---

## Example: Complete Sprite Animation

```lua
-- sprite_demo.ns
#include "stdlib.ns"

-- Constants
local SPRITE_ID = 0
local SCREEN_WIDTH = 320
local WRAP_X = 336

-- Sprite state
local sprite = {
    x = 50,
    y = 50,
    tile = 0,
    palette = 1,
    speed = 1
}

-- Initialize sprite
function init_sprite()
    ppu.set_sprite_pos(SPRITE_ID, sprite.x, sprite.y)
    ppu.set_sprite_tile(SPRITE_ID, sprite.tile)
    ppu.set_sprite_palette(SPRITE_ID, sprite.palette)
    ppu.set_sprite_size(SPRITE_ID, 16)  -- 16x16
    ppu.enable_sprite(SPRITE_ID, true)
end

-- Update sprite position
function update_sprite()
    -- Increment X
    sprite.x = sprite.x + sprite.speed
    
    -- Wrap if off-screen
    if sprite.x >= WRAP_X then
        sprite.x = -16  -- Off-screen left
    end
    
    -- Update sprite position
    ppu.set_sprite_pos(SPRITE_ID, sprite.x, sprite.y)
end

-- Main loop
function main()
    init_sprite()
    
    while true do
        -- Wait for VBlank
        while not ppu.vblank() do
            -- Busy wait
        end
        
        -- Update sprite
        update_sprite()
        
        -- Wait for VBlank to clear
        while ppu.vblank() do
            -- Busy wait
        end
    end
end

-- Entry point
main()
```

---

## Compiler Implementation Plan

### Phase 1: Basic Compiler (Week 1-2)
- Lexer for basic syntax
- Parser for simple expressions
- Code generator for basic operations
- Output to assembly (not bytecode yet)

### Phase 2: Functions and Control Flow (Week 2-3)
- Function parsing and code generation
- If/else, while, for loops
- Local variable scoping
- Basic register allocation

### Phase 3: Standard Library (Week 3-4)
- PPU wrapper functions
- Memory access functions
- Input/Output functions
- Audio functions

### Phase 4: Assembly Integration (Week 4-5)
- Inline assembly parsing
- Variable substitution in assembly
- Register access from high-level code
- Memory access from high-level code

### Phase 5: Optimization (Week 5-6)
- Register allocation optimization
- Dead code elimination
- Constant folding
- Function inlining

### Phase 6: Advanced Features (Week 6-7)
- Tables/objects
- Metatables (optional)
- Closures
- Coroutines (optional)

---

## Tooling Integration

### Language Server Protocol (LSP)
- Syntax highlighting
- Auto-completion
- Error checking
- Go-to definition
- Hover documentation

### IDE Integration
- VS Code extension
- Syntax highlighting
- Debugger integration
- Breakpoints
- Variable inspection

### Build System
```bash
# Compile NitroLang to ROM
nitrolang build sprite_demo.nl -o sprite_demo.rom

# Run with emulator
nitro-core-dx -rom sprite_demo.rom

# Debug mode
nitrolang build sprite_demo.nl -o sprite_demo.rom --debug
nitro-core-dx -rom sprite_demo.rom --debug
```

---

## Comparison with Other Languages

| Feature | NitroLang | Lua | Assembly | C |
|---------|-----------|-----|----------|---|
| Syntax | Lua-like | Lua | Assembly | C |
| Typing | Dynamic | Dynamic | N/A | Static |
| Compilation | Bytecode (compiled) | Bytecode (interpreted) | Machine code | Machine code |
| Assembly Access | Yes | No | Yes | Yes (inline) |
| Standard Library | Yes | Yes | No | Yes |
| Learning Curve | Easy | Easy | Hard | Medium |
| Execution Model | Compiled | Interpreted | Compiled | Compiled |

---

## Future Enhancements

1. **Type System**: Optional gradual typing
2. **Macros**: Lisp-like macros for code generation
3. **Modules**: Import/export system
4. **Generics**: Generic functions/types
5. **Async/Await**: For handling VBlank and timing
6. **Hot Reload**: Update code without restarting ROM

---

**End of Design Document**
