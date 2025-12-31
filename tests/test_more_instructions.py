"""
test_more_instructions.py - Test additional CPU instructions
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


def test_add_register():
    """Test ADD R1, R2"""
    print("Test 1: ADD R1, R2 (register to register)")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    emu.cpu.r1 = 0x1000
    emu.cpu.r2 = 0x0234
    
    # ADD R1, R2
    instruction = (0x2 << 12) | (0x0 << 8) | (0x1 << 4) | 0x2
    cpu.cpu_execute_instruction(instruction)
    
    expected = 0x1234
    if emu.cpu.r1 == expected:
        print(f"  ✓ PASS: R1 = {emu.cpu.r1:04X}")
        return True
    else:
        print(f"  ✗ FAIL: R1 = {emu.cpu.r1:04X}, expected {expected:04X}")
        return False


def test_add_immediate():
    """Test ADD R0, #0x0100"""
    print("\nTest 2: ADD R0, #0x0100 (immediate)")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    emu.cpu.r0 = 0x2000
    emu.cpu.pc_offset = 0x1000
    memory.memory_write16(0, 0x1000, 0x0100)
    
    # ADD R0, #imm
    instruction = (0x2 << 12) | (0x1 << 8) | (0x0 << 4) | 0x0
    cpu.cpu_execute_instruction(instruction)
    
    expected = 0x2100
    if emu.cpu.r0 == expected:
        print(f"  ✓ PASS: R0 = {emu.cpu.r0:04X}")
        return True
    else:
        print(f"  ✗ FAIL: R0 = {emu.cpu.r0:04X}, expected {expected:04X}")
        return False


def test_sub_register():
    """Test SUB R1, R2"""
    print("\nTest 3: SUB R1, R2 (register to register)")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    emu.cpu.r1 = 0x1234
    emu.cpu.r2 = 0x0234
    
    # SUB R1, R2
    instruction = (0x3 << 12) | (0x0 << 8) | (0x1 << 4) | 0x2
    cpu.cpu_execute_instruction(instruction)
    
    expected = 0x1000
    if emu.cpu.r1 == expected:
        print(f"  ✓ PASS: R1 = {emu.cpu.r1:04X}")
        return True
    else:
        print(f"  ✗ FAIL: R1 = {emu.cpu.r1:04X}, expected {expected:04X}")
        return False


def test_and_register():
    """Test AND R1, R2"""
    print("\nTest 4: AND R1, R2 (bitwise AND)")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    emu.cpu.r1 = 0xFF00
    emu.cpu.r2 = 0x00FF
    
    # AND R1, R2
    instruction = (0x6 << 12) | (0x0 << 8) | (0x1 << 4) | 0x2
    cpu.cpu_execute_instruction(instruction)
    
    expected = 0x0000  # 0xFF00 & 0x00FF = 0x0000
    if emu.cpu.r1 == expected:
        print(f"  ✓ PASS: R1 = {emu.cpu.r1:04X}")
        return True
    else:
        print(f"  ✗ FAIL: R1 = {emu.cpu.r1:04X}, expected {expected:04X}")
        return False


def test_or_register():
    """Test OR R1, R2"""
    print("\nTest 5: OR R1, R2 (bitwise OR)")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    emu.cpu.r1 = 0xFF00
    emu.cpu.r2 = 0x00FF
    
    # OR R1, R2
    instruction = (0x7 << 12) | (0x0 << 8) | (0x1 << 4) | 0x2
    cpu.cpu_execute_instruction(instruction)
    
    expected = 0xFFFF  # 0xFF00 | 0x00FF = 0xFFFF
    if emu.cpu.r1 == expected:
        print(f"  ✓ PASS: R1 = {emu.cpu.r1:04X}")
        return True
    else:
        print(f"  ✗ FAIL: R1 = {emu.cpu.r1:04X}, expected {expected:04X}")
        return False


