# Nitro-Core-DX V1 Risk Register

Status: Active  
Last Updated: July 22, 2026

Scoring:
- Severity: High / Medium / Low
- Likelihood: High / Medium / Low

## Risk Register

| ID | Area | Risk | Severity | Likelihood | Mitigation | Owner | Status |
|---|---|---|---|---|---|---|---|
| R-001 | Editor | Native editor engine complexity (input/render/selection/perf) delays IDE milestone | High | Medium | Stabilize single-ownership editor model, keep interaction test matrix active, and gate merges on editor latency/selection correctness checks | Dev Kit | Open |
| R-002 | Tooling | Sprite/Tilemap/Sound tools produce incompatible assets or stale language snippets | High | Medium | Keep shared asset contracts under test; compile every generated snippet/template; require manifest/source round-trip tests for each tool | CoreLX + Tools | Mitigating |
| R-003 | Audio | YM2608 conformance/reference scope expands unexpectedly after runtime plumbing is already usable | High | Medium | Separate runtime readiness from conformance quality gates; keep Sound Studio MVP limited to import/preview/export; freeze acceptance references before release-candidate work | Audio | Mitigating |
| R-004 | Performance | New tools/editor regress Build+Run responsiveness | Medium | Medium | Add performance baselines and CI perf checks for Dev Kit workflows | Dev Kit | Open |
| R-005 | Stability | Session persistence introduces corruption/lost-work scenarios | High | Medium | Add atomic settings writes, recovery paths, and crash-restart tests | Dev Kit | Open |
| R-006 | Docs | Manual diverges from real game code and APIs | High | Medium | Enforce snippet-run checks in CI and map sections to live source files | Docs | Open |
| R-007 | Release | Linux/Windows packaging drift creates last-minute failures | High | Medium | Maintain release matrix CI with artifact smoke tests on every RC | Release | Open |
| R-008 | Scope | Uncontrolled feature additions push out V1 target | High | High | Enforce `v1-scope-change-approved` rule and explicit trade-offs | PM/Leads | Open |
| R-009 | Game | NitroPackInDemo proof-of-concept scope takes longer than planned | High | Medium | Stage content milestones with playable checkpoints and keep the ROM-first demo limited to the agreed end-to-end slice before adding polish or extra scenes | Game | Open |
| R-010 | Debugger | CPU-step semantics conflict with frame-synchronized systems | Medium | Medium | Clearly document step modes and add deterministic debugger tests | Emulator | Open |
| R-011 | Dev Kit Windowing | UI/layout refactors accidentally disable native maximize/minimize behavior on some desktops | Medium | Medium | Treat native window behavior as release gate (`ACC-DK-2`), avoid fixed-size/decor-hint overrides, and require Linux+Windows smoke checks for any window-flag changes | Dev Kit | Open |
| R-012 | Sound Studio | Sound Studio grows into a full tracker/composer before the V1 import/preview/export workflow ships | High | Medium | Define MVP as VGM/VGZ import, `.ncdxmusic` inspection/export, emulator-backed preview, and source/manifest insertion; defer full composition unless explicitly approved | Tools + Audio | Open |
| R-013 | Large CoreLX Demo | NitroPackInDemo CoreLX rebuild exposes codegen/banking edge cases that block full-suite validation | High | Medium | Treat the demo as the compiler stress target; keep ROM-first demo runnable; gate "CoreLX complete" on the large-program failure being resolved | CoreLX | Open |

## Active Watchlist

- YM2608 conformance/reference gate (R-003)
- Sound Studio MVP scope control (R-012)
- NitroPackInDemo proof-of-concept scope (R-009)
- NitroPackInDemo CoreLX large-program stabilization (R-013)
- Editor migration delivery risk (R-001)

## Escalation Rule

Escalate immediately when:

1. Any High severity risk lacks active mitigation owner
2. Any risk blocks two or more charter IDs
3. A mitigation requires scope change approval
