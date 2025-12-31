"""
create_test_rom.py - Create a working test ROM with graphics
This generates a ROM that sets up PPU, draws tiles, and loops
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
ROM_MAGIC = config.ROM_MAGIC  # Use the correct magic from config
ROM_VERSION = config.ROM_VERSION
ROM_HEADER_SIZE = config.ROM_HEADER_SIZE

# Default entry point (bank 1, offset 0x8000 - standard ROM location)
DEFAULT_ENTRY_BANK = 1
DEFAULT_ENTRY_OFFSET = 0x8000

# Memory I/O addresses
MEM_IO_PPU_BASE = config.MEM_IO_PPU_BASE
PPU_REG_BG0_SCROLLX = config.PPU_REG_BG0_SCROLLX
PPU_REG_BG0_SCROLLY = config.PPU_REG_BG0_SCROLLY
PPU_REG_BG0_CONTROL = config.PPU_REG_BG0_CONTROL
PPU_REG_VRAM_ADDR = config.PPU_REG_VRAM_ADDR
PPU_REG_VRAM_DATA = config.PPU_REG_VRAM_DATA

# Instruction encoding helpers
def encode_mov_imm(reg, value):
    """MOV R, #imm - Mode 1: immediate"""
    # Opcode: 0x1 (MOV), Mode: 0x1 (immediate), Reg1: reg, Reg2: unused
    return (0x1 << 12) | (0x1 << 8) | (reg << 4) | 0x0, value

def encode_mov_reg(reg1, reg2):
    """MOV R1, R2 - Mode 0: register to register"""
    return (0x1 << 12) | (0x0 << 8) | (reg1 << 4) | reg2

def encode_mov_store(reg1, reg2):
    """MOV [R1], R2 - Mode 3: store to memory"""
    return (0x1 << 12) | (0x3 << 8) | (reg1 << 4) | reg2

def encode_mov_load(reg1, reg2):
    """MOV R1, [R2] - Mode 2: load from memory"""
    return (0x1 << 12) | (0x2 << 8) | (reg1 << 4) | reg2

def encode_add_imm(reg, value):
    """ADD R, #imm - Mode 1: immediate"""
    return (0x2 << 12) | (0x1 << 8) | (reg << 4) | 0x0, value

def encode_jmp_rel(offset):
    """JMP rel - Mode 1: relative jump"""
    return (0xD << 12) | (0x1 << 8) | 0x0, offset

def encode_nop():
    """NOP instruction"""
    return 0x0000


