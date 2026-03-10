# YM2608 Implementation Notes (Current + Historical Slice Log)

## Current Runtime Snapshot (2026-03-09)
- YM2608/OPNA playback is operational through the FM host MMIO window (`0x9100-0x91FF`) using the YMFM backend when available.
- Backend selection is runtime-configurable through `NCDX_YM_BACKEND`:
  - `auto` (default): YMFM when available, otherwise in-tree legacy FM path
  - `ymfm`: force YMFM (falls back with warning if unavailable)
  - `legacy`: force in-tree FM path
- Emulator and Dev Kit CLIs expose `-audio-backend auto|ymfm|legacy` and default to `auto`.
- CoreLX language-level audio APIs are still legacy-oriented; YM2608 control is currently MMIO-driven from ROM/tooling paths.

## Repository Snapshot Note
- This document includes historical slice-log entries from earlier implementation stages.
- Some historical notes reference helper tooling or CI files that are not present in the current repository snapshot.
- Current local policy validation should be driven through the guarded make targets in the root `Makefile`; direct manifest-tool commands are optional and only apply when those helper files exist locally.

## Integration Lessons Captured
- Frame pacing is non-negotiable for YM2608 replay quality:
  - game logic + YM write streams must be gated to one update per real frame boundary
  - VBlank-only rendering plus frame-edge gating prevents accelerated/garbled playback
- CPU instruction encoding caveat:
  - `CMP R0,#imm` can alias branch decode if emitted as mode=1 with `reg1=0,reg2=0`
  - tooling now avoids ambiguous encoding for R0 immediate compares

## Scope
Stages 1-2 introduce the YM2608 core scaffold used by Nitro-Core-DX APU MMIO and FM register-state latching.

Implemented:
- Dual YM2608 host ports (`A1=0` and `A1=1`) in the existing CPU-visible window `0x9100-0x91FF`
- Register shadowing for both ports
- Status0/Status1 read paths
- Timer A / Timer B register behavior (`$24`, `$25`, `$26`, `$27`, `$29`)
- Timer flags and IRQ state propagation into host-visible status
- Rising-edge IRQ callback integration through APU -> CPU timer interrupt path
- FM register decode/state latch for 6 channels and 4 operators per channel (no full synthesis yet)
- Key-on/key-off state latch from `$28`
- Channel/operator parameter latching from `$30-$B6` across both ports

Deferred (explicitly):
- FM operator synthesis output path (6 channels, 4-op signal generation)
- SSG tone/noise/envelope synthesis
- ADPCM A/B decode/playback
- Rhythm block sample playback/mixing
- Cycle-accurate bus wait timing beyond deterministic placeholder

## Manual-Driven Register Behavior Used
Reference file: `YM2608J Translated.pdf` (English translation).

Used sections:
- Data bus control (`A1/A0` address/data read/write modes)
- Timer formulas:
  - `tA = 72 * (1024 - NA) / φM`
  - `tB = 1152 * (256 - NB) / φM`
- Timer controller `$27` (load/start, flag-enable, flag-reset actions)
- IRQ enable register `$29` (`EN_TA`, `EN_TB`)

## Architectural Notes
- Core is isolated in `internal/apu/ym2608.go`.
- Runtime FM path is backend-selectable (`internal/apu/fm_opm.go` + YMFM bridge when built with `ymfm_cgo`).
- Legacy 4-channel APU path still exists for compatibility and non-YM2608 fallback behavior.
- Host-cycle to YM-master-clock conversion is explicit and deterministic:
  - Host domain default: `7.67 MHz`
  - YM master clock default: `8.00 MHz`
- Current bus busy handling is deterministic placeholder (not yet cycle-exact against full chip wait table).

## Assumptions / Ambiguities
- Translation text around status-flag wording is ambiguous in places; implementation currently mirrors timer flags into both status ports.
- IRQ generation is currently modeled from timer flags + `$29` timer enable bits only.
- Non-timer status bits (`EOS`, `BRDY`, `ZERO`) are reserved for future ADPCM/rhythm stages.

## Validation Added in Stages 1-2
- Unit:
  - Port0/Port1 register independence
  - Timer A overflow -> status flag behavior
  - Timer A clear action via `$27`
  - IRQ flag assertion/clear behavior
  - FM register decode across port0/port1
  - FM frequency/block latch decode
  - FM key-on/key-off latch decode
- Integration:
  - Bus-level MMIO timer programming + status read
  - Emulator-level IRQ wiring (APU timer -> CPU timer interrupt pending)

## Stage 3 Update (Slices 2-4)
Implemented:
- Deterministic FM envelope state machine scaffolding per operator:
  - `Off -> Attack -> Decay -> Sustain -> Release -> Off`
- Key-on/key-off transitions now drive envelope phase transitions.
- Envelope progression tests added for:
  - attack/decay/sustain transitions
  - key-off release to off
- Deterministic FM phase accumulator core per operator:
  - per-step phase increment derived from channel `FNUM/BLOCK` and operator `MUL`
  - phase wrap constrained to fixed accumulator width
  - key-on resets active operator phase
- Phase progression tests added for:
  - deterministic increment stepping
  - wrap behavior
  - active-operator-only key-on phase reset
- Calibration pass updates:
  - Envelope progression now uses a deterministic rate/period model instead of fixed linear-per-tick stepping.
  - Operator key scaling (`KS`) and channel block now influence effective envelope rates.
  - `MUL` handling moved to an explicit OPN-style multiplier table (`1,2,4,...,30`) and phase increment now models `MUL=0` as half-rate behavior.
  - Added tests for multiplier mapping, half-rate `MUL=0` phase behavior, and higher-rate envelope progression.

Assumptions (temporary placeholders, explicitly non-final):
- Envelope tick cadence currently uses fixed scheduler bucket (`24` YM cycles per envelope tick).
- Envelope period/step curves are deterministic calibrated approximations (not yet validated against full conformance vectors).
- Effective rate path currently uses block/KS influence without full keycode/detune coupling.
- This is intentionally a conformance scaffold, not final chip-accurate curve/timing yet.

## Stage 4 Update (Slice 1)
Implemented:
- Constrained audible FM bring-up path:
  - deterministic single-carrier render path (operator 0 per channel)
  - envelope/TL attenuation applied to output
  - explicit float/fixed clipping safety
- Added audible-path tests for:
  - non-zero waveform generation under valid channel/operator state
  - mute/enable gating behavior
  - clipping/range safety for float + fixed outputs

Assumptions (temporary placeholders, explicitly non-final):
- No full algorithm graph yet (operators 1-3 not yet modulating carrier path).
- Output amplitude model currently uses deterministic linear attenuation (envelope + TL), not full YM dB/log curve.
- Sine generation currently uses a scalar math path for deterministic bring-up; replace with hardware-aligned lookup/exp/log path in subsequent slices.

## Stage 4 Update (Slice 2)
Implemented:
- Constrained operator routing for one algorithm path (`ALG=0`):
  - modulator (`op1`) phase-modulates carrier (`op0`)
  - modulation depth tied to channel feedback field (`FB`)
- Added deterministic tests for:
  - modulation impact against carrier baseline
  - modulation polarity effect
  - output-bound safety under modulation

Assumptions (temporary placeholders, explicitly non-final):
- `ALG=0` path currently models only a minimal `op1 -> op0` chain, not full 4-operator graph behavior.
- Modulation depth mapping from `FB` is deterministic scaffolding, not final YM-accurate operator pipeline gain.
- Phase/modulation still uses scalar sine path pending log-sine/exp hardware-aligned pipeline bring-up.

## Stage 4 Update (Slice 3)
Implemented:
- Replaced scalar runtime sine path with deterministic LUT-based phase output path.
- Replaced linear attenuation model with deterministic log-style attenuation LUT path.
- Expanded constrained algorithm coverage:
  - added `ALG=7` mapping as a multi-carrier scaffold path.
- Added deterministic tests for:
  - phase LUT quadrant behavior
  - gain LUT monotonic attenuation behavior
  - `ALG=7` mapping effect on channel output

Assumptions (temporary placeholders, explicitly non-final):
- LUT sizes/resolution are bring-up scaffolding and not yet final chip-conformance dimensions.
- `ALG=7` mapping is a constrained scaffold (not full YM per-algorithm operator graph fidelity).
- Log-domain/gain behavior is deterministic and hardware-minded, but still needs conformance tuning against full YM operator tables.

## Stage 4 Update (Slice 4)
Implemented:
- Expanded constrained algorithm coverage with `ALG=4` dual-branch scaffold:
  - `op1 -> op0` branch
  - `op3 -> op2` branch
  - branch-mixed channel output
- Replaced formula-based feedback depth with explicit table-driven mapping (`FB` depth LUT).
- Added deterministic tests for:
  - feedback depth LUT monotonicity
  - stronger modulation response at high `FB` vs low `FB`
  - `ALG=4` operator-role behavior (branch cancellation and branch-isolation effects)

Assumptions (temporary placeholders, explicitly non-final):
- `ALG=4` path is constrained scaffolding and not full YM per-algorithm operator graph fidelity.
- Feedback depth LUT is deterministic bring-up calibration in phase units, not yet manual-conformance final values.
- Operator graph/mix behavior remains intentionally simplified until algorithm coverage reaches conformance lock.

## Stage 5 Update (Slice 1)
Implemented:
- Added constrained `ALG=2` mapping scaffold:
  - chain branch: `op2 -> op1 -> op0`
  - carrier branch: `op3`
