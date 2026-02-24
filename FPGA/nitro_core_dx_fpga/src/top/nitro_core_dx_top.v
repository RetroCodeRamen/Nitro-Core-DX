//============================================================================
// Nitro-Core-DX Top-Level Module
// Target: Tang Mega 60K (GW5AT-LV60PG484A)
// Description: Complete 16-bit fantasy console implementation
//============================================================================

`timescale 1ns / 1ps

module nitro_core_dx_top (
    // Clock and Reset
    input  wire        clk_50m,         // 50MHz system clock
    input  wire        reset_n,         // Active-low reset
    
    // HDMI Output
    output wire        hdmi_clk_p,
    output wire        hdmi_clk_n,
    output wire        hdmi_d0_p,        // TMDS Data 0 (Blue)
    output wire        hdmi_d0_n,
    output wire        hdmi_d1_p,        // TMDS Data 1 (Green)
    output wire        hdmi_d1_n,
    output wire        hdmi_d2_p,        // TMDS Data 2 (Red)
    output wire        hdmi_d2_n,
    
    // RGB LCD Interface (Optional)
    output wire [7:0]  lcd_r,
    output wire [7:0]  lcd_g,
    output wire [7:0]  lcd_b,
    output wire        lcd_pclk,
    output wire        lcd_hsync,
    output wire        lcd_vsync,
    output wire        lcd_de,
    
    // Audio I2S Output
    output wire        i2s_bclk,
    output wire        i2s_lrclk,
    output wire        i2s_data,
    output wire        i2s_mclk,
    
    // DB9 Controller Interface (Custom Nitro-Core-DX controllers)
    input  wire [8:0]  db9_1,           // Controller 1 DB9 pins
    output wire        db9_1_select,    // Controller 1 SELECT line
    input  wire [8:0]  db9_2,           // Controller 2 DB9 pins
    output wire        db9_2_select,    // Controller 2 SELECT line
    
    // SD Card Interface
    output wire        sd_clk,
    inout  wire        sd_cmd,
    inout  wire [3:0]  sd_dat,
    
    // SPI Flash
    output wire        flash_cs_n,
    output wire        flash_clk,
    output wire        flash_di,
    input  wire        flash_do,
    
    // Status LEDs
    output wire [3:0]  led
);

//============================================================================
// Clock Generation (PLL)
//============================================================================

wire clk_10m;       // CPU clock
wire clk_25m;       // PPU clock
wire clk_74_25m;    // HDMI pixel clock
wire clk_11_2896m;  // Audio MCLK
wire clk_2_8224m;   // Audio BCLK
wire clk_44_1k;     // Audio LRCLK
wire pll_locked;

clock_gen pll_inst (
    .clk_in     (clk_50m),
    .reset_n    (reset_n),
    .clk_10m    (clk_10m),
    .clk_25m    (clk_25m),
    .clk_74_25m (clk_74_25m),
    .clk_11_2896m(clk_11_2896m),
    .clk_2_8224m(clk_2_8224m),
    .clk_44_1k  (clk_44_1k),
    .locked     (pll_locked)
);

wire sys_reset_n = reset_n & pll_locked;

//============================================================================
// System Bus
//============================================================================

// CPU Interface
wire [23:0] cpu_addr;
wire [15:0] cpu_wdata;
wire [15:0] cpu_rdata;
wire        cpu_we;
wire        cpu_re;
wire        cpu_ready;

// PPU Interface
wire [23:0] ppu_vram_addr;
wire [15:0] ppu_vram_wdata;
wire [15:0] ppu_vram_rdata;
wire        ppu_vram_we;
wire        ppu_vram_re;

// APU Interface
wire [15:0] apu_sample_l;
wire [15:0] apu_sample_r;
wire        apu_sample_valid;

// Controller Interface
wire [11:0] controller1_buttons;
wire [11:0] controller2_buttons;
wire        controller_valid;

//============================================================================
// Memory Map Decoding
//============================================================================

// Address space decoding
wire cpu_io_space    = (cpu_addr[23:16] == 8'h00) && (cpu_addr[15] == 1'b1);  // 0x008000-0x00FFFF
wire cpu_vram_space  = (cpu_addr[23:16] == 8'h00) && (cpu_addr[15:14] == 2'b00); // 0x000000-0x003FFF
wire cpu_rom_space   = (cpu_addr[23:16] != 8'h00);  // Banks 1-255

// I/O Register addresses
wire ppu_reg_sel     = cpu_io_space && (cpu_addr[15:8] == 8'h80);
wire apu_reg_sel     = cpu_io_space && (cpu_addr[15:8] == 8'h90);
wire input_reg_sel   = cpu_io_space && (cpu_addr[15:8] == 8'hA0);
wire sys_reg_sel     = cpu_io_space && (cpu_addr[15:8] == 8'h00);

//============================================================================
// CPU Core Instance
//============================================================================

cpu_core cpu_inst (
    .clk        (clk_10m),
    .reset_n    (sys_reset_n),
    
    // Memory interface
    .addr       (cpu_addr),
    .wdata      (cpu_wdata),
    .rdata      (cpu_rdata),
    .we         (cpu_we),
    .re         (cpu_re),
    .ready      (cpu_ready),
    
    // Interrupts
    .irq_n      (~ppu_vblank_irq),
    .nmi_n      (1'b1),
    
    // Status
    .halted     (),
    .error      ()
);

//============================================================================
// PPU (Picture Processing Unit)
//============================================================================

wire [7:0]  ppu_vram_r, ppu_vram_g, ppu_vram_b;
wire        ppu_vram_hsync, ppu_vram_vsync, ppu_vram_de;
wire        ppu_vblank_irq;

ppu_core ppu_inst (
    .clk        (clk_25m),
    .reset_n    (sys_reset_n),
    
    // VRAM interface
    .vram_addr  (ppu_vram_addr),
    .vram_wdata (ppu_vram_wdata),
    .vram_rdata (ppu_vram_rdata),
    .vram_we    (ppu_vram_we),
    .vram_re    (ppu_vram_re),
    
    // Register interface (from CPU)
    .reg_addr   (cpu_addr[7:0]),
    .reg_wdata  (cpu_wdata[7:0]),
    .reg_rdata  (),
    .reg_we     (ppu_reg_sel && cpu_we),
    .reg_re     (ppu_reg_sel && cpu_re),
    
    // Video output (320x200 @ 60Hz)
    .pixel_r    (ppu_vram_r),
    .pixel_g    (ppu_vram_g),
    .pixel_b    (ppu_vram_b),
    .hsync      (ppu_vram_hsync),
    .vsync      (ppu_vram_vsync),
    .de         (ppu_vram_de),
    
    // Interrupt
    .vblank_irq (ppu_vblank_irq)
);

//============================================================================
// Video Output (HDMI + RGB LCD)
//============================================================================

video_output video_inst (
    // Input from PPU (320x200)
    .clk_in     (clk_25m),
    .reset_n    (sys_reset_n),
    .pixel_r    (ppu_vram_r),
    .pixel_g    (ppu_vram_g),
    .pixel_b    (ppu_vram_b),
    .hsync_in   (ppu_vram_hsync),
    .vsync_in   (ppu_vram_vsync),
    .de_in      (ppu_vram_de),
    
    // HDMI Output (720p)
    .clk_hdmi   (clk_74_25m),
    .hdmi_clk_p (hdmi_clk_p),
    .hdmi_clk_n (hdmi_clk_n),
    .hdmi_d0_p  (hdmi_d0_p),
    .hdmi_d0_n  (hdmi_d0_n),
    .hdmi_d1_p  (hdmi_d1_p),
    .hdmi_d1_n  (hdmi_d1_n),
    .hdmi_d2_p  (hdmi_d2_p),
    .hdmi_d2_n  (hdmi_d2_n),
    
    // RGB LCD Output
    .lcd_r      (lcd_r),
    .lcd_g      (lcd_g),
    .lcd_b      (lcd_b),
    .lcd_pclk   (lcd_pclk),
    .lcd_hsync  (lcd_hsync),
    .lcd_vsync  (lcd_vsync),
    .lcd_de     (lcd_de)
);

//============================================================================
// APU (Audio Processing Unit)
//============================================================================

apu_core apu_inst (
    .clk        (clk_10m),
    .reset_n    (sys_reset_n),
    
    // Register interface (from CPU)
    .reg_addr   (cpu_addr[7:0]),
    .reg_wdata  (cpu_wdata[15:0]),
    .reg_rdata  (),
    .reg_we     (apu_reg_sel && cpu_we),
    .reg_re     (apu_reg_sel && cpu_re),
    
    // Audio output
    .sample_l   (apu_sample_l),
    .sample_r   (apu_sample_r),
    .sample_valid(apu_sample_valid)
);

// I2S Audio Interface
i2s_interface i2s_inst (
    .mclk       (clk_11_2896m),
    .bclk       (clk_2_8224m),
    .lrclk      (clk_44_1k),
    .reset_n    (sys_reset_n),
    
    .sample_l   (apu_sample_l),
    .sample_r   (apu_sample_r),
    .sample_valid(apu_sample_valid),
    
    .i2s_bclk   (i2s_bclk),
    .i2s_lrclk  (i2s_lrclk),
    .i2s_data   (i2s_data),
    .i2s_mclk   (i2s_mclk)
);

//============================================================================
// Input Controller Interface (DB9)
//============================================================================

input_controller input_inst (
    .clk        (clk_10m),
    .reset_n    (sys_reset_n),
    
    // DB9 Controller 1 interface
    .db9_1      (db9_1),
    .db9_1_select(db9_1_select),
    
    // DB9 Controller 2 interface
    .db9_2      (db9_2),
    .db9_2_select(db9_2_select),
    
    // Register interface (from CPU)
    .reg_addr   (cpu_addr[7:0]),
    .reg_rdata  (),
    .reg_re     (input_reg_sel && cpu_re),
    
    // Controller outputs
    .buttons1   (controller1_buttons),
    .buttons2   (controller2_buttons),
    .valid      (controller_valid)
);

//============================================================================
// Memory Controller
//============================================================================

memory_controller mem_inst (
    .clk        (clk_50m),
    .reset_n    (sys_reset_n),
    
    // CPU interface
    .cpu_addr   (cpu_addr),
    .cpu_wdata  (cpu_wdata),
    .cpu_rdata  (cpu_rdata),
    .cpu_we     (cpu_we),
    .cpu_re     (cpu_re),
    .cpu_ready  (cpu_ready),
    
    // PPU VRAM interface
    .ppu_addr   (ppu_vram_addr),
    .ppu_wdata  (ppu_vram_wdata),
    .ppu_rdata  (ppu_vram_rdata),
    .ppu_we     (ppu_vram_we),
    .ppu_re     (ppu_vram_re),
    
    // SPI Flash interface (for ROM)
    .flash_cs_n (flash_cs_n),
    .flash_clk  (flash_clk),
    .flash_di   (flash_di),
    .flash_do   (flash_do),
    
    // SD Card interface (for game loading)
    .sd_clk     (sd_clk),
    .sd_cmd     (sd_cmd),
    .sd_dat     (sd_dat)
);

//============================================================================
// Status LEDs
//============================================================================

reg [24:0] led_counter;
always @(posedge clk_50m or negedge sys_reset_n) begin
    if (!sys_reset_n) begin
        led_counter <= 0;
    end else begin
        led_counter <= led_counter + 1;
    end
end

assign led[0] = pll_locked;                          // PLL locked
assign led[1] = ppu_vblank_irq;                      // VBlank indicator
assign led[2] = led_counter[24];                     // Heartbeat
assign led[3] = controller_valid;                    // Controller connected

endmodule
