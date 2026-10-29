# Nitro-Core-DX CPU Design Review: Comparison with SNES and Genesis

**Purpose:** Assess whether the Nitro-Core-DX CPU is fully robust or has a limited instruction set compared to SNES (65C816) and Genesis (68000) era CPUs.

**Sources:** Project `internal/cpu/`, `docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md`, `docs/archive/plans/CPU_COMPARISON.md`, and reference data for 65C816 and 68000.

---

## 1. Executive Summary

| Aspect | Nitro-Core-DX | SNES (65C816) | Genesis (68000) |
|--------|----------------|---------------|-----------------|
| **Philosophy** | Clean, orthogonal, 16-bit | 8/16-bit accumulator, 6502 heritage | 32-bit CISC, many modes |
| **Registers** | 8 GPRs (R0–R7), 16-bit | A, X, Y, S (8/16-bit) | D0–D7, A0–A7 (32-bit) |
| **Instruction count** | ~17 opcode families, ~25 distinct ops | ~92 opcodes, many modes | 56+ instruction types, 14 EA combos |
| **Addressing** | Reg, imm, [R], PUSH/POP | Many (dp, abs, (dp,X), etc.) | 14 addressing mode combinations |
| **Control flow** | Relative JMP/CALL/branches only | Relative + absolute, long branches | Full absolute, PC-relative, etc. |
| **Verdict** | **Focused and sufficient** for target | Richer ops/modes, more legacy quirks | Much richer; full CISC |

**Conclusion:** The Nitro-Core-DX CPU is **intentionally minimal and robust for its role**: fixed 16-bit GPRs, banked 24-bit space, and a small but consistent instruction set. It is **more limited** than both the SNES and Genesis in terms of number of opcodes and addressing modes, but it is **not underpowered for a “SNES-like graphics + Genesis-like CPU power” goal** because of higher clock rate, orthogonal registers, and a compact encoding. Gaps (e.g. no indexed addressing, no absolute 24-bit JMP) are design trade-offs rather than oversights.

---

## 2. Nitro-Core-DX CPU Summary

### 2.1 Registers and Data Widths

- **R0–R7:** 16-bit general-purpose; no dedicated accumulator or index registers.
- **PC:** 24-bit logical (8-bit bank + 16-bit offset); instructions only in ROM range (e.g. offset ≥ 0x8000).
- **PBR, DBR:** 8-bit bank registers for fetch and data access.
- **SP:** 16-bit, bank 0, grows downward (e.g. from 0x1FFF).
- **Flags:** Z, N, C, V, I, D (6 bits).

All arithmetic and logic is **16-bit**; 8-bit is only for load/store (zero-extended or low-byte write).

### 2.2 Instruction Set (from emulator + spec)

| Opcode | Mnemonic | Modes / Notes |
|--------|----------|----------------|
| 0x0 | NOP | — |
| 0x1 | MOV | 8 modes: R↔R, imm→R, [R]→R (16/8), R→[R] (16/8), PUSH R, POP R |
| 0x2 | ADD | R, #imm |
| 0x3 | SUB | R, #imm |
| 0x4 | MUL | R, #imm (low 16 bits) |
| 0x5 | DIV | R, #imm (D flag on divide-by-zero) |
| 0x6 | AND | R, #imm |
| 0x7 | OR | R, #imm |
| 0x8 | XOR | R, #imm |
| 0x9 | NOT | R only |
| 0xA | SHL | R, R or #imm (0–15) |
| 0xB | SHR | R, R or #imm (logical; 0–15) |
| 0xC | CMP / branches | CMP R,R / CMP R,#imm; BEQ, BNE, BGT, BLT, BGE, BLE (relative) |
| 0xD | JMP | Relative 16-bit offset (same bank) |
| 0xE | CALL | Relative 16-bit offset; pushes PBR, PCOffset, Flags |
| 0xF | RET | Pops Flags, PCOffset, PBR |

- **Instruction format:** 16-bit; opcode [15:12], mode [11:8], reg1 [7:4], reg2 [3:0]; immediates and branch offsets are extra 16-bit words.
- **Memory addressing:** Only **register indirect**: address = (DBR, R). No displacement, no index register, no (R)+ / -(R), no PC-relative data.

---

## 3. SNES (65C816) Comparison

### 3.1 What the 65C816 Has That Nitro-Core-DX Doesn’t

