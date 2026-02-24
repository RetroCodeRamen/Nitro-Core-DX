//============================================================================
// Nitro-Core-DX CPU Core
// 16-bit custom instruction set processor
// 8 general-purpose registers, banked memory addressing
//============================================================================

`timescale 1ns / 1ps

module cpu_core (
    input  wire        clk,
    input  wire        reset_n,
    
    // Memory interface
    output reg  [23:0] addr,        // 24-bit address (8-bit bank + 16-bit offset)
    output reg  [15:0] wdata,       // Write data
    input  wire [15:0] rdata,       // Read data
    output reg         we,          // Write enable
    output reg         re,          // Read enable
    input  wire        ready,       // Memory ready
    
    // Interrupts
    input  wire        irq_n,       // Interrupt request (active low)
    input  wire        nmi_n,       // Non-maskable interrupt (active low)
    
    // Status
    output wire        halted,
    output wire        error
);

//============================================================================
// CPU State
//============================================================================

// Program Counter
reg [7:0]  pc_bank;     // Program bank register
reg [15:0] pc_offset;   // Program counter offset

// Data Bank Register
reg [7:0]  db_bank;     // Data bank register

// Stack Pointer
reg [15:0] sp;          // Stack pointer

// General Purpose Registers R0-R7
reg [15:0] regs [0:7];

// Flags Register
// Bit 0: Z (Zero)
// Bit 1: N (Negative)
// Bit 2: C (Carry)
// Bit 3: V (Overflow)
// Bit 4: I (Interrupt mask)
reg [4:0] flags;

// Internal state
reg [2:0] state;
localparam STATE_FETCH   = 3'd0;
localparam STATE_DECODE  = 3'd1;
localparam STATE_EXECUTE = 3'd2;
localparam STATE_MEMORY  = 3'd3;
localparam STATE_WRITEBACK = 3'd4;
localparam STATE_INTERRUPT = 3'd5;

// Instruction register
reg [15:0] ir;

// Effective address
reg [23:0] effective_addr;

// ALU result
reg [15:0] alu_result;
reg        alu_carry;
reg        alu_overflow;

// Interrupt handling
reg        irq_pending;
reg        nmi_pending;
reg [7:0]  interrupt_vector_bank;
reg [15:0] interrupt_vector_offset;

//============================================================================
// Flag Access
//============================================================================

wire flag_z = flags[0];
wire flag_n = flags[1];
wire flag_c = flags[2];
wire flag_v = flags[3];
wire flag_i = flags[4];

//============================================================================
// Instruction Decoding
//============================================================================

// Instruction format: [15:12] = opcode, [11:0] = operands
wire [3:0] opcode = ir[15:12];
wire [2:0] rd     = ir[11:9];   // Destination register
wire [2:0] rs     = ir[8:6];    // Source register
wire [2:0] rt     = ir[5:3];    // Third register
wire [5:0] imm6   = ir[5:0];    // 6-bit immediate
wire [8:0] imm9   = ir[8:0];    // 9-bit immediate

// Sign-extended immediates
wire [15:0] imm6_se = {{10{imm6[5]}}, imm6};
wire [15:0] imm9_se = {{7{imm9[8]}}, imm9};

//============================================================================
// Instruction Opcodes
//============================================================================

localparam OP_NOP    = 4'h0;
localparam OP_MOV    = 4'h1;
localparam OP_ADD    = 4'h2;
localparam OP_SUB    = 4'h3;
localparam OP_AND    = 4'h4;
localparam OP_OR     = 4'h5;
localparam OP_XOR    = 4'h6;
localparam OP_CMP    = 4'h7;
localparam OP_SHIFT  = 4'h8;
localparam OP_MUL    = 4'h9;
localparam OP_DIV    = 4'hA;
localparam OP_JUMP   = 4'hB;
localparam OP_BRANCH = 4'hC;
localparam OP_LOAD   = 4'hD;
localparam OP_STORE  = 4'hE;
localparam OP_SPECIAL = 4'hF;

//============================================================================
// ALU Operations
//============================================================================

wire [15:0] reg_rs = regs[rs];
wire [15:0] reg_rt = regs[rt];
wire [15:0] reg_rd = regs[rd];

always @(*) begin
    alu_result = 16'd0;
    alu_carry = 1'b0;
    alu_overflow = 1'b0;
    
    case (opcode)
        OP_ADD: begin
            {alu_carry, alu_result} = reg_rs + reg_rt;
            alu_overflow = (reg_rs[15] == reg_rt[15]) && (alu_result[15] != reg_rs[15]);
        end
        
        OP_SUB: begin
            {alu_carry, alu_result} = reg_rs - reg_rt;
            alu_overflow = (reg_rs[15] != reg_rt[15]) && (alu_result[15] != reg_rs[15]);
        end
        
        OP_AND: begin
            alu_result = reg_rs & reg_rt;
        end
        
        OP_OR: begin
            alu_result = reg_rs | reg_rt;
        end
        
        OP_XOR: begin
            alu_result = reg_rs ^ reg_rt;
        end
        
        OP_CMP: begin
            {alu_carry, alu_result} = reg_rs - reg_rt;
        end
        
        OP_SHIFT: begin
            case (ir[5:4])
                2'b00: alu_result = reg_rs << imm6[3:0];  // SHL
                2'b01: alu_result = reg_rs >> imm6[3:0];  // SHR
                2'b10: alu_result = $signed(reg_rs) >>> imm6[3:0]; // SAR
                2'b11: {alu_result, alu_carry} = {flag_c, reg_rs}; // ROL/ROR
            endcase
        end
        
        OP_MUL: begin
            alu_result = reg_rs * reg_rt;  // Lower 16 bits
        end
        
        OP_DIV: begin
            if (reg_rt != 0)
                alu_result = reg_rs / reg_rt;
            else
                alu_result = 16'hFFFF;  // Division by zero
        end
        
        default: alu_result = 16'd0;
    endcase
end

//============================================================================
// Update Flags
//============================================================================

task update_flags;
    input [15:0] result;
    input        carry;
    input        overflow;
    input        update_zn;
    input        update_cv;
    begin
        if (update_zn) begin
            flags[0] <= (result == 16'd0);  // Z
            flags[1] <= result[15];          // N
        end
        if (update_cv) begin
            flags[2] <= carry;               // C
            flags[3] <= overflow;            // V
        end
    end
endtask

//============================================================================
// CPU Main State Machine
//============================================================================

always @(posedge clk or negedge reset_n) begin
    if (!reset_n) begin
        // Reset state
        pc_bank <= 8'h01;       // Start in bank 1 (ROM)
        pc_offset <= 16'h8000;  // Start at offset 0x8000
        db_bank <= 8'h00;
        sp <= 16'h1FFF;         // Stack at top of bank 0
        flags <= 5'b0;
        state <= STATE_FETCH;
        we <= 1'b0;
        re <= 1'b0;
        irq_pending <= 1'b0;
        nmi_pending <= 1'b0;
        
        // Clear registers
        regs[0] <= 16'd0;
        regs[1] <= 16'd0;
        regs[2] <= 16'd0;
        regs[3] <= 16'd0;
        regs[4] <= 16'd0;
        regs[5] <= 16'd0;
        regs[6] <= 16'd0;
        regs[7] <= 16'd0;
    end else begin
        // Latch interrupts
        if (!irq_n && !flag_i)
            irq_pending <= 1'b1;
        if (!nmi_n)
            nmi_pending <= 1'b1;
        
        case (state)
            STATE_FETCH: begin
                // Fetch instruction
                addr <= {pc_bank, pc_offset};
                re <= 1'b1;
                we <= 1'b0;
                
                if (ready) begin
                    ir <= rdata;
                    re <= 1'b0;
                    pc_offset <= pc_offset + 2;  // PC + 2 (16-bit instructions)
                    state <= STATE_DECODE;
                end
            end
            
            STATE_DECODE: begin
                // Decode instruction and calculate effective address if needed
                case (opcode)
                    OP_LOAD, OP_STORE: begin
                        // Memory access instructions
                        // Address = [DBR : (Rs + offset)]
                        effective_addr <= {db_bank, reg_rs + imm9_se};
                        state <= STATE_MEMORY;
                    end
                    
                    OP_JUMP: begin
                        // Jump instructions
                        state <= STATE_EXECUTE;
                    end
                    
                    OP_BRANCH: begin
                        // Branch instructions - check condition
                        state <= STATE_EXECUTE;
                    end
                    
                    default: begin
                        // Register operations
                        state <= STATE_EXECUTE;
                    end
                endcase
            end
            
            STATE_MEMORY: begin
                // Memory read/write
                addr <= effective_addr;
                
                if (opcode == OP_LOAD) begin
                    re <= 1'b1;
                    if (ready) begin
                        re <= 1'b0;
                        state <= STATE_WRITEBACK;
                    end
                end else if (opcode == OP_STORE) begin
                    wdata <= reg_rd;
                    we <= 1'b1;
                    if (ready) begin
                        we <= 1'b0;
                        state <= STATE_FETCH;
                    end
                end
            end
            
            STATE_EXECUTE: begin
                // Execute instruction
                case (opcode)
                    OP_NOP: begin
                        state <= STATE_FETCH;
                    end
                    
                    OP_MOV: begin
                        case (ir[11:9])
                            3'b000: begin  // MOV Rd, Rs
                                regs[rd] <= reg_rs;
                                update_flags(reg_rs, 1'b0, 1'b0, 1'b1, 1'b0);
                            end
                            3'b001: begin  // MOV Rd, #imm
                                regs[rd] <= imm9_se;
                                update_flags(imm9_se, 1'b0, 1'b0, 1'b1, 1'b0);
                            end
                            3'b010: begin  // MOV DBR, Rs
                                db_bank <= reg_rs[7:0];
                            end
                            3'b011: begin  // MOV Rs, DBR
                                regs[rd] <= {8'd0, db_bank};
                            end
                            default: begin
                                regs[rd] <= reg_rs;
                            end
                        endcase
                        state <= STATE_FETCH;
                    end
                    
                    OP_ADD: begin
                        regs[rd] <= alu_result;
                        update_flags(alu_result, alu_carry, alu_overflow, 1'b1, 1'b1);
                        state <= STATE_FETCH;
                    end
                    
                    OP_SUB: begin
                        regs[rd] <= alu_result;
                        update_flags(alu_result, alu_carry, alu_overflow, 1'b1, 1'b1);
                        state <= STATE_FETCH;
                    end
                    
                    OP_AND: begin
                        regs[rd] <= alu_result;
                        update_flags(alu_result, 1'b0, 1'b0, 1'b1, 1'b0);
                        state <= STATE_FETCH;
                    end
                    
                    OP_OR: begin
                        regs[rd] <= alu_result;
                        update_flags(alu_result, 1'b0, 1'b0, 1'b1, 1'b0);
                        state <= STATE_FETCH;
                    end
                    
                    OP_XOR: begin
                        regs[rd] <= alu_result;
                        update_flags(alu_result, 1'b0, 1'b0, 1'b1, 1'b0);
                        state <= STATE_FETCH;
                    end
                    
                    OP_CMP: begin
                        // CMP only updates flags, not destination
                        update_flags(alu_result, alu_carry, alu_overflow, 1'b1, 1'b1);
                        state <= STATE_FETCH;
                    end
                    
                    OP_SHIFT: begin
                        regs[rd] <= alu_result;
                        update_flags(alu_result, alu_carry, 1'b0, 1'b1, 1'b1);
                        state <= STATE_FETCH;
                    end
                    
                    OP_MUL: begin
                        regs[rd] <= alu_result;
                        update_flags(alu_result, 1'b0, 1'b0, 1'b1, 1'b0);
                        state <= STATE_FETCH;
                    end
                    
                    OP_DIV: begin
                        regs[rd] <= alu_result;
                        update_flags(alu_result, 1'b0, 1'b0, 1'b1, 1'b0);
                        state <= STATE_FETCH;
                    end
                    
                    OP_JUMP: begin
                        case (ir[11:9])
                            3'b000: begin  // JMP addr
                                pc_offset <= imm9_se;
                            end
                            3'b001: begin  // JMP Rs
                                pc_offset <= reg_rs;
                            end
                            3'b010: begin  // JSR addr (Jump to Subroutine)
                                // Push return address to stack
                                sp <= sp - 2;
                                addr <= {8'h00, sp - 2};
                                wdata <= pc_offset;
                                we <= 1'b1;
                            end
                            3'b011: begin  // RTS (Return from Subroutine)
                                // Pop return address from stack
                                addr <= {8'h00, sp};
                                re <= 1'b1;
                                sp <= sp + 2;
                            end
                            3'b100: begin  // JMP far (bank switch)
                                pc_bank <= reg_rs[7:0];
                                pc_offset <= reg_rt;
                            end
                            default: begin
                                pc_offset <= imm9_se;
                            end
                        endcase
                        state <= STATE_FETCH;
                    end
                    
                    OP_BRANCH: begin
                        // Branch conditions
                        reg branch_taken;
                        case (ir[11:9])
                            3'b000: branch_taken = flag_z;           // BEQ
                            3'b001: branch_taken = !flag_z;          // BNE
                            3'b010: branch_taken = flag_c;           // BCS/BLO
                            3'b011: branch_taken = !flag_c;          // BCC/BHS
                            3'b100: branch_taken = flag_n;           // BMI
                            3'b101: branch_taken = !flag_n;          // BPL
                            3'b110: branch_taken = flag_v;           // BVS
                            3'b111: branch_taken = !flag_v;          // BVC
                        endcase
                        
                        if (branch_taken) begin
                            pc_offset <= pc_offset + imm9_se;
                        end
                        state <= STATE_FETCH;
                    end
                    
                    OP_SPECIAL: begin
                        case (ir[11:9])
                            3'b000: begin  // PUSH Rs
                                sp <= sp - 2;
                                addr <= {8'h00, sp - 2};
                                wdata <= reg_rs;
                                we <= 1'b1;
                            end
                            3'b001: begin  // POP Rd
                                addr <= {8'h00, sp};
                                re <= 1'b1;
                                sp <= sp + 2;
                            end
                            3'b010: begin  // SEI (Set Interrupt Mask)
                                flags[4] <= 1'b1;
                            end
                            3'b011: begin  // CLI (Clear Interrupt Mask)
                                flags[4] <= 1'b0;
                            end
                            3'b100: begin  // CLC (Clear Carry)
                                flags[2] <= 1'b0;
                            end
                            3'b101: begin  // SEC (Set Carry)
                                flags[2] <= 1'b1;
                            end
                            3'b110: begin  // HALT
                                // Halt CPU
                            end
                            3'b111: begin  // RTI (Return from Interrupt)
                                // Pop flags and return address
                                addr <= {8'h00, sp};
                                re <= 1'b1;
                                sp <= sp + 4;
                            end
                        endcase
                        state <= STATE_FETCH;
                    end
                    
                    default: begin
                        state <= STATE_FETCH;
                    end
                endcase
            end
            
            STATE_WRITEBACK: begin
                // Write back loaded data to register
                if (opcode == OP_LOAD) begin
                    regs[rd] <= rdata;
                    update_flags(rdata, 1'b0, 1'b0, 1'b1, 1'b0);
                end else if (opcode == OP_SPECIAL && ir[11:9] == 3'b001) begin
                    // POP
                    regs[rd] <= rdata;
                end
                state <= STATE_FETCH;
            end
            
            STATE_INTERRUPT: begin
                // Handle interrupt
                // Push PC and flags to stack
                // Jump to interrupt vector
                
                if (nmi_pending) begin
                    nmi_pending <= 1'b0;
                    interrupt_vector_bank <= 8'h00;
                    interrupt_vector_offset <= 16'hFFE2;  // NMI vector
                end else if (irq_pending) begin
                    irq_pending <= 1'b0;
                    interrupt_vector_bank <= 8'h00;
                    interrupt_vector_offset <= 16'hFFE0;  // IRQ vector
                end
                
                // Push current state
                sp <= sp - 6;
                // Save PC bank, PC offset, and flags
                
                // Load interrupt vector
                pc_bank <= interrupt_vector_bank;
                pc_offset <= interrupt_vector_offset;
                
                state <= STATE_FETCH;
            end
            
            default: begin
                state <= STATE_FETCH;
            end
        endcase
        
        // Check for interrupts at fetch stage
        if (state == STATE_FETCH && (irq_pending || nmi_pending)) begin
            state <= STATE_INTERRUPT;
        end
    end
end

//============================================================================
// Status Outputs
//============================================================================

assign halted = (state == STATE_FETCH) && (ir == 16'hF600);  // HALT instruction
assign error = (opcode == OP_DIV) && (reg_rt == 0);  // Division by zero

endmodule
