# Dev Kit Architecture (Frontend/Backend Boundary)

Status: Active (pre-alpha)

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

## Why this split matters

This allows future frontend evolution (native Fyne editor improvements or alternate UI shell) without rewriting:
- compiler backend
- emulator backend
- FPGA-aligned hardware logic
- ROM packaging/diagnostics contracts

## Next migration steps

1. Continue moving non-UI workflow logic out of `cmd/corelx_devkit` into `internal/devkit`.
2. Define a stable Dev Kit backend API contract (Go interface and/or JSON-RPC/IPC for alternate frontends).
3. Build a new frontend against `internal/devkit` while keeping the Fyne frontend as a reference/fallback during transition.

## Invariants

- Same ROM + same input sequence must produce the same emulator behavior regardless of Dev Kit frontend.
- UI timing/presentation may differ; emulator core timing semantics must not.
- Build state in the UI (`Draft` / `Validating...` / `Validated` / `Error`) must reflect compiler service results, not frontend-only assumptions.
- Window management must remain OS-native by default:
  - maximize/restore/minimize come from system title bar behavior
  - fullscreen toggle (for example, `F11`) is distinct from maximized window mode
  - Dev Kit UI/layout changes must not disable native window decorations or turn the app into fixed-size mode
- Any change touching window flags, decoration hints, or platform-specific window APIs requires smoke validation on Linux and Windows before merge.
