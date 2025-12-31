"""
test_basic.py - Simple test to verify Python conversion works
Run with: python src_python/test_basic.py
"""

import sys
import os

# Add src_python to path
script_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(script_dir)
src_python_path = os.path.join(project_root, 'src_python')
sys.path.insert(0, src_python_path)

print("=" * 60)
print("Fantasy Console Emulator - Python Conversion Test")
print("=" * 60)
print()

# Test 1: Import config
print("Test 1: Importing config...")
try:
    import config
    print("  ✓ config imported successfully")
    print(f"  ✓ DISPLAY_WIDTH = {config.DISPLAY_WIDTH}")
    print(f"  ✓ TARGET_FPS = {config.TARGET_FPS}")
except Exception as e:
    print(f"  ✗ Error: {e}")
    sys.exit(1)

# Test 2: Initialize emulator
print("\nTest 2: Initializing emulator state...")
try:
    emu = config.emulator
    print("  ✓ Emulator state created")
    print(f"  ✓ CPU state: R0={emu.cpu.r0}, PC={emu.cpu.pc_bank}:{emu.cpu.pc_offset}")
    print(f"  ✓ Memory state: ROM size={emu.memory.rom_size}")
except Exception as e:
    print(f"  ✗ Error: {e}")
    sys.exit(1)

# Test 3: Test CPU reset
print("\nTest 3: Testing CPU reset...")
try:
    import cpu
    cpu.cpu_reset()
    print("  ✓ CPU reset successful")
    print(f"  ✓ CPU R0={emu.cpu.r0}, SP={emu.cpu.sp}")
except Exception as e:
    print(f"  ✗ Error: {e}")
    import traceback
    traceback.print_exc()
    sys.exit(1)

# Test 4: Test memory reset
print("\nTest 4: Testing memory reset...")
try:
    import memory
    memory.memory_reset()
    print("  ✓ Memory reset successful")
    print(f"  ✓ WRAM[0] = {config.memory_wram[0]}")
    print(f"  ✓ WRAM size = {len(config.memory_wram)}")
except Exception as e:
    print(f"  ✗ Error: {e}")
    import traceback
    traceback.print_exc()
    sys.exit(1)

# Test 5: Test memory read/write
print("\nTest 5: Testing memory read/write...")
try:
    # Write test value
    memory.memory_write8(0, 0x1000, 0x42)
    value = memory.memory_read8(0, 0x1000)
    if value == 0x42:
        print("  ✓ Memory read/write works correctly")
        print(f"  ✓ Wrote 0x42, read {hex(value)}")
    else:
        print(f"  ✗ Memory read/write failed: wrote 0x42, read {hex(value)}")
        sys.exit(1)
except Exception as e:
    print(f"  ✗ Error: {e}")
    import traceback
    traceback.print_exc()
    sys.exit(1)

# Test 6: Test 16-bit memory
print("\nTest 6: Testing 16-bit memory...")
try:
    memory.memory_write16(0, 0x2000, 0x1234)
    value = memory.memory_read16(0, 0x2000)
    if value == 0x1234:
        print("  ✓ 16-bit memory read/write works correctly")
        print(f"  ✓ Wrote 0x1234, read {hex(value)}")
    else:
        print(f"  ✗ 16-bit memory failed: wrote 0x1234, read {hex(value)}")
        sys.exit(1)
except Exception as e:
    print(f"  ✗ Error: {e}")
    import traceback
    traceback.print_exc()
    sys.exit(1)

print("\n" + "=" * 60)
print("✓ All basic tests passed!")
print("=" * 60)
print("\nThe Python conversion is working correctly.")
print("You can now debug together easily!")
print("\nNext steps:")
print("  1. Complete remaining modules (rom, ppu, apu, input, debug, main)")
print("  2. Install pygame: pip install pygame")
print("  3. Run full emulator: python src_python/main.py")

