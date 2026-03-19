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
func emitVRAMDMAChunks(a *asm, destAddr uint16, ref romDataRef) {
	romutil.EmitVRAMDMAChunks(a.Asm, destAddr, ref)
}
func emitMatrixRowDMAChunks(a *asm, channel uint8, ref romDataRef) {
	romutil.EmitMatrixRowDMAChunks(a.Asm, channel, ref)
}
func emitInitTrigTable(a *asm, tableBase uint16, steps int) {
	romutil.EmitInitTrigTable(a.Asm, tableBase, steps)
}
func emitLoadTrigPair(a *asm, tableBase uint16, indexReg, cosReg, sinReg uint8) {
	romutil.EmitLoadTrigPair(a.Asm, tableBase, indexReg, cosReg, sinReg)
}

var (
	allocateROMData = romutil.AllocateROMData
	appendDataBlob  = romutil.AppendDataBlob
	padPayloadToRef = romutil.PadPayloadToRef
)

func emitInitHeadingTable(a *asm, tableBase uint16, steps int, moveSpeed float64) {
	for i := 0; i < steps; i++ {
		angle := (2.0 * math.Pi * float64(i)) / float64(steps)
		cosv := int16(math.Round(math.Cos(angle) * 256.0))
		sinv := int16(math.Round(math.Sin(angle) * 256.0))
		moveX := int16(math.Round(math.Cos(angle) * moveSpeed))
		moveY := int16(math.Round(math.Sin(angle) * moveSpeed))
		write16s(a, tableBase+uint16(i*8), cosv)
		write16s(a, tableBase+uint16(i*8)+2, sinv)
		write16s(a, tableBase+uint16(i*8)+4, moveX)
		write16s(a, tableBase+uint16(i*8)+6, moveY)
	}
}

func emitLoadHeadingEntry(a *asm, tableBase uint16, indexReg, headingXReg, headingYReg, moveXReg, moveYReg uint8) {
	a.movReg(4, indexReg)
	a.addReg(4, 4)
	a.addReg(4, 4)
	a.addReg(4, 4)
	a.addImm(4, tableBase)
	a.movLoad(headingXReg, 4)
	a.addImm(4, 2)
	a.movLoad(headingYReg, 4)
	a.addImm(4, 2)
	a.movLoad(moveXReg, 4)
	a.addImm(4, 2)
	a.movLoad(moveYReg, 4)
}

func emitWriteMatrixRegs(a *asm, controlAddr, aAddr, bAddr, cAddr, dAddr, cxAddr, cyAddr uint16, controlValue uint8, aReg, bReg, cReg, dReg uint8, centerX, centerY int16) {
	romutil.EmitWriteMatrixRegs(a.Asm, controlAddr, aAddr, bAddr, cAddr, dAddr, cxAddr, cyAddr, controlValue, aReg, bReg, cReg, dReg, centerX, centerY)
}

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

func packExpanded4BPP(pixels []uint8) []byte {
	packed := make([]byte, (len(pixels)+1)/2)
	for i := 0; i < len(pixels); i += 2 {
		hi := pixels[i] & 0x0F
		lo := uint8(0)
		if i+1 < len(pixels) {
			lo = pixels[i+1] & 0x0F
		}
		packed[i/2] = (hi << 4) | lo
	}
	return packed
}

type quantSample struct {
	r float64
	g float64
	b float64
}

func (s quantSample) distanceSquared(other quantSample) float64 {
	dr := s.r - other.r
	dg := s.g - other.g
	db := s.b - other.b
	return dr*dr + dg*dg + db*db
}

func (s quantSample) luminance() float64 {
	return 0.2126*s.r + 0.7152*s.g + 0.0722*s.b
}

func (s quantSample) saturation() float64 {
	maxc := s.r
	minc := s.r
	if s.g > maxc {
		maxc = s.g
	}
	if s.b > maxc {
		maxc = s.b
	}
	if s.g < minc {
		minc = s.g
	}
	if s.b < minc {
		minc = s.b
	}
	return maxc - minc
}

