package cpu

import (
	"testing"
)

// TestRETFromInterrupt tests RET when returning from interrupt
func TestRETFromInterrupt(t *testing.T) {
	mem := &testMemory{
		wram: make([]uint8, 32768),
	}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)
	cpu.SetEntryPoint(1, 0x8000)

	// Set up stack as interrupt handler would: PBR, PCOffset, Flags
	cpu.State.SP = 0x1FFF
	cpu.Push16(0x0001) // PBR
	cpu.Push16(0x8102) // PCOffset
	cpu.Push16(0x0005) // Flags

	spAfterPush := cpu.State.SP
	if spAfterPush != 0x1FF9 {
		t.Fatalf("After pushing: Expected SP=0x1FF9, got 0x%04X", spAfterPush)
	}

	// Verify stack contents manually
	t.Logf("Stack after push (SP=0x%04X):", spAfterPush)
	for i := uint16(0x1FF9); i <= 0x1FFF; i++ {
		t.Logf("  [0x%04X] = 0x%02X", i, mem.Read8(0, i))
	}

	// Test Pop16 directly
	spBeforePop := cpu.State.SP
	flags, err := cpu.Pop16()
	spAfterPop := cpu.State.SP
	t.Logf("Direct Pop16 test: SP before=0x%04X, after=0x%04X, value=0x%04X, err=%v", spBeforePop, spAfterPop, flags, err)
	if err != nil {
		t.Fatalf("Pop16 failed: %v", err)
	}
	if flags != 0x0005 {
		t.Errorf("Pop16: Expected 0x0005, got 0x%04X", flags)
	}
	if spAfterPop != spBeforePop+2 {
		t.Errorf("Pop16: SP should increase by 2 (from 0x%04X to 0x%04X), got 0x%04X", spBeforePop, spBeforePop+2, spAfterPop)
	}

	// Reset SP and re-push all values cleanly for RET test
	// Note: SP was modified by direct Pop16 test, so we need to reset everything
	cpu.State.SP = 0x1FFF
	cpu.Push16(0x0001) // PBR
	cpu.Push16(0x8102) // PCOffset
	cpu.Push16(0x0005) // Flags

	// Set PC to interrupt handler address
	cpu.State.PCBank = 1
	cpu.State.PCOffset = 0x9000
	cpu.State.PBR = 1

	// Write RET instruction at 0x9000
	mem.Write8(1, 0x9000, 0x00)
	mem.Write8(1, 0x9001, 0xF0) // RET: opcode=0xF

	// Verify stack before RET
	t.Logf("Stack before RET (SP=0x%04X):", cpu.State.SP)
	for i := uint16(0x1FF7); i <= 0x1FFF; i++ {
		t.Logf("  [0x%04X] = 0x%02X", i, mem.Read8(0, i))
	}

	// Execute RET via ExecuteInstruction
	spBeforeRET := cpu.State.SP
	pcBeforeRET := cpu.State.PCOffset
	t.Logf("Before ExecuteInstruction: SP=0x%04X, PC=0x%04X", spBeforeRET, pcBeforeRET)
	if err := cpu.ExecuteInstruction(); err != nil {
		t.Fatalf("RET failed: %v", err)
	}
	spAfterRET := cpu.State.SP
	pcAfterRET := cpu.State.PCOffset

	t.Logf("After RET: SP=0x%04X, PC=0x%04X, PBR=%d, Flags=0x%02X", spAfterRET, pcAfterRET, cpu.State.PBR, cpu.State.Flags)
	
	// Verify stack after RET
	t.Logf("Stack after RET (SP=0x%04X):", spAfterRET)
	for i := uint16(0x1FF7); i <= 0x1FFF; i++ {
		t.Logf("  [0x%04X] = 0x%02X", i, mem.Read8(0, i))
	}

	// Check results
	if spAfterRET != 0x1FFF {
		t.Errorf("RET: Expected SP=0x1FFF (stack empty), got 0x%04X", spAfterRET)
	}
	if pcAfterRET != 0x8102 {
		t.Errorf("RET: Expected PCOffset=0x8102, got 0x%04X", pcAfterRET)
	}
	if cpu.State.PBR != 1 {
		t.Errorf("RET: Expected PBR=1, got %d", cpu.State.PBR)
	}
	if cpu.State.Flags != 0x05 {
		t.Errorf("RET: Expected Flags=0x05, got 0x%02X", cpu.State.Flags)
	}
}
