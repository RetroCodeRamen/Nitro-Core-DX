"""
debug_session.py - Interactive debugging session we can use together
Run this and we can debug issues in real-time!
"""

import sys
import os
script_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(script_dir)
src_python_path = os.path.join(project_root, 'src_python')
sys.path.insert(0, src_python_path)

import config
import cpu
import memory

# Initialize
cpu.cpu_reset()
memory.memory_reset()
emu = config.emulator

print("="*70)
print("Python Emulator - Interactive Debug Session")
print("="*70)
print()
print("This is how we can debug together:")
print("  - I can see exactly what's happening")
print("  - You can show me errors and I'll understand them")
print("  - We can test changes immediately")
print()
print("Current State:")
print("-"*70)

def show_state():
    """Show current emulator state"""
    print(f"\nCPU:")
    print(f"  R0={emu.cpu.r0:04X} R1={emu.cpu.r1:04X} R2={emu.cpu.r2:04X} R3={emu.cpu.r3:04X}")
    print(f"  R4={emu.cpu.r4:04X} R5={emu.cpu.r5:04X} R6={emu.cpu.r6:04X} R7={emu.cpu.r7:04X}")
    print(f"  PC={emu.cpu.pc_bank:02X}:{emu.cpu.pc_offset:04X} SP={emu.cpu.sp:04X}")
    print(f"  Flags={emu.cpu.flags:04X} Cycles={emu.cpu.cycles}")
    print(f"\nMemory:")
    print(f"  WRAM size: {len(config.memory_wram)} bytes")
    print(f"  ROM size: {emu.memory.rom_size} bytes")
    print(f"  Sample WRAM[0x1000] = {config.memory_wram[0x1000]:02X}")

show_state()

print("\n" + "="*70)
print("Example: Let's test some operations")
print("="*70)

# Example operations
print("\n1. Setting CPU registers:")
emu.cpu.r0 = 0x1234
emu.cpu.r1 = 0x5678
print(f"   Set R0 = 0x1234, R1 = 0x5678")

print("\n2. Writing to memory:")
memory.memory_write8(0, 0x1000, 0x42)
memory.memory_write8(0, 0x2000, 0xAB)
print(f"   Wrote 0x42 to [0:0x1000]")
print(f"   Wrote 0xAB to [0:0x2000]")

print("\n3. Reading back:")
val1 = memory.memory_read8(0, 0x1000)
val2 = memory.memory_read8(0, 0x2000)
print(f"   Read [0:0x1000] = {val1:02X}")
print(f"   Read [0:0x2000] = {val2:02X}")

print("\n4. Updated state:")
show_state()

print("\n" + "="*70)
print("If something breaks, Python will tell us exactly what!")
print("="*70)
print()
print("For example, if we try to access invalid memory:")
print("  (In QB64: 'Syntax error' - no help)")
print("  (In Python: 'IndexError: list index out of range' - clear!)")
print()

# Demonstrate error handling
try:
    # This would work
    test_val = memory.memory_read8(0, 0x1000)
    print(f"✓ Valid access works: {test_val:02X}")
    
    # Show what happens with invalid access (but catch it)
    print("\nTrying invalid bank (999):")
    invalid_val = memory.memory_read8(999, 0x1000)
    print(f"  Result: {invalid_val:02X} (returns 0 for invalid)")
    
except Exception as e:
    print(f"✗ Error: {type(e).__name__}: {e}")
    print("  But we caught it and can fix it!")

print("\n" + "="*70)
print("✓ Python makes debugging together SO much easier!")
print("="*70)
print()
print("You can:")
print("  1. Run this script: python src_python/debug_session.py")
print("  2. Show me any errors you get")
print("  3. I can see exactly what's wrong and help fix it")
print("  4. Test changes immediately")
print()
print("Ready to continue? I can complete the full conversion now!")

