//go:build testrom_tools
// +build testrom_tools

package main

import (
	"os"
	"path/filepath"
	"testing"

	"nitro-core-dx/internal/emulator"
)

func runFrames(t *testing.T, emu *emulator.Emulator, buttons uint16, frames int) {
	t.Helper()
	for i := 0; i < frames; i++ {
		emu.SetInputButtons(buttons)
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed at frame %d with buttons 0x%04X: %v", i, buttons, err)
		}
	}
}

func TestNitroPackInDemoSceneFlow(t *testing.T) {
	floorImg, err := loadPNG(filepath.Join(".", "park.png"))
	if err != nil {
		t.Fatalf("load floor image: %v", err)
	}
	billboardImg, err := loadPNG(filepath.Join(".", "building.png"))
	if err != nil {
		t.Fatalf("load billboard image: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "nitro_pack_in_demo.rom")
	if err := buildNitroPackInDemoROM(floorImg, billboardImg, outPath); err != nil {
		t.Fatalf("build ROM: %v", err)
	}

	romData, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read built ROM: %v", err)
	}

	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("load ROM into emulator: %v", err)
	}
	emu.Running = true
	emu.SetFrameLimit(false)

	const (
		sceneAddr   = 0x0202
		cameraYAddr = 0x020E

		sceneTitle     = 0
		sceneOverworld = 1
		sceneInterior  = 2

		btnUp    = 1 << 0
		btnA     = 1 << 4
		btnStart = 1 << 10
	)

	if got := emu.Bus.Read16(0, sceneAddr); got != sceneTitle {
		t.Fatalf("initial scene: got %d, want %d", got, sceneTitle)
	}

	runFrames(t, emu, 0, 20)
	runFrames(t, emu, btnStart, 1)
	runFrames(t, emu, 0, 2)

	if got := emu.Bus.Read16(0, sceneAddr); got != sceneOverworld {
		t.Fatalf("scene after pressing START: got %d, want %d", got, sceneOverworld)
	}
	if got := int16(emu.PPU.MatrixPlanes[0].HeadingX); got != 0 {
		t.Fatalf("plane 0 heading X after entering overworld: got %d, want 0", got)
	}
	if got := int16(emu.PPU.MatrixPlanes[0].HeadingY); got != -256 {
		t.Fatalf("plane 0 heading Y after entering overworld: got %d, want -256", got)
	}
	if got := int16(emu.PPU.MatrixPlanes[0].CameraX); got != 512 {
		t.Fatalf("plane 0 camera X after entering overworld: got %d, want 512", got)
	}
	if got := int16(emu.PPU.MatrixPlanes[0].CameraY); got != 768 {
		t.Fatalf("plane 0 camera Y after entering overworld: got %d, want 768", got)
	}
	if emu.PPU.MatrixPlanes[1].Horizon != emu.PPU.MatrixPlanes[0].Horizon {
		t.Fatalf("plane 1 horizon should match floor horizon: got %d want %d", emu.PPU.MatrixPlanes[1].Horizon, emu.PPU.MatrixPlanes[0].Horizon)
	}
	if emu.PPU.MatrixPlanes[1].BaseDistance != emu.PPU.MatrixPlanes[0].BaseDistance {
		t.Fatalf("plane 1 base distance should match floor base distance: got 0x%04X want 0x%04X", emu.PPU.MatrixPlanes[1].BaseDistance, emu.PPU.MatrixPlanes[0].BaseDistance)
	}
	if emu.PPU.MatrixPlanes[1].FocalLength != emu.PPU.MatrixPlanes[0].FocalLength {
		t.Fatalf("plane 1 focal length should match floor focal length: got 0x%04X want 0x%04X", emu.PPU.MatrixPlanes[1].FocalLength, emu.PPU.MatrixPlanes[0].FocalLength)
	}

	runFrames(t, emu, btnUp, 20)

	if got := emu.Bus.Read16(0, cameraYAddr); got != 688 {
		t.Fatalf("camera Y after walking toward the door: got %d, want 688", got)
	}

	runFrames(t, emu, btnA, 1)
	runFrames(t, emu, 0, 2)

	if got := emu.Bus.Read16(0, sceneAddr); got != sceneInterior {
		t.Fatalf("scene after door interaction: got %d, want %d", got, sceneInterior)
	}

	runFrames(t, emu, btnA, 1)
	runFrames(t, emu, 0, 2)

	if got := emu.Bus.Read16(0, sceneAddr); got != sceneOverworld {
		t.Fatalf("scene after leaving interior stub: got %d, want %d", got, sceneOverworld)
	}
}

