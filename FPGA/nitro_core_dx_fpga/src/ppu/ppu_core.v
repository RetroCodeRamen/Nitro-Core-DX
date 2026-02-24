//============================================================================
// Nitro-Core-DX PPU (Picture Processing Unit)
// 4 background layers, 128 sprites, matrix mode
//============================================================================

`timescale 1ns / 1ps

module ppu_core (
    input  wire        clk,
    input  wire        reset_n,
    
    // VRAM interface
    output reg  [23:0] vram_addr,
    output reg  [15:0] vram_wdata,
    input  wire [15:0] vram_rdata,
    output reg         vram_we,
    output reg         vram_re,
    
    // Register interface (from CPU)
    input  wire [7:0]  reg_addr,
    input  wire [7:0]  reg_wdata,
    output reg  [7:0]  reg_rdata,
    input  wire        reg_we,
    input  wire        reg_re,
    
    // Video output (320x200 @ 60Hz)
    output reg  [7:0]  pixel_r,
    output reg  [7:0]  pixel_g,
    output reg  [7:0]  pixel_b,
    output reg         hsync,
    output reg         vsync,
    output reg         de,
    
    // Interrupt
    output reg         vblank_irq
);

//============================================================================
// Display Timing (320x200 @ 60Hz)
//============================================================================

// Horizontal timing
localparam H_VISIBLE    = 320;
localparam H_FRONT      = 8;
localparam H_SYNC       = 32;
localparam H_BACK       = 40;
localparam H_TOTAL      = H_VISIBLE + H_FRONT + H_SYNC + H_BACK;  // 400

// Vertical timing
localparam V_VISIBLE    = 200;
localparam V_FRONT      = 3;
localparam V_SYNC       = 6;
localparam V_BACK       = 21;
localparam V_TOTAL      = V_VISIBLE + V_FRONT + V_SYNC + V_BACK;  // 230

// Counters
reg [8:0] h_count;
reg [7:0] v_count;

wire h_active = (h_count < H_VISIBLE);
wire v_active = (v_count < V_VISIBLE);
wire active = h_active && v_active;

wire h_sync_start = (h_count == H_VISIBLE + H_FRONT);
wire h_sync_end   = (h_count == H_VISIBLE + H_FRONT + H_SYNC);
wire v_sync_start = (v_count == V_VISIBLE + V_FRONT) && h_sync_start;
wire v_sync_end   = (v_count == V_VISIBLE + V_FRONT + V_SYNC) && h_sync_start;

//============================================================================
// PPU Registers
//============================================================================

// Control registers
reg [7:0] ppuctrl;      // PPU control
reg [7:0] ppumask;      // PPU mask (enable layers)
reg [7:0] ppustatus;    // PPU status

// Scroll registers for each layer
reg [15:0] scroll_x [0:3];  // BG0-BG3 scroll X
reg [15:0] scroll_y [0:3];  // BG0-BG3 scroll Y

// Matrix mode registers
reg [15:0] matrix_a, matrix_b, matrix_c;
reg [15:0] matrix_d, matrix_e, matrix_f;
reg        matrix_enable;

// CGRAM (Color Palette RAM) - 256 colors × 2 bytes (RGB555)
reg [15:0] cgram [0:255];

// OAM (Object Attribute Memory) - 128 sprites
reg [63:0] oam [0:127];  // Each sprite: 64 bits

// VBlank flag (one-shot, cleared on read)
reg vblank_flag;
reg [15:0] frame_counter;

//============================================================================
// VRAM Organization
//============================================================================

// VRAM layout:
// 0x0000-0x0FFF: BG0 tilemap (64×64 = 4096 tiles)
// 0x1000-0x1FFF: BG1 tilemap
// 0x2000-0x2FFF: BG2 tilemap
// 0x3000-0x3FFF: BG3 tilemap
// 0x4000-0x7FFF: Pattern tables (tile graphics)
// 0x8000-0xBFFF: Sprite patterns

localparam VRAM_BG0_TILEMAP = 16'h0000;
localparam VRAM_BG1_TILEMAP = 16'h1000;
localparam VRAM_BG2_TILEMAP = 16'h2000;
localparam VRAM_BG3_TILEMAP = 16'h3000;
localparam VRAM_PATTERN_TBL = 16'h4000;
localparam VRAM_SPRITE_PAT  = 16'h8000;

//============================================================================
// Background Layer Rendering
//============================================================================

// Current tile being rendered
reg [5:0]  tile_x;
reg [5:0]  tile_y;
reg [11:0] tile_index;
reg [15:0] tile_data;

// Pixel within tile
reg [2:0] pixel_x;
reg [2:0] pixel_y;

// Layer priority and color
reg [7:0] layer_color [0:3];
reg       layer_opaque [0:3];
reg [1:0] layer_priority [0:3];

// Background rendering state machine
reg [2:0] bg_state;
localparam BG_IDLE      = 3'd0;
localparam BG_READ_TILE = 3'd1;
localparam BG_READ_PAT  = 3'd2;
localparam BG_RENDER    = 3'd3;

//============================================================================
// Sprite Rendering
//============================================================================

// Sprite evaluation
reg [6:0] sprite_index;
reg [7:0] sprite_x [0:127];
reg [7:0] sprite_y [0:127];
reg       sprite_active [0:127];

// Sprite buffer for current scanline
reg [7:0] sprite_line_x [0:31];  // Max 32 sprites per line
reg [7:0] sprite_line_color [0:31];
reg [4:0] sprite_line_count;

//============================================================================
// Matrix Mode (Mode 7-style)
//============================================================================

// Matrix transformation for BG0
wire signed [15:0] mtx_a = $signed(matrix_a);
wire signed [15:0] mtx_b = $signed(matrix_b);
wire signed [15:0] mtx_c = $signed(matrix_c);
wire signed [15:0] mtx_d = $signed(matrix_d);

reg signed [23:0] matrix_x;
reg signed [23:0] matrix_y;
reg signed [23:0] matrix_hx;
reg signed [23:0] matrix_hy;

// Apply matrix transformation
wire signed [23:0] transformed_x = (mtx_a * h_count + mtx_b * v_count) >>> 8;
wire signed [23:0] transformed_y = (mtx_c * h_count + mtx_d * v_count) >>> 8;

//============================================================================
// Register Access
//============================================================================

always @(posedge clk or negedge reset_n) begin
    if (!reset_n) begin
        ppuctrl <= 8'h00;
        ppumask <= 8'h00;
        ppustatus <= 8'h00;
        
        scroll_x[0] <= 16'd0;
        scroll_x[1] <= 16'd0;
        scroll_x[2] <= 16'd0;
        scroll_x[3] <= 16'd0;
        scroll_y[0] <= 16'd0;
        scroll_y[1] <= 16'd0;
        scroll_y[2] <= 16'd0;
        scroll_y[3] <= 16'd0;
        
        matrix_a <= 16'h0100;  // Identity matrix
        matrix_b <= 16'h0000;
        matrix_c <= 16'h0000;
        matrix_d <= 16'h0100;
        matrix_e <= 16'h0000;
        matrix_f <= 16'h0000;
        matrix_enable <= 1'b0;
        
        vblank_flag <= 1'b0;
        frame_counter <= 16'd0;
    end else begin
        // Register writes
        if (reg_we) begin
            case (reg_addr)
                8'h00: ppuctrl <= reg_wdata;
                8'h01: ppumask <= reg_wdata;
                
                // Scroll registers
                8'h10: scroll_x[0][7:0]  <= reg_wdata;
                8'h11: scroll_x[0][15:8] <= reg_wdata;
                8'h12: scroll_y[0][7:0]  <= reg_wdata;
                8'h13: scroll_y[0][15:8] <= reg_wdata;
                8'h14: scroll_x[1][7:0]  <= reg_wdata;
                8'h15: scroll_x[1][15:8] <= reg_wdata;
                8'h16: scroll_y[1][7:0]  <= reg_wdata;
                8'h17: scroll_y[1][15:8] <= reg_wdata;
                8'h18: scroll_x[2][7:0]  <= reg_wdata;
                8'h19: scroll_x[2][15:8] <= reg_wdata;
                8'h1A: scroll_y[2][7:0]  <= reg_wdata;
                8'h1B: scroll_y[2][15:8] <= reg_wdata;
                8'h1C: scroll_x[3][7:0]  <= reg_wdata;
                8'h1D: scroll_x[3][15:8] <= reg_wdata;
                8'h1E: scroll_y[3][7:0]  <= reg_wdata;
                8'h1F: scroll_y[3][15:8] <= reg_wdata;
                
                // Matrix registers
                8'h20: matrix_a[7:0]  <= reg_wdata;
                8'h21: matrix_a[15:8] <= reg_wdata;
                8'h22: matrix_b[7:0]  <= reg_wdata;
                8'h23: matrix_b[15:8] <= reg_wdata;
                8'h24: matrix_c[7:0]  <= reg_wdata;
                8'h25: matrix_c[15:8] <= reg_wdata;
                8'h26: matrix_d[7:0]  <= reg_wdata;
                8'h27: matrix_d[15:8] <= reg_wdata;
                8'h28: matrix_e[7:0]  <= reg_wdata;
                8'h29: matrix_e[15:8] <= reg_wdata;
                8'h2A: matrix_f[7:0]  <= reg_wdata;
                8'h2B: matrix_f[15:8] <= reg_wdata;
                8'h2C: matrix_enable  <= reg_wdata[0];
                
                // CGRAM access
                8'h40: begin
                    // CGRAM data write (address set separately)
                end
                
                default: begin
                    // Unknown register
                end
            endcase
        end
        
        // Register reads
        if (reg_re) begin
            case (reg_addr)
                8'h00: reg_rdata <= ppuctrl;
                8'h01: reg_rdata <= ppumask;
                8'h02: begin
                    reg_rdata <= {vblank_flag, 7'b0};
                    vblank_flag <= 1'b0;  // Clear on read
                end
                8'h03: reg_rdata <= frame_counter[7:0];
                8'h04: reg_rdata <= frame_counter[15:8];
                default: reg_rdata <= 8'h00;
            endcase
        end
        
        // Frame counter increment at VBlank
        if (v_count == V_VISIBLE && h_count == 0) begin
            frame_counter <= frame_counter + 1;
            vblank_flag <= 1'b1;
        end
    end
end

//============================================================================
// Display Timing Generation
//============================================================================

always @(posedge clk or negedge reset_n) begin
    if (!reset_n) begin
        h_count <= 0;
        v_count <= 0;
        hsync <= 1'b1;
        vsync <= 1'b1;
        de <= 1'b0;
        vblank_irq <= 1'b0;
    end else begin
        // Horizontal counter
        if (h_count == H_TOTAL - 1) begin
            h_count <= 0;
            
            // Vertical counter
            if (v_count == V_TOTAL - 1) begin
                v_count <= 0;
            end else begin
                v_count <= v_count + 1;
            end
        end else begin
            h_count <= h_count + 1;
        end
        
        // Horizontal sync
        if (h_sync_start)
            hsync <= 1'b0;
        else if (h_sync_end)
            hsync <= 1'b1;
        
        // Vertical sync
        if (v_sync_start)
            vsync <= 1'b0;
        else if (v_sync_end)
            vsync <= 1'b1;
        
        // Data enable
        de <= active;
        
        // VBlank interrupt (at start of VBlank)
        if (v_count == V_VISIBLE && h_count == 0)
            vblank_irq <= 1'b1;
        else
            vblank_irq <= 1'b0;
    end
end

//============================================================================
// Background Layer Rendering
//============================================================================

// Render 4 background layers
// Each layer has independent scroll and priority

task render_background;
    input [1:0] layer;
    input [8:0] screen_x;
    input [7:0] screen_y;
    output [7:0] color;
    output       opaque;
    begin
        reg [15:0] scroll_pos_x;
        reg [15:0] scroll_pos_y;
        reg [23:0] tilemap_addr;
        reg [15:0] tile_info;
        reg [10:0] pattern_addr;
        reg [7:0]  pattern_data;
        
        // Calculate scrolled position
        if (layer == 2'd0 && matrix_enable) begin
            // Matrix mode for BG0
            scroll_pos_x = transformed_x[15:0];
            scroll_pos_y = transformed_y[15:0];
        end else begin
            scroll_pos_x = screen_x + scroll_x[layer];
            scroll_pos_y = screen_y + scroll_y[layer];
        end
        
        // Calculate tile coordinates
        tile_x = scroll_pos_x[8:3];  // Divide by 8 (8x8 tiles)
        tile_y = scroll_pos_y[7:3];
        pixel_x = scroll_pos_x[2:0];
        pixel_y = scroll_pos_y[2:0];
        
        // Read tilemap
        case (layer)
            2'd0: tilemap_addr = VRAM_BG0_TILEMAP;
            2'd1: tilemap_addr = VRAM_BG1_TILEMAP;
            2'd2: tilemap_addr = VRAM_BG2_TILEMAP;
            2'd3: tilemap_addr = VRAM_BG3_TILEMAP;
        endcase
        
        tilemap_addr = tilemap_addr + {tile_y, tile_x};
        
        // Fetch tile data (simplified - would need state machine for real VRAM)
        tile_info = 16'h0000;  // Placeholder
        
        // Calculate pattern address
        pattern_addr = VRAM_PATTERN_TBL + (tile_info[9:0] * 64) + {pixel_y, pixel_x};
        
        // Get pixel color
        pattern_data = 8'h00;  // Placeholder
        
        color = pattern_data;
        opaque = (pattern_data != 8'h00);
    end
endtask

//============================================================================
// Sprite Rendering
//============================================================================

task evaluate_sprites;
    input [7:0] scanline;
    begin
        reg [6:0] i;
        reg [7:0] sprite_y_pos;
        reg [7:0] sprite_height;
        
        sprite_line_count = 0;
        
        for (i = 0; i < 128; i = i + 1) begin
            // Read sprite Y position from OAM
            sprite_y_pos = oam[i][39:32];
            sprite_height = 8;  // 8x8 sprites (configurable)
            
            // Check if sprite is on this scanline
            if (scanline >= sprite_y_pos && scanline < sprite_y_pos + sprite_height) begin
                if (sprite_line_count < 32) begin
                    sprite_line_x[sprite_line_count] = oam[i][55:48];
                    sprite_line_color[sprite_line_count] = 8'hFF;  // Placeholder
                    sprite_line_count = sprite_line_count + 1;
                end
            end
        end
    end
endtask

//============================================================================
// Pixel Pipeline
//============================================================================

// Combine all layers and sprites to produce final pixel
always @(posedge clk) begin
    reg [7:0] bg_color [0:3];
    reg       bg_opaque [0:3];
    reg [7:0] sprite_color;
    reg       sprite_opaque;
    reg [7:0] final_color;
    reg [15:0] rgb555;
    
    if (active) begin
        // Render all background layers
        render_background(2'd0, h_count, v_count, bg_color[0], bg_opaque[0]);
        render_background(2'd1, h_count, v_count, bg_color[1], bg_opaque[1]);
        render_background(2'd2, h_count, v_count, bg_color[2], bg_opaque[2]);
        render_background(2'd3, h_count, v_count, bg_color[3], bg_opaque[3]);
        
        // Priority mixing (BG3 > BG2 > BG1 > BG0 > Backdrop)
        if (bg_opaque[3] && ppumask[3])
            final_color = bg_color[3];
        else if (bg_opaque[2] && ppumask[2])
            final_color = bg_color[2];
        else if (bg_opaque[1] && ppumask[1])
            final_color = bg_color[1];
        else if (bg_opaque[0] && ppumask[0])
            final_color = bg_color[0];
        else
            final_color = 8'h00;  // Backdrop color
        
        // Look up in CGRAM (RGB555 to RGB888 conversion)
        rgb555 = cgram[final_color];
        
        // Convert RGB555 to RGB888
        pixel_r <= {rgb555[14:10], rgb555[14:12]};  // 5 bits -> 8 bits
        pixel_g <= {rgb555[9:5],   rgb555[9:7]  };
        pixel_b <= {rgb555[4:0],   rgb555[4:2]  };
    end else begin
        pixel_r <= 8'd0;
        pixel_g <= 8'd0;
        pixel_b <= 8'd0;
    end
end

endmodule
