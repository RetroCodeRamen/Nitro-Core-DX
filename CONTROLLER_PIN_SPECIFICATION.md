# Controller Pin Specification

**Version 1.0**  
**Last Updated: January 30, 2026**  
**Purpose: Physical controller interface specification using DB-9 connectors**

---

## Summary

**Connector Type: DB-9 (DE-9) Female on Console**  
**Total Pins: 9 pins**  
**Interface Type: Serial Shift Register (SNES-style)**

This specification defines the physical interface for Nitro-Core-DX controllers using standard DB-9 connectors, similar to Atari/Commodore style connectors.

---

## Controller Requirements

### Button Count
- **12 buttons per controller**: UP, DOWN, LEFT, RIGHT, A, B, X, Y, L, R, START, Z
- **2 controllers supported**: Controller 1 and Controller 2

### Button Mapping

| Button | Bit Position | Description |
|--------|--------------|-------------|
| UP | 0 | D-pad up |
| DOWN | 1 | D-pad down |
| LEFT | 2 | D-pad left |
| RIGHT | 3 | D-pad right |
| A | 4 | Primary action button |
| B | 5 | Secondary action button |
| X | 6 | Tertiary action button |
| Y | 7 | Quaternary action button |
| L | 8 | Left shoulder button |
| R | 9 | Right shoulder button |
| START | 10 | Start/pause button |
| Z | 11 | Additional shoulder button |

**Total: 12 buttons = 12 bits (fits in 16-bit word)**

---

## DB-9 Pinout Specification

### Pin Assignment

```
DB-9 Connector (Female on Console, Male on Controller Cable):

    5  4  3  2  1
     \  \  \  \  \
      \  \  \  \  \
       \  \  \  \  \
        \  \  \  \  \
         \  \  \  \  \
          └──┴──┴──┴──┘
             9  8  7  6

Pin Layout (viewed from front of connector):

Pin 1: +5V (Power)           [Red wire recommended]
Pin 2: GND (Ground)          [Black wire recommended]
Pin 3: DATA (Serial Data In) [Data from controller]
Pin 4: LATCH (Latch Signal)  [Output from console]
Pin 5: CLK (Clock Signal)    [Output from console]
Pin 6: GND (Ground)           [Additional ground for signal integrity]
Pin 7: Reserved              [Future use / No Connect]
Pin 8: Reserved              [Future use / No Connect]
Pin 9: Reserved              [Future use / No Connect]
```

### Signal Descriptions

| Pin | Signal | Type | Description |
|-----|--------|------|-------------|
| 1 | +5V | Power | 5V power supply for controller (max 100mA) |
| 2 | GND | Ground | Ground reference |
| 3 | DATA | Input | Serial data input from controller (active high) |
| 4 | LATCH | Output | Latch signal (active high, pulses to capture button state) |
| 5 | CLK | Output | Clock signal (active high, shifts data on rising edge) |
| 6 | GND | Ground | Additional ground for signal integrity |
| 7-9 | Reserved | - | Reserved for future expansion (analog sticks, rumble, etc.) |

**Note:** Pins 7-9 are reserved for future features such as:
- Analog joystick support
- Rumble/vibration motors
- Additional buttons
- Controller ID/authentication

---

## Communication Protocol

### Serial Shift Register Interface

The controller uses a **serial shift register** interface, similar to the SNES controller protocol:

1. **Latch Phase**: Console pulses LATCH high to capture current button states
2. **Shift Phase**: Console pulses CLK to shift button data out serially
3. **Data Phase**: Controller outputs button states on DATA line, one bit per clock

### Timing Diagram

```
LATCH:  ─────┐     ┌──────────────────────────────────────
             └─────┘
             
CLK:    ────┐ ┌──┐ ┌──┐ ┌──┐ ┌──┐ ┌──┐ ┌──┐ ┌──┐ ┌──┐ ┌──┐ ┌──┐ ┌──┐ ┌──┐
            └─┘ └─┘ └─┘ └─┘ └─┘ └─┘ └─┘ └─┘ └─┘ └─┘ └─┘ └─┘ └─┘ └─┘ └─┘
            
DATA:   ────[B0][B1][B2][B3][B4][B5][B6][B7][B8][B9][B10][B11][NC][NC][NC][NC]
        UP  DN  LT  RT  A   B   X   Y   L   R   ST  Z   (reserved bits)
        
        ↑ Data is valid on rising edge of CLK
        ↑ Read DATA line on rising edge of CLK
```

