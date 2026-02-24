# Prompt To Build The Nitro Core 8 CoreLX Compiler

Use this design document to implement a **Nitro Core 8 CoreLX compiler** that is intentionally compatible with the existing CoreLX language used by Nitro Core DX.

Constraints and goals:
- Reuse as much of the existing compiler frontend as possible from `internal/corelx/` (`lexer.go`, `parser.go`, `ast.go`, `semantic.go`).
- Preserve **CoreLX language behavior** by default so the language feels the same across Nitro Core DX and Nitro Core 8.
- Implement Nitro Core 8 support as a **target backend + hardware profile + built-in namespace extensions**, not as a forked language.
- Keep the compiler output and runtime semantics deterministic and hardware-oriented (ROM-first, no VM, no runtime scripting).
- Do not break the existing Nitro Core DX compiler flow while adding Nitro Core 8.

Deliverables:
- A new target-aware compiler architecture (shared frontend, target-specific backend)
- Nitro Core 8 hardware profile (register map/capabilities abstraction)
- Nitro Core 8 built-in bindings and codegen support
- CLI support for selecting target (`dx` vs `nc8`)
- Tests proving shared-language compatibility and target-specific extension behavior
- Documentation for users writing portable CoreLX code across both systems

Acceptance criteria:
- A CoreLX program using the common subset compiles for both Nitro Core DX and Nitro Core 8 without syntax changes.
- Nitro Core 8-only features are exposed through clear namespaces/extensions and fail gracefully on DX target.
- Compiler diagnostics explicitly state whether an error is language-level, type-level, or target-hardware capability mismatch.
- The design remains compatible with future FPGA/hardware implementations (deterministic codegen, no host-only assumptions).

---

# CoreLX Compiler Design For Nitro Core 8 (Post-Prequel Platform)

## 1. Purpose

This document defines how to build a **Nitro Core 8 CoreLX compiler** that:
- Feels like the same language as CoreLX on Nitro Core DX
- Maximizes source portability between both consoles
- Adds hardware power through target-specific extensions
- Keeps the language easy to program for game developers
- Preserves the project’s hardware-first philosophy (ROMs should map cleanly to real hardware / future FPGA implementations)

The key idea is:

**One CoreLX language family, multiple hardware targets.**

Not:
- “DX CoreLX” and “NC8 CoreLX” as separate languages

But:
- **CoreLX (common language)**
- plus **Target Profiles** (`dx`, `nc8`)
- plus **Target Extensions** (capability-gated built-ins)

## 2. Design Principles

### 2.1 Compatibility First
- Existing CoreLX syntax and common semantics should remain unchanged.
- Portable code should compile to both targets with minimal/no changes.
- Target-specific features should be additive, not replacements.

### 2.2 Hardware-First and Deterministic
- Codegen must produce deterministic ROM code.
- Built-ins map to real registers/memory behaviors, not host abstractions.
- No runtime reflection / JIT / VM.

### 2.3 Easy To Program
- Keep high-level, ergonomic built-ins for common workflows.
- Expose advanced NC8 power through intuitive namespaces and typed APIs.
- Prefer compile-time validation over runtime surprises.

### 2.4 Scalable To Future Consoles
- Compiler architecture should support more targets later (`nc16`, `fpga-dev`, etc.).
- Hardware differences should live in target profiles and built-in binding tables.

## 3. Scope

### In Scope
- Language compatibility model
- Compiler architecture (frontend/shared + backend/target-specific)
- Target profile system
- Built-in namespace strategy for common vs NC8 extensions
- CLI interface and diagnostics
- Test and compatibility strategy

### Out of Scope
- Final Nitro Core 8 hardware register map specifics (this doc defines the integration pattern)
- Full optimizer implementation details
- IDE/LSP implementation

## 4. User Experience Goals

### 4.1 What A Developer Should Feel
- “I already know CoreLX, so I can write Nitro Core 8 games immediately.”
- “If I use advanced NC8 features, the compiler tells me exactly what target I need.”
- “I can intentionally write portable code across DX and NC8.”

### 4.2 Portability Modes

The compiler should support:
- `portable` mode: disallow target-specific built-ins, enforce common subset
- `dx` mode: allow DX common + DX-specific extensions
- `nc8` mode: allow common + NC8 extensions

Recommended CLI behavior:
- `corelx build game.corelx out.rom --target dx`
- `corelx build game.corelx out.rom --target nc8`
- `corelx check game.corelx --target portable`

## 5. Language Compatibility Model

