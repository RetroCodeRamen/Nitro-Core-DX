package ppu

import (
	"testing"

	"nitro-core-dx/internal/debug"
)

func TestMatrixRegistersDriveTransformChannelZero(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.Write8(0x18, 0x27) // enable, mirror H/V, outside mode=0, direct color=1
	ppu.Write8(0x19, 0x34)
	ppu.Write8(0x1A, 0x12)
	ppu.Write8(0x1B, 0x78)
	ppu.Write8(0x1C, 0x56)
	ppu.Write8(0x1D, 0xBC)
	ppu.Write8(0x1E, 0x9A)
	ppu.Write8(0x1F, 0xF0)
	ppu.Write8(0x20, 0xDE)
	ppu.Write8(0x27, 0x11)
	ppu.Write8(0x28, 0x22)
	ppu.Write8(0x29, 0x33)
	ppu.Write8(0x2A, 0x44)

	channel := ppu.TransformChannels[0]
	if !channel.Enabled {
		t.Fatal("transform channel 0 should mirror BG0 matrix enable bit")
	}
	if !channel.MirrorH || !channel.MirrorV {
		t.Fatal("transform channel 0 mirror flags should mirror control bits")
	}
	if !channel.DirectColor {
		t.Fatal("transform channel 0 direct color should be set by control path")
	}
	if got := uint16(channel.A); got != 0x1234 {
		t.Fatalf("channel.A = 0x%04X, want 0x1234", got)
	}
	if got := uint16(channel.B); got != 0x5678 {
		t.Fatalf("channel.B = 0x%04X, want 0x5678", got)
	}
	if got := uint16(channel.C); got != 0x9ABC {
		t.Fatalf("channel.C = 0x%04X, want 0x9ABC", got)
	}
	if got := uint16(channel.D); got != 0xDEF0 {
		t.Fatalf("channel.D = 0x%04X, want 0xDEF0", got)
	}
	if got := uint16(channel.CenterX); got != 0x2211 {
		t.Fatalf("channel.CenterX = 0x%04X, want 0x2211", got)
	}
	if got := uint16(channel.CenterY); got != 0x4433 {
		t.Fatalf("channel.CenterY = 0x%04X, want 0x4433", got)
	}
}

func TestDefaultTransformBindingsMirrorCanonicalLayerMatrixState(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	if ppu.BG0.TransformChannel != 0 || ppu.BG1.TransformChannel != 1 || ppu.BG2.TransformChannel != 2 || ppu.BG3.TransformChannel != 3 {
		t.Fatalf("default transform bindings = [%d %d %d %d], want [0 1 2 3]",
			ppu.BG0.TransformChannel, ppu.BG1.TransformChannel, ppu.BG2.TransformChannel, ppu.BG3.TransformChannel)
	}

	channel := ppu.getLayerBoundTransformChannel(2)
	channel.Enabled = true
	channel.A = 0x0123
	channel.B = -0x0020
	channel.C = 0x0007
	channel.D = 0x00F0
	channel.CenterX = 42
	channel.CenterY = 84
	channel.MirrorH = true
	channel.MirrorV = true
	channel.OutsideMode = 2
	channel.DirectColor = true

	_, resolved := ppu.resolveLayerTransformChannel(2)
	if resolved == nil {
		t.Fatal("expected BG2 to resolve a bound transform channel")
	}
	if *resolved != *channel {
		t.Fatal("default binding should resolve BG2 directly to transform channel 2")
	}
}

func TestNonDefaultTransformBindingDrivesMatrixRendering(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.BG0.Enabled = true
	ppu.BG0.TransformChannel = 1
	ppu.BG0.TilemapBase = 0x4000
	ppu.BG0.ScrollX = 0
	ppu.BG0.ScrollY = 0
	ppu.TransformChannels[1] = TransformChannel{
		Enabled:     true,
		A:           0x0100,
		D:           0x0100,
		CenterX:     0,
		CenterY:     0,
		OutsideMode: 0,
		DirectColor: true,
	}

	// Tilemap entry 0 -> tile 0 using palette 1.
	ppu.VRAM[0x4000] = 0x00
	ppu.VRAM[0x4001] = 0x01
	// Tile 0 first byte contains two visible palette index 1 pixels.
	ppu.VRAM[0x0000] = 0x11
	ppu.renderDotMatrixMode(0, 0, 0)

	if got := ppu.OutputBuffer[0]; got == 0 {
		t.Fatal("expected non-default bound transform channel to drive BG0 matrix rendering")
	}
}

