# NitroPackInDemo

This folder is the canonical home for the ROM-first pack-in demo project.

The full design and build plan lives in:

- [DESIGN.md](/home/aj/Documents/Development/Nitro-Core-DX/Games/NitroPackInDemo/DESIGN.md)

Current status:

- ROM-first implementation path is active
- `build_rom.go` is the current playable vertical-slice builder
- `build_rom_test.go` covers scene flow, movement, bounds, and plane camera-model assumptions
- CoreLX rebuild is still planned after the ROM version is complete and validated

## Build

From the repository root:

```bash
go run -tags testrom_tools ./Games/NitroPackInDemo -out roms/nitro_pack_in_demo.rom
```

Then run it with:

```bash
./emulator -rom roms/nitro_pack_in_demo.rom
```

Default placeholder assets:

- Floor: `Games/NitroPackInDemo/park.png`
- Billboard/building: `Games/NitroPackInDemo/building.png`

Optional overrides:

```bash
go run -tags testrom_tools ./Games/NitroPackInDemo \
  -floor Games/NitroPackInDemo/park.png \
  -billboard Games/NitroPackInDemo/building.png \
  -out roms/nitro_pack_in_demo.rom
```

## Current Scope

The current ROM covers the active playable vertical slice:

- Title scene with `PRESS START`
- Overworld pseudo-3D slice using `park.png` as the floor
- Main building facade from `building.png`
- Interior placeholder scene with return-to-overworld stub
- Explicit scene state in WRAM
- Input polling and start-edge handling
- Matrix floor + vertical projected quad facade overworld
- Open park-floor walk bounds with world clamps at map edges
- Generic matrix-floor movement: `Up/Down` move, `Left/Right` turn
- Centered placeholder player sprite in the overworld
- Pause overlay on `START`
- Door trigger and scene transition on `A`
- Automated scene-flow and camera-model tests in `build_rom_test.go`

## Active Tuning Focus

The main open rendering issue is the overworld building facade.

Ground anchoring is substantially improved, but the next pass still needs to
make the facade feel less like a camera-facing sprite and more like a properly
turning/foreshortening vertical surface when viewed from an angle.

That work is currently being validated in both:

- `internal/ppu/scanline.go`
- `internal/ppu/features_test.go`

before more overworld props or interior-room features are layered on top.
