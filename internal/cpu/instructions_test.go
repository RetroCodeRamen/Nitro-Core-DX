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
	cpu.SetRegister(1, 0x1000)            // Address: WRAM
	cpu.SetRegister(2, 0x5678)            // Value: 0x5678

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

func TestMOVMode7IOWriteLowByteOnly(t *testing.T) {
	mem := &testMemory{
		wram:     make([]uint8, 32768),
		ioWrites: make(map[uint16]uint8),
	}
	logger := &mockLogger{}

	cpu := NewCPU(mem, logger)
	cpu.State.DBR = 0
	cpu.SetRegister(1, 0x8000)
	cpu.SetRegister(2, 0x1234)

	if err := cpu.executeMOV(7, 1, 2); err != nil {
		t.Fatalf("executeMOV mode 7 failed: %v", err)
	}
	if got := mem.ioWrites[0x8000]; got != 0x34 {
		t.Fatalf("mode 7 I/O write: got 0x%02X, want 0x34", got)
	}
}

func TestMOVMode7HighROMAddressUsesDBRBank(t *testing.T) {
	mem := &testMemoryWithROM{
		wram: make([]uint8, 32768),
		rom:  make([]uint8, 65536),
	}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)
	cpu.State.DBR = 2

	cpu.SetRegister(1, 0x9000)
	cpu.SetRegister(2, 0x00CD)

	if err := cpu.executeMOV(7, 1, 2); err != nil {
		t.Fatalf("executeMOV mode 7 failed: %v", err)
	}
	romOffset := 32768 + 0x1000
	if got := mem.rom[romOffset]; got != 0xCD {
		t.Fatalf("mode 7 banked high-window write: got 0x%02X, want 0xCD", got)
	}
	if len(mem.ioSpace) != 0 {
		t.Fatalf("mode 7 banked high-window write should not touch I/O, got %d writes", len(mem.ioSpace))
	}
}

func TestMOVMode9IndexedLoad(t *testing.T) {
	mem := &testMemory{
		wram:     make([]uint8, 32768),
		ioWrites: make(map[uint16]uint8),
		ioReads:  make(map[uint16]uint8),
	}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)
	cpu.State.DBR = 0
	cpu.State.PCBank = 0

	// Positive displacement: base 0x2000 + 0x0004 = 0x2004
	cpu.State.PCOffset = 0x0100
	mem.wram[0x0100] = 0x04
	mem.wram[0x0101] = 0x00
	cpu.SetRegister(2, 0x2000)
	mem.wram[0x2004] = 0xEF
	mem.wram[0x2005] = 0xBE

	if err := cpu.executeMOV(9, 1, 2); err != nil {
		t.Fatalf("executeMOV mode 9 failed: %v", err)
	}
	if got := cpu.GetRegister(1); got != 0xBEEF {
		t.Fatalf("mode 9 positive disp: got 0x%04X, want 0xBEEF", got)
	}

	// Negative displacement: base 0x2008 + (-4) = 0x2004
	cpu.State.PCOffset = 0x0110
	mem.wram[0x0110] = 0xFC
	mem.wram[0x0111] = 0xFF
	cpu.SetRegister(2, 0x2008)
	if err := cpu.executeMOV(9, 1, 2); err != nil {
		t.Fatalf("executeMOV mode 9 failed (negative disp): %v", err)
	}
	if got := cpu.GetRegister(1); got != 0xBEEF {
		t.Fatalf("mode 9 negative disp: got 0x%04X, want 0xBEEF", got)
	}
}

func TestMOVMode9IndexedLoadIOZeroExtended(t *testing.T) {
	mem := &testMemory{
		wram:     make([]uint8, 32768),
		ioWrites: make(map[uint16]uint8),
		ioReads:  make(map[uint16]uint8),
	}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)
	cpu.State.DBR = 0
	cpu.State.PCBank = 0

	// base 0x7FFE + 0x0002 => 0x8000 (I/O)
	cpu.State.PCOffset = 0x0120
	mem.wram[0x0120] = 0x02
	mem.wram[0x0121] = 0x00
	cpu.SetRegister(2, 0x7FFE)
	mem.ioReads[0x8000] = 0x5A

	if err := cpu.executeMOV(9, 1, 2); err != nil {
		t.Fatalf("executeMOV mode 9 I/O failed: %v", err)
	}
	if got := cpu.GetRegister(1); got != 0x005A {
		t.Fatalf("mode 9 I/O zero-extend: got 0x%04X, want 0x005A", got)
	}
}

