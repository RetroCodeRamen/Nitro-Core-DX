package cpu

import (
	"testing"
)

// TestPop16ModifiesSP tests that Pop16 actually modifies SP
func TestPop16ModifiesSP(t *testing.T) {
	mem := &testMemory{
		wram: make([]uint8, 32768),
	}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	// Push some values
	cpu.State.SP = 0x1FFF
	cpu.Push16(0x1234)
	spAfterPush := cpu.State.SP
	if spAfterPush != 0x1FFD {
		t.Errorf("After Push16: Expected SP=0x1FFD, got 0x%04X", spAfterPush)
	}

	// Pop value
	spBeforePop := cpu.State.SP
	value, err := cpu.Pop16()
	if err != nil {
		t.Fatalf("Pop16 failed: %v", err)
	}
	spAfterPop := cpu.State.SP

	if value != 0x1234 {
		t.Errorf("Pop16: Expected value 0x1234, got 0x%04X", value)
	}
	if spAfterPop != spBeforePop+2 {
		t.Errorf("Pop16: Expected SP to increase by 2 (from 0x%04X to 0x%04X), but got 0x%04X", spBeforePop, spBeforePop+2, spAfterPop)
	}
	if spAfterPop != 0x1FFF {
		t.Errorf("Pop16: Expected SP=0x1FFF (back to original), got 0x%04X", spAfterPop)
	}
}

// TestRETPopOrder tests RET pop order for interrupt returns
func TestRETPopOrder(t *testing.T) {
	mem := &testMemory{
		wram: make([]uint8, 32768),
	}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	// Set up stack as interrupt handler would: PBR, PCOffset, Flags
	cpu.State.SP = 0x1FFF
	cpu.Push16(0x0001) // PBR
	cpu.Push16(0x8102) // PCOffset
	cpu.Push16(0x0005) // Flags

	spAfterPush := cpu.State.SP
	if spAfterPush != 0x1FF9 {
		t.Errorf("After pushing 3 values: Expected SP=0x1FF9, got 0x%04X", spAfterPush)
	}

	// Now simulate RET popping: Flags, PCOffset, PBR
	// Pop Flags
	flags, err := cpu.Pop16()
	if err != nil {
		t.Fatalf("Pop Flags failed: %v", err)
	}
	if flags != 0x0005 {
		t.Errorf("Pop Flags: Expected 0x0005, got 0x%04X", flags)
	}
	spAfterFlags := cpu.State.SP
	if spAfterFlags != 0x1FFB {
		t.Errorf("After popping Flags: Expected SP=0x1FFB, got 0x%04X", spAfterFlags)
	}

	// Pop PCOffset
	pcOffset, err := cpu.Pop16()
	if err != nil {
		t.Fatalf("Pop PCOffset failed: %v", err)
	}
	if pcOffset != 0x8102 {
		t.Errorf("Pop PCOffset: Expected 0x8102, got 0x%04X", pcOffset)
	}
	spAfterPC := cpu.State.SP
	if spAfterPC != 0x1FFD {
		t.Errorf("After popping PCOffset: Expected SP=0x1FFD, got 0x%04X", spAfterPC)
	}

	// Pop PBR
	pbr, err := cpu.Pop16()
	if err != nil {
		t.Fatalf("Pop PBR failed: %v", err)
	}
	if pbr != 0x0001 {
		t.Errorf("Pop PBR: Expected 0x0001, got 0x%04X", pbr)
	}
	spAfterPBR := cpu.State.SP
	if spAfterPBR != 0x1FFF {
		t.Errorf("After popping PBR: Expected SP=0x1FFF, got 0x%04X", spAfterPBR)
	}
}
