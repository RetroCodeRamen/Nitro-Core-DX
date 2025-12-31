"""
test_debug_example.py - Example showing how easy debugging is in Python
This demonstrates why Python is better for working together
"""

import sys
import os
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
print("Python Debugging Example - See How Easy It Is!")
print("=" * 60)
print()

# Initialize
cpu.cpu_reset()
memory.memory_reset()
emu = config.emulator

# Example 1: Easy inspection
print("1. Inspect CPU state (just print it!):")
print(f"   CPU R0 = {emu.cpu.r0}")
print(f"   CPU PC = {emu.cpu.pc_bank}:{emu.cpu.pc_offset}")
print(f"   CPU Flags = {bin(emu.cpu.flags)}")
print()

# Example 2: Easy modification
print("2. Modify state easily:")
emu.cpu.r0 = 0x1234
emu.cpu.r1 = 0x5678
print(f"   Set R0 = {emu.cpu.r0:04X}")
print(f"   Set R1 = {emu.cpu.r1:04X}")
print()

# Example 3: Easy memory inspection
print("3. Inspect memory:")
memory.memory_write8(0, 0x1000, 0x42)
print(f"   WRAM[0x1000] = {config.memory_wram[0x1000]:02X}")
print(f"   Direct access: config.memory_wram[0x1000] = {config.memory_wram[0x1000]:02X}")
print()

# Example 4: Easy error catching
print("4. Try something that might fail (with clear errors):")
try:
    # This will work
    value = memory.memory_read8(0, 0x1000)
    print(f"   ✓ Read successful: {value:02X}")
    
    # This would fail with a clear error
    # value = memory.memory_read8(999, 0x1000)  # Invalid bank
    # But Python tells you exactly what's wrong!
    
except Exception as e:
    print(f"   ✗ Error: {type(e).__name__}: {e}")
    print("   Python tells you EXACTLY what went wrong!")
print()

# Example 5: Easy testing
print("5. Test memory operations:")
test_values = [0x00, 0x42, 0xFF, 0x1234]
for val in test_values:
    memory.memory_write8(0, 0x3000, val & 0xFF)
    read_back = memory.memory_read8(0, 0x3000)
    status = "✓" if (read_back == (val & 0xFF)) else "✗"
    print(f"   {status} Write {val:04X} -> Read {read_back:02X}")
print()

print("=" * 60)
print("See? Much easier than QB64!")
print("=" * 60)
print()
print("Benefits:")
print("  ✓ Clear variable names (emu.cpu.r0, not Emulator.CPU.R0)")
print("  ✓ Easy inspection (just print anything)")
print("  ✓ Clear errors (TypeError: 'int' object is not subscriptable)")
print("  ✓ Interactive debugging (can run code line by line)")
print("  ✓ I can help you directly (I can see the code clearly)")
print()

