//============================================================================
// Memory Controller
// Manages access to ROM, WRAM, and VRAM
//============================================================================

`timescale 1ns / 1ps

module memory_controller (
    input  wire        clk,
    input  wire        reset_n,
    
    // CPU interface
    input  wire [23:0] cpu_addr,
    input  wire [15:0] cpu_wdata,
    output reg  [15:0] cpu_rdata,
    input  wire        cpu_we,
    input  wire        cpu_re,
    output reg         cpu_ready,
    
    // PPU VRAM interface
    input  wire [23:0] ppu_addr,
    input  wire [15:0] ppu_wdata,
    output reg  [15:0] ppu_rdata,
    input  wire        ppu_we,
    input  wire        ppu_re,
    
    // SPI Flash interface (for ROM)
    output reg         flash_cs_n,
    output reg         flash_clk,
    output reg         flash_di,
    input  wire        flash_do,
    
    // SD Card interface (for game loading)
    output wire        sd_clk,
    inout  wire        sd_cmd,
    inout  wire [3:0]  sd_dat
);

//============================================================================
// Memory Map
//============================================================================

// Bank 0: WRAM (0x0000-0x7FFF) + I/O (0x8000-0xFFFF)
// Banks 1-255: ROM space

wire cpu_wram_space = (cpu_addr[23:16] == 8'h00) && (cpu_addr[15] == 1'b0);
wire cpu_io_space   = (cpu_addr[23:16] == 8'h00) && (cpu_addr[15] == 1'b1);
wire cpu_rom_space  = (cpu_addr[23:16] != 8'h00);

//============================================================================
// WRAM (32KB)
//============================================================================

reg [15:0] wram [0:16383];  // 32KB / 2 = 16384 words

// WRAM access
wire [13:0] wram_addr = cpu_addr[14:1];
wire [15:0] wram_rdata = wram[wram_addr];

always @(posedge clk) begin
    if (cpu_we && cpu_wram_space) begin
        wram[wram_addr] <= cpu_wdata;
    end
end

//============================================================================
// VRAM (64KB for PPU)
//============================================================================

reg [15:0] vram [0:32767];  // 64KB / 2 = 32768 words

// VRAM access (PPU has priority)
wire [14:0] vram_addr_ppu = ppu_addr[15:1];
wire [14:0] vram_addr_cpu = cpu_addr[15:1];

always @(posedge clk) begin
    // PPU access (priority)
    if (ppu_we) begin
        vram[vram_addr_ppu] <= ppu_wdata;
    end
    
    // CPU access to VRAM
    if (cpu_we && cpu_addr[23:16] == 8'h00 && cpu_addr[15:14] == 2'b00) begin
        vram[vram_addr_cpu] <= cpu_wdata;
    end
end

always @(*) begin
    if (ppu_re) begin
        ppu_rdata = vram[vram_addr_ppu];
    end else begin
        ppu_rdata = 16'd0;
    end
end

//============================================================================
// SPI Flash Interface (for ROM)
//============================================================================

reg [7:0]  spi_state;
reg [23:0] spi_addr;
reg [7:0]  spi_cmd;
reg [15:0] spi_data;
reg [3:0]  spi_bit_cnt;
reg [7:0]  spi_byte_cnt;

localparam SPI_IDLE       = 8'd0;
localparam SPI_SEND_CMD   = 8'd1;
localparam SPI_SEND_ADDR  = 8'd2;
localparam SPI_READ_DATA  = 8'd3;
localparam SPI_DONE       = 8'd4;

// SPI command: READ (0x03)
localparam CMD_READ = 8'h03;

always @(posedge clk or negedge reset_n) begin
    if (!reset_n) begin
        flash_cs_n <= 1'b1;
        flash_clk <= 1'b0;
        flash_di <= 1'b0;
        spi_state <= SPI_IDLE;
        spi_bit_cnt <= 4'd0;
        spi_byte_cnt <= 8'd0;
        cpu_ready <= 1'b1;
    end else begin
        case (spi_state)
            SPI_IDLE: begin
                flash_cs_n <= 1'b1;
                flash_clk <= 1'b0;
                cpu_ready <= 1'b1;
                
                // Start SPI read on ROM access
                if (cpu_re && cpu_rom_space) begin
                    spi_state <= SPI_SEND_CMD;
                    spi_addr <= cpu_addr;
                    spi_cmd <= CMD_READ;
                    spi_bit_cnt <= 4'd0;
                    flash_cs_n <= 1'b0;
                    cpu_ready <= 1'b0;
                end
            end
            
            SPI_SEND_CMD: begin
                // Send READ command (8 bits)
                flash_di <= spi_cmd[7 - spi_bit_cnt];
                flash_clk <= ~flash_clk;
                
                if (flash_clk == 1'b0) begin
                    spi_bit_cnt <= spi_bit_cnt + 1;
                    
                    if (spi_bit_cnt == 7) begin
                        spi_bit_cnt <= 0;
                        spi_state <= SPI_SEND_ADDR;
                    end
                end
            end
            
            SPI_SEND_ADDR: begin
                // Send address (24 bits)
                flash_di <= spi_addr[23 - spi_bit_cnt];
                flash_clk <= ~flash_clk;
                
                if (flash_clk == 1'b0) begin
                    spi_bit_cnt <= spi_bit_cnt + 1;
                    
                    if (spi_bit_cnt == 23) begin
                        spi_bit_cnt <= 0;
                        spi_state <= SPI_READ_DATA;
                        spi_byte_cnt <= 0;
                    end
                end
            end
            
            SPI_READ_DATA: begin
                // Read data (2 bytes for 16-bit word)
                flash_clk <= ~flash_clk;
                
                if (flash_clk == 1'b0) begin
                    // Sample data on rising edge
                    spi_data <= {spi_data[14:0], flash_do};
                    spi_bit_cnt <= spi_bit_cnt + 1;
                    
                    if (spi_bit_cnt == 15) begin
                        spi_bit_cnt <= 0;
                        spi_state <= SPI_DONE;
                    end
                end
            end
            
            SPI_DONE: begin
                flash_cs_n <= 1'b1;
                flash_clk <= 1'b0;
                cpu_ready <= 1'b1;
                spi_state <= SPI_IDLE;
            end
            
            default: spi_state <= SPI_IDLE;
        endcase
    end
end

//============================================================================
// CPU Read Data Mux
//============================================================================

always @(*) begin
    if (cpu_wram_space) begin
        cpu_rdata = wram_rdata;
    end else if (cpu_rom_space && spi_state == SPI_DONE) begin
        cpu_rdata = spi_data;
    end else if (cpu_addr[23:16] == 8'h00 && cpu_addr[15:14] == 2'b00) begin
        // VRAM access
        cpu_rdata = vram[vram_addr_cpu];
    end else begin
        cpu_rdata = 16'h0000;
    end
end

//============================================================================
// SD Card Interface (placeholder)
//============================================================================

assign sd_clk = 1'b0;

endmodule
