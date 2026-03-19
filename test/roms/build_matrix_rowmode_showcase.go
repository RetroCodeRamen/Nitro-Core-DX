//go:build testrom_tools
// +build testrom_tools

package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"

	"nitro-core-dx/internal/emulator"
	ppucore "nitro-core-dx/internal/ppu"
	"nitro-core-dx/test/roms/romutil"
)

type asm struct{ *romutil.Asm }
type romDataRef = romutil.DataRef

func newASM(bank uint8) *asm { return &asm{Asm: romutil.NewASM(bank)} }

func (a *asm) pc() uint16                  { return a.PC() }
func (a *asm) mark(name string)            { a.Mark(name) }
func (a *asm) inst(w uint16)               { a.Inst(w) }
func (a *asm) imm(v uint16)                { a.Imm(v) }
func (a *asm) uniq(prefix string) string   { return a.Uniq(prefix) }
func (a *asm) movImm(reg uint8, v uint16)  { a.MovImm(reg, v) }
func (a *asm) movReg(dst, src uint8)       { a.MovReg(dst, src) }
func (a *asm) movLoad(dst, addrReg uint8)  { a.MovLoad(dst, addrReg) }
func (a *asm) movLoad8(dst, addrReg uint8) { a.MovLoad8(dst, addrReg) }
func (a *asm) movStore(addrReg, src uint8) { a.MovStore(addrReg, src) }
func (a *asm) setDBR(src uint8)            { a.SetDBR(src) }
func (a *asm) addImm(reg uint8, v uint16)  { a.AddImm(reg, v) }
func (a *asm) addReg(dst, src uint8)       { a.AddReg(dst, src) }
func (a *asm) subImm(reg uint8, v uint16)  { a.SubImm(reg, v) }
func (a *asm) subReg(dst, src uint8)       { a.SubReg(dst, src) }
func (a *asm) andImm(reg uint8, v uint16)  { a.AndImm(reg, v) }
func (a *asm) cmpImm(reg uint8, v uint16)  { a.CmpImm(reg, v) }
func (a *asm) cmpReg(r1, r2 uint8)         { a.CmpReg(r1, r2) }
func (a *asm) shrImm(reg uint8, v uint16)  { a.ShrImm(reg, v) }
func (a *asm) beq(label string)            { a.Beq(label) }
func (a *asm) bne(label string)            { a.Bne(label) }
func (a *asm) jmp(label string)            { a.Jmp(label) }
func (a *asm) call(label string)           { a.Call(label) }
func (a *asm) ret()                        { a.Ret() }
func (a *asm) resolve() error              { return a.Resolve() }

