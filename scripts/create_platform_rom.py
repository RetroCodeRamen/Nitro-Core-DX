"""
create_platform_rom.py - Create a simple platform game ROM
Player: 5x10 cube (red top, blue bottom)
Controls: Arrow keys to move, A to jump, B to shoot
Background: Light blue
Floor: Green line at bottom third
Bullets: 2x2 black cubes
"""

import struct
import sys
import os

# Add src_python to path
script_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(script_dir)
src_python_path = os.path.join(project_root, 'src_python')
sys.path.insert(0, src_python_path)

import config

# ROM Header Constants
ROM_MAGIC = config.ROM_MAGIC
ROM_VERSION = config.ROM_VERSION

# Default entry point
DEFAULT_ENTRY_BANK = 1
DEFAULT_ENTRY_OFFSET = 0x8000

# Memory I/O addresses
MEM_IO_PPU_BASE = config.MEM_IO_PPU_BASE
MEM_IO_INPUT_BASE = config.MEM_IO_INPUT_BASE
PPU_REG_BG0_CONTROL = config.PPU_REG_BG0_CONTROL
PPU_REG_BG0_SCROLLX = config.PPU_REG_BG0_SCROLLX
PPU_REG_BG0_SCROLLY = config.PPU_REG_BG0_SCROLLY
PPU_REG_VRAM_ADDR = config.PPU_REG_VRAM_ADDR
PPU_REG_VRAM_DATA = config.PPU_REG_VRAM_DATA
PPU_REG_CGRAM_ADDR = config.PPU_REG_CGRAM_ADDR
PPU_REG_CGRAM_DATA = config.PPU_REG_CGRAM_DATA
INPUT_REG_CONTROLLER1 = config.INPUT_REG_CONTROLLER1
INPUT_REG_CONTROLLER1_LATCH = config.INPUT_REG_CONTROLLER1_LATCH

# Input button masks
INPUT_BUTTON_LEFT = config.INPUT_BUTTON_LEFT
INPUT_BUTTON_RIGHT = config.INPUT_BUTTON_RIGHT
INPUT_BUTTON_UP = config.INPUT_BUTTON_UP
INPUT_BUTTON_DOWN = config.INPUT_BUTTON_DOWN
INPUT_BUTTON_A = config.INPUT_BUTTON_A
INPUT_BUTTON_B = config.INPUT_BUTTON_B

# Screen dimensions
SCREEN_WIDTH = 320
SCREEN_HEIGHT = 200
FLOOR_Y = 133  # Bottom third (200 * 2/3 = 133)

# Instruction encoding helpers
def encode_mov_imm(reg, value):
    """MOV R, #imm"""
    return (0x1 << 12) | (0x1 << 8) | (reg << 4) | 0x0, value

def encode_mov_store(reg1, reg2):
    """MOV [R1], R2 - store 16-bit value (low byte will be used for 8-bit I/O)"""
    return (0x1 << 12) | (0x3 << 8) | (reg1 << 4) | reg2

def encode_mov_reg(reg1, reg2):
    """MOV R1, R2 - register to register"""
    return (0x1 << 12) | (0x0 << 8) | (reg1 << 4) | reg2

def encode_mov_load(reg1, reg2):
    """MOV R1, [R2] - load from memory"""
    return (0x1 << 12) | (0x2 << 8) | (reg1 << 4) | reg2

def encode_add_reg(reg1, reg2):
    """ADD R1, R2"""
    return (0x2 << 12) | (0x0 << 8) | (reg1 << 4) | reg2

def encode_add_imm(reg, value):
    """ADD R, #imm"""
    return (0x2 << 12) | (0x1 << 8) | (reg << 4) | 0x0, value

def encode_sub_imm(reg, value):
    """SUB R, #imm"""
    return (0x3 << 12) | (0x1 << 8) | (reg << 4) | 0x0, value

def encode_cmp_imm(reg, value):
    """CMP R, #imm"""
    return (0xC << 12) | (0x1 << 8) | (reg << 4) | 0x0, value

def encode_bne(offset):
    """BNE - Branch if not equal"""
    return (0xC << 12) | (0x2 << 8) | 0x0, offset

def encode_beq(offset):
    """BEQ - Branch if equal"""
    return (0xC << 12) | (0x1 << 8) | 0x0, offset

