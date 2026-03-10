# Nitro-Core-DX Project State Review

**Date:** March 9, 2026  
**Type:** Planning and documentation pass only (no implementation changes)  
**Purpose:** High-level repository state review, cleanup/refactor planning, and progress assessment before further implementation.

---

## Executive Summary

Nitro-Core-DX is a retro-inspired fantasy console (Go emulator + FPGA RTL) with SNES-inspired graphics goals and Genesis-like CPU speed. The **emulator codebase is in good shape**: core CPU, memory bus, PPU (including per-layer Matrix Mode and HDMA), APU (legacy + FM extension with YM2608 path), and input are implemented and tested. All relevant Go tests pass. The **primary gaps** are: (1) **remaining compatibility/deprecation debt** (PPU/APU deprecated paths retained intentionally for compatibility); (2) **FPGA PPU** does not yet mirror the “four matrix engines” design (single matrix in RTL); (3) **known spec/impl discrepancies** (PPU register read/write conflicts) already documented in SPEC_AUDIT_DISCREPANCIES; and (4) **continued YM2608 conformance refinement** beyond the now-working runtime/backend plumbing. Technical debt is moderate and localized; the main risk is continuing feature work without preserving clear boundaries between active runtime paths and compatibility shims.

---

## Repository Structure Overview

- **Implementation:** Go under `cmd/` (emulator, corelx, asm, rombuilder, debugger, devkit, testrom, audiotest, etc.) and `internal/` (apu, asm, clock, corelx, cpu, debug, devkit, editor/native, emulator, harness, input, memory, ppu, rom, ui). FPGA RTL under `FPGA/nitro_core_dx_fpga/src/` (cpu, ppu, apu, memory, video, io, top).
- **Documentation:** `docs/` (specifications/, planning/, guides/, testing/, issues/, archive/), root README, SYSTEM_MANUAL, PROGRAMMING_MANUAL, CHANGELOG.
- **Third-party / reference:** `Resources/` (BambooTracker-master, ymfm-main, libOPNMIDI-master, PMDWinS036-master, fmopna_impl.h). Emulator APU uses `internal/apu` with CGo to `Resources/ymfm-main`; BambooTracker is reference only, not linked into the Go build.

**Entry points:** Build/test via Makefile (`test-fast`, `test-full`, `test-emulator`, `ci-audio`, `release-linux`). Main emulator: `cmd/emulator` → `nitro-core-dx`. Emulator uses `memory.Bus` and `memory.Cartridge`; it does **not** use `memory.MemorySystem`.

---

## Completed / Partial / Stale / Duplicate / Abandoned

### Completed (evidence: code + tests + specs)

- **CPU:** Full instruction set, 8 GPRs, 24-bit banked addressing, cycle counting, flags (Z,N,C,V,I,D), interrupts (IRQ/NMI), stack. Matches COMPLETE_HARDWARE_SPECIFICATION_V2.1 and SPEC_AUDIT (no CPU discrepancies).
- **Memory:** Bus-based bank layout (bank 0 WRAM + I/O, 1–125 ROM, 126–127 extended WRAM), LoROM cartridge mapping, I/O dispatch to PPU/APU/Input. System vectors in bank 0 (0xFFE0+). Spec and implementation align.
- **PPU (Go):** VRAM/CGRAM/OAM, 4 BGs with per-layer scroll, tile size, tilemap base; per-layer Matrix Mode (A/B/C/D, center, mirror, outside mode, direct color); HDMA scroll + matrix per scanline; windows; DMA (cycle-accurate path in use); 320×200, 581 dots/scanline, 220 scanlines. `StepPPU()` is the active path; `RenderFrame()` is deprecated.
- **APU (Go):** 4 channels (frequency, volume, waveform, duration), waveforms (sine, square, saw, noise), PCM support, FM extension at 0x9100–0x91FF, YM2608 backend (YMFM) with legacy fallback. Phase is fixed-point; legacy float fields kept for compatibility.
- **Input:** Two controllers, 12 buttons, latch mechanism. SNES-style serial shift register documented.
- **Synchronization:** Clock-driven frame loop (~7.67 MHz CPU, 127,820 cycles/frame), VBlank flag, frame counter. Emulator uses scheduler to step CPU, PPU, APU.

