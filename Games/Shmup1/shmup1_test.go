package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"nitro-core-dx/internal/corelx"
	"nitro-core-dx/internal/emulator"
)

// compileROM compiles a CoreLX source file to ROM bytes using the shared game-test helper pattern.
func compileROM(t *testing.T, srcPath string) []byte {
	t.Helper()
	result, err := corelx.CompileSource(
		readFile(t, srcPath),
		srcPath,
		&corelx.CompileOptions{EmitROMBytes: true},
	)
	if err != nil {
		for _, d := range result.Diagnostics {
			t.Logf("  %s: %s", d.Stage, d.Message)
		}
		t.Fatalf("compilation failed: %v", err)
	}
	return result.ROMBytes
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// bufferToASCII downsamples the 320x200 framebuffer to cols x rows text and maps brightness to ASCII.
func bufferToASCII(buf []uint32, width, height, cols, rows int) []string {
	if len(buf) < width*height {
		return []string{"<buffer too small>"}
	}
	if cols <= 0 || rows <= 0 {
		return []string{"<invalid cols/rows>"}
	}
	cellW := width / cols
	if cellW <= 0 {
		cellW = 1
	}
	cellH := height / rows
	if cellH <= 0 {
		cellH = 1
	}

	lines := make([]string, 0, rows)
	for ry := 0; ry < rows; ry++ {
		y0 := ry * cellH
		if y0 >= height {
			y0 = height - 1
		}
		y1 := y0 + cellH
		if y1 > height {
			y1 = height
		}
		line := make([]byte, cols)
		for rx := 0; rx < cols; rx++ {
			x0 := rx * cellW
			if x0 >= width {
				x0 = width - 1
			}
			x1 := x0 + cellW
			if x1 > width {
				x1 = width
			}
			var sum uint32
			var count uint32
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					c := buf[y*width+x]
					r := (c >> 16) & 0xFF
					g := (c >> 8) & 0xFF
					b := c & 0xFF
					// Simple brightness: average of RGB
					bright := (r + g + b) / 3
					sum += bright
					count++
				}
			}
			var ch byte = ' '
			if count > 0 {
				avg := sum / count
				switch {
				case avg < 16:
					ch = ' '
				case avg < 64:
					ch = '.'
				case avg < 128:
					ch = '*'
				case avg < 192:
					ch = 'o'
				default:
					ch = '#'
				}
			}
			line[rx] = ch
		}
		lines = append(lines, string(line))
	}
	return lines
}

// writeASCIIFrame saves an ASCII representation of the current framebuffer to testdata.
func writeASCIIFrame(t *testing.T, buf []uint32, width, height int, name string) {
	t.Helper()
	lines := bufferToASCII(buf, width, height, 80, 50)

	testdata := filepath.Join("testdata")
	if err := os.MkdirAll(testdata, 0o755); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}
	path := filepath.Join(testdata, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()

	for _, line := range lines {
		if _, err := fmt.Fprintln(f, line); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	t.Logf("Wrote ASCII frame to %s", path)
}

// TestShmup1TitleASCII renders a single frame of the Shmup1 title screen to ASCII
// so you can inspect the layout and patterns in text form.
func TestShmup1TitleASCII(t *testing.T) {
	srcPath := filepath.Join("main.corelx")
	romData := compileROM(t, srcPath)

	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	emu.Start()
	emu.SetFrameLimit(false)

	// Run a few frames to let the title screen and starfield fully draw.
	const frames = 5
	for i := 0; i < frames; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame %d: %v", i, err)
		}
	}

	buf := emu.GetOutputBuffer()
	writeASCIIFrame(t, buf, 320, 200, "shmup1_title_frame.txt")
}
