"""
config.py - Configuration constants and shared type definitions
Python version - maintains BASIC-like simplicity
"""

# System Constants
FALSE = 0
TRUE = 1

# Display constants
DISPLAY_WIDTH = 320
DISPLAY_HEIGHT = 200
DISPLAY_WIDTH_PORTRAIT = 200
DISPLAY_HEIGHT_PORTRAIT = 320
TARGET_FPS = 60

# Memory constants
MEMORY_BANK_SIZE = 65536  # 64KB per bank
MEMORY_MAX_BANKS = 256  # 16MB total (256 * 64KB)
MEMORY_WRAM_BANK = 0
MEMORY_IO_BANK = 0
MEMORY_ROM_START_BANK = 1
MEMORY_ROM_END_BANK = 125  # Banks 1-125 for ROM (LoROM-like)
MEMORY_WRAM_EXTENDED_START = 126  # Banks 126-127 for extended WRAM

# CPU constants
# Target: 10-12 MHz CPU (Nitro-Core-DX spec)
# 10 MHz @ 60 FPS = 166,667 cycles/frame
# 12 MHz @ 60 FPS = 200,000 cycles/frame
# Using 10 MHz for now (can be increased to 12 MHz later)
CPU_CYCLES_PER_FRAME = 166667  # Cycles per frame at 60 FPS (10 MHz CPU)
CPU_STACK_BASE = 0x1FFF  # Stack starts at top of WRAM bank 0

# PPU constants
PPU_TILE_SIZE_8X8 = 8
PPU_TILE_SIZE_16X16 = 16
PPU_MAX_SPRITES = 128
PPU_PALETTE_SIZE = 256
PPU_VRAM_SIZE = 65536  # 64KB VRAM
PPU_CGRAM_SIZE = 512  # 256 colors * 2 bytes (RGB555)

# APU constants
APU_SAMPLE_RATE = 44100
APU_NUM_CHANNELS = 4
APU_BUFFER_SIZE = 4096  # Audio buffer size in samples
APU_VOLUME_MAX = 255

# ROM constants
ROM_MAGIC = 0x46434D52  # "RMCF" (Fantasy Console ROM)
ROM_HEADER_SIZE = 32
ROM_VERSION = 1

# Input constants
INPUT_BUTTON_UP = 0x01
INPUT_BUTTON_DOWN = 0x02
INPUT_BUTTON_LEFT = 0x04
INPUT_BUTTON_RIGHT = 0x08
INPUT_BUTTON_A = 0x10
INPUT_BUTTON_B = 0x20
INPUT_BUTTON_X = 0x40
INPUT_BUTTON_Y = 0x80
INPUT_BUTTON_L = 0x100
INPUT_BUTTON_R = 0x200
INPUT_BUTTON_START = 0x400
INPUT_BUTTON_SELECT = 0x800

# Memory Mapper Types
MAPPER_LOROM = 0  # SNES-like LoROM mapping
MAPPER_HIROM = 1  # SNES-like HiROM mapping (future)
MAPPER_LARGE = 2  # Extended mapper for large ROMs

# Memory I/O Register Addresses
MEM_IO_PPU_BASE = 0x8000  # PPU registers start here
MEM_IO_APU_BASE = 0x9000  # APU registers start here
MEM_IO_INPUT_BASE = 0xA000  # Input controller registers
MEM_IO_TIMER_BASE = 0xB000  # Timer registers (future)