func write8(a *asm, addr uint16, value uint8) { romutil.Write8(a.Asm, addr, value) }
func write8Scratch(a *asm, addr uint16, value uint8, addrReg, valueReg uint8) {
	romutil.Write8Scratch(a.Asm, addr, value, addrReg, valueReg)
}
func write8Reg(a *asm, addr uint16, reg uint8)       { romutil.Write8Reg(a.Asm, addr, reg) }
func write16(a *asm, addr uint16, value uint16)      { romutil.Write16(a.Asm, addr, value) }
func write16s(a *asm, addr uint16, value int16)      { romutil.Write16S(a.Asm, addr, value) }
func write16Reg(a *asm, addr uint16, reg uint8)      { romutil.Write16Reg(a.Asm, addr, reg) }
func write16RegBytes(a *asm, addr uint16, reg uint8) { romutil.Write16RegBytes(a.Asm, addr, reg) }
func emitText(a *asm, x uint16, y uint8, r, g, b uint8, text string) {
	romutil.EmitText(a.Asm, x, y, r, g, b, text)
}
func setCGRAMColor(a *asm, colorIndex uint8, rgb555 uint16) {
	romutil.SetCGRAMColor(a.Asm, colorIndex, rgb555)
}
func emitWaitOneFrame(a *asm, wramLastFrame uint16) { romutil.EmitWaitOneFrame(a.Asm, wramLastFrame) }
func emitUploadRoutine(a *asm, label string)        { romutil.EmitUploadRoutine(a.Asm, label) }
func emitUploadChunks(a *asm, routine string, targetPort uint16, ref romDataRef) {
	romutil.EmitUploadChunks(a.Asm, routine, targetPort, ref)
}
func emitWaitForDMAIdle(a *asm) { romutil.EmitWaitForDMAIdle(a.Asm) }
func emitMatrixBitmapDMAChunks(a *asm, channel uint8, ref romDataRef) {
	romutil.EmitMatrixBitmapDMAChunks(a.Asm, channel, ref)
}
func emitMatrixRowDMAChunks(a *asm, channel uint8, ref romDataRef) {
	romutil.EmitMatrixRowDMAChunks(a.Asm, channel, ref)
}
func emitVRAMDMAChunks(a *asm, destAddr uint16, ref romDataRef) {
	romutil.EmitVRAMDMAChunks(a.Asm, destAddr, ref)
}
func emitInitTrigTable(a *asm, tableBase uint16, steps int) {
	romutil.EmitInitTrigTable(a.Asm, tableBase, steps)
}
func emitLoadTrigPair(a *asm, tableBase uint16, indexReg, cosReg, sinReg uint8) {
	romutil.EmitLoadTrigPair(a.Asm, tableBase, indexReg, cosReg, sinReg)
}
func emitWriteMatrixRegs(a *asm, controlAddr, aAddr, bAddr, cAddr, dAddr, cxAddr, cyAddr uint16, controlValue uint8, aReg, bReg, cReg, dReg uint8, centerX, centerY int16) {
	romutil.EmitWriteMatrixRegs(a.Asm, controlAddr, aAddr, bAddr, cAddr, dAddr, cxAddr, cyAddr, controlValue, aReg, bReg, cReg, dReg, centerX, centerY)
}

var (
	allocateROMData = romutil.AllocateROMData
	appendDataBlob  = romutil.AppendDataBlob
	padPayloadToRef = romutil.PadPayloadToRef
)

func put16LE(buf []byte, value int16) {
	u := uint16(value)
	buf[0] = byte(u)
	buf[1] = byte(u >> 8)
}

func put32LE(buf []byte, value int32) {
	u := uint32(value)
	buf[0] = byte(u)
	buf[1] = byte(u >> 8)
	buf[2] = byte(u >> 16)
	buf[3] = byte(u >> 24)
}

type trackPoint struct {
	x float64
	y float64
}

func matchesTrackLineColor(r, g, b float64) bool {
	const (
		targetR = 255.0
		targetG = 127.0
		targetB = 127.0
		tol     = 30.0
	)
	return math.Abs(r-targetR) <= tol && math.Abs(g-targetG) <= tol && math.Abs(b-targetB) <= tol
}

func matchesFallbackGrayTrack(r, g, b float64) bool {
	hi := math.Max(r, math.Max(g, b))
	lo := math.Min(r, math.Min(g, b))
	lum := (r + g + b) / 3.0
	return hi-lo < 26.0 && lum >= 65.0 && lum <= 210.0
}