### Partially completed / in progress

- **HDMA:** Table format and per-scanline scroll/matrix reads are implemented; HARDWARE_FEATURES_STATUS still calls out “full per-layer scroll HDMA support” as a need. Code in `scanline.go` (e.g. `updateHDMA`) does read scroll and matrix per layer per scanline—clarification needed whether “partial” refers to edge cases or missing features.
- **YM2608:** FM host interface and YMFM path are operational; conformance refinement and broader subsystem parity (SSG/ADPCM/rhythm) are in progress per planning docs.
- **FPGA PPU:** RTL has 4 BGs and sprites but a **single** matrix (matrix_a–f, matrix_enable), fixed VRAM layout (BG tilemaps at 0x0000–0x3FFF, pattern 0x4000, sprites 0x8000). Go PPU has **four** per-layer matrix engines and configurable tilemap base. FPGA does not yet implement “four simultaneous matrix transformation engines.”

### Stale (documentation)

- **CLEANUP_SUMMARY.md:** Dated Jan 2026; references BUILD_INSTRUCTIONS.md in root; docs/README.md is now the doc map. Content is historical; still useful as snapshot.

### Duplicate / legacy (code)

- **PPU legacy matrix:** `MatrixEnabled`, `MatrixA`–`MatrixD`, etc. on PPU struct map to BG0; per-layer matrix is the real implementation. Kept for “backward compatibility” per comments.
- **PPU RenderFrame():** Deprecated; callers use `StepPPU()`. Still present for compatibility.
- **PPU executeDMA (immediate):** Commented as “legacy function, kept for compatibility”; cycle-accurate DMA runs via `stepDMA()` during `StepPPU`.
- **APU Phase/PhaseIncrement (float):** Deprecated in favor of PhaseFixed/PhaseIncrementFixed; still updated for compatibility.

### Unreconciled “in progress” states

- **Matrix Mode:** Design doc and README describe “four simultaneous matrix transformation engines.” Go PPU implements exactly that (one affine engine per BG). FPGA has one matrix. No conflict in Go; FPGA is behind.
- **CPU speed:** Original design doc (archive) said 10 MHz; current spec and code use ~7.67 MHz (Genesis-like). Reconciled in COMPLETE_HARDWARE_SPECIFICATION_V2.1 and CHANGELOG.
- **ROM size:** Design doc mentioned “64KB per bank” for ROM; evidence in cartridge.go uses 32KB per bank (LoROM: (bank-1)*32768 + (offset-0x8000)). Spec V2.1 and cartridge code align on 32KB mapping.

---

## Technical Debt Hotspots

1. **PPU register map:** VBLANK_FLAG/FRAME_COUNTER reads vs writes to same offsets affect different registers (writes go to BG2 matrix). Documented in SPEC_AUDIT_DISCREPANCIES; the remaining need is explicit documentation discipline.
2. **Deprecated PPU/APU code paths:** Multiple deprecated fields and one deprecated function (RenderFrame) are still retained intentionally. Low runtime cost, but they need a deliberate removal plan to avoid accidental hard breaks.
3. **FPGA vs spec:** FPGA PPU does not implement four matrix engines or configurable tilemap base; FPGA readiness docs may overstate parity.
4. **YM2608 conformance scope:** Runtime path is operational, but broader subsystem parity and timbre/pitch refinement still need continued validation.

---

## CPU Review

**Intended (from specs and design doc):** Custom 16-bit CPU, 8 GPRs, 24-bit banked addressing, orthogonal instruction set, ~7.67 MHz (Genesis-like), interrupts (VBlank, NMI, timer planned).

**Implemented:** Matches. Instruction set (NOP, MOV with multiple modes, ADD, SUB, MUL, DIV, AND, OR, XOR, NOT, SHL, SHR, CMP, branches, JMP, CALL, RET), I/O 8-bit handling for bank 0 offset 0x8000+, interrupt vectors at 0xFFE0–0xFFE3, 7-cycle interrupt overhead. Cycle counting and flags (Z,N,C,V,I,D) implemented.

**Coherent:** Yes. Single decode/execute path, clear MemoryInterface, no duplicate cores.

**Matches project goals:** Yes. CPU design review (SNES/Genesis comparison) concluded the set is sufficient for “SNES-like graphics + Genesis-like CPU power.”

