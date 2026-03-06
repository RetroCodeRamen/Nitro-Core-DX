# Nitro-Core-DX v0.1.8

## What Changed (Plain-English)

This release focuses on making the Dev Kit feel more reliable and more "real tool" than prototype.

### 1) The Code Editor Is More Stable

We started moving to a native editor foundation so typing and editing behavior is handled by one owner instead of a fragile hybrid path.

What that means for you:
- better input consistency while typing
- fewer weird cursor/selection edge cases
- cleaner path for future editor upgrades (find/replace, symbol tools, richer diagnostics)

### 2) Sprite Lab Got Better for Real Work

Sprite Lab picked up practical workflow upgrades:

- **Wrap Shift Controls** for sprites (Up/Down/Left/Right)
  - move all pixels one step and wrap at edges
  - useful for quick animation/frame tweaks
- **Palette RGB555 Slider + Full Hex Flow**
  - easier color dialing with slider control
  - hex value stays visible and synchronized
- **Preview Aspect Fix**
  - sprite preview now keeps correct proportions when resizing windows
  - no more stretched/wide preview distortion

### 3) Native Window Behavior Was Locked Down

We cleaned up maximize/minimize behavior and added guardrails so window management regressions are less likely to come back.

- system title-bar maximize/minimize remains the expected behavior
- fullscreen remains distinct from maximize
- guard test added to keep platform-specific hinting constrained

### 4) V1 Plan Direction Updated: YM2608

The V1 audio target is now **YM2608**.

This does **not** mean YM2608 is fully implemented in this release. It means the release plan has been updated so the final V1 sound target is clear.

Execution order is now explicitly gated:
1. finish Sprite Lab + Dev Kit stabilization
2. complete required tilemap flow
3. start Sound Studio
4. then begin YM2608 implementation

## Why v0.1.8 Matters

v0.1.8 is less about flashy new subsystems and more about maturity:
- stronger editor behavior
- stronger art workflow
- cleaner planning discipline for V1
- fewer regressions in core UX

## Downloads

- **Linux:** `nitrocoredx-v0.1.8-linux-amd64.tar.gz`
- **Windows:** `nitrocoredx-v0.1.8-windows-amd64.zip`
