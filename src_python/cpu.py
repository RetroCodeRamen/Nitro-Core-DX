"""
cpu.py - CPU emulation (custom 16-bit CPU with banked addressing)
Python version - maintains BASIC-like simplicity
"""

import config

# Try to import Cython-optimized CPU module (optional)
_cython_available = False
try:
    import cpu_cython
    _cython_available = True
except ImportError:
    pass


def cpu_reset():
    """Initialize CPU to default state"""
    emu = config.emulator
    
    # Clear all registers
    emu.cpu.r0 = 0
    emu.cpu.r1 = 0
    emu.cpu.r2 = 0
    emu.cpu.r3 = 0
    emu.cpu.r4 = 0
    emu.cpu.r5 = 0
    emu.cpu.r6 = 0
    emu.cpu.r7 = 0
    
    # Initialize PC - if ROM is loaded, use its entry point, otherwise 0:0
    if hasattr(emu, 'rom') and emu.rom and emu.rom.entry_point_bank > 0:
        # ROM is loaded - bootstrap to ROM entry point
        emu.cpu.pc_bank = emu.rom.entry_point_bank
        emu.cpu.pc_offset = emu.rom.entry_point_offset
        emu.cpu.pbr = emu.rom.entry_point_bank
        import ui
        ui.logger.info(f"CPU Reset: Bootstrapping to ROM entry point {emu.rom.entry_point_bank:02X}:{emu.rom.entry_point_offset:04X}", "CPU")
    else:
        # No ROM loaded - reset to 0:0
        emu.cpu.pc_bank = 0
        emu.cpu.pc_offset = 0
        emu.cpu.pbr = 0
    
    # Initialize stack pointer
    emu.cpu.sp = config.CPU_STACK_BASE
    
    # Initialize bank registers (if no ROM, set to 0)
    if not (hasattr(emu, 'rom') and emu.rom and emu.rom.entry_point_bank > 0):
        emu.cpu.pbr = 0
    emu.cpu.dbr = 0
    
    # Clear flags
    emu.cpu.flags = 0
    emu.cpu.interrupt_mask = 0  # Interrupts disabled initially
    emu.cpu.interrupt_pending = config.INT_NONE
    
    # Reset cycle counter
    emu.cpu.cycles = 0


def cpu_fetch_instruction():
    """Fetch next instruction word from memory - optimized hot path"""
    import memory  # Import here to avoid circular import
    emu = config.emulator
    
    # Optimized: Cache ROM check (hasattr is expensive - 10,723 calls in profile!)
    # Only check bootstrap conditions if ROM exists and is in invalid state
    # Most of the time, PC is valid, so skip expensive checks
    rom_loaded = emu.rom and emu.rom.entry_point_bank > 0
    if rom_loaded:
        # ROM is loaded - only check bootstrap if PC is in invalid state (optimization)
        # Most of the time PC is valid, so skip expensive checks
        if emu.cpu.pc_offset < 0x8000 or emu.cpu.pbr != emu.rom.entry_point_bank:
            # Only log errors if logging is enabled (optimization)
            import ui
            if ui.logger.enabled:
                if emu.cpu.pbr != emu.rom.entry_point_bank:
                    ui.logger.error(f"CPU ERROR: PBR ({emu.cpu.pbr:02X}) doesn't match ROM entry bank ({emu.rom.entry_point_bank:02X})! Fixing...", "CPU")
                    emu.cpu.pbr = emu.rom.entry_point_bank
                    emu.cpu.pc_bank = emu.rom.entry_point_bank
                
                if emu.cpu.pc_offset < 0x8000:
                    ui.logger.error(f"CPU ERROR: PC in invalid ROM region! PC={emu.cpu.pbr:02X}:{emu.cpu.pc_offset:04X}, bootstrapping to entry point {emu.rom.entry_point_bank:02X}:{emu.rom.entry_point_offset:04X}", "CPU")
                    emu.cpu.pc_bank = emu.rom.entry_point_bank
                    emu.cpu.pc_offset = emu.rom.entry_point_offset
                    emu.cpu.pbr = emu.rom.entry_point_bank
            else:
                # Fast path: just fix it without logging
                if emu.cpu.pbr != emu.rom.entry_point_bank:
                    emu.cpu.pbr = emu.rom.entry_point_bank
                    emu.cpu.pc_bank = emu.rom.entry_point_bank
                if emu.cpu.pc_offset < 0x8000:
                    emu.cpu.pc_bank = emu.rom.entry_point_bank
                    emu.cpu.pc_offset = emu.rom.entry_point_offset
                    emu.cpu.pbr = emu.rom.entry_point_bank
    
    # Optimized: Skip expensive validation checks in hot path
    # Only validate if PC is clearly out of bounds (rare case)
    current_bank = emu.cpu.pbr
    if config.MEMORY_ROM_START_BANK <= current_bank <= config.MEMORY_ROM_END_BANK:
        # Fast path: Most ROM accesses are valid, only check edge cases
        if emu.cpu.pc_offset >= 0x10000:
            # PC wrapped - clamp it (rare, so logging is OK)
            import ui
            if ui.logger.enabled:
                ui.logger.error(f"CPU ERROR: PC wrapped beyond bank! PC={current_bank:02X}:{emu.cpu.pc_offset:04X}, clamping to 0xFFFF", "CPU")
            emu.cpu.pc_offset = 0xFFFF
            emu.cpu.pc_bank = current_bank
    
    # Read 16-bit word from memory at PC (PBR:PC_Offset)
    instruction = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
    
    # Increment PC_Offset by 2 (optimized: do this before logging to avoid extra variable)
    emu.cpu.pc_offset += 2
    
    # Log instruction fetch (only if detailed logging enabled - moved after PC increment)
    import ui
    if ui.logger.enabled and ui.logger.detailed_logging:
        ui.logger.trace(f"Fetch: PC={emu.cpu.pc_bank:02X}:{emu.cpu.pc_offset-2:04X} -> {instruction:04X}", "CPU")
    
    # Handle bank wrapping/clamping
    if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
        # For ROM banks, wrap to 0x8000 (start of ROM), not 0x0000
        if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
            # Wrap to start of ROM region in this bank
            emu.cpu.pc_offset = 0x8000 + (emu.cpu.pc_offset - config.MEMORY_BANK_SIZE)
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                emu.cpu.pc_offset = 0x8000  # Clamp if still out of bounds
        else:
            # For non-ROM banks, wrap normally
            emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
    
    return instruction