**Missing / questionable:** CPU_AMPED_EXTENSION_DESIGN describes future indexed/indirect ops, shifts/rotates, 8-bit ADD/SUB—not implemented; no architectural conflict. Design is forward-compatible.

**Design drift:** None identified. Spec and code aligned per SPEC_AUDIT_DISCREPANCIES.

**Follow-up:** None for current scope. When implementing amped extension, follow CPU_AMPED_EXTENSION_DESIGN and FPGA spec.

---

## APU Review

**Intended:** Legacy 4-channel PSG (sine, square, saw, noise) + PCM, plus FM extension (YM2608 direction) at 0x9100–0x91FF, mixer combining legacy and FM, runtime backend selection (YMFM vs legacy).

**Implemented:** Legacy APU complete (frequency, volume, waveform, duration, master volume, completion status). FM extension: MMIO at 0x9100–0x91FF, FMOPM struct, YMFM backend via CGo to Resources/ymfm-main, legacy fallback, `NCDX_YM_BACKEND=auto|ymfm|legacy`. Timer/IRQ bridge and deterministic placeholder timing present. Tests force `legacy` for determinism where needed.

**Coherent:** Yes. Single APU struct with optional FM; backend selection at init; no duplicate channel logic.

**Matches project goals:** YM2608-style APU work is underway; legacy preserved; FM host interface and playback path operational. Conformance refinement and full subsystem parity (SSG/ADPCM/rhythm) are documented as in progress.

**Missing / questionable:** Deprecated float phase fields still exist for compatibility; remove only with an explicit savestate migration/version step. Broader YM2608 test coverage and subsystem parity work remain ongoing.

**Design drift:** None. APU_FM_OPM_EXTENSION_SPEC and planning docs describe current direction (extension, not replacement; YM2608 target).

**Follow-up:** Keep YM2608 docs/tests aligned with the guarded-tooling model; consider a versioned deprecation path for APU float phase fields; continue YM2608 conformance work per planning.

---

## PPU Review

**Intended (project goals):** PPU broadly SNES-inspired, with **four simultaneous matrix transformation engines** for multiple Mode-7-like transformations. 4 BGs, 128 sprites, tile + sprite rendering, windows, HDMA, 320×200, 256 colors (CGRAM).

**Implemented (Go):** 4 background layers with independent scroll, tilemap base, tile size; **per-layer** Matrix Mode (A,B,C,D 8.8 fixed-point, center, mirror, outside mode, direct color); HDMA for scroll and matrix per scanline (table in VRAM, 16 bytes per layer when matrix enabled); 128 sprites with priority, blending, transparency; windows; VRAM/CGRAM/OAM; cycle-accurate DMA; 581 dots/scanline, 220 scanlines, VBlank at 200. Single `renderDotMatrixMode(layerNum, x, y)` used for all four BGs—i.e. four logical “engines” (one per layer), not four separate hardware blocks.

**Coherent:** Yes. One render path (StepPPU → stepDot → priority sort → BG layers then sprites → renderDotMatrixMode or tile path). Legacy matrix and RenderFrame are clearly marked deprecated.

**Matches project goals:** Yes for “four simultaneous matrix transformations”—each BG can have an independent affine transform. SNES Mode 7 had one such layer; Nitro-Core-DX has four. Implementation is aligned with that goal.

**Missing / inconsistent:** (1) VBLANK_FLAG/FRAME_COUNTER write behavior is still non-obvious because writes alias BG2 matrix registers. (2) Vertical sprites for Matrix Mode (sprites scaled/positioned by transform) and large-world tile stitching are listed as future in HARDWARE_FEATURES_STATUS—not in scope of “current” PPU.

**Architecturally questionable:** None. Per-layer matrix and HDMA matrix updates are structured cleanly for future FPGA (e.g. four parallel units).

**Design drift:** Implementation has moved ahead of some docs (SYSTEM_MANUAL). No drift away from SNES-like + 4 matrix engines.

**Follow-up:** Keep the read/write conflict documentation explicit for 0x803E–0x8040; when bringing FPGA PPU to parity, add four matrix units and document VRAM/tilemap layout alignment with Go.

---

## Memory Layout Review

