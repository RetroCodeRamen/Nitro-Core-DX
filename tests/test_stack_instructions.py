"""
test_stack_instructions.py - Test stack and remaining CPU instructions
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


def test_mul_register():
    """Test MUL R1, R2"""
    print("Test 1: MUL R1, R2")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    emu.cpu.r1 = 0x0010  # 16
    emu.cpu.r2 = 0x0004  # 4
    
    instruction = (0x4 << 12) | (0x0 << 8) | (0x1 << 4) | 0x2
    cpu.cpu_execute_instruction(instruction)
    
    expected = 0x0040  # 16 * 4 = 64
    if emu.cpu.r1 == expected:
        print(f"  ✓ PASS: R1 = {emu.cpu.r1:04X}")
        return True
    else:
        print(f"  ✗ FAIL: R1 = {emu.cpu.r1:04X}, expected {expected:04X}")
        return False


def test_div_register():
    """Test DIV R1, R2"""
    print("\nTest 2: DIV R1, R2")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    emu.cpu.r1 = 0x0040  # 64
    emu.cpu.r2 = 0x0004  # 4
    
    instruction = (0x5 << 12) | (0x0 << 8) | (0x1 << 4) | 0x2
    cpu.cpu_execute_instruction(instruction)
    
    expected = 0x0010  # 64 / 4 = 16
    if emu.cpu.r1 == expected:
        print(f"  ✓ PASS: R1 = {emu.cpu.r1:04X}")
        return True
    else:
        print(f"  ✗ FAIL: R1 = {emu.cpu.r1:04X}, expected {expected:04X}")
        return False


def test_shl_register():
    """Test SHL R1, R2"""
    print("\nTest 3: SHL R1, R2")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    emu.cpu.r1 = 0x0001  # 1
    emu.cpu.r2 = 0x0004  # Shift by 4
    
    instruction = (0xA << 12) | (0x0 << 8) | (0x1 << 4) | 0x2
    cpu.cpu_execute_instruction(instruction)
    
    expected = 0x0010  # 1 << 4 = 16
    if emu.cpu.r1 == expected:
        print(f"  ✓ PASS: R1 = {emu.cpu.r1:04X}")
        return True
    else:
        print(f"  ✗ FAIL: R1 = {emu.cpu.r1:04X}, expected {expected:04X}")
        return False


def test_shr_register():
    """Test SHR R1, R2"""
    print("\nTest 4: SHR R1, R2")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    emu.cpu.r1 = 0x0010  # 16
    emu.cpu.r2 = 0x0004  # Shift by 4
    
    instruction = (0xB << 12) | (0x0 << 8) | (0x1 << 4) | 0x2
    cpu.cpu_execute_instruction(instruction)
    
    expected = 0x0001  # 16 >> 4 = 1
    if emu.cpu.r1 == expected:
        print(f"  ✓ PASS: R1 = {emu.cpu.r1:04X}")
        return True
    else:
        print(f"  ✗ FAIL: R1 = {emu.cpu.r1:04X}, expected {expected:04X}")
        return False


def test_push_pop():
    """Test PUSH and POP"""
    print("\nTest 5: PUSH R1; POP R2")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    emu.cpu.r1 = 0x1234
    initial_sp = emu.cpu.sp
    
    # PUSH R1 (MOV with mode 4)
    instruction1 = (0x1 << 12) | (0x4 << 8) | (0x1 << 4) | 0x0
    cpu.cpu_execute_instruction(instruction1)
    
    # POP R2 (MOV with mode 5)
    instruction2 = (0x1 << 12) | (0x5 << 8) | (0x2 << 4) | 0x0
    cpu.cpu_execute_instruction(instruction2)
    
    if emu.cpu.r2 == 0x1234 and emu.cpu.sp == initial_sp:
        print(f"  ✓ PASS: R2 = {emu.cpu.r2:04X}, SP restored")
        return True
    else:
        print(f"  ✗ FAIL: R2 = {emu.cpu.r2:04X}, SP = {emu.cpu.sp:04X}")
        return False


def test_call_ret():
    """Test CALL and RET"""
    print("\nTest 6: CALL rel; RET")
    
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    # Set up: instruction at 0x1000, offset word at 0x1002
    emu.cpu.pc_offset = 0x1000
    emu.cpu.pbr = 0
    memory.memory_write16(0, 0x1000, (0xE << 12) | (0x1 << 8) | 0x0)  # CALL rel instruction
    memory.memory_write16(0, 0x1002, 0x0100)  # Call +0x0100 (relative)
    
    # Write RET at target (0x1104 after fetch and offset)
    memory.memory_write16(0, 0x1104, 0xF000)  # RET
    
    # Simulate fetch_instruction: PC starts at 0x1000
    # After fetch, PC is at 0x1002, then CALL reads offset from 0x1002
    # Return address should be 0x1004 (after offset word)
    # Then jump to 0x1004 + 0x0100 = 0x1104
    instruction1 = cpu.cpu_fetch_instruction()  # This increments PC to 0x1002
    cpu.cpu_execute_instruction(instruction1)
    
    # Now PC should be at 0x1104 (target)
    if emu.cpu.pc_offset != 0x1104:
        print(f"  ✗ FAIL: After CALL, PC = {emu.cpu.pc_offset:04X}, expected 0x1104")
        return False
    
    # RET: fetch instruction (PC goes to 0x1106), then restore saved PC
    instruction2 = cpu.cpu_fetch_instruction()  # This increments PC to 0x1106
    cpu.cpu_execute_instruction(instruction2)
    
    # Should return to after the call (0x1004)
    expected = 0x1004  # After call instruction + offset word
    if emu.cpu.pc_offset == expected:
        print(f"  ✓ PASS: PC = {emu.cpu.pc_offset:04X}")
        return True
    else:
        print(f"  ✗ FAIL: PC = {emu.cpu.pc_offset:04X}, expected {expected:04X}")
        return False


def main():
    print("=" * 60)
    print("Stack and Remaining Instruction Tests")
    print("=" * 60)
    print()
    
    tests = [
        test_mul_register,
        test_div_register,
        test_shl_register,
        test_shr_register,
        test_push_pop,
        test_call_ret,
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
        print("\n✓ All stack instruction tests passed!")
    else:
        print(f"\n⚠ {total - passed} test(s) failed")


if __name__ == "__main__":
    main()