func TestHDMAUpdatesEnabledBoundTransformChannel(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.HDMAEnabled = true
	ppu.HDMAControl = 0x02 // Enable BG0 HDMA
	ppu.HDMATableBase = 0x2000
	ppu.BG0.TransformChannel = 0
	ppu.TransformChannels[0].Enabled = true

	// Scroll X/Y
	ppu.VRAM[0x2000] = 0x34
	ppu.VRAM[0x2001] = 0x12
	ppu.VRAM[0x2002] = 0x78
	ppu.VRAM[0x2003] = 0x56
	// Matrix A/B/C/D
	ppu.VRAM[0x2004] = 0x11
	ppu.VRAM[0x2005] = 0x01
	ppu.VRAM[0x2006] = 0x22
	ppu.VRAM[0x2007] = 0x02
	ppu.VRAM[0x2008] = 0x33
	ppu.VRAM[0x2009] = 0x03
	ppu.VRAM[0x200A] = 0x44
	ppu.VRAM[0x200B] = 0x04
	// Center X/Y
	ppu.VRAM[0x200C] = 0x55
	ppu.VRAM[0x200D] = 0x05
	ppu.VRAM[0x200E] = 0x66
	ppu.VRAM[0x200F] = 0x06

	ppu.updateHDMA(0)

	if ppu.TransformChannels[0].A != 0x0111 || ppu.TransformChannels[0].B != 0x0222 ||
		ppu.TransformChannels[0].C != 0x0333 || ppu.TransformChannels[0].D != 0x0444 {
		t.Fatal("HDMA matrix update should target the bound runtime transform channel")
	}
	if ppu.TransformChannels[0].CenterX != 0x0555 || ppu.TransformChannels[0].CenterY != 0x0666 {
		t.Fatal("HDMA center update should target the bound runtime transform channel")
	}
}

func TestHDMARebindAppliesBeforeTransformPayload(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.HDMAEnabled = true
	ppu.HDMAControl = 0x22 // BG0 enabled + scanline rebind bytes present
	ppu.HDMATableBase = 0x2400
	ppu.BG0.TransformChannel = 0
	ppu.TransformChannels[0].Enabled = true
	ppu.TransformChannels[0].A = 0x0001
	ppu.TransformChannels[1].Enabled = true

	// Base 16-byte BG0 payload.
	ppu.VRAM[0x2400] = 0x10
	ppu.VRAM[0x2401] = 0x00
	ppu.VRAM[0x2402] = 0x20
	ppu.VRAM[0x2403] = 0x00
	ppu.VRAM[0x2404] = 0x34
	ppu.VRAM[0x2405] = 0x12
	ppu.VRAM[0x2406] = 0x78
	ppu.VRAM[0x2407] = 0x56
	ppu.VRAM[0x2408] = 0xBC
	ppu.VRAM[0x2409] = 0x9A
	ppu.VRAM[0x240A] = 0xF0
	ppu.VRAM[0x240B] = 0xDE
	ppu.VRAM[0x240C] = 0x44
	ppu.VRAM[0x240D] = 0x22
	ppu.VRAM[0x240E] = 0x88
	ppu.VRAM[0x240F] = 0x66
	// Rebind bytes start after the 64-byte payload block.
	ppu.VRAM[0x2440] = 0x01 // BG0 -> channel 1
	ppu.VRAM[0x2441] = 0xFF
	ppu.VRAM[0x2442] = 0xFF
	ppu.VRAM[0x2443] = 0xFF

	ppu.updateHDMA(0)

	if ppu.BG0.TransformChannel != 1 {
		t.Fatalf("BG0.TransformChannel = %d, want 1 after scanline rebind", ppu.BG0.TransformChannel)
	}
	if uint16(ppu.TransformChannels[1].A) != 0x1234 || uint16(ppu.TransformChannels[1].B) != 0x5678 ||
		uint16(ppu.TransformChannels[1].C) != 0x9ABC || uint16(ppu.TransformChannels[1].D) != 0xDEF0 {
		t.Fatal("transform payload should apply to the newly rebound channel")
	}
	if ppu.TransformChannels[0].A != 0x0001 {
		t.Fatal("old channel should remain untouched after scanline rebind")
	}
}

