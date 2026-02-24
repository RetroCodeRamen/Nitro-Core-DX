# Specifications (Current vs Historical)

This directory contains hardware specifications, pin definitions, FPGA documentation, and extension specs.

## Current / Preferred Specs

- `COMPLETE_HARDWARE_SPECIFICATION_V2.1.md`
  - Current evidence-based base hardware spec (CPU/PPU/APU/input/timing/registers)
- `APU_FM_OPM_EXTENSION_SPEC.md`
  - Current FM extension design + implementation status (legacy APU + FM host interface)
- `CARTRIDGE_PIN_SPECIFICATION.md`
  - Cartridge connector pinout/spec
- `CONTROLLER_PIN_SPECIFICATION.md`
  - Controller connector pinout/spec

## Active Supporting Specs / FPGA Docs

- `FPGA_IMPLEMENTATION_SPECIFICATION.md`
- `FPGA_ARCHITECTURE_RECOMMENDATION.md`
- `FPGA_READINESS_ASSESSMENT.md`
- `FPGA_READINESS_COMPARISON.md`
- `SPEC_AUDIT_DISCREPANCIES.md`

## Historical / Superseded Specs (Keep for Context)

- `HARDWARE_SPECIFICATION.md` (older v1.0; contains stale APU/FM details)
- `COMPLETE_HARDWARE_SPECIFICATION.md` (older v2.0; superseded by v2.1)

## Notes

- If hardware/audio details conflict, prefer:
  1. `COMPLETE_HARDWARE_SPECIFICATION_V2.1.md`
  2. `APU_FM_OPM_EXTENSION_SPEC.md` (for FM-specific behavior/status)
  3. current source code/tests
