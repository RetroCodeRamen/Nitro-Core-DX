# Development Notes

This document contains narrative notes about the development process, challenges, and philosophy behind Nitro-Core-DX. For technical documentation, see [SYSTEM_MANUAL.md](../SYSTEM_MANUAL.md) and [PROGRAMMING_MANUAL.md](../PROGRAMMING_MANUAL.md).

---

## The Three-Layer Challenge: Hardware, Emulator, and Compiler

Nitro-Core-DX isn't just an emulator—it's a complete system built from scratch. This project involves three major components, each with its own complexity:

### 1. **Hardware Architecture Design**
The foundation: designing a custom 16-bit CPU, memory map, PPU (graphics), APU (audio), and I/O systems. This includes:
- Custom instruction set with 16-bit operations
- Banked memory architecture (256 banks × 64KB = 16MB addressable)
- Graphics pipeline with 4 background layers, sprites, Matrix Mode
- Audio synthesis with 4 channels and waveform generation
- Memory-mapped I/O registers for hardware control

**The Challenge**: Every design decision affects everything else. Change a register layout? Update the emulator. Modify the instruction encoding? Fix the compiler. It's a delicate balance between "what's possible" and "what's practical."

### 2. **Emulator Implementation**
The execution layer: cycle-accurate CPU emulation, pixel-perfect PPU rendering, sample-accurate audio synthesis. This includes:
- CPU instruction execution with precise cycle counting
- PPU rendering pipeline (tiles, sprites, layers, Matrix Mode)
- APU waveform generation and mixing
- Memory bus routing and bank switching
- Synchronization (VBlank, frame counter, completion flags)

**The Challenge**: The emulator must match the hardware specification exactly. A single cycle off can cause timing issues. A register read/write bug can break entire games. And when something doesn't work, is it the emulator's fault or the hardware design?

### 3. **CoreLX Compiler**
The language layer: a custom compiled language (CoreLX) with Lua-like syntax, designed for hardware-first programming. This includes:
- Lexer and parser for CoreLX syntax
- Semantic analysis and type checking
- Code generation (translating CoreLX to Nitro-Core-DX assembly)
- Built-in function mapping (PPU, APU, sprite operations)
- Asset embedding and ROM building

**The Challenge**: The compiler must generate correct assembly code that matches the hardware's expectations. A wrong register allocation? The ROM crashes. An incorrect memory address? Graphics glitch. And when the ROM doesn't work, is it the compiler's fault, the emulator's fault, or the hardware design?

---

## The Debugging Nightmare: Where's the Bug?

This is where things get interesting—and frustrating. When something doesn't work, there are **four potential sources of the problem**:

### 🔍 **Is it the ROM code?**
- Did I write the CoreLX code correctly?
- Is the logic sound?
- Are the function calls correct?

### 🔍 **Is it the compiler?**
- Did the compiler generate the wrong assembly?
- Is register allocation incorrect?
- Are memory addresses calculated wrong?
- Did built-in functions get translated incorrectly?

### 🔍 **Is it the emulator?**
- Is the CPU executing instructions correctly?
- Are memory reads/writes working?
- Is the PPU rendering correctly?
- Are I/O registers responding as expected?
- Is synchronization working?

### 🔍 **Is it the hardware design?**
- Is the instruction set complete?
- Are the register layouts correct?
- Is the memory map sound?
- Are there design flaws that need fixing?

**The Reality**: Most bugs involve multiple layers. A compiler bug might generate code that exposes an emulator bug, which reveals a hardware design flaw. Or vice versa. It's like debugging a house of cards—fix one thing, and three others might fall over.

---

## Flying Blind: The Learning Experience

I'm building this project "flying blind"—learning as I go, with AI assistance helping where I'm technically weak. It's an incredible learning experience, but it comes with unique challenges:

### The Good
- **Deep Understanding**: Building everything from scratch means understanding every layer
- **Creative Freedom**: No legacy constraints—design what makes sense
- **AI Assistance**: AI helps with heavy lifting (code generation, documentation, debugging suggestions)
- **Real Learning**: Every bug teaches something new about hardware, compilers, or emulation

