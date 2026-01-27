# Building and Running the Bouncing Ball ROM

## Quick Start

From the **project root** (`/home/aj/Documents/Development/Nitro-Core-DX`):

### 1. Build the ROM

```bash
cd test/roms
go run build_bouncing_ball_sprite.go bouncing_ball.rom
```

### 2. Run the Emulator

From the **project root**:

```bash
./nitro-core-dx -rom test/roms/bouncing_ball.rom
```

Or if the binary is in `cmd/emulator`:

```bash
./cmd/emulator/nitro-core-dx -rom test/roms/bouncing_ball.rom
```

## Full Paths (if you're in a different directory)

### Build ROM:
```bash
cd /home/aj/Documents/Development/Nitro-Core-DX/test/roms
go run build_bouncing_ball_sprite.go bouncing_ball.rom
```

### Run Emulator:
```bash
cd /home/aj/Documents/Development/Nitro-Core-DX
./nitro-core-dx -rom test/roms/bouncing_ball.rom
```

## Troubleshooting

- **"No such file or directory"**: Make sure you're in the project root, not in `nxbrew-dl` or other subdirectories
- **"command not found"**: Make sure you've built the emulator: `go build -tags "no_sdl_ttf" ./cmd/emulator`
- **ROM not found**: Make sure you built the ROM first using the build script

## Project Structure

```
Nitro-Core-DX/
├── test/
│   └── roms/
│       ├── build_bouncing_ball_sprite.go  ← ROM builder script
│       └── bouncing_ball.rom                ← Built ROM (after running builder)
├── cmd/
│   └── emulator/
│       └── main.go                         ← Emulator main program
└── nitro-core-dx                           ← Built emulator binary (after building)
```
