#!/bin/bash
# Build Galaxy Force ROM
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "Building Galaxy Force..."
cd "$PROJECT_ROOT"
go run ./cmd/corelx "$SCRIPT_DIR/main.corelx" "$SCRIPT_DIR/galaxy_force.rom"
echo "ROM built: $SCRIPT_DIR/galaxy_force.rom"
echo ""
echo "Run with:"
echo "  go run -tags no_sdl_ttf ./cmd/emulator -rom Games/GalaxyForce/galaxy_force.rom"
