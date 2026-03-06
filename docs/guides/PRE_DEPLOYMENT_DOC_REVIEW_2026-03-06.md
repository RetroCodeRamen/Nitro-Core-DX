# Pre-Deployment Documentation Review - 2026-03-06

## Scope

This review was performed before preparing release `v0.1.8`.

Primary goals:
- ensure programming/user manuals are current with implemented behavior
- ensure planning docs reflect approved direction changes
- ensure release notes/changelog are aligned with current state

## Documents Reviewed

- `README.md`
- `PROGRAMMING_MANUAL.md`
- `docs/CORELX.md`
- `docs/guides/PROGRAMMING_GUIDE.md`
- `docs/guides/README.md`
- `docs/DEVKIT_ARCHITECTURE.md`
- `docs/planning/V1_CHARTER.md`
- `docs/planning/V1_ACCEPTANCE.md`
- `docs/planning/V1_RISKS.md`
- `docs/planning/README.md`
- `docs/specifications/README.md`
- `docs/specifications/APU_FM_OPM_EXTENSION_SPEC.md`
- `CHANGELOG.md`
- `RELEASE_NOTES.md`

## Key Corrections Made

### Editor / Dev Kit Direction

- Removed stale Monaco/webview wording from active V1 planning scope.
- Updated editor risk language to native editor engine complexity.
- Kept architecture docs aligned with frontend-agnostic backend design while removing specific Monaco-forward assumptions.

### Audio Direction and Naming

- Updated active planning docs to reflect YM2608 as V1 audio target.
- Clarified that current OPM-lite path is transitional runtime behavior, not final V1 target identity.
- Updated manuals to avoid implying YM2151/OPM-lite is the final release target.

### Sprite Lab Documentation

- Updated manuals/guides to include:
  - wrapped sprite shift controls
  - palette slider + full hex entry workflow
  - aspect-preserving preview behavior

### Release Artifacts

- Added `0.1.8` changelog entry with implementation and planning changes.
- Replaced release notes content with layman-oriented `v0.1.8` summary.

## Outcome

- Core manuals and user-facing guides are aligned with current shipped behavior and near-term release direction.
- Planning docs are aligned with approved execution order and YM2608 target decision.
- Release docs are prepared for `v0.1.8` publication.
