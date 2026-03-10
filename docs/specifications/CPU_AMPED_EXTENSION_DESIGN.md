# Nitro-Core-DX CPU: Amped Extension Design (Best of Both Worlds)

**Goal:** Make the CPU “similar in power but slightly amped up” by adopting the best ideas from SNES (65C816) and Genesis (68000) while keeping Nitro-Core-DX’s clean, orthogonal design.

**Design philosophy: FPGA-first.**  
Nitro-Core-DX is intended to be **ported to an FPGA when complete**. The emulator is the reference implementation; every behavior specified here must be **deterministic**, **hardware-realizable**, and **documented with explicit cycle/state semantics** so that RTL (e.g. `FPGA/nitro_core_dx_fpga/src/cpu/cpu_core.v`) can match the spec. No feature in this document may rely on host-only behavior, unbounded variable-length decode, or non-deterministic timing. This spec aligns with the **FPGA-implementable** stance of `COMPLETE_HARDWARE_SPECIFICATION_V2.1.md` and with the project’s hardware-first, ROM-first, no-VM philosophy (see `docs/CORELX.md`, `docs/specifications/APU_FM_OPM_EXTENSION_SPEC.md`).

**Principles:**
- Keep 8 GPRs, 16-bit base, banked 24-bit, and the existing instruction format — all easily mapped to a single clock-domain, fixed-width datapath on FPGA.
- Add **addressing and control-flow** features that give the most benefit for game code and compilers while remaining **simple to implement in RTL** (fixed encoding, no microcode, explicit cycle counts).
- Avoid CISC bloat: no BCD, no 32-bit ops, no privilege mode. Add only what closes the biggest gaps and keeps the CPU suitable for synthesis and timing closure.

---

## 1. Summary of Additions

| Tier | Feature | From | Encoding | Impact |
|------|---------|------|----------|--------|
| 1 | Indexed load/store [R+imm] | Both | MOV modes 9–10 | Struct/array access without extra ADDs |
| 1 | Auto-increment/decrement [R]+ / [R]- | Genesis | MOV modes 11–12 | Tighter loops, queue/stack-style access |
| 1 | Indirect JMP/CALL (24-bit) | SNES/Genesis | JMP/CALL mode 1 | Cross-bank calls, jump tables |
| 1 | SAR, ROL, ROR | Both | SHR opcode modes 2–5 | Signed shift, bit rotate for graphics/crypto |
| 2 | 8-bit ADD/SUB (optional) | SNES | ADD/SUB mode 2 | Byte loops, smaller code |

All new behavior fits the current **16-bit instruction word** `[opcode|mode|reg1|reg2]` plus existing immediate words where needed. Every addition is **FPGA-suitable**: fixed-width operands, deterministic cycle counts, and no host or OS dependencies. The emulator and FPGA core must stay in sync so that ROMs run identically in both.

---

## 1.1 Implementation Status (2026-03-09)

- ✅ Emulator Step 1 started: `MOV` modes **9-12** implemented in `internal/cpu/instructions.go`:
  - mode 9: `MOV R1, [R2+imm]`
  - mode 10: `MOV [R1+imm], R2`
  - mode 11: `MOV R1, [R2]+`
  - mode 12: `MOV [R1]-, R2`
- ✅ Unit tests added for new MOV modes (indexed load/store, signed displacement, I/O semantics, auto inc/dec).
- ✅ Emulator Step 3 complete: `JMP/CALL` mode **1** absolute `(bank:offset)` implemented with ROM window validation and return-frame tests.
- ✅ Emulator Step 4 complete: opcode `0xB` modes **2-5** implemented:
  - mode 2: `SAR R1, R2`
  - mode 3: `SAR R1, #imm`
  - mode 4: `ROL R1, R2` (through carry)
  - mode 5: `ROR R1, R2` (through carry)
- ✅ Unit tests added for SAR/ROL/ROR result and carry behavior.
- ✅ Emulator Step 5 complete (optional tier): `ADD/SUB` modes **2-3** implemented:
  - mode 2: `ADD.B/SUB.B R1, R2` (low-byte arithmetic, result zero-extended)
  - mode 3: `ADD.B/SUB.B R1, #imm` (low-byte immediate, result zero-extended)