### Protocol Steps

1. **Latch Button States**:
   - Set LATCH = HIGH (captures current button states in shift register)
   - Wait minimum 12μs (latch setup time)
   - Set LATCH = LOW

2. **Read Button Data**:
   - For each of 16 clock pulses:
     - Set CLK = HIGH
     - Wait minimum 6μs (clock high time)
     - Read DATA line (button state)
     - Set CLK = LOW
     - Wait minimum 6μs (clock low time)
   - First 12 bits contain button data (UP through Z)
   - Last 4 bits are reserved (always 0)

3. **Button State**:
   - 1 = Button pressed
   - 0 = Button released

### Timing Specifications

| Parameter | Min | Max | Unit | Description |
|-----------|-----|-----|------|-------------|
| tLATCH_HIGH | 12 | 100 | μs | Latch pulse width |
| tCLK_HIGH | 6 | 50 | μs | Clock high time |
| tCLK_LOW | 6 | 50 | μs | Clock low time |
| tDATA_SETUP | 1 | - | μs | Data setup before clock |
| tDATA_HOLD | 1 | - | μs | Data hold after clock |
| tREAD_CYCLE | - | 1000 | μs | Total read cycle time |

**Recommended Clock Frequency**: 100 kHz (10μs period = 5μs high + 5μs low)

---

## Controller Hardware Design

### Design Option 1: ESP32-Based Controller (Recommended)

**Why ESP32?**
- ✅ Very cheap (~$3-5 per module)
- ✅ Plenty of GPIO pins (12+ buttons, easy)
- ✅ Easy to program (Arduino IDE, ESP-IDF)
- ✅ Built-in WiFi/Bluetooth (wireless support later!)
- ✅ Built-in ADC (analog sticks in future)
- ✅ Can handle debouncing in software
- ✅ More flexible than shift register
- ✅ Can add features like LEDs, rumble, etc.

**ESP32 Controller Circuit:**

```
Controller PCB:

ESP32 Module (ESP32-WROOM-32 or ESP32-S2):
  - GPIO pins for buttons (12 buttons = 12 GPIO)
  - One GPIO for DATA output (serial)
  - One GPIO for LATCH input (from console)
  - One GPIO for CLK input (from console)
  - 3.3V power (from 5V via regulator or direct if console provides 3.3V)

Buttons (12x):
  UP, DOWN, LEFT, RIGHT, A, B, X, Y, L, R, START, Z
  └─ Each button connects to one GPIO pin
  └─ Use internal pull-up resistors (ESP32 has built-in pull-ups)
  └─ Button pressed = GPIO reads LOW (active low)

Power:
  - +5V from Pin 1 → 3.3V regulator (AMS1117-3.3) → ESP32 VIN
  - Or: Console provides 3.3V directly (better efficiency)
  - GND from Pin 2 and Pin 6

Optional:
  - Status LED (GPIO)
  - Rumble motor (PWM via GPIO)
  - Battery for wireless mode (future)
```

**ESP32 Pin Assignment:**

| ESP32 Pin | Function | DB-9 Connection |
|-----------|----------|-----------------|
| GPIO 0-11 | Button inputs (12 buttons) | Internal |
| GPIO 12 | DATA output | Pin 3 |
| GPIO 13 | LATCH input | Pin 4 |
| GPIO 14 | CLK input | Pin 5 |
| GPIO 15 | Status LED (optional) | - |
| GPIO 16 | Rumble PWM (optional) | - |
| VIN | Power (3.3V or 5V) | Pin 1 (via regulator) |
| GND | Ground | Pin 2, 6 |

**ESP32 Firmware (Arduino-style):**

