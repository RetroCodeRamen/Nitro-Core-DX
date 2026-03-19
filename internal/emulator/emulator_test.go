package emulator

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nitro-core-dx/internal/apu"
)

// TestResetReloadsEntryPoint tests that emulator.Reset() reloads entry point correctly
func TestResetReloadsEntryPoint(t *testing.T) {
	emu := NewEmulator()

	// Create a minimal ROM with entry point at bank 1, offset 0x8000
	romData := make([]uint8, 64)
	// Magic: "RMCF"
	romData[0] = 0x52
	romData[1] = 0x4D
	romData[2] = 0x43
	romData[3] = 0x46
	// Version: 1
	romData[4] = 0x01
	romData[5] = 0x00
	// ROM Size: 32 bytes
	romData[6] = 0x20
	romData[7] = 0x00
	romData[8] = 0x00
	romData[9] = 0x00
	// Entry Bank: 1
	romData[10] = 0x01
	romData[11] = 0x00
	// Entry Offset: 0x8000
	romData[12] = 0x00
	romData[13] = 0x80

	// Load ROM
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// Verify entry point is set
	if emu.CPU.State.PCBank != 1 {
		t.Errorf("Expected PCBank=1 after LoadROM, got %d", emu.CPU.State.PCBank)
	}
	if emu.CPU.State.PCOffset != 0x8000 {
		t.Errorf("Expected PCOffset=0x8000 after LoadROM, got 0x%04X", emu.CPU.State.PCOffset)
	}

	// Modify PC to simulate execution
	emu.CPU.State.PCBank = 2
	emu.CPU.State.PCOffset = 0x9000

	// Call Reset() - should reload entry point
	emu.Reset()

	// Verify entry point is reloaded correctly
	if emu.CPU.State.PCBank != 1 {
		t.Errorf("After Reset(): Expected PCBank=1, got %d (entry point should be reloaded)", emu.CPU.State.PCBank)
	}
	if emu.CPU.State.PCOffset != 0x8000 {
		t.Errorf("After Reset(): Expected PCOffset=0x8000, got 0x%04X (entry point should be reloaded)", emu.CPU.State.PCOffset)
	}
}

func TestFMExtensionProgrammingThroughEmulatorSmoke(t *testing.T) {
	t.Setenv("NCDX_YM_BACKEND", "ymfm")
	emu := NewEmulator()

	emu.APU.Write8(apu.FMExtensionOffsetBase+apu.FMRegControl, 0x01) // enable FM
	emu.APU.Write8(apu.FMExtensionOffsetBase+apu.FMRegAddr, 0x10)
	emu.APU.Write8(apu.FMExtensionOffsetBase+apu.FMRegData, 0xFF)
	emu.APU.Write8(apu.FMExtensionOffsetBase+apu.FMRegAddr, 0x11)
	emu.APU.Write8(apu.FMExtensionOffsetBase+apu.FMRegData, 0x03)
	emu.APU.Write8(apu.FMExtensionOffsetBase+apu.FMRegAddr, 0x14)
	emu.APU.Write8(apu.FMExtensionOffsetBase+apu.FMRegData, 0x11) // start A + IRQ enable A

	if err := emu.APU.StepAPU(4096); err != nil {
		t.Fatalf("APU step failed: %v", err)
	}
	_ = emu.APU.Read8(apu.FMExtensionOffsetBase + apu.FMRegStatus)
}