# CPU Constants
# Instruction Opcodes
OP_NOP = 0x0000
OP_MOV = 0x1000  # Move/load/store family
OP_ADD = 0x2000  # Arithmetic operations
OP_SUB = 0x3000
OP_MUL = 0x4000
OP_DIV = 0x5000
OP_AND = 0x6000  # Logical operations
OP_OR = 0x7000
OP_XOR = 0x8000
OP_NOT = 0x9000
OP_SHL = 0xA000  # Shift operations
OP_SHR = 0xB000
OP_CMP = 0xC000  # Compare
OP_BEQ = 0xC100  # Branch if equal (Z flag set)
OP_BNE = 0xC200  # Branch if not equal (Z flag clear)
OP_BGT = 0xC300  # Branch if greater than (Z=0 and N=0)
OP_BLT = 0xC400  # Branch if less than (N flag set)
OP_BGE = 0xC500  # Branch if greater or equal (N=0)
OP_BLE = 0xC600  # Branch if less or equal (Z=1 or N=1)
OP_JMP = 0xD000  # Jump/Branch
OP_CALL = 0xE000  # Subroutine call
OP_RET = 0xF000  # Return
OP_PUSH = 0x10000  # Stack operations
OP_POP = 0x11000
OP_INT = 0x12000  # Interrupt operations
OP_RTI = 0x13000  # Return from interrupt

# Addressing Modes
ADDR_REGISTER = 0  # Register direct
ADDR_IMMEDIATE = 1  # Immediate value
ADDR_DIRECT = 2  # Direct address (bank:offset)
ADDR_INDIRECT = 3  # Indirect via register
ADDR_INDEXED = 4  # Register + offset

# Interrupt Types
INT_NONE = 0
INT_VBLANK = 1
INT_TIMER = 2
INT_NMI = 3