## 5.1 CoreLX Common Subset (Shared Across DX and NC8)

These should remain identical:
- Lexical syntax (comments, identifiers, literals)
- Indentation-based blocks
- Variables and assignment
- Functions
- Structs
- Control flow (`if`, `while`, `for`)
- Expressions and operators
- Type system fundamentals (`u8`, `u16`, `i16`, `bool`, pointers, fixed-point types)
- Asset declarations (if present in current compiler/frontend)
- Core utility built-ins that are target-agnostic (e.g., `wait_vblank()` only if semantics can be mapped consistently)

Important:
- Keep parser grammar shared.
- Avoid adding NC8 syntax that forces parser divergence unless absolutely necessary.

## 5.2 Target-Specific Extensions (Additive)

NC8 differences should be exposed as:
- New built-in namespaces
- Additional functions in existing namespaces
- Capability-gated constants/types
- Optional compiler directives/attributes (only if needed)

Example pattern:
- Common:
  - `ppu.enable_display()`
  - `sprite.set_pos(id, x, y)`
- NC8-only:
  - `ppu.layer_set_blend(layer, mode, alpha)`
  - `dma.copy_async(src, dst, len)`
  - `audio.stream_play(channel, asset)`
  - `tilemap.set_mode7_params(...)`

## 5.3 Target Capability Diagnostics

If code uses an unsupported feature on a target, emit errors like:
- `NC8 feature 'audio.stream_play' is not available on target 'dx'`
- `Function 'ppu.layer_set_blend' requires target capability: ppu.blending`

This is better than generic “unknown identifier” errors when a symbol exists in another target.

## 6. Compiler Architecture

## 6.1 Proposed Architecture

Refactor compiler into layers:

1. Frontend (shared)
- Lexer
- Parser
- AST
- Semantic analysis (target-aware symbol table/built-ins)

2. Middle Layer (shared)
- Typed AST or lightweight IR
- Constant folding (optional)
- Target capability validation

3. Backend (target-specific)
- Built-in lowering (namespace/function to hardware sequence)
- Memory/register mapping
- ROM layout generation
- Code generation to target machine code/ROM image format

4. Packaging Layer (target-specific)
- ROM header/entrypoint format
- Asset packing rules

## 6.2 Minimal Refactor Strategy (Pragmatic)

To move fast without rewriting everything:
- Keep existing `internal/corelx` parser/AST as the source of truth.
- Introduce interfaces/structs for target-dependent behavior:
  - `TargetProfile`
  - `BuiltinRegistry`
  - `CodegenBackend`
- Thread `TargetProfile` through semantic analysis and codegen.

## 6.3 Suggested Package Layout

Example (does not require immediate full reorg):

```text
internal/corelx/
  ast.go
  lexer.go
  parser.go
  semantic.go          # target-aware semantic checks
  codegen_common.go    # shared helpers
  codegen_dx.go        # Nitro Core DX backend
  codegen_nc8.go       # Nitro Core 8 backend
  targets.go           # TargetProfile definitions/interfaces
  builtins_common.go
  builtins_dx.go
  builtins_nc8.go
```

If you prefer lower churn, keep current files and add target registries first, then split later.

## 7. Target Profile System (Critical)

## 7.1 `TargetProfile` Concept

A target profile describes the hardware-facing contract used by the compiler.

Example fields (conceptual):

```go
type TargetProfile struct {
    Name            string // "dx" or "nc8"
    CPU             CPUProfile
    Memory          MemoryProfile
    ROM             ROMProfile
    PPU             PPUProfile
    APU             APUProfile
    Input           InputProfile
    Builtins        BuiltinRegistry
    Capabilities    map[string]bool
}
```

## 7.2 Why This Matters

This avoids hardcoding DX assumptions into the language:
- register addresses
- max sprite count
- tile formats
- DMA availability
- audio channel features
- advanced blending/matrix features

The language stays stable; only the target profile changes.

## 7.3 Capability Flags (Recommended)

Use explicit capability keys for diagnostics and feature gating, e.g.:
- `ppu.layers.4`
- `ppu.blending`
- `ppu.affine`
- `ppu.hdma`
- `ppu.palette_256`
- `sprite.max_128`
- `audio.streaming`
- `audio.channels.8`
- `dma.async`
- `storage.save_ram`

This is cleaner than checking target name strings everywhere.

## 8. Built-in Namespace Strategy

## 8.1 Keep Existing CoreLX Namespaces Stable