def cpu_decode_instruction(instruction):
    """
    Decode instruction - optimized with inline opcode mapping
    Instruction format: [15:12] = opcode family, [11:8] = mode/subop, [7:4] = reg1, [3:0] = reg2
    Returns: (opcode, mode, reg1, reg2)
    """
    # Extract fields (optimized: use bitwise ops directly)
    opcode_family = (instruction >> 12) & 0xF
    mode = (instruction >> 8) & 0xF
    reg1 = (instruction >> 4) & 0xF
    reg2 = instruction & 0xF
    
    # Optimized: Inline opcode mapping instead of dict lookup (faster)
    # Map opcode family to actual opcode using if/elif (faster than dict for small sets)
    if opcode_family == 0x0:
        opcode = config.OP_NOP
    elif opcode_family == 0x1:
        opcode = config.OP_MOV
    elif opcode_family == 0x2:
        opcode = config.OP_ADD
    elif opcode_family == 0x3:
        opcode = config.OP_SUB
    elif opcode_family == 0x4:
        opcode = config.OP_MUL
    elif opcode_family == 0x5:
        opcode = config.OP_DIV
    elif opcode_family == 0x6:
        opcode = config.OP_AND
    elif opcode_family == 0x7:
        opcode = config.OP_OR
    elif opcode_family == 0x8:
        opcode = config.OP_XOR
    elif opcode_family == 0x9:
        opcode = config.OP_NOT
    elif opcode_family == 0xA:
        opcode = config.OP_SHL
    elif opcode_family == 0xB:
        opcode = config.OP_SHR
    elif opcode_family == 0xD:
        opcode = config.OP_JMP
    elif opcode_family == 0xE:
        opcode = config.OP_CALL
    elif opcode_family == 0xF:
        opcode = config.OP_RET
    else:
        opcode = config.OP_NOP  # Default fallback
    
    # Check for extended opcodes (Cxxx for CMP and branches)
    if opcode_family == 0xC:
        # CMP and branch instructions - check full opcode
        # CMP uses mode bits to distinguish addressing modes, but branches use mode bits differently
        # CMP: 0xC000 (mode 0), 0xC100 (mode 1), 0xC200 (mode 2), etc.
        # Branches: 0xC100 (BEQ), 0xC200 (BNE), 0xC300 (BGT), 0xC400 (BLT), 0xC500 (BGE), 0xC600 (BLE)
        # The issue: CMP immediate (mode 1) = 0xC100 conflicts with BEQ = 0xC100
        # Solution: Check if it's a branch first (branches have specific mode values), then CMP
        full_opcode = instruction & 0xFF00
        # Check for branch instructions first (they have specific mode values)
        if full_opcode == 0xC100 and mode == 1 and reg1 == 0 and reg2 == 0:
            # BEQ: 0xC100 with mode=1, reg1=0, reg2=0
            opcode = config.OP_BEQ
        elif full_opcode == 0xC200 and mode == 2 and reg1 == 0 and reg2 == 0:
            # BNE: 0xC200 with mode=2, reg1=0, reg2=0
            opcode = config.OP_BNE
        elif full_opcode == 0xC300 and mode == 3 and reg1 == 0 and reg2 == 0:
            # BGT: 0xC300 with mode=3, reg1=0, reg2=0
            opcode = config.OP_BGT
        elif full_opcode == 0xC400 and mode == 4 and reg1 == 0 and reg2 == 0:
            # BLT: 0xC400 with mode=4, reg1=0, reg2=0
            opcode = config.OP_BLT
        elif full_opcode == 0xC500 and mode == 5 and reg1 == 0 and reg2 == 0:
            # BGE: 0xC500 with mode=5, reg1=0, reg2=0
            opcode = config.OP_BGE
        elif full_opcode == 0xC600 and mode == 6 and reg1 == 0 and reg2 == 0:
            # BLE: 0xC600 with mode=6, reg1=0, reg2=0
            opcode = config.OP_BLE
        else:
            # Not a branch - must be CMP
            opcode = config.OP_CMP
    # else: opcode already set above from if/elif chain
    
    return opcode, mode, reg1, reg2


