package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"nitro-core-dx/internal/emulator"
)

func main() {
	inPath := flag.String("in", "/home/aj/Downloads/Test.png", "input PNG image")
	outDir := flag.String("out", "/tmp/matrixpng", "output directory")
	flag.Parse()

	img, err := loadPNG(*inPath)
	if err != nil {
		panic(err)
	}
	asset, err := emulator.BuildBitmapMatrixPlaneAssetFromImage(img, 0, 2, 1)
	if err != nil {
		panic(err)
	}

	phases := emulator.DefaultBitmapMatrixPlaneValidationPhases()
	outputs := []string{
		"phase1_identity.png",
		"phase2_rotate_22_5.png",
		"phase3_rotate_45.png",
		"phase4_rotate_45_clamp.png",
		"phase5_skew_pan.png",
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		panic(err)
	}
	for i, p := range phases {
		fb, err := emulator.RenderBitmapMatrixPlanePhase(asset, p)
		if err != nil {
			panic(err)
		}
		if err := writeFramebufferPNG(filepath.Join(*outDir, outputs[i]), fb); err != nil {
			panic(err)
		}
	}

	summaryBytes, err := json.MarshalIndent(struct {
		Input   string                            `json:"input"`
		Outputs []string                          `json:"outputs"`
		Phases  []emulator.MatrixPlaneRenderPhase `json:"phases"`
	}{
		Input:   *inPath,
		Outputs: outputs,
		Phases:  phases,
	}, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(*outDir, "summary.json"), summaryBytes, 0o644); err != nil {
		panic(err)
	}

	fmt.Printf("wrote %d captures to %s\n", len(phases), filepath.Clean(*outDir))
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func writeFramebufferPNG(path string, fb []uint32) error {
	if len(fb) < 320*200 {
		return fmt.Errorf("framebuffer too small: %d", len(fb))
	}
	img := image.NewRGBA(image.Rect(0, 0, 320, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			c := fb[y*320+x]
			img.Set(x, y, color.RGBA{
				R: uint8((c >> 16) & 0xFF),
				G: uint8((c >> 8) & 0xFF),
				B: uint8(c & 0xFF),
				A: 0xFF,
			})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