func TestMOVMode10IndexedStore(t *testing.T) {
	mem := &testMemory{
		wram:     make([]uint8, 32768),
		ioWrites: make(map[uint16]uint8),
		ioReads:  make(map[uint16]uint8),
	}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)
	cpu.State.DBR = 0
	cpu.State.PCBank = 0

	// base 0x1000 + 0x0002 => 0x1002
	cpu.State.PCOffset = 0x0130
	mem.wram[0x0130] = 0x02
	mem.wram[0x0131] = 0x00
	cpu.SetRegister(1, 0x1000)
	cpu.SetRegister(2, 0x5678)

	if err := cpu.executeMOV(10, 1, 2); err != nil {
		t.Fatalf("executeMOV mode 10 failed: %v", err)
	}
	got := uint16(mem.wram[0x1002]) | (uint16(mem.wram[0x1003]) << 8)
	if got != 0x5678 {
		t.Fatalf("mode 10 store: got 0x%04X, want 0x5678", got)
	}
}

func TestMOVMode10IndexedStoreIO8Bit(t *testing.T) {
	mem := &testMemory{
		wram:     make([]uint8, 32768),
		ioWrites: make(map[uint16]uint8),
		ioReads:  make(map[uint16]uint8),
	}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)
	cpu.State.DBR = 0
	cpu.State.PCBank = 0

	// base 0x7FFE + 0x0002 => 0x8000 (I/O)
	cpu.State.PCOffset = 0x0140
	mem.wram[0x0140] = 0x02
	mem.wram[0x0141] = 0x00
	cpu.SetRegister(1, 0x7FFE)
	cpu.SetRegister(2, 0xABCD)

	if err := cpu.executeMOV(10, 1, 2); err != nil {
		t.Fatalf("executeMOV mode 10 I/O failed: %v", err)
	}
	if got := mem.ioWrites[0x8000]; got != 0xCD {
		t.Fatalf("mode 10 I/O write low byte: got 0x%02X, want 0xCD", got)
	}
}

func TestMOVMode13IndexedLoadIO8BitZeroExtend(t *testing.T) {
	mem := &testMemory{
		wram:     make([]uint8, 32768),
		ioWrites: make(map[uint16]uint8),
		ioReads:  make(map[uint16]uint8),
	}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)
	cpu.State.DBR = 0
	cpu.State.PCBank = 0

	// base 0x7FFE + 0x0002 => 0x8000 (I/O)
	cpu.State.PCOffset = 0x0140
	mem.wram[0x0140] = 0x02
	mem.wram[0x0141] = 0x00
	cpu.SetRegister(2, 0x7FFE)
	mem.ioReads[0x8000] = 0x5A

	// Execute MOV R1, [R2+imm] (mode 13)
	if err := cpu.executeMOV(13, 1, 2); err != nil {
		t.Fatalf("executeMOV mode 13 I/O failed: %v", err)
	}
	if got := cpu.GetRegister(1); got != 0x005A {
		t.Fatalf("mode 13 I/O zero-extend: got 0x%04X, want 0x005A", got)
	}
}

func TestMOVMode14IndexedStoreIO8Bit(t *testing.T) {
	mem := &testMemory{
		wram:     make([]uint8, 32768),
		ioWrites: make(map[uint16]uint8),
		ioReads:  make(map[uint16]uint8),
	}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)
	cpu.State.DBR = 0
	cpu.State.PCBank = 0

	// base 0x7FFE + 0x0002 => 0x8000 (I/O)
	cpu.State.PCOffset = 0x0150
	mem.wram[0x0150] = 0x02
	mem.wram[0x0151] = 0x00
	cpu.SetRegister(1, 0x7FFE)
	cpu.SetRegister(2, 0xABCD)

	// Execute MOV [R1+imm], R2 (mode 14)
	if err := cpu.executeMOV(14, 1, 2); err != nil {
		t.Fatalf("executeMOV mode 14 I/O failed: %v", err)
	}
	if got := mem.ioWrites[0x8000]; got != 0xCD {
		t.Fatalf("mode 14 I/O write low byte: got 0x%02X, want 0xCD", got)
	}
}

