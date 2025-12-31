"""
test_branches.py - Test branch instructions
"""

import sys
import os
script_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(script_dir)
src_python_path = os.path.join(project_root, "src_python")
sys.path.insert(0, src_python_path)

import config
import cpu
import memory


def test_beq_taken():
    """Test BEQ when condition is true"""
    print("Test 1: BEQ rel (branch taken)")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    # Set Z flag (equal)
    cpu.cpu_set_flag(0, True)
    emu.cpu.pc_offset = 0x1000
    
    # Simulate fetch (increment PC)
    emu.cpu.pc_offset += 2
    
    # Write branch offset (+0x0100) at current PC
    memory.memory_write16(0, 0x1002, 0x0100)
    
    # BEQ rel
    instruction = 0xC100  # BEQ, mode 0
    cpu.cpu_execute_instruction(instruction)
    
    # Should branch: PC was at 0x1002, read offset, PC becomes 0x1004, then add offset: 0x1004 + 0x0100 = 0x1104
    expected = 0x1104
    if emu.cpu.pc_offset == expected:
        print(f"  ✓ PASS: PC = {emu.cpu.pc_offset:04X}")
        return True
    else:
        print(f"  ✗ FAIL: PC = {emu.cpu.pc_offset:04X}, expected {expected:04X}")
        return False


def test_beq_not_taken():
    """Test BEQ when condition is false"""
    print("\nTest 2: BEQ rel (branch not taken)")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    # Clear Z flag (not equal)
    cpu.cpu_set_flag(0, False)
    emu.cpu.pc_offset = 0x1000
    
    # Write branch offset (+0x0100)
    memory.memory_write16(0, 0x1000, 0x0100)
    
    # BEQ rel
    instruction = 0xC100  # BEQ, mode 0
    cpu.cpu_execute_instruction(instruction)
    
    # Should not branch: 0x1000 + 2 = 0x1002 (skip offset word)
    expected = 0x1002
    if emu.cpu.pc_offset == expected:
        print(f"  ✓ PASS: PC = {emu.cpu.pc_offset:04X}")
        return True
    else:
        print(f"  ✗ FAIL: PC = {emu.cpu.pc_offset:04X}, expected {expected:04X}")
        return False


def test_bne_taken():
    """Test BNE when condition is true"""
    print("\nTest 3: BNE rel (branch taken)")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    # Clear Z flag (not equal)
    cpu.cpu_set_flag(0, False)
    emu.cpu.pc_offset = 0x2000
    
    # Simulate fetch (increment PC)
    emu.cpu.pc_offset += 2
    
    # Write branch offset (+0x0200) at current PC
    memory.memory_write16(0, 0x2002, 0x0200)
    
    # BNE rel
    instruction = 0xC200  # BNE, mode 0
    cpu.cpu_execute_instruction(instruction)
    
    # Should branch: PC was at 0x2002, read offset, PC becomes 0x2004, then add offset: 0x2004 + 0x0200 = 0x2204
    expected = 0x2204
    if emu.cpu.pc_offset == expected:
        print(f"  ✓ PASS: PC = {emu.cpu.pc_offset:04X}")
        return True
    else:
        print(f"  ✗ FAIL: PC = {emu.cpu.pc_offset:04X}, expected {expected:04X}")
        return False


def test_bgt_taken():
    """Test BGT when condition is true"""
    print("\nTest 4: BGT rel (branch taken)")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    # Set flags: Z=0, N=0 (greater than)
    cpu.cpu_set_flag(0, False)
    cpu.cpu_set_flag(1, False)
    emu.cpu.pc_offset = 0x3000
    
    # Simulate fetch (increment PC)
    emu.cpu.pc_offset += 2
    
    # Write branch offset (+0x0300) at current PC
    memory.memory_write16(0, 0x3002, 0x0300)
    
    # BGT rel
    instruction = 0xC300  # BGT, mode 0
    cpu.cpu_execute_instruction(instruction)
    
    # Should branch: PC was at 0x3002, read offset, PC becomes 0x3004, then add offset: 0x3004 + 0x0300 = 0x3304
    expected = 0x3304
    if emu.cpu.pc_offset == expected:
        print(f"  ✓ PASS: PC = {emu.cpu.pc_offset:04X}")
        return True
    else:
        print(f"  ✗ FAIL: PC = {emu.cpu.pc_offset:04X}, expected {expected:04X}")
        return False


def test_cmp_beq_sequence():
    """Test CMP followed by BEQ"""
    print("\nTest 5: CMP R1, R2; BEQ (full sequence)")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    # Set registers to equal values
    emu.cpu.r1 = 0x1234
    emu.cpu.r2 = 0x1234
    emu.cpu.pc_offset = 0x4000
    
    # CMP R1, R2 (simulate fetch first)
    emu.cpu.pc_offset += 2
    instruction1 = (0xC << 12) | (0x0 << 8) | (0x1 << 4) | 0x2
    cpu.cpu_execute_instruction(instruction1)
    
    # Check Z flag is set
    if not cpu.cpu_get_flag(0):
        print("  ✗ FAIL: Z flag not set after CMP")
        return False
    
    # Simulate fetch for BEQ (increment PC)
    emu.cpu.pc_offset += 2
    
    # Write branch offset
    memory.memory_write16(0, 0x4004, 0x0100)
    
    # BEQ rel (should branch)
    instruction2 = 0xC100
    cpu.cpu_execute_instruction(instruction2)
    
    # Should branch: PC was at 0x4004, read offset, PC becomes 0x4006, then add offset: 0x4006 + 0x0100 = 0x4106
    expected = 0x4106
    if emu.cpu.pc_offset == expected:
        print(f"  ✓ PASS: PC = {emu.cpu.pc_offset:04X}")
        return True
    else:
        print(f"  ✗ FAIL: PC = {emu.cpu.pc_offset:04X}, expected {expected:04X}")
        return False


def main():
    print("=" * 60)
    print("Branch Instruction Tests")
    print("=" * 60)
    print()
    
    tests = [
        test_beq_taken,
        test_beq_not_taken,
        test_bne_taken,
        test_bgt_taken,
        test_cmp_beq_sequence,
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
        print("\n✓ All branch tests passed!")
    else:
        print(f"\n⚠ {total - passed} test(s) failed")


if __name__ == "__main__":
    main()