def encode_blt(offset):
    """BLT - Branch if less than"""
    return (0xC << 12) | (0x4 << 8) | 0x0, offset

def encode_bge(offset):
    """BGE - Branch if greater or equal"""
    return (0xC << 12) | (0x5 << 8) | 0x0, offset

def encode_jmp_rel(offset):
    """JMP rel"""
    return (0xD << 12) | (0x1 << 8) | 0x0, offset

def encode_and_imm(reg, value):
    """AND R, #imm"""
    return (0x6 << 12) | (0x1 << 8) | (reg << 4) | 0x0, value

def encode_shl_imm(reg, value):
    """SHL R, #imm"""
    return (0xA << 12) | (0x1 << 8) | (reg << 4) | 0x0, value

def encode_shr_imm(reg, value):
    """SHR R, #imm"""
    return (0xB << 12) | (0x1 << 8) | (reg << 4) | 0x0, value

def encode_nop():
    return 0x0000


def create_platform_rom(output_file="platform.rom"):
    """
    Create a platform game ROM
    """
    
    code = []
    
    # ============================================================================
    # INITIALIZATION
    # ============================================================================
    
    # R0 = I/O base (0x8000)
    inst, imm = encode_mov_imm(0, MEM_IO_PPU_BASE)
    code.append(inst)
    code.append(imm)
    
    # Set up palette colors
    # Palette 0: Transparent (index 0), Light Blue background (index 1), Green floor (index 2)
    # Palette 1: Player - Red top (index 1), Blue bottom (index 2), Black bullets (index 3)
    
    # Set CGRAM address to 0 (palette 0, color 0)
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_CGRAM_ADDR)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x00)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Color 0: Black (transparent) - RGB555: 0x0000
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_CGRAM_DATA)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0000)  # Low byte
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(2, 0x0000)  # High byte
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Color 1: Light Blue - RGB555: R=10, G=20, B=31 -> 0x4A7A
    # RGB555: bits 14-10=B, 9-5=G, 4-0=R
    # R=10 (0x0A), G=20 (0x14), B=31 (0x1F)
    # = 0x0A | (0x14 << 5) | (0x1F << 10) = 0x0A | 0x0280 | 0x7C00 = 0x7E8A
    light_blue = 0x0A | (0x14 << 5) | (0x1F << 10)  # 0x7E8A
    inst, imm = encode_mov_imm(2, light_blue & 0xFF)  # Low byte
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(2, (light_blue >> 8) & 0xFF)  # High byte
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Color 2: Green - RGB555: R=0, G=31, B=0 -> 0x03E0
    green = (0x1F << 5)  # 0x03E0
    inst, imm = encode_mov_imm(2, green & 0xFF)  # Low byte
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(2, (green >> 8) & 0xFF)  # High byte
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Palette 1: Player colors
    # Color 16 (palette 1, index 0): Transparent
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_CGRAM_ADDR)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 32)  # Palette 1, color 0 = index 16 * 2 = 32
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_CGRAM_DATA)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0000)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(2, 0x0000)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Color 17: Red - RGB555: R=31, G=0, B=0 -> 0x001F
    red = 0x1F  # 0x001F
    inst, imm = encode_mov_imm(2, red & 0xFF)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(2, (red >> 8) & 0xFF)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Color 18: Blue - RGB555: R=0, G=0, B=31 -> 0x7C00
    blue = (0x1F << 10)  # 0x7C00
    inst, imm = encode_mov_imm(2, blue & 0xFF)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(2, (blue >> 8) & 0xFF)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Color 19: Black - RGB555: R=0, G=0, B=0 -> 0x0000
    inst, imm = encode_mov_imm(2, 0x0000)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(2, 0x0000)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Enable BG0
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_BG0_CONTROL)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x01)  # Enable BG0, 8x8 tiles
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Write tilemap: Fill with light blue background (tile 1, palette 0)
    # Set VRAM address to tilemap base (0x0000)
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0000)  # Address low
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR + 1)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0000)  # Address high
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Fill tilemap with background tile (tile 1, palette 0)
    # Tilemap is 32x32 = 1024 entries, each 2 bytes
    # We'll fill a smaller area for now (screen is 40x25 tiles at 8x8)
    # For simplicity, fill entire tilemap
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_DATA)
    code.append(inst)
    code.append(imm)
    
    # Fill tilemap: tile index 1, palette 0 = 0x0001 (low byte = tile, high byte = palette << 4)
    # We'll write this in a loop, but for ROM size, let's write floor tiles directly
    # Actually, let's write floor tiles at specific positions
    # Floor is at Y = 133, which is tile Y = 133/8 = 16 (rounded down)
    # We need to write to tilemap positions for row 16
    
    # First, write background tile (tile 1) to all positions
    # Then overwrite floor row with green tile (tile 2)
    # For now, let's write a simple pattern
    
    # Write background tiles (tile 1, palette 0) - we'll do this in the game loop
    # For initialization, just set up tile data
    
    # Write tile data
    # Tile 1: Light blue (solid color 1)
    # Tile 2: Green floor (solid color 2)
    # Tile 3: Player top (red, color 1 from palette 1)
    # Tile 4: Player bottom (blue, color 2 from palette 1)
    # Tile 5: Bullet (black, color 3 from palette 1)
    
    # Write tile data starting at address 0
    # Tile 0: Transparent (all 0s - already zero, skip)
    # Tile 1: Light blue background
    # Tile 2: Green floor
    # Tile 3: Player top (red)
    # Tile 4: Player bottom (blue)
    # Tile 5: Bullet (black)
    # Each tile is 32 bytes (8x8, 4bpp)
    # So tiles 0-5 = 6 * 32 = 192 bytes
    # We'll write tile data starting at 0x0000 (tile 0 is transparent, so we start with tile 1)
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0000)  # Start at address 0 (tile 0 = transparent, skip)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR + 1)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0000)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Skip tile 0 (transparent, already zero), start with tile 1
    # Add 32 bytes to skip tile 0
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0020)  # Address = 32 (skip tile 0)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR + 1)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0000)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Write tile data
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_DATA)
    code.append(inst)
    code.append(imm)
    
    # Tile 1: Light blue solid (all pixels = color 1)
    # 4bpp: 2 pixels per byte, 4 bytes per row, 32 bytes per tile
    for _ in range(32):
        inst, imm = encode_mov_imm(2, 0x11)  # Two pixels of color 1
        code.append(inst)
        code.append(imm)
        code.append(encode_mov_store(1, 2))
    
    # Tile 2: Green floor (all pixels = color 2)
    for _ in range(32):
        inst, imm = encode_mov_imm(2, 0x22)  # Two pixels of color 2
        code.append(inst)
        code.append(imm)
        code.append(encode_mov_store(1, 2))
    
    # Tile 3: Player top half (red, color 1 from palette 1)
    # 5 pixels wide x 5 pixels tall, centered in 8x8 tile
    # Top 5 rows, middle 5 columns
    player_top_data = [
        0x00, 0x11, 0x10, 0x00,  # Row 0: 0, 1, 1, 1, 1, 1, 0, 0
        0x00, 0x11, 0x10, 0x00,  # Row 1
        0x00, 0x11, 0x10, 0x00,  # Row 2
        0x00, 0x11, 0x10, 0x00,  # Row 3
        0x00, 0x11, 0x10, 0x00,  # Row 4
        0x00, 0x00, 0x00, 0x00,  # Row 5
        0x00, 0x00, 0x00, 0x00,  # Row 6
        0x00, 0x00, 0x00, 0x00,  # Row 7
    ]
    for byte_val in player_top_data:
        inst, imm = encode_mov_imm(2, byte_val)
        code.append(inst)
        code.append(imm)
        code.append(encode_mov_store(1, 2))
    
    # Tile 4: Player bottom half (blue, color 2 from palette 1)
    player_bottom_data = [
        0x00, 0x00, 0x00, 0x00,  # Row 0
        0x00, 0x00, 0x00, 0x00,  # Row 1
        0x00, 0x00, 0x00, 0x00,  # Row 2
        0x00, 0x22, 0x20, 0x00,  # Row 3: 0, 2, 2, 2, 2, 2, 0, 0
        0x00, 0x22, 0x20, 0x00,  # Row 4
        0x00, 0x22, 0x20, 0x00,  # Row 5
        0x00, 0x22, 0x20, 0x00,  # Row 6
        0x00, 0x22, 0x20, 0x00,  # Row 7
    ]
    for byte_val in player_bottom_data:
        inst, imm = encode_mov_imm(2, byte_val)
        code.append(inst)
        code.append(imm)
        code.append(encode_mov_store(1, 2))
    
    # Tile 5: Bullet (black 2x2, color 3 from palette 1)
    # 2x2 black square centered in 8x8
    bullet_data = [
        0x00, 0x00, 0x00, 0x00,  # Row 0
        0x00, 0x00, 0x00, 0x00,  # Row 1
        0x00, 0x00, 0x00, 0x00,  # Row 2
        0x00, 0x33, 0x30, 0x00,  # Row 3: 0, 3, 3, 0, 0, 0, 0, 0
        0x00, 0x33, 0x30, 0x00,  # Row 4
        0x00, 0x00, 0x00, 0x00,  # Row 5
        0x00, 0x00, 0x00, 0x00,  # Row 6
        0x00, 0x00, 0x00, 0x00,  # Row 7
    ]
    for byte_val in bullet_data:
        inst, imm = encode_mov_imm(2, byte_val)
        code.append(inst)
        code.append(imm)
        code.append(encode_mov_store(1, 2))
    
    # Initialize game state
    # R3 = player X position (starts at 160 = center)
    inst, imm = encode_mov_imm(3, 160)
    code.append(inst)
    code.append(imm)
    
    # R4 = player Y position (starts at FLOOR_Y - 10 = 123)
    inst, imm = encode_mov_imm(4, FLOOR_Y - 10)
    code.append(inst)
    code.append(imm)
    
    # R5 = player velocity Y (0 = on ground, negative = jumping, positive = falling)
    inst, imm = encode_mov_imm(5, 0x0000)
    code.append(inst)
    code.append(imm)
    
    # R6 = jump button pressed last frame (for edge detection)
    inst, imm = encode_mov_imm(6, 0x0000)
    code.append(inst)
    code.append(imm)
    
    # R7 = shoot button pressed last frame (for edge detection)
    inst, imm = encode_mov_imm(7, 0x0000)
    code.append(inst)
    code.append(imm)
    
    # Initialize bullet positions (we'll support 4 bullets max)
    # Store bullets in WRAM: each bullet = 4 bytes (X low, X high, Y low, Y high)
    # Bullet 0: WRAM[0x0000-0x0003]
    # Bullet 1: WRAM[0x0004-0x0007]
    # Bullet 2: WRAM[0x0008-0x000B]
    # Bullet 3: WRAM[0x000C-0x000F]
    # Active flag: WRAM[0x0010-0x0013] (1 byte per bullet)
    
    # Clear all bullets (set X/Y to 0, active to 0)
    # We'll do this by writing 0 to WRAM addresses
    # For now, bullets start inactive
    
    # ============================================================================
    # MAIN GAME LOOP
    # ============================================================================
    
    game_loop_start = len(code)
    
    # Fill tilemap with background tiles
    # Tilemap starts at 0x0000, but we need to put it after tile data
    # Tile data (tiles 0-5) = 6 * 32 = 192 bytes (0x00C0)
    # So tilemap should start at 0x0800 to avoid conflict
    # But PPU defaults tile_map_base to 0, so we'll put tilemap at 0x0000
    # and tile data starting at tile index 0 (address 0)
    # Actually, let's put tilemap at 0x0800 and add a register for it later
    # For now, let's use a simpler approach: tilemap at 0, tile data at 0x0800
    
    # Actually, the simplest: write tilemap starting at 0x0800 (after tile data)
    # But PPU reads from tile_map_base which defaults to 0
    # So let's write tilemap at 0, and tile data will be at 0x0800
    # But we need tile_data_base = 0x0800, which isn't settable
    
    # Simplest fix: Write tile data at address 0 (tile 0 = transparent, tile 1 = bg, etc.)
    # Write tilemap starting at 0x0800, but PPU reads from 0
    # So I need to either:
    # 1. Add tile_map_base register, OR
    # 2. Write tilemap at 0 and tile data elsewhere
    
    # For now, let's write tilemap at 0x0800 and I'll add a register to set tile_map_base
    # But that requires PPU changes. Let me use a workaround:
    # Write tile data at 0x0800, keep tilemap at 0, but reference tiles starting from index 32
    # (since 0x0800 / 32 = 128, so tile index 128 = address 0x0800)
    
    # Actually, simplest: Write tilemap at 0, fill it with background (tile 1)
    # Tile data is already written starting at address 0x0020 (tile 1)
    # But tile_data_base is 0, so tile 1 is at address 0 + 32 = 32, which matches!
    
    # Fill visible tilemap area (40x25 = 1000 tiles) with background tile 1
    # We'll write in a loop, but for ROM size, let's write key rows
    # Set VRAM address to tilemap start (0x0000)
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0000)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR + 1)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0000)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Write background tile (tile 1, palette 0) to visible area
    # Screen is 40 tiles wide x 25 tiles tall = 1000 entries
    # But that's too many. Let's write a loop using R7 as counter
    # R7 = counter (starts at 1000, counts down)
    inst, imm = encode_mov_imm(7, 1000)  # 40 * 25 = 1000 tiles
    code.append(inst)
    code.append(imm)
    fill_bg_loop_start = len(code)
    # Write tile 1, palette 0
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_DATA)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x01)  # Tile 1, low byte
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(2, 0x00)  # Palette 0, high byte
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    # Decrement counter
    inst, imm = encode_sub_imm(7, 1)
    code.append(inst)
    code.append(imm)
    # BNE fill_bg_loop_start
    fill_bg_loop_bne_placeholder = len(code)
    inst, imm = encode_bne(0x0000)
    code.append(inst)
    code.append(imm)
    fill_bg_loop_end = len(code)
    
    # Now overwrite floor row (row 16) with green tiles
    # Calculate address: row 16 * 32 tiles/row * 2 bytes/tile = 1024 = 0x0400
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0400 & 0xFF)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR + 1)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, (0x0400 >> 8) & 0xFF)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Write floor tiles (tile 2, palette 0) for 40 tiles
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_DATA)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(7, 40)  # Counter for floor tiles
    code.append(inst)
    code.append(imm)
    fill_floor_loop_start = len(code)
    inst, imm = encode_mov_imm(2, 0x02)  # Tile 2, low byte
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(2, 0x00)  # Palette 0, high byte
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_sub_imm(7, 1)
    code.append(inst)
    code.append(imm)
    fill_floor_loop_bne_placeholder = len(code)
    inst, imm = encode_bne(0x0000)
    code.append(inst)
    code.append(imm)
    fill_floor_loop_end = len(code)
    
    # Clear old player position (write background tile)
    # We'll store old player position in WRAM for clearing
    # For now, just draw player - we'll optimize clearing later
    
    # Read input and handle movement
    # Latch controller
    inst, imm = encode_mov_imm(1, MEM_IO_INPUT_BASE + INPUT_REG_CONTROLLER1_LATCH)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x01)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(2, 0x00)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Read input buttons using shift register
    # I/O is at bank 0, so MOV R2, [R1] where R1 = I/O address will read from I/O
    # Input register address
    inst, imm = encode_mov_imm(1, MEM_IO_INPUT_BASE + INPUT_REG_CONTROLLER1)
    code.append(inst)
    code.append(imm)
    
    # Read and discard UP (button 0)
    code.append(encode_mov_load(2, 1))  # Read UP (discard)
    # Read and discard DOWN (button 1)
    code.append(encode_mov_load(2, 1))  # Read DOWN (discard)
    # Read LEFT (button 2) into R2
    code.append(encode_mov_load(2, 1))  # Read LEFT
    # Store LEFT in WRAM[0x0020]
    inst, imm = encode_mov_imm(1, 0x0020)  # WRAM address
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))  # Store LEFT
    
    # Read RIGHT (button 3)
    inst, imm = encode_mov_imm(1, MEM_IO_INPUT_BASE + INPUT_REG_CONTROLLER1)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_load(2, 1))  # Read RIGHT
    inst, imm = encode_mov_imm(1, 0x0021)  # WRAM address
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))  # Store RIGHT
    
    # Read A (button 4)
    inst, imm = encode_mov_imm(1, MEM_IO_INPUT_BASE + INPUT_REG_CONTROLLER1)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_load(2, 1))  # Read A
    inst, imm = encode_mov_imm(1, 0x0022)  # WRAM address
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))  # Store A
    
    # Read B (button 5)
    inst, imm = encode_mov_imm(1, MEM_IO_INPUT_BASE + INPUT_REG_CONTROLLER1)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_load(2, 1))  # Read B
    inst, imm = encode_mov_imm(1, 0x0023)  # WRAM address
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))  # Store B
    
    # Handle LEFT movement
    # Load LEFT button state from WRAM[0x0020]
    inst, imm = encode_mov_imm(1, 0x0020)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_load(2, 1))  # R2 = LEFT button state
    # CMP R2, #0 (check if pressed)
    inst, imm = encode_cmp_imm(2, 0)
    code.append(inst)
    code.append(imm)
    # BEQ skip_left (if not pressed, skip)
    skip_left_placeholder = len(code)
    inst, imm = encode_beq(0x0000)
    code.append(inst)
    code.append(imm)
    # LEFT pressed: SUB R3, #2 (move left 2 pixels)
    inst, imm = encode_sub_imm(3, 2)
    code.append(inst)
    code.append(imm)
    # Clamp to screen: if R3 < 0, set to 0
    inst, imm = encode_cmp_imm(3, 0)
    code.append(inst)
    code.append(imm)
    # BGE skip_left_clamp
    skip_left_clamp_placeholder = len(code)
    inst, imm = encode_bge(0x0000)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(3, 0x0000)  # Clamp to 0
    code.append(inst)
    code.append(imm)
    skip_left_clamp_label = len(code)
    skip_left_label = len(code)
    
    # Handle RIGHT movement
    inst, imm = encode_mov_imm(1, 0x0021)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_load(2, 1))  # R2 = RIGHT button state
    inst, imm = encode_cmp_imm(2, 0)
    code.append(inst)
    code.append(imm)
    skip_right_placeholder = len(code)
    inst, imm = encode_beq(0x0000)
    code.append(inst)
    code.append(imm)
    # RIGHT pressed: ADD R3, #2 (move right 2 pixels)
    inst, imm = encode_add_imm(3, 2)
    code.append(inst)
    code.append(imm)
    # Clamp to screen: if R3 > 315 (320-5), set to 315
    inst, imm = encode_cmp_imm(3, 315)
    code.append(inst)
    code.append(imm)
    skip_right_clamp_placeholder = len(code)
    inst, imm = encode_blt(0x0000)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(3, 315)  # Clamp to max
    code.append(inst)
    code.append(imm)
    skip_right_clamp_label = len(code)
    skip_right_label = len(code)
    
    # Handle jumping
    # Load A button state from WRAM[0x0022]
    inst, imm = encode_mov_imm(1, 0x0022)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_load(2, 1))  # R2 = A button state
    # Check if button pressed (R2 != 0)
    inst, imm = encode_cmp_imm(2, 0)
    code.append(inst)
    code.append(imm)
    # BEQ skip_jump (if not pressed, skip)
    skip_jump_placeholder = len(code)
    inst, imm = encode_beq(0x0000)
    code.append(inst)
    code.append(imm)
    # Check if on ground (R4 == FLOOR_Y - 10)
    inst, imm = encode_cmp_imm(4, FLOOR_Y - 10)
    code.append(inst)
    code.append(imm)
    # BNE skip_jump (if not on ground, can't jump)
    skip_jump_ground_placeholder = len(code)
    inst, imm = encode_bne(0x0000)
    code.append(inst)
    code.append(imm)
    # Check velocity (R5 == 0 means on ground)
    inst, imm = encode_cmp_imm(5, 0)
    code.append(inst)
    code.append(imm)
    # BNE skip_jump (if already jumping/falling, can't jump)
    skip_jump_vel_placeholder = len(code)
    inst, imm = encode_bne(0x0000)
    code.append(inst)
    code.append(imm)
    # Start jump: R5 = -8 (negative velocity = upward)
    inst, imm = encode_mov_imm(5, 0xFFF8)  # -8 in two's complement
    code.append(inst)
    code.append(imm)
    skip_jump_vel_label = len(code)
    skip_jump_ground_label = len(code)
    skip_jump_label = len(code)
    # Update last frame button state: R6 = R2
    code.append(encode_mov_reg(6, 2))
    
    # Handle shooting
    inst, imm = encode_mov_imm(1, 0x0023)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_load(2, 1))  # R2 = B button state
    inst, imm = encode_cmp_imm(2, 0)
    code.append(inst)
    code.append(imm)
    skip_shoot_placeholder = len(code)
    inst, imm = encode_beq(0x0000)
    code.append(inst)
    code.append(imm)
    # Check if button just pressed
    # For now, just shoot if pressed (we'll add edge detection later)
    # Find inactive bullet slot and create bullet
    # We'll implement bullet creation later - for now skip
    skip_shoot_label = len(code)
    code.append(encode_mov_reg(7, 2))  # Update last frame button state
    
    # Apply gravity and update Y position
    # If not on ground, apply gravity
    inst, imm = encode_cmp_imm(4, FLOOR_Y - 10)  # Check if on ground
    code.append(inst)
    code.append(imm)
    # BEQ skip_gravity (if on ground)
    skip_gravity_placeholder = len(code)
    inst, imm = encode_beq(0x0000)
    code.append(inst)
    code.append(imm)
    # Apply gravity: ADD R5, #1 (increase downward velocity)
    inst, imm = encode_add_imm(5, 1)
    code.append(inst)
    code.append(imm)
    # Clamp velocity to max fall speed (e.g., 8)
    inst, imm = encode_cmp_imm(5, 8)
    code.append(inst)
    code.append(imm)
    # BLT skip_velocity_clamp
    skip_velocity_clamp_placeholder = len(code)
    inst, imm = encode_blt(0x0000)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(5, 8)  # Clamp to max
    code.append(inst)
    code.append(imm)
    skip_velocity_clamp_label = len(code)
    skip_gravity_label = len(code)
    
    # Update Y position: ADD R4, R5
    code.append(encode_add_reg(4, 5))
    # Check if hit ground
    inst, imm = encode_cmp_imm(4, FLOOR_Y - 10)
    code.append(inst)
    code.append(imm)
    # BLT skip_ground_hit
    skip_ground_hit_placeholder = len(code)
    inst, imm = encode_blt(0x0000)
    code.append(inst)
    code.append(imm)
    # Hit ground: set Y to floor, clear velocity
    inst, imm = encode_mov_imm(4, FLOOR_Y - 10)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(5, 0x0000)
    code.append(inst)
    code.append(imm)
    skip_ground_hit_label = len(code)
    
    # Calculate player tile position for rendering
    
    # Player Y tile (top half)
    # MOV R1, R4 (player Y)
    code.append(encode_mov_reg(1, 4))
    # SHR R1, #3 (divide by 8)
    inst, imm = encode_shr_imm(1, 3)
    code.append(inst)
    code.append(imm)
    # Calculate: R1 * 32 = R1 * 2^5 = shift left 5 times
    # MOV R2, R1
    code.append(encode_mov_reg(2, 1))
    # SHL R2, #5 (multiply by 32)
    inst, imm = encode_shl_imm(2, 5)
    code.append(inst)
    code.append(imm)
    # Now R2 = player_y_tile * 32
    
    # Player X tile
    # MOV R1, R3 (player X)
    code.append(encode_mov_reg(1, 3))
    # SHR R1, #3 (divide by 8)
    inst, imm = encode_shr_imm(1, 3)
    code.append(inst)
    code.append(imm)
    # ADD R2, R1 (R2 = player_y_tile * 32 + player_x_tile)
    code.append(encode_add_reg(2, 1))
    # SHL R2, #1 (multiply by 2 for byte address)
    inst, imm = encode_shl_imm(2, 1)
    code.append(inst)
    code.append(imm)
    # Now R2 = tilemap byte address for player top tile
    
    # Set VRAM address to player position
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))  # Write address low (R2)
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR + 1)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0000)  # Address high = 0 (tilemap is in first 2KB)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Write player top tile (tile 3, palette 1)
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_DATA)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x03)  # Tile 3, low byte
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(2, 0x10)  # Palette 1 << 4 = 0x10, high byte
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Write player bottom tile (tile 4, palette 1) - one row down
    # Address += 32 * 2 = 64
    # ADD R2, #64
    inst, imm = encode_add_imm(2, 64)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR + 1)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0000)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_DATA)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x04)  # Tile 4, low byte
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    inst, imm = encode_mov_imm(2, 0x10)  # Palette 1, high byte
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # Delay loop for frame sync
    delay_loop_start = len(code)
    inst, imm = encode_mov_imm(7, 0x2B5C)  # Delay counter
    code.append(inst)
    code.append(imm)
    delay_sub_label = len(code)
    inst, imm = encode_sub_imm(7, 1)
    code.append(inst)
    code.append(imm)
    delay_bne_placeholder = len(code)
    inst, imm = encode_bne(0x0000)
    code.append(inst)
    code.append(imm)
    delay_loop_end = len(code)
    
    # Jump back to game loop
    anim_branch_placeholder = len(code)
    inst, imm = encode_jmp_rel(0x0000)
    code.append(inst)
    code.append(imm)
    
    # Calculate branch offsets
    byte_positions = []
    current_byte = DEFAULT_ENTRY_OFFSET
    for word in code:
        if isinstance(word, tuple):
            byte_positions.append((current_byte, current_byte + 4))
            current_byte += 4
        else:
            byte_positions.append((current_byte, current_byte + 2))
            current_byte += 2
    
    def fix_branch_offset(placeholder_idx, target_idx, is_jmp=False):
        if is_jmp:
            jmp_pc = byte_positions[placeholder_idx][0]
            branch_end = jmp_pc + 4
        else:
            branch_inst_addr = byte_positions[placeholder_idx][0]
            branch_end = branch_inst_addr + 4
        target_byte = byte_positions[target_idx][0]
        offset_signed = target_byte - branch_end
        
        if offset_signed < 0:
            offset = (offset_signed & 0xFFFF) | 0xFFFF0000
            offset = offset & 0xFFFF
        else:
            offset = offset_signed & 0xFFFF
        
        if placeholder_idx + 1 < len(code):
            code[placeholder_idx + 1] = offset
    
    # Fix all branch offsets
    fix_branch_offset(fill_bg_loop_bne_placeholder, fill_bg_loop_start)
    fix_branch_offset(fill_floor_loop_bne_placeholder, fill_floor_loop_start)
    fix_branch_offset(skip_left_placeholder, skip_left_label)
    fix_branch_offset(skip_left_clamp_placeholder, skip_left_clamp_label)
    fix_branch_offset(skip_right_placeholder, skip_right_label)
    fix_branch_offset(skip_right_clamp_placeholder, skip_right_clamp_label)
    fix_branch_offset(skip_jump_placeholder, skip_jump_label)
    fix_branch_offset(skip_jump_ground_placeholder, skip_jump_ground_label)
    fix_branch_offset(skip_jump_vel_placeholder, skip_jump_vel_label)
    fix_branch_offset(skip_shoot_placeholder, skip_shoot_label)
    fix_branch_offset(skip_gravity_placeholder, skip_gravity_label)
    fix_branch_offset(skip_velocity_clamp_placeholder, skip_velocity_clamp_label)
    fix_branch_offset(skip_ground_hit_placeholder, skip_ground_hit_label)
    fix_branch_offset(delay_bne_placeholder, delay_sub_label)
    fix_branch_offset(anim_branch_placeholder, game_loop_start, is_jmp=True)
    
    # Pack code into bytes
    code_bytes = bytearray()
    for word in code:
        if isinstance(word, tuple):
            code_bytes.extend(struct.pack('<H', word[0]))
            code_bytes.extend(struct.pack('<H', word[1]))
        else:
            code_bytes.extend(struct.pack('<H', word))
    
    # Build ROM header
    rom_size = len(code_bytes)
    header = bytearray(32)
    struct.pack_into('<L', header, 0, ROM_MAGIC)
    struct.pack_into('<H', header, 4, ROM_VERSION)
    struct.pack_into('<L', header, 6, rom_size)
    struct.pack_into('<H', header, 10, DEFAULT_ENTRY_BANK)
    struct.pack_into('<H', header, 12, DEFAULT_ENTRY_OFFSET)
    struct.pack_into('<H', header, 14, 0)  # Mapper flags
    struct.pack_into('<L', header, 16, 0)  # Checksum
    
    # Build complete ROM
    rom_data = bytearray()
    rom_data.extend(header)
    rom_data.extend(code_bytes)
    
    # Write ROM file
    with open(output_file, 'wb') as f:
        f.write(rom_data)
    
    print(f"Created platform game ROM: {output_file}")
    print(f"  Size: {len(rom_data)} bytes")
    print(f"  Entry point: Bank {DEFAULT_ENTRY_BANK:02X}, Offset {DEFAULT_ENTRY_OFFSET:04X}")
    print(f"  Code size: {len(code_bytes)} bytes")


if __name__ == "__main__":
    if len(sys.argv) > 1:
        output_file = sys.argv[1]
    else:
        output_file = "platform.rom"
    
    print("Platform Game ROM Builder")
    print("=" * 50)
    print()
    
    create_platform_rom(output_file)
    print()
    print("To use this ROM:")
    print(f"  py src_python/main.py {output_file}")
