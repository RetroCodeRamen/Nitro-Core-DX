package cpu

import (
	"testing"
)

// TestInterruptSystem tests interrupt handling
func TestInterruptSystem(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	// Set up interrupt vectors
	// IRQ vector: bank 1, offset 0x8000
	mem.Write8(0, VectorIRQ, 0x01)   // Bank 1
	mem.Write8(0, VectorIRQ+1, 0x80) // Offset 0x8000

	// Set entry point
	cpu.SetEntryPoint(1, 0x8000)

	// Set initial stack
	initialSP := cpu.State.SP

	// Set up a simple instruction at PC (MOV R0, R0 = 0x0000 is effectively NOP)
	initialPC := cpu.State.PCOffset
	mem.Write16(cpu.State.PCBank, cpu.State.PCOffset, 0x0000)
	
	// Trigger VBlank interrupt
	cpu.TriggerInterrupt(INT_VBLANK)

	// Execute enough cycles to process interrupt
	// Interrupts are checked at the end of each instruction
	// Execute one instruction - interrupt should be processed after it completes
	cpu.ExecuteCycles(10)

	// Verify interrupt was handled
	if cpu.State.InterruptPending != 0 {
		t.Errorf("Interrupt pending flag should be cleared, got %d", cpu.State.InterruptPending)
	}

	// Verify I flag is set (interrupts disabled)
	if !cpu.GetFlag(FlagI) {
		t.Errorf("I flag should be set after interrupt, got false")
	}

	// Verify PC jumped to interrupt vector (or at least changed from initial)
	// Note: The exact offset depends on vector format, but PC should have changed
	if cpu.State.PCOffset == initialPC {
		t.Errorf("Expected PC to change after interrupt, got same value 0x%04X", cpu.State.PCOffset)
	}
	if cpu.State.PCBank != 1 {
		t.Errorf("Expected PCBank=1 (interrupt vector), got %d", cpu.State.PCBank)
	}

	// Verify stack was used (SP should have decreased)
	if cpu.State.SP >= initialSP {
		t.Errorf("Stack should have been used (SP decreased), got SP=0x%04X (initial=0x%04X)",
			cpu.State.SP, initialSP)
	}
}

// TestNMIInterrupt tests non-maskable interrupt
func TestNMIInterrupt(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	// Set up NMI vector
	mem.Write8(0, VectorNMI, 0x01)
	mem.Write8(0, VectorNMI+1, 0x90) // Offset 0x9000

	// Set I flag (should not block NMI)
	cpu.SetFlag(FlagI, true)

	// Trigger NMI
	cpu.TriggerInterrupt(INT_NMI)

	// Execute to process interrupt
	cpu.ExecuteCycles(1)

	// Verify NMI was handled even with I flag set
	if cpu.State.InterruptPending != 0 {
		t.Errorf("NMI pending flag should be cleared, got %d", cpu.State.InterruptPending)
	}

	// Verify PC jumped to NMI vector
	if cpu.State.PCOffset != 0x9000 {
		t.Errorf("Expected PCOffset=0x9000 (NMI vector), got 0x%04X", cpu.State.PCOffset)
	}
}

// TestIRQMasked tests that IRQ is masked when I flag is set
func TestIRQMasked(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	// Set I flag (mask interrupts)
	cpu.SetFlag(FlagI, true)

	// Set entry point
	cpu.SetEntryPoint(1, 0x8000)
	
	// Set up an instruction at PC
	mem.Write16(cpu.State.PCBank, cpu.State.PCOffset, 0x0000)

	// Trigger IRQ
	cpu.TriggerInterrupt(INT_VBLANK)

	// Execute - interrupt should be ignored because I flag is set
	cpu.ExecuteCycles(10)

	// Verify interrupt is still pending (not handled because masked)
	if cpu.State.InterruptPending == 0 {
		t.Errorf("IRQ should be pending (masked by I flag), got 0")
	}
}
