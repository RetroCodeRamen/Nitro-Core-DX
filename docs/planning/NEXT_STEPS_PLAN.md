# Nitro-Core-DX — Comprehensive Next Steps Plan

> Created: 2026-04-20  
> Based on: Full project review (architecture, implementation, dev state, documentation)  
> Status: Active, partially superseded — see status note below

> **Status note (2026-06-12):** Phase 0.1's regression test
> (`TestMatrixFloorBillboardReferenceFramebufferChangesWithInput`) now passes.
> The NitroPackInDemo ROM demo is complete (M1-M7), and the CoreLX v1 language
> design is settled. The current sequenced workstream is the M8 CoreLX rebuild
> (`Games/NitroPackInDemo/CORELX_EXTRACTION.md` §12); CoreLX-related phases in
> this plan should be read through that decision record.

This document captures every open workstream identified in the April 2026 project review, sequenced by dependency and priority. It is organized into phases. Phases do not need to be strictly serial — items within a phase and across independent tracks can proceed in parallel.

---

## Phase 0 — Immediate Blockers (Do First)

These must be resolved before any work in Phase 1 can be trusted.

### 0.1 Fix `TestMatrixFloorBillboardReferenceFramebufferChangesWithInput` regression
- **Problem:** WRAM camera and `plane1.CameraX` diverge (`2` vs `0x8002`); framebuffer checksum does not change after input on the reference ROM path.
- **Where:** `internal/ppu/scanline.go`, `internal/emulator/emulator.go`, reference ROM in `test/`
- **Goal:** Test passes in both debug and optimized modes, determinism harness stays green.
- **Why now:** This test is a core gate. A failing reference ROM signal means the matrix-floor rendering path cannot be trusted as a baseline while actively iterating NitroPackInDemo.

### 0.2 Add committed binaries and logs to `.gitignore`
- **Problem:** `nitro-core-dx` (39 MB), `emulator` (39 MB), `corelx_devkit` (36 MB), `debugger`, `testrom`, `audiotest`, `logs_*.txt`, and other build artifacts are tracked in the repo root.
- **Action:** Add entries to `.gitignore`, remove tracked artifacts with `git rm --cached`, confirm CI release workflow still produces them as build outputs.
- **Why now:** Repo bloat compounds with every commit. Clean this before adding more game assets.

---

## Phase 1 — NitroPackInDemo Completion (V1-GAME)

The pack-in demo is the primary gating deliverable for V1. It must be built ROM-first through completion before CoreLX feature extraction can happen. Milestones 1–3 are done. Milestones 4–8 remain.

### 1.1 Milestone 4 — Interior Room
- Enterable building: transition from overworld to interior scene on interact
- Interior tilemap render with correct layer/priority assignments
- Exit interaction returns player to overworld at correct position

### 1.2 Milestone 5 — NPC Interaction
- At least one NPC sprite placed in interior
- Proximity/facing detection triggers interaction
- NPC responds with a fixed dialogue trigger

### 1.3 Milestone 6 — Dialogue System
- Text box rendering (HDMA-backed window or BG overlay)
- Character-by-character text advance on button press
- Multi-page dialogue support
- Dialogue terminates cleanly and returns control to player

### 1.4 Milestone 7 — Credits Screen
- Full-screen scroll or static credits scene
- Returns to title on completion
- Music/SFX hook (can be silent placeholder if YM2608 conformance is not yet cleared)

### 1.5 Milestone 8a — Polish Pass
- Title screen, overworld, interior, dialogue, credits flow end-to-end without hang or visual glitch
- Input feel review (camera speed, collision margins)
- Audio pass (APU legacy channels minimum; FM if APU exit checklist is cleared)
- Frame-pacing review (sustained 60 FPS, no audio drift)

### 1.6 Milestone 8b — CoreLX Feature Extraction
- Audit every hardware call and pattern used in the ROM-first `.asm` implementation
- Identify which patterns need first-class CoreLX syntax or builtins
- Document the required CoreLX API surface (feeds directly into Phase 3)

