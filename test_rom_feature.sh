#!/bin/bash
# Test ROM Feature-by-Feature Build and Test Script

ROM_NAME=$1
if [ -z "$ROM_NAME" ]; then
    echo "Usage: $0 <rom_name>"
    echo "  rom_name: minimal, input, colors, audio, complete"
    exit 1
fi

echo "=========================================="
echo "Testing ROM: test_${ROM_NAME}.rom"
echo "=========================================="
echo ""

# Check if ROM exists
ROM_PATH="test/roms/test_${ROM_NAME}.rom"
if [ ! -f "$ROM_PATH" ]; then
    echo "❌ ROM not found: $ROM_PATH"
    echo "   Building it first..."
    
    case $ROM_NAME in
        minimal)
            ./testrom_minimal "$ROM_PATH"
            ;;
        *)
            echo "   Build script for $ROM_NAME not implemented yet"
            exit 1
            ;;
    esac
fi

echo "ROM found: $ROM_PATH"
echo "Size: $(ls -lh "$ROM_PATH" | awk '{print $5}')"
echo ""
echo "Running emulator..."
echo "Press ESC to quit after testing"
echo ""

# Run emulator
./nitro-core-dx -rom "$ROM_PATH"

echo ""
echo "=========================================="
echo "Test complete!"
echo "=========================================="
