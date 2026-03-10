package ppu

import (
	"testing"

	"nitro-core-dx/internal/debug"
)

func TestApplyScanlineCommandLayoutProgramsControls(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.ApplyScanlineCommandLayout(ScanlineCommandLayout{
		Enabled:            true,
		LayerMask:          0x05, // BG0 + BG2
		IncludeRebind:      true,
		IncludePriority:    true,
		IncludeTilemapBase: true,
		IncludeSourceMode:  true,
	})

	if !ppu.HDMAEnabled {
		t.Fatal("HDMAEnabled should be true after applying layout")
	}
	if got, want := ppu.HDMAControl, uint8(0xEB); got != want {
		t.Fatalf("HDMAControl = 0x%02X, want 0x%02X", got, want)
	}
	if got, want := ppu.HDMAExtControl, uint8(0x01); got != want {
		t.Fatalf("HDMAExtControl = 0x%02X, want 0x%02X", got, want)
	}
}

func TestWriteScanlineCommandProgramRoundTripsThroughDecoder(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.HDMATableBase = 0x3000
	ppu.ApplyScanlineCommandLayout(ScanlineCommandLayout{
		Enabled:            true,
		LayerMask:          0x03, // BG0 + BG1
		IncludeRebind:      true,
		IncludePriority:    true,
		IncludeTilemapBase: true,
		IncludeSourceMode:  true,
	})

	program := ScanlineCommandProgram{
		Layers: [4]ScanlineLayerProgram{
			{
				ScrollX:        0x0123,
				ScrollY:        0x0456,
				TransformA:     0x0111,
				TransformB:     0x0222,
				TransformC:     0x0333,
				TransformD:     0x0444,
				CenterX:        0x0555,
				CenterY:        0x0666,
				HasRebind:      true,
				Rebind:         2,
				HasPriority:    true,
				Priority:       3,
				HasTilemapBase: true,
				TilemapBase:    0x4A00,
				HasSourceMode:  true,
				SourceMode:     1,
			},
			{
				ScrollX:        -12,
				ScrollY:        34,
				TransformA:     0x0100,
				TransformB:     0,
				TransformC:     0,
				TransformD:     0x0100,
				CenterX:        10,
				CenterY:        20,
				HasPriority:    true,
				Priority:       1,
				HasTilemapBase: true,
				TilemapBase:    0x5200,
			},
		},
	}

	if err := ppu.WriteScanlineCommandProgram(3, program); err != nil {
		t.Fatalf("WriteScanlineCommandProgram returned error: %v", err)
	}

	commands := ppu.decodeScanlineCommands(3)
	bg0 := commands.Layers[0]
	if !bg0.LayerEnabled {
		t.Fatal("BG0 should be enabled by layer mask")
	}
	if bg0.ScrollX != 0x0123 || bg0.ScrollY != 0x0456 {
		t.Fatalf("BG0 scroll decoded as (%d,%d), want (291,1110)", bg0.ScrollX, bg0.ScrollY)
	}
	if !bg0.ApplyRebind || bg0.TransformBinding != 2 {
		t.Fatalf("BG0 rebind decode = (%v,%d), want (true,2)", bg0.ApplyRebind, bg0.TransformBinding)
	}
	if !bg0.ApplyPriority || bg0.Priority != 3 {
		t.Fatalf("BG0 priority decode = (%v,%d), want (true,3)", bg0.ApplyPriority, bg0.Priority)
	}
	if !bg0.ApplyTilemapBase || bg0.TilemapBase != 0x4A00 {
		t.Fatalf("BG0 tilemap decode = (%v,0x%04X), want (true,0x4A00)", bg0.ApplyTilemapBase, bg0.TilemapBase)
	}
	if !bg0.ApplySourceMode || bg0.SourceMode != 1 {
		t.Fatalf("BG0 source-mode decode = (%v,%d), want (true,1)", bg0.ApplySourceMode, bg0.SourceMode)
	}

	bg1 := commands.Layers[1]
	if !bg1.LayerEnabled {
		t.Fatal("BG1 should be enabled by layer mask")
	}
	if bg1.ApplyRebind {
		t.Fatal("BG1 rebind should remain sentinel/keep")
	}
	if !bg1.ApplyPriority || bg1.Priority != 1 {
		t.Fatalf("BG1 priority decode = (%v,%d), want (true,1)", bg1.ApplyPriority, bg1.Priority)
	}
	if !bg1.ApplyTilemapBase || bg1.TilemapBase != 0x5200 {
		t.Fatalf("BG1 tilemap decode = (%v,0x%04X), want (true,0x5200)", bg1.ApplyTilemapBase, bg1.TilemapBase)
	}
	if bg1.ApplySourceMode {
		t.Fatal("BG1 source-mode should remain sentinel/keep")
	}

	if commands.Layers[2].LayerEnabled || commands.Layers[3].LayerEnabled {
		t.Fatal("BG2/BG3 should be disabled by layer mask")
	}
}

func TestWriteScanlineCommandProgramDrivesUpdateHDMA(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.HDMATableBase = 0x3400
	ppu.TransformChannels[3].Enabled = true
	ppu.ApplyScanlineCommandLayout(ScanlineCommandLayout{
		Enabled:            true,
		LayerMask:          0x01, // BG0
		IncludeRebind:      true,
		IncludePriority:    true,
		IncludeTilemapBase: true,
		IncludeSourceMode:  true,
	})

	program := ScanlineCommandProgram{
		Layers: [4]ScanlineLayerProgram{
			{
				ScrollX:        22,
				ScrollY:        44,
				TransformA:     0x0100,
				TransformB:     0x0001,
				TransformC:     0x0002,
				TransformD:     0x0100,
				CenterX:        160,
				CenterY:        100,
				HasRebind:      true,
				Rebind:         3,
				HasPriority:    true,
				Priority:       2,
				HasTilemapBase: true,
				TilemapBase:    0x5800,
				HasSourceMode:  true,
				SourceMode:     1,
			},
		},
	}

	if err := ppu.WriteScanlineCommandProgram(0, program); err != nil {
		t.Fatalf("WriteScanlineCommandProgram returned error: %v", err)
	}

	ppu.updateHDMA(0)

	if ppu.BG0.TransformChannel != 3 {
		t.Fatalf("BG0.TransformChannel = %d, want 3", ppu.BG0.TransformChannel)
	}
	if ppu.BG0.Priority != 2 {
		t.Fatalf("BG0.Priority = %d, want 2", ppu.BG0.Priority)
	}
	if ppu.BG0.TilemapBase != 0x5800 {
		t.Fatalf("BG0.TilemapBase = 0x%04X, want 0x5800", ppu.BG0.TilemapBase)
	}
	if ppu.BG0.SourceMode != 1 {
		t.Fatalf("BG0.SourceMode = %d, want 1", ppu.BG0.SourceMode)
	}
	if ppu.BG0.ScrollX != 22 || ppu.BG0.ScrollY != 44 {
		t.Fatalf("BG0 scroll = (%d,%d), want (22,44)", ppu.BG0.ScrollX, ppu.BG0.ScrollY)
	}
	if ch := ppu.TransformChannels[3]; ch.A != 0x0100 || ch.B != 0x0001 || ch.C != 0x0002 || ch.D != 0x0100 || ch.CenterX != 160 || ch.CenterY != 100 {
		t.Fatalf("transform channel 3 state = %+v, want programmed transform payload", ch)
	}
}
