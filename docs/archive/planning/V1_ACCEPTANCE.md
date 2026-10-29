# Nitro-Core-DX V1 Acceptance Criteria

Status: Active  
Last Updated: July 22, 2026

This document defines release-blocking acceptance gates tied to `V1_CHARTER.md`.

Current alignment note: YM2608 runtime playback is now implemented well enough
to support Sound Studio MVP work. V1 still requires conformance/reference
evidence before release.

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

### ACC-TOOLS-1B Larger Sprite Workflow
- Native larger hardware sprite sizes are surfaced deliberately in the workflow
  rather than forcing manual multi-OAM composition.
- V1 must either support every hardware size from 8×8 through 128×128 in the
  tool UI, or clearly document/tool-enforce the supported subset and provide a
  safe path for larger assets.

### ACC-TOOLS-2 Tilemap Tool
- Tilemap edits round-trip with layer/attribute integrity.
- Tool output builds and previews in emulator.
- Manifest-backed and source-backed tile assets both work without manual cleanup.
- Generated snippets compile against the current language/toolchain.

### ACC-TOOLS-3 Sound Studio
- VGM/VGZ import converts to compact `.ncdxmusic`.
- Existing `.ncdxmusic` files can be inspected for duration, frames, write count, and ROM footprint.
- Playback preview uses the same emulator/YM2608 path as Build+Run, not a separate fake host player.
- Exported music assets are added to the project and consumed by the build pipeline.
- Generated source/manifest snippets compile without manual edits.
- SFX MVP provides at least a safe register-snippet/preset workflow over the current `ym.*`/`sfx` layer; full tracker composition is post-MVP unless explicitly pulled into scope.

### ACC-TOOLS-4 Sequencing Gate
- Sound Studio MVP may begin once the YM2608 runtime baseline exists:
  - MMIO register path implemented
  - timer/status/IRQ behavior path implemented
  - baseline `.ncdxmusic` playback path available for in-tool preview integration
- Sound Studio cannot be marked V1-complete until Sprite Lab, Tilemap Lab, Build+Run packaging, and generated-code compile gates all pass together.

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
- Report must distinguish runtime plumbing pass/fail from timbre/pitch/reference-quality pass/fail.

### ACC-AUDIO-4 Migration Non-Regression
- During the migration, the legacy 4-channel path (temporary scaffolding) must
  not destabilize YM2608 playback; existing ROMs keep working until legacy is
  removed. This is a transitional gate, not a permanent dual-architecture
  requirement.

### ACC-AUDIO-5 V1 Target Identity
- The single, final audio subsystem is **YM2608 / OPNA**. V1 release docs, APIs,
  and acceptance references target YM2608.
- The legacy 4-channel APU is temporary migration scaffolding, not final hardware.
- YM2151/OPM-lite is not the V1 release target or audio identity.
- Sound Studio documentation and UI copy must present YM2608/OPNA as the audio
  identity and `.ncdxmusic` as the V1 music asset format.

## 7. NitroPackInDemo + Documentation Gates

### ACC-GAME-1 Full Concept
- Title screen, pseudo-3D overworld, building interaction, interior showcase room, NPC interaction, and credits path playable end-to-end.

### ACC-GAME-2 Regression
- Scripted smoke tests validate scene flow, pseudo-3D traversal, and interaction transitions.

### ACC-DOCS-1 Manual Coherence
- Programming manual sections use NitroPackInDemo code excerpts that build/run.

### ACC-DOCS-2 Runnable Snippets
- In-app docs snippets load/run successfully for mapped sections.

## 8. Gate Evidence

Release candidate requires evidence artifacts:

- test logs
- conformance report outputs
- performance snapshots
- smoke run checklist
- known issues file for that RC