From current CoreLX usage/docs/compiler, preserve namespaces such as:
- `ppu`
- `sprite`
- `oam`
- `apu`
- `gfx`

These should remain the familiar entry points.

## 8.2 Add NC8 Extensions In Two Ways

### Option A (Recommended): Extend Existing Namespaces + Capability Gating
- `ppu.set_mode(...)` (common)
- `ppu.set_blend(...)` (NC8-only)
- `apu.play_note(...)` (common)
- `apu.stream_play(...)` (NC8-only)

Pros:
- Familiar API
- Least fragmentation

### Option B: New Explicit Extension Namespaces
- `ppu8.*`
- `audio8.*`
- `dma.*`

Pros:
- Very explicit portability boundaries

Recommended hybrid:
- Common concepts stay in existing namespaces
- Advanced subsystems get dedicated namespaces (`dma`, `stream`, `mathx`)

## 8.3 Portable API Pattern

For functionality that differs by target power:
- keep a common baseline API
- add optional overloads/advanced functions

Example:
- Common: `sprite.set_pos(id, x, y)`
- NC8: `sprite.set_transform(id, x, y, scaleX, scaleY, rot)`

## 9. Type System Strategy For NC8

Keep the same base scalar types, but allow target-specific aliases/types if useful:
- `color15` / `color16`
- `tile_id`
- `layer_id`
- `channel_id`

Guidelines:
- Prefer aliases/type names that improve readability, but compile to base integer types.
- Avoid introducing complex runtime types that require a runtime system.

Optional addition (future-safe):
- `@target(nc8)` annotations for functions or constants
- `@requires(capability)` on built-ins (compiler metadata only)

## 10. Code Generation Strategy

## 10.1 Shared Codegen vs Target Lowering

Split codegen into:
- Generic language constructs (variables, loops, arithmetic, calls)
- Target built-in lowering (register writes, DMA setup, OAM writes, etc.)

This keeps most of the compiler reusable.

## 10.2 Built-in Lowering Contract

Represent built-ins as typed operations before final emission.

Example conceptual flow:
- parse `ppu.enable_display()`
- semantic resolves to builtin symbol `ppu.enable_display`
- codegen emits builtin op `BuiltinCall{Namespace:"ppu", Name:"enable_display"}`
- backend lowers to target-specific register writes

For NC8, the same builtin name may lower to different registers/sequence if behavior is logically equivalent.

## 10.3 ROM Generation

Keep ROM generation target-specific:
- DX ROM packer
- NC8 ROM packer

Even if instruction set stays similar, do not assume identical headers/maps.

## 11. Hardware Extension Model For Nitro Core 8

This section defines how to expose “more powerful hardware” without changing CoreLX itself.

## 11.1 Extension Categories

Typical NC8 extension areas:
- More layers / modes
- Richer sprite attributes
- More palettes / color formats
- Better DMA / HDMA
- More audio channels / streaming / envelopes
- Timers / interrupts
- Save RAM / cartridge metadata

## 11.2 API Design Rules For Extensions

1. Additive only
- Do not rename common APIs unless absolutely necessary.

2. Typed and explicit
- Prefer `ppu.set_layer_blend(layer: u8, mode: u8, alpha: u8)` over magic constants hidden in strings.

3. Compiler-validated
- Range checks where possible (compile-time for constants).

4. Hardware-mappable
- Built-ins should lower to clear register-level actions.

## 11.3 Example NC8 Built-ins (Illustrative)

These are placeholders until the final NC8 hardware spec exists.

```corelx
function Start()
    ppu.enable_display()
    ppu.set_mode(2)
    ppu.layer_enable(0, true)
    ppu.layer_enable(1, true)

    -- NC8 extension: per-layer blend
    ppu.set_layer_blend(1, ppu.BLEND_ALPHA, 8)

    -- NC8 extension: DMA transfer helper
    dma.copy(vram.asset_addr("tiles"), ppu.VRAM_BASE, 2048)

    -- NC8 extension: streamed audio
    audio.stream_play(0, "intro_pcm")

    while true
        wait_vblank()
```

Compiler behavior:
- `--target nc8`: OK
- `--target dx`: clear capability error(s)

## 12. Diagnostics and Developer Experience

## 12.1 Error Categories (Make This Explicit)

Diagnostics should identify one of:
- Syntax error
- Type error
- Name resolution error
- Target capability error
- Codegen limitation / compiler bug

This is important for debugging multi-layer issues (compiler vs emulator vs ROM logic).

## 12.2 Helpful Error Messages