func TestNitroPackInDemoTurningChangesMovementVector(t *testing.T) {
	floorImg, err := loadPNG(filepath.Join(".", "park.png"))
	if err != nil {
		t.Fatalf("load floor image: %v", err)
	}
	billboardImg, err := loadPNG(filepath.Join(".", "building.png"))
	if err != nil {
		t.Fatalf("load billboard image: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "nitro_pack_in_demo.rom")
	if err := buildNitroPackInDemoROM(floorImg, billboardImg, outPath); err != nil {
		t.Fatalf("build ROM: %v", err)
	}

	romData, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read built ROM: %v", err)
	}

	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("load ROM into emulator: %v", err)
	}
	emu.Running = true
	emu.SetFrameLimit(false)

	const (
		headingAddr = 0x020A
		cameraXAddr = 0x020C
		cameraYAddr = 0x020E

		btnUp    = 1 << 0
		btnLeft  = 1 << 2
		btnStart = 1 << 10
	)

	runFrames(t, emu, 0, 20)
	runFrames(t, emu, btnStart, 1)
	runFrames(t, emu, 0, 2)

	startX := emu.Bus.Read16(0, cameraXAddr)
	startY := emu.Bus.Read16(0, cameraYAddr)
	startHeading := emu.Bus.Read16(0, headingAddr)

	runFrames(t, emu, btnLeft, 12)
	turnedHeading := emu.Bus.Read16(0, headingAddr)
	if turnedHeading == startHeading {
		t.Fatalf("heading did not change after turning: got %d", turnedHeading)
	}

	runFrames(t, emu, btnUp, 12)
	endX := emu.Bus.Read16(0, cameraXAddr)
	endY := emu.Bus.Read16(0, cameraYAddr)

	if endX == startX {
		t.Fatalf("camera X did not change after turning then moving: start=%d end=%d", startX, endX)
	}
	if endY == startY {
		t.Fatalf("camera Y did not change after turning then moving: start=%d end=%d", startY, endY)
	}
}

func TestNitroPackInDemoWorldBoundsClamp(t *testing.T) {
	floorImg, err := loadPNG(filepath.Join(".", "park.png"))
	if err != nil {
		t.Fatalf("load floor image: %v", err)
	}
	billboardImg, err := loadPNG(filepath.Join(".", "building.png"))
	if err != nil {
		t.Fatalf("load billboard image: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "nitro_pack_in_demo.rom")
	if err := buildNitroPackInDemoROM(floorImg, billboardImg, outPath); err != nil {
		t.Fatalf("build ROM: %v", err)
	}

	romData, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read built ROM: %v", err)
	}

	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("load ROM into emulator: %v", err)
	}
	emu.Running = true
	emu.SetFrameLimit(false)

	const (
		cameraXAddr = 0x020C
		btnUp       = 1 << 0
		btnLeft     = 1 << 2
		btnStart    = 1 << 10
		worldMinX   = 0
	)

	runFrames(t, emu, 0, 20)
	runFrames(t, emu, btnStart, 1)
	runFrames(t, emu, 0, 2)

	runFrames(t, emu, btnLeft, 16)
	runFrames(t, emu, 0, 2)
	runFrames(t, emu, btnUp, 240)
	if got := emu.Bus.Read16(0, cameraXAddr); got != worldMinX {
		t.Fatalf("camera X after walking into left edge: got %d, want %d", got, worldMinX)
	}

	runFrames(t, emu, btnUp, 20)
	if got := emu.Bus.Read16(0, cameraXAddr); got != worldMinX {
		t.Fatalf("camera X after pushing against left edge: got %d, want %d", got, worldMinX)
	}
}
