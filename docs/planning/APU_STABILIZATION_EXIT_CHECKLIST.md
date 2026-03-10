# APU Stabilization Exit Checklist (Pre-CPU Amped Work)

**Date:** March 9, 2026  
**Purpose:** Define the minimum “audio is stable enough” bar before broad CPU ISA changes.

## Exit Criteria

All items below should pass in one clean run before moving to larger CPU surface-area changes.

1. Backend selection behavior is deterministic and documented
- `NCDX_YM_BACKEND=auto` prefers YMFM when built with `ymfm_cgo`, else falls back cleanly.
- `NCDX_YM_BACKEND=ymfm` warns/falls back when YMFM is unavailable.
- `NCDX_YM_BACKEND=legacy` forces in-tree fallback path.

2. APU/YM2608 tests pass for the active build profile
- Core APU package tests pass.
- Emulator integration tests touching APU path pass.

3. Golden/reference audio guardrail is reproducible
- A fixed ROM+frame-count capture can be regenerated.
- SHA/hash or metrics comparison is stable run-to-run.

4. Runtime smoke behavior is clean
- Demo song ROM and gameplay-BGM ROM run without lockups.
- No runaway frame pacing regressions (audio does not “advance multiple song frames per video frame”).

## Command Checklist

Run in repo root.

### A) CPU/APU baseline compile+tests
```bash
go test ./internal/cpu ./internal/apu ./internal/emulator
```

### B) YMFM-backed tests (when YMFM build tags are available)
```bash
NCDX_YM_BACKEND=ymfm go test -tags ymfm_cgo ./internal/apu ./internal/emulator
```

### C) Build and capture deterministic YM2608 song output
```bash
go run -tags testrom_tools ./test/roms/build_ym2608_demo_song.go \
  -in Resources/Demo.vgz \
  -out roms/ym2608_demo_song.rom \
  -frames-per-bank 70

go run ./cmd/rom_audio_capture \
  -rom roms/ym2608_demo_song.rom \
  -out /tmp/ym2608_demo_song_capture.wav \
  -frames 1800 \
  -audio-backend ymfm
```

### D) Optional reference compare against known render
```bash
go run ./cmd/wav_compare \
  -ref Resources/Demo.wav \
  -got /tmp/ym2608_demo_song_capture.wav \
  -seconds 20
```

### E) Gameplay + BGM smoke (manual)
```bash
go run -tags testrom_tools ./test/roms/build_pong_ym2608.go \
  -in Resources/Demo.vgz \
  -out roms/pong_ym2608_demo.rom

go run -tags ymfm_cgo,no_sdl_ttf ./cmd/emulator \
  -rom roms/pong_ym2608_demo.rom \
  -audio-backend ymfm
```

Manual check:
- Press `START`.
- Confirm music starts at match start.
- Confirm paddle up/down, opponent, ball, and scoring are active.
- Confirm no rapid/garbled song progression.

## Evidence to Save

- Test command logs (A + B).
- Capture hash and file path from command C.
- Optional compare output from D.
- One short note for command E smoke result.

## Decision Rule

- If all required checks pass: proceed to CPU Amped Tier 1 implementation.
- If any required check fails: fix APU/runtime first; do not expand CPU ISA yet.
