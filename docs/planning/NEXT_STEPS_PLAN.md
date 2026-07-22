# Nitro-Core-DX — Current Next Steps Plan

Created: 2026-04-20
Last aligned: 2026-07-22
Status: Active product/tooling milestone plan

This document tracks the current non-CoreLX product milestones. CoreLX language
and manual details are intentionally deferred to a separate CoreLX deep-dive
documentation pass. This file should stay focused on release sequencing, Dev Kit
readiness, audio tooling, hardware validation, and documentation alignment.

Authoritative scope references:

- `V1_CHARTER.md` — product V1 scope
- `V1_ACCEPTANCE.md` — release-blocking gates
- `V1_RISKS.md` — active risk register
- `../README.md` — public status snapshot

---

## Current Snapshot (2026-07-22)

### Done / Real Enough To Build On

- ROM-first NitroPackInDemo showcase is complete enough to serve as the visual
  proof of concept.
- Emulator hardware foundation is software-ready: CPU, memory, PPU, input,
  Matrix Mode, DMA/HDMA, native larger hardware sprites, and the YM2608 runtime
  path are operational.
- Dev Kit has Build/Run, embedded emulator, diagnostics/output/manifest panes,
  autosave, settings persistence, view modes, Sprite Lab, and Tilemap Lab.
- Sprite Lab has the strongest tool maturity: edit/import/export, palette
  handling, undo/redo, project insertion, manifest flow, and focused tests.
- Tilemap Lab is usable for map editing and source insertion, with tests around
  packing, snippets, import/export, and upsert behavior.
- `.ncdxmusic` assets, YM stream decode/encode, CoreLX `music.*` playback, and
  the bus-side YM burst streamer are implemented and emulator-tested.

### Active WIP / Release Blockers

- NitroPackInDemo CoreLX rebuild is the active M8 acceptance target and is
  currently exposing a large-program/codegen/banking stress case.
- Dev Kit-generated templates/snippets need to stay aligned with this week's
  language changes.
- Tilemap Lab needs production workflow hardening: manifest/source parity,
  generated-source compile coverage, and visible emulator preview evidence.
- Sound Studio is still a placeholder. Runtime support exists; authoring UI does
  not.
- YM2608 conformance/polish remains open: runtime plumbing works, but V1 still
  needs reference-quality evidence and broader edge-behavior coverage.
- Editor essentials remain open: find/replace, go-to-line, symbol navigation,
  current namespace/builtin intelligence, and squiggle polish.
- Debugger UX remains open: pause/resume, frame step, CPU step, register/PC
  panels, and memory watch workflow.

---

## Phase 0 — Stabilize The Current Baseline

These items protect every other milestone.

1. Keep ROM-first NitroPackInDemo runnable as the stable showcase while the
   CoreLX rebuild is under compiler stress.
2. Resolve the large-program branch-offset/codegen failure surfaced by the
   CoreLX overworld rebuild.
3. Keep focused regression commands green for Dev Kit, YM2608, music assets,
   Sprite Lab, Tilemap Lab, and core emulator packages.
4. Compile every Dev Kit template and every tool-generated snippet in tests.
5. Keep the documentation map current when a doc is active, stale, or archived.

---

## Phase 1 — Dev Kit Art Suite Readiness

Goal: make visual tooling dependable enough for the pack-in demo and V1 docs.

### Sprite Lab

Status: mostly implemented.

Remaining:

- Ensure generated source follows current CoreLX syntax.
- Ensure palette initialization snippets compile legally in generated projects.
- Treat the new hardware sprite sizes as first-class targets: 32×16, 32×32,
  64×32, 64×64, 128×64, and 128×128 should not require manual composite OAM
  authoring.
- Keep manifest and inline insertion paths covered by compile tests.
- Add at least one emulator-visible sprite asset round-trip acceptance test.

### Tilemap Lab

Status: usable but not release-complete.

Remaining:

- Decide and document the max supported map size in tool UI, README, and tests.
- Load/recognize manifest-backed assets, not only source-editor inline assets.
- Add source/manifest round-trip tests that compile and render a visible tilemap.
- Make generated snippets and template interactions current with language rules.
- Tighten error messages for missing tilesets, stale asset names, and invalid
  map dimensions.

### Image / Plane Import

Status: CLI exists; Dev Kit UI missing.

Remaining:

- Add Dev Kit import flow for PNG -> `.cxasset`.
- Insert/package `asset Name: image "file.cxasset"` declarations.
- Generate a valid `matrix_plane.load_bitmap(Name, channel)` starter snippet.
- Show import stats: source dimensions, plane size, palette bank, ROM footprint.