- Added first algorithm-role matrix tests covering constrained role expectations for:
  - `ALG=0`
  - `ALG=2`
  - `ALG=4`
  - `ALG=7`
- Added explicit feedback-depth LUT monotonicity + role sensitivity checks in test suite.

Assumptions (temporary placeholders, explicitly non-final):
- `ALG=2` includes a small direct op2 pre-modulation contribution in the constrained scaffold to keep operator-role influence observable at current LUT resolution.
- Role matrix currently validates constrained scaffold behavior, not full YM-conformance per-operator signal graph.
- Feedback-depth LUT values remain calibrated scaffolding and require conformance-vector alignment.

## Stage 5 Update (Slice 2)
Implemented:
- Added constrained `ALG=6` mapping scaffold:
  - direct carriers: `op0`, `op1`
  - modulated carrier path: `op3 -> op2`
- Expanded algorithm-role matrix coverage with explicit `ALG=6` role checks.
- Tightened feedback-depth mapping into a shift-ladder style LUT and added ladder-shape validation tests.

Assumptions (temporary placeholders, explicitly non-final):
- `ALG=6` path is a constrained scaffold and not full YM per-algorithm graph fidelity.
- Feedback ladder values are still calibrated scaffolding in phase-domain units, not final conformance-locked values.
- Role matrix now covers broader constrained behavior, but still requires expansion before declaring algorithm-graph conformance.

## Stage 5 Update (Slice 3)
Implemented:
- Added constrained `ALG=3` mapping scaffold:
  - dual modulators into `op0` (`op1`, `op2`)
  - independent carrier `op3`
- Extended role-matrix coverage across all currently implemented constrained algorithms:
  - `ALG=0`, `ALG=2`, `ALG=3`, `ALG=4`, `ALG=6`, `ALG=7`
- Added deterministic feedback-depth reference-vector hook for conformance tuning:
  - stable per-`FB` modulation delta vector generation for future manual/LLE comparison tooling.

Assumptions (temporary placeholders, explicitly non-final):
- `ALG=3` remains constrained scaffolding, not full YM per-algorithm graph fidelity.
- Role-matrix assertions are tuned to current LUT-resolution scaffolding and may tighten as tables/curves converge.
- Feedback reference vectors currently validate deterministic internal behavior, not yet external conformance targets.

## Stage 5 Update (Slice 4)
Implemented:
- Added external conformance fixture for feedback-depth reference behavior:
  - `internal/apu/testdata/ym2608_fb_reference_vector.json`
- Added fixture conformance test that compares the internal deterministic reference hook against fixture values with explicit tolerance.
- Kept existing role-matrix coverage intact while establishing a baseline mechanism for future conformance-vector updates.

Assumptions (temporary placeholders, explicitly non-final):
- Current fixture values represent deterministic scaffold behavior, not yet final hardware-conformance targets.
- Fixture tolerance is currently tight for deterministic regression tracking and may be adjusted only when reference-vector methodology changes.

## Stage 5 Update (Slice 5)
Implemented:
- Added algorithm-role external fixture:
  - `internal/apu/testdata/ym2608_algorithm_role_vector.json`
- Added fixture conformance test for per-algorithm/per-operator deltas using the deterministic role-reference hook.
- Added reusable deterministic role-reference hook in YM2608 core:
  - `AlgorithmRoleReferenceVector()`

Assumptions (temporary placeholders, explicitly non-final):
- Fixture values represent current constrained scaffold behavior and are intended as regression baselines, not final silicon-conformance truth.
- Tolerance window is currently chosen for deterministic CI stability and will tighten/adjust as conformance vectors mature.

## Stage 7 Audio-Triage Update (Demo.vgz / Demo.wav)
Implemented:
- Added an opt-in FM accuracy mode on YM control bit `0x20`:
  - control write `0xA1` (reset+enable+accurate)
  - then `0x21` (enable+accurate)
- Runtime behavior in accurate mode:
  - canonical OPN 4-op algorithm routing table for ALG0-ALG7
  - operator feedback applied on OP1 path only
  - OPN slot decode correction (`+0/+4/+8/+C => OP1/OP3/OP2/OP4`)
  - tighter phase-step scaling for VGM pitch alignment
- Kept legacy constrained FM path as default (bit `0x20` clear) to preserve existing slice-regression suite behavior.
- Updated YM song/scale ROM builders to enable accurate mode by default.
- Added reference diagnostics:
  - `internal/apu/ym2608_vgm_reference_diagnostic_test.go` (direct VGM->YM compare)
  - `internal/emulator/ym2608_demo_song_reference_diagnostic_test.go` (ROM playback compare)

Current status:
- Direct VGM replay median pitch aligns to reference (`ratio ~ 1.0` in current diagnostic window).
- Timbre still shows excess high-frequency content versus reference (ZCR ratio remains elevated), so further operator/envelope/feedback conformance refinement is still required.

## Stage 5 Update (Slice 6)
Implemented:
- Added first conformance-tuning candidate FB-depth table in YM2608 core (non-runtime-swapped).
- Added candidate-vs-baseline drift guardrail fixture:
  - `internal/apu/testdata/ym2608_fb_candidate_guardrails.json`
- Added fixture conformance test for candidate drift metrics:
  - per-index drift ceiling
  - max absolute drift ceiling
  - aggregate drift ceiling
  - minimum changed-index check (ensures candidate meaningfully differs)

Assumptions (temporary placeholders, explicitly non-final):
- Candidate table remains evaluation-only; runtime still uses baseline FB table.
- Guardrail thresholds are scaffold-level risk controls and should be revised when conformance vectors from external references are introduced.

## Stage 5 Update (Slice 7)
Implemented:
- Added external A/B candidate-table fixture:
  - `internal/apu/testdata/ym2608_fb_candidate_sets.json`
- Added test-only A/B candidate-selection harness:
  - evaluates each candidate against baseline vector drift guardrails
  - deterministically selects the best eligible candidate by drift score
  - validates expected winner from fixture metadata
- Added explicit candidate-table parsing/validation helpers in test code to keep fixture workflow strict.

Assumptions (temporary placeholders, explicitly non-final):
- Candidate sets are provisional external conformance trends and are not automatically promoted to runtime behavior.
- Selection harness remains test-only; runtime FB table is still unchanged.

## Stage 5 Update (Slice 8)
Implemented:
- Evolved candidate fixture to multi-candidate ranking metadata with explicit promotion gates:
  - added candidate C
  - added deterministic expected rank order for guardrail-passing candidates
  - added promotion gate thresholds and expected promoted candidate field
- Upgraded selection harness to ranking + promotion-gate validation:
  - ranks all guardrail-passing candidates by `(sumDrift, maxDrift, name)`
  - validates expected winner and expected order
  - evaluates stricter promotion gates separately from guardrail eligibility
  - logs deterministic drift/ranking report for conformance workflow visibility
- Kept runtime behavior unchanged (still baseline FB table only).

Assumptions (temporary placeholders, explicitly non-final):
- Candidate ranking and promotion checks remain test-only governance and do not auto-swap runtime FB tables.
- Promotion gate thresholds are intentionally stricter than guardrails and currently block promotion pending stronger external conformance evidence.

## Stage 5 Update (Slice 9)
Implemented:
- Added external multi-scenario candidate fixture:
  - `internal/apu/testdata/ym2608_fb_candidate_scenarios.json`
  - includes manual-derived, die-shot-informed, and hybrid reference vectors
- Added scenario-stability harness:
  - `TestYM2608Stage5Slice9FBCandidateScenarioStability`
  - ranks candidates per scenario by `(sumErr, maxErr, name)`
  - enforces stable winner/order across all scenarios
  - verifies scenario-level rejection pressure so rankings remain meaningful
- Kept runtime FB table unchanged; scenario workflow is conformance governance only.

Assumptions (temporary placeholders, explicitly non-final):
- Scenario vectors are provisional external trend targets, not yet lab-grade silicon captures.
- Stability checks validate relative ordering robustness, not final absolute conformance lock.

## Stage 5 Update (Slice 10)
Implemented:
- Added explicit runtime-swap dry-run gate harness:
  - `TestYM2608Stage5Slice10RuntimeSwapDryRunGate`
  - combines promotion thresholds (from candidate fixture) with scenario stability checks
  - refuses swap unless both promotion and scenario-stability gates pass
- Dry-run gate is fixture-driven and currently expected to remain closed (no runtime table swap).
- Kept runtime YM2608 feedback table path unchanged.

Assumptions (temporary placeholders, explicitly non-final):
- Dry-run gate still uses provisional trend vectors and fixture thresholds; it is a governance mechanism, not final silicon promotion logic.
- Gate closure/open state is controlled by fixture metadata and test criteria, not runtime auto-promotion behavior.

## Stage 5 Update (Slice 11)
Implemented:
- Added scenario-set versioning/provenance metadata to scenario fixture:
  - `scenario_set_id`
  - `promotion_requires_measured`
  - per-scenario provenance block (`kind`, `reference`, `trace_id`, `trace_sha256`, `reviewed_by`, `review_state`)
- Added fixture integrity test for provenance governance:
  - `TestYM2608Stage5Slice11ScenarioFixtureProvenanceIntegrity`
  - validates scenario/candidate uniqueness and provenance field completeness
  - enforces stricter rules for measured traces (64-hex hash + approved review state)
- Updated dry-run runtime-swap gate to require measured scenarios when `promotion_requires_measured=true`.
- Runtime FB table remains unchanged.

