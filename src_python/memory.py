"""
memory.py - Memory management and banked addressing
Python version - maintains BASIC-like simplicity
"""

import config


def memory_reset():
    """Initialize memory system"""
    emu = config.emulator
    
    # Clear WRAM
    config.memory_wram[:] = [0] * len(config.memory_wram)
    
    # Clear extended WRAM
    config.memory_wram_extended[:] = [0] * len(config.memory_wram_extended)
    
    # Clear ROM data
    emu.memory.rom_data = b''
    emu.memory.rom_size = 0
    emu.memory.rom_banks = 0
    
    # Reset mapper
    emu.memory.mapper_type = config.MAPPER_LOROM
    emu.memory.mapper_flags = 0


def memory_read8(bank, offset):
    """Read 8-bit value from memory (24-bit address via bank:offset)"""
    import ui
    emu = config.emulator
    
    # Clamp offset to valid range
    offset = offset & 0xFFFF
    
    # Map address based on bank
    if bank == 0:
        # Bank 0: WRAM (0x0000-0x7FFF) or I/O (0x8000-0xFFFF)
        if offset < 0x8000:
            # Work RAM
            value = config.memory_wram[offset]
            ui.logger.trace(f"WRAM[{offset:04X}] -> {value:02X}", "MEM")
            return value
        else:
            # I/O region - delegate to I/O handlers
            value = memory_read_io8(offset)
            ui.logger.trace(f"I/O Read: {bank:02X}:{offset:04X} -> {value:02X}", "MEM")
            return value
    
    elif 126 <= bank <= 127:
        # Extended WRAM banks
        addr = (bank - 126) * config.MEMORY_BANK_SIZE + offset
        if addr < 131072:
            return config.memory_wram_extended[addr]
        else:
            return 0
    
    elif 1 <= bank <= 125:
        # ROM banks (LoROM-like mapping)
        return memory_read_rom8(bank, offset)
    
    else:
        # Invalid bank
        return 0


def memory_write8(bank, offset, value):
    """Write 8-bit value to memory"""
    import ui
    emu = config.emulator
    
    # Clamp offset to valid range
    offset = offset & 0xFFFF
    value = value & 0xFF  # Ensure 8-bit
    
    # Map address based on bank
    if bank == 0:
        # Bank 0: WRAM (0x0000-0x7FFF) or I/O (0x8000-0xFFFF)
        if offset < 0x8000:
            # Work RAM - writable
            old_val = config.memory_wram[offset]
            config.memory_wram[offset] = value
            if old_val != value:
                ui.logger.trace(f"WRAM[{offset:04X}] = {value:02X} (was {old_val:02X})", "MEM")
        else:
            # I/O region - delegate to I/O handlers
            ui.logger.trace(f"I/O Write: {bank:02X}:{offset:04X} = {value:02X}", "MEM")
            memory_write_io8(offset, value)
    
    elif 126 <= bank <= 127:
        # Extended WRAM banks - writable
        addr = (bank - 126) * config.MEMORY_BANK_SIZE + offset
        if addr < 131072:
            config.memory_wram_extended[addr] = value
    
    # ROM banks (1-125) are read-only, ignore write


def memory_read16(bank, offset):
    """Read 16-bit value from memory (little-endian)"""
    low = memory_read8(bank, offset)
    high = memory_read8(bank, offset + 1)
    return low | (high << 8)


def memory_write16(bank, offset, value):
    """Write 16-bit value to memory (little-endian)"""
    # Check if writing to I/O region (8-bit registers)
    if bank == 0 and offset >= 0x8000:
        # I/O registers are 8-bit - only write low byte
        # Writing high byte to offset+1 would corrupt adjacent register
        memory_write8(bank, offset, value & 0xFF)
    else:
        # Normal memory - write both bytes
        memory_write8(bank, offset, value & 0xFF)
        memory_write8(bank, offset + 1, (value >> 8) & 0xFF)


def memory_read_rom8(bank, offset):
    """Read from ROM (internal helper)"""
    emu = config.emulator
    
    # LoROM-like mapping: ROM data appears at 0x8000-0xFFFF in each bank
    if offset < 0x8000:
        # Lower half of bank is typically mirror or empty
        return 0
    
    # Calculate ROM offset: (bank - 1) * 32KB + (offset - 0x8000)
    rom_offset = (bank - 1) * 32768 + (offset - 0x8000)
    
    # Check bounds
    if 0 <= rom_offset < emu.memory.rom_size:
        # Read from ROM bytes
        return emu.memory.rom_data[rom_offset]
    else:
        # Out of bounds - return 0
        return 0


def memory_read_io8(offset):
    """Read from I/O region (delegates to PPU/APU/Input)"""
    # Route to appropriate I/O handler
    if config.MEM_IO_PPU_BASE <= offset < config.MEM_IO_APU_BASE:
        # PPU registers
        import ppu
        return ppu.ppu_read_reg(offset - config.MEM_IO_PPU_BASE)
    elif config.MEM_IO_APU_BASE <= offset < config.MEM_IO_INPUT_BASE:
        # APU registers
        import apu
        return apu.apu_read_reg(offset - config.MEM_IO_APU_BASE)
    elif config.MEM_IO_INPUT_BASE <= offset < config.MEM_IO_TIMER_BASE:
        # Input registers
        import input
        return input.input_read_reg(offset - config.MEM_IO_INPUT_BASE)
    else:
        # Unknown I/O - return 0
        return 0


def memory_write_io8(offset, value):
    """Write to I/O region (delegates to PPU/APU/Input)"""
    # Route to appropriate I/O handler
    if config.MEM_IO_PPU_BASE <= offset < config.MEM_IO_APU_BASE:
        # PPU registers - only use low byte (8-bit registers)
        import ppu
        ppu.ppu_write_reg(offset - config.MEM_IO_PPU_BASE, value & 0xFF)
    elif config.MEM_IO_APU_BASE <= offset < config.MEM_IO_INPUT_BASE:
        # APU registers
        import apu
        apu.apu_write_reg(offset - config.MEM_IO_APU_BASE, value)
    elif config.MEM_IO_INPUT_BASE <= offset < config.MEM_IO_TIMER_BASE:
        # Input registers
        import input
        input.input_write_reg(offset - config.MEM_IO_INPUT_BASE, value)
    # Unknown I/O writes are ignored


def memory_load_rom(rom_data, rom_size, mapper_type):
    """Load ROM data into memory (called by ROM loader)"""
    emu = config.emulator
    emu.memory.rom_data = rom_data
    emu.memory.rom_size = rom_size
    emu.memory.mapper_type = mapper_type
    
    # Calculate number of banks used
    emu.memory.rom_banks = (rom_size + 32767) // 32768  # Round up to 32KB banks

