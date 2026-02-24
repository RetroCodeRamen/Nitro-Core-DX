package memory

import (
	"testing"

	"nitro-core-dx/internal/apu"
)

func TestBusFMExtensionMMIO(t *testing.T) {
	bus := NewBus(NewCartridge())
	audio := apu.NewAPU(44100, nil)
	bus.APUHandler = audio

	// CPU-visible FM host interface is 0x9100-0x91FF (APU offsets 0x0100-0x01FF).
	// Enable FM extension and set an OPM register via ADDR/DATA.
	bus.Write8(0, 0x9103, 0x01) // FM_CONTROL: enable
	bus.Write8(0, 0x9100, 0x22) // FM_ADDR
	bus.Write8(0, 0x9101, 0x99) // FM_DATA

	if got := bus.Read8(0, 0x9102); got&apu.FMStatusBusy == 0 {
		t.Fatalf("FM busy flag not set after FM write, status=0x%02X", got)
	}

	// Advance enough cycles for the phase-1 busy flag to clear.
	if err := audio.StepAPU(32); err != nil {
		t.Fatalf("StepAPU failed: %v", err)
	}
	if got := bus.Read8(0, 0x9102); got&apu.FMStatusBusy != 0 {
		t.Fatalf("FM busy flag did not clear, status=0x%02X", got)
	}

	// Read back the selected OPM register through the same MMIO path.
	bus.Write8(0, 0x9100, 0x22)
	if got := bus.Read8(0, 0x9101); got != 0x99 {
		t.Fatalf("FM register readback via bus: got 0x%02X, want 0x99", got)
	}
}

func TestBusFMExtensionTimerStatus(t *testing.T) {
	bus := NewBus(NewCartridge())
	audio := apu.NewAPU(44100, nil)
	bus.APUHandler = audio

	// Enable FM extension.
	bus.Write8(0, 0x9103, 0x01)

	// Program Timer A to shortest phase-1 period: raw=0x3FF => 64 cycles.
	bus.Write8(0, 0x9100, 0x10) // Timer A high
	bus.Write8(0, 0x9101, 0xFF)
	bus.Write8(0, 0x9100, 0x11) // Timer A low (2 bits)
	bus.Write8(0, 0x9101, 0x03)
	bus.Write8(0, 0x9100, 0x14) // Timer control: start A + IRQ enable A
	bus.Write8(0, 0x9101, 0x11)

	if err := audio.StepAPU(64); err != nil {
		t.Fatalf("StepAPU failed: %v", err)
	}
	status := bus.Read8(0, 0x9102)
	if status&apu.FMStatusTimerA == 0 {
		t.Fatalf("Timer A flag not set via bus path, status=0x%02X", status)
	}
	if status&apu.FMStatusIRQ == 0 {
		t.Fatalf("IRQ flag not set via bus path, status=0x%02X", status)
	}

	// Clear timer A flag via timer control bit2, keep start+IRQ enable.
	bus.Write8(0, 0x9100, 0x14)
	bus.Write8(0, 0x9101, 0x15)
	status = bus.Read8(0, 0x9102)
	if status&(apu.FMStatusTimerA|apu.FMStatusIRQ) != 0 {
		t.Fatalf("Timer A/IRQ flags not cleared via bus path, status=0x%02X", status)
	}
}
