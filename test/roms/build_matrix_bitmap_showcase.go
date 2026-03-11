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
	"nitro-core-dx/internal/rom"
)

type patchRef struct {
	wordIndex int
	currentPC uint16
	target    string
}

type asm struct {
	b       *rom.BankedROMBuilder
	bank    uint8
	labels  map[string]uint16
	patches []patchRef
	uniqID  int
}

func newASM(bank uint8) *asm {
	return &asm{
		b:      rom.NewBankedROMBuilder(),
		bank:   bank,
		labels: make(map[string]uint16),
	}
}

func (a *asm) pc() uint16                { return a.b.PC(a.bank) }
func (a *asm) mark(name string)          { a.labels[name] = a.pc() }
func (a *asm) inst(w uint16)             { a.b.AddInstruction(a.bank, w) }
func (a *asm) imm(v uint16)              { a.b.AddImmediate(a.bank, v) }
func (a *asm) uniq(prefix string) string { a.uniqID++; return fmt.Sprintf("%s_%d", prefix, a.uniqID) }

func (a *asm) movImm(reg uint8, v uint16)  { a.inst(rom.EncodeMOV(1, reg, 0)); a.imm(v) }
func (a *asm) movReg(dst, src uint8)       { a.inst(rom.EncodeMOV(0, dst, src)) }
func (a *asm) movLoad(dst, addrReg uint8)  { a.inst(rom.EncodeMOV(2, dst, addrReg)) }
func (a *asm) movLoad8(dst, addrReg uint8) { a.inst(rom.EncodeMOV(6, dst, addrReg)) }
func (a *asm) movStore(addrReg, src uint8) { a.inst(rom.EncodeMOV(3, addrReg, src)) }
func (a *asm) setDBR(src uint8)            { a.inst(rom.EncodeMOV(8, src, 0)) }
func (a *asm) addImm(reg uint8, v uint16)  { a.inst(rom.EncodeADD(1, reg, 0)); a.imm(v) }
func (a *asm) addReg(dst, src uint8)       { a.inst(rom.EncodeADD(0, dst, src)) }
func (a *asm) subImm(reg uint8, v uint16)  { a.inst(rom.EncodeSUB(1, reg, 0)); a.imm(v) }
func (a *asm) subReg(dst, src uint8)       { a.inst(rom.EncodeSUB(0, dst, src)) }
func (a *asm) andImm(reg uint8, v uint16)  { a.inst(rom.EncodeAND(1, reg, 0)); a.imm(v) }
func (a *asm) cmpImm(reg uint8, v uint16)  { a.inst(rom.EncodeCMP(7, reg, 0)); a.imm(v) }
func (a *asm) cmpReg(r1, r2 uint8)         { a.inst(rom.EncodeCMP(0, r1, r2)) }
func (a *asm) shrImm(reg uint8, v uint16)  { a.inst(rom.EncodeSHR(1, reg, 0)); a.imm(v) }

func (a *asm) branch(op uint16, label string) {
	a.inst(op)
	pc := a.pc()
	a.imm(0)
	a.patches = append(a.patches, patchRef{
		wordIndex: a.b.GetCodeLength(a.bank) - 1,
		currentPC: pc,
		target:    label,
	})
}

func (a *asm) beq(label string)  { a.branch(rom.EncodeBEQ(), label) }
func (a *asm) bne(label string)  { a.branch(rom.EncodeBNE(), label) }
func (a *asm) jmp(label string)  { a.branch(rom.EncodeJMP(), label) }
func (a *asm) call(label string) { a.branch(rom.EncodeCALL(), label) }
func (a *asm) ret()              { a.inst(rom.EncodeRET()) }

func (a *asm) resolve() error {
	for _, p := range a.patches {
		targetPC, ok := a.labels[p.target]
		if !ok {
			return fmt.Errorf("unknown label %q", p.target)
		}
		a.b.SetImmediateAt(a.bank, p.wordIndex, uint16(rom.CalculateBranchOffset(p.currentPC, targetPC)))
	}
	return nil
}

