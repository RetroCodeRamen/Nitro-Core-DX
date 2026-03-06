package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"nitro-core-dx/internal/corelx"
	"nitro-core-dx/internal/emulator"
)

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

// Expected 2x2 grid: grid_x=152, grid_y=92. Each tile 8x8.
// Sample center of each quadrant (offset 4,4 within each 8x8).
const (
	gridX   = 152
	gridY   = 92
	sampleO = 4 // offset from tile top-left to sample
)

func TestSpriteProbeFourTiles_CompareOutput(t *testing.T) {
	romData := compileROM(t, filepath.Join("main.corelx"))

	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	emu.Start()
	emu.SetFrameLimit(false)

	// Run a few frames so display is stable
	for i := 0; i < 5; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame %d: %v", i, err)
		}
	}

	buf := emu.GetOutputBuffer()
	if len(buf) < 320*200 {
		t.Fatalf("framebuffer too small: %d", len(buf))
	}

	// Sample center of each 8x8 tile. Expected palette 1 colors (RGB888):
	// 1=red, 2=green, 3=blue, 4=yellow
	quads := []struct {
		name           string
		x, y           int
		expectR, expectG, expectB uint32 // dominant channel(s); allow some tolerance
	}{
		{"top-left (red)", gridX + sampleO, gridY + sampleO, 255, 0, 0},
		{"top-right (green)", gridX + 8 + sampleO, gridY + sampleO, 0, 255, 0},
		{"bottom-left (blue)", gridX + sampleO, gridY + 8 + sampleO, 0, 0, 255},
		{"bottom-right (yellow)", gridX + 8 + sampleO, gridY + 8 + sampleO, 255, 255, 0},
	}

	var failures []string
	for _, q := range quads {
		if q.x >= 320 || q.y >= 200 {
			continue
		}
		c := buf[q.y*320+q.x]
		r := (c >> 16) & 0xFF
		g := (c >> 8) & 0xFF
		b := c & 0xFF

		// Check dominant channel (tolerance for CGRAM conversion)
		dominantRed := r > 200 && g < 100 && b < 100
		dominantGreen := g > 200 && r < 100 && b < 100
		dominantBlue := b > 200 && r < 100 && g < 100
		dominantYellow := r > 200 && g > 200 && b < 100

		ok := false
		switch q.name {
		case "top-left (red)":
			ok = dominantRed
		case "top-right (green)":
			ok = dominantGreen
		case "bottom-left (blue)":
			ok = dominantBlue
		case "bottom-right (yellow)":
			ok = dominantYellow
		}
		if !ok {
			failures = append(failures, fmt.Sprintf("%s at (%d,%d): got R=%d G=%d B=%d (0x%06X)", q.name, q.x, q.y, r, g, b, c&0xFFFFFF))
		}
	}

	// Always write actual ASCII and a small expected-vs-actual report for inspection
	testdata := filepath.Join("testdata")
	_ = os.MkdirAll(testdata, 0755)
	actualPath := filepath.Join(testdata, "sprite_probe_actual.txt")
	expectedPath := filepath.Join(testdata, "sprite_probe_expected.txt")

	// Dump ASCII (80x50) of actual buffer
	lines := bufferToASCII(buf, 320, 200, 80, 50)
	if f, err := os.Create(actualPath); err == nil {
		for _, line := range lines {
			fmt.Fprintln(f, line)
		}
		f.Close()
		t.Logf("Actual frame written to %s", actualPath)
	}

	// Write expected description
	expectedDesc := `Expected 2x2 grid at (152,92), each cell 8x8:
  top-left: RED    (R high, G,B low)
  top-right: GREEN (G high, R,B low)
  bottom-left: BLUE (B high, R,G low)
  bottom-right: YELLOW (R,G high, B low)
`
	if f, err := os.Create(expectedPath); err == nil {
		f.WriteString(expectedDesc)
		f.WriteString("\nActual quadrant center colors (from test run):\n")
		for _, q := range quads {
			c := buf[q.y*320+q.x]
			r := (c >> 16) & 0xFF
			g := (c >> 8) & 0xFF
			b := c & 0xFF
			fmt.Fprintf(f, "  %s (%d,%d): R=%d G=%d B=%d 0x%06X\n", q.name, q.x, q.y, r, g, b, c&0xFFFFFF)
		}
		f.Close()
		t.Logf("Expected vs actual summary: %s", expectedPath)
	}

	if len(failures) > 0 {
		t.Errorf("quadrant color mismatch (see %s and %s):", actualPath, expectedPath)
		for _, s := range failures {
			t.Error("  ", s)
		}
	}
}

// TestShip_Visible compiles ship.corelx, runs a few frames, and checks that the ship
// region is not all black (so we can verify compiler/PPU fixes). Writes ASCII dump to
// testdata/ship_actual.txt for inspection.
func TestShip_Visible(t *testing.T) {
	romData := compileROM(t, filepath.Join("ship.corelx"))

	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	emu.Start()
	emu.SetFrameLimit(false)

	for i := 0; i < 8; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame %d: %v", i, err)
		}
	}

	buf := emu.GetOutputBuffer()
	if len(buf) < 320*200 {
		t.Fatalf("framebuffer too small: %d", len(buf))
	}

	// Ship is 16x16 at (152, 92). Sample region for any non-black pixel.
	const shipX, shipY, shipW, shipH = 152, 92, 16, 16
	var maxBrightness uint32
	for y := shipY; y < shipY+shipH && y < 200; y++ {
		for x := shipX; x < shipX+shipW && x < 320; x++ {
			c := buf[y*320+x]
			r := (c >> 16) & 0xFF
			g := (c >> 8) & 0xFF
			b := c & 0xFF
			bright := (r + g + b) / 3
			if bright > maxBrightness {
				maxBrightness = bright
			}
		}
	}

	// Always write ASCII dump for inspection
	testdata := filepath.Join("testdata")
	_ = os.MkdirAll(testdata, 0755)
	actualPath := filepath.Join(testdata, "ship_actual.txt")
	lines := bufferToASCII(buf, 320, 200, 80, 50)
	if f, err := os.Create(actualPath); err == nil {
		for _, line := range lines {
			fmt.Fprintln(f, line)
		}
		f.Close()
		t.Logf("Ship frame ASCII written to %s (ship region: max brightness=%d)", actualPath, maxBrightness)
	}

	if maxBrightness < 32 {
		t.Errorf("ship region appears black (max brightness=%d); see %s. Possible loader stride or PPU tile index mismatch.", maxBrightness, actualPath)
	}
}

// bufferToASCII downsamples 320x200 to cols x rows ASCII by brightness
func bufferToASCII(buf []uint32, width, height, cols, rows int) []string {
	if len(buf) < width*height || cols <= 0 || rows <= 0 {
		return nil
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
					sum += (r + g + b) / 3
					count++
				}
			}
			ch := byte(' ')
			if count > 0 {
				avg := sum / count
				switch {
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
