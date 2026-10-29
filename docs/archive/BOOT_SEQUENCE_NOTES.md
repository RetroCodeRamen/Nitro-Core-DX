# Boot sequence — implementation notes (local only, not part of the public docs)

Every compiled CoreLX ROM gets a default boot presentation: the embedded
`NitroCore DX` logo slides down from off-screen and holds for ~2.5s before
the game's own `Start()` ever runs. This is intentional and automatic — no
project needs to opt in.

## The override: `function __Boot()`

If a project defines its own `function __Boot()`, the compiler uses it as
the real entry point instead of the automatic default — this is not new
compiler machinery, it's the entry-point selection `Generate()`/
`semantic.go` already had (see `test/roms/pellet_game.corelx`, which already
used this to skip the boot screen before this feature existed). Nothing
about `__Boot()` itself was changed.

- **Empty / does its own thing, calls `Start()`**: no splash at all, straight
  to the game.
- **Calls `boot.show_default()`**: plays the exact stock slide+hold sequence
  (same asset, same timing), then returns — the project decides what
  happens next (usually just call `Start()`).
- **Reads input first**: `input.poll()`/`input.held()` work normally inside
  `__Boot()`, before `Start()` runs. This is the actual point of the
  feature — a project can check for a button combo at power-on and set
  ordinary global state (e.g. `secret_mode = 1`) before the game starts,
  the same way old cartridges hid debug menus or cheat codes behind
  power-on input reads. No new builtin needed for this part; it's just
  input.poll() called earlier than usual.

Example:

```corelx
var secret_mode: int = 0

function __Boot()
    input.poll()
    if input.held(A) && input.held(B)
        secret_mode = 1  -- e.g. unlock a level-select menu from Start()
    Start()

function Start()
    if secret_mode == 1
        -- ...
```

There's no `boot.show_image(customAsset)` builtin (scoped out for time —
see below). A project wanting a fully custom splash image can just write it
by hand inside `__Boot()` using the same builtins the default sequence
itself uses (see "How the default sequence is built" below) with the
project's own declared image asset instead of the embedded one.

## Why this is undocumented

Deliberate. Not in `PROGRAMMING_MANUAL.md`, not in
`CORELX_SYNTAX_V1.md`, not in any builtin reference. `__Boot()` and
`boot.show_default()` are real, working, and will show up to anyone reading
`internal/corelx/boot_splash.go` or `internal/corelx/codegen.go` directly —
this is "undocumented," not "literally undiscoverable in the source," the
same way real retro cartridges hid stuff that was still sitting right there
in the ROM for anyone who disassembled it. Public-facing text (manual,
syntax spec, in-app docs, error messages) should never mention `__Boot()`,
never say anything like "there's no way to skip the boot logo" — just
silence. The public docs already say "a startup logo/boot presentation is
planned" (`PROGRAMMING_MANUAL.md` line ~211) and nothing more; leave it at
that.

## How it's implemented (for future-me)

- `internal/corelx/boot_splash.go` — `injectBootEntry`, called from
  `CompileSource` right after parsing, before semantic analysis.
- The embedded logo lives at `internal/corelx/bootasset/embedded_boot_logo.cxasset`
  (compiled in via `//go:embed`). It's a copy of `Images/boot.png`, resized
  to 200x200 and run through `cmd/corelx_import ... BootLogo 32 0 ...`
  (plane size 32 = 256x256 canvas, palette bank 0). If `Images/boot.png`
  changes, regenerate: resize to 200x200, re-run corelx_import, copy the
  resulting `.cxasset` text over the embedded one.
  - It's in a *subdirectory* of `internal/corelx/`, not directly in it —
    `loadImageAssets`'s orphan-.cxasset check does a non-recursive scan of
    the *project's own* source directory and would otherwise mistake it for
    a stray project asset during any test compiled with a bare relative
    sourcePath (cwd during `go test` is the package dir).
- The slide+hold sequence itself (`bootShowDefaultSource`,
  `__boot_show_default()`) is written as plain CoreLX text, lexed/parsed
  and merged into the program's AST — not hand-built AST nodes and not raw
  codegen. Keeps it going through the exact same pipeline as any other
  CoreLX code, and made it possible to visually iterate on the animation
  quickly.
  - Uses `matrix_plane` in bitmap mode on channel/BG 0, `bg.set_scroll` to
    animate, `matrix.set_flags(0, false, false, 1, false)` for
    outside_mode=1 (backdrop/solid-color) — **this part was non-obvious and
    cost real debugging time, worth remembering**: `matrix_plane`'s
    bitmap-mode image gets *stretched* to fill its whole plane canvas
    (`resizeImageNearest` in `internal/emulator/matrix_plane_image.go`), it
    is **not** padded with blank margin around the actual image content.
    The default `OutsideMode` (wrap/repeat) means scrolling "off" the plane
    just shows wrapped/repeated content, not blank space — there is no way
    to get a genuine off-screen-and-invisible slide-in without setting
    outside_mode to backdrop (1) via `matrix.set_flags`.
  - Scroll math: `worldY = screenY + ScrollY` (screen-space plus scroll,
    confirmed empirically after getting the sign backwards twice — see the
    session transcript around 2026-07-21 if this ever needs re-deriving).
    Slide is `__boot_scroll` from 200 down to 0 in steps of 5 (~40 frames);
    hold is a plain 150-frame counter. Both loops use the
    `frame_counter()`-debounce pattern (`corelx-wait-vblank-multi-iteration-pitfall`
    in the memory system) — without it the timing is not deterministic.
- `boot.show_default` is a real builtin (`semantic.go`'s
  `registerBuiltinFunctions` + `builtinNamespaces["boot"]`, and a codegen
  case in `codegen.go` that just emits a `CALL` to the merged-in
  `__boot_show_default` function via the normal `callPatches` mechanism).
- **Test compatibility** (`CompileOptions.ForceBootSplash` +
  `testing.Testing()`): under `go test`, `injectBootEntry` auto-injects a
  trivial `__Boot(): Start()` unless the program already defines `__Boot()`
  itself, or the caller explicitly passes `ForceBootSplash: true`. This
  means the entire rest of the test suite (hundreds of tests compiling
  small CoreLX snippets) needed zero changes to skip the ~3.2s splash they
  never asked to test. `testing.Testing()` is false for the real `corelx`
  CLI binary, so real compiled games always get the genuine behavior.
  Verified directly (not just inferred): compiled a ROM with the real CLI
  (`go run ./cmd/corelx`, not `go test`) and confirmed the splash shows.
  - Dedicated tests that *do* want to see the real splash pass
    `&CompileOptions{ForceBootSplash: true}` explicitly:
    `internal/corelx/boot_splash_test.go`.

## Open items / not done

- No `boot.show_image(customAsset)` convenience builtin — would need to
  pre-scan the AST for distinct assets referenced this way before codegen
  (codegen itself is too late to inject new functions), since CoreLX can't
  parameterize a merged-in helper function by an asset identifier at
  runtime. Scoped out for time; the workaround (hand-write the sequence
  with your own asset) is documented above and uses only builtins that
  already exist and are already tested.
- Never got around to precisely matching the "2-3 seconds" ask to the
  letter — it's ~0.7s slide + 2.5s hold = ~3.2s total. The *hold* alone is
  in range; if that's not close enough, `__boot_hold`'s `150` in
  `bootShowDefaultSource` is the number to change (150 frames = 2.5s at
  60fps; each unit is 1/60s).
