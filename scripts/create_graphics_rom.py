"""
create_graphics_rom.py - Create a ROM that displays a 5x5 pixel box
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
PPU_REG_BG0_CONTROL = config.PPU_REG_BG0_CONTROL
PPU_REG_BG0_SCROLLX = config.PPU_REG_BG0_SCROLLX
PPU_REG_BG0_SCROLLY = config.PPU_REG_BG0_SCROLLY
PPU_REG_VRAM_ADDR = config.PPU_REG_VRAM_ADDR
PPU_REG_VRAM_DATA = config.PPU_REG_VRAM_DATA

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

def encode_add_reg(reg1, reg2):
    """ADD R1, R2"""
    return (0x2 << 12) | (0x0 << 8) | (reg1 << 4) | reg2

def encode_add_imm(reg, value):
    """ADD R, #imm"""
    return (0x2 << 12) | (0x1 << 8) | (reg << 4) | 0x0, value

def encode_cmp_imm(reg, value):
    """CMP R, #imm - Mode 1 (immediate)"""
    # CMP immediate: opcode 0xC000, mode 0x1 (immediate), reg1=reg, reg2=0
    # This encodes as 0xC100 | (reg << 4) = 0xC130 for R3
    # The decoder now checks if it's a branch (mode matches AND reg1=0, reg2=0)
    # If not a branch, it's CMP, so this should work correctly now
    return (0xC << 12) | (0x1 << 8) | (reg << 4) | 0x0, value

def encode_sub_imm(reg, value):
    """SUB R, #imm"""
    return (0x3 << 12) | (0x1 << 8) | (reg << 4) | 0x0, value

def encode_bne(offset):
    """BNE - Branch if not equal"""
    # BNE: opcode 0xC, mode 0x2 (relative), reg1=0, reg2=0
    # BNE = 0xC200 (not 0xC100, which is BEQ)
    return (0xC << 12) | (0x2 << 8) | 0x0, offset  # BNE = 0xC200

def encode_bgt(offset):
    """BGT - Branch if greater than"""
    # BGT: opcode 0xC, mode 0x3, reg1=0, reg2=0
    return (0xC << 12) | (0x3 << 8) | 0x0, offset  # BGT = 0xC300

def encode_blt(offset):
    """BLT - Branch if less than"""
    # BLT: opcode 0xC, mode 0x4, reg1=0, reg2=0
    return (0xC << 12) | (0x4 << 8) | 0x0, offset  # BLT = 0xC400

def encode_bge(offset):
    """BGE - Branch if greater or equal"""
    # BGE: opcode 0xC, mode 0x5, reg1=0, reg2=0
    return (0xC << 12) | (0x5 << 8) | 0x0, offset  # BGE = 0xC500

def encode_jmp_rel(offset):
    """JMP rel"""
    return (0xD << 12) | (0x1 << 8) | 0x0, offset

def encode_nop():
    return 0x0000


