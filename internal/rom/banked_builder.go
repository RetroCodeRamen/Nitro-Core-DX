package rom

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

const (
	// LoROM bank layout used by Nitro-Core-DX.
	ROMBankOffsetBase = 0x8000
	ROMBankSizeBytes  = 0x8000
	ROMBankSizeWords  = ROMBankSizeBytes / 2
	ROMMinProgramBank = 1
	ROMMaxProgramBank = 125
)

type RelocKind uint8

const (
	// RelocRelative16 patches a 16-bit signed relative branch/jump/call offset.
	// CurrentPC must point to the offset word address (same convention as CalculateBranchOffset).
	RelocRelative16 RelocKind = iota
)

type bankRelocation struct {
	wordIndex   int
	currentPC   uint16
	targetBank  uint8
	targetLabel string
	kind        RelocKind
}

type bankProgram struct {
	code   []uint16
	labels map[string]uint16 // absolute bank-local offsets (0x8000+)
	relocs []bankRelocation
}

// BankedROMBuilder is a bank-aware ROM code/image builder skeleton.
// It supports per-bank code streams plus relocations, while preserving the current ROM file format.
type BankedROMBuilder struct {
	banks map[uint8]*bankProgram
}

func NewBankedROMBuilder() *BankedROMBuilder {
	return &BankedROMBuilder{
		banks: make(map[uint8]*bankProgram),
	}
}

func (b *BankedROMBuilder) bank(bank uint8) *bankProgram {
	if bank < ROMMinProgramBank || bank > ROMMaxProgramBank {
		panic(fmt.Sprintf("invalid ROM bank %d (expected %d-%d)", bank, ROMMinProgramBank, ROMMaxProgramBank))
	}
	p := b.banks[bank]
	if p == nil {
		p = &bankProgram{
			code:   make([]uint16, 0),
			labels: make(map[string]uint16),
		}
		b.banks[bank] = p
	}
	return p
}

func (b *BankedROMBuilder) AddInstruction(bank uint8, instruction uint16) {
	p := b.bank(bank)
	if len(p.code) >= ROMBankSizeWords {
		panic(fmt.Sprintf("bank %d code overflow: exceeds %d words", bank, ROMBankSizeWords))
	}
	p.code = append(p.code, instruction)
}

func (b *BankedROMBuilder) AddImmediate(bank uint8, value uint16) {
	b.AddInstruction(bank, value)
}

func (b *BankedROMBuilder) SetImmediateAt(bank uint8, wordIndex int, value uint16) {
	p := b.bank(bank)
	if wordIndex < 0 || wordIndex >= len(p.code) {
		panic(fmt.Sprintf("SetImmediateAt(bank=%d): index %d out of range (len=%d)", bank, wordIndex, len(p.code)))
	}
	p.code[wordIndex] = value
}

func (b *BankedROMBuilder) GetCodeLength(bank uint8) int {
	return len(b.bank(bank).code)
}

// PC returns the current bank-local program counter (offset within the CPU bank space, 0x8000+).
func (b *BankedROMBuilder) PC(bank uint8) uint16 {
	return uint16(ROMBankOffsetBase + (b.GetCodeLength(bank) * 2))
}

func (b *BankedROMBuilder) MarkLabel(bank uint8, name string) {
	p := b.bank(bank)
	p.labels[name] = b.PC(bank)
}

// AddRelative16Relocation registers a same-bank relative relocation for the last-written placeholder word.
// targetBank is included for future cross-bank relocation support; current implementation only supports same-bank.
func (b *BankedROMBuilder) AddRelative16Relocation(bank uint8, wordIndex int, currentPC uint16, targetBank uint8, targetLabel string) {
	p := b.bank(bank)
	p.relocs = append(p.relocs, bankRelocation{
		wordIndex:   wordIndex,
		currentPC:   currentPC,
		targetBank:  targetBank,
		targetLabel: targetLabel,
		kind:        RelocRelative16,
	})
}

func (b *BankedROMBuilder) ResolveRelocations() error {
	for srcBank, p := range b.banks {
		for _, r := range p.relocs {
			switch r.kind {
			case RelocRelative16:
				if r.targetBank != srcBank {
					return fmt.Errorf("cross-bank relative relocation not supported yet: %d:%04X -> bank %d label %q",
						srcBank, r.currentPC, r.targetBank, r.targetLabel)
				}
				targetProg := b.bank(r.targetBank)
				targetPC, ok := targetProg.labels[r.targetLabel]
				if !ok {
					return fmt.Errorf("unknown label %q in bank %d", r.targetLabel, r.targetBank)
				}
				offset := CalculateBranchOffset(r.currentPC, targetPC)
				b.SetImmediateAt(srcBank, r.wordIndex, uint16(offset))
			default:
				return fmt.Errorf("unsupported relocation kind %d", r.kind)
			}
		}
	}
	return nil
}

// BuildROMBytes builds a ROM image (header + padded per-bank LoROM data) in memory.
func (b *BankedROMBuilder) BuildROMBytes(entryBank uint8, entryOffset uint16) ([]byte, error) {
	if len(b.banks) == 0 {
		return nil, fmt.Errorf("no banked ROM code added")
	}
	if entryBank < ROMMinProgramBank || entryBank > ROMMaxProgramBank {
		return nil, fmt.Errorf("invalid entry bank %d", entryBank)
	}
	if entryOffset < ROMBankOffsetBase {
		return nil, fmt.Errorf("invalid entry offset 0x%04X (must be >= 0x%04X)", entryOffset, ROMBankOffsetBase)
	}

	if err := b.ResolveRelocations(); err != nil {
		return nil, err
	}

	usedBanks := make([]int, 0, len(b.banks))
	for bank := range b.banks {
		usedBanks = append(usedBanks, int(bank))
	}
	sort.Ints(usedBanks)

	highestBank := uint8(usedBanks[len(usedBanks)-1])
	romSize := uint32(highestBank) * ROMBankSizeBytes // bank 1 starts at ROM offset 0
	romData := make([]byte, 32+romSize)

	// Header (same format as ROMBuilder)
	binary.LittleEndian.PutUint32(romData[0:4], 0x46434D52) // "RMCF"
	binary.LittleEndian.PutUint16(romData[4:6], 1)          // version
	binary.LittleEndian.PutUint32(romData[6:10], romSize)   // size
	binary.LittleEndian.PutUint16(romData[10:12], uint16(entryBank))
	binary.LittleEndian.PutUint16(romData[12:14], entryOffset)
	binary.LittleEndian.PutUint16(romData[14:16], 0) // mapper flags (LoROM)
	binary.LittleEndian.PutUint32(romData[16:20], 0) // checksum unused

	// Write bank payloads padded to 32KB each.
	for bank, p := range b.banks {
		if len(p.code) > ROMBankSizeWords {
			return nil, fmt.Errorf("bank %d overflow: %d words > %d", bank, len(p.code), ROMBankSizeWords)
		}
		base := 32 + int(bank-1)*ROMBankSizeBytes
		for i, word := range p.code {
			off := base + i*2
			binary.LittleEndian.PutUint16(romData[off:off+2], word)
		}
	}

	return romData, nil
}

func (b *BankedROMBuilder) BuildROM(entryBank uint8, entryOffset uint16, outputPath string) error {
	data, err := b.BuildROMBytes(entryBank, entryOffset)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}