```cpp
// ESP32 Controller Firmware
#define BUTTON_COUNT 12
#define DATA_PIN 12
#define LATCH_PIN 13
#define CLK_PIN 14

// Button GPIO pins (assign as needed)
const int buttonPins[BUTTON_COUNT] = {
  0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11  // UP, DOWN, LEFT, RIGHT, A, B, X, Y, L, R, START, Z
};

uint16_t buttonState = 0;
bool latchActive = false;

void setup() {
  // Configure button pins with internal pull-ups
  for (int i = 0; i < BUTTON_COUNT; i++) {
    pinMode(buttonPins[i], INPUT_PULLUP);
  }
  
  // Configure interface pins
  pinMode(DATA_PIN, OUTPUT);
  pinMode(LATCH_PIN, INPUT);
  pinMode(CLK_PIN, INPUT);
  
  digitalWrite(DATA_PIN, LOW);
}

void loop() {
  // Check for latch signal
  bool latch = digitalRead(LATCH_PIN);
  
  if (latch && !latchActive) {
    // Latch rising edge - capture button states
    buttonState = 0;
    for (int i = 0; i < BUTTON_COUNT; i++) {
      // Read button (active low, so invert)
      if (!digitalRead(buttonPins[i])) {
        buttonState |= (1 << i);
      }
    }
    latchActive = true;
  } else if (!latch) {
    latchActive = false;
  }
  
  // Shift data out on clock
  static bool lastClock = false;
  static int bitPosition = 0;
  
  bool clock = digitalRead(CLK_PIN);
  
  if (clock && !lastClock) {
    // Clock rising edge - shift out next bit
    bool bit = (buttonState >> bitPosition) & 1;
    digitalWrite(DATA_PIN, bit);
    bitPosition++;
    if (bitPosition >= 16) {
      bitPosition = 0;
    }
  } else if (!clock && lastClock) {
    // Clock falling edge - prepare for next bit
  }
  
  lastClock = clock;
}
```

**ESP32 Advantages:**
- Very cheap (~$3-5)
- Easy to program and update firmware
- Can add wireless support later (WiFi/Bluetooth)
- Can add features like rumble, LEDs, analog sticks
- Software debouncing (no hardware needed)
- More flexible for future expansion

**ESP32 Power Considerations:**
- ESP32 needs 3.3V (can take 5V on VIN with internal regulator)
- Current draw: ~80-240mA (active), ~10μA (deep sleep)
- Console needs to provide 3.3V or 5V (with regulator on controller)
- **Recommendation**: Console provides 3.3V directly (more efficient)

---

### Design Option 2: Shift Register (Traditional)

**Controller Circuit (Simplified):**

```
Controller PCB:

Buttons (12x):
  UP, DOWN, LEFT, RIGHT, A, B, X, Y, L, R, START, Z
  └─ All buttons connect to shift register inputs

Shift Register (74HC165 or equivalent):
  - 8-bit parallel-in, serial-out shift register
  - Or use two cascaded 8-bit registers for 16 bits
  - Data input: Button states (active low buttons → invert)
  - Serial output: DATA pin
  - Clock input: CLK pin
  - Latch input: LATCH pin (active low → invert)

Power:
  - +5V from Pin 1
  - GND from Pin 2 and Pin 6

Optional: Pull-up resistors on DATA line (10kΩ)
Optional: Debounce capacitors on button inputs (0.1μF)
```

**Shift Register Options:**

**Option 2A: Two 8-bit Cascaded**
- Use two 74HC165 chips cascaded
- First chip: Buttons 0-7 (UP through Y)
- Second chip: Buttons 8-11 + 4 reserved bits (L, R, START, Z, + 4 NC)
- Cost: ~$0.50 per chip = $1.00 total

**Option 2B: Single 16-bit Solution**
- Use specialized 16-bit shift register (if available)
- Or use two 74HC165 cascaded (same as 2A)

**Shift Register Advantages:**
- Very cheap (~$1 total)
- Very low power (~5-10mA)
- Simple, reliable
- No programming needed

**Shift Register Disadvantages:**
- Less flexible
- No wireless support
- Harder to add features
- Requires hardware debouncing