def create_graphics_rom(output_file="graphics.rom"):
    """
    Create a ROM that displays a 5x5 pixel box
    """
    
    code = []
    
    # R0 = I/O base (0x8000)
    inst, imm = encode_mov_imm(0, MEM_IO_PPU_BASE)
    code.append(inst)
    code.append(imm)
    
    # Enable BG0: Write 0x01 to PPU_REG_BG0_CONTROL (0x8008)
    # R1 = 0x01 (enable BG0, 8x8 tiles - bit 0 = enable, bit 1 = 0 for 8x8)
    inst, imm = encode_mov_imm(1, 0x01)
    code.append(inst)
    code.append(imm)
    
    # R2 = 0x8008 (BG0_CONTROL address)
    inst, imm = encode_mov_imm(2, 0x8008)
    code.append(inst)
    code.append(imm)
    
    # MOV [R2], R1 - Write enable byte to BG0_CONTROL
    code.append(encode_mov_store(2, 1))
    
    # Write tilemap entry FIRST at offset 0
    # (tile_map_base defaults to 0, so tilemap[0] must be at offset 0)
    # We'll use tile index 1, so tile data will be at offset 64 (one tile = 64 bytes)
    # Set VRAM address to 0
    # R1 = 0x800A (VRAM_ADDR low)
    inst, imm = encode_mov_imm(1, 0x800A)
    code.append(inst)
    code.append(imm)
    
    # R2 = 0 (address low byte)
    inst, imm = encode_mov_imm(2, 0x00)
    code.append(inst)
    code.append(imm)
    
    # MOV [R1], R2 - Set VRAM address low
    code.append(encode_mov_store(1, 2))
    
    # R1 = 0x800B (VRAM_ADDR high)
    inst, imm = encode_mov_imm(1, 0x800B)
    code.append(inst)
    code.append(imm)
    
    # MOV [R1], R2 - Set VRAM address high (still 0)
    code.append(encode_mov_store(1, 2))
    
    # R1 = 0x800C (VRAM_DATA) - set this BEFORE writing tilemap entry
    inst, imm = encode_mov_imm(1, 0x800C)
    code.append(inst)
    code.append(imm)
    
    # Write tilemap entry: tile index 1, palette 0
    # (Tile index 1 means tile data starts at offset 64)
    # IMPORTANT: Write tile index FIRST (low byte), then attributes (high byte)
    # R2 = 0x01 (low byte - tile index 1)
    inst, imm = encode_mov_imm(2, 0x01)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))  # Write low byte to VRAM[0000]
    
    # R2 = 0x00 (high byte - palette 0, no flips)
    inst, imm = encode_mov_imm(2, 0x00)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))  # Write high byte to VRAM[0001]
    
    # Now write tile data starting at offset 32 (tile index 1, 32 bytes per tile for 4bpp)
    # Set VRAM address to 32 (0x20)
    # R1 = 0x800A (VRAM_ADDR low)
    inst, imm = encode_mov_imm(1, 0x800A)
    code.append(inst)
    code.append(imm)
    
    # R2 = 0x20 (address low byte - 32 = 0x20 for tile index 1)
    inst, imm = encode_mov_imm(2, 0x20)
    code.append(inst)
    code.append(imm)
    
    # MOV [R1], R2 - Set VRAM address low
    code.append(encode_mov_store(1, 2))
    
    # R1 = 0x800B (VRAM_ADDR high)
    inst, imm = encode_mov_imm(1, 0x800B)
    code.append(inst)
    code.append(imm)
    
    # R2 = 0x00 (address high byte)
    inst, imm = encode_mov_imm(2, 0x00)
    code.append(inst)
    code.append(imm)
    
    # MOV [R1], R2 - Set VRAM address high
    code.append(encode_mov_store(1, 2))
    
    # R1 = 0x800C (VRAM_DATA)
    inst, imm = encode_mov_imm(1, 0x800C)
    code.append(inst)
    code.append(imm)
    
    # Now write tile data (5x5 box pattern)
    # Tile format: 4bpp, 8x8 tile = 4 bytes per row = 32 bytes total
    # Each byte = 2 pixels (4 bits each)
    # For a 5x5 box, we'll use color 1 (palette index 1) for the box
    # Pattern: first 5 rows, first 5 columns = color 1, rest = 0 (transparent)
    
    # R1 = 0x800C (VRAM_DATA)
    inst, imm = encode_mov_imm(1, 0x800C)
    code.append(inst)
    code.append(imm)
    
    # Write tile data row by row (4 bytes per row for 4bpp)
    # For 5 pixels in a row (4bpp = 2 pixels per byte):
    # Byte 0: pixels 0-1 = 0x11 (both color 1)
    # Byte 1: pixels 2-3 = 0x11 (both color 1)
    # Byte 2: pixel 4-5 = 0x10 (pixel 4 = 1, pixel 5 = 0)
    # Byte 3: pixels 6-7 = 0x00 (transparent)
    
    tile_data = [
        0x11, 0x11, 0x10, 0x00,  # Row 0: 5 pixels (4 bytes)
        0x11, 0x11, 0x10, 0x00,  # Row 1: 5 pixels
        0x11, 0x11, 0x10, 0x00,  # Row 2: 5 pixels
        0x11, 0x11, 0x10, 0x00,  # Row 3: 5 pixels
        0x11, 0x11, 0x10, 0x00,  # Row 4: 5 pixels
        0x00, 0x00, 0x00, 0x00,  # Row 5: empty
        0x00, 0x00, 0x00, 0x00,  # Row 6: empty
        0x00, 0x00, 0x00, 0x00,  # Row 7: empty
    ]
    
    # Write tile data (32 bytes for 8x8 4bpp tile)
    for byte_val in tile_data:
        # R2 = byte value
        inst, imm = encode_mov_imm(2, byte_val)
        code.append(inst)
        code.append(imm)
        
        # MOV [R1], R2 - Write byte to VRAM_DATA (auto-increments VRAM address)
        code.append(encode_mov_store(1, 2))
    
    # Initialize animation variables (ONCE at startup, NOT in the loop)
    # R3 = scroll X counter (starts at 0)
    inst, imm = encode_mov_imm(3, 0x0000)
    code.append(inst)
    code.append(imm)
    
    # R4 = scroll Y counter (starts at 0)
    inst, imm = encode_mov_imm(4, 0x0000)
    code.append(inst)
    code.append(imm)
    
    # R5 = X direction (0 = right, 1 = left) - starts at 0 (right)
    inst, imm = encode_mov_imm(5, 0x0000)
    code.append(inst)
    code.append(imm)
    
    # R6 = Y direction (0 = down, 1 = up) - starts at 0 (down)
    inst, imm = encode_mov_imm(6, 0x0000)
    code.append(inst)
    code.append(imm)
    
    # R7 = palette index (starts at 0, will increment on bounce)
    inst, imm = encode_mov_imm(7, 0x0000)
    code.append(inst)
    code.append(imm)
    
    # Label: animation_loop
    # IMPORTANT: This label marks where the loop STARTS - after all initialization
    # The initialization code above (R3=0, R4=0, R5=0, R6=0) runs ONCE at startup
    # The loop will jump back to HERE, not to the initialization code
    animation_loop_start = len(code)
    
    # Update scroll X based on direction
    # Check if moving right (R5 == 0)
    # CMP R5, #0
    inst, imm = encode_cmp_imm(5, 0)
    code.append(inst)
    code.append(imm)
    
    # BNE skip_x_inc (if not 0, skip increment)
    x_inc_skip_placeholder = len(code)
    inst, imm = encode_bne(0x0000)  # Placeholder
    code.append(inst)
    code.append(imm)
    
    # Check if R3 would go below 0 BEFORE subtracting (if R3 < 1, bounce)
    # When scroll_x = 0, box is at left edge. When scroll_x = 315, box is at right edge.
    # Check if R3 < 1 (would go to 0 or negative)
    inst, imm = encode_cmp_imm(3, 1)
    code.append(inst)
    code.append(imm)
    
    # BGE skip_x_bounce (if >= 1, safe to subtract)
    x_bounce_skip_placeholder = len(code)
    inst, imm = encode_bge(0x0000)  # Placeholder - BGE = branch if greater or equal
    code.append(inst)
    code.append(imm)
    
    # Would go below 0, bounce: reverse direction (R5 = 1) and clamp to 0
    inst, imm = encode_mov_imm(5, 0x0001)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(3, 0x0000)  # Clamp to 0 (left edge)
    code.append(inst)
    code.append(imm)
    
    # Change color on bounce: increment palette (R7) and wrap at 16
    # ADD R7, #1
    inst, imm = encode_add_imm(7, 1)
    code.append(inst)
    code.append(imm)
    # CMP R7, #16 (check if >= 16)
    inst, imm = encode_cmp_imm(7, 16)
    code.append(inst)
    code.append(imm)
    # BLT skip_palette_wrap (if < 16, skip wrap)
    palette_wrap_skip_placeholder = len(code)
    inst, imm = encode_blt(0x0000)  # Placeholder
    code.append(inst)
    code.append(imm)
    # Wrap: R7 = 0
    inst, imm = encode_mov_imm(7, 0x0000)
    code.append(inst)
    code.append(imm)
    # Label: skip_palette_wrap
    skip_palette_wrap_label = len(code)
    
    # Update tilemap palette: write to VRAM[1] (tilemap entry high byte)
    # First calculate palette value: R7 << 4 (palette in upper 4 bits)
    # Use R2 as temp: R2 = R7, then shift left 4 times
    code.append(encode_mov_reg(2, 7))  # R2 = R7
    code.append(encode_add_reg(2, 2))  # R2 = R2 * 2
    code.append(encode_add_reg(2, 2))  # R2 = R2 * 4
    code.append(encode_add_reg(2, 2))  # R2 = R2 * 8
    code.append(encode_add_reg(2, 2))  # R2 = R2 * 16 = R7 << 4
    
    # Set VRAM address to 1 (save R2 first in R0 temporarily)
    code.append(encode_mov_reg(0, 2))  # R0 = R2 (save palette value)
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0001)  # Address low = 1
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))  # Write address low
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR + 1)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0000)  # Address high = 0
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))  # Write address high
    
    # Restore palette value and write to VRAM_DATA
    code.append(encode_mov_reg(2, 0))  # R2 = R0 (restore palette value)
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_DATA)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))  # Write palette byte
    
    # JMP skip_x_add (skip the ADD instruction)
    x_add_skip_placeholder = len(code)
    inst, imm = encode_jmp_rel(0x0000)  # Placeholder
    code.append(inst)
    code.append(imm)
    
    # Label: skip_x_bounce (safe to add)
    skip_x_bounce_label = len(code)
    
    # Moving right: SUB R3, #1 (1 pixel per frame)
    # Note: Increasing scroll moves tile LEFT, so to move RIGHT we DECREASE scroll
    inst, imm = encode_sub_imm(3, 1)
    code.append(inst)
    code.append(imm)
    
    # Label: skip_x_add (label is at the position after ADD R3)
    skip_x_add_label = len(code)  # Label is after ADD (at the JMP position)
    
    # JMP skip_x_dec (skip left-moving code)
    x_dec_skip_placeholder = len(code)
    inst, imm = encode_jmp_rel(0x0000)  # Placeholder
    code.append(inst)
    code.append(imm)
    
    # Label: skip_x_inc (moving left)
    skip_x_inc_label = len(code)
    
    # Check if R3 >= 0x013B BEFORE adding (if so, we'll exceed max, so bounce)
    # Max scroll X = 315 (0x013B) for right edge
    inst, imm = encode_cmp_imm(3, 0x013B)
    code.append(inst)
    code.append(imm)
    
    # BLT skip_x_bounce2 (if less than 0x013B, safe to add)
    x_bounce2_skip_placeholder = len(code)
    inst, imm = encode_blt(0x0000)  # Placeholder - will jump past bounce code
    code.append(inst)
    code.append(imm)
    
    # Would exceed max, bounce: reverse direction (R5 = 0) and clamp to max
    inst, imm = encode_mov_imm(5, 0x0000)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(3, 0x013B)  # Clamp to max (315 = right edge)
    code.append(inst)
    code.append(imm)
    
    # Change color on bounce (same code as left bounce)
    inst, imm = encode_add_imm(7, 1)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_cmp_imm(7, 16)
    code.append(inst)
    code.append(imm)
    palette_wrap_skip_placeholder2 = len(code)
    inst, imm = encode_blt(0x0000)  # Placeholder
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(7, 0x0000)
    code.append(inst)
    code.append(imm)
    skip_palette_wrap_label2 = len(code)
    # Calculate palette and update tilemap (same as first bounce)
    code.append(encode_mov_reg(2, 7))
    code.append(encode_add_reg(2, 2))
    code.append(encode_add_reg(2, 2))
    code.append(encode_add_reg(2, 2))
    code.append(encode_add_reg(2, 2))
    code.append(encode_mov_reg(0, 2))  # Save palette in R0
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0001)
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
    code.append(encode_mov_reg(2, 0))  # Restore palette from R0
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_DATA)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # JMP skip_x_bounce2_end (skip the ADD instruction)
    x_bounce2_end_jmp_placeholder = len(code)
    inst, imm = encode_jmp_rel(0x0000)  # Placeholder
    code.append(inst)
    code.append(imm)
    
    # Label: skip_x_bounce2 (safe to add)
    skip_x_bounce2_label = len(code)
    
    # Not bouncing, add normally
    # Moving left: ADD R3, #1 (1 pixel per frame)
    # Note: Decreasing scroll moves tile RIGHT, so to move LEFT we INCREASE scroll
    inst, imm = encode_add_imm(3, 1)
    code.append(inst)
    code.append(imm)
    
    # Label: skip_x_bounce2_end
    skip_x_bounce2_end_label = len(code)
    
    # Label: skip_x_dec (this label points to the CMP R6, #0 instruction)
    # Update scroll Y based on direction (move 5 pixels per frame, same as X)
    # Check if moving down (R6 == 0)
    skip_x_dec_label = len(code)  # Label is at the CMP instruction position (before appending)
    inst, imm = encode_cmp_imm(6, 0)
    code.append(inst)  # First entry: instruction
    code.append(imm)   # Second entry: immediate
    
    # BNE skip_y_inc
    y_inc_skip_placeholder = len(code)
    inst, imm = encode_bne(0x0000)  # Placeholder
    code.append(inst)
    code.append(imm)
    
    # Check if R4 < 1 BEFORE subtracting (if so, we'll go negative, so bounce)
    # When scroll_y = 0, box is at top edge. When scroll_y = 195, box is at bottom edge.
    inst, imm = encode_cmp_imm(4, 1)
    code.append(inst)
    code.append(imm)
    
    # BGE skip_y_bounce (if >= 1, safe to subtract)
    y_bounce_skip_placeholder = len(code)
    inst, imm = encode_bge(0x0000)  # Placeholder
    code.append(inst)
    code.append(imm)
    
    # Would go below 0, bounce: reverse direction (R6 = 1) and clamp to 0
    inst, imm = encode_mov_imm(6, 0x0001)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(4, 0x0000)  # Clamp to 0 (top edge)
    code.append(inst)
    code.append(imm)
    
    # Change color on bounce
    inst, imm = encode_add_imm(7, 1)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_cmp_imm(7, 16)
    code.append(inst)
    code.append(imm)
    palette_wrap_skip_placeholder3 = len(code)
    inst, imm = encode_blt(0x0000)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(7, 0x0000)
    code.append(inst)
    code.append(imm)
    skip_palette_wrap_label3 = len(code)
    # Calculate palette and update tilemap (same as first bounce)
    code.append(encode_mov_reg(2, 7))
    code.append(encode_add_reg(2, 2))
    code.append(encode_add_reg(2, 2))
    code.append(encode_add_reg(2, 2))
    code.append(encode_add_reg(2, 2))
    code.append(encode_mov_reg(0, 2))  # Save palette in R0
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0001)
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
    code.append(encode_mov_reg(2, 0))  # Restore palette from R0
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_DATA)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # JMP skip_y_add (skip the SUB instruction)
    y_add_skip_placeholder = len(code)
    inst, imm = encode_jmp_rel(0x0000)  # Placeholder
    code.append(inst)
    code.append(imm)
    
    # Label: skip_y_bounce (safe to subtract)
    skip_y_bounce_label = len(code)
    
    # Moving down: SUB R4, #1 (1 pixel per frame)
    # Note: Increasing scroll moves tile UP, so to move DOWN we DECREASE scroll
    inst, imm = encode_sub_imm(4, 1)
    code.append(inst)
    code.append(imm)
    
    # Label: skip_y_add
    skip_y_add_label = len(code)
    
    # JMP skip_y_dec (skip up-moving code)
    y_dec_skip_placeholder = len(code)
    inst, imm = encode_jmp_rel(0x0000)  # Placeholder
    code.append(inst)
    code.append(imm)
    
    # Label: skip_y_inc (moving up)
    skip_y_inc_label = len(code)
    
    # Check if R4 >= 0x00C3 BEFORE adding (if so, we'll exceed max, so bounce)
    # Max scroll Y = 195 (0x00C3) for bottom edge
    inst, imm = encode_cmp_imm(4, 0x00C3)
    code.append(inst)
    code.append(imm)
    
    # BLT skip_y_bounce2 (if less than 0x00C3, safe to add)
    y_bounce2_skip_placeholder = len(code)
    inst, imm = encode_blt(0x0000)  # Placeholder - will jump past bounce code
    code.append(inst)
    code.append(imm)
    
    # Would exceed max, bounce: reverse direction (R6 = 0) and clamp to max
    inst, imm = encode_mov_imm(6, 0x0000)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(4, 0x00C3)  # Clamp to max (195 = bottom edge)
    code.append(inst)
    code.append(imm)
    
    # Change color on bounce
    inst, imm = encode_add_imm(7, 1)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_cmp_imm(7, 16)
    code.append(inst)
    code.append(imm)
    palette_wrap_skip_placeholder4 = len(code)
    inst, imm = encode_blt(0x0000)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(7, 0x0000)
    code.append(inst)
    code.append(imm)
    skip_palette_wrap_label4 = len(code)
    # Calculate palette and update tilemap (same as first bounce)
    code.append(encode_mov_reg(2, 7))
    code.append(encode_add_reg(2, 2))
    code.append(encode_add_reg(2, 2))
    code.append(encode_add_reg(2, 2))
    code.append(encode_add_reg(2, 2))
    code.append(encode_mov_reg(0, 2))  # Save palette in R0
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_ADDR)
    code.append(inst)
    code.append(imm)
    inst, imm = encode_mov_imm(2, 0x0001)
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
    code.append(encode_mov_reg(2, 0))  # Restore palette from R0
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_VRAM_DATA)
    code.append(inst)
    code.append(imm)
    code.append(encode_mov_store(1, 2))
    
    # JMP skip_y_bounce2_end (skip the ADD instruction)
    y_bounce2_end_jmp_placeholder = len(code)
    inst, imm = encode_jmp_rel(0x0000)  # Placeholder
    code.append(inst)
    code.append(imm)
    
    # Label: skip_y_bounce2 (safe to add)
    skip_y_bounce2_label = len(code)
    
    # Not bouncing, add normally
    # Moving up: ADD R4, #1 (1 pixel per frame)
    # Note: Decreasing scroll moves tile DOWN, so to move UP we INCREASE scroll
    inst, imm = encode_add_imm(4, 1)
    code.append(inst)
    code.append(imm)
    
    # Label: skip_y_bounce2_end
    skip_y_bounce2_end_label = len(code)
    
    # Label: skip_y_dec
    skip_y_dec_label = len(code)
    
    # Write scroll X to PPU
    # R1 = PPU_REG_BG0_SCROLLX
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_BG0_SCROLLX)
    code.append(inst)
    code.append(imm)
    
    # MOV [R1], R3 - Write scroll X low byte
    code.append(encode_mov_store(1, 3))
    
    # R1 = PPU_REG_BG0_SCROLLX + 1
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_BG0_SCROLLX + 1)
    code.append(inst)
    code.append(imm)
    
    # R2 = 0x00 (scroll X high byte)
    inst, imm = encode_mov_imm(2, 0x00)
    code.append(inst)
    code.append(imm)
    
    # MOV [R1], R2 - Write scroll X high byte
    code.append(encode_mov_store(1, 2))
    
    # Write scroll Y to PPU
    # R1 = PPU_REG_BG0_SCROLLY
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_BG0_SCROLLY)
    code.append(inst)
    code.append(imm)
    
    # MOV [R1], R4 - Write scroll Y low byte
    code.append(encode_mov_store(1, 4))
    
    # R1 = PPU_REG_BG0_SCROLLY + 1
    inst, imm = encode_mov_imm(1, MEM_IO_PPU_BASE + PPU_REG_BG0_SCROLLY + 1)
    code.append(inst)
    code.append(imm)
    
    # MOV [R1], R2 - Write scroll Y high byte (R2 still has 0x00)
    code.append(encode_mov_store(1, 2))
    
    # Delay loop to synchronize with frame rate
    # Use R7 as delay counter (R7 is free)
    # Target: ~44,667 cycles per frame
    # Animation loop takes ~200-300 cycles, so delay needs ~44,400 cycles
    # Each delay loop iteration: SUB (2 cycles) + BNE (2 cycles if taken) = ~4 cycles
    # So we need ~11,100 iterations: 0x2B5C
    delay_loop_start = len(code)
    
    # Initialize delay counter (only once, but we'll reset it each frame)
    # R7 = delay counter value (0x2B5C = 11,100 iterations)
    inst, imm = encode_mov_imm(7, 0x2B5C)
    code.append(inst)
    code.append(imm)
    
    # Delay loop: SUB R7, #1
    delay_sub_label = len(code)
    inst, imm = encode_sub_imm(7, 1)
    code.append(inst)
    code.append(imm)
    
    # BNE delay_sub_label (if R7 != 0, loop back to SUB)
    delay_bne_placeholder = len(code)
    inst, imm = encode_bne(0x0000)  # Placeholder - will jump back to delay_sub_label
    code.append(inst)
    code.append(imm)
    
    # Delay loop end label
    delay_loop_end = len(code)
    
    # Jump back to animation_loop
    # We'll calculate this offset after packing too
    anim_branch_placeholder = len(code)
    inst, imm = encode_jmp_rel(0x0000)  # Placeholder offset
    code.append(inst)
    code.append(imm)
    
    # Pack code into bytes first to calculate offsets
    code_bytes_temp = bytearray()
    code_with_placeholders = []
    for word in code:
        if isinstance(word, tuple):
            code_with_placeholders.append(word)
            code_bytes_temp.extend(struct.pack('<H', word[0]))
            code_bytes_temp.extend(struct.pack('<H', word[1]))
        else:
            code_with_placeholders.append(word)
            code_bytes_temp.extend(struct.pack('<H', word))
    
    # Calculate actual byte positions for branch offsets
    # First, calculate where each instruction is in bytes
    byte_positions = []
    current_byte = DEFAULT_ENTRY_OFFSET
    for word in code:
        if isinstance(word, tuple):
            byte_positions.append((current_byte, current_byte + 4))  # Instruction + immediate = 4 bytes
            current_byte += 4
        else:
            byte_positions.append((current_byte, current_byte + 2))  # Single instruction = 2 bytes
            current_byte += 2
    
    # Helper function to calculate and fix branch offset
    def fix_branch_offset(placeholder_idx, target_idx, is_jmp=False):
        """Fix a branch/jump offset"""
        if is_jmp:
            # For JMP rel, the CPU:
            # 1. Fetches JMP instruction at PC (e.g., 0x81FC)
            # 2. cpu_fetch_instruction increments PC by 2 (to 0x81FE)
            # 3. Reads offset from PC (0x81FE)
            # 4. Increments PC by 2 to consume offset (to 0x8200)
            # 5. Adds offset to PC: new_pc = 0x8200 + offset
            # So: new_pc = jmp_pc + 4 + offset
            # Therefore: offset = target_pc - (jmp_pc + 4)
            jmp_pc = byte_positions[placeholder_idx][0]  # JMP instruction address
            branch_end = jmp_pc + 4  # PC after consuming offset word
        else:
            # For branch instructions, placeholder_idx points to the instruction word
            # The offset word is at placeholder_idx + 1
            # After the CPU fetches the instruction (PC += 2) and reads the offset (PC += 2),
            # the PC is at instruction_address + 4
            branch_inst_addr = byte_positions[placeholder_idx][0]  # Instruction address
            branch_end = branch_inst_addr + 4  # PC after reading offset word
        target_byte = byte_positions[target_idx][0]
        offset_signed = target_byte - branch_end
        
        # Debug output for animation loop JMP
        if is_jmp and placeholder_idx == anim_branch_placeholder:
            print(f"DEBUG: JMP offset calculation:")
            print(f"  JMP at byte position: {byte_positions[placeholder_idx][0]:04X}")
            print(f"  branch_end (PC after JMP, before offset): {branch_end:04X}")
            print(f"  target_byte (animation_loop_start): {target_byte:04X}")
            print(f"  offset_signed: {offset_signed:+d} (0x{offset_signed & 0xFFFFFFFF:08X})")
        
        # Validate offset doesn't jump to invalid ROM region
        new_pc = (branch_end + offset_signed) & 0xFFFF
        if new_pc < DEFAULT_ENTRY_OFFSET:
            print(f"WARNING: Branch/jump from 0x{branch_end:04X} to 0x{target_byte:04X} would result in PC=0x{new_pc:04X} (invalid, below 0x8000)")
            print(f"  This indicates the ROM code is too long or offset calculation is wrong")
            # Clamp to valid region (though this shouldn't happen with correct ROM layout)
            offset_signed = DEFAULT_ENTRY_OFFSET - branch_end
        
        # Convert signed offset to 16-bit signed integer
        if offset_signed < 0:
            # Negative offset: sign extend to 16 bits
            offset = (offset_signed & 0xFFFF) | 0xFFFF0000
            offset = offset & 0xFFFF  # Keep only 16 bits
        else:
            # Positive offset: just use as-is (max 32767)
            offset = offset_signed & 0xFFFF
        
        # Update the immediate value entry (placeholder_idx + 1) directly
        # This preserves the structure (instruction + immediate as separate entries)
        # and doesn't change byte positions of subsequent entries
        if placeholder_idx + 1 < len(code):
            code[placeholder_idx + 1] = offset
        else:
            # Fallback: convert to tuple if structure is unexpected
            if isinstance(code[placeholder_idx], tuple):
                code[placeholder_idx] = (code[placeholder_idx][0], offset)
            else:
                code[placeholder_idx] = (code[placeholder_idx], offset)
        
        # Debug output for final offset
        if is_jmp and placeholder_idx == anim_branch_placeholder:
            print(f"  Final offset: 0x{offset:04X} (signed: {offset if offset < 32768 else offset - 65536:+d})")
            print(f"  Expected new PC: 0x{new_pc:04X}")
    
    # Fix all branch offsets
    # IMPORTANT: Fix animation loop LAST, because fixing other branches changes
    # the byte positions, which affects the animation loop JMP offset
    fix_branch_offset(x_inc_skip_placeholder, skip_x_inc_label)
    fix_branch_offset(x_bounce_skip_placeholder, skip_x_bounce_label)
    fix_branch_offset(x_add_skip_placeholder, skip_x_add_label)
    # Debug output for x_dec_skip JMP
    fix_branch_offset(x_dec_skip_placeholder, skip_x_dec_label, is_jmp=True)
    fix_branch_offset(x_bounce2_skip_placeholder, skip_x_bounce2_label)
    fix_branch_offset(x_bounce2_end_jmp_placeholder, skip_x_bounce2_end_label)
    fix_branch_offset(y_inc_skip_placeholder, skip_y_inc_label)
    fix_branch_offset(y_bounce_skip_placeholder, skip_y_bounce_label)
    fix_branch_offset(y_add_skip_placeholder, skip_y_add_label)
    fix_branch_offset(y_dec_skip_placeholder, skip_y_dec_label)
    fix_branch_offset(y_bounce2_skip_placeholder, skip_y_bounce2_label)
    fix_branch_offset(y_bounce2_end_jmp_placeholder, skip_y_bounce2_end_label)
    
    # Fix palette wrap branch offsets
    fix_branch_offset(palette_wrap_skip_placeholder, skip_palette_wrap_label)
    fix_branch_offset(palette_wrap_skip_placeholder2, skip_palette_wrap_label2)
    fix_branch_offset(palette_wrap_skip_placeholder3, skip_palette_wrap_label3)
    fix_branch_offset(palette_wrap_skip_placeholder4, skip_palette_wrap_label4)
    
    # Fix delay loop branch offset
    fix_branch_offset(delay_bne_placeholder, delay_sub_label)
    
    # Animation loop branch: from anim_branch_placeholder to animation_loop_start
    # IMPORTANT: Recalculate byte positions AFTER fixing all other branches,
    # because fixing branches changes the code size and byte positions
    byte_positions = []
    current_byte = DEFAULT_ENTRY_OFFSET
    for word in code:
        if isinstance(word, tuple):
            byte_positions.append((current_byte, current_byte + 4))  # Instruction + immediate = 4 bytes
            current_byte += 4
        else:
            byte_positions.append((current_byte, current_byte + 2))  # Single instruction = 2 bytes
            current_byte += 2
    
    # Now fix the animation loop offset with the updated byte positions
    fix_branch_offset(anim_branch_placeholder, animation_loop_start, is_jmp=True)
    
    # Pack code into bytes
    code_bytes = bytearray()
    for word in code:
        if isinstance(word, tuple):
            code_bytes.extend(struct.pack('<H', word[0]))
            code_bytes.extend(struct.pack('<H', word[1]))
        else:
            code_bytes.extend(struct.pack('<H', word))
    
    # Calculate ROM size (data only, not including header)
    rom_size = len(code_bytes)
    
    # Build header (32 bytes)
    header = bytearray(32)
    struct.pack_into('<L', header, 0, ROM_MAGIC)
    struct.pack_into('<H', header, 4, ROM_VERSION)
    struct.pack_into('<L', header, 6, rom_size)
    struct.pack_into('<H', header, 10, DEFAULT_ENTRY_BANK)
    struct.pack_into('<H', header, 12, DEFAULT_ENTRY_OFFSET)
    struct.pack_into('<H', header, 14, 0)  # Mapper flags
    struct.pack_into('<L', header, 16, 0)  # Checksum
    for i in range(12):
        header[20 + i] = 0  # Reserved
    
    # Build complete ROM
    rom_data = bytearray()
    rom_data.extend(header)
    rom_data.extend(code_bytes)
    
    # Write ROM file
    with open(output_file, 'wb') as f:
        f.write(rom_data)
    
    print(f"Created graphics ROM: {output_file}")
    print(f"  Size: {len(rom_data)} bytes")
    print(f"  Entry point: Bank {DEFAULT_ENTRY_BANK:02X}, Offset {DEFAULT_ENTRY_OFFSET:04X}")
    print(f"  Code size: {len(code_bytes)} bytes")
    print()
    print("This ROM displays a 5x5 pixel box that moves diagonally across the screen.")


if __name__ == "__main__":
    if len(sys.argv) > 1:
        output_file = sys.argv[1]
    else:
        output_file = "graphics.rom"
    
    print("Graphics ROM Builder - Animated 5x5 Box")
    print("=" * 50)
    print()
    
    create_graphics_rom(output_file)
    print()
    print("To use this ROM:")
    print(f"  python3 src_python/main.py {output_file}")

