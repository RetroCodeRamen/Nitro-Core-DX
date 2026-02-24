# CoreLX Data Model Plan

Status: Planning document (Phase 1 prerequisite for Dev Kit expansion)

Scope: Nitro-Core-DX CoreLX language/compiler pipeline, asset model, ROM packaging model, error system design, and audio channel conventions.

This document is intentionally planning-first. It defines the target architecture and migration path before major compiler/dev-tooling changes.

## 1. Goals

CoreLX should evolve from a language demo + direct code generator into a deterministic, constraint-aware SDK pipeline that supports:

- Diff-friendly text assets
- Predictable memory layout and overflow checks
- Fast Edit -> Build -> Run workflows
- Clear, structured compiler/packaging errors
- Stable asset references usable by future Dev Kit tools (Sprite Lab, Tilemap Editor, Sound Studio)
- ROMs that remain compatible with the future FPGA hardware implementation

Primary design principle:

- Keep the language and asset pipeline deterministic and hardware-oriented.
- Developer ergonomics improve through tooling and compiler packaging, not by hiding hardware constraints.

## 2. Current-State Audit (What Exists Today)

### 2.1 CoreLX language structure (current)

Current frontend supports a single-file, indentation-sensitive language with top-level declarations:

- `asset ...`
- `type ... = struct ...`
- `function ...`

Observed characteristics in current implementation (`internal/corelx`):

- Lexer/parser track line and column positions (good base for structured errors).
- No module/import system yet (single compilation unit model).
- Asset declarations are currently limited to a narrow built-in format (`tiles8` / `tiles16` with `hex` or `b64` data payload).
- Asset AST stores raw text payload, not typed asset structures.

### 2.2 Constants/data declarations (current)

- Function-local variables and constants are handled procedurally in codegen.
- Semantic analyzer registers symbols in a single global map (limited scoping model).
- Assets are exposed semantically as synthetic constants (`ASSET_<name>`) of type `u16`.
- Asset references are not yet linked to a packed ROM asset table; they are mostly compile-time placeholders.

### 2.3 Sprite/gamedata/audio handling (current)

Sprites / graphics:

- Tile assets can be declared in CoreLX, but usage is currently special-cased in codegen (e.g. inline tile upload sequences for `gfx.load_tiles(...)`).
- Asset bytes are not generally packed into dedicated ROM sections yet.

Game data:

- No first-class gamedata asset type yet.
- Developers must encode data manually in code, use constants, or custom patterns.

Audio:

- CoreLX currently targets the legacy APU via procedural built-ins (`apu.set_channel_*`, `apu.note_on/off`, etc.).
- No first-class music/SFX/ambience asset types.
- No standardized packaging format for audio assets.
- New FM/OPM extension exists in emulator/APU MMIO (`0x9100+`) but is not yet represented in CoreLX language/built-ins.

### 2.4 Compile stages (current)

Current practical pipeline (currently embodied in test helper `CompileFile(...)` in `internal/corelx/corelx_test.go`):

1. Read source file
2. Lex/tokenize
3. Parse AST
4. Semantic analysis
5. Code generation into `rom.ROMBuilder`
6. `BuildROM()` output

Limitations:

- The compile pipeline entrypoint is effectively test-only helper code, not a formal library/compiler package API.
- Error propagation is mostly raw `error` values with inconsistent structure.
- Packaging/linking is not separated from code generation.
- Asset packing is not a formal compile stage.

### 2.5 ROM layout and memory handling (current)

Current ROM builder (`internal/rom/builder.go`):

- Single linear code stream (`[]uint16`)
- Writes header + contiguous code blob
- No sectioned ROM layout for code/assets/debug metadata
- Relative branch relocation only (16-bit); no far/cross-bank control flow support in classic builder

Current runtime ROM mapping (`internal/memory`):

- LoROM-style mapping exists and works for banked ROM access in hardware model
- CPU can read ROM in banks (`1..125`) at `0x8000..0xFFFF`
- This is sufficient for future banked code/data, but compiler/linker support is not yet complete

### 2.6 Audio channel layout (current runtime engine)

Legacy APU:

