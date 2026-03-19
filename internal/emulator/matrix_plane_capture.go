package emulator

import (
	"fmt"

	"nitro-core-dx/internal/debug"
	ppucore "nitro-core-dx/internal/ppu"
)

type MatrixPlaneRenderPhase struct {
	Name        string
	Description string
	A           int16
	B           int16
	C           int16
	D           int16
	ScrollX     int16
	ScrollY     int16
	CenterX     int16
	CenterY     int16
	OutsideMode uint8
}

func DefaultBitmapMatrixPlaneValidationPhases() []MatrixPlaneRenderPhase {
	return []MatrixPlaneRenderPhase{
		{
			Name:        "identity_topleft",
			Description: "Identity transform, top-left anchored view of the 1024x1024 source",
			A:           0x0100, D: 0x0100,
			CenterX: 0, CenterY: 0,
			ScrollX: 0, ScrollY: 0,
			OutsideMode: 0,
		},
		{
			Name:        "rotate_22_5_wrap",
			Description: "22.5 degree wrap rotation around the source center",
			A:           236, B: -98, C: 98, D: 236,
			CenterX: 160, CenterY: 100,
			ScrollX: 512, ScrollY: 512,
			OutsideMode: 0,
		},
		{
			Name:        "rotate_45_wrap",
			Description: "45 degree wrap rotation around the source center",
			A:           181, B: -181, C: 181, D: 181,
			CenterX: 160, CenterY: 100,
			ScrollX: 512, ScrollY: 512,
			OutsideMode: 0,
		},
		{
			Name:        "rotate_45_clamp",
			Description: "45 degree clamp rotation near the source edge",
			A:           181, B: -181, C: 181, D: 181,
			CenterX: 160, CenterY: 100,
			ScrollX: 128, ScrollY: 128,
			OutsideMode: 3,
		},
		{
			Name:        "skew_pan",
			Description: "Skewed affine sample with panning offset",
			A:           0x0100, B: 0x0040, C: -0x0020, D: 0x0100,
			CenterX: 160, CenterY: 100,
			ScrollX: 420, ScrollY: 300,
			OutsideMode: 0,
		},
	}
}

func RenderBitmapMatrixPlanePhase(asset *MatrixPlaneBitmapAsset, phase MatrixPlaneRenderPhase) ([]uint32, error) {
	if asset == nil {
		return nil, fmt.Errorf("matrix plane bitmap asset is required")
	}
	logger := debug.NewLogger(1000)
	ppu := ppucore.NewPPU(logger)

	ppu.BG0.Enabled = true
	ppu.BG0.Priority = 0
	ppu.BG0.TileSize = false
	ppu.BG0.TilemapSize = ppucore.TilemapSize128x128
	ppu.BG0.TransformChannel = asset.Program.Channel
	ppu.BG0.ScrollX = phase.ScrollX
	ppu.BG0.ScrollY = phase.ScrollY

	chIdx := int(asset.Program.Channel) % ppucore.NumTransformChannels
	channel := &ppu.TransformChannels[chIdx]
	channel.Enabled = true
	channel.A = phase.A
	channel.B = phase.B
	channel.C = phase.C
	channel.D = phase.D
	channel.CenterX = phase.CenterX
	channel.CenterY = phase.CenterY
	channel.OutsideMode = phase.OutsideMode

	for i, c := range asset.Palette {
		writeMatrixPlanePalette(ppu, asset.Program.BitmapPalette, uint8(i), c)
	}
	if err := ProgramMatrixPlaneThroughMMIO(ppu, asset.Program); err != nil {
		return nil, err
	}
	if err := ppu.StepPPU(127820); err != nil {
		return nil, err
	}

	out := make([]uint32, len(ppu.OutputBuffer))
	copy(out, ppu.OutputBuffer[:])
	return out, nil
}

func writeMatrixPlanePalette(ppu *ppucore.PPU, bank, index uint8, rgb555 uint16) {
	ppu.Write8(0x12, bank*16+index)
	ppu.Write8(0x13, uint8(rgb555&0xFF))
	ppu.Write8(0x13, uint8(rgb555>>8))
}
