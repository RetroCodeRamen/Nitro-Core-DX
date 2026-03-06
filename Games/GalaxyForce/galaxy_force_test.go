package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
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

func saveFrameAsPNG(buf []uint32, width, height int, path string) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := buf[y*width+x]
			img.Set(x, y, color.RGBA{
				R: uint8((c >> 16) & 0xFF),
				G: uint8((c >> 8) & 0xFF),
				B: uint8(c & 0xFF),
				A: 255,
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

func TestGalaxyForceTitleScreen(t *testing.T) {
	srcPath := filepath.Join("main.corelx")
	romData := compileROM(t, srcPath)
	t.Logf("ROM size: %d bytes", len(romData))

	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	emu.Start()
	emu.SetFrameLimit(false)

	// Run 120 frames (~2 seconds) to catch any runtime errors
	for i := 0; i < 120; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("HARDWARE ERROR at frame %d: %v", i, err)
		}
	}

	buf := emu.GetOutputBuffer()
	outPath := filepath.Join("frame_title.png")
	if err := saveFrameAsPNG(buf, 320, 200, outPath); err != nil {
		t.Fatalf("save image: %v", err)
	}
	t.Logf("Saved title screen frame 120 to %s", outPath)

	// Count non-black pixels
	nonBlack := 0
	for _, c := range buf {
		if c != 0x000000 {
			nonBlack++
		}
	}
	t.Logf("Non-black pixels: %d / %d (%.1f%%)", nonBlack, 320*200, float64(nonBlack)/float64(320*200)*100)

	// Dump enabled sprites
	enabledCount := 0
	for i := 0; i < 128; i++ {
		oamAddr := i * 6
		ctrl := emu.PPU.OAM[oamAddr+5]
		if (ctrl & 0x01) != 0 {
			enabledCount++
		}
	}
	t.Logf("Total enabled sprites: %d", enabledCount)

	// CPU state
	t.Logf("CPU: PCBank=%d PCOffset=0x%04X Cycles=%d",
		emu.CPU.State.PCBank, emu.CPU.State.PCOffset, emu.CPU.State.Cycles)
	t.Logf("CPU: R0=%d R1=%d R2=%d R3=%d R4=%d R5=%d R6=%d R7=%d",
		emu.CPU.State.R0, emu.CPU.State.R1, emu.CPU.State.R2, emu.CPU.State.R3,
		emu.CPU.State.R4, emu.CPU.State.R5, emu.CPU.State.R6, emu.CPU.State.R7)
	t.Logf("CPU: SP=0x%04X Flags=0x%02X DBR=%d", emu.CPU.State.SP, emu.CPU.State.Flags, emu.CPU.State.DBR)

	// Text rendering state
	t.Logf("PPU Text: X=%d Y=%d R=%d G=%d B=%d TextCount=%d",
		emu.PPU.TextX, emu.PPU.TextY, emu.PPU.TextR, emu.PPU.TextG, emu.PPU.TextB, emu.PPU.GetTextCount())

	// Check for non-black pixels in several text areas
	for _, area := range [][4]int{
		{112, 30, 208, 38},  // GALAXY FORCE
		{108, 46, 212, 54},  // NITRO CORE DX
		{120, 170, 208, 178}, // PRESS START
		{4, 4, 100, 12},     // HP area
		{216, 4, 320, 12},   // SCORE area
	} {
		found := 0
		for y := area[1]; y < area[3]; y++ {
			for x := area[0]; x < area[2]; x++ {
				if buf[y*320+x] != 0 {
					found++
				}
			}
		}
		t.Logf("Non-black pixels in area (%d,%d)-(%d,%d): %d", area[0], area[1], area[2], area[3], found)
	}

	// Check what game state the game is in by examining WRAM
	// game_state is a local variable, hard to check directly.
	// Instead, check if title sprites are enabled (state 0 indicator)
	dumpOAMSprites(t, emu)

	if nonBlack == 0 {
		t.Error("FAIL: Screen is entirely black - nothing rendered")
	}
}