- 4 channels total (`CH0..CH3`)
- Waveforms include sine/square/saw/noise (with channel-specific behavior)
- Duration/completion-status system exists and is useful for sequencing
- PCM support exists per channel

FM extension (new APU extension):

- Host interface mapped at CPU-visible `0x9100..0x91FF`
- OPM-lite software implementation exists (subset, deterministic, FPGA-oriented)
- Timer/status/IRQ path exists
- Not yet integrated into CoreLX language asset model or high-level APIs

## 3. Planning Decisions (High-Level)

### 3.1 Asset representation choice (final recommendation)

Recommended model: **Hybrid text-based structured assets**

- Primary format: human-readable, diff-friendly text files (`.clxasset` and `.clxmap` / `.clxaudio` family)
- Optional inline text blocks inside `.corelx` for small demos/prototypes
- Compiler/package pipeline normalizes both into the same typed asset IR

Why this is the right choice

- Matches your preference (Git-friendly, diffable)
- Enables Dev Kit editors to round-trip exactly without binary noise
- Keeps hand-authoring possible for advanced users
- Avoids locking the compiler to inline-only assets (which becomes unwieldy fast)
- Still supports tiny single-file examples for tutorials

Design principle:

- Source authoring format is text.
- Compiler internal format is typed, validated asset IR.
- ROM packaged format is binary/packed and optimized for runtime loading.

## 4. Unified Asset Model (Target)

CoreLX should move from a narrow `asset <name>: tiles8|tiles16` model to a typed asset system.

### 4.1 Asset categories (first-class)

Proposed core asset kinds:

- `sprite`
- `tileset`
- `tilemap`
- `palette`
- `music`
- `sfx`
- `ambience`
- `gamedata`
- `blob` (escape hatch, low-level raw data)

Notes:

- `sprite` is authoring-facing and may compile into `tileset + metadata` internally.
- `tileset` and `tilemap` are explicit because they map cleanly to hardware VRAM/tilemap workflows.
- `music/sfx/ambience` separate intent for tooling, even if runtime representation shares a common event-stream format.
- `blob` avoids blocking low-level experimentation while the higher-level formats mature.

### 4.2 Common asset metadata schema

Every asset should have a normalized metadata record in the compiler IR:

- `kind` (enum)
- `symbol` (fully qualified name)
- `source_file`
- `source_span` (line/column range)
- `tags` (optional key-value metadata for tools)
- `target_profile` (`dx`, later `nc8`, or `portable` subset)
- `raw_text_hash` (for incremental build caching)
- `compiled_size_bytes`
- `alignment`
- `section` (pack target)

## 5. Asset Data Representations (Target)

### 5.1 Sprite data

Authoring representation (recommended)

- `.clxasset` text format (palette-index grid, two-hex per pixel for exact round-trip)
- Supports at minimum `8x8` and `16x16`
- Optional metadata header fields:
  - size
  - bpp/palette mode
  - palette reference
  - origin/pivot
  - tags

Compiler normalized representation

- `SpriteAssetIR`
  - dimensions
  - pixel format (`indexed4`, `indexed8`, etc.)
  - palette reference (symbol)
  - pixel indices (`[]uint8`)
  - derived tile data chunks

Packaged ROM representation

- Tile bytes (hardware-ready layout) placed in a graphics asset section
- Optional sprite metadata table (dimensions, tile count, default palette ID)

Key design note

- Sprite authoring stays semantic/textual.
- ROM packaging emits hardware-ready tiled bytes so runtime loading is cheap and deterministic.

### 5.2 Tilemap data

Authoring representation (recommended)

- Text-based structured tilemap file (`.clxmap`) or embedded block
- Grid of tile references / tile indices + attributes (palette, flip, priority where supported)
- References can be symbolic during authoring/tooling and resolved to numeric indices during pack

Compiler normalized representation

- `TilemapAssetIR`
  - width / height (in tiles)
  - layer count (start with at least 1)
  - per-cell records: tile index, palette, flip flags, priority, optional collision/meta tags
  - tileset symbol dependency

Packaged ROM representation

- Packed tilemap bytes in a tilemap section (hardware-friendly entry layout)
- Optional sidecar metadata block for editor-only debug/collision annotations (excluded from release build or placed in debug section)