- ✅ Unit tests added for byte-mode result, carry/borrow, overflow, sign, and zero behavior.
- ✅ Step 6 (DX CoreLX backend) complete: codegen now adopts amped CPU modes in struct-member paths where semantics are safe:
  - `Vec2` (`i16` fields) member load/store uses indexed word access (`MOV` mode 9/10).
  - `Sprite` byte fields remain on byte access path (`MOV` mode 6/7) to avoid width regressions.
  - Struct zero-initialization now zeroes the full object in deterministic 16-bit chunks (`MOV` mode 10 with displacement), replacing the old single-byte partial clear.
  - CoreLX codegen tests added to verify emitted mode usage and to prevent `Vec2.y` width regressions.
- 🚧 Remaining post-Step-6 work: FPGA parity and backend coverage expansion (if/when additional high-level patterns are promoted to new modes).
- 🚧 FPGA parity for these modes is still pending and should be implemented before relying on them as hardware-contract complete.

---

## 2. Current Instruction Format (Unchanged)

```
Word 0: [15:12] opcode, [11:8] mode, [7:4] reg1, [3:0] reg2
Word 1: optional 16-bit immediate (for imm, displacement, or branch offset)
```

- **MOV** already uses modes 0–7; modes 8–15 are reserved → use 9–12 for new addressing.
- **JMP** and **CALL** currently ignore mode/reg; use **mode** to select relative (0) vs absolute (1).
- **SHR** uses mode 0 (reg) and 1 (imm); use modes 2–5 for SAR/ROL/ROR.

---

## 3. Tier 1 Extensions (Recommended)

Cycle counts and flag behavior below are **binding for both emulator and FPGA**; RTL must match so ROMs behave identically.

### 3.1 Indexed Load/Store: [R + imm16]

**SNES/Genesis:** Both have (base + displacement) and (base + index). We add **base + 16-bit displacement** only; base+index can be done with one ADD.

- **MOV mode 9 — Load from [R2 + imm]:**  
  `MOV R1, [R2+imm]`  
  - reg1 = destination register, reg2 = base address register.  
  - Fetch next word as **signed 16-bit displacement**.  
  - Address = (DBR, R2 + displacement).  
  - Load 16-bit into R1; I/O and 8-bit rules same as MOV mode 2.  
  - Cycles: 1 (fetch imm) + 2 (mem read) + 1 = 4 base.

- **MOV mode 10 — Store to [R1 + imm]:**  
  `MOV [R1+imm], R2`  
  - reg1 = base, reg2 = value.  
  - Fetch next word as **signed 16-bit displacement**.  
  - Address = (DBR, R1 + displacement).  
  - Store 16-bit from R2; I/O and 8-bit rules same as MOV mode 3.  
  - Cycles: same idea as mode 9.

**Byte variants (optional, keep encoding simple first):**  
You can add mode 13 = load 8-bit from [R2+imm], mode 14 = store 8-bit to [R1+imm] later.

**Encoding:**  
- Mode 9: `0x19xy` (opcode 1, mode 9, reg1=x, reg2=y); then one immediate word.  
- Mode 10: `0x1Axy`; then one immediate word.

---

### 3.2 Auto-Increment / Auto-Decrement (Genesis Style)

**Genesis:** (An)+ and -(An) for post-increment load and pre-decrement store. We add the same idea for **word** access (increment/decrement by 2).

- **MOV mode 11 — Load and post-increment:**  
  `MOV R1, [R2]+`  
  - Load from (DBR, R2) into R1 (same as mode 2).  
  - Then R2 += 2.  
  - reg2 is the pointer; it is updated.  
  - Cycles: 2 (mem) + 1 = 3.

- **MOV mode 12 — Pre-decrement and store:**  
  `MOV [R1]-, R2`  
  - First R1 -= 2.  
  - Then store R2 to (DBR, R1).  
  - reg1 is the pointer; it is updated.  
  - Cycles: 2 (mem) + 1 = 3.

Use for: scanning arrays, stack-style buffers, DMA-style queues. No extra word; just new mode.

**Encoding:**  
- Mode 11: `0x1Bxy` (load to R1 from [R2], then R2+=2).  
- Mode 12: `0x1Cxy` (R1-=2, then store R2 to [R1]).

---

### 3.3 Indirect / Absolute JMP and CALL (24-bit)

**SNES:** JML/JSL absolute long. **Genesis:** JMP (An), JSR (An). We add **one** mode: jump/call to (bank, offset) from two registers so cross-bank and jump tables are easy.

- **JMP mode 0 (current):** Relative 16-bit offset (next word). Same as today.
- **JMP mode 1 — Absolute 24-bit:**  
  `JMP (R1:R2)`  
  - reg1 = bank (use low 8 bits only).  
  - reg2 = 16-bit offset.  
  - Set PBR = R1 & 0xFF, PCBank = same, PCOffset = R2 & ~1.  
  - Validate: bank 1–125, offset >= 0x8000 (ROM).  
  - No extra word. Cycles: 2.