Good:
- `target 'dx' does not support builtin 'audio.stream_play' (requires capability 'audio.streaming')`

Bad:
- `unknown function`

## 12.3 Portability Hints (Optional But Valuable)

When using an NC8-only feature, optionally emit note:
- `note: this program is no longer portable to target 'dx'`

## 13. CLI Design

## 13.1 Recommended CLI Commands

Reuse current `cmd/corelx` flow where possible.

Examples:

```bash
# Build for Nitro Core DX
corelx build game.corelx game_dx.rom --target dx

# Build for Nitro Core 8
corelx build game.corelx game_nc8.rom --target nc8

# Check portability only
corelx check game.corelx --target portable

# Emit diagnostics + optional IR for debugging
corelx build game.corelx game_nc8.rom --target nc8 --emit-ir
```

## 13.2 Backward Compatibility

If current CLI is positional (`./corelx in.corelx out.rom`), preserve it:
- default target = `dx`
- add optional `--target`

## 14. Testing Strategy (Required)

## 14.1 Shared Language Regression Tests

Compile the same CoreLX programs for both targets (common subset) and verify:
- parse success
- semantic success
- codegen success
- ROM generation success

Use existing CoreLX examples/tests as seed corpus from `test/roms/*.corelx`.

## 14.2 Target-Specific Built-in Tests

For each NC8-only builtin:
- `--target nc8` succeeds
- `--target dx` fails with capability error

## 14.3 Golden Tests (Recommended)

Store compiler outputs/IR snapshots for stable examples:
- typed AST (optional)
- lowered builtin op stream (great for debug)
- ROM metadata/header summary

## 14.4 Determinism Tests

Given same source + same target profile:
- emitted ROM bytes should be identical across runs

This is especially important for hardware reproducibility and FPGA validation workflows.

## 15. Migration Strategy From Current Compiler

## Phase 1: Target Plumbing (Low Risk)
- Add `TargetProfile`
- Add `--target` CLI flag (default `dx`)
- Move existing DX built-ins into target registry
- Keep current behavior unchanged for DX

## Phase 2: Target-Aware Semantic Checks
- Resolve built-ins through profile registry
- Add capability-aware diagnostics

## Phase 3: NC8 Backend Skeleton
- ROM packaging
- Register map placeholders
- Stub built-ins with “not implemented” compiler diagnostics

## Phase 4: NC8 Built-in Implementation
- Implement actual lowering per subsystem (PPU/APU/DMA/etc.)
- Add tests per builtin

## Phase 5: Portability Tooling
- `portable` target mode
- optional lint/hints for non-portable usage

## 16. Developer Rules (To Keep The Language Unified)

1. Do not add NC8 syntax unless a namespace/function addition cannot express it.
2. Prefer target capabilities over target-name conditionals in compiler logic.
3. Keep common semantics in one place (frontend/shared semantic rules).
4. Treat “portable CoreLX” as a first-class use case.
5. Ensure every NC8 extension has a clear error path on DX target.

## 17. Example “Portable + Extended” Workflow

### Portable game prototype
```corelx
function Start()
    ppu.enable_display()
    while true
        wait_vblank()
```

Compiles on both:
- DX ✅
- NC8 ✅

### Enhanced NC8 version (same language, added extension)
```corelx
function Start()
    ppu.enable_display()
    ppu.set_layer_blend(1, ppu.BLEND_ALPHA, 8) -- NC8 extension
    while true
        wait_vblank()
```

Compiles:
- DX ❌ (capability error, expected)
- NC8 ✅

## 18. Recommended First Implementation Checklist

- [ ] Add `TargetProfile` and `BuiltinRegistry` abstractions
- [ ] Add `--target` flag to `cmd/corelx` (default `dx`)
- [ ] Move current DX built-ins into a DX profile registry
- [ ] Thread target profile through semantic analysis
- [ ] Thread target profile through codegen
- [ ] Add `portable` target mode
- [ ] Add first NC8 profile with placeholder capabilities
- [ ] Add 3-5 NC8-only built-ins and capability diagnostics
- [ ] Add tests: common subset compiles on both, NC8 extension fails on DX

## 19. Final Notes

This design intentionally protects your long-term goal:
- **Same CoreLX language family**
- **ROMs that map to real hardware behavior**
- **Compiler architecture that scales to FPGA-backed systems**

Nitro Core DX and Nitro Core 8 should feel like:
- same language
- different hardware profiles
- different power ceilings

That is the right model for maintainability, portability, and hardware reproducibility.
