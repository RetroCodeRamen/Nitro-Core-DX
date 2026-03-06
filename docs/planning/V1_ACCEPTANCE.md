# Nitro-Core-DX V1 Acceptance Criteria

Status: Active  
Last Updated: March 6, 2026

This document defines release-blocking acceptance gates tied to `V1_CHARTER.md`.

## 1. Global Release Gates

### ACC-REL-1 Test Gate
- Full project tests pass with release tags used in CI.
- Determinism tests pass for emulator and selected ROM scenarios.

### ACC-REL-2 Performance Gate
- Representative scenes maintain target frame pacing for release profile.
- No critical frame-time regressions versus pre-RC baseline.

### ACC-REL-3 Stability Gate
- Crash-free smoke validation on Linux + Windows packaged binaries.
- Known-issues list generated and reviewed before RC publish.

## 2. Dev Kit and Editor Gates

### ACC-DK-1 Session Restore ✅
- Re-launch restores last open file, view mode, input-capture state, and remembered directories.
- Open/save dialogs default to last relevant directory.
- *Implemented: settings persistence covers view mode, split offsets, recent files, UI density, last directories.*

### ACC-EDITOR-1 IDE Usability
- CoreLX syntax highlighting active in primary editor.
- Diagnostics squiggles and diagnostics panel point to same source location.
- Find/replace and go-to-line available.

### ACC-EDITOR-2 Recovery ✅
- Unsaved work survives crash/restart through autosave/recovery journal.
- *Implemented: autosave with crash recovery.*

### ACC-DK-2 Native Window Behavior
- Linux + Windows Dev Kit builds preserve native OS title bar behavior:
  - maximize/restore via title bar controls and double-click title bar
  - minimize via title bar control
- Fullscreen toggle remains separate from maximized windowed mode.
- No regressions that grey out OS maximize control in standard windowed mode.

## 3. Debugger Gates

### ACC-DBG-1 Pause/Resume
- Pause/resume transitions are stable through repeated toggles.

### ACC-DBG-2 Frame Step
- Frame-step advances exactly requested frame count while preserving paused mode after stepping.

### ACC-DBG-3 CPU Step
- CPU-step advances instruction execution deterministically.
- Register/PC snapshots update correctly after each step.

## 4. Tool Suite Gates

### ACC-TOOLS-1 Sprite Tool ✅
- Create/edit/save/reload sprite assets round-trip without data loss.
- Tool output builds and previews in emulator.
- *Implemented: Sprite Lab with .clxsprite import/export, palette banks, and CoreLX code generation.*

### ACC-TOOLS-2 Tilemap Tool
- Tilemap edits round-trip with layer/attribute integrity.
- Tool output builds and previews in emulator.

### ACC-TOOLS-3 Sound Studio
- Music/SFX/ambience assets authored and exported via stable format.
- Playback preview works and build pipeline consumes exported assets.

### ACC-TOOLS-4 Sequencing Gate
- Sound Studio implementation does not begin until Sprite Lab + Dev Kit stabilization and required Tilemap flow gates are complete.
- YM2608 implementation work does not begin before Sound Studio start gate is met.

## 5. CoreLX/Compiler Gates

### ACC-CORELX-1 Stable Service API
- Dev Kit build flows use stable compile/service entrypoints.

### ACC-CORELX-2 Deterministic Packaging
- Manifest + bundle outputs deterministic for equivalent inputs.

### ACC-CORELX-3 Tool-Generated Diagnostics
- Compiler reports actionable diagnostics for invalid tool-generated assets/references.

## 6. YM2608 Audio Gates (Behavioral Parity Profile)

### ACC-AUDIO-1 MMIO Behavior
- YM2608 register read/write behavior conforms to profile tests.

### ACC-AUDIO-2 Timer/IRQ/Status
- Timer, status, and IRQ semantics pass defined conformance ROM suite.

### ACC-AUDIO-3 Audio Reference
- Curated patch/test set passes approved reference comparison thresholds.

### ACC-AUDIO-4 Mixed Audio Non-Regression
- Legacy APU + YM2608 mixed playback remains stable and deterministic.

### ACC-AUDIO-5 V1 Target Identity
- V1 release docs, APIs, and acceptance references target YM2608 as the canonical FM chip.
- YM2151/OPM-lite is not the V1 release target.

## 7. Galaxy Force + Documentation Gates

### ACC-GAME-1 Full Concept
- Vertical shmup core + Matrix Mode transition + showcase boss path playable end-to-end.

### ACC-GAME-2 Regression
- Scripted smoke tests validate core game loops and transitions.

### ACC-DOCS-1 Manual Coherence
- Programming manual sections use Galaxy Force code excerpts that build/run.

### ACC-DOCS-2 Runnable Snippets
- In-app docs snippets load/run successfully for mapped sections.

## 8. Gate Evidence

Release candidate requires evidence artifacts:

- test logs
- conformance report outputs
- performance snapshots
- smoke run checklist
- known issues file for that RC
