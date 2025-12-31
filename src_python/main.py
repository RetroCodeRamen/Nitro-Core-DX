"""
main.py - Fantasy Console Emulator Entry Point
Python version with Pygame for graphics and UI
"""

import sys
import os
import time
import argparse

# Add src_python to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__)))

import config
import cpu
import memory
import rom
import ppu
import apu
import input as input_module
import debug
import ui

# Try to import pygame
try:
    import pygame
    PYGAME_AVAILABLE = True
except ImportError:
    PYGAME_AVAILABLE = False
    print("WARNING: Pygame not installed. Graphics will not work.")
    print("Install with: pip install pygame")

# Try to import numpy for rendering
try:
    import numpy as np
    NUMPY_AVAILABLE = True
except ImportError:
    NUMPY_AVAILABLE = False


def load_rom_file(filename):
    """Load a ROM file and return success status"""
    ui.logger.info(f"Loading ROM: {filename}")
    success = rom.rom_load_file(filename)
    if success:
        ui.logger.info("ROM loaded successfully!")
        # Reset CPU after ROM load to ensure clean state
        cpu.cpu_reset()
        debug.debug_print_rom_info()
        return True
    else:
        ui.logger.error("Failed to load ROM file!")
        return False


def reset_emulator():
    """Reset all emulator subsystems"""
    ui.logger.info("Resetting emulator subsystems...")
    memory.memory_reset()
    cpu.cpu_reset()
    ppu.ppu_reset()
    apu.apu_reset()
    input_module.input_reset()
    debug.debug_reset()
    ui.logger.info("Emulator reset complete")


def start_emulation():
    """Start emulation"""
    emu = config.emulator
    if emu.paused:
        emu.paused = False
        ui.logger.info("Emulation started")
    else:
        ui.logger.info("Emulation already running")


def stop_emulation():
    """Stop emulation"""
    emu = config.emulator
    emu.paused = True
    ui.logger.info("Emulation stopped")


def step_emulation():
    """Step one instruction"""
    emu = config.emulator
    if emu.paused:
        # Run one instruction
        instruction = cpu.cpu_fetch_instruction()
        cpu.cpu_execute_instruction(instruction)
        ui.logger.debug(f"Stepped: PC={emu.cpu.pc_bank:02X}:{emu.cpu.pc_offset:04X}")
    else:
        ui.logger.warning("Cannot step while emulation is running")


