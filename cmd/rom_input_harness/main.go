package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"nitro-core-dx/internal/emulator"
)

// FrameInput describes the controller state for one frame.
type FrameInput struct {
	Buttons uint16
	Frames  int
}

func runScript(emu *emulator.Emulator, script []FrameInput) error {
	emu.Running = true
	emu.SetFrameLimit(false)

	for _, step := range script {
		for i := 0; i < step.Frames; i++ {
			emu.SetInputButtons(step.Buttons)
			if err := emu.RunFrame(); err != nil {
				return err
			}
		}
	}
	// Clear input at the end so latched state doesn't keep buttons stuck.
	emu.SetInputButtons(0)
	return nil
}

func main() {
	romPath := flag.String("rom", "roms/matrix_floor_billboard_reference.rom", "ROM path to test")
	headingAddr := flag.Uint("heading", 0x0202, "WRAM address of heading index (bank 0)")
	cameraXAddr := flag.Uint("camx", 0x0204, "WRAM address of camera X (bank 0)")
	cameraYAddr := flag.Uint("camy", 0x0206, "WRAM address of camera Y (bank 0)")
	forwardFrames := flag.Int("forward_frames", 30, "Frames to hold UP (move forward)")
	backwardFrames := flag.Int("backward_frames", 0, "Frames to hold DOWN (move backward)")
	leftFrames := flag.Int("left_frames", 15, "Frames to hold LEFT (turn left)")
	rightFrames := flag.Int("right_frames", 0, "Frames to hold RIGHT (turn right)")
	warmupFrames := flag.Int("warmup_frames", 5, "Frames to run with no input before measurements")
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

	// Small delay so logs are readable when run interactively.
	time.Sleep(50 * time.Millisecond)

	// Warmup
	if err := runScript(emu, []FrameInput{{Buttons: 0, Frames: *warmupFrames}}); err != nil {
		fmt.Fprintf(os.Stderr, "warmup RunFrame error: %v\n", err)
		os.Exit(1)
	}

	hAddr := uint16(*headingAddr)
	cxAddr := uint16(*cameraXAddr)
	cyAddr := uint16(*cameraYAddr)

	readHeading := func() uint16 { return emu.Bus.Read16(0, hAddr) }
	readCamX := func() uint16 { return emu.Bus.Read16(0, cxAddr) }
	readCamY := func() uint16 { return emu.Bus.Read16(0, cyAddr) }

	fmt.Printf("ROM input harness\n")
	fmt.Printf("  ROM: %s\n", *romPath)
	fmt.Printf("  Heading @ $%04X, CamX @ $%04X, CamY @ $%04X\n", hAddr, cxAddr, cyAddr)

	baseHeading := readHeading()
	baseCamX := readCamX()
	baseCamY := readCamY()
	fmt.Printf("Baseline: heading=%d cam=(%d,%d)\n", baseHeading, baseCamX, baseCamY)

	// Forward test (UP)
	if *forwardFrames > 0 {
		if err := runScript(emu, []FrameInput{{Buttons: 0x0001, Frames: *forwardFrames}}); err != nil {
			fmt.Fprintf(os.Stderr, "forward RunFrame error: %v\n", err)
			os.Exit(1)
		}
		forwardCamX := readCamX()
		forwardCamY := readCamY()
		fmt.Printf("After holding UP for %d frames: cam=(%d,%d) delta=(%+d,%+d)\n",
			*forwardFrames,
			forwardCamX, forwardCamY,
			int32(forwardCamX)-int32(baseCamX),
			int32(forwardCamY)-int32(baseCamY),
		)
	}

	// Backward test (DOWN)
	if *backwardFrames > 0 {
		if err := runScript(emu, []FrameInput{{Buttons: 0x0002, Frames: *backwardFrames}}); err != nil {
			fmt.Fprintf(os.Stderr, "backward RunFrame error: %v\n", err)
			os.Exit(1)
		}
		backCamX := readCamX()
		backCamY := readCamY()
		fmt.Printf("After holding DOWN for %d frames: cam=(%d,%d) delta=(%+d,%+d)\n",
			*backwardFrames,
			backCamX, backCamY,
			int32(backCamX)-int32(baseCamX),
			int32(backCamY)-int32(baseCamY),
		)
	}

	// Left turn test
	if *leftFrames > 0 {
		if err := runScript(emu, []FrameInput{{Buttons: 0x0004, Frames: *leftFrames}}); err != nil {
			fmt.Fprintf(os.Stderr, "left RunFrame error: %v\n", err)
			os.Exit(1)
		}
		leftHeading := readHeading()
		fmt.Printf("After holding LEFT for %d frames: heading=%d delta=%+d\n",
			*leftFrames, leftHeading, int32(leftHeading)-int32(baseHeading))
	}

	// Right turn test
	if *rightFrames > 0 {
		if err := runScript(emu, []FrameInput{{Buttons: 0x0008, Frames: *rightFrames}}); err != nil {
			fmt.Fprintf(os.Stderr, "right RunFrame error: %v\n", err)
			os.Exit(1)
		}
		rightHeading := readHeading()
		fmt.Printf("After holding RIGHT for %d frames: heading=%d delta=%+d\n",
			*rightFrames, rightHeading, int32(rightHeading)-int32(baseHeading))
	}
}