- **CALL mode 0 (current):** Relative offset; push PBR, PCOffset, Flags; then relative jump.
- **CALL mode 1 — Absolute 24-bit:**  
  `CALL (R1:R2)`  
  - Push PBR, PCOffset, Flags (same as now).  
  - Set PBR/PCBank/PCOffset from R1 (bank) and R2 (offset).  
  - Same validation as JMP.  
  - Cycles: 4–5.

**Encoding:**  
- JMP: `0xD0xx` = relative (current); `0xD1xy` = absolute with R1=bank, R2=offset.  
- CALL: `0xE0xx` = relative; `0xE1xy` = absolute with R1=bank, R2=offset.

**Jump table:** Load bank and offset from memory into R1/R2, then `JMP (R1:R2)` or use a small trampoline that does the same.

---

### 3.4 SAR (Arithmetic Right Shift) and ROL/ROR

**Current:** SHL (opcode 0xA), SHR (opcode 0xB) logical only. **SNES/Genesis:** arithmetic shift and rotate common.

Use **opcode 0xB** (current SHR) and add modes so one opcode family covers all shifts/rotates:

| Mode | Name | Description |
|------|------|-------------|
| 0 | SHR R1, R2 | Logical right (R2 & 0xF = count) — existing |
| 1 | SHR R1, #imm | Logical right, imm in next word (low 4 bits) — existing |
| 2 | SAR R1, R2 | Arithmetic right (sign-extend); count in R2 & 0xF |
| 3 | SAR R1, #imm | Arithmetic right, imm in next word |
| 4 | ROL R1, R2 | Rotate left through carry; count in R2 & 0xF |
| 5 | ROR R1, R2 | Rotate right through carry; count in R2 & 0xF |

For ROL/ROR, “through carry” means: 17-bit value (C || R1), rotate by count, then result low 16 bits → R1, new C = bit that shifted out. Same as 68000 ROL/ROR.

**Encoding:**  
- 0xB0xy, 0xB1xy = SHR (unchanged).  
- 0xB2xy, 0xB3xy = SAR (reg and imm; for 3, fetch imm word and use low 4 bits).  
- 0xB4xy = ROL R1, R2 (reg2 gives count).  
- 0xB5xy = ROR R1, R2.

**Flags:** Z, N from result; C from last bit shifted/rotated out; V undefined for SAR/ROL/ROR (or clear).

---

## 4. Tier 2 (Optional) — 8-bit ADD/SUB

**SNES:** 8-bit A operations. For “slightly amped” we can add **byte ADD/SUB** so byte loops and counters don’t need masking.

- **ADD mode 2 — Byte add:**  
  `ADD.B R1, R2` or `ADD.B R1, #imm`  
  - Operand size = 8 bits (low byte of R1 and R2 or imm).  
  - Result 8-bit, zero-extended into R1; set Z, N, C, V on 8-bit result.  
  - One extra word for imm if immediate form; or use reg2 for second operand only (mode 2 = byte, reg only).

Simplest encoding: **mode 2** = byte. So ADD mode 0 = 16-bit reg, mode 1 = 16-bit imm, mode 2 = 8-bit reg (R1 low byte += R2 low byte), mode 3 = 8-bit imm. Same for SUB. This keeps orthogonality and gives real 8-bit arithmetic without a separate “A” register.

---

## 5. Encoding Summary (Quick Reference)

### MOV (opcode 0x1) — new modes

| Mode | Mnemonic | Extra words | Notes |
|------|----------|-------------|--------|
| 0–7 | (existing) | 0 or 1 | R↔R, imm, [R], [R] byte, PUSH, POP |
| 9 | MOV R1, [R2+imm] | 1 (disp) | Indexed load |
| 10 | MOV [R1+imm], R2 | 1 (disp) | Indexed store |
| 11 | MOV R1, [R2]+ | 0 | Load, then R2+=2 |
| 12 | MOV [R1]-, R2 | 0 | R1-=2, then store |

### JMP (opcode 0xD)

| Mode | Mnemonic | Extra words |
|------|----------|-------------|
| 0 | JMP offset | 1 (relative) |
| 1 | JMP (R1:R2) | 0 |

### CALL (opcode 0xE)

| Mode | Mnemonic | Extra words |
|------|----------|-------------|
| 0 | CALL offset | 1 (relative) |
| 1 | CALL (R1:R2) | 0 |

### SHR / SAR / ROL / ROR (opcode 0xB)

