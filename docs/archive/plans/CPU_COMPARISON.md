# CPU Architecture & Performance Comparison

## Register Architecture Comparison

### Nitro-Core-DX

**General Purpose Registers:**
- **8 registers (R0-R7)**: All 16-bit, fully general-purpose
- All registers are equal - no special-purpose restrictions
- Can be used for arithmetic, data movement, addressing, or any purpose

**Special Registers:**
- **PC (Program Counter)**: 24-bit logical address (bank:offset)
  - `PCBank` (8-bit): Bank number (0-255)
  - `PCOffset` (16-bit): Offset within bank (0x0000-0xFFFF)
- **SP (Stack Pointer)**: 16-bit offset in bank 0 (starts at 0x1FFF)
- **PBR (Program Bank Register)**: 8-bit, current program bank
- **DBR (Data Bank Register)**: 8-bit, current data bank
- **Flags Register**: 8-bit (Z, N, C, V, I, D flags)

**Key Features:**
- Simple, orthogonal design - all registers are equal
- Banked 24-bit addressing (16MB address space)
- Clean separation between program and data banks

---

### SNES (Super Nintendo Entertainment System)

**CPU:** Ricoh 5A22 (based on 65C816, 16-bit variant of 6502)

**Registers:**
- **A (Accumulator)**: 8-bit or 16-bit (switchable via flag)
- **X, Y (Index Registers)**: 8-bit or 16-bit (switchable via flag)
- **S (Stack Pointer)**: 16-bit, always points to bank 0
- **PC (Program Counter)**: 24-bit (8-bit bank + 16-bit offset)
- **P (Processor Status)**: 8-bit flags register
- **DBR (Data Bank Register)**: 8-bit, for data access
- **PBR (Program Bank Register)**: 8-bit, for instruction fetch

**Key Features:**
- **Accumulator-based architecture**: A register is special-purpose
- **Index registers**: X and Y are specialized for indexing
- **8/16-bit mode switching**: Can switch between 8-bit and 16-bit operations
- **Banked addressing**: Similar 24-bit addressing to Nitro-Core-DX
- **6502 heritage**: Instruction set based on 6502, with 16-bit extensions

**Register Limitations:**
- A register is heavily used (accumulator-based)
- X and Y are specialized for indexing/looping
- Fewer general-purpose registers than Nitro-Core-DX

---

### Sega Genesis (Mega Drive)

**CPU:** Motorola 68000 (16/32-bit processor)

**Registers:**
- **D0-D7 (Data Registers)**: 8 × 32-bit registers
  - Can be accessed as 8-bit, 16-bit, or 32-bit
  - Used for arithmetic and data operations
- **A0-A6 (Address Registers)**: 7 × 32-bit registers
  - Used for addressing and pointer operations
  - A7 is the stack pointer (user and supervisor stacks)
- **A7/SP (Stack Pointer)**: 32-bit, two stacks (user/supervisor)
- **PC (Program Counter)**: 32-bit
- **SR (Status Register)**: 16-bit (condition codes + control bits)

**Key Features:**
- **32-bit internal architecture**: All registers are 32-bit internally
- **Register specialization**: Data vs Address registers
- **Rich addressing modes**: Many addressing modes (68000 is CISC)
- **Flat addressing**: 32-bit address space (24-bit externally on Genesis)
- **Powerful instruction set**: Complex instructions, many addressing modes

**Register Advantages:**
- More registers (15 total: 8 data + 7 address)
- 32-bit operations available
- Separate data and address registers allow efficient code

---

## Processing Power Comparison

### Clock Speed & Cycles Per Frame

| System | CPU | Clock Speed | Cycles/Frame (60 FPS) | Effective Throughput |
|--------|-----|-------------|----------------------|---------------------|
| **Nitro-Core-DX** | Custom 16-bit | **10 MHz** | **166,667** | Highest |
| **Sega Genesis** | Motorola 68000 | **7.67 MHz** (NTSC) | ~127,833 | High |
| **SNES** | Ricoh 5A22 | **2.68 MHz** (normal) / 3.58 MHz (high-speed) | ~44,667 (normal) / ~59,667 (high-speed) | Lower |