func TestGalaxyForceExtendedRun(t *testing.T) {
	srcPath := filepath.Join("main.corelx")
	romData := compileROM(t, srcPath)

	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	emu.Start()
	emu.SetFrameLimit(false)

	totalFrames := 3600 // 60 seconds at 60fps
	for i := 0; i < totalFrames; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Logf("CPU at error: PCBank=%d PCOffset=0x%04X Cycles=%d SP=0x%04X",
				emu.CPU.State.PCBank, emu.CPU.State.PCOffset, emu.CPU.State.Cycles, emu.CPU.State.SP)
			t.Logf("CPU regs: R0=%d R1=%d R2=%d R3=%d R4=%d R5=%d R6=%d R7=%d",
				emu.CPU.State.R0, emu.CPU.State.R1, emu.CPU.State.R2, emu.CPU.State.R3,
				emu.CPU.State.R4, emu.CPU.State.R5, emu.CPU.State.R6, emu.CPU.State.R7)
			t.Logf("CPU Flags: Z=%v N=%v C=%v V=%v",
				(emu.CPU.State.Flags>>0)&1 == 1,
				(emu.CPU.State.Flags>>1)&1 == 1,
				(emu.CPU.State.Flags>>2)&1 == 1,
				(emu.CPU.State.Flags>>3)&1 == 1)

			buf := emu.GetOutputBuffer()
			saveFrameAsPNG(buf, 320, 200, "frame_error.png")

			t.Fatalf("HARDWARE ERROR at frame %d: %v", i, err)
		}
	}

	t.Logf("Ran %d frames without errors", totalFrames)
	t.Logf("CPU: PCBank=%d PCOffset=0x%04X Cycles=%d SP=0x%04X",
		emu.CPU.State.PCBank, emu.CPU.State.PCOffset, emu.CPU.State.Cycles, emu.CPU.State.SP)
}

func dumpOAMSprites(t *testing.T, emu *emulator.Emulator) {
	t.Helper()
	for i := 0; i < 128; i++ {
		oamAddr := i * 6
		xLow := emu.PPU.OAM[oamAddr]
		xHigh := emu.PPU.OAM[oamAddr+1]
		y := emu.PPU.OAM[oamAddr+2]
		tile := emu.PPU.OAM[oamAddr+3]
		attr := emu.PPU.OAM[oamAddr+4]
		ctrl := emu.PPU.OAM[oamAddr+5]
		enabled := (ctrl & 0x01) != 0
		if enabled {
			x := int(xLow) | (int(xHigh) << 8)
			if xHigh&0x80 != 0 {
				x |= ^0xFFFF
			}
			palette := attr & 0x0F
			priority := (attr >> 6) & 0x03
			size16 := (ctrl & 0x02) != 0
			sizeStr := "8x8"
			if size16 {
				sizeStr = "16x16"
			}
			t.Logf("  Sprite %3d: X=%4d Y=%3d Tile=%d Pal=%d Pri=%d %s",
				i, x, y, tile, palette, priority, sizeStr)
		}
	}
}

func dumpBGState(t *testing.T, emu *emulator.Emulator) {
	t.Helper()
	t.Logf("  BG0: Enabled=%v ScrollX=%d ScrollY=%d",
		emu.PPU.BG0.Enabled, emu.PPU.BG0.ScrollX, emu.PPU.BG0.ScrollY)
}

func dumpVRAM(t *testing.T, emu *emulator.Emulator, start, count int) {
	t.Helper()
	for row := 0; row < count/16; row++ {
		line := fmt.Sprintf("  %04X:", start+row*16)
		for col := 0; col < 16; col++ {
			line += fmt.Sprintf(" %02X", emu.PPU.VRAM[start+row*16+col])
		}
		t.Log(line)
	}
}
