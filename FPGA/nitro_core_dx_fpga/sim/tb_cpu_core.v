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

// Test memory (simple RAM)
reg [15:0] test_mem [0:1023];

// Clock generation (10MHz = 100ns period)
initial begin
    clk = 0;
    forever #50 clk = ~clk;
end

// Test sequence
initial begin
    // Initialize
    reset_n = 0;
    ready = 1;
    irq_n = 1;
    nmi_n = 1;
    rdata = 0;
    
    // Initialize test memory with program
    // Program: Load immediate values, add them, store result
    test_mem[0] = 16'h1205;  // MOV R0, #5
    test_mem[1] = 16'h120A;  // MOV R1, #10
    test_mem[2] = 16'h2201;  // ADD R0, R0, R1  (R0 = 15)
    test_mem[3] = 16'hF600;  // HALT
    
    // Reset
    #200;
    reset_n = 1;
    
    // Run for a while
    #5000;
    
    // Check results
    $display("CPU Test Complete");
    $display("R0 = %d (expected 15)", uut.regs[0]);
    $display("R1 = %d (expected 10)", uut.regs[1]);
    
    if (uut.regs[0] == 15 && uut.regs[1] == 10)
        $display("TEST PASSED");
    else
        $display("TEST FAILED");
    
    $finish;
end

// Memory response
always @(posedge clk) begin
    if (re) begin
        // Map address to test memory
        if (addr[15:0] < 1024)
            rdata <= test_mem[addr[9:0]];
        else
            rdata <= 16'h0000;
    end
    
    if (we) begin
        if (addr[15:0] < 1024)
            test_mem[addr[9:0]] <= wdata;
    end
end

// Monitor
always @(posedge clk) begin
    if (reset_n && re)
        $display("Time=%0t: Fetch from addr=%h, data=%h", $time, addr, rdata);
end

endmodule
