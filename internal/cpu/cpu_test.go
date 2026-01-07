package cpu

import (
	"testing"
)

// TestResetPreservesPC tests that Reset() doesn't corrupt PCBank/PCOffset
func TestResetPreservesPC(t *testing.T) {
	// Create a mock memory interface
	mem := &mockMemory{}
	logger := &mockLogger{}
	
	cpu := NewCPU(mem, logger)
	
	// Set entry point (simulating ROM load)
	cpu.SetEntryPoint(1, 0x8000)
	
	// Verify entry point is set
	if cpu.State.PCBank != 1 {
		t.Errorf("Expected PCBank=1, got %d", cpu.State.PCBank)
	}
	if cpu.State.PCOffset != 0x8000 {
		t.Errorf("Expected PCOffset=0x8000, got 0x%04X", cpu.State.PCOffset)
	}
	
	// Call Reset() - should NOT corrupt PCBank/PCOffset
	cpu.Reset()
	
	// Verify PCBank/PCOffset are still set (not reset to 0)
	if cpu.State.PCBank != 1 {
		t.Errorf("After Reset(): Expected PCBank=1, got %d (should NOT be reset)", cpu.State.PCBank)
	}
	if cpu.State.PCOffset != 0x8000 {
		t.Errorf("After Reset(): Expected PCOffset=0x8000, got 0x%04X (should NOT be reset)", cpu.State.PCOffset)
	}
	
	// Verify other registers ARE reset
	if cpu.State.R0 != 0 {
		t.Errorf("After Reset(): Expected R0=0, got %d", cpu.State.R0)
	}
	if cpu.State.SP != 0x1FFF {
		t.Errorf("After Reset(): Expected SP=0x1FFF, got 0x%04X", cpu.State.SP)
	}
}

// Mock memory for testing
type mockMemory struct{}

func (m *mockMemory) Read8(bank uint8, offset uint16) uint8 {
	return 0
}

func (m *mockMemory) Write8(bank uint8, offset uint16, value uint8) {
}

func (m *mockMemory) Read16(bank uint8, offset uint16) uint16 {
	return 0
}

func (m *mockMemory) Write16(bank uint8, offset uint16, value uint16) {
}

// Mock logger for testing
type mockLogger struct{}

func (m *mockLogger) LogCPU(instruction uint16, state CPUState, cycles uint32) {
}

