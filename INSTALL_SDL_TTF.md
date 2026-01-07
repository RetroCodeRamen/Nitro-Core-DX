# Installing SDL_ttf for System Fonts

To use system fonts instead of custom bitmap fonts, you need to install SDL_ttf:

## Linux (Ubuntu/Debian/Pop!_OS)
```bash
sudo apt-get update
sudo apt-get install libsdl2-ttf-dev
```

## macOS
```bash
brew install sdl2_ttf
```

## Windows
Download SDL_ttf development libraries from: https://www.libsdl.org/projects/SDL_ttf/

After installing, rebuild the project:
```bash
go build -o nitro-core-dx ./cmd/emulator
```

The code will automatically detect SDL_ttf and use system fonts instead of the bitmap font fallback.

