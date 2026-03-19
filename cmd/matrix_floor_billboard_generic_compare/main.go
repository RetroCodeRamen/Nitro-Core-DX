package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
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
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			if i == 0 {
				return "/"
			}
			return path[:i]
		}
	}
	return "."
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

func runFrames(emu *emulator.Emulator, buttons uint16, frames int) error {
	for i := 0; i < frames; i++ {
		emu.SetInputButtons(buttons)
		if err := emu.RunFrame(); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	romPath := flag.String("rom", "roms/matrix_floor_billboard_generic.rom", "ROM to run")
	outBefore := flag.String("before", ".tmp_matrixfloor_generic_compare/frame_before.png", "output PNG path for baseline frame")
	outAfter := flag.String("after", ".tmp_matrixfloor_generic_compare/frame_after.png", "output PNG path for frame after input")
	warmupFrames := flag.Int("warmup_frames", 5, "frames to run before baseline capture")
	baselineFrames := flag.Int("baseline_frames", 10, "extra frames with no input before baseline capture")
	inputFrames := flag.Int("input_frames", 20, "frames to hold input before second capture")
	buttons := flag.Uint("buttons", 0x0001, "controller buttons mask for second phase (default: UP)")
	flag.Parse()

	data, err := os.ReadFile(*romPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read ROM: %v\n", err)
		os.Exit(1)
	}

	emu := emulator.NewEmulator()
	if err := emu.LoadROM(data); err != nil {
		fmt.Fprintf(os.Stderr, "load ROM: %v\n", err)
		os.Exit(1)
	}
	emu.Running = true
	emu.SetFrameLimit(false)

	if err := runFrames(emu, 0, *warmupFrames); err != nil {
		fmt.Fprintf(os.Stderr, "warmup RunFrame error: %v\n", err)
		os.Exit(1)
	}
	if err := runFrames(emu, 0, *baselineFrames); err != nil {
		fmt.Fprintf(os.Stderr, "baseline RunFrame error: %v\n", err)
		os.Exit(1)
	}

	bufBefore := emu.GetOutputBuffer()
	hashBefore := framebufferHash(bufBefore)

	// Snapshot camera before input.
	wramCamXBefore := emu.Bus.Read16(0, 0x0204)
	wramCamYBefore := emu.Bus.Read16(0, 0x0206)
	wramHeadingBefore := emu.Bus.Read16(0, 0x0202)
	plane0 := &emu.PPU.MatrixPlanes[0]
	plane1 := &emu.PPU.MatrixPlanes[1]

	if err := writePNG(*outBefore, bufBefore); err != nil {
		fmt.Fprintf(os.Stderr, "write before PNG: %v\n", err)
		os.Exit(1)
	}

	if err := runFrames(emu, uint16(*buttons), *inputFrames); err != nil {
		fmt.Fprintf(os.Stderr, "input RunFrame error: %v\n", err)
		os.Exit(1)
	}

	bufAfter := emu.GetOutputBuffer()
	hashAfter := framebufferHash(bufAfter)

	wramCamXAfter := emu.Bus.Read16(0, 0x0204)
	wramCamYAfter := emu.Bus.Read16(0, 0x0206)
	wramHeadingAfter := emu.Bus.Read16(0, 0x0202)

	if err := writePNG(*outAfter, bufAfter); err != nil {
		fmt.Fprintf(os.Stderr, "write after PNG: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Matrix floor+billboard (generic) compare capture\n")
	fmt.Printf("  ROM: %s\n", *romPath)
	fmt.Printf("  Baseline frame: %s\n", *outBefore)
	fmt.Printf("  After-input frame: %s\n", *outAfter)
	fmt.Printf("  Buttons mask: 0x%04X, input_frames=%d\n\n", uint16(*buttons), *inputFrames)

	fmt.Printf("WRAM state:\n")
	fmt.Printf("  Before: heading=%d cam=(%d,%d)\n", wramHeadingBefore, int16(wramCamXBefore), int16(wramCamYBefore))
	fmt.Printf("  After : heading=%d cam=(%d,%d)\n\n", wramHeadingAfter, int16(wramCamXAfter), int16(wramCamYAfter))

	fmt.Printf("Plane state (Camera/Heading):\n")
	fmt.Printf("  Plane0: cam=(%d,%d) heading=(%d,%d)\n", int16(plane0.CameraX), int16(plane0.CameraY), int16(plane0.HeadingX), int16(plane0.HeadingY))
	fmt.Printf("  Plane1: cam=(%d,%d) heading=(%d,%d)\n\n", int16(plane1.CameraX), int16(plane1.CameraY), int16(plane1.HeadingX), int16(plane1.HeadingY))

	fmt.Printf("Plane1 billboard projection regs:\n")
	fmt.Printf("  Enabled=%v Size=%d SourceMode=%d Transparent0=%v TwoSided=%v\n",
		plane1.Enabled, plane1.Size, plane1.SourceMode, plane1.Transparent0, plane1.TwoSided)
	fmt.Printf("  ProjectionMode=%d Horizon=%d\n", plane1.ProjectionMode, plane1.Horizon)
	fmt.Printf("  BaseDistance=0x%04X FocalLength=0x%04X WidthScale=0x%04X\n",
		plane1.BaseDistance, plane1.FocalLength, plane1.WidthScale)
	fmt.Printf("  Origin=(%d,%d) Facing=(0x%04X,0x%04X) HeightScale=0x%04X\n\n",
		int16(plane1.OriginX), int16(plane1.OriginY), uint16(plane1.FacingX), uint16(plane1.FacingY), plane1.HeightScale)

	fmt.Printf("Framebuffer SHA-256:\n")
	fmt.Printf("  Before: %s\n", hashBefore)
	fmt.Printf("  After : %s\n", hashAfter)
	if hashBefore == hashAfter {
		fmt.Printf("  NOTE: Framebuffer hash did NOT change (visual output is identical).\n")
	} else {
		fmt.Printf("  OK: Framebuffer hash changed (visual output differs).\n")
	}
}
