# Nitro-Core-DX V1 Risk Register

Status: Active  
Last Updated: March 6, 2026

Scoring:
- Severity: High / Medium / Low
- Likelihood: High / Medium / Low

## Risk Register

| ID | Area | Risk | Severity | Likelihood | Mitigation | Owner | Status |
|---|---|---|---|---|---|---|---|
| R-001 | Editor | Native editor engine complexity (input/render/selection/perf) delays IDE milestone | High | Medium | Stabilize single-ownership editor model, keep interaction test matrix active, and gate merges on editor latency/selection correctness checks | Dev Kit | Open |
| R-002 | Tooling | Sprite/Tilemap/Sound tools produce incompatible assets | High | Medium | Define shared asset contracts + golden round-trip tests early | CoreLX + Tools | Open |
| R-003 | Audio | YM2608 migration + parity profile expands unexpectedly | High | High | Freeze YM2608 conformance profile early; constrain sequence (Sprite/Dev Kit -> Sound Studio -> YM2608); enforce review gates before enabling new chip behavior | Audio | Open |
| R-004 | Performance | New tools/editor regress Build+Run responsiveness | Medium | Medium | Add performance baselines and CI perf checks for Dev Kit workflows | Dev Kit | Open |
| R-005 | Stability | Session persistence introduces corruption/lost-work scenarios | High | Medium | Add atomic settings writes, recovery paths, and crash-restart tests | Dev Kit | Open |
| R-006 | Docs | Manual diverges from real game code and APIs | High | Medium | Enforce snippet-run checks in CI and map sections to live source files | Docs | Open |
| R-007 | Release | Linux/Windows packaging drift creates last-minute failures | High | Medium | Maintain release matrix CI with artifact smoke tests on every RC | Release | Open |
| R-008 | Scope | Uncontrolled feature additions push out V1 target | High | High | Enforce `v1-scope-change-approved` rule and explicit trade-offs | PM/Leads | Open |
| R-009 | Game | Galaxy Force full concept takes longer than planned | High | Medium | Stage content milestones with playable checkpoints and integration tests | Game | Open |
| R-010 | Debugger | CPU-step semantics conflict with frame-synchronized systems | Medium | Medium | Clearly document step modes and add deterministic debugger tests | Emulator | Open |
| R-011 | Dev Kit Windowing | UI/layout refactors accidentally disable native maximize/minimize behavior on some desktops | Medium | Medium | Treat native window behavior as release gate (`ACC-DK-2`), avoid fixed-size/decor-hint overrides, and require Linux+Windows smoke checks for any window-flag changes | Dev Kit | Open |

## Active Watchlist

- YM2608 migration/parity gate (R-003)
- Galaxy Force full-concept scope (R-009)
- Editor migration delivery risk (R-001)

## Escalation Rule

Escalate immediately when:

1. Any High severity risk lacks active mitigation owner
2. Any risk blocks two or more charter IDs
3. A mitigation requires scope change approval