func TestMOVMode2HighROMAddressUsesDBRBank(t *testing.T) {
	mem := &testMemoryWithROM{
		wram: make([]uint8, 32768),
		rom:  make([]uint8, 65536),
	}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)
	cpu.State.DBR = 2

	cpu.SetRegister(2, 0x9000)
	romOffset := 32768 + 0x1000
	mem.rom[romOffset] = 0xEF
	mem.rom[romOffset+1] = 0xBE

	if err := cpu.executeMOV(2, 1, 2); err != nil {
		t.Fatalf("executeMOV mode 2 failed: %v", err)
	}
	if got := cpu.GetRegister(1); got != 0xBEEF {
		t.Fatalf("mode 2 banked ROM read: got 0x%04X, want 0xBEEF", got)
	}
	if len(mem.ioSpace) != 0 {
		t.Fatalf("mode 2 banked ROM read should not touch I/O, got %d writes", len(mem.ioSpace))
	}
}

func TestMOVMode3HighROMAddressUsesDBRBank(t *testing.T) {
	mem := &testMemoryWithROM{
		wram: make([]uint8, 32768),
		rom:  make([]uint8, 65536),
	}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)
	cpu.State.DBR = 2

	cpu.SetRegister(1, 0x9000)
	cpu.SetRegister(2, 0xCAFE)

	if err := cpu.executeMOV(3, 1, 2); err != nil {
		t.Fatalf("executeMOV mode 3 failed: %v", err)
	}
	romOffset := 32768 + 0x1000
	if mem.rom[romOffset] != 0xFE || mem.rom[romOffset+1] != 0xCA {
		t.Fatalf("mode 3 banked high-window write: got [%02X %02X], want [FE CA]", mem.rom[romOffset], mem.rom[romOffset+1])
	}
	if len(mem.ioSpace) != 0 {
		t.Fatalf("mode 3 banked high-window write should not touch I/O, got %d writes", len(mem.ioSpace))
	}
}

func TestMOVMode13HighROMAddressUsesDBRBank(t *testing.T) {
	mem := &testMemoryWithROM{
		wram: make([]uint8, 32768),
		rom:  make([]uint8, 65536),
	}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)
	cpu.State.DBR = 2
	cpu.State.PCBank = 0

	cpu.State.PCOffset = 0x0160
	mem.wram[0x0160] = 0x02
	mem.wram[0x0161] = 0x00
	cpu.SetRegister(2, 0x8FFE)
	romOffset := 32768 + 0x1000
	mem.rom[romOffset] = 0x5A

	if err := cpu.executeMOV(13, 1, 2); err != nil {
		t.Fatalf("executeMOV mode 13 failed: %v", err)
	}
	if got := cpu.GetRegister(1); got != 0x005A {
		t.Fatalf("mode 13 banked ROM read: got 0x%04X, want 0x005A", got)
	}
	if len(mem.ioSpace) != 0 {
		t.Fatalf("mode 13 banked ROM read should not touch I/O, got %d writes", len(mem.ioSpace))
	}
}

func TestMOVMode14HighROMAddressUsesDBRBank(t *testing.T) {
	mem := &testMemoryWithROM{
		wram: make([]uint8, 32768),
		rom:  make([]uint8, 65536),
	}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)
	cpu.State.DBR = 2
	cpu.State.PCBank = 0

	cpu.State.PCOffset = 0x0170
	mem.wram[0x0170] = 0x02
	mem.wram[0x0171] = 0x00
	cpu.SetRegister(1, 0x8FFE)
	cpu.SetRegister(2, 0xABCD)

	if err := cpu.executeMOV(14, 1, 2); err != nil {
		t.Fatalf("executeMOV mode 14 failed: %v", err)
	}
	romOffset := 32768 + 0x1000
	if mem.rom[romOffset] != 0xCD {
		t.Fatalf("mode 14 banked high-window write: got 0x%02X, want 0xCD", mem.rom[romOffset])
	}
	if len(mem.ioSpace) != 0 {
		t.Fatalf("mode 14 banked high-window write should not touch I/O, got %d writes", len(mem.ioSpace))
	}
}