# PPU Constants
# PPU Register Addresses (relative to MEM_IO_PPU_BASE)
PPU_REG_BG0_SCROLLX = 0x00  # BG0 scroll X (16-bit)
PPU_REG_BG0_SCROLLY = 0x02  # BG0 scroll Y (16-bit)
PPU_REG_BG1_SCROLLX = 0x04  # BG1 scroll X (16-bit)
PPU_REG_BG1_SCROLLY = 0x06  # BG1 scroll Y (16-bit)
PPU_REG_BG0_CONTROL = 0x08  # BG0 control (tile size, enable, etc.)
PPU_REG_BG1_CONTROL = 0x09  # BG1 control
PPU_REG_BG2_SCROLLX = 0x0A  # BG2 scroll X (16-bit, low byte)
PPU_REG_BG2_SCROLLX_H = 0x0B  # BG2 scroll X (high byte)
PPU_REG_BG2_SCROLLY = 0x0C  # BG2 scroll Y (16-bit, low byte)
PPU_REG_BG2_SCROLLY_H = 0x0D  # BG2 scroll Y (high byte)
PPU_REG_BG2_CONTROL = 0x21  # BG2 control (after Matrix registers)
PPU_REG_BG3_SCROLLX = 0x22  # BG3 scroll X (16-bit, low byte)
PPU_REG_BG3_SCROLLX_H = 0x23  # BG3 scroll X (high byte)
PPU_REG_BG3_SCROLLY = 0x24  # BG3 scroll Y (16-bit, low byte)
PPU_REG_BG3_SCROLLY_H = 0x25  # BG3 scroll Y (high byte)
PPU_REG_BG3_CONTROL = 0x26  # BG3 control (can be used as dedicated affine layer)
PPU_REG_VRAM_ADDR = 0x0E  # VRAM address (16-bit, low byte)
PPU_REG_VRAM_ADDR_H = 0x0F  # VRAM address (high byte)
PPU_REG_VRAM_DATA = 0x10  # VRAM data (8-bit)
PPU_REG_CGRAM_ADDR = 0x0E  # CGRAM address (9-bit, but 8-bit register)
PPU_REG_CGRAM_DATA = 0x13  # CGRAM data (16-bit RGB555)
PPU_REG_OAM_ADDR = 0x14  # OAM address (8-bit)
PPU_REG_OAM_DATA = 0x15  # OAM data (multiple bytes per sprite)
PPU_REG_FRAMEBUFFER_ENABLE = 0x16  # Framebuffer enable
PPU_REG_DISPLAY_MODE = 0x17  # Display mode (portrait/landscape)
# Matrix Mode registers (90's retro-futuristic perspective/rotation effects)
PPU_REG_MATRIX_CONTROL = 0x18  # Matrix Mode control (bit 0=enable, bit 1=mirror_h, bit 2=mirror_v)
PPU_REG_MATRIX_A = 0x19  # Matrix A (16-bit, low byte)
PPU_REG_MATRIX_A_H = 0x1A  # Matrix A (high byte)
PPU_REG_MATRIX_B = 0x1B  # Matrix B (16-bit, low byte)
PPU_REG_MATRIX_B_H = 0x1C  # Matrix B (high byte)
PPU_REG_MATRIX_C = 0x1D  # Matrix C (16-bit, low byte)
PPU_REG_MATRIX_C_H = 0x1E  # Matrix C (high byte)
PPU_REG_MATRIX_D = 0x1F  # Matrix D (16-bit, low byte)
PPU_REG_MATRIX_D_H = 0x20  # Matrix D (high byte)
PPU_REG_MATRIX_CENTER_X = 0x27  # Center X (16-bit, low byte)
PPU_REG_MATRIX_CENTER_X_H = 0x28  # Center X (high byte)
PPU_REG_MATRIX_CENTER_Y = 0x29  # Center Y (16-bit, low byte)
PPU_REG_MATRIX_CENTER_Y_H = 0x2A  # Center Y (high byte)
# Windowing system registers
PPU_REG_WINDOW0_LEFT = 0x2B  # Window 0 left edge (8-bit)
PPU_REG_WINDOW0_RIGHT = 0x2C  # Window 0 right edge (8-bit)
PPU_REG_WINDOW0_TOP = 0x2D  # Window 0 top edge (8-bit)
PPU_REG_WINDOW0_BOTTOM = 0x2E  # Window 0 bottom edge (8-bit)
PPU_REG_WINDOW1_LEFT = 0x2F  # Window 1 left edge (8-bit)
PPU_REG_WINDOW1_RIGHT = 0x30  # Window 1 right edge (8-bit)
PPU_REG_WINDOW1_TOP = 0x31  # Window 1 top edge (8-bit)
PPU_REG_WINDOW1_BOTTOM = 0x32  # Window 1 bottom edge (8-bit)
PPU_REG_WINDOW_CONTROL = 0x33  # Window control: bit 0=Window0 enable, bit 1=Window1 enable, bits 2-3=logic (0=OR, 1=AND, 2=XOR, 3=XNOR)
PPU_REG_WINDOW_MAIN_ENABLE = 0x34  # Main window enable per layer: bit 0=BG0, 1=BG1, 2=BG2, 3=BG3, 4=sprites
PPU_REG_WINDOW_SUB_ENABLE = 0x35  # Sub window enable (for color math, future use)
# HDMA (per-scanline scroll) registers
PPU_REG_HDMA_CONTROL = 0x36  # HDMA control: bit 0=enable, bits 1-3=layer enable (bit 1=BG0, 2=BG1, 3=BG2, 4=BG3)
PPU_REG_HDMA_TABLE_BASE_L = 0x37  # HDMA table base address (low byte, in WRAM)
PPU_REG_HDMA_TABLE_BASE_H = 0x38  # HDMA table base address (high byte)
PPU_REG_HDMA_SCANLINE = 0x39  # Current scanline for HDMA write (0-199)
PPU_REG_HDMA_BG0_SCROLLX_L = 0x3A  # HDMA: BG0 scroll X for current scanline (low byte)
PPU_REG_HDMA_BG0_SCROLLX_H = 0x3B  # HDMA: BG0 scroll X (high byte)
PPU_REG_HDMA_BG0_SCROLLY_L = 0x3C  # HDMA: BG0 scroll Y (low byte)
PPU_REG_HDMA_BG0_SCROLLY_H = 0x3D  # HDMA: BG0 scroll Y (high byte)
# Similar registers for BG1, BG2, BG3 (0x3E-0x4D)

