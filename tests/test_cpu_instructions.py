"""
test_cpu_instructions.py - Test CPU instruction implementation
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

def test_mov_register_to_register():
    """Test MOV R1, R2"""
    print("Test 1: MOV R1, R2 (register to register)")
    
    # Reset
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    # Set R2 to test value
    emu.cpu.r2 = 0x1234
    
    # Create instruction: MOV R1, R2
    # Format: [15:12]=opcode(1), [11:8]=mode(0), [7:4]=dest(R1), [3:0]=src(R2)
    instruction = (0x1 << 12) | (0x0 << 8) | (0x1 << 4) | 0x2
    
    # Execute
    cpu.cpu_execute_instruction(instruction)
    
    # Check result
    if emu.cpu.r1 == 0x1234:
        print("  ✓ PASS: R1 = 0x1234")
        return True
    else:
        print(f"  ✗ FAIL: R1 = {emu.cpu.r1:04X}, expected 0x1234")
        return False


def test_mov_immediate():
    """Test MOV R0, #0x5678"""
    print("\nTest 2: MOV R0, #0x5678 (immediate)")
    
    # Reset
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    # Write immediate value to memory at PC
    emu.cpu.pc_offset = 0x1000
    memory.memory_write16(emu.cpu.pbr, 0x1000, 0x5678)
    emu.cpu.pc_offset = 0x1000  # Reset PC for instruction
    
    # Create instruction: MOV R0, #imm
    # Format: [15:12]=opcode(1), [11:8]=mode(1), [7:4]=dest(R0), [3:0]=unused
    instruction = (0x1 << 12) | (0x1 << 8) | (0x0 << 4) | 0x0
    
    # Execute
    cpu.cpu_execute_instruction(instruction)
    
    # Check result
    if emu.cpu.r0 == 0x5678:
        print("  ✓ PASS: R0 = 0x5678")
        return True
    else:
        print(f"  ✗ FAIL: R0 = {emu.cpu.r0:04X}, expected 0x5678")
        return False


def test_jmp_relative():
    """Test JMP rel (relative jump)"""
    print("\nTest 3: JMP rel (relative jump)")
    
    # Reset
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    # Set initial PC
    emu.cpu.pc_offset = 0x2000
    
    # Write offset to memory at PC (next word after instruction)
    memory.memory_write16(emu.cpu.pbr, 0x2000, 0x0100)  # +0x0100 offset
    
    # Create instruction: JMP rel
    # Format: [15:12]=opcode(D), [11:8]=mode(1), [7:4]=unused, [3:0]=unused
    instruction = (0xD << 12) | (0x1 << 8) | 0x0
    
    # Execute
    cpu.cpu_execute_instruction(instruction)
    
    # Check result (should be 0x2000 + 2 (consumed offset word) + 0x0100 = 0x2102)
    # Actually, PC should be: 0x2000 + 2 (after reading offset) + 0x0100 = 0x2102
    expected = 0x2102
    if emu.cpu.pc_offset == expected:
        print(f"  ✓ PASS: PC = {emu.cpu.pc_offset:04X}")
        return True
    else:
        print(f"  ✗ FAIL: PC = {emu.cpu.pc_offset:04X}, expected {expected:04X}")
        return False


def test_jmp_absolute():
    """Test JMP addr (absolute jump)"""
    print("\nTest 4: JMP addr (absolute jump)")
    
    # Reset
    cpu.cpu_reset()
    memory.memory_reset()
    emu = config.emulator
    
    # Use bank 0 (WRAM) which is writable, and use address in writable range
    # Bank 0: 0x0000-0x7FFF is WRAM (writable)
    emu.cpu.pbr = 0
    emu.cpu.pc_offset = 0x1000
    
    # Write instruction and offset to WRAM
    # Instruction at 0x1000: JMP bank=2, offset=0x4000
    instruction = (0xD << 12) | (0x0 << 8) | (0x2 << 4) | 0x0
    memory.memory_write16(0, 0x1000, instruction)
    memory.memory_write16(0, 0x1002, 0x4000)  # Offset word
    
    # Simulate full fetch-execute cycle
    fetched = cpu.cpu_fetch_instruction()  # This increments PC to 0x1002
    cpu.cpu_execute_instruction(fetched)  # This reads offset from PC (0x1002)
    
    # Check result
    if emu.cpu.pbr == 2 and emu.cpu.pc_offset == 0x4000:
        print(f"  ✓ PASS: PC = {emu.cpu.pbr:02X}:{emu.cpu.pc_offset:04X}")
        return True
    else:
        print(f"  ✗ FAIL: PC = {emu.cpu.pbr:02X}:{emu.cpu.pc_offset:04X}, expected 02:4000")
        # Debug: check what was actually read
        read_value = memory.memory_read16(0, 0x1002)
        print(f"      Debug: Value at 0:1002 = {read_value:04X}")
        return False


def main():
    print("=" * 60)
    print("CPU Instruction Tests")
    print("=" * 60)
    print()
    
    tests = [
        test_mov_register_to_register,
        test_mov_immediate,
        test_jmp_relative,
        test_jmp_absolute,
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
        print("\n✓ All CPU instruction tests passed!")
    else:
        print(f"\n⚠ {total - passed} test(s) failed")


if __name__ == "__main__":
    main()

