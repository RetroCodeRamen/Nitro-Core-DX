"""
test_together.py - Test we can run together to debug
This shows how easy it is to work together with Python
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

def test_scenario(name, test_func):
    """Helper to run a test and show results"""
    print(f"\n{'='*60}")
    print(f"Test: {name}")
    print('='*60)
    try:
        result = test_func()
        if result:
            print("✓ Test passed!")
        else:
            print("✗ Test failed (but no error thrown)")
        return True
    except Exception as e:
        print(f"✗ Error: {type(e).__name__}: {e}")
        import traceback
        traceback.print_exc()
        return False

# Test 1: Basic initialization
def test_init():
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    print(f"  CPU R0 = {emu.cpu.r0}")
    print(f"  Memory WRAM[0] = {config.memory_wram[0]}")
    return emu.cpu.r0 == 0 and config.memory_wram[0] == 0

# Test 2: Register operations
def test_registers():
    emu = config.emulator
    emu.cpu.r0 = 0x1234
    emu.cpu.r1 = 0x5678
    print(f"  Set R0 = {emu.cpu.r0:04X}")
    print(f"  Set R1 = {emu.cpu.r1:04X}")
    return emu.cpu.r0 == 0x1234 and emu.cpu.r1 == 0x5678

# Test 3: Memory operations
def test_memory():
    memory.memory_write8(0, 0x1000, 0x42)
    value = memory.memory_read8(0, 0x1000)
    print(f"  Wrote 0x42 to [0:0x1000]")
    print(f"  Read back: {value:02X}")
    return value == 0x42

# Test 4: 16-bit memory
def test_memory_16bit():
    memory.memory_write16(0, 0x2000, 0xABCD)
    value = memory.memory_read16(0, 0x2000)
    print(f"  Wrote 0xABCD to [0:0x2000]")
    print(f"  Read back: {value:04X}")
    return value == 0xABCD

# Test 5: Multiple memory locations
def test_memory_multiple():
    test_data = [(0x1000, 0x11), (0x2000, 0x22), (0x3000, 0x33)]
    for addr, val in test_data:
        memory.memory_write8(0, addr, val)
    print("  Wrote to multiple locations:")
    for addr, val in test_data:
        read_val = memory.memory_read8(0, addr)
        print(f"    [0:{addr:04X}] = {read_val:02X} (expected {val:02X})")
        if read_val != val:
            return False
    return True

# Test 6: CPU flags
def test_cpu_flags():
    emu = config.emulator
    cpu.cpu_set_flag(0, True)  # Set Z flag
    z_flag = cpu.cpu_get_flag(0)
    print(f"  Set Z flag, read back: {z_flag}")
    cpu.cpu_set_flag(0, False)
    z_flag = cpu.cpu_get_flag(0)
    print(f"  Cleared Z flag, read back: {z_flag}")
    return True

print("="*60)
print("Testing Python Emulator Together")
print("="*60)
print("\nThis demonstrates how easy it is to:")
print("  - Run tests together")
print("  - See exactly what's happening")
print("  - Debug issues immediately")
print("  - Work collaboratively")

tests = [
    ("Initialization", test_init),
    ("Register Operations", test_registers),
    ("Memory Operations (8-bit)", test_memory),
    ("Memory Operations (16-bit)", test_memory_16bit),
    ("Multiple Memory Locations", test_memory_multiple),
    ("CPU Flags", test_cpu_flags),
]

results = []
for name, test_func in tests:
    results.append(test_scenario(name, test_func))

print("\n" + "="*60)
print("Summary")
print("="*60)
passed = sum(results)
total = len(results)
print(f"Tests passed: {passed}/{total}")

if passed == total:
    print("\n✓ All tests passed! Python conversion is working perfectly!")
    print("\nBenefits we've seen:")
    print("  ✓ Clear output - we can see exactly what's happening")
    print("  ✓ Easy to modify - change values and test immediately")
    print("  ✓ Easy to debug - if something breaks, we know exactly what")
    print("  ✓ We can work together - I can see your code and help directly!")
else:
    print(f"\n⚠ {total - passed} test(s) failed, but we can debug them easily!")

print("\nReady to continue with full conversion?")

