package main

import (
	"flag"
	"image"
	"image/png"
	"os"

	"nitro-core-dx/internal/emulator"
)

func writePNG(path string, fb []uint32) error {
	img := image.NewNRGBA(image.Rect(0, 0, 320, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			c := fb[y*320+x]
			i := img.PixOffset(x, y)
			img.Pix[i+0] = uint8((c >> 16) & 0xFF)
			img.Pix[i+1] = uint8((c >> 8) & 0xFF)
			img.Pix[i+2] = uint8(c & 0xFF)
			img.Pix[i+3] = 0xFF
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func main() {
	romPath := flag.String("rom", "roms/matrix_floor_billboard_reference.rom", "ROM to run")
	outPath := flag.String("out", ".tmp_matrixfloor_reference/frame_000.png", "output PNG path")
	frames := flag.Int("frames", 32, "frames to run before capture")
	flag.Parse()

	romData, err := os.ReadFile(*romPath)
	if err != nil {
		panic(err)
	}
	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		panic(err)
	}
	emu.Running = true

	for i := 0; i < *frames; i++ {
		if err := emu.RunFrame(); err != nil {
			panic(err)
		}
	}

	if err := os.MkdirAll(".tmp_matrixfloor_reference", 0o755); err != nil {
		panic(err)
	}
	if err := writePNG(*outPath, emu.PPU.DisplayBuffer[:]); err != nil {
		panic(err)
	}
}