func nearestQuantCentroid(s quantSample, centroids []quantSample) int {
	best := 0
	bestDist := -1.0
	for i, c := range centroids {
		dist := s.distanceSquared(c)
		if bestDist < 0 || dist < bestDist {
			best = i
			bestDist = dist
		}
	}
	return best
}

func chooseQuantCentroids(samples []quantSample, count int) []quantSample {
	if count <= 0 {
		return nil
	}
	if len(samples) == 0 {
		return []quantSample{{}}
	}
	pushUnique := func(dst []quantSample, candidate quantSample) []quantSample {
		for _, existing := range dst {
			if existing.distanceSquared(candidate) < 9.0 {
				return dst
			}
		}
		return append(dst, candidate)
	}
	darkest := samples[0]
	brightest := samples[0]
	reddest := samples[0]
	greenest := samples[0]
	bluest := samples[0]
	mostSaturated := samples[0]
	for _, s := range samples[1:] {
		if s.luminance() < darkest.luminance() {
			darkest = s
		}
		if s.luminance() > brightest.luminance() {
			brightest = s
		}
		if (s.r - 0.5*s.g - 0.5*s.b) > (reddest.r - 0.5*reddest.g - 0.5*reddest.b) {
			reddest = s
		}
		if (s.g - 0.5*s.r - 0.5*s.b) > (greenest.g - 0.5*greenest.r - 0.5*greenest.b) {
			greenest = s
		}
		if (s.b - 0.5*s.r - 0.5*s.g) > (bluest.b - 0.5*bluest.r - 0.5*bluest.g) {
			bluest = s
		}
		if s.saturation() > mostSaturated.saturation() {
			mostSaturated = s
		}
	}
	centroids := make([]quantSample, 0, count)
	for _, seed := range []quantSample{darkest, brightest, reddest, greenest, bluest, mostSaturated} {
		centroids = pushUnique(centroids, seed)
		if len(centroids) == count {
			return centroids
		}
	}
	for len(centroids) < count {
		best := samples[len(centroids)%len(samples)]
		bestDist := -1.0
		for _, s := range samples {
			minDist := -1.0
			for _, c := range centroids {
				d := s.distanceSquared(c)
				if minDist < 0 || d < minDist {
					minDist = d
				}
			}
			if minDist > bestDist {
				best = s
				bestDist = minDist
			}
		}
		centroids = pushUnique(centroids, best)
		if len(centroids) > 0 && len(centroids) < count && bestDist <= 0 {
			centroids = append(centroids, samples[len(centroids)%len(samples)])
		}
	}
	return centroids[:count]
}

func rgbToRGB555Local(r, g, b uint8) uint16 {
	r5 := uint16((uint32(r) * 31) / 255)
	g5 := uint16((uint32(g) * 31) / 255)
	b5 := uint16((uint32(b) * 31) / 255)
	return r5 | (g5 << 5) | (b5 << 10)
}

func colorNearKey(c color.NRGBA, key color.NRGBA) bool {
	return math.Abs(float64(c.R)-float64(key.R)) <= 28 &&
		math.Abs(float64(c.G)-float64(key.G)) <= 28 &&
		math.Abs(float64(c.B)-float64(key.B)) <= 28
}