func write8(a *asm, addr uint16, value uint8) {
	a.movImm(4, addr)
	a.movImm(5, uint16(value))
	a.movStore(4, 5)
}

func write8Reg(a *asm, addr uint16, reg uint8) {
	a.movImm(4, addr)
	a.movStore(4, reg)
}

func write16(a *asm, addr uint16, value uint16) {
	write8(a, addr, uint8(value&0xFF))
	write8(a, addr+1, uint8(value>>8))
}

func write16s(a *asm, addr uint16, value int16) {
	write16(a, addr, uint16(value))
}

func write16Reg(a *asm, addr uint16, reg uint8) {
	write8Reg(a, addr, reg)
	a.movReg(7, reg)
	a.shrImm(7, 8)
	write8Reg(a, addr+1, 7)
}

func emitText(a *asm, x uint16, y uint8, r, g, b uint8, text string) {
	write16(a, 0x8070, x)
	write8(a, 0x8072, y)
	write8(a, 0x8073, r)
	write8(a, 0x8074, g)
	write8(a, 0x8075, b)
	for i := 0; i < len(text); i++ {
		write8(a, 0x8076, text[i])
	}
}

func setCGRAMColor(a *asm, colorIndex uint8, rgb555 uint16) {
	write8(a, 0x8012, colorIndex)
	write8(a, 0x8013, uint8(rgb555&0xFF))
	write8(a, 0x8013, uint8(rgb555>>8))
}

func emitWaitOneFrame(a *asm, wramLastFrame uint16) {
	waitNotVBlank := a.uniq("wait_not_vblank")
	waitFrameEdge := a.uniq("wait_frame_edge")
	waitVBlank := a.uniq("wait_vblank")

	a.mark(waitNotVBlank)
	a.movImm(4, 0x803E)
	a.movLoad(2, 4)
	a.cmpImm(2, 0)
	a.bne(waitNotVBlank)

	a.mark(waitFrameEdge)
	a.movImm(4, 0x803F)
	a.movLoad(2, 4)
	a.movImm(4, wramLastFrame)
	a.movLoad(3, 4)
	a.cmpReg(2, 3)
	a.beq(waitFrameEdge)
	a.movImm(4, wramLastFrame)
	a.movStore(4, 2)

	a.mark(waitVBlank)
	a.movImm(4, 0x803E)
	a.movLoad(2, 4)
	a.cmpImm(2, 0)
	a.beq(waitVBlank)
}

type romDataRef struct {
	bank   uint8
	offset uint16
	length int
}

type uploadChunk struct {
	bank   uint8
	offset uint16
	count  uint16
}

func allocateROMData(cursor int, payload []byte) (romDataRef, int) {
	return romDataRef{
		bank:   uint8(rom.ROMMinProgramBank + 1 + cursor/rom.ROMBankSizeBytes),
		offset: uint16(rom.ROMBankOffsetBase + (cursor % rom.ROMBankSizeBytes)),
		length: len(payload),
	}, cursor + len(payload)
}

func splitROMData(ref romDataRef) []uploadChunk {
	remaining := ref.length
	bank := ref.bank
	offset := ref.offset
	chunks := make([]uploadChunk, 0, (remaining/rom.ROMBankSizeBytes)+1)
	for remaining > 0 {
		avail := 0x10000 - int(offset)
		if avail > remaining {
			avail = remaining
		}
		chunks = append(chunks, uploadChunk{
			bank:   bank,
			offset: offset,
			count:  uint16(avail),
		})
		remaining -= avail
		bank++
		offset = rom.ROMBankOffsetBase
	}
	return chunks
}

func appendDataBlob(b *rom.BankedROMBuilder, startBank uint8, payload []byte) error {
	for i := 0; i < len(payload); i += 2 {
		bank := uint8(int(startBank) + (i / rom.ROMBankSizeBytes))
		if bank > rom.ROMMaxProgramBank {
			return fmt.Errorf("data exceeds ROM bank budget at bank %d", bank)
		}
		word := uint16(payload[i])
		if i+1 < len(payload) {
			word |= uint16(payload[i+1]) << 8
		}
		b.AddInstruction(bank, word)
	}
	return nil
}

