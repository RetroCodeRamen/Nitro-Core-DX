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
	romPath := flag.String("rom", "roms/matrix_floor_billboard_reference.rom", "ROM to run")
	outBefore := flag.String("before", ".tmp_matrixfloor_compare/frame_before.png", "output PNG path for baseline frame")
	outAfter := flag.String("after", ".tmp_matrixfloor_compare/frame_after.png", "output PNG path for frame after input")
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

	// Warmup to let ROM finish init and reach main loop.
	if err := runFrames(emu, 0, *warmupFrames); err != nil {
		fmt.Fprintf(os.Stderr, "warmup RunFrame error: %v\n", err)
		os.Exit(1)
	}

	// Baseline: run a few frames with no input, then capture.
	if err := runFrames(emu, 0, *baselineFrames); err != nil {
		fmt.Fprintf(os.Stderr, "baseline RunFrame error: %v\n", err)
		os.Exit(1)
	}

	bufBefore := emu.GetOutputBuffer()
	hashBefore := framebufferHash(bufBefore)

	// Snapshot WRAM + plane 1 state before input.
	wramCamXBefore := emu.Bus.Read16(0, 0x0204)
	wramCamYBefore := emu.Bus.Read16(0, 0x0206)
	wramHeadingBefore := emu.Bus.Read16(0, 0x0202)
	plane1 := &emu.PPU.MatrixPlanes[1]
	planeCamXBefore := int16(plane1.CameraX)
	planeCamYBefore := int16(plane1.CameraY)
	planeHeadingXBefore := int16(plane1.HeadingX)
	planeHeadingYBefore := int16(plane1.HeadingY)

	if err := writePNG(*outBefore, bufBefore); err != nil {
		fmt.Fprintf(os.Stderr, "write baseline PNG: %v\n", err)
		os.Exit(1)
	}

	// Apply input for a number of frames (e.g. hold UP).
	if err := runFrames(emu, uint16(*buttons), *inputFrames); err != nil {
		fmt.Fprintf(os.Stderr, "input RunFrame error: %v\n", err)
		os.Exit(1)
	}
	// Clear input at the end.
	emu.SetInputButtons(0)

	bufAfter := emu.GetOutputBuffer()
	hashAfter := framebufferHash(bufAfter)

	// Snapshot WRAM + plane 1 state after input.
	wramCamXAfter := emu.Bus.Read16(0, 0x0204)
	wramCamYAfter := emu.Bus.Read16(0, 0x0206)
	wramHeadingAfter := emu.Bus.Read16(0, 0x0202)
	planeCamXAfter := int16(plane1.CameraX)
	planeCamYAfter := int16(plane1.CameraY)
	planeHeadingXAfter := int16(plane1.HeadingX)
	planeHeadingYAfter := int16(plane1.HeadingY)

	if err := writePNG(*outAfter, bufAfter); err != nil {
		fmt.Fprintf(os.Stderr, "write input PNG: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Matrix floor+billboard compare capture\n")
	fmt.Printf("  ROM: %s\n", *romPath)
	fmt.Printf("  Baseline frame: %s\n", *outBefore)
	fmt.Printf("  After-input frame: %s\n", *outAfter)
	fmt.Printf("  Buttons mask: 0x%04X, input_frames=%d\n", uint16(*buttons), *inputFrames)
	fmt.Printf("\n")
	fmt.Printf("WRAM state:\n")
	fmt.Printf("  Before: heading=%d cam=(%d,%d)\n", wramHeadingBefore, wramCamXBefore, wramCamYBefore)
	fmt.Printf("  After : heading=%d cam=(%d,%d)\n", wramHeadingAfter, wramCamXAfter, wramCamYAfter)
	fmt.Printf("\n")
	fmt.Printf("Plane 1 state (Camera/Heading):\n")
	fmt.Printf("  Before: cam=(%d,%d) heading=(%d,%d)\n", planeCamXBefore, planeCamYBefore, planeHeadingXBefore, planeHeadingYBefore)
	fmt.Printf("  After : cam=(%d,%d) heading=(%d,%d)\n", planeCamXAfter, planeCamYAfter, planeHeadingXAfter, planeHeadingYAfter)
	fmt.Printf("\n")
	fmt.Printf("Framebuffer SHA-256:\n")
	fmt.Printf("  Before: %s\n", hashBefore)
	fmt.Printf("  After : %s\n", hashAfter)
	if hashBefore == hashAfter {
		fmt.Printf("  NOTE: Framebuffer hash did NOT change (visual output is identical).\n")
	} else {
		fmt.Printf("  OK: Framebuffer hash changed (visual output differs).\n")
	}
}