---

## Phase 2 — Sound Studio MVP

Goal: ship the smallest useful audio studio that uses the real YM2608 runtime.

### MVP Scope

- Import `.vgm` / `.vgz`.
- Convert to `.ncdxmusic` through `internal/ymstream`.
- Open existing `.ncdxmusic`.
- Show duration, frame count, write count, frame-sample rate, compact size, and
  estimated ROM footprint.
- Preview playback through the embedded emulator/YM2608 audio path.
- Export/add the music file to the project.
- Insert or update `asset Theme: music "theme.ncdxmusic"`.
- Offer snippets for `music.play`, `music.play_loop`, `music.play_jingle`,
  `music.stop`, `music.set_volume`, and `music.fade_to`.

### Explicitly Not MVP

- Full tracker composition.
- Piano-roll editing.
- Instrument bank editing as a complete product.
- ADPCM sample authoring.
- Automatic arrangement tools.

Those can follow once the import/preview/export loop is boringly reliable.

### Tests Required

- VGM/VGZ import success and unsupported-command diagnostics.
- `.ncdxmusic` open/inspect round-trip.
- Generated source compiles.
- Build pipeline packages the exported music asset.
- Preview ROM produces expected YM writes and audible samples through the same
  path used by Build+Run.

---

## Phase 3 — YM2608 Conformance And Audio Release Gate

Goal: separate "audio runtime works" from "audio is release-quality."

Runtime baseline exists:

- YM2608 MMIO host interface
- dual-port register writes
- timer/status/IRQ bridge behavior
- `.ncdxmusic` stream playback
- bus-side burst streamer
- Dev Kit SDL audio queue

Remaining V1 work:

- Freeze behavioral parity profile.
- Curate reference material and thresholds.
- Expand SSG/rhythm/ADPCM edge-behavior coverage.
- Keep `ci-audio` and YM2608 policy/manifest checks green.
- Ensure docs/UI consistently present YM2608/OPNA as the final audio identity
  and the legacy 4-channel synth as temporary migration scaffolding.

---

## Phase 4 — Editor And Debugger Readiness

Goal: make the Dev Kit feel like an IDE rather than just a launcher.

Editor:

- Find/replace.
- Go-to-line.
- Basic function/symbol navigation.
- Current syntax/builtin highlighting.
- Diagnostics squiggle and panel-location sync.

Debugger:

- Pause/resume.
- Frame step.
- CPU instruction step.
- Register/PC/state panel.
- Memory watch panel.
- Deterministic tests for step modes.

---

## Phase 5 — Documentation Alignment

Goal: keep active docs true and let historical docs stay historical.

Current rules:

- Do not use archived files as current project status.
- Prefer `README.md`, `docs/README.md`, `V1_CHARTER.md`,
  `V1_ACCEPTANCE.md`, and `V1_RISKS.md` for milestone truth.
- Prefer `COMPLETE_HARDWARE_SPECIFICATION_V2.1.md` and
  `APU_FM_OPM_EXTENSION_SPEC.md` for hardware/audio behavior.
- CoreLX manuals/specs are reserved for a dedicated CoreLX documentation pass.

Remaining:

- Keep `SYSTEM_MANUAL.md` clearly marked as system context under revision.
- Keep `HARDWARE_FEATURES_STATUS.md` aligned with hardware reality.
- Keep old point-in-time readiness assessments clearly marked as historical.
- Add release evidence links once V1 gates start producing RC artifacts.

---

## Phase 6 — V1 Release Candidate Gate

Before V1 RC:

- [ ] NitroPackInDemo runs through the target start-to-finish flow.
- [ ] CoreLX rebuild is accepted by the dedicated CoreLX gate.
- [ ] Sprite, Tilemap, Image Import, and Sound Studio MVP workflows compile,
      package, and preview without manual asset cleanup.
- [ ] YM2608 conformance/reference evidence is produced and reviewed.
- [ ] Editor and debugger gates pass.
- [ ] Linux and Windows packaged builds pass smoke validation.
- [ ] Known issues are documented.
- [ ] Active docs agree on done/WIP/deferred status.

---

## Post-V1 Parking Lot

- Full Sound Studio tracker/composer.
- Advanced FM instrument editor.
- Vertical sprites for Matrix Mode.
- Large-world streaming/tilemap workflows.
- Additional first-party templates and sample games.
- FPGA bring-up.
- Physical console shell/controller work.
