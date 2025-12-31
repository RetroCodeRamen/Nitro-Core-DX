# cython: language_level=3
# cython: boundscheck=False
# cython: wraparound=False
# cython: cdivision=True
"""
Cython-optimized CPU execution loop
This provides a fast C-compiled version of the CPU execution loop
"""

cimport cython
from libc.stdint cimport uint16_t, uint32_t, int16_t

# Import Python modules
import config
cimport config as config_module

# Cached references for fast access
cdef object _emu = None
cdef object _memory_module = None
cdef object _ui_module = None

def init_cython_cpu():
    """Initialize Cython CPU module - cache module references"""
    global _emu, _memory_module, _ui_module
    _emu = config.emulator
    import memory
    _memory_module = memory
    import ui
    _ui_module = ui

@cython.boundscheck(False)
@cython.wraparound(False)
cdef inline uint16_t get_register_fast(int reg_num):
    """Fast register getter - direct attribute access"""
    cdef object cpu = _emu.cpu
    if reg_num == 0:
        return cpu.r0
    elif reg_num == 1:
        return cpu.r1
    elif reg_num == 2:
        return cpu.r2
    elif reg_num == 3:
        return cpu.r3
    elif reg_num == 4:
        return cpu.r4
    elif reg_num == 5:
        return cpu.r5
    elif reg_num == 6:
        return cpu.r6
    elif reg_num == 7:
        return cpu.r7
    return 0

@cython.boundscheck(False)
@cython.wraparound(False)
cdef inline void set_register_fast(int reg_num, uint16_t value):
    """Fast register setter - direct attribute access"""
    cdef object cpu = _emu.cpu
    value = value & 0xFFFF
    if reg_num == 0:
        cpu.r0 = value
    elif reg_num == 1:
        cpu.r1 = value
    elif reg_num == 2:
        cpu.r2 = value
    elif reg_num == 3:
        cpu.r3 = value
    elif reg_num == 4:
        cpu.r4 = value
    elif reg_num == 5:
        cpu.r5 = value
    elif reg_num == 6:
        cpu.r6 = value
    elif reg_num == 7:
        cpu.r7 = value

@cython.boundscheck(False)
@cython.wraparound(False)
cdef inline uint16_t decode_opcode_family(uint16_t instruction):
    """Fast opcode family decoder"""
    return (instruction >> 12) & 0xF

@cython.boundscheck(False)
@cython.wraparound(False)
def cpu_execute_instruction_cython(uint16_t instruction, uint16_t opcode, uint16_t mode, uint16_t reg1, uint16_t reg2):
    """
    Cython-optimized instruction execution
    Pre-decoded instruction parameters for maximum speed
    Returns: cycles executed
    """
    cdef uint16_t cycles = 1
    cdef uint16_t src_value, dest_value, result
    cdef uint16_t addr, value
    cdef int16_t offset
    cdef object cpu = _emu.cpu
    
    # MOV instruction (most common)
    if opcode == config_module.OP_MOV:
        if mode == 0:  # Register to register
            src_value = get_register_fast(reg2)
            set_register_fast(reg1, src_value)
            cycles = 1
        elif mode == 1:  # Immediate
            # Fetch immediate from memory
            value = _memory_module.memory_read16(cpu.pbr, cpu.pc_offset)
            cpu.pc_offset = (cpu.pc_offset + 2) & 0xFFFF
            set_register_fast(reg1, value)
            cycles = 2
        elif mode == 2:  # Load from memory
            addr = get_register_fast(reg2)
            value = _memory_module.memory_read16(cpu.dbr, addr)
            set_register_fast(reg1, value)
            cycles = 2
        elif mode == 3:  # Store to memory
            addr = get_register_fast(reg1)
            value = get_register_fast(reg2)
            _memory_module.memory_write16(cpu.dbr, addr, value)
            cycles = 2
    
    # ADD instruction
    elif opcode == config_module.OP_ADD:
        dest_value = get_register_fast(reg1)
        if mode == 0:  # Register to register
            src_value = get_register_fast(reg2)
            result = dest_value + src_value
            set_register_fast(reg1, result)
            cycles = 1
        elif mode == 1:  # Immediate
            value = _memory_module.memory_read16(cpu.pbr, cpu.pc_offset)
            cpu.pc_offset = (cpu.pc_offset + 2) & 0xFFFF
            result = dest_value + value
            set_register_fast(reg1, result)
            cycles = 2
    
    # JMP instruction
    elif opcode == config_module.OP_JMP:
        if mode == 1:  # Relative jump
            offset = <int16_t>_memory_module.memory_read16(cpu.pbr, cpu.pc_offset)
            cpu.pc_offset = (cpu.pc_offset + 2) & 0xFFFF
            cpu.pc_offset = (cpu.pc_offset + offset) & 0xFFFF
            cycles = 3
    
    # NOP
    elif opcode == config_module.OP_NOP:
        cycles = 1
    
    # Update cycle counter
    cpu.cycles += cycles
    
    return cycles