# APU Constants
# APU Register Addresses (relative to MEM_IO_APU_BASE)
APU_REG_CH0_FREQ_LOW = 0x00  # Channel 0 frequency low byte
APU_REG_CH0_FREQ_HIGH = 0x01  # Channel 0 frequency high byte
APU_REG_CH0_VOLUME = 0x02  # Channel 0 volume (0-255)
APU_REG_CH0_CONTROL = 0x03  # Channel 0 control (enable, waveform)
APU_REG_CH1_FREQ_LOW = 0x04  # Channel 1 (same pattern)
APU_REG_CH1_FREQ_HIGH = 0x05
APU_REG_CH1_VOLUME = 0x06
APU_REG_CH1_CONTROL = 0x07
APU_REG_CH2_FREQ_LOW = 0x08  # Channel 2
APU_REG_CH2_FREQ_HIGH = 0x09
APU_REG_CH2_VOLUME = 0x0A
APU_REG_CH2_CONTROL = 0x0B
APU_REG_CH3_FREQ_LOW = 0x0C  # Channel 3 (noise/square)
APU_REG_CH3_FREQ_HIGH = 0x0D
APU_REG_CH3_VOLUME = 0x0E
APU_REG_CH3_CONTROL = 0x0F  # Bit 0=enable, bit 1=noise mode (1=noise, 0=square)
APU_REG_MASTER_VOLUME = 0x10  # Master volume (0-255)

# Waveform Types
WAVEFORM_SINE = 0
WAVEFORM_SQUARE = 1
WAVEFORM_SAW = 2
WAVEFORM_NOISE = 3

# Input Constants
# Input Register Addresses (relative to MEM_IO_INPUT_BASE)
INPUT_REG_CONTROLLER1 = 0x00  # Controller 1 data (read)
INPUT_REG_CONTROLLER1_LATCH = 0x01  # Controller 1 latch (write 1 to latch, 0 to release)


# Type Definitions (using dataclasses for clarity)
from dataclasses import dataclass
from typing import List


@dataclass
class CPUState:
    """CPU state - 16-bit CPU with banked addressing"""
    # General purpose registers (16-bit)
    r0: int = 0
    r1: int = 0
    r2: int = 0
    r3: int = 0
    r4: int = 0
    r5: int = 0
    r6: int = 0
    r7: int = 0
    
    # Program Counter (24-bit logical: bank:offset)
    pc_bank: int = 0  # Bank register for PC
    pc_offset: int = 0  # 16-bit offset within bank
    
    # Stack Pointer (16-bit offset in stack bank)
    sp: int = CPU_STACK_BASE
    
    # Bank Registers
    pbr: int = 0  # Program Bank Register
    dbr: int = 0  # Data Bank Register
    
    # Flags Register (Z, N, C, V, I)
    flags: int = 0  # Bit 0=Z, 1=N, 2=C, 3=V, 4=I
    
    # Interrupt state
    interrupt_mask: int = 0  # I flag state
    interrupt_pending: int = INT_NONE  # Pending interrupt type
    
    # Cycle counter
    cycles: int = 0


@dataclass
class MemoryState:
    """Memory state - banked 24-bit address space"""
    # Work RAM (bank 0, 0x0000-0x7FFF)
    wram: List[int] = None  # Will be initialized as [0] * 32768
    
    # Extended WRAM (banks 126-127)
    wram_extended: List[int] = None  # Will be initialized as [0] * 131072
    
    # ROM data (loaded from file)
    rom_data: bytes = b''  # Will hold ROM bytes
    rom_size: int = 0
    rom_banks: int = 0
    
    # Memory mapper state
    mapper_type: int = MAPPER_LOROM
    mapper_flags: int = 0
    
    def __post_init__(self):
        if self.wram is None:
            self.wram = [0] * 32768
        if self.wram_extended is None:
            self.wram_extended = [0] * 131072


