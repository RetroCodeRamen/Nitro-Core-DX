//============================================================================
// Nitro-Core-DX Input Controller
// Custom controller with DB9 interface
// 3 main buttons + L + R + Start + Select + D-pad
//============================================================================

`timescale 1ns / 1ps

module input_controller (
    input  wire        clk,
    input  wire        reset_n,
    
    // DB9 Controller 1 interface
    input  wire [8:0]  db9_1,           // DB9 pins (active low)
    output reg         db9_1_select,    // Select line output
    
    // DB9 Controller 2 interface  
    input  wire [8:0]  db9_2,
    output reg         db9_2_select,
    
    // Register interface (to CPU)
    input  wire [7:0]  reg_addr,
    output reg  [15:0] reg_rdata,
    input  wire        reg_re,
    
    // Controller outputs
    output reg  [11:0] buttons1,        // Controller 1 button state
    output reg  [11:0] buttons2,        // Controller 2 button state
    output reg         valid            // Controller data valid
);

//============================================================================
// Nitro-Core-DX Controller Button Mapping (12-bit)
//============================================================================
//
// Bit 0:  UP       - D-pad Up
// Bit 1:  DOWN     - D-pad Down
// Bit 2:  LEFT     - D-pad Left
// Bit 3:  RIGHT    - D-pad Right
// Bit 4:  BTN_A    - Main Button A (leftmost)
// Bit 5:  BTN_B    - Main Button B (center)
// Bit 6:  BTN_C    - Main Button C (rightmost)
// Bit 7:  L        - Left shoulder button
// Bit 8:  R        - Right shoulder button
// Bit 9:  SELECT   - Select button
// Bit 10: START    - Start button
// Bit 11: (reserved)

//============================================================================
// DB9 Pinout (Atari/Genesis style - 9-pin D-sub)
//============================================================================
//
// DB9 Female (looking at connector on console):
//
//    5 4 3 2 1
//     9 8 7 6
//
// Pin 1: UP       - D-pad Up
// Pin 2: DOWN     - D-pad Down
// Pin 3: LEFT     - D-pad Left
// Pin 4: RIGHT    - D-pad Right
// Pin 5: +5V      - Power (to controller)
// Pin 6: BTN_A    - Main Button A (also Fire on Atari)
// Pin 7: SELECT   - Select line (input to controller)
// Pin 8: GND      - Ground
// Pin 9: BTN_B    - Main Button B (also used for multiplexing)
//
// For 6-button controllers (Genesis style), SELECT line multiplexes:
// SELECT=0: Read D-pad + BTN_A + BTN_B + BTN_C
// SELECT=1: Read L + R + START + SELECT

//============================================================================
// Controller State Machine
//============================================================================

reg [2:0] state;
reg [15:0] poll_counter;
reg select_state;

localparam STATE_IDLE      = 3'd0;
localparam STATE_SELECT_LO = 3'd1;
localparam STATE_READ_LO   = 3'd2;
localparam STATE_SELECT_HI = 3'd3;
localparam STATE_READ_HI   = 3'd4;
localparam STATE_LATCH     = 3'd5;

//============================================================================
// Raw DB9 Input Registers (active low, so invert)
//============================================================================

// Controller 1 raw inputs
wire db9_1_up       = ~db9_1[0];
wire db9_1_down     = ~db9_1[1];
wire db9_1_left     = ~db9_1[2];
wire db9_1_right    = ~db9_1[3];
wire db9_1_btn_a    = ~db9_1[5];
wire db9_1_btn_b    = ~db9_1[8];
// BTN_C is on pin 9 when SELECT is high (Genesis 6-button)
// For 3-button, BTN_C is read differently

// Controller 2 raw inputs
wire db9_2_up       = ~db9_2[0];
wire db9_2_down     = ~db9_2[1];
wire db9_2_left     = ~db9_2[2];
wire db9_2_right    = ~db9_2[3];
wire db9_2_btn_a    = ~db9_2[5];
wire db9_2_btn_b    = ~db9_2[8];

// Extended button registers (for 6-button mode)
reg db9_1_btn_c, db9_1_l, db9_1_r, db9_1_start, db9_1_select;
reg db9_2_btn_c, db9_2_l, db9_2_r, db9_2_start, db9_2_select;

//============================================================================
// Polling Counter
//============================================================================

// Poll controllers at ~1kHz (every 10000 cycles at 10MHz)
localparam POLL_PERIOD = 16'd10000;

always @(posedge clk or negedge reset_n) begin
    if (!reset_n) begin
        poll_counter <= 16'd0;
    end else begin
        if (poll_counter >= POLL_PERIOD) begin
            poll_counter <= 16'd0;
        end else begin
            poll_counter <= poll_counter + 1;
        end
    end
end

wire poll_tick = (poll_counter == POLL_PERIOD);

//============================================================================
// Controller State Machine
//============================================================================

always @(posedge clk or negedge reset_n) begin
    if (!reset_n) begin
        state <= STATE_IDLE;
        db9_1_select <= 1'b1;
        db9_2_select <= 1'b1;
        select_state <= 1'b0;
        
        buttons1 <= 12'd0;
        buttons2 <= 12'd0;
        valid <= 1'b0;
        
        db9_1_btn_c <= 1'b0;
        db9_1_l <= 1'b0;
        db9_1_r <= 1'b0;
        db9_1_start <= 1'b0;
        db9_1_select <= 1'b0;
        
        db9_2_btn_c <= 1'b0;
        db9_2_l <= 1'b0;
        db9_2_r <= 1'b0;
        db9_2_start <= 1'b0;
        db9_2_select <= 1'b0;
    end else begin
        case (state)
            STATE_IDLE: begin
                valid <= 1'b0;
                if (poll_tick) begin
                    state <= STATE_SELECT_LO;
                    db9_1_select <= 1'b0;
                    db9_2_select <= 1'b0;
                    select_state <= 1'b0;
                end
            end
            
            STATE_SELECT_LO: begin
                // Wait for select line to settle
                state <= STATE_READ_LO;
            end
            
            STATE_READ_LO: begin
                // Read buttons with SELECT=0
                // Standard 3-button: D-pad + A + B
                // On Genesis 6-button, this also gives us C
                db9_1_btn_c <= ~db9_1[8];  // BTN_C when SELECT=0
                db9_2_btn_c <= ~db9_2[8];
                
                state <= STATE_SELECT_HI;
                db9_1_select <= 1'b1;
                db9_2_select <= 1'b1;
                select_state <= 1'b1;
            end
            
            STATE_SELECT_HI: begin
                // Wait for select line to settle
                state <= STATE_READ_HI;
            end
            
            STATE_READ_HI: begin
                // Read extended buttons with SELECT=1
                // Genesis 6-button: L, R, START, SELECT
                // These are multiplexed on the D-pad pins
                db9_1_l      <= ~db9_1[0];  // UP becomes L
                db9_1_r      <= ~db9_1[1];  // DOWN becomes R
                db9_1_start  <= ~db9_1[2];  // LEFT becomes START
                db9_1_select <= ~db9_1[3];  // RIGHT becomes SELECT
                
                db9_2_l      <= ~db9_2[0];
                db9_2_r      <= ~db9_2[1];
                db9_2_start  <= ~db9_2[2];
                db9_2_select <= ~db9_2[3];
                
                state <= STATE_LATCH;
            end
            
            STATE_LATCH: begin
                // Latch all button states
                buttons1 <= {
                    1'b0,           // Bit 11 (reserved)
                    db9_1_start,
                    db9_1_select,
                    db9_1_r,
                    db9_1_l,
                    db9_1_btn_c,
                    db9_1_btn_b,
                    db9_1_btn_a,
                    db9_1_right,
                    db9_1_left,
                    db9_1_down,
                    db9_1_up
                };
                
                buttons2 <= {
                    1'b0,           // Bit 11 (reserved)
                    db9_2_start,
                    db9_2_select,
                    db9_2_r,
                    db9_2_l,
                    db9_2_btn_c,
                    db9_2_btn_b,
                    db9_2_btn_a,
                    db9_2_right,
                    db9_2_left,
                    db9_2_down,
                    db9_2_up
                };
                
                valid <= 1'b1;
                state <= STATE_IDLE;
                db9_1_select <= 1'b1;
                db9_2_select <= 1'b1;
            end
            
            default: state <= STATE_IDLE;
        endcase
    end
end

//============================================================================
// CPU Register Interface
//============================================================================

always @(posedge clk or negedge reset_n) begin
    if (!reset_n) begin
        reg_rdata <= 16'd0;
    end else begin
        if (reg_re) begin
            case (reg_addr)
                8'h00: reg_rdata <= {4'd0, buttons1};     // Controller 1 buttons
                8'h01: reg_rdata <= {4'd0, buttons2};     // Controller 2 buttons
                8'h02: reg_rdata <= {15'd0, valid};       // Status
                default: reg_rdata <= 16'h0000;
            endcase
        end
    end
end

endmodule