func emitUploadRoutine(a *asm, label string) {
	a.mark(label)
	loop := a.uniq("upload_loop")
	done := a.uniq("upload_done")
	a.setDBR(0)
	a.mark(loop)
	a.cmpImm(2, 0)
	a.beq(done)
	a.movLoad8(3, 1)
	a.movStore(4, 3)
	a.addImm(1, 1)
	a.subImm(2, 1)
	a.jmp(loop)
	a.mark(done)
	a.ret()
}

func emitUploadChunks(a *asm, routine string, targetPort uint16, ref romDataRef) {
	for _, chunk := range splitROMData(ref) {
		a.movImm(0, uint16(chunk.bank))
		a.movImm(1, chunk.offset)
		a.movImm(2, chunk.count)
		a.movImm(4, targetPort)
		a.call(routine)
	}
}

func emitWaitForDMAIdle(a *asm) {
	loop := a.uniq("wait_dma_idle")
	a.mark(loop)
	a.movImm(4, 0x8060)
	a.movLoad(2, 4)
	a.cmpImm(2, 0)
	a.bne(loop)
}

func emitMatrixBitmapDMAChunks(a *asm, channel uint8, ref romDataRef) {
	var destOffset uint32
	for _, chunk := range splitROMData(ref) {
		write8(a, 0x8080, channel)
		write8(a, 0x8088, uint8(destOffset&0xFF))
		write8(a, 0x8089, uint8((destOffset>>8)&0xFF))
		write8(a, 0x808A, uint8((destOffset>>16)&0x07))
		write8(a, 0x8061, chunk.bank)
		write16(a, 0x8062, chunk.offset)
		write16(a, 0x8064, 0x0000)
		write16(a, 0x8066, chunk.count)
		write8(a, 0x8060, 0x15) // enable | copy | matrix bitmap destination
		emitWaitForDMAIdle(a)
		destOffset += uint32(chunk.count)
	}
}

func emitVRAMDMAChunks(a *asm, destAddr uint16, ref romDataRef) {
	var offset uint16
	for _, chunk := range splitROMData(ref) {
		write8(a, 0x8061, chunk.bank)
		write16(a, 0x8062, chunk.offset)
		write16(a, 0x8064, destAddr+offset)
		write16(a, 0x8066, chunk.count)
		write8(a, 0x8060, 0x01) // enable | copy | VRAM destination
		emitWaitForDMAIdle(a)
		offset += chunk.count
	}
}

func emitInitTrigTable(a *asm, tableBase uint16, steps int) {
	for i := 0; i < steps; i++ {
		angle := (2.0 * math.Pi * float64(i)) / float64(steps)
		cosv := int16(math.Round(math.Cos(angle) * 256.0))
		sinv := int16(math.Round(math.Sin(angle) * 256.0))
		write16s(a, tableBase+uint16(i*4), cosv)
		write16s(a, tableBase+uint16(i*4)+2, sinv)
	}
}

func emitLoadTrigPair(a *asm, tableBase uint16, indexReg, cosReg, sinReg uint8) {
	a.addReg(indexReg, indexReg)
	a.addReg(indexReg, indexReg)
	a.addImm(indexReg, tableBase)
	a.movLoad(cosReg, indexReg)
	a.addImm(indexReg, 2)
	a.movLoad(sinReg, indexReg)
}

func emitWriteMatrixRegs(a *asm, controlAddr, aAddr, bAddr, cAddr, dAddr, cxAddr, cyAddr uint16, controlValue uint8, aReg, bReg, cReg, dReg uint8, centerX, centerY int16) {
	write8(a, controlAddr, controlValue)
	write16Reg(a, aAddr, aReg)
	write16Reg(a, bAddr, bReg)
	write16Reg(a, cAddr, cReg)
	write16Reg(a, dAddr, dReg)
	write16s(a, cxAddr, centerX)
	write16s(a, cyAddr, centerY)
}

