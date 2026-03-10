package emulator

import (
	"fmt"
	"sort"

	"nitro-core-dx/internal/ppu"
)

// RasterProgramLayout describes the authored scanline-command layout at the
// emulator boundary without exposing the raw PPU package types.
type RasterProgramLayout struct {
	Enabled            bool
	LayerMask          uint8
	IncludeRebind      bool
	IncludePriority    bool
	IncludeTilemapBase bool
	IncludeSourceMode  bool
}

// RasterLayerProgram is the high-level authored scanline state for one layer.
type RasterLayerProgram struct {
	ScrollX, ScrollY int16
	TransformA       int16
	TransformB       int16
	TransformC       int16
	TransformD       int16
	CenterX          int16
	CenterY          int16

	HasRebind      bool
	Rebind         uint8
	HasPriority    bool
	Priority       uint8
	HasTilemapBase bool
	TilemapBase    uint16
	HasSourceMode  bool
	SourceMode     uint8
}

// RasterScanlineProgram contains the authored state for one scanline.
type RasterScanlineProgram struct {
	Scanline int
	Layers   [4]RasterLayerProgram
}

// RasterProgram is a higher-level authored raster program installable on the
// emulator without requiring direct VRAM packing by callers.
type RasterProgram struct {
	TableBase uint16
	Layout    RasterProgramLayout
	Scanlines []RasterScanlineProgram
}

// RasterProgramBuilder builds a raster program over a visible-frame scanline set.
type RasterProgramBuilder struct {
	tableBase uint16
	layout    RasterProgramLayout
	scanlines map[int]RasterScanlineProgram
}

func NewRasterProgramBuilder(tableBase uint16, layout RasterProgramLayout) *RasterProgramBuilder {
	return &RasterProgramBuilder{
		tableBase: tableBase,
		layout:    layout,
		scanlines: make(map[int]RasterScanlineProgram, ppu.VisibleScanlines),
	}
}

func (b *RasterProgramBuilder) SetScanline(scanline int, program RasterScanlineProgram) error {
	if scanline < 0 || scanline >= ppu.VisibleScanlines {
		return fmt.Errorf("scanline %d out of range", scanline)
	}
	program.Scanline = scanline
	b.scanlines[scanline] = program
	return nil
}

func (b *RasterProgramBuilder) FillScanlineRange(start, end int, program RasterScanlineProgram) error {
	if start < 0 || end < start || end >= ppu.VisibleScanlines {
		return fmt.Errorf("scanline range %d-%d out of range", start, end)
	}
	for scanline := start; scanline <= end; scanline++ {
		next := program
		next.Scanline = scanline
		b.scanlines[scanline] = next
	}
	return nil
}

func (b *RasterProgramBuilder) Build() RasterProgram {
	scanlines := make([]RasterScanlineProgram, 0, len(b.scanlines))
	for _, program := range b.scanlines {
		scanlines = append(scanlines, program)
	}
	sort.Slice(scanlines, func(i, j int) bool {
		return scanlines[i].Scanline < scanlines[j].Scanline
	})
	return RasterProgram{
		TableBase: b.tableBase,
		Layout:    b.layout,
		Scanlines: scanlines,
	}
}

// InstallRasterProgram applies an authored raster program to the live PPU.
func (e *Emulator) InstallRasterProgram(program RasterProgram) error {
	if e == nil || e.PPU == nil {
		return fmt.Errorf("emulator PPU unavailable")
	}

	stride := rasterProgramStride(program.Layout)
	totalBytes := uint32(stride) * uint32(ppu.VisibleScanlines)
	base := uint32(program.TableBase)
	if base+totalBytes > uint32(len(e.PPU.VRAM)) {
		return fmt.Errorf("raster program exceeds VRAM: base=0x%04X bytes=%d", program.TableBase, totalBytes)
	}

	clear(e.PPU.VRAM[base : base+totalBytes])
	e.PPU.HDMATableBase = program.TableBase
	e.PPU.ApplyScanlineCommandLayout(ppu.ScanlineCommandLayout{
		Enabled:            program.Layout.Enabled,
		LayerMask:          program.Layout.LayerMask,
		IncludeRebind:      program.Layout.IncludeRebind,
		IncludePriority:    program.Layout.IncludePriority,
		IncludeTilemapBase: program.Layout.IncludeTilemapBase,
		IncludeSourceMode:  program.Layout.IncludeSourceMode,
	})

	for _, authored := range program.Scanlines {
		var scanlineProgram ppu.ScanlineCommandProgram
		for layerNum := 0; layerNum < 4; layerNum++ {
			layer := authored.Layers[layerNum]
			scanlineProgram.Layers[layerNum] = ppu.ScanlineLayerProgram{
				ScrollX:        layer.ScrollX,
				ScrollY:        layer.ScrollY,
				TransformA:     layer.TransformA,
				TransformB:     layer.TransformB,
				TransformC:     layer.TransformC,
				TransformD:     layer.TransformD,
				CenterX:        layer.CenterX,
				CenterY:        layer.CenterY,
				HasRebind:      layer.HasRebind,
				Rebind:         layer.Rebind,
				HasPriority:    layer.HasPriority,
				Priority:       layer.Priority,
				HasTilemapBase: layer.HasTilemapBase,
				TilemapBase:    layer.TilemapBase,
				HasSourceMode:  layer.HasSourceMode,
				SourceMode:     layer.SourceMode,
			}
		}
		if err := e.PPU.WriteScanlineCommandProgram(authored.Scanline, scanlineProgram); err != nil {
			return fmt.Errorf("write scanline %d: %w", authored.Scanline, err)
		}
	}

	return nil
}

func (e *Emulator) ClearRasterProgram() error {
	if e == nil || e.PPU == nil {
		return fmt.Errorf("emulator PPU unavailable")
	}
	e.PPU.HDMAEnabled = false
	e.PPU.HDMAControl = 0
	e.PPU.HDMAExtControl = 0
	return nil
}

func rasterProgramStride(layout RasterProgramLayout) uint32 {
	stride := uint32(64)
	if layout.IncludeRebind {
		stride += 4
	}
	if layout.IncludePriority {
		stride += 4
	}
	if layout.IncludeTilemapBase {
		stride += 8
	}
	if layout.IncludeSourceMode {
		stride += 4
	}
	return stride
}