func TestHDMAPriorityUpdateOverridesDefaultLayerOrdering(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.HDMAEnabled = true
	ppu.HDMAControl = 0x47 // BG0/BG1 enabled + per-scanline priority bytes present
	ppu.HDMATableBase = 0x2600

	if ppu.BG0.Priority != 0 || ppu.BG1.Priority != 1 {
		t.Fatal("test expects default priorities before HDMA override")
	}

	// Priority bytes begin after the 64-byte base block when no rebind bytes are present.
	ppu.VRAM[0x2640] = 0x03 // BG0 -> priority 3
	ppu.VRAM[0x2641] = 0x00 // BG1 -> priority 0
	ppu.VRAM[0x2642] = 0xFF
	ppu.VRAM[0x2643] = 0xFF

	ppu.updateHDMA(0)

	if ppu.BG0.Priority != 3 {
		t.Fatalf("BG0.Priority = %d, want 3 after HDMA priority update", ppu.BG0.Priority)
	}
	if ppu.BG1.Priority != 0 {
		t.Fatalf("BG1.Priority = %d, want 0 after HDMA priority update", ppu.BG1.Priority)
	}
}

func TestHDMATilemapBaseUpdateOverridesLayerSourceBase(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.HDMAEnabled = true
	ppu.HDMAControl = 0x82 // BG0 enabled + per-scanline tilemap-base table present
	ppu.HDMATableBase = 0x2800
	ppu.BG0.TilemapBase = 0x4000

	// Tilemap base table starts after the 64-byte base payload block when no
	// rebind or priority tables are present.
	ppu.VRAM[0x2840] = 0x00
	ppu.VRAM[0x2841] = 0x50 // BG0 -> 0x5000
	ppu.VRAM[0x2842] = 0xFF
	ppu.VRAM[0x2843] = 0xFF
	ppu.VRAM[0x2844] = 0xFF
	ppu.VRAM[0x2845] = 0xFF
	ppu.VRAM[0x2846] = 0xFF
	ppu.VRAM[0x2847] = 0xFF

	ppu.updateHDMA(0)

	if ppu.BG0.TilemapBase != 0x5000 {
		t.Fatalf("BG0.TilemapBase = 0x%04X, want 0x5000 after HDMA tilemap-base update", ppu.BG0.TilemapBase)
	}
}

func TestLayerControlRegisterRoundTrip(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.Write8(0x08, 0x0F) // BG0: enabled, large tiles, priority 3
	ppu.Write8(0x09, 0x04) // BG1: disabled, small tiles, priority 1
	ppu.Write8(0x21, 0x07) // BG2: enabled, large tiles, priority 1
	ppu.Write8(0x26, 0x08) // BG3: disabled, small tiles, priority 2

	if !ppu.BG0.Enabled || !ppu.BG0.TileSize || ppu.BG0.Priority != 3 {
		t.Fatal("BG0 control decode did not update layer state correctly")
	}
	if ppu.BG1.Enabled || ppu.BG1.TileSize || ppu.BG1.Priority != 1 {
		t.Fatal("BG1 control decode did not update layer state correctly")
	}
	if !ppu.BG2.Enabled || !ppu.BG2.TileSize || ppu.BG2.Priority != 1 {
		t.Fatal("BG2 control decode did not update layer state correctly")
	}
	if ppu.BG3.Enabled || ppu.BG3.TileSize || ppu.BG3.Priority != 2 {
		t.Fatal("BG3 control decode did not update layer state correctly")
	}

	if got := ppu.Read8(0x08); got != 0x0F {
		t.Fatalf("BG0 control readback = 0x%02X, want 0x0F", got)
	}
	if got := ppu.Read8(0x09); got != 0x04 {
		t.Fatalf("BG1 control readback = 0x%02X, want 0x04", got)
	}
	if got := ppu.Read8(0x21); got != 0x07 {
		t.Fatalf("BG2 control readback = 0x%02X, want 0x07", got)
	}
	if got := ppu.Read8(0x26); got != 0x08 {
		t.Fatalf("BG3 control readback = 0x%02X, want 0x08", got)
	}
}