- **~92 opcodes** with many addressing modes per opcode (immediate, absolute, absolute long, direct page, direct page indexed, (dp), (dp,X), [dp], (dp),Y, (sr,S),Y, etc.).
- **8- and 16-bit operations** (A, X, Y switchable); byte operations and sign-extension without extra instructions.
- **Accumulator + index model:** A for math, X/Y for indexing/loops; dedicated STA/STX/STY, LDX/LDY, INX/DEX, etc.
- **Direct page (zero page):** Short addresses for fast, compact code.
- **Long branches (BRL)** and **absolute long JMP/JSR** for full 24-bit targets.
- **Block move (MVN/MVP)** for memory-to-memory copy.
- **Decimal mode** and BCD arithmetic.
- **Bit test and branch** (e.g. BBR/BBS).
- **Stack-relative addressing** (e.g. (sr,S),Y).

### 3.2 Where Nitro-Core-DX Is Simpler or Stronger

- **8 symmetric GPRs** vs one accumulator + two index registers: easier register allocation and fewer data moves.
- **Single 16-bit data width** in the ISA (no 8/16 switching): simpler mental model and codegen.
- **Explicit MUL/DIV** in one op; 65C816 has no divide and multiply is limited.
- **Orthogonal MOV:** one opcode covers load/store/push/pop with modes; 65C816 uses many separate opcodes (LDA, STA, LDX, PHA, PLA, etc.).
- **Higher clock and more cycles per frame** (see existing CPU_COMPARISON.md): more work per frame despite fewer opcode variants.

So: Nitro-Core-DX has a **smaller, cleaner instruction set** and **better raw cycle budget**, but **fewer addressing modes and no 8-bit/BCD/block-move/decimal** features.

---

## 4. Genesis (68000) Comparison

### 4.1 What the 68000 Has That Nitro-Core-DX Doesn’t

- **14 addressing mode combinations:** Dn, An, (An), (An)+, -(An), (d,An), (d,An,Xi), (xxx).W, (xxx).L, (d,PC), (d,PC,Xi), #imm; plus size variants (byte/word/long).
- **32-bit operations** (byte/word/long) and 32-bit internal registers; Nitro-Core-DX is 16-bit only.
- **Auto-increment/decrement:** (An)+, -(An) for stacks and queues.
- **Indexed addressing:** (d,An,Xi) and (d,PC,Xi) with scale and size.
- **Rich CISC set:** MULU/MULS, DIVU/DIVS, CHK, TRAP, LINK/UNLK, MOVEM, EXG, SWAP, sign-extend, byte swap, multiple condition codes and branch types.
- **Privilege model:** user/supervisor, separate stack pointers.
- **Exception vector table** and many exception types.

### 4.2 Where Nitro-Core-DX Is Simpler or Aligned

- **Simpler encoding:** Fixed 16-bit instruction word + immediates; no complex extension words.
- **Banked 24-bit space** is a good fit for cartridge ROM; 68000’s 24-bit linear space is different but not strictly “more” for this use case.
- **Same core operations:** add, sub, mul, div, and, or, xor, compare, branches; no decimal/BCD.
- **Interrupt model:** NMI + maskable IRQ and vector table is enough for a console; no need for full 68000 exception set.

So: Nitro-Core-DX is **much more limited** than the 68000 in opcode count, addressing, and operand sizes, but **deliberately minimal** and still capable for game logic when combined with higher clock and good compiler use.

---

## 5. Gaps and Limitations (Nitro-Core-DX)

### 5.1 Addressing

- **No indexed addressing:** Only [R] (address in R, bank in DBR). No [R+disp], [R+index], or [base+index×scale].  
  **Impact:** Struct/array access and string/table walks need extra instructions (compute address in a register first).
- **No PC-relative data:** No “load from (PC + displacement)”.  
  **Impact:** Position-independent data (e.g. constants next to code) requires loading address via register.
- **No auto-increment/ decrement:** No (R)+ or -(R).  
  **Impact:** Loops over arrays are done with explicit ADD to the pointer.

### 5.2 Control Flow

- **JMP/CALL are relative only:** 16-bit signed offset from current PC; they do **not** change bank.  
  **Impact:** Cross-bank calls are done by building bank+offset (e.g. in registers) and using a small trampoline or indirect mechanism; no single “JMP/CALL to 24-bit absolute” instruction.
- **No indirect jump/call:** No “JMP (R)” or “CALL (R)” in the documented instruction set.  
  **Impact:** Jump tables and dynamic dispatch need a short sequence (e.g. load target into a register and then a known trampoline).

### 5.3 Arithmetic and Bits

- **16-bit only:** No native 8-bit arithmetic; 8-bit is only in load/store.  
  **Impact:** Byte-heavy code may use more masking and shifts.
