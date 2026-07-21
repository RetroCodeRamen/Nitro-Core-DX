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
		// Hashes updated 2026-07-20: the entry function now redirects the
		// IRQ/NMI vectors away from the ROM entry point (see
		// CodeGenerator.emitIRQVectorFix) instead of leaving them aliased to
		// it. The PPU fires an IRQ unconditionally on every VBlank; with the
		// old default vectors, the very first VBlank silently re-entered
		// Start() from the top once (interrupts then stay masked), so these
		// hashes previously captured a ROM whose entry-function setup ran
		// twice before settling. The scene composition is unchanged (still
		// matches each phase's documented expected result); only the exact
		// framebuffer bytes shifted along with the corrected, single-run
		// boot sequence.
		{frame: 120, hash: "7a6feccbd8ce2ccd1622b40a718e4e72054047f7d0d13f928f64aac273b054bf", name: "phase1_static"},
		{frame: 240, hash: "d6c331c270915748091b3baa2994acd8cf2deef58190a7e4dec78e7042d41b58", name: "phase2_sprite"},
		{frame: 420, hash: "b020c4ff5defffe938c27a3fd54a225f10742d36981f7c2c611c8d049cd8e6c7", name: "phase3_split"},
		{frame: 600, hash: "ce0c848072a51e23c7010a8cceda8bb704c851c79e95fe84328568abbb9598d6", name: "phase4_warp"},
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
