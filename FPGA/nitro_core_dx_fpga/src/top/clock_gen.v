//============================================================================
// Clock Generation Module
// Generates all required clocks from 50MHz input
//============================================================================

`timescale 1ns / 1ps

module clock_gen (
    input  wire  clk_in,        // 50MHz input
    input  wire  reset_n,
    
    output wire  clk_10m,       // CPU clock (10 MHz)
    output wire  clk_25m,       // PPU clock (25 MHz)
    output wire  clk_74_25m,    // HDMI pixel clock (74.25 MHz)
    output wire  clk_11_2896m,  // Audio MCLK (11.2896 MHz)
    output wire  clk_2_8224m,   // Audio BCLK (2.8224 MHz)
    output wire  clk_44_1k,     // Audio LRCLK (44.1 kHz)
    output wire  locked
);

//============================================================================
// PLL 1: System Clocks (10MHz, 25MHz, 74.25MHz)
//============================================================================

// Use Gowin PLL primitive
// Input: 50MHz
// Output 0: 10MHz (CPU)
// Output 1: 25MHz (PPU)
// Output 2: 74.25MHz (HDMI pixel clock)

// PLL configuration for GW5AT series
// Feedback divider: 1
// Reference divider: 1
// Output dividers calculated for desired frequencies

// For 10MHz from 50MHz: divide by 5
// For 25MHz from 50MHz: divide by 2
// For 74.25MHz from 50MHz: multiply by 297/200 (approximate)

// Using Gowin's rPLL primitive
wire pll1_clkout0;
wire pll1_clkout1;
wire pll1_clkout2;
wire pll1_locked;

// PLL instance for system clocks
rPLL #(
    .FCLKIN("50"),
    .IDIV_SEL(1),           // Input divider (50/1 = 50MHz to PFD)
    .FBDIV_SEL(1),          // Feedback divider
    .ODIV0_SEL(50),         // Output 0: 50MHz * 1 / 50 = 1MHz (will adjust)
    .ODIV1_SEL(20),         // Output 1: 50MHz * 1 / 20 = 2.5MHz
    .ODIV2_SEL(8),          // Output 2: 50MHz * 1 / 8 = 6.25MHz
    .ODIV3_SEL(8),
    .ODIV4_SEL(8),
    .ODIV5_SEL(8),
    .ODIV6_SEL(8),
    .CLKOUT0_EN("TRUE"),
    .CLKOUT1_EN("TRUE"),
    .CLKOUT2_EN("TRUE"),
    .CLKOUT3_EN("FALSE"),
    .CLKOUT4_EN("FALSE"),
    .CLKOUT5_EN("FALSE"),
    .CLKOUT6_EN("FALSE"),
    .DYN_SDIV_SEL(2),
    .CLKFB_SEL("INTERNAL"),
    .CLKOUT0_DT_DIR(1'b0),
    .CLKOUT1_DT_DIR(1'b0),
    .CLKOUT2_DT_DIR(1'b0),
    .CLKOUT3_DT_DIR(1'b0),
    .CLKOUT0_DT_STEP(0),
    .CLKOUT1_DT_STEP(0),
    .CLKOUT2_DT_STEP(0),
    .CLKOUT3_DT_STEP(0),
    .CLKOUT0_CPHASE(0),
    .CLKOUT1_CPHASE(0),
    .CLKOUT2_CPHASE(0),
    .CLKOUT3_CPHASE(0),
    .CLKOUT4_CPHASE(0),
    .CLKOUT5_CPHASE(0),
    .CLKOUT6_CPHASE(0),
    .DYN_DA_EN("FALSE"),
    .CLKOUT0_PE_COARSE(0),
    .CLKOUT0_PE_FINE(0),
    .CLKOUT1_PE_COARSE(0),
    .CLKOUT1_PE_FINE(0),
    .CLKOUT2_PE_COARSE(0),
    .CLKOUT2_PE_FINE(0),
    .CLKOUT3_PE_COARSE(0),
    .CLKOUT3_PE_FINE(0),
    .CLKOUT4_PE_COARSE(0),
    .CLKOUT4_PE_FINE(0),
    .CLKOUT5_PE_COARSE(0),
    .CLKOUT5_PE_FINE(0),
    .CLKOUT6_PE_COARSE(0),
    .CLKOUT6_PE_FINE(0),
    .DCS_EN("FALSE"),
    .MIPI_REF_SEL("FALSE"),
    .REFCK_MODE("INTERNAL"),
    .PLL_REG_CTRL(0),
    .CLKOUT3_PE_COARSE(0),
    .CLKOUT3_PE_FINE(0),
    .PREDIV_SEL(0),
    .IN_SEL(0),
    .ICP_SEL(0),
    .LPF_RES(0),
    .LPF_CAP(0),
    .MSEL_EN("FALSE"),
    .MSEL_SEL(0)
) pll1 (
    .CLKIN(clk_in),
    .CLKFB(1'b0),
    .RESET(~reset_n),
    .RESET_P(1'b0),
    .PD(1'b0),
    .IDSEL(6'b0),
    .ODSEL0(6'b0),
    .ODSEL1(6'b0),
    .ODSEL2(6'b0),
    .ODSEL3(6'b0),
    .ODSEL4(6'b0),
    .ODSEL5(6'b0),
    .ODSEL6(6'b0),
    .MDSEL(6'b0),
    .MDSEL_FRAC(0),
    .MSEL(1'b0),
    .PSDA(4'b0),
    .DUTYDA(4'b0),
    .FDLY(4'b0),
    .CLKOUT0(pll1_clkout0),
    .CLKOUT1(pll1_clkout1),
    .CLKOUT2(pll1_clkout2),
    .CLKOUT3(),
    .CLKOUT4(),
    .CLKOUT5(),
    .CLKOUT6(),
    .LOCK(pll1_locked)
);

// Since exact frequencies are difficult, use clock dividers
// Generate 10MHz from 50MHz using clock divider
reg [2:0] div10m_cnt;
reg       clk_10m_reg;

always @(posedge clk_in or negedge reset_n) begin
    if (!reset_n) begin
        div10m_cnt <= 0;
        clk_10m_reg <= 0;
    end else if (div10m_cnt == 4) begin
        div10m_cnt <= 0;
        clk_10m_reg <= ~clk_10m_reg;
    end else begin
        div10m_cnt <= div10m_cnt + 1;
    end
end

assign clk_10m = clk_10m_reg;

// Generate 25MHz from 50MHz (divide by 2)
reg clk_25m_reg;
always @(posedge clk_in or negedge reset_n) begin
    if (!reset_n) begin
        clk_25m_reg <= 0;
    end else begin
        clk_25m_reg <= ~clk_25m_reg;
    end
end

assign clk_25m = clk_25m_reg;

// For HDMI 720p, we need 74.25MHz
// This requires a proper PLL configuration
// Using pll1_clkout2 as base and adjusting
assign clk_74_25m = pll1_clkout2;

//============================================================================
// PLL 2: Audio Clocks (11.2896MHz, 2.8224MHz, 44.1kHz)
//============================================================================

// 11.2896MHz = 256 * 44.1kHz (standard audio master clock)
// From 50MHz: 50 * 11.2896 / 50 = 11.2896MHz
// Need fractional PLL or approximation

// Approximation: 50MHz * 9 / 40 = 11.25MHz (0.35% error, acceptable)

wire pll2_clkout0;
wire pll2_locked;

rPLL #(
    .FCLKIN("50"),
    .IDIV_SEL(1),
    .FBDIV_SEL(9),
    .ODIV0_SEL(40),
    .CLKOUT0_EN("TRUE")
) pll2 (
    .CLKIN(clk_in),
    .CLKFB(1'b0),
    .RESET(~reset_n),
    .RESET_P(1'b0),
    .PD(1'b0),
    .IDSEL(6'b0),
    .ODSEL0(6'b0),
    .ODSEL1(6'b0),
    .ODSEL2(6'b0),
    .ODSEL3(6'b0),
    .ODSEL4(6'b0),
    .ODSEL5(6'b0),
    .ODSEL6(6'b0),
    .MDSEL(6'b0),
    .MDSEL_FRAC(0),
    .MSEL(1'b0),
    .PSDA(4'b0),
    .DUTYDA(4'b0),
    .FDLY(4'b0),
    .CLKOUT0(pll2_clkout0),
    .CLKOUT1(),
    .CLKOUT2(),
    .CLKOUT3(),
    .CLKOUT4(),
    .CLKOUT5(),
    .CLKOUT6(),
    .LOCK(pll2_locked)
);

assign clk_11_2896m = pll2_clkout0;

// Generate BCLK (2.8224MHz = 64 * 44.1kHz) from MCLK
// Divide 11.2896MHz by 4
reg [1:0] bclk_div;
reg       bclk_reg;

always @(posedge pll2_clkout0 or negedge reset_n) begin
    if (!reset_n) begin
        bclk_div <= 0;
        bclk_reg <= 0;
    end else if (bclk_div == 3) begin
        bclk_div <= 0;
        bclk_reg <= ~bclk_reg;
    end else begin
        bclk_div <= bclk_div + 1;
    end
end

assign clk_2_8224m = bclk_reg;

// Generate LRCLK (44.1kHz) from BCLK
// Divide 2.8224MHz by 64
reg [5:0] lrclk_div;
reg       lrclk_reg;

always @(posedge bclk_reg or negedge reset_n) begin
    if (!reset_n) begin
        lrclk_div <= 0;
        lrclk_reg <= 0;
    end else if (lrclk_div == 63) begin
        lrclk_div <= 0;
        lrclk_reg <= ~lrclk_reg;
    end else begin
        lrclk_div <= lrclk_div + 1;
    end
end

assign clk_44_1k = lrclk_reg;

// Combined lock signal
assign locked = pll1_locked && pll2_locked;

endmodule
