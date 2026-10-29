# Nitro-Core-DX YM2608 Audio Subsystem Specification

**Version 0.5 (Draft)**  
**Date:** March 19, 2026 (audio-direction reframe 2026-06-14)  
**Purpose:** Specify the YM2608/OPNA audio subsystem — the final audio
subsystem of Nitro-Core-DX — including its host MMIO interface and emulator
runtime.

> **File-rename note:** this file is still named `APU_FM_OPM_EXTENSION_SPEC.md`
> for link stability; a rename to a YM2608-named file is a later cleanup step.

> **Audio direction (2026-06-14):** Nitro-Core-DX has **one final audio
> subsystem: YM2608 / OPNA** (FM, SSG, rhythm, ADPCM). The older 4-channel
> "fantasy APU" is **temporary migration scaffolding only** — not final
> hardware — and will be removed. "OPM-lite," the `NCDX_YM_BACKEND` selector,
> and any non-YMFM path are **internal implementation/compatibility details**,
> not part of the console's audio identity.

> **Release scope:**
> - The audio subsystem is **YM2608/OPNA**.
> - Emulator/devkit frontends use the YMFM-backed runtime (`-audio-backend ymfm`).
> - Use `docs/planning/V1_CHARTER.md` and `docs/planning/V1_ACCEPTANCE.md` for
>   release-target scope and gates.

> **Implementation snapshot (honest status):**
> - YM2608 host MMIO interface at `0x9100-0x91FF` is implemented in the emulator.
> - The active path uses `0x9100/0x9101` for port 0, `0x9102` for host status,
>   `0x9103` for control, and `0x9104/0x9105` for port 1.
> - `FM_STATUS` timer/busy/IRQ flags and the timer IRQ bridge are implemented.
> - YM2608 runtime playback is **operational through YMFM-backed builds**;
>   hardware conformance is **still being refined and is not yet fully verified**.
> - The active emulator runtime uses fixed-point sample generation.
> - *(Internal:)* an in-tree OPM-lite model exists as a code-level path for
>   non-YMFM builds; it is an implementation detail, not a user-facing option.

---

## Goals

- Specify the **YM2608/OPNA** audio subsystem as the console's single, final audio hardware
- Define the YM2608 host MMIO interface (`0x9100-0x91FF`) and emulator runtime
- Provide a **software emulator implementation first** for validation and tooling
- Keep the design **FPGA-implementable** and suitable for future hardware integration
- Enable future CoreLX audio APIs (the planned `music.*` surface) that are easy to use while still exposing low-level YM2608 control

## Non-Goals (This Phase)

- Audio “sound design polish” features unrelated to hardware behavior
- Finalizing every YM2608 analog/output characteristic in the first pass
- Building the FPGA implementation immediately (this doc defines the path)
- Designing the CoreLX `music.*` API here (tracked separately)

(The legacy `0x9000-0x90FF` 4-channel path is **migration scaffolding** that
stays only until the YM2608 audio surface is complete; it is not a long-term
goal of this subsystem.)

---

## High-Level Architecture

### Target architecture

The final Nitro-Core-DX audio subsystem is **YM2608 / OPNA**:

- **YM2608/OPNA subsystem:** dual-port host interface (`0x9100-0x91FF`) + FM, SSG, rhythm, and ADPCM synthesis, played through the YMFM-backed runtime
- **Legacy 4-channel path (scaffolding):** the `0x9000-0x90FF` PSG/PCM registers remain only during migration so existing ROMs keep working; they are slated for removal

```
CPU writes audio regs
    │
    ├── YM2608 host regs (0x9100-0x91FF) ──► YM2608/OPNA subsystem (FM/SSG/rhythm/ADPCM)
    │                                          │
    └── Legacy regs (0x9000-0x90FF) ──────────┤   (temporary scaffolding)
        [migration scaffolding, to be removed] │
                                               └─► Audio backend (YMFM)
```

### Why the legacy path remains (temporarily)

The legacy `0x9000-0x90FF` path is kept *only as migration scaffolding* during
the transition to YM2608-only audio:

- existing ROMs and CoreLX code keep compiling/playing while the YM2608 surface is built
- it reduces bring-up risk during the migration
- it is **not** a permanent dual-architecture design — once the YM2608 audio
  API and demos are in place, the legacy path is removed

---

## Memory Map Proposal (Draft)

### Existing APU (Unchanged)

- `0x9000 - 0x90FF` : Existing Nitro-Core-DX APU registers

### YM2608 Host Registers

- `0x9100 - 0x91FF` : YM2608 host interface + status
- `0x9200 - 0x92FF` : Optional mirror/debug windows (phase 2, optional)

### FM Host Interface (Current Runtime Contract)