func extractTrackPath(img image.Image, phaseCount int) []trackPoint {
	if img == nil || phaseCount <= 0 {
		return nil
	}
	resized := scaleNearest(img, 1024, 1024)
	bounds := resized.Bounds()
	cx := float64(bounds.Dx()) / 2.0
	cy := float64(bounds.Dy()) / 2.0
	maxRadius := int(math.Min(cx, cy)) - 12
	if maxRadius <= 96 {
		return nil
	}
	path := make([]trackPoint, phaseCount)
	valid := make([]bool, phaseCount)
	for i := 0; i < phaseCount; i++ {
		phase := (2.0 * math.Pi * float64(i)) / float64(phaseCount)
		cosv := math.Cos(phase)
		sinv := math.Sin(phase)
		lineRadii := make([]float64, 0, 16)
		grayRadii := make([]float64, 0, 64)
		for r := 96; r <= maxRadius; r++ {
			x := int(math.Round(cx + cosv*float64(r)))
			y := int(math.Round(cy + sinv*float64(r)))
			if x < 0 || y < 0 || x >= bounds.Dx() || y >= bounds.Dy() {
				continue
			}
			cr, cg, cb, _ := resized.At(x, y).RGBA()
			rr := float64(uint8(cr >> 8))
			gg := float64(uint8(cg >> 8))
			bb := float64(uint8(cb >> 8))
			if matchesTrackLineColor(rr, gg, bb) {
				lineRadii = append(lineRadii, float64(r))
			} else if matchesFallbackGrayTrack(rr, gg, bb) {
				grayRadii = append(grayRadii, float64(r))
			}
		}
		var radius float64
		if len(lineRadii) > 0 {
			radius = lineRadii[len(lineRadii)/2]
		} else if len(grayRadii) > 0 {
			radius = grayRadii[len(grayRadii)/2]
		} else {
			continue
		}
		path[i] = trackPoint{
			x: cx + cosv*radius,
			y: cy + sinv*radius,
		}
		valid[i] = true
	}

	last := -1
	for i := 0; i < phaseCount; i++ {
		if valid[i] {
			last = i
			break
		}
	}
	if last < 0 {
		return nil
	}
	for i := 0; i < phaseCount; i++ {
		if valid[i] {
			last = i
			continue
		}
		next := -1
		for j := 1; j < phaseCount; j++ {
			idx := (i + j) % phaseCount
			if valid[idx] {
				next = idx
				break
			}
		}
		if next < 0 {
			path[i] = path[last]
			continue
		}
		prev := last
		if prev < 0 {
			path[i] = path[next]
			continue
		}
		distNext := (next - i + phaseCount) % phaseCount
		distPrev := (i - prev + phaseCount) % phaseCount
		total := float64(distPrev + distNext)
		t := float64(distPrev) / total
		path[i] = trackPoint{
			x: path[prev].x + (path[next].x-path[prev].x)*t,
			y: path[prev].y + (path[next].y-path[prev].y)*t,
		}
	}

	smoothed := make([]trackPoint, phaseCount)
	for i := 0; i < phaseCount; i++ {
		var sx, sy float64
		var n float64
		for k := -2; k <= 2; k++ {
			idx := (i + k + phaseCount) % phaseCount
			weight := 1.0
			if k == 0 {
				weight = 2.0
			}
			sx += path[idx].x * weight
			sy += path[idx].y * weight
			n += weight
		}
		smoothed[i] = trackPoint{x: sx / n, y: sy / n}
	}
	return smoothed
}

func buildPerspectiveFloorTable(phase float64, path []trackPoint, pathIndex int) []byte {
	const (
		horizonY = 92
		stride   = 64
		screenCX = 160.0
		sourceCX = 512.0
		sourceCY = 512.0
		orbitRX  = 340.0
		orbitRY  = 300.0
	)
	table := make([]byte, ppucore.VisibleScanlines*stride)
	cameraX := sourceCX + orbitRX*math.Cos(phase)
	cameraY := sourceCY + orbitRY*math.Sin(phase)
	tangentX := -math.Sin(phase)
	tangentY := math.Cos(phase)
	if len(path) > 0 {
		curr := path[pathIndex%len(path)]
		prev := path[(pathIndex-1+len(path))%len(path)]
		next := path[(pathIndex+1)%len(path)]
		cameraX = curr.x
		cameraY = curr.y
		tangentX = next.x - prev.x
		tangentY = next.y - prev.y
	}
	inwardX := sourceCX - cameraX
	inwardY := sourceCY - cameraY
	inwardLen := math.Hypot(inwardX, inwardY)
	if inwardLen != 0 {
		inwardX /= inwardLen
		inwardY /= inwardLen
	}
	forwardX := tangentX*0.92 + inwardX*0.28
	forwardY := tangentY*0.92 + inwardY*0.28
	forwardLen := math.Hypot(forwardX, forwardY)
	if forwardLen != 0 {
		forwardX /= forwardLen
		forwardY /= forwardLen
	}
	rightX := forwardY
	rightY := -forwardX
	for y := 0; y < ppucore.VisibleScanlines; y++ {
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

			du := rightX * step
			dv := rightY * step
			rowCenterX := cameraX + forwardX*forward
			rowCenterY := cameraY + forwardY*forward
			rowStartX := rowCenterX - screenCX*du
			rowStartY := rowCenterY - screenCX*dv

			aCoeff = int16(math.Round(du * 256.0))
			cCoeff = int16(math.Round(dv * 256.0))
			scrollX = int16(math.Round(rowStartX))
			scrollY = int16(math.Round(rowStartY))
		}
		put16LE(table[base+0:base+2], scrollX)
		put16LE(table[base+2:base+4], scrollY)
		put16LE(table[base+4:base+6], aCoeff)
		put16LE(table[base+6:base+8], bCoeff)
		put16LE(table[base+8:base+10], cCoeff)
		put16LE(table[base+10:base+12], dCoeff)
		put16LE(table[base+12:base+14], centerX)
		put16LE(table[base+14:base+16], centerY)
	}
	return table
}