func TestLayerSourceModeRegisterRoundTrip(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.Write8(0x68, 0x01)
	ppu.Write8(0x69, 0x00)
	ppu.Write8(0x6A, 0x03) // masked to bitmap mode
	ppu.Write8(0x6B, 0x02) // masked to tilemap mode

	if ppu.BG0.SourceMode != 1 {
		t.Fatalf("BG0.SourceMode = %d, want 1", ppu.BG0.SourceMode)
	}
	if ppu.BG1.SourceMode != 0 {
		t.Fatalf("BG1.SourceMode = %d, want 0", ppu.BG1.SourceMode)
	}
	if ppu.BG2.SourceMode != 1 {
		t.Fatalf("BG2.SourceMode = %d, want 1", ppu.BG2.SourceMode)
	}
	if ppu.BG3.SourceMode != 0 {
		t.Fatalf("BG3.SourceMode = %d, want 0", ppu.BG3.SourceMode)
	}

	if got := ppu.Read8(0x68); got != 0x01 {
		t.Fatalf("BG0 source-mode readback = 0x%02X, want 0x01", got)
	}
	if got := ppu.Read8(0x69); got != 0x00 {
		t.Fatalf("BG1 source-mode readback = 0x%02X, want 0x00", got)
	}
	if got := ppu.Read8(0x6A); got != 0x01 {
		t.Fatalf("BG2 source-mode readback = 0x%02X, want 0x01", got)
	}
	if got := ppu.Read8(0x6B); got != 0x00 {
		t.Fatalf("BG3 source-mode readback = 0x%02X, want 0x00", got)
	}
}

func TestLayerTransformBindRegisterRoundTrip(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.Write8(0x6C, 0x03)
	ppu.Write8(0x6D, 0x02)
	ppu.Write8(0x6E, 0x05) // masked to channel 1
	ppu.Write8(0x6F, 0x00)

	if ppu.BG0.TransformChannel != 3 {
		t.Fatalf("BG0.TransformChannel = %d, want 3", ppu.BG0.TransformChannel)
	}
	if ppu.BG1.TransformChannel != 2 {
		t.Fatalf("BG1.TransformChannel = %d, want 2", ppu.BG1.TransformChannel)
	}
	if ppu.BG2.TransformChannel != 1 {
		t.Fatalf("BG2.TransformChannel = %d, want 1", ppu.BG2.TransformChannel)
	}
	if ppu.BG3.TransformChannel != 0 {
		t.Fatalf("BG3.TransformChannel = %d, want 0", ppu.BG3.TransformChannel)
	}

	if got := ppu.Read8(0x6C); got != 0x03 {
		t.Fatalf("BG0 transform-bind readback = 0x%02X, want 0x03", got)
	}
	if got := ppu.Read8(0x6D); got != 0x02 {
		t.Fatalf("BG1 transform-bind readback = 0x%02X, want 0x02", got)
	}
	if got := ppu.Read8(0x6E); got != 0x01 {
		t.Fatalf("BG2 transform-bind readback = 0x%02X, want 0x01", got)
	}
	if got := ppu.Read8(0x6F); got != 0x00 {
		t.Fatalf("BG3 transform-bind readback = 0x%02X, want 0x00", got)
	}
}

