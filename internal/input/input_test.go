package input

import "testing"

// TestLatchBehavior tests that the latch correctly captures button states
func TestLatchBehavior(t *testing.T) {
	input := NewInputSystem()

	// Set a button
	input.SetButton(ButtonUP, true)

	// Initially, latched state should be 0
	if input.Controller1Latched != 0 {
		t.Errorf("Expected latched state to be 0 initially, got %d", input.Controller1Latched)
	}

	// Latch (write 1) - should capture current state
	input.Write8(0x01, 1)

	// Now latched state should capture current state
	expected := uint16(1 << ButtonUP)
	if input.Controller1Latched != expected {
		t.Errorf("Expected latched state to capture UP button (0x%04X), got 0x%04X", expected, input.Controller1Latched)
	}

	// Read should return latched state
	lowByte := input.Read8(0x00)
	if lowByte != 0x01 {
		t.Errorf("Expected read to return latched state (0x01), got 0x%02X", lowByte)
	}

	// Change current state (but don't re-latch)
	input.SetButton(ButtonUP, false)
	input.SetButton(ButtonDOWN, true)

	// Read should still return old latched state (UP, not DOWN)
	lowByte = input.Read8(0x00)
	if lowByte != 0x01 {
		t.Errorf("Expected read to still return old latched state (0x01 = UP), got 0x%02X", lowByte)
	}

	// Re-latch to capture new state (release first, then latch again)
	input.Write8(0x01, 0) // Release latch
	input.Write8(0x01, 1) // Latch again (rising edge)

	// Now read should return new state (DOWN)
	lowByte = input.Read8(0x00)
	if lowByte != 0x02 { // DOWN button
		t.Errorf("Expected read to return new latched state (0x02 = DOWN), got 0x%02X", lowByte)
	}
}

// TestEdgeTriggeredLatch tests that latch is edge-triggered (only captures on 0->1 transition)
func TestEdgeTriggeredLatch(t *testing.T) {
	input := NewInputSystem()

	// Set button
	input.SetButton(ButtonA, true)

	// First latch (rising edge: 0->1)
	input.Write8(0x01, 1)
	expected := uint16(1 << ButtonA)
	if input.Controller1Latched != expected {
		t.Errorf("First latch should capture button state (0x%04X), got 0x%04X", expected, input.Controller1Latched)
	}

	// Write 1 again (should not re-capture if already latched)
	oldLatched := input.Controller1Latched
	input.Write8(0x01, 1)
	if input.Controller1Latched != oldLatched {
		t.Errorf("Writing 1 again should not re-capture (edge-triggered). Expected 0x%04X, got 0x%04X", oldLatched, input.Controller1Latched)
	}

	// Release latch
	input.Write8(0x01, 0)

	// Change button state
	input.SetButton(ButtonA, false)
	input.SetButton(ButtonB, true)

	// Latch again (rising edge: 0->1)
	input.Write8(0x01, 1)
	expected = uint16(1 << ButtonB)
	if input.Controller1Latched != expected {
		t.Errorf("Second latch should capture new button state (0x%04X), got 0x%04X", expected, input.Controller1Latched)
	}
}

// TestMultipleButtons tests that multiple buttons can be latched simultaneously
func TestMultipleButtons(t *testing.T) {
	input := NewInputSystem()

	// Set multiple buttons
	input.SetButton(ButtonUP, true)
	input.SetButton(ButtonA, true)
	input.SetButton(ButtonSTART, true) // High byte

	// Latch
	input.Write8(0x01, 1)

	// Check low byte
	lowByte := input.Read8(0x00)
	expectedLow := uint8((1 << ButtonUP) | (1 << ButtonA))
	if lowByte != expectedLow {
		t.Errorf("Expected low byte 0x%02X (UP + A), got 0x%02X", expectedLow, lowByte)
	}

	// Check high byte
	highByte := input.Read8(0x01)
	expectedHigh := uint8(1 << (ButtonSTART - 8)) // START is bit 2 in high byte
	if highByte != expectedHigh {
		t.Errorf("Expected high byte 0x%02X (START), got 0x%02X", expectedHigh, highByte)
	}
}

// TestController2 tests that controller 2 has independent latch
func TestController2(t *testing.T) {
	input := NewInputSystem()

	// Set controller 1 button
	input.SetButton(ButtonUP, true)

	// Set controller 2 button
	input.SetButton2(ButtonDOWN, true)

	// Latch controller 1
	input.Write8(0x01, 1) // Controller 1 latch

	// Latch controller 2
	input.Write8(0x03, 1) // Controller 2 latch

	// Read controller 1 (should have UP)
	ctrl1Low := input.Read8(0x00)
	if ctrl1Low != 0x01 {
		t.Errorf("Controller 1 should have UP (0x01), got 0x%02X", ctrl1Low)
	}

	// Read controller 2 (should have DOWN)
	ctrl2Low := input.Read8(0x02)
	if ctrl2Low != 0x02 {
		t.Errorf("Controller 2 should have DOWN (0x02), got 0x%02X", ctrl2Low)
	}
}

// TestRead16 tests that Read16 returns correct 16-bit value
func TestRead16(t *testing.T) {
	input := NewInputSystem()

	// Set buttons in both low and high bytes
	input.SetButton(ButtonUP, true)      // Bit 0 (low byte)
	input.SetButton(ButtonSTART, true)   // Bit 10 (high byte, bit 2)

	// Latch
	input.Write8(0x01, 1)

	// Read 16-bit value
	value := input.Read16(0x00)
	expected := uint16((1 << ButtonUP) | (1 << ButtonSTART))
	if value != expected {
		t.Errorf("Expected 16-bit value 0x%04X, got 0x%04X", expected, value)
	}
}