### 5.3 Audio data (music, ambience, SFX)

Authoring representation (recommended)

- Text-based event/sequence format (`.clxaudio`), plus optional specialized views in Sound Studio
- Separate logical asset kinds:
  - `music` (looping or scripted score)
  - `ambience` (looping low-priority layers / drones / environmental patterns)
  - `sfx` (short one-shot events)

Compiler normalized representation

- `AudioAssetIR`
  - `intent` (`music|ambience|sfx`)
  - channel plan (requested/allowed channels)
  - event tracks (note on/off, instrument/patch, volume, wait/duration, loop markers, control changes)
  - target backend metadata (`legacy_apu`, `fm_extension`, `hybrid`)
  - optional imports/transcoded note tables (future tooling)

Packaged ROM representation

- Packed event streams + patch tables + lookup metadata
- Separate sections for:
  - sequence/event data
  - instrument/patch definitions
  - optional PCM sample blobs (later)

Important constraint

- The compiler packages audio as deterministic event/pattern data, not host-runtime streaming assets.
- Runtime playback must remain reproducible on emulator and FPGA hardware.

### 5.4 Game data

Authoring representation (recommended)

- Structured text (`.clxdata`) or embedded block
- Supports typed records/arrays and named tables for gameplay tuning values
- Intended for levels, stats, dialog tables, AI params, lookup tables, etc.

Compiler normalized representation

- `GameDataAssetIR`
  - schema-validated typed fields / arrays
  - optional symbolic references to other assets
  - packed layout plan (endianness/size explicit)

Packaged ROM representation

- Binary packed records in a `gamedata` section
- Optional generated symbol table for offsets/handles (for compile-time constants and Dev Kit memory viewer)

## 6. Asset Authoring Model (Embedded vs Separate vs Hybrid)

Final recommendation: **Hybrid** (separate files as default, embedded blocks supported)

### 6.1 Default (recommended for real projects)

Use separate text files:

- `assets/sprites/*.clxasset`
- `assets/tilemaps/*.clxmap`
- `assets/audio/*.clxaudio`
- `assets/data/*.clxdata`

Advantages:

- Diff-friendly and Git-friendly
- Cleaner source modules (game logic stays readable)
- Natural integration with Dev Kit editors
- Enables isolated asset validation and caching

### 6.2 Embedded blocks (supported, not preferred at scale)

Allow inline asset blocks for:

- examples/tutorials
- tiny prototypes
- generated snippets pasted from Sprite Lab

Compiler behavior:

- Embedded and file-based assets are normalized into identical asset IR records
- The rest of the pipeline does not care where the asset came from

## 7. Asset Referencing and Linking

### 7.1 How code should reference assets

Recommended reference model

- Code references assets by **symbol** at source level
- Compiler resolves symbols to **typed handles/IDs + section offsets** during link/pack

Example conceptual forms (not final syntax):

- `gfx.load_tiles(MyTiles, base=0)`
- `map.draw(Level1_BG)`
- `audio.play_music(MountainKingIntro)`
- `audio.play_sfx(JumpBewp)`
- `data.load(PlayerStatsTable)`

Compiler/runtime representation

- Asset symbol -> `AssetHandle`
- `AssetHandle` contains (conceptually):
  - asset kind
  - section ID
  - bank/offset
  - size
  - format metadata ID

### 7.2 Linking responsibilities

Introduce a formal packaging/link stage after code generation:

1. Collect all code and asset declarations
2. Validate asset types + references + dependencies
3. Compile/normalize assets into binary payloads
4. Assign sections and placements (bank/offset/alignment)
5. Resolve asset handles/constants in code
6. Emit ROM image + manifest (for Dev Kit)

This separates concerns cleanly:

- frontend = syntax/semantic correctness
- codegen = instruction emission
- linker/packer = layout, references, overflow, manifests

### 7.3 Name collisions

Recommended rules

- Global asset/type/function namespace should become **module-scoped** (future), but in Phase 1 we can still improve diagnostics in a flat namespace.
- Collisions are compile errors with category `E_SYMBOL_DUPLICATE`.

Collision policy

