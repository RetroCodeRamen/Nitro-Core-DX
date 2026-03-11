package emulator

import (
	"testing"

	"nitro-core-dx/internal/ppu"
)

func TestInstallMatrixPlaneProgramProgramsPPUMMIO(t *testing.T) {
	emu := NewEmulator()

	tilemap := make([]byte, 32*32*2)
	tilemap[0] = 0x12
	tilemap[1] = 0x34
	tilemap[len(tilemap)-2] = 0x56
	tilemap[len(tilemap)-1] = 0x78

	err := emu.InstallMatrixPlaneProgram(MatrixPlaneProgram{
		Channel: 2,
		Enabled: true,
		Size:    ppu.TilemapSize32x32,
		Tilemap: tilemap,
		Pattern: []byte{0x9A, 0xBC, 0xDE},
	})
	if err != nil {
		t.Fatalf("InstallMatrixPlaneProgram failed: %v", err)
	}

	plane := emu.PPU.MatrixPlanes[2]
	if !plane.Enabled {
		t.Fatal("expected installed matrix plane to be enabled")
	}
	if plane.Size != ppu.TilemapSize32x32 {
		t.Fatalf("plane.Size = %d, want %d", plane.Size, ppu.TilemapSize32x32)
	}
	if plane.Tilemap[0] != 0x12 || plane.Tilemap[1] != 0x34 {
		t.Fatalf("plane tilemap first entry = [%02X %02X], want [12 34]", plane.Tilemap[0], plane.Tilemap[1])
	}
	if plane.Tilemap[len(tilemap)-2] != 0x56 || plane.Tilemap[len(tilemap)-1] != 0x78 {
		t.Fatalf("plane tilemap last entry = [%02X %02X], want [56 78]", plane.Tilemap[len(tilemap)-2], plane.Tilemap[len(tilemap)-1])
	}
	if plane.Pattern[0] != 0x9A || plane.Pattern[1] != 0xBC || plane.Pattern[2] != 0xDE {
		t.Fatalf("plane pattern first bytes = [%02X %02X %02X], want [9A BC DE]", plane.Pattern[0], plane.Pattern[1], plane.Pattern[2])
	}
}

func TestInstallMatrixPlaneProgramProgramsBitmapMode(t *testing.T) {
	emu := NewEmulator()

	tilemap := make([]byte, 32*32*2)
	bitmap := []byte{0x12, 0x34, 0x56}
	err := emu.InstallMatrixPlaneProgram(MatrixPlaneProgram{
		Channel:       1,
		Enabled:       true,
		Size:          ppu.TilemapSize32x32,
		SourceMode:    ppu.MatrixPlaneSourceBitmap,
		BitmapPalette: 3,
		Transparent0:  true,
		Tilemap:       tilemap,
		Bitmap:        bitmap,
	})
	if err != nil {
		t.Fatalf("InstallMatrixPlaneProgram failed: %v", err)
	}

	plane := emu.PPU.MatrixPlanes[1]
	if plane.SourceMode != ppu.MatrixPlaneSourceBitmap {
		t.Fatalf("plane.SourceMode = %d, want bitmap", plane.SourceMode)
	}
	if plane.BitmapPalette != 3 {
		t.Fatalf("plane.BitmapPalette = %d, want 3", plane.BitmapPalette)
	}
	if !plane.Transparent0 {
		t.Fatal("plane.Transparent0 = false, want true")
	}
	if plane.Bitmap[0] != 0x12 || plane.Bitmap[1] != 0x34 || plane.Bitmap[2] != 0x56 {
		t.Fatalf("plane bitmap first bytes = [%02X %02X %02X], want [12 34 56]", plane.Bitmap[0], plane.Bitmap[1], plane.Bitmap[2])
	}
}

func TestInstallMatrixPlaneProgramRejectsWrongTilemapLength(t *testing.T) {
	emu := NewEmulator()
	err := emu.InstallMatrixPlaneProgram(MatrixPlaneProgram{
		Channel: 0,
		Enabled: true,
		Size:    ppu.TilemapSize64x64,
		Tilemap: make([]byte, 64),
	})
	if err == nil {
		t.Fatal("expected wrong tilemap length to fail")
	}
}

func TestMatrixPlaneBuilderBuildsProgram(t *testing.T) {
	builder, err := NewMatrixPlaneBuilder(1, ppu.TilemapSize64x64)
	if err != nil {
		t.Fatalf("NewMatrixPlaneBuilder failed: %v", err)
	}
	if err := builder.SetTile(2, 3, 0x12, 0x34); err != nil {
		t.Fatalf("SetTile failed: %v", err)
	}
	if err := builder.FillRect(4, 5, 2, 2, 0x56, 0x78); err != nil {
		t.Fatalf("FillRect failed: %v", err)
	}
	if err := builder.SetPatternTile8x8(3, make([]byte, 32)); err != nil {
		t.Fatalf("SetPatternTile8x8 failed: %v", err)
	}
	pattern := make([]byte, 32)
	pattern[0] = 0xAB
	if err := builder.SetPatternTile8x8(4, pattern); err != nil {
		t.Fatalf("SetPatternTile8x8 second failed: %v", err)
	}

	program := builder.Build()
	if program.Channel != 1 || !program.Enabled || program.Size != ppu.TilemapSize64x64 {
		t.Fatalf("unexpected built program header: %+v", program)
	}

	entry := (3*64 + 2) * 2
	if program.Tilemap[entry] != 0x12 || program.Tilemap[entry+1] != 0x34 {
		t.Fatalf("built tilemap entry = [%02X %02X], want [12 34]", program.Tilemap[entry], program.Tilemap[entry+1])
	}
	rectEntry := (5*64 + 4) * 2
	if program.Tilemap[rectEntry] != 0x56 || program.Tilemap[rectEntry+1] != 0x78 {
		t.Fatalf("built rect tile entry = [%02X %02X], want [56 78]", program.Tilemap[rectEntry], program.Tilemap[rectEntry+1])
	}
	if len(program.Pattern) != 129 {
		t.Fatalf("built pattern length = %d, want 129", len(program.Pattern))
	}
	if program.Pattern[4*32] != 0xAB {
		t.Fatalf("built pattern byte = 0x%02X, want 0xAB", program.Pattern[4*32])
	}
}

func TestMatrixPlaneBuilderBuildsBitmapProgram(t *testing.T) {
	builder, err := NewMatrixPlaneBuilder(0, ppu.TilemapSize32x32)
	if err != nil {
		t.Fatalf("NewMatrixPlaneBuilder failed: %v", err)
	}
	builder.SetBitmapMode(2)
	builder.SetBitmapTransparency(true)
	if err := builder.SetBitmapPixel(0, 0, 0x0A); err != nil {
		t.Fatalf("SetBitmapPixel failed: %v", err)
	}
	if err := builder.SetBitmapPixel(1, 0, 0x05); err != nil {
		t.Fatalf("SetBitmapPixel second failed: %v", err)
	}

	program := builder.Build()
	if program.SourceMode != ppu.MatrixPlaneSourceBitmap {
		t.Fatalf("program.SourceMode = %d, want bitmap", program.SourceMode)
	}
	if program.BitmapPalette != 2 {
		t.Fatalf("program.BitmapPalette = %d, want 2", program.BitmapPalette)
	}
	if !program.Transparent0 {
		t.Fatal("program.Transparent0 = false, want true")
	}
	if len(program.Bitmap) == 0 || program.Bitmap[0] != 0xA5 {
		t.Fatalf("program.Bitmap[0] = 0x%02X, want 0xA5", program.Bitmap[0])
	}
}
