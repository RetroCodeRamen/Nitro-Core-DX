"""
ui.py - User Interface components (menu bar, windows, dialogs)
"""

import pygame
import sys
import os
import config
import cpu
import memory
import rom
import threading

# Try to import tkinter for file dialogs
try:
    import tkinter as tk
    from tkinter import filedialog
    TKINTER_AVAILABLE = True
except ImportError:
    TKINTER_AVAILABLE = False

# UI Constants
MENU_BAR_HEIGHT = 25
MENU_ITEM_PADDING = 10
WINDOW_BG_COLOR = (40, 40, 40)
WINDOW_TEXT_COLOR = (255, 255, 255)
MENU_BG_COLOR = (50, 50, 50)
MENU_HOVER_COLOR = (70, 70, 70)
MENU_TEXT_COLOR = (255, 255, 255)
DEBUG_WINDOW_WIDTH = 600
DEBUG_WINDOW_HEIGHT = 400
SETTINGS_WINDOW_WIDTH = 400
SETTINGS_WINDOW_HEIGHT = 400  # Increased for save log button


class Logger:
    """Enhanced logger with log levels and filtering"""
    # Log levels (higher number = more verbose)
    LEVEL_ERROR = 0
    LEVEL_WARNING = 1
    LEVEL_INFO = 2
    LEVEL_DEBUG = 3
    LEVEL_TRACE = 4  # Very detailed (CPU instructions, memory access, etc.)
    
    def __init__(self):
        self.messages = []
        self.max_messages = 100000  # Increased to 100,000 to capture boot sequence
        self.enabled = True
        self.detailed_logging = False  # Detailed logging disabled by default (only boot log captured)
        self.log_level = self.LEVEL_INFO  # Default to INFO level (ERROR, WARNING, INFO only)
        self.log_cpu = False  # Log CPU instructions
        self.log_memory = False  # Log memory reads/writes
        self.log_ppu = False  # Log PPU operations
        self.log_rom = False  # Log ROM operations
        self.auto_save_count = 0  # Track messages for auto-save
        self.auto_save_done = False  # Flag to prevent multiple auto-saves
        
    def set_level(self, level):
        """Set minimum log level (ERROR=0, WARNING=1, INFO=2, DEBUG=3, TRACE=4)"""
        self.log_level = level
    
    def set_detailed_logging(self, enabled):
        """Enable/disable detailed logging (TRACE level)"""
        self.detailed_logging = enabled
        if enabled:
            self.log_level = self.LEVEL_TRACE
    
    def set_category_logging(self, cpu=False, memory=False, ppu=False, rom=False):
        """Enable/disable logging for specific categories"""
        self.log_cpu = cpu
        self.log_memory = memory
        self.log_ppu = ppu
        self.log_rom = rom
        
    def log(self, level, message, category=None):
        """Log a message with level and optional category"""
        if not self.enabled:
            return
        
        # Check if this level should be logged
        if level > self.log_level:
            return
        
        # Check category-specific logging
        if category == "CPU" and not self.log_cpu and not self.detailed_logging:
            return
        if category == "MEM" and not self.log_memory and not self.detailed_logging:
            return
        if category == "PPU" and not self.log_ppu and not self.detailed_logging:
            return
        if category == "ROM" and not self.log_rom and not self.detailed_logging:
            return
        
        import time
        timestamp = time.strftime("%H:%M:%S")
        level_name = ["ERROR", "WARN", "INFO", "DEBUG", "TRACE"][min(level, 4)]
        if category:
            log_entry = f"[{timestamp}] [{level_name}] [{category}] {message}"
        else:
            log_entry = f"[{timestamp}] [{level_name}] {message}"
        print(log_entry)
        self.messages.append(log_entry)
        if len(self.messages) > self.max_messages:
            self.messages.pop(0)
        
        # Auto-save first 100000 messages to file (only if we have enough messages)
        self.auto_save_count += 1
        if not self.auto_save_done and self.auto_save_count >= 100000 and len(self.messages) >= 100000:
            self._auto_save_boot_log()
            self.auto_save_done = True
    
    def _auto_save_boot_log(self):
        """Auto-save the first 100000 messages to a file"""
        import time
        try:
            timestamp = time.strftime("%Y%m%d_%H%M%S")
            filename = f"emulator_boot_log_{timestamp}.txt"
            
            # Save first 100000 messages
            with open(filename, 'w') as f:
                f.write("=" * 80 + "\n")
                f.write("Fantasy Console Emulator - Auto-Saved Boot Log\n")
                f.write(f"Generated: {time.strftime('%Y-%m-%d %H:%M:%S')}\n")
                f.write(f"Total messages: {min(100000, len(self.messages))}\n")
                f.write("=" * 80 + "\n")
                f.write("\n")
                
                # Write first 100000 messages
                for msg in self.messages[:100000]:
                    f.write(msg + "\n")
            
            print(f"[AUTO-SAVE] Boot log saved to: {filename}")
        except Exception as e:
            print(f"[AUTO-SAVE] Failed to save boot log: {e}")
    
    def info(self, message, category=None):
        self.log(self.LEVEL_INFO, message, category)
    
    def error(self, message, category=None):
        self.log(self.LEVEL_ERROR, message, category)
    
    def warning(self, message, category=None):
        self.log(self.LEVEL_WARNING, message, category)
    
    def debug(self, message, category=None):
        self.log(self.LEVEL_DEBUG, message, category)
    
    def trace(self, message, category=None):
        """Trace level - very detailed logging"""
        self.log(self.LEVEL_TRACE, message, category)
    
    def save_to_file(self, filename=None):
        """Save all logged messages to a file"""
        import time
        import os
        
        if filename is None:
            # Generate default filename with timestamp
            timestamp = time.strftime("%Y%m%d_%H%M%S")
            filename = f"emulator_log_{timestamp}.txt"
        
        try:
            with open(filename, 'w') as f:
                f.write("=" * 80 + "\n")
                f.write(f"Fantasy Console Emulator - Log File\n")
                f.write(f"Generated: {time.strftime('%Y-%m-%d %H:%M:%S')}\n")
                f.write(f"Total messages: {len(self.messages)}\n")
                f.write("=" * 80 + "\n\n")
                for msg in self.messages:
                    f.write(msg + "\n")
            self.info(f"Log saved to {filename} ({len(self.messages)} messages)")
            return filename
        except Exception as e:
            self.error(f"Failed to save log file: {e}")
            return None


# Global logger instance
logger = Logger()


