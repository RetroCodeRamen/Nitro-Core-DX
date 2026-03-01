# Input System Testing Guide

**Purpose**: Test the FPGA-compatible latch-based input system

## Quick Test

### 1. Build the Input Test ROM

```bash
# From project root
go build -o testrom_input ./cmd/testrom/input
./testrom_input test/roms/input_test.rom
```

### 2. Run the Emulator

```bash
# Make sure emulator is built
go build -tags "no_sdl_ttf" -o nitro-core-dx ./cmd/emulator

# Run the input test ROM
./nitro-core-dx -rom test/roms/input_test.rom -scale 3
```

### 3. Test Input Behavior

**Expected Behavior:**
- A white 8x8 sprite appears on screen
- Press **Arrow Keys** or **WASD** to move the sprite
- The sprite should move smoothly in response to input

**Controls:**
- **UP / W**: Move sprite up
- **DOWN / S**: Move sprite down
- **LEFT / A**: Move sprite left
- **RIGHT / D**: Move sprite right

## Detailed Testing

### Test 1: Basic Input Reading

The test ROM does the following sequence:
1. **Latch controller**: Writes 1 to `0xA001` (CONTROLLER1_LATCH)
2. **Read button state**: Reads from `0xA000` (CONTROLLER1 low byte)
3. **Release latch**: Writes 0 to `0xA001`

**What to verify:**
- ✅ Sprite moves when you press keys
- ✅ Sprite stops when you release keys
- ✅ Multiple keys can be pressed simultaneously (diagonal movement)

### Test 2: Latch Persistence

**Test the latch behavior:**
1. Press and hold a key (e.g., UP)
2. The sprite should move
3. Release the key
4. The sprite should stop

**What this verifies:**
- The latch captures button state when written
- Reading returns the latched state (not current state)
- Multiple reads return the same latched value until next latch

### Test 3: Edge-Triggered Latch

**Test edge detection:**
1. Press and hold UP
2. The sprite moves
3. While holding UP, the ROM latches again (every frame)
4. The sprite should continue moving

**What this verifies:**
- Latch captures on rising edge (0→1 transition)
- Writing 1 multiple times doesn't re-capture unless it was 0 first

## Automated Testing

### Create a Simple Test Script

Create `test_input.sh`:

```bash
#!/bin/bash

echo "=========================================="
echo "Testing Input System"
echo "=========================================="

# Build test ROM
echo "Building input test ROM..."
go build -o testrom_input ./cmd/testrom/input
./testrom_input test/roms/input_test.rom

if [ $? -ne 0 ]; then
    echo "❌ Failed to build test ROM"
    exit 1
fi

echo "✅ Test ROM built: test/roms/input_test.rom"
echo ""
echo "Manual Testing Required:"
echo "1. Run: ./nitro-core-dx -rom test/roms/input_test.rom"
echo "2. Press Arrow Keys / WASD to move sprite"
echo "3. Verify sprite responds to input"
echo ""
echo "Expected: White sprite moves smoothly with keyboard input"
```

Make it executable:
```bash
chmod +x test_input.sh
./test_input.sh
```

## Unit Test (Optional)

You can also create a Go unit test to verify the latch behavior programmatically:

Create `internal/input/input_test.go`:

```go
package input

import "testing"

func TestLatchBehavior(t *testing.T) {
    input := NewInputSystem()
    
    // Set a button
    input.SetButton(ButtonUP, true)
    
    // Initially, latched state should be 0
    if input.Controller1Latched != 0 {
        t.Errorf("Expected latched state to be 0 initially, got %d", input.Controller1Latched)
    }
    
    // Latch (write 1)
    input.Write8(0x01, 1)
    
    // Now latched state should capture current state
    if input.Controller1Latched != (1 << ButtonUP) {
        t.Errorf("Expected latched state to capture UP button, got %d", input.Controller1Latched)
    }
    
    // Read should return latched state
    lowByte := input.Read8(0x00)
    if lowByte != 0x01 {
        t.Errorf("Expected read to return latched state (0x01), got 0x%02X", lowByte)
    }
    
    // Change current state (but don't re-latch)
    input.SetButton(ButtonUP, false)
    input.SetButton(ButtonDOWN, true)
    
    // Read should still return old latched state
    lowByte = input.Read8(0x00)
    if lowByte != 0x01 {
        t.Errorf("Expected read to still return old latched state (0x01), got 0x%02X", lowByte)
    }
    
    // Re-latch to capture new state
    input.Write8(0x01, 0) // Release latch first
    input.Write8(0x01, 1) // Latch again
    
    // Now read should return new state
    lowByte = input.Read8(0x00)
    if lowByte != 0x02 { // DOWN button
        t.Errorf("Expected read to return new latched state (0x02), got 0x%02X", lowByte)
    }
}

func TestEdgeTriggeredLatch(t *testing.T) {
    input := NewInputSystem()
    
    // Set button
    input.SetButton(ButtonA, true)
    
    // First latch
    input.Write8(0x01, 1)
    if input.Controller1Latched != (1 << ButtonA) {
        t.Errorf("First latch should capture button state")
    }
    
    // Write 1 again (should not re-capture if already latched)
    oldLatched := input.Controller1Latched
    input.Write8(0x01, 1)
    if input.Controller1Latched != oldLatched {
        t.Errorf("Writing 1 again should not re-capture (edge-triggered)")
    }
    
    // Release latch
    input.Write8(0x01, 0)
    
    // Change button state
    input.SetButton(ButtonA, false)
    input.SetButton(ButtonB, true)
    
    // Latch again (rising edge)
    input.Write8(0x01, 1)
    if input.Controller1Latched != (1 << ButtonB) {
        t.Errorf("Second latch should capture new button state")
    }
}
```

Run the test:
```bash
go test ./internal/input -v
```

## Verification Checklist

- [ ] Test ROM builds successfully
- [ ] Emulator runs without errors
- [ ] Sprite appears on screen
- [ ] Arrow keys move sprite up/down/left/right
- [ ] WASD keys move sprite up/down/left/right
- [ ] Sprite stops when keys are released
- [ ] Multiple keys work simultaneously (diagonal movement)
- [ ] No input lag or jitter
- [ ] Unit tests pass (if created)

## Troubleshooting

### Sprite doesn't move
- **Check**: Are you pressing the correct keys? (Arrow Keys or WASD)
- **Check**: Is the emulator window focused? (click on it)
- **Check**: Are there any error messages in the console?

### Input feels laggy
- **Check**: Frame rate (should be ~60 FPS)
- **Check**: VBlank synchronization in ROM code
- **Note**: Some lag is expected if VBlank wait is not implemented

### Sprite moves but stops immediately
- **This might indicate**: Latch is not persisting correctly
- **Check**: ROM code should latch, read, then release latch
- **Verify**: Multiple reads return same value until re-latch

## Expected FPGA Behavior

When ported to FPGA, the input system should behave identically:
1. CPU writes 1 to latch register → FPGA pulses LATCH signal to controller
2. Controller captures button states into shift register
3. FPGA reads serial data from controller (16 clock pulses)
4. FPGA stores data in latched register
5. CPU reads data register → FPGA returns latched value

The emulator now matches this behavior, so ROMs written for the emulator will work on FPGA hardware.