Assumptions (temporary placeholders, explicitly non-final):
- Current scenarios remain provisional; measured-ready gate intentionally keeps runtime promotion closed.
- Provenance metadata governs fixture quality, not synthesis correctness directly.

## Stage 5 Update (Slice 12)
Implemented:
- Added explicit YM2608-only source allowlist fixture:
  - `internal/apu/testdata/ym2608_source_allowlist.json`
  - curated to OPNA/YM2608-targeted source files only
- Added allowlist integrity gate:
  - `TestYM2608Stage5Slice12YM2608SourceAllowlistIntegrity`
  - verifies allowlist entries exist and are YM2608/OPNA-specific by path policy
- Extended provenance integrity gate to accept `kind=emulator_source` only when the referenced file is in the YM2608 allowlist.
- Updated scenario provenance references away from mixed-chip artifacts to YM2608-specific paths in `Resources`.
- Kept `promotion_requires_measured=true`; runtime promotion remains locked.

Assumptions (temporary placeholders, explicitly non-final):
- Emulator-source provenance improves source targeting but is not a substitute for measured silicon captures.
- Allowlist policy is path-level and can be tightened further with deeper semantic classification if needed.

## Stage 5 Update (Slice 13)
Implemented:
- Added an additional independent YM2608 emulator-source scenario:
  - `ymfm_opna_trend` (referencing allowlisted `ymfm_opna.cpp`)
- Added emulator-source consensus stability gate:
  - `TestYM2608Stage5Slice13EmulatorSourceConsensusStability`
  - requires at least 3 emulator-source scenarios
  - enforces stable winner/order across emulator-source scenarios
  - keeps rejection-pressure checks active per scenario
- Kept measured-promotion requirement intact (`promotion_requires_measured=true`), so runtime swap remains blocked.

Assumptions (temporary placeholders, explicitly non-final):
- Emulator-source consensus improves confidence but is still non-silicon validation.
- Consensus checks currently use fixture-level stable order as the declared target.

## Stage 5 Update (Slice 14)
Implemented:
- Switched promotion policy from measured-only to emulator-consensus gating for current workflow:
  - `promotion_requires_measured=false` in scenario fixture
  - promotion thresholds adjusted to allow a unique promoted candidate under current consensus vectors
  - promoted candidate now expected as `conformance_a`
- Kept all deterministic safeguards active:
  - candidate drift guardrails
  - ranking/order checks
  - scenario stability checks
  - emulator-source consensus checks
- Runtime table swap remains governed by dry-run gate tests; this change removes only the hard measured-data prerequisite.

Assumptions (temporary placeholders, explicitly non-final):
- Promotion under this policy is consensus-validated, not silicon-validated.
- If measured captures become available later, policy can be tightened back to measured-only without architectural rewrites.

## Stage 5 Update (Slice 15)
Implemented:
- Added provenance pinning requirements for `emulator_source` scenarios:
  - `source_repo`
  - `source_revision` (validated format: `file-sha256:<64hex>` or `commit:<hex>`)
  - `extraction_tool`
  - `extraction_version`
- Pinned current emulator-source scenarios to exact source file hashes.
- Added explicit provenance pin validation test:
  - `TestYM2608Stage5Slice15EmulatorSourceProvenancePins`
  - verifies pinned source hash matches current source file content for each emulator-source scenario.

Assumptions (temporary placeholders, explicitly non-final):
- Current pinning uses file-content hashes rather than upstream commit SHAs because sources are local snapshots.
- Extraction metadata is currently curated manually and should be migrated to scripted extraction provenance over time.

## Stage 5 Update (Slice 16)
Implemented:
- Added extraction-profile provenance fields for `emulator_source` scenarios:
  - `stimulus_script_id`
  - `clock_config_id`
- Enforced extraction-profile fields in provenance integrity checks.
- Added extraction-profile consistency gate:
  - `TestYM2608Stage5Slice16ExtractionProfileConsistency`
  - requires emulator-source scenarios to share the same stimulus/clock profile for apples-to-apples ranking.
- Updated current emulator-source scenarios with the same extraction profile IDs.

Assumptions (temporary placeholders, explicitly non-final):
- Current stimulus/clock profile IDs are manually curated identifiers.
- Consistency is enforced at the identifier layer; deep stimulus program equivalence is still a future hardening step.

## Stage 5 Update (Slice 17)
Implemented:
- Added commit-SHA pin support path in source-revision validation:
  - `source_revision` now supports `commit:<hex>` in addition to `file-sha256:<64hex>`.
  - commit reachability is validated when git metadata is present in the pinned source repo.
- Added reproducible extraction manifest fixture:
  - `internal/apu/testdata/ym2608_extraction_manifest.json`
- Added extraction manifest cross-check gate:
  - `TestYM2608Stage5Slice17ExtractionManifestCrosscheck`
  - verifies manifest entries match emulator-source scenario provenance fields and profile IDs.

Assumptions (temporary placeholders, explicitly non-final):
- Current local source snapshots primarily use file-content hash pins; commit pins are now supported but optional.
- Manifest IDs still depend on curated extraction profile identifiers until scripted extraction manifests are in place.

## Stage 5 Update (Slice 18)
Implemented:
- Added deterministic extraction-manifest generator command:
  - `cmd/ym2608_manifest_gen/main.go`
- Added convenience regeneration script:
  - `scripts/generate_ym2608_extraction_manifest.sh`
- Added deterministic parity gate:
  - `TestYM2608Stage5Slice18ExtractionManifestDeterministicParity`
  - generates expected manifest from scenario provenance and requires exact parity with checked-in manifest.
- Kept prior manifest cross-check gate intact.

Assumptions (temporary placeholders, explicitly non-final):
- Deterministic generation currently derives from scenario fixture semantics and sorted scenario names.
- Manual edits to manifest are still possible but now test-gated against deterministic regeneration.

## Stage 5 Update (Slice 19)
Implemented:
- Added manifest generator check mode (`-check`) in `cmd/ym2608_manifest_gen`.
- Added check script:
  - `scripts/check_ym2608_extraction_manifest.sh`
- Added Makefile CI targets:
  - `check-ym2608-manifest`
  - `ci-audio` (manifest check + YM2608 test packages)
- Added dedicated CI workflow:
  - `.github/workflows/ym2608-ci.yml`
  - fails on manifest drift before running YM2608 audio gate tests.

Assumptions (temporary placeholders, explicitly non-final):
- CI gate currently targets manifest parity + core audio packages and can be broadened later if needed.
- Commit-pinned validation remains optional where source repos lack stable git metadata.

## Stage 5 Update (Slice 20)
Implemented:
- Expanded manifest check mode in `cmd/ym2608_manifest_gen` to emit a deterministic minimal diff summary on drift.
  - Summary includes top-level manifest fields and per-scenario entry field deltas.
  - Falls back to first differing line when structured parse comparison is not possible.
- Added optional strict provenance policy in manifest tooling:
  - new flag: `-strict-commit-pins`
  - when enabled, emulator-source scenarios must use `commit:<sha>` source pins if the referenced source repo has git metadata.
  - strict mode validates commit format and commit reachability with `git rev-parse`.
- Added optional strict script/Make integration:
  - `scripts/check_ym2608_extraction_manifest.sh` now accepts `YM2608_STRICT_COMMIT_PINS=1`.
  - Make target added: `check-ym2608-manifest-strict`.
- Added focused unit tests in `cmd/ym2608_manifest_gen/main_test.go` for:
  - diff-summary output behavior
  - strict commit-pin policy pass/fail paths.

Assumptions (temporary placeholders, explicitly non-final):
- Strict commit-pin enforcement remains opt-in until external source repos are consistently available with git metadata.
- Diff summary is intended for review speed and does not replace deterministic byte-level parity checks.

## Stage 5 Update (Slice 21)
Implemented:
- Added explicit provenance policy fixture for emulator-source scenarios:
  - `internal/apu/testdata/ym2608_provenance_policy.json`
  - per-scenario fields:
    - `default_required_pin`
    - `strict_git_required_pin`
  - current policy defaults to file-hash pins, with strict-mode commit requirements declared for git-backed repos.
- Added provenance-policy enforcement tests:
  - `TestYM2608Stage5Slice21ProvenancePolicyFixtureIntegrity`
  - `TestYM2608Stage5Slice21ProvenancePolicyEnforcement`
  - validates policy fixture integrity, one-to-one coverage with emulator-source scenarios, and pin-format compliance.
  - validates strict-policy intent when git metadata is present.
- Added optional non-blocking strict CI signal job:
  - `.github/workflows/ym2608-ci.yml` job `ym2608-strict-policy-signal`
  - runs `make check-ym2608-manifest-strict` with `continue-on-error: true`.

Assumptions (temporary placeholders, explicitly non-final):
- Strict-policy CI signal is intentionally non-blocking while source snapshots remain mixed between extracted archives and git-backed trees.
- Per-scenario policy currently governs pin format and strict intent; extraction-method policy details remain out of scope for this slice.

## Stage 5 Update (Slice 22)
Implemented:
- Added policy-aware tooling checks directly in `cmd/ym2608_manifest_gen`:
  - new `-policy` flag to load provenance-policy fixture
  - validates one-to-one coverage between emulator-source scenarios and policy scenarios
  - enforces per-scenario `default_required_pin` on every run
  - enforces per-scenario `strict_git_required_pin` under `-strict-commit-pins` when git metadata is present
  - preserves deterministic manifest parity checks and diff summary behavior.