class MenuBar:
    """Menu bar with dropdown menus"""
    def __init__(self, width):
        self.width = width
        self.height = MENU_BAR_HEIGHT
        self.menus = {}
        self.active_menu = None
        self.active_submenu = None
        
    def add_menu(self, name, items):
        """Add a menu with items"""
        self.menus[name] = items
    
    def draw(self, surface, font):
        """Draw the menu bar"""
        # Draw background
        pygame.draw.rect(surface, MENU_BG_COLOR, (0, 0, self.width, self.height))
        
        # Draw menu items
        x = MENU_ITEM_PADDING
        for menu_name in self.menus.keys():
            text = font.render(menu_name, True, MENU_TEXT_COLOR)
            text_rect = text.get_rect()
            text_rect.topleft = (x, (self.height - text_rect.height) // 2)
            surface.blit(text, text_rect)
            
            # Draw underline if active
            if self.active_menu == menu_name:
                pygame.draw.line(surface, MENU_TEXT_COLOR, 
                               (x, self.height - 2), 
                               (x + text_rect.width, self.height - 2), 2)
            
            x += text_rect.width + MENU_ITEM_PADDING * 2
    
    def handle_click(self, pos, font):
        """Handle mouse click on menu bar"""
        x, y = pos
        if y > self.height:
            return None
        
        # Check which menu was clicked
        current_x = MENU_ITEM_PADDING
        for menu_name in self.menus.keys():
            # Get actual text width
            text = font.render(menu_name, True, MENU_TEXT_COLOR)
            menu_width = text.get_width() + MENU_ITEM_PADDING * 2
            if current_x <= x < current_x + menu_width:
                if self.active_menu == menu_name:
                    self.active_menu = None
                    self.active_submenu = None
                else:
                    self.active_menu = menu_name
                    self.active_submenu = None
                return menu_name
            current_x += menu_width
        
        return None
    
    def get_active_menu_items(self):
        """Get items for active menu"""
        if self.active_menu:
            return self.menus.get(self.active_menu, [])
        return []


# Global z-order counter for windows
_window_z_order_counter = 0

def get_next_z_order():
    """Get the next z-order value (higher = on top)"""
    global _window_z_order_counter
    _window_z_order_counter += 1
    return _window_z_order_counter

class HexDebuggerWindow:
    """Hex debugger window showing memory contents"""
    def __init__(self, x, y):
        self.x = x
        self.y = y
        self.width = DEBUG_WINDOW_WIDTH
        self.height = DEBUG_WINDOW_HEIGHT
        self.visible = False
        self.scroll_offset = 0
        self.memory_bank = 1  # Default to VRAM (bank 1 = VRAM)
        self.memory_start = 0
        self.selection_start = None  # (line, char) tuple for selection start
        self.selection_end = None  # (line, char) tuple for selection end
        self.selecting = False  # Whether user is currently selecting
        self.dragging = False  # Whether window is being dragged
        self.drag_offset = (0, 0)  # Offset from mouse to window corner when dragging
        self.z_order = get_next_z_order()  # Z-order for window stacking
        # CPU state capture
        self.cpu_capture_enabled = False  # Whether we're capturing CPU states
        self.cpu_capture_count = 10000  # Number of CPU cycles to capture (default 10000)
        self.cpu_capture_buffer = []  # Buffer to store CPU state snapshots
        self.cpu_capture_start_cycles = 0  # CPU cycles when capture started
        
    def draw(self, surface, font):
        """Draw the hex debugger window"""
        if not self.visible:
            return
        
        # Draw window background
        pygame.draw.rect(surface, WINDOW_BG_COLOR, 
                        (self.x, self.y, self.width, self.height))
        pygame.draw.rect(surface, MENU_TEXT_COLOR, 
                        (self.x, self.y, self.width, self.height), 2)
        
        # Draw title bar (draggable area)
        title_bar_height = 25
        pygame.draw.rect(surface, (60, 60, 60), 
                        (self.x, self.y, self.width, title_bar_height))
        pygame.draw.line(surface, MENU_TEXT_COLOR, 
                        (self.x, self.y + title_bar_height), 
                        (self.x + self.width, self.y + title_bar_height), 1)
        
        # Draw title
        title = font.render("Hex Debugger", True, MENU_TEXT_COLOR)
        surface.blit(title, (self.x + 10, self.y + 5))
        
        # Draw close button (X) - always visible
        close_button_size = 20
        close_button_rect = pygame.Rect(self.x + self.width - close_button_size - 5, 
                                        self.y + 2, close_button_size, close_button_size)
        pygame.draw.rect(surface, (200, 50, 50), close_button_rect)
        pygame.draw.rect(surface, MENU_TEXT_COLOR, close_button_rect, 1)
        # Draw X
        close_x = font.render("×", True, MENU_TEXT_COLOR)
        surface.blit(close_x, (close_button_rect.centerx - close_x.get_width() // 2,
                               close_button_rect.centery - close_x.get_height() // 2))
        
        # Draw memory bank selector with buttons (offset by title bar) - always visible
        if self.memory_bank == 0xFF:
            bank_text = font.render("CPU Registers", True, MENU_TEXT_COLOR)
        else:
            bank_text = font.render(f"Bank: {self.memory_bank:02X}", True, MENU_TEXT_COLOR)
        surface.blit(bank_text, (self.x + 10, self.y + 35))
        
        # Bank selection buttons - always visible
        bank_prev_rect = pygame.Rect(self.x + 120, self.y + 32, 30, 20)
        bank_next_rect = pygame.Rect(self.x + 155, self.y + 32, 30, 20)
        pygame.draw.rect(surface, MENU_BG_COLOR, bank_prev_rect)
        pygame.draw.rect(surface, MENU_TEXT_COLOR, bank_prev_rect, 1)
        pygame.draw.rect(surface, MENU_BG_COLOR, bank_next_rect)
        pygame.draw.rect(surface, MENU_TEXT_COLOR, bank_next_rect, 1)
        prev_text = font.render("<", True, MENU_TEXT_COLOR)
        next_text = font.render(">", True, MENU_TEXT_COLOR)
        surface.blit(prev_text, (bank_prev_rect.centerx - 5, bank_prev_rect.centery - 8))
        surface.blit(next_text, (bank_next_rect.centerx - 5, bank_next_rect.centery - 8))
        
        # Memory type selector - always visible
        mem_types = ["WRAM", "VRAM", "CGRAM", "ROM"]
        if self.memory_bank == 0xFF:
            mem_type_text = font.render("Type: CPU_REGS", True, MENU_TEXT_COLOR)
        else:
            mem_type_text = font.render(f"Type: {mem_types[min(self.memory_bank, len(mem_types)-1)]}", True, MENU_TEXT_COLOR)
        surface.blit(mem_type_text, (self.x + 200, self.y + 35))
        
        # Dump to file button - always visible
        dump_button_rect = pygame.Rect(self.x + self.width - 100, self.y + 32, 80, 20)
        pygame.draw.rect(surface, MENU_BG_COLOR, dump_button_rect)
        pygame.draw.rect(surface, MENU_TEXT_COLOR, dump_button_rect, 1)
        dump_text = font.render("Dump", True, MENU_TEXT_COLOR)
        surface.blit(dump_text, (dump_button_rect.centerx - dump_text.get_width() // 2, 
                                 dump_button_rect.centery - dump_text.get_height() // 2))
        
        # Copy button (only show if selection exists and not viewing CPU registers)
        if self.memory_bank != 0xFF and self.selection_start and self.selection_end:
            copy_button_rect = pygame.Rect(self.x + self.width - 200, self.y + 32, 80, 20)
            pygame.draw.rect(surface, MENU_BG_COLOR, copy_button_rect)
            pygame.draw.rect(surface, MENU_TEXT_COLOR, copy_button_rect, 1)
            copy_text = font.render("Copy", True, MENU_TEXT_COLOR)
            surface.blit(copy_text, (copy_button_rect.centerx - copy_text.get_width() // 2,
                                    copy_button_rect.centery - copy_text.get_height() // 2))
        
        # Draw CPU registers if viewing bank 0xFF (special bank for CPU registers)
        if self.memory_bank == 0xFF:
            # Show CPU registers instead of memory
            emu = config.emulator
            y_offset = 60
            reg_text = [
                f"R0={emu.cpu.r0:04X}  R1={emu.cpu.r1:04X}  R2={emu.cpu.r2:04X}  R3={emu.cpu.r3:04X}",
                f"R4={emu.cpu.r4:04X}  R5={emu.cpu.r5:04X}  R6={emu.cpu.r6:04X}  R7={emu.cpu.r7:04X}",
                f"PC={emu.cpu.pc_bank:02X}:{emu.cpu.pc_offset:04X}  SP={emu.cpu.sp:04X}",
                f"PBR={emu.cpu.pbr:02X}  DBR={emu.cpu.dbr:02X}  Flags={emu.cpu.flags:04X}",
                f"Cycles={emu.cpu.cycles}  Paused={emu.paused}  Step={emu.step_mode}",
            ]
            for i, text in enumerate(reg_text):
                reg_surface = font.render(text, True, (200, 255, 200))
                surface.blit(reg_surface, (self.x + 10, self.y + y_offset + i * 20))
            
            # Show capture status
            y_offset += 120
            if self.cpu_capture_enabled:
                captured = len(self.cpu_capture_buffer)
                remaining = max(0, self.cpu_capture_count - captured)
                capture_text = f"Capturing: {captured}/{self.cpu_capture_count} states"
                if remaining > 0:
                    capture_text += f" ({remaining} remaining)"
                else:
                    capture_text += " (Complete - saving...)"
                capture_surface = font.render(capture_text, True, (255, 200, 100))
                surface.blit(capture_surface, (self.x + 10, self.y + y_offset))
            else:
                capture_text = f"Click 'Dump' to capture {self.cpu_capture_count} CPU states"
                capture_surface = font.render(capture_text, True, (150, 150, 150))
                surface.blit(capture_surface, (self.x + 10, self.y + y_offset))
            
            return  # Don't show memory dump for CPU register view
        
        # Draw hex dump (16 bytes per line)
        y_offset = 60
        bytes_per_line = 16
        lines_to_show = (self.height - y_offset - 10) // 20
        
        # Store line data for selection (reset each frame)
        self.line_data = []  # List of (addr, hex_str, ascii_str) tuples
        
        for line in range(lines_to_show):
            addr = self.memory_start + (self.scroll_offset + line) * bytes_per_line
            addr_text = f"{addr:04X}: "
            
            # Draw address
            addr_surface = font.render(addr_text, True, MENU_TEXT_COLOR)
            surface.blit(addr_surface, (self.x + 10, self.y + y_offset + line * 20))
            
            # Draw hex bytes
            hex_str = ""
            ascii_str = ""
            x_offset = 80
            for i in range(bytes_per_line):
                byte_addr = addr + i
                # Read from appropriate memory based on bank/type
                if self.memory_bank == 0:  # WRAM
                    if byte_addr < 0x8000:
                        byte_val = memory.memory_read8(0, byte_addr)
                    else:
                        byte_val = 0
                elif self.memory_bank == 1:  # VRAM
                    if byte_addr < config.PPU_VRAM_SIZE:
                        byte_val = config.ppu_vram[byte_addr]
                    else:
                        byte_val = 0
                elif self.memory_bank == 2:  # CGRAM
                    if byte_addr < config.PPU_CGRAM_SIZE:
                        byte_val = config.ppu_cgram[byte_addr]
                    else:
                        byte_val = 0
                else:  # ROM (bank 1+)
                    # ROM uses LoROM mapping: data appears at 0x8000-0xFFFF
                    # For hex debugger, show raw ROM data (not mapped)
                    emu = config.emulator
                    if hasattr(emu.memory, 'rom_data') and emu.memory.rom_data:
                        # Calculate raw ROM offset
                        # Bank 1 = first 32KB, Bank 2 = second 32KB, etc.
                        rom_offset = (self.memory_bank - 1) * 32768 + byte_addr
                        if 0 <= rom_offset < emu.memory.rom_size:
                            byte_val = emu.memory.rom_data[rom_offset]
                        else:
                            byte_val = 0
                    else:
                        # Fallback to mapped read (will return 0 for offset < 0x8000)
                        byte_val = memory.memory_read8(self.memory_bank, byte_addr)
                
                hex_str += f"{byte_val:02X} "
                ascii_str += chr(byte_val) if 32 <= byte_val < 127 else "."
            
            # Store line data
            self.line_data.append((addr, hex_str, ascii_str))
            
            # Check if this line is selected
            line_selected = False
            if self.selection_start and self.selection_end:
                start_line, start_char = self.selection_start
                end_line, end_char = self.selection_end
                actual_line = self.scroll_offset + line
                if start_line <= actual_line <= end_line or end_line <= actual_line <= start_line:
                    line_selected = True
            
            # Draw hex (highlight if selected)
            hex_color = (255, 255, 200) if line_selected else (200, 200, 255)
            hex_surface = font.render(hex_str, True, hex_color)
            surface.blit(hex_surface, (self.x + x_offset, self.y + y_offset + line * 20))
            
            # Draw ASCII (highlight if selected)
            ascii_color = (255, 255, 200) if line_selected else (200, 255, 200)
            ascii_surface = font.render(ascii_str, True, ascii_color)
            surface.blit(ascii_surface, (self.x + x_offset + 200, self.y + y_offset + line * 20))
    
    def handle_scroll(self, direction):
        """Handle scroll wheel"""
        if direction > 0:
            self.scroll_offset = max(0, self.scroll_offset - 1)
        else:
            self.scroll_offset += 1
    
    def get_line_at_pos(self, pos):
        """Get line number and character position from mouse position"""
        x, y = pos
        y_offset = 60
        bytes_per_line = 16
        
        # Check if click is in hex dump area
        if y < self.y + y_offset:
            return None
        
        line = (y - self.y - y_offset) // 20
        if line < 0:
            return None
        
        # Determine if clicking on hex or ASCII
        x_offset = 80
        if x < self.x + x_offset:
            return None  # Clicked on address
        elif x < self.x + x_offset + 200:
            # Clicked on hex
            char_pos = (x - self.x - x_offset) // 12  # Approximate char width
            return (line, 'hex', char_pos)
        else:
            # Clicked on ASCII
            char_pos = (x - self.x - x_offset - 200) // 8  # Approximate char width
            return (line, 'ascii', char_pos)
    
    def handle_click(self, pos, button=1):
        """Handle click on hex debugger window
        button: 1 = left, 3 = right
        """
        if not self.visible:
            return False
        
        x, y = pos
        if not (self.x <= x < self.x + self.width and self.y <= y < self.y + self.height):
            return False
        
        # Check close button
        close_button_size = 20
        close_button_rect = pygame.Rect(self.x + self.width - close_button_size - 5,
                                        self.y + 2, close_button_size, close_button_size)
        if close_button_rect.collidepoint(pos):
            self.visible = False
            logger.info("Hex debugger closed")
            return True
        
        # Check title bar for dragging
        title_bar_height = 25
        title_bar_rect = pygame.Rect(self.x, self.y, self.width, title_bar_height)
        if title_bar_rect.collidepoint(pos) and button == 1:
            # Bring window to front when clicked
            self.z_order = get_next_z_order()
            # Start dragging
            self.dragging = True
            self.drag_offset = (x - self.x, y - self.y)
            return True
        
        # Bring window to front when any part is clicked (before processing other clicks)
        self.z_order = get_next_z_order()
        
        # Check dump button
        dump_button_rect = pygame.Rect(self.x + self.width - 100, self.y + 32, 80, 20)
        if dump_button_rect.collidepoint(pos):
            self.dump_to_file()
            return True
        
        # Check copy button
        if self.selection_start and self.selection_end:
            copy_button_rect = pygame.Rect(self.x + self.width - 200, self.y + 32, 80, 20)
            if copy_button_rect.collidepoint(pos):
                self.copy_selection()
                return True
        
        # Check bank selection buttons
        bank_prev_rect = pygame.Rect(self.x + 120, self.y + 32, 30, 20)
        bank_next_rect = pygame.Rect(self.x + 155, self.y + 32, 30, 20)
        
        if bank_prev_rect.collidepoint(pos):
            if self.memory_bank == 0:
                self.memory_bank = 0xFF  # Wrap to CPU registers
            elif self.memory_bank == 0xFF:
                self.memory_bank = 125  # Wrap from CPU to last ROM bank
            else:
                self.memory_bank = max(0, self.memory_bank - 1)
            self.scroll_offset = 0
            self.selection_start = None
            self.selection_end = None
            logger.info(f"Hex debugger: switched to bank {self.memory_bank:02X}")
            return True
        elif bank_next_rect.collidepoint(pos):
            if self.memory_bank == 0xFF:
                self.memory_bank = 0  # Wrap from CPU to WRAM
            elif self.memory_bank == 125:
                self.memory_bank = 0xFF  # Wrap to CPU registers
            else:
                self.memory_bank = min(125, self.memory_bank + 1)
            self.scroll_offset = 0
            self.selection_start = None
            self.selection_end = None
            logger.info(f"Hex debugger: switched to bank {self.memory_bank:02X}")
            return True
        
        # Handle selection (left click and drag)
        if button == 1:
            line_info = self.get_line_at_pos(pos)
            if line_info:
                line, area, char_pos = line_info
                actual_line = self.scroll_offset + line
                if not self.selecting:
                    # Start selection
                    self.selection_start = (actual_line, char_pos)
                    self.selection_end = (actual_line, char_pos)
                    self.selecting = True
                else:
                    # Update selection end
                    self.selection_end = (actual_line, char_pos)
                return True
        
        return False
    
    def handle_mouse_up(self, pos):
        """Handle mouse button release"""
        if self.selecting:
            self.selecting = False
        if self.dragging:
            self.dragging = False
    
    def handle_mouse_move(self, pos):
        """Handle mouse movement (for dragging)"""
        if self.dragging:
            self.x = pos[0] - self.drag_offset[0]
            self.y = pos[1] - self.drag_offset[1]
            # Keep window on screen
            screen_width = pygame.display.get_surface().get_width()
            screen_height = pygame.display.get_surface().get_height()
            self.x = max(0, min(self.x, screen_width - self.width))
            # Keep window above status bar (status bar is 25 pixels tall)
            STATUS_BAR_HEIGHT = 25
            self.y = max(MENU_BAR_HEIGHT, min(self.y, screen_height - self.height - STATUS_BAR_HEIGHT))
    
    def copy_selection(self):
        """Copy selected text to clipboard"""
        if not self.selection_start or not self.selection_end:
            return
        
        if not TKINTER_AVAILABLE:
            logger.warning("tkinter not available, cannot copy to clipboard")
            return
        
        start_line, start_char = self.selection_start
        end_line, end_char = self.selection_end
        
        # Swap if start > end
        if start_line > end_line or (start_line == end_line and start_char > end_char):
            start_line, start_char, end_line, end_char = end_line, end_char, start_line, start_char
        
        # Build text to copy
        lines_to_copy = []
        bytes_per_line = 16
        y_offset = 60
        
        for line_num in range(start_line, end_line + 1):
            actual_line = line_num - self.scroll_offset
            if 0 <= actual_line < len(self.line_data):
                addr, hex_str, ascii_str = self.line_data[actual_line]
                
                if line_num == start_line == end_line:
                    # Single line selection
                    hex_part = hex_str[start_char * 3:(end_char + 1) * 3].strip()
                    ascii_part = ascii_str[start_char:end_char + 1]
                    lines_to_copy.append(f"{addr:04X}: {hex_part}  {ascii_part}")
                elif line_num == start_line:
                    # First line
                    hex_part = hex_str[start_char * 3:].strip()
                    ascii_part = ascii_str[start_char:]
                    lines_to_copy.append(f"{addr:04X}: {hex_part}  {ascii_part}")
                elif line_num == end_line:
                    # Last line
                    hex_part = hex_str[:(end_char + 1) * 3].strip()
                    ascii_part = ascii_str[:end_char + 1]
                    lines_to_copy.append(f"{addr:04X}: {hex_part}  {ascii_part}")
                else:
                    # Middle line
                    lines_to_copy.append(f"{addr:04X}: {hex_str.strip()}  {ascii_str}")
        
        text_to_copy = "\n".join(lines_to_copy)
        
        # Copy to clipboard using tkinter
        root = tk.Tk()
        root.withdraw()
        root.clipboard_clear()
        root.clipboard_append(text_to_copy)
        root.update()
        root.destroy()
        
        logger.info(f"Copied {len(lines_to_copy)} lines to clipboard")
    
    def _dump_to_file_threaded(self, default_name, mem_type):
        """Internal method to show file dialog in a separate thread"""
        try:
            logger.info("Opening file save dialog in thread...")
            # Create tkinter root in this thread (tkinter requires root in same thread)
            root = tk.Tk()
            root.withdraw()
            try:
                root.attributes('-topmost', True)
            except:
                pass
            root.update()
            root.lift()
            try:
                root.focus_force()
            except:
                pass
            
            logger.info(f"Showing dialog with default name: {default_name}")
            filename = filedialog.asksaveasfilename(
                title="Save Memory Dump",
                defaultextension=".txt",
                filetypes=[("Text files", "*.txt"), ("All files", "*.*")],
                initialfile=default_name,
                parent=root
            )
            
            root.quit()  # Quit the tkinter mainloop
            root.destroy()
            
            if filename:
                logger.info(f"User selected file: {filename}")
                # Save the file
                self._do_dump_save(filename, mem_type)
            else:
                logger.info("Memory dump cancelled by user")
        except Exception as e:
            logger.error(f"Error in file dialog thread: {e}")
            import traceback
            traceback.print_exc()
            # Fallback: save to default location
            logger.info(f"Using fallback filename: {default_name}")
            try:
                self._do_dump_save(default_name, mem_type)
            except Exception as e2:
                logger.error(f"Failed to save with fallback: {e2}")
    
    def _do_dump_save(self, filename, mem_type):
        """Actually save the dump to file (called after filename is determined)"""
        
        # Get memory data
        bytes_per_line = 16
        lines_to_dump = (self.height - 60 - 10) // 20
        
        dump_lines = []
        dump_lines.append(f"Memory Dump - Bank {self.memory_bank:02X} ({mem_type})")
        dump_lines.append(f"Start Address: {self.memory_start:04X}")
        dump_lines.append("=" * 60)
        dump_lines.append("")
        
        for line in range(lines_to_dump):
            addr = self.memory_start + (self.scroll_offset + line) * bytes_per_line
            hex_str = ""
            ascii_str = ""
            
            for i in range(bytes_per_line):
                byte_addr = addr + i
                # Read from appropriate memory based on bank/type
                if self.memory_bank == 0:  # WRAM
                    if byte_addr < 0x8000:
                        byte_val = memory.memory_read8(0, byte_addr)
                    else:
                        byte_val = 0
                elif self.memory_bank == 1:  # VRAM
                    if byte_addr < config.PPU_VRAM_SIZE:
                        byte_val = config.ppu_vram[byte_addr]
                    else:
                        byte_val = 0
                elif self.memory_bank == 2:  # CGRAM
                    if byte_addr < config.PPU_CGRAM_SIZE:
                        byte_val = config.ppu_cgram[byte_addr]
                    else:
                        byte_val = 0
                else:  # ROM (bank 1+)
                    # ROM uses LoROM mapping: data appears at 0x8000-0xFFFF
                    # For hex debugger, show raw ROM data (not mapped)
                    emu = config.emulator
                    if hasattr(emu.memory, 'rom_data') and emu.memory.rom_data:
                        # Calculate raw ROM offset
                        # Bank 1 = first 32KB, Bank 2 = second 32KB, etc.
                        rom_offset = (self.memory_bank - 1) * 32768 + byte_addr
                        if 0 <= rom_offset < emu.memory.rom_size:
                            byte_val = emu.memory.rom_data[rom_offset]
                        else:
                            byte_val = 0
                    else:
                        # Fallback to mapped read (will return 0 for offset < 0x8000)
                        byte_val = memory.memory_read8(self.memory_bank, byte_addr)
                
                hex_str += f"{byte_val:02X} "
                ascii_str += chr(byte_val) if 32 <= byte_val < 127 else "."
            
            dump_lines.append(f"{addr:04X}: {hex_str}  {ascii_str}")
        
        # Write to file
        try:
            with open(filename, 'w') as f:
                f.write("\n".join(dump_lines))
            logger.info(f"Memory dump saved to {filename} ({len(dump_lines)} lines)")
            logger.info(f"File saved successfully!")
        except Exception as e:
            logger.error(f"Failed to save memory dump: {e}")
            import traceback
            traceback.print_exc()
    
    def dump_to_file(self):
        """Dump current memory view to a file, or start CPU state capture"""
        logger.info("Dump button clicked")
        
        mem_types = ["WRAM", "VRAM", "CGRAM", "ROM"]
        if self.memory_bank == 0xFF:
            # CPU register view - start CPU state capture
            if not self.cpu_capture_enabled:
                # Start capturing CPU states
                self.cpu_capture_enabled = True
                self.cpu_capture_buffer = []
                emu = config.emulator
                self.cpu_capture_start_cycles = emu.cpu.cycles
                logger.info(f"Started CPU state capture: will capture {self.cpu_capture_count} cycles")
                return
            else:
                # Already capturing - save what we have so far
                logger.info("CPU capture already in progress, saving current buffer...")
                self._save_cpu_capture()
                return
        
        # Regular memory dump
        mem_type = mem_types[min(self.memory_bank, len(mem_types)-1)]
        default_name = f"memory_dump_bank{self.memory_bank:02X}_{mem_type}.txt"
        
        if not TKINTER_AVAILABLE:
            logger.warning("tkinter not available, saving to default location")
            logger.info(f"Using fallback filename: {default_name}")
            self._do_dump_save(default_name, mem_type)
        else:
            # Run file dialog in a separate thread to avoid blocking pygame
            thread = threading.Thread(
                target=self._dump_to_file_threaded,
                args=(default_name, mem_type),
                daemon=True
            )
            thread.start()
            logger.info("File dialog thread started (non-blocking)")
    
    def capture_cpu_state(self):
        """Capture current CPU state (called from cpu_execute_instruction after each instruction)"""
        if not self.cpu_capture_enabled:
            return
        
        emu = config.emulator
        
        # Check if we've captured enough states
        if len(self.cpu_capture_buffer) >= self.cpu_capture_count:
            # Capture complete, save to file
            self.cpu_capture_enabled = False  # Stop capturing
            self._save_cpu_capture()
            return
        
        # Capture CPU state snapshot (after instruction execution)
        state = {
            'cycles': emu.cpu.cycles,
            'r0': emu.cpu.r0, 'r1': emu.cpu.r1, 'r2': emu.cpu.r2, 'r3': emu.cpu.r3,
            'r4': emu.cpu.r4, 'r5': emu.cpu.r5, 'r6': emu.cpu.r6, 'r7': emu.cpu.r7,
            'pc_bank': emu.cpu.pc_bank, 'pc_offset': emu.cpu.pc_offset,
            'sp': emu.cpu.sp, 'pbr': emu.cpu.pbr, 'dbr': emu.cpu.dbr,
            'flags': emu.cpu.flags, 'paused': emu.paused, 'step_mode': emu.step_mode
        }
        self.cpu_capture_buffer.append(state)
    
    def _save_cpu_capture(self):
        """Save captured CPU states to file"""
        if not self.cpu_capture_buffer:
            logger.warning("No CPU states captured to save")
            self.cpu_capture_enabled = False
            return
        
        import time
        timestamp = time.strftime("%Y%m%d_%H%M%S")
        default_name = f"cpu_state_capture_{timestamp}.txt"
        
        if not TKINTER_AVAILABLE:
            logger.warning("tkinter not available, saving to default location")
            logger.info(f"Using fallback filename: {default_name}")
            self._do_save_cpu_capture(default_name)
        else:
            # Run file dialog in a separate thread
            thread = threading.Thread(
                target=self._save_cpu_capture_threaded,
                args=(default_name,),
                daemon=True
            )
            thread.start()
            logger.info("CPU capture save dialog thread started (non-blocking)")
    
    def _save_cpu_capture_threaded(self, default_name):
        """Internal method to show file dialog for CPU capture in a separate thread"""
        try:
            logger.info("Opening file save dialog for CPU capture...")
            root = tk.Tk()
            root.withdraw()
            try:
                root.attributes('-topmost', True)
            except:
                pass
            root.update()
            root.lift()
            try:
                root.focus_force()
            except:
                pass
            
            filename = filedialog.asksaveasfilename(
                title="Save CPU State Capture",
                defaultextension=".txt",
                filetypes=[("Text files", "*.txt"), ("All files", "*.*")],
                initialfile=default_name,
                parent=root
            )
            
            root.quit()
            root.destroy()
            
            if filename:
                self._do_save_cpu_capture(filename)
            else:
                logger.info("CPU capture save cancelled by user")
        except Exception as e:
            logger.error(f"Error in CPU capture save dialog thread: {e}")
            import traceback
            traceback.print_exc()
            # Fallback: save to default location
            self._do_save_cpu_capture(default_name)
    
    def _do_save_cpu_capture(self, filename):
        """Actually save CPU capture to file"""
        try:
            with open(filename, 'w') as f:
                f.write("CPU State Capture\n")
                f.write("=" * 80 + "\n")
                f.write(f"Captured {len(self.cpu_capture_buffer)} CPU states\n")
                f.write(f"Capture started at cycle: {self.cpu_capture_start_cycles}\n")
                f.write("=" * 80 + "\n\n")
                
                for i, state in enumerate(self.cpu_capture_buffer):
                    f.write(f"State {i+1} (Cycle {state['cycles']}):\n")
                    f.write(f"  R0={state['r0']:04X}  R1={state['r1']:04X}  R2={state['r2']:04X}  R3={state['r3']:04X}\n")
                    f.write(f"  R4={state['r4']:04X}  R5={state['r5']:04X}  R6={state['r6']:04X}  R7={state['r7']:04X}\n")
                    f.write(f"  PC={state['pc_bank']:02X}:{state['pc_offset']:04X}  SP={state['sp']:04X}\n")
                    f.write(f"  PBR={state['pbr']:02X}  DBR={state['dbr']:02X}  Flags={state['flags']:04X}\n")
                    f.write(f"  Paused={state['paused']}  Step={state['step_mode']}\n")
                    f.write("\n")
            
            logger.info(f"CPU state capture saved to {filename} ({len(self.cpu_capture_buffer)} states)")
            self.cpu_capture_enabled = False
            self.cpu_capture_buffer = []
        except Exception as e:
            logger.error(f"Failed to save CPU capture: {e}")
            import traceback
            traceback.print_exc()


class SettingsWindow:
    """Settings window"""
    def __init__(self, x, y):
        self.x = x
        self.y = y
        self.width = SETTINGS_WINDOW_WIDTH
        self.height = SETTINGS_WINDOW_HEIGHT
        self.visible = False
        self.render_scale = 2  # Default 2x scale
        self.dragging = False  # Whether window is being dragged
        self.drag_offset = (0, 0)  # Offset from mouse to window corner when dragging
        self.z_order = get_next_z_order()  # Z-order for window stacking
        # Logging settings
        self.detailed_logging = True  # Match logger's default (enabled by default)
        self.log_cpu = False
        self.log_memory = False
        self.log_ppu = False
        self.log_rom = False
        
    def draw(self, surface, font):
        """Draw settings window"""
        if not self.visible:
            return
        
        # Draw window background
        pygame.draw.rect(surface, WINDOW_BG_COLOR, 
                        (self.x, self.y, self.width, self.height))
        pygame.draw.rect(surface, MENU_TEXT_COLOR, 
                        (self.x, self.y, self.width, self.height), 2)
        
        # Draw title bar (draggable area)
        title_bar_height = 25
        pygame.draw.rect(surface, (60, 60, 60), 
                        (self.x, self.y, self.width, title_bar_height))
        pygame.draw.line(surface, MENU_TEXT_COLOR, 
                        (self.x, self.y + title_bar_height), 
                        (self.x + self.width, self.y + title_bar_height), 1)
        
        # Draw title
        title = font.render("Settings", True, MENU_TEXT_COLOR)
        surface.blit(title, (self.x + 10, self.y + 5))
        
        # Draw close button (X)
        close_button_size = 20
        close_button_rect = pygame.Rect(self.x + self.width - close_button_size - 5, 
                                        self.y + 2, close_button_size, close_button_size)
        pygame.draw.rect(surface, (200, 50, 50), close_button_rect)
        pygame.draw.rect(surface, MENU_TEXT_COLOR, close_button_rect, 1)
        # Draw X
        close_x = font.render("×", True, MENU_TEXT_COLOR)
        surface.blit(close_x, (close_button_rect.centerx - close_x.get_width() // 2,
                               close_button_rect.centery - close_x.get_height() // 2))
        
        # Draw render scale setting (offset by title bar)
        y_offset = 50
        scale_text = font.render(f"Render Scale: {self.render_scale}x", True, MENU_TEXT_COLOR)
        surface.blit(scale_text, (self.x + 10, self.y + y_offset))
        
        # Draw scale buttons
        button_y = self.y + y_offset + 25
        for scale in [1, 2, 3, 4]:
            button_rect = pygame.Rect(self.x + 10 + (scale - 1) * 60, button_y, 50, 25)
            color = MENU_HOVER_COLOR if self.render_scale == scale else MENU_BG_COLOR
            pygame.draw.rect(surface, color, button_rect)
            pygame.draw.rect(surface, MENU_TEXT_COLOR, button_rect, 1)
            scale_label = font.render(f"{scale}x", True, MENU_TEXT_COLOR)
            label_rect = scale_label.get_rect(center=button_rect.center)
            surface.blit(scale_label, label_rect)
        
        # Draw logging settings
        y_offset = 120
        logging_text = font.render("Logging:", True, MENU_TEXT_COLOR)
        surface.blit(logging_text, (self.x + 10, self.y + y_offset))
        
        # Disable All Logging toggle (at the top, most prominent)
        y_offset += 25
        disable_text = font.render("Disable All Logging", True, MENU_TEXT_COLOR)
        surface.blit(disable_text, (self.x + 10, self.y + y_offset))
        disable_button = pygame.Rect(self.x + 150, self.y + y_offset, 60, 20)
        color = MENU_HOVER_COLOR if not logger.enabled else MENU_BG_COLOR
        pygame.draw.rect(surface, color, disable_button)
        pygame.draw.rect(surface, MENU_TEXT_COLOR, disable_button, 1)
        disable_label = font.render("ON" if not logger.enabled else "OFF", True, MENU_TEXT_COLOR)
        label_rect = disable_label.get_rect(center=disable_button.center)
        surface.blit(disable_label, label_rect)
        
        # Detailed logging toggle (only show if logging is enabled)
        if logger.enabled:
            y_offset += 25
            detail_text = font.render("Detailed Logging", True, MENU_TEXT_COLOR)
            surface.blit(detail_text, (self.x + 10, self.y + y_offset))
            detail_button = pygame.Rect(self.x + 150, self.y + y_offset, 60, 20)
            color = MENU_HOVER_COLOR if self.detailed_logging else MENU_BG_COLOR
            pygame.draw.rect(surface, color, detail_button)
            pygame.draw.rect(surface, MENU_TEXT_COLOR, detail_button, 1)
            detail_label = font.render("ON" if self.detailed_logging else "OFF", True, MENU_TEXT_COLOR)
            label_rect = detail_label.get_rect(center=detail_button.center)
            surface.blit(detail_label, label_rect)
        
        # Category toggles (only show if logging is enabled and detailed logging is off)
        if logger.enabled and not self.detailed_logging:
            y_offset += 30
            cpu_text = font.render("CPU", True, MENU_TEXT_COLOR)
            surface.blit(cpu_text, (self.x + 10, self.y + y_offset))
            cpu_button = pygame.Rect(self.x + 60, self.y + y_offset, 40, 20)
            color = MENU_HOVER_COLOR if self.log_cpu else MENU_BG_COLOR
            pygame.draw.rect(surface, color, cpu_button)
            pygame.draw.rect(surface, MENU_TEXT_COLOR, cpu_button, 1)
            cpu_label = font.render("ON" if self.log_cpu else "OFF", True, MENU_TEXT_COLOR)
            label_rect = cpu_label.get_rect(center=cpu_button.center)
            surface.blit(cpu_label, label_rect)
            
            mem_text = font.render("Memory", True, MENU_TEXT_COLOR)
            surface.blit(mem_text, (self.x + 110, self.y + y_offset))
            mem_button = pygame.Rect(self.x + 170, self.y + y_offset, 40, 20)
            color = MENU_HOVER_COLOR if self.log_memory else MENU_BG_COLOR
            pygame.draw.rect(surface, color, mem_button)
            pygame.draw.rect(surface, MENU_TEXT_COLOR, mem_button, 1)
            mem_label = font.render("ON" if self.log_memory else "OFF", True, MENU_TEXT_COLOR)
            label_rect = mem_label.get_rect(center=mem_button.center)
            surface.blit(mem_label, label_rect)
            
            y_offset += 25
            ppu_text = font.render("PPU", True, MENU_TEXT_COLOR)
            surface.blit(ppu_text, (self.x + 10, self.y + y_offset))
            ppu_button = pygame.Rect(self.x + 60, self.y + y_offset, 40, 20)
            color = MENU_HOVER_COLOR if self.log_ppu else MENU_BG_COLOR
            pygame.draw.rect(surface, color, ppu_button)
            pygame.draw.rect(surface, MENU_TEXT_COLOR, ppu_button, 1)
            ppu_label = font.render("ON" if self.log_ppu else "OFF", True, MENU_TEXT_COLOR)
            label_rect = ppu_label.get_rect(center=ppu_button.center)
            surface.blit(ppu_label, label_rect)
            
            rom_text = font.render("ROM", True, MENU_TEXT_COLOR)
            surface.blit(rom_text, (self.x + 110, self.y + y_offset))
            rom_button = pygame.Rect(self.x + 170, self.y + y_offset, 40, 20)
            color = MENU_HOVER_COLOR if self.log_rom else MENU_BG_COLOR
            pygame.draw.rect(surface, color, rom_button)
            pygame.draw.rect(surface, MENU_TEXT_COLOR, rom_button, 1)
            rom_label = font.render("ON" if self.log_rom else "OFF", True, MENU_TEXT_COLOR)
            label_rect = rom_label.get_rect(center=rom_button.center)
            surface.blit(rom_label, label_rect)
        
        # Save log button
        y_offset += 40
        save_log_button = pygame.Rect(self.x + 10, self.y + y_offset, self.width - 20, 30)
        pygame.draw.rect(surface, MENU_HOVER_COLOR, save_log_button)
        pygame.draw.rect(surface, MENU_TEXT_COLOR, save_log_button, 1)
        save_log_text = font.render("Save Log to File", True, MENU_TEXT_COLOR)
        label_rect = save_log_text.get_rect(center=save_log_button.center)
        surface.blit(save_log_text, label_rect)
        
        # Show log count
        y_offset += 35
        log_count_text = font.render(f"Messages in buffer: {len(logger.messages)}", True, MENU_TEXT_COLOR)
        surface.blit(log_count_text, (self.x + 10, self.y + y_offset))
    
    def handle_click(self, pos, button=1):
        """Handle click on settings window"""
        if not self.visible:
            return False
        
        x, y = pos
        if not (self.x <= x < self.x + self.width and self.y <= y < self.y + self.height):
            return False
        
        # Check close button
        close_button_size = 20
        close_button_rect = pygame.Rect(self.x + self.width - close_button_size - 5,
                                        self.y + 2, close_button_size, close_button_size)
        if close_button_rect.collidepoint(pos):
            self.visible = False
            logger.info("Settings window closed")
            return True
        
        # Check title bar for dragging
        title_bar_height = 25
        title_bar_rect = pygame.Rect(self.x, self.y, self.width, title_bar_height)
        if title_bar_rect.collidepoint(pos) and button == 1:
            # Bring window to front when clicked
            self.z_order = get_next_z_order()
            # Start dragging
            self.dragging = True
            self.drag_offset = (x - self.x, y - self.y)
            return True
        
        # Bring window to front when any part is clicked (before processing other clicks)
        self.z_order = get_next_z_order()
        
        # Check scale buttons
        button_y = self.y + 75
        for scale in [1, 2, 3, 4]:
            button_rect = pygame.Rect(self.x + 10 + (scale - 1) * 60, button_y, 50, 25)
            if button_rect.collidepoint(pos):
                self.render_scale = scale
                logger.info(f"Render scale set to {scale}x")
                return True
        
        # Check disable all logging toggle
        y_offset = 145
        disable_button = pygame.Rect(self.x + 150, self.y + y_offset, 60, 20)
        if disable_button.collidepoint(pos):
            logger.enabled = not logger.enabled
            # When disabling, also turn off detailed logging
            if not logger.enabled:
                self.detailed_logging = False  # Match logger's default (disabled by default)
                logger.set_detailed_logging(False)
            # Use print instead of logger since we might be disabling logging
            print(f"All logging: {'DISABLED' if not logger.enabled else 'ENABLED'}")
            return True
        
        # Check detailed logging toggle (only if logging is enabled)
        if logger.enabled:
            y_offset = 170
            detail_button = pygame.Rect(self.x + 150, self.y + y_offset, 60, 20)
            if detail_button.collidepoint(pos):
                self.detailed_logging = not self.detailed_logging
                logger.set_detailed_logging(self.detailed_logging)
                logger.info(f"Detailed logging: {'ON' if self.detailed_logging else 'OFF'}")
                return True
        
        # Check category toggles (only if logging is enabled and detailed logging is off)
        if logger.enabled and not self.detailed_logging:
            y_offset = 175
            # CPU toggle
            cpu_button = pygame.Rect(self.x + 60, self.y + y_offset, 40, 20)
            if cpu_button.collidepoint(pos):
                self.log_cpu = not self.log_cpu
                logger.set_category_logging(cpu=self.log_cpu, memory=self.log_memory, 
                                          ppu=self.log_ppu, rom=self.log_rom)
                logger.info(f"CPU logging: {'ON' if self.log_cpu else 'OFF'}")
                return True
            
            # Memory toggle
            mem_button = pygame.Rect(self.x + 170, self.y + y_offset, 40, 20)
            if mem_button.collidepoint(pos):
                self.log_memory = not self.log_memory
                logger.set_category_logging(cpu=self.log_cpu, memory=self.log_memory, 
                                          ppu=self.log_ppu, rom=self.log_rom)
                logger.info(f"Memory logging: {'ON' if self.log_memory else 'OFF'}")
                return True
            
            y_offset = 200
            # PPU toggle
            ppu_button = pygame.Rect(self.x + 60, self.y + y_offset, 40, 20)
            if ppu_button.collidepoint(pos):
                self.log_ppu = not self.log_ppu
                logger.set_category_logging(cpu=self.log_cpu, memory=self.log_memory, 
                                          ppu=self.log_ppu, rom=self.log_rom)
                logger.info(f"PPU logging: {'ON' if self.log_ppu else 'OFF'}")
                return True
            
            # ROM toggle
            rom_button = pygame.Rect(self.x + 170, self.y + y_offset, 40, 20)
            if rom_button.collidepoint(pos):
                self.log_rom = not self.log_rom
                logger.set_category_logging(cpu=self.log_cpu, memory=self.log_memory, 
                                          ppu=self.log_ppu, rom=self.log_rom)
                logger.info(f"ROM logging: {'ON' if self.log_rom else 'OFF'}")
                return True
        
        # Check save log button
        # Calculate button position to match draw() logic
        if not logger.enabled:
            # If logging is disabled, button is after disable toggle (y_offset 145) + 40
            y_offset = 185
        elif not self.detailed_logging:
            # Category toggles are at y_offset 200, button is 40 pixels below
            y_offset = 240
        else:
            # Detailed logging toggle is at y_offset 170, button is 40 pixels below
            y_offset = 210
        save_log_button = pygame.Rect(self.x + 10, self.y + y_offset, self.width - 20, 30)
        if save_log_button.collidepoint(pos):
            # Save log using file dialog
            self._save_log_with_dialog()
            return True
        
        return False
    
    def _save_log_with_dialog(self):
        """Save log file using file dialog (non-blocking)"""
        if not TKINTER_AVAILABLE:
            # Fallback: save to default location
            import time
            timestamp = time.strftime("%Y%m%d_%H%M%S")
            filename = f"emulator_log_{timestamp}.txt"
            saved_file = logger.save_to_file(filename)
            if saved_file:
                logger.info(f"Log saved to: {saved_file}")
            return
        
        # Use threaded file dialog
        thread = threading.Thread(target=self._save_log_dialog_threaded, daemon=True)
        thread.start()
        logger.info("Save log dialog thread started (non-blocking)")
    
    def _save_log_dialog_threaded(self):
        """Internal method to show save dialog in a separate thread"""
        try:
            import time
            timestamp = time.strftime("%Y%m%d_%H%M%S")
            default_name = f"emulator_log_{timestamp}.txt"
            
            logger.info("Opening file save dialog for log...")
            # Create tkinter root in this thread
            root = tk.Tk()
            root.withdraw()
            try:
                root.attributes('-topmost', True)
            except:
                pass
            root.update()
            root.lift()
            try:
                root.focus_force()
            except:
                pass
            
            filename = filedialog.asksaveasfilename(
                title="Save Log File",
                defaultextension=".txt",
                filetypes=[("Text files", "*.txt"), ("All files", "*.*")],
                initialfile=default_name,
                parent=root
            )
            
            root.quit()
            root.destroy()
            
            if filename:
                saved_file = logger.save_to_file(filename)
                if saved_file:
                    logger.info(f"Log saved to: {saved_file}")
            else:
                logger.info("Log save cancelled by user")
        except Exception as e:
            logger.error(f"Error in save log dialog thread: {e}")
            import traceback
            traceback.print_exc()
            # Fallback: save to default location
            import time
            timestamp = time.strftime("%Y%m%d_%H%M%S")
            default_name = f"emulator_log_{timestamp}.txt"
            saved_file = logger.save_to_file(default_name)
            if saved_file:
                logger.info(f"Log saved to default location: {saved_file}")
    
    def handle_mouse_up(self, pos):
        """Handle mouse button release"""
        if self.dragging:
            self.dragging = False
    
    def handle_mouse_move(self, pos):
        """Handle mouse movement (for dragging)"""
        if self.dragging:
            self.x = pos[0] - self.drag_offset[0]
            self.y = pos[1] - self.drag_offset[1]
            # Keep window on screen
            screen_width = pygame.display.get_surface().get_width()
            screen_height = pygame.display.get_surface().get_height()
            self.x = max(0, min(self.x, screen_width - self.width))
            # Keep window above status bar (status bar is 25 pixels tall)
            STATUS_BAR_HEIGHT = 25
            self.y = max(MENU_BAR_HEIGHT, min(self.y, screen_height - self.height - STATUS_BAR_HEIGHT))


# Global variable to store file dialog result (thread-safe with lock)
_file_dialog_result = None
_file_dialog_lock = threading.Lock()
_file_dialog_ready = threading.Event()

def _open_file_dialog_threaded():
    """Internal function to show file dialog in a separate thread"""
    global _file_dialog_result
    try:
        logger.info("Opening file dialog in thread...")
        # Create tkinter root in this thread (tkinter requires root in same thread)
        root = tk.Tk()
        root.withdraw()
        try:
            root.attributes('-topmost', True)
        except:
            pass
        root.update()
        root.lift()
        try:
            root.focus_force()
        except:
            pass
        
        logger.info("Showing file open dialog...")
        filename = filedialog.askopenfilename(
            title="Open ROM File",
            filetypes=[("ROM files", "*.rom"), ("All files", "*.*")],
            parent=root
        )
        
        root.quit()  # Quit the tkinter mainloop
        root.destroy()
        
        with _file_dialog_lock:
            _file_dialog_result = filename if filename else None
        _file_dialog_ready.set()
        
        if filename:
            logger.info(f"User selected file: {filename}")
        else:
            logger.info("File dialog cancelled by user")
    except Exception as e:
        logger.error(f"Error in file dialog thread: {e}")
        import traceback
        traceback.print_exc()
        with _file_dialog_lock:
            _file_dialog_result = None
        _file_dialog_ready.set()

def open_file_dialog():
    """Open a file dialog to select a ROM file (non-blocking)"""
    global _file_dialog_result, _file_dialog_ready
    logger.info("Open file dialog requested")
    
    if not TKINTER_AVAILABLE:
        logger.error("tkinter not available, cannot show file dialog")
        logger.error("Please install tkinter: sudo apt-get install python3-tk (Linux)")
        return None
    
    # Reset the result and event
    with _file_dialog_lock:
        _file_dialog_result = None
    _file_dialog_ready.clear()
    
    # Start dialog in a separate thread
    # Note: tkinter requires the root to be created in the same thread that uses it
    thread = threading.Thread(target=_open_file_dialog_threaded, daemon=True)
    thread.start()
    logger.info("File dialog thread started (non-blocking)")
    
    # Return None immediately - caller should poll check_file_dialog_result()
    return None

def check_file_dialog_result():
    """Check if file dialog has completed (non-blocking) - returns filename or None if still waiting"""
    global _file_dialog_result
    if _file_dialog_ready.is_set():
        with _file_dialog_lock:
            result = _file_dialog_result
            _file_dialog_result = None  # Clear after reading
            _file_dialog_ready.clear()
            return result
    return None  # None means still waiting (empty string would mean cancelled, but we use None for that too)


def draw_menu_dropdown(surface, font, menu_bar, mouse_pos):
    """Draw active menu dropdown"""
    if not menu_bar.active_menu:
        return
    
    items = menu_bar.get_active_menu_items()
    if not items:
        return
    
    # Calculate dropdown position
    menu_x = MENU_ITEM_PADDING
    for menu_name in menu_bar.menus.keys():
        if menu_name == menu_bar.active_menu:
            break
        text = font.render(menu_name, True, MENU_TEXT_COLOR)
        menu_x += text.get_width() + MENU_ITEM_PADDING * 2
    
    dropdown_y = MENU_BAR_HEIGHT
    item_height = 25
    dropdown_width = 200
    dropdown_height = len(items) * item_height
    
    # Draw dropdown background
    pygame.draw.rect(surface, MENU_BG_COLOR, 
                    (menu_x, dropdown_y, dropdown_width, dropdown_height))
    pygame.draw.rect(surface, MENU_TEXT_COLOR, 
                    (menu_x, dropdown_y, dropdown_width, dropdown_height), 1)
    
    # Draw menu items
    for i, item in enumerate(items):
        item_y = dropdown_y + i * item_height
        item_rect = pygame.Rect(menu_x, item_y, dropdown_width, item_height)
        
        # Highlight on hover
        if item_rect.collidepoint(mouse_pos):
            pygame.draw.rect(surface, MENU_HOVER_COLOR, item_rect)
        
        # Draw item text
        text = font.render(item, True, MENU_TEXT_COLOR)
        surface.blit(text, (menu_x + 5, item_y + 5))
    
    return menu_x, dropdown_y, dropdown_width, dropdown_height


def handle_menu_click(menu_bar, pos, font):
    """Handle click on menu dropdown"""
    if not menu_bar.active_menu:
        return None
    
    items = menu_bar.get_active_menu_items()
    if not items:
        return None
    
    # Calculate dropdown position
    menu_x = MENU_ITEM_PADDING
    for menu_name in menu_bar.menus.keys():
        if menu_name == menu_bar.active_menu:
            break
        text = font.render(menu_name, True, MENU_TEXT_COLOR)
        menu_x += text.get_width() + MENU_ITEM_PADDING * 2
    
    dropdown_y = MENU_BAR_HEIGHT
    item_height = 25
    dropdown_width = 200
    
    x, y = pos
    if menu_x <= x < menu_x + dropdown_width and dropdown_y <= y < dropdown_y + len(items) * item_height:
        item_index = (y - dropdown_y) // item_height
        if 0 <= item_index < len(items):
            return items[item_index]
    
    return None

