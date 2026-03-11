package emulator

import (
	"image"
	"image/color"
	"testing"
)

func TestRenderBitmapMatrixPlaneValidationPhasesDiffer(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 128, 128))
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			switch {
			case x < 16 && y < 16:
				img.Set(x, y, color.RGBA{R: 255, G: 255, A: 255})
			case x > 96 && y < 24:
				img.Set(x, y, color.RGBA{R: 255, A: 255})
			case x < 24 && y > 96:
				img.Set(x, y, color.RGBA{G: 255, A: 255})
			case (x+y)%19 == 0:
				img.Set(x, y, color.RGBA{B: 255, A: 255})
			case x > y && x < 96:
				img.Set(x, y, color.RGBA{R: 255, B: 255, A: 255})
			default:
				img.Set(x, y, color.RGBA{
					R: uint8((x * 255) / 127),
					G: uint8((y * 255) / 127),
					B: uint8(((x ^ y) * 255) / 127),
					A: 255,
				})
			}
		}
	}

	asset, err := BuildBitmapMatrixPlaneAssetFromImage(img, 0, 2, 1)
	if err != nil {
		t.Fatalf("BuildBitmapMatrixPlaneAssetFromImage: %v", err)
	}
	phases := DefaultBitmapMatrixPlaneValidationPhases()
	hashes := make(map[uint64]string)
	for _, phase := range phases {
		fb, err := RenderBitmapMatrixPlanePhase(asset, phase)
		if err != nil {
			t.Fatalf("RenderBitmapMatrixPlanePhase(%s): %v", phase.Name, err)
		}
		h := framebufferHash64(fb)
		if prev, exists := hashes[h]; exists {
			t.Fatalf("phase %s matched framebuffer hash of %s", phase.Name, prev)
		}
		hashes[h] = phase.Name
	}
}

func TestRenderBitmapMatrixPlaneClampDiffersFromWrapAtEdge(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			if x < 8 || y < 8 {
				img.Set(x, y, color.RGBA{R: 255, A: 255})
			} else {
				img.Set(x, y, color.RGBA{G: 255, A: 255})
			}
		}
	}

	asset, err := BuildBitmapMatrixPlaneAssetFromImage(img, 0, 2, 1)
	if err != nil {
		t.Fatalf("BuildBitmapMatrixPlaneAssetFromImage: %v", err)
	}

	wrap := MatrixPlaneRenderPhase{
		Name: "wrap",
		A:    0x0100, B: 0, C: 0, D: 0x0100,
		CenterX: 0, CenterY: 0,
		ScrollX: -96, ScrollY: -96,
		OutsideMode: 0,
	}
	clamp := wrap
	clamp.Name = "clamp"
	clamp.OutsideMode = 3

	fbWrap, err := RenderBitmapMatrixPlanePhase(asset, wrap)
	if err != nil {
		t.Fatalf("RenderBitmapMatrixPlanePhase(wrap): %v", err)
	}
	fbClamp, err := RenderBitmapMatrixPlanePhase(asset, clamp)
	if err != nil {
		t.Fatalf("RenderBitmapMatrixPlanePhase(clamp): %v", err)
	}
	if framebufferHash64(fbWrap) == framebufferHash64(fbClamp) {
		t.Fatal("wrap and clamp framebuffers matched; expected edge behavior difference")
	}
}

func framebufferHash64(fb []uint32) uint64 {
	var h uint64 = 1469598103934665603
	for _, px := range fb {
		h ^= uint64(px)
		h *= 1099511628211
	}
	return h
}