func fitImageToCanvasNearest(src image.Image, width, height int, key color.NRGBA) *image.NRGBA {
	bounds := src.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X-1, bounds.Min.Y-1
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
			if colorNearKey(c, key) {
				continue
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if maxX < minX || maxY < minY {
		minX, minY = bounds.Min.X, bounds.Min.Y
		maxX, maxY = bounds.Max.X-1, bounds.Max.Y-1
	}
	cropW := maxX - minX + 1
	cropH := maxY - minY + 1
	scale := math.Min(float64(width)/float64(cropW), float64(height)/float64(cropH))
	if scale <= 0 {
		scale = 1
	}
	scaledW := int(math.Round(float64(cropW) * scale))
	scaledH := int(math.Round(float64(cropH) * scale))
	if scaledW < 1 {
		scaledW = 1
	}
	if scaledH < 1 {
		scaledH = 1
	}
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(dst, dst.Bounds(), image.Transparent, image.Point{}, draw.Src)
	offX := (width - scaledW) / 2
	offY := height - scaledH
	if offY < 0 {
		offY = 0
	}
	for y := 0; y < scaledH; y++ {
		sy := minY + (y*cropH)/scaledH
		for x := 0; x < scaledW; x++ {
			sx := minX + (x*cropW)/scaledW
			dst.Set(offX+x, offY+y, src.At(sx, sy))
		}
	}
	return dst
}

func makeBillboardAssetFromImage(img image.Image, width, height int) ([]uint16, [][]byte, error) {
	if img == nil {
		return nil, nil, fmt.Errorf("billboard image is required")
	}
	key := color.NRGBAModel.Convert(img.At(img.Bounds().Min.X, img.Bounds().Min.Y)).(color.NRGBA)
	scaled := fitImageToCanvasNearest(img, width, height, key)
	transparent := make([]bool, width*height)
	samples := make([]quantSample, 0, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := color.NRGBAModel.Convert(scaled.At(x, y)).(color.NRGBA)
			idx := y*width + x
			if c.A < 128 || colorNearKey(c, key) {
				transparent[idx] = true
				continue
			}
			samples = append(samples, quantSample{r: float64(c.R), g: float64(c.G), b: float64(c.B)})
		}
	}
	if len(samples) == 0 {
		return nil, nil, fmt.Errorf("billboard image quantized to empty content")
	}
	centroids := chooseQuantCentroids(samples, 15)
	for iter := 0; iter < 8; iter++ {
		type accum struct {
			r float64
			g float64
			b float64
			n int
		}
		accums := make([]accum, len(centroids))
		for _, s := range samples {
			best := nearestQuantCentroid(s, centroids)
			accums[best].r += s.r
			accums[best].g += s.g
			accums[best].b += s.b
			accums[best].n++
		}
		for i := range centroids {
			if accums[i].n == 0 {
				continue
			}
			centroids[i] = quantSample{
				r: accums[i].r / float64(accums[i].n),
				g: accums[i].g / float64(accums[i].n),
				b: accums[i].b / float64(accums[i].n),
			}
		}
	}
	palette := make([]uint16, 16)
	palette[0] = 0x0000
	for i, c := range centroids {
		palette[i+1] = rgbToRGB555Local(uint8(c.r+0.5), uint8(c.g+0.5), uint8(c.b+0.5))
	}
	indexed := make([]uint8, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			if transparent[idx] {
				indexed[idx] = 0
				continue
			}
			c := color.NRGBAModel.Convert(scaled.At(x, y)).(color.NRGBA)
			best := nearestQuantCentroid(quantSample{r: float64(c.R), g: float64(c.G), b: float64(c.B)}, centroids)
			indexed[idx] = uint8(best + 1)
		}
	}
	tileCols := width / 16
	tileRows := height / 16
	tiles := make([][]byte, 0, tileCols*tileRows)
	for ty := 0; ty < tileRows; ty++ {
		for tx := 0; tx < tileCols; tx++ {
			pixels := make([]uint8, 16*16)
			for y := 0; y < 16; y++ {
				for x := 0; x < 16; x++ {
					srcX := tx*16 + x
					srcY := ty*16 + y
					pixels[y*16+x] = indexed[srcY*width+srcX]
				}
			}
			tiles = append(tiles, packExpanded4BPP(pixels))
		}
	}
	return palette, tiles, nil
}

func makeBillboardMarker16() []byte {
	pixels := make([]uint8, 16*16)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			idx := y*16 + x
			switch {
			case x >= 7 && x <= 8 && y >= 9:
				pixels[idx] = 1
			case y >= 1 && y <= 10 && x >= 2 && x <= 13:
				if x == 2 || x == 13 || y == 1 || y == 10 {
					pixels[idx] = 1
				} else if y >= 4 && y <= 5 {
					pixels[idx] = 3
				} else {
					pixels[idx] = 2
				}
			}
		}
	}
	return packExpanded4BPP(pixels)
}

