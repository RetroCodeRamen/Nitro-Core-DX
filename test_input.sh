#!/bin/bash
# Quick Input System Test Script

set -e

echo "=========================================="
echo "Input System Testing"
echo "=========================================="
echo ""

# Check if we're in the project root
if [ ! -f "go.mod" ]; then
    echo "❌ Error: Must run from project root directory"
    exit 1
fi

# Build test ROM generator
echo "📦 Building input test ROM generator..."
go build -o testrom_input ./cmd/testrom_input

# Generate test ROM
echo "🔨 Generating input test ROM..."
./testrom_input test/roms/input_test.rom

if [ ! -f "test/roms/input_test.rom" ]; then
    echo "❌ Failed to generate test ROM"
    exit 1
fi

echo "✅ Test ROM created: test/roms/input_test.rom"
echo ""

# Run unit tests
echo "🧪 Running unit tests..."
go test ./internal/input -v

if [ $? -ne 0 ]; then
    echo "❌ Unit tests failed"
    exit 1
fi

echo ""
echo "✅ All unit tests passed!"
echo ""

# Check if emulator is built
if [ ! -f "nitro-core-dx" ]; then
    echo "📦 Building emulator..."
    go build -tags "no_sdl_ttf" -o nitro-core-dx ./cmd/emulator
fi

echo ""
echo "=========================================="
echo "Manual Testing"
echo "=========================================="
echo ""
echo "To test input interactively, run:"
echo "  ./nitro-core-dx -rom test/roms/input_test.rom -scale 3"
echo ""
echo "Expected behavior:"
echo "  - White 8x8 sprite appears on screen"
echo "  - Press Arrow Keys or WASD to move sprite"
echo "  - Sprite should move smoothly in response"
echo ""
echo "Controls:"
echo "  UP/W    - Move sprite up"
echo "  DOWN/S  - Move sprite down"
echo "  LEFT/A  - Move sprite left"
echo "  RIGHT/D - Move sprite right"
echo ""
echo "Press any key to launch emulator, or Ctrl+C to exit..."
read -n 1 -s

echo ""
echo "🚀 Launching emulator..."
./nitro-core-dx -rom test/roms/input_test.rom -scale 3
