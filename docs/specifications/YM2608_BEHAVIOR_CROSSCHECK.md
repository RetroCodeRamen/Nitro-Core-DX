# YM2608 Behavior Cross-Check (Manual + Die-Shot LLE Reference)

## Purpose
Track Nitro-Core-DX YM2608 implementation status against:
- Primary source: Yamaha YM2608 English manual (`/home/aj/Downloads/YM2608J Translated.pdf`)
- Secondary reference: low-level die-shot-informed implementation notes (`Resources/fmopna_impl.h`)

Note: `Resources/fmopna_impl.h` is GPL-licensed reference material. It is used only for behavioral cross-checking guidance. No implementation code is copied.

## Current Runtime State
- FM runtime backend is selectable via `NCDX_YM_BACKEND`:
  - `auto` (default): YMFM OPNA backend when available, otherwise in-tree legacy backend
  - `ymfm`: force YMFM
  - `legacy`: force in-tree backend
- Emulator/Dev Kit frontends expose `-audio-backend auto|ymfm|legacy`.
- YM2608 host MMIO path is active at `0x9100-0x91FF` with register/timer/status plumbing implemented.
- Song replay via YM write streams is operational in ROM diagnostics and in gameplay loops.
- Legacy 4-channel APU behavior remains available as a fallback/runtime compatibility path.

## Cross-Check Matrix

### 1) Bus/Port Interface
Manual expectation:
- Distinct behavior by A1/A0 addressing/data/status reads
- Separate status read paths (status 0 / status 1)

Current:
- Implemented host ports at `0x9100-0x9107` with addr/data for port0 and port1
- Separate status0/status1 implemented

Gap:
- Bus wait-state timing currently deterministic placeholder, not full timing table fidelity

### 2) Timer Model
Manual expectation:
- Timer A: `tA = 72 * (1024 - NA) / φM`
- Timer B: `tB = 1152 * (256 - NB) / φM`
- `$27` load/enable/reset semantics
- IRQ behavior gated by enable bits

Current:
- Implemented with host-cycle -> YM-clock conversion and deterministic counters
- `$24/$25/$26/$27/$29` behavior covered by unit/integration tests

Gap:
- Exact edge/phase interactions under unusual write ordering still need exhaustive conformance vectors

### 3) FM Register Decode
Manual expectation:
- Channel/operator parameter latching across register groups (`$30-$B6`)
- Key on/off via `$28`

Current:
- Implemented register decode/latch structure
- Unit tests cover representative writes for:
  - ALG/FB
  - L/R + AMS/PMS
  - F-Number + Block
  - key on/off

Gap:
- Register-edge behavior under unusual write ordering still needs conformance vectors

### 4) FM Signal Path
Manual expectation:
- 6 channels, 4 operators, envelope and phase modulation behavior

Current:
- Operational runtime now includes a working YM2608 playback backend path (YMFM when available).
- Deterministic scaffolding and conformance fixtures from earlier stages remain in-tree as regression infrastructure.
- Practical integration checks now include ROM-driven playback and in-game background music replay.

Gap:
- Final chip-conformance envelope/phase/timbre parity across broader references
- Continued refinement of operator behavior and mixing parity vs external reference renders

### 5) SSG / ADPCM / Rhythm
Manual expectation:
- SSG core behavior + readable registers
- ADPCM and rhythm control/data flow

Current:
- Not implemented yet (intentional)

Gap:
- Full subsystem implementations and status coupling

## Stage 3 (Execution Status)

1. FM slot/channel mapping conformance pass
- Add exhaustive decode tests for `$30-$B6` slot/channel mapping across both ports
- Lock mapping before any synthesis behavior
Status: Complete

2. FM envelope/timing state-machine skeleton
- Implement deterministic operator envelope states (no final waveform output yet)
- Add step-based state transition tests (attack/decay/sustain/release control path)
Status: Complete

3. FM phase accumulator core
- Implement per-operator phase increment path with deterministic stepping
- Add numeric-state tests (not audio-golden yet)
Status: Complete

4. Controlled audible bring-up
- Enable one minimal algorithm path only after above tests are green
- Add waveform-shape sanity tests and clipping/mix safety tests
Status: Operational baseline established; conformance refinement ongoing

## Risk Controls
- Do not introduce SSG/ADPCM/rhythm behavior until FM envelope/phase core is stable.
- Keep each subsystem in separate modules/interfaces to maintain FPGA portability.
- Document every approximation in `YM2608_IMPLEMENTATION_NOTES.md` as it is introduced.
- Keep extraction-manifest drift reviewable (diff summary) and provenance pin policy enforceable (optional strict commit-pin mode) before broadening audio-synthesis scope.
- Keep per-scenario provenance-pin intent explicit in fixture policy and test-gated before promoting stricter CI enforcement.
- Use consolidated policy validation (`make check-ym2608-policy`) as the governance baseline before entering new chip-behavior slices. In the current tree this target is guarded and may skip optional manifest-tool checks when the helper files are absent locally.