func makeBillboardMarker64x80Tiles() [][]byte {
	const width = 64
	const height = 80
	pixels := make([]uint8, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			switch {
			case x >= 30 && x <= 33 && y >= 42:
				pixels[idx] = 1
			case y >= 8 && y <= 43 && x >= 8 && x <= 55:
				if x == 8 || x == 55 || y == 8 || y == 43 {
					pixels[idx] = 1
				} else if y >= 18 && y <= 23 {
					pixels[idx] = 3
				} else {
					pixels[idx] = 2
				}
			case y >= 32 && y <= 36 && x >= 20 && x <= 43:
				pixels[idx] = 1
			}
		}
	}
	tiles := make([][]byte, 20)
	for tileY := 0; tileY < 5; tileY++ {
		for tileX := 0; tileX < 4; tileX++ {
			tilePixels := make([]uint8, 16*16)
			for y := 0; y < 16; y++ {
				for x := 0; x < 16; x++ {
					srcX := tileX*16 + x
					srcY := tileY*16 + y
					tilePixels[y*16+x] = pixels[srcY*width+srcX]
				}
			}
			tiles[tileY*4+tileX] = packExpanded4BPP(tilePixels)
		}
	}
	return tiles
}

func makeBillboardMarker8() []byte {
	pixels := make([]uint8, 8*8)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			idx := y*8 + x
			switch {
			case x == 3 || x == 4:
				if y >= 5 {
					pixels[idx] = 1
				}
			case y >= 1 && y <= 4 && x >= 1 && x <= 6:
				if x == 1 || x == 6 || y == 1 || y == 4 {
					pixels[idx] = 1
				} else if y == 2 {
					pixels[idx] = 3
				} else {
					pixels[idx] = 2
				}
			}
		}
	}
	return packExpanded4BPP(pixels)
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
	// Drive around the perimeter but bias the view slightly inward so the
	// motion reads like circling a track instead of just spinning in place.
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
			// Treat each scanline as a projected floor row:
			//   world(x) = rowOrigin + x * rowStep
			// This is closer to how SNES-style floor effects are driven in
			// practice: HDMA changes the row mapping, rather than using one
			// full-screen affine coefficient set and hoping it reads as
			// perspective.
			line := float64(y-horizonY) + 1.0
			// Keep the far rows compressed, but let the lower rows reuse
			// neighboring texels heavily. The earlier curve stepped through
			// source space too aggressively, which made the floor look torn
			// apart instead of stretched into perspective.
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

