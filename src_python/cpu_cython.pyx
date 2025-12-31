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

# Import Python modules (can't cimport Python modules, must import normally)
cdef object config_module = None
cdef object _emu = None
cdef object _memory_module = None

# Opcode constants (cached for fast access)
cdef uint16_t OP_NOP = 0x0000
cdef uint16_t OP_MOV = 0x1000
cdef uint16_t OP_ADD = 0x2000
cdef uint16_t OP_SUB = 0x3000
cdef uint16_t OP_MUL = 0x4000
cdef uint16_t OP_DIV = 0x5000
cdef uint16_t OP_AND = 0x6000
cdef uint16_t OP_OR = 0x7000
cdef uint16_t OP_XOR = 0x8000
cdef uint16_t OP_NOT = 0x9000
cdef uint16_t OP_SHL = 0xA000
cdef uint16_t OP_SHR = 0xB000
cdef uint16_t OP_CMP = 0xC000
cdef uint16_t OP_BEQ = 0xC100
cdef uint16_t OP_BNE = 0xC200
cdef uint16_t OP_BGT = 0xC300
cdef uint16_t OP_BLT = 0xC400
cdef uint16_t OP_BGE = 0xC500
cdef uint16_t OP_BLE = 0xC600
cdef uint16_t OP_JMP = 0xD000
cdef uint16_t OP_CALL = 0xE000
cdef uint16_t OP_RET = 0xF000

def init_cython_cpu():
    """Initialize Cython CPU module - cache module references"""
    global _emu, _memory_module, config_module
    import sys
    import os
    # Add src_python to path if not already there
    src_dir = os.path.dirname(__file__)
    if src_dir not in sys.path:
        sys.path.insert(0, src_dir)
    import config
    config_module = config
    _emu = config.emulator
    import memory
    _memory_module = memory

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
    Returns: cycles executed, or 0 if instruction not handled (fallback to Python)
    """
    cdef uint16_t cycles = 0  # 0 means "not handled, use Python"
    cdef uint16_t src_value, dest_value, result
    cdef uint16_t addr, value
    cdef int16_t offset
    cdef object cpu = _emu.cpu
    
    # MOV instruction (most common)
    if opcode == OP_MOV:
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
        elif mode == 4:  # PUSH
            value = get_register_fast(reg1)
            cpu.sp = (cpu.sp - 2) & 0xFFFF
            if cpu.sp < 0:
                cpu.sp = 0x1FFF  # Stack base
            _memory_module.memory_write16(0, cpu.sp, value)
            cycles = 2
        elif mode == 5:  # POP
            value = _memory_module.memory_read16(0, cpu.sp)
            cpu.sp = (cpu.sp + 2) & 0xFFFF
            if cpu.sp > 0x1FFF:
                cpu.sp = 0
            set_register_fast(reg1, value)
            cycles = 2
    
    # ADD instruction
    elif opcode == OP_ADD:
        dest_value = get_register_fast(reg1)
        if mode == 0:  # Register to register
            src_value = get_register_fast(reg2)
            result = (dest_value + src_value) & 0xFFFF
            set_register_fast(reg1, result)
            cycles = 1
        elif mode == 1:  # Immediate
            value = _memory_module.memory_read16(cpu.pbr, cpu.pc_offset)
            cpu.pc_offset = (cpu.pc_offset + 2) & 0xFFFF
            result = (dest_value + value) & 0xFFFF
            set_register_fast(reg1, result)
            cycles = 2
    
    # SUB instruction
    elif opcode == OP_SUB:
        dest_value = get_register_fast(reg1)
        if mode == 0:  # Register to register
            src_value = get_register_fast(reg2)
            result = (dest_value - src_value) & 0xFFFF
            set_register_fast(reg1, result)
            cycles = 1
        elif mode == 1:  # Immediate
            value = _memory_module.memory_read16(cpu.pbr, cpu.pc_offset)
            cpu.pc_offset = (cpu.pc_offset + 2) & 0xFFFF
            result = (dest_value - value) & 0xFFFF
            set_register_fast(reg1, result)
            cycles = 2
    
    # CMP instruction (sets flags, doesn't store result)
    elif opcode == OP_CMP:
        dest_value = get_register_fast(reg1)
        if mode == 1:  # Immediate (most common)
            value = _memory_module.memory_read16(cpu.pbr, cpu.pc_offset)
            cpu.pc_offset = (cpu.pc_offset + 2) & 0xFFFF
            result = (dest_value - value) & 0xFFFF
            # Update flags (simplified - just Z and N)
            if result == 0:
                cpu.flags = (cpu.flags | 0x01) & ~0x02  # Set Z, clear N
            else:
                cpu.flags = (cpu.flags & ~0x01) | ((result >> 15) & 0x02)  # Clear Z, set N if negative
            cycles = 2
        elif mode == 0:  # Register to register
            src_value = get_register_fast(reg2)
            result = (dest_value - src_value) & 0xFFFF
            if result == 0:
                cpu.flags = (cpu.flags | 0x01) & ~0x02
            else:
                cpu.flags = (cpu.flags & ~0x01) | ((result >> 15) & 0x02)
            cycles = 1
    
    # Branch instructions (BNE, BLT, BGE, etc.)
    elif opcode == OP_BNE:
        if (cpu.flags & 0x01) == 0:  # Z flag clear
            offset = <int16_t>_memory_module.memory_read16(cpu.pbr, cpu.pc_offset)
            cpu.pc_offset = (cpu.pc_offset + 2) & 0xFFFF
            cpu.pc_offset = (cpu.pc_offset + offset) & 0xFFFF
            cycles = 3
        else:
            cpu.pc_offset = (cpu.pc_offset + 2) & 0xFFFF
            cycles = 2
    elif opcode == OP_BLT:
        if (cpu.flags & 0x02) != 0:  # N flag set
            offset = <int16_t>_memory_module.memory_read16(cpu.pbr, cpu.pc_offset)
            cpu.pc_offset = (cpu.pc_offset + 2) & 0xFFFF
            cpu.pc_offset = (cpu.pc_offset + offset) & 0xFFFF
            cycles = 3
        else:
            cpu.pc_offset = (cpu.pc_offset + 2) & 0xFFFF
            cycles = 2
    elif opcode == OP_BGE:
        if (cpu.flags & 0x02) == 0:  # N flag clear
            offset = <int16_t>_memory_module.memory_read16(cpu.pbr, cpu.pc_offset)
            cpu.pc_offset = (cpu.pc_offset + 2) & 0xFFFF
            cpu.pc_offset = (cpu.pc_offset + offset) & 0xFFFF
            cycles = 3
        else:
            cpu.pc_offset = (cpu.pc_offset + 2) & 0xFFFF
            cycles = 2
    
    # JMP instruction
    elif opcode == OP_JMP:
        if mode == 1:  # Relative jump
            offset = <int16_t>_memory_module.memory_read16(cpu.pbr, cpu.pc_offset)
            cpu.pc_offset = (cpu.pc_offset + 2) & 0xFFFF
            cpu.pc_offset = (cpu.pc_offset + offset) & 0xFFFF
            cycles = 3
    
    # NOP
    elif opcode == OP_NOP:
        cycles = 1
    
    # Return 0 if not handled (will fall back to Python)
    return cycles

