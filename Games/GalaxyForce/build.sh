#!/bin/bash
# Build Galaxy Force ROM
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "Building Galaxy Force..."
cd "$PROJECT_ROOT"
go run ./cmd/corelx "$SCRIPT_DIR/main.corelx" "$PROJECT_ROOT/roms/galaxy_force.rom"
echo "ROM built: $PROJECT_ROOT/roms/galaxy_force.rom"
echo ""
echo "Run with:"
echo "  go run -tags no_sdl_ttf ./cmd/emulator -rom roms/galaxy_force.rom"
