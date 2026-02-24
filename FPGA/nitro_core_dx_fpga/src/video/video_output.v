//============================================================================
// Video Output Module
// Scales 320x200 to 720p HDMI and provides RGB LCD output
//============================================================================

`timescale 1ns / 1ps

module video_output (
    // Input from PPU (320x200 @ 60Hz)
    input  wire        clk_in,
    input  wire        reset_n,
    input  wire [7:0]  pixel_r,
    input  wire [7:0]  pixel_g,
    input  wire [7:0]  pixel_b,
    input  wire        hsync_in,
    input  wire        vsync_in,
    input  wire        de_in,
    
    // HDMI Output (1280x720 @ 60Hz)
    input  wire        clk_hdmi,
    output wire        hdmi_clk_p,
    output wire        hdmi_clk_n,
    output wire        hdmi_d0_p,
    output wire        hdmi_d0_n,
    output wire        hdmi_d1_p,
    output wire        hdmi_d1_n,
    output wire        hdmi_d2_p,
    output wire        hdmi_d2_n,
    
    // RGB LCD Output (optional)
    output reg  [7:0]  lcd_r,
    output reg  [7:0]  lcd_g,
    output reg  [7:0]  lcd_b,
    output reg         lcd_pclk,
    output reg         lcd_hsync,
    output reg         lcd_vsync,
    output reg         lcd_de
);

//============================================================================
// Frame Buffer for Scaling
//============================================================================

// Line buffer for scan doubling
reg [23:0] line_buffer [0:319];  // 320 pixels × 24-bit RGB
reg [8:0]  write_ptr;
reg [8:0]  read_ptr;
reg        line_toggle;

// Current input pixel
wire [23:0] pixel_in = {pixel_r, pixel_g, pixel_b};

// Write to line buffer
always @(posedge clk_in) begin
    if (de_in) begin
        line_buffer[write_ptr] <= pixel_in;
        write_ptr <= write_ptr + 1;
    end
    
    if (hsync_in) begin
        write_ptr <= 0;
        line_toggle <= ~line_toggle;
    end
end

//============================================================================
// HDMI Timing (1280x720 @ 60Hz)
//============================================================================

// Horizontal timing
localparam HDMI_H_VISIBLE = 1280;
localparam HDMI_H_FRONT   = 110;
localparam HDMI_H_SYNC    = 40;
localparam HDMI_H_BACK    = 220;
localparam HDMI_H_TOTAL   = HDMI_H_VISIBLE + HDMI_H_FRONT + HDMI_H_SYNC + HDMI_H_BACK;  // 1650

// Vertical timing
localparam HDMI_V_VISIBLE = 720;
localparam HDMI_V_FRONT   = 5;
localparam HDMI_V_SYNC    = 5;
localparam HDMI_V_BACK    = 20;
localparam HDMI_V_TOTAL   = HDMI_V_VISIBLE + HDMI_V_FRONT + HDMI_V_SYNC + HDMI_V_BACK;  // 750

// Scale factors
// 320x200 → 1280x720 = 4x horizontal, 3.6x vertical
// We'll use integer scaling: 4x horizontal, 4x vertical (centered with borders)
// Result: 1280x800, cropped to 1280x720

localparam SCALE_H = 4;
localparam SCALE_V = 4;

localparam SRC_WIDTH  = 320;
localparam SRC_HEIGHT = 200;
localparam DST_WIDTH  = SRC_WIDTH * SCALE_H;   // 1280
localparam DST_HEIGHT = SRC_HEIGHT * SCALE_V;  // 800

// Center vertically in 720p
localparam V_BORDER = (HDMI_V_VISIBLE - DST_HEIGHT) / 2;  // (720 - 800) / 2 = -40 (crop instead)

//============================================================================
// HDMI Output Timing Generation
//============================================================================

reg [10:0] hdmi_h_count;
reg [9:0]  hdmi_v_count;
reg        hdmi_hsync;
reg        hdmi_vsync;
reg        hdmi_de;

// Source coordinates
reg [8:0]  src_x;
reg [7:0]  src_y;
reg [2:0]  scale_h_cnt;
reg [2:0]  scale_v_cnt;

// Output pixel
reg [7:0]  out_r, out_g, out_b;

always @(posedge clk_hdmi or negedge reset_n) begin
    if (!reset_n) begin
        hdmi_h_count <= 0;
        hdmi_v_count <= 0;
        hdmi_hsync <= 1;
        hdmi_vsync <= 1;
        hdmi_de <= 0;
        src_x <= 0;
        src_y <= 0;
        scale_h_cnt <= 0;
        scale_v_cnt <= 0;
    end else begin
        // Horizontal counter
        if (hdmi_h_count == HDMI_H_TOTAL - 1) begin
            hdmi_h_count <= 0;
            
            // Vertical counter
            if (hdmi_v_count == HDMI_V_TOTAL - 1) begin
                hdmi_v_count <= 0;
                src_y <= 0;
                scale_v_cnt <= 0;
            end else begin
                hdmi_v_count <= hdmi_v_count + 1;
                
                // Update vertical scale counter
                if (hdmi_v_count >= HDMI_V_SYNC + HDMI_V_BACK &&
                    hdmi_v_count < HDMI_V_SYNC + HDMI_V_BACK + DST_HEIGHT) begin
                    if (scale_v_cnt == SCALE_V - 1) begin
                        scale_v_cnt <= 0;
                        src_y <= src_y + 1;
                    end else begin
                        scale_v_cnt <= scale_v_cnt + 1;
                    end
                end
            end
        end else begin
            hdmi_h_count <= hdmi_h_count + 1;
        end
        
        // Horizontal sync
        if (hdmi_h_count == HDMI_H_VISIBLE + HDMI_H_FRONT - 1)
            hdmi_hsync <= 0;
        else if (hdmi_h_count == HDMI_H_VISIBLE + HDMI_H_FRONT + HDMI_H_SYNC - 1)
            hdmi_hsync <= 1;
        
        // Vertical sync
        if (hdmi_v_count == HDMI_V_VISIBLE + HDMI_V_FRONT - 1)
            hdmi_vsync <= 0;
        else if (hdmi_v_count == HDMI_V_VISIBLE + HDMI_V_FRONT + HDMI_V_SYNC - 1)
            hdmi_vsync <= 1;
        
        // Data enable and scaling
        if (hdmi_h_count < HDMI_H_VISIBLE && hdmi_v_count < HDMI_V_VISIBLE) begin
            // Check if we're in the active scaled region
            if (hdmi_h_count < DST_WIDTH &&
                hdmi_v_count >= 0 && hdmi_v_count < DST_HEIGHT) begin
                
                hdmi_de <= 1;
                
                // Calculate source X coordinate
                if (hdmi_h_count == 0 || scale_h_cnt == SCALE_H - 1) begin
                    scale_h_cnt <= 0;
                    if (hdmi_h_count < DST_WIDTH)
                        src_x <= hdmi_h_count / SCALE_H;
                end else begin
                    scale_h_cnt <= scale_h_cnt + 1;
                end
                
                // Read from line buffer
                if (src_x < SRC_WIDTH && src_y < SRC_HEIGHT) begin
                    out_r <= line_buffer[src_x][23:16];
                    out_g <= line_buffer[src_x][15:8];
                    out_b <= line_buffer[src_x][7:0];
                end else begin
                    out_r <= 8'd0;
                    out_g <= 8'd0;
                    out_b <= 8'd0;
                end
            end else begin
                // Border area (black)
                hdmi_de <= 0;
                out_r <= 8'd0;
                out_g <= 8'd0;
                out_b <= 8'd0;
            end
        end else begin
            hdmi_de <= 0;
        end
    end
end

//============================================================================
// HDMI TMDS Encoder
//============================================================================

// TMDS encoding for each channel
wire [9:0] tmds_red, tmds_green, tmds_blue;

// Blue channel carries control signals during blanking
wire [1:0] ctrl = {hdmi_vsync, hdmi_hsync};

tmds_encoder encode_red (
    .clk    (clk_hdmi),
    .data   (out_r),
    .ctrl   (2'b00),
    .de     (hdmi_de),
    .tmds   (tmds_red)
);

tmds_encoder encode_green (
    .clk    (clk_hdmi),
    .data   (out_g),
    .ctrl   (2'b00),
    .de     (hdmi_de),
    .tmds   (tmds_green)
);

tmds_encoder encode_blue (
    .clk    (clk_hdmi),
    .data   (out_b),
    .ctrl   (ctrl),
    .de     (hdmi_de),
    .tmds   (tmds_blue)
);

//============================================================================
// TMDS Output Buffers
//============================================================================

// Serialize TMDS data (10:1 serialization at 74.25MHz = 742.5Mbps per channel)
// For simplicity, we'll use DDR output

// TMDS clock (1/10 of bit rate = 74.25MHz)
// This is already our clk_hdmi

// Output the TMDS data directly (simplified - real implementation needs serializer)
assign hdmi_clk_p = clk_hdmi;
assign hdmi_clk_n = ~clk_hdmi;

// For a real implementation, you'd use a serializer PLL
// For now, we'll output the encoded data directly
assign hdmi_d2_p = tmds_red[0];   // Red (MSB of encoded data)
assign hdmi_d2_n = ~tmds_red[0];
assign hdmi_d1_p = tmds_green[0]; // Green
assign hdmi_d1_n = ~tmds_green[0];
assign hdmi_d0_p = tmds_blue[0];  // Blue
assign hdmi_d0_n = ~tmds_blue[0];

//============================================================================
// RGB LCD Output (pass-through)
//============================================================================

always @(posedge clk_in or negedge reset_n) begin
    if (!reset_n) begin
        lcd_r <= 8'd0;
        lcd_g <= 8'd0;
        lcd_b <= 8'd0;
        lcd_pclk <= 0;
        lcd_hsync <= 1;
        lcd_vsync <= 1;
        lcd_de <= 0;
    end else begin
        lcd_r <= pixel_r;
        lcd_g <= pixel_g;
        lcd_b <= pixel_b;
        lcd_pclk <= clk_in;
        lcd_hsync <= hsync_in;
        lcd_vsync <= vsync_in;
        lcd_de <= de_in;
    end
end

endmodule

//============================================================================
// TMDS Encoder Module
//============================================================================

module tmds_encoder (
    input  wire        clk,
    input  wire [7:0]  data,
    input  wire [1:0]  ctrl,
    input  wire        de,
    output reg  [9:0]  tmds
);

// TMDS encoding algorithm
// Based on DVI/HDMI specification

reg [3:0] ones;
reg [8:0] q_m;
reg [4:0] disparity;
reg [9:0] encoded;

integer i;

always @(*) begin
    // Count ones in data
    ones = 0;
    for (i = 0; i < 8; i = i + 1)
        ones = ones + data[i];
    
    // Choose encoding based on ones count
    if (ones > 4 || (ones == 4 && data[0] == 0)) begin
        // XOR encoding
        q_m[0] = data[0];
        for (i = 1; i < 8; i = i + 1)
            q_m[i] = q_m[i-1] ^ data[i];
        q_m[8] = 0;
    end else begin
        // XNOR encoding
        q_m[0] = data[0];
        for (i = 1; i < 8; i = i + 1)
            q_m[i] = ~(q_m[i-1] ^ data[i]);
        q_m[8] = 1;
    end
end

always @(posedge clk) begin
    if (!de) begin
        // Control period encoding
        case (ctrl)
            2'b00: tmds <= 10'b1101010100;
            2'b01: tmds <= 10'b0010101011;
            2'b10: tmds <= 10'b0101010100;
            2'b11: tmds <= 10'b1010101011;
        endcase
        disparity <= 0;
    end else begin
        // Data period encoding
        // Simplified - full implementation needs DC balancing
        tmds <= {~q_m[8], q_m[8], q_m[7:0]};
    end
end

endmodule
