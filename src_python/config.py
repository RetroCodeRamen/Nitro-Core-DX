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
# SNES 65816 runs at 2.68 MHz (typical) = 44,667 cycles/frame at 60 FPS
# We'll use period-accurate SNES timing: 44,667 cycles/frame (2.68 MHz)
CPU_CYCLES_PER_FRAME = 44667  # Cycles per frame at 60 FPS (SNES 2.68 MHz period-accurate)
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
PPU_REG_VRAM_ADDR = 0x0A  # VRAM address (16-bit)
PPU_REG_VRAM_DATA = 0x0C  # VRAM data (8-bit)
PPU_REG_CGRAM_ADDR = 0x0E  # CGRAM address (9-bit, but 8-bit register)
PPU_REG_CGRAM_DATA = 0x0F  # CGRAM data (16-bit RGB555)
PPU_REG_OAM_ADDR = 0x10  # OAM address (8-bit)
PPU_REG_OAM_DATA = 0x11  # OAM data (multiple bytes per sprite)
PPU_REG_FRAMEBUFFER_ENABLE = 0x12  # Framebuffer enable
PPU_REG_DISPLAY_MODE = 0x13  # Display mode (portrait/landscape)
# Matrix Mode registers (90's retro-futuristic perspective/rotation effects)
PPU_REG_MATRIX_CONTROL = 0x14  # Matrix Mode control (bit 0=enable, bit 1=mirror_h, bit 2=mirror_v)
PPU_REG_MATRIX_A = 0x15  # Matrix A (16-bit, low byte)
PPU_REG_MATRIX_A_H = 0x16  # Matrix A (high byte)
PPU_REG_MATRIX_B = 0x17  # Matrix B (16-bit, low byte)
PPU_REG_MATRIX_B_H = 0x18  # Matrix B (high byte)
PPU_REG_MATRIX_C = 0x19  # Matrix C (16-bit, low byte)
PPU_REG_MATRIX_C_H = 0x1A  # Matrix C (high byte)
PPU_REG_MATRIX_D = 0x1B  # Matrix D (16-bit, low byte)
PPU_REG_MATRIX_D_H = 0x1C  # Matrix D (high byte)
PPU_REG_MATRIX_CENTER_X = 0x1D  # Center X (16-bit, low byte)
PPU_REG_MATRIX_CENTER_X_H = 0x1E  # Center X (high byte)
PPU_REG_MATRIX_CENTER_Y = 0x1F  # Center Y (16-bit, low byte)
PPU_REG_MATRIX_CENTER_Y_H = 0x20  # Center Y (high byte)

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
    priority: int = 0  # Priority (0-3)
    flip_x: bool = False  # Horizontal flip
    flip_y: bool = False  # Vertical flip
    size: int = 0  # 0=8x8, 1=16x16
    enabled: bool = False


@dataclass
class TileLayer:
    """Tilemap background layer"""
    scroll_x: int = 0
    scroll_y: int = 0
    tile_size: int = PPU_TILE_SIZE_8X8  # 8 or 16
    enabled: bool = False
    tile_map_base: int = 0  # VRAM offset for tilemap
    tile_data_base: int = 0  # VRAM offset for tile data


@dataclass
class PPUState:
    """PPU (Picture Processing Unit) state"""
    # VRAM (tile patterns, tilemaps)
    vram: List[int] = None  # Will be [0] * PPU_VRAM_SIZE
    
    # CGRAM (palette)
    cgram: List[int] = None  # Will be [0] * PPU_CGRAM_SIZE
    
    # OAM (sprite attributes)
    oam: List[SpriteEntry] = None  # Will be [SpriteEntry()] * PPU_MAX_SPRITES
    
    # Background layers
    bg0: TileLayer = None
    bg1: TileLayer = None
    
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
        if self.frame_buffer is None:
            self.frame_buffer = [0] * (DISPLAY_WIDTH * DISPLAY_HEIGHT)
        if self.output_buffer is None:
            self.output_buffer = [0] * (DISPLAY_WIDTH * DISPLAY_HEIGHT)


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