def cpu_execute_instruction(instruction):
    """Execute a single instruction"""
    import memory  # Import here to avoid circular import
    import ui
    emu = config.emulator
    
    # Decode instruction
    opcode, mode, reg1, reg2 = cpu_decode_instruction(instruction)
    
    # Log instruction execution (only if detailed logging enabled)
    # Cache opcode names to avoid dictionary lookup overhead when logging is disabled
    if ui.logger.enabled and ui.logger.detailed_logging:
        opcode_names = {
            config.OP_NOP: "NOP", config.OP_MOV: "MOV", config.OP_ADD: "ADD", 
            config.OP_SUB: "SUB", config.OP_MUL: "MUL", config.OP_DIV: "DIV",
            config.OP_AND: "AND", config.OP_OR: "OR", config.OP_XOR: "XOR",
            config.OP_NOT: "NOT", config.OP_SHL: "SHL", config.OP_SHR: "SHR",
            config.OP_CMP: "CMP", config.OP_BEQ: "BEQ", config.OP_BNE: "BNE",
            config.OP_BGT: "BGT", config.OP_BLT: "BLT", config.OP_BGE: "BGE",
            config.OP_BLE: "BLE", config.OP_JMP: "JMP", config.OP_CALL: "CALL",
            config.OP_RET: "RET"
        }
        op_name = opcode_names.get(opcode, f"OP{opcode:02X}")
        ui.logger.trace(f"Exec: {op_name} mode={mode} reg1={reg1} reg2={reg2} PC={emu.cpu.pc_bank:02X}:{emu.cpu.pc_offset:04X}", "CPU")
    
    # Optimized register access - cache logging state to avoid repeated checks
    _log_enabled = ui.logger.enabled and ui.logger.detailed_logging
    
    # Fast register getter - direct attribute access (no function call, no list creation)
    # Using direct attribute access is fastest in Python
    def get_register(reg_num):
        """Get register value by number - optimized with direct attribute access"""
        if reg_num == 0:
            return emu.cpu.r0
        elif reg_num == 1:
            return emu.cpu.r1
        elif reg_num == 2:
            return emu.cpu.r2
        elif reg_num == 3:
            return emu.cpu.r3
        elif reg_num == 4:
            return emu.cpu.r4
        elif reg_num == 5:
            return emu.cpu.r5
        elif reg_num == 6:
            return emu.cpu.r6
        elif reg_num == 7:
            return emu.cpu.r7
        return 0
    
    # Fast register setter - direct attribute access, conditional logging
    def set_register(reg_num, value):
        """Set register value by number - optimized"""
        value = value & 0xFFFF  # Ensure 16-bit
        if reg_num == 0:
            if _log_enabled:
                old = emu.cpu.r0
            emu.cpu.r0 = value
            if _log_enabled and old != value:
                ui.logger.trace(f"R0 = {value:04X} (was {old:04X})", "CPU")
        elif reg_num == 1:
            if _log_enabled:
                old = emu.cpu.r1
            emu.cpu.r1 = value
            if _log_enabled and old != value:
                ui.logger.trace(f"R1 = {value:04X} (was {old:04X})", "CPU")
        elif reg_num == 2:
            if _log_enabled:
                old = emu.cpu.r2
            emu.cpu.r2 = value
            if _log_enabled and old != value:
                ui.logger.trace(f"R2 = {value:04X} (was {old:04X})", "CPU")
        elif reg_num == 3:
            if _log_enabled:
                old = emu.cpu.r3
            emu.cpu.r3 = value
            if _log_enabled and old != value:
                ui.logger.trace(f"R3 = {value:04X} (was {old:04X})", "CPU")
        elif reg_num == 4:
            if _log_enabled:
                old = emu.cpu.r4
            emu.cpu.r4 = value
            if _log_enabled and old != value:
                ui.logger.trace(f"R4 = {value:04X} (was {old:04X})", "CPU")
        elif reg_num == 5:
            if _log_enabled:
                old = emu.cpu.r5
            emu.cpu.r5 = value
            if _log_enabled and old != value:
                ui.logger.trace(f"R5 = {value:04X} (was {old:04X})", "CPU")
        elif reg_num == 6:
            if _log_enabled:
                old = emu.cpu.r6
            emu.cpu.r6 = value
            if _log_enabled and old != value:
                ui.logger.trace(f"R6 = {value:04X} (was {old:04X})", "CPU")
        elif reg_num == 7:
            if _log_enabled:
                old = emu.cpu.r7
            emu.cpu.r7 = value
            if _log_enabled and old != value:
                ui.logger.trace(f"R7 = {value:04X} (was {old:04X})", "CPU")
    
    # Execute based on opcode
    cycles = 1  # Default cycle count
    
    if opcode == config.OP_NOP:
        # No operation
        pass
    
    elif opcode == config.OP_MOV:
        # MOV instruction
        # Mode 0: MOV R1, R2 (register to register)
        # Mode 1: MOV R1, #imm (immediate - next word is immediate value)
        # Mode 2: MOV R1, [R2] (load from memory at address in R2)
        # Mode 3: MOV [R1], R2 (store to memory at address in R1)
        # Mode 4: PUSH R1 (push register to stack)
        # Mode 5: POP R1 (pop stack to register)
        
        if mode == 4:  # PUSH R1
            value = get_register(reg1)
            # Push to stack (stack grows downward)
            emu.cpu.sp -= 2
            if emu.cpu.sp < 0:
                emu.cpu.sp = config.CPU_STACK_BASE  # Wrap around
            memory.memory_write16(0, emu.cpu.sp, value)
            cycles = 2
        
        elif mode == 5:  # POP R1
            # Pop from stack
            value = memory.memory_read16(0, emu.cpu.sp)
            emu.cpu.sp += 2
            if emu.cpu.sp > config.CPU_STACK_BASE:
                emu.cpu.sp = 0  # Wrap around
            set_register(reg1, value)
            cycles = 2
        
        elif mode == 0:  # Register to register
            src_value = get_register(reg2)
            set_register(reg1, src_value)
            cycles = 1
        
        elif mode == 1:  # Immediate
            # Fetch immediate value from next instruction word
            # Note: PC was already advanced by 2 in cpu_fetch_instruction, so we read from current PC
            imm = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            import ui
            ui.logger.trace(f"MOV R{reg1}, #{imm:04X}", "CPU")
            emu.cpu.pc_offset += 2
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            set_register(reg1, imm)
            cycles = 2
        
        elif mode == 2:  # Load from memory [R2]
            addr = get_register(reg2)
            # Read from memory using DBR:addr
            value = memory.memory_read16(emu.cpu.dbr, addr)
            set_register(reg1, value)
            cycles = 2
        
        elif mode == 3:  # Store to memory [R1]
            addr = get_register(reg1)
            value = get_register(reg2)
            # Write to memory using DBR:addr
            import ui
            ui.logger.trace(f"MOV [R{reg1}], R{reg2}: DBR={emu.cpu.dbr:02X}:{addr:04X} = {value:04X}", "CPU")
            memory.memory_write16(emu.cpu.dbr, addr, value)
            cycles = 2
    
    elif opcode == config.OP_ADD:
        # ADD instruction
        # Mode 0: ADD R1, R2 (R1 = R1 + R2)
        # Mode 1: ADD R1, #imm (R1 = R1 + immediate)
        # Mode 2: ADD R1, [R2] (R1 = R1 + memory[R2])
        
        dest_reg = reg1
        dest_value = get_register(dest_reg)
        
        if mode == 0:  # Register to register
            src_value = get_register(reg2)
            result = dest_value + src_value
            cycles = 1
        
        elif mode == 1:  # Immediate
            imm = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            src_value = imm
            result = dest_value + src_value
            cycles = 2
        
        elif mode == 2:  # Memory
            addr = get_register(reg2)
            src_value = memory.memory_read16(emu.cpu.dbr, addr)
            result = dest_value + src_value
            cycles = 2
        
        else:
            result = dest_value
            cycles = 1
        
        # Set result (16-bit wrap)
        result = result & 0xFFFF
        set_register(dest_reg, result)
        
        # Update flags
        # Zero flag
        cpu_set_flag(0, (result == 0))
        # Negative flag
        cpu_set_flag(1, ((result & 0x8000) != 0))
        # Carry flag (overflow beyond 16 bits)
        cpu_set_flag(2, (result < dest_value))
        # Overflow flag (signed overflow)
        # Overflow occurs when adding two positive numbers gives negative, or two negatives gives positive
        dest_signed = dest_value if dest_value < 0x8000 else dest_value - 0x10000
        src_signed = src_value if src_value < 0x8000 else src_value - 0x10000
        result_signed = result if result < 0x8000 else result - 0x10000
        overflow = ((dest_signed > 0 and src_signed > 0 and result_signed < 0) or
                   (dest_signed < 0 and src_signed < 0 and result_signed > 0))
        cpu_set_flag(3, overflow)
    
    elif opcode == config.OP_SUB:
        # SUB instruction
        # Mode 0: SUB R1, R2 (R1 = R1 - R2)
        # Mode 1: SUB R1, #imm (R1 = R1 - immediate)
        # Mode 2: SUB R1, [R2] (R1 = R1 - memory[R2])
        
        dest_reg = reg1
        dest_value = get_register(dest_reg)
        
        if mode == 0:  # Register to register
            src_value = get_register(reg2)
            result = dest_value - src_value
            cycles = 1
        
        elif mode == 1:  # Immediate
            imm = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            src_value = imm
            result = dest_value - src_value
            cycles = 2
        
        elif mode == 2:  # Memory
            addr = get_register(reg2)
            src_value = memory.memory_read16(emu.cpu.dbr, addr)
            result = dest_value - src_value
            cycles = 2
        
        else:
            result = dest_value
            cycles = 1
        
        # Set result (16-bit wrap)
        result = result & 0xFFFF
        set_register(dest_reg, result)
        
        # Update flags
        # Zero flag
        cpu_set_flag(0, (result == 0))
        # Negative flag
        cpu_set_flag(1, ((result & 0x8000) != 0))
        # Carry flag (borrow - set if result > dest_value, meaning we borrowed)
        cpu_set_flag(2, (result > dest_value))
        # Overflow flag (signed overflow)
        dest_signed = dest_value if dest_value < 0x8000 else dest_value - 0x10000
        src_signed = src_value if src_value < 0x8000 else src_value - 0x10000
        result_signed = result if result < 0x8000 else result - 0x10000
        overflow = ((dest_signed > 0 and src_signed < 0 and result_signed < 0) or
                   (dest_signed < 0 and src_signed > 0 and result_signed > 0))
        cpu_set_flag(3, overflow)
    
    elif opcode == config.OP_AND:
        # AND instruction (bitwise AND)
        # Mode 0: AND R1, R2 (R1 = R1 & R2)
        # Mode 1: AND R1, #imm (R1 = R1 & immediate)
        # Mode 2: AND R1, [R2] (R1 = R1 & memory[R2])
        
        dest_reg = reg1
        dest_value = get_register(dest_reg)
        
        if mode == 0:  # Register to register
            src_value = get_register(reg2)
            result = dest_value & src_value
            cycles = 1
        elif mode == 1:  # Immediate
            imm = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            result = dest_value & imm
            cycles = 2
        elif mode == 2:  # Memory
            addr = get_register(reg2)
            src_value = memory.memory_read16(emu.cpu.dbr, addr)
            result = dest_value & src_value
            cycles = 2
        else:
            result = dest_value
            cycles = 1
        
        result = result & 0xFFFF
        set_register(dest_reg, result)
        cpu_update_flags(result, True)  # Update Z and N flags
    
    elif opcode == config.OP_OR:
        # OR instruction (bitwise OR)
        # Mode 0: OR R1, R2 (R1 = R1 | R2)
        # Mode 1: OR R1, #imm (R1 = R1 | immediate)
        # Mode 2: OR R1, [R2] (R1 = R1 | memory[R2])
        
        dest_reg = reg1
        dest_value = get_register(dest_reg)
        
        if mode == 0:  # Register to register
            src_value = get_register(reg2)
            result = dest_value | src_value
            cycles = 1
        elif mode == 1:  # Immediate
            imm = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            result = dest_value | imm
            cycles = 2
        elif mode == 2:  # Memory
            addr = get_register(reg2)
            src_value = memory.memory_read16(emu.cpu.dbr, addr)
            result = dest_value | src_value
            cycles = 2
        else:
            result = dest_value
            cycles = 1
        
        result = result & 0xFFFF
        set_register(dest_reg, result)
        cpu_update_flags(result, True)  # Update Z and N flags
    
    elif opcode == config.OP_XOR:
        # XOR instruction (bitwise XOR)
        # Mode 0: XOR R1, R2 (R1 = R1 ^ R2)
        # Mode 1: XOR R1, #imm (R1 = R1 ^ immediate)
        # Mode 2: XOR R1, [R2] (R1 = R1 ^ memory[R2])
        
        dest_reg = reg1
        dest_value = get_register(dest_reg)
        
        if mode == 0:  # Register to register
            src_value = get_register(reg2)
            result = dest_value ^ src_value
            cycles = 1
        elif mode == 1:  # Immediate
            imm = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            result = dest_value ^ imm
            cycles = 2
        elif mode == 2:  # Memory
            addr = get_register(reg2)
            src_value = memory.memory_read16(emu.cpu.dbr, addr)
            result = dest_value ^ src_value
            cycles = 2
        else:
            result = dest_value
            cycles = 1
        
        result = result & 0xFFFF
        set_register(dest_reg, result)
        cpu_update_flags(result, True)  # Update Z and N flags
    
    elif opcode == config.OP_NOT:
        # NOT instruction (bitwise NOT - one operand)
        # Mode 0: NOT R1 (R1 = ~R1)
        # reg2 is unused
        
        dest_reg = reg1
        dest_value = get_register(dest_reg)
        result = (~dest_value) & 0xFFFF  # Invert and mask to 16 bits
        set_register(dest_reg, result)
        cpu_update_flags(result, True)  # Update Z and N flags
        cycles = 1
    
    elif opcode == config.OP_MUL:
        # MUL instruction (multiply)
        # Mode 0: MUL R1, R2 (R1 = R1 * R2)
        # Mode 1: MUL R1, #imm (R1 = R1 * immediate)
        # Mode 2: MUL R1, [R2] (R1 = R1 * memory[R2])
        
        dest_reg = reg1
        dest_value = get_register(dest_reg)
        
        if mode == 0:  # Register to register
            src_value = get_register(reg2)
            result = dest_value * src_value
            cycles = 2  # Multiplication takes longer
        elif mode == 1:  # Immediate
            imm = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            src_value = imm
            result = dest_value * src_value
            cycles = 3
        elif mode == 2:  # Memory
            addr = get_register(reg2)
            src_value = memory.memory_read16(emu.cpu.dbr, addr)
            result = dest_value * src_value
            cycles = 3
        else:
            result = dest_value
            cycles = 1
        
        # Set result (16-bit wrap)
        result = result & 0xFFFF
        set_register(dest_reg, result)
        
        # Update flags
        cpu_set_flag(0, (result == 0))
        cpu_set_flag(1, ((result & 0x8000) != 0))
        # Carry flag: set if result overflowed 16 bits
        full_result = dest_value * src_value
        cpu_set_flag(2, (full_result > 0xFFFF))
        # Overflow flag: signed overflow
        dest_signed = dest_value if dest_value < 0x8000 else dest_value - 0x10000
        src_signed = src_value if src_value < 0x8000 else src_value - 0x10000
        result_signed = result if result < 0x8000 else result - 0x10000
        full_signed = dest_signed * src_signed
        overflow = (full_signed != result_signed)
        cpu_set_flag(3, overflow)
    
    elif opcode == config.OP_DIV:
        # DIV instruction (divide)
        # Mode 0: DIV R1, R2 (R1 = R1 / R2)
        # Mode 1: DIV R1, #imm (R1 = R1 / immediate)
        # Mode 2: DIV R1, [R2] (R1 = R1 / memory[R2])
        
        dest_reg = reg1
        dest_value = get_register(dest_reg)
        
        if mode == 0:  # Register to register
            src_value = get_register(reg2)
            cycles = 4  # Division takes longer
        elif mode == 1:  # Immediate
            imm = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            src_value = imm
            cycles = 5
        elif mode == 2:  # Memory
            addr = get_register(reg2)
            src_value = memory.memory_read16(emu.cpu.dbr, addr)
            cycles = 5
        else:
            src_value = 1
            cycles = 1
        
        # Handle division by zero
        if src_value == 0:
            # Division by zero - set flags and leave result unchanged
            cpu_set_flag(0, False)  # Not zero
            cpu_set_flag(1, True)   # Negative (error indicator)
            cpu_set_flag(2, True)   # Carry (error)
            # Don't change register
        else:
            result = dest_value // src_value  # Integer division
            result = result & 0xFFFF
            set_register(dest_reg, result)
            
            # Update flags
            cpu_set_flag(0, (result == 0))
            cpu_set_flag(1, ((result & 0x8000) != 0))
            cpu_set_flag(2, False)  # No carry for division
            cpu_set_flag(3, False)   # No overflow for division
    
    elif opcode == config.OP_SHL:
        # SHL instruction (shift left)
        # Mode 0: SHL R1, R2 (R1 = R1 << R2, shift count in R2)
        # Mode 1: SHL R1, #imm (R1 = R1 << immediate)
        # Mode 2: SHL R1, [R2] (R1 = R1 << memory[R2])
        
        dest_reg = reg1
        dest_value = get_register(dest_reg)
        
        if mode == 0:  # Register to register
            shift_count = get_register(reg2) & 0xF  # Limit to 0-15
            result = (dest_value << shift_count) & 0xFFFF
            cycles = 1
        elif mode == 1:  # Immediate
            imm = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            shift_count = imm & 0xF  # Limit to 0-15
            result = (dest_value << shift_count) & 0xFFFF
            cycles = 2
        elif mode == 2:  # Memory
            addr = get_register(reg2)
            shift_count = memory.memory_read16(emu.cpu.dbr, addr) & 0xF
            result = (dest_value << shift_count) & 0xFFFF
            cycles = 2
        else:
            result = dest_value
            cycles = 1
        
        set_register(dest_reg, result)
        
        # Update flags
        cpu_set_flag(0, (result == 0))
        cpu_set_flag(1, ((result & 0x8000) != 0))
        # Carry flag: bit shifted out (MSB before shift)
        if shift_count > 0:
            bit_shifted_out = (dest_value >> (16 - shift_count)) & 1
            cpu_set_flag(2, (bit_shifted_out != 0))
        else:
            cpu_set_flag(2, False)
        cpu_set_flag(3, False)  # No overflow for shift
    
    elif opcode == config.OP_SHR:
        # SHR instruction (shift right, logical)
        # Mode 0: SHR R1, R2 (R1 = R1 >> R2, shift count in R2)
        # Mode 1: SHR R1, #imm (R1 = R1 >> immediate)
        # Mode 2: SHR R1, [R2] (R1 = R1 >> memory[R2])
        
        dest_reg = reg1
        dest_value = get_register(dest_reg)
        
        if mode == 0:  # Register to register
            shift_count = get_register(reg2) & 0xF  # Limit to 0-15
            result = dest_value >> shift_count
            cycles = 1
        elif mode == 1:  # Immediate
            imm = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            shift_count = imm & 0xF  # Limit to 0-15
            result = dest_value >> shift_count
            cycles = 2
        elif mode == 2:  # Memory
            addr = get_register(reg2)
            shift_count = memory.memory_read16(emu.cpu.dbr, addr) & 0xF
            result = dest_value >> shift_count
            cycles = 2
        else:
            result = dest_value
            cycles = 1
        
        result = result & 0xFFFF
        set_register(dest_reg, result)
        
        # Update flags
        cpu_set_flag(0, (result == 0))
        cpu_set_flag(1, False)  # Logical shift right always clears sign bit
        # Carry flag: bit shifted out (LSB)
        if shift_count > 0:
            bit_shifted_out = (dest_value >> (shift_count - 1)) & 1
            cpu_set_flag(2, (bit_shifted_out != 0))
        else:
            cpu_set_flag(2, False)
        cpu_set_flag(3, False)  # No overflow for shift
    
    elif opcode == config.OP_CALL:
        # CALL instruction (subroutine call)
        # Mode 0: CALL addr (absolute call - next word is offset, reg1 is bank)
        # Mode 1: CALL rel (relative call - next word is signed offset)
        # Mode 2: CALL [R1] (indirect call - address in register)
        
        # Calculate return address (PC after this instruction completes)
        return_pc_offset = emu.cpu.pc_offset
        return_pc_bank = emu.cpu.pbr
        
        if mode == 0:  # Absolute call
            # Read offset word
            new_bank = reg1 & 0x7F
            new_offset = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            return_pc_offset += 2  # After reading offset word
            if return_pc_offset >= config.MEMORY_BANK_SIZE:
                return_pc_offset = return_pc_offset % config.MEMORY_BANK_SIZE
            emu.cpu.pc_offset = new_offset
            emu.cpu.pbr = new_bank
            cycles = 3
        elif mode == 1:  # Relative call
            # Read offset word
            offset = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            return_pc_offset += 2  # After reading offset word
            if return_pc_offset >= config.MEMORY_BANK_SIZE:
                return_pc_offset = return_pc_offset % config.MEMORY_BANK_SIZE
            if offset & 0x8000:
                offset = offset | 0xFFFF0000
            new_offset = (return_pc_offset + offset) & 0xFFFF
            emu.cpu.pc_offset = new_offset
            cycles = 3
        elif mode == 2:  # Indirect call
            addr = get_register(reg1)
            emu.cpu.pc_offset = addr
            cycles = 3
        
        # Save return address (push bank first, then offset)
        emu.cpu.sp -= 2
        if emu.cpu.sp < 0:
            emu.cpu.sp = config.CPU_STACK_BASE
        memory.memory_write16(0, emu.cpu.sp, return_pc_bank)
        emu.cpu.sp -= 2
        if emu.cpu.sp < 0:
            emu.cpu.sp = config.CPU_STACK_BASE
        memory.memory_write16(0, emu.cpu.sp, return_pc_offset)
    
    elif opcode == config.OP_RET:
        # RET instruction (return from subroutine)
        # Pop return address from stack (offset first, then bank)
        import ui
        
        # Pop PC offset and bank
        return_pc_offset = memory.memory_read16(0, emu.cpu.sp)
        emu.cpu.sp += 2
        if emu.cpu.sp > config.CPU_STACK_BASE:
            emu.cpu.sp = 0
        return_pc_bank = memory.memory_read16(0, emu.cpu.sp)
        emu.cpu.sp += 2
        if emu.cpu.sp > config.CPU_STACK_BASE:
            emu.cpu.sp = 0
        
        # Validate return address - if stack was empty (all zeros), this is invalid
        if return_pc_bank == 0 and return_pc_offset == 0:
            ui.logger.error(f"CPU ERROR: RET attempted with empty stack (popped 00:0000)! This usually means RET without matching CALL. Bootstrapping to ROM entry point.", "CPU")
            # Bootstrap to ROM entry point instead
            if hasattr(emu, 'rom') and emu.rom and emu.rom.entry_point_bank > 0:
                return_pc_bank = emu.rom.entry_point_bank
                return_pc_offset = emu.rom.entry_point_offset
            else:
                # No ROM - can't bootstrap, just log error
                ui.logger.error(f"CPU ERROR: No ROM loaded, cannot bootstrap from invalid RET!", "CPU")
        
        # Validate return address is in valid ROM region (if returning to ROM)
        if config.MEMORY_ROM_START_BANK <= return_pc_bank <= config.MEMORY_ROM_END_BANK:
            if return_pc_offset < 0x8000:
                ui.logger.error(f"CPU ERROR: RET would return to invalid ROM region! {return_pc_bank:02X}:{return_pc_offset:04X}, clamping to 0x8000", "CPU")
                return_pc_offset = 0x8000
            elif return_pc_offset >= 0x10000:
                ui.logger.error(f"CPU ERROR: RET would return beyond bank! {return_pc_bank:02X}:{return_pc_offset:04X}, clamping to 0xFFFF", "CPU")
                return_pc_offset = 0xFFFF
        
        # Restore PC
        emu.cpu.pc_offset = return_pc_offset
        emu.cpu.pbr = return_pc_bank
        emu.cpu.pc_bank = return_pc_bank  # Sync pc_bank
        ui.logger.trace(f"RET: Returning to {return_pc_bank:02X}:{return_pc_offset:04X}", "CPU")
        cycles = 3
    
    elif opcode == config.OP_CMP:
        # CMP instruction (compare - sets flags but doesn't store result)
        # Mode 0: CMP R1, R2 (compare R1 with R2)
        # Mode 1: CMP R1, #imm (compare R1 with immediate)
        # Mode 2: CMP R1, [R2] (compare R1 with memory[R2])
        
        dest_value = get_register(reg1)
        
        if mode == 0:  # Register to register
            src_value = get_register(reg2)
            result = dest_value - src_value
            cycles = 1
        elif mode == 1:  # Immediate
            imm = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            src_value = imm
            result = dest_value - src_value
            cycles = 2
        elif mode == 2:  # Memory
            addr = get_register(reg2)
            src_value = memory.memory_read16(emu.cpu.dbr, addr)
            result = dest_value - src_value
            cycles = 2
        else:
            result = 0
            cycles = 1
        
        # Set flags based on comparison (don't store result)
        result = result & 0xFFFF
        z_flag = (result == 0)
        n_flag = ((result & 0x8000) != 0)
        c_flag = (result > dest_value)  # Carry flag (borrow - set if dest < src)
        # Overflow flag (signed comparison)
        dest_signed = dest_value if dest_value < 0x8000 else dest_value - 0x10000
        src_signed = src_value if src_value < 0x8000 else src_value - 0x10000
        result_signed = result if result < 0x8000 else result - 0x10000
        overflow = ((dest_signed > 0 and src_signed < 0 and result_signed < 0) or
                   (dest_signed < 0 and src_signed > 0 and result_signed > 0))
        
        cpu_set_flag(0, z_flag)  # Zero flag (equal)
        cpu_set_flag(1, n_flag)  # Negative flag (less than)
        cpu_set_flag(2, c_flag)  # Carry flag
        cpu_set_flag(3, overflow)  # Overflow flag
        
        # Log flag changes for debugging (only if detailed logging enabled)
        import ui
        if ui.logger.enabled and ui.logger.detailed_logging:
            ui.logger.trace(f"CMP R{reg1}, {'R' + str(reg2) if mode == 0 else ('#' + str(src_value) if mode == 1 else '[R' + str(reg2) + ']')}: dest={dest_value:04X}, src={src_value:04X}, result={result:04X}, flags: Z={z_flag} N={n_flag} C={c_flag} V={overflow}", "CPU")
    
    elif opcode == config.OP_BEQ:
        # BEQ - Branch if equal (Z flag set)
        # Branch instructions: mode 0 = relative, mode 1 = absolute
        # But instruction format has mode in [11:8], so BEQ (0xC100) has mode=1
        # We'll treat mode 0 and 1 as relative/absolute based on the actual mode value
        import ui
        z_flag = cpu_get_flag(0)
        if z_flag:
            # For branch instructions, check if mode indicates relative (0) or absolute (1)
            # But BEQ=0xC100 has mode=1 in the instruction, so we need to check mode differently
            # Actually, let's use the mode from decode: if mode==0, relative; if mode==1, absolute
            if mode == 0:  # Relative (but BEQ=0xC100 has mode=1, so this won't match)
                # This path won't be taken for 0xC100
                offset = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
                emu.cpu.pc_offset += 2
                if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                    emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
                if offset & 0x8000:
                    offset = offset | 0xFFFF0000
                new_pc = (emu.cpu.pc_offset + offset) & 0xFFFF
                emu.cpu.pc_offset = new_pc
                cycles = 2
            elif mode == 1:  # Absolute (this is what 0xC100 gives us)
                # But we want relative for the test! Let's make mode 1 also do relative for branches
                # Actually, let's just make all BEQ relative for now
                offset = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
                emu.cpu.pc_offset += 2
                # Handle wrapping for ROM banks
                if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                    if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                        emu.cpu.pc_offset = 0x8000 + (emu.cpu.pc_offset - config.MEMORY_BANK_SIZE)
                        if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                            emu.cpu.pc_offset = 0x8000
                    else:
                        emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
                if offset & 0x8000:
                    offset = offset | 0xFFFF0000
                old_pc = emu.cpu.pc_offset
                new_pc = (emu.cpu.pc_offset + offset) & 0xFFFF
                
                # Validate new PC is in valid ROM region (for ROM banks)
                if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                    if new_pc < 0x8000:
                        ui.logger.error(f"CPU ERROR: BEQ would branch to invalid ROM region! PC={old_pc:04X}, offset={offset:+d}, new={new_pc:04X}, clamping to 0x8000", "CPU")
                        new_pc = 0x8000
                    elif new_pc >= 0x10000:
                        ui.logger.error(f"CPU ERROR: BEQ would branch beyond bank! PC={old_pc:04X}, offset={offset:+d}, new={new_pc:04X}, clamping to 0xFFFF", "CPU")
                        new_pc = 0xFFFF
                
                emu.cpu.pc_offset = new_pc
                ui.logger.trace(f"BEQ: Branch taken, PC {old_pc:04X} -> {new_pc:04X} (offset={offset:+d})", "CPU")
                cycles = 2
            else:
                cycles = 1
        else:
            # Don't branch - skip offset word
            emu.cpu.pc_offset += 2
            # Handle wrapping for ROM banks
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                    emu.cpu.pc_offset = 0x8000 + (emu.cpu.pc_offset - config.MEMORY_BANK_SIZE)
                    if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                        emu.cpu.pc_offset = 0x8000
                else:
                    emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            ui.logger.trace(f"BEQ: Branch not taken (Z=0)", "CPU")
            cycles = 2
    
    elif opcode == config.OP_BNE:
        # BNE - Branch if not equal (Z flag clear)
        # Branch instructions always use relative mode
        z_flag = cpu_get_flag(0)
        if not z_flag:
            # Relative branch
            offset = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2
            # Handle wrapping for ROM banks
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                    emu.cpu.pc_offset = 0x8000 + (emu.cpu.pc_offset - config.MEMORY_BANK_SIZE)
                    if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                        emu.cpu.pc_offset = 0x8000
                else:
                    emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            if offset & 0x8000:
                offset = offset | 0xFFFF0000
            old_pc = emu.cpu.pc_offset
            new_pc = (emu.cpu.pc_offset + offset) & 0xFFFF
            
            # Validate new PC is in valid ROM region (for ROM banks)
            if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                if new_pc < 0x8000:
                    import ui
                    ui.logger.error(f"CPU ERROR: BNE would branch to invalid ROM region! PC={old_pc:04X}, offset={offset:+d}, new={new_pc:04X}, clamping to 0x8000", "CPU")
                    new_pc = 0x8000
                elif new_pc >= 0x10000:
                    import ui
                    ui.logger.error(f"CPU ERROR: BNE would branch beyond bank! PC={old_pc:04X}, offset={offset:+d}, new={new_pc:04X}, clamping to 0xFFFF", "CPU")
                    new_pc = 0xFFFF
            
            emu.cpu.pc_offset = new_pc
            cycles = 2
        else:
            # Don't branch - skip offset word
            emu.cpu.pc_offset += 2
            # Handle wrapping for ROM banks
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                    emu.cpu.pc_offset = 0x8000 + (emu.cpu.pc_offset - config.MEMORY_BANK_SIZE)
                    if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                        emu.cpu.pc_offset = 0x8000
                else:
                    emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            cycles = 2
    
    elif opcode == config.OP_BGT:
        # BGT - Branch if greater than (Z=0 and N=0)
        # Branch instructions always use relative mode
        z_flag = cpu_get_flag(0)
        n_flag = cpu_get_flag(1)
        if not z_flag and not n_flag:
            # Relative branch
            offset = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2
            # Handle wrapping for ROM banks
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                    emu.cpu.pc_offset = 0x8000 + (emu.cpu.pc_offset - config.MEMORY_BANK_SIZE)
                    if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                        emu.cpu.pc_offset = 0x8000
                else:
                    emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            if offset & 0x8000:
                offset = offset | 0xFFFF0000
            old_pc = emu.cpu.pc_offset
            new_pc = (emu.cpu.pc_offset + offset) & 0xFFFF
            
            # Validate new PC is in valid ROM region (for ROM banks)
            if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                if new_pc < 0x8000:
                    import ui
                    ui.logger.error(f"CPU ERROR: BGT would branch to invalid ROM region! PC={old_pc:04X}, offset={offset:+d}, new={new_pc:04X}, clamping to 0x8000", "CPU")
                    new_pc = 0x8000
                elif new_pc >= 0x10000:
                    import ui
                    ui.logger.error(f"CPU ERROR: BGT would branch beyond bank! PC={old_pc:04X}, offset={offset:+d}, new={new_pc:04X}, clamping to 0xFFFF", "CPU")
                    new_pc = 0xFFFF
            
            emu.cpu.pc_offset = new_pc
            cycles = 2
        else:
            # Don't branch - skip offset word
            emu.cpu.pc_offset += 2
            # Handle wrapping for ROM banks
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                    emu.cpu.pc_offset = 0x8000 + (emu.cpu.pc_offset - config.MEMORY_BANK_SIZE)
                    if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                        emu.cpu.pc_offset = 0x8000
                else:
                    emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            cycles = 2
    
    elif opcode == config.OP_BLT:
        # BLT - Branch if less than (N flag set)
        # Branch instructions always use relative mode
        import ui
        n_flag = cpu_get_flag(1)
        z_flag = cpu_get_flag(0)
        current_flags = emu.cpu.flags
        
        # Log flag state for debugging (only if detailed logging enabled)
        if ui.logger.enabled and ui.logger.detailed_logging:
            ui.logger.trace(f"BLT: PC={emu.cpu.pc_offset:04X}, flags=0x{current_flags:04X}, N={n_flag}, Z={z_flag}, will {'BRANCH' if n_flag else 'NOT BRANCH'}", "CPU")
        
        if n_flag:
            # Relative branch
            offset = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2
            # Handle wrapping for ROM banks
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                    emu.cpu.pc_offset = 0x8000 + (emu.cpu.pc_offset - config.MEMORY_BANK_SIZE)
                    if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                        emu.cpu.pc_offset = 0x8000
                else:
                    emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            if offset & 0x8000:
                offset = offset | 0xFFFF0000
            old_pc = emu.cpu.pc_offset
            new_pc = (emu.cpu.pc_offset + offset) & 0xFFFF
            
            # Validate new PC is in valid ROM region (for ROM banks)
            if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                if new_pc < 0x8000:
                    ui.logger.error(f"CPU ERROR: BLT would branch to invalid ROM region! PC={old_pc:04X}, offset={offset:+d}, new={new_pc:04X}, clamping to 0x8000", "CPU")
                    new_pc = 0x8000
                elif new_pc >= 0x10000:
                    ui.logger.error(f"CPU ERROR: BLT would branch beyond bank! PC={old_pc:04X}, offset={offset:+d}, new={new_pc:04X}, clamping to 0xFFFF", "CPU")
                    new_pc = 0xFFFF
            
            emu.cpu.pc_offset = new_pc
            if ui.logger.enabled and ui.logger.detailed_logging:
                ui.logger.trace(f"BLT: Branch taken, PC {old_pc:04X} -> {new_pc:04X} (offset={offset:+d})", "CPU")
            cycles = 2
        else:
            # Don't branch - skip offset word
            emu.cpu.pc_offset += 2
            # Handle wrapping for ROM banks
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                    emu.cpu.pc_offset = 0x8000 + (emu.cpu.pc_offset - config.MEMORY_BANK_SIZE)
                    if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                        emu.cpu.pc_offset = 0x8000
                else:
                    emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            if ui.logger.enabled and ui.logger.detailed_logging:
                ui.logger.trace(f"BLT: Branch not taken (N=0), PC={emu.cpu.pc_offset:04X}", "CPU")
            cycles = 2
    
    elif opcode == config.OP_BGE:
        # BGE - Branch if greater or equal (N=0)
        # Branch instructions always use relative mode
        n_flag = cpu_get_flag(1)
        if not n_flag:
            # Relative branch
            offset = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2
            # Handle wrapping for ROM banks
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                    emu.cpu.pc_offset = 0x8000 + (emu.cpu.pc_offset - config.MEMORY_BANK_SIZE)
                    if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                        emu.cpu.pc_offset = 0x8000
                else:
                    emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            if offset & 0x8000:
                offset = offset | 0xFFFF0000
            old_pc = emu.cpu.pc_offset
            new_pc = (emu.cpu.pc_offset + offset) & 0xFFFF
            
            # Validate new PC is in valid ROM region (for ROM banks)
            if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                if new_pc < 0x8000:
                    import ui
                    ui.logger.error(f"CPU ERROR: BGE would branch to invalid ROM region! PC={old_pc:04X}, offset={offset:+d}, new={new_pc:04X}, clamping to 0x8000", "CPU")
                    new_pc = 0x8000
                elif new_pc >= 0x10000:
                    import ui
                    ui.logger.error(f"CPU ERROR: BGE would branch beyond bank! PC={old_pc:04X}, offset={offset:+d}, new={new_pc:04X}, clamping to 0xFFFF", "CPU")
                    new_pc = 0xFFFF
            
            emu.cpu.pc_offset = new_pc
            cycles = 2
        else:
            # Don't branch - skip offset word
            emu.cpu.pc_offset += 2
            # Handle wrapping for ROM banks
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                    emu.cpu.pc_offset = 0x8000 + (emu.cpu.pc_offset - config.MEMORY_BANK_SIZE)
                    if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                        emu.cpu.pc_offset = 0x8000
                else:
                    emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            cycles = 2
    
    elif opcode == config.OP_BLE:
        # BLE - Branch if less or equal (Z=1 or N=1)
        # Branch instructions always use relative mode
        z_flag = cpu_get_flag(0)
        n_flag = cpu_get_flag(1)
        if z_flag or n_flag:
            # Relative branch
            offset = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2
            # Handle wrapping for ROM banks
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                    emu.cpu.pc_offset = 0x8000 + (emu.cpu.pc_offset - config.MEMORY_BANK_SIZE)
                    if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                        emu.cpu.pc_offset = 0x8000
                else:
                    emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            if offset & 0x8000:
                offset = offset | 0xFFFF0000
            old_pc = emu.cpu.pc_offset
            new_pc = (emu.cpu.pc_offset + offset) & 0xFFFF
            
            # Validate new PC is in valid ROM region (for ROM banks)
            if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                if new_pc < 0x8000:
                    import ui
                    ui.logger.error(f"CPU ERROR: BLE would branch to invalid ROM region! PC={old_pc:04X}, offset={offset:+d}, new={new_pc:04X}, clamping to 0x8000", "CPU")
                    new_pc = 0x8000
                elif new_pc >= 0x10000:
                    import ui
                    ui.logger.error(f"CPU ERROR: BLE would branch beyond bank! PC={old_pc:04X}, offset={offset:+d}, new={new_pc:04X}, clamping to 0xFFFF", "CPU")
                    new_pc = 0xFFFF
            
            emu.cpu.pc_offset = new_pc
            cycles = 2
        else:
            # Don't branch - skip offset word
            emu.cpu.pc_offset += 2
            # Handle wrapping for ROM banks
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                    emu.cpu.pc_offset = 0x8000 + (emu.cpu.pc_offset - config.MEMORY_BANK_SIZE)
                    if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                        emu.cpu.pc_offset = 0x8000
                else:
                    emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            cycles = 2
    
    elif opcode == config.OP_JMP:
        # JMP instruction
        # Mode 0: JMP addr (absolute - next word is offset, reg1 is bank)
        # Mode 1: JMP rel (relative - next word is signed 16-bit offset)
        # Mode 2: JMP [R1] (indirect - jump to address in register)
        
        import ui
        if mode == 0:  # Absolute jump
            # reg1 contains bank, next word contains offset
            # Note: PC was already incremented by fetch_instruction, so we read from current PC
            new_bank = reg1 & 0x7F  # 7-bit bank number
            new_offset = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2  # Consume the offset word
            if emu.cpu.pc_offset >= config.MEMORY_BANK_SIZE:
                emu.cpu.pc_offset = emu.cpu.pc_offset % config.MEMORY_BANK_SIZE
            # Now set PC to the new location
            old_pc = f"{emu.cpu.pc_bank:02X}:{emu.cpu.pc_offset:04X}"
            
            # Validate new PC is in valid ROM region (for ROM banks)
            if config.MEMORY_ROM_START_BANK <= new_bank <= config.MEMORY_ROM_END_BANK:
                if new_offset < 0x8000:
                    ui.logger.error(f"CPU ERROR: JMP abs would jump to invalid ROM region! {old_pc} -> {new_bank:02X}:{new_offset:04X}, clamping to 0x8000", "CPU")
                    new_offset = 0x8000
                elif new_offset >= 0x10000:
                    ui.logger.error(f"CPU ERROR: JMP abs would jump beyond bank! {old_pc} -> {new_bank:02X}:{new_offset:04X}, clamping to 0xFFFF", "CPU")
                    new_offset = 0xFFFF
            
            emu.cpu.pc_offset = new_offset
            emu.cpu.pbr = new_bank
            ui.logger.trace(f"JMP abs: {old_pc} -> {new_bank:02X}:{new_offset:04X}", "CPU")
            cycles = 2
        
        elif mode == 1:  # Relative jump
            # Next word contains signed 16-bit offset
            jmp_pc_before_read = emu.cpu.pc_offset
            offset = memory.memory_read16(emu.cpu.pbr, emu.cpu.pc_offset)
            emu.cpu.pc_offset += 2  # Consume the offset word
            # Sign extend if bit 15 is set
            if offset & 0x8000:
                offset = offset | 0xFFFF0000  # Sign extend to 32-bit
            # Add to current PC offset
            old_pc = emu.cpu.pc_offset
            new_offset = (emu.cpu.pc_offset + offset) & 0xFFFF
            
            # Enhanced logging for JMP rel
            ui.logger.trace(f"JMP rel: PC before read={jmp_pc_before_read:04X}, offset word=0x{offset & 0xFFFF:04X} (signed={offset:+d}), PC after read={old_pc:04X}, new PC={new_offset:04X}", "CPU")
            
            # Validate new PC is in valid ROM region (for ROM banks)
            if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                if new_offset < 0x8000:
                    ui.logger.error(f"CPU ERROR: JMP rel would jump to invalid ROM region! PC={old_pc:04X}, offset={offset:+d}, new={new_offset:04X}, clamping to 0x8000", "CPU")
                    new_offset = 0x8000
                elif new_offset >= 0x10000:
                    ui.logger.error(f"CPU ERROR: JMP rel would jump beyond bank! PC={old_pc:04X}, offset={offset:+d}, new={new_offset:04X}, clamping to 0xFFFF", "CPU")
                    new_offset = 0xFFFF
            
            emu.cpu.pc_offset = new_offset
            ui.logger.trace(f"JMP rel: PC {old_pc:04X} -> {new_offset:04X} (offset={offset:+d})", "CPU")
            cycles = 2
        
        elif mode == 2:  # Indirect jump [R1]
            addr = get_register(reg1)
            old_pc = emu.cpu.pc_offset
            # Jump to address in register (using current bank)
            new_offset = addr & 0xFFFF
            
            # Validate new PC is in valid ROM region (for ROM banks)
            if config.MEMORY_ROM_START_BANK <= emu.cpu.pbr <= config.MEMORY_ROM_END_BANK:
                if new_offset < 0x8000:
                    ui.logger.error(f"CPU ERROR: JMP [R{reg1}] would jump to invalid ROM region! PC={old_pc:04X}, addr={addr:04X}, clamping to 0x8000", "CPU")
                    new_offset = 0x8000
                elif new_offset >= 0x10000:
                    ui.logger.error(f"CPU ERROR: JMP [R{reg1}] would jump beyond bank! PC={old_pc:04X}, addr={addr:04X}, clamping to 0xFFFF", "CPU")
                    new_offset = 0xFFFF
            
            emu.cpu.pc_offset = new_offset
            ui.logger.trace(f"JMP [R{reg1}]: PC {old_pc:04X} -> {new_offset:04X}", "CPU")
            cycles = 2
    
    # Update cycle counter
    emu.cpu.cycles += cycles
    
    # Capture CPU state if enabled (for hex debugger CPU state capture)
    # Only capture once per instruction (not per cycle)
    try:
        import ui
        if hasattr(ui, 'hex_debugger_instance') and ui.hex_debugger_instance:
            if ui.hex_debugger_instance.cpu_capture_enabled:
                ui.hex_debugger_instance.capture_cpu_state()
    except:
        pass  # Silently fail if ui module not available


