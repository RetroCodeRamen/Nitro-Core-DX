# Nitro-Core-DX Project Cleanup Plan

**Date:** March 9, 2026  
**Type:** Proposed changes only—do not execute in this pass.  
**Purpose:** Categorized list of cleanup, archive, refactor, and documentation actions for a future approved cleanup pass.

---

## Safe Cleanup Candidates

| Area / File / Module | Issue | Why change | Risk | Recommended action | Category |
|----------------------|--------|------------|------|--------------------|----------|
| `docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md` | PPU register 0x803E–0x8040: write behavior (to BG2 matrix) not documented | Spec should describe read-only system registers and that writes hit BG2 matrix | Low | Add short note in PPU register section | document |
| `internal/ppu/ppu.go` (Read8) | DMA_LENGTH read from 0x8061/0x8062 instead of 0x8066/0x8067 | Matches write addresses and spec; fixes bug | Low | Read DMA_LENGTH from 0x66/0x67 in Read8 | cleanup |
| `docs/reports/` | New reports (this file, project-state-review, project-next-steps) | Already created in this planning pass | None | Keep; no further change | — |
| `SYSTEM_MANUAL.md` (Matrix Mode sections) | States Matrix Mode “structure in place, needs transformation matrix implementation” and “❌ Matrix Mode transformation calculations” | Implementation exists in ppu/scanline.go and ppu.go; text is stale | Low | Replace with “implemented” and remove ❌ lines | document |

---

## Risky Cleanup Candidates (Requiring Approval)

| Area / File / Module | Issue | Why change | Risk | Recommended action | Category |
|----------------------|--------|------------|------|--------------------|----------|
| `internal/memory/memory.go` (MemorySystem) | Entire type and all methods unused by emulator; Bus is the only path | Removes dead code and avoids confusion | Medium: unknown external or test use | After confirming no imports and no savestate use: remove file or move to archive; document Bus as canonical | cleanup / archive |
| `internal/ppu/ppu.go` | Legacy matrix fields (MatrixEnabled, MatrixA–D, Center, Mirror, etc.) and legacy path that maps to BG0 | Reduces duplication and comment noise | Medium: ROMs or tests might write legacy registers | After grep for writes to legacy matrix regs and checking savestates: deprecate clearly or remove and route to BG0 only | refactor / cleanup |
| `internal/ppu/ppu.go` | `RenderFrame()` deprecated; only StepPPU used by emulator | Removes dead API surface | Low–medium: external callers? | After grep for RenderFrame(): remove or keep with “do not use” comment | cleanup |
| `internal/ppu/ppu.go` | `executeDMA()` (immediate DMA) marked legacy; cycle-accurate stepDMA used | Same as above | Low | Document or remove after confirming no callers | cleanup |
| `internal/apu/apu.go` | Phase, PhaseIncrement (float) deprecated; PhaseFixed used | Cleaner struct, less dual update | Low–medium: savestates or external reads? | After checking savestates and callers: stop updating float fields and eventually remove | refactor |
| `Makefile` | Target `check-ym2608-policy` (and related) invokes `./cmd/ym2608_manifest_gen` which does not exist | Build/docs consistency | Medium: CI or scripts may depend on target | Either restore `cmd/ym2608_manifest_gen` or remove/guard targets and update docs (YM2608_IMPLEMENTATION_NOTES, YM2608_SOURCE_SELECTION) | cleanup / document |

---

## Archive / Legacy Candidates

| Area / File / Module | Issue | Why change | Risk | Recommended action | Category |
|----------------------|--------|------------|------|--------------------|----------|
| `internal/memory/memory.go` | MemorySystem unused but may have historical value | Preserve history without cluttering active code | Low if moved with clear label | Option A: Move to `internal/memory/legacy.go` or `docs/archive/code_snippets/` as reference. Option B: Delete after approval. Document decision in docs/README or CLEANUP_SUMMARY | archive |
| `docs/CLEANUP_SUMMARY.md` | Dated Jan 2026; references old doc locations | Keep as historical snapshot | None | Add one-line note at top: “Historical snapshot; current doc map is docs/README.md” | document |
| `docs/archive/NITRO_CORE_DX_DESIGN_DOCUMENT.md` | Original design (10 MHz, etc.); superseded by V2.1 spec | Already in archive | None | Leave as-is; reference from specs README if needed | — |

---

## Mild Refactor Candidates