### 1.7 Milestone 8c — CoreLX Rebuild
- Rewrite NitroPackInDemo in CoreLX from scratch using the API surface defined in 1.6
- The CoreLX-built ROM must produce a pixel-identical or better result to the `.asm` ROM
- This is the acceptance proof for the CoreLX compiler against a real game

---

## Phase 2 — Dev Kit Tools (V1-EDITOR, V1-DBG, V1-TOOLS)

Tracks the charter items for the Dev Kit IDE. These can run in parallel with Phase 1 game work since they are independent of the ROM codebase.

### 2.1 Native Editor Stabilization (V1-EDITOR-1)
- Resolve any remaining ownership/state bugs in `internal/editor/native/`
- Ensure cursor, selection, undo/redo, and large-file performance are stable
- Acceptance: no editor state corruption on hour-long editing sessions

### 2.2 CoreLX Squiggle Diagnostics (V1-EDITOR-2)
- Wire compiler structured diagnostics from `internal/corelx/` into the editor underline/squiggle system
- Red squiggles for errors, yellow for warnings, on the correct line/col
- Tooltip shows the diagnostic message on hover

### 2.3 Find/Replace + Go-to-Line + Symbol Navigation (V1-EDITOR-3 / V1-EDITOR-4)
- Find and replace with regex support
- Ctrl+G go-to-line
- Symbol list (functions, labels) populated from the CoreLX semantic pass

### 2.4 Debugger Wiring into Dev Kit UX (V1-DBG-1..4)
- `internal/debug/debugger.go` scaffolding exists but is not connected to the Fyne UI
- **DBG-1:** Pause / resume execution from UI
- **DBG-2:** Frame-step (advance one frame, freeze)
- **DBG-3:** CPU instruction-step (single opcode, with register dump)
- **DBG-4:** Register panel + PC display + memory watch panel in Dev Kit window
- This is the highest-leverage tool investment after the editor — actively debugging NitroPackInDemo would be faster with it

### 2.5 Tilemap Lab Round-Trip (V1-TOOLS-2)
- Complete the asset flow: draw in Tilemap Lab → export asset → CoreLX embeds it → emulator renders it
- Live preview of the tilemap in the emulator pane
- Round-trip: load a ROM back and re-display its tilemap assets in the editor

### 2.6 Sound Studio (V1-TOOLS-3)
- **Blocked until:** APU stabilization exit checklist (Phase 4) is cleared AND Tilemap Lab round-trip (2.5) is done
- Basic channel mixer UI for legacy APU channels
- YM2608 patch editor (operator parameters, envelope)
- Export to `.ncdxmusic` format / inline CoreLX audio calls
- Preview playback via the embedded emulator

---

## Phase 3 — CoreLX Compiler (Post-Game Extraction)

Do not begin this phase until Milestone 8b (feature extraction) is complete. The language surface must be known before refactoring the compiler.

### 3.1 CoreLX API Surface Completion
- Implement any builtins or syntax identified in Milestone 8b that are missing from the current compiler
- Extend the language reference in `docs/CORELX.md` to match

### 3.2 `codegen.go` Split
- `internal/corelx/codegen.go` is 4,394 LOC in one file — the largest maintainability risk in the project
- **After 8b only:** Split into logical domain files:
  - `codegen_expr.go` — expression evaluation and type coercion
  - `codegen_stmt.go` — control flow, assignments, calls
  - `codegen_hardware.go` — PPU/APU/input hardware builtins
  - `codegen_assets.go` — asset manifest, tile/sprite/music embedding
  - `codegen_rom.go` — ROM layout, bank allocation, header emission
- Each file should be independently testable

### 3.3 Parser Error Handling
- `internal/corelx/parser.go` uses `recover()` to catch panics as "parser errors"
- Replace with explicit error-return propagation throughout the parse tree
- Goal: no panics in the compiler pipeline on any user-authored input