- `asset Foo` and `function Foo` in same namespace: error
- duplicate asset names: error
- duplicate type names: error
- duplicate local variables in same scope: error
- shadowing policy (future): allowed only with explicit local scope rules; disallow in first structured-error pass if needed for simplicity

Error message should include:

- current declaration location
- previous declaration location
- symbol kind(s)

## 8. Memory Layout and ROM Sections (Target)

### 8.1 Packaging model for ROM sections

Define explicit logical ROM sections (compiler/packer view):

- `code`
- `rodata` (constants, lookup tables)
- `gfx_tiles`
- `tilemaps`
- `palettes`
- `audio_seq`
- `audio_patch`
- `audio_pcm` (future)
- `gamedata`
- `debug_meta` (optional dev builds only)

Why this matters

- Enables overflow detection per domain
- Enables Dev Kit memory usage viewer by section
- Makes banking/linking deterministic and inspectable

### 8.2 Placement model

Short-term (Phase 1/2 compatible)

- Continue supporting current single-bank codegen path for small projects
- Add packer/linker section planner that can place data sections in LoROM banks even before far-call codegen is finished

Medium-term

- Code and assets both bank-aware
- `BankedROMBuilder` (already started) becomes the primary packaging backend
- Cross-bank code/data references resolved via relocations/handles

### 8.3 Sprite/tile/audio memory regions (runtime-facing planning)

The compiler/packer should define **ROM sections** and generate metadata for runtime loading into hardware memory regions. It should not silently assume VRAM/CGRAM residency.

Runtime target regions (conceptual):

- VRAM: tiles, tilemaps (loaded on demand or init)
- CGRAM/palette memory: palettes
- APU/FM registers or APU RAM (via playback engine): audio events/patches
- WRAM: runtime game state + temporary decoded assets (if needed)

Compiler responsibilities

- Record packaged ROM offsets and sizes
- Validate section size limits (ROM-side)
- Emit metadata/handles that runtime loaders can use to DMA/copy into VRAM/CGRAM/etc.

### 8.4 Overflow detection

Required overflow checks (pack time)

- Total ROM size exceeds format/platform limit
- Section exceeds configured size budget
- Bank placement overflow/alignment failure
- Asset payload exceeds declared/allowed limits for its type
- Tilemap dimensions exceed PPU/runtime supported dimensions (configurable limits)
- Audio event data exceeds engine limits (channel count, patch count, command size, etc.)

Behavior

- Compiler/packer must return structured errors (do not panic)
- All detected errors should be reported in one pass where possible

### 8.5 Memory usage reporting for future Dev Kit

Packer should emit a machine-readable build manifest (e.g. JSON) containing:

- total ROM size
- section sizes and addresses
- per-asset size + placement
- unresolved/warn-level compression opportunities (future)
- bank usage heatmap data (future)

This manifest will drive:

- Memory & Asset Viewer
- Build output summary
- overflow diagnostics in packaging panel

## 9. Packaging Model (Code + Assets Assembly)

### 9.1 Proposed compiler pipeline (target)

Formalize the pipeline into explicit stages:

1. **Load Sources**
   - `.corelx` modules + referenced asset files
2. **Lex / Parse**
   - produce AST(s)
3. **Semantic Analysis**
   - symbol/type checks, entrypoint checks, asset declaration validation
4. **Asset Normalization**
   - parse text asset formats into typed IR
5. **Code Generation**
   - emit code IR / machine code + relocation records
6. **Link / Pack**
   - resolve symbols/assets, assign sections/banks, detect overflow
7. **ROM Emit**
   - write ROM image
8. **Manifest Emit** (dev builds by default)
   - write build manifest + diagnostics artifacts

### 9.2 Build / Build+Run workflow expectations

Packaging tool (Dev Kit) should wrap the same pipeline and expose:

- `Build` (compile + package + report)
- `Build + Run` (compile + package + launch emulator)

Success path

- ROM packaged
- manifest generated
- emulator launched with produced ROM

Failure path

- no crash
- structured errors returned
- editor panel highlights file/line and category
- previous successful ROM can remain runnable (optional UX feature)

## 10. Error System Design (Compiler + Packer)

### 10.1 Error categories (required)

Define structured diagnostic categories (examples):