| Area / File / Module | Issue | Why change | Risk | Recommended action | Category |
|----------------------|--------|------------|------|--------------------|----------|
| `internal/memory` package | Two types (Bus, MemorySystem) with overlapping responsibility | Clear ownership: Bus = emulator memory model | Low | Add package comment: “Bus is the canonical memory model for the emulator. MemorySystem is legacy/unused.” If MemorySystem is removed, omit. | refactor |
| `internal/ppu/ppu.go` (struct comments) | Legacy matrix and DMA fields mixed with active ones | Easier to see what is deprecated | Low | Group deprecated fields under a single “Legacy (deprecated)” comment block | refactor |
| `internal/apu/apu.go` (struct comments) | Same for Phase/PhaseIncrement | Same | Low | Keep “DEPRECATED” comment; consider grouping | refactor |
| Naming | “Legacy” used for APU backend and PPU matrix | Consistency | Low | No rename required; ensure docs use “legacy” consistently (backend vs matrix) | refactor (optional) |

---

## Documentation Fixes Needed

| Area / File / Module | Issue | Why change | Risk | Recommended action | Category |
|----------------------|--------|------------|------|--------------------|----------|
| `SYSTEM_MANUAL.md` | Matrix Mode and transformation described as not implemented | Accurate status for readers and future contributors | Low | Update sections ~340 and ~362 to state Matrix Mode is implemented (per-layer, HDMA); remove ❌ for transformation calculations | document |
| `docs/specifications/README.md` (or index) | No pointer to implementation status vs spec | Hard to find `docs/specifications/SPEC_AUDIT_DISCREPANCIES.md` and `docs/HARDWARE_FEATURES_STATUS.md` | Low | Add “Implementation status” line linking to `../HARDWARE_FEATURES_STATUS.md` and `SPEC_AUDIT_DISCREPANCIES.md` | document |
| `docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md` | PPU 0x803E–0x8040: read vs write behavior | Completeness and consistency with impl | Low | Add note: reads = VBLANK_FLAG / FRAME_COUNTER; writes = BG2 matrix registers | document |
| YM2608 docs + Makefile | Reference to `cmd/ym2608_manifest_gen` | Tool missing; readers and CI may be confused | Medium | Either add tool or replace references with “(optional; tool not present)” and adjust Makefile targets | document |

---

## Areas to Leave Alone for Now

| Area | Reason |
|------|--------|
| CPU execution path (`internal/cpu/`) | Stable; spec matches; no known debt. |
| Bus and Cartridge (`internal/memory/bus.go`, `cartridge.go`) | Single active memory path; no change needed. |
| PPU StepPPU / scanline / renderDot* (active path) | Core behavior; only register/docs fixes recommended. |
| APU channel and FM host logic (non-deprecated) | Working; only deprecated field cleanup later. |
| CORELX, asm, rom, devkit, emulator frame loop | Out of scope for this cleanup plan. |
| FPGA RTL (beyond documenting gap) | Do not refactor in this pass; document PPU matrix/tilemap gap only. |
| Resources/ (BambooTracker, ymfm, etc.) | Reference; do not modify. |
| Test ROMs and game projects | Do not change unless cleanup requires it. |

---

## Suggested Order of Operations for a Future Cleanup Pass

1. **Resolve missing tool (unblocks checks first)**  
   - Either introduce `cmd/ym2608_manifest_gen` or update Makefile and YM2608 docs so they do not reference a missing binary.  
   - Run or adjust CI/local flows that use `check-ym2608-policy`.

2. **Documentation only (low risk)**  
   - Update SYSTEM_MANUAL Matrix Mode sections.  
   - Add PPU 0x803E–0x8040 note to COMPLETE_HARDWARE_SPECIFICATION_V2.1.  
   - Add “Implementation status” link in specs index.  
   - Add one-line “historical” note to CLEANUP_SUMMARY.

3. **Single code fix (spec-aligned)**  
   - Fix PPU Read8 DMA_LENGTH to use 0x8066/0x8067.  
   - Re-run PPU and emulator tests.

4. **Decide MemorySystem fate**  
   - Grep for MemorySystem/NewMemorySystem; check savestates.  
   - Either remove `memory.go` (or MemorySystem only) or move to legacy/archive and document.

5. **Deprecated PPU/APU (after verification)**  
   - Grep for RenderFrame, executeDMA, legacy matrix register writes, APU Phase (float) usage.  
   - Check savestate format for deprecated fields.  
   - Add a deprecation window (at least one release) with compatibility shims before hard removal.
   - Then: remove or strongly deprecate RenderFrame, executeDMA; consider legacy matrix and APU float removal in a later pass.

6. **Ongoing**  
   - Keep HARDWARE_FEATURES_STATUS and SPEC_AUDIT_DISCREPANCIES updated as fixes land.  
   - When touching FPGA PPU, document and then implement four-matrix and tilemap layout alignment with Go.

---

*End of project-cleanup-plan.*
