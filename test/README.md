# Test ROMs

This directory contains test ROMs for the Nitro-Core-DX emulator.

## ROM Files

- `demo.rom`: Demo ROM with movable box, color cycling, and audio
- `audiotest.rom`: Audio test ROM with arpeggios and chords
- `test.rom`: Basic test ROM

## Generating ROMs

To generate ROMs, build the ROM generators and run them:

```bash
# Build ROM generators
go build -o demorom ./cmd/demorom
go build -o audiotest ./cmd/audiotest
go build -o testrom ./cmd/testrom

# Generate ROMs
./demorom demo.rom
./audiotest audiotest.rom
./testrom test.rom
```

## Running Test ROMs

```bash
# Build emulator
go build -o nitro-core-dx ./cmd/emulator

# Run a test ROM
./nitro-core-dx -rom test/roms/demo.rom -scale 3
```