**Intended:** Bank 0: WRAM 0x0000–0x7FFF, I/O 0x8000–0xFFDF, system vectors 0xFFE0–0xFFFF. Banks 1–125: ROM (LoROM). Banks 126–127: extended WRAM. I/O: PPU 0x8000–0x8FFF, APU 0x9000–0x9FFF, Input 0xA000–0xAFFF.

**Implemented:** Bus implements exactly that. Cartridge uses LoROM: `(bank-1)*32768 + (offset-0x8000)`. ROM size up to 7.8 MB (125×32 KB).

**Coherent:** Bus is single source of truth for the running emulator. Cartridge is the only ROM backend used by Bus.

**Matches project goals:** Yes. Matches COMPLETE_HARDWARE_SPECIFICATION_V2.1 and SPEC_AUDIT (memory section matches).

**Missing / inconsistent:** No open-bus behavior (spec notes this as optional). The prior MemorySystem duplication issue is resolved.

**Follow-up:** Optionally add open-bus later if ROM compatibility demands it.

---

## PPU Comparison: SNES-like Goals vs Genesis-like Traits

**Goal stated:** PPU broadly similar in spirit to SNES PPU, expanded with four simultaneous matrix transformation engines (Mode-7-style).

### How close is the current PPU to that goal?

- **Very close.** The Go PPU has: 4 BGs, 128 sprites, tilemaps, CGRAM (256 colors), windows, HDMA (per-scanline scroll and matrix), and **per-layer** affine (Mode 7–style) matrix with A/B/C/D, center, mirror, outside mode, direct color. That is “SNES-like” plus four matrix engines instead of one.

### Is the implementation trending toward SNES-like behavior or drifting elsewhere?

- **Trending toward SNES-like.** Affine math is 8.8 fixed-point, screen-to-background transform, tilemap + tile data fetch, CGRAM lookup—all in line with SNES Mode 7. No drift toward a different paradigm (e.g. bitmap or Genesis-style plane/tile mix in a different way).

### Aside from matrix math, how does the current PPU compare conceptually to the SNES PPU?

- **Similar:** 4 background layers, tile + tilemap, sprite OAM, priority (BG and sprite), windows, HDMA for per-scanline updates, VBlank flag, 8×8/16×16 tiles, 4bpp tiles, palette (CGRAM). **Differences:** Resolution 320×200 (vs SNES 256×224–239); single CGRAM size; no SNES-specific OBJ mode bits or EXTBG; sprite size and count differ. Conceptually it is the same family: tile layers + sprites + windows + HDMA.

### Where is it more like the Genesis VDP?

- **Limited.** Genesis VDP has 2–4 scroll planes (depending on mode), 80-tile line buffer, no affine matrix in hardware. Nitro-Core-DX has 4 BGs with optional affine per layer and HDMA—closer to SNES. Timing (dots per scanline, etc.) is documented as “Genesis-like” for CPU clock, not for PPU layout; PPU structure (layers, sprites, windows, HDMA) is SNES-like. So: “Genesis-like” applies to CPU speed and possibly timing constants, not to the PPU’s conceptual layout.

### Features: present, missing, simplified, or structurally different

- **Present:** 4 BGs, per-BG scroll and tilemap base, per-BG matrix (4 engines), HDMA scroll+matrix, 128 sprites, priority and blending, windows, mosaic, DMA to VRAM/CGRAM/OAM, VBlank, frame counter. **Missing (documented as future):** Vertical sprites (sprites in matrix space), large-world tile stitching. **Simplified:** One tilemap size (32×32) per layer; no resolution modes. **Structurally different:** Four independent matrix engines vs SNES’s single Mode 7 layer—expansion, not reduction.

### Is the current design capable of supporting the intended 4-transformer direction cleanly?

- **Yes.** Per-layer state (BackgroundLayer with Matrix* fields) and one `renderDotMatrixMode(layerNum, …)` give a clear mapping to four logical engines. Adding four parallel units on FPGA would align with this. No structural blocker.

### Structural issues that could make advanced transformation features difficult later