### 3.4 CoreLX Documentation Alignment (V1-DOCS-1..3)
- Audit `docs/CORELX.md` §12 (Built-in Functions Reference) against the live codegen surface
- Add runnable in-app snippets for each hardware category (PPU, APU, input, matrix)
- Ensure every example in the docs compiles cleanly with the current compiler

---

## Phase 4 — APU / YM2608 Stabilization (V1-AUDIO)

### 4.1 YM2608 Conformance Refinement (V1-AUDIO-1..3)
- Work through `docs/archive/plans/APU_STABILIZATION_EXIT_CHECKLIST.md` item by item (checklist archived after stabilization)
- Golden audio comparisons for each operator/envelope configuration
- Subsystem parity: every MMIO register at `0x9100-0x91FF` behaves per spec

### 4.2 Exit Checklist Sign-Off (V1-AUDIO-4)
- All items in the checklist marked green
- `ci-audio` Makefile target passes clean
- YM2608 manifest/policy gates (`check-ym2608-manifest-strict`, `check-ym2608-policy`) pass

### 4.3 Stale Comment Cleanup
- `internal/apu/fm_opm.go:17` — remove "stubbed in phase 1" comment, ymfm backend now drives it

---

## Phase 5 — Documentation Cleanup (V1-DOCS)

Can be done incrementally alongside any other phase. None of these are blockers for code work.

### 5.1 Archive / Rewrite `MASTER_PLAN.md` — DONE 2026-06-12 (moved to `docs/archive/plans/MASTER_PLAN_CONSOLIDATED_2026-01.md`)
- Sections still showing "⏳ NOT IMPLEMENTED" for Interrupt System and Matrix Mode (both complete) must be struck or archived
- Either: update the status table to reflect reality, or move the file fully into `docs/archive/` and link from `docs/README.md`

### 5.2 Consolidate Hardware Specifications
- Three hardware spec files coexist: `HARDWARE_SPECIFICATION.md`, `COMPLETE_HARDWARE_SPECIFICATION.md`, `COMPLETE_HARDWARE_SPECIFICATION_V2.1.md`
- Only V2.1 is current — move the others to `docs/archive/`
- Update `docs/README.md` doc map to point only at V2.1

### 5.3 Revise `SYSTEM_MANUAL.md`
- Currently marked "under revision" and not trusted
- Align with current hardware spec V2.1 and live emulator behavior
- Remove or annotate any sections that describe unimplemented features

### 5.4 Revise `PROGRAMMING_MANUAL.md`
- Currently marked "under revision" and not trusted
- Update to reflect current CoreLX syntax, assembler ISA, memory map, MMIO addresses
- This is the primary reference for game developers — must be authoritative by V1

### 5.5 Write `EMULATOR_ARCHITECTURE.md`
- No single document explains the clock-driven CPU/PPU/APU scheduling design
- Document: `MasterClock.Step()` contract, fractional APU accumulator, frame execution loop, determinism harness, save-state protocol
- Audience: future contributors and FPGA implementers

### 5.6 `DEV_TOOLS_IMPLEMENTATION_PLAN.md` Review (plan archived to `docs/archive/plans/` 2026-06-12)
- Currently contains stale status markers similar to `MASTER_PLAN.md`
- Update or archive

### 5.7 `WEEKLY_UPDATE_LOG.md`
- Has one entry from early 2025, nothing since
- Decision: either commit to maintaining it weekly (move to `docs/` and update the format) or formally mark it as archived

### 5.8 `Note To Codex` Formalization
- The repo root `Note To Codex` file is an informal session handoff note
- Establish a convention: either keep it as a living "current session context" file that gets overwritten, or replace it with a proper `CURRENT_FOCUS.md` in `docs/` that is part of the normal doc map

---

## Phase 6 — Codebase Health (Technical Debt)

Low urgency — none of these are blocking — but worth scheduling before V1 release.