@dataclass
class SpriteEntry:
    """Sprite entry in OAM"""
    x: int = 0  # X position (signed)
    y: int = 0  # Y position (signed)
    tile_index: int = 0  # Tile number
    palette: int = 0  # Palette index (0-15)
    priority: int = 0  # Priority (0-3): 0=lowest (behind all BGs), 3=highest (in front of all BGs)
    flip_x: bool = False  # Horizontal flip
    flip_y: bool = False  # Vertical flip
    size: int = 0  # 0=8x8, 1=16x16
    enabled: bool = False
    blend_mode: int = 0  # Blending mode: 0=normal (opaque), 1=alpha blend, 2=additive, 3=subtractive
    alpha: int = 255  # Alpha value (0-255) for blending modes


@dataclass
class TileLayer:
    """Tilemap background layer"""
    scroll_x: int = 0
    scroll_y: int = 0
    tile_size: int = PPU_TILE_SIZE_8X8  # 8 or 16
    enabled: bool = False
    tile_map_base: int = 0  # VRAM offset for tilemap
    tile_data_base: int = 0  # VRAM offset for tile data
    window_enable: int = 0  # Window enable bits: bit 0=Window0, bit 1=Window 1, bit 2=invert Window 0, bit 3=invert Window 1


@dataclass
class PPUState:
    """PPU (Picture Processing Unit) state"""
    # VRAM (tile patterns, tilemaps)
    vram: List[int] = None  # Will be [0] * PPU_VRAM_SIZE
    
    # CGRAM (palette)
    cgram: List[int] = None  # Will be [0] * PPU_CGRAM_SIZE
    
    # OAM (sprite attributes)
    oam: List[SpriteEntry] = None  # Will be [SpriteEntry()] * PPU_MAX_SPRITES
    
    # Background layers (4 layers for Nitro-Core-DX)
    bg0: TileLayer = None
    bg1: TileLayer = None
    bg2: TileLayer = None
    bg3: TileLayer = None  # Can be used as dedicated affine layer (Matrix Mode)
    
    # Framebuffer layer (optional, 8-bit indexed)
    frame_buffer: List[int] = None  # Will be [0] * (DISPLAY_WIDTH * DISPLAY_HEIGHT)
    frame_buffer_enabled: bool = False
    
    # Output buffer (RGB for display)
    output_buffer: List[int] = None  # Will be [0] * (DISPLAY_WIDTH * DISPLAY_HEIGHT)
    
    # Rotation state
    portrait_mode: bool = False  # If True, rotate output 90 degrees
    
    # VBlank state
    vblank_active: bool = False
    vblank_counter: int = 0
    
    # Matrix Mode state (90's retro-futuristic perspective/rotation effects)
    matrix_enabled: bool = False
    matrix_mirror_h: bool = False
    matrix_mirror_v: bool = False
    matrix_a: int = 0x0100  # 1.0 in 8.8 fixed point
    matrix_b: int = 0x0000
    matrix_c: int = 0x0000
    matrix_d: int = 0x0100  # 1.0 in 8.8 fixed point
    matrix_center_x: int = 0  # Center point X
    matrix_center_y: int = 0  # Center point Y
    
    # Windowing system (SNES-style: 2 windows with AND/OR/XOR logic)
    window0_left: int = 0  # Window 0 left edge (0-319)
    window0_right: int = 0  # Window 0 right edge (0-319)
    window0_top: int = 0  # Window 0 top edge (0-199)
    window0_bottom: int = 0  # Window 0 bottom edge (0-199)
    window0_enabled: bool = False  # Window 0 enable
    
    window1_left: int = 0  # Window 1 left edge
    window1_right: int = 0  # Window 1 right edge
    window1_top: int = 0  # Window 1 top edge
    window1_bottom: int = 0  # Window 1 bottom edge
    window1_enabled: bool = False  # Window 1 enable
    
    window_logic: int = 0  # Window logic: 0=OR, 1=AND, 2=XOR, 3=XNOR
    window_main_enable: int = 0  # Main window enable (bit per layer: bit 0=BG0, 1=BG1, 2=BG2, 3=BG3, 4=sprites)
    window_sub_enable: int = 0  # Sub window enable (for color math, future use)
    
    # Sprite blending and color math
    sprite_blend_enabled: bool = False  # Enable sprite blending globally
    color_math_enabled: bool = False  # Enable color math (additive/subtractive blending)
    color_math_mode: int = 0  # Color math mode: 0=additive, 1=subtractive, 2=multiply
    
    # Per-scanline scroll (HDMA-style) - allows different scroll values per scanline
    # For parallax scrolling and perspective effects
    hdma_enabled: bool = False  # Enable HDMA per-scanline scroll
    hdma_table_base: int = 0  # Base address in WRAM for HDMA table
    # HDMA table format: For each scanline (0-199), 2 bytes per layer (scroll X, scroll Y)
    # For 4 layers: 8 bytes per scanline = 1600 bytes total
    # Alternative: Store per-scanline scroll in arrays (simpler for now)
    hdma_bg0_scroll_x: List[int] = None  # Per-scanline scroll X for BG0 (200 entries)
    hdma_bg0_scroll_y: List[int] = None  # Per-scanline scroll Y for BG0
    hdma_bg1_scroll_x: List[int] = None  # Per-scanline scroll X for BG1
    hdma_bg1_scroll_y: List[int] = None  # Per-scanline scroll Y for BG1
    hdma_bg2_scroll_x: List[int] = None  # Per-scanline scroll X for BG2
    hdma_bg2_scroll_y: List[int] = None  # Per-scanline scroll Y for BG2
    hdma_bg3_scroll_x: List[int] = None  # Per-scanline scroll X for BG3
    hdma_bg3_scroll_y: List[int] = None  # Per-scanline scroll Y for BG3
    
    def __post_init__(self):
        if self.vram is None:
            self.vram = [0] * PPU_VRAM_SIZE
        if self.cgram is None:
            self.cgram = [0] * PPU_CGRAM_SIZE
        if self.oam is None:
            self.oam = [SpriteEntry() for _ in range(PPU_MAX_SPRITES)]
        if self.bg0 is None:
            self.bg0 = TileLayer()
        if self.bg1 is None:
            self.bg1 = TileLayer()
        if self.bg2 is None:
            self.bg2 = TileLayer()
        if self.bg3 is None:
            self.bg3 = TileLayer()
        if self.frame_buffer is None:
            self.frame_buffer = [0] * (DISPLAY_WIDTH * DISPLAY_HEIGHT)
        if self.output_buffer is None:
            self.output_buffer = [0] * (DISPLAY_WIDTH * DISPLAY_HEIGHT)
        # Initialize HDMA per-scanline scroll arrays
        if self.hdma_bg0_scroll_x is None:
            self.hdma_bg0_scroll_x = [0] * DISPLAY_HEIGHT
            self.hdma_bg0_scroll_y = [0] * DISPLAY_HEIGHT
            self.hdma_bg1_scroll_x = [0] * DISPLAY_HEIGHT
            self.hdma_bg1_scroll_y = [0] * DISPLAY_HEIGHT
            self.hdma_bg2_scroll_x = [0] * DISPLAY_HEIGHT
            self.hdma_bg2_scroll_y = [0] * DISPLAY_HEIGHT
            self.hdma_bg3_scroll_x = [0] * DISPLAY_HEIGHT
            self.hdma_bg3_scroll_y = [0] * DISPLAY_HEIGHT