### Button Wiring

**Active-Low Buttons** (recommended, common in retro controllers):
```
Button → GND when pressed
Button → +5V (via pull-up) when released
```

**Active-High Buttons**:
```
Button → +5V when pressed
Button → GND (via pull-down) when released
```

**Note:** The shift register input polarity should match button wiring. Most shift registers expect active-low inputs, so active-low buttons work directly.

---

## FPGA Implementation

### Controller Interface Module

```verilog
module controller_interface (
    // System signals
    input wire clk,           // System clock (10 MHz)
    input wire rst_n,
    
    // Controller port (DB-9)
    output reg latch,        // Pin 4: Latch signal
    output reg clock,        // Pin 5: Clock signal
    input wire data,         // Pin 3: Serial data input
    
    // Controller data output
    output reg [15:0] button_state,
    output reg data_valid,
    input wire read_enable   // Trigger read cycle
);

    // State machine
    reg [4:0] state;
    reg [4:0] bit_count;
    
    localparam IDLE = 5'd0;
    localparam LATCH_HIGH = 5'd1;
    localparam LATCH_LOW = 5'd2;
    localparam SHIFT = 5'd3;
    localparam DONE = 5'd4;
    
    // Timing counters (for 10 MHz clock)
    reg [7:0] timer;
    localparam LATCH_TIME = 8'd120;  // 12μs at 10 MHz
    localparam CLK_HIGH_TIME = 8'd6;   // 0.6μs at 10 MHz
    localparam CLK_LOW_TIME = 8'd6;    // 0.6μs at 10 MHz
    
    always @(posedge clk) begin
        if (!rst_n) begin
            state <= IDLE;
            latch <= 1'b0;
            clock <= 1'b0;
            button_state <= 16'h0000;
            bit_count <= 5'd0;
            data_valid <= 1'b0;
            timer <= 8'd0;
        end else begin
            case (state)
                IDLE: begin
                    data_valid <= 1'b0;
                    if (read_enable) begin
                        state <= LATCH_HIGH;
                        latch <= 1'b1;
                        timer <= 8'd0;
                    end
                end
                
                LATCH_HIGH: begin
                    if (timer >= LATCH_TIME) begin
                        state <= LATCH_LOW;
                        latch <= 1'b0;
                        timer <= 8'd0;
                        bit_count <= 5'd0;
                        button_state <= 16'h0000;
                    end else begin
                        timer <= timer + 1;
                    end
                end
                
                LATCH_LOW: begin
                    if (timer >= LATCH_TIME) begin
                        state <= SHIFT;
                        clock <= 1'b1;
                        timer <= 8'd0;
                    end else begin
                        timer <= timer + 1;
                    end
                end
                
                SHIFT: begin
                    if (clock) begin
                        // Clock is high, read data
                        button_state <= {button_state[14:0], data};
                        if (timer >= CLK_HIGH_TIME) begin
                            clock <= 1'b0;
                            timer <= 8'd0;
                            if (bit_count >= 15) begin
                                state <= DONE;
                            end
                        end else begin
                            timer <= timer + 1;
                        end
                    end else begin
                        // Clock is low
                        if (timer >= CLK_LOW_TIME) begin
                            clock <= 1'b1;
                            timer <= 8'd0;
                            bit_count <= bit_count + 1;
                        end else begin
                            timer <= timer + 1;
                        end
                    end
                end
                
                DONE: begin
                    data_valid <= 1'b1;
                    state <= IDLE;
                end
            endcase
        end
    end

endmodule
```

### Dual Controller Interface

