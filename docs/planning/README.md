# Planning Documents

This directory contains active planning, roadmap, and risk documents for the
**product** V1 (emulator + SDK + tools release).

## Current Planning Inputs

- `V1_CHARTER.md`
  - Canonical product V1 scope contract and release-blocking IDs
  - (The CoreLX *language* v1 charter is separate:
    `docs/specifications/CORELX_SYNTAX_V1.md`)
- `V1_ACCEPTANCE.md`
  - Release acceptance gates and required evidence
- `V1_RISKS.md`
  - Active V1 risk register (owner + mitigation tracking)
- `NEXT_STEPS_PLAN.md`
  - Sequenced workstreams from the April 2026 project review (see its status
    note — CoreLX phases are governed by the M7/M8 decision record)
- `FUTURE_FEATURES_PARKING_LOT.md`
  - Deferred ideas / future expansion concepts (keyboard bus, YM2608 follow-ups, etc.)

## Usage Guidance

- For "what should we build next?" decisions, prefer:
  1. current test results + recent worklog
  2. `Games/NitroPackInDemo/CORELX_EXTRACTION.md` §12 (for CoreLX/M8 work)
  3. targeted planning docs in this folder
- Completed or superseded plans live in `docs/archive/plans/`
  (master plan, dev-tools plan, APU stabilization checklist, CoreLX data
  model plan).
