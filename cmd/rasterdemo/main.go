package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"nitro-core-dx/internal/devkit"
)

const idleRasterDemoSource = `
function Start()
    while true
        wait_vblank()
`

func main() {
	var (
		demoName   = flag.String("demo", devkit.RasterDemoSplitTilemap, "raster demo to render: split-tilemap, rebind-priority, scroll-affine, or matrix-plane")
		outputPath = flag.String("out", "raster_demo.png", "output PNG path")
	)
	flag.Parse()

	tmpDir, err := os.MkdirTemp("", "ncdx-rasterdemo-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	svc := devkit.NewService(tmpDir)
	defer svc.Shutdown()

	build, err := svc.BuildSource(idleRasterDemoSource, "raster_demo.corelx")
	if err != nil {
		panic(fmt.Errorf("build demo ROM: %w", err))
	}
	if err := svc.LoadROMBytes(build.Result.ROMBytes); err != nil {
		panic(fmt.Errorf("load demo ROM: %w", err))
	}
	if err := svc.InstallRasterDemo(*demoName); err != nil {
		panic(fmt.Errorf("install raster demo %q: %w", *demoName, err))
	}
	if err := svc.RunFrame(); err != nil {
		panic(fmt.Errorf("run frame: %w", err))
	}

	if err := writeFramebufferPNG(*outputPath, svc.FramebufferCopy()); err != nil {
		panic(fmt.Errorf("write png: %w", err))
	}

	fmt.Printf("wrote %s using demo %s\n", filepath.Clean(*outputPath), *demoName)
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