- Updated manifest scripts to include policy validation by default:
  - `scripts/check_ym2608_extraction_manifest.sh`
  - `scripts/generate_ym2608_extraction_manifest.sh`
  - both now pass `-policy internal/apu/testdata/ym2608_provenance_policy.json` (override via `YM2608_POLICY_PATH`).
- Added policy-aware unit coverage in `cmd/ym2608_manifest_gen/main_test.go` for:
  - scenario/policy coverage parity
  - default pin enforcement
  - strict-mode opt-in enforcement
  - strict git-backed commit requirement behavior.

Assumptions (temporary placeholders, explicitly non-final):
- Policy default pin requirement currently remains `file-sha256` for existing emulator-source scenarios to keep local snapshot workflows stable.
- Strict git-backed commit requirement remains active only when strict mode is requested.

## Stage 5 Update (Slice 23)
Implemented:
- Added consolidated policy-validation bundle target:
  - `make check-ym2608-policy`
  - runs default + strict manifest checks and policy-focused tests.
- Upgraded strict CI signal job to publish triage artifacts:
  - `.github/workflows/ym2608-ci.yml` strict job now runs `make check-ym2608-policy`
  - uploads `ym2608-policy-signal-log` artifact with full output for drift diagnosis.

Assumptions (temporary placeholders, explicitly non-final):
- Strict-policy CI remains non-blocking (`continue-on-error: true`) while source provenance remains mixed between archive snapshots and git-backed trees.

## Stage 5 Update (Slice 24)
Implemented:
- Shifted from policy governance back to YM2608 functional behavior work.
- Corrected FM key-on register (`$28`) channel decode to OPN-style mapping:
  - valid channel-bit patterns: `0,1,2,4,5,6` -> channels `0..5`
  - invalid/unmapped patterns: `3,7` (ignored)
- Added deterministic conformance test for `$28` key-on mapping and invalid-pattern rejection:
  - `TestYM2608Stage6Slice24KeyOnChannelDecodeOPNMapping`
- Updated existing Stage-2 key-on decode test vector to use the OPN-valid channel-bit pattern for channel index 5 (`ch bits=6`).

Assumptions (temporary placeholders, explicitly non-final):
- This slice aligns decode behavior with OPN key-on channel selection semantics but does not yet cover CSM/special mode key-on interactions.
- Functional scope remains FM register/timing path only; SSG/ADPCM/rhythm remain deferred.

## Stage 5 Update (Slice 25)
Implemented:
- Added deterministic FM timer/IRQ edge-order conformance tests for timer-B and mixed timer flag interactions:
  - `TestYM2608Stage6Slice25TimerBEnableAfterOverflowLatchesIRQ`
  - `TestYM2608Stage6Slice25TimerBClearBeforeEnableAvoidsStaleIRQ`
  - `TestYM2608Stage6Slice25TimerAAndBFlagORUnderIRQMasks`
- Covered key control-path ordering expectations:
  - masked timer-B flag can assert IRQ immediately when `$29` timer-B IRQ is enabled later
  - clear-before-enable sequence does not resurrect stale IRQ
  - IRQ remains asserted while either enabled timer flag remains latched (A/B OR behavior), and clears once both are cleared
- No YM core behavior changes were required for this slice; existing implementation passed the added conformance vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Timer conformance currently targets deterministic flag/IRQ control-path ordering and not yet full silicon-verified sub-cycle timing nuances.
- CSM/special mode timer interactions remain deferred.

## Stage 5 Update (Slice 26)
Implemented:
- Added deterministic FM key-on/key-off edge conformance tests for mixed operator masks and rapid write ordering:
  - `TestYM2608Stage6Slice26MixedMaskSelectiveReleaseAndRetrigger`
  - `TestYM2608Stage6Slice26ZeroMaskReleasesOnlyHeldOperators`
  - `TestYM2608Stage6Slice26RapidMixedMaskOrdering`
- Covered key mask transition behavior at the operator level:
  - selective release for dropped operators
  - retrigger/reset behavior for operators present in a new key-on mask
  - zero-mask behavior only releasing previously held operators
- No YM core behavior changes were required for this slice; existing key-on control-path implementation passed added vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Current conformance focuses on deterministic control-path semantics for operator key state and phase reset, not yet full special-mode (CSM) key-on behavior.
- Rapid-write coverage is functional ordering at register-write granularity, not sub-cycle bus timing.

## Stage 5 Update (Slice 27)
Implemented:
- Added deterministic register-edge conformance tests for `$28` key-on writes interleaved with FM parameter/control writes (`$30-$B6`, `$A0/$A4`):
  - `TestYM2608Stage6Slice27KeyOnInterleavedWithOperatorLatchPreservesKeyState`
  - `TestYM2608Stage6Slice27KeyOnAndFrequencyOrderingEquivalence`
  - `TestYM2608Stage6Slice27KeyOnInterleavedWithChannelControlWrites`
- Covered ordering-sensitive state guarantees:
  - operator/channel parameter latches do not clobber key-held/release state transitions
  - key-on/frequency write ordering converges to equivalent phase progression once final register state is identical before stepping
  - interleaved channel-control writes preserve final key-mask semantics and channel latch integrity
- No YM core behavior changes were required for this slice; existing implementation passed added ordering vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Ordering checks are deterministic host-write ordering conformance, not cycle-exact sub-step bus hazard modeling.
- Special mode/CSM key-on timing remains deferred.

## Stage 5 Update (Slice 28)
Implemented:
- Added deterministic register-edge conformance tests for timer/control ordering interactions across `$27/$29/$28`:
  - `TestYM2608Stage6Slice28TimerAEnableVsKeyOnOrdering`
  - `TestYM2608Stage6Slice28TimerBClearInterleavedWithKeyOnDoesNotResurrectIRQ`
  - `TestYM2608Stage6Slice28TimerAClearEnableInterleaveConvergesAfterNextExpiry`
- Covered ordering-sensitive control-path guarantees:
  - timer-flag/IRQ behavior is stable under `$29` enable ordering interleaved with key-on writes
  - timer-flag clear with key-on interleaving does not resurrect stale IRQ/status bits
  - alternate clear/enable/key-on ordering converges to equivalent post-expiry timer/IRQ state
- No YM core behavior changes were required for this slice; existing implementation passed the added vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Register-edge conformance remains deterministic at host-write granularity; cycle-exact sub-step hazard modeling is still deferred.
- CSM/special mode and non-timer status-bit interactions remain deferred.

## Stage 5 Update (Slice 29)
Implemented:
- Began FM algorithm-graph expansion beyond constrained fallback behavior by adding explicit render paths for previously unimplemented algorithms:
  - `ALG=1` (`ymRenderAlg1`)
  - `ALG=5` (`ymRenderAlg5`)
- Updated channel render dispatch to route `ALG=1` and `ALG=5` through dedicated graph paths instead of carrier-only fallback.
- Added deterministic role-regression tests for expanded algorithms:
  - `TestYM2608Stage6Slice29Alg1ExpandedRoleInfluence`
  - `TestYM2608Stage6Slice29Alg5ExpandedRoleInfluence`
  - `TestYM2608Stage6Slice29Alg1AndAlg5RoleRegressionVectorDeterministic`
- Applied one targeted conformance fix during bring-up:
  - strengthened `ALG=1` op2 coupling into the modulator chain so op2 influence is observably preserved under deterministic phase/LUT conditions.

Assumptions (temporary placeholders, explicitly non-final):
- `ALG=1` and `ALG=5` are expanded scaffolds intended to improve per-operator routing fidelity, not yet declared final silicon-accurate OPN graph parity.
- Role-regression vectors are deterministic internal conformance guards and should later be compared against stronger external references.

## Stage 5 Update (Slice 30)
Implemented:
- Expanded algorithm-role matrix coverage to all 8 algorithms under deterministic conformance harness:
  - added `TestYM2608Stage6Slice30AlgorithmRoleMatrixAll8`
  - includes explicit sensitivity expectations for `ALG=1` and `ALG=5` along with existing algorithms.
- Expanded fixture-backed role regression vectors to include all 8 algorithms:
  - updated `internal/apu/testdata/ym2608_algorithm_role_vector.json`
  - updated fixture conformance checks to require 8 algorithm rows.
- Expanded internal deterministic role reference shape:
  - `YMAlgorithmRoleReference` now carries 8 algorithms and 8x4 delta matrix.
  - `AlgorithmRoleReferenceVector()` now emits ordered algorithms `[0,1,2,3,4,5,6,7]`.

Assumptions (temporary placeholders, explicitly non-final):
- Role matrix for `ALG=1` and `ALG=5` reflects expanded deterministic scaffold behavior and is not yet full silicon-validated graph parity.
- Fixture vectors are deterministic regression locks for implementation stability and should be recalibrated with stronger external references when available.

## Stage 5 Update (Slice 31)
Implemented:
- Refined FM envelope key-scaling coupling to use a pitch-aware keycode derived from both channel `BLOCK` and high `FNUM` bits:
  - added `ymKeyCode(block, fnum)` and `ymKeyScaleOffset(ks, keyCode)` helpers.
  - updated `ymEffectiveRate` and envelope phase rate calculations (attack/decay/sustain/release) to consume keycode-aware scaling.
- Added deterministic conformance vectors for key-scaling/rate-phase behavior:
  - `TestYM2608Stage6Slice31EffectiveRateIncludesFNumKeyCode`
  - `TestYM2608Stage6Slice31HigherKeyCodeAdvancesEnvelopeFaster`
  - `TestYM2608Stage6Slice31KeyScaleEnvelopeDeltaDoesNotDisturbPhaseStep`
- Verified no regression in all-8 algorithm role matrix and fixture conformance gates while applying envelope refinements.