```verilog
module dual_controller_interface (
    input wire clk,
    input wire rst_n,
    
    // Controller 1 (DB-9)
    output wire ctrl1_latch,
    output wire ctrl1_clock,
    input wire ctrl1_data,
    
    // Controller 2 (DB-9)
    output wire ctrl2_latch,
    output wire ctrl2_clock,
    input wire ctrl2_data,
    
    // Outputs
    output reg [15:0] controller1_buttons,
    output reg [15:0] controller2_buttons,
    output reg data_valid,
    
    // Control
    input wire read_enable
);

    wire [15:0] ctrl1_state, ctrl2_state;
    wire ctrl1_valid, ctrl2_valid;
    
    controller_interface ctrl1 (
        .clk(clk),
        .rst_n(rst_n),
        .latch(ctrl1_latch),
        .clock(ctrl1_clock),
        .data(ctrl1_data),
        .button_state(ctrl1_state),
        .data_valid(ctrl1_valid),
        .read_enable(read_enable)
    );
    
    controller_interface ctrl2 (
        .clk(clk),
        .rst_n(rst_n),
        .latch(ctrl2_latch),
        .clock(ctrl2_clock),
        .data(ctrl2_data),
        .button_state(ctrl2_state),
        .data_valid(ctrl2_valid),
        .read_enable(read_enable)
    );
    
    always @(posedge clk) begin
        if (ctrl1_valid && ctrl2_valid) begin
            controller1_buttons <= ctrl1_state;
            controller2_buttons <= ctrl2_state;
            data_valid <= 1'b1;
        end else begin
            data_valid <= 1'b0;
        end
    end

endmodule
```

---

## Physical Connector Details

### DB-9 Connector Specifications

**Console Side (Female/Receptacle):**
- **Type**: DB-9 Female (DE-9F)
- **Mounting**: Panel mount or PCB mount
- **Pitch**: Standard 0.1" (2.54mm)
- **Material**: Gold-plated contacts recommended

**Controller Cable (Male/Plug):**
- **Type**: DB-9 Male (DE-9M)
- **Cable**: Shielded cable recommended (reduces interference)
- **Length**: 1.5-2 meters typical
- **Strain Relief**: Recommended for durability

### Wiring Colors (Recommended)

| Pin | Signal | Color | Alternative |
|-----|--------|-------|-------------|
| 1 | +5V | Red | Orange |
| 2 | GND | Black | Brown |
| 3 | DATA | Green | Yellow |
| 4 | LATCH | Blue | White |
| 5 | CLK | Yellow | Purple |
| 6 | GND | Black | Brown (shield) |

**Note:** Use consistent color coding for manufacturing and repair.

### Connector Placement

**Console Design:**
- Two DB-9 connectors on front panel
- Labeled "CONTROLLER 1" and "CONTROLLER 2"
- Spaced appropriately for comfortable use
- Consider ergonomics (not too close together)

---

## Power Requirements

### Controller Power Consumption

**ESP32-Based Controller:**
| Component | Current | Notes |
|-----------|---------|-------|
| ESP32 (active) | ~80-240 mA | Depends on CPU speed, WiFi usage |
| ESP32 (light sleep) | ~0.8-1.5 mA | When not actively reading buttons |
| ESP32 (deep sleep) | ~10 μA | When in sleep mode |
| Buttons | ~0.1 mA | Negligible (internal pull-ups) |
| LEDs (optional) | 0-20 mA | If controller has status LEDs |
| **Total (active)** | **~80-260 mA** | Per controller (active use) |
| **Total (idle)** | **~1-2 mA** | Per controller (light sleep) |

**Shift Register Controller:**
| Component | Current | Notes |
|-----------|---------|-------|
| Shift Register (74HC165) | ~5-10 mA | Low power CMOS |
| Button Pull-ups | ~1-2 mA | If using pull-up resistors |
| LEDs (optional) | 0-20 mA | If controller has status LEDs |
| **Total** | **~10-30 mA** | Per controller |

**Console Power Supply:**

**For ESP32 Controllers:**
- **Option A**: Provide 3.3V directly (recommended)
  - 3.3V at 500mA minimum per controller port
  - Total: 1A for two controllers
  - More efficient (no regulator losses)
  
- **Option B**: Provide 5V, controller has regulator
  - 5V at 300mA minimum per controller port
  - Total: 600mA for two controllers
  - Controller has AMS1117-3.3 regulator (~40% efficiency loss)

**For Shift Register Controllers:**
- Provide +5V at 100mA minimum per controller port
- Total: 200mA for two controllers

