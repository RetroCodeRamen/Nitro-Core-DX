package memory

import (
	"encoding/binary"
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

	// Different FM backends may advance timer flags over slightly different
	// cycle counts, so poll instead of hard-coding 64 cycles.
	const (
		pollStepCycles = uint64(32)
		pollMaxCycles  = uint64(4096)
	)

	var status uint8
	timerARaised := false
	for waited := uint64(0); waited <= pollMaxCycles; waited += pollStepCycles {
		if err := audio.StepAPU(pollStepCycles); err != nil {
			t.Fatalf("StepAPU failed: %v", err)
		}
		status = bus.Read8(0, 0x9102)
		if status&apu.FMStatusTimerA != 0 {
			timerARaised = true
			break
		}
	}

	if !timerARaised {
		t.Fatalf("Timer A flag not set via bus path after polling, status=0x%02X", status)
	}
	if status&apu.FMStatusIRQ == 0 {
		t.Fatalf("IRQ flag not set via bus path after TimerA raised, status=0x%02X", status)
	}

	// Clear timer A flag via timer control bit2, keep start+IRQ enable.
	bus.Write8(0, 0x9100, 0x14)
	bus.Write8(0, 0x9101, 0x15)

	// Poll until TimerA/IRQ bits clear.
	for waited := uint64(0); waited <= pollMaxCycles; waited += pollStepCycles {
		status = bus.Read8(0, 0x9102)
		if status&(apu.FMStatusTimerA|apu.FMStatusIRQ) == 0 {
			return
		}
		if err := audio.StepAPU(pollStepCycles); err != nil {
			t.Fatalf("StepAPU failed during clear polling: %v", err)
		}
	}

	status = bus.Read8(0, 0x9102)
	if status&(apu.FMStatusTimerA|apu.FMStatusIRQ) != 0 {
		t.Fatalf("Timer A/IRQ flags not cleared via bus path after polling, status=0x%02X", status)
	}
}

func TestBusYMBurstStreamer(t *testing.T) {
	cart := NewCartridge()
	romBytes := make([]byte, 32+65536)
	copy(romBytes[0:4], []byte{'R', 'M', 'C', 'F'})
	binary.LittleEndian.PutUint16(romBytes[4:6], 1)
	binary.LittleEndian.PutUint32(romBytes[6:10], 65536)
	binary.LittleEndian.PutUint16(romBytes[10:12], 1)
	binary.LittleEndian.PutUint16(romBytes[12:14], 0x8000)
	// Bank 2, offset 0x8000 => ROM data offset 32768.
	romBytes[32+32768] = 0x00
	romBytes[32+32769] = 0x22
	romBytes[32+32770] = 0x99
	if err := cart.LoadROM(romBytes); err != nil {
		t.Fatalf("LoadROM failed: %v", err)
	}

	bus := NewBus(cart)
	audio := apu.NewAPU(44100, nil)
	bus.APUHandler = audio

	bus.Write8(0, 0x9103, 0x01) // FM_CONTROL enable
	bus.Write8(0, 0x9110, 0x01) // count low
	bus.Write8(0, 0x9111, 0x00) // count high
	bus.Write8(0, 0x9112, 0x02) // source bank
	bus.Write8(0, 0x9113, 0x00) // source offset low
	bus.Write8(0, 0x9114, 0x80) // source offset high
	bus.Write8(0, 0x9115, 0x01) // trigger

	bus.Write8(0, 0x9100, 0x22)
	if got := bus.Read8(0, 0x9101); got != 0x99 {
		t.Fatalf("YM burst register writeback mismatch: got 0x%02X want 0x99", got)
	}
}
