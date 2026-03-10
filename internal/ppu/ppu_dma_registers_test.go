package ppu

import (
	"testing"

	"nitro-core-dx/internal/debug"
)

func TestDMARegisterReadMapping(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.DMASourceBank = 0x12
	ppu.DMASourceOffset = 0x3456
	ppu.DMADestAddr = 0x789A
	ppu.DMALength = 0xBEEF
	ppu.DMAEnabled = true
	ppu.DMAProgress = 0

	if got := ppu.Read8(0x60); got != 0x01 {
		t.Fatalf("DMA_STATUS = 0x%02X, want 0x01", got)
	}
	if got := ppu.Read8(0x61); got != 0x12 {
		t.Fatalf("DMA_SOURCE_BANK = 0x%02X, want 0x12", got)
	}
	if got := ppu.Read8(0x62); got != 0x56 {
		t.Fatalf("DMA_SOURCE_OFFSET_L = 0x%02X, want 0x56", got)
	}
	if got := ppu.Read8(0x63); got != 0x34 {
		t.Fatalf("DMA_SOURCE_OFFSET_H = 0x%02X, want 0x34", got)
	}
	if got := ppu.Read8(0x64); got != 0x9A {
		t.Fatalf("DMA_DEST_ADDR_L = 0x%02X, want 0x9A", got)
	}
	if got := ppu.Read8(0x65); got != 0x78 {
		t.Fatalf("DMA_DEST_ADDR_H = 0x%02X, want 0x78", got)
	}
	if got := ppu.Read8(0x66); got != 0xEF {
		t.Fatalf("DMA_LENGTH_L = 0x%02X, want 0xEF", got)
	}
	if got := ppu.Read8(0x67); got != 0xBE {
		t.Fatalf("DMA_LENGTH_H = 0x%02X, want 0xBE", got)
	}
}
