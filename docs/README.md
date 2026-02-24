# Documentation Map (Current + Historical)

This directory contains the project's active documentation, historical reviews, and archived planning notes.

This file is the primary navigation entry for docs cleanup and ongoing maintenance.

## Current Sources of Truth (Read These First)

- `../README.md`
  - Project overview, current status snapshot, quick start
- `../PROGRAMMING_MANUAL.md`
  - High-level developer-facing programming manual (currently under revision; use with `docs/CORELX.md`)
- `../SYSTEM_MANUAL.md`
  - System architecture manual (currently under revision; verify against current hardware specs/tests)
- `docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md`
  - Current evidence-based hardware specification (base hardware)
- `docs/specifications/APU_FM_OPM_EXTENSION_SPEC.md`
  - Current FM extension design + implementation status (legacy APU + FM path)
- `docs/CORELX.md`
  - Current CoreLX language reference (implementation-aware)
- `docs/CORELX_DATA_MODEL_PLAN.md`
  - Current CoreLX/Dev Kit Phase 1 planning baseline (asset model, packaging, diagnostics)
- `docs/testing/README.md`
  - Current test command entrypoints and testing docs map

## Documentation Organization

- `docs/specifications/`
  - Hardware specs, pinouts, FPGA docs, FM extension spec
- `docs/planning/`
  - Current planning/roadmap docs (some files are historical planning snapshots)
- `docs/testing/`
  - Test procedures, testing plans, historical test results/summaries
- `docs/guides/`
  - Setup/procedural guides (build, SDL_ttf, EOD procedure, etc.)
- `docs/issues/`
  - Incident reports, bug investigations, fix writeups (historical troubleshooting records)
- `docs/archive/`
  - Archived/superseded docs intentionally retained for history/reference

## Documentation Status Conventions (Use Going Forward)

- `Current`: intended as source-of-truth / active reference
- `Under Revision`: useful but may contain stale assumptions; verify against current specs/tests
- `Historical Snapshot`: retained for context/history; do not use as current project status
- `Archive`: superseded content moved out of the active docs path

## Cleanup Notes (2026-02-24)

- Multiple older hardware specs/manuals contain stale APU/audio/FM details.
- Historical notes have been added to older specs to steer readers to the current sources.
- The project is pre-alpha; docs should prioritize correctness and clarity over backward-compatibility promises.