- **SHR is logical only** in the emulator; no arithmetic right shift (SAR) or rotate (ROL/ROR) in the Go implementation (FPGA may differ).
- **No BCD, no decimal mode.**

### 5.4 Other

- **No block move:** No DMA-style “move N words” in the CPU ISA (hardware DMA is separate).
- **No string/repeat** instructions (e.g. 68000 MOVEM or 65C816 block move).
- **No explicit bit test-and-branch** instruction (can be done with AND + branch).

These are consistent with a **minimal, RISC-like** design rather than with SNES/Genesis-level CISC richness.

---

## 6. Robustness Assessment

### 6.1 What Is Robust

- **Register set:** 8 GPRs and clear roles for PC, SP, PBR, DBR, and flags; no accumulator bottleneck.
- **Instruction encoding:** Regular format; opcode/mode/reg fields are consistent and easy to decode (emulator and FPGA).
- **Memory model:** Banked 24-bit space is well defined; I/O vs RAM vs ROM behavior is specified (e.g. 8-bit I/O, 16-bit elsewhere).
- **Interrupts:** NMI + IRQ, vector table, and state save/restore (PBR, PC, flags) are well specified and implemented.
- **Arithmetic:** Add, sub, mul, div with correct flags (Z, N, C, V) and division-by-zero handling (D flag).
- **Control flow:** CMP + conditional branches cover signed comparisons; CALL/RET and stack discipline are consistent.

### 6.2 Where It Is “Limited” vs “Broken”

- **Limited:** Few addressing modes, no 24-bit absolute JMP/CALL, no indexed/PC-relative data, 16-bit-only ALU, no BCD/block-move. These are **design choices** that simplify the CPU and still allow capable game code.
- **Not broken:** The implemented subset is coherent; no obvious missing essentials for a 16-bit banked console CPU (e.g. you can implement all control flow and data access with the current instructions, at the cost of more instructions for some patterns).

---

## 7. Comparison Table (Instruction Set and Features)

| Feature | Nitro-Core-DX | SNES (65C816) | Genesis (68000) |
|---------|----------------|---------------|-----------------|
| GPRs | 8 × 16-bit | 1 A + X, Y (8/16) | 8 D + 7 A (32-bit) |
| Data width | 16-bit (8-bit load/store only) | 8/16-bit | 8/16/32-bit |
| Addressing | Reg, imm, [R], PUSH/POP | Many (dp, abs, (dp,X), etc.) | 14 combinations |
| Indexed | No | Yes (dp,X), (abs,X), etc. | Yes (d,An,Xi), (d,PC,Xi) |
| JMP/CALL | Relative only | Relative + absolute long | Absolute, PC-relative |
| MUL/DIV | Yes (16-bit) | No DIV; limited MUL | MULU/MULS, DIVU/DIVS |
| Shifts | SHL, SHR (logical) | No dedicated shift opcodes | ASR, LSR, ROL, ROR, etc. |
| Block move | No | MVP/MVN | MOVEM, etc. |
| BCD / decimal | No | Yes | No |
| Approx. opcodes | ~25 distinct | ~92 | 56+ types, many modes |

---

## 8. Recommendations

1. **Keep the current design as the baseline:** It is consistent, implementable, and sufficient for the stated “SNES-like graphics, Genesis-like CPU power” goal when combined with clock rate and compiler.
2. **Document the gaps** (indexed addressing, absolute JMP/CALL, SAR/ROL/ROR) in the hardware spec and programming manual so compiler and hand-coders know the idioms (e.g. how to do cross-bank calls and jump tables).
3. **Optional future extensions** (if hardware/FPGA is revised):
   - One or two indexed load/store modes, e.g. [R1 + R2] or [R1 + imm].
   - Optional “JMP (R)” or “CALL (R)” for tables and dispatch.
   - Optional SAR (arithmetic right shift) and maybe ROL/ROR for crypto and graphics.
4. **Align FPGA and emulator:** Confirm whether the FPGA has SAR/ROL/ROR (as suggested in `cpu_core.v`) and document the canonical shift/rotate set so emulator and compiler match.

---

## 9. References

- **In-tree:** `internal/cpu/cpu.go`, `internal/cpu/instructions.go`, `docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md`, `docs/archive/plans/CPU_COMPARISON.md`, `FPGA/nitro_core_dx_fpga/src/cpu/cpu_core.v`.
- **65C816:** ~92 opcodes, many addressing modes; SNES Ricoh 5A22.
- **68000:** 56+ instruction types, 14 addressing mode combinations; Genesis main CPU.
