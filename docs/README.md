# Documentation Map (Current + Historical)

This directory contains the project's active documentation, historical reviews, and archived planning notes.

This file is the primary navigation entry for docs maintenance.

**Last alignment pass:** 2026-07-22. This pass updated product, tooling,
audio, milestone, and documentation-routing status. CoreLX-specific manuals and
specs are intentionally waiting for a separate deep dive.

## Current Sources of Truth (Read These First)

- `../README.md`
  - Project overview, current status snapshot, quick start
- `planning/NEXT_STEPS_PLAN.md`
  - Current product/tooling milestone sequence and release-blocking work
- `planning/V1_CHARTER.md`, `planning/V1_ACCEPTANCE.md`, `planning/V1_RISKS.md`
  - Product V1 scope, release gates, and live risk register
- `CORELX_V1_IMPLEMENTATION_STATUS.md`
  - CoreLX-focused implementation handoff. Scheduled for a separate CoreLX
    deep-dive alignment pass; verify live details against compiler tests/code
    until that pass is complete.
- `specifications/CORELX_SYNTAX_V1.md`
  - CoreLX v1 language syntax charter. Also scheduled for the separate CoreLX
    documentation review before it should be treated as final prose.
- `specifications/CORELX_CARTRIDGE_FORMAT.md`
  - CoreLX single-file cartridge format (draft)
- `../Games/NitroPackInDemo/CORELX_EXTRACTION.md`
  - CoreLX design decision record (M7) and M8 build order
- `specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md`
  - Current evidence-based hardware specification (base hardware)
- `specifications/APU_FM_OPM_EXTENSION_SPEC.md`
  - YM2608 audio subsystem design + implementation status
- `CORELX.md`
  - Older CoreLX reference. Treat as stale until the CoreLX documentation deep
    dive is complete.
- `testing/README.md`
  - Current test command entrypoints and testing docs map

### The two end-user books (v1, in progress)

The project ships two distinct manuals for two audiences:

- `NITRO_CORE_DX_OWNERS_MANUAL.md` — **Console Owner's Manual** (player-facing):
  what the console is, the controller, running games. Clean Retro Code Ramen
  product voice.
- `../PROGRAMMING_MANUAL.md` — **Programming Guide** (programmer-facing): the
  full DevKit + CoreLX teaching, taught by Fletcher. Every demo program in it is
  compiled and run against the emulator by the test suite
  (`internal/corelx/manual_examples_test.go`, sources in `manual_examples/`).
  Style governed by `CORELX_MANUAL_STYLE_GUIDE.md`. As of 2026-07-21 this
  merges what used to be two separate, drifting docs (the old
  `PROGRAMMING_MANUAL.md` and `CORELX_PROGRAMMING_GUIDE.md`) into one current
  manual — `CORELX_PROGRAMMING_GUIDE.md` no longer exists as a separate file.

## Deferred Until CoreLX v1 Ships

- `guides/PROGRAMMING_GUIDE.md`
  - An older, pre-v1 project-based walkthrough (a Pong-style mini-game
    tutorial). Documents the current pre-v1 compiler and will be rewritten (or
    folded into `../PROGRAMMING_MANUAL.md`'s Part 2) against the finished v1
    language. Until then it carries scope notes and remains usable for the
    shipping compiler.
- `../SYSTEM_MANUAL.md`
  - Under revision; verify claims against current hardware specs/tests.

## Documentation Organization

- `specifications/`
  - Language specs (CoreLX v1), hardware specs, pinouts, FPGA docs, YM2608 audio subsystem spec
- `planning/`
  - Active product planning: `V1_CHARTER.md` (product V1 scope — distinct from
    the CoreLX *language* v1 charter), `V1_ACCEPTANCE.md`, `V1_RISKS.md`,
    `NEXT_STEPS_PLAN.md`, `FUTURE_FEATURES_PARKING_LOT.md`
- `testing/`
  - Test procedures and current workflows
- `guides/`
  - Setup/procedural guides (build, releases, debugging, EOD procedure)
- `archive/`
  - Superseded plans, historical reviews/audits, incident postmortems —
    retained for history, never current status

## Current Milestone Snapshot (2026-07-22)

- **Hardware/emulator:** software-ready; YM2608 runtime path is operational,
  with conformance/timbre/subsystem parity still under refinement.
- **CoreLX/toolchain:** active M8 stabilization. The NitroPackInDemo CoreLX
  rebuild is the main acceptance target and currently exposes a large-program
  codegen/banking stress issue.
- **Dev Kit:** Build/Run, embedded emulator, diagnostics panel, Sprite Lab,
  Tilemap Lab, autosave, settings, and view modes are present. Find/replace,
  deeper debugger UX, editor intelligence polish, and Sound Studio remain open.
- **Art suite:** Sprite Lab is the strongest and most complete; Tilemap Lab is
  usable but needs production round-trip and manifest/source parity hardening.
- **Audio suite:** `.ncdxmusic` runtime playback exists; Sound Studio is not
  implemented beyond the placeholder tab.
- **Docs:** product/tooling docs are aligned in this pass; CoreLX docs are
  intentionally deferred.

## Documentation Status Conventions

- `Current`: intended as source-of-truth / active reference
- `Under Revision`: useful but may contain stale assumptions; verify against current specs/tests
- `Historical Snapshot`: retained for context/history; do not use as current project status
- `Archive`: superseded content moved out of the active docs path

## Cleanup Notes (2026-06-12)

- Documentation pass aligned everything with the CoreLX v1 design decisions:
  resolved-issue postmortems and stale meta-cleanup docs were deleted (history
  lives in git); historical audits, the NitroLang design doc, the CoreLX data
  model plan, and completed planning checklists moved to `archive/`.
- Two "V1"s exist by design: the **product** V1 (`planning/V1_CHARTER.md` —
  SDK/emulator release scope) and the **CoreLX language** v1
  (`specifications/CORELX_SYNTAX_V1.md` — language freeze). Cross-references
  in both files distinguish them.