def main():
    """Main program entry point"""
    # Parse command line arguments
    parser = argparse.ArgumentParser(description='Fantasy Console Emulator')
    parser.add_argument('romfile', nargs='?', help='ROM file to load')
    args = parser.parse_args()
    
    rom_file = args.romfile if args.romfile else ""
    
    # Initialize logging
    ui.logger.info("=" * 60)
    ui.logger.info("Fantasy Console Emulator - Starting...")
    ui.logger.info("=" * 60)
    
    # Initialize emulator
    reset_emulator()
    
    emu = config.emulator
    
    # Calculate frame time (target 60 FPS)
    emu.frame_time = 1.0 / config.TARGET_FPS
    emu.frame_count = 0
    emu.last_frame_time = time.time()
    
    # Load ROM if specified
    rom_loaded = False
    current_rom_file = rom_file  # Store for reload
    if rom_file:
        rom_loaded = load_rom_file(rom_file)
    else:
        ui.logger.info("No ROM file specified. Use File > Open ROM to load one.")
    
    # Initialize display
    if not PYGAME_AVAILABLE:
        ui.logger.error("Pygame not available. Cannot run emulator.")
        print("Install with: pip install pygame")
        return
    
    ui.logger.info("Initializing Pygame...")
    pygame.init()
    
    # UI setup
    ui.logger.info("Setting up UI...")
    base_width = config.DISPLAY_WIDTH
    base_height = config.DISPLAY_HEIGHT
    
    # Create menu bar
    menu_bar = ui.MenuBar(base_width)
    menu_bar.add_menu("File", ["Open ROM...", "Exit"])
    menu_bar.add_menu("Emulation", ["Start", "Stop", "Reset", "Pause/Resume", "Step"])
    menu_bar.add_menu("View", ["Settings", "Hex Debugger"])
    menu_bar.add_menu("Tools", ["CPU State", "Memory Info", "PPU State", "APU State"])
    
    # Create windows
    settings_window = ui.SettingsWindow(50, 50)
    hex_debugger = ui.HexDebuggerWindow(100, 100)
    
    # Make hex_debugger accessible to cpu.py for state capture
    import cpu
    # Store reference in ui module so cpu.py can access it
    ui.hex_debugger_instance = hex_debugger
    
    # File dialog state
    pending_file_dialog = False
    pending_file_dialog_callback = None
    
    # Calculate display size with scale
    render_scale = settings_window.render_scale
    STATUS_BAR_HEIGHT = 25
    display_width = base_width * render_scale
    display_height = base_height * render_scale + ui.MENU_BAR_HEIGHT + STATUS_BAR_HEIGHT
    
    # Create main screen
    screen = pygame.display.set_mode((display_width, display_height))
    pygame.display.set_caption("Fantasy Console Emulator")
    clock = pygame.time.Clock()
    
    # Create font for UI
    try:
        font = pygame.font.Font(None, 20)
        small_font = pygame.font.Font(None, 16)
    except:
        font = pygame.font.SysFont("arial", 16)
        small_font = pygame.font.SysFont("arial", 14)
    
    # Create emulator display surface (scaled)
    # Create emulator surface without alpha for numpy compatibility
    # pygame.surfarray requires a surface without alpha channel
    emulator_surface = pygame.Surface((base_width, base_height))
    emulator_surface = emulator_surface.convert()  # Convert to display format
    
    # Initialize audio
    audio_available = False
    try:
        pygame.mixer.init(frequency=config.APU_SAMPLE_RATE, size=-16, channels=1, buffer=512)
        import numpy as np
        
        class AudioStream:
            def __init__(self):
                self.buffer = []
                self.max_buffer_size = 2048
                
            def queue(self, sample_bytes):
                samples = np.frombuffer(sample_bytes, dtype=np.int16)
                self.buffer.extend(samples)
                
                if len(self.buffer) >= 512:
                    chunk = np.array(self.buffer[:512], dtype=np.int16)
                    self.buffer = self.buffer[512:]
                    try:
                        sound_chunk = pygame.sndarray.make_sound(chunk)
                        sound_chunk.play()
                    except Exception:
                        pass
        
        audio_stream = AudioStream()
        apu.apu_set_audio_stream(audio_stream)
        audio_available = True
        ui.logger.info("Audio initialized")
    except Exception as e:
        ui.logger.warning(f"Audio initialization failed: {e}")
        audio_available = False
    
    ui.logger.info("Initialization complete!")
    ui.logger.info(f"Display: {display_width}x{display_height} (scale: {render_scale}x)")
    if rom_loaded:
        ui.logger.info("ROM loaded - ready to start emulation")
    else:
        ui.logger.info("No ROM loaded - use File > Open ROM to load one")
    
    # Main emulation loop
    running = True
    mouse_pos = (0, 0)
    
    while running:
        # Calculate delta time
        current_time = time.time()
        delta_time = current_time - emu.last_frame_time
        
        # Handle pygame events
        for event in pygame.event.get():
            if event.type == pygame.QUIT:
                running = False
            elif event.type == pygame.KEYDOWN:
                key = event.key
                modifiers = pygame.key.get_mods()
                
                if key == pygame.K_ESCAPE:
                    running = False
                elif key == pygame.K_p:
                    debug.debug_toggle_pause()
                    if emu.paused:
                        stop_emulation()
                    else:
                        start_emulation()
                elif key == pygame.K_d:
                    debug.debug_toggle()
                elif key == pygame.K_s:
                    debug.debug_toggle_step_mode()
                elif key == pygame.K_c and (modifiers & pygame.KMOD_CTRL):
                    # Ctrl+C to copy selection in hex debugger
                    if hex_debugger.visible and hex_debugger.selection_start and hex_debugger.selection_end:
                        hex_debugger.copy_selection()
            elif event.type == pygame.MOUSEBUTTONDOWN:
                mouse_pos = pygame.mouse.get_pos()
                mouse_button = event.button  # 1 = left, 3 = right
                
                # Check if clicking on menu dropdown first
                if menu_bar.active_menu:
                    clicked_item = ui.handle_menu_click(menu_bar, mouse_pos, font)
                    if clicked_item:
                        menu_bar.active_menu = None
                        
                        # Handle menu actions
                        if clicked_item == "Open ROM...":
                            # Start file dialog (non-blocking)
                            ui.open_file_dialog()
                            pending_file_dialog = True
                            pending_file_dialog_callback = "open_rom"
                        elif clicked_item == "Exit":
                            running = False
                        elif clicked_item == "Start":
                            start_emulation()
                        elif clicked_item == "Stop":
                            stop_emulation()
                        elif clicked_item == "Reset":
                            reset_emulator()
                            if current_rom_file:
                                # Reload ROM
                                rom_loaded = load_rom_file(current_rom_file)
                        elif clicked_item == "Pause/Resume":
                            if emu.paused:
                                start_emulation()
                            else:
                                stop_emulation()
                        elif clicked_item == "Step":
                            step_emulation()
                        elif clicked_item == "Settings":
                            settings_window.visible = not settings_window.visible
                        elif clicked_item == "Hex Debugger":
                            hex_debugger.visible = not hex_debugger.visible
                        elif clicked_item == "CPU State":
                            debug.debug_print_cpu_state()
                        elif clicked_item == "Memory Info":
                            debug.debug_print_memory_info()
                        elif clicked_item == "PPU State":
                            debug.debug_print_ppu_state()
                        elif clicked_item == "APU State":
                            debug.debug_print_apu_state()
                    else:
                        # Clicked outside dropdown, close it
                        menu_bar.active_menu = None
                
                # Check menu bar click (only if no dropdown is open or if clicking on menu bar)
                if mouse_pos[1] < ui.MENU_BAR_HEIGHT:
                    # If clicking on menu bar, toggle the menu
                    clicked_menu = menu_bar.handle_click(mouse_pos, font)
                    if clicked_menu:
                        ui.logger.debug(f"Menu '{clicked_menu}' clicked, active_menu={menu_bar.active_menu}")
                
                # Check hex debugger click
                if hex_debugger.handle_click(mouse_pos, mouse_button):
                    pass  # Handled in hex_debugger
                
                # Check settings window click
                if settings_window.handle_click(mouse_pos, mouse_button):
                    # Update display size if scale changed
                    new_scale = settings_window.render_scale
                    if new_scale != render_scale:
                        render_scale = new_scale
                        display_width = base_width * render_scale
                        display_height = base_height * render_scale + ui.MENU_BAR_HEIGHT + STATUS_BAR_HEIGHT
                        screen = pygame.display.set_mode((display_width, display_height))
                        ui.logger.info(f"Display scale changed to {render_scale}x")
            
            elif event.type == pygame.MOUSEMOTION:
                mouse_pos = pygame.mouse.get_pos()
                # Update selection if dragging text
                if hex_debugger.selecting:
                    hex_debugger.handle_click(mouse_pos, 1)
                # Handle window dragging
                if hex_debugger.dragging:
                    hex_debugger.handle_mouse_move(mouse_pos)
                if settings_window.dragging:
                    settings_window.handle_mouse_move(mouse_pos)
            
            elif event.type == pygame.MOUSEBUTTONUP:
                mouse_pos = pygame.mouse.get_pos()
                hex_debugger.handle_mouse_up(mouse_pos)
                settings_window.handle_mouse_up(mouse_pos)
            
            elif event.type == pygame.MOUSEWHEEL:
                # Handle scroll in hex debugger
                if hex_debugger.visible:
                    hex_debugger.handle_scroll(event.y)
        
        # Clear screen
        screen.fill((0, 0, 0))
        
        # Draw menu bar
        menu_bar.draw(screen, font)
        
        # Draw menu dropdown (before emulator surface so it's on top)
        ui.draw_menu_dropdown(screen, font, menu_bar, mouse_pos)
        
        # Check if it's time for a new frame (60 FPS)
        # Log frame timing check (only occasionally to avoid spam, and only if logging enabled)
        if ui.logger.enabled and ui.logger.detailed_logging and emu.frame_count % 60 == 0:  # Log every 60 frames
            ui.logger.trace(f"Main: Frame timing check - delta_time={delta_time:.6f}, frame_time={emu.frame_time:.6f}, delta_time >= frame_time={delta_time >= emu.frame_time}", "PPU")
        
        if delta_time >= emu.frame_time:
            # Update input
            keys = pygame.key.get_pressed()
            input_module.input_update_from_pygame(keys)
            
            # If paused, wait
            if emu.paused and not emu.step_mode:
                clock.tick(10)  # Low CPU usage when paused
            else:
                # Run CPU for one frame's worth of cycles
                if rom_loaded:
                    target_cycles = emu.cpu.cycles + config.CPU_CYCLES_PER_FRAME
                    cpu.cpu_run_cycles(target_cycles)
                
                # Update APU
                apu.apu_update()
                
                # Render frame to emulator surface
                if ui.logger.enabled and ui.logger.detailed_logging:
                    ui.logger.trace(f"Main: Calling ppu_render_frame() for frame {emu.frame_count}, paused={emu.paused}, step_mode={emu.step_mode}, rom_loaded={rom_loaded}", "PPU")
                try:
                    ppu.ppu_render_frame()
                except Exception as e:
                    ui.logger.error(f"Exception in ppu_render_frame(): {e}", "PPU")
                    import traceback
                    ui.logger.error(traceback.format_exc(), "PPU")
                
                # Draw to emulator surface using faster method
                # Convert output buffer to bytes and use pygame.image.fromstring for speed
                # Optimized: pre-allocate and use direct indexing with memoryview for speed
                buffer_size = base_width * base_height
                pixel_bytes = bytearray(buffer_size * 3)
                output_buffer = config.ppu_output_buffer
                # Use memoryview for faster access (if available)
                try:
                    pixel_view = memoryview(pixel_bytes)
                    for i in range(min(len(output_buffer), buffer_size)):
                        rgb = output_buffer[i]
                        byte_index = i * 3
                        pixel_view[byte_index] = (rgb >> 16) & 0xFF  # R
                        pixel_view[byte_index + 1] = (rgb >> 8) & 0xFF  # G
                        pixel_view[byte_index + 2] = rgb & 0xFF  # B
                except:
                    # Fallback if memoryview fails
                    for i in range(min(len(output_buffer), buffer_size)):
                        rgb = output_buffer[i]
                        byte_index = i * 3
                        pixel_bytes[byte_index] = (rgb >> 16) & 0xFF  # R
                        pixel_bytes[byte_index + 1] = (rgb >> 8) & 0xFF  # G
                        pixel_bytes[byte_index + 2] = rgb & 0xFF  # B
                
                # Create surface from bytes (much faster than set_at)
                pixel_string = bytes(pixel_bytes)
                temp_surface = pygame.image.fromstring(pixel_string, (base_width, base_height), 'RGB')
                emulator_surface.blit(temp_surface, (0, 0))
                
                # End VBlank
                emu.ppu.vblank_active = False
                
                # Update frame counter
                emu.frame_count += 1
                emu.last_frame_time = current_time
                
                # If step mode, pause after this frame
                if emu.step_mode:
                    emu.paused = True
        
        # Scale and blit emulator surface to screen (offset by menu bar height)
        scaled_surface = pygame.transform.scale(emulator_surface, 
                                                (base_width * render_scale, 
                                                 base_height * render_scale))
        screen.blit(scaled_surface, (0, ui.MENU_BAR_HEIGHT))
        
        # Draw windows in z-order (lowest z-order first, so highest draws last/on top)
        windows = []
        if settings_window.visible:
            windows.append(settings_window)
        if hex_debugger.visible:
            windows.append(hex_debugger)
        
        # Sort by z-order (ascending - lower numbers draw first, higher numbers draw last/on top)
        windows.sort(key=lambda w: w.z_order)
        
        # Draw windows in z-order
        for window in windows:
            if isinstance(window, ui.SettingsWindow):
                window.draw(screen, font)
            elif isinstance(window, ui.HexDebuggerWindow):
                window.draw(screen, small_font)
        
        # Draw menu dropdown last (always on top)
        ui.draw_menu_dropdown(screen, font, menu_bar, mouse_pos)
        
        # Check for file dialog result (non-blocking)
        if pending_file_dialog:
            filename = ui.check_file_dialog_result()
            if filename is not None:  # None means still waiting, empty string means cancelled
                pending_file_dialog = False
                if filename:  # Non-empty string means file was selected
                    if pending_file_dialog_callback == "open_rom":
                        reset_emulator()
                        current_rom_file = filename
                        rom_loaded = load_rom_file(filename)
                pending_file_dialog_callback = None
        
        # Draw status bar at bottom (after windows so it's visible)
        status_bar_y = display_height - STATUS_BAR_HEIGHT
        pygame.draw.rect(screen, (40, 40, 40), (0, status_bar_y, display_width, STATUS_BAR_HEIGHT))
        pygame.draw.line(screen, (100, 100, 100), (0, status_bar_y), (display_width, status_bar_y), 1)
        
        # Calculate FPS (smoothed)
        if delta_time > 0:
            current_fps = 1.0 / delta_time
        else:
            current_fps = 0
        
        # Build status text
        status_parts = []
        status_parts.append(f"Frame: {emu.frame_count}")
        if rom_loaded:
            status_parts.append(f"PC: {emu.cpu.pc_bank:02X}:{emu.cpu.pc_offset:04X}")
        status_parts.append(f"FPS: {current_fps:.1f}")
        if emu.paused:
            status_parts.append("PAUSED")
        elif emu.step_mode:
            status_parts.append("STEP MODE")
        else:
            status_parts.append("RUNNING")
        
        status_text = " | ".join(status_parts)
        status_surface = small_font.render(status_text, True, (200, 200, 200))
        screen.blit(status_surface, (5, status_bar_y + 5))
        
        pygame.display.flip()
        
        # Limit to 60 FPS
        clock.tick(config.TARGET_FPS)
    
    # Cleanup
    pygame.quit()
    ui.logger.info("=" * 60)
    ui.logger.info("Emulator shutting down...")
    ui.logger.info(f"Total frames: {emu.frame_count}")
    ui.logger.info("=" * 60)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\nInterrupted by user")
        sys.exit(0)
    except Exception as e:
        ui.logger.error(f"{type(e).__name__}: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
