"""
ppu.py - PPU (Picture Processing Unit) - tile/sprite rendering
Python version - maintains BASIC-like simplicity
"""

import config
import math


# Static variables for PPU registers (like STATIC in QB64)
_vram_addr = 0
_cgram_addr = 0
_oam_addr = 0


def ppu_reset():
    """Initialize PPU to default state"""
    emu = config.emulator
    
    # Clear VRAM
    config.ppu_vram[:] = [0] * len(config.ppu_vram)
    
    # Clear CGRAM (initialize to default palette)
    config.ppu_cgram[:] = [0] * len(config.ppu_cgram)
    
    # Initialize default palette (simple grayscale for now)
    # But make sure we have some visible colors
    for i in range(256):
        # Convert index to 5-bit RGB (0-31 range)
        # For better visibility, use a brighter scale
        rgb_val = min(31, (i * 31) // 255)  # Scale 0-255 to 0-31
        # Directly set CGRAM to avoid cache invalidation overhead
        rgb555 = rgb_val | (rgb_val << 5) | (rgb_val << 10)
        cgram_index = i * 2
        if cgram_index < config.PPU_CGRAM_SIZE:
            config.ppu_cgram[cgram_index] = rgb555 & 0xFF
            config.ppu_cgram[cgram_index + 1] = (rgb555 >> 8) & 0xFF
    
    # Set some bright colors for visibility
    ppu_set_palette_color(1, 31, 31, 31)  # White (for pixel value 1)
    ppu_set_palette_color(2, 31, 0, 0)    # Red
    ppu_set_palette_color(3, 0, 31, 0)    # Green
    ppu_set_palette_color(16, 0, 0, 31)   # Blue
    
    # Invalidate palette cache after reset
    ppu_invalidate_palette_cache()
    
    # Clear OAM
    for i in range(config.PPU_MAX_SPRITES):
        sprite = config.ppu_oam[i]
        sprite.x = 0
        sprite.y = 0
        sprite.tile_index = 0
        sprite.palette = 0
        sprite.priority = 0
        sprite.flip_x = False
        sprite.flip_y = False
        sprite.size = 0  # 8x8
        sprite.enabled = False
        sprite.blend_mode = 0  # Normal (opaque)
        sprite.alpha = 255  # Full opacity
    
    # Initialize background layers
    emu.ppu.bg0.scroll_x = 0
    emu.ppu.bg0.scroll_y = 0
    emu.ppu.bg0.tile_size = config.PPU_TILE_SIZE_8X8
    emu.ppu.bg0.enabled = False
    emu.ppu.bg0.tile_map_base = 0
    emu.ppu.bg0.tile_data_base = 0
    
    emu.ppu.bg1.scroll_x = 0
    emu.ppu.bg1.scroll_y = 0
    emu.ppu.bg1.tile_size = config.PPU_TILE_SIZE_8X8
    emu.ppu.bg1.enabled = False
    emu.ppu.bg1.tile_map_base = 0
    emu.ppu.bg1.tile_data_base = 0
    
    # Initialize BG2 and BG3 layers (Nitro-Core-DX: 4 background layers)
    emu.ppu.bg2.scroll_x = 0
    emu.ppu.bg2.scroll_y = 0
    emu.ppu.bg2.tile_size = config.PPU_TILE_SIZE_8X8
    emu.ppu.bg2.enabled = False
    emu.ppu.bg2.tile_map_base = 0
    emu.ppu.bg2.tile_data_base = 0
    
    emu.ppu.bg3.scroll_x = 0
    emu.ppu.bg3.scroll_y = 0
    emu.ppu.bg3.tile_size = config.PPU_TILE_SIZE_8X8
    emu.ppu.bg3.enabled = False
    emu.ppu.bg3.tile_map_base = 0
    emu.ppu.bg3.tile_data_base = 0
    
    # Clear framebuffer
    config.ppu_frame_buffer[:] = [0] * len(config.ppu_frame_buffer)
    emu.ppu.frame_buffer_enabled = False
    
    # Clear output buffer
    config.ppu_output_buffer[:] = [0] * len(config.ppu_output_buffer)
    
    # Default to landscape mode
    emu.ppu.portrait_mode = False
    
    # VBlank state
    emu.ppu.vblank_active = False
    emu.ppu.vblank_counter = 0
    
    # Initialize windowing system
    emu.ppu.window0_left = 0
    emu.ppu.window0_right = 0
    emu.ppu.window0_top = 0
    emu.ppu.window0_bottom = 0
    emu.ppu.window0_enabled = False
    emu.ppu.window1_left = 0
    emu.ppu.window1_right = 0
    emu.ppu.window1_top = 0
    emu.ppu.window1_bottom = 0
    emu.ppu.window1_enabled = False
    emu.ppu.window_logic = 0  # OR
    emu.ppu.window_main_enable = 0
    emu.ppu.window_sub_enable = 0
    
    # Initialize HDMA per-scanline scroll
    emu.ppu.hdma_enabled = False
    emu.ppu.hdma_table_base = 0
    # Clear HDMA scroll arrays (use layer scroll as default)
    for i in range(config.DISPLAY_HEIGHT):
        emu.ppu.hdma_bg0_scroll_x[i] = 0
        emu.ppu.hdma_bg0_scroll_y[i] = 0
        emu.ppu.hdma_bg1_scroll_x[i] = 0
        emu.ppu.hdma_bg1_scroll_y[i] = 0
        emu.ppu.hdma_bg2_scroll_x[i] = 0
        emu.ppu.hdma_bg2_scroll_y[i] = 0
        emu.ppu.hdma_bg3_scroll_x[i] = 0
        emu.ppu.hdma_bg3_scroll_y[i] = 0


def ppu_read_reg(reg_addr):
    """Read PPU register"""
    emu = config.emulator
    
    # TODO: Implement register reads
    # For now, return 0 for most registers
    if reg_addr == config.PPU_REG_BG0_SCROLLX:
        return emu.ppu.bg0.scroll_x & 0xFF
    elif reg_addr == config.PPU_REG_BG0_SCROLLX + 1:
        return (emu.ppu.bg0.scroll_x >> 8) & 0xFF
    elif reg_addr == config.PPU_REG_VRAM_DATA:
        # TODO: Read from VRAM at current VRAM address
        return 0
    else:
        return 0


def ppu_write_reg(reg_addr, value):
    """Write PPU register"""
    import ui
    global _vram_addr, _cgram_addr, _oam_addr
    emu = config.emulator
    
    # Log register write
    reg_names = {
        config.PPU_REG_BG0_CONTROL: "BG0_CTRL",
        config.PPU_REG_BG0_SCROLLX: "BG0_SCROLLX_L",
        config.PPU_REG_BG0_SCROLLX + 1: "BG0_SCROLLX_H",
        config.PPU_REG_BG0_SCROLLY: "BG0_SCROLLY_L",
        config.PPU_REG_BG0_SCROLLY + 1: "BG0_SCROLLY_H",
        config.PPU_REG_BG2_SCROLLX: "BG2_SCROLLX_L",
        config.PPU_REG_BG2_SCROLLX_H: "BG2_SCROLLX_H",
        config.PPU_REG_BG2_SCROLLY: "BG2_SCROLLY_L",
        config.PPU_REG_BG2_SCROLLY_H: "BG2_SCROLLY_H",
        config.PPU_REG_BG2_CONTROL: "BG2_CTRL",
        config.PPU_REG_BG3_SCROLLX: "BG3_SCROLLX_L",
        config.PPU_REG_BG3_SCROLLX_H: "BG3_SCROLLX_H",
        config.PPU_REG_BG3_SCROLLY: "BG3_SCROLLY_L",
        config.PPU_REG_BG3_SCROLLY_H: "BG3_SCROLLY_H",
        config.PPU_REG_BG3_CONTROL: "BG3_CTRL",
        config.PPU_REG_WINDOW0_LEFT: "WIN0_LEFT",
        config.PPU_REG_WINDOW0_RIGHT: "WIN0_RIGHT",
        config.PPU_REG_WINDOW0_TOP: "WIN0_TOP",
        config.PPU_REG_WINDOW0_BOTTOM: "WIN0_BOTTOM",
        config.PPU_REG_WINDOW1_LEFT: "WIN1_LEFT",
        config.PPU_REG_WINDOW1_RIGHT: "WIN1_RIGHT",
        config.PPU_REG_WINDOW1_TOP: "WIN1_TOP",
        config.PPU_REG_WINDOW1_BOTTOM: "WIN1_BOTTOM",
        config.PPU_REG_WINDOW_CONTROL: "WIN_CTRL",
        config.PPU_REG_WINDOW_MAIN_ENABLE: "WIN_MAIN",
        config.PPU_REG_WINDOW_SUB_ENABLE: "WIN_SUB",
        config.PPU_REG_VRAM_ADDR: "VRAM_ADDR_L",
        config.PPU_REG_VRAM_ADDR_H: "VRAM_ADDR_H",
        config.PPU_REG_VRAM_DATA: "VRAM_DATA",
        config.PPU_REG_CGRAM_ADDR: "CGRAM_ADDR",
        config.PPU_REG_CGRAM_DATA: "CGRAM_DATA",
    }
    reg_name = reg_names.get(reg_addr, f"REG{reg_addr:02X}")
    reg_name = reg_names.get(reg_addr, f"REG_{reg_addr:02X}")
    ui.logger.trace(f"PPU Write: {reg_name} = {value:02X}", "PPU")
    
    # TODO: Implement full register write handling
    if reg_addr == config.PPU_REG_BG0_SCROLLX:
        # Low byte of scroll X
        emu.ppu.bg0.scroll_x = (emu.ppu.bg0.scroll_x & 0xFF00) | value
    elif reg_addr == config.PPU_REG_BG0_SCROLLX + 1:
        # High byte of scroll X
        emu.ppu.bg0.scroll_x = (emu.ppu.bg0.scroll_x & 0xFF) | (value << 8)
    elif reg_addr == config.PPU_REG_BG0_SCROLLY:
        emu.ppu.bg0.scroll_y = (emu.ppu.bg0.scroll_y & 0xFF00) | value
    elif reg_addr == config.PPU_REG_BG0_SCROLLY + 1:
        emu.ppu.bg0.scroll_y = (emu.ppu.bg0.scroll_y & 0xFF) | (value << 8)
    elif reg_addr == config.PPU_REG_BG0_CONTROL:
        # Control byte: bit 0 = enable, bit 1 = tile size (0=8x8, 1=16x16)
        emu.ppu.bg0.enabled = (value & 0x01) != 0
        if (value & 0x02) != 0:
            emu.ppu.bg0.tile_size = config.PPU_TILE_SIZE_16X16
        else:
            emu.ppu.bg0.tile_size = config.PPU_TILE_SIZE_8X8
    # BG2 registers (Nitro-Core-DX: 4 background layers)
    elif reg_addr == config.PPU_REG_BG2_SCROLLX:
        emu.ppu.bg2.scroll_x = (emu.ppu.bg2.scroll_x & 0xFF00) | value
    elif reg_addr == config.PPU_REG_BG2_SCROLLX_H:
        emu.ppu.bg2.scroll_x = (emu.ppu.bg2.scroll_x & 0xFF) | (value << 8)
    elif reg_addr == config.PPU_REG_BG2_SCROLLY:
        emu.ppu.bg2.scroll_y = (emu.ppu.bg2.scroll_y & 0xFF00) | value
    elif reg_addr == config.PPU_REG_BG2_SCROLLY_H:
        emu.ppu.bg2.scroll_y = (emu.ppu.bg2.scroll_y & 0xFF) | (value << 8)
    elif reg_addr == config.PPU_REG_BG2_CONTROL:
        emu.ppu.bg2.enabled = (value & 0x01) != 0
        if (value & 0x02) != 0:
            emu.ppu.bg2.tile_size = config.PPU_TILE_SIZE_16X16
        else:
            emu.ppu.bg2.tile_size = config.PPU_TILE_SIZE_8X8
    # BG3 registers (can be used as dedicated affine layer)
    elif reg_addr == config.PPU_REG_BG3_SCROLLX:
        emu.ppu.bg3.scroll_x = (emu.ppu.bg3.scroll_x & 0xFF00) | value
    elif reg_addr == config.PPU_REG_BG3_SCROLLX_H:
        emu.ppu.bg3.scroll_x = (emu.ppu.bg3.scroll_x & 0xFF) | (value << 8)
    elif reg_addr == config.PPU_REG_BG3_SCROLLY:
        emu.ppu.bg3.scroll_y = (emu.ppu.bg3.scroll_y & 0xFF00) | value
    elif reg_addr == config.PPU_REG_BG3_SCROLLY_H:
        emu.ppu.bg3.scroll_y = (emu.ppu.bg3.scroll_y & 0xFF) | (value << 8)
    elif reg_addr == config.PPU_REG_BG3_CONTROL:
        emu.ppu.bg3.enabled = (value & 0x01) != 0
        if (value & 0x02) != 0:
            emu.ppu.bg3.tile_size = config.PPU_TILE_SIZE_16X16
        else:
            emu.ppu.bg3.tile_size = config.PPU_TILE_SIZE_8X8
    elif reg_addr == config.PPU_REG_VRAM_ADDR:
        _vram_addr = (_vram_addr & 0xFF00) | value
    elif reg_addr == config.PPU_REG_VRAM_ADDR_H:
        _vram_addr = (_vram_addr & 0xFF) | (value << 8)
    elif reg_addr == config.PPU_REG_VRAM_DATA:
        # Write to VRAM and auto-increment address
        if _vram_addr < config.PPU_VRAM_SIZE:
            old_val = config.ppu_vram[_vram_addr]
            config.ppu_vram[_vram_addr] = value
            import ui
            ui.logger.trace(f"VRAM[{_vram_addr:04X}] = {value:02X} (was {old_val:02X})", "PPU")
            _vram_addr += 1
    elif reg_addr == config.PPU_REG_CGRAM_ADDR:
        _cgram_addr = value
    elif reg_addr == config.PPU_REG_CGRAM_DATA:
        # TODO: Write 16-bit RGB555 color (needs two writes)
        # For now, just store byte
        if _cgram_addr < config.PPU_CGRAM_SIZE:
            config.ppu_cgram[_cgram_addr] = value
            _cgram_addr += 1
            # Invalidate palette cache when CGRAM is written
            ppu_invalidate_palette_cache()
    elif reg_addr == config.PPU_REG_FRAMEBUFFER_ENABLE:
        emu.ppu.frame_buffer_enabled = (value != 0)
    elif reg_addr == config.PPU_REG_DISPLAY_MODE:
        emu.ppu.portrait_mode = (value != 0)
    elif reg_addr == config.PPU_REG_MATRIX_CONTROL:
        # Matrix Mode control: bit 0 = enable, bit 1 = mirror_h, bit 2 = mirror_v
        emu.ppu.matrix_enabled = (value & 0x01) != 0
        emu.ppu.matrix_mirror_h = (value & 0x02) != 0
        emu.ppu.matrix_mirror_v = (value & 0x04) != 0
    elif reg_addr == config.PPU_REG_MATRIX_A:
        emu.ppu.matrix_a = (emu.ppu.matrix_a & 0xFF00) | value
    elif reg_addr == config.PPU_REG_MATRIX_A_H:
        emu.ppu.matrix_a = (emu.ppu.matrix_a & 0xFF) | (value << 8)
    elif reg_addr == config.PPU_REG_MATRIX_B:
        emu.ppu.matrix_b = (emu.ppu.matrix_b & 0xFF00) | value
    elif reg_addr == config.PPU_REG_MATRIX_B_H:
        emu.ppu.matrix_b = (emu.ppu.matrix_b & 0xFF) | (value << 8)
    elif reg_addr == config.PPU_REG_MATRIX_C:
        emu.ppu.matrix_c = (emu.ppu.matrix_c & 0xFF00) | value
    elif reg_addr == config.PPU_REG_MATRIX_C_H:
        emu.ppu.matrix_c = (emu.ppu.matrix_c & 0xFF) | (value << 8)
    elif reg_addr == config.PPU_REG_MATRIX_D:
        emu.ppu.matrix_d = (emu.ppu.matrix_d & 0xFF00) | value
    elif reg_addr == config.PPU_REG_MATRIX_D_H:
        emu.ppu.matrix_d = (emu.ppu.matrix_d & 0xFF) | (value << 8)
    elif reg_addr == config.PPU_REG_MATRIX_CENTER_X:
        emu.ppu.matrix_center_x = (emu.ppu.matrix_center_x & 0xFF00) | value
    elif reg_addr == config.PPU_REG_MATRIX_CENTER_X_H:
        emu.ppu.matrix_center_x = (emu.ppu.matrix_center_x & 0xFF) | (value << 8)
    elif reg_addr == config.PPU_REG_MATRIX_CENTER_Y:
        emu.ppu.matrix_center_y = (emu.ppu.matrix_center_y & 0xFF00) | value
    elif reg_addr == config.PPU_REG_MATRIX_CENTER_Y_H:
        emu.ppu.matrix_center_y = (emu.ppu.matrix_center_y & 0xFF) | (value << 8)
    # Windowing system registers
    elif reg_addr == config.PPU_REG_WINDOW0_LEFT:
        emu.ppu.window0_left = value
    elif reg_addr == config.PPU_REG_WINDOW0_RIGHT:
        emu.ppu.window0_right = value
    elif reg_addr == config.PPU_REG_WINDOW0_TOP:
        emu.ppu.window0_top = value
    elif reg_addr == config.PPU_REG_WINDOW0_BOTTOM:
        emu.ppu.window0_bottom = value
    elif reg_addr == config.PPU_REG_WINDOW1_LEFT:
        emu.ppu.window1_left = value
    elif reg_addr == config.PPU_REG_WINDOW1_RIGHT:
        emu.ppu.window1_right = value
    elif reg_addr == config.PPU_REG_WINDOW1_TOP:
        emu.ppu.window1_top = value
    elif reg_addr == config.PPU_REG_WINDOW1_BOTTOM:
        emu.ppu.window1_bottom = value
    elif reg_addr == config.PPU_REG_WINDOW_CONTROL:
        emu.ppu.window0_enabled = (value & 0x01) != 0
        emu.ppu.window1_enabled = (value & 0x02) != 0
        emu.ppu.window_logic = (value >> 2) & 0x03  # 0=OR, 1=AND, 2=XOR, 3=XNOR
    elif reg_addr == config.PPU_REG_WINDOW_MAIN_ENABLE:
        emu.ppu.window_main_enable = value  # Bit per layer: 0=BG0, 1=BG1, 2=BG2, 3=BG3, 4=sprites
    elif reg_addr == config.PPU_REG_WINDOW_SUB_ENABLE:
        emu.ppu.window_sub_enable = value  # For color math (future use)
    # HDMA (per-scanline scroll) registers
    elif reg_addr == config.PPU_REG_HDMA_CONTROL:
        emu.ppu.hdma_enabled = (value & 0x01) != 0
        # Bits 1-4: layer enable (1=BG0, 2=BG1, 4=BG2, 8=BG3) - stored in control for now
    elif reg_addr == config.PPU_REG_HDMA_TABLE_BASE_L:
        emu.ppu.hdma_table_base = (emu.ppu.hdma_table_base & 0xFF00) | value
    elif reg_addr == config.PPU_REG_HDMA_TABLE_BASE_H:
        emu.ppu.hdma_table_base = (emu.ppu.hdma_table_base & 0xFF) | (value << 8)
    elif reg_addr == config.PPU_REG_HDMA_BG0_SCROLLX_L:
        # Set scroll X for all scanlines (simplified - can be made per-scanline later)
        for i in range(config.DISPLAY_HEIGHT):
            emu.ppu.hdma_bg0_scroll_x[i] = (emu.ppu.hdma_bg0_scroll_x[i] & 0xFF00) | value
    elif reg_addr == config.PPU_REG_HDMA_BG0_SCROLLX_H:
        for i in range(config.DISPLAY_HEIGHT):
            emu.ppu.hdma_bg0_scroll_x[i] = (emu.ppu.hdma_bg0_scroll_x[i] & 0xFF) | (value << 8)
    elif reg_addr == config.PPU_REG_HDMA_BG0_SCROLLY_L:
        for i in range(config.DISPLAY_HEIGHT):
            emu.ppu.hdma_bg0_scroll_y[i] = (emu.ppu.hdma_bg0_scroll_y[i] & 0xFF00) | value
    elif reg_addr == config.PPU_REG_HDMA_BG0_SCROLLY_H:
        for i in range(config.DISPLAY_HEIGHT):
            emu.ppu.hdma_bg0_scroll_y[i] = (emu.ppu.hdma_bg0_scroll_y[i] & 0xFF) | (value << 8)
    # Unknown register - ignore


def ppu_check_window(x, y, layer_index):
    """
    Check if a pixel at (x, y) should be drawn based on window settings for the given layer.
    Returns True if pixel should be drawn (inside window), False if clipped.
    
    layer_index: 0=BG0, 1=BG1, 2=BG2, 3=BG3, 4=sprites
    """
    emu = config.emulator
    
    # Check if windowing is enabled for this layer
    if (emu.ppu.window_main_enable & (1 << layer_index)) == 0:
        # Windowing disabled for this layer - always draw
        return True
    
    # Check if any windows are enabled
    if not emu.ppu.window0_enabled and not emu.ppu.window1_enabled:
        # No windows enabled - always draw
        return True
    
    # Check if pixel is inside Window 0
    in_window0 = False
    if emu.ppu.window0_enabled:
        in_window0 = (emu.ppu.window0_left <= x <= emu.ppu.window0_right and
                     emu.ppu.window0_top <= y <= emu.ppu.window0_bottom)
    
    # Check if pixel is inside Window 1
    in_window1 = False
    if emu.ppu.window1_enabled:
        in_window1 = (emu.ppu.window1_left <= x <= emu.ppu.window1_right and
                     emu.ppu.window1_top <= y <= emu.ppu.window1_bottom)
    
    # Apply window logic
    if not emu.ppu.window0_enabled:
        # Only Window 1
        return in_window1
    elif not emu.ppu.window1_enabled:
        # Only Window 0
        return in_window0
    else:
        # Both windows enabled - apply logic
        if emu.ppu.window_logic == 0:  # OR
            return in_window0 or in_window1
        elif emu.ppu.window_logic == 1:  # AND
            return in_window0 and in_window1
        elif emu.ppu.window_logic == 2:  # XOR
            return in_window0 != in_window1
        else:  # XNOR (3)
            return in_window0 == in_window1


def ppu_render_frame():
    """Render a complete frame"""
    emu = config.emulator
    import ui
    
    # Log that rendering is happening (only if detailed logging enabled)
    if ui.logger.enabled and ui.logger.detailed_logging:
        ui.logger.trace(f"PPU Render: Starting frame render, BG0.enabled={emu.ppu.bg0.enabled}, BG1.enabled={emu.ppu.bg1.enabled}", "PPU")
    
    # Ensure palette cache is built (do this once per frame)
    global _palette_cache, _palette_cache_dirty
    if _palette_cache_dirty:
        ppu_get_palette_color(0)  # Rebuild cache
    
    # Clear output buffer to background color (palette entry 0)
    # Cache the background color to avoid repeated palette lookups
    bg_color = _palette_cache[0]
    config.ppu_output_buffer[:] = [bg_color] * len(config.ppu_output_buffer)
    
    # Render background layers in priority order (BG3 -> BG2 -> BG1 -> BG0)
    # Lower layers render first, higher layers on top
    # BG3 can be used as dedicated affine layer (Matrix Mode)
    
    # BG3 (highest priority, can be affine layer)
    if emu.ppu.bg3.enabled:
        if ui.logger.enabled and ui.logger.detailed_logging:
            ui.logger.trace(f"PPU Render: Rendering BG3", "PPU")
        ppu_render_tile_layer(emu.ppu.bg3)
    
    # BG2
    if emu.ppu.bg2.enabled:
        if ui.logger.enabled and ui.logger.detailed_logging:
            ui.logger.trace(f"PPU Render: Rendering BG2", "PPU")
        ppu_render_tile_layer(emu.ppu.bg2)
    
    # BG1
    if emu.ppu.bg1.enabled:
        if ui.logger.enabled and ui.logger.detailed_logging:
            ui.logger.trace(f"PPU Render: Rendering BG1", "PPU")
        ppu_render_tile_layer(emu.ppu.bg1)
    
    # BG0 (lowest priority, can use Matrix Mode)
    if emu.ppu.bg0.enabled:
        if emu.ppu.matrix_enabled:
            # Render BG0 with Matrix Mode transformation
            if ui.logger.enabled and ui.logger.detailed_logging:
                ui.logger.trace(f"PPU Render: BG0 with Matrix Mode enabled", "PPU")
            ppu_render_matrix()
        else:
            # Render BG0 normally
            if ui.logger.enabled and ui.logger.detailed_logging:
                ui.logger.trace(f"PPU Render: BG0 enabled, scroll=({emu.ppu.bg0.scroll_x}, {emu.ppu.bg0.scroll_y}), tile_data_base={emu.ppu.bg0.tile_data_base}, tile_map_base={emu.ppu.bg0.tile_map_base}", "PPU")
            ppu_render_tile_layer(emu.ppu.bg0)
    elif ui.logger.enabled and ui.logger.detailed_logging:
        ui.logger.trace(f"PPU Render: BG0 is NOT enabled! BG0_CTRL was written but enabled=False", "PPU")
    
    # Render sprites
    ppu_render_sprites()
    
    # Composite framebuffer layer (if enabled)
    if emu.ppu.frame_buffer_enabled:
        ppu_composite_frame_buffer()
    
    # Apply rotation if in portrait mode
    if emu.ppu.portrait_mode:
        ppu_rotate_output()
    
    # Trigger VBlank
    emu.ppu.vblank_active = True
    import cpu
    cpu.cpu_trigger_vblank()


def ppu_render_tile_layer(layer):
    """Render a tile layer (internal helper)"""
    emu = config.emulator
    
    scroll_x = layer.scroll_x
    scroll_y = layer.scroll_y
    tile_size = layer.tile_size
    
    # Tilemap is 32x32 tiles (1024 entries)
    # Each tilemap entry is 16 bits: [15:10] = tile index, [9:6] = palette, [5] = flip_x, [4] = flip_y, [3:0] = priority
    TILEMAP_WIDTH = 32
    TILEMAP_HEIGHT = 32
    
    # Calculate visible tile range
    # Account for scroll offset - tiles can be partially visible
    # Start from tile that might be partially visible above/left of screen
    start_tile_x = scroll_x // tile_size
    start_tile_y = scroll_y // tile_size
    # End at tile that might be partially visible below/right of screen
    end_tile_x = start_tile_x + (config.DISPLAY_WIDTH // tile_size) + 2  # +2 for partial tiles
    end_tile_y = start_tile_y + (config.DISPLAY_HEIGHT // tile_size) + 2
    
    # The tilemap wraps, so we need to be careful not to check the same tile twice
    # Limit the range to at most one full tilemap wrap to prevent duplicates
    # Calculate how many tiles we need to cover the screen
    tiles_needed_x = (config.DISPLAY_WIDTH // tile_size) + 2
    tiles_needed_y = (config.DISPLAY_HEIGHT // tile_size) + 2
    
    # Clamp the range to prevent checking tiles that wrap to the same position
    # We only need to check enough tiles to cover the screen, not multiple wraps
    actual_start_tile_y = start_tile_y
    actual_start_tile_x = start_tile_x
    actual_end_tile_y = start_tile_y + tiles_needed_y
    actual_end_tile_x = start_tile_x + tiles_needed_x
    
    # Debug: Log tile range calculation (only if detailed logging enabled)
    import ui
    if ui.logger.enabled and ui.logger.detailed_logging:
        ui.logger.trace(f"PPU Render: Tile range - scroll=({scroll_x}, {scroll_y}), tile_size={tile_size}, start_tile=({actual_start_tile_x}, {actual_start_tile_y}), end_tile=({actual_end_tile_x}, {actual_end_tile_y}), tiles_needed=({tiles_needed_x}, {tiles_needed_y})", "PPU")
    
    # Track which tilemap positions we've rendered to prevent duplicates
    # Use a set to track (map_tile_x, map_tile_y) pairs we've already rendered
    rendered_tiles = set()
    
    # Render each visible tile
    for tile_y in range(actual_start_tile_y, actual_end_tile_y):
        for tile_x in range(actual_start_tile_x, actual_end_tile_x):
            # Wrap tile coordinates (tilemap wraps)
            map_tile_x = tile_x % TILEMAP_WIDTH
            map_tile_y = tile_y % TILEMAP_HEIGHT
            
            # Check if we've already rendered this tilemap position in this frame
            # This prevents the same tile from appearing twice when scroll wraps
            tile_key = (map_tile_x, map_tile_y)
            if tile_key in rendered_tiles:
                continue  # Skip - already rendered this tilemap position
            rendered_tiles.add(tile_key)
            
            # Read tilemap entry
            tilemap_index = map_tile_y * TILEMAP_WIDTH + map_tile_x
            tilemap_addr = layer.tile_map_base + (tilemap_index * 2)  # 2 bytes per entry
            
            if tilemap_addr + 1 >= config.PPU_VRAM_SIZE:
                continue
            
            # Read tilemap entry (16-bit, little-endian)
            # ROM writes: low byte = tile index, high byte = attributes
            # Memory layout: VRAM[addr] = low byte, VRAM[addr+1] = high byte
            # So if ROM writes 0x01 then 0x00, memory is: [0x01, 0x00] = 0x0001
            # But we read it as: low = VRAM[addr], high = VRAM[addr+1]
            # So 0x0001 means: low=0x01, high=0x00
            tilemap_entry_low = config.ppu_vram[tilemap_addr]      # First byte = tile index
            tilemap_entry_high = config.ppu_vram[tilemap_addr + 1] # Second byte = attributes
            tilemap_entry = tilemap_entry_low | (tilemap_entry_high << 8)
            
            # Debug: Log tilemap read for first few tiles (only if detailed logging enabled)
            if ui.logger.enabled and ui.logger.detailed_logging and tile_x < start_tile_x + 3 and tile_y < start_tile_y + 3:
                import ui
                ui.logger.trace(f"PPU Render: Reading tilemap - tile=({tile_x}, {tile_y}), map_tile=({map_tile_x}, {map_tile_y}), tilemap_index={tilemap_index}, tilemap_addr={tilemap_addr:04X}, entry=0x{tilemap_entry_low:02X}{tilemap_entry_high:02X}", "PPU")
            
            # Decode tilemap entry format (as written by ROM):
            # Low byte [7:0] = tile index (0-255)
            # High byte [7:4] = palette index (0-15)
            # High byte [3] = flip_x
            # High byte [2] = flip_y
            # High byte [1:0] = priority (unused for now)
            tile_index = tilemap_entry_low  # First byte = tile index
            
            # Skip empty tiles (tile index 0 = no tile)
            if tile_index == 0:
                continue
            
            # Debug: Log first tile found (only if detailed logging enabled)
            if ui.logger.enabled and ui.logger.detailed_logging and tile_x == start_tile_x and tile_y == start_tile_y:
                import ui
                ui.logger.trace(f"PPU Render: Found tile at ({tile_x}, {tile_y}), tile_index={tile_index}, tilemap_addr={tilemap_addr:04X}, entry=0x{tilemap_entry_low:02X}{tilemap_entry_high:02X}", "PPU")
            
            palette_index = (tilemap_entry_high >> 4) & 0xF  # Upper 4 bits of high byte = palette
            flip_x = (tilemap_entry_high & 0x08) != 0  # Bit 3 of high byte = flip_x
            flip_y = (tilemap_entry_high & 0x04) != 0  # Bit 2 of high byte = flip_y
            
            # Calculate tile pixel position
            tile_pixel_x = tile_x * tile_size - scroll_x
            tile_pixel_y = tile_y * tile_size - scroll_y
            
            # Pre-calculate tile data base address (optimization)
            # For 4bpp (4 bits per pixel): 8 pixels per row = 4 bytes per row
            bytes_per_tile_row = tile_size // 2  # 4bpp: 2 pixels per byte, so 8 pixels = 4 bytes
            bytes_per_tile = tile_size * bytes_per_tile_row  # 8 rows * 4 bytes = 32 bytes per tile
            tile_data_base_addr = layer.tile_data_base + (tile_index * bytes_per_tile)
            
            # Pre-calculate palette base (optimization)
            palette_base = palette_index * 16
            
            # Get output buffer and VRAM references (optimization)
            output_buffer = config.ppu_output_buffer
            vram = config.ppu_vram
            output_width = config.DISPLAY_WIDTH
            
            # Calculate valid screen bounds for this tile (optimization)
            min_screen_y = max(0, tile_pixel_y)
            max_screen_y = min(config.DISPLAY_HEIGHT, tile_pixel_y + tile_size)
            min_screen_x = max(0, tile_pixel_x)
            max_screen_x = min(config.DISPLAY_WIDTH, tile_pixel_x + tile_size)
            
            # Determine layer index once (optimization - moved outside inner loop)
            # BG0=0, BG1=1, BG2=2, BG3=3
            layer_index = 0  # Default to BG0
            if layer == emu.ppu.bg1:
                layer_index = 1
            elif layer == emu.ppu.bg2:
                layer_index = 2
            elif layer == emu.ppu.bg3:
                layer_index = 3
            
            # Cache windowing state check (optimization - check if windowing is enabled for this layer)
            # If no windows are enabled for this layer, skip window checks entirely
            window_enable_bits = layer.window_enable
            windowing_enabled = (window_enable_bits & 0x03) != 0  # Check if any window is enabled
            
            # Render tile pixels (optimized inner loop)
            for py in range(tile_size):
                screen_y = tile_pixel_y + py
                if screen_y < min_screen_y or screen_y >= max_screen_y:
                    continue
                
                tile_y_local = tile_size - 1 - py if flip_y else py
                tile_row_base = tile_data_base_addr + (tile_y_local * bytes_per_tile_row)
                
                for px in range(tile_size):
                    screen_x = tile_pixel_x + px
                    if screen_x < min_screen_x or screen_x >= max_screen_x:
                        continue
                    
                    tile_x_local = tile_size - 1 - px if flip_x else px
                    
                    # Calculate tile data address (optimized)
                    tile_data_addr = tile_row_base + (tile_x_local // 2)
                    
                    # Bounds check (only one check needed)
                    if tile_data_addr >= len(vram):
                        continue
                    
                    # Read pixel byte
                    pixel_byte = vram[tile_data_addr]
                    
                    # Extract pixel (optimized - avoid modulo, use bitwise ops)
                    if tile_x_local & 1:  # Odd pixel (lower 4 bits)
                        pixel_value = pixel_byte & 0xF
                    else:  # Even pixel (upper 4 bits)
                        pixel_value = (pixel_byte >> 4) & 0xF
                    
                    # Skip transparent pixels
                    if pixel_value == 0:
                        continue
                    
                    # Check windowing only if enabled (optimization - skip function call when disabled)
                    if windowing_enabled:
                        if not ppu_check_window(screen_x, screen_y, layer_index):
                            continue  # Pixel is outside window - skip
                    
                    # Calculate final palette index and write (optimized)
                    # Direct palette cache access (cache is built at start of frame)
                    final_palette_index = palette_base + pixel_value
                    # Bounds check for palette index (optimization - avoid if possible)
                    if final_palette_index >= 256:
                        continue  # Skip invalid palette index
                    
                    output_index = screen_y * output_width + screen_x
                    output_buffer[output_index] = _palette_cache[final_palette_index]


def ppu_blend_colors(bg_color, sprite_color, blend_mode, alpha):
    """
    Blend two RGB555 colors based on blend mode
    Returns blended color as RGB555
    """
    # Extract RGB components from RGB555
    def rgb555_to_rgb(color):
        r = (color >> 10) & 0x1F
        g = (color >> 5) & 0x1F
        b = color & 0x1F
        return r, g, b
    
    def rgb_to_rgb555(r, g, b):
        return ((r & 0x1F) << 10) | ((g & 0x1F) << 5) | (b & 0x1F)
    
    bg_r, bg_g, bg_b = rgb555_to_rgb(bg_color)
    spr_r, spr_g, spr_b = rgb555_to_rgb(sprite_color)
    
    if blend_mode == 0:  # Normal (opaque)
        return sprite_color
    elif blend_mode == 1:  # Alpha blend
        # Alpha blend: result = sprite * alpha + bg * (1 - alpha)
        alpha_norm = alpha / 255.0
        r = int(spr_r * alpha_norm + bg_r * (1 - alpha_norm))
        g = int(spr_g * alpha_norm + bg_g * (1 - alpha_norm))
        b = int(spr_b * alpha_norm + bg_b * (1 - alpha_norm))
        return rgb_to_rgb555(r, g, b)
    elif blend_mode == 2:  # Additive
        r = min(31, bg_r + spr_r)
        g = min(31, bg_g + spr_g)
        b = min(31, bg_b + spr_b)
        return rgb_to_rgb555(r, g, b)
    elif blend_mode == 3:  # Subtractive
        r = max(0, bg_r - spr_r)
        g = max(0, bg_g - spr_g)
        b = max(0, bg_b - spr_b)
        return rgb_to_rgb555(r, g, b)
    else:
        return sprite_color


def ppu_render_sprites():
    """Render sprites with priority and blending support"""
    emu = config.emulator
    output_buffer = config.ppu_output_buffer
    output_width = config.DISPLAY_WIDTH
    
    # Sort sprites by priority (3 = highest, 0 = lowest), then by index (lower index = higher priority if same priority)
    sprite_list = []
    for i in range(config.PPU_MAX_SPRITES):
        sprite = config.ppu_oam[i]
        if sprite.enabled:
            sprite_list.append((i, sprite))
    
    # Sort by priority (descending), then by index (ascending for tie-breaking)
    sprite_list.sort(key=lambda x: (x[1].priority, -x[0]), reverse=True)
    
    # Render sprites in priority order
    for sprite_idx, sprite in sprite_list:
        
        # Calculate sprite size
        sprite_size = 8 if sprite.size == 0 else 16
        
        # Render sprite tile
        for py in range(sprite_size):
            screen_y = sprite.y + py
            if screen_y < 0 or screen_y >= config.DISPLAY_HEIGHT:
                continue
            
            tile_y_local = py
            if sprite.flip_y:
                tile_y_local = sprite_size - 1 - tile_y_local
            
            for px in range(sprite_size):
                screen_x = sprite.x + px
                if screen_x < 0 or screen_x >= config.DISPLAY_WIDTH:
                    continue
                
                tile_x_local = px
                if sprite.flip_x:
                    tile_x_local = sprite_size - 1 - tile_x_local
                
                # Read pixel from tile data
                # Use BG0 tile data base for sprites (could be configurable)
                tile_data_base = emu.ppu.bg0.tile_data_base
                bytes_per_tile_row = sprite_size // 2  # 4 pixels per byte (4bpp)
                tile_data_addr = tile_data_base + (sprite.tile_index * sprite_size * bytes_per_tile_row)
                tile_data_addr += tile_y_local * bytes_per_tile_row
                tile_data_addr += tile_x_local // 4
                
                if tile_data_addr >= config.PPU_VRAM_SIZE:
                    continue
                
                # Read byte containing pixel
                pixel_byte = config.ppu_vram[tile_data_addr]
                
                # Extract pixel (2 bits per pixel, 4 pixels per byte)
                pixel_in_byte = tile_x_local % 4
                pixel_value = (pixel_byte >> (pixel_in_byte * 2)) & 0x3
                
                # Skip transparent pixels (palette index 0)
                if pixel_value == 0:
                    continue
                
                # Check windowing for sprites (layer_index = 4 for sprites)
                if not ppu_check_window(screen_x, screen_y, 4):
                    continue  # Pixel is outside window - skip
                
                # Calculate final palette index (sprites use palette 0-15, each with 16 colors)
                final_palette_index = sprite.palette * 16 + pixel_value
                sprite_color = ppu_get_palette_color(final_palette_index)
                
                # Get background color at this position
                output_index = screen_y * output_width + screen_x
                bg_color = output_buffer[output_index]
                
                # Apply blending if enabled
                if sprite.blend_mode > 0:
                    final_color = ppu_blend_colors(bg_color, sprite_color, sprite.blend_mode, sprite.alpha)
                else:
                    final_color = sprite_color
                
                # Write to output buffer
                output_buffer[output_index] = final_color


def ppu_render_matrix():
    """Render background with Matrix Mode transformation (90's retro-futuristic perspective/rotation effects)"""
    emu = config.emulator
    layer = emu.ppu.bg0
    vram = config.ppu_vram
    output_width = config.DISPLAY_WIDTH
    output_height = config.DISPLAY_HEIGHT
    
    # Matrix Mode uses a 128x128 tilemap
    TILEMAP_SIZE = 128
    TILE_SIZE = 8
    
    # Get transformation matrix (8.8 fixed point)
    # Convert from 16-bit signed to float
    def fixed_to_float(fixed):
        # Interpret as signed 16-bit, then divide by 256
        if fixed >= 0x8000:
            fixed = fixed - 0x10000
        return fixed / 256.0
    
    a = fixed_to_float(emu.ppu.matrix_a)
    b = fixed_to_float(emu.ppu.matrix_b)
    c = fixed_to_float(emu.ppu.matrix_c)
    d = fixed_to_float(emu.ppu.matrix_d)
    cx = emu.ppu.matrix_center_x
    cy = emu.ppu.matrix_center_y
    
    # For each screen pixel, calculate the source tilemap coordinate
    for screen_y in range(output_height):
        for screen_x in range(output_width):
            # Apply inverse transformation
            # Screen coordinates relative to center
            dx = screen_x - cx
            dy = screen_y - cy
            
            # Apply inverse matrix: [x'] = [a b]^-1 [dx]
            #                      [y']   [c d]    [dy]
            # Inverse of 2x2 matrix: 1/(ad-bc) * [d -b]
            #                                    [-c a]
            det = a * d - b * c
            if abs(det) < 0.001:  # Avoid division by zero
                continue
            
            inv_det = 1.0 / det
            map_x = (d * dx - b * dy) * inv_det
            map_y = (-c * dx + a * dy) * inv_det
            
            # Add scroll offset
            map_x += layer.scroll_x
            map_y += layer.scroll_y
            
            # Apply mirroring
            if emu.ppu.matrix_mirror_h:
                map_x = TILEMAP_SIZE * TILE_SIZE - map_x
            if emu.ppu.matrix_mirror_v:
                map_y = TILEMAP_SIZE * TILE_SIZE - map_y
            
            # Convert to tile coordinates
            tile_x = int(map_x // TILE_SIZE) % TILEMAP_SIZE
            tile_y = int(map_y // TILE_SIZE) % TILEMAP_SIZE
            
            # Get pixel within tile
            pixel_x = int(map_x) % TILE_SIZE
            pixel_y = int(map_y) % TILE_SIZE
            
            # Read tilemap entry
            tilemap_index = tile_y * TILEMAP_SIZE + tile_x
            tilemap_addr = layer.tile_map_base + (tilemap_index * 2)
            
            if tilemap_addr + 1 >= len(vram):
                continue
            
            tile_index = vram[tilemap_addr]
            tile_attrs = vram[tilemap_addr + 1]
            palette = (tile_attrs >> 4) & 0xF
            flip_x = (tile_attrs & 0x20) != 0
            flip_y = (tile_attrs & 0x40) != 0
            
            # Apply tile flipping
            if flip_x:
                pixel_x = TILE_SIZE - 1 - pixel_x
            if flip_y:
                pixel_y = TILE_SIZE - 1 - pixel_y
            
            # Read pixel from tile data
            bytes_per_tile = TILE_SIZE * (TILE_SIZE // 2)  # 4bpp: 2 pixels per byte
            tile_data_addr = layer.tile_data_base + (tile_index * bytes_per_tile)
            tile_data_addr += pixel_y * (TILE_SIZE // 2)  # Bytes per row
            tile_data_addr += pixel_x // 2  # Byte containing pixel
            
            if tile_data_addr >= len(vram):
                continue
            
            pixel_byte = vram[tile_data_addr]
            if pixel_x & 1:  # Odd pixel (lower 4 bits)
                pixel_value = pixel_byte & 0xF
            else:  # Even pixel (upper 4 bits)
                pixel_value = (pixel_byte >> 4) & 0xF
            
            # Skip transparent pixels
            if pixel_value == 0:
                continue
            
            # Calculate final palette index
            palette_base = palette * 16
            final_palette_index = palette_base + pixel_value
            
            # Write to output buffer
            output_index = screen_y * output_width + screen_x
            if final_palette_index < 256:
                config.ppu_output_buffer[output_index] = ppu_get_palette_color(final_palette_index)


def ppu_composite_frame_buffer():
    """Composite framebuffer layer (internal helper)"""
    # TODO: Implement framebuffer compositing
    # For now, just copy framebuffer pixels to output (overwrite)
    for i in range(config.DISPLAY_WIDTH * config.DISPLAY_HEIGHT):
        palette_index = config.ppu_frame_buffer[i]
        if palette_index != 0:  # 0 = transparent
            config.ppu_output_buffer[i] = ppu_get_palette_color(palette_index)


def ppu_rotate_output():
    """Rotate output for portrait mode (internal helper)"""
    # Rotate 90 degrees clockwise: (x, y) -> (DISPLAY_HEIGHT - 1 - y, x)
    rotated_buffer = [0] * (config.DISPLAY_WIDTH_PORTRAIT * config.DISPLAY_HEIGHT_PORTRAIT)
    
    for y in range(config.DISPLAY_HEIGHT):
        for x in range(config.DISPLAY_WIDTH):
            src_index = y * config.DISPLAY_WIDTH + x
            dst_index = x * config.DISPLAY_HEIGHT_PORTRAIT + (config.DISPLAY_HEIGHT - 1 - y)
            if 0 <= dst_index < len(rotated_buffer):
                rotated_buffer[dst_index] = config.ppu_output_buffer[src_index]
    
    # Copy rotated buffer back (would need to swap width/height for actual display)
    # For now, this is a placeholder - actual rotation would be handled in main display code


def ppu_set_palette_color(index, r, g, b):
    """Set palette color (RGB555 format)"""
    # Clamp values
    r = r & 0x1F  # 5 bits
    g = g & 0x1F
    b = b & 0x1F
    
    # Pack into RGB555: bit 15=0, bits 14-10=B, 9-5=G, 4-0=R
    rgb555 = r | (g << 5) | (b << 10)
    
    # Store in CGRAM (2 bytes per color)
    cgram_index = index * 2
    if cgram_index < config.PPU_CGRAM_SIZE:
        config.ppu_cgram[cgram_index] = rgb555 & 0xFF
        config.ppu_cgram[cgram_index + 1] = (rgb555 >> 8) & 0xFF
    
    # Invalidate palette cache
    ppu_invalidate_palette_cache()


# Cache for palette colors (performance optimization)
_palette_cache = [0] * 256
_palette_cache_dirty = True

def ppu_invalidate_palette_cache():
    """Mark palette cache as dirty (call when palette changes)"""
    global _palette_cache_dirty
    _palette_cache_dirty = True

def ppu_get_palette_color(index):
    """Get palette color as RGB (for display) - cached for performance"""
    global _palette_cache, _palette_cache_dirty
    
    # Rebuild cache if dirty
    if _palette_cache_dirty:
        cgram = config.ppu_cgram
        for i in range(256):
            cgram_index = i * 2
            if cgram_index >= config.PPU_CGRAM_SIZE:
                _palette_cache[i] = 0  # Black if out of bounds
                continue
            
            # Read RGB555 value (optimized - direct array access)
            rgb555 = cgram[cgram_index] | (cgram[cgram_index + 1] << 8)
            
            # Extract and convert components (optimized)
            r = (rgb555 & 0x1F) * 255 // 31
            g = ((rgb555 >> 5) & 0x1F) * 255 // 31
            b = ((rgb555 >> 10) & 0x1F) * 255 // 31
            
            # Pack into 24-bit RGB (for pygame)
            _palette_cache[i] = (r << 16) | (g << 8) | b
        
        _palette_cache_dirty = False
    
    # Return cached value (optimized - direct array access, no bounds check needed if index is valid)
    return _palette_cache[index] if 0 <= index < 256 else 0

