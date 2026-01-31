# Cartridge Pin Specification

**Version 1.0**  
**Last Updated: January 30, 2026**  
**Purpose: Pin count and signal specification for cartridge connector design**

---

## Summary

**Total Pins Required: 40 pins** (minimum)

This specification provides the complete pinout for the Nitro-Core-DX cartridge connector, suitable for FPGA implementation and physical cartridge manufacturing.

---

## Pin Breakdown

### Address Bus (24 pins)

| Signal | Bits | Description |
|--------|------|-------------|
| A[23:16] | 8 | Bank number (0-255, ROM uses banks 1-125) |
| A[15:0] | 16 | Offset within bank (0x0000-0xFFFF, ROM uses 0x8000-0xFFFF) |

**Total: 24 address pins**

**Note:** While ROM only uses banks 1-125 and offsets 0x8000-0xFFFF, the full address bus is provided for:
- Future expansion
- Simplified address decoding
- Compatibility with standard memory interfaces

### Data Bus (8 pins)

| Signal | Bits | Description |
|--------|------|-------------|
| D[7:0] | 8 | Data bus (bidirectional, but ROM is read-only) |

**Total: 8 data pins**

**Note:** The data bus is 8-bit. 16-bit operations are performed as two sequential 8-bit reads.

### Control Signals (5 pins)

| Signal | Type | Description |
|--------|------|-------------|
| /CE | Input | Chip Enable (active low) - Enables cartridge when bank 1-125 is accessed |
| /OE | Input | Output Enable (active low) - Enables data output from cartridge |
| /RD | Input | Read Strobe (active low) - Indicates read operation |
| CLK | Input | System Clock (10 MHz) - Optional, for synchronous ROM access |
| /RST | Input | Reset (active low) - Optional, for cartridge initialization |

**Total: 5 control pins**

**Note:** 
- `/CE` can be generated from address decoding (banks 1-125)
- `/OE` and `/RD` may be combined into a single signal depending on interface design
- `CLK` is optional if using asynchronous ROM (most common)
- `/RST` is optional but recommended for proper initialization

### Power and Ground (3 pins minimum)

| Signal | Type | Description |
|--------|------|-------------|
| VCC | Power | Power supply (3.3V or 5V, depending on FPGA I/O standard) |
| GND | Ground | Ground reference |
| GND | Ground | Additional ground for signal integrity (recommended) |

**Total: 3 power/ground pins** (minimum, more recommended for signal integrity)

---

## Pin Count Summary

| Category | Pins | Notes |
|----------|------|-------|
| Address Bus | 24 | A[23:0] |
| Data Bus | 8 | D[7:0] |
| Control Signals | 5 | /CE, /OE, /RD, CLK, /RST |
| Power/Ground | 3+ | VCC, GND (multiple) |
| **TOTAL** | **40** | Minimum pin count |

---

## Recommended Connector

