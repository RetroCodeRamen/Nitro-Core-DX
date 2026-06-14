# CoreLX v1 — Implementation Status & Handoff

**Last updated: 2026-06-14.** This is the **first document to read** if you are
picking up CoreLX work. It states exactly what is built and verified, what is
pending, how the asset/project pipeline works, and where every design decision
is recorded. It is kept current as M8 progresses.

Authoritative references (do not duplicate them here — this doc points to them):

- **Design decisions:** `Games/NitroPackInDemo/CORELX_EXTRACTION.md` §13 (the
  decision record) and the per-session memory.
- **Syntax (frozen at v1):** `docs/specifications/CORELX_SYNTAX_V1.md`.
- **Cartridge/project format:** `docs/specifications/CORELX_CARTRIDGE_FORMAT.md`.
- **Build order:** `CORELX_EXTRACTION.md` §12 (now status-annotated).

The guiding rule for all of this: **the emulator implementation is the
authoritative hardware contract** (the FPGA is built to match it). Every CoreLX
feature lands with an emulator-executed test — compile → ROM → run on the
emulator core → assert machine state or framebuffer. No feature is "done" on a
"the compiler makes sense" basis alone.

---

## 1. Where we are

M8 = rebuild the NitroPackInDemo overworld in CoreLX, using the demo as the
acceptance test that drives what the language needs. The **language core and the
asset/image pipeline are built and verified**; the **demo rebuild is in
progress** (floor renders from the real image; building billboard, smooth
turning, and the door interaction are the remaining visible pieces).

---

## 2. Implemented & verified

Every item below compiles and runs on the emulator with a passing test in
`internal/corelx/`.

### Language core
| Feature | Notes | Test |
|---|---|---|
| `const` | compile-time fold, fixed-aware, immutable | `globals_test.go` |
| globals (`var`) | auto-allocated WRAM from `0x2100`; `at` pins (overlap-checked); reserved runtime block `0x2000-0x20FF`; user scratch `0x7000-0x7FFF`; `.memmap` emitted per build | `globals_test.go` |
| arrays `T[N]` | zero-init; constant index bounds-checked at compile time; runtime index unchecked (documented) | `globals_test.go` |
| array initializers `= [..]` | constant value lists (data tables, e.g. heading tables) | `arrayinit_test.go` |
| `fixed` (8.8) | decimal literals **are** fixed; hardware MUL/DIV; `__fixmul` routine; `int()/fixed()` conversions; fixed/int mixing is an error; **const fold == runtime fixmul bit-for-bit** | `hwaccuracy_test.go` |
| strings | labels only (no string type); inline-streamed to text port | `text_test.go` |
| `for i = 0 to N [step M]` | BASIC inclusive loop (replaced the unused C-style for) | `forloop_test.go` |
| `&` removed | structs are reference types (charter D8) | `ampersand_test.go` |
| `mem.read16/write16` | hardware-accurate WRAM 16-bit / I/O low-byte routing | `mem16_test.go` |

### Builtins (hardware-register tier)
| Builtin | Purpose | Test |
|---|---|---|
| `input.poll/held/pressed/released` + button constants (`UP`..`Z`) | edge-detected input, state in runtime block | `input_test.go` |
| `text.draw(x,y,r,g,b,"str")` | HUD text via the text port | `text_test.go` |
| `text.draw_int(x,y,r,g,b,value)` | signed integer → digits (scores/counters) | `drawint_test.go` |
| `matrix_plane.set_projection/set_depth/set_camera/set_surface` | generic plane projection (perspective + vertical-quad), camera, surface placement | `projection_test.go` |
| `matrix_plane.load_bitmap(asset, channel)` | palette → CGRAM, bitmap-source plane control, chunked DMA from ROM | `loadbitmap_test.go` |