func TestRunFrameUsesFixedPointAPUPath(t *testing.T) {
	emu := NewEmulator()

	romData := make([]uint8, 64)
	romData[0] = 0x52
	romData[1] = 0x4D
	romData[2] = 0x43
	romData[3] = 0x46
	romData[4] = 0x01
	romData[6] = 0x20
	romData[10] = 0x01
	romData[12] = 0x00
	romData[13] = 0x80
	romData[32] = 0x00 // NOP low
	romData[33] = 0x00 // NOP high
	romData[34] = 0xD0 // JMP opcode
	romData[35] = 0x00
	romData[36] = 0xFD // relative offset low
	romData[37] = 0xFF // relative offset high

	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()
	emu.Clock.CPUStep = func(cycles uint64) error {
		emu.CPU.State.Cycles += uint32(cycles)
		return nil
	}

	emu.APU.Write8(0x00, 0xB8) // freq low
	emu.APU.Write8(0x01, 0x01) // freq high => 440 Hz
	emu.APU.Write8(0x02, 0xFF) // volume
	emu.APU.Write8(0x03, 0x03) // enable + square wave

	// Corrupt the deprecated float-phase state. RunFrame should still use the
	// fixed-point path and generate audio samples from PhaseFixed/PhaseIncrementFixed.
	emu.APU.Channels[0].Phase = math.NaN()
	emu.APU.Channels[0].PhaseIncrement = 0

	if err := emu.RunFrame(); err != nil {
		t.Fatalf("RunFrame failed: %v", err)
	}

	nonZero := false
	for _, sample := range emu.AudioSampleBuffer {
		if sample != 0 {
			nonZero = true
			break
		}
	}
	if !nonZero {
		t.Fatal("expected non-zero audio from fixed-point APU path during RunFrame")
	}
}

func TestFrameCountRemainsMonotonicAcrossFPSUpdates(t *testing.T) {
	emu := NewEmulator()

	romData := make([]uint8, 64)
	romData[0] = 0x52
	romData[1] = 0x4D
	romData[2] = 0x43
	romData[3] = 0x46
	romData[4] = 0x01
	romData[6] = 0x20
	romData[10] = 0x01
	romData[12] = 0x00
	romData[13] = 0x80

	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()
	emu.FrameLimitEnabled = false
	emu.Clock.CPUStep = func(cycles uint64) error {
		emu.CPU.State.Cycles += uint32(cycles)
		return nil
	}

	if err := emu.RunFrame(); err != nil {
		t.Fatalf("RunFrame failed: %v", err)
	}
	if emu.FrameCount != 1 {
		t.Fatalf("FrameCount after first frame = %d, want 1", emu.FrameCount)
	}

	// Force the FPS refresh path on the next frame. Total frame count must still
	// increase monotonically even though the FPS accumulator resets.
	emu.FPSUpdateTime = time.Now().Add(-2 * time.Second)
	if err := emu.RunFrame(); err != nil {
		t.Fatalf("RunFrame failed on forced FPS update: %v", err)
	}
	if emu.FrameCount != 2 {
		t.Fatalf("FrameCount after FPS update = %d, want 2", emu.FrameCount)
	}
}