def test_xor_register():
    """Test XOR R1, R2"""
    print("\nTest 6: XOR R1, R2 (bitwise XOR)")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    emu.cpu.r1 = 0xFF00
    emu.cpu.r2 = 0x00FF
    
    # XOR R1, R2
    instruction = (0x8 << 12) | (0x0 << 8) | (0x1 << 4) | 0x2
    cpu.cpu_execute_instruction(instruction)
    
    expected = 0xFFFF  # 0xFF00 ^ 0x00FF = 0xFFFF
    if emu.cpu.r1 == expected:
        print(f"  ✓ PASS: R1 = {emu.cpu.r1:04X}")
        return True
    else:
        print(f"  ✗ FAIL: R1 = {emu.cpu.r1:04X}, expected {expected:04X}")
        return False


def test_not_register():
    """Test NOT R1"""
    print("\nTest 7: NOT R1 (bitwise NOT)")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    emu.cpu.r1 = 0xFF00
    
    # NOT R1
    instruction = (0x9 << 12) | (0x0 << 8) | (0x1 << 4) | 0x0
    cpu.cpu_execute_instruction(instruction)
    
    expected = 0x00FF  # ~0xFF00 & 0xFFFF = 0x00FF
    if emu.cpu.r1 == expected:
        print(f"  ✓ PASS: R1 = {emu.cpu.r1:04X}")
        return True
    else:
        print(f"  ✗ FAIL: R1 = {emu.cpu.r1:04X}, expected {expected:04X}")
        return False


def test_cmp_register():
    """Test CMP R1, R2"""
    print("\nTest 8: CMP R1, R2 (compare)")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    emu.cpu.r1 = 0x1234
    emu.cpu.r2 = 0x1234
    
    # CMP R1, R2
    instruction = (0xC << 12) | (0x0 << 8) | (0x1 << 4) | 0x2
    cpu.cpu_execute_instruction(instruction)
    
    # Check flags: Z should be set (equal)
    z_flag = cpu.cpu_get_flag(0)
    if z_flag and emu.cpu.r1 == 0x1234:  # R1 should be unchanged
        print(f"  ✓ PASS: Z flag set, R1 unchanged = {emu.cpu.r1:04X}")
        return True
    else:
        print(f"  ✗ FAIL: Z flag = {z_flag}, R1 = {emu.cpu.r1:04X}")
        return False


def test_add_flags():
    """Test ADD with flag updates"""
    print("\nTest 9: ADD with flag updates")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    # Test zero flag: 0x0000 + 0x0000 = 0x0000
    emu.cpu.r0 = 0x0000
    emu.cpu.r1 = 0x0000
    instruction = (0x2 << 12) | (0x0 << 8) | (0x0 << 4) | 0x1
    cpu.cpu_execute_instruction(instruction)
    
    z_flag = cpu.cpu_get_flag(0)
    if z_flag and emu.cpu.r0 == 0x0000:
        print(f"  ✓ PASS: Zero flag set correctly")
        return True
    else:
        print(f"  ✗ FAIL: Zero flag = {z_flag}, R0 = {emu.cpu.r0:04X}")
        return False


def main():
    print("=" * 60)
    print("Additional CPU Instruction Tests")
    print("=" * 60)
    print()
    
    tests = [
        test_add_register,
        test_add_immediate,
        test_sub_register,
        test_and_register,
        test_or_register,
        test_xor_register,
        test_not_register,
        test_cmp_register,
        test_add_flags,
    ]
    
    results = []
    for test in tests:
        try:
            result = test()
            results.append(result)
        except Exception as e:
            print(f"  ✗ ERROR: {type(e).__name__}: {e}")
            import traceback
            traceback.print_exc()
            results.append(False)
    
    print("\n" + "=" * 60)
    print("Summary")
    print("=" * 60)
    passed = sum(results)
    total = len(results)
    print(f"Tests passed: {passed}/{total}")
    
    if passed == total:
        print("\n✓ All instruction tests passed!")
    else:
        print(f"\n⚠ {total - passed} test(s) failed")


if __name__ == "__main__":
    main()

