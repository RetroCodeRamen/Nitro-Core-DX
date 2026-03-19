# YM2608 Source Selection Policy

## Purpose
Prevent cross-chip contamination during YM2608 conformance tuning by explicitly limiting emulator-source references to YM2608/OPNA-targeted files.

## Runtime Context (2026-03-19)
- cgo-backed emulator/devkit entrypoints currently default `NCDX_YM_BACKEND` to `ymfm` and expose `-audio-backend ymfm`.
- The in-tree OPM-lite model remains a code-level fallback when YMFM is unavailable.
- This policy file governs conformance/reference inputs only; it does not force runtime backend mode.

## Active Allowlist
Canonical allowlist fixture:
- `internal/apu/testdata/ym2608_source_allowlist.json`
Canonical provenance-policy fixture:
- `internal/apu/testdata/ym2608_provenance_policy.json`

Current source families included:
- `Resources/libOPNMIDI-master/src/chips/ym2608_lle.*`
- `Resources/libOPNMIDI-master/src/chips/ymfm_opna.*`
- `Resources/libOPNMIDI-master/src/chips/mame_opna.*`
- `Resources/libOPNMIDI-master/src/chips/np2_opna.*`
- `Resources/libOPNMIDI-master/src/chips/pmdwin_opna.*`
- `Resources/PMDWinS036-master/fmgen/opna.*`

## Explicitly Excluded For This Workflow
- YM2612/OPN2-specific files
- YM2610/OPNB-specific files
- Mixed-family source files not pinned to YM2608/OPNA behavior

## Validation
Enforced by:
- `TestYM2608Stage5Slice11ScenarioFixtureProvenanceIntegrity`
- `TestYM2608Stage5Slice12YM2608SourceAllowlistIntegrity`
- `TestYM2608Stage5Slice15EmulatorSourceProvenancePins`
- `TestYM2608Stage5Slice17ExtractionManifestCrosscheck`
- `TestYM2608Stage5Slice18ExtractionManifestDeterministicParity`
- `TestYM2608Stage5Slice21ProvenancePolicyFixtureIntegrity`
- `TestYM2608Stage5Slice21ProvenancePolicyEnforcement`

Canonical extraction manifest fixture:
- `internal/apu/testdata/ym2608_extraction_manifest.json`

Deterministic regeneration command:
- When the local tree includes `cmd/ym2608_manifest_gen`:
  - `go run ./cmd/ym2608_manifest_gen -write`
  - `go run ./cmd/ym2608_manifest_gen -check`
  - `go run ./cmd/ym2608_manifest_gen -check -strict-commit-pins` (optional strict provenance policy)
  - `go run ./cmd/ym2608_manifest_gen -check -policy internal/apu/testdata/ym2608_provenance_policy.json`
- When the local tree includes helper scripts:
  - `scripts/generate_ym2608_extraction_manifest.sh`
  - `scripts/check_ym2608_extraction_manifest.sh`
- In the current repository snapshot, these helper tools may be absent. Use the guarded make targets below as the canonical local entrypoint.

CI hooks:
- `make check-ym2608-manifest` (skip-tolerant when optional local tooling is absent)
- `make check-ym2608-manifest-strict` (optional strict provenance policy; skip-tolerant when tooling is absent)
- `make check-ym2608-policy` (consolidated policy-validation bundle; guarded for missing optional tooling)
- `make ci-audio`
- `.github/workflows/ym2608-ci.yml` when present in the tree
  - includes non-blocking strict-policy signal job (`ym2608-strict-policy-signal`)
  - strict-policy signal uploads artifact: `ym2608-policy-signal-log`

Required emulator-source provenance pins:
- `source_repo`
- `source_revision` (`file-sha256:<64hex>` or `commit:<hex>`)
- `extraction_tool`
- `extraction_version`
- `stimulus_script_id`
- `clock_config_id`