- **Few.** (1) Per-dot sprite list build (all 128 sprites scanned per dot) could become a bottleneck if sprite count or effects grow—optimization (e.g. scanline-based sprite list) would help. (2) FPGA PPU currently has one matrix and fixed VRAM layout—needs to be brought in line with Go for “four engines” and configurable tilemap base. (3) HDMA table format (16 bytes per layer with matrix) is fixed; if future features need more per-scanline data, the format might need a version or extension.

---

## What Is Meeting Goals

- **CPU:** Complete, spec-aligned, sufficient for stated “SNES-like graphics + Genesis-like CPU power” per design review.
- **Memory/Bus:** Correct layout, LoROM, I/O routing; single active path (Bus + Cartridge).
- **PPU (Go):** Four matrix engines (per-layer), Mode 7–style math, HDMA, sprites, windows, DMA—matches “SNES-like + 4 matrix engines” goal.
- **APU:** Legacy channels + FM extension with YM2608 path and fallback; matches “YM2608-style APU work underway.”
- **Input, sync, clock:** Implemented and used by emulator.
- **Tests:** Core packages and emulator tests pass (test-fast, test-full, test-emulator run successfully).
- **Docs (where updated):** COMPLETE_HARDWARE_SPECIFICATION_V2.1, HARDWARE_FEATURES_STATUS, SPEC_AUDIT_DISCREPANCIES, APU_FM_OPM_EXTENSION_SPEC, and planning docs reflect current direction.

---

## What Is Falling Short

- **Documentation:** Some review/planning docs age quickly; active runtime/spec docs are now materially better aligned than they were at review time.
- **Spec/impl:** PPU register read/write conflicts remain a real documentation hazard (documented in SPEC_AUDIT_DISCREPANCIES).
- **FPGA PPU:** One matrix, fixed VRAM layout—does not yet match “four matrix engines” or configurable tilemap base.
- **Compatibility debt:** Deprecated PPU/APU paths remain and should be managed explicitly rather than allowed to drift.

---

## Cleanup/Refactor Recommendations

- **Safe:** Keep active specs/manuals aligned with current runtime/backend behavior; keep 0x803E–0x8040 register conflict documentation explicit; continue low-risk doc sync as cleanup proceeds.
- **Risky (review first):** Removing deprecated PPU/APU compatibility paths (confirm no external callers or savestate breakage); advancing FPGA parity claims beyond implemented RTL.
- **Mild refactor:** Name “legacy” vs “current” in memory package (e.g. document Bus as canonical); add a short “Implementation status” note in specs index pointing to HARDWARE_FEATURES_STATUS and SPEC_AUDIT_DISCREPANCIES.
- **Do not touch yet:** Active PPU/APU/CPU execution paths; ROM format; CORELX toolchain; Resources/ contents (reference only); FPGA RTL beyond documentation of current gap.

---

## Risks and Unknowns

1. **Savestates:** If savestates persist deprecated fields (e.g. PPU legacy matrix, APU float phase), removing those fields could break save compatibility—needs versioning/migration before removal.
2. **FPGA parity:** Extent to which games or tests assume FPGA behavior identical to Go is unclear; documenting FPGA PPU as “subset” until four-matrix and tilemap layout are aligned reduces risk.
3. **HDMA “partial”:** HARDWARE_FEATURES_STATUS says HDMA scroll is “structure exists” and “needs full per-layer scroll HDMA support.” Code appears to implement it; clarify and update status to avoid duplicate work or wrong assumptions.
4. **External dependents:** Unknown whether any external tool or ROM depends on `RenderFrame()`, `executeDMA()`, or legacy PPU matrix registers; recommend release-note/deprecation discipline before removal.

---

## Prioritized Recommendations

1. **High:** Clarify HDMA status in HARDWARE_FEATURES_STATUS; keep 0x803E–0x8040 register conflict documentation explicit; keep active docs aligned with current guarded YM2608 tooling/runtime behavior.
2. **Medium:** Plan deprecation/removal windows for APU float phase and PPU RenderFrame/legacy matrix only after confirming caller and savestate expectations.
3. **Medium:** Document FPGA PPU gap (four matrix engines, tilemap layout) in FPGA or spec docs before RTL parity work resumes.
4. **Ongoing:** Continue YM2608 conformance and APU stabilization per planning; when touching FPGA PPU, align with Go (four matrix engines, configurable tilemap base).

---

*End of project-state-review.*