### Performance Analysis

#### Nitro-Core-DX: **10 MHz**
- **166,667 cycles per frame** at 60 FPS
- **Fastest of the three systems**
- Simple, orthogonal instruction set = efficient execution
- 8 equal general-purpose registers = good register allocation
- **~3.7× faster than SNES (normal mode)**
- **~1.3× faster than Genesis**

#### Sega Genesis: **7.67 MHz**
- **~127,833 cycles per frame** at 60 FPS
- **Second fastest**
- 68000 is a powerful CISC processor
- Rich instruction set with many addressing modes
- 32-bit internal operations (though 16-bit externally)
- **~2.9× faster than SNES (normal mode)**

#### SNES: **2.68 MHz (normal) / 3.58 MHz (high-speed)**
- **~44,667 cycles/frame (normal)** or **~59,667 cycles/frame (high-speed)**
- **Slowest of the three**
- 6502 heritage = simple but limited instruction set
- Accumulator-based = more instructions needed for some operations
- 8/16-bit mode switching adds complexity
- **However**: SNES has powerful co-processors (PPU, DSP) that offload work

---

## Architectural Philosophy Comparison

### Nitro-Core-DX
- **Design Philosophy**: Simple, clean, orthogonal
- **Register Model**: 8 equal general-purpose registers
- **Addressing**: Banked 24-bit (16MB space)
- **Instruction Set**: Custom, designed for clarity and efficiency
- **Goal**: Combine SNES graphics with Genesis-level CPU power

### SNES
- **Design Philosophy**: Extend 6502 to 16-bit with banking
- **Register Model**: Accumulator-based (A is special)
- **Addressing**: Banked 24-bit (similar to Nitro-Core-DX)
- **Instruction Set**: 6502-based, with 16-bit extensions
- **Trade-off**: Simpler CPU, but powerful co-processors handle graphics/audio

### Genesis
- **Design Philosophy**: Use powerful off-the-shelf 32-bit CPU
- **Register Model**: Separate data and address registers
- **Addressing**: Flat 24-bit (32-bit internally)
- **Instruction Set**: Full 68000 CISC instruction set
- **Trade-off**: More complex, but very powerful and flexible

---

## Practical Implications

### Why Nitro-Core-DX's Design is Powerful

1. **More General-Purpose Registers**: 8 equal registers vs SNES's accumulator-based model means less register pressure and more efficient code generation.

2. **Higher Clock Speed**: 10 MHz gives significantly more cycles per frame than either SNES or Genesis, allowing for:
   - More complex game logic
   - More AI processing
   - More physics calculations
   - More complex graphics effects

3. **Simple but Effective**: Unlike Genesis's complex CISC instruction set, Nitro-Core-DX's simpler instruction set is easier to optimize and can achieve better instruction-level parallelism.

4. **Best of Both Worlds**: 
   - SNES-level graphics capabilities (4 layers, Matrix Mode, etc.)
   - Genesis-level (actually higher) CPU performance
   - Clean, modern architecture

### Performance Summary

**Raw CPU Power (cycles per frame):**
1. **Nitro-Core-DX**: 166,667 cycles/frame ⚡
2. **Genesis**: ~127,833 cycles/frame
3. **SNES**: ~44,667-59,667 cycles/frame

**Effective Performance (considering architecture):**
- **Nitro-Core-DX**: Highest - simple, fast, many registers
- **Genesis**: High - powerful CISC, but more complex
- **SNES**: Lower CPU, but excellent co-processors

**Result**: Nitro-Core-DX delivers the **SNES graphics** with **significantly more CPU power** than either original system, enabling more complex games and effects.
