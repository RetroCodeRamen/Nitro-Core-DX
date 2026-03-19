package emulator

import (
	"bytes"
	"encoding/gob"
	"os"
	"path/filepath"
	"testing"

	"nitro-core-dx/internal/ppu"
)

// TestSaveLoadState tests that save/load state works correctly
func TestSaveLoadState(t *testing.T) {
	emu := NewEmulator()

	// Create minimal ROM
	romData := make([]uint8, 64)
	romData[0] = 0x52 // "RMCF"
	romData[1] = 0x4D
	romData[2] = 0x43
	romData[3] = 0x46
	romData[4] = 0x01  // Version 1
	romData[6] = 0x20  // ROM size 32
	romData[10] = 0x01 // Entry bank 1
	romData[12] = 0x00 // Entry offset 0x8000
	romData[13] = 0x80

	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// Modify some state
	emu.CPU.State.R0 = 0x1234
	emu.CPU.State.R1 = 0x5678
	emu.CPU.State.PCBank = 2
	emu.CPU.State.PCOffset = 0x9000
	emu.Bus.WRAM[0x1000] = 0xAB
	emu.Bus.WRAM[0x1001] = 0xCD
	emu.PPU.VRAM[0x2000] = 0xEF
	emu.PPU.CGRAM[0] = 0x12
	emu.PPU.FrameCounter = 42
	emu.APU.MasterVolume = 128
	emu.APU.Channels[0].Frequency = 440
	emu.APU.Channels[0].Volume = 200
	emu.Input.Controller1Buttons = 0x1234

	// Save state
	savedData, err := emu.SaveState()
	if err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	if len(savedData) == 0 {
		t.Fatal("SaveState returned empty data")
	}

	// Modify state to verify it changes
	emu.CPU.State.R0 = 0x9999
	emu.CPU.State.R1 = 0x8888
	emu.Bus.WRAM[0x1000] = 0xFF
	emu.PPU.VRAM[0x2000] = 0x00
	emu.PPU.FrameCounter = 999
	emu.APU.MasterVolume = 255
	emu.APU.Channels[0].Frequency = 880

	// Load state
	if err := emu.LoadState(savedData); err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	// Verify state was restored
	if emu.CPU.State.R0 != 0x1234 {
		t.Errorf("R0 not restored: expected 0x1234, got 0x%04X", emu.CPU.State.R0)
	}
	if emu.CPU.State.R1 != 0x5678 {
		t.Errorf("R1 not restored: expected 0x5678, got 0x%04X", emu.CPU.State.R1)
	}
	if emu.CPU.State.PCBank != 2 {
		t.Errorf("PCBank not restored: expected 2, got %d", emu.CPU.State.PCBank)
	}
	if emu.CPU.State.PCOffset != 0x9000 {
		t.Errorf("PCOffset not restored: expected 0x9000, got 0x%04X", emu.CPU.State.PCOffset)
	}
	if emu.Bus.WRAM[0x1000] != 0xAB {
		t.Errorf("WRAM[0x1000] not restored: expected 0xAB, got 0x%02X", emu.Bus.WRAM[0x1000])
	}
	if emu.Bus.WRAM[0x1001] != 0xCD {
		t.Errorf("WRAM[0x1001] not restored: expected 0xCD, got 0x%02X", emu.Bus.WRAM[0x1001])
	}
	if emu.PPU.VRAM[0x2000] != 0xEF {
		t.Errorf("VRAM[0x2000] not restored: expected 0xEF, got 0x%02X", emu.PPU.VRAM[0x2000])
	}
	if emu.PPU.CGRAM[0] != 0x12 {
		t.Errorf("CGRAM[0] not restored: expected 0x12, got 0x%02X", emu.PPU.CGRAM[0])
	}
	if emu.PPU.FrameCounter != 42 {
		t.Errorf("FrameCounter not restored: expected 42, got %d", emu.PPU.FrameCounter)
	}
	if emu.APU.MasterVolume != 128 {
		t.Errorf("MasterVolume not restored: expected 128, got %d", emu.APU.MasterVolume)
	}
	if emu.APU.Channels[0].Frequency != 440 {
		t.Errorf("Channel 0 Frequency not restored: expected 440, got %d", emu.APU.Channels[0].Frequency)
	}
	if emu.APU.Channels[0].Volume != 200 {
		t.Errorf("Channel 0 Volume not restored: expected 200, got %d", emu.APU.Channels[0].Volume)
	}
	if emu.Input.Controller1Buttons != 0x1234 {
		t.Errorf("Controller1Buttons not restored: expected 0x1234, got 0x%04X", emu.Input.Controller1Buttons)
	}
}