- `SyntaxError`
- `LexError`
- `SymbolError`
- `TypeError`
- `ValidationError` (semantic rules, entrypoint, unsupported feature use)
- `AssetParseError`
- `AssetReferenceError`
- `AssetFormatError`
- `LayoutError` (placement/alignment)
- `OverflowError`
- `BackendCodegenError`
- `InternalCompilerError` (unexpected bugs; should still not panic user-facing tooling)

### 10.2 Diagnostic structure (target)

Introduce a diagnostic struct used across stages:

- `Category`
- `Code` (stable ID like `E_ASSET_UNKNOWN_SYMBOL`)
- `Message` (human-readable)
- `File`
- `Line`
- `Column`
- `EndLine` / `EndColumn` (optional span)
- `Notes []string` (actionable follow-up hints)
- `Related []Location` (e.g. previous declaration)
- `Severity` (`error|warning|info`)
- `Stage` (`lexer|parser|semantic|asset|codegen|link|pack`)

### 10.3 Message quality requirements

Messages must be:

- specific
- actionable
- non-snarky
- include expected vs actual where possible

Examples of good behavior

- “Unknown asset `PlayerTiles`; did you mean `PlayerTileSet`?”
- “Tilemap `Level1` references tile index 513 but tileset `DungeonTiles` contains 256 tiles.”
- “ROM overflow: `audio_seq` section exceeds budget by 2048 bytes (budget 32768, used 34816).”

### 10.4 Non-crashing failure behavior

Requirements

- No `panic` for user-authored errors (syntax, symbols, overflow, bad assets)
- Pipeline accumulates diagnostics where safe and returns all relevant issues
- Internal crashes are caught at tool boundary and surfaced as `InternalCompilerError` with stack/log path in dev mode

## 11. Audio Channel Standardization (Recommended Default)

Goal: make audio predictable for developers while preserving flexibility and future FM integration.

### 11.1 Current hardware/software reality

- Legacy APU has 4 channels (`CH0..CH3`) with a noise-capable path
- FM extension provides additional FM voices (OPM-lite now, richer later)
- CoreLX high-level audio layer does not exist yet (procedural APU calls only)

### 11.2 Recommended default channel convention (DX SDK v1)

Legacy APU (default/simple engine profile)

- `CH0`: Music melody / lead (or SFX fallback if music idle)
- `CH1`: Music harmony / countermelody
- `CH2`: Music bass / drone / ambience tonal layer
- `CH3`: Shared percussion/noise OR SFX priority channel

SFX reservation policy (recommended default)

- Reserve **1 dedicated SFX channel** by default: `CH3`
- Allow optional override to reserve **2 SFX channels** (`CH2 + CH3`) for action-heavy games

Tradeoff reasoning

- Reserving 2 SFX channels makes gameplay feedback stronger but noticeably weakens music on the 4-channel legacy APU.
- Reserving only 1 SFX channel gives better music layering (melody/harmony/bass) and is a better default for demos and general SDK examples.
- Developers can opt into an action profile that sacrifices music richness.

Noise channel policy

- `CH3` noise is **shared-use by policy**:
  - music percussion when no active high-priority SFX
  - preempted by SFX when needed

Engine/tooling implication

- Sound Studio and runtime mixer/sequencer should support channel priority and preemption metadata.

### 11.3 FM extension standardization (future-facing, recommended)

When FM path is enabled:

- FM voices handle music lead/harmony/pads/bass by default
- Legacy APU channels shift toward:
  - SFX
  - percussion/noise
  - UI sounds
  - simple backup ambience

Recommended hybrid profile (future default for advanced projects)

- FM: music primary voices
- Legacy `CH0-CH1`: SFX/UI
- Legacy `CH2`: ambience/drone or bass reinforcement
- Legacy `CH3`: noise percussion / transient SFX

This keeps the legacy APU useful and avoids forcing every game to program FM directly.

## 12. Dev Kit Integration Requirements (Architectural Hooks to Plan Now)

Even before implementing Dev Kit features, Phase 1 must produce stable interfaces for:

- Structured diagnostics (for clickable error panel)
- Build manifest (for memory/asset viewer)
- Asset IR + parsers (for Sprite Lab / Tilemap Editor / Sound Studio export)
- Deterministic packer (for Build + Run workflow)