### 6.1 CPU PBR/PCBank Defensive Sync
- `internal/cpu/cpu.go` has a defensive self-heal where `PBR != PCBank` triggers a sync rather than maintaining a strict invariant
- Audit the two entry points where this can be violated (RET, interrupt) and enforce the invariant there instead
- Remove the defensive sync once the invariant is proven

### 6.2 Dual Frame Execution Paths
- `internal/emulator/emulator.go` has two divergent code paths: cycle-by-cycle and chunked
- Both are tested for equivalence by the determinism harness, but duplication invites drift
- Evaluate whether one can be retired or whether both are load-bearing for different use cases (record/playback vs. performance)

### 6.3 CGRAM 16-bit Write Special Case
- `internal/memory/bus.go:147-159` hardcodes a CGRAM-addr/data special case for 16-bit writes
- Refactor to route through the PPU MMIO handler uniformly like all other 16-bit MMIO writes

### 6.4 YM Burst Streamer Bank Rollover
- `internal/memory/bus.go:257-291` does raw cartridge reads with bank rollover at `off == 0`
- The rest of the memory system does not model bank boundary behavior this way
- Audit and align before FPGA translation work begins

---

## Phase 7 — V1 Acceptance Gate

Before calling V1 done, all of the following must be true per `docs/planning/V1_ACCEPTANCE.md`:

- [ ] NitroPackInDemo runs start-to-finish (title → overworld → interior → dialogue → credits) in the emulator without crash or glitch
- [ ] NitroPackInDemo CoreLX rebuild (Milestone 8c) produces a correct ROM
- [ ] Dev Kit: editor stable, diagnostics wired, debugger connected (pause/step/registers)
- [ ] Tilemap Lab round-trip working
- [ ] APU stabilization exit checklist signed off
- [ ] `SYSTEM_MANUAL.md` and `PROGRAMMING_MANUAL.md` marked current and accurate
- [ ] `docs/README.md` doc map reflects current state of all docs
- [ ] CI green: all test tiers pass, YM2608 policy gates pass
- [ ] Linux and Windows release builds produce correct artifacts

---

## Phase 8 — Post-V1 (Parking Lot)

These are acknowledged goals deferred until after V1. Do not let them pull scope into V1.

- **Sound Studio** full feature set (beyond basic V1 tool)
- **Vertical-sprite 3D scaling** for Matrix Mode
- **Large-world tilemap support** (streaming, extended map format)
- **Dev Kit**: additional game templates, asset import pipeline improvements
- **CoreLX v2 language features** (structs, modules, multiple files)
- **FPGA bring-up** (`docs/specifications/FPGA_ARCHITECTURE_RECOMMENDATION.md`, `FPGA_IMPLEMENTATION_SPECIFICATION.md`)
- **Physical console** (3D-printable shell + custom controller PCB)
- **≥3 shipped games** (required before FPGA/physical hardware investment per `MASTER_PLAN.md`)

---

## Dependency Graph (summary)

```
Phase 0 (blockers)
  └── Phase 1 (NitroPackInDemo)
        ├── 1.6/1.7 (CoreLX extraction) ──► Phase 3 (CoreLX compiler)
        └── unblocks Phase 7 (V1 gate)

Phase 2 (Dev Kit tools) — parallel to Phase 1
  └── 2.6 (Sound Studio) blocked on Phase 4 (APU) + 2.5 (Tilemap Lab)

Phase 4 (APU stabilization) — parallel to Phase 1
  └── unblocks 2.6 (Sound Studio)

Phase 5 (Docs) — parallel to everything, incremental

Phase 6 (Tech debt) — parallel, low urgency

Phase 7 (V1 gate) — requires Phases 1, 2, 4, 5 complete

Phase 8 (Post-V1) — after Phase 7 only
```

---

*This plan supersedes `docs/reports/project-next-steps.md` for active planning purposes. `docs/planning/V1_CHARTER.md` and `docs/planning/V1_ACCEPTANCE.md` remain the authoritative V1 scope documents.*
