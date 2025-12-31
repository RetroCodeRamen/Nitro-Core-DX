"""
input.py - Input controller handling (SNES-like 10-button controller)
Python version - maintains BASIC-like simplicity
"""

import config

# Pygame will be imported in main.py, we'll use a placeholder here
_pygame_available = False
_pygame_keys = None

def _init_pygame():
    """Initialize pygame if available"""
    global _pygame_available, _pygame_keys
    try:
        import pygame
        _pygame_available = True
        _pygame_keys = pygame.key
        return True
    except ImportError:
        return False


def input_reset():
    """Initialize input system"""
    emu = config.emulator
    
    # Clear controller state
    emu.input.controller1 = 0
    emu.input.controller1_latch = 0
    emu.input.controller1_shift = 0
    
    # Initialize keyboard mapping (default mappings)
    # TODO: Load from config file or allow customization
    config.input_keyboard_map[:] = [0] * 256
    
    # Default keyboard mappings will be set in main.py when pygame is available


def input_update():
    """Update input state (read keyboard and update controller state)"""
    emu = config.emulator
    buttons = 0
    
    if not _pygame_available:
        # Pygame not available - can't read input
        emu.input.controller1 = buttons
        return
    
    # TODO: Read keyboard state using pygame
    # This will be implemented in main.py where pygame is available
    # For now, just update the state
    emu.input.controller1 = buttons
    
    # If latch is active, update latched state
    if emu.input.controller1_latch != 0:
        emu.input.controller1_latch = buttons
        emu.input.controller1_shift = 0  # Reset shift position


def input_read_reg(reg_addr):
    """Read input register"""
    emu = config.emulator
    
    if reg_addr == config.INPUT_REG_CONTROLLER1:
        # SNES-style shift register read
        # Each read returns one button state and shifts to next
        if emu.input.controller1_shift < 12:
            # Read button at current shift position
            button = (emu.input.controller1_latch >> emu.input.controller1_shift) & 1
            emu.input.controller1_shift += 1
            return button
        else:
            # After all buttons read, return 1 (SNES behavior)
            return 1
    
    return 0


def input_write_reg(reg_addr, value):
    """Write input register"""
    emu = config.emulator
    
    if reg_addr == config.INPUT_REG_CONTROLLER1_LATCH:
        # Latch controller state
        if value != 0:
            emu.input.controller1_latch = emu.input.controller1
            emu.input.controller1_shift = 0
        # Release latch (shift register mode)
        # Latch stays latched until next latch command


def input_get_button_state(button_mask):
    """Get current button state (for direct access, not via memory-mapped I/O)"""
    emu = config.emulator
    return (emu.input.controller1 & button_mask) != 0


def input_update_from_pygame(pygame_keys):
    """Update input from pygame key state (called from main.py)"""
    import pygame
    emu = config.emulator
    buttons = 0
    
    # Map pygame keys to buttons
    # Arrow keys for D-pad
    if pygame_keys[pygame.K_UP]:
        buttons |= config.INPUT_BUTTON_UP
    if pygame_keys[pygame.K_DOWN]:
        buttons |= config.INPUT_BUTTON_DOWN
    if pygame_keys[pygame.K_LEFT]:
        buttons |= config.INPUT_BUTTON_LEFT
    if pygame_keys[pygame.K_RIGHT]:
        buttons |= config.INPUT_BUTTON_RIGHT
    
    # Z/X for A/B
    if pygame_keys[pygame.K_z]:
        buttons |= config.INPUT_BUTTON_A
    if pygame_keys[pygame.K_x]:
        buttons |= config.INPUT_BUTTON_B
    
    # A/S for X/Y
    if pygame_keys[pygame.K_a]:
        buttons |= config.INPUT_BUTTON_X
    if pygame_keys[pygame.K_s]:
        buttons |= config.INPUT_BUTTON_Y
    
    # Q/E for L/R
    if pygame_keys[pygame.K_q]:
        buttons |= config.INPUT_BUTTON_L
    if pygame_keys[pygame.K_e]:
        buttons |= config.INPUT_BUTTON_R
    
    # Enter/Space for Start/Select
    if pygame_keys[pygame.K_RETURN]:
        buttons |= config.INPUT_BUTTON_START
    if pygame_keys[pygame.K_SPACE]:
        buttons |= config.INPUT_BUTTON_SELECT
    
    emu.input.controller1 = buttons
    
    # If latch is active, update latched state
    if emu.input.controller1_latch != 0:
        emu.input.controller1_latch = buttons
        emu.input.controller1_shift = 0