func buildPerspectiveFloorTables(phaseCount int, path []trackPoint) [][]byte {
	tables := make([][]byte, phaseCount)
	for i := 0; i < phaseCount; i++ {
		phase := (2.0 * math.Pi * float64(i)) / float64(phaseCount)
		tables[i] = buildPerspectiveFloorTable(phase, path, i)
	}
	return tables
}

func buildPerspectiveRowTable(phaseCount int, path []trackPoint, idx int) []byte {
	const (
		horizonY = 92
		screenCX = 160.0
		stride   = 16
		sourceCX = 512.0
		sourceCY = 512.0
		orbitRX  = 340.0
		orbitRY  = 300.0
	)
	table := make([]byte, ppucore.VisibleScanlines*stride)
	phase := (2.0 * math.Pi * float64(idx)) / float64(phaseCount)
	cameraX := sourceCX + orbitRX*math.Cos(phase)
	cameraY := sourceCY + orbitRY*math.Sin(phase)
	tangentX := -math.Sin(phase)
	tangentY := math.Cos(phase)
	if len(path) > 0 {
		curr := path[idx%len(path)]
		prev := path[(idx-1+len(path))%len(path)]
		next := path[(idx+1)%len(path)]
		cameraX = curr.x
		cameraY = curr.y
		tangentX = next.x - prev.x
		tangentY = next.y - prev.y
	}

	inwardX := sourceCX - cameraX
	inwardY := sourceCY - cameraY
	inwardLen := math.Hypot(inwardX, inwardY)
	if inwardLen != 0 {
		inwardX /= inwardLen
		inwardY /= inwardLen
	}
	forwardX := tangentX*0.92 + inwardX*0.28
	forwardY := tangentY*0.92 + inwardY*0.28
	forwardLen := math.Hypot(forwardX, forwardY)
	if forwardLen != 0 {
		forwardX /= forwardLen
		forwardY /= forwardLen
	}
	rightX := forwardY
	rightY := -forwardX

	for y := 0; y < ppucore.VisibleScanlines; y++ {
		base := y * stride
		startX := int32(-1 << 16)
		startY := int32(0)
		stepX := int32(0)
		stepY := int32(0)

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

			du := rightX * step
			dv := rightY * step
			rowCenterX := cameraX + forwardX*forward
			rowCenterY := cameraY + forwardY*forward
			rowStartX := rowCenterX - screenCX*du
			rowStartY := rowCenterY - screenCX*dv

			startX = int32(math.Round(rowStartX * 65536.0))
			startY = int32(math.Round(rowStartY * 65536.0))
			stepX = int32(math.Round(du * 65536.0))
			stepY = int32(math.Round(dv * 65536.0))
		}

		put32LE(table[base+0:base+4], startX)
		put32LE(table[base+4:base+8], startY)
		put32LE(table[base+8:base+12], stepX)
		put32LE(table[base+12:base+16], stepY)
	}
	return table
}