func TestMOVMode11LoadPostIncrement(t *testing.T) {
	mem := &testMemory{
		wram:     make([]uint8, 32768),
		ioWrites: make(map[uint16]uint8),
		ioReads:  make(map[uint16]uint8),
	}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)
	cpu.State.DBR = 0

	cpu.SetRegister(2, 0x3000)
	mem.wram[0x3000] = 0x34
	mem.wram[0x3001] = 0x12

	if err := cpu.executeMOV(11, 1, 2); err != nil {
		t.Fatalf("executeMOV mode 11 failed: %v", err)
	}
	if got := cpu.GetRegister(1); got != 0x1234 {
		t.Fatalf("mode 11 load: got 0x%04X, want 0x1234", got)
	}
	if got := cpu.GetRegister(2); got != 0x3002 {
		t.Fatalf("mode 11 post-inc: got 0x%04X, want 0x3002", got)
	}
}

func TestMOVMode12PreDecrementStore(t *testing.T) {
	mem := &testMemory{
		wram:     make([]uint8, 32768),
		ioWrites: make(map[uint16]uint8),
		ioReads:  make(map[uint16]uint8),
	}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)
	cpu.State.DBR = 0

	cpu.SetRegister(1, 0x3002)
	cpu.SetRegister(2, 0x1234)

	if err := cpu.executeMOV(12, 1, 2); err != nil {
		t.Fatalf("executeMOV mode 12 failed: %v", err)
	}
	if got := cpu.GetRegister(1); got != 0x3000 {
		t.Fatalf("mode 12 pre-dec pointer: got 0x%04X, want 0x3000", got)
	}
	val := uint16(mem.wram[0x3000]) | (uint16(mem.wram[0x3001]) << 8)
	if val != 0x1234 {
		t.Fatalf("mode 12 store: got 0x%04X, want 0x1234", val)
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
	cpu.State.SP = 0x1FFD           // Two bytes below top
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

func TestJMPMode1AbsoluteBankOffset(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	cpu.State.PCBank = 1
	cpu.State.PBR = 1
	cpu.State.PCOffset = 0x9000
	cpu.SetRegister(1, 5)      // target bank
	cpu.SetRegister(2, 0x8123) // target offset (will be aligned)

	if err := cpu.executeJMP(1, 1, 2); err != nil {
		t.Fatalf("executeJMP mode 1 failed: %v", err)
	}

	if got := cpu.State.PCBank; got != 5 {
		t.Fatalf("PCBank: got %d, want 5", got)
	}
	if got := cpu.State.PBR; got != 5 {
		t.Fatalf("PBR: got %d, want 5", got)
	}
	if got := cpu.State.PCOffset; got != 0x8122 {
		t.Fatalf("PCOffset: got 0x%04X, want 0x8122", got)
	}
}

func TestJMPMode1AbsoluteValidation(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	tests := []struct {
		name   string
		bank   uint16
		offset uint16
	}{
		{name: "bank zero invalid", bank: 0, offset: 0x8000},
		{name: "bank out of range invalid", bank: 126, offset: 0x8000},
		{name: "offset below rom window invalid", bank: 2, offset: 0x7FFE},
	}

	for _, tc := range tests {
		cpu.SetRegister(1, tc.bank)
		cpu.SetRegister(2, tc.offset)

		if err := cpu.executeJMP(1, 1, 2); err == nil {
			t.Fatalf("%s: expected error, got nil", tc.name)
		}
	}
}

func TestCALLMode1AbsoluteAndRETRoundTrip(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	cpu.State.PCBank = 2
	cpu.State.PBR = 2
	cpu.State.PCOffset = 0x9234
	cpu.State.Flags = 0x1A
	cpu.State.SP = 0x1FFF

	cpu.SetRegister(1, 6)      // target bank
	cpu.SetRegister(2, 0x8B11) // target offset (will be aligned)

	if err := cpu.executeCALL(1, 1, 2); err != nil {
		t.Fatalf("executeCALL mode 1 failed: %v", err)
	}

	if got := cpu.State.SP; got != 0x1FF9 {
		t.Fatalf("SP after CALL: got 0x%04X, want 0x1FF9", got)
	}
	if got := cpu.State.PCBank; got != 6 {
		t.Fatalf("PCBank after CALL: got %d, want 6", got)
	}
	if got := cpu.State.PBR; got != 6 {
		t.Fatalf("PBR after CALL: got %d, want 6", got)
	}
	if got := cpu.State.PCOffset; got != 0x8B10 {
		t.Fatalf("PCOffset after CALL: got 0x%04X, want 0x8B10", got)
	}

	if err := cpu.executeRET(); err != nil {
		t.Fatalf("executeRET failed: %v", err)
	}

	if got := cpu.State.SP; got != 0x1FFF {
		t.Fatalf("SP after RET: got 0x%04X, want 0x1FFF", got)
	}
	if got := cpu.State.PCBank; got != 2 {
		t.Fatalf("PCBank after RET: got %d, want 2", got)
	}
	if got := cpu.State.PBR; got != 2 {
		t.Fatalf("PBR after RET: got %d, want 2", got)
	}
	if got := cpu.State.PCOffset; got != 0x9234 {
		t.Fatalf("PCOffset after RET: got 0x%04X, want 0x9234", got)
	}
	if got := cpu.State.Flags; got != 0x1A {
		t.Fatalf("Flags after RET: got 0x%02X, want 0x1A", got)
	}
}

func TestExecuteInstructionDispatchesJMPMode1(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	cpu.SetEntryPoint(1, 0x8000)
	cpu.SetRegister(1, 4)
	cpu.SetRegister(2, 0x8A41)

	// 0xD112 => opcode 0xD (JMP), mode 1, reg1=1, reg2=2
	mem.Write8(1, 0x8000, 0x12)
	mem.Write8(1, 0x8001, 0xD1)

	if err := cpu.ExecuteInstruction(); err != nil {
		t.Fatalf("ExecuteInstruction failed: %v", err)
	}

	if got := cpu.State.PCBank; got != 4 {
		t.Fatalf("PCBank: got %d, want 4", got)
	}
	if got := cpu.State.PBR; got != 4 {
		t.Fatalf("PBR: got %d, want 4", got)
	}
	if got := cpu.State.PCOffset; got != 0x8A40 {
		t.Fatalf("PCOffset: got 0x%04X, want 0x8A40", got)
	}
}

func TestSHRMode2SARRegister(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	cpu.SetRegister(1, 0xF00F)
	cpu.SetRegister(2, 4)

	if err := cpu.executeSHR(2, 1, 2); err != nil {
		t.Fatalf("executeSHR mode 2 failed: %v", err)
	}

	if got := cpu.GetRegister(1); got != 0xFF00 {
		t.Fatalf("SAR result: got 0x%04X, want 0xFF00", got)
	}
	if !cpu.GetFlag(FlagC) {
		t.Fatalf("SAR carry flag: got false, want true")
	}
	if !cpu.GetFlag(FlagN) {
		t.Fatalf("SAR negative flag: got false, want true")
	}
	if cpu.GetFlag(FlagZ) {
		t.Fatalf("SAR zero flag: got true, want false")
	}
}

func TestSHRMode3SARImmediate(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	cpu.State.PCBank = 0
	cpu.State.PCOffset = 0x0200
	mem.Write8(0, 0x0200, 0x01) // immediate = 1
	mem.Write8(0, 0x0201, 0x00)
	cpu.SetRegister(1, 0x7001)

	if err := cpu.executeSHR(3, 1, 0); err != nil {
		t.Fatalf("executeSHR mode 3 failed: %v", err)
	}

	if got := cpu.GetRegister(1); got != 0x3800 {
		t.Fatalf("SAR immediate result: got 0x%04X, want 0x3800", got)
	}
	if !cpu.GetFlag(FlagC) {
		t.Fatalf("SAR immediate carry flag: got false, want true")
	}
	if cpu.GetFlag(FlagN) {
		t.Fatalf("SAR immediate negative flag: got true, want false")
	}
	if cpu.GetFlag(FlagZ) {
		t.Fatalf("SAR immediate zero flag: got true, want false")
	}
}

func TestSHRMode4ROLThroughCarry(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	cpu.SetRegister(1, 0x8001)
	cpu.SetRegister(2, 1)
	cpu.SetFlag(FlagC, true)

	if err := cpu.executeSHR(4, 1, 2); err != nil {
		t.Fatalf("executeSHR mode 4 failed: %v", err)
	}

	if got := cpu.GetRegister(1); got != 0x0003 {
		t.Fatalf("ROL result: got 0x%04X, want 0x0003", got)
	}
	if !cpu.GetFlag(FlagC) {
		t.Fatalf("ROL carry flag: got false, want true")
	}
	if cpu.GetFlag(FlagN) {
		t.Fatalf("ROL negative flag: got true, want false")
	}
	if cpu.GetFlag(FlagZ) {
		t.Fatalf("ROL zero flag: got true, want false")
	}
}

func TestSHRMode5RORThroughCarry(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	cpu.SetRegister(1, 0x0002)
	cpu.SetRegister(2, 1)
	cpu.SetFlag(FlagC, true)

	if err := cpu.executeSHR(5, 1, 2); err != nil {
		t.Fatalf("executeSHR mode 5 failed: %v", err)
	}

	if got := cpu.GetRegister(1); got != 0x8001 {
		t.Fatalf("ROR result: got 0x%04X, want 0x8001", got)
	}
	if cpu.GetFlag(FlagC) {
		t.Fatalf("ROR carry flag: got true, want false")
	}
	if !cpu.GetFlag(FlagN) {
		t.Fatalf("ROR negative flag: got false, want true")
	}
	if cpu.GetFlag(FlagZ) {
		t.Fatalf("ROR zero flag: got true, want false")
	}
}

func TestSHRUnknownModeError(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	if err := cpu.executeSHR(9, 1, 2); err == nil {
		t.Fatalf("expected unknown shift/rotate mode error, got nil")
	}
}

func TestExecuteInstructionDispatchesSARMode2(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	cpu.SetEntryPoint(1, 0x8000)
	cpu.SetRegister(1, 0xF00F)
	cpu.SetRegister(2, 4)

	// 0xB212 => opcode 0xB (SHR family), mode 2 (SAR), reg1=1, reg2=2
	mem.Write8(1, 0x8000, 0x12)
	mem.Write8(1, 0x8001, 0xB2)

	if err := cpu.ExecuteInstruction(); err != nil {
		t.Fatalf("ExecuteInstruction failed: %v", err)
	}

	if got := cpu.GetRegister(1); got != 0xFF00 {
		t.Fatalf("SAR dispatch result: got 0x%04X, want 0xFF00", got)
	}
	if !cpu.GetFlag(FlagC) {
		t.Fatalf("SAR dispatch carry flag: got false, want true")
	}
}

func TestADDMode2ByteRegister(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	cpu.SetRegister(1, 0x00F0)
	cpu.SetRegister(2, 0x0030)

	if err := cpu.executeADD(2, 1, 2); err != nil {
		t.Fatalf("executeADD mode 2 failed: %v", err)
	}

	if got := cpu.GetRegister(1); got != 0x0020 {
		t.Fatalf("ADD.B reg result: got 0x%04X, want 0x0020", got)
	}
	if !cpu.GetFlag(FlagC) {
		t.Fatalf("ADD.B reg carry: got false, want true")
	}
	if cpu.GetFlag(FlagV) {
		t.Fatalf("ADD.B reg overflow: got true, want false")
	}
	if cpu.GetFlag(FlagN) {
		t.Fatalf("ADD.B reg negative: got true, want false")
	}
	if cpu.GetFlag(FlagZ) {
		t.Fatalf("ADD.B reg zero: got true, want false")
	}
}

func TestADDMode3ByteImmediateOverflow(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	cpu.State.PCBank = 0
	cpu.State.PCOffset = 0x0300
	mem.Write8(0, 0x0300, 0x01) // low byte immediate
	mem.Write8(0, 0x0301, 0xAA) // high byte ignored for byte mode
	cpu.SetRegister(1, 0x007F)

	if err := cpu.executeADD(3, 1, 0); err != nil {
		t.Fatalf("executeADD mode 3 failed: %v", err)
	}

	if got := cpu.GetRegister(1); got != 0x0080 {
		t.Fatalf("ADD.B imm result: got 0x%04X, want 0x0080", got)
	}
	if cpu.GetFlag(FlagC) {
		t.Fatalf("ADD.B imm carry: got true, want false")
	}
	if !cpu.GetFlag(FlagV) {
		t.Fatalf("ADD.B imm overflow: got false, want true")
	}
	if !cpu.GetFlag(FlagN) {
		t.Fatalf("ADD.B imm negative: got false, want true")
	}
	if cpu.GetFlag(FlagZ) {
		t.Fatalf("ADD.B imm zero: got true, want false")
	}
}

func TestSUBMode2ByteRegister(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	cpu.SetRegister(1, 0x0010)
	cpu.SetRegister(2, 0x0020)

	if err := cpu.executeSUB(2, 1, 2); err != nil {
		t.Fatalf("executeSUB mode 2 failed: %v", err)
	}

	if got := cpu.GetRegister(1); got != 0x00F0 {
		t.Fatalf("SUB.B reg result: got 0x%04X, want 0x00F0", got)
	}
	if cpu.GetFlag(FlagC) {
		t.Fatalf("SUB.B reg carry: got true, want false (borrow expected)")
	}
	if cpu.GetFlag(FlagV) {
		t.Fatalf("SUB.B reg overflow: got true, want false")
	}
	if !cpu.GetFlag(FlagN) {
		t.Fatalf("SUB.B reg negative: got false, want true")
	}
	if cpu.GetFlag(FlagZ) {
		t.Fatalf("SUB.B reg zero: got true, want false")
	}
}

func TestSUBMode3ByteImmediateEqual(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	cpu.State.PCBank = 0
	cpu.State.PCOffset = 0x0310
	mem.Write8(0, 0x0310, 0xAA) // low byte immediate
	mem.Write8(0, 0x0311, 0x55) // high byte ignored for byte mode
	cpu.SetRegister(1, 0x00AA)

	if err := cpu.executeSUB(3, 1, 0); err != nil {
		t.Fatalf("executeSUB mode 3 failed: %v", err)
	}

	if got := cpu.GetRegister(1); got != 0x0000 {
		t.Fatalf("SUB.B imm result: got 0x%04X, want 0x0000", got)
	}
	if !cpu.GetFlag(FlagC) {
		t.Fatalf("SUB.B imm carry: got false, want true")
	}
	if cpu.GetFlag(FlagV) {
		t.Fatalf("SUB.B imm overflow: got true, want false")
	}
	if cpu.GetFlag(FlagN) {
		t.Fatalf("SUB.B imm negative: got true, want false")
	}
	if !cpu.GetFlag(FlagZ) {
		t.Fatalf("SUB.B imm zero: got false, want true")
	}
}

func TestADDAndSUBUnknownByteModeErrors(t *testing.T) {
	mem := &mockMemory{}
	logger := &mockLogger{}
	cpu := NewCPU(mem, logger)

	if err := cpu.executeADD(9, 1, 2); err == nil {
		t.Fatalf("expected unknown ADD mode error, got nil")
	}
	if err := cpu.executeSUB(9, 1, 2); err == nil {
		t.Fatalf("expected unknown SUB mode error, got nil")
	}
}

// Test memory for testing
type testMemory struct {
	wram     []uint8
	ioWrites map[uint16]uint8
	ioReads  map[uint16]uint8
}

func (m *testMemory) Read8(bank uint8, offset uint16) uint8 {
	if bank == 0 && offset < 0x8000 {
		return m.wram[offset]
	}
	if bank == 0 && offset >= 0x8000 {
		if m.ioReads == nil {
			return 0
		}
		return m.ioReads[offset]
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
