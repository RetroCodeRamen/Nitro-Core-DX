#!/usr/bin/env python3
"""
Test Cython usage - verify it's being called
"""

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'src_python'))

import config
import cpu
import memory
import rom

# Reset emulator
config.emulator = config.EmulatorState()
cpu.cpu_reset()
memory.memory_reset()

# Load test ROM
rom_file = os.path.join(os.path.dirname(__file__), '..', 'roms', 'graphics.rom')
rom.rom_load_file(rom_file)
cpu.cpu_reset()

# Disable logging
import ui
ui.logger.enabled = False

# Check if Cython is available
try:
    import cpu_cython
    print("✓ Cython module available")
    cpu_cython.init_cython_cpu()
    print("✓ Cython initialized")
except ImportError as e:
    print(f"✗ Cython not available: {e}")
    sys.exit(1)

# Count instructions executed
instruction_count = 0
cython_count = 0
python_count = 0

# Monkey patch to count
original_execute = cpu.cpu_execute_instruction
def counting_execute(instruction):
    global python_count
    python_count += 1
    return original_execute(instruction)
cpu.cpu_execute_instruction = counting_execute

# Run for a small number of cycles
target = config.emulator.cpu.cycles + 1000
while config.emulator.cpu.cycles < target:
    instruction = cpu.cpu_fetch_instruction()
    opcode, mode, reg1, reg2 = cpu.cpu_decode_instruction(instruction)
    
    try:
        cycles = cpu_cython.cpu_execute_instruction_cython(instruction, opcode, mode, reg1, reg2)
        if cycles > 0:
            cython_count += 1
            config.emulator.cpu.cycles += cycles
        else:
            cpu.cpu_execute_instruction(instruction)
    except Exception as e:
        print(f"Error in Cython: {e}")
        cpu.cpu_execute_instruction(instruction)
    
    instruction_count += 1
    if instruction_count >= 100:
        break

print(f"\nInstructions executed: {instruction_count}")
print(f"Cython handled: {cython_count} ({cython_count/instruction_count*100:.1f}%)")
print(f"Python handled: {python_count} ({python_count/instruction_count*100:.1f}%)")