func put16LE(buf []byte, value int16) {
	u := uint16(value)
	buf[0] = byte(u)
	buf[1] = byte(u >> 8)
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

func buildMatrixBitmapShowcaseROM(img image.Image, outPath string) error {
	const (
		codeBank          = 1
		dataStartBank     = 2
		wramLastFrame     = 0x0200
		wramTrigTableBase = 0x0300
		trigSteps         = 32
		hdmaTableBase     = 0x0000
		floorPhaseCount   = 128
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
	floorTables := buildPerspectiveFloorTables(floorPhaseCount, floorPath)

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
	floorTableRefs := make([]romDataRef, floorPhaseCount)
	for i := range floorTables {
		floorTableRefs[i], cursor = allocateROMData(cursor, floorTables[i])
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

	emitVRAMDMAChunks(a, hdmaTableBase, floorTableRefs[0])

	emitInitTrigTable(a, wramTrigTableBase, trigSteps)

	a.movImm(4, 0x803F)
	a.movLoad(2, 4)
	a.movImm(4, wramLastFrame)
	a.movStore(4, 2)

	a.mark("main_loop")
	emitWaitOneFrame(a, wramLastFrame)

	// Floor plane: upload one fine-grained perspective table every frame.
	a.movImm(4, 0x803F)
	a.movLoad(0, 4)
	a.shrImm(0, floorPhaseShift)
	a.andImm(0, floorPhaseCount-1)
	labelAfterFloorTable := a.uniq("after_floor_table")
	phaseLabels := make([]string, floorPhaseCount)
	for i := 0; i < floorPhaseCount; i++ {
		phaseLabels[i] = a.uniq(fmt.Sprintf("floor_table_%02d", i))
	}
	for i := 0; i < floorPhaseCount; i++ {
		a.cmpImm(0, uint16(i))
		a.beq(phaseLabels[i])
	}
	emitVRAMDMAChunks(a, hdmaTableBase, floorTableRefs[floorPhaseCount-1])
	a.jmp(labelAfterFloorTable)
	for i := 0; i < floorPhaseCount; i++ {
		a.mark(phaseLabels[i])
		emitVRAMDMAChunks(a, hdmaTableBase, floorTableRefs[i])
		if i != floorPhaseCount-1 {
			a.jmp(labelAfterFloorTable)
		}
	}
	a.mark(labelAfterFloorTable)
	write16(a, 0x805E, hdmaTableBase)
	write8(a, 0x805D, 0x03) // enable HDMA, BG0 only
	write8(a, 0x807F, 0x00)
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
	emitText(a, 8, 20, 0xB0, 0xE0, 0xFF, "BG0 FLOOR  BG1 SKY  BG2 SIDE")

	a.jmp("main_loop")

	emitUploadRoutine(a, uploadRoutine)

	if err := a.resolve(); err != nil {
		return err
	}

	payload := append([]byte{}, floorAsset.Program.Bitmap...)
	payload = append(payload, skyAsset.Program.Bitmap...)
	payload = append(payload, sideAsset2.Program.Bitmap...)
	for i := range floorTables {
		payload = append(payload, floorTables[i]...)
	}
	if err := appendDataBlob(a.b, dataStartBank, payload); err != nil {
		return err
	}

	return a.b.BuildROM(codeBank, 0x8000, outPath)
}

func main() {
	inPath := flag.String("in", "/home/aj/Downloads/Test.png", "input PNG image")
	outPath := flag.String("out", "roms/matrix_floor_bitmap_showcase.rom", "output ROM path")
	flag.Parse()

	img, err := loadPNG(*inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load %s: %v\n", *inPath, err)
		os.Exit(1)
	}
	if err := buildMatrixBitmapShowcaseROM(img, *outPath); err != nil {
		fmt.Fprintf(os.Stderr, "build ROM: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Built %s using %s\n", *outPath, *inPath)
}