func TestLayerTilemapBaseRegisterRoundTrip(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.Write8(0x77, 0x34)
	ppu.Write8(0x78, 0x12)
	ppu.Write8(0x79, 0x78)
	ppu.Write8(0x7A, 0x56)
	ppu.Write8(0x7B, 0xBC)
	ppu.Write8(0x7C, 0x9A)
	ppu.Write8(0x7D, 0xF0)
	ppu.Write8(0x7E, 0xDE)

	if ppu.BG0.TilemapBase != 0x1234 {
		t.Fatalf("BG0.TilemapBase = 0x%04X, want 0x1234", ppu.BG0.TilemapBase)
	}
	if ppu.BG1.TilemapBase != 0x5678 {
		t.Fatalf("BG1.TilemapBase = 0x%04X, want 0x5678", ppu.BG1.TilemapBase)
	}
	if ppu.BG2.TilemapBase != 0x9ABC {
		t.Fatalf("BG2.TilemapBase = 0x%04X, want 0x9ABC", ppu.BG2.TilemapBase)
	}
	if ppu.BG3.TilemapBase != 0xDEF0 {
		t.Fatalf("BG3.TilemapBase = 0x%04X, want 0xDEF0", ppu.BG3.TilemapBase)
	}

	if got := ppu.Read8(0x77); got != 0x34 {
		t.Fatalf("BG0 tilemap low readback = 0x%02X, want 0x34", got)
	}
	if got := ppu.Read8(0x78); got != 0x12 {
		t.Fatalf("BG0 tilemap high readback = 0x%02X, want 0x12", got)
	}
	if got := ppu.Read8(0x79); got != 0x78 {
		t.Fatalf("BG1 tilemap low readback = 0x%02X, want 0x78", got)
	}
	if got := ppu.Read8(0x7A); got != 0x56 {
		t.Fatalf("BG1 tilemap high readback = 0x%02X, want 0x56", got)
	}
	if got := ppu.Read8(0x7B); got != 0xBC {
		t.Fatalf("BG2 tilemap low readback = 0x%02X, want 0xBC", got)
	}
	if got := ppu.Read8(0x7C); got != 0x9A {
		t.Fatalf("BG2 tilemap high readback = 0x%02X, want 0x9A", got)
	}
	if got := ppu.Read8(0x7D); got != 0xF0 {
		t.Fatalf("BG3 tilemap low readback = 0x%02X, want 0xF0", got)
	}
	if got := ppu.Read8(0x7E); got != 0xDE {
		t.Fatalf("BG3 tilemap high readback = 0x%02X, want 0xDE", got)
	}
}

func TestHDMAExtensionControlRegisterRoundTrip(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.Write8(0x7F, 0x01)
	if ppu.HDMAExtControl != 0x01 {
		t.Fatalf("HDMAExtControl = 0x%02X, want 0x01", ppu.HDMAExtControl)
	}
	if got := ppu.Read8(0x7F); got != 0x01 {
		t.Fatalf("HDMA extension control readback = 0x%02X, want 0x01", got)
	}
}

func TestHDMASourceModeUpdateOverridesLayerSourceMode(t *testing.T) {
	logger := debug.NewLogger(1000)
	ppu := NewPPU(logger)

	ppu.HDMAEnabled = true
	ppu.HDMAControl = 0x03 // HDMA enable + BG0 scanline updates enabled
	ppu.HDMAExtControl = 0x01
	ppu.HDMATableBase = 0x2A00
	ppu.BG0.SourceMode = 0

	// Source-mode table starts after the 64-byte base payload block when no
	// rebind, priority, or tilemap-base tables are present.
	ppu.VRAM[0x2A40] = 0x01 // BG0 -> bitmap mode
	ppu.VRAM[0x2A41] = 0xFF
	ppu.VRAM[0x2A42] = 0xFF
	ppu.VRAM[0x2A43] = 0xFF

	ppu.updateHDMA(0)

	if ppu.BG0.SourceMode != 1 {
		t.Fatalf("BG0.SourceMode = %d, want 1 after HDMA source-mode update", ppu.BG0.SourceMode)
	}
}
