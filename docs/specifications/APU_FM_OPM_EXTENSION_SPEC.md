# Nitro-Core-DX APU FM (OPM/YM2151-Compatible) Extension Specification

**Version 0.2 (Draft, Updated to Current Emulator State)**  
**Date:** February 24, 2026  
**Purpose:** Add a hardware-implementable FM synthesis extension (YM2151/OPM-compatible model) without breaking the current APU

> **Design Direction:** Keep the existing APU as the default developer-friendly audio path, and add an optional FM extension block for richer audio and future FPGA implementation.

> **Implementation Snapshot (2026-02-24):**
> - FM host MMIO interface at `0x9100-0x91FF` is implemented in the emulator/APU.
> - `FM_STATUS` timer/busy/IRQ flags and FM timer IRQ bridge are implemented (deterministic placeholder timing model).
> - An audible OPM-lite software synthesis path is implemented (8 voices, simplified 2-operator behavior), mixed with the legacy APU.
> - This is not yet full YM2151/OPM accuracy.

---

## Goals

- Add a **YM2151/OPM-compatible FM synthesis extension** as an APU subsystem
- Preserve existing `0x9000+` APU behavior and ROM compatibility
- Provide a **software emulator implementation first** for validation and tooling
- Keep the design **FPGA-implementable** and suitable for future hardware integration
- Enable future CoreLX APIs that are easy to use while still exposing low-level FM control

## Non-Goals (This Phase)

- Replacing the current APU/PSG+PCM path
- Audio “sound design polish” features unrelated to hardware behavior
- Finalizing every YM2151 analog/output characteristic in the first pass
- Building the FPGA implementation immediately (this doc defines the path)

---

## High-Level Architecture

### APU v2 (Recommended)

The Nitro-Core-DX audio subsystem evolves into a mixed architecture:

- **APU Core (existing):** Simple PSG/PCM-style channels (developer-friendly)
- **FM Extension (new):** OPM/YM2151-compatible register interface + FM synthesis engine
- **Mixer (new/expanded):** Combines legacy APU output + FM output into a unified stream

```
CPU writes audio regs
    │
    ├── Legacy APU regs (0x9000-0x90FF) ──► Existing APU (PSG/PCM)
    │
    └── FM OPM regs   (new range)       ──► FM Extension (YM2151-compatible model)
                                             │
                             Legacy APU out ─┼─► Unified Mixer ─► Audio backend
                                   FM out ───┘
```

### Why Extension Instead of Replacement

- Preserves existing ROMs and test ROMs
- Maintains easy audio programming for simple projects
- Reduces bring-up/debug risk while FM block matures
- Fits the long-term “development kit” goal (simple + advanced paths)

---

## Memory Map Proposal (Draft)

### Existing APU (Unchanged)

- `0x9000 - 0x90FF` : Existing Nitro-Core-DX APU registers

### New FM Extension (Proposed)

- `0x9100 - 0x91FF` : FM extension host interface + status
- `0x9200 - 0x92FF` : Optional mirror/debug windows (phase 2, optional)

### FM Host Interface (Draft)

This wraps the OPM/YM2151-style register bus in a CPU-friendly MMIO shell:

- `0x9100` `FM_ADDR` (write): OPM register address select
- `0x9101` `FM_DATA` (write/read): OPM register data port
- `0x9102` `FM_STATUS` (read): Busy / timer flags / IRQ pending
- `0x9103` `FM_CONTROL` (write): Enable/mute/reset/debug options
- `0x9104` `FM_MIX_L` (write): FM contribution to left mix (phase 2 if stereo)
- `0x9105` `FM_MIX_R` (write): FM contribution to right mix (phase 2 if stereo)

Notes:
- OPM compatibility should be implemented primarily through the `FM_ADDR/FM_DATA` register pair.
- `FM_STATUS` should expose timer/IRQ behavior in a Nitro-friendly way while remaining OPM-accurate enough for software compatibility.

---

## Compatibility Model

### Target Compatibility

- **Behavioral compatibility:** YM2151/OPM register behavior sufficient for music drivers and patch programming
- **Software compatibility:** Register-level workflows modeled after YM2151 (timers, key on/off, operator/channel params)
- **FPGA compatibility:** Design maps cleanly to a hardware FM core (AURA/OPM-style)

### Explicit Compatibility Scope (Phase 1)

- Register write/read behavior
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

The FM extension must be emulated in software before FPGA implementation.

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
- No additional `internal/memory/bus.go` range routing was required because the bus already forwards `0x9000-0x9FFF` to the APU. The FM extension lives inside the APU offset space (`0x0100-0x01FF` => CPU `0x9100-0x91FF`).

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

The FM extension can be target-gated:
- `dx` target supports legacy APU + FM extension
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

The software FM extension should model the same externally visible behavior expected from a hardware OPM-compatible block.

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

### Phase 1: Emulator FM Extension Skeleton ✅ (Implemented)

- ✅ MMIO register range reserved and reachable at `0x9100-0x91FF`
- ✅ FM host interface registers implemented
- ✅ Status/busy/timer smoke tests
- ✅ Mixer integration hooks

### Phase 2: Software OPM Behavior 🚧 (In Progress)

- ✅ Timer/status behavior (deterministic placeholder timing model)
- ✅ FM timer IRQ bridge to CPU interrupt system
- ✅ Audible OPM-lite subset (software-first, hardware-oriented)
- ✅ Test ROM coverage for FM MMIO/timer and audible demos
- ❌ Full YM2151 register/operator behavior accuracy
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
- Keep legacy APU operational as fallback path

---

## Immediate Next Steps (Recommended)

1. Update the main hardware specification docs to include FM host MMIO (`0x9100+`) and current status (this is now underway)
2. Tighten OPM compatibility in the software FM path (timers, register semantics, operator behavior)
3. Improve FM voice envelopes/timbre (reduce note-edge artifacts while preserving deterministic behavior)
4. Add CoreLX API placeholders / abstractions (`fm.*`, `audio.*` high-level FM-backed helpers)
5. Build stronger determinism test vectors (same writes/input => same status/sample output)
