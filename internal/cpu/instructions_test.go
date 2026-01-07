package cpu

import (
	"testing"
)

// TestMOVMode3IOWrite tests that MOV mode 3 writes 8-bit to I/O, 16-bit to normal memory
func TestMOVMode3IOWrite(t *testing.T) {
	mem := &testMemory{
		wram:     make([]uint8, 32768),
		ioWrites: make(map[uint16]uint8),
	}
	logger := &mockLogger{}
	
	cpu := NewCPU(mem, logger)
	cpu.State.DBR = 0
	
	// Test 1: Write to I/O address (bank 0, offset 0x8000+) - should write 8-bit only
	cpu.SetRegister(1, 0x8000) // Address: I/O register
	cpu.SetRegister(2, 0x1234) // Value: 0x1234
	
	// Execute MOV [R1], R2 (mode 3)
	if err := cpu.executeMOV(3, 1, 2); err != nil {
		t.Fatalf("executeMOV failed: %v", err)
	}
	
	// Verify only low byte (0x34) was written to I/O
	if mem.ioWrites[0x8000] != 0x34 {
		t.Errorf("I/O write: Expected 0x34 (low byte), got 0x%02X", mem.ioWrites[0x8000])
	}
	
	// Test 2: Write to normal memory (WRAM, offset < 0x8000) - should write 16-bit
	mem.ioWrites = make(map[uint16]uint8) // Clear I/O writes
	cpu.SetRegister(1, 0x1000) // Address: WRAM
	cpu.SetRegister(2, 0x5678) // Value: 0x5678
	
	// Execute MOV [R1], R2 (mode 3)
	if err := cpu.executeMOV(3, 1, 2); err != nil {
		t.Fatalf("executeMOV failed: %v", err)
	}
	
	// Verify 16-bit write to WRAM
	low := mem.wram[0x1000]
	high := mem.wram[0x1001]
	value := uint16(low) | (uint16(high) << 8)
	if value != 0x5678 {
		t.Errorf("WRAM write: Expected 0x5678, got 0x%04X (low=0x%02X, high=0x%02X)", value, low, high)
	}
	
	// Verify no I/O write occurred
	if len(mem.ioWrites) != 0 {
		t.Errorf("Expected no I/O writes for WRAM address, but got %d writes", len(mem.ioWrites))
	}
}

// TestDivisionByZero tests that division by zero sets the FlagD flag
func TestDivisionByZero(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	
	cpu := NewCPU(mem, logger)
	
	// Set up division: R1 = 100, R2 = 0 (division by zero)
	cpu.SetRegister(1, 100)
	cpu.SetRegister(2, 0)
	
	// Clear FlagD first
	cpu.SetFlag(FlagD, false)
	
	// Execute DIV R1, R2 (mode 0)
	if err := cpu.executeDIV(0, 1, 2); err != nil {
		t.Fatalf("executeDIV failed: %v", err)
	}
	
	// Verify FlagD is set
	if !cpu.GetFlag(FlagD) {
		t.Error("Division by zero: Expected FlagD to be set, but it's not")
	}
	
	// Verify result is 0xFFFF
	if cpu.GetRegister(1) != 0xFFFF {
		t.Errorf("Division by zero: Expected result 0xFFFF, got 0x%04X", cpu.GetRegister(1))
	}
	
	// Test normal division clears FlagD
	cpu.SetRegister(1, 100)
	cpu.SetRegister(2, 5)
	cpu.SetFlag(FlagD, true) // Set flag first
	
	// Execute DIV R1, R2 (mode 0)
	if err := cpu.executeDIV(0, 1, 2); err != nil {
		t.Fatalf("executeDIV failed: %v", err)
	}
	
	// Verify FlagD is cleared
	if cpu.GetFlag(FlagD) {
		t.Error("Normal division: Expected FlagD to be cleared, but it's still set")
	}
	
	// Verify result is correct
	if cpu.GetRegister(1) != 20 {
		t.Errorf("Normal division: Expected result 20, got %d", cpu.GetRegister(1))
	}
}

// TestStackUnderflow tests that Pop16 returns error on stack underflow
func TestStackUnderflow(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	
	cpu := NewCPU(mem, logger)
	
	// Set SP to top of stack (empty stack)
	cpu.State.SP = 0x1FFF
	
	// Try to pop from empty stack
	value, err := cpu.Pop16()
	if err == nil {
		t.Error("Expected error on stack underflow, but got none")
	}
	if value != 0 {
		t.Errorf("Expected 0 on stack underflow, got 0x%04X", value)
	}
	
	// Test with corrupted stack (SP too low)
	cpu.State.SP = 0x0050
	
	value, err = cpu.Pop16()
	if err == nil {
		t.Error("Expected error on corrupted stack, but got none")
	}
	
	// Test normal pop works - use testMemory that actually stores values
	testMem := &testMemory{
		wram:     make([]uint8, 32768),
		ioWrites: make(map[uint16]uint8),
	}
	cpu.Mem = testMem
	cpu.State.SP = 0x1FFD // Two bytes below top
	testMem.Write8(0, 0x1FFE, 0x12) // High byte
	testMem.Write8(0, 0x1FFF, 0x34) // Low byte
	
	value, err = cpu.Pop16()
	if err != nil {
		t.Errorf("Normal pop failed: %v", err)
	}
	if value != 0x1234 {
		t.Errorf("Expected 0x1234, got 0x%04X", value)
	}
}

// Test memory for testing
type testMemory struct {
	wram     []uint8
	ioWrites map[uint16]uint8
}

func (m *testMemory) Read8(bank uint8, offset uint16) uint8 {
	if bank == 0 && offset < 0x8000 {
		return m.wram[offset]
	}
	return 0
}

func (m *testMemory) Write8(bank uint8, offset uint16, value uint8) {
	if bank == 0 && offset < 0x8000 {
		m.wram[offset] = value
	} else if bank == 0 && offset >= 0x8000 {
		// Track I/O writes
		m.ioWrites[offset] = value
	}
}

func (m *testMemory) Read16(bank uint8, offset uint16) uint16 {
	low := m.Read8(bank, offset)
	high := m.Read8(bank, offset+1)
	return uint16(low) | (uint16(high) << 8)
}

func (m *testMemory) Write16(bank uint8, offset uint16, value uint16) {
	m.Write8(bank, offset, uint8(value&0xFF))
	m.Write8(bank, offset+1, uint8(value>>8))
}
