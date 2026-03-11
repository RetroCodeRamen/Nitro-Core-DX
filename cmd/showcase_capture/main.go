package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"nitro-core-dx/internal/corelx"
	"nitro-core-dx/internal/emulator"
)

type frameSummary struct {
	Frame int    `json:"frame"`
	Name  string `json:"name"`
	Hash  string `json:"hash"`
	PNG   string `json:"png"`
}

func main() {
	var (
		sourcePath = flag.String("source", "test/roms/graphics_pipeline_showcase.corelx", "CoreLX showcase source path")
		outDir     = flag.String("out", "showcase_frames", "output directory for captured PNG frames")
	)
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		panic(fmt.Errorf("create output dir: %w", err))
	}

	romPath := filepath.Join(*outDir, "graphics_pipeline_showcase.rom")
	if err := corelx.CompileFile(*sourcePath, romPath); err != nil {
		panic(fmt.Errorf("compile showcase rom: %w", err))
	}

	romData, err := os.ReadFile(romPath)
	if err != nil {
		panic(fmt.Errorf("read rom: %w", err))
	}

	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		panic(fmt.Errorf("load rom: %w", err))
	}
	emu.Start()

	checkpoints := []struct {
		frame int
		name  string
	}{
		{frame: 120, name: "phase1_static"},
		{frame: 240, name: "phase2_sprite"},
		{frame: 420, name: "phase3_split"},
		{frame: 600, name: "phase4_warp"},
	}

	summaries := make([]frameSummary, 0, len(checkpoints))
	frame := 0
	for _, cp := range checkpoints {
		for frame < cp.frame {
			if err := emu.RunFrame(); err != nil {
				panic(fmt.Errorf("run frame %d: %w", frame, err))
			}
			frame++
		}

		pngName := cp.name + ".png"
		pngPath := filepath.Join(*outDir, pngName)
		if err := writeFramebufferPNG(pngPath, emu.PPU.OutputBuffer[:]); err != nil {
			panic(fmt.Errorf("write %s: %w", pngName, err))
		}
		summaries = append(summaries, frameSummary{
			Frame: cp.frame,
			Name:  cp.name,
			Hash:  framebufferHash(emu.PPU.OutputBuffer[:]),
			PNG:   pngName,
		})
	}

	summaryPath := filepath.Join(*outDir, "summary.json")
	data, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		panic(fmt.Errorf("marshal summary: %w", err))
	}
	if err := os.WriteFile(summaryPath, data, 0o644); err != nil {
		panic(fmt.Errorf("write summary: %w", err))
	}

	fmt.Printf("wrote showcase captures to %s\n", filepath.Clean(*outDir))
}

func framebufferHash(buf []uint32) string {
	raw := make([]byte, len(buf)*4)
	for i, px := range buf {
		raw[i*4+0] = byte(px)
		raw[i*4+1] = byte(px >> 8)
		raw[i*4+2] = byte(px >> 16)
		raw[i*4+3] = byte(px >> 24)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
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