Assumptions (temporary placeholders, explicitly non-final):
- Keycode composition (`BLOCK` + high `FNUM` bits) is a hardware-minded deterministic scaffold and not yet declared final silicon-exact OPN envelope-rate behavior.
- SSG-EG and special mode/CSM timing interactions are still deferred and may require additional envelope scheduler adjustments.

## Stage 5 Update (Slice 32)
Implemented:
- Added deterministic envelope boundary conformance vectors for dynamic keycode/key-scaling rewrites near transition edges:
  - `TestYM2608Stage6Slice32AttackBoundaryRespectsLateKeycodeRewrite`
  - `TestYM2608Stage6Slice32DecayBoundaryRespectsLateKeycodeRewrite`
  - `TestYM2608Stage6Slice32ReleaseOffAfterRapidRekeyWithMixedKeycode`
- Validated transition ordering behavior for:
  - attack->decay edge when late `BLOCK/FNUM` rewrite changes effective envelope period.
  - decay->sustain edge when late keycode rewrite changes effective decay step at a shared boundary tick.
  - release->off edge convergence under rapid re-key sequences interleaved with keycode changes.
- No YM2608 core code-path change was required in this slice; existing logic conformed once edge vectors were added.

Assumptions (temporary placeholders, explicitly non-final):
- Envelope-edge vectors are deterministic scaffolds at host-write granularity; they do not yet claim cycle-exact sub-step bus timing parity.
- SSG-EG interactions and special mode/CSM envelope behavior remain deferred.

## Stage 5 Update (Slice 33)
Implemented:
- Added deterministic conformance for register-write ordering inside one envelope scheduler window with `KS`, `BLOCK/FNUM`, and key-on/key-off permutations:
  - `TestYM2608Stage6Slice33EquivalentWritePermutationsConvergeWithinEnvelopeWindow`
  - `TestYM2608Stage6Slice33EquivalentKeyOffPermutationsConvergeWithinEnvelopeWindow`
  - `TestYM2608Stage6Slice33RapidRekeyPermutationDivergenceIsLocked`
- Locked expected behavior classes explicitly:
  - equivalent final-state permutations converge after the next envelope tick.
  - intentional rapid re-key edge insertion yields deterministic divergence in envelope level despite shared final key-off state.
- No YM2608 core behavior change was required for this slice; ordering vectors passed against current implementation.

Assumptions (temporary placeholders, explicitly non-final):
- Intra-window vectors currently validate deterministic ordering at the scheduler/window level, not cycle-exact bus micro-step interlocks.
- SSG-EG and CSM/special mode coupling to these ordering windows remain deferred.

## Stage 5 Update (Slice 34)
Implemented:
- Added envelope-window split-step conformance vectors around `envRemainder` boundary behavior:
  - `TestYM2608Stage6Slice34SplitStepPreBoundaryWriteConverges`
  - `TestYM2608Stage6Slice34SplitStepPostBoundaryWriteDiverges`
  - `TestYM2608Stage6Slice34SplitStepKeyOnAfterBoundaryDiverges`
- Locked deterministic boundary expectations for:
  - convergence when writes land before the envelope tick boundary in both single-step and split-step paths.
  - divergence when writes are intentionally moved after boundary consumption in split-step paths.
  - key-on timing sensitivity across the same boundary window.
- No YM2608 core code changes were required in this slice; current scheduler/remainder behavior satisfied added vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Boundary vectors currently model scheduler-window semantics (`envRemainder` + envelope tick cadence), not cycle-exact YM internal micro-timing.
- Timer/CSM/SSG-EG coupling to split-step envelope boundaries remains deferred.

## Stage 5 Update (Slice 35)
Implemented:
- Added deterministic mixed-domain boundary vectors combining envelope split-step windows with timer flag/IRQ transitions:
  - `TestYM2608Stage6Slice35PreBoundaryMixedWritesConvergeTimerAndEnvelope`
  - `TestYM2608Stage6Slice35PostBoundaryClearAndEnableDivergenceLocked`
  - `TestYM2608Stage6Slice35KeyPitchKSWritesDoNotResurrectClearedIRQAcrossSplitWindow`
- Locked cross-domain behavior for:
  - convergence when mixed key/pitch/KS + `$29` writes are applied pre-boundary in both single-step and split-step paths.
  - deterministic divergence when timer clear/enable writes are intentionally shifted post-boundary.
  - no stale timerA/IRQ resurrection after explicit clear, even with interleaved key/pitch/KS writes.
- No YM2608 core code-path change was required in this slice; current timer/envelope ordering satisfied the vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Mixed-domain vectors validate deterministic ordering at scheduler/boundary granularity, not full cycle-accurate YM internal arbitration timing.
- ADPCM/SSG/rhythm sideband status interactions with these boundary windows remain deferred.

## Stage 5 Update (Slice 36)
Implemented:
- Added deterministic status-mirror and control-port coupling vectors under split-step pressure:
  - `TestYM2608Stage6Slice36StatusMirrorStableAcrossSplitBoundaryWrites`
  - `TestYM2608Stage6Slice36PortControlPermutationsConvergeInSplitWindow`
  - `TestYM2608Stage6Slice36PostBoundaryControlShiftDivergesButPortLatchesMatch`
- Locked behavior for:
  - `status0/status1` mirror stability across boundary overflow, clear actions, and interleaved port0/port1 writes.
  - convergence for equivalent `$27/$29` + port0/port1 permutations in the same split window.
  - deterministic divergence when control timing is shifted post-boundary while port latches remain equivalent.
- No YM2608 core code changes were required in this slice; current implementation satisfied the vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Mirror/control-port vectors are deterministic scheduler-window conformance checks, not cycle-exact dual-port bus arbitration modeling.
- ADPCM/SSG/rhythm status-bit coupling to these same control windows remains deferred.

## Stage 5 Update (Slice 37)
Implemented:
- Added deterministic BUSY-flag timing vectors under combined port0/port1 + timer boundary stress:
  - `TestYM2608Stage6Slice37BusyMirrorWithTimerBoundaryStress`
  - `TestYM2608Stage6Slice37EquivalentSchedulesConvergeBusyAndStatus`
  - `TestYM2608Stage6Slice37BoundaryShiftedScheduleLocksBusyDivergence`
- Locked behavior for:
  - BUSY mirror stability (`status0/status1`) through mixed port/control writes and timer-boundary events.
  - convergence for equivalent write schedules after BUSY decay windows.
  - deterministic divergence when identical writes are shifted post-boundary, altering elapsed BUSY decay time.
- No YM2608 core code changes were required in this slice; current BUSY/status/timer ordering passed vectors.

Assumptions (temporary placeholders, explicitly non-final):
- BUSY vectors validate host-cycle scheduler behavior and mirror invariants, not die-level command bus arbitration latency.
- Deferred subsystems (SSG/ADPCM/rhythm) may introduce additional BUSY/status interactions not yet modeled.

## Stage 5 Update (Slice 38)
Implemented:
- Added deterministic port-address latch and readback invariance vectors under split-step stress:
  - `TestYM2608Stage6Slice38AddrLatchIntegrityUnderSplitBoundaryStress`
  - `TestYM2608Stage6Slice38ReadbackConsistencyAcrossBusyAndTimerBoundary`
  - `TestYM2608Stage6Slice38BoundaryShiftedAddressSchedulesLockBusyDivergenceWithReadbackInvariant`
- Locked behavior for:
  - `addr0/addr1` latch integrity while timer/IRQ/BUSY events occur in the same split windows.
  - per-port readback consistency under concurrent BUSY and timer-boundary activity.
  - boundary-shifted schedule divergence on BUSY while address/data readback invariants remain stable.
- No YM2608 core code changes were required in this slice; current address-latch/readback behavior satisfied vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Address/readback vectors validate deterministic host-side register-latch semantics and are not cycle-accurate die-level bus contention models.
- Deferred subsystem register banks (SSG/ADPCM/rhythm) may later require additional port-latch contention vectors.

## Stage 5 Update (Slice 39)
Implemented:
- Added deterministic host-interface control invariance vectors for `YMRegControl` (`Enabled`/`Muted`/reset) under split-step timer/BUSY pressure:
  - `TestYM2608Stage6Slice39ControlPermutationsConvergeUnderSplitPressure`
  - `TestYM2608Stage6Slice39BoundaryShiftedControlToggleLocksDivergence`
  - `TestYM2608Stage6Slice39ResetEdgeConvergesStateUnderConcurrentPortTraffic`
- Locked behavior for:
  - convergence of equivalent control+port permutations to the same enabled/unmuted audio and timer/IRQ state.
  - deterministic divergence when control disable is shifted across the timer boundary.
  - reset-edge convergence to a canonical cleared state despite concurrent port traffic and different pre-reset boundary timing.
- No YM2608 core code-path change was required in this slice; current control/reset semantics satisfied vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Control vectors validate deterministic host-interface behavior at scheduler-window granularity, not internal cycle-accurate reset sequencing.
- Deferred subsystem domains (SSG/ADPCM/rhythm) may introduce additional control/reset coupling not yet covered.

## Stage 5 Update (Slice 40)
Implemented:
- Added deterministic timer-load/reload edge vectors under split-step and control pressure:
  - `TestYM2608Stage6Slice40EquivalentTimerAReloadSchedulesConvergeWithControlToggles`
  - `TestYM2608Stage6Slice40BoundaryShiftedTimerAReloadLocksDivergence`
  - `TestYM2608Stage6Slice40BoundaryShiftedTimerBReloadWithControlToggleLocksDivergence`