// framebufferChecksum returns a SHA256 hex of the display buffer (used to detect when frame content changes).
func framebufferChecksum(buf []uint32) string {
	raw := make([]byte, len(buf)*4)
	for i, px := range buf {
		raw[i*4+0] = byte(px)
		raw[i*4+1] = byte(px >> 8)
		raw[i*4+2] = byte(px >> 16)
		raw[i*4+3] = byte(px >> 24)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// TestMatrixFloorBillboardReferenceFramebufferChangesWithInput verifies that the matrix floor+billboard
// reference ROM updates the displayed framebuffer when controller input is applied (WRAM and PPU plane
// state change; the picture must change too). If this test fails, the PPU is not rendering from the
// updated matrix plane camera/heading/row state.
func TestMatrixFloorBillboardReferenceFramebufferChangesWithInput(t *testing.T) {
	// Try multiple paths: module root (go test from repo) and package dir (go test ./internal/emulator).
	possiblePaths := []string{
		"roms/matrix_floor_billboard_reference.rom",
		"../roms/matrix_floor_billboard_reference.rom",
		"../../roms/matrix_floor_billboard_reference.rom",
	}
	var romPath string
	for _, p := range possiblePaths {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(abs); err == nil {
			romPath = abs
			break
		}
		if _, err := os.Stat(p); err == nil {
			romPath = p
			break
		}
	}
	if romPath == "" {
		t.Skip("matrix_floor_billboard_reference.rom not found (build with: go run ./test/roms/build_matrix_floor_billboard_reference.go)")
	}

	data, err := os.ReadFile(romPath)
	if err != nil {
		t.Fatalf("read ROM: %v", err)
	}

	emu := NewEmulator()
	if err := emu.LoadROM(data); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	emu.Running = true
	emu.SetFrameLimit(false)

	// Warmup so ROM is past init and in main loop
	for i := 0; i < 5; i++ {
		emu.SetInputButtons(0)
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("warmup RunFrame: %v", err)
		}
	}

	// Baseline: a few frames with no input
	emu.SetInputButtons(0)
	for i := 0; i < 10; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("baseline RunFrame: %v", err)
		}
	}
	checksumNoInput := framebufferChecksum(emu.GetOutputBuffer())

	// Apply forward (UP) input for several frames so camera/heading and matrix plane state change
	emu.SetInputButtons(0x0001) // UP
	for i := 0; i < 20; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("forward RunFrame: %v", err)
		}
	}
	checksumWithInput := framebufferChecksum(emu.GetOutputBuffer())

	// After running with input, WRAM camera should have changed and the ROM writes that to plane 1 each frame.
	// If plane 1 matches WRAM but the framebuffer didn't change, the render path isn't using the plane state.
	wramCamX := emu.Bus.Read16(0, 0x0204)
	wramCamY := emu.Bus.Read16(0, 0x0206)
	plane1 := &emu.PPU.MatrixPlanes[1]
	planeCamX := uint16(int16(plane1.CameraX))
	planeCamY := uint16(int16(plane1.CameraY))
	if wramCamX != planeCamX || wramCamY != planeCamY {
		t.Logf("WRAM camera=(%d,%d) plane1.Camera=(%d,%d) — ROM may not be writing plane 1 each frame or timing differs",
			wramCamX, wramCamY, plane1.CameraX, plane1.CameraY)
	}

	if checksumNoInput == checksumWithInput {
		t.Errorf("framebuffer did not change after applying input: checksum %s (no input) == %s (with UP); "+
			"PPU should render from updated matrix plane camera/heading", checksumNoInput, checksumWithInput)
	}
}

func TestMatrixFloorBillboardGenericFramebufferChangesWithInput(t *testing.T) {
	data, err := os.ReadFile("../../roms/matrix_floor_billboard_generic.rom")
	if err != nil {
		t.Fatalf("read ROM: %v", err)
	}

	emu := NewEmulator()
	if err := emu.LoadROM(data); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	emu.Running = true
	emu.SetFrameLimit(false)

	// Warmup so ROM finishes init.
	for i := 0; i < 10; i++ {
		emu.SetInputButtons(0)
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("warmup RunFrame: %v", err)
		}
	}

	// Baseline frame (no input).
	emu.SetInputButtons(0)
	if err := emu.RunFrame(); err != nil {
		t.Fatalf("baseline RunFrame: %v", err)
	}
	before := framebufferChecksum(emu.GetOutputBuffer())

	// Apply UP for a few frames.
	for i := 0; i < 10; i++ {
		emu.SetInputButtons(0x0001)
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("input RunFrame: %v", err)
		}
	}
	after := framebufferChecksum(emu.GetOutputBuffer())

	wramCamX := int16(emu.Bus.Read16(0, 0x0204))
	wramCamY := int16(emu.Bus.Read16(0, 0x0206))
	p0 := &emu.PPU.MatrixPlanes[0]
	p1 := &emu.PPU.MatrixPlanes[1]
	t.Logf("WRAM camera=(%d,%d) plane0.Camera=(%d,%d) plane1.Camera=(%d,%d)",
		wramCamX, wramCamY, int16(p0.CameraX), int16(p0.CameraY), int16(p1.CameraX), int16(p1.CameraY))

	if before == after {
		t.Fatalf("framebuffer did not change after applying input: checksum %s == %s", before, after)
	}
}
