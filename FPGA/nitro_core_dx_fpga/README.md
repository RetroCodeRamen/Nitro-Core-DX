# Nitro-Core-DX FPGA Implementation

Complete Verilog implementation of the Nitro-Core-DX fantasy console for the Tang Mega 60K FPGA board.

## Project Structure

```
nitro_core_dx_fpga/
├── src/
│   ├── cpu/           # CPU Core (16-bit custom ISA)
│   │   └── cpu_core.v
│   ├── ppu/           # Picture Processing Unit
│   │   └── ppu_core.v
│   ├── apu/           # Audio Processing Unit
│   │   ├── apu_core.v
│   │   └── i2s_interface.v
│   ├── memory/        # Memory Controller
│   │   └── memory_controller.v
│   ├── video/         # Video Output
│   │   └── video_output.v
│   ├── io/            # Input/Output
│   │   └── input_controller.v
│   └── top/           # Top-level modules
│       ├── nitro_core_dx_top.v
│       └── clock_gen.v
├── constraints/
│   └── tang_mega_60k.cst    # Pin constraints
├── sim/               # Testbenches
│   └── tb_cpu_core.v
└── README.md
```

## Hardware Specifications

| Feature | Specification |
|---------|--------------|
| **Target Board** | Sipeed Tang Mega 60K |
| **FPGA** | GW5AT-LV60PG484A |
| **CPU** | Custom 16-bit @ 10 MHz |
| **PPU** | 4 background layers, 128 sprites, Matrix mode |
| **APU** | 4 channels, 44.1kHz, Sine/Square/Saw/Noise |
| **Resolution** | 320×200 → 720p HDMI |
| **Controllers** | 2× DB9 custom controllers |

## Controller Interface (DB9)

### Pinout (Female connector on console)

```
    5 4 3 2 1
     9 8 7 6

Pin 1: UP       - D-pad Up
Pin 2: DOWN     - D-pad Down
Pin 3: LEFT     - D-pad Left
Pin 4: RIGHT    - D-pad Right
Pin 5: +5V      - Power to controller
Pin 6: BTN_A    - Main Button A
Pin 7: SELECT   - Select line (output from console)
Pin 8: GND      - Ground
Pin 9: BTN_B    - Main Button B (BTN_C when SELECT=0)
```

### Button Mapping (12-bit register)

| Bit | Button |
|-----|--------|
| 0   | UP |
| 1   | DOWN |
| 2   | LEFT |
| 3   | RIGHT |
| 4   | BTN_A |
| 5   | BTN_B |
| 6   | BTN_C |
| 7   | L (shoulder) |
| 8   | R (shoulder) |
| 9   | SELECT |
| 10  | START |
| 11  | (reserved) |

### Extended Buttons (Genesis 6-button style)

When SELECT line is HIGH, the D-pad pins are multiplexed:
- UP → L
- DOWN → R
- LEFT → START
- RIGHT → SELECT

## Building the Project

### Prerequisites

- Gowin IDE V1.9.11.03 or later (Educational version)
- Tang Mega 60K development board

### Build Steps

1. Open Gowin IDE
2. Create new project targeting GW5AT-LV60PG484A
3. Add all source files from `src/` directory
4. Import pin constraints from `constraints/tang_mega_60k.cst`
5. Synthesize and implement
6. Generate bitstream
7. Program FPGA via USB-C

### Programming

```bash
# Using Gowin Programmer CLI
programmer_cli --device GW5AT-LV60PG484A \
    --operation "exFlash Erase, Program through GAO-Bridge 5A" \
    --file nitro_core_dx.fs
```

## Clock Domains

| Clock | Frequency | Purpose |
|-------|-----------|---------|
| clk_50m | 50 MHz | Input reference |
| clk_10m | 10 MHz | CPU core |
| clk_25m | 25 MHz | PPU rendering |
| clk_74_25m | 74.25 MHz | HDMI 720p pixel clock |
| clk_11_2896m | 11.2896 MHz | Audio master clock |

## Memory Map

| Address Range | Description |
|---------------|-------------|
| 0x0000-0x7FFF | WRAM (32KB) |
| 0x8000-0x80FF | PPU Control Registers |
| 0x8100-0x81FF | PPU Scroll Registers |
| 0x8200-0x82FF | Matrix Mode Registers |
| 0x8300-0x83FF | Sprite OAM |
| 0x8400-0x84FF | CGRAM (Palette) |
| 0x9000-0x902F | APU Registers |
| 0xA000-0xA00F | Input Controller |
| 0x010000+ | ROM Space (Bank 1+) |

## I/O Registers

### PPU Registers (0x8000-0x80FF)

| Address | Register | Description |
|---------|----------|-------------|
| 0x8000 | PPUCTRL | Control flags |
| 0x8001 | PPUMASK | Layer enable mask |
| 0x8002 | PPUSTATUS | Status/VBlank |
| 0x8010-0x8013 | BG0_SCROLL_X/Y | Background 0 scroll |
| 0x8014-0x8017 | BG1_SCROLL_X/Y | Background 1 scroll |
| 0x8018-0x801B | BG2_SCROLL_X/Y | Background 2 scroll |
| 0x801C-0x801F | BG3_SCROLL_X/Y | Background 3 scroll |

### APU Registers (0x9000-0x902F)

| Address | Register | Description |
|---------|----------|-------------|
| 0x9000 | CH0_CTRL | Channel 0 control |
| 0x9001 | CH0_FREQ_L | Channel 0 frequency (low) |
| 0x9002 | CH0_FREQ_H | Channel 0 frequency (high) |
| 0x9003 | CH0_VOL | Channel 0 volume |
| 0x9020 | MASTER_VOL | Master volume |
| 0x9021 | COMPLETION | Channel completion status |

### Input Registers (0xA000-0xA00F)

| Address | Register | Description |
|---------|----------|-------------|
| 0xA000 | CONTROLLER1 | Controller 1 buttons |
| 0xA001 | CONTROLLER2 | Controller 2 buttons |
| 0xA002 | STATUS | Controller status |

## License

This implementation is provided as reference for the Nitro-Core-DX fantasy console project.