- Locked behavior for:
  - convergence of equivalent timer-A reload schedules with interleaved `YMRegControl` toggles.
  - deterministic divergence when timer-A reload programming is shifted across the boundary.
  - deterministic divergence when timer-B reload and control toggles are shifted across the boundary.
- No YM2608 core code changes were required in this slice; existing timer reload/control behavior satisfied vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Reload vectors validate deterministic host-window sequencing and not cycle-exact internal timer datapath latencies.
- Deferred subsystem interactions (SSG/ADPCM/rhythm) may later introduce additional reload/counter coupling vectors.

## Stage 5 Update (Slice 41)
Implemented:
- Added deterministic timer programming granularity/order vectors for post-overflow recovery and partial register sequencing:
  - `TestYM2608Stage6Slice41TimerAHiLoPermutationConvergesWithinWindow`
  - `TestYM2608Stage6Slice41BoundaryShiftedTimerAProgrammingLocksDivergence`
  - `TestYM2608Stage6Slice41TimerBPostOverflowRecoveryPermutationsConverge`
- Locked behavior for:
  - convergence of equivalent timer-A high/low write permutations within the same window.
  - deterministic divergence when timer-A high/low programming is shifted after boundary overflow.
  - convergence of timer-B post-overflow recovery permutations (clear-first vs reload-first).
- No YM2608 core code changes were required in this slice; current timer register/counter path satisfied vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Timer programming vectors are deterministic conformance locks for host-write ordering and do not yet model die-level sub-cycle register sampling edges.
- CSM/special-mode timer coupling remains deferred.

## Stage 5 Update (Slice 42)
Implemented:
- Added deterministic host-read status sampling vectors under active timer progression:
  - `TestYM2608Stage6Slice42StatusSamplingBeforeAfterBoundaryTransition`
  - `TestYM2608Stage6Slice42StatusReadOrderInvariantUnderSplitTraffic`
  - `TestYM2608Stage6Slice42StatusSamplingAfterClearRemainsStableWithInterleavedWrites`
- Locked behavior for:
  - stable pre/post-boundary status sampling transitions under active timers.
  - read-order invariance across `status0/status1` under split-window interleaved port traffic.
  - stable cleared status sampling after flag-clear actions without stale resurrection.
- No YM2608 core code changes were required in this slice; current status sampling semantics satisfied vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Status sampling vectors validate host-visible register behavior at scheduler-window granularity, not cycle-exact internal bus read timing.
- Deferred subsystem status domains (SSG/ADPCM/rhythm) may later add additional sampling interactions.

## Stage 5 Update (Slice 43)
Implemented:
- Added deterministic status read-order and control-coupled sampling vectors:
  - `TestYM2608Stage6Slice43StatusReadOrderConvergesAfterBusyDecay`
  - `TestYM2608Stage6Slice43BoundaryShiftedStatusSamplingLocksDivergence`
  - `TestYM2608Stage6Slice43DisableEnableFreezesThenResumesStatusProgression`
- Locked behavior for:
  - read-order convergence after BUSY decay.
  - deterministic divergence of sampled status when reads are shifted across the timer boundary.
  - disable/enable gating semantics on timer progression as observed via status sampling.
- No YM2608 core code changes were required in this slice; existing control/status progression behavior satisfied vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Read-order vectors capture deterministic host-observable semantics and do not model die-level asynchronous read hazards.
- CSM/special mode status side-effects remain deferred.

## Stage 5 Update (Slice 44)
Implemented:
- Added deterministic dual-timer interaction vectors under split-window status sampling pressure:
  - `TestYM2608Stage6Slice44DualTimerClearMaskPermutationsConverge`
  - `TestYM2608Stage6Slice44DualTimerBoundaryShiftedClearLocksDivergence`
  - `TestYM2608Stage6Slice44DualTimerMaskBoundaryShiftLocksDivergence`
- Locked behavior for:
  - convergence of equivalent dual-timer clear/mask schedules.
  - deterministic divergence when dual-timer clear timing is shifted across the boundary.
  - deterministic expectations for IRQ mask timing relative to dual-timer flag sampling.
- No YM2608 core code changes were required in this slice; current dual-timer status behavior satisfied vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Dual-timer vectors validate host-visible ordering semantics and not cycle-exact internal timer cross-coupling.
- CSM/special mode timer interactions remain deferred.

## Stage 5 Update (Slice 45)
Implemented:
- Added deterministic dual-timer read-order and sampling vectors:
  - `TestYM2608Stage6Slice45DualTimerStatusReadOrderInvariantAfterMixedClearMask`
  - `TestYM2608Stage6Slice45DualTimerBoundaryShiftedSamplingLocksDivergence`
  - `TestYM2608Stage6Slice45DualTimerClearOrderConvergesAfterBothClears`
- Locked behavior for:
  - read-order invariance across `status0/status1` after mixed dual-timer clear/mask operations.
  - deterministic divergence between pre/post-boundary sampling.
  - convergence after both timer flags are explicitly cleared regardless clear order.
- No YM2608 core code changes were required in this slice; existing sampling/clear behavior satisfied vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Dual-timer sampling vectors are deterministic host-interface checks, not asynchronous die-level sampling models.
- Deferred subsystem status paths (SSG/ADPCM/rhythm) are not yet included.

## Stage 5 Update (Slice 46)
Implemented:
- Added deterministic dual-timer control/reset gating vectors:
  - `TestYM2608Stage6Slice46DualTimerDisableEnableFreezeResumeConverges`
  - `TestYM2608Stage6Slice46DualTimerResetConvergesFromDivergedStates`
  - `TestYM2608Stage6Slice46DualTimerMaskAfterResetIsDeterministic`
- Locked behavior for:
  - disable/enable freeze-resume semantics with both timers active.
  - reset convergence to canonical cleared dual-timer state from diverged pre-reset states.
  - deterministic post-reset masking semantics with status mirror integrity.
- No YM2608 core code changes were required in this slice; control/reset dual-timer behavior satisfied vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Dual-timer reset vectors capture deterministic host-visible behavior and do not yet assert cycle-accurate reset propagation internals.
- CSM/special mode and deferred subsystem timer-like paths remain out of scope.

## Stage 5 Update (Slice 47)
Implemented:
- Added deterministic dual-timer BUSY coupling vectors:
  - `TestYM2608Stage6Slice47DualTimerDenseWritesKeepBusyMirroredAndDeterministic`
  - `TestYM2608Stage6Slice47EquivalentDualTimerBusySchedulesConverge`
  - `TestYM2608Stage6Slice47BoundaryShiftedDualTimerBusyScheduleLocksDivergence`
- Locked behavior for:
  - BUSY/status mirror co-evolution under dense dual-timer clear/mask/control writes.
  - convergence for equivalent dual-timer BUSY schedules.
  - deterministic BUSY divergence when identical schedules are boundary-shifted while dual-timer flags remain aligned.
- No YM2608 core code changes were required in this slice; current BUSY/status behavior satisfied vectors.

Assumptions (temporary placeholders, explicitly non-final):
- BUSY coupling vectors validate host-visible scheduler semantics, not cycle-exact die bus contention behavior.
- Deferred subsystem paths (SSG/ADPCM/rhythm) may add additional BUSY/status coupling later.

## Stage 5 Update (Slice 48)
Implemented:
- Added deterministic dual-timer port/readback invariance vectors under BUSY coupling:
  - `TestYM2608Stage6Slice48DualTimerPortReadbackInvariantUnderBusyCoupling`
  - `TestYM2608Stage6Slice48EquivalentDualTimerPortSchedulesConverge`
  - `TestYM2608Stage6Slice48BoundaryShiftedDualTimerPortScheduleLocksBusyDivergence`
- Locked behavior for:
  - per-port readback invariance during dual-timer BUSY stress.
  - convergence of equivalent dual-timer port-write schedules.
  - deterministic BUSY divergence with preserved readback invariants under boundary-shifted schedules.
- No YM2608 core code changes were required in this slice; existing readback/latch behavior satisfied vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Port/readback vectors are deterministic host-latch conformance checks and do not model sub-cycle internal readback arbitration.
- Deferred subsystem banks may require additional port coupling vectors later.

## Stage 5 Update (Slice 49)
Implemented:
- Added deterministic dual-timer control/reset coupling vectors:
  - `TestYM2608Stage6Slice49DualTimerControlPermutationsConvergePostBusyDecay`
  - `TestYM2608Stage6Slice49DualTimerBoundaryShiftedDisableLocksDivergence`
  - `TestYM2608Stage6Slice49DualTimerResetAfterDenseBusyTrafficIsDeterministic`
- Locked behavior for:
  - convergence of equivalent dual-timer control permutations after BUSY decay.
  - deterministic divergence when disable timing is shifted across boundary under active dual timers.
  - deterministic reset convergence after dense dual-timer BUSY traffic.
- No YM2608 core code changes were required in this slice; control/reset dual-timer behavior satisfied vectors.

Assumptions (temporary placeholders, explicitly non-final):
- Dual-timer control/reset vectors capture deterministic host-observable behavior and not cycle-accurate reset propagation internals.
- CSM/special mode interactions remain deferred.

## Stage 5 Update (Slice 50)
Implemented:
- Added deterministic FM-audio coupling vectors under dual-timer control pressure:
  - `TestYM2608Stage6Slice50DualTimerAudioPermutationsConvergeAfterBusyDecay`
  - `TestYM2608Stage6Slice50BoundaryShiftedDisableLocksStatusAndAudioDivergence`
  - `TestYM2608Stage6Slice50MutedRunsTimersWhileDisabledFreezesThem`
