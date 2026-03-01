# Release Binaries (Linux + Windows)

This project now includes a release build workflow for the integrated **Nitro-Core-DX** app (editor + embedded emulator).

## What Gets Built

- **Linux amd64** archive: `nitrocoredx-<version>-linux-amd64.tar.gz`
- **Windows amd64** archive: `nitrocoredx-<version>-windows-amd64.zip`

These are single downloadable package files (archives).  
The Windows package includes `SDL2.dll`.

## GitHub Actions Workflow

Workflow file:

- `.github/workflows/release-binaries.yml`

### Triggers

- `workflow_dispatch` (manual run)
- `push` tags matching `v*` (example: `v0.1.0`)

### Release Behavior

- On a tag push, the workflow builds both platforms and attaches the archives to the GitHub Release for that tag.
- On manual runs, it uploads the archives as workflow artifacts.

## Local Linux Package (for quick testing)

You can generate a Linux release archive locally:

```bash
make release-linux
```

This creates:

- `dist/nitrocoredx-<version>-linux-amd64.tar.gz`

Default version is derived from `git describe --tags --always --dirty`.

## Notes / Limitations

- Builds use the `no_sdl_ttf` tag (SDL2_ttf is optional and not included).
- **Linux** still requires SDL2 runtime libraries installed on the target machine.
- **Windows** package includes `SDL2.dll`, but still depends on normal Windows graphics/runtime components.
- The packaged app is the integrated **Nitro-Core-DX** app; users can switch to **Emulator Focus** or **Code Only** view inside the app.
