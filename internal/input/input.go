package input

// InputSystem represents the input system
// It implements the memory.IOHandler interface
//
// FPGA Behavior: Uses serial shift register interface (SNES-style)
// - Writing 1 to latch register pulses LATCH signal to controller
// - Controller captures current button states into shift register
// - FPGA reads serial data and stores in latched register
// - Reading data register returns latched state (not current state)
// - Latch is edge-triggered: capture happens on write of 1, not persistent
type InputSystem struct {
	// Current button states (updated by UI/emulator)
	Controller1Buttons uint16
	Controller2Buttons uint16

	// Latched button states (captured when latch is written)
	// These are what the CPU reads after latching
	Controller1Latched uint16
	Controller2Latched uint16

	// Latch state (for tracking edge detection)
	Controller1LatchState bool
	Controller2LatchState bool
}

// NewInputSystem creates a new input system
func NewInputSystem() *InputSystem {
	return &InputSystem{
		Controller1Buttons:    0,
		Controller2Buttons:    0,
		Controller1Latched:    0,
		Controller2Latched:    0,
		Controller1LatchState: false,
		Controller2LatchState: false,
	}
}

// Read8 reads an 8-bit value from input registers
// Returns the LATCHED button state (not current state)
// This matches FPGA behavior: after latching, reads return the captured state
func (i *InputSystem) Read8(offset uint16) uint8 {
	switch offset {
	case 0x00: // CONTROLLER1 (low byte) - returns latched state
		value := uint8(i.Controller1Latched & 0xFF)
		// Debug: Log if we're returning non-zero when we shouldn't
		// (This will help identify if stale data is being read)
		return value
	case 0x01: // CONTROLLER1 (high byte) - returns latched state
		// Note: Writing to 0x01 is latch control, reading is data
		return uint8((i.Controller1Latched >> 8) & 0xFF)
	case 0x02: // CONTROLLER2 (low byte) - returns latched state
		return uint8(i.Controller2Latched & 0xFF)
	case 0x03: // CONTROLLER2 (high byte) - returns latched state
		// Note: Writing to 0x03 is latch control, reading is data
		return uint8((i.Controller2Latched >> 8) & 0xFF)
	default:
		return 0
	}
}

// Write8 writes an 8-bit value to input registers
// Latch control: Writing 1 to latch register captures current button state
// This matches FPGA behavior: pulse LATCH signal to controller, capture button states
func (i *InputSystem) Write8(offset uint16, value uint8) {
	switch offset {
	case 0x01: // CONTROLLER1_LATCH
		// Edge-triggered: capture on rising edge (0->1 transition)
		// In FPGA: this pulses the LATCH signal to the controller
		// The controller captures current button states into its shift register
		// The FPGA then reads the serial data and stores it
		if value == 1 {
			if !i.Controller1LatchState {
				// Rising edge: capture current button state into latched register
				// This simulates the FPGA reading serial data from controller
				i.Controller1Latched = i.Controller1Buttons
			}
			// If already latched (value == 1 and state == true), don't re-capture
			// This ensures edge-triggered behavior
		} else if value == 0 {
			// Writing 0 releases the latch (falling edge)
			// This allows the next write of 1 to be a rising edge again
		}
		// Update latch state
		i.Controller1LatchState = (value == 1)
	case 0x03: // CONTROLLER2_LATCH
		// Same behavior for controller 2
		if value == 1 {
			if !i.Controller2LatchState {
				// Rising edge: capture current button state into latched register
				i.Controller2Latched = i.Controller2Buttons
			}
		}
		i.Controller2LatchState = (value == 1)
	}
}

// Read16 reads a 16-bit value from input registers
func (i *InputSystem) Read16(offset uint16) uint16 {
	low := i.Read8(offset)
	high := i.Read8(offset + 1)
	return uint16(low) | (uint16(high) << 8)
}

// Write16 writes a 16-bit value to input registers
func (i *InputSystem) Write16(offset uint16, value uint16) {
	i.Write8(offset, uint8(value&0xFF))
	i.Write8(offset+1, uint8(value>>8))
}

// SetButton sets a button state for Controller 1
func (i *InputSystem) SetButton(button uint8, pressed bool) {
	if pressed {
		i.Controller1Buttons |= (1 << button)
	} else {
		i.Controller1Buttons &^= (1 << button)
	}
}

// SetButton2 sets a button state for Controller 2
func (i *InputSystem) SetButton2(button uint8, pressed bool) {
	if pressed {
		i.Controller2Buttons |= (1 << button)
	} else {
		i.Controller2Buttons &^= (1 << button)
	}
}

// Button constants
const (
	ButtonUP    = 0
	ButtonDOWN  = 1
	ButtonLEFT  = 2
	ButtonRIGHT = 3
	ButtonA     = 4
	ButtonB     = 5
	ButtonX     = 6
	ButtonY     = 7
	ButtonL     = 8
	ButtonR     = 9
	ButtonSTART = 10
	ButtonZ     = 11
)
