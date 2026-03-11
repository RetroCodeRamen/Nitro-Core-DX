package emulator

import (
	"fmt"
	"image"
	"image/color"
)

type MatrixPlaneBitmapAsset struct {
	Program MatrixPlaneProgram
	Palette []uint16
}

type imageSample struct {
	r float64
	g float64
	b float64
}

func (s imageSample) distanceSquared(other imageSample) float64 {
	dr := s.r - other.r
	dg := s.g - other.g
	db := s.b - other.b
	return dr*dr + dg*dg + db*db
}

func (s imageSample) luminance() float64 {
	return 0.2126*s.r + 0.7152*s.g + 0.0722*s.b
}

func (s imageSample) saturation() float64 {
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

func BuildBitmapMatrixPlaneAssetFromImage(img image.Image, channel, sizeMode, paletteBank uint8) (*MatrixPlaneBitmapAsset, error) {
	if img == nil {
		return nil, fmt.Errorf("image is required")
	}
	builder, err := NewMatrixPlaneBuilder(channel, sizeMode)
	if err != nil {
		return nil, err
	}

	targetTiles, err := matrixPlaneWidth(sizeMode)
	if err != nil {
		return nil, err
	}
	targetPixels := targetTiles * 8
	resized := resizeImageNearest(img, targetPixels, targetPixels)
	palette, indexed, hasTransparency := quantizeImageTo4bpp(resized, 16)

	packed := make([]byte, (targetPixels*targetPixels)/2)
	for i, idx := range indexed {
		byteOffset := i / 2
		if i%2 == 0 {
			packed[byteOffset] = (packed[byteOffset] & 0x0F) | ((idx & 0x0F) << 4)
		} else {
			packed[byteOffset] = (packed[byteOffset] & 0xF0) | (idx & 0x0F)
		}
	}
	if err := builder.SetBitmapPacked4bpp(packed, paletteBank); err != nil {
		return nil, err
	}
	builder.SetBitmapTransparency(hasTransparency)

	return &MatrixPlaneBitmapAsset{
		Program: builder.Build(),
		Palette: palette,
	}, nil
}

func resizeImageNearest(src image.Image, width, height int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	for y := 0; y < height; y++ {
		sy := srcBounds.Min.Y + (y*srcH)/height
		for x := 0; x < width; x++ {
			sx := srcBounds.Min.X + (x*srcW)/width
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func quantizeImageTo4bpp(img image.Image, maxColors int) ([]uint16, []uint8, bool) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	totalPixels := width * height
	if maxColors < 1 {
		maxColors = 1
	}
	if maxColors > 16 {
		maxColors = 16
	}

	transparent := make([]bool, totalPixels)
	hasTransparency := false
	samples := make([]imageSample, 0, 16384)
	step := 1
	if totalPixels > 16384 {
		for step*step < totalPixels/16384 {
			step++
		}
	}
	pixelIndex := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if uint8(a>>8) < 128 {
				transparent[pixelIndex] = true
				hasTransparency = true
			}
			pixelIndex++
		}
	}

	clusterColors := maxColors
	if hasTransparency {
		clusterColors--
		if clusterColors < 1 {
			clusterColors = 1
		}
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			r, g, b, a := img.At(x, y).RGBA()
			if hasTransparency && uint8(a>>8) < 128 {
				continue
			}
			samples = append(samples, imageSample{
				r: float64(uint8(r >> 8)),
				g: float64(uint8(g >> 8)),
				b: float64(uint8(b >> 8)),
			})
		}
	}
	if len(samples) == 0 {
		samples = append(samples, imageSample{})
	}

	centroids := chooseInitialCentroids(samples, clusterColors)

	for iter := 0; iter < 8; iter++ {
		type accum struct {
			r, g, b float64
			n       int
		}
		accums := make([]accum, clusterColors)
		for _, s := range samples {
			best := nearestCentroid(s, centroids)
			accums[best].r += s.r
			accums[best].g += s.g
			accums[best].b += s.b
			accums[best].n++
		}
		for i := range centroids {
			if accums[i].n == 0 {
				centroids[i] = samples[(i*len(samples))/clusterColors]
				continue
			}
			centroids[i] = imageSample{
				r: accums[i].r / float64(accums[i].n),
				g: accums[i].g / float64(accums[i].n),
				b: accums[i].b / float64(accums[i].n),
			}
		}
	}

	palette := make([]uint16, maxColors)
	paletteBase := 0
	if hasTransparency {
		palette[0] = 0x0000
		paletteBase = 1
	}
	for i, c := range centroids {
		palette[paletteBase+i] = rgbToRGB555(uint8(c.r+0.5), uint8(c.g+0.5), uint8(c.b+0.5))
	}

	indexed := make([]uint8, totalPixels)
	out := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if hasTransparency && transparent[out] {
				indexed[out] = 0
				out++
				continue
			}
			r, g, b, _ := img.At(x, y).RGBA()
			best := nearestCentroid(imageSample{
				r: float64(uint8(r >> 8)),
				g: float64(uint8(g >> 8)),
				b: float64(uint8(b >> 8)),
			}, centroids)
			indexed[out] = uint8(best + paletteBase)
			out++
		}
	}

		return palette, indexed, hasTransparency
}

func nearestCentroid(s imageSample, centroids []imageSample) int {
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

func chooseInitialCentroids(samples []imageSample, count int) []imageSample {
	if count <= 0 {
		return nil
	}
	if len(samples) == 0 {
		return []imageSample{{}}
	}

	pushUnique := func(dst []imageSample, candidate imageSample) []imageSample {
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

	centroids := make([]imageSample, 0, count)
	seedOrder := []imageSample{darkest, brightest, reddest, greenest, bluest, mostSaturated}
	for _, seed := range seedOrder {
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

func rgbToRGB555(r, g, b uint8) uint16 {
	r5 := uint16((uint32(r) * 31) / 255)
	g5 := uint16((uint32(g) * 31) / 255)
	b5 := uint16((uint32(b) * 31) / 255)
	return r5 | (g5 << 5) | (b5 << 10)
}

func RGB555ToNRGBA(v uint16) color.NRGBA {
	r5 := uint32(v & 0x1F)
	g5 := uint32((v >> 5) & 0x1F)
	b5 := uint32((v >> 10) & 0x1F)
	return color.NRGBA{
		R: uint8((r5 * 255) / 31),
		G: uint8((g5 * 255) / 31),
		B: uint8((b5 * 255) / 31),
		A: 0xFF,
	}
}