**Power Supply Design:**
- Use voltage regulator (7805 for 5V, AMS1117-3.3 for 3.3V)
- Or switching regulator (more efficient, e.g., LM2596)
- Add decoupling capacitors (100μF + 0.1μF) near connectors
- Consider current limiting or polyfuse protection

### Protection

**Recommended Protection:**
- Current limiting resistors on +5V lines (optional, 10Ω)
- TVS diodes on data lines (protect against ESD)
- Fuse or polyfuse on +5V supply (500mA)

---

## Testing and Verification

### Controller Test Procedure

1. **Power Test**:
   - Measure +5V on Pin 1 (should be 4.75V - 5.25V)
   - Measure GND continuity on Pins 2 and 6

2. **Latch Test**:
   - Pulse LATCH high
   - Verify controller captures button states
   - Measure LATCH pulse width (should be > 12μs)

3. **Clock Test**:
   - Generate 16 clock pulses
   - Verify clock frequency (100 kHz recommended)
   - Measure clock high/low times

4. **Data Test**:
   - Press known buttons
   - Verify data bits match button states
   - Test all 12 buttons individually

5. **Timing Test**:
   - Measure total read cycle time (< 1ms)
   - Verify data setup/hold times

---

## Future Expansion (Pins 7-9)

### Potential Uses for Reserved Pins

**Pin 7: Analog Stick X-axis**
- ADC output (0-5V = 0-255)
- Or PWM signal
- Or I2C/SPI interface

**Pin 8: Analog Stick Y-axis**
- ADC output (0-5V = 0-255)
- Or PWM signal

**Pin 9: Additional Features**
- Rumble motor control (PWM)
- Controller ID/authentication
- Additional buttons (via multiplexing)
- Status LED control

**Note:** Future expansion should maintain backward compatibility with basic controllers.

---

## Compatibility Notes

### DB-9 vs Other Connectors

**Advantages of DB-9:**
- ✅ Widely available
- ✅ Inexpensive
- ✅ Robust (metal shell)
- ✅ Easy to source cables
- ✅ Standard pin spacing
- ✅ Good for prototyping

**Alternatives Considered:**
- **USB**: More complex, requires USB controller chip
- **RJ-45 (Ethernet)**: 8 pins only, not enough for future expansion
- **Custom connector**: Expensive tooling, harder to source
- **Mini-DIN**: Less common, more expensive

**DB-9 is the best choice** for this application.

---

## Summary

**Connector**: DB-9 (DE-9) Female on console  
**Pins Used**: 6 pins (3 reserved for future)  
**Interface**: Serial shift register (SNES-style)  
**Buttons**: 12 buttons per controller  
**Protocol**: Serial, 16 bits, ~100 kHz clock  

### Recommended Controller Design: ESP32-Based

**Why ESP32?**
- ✅ Very cheap (~$3-5 per module)
- ✅ Easy to program and update
- ✅ Can add wireless support later (WiFi/Bluetooth)
- ✅ Flexible for future features (rumble, LEDs, analog)
- ✅ Software debouncing (no hardware needed)
- ✅ More modern approach while maintaining compatibility

**Power Requirements (ESP32):**
- **Option A (Recommended)**: 3.3V @ 500mA per controller
- **Option B**: 5V @ 300mA per controller (with on-board regulator)

### Alternative: Shift Register Design

**Why Shift Register?**
- ✅ Very cheap (~$1 total)
- ✅ Very low power (~10-30mA)
- ✅ Simple, reliable
- ✅ No programming needed

**Power Requirements (Shift Register):**
- 5V @ 100mA per controller

**Key Features:**
- Simple, reliable interface
- Standard connector (easy to source)
- Room for future expansion (3 reserved pins)
- Compatible with retro console design aesthetic
- Easy to implement in FPGA
- **ESP32 option adds modern flexibility while maintaining compatibility**

---

## References

- **Hardware Specification**: `HARDWARE_SPECIFICATION.md` - Section 7: Input System
- **Input System Code**: `internal/input/input.go`
- **Programming Manual**: `NITRO_CORE_DX_PROGRAMMING_MANUAL.md` - Input System section

---

**End of Specification**
