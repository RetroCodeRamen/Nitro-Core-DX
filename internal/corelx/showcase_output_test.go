package corelx

import (
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"nitro-core-dx/internal/emulator"
)

func compileGraphicsPipelineShowcaseROM(t *testing.T) []byte {
	t.Helper()

	var sourcePath string
	possiblePaths := []string{
		"test/roms/graphics_pipeline_showcase.corelx",
		"../../test/roms/graphics_pipeline_showcase.corelx",
		"../test/roms/graphics_pipeline_showcase.corelx",
	}
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			sourcePath = path
			break
		}
	}
	if sourcePath == "" {
		t.Skip("graphics pipeline showcase not found")
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "graphics_pipeline_showcase.rom")
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	return romData
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
		return os.ErrInvalid
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

func maybeDumpShowcaseFrame(t *testing.T, name string, fb []uint32) {
	t.Helper()
	outDir := os.Getenv("NCDX_SHOWCASE_DUMP_DIR")
	if outDir == "" {
		return
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("create showcase dump dir: %v", err)
	}
	path := filepath.Join(outDir, name+".png")
	if err := writeFramebufferPNG(path, fb); err != nil {
		t.Fatalf("write showcase png %s: %v", path, err)
	}
}

func TestGraphicsPipelineShowcaseGoldenFrames(t *testing.T) {
	romData := compileGraphicsPipelineShowcaseROM(t)

	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	checkpoints := []struct {
		frame int
		hash  string
		name  string
	}{
		{frame: 120, hash: "e7fad03b63a3afb48799386efe751276d4ae14ec8b6cf98a9fe2a4841a8d280b", name: "phase1_static"},
		{frame: 240, hash: "64ab9a9efeb179f76d07bc2620d4443d9d0d444db445d3886d487839c4746906", name: "phase2_sprite"},
		{frame: 420, hash: "e3f9c4dadd6885e8d254d0a752e00432b27e228fee917239fe220c43015d9bb7", name: "phase3_split"},
		{frame: 600, hash: "97ac6c794dfdf97d9eafea3e234f48faaeef4d28d387f30a1ba91f0f0350eafa", name: "phase4_warp"},
	}

	frame := 0
	for _, cp := range checkpoints {
		for frame < cp.frame {
			if err := emu.RunFrame(); err != nil {
				t.Fatalf("RunFrame failed at frame %d: %v", frame, err)
			}
			frame++
		}
		got := framebufferHash(emu.PPU.OutputBuffer[:])
		maybeDumpShowcaseFrame(t, cp.name, emu.PPU.OutputBuffer[:])
		if got != cp.hash {
			t.Fatalf("%s framebuffer hash mismatch: got=%s want=%s", cp.name, got, cp.hash)
		}
	}
}