- Locked behavior for:
  - convergence of equivalent dual-timer/control permutations when sample generation is active.
  - deterministic divergence of both status and rendered sample under boundary-shifted disable scheduling.
  - explicit control-plane contract: muted keeps timer progression active; disabled freezes progression.

Assumptions (temporary placeholders, explicitly non-final):
- Audio coupling vectors validate host-visible scheduling behavior, not cycle-exact mixer pipeline timing.
- FM-only coupling coverage; SSG/ADPCM/rhythm blocks remain deferred.

## Stage 5 Update (Slice 51)
Implemented:
- Added LFO control-path + PMS modulation scaffolding in YM2608 core:
  - decoded `reg 0x22` (`ymRegLFO`) with enable/rate latch path.
  - added deterministic LFO phase stepping (`stepLFO`) with isolated rate/depth tables for future measured replacement.
  - applied PMS depth to per-operator phase increment path (`applyPMS`).
- Added validation vectors:
  - `TestYM2608Stage6Slice51LFOControlRegisterAndPhaseAdvance`
  - `TestYM2608Stage6Slice51PMSAppliesLFODepthToPhaseIncrement`

Assumptions (temporary placeholders, explicitly non-final):
- LFO step periods and PMS depth mapping are deterministic placeholder tables and must be replaced/tuned against stronger references later.
- This slice targets stable control/dataflow architecture first, not final chip-accurate depth curves.

## Stage 5 Update (Slice 52)
Implemented:
- Added AM path wiring for FM operator gain:
  - introduced `operatorGain` path that applies AMS depth only when LFO is enabled and operator `AMON` is set.
  - threaded YM2608 render path through per-operator gain helper so amplitude modulation remains localized and replaceable.
- Added validation vector:
  - `TestYM2608Stage6Slice52AMONAndAMSModulateOperatorGainOnlyWhenLFOEnabled`

Assumptions (temporary placeholders, explicitly non-final):
- AMS depth table is a deterministic scaffold pending stronger calibration.
- AM behavior currently scoped to FM operator amplitude path; non-FM subsystem coupling remains deferred.

## Stage 5 Update (Slice 53)
Implemented:
- Added SSG register/control front-end latch path in YM2608 core (`regs 0x00..0x0D`):
  - tone period decode/latch for channels A/B/C (fine/coarse).
  - noise period, mixer, channel volume, envelope period/shape decode with register-width masking.
  - strict isolation of SSG register state from FM/timer execution path.
- Added validation vectors:
  - `TestYM2608Stage6Slice53SSGRegisterLatchReadbackAndMasking`
  - `TestYM2608Stage6Slice53SSGEquivalentWritePermutationsConverge`
  - `TestYM2608Stage6Slice53SSGWritesDoNotPerturbDualTimerStatus`

Assumptions (temporary placeholders, explicitly non-final):
- Slice is register/control-only; no SSG tone/noise/envelope synthesis is active yet.
- SSG state currently models deterministic host-visible decode/latch behavior and intentionally avoids timing shortcuts that would block later hardware-aligned clocking.

## Stage 5 Update (Slice 54)
Implemented:
- Added SSG runtime clock-domain scaffolding:
  - tone counters + per-channel square output state.
  - noise counter + deterministic LFSR evolution.
  - envelope counter/step/direction/hold runtime state.
- Added vectors:
  - `TestYM2608Stage6Slice54SSGToneCounterTogglesAtProgrammedPeriod`
  - `TestYM2608Stage6Slice54SSGNoiseLFSRAdvancesDeterministically`

Assumptions (temporary placeholders, explicitly non-final):
- Noise LFSR taps are deterministic scaffold values and may require measured tuning.
- Clock divider semantics are hardware-minded but not yet promoted to cycle-accurate reference status.

## Stage 5 Update (Slice 55)
Implemented:
- Added initial SSG synthesis path integration:
  - per-channel level derivation with fixed-volume and envelope-volume modes.
  - mixer gate combination (tone/noise enables) feeding SSG output sample path.
  - SSG sample mixed into existing YM2608 output path with conservative headroom scaling.
- Added vectors:
  - `TestYM2608Stage6Slice55SSGTonePathProducesAudibleSignal`
  - `TestYM2608Stage6Slice55SSGEnvelopeModeDrivesChannelLevel`

Assumptions (temporary placeholders, explicitly non-final):
- SSG output gain curve is a deterministic linear scaffold and may be replaced by measured/known table mapping.
- Envelope shape behavior is a clean deterministic baseline, not final die-verified micro-timing.

## Stage 5 Update (Slice 56)
Implemented:
- Added stabilization/integration vectors for FM+SSG mixed runtime:
  - equivalent FM+SSG schedules converge in status/sample.
  - reset deterministically clears SSG runtime state and silences default post-reset output.
  - `Enabled` gating freezes and resumes SSG clock-state progression deterministically.
- Added vectors:
  - `TestYM2608Stage6Slice56FMAndSSGEquivalentSchedulesConverge`
  - `TestYM2608Stage6Slice56ResetClearsSSGRuntimeAndSilencesOutput`
  - `TestYM2608Stage6Slice56DisabledFreezesSSGClockState`

Assumptions (temporary placeholders, explicitly non-final):
- These stabilization vectors lock host-visible determinism at current abstraction level and do not claim transistor-level parity.
- ADPCM/rhythm sidebands are still deferred and may introduce further integration vectors.

## Stage 5 Update (Slice 57)
Implemented:
- Added ADPCM-A register/control front-end scaffold in YM2608 core:
  - global control, key-on/key-off, total-level, EOS-IRQ mask handling.
  - per-channel level/seed/length/step register latching.
  - deterministic channel start/stop behavior with EOS latch clear-on-start.
- Added validation vector:
  - `TestYM2608Stage6Slice57ADPCMARegisterLatchAndKeyControl`

Assumptions (temporary placeholders, explicitly non-final):
- ADPCM-A register map is currently a deterministic scaffold region isolated in code constants and may be remapped to stricter hardware-verified addresses later.
- Front-end scope is latch/control correctness, not final decode fidelity.

## Stage 5 Update (Slice 58)
Implemented:
- Added ADPCM-A playback/mix scaffold:
  - per-channel stepping and deterministic nibble-stream decode scaffold.
  - EOS latching on sample-end and active-channel mix contribution.
  - total-level and per-channel level gain path into YM2608 mixed output.
- Added validation vector:
  - `TestYM2608Stage6Slice58ADPCMAPlaybackAndMixScaling`

Assumptions (temporary placeholders, explicitly non-final):
- Decode core currently uses deterministic placeholder delta behavior and not final chip-accurate ADPCM tables.
- Mix scaling is intentionally conservative pending broader cross-block calibration.

## Stage 5 Update (Slice 59)
Implemented:
- Added ADPCM-A EOS/IRQ/status interaction path:
  - host-visible EOS status bit latching/clearing.
  - IRQ assertion for unmasked EOS sources.
  - OR behavior with timer-based IRQ sources via shared status/IRQ refresh.
- Added validation vector:
  - `TestYM2608Stage6Slice59ADPCMAEOSAndIRQInteractions`

Assumptions (temporary placeholders, explicitly non-final):
- EOS/IRQ behavior is locked at host-visible determinism granularity; sub-cycle status race behavior remains out of scope for now.
- BRDY/ZERO and deeper ADPCM-B/rhythm status coupling remain deferred.

## Stage 5 Update (Slice 60)
Implemented:
- Added ADPCM-B register/control front-end scaffold (port1 mapped):
  - control, key-on/key-off, total-level, IRQ-mask, seed/length/step latching.
  - deterministic playback start/stop state transitions.
- Added validation vector:
  - `TestYM2608Stage6Slice60ADPCMBRegisterLatchAndControl`

Assumptions (temporary placeholders, explicitly non-final):
- ADPCM-B register map is currently a deterministic scaffold region and may be remapped once stronger conformance anchors are available.
- Front-end scope is state/control correctness, not final decode fidelity.

## Stage 5 Update (Slice 61)
Implemented:
- Added ADPCM-B playback/mix scaffold:
  - deterministic stepping/decode progression from seed/length/step settings.
  - total-level attenuation path integrated into YM2608 mixer.
- Added validation vector:
  - `TestYM2608Stage6Slice61ADPCMBPlaybackAndMixScaling`

Assumptions (temporary placeholders, explicitly non-final):
- Decode core uses deterministic placeholder delta behavior pending stronger references.
- Mix scaling is conservative and subject to later whole-chip calibration.

## Stage 5 Update (Slice 62)
Implemented:
- Added ADPCM-B status/IRQ interaction path:
  - BRDY and ZERO status-bit latching/clearing.
  - IRQ assertion for unmasked BRDY/ZERO sources.
  - OR-preserved coexistence with timer-sourced IRQ status.
- Added validation vector:
  - `TestYM2608Stage6Slice62ADPCMBStatusAndIRQInteractions`

Assumptions (temporary placeholders, explicitly non-final):
- BRDY/ZERO behavior is currently locked at host-visible deterministic granularity.
- Deeper sub-cycle race timing and final silicon-accurate status choreography remain deferred.

## Stage 5 Update (Slice 63)
Implemented:
- Added rhythm block front-end scaffold (port1 mapped):
  - global control + total-level + key-on/key-off trigger path.
  - per-voice level/seed/length/step latching.
  - deterministic trigger semantics with per-voice runtime start/stop.
