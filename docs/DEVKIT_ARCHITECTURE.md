# Dev Kit Architecture (Frontend/Backend Boundary)

Status: Active (pre-alpha)
Last aligned: 2026-07-22

## Goal

Build a proper development environment wrapper around Nitro-Core-DX without changing emulator core behavior or hardware/FPGA-oriented semantics.

## Rule

- `internal/*` emulator/compiler/hardware logic is the source of truth.
- Dev Kit UI layers may be rewritten freely.
- Frontend changes must not alter observable emulator behavior for a given ROM/input sequence.
- Frontend tools (Sprite Lab/Tilemap Lab/editors) may propose source edits, but compiler outputs remain authoritative for final build mapping/artifacts.

## Current split

### Backend (UI-agnostic)

- `internal/corelx/*`
  - CoreLX compiler pipeline, diagnostics, manifest, bundle outputs
  - Project asset manifest ingestion (`corelx.assets.json`) in compile-service path
- `internal/emulator/*`
  - Emulator core (CPU/PPU/APU/Bus/Input timing)
- `internal/devkit/service.go`
  - Dev Kit wrapper service for frontend use
  - Responsibilities:
    - Build source (`BuildSource`) and emit artifacts (ROM/manifest/diagnostics/bundle)
    - Own embedded emulator session lifecycle (`LoadROMBytes`, `Shutdown`)
    - Thread-safe emulator control (`ResetEmulator`, `TogglePause`, `SetInputButtons`, `RunFrame`)
    - Thread-safe snapshots (`Snapshot`, `FramebufferCopy`, `AudioSamplesFixedCopy`)

### Frontend (replaceable)

- `cmd/corelx_devkit`
  - Current Fyne-based Dev Kit frontend
  - Responsibilities:
    - Layout/panes/tabs
    - Editor UI
    - Diagnostics filtering/search UI
    - Input routing policy (when keyboard drives editor vs emulator)
    - Framebuffer rendering/presentation
    - Host audio output queueing

## Current Tool Status (2026-07-22)

- **Code/Build/Run:** active and usable. The Dev Kit can compile, show
  diagnostics, load ROM bytes into the embedded emulator, and queue emulator
  audio frames to SDL.
- **Sprite Lab:** strongest tool in the suite. Editing, palette handling,
  import/export, undo/redo, project insertion, and manifest flows exist and are
  backed by focused tests. It should now treat native larger hardware sprites
  as first-class output targets rather than forcing composite OAM workarounds.
- **Tilemap Lab:** usable, but not release-complete. It needs stronger
  manifest-backed asset handling, generated-source compile tests, map-size
  alignment, and emulator-visible round-trip acceptance tests.
- **Image/Plane Import:** CLI path exists outside the Dev Kit; integrated UI is
  still missing.
- **Sound Studio:** placeholder tab only. Runtime support exists through
  `.ncdxmusic`, YM2608 playback, and Dev Kit audio queueing; the missing work is
  import/inspect/preview/export UI.
- **Debugger:** backend/frame-control pieces exist, but V1 debugger UX still
  needs pause/resume, frame-step, CPU-step, register/PC panels, and memory
  watch workflow.

## Why this split matters

This allows future frontend evolution (native Fyne editor improvements or alternate UI shell) without rewriting:
- compiler backend
- emulator backend
- FPGA-aligned hardware logic
- ROM packaging/diagnostics contracts

## Next migration steps

1. Continue moving non-UI workflow logic out of `cmd/corelx_devkit` into `internal/devkit`.
2. Define a stable Dev Kit backend API contract (Go interface and/or JSON-RPC/IPC for alternate frontends).
3. Add tool-service helpers for asset import flows that should not live in UI
   widgets: image import, music import, project manifest update, and preview ROM
   construction.
4. Build a new frontend against `internal/devkit` while keeping the Fyne frontend as a reference/fallback during transition.

## Sound Studio Integration Direction

The Sound Studio MVP should use existing backend/runtime pieces instead of
creating a separate audio implementation:

- Use `internal/ymstream` for VGM/VGZ read, parse, encode, decode, and stream
  inspection.
- Write `.ncdxmusic` files into the current project directory.
- Update source/project manifest with `asset Name: music "file.ncdxmusic"`.
- Generate current-language snippets for `music.play`, `music.play_loop`,
  `music.play_jingle`, `music.stop`, `music.set_volume`, and `music.fade_to`.
- Preview through a temporary CoreLX project loaded into the embedded emulator
  so preview audio follows the same YM2608 path as Build+Run.
- Reuse the existing SDL audio queue owned by the Dev Kit frontend.

## Invariants

- Same ROM + same input sequence must produce the same emulator behavior regardless of Dev Kit frontend.
- UI timing/presentation may differ; emulator core timing semantics must not.
- Build state in the UI (`Draft` / `Validating...` / `Validated` / `Error`) must reflect compiler service results, not frontend-only assumptions.
- Window management must remain OS-native by default:
  - maximize/restore/minimize come from system title bar behavior
  - fullscreen toggle (for example, `F11`) is distinct from maximized window mode
  - Dev Kit UI/layout changes must not disable native window decorations or turn the app into fixed-size mode
- Any change touching window flags, decoration hints, or platform-specific window APIs requires smoke validation on Linux and Windows before merge.
