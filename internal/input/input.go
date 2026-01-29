package input

// InputSystem represents the input system
// It implements the memory.IOHandler interface
type InputSystem struct {
	Controller1Buttons uint16
	Controller2Buttons uint16
	LatchActive        bool
	Controller2LatchActive bool
}

// NewInputSystem creates a new input system
func NewInputSystem() *InputSystem {
	return &InputSystem{
		Controller1Buttons: 0,
		Controller2Buttons: 0,
		LatchActive:        false,
		Controller2LatchActive: false,
	}
}

// Read8 reads an 8-bit value from input registers
func (i *InputSystem) Read8(offset uint16) uint8 {
	switch offset {
	case 0x00: // CONTROLLER1 (low byte)
		return uint8(i.Controller1Buttons & 0xFF)
	case 0x01: // CONTROLLER1 (high byte)
		return uint8((i.Controller1Buttons >> 8) & 0xFF)
	case 0x02: // CONTROLLER2 (low byte)
		return uint8(i.Controller2Buttons & 0xFF)
	case 0x03: // CONTROLLER2 (high byte)
		return uint8((i.Controller2Buttons >> 8) & 0xFF)
	default:
		return 0
	}
}

// Write8 writes an 8-bit value to input registers
func (i *InputSystem) Write8(offset uint16, value uint8) {
	switch offset {
	case 0x01: // CONTROLLER1_LATCH
		if value == 1 {
			i.LatchActive = true
		} else {
			i.LatchActive = false
		}
	case 0x03: // CONTROLLER2_LATCH
		if value == 1 {
			i.Controller2LatchActive = true
		} else {
			i.Controller2LatchActive = false
		}
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