# Try to import Cython-optimized CPU module (optional)
_cython_available = False
_cython_initialized = False
try:
    import cpu_cython
    _cython_available = True
except ImportError:
    pass

def cpu_run_cycles(target_cycles):
    """Run CPU for specified number of cycles"""
    global _cython_initialized
    emu = config.emulator
    
    # Initialize Cython module if available (first time only)
    if _cython_available and not _cython_initialized:
        try:
            cpu_cython.init_cython_cpu()
            _cython_initialized = True
        except:
            pass  # Fall back to Python if initialization fails
    
    # TODO: Implement cycle-accurate execution
    # For now, just execute instructions until cycle limit
    while emu.cpu.cycles < target_cycles and not emu.paused:
        # Check for pending interrupts
        if emu.cpu.interrupt_pending != config.INT_NONE:
            cpu_handle_interrupt()
        
        # Fetch and decode instruction
        instruction = cpu_fetch_instruction()
        opcode, mode, reg1, reg2 = cpu_decode_instruction(instruction)
        
        # Try Cython execution first (if available), fall back to Python
        if _cython_available and _cython_initialized:
            try:
                cycles = cpu_cython.cpu_execute_instruction_cython(instruction, opcode, mode, reg1, reg2)
                if cycles > 0:
                    # Cython handled the instruction - update cycle counter
                    emu.cpu.cycles += cycles
                else:
                    # Cython didn't handle it (returned 0) - fall back to Python
                    cpu_execute_instruction(instruction)
            except:
                # Fall back to Python execution if Cython fails
                cpu_execute_instruction(instruction)
        else:
            # Use Python execution
            cpu_execute_instruction(instruction)
        
        # Step mode: pause after each instruction
        if emu.step_mode:
            emu.paused = True


