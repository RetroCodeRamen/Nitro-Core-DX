package emulator

import "fmt"

// RasterTransformState describes one authored affine state for a raster preset.
type RasterTransformState struct {
	ScrollX, ScrollY int16
	A, B, C, D       int16
	CenterX, CenterY int16
}

func IdentityRasterTransform() RasterTransformState {
	return RasterTransformState{
		A: 0x0100,
		D: 0x0100,
	}
}

// BuildSplitTilemapRasterProgram creates a raster program that switches one
// layer between two tilemap bases at a scanline split.
func BuildSplitTilemapRasterProgram(tableBase uint16, layerNum, splitScanline int, topTilemapBase, bottomTilemapBase uint16) (RasterProgram, error) {
	if layerNum < 0 || layerNum > 3 {
		return RasterProgram{}, fmt.Errorf("layer %d out of range", layerNum)
	}
	if splitScanline < 0 || splitScanline > 200 {
		return RasterProgram{}, fmt.Errorf("split scanline %d out of range", splitScanline)
	}

	builder := NewRasterProgramBuilder(tableBase, RasterProgramLayout{
		Enabled:            true,
		LayerMask:          1 << layerNum,
		IncludeTilemapBase: true,
	})

	identity := IdentityRasterTransform()
	top := RasterScanlineProgram{}
	top.Layers[layerNum] = RasterLayerProgram{
		ScrollX:        identity.ScrollX,
		ScrollY:        identity.ScrollY,
		TransformA:     identity.A,
		TransformB:     identity.B,
		TransformC:     identity.C,
		TransformD:     identity.D,
		CenterX:        identity.CenterX,
		CenterY:        identity.CenterY,
		HasTilemapBase: true,
		TilemapBase:    topTilemapBase,
	}
	bottom := top
	bottom.Layers[layerNum].TilemapBase = bottomTilemapBase

	if splitScanline > 0 {
		if err := builder.FillScanlineRange(0, splitScanline-1, top); err != nil {
			return RasterProgram{}, err
		}
	}
	if splitScanline < 200 {
		if err := builder.FillScanlineRange(splitScanline, 199, bottom); err != nil {
			return RasterProgram{}, err
		}
	}

	return builder.Build(), nil
}

// BuildRebindPriorityRasterProgram creates a raster program that can swap a
// layer's transform-channel binding and priority at a scanline split.
func BuildRebindPriorityRasterProgram(tableBase uint16, layerNum, splitScanline int, topChannel, bottomChannel, topPriority, bottomPriority uint8, topTransform, bottomTransform RasterTransformState) (RasterProgram, error) {
	if layerNum < 0 || layerNum > 3 {
		return RasterProgram{}, fmt.Errorf("layer %d out of range", layerNum)
	}
	if splitScanline < 0 || splitScanline > 200 {
		return RasterProgram{}, fmt.Errorf("split scanline %d out of range", splitScanline)
	}

	builder := NewRasterProgramBuilder(tableBase, RasterProgramLayout{
		Enabled:         true,
		LayerMask:       1 << layerNum,
		IncludeRebind:   true,
		IncludePriority: true,
	})

	top := RasterScanlineProgram{}
	top.Layers[layerNum] = RasterLayerProgram{
		ScrollX:     topTransform.ScrollX,
		ScrollY:     topTransform.ScrollY,
		TransformA:  topTransform.A,
		TransformB:  topTransform.B,
		TransformC:  topTransform.C,
		TransformD:  topTransform.D,
		CenterX:     topTransform.CenterX,
		CenterY:     topTransform.CenterY,
		HasRebind:   true,
		Rebind:      topChannel & 0x03,
		HasPriority: true,
		Priority:    topPriority & 0x03,
	}

	bottom := RasterScanlineProgram{}
	bottom.Layers[layerNum] = RasterLayerProgram{
		ScrollX:     bottomTransform.ScrollX,
		ScrollY:     bottomTransform.ScrollY,
		TransformA:  bottomTransform.A,
		TransformB:  bottomTransform.B,
		TransformC:  bottomTransform.C,
		TransformD:  bottomTransform.D,
		CenterX:     bottomTransform.CenterX,
		CenterY:     bottomTransform.CenterY,
		HasRebind:   true,
		Rebind:      bottomChannel & 0x03,
		HasPriority: true,
		Priority:    bottomPriority & 0x03,
	}

	if splitScanline > 0 {
		if err := builder.FillScanlineRange(0, splitScanline-1, top); err != nil {
			return RasterProgram{}, err
		}
	}
	if splitScanline < 200 {
		if err := builder.FillScanlineRange(splitScanline, 199, bottom); err != nil {
			return RasterProgram{}, err
		}
	}

	return builder.Build(), nil
}

// BuildScrollAffineRasterProgram creates a raster program that changes scroll
// and affine sampling state across a scanline split for a single layer.
func BuildScrollAffineRasterProgram(tableBase uint16, layerNum, splitScanline int, topTransform, bottomTransform RasterTransformState) (RasterProgram, error) {
	if layerNum < 0 || layerNum > 3 {
		return RasterProgram{}, fmt.Errorf("layer %d out of range", layerNum)
	}
	if splitScanline < 0 || splitScanline > 200 {
		return RasterProgram{}, fmt.Errorf("split scanline %d out of range", splitScanline)
	}

	builder := NewRasterProgramBuilder(tableBase, RasterProgramLayout{
		Enabled:   true,
		LayerMask: 1 << layerNum,
	})

	top := RasterScanlineProgram{}
	top.Layers[layerNum] = RasterLayerProgram{
		ScrollX:    topTransform.ScrollX,
		ScrollY:    topTransform.ScrollY,
		TransformA: topTransform.A,
		TransformB: topTransform.B,
		TransformC: topTransform.C,
		TransformD: topTransform.D,
		CenterX:    topTransform.CenterX,
		CenterY:    topTransform.CenterY,
	}

	bottom := RasterScanlineProgram{}
	bottom.Layers[layerNum] = RasterLayerProgram{
		ScrollX:    bottomTransform.ScrollX,
		ScrollY:    bottomTransform.ScrollY,
		TransformA: bottomTransform.A,
		TransformB: bottomTransform.B,
		TransformC: bottomTransform.C,
		TransformD: bottomTransform.D,
		CenterX:    bottomTransform.CenterX,
		CenterY:    bottomTransform.CenterY,
	}

	if splitScanline > 0 {
		if err := builder.FillScanlineRange(0, splitScanline-1, top); err != nil {
			return RasterProgram{}, err
		}
	}
	if splitScanline < 200 {
		if err := builder.FillScanlineRange(splitScanline, 199, bottom); err != nil {
			return RasterProgram{}, err
		}
	}

	return builder.Build(), nil
}