func buildInteractivePerspectiveRowTable(positionCount int, path []trackPoint, pathIndex int, yawOffset float64) []byte {
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
	phase := (2.0 * math.Pi * float64(pathIndex)) / float64(positionCount)
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
	baseForwardX := tangentX*0.92 + inwardX*0.28
	baseForwardY := tangentY*0.92 + inwardY*0.28
	baseForwardLen := math.Hypot(baseForwardX, baseForwardY)
	if baseForwardLen != 0 {
		baseForwardX /= baseForwardLen
		baseForwardY /= baseForwardLen
	}
	baseRightX := baseForwardY
	baseRightY := -baseForwardX

	cosYaw := math.Cos(yawOffset)
	sinYaw := math.Sin(yawOffset)
	forwardX := baseForwardX*cosYaw + baseRightX*sinYaw
	forwardY := baseForwardY*cosYaw + baseRightY*sinYaw
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

func buildInteractivePerspectiveRowTables(positionCount int, yawOffsets []float64, path []trackPoint) [][]byte {
	tables := make([][]byte, positionCount*len(yawOffsets))
	for pos := 0; pos < positionCount; pos++ {
		for look := range yawOffsets {
			tables[pos*len(yawOffsets)+look] = buildInteractivePerspectiveRowTable(positionCount, path, pos, yawOffsets[look])
		}
	}
	return tables
}

func closestTrackIndex(path []trackPoint, x, y float64) int {
	if len(path) == 0 {
		return 0
	}
	best := 0
	bestDist := math.MaxFloat64
	for i, pt := range path {
		dx := pt.x - x
		dy := pt.y - y
		dist := dx*dx + dy*dy
		if dist < bestDist {
			best = i
			bestDist = dist
		}
	}
	return best
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

func makeDiagnosticFloorCanvas(canvasPixels int) image.Image {
	dst := image.NewNRGBA(image.Rect(0, 0, canvasPixels, canvasPixels))
	centerX := canvasPixels / 2
	cellColors := []color.NRGBA{
		{R: 0x70, G: 0x70, B: 0x78, A: 0xFF},
		{R: 0x90, G: 0x78, B: 0x58, A: 0xFF},
		{R: 0x58, G: 0x88, B: 0x90, A: 0xFF},
		{R: 0x88, G: 0x90, B: 0x58, A: 0xFF},
	}

	for y := 0; y < canvasPixels; y++ {
		for x := 0; x < canvasPixels; x++ {
			cellX := (x / 64) % len(cellColors)
			cellY := (y / 64) % len(cellColors)
			dst.SetNRGBA(x, y, cellColors[(cellX+cellY)%len(cellColors)])
		}
	}

	for y := 0; y < canvasPixels; y += 64 {
		for x := 0; x < canvasPixels; x++ {
			c := color.NRGBA{R: 0xF0, G: 0xF0, B: 0xF0, A: 0xFF}
			dst.SetNRGBA(x, y, c)
			if y+1 < canvasPixels {
				dst.SetNRGBA(x, y+1, c)
			}
		}
	}

	for x := 0; x < canvasPixels; x += 64 {
		for y := 0; y < canvasPixels; y++ {
			c := color.NRGBA{R: 0xE8, G: 0xE8, B: 0xE8, A: 0xFF}
			dst.SetNRGBA(x, y, c)
			if x+1 < canvasPixels {
				dst.SetNRGBA(x+1, y, c)
			}
		}
	}

	for y := 0; y < canvasPixels; y++ {
		for dx := -8; dx <= 8; dx++ {
			x := centerX + dx
			if x >= 0 && x < canvasPixels {
				dst.SetNRGBA(x, y, color.NRGBA{R: 0xF8, G: 0xD8, B: 0x30, A: 0xFF})
			}
		}
	}

	for y := 0; y < canvasPixels; y++ {
		for dx := 0; dx < 12; dx++ {
			if dx < canvasPixels {
				dst.SetNRGBA(dx, y, color.NRGBA{R: 0xE0, G: 0x30, B: 0x30, A: 0xFF})
				dst.SetNRGBA(canvasPixels-1-dx, y, color.NRGBA{R: 0x30, G: 0x60, B: 0xF0, A: 0xFF})
			}
		}
	}

	for y := 96; y < canvasPixels; y += 160 {
		for i := 0; i < 72; i++ {
			yy := y + i
			if yy >= canvasPixels {
				break
			}
			for dx := -i; dx <= i; dx++ {
				x := centerX + dx
				if x >= 0 && x < canvasPixels {
					dst.SetNRGBA(x, yy, color.NRGBA{R: 0x40, G: 0xD0, B: 0x80, A: 0xFF})
				}
			}
		}
	}

	return dst
}

func buildMatrixFloorBillboardShowcaseROM(floorImg image.Image, billboardImg image.Image, outPath string) error {
	const (
		codeBank             = 1
		dataStartBank        = 2
		wramLastFrame        = 0x0200
		wramHeadingIndex     = 0x0202
		wramCameraX          = 0x0204
		wramCameraY          = 0x0206
		headingTableBase     = 0x0300
		headingSteps         = 64
		turnStep             = 2
		moveSpeed            = 12.0
		floorHorizon         = 72
		billboardHorizon     = 72
		floorBaseDist        = 0x01C0
		floorFocal           = 0x3A00
		floorWidthScale      = 0x00C0
		billboardBaseDist    = 0x01C0
		billboardFocal       = 0x3A00
		billboardWidthScale  = 0x00B8
		billboardOriginX     = 512
		billboardOriginY     = 640
		billboardHeightScale = 0x02C0
		matrixPlane0Ctl      = 0x1D // enabled, 128x128, bitmap, palette bank 1
		matrixPlane1Ctl      = 0x2D // enabled, 128x128, bitmap, palette bank 2
		matrixPlane2Ctl      = 0x3B // enabled, 64x64, bitmap, palette bank 3
		matrixPlane0Flags    = 0x00 // opaque floor
		matrixPlane1Flags    = 0x01 // palette index 0 transparent
		matrixPlane2Flags    = 0x03 // transparent index 0 + two sided
	)

	floorAsset, err := emulator.BuildBitmapMatrixPlaneAssetFromImage(floorImg, 0, ppucore.TilemapSize128x128, 1)
	if err != nil {
		return err
	}
	skyOverlay := makeSkyOverlayCanvas(1024)
	skyAsset, err := emulator.BuildBitmapMatrixPlaneAssetFromImage(skyOverlay, 1, ppucore.TilemapSize128x128, 2)
	if err != nil {
		return err
	}
	if billboardImg == nil {
		return fmt.Errorf("billboard image is required")
	}
	key := color.NRGBAModel.Convert(billboardImg.At(billboardImg.Bounds().Min.X, billboardImg.Bounds().Min.Y)).(color.NRGBA)
	billboardCanvas := fitImageToCanvasNearest(billboardImg, 512, 512, key)
	billboardAsset, err := emulator.BuildBitmapMatrixPlaneAssetFromImage(billboardCanvas, 2, ppucore.TilemapSize64x64, 3)
	if err != nil {
		return err
	}

	floorRef, cursor := allocateROMData(0, floorAsset.Program.Bitmap)
	skyRef, cursor := allocateROMData(cursor, skyAsset.Program.Bitmap)
	billboardRef, cursor := allocateROMData(cursor, billboardAsset.Program.Bitmap)
	_ = billboardRef
	_ = cursor

	a := newASM(codeBank)
	// Palette setup.
	for i, c := range floorAsset.Palette {
		setCGRAMColor(a, uint8(1*16+i), c)
	}
	for i, c := range skyAsset.Palette {
		setCGRAMColor(a, uint8(2*16+i), c)
	}
	for i, c := range billboardAsset.Palette {
		setCGRAMColor(a, uint8(3*16+i), c)
	}
	setCGRAMColor(a, 0, 0)
	write16(a, wramHeadingIndex, 48) // roughly "up" in source space
	write16(a, wramCameraX, 512)
	write16(a, wramCameraY, 768)
	emitInitHeadingTable(a, headingTableBase, headingSteps, moveSpeed)

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

	// Upload plane 2 bitmap (upright billboard/object).
	write8(a, 0x8080, 0x02)
	write8(a, 0x8081, matrixPlane2Ctl)
	write8(a, 0x808C, matrixPlane2Flags)
	write8(a, 0x8088, 0x00)
	write8(a, 0x8089, 0x00)
	write8(a, 0x808A, 0x00)
	emitMatrixBitmapDMAChunks(a, 0x02, billboardRef)

	a.movImm(0, 0x0000)
	a.setDBR(0)

	a.movImm(4, 0x803F)
	a.movLoad(2, 4)
	a.movImm(4, wramLastFrame)
	a.movStore(4, 2)

	write8(a, 0x805D, 0x00) // HDMA off; row tables are uploaded directly each frame
	write8(a, 0x807F, 0x00)
	write8(a, 0x8080, 0x00)
	write8(a, 0x8081, matrixPlane0Ctl)
	write8(a, 0x808C, matrixPlane0Flags)
	write8(a, 0x8091, 0x01) // generic perspective projection mode
	write8(a, 0x8092, floorHorizon)
	write16(a, 0x809B, floorBaseDist)
	write16(a, 0x809D, floorFocal)
	write16(a, 0x809F, floorWidthScale)
	a.movImm(0, 48)
	emitLoadHeadingEntry(a, headingTableBase, 0, 1, 2, 3, 6)
	write16(a, 0x8093, 512)
	write16(a, 0x8095, 768)
	write16RegBytes(a, 0x8097, 1)
	write16RegBytes(a, 0x8099, 2)
	write8(a, 0x8018, 0x01) // BG0 matrix mode enabled, wrap outside behavior

	// Full-screen sky/horizon overlay: identity transform, clamp.
	write16(a, 0x8004, 512)
	write16(a, 0x8006, 512)
	a.movImm(2, 0x0100)
	a.movImm(3, 0x0000)
	a.movImm(6, 0x0000)
	emitWriteMatrixRegs(a, 0x802B, 0x802C, 0x802E, 0x8030, 0x8032, 0x8034, 0x8036, 0x19, 2, 3, 6, 2, 160, 100)

	// Upright world-anchored billboard on a separate matrix plane.
	write8(a, 0x8080, 0x02)
	write8(a, 0x8081, matrixPlane2Ctl)
	write8(a, 0x808C, matrixPlane2Flags)
	write8(a, 0x8091, 0x02) // generic vertical projected quad
	write8(a, 0x8092, billboardHorizon)
	write16(a, 0x8093, 512)
	write16(a, 0x8095, 768)
	write16RegBytes(a, 0x8097, 1)
	write16RegBytes(a, 0x8099, 2)
	write16(a, 0x809B, billboardBaseDist)
	write16(a, 0x809D, billboardFocal)
	write16(a, 0x809F, billboardWidthScale)
	write16(a, 0x80A1, billboardOriginX)
	write16(a, 0x80A3, billboardOriginY)
	write16(a, 0x80A5, 0x0000)
	write16(a, 0x80A7, 0x0100)
	write16(a, 0x80A9, billboardHeightScale)
	emitWriteMatrixRegs(a, 0x8038, 0x8039, 0x803B, 0x803D, 0x803F, 0x8041, 0x8043, 0x19, 2, 3, 6, 2, 160, billboardHorizon)
	write8(a, 0x8011, 0x01) // DISPLAY_CONTROL: enable display

	a.mark("main_loop")
	emitWaitOneFrame(a, wramLastFrame)

	// Controller-driven interactive row-mode floor:
	//   LEFT/RIGHT = look
	//   UP/DOWN    = move along track
	write8(a, 0xA001, 0x01)
	a.movImm(4, 0xA000)
	a.movLoad(2, 4) // controller low word; low byte carries d-pad/buttons
	write8(a, 0xA001, 0x00)
	a.movReg(5, 2)

	a.movImm(4, wramHeadingIndex)
	a.movLoad(0, 4)

	noTurnLeft := a.uniq("no_turn_left")
	a.movReg(4, 5)
	a.andImm(4, 0x0004)
	a.cmpImm(4, 0)
	a.beq(noTurnLeft)
	afterTurnLeft := a.uniq("after_turn_left")
	lookLeftWrap0 := a.uniq("look_left_wrap0")
	lookLeftWrap1 := a.uniq("look_left_wrap1")
	a.cmpImm(0, 0)
	a.beq(lookLeftWrap0)
	a.cmpImm(0, 1)
	a.beq(lookLeftWrap1)
	a.subImm(0, turnStep)
	a.jmp(afterTurnLeft)
	a.mark(lookLeftWrap0)
	a.movImm(0, headingSteps-turnStep)
	a.jmp(afterTurnLeft)
	a.mark(lookLeftWrap1)
	a.movImm(0, headingSteps-1)
	a.mark(afterTurnLeft)
	a.mark(noTurnLeft)

	noTurnRight := a.uniq("no_turn_right")
	a.movReg(4, 5)
	a.andImm(4, 0x0008)
	a.cmpImm(4, 0)
	a.beq(noTurnRight)
	afterTurnRight := a.uniq("after_turn_right")
	lookRightWrap62 := a.uniq("look_right_wrap62")
	lookRightWrap63 := a.uniq("look_right_wrap63")
	a.cmpImm(0, headingSteps-turnStep)
	a.beq(lookRightWrap62)
	a.cmpImm(0, headingSteps-1)
	a.beq(lookRightWrap63)
	a.addImm(0, turnStep)
	a.jmp(afterTurnRight)
	a.mark(lookRightWrap62)
	a.movImm(0, 0)
	a.jmp(afterTurnRight)
	a.mark(lookRightWrap63)
	a.movImm(0, 1)
	a.mark(afterTurnRight)
	a.mark(noTurnRight)
	a.movImm(4, wramHeadingIndex)
	a.movStore(4, 0)

	emitLoadHeadingEntry(a, headingTableBase, 0, 1, 2, 3, 6)

	a.movImm(7, wramCameraX)
	a.movLoad(4, 7)
	a.movImm(7, wramCameraY)
	a.movLoad(0, 7)

	noMoveForward := a.uniq("no_move_forward")
	a.movReg(7, 5)
	a.andImm(7, 0x0001)
	a.cmpImm(7, 0)
	a.beq(noMoveForward)
	a.addReg(4, 3)
	a.addReg(0, 6)
	a.mark(noMoveForward)

	noMoveBackward := a.uniq("no_move_backward")
	a.movReg(7, 5)
	a.andImm(7, 0x0002)
	a.cmpImm(7, 0)
	a.beq(noMoveBackward)
	a.subReg(4, 3)
	a.subReg(0, 6)
	a.mark(noMoveBackward)

	a.movImm(7, wramCameraX)
	a.movStore(7, 4)
	a.movImm(7, wramCameraY)
	a.movStore(7, 0)

	write8Scratch(a, 0x8080, 0x00, 7, 5)
	a.movReg(3, 4)
	a.movReg(6, 0)
	write16RegBytes(a, 0x8093, 3)
	write16RegBytes(a, 0x8095, 6)
	write16RegBytes(a, 0x8097, 1)
	write16RegBytes(a, 0x8099, 2)

	write8Scratch(a, 0x8080, 0x02, 7, 5)
	write16RegBytes(a, 0x8093, 3)
	write16RegBytes(a, 0x8095, 6)
	write16RegBytes(a, 0x8097, 1)
	write16RegBytes(a, 0x8099, 2)

	emitText(a, 8, 8, 0xF8, 0xF8, 0xF8, "FLOOR + BILLBOARD MATRIX DEMO")
	emitText(a, 8, 20, 0xB0, 0xE0, 0xFF, "BG0 FLOOR  BG1 SKY  BG2 OBJECT")
	emitText(a, 8, 32, 0xE0, 0xE0, 0x90, "UP/DOWN MOVE  LEFT/RIGHT TURN")
	emitText(a, 8, 44, 0xFF, 0xD0, 0x70, "GENERIC PROJECTION MODES")

	a.jmp("main_loop")

	if err := a.resolve(); err != nil {
		return err
	}

	payload := append([]byte{}, floorAsset.Program.Bitmap...)
	payload = append(payload, skyAsset.Program.Bitmap...)
	payload = append(payload, billboardAsset.Program.Bitmap...)
	if err := appendDataBlob(a.B, dataStartBank, payload); err != nil {
		return err
	}

	return a.B.BuildROM(codeBank, 0x8000, outPath)
}

func main() {
	inPath := flag.String("in", "Resources/kart.png", "input PNG image")
	billboardPath := flag.String("billboard", "Resources/Test.png", "billboard PNG image")
	outPath := flag.String("out", "roms/matrix_floor_billboard_showcase.rom", "output ROM path")
	diagnostic := flag.Bool("diagnostic", false, "use a generated diagnostic floor image instead of loading a PNG")
	flag.Parse()

	var (
		floorImg image.Image
		err      error
	)
	if *diagnostic {
		floorImg = makeDiagnosticFloorCanvas(1024)
	} else {
		floorImg, err = loadPNG(*inPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load %s: %v\n", *inPath, err)
			os.Exit(1)
		}
	}
	billboardImg, err := loadPNG(*billboardPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load %s: %v\n", *billboardPath, err)
		os.Exit(1)
	}
	if err := buildMatrixFloorBillboardShowcaseROM(floorImg, billboardImg, *outPath); err != nil {
		fmt.Fprintf(os.Stderr, "build ROM: %v\n", err)
		os.Exit(1)
	}
	if *diagnostic {
		fmt.Printf("Built %s using generated diagnostic floor image and %s billboard\n", *outPath, *billboardPath)
	} else {
		fmt.Printf("Built %s using %s floor and %s billboard\n", *outPath, *inPath, *billboardPath)
	}
}