// TestSaveLoadStateFile tests file-based save/load state APIs.
func TestSaveLoadStateFile(t *testing.T) {
	emu := NewEmulator()

	romData := make([]uint8, 64)
	romData[0] = 0x52 // "RMCF"
	romData[1] = 0x4D
	romData[2] = 0x43
	romData[3] = 0x46
	romData[4] = 0x01  // Version 1
	romData[6] = 0x20  // ROM size 32
	romData[10] = 0x01 // Entry bank 1
	romData[12] = 0x00 // Entry offset 0x8000
	romData[13] = 0x80

	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	emu.CPU.State.R0 = 0xBEEF
	emu.CPU.State.R1 = 0xCAFE
	emu.Bus.WRAM[0x200] = 0x44
	emu.PPU.FrameCounter = 77
	emu.APU.MasterVolume = 99

	savePath := filepath.Join(t.TempDir(), "test_state.sav")
	if err := emu.SaveStateToFile(savePath); err != nil {
		t.Fatalf("SaveStateToFile failed: %v", err)
	}

	info, err := os.Stat(savePath)
	if err != nil {
		t.Fatalf("expected save file to exist: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("save file is empty")
	}

	// Mutate state to prove load restores from file.
	emu.CPU.State.R0 = 0x0000
	emu.CPU.State.R1 = 0x0000
	emu.Bus.WRAM[0x200] = 0x00
	emu.PPU.FrameCounter = 0
	emu.APU.MasterVolume = 0

	if err := emu.LoadStateFromFile(savePath); err != nil {
		t.Fatalf("LoadStateFromFile failed: %v", err)
	}

	if emu.CPU.State.R0 != 0xBEEF {
		t.Errorf("R0 not restored from file: expected 0xBEEF, got 0x%04X", emu.CPU.State.R0)
	}
	if emu.CPU.State.R1 != 0xCAFE {
		t.Errorf("R1 not restored from file: expected 0xCAFE, got 0x%04X", emu.CPU.State.R1)
	}
	if emu.Bus.WRAM[0x200] != 0x44 {
		t.Errorf("WRAM[0x200] not restored from file: expected 0x44, got 0x%02X", emu.Bus.WRAM[0x200])
	}
	if emu.PPU.FrameCounter != 77 {
		t.Errorf("FrameCounter not restored from file: expected 77, got %d", emu.PPU.FrameCounter)
	}
	if emu.APU.MasterVolume != 99 {
		t.Errorf("MasterVolume not restored from file: expected 99, got %d", emu.APU.MasterVolume)
	}
}

func TestSavePPUStatePersistsTransformChannels(t *testing.T) {
	emu := NewEmulator()

	emu.PPU.TransformChannels[0] = ppu.TransformChannel{
		Enabled: true,
		A:       0x0100,
		B:       0x0020,
		C:       -0x0010,
		D:       0x00C0,
		CenterX: 160,
		CenterY: 100,
		MirrorH: true,
		MirrorV: false,
	}

	state := emu.savePPUState()

	if state.TransformChannels[0] != emu.PPU.TransformChannels[0] {
		t.Fatal("transform channel 0 should persist exactly in saved state")
	}
}

func TestLoadPPUStateRestoresTransformChannels(t *testing.T) {
	emu := NewEmulator()

	state := PPUState{
		TransformChannels: [4]ppu.TransformChannel{
			{
				Enabled: true,
				A:       0x0100,
				B:       0x0020,
				C:       -0x0010,
				D:       0x00C0,
				CenterX: 160,
				CenterY: 100,
				MirrorH: true,
				MirrorV: true,
			},
		},
	}

	emu.loadPPUState(state, saveStateVersion2)

	if emu.PPU.TransformChannels[0] != state.TransformChannels[0] {
		t.Fatal("transform channel 0 should match loaded state")
	}
}

func TestSaveLoadStateRestoresMatrixPlanes(t *testing.T) {
	emu := NewEmulator()

	plane := &emu.PPU.MatrixPlanes[1]
	plane.Enabled = true
	plane.Size = ppu.TilemapSize64x64
	plane.SourceMode = ppu.MatrixPlaneSourceBitmap
	plane.BitmapPalette = 3
	plane.Transparent0 = true
	plane.TwoSided = true
	plane.RowModeEnabled = true
	plane.ProjectionMode = ppu.MatrixPlaneProjectionVertical
	plane.Horizon = 63
	plane.CameraX = 512
	plane.CameraY = 688
	plane.HeadingX = 0x0010
	plane.HeadingY = -0x0100
	plane.BaseDistance = 0x01C0
	plane.FocalLength = 0x3A00
	plane.WidthScale = 0x00B8
	plane.OriginX = 512
	plane.OriginY = 686
	plane.FacingX = 0
	plane.FacingY = 0x0100
	plane.HeightScale = 0x9A00
	plane.Tilemap[0x1234] = 0x56
	plane.Pattern[0x2345] = 0x78
	plane.Bitmap[0x3456] = 0x9A
	plane.Rows[17] = ppu.MatrixPlaneRowParams{
		StartX: 0x00112233,
		StartY: -0x00020304,
		StepX:  0x00001020,
		StepY:  -0x00003040,
	}

	emu.PPU.MatrixPlaneSelect = 1
	emu.PPU.MatrixPlaneAddr = 0x1200
	emu.PPU.MatrixPlanePatternAddr = 0x3400
	emu.PPU.MatrixPlaneBitmapAddr = 0x00045678
	emu.PPU.MatrixPlaneRowAddr = 0x5600
	emu.PPU.HDMAControl = 0x21
	emu.PPU.HDMAExtControl = 0x01
	emu.PPU.DMAEnabled = true
	emu.PPU.DMASourceBank = 2
	emu.PPU.DMASourceOffset = 0x8000
	emu.PPU.DMADestType = 6
	emu.PPU.DMADestAddr = 0x0040
	emu.PPU.DMALength = 0x0080
	emu.PPU.DMAMode = 1
	emu.PPU.DMACycles = 99
	emu.PPU.DMAProgress = 17
	emu.PPU.DMACurrentSrc = 0x8011
	emu.PPU.DMACurrentDest = 0x0051
	emu.PPU.DMAFillValue = 0xCC

	savedData, err := emu.SaveState()
	if err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	plane.Enabled = false
	plane.Size = ppu.TilemapSize32x32
	plane.SourceMode = ppu.MatrixPlaneSourceTilemap
	plane.BitmapPalette = 0
	plane.Transparent0 = false
	plane.TwoSided = false
	plane.RowModeEnabled = false
	plane.ProjectionMode = ppu.MatrixPlaneProjectionNone
	plane.Horizon = 0
	plane.CameraX = 0
	plane.CameraY = 0
	plane.HeadingX = 0
	plane.HeadingY = 0
	plane.BaseDistance = 0
	plane.FocalLength = 0
	plane.WidthScale = 0
	plane.OriginX = 0
	plane.OriginY = 0
	plane.FacingX = 0
	plane.FacingY = 0
	plane.HeightScale = 0
	plane.Tilemap[0x1234] = 0
	plane.Pattern[0x2345] = 0
	plane.Bitmap[0x3456] = 0
	plane.Rows[17] = ppu.MatrixPlaneRowParams{}

	emu.PPU.MatrixPlaneSelect = 0
	emu.PPU.MatrixPlaneAddr = 0
	emu.PPU.MatrixPlanePatternAddr = 0
	emu.PPU.MatrixPlaneBitmapAddr = 0
	emu.PPU.MatrixPlaneRowAddr = 0
	emu.PPU.HDMAControl = 0
	emu.PPU.HDMAExtControl = 0
	emu.PPU.DMAEnabled = false
	emu.PPU.DMASourceBank = 0
	emu.PPU.DMASourceOffset = 0
	emu.PPU.DMADestType = 0
	emu.PPU.DMADestAddr = 0
	emu.PPU.DMALength = 0
	emu.PPU.DMAMode = 0
	emu.PPU.DMACycles = 0
	emu.PPU.DMAProgress = 0
	emu.PPU.DMACurrentSrc = 0
	emu.PPU.DMACurrentDest = 0
	emu.PPU.DMAFillValue = 0

	if err := emu.LoadState(savedData); err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	plane = &emu.PPU.MatrixPlanes[1]
	if !plane.Enabled || plane.Size != ppu.TilemapSize64x64 || plane.SourceMode != ppu.MatrixPlaneSourceBitmap {
		t.Fatal("matrix plane 1 control fields were not restored")
	}
	if plane.BitmapPalette != 3 || !plane.Transparent0 || !plane.TwoSided || !plane.RowModeEnabled {
		t.Fatal("matrix plane 1 flags were not restored")
	}
	if plane.ProjectionMode != ppu.MatrixPlaneProjectionVertical || plane.Horizon != 63 {
		t.Fatal("matrix plane 1 projection fields were not restored")
	}
	if plane.CameraX != 512 || plane.CameraY != 688 || plane.OriginY != 686 || plane.HeightScale != 0x9A00 {
		t.Fatal("matrix plane 1 geometry fields were not restored")
	}
	if plane.Tilemap[0x1234] != 0x56 || plane.Pattern[0x2345] != 0x78 || plane.Bitmap[0x3456] != 0x9A {
		t.Fatal("matrix plane backing data was not restored")
	}
	if got := plane.Rows[17]; got != (ppu.MatrixPlaneRowParams{
		StartX: 0x00112233,
		StartY: -0x00020304,
		StepX:  0x00001020,
		StepY:  -0x00003040,
	}) {
		t.Fatalf("matrix plane row params not restored: got %+v", got)
	}
	if emu.PPU.MatrixPlaneSelect != 1 || emu.PPU.MatrixPlaneAddr != 0x1200 || emu.PPU.MatrixPlanePatternAddr != 0x3400 {
		t.Fatal("matrix plane MMIO addressing registers were not restored")
	}
	if emu.PPU.MatrixPlaneBitmapAddr != 0x00045678 || emu.PPU.MatrixPlaneRowAddr != 0x5600 {
		t.Fatal("matrix plane bitmap/row address registers were not restored")
	}
	if emu.PPU.HDMAControl != 0x21 || emu.PPU.HDMAExtControl != 0x01 {
		t.Fatal("HDMA control registers were not restored")
	}
	if !emu.PPU.DMAEnabled || emu.PPU.DMASourceBank != 2 || emu.PPU.DMADestType != 6 || emu.PPU.DMAFillValue != 0xCC {
		t.Fatal("DMA state was not restored")
	}
}

func TestLoadStateVersion1ResetsExtendedMatrixPlaneState(t *testing.T) {
	emu := NewEmulator()
	emu.PPU.MatrixPlanes[0].Enabled = true
	emu.PPU.MatrixPlanes[0].BaseDistance = 0x7777
	emu.PPU.MatrixPlanes[0].FocalLength = 0x6666
	emu.PPU.MatrixPlanes[0].WidthScale = 0x5555
	emu.PPU.MatrixPlanes[0].HeightScale = 0x4444
	emu.PPU.MatrixPlanes[0].HeadingY = 0x0123
	emu.PPU.MatrixPlanes[0].FacingY = 0x0456
	emu.PPU.MatrixPlanes[0].Bitmap[7] = 0xAA
	emu.PPU.MatrixPlaneSelect = 3
	emu.PPU.HDMAControl = 0xFF
	emu.PPU.DMAEnabled = true

	state := SaveState{
		Version: saveStateVersion1,
		PPUState: PPUState{
			BG0: ppu.BackgroundLayer{Enabled: true},
		},
	}

	savedData, err := encodeSaveStateForTest(state)
	if err != nil {
		t.Fatalf("encode save state: %v", err)
	}

	if err := emu.LoadState(savedData); err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	plane := emu.PPU.MatrixPlanes[0]
	if plane.Enabled {
		t.Fatal("version 1 load should reset matrix plane enabled state")
	}
	if plane.BaseDistance != 0x0100 || plane.FocalLength != 0x3000 || plane.WidthScale != 0x0100 || plane.HeightScale != 0x0200 {
		t.Fatalf("version 1 load should restore default matrix plane parameters, got base=0x%04X focal=0x%04X width=0x%04X height=0x%04X",
			plane.BaseDistance, plane.FocalLength, plane.WidthScale, plane.HeightScale)
	}
	if plane.HeadingY != -0x0100 || plane.FacingY != -0x0100 {
		t.Fatalf("version 1 load should restore default facing, got headingY=0x%04X facingY=0x%04X", uint16(plane.HeadingY), uint16(plane.FacingY))
	}
	if plane.Bitmap[7] != 0 {
		t.Fatal("version 1 load should clear stale matrix plane backing data")
	}
	if emu.PPU.MatrixPlaneSelect != 0 || emu.PPU.HDMAControl != 0 || emu.PPU.DMAEnabled {
		t.Fatal("version 1 load should clear extended matrix plane control state")
	}
}

func encodeSaveStateForTest(state SaveState) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(state); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