| Mode | Mnemonic | Extra words |
|------|----------|-------------|
| 0 | SHR R1, R2 | 0 |
| 1 | SHR R1, #imm | 1 |
| 2 | SAR R1, R2 | 0 |
| 3 | SAR R1, #imm | 1 |
| 4 | ROL R1, R2 | 0 |
| 5 | ROR R1, R2 | 0 |

---

## 6. Implementation Order

Implement in **emulator and FPGA in lockstep** so behavior stays identical and the project remains FPGA-portable.

1. **Spec and tests** — Update `COMPLETE_HARDWARE_SPECIFICATION_V2.1.md` and add unit tests for each new mode (emulator). Tests are the contract for FPGA: RTL must pass the same cases.
2. **MOV 9–10, 11–12** — Indexed and auto-inc/dec in `executeMOV()` (emulator) and in `cpu_core.v` (FPGA); no new opcodes.
3. **JMP/CALL mode 1** — In `executeJMP()` / `executeCALL()` and in FPGA decode; implement (R1:R2) path with same cycle/state semantics.
4. **SAR/ROL/ROR** — In `executeSHR()` (or rename to `executeShiftRotate()`) and in FPGA ALU; add modes 2–5 with defined flag behavior.
5. **Tier 2** — 8-bit ADD/SUB if desired; add last, with matching emulator + FPGA behavior.
6. **CoreLX review** — After the CPU (and optionally FPGA) is updated, perform a **CoreLX language and compiler review** so the new features are handled properly in the programming language. See §6.1 below.

### 6.1 CoreLX language review

Once the amped CPU extensions are implemented in the emulator (and FPGA), the CoreLX compiler and language must be reviewed to ensure:

- **Code generation** uses the new instructions where appropriate: indexed load/store `[R+imm]` for struct/array access, auto-increment/decrement for loops, indirect JMP/CALL for cross-bank calls or jump tables, and SAR/ROL/ROR for bitwise/signed operations.
- **Built-ins and standard library** (if any) that touch memory or control flow are updated or documented to use the new modes where beneficial.
- **Diagnostics** remain accurate (e.g. no misleading errors when targeting the extended instruction set).
- **Documentation** (e.g. `PROGRAMMING_MANUAL.md`, `docs/guides/PROGRAMMING_GUIDE.md`, CoreLX language docs) reflects the new capabilities and any recommended idioms.

The review should reference the CoreLX compiler and codegen (e.g. `internal/corelx/`, target `dx` backend) and the hardware-first design in `docs/CORELX.md` and `docs/specifications/CORELX_NITRO_CORE_8_COMPILER_DESIGN.md` so that the language stays aligned with the FPGA-portable CPU spec.

---

## 7. What We’re Not Adding (By Design)

All exclusions keep the CPU **simple to implement and verify on FPGA** and avoid extra datapath or control logic:

- **32-bit operations** — Stays 16-bit; keeps datapath and register file width fixed for synthesis and timing.
- **BCD / decimal mode** — Not needed for typical game logic; avoids extra ALU and flag logic on FPGA.
- **Block move (MVP/MVN)** — Better handled by dedicated hardware DMA; keeps CPU FSM and bus interface simple.
- **Privilege / supervisor** — Single execution mode; no extra state or decode paths in RTL.
- **PC-relative data load** — Nice for PIC; can be added later as another MOV mode if needed; omitted here to limit scope.
- **Base+index [R1+R2]** — One ADD + [R] is two instructions; acceptable; avoids 3-register encoding and extra address adder in one cycle.

This keeps the CPU “slightly amped” and “best of both worlds” without turning it into a full CISC or a second 68000, and ensures the **entire design remains portable to FPGA** when the project is complete.

---

## 8. FPGA Implementation Notes

- **Single clock domain:** All new instructions use the same fetch/decode/execute/memory pipeline as the existing CPU; no new clock domains or async logic.
- **Deterministic cycles:** Cycle counts in this doc are part of the contract; FPGA must match so that frame timing and ROM behavior are identical to the emulator.
- **State machine:** New MOV modes and JMP/CALL mode 1 fit the existing `STATE_FETCH` / `STATE_DECODE` / `STATE_EXECUTE` / `STATE_MEMORY` / `STATE_WRITEBACK` model in `cpu_core.v`; add states or sub-cycles only where necessary (e.g. one extra cycle for indexed address computation).
- **Alignment with main spec:** This extension spec is a companion to `COMPLETE_HARDWARE_SPECIFICATION_V2.1.md` and its **FPGA Implementation Guidelines** (clock domains, state machines, register access, memory interfaces). Any RTL changes must stay consistent with that document.