func buildPerspectiveRowTables(phaseCount int, path []trackPoint) [][]byte {
	tables := make([][]byte, phaseCount)
	for i := 0; i < phaseCount; i++ {
		tables[i] = buildPerspectiveRowTable(phaseCount, path, i)
	}
	return tables
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func scaleNearest(src image.Image, width, height int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	sb := src.Bounds()
	sw := sb.Dx()
	sh := sb.Dy()
	for y := 0; y < height; y++ {
		sy := sb.Min.Y + (y*sh)/height
		for x := 0; x < width; x++ {
			sx := sb.Min.X + (x*sw)/width
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func makeTransparentCanvas(src image.Image, canvasPixels, imagePixels int) image.Image {
	dst := image.NewNRGBA(image.Rect(0, 0, canvasPixels, canvasPixels))
	draw.Draw(dst, dst.Bounds(), image.Transparent, image.Point{}, draw.Src)
	scaled := scaleNearest(src, imagePixels, imagePixels)
	off := image.Pt((canvasPixels-imagePixels)/2, (canvasPixels-imagePixels)/2)
	draw.Draw(dst, image.Rect(off.X, off.Y, off.X+imagePixels, off.Y+imagePixels), scaled, image.Point{}, draw.Over)
	return dst
}

func makeSkyOverlayCanvas(canvasPixels int) image.Image {
	dst := image.NewNRGBA(image.Rect(0, 0, canvasPixels, canvasPixels))
	draw.Draw(dst, dst.Bounds(), image.Transparent, image.Point{}, draw.Src)

	// The visible identity window for a centered 320x200 viewport against a 1024x1024
	// source runs through the middle of the bitmap. Fill its upper half with sky and
	// a bright horizon so the floor only reads in the lower half of the screen.
	top := 412
	horizon := 512
	for y := top; y < horizon; y++ {
		for x := 0; x < canvasPixels; x++ {
			dst.SetNRGBA(x, y, color.NRGBA{R: 0x58, G: 0x88, B: 0xD8, A: 0xFF})
		}
	}
	for y := horizon - 4; y < horizon+2; y++ {
		for x := 0; x < canvasPixels; x++ {
			dst.SetNRGBA(x, y, color.NRGBA{R: 0xF0, G: 0xF0, B: 0xA0, A: 0xFF})
		}
	}
	return dst
}

func buildMatrixRowModeShowcaseROM(img image.Image, outPath string) error {
	const (
		codeBank          = 1
		dataStartBank     = 2
		wramLastFrame     = 0x0200
		wramTrigTableBase = 0x0300
		trigSteps         = 32
		floorPhaseCount   = 64
		floorPhaseShift   = 4
		matrixPlane0Ctl   = 0x1D // enabled, 128x128, bitmap, palette bank 1
		matrixPlane1Ctl   = 0x2D // enabled, 128x128, bitmap, palette bank 2
		matrixPlane2Ctl   = 0x3B // enabled, 64x64, bitmap, palette bank 3
		matrixPlane0Flags = 0x00 // opaque floor
		matrixPlane1Flags = 0x01 // palette index 0 transparent
		matrixPlane2Flags = 0x01 // palette index 0 transparent
	)

	floorAsset, err := emulator.BuildBitmapMatrixPlaneAssetFromImage(img, 0, ppucore.TilemapSize128x128, 1)
	if err != nil {
		return err
	}
	skyOverlay := makeSkyOverlayCanvas(1024)
	sideCanvas := makeTransparentCanvas(img, 512, 192)
	skyAsset, err := emulator.BuildBitmapMatrixPlaneAssetFromImage(skyOverlay, 1, ppucore.TilemapSize128x128, 2)
	if err != nil {
		return err
	}
	sideAsset2, err := emulator.BuildBitmapMatrixPlaneAssetFromImage(sideCanvas, 2, ppucore.TilemapSize64x64, 3)
	if err != nil {
		return err
	}
	floorPath := extractTrackPath(img, floorPhaseCount)
	rowTables := buildPerspectiveRowTables(floorPhaseCount, floorPath)

	a := newASM(codeBank)
	uploadRoutine := "upload_bytes"

	// Init palette banks.
	for i, c := range floorAsset.Palette {
		setCGRAMColor(a, uint8(1*16+i), c)
	}
	for i, c := range skyAsset.Palette {
		setCGRAMColor(a, uint8(2*16+i), c)
	}
	for i, c := range sideAsset2.Palette {
		setCGRAMColor(a, uint8(3*16+i), c)
	}
	write8(a, 0x8012, 0) // backdrop palette index
	write8(a, 0x8013, 0)
	write8(a, 0x8013, 0)

	// Configure BGs and transform bindings.
	write8(a, 0x8008, 0x21) // BG0 enabled, prio 0, 128x128
	write8(a, 0x8009, 0x25) // BG1 enabled, prio 1, 128x128
	write8(a, 0x8021, 0x19) // BG2 enabled, prio 2, 64x64
	write8(a, 0x806C, 0x00)
	write8(a, 0x806D, 0x01)
	write8(a, 0x806E, 0x02)

	// Upload bitmap plane 0.
	write8(a, 0x8080, 0x00)
	write8(a, 0x8081, matrixPlane0Ctl)
	write8(a, 0x808C, matrixPlane0Flags)
	write8(a, 0x8088, 0x00)
	write8(a, 0x8089, 0x00)
	write8(a, 0x808A, 0x00)

	// Plane 1.
	write8(a, 0x8080, 0x01)
	write8(a, 0x8081, matrixPlane1Ctl)
	write8(a, 0x808C, matrixPlane1Flags)
	write8(a, 0x8088, 0x00)
	write8(a, 0x8089, 0x00)
	write8(a, 0x808A, 0x00)

	// Plane 2.
	write8(a, 0x8080, 0x02)
	write8(a, 0x8081, matrixPlane2Ctl)
	write8(a, 0x808C, matrixPlane2Flags)
	write8(a, 0x8088, 0x00)
	write8(a, 0x8089, 0x00)
	write8(a, 0x808A, 0x00)

	// Initialize trig table for animation.
	emitInitTrigTable(a, wramTrigTableBase, trigSteps)

	// Reset frame baseline.
	write8(a, 0x8010, 0x00) // FRAME_COUNTER reset not implemented; just latch current low byte
	a.movImm(4, 0x803F)
	a.movLoad(2, 4)
	a.movImm(4, wramLastFrame)
	a.movStore(4, 2)

	// Turn on display after upload and init complete.
	write8(a, 0x802A, 0x00) // harmless BG0 center Y low write ensures no hidden stale? (left as no-op-ish register state)
	write8(a, 0x800C, 0x00) // BACKDROP_COLOR fallback
	write8(a, 0x801B, 0x00) // BG0 matrix B low reset
	write8(a, 0x8018, 0x00) // clear before final animation loop writes
	write8(a, 0x8011, 0x01) // DISPLAY_CONTROL: enable display

	a.mark("main_loop")
	emitWaitOneFrame(a, wramLastFrame)

	// Floor angle: slow rotation.
	a.movImm(4, 0x803F)
	a.movLoad(0, 4)
	a.shrImm(0, 3)
	a.andImm(0, trigSteps-1)
	emitLoadTrigPair(a, wramTrigTableBase, 0, 2, 3)
	a.movImm(6, 0)
	a.subReg(6, 3)          // -sin
	write16(a, 0x0000, 0)   // no-op scratch to keep codegen stable? removed later if needed
	write16(a, 0x8000, 512) // BG0 scroll X
	write16(a, 0x8002, 512) // BG0 scroll Y
	emitWriteMatrixRegs(a, 0x8018, 0x8019, 0x801B, 0x801D, 0x801F, 0x8027, 0x8029, 0x01, 2, 6, 3, 2, 160, 132)

	// Full-screen sky/horizon overlay: identity transform, clamp.
	write16(a, 0x8004, 512) // BG1 scroll X
	write16(a, 0x8006, 512) // BG1 scroll Y
	a.movImm(2, 0x0100)
	a.movImm(3, 0x0000)
	a.movImm(6, 0x0000)
	emitWriteMatrixRegs(a, 0x802B, 0x802C, 0x802E, 0x8030, 0x8032, 0x8034, 0x8036, 0x19, 2, 3, 6, 2, 160, 100)

	// Right side plane: counter-rotation, clamp.
	a.movImm(4, 0x803F)
	a.movLoad(0, 4)
	a.shrImm(0, 2)
	a.andImm(0, trigSteps-1)
	a.movImm(5, trigSteps)
	a.subReg(5, 0)
	a.andImm(5, trigSteps-1)
	emitLoadTrigPair(a, wramTrigTableBase, 5, 2, 3)
	a.movImm(6, 0)
	a.subReg(6, 3)
	write16(a, 0x800A, 256) // BG2 scroll X
	write16(a, 0x800C, 256) // BG2 scroll Y
	emitWriteMatrixRegs(a, 0x8038, 0x8039, 0x803B, 0x803D, 0x803F, 0x8041, 0x8043, 0x19, 2, 3, 6, 2, 232, 74)

	a.jmp("main_loop")

	emitUploadRoutine(a, uploadRoutine)

	if err := a.resolve(); err != nil {
		return err
	}

	// Build data region and inject upload calls at the top after code is finalized.
	floorRef, cursor := allocateROMData(0, floorAsset.Program.Bitmap)
	skyRef, cursor := allocateROMData(cursor, skyAsset.Program.Bitmap)
	side2Ref, cursor := allocateROMData(cursor, sideAsset2.Program.Bitmap)
	rowTableRefs := make([]romDataRef, floorPhaseCount)
	for i := range rowTables {
		rowTableRefs[i], cursor = allocateROMData(cursor, rowTables[i])
	}
	_ = cursor

	// Rebuild with upload calls before init table/display.
	// Inserted just after plane control writes by simply appending now would be too late,
	// so use a second banked builder pass would be cleaner. Keep this sprint pragmatic:
	// patch upload calls into the existing code path by rebuilding from scratch.
	// Since the builder is small, restart with a fresh assembler and data refs.
	a = newASM(codeBank)
	// Palette setup.
	for i, c := range floorAsset.Palette {
		setCGRAMColor(a, uint8(1*16+i), c)
	}
	for i, c := range skyAsset.Palette {
		setCGRAMColor(a, uint8(2*16+i), c)
	}
	for i, c := range sideAsset2.Palette {
		setCGRAMColor(a, uint8(3*16+i), c)
	}
	setCGRAMColor(a, 0, 0)

	// BG config.
	write8(a, 0x8008, 0x21)
	write8(a, 0x8009, 0x25)
	write8(a, 0x8021, 0x19)
	write8(a, 0x806C, 0x00)
	write8(a, 0x806D, 0x01)
	write8(a, 0x806E, 0x02)

	// Upload plane 0 bitmap.
	write8(a, 0x8080, 0x00)
	write8(a, 0x8081, matrixPlane0Ctl)
	write8(a, 0x808C, matrixPlane0Flags)
	write8(a, 0x8088, 0x00)
	write8(a, 0x8089, 0x00)
	write8(a, 0x808A, 0x00)
	emitMatrixBitmapDMAChunks(a, 0x00, floorRef)

	// Upload plane 1 bitmap (sky/horizon overlay).
	write8(a, 0x8080, 0x01)
	write8(a, 0x8081, matrixPlane1Ctl)
	write8(a, 0x808C, matrixPlane1Flags)
	write8(a, 0x8088, 0x00)
	write8(a, 0x8089, 0x00)
	write8(a, 0x808A, 0x00)
	emitMatrixBitmapDMAChunks(a, 0x01, skyRef)

	// Upload plane 2 bitmap.
	write8(a, 0x8080, 0x02)
	write8(a, 0x8081, matrixPlane2Ctl)
	write8(a, 0x808C, matrixPlane2Flags)
	write8(a, 0x8088, 0x00)
	write8(a, 0x8089, 0x00)
	write8(a, 0x808A, 0x00)
	emitMatrixBitmapDMAChunks(a, 0x02, side2Ref)

	emitMatrixRowDMAChunks(a, 0x00, rowTableRefs[0])

	emitInitTrigTable(a, wramTrigTableBase, trigSteps)

	a.movImm(4, 0x803F)
	a.movLoad(2, 4)
	a.movImm(4, wramLastFrame)
	a.movStore(4, 2)

	a.mark("main_loop")
	emitWaitOneFrame(a, wramLastFrame)

	// Floor plane: upload one fine-grained row-parameter table every frame.
	a.movImm(4, 0x803F)
	a.movLoad(0, 4)
	a.shrImm(0, floorPhaseShift)
	a.andImm(0, floorPhaseCount-1)
	labelAfterFloorTable := a.uniq("after_row_table")
	phaseLabels := make([]string, floorPhaseCount)
	for i := 0; i < floorPhaseCount; i++ {
		phaseLabels[i] = a.uniq(fmt.Sprintf("row_table_%02d", i))
	}
	for i := 0; i < floorPhaseCount; i++ {
		a.cmpImm(0, uint16(i))
		a.beq(phaseLabels[i])
	}
	emitMatrixRowDMAChunks(a, 0x00, rowTableRefs[floorPhaseCount-1])
	a.jmp(labelAfterFloorTable)
	for i := 0; i < floorPhaseCount; i++ {
		a.mark(phaseLabels[i])
		emitMatrixRowDMAChunks(a, 0x00, rowTableRefs[i])
		if i != floorPhaseCount-1 {
			a.jmp(labelAfterFloorTable)
		}
	}
	a.mark(labelAfterFloorTable)
	write8(a, 0x8080, 0x00)
	write8(a, 0x808D, 0x01)
	write8(a, 0x8018, 0x01) // BG0 matrix mode enabled, wrap outside behavior

	// Full-screen sky/horizon overlay: identity transform, clamp.
	write16(a, 0x8004, 512)
	write16(a, 0x8006, 512)
	a.movImm(2, 0x0100)
	a.movImm(3, 0x0000)
	a.movImm(6, 0x0000)
	emitWriteMatrixRegs(a, 0x802B, 0x802C, 0x802E, 0x8030, 0x8032, 0x8034, 0x8036, 0x19, 2, 3, 6, 2, 160, 100)

	// Right-side comparison plane: fixed inset, no rotation.
	write16(a, 0x800A, 256)
	write16(a, 0x800C, 256)
	a.movImm(2, 0x0100)
	a.movImm(3, 0x0000)
	a.movImm(6, 0x0000)
	emitWriteMatrixRegs(a, 0x8038, 0x8039, 0x803B, 0x803D, 0x803F, 0x8041, 0x8043, 0x19, 2, 3, 6, 2, 264, 76)

	emitText(a, 8, 8, 0xF8, 0xF8, 0xF8, "LOWER HALF FLOOR DEMO")
	emitText(a, 8, 20, 0xB0, 0xE0, 0xFF, "BG0 ROW FLOOR  BG1 SKY  BG2 SIDE")

	a.jmp("main_loop")

	emitUploadRoutine(a, uploadRoutine)

	if err := a.resolve(); err != nil {
		return err
	}

	payload := append([]byte{}, floorAsset.Program.Bitmap...)
	payload = append(payload, skyAsset.Program.Bitmap...)
	payload = append(payload, sideAsset2.Program.Bitmap...)
	for i := range rowTables {
		payload = append(payload, rowTables[i]...)
	}
	if err := appendDataBlob(a.B, dataStartBank, payload); err != nil {
		return err
	}

	return a.B.BuildROM(codeBank, 0x8000, outPath)
}

func main() {
	inPath := flag.String("in", "Resources/kart.png", "input PNG image")
	outPath := flag.String("out", "roms/matrix_rowmode_showcase.rom", "output ROM path")
	flag.Parse()

	img, err := loadPNG(*inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load %s: %v\n", *inPath, err)
		os.Exit(1)
	}
	if err := buildMatrixRowModeShowcaseROM(img, *outPath); err != nil {
		fmt.Fprintf(os.Stderr, "build ROM: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Built %s using %s\n", *outPath, *inPath)
}
