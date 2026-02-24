//============================================================================
// Nitro-Core-DX APU (Audio Processing Unit)
// 4 channels: Sine, Square, Saw, Noise
// 44.1kHz sample rate
//============================================================================

`timescale 1ns / 1ps

module apu_core (
    input  wire        clk,
    input  wire        reset_n,
    
    // Register interface (from CPU)
    input  wire [7:0]  reg_addr,
    input  wire [15:0] reg_wdata,
    output reg  [15:0] reg_rdata,
    input  wire        reg_we,
    input  wire        reg_re,
    
    // Audio output (to I2S interface)
    output reg  [15:0] sample_l,
    output reg  [15:0] sample_r,
    output reg         sample_valid
);

//============================================================================
// APU Registers
//============================================================================

// Channel control registers
reg [7:0]  ch_ctrl    [0:3];  // Channel control (enable, waveform select)
reg [15:0] ch_freq    [0:3];  // Channel frequency
reg [15:0] ch_phase   [0:3];  // Channel phase accumulator
reg [7:0]  ch_vol     [0:3];  // Channel volume (0-255)
reg [7:0]  ch_pan     [0:3];  // Channel panning (0=left, 128=center, 255=right)
reg [15:0] ch_duty    [0:3];  // Duty cycle for square wave

// Master control
reg [7:0]  master_vol;         // Master volume
reg [7:0]  completion_status;  // Channel completion flags

// Sample counter for 44.1kHz
reg [7:0] sample_counter;
localparam SAMPLE_PERIOD = 227;  // 10MHz / 44.1kHz ≈ 227

//============================================================================
// Waveform Generation
//============================================================================

// Waveform lookup tables
// Sine table (256 entries, 8-bit signed)
reg signed [7:0] sine_table [0:255];

// Initialize sine table
integer i;
initial begin
    for (i = 0; i < 256; i = i + 1) begin
        sine_table[i] = $sin(2.0 * 3.14159 * i / 256.0) * 127;
    end
end

// LFSR for noise generation
reg [15:0] noise_lfsr [0:3];

//============================================================================
// Channel Output Calculation
//============================================================================

wire signed [15:0] ch_out [0:3];
reg signed [15:0] ch_wave [0:3];

// Generate waveform for each channel
genvar ch;
generate
    for (ch = 0; ch < 4; ch = ch + 1) begin : channel_gen
        
        // Phase accumulator (NCO)
        reg [15:0] phase_acc;
        reg [7:0]  phase_addr;
        
        always @(posedge clk or negedge reset_n) begin
            if (!reset_n) begin
                phase_acc <= 16'd0;
                noise_lfsr[ch] <= 16'hACE1;  // Seed value
            end else if (ch_ctrl[ch][0]) begin  // Channel enabled
                // Update phase accumulator
                phase_acc <= phase_acc + ch_freq[ch];
                
                // Update noise LFSR
                if (ch_ctrl[ch][3:1] == 3'b100) begin  // Noise waveform
                    noise_lfsr[ch] <= {noise_lfsr[ch][14:0], 
                                       noise_lfsr[ch][15] ^ noise_lfsr[ch][14]};
                end
            end
        end
        
        // Get phase address (upper 8 bits of accumulator)
        always @(*) begin
            phase_addr = phase_acc[15:8];
            
            // Waveform selection
            case (ch_ctrl[ch][3:1])
                3'b000: begin  // Sine
                    ch_wave[ch] = sine_table[phase_addr];
                end
                
                3'b001: begin  // Square
                    if (phase_addr < ch_duty[ch][7:0])
                        ch_wave[ch] = 8'sd127;
                    else
                        ch_wave[ch] = -8'sd128;
                end
                
                3'b010: begin  // Saw
                    ch_wave[ch] = $signed(phase_addr) - 8'sd128;
                end
                
                3'b011: begin  // Triangle
                    if (phase_addr < 8'd128)
                        ch_wave[ch] = ($signed(phase_addr) << 1) - 8'sd128;
                    else
                        ch_wave[ch] = 8'sd127 - (($signed(phase_addr) - 8'sd128) << 1);
                end
                
                3'b100: begin  // Noise
                    ch_wave[ch] = $signed(noise_lfsr[ch][7:0]) >>> 1;
                end
                
                default: ch_wave[ch] = 8'sd0;
            endcase
        end
        
        // Apply volume
        // ch_out = ch_wave * volume / 256
        assign ch_out[ch] = (ch_wave[ch] * $signed({1'b0, ch_vol[ch]})) >>> 8;
        
    end
endgenerate

//============================================================================
// Register Access
//============================================================================

always @(posedge clk or negedge reset_n) begin
    if (!reset_n) begin
        // Reset all channels
        ch_ctrl[0] <= 8'h00;
        ch_ctrl[1] <= 8'h00;
        ch_ctrl[2] <= 8'h00;
        ch_ctrl[3] <= 8'h00;
        
        ch_freq[0] <= 16'd0;
        ch_freq[1] <= 16'd0;
        ch_freq[2] <= 16'd0;
        ch_freq[3] <= 16'd0;
        
        ch_vol[0] <= 8'hFF;
        ch_vol[1] <= 8'hFF;
        ch_vol[2] <= 8'hFF;
        ch_vol[3] <= 8'hFF;
        
        ch_pan[0] <= 8'h80;  // Center
        ch_pan[1] <= 8'h80;
        ch_pan[2] <= 8'h80;
        ch_pan[3] <= 8'h80;
        
        ch_duty[0] <= 16'h8000;  // 50% duty
        ch_duty[1] <= 16'h8000;
        ch_duty[2] <= 16'h8000;
        ch_duty[3] <= 16'h8000;
        
        master_vol <= 8'hFF;
        completion_status <= 8'h00;
        
        sample_counter <= 8'd0;
        sample_valid <= 1'b0;
    end else begin
        // Register writes
        if (reg_we) begin
            case (reg_addr)
                // Channel 0 registers
                8'h00: ch_ctrl[0] <= reg_wdata[7:0];
                8'h01: ch_freq[0][7:0]  <= reg_wdata[7:0];
                8'h02: ch_freq[0][15:8] <= reg_wdata[7:0];
                8'h03: ch_vol[0] <= reg_wdata[7:0];
                8'h04: ch_pan[0] <= reg_wdata[7:0];
                8'h05: ch_duty[0][7:0]  <= reg_wdata[7:0];
                8'h06: ch_duty[0][15:8] <= reg_wdata[7:0];
                
                // Channel 1 registers
                8'h08: ch_ctrl[1] <= reg_wdata[7:0];
                8'h09: ch_freq[1][7:0]  <= reg_wdata[7:0];
                8'h0A: ch_freq[1][15:8] <= reg_wdata[7:0];
                8'h0B: ch_vol[1] <= reg_wdata[7:0];
                8'h0C: ch_pan[1] <= reg_wdata[7:0];
                8'h0D: ch_duty[1][7:0]  <= reg_wdata[7:0];
                8'h0E: ch_duty[1][15:8] <= reg_wdata[7:0];
                
                // Channel 2 registers
                8'h10: ch_ctrl[2] <= reg_wdata[7:0];
                8'h11: ch_freq[2][7:0]  <= reg_wdata[7:0];
                8'h12: ch_freq[2][15:8] <= reg_wdata[7:0];
                8'h13: ch_vol[2] <= reg_wdata[7:0];
                8'h14: ch_pan[2] <= reg_wdata[7:0];
                8'h15: ch_duty[2][7:0]  <= reg_wdata[7:0];
                8'h16: ch_duty[2][15:8] <= reg_wdata[7:0];
                
                // Channel 3 registers
                8'h18: ch_ctrl[3] <= reg_wdata[7:0];
                8'h19: ch_freq[3][7:0]  <= reg_wdata[7:0];
                8'h1A: ch_freq[3][15:8] <= reg_wdata[7:0];
                8'h1B: ch_vol[3] <= reg_wdata[7:0];
                8'h1C: ch_pan[3] <= reg_wdata[7:0];
                8'h1D: ch_duty[3][7:0]  <= reg_wdata[7:0];
                8'h1E: ch_duty[3][15:8] <= reg_wdata[7:0];
                
                // Master control
                8'h20: master_vol <= reg_wdata[7:0];
                
                default: begin
                    // Unknown register
                end
            endcase
        end
        
        // Register reads
        if (reg_re) begin
            case (reg_addr)
                8'h00: reg_rdata <= {8'd0, ch_ctrl[0]};
                8'h01: reg_rdata <= {8'd0, ch_freq[0][7:0]};
                8'h02: reg_rdata <= {8'd0, ch_freq[0][15:8]};
                8'h03: reg_rdata <= {8'd0, ch_vol[0]};
                8'h21: begin
                    reg_rdata <= {8'd0, completion_status};
                    completion_status <= 8'h00;  // Clear on read
                end
                default: reg_rdata <= 16'h0000;
            endcase
        end
        
        // Sample generation at 44.1kHz
        sample_valid <= 1'b0;
        if (sample_counter >= SAMPLE_PERIOD) begin
            sample_counter <= 8'd0;
            sample_valid <= 1'b1;
            
            // Mix all channels
            reg signed [31:0] mix_l, mix_r;
            reg signed [15:0] ch_l, ch_r;
            reg [7:0] pan_inv;
            
            mix_l = 0;
            mix_r = 0;
            
            // Mix each channel with panning
            for (i = 0; i < 4; i = i + 1) begin
                if (ch_ctrl[i][0]) begin  // Channel enabled
                    pan_inv = 8'hFF - ch_pan[i];
                    
                    // Calculate left and right channels based on pan
                    ch_l = (ch_out[i] * $signed({1'b0, pan_inv})) >>> 8;
                    ch_r = (ch_out[i] * $signed({1'b0, ch_pan[i]})) >>> 8;
                    
                    mix_l = mix_l + ch_l;
                    mix_r = mix_r + ch_r;
                end
            end
            
            // Apply master volume and clamp
            mix_l = (mix_l * $signed({1'b0, master_vol})) >>> 8;
            mix_r = (mix_r * $signed({1'b0, master_vol})) >>> 8;
            
            // Clamp to 16-bit range
            if (mix_l > 32767) mix_l = 32767;
            if (mix_l < -32768) mix_l = -32768;
            if (mix_r > 32767) mix_r = 32767;
            if (mix_r < -32768) mix_r = -32768;
            
            sample_l <= mix_l[15:0];
            sample_r <= mix_r[15:0];
            
            // Update completion status (example: when channels finish)
            // In a real implementation, this would track envelope completion
            completion_status <= 8'h00;
            
        end else begin
            sample_counter <= sample_counter + 1;
        end
    end
end

endmodule
