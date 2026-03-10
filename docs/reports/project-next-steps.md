# Nitro-Core-DX Project Next Steps

**Date:** March 9, 2026  
**Purpose:** Short checklist for immediate review, high-priority cleanup approval, architecture decisions, and stabilization before new CPU/APU/PPU feature work.

---

## Immediate Review Items (For You)

- [ ] **Read** [docs/reports/project-state-review.md](project-state-review.md): executive summary, PPU vs SNES/Genesis section, and “What is meeting / falling short.”
- [ ] **Read** [docs/reports/project-cleanup-plan.md](project-cleanup-plan.md): safe vs risky cleanup tables and suggested order of operations.
- [x] **Resolved** YM2608 local policy tooling drift: Makefile targets are now guarded and docs treat manifest tooling as optional when absent locally.
- [x] **Resolved** SYSTEM_MANUAL Matrix Mode wording drift: active docs now describe Matrix Mode as implemented.
- [x] **Resolved** PPU DMA fix: Read8 DMA_LENGTH now uses 0x8066/0x8067 and test coverage exists.

---

## Highest-Priority Cleanup/Refactor to Approve

1. **Documentation baseline:** Keep active YM2608/APU/PPU docs aligned with current guarded tooling/runtime behavior.
2. **Compatibility cleanup:** Decide deprecation timeline for legacy PPU/APU compatibility surfaces (`RenderFrame()`, `executeDMA()`, legacy matrix shim, legacy float APU phase fields).
3. **FPGA parity planning:** Document and prioritize the remaining FPGA-vs-emulator gaps before new major feature work.

The original high-priority cleanup items from this report have already been executed.

---

## Architecture Questions Needing Decisions

1. **PPU legacy matrix:** Keep indefinitely for compatibility, or set a release-bound deprecation/removal plan now that BG0 reconciliation is explicit?
2. **PPU legacy frame/DMA APIs:** Keep `RenderFrame()` / `executeDMA()` as compatibility-only shims, or schedule hard removal after one deprecation window?
3. **FPGA PPU:** Confirm target is “four matrix engines + configurable tilemap base” to match Go; document as the intended FPGA roadmap before RTL work continues.
4. **Savestates:** PPU legacy matrix and APU float phase compatibility are now known savestate concerns. Decide whether to preserve indefinitely or version the save format before removal.

---

## What to Stabilize Before Adding New CPU/APU/PPU Features

- **Docs vs code:** Complete the safe documentation updates above so new work doesn’t rely on stale SYSTEM_MANUAL or spec gaps.
- **Single source of memory truth:** Done. `Bus` is the canonical emulator memory model and `MemorySystem` has been removed.
- **PPU register map:** Apply DMA_LENGTH read fix and document 0x803E–0x8040; then treat PPU register map as stable for new features.
- **YM2608 tooling:** Done for local flows. Guarded make targets now treat absent manifest tooling as optional instead of broken.
- **Test baseline:** Keep running `test-fast` / `test-full` / `test-emulator` after any cleanup; no new feature work that skips these.

---

## Optional Follow-Up

- Clarify in HARDWARE_FEATURES_STATUS whether HDMA scroll/matrix is “complete” or “partial” (code suggests full per-layer scroll+matrix HDMA is implemented).
- When touching APU: plan removal of deprecated float phase fields only alongside a deliberate savestate migration/version step.
- When touching FPGA: add a short “FPGA vs emulator” subsection in the FPGA or hardware spec listing PPU gaps (four matrix engines, tilemap layout) until RTL is aligned.

---

*End of project-next-steps.*
