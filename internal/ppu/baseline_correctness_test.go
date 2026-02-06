package ppu

import (
	"testing"

	"nitro-core-dx/internal/debug"
)

// TestVBlankIRQTiming tests VBlank IRQ timing
// ISSUE: VBlank flag and IRQ timing may not match spec
func TestVBlankIRQTiming(t *testing.T) {
	logger := debug.NewLogger(10000)
	ppu := NewPPU(logger)

	irqTriggered := false
	irqScanline := -1
	ppu.InterruptCallback = func(interruptType uint8) {
		if interruptType == 1 { // INT_VBLANK
			irqTriggered = true
			irqScanline = ppu.currentScanline
		}
	}

	// Step through frame
	// VBlank should start at scanline 200 (end of visible scanlines)
	// IRQ should be triggered at the END of scanline 199 (before incrementing to 200)

	// Step to end of scanline 199
	cyclesToScanline199End := uint64(199*DotsPerScanline + DotsPerScanline - 1)
	if err := ppu.StepPPU(cyclesToScanline199End); err != nil {
		t.Fatalf("StepPPU failed: %v", err)
	}

	// At end of scanline 199, VBlank flag should be set and IRQ triggered
	// Step one more cycle to trigger end-of-scanline logic
	if err := ppu.StepPPU(1); err != nil {
		t.Fatalf("StepPPU (end of scanline) failed: %v", err)
	}

	// Check that VBlank flag is set
	if !ppu.VBlankFlag {
		t.Error("VBlank flag should be set at start of VBlank period")
	}

	// Check that IRQ was triggered
	if !irqTriggered {
		t.Error("VBlank IRQ should be triggered at start of VBlank period")
	}

	// Check that IRQ was triggered at correct scanline (should be 199, before incrementing)
	if irqScanline != 199 {
		t.Errorf("VBlank IRQ should be triggered at scanline 199, got %d", irqScanline)
	}

	// Check that current scanline is now 200 (VBlank period)
	if ppu.currentScanline != 200 {
		t.Errorf("After VBlank start, scanline should be 200, got %d", ppu.currentScanline)
	}
}

// TestDMALengthVsCycles tests DMA length vs cycles
// ISSUE: DMA may not advance proportionally per consumed cycle
func TestDMALengthVsCycles(t *testing.T) {
	logger := debug.NewLogger(10000)
	ppu := NewPPU(logger)

	// Set up memory reader
	memoryData := make([]uint8, 65536)
	for i := range memoryData {
		memoryData[i] = uint8(i & 0xFF)
	}
	ppu.MemoryReader = func(bank uint8, offset uint16) uint8 {
		if bank == 1 && offset >= 0x8000 {
			romOffset := offset - 0x8000
			if romOffset < uint16(len(memoryData)) {
				return memoryData[romOffset]
			}
		}
		return 0
	}

	// Set up DMA transfer: copy 100 bytes from ROM to VRAM
	ppu.DMASourceBank = 1
	ppu.DMASourceOffset = 0x8000
	ppu.DMADestType = 0 // VRAM
	ppu.DMADestAddr = 0x0000
	ppu.DMALength = 100
	ppu.DMAMode = 0 // Copy mode

	// Enable DMA
	ppu.DMAEnabled = true
	ppu.DMAProgress = 0
	ppu.DMACurrentSrc = ppu.DMASourceOffset
	ppu.DMACurrentDest = ppu.DMADestAddr

	// Step PPU for 50 cycles - DMA should transfer 50 bytes (1 byte per cycle)
	if err := ppu.StepPPU(50); err != nil {
		t.Fatalf("StepPPU failed: %v", err)
	}

	// Check DMA progress
	expectedProgress := uint16(50)
	if ppu.DMAProgress != expectedProgress {
		t.Errorf("DMA progress: Expected %d bytes transferred after 50 cycles, got %d", expectedProgress, ppu.DMAProgress)
	}

	// Check that VRAM was written correctly
	for i := uint16(0); i < 50; i++ {
		expectedValue := memoryData[0x8000+i]
		if ppu.VRAM[i] != expectedValue {
			t.Errorf("DMA VRAM write: Expected VRAM[%d]=0x%02X, got 0x%02X", i, expectedValue, ppu.VRAM[i])
		}
	}

	// Step another 50 cycles - should complete DMA
	if err := ppu.StepPPU(50); err != nil {
		t.Fatalf("StepPPU (complete DMA) failed: %v", err)
	}

	// Check DMA is complete
	if ppu.DMAEnabled {
		t.Error("DMA should be disabled after transfer completes")
	}
	if ppu.DMAProgress != 0 {
		t.Errorf("DMA progress should be reset after completion, got %d", ppu.DMAProgress)
	}

	// Check all 100 bytes were transferred
	for i := uint16(0); i < 100; i++ {
		expectedValue := memoryData[0x8000+i]
		if ppu.VRAM[i] != expectedValue {
			t.Errorf("DMA VRAM write: Expected VRAM[%d]=0x%02X, got 0x%02X", i, expectedValue, ppu.VRAM[i])
		}
	}
}

// TestDMADuringScanline tests DMA stepping during scanline rendering
// ISSUE: DMA timing must be correct even during scanline inner loops
func TestDMADuringScanline(t *testing.T) {
	logger := debug.NewLogger(10000)
	ppu := NewPPU(logger)

	// Set up memory reader
	memoryData := make([]uint8, 65536)
	for i := range memoryData {
		memoryData[i] = uint8((i + 0x42) & 0xFF)
	}
	ppu.MemoryReader = func(bank uint8, offset uint16) uint8 {
		if bank == 1 && offset >= 0x8000 {
			romOffset := offset - 0x8000
			if romOffset < uint16(len(memoryData)) {
				return memoryData[romOffset]
			}
		}
		return 0
	}

	// Set up DMA transfer
	ppu.DMASourceBank = 1
	ppu.DMASourceOffset = 0x8000
	ppu.DMADestType = 0 // VRAM
	ppu.DMADestAddr = 0x1000
	ppu.DMALength = 320 // Transfer during one scanline (320 visible dots)
	ppu.DMAMode = 0     // Copy mode
	ppu.DMAEnabled = true
	ppu.DMAProgress = 0
	ppu.DMACurrentSrc = ppu.DMASourceOffset
	ppu.DMACurrentDest = ppu.DMADestAddr

	// Step PPU for one full scanline (581 cycles)
	// DMA should transfer 320 bytes (one per visible dot) + some during HBlank
	if err := ppu.StepPPU(DotsPerScanline); err != nil {
		t.Fatalf("StepPPU (one scanline) failed: %v", err)
	}

	// Check DMA progress - should have advanced by at least 320 bytes (one per visible dot)
	if ppu.DMAProgress < 320 {
		t.Errorf("DMA progress: Expected at least 320 bytes transferred during scanline, got %d", ppu.DMAProgress)
	}

	// Check that DMA advances proportionally
	// After 581 cycles, DMA should have transferred approximately 581 bytes (if not complete)
	// But DMA may complete before scanline ends if length < 581
	if ppu.DMAProgress < 320 {
		t.Errorf("DMA progress during scanline: Expected at least 320 bytes, got %d", ppu.DMAProgress)
	}
}