@dataclass
class AudioChannel:
    """Audio channel state"""
    waveform_type: int = WAVEFORM_SINE  # 0=sine, 1=square, 2=saw, 3=noise
    frequency: int = 0  # Fixed-point frequency
    volume: int = 0  # 0-255
    enabled: bool = False
    phase: float = 0.0  # Current phase for oscillators
    phase_increment: float = 0.0  # Phase increment per sample
    # Noise generator state (LFSR)
    noise_lfsr: int = 0x7FFF  # Initialize LFSR to non-zero
    # ADSR (future enhancement - stubbed)
    attack_rate: int = 0
    decay_rate: int = 0
    sustain_level: int = 255
    release_rate: int = 0


@dataclass
class APUState:
    """APU (Audio Processing Unit) state"""
    channels: List[AudioChannel] = None  # Will be [AudioChannel()] * APU_NUM_CHANNELS
    sample_buffer: List[int] = None  # Will be [0] * APU_BUFFER_SIZE
    buffer_write_pos: int = 0
    buffer_read_pos: int = 0
    master_volume: int = 255  # 0-255
    
    def __post_init__(self):
        if self.channels is None:
            self.channels = [AudioChannel() for _ in range(APU_NUM_CHANNELS)]
            # Channel 3 defaults to noise
            self.channels[3].waveform_type = WAVEFORM_NOISE
        if self.sample_buffer is None:
            self.sample_buffer = [0] * APU_BUFFER_SIZE