- Added validation vector:
  - `TestYM2608Stage6Slice63RhythmRegisterLatchAndControl`

Assumptions (temporary placeholders, explicitly non-final):
- Rhythm register map is deterministic scaffolded address space and may be remapped later if stricter references require it.
- Front-end scope is deterministic control semantics, not final chip-accurate sample set behavior.

## Stage 5 Update (Slice 64)
Implemented:
- Added rhythm playback/mix scaffold:
  - deterministic per-voice stepping/decode progression.
  - per-voice level and global total-level scaling integrated into YM2608 mixer.
- Added validation vector:
  - `TestYM2608Stage6Slice64RhythmPlaybackAndMixScaling`

Assumptions (temporary placeholders, explicitly non-final):
- Decode and gain curves use deterministic placeholders pending stronger calibration anchors.
- Mixer headroom is conservative and may be tuned in final whole-chip balance pass.

## Stage 5 Update (Slice 65)
Implemented:
- Added rhythm integration stabilization vectors:
  - equivalent mixed scheduling convergence with FM/timer pressure.
  - deterministic reset clearing of rhythm runtime state.
  - enable-gating freeze/resume behavior for rhythm progression.
- Added validation vectors:
  - `TestYM2608Stage6Slice65FMSSGADPCMARhythmConvergence`
  - `TestYM2608Stage6Slice65ResetClearsRhythmAndSilencesOutput`
  - `TestYM2608Stage6Slice65DisabledFreezesRhythmProgression`

Assumptions (temporary placeholders, explicitly non-final):
- Integration vectors lock host-visible deterministic behavior and do not yet claim full chip-accurate rhythm micro-timing.
- Final balance/timing refinement remains scheduled in later hardening slices.

## Stage 5 Update (Slice 66)
Implemented:
- Hardened cross-source status/IRQ orchestration:
  - status IRQ refresh now recomputes EOS/BRDY/ZERO from source state before evaluating IRQ.
  - non-timer status bits are now source-derived instead of stale latch-driven.
- Added validation vectors:
  - `TestYM2608Stage6Slice66CrossSourceIRQClearSemantics`
  - `TestYM2608Stage6Slice66AuxStatusRecomputedFromSourceState`

Assumptions (temporary placeholders, explicitly non-final):
- Cross-source behavior is locked at host-visible register semantics; sub-cycle bus race timing remains deferred.

## Stage 5 Update (Slice 67)
Implemented:
- Refined BUSY timing progression to follow YM-cycle conversion path (host cycles -> YM cycles):
  - BUSY now decays in YM clock domain, including fractional host/YM remainder handling.
- Added validation vectors:
  - `TestYM2608Stage6Slice67BusyDurationUsesYMClockRatio`
  - `TestYM2608Stage6Slice67BusyConvergesAcrossEquivalentFractionalSchedules`

Assumptions (temporary placeholders, explicitly non-final):
- BUSY duration uses deterministic 32-cycle placeholder pending tighter silicon-calibrated timing.

## Stage 5 Update (Slice 68)
Implemented:
- Stabilized APU-facing YM2608 runtime API:
  - `ConfigureYM2608Clocks(masterHz, hostHz)`
  - `YM2608RuntimeState() (YMRuntimeState, bool)`
  - runtime snapshot now reports SSG activity via channel-level active-state logic rather than mixer bits alone.
- Added validation vectors:
  - `TestAPUStage6Slice68ClockConfigProxyAndRuntimeStateSnapshot`
  - `TestAPUStage6Slice68RuntimeStateUnavailableWithoutCore`

Assumptions (temporary placeholders, explicitly non-final):
- Runtime-state projection targets host integration stability first; low-level analog-domain conformance is still deferred to later calibration slices.

## Stage 5 Update (Slice 69)
Implemented:
- Added whole-chip readiness hardening vectors spanning FM + SSG + ADPCM-A + ADPCM-B + rhythm under timer-IRQ pressure.
- Added APU-facing runtime-state conformance vector to verify host MMIO status and runtime snapshot consistency for enabled YM2608 sub-blocks.
- Added validation vectors:
  - `TestYM2608Stage6Slice69WholeChipConvergesUnderTimerIRQPressure`
  - `TestAPUStage6Slice69RuntimeStateMirrorsWholeChipStatusAndEnablement`

Assumptions (temporary placeholders, explicitly non-final):
- Integration vectors lock deterministic host-visible behavior and cross-block coherence; they do not yet claim cycle-exact analog-domain conformance.

## Stage 5 Update (Slice 70)
Implemented:
- Added deterministic APU/CoreLX mixed-access convergence vector under active multi-block playback:
  - interleaved register writes, data reads, and status sampling with equivalent split schedules.
  - locked convergence for final audio sample, mirrored status bytes, and runtime-state projection.
- Added runtime-state boundary invariants with all major YM2608 blocks enabled:
  - mute preserves enabled progression but forces silent output.
  - disable freezes pending timer/IRQ progression.
  - re-enable resumes pending progression and latches timer/IRQ deterministically.
  - reset clears status/sub-block runtime enables and returns to silent baseline.
- Added validation vectors:
  - `TestAPUStage6Slice70MixedAccessPatternConvergenceUnderPlayback`
  - `TestAPUStage6Slice70RuntimeStateDisableMuteResetBoundariesAllBlocks`

Assumptions (temporary placeholders, explicitly non-final):
- Access-pattern vectors target deterministic host-visible scheduler-level behavior; sub-cycle bus contention nuances remain deferred.

## Stage 5 Update (Slice 71)
Implemented:
- Added long-horizon multi-frame convergence vector:
  - equivalent cycle budgets with different split schedules under full multi-block activity.
  - verifies audio/status drift stability via deterministic rolling hashes and runtime/MMIO status parity.
- Added ADPCM/rhythm completion lifecycle vector under concurrent timer pressure:
  - completion, status clear, restart, explicit stop, and second restart loops for ADPCM-A, ADPCM-B, and rhythm.
  - verifies lifecycle repeatability while timerA/IRQ pressure remains active.
- Added validation vectors:
  - `TestAPUStage6Slice71LongHorizonDriftConvergence`
  - `TestAPUStage6Slice71ADPCMRhythmLifecycleUnderTimerPressure`

Assumptions (temporary placeholders, explicitly non-final):
- Drift checks target deterministic scheduler-level integration behavior and do not yet assert chip-internal sub-cycle contention conformance.

## Stage 5 Update (Slice 72)
Implemented:
- Added sustained playback continuity smoke vector through repeated control churn:
  - enable/mute/disable/reset/IRQ-mask toggles during active multi-block playback.
  - verifies status-mirror parity, runtime/MMIO status parity, silence invariants when muted/disabled, and resumed audibility when enabled+unmuted.
- Added churn-schedule convergence vector:
  - equivalent control events with different cycle partitioning must converge final audio hash, mirrored status bytes, and runtime projection.
- Added validation vectors:
  - `TestAPUStage6Slice72SustainedPlaybackContinuityAcrossControlChurn`
  - `TestAPUStage6Slice72ControlChurnSchedulesConverge`

Assumptions (temporary placeholders, explicitly non-final):
- Churn vectors lock scheduler-visible stability and determinism; sub-cycle bus arbitration and analog response transients remain outside current scope.

## Stage 5 Update (Slice 73)
Implemented:
- Executed final “usable baseline” closure checklist across YM2608 runtime vectors.
- Confirmed late-stage hardening vectors (Slices 66-72) and full audio/policy gates in one closure pass.

Acceptance Checklist Execution (closure run):
- `go test ./internal/apu -run 'Slice(66|67|68|69|70|71|72)' -count=1` passed
- `go test ./internal/apu -run 'TestAPUStage6Slice(69|70|71|72)' -count=1` passed
- `go test ./internal/apu -count=1` passed
- `make ci-audio` passed
- `make check-ym2608-policy` passed

Usable-Baseline Gate Readout:
- ACC-AUDIO-1 (MMIO behavior): pass at deterministic host-visible conformance level.
- ACC-AUDIO-2 (Timer/IRQ/Status): pass at deterministic host-visible conformance level.
- ACC-AUDIO-5 (V1 target identity): pass (YM2608 canonical target across runtime/spec notes).

Non-Blocking Conformance Debt (deferred to deep calibration phase):
- BUSY timing remains deterministic placeholder (`32` YM cycles), not yet silicon-calibrated wait-state model.
- FM operator/timing internals still use calibrated scaffolding for envelope-rate curves, keycode/detune coupling, and algorithm graph fidelity.
- ADPCM-A/ADPCM-B decode behavior remains deterministic placeholder modeling, not final bit-accurate silicon decode conformance.
- Rhythm block currently uses deterministic synthetic playback model, not final sample/mix behavior from hardware-verified references.
- Mixer calibration (absolute gain law, headroom, clipping character, panning law) is stable but not yet hardware-calibrated.
- ACC-AUDIO-3 (audio reference thresholds) still needs stronger measured/hardware-grade reference vectors.
- ACC-AUDIO-4 wording in `V1_ACCEPTANCE.md` references legacy+YM mixed playback; runtime is now YM2608-only and this acceptance text should be revised.

## Next Safe Step (Stage 6, Calibration Track 1)
- Begin deep conformance calibration without destabilizing usable baseline:
  - introduce stronger external references (measured when available, multi-emulator cross-check otherwise),
  - tune FM/ADPCM/rhythm timing and gain laws behind deterministic fixture gates,
  - keep current Stage 5 vectors as regression locks during calibration.
