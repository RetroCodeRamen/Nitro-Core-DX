package corelx

import (
	"testing"

	"nitro-core-dx/internal/apu"
)

// TestYMWritePort0 verifies ym.write drives the YM2608 host interface port 0:
// the register address reaches the address-select latch and the data reaches
// the data port, observed through the real emulator/bus/APU path.
func TestYMWritePort0(t *testing.T) {
	source := `function Start()
    ym.write(0x28, 0xF0)
    while true
        wait_vblank()
`
	emu, _ := compileAndBoot(t, source, 600)
	if got := emu.APU.FM.Addr; got != 0x28 {
		t.Errorf("port 0 address-select latch: want 0x28, got 0x%02X", got)
	}
	// Read back through the host data port (FMRegData returns the selected
	// register's value), matching the host-interface readback contract.
	if got := emu.APU.FM.Read8(apu.FMRegData); got != 0xF0 {
		t.Errorf("port 0 data (FMRegData readback at addr 0x28): want 0xF0, got 0x%02X", got)
	}
}

// TestYMWritePort1 verifies ym.write_port1 drives the YM2608 host interface
// upper port: address -> port-1 address register, value -> port-1 data
// register. These land in MixL/MixR in both the YMFM-backed and in-tree paths.
func TestYMWritePort1(t *testing.T) {
	source := `function Start()
    ym.write_port1(0x10, 0x34)
    while true
        wait_vblank()
`
	emu, _ := compileAndBoot(t, source, 600)
	if got := emu.APU.FM.MixL; got != 0x10 {
		t.Errorf("port 1 address register (MixL): want 0x10, got 0x%02X", got)
	}
	if got := emu.APU.FM.MixR; got != 0x34 {
		t.Errorf("port 1 data register (MixR): want 0x34, got 0x%02X", got)
	}
}