@dataclass
class InputState:
    """Input controller state"""
    controller1: int = 0  # Bitfield of pressed buttons
    controller1_latch: int = 0  # Latched state for shift-read
    controller1_shift: int = 0  # Current shift position
    keyboard_map: List[int] = None  # Keyboard scancode -> button mapping
    
    def __post_init__(self):
        if self.keyboard_map is None:
            self.keyboard_map = [0] * 256


@dataclass
class ROMHeader:
    """ROM file header"""
    magic: int = 0  # ROM_MAGIC
    version: int = 0
    rom_size: int = 0  # Size in bytes
    entry_point_bank: int = 0
    entry_point_offset: int = 0
    mapper_flags: int = 0
    checksum: int = 0  # Optional checksum
    reserved: List[int] = None  # Padding to 32 bytes
    
    def __post_init__(self):
        if self.reserved is None:
            self.reserved = [0] * 8


@dataclass
class EmulatorState:
    """Main emulator state - holds all subsystems"""
    cpu: CPUState = None
    memory: MemoryState = None
    ppu: PPUState = None
    apu: APUState = None
    input: InputState = None
    rom: ROMHeader = None
    
    # Timing
    frame_count: int = 0
    last_frame_time: float = 0.0
    frame_time: float = 1.0 / TARGET_FPS  # Target frame time (1/60 seconds)
    
    # Debug
    debug_mode: bool = False
    paused: bool = False
    step_mode: bool = False
    
    def __post_init__(self):
        if self.cpu is None:
            self.cpu = CPUState()
        if self.memory is None:
            self.memory = MemoryState()
        if self.ppu is None:
            self.ppu = PPUState()
        if self.apu is None:
            self.apu = APUState()
        if self.input is None:
            self.input = InputState()
        if self.rom is None:
            self.rom = ROMHeader()


# Global Emulator Instance
emulator = EmulatorState()

# Separate Array Declarations (for compatibility with original design)
# These are accessed directly as module-level variables
memory_wram = [0] * 32768
memory_wram_extended = [0] * 131072

ppu_vram = [0] * PPU_VRAM_SIZE
ppu_cgram = [0] * PPU_CGRAM_SIZE
ppu_oam = [SpriteEntry() for _ in range(PPU_MAX_SPRITES)]
ppu_frame_buffer = [0] * (DISPLAY_WIDTH * DISPLAY_HEIGHT)
ppu_output_buffer = [0] * (DISPLAY_WIDTH * DISPLAY_HEIGHT)

apu_channels = [AudioChannel() for _ in range(APU_NUM_CHANNELS)]
apu_sample_buffer = [0] * APU_BUFFER_SIZE

input_keyboard_map = [0] * 256

