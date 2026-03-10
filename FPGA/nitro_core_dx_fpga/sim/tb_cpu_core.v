//============================================================================
// CPU Core Testbench
//============================================================================

`timescale 1ns / 1ps

module tb_cpu_core;

// Clock and reset
reg clk;
reg reset_n;

// Memory interface
wire [23:0] addr;
wire [15:0] wdata;
reg  [15:0] rdata;
wire        we;
wire        re;
reg         ready;

// Interrupts
reg irq_n;
reg nmi_n;

// Status
wire halted;
wire error;

// Instantiate CPU
cpu_core uut (
    .clk    (clk),
    .reset_n(reset_n),
    .addr   (addr),
    .wdata  (wdata),
    .rdata  (rdata),
    .we     (we),
    .re     (re),
    .ready  (ready),
    .irq_n  (irq_n),
    .nmi_n  (nmi_n),
    .halted (halted),
    .error  (error)
);

// Test memory (simple ROM image in word-addressed slots)
reg [15:0] test_mem [0:1023];
integer k;
integer idx;

// Clock generation (10MHz = 100ns period)
initial begin
    clk = 0;
    forever #50 clk = ~clk;
end

// Test sequence
initial begin
    // Initialize
    reset_n = 0;
    ready = 0;
    irq_n = 1;
    nmi_n = 1;
    rdata = 0;
    
    // Clear test memory
    for (k = 0; k < 1024; k = k + 1)
        test_mem[k] = 16'h0000;

    // Program (emulator ISA encoding):
    // R1 = 0xF00F; R2 = 4; SAR R1,R2 => 0xFF00
    // R3 = 0x007F; ADD.B R3,#1 => 0x0080
    // R4 = 0x0010; R5 = 0x0020; SUB.B R4,R5 => 0x00F0
    // HALT
    test_mem[0]  = 16'h1110;  // MOV mode1, R1, #imm
    test_mem[1]  = 16'hF00F;  // imm
    test_mem[2]  = 16'h1120;  // MOV mode1, R2, #imm
    test_mem[3]  = 16'h0004;  // imm
    test_mem[4]  = 16'hB212;  // SHR family mode2 (SAR), R1, R2
    test_mem[5]  = 16'h1130;  // MOV mode1, R3, #imm
    test_mem[6]  = 16'h007F;  // imm
    test_mem[7]  = 16'h2330;  // ADD mode3 (ADD.B #imm), R3
    test_mem[8]  = 16'h0001;  // imm
    test_mem[9]  = 16'h1140;  // MOV mode1, R4, #imm
    test_mem[10] = 16'h0010;  // imm
    test_mem[11] = 16'h1150;  // MOV mode1, R5, #imm
    test_mem[12] = 16'h0020;  // imm
    test_mem[13] = 16'h3245;  // SUB mode2 (SUB.B), R4, R5
    test_mem[14] = 16'hF600;  // HALT
    
    // Reset
    #200;
    reset_n = 1;
    
    // Run for a while
    #12000;
    
    // Check results
    $display("CPU Test Complete");
    $display("R1 = 0x%04h (expected 0xFF00)", uut.regs[1]);
    $display("R3 = 0x%04h (expected 0x0080)", uut.regs[3]);
    $display("R4 = 0x%04h (expected 0x00F0)", uut.regs[4]);
    
    if (uut.regs[1] == 16'hFF00 &&
        uut.regs[3] == 16'h0080 &&
        uut.regs[4] == 16'h00F0)
        $display("TEST PASSED");
    else
        $display("TEST FAILED");
    
    $finish;
end

// Memory response
always @(posedge clk) begin
    idx = (addr[15:0] - 16'h8000) >> 1;

    // 1-cycle memory handshake: acknowledge previous-cycle requests.
    ready <= re || we;

    if (re) begin
        // Map ROM window [bank:0x8000+] to word-indexed test memory.
        if (addr[23:16] == 8'h01 && addr[15:0] >= 16'h8000 && idx >= 0 && idx < 1024)
            rdata <= test_mem[idx];
        else
            rdata <= 16'h0000;
    end
    
    if (we) begin
        if (addr[23:16] == 8'h01 && addr[15:0] >= 16'h8000 && idx >= 0 && idx < 1024)
            test_mem[idx] <= wdata;
    end
end

// Monitor
always @(posedge clk) begin
    if (reset_n && re)
        $display("Time=%0t: Fetch from addr=%h, data=%h", $time, addr, rdata);
end

endmodule