Required compiler outputs (beyond ROM)

- `Diagnostics[]`
- `BuildManifest`
- Optional `SymbolMap`
- Optional `AssetCatalog` (resolved handles, kinds, placements)

## 13. Implementation Order (Refined, Planning-Aligned)

This aligns with your requested order while grounding it in the current codebase.

1. **Audit and freeze plan** (this document)
2. **Extract formal compiler entrypoint API from test helper**
   - move `CompileFile(...)` logic out of tests into production package API
   - return structured result (`ROM bytes/path + diagnostics + manifest`) instead of plain `error`
3. **Implement structured diagnostics across lexer/parser/semantic/codegen/packer**
4. **Implement asset normalization layer + unified asset IR**
   - keep current `tiles8/tiles16` support as compatibility path
5. **Implement packaging/link stage with section layout + overflow detection**
   - start with data sections and current code builder
   - migrate toward `BankedROMBuilder`
6. **Expose build manifest for Dev Kit**
7. **Build Packaging Tool (Build / Build+Run) in Dev Kit UI**
8. **Sprite Lab** (`.clxasset` round-trip exact)
9. **Tilemap Editor** (`.clxmap` export/import)
10. **Sound Studio** (`.clxaudio` export/import with channel policy metadata)
11. **Debug Overlay**
12. **Memory & Asset Viewer** (manifest-driven)

## 14. Compatibility / Migration Strategy

### 14.0 Pre-Alpha Compatibility Policy (Updated)

Per current project direction, **backward compatibility is not a hard requirement before alpha**.

Implications for Phase 1/2 work:

- We may redesign CoreLX syntax/asset declarations/compiler APIs if it materially improves the final SDK architecture.
- Existing examples/tests are useful references, but they do not block structural improvements.
- The primary compatibility target right now is **design consistency + hardware/FPGA reproducibility**, not legacy CoreLX source stability.

### 14.1 Preserve current CoreLX examples and tests

- Prefer preserving useful examples where cheap, but not at the expense of the Phase 1 architecture.
- Existing `asset tiles8/tiles16` declarations can be retained temporarily or replaced with a migration tool/shim.
- Existing APU built-ins may be reworked if the new unified asset/audio model requires cleaner APIs.

### 14.2 Introduce new features incrementally

- Add new asset forms behind parser extensions, not breaking changes
- Add packer/linker stage while preserving current ROM output path for small projects
- Add FM/audio asset abstractions without removing low-level `apu.*` and `fm.*` access

### 14.3 FPGA compatibility guardrail

All packaging/runtime asset decisions must preserve:

- deterministic playback/loading behavior
- explicit memory placement and size constraints
- no host-only asset streaming assumptions

## 15. Immediate Next Actions (After Plan Approval)

1. Extract a production `CompileProject` / `CompileFile` API from `internal/corelx/corelx_test.go`
2. Define and implement a shared `Diagnostic` type used by parser + semantic analyzer first
3. Implement `BuildManifest` skeleton emitted by the existing ROM build path
4. Add compatibility asset IR for current `tiles8` / `tiles16` assets
5. Add pack-time section accounting and overflow checks (even before full bank linking)

---

## Appendix A: Practical Phase 1 Deliverables (Minimum)

To unblock Dev Kit work quickly, Phase 1 should produce at least:

- `corelx.CompileProject(...)` (production API)
- Structured diagnostics with file/line/column
- Basic asset IR for `tileset/palette/blob` + compatibility tiles assets
- Packer section accounting (`code`, `gfx_tiles`, `gamedata`, `audio_seq` placeholder)
- Build manifest JSON output

This is enough to build:

- Packaging Tool (Build / Build+Run)
- Error panel (clickable diagnostics)
- Memory/Asset viewer (initial version)

## Appendix B: Why not binary-first assets?

Binary-only assets would be simpler for packing but are the wrong default for your goals because they:

- produce poor diffs/reviews
- make hand-editing and debugging harder
- complicate deterministic tool round-trip visibility
- increase friction before the Dev Kit editors are mature

Text-first authoring with typed normalization gives the best long-term developer experience while keeping the packaged runtime format efficient.