### Option 1: 40-pin Connector (Minimum)
- **Type**: 2×20 pin header (0.1" pitch) or edge connector
- **Pin Count**: 40 pins
- **Suitable for**: Basic ROM-only cartridges

### Option 2: 50-pin Connector (Recommended)
- **Type**: 2×25 pin header (0.1" pitch) or edge connector
- **Pin Count**: 50 pins
- **Allocation**:
  - 40 pins: Cartridge interface (can be optimized to 38-39 pins)
  - 8-10 pins: Expansion port + extra grounds
- **Suitable for**: Production cartridges with expansion port support

### Option 3: 60-pin Connector (Future-Proof)
- **Type**: 2×30 pin header (0.1" pitch) or edge connector
- **Pin Count**: 60 pins
- **Additional pins**: Multiple grounds, reserved pins for:
  - Battery-backed SRAM (save games)
  - Additional ROM banks
  - Special chips (co-processors, etc.)
- **Suitable for**: Advanced cartridges with save functionality

---

## Signal Timing

### Read Operation

```
Timing Diagram (typical asynchronous ROM):

    CLK  ──┐     ┌────┐     ┌────┐
           └─────┘     └─────┘     └──

    A[23:0] ────[Bank:Offset]───────────────

    /CE    ─────┐                    └─────
                └────────────────────

    /OE    ─────┐                    └─────
                └────────────────────

    /RD    ─────┐                    └─────
                └────────────────────

    D[7:0] ─────────────────[Data]─────────
                (valid after tACC)
```

**Timing Parameters:**
- **tACC** (Address to Data): Maximum ROM access time (typically 70-150ns for parallel flash)
- **tOE** (Output Enable): Time from /OE low to data valid (typically 20-50ns)
- **tCE** (Chip Enable): Time from /CE low to data valid (typically 70-150ns)

---

## Address Decoding

### Cartridge Enable Logic

The cartridge should be enabled when:
- Bank number is in range 1-125 (0x01-0x7D)
- Offset is in range 0x8000-0xFFFF (for LoROM mapping)

**FPGA Logic:**
```verilog
// Cartridge enable signal
assign cart_ce = (bank >= 8'd1) && (bank <= 8'd125) && (offset >= 16'h8000);

// Address decoding for ROM
// LoROM mapping: romOffset = (bank-1) * 32768 + (offset - 0x8000)
wire [22:0] rom_address;
assign rom_address = ((bank - 8'd1) * 23'd32768) + (offset - 16'h8000);
```

**Note:** The cartridge may implement its own address decoding, or the FPGA can provide a linear address.

---

## ROM Mapping Details

### LoROM Mapping

- **ROM Banks**: 1-125 (0x01-0x7D)
- **Bank Size**: 64KB (0x0000-0xFFFF)
- **ROM Window**: 0x8000-0xFFFF (32KB per bank)
- **Total ROM Space**: 7.8MB (125 banks × 32KB)

### Address Translation

```
CPU Address: [Bank:Offset]
  Bank = 1-125
  Offset = 0x8000-0xFFFF

ROM Linear Address = (Bank - 1) × 32768 + (Offset - 0x8000)
```

**Example:**
- CPU reads from Bank 2, Offset 0x9000
- ROM Address = (2-1) × 32768 + (0x9000 - 0x8000)
- ROM Address = 32768 + 4096 = 36864 (0x9000 in ROM)

---

## Physical Connector Considerations

### Pin Assignment (Recommended Layout)

**40-pin Connector (2×20):**

```
Pin Layout (viewed from cartridge side):

    1   2   3   4   5   6   7   8   9  10  11  12  13  14  15  16  17  18  19  20
    ┌───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┐
    │GND│VCC│A23│A22│A21│A20│A19│A18│A17│A16│A15│A14│A13│A12│A11│A10│ A9│ A8│ A7│ A6│
    ├───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┤
    │ A5│ A4│ A3│ A2│ A1│ A0│ D7│ D6│ D5│ D4│ D3│ D2│ D1│ D0│/CE│/OE│/RD│CLK│/RST│GND│
    └───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┘
    21  22  23  24  25  26  27  28  29  30  31  32  33  34  35  36  37  38  39  40
```

**Notes:**
- Power and ground on both ends for better distribution
- Address bus grouped together
- Data bus grouped together
- Control signals grouped together
- Multiple ground pins for signal integrity

**50-pin Connector with Expansion Port (2×25):**

```
Pin Layout (viewed from cartridge side):

    1   2   3   4   5   6   7   8   9  10  11  12  13  14  15  16  17  18  19  20  21  22  23  24  25
    ┌───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┐
    │GND│VCC│A23│A22│A21│A20│A19│A18│A17│A16│A15│A14│A13│A12│A11│A10│ A9│ A8│ A7│ A6│ A5│ A4│ A3│ A2│ A1│
    ├───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┼───┤
    │ A0│ D7│ D6│ D5│ D4│ D3│ D2│ D1│ D0│/CE│/OE│/RD│CLK│/RST│GND│EXT│EXT│EXT│EXT│EXT│EXT│EXT│EXT│GND│VCC│
    │   │   │   │   │   │   │   │   │   │   │   │   │   │    │   │D0 │D1 │D2 │D3 │/EN│/RD│/WR│   │   │   │
    └───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┴───┘
    26  27  28  29  30  31  32  33  34  35  36  37  38  39  40  41  42  43  44  45  46  47  48  49  50

Cartridge Interface: Pins 1-40 (40 pins)
Expansion Port: Pins 41-48 (8 pins)
  - EXT_D[3:0]: 4 data lines (bidirectional)
  - EXT_/EN: Enable (active low)
  - EXT_/RD: Read strobe (active low)
  - EXT_/WR: Write strobe (active low)
Power/Ground: Pins 1, 2, 39, 49, 50 (5 pins)
```

**Pin Allocation Summary:**
- **Cartridge Interface**: 40 pins (A[23:0], D[7:0], /CE, /OE, /RD, CLK, /RST)
- **Expansion Port**: 8 pins (EXT_D[3:0], EXT_/EN, EXT_/RD, EXT_/WR)
- **Power/Ground**: 5 pins (VCC×2, GND×3)
- **Total**: 50 pins (2 pins reserved for future use)

**Optimization Option:** If you need more expansion pins, you can:
- Combine /OE and /RD into a single /RD signal (saves 1 pin)
- Make CLK optional if using async ROM (saves 1 pin)
- This would free up 2 pins for expansion port (10 pins total)

### Connector Types

1. **Edge Connector** (like SNES/NES)
   - Gold-plated edge contacts
   - Cartridge slides into slot
   - Common in retro consoles

2. **Pin Header** (like Game Boy)
   - 0.1" pitch pin header
   - Cartridge plugs into header
   - Easier for prototyping

3. **Card Edge Connector**
   - Similar to edge connector
   - Better for production

---

## ROM Chip Selection

### Recommended ROM Types

1. **Parallel NOR Flash** (most common)
   - **Examples**: SST39VF series, AT29C series
   - **Speed**: 70-150ns access time
   - **Capacity**: Up to 8MB easily available
   - **Interface**: Standard parallel interface

2. **Parallel EEPROM**
   - **Examples**: AT28C series
   - **Speed**: 150-200ns access time
   - **Capacity**: Up to 1MB common
   - **Interface**: Standard parallel interface

3. **SRAM** (for development/testing)
   - **Examples**: 62256, 628128
   - **Speed**: 10-70ns access time
   - **Capacity**: Up to 512KB common
   - **Interface**: Standard parallel interface

### ROM Requirements

- **Minimum Capacity**: 32KB (1 bank)
- **Maximum Capacity**: 7.8MB (125 banks)
- **Access Time**: < 200ns (for 10 MHz CPU)
- **Interface**: 8-bit parallel
- **Voltage**: 3.3V or 5V (match FPGA I/O standard)

---

## FPGA Interface Example

### Verilog Module

```verilog
module cartridge_interface (
    // System signals
    input wire clk,
    input wire rst_n,
    
    // CPU interface
    input wire [7:0] bank,
    input wire [15:0] offset,
    input wire read_enable,
    output reg [7:0] data_out,
    output reg data_valid,
    
    // Cartridge connector
    output reg [23:0] cart_addr,
    output reg cart_ce_n,
    output reg cart_oe_n,
    output reg cart_rd_n,
    input wire [7:0] cart_data
);

    // Address decoding
    wire cart_enable;
    assign cart_enable = (bank >= 8'd1) && (bank <= 8'd125) && (offset >= 16'h8000);
    
    // ROM address calculation (LoROM mapping)
    wire [22:0] rom_addr;
    assign rom_addr = ((bank - 8'd1) * 23'd32768) + (offset - 16'h8000);
    
    // Control signals
    always @(posedge clk) begin
        if (!rst_n) begin
            cart_ce_n <= 1'b1;
            cart_oe_n <= 1'b1;
            cart_rd_n <= 1'b1;
            cart_addr <= 24'h000000;
            data_out <= 8'h00;
            data_valid <= 1'b0;
        end else begin
            if (read_enable && cart_enable) begin
                cart_ce_n <= 1'b0;
                cart_oe_n <= 1'b0;
                cart_rd_n <= 1'b0;
                cart_addr <= {1'b0, rom_addr}; // 24-bit address
                data_out <= cart_data;
                data_valid <= 1'b1;
            end else begin
                cart_ce_n <= 1'b1;
                cart_oe_n <= 1'b1;
                cart_rd_n <= 1'b1;
                data_valid <= 1'b0;
            end
        end
    end

endmodule
```

---

## Expansion Port Specification

### Expansion Port Pinout (8 pins minimum)

The expansion port provides a general-purpose I/O interface for external devices such as:
- Memory expansion (additional RAM)
- Co-processors
- Communication modules (WiFi, Bluetooth, etc.)
- Debug interfaces
- Custom hardware

| Pin | Signal | Type | Description |
|-----|--------|------|-------------|
| EXT_D[3:0] | 4 | Bidirectional | 4-bit data bus (can be extended to 8-bit by sharing cartridge data bus) |
| EXT_/EN | 1 | Input | Enable signal (active low) - Enables expansion device |
| EXT_/RD | 1 | Input | Read strobe (active low) - Indicates read operation |
| EXT_/WR | 1 | Input | Write strobe (active low) - Indicates write operation |
| **Total** | **7** | | Minimum expansion port |

**Note:** The expansion port uses a 4-bit data bus. For 8-bit transfers, you can:
- Use two sequential 4-bit transfers
- Share the cartridge data bus (requires bus arbitration)
- Use a separate 8-bit expansion port (requires more pins)

### Expansion Port Address Space

The expansion port can be mapped to reserved memory banks (e.g., banks 0x80-0xFF) or use I/O register space. The exact mapping depends on your FPGA implementation.

**Recommended Mapping:**
- **I/O Space**: 0xB000-0xBFFF (bank 0, offset 0xB000+)
- **Memory Space**: Banks 0x80-0xFF (reserved area)

### Expansion Port Example Use Cases

1. **Memory Expansion**
   - Additional SRAM for save games
   - Extended work RAM
   - Battery-backed memory

2. **Co-Processor**
   - Math co-processor
   - Graphics accelerator
   - Audio enhancement chip

3. **Communication**
   - Serial communication (UART)
   - Network interface
   - Wireless module

4. **Debug Interface**
   - JTAG connection
   - Logic analyzer interface
   - Development tools

---

## Future Expansion

### Additional Pins for Advanced Features

If you plan to add features later, consider reserving pins for:

1. **Save RAM** (battery-backed SRAM)
   - Additional address/data lines
   - Write enable signal
   - Battery voltage monitoring

2. **Special Chips**
   - Co-processor communication
   - Additional control signals
   - Interrupt signals

3. **Audio/Video Enhancement**
   - Audio output lines
   - Video enhancement signals

4. **Debug/Development**
   - JTAG interface
   - Debug signals
   - Test points

---

## Summary

**Minimum Pin Count: 40 pins** (cartridge only)

This includes:
- 24 address pins (A[23:0])
- 8 data pins (D[7:0])
- 5 control pins (/CE, /OE, /RD, CLK, /RST)
- 3+ power/ground pins (VCC, GND)

**Recommended: 50 pins** for:
- Cartridge interface (40 pins)
- Expansion port (8 pins minimum)
- Better signal integrity (extra grounds)
- Production reliability

**With 50 pins, you have:**
- ✅ Full 40-pin cartridge interface
- ✅ 8-pin expansion port (4 data + 4 control)
- ✅ Adequate power/ground distribution
- ✅ Room for future expansion (2 reserved pins)

**If you need more expansion pins:**
- **Option 1**: Optimize cartridge interface (combine /OE+/RD, make CLK optional) → frees 2 pins → 10 expansion pins
- **Option 2**: Use 60-pin connector → 40 cartridge + 16 expansion + 4 power/ground

---

## References

- **Hardware Specification**: `HARDWARE_SPECIFICATION.md`
- **Memory Map**: See Hardware Specification, Section 3
- **ROM Format**: See Hardware Specification, Section 9

---

**End of Specification**
