//============================================================================
// I2S Audio Interface
// Converts parallel audio samples to I2S serial format
//============================================================================

`timescale 1ns / 1ps

module i2s_interface (
    input  wire        mclk,        // Master clock (11.2896 MHz)
    input  wire        bclk,        // Bit clock (2.8224 MHz)
    input  wire        lrclk,       // Left/Right clock (44.1 kHz)
    input  wire        reset_n,
    
    // Audio samples
    input  wire [15:0] sample_l,
    input  wire [15:0] sample_r,
    input  wire        sample_valid,
    
    // I2S output
    output reg         i2s_bclk,
    output reg         i2s_lrclk,
    output reg         i2s_data,
    output reg         i2s_mclk
);

//============================================================================
// Clock Output
//============================================================================

always @(*) begin
    i2s_mclk = mclk;
    i2s_bclk = bclk;
    i2s_lrclk = lrclk;
end

//============================================================================
// I2S Data Generation
//============================================================================

reg [15:0] shift_reg;
reg [4:0]  bit_counter;
reg        lrclk_prev;
reg [15:0] current_sample_l;
reg [15:0] current_sample_r;
reg        channel_select;  // 0 = left, 1 = right

always @(posedge bclk or negedge reset_n) begin
    if (!reset_n) begin
        shift_reg <= 16'd0;
        bit_counter <= 5'd0;
        lrclk_prev <= 1'b0;
        i2s_data <= 1'b0;
        channel_select <= 1'b0;
        current_sample_l <= 16'd0;
        current_sample_r <= 16'd0;
    end else begin
        lrclk_prev <= lrclk;
        
        // Detect LRCLK edge (start of new sample period)
        if (lrclk != lrclk_prev) begin
            bit_counter <= 5'd0;
            channel_select <= lrclk;  // LRCLK=0 is left, LRCLK=1 is right
            
            // Load new sample
            if (lrclk == 1'b0) begin
                // Left channel
                shift_reg <= current_sample_l;
            end else begin
                // Right channel
                shift_reg <= current_sample_r;
            end
        end else begin
            // Shift out data (MSB first)
            if (bit_counter < 16) begin
                i2s_data <= shift_reg[15];
                shift_reg <= {shift_reg[14:0], 1'b0};
                bit_counter <= bit_counter + 1;
            end else begin
                i2s_data <= 1'b0;  // Padding
            end
        end
    end
end

//============================================================================
// Sample Buffer
//============================================================================

always @(posedge mclk or negedge reset_n) begin
    if (!reset_n) begin
        current_sample_l <= 16'd0;
        current_sample_r <= 16'd0;
    end else begin
        if (sample_valid) begin
            current_sample_l <= sample_l;
            current_sample_r <= sample_r;
        end
    end
end

endmodule