This wraps the YM2608/OPNA dual-port register bus in a CPU-friendly MMIO shell:

- `0x9100` `FM_ADDR` (write/read): Port 0 register address select
- `0x9101` `FM_DATA` (write/read): Port 0 data port
- `0x9102` `FM_STATUS` (read): Shared busy / timer flags / IRQ pending
- `0x9103` `FM_CONTROL` (write/read): Enable/mute/reset/debug options
- `0x9104` `FM_PORT1_ADDR` (`FM_MIX_L` legacy alias) (write/read): Port 1 register address select
- `0x9105` `FM_PORT1_DATA` (`FM_MIX_R` legacy alias) (write/read): Port 1 data port
- `0x9106` `FM_VOLUME` (write/read): Master output gain (0-255, default 255), applied after either FM synthesis path. Backs CoreLX's `music.set_volume`/`music.fade_to`.

Notes:
- The active YMFM backend treats `0x9104/0x9105` as YM2608 upper-port address/data, not just abstract mix-gain bytes.
- `FM_STATUS` is the current Nitro host-status register; a dedicated `status1` mirror is not presently exposed.
- Timer-oriented OPM-style host writes (`0x10/0x11/0x12/0x14`) are translated to the YM2608 timer register set in the active backend.

---

## Compatibility Model

### Target Compatibility

- **Behavioral compatibility:** YM2608/OPNA register behavior sufficient for ROM playback, music drivers, and chip-facing diagnostics
- **Software compatibility:** Preserve the existing host MMIO shell and OPM-style timer/control expectations where older tooling depends on them
- **FPGA compatibility:** Design maps cleanly to a hardware FM core plus a Nitro bus wrapper

### Explicit Compatibility Scope (Phase 1)

- Dual-port host write/read behavior
- Key on/off behavior
- Channel/operator parameter updates
- Timer behavior and status flags
- Deterministic sample generation and mixing integration

### Deferred / Phase 2 Compatibility

- Fine-grained analog output characteristics
- Bit-exact chip quirks if they complicate first implementation
- Full stereo mixing/panning UI polish in the emulator frontend

---

## Software Emulation First (Required)

The YM2608 audio subsystem must be emulated in software before FPGA implementation.

### Why

- Validate register map and ROM behavior now
- Build CoreLX support and developer tooling against a stable target
- Reduce FPGA bring-up risk by proving the programming model early

### Emulator Implementation Plan

Implemented / planned components:

- `internal/apu/fm_opm.go`
- `internal/apu/fm_opm_test.go`

Integrate with:

- `internal/apu/apu.go` (mixer + MMIO routing)
- `internal/emulator/emulator.go` (FM timer IRQ bridge to CPU interrupt source)

Note:
- No additional `internal/memory/bus.go` range routing was required because the bus already forwards `0x9000-0x9FFF` to the APU. The YM2608 host interface lives inside the APU offset space (`0x0100-0x01FF` => CPU `0x9100-0x91FF`).
- In the current emulator runtime, fixed-point generation is the authoritative audio path. Legacy `GenerateSample()` behavior and float phase state are retained only as migration-scaffolding compatibility.

### Implementation Model Requirements

- Deterministic state machine style (hardware-friendly)
- Fixed-size state, no dynamic behavior in the synthesis core hot path
- Fixed-point arithmetic preferred where practical
- No host-timing-dependent behavior (sample generation driven by emulator timing)

### Acceptable Phase 1 Implementation Options

1. **Wrapper/adaptor around a proven YM2151 software core**
2. **Native Nitro implementation that mirrors OPM behavior**

Either is acceptable as long as:
- MMIO contract is stable
- Tests prove deterministic behavior
- Future FPGA mapping remains straightforward

---

## Mixer Integration (APU v2)

### Phase 1 (Minimal, Practical)

- Mix existing APU output and FM output into current audio stream
- Preserve current output path behavior when FM is disabled
- Add FM mute/enable control for testing

### Phase 2 (Recommended)

- Stereo output support in emulator backend
- Per-source mix levels (Legacy APU vs FM)
- Optional debug meter/visualization in UI

### Hardware Constraint

Mixer math must remain FPGA-realizable:
- Fixed-point accumulation
- Saturating clamp
- Explicit channel gain paths

---

## Timing and Interrupts (Draft Contract)

### Sample Generation

- FM samples must be generated using the same audio scheduling model as the existing APU path
- FM register writes become visible in deterministic order relative to CPU execution and sample generation

### Timers / IRQ

If OPM-compatible timers are implemented:

- Timer state updates must be deterministic
- Status flags exposed via `FM_STATUS`
- IRQ signaling must have explicit mapping into Nitro interrupt system (proposal below)

#### IRQ Mapping Proposal (Draft)