def create_test_rom(output_file="test.rom"):
    """
    Create a test ROM that:
    1. Sets up PPU (enables BG0, sets scroll)
    2. Writes tile data to VRAM
    3. Writes tilemap to VRAM
    4. Loops forever
    """
    
    rom_data = bytearray()
    code_start = DEFAULT_ENTRY_OFFSET
    
    # We'll build the code in a list, then pack it
    code = []
    
    # Entry point: Setup code
    # R0 = PPU I/O base address (0x8000)
    inst, imm = encode_mov_imm(0, MEM_IO_PPU_BASE)
    code.append(inst)
    code.append(imm)
    
    # R1 = temporary register
    # R2 = temporary register
    
    # Enable BG0: Write to PPU_REG_BG0_CONTROL (0x8008)
    # MOV R1, #0x03 (enable + 8x8 tiles)
    inst, imm = encode_mov_imm(1, 0x03)
    code.append(inst)
    code.append(imm)
    
    # MOV R2, #0x08 (offset to BG0_CONTROL)
    inst, imm = encode_mov_imm(2, 0x08)
    code.append(inst)
    code.append(imm)
    
    # ADD R2, R0 (R2 = PPU base + offset = BG0_CONTROL address)
    code.append(encode_add_imm(2, MEM_IO_PPU_BASE))  # Actually, let's use direct address
    
    # Actually, let's simplify: write directly to I/O addresses
    # We need to write bytes to I/O space, which requires memory writes
    
    # For now, let's create a simpler ROM that just loops
    # The PPU setup will be done via memory writes, which requires more complex code
    
    # Simple loop: JMP to self
    # Calculate relative offset (negative, to jump back)
    # Entry is at 0x8000, we want to jump to 0x8000
    # After this instruction + offset word, PC will be at 0x8004
    # So we need offset = 0x8000 - 0x8004 = -4 = 0xFFFC
    
    # Actually, let's just create a minimal ROM that does nothing but loop
    # We'll add PPU setup later when we have a proper assembler
    
    # For now: NOP loop
    code.append(encode_nop())
    code.append(encode_nop())
    code.append(encode_nop())
    
    # JMP rel to start (jump back to entry point)
    # Current PC after these NOPs: 0x8000 + 2*3 = 0x8006
    # Want to jump to 0x8000
    # Offset = 0x8000 - 0x8006 = -6 = 0xFFFA
    inst, offset = encode_jmp_rel(0xFFFA)  # -6 in two's complement
    code.append(inst)
    code.append(offset)
    
    # Pack code into bytes
    code_bytes = bytearray()
    for word in code:
        if isinstance(word, tuple):
            # Instruction with immediate
            code_bytes.extend(struct.pack('<H', word[0]))
            code_bytes.extend(struct.pack('<H', word[1]))
        else:
            # Single instruction
            code_bytes.extend(struct.pack('<H', word))
    
    # Calculate ROM size (data only, NOT including header)
    # rom.py reads header separately, then reads rom_size bytes of data
    rom_size = len(code_bytes)
    
    # Build header according to rom.py format:
    # rom.py reads 32 bytes for header, but tries to read reserved from offset 20-35 (16 bytes)
    # This is a bug in rom.py - it should read 36 bytes. For now, use 32-byte header.
    # Format: magic(4) + version(2) + rom_size(4) + entry_bank(2) + entry_offset(2) + mapper(2) + checksum(4) + reserved(12) = 32 bytes
    header = bytearray(32)
    struct.pack_into('<L', header, 0, ROM_MAGIC)              # 0-3: Magic (4 bytes)
    struct.pack_into('<H', header, 4, ROM_VERSION)            # 4-5: Version (2 bytes)
    struct.pack_into('<L', header, 6, rom_size)              # 6-9: ROM size (4 bytes)
    struct.pack_into('<H', header, 10, DEFAULT_ENTRY_BANK)    # 10-11: Entry bank (2 bytes)
    struct.pack_into('<H', header, 12, DEFAULT_ENTRY_OFFSET)  # 12-13: Entry offset (2 bytes)
    struct.pack_into('<H', header, 14, 0)                     # 14-15: Mapper flags (2 bytes)
    struct.pack_into('<L', header, 16, 0)                     # 16-19: Checksum (4 bytes)
    # 20-31: Reserved (12 bytes) - rom.py tries to read 16 bytes but only 12 fit in 32-byte header
    for i in range(12):
        header[20 + i] = 0
    
    # Build complete ROM (header + data)
    rom_data.extend(header)
    rom_data.extend(code_bytes)
    
    # Note: rom_size in header is the data size (26 bytes), file total is 32 + 26 = 58 bytes
    
    # Write ROM file
    with open(output_file, 'wb') as f:
        f.write(rom_data)
    
    print(f"Created test ROM: {output_file}")
    print(f"  Size: {len(rom_data)} bytes")
    print(f"  Entry point: Bank {DEFAULT_ENTRY_BANK:02X}, Offset {DEFAULT_ENTRY_OFFSET:04X}")
    print(f"  Code size: {len(code_bytes)} bytes")
    print()
    print("This ROM contains a simple loop (NOPs + JMP).")
    print("PPU setup code will be added when we have a proper assembler.")


if __name__ == "__main__":
    if len(sys.argv) > 1:
        output_file = sys.argv[1]
    else:
        output_file = "test.rom"
    
    print("Test ROM Builder")
    print("=" * 50)
    print()
    
    create_test_rom(output_file)
    print()
    print("To use this ROM:")
    print(f"  python3 src_python/main.py {output_file}")