### The Hard Parts
- **Isolation is Difficult**: When a ROM crashes, which layer is at fault?
- **Testing is Complex**: Need to test hardware design, emulator accuracy, and compiler correctness
- **Documentation is Critical**: Without good docs, it's impossible to know what "correct" behavior is
- **No Reference Implementation**: Can't compare to "known good" behavior—we're defining what "good" is

### The Strategy
1. **Test Each Layer Independently**: Write assembly ROMs to test emulator. Write simple CoreLX to test compiler.
2. **Comprehensive Logging**: Log everything—CPU cycles, memory access, register changes, PPU state
3. **Incremental Development**: Build one feature at a time, test thoroughly before moving on
4. **Document Everything**: Write down expected behavior, test cases, known issues
5. **Use AI Strategically**: AI helps with code generation and debugging, but I make the architectural decisions

---

## Development Philosophy: Building Something Real

This project is a **massive undertaking**—even with AI assistance doing heavy lifting in areas where I'm technically weak. Creating custom hardware, software, and a compiler for a new language is not trivial. Every component depends on every other component, and bugs can hide in any layer.

### Why This Matters

Most emulator projects start with existing hardware—you emulate what already exists, and you can compare your results to real hardware. But Nitro-Core-DX is different:

- **No Reference Hardware**: We're defining what the hardware *should* do, not emulating what it *does* do
- **No Existing Software**: We're creating the first software for this platform
- **No Existing Compiler**: We're building the first compiler for CoreLX
- **No Test Suite**: We're creating the test suite as we go

This means **every bug is a learning opportunity**, but it also means **debugging is incredibly difficult**. When a ROM crashes, is it:
- A bug in my ROM code?
- A bug in the compiler?
- A bug in the emulator?
- A design flaw in the hardware specification?

Often, it's a combination of all four.

### The Testing Challenge

The hardest part of this project isn't writing code—it's **isolating where problems come from**. Here's the typical debugging flow:

1. **Write a CoreLX program** → Compile it → Run it in emulator → It crashes
2. **Check the ROM**: Is the CoreLX code correct? (Maybe)
3. **Check the compiler output**: Did it generate correct assembly? (Maybe)
4. **Check the emulator**: Is it executing correctly? (Maybe)
5. **Check the hardware design**: Is the specification correct? (Maybe)

The answer is usually "a little bit of everything." A compiler bug might generate code that exposes an emulator bug, which reveals a hardware design issue. Fix one thing, and three others might break.

### The Learning Journey

I'm building this "flying blind"—learning as I go, with AI helping where I'm weak. It's an incredible learning experience:

- **Hardware Design**: Learning CPU architecture, memory systems, graphics pipelines
- **Emulator Development**: Learning cycle-accurate emulation, synchronization, timing
- **Compiler Design**: Learning lexing, parsing, code generation, optimization
- **System Integration**: Learning how all the pieces fit together

But it's also frustrating. When you're stuck, you can't just "look it up"—you're creating the reference. You can't "compare to real hardware"—you're defining what real hardware should be.

### The Strategy

1. **Test Each Layer Independently**
   - Write assembly ROMs to test emulator (bypass compiler)
   - Write simple CoreLX to test compiler (isolate compiler bugs)
   - Test hardware design with known-good code

2. **Comprehensive Logging**
   - Log CPU cycles, register changes, memory access
   - Log compiler intermediate representations
   - Log emulator state at every step

3. **Incremental Development**
   - Build one feature at a time
   - Test thoroughly before moving on
   - Don't move forward until current layer works

4. **Document Everything**
   - Write down expected behavior
   - Document test cases
   - Keep track of known issues
   - Update manuals when things change

5. **Use AI Strategically**
   - AI helps with code generation and debugging suggestions
   - But I make the architectural decisions
   - AI is a tool, not a replacement for understanding

### The Result

This project is a **real learning experience**. Every bug teaches something new. Every feature reveals new challenges. And every success feels like a genuine achievement—because it is. We're not just emulating something that exists; we're creating something new.
