package rom

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestBankedROMBuilderBuildROMBytesPaddedBanks(t *testing.T) {
	b := NewBankedROMBuilder()
	b.AddInstruction(1, 0x1111)
	b.AddInstruction(3, 0x3333)

	data, err := b.BuildROMBytes(1, 0x8000)
	if err != nil {
		t.Fatalf("BuildROMBytes failed: %v", err)
	}

	if got := binary.LittleEndian.Uint32(data[0:4]); got != 0x46434D52 {
		t.Fatalf("magic = 0x%08X, want RMCF", got)
	}
	romSize := binary.LittleEndian.Uint32(data[6:10])
	wantSize := uint32(3 * ROMBankSizeBytes) // banks 1..3 inclusive, bank 2 padded
	if romSize != wantSize {
		t.Fatalf("romSize = %d, want %d", romSize, wantSize)
	}
	if len(data) != int(32+wantSize) {
		t.Fatalf("total file size = %d, want %d", len(data), 32+wantSize)
	}

	// Bank 1 first word
	if got := binary.LittleEndian.Uint16(data[32:34]); got != 0x1111 {
		t.Fatalf("bank1 word0 = 0x%04X, want 0x1111", got)
	}
	// Bank 2 is padded zeros
	bank2Base := 32 + ROMBankSizeBytes
	if got := binary.LittleEndian.Uint16(data[bank2Base : bank2Base+2]); got != 0x0000 {
		t.Fatalf("bank2 word0 = 0x%04X, want 0x0000", got)
	}
	// Bank 3 first word
	bank3Base := 32 + 2*ROMBankSizeBytes
	if got := binary.LittleEndian.Uint16(data[bank3Base : bank3Base+2]); got != 0x3333 {
		t.Fatalf("bank3 word0 = 0x%04X, want 0x3333", got)
	}
}

func TestBankedROMBuilderResolveRelativeRelocationSameBank(t *testing.T) {
	b := NewBankedROMBuilder()
	const bank = 1

	b.AddInstruction(bank, EncodeJMP())
	currentPC := b.PC(bank) // points at offset word address
	b.AddImmediate(bank, 0x0000)
	wordIndex := b.GetCodeLength(bank) - 1
	b.AddRelative16Relocation(bank, wordIndex, currentPC, bank, "target")

	b.AddInstruction(bank, EncodeNOP()) // one word between jmp and target
	b.MarkLabel(bank, "target")
	targetPC := b.PC(bank)
	b.AddInstruction(bank, EncodeNOP())

	if err := b.ResolveRelocations(); err != nil {
		t.Fatalf("ResolveRelocations failed: %v", err)
	}

	got := b.bank(bank).code[wordIndex]
	want := uint16(CalculateBranchOffset(currentPC, targetPC))
	if got != want {
		t.Fatalf("patched relative offset = 0x%04X, want 0x%04X", got, want)
	}
}

func TestBankedROMBuilderRejectsCrossBankRelativeRelocation(t *testing.T) {
	b := NewBankedROMBuilder()

	b.AddInstruction(1, EncodeJMP())
	currentPC := b.PC(1)
	b.AddImmediate(1, 0x0000)
	wordIndex := b.GetCodeLength(1) - 1
	b.AddRelative16Relocation(1, wordIndex, currentPC, 2, "target")

	b.MarkLabel(2, "target")
	b.AddInstruction(2, EncodeNOP())

	err := b.ResolveRelocations()
	if err == nil {
		t.Fatalf("expected cross-bank relative relocation error, got nil")
	}
	if !strings.Contains(err.Error(), "cross-bank relative relocation not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}