(Pre-existing builtins — `bg.*`, `matrix.*`, `matrix_plane.load_tiles/...`,
`oam.*`, `sprite.*`, `gfx.*`, `raster.*`, `apu.*` — are listed in
`docs/CORELX.md`. The full live list is the registration block in
`internal/corelx/semantic.go`.)

### Asset & project pipeline
| Piece | Notes | Test |
|---|---|---|
| `corelx_import` | PNG → `.cxasset` text (conversion frozen here for determinism) | `cmd/corelx_import/` |
| `asset X: image "file.cxasset"` | one-line external image reference | `loadbitmap_test.go` |
| ROM data region | bitmap blobs placed in ROM banks 2+ (`rom.ROMBuilder.SetDataRegion`) | `internal/rom` |
| `.ncdx` container | project = ZIP of `main.corelx` + `.cxasset` + `project.toml`; compiler reads it (extract-to-temp) or a plain folder | `container_test.go` |
| `.cart` output | compiled ROM | — |
| validation | missing reference **and** orphan `.cxasset` are blocking errors | `container_test.go`, `loadbitmap_test.go` |

### Demo rebuild progress
- `Games/NitroPackInDemo/corelx/overworld.corelx` — walkable floor + player
  sprite + HUD; the **real `park.png` renders as a pseudo-3D floor** from a
  `.cxasset`. Tests: `overworld_test.go`, `floorrender_test.go`.

---

## 3. Asset & project workflow (how to use it)

```
# 1. Convert art to a text asset (devkit importer):
go run -tags no_sdl_ttf ./cmd/corelx_import  myart.png  MyArt  32  1  out/my_art.cxasset
#                                              image      name  size palBank  output
#   size = matrix-plane tiles per side (32/64/128); palBank = CGRAM bank.

# 2. Reference it in main.corelx and use it:
#      asset MyArt: image "my_art.cxasset"
#      matrix_plane.load_bitmap(MyArt, 0)

# 3. A project is a folder (or a .ncdx zip of it) containing main.corelx + the
#    .cxasset files + project.toml. Compile to a .cart:
./corelx  MyGame.ncdx  MyGame.cart        # or: ./corelx projectFolder/ out.cart

# 4. Run:
./emulator -rom MyGame.cart
```

Hard rules enforced by the compiler: a referenced asset that is missing is an
error; a `.cxasset` in the project that nothing references is an error.

---

## 4. Pending (M8 remaining)

- **Demo visuals:** building billboard from `building.png` (vertical projection
  via `set_surface`), smooth 64-direction turning (heading table via array
  initializers), door proximity + A-to-enter.
- **Modules:** the `--!` directive system and the genre-neutral `anim` + `sfx`
  modules (CORELX_EXTRACTION.md §12 step 6).
- **Audio:** the fuller music API + the `.cxasset`/inline audio text format
  (co-design — CORELX_CARTRIDGE_FORMAT.md §5).
- **Scenes/dialogue/credits** as demo game code (patterns, not language).

---

## 5. Build / test / run commands

```
# Build the toolchain (this machine: needs -tags no_sdl_ttf; never `go build ./...`):
go build -tags no_sdl_ttf -o corelx ./cmd/corelx
go build -tags no_sdl_ttf -o corelx_import ./cmd/corelx_import

# Tests (every CoreLX feature is emulator-verified here):
go test -tags no_sdl_ttf ./internal/corelx -timeout 120s
make test-full          # whole project

# The emulator binary is prebuilt at ./emulator (see kitsune-dev-environment memory).
```

---

## 6. Open design questions (not yet decided)

- Audio text format + music API specifics (CORELX_CARTRIDGE_FORMAT.md §5).
- `project.toml` schema (currently just carried in the container; fields TBD).
- Whether sprite art moves to `.cxasset` too, or stays inline per the hard split
  (current decision: small sprite/tile data stays inline; only bitmap planes and
  samples are external).

When any of these is decided, record it in CORELX_EXTRACTION.md §13 and update
this status doc.
