"""
test_interactive.py - Interactive test to explore the emulator
Run with: python src_python/test_interactive.py
"""

import sys
import os

# Add src_python to path
script_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(script_dir)
src_python_path = os.path.join(project_root, "src_python")
sys.path.insert(0, src_python_path)
    project_root = os.path.dirname(script_dir)
src_python_path = os.path.join(project_root, "src_python")
sys.path.insert(0, src_python_path)

import config
import cpu
import memory

print("=" * 60)
print("Fantasy Console Emulator - Interactive Test")
print("=" * 60)
print()
print("This lets you explore the emulator state interactively.")
print("Type 'help' for commands, 'quit' to exit.")
print()

# Initialize
cpu.cpu_reset()
memory.memory_reset()
emu = config.emulator

def print_cpu_state():
    """Print current CPU state"""
    print(f"\nCPU State:")
    print(f"  Registers: R0={emu.cpu.r0:04X} R1={emu.cpu.r1:04X} R2={emu.cpu.r2:04X} R3={emu.cpu.r3:04X}")
    print(f"             R4={emu.cpu.r4:04X} R5={emu.cpu.r5:04X} R6={emu.cpu.r6:04X} R7={emu.cpu.r7:04X}")
    print(f"  PC: {emu.cpu.pc_bank:02X}:{emu.cpu.pc_offset:04X}")
    print(f"  SP: {emu.cpu.sp:04X}")
    print(f"  Flags: {emu.cpu.flags:04X} (Z={cpu.cpu_get_flag(0)} N={cpu.cpu_get_flag(1)} C={cpu.cpu_get_flag(2)} V={cpu.cpu_get_flag(3)} I={cpu.cpu_get_flag(4)})")
    print(f"  Cycles: {emu.cpu.cycles}")

def print_memory_info():
    """Print memory info"""
    print(f"\nMemory State:")
    print(f"  ROM Size: {emu.memory.rom_size} bytes")
    print(f"  ROM Banks: {emu.memory.rom_banks}")
    print(f"  Mapper: {emu.memory.mapper_type}")

def help_command():
    """Show help"""
    print("\nCommands:")
    print("  cpu          - Show CPU state")
    print("  mem          - Show memory info")
    print("  setr <reg> <value> - Set register (e.g., 'setr r0 0x1234')")
    print("  read <bank> <offset> - Read memory (e.g., 'read 0 0x1000')")
    print("  write <bank> <offset> <value> - Write memory (e.g., 'write 0 0x1000 0x42')")
    print("  reset        - Reset CPU and memory")
    print("  help         - Show this help")
    print("  quit         - Exit")

# Main loop
while True:
    try:
        cmd = input("\n> ").strip().lower()
        
        if cmd == "quit" or cmd == "q" or cmd == "exit":
            print("Goodbye!")
            break
        
        elif cmd == "help" or cmd == "h":
            help_command()
        
        elif cmd == "cpu":
            print_cpu_state()
        
        elif cmd == "mem":
            print_memory_info()
        
        elif cmd == "reset":
            cpu.cpu_reset()
            memory.memory_reset()
            print("✓ CPU and memory reset")
        
        elif cmd.startswith("setr "):
            parts = cmd.split()
            if len(parts) == 3:
                reg = parts[1].upper()
                try:
                    value = int(parts[2], 0)  # Auto-detect hex/dec
                    if reg == "R0": emu.cpu.r0 = value & 0xFFFF
                    elif reg == "R1": emu.cpu.r1 = value & 0xFFFF
                    elif reg == "R2": emu.cpu.r2 = value & 0xFFFF
                    elif reg == "R3": emu.cpu.r3 = value & 0xFFFF
                    elif reg == "R4": emu.cpu.r4 = value & 0xFFFF
                    elif reg == "R5": emu.cpu.r5 = value & 0xFFFF
                    elif reg == "R6": emu.cpu.r6 = value & 0xFFFF
                    elif reg == "R7": emu.cpu.r7 = value & 0xFFFF
                    else:
                        print(f"Unknown register: {reg}")
                        continue
                    print(f"✓ Set {reg} = {value:04X}")
                except ValueError:
                    print("Invalid value (use decimal or 0x hex)")
            else:
                print("Usage: setr <reg> <value> (e.g., 'setr r0 0x1234')")
        
        elif cmd.startswith("read "):
            parts = cmd.split()
            if len(parts) == 3:
                try:
                    bank = int(parts[1], 0)
                    offset = int(parts[2], 0)
                    value = memory.memory_read8(bank, offset)
                    print(f"Memory[{bank:02X}:{offset:04X}] = {value:02X} ({value})")
                except ValueError:
                    print("Invalid address (use decimal or 0x hex)")
            else:
                print("Usage: read <bank> <offset> (e.g., 'read 0 0x1000')")
        
        elif cmd.startswith("write "):
            parts = cmd.split()
            if len(parts) == 4:
                try:
                    bank = int(parts[1], 0)
                    offset = int(parts[2], 0)
                    value = int(parts[3], 0)
                    memory.memory_write8(bank, offset, value)
                    print(f"✓ Wrote {value:02X} to [{bank:02X}:{offset:04X}]")
                except ValueError:
                    print("Invalid address/value (use decimal or 0x hex)")
            else:
                print("Usage: write <bank> <offset> <value> (e.g., 'write 0 0x1000 0x42')")
        
        else:
            print(f"Unknown command: {cmd}")
            print("Type 'help' for available commands")
    
    except KeyboardInterrupt:
        print("\n\nGoodbye!")
        break
    except Exception as e:
        print(f"Error: {e}")
        import traceback
        traceback.print_exc()

