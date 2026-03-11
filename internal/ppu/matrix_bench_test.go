package ppu

import (
	"encoding/binary"
	"math"
	"testing"

	"nitro-core-dx/internal/debug"
)

func setBenchmarkBitmapPixel(buf []uint8, width, x, y int, color uint8) {
	pixelOffset := y*width + x
	byteOffset := pixelOffset / 2
	if pixelOffset%2 == 0 {
		buf[byteOffset] = (color << 4) | (buf[byteOffset] & 0x0F)
		return
	}
	buf[byteOffset] = (buf[byteOffset] & 0xF0) | (color & 0x0F)
}

func setBenchmarkCGRAM(p *PPU, index uint8, rgb555 uint16) {
	addr := int(index) * 2
	p.CGRAM[addr] = uint8(rgb555 & 0xFF)
	p.CGRAM[addr+1] = uint8(rgb555 >> 8)
}

func buildBenchmarkFloorHDMATable(theta float64) []byte {
	const (
		horizonY = 92
		stride   = 64
		screenCX = 160.0
		sourceCX = 512.0
		sourceCY = 512.0
	)
	table := make([]byte, VisibleScanlines*stride)
	cosTheta := math.Cos(theta)
	sinTheta := math.Sin(theta)
	for y := 0; y < VisibleScanlines; y++ {
		base := y * stride
		scrollX := int16(sourceCX)
		scrollY := int16(sourceCY)
		aCoeff := int16(0x0100)
		bCoeff := int16(0)
		cCoeff := int16(0)
		dCoeff := int16(0)
		centerX := int16(0)
		centerY := int16(0)
		if y >= horizonY {
			line := float64(y-horizonY) + 1.0
			step := 1.6 / (1.0 + line/18.0)
			if step < 0.08 {
				step = 0.08
			}
			if step > 1.6 {
				step = 1.6
			}
			forward := 3072.0 / (line + 6.0)

			rightX := cosTheta
			rightY := -sinTheta
			forwardX := sinTheta
			forwardY := cosTheta

			du := rightX * step
			dv := rightY * step
			rowCenterX := sourceCX + forwardX*forward
			rowCenterY := sourceCY + forwardY*forward
			rowStartX := rowCenterX - screenCX*du
			rowStartY := rowCenterY - screenCX*dv

			aCoeff = int16(math.Round(du * 256.0))
			cCoeff = int16(math.Round(dv * 256.0))
			scrollX = int16(math.Round(rowStartX))
			scrollY = int16(math.Round(rowStartY))
		}
		binary.LittleEndian.PutUint16(table[base+0:base+2], uint16(scrollX))
		binary.LittleEndian.PutUint16(table[base+2:base+4], uint16(scrollY))
		binary.LittleEndian.PutUint16(table[base+4:base+6], uint16(aCoeff))
		binary.LittleEndian.PutUint16(table[base+6:base+8], uint16(bCoeff))
		binary.LittleEndian.PutUint16(table[base+8:base+10], uint16(cCoeff))
		binary.LittleEndian.PutUint16(table[base+10:base+12], uint16(dCoeff))
		binary.LittleEndian.PutUint16(table[base+12:base+14], uint16(centerX))
		binary.LittleEndian.PutUint16(table[base+14:base+16], uint16(centerY))
	}
	return table
}

func newBenchmarkMatrixFloorPPU() *PPU {
	p := NewPPU(debug.NewLogger(1))

	// Floor palette bank 1.
	setBenchmarkCGRAM(p, 0x10, 0x0000)
	setBenchmarkCGRAM(p, 0x11, 0x7FFF)
	setBenchmarkCGRAM(p, 0x12, 0x03E0)
	setBenchmarkCGRAM(p, 0x13, 0x7C00)
	setBenchmarkCGRAM(p, 0x14, 0x001F)
	setBenchmarkCGRAM(p, 0x15, 0x03FF)
	setBenchmarkCGRAM(p, 0x16, 0x7FE0)
	setBenchmarkCGRAM(p, 0x17, 0x7C1F)

	// Sky palette bank 2.
	setBenchmarkCGRAM(p, 0x20, 0x0000)
	setBenchmarkCGRAM(p, 0x21, 0x4D9F)
	setBenchmarkCGRAM(p, 0x22, 0x7FFF)
	setBenchmarkCGRAM(p, 0x23, 0x03FF)

	// Opaque bitmap-backed floor on plane 0.
	p.BG0.Enabled = true
	p.BG0.TilemapSize = TilemapSize128x128
	p.BG0.TransformChannel = 0
	p.TransformChannels[0].Enabled = true
	p.TransformChannels[0].OutsideMode = 0
	p.MatrixPlanes[0].Enabled = true
	p.MatrixPlanes[0].Size = TilemapSize128x128
	p.MatrixPlanes[0].SourceMode = MatrixPlaneSourceBitmap
	p.MatrixPlanes[0].BitmapPalette = 1
	p.MatrixPlanes[0].Transparent0 = false
	for y := 0; y < 1024; y++ {
		for x := 0; x < 1024; x++ {
			colorIndex := uint8((((x / 32) + (y / 32)) % 7) + 1)
			setBenchmarkBitmapPixel(p.MatrixPlanes[0].Bitmap[:], 1024, x, y, colorIndex)
		}
	}

	// Transparent sky/horizon overlay on plane 1.
	p.BG1.Enabled = true
	p.BG1.TilemapSize = TilemapSize128x128
	p.BG1.TransformChannel = 1
	p.TransformChannels[1].Enabled = true
	p.TransformChannels[1].A = 0x0100
	p.TransformChannels[1].D = 0x0100
	p.TransformChannels[1].CenterX = 160
	p.TransformChannels[1].CenterY = 100
	p.TransformChannels[1].OutsideMode = 3
	p.MatrixPlanes[1].Enabled = true
	p.MatrixPlanes[1].Size = TilemapSize128x128
	p.MatrixPlanes[1].SourceMode = MatrixPlaneSourceBitmap
	p.MatrixPlanes[1].BitmapPalette = 2
	p.MatrixPlanes[1].Transparent0 = true
	for y := 412; y < 512; y++ {
		for x := 0; x < 1024; x++ {
			setBenchmarkBitmapPixel(p.MatrixPlanes[1].Bitmap[:], 1024, x, y, 1)
		}
	}
	for y := 508; y < 514; y++ {
		for x := 0; x < 1024; x++ {
			setBenchmarkBitmapPixel(p.MatrixPlanes[1].Bitmap[:], 1024, x, y, 2)
		}
	}

	// Scanline floor table in VRAM, BG0 only.
	floorTable := buildBenchmarkFloorHDMATable(12.0 * math.Pi / 180.0)
	copy(p.VRAM[:], floorTable)
	p.HDMAEnabled = true
	p.HDMATableBase = 0
	p.HDMAControl = 0x03
	p.HDMAExtControl = 0x00

	return p
}

func BenchmarkStepPPUBitmapMatrixFloorFrame(b *testing.B) {
	cyclesPerFrame := uint64(TotalScanlines * DotsPerScanline)
	b.ReportAllocs()
	b.SetBytes(int64(ScreenWidth * ScreenHeight))
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		p := newBenchmarkMatrixFloorPPU()
		b.StartTimer()
		if err := p.StepPPU(cyclesPerFrame); err != nil {
			b.Fatalf("StepPPU failed: %v", err)
		}
	}
}