- FM timer IRQ routed to a new interrupt source in the emulator core (or multiplexed via existing IRQ until expanded)
- CPU-visible status still available through `FM_STATUS`

This may require expanding interrupt source definitions later; do not block phase 1 software FM bring-up on final IRQ wiring.

---

## CoreLX Integration Strategy (Developer-Friendly)

### Layered API Design

CoreLX should expose two levels:

#### 1. Easy API (default)

- `audio.play_note(channel, note, duration)`
- `audio.set_instrument(channel, preset)`
- `audio.stop(channel)`

These can choose the best backend (legacy APU or FM) based on target/profile.

#### 2. Advanced FM API

- `fm.write_reg(addr, value)`
- `fm.key_on(channel, operators)`
- `fm.key_off(channel)`
- `fm.load_patch(channel, patch)`
- `fm.set_patch_param(...)`

### Compiler/Runtime Implication

This fits the target-profile approach already described in:

- `docs/specifications/CORELX_NITRO_CORE_8_COMPILER_DESIGN.md`

The YM2608 audio subsystem can be target-gated:
- `dx` target uses the YM2608 audio subsystem (the legacy APU path is temporary migration scaffolding)
- `portable` target may limit to high-level audio API only

---

## Validation Plan (Before FPGA)

### Unit Tests

- MMIO register write/read behavior
- Timer/status behavior
- Deterministic state transitions (same writes => same internal state)
- Mixer output non-regression when FM disabled

### Integration Tests

- FM note key-on/key-off test ROM
- Patch load + note playback test ROM
- Timer interrupt/status ROM
- Mixed playback (legacy APU + FM simultaneously)

### Determinism Tests

Same ROM + same input sequence must yield:

- Same FM register state
- Same timer/status flags
- Same generated sample stream (or same fixed-point buffer output)

### Manual Dev-Kit Tests

- CoreLX sample program plays FM patch
- Debug UI can show FM register/status state
- Audio backend output remains stable under normal emulator frame pacing

---

## FPGA Mapping Plan (AURA/OPM-Oriented)

### Design Intent

The software YM2608 model should reproduce the externally visible behavior expected from the YM2608/OPNA hardware block.

### Future FPGA Partitioning

- `FM Core` (OPM-compatible engine)
- `MMIO Wrapper` (Nitro bus-facing register adapter)
- `Timer/IRQ bridge`
- `Digital audio output / mixer path`

### Board Integration Benefits

Using an FPGA OPM-compatible approach (AURA-style direction) supports:

- Easier integration with modern digital audio paths
- Cleaner system integration on a custom FPGA console board
- Reduced dependency on legacy analog support circuitry

---

## Development Phases

### Phase 1: Emulator YM2608 Skeleton ✅ (Implemented)

- ✅ MMIO register range reserved and reachable at `0x9100-0x91FF`
- ✅ FM host interface registers implemented
- ✅ Status/busy/timer smoke tests
- ✅ Mixer integration hooks

### Phase 2: YM2608 Runtime Baseline 🚧 (Operational, Conformance Ongoing)

- ✅ Timer/status behavior (deterministic placeholder timing model)
- ✅ FM timer IRQ bridge to CPU interrupt system
- ✅ YM2608 runtime operational through the YMFM-backed runtime
- ✅ Test ROM coverage for FM MMIO/timer and audible demos
- ❌ Full conformance parity across expanded YM2608 reference vectors
- ❌ Final envelope/timbre model and polish

### Phase 3: CoreLX Support

- High-level audio helpers
- FM patch abstractions
- Sample projects/examples

### Phase 4: FPGA Validation Path

- Match emulator-visible behavior to FPGA core wrapper
- Hardware bring-up test ROM suite reuse

---

## Risks and Mitigations

### Risk: FM programming complexity hurts usability

Mitigation:
- CoreLX high-level wrappers and presets
- Keep legacy APU path for simple games and testing

### Risk: Compatibility ambiguities slow implementation

Mitigation:
- Define Nitro FM MMIO wrapper contract first
- Lock deterministic test vectors early

### Risk: FPGA resources become tight

Mitigation:
- Treat FM as modular/optional block
- Keep the legacy APU path operational as temporary migration scaffolding

---

## Immediate Next Steps (Recommended)

1. Update the main hardware specification docs to include FM host MMIO (`0x9100+`) and current status (this is now underway)
2. Tighten OPM compatibility in the software FM path (timers, register semantics, operator behavior)
3. Improve FM voice envelopes/timbre (reduce note-edge artifacts while preserving deterministic behavior)
4. Add CoreLX API placeholders / abstractions (`fm.*`, `audio.*` high-level FM-backed helpers)
5. Build stronger determinism test vectors (same writes/input => same status/sample output)
