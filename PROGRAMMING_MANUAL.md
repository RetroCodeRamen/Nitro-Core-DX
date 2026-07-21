# The Nitro-Core-DX Programming Guide

### Making games with CoreLX and the DevKit

**A Retro Code Ramen publication for Nitro-Core-DX.**
Written by AJ / Retro Code Ramen. Your guide through the machine is Fletcher,
who is not the author, does not want the paperwork that comes with being the
author, and would like that noted up front.

> **This is the programmer's book.** If you just want to play games on your
> Nitro-Core-DX, you want the Console Owner's Manual
> (`docs/NITRO_CORE_DX_OWNERS_MANUAL.md`) instead. This guide is for building
> them: the CoreLX language, the DevKit, and how to make the machine do what
> you want.

> **Status: living manual.** Sections appear here only after the feature they
> describe is implemented and verified running on the Nitro-Core-DX core — so
> nothing in this guide is a promise. It already works, or it isn't in here
> yet. Every demo program in this book is compiled and run against the real
> emulator core by the test suite (`internal/corelx/manual_examples_test.go`);
> if one ever stops working, the build breaks. If something in this manual and
> the code disagree, trust the code and tests first.

---

## Table of Contents

**Getting Started**
1. [Two Ways to Program Nitro-Core-DX](#two-ways-to-program-nitro-core-dx)
2. [The Nitro-Core-DX App](#the-nitro-core-dx-app)
3. [CoreLX Quick Start](#corelx-quick-start)

**Part 1 — The CoreLX Language**
4. [Two Kinds of Numbers, and Why One of Them Lies to You](#chapter-1--two-kinds-of-numbers-and-why-one-of-them-lies-to-you)
5. [Constants: Naming Things So You Stop Lying to Yourself](#chapter-2--constants-naming-things-so-you-stop-lying-to-yourself)
6. [Globals, and the Map of Where Everything Lives](#chapter-3--globals-and-the-map-of-where-everything-lives)
7. [Arrays: When You Need a Row of the Same Thing](#chapter-4--arrays-when-you-need-a-row-of-the-same-thing)
8. [Loops That Count: `for i = 0 to N`](#chapter-5--loops-that-count-for-i--0-to-n)
9. [Conditions, and Why There's No `else if` Yet](#chapter-6--conditions-and-why-theres-no-else-if-yet)
10. [Structs: Beyond `Sprite()`](#chapter-7--structs-beyond-sprite)
11. [Putting Words on the Screen](#chapter-8--putting-words-on-the-screen)
12. [Reading the Controller](#chapter-9--reading-the-controller)
13. [The Game Loop: Frames, VBlank, and `wait_vblank()`](#chapter-10--the-game-loop-frames-vblank-and-wait_vblank)
14. [Modules: Sharing Code With `--!`](#chapter-11--modules-sharing-code-with---)
15. [Sprites and OAM](#chapter-12--sprites-and-oam)
16. [Matrix Planes: Floors and Billboards](#chapter-13--matrix-planes-floors-and-billboards)
17. [Audio: Music and Sound Effects](#chapter-14--audio-music-and-sound-effects)
18. [Assets and the `.ncdx` Project Format](#chapter-15--assets-and-the-ncdx-project-format)

**Part 2 — Building a Game**
19. [The Demo Programs](#the-demo-programs)
20. [What's Next](#whats-next)

**Part 3 — Tools and Reference**
21. [Build and Run Workflows](#build-and-run-workflows)
22. [Assembly (Advanced Users)](#assembly-advanced-users)
23. [CoreLX vs. Assembly: When to Use Which](#corelx-vs-assembly-when-to-use-which)
24. [Troubleshooting Guide](#troubleshooting-guide)
25. [What Is Planned](#what-is-planned)
26. [Reference Links](#reference-links)
27. [Final Advice](#final-advice)

---

## A Short Word Before Fletcher Takes Over

CoreLX is the language you use to make the Nitro-Core-DX do things. Not a
general-purpose language you later bend toward a console — a language shaped
around *this* machine: its 16-bit registers, its WRAM, its very specific
opinions about numbers. You will learn it by building real programs and then
breaking them on purpose, because that is the fastest way to understand any
machine that has ever existed.

Fletcher will handle the rest. He has been here longer than the documentation.

---

## Two Ways to Program Nitro-Core-DX

### 1. CoreLX (Recommended for Most Projects)

CoreLX is the main language for game and app development:

- indentation-based syntax
- hardware-oriented built-ins (`ppu.*`, `gfx.*`, `sprite.*`, `oam.*`,
  `input.*`, `bg.*`, `matrix.*`, `matrix_plane.*`, `music.*`, `mem.*`)
- compiles directly to machine-code ROMs
- integrated into the Nitro-Core-DX app's `Build` / `Build + Run` flow

### 2. Assembly (Advanced / Low-Level)

A real v1 text assembler (`.asm` -> `.rom`) exists for advanced users. Use it
when you want exact instruction-level behavior, hardware bring-up tests, or to
learn the CPU and machine model directly.

> **Important:** CoreLX and Assembly are currently **separate build paths**.
> Inline mixed-mode `asm { ... }` inside CoreLX is not implemented yet.

---

## The Nitro-Core-DX App

Nitro-Core-DX (the integrated app / DevKit) is a professional IDE for
day-to-day development.

### Run It

```bash
go run ./cmd/corelx_devkit
```

### IDE Structure

The app uses a traditional IDE layout with a menu bar and domain-grouped
toolbar:

**Menu Bar:** File, Edit, View, Build, Debug, Tools, Help

**Toolbar Groups (left to right):**
- **Project:** New, Open, Save, Load ROM
- **Build:** Build, Build + Run (primary action)
- **Run/Debug:** Run, Pause, Stop, Step Frame, Step CPU
- **View:** Split View, Emulator Focus, Code Only

### View Modes

- **Split View** — Editor + emulator side by side (default)
- **Emulator Focus** — Emulator fills the workspace for play/test workflows
- **Code Only** — Editor fills the workspace, emulator hidden for focused coding

### DevKit Features

- **Project Templates:** Blank Game, Minimal Loop, Sprite Demo, Tilemap Demo,
  Shmup Starter, Matrix Mode Demo
- **CoreLX Editor:** inline syntax highlighting, line numbers, active-line
  emphasis, diagnostics jump
- **Sprite Lab:** pixel-art sprite editor (see below)
- **Tilemap Lab:** tilemap paint/edit tool (see below)
- **Diagnostics Panel:** compiler errors/warnings with severity filtering
- **Build State:** `Draft`, `Validating...`, `Validated`, `Error`
- **Build Output / Manifest / Debug Panels**
- **Autosave** and **Settings Persistence** across sessions
- **Load ROM:** test prebuilt `.rom` files directly without recompilation

### Sprite Lab

- Canvas sizes 8x8 to 64x64 (step of 8)
- 16 palette banks, 16 colors each (RGB555)
- Pencil/Erase, optional Mirror X painting
- Wrapped sprite shifting (Shift Up/Down/Left/Right)
- Grid overlay, hover highlighting
- Undo/Redo (up to 128 states)
- Import/Export `.clxsprite`
- **Apply To Manifest** / **Insert CoreLX Asset** / **Apply To Project**
- Preview pane with packed 4bpp hex output, transparent index-0 checkerboard

### Tilemap Lab

- Map sizes 8x8 to 64x64 (step of 8)
- Tile-entry editing as packed `(tile, attr)` values
- Brush/fill/erase, undo/redo
- Palette/flip attribute editing (`pal`, `flipX`, `flipY`)
- Parses tile assets from the current source (`tiles8`, `tileset`, `sprite`)
  into a selectable tile atlas
- Import/Export `.clxtilemap`

### Typical Workflow

1. **New** from a template, or **Open** an existing `.corelx` file
2. Edit code in the CoreLX editor
3. **Build + Run** to compile and run in the embedded emulator
4. Use **Sprite Lab** / **Tilemap Lab** to create assets, apply them to source
5. **Load ROM** to test prebuilt ROMs without recompiling
6. Confirm the build status returns to **Validated** after edits/builds

### Project Asset Manifest (`corelx.assets.json`)

The compiler service path (used by DevKit Build/Build+Run) automatically
checks for `corelx.assets.json` next to your `.corelx` file. If present,
manifest assets are loaded and merged with in-source `asset` declarations.
Editor tools (Sprite Lab, Tilemap Lab) are proposal/edit helpers — compiler
output remains the source of truth for what actually ships in the ROM.

> **Quick Note:** If game input seems unresponsive, make sure **Capture Game
> Input** is enabled and click the emulator pane once.

---

## CoreLX Quick Start

### Your Entry Point: `Start()`

Every normal CoreLX program begins with:

```corelx
function Start()
    -- your game/app code here
```

Before `Start()` runs, the compiler always shows the Nitro-Core-DX boot logo —
a brief slide-in-and-hold splash — unless your program defines its own
`__Boot()` function to take over the entry point (see Chapter 10 for when and
why you'd do that; most programs never need to).

### Smallest Useful Program

```corelx
function Start()
    ppu.enable_display()

    while true
        wait_vblank()
```

This turns on display output and waits forever.

### Compile from the CLI

```bash
go run ./cmd/corelx hello.corelx hello.rom
```

### Run in Emulator

```bash
go run ./cmd/emulator -rom hello.rom
```

---

# Part 1 — The CoreLX Language

## Chapter 1 — Two Kinds of Numbers, and Why One of Them Lies to You

You write `3`. You write `3.6`. Looks like the same kind of thing, right? Two
numbers, one of them has a dot. On most machines you'd be correct and we could
all go home.

Not here.

> **Fletcher:** Sit down. The first thing that bites everybody on this board is
> numbers, and it bites them precisely *because* they assume numbers are
> boring. On the DX, the moment you put a dot in a number, you've changed what
> kind of number it is. Not how it prints. What it *is*. I have watched grown
> engineers lose an afternoon to this. We're going to lose ninety seconds
> instead.

CoreLX has two numeric types you'll touch constantly:

- **`int`** — a 16-bit signed whole number. Range −32768 to 32767. Every plain
  integer you write is an `int`: `3`, `1023`, `0x8010`.
- **`fixed`** — a fractional number stored in **8.8 fixed-point**. Eight bits
  for the whole part, eight for the fraction. And here is the part that trips
  people: **every decimal literal is a `fixed`.** Write `3.6`, `0.5`, `1.75`
  and you have written `fixed` values. The machine made that decision for you
  the instant you typed the dot.

`fixed` covers roughly −128.0 to +127.996, in steps of 1/256. That's your
fraction resolution: 1/256, about 0.004. Smaller than that and the DX shrugs.

Here is a small program that moves a number around. It compiles and runs:

```corelx
const SPEED = 3.6            -- has a dot, therefore fixed
const WORLD_MAX = 1023       -- no dot, therefore int

var x: fixed = 64.0
var lives: int = 3

function Start()
    x = x + SPEED            -- fixed plus fixed: the machine is content
    x = x * 0.5               -- fixed times fixed: also fine, full precision
    while true
        wait_vblank()
```

Nothing surprising yet. Now let's make it angry.

### Breaking it on purpose

You've got a speed in `fixed` and a counter in `int`. Naturally you try to
multiply them, because that's a completely reasonable thing a person would do:

```corelx
var speed: fixed = 1.5
var count: int = 3
out = speed * count          -- this does NOT compile
```

The compiler stops you cold:

```
cannot mix fixed and int in '*' — convert explicitly with int(x) or fixed(x)
```

> **Fletcher:** Good. That error is the machine doing you a favor, even though
> it doesn't feel like one. `fixed` and `int` store their bits completely
> differently — `1.5` in `fixed` is the bit pattern `0x0180`, not the number
> one-and-a-half sitting in a register being polite. If the compiler let you
> multiply them as if they were the same thing, you wouldn't get an error.
> You'd get a *wrong answer*, silently, three weeks from now, in a build you've
> already shown people. I would rather yell at you today.

### What went wrong, and how to say what you meant

You have to convert, out loud, in the direction you actually want:

```corelx
out = speed * fixed(count)   -- 1.5 * 3.0 = 4.5  (count promoted to fixed)
whole = int(speed) * count   -- 1 * 3 = 3        (speed chopped down to int)
```

`fixed(i)` turns an int into a fixed: `3` becomes `3.0`. `int(f)` throws away
the fraction and keeps the whole part: `4.5` becomes `4`. That's the entire
conversion story. Two functions, both say exactly what they do.

> **Field Notes:** The DX has no floating-point unit. None. It was never going
> to. Floating-point hardware is expensive in gates and this is a console that
> wants those gates for making things move on screen. `fixed` is the old,
> honest trick: store the fraction as a plain integer count of 1/256ths and
> agree, as a civilization, where the decimal point lives. Every racing game's
> sense of speed on hardware like this was built on exactly this idea.

> **Raccoon Engineering:** Need to divide a `fixed` by two? You *can't* divide
> `fixed` by `fixed` at all right now — the compiler rejects it outright
> (`fixed division is not implemented yet; multiply by a reciprocal constant
> instead`) rather than silently doing something wrong. Multiply by `0.5`
> instead. Want a third? Multiply by `0.333`. Reciprocals are your friend,
> they're faster anyway, and the multiply path is fully supported and keeps
> full precision.

> **Fletcher's Warning Label:** Two real limits, both of which the compiler
> will tell you about so you don't have to memorize them: (1) `fixed / fixed`
> doesn't exist yet — use the reciprocal trick above, every time, no
> exceptions. (2) Integer division (`int / int`) is **unsigned** in v1 — it's
> a raw hardware DIV instruction with no sign correction. If you're dividing
> negative `int`s and getting numbers from the upside-down, that's why. Same
> goes for `>>` (right-shift): it's always a logical shift, never arithmetic,
> so shifting a negative `int` right does not sign-extend the way you'd expect
> from most languages. Work in positives for both `/` and `>>`, or precompute
> the values you need at compile time (Chapter 2 shows you how) so the
> division/shift never has to happen on a negative number at runtime at all.

### Try This Before You Panic

Make a `fixed` variable, set it to `0.1`, and add it to itself ten times.
Print the result or peek at it in the debugger. It will **not** be exactly
`1.0`, and that is not a bug — `0.1` isn't perfectly representable in 1/256
steps, so a tiny error rides along each time. This is the single most important
thing to feel in your hands early: `fixed` is precise, but it is not *infinite*.
Knowing where it rounds is the difference between a smooth game and a jittery
one.

---

## Chapter 2 — Constants: Naming Things So You Stop Lying to Yourself

Here's a number from a real program: `1023`. What is it? The right edge of the
world? A bitmask? The price of something? You wrote it last week and now you're
staring at it like it owes you money.

> **Fletcher:** Magic numbers are how code rots. Not dramatically — quietly. One
> day `1023` means the world edge, and six months later you change the world
> size, miss one of the eleven places you typed `1023`, and now your player can
> walk through a wall on the east side of the map only. I have debugged that
> exact bug. It took an hour. It should have taken zero, because the number
> should have had a *name*.

A `const` gives a value a name that's computed once, at compile time. It costs
you nothing at runtime — no memory, no slowdown. Every place you use it, the
value gets baked straight into the program.

```corelx
const BASE = 100
const DOUBLE = BASE * 2          -- constants can build on earlier ones
const FLAGS = 0x10 | 0x02        -- full integer math and bitwise ops
const HALF_SPEED = SPEED / 2.0   -- fixed constants work too: + - * /
```

Use them for everything that's a *fact* about your game: world bounds, zone
edges, speeds, hardware register values, the number of lives you start with.
If you're typing the same number twice, it wants to be a `const`.

### Breaking it on purpose

Try to change one:

```corelx
const LIMIT = 10
function Start()
    LIMIT = 5            -- nope
```

```
cannot assign to constant LIMIT
```

> **Fletcher:** Right, and that's the *point* of the word "constant," so I'm not
> going to pretend that's a surprise. But here's the genuinely useful part:
> because constants are resolved at compile time, the machine does the math
> *before the game ever runs*. `DOUBLE = BASE * 2` isn't a multiply on the
> console — by the time your cartridge boots, `DOUBLE` is just `200`, sitting
> there, already done. Free arithmetic. Use it shamelessly.

> **Field Notes:** Remember Chapter 1 — `HALF_SPEED = SPEED / 2.0` works even
> though you can't divide `fixed` by `fixed` at *runtime*. That's because a
> constant divide happens in the compiler, on the workbench, not on the
> machine. The compiler is allowed tools the console isn't. It computes the
> answer once and hands the console a finished number. The same trick bails you
> out of the `>>`/`/` sign problem from Chapter 1: if you need `heading / 4`
> for 64 different possible headings, don't divide at runtime — compute all 64
> answers as a `const` array at compile time and look them up. Half of the
> pseudo-3D math in this guide's own demo game is built on exactly that trick.

### Try This Before You Panic

Take a program you've already got and hunt for every raw number that appears
more than once. Give each one a `const` with a name that says what it *is*, not
what it equals. `const WORLD_MAX = 1023`, not `const TEN_TWENTY_THREE = 1023`.
Future-you is the one you're writing these names for.

---

## Chapter 3 — Globals, and the Map of Where Everything Lives

You've got a score. The title screen needs it, the gameplay needs it, the
game-over screen needs it. It has to live somewhere every part of your program
can reach. That somewhere is a **global**, and it lives in WRAM — the
machine's working memory.

```corelx
var scene: int = 0           -- the compiler picks the address
var score: int               -- no initializer means it starts at 0
var energy: u8 = 255         -- u8: one byte, holds 0 to 255
var player_x: fixed = 64.0
```

A `var` at the top level — outside any function — is global. It exists for the
whole life of the game, and every function can see it. Initializers run once,
at power-on, before `Start()` does anything.

> **Fletcher:** On a lot of machines you'd be picking memory addresses by hand
> here, like an animal, writing `score` at `0x2100` and `lives` at `0x2102` and
> praying you never overlap two things. I did that for years. It is exactly as
> fun as it sounds. CoreLX does it for you now, and — this is the part I
> actually like — it writes down *where it put everything*.

### The map

The compiler allocates globals automatically, starting at WRAM address
`0x2100`. You never pick addresses, and you can never accidentally land two
variables on the same spot. And every time you build, it drops a **memory map**
file next to your ROM — `yourgame.rom.memmap` — listing every global, its
address, and its size. When you're knee-deep in the debugger at 2 a.m., that
file tells you exactly where `score` actually lives.

Three regions of WRAM are worth knowing:

| Region | Whose it is |
|---|---|
| `0x2000`–`0x20FF` | the compiler's own runtime scratch — **hands off** |
| `0x2100` upward | your globals, placed automatically |
| `0x7000`–`0x7FFF` | **yours, forever** — the compiler never touches it |

That last region matters. If you're doing something raw and clever with
`mem.*` pokes and you want memory the compiler will *never* step on, `0x7000`
to `0x7FFF` is your sandbox. Guaranteed.

### When you actually do need a specific address

Sometimes the hardware cares where something lives — a buffer you're going to
stream to a register, a table the DMA reads. You can pin a global to an exact
address:

```corelx
var dma_buffer at 0x7200: u8[96]
```

### Breaking it on purpose

Pin something somewhere stupid:

```corelx
var oops at 0x2080: int        -- inside the compiler's runtime block
```

```
global oops pinned at 0x2080 overlaps the reserved runtime block (0x2000-0x20FF)
```

> **Fletcher:** And it'll catch you the same way if you pin two things on top of
> each other, or drop a pin into the I/O registers up at `0x8000`. Pinning is a
> sharp tool. The compiler checks the blade before you grab it. Notice I pinned
> `dma_buffer` at `0x7200` — up in *your* region, not down in the auto-allocated
> pile where it'd collide with the variables the compiler is placing. Pin into
> your own sandbox, not someone else's workbench.

> **Tape Jam:** Variable not holding what you expect, and the value looks like
> garbage that *almost* makes sense? Before you blame your logic, open the
> `.memmap` file and confirm the address you're poking in the debugger is
> actually the variable you think it is. Half of all "impossible" memory bugs
> are just looking at the wrong address with great confidence.

### Try This Before You Panic

Build any program with two or three globals in it, then open the `.memmap`
file the compiler wrote. Read it. See `score` at `0x2100`, see the next one
sitting right after it. Get comfortable with that file now, while the stakes
are low — it becomes your best friend the first time something goes wrong in
memory, which it will.

---

## Chapter 4 — Arrays: When You Need a Row of the Same Thing

One score is a `var`. But sixty-four heading angles for a turning camera, or a
field of stars, or a row of tile values — those want to live together, in
order, reachable by number. That's an **array**.

```corelx
const N = 8
var table: int[8]
var palette_rows: u8[4]

function Start()
    i := 0
    while i < N
        table[i] = i * 10        -- any expression for the index
        i = i + 1
    total := table[3] + table[7] -- read them back by number
```

Arrays start zeroed — every slot is `0` until you put something there.
`int[n]` and `fixed[n]` use two bytes per slot; `u8[n]` uses one. They live in
WRAM right alongside your other globals, and they show up in the memory map.

You can also give an array its full contents up front, as a literal — this is
how you'd build a lookup table (Chapter 2's "precompute instead of divide"
trick, in practice):

```corelx
var doubles: int[5] = [0, 2, 4, 6, 8]
```

### Breaking it on purpose

Reach past the end:

```corelx
var table: int[4]
function Start()
    table[4] = 1            -- there is no slot 4; valid slots are 0,1,2,3
```

```
index 4 out of bounds for table[4]
```

> **Fletcher:** Caught at *compile time*, before the cartridge ever boots,
> because that index was a constant the compiler could check. That's the good
> case. Now, the honest part you need to hear: if your index is something the
> machine only figures out *while running* — a variable, a result of math — the
> DX does **not** check it for you. It can't afford to. This is a 16-bit
> console with frames to render; it is not going to spend cycles babysitting
> every array access. Write past the end with a runtime index and you'll
> happily stomp on whatever WRAM sits after your array, and the machine will
> let you, whistling.

> **Fletcher's Warning Label:** A constant index, like `table[4]`, gets checked
> when you build. A computed index, like `table[i]`, does **not** get checked
> when it runs. That's not laziness, it's the deal you make for speed on real
> hardware. Keep your loop bounds honest — `while i < N`, not `while i <= N` —
> and you'll never feel the missing seatbelt. Get the bound wrong and you'll
> feel it as the strangest bug of your week.

> **Raccoon Engineering:** A pre-computed table beats math every time on this
> machine. If your game keeps calculating the same handful of values —
> sines for a spin, speeds for each heading — compute them once into a `const`-
> sized array at startup (or bake the literal in directly, as above) and just
> *look them up* after that. The DX reads memory faster than it grinds
> arithmetic. Trade a little WRAM for a lot of frame time. That's the whole
> trick behind half the smooth-looking effects on hardware like this — and the
> only real workaround for the `>>`/`/`-on-negative-numbers gotcha from
> Chapter 1, since the table itself can hold pre-computed negative values
> just fine even though computing them via a shift/divide at runtime can't.

### Try This Before You Panic

Make an `int[8]`, fill it in a loop with `table[i] = i * i`, and read the
values back in the debugger using the address from your `.memmap` file. You'll
see `0, 1, 4, 9, 16, 25, 36, 49` laid out in WRAM, two bytes each. Now
deliberately change your loop to `while i <= 8` and watch it write one slot too
far. Find what it landed on in the memory map. That's the bug you're learning
to never ship.

### A note on local variables (you've been using them)

Inside a function, `:=` makes a local and figures out the type from what you
give it:

```corelx
x := 5          -- int (no dot)
speed := 2.5    -- fixed (dot)
x = x + 1       -- plain = changes something that already exists
```

Three ways to make a name, and each one tells you that name's whole life at a
glance: `:=` is local to its function, `var` is a global in WRAM, `const` is a
compile-time fact with no storage at all. You'll never have to wonder how long
a name lives — you can see it in how it was born.

---

## Chapter 5 — Loops That Count: `for i = 0 to N`

You want to do something eight times. Fill eight table slots, draw eight stars,
check eight collision boxes. You *could* write it out eight times like you're
being paid by the line. You could also set up a `while` loop with a counter and
a manual increment and an off-by-one bug waiting to happen. Or you could just
say what you mean.

```corelx
for i = 0 to 7
    table[i] = i * 10
```

That runs with `i` equal to 0, 1, 2, 3, 4, 5, 6, 7. **Eight times.** The bounds
are *inclusive* — `0 to 7` means zero through seven, both ends, exactly like you
read it out loud.

> **Fletcher:** I want to stop you on that word "inclusive" because it is the
> single most common place people miscount. `for i = 0 to 7` runs eight times,
> not seven. If you've come from machines where the loop limit means "stop
> *before* this," unlearn that here. On the DX, `to 7` includes 7. Say the
> range out loud — "zero to seven" — and count on your fingers if you have to. I
> still do.

Need to count down, or skip? Add a `step`:

```corelx
for i = 10 to 0 step -2
    -- i is 10, 8, 6, 4, 2, 0  (six times)
```

### Breaking it on purpose

The `step` has to be a number the compiler knows when it builds — a constant,
not something computed while the game runs. Try to make it a variable:

```corelx
for i = 0 to 10 step my_var      -- nope
```

```
for loop 'step' must be a constant
```

> **Fletcher:** That's deliberate, and here's the why: the compiler needs to
> know which *direction* you're counting so it knows when to stop. Counting up,
> it stops when you pass the top. Counting down, it stops when you pass the
> bottom. If your step could secretly be positive *or* negative depending on
> some variable, the compiler can't pick the right finish line, and a loop with
> the wrong finish line either stops too early or runs until the heat death of
> your cartridge. So: `step` is a constant. Pick a direction at write-time.

> **Try This Before You Panic:** Loop `for i = 0 to 5` and in the body draw the
> counter on its own line: `text.draw_int(80, 40 + i * 12, 255, 255, 255, i)`.
> The `40 + i * 12` pushes each number twelve pixels lower than the last, so you
> get a column 0,1,2,3,4,5. Count the lines — there are six, not five. Now flip
> it to `for i = 5 to 0 step -1` and watch the column come out upside down.
> Feeling the inclusive bounds with your own eyes once beats me telling you ten
> times. (`text.draw_int` is in Chapter 8 — it draws a number instead of a
> string.)

---

## Chapter 6 — Conditions, and Why There's No `else if` Yet

You've already seen `if` used bare. It also takes an `else`:

```corelx
if x < 10
    x = x + 1
else
    x = 0
```

That's the whole shape for a two-way branch. Real games need more than two
ways, though — a scene variable that's title, or playing, or game-over. Most
languages let you write `else if` to chain those. **CoreLX cannot do that yet,
and it will not tell you it failed.**

### Breaking it on purpose (the dangerous one)

```corelx
if scene == SCENE_TITLE
    x = 1
else if scene == SCENE_PLAYING     -- looks completely normal
    x = 2
```

This **compiles with no error or warning**. But the `if scene == SCENE_PLAYING`
does not become part of the `else` branch the way it would in almost any other
language. It's parsed as an unconditional statement that runs every single
pass, regardless of what the outer `if`/`else` decided. If `x = 2` is supposed
to only happen when you're NOT on the title screen, you've just written a bug
that silently fires all the time.

> **Fletcher:** This is the one that'll get you, because everything about it
> *looks* fine. It reads fine, it compiles clean, and the bug it produces
> doesn't look like a syntax problem — it looks like your scene logic itself is
> broken, so that's where you'll go looking. You'll spend an hour staring at
> your state machine before you think to suspect the `else if` itself. I'm
> telling you now so you skip that hour.

The one-word `elseif` keyword doesn't save you either — it currently fails to
compile outright (`Expected statement`). Annoying, but at least it's honest
about failing.

### The only form that works today

Write `else` alone, on its own line, with the next `if` fully nested one
indent level deeper:

```corelx
if scene == SCENE_TITLE
    x = 1
else
    if scene == SCENE_PLAYING
        x = 2
    else
        if scene == SCENE_GAMEOVER
            x = 3
```

It's more indentation than you're probably used to for a multi-way branch, but
it's the reliable form — verified directly against the compiler, not assumed.
This is a known compiler limitation, not a style preference; expect `else if`
to start working the way you'd expect once a future compiler pass fixes it.
Until then, every multi-way branch in this guide (and in the demo game) uses
the nested form above.

### `break` and `continue`

Inside a `while` or `for` loop, `break` exits the loop immediately, and
`continue` skips the rest of the current iteration and moves on to the next
one. In a `for` loop, the loop variable still advances when you `continue` —
it never skips that part.

```corelx
-- Sums 0..9, skipping 5 and stopping dead at 8:
--   0+1+2+3+4+6+7 = 23
for i = 0 to 9
    if i == 5
        continue          -- skip just this iteration
    if i == 8
        break              -- stop the loop entirely
    total = total + i
```

That's `break_continue.corelx` from the demo programs (Part 2) — run it and
watch the HUD confirm `23`, not `45` (what you'd get if you never skipped or
stopped at all) and not `28` (what you'd get if `break` fired where `continue`
should have).

> **Why This Matters:** Most game code is just "read input -> update variables
> -> draw state" repeated every frame. Conditions decide *what* updates; loops
> decide *how many times*. Getting both exactly right, including their sharp
> edges, is most of what separates code that works from code that almost does.

---

## Chapter 7 — Structs: Beyond `Sprite()`

You've seen `Sprite()` already if you've looked ahead at any sprite code — it's
a built-in struct CoreLX gives you for OAM sprite records. But you aren't
limited to it. Declare your own with `struct Name:` followed by an indented
list of `field: type` lines:

```corelx
struct Player:
    x: int
    y: int
    lives: int
```

`Player()` creates one, and you read/write its fields with `.`:

```corelx
hero := Player()
hero.x = 100
hero.y = 50
hero.lives = 3
```

### Structs are reference types

Pass a struct to a function and the function shares the *same* struct — its
edits are visible to the caller when the function returns. There's no `&`, no
pointers, because CoreLX has no unary address-of operator at all; you never
write `&hero`, you just pass `hero`:

```corelx
function damage(p: Player)
    p.lives = p.lives - 1

function Start()
    hero := Player()
    hero.lives = 3
    damage(hero)
    damage(hero)
    -- hero.lives is now 1, not 3 -- damage() shared the same struct
```

Function parameters need an explicit type — `function damage(p: Player)`, not
`function damage(p)` — the same way `var`/`asset` declarations do.

> **Fletcher:** If you've used a language with real pointers, "reference type,
> no address-of operator" might look like a contradiction. It isn't — it just
> means the *only* way to hand a struct to something is to hand over the real
> thing, never a copy and never a raw address you could get clever with. You
> can't accidentally alias two unrelated structs by messing up a pointer,
> because there's no pointer to mess up. You also can't do certain low-level
> tricks people do with pointers on other machines. For a 16-bit console
> that's a trade I'll take every time — most of those tricks are exactly the
> kind of thing that turns into a 2 a.m. debugging session.

See `structs.corelx` in the demo programs (Part 2) for the full, verified
version of this example, including how to check a struct field's value from
outside the program (a local struct doesn't get a fixed WRAM address the way a
`var` global does, so the demo mirrors `hero.lives` out to a small global
before entering its main loop — the same thing you'd do to feed a value to a
HUD anyway).

---

## Chapter 8 — Putting Words on the Screen

Your game runs. Brilliant. But it's a black void, and the only person who knows
anything is happening is you, squinting at a debugger. Time to put something on
the glass. Start with text, because text is how you say `SCORE`, `GAME OVER`,
and `PRESS START` — the words every game needs.

```corelx
text.draw(40, 80, 255, 255, 255, "HELLO NITRO")
```

Six arguments: X, Y, then **three separate color numbers** — red, green, blue,
each 0 to 255 — and finally the string. So `255, 255, 255` is white, `255, 0, 0`
is red, and so on. The text appears at pixel (40, 80) and marches to the right,
eight pixels per character.

> **Fletcher:** I know what you're about to ask. "Why three color numbers?
> Every other system lets me pass one color." Because the DX's text port has
> three separate eight-bit channels — red, green, blue — sitting at three
> separate hardware addresses, and a single number on this machine is sixteen
> bits. Sixteen bits cannot hold three eight-bit channels. The math doesn't
> fit. So instead of lying to you with a fake "color" that secretly loses
> information, CoreLX hands the port exactly what it wants: R, G, B, each on its
> own. It's one more number to type and zero surprises later. I'll take that
> trade every day.

> **Field Notes:** When you write a string, the characters stream out to a
> single hardware register one at a time, and the port quietly advances the
> cursor eight pixels after each one. That's why your text flows left to right
> without you tracking position — the *port* is keeping the cursor, not you.

### Breaking it on purpose

Strings are special on the DX. They're **labels**, not a data type you can
store and shuffle around. Try to put one in a variable:

```corelx
var name: int = 0
name = "PLAYER"          -- the compiler stops you
```

```
strings can only be used directly as a text.draw argument in v1
```

> **Fletcher:** Right, and before you grumble — this is a 16-bit console, not a
> word processor. v1 strings exist to *label things on screen*: scores, menus,
> the word "PAUSED." They are not a place to store the player's name and do
> clever text manipulation. For now: a string goes straight into `text.draw`,
> full stop.

### Numbers on the screen

A string is a fixed label. But a *score* changes — it's a number that lives in a
variable and goes up. For that, there's a second function:

```corelx
var score: int = 0

function Start()
    score = 1230
    while true
        wait_vblank()
        text.draw(120, 80, 255, 255, 255, "SCORE")
        text.draw_int(140, 100, 255, 255, 0, score)
```

`text.draw_int` takes the same first five arguments as `text.draw` — X, Y, and
the three color channels — but the last argument is a **number**, not a string.
It prints it as digits: `1230` shows up as `1230`. Negatives get a minus sign,
and leading zeros are dropped, so `42` prints as `42`, not `00042`.

> **Tape Jam:** Text not showing up? Two usual culprits. One: you drew it once,
> at startup, but the screen clears every frame and your text only lived for
> that first frame — draw it *every* frame, inside your loop, if you want it to
> stay. Two: you drew it at a coordinate off the edge of the 320×200 screen,
> where it is rendering perfectly into the void.

---

## Chapter 9 — Reading the Controller

A game nobody can control is a screensaver. Let's read the pad. On the DX the
pattern is always the same three beats: **poll once per frame, then ask
questions.**

```corelx
function Start()
    while true
        wait_vblank()
        input.poll()                 -- read the controller, once, up top
        if input.held(LEFT)
            -- move left while LEFT is down
        if input.pressed(A)
            -- fire ONCE, the instant A goes down
```

`input.poll()` reads the controller hardware and remembers it. Then:

- `input.held(BUTTON)` is true *while* the button is down — for movement, where
  you want continuous action.
- `input.pressed(BUTTON)` is true only on the **single frame** the button goes
  from up to down — for actions you want to happen *once* per press: firing,
  jumping, confirming a menu.
- `input.released(BUTTON)` is the opposite edge — true the frame a button comes
  back up.

The buttons have names you just use: `UP DOWN LEFT RIGHT A B X Y L R START Z`.

> **Fletcher:** The difference between `held` and `pressed` is the thing that
> separates a game that feels right from one that feels broken, so let me make
> it stick. Use `held` for walking: you hold left, you keep walking left, frame
> after frame. Use `pressed` for firing: you press A, *one* shot comes out, and
> you don't get another until you let go and press again. If you wire your fire
> button to `held` by mistake, the player taps A once and unloads forty bullets
> in two-thirds of a second because the button was "down" for forty frames.
> I have shipped that bug. The playtesters called it "the machine gun glitch."
> It was not a feature.

> **Field Notes:** `input.pressed` has to know what the button was doing *last*
> frame to spot the edge. CoreLX remembers that for you in a corner of memory it
> owns and you never see — but only correctly if you `poll()` exactly once per
> frame, at the top of your loop. Poll twice and you'll smear two reads
> together; forget to poll and you're reading a stale frame. Once per frame, up
> top. Make it a habit you don't think about.

### Breaking it on purpose

Wire a fire button to the wrong question:

```corelx
input.poll()
if input.held(A)
    spawn_bullet()      -- a bullet EVERY frame A is down: the machine gun glitch
```

Hold A for one comfortable second and you've spawned sixty bullets. The fix is
one word — `held` becomes `pressed` — and now it's one bullet per press, the way
a person expects.

> **Raccoon Engineering:** You can build a "tap to start, hold to fast-forward"
> control out of these two cheaply: `pressed` triggers the first action
> immediately, and `held` (maybe gated behind a frame counter — Chapter 10)
> takes over if they keep the button down. Menus that advance once on tap but
> scroll when held are built on exactly this pair.

> **Try This Before You Panic:** Make a global `var count: int = 0`. In your
> loop, `poll()`, then `if input.pressed(A)` add one to `count`, and draw it
> with `text.draw_int`. Mash A and watch it climb by exactly one per press. Now
> change `pressed` to `held` and hold A — watch it rocket upward. That runaway
> number *is* the machine gun glitch, and now you'll recognize it on sight.
> (This is `counter.corelx` in the demo programs — a complete, working version
> is waiting for you.)

---

## Chapter 10 — The Game Loop: Frames, VBlank, and `wait_vblank()`

Every CoreLX program you've seen so far ends in the same shape:

```corelx
while true
    wait_vblank()
    -- read input, update state, draw
```

`wait_vblank()` pauses your program until the next VBlank — the brief window
between video frames where it's safe to update what's about to be drawn.
Calling it once per loop is what makes your game run at one update per real
video frame, instead of running as fast as the CPU possibly can (which would
make your game speed depend on how fast the machine happens to be, not on
anything you control).

### Breaking it on purpose

Here's the part that isn't obvious: VBlank isn't an instant, it's a *window* —
the flag that says "we're in VBlank" stays set for a whole scanline's worth of
time, not just one tick. That means a loop like this:

```corelx
while true
    wait_vblank()
    x = x + 1
```

can, under the wrong conditions, run its body **more than once** for the same
real video frame — `wait_vblank()` returns immediately again if VBlank is still
active, before the next real frame has actually started. Held-input logic
(walking, turning) isn't self-correcting the way `input.pressed` edge-detection
is, so if this fires twice in one real frame, your player moves twice as fast
as you intended, unpredictably, depending on exact timing.

> **Fletcher:** This one's sneaky because it doesn't happen every frame, and it
> doesn't happen the same amount every time it does happen — which makes it
> look like "sometimes movement feels floaty" rather than a specific,
> nameable bug. It's the same root cause every time, though: your loop body ran
> twice for one frame's worth of real time.

### The fix: a frame-counter debounce

Every per-frame loop body in this guide's demo game uses this exact pattern:

```corelx
var last_frame: int = 0

function Start()
    last_frame = frame_counter()
    while true
        while frame_counter() == last_frame
            wait_vblank()
        last_frame = frame_counter()
        -- now this runs exactly once per real video frame
```

`frame_counter()` is a running count of real video frames that have elapsed.
The inner `while` loop keeps calling `wait_vblank()` until the frame counter
has actually ticked forward — so no matter how many times VBlank's flag lets
`wait_vblank()` return inside one real frame, your loop body below it only
ever runs once per frame. This is the same pattern the music player and the
demo game's movement code both use.

> **Fletcher's Warning Label:** You don't need this debounce for a program
> that's *only* drawing static text or reading `input.pressed` (edge-triggered
> input is naturally safe against running twice — a press either happened or
> it didn't). You need it the moment you're accumulating state every frame:
> position from `input.held`, a counter, an animation timer. If your movement
> "feels too fast" or "sometimes jumps," check this before you check anything
> else.

### Before `Start()`: `__Boot()` and the Splash Screen

Every compiled program shows the Nitro-Core-DX boot logo — a brief slide-in
and hold — before `Start()` ever runs. That happens because the compiler
quietly injects a default `__Boot()` function that calls
`boot.show_default()` (the stock slide+hold sequence) and then calls
`Start()`. You never see this unless you go looking for it.

You can take over the entry point yourself by defining your own `__Boot()`:

```corelx
function __Boot()
    input.poll()
    if input.held(A)
        secret_mode = 1
    Start()
```

Defining `__Boot()` replaces the default entirely — the stock splash will
**not** show unless you call it yourself. If you still want the stock
presentation before doing your own thing, call `boot.show_default()` as the
first line of your own `__Boot()`:

```corelx
function __Boot()
    boot.show_default()   -- stock slide+hold, then continue
    input.poll()
    if input.held(A)
        secret_mode = 1
    Start()
```

> **Watch Out:** A custom `__Boot()` runs *before* `Start()`'s own setup —
> `wait_vblank()`/`input.poll()` work identically there, but nothing you set
> up in `Start()` exists yet. Most programs never need a custom `__Boot()` at
> all; it exists for the rare case where you want to read input or make a
> decision before the game's normal setup even begins.

---

## Chapter 11 — Modules: Sharing Code With `--!`

`--!` lines are directives, not comments — they're only legal at the very top
of the file, before any code, and declare things about the file itself:

```corelx
--! corelx 1.0
--! modules: anim, sfx
```

- `--! corelx <version>` records which CoreLX version the file targets.
- `--! modules: name, name, ...` pulls in one or more modules — plain
  `.corelx` files that live in a `modules/` folder next to your project.
  Functions inside a module are called the same way as builtins, namespaced
  by the module's name.

A module is just a normal CoreLX file — functions, and any `const`/`var`
declarations those functions need — nothing special about its own syntax. If a
named module isn't found in the `modules/` folder, you get a clear error
(`module 'name' not installed`) rather than a confusing "unknown function" at
the call site. An unrecognized directive is also a compile error, not a
silently-ignored line.

Two modules ship with the project today:

### The `anim` Module (Sprite Animation)

`anim` (`modules/anim.corelx`) handles the two genuinely reusable parts of
sprite animation: frame timing and mirroring. Frame lists themselves stay as
plain array constants in your own code — a module can only index arrays
declared in the same file:

```corelx
--! modules: anim

const WALK_FRAME_COUNT = 4
var walk_frames: int[4] = [1, 2, 3, 4]

function Start()
    hero := Sprite()
    while true
        wait_vblank()
        idx := anim.frame_index(WALK_FRAME_COUNT, 8)  -- new frame every 8 ticks
        hero.tile = walk_frames[idx]
        anim.set_mirror(hero, 0)                      -- 1 to flip horizontally
```

`anim.frame_index(frame_count, ticks_per_frame)` returns which frame (0 to
`frame_count - 1`) should be showing right now, looping back to 0 after the
last one. `anim.set_mirror(sprite, mirror)` sets or clears horizontal flip —
useful for getting a second direction (e.g. "walk right") out of frames you
only drew once (e.g. "walk left"). See `anim_module.corelx` in the demo
programs for the complete, verified version.

### The `sfx` Module (Sound Effect Triggers)

`sfx` (`modules/sfx.corelx`) wraps a free FM channel's key-on/key-off for
quick one-shot sound effects:

```corelx
--! modules: sfx

function Start()
    while true
        wait_vblank()
        input.poll()
        if input.pressed(A)
            sfx.play(0)   -- key-on channel 0
```

> **Watch Out:** `sfx` deliberately does not set pitch or instrument — it's a
> key-on/key-off convenience wrapper only. Set those up through the low-level
> `ym.write` register path (Chapter 14) before triggering the effect, or wait
> for a future pass that gives `sfx` a real pitch/instrument argument.

---

## Chapter 12 — Sprites and OAM

A sprite is a small, independently-positioned image the PPU can move around
without redrawing the background underneath it — the classic "one moving
object" primitive every 2D console has had since the beginning. CoreLX gives
you a built-in `Sprite()` struct shaped exactly like one OAM (Object
Attribute Memory) record.

```corelx
box := Sprite()
sprite.set_pos(box, 152, 92)
box.tile = tile_base
box.attr = SPR_PAL(1) | SPR_PRI(0)
box.ctrl = SPR_ENABLE() | SPR_SIZE_16()

oam.write(0, box)
oam.flush()
```

- `sprite.set_pos(sprite, x, y)` sets screen position.
- `.tile` is the base tile index (from `gfx.load_tiles`, Chapter 15).
- `.attr` packs palette, priority, and flip bits — build it from the helper
  functions below, combined with `|`.
- `.ctrl` packs enable and size bits, same idea.
- `oam.write(index, sprite)` copies the struct's fields into OAM slot `index`.
- `oam.flush()` finalizes the write so it actually reaches the PPU.

Just like every other struct (Chapter 7), pass `box` itself — never `&box`.
CoreLX has no address-of operator, and `Sprite()` is no exception to that.

### Sprite Helper Functions

- `SPR_PAL(n)` — sprite palette select bits
- `SPR_PRI(n)` — sprite priority bits (0-3; see the note below)
- `SPR_ENABLE()` — enable bit
- `SPR_SIZE_8()` / `SPR_SIZE_16()` — 8x8 or 16x16 sprite
- `SPR_HFLIP()` / `SPR_VFLIP()` — horizontal/vertical flip
- `SPR_BLEND(mode)` / `SPR_ALPHA(a)` — blend mode and alpha bits

> **Why This Matters — Priority:** Sprites and background layers (Chapter 13)
> share one compositing order, sorted by priority (`SPR_PRI(n)` for sprites,
> `bg.set_priority(layer, n)` for backgrounds) — whichever has the *higher*
> number is drawn *later*, which means it ends up on top. If your sprite
> vanishes behind a background layer you were sure was "behind" it, check both
> priorities: a background left at a higher priority number than your sprite
> will paint right over it every frame.

### Moving It Around

Reading the controller (Chapter 9) and moving a sprite is the same `held` +
clamp pattern you've already seen for a camera:

```corelx
if input.held(RIGHT)
    box_x = box_x + SPEED
if box_x > 312
    box_x = 312
```

See `sprite.corelx` in the demo programs (Part 2) for the complete, verified
version — D-pad moves an 8x8 box around the screen, clamped so it never runs
off the edge.

### OAM Notes

- `oam.write(index, sprite)` writes one sprite record from a `Sprite()`
  variable.
- `oam.write_sprite_data(id, x, y, tile, attr, ctrl)` writes the same record
  from plain values instead, when you don't want to keep a `Sprite()` variable
  around.
- `oam.clear_sprite(id)` disables a sprite by zeroing its control byte.
- `oam.flush()` is the write-finalization call every one of the above needs
  before the change actually reaches the PPU.

---

## Chapter 13 — Matrix Planes: Floors and Billboards

Nitro-Core-DX has a dedicated matrix-plane model in the emulator/runtime,
separate from ordinary BG tilemap usage. This is the machinery behind the
pseudo-3D floor in `floor.corelx` (Part 2) and behind vertical "billboard"
objects — buildings, NPCs, anything that should stand upright on that floor
and scale correctly as you approach it.

### Why This Exists

A BG tilemap layer is great for a flat, 2D scrolling background. It is not
built to answer "what does this floor look like from a moving, turning
camera, projected in perspective." Matrix planes exist specifically for that:
each matrix-capable layer can bind to a transform channel that sources from
its own tilemap memory, pattern memory, *or* a dedicated bitmap, and can drive
a generic per-plane projection (flat/affine, perspective floor, or vertical
billboard) instead of a plain scroll.

### Current Matrix Plane Capabilities

- Plane sizes: 32x32, 64x64, 128x128 (in 8x8 or 16x16 tiles)
- Source modes: tilemap/pattern-backed, or bitmap-backed (for large imported
  images that don't fit the tile-backed plane's 256-tile index ceiling)
- Projection modes: `0` none/manual rows, `1` perspective row projection
  (the floor), `2` vertical projected quad (a billboard)
- Outside-plane behavior: wrap, backdrop color, tile 0, or clamp

### The Recommended High-Level Path

```corelx
matrix_plane.set_projection(channel, mode, horizon)   -- mode: 0 none, 1 perspective rows, 2 vertical quad
matrix_plane.set_depth(channel, base_distance, focal_length, width_scale)
matrix_plane.set_camera(channel, x, y, heading_x, heading_y)
matrix_plane.set_surface(channel, origin_x, origin_y, facing_x, facing_y)  -- vertical quads only
```

Vertical projected quads are treated as real world-space planes, not
screen-facing billboards in the Doom sense: the renderer intersects the camera
ray with the plane defined by `ORIGIN_*`/`FACING_*`, the bottom of the quad is
anchored to the ground position `ORIGIN_*` represents, and off-angle views
narrow and foreshorten realistically instead of always facing the camera.

### Pitfall: Every Plane In A Scene Must Share One Camera-Eye

If a scene combines a perspective floor with one or more vertical-billboard
planes (a building, an NPC, any object standing "on" that floor), every one of
those planes' `matrix_plane.set_camera(channel, x, y, heading_x, heading_y)`
calls must be fed the **exact same** `x, y` position for a given frame — not
just the same *player* position, but the same fully-computed camera-eye
position, including any camera trick layered on top of it.

This bit a real demo directly. A common technique for a walking-around game is
a "feet pivot": instead of rendering the floor from the player's raw position,
the floor's camera trails the player by a fixed world-unit offset in the
direction opposite of facing —

```corelx
eye_x = cam_x - pivot_x[heading_index]
eye_y = cam_y - pivot_y[heading_index]
matrix_plane.set_camera(0, eye_x, eye_y, heading_x[heading_index], heading_y[heading_index])
```

— so that turning visually pivots around the character's own feet/screen
position instead of around the floor's raw coordinate. The mistake is
assuming a billboard object (a building, say) should track the player's *raw*
position instead, on the reasoning that "the billboard should scale off the
real player position, not some camera trick." That reasoning sounds
principled, but it's wrong: it silently makes the billboard render from a
**different eye position than the floor**, and since the pivot offset itself
rotates with heading, the mismatch between the two eyes also rotates —
visually, the billboard appears to drift/slip across the floor as the camera
turns, even though the object's own world position never changed. It looks
exactly like a physics bug (the building "isn't anchored to the ground
right"), but the actual cause is a camera inconsistency between planes, not
anything wrong with the billboard's own placement math.

**The fix**: compute the camera-eye position once per frame, and pass that
same value to `matrix_plane.set_camera` for every plane in the scene that's
meant to share one coherent 3D space — the floor, every billboard standing on
it, and, if applicable, the audio/gameplay logic that also cares about "where
the camera is." Never let one plane use a raw position while a sibling plane
in the same scene uses an adjusted one, even if the adjustment seems purely
cosmetic (like a feet-pivot offset). If you have multiple independent scenes
(say, an outdoor overworld and an interior room), each scene needs its own
consistently-shared eye — don't mix eyes from different scenes either.

```corelx
-- WRONG: billboard uses a different eye than the floor it stands on.
matrix_plane.set_camera(0, cam_x - pivot_x[h], cam_y - pivot_y[h], heading_x[h], heading_y[h])
matrix_plane.set_camera(1, cam_x, cam_y, heading_x[h], heading_y[h])

-- RIGHT: compute the eye once, feed it to every plane in the scene.
eye_x = cam_x - pivot_x[h]
eye_y = cam_y - pivot_y[h]
matrix_plane.set_camera(0, eye_x, eye_y, heading_x[h], heading_y[h])
matrix_plane.set_camera(1, eye_x, eye_y, heading_x[h], heading_y[h])
```

If you actually *want* an object to visually behave differently from the
floor (e.g. a HUD-anchored object that should never move relative to the
screen), that's a real design choice — but it means the object isn't meant to
occupy the same 3D space as the floor at all, and shouldn't be using
`matrix_plane.set_camera`'s world-position semantics for it.

### Low-Level MMIO Programming Path

If you are writing low-level ROM code, the dedicated matrix-plane aperture is:

- `0x8080` `MATRIX_PLANE_SELECT` — selects plane `0-3`
- `0x8081` `MATRIX_PLANE_CONTROL` — bit 0 enable; bits [2:1] size (`0`=32x32,
  `1`=64x64, `2`=128x128); bit 3 source mode (`0`=tilemap/pattern,
  `1`=bitmap); bits [7:4] bitmap palette bank
- `0x8082`/`0x8083` `MATRIX_PLANE_ADDR_L/H` — tilemap upload address
- `0x8084` `MATRIX_PLANE_DATA` — writes one tilemap byte, auto-increments
- `0x8085`/`0x8086` `MATRIX_PLANE_PATTERN_ADDR_L/H`
- `0x8087` `MATRIX_PLANE_PATTERN_DATA` — writes one pattern byte, auto-increments
- `0x8088`-`0x808A` `MATRIX_PLANE_BITMAP_ADDR_L/M/H`
- `0x808B` `MATRIX_PLANE_BITMAP_DATA` — writes one bitmap byte, auto-increments
- `0x808C` `MATRIX_PLANE_FLAGS` — bit 0 palette-index-0-is-transparent, bit 1
  visible from both sides
- `0x808D` `MATRIX_PLANE_ROW_CONTROL` — bit 0 row mode enabled
- `0x808E`/`0x808F` `MATRIX_PLANE_ROW_ADDR_L/H`
- `0x8090` `MATRIX_PLANE_ROW_DATA` — writes one row-parameter byte
- `0x8091` `MATRIX_PLANE_PROJECTION_CONTROL`
- `0x8092` `MATRIX_PLANE_HORIZON`
- `0x8093`-`0x8096` `MATRIX_PLANE_CAMERA_X/Y_L/H`
- `0x8097`-`0x809A` `MATRIX_PLANE_HEADING_X/Y_L/H` (8.8 fixed forward vector)
- `0x809B`/`0x809C` `MATRIX_PLANE_BASE_DISTANCE_L/H`
- `0x809D`/`0x809E` `MATRIX_PLANE_FOCAL_LENGTH_L/H`
- `0x809F`/`0x80A0` `MATRIX_PLANE_WIDTH_SCALE_L/H`
- `0x80A1`-`0x80A4` `MATRIX_PLANE_ORIGIN_X/Y_L/H` (vertical quads)
- `0x80A5`-`0x80A8` `MATRIX_PLANE_FACING_X/Y_L/H` (8.8 fixed, vertical quads)
- `0x80A9`/`0x80AA` `MATRIX_PLANE_HEIGHT_SCALE_L/H`

#### Row Table Layout

Each visible scanline has a 16-byte row record: bytes 0-3 `StartX`, 4-7
`StartY`, 8-11 `StepX`, 12-15 `StepY`. There are 200 visible scanlines, so one
plane's row table is 3200 bytes.

#### Typical Upload Sequence

1. select the matrix plane
2. configure its size and enable bit
3. choose source type
4. if tile-backed: upload tilemap bytes (`0x8082-0x8084`), then pattern bytes
   (`0x8085-0x8087`)
5. if bitmap-backed: upload bitmap bytes (`0x8088-0x808B`)
6. if row-driven: upload row parameters (`0x808E-0x8090`)
7. bind a visible layer to that transform channel, and enable matrix mode on
   that channel

```text
write8(0x8080, 0)      ; plane 0
write8(0x8081, 0x05)   ; enable + 128x128

write8(0x8082, 0x00)
write8(0x8083, 0x00)
for each tilemap byte
    write8(0x8084, byte)

write8(0x8085, 0x00)
write8(0x8086, 0x00)
for each pattern byte
    write8(0x8087, byte)
```

Bitmap-backed upload:

```text
write8(0x8080, 0)      ; plane 0
write8(0x8081, 0x1D)   ; enable + 128x128 + bitmap source + palette bank 1

write8(0x8088, 0x00)
write8(0x8089, 0x00)
write8(0x808A, 0x00)
for each packed bitmap byte
    write8(0x808B, byte)
```

Row-mode upload:

```text
write8(0x8080, 0)      ; plane 0
write8(0x808D, 0x01)   ; row mode enabled

write8(0x808E, 0x00)
write8(0x808F, 0x00)
for each row-table byte
    write8(0x8090, byte)
```

#### Pattern Memory Format

Same packed 4bpp tile format as normal tile data: `8x8` tile = 32 bytes,
`16x16` tile = 128 bytes. The layer's tile-size setting still controls how the
matrix renderer interprets the pattern data — if BG0 is `8x8`, your dedicated
matrix plane patterns must be authored as `8x8` tiles too.

#### Bitmap Plane Memory Format

Packed indexed 4bpp pixels: two pixels per byte, high nibble = even pixel,
low nibble = odd pixel, palette bank from `MATRIX_PLANE_CONTROL[7:4]`.
Bitmap-backed planes are the direct validation path for large imported images
that don't fit cleanly into the tile-backed plane's 256-tile index ceiling.

#### Tilemap Entry Format

Each tilemap entry is byte 0 = tile index, byte 1 = attributes (palette low
bits + flip bits).

#### Outside Behavior

The transform channel's matrix-control register defines outside behavior: `0`
= wrap (classic repeating floor), `1` = backdrop, `2` = tile 0, `3` = clamp
(hold the edge instead of repeating).

### Practical Advice

- Use dedicated matrix planes for large rotated/scaled backgrounds; keep
  ordinary BG tilemaps for conventional HUD/background work.
- Start with `128x128 @ 8x8` when you want a true `1024x1024` source plane.
- **The plane's canvas size should match your world's coordinate range 1:1.**
  A bitmap import stretches the source image to fill the *entire* plane
  canvas rather than padding it — if your plane is smaller than the range
  your camera actually walks over, the texture will visibly tile/repeat well
  before you reach the edges of your intended world.
- Don't hand-pack tilemap/pattern uploads unless you're explicitly doing
  low-level hardware work — use the high-level path above.

---

## Chapter 14 — Audio: Music and Sound Effects

Nitro-Core-DX's audio subsystem is the **YM2608 / OPNA** chip (FM, SSG,
rhythm, ADPCM), driven through a hardware/MMIO host interface at
`0x9100-0x91FF`. CoreLX's `music.*` built-ins are the real, primary,
emulator-tested way to play music from a CoreLX program.

### The `music.*` Built-ins (Song Playback)

A music asset is a compiled `.ncdxmusic` stream (built with
`cmd/vgm_to_ncdxmusic`, or written directly) declared like any other asset:

```corelx
asset Theme: music "theme.ncdxmusic"
asset Fanfare: music "fanfare.ncdxmusic"

function Start()
    music.play_loop(Theme)
    while true
        wait_vblank()
```

- `music.play(asset)` plays once and stops (silences the chip and clears
  playback state when the song ends).
- `music.play_loop(asset)` plays and wraps back to the start forever.
- `music.play_jingle(asset)` stashes whatever's currently playing (including
  "nothing"), plays the given song once, then restores exactly what was
  playing before — frame index and all, not restarted from the top. Use it
  for a one-off sting (level-clear fanfare, item pickup) over a looping BGM
  track.
- `music.stop()` silences the chip immediately and clears playback state.
- `music.set_volume(level)` sets output volume (0-255) immediately.
- `music.fade_to(level, frames)` ramps volume to `level` over `frames` real
  frames — call it once, the ramp runs on its own each `wait_vblank()`.

All of these drive off `wait_vblank()` (Chapter 10) — the per-frame advance
only happens when your code calls `wait_vblank()`, the same place every other
per-frame system (input, animation) already expects to run.

### Sound Effects: The `sfx` Module

For one-shot sound effects triggered by gameplay events (a hit, a pickup, a
jump), use the `sfx` module (Chapter 11) rather than `music.*` — it's built
for quick key-on/key-off triggers on a free FM channel, not full song
playback. Remember it doesn't set pitch/instrument on its own yet; wire that
up through the low-level `ym.write` register path first if your effect needs
a specific tone.

### Low-Level FM Access

Advanced FM sound design — instrument definition, pitch, per-voice register
tweaking — is done through `ym.write`/`ym.write_port1` (the low-level YM2608
register escape hatch) or raw ROM code against the `0x9100-0x91FF` host
registers directly, until a higher-level CoreLX instrument API exists.

### Legacy `apu.*` Built-ins (Being Phased Out)

CoreLX still exposes an older 4-channel synth (`apu.enable`,
`apu.set_channel_wave/freq/volume`, `apu.note_on/off`) from before the YM2608
migration. **Treat these as legacy** — they target a different, non-final
synth and exist only so old code keeps compiling. New programs should use
`music.*` and `sfx`/`ym.*` instead.

---

## Chapter 15 — Assets and the `.ncdx` Project Format

### Why your art isn't *in* your code

You write code as text. You'd think a picture could be text too — and it can,
technically — but a single floor image is over a hundred thousand characters of
hex. Paste that into your `main.corelx` and you'd scroll past a wall of
gibberish for ten minutes to find your `while` loop. So we don't.

Instead, each image is its own file — a **`.cxasset`** — sitting next to your
code inside the project. You make one by running an image through the
importer, which converts your PNG into the DX's format once:

```
corelx_import  park.png  ParkFloor  32  1  park_floor.cxasset
```

Then in code you just *name* it and use it:

```corelx
asset ParkFloor: image "park_floor.cxasset"

function Start()
    matrix_plane.load_bitmap(ParkFloor, 0)
    ...
```

> **Fletcher's Warning Label:** The compiler is strict about the project and the
> code agreeing, in *both* directions, and it will refuse to build if they
> don't. Reference a `.cxasset` that isn't in the project? Error. Leave a
> `.cxasset` in the project that no code uses? Also an error — a "you forgot to
> wire this up, or you forgot to delete it" error. No mystery files, no dead
> weight.

### Tile Assets (Inline)

Small assets can be written directly inline as hex, right in your source:

```corelx
asset BoxTile: tiles16
    hex
        11 11 11 11 11 11 11 11
        ...
```

Supported encodings today: `hex` (most common, used throughout this guide and
its tests), `b64`, and `text`. Assets are referenced in code via a generated
constant: `ASSET_BoxTile`.

### What a Game Actually Is on Disk

A whole game is a single file: **`MyGame.ncdx`**. Open one in the Studio and
you see a project. Look at it in your file browser and you see one icon, one
file. That's deliberate — a game should be one thing you can hand to a friend,
not a folder of loose pieces to lose track of.

> **Fletcher:** Here's the honest truth under the hood, because you'll want to
> know it the first time something goes weird: a `.ncdx` is a **zip file**
> wearing a different hat. Inside it there's your `main.corelx`, your image
> assets, and a little `project.toml` with the title and such. The Studio packs
> and unpacks it for you so you never think about it. But it's a zip. Remember
> that.

### Editing the Guts Like an Admin Boss

Ninety-nine times out of a hundred, you let the Studio handle the `.ncdx` and
you never touch the zip. But sometimes you need to get in there:

1. Make a copy (always work on a copy when going in by hand).
2. Rename `MyGame.ncdx` → `MyGame.zip`.
3. Unzip it. There's your `main.corelx`, your `.cxasset` files, your
   `project.toml`.
4. Do your surgery.
5. Zip the *contents* back up (the files at the root of the zip, not a folder
   containing them — that trips people up).
6. Rename `.zip` → `.ncdx`.

> **Fletcher:** Mind the one rule that bites everyone: when you re-zip, the
> files go at the **top level** of the archive — `main.corelx` should be right
> there when you open the zip, not buried inside a `MyGame/` folder. The
> compiler looks for `main.corelx` at the root.

> **Raccoon Engineering:** Because a `.ncdx` is just a zip, every tool you
> already own works on it — git can store it, a script can rip an asset out of
> a hundred of them, a diff tool can show you what changed between two builds
> (work on the unzipped folders for that; zips diff badly).

---

# Part 2 — Building a Game

Everything in Part 1 is a piece. This part is where the pieces become a game —
following one real project, built feature by feature, each one landing here
only once it's actually built and verified running on the emulator.

## The Demo Programs

These aren't sketches — they live in `docs/manual_examples/`, and the test
suite compiles and runs every one of them against the real emulator on every
build. If a demo here ever stopped working, the build would break.

### `hello.corelx` — words on the glass

The smallest complete program that shows you something. Run it: cyan
`HELLO NITRO` near the middle of the screen.

```corelx
function Start()
    while true
        wait_vblank()
        text.draw(96, 96, 64, 220, 255, "HELLO NITRO")
```

Notice the text is drawn *inside* the loop — draw it once outside and it
flashes for a single frame and vanishes, because the screen clears every frame
(Chapter 8's Tape Jam, made real).

### `counter.corelx` — a number that counts

Press A, the number goes up by one. Hold A, it *still* only goes up by one,
because `input.pressed` fires on the press, not every frame (Chapter 9). This
is the machine-gun-glitch lesson turned into a thing you can hold.

```corelx
var count: int = 0

function Start()
    while true
        wait_vblank()
        input.poll()
        if input.pressed(A)
            count = count + 1
        text.draw(72, 64, 255, 255, 255, "PRESS A TO COUNT")
        text.draw(132, 96, 120, 255, 120, "COUNT")
        text.draw_int(150, 116, 255, 255, 0, count)
```

### `sprite.corelx` — a sprite you can move

The Sprite()/OAM workflow from Chapter 12, made concrete: an 8x8 box you drive
with the D-pad, clamped to the screen.

```corelx
const SPEED = 2
var box_x: int = 152
var box_y: int = 92

function Start()
    gfx.init_default_palettes()
    tile_base := gfx.load_tiles(ASSET_BoxTile, 8)

    box := Sprite()
    box.tile = tile_base
    box.attr = SPR_PAL(1)
    box.ctrl = SPR_ENABLE() | SPR_SIZE_8()

    ppu.enable_display()

    while true
        wait_vblank()
        input.poll()
        if input.held(RIGHT)
            box_x = box_x + SPEED
        -- (full clamp logic in docs/manual_examples/sprite.corelx)
        sprite.set_pos(box, box_x, box_y)
        oam.write(0, box)
        oam.flush()
```

### `structs.corelx` — your own struct

Chapter 7's `Player` struct, made concrete, including the reference-type
sharing behavior across a function call.

### `break_continue.corelx` — skip one, stop dead

Chapter 6's `break`/`continue` sum, made concrete: `23`, not `45` or `28`.

### `anim_module.corelx` — the `anim` module

Chapter 11's `anim.frame_index` example, made concrete and running.

### `floor.corelx` — walk the floor

The big one: a pseudo-3D floor that rushes toward you as you drive the D-pad.
It pulls together a tile asset, matrix-plane setup, the projection and camera
builtins, global state, `input.held` movement, signed clamping to keep you
inside the world, and a HUD.

```corelx
asset Floor: tiles8 hex
    11 11 22 22 11 11 22 22
    11 11 22 22 11 11 22 22
    22 22 11 11 22 22 11 11
    22 22 11 11 22 22 11 11
    11 11 22 22 11 11 22 22
    11 11 22 22 11 11 22 22
    22 22 11 11 22 22 11 11
    22 22 11 11 22 22 11 11

const MOVE = 6
var cam_x: int = 512
var cam_y: int = 768

function Start()
    gfx.init_default_palettes()
    bg.enable(0)
    bg.bind_transform(0, 0)
    bg.set_priority(0, 2)
    matrix.enable(0)
    matrix.identity(0)
    matrix.set_center(0, 160, 100)
    matrix_plane.enable(0, 128)
    matrix_plane.load_tiles(ASSET_Floor, 0, 0)
    matrix_plane.clear(0, 0, 0)
    matrix_plane.set_projection(0, 1, 113)
    matrix_plane.set_depth(0, 0x0C00, 0xC000, 0x00C0)
    ppu.enable_display()

    while true
        wait_vblank()
        input.poll()
        if input.held(UP)
            cam_y = cam_y - MOVE
        -- (full input/clamp logic in docs/manual_examples/floor.corelx)
        matrix_plane.set_camera(0, cam_x, cam_y, 0, 256)
        text.draw(8, 8, 255, 255, 255, "WALK THE FLOOR")
        text.draw(8, 184, 160, 200, 255, "DPAD TO MOVE")
```

> **Fletcher:** Read that last one top to bottom and notice there's nothing in
> it you haven't already met. Setup happens once, before the loop. The loop runs
> forever: wait for the frame, read the pad, move, *clamp so you can't walk off
> the edge of the world*, push the camera to the plane, draw the HUD. That
> shape — setup, then `while true` of poll/update/draw — is the skeleton under
> every game on this machine. Learn that rhythm and the rest is just filling in
> what happens in the middle.

## What's Next

This part grows as the real demo game grows. The current build (see
`Games/NitroPackInDemo/`) already goes well beyond these small teaching
demos — a full overworld with a walkable pseudo-3D floor, a building you can
enter, an interior scene, an NPC with dialogue, and a credits screen — and
each new system that lands there (enemies, combat, a health bar, an
inventory, background music and sound effects) gets its own chapter here once
it's built and verified, the same way every chapter above did.

---

# Part 3 — Tools and Reference

## Build and Run Workflows

### Workflow A: Nitro-Core-DX App (Recommended)

1. Open Nitro-Core-DX
2. Open a `.corelx` file
3. Click `Build + Run`
4. The ROM is compiled and loaded into the embedded emulator

Use `Load ROM` to test ROMs built by the CoreLX CLI, the assembler CLI, or Go
test ROM generators — useful for validating emulator behavior separately from
compiler behavior.

### Workflow B: CoreLX CLI

```bash
go run ./cmd/corelx mygame.corelx mygame.rom
go run ./cmd/emulator -rom mygame.rom
```

### Workflow C: Assembly CLI

```bash
go run ./cmd/asm mygame.asm mygame.rom
```

Then use Nitro-Core-DX `Load ROM` (recommended) or the standalone emulator CLI.

---

## Assembly (Advanced Users)

### Current Status (v1 Assembler)

- text `.asm` input -> `.rom` output ✅
- labels ✅
- branches/jumps/calls ✅
- full current CPU opcode coverage ✅
- simple directives (`.entry`, `.word`) ✅

### What Assembly Is Good For Right Now

- hardware tests
- timing experiments
- low-level demos
- FM MMIO experiments before higher-level CoreLX audio APIs cover them
- debugging compiler output assumptions

### Assembly Syntax (v1)

**Registers:** `R0` to `R7`

**Immediate values** — prefix with `#`:

```asm
MOV R0, #1
MOV R1, #0x8008
MOV R2, #$FF
```

**Memory access** — use `[Rn]`:

```asm
MOV R0, [R1]      ; 16-bit load (MMIO-safe for IO because CPU handles IO reads as 8-bit zero-extended)
MOV [R1], R0      ; 16-bit store (IO writes become low-byte writes)
MOV.B R2, [R1]    ; explicit 8-bit load
MOV.B [R1], R2    ; explicit 8-bit store
```

**Comments** — both styles accepted: `; comment` or `-- comment`.

**Labels:**

```asm
start:
    MOV R0, #0
loop:
    ADD R0, #1
    CMP R0, #10
    BLT loop
    RET
```

### Directives (v1)

- `.entry bank, offset` — set entry bank/offset (defaults: bank `1`, offset
  `0x8000`)
- `.word value` — emit one raw 16-bit word

### Supported Instructions (v1)

**Data/Memory:** `NOP` `MOV` `MOV.B` `PUSH` `POP`
**Arithmetic/Logic:** `ADD` `SUB` `MUL` `DIV` `AND` `OR` `XOR` `NOT` `SHL`
`SHR` `CMP`
**Control Flow:** `BEQ` `BNE` `BGT` `BLT` `BGE` `BLE` `JMP` `CALL` `RET`

> **Watch Out:** `DIV` is unsigned with no sign correction, and `SHR` is
> always logical, never arithmetic — the exact same two pitfalls as CoreLX's
> `/` and `>>` from Chapter 1, because CoreLX's `/`/`>>` compile straight down
> to these same instructions.

### Example: Tiny Assembly ROM

```asm
.entry 1, 0x8000

start:
    MOV R4, #0x8008      ; BG0_CONTROL
    MOV R5, #0x01        ; enable display
    MOV [R4], R5

main_loop:
    JMP main_loop
```

```bash
go run ./cmd/asm tiny.asm tiny.rom
```

> **Watch Out:** v1 assembler is same-bank / relative-control-flow oriented.
> Far-call/banked-assembler workflows are future work.

---

## CoreLX vs. Assembly: When to Use Which

**Use CoreLX when you want:** fast iteration, readable gameplay logic, easier
onboarding, `Build + Run`, fewer hardware details in your face.

**Use Assembly when you want:** exact instruction behavior, hardware
validation, custom low-level routines, MMIO experiments not wrapped by CoreLX
yet.

Since mixed-mode inline assembly isn't implemented yet, the practical combined
workflow is: write gameplay in CoreLX, write low-level experiments/tests in
assembly, and load both types of ROMs in Nitro-Core-DX (CoreLX via
`Build + Run`, assembly via `Load ROM`).

---

## Troubleshooting Guide

### "My ROM compiles but the screen is black"

Check these first:

- Did you call `bg.enable(layer)` for every layer you're using, and
  `ppu.enable_display()` once overall?
- Did you load tile data (`gfx.load_tiles`) before using the tile index?
- Did you set visible palette colors (`gfx.init_default_palettes()` or
  `gfx.set_palette_color(...)`)?
- If using sprites: did you write to OAM (`oam.write`) and call
  `oam.flush()`?
- If using a matrix plane: did you `matrix.enable(channel)` *and*
  `matrix_plane.enable(channel, size)` *and* `bg.bind_transform(layer,
  channel)`? All three are required — missing any one leaves the plane
  configured but invisible.

### "Movement is way too fast, or inconsistent frame to frame"

This is almost always the `wait_vblank()` multi-iteration pitfall from
Chapter 10 — your loop body is running more than once for some real video
frames. Add the `frame_counter()` debounce shown there.

### "Input does nothing in Nitro-Core-DX"

- Make sure **Capture Game Input** is enabled and click the emulator pane once.
- Confirm input works with a known-good ROM via `Load ROM` — this separates
  emulator issues from compiler issues.
- If a button seems to fire constantly, check whether you meant
  `input.pressed` and wrote `input.held` (Chapter 9's machine gun glitch).

### "Audio is silent"

- Make sure your system audio output is active.
- Confirm `music.play`/`music.play_loop` was actually called, and that the
  asset compiled without a diagnostic.
- Test with a known-good audio ROM first to separate a project-specific issue
  from an environment issue.

### "The building/billboard visually slips or drifts as I turn"

See Chapter 13's camera-eye pitfall — check that every `matrix_plane` in the
scene is fed the identical computed eye position each frame.

### "`else if` doesn't seem to work / a branch always runs"

See Chapter 6 — `else if` on one line silently misparses. Use `else` alone on
its own line with a fully nested `if`.

### "Assembly ROM runs strangely"

- Check label placement and branch targets.
- Prefer labels over manual branch offsets.
- Remember branches/jumps are PC-relative.
- Start with a tiny loop and add one feature at a time.

---

## What Is Planned

These are active directions, not promises of exact syntax:

- `else if`/`elseif` chaining fixed to actually chain (Chapter 6)
- `fixed / fixed` division
- Signed `/` and arithmetic `>>` (or an explicit signed-shift alternative)
- Richer asset model (tilemaps, palettes, gamedata, packaging integration)
- Better diagnostics and editor integration (squiggles, find/replace)
- Eventual mixed CoreLX + assembly support
- Sound Studio (in-app audio authoring)
- Debug overlays / memory viewers
- Continued FM accuracy and performance improvements for the YM2608 backend

---

## Reference Links

Use these for deeper details after you finish this manual:

- `docs/README.md` — docs map / source-of-truth guide
- `docs/CORELX.md` — language reference (partially stale, scheduled for a
  rewrite; the authoritative live builtin list is the registration block in
  `internal/corelx/semantic.go`)
- `docs/specifications/CORELX_CARTRIDGE_FORMAT.md` — CoreLX cartridge/asset
  format (v1 draft)
- `docs/DEVKIT_ARCHITECTURE.md` — backend/frontend split for Nitro-Core-DX
- `docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md` — current
  hardware spec reference
- `docs/specifications/APU_FM_OPM_EXTENSION_SPEC.md` — YM2608 audio subsystem
  runtime architecture/status
- `Games/NitroPackInDemo/` — the running demo game this manual's Part 2
  follows

---

## Final Advice

Start small. A great first Nitro-Core-DX project is:

1. draw one sprite
2. move it with input
3. change a color on button press
4. add one sound effect
5. then add a second object

That path teaches the whole system without overwhelming you.

And if you're an experienced programmer: treat Nitro-Core-DX like a hardware
platform, not a desktop app. Frame timing, memory layout, and simple pipelines
matter here in a good way.
