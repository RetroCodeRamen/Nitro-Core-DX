# Nitro-Core-DX v0.2.0

## What Changed (Plain-English)

This release closes out the ROM-first era and sets the table for CoreLX v1. The pack-in demo is finished, the CoreLX language design is fully decided and documented, and the repository itself got a deep cleanup so the language work starts from a trustworthy foundation.

### 1) The NitroPackInDemo Pack-In Demo Is Complete

The ROM-first showcase now runs its entire loop: title screen, pseudo-3D overworld park, walking into the building, an interior room with a checkered matrix-plane floor and an NPC guide, a two-page typewriter dialogue, credits, and a clean reset back to the title.

Highlights:
- matrix-plane floor plus vertical projected facades, indoors and out, on one coherent camera model
- NPC collision, door/exit trigger zones, pause overlay, and edge-handled input
- the full scene loop is exercised headlessly by automated tests

Run it: `./emulator -rom roms/nitro_pack_in_demo.rom`

What that means for you:
- the console has a real, playable reference game proving the pseudo-3D feature set
- this demo is the acceptance test for the CoreLX rebuild — same game, new language, compared frame by frame

### 2) CoreLX v1 Is Designed

The complete language design is settled and written down — not implemented yet, but decided, with rationale:

- **Syntax charter** (`docs/specifications/CORELX_SYNTAX_V1.md`): learnability-first syntax anchored in Lua, BASIC, and Go. Two numeric types to learn (`int` and `fixed`, with decimal literals just working as fixed-point), structs that pass by reference, BASIC-style counting loops, and no second way to do anything.
- **Cartridge format** (`docs/specifications/CORELX_CARTRIDGE_FORMAT.md`): one text file is the whole game — code plus sprites, backgrounds, and audio as readable text sections. Editors convert PNGs and samples to text at import time; the compiler is a deterministic text-to-ROM function.
- **Decision record** (`Games/NitroPackInDemo/CORELX_EXTRACTION.md`): every design question raised by the demo extraction, answered — memory model, modules, generic transformation planes, what ships in v1, and the stability contract: once v1 ships, the language only grows additively; programs that compile on v1 behave identically forever.

What that means for you:
- the next development cycle is implementation, not design churn
- the language you learn at v1 is the language, permanently

### 3) The Repository Got a Deep Cleanup

- documentation pass over everything: dead docs deleted, historical plans and audits archived, navigation maps rewritten around the current sources of truth, stale status claims corrected
- file-mode drift from a machine migration fixed (1,400 files), so git history stays honest
- build artifacts, logs, stale release staging, and an accidentally committed third-party installer removed
- `make test-full` now covers exactly the project's packages (including the demo's ROM-builder tests) and passes clean

## Versioning Note

This is a minor-version bump (0.1.x → 0.2.0) marking the transition from "prove the console" to "build the language." The next cycle is M8: implementing CoreLX v1 and rebuilding the demo in it.
