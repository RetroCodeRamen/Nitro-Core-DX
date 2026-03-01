# Nitro-Core-DX V1 Charter

Status: Active (V1 source of truth)  
Last Updated: February 28, 2026

This charter is the canonical scope contract for Nitro-Core-DX V1.0.  
Other planning files may contain ideas/history; this document defines what can block V1 release.

## 1. Product Goal

Ship Nitro-Core-DX as a product-complete desktop SDK (Linux + Windows first-class) with:

- Stable integrated CoreLX Dev Kit workflow
- Complete V1 tool suite MVP (Sprite, Tilemap, Sound authoring)
- Debugger stepping (pause/resume + frame-step + CPU-step)
- FM audio at the approved behavioral parity profile
- Built-in documentation with runnable snippets
- Galaxy Force flagship game and manual integration

## 2. In Scope (Release-Blocking)

### V1-PLAT: Platform + Release
- V1-PLAT-1 Linux release artifacts and smoke validation
- V1-PLAT-2 Windows release artifacts and smoke validation
- V1-PLAT-3 Strict release gates enforced in CI and release checklist

### V1-DK: Dev Kit Core Experience
- V1-DK-1 Build / Build+Run stable in integrated app ✅
- V1-DK-2 Session persistence (last dirs/files/view/capture/recent files) ✅
- V1-DK-3 Embedded emulator lifecycle hardened ✅
- V1-DK-4 Help center + programming docs in-app ✅

### V1-EDITOR: IDE-grade CoreLX editing
- V1-EDITOR-1 Monaco/webview integration
- V1-EDITOR-2 CoreLX syntax highlighting
- V1-EDITOR-3 Diagnostics squiggles + panel sync + jump-to-location
- V1-EDITOR-4 Editor essentials: find/replace, go-to-line, basic symbol navigation
- V1-EDITOR-5 Autosave and crash recovery journal ✅

### V1-DBG: Debugger
- V1-DBG-1 Pause/resume UX
- V1-DBG-2 Frame-step deterministic behavior
- V1-DBG-3 CPU instruction-step deterministic behavior
- V1-DBG-4 Register/PC state snapshots exposed and visible in UI

### V1-TOOLS: Tool suite MVP
- V1-TOOLS-1 Sprite Editor round-trip asset flow + live preview ✅
- V1-TOOLS-2 Tilemap Designer round-trip asset flow + live preview
- V1-TOOLS-3 Sound Studio MVP with playback preview and export
- V1-TOOLS-4 Tool outputs consumed by build pipeline without manual edits ✅ (Sprite Lab)

### V1-CORELX: Compiler/toolchain stabilization
- V1-CORELX-1 Stable compile/service API
- V1-CORELX-2 Unified asset model required by V1 tools/game
- V1-CORELX-3 Deterministic packaging/manifest outputs
- V1-CORELX-4 Diagnostics for tool-generated assets/references
- V1-CORELX-5 Project templates for V1 workflows ✅

### V1-FM: FM acceptance gate
- V1-FM-1 Compatibility profile document (behavioral parity)
- V1-FM-2 FM MMIO/timer/status/IRQ behavior verified against profile
- V1-FM-3 Curated FM audio acceptance references pass thresholds
- V1-FM-4 FM no longer documented as experimental for V1 scope

### V1-GAME: Galaxy Force flagship
- V1-GAME-1 Vertical shmup core gameplay
- V1-GAME-2 Matrix Mode transition sequence(s)
- V1-GAME-3 Pseudo-3D style boss encounter (or equivalent Matrix showcase)
- V1-GAME-4 Game code organized as canonical manual example source

### V1-DOCS: Documentation integration
- V1-DOCS-1 Programming manual sections mapped to real Galaxy Force modules
- V1-DOCS-2 Runnable docs snippets in-app
- V1-DOCS-3 Manual and guide content validated against current shipped behavior

## 3. Out of Scope (Post-V1 or Non-Blocking)

- macOS as first-class release target
- FPGA implementation planning and architecture work
- Experimental features not listed in this charter
- Non-critical visual polish that does not affect correctness/usability
- New language feature expansions not required by V1-GAME/V1-TOOLS

## 4. Scope-Change Rule (Required)

Any new feature request that impacts schedule must include:

1. Charter impact (which V1-* IDs are touched)
2. Priority and trade-off (what gets de-scoped if accepted)
3. Owner + estimate + risk
4. Explicit approval label: `v1-scope-change-approved`

Without that approval label, change is deferred to post-V1 backlog.

## 5. Exit Criteria

V1 release candidate can be cut only when:

1. All in-scope V1-* IDs are complete
2. Acceptance criteria in `V1_ACCEPTANCE.md` pass
3. Risks in `V1_RISKS.md` are resolved or explicitly accepted for ship
