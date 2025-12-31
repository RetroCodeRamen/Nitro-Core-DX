"""
debug.py - Debugging utilities and logging
Python version - maintains BASIC-like simplicity
"""

import config
import cpu
import memory
import rom


def debug_reset():
    """Initialize debug system"""
    emu = config.emulator
    emu.debug_mode = False
    emu.paused = False
    emu.step_mode = False


def debug_print_cpu_state():
    """Print CPU register state"""
    emu = config.emulator
    print("=== CPU State ===")
    print(f"R0={emu.cpu.r0:04X} R1={emu.cpu.r1:04X}")
    print(f"R2={emu.cpu.r2:04X} R3={emu.cpu.r3:04X}")
    print(f"R4={emu.cpu.r4:04X} R5={emu.cpu.r5:04X}")
    print(f"R6={emu.cpu.r6:04X} R7={emu.cpu.r7:04X}")
    print(f"PC={emu.cpu.pc_bank:02X}:{emu.cpu.pc_offset:04X}")
    print(f"SP={emu.cpu.sp:04X}")
    print(f"PBR={emu.cpu.pbr:02X} DBR={emu.cpu.dbr:02X}")
    print(f"Flags={emu.cpu.flags:04X} (Z={cpu.cpu_get_flag(0)} N={cpu.cpu_get_flag(1)} C={cpu.cpu_get_flag(2)} V={cpu.cpu_get_flag(3)} I={cpu.cpu_get_flag(4)})")
    print(f"Cycles={emu.cpu.cycles}")
    print()


def debug_print_memory_info():
    """Print memory map info"""
    emu = config.emulator
    print("=== Memory Info ===")
    print(f"ROM Size={emu.memory.rom_size} bytes")
    print(f"ROM Banks={emu.memory.rom_banks}")
    print(f"Mapper Type={emu.memory.mapper_type}")
    print()


def debug_print_ppu_state():
    """Print PPU state"""
    emu = config.emulator
    print("=== PPU State ===")
    print(f"BG0: Enabled={emu.ppu.bg0.enabled} Scroll=({emu.ppu.bg0.scroll_x},{emu.ppu.bg0.scroll_y})")
    print(f"BG1: Enabled={emu.ppu.bg1.enabled} Scroll=({emu.ppu.bg1.scroll_x},{emu.ppu.bg1.scroll_y})")
    print(f"Framebuffer: Enabled={emu.ppu.frame_buffer_enabled}")
    print(f"Portrait Mode={emu.ppu.portrait_mode}")
    print(f"VBlank Active={emu.ppu.vblank_active}")
    print()


def debug_print_apu_state():
    """Print APU state"""
    emu = config.emulator
    print("=== APU State ===")
    for ch in range(config.APU_NUM_CHANNELS):
        channel = config.apu_channels[ch]
        print(f"Ch{ch}: Enabled={channel.enabled} Freq={channel.frequency} Vol={channel.volume} Wave={channel.waveform_type}")
    print(f"Master Volume={emu.apu.master_volume}")
    print()


def debug_print_input_state():
    """Print input state"""
    emu = config.emulator
    print("=== Input State ===")
    print(f"Controller1={emu.input.controller1:04X}")
    print(f"Latch={emu.input.controller1_latch:04X} Shift={emu.input.controller1_shift}")
    print()


def debug_print_rom_info():
    """Print ROM info"""
    info = rom.rom_get_info()
    print("=== ROM Info ===")
    print(f"Magic={info['magic']:08X}")
    print(f"Version={info['version']}")
    print(f"Size={info['rom_size']} bytes")
    print(f"Entry Point={info['entry_bank']:02X}:{info['entry_offset']:04X}")
    print()


def debug_print_full_state():
    """Print full emulator state"""
    import os
    os.system('clear' if os.name != 'nt' else 'cls')  # Clear screen
    
    debug_print_rom_info()
    debug_print_cpu_state()
    debug_print_memory_info()
    debug_print_ppu_state()
    debug_print_apu_state()
    debug_print_input_state()
    
    emu = config.emulator
    print(f"Frame={emu.frame_count}")
    if emu.frame_time > 0:
        print(f"FPS={1.0 / emu.frame_time:.1f}")


def debug_log(message):
    """Log message (placeholder - could write to file)"""
    emu = config.emulator
    if emu.debug_mode:
        print(f"[DEBUG] {message}")


def debug_toggle():
    """Toggle debug mode"""
    emu = config.emulator
    emu.debug_mode = not emu.debug_mode
    if emu.debug_mode:
        print("Debug mode ON")
    else:
        print("Debug mode OFF")


def debug_toggle_pause():
    """Toggle pause"""
    emu = config.emulator
    emu.paused = not emu.paused
    if emu.paused:
        print("PAUSED")
    else:
        print("RESUMED")


def debug_toggle_step_mode():
    """Toggle step mode"""
    emu = config.emulator
    emu.step_mode = not emu.step_mode
    if emu.step_mode:
        print("Step mode ON - press any key to step")
    else:
        print("Step mode OFF")

