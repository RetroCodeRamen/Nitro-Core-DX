# The Nitro-Core-DX Manual

### The console, the hardware, and the Dev Kit

**A Retro Code Ramen publication.** Thank you for choosing Nitro-Core-DX.

> **This is the system book.** How to run the console, what is inside it, and
> how to use the Dev Kit to build and test a game. The language itself —
> CoreLX, its builtins, Sprite Lab / Tilemap Lab / Sound Studio *as they
> write into your source* — lives in the
> [CoreLX Programming Manual](CORELX_PROGRAMMING_MANUAL.md).

> **Status: living manual.** The numbers in this book are the numbers the
> emulator actually uses. If this page and the code disagree, trust the
> code. Historical design notes and FPGA drafts live under
> `docs/archive/`.

---

## Table of Contents

**Part 1 — Playing**
1. [Welcome](#1-welcome)
2. [Getting Started](#2-getting-started)
3. [The Controller](#3-the-controller)
4. [What Your System Can Do](#4-what-your-system-can-do)
5. [Care and Common Questions](#5-care-and-common-questions)

**Part 2 — The Hardware**
6. [System Specifications](#6-system-specifications)
7. [CPU](#7-cpu)
8. [Memory Map](#8-memory-map)
9. [PPU: Picture Processing](#9-ppu-picture-processing)
10. [APU: Sound](#10-apu-sound)
11. [Input Hardware](#11-input-hardware)
12. [I/O Register Map](#12-io-register-map)
13. [Timing, VBlank, and the Frame](#13-timing-vblank-and-the-frame)
14. [Cartridge Format](#14-cartridge-format)
15. [Reset and Power-On](#15-reset-and-power-on)

**Part 3 — The Dev Kit**
16. [Installing and Launching](#16-installing-and-launching)
17. [The App](#17-the-app)
18. [Building and Running](#18-building-and-running)
19. [The Editor](#19-the-editor)
20. [Sprite Lab](#20-sprite-lab)
21. [Tilemap Lab](#21-tilemap-lab)
22. [Sound Studio](#22-sound-studio)
23. [The Embedded Emulator](#23-the-embedded-emulator)
24. [Debugging in the Dev Kit](#24-debugging-in-the-dev-kit)
25. [Help Center](#25-help-center)
26. [Command-Line Tools](#26-command-line-tools)
27. [Standalone Emulator and Debugger](#27-standalone-emulator-and-debugger)
28. [Where to Go Next](#28-where-to-go-next)

---

# Part 1 — Playing

## 1. Welcome

Nitro-Core-DX is a 16-bit fantasy console — a machine from an alternate
timeline where the best ideas of the 16-bit era all landed in one box. It
pairs SNES-style graphics (layered tiles, Mode 7-style perspective, a
32,768-color palette) with Genesis-style speed (a fast CPU, DMA, a
steady 60 frames a second) and a Yamaha YM2608 for FM, SSG, rhythm, and
ADPCM sound.

You don't need to know any of that to play. You need to know how to
start it, hold the controller, and load a game. That's this part.

If you want to *make* games, skip to [Part 3](#16-installing-and-launching)
and then open the [CoreLX Programming Manual](CORELX_PROGRAMMING_MANUAL.md).

## 2. Getting Started

Open the program. A release package starts as `nitrocoredx` on Linux
or `nitrocoredx.exe` on Windows. From source, this opens the same
window:

```
go run ./cmd/corelx_devkit
```

That window is the Dev Kit: your source on one side, the game screen
on the other. **File → Open Project…** a `.corelx`, then **Build +
Run**. Or **File → Load ROM…** a cartridge you already have. You play
and build in that window. The terminal is how you launch it from
source, not how you drive it.

Games are **`.rom`** or **`.cart`** files. Same cartridge, 32-byte
header plus payload.

The picture below is the CoreLX overworld, the game this book keeps
coming back to: a floor, a building, a sprite, and a HUD, running on
the real screen.

![The overworld at the start of a game: a character facing a building across a green floor, with HP at the top right](docs/manual/images/overworld.png)

There is also a smaller player, for when you only want to run a
cartridge. Open it and use **File → Open ROM…**. A `.rom` and a
`.cart` both load. You do not have to put the path on the command
line.

```
go run ./cmd/emulator
```

Every program shows the boot logo first, then the game.

![The Nitro-Core-DX boot logo, held on a black screen before the game starts](docs/manual/images/boot.png)

## 3. The Controller

The Nitro-Core-DX controller is a classic twelve-button layout.

```
       _____________________________
      /                             \
     |  [L]                     [R]  |
     |   [↑]               (X)       |
     | [←][→]           (Y)   (A)    |
     |   [↓]               (B)       |
     |        [Z]  [START]           |
      \_____________________________/
```

| Control | What it does (in most games) | Keyboard |
|---------|------------------------------|----------|
| **D-Pad** (↑ ↓ ← →) | Move. Navigate menus. | Arrow keys, or WASD |
| **A** | Confirm / jump / primary action | **Z** |
| **B** | Cancel / secondary action | **X** |
| **X**, **Y** | Extra actions (game-specific) | **V**, **C** |
| **L**, **R** | Shoulder buttons (game-specific) | **Q**, **E** |
| **START** | Start the game / pause | **Enter** |
| **Z** | Menu / mode switch (game-specific) | **Backspace** |

Exactly what each button does is up to each game, but **START** to
begin and the **D-Pad** to move are nearly universal.

In the Dev Kit, check **Capture Input** on the emulator pane or the
keys will go to the editor instead. Click the emulator pane once after
you check it.

## 4. What Your System Can Do

- **Gorgeous color.** 320×200 pixels, drawn from a palette of **32,768
  colors** (RGB555) into a 256-color indexed display.
- **Layered worlds.** Four independent background layers (BG0–BG3)
  stack and scroll at their own speeds.
- **Matrix Mode.** Mode 7-style perspective that can tilt a floor or a
  wall into the distance, rotate it, and rush it toward you — on
  multiple layers at once.
- **128 sprites.** Hardware sprites from 8×8 through 128×128, with
  priority, flip, and blending.
- **Real speed.** A ~7.67 MHz CPU, 127,820 cycles a frame, a steady
  **60 FPS**.
- **Full sound.** YM2608 / OPNA: FM, SSG, rhythm, and ADPCM, at 44,100
  Hz.

The checkerboard is a matrix floor. The box is one hardware sprite,
8×8, with the text port drawing the line above it. The overworld in
[Getting Started](#2-getting-started) is those pieces in one scene.

![A perspective checkerboard floor with WALK THE FLOOR at the top](docs/manual/images/floor.png)

![An 8×8 sprite in the middle of the screen, under the line DPAD TO MOVE THE BOX](docs/manual/images/sprite.png)

## 5. Care and Common Questions

**Take breaks.** Sixty frames a second is easy on the eyes, but any
screen deserves a rest.

**"The game won't start."** Confirm the path is a real `.rom` / `.cart`
and is spelled correctly. If the window opens but the screen is black,
press START — many games wait on their title screen.

**"It's a black screen and nothing happens."** Confirm the file isn't
empty and that it is a Nitro-Core-DX cartridge (magic `RMCF` at byte
0). Try `roms/nitro_pack_in_demo.rom` to check the system itself.

**"The controls feel off."** Different games map the buttons
differently. D-Pad moves and START begins in almost everything. In the
Dev Kit, confirm **Capture Input** is checked.

---

# Part 2 — The Hardware

These figures describe the Nitro-Core-DX as implemented by the
emulator, which is the authoritative reference for the system's
behavior. FPGA drafts, pinouts, and historical specs are in
`docs/archive/`.

## 6. System Specifications

| Feature | Specification |
|---------|--------------|
| Display | 320×200 pixels (landscape); 200×320 portrait is reserved |
| Color | 256-color CGRAM, RGB555 (32,768 possible colors) |
| Tile size | 8×8 or 16×16, per background layer |
| Sprites | 128, sizes 8×8, 16×16, 32×16, 32×32, 64×32, 64×64, 128×64, 128×128 |
| Backgrounds | 4 independent layers (BG0–BG3) |
| Matrix Mode | 4 transform channels + 4 dedicated matrix planes |
| Audio | YM2608 / OPNA (FM, SSG, rhythm, ADPCM) at `0x9100–0x91FF` |
| Sample rate | 44,100 Hz (735 samples per frame) |
| CPU | ~7.67 MHz, 8× 16-bit registers, 24-bit banked addressing |
| Memory | 256 banks × 64 KB = 16 MB address space |
| Cartridge ROM | Banks 1–125, 32 KB each = **4,096,000 bytes** of payload |
| Frame | 60 FPS, 220 scanlines × 581 dots = 127,820 cycles |

```
┌─────────────────────────────────────────────────────────┐
│                    Nitro-Core-DX                        │
├─────────────────────────────────────────────────────────┤
│  CPU (~7.67 MHz)                                        │
│  ├─ R0–R7, 16-bit                                       │
│  ├─ 24-bit banked addressing                            │
│  └─ Custom 16-bit instruction set                       │
├─────────────────────────────────────────────────────────┤
│  Memory                                                 │
│  ├─ Bank 0: WRAM (32 KB) + I/O (32 KB)                  │
│  ├─ Banks 1–125: ROM (4,096,000 bytes)                  │
│  └─ Banks 126–127: Extended WRAM (128 KB)               │
├─────────────────────────────────────────────────────────┤
│  PPU                                                    │
│  ├─ VRAM 64 KB, CGRAM 512 B, OAM 768 B                  │
│  ├─ 4 background layers, 128 sprites                    │
│  ├─ 4 transform channels + 4 matrix planes              │
│  └─ Windowing, DMA, HDMA                                │
├─────────────────────────────────────────────────────────┤
│  APU — YM2608 / OPNA                                    │
│  ├─ Host interface 0x9100–0x91FF                        │
│  └─ Burst streamer 0x9110–0x9115                        │
├─────────────────────────────────────────────────────────┤
│  Input — two 12-button controllers                      │
└─────────────────────────────────────────────────────────┘
```

A 4-channel synth still answers at `0x9000–0x90FF`. It is temporary
migration scaffolding so old ROMs keep working. New work uses the
YM2608.

## 7. CPU

Eight general-purpose 16-bit registers, R0–R7. The program counter is
24-bit: an 8-bit bank plus a 16-bit offset. `PBR` is the program bank,
`DBR` the data bank. The stack pointer starts at `0x1FFF` in bank 0
and grows downward.

Flags: Z (zero), N (negative), C (carry), V (overflow), I (interrupt
disable), D (division by zero).

Instructions are 16-bit words. The families:

| Opcode | Instruction |
|--------|-------------|
| `0x0` | NOP |
| `0x1` | MOV (register, immediate, 16-bit and 8-bit load/store, PUSH, POP) |
| `0x2` | ADD |
| `0x3` | SUB |
| `0x4` | MUL |
| `0x5` | DIV (unsigned at the hardware; CoreLX `/` is signed — see the programming manual) |
| `0x6` | AND |
| `0x7` | OR |
| `0x8` | XOR |
| `0x9` | NOT |
| `0xA` | SHL |
| `0xB` | SHR (logical at the hardware; CoreLX `>>` is arithmetic) |
| `0xC` | CMP and the branches BEQ BNE BGT BLT BGE BLE |
| `0xD` | JMP (relative) |
| `0xE` | CALL |
| `0xF` | RET |

I/O at bank 0, offset `0x8000+` is 8-bit: a 16-bit load zero-extends,
a 16-bit store writes the low byte.

Interrupt vectors live at `0xFFE0–0xFFE3` in bank 0 (IRQ bank, IRQ
offset high, NMI bank, NMI offset high). VBlank raises IRQ. Saving
state costs 7 cycles. NMI is non-maskable; IRQ respects the I flag.

Hardware `DIV` by zero yields `0xFFFF` and sets D. Unknown MOV modes
8–15 are treated as NOP; unknown opcodes are an error.

The assembler (`go run ./cmd/asm in.asm out.rom`) is the direct way to
write this instruction set. CoreLX compiles to it. Mixing the two in
one file (`asm { ... }` inside CoreLX) is not implemented.

## 8. Memory Map

| Banks | What | Notes |
|-------|------|-------|
| 0 | 32 KB WRAM (`0x0000–0x7FFF`) + 32 KB I/O (`0x8000–0xFFFF`) | Stack from `0x1FFF` down. CoreLX locals live above `0x0100`; globals auto-place from `0x2100`. `0x2000–0x20FF` is compiler runtime scratch. `0x7000–0x7FFF` is yours — the compiler never touches it. |
| 1–125 | Cartridge ROM | LoROM: only `0x8000–0xFFFF` of each bank is wired. `romOffset = (bank-1)×32768 + (offset-0x8000)`. Writes are ignored. The low half of a ROM bank reads as 0. |
| 126–127 | Extended WRAM, 128 KB | `extOffset = (bank-126)×65536 + offset` |
| 128–255 | Reserved | Not implemented |

I/O decode: PPU `0x8000–0x8FFF`, APU `0x9000–0x9FFF`, Input
`0xA000–0xAFFF`.

Unmapped reads return 0. There is no open-bus simulation.

## 9. PPU: Picture Processing

**VRAM** 64 KB, addressed by `VRAM_ADDR_L/H` (`0x800E/0x800F`), data at
`VRAM_DATA` (`0x8010`), auto-increment.

**CGRAM** 512 bytes, 256 RGB555 colors in 16 palettes of 16. Address at
`0x8012`, data at `0x8013` (low byte then high byte). Color 0 of a
sprite palette is transparent.

**OAM** 768 bytes, 128 sprites × 6 bytes. Address (sprite id) at
`0x8014`, data at `0x8015`. Writes are blocked during visible
scanlines 0–199.

Each sprite:

| Byte | Meaning |
|------|---------|
| 0 | X low |
| 1 | Bit 0 = X sign (9-bit X, −256..255); bits [3:1] = size code 0–7 |
| 2 | Y (0–255) |
| 3 | Tile index |
| 4 | Attr: palette [3:0], flip X bit 4, flip Y bit 5, priority [7:6] |
| 5 | Control: enable bit 0, blend mode [3:2], alpha [7:4]. Bit 1 is a legacy 16×16 flag; the size code in byte 1 wins. |

Size codes: 0 = 8×8 (contiguous 32-byte tile), 1 = 16×16 (contiguous
128-byte tile), 2–7 = tile-grid of 8×8 tiles from the base index
(32×16, 32×32, 64×32, 64×64, 128×64, 128×128). A scanline may fetch
128 bytes of sprite pixel data; when that budget is exceeded, lower
priority sprites are dropped for that line only.

**Backgrounds.** Four layers. Per layer: 16-bit signed scroll, enable,
tile size (8 or 16), priority 0–3, tilemap size 32 / 64 / 128, a
tilemap base, a source-mode latch, and a bind to one of four transform
channels.

**Matrix Mode.** Four transform engines (A, B, C, D in 8.8, plus a
center) independently bindable to BG0–BG3, plus four dedicated matrix
planes that hold the actual tilemap / pattern / bitmap / row table.
Outside-screen modes: wrap, backdrop, tile 0, clamp. Direct color
bypasses CGRAM.

**Windows.** Two rectangular windows with OR / AND / XOR / XNOR logic,
main and sub enables.

**DMA.** One byte per cycle, source in ROM banks 1–125, destination
VRAM / CGRAM / OAM / matrix tilemap / pattern / bitmap / row table.
Mode 0 copy, mode 1 fill. Status bit at `0x8060` read.

**HDMA.** Per-scanline updates of scroll, matrix, transform bind,
priority, tilemap base, and source mode from a table in VRAM.

Compositing sorts backgrounds and sprites by priority (higher number
on top) and applies sprite blend modes.

## 10. APU: Sound

Final audio is the **YM2608 / OPNA**, reached through the host window
at `0x9100–0x91FF`:

| Address | Name |
|---------|------|
| `0x9100` | FM_ADDR (port 0) |
| `0x9101` | FM_DATA (port 0) |
| `0x9102` | FM_STATUS (timer A/B, busy, IRQ) |
| `0x9103` | FM_CONTROL (enable, mute, write-one reset) |
| `0x9104` | FM_PORT1_ADDR |
| `0x9105` | FM_PORT1_DATA |
| `0x9106` | FM_VOLUME (0–255, default 255) |

A burst streamer at `0x9110–0x9115` is what CoreLX `music.*` uses to
play a `.ncdxmusic` stream without a write per register from the CPU.
The stream format (magic `NCDXMUS1`) is a 16-byte header plus
frame-timed YM writes; `cmd/vgm_to_ncdxmusic` and Sound Studio both
produce it from VGM/VGZ.

The 4-channel synth at `0x9000–0x90FF` (sine / square / saw / noise,
master volume `0x9020`, one-shot completion `0x9021`) is the legacy
path. Do not write new software against it.

Sample rate is 44,100 Hz, 735 samples a frame, mixed to stereo by the
host.

## 11. Input Hardware

Two controllers. Per controller, 12 buttons. To read:

1. Write `1` to the latch (`0xA001` for controller 1, `0xA003` for
   controller 2). That captures the current buttons.
2. Read the data (`0xA000`/`0xA001` for 1, `0xA002`/`0xA003` for 2).
3. Write `0` to the latch to release.

A bit is 1 when the button is down.

| Byte | Bits |
|------|------|
| Low | UP 0, DOWN 1, LEFT 2, RIGHT 3, A 4, B 5, X 6, Y 7 |
| High | L 0, R 1, START 2, Z 3 |

CoreLX `input.poll` / `input.held` / `input.pressed` / `input.released`
wrap this. You do not latch by hand from CoreLX.

## 12. I/O Register Map

The tables below are the addresses the emulator actually decodes.
Writes to `0x803E–0x8040` are shared with BG2 matrix A/B low bytes on
the write path; reads are the VBlank / frame-counter registers.

**System**

| Address | Name |
|---------|------|
| `0x803E` | VBLANK_FLAG (bit 0; cleared on read; re-set if still in VBlank) |
| `0x803F` | FRAME_COUNTER_LOW |
| `0x8040` | FRAME_COUNTER_HIGH |

**Background scroll / control** — BG0 `0x8000–0x8003` scroll, `0x8008`
control; BG1 `0x8004–0x8007` scroll, `0x8009` control; BG2
`0x800A–0x800D` scroll, `0x8021` control; BG3 `0x8022–0x8025` scroll,
`0x8026` control. Source-mode latches `0x8068–0x806B`. Transform binds
`0x806C–0x806F`. Tilemap bases `0x8077–0x807E`.

**VRAM / CGRAM / OAM** — `0x800E–0x8010`, `0x8012–0x8015`.

**Matrix (legacy per-layer windows)** — BG0 `0x8018–0x802A`, BG1
`0x802B–0x8037`, BG2 `0x8038–0x8044`, BG3 `0x8045–0x8051`.

**Dedicated matrix planes** — select at `0x8080`, control `0x8081`,
then tilemap (`0x8082–0x8084`), pattern (`0x8085–0x8087`), bitmap
(`0x8088–0x808B`), flags / row / projection / camera / heading /
origin / facing through `0x80AA`.

**Windows** `0x8052–0x805C`. **HDMA** `0x805D–0x805F`, `0x807F`.
**DMA** `0x8060–0x8067`.

**Text port** (what CoreLX `text.draw` writes): X low `0x8070`, X high
`0x8071`, Y `0x8072`, R `0x8073`, G `0x8074`, B `0x8075`, character
`0x8076` (auto-advances X by 8).

**YM2608** `0x9100–0x9106` as in [§10](#10-apu-sound). **Input**
`0xA000–0xA003` as in [§11](#11-input-hardware).

## 13. Timing, VBlank, and the Frame

One frame is 220 scanlines of 581 dots: 200 visible (320 dots + 261
HBlank) then 20 VBlank. That is 127,820 cycles at ~7.67 MHz, 60 times
a second.

VBlank flag is set at the *end* of scanline 199, before scanline 200
starts, and an IRQ is raised. OAM writes are legal during VBlank.
HDMA runs at the start of each visible scanline. The 16-bit frame
counter increments once at the start of the frame.

CoreLX `wait_vblank()` is the high-level wait; `frame_counter()`
reads `0x803F/0x8040`. The programming manual's Chapter 10 covers
the software side, including the "your loop ran twice in one video
frame" pitfall.

## 14. Cartridge Format

32-byte header, then the payload.

| Offset | Size | Field |
|--------|------|-------|
| `0x00` | 4 | Magic `"RMCF"` |
| `0x04` | 2 | Format version (1) |
| `0x06` | 4 | ROM size in bytes |
| `0x0A` | 2 | Entry bank (1–125) |
| `0x0C` | 2 | Entry offset (`0x8000+`) |
| `0x0E` | 2 | Mapper flags (0 = LoROM; unused by the emulator) |
| `0x10` | 4 | Checksum (currently 0, not verified) |
| `0x14` | 12 | Reserved |

A single-bank file is 32,768 + 32 = 32,800 bytes. A full cartridge is
4,096,000 + 32 = 4,096,032 bytes. `.rom` and `.cart` are the same
bytes; the compiler CLI's usage line says `.cart`, the shipped demos
use both.

CoreLX's three-pass banking (compact, then measure, then pack) and
the locals budget are in the programming manual, Chapter 15.

## 15. Reset and Power-On

CPU: R0–R7 = 0, DBR = 0, SP = `0x1FFF`, flags = 0, cycles = 0. PC is
*not* cleared by Reset (so a reset after load does not lose the
entry); `SetEntryPoint` after ROM load sets it from the header.

WRAM, extended WRAM, VRAM, CGRAM, OAM start at zero. Backgrounds
disabled, matrix off, VBlank false, frame counter 0. YM2608 volume
255. Controllers released, latches open.

---

# Part 3 — The Dev Kit

The Dev Kit is the window you spend the day in: editor, Sprite Lab,
Tilemap Lab, Sound Studio, Build + Run, and an embedded emulator.
This part is the user guide for that window. What the labs *write
into your source* is in the programming manual, Chapter 15.

## 16. Installing and Launching

### A release package

[Releases](https://github.com/RetroCodeRamen/Nitro-Core-DX/releases)
ship as `nitrocoredx-<version>-linux-amd64.tar.gz` and
`nitrocoredx-<version>-windows-amd64.zip`. After extracting:

- Linux: `./nitrocoredx`
- Windows: `nitrocoredx.exe`

### From source

Go 1.22+ and SDL2.

```bash
# Ubuntu/Debian
sudo apt-get install libsdl2-dev
# Fedora/RHEL
sudo dnf install SDL2-devel
# macOS
brew install sdl2
```

```bash
git clone https://github.com/RetroCodeRamen/Nitro-Core-DX.git
cd Nitro-Core-DX
go run ./cmd/corelx_devkit
```

Optional: `go run ./cmd/corelx_devkit -file path/to/game.corelx`.
`-audio-backend ymfm` is accepted and is already the default.

Do not `go build ./...` as the project build; it walks vendored
C code under `Resources/` that this project does not need.

## 17. The App

Window title: `Nitro-Core-DX - <file> [<view>]`, with `*` when
unsaved.

### Menus

**File** — New Project, Open Project…, Load ROM…, Save, Save As…,
Recover Autosave, Open Recent.

**Edit** — Undo, Redo (shown, currently disabled), Find (Ctrl+F),
Find Next, Find Previous, Replace (Ctrl+H), Go to Line (Ctrl+G).

**View** — Code Only, Split View, Emulator Focus, Focus Active Panel.

**Build** — Build, Build + Run.

**Debug** — Run, Pause, Stop, Step Frame, Step CPU, Hardware Reset.

**Tools** — layout presets (Balanced, Code Focus, Art Mode, Debug
Mode, Emulator Focus); Theme: Nitro Industrial (default) or Nitro
Classic; UI Density: Compact (default) or Standard.

**Help** — Help Center, Open Docs on GitHub, About Nitro-Core-DX.

### Toolbar

Left to right, grouped: **New / Open / Save / Load ROM**, then
**Build / Build + Run**, then **Run / Pause / Stop**, then **Step
Frame / Step CPU / Mark Frame**, then the three view modes and
Focus Panel.

The identity row above that shows `NITRO CORE DX`, the current path,
and the build state: `Draft`, `Validating...`, `Validated`, or
`Error`. The bottom status line is the prose version (`Ready`,
`Building...`, `Build succeeded`).

### View modes

| Mode | What you see |
|------|----------------|
| Split View (default) | Emulator + bottom tabs on the left, workbench on the right |
| Code Only | Workbench over the bottom tabs, no emulator |
| Emulator Focus | Emulator only |

Workbench tabs: **Code**, **Sprite Lab**, **Tilemap**, **Sound**.
Bottom tabs: **Diagnostics**, **Output**, **Manifest**, **Debugger**.

### Opening files

**Open Project…** accepts `.corelx`, `.clx`, `.txt`, and (under "All
Supported") `.rom`. **Load ROM…** accepts `.rom` only. A `.ncdx`
project or a project folder is compiled with the CoreLX CLI
(`go run ./cmd/corelx MyGame.ncdx out.cart`); the Dev Kit edits the
`.corelx` inside.

### Settings and autosave

Settings live in `~/.config/nitro-core-dx/devkit_settings.json`:
view mode, layout, split offsets, Capture Input (default on),
recent files (15), density, theme, last directories. Autosave
writes `devkit_autosave.json` on every edit of a dirty buffer, and
offers **Recover Autosave** on the next launch if you didn't Save.

## 18. Building and Running

1. **File → New Project** and pick a template, or **Open Project…**
   a `.corelx`.
2. Edit.
3. **Build + Run**. The compiler runs, the ROM is loaded into the
   embedded emulator, and the pane starts.
4. Watch Diagnostics. Click a row to jump to the line.
5. Use Sprite Lab / Tilemap / Sound, then Build + Run again.

Templates (File → New Project):

| Name | What you get |
|------|----------------|
| Blank Game | Display on, a `wait_vblank` loop |
| Minimal Loop | A frame-cadence scaffold |
| Sprite Demo | One sprite and an OAM flush |
| Tilemap Demo | A tilemap and a camera |
| Shmup Starter | Vertical scroller, player, bullets |
| Matrix Mode Demo | A transform-channel starter |

**Build** compiles without running. **Load ROM** skips the compiler
and puts a prebuilt `.rom` in the emulator (useful for assembler
output or a CI artifact).

If `corelx.assets.json` sits next to the `.corelx`, its records are
merged with in-source `asset` declarations. Duplicate names are
`E_ASSET_DUPLICATE`. The Manifest tab shows the compile manifest
JSON; Output is a timestamped build log.

## 19. The Editor

The **Code** tab. Line numbers, CoreLX syntax highlighting, the
current line marked. Cut / Copy / Paste / Select All from the
context menu. Diagnostics do not squiggle in the buffer; they live
in the Diagnostics tab and jump here on click.

Find bar (Ctrl+F) sits above the source: Find, Replace, Line, Next /
Prev, Match case, Replace, All, Go, Close. Wrap-around is on.
Ctrl+H is Replace, Ctrl+G is Go to Line.

## 20. Sprite Lab

The **Sprite Lab** tab. Default canvas 24×24; sizes 8 through 64 in
steps of 8. Sixteen palette banks of sixteen RGB555 colors.

Tools: Pencil, Erase. Options: Mirror X, Show Grid, Index 0
Transparent (on by default). Undo / Redo (128 deep; Ctrl+Z / Ctrl+Y
while this tab is focused), Clear, Fill, Shift Up / Down / Left /
Right (wrap).

Palette: Bank 0–15, 16 chips, R/G/B, Hex, Apply RGB, Apply Hex, an
RGB555 slider.

Hotkeys on this tab: B pencil, E erase, X mirror, G grid, T
transparency, I/J/K/L shift wrap.

**Export .clxsprite** / **Import .clxsprite** (JSON `clxsprite-v1`,
`.json` also accepted) round-trip pixels and palettes.

Putting the drawing in the game — **Insert Source Snippet**, **Apply
Source Asset**, **Apply Manifest Asset**, **Copy Tile Hex** — is
specified in the programming manual, Chapter 15. The type rule: 8×8
→ `tiles8`, 16×16 → `tiles16`, anything else → `tileset`.

## 21. Tilemap Lab

The **Tilemap** tab. Maps 8×8 to 32×32, step of 8, default 32×32.
Pencil / Erase, Show Grid, Fill, Clear. Tile Index 0–255, Palette
0–15, Flip X, Flip Y. Undo depth 128.

**Refresh Tiles From Code** parses `tiles8` / `tileset` / `sprite`
hex assets already in the file into a clickable atlas.

**Export .clxtilemap** / **Import .clxtilemap** (`clxtilemap-v1`).
**Insert CoreLX Asset**, **Apply To Project**, **Apply To Manifest**
are in the programming manual, Chapter 15.

## 22. Sound Studio

The **Sound** tab.

- **Import VGM** — `.vgm` / `.vgz` → in-memory `.ncdxmusic`.
- **Open .ncdxmusic** — load and inspect one you already have.
- **Add to project** — the `.corelx` must already be saved. Writes
  `<Name>.ncdxmusic` next to it and upserts
  `asset Name: music "Name.ncdxmusic"`.
- **Preview** / **Pause preview** — builds a tiny ROM around the
  stream and loads it in the embedded emulator.
- INSERT snippets: Play once, Play loop, Play jingle, Stop, Set
  volume, Fade out — dropped at the cursor.

There is no separate export-file button and no manifest apply for
music; **Add to project** is the export. Chapter 14 of the
programming manual is the `music.*` reference.

## 23. The Embedded Emulator

Header: `DISPLAY`, a status line, and the **Capture Input**
checkbox. Idle overlay reads `STATE IDLE` / `PROGRAM NONE`. While
running: `running | FPS … | CPU … cyc/frame | Frame … | MM:SS.cs`.

When Capture Input is checked, keys go to the emulator even if the
editor is focused. Mapping is the table in [§3](#3-the-controller).

Audio is SDL, 44,100 Hz, stereo float, 735 samples a frame. If SDL
audio fails to init, the status line says `Ready (audio unavailable)`.

## 24. Debugging in the Dev Kit

**Run / Pause / Stop** on the toolbar and the Debug menu. **Step
Frame** and **Step CPU** require Pause first and currently step by
1 (the count field is not shown). **Hardware Reset** resets the
machine. **Mark Frame** appends a mark to Output.

The **Debugger** tab, after a successful Build + Run, shows
running/paused, PC (bank:offset, PBR, DBR, SP), flags, cycles,
frame, and R0–R7. There is no memory dump in this pane; use the
standalone debugger in [§27](#27-standalone-emulator-and-debugger)
for that.

Diagnostics: summary (`No build issues` or `Errors: N Warnings: N
Info: N`), filter All / Errors / Warnings / Info, a search box,
Collapse / Expand, list + detail. Click a row to jump.

## 25. Help Center

Help → Help Center. Left: the document catalog (this manual, the
CoreLX Programming Manual, README) and an outline of headings.
Right: the rendered page, Open on GitHub, Reload. There is no
full-text search; pick a document and use the outline.

The app finds the docs by looking for `README.md`,
`CORELX_PROGRAMMING_MANUAL.md`, and `docs/README.md` next to the
working directory or the executable.

## 26. Command-Line Tools

These are the tools an end user actually runs. Everything else under
`cmd/` is a test ROM generator, a capture harness, or an internal
trace.

```bash
# Dev Kit
go run ./cmd/corelx_devkit
go run ./cmd/corelx_devkit -file game.corelx

# Compile a project (.ncdx, a folder, or main.corelx) to a cartridge
go run ./cmd/corelx game.corelx game.cart

# Assemble
go run ./cmd/asm [--entry-bank N] [--entry-offset 0x8000] in.asm out.rom

# PNG → bitmap-plane asset (plane size 32, 64, or 128)
go run ./cmd/corelx_import park.png ParkFloor 128 1 park_floor.cxasset

# VGM/VGZ → .ncdxmusic
go run ./cmd/vgm_to_ncdxmusic -in theme.vgm -out theme.ncdxmusic

# Standalone player. Opens a window. File → Open ROM…,
# or pass a cartridge so it loads as the window opens.
go run ./cmd/emulator
go run ./cmd/emulator -rom game.cart
go run ./cmd/emulator -rom game.cart -scale 4 -unlimited

# Interactive debugger
go run ./cmd/debugger game.rom
```

Emulator flags: `-rom` (optional; File → Open ROM… if you leave it
off), `-unlimited`, `-scale 1–6`
(default 3), `-audio-backend ymfm`, `-log`, `-cyclelog file`,
`-maxcycles N` (default 100000, 0 = unlimited), `-cyclestart N`.

The memory map the compiler writes next to the ROM is
`<name>.rom.memmap` or `<name>.cart.memmap`.

## 27. Standalone Emulator and Debugger

The standalone emulator (`cmd/emulator`) opens as a window of its
own. **File → Open ROM…** takes a `.rom` or a `.cart`. Passing
`-rom your_game.cart` on the command line loads that cartridge as
the window opens; leaving it off just opens the player. Emulation →
Start / Pause / Resume / Stop / Restart / Step Frame. View and Debug
both expose Log Viewer, Registers, Memory Viewer, Tile Viewer.
Logging toggles per component (CPU, PPU, APU, Memory, Input, UI,
System) plus cycle logging.

The interactive debugger (`go run ./cmd/debugger game.rom`) starts
paused. Useful commands: `continue` / `c`, `step [n]` / `s`,
`pause` / `p`, `frame` / `f`, `break bank:offset`, `registers`,
`memory bank:offset [count]`, `stack`, `oam`, `ppu`, `watch expr`,
`callstack`, `help`, `quit`. Break example: `break 1:0x8000`.

Cycle logs from the emulator (`-cyclelog debug.log`) write cycle,
PC, R0–R7, SP, PBR, DBR, flags, and a PPU/APU snapshot per line.
They grow fast; cap them with `-maxcycles`.

## 28. Where to Go Next

- Play something in `roms/`.
- Open the Dev Kit, File → New Project → Sprite Demo, press Build +
  Run, move the box with the D-Pad.
- Then open the [CoreLX Programming Manual](CORELX_PROGRAMMING_MANUAL.md)
  and write the next one yourself.

Welcome to the machine.

— Retro Code Ramen
