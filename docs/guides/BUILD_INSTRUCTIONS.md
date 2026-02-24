# Build Instructions

**Last Updated: January 6, 2026**

## Prerequisites

### Go Installation
- Go 1.22 or later is required
- Download from: https://golang.org/dl/

### SDL2 Development Libraries

The UI requires SDL2 to be installed on your system:

**Ubuntu/Debian:**
```bash
sudo apt-get install libsdl2-dev
```

**Fedora/RHEL:**
```bash
sudo dnf install SDL2-devel
```

**macOS (using Homebrew):**
```bash
brew install sdl2
```

**Windows:**
- Download SDL2 development libraries from: https://www.libsdl.org/download-2.0.php
- Extract and set environment variables as needed

## Building the Project

### 1. Build the Test ROM Generator
```bash
go build -o testrom ./cmd/testrom
```

### 2. Generate a Test ROM
```bash
./testrom test.rom
```

### 3. Build the Emulator

**Note:** The emulator binary is named `nitro-core-dx` (not `emulator`). This is the FPGA-ready, clock-driven emulator.

**Without SDL2_ttf (recommended if SDL2_ttf is not installed):**
```bash
go build -tags "no_sdl_ttf" -o nitro-core-dx ./cmd/emulator
```

**With SDL2_ttf (if you have SDL2_ttf installed):**
```bash
go build -o nitro-core-dx ./cmd/emulator
```

### 4. Run the Emulator
```bash
./nitro-core-dx -rom test.rom
```

## Command Line Options

- `-rom <path>`: Path to ROM file (required)
- `-unlimited`: Run at unlimited speed (no frame limit)
- `-scale <1-6>`: Display scale multiplier (default: 3)

## Example Usage

```bash
# Run with default 3x scale
./nitro-core-dx -rom test.rom

# Run at unlimited speed with 4x scale
./nitro-core-dx -rom test.rom -unlimited -scale 4

# Run with 1x scale (native resolution)
./nitro-core-dx -rom test.rom -scale 1
```

## Controls

- **Arrow Keys / WASD**: Move block
- **Z / W**: A button (change block color)
- **X**: B button (change background color)
- **Space**: Pause/Resume
- **Ctrl+R**: Reset emulator
- **Alt+F**: Toggle fullscreen
- **ESC**: Quit

## Debugging Tools

### Logging Output

The `run_with_log.sh` script can be used to capture all emulator output to a log file:

```bash
./run_with_log.sh -rom test/roms/demo.rom -scale 3
```

This will create a timestamped log file (e.g., `emulator_log_20260106_193000.txt`) containing all console output.

## Troubleshooting

### SDL2 Not Found
If you get an error about SDL2 not being found:
1. Install SDL2 development libraries (see Prerequisites)
2. Make sure `pkg-config` can find SDL2: `pkg-config --modversion sdl2`
3. If using a custom SDL2 installation, set `PKG_CONFIG_PATH` environment variable

### Build Errors
- Make sure Go is properly installed: `go version`
- Make sure all dependencies are downloaded: `go mod download`
- Clean and rebuild: `go clean -cache && go build ./...`

### Runtime Errors
- Check that the ROM file exists and is readable
- Verify the ROM file is a valid Nitro-Core-DX ROM (magic number "RMCF")
- Check console output for specific error messages