def cpu_handle_interrupt():
    """Handle interrupt"""
    emu = config.emulator
    
    # Check if interrupts are enabled
    if (emu.cpu.flags & 0x10) == 0:  # I flag clear = interrupts enabled
        int_type = emu.cpu.interrupt_pending
        emu.cpu.interrupt_pending = config.INT_NONE
        
        # TODO: Push PC and flags to stack
        # TODO: Set I flag (disable interrupts)
        # TODO: Jump to interrupt vector based on intType
        # For now, just clear pending interrupt
        pass


def cpu_trigger_vblank():
    """Trigger VBlank interrupt"""
    emu = config.emulator
    
    if (emu.cpu.flags & 0x10) == 0:  # Interrupts enabled
        emu.cpu.interrupt_pending = config.INT_VBLANK


def cpu_set_flag(flag_bit, value):
    """Set CPU flag"""
    emu = config.emulator
    
    if value:
        emu.cpu.flags = emu.cpu.flags | (1 << flag_bit)
    else:
        emu.cpu.flags = emu.cpu.flags & ~(1 << flag_bit)


def cpu_get_flag(flag_bit):
    """Get CPU flag"""
    emu = config.emulator
    return (emu.cpu.flags & (1 << flag_bit)) != 0


def cpu_update_flags(result, is_16bit):
    """Update flags based on result value (for arithmetic/logical ops)"""
    if is_16bit:
        mask = 0xFFFF
    else:
        mask = 0xFF
    
    result = result & mask
    
    # Zero flag
    cpu_set_flag(0, (result == 0))
    
    # Negative flag (sign bit)
    if is_16bit:
        cpu_set_flag(1, ((result & 0x8000) != 0))
    else:
        cpu_set_flag(1, ((result & 0x80) != 0))
    
    # Carry and Overflow flags would be set by specific operations
    # This is a generic helper - actual ops should set C/V appropriately

