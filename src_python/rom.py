"""
rom.py - ROM file loading and header parsing
Python version - maintains BASIC-like simplicity
"""

import config
import memory


def rom_load_file(filename):
    """Load ROM file from disk"""
    import ui
    emu = config.emulator
    
    ui.logger.info(f"Loading ROM file: {filename}", "ROM")
    
    try:
        # Read ROM file
        with open(filename, 'rb') as f:
            # Read header (32 bytes)
            header_data = f.read(32)
            
            if len(header_data) < 32:
                print(f"ERROR: ROM file too small (only {len(header_data)} bytes)")
                return False
            
            # Parse header
            header = config.ROMHeader()
            header.magic = int.from_bytes(header_data[0:4], 'little')
            header.version = int.from_bytes(header_data[4:6], 'little')
            header.rom_size = int.from_bytes(header_data[6:10], 'little')
            header.entry_point_bank = int.from_bytes(header_data[10:12], 'little')
            header.entry_point_offset = int.from_bytes(header_data[12:14], 'little')
            header.mapper_flags = int.from_bytes(header_data[14:16], 'little')
            header.checksum = int.from_bytes(header_data[16:20], 'little')
            
            # Read reserved bytes
            for i in range(8):
                header.reserved[i] = int.from_bytes(header_data[20 + i*2:22 + i*2], 'little')
            
            # Validate magic number
            if header.magic != config.ROM_MAGIC:
                print(f"ERROR: Invalid ROM magic number: {hex(header.magic)}")
                print(f"Expected: {hex(config.ROM_MAGIC)}")
                return False
            
            # Validate version
            if header.version > config.ROM_VERSION:
                print(f"WARNING: ROM version {header.version} > supported version {config.ROM_VERSION}")
            
            # Validate ROM size
            if header.rom_size <= 0 or header.rom_size > 16777216:  # Max 16MB
                print(f"ERROR: Invalid ROM size: {header.rom_size} bytes")
                return False
            
            # Read ROM data
            rom_data = f.read(header.rom_size)
            
            if len(rom_data) < header.rom_size:
                print(f"ERROR: ROM file truncated (expected {header.rom_size}, got {len(rom_data)})")
                return False
        
        # Store header in emulator state
        emu.rom = header
        
        # Load ROM into memory system
        memory.memory_load_rom(rom_data, header.rom_size, header.mapper_flags & 0xF)
        
        # Set CPU entry point
        emu.cpu.pc_bank = header.entry_point_bank
        emu.cpu.pc_offset = header.entry_point_offset
        emu.cpu.pbr = header.entry_point_bank  # PBR is used for instruction fetches
        
        import ui
        ui.logger.info(f"ROM loaded: {header.rom_size} bytes, Entry: {header.entry_point_bank:02X}:{header.entry_point_offset:04X}", "ROM")
        ui.logger.debug(f"ROM header: magic={header.magic:08X}, version={header.version}, mapper={header.mapper_flags & 0xF}", "ROM")
        
        # TODO: Validate checksum if implemented
        
        return True
        
    except FileNotFoundError:
        print(f"ERROR: ROM file not found: {filename}")
        return False
    except Exception as e:
        print(f"ERROR: Failed to load ROM: {type(e).__name__}: {e}")
        import traceback
        traceback.print_exc()
        return False


def rom_validate_header(filename):
    """Validate ROM header (without loading full ROM)"""
    try:
        with open(filename, 'rb') as f:
            header_data = f.read(32)
            
            if len(header_data) < 32:
                return False
            
            magic = int.from_bytes(header_data[0:4], 'little')
            version = int.from_bytes(header_data[4:6], 'little')
            rom_size = int.from_bytes(header_data[6:10], 'little')
            
            # Check magic
            if magic != config.ROM_MAGIC:
                return False
            
            # Check size is reasonable
            if rom_size <= 0 or rom_size > 16777216:
                return False
            
            return True
            
    except Exception:
        return False


def rom_get_info():
    """Get ROM info (for display/debugging)"""
    emu = config.emulator
    return {
        'magic': emu.rom.magic,
        'version': emu.rom.version,
        'rom_size': emu.rom.rom_size,
        'entry_bank': emu.rom.entry_point_bank,
        'entry_offset': emu.rom.entry_point_offset
    }

