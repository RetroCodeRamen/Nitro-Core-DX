"""
rom_builder.py - Simple tool to create test ROMs
Creates minimal ROMs for testing the emulator
"""

import struct
import sys

# ROM Header Constants
ROM_MAGIC = 0x46434F4E  # "FCON" (Fantasy CONsole)
ROM_VERSION = 1
ROM_HEADER_SIZE = 32

# Default entry point (bank 1, offset 0x8000 - standard ROM location)
DEFAULT_ENTRY_BANK = 1
DEFAULT_ENTRY_OFFSET = 0x8000


def create_simple_test_rom(output_file="test.rom"):
    """
    Create a minimal test ROM that:
    1. Sets up PPU
    2. Writes a simple pattern
    3. Enables background
    4. Loops forever
    """
    
    # For now, create a ROM with just a header and minimal code
    # This is a placeholder - actual ROM would need assembled code
    
    # ROM code (placeholder - would be assembled instructions)
    # For now, just create a valid header with minimal data
    
    rom_data = bytearray()
    
    # Header
    header = struct.pack(
        '<LHHLL',  # Little-endian: long, short, short, long, long
        ROM_MAGIC,           # Magic number
        ROM_VERSION,          # Version
        1024,                 # ROM size (1KB for now - minimal)
        DEFAULT_ENTRY_BANK,   # Entry point bank
        DEFAULT_ENTRY_OFFSET # Entry point offset
    )
    
    # Mapper flags and checksum (separate)
    mapper_flags = struct.pack('<H', 0)  # 2 bytes
    checksum = struct.pack('<L', 0)      # 4 bytes
    
    # Reserved bytes (16 bytes)
    reserved = b'\x00' * 16
    
    rom_data.extend(header)
    rom_data.extend(reserved)
    
    # ROM code area (minimal - just NOPs for now)
    # In a real ROM, this would be assembled instructions
    code_size = 1024 - ROM_HEADER_SIZE
    rom_data.extend(b'\x00' * code_size)
    
    # Write ROM file
    with open(output_file, 'wb') as f:
        f.write(rom_data)
    
    print(f"Created test ROM: {output_file}")
    print(f"  Size: {len(rom_data)} bytes")
    print(f"  Entry point: {DEFAULT_ENTRY_BANK:02X}:{DEFAULT_ENTRY_OFFSET:04X}")
    print()
    print("Note: This ROM contains only NOPs (no actual code yet).")
    print("      Implement CPU instructions first to run actual code.")


def create_header_only_rom(output_file="header_test.rom"):
    """Create a ROM with just a valid header (for testing ROM loading)"""
    
    rom_data = bytearray()
    
    # Header
    header = struct.pack(
        '<LHHLL',
        ROM_MAGIC,
        ROM_VERSION,
        32,  # Just header size
        DEFAULT_ENTRY_BANK,
        DEFAULT_ENTRY_OFFSET
    )
    
    # Mapper flags and checksum
    mapper_flags = struct.pack('<H', 0)
    checksum = struct.pack('<L', 0)
    
    # Reserved bytes
    reserved = b'\x00' * 16
    
    rom_data.extend(header)
    rom_data.extend(reserved)
    
    with open(output_file, 'wb') as f:
        f.write(rom_data)
    
    print(f"Created header-only ROM: {output_file}")
    print(f"  Size: {len(rom_data)} bytes (header only)")


if __name__ == "__main__":
    if len(sys.argv) > 1:
        output_file = sys.argv[1]
    else:
        output_file = "test.rom"
    
    print("ROM Builder - Creating test ROM")
    print("=" * 50)
    print()
    
    create_simple_test_rom(output_file)
    print()
    print("To use this ROM:")
    print(f"  python src_python/main.py {output_file}")

