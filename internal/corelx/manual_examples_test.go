package corelx

import (
	"os"
	"path/filepath"
	"testing"

	"nitro-core-dx/internal/emulator"
)

// The manual/guide's demo programs are verified here: each one compiles, loads,
// runs on the emulator, and produces the documented visual output. If a demo in
// the guide ever stops matching reality, this test fails. Demos live in
// docs/manual_examples/.

func examplePath(name string) string {
	return filepath.Join("..", "..", "docs", "manual_examples", name)
}

func compileExample(t *testing.T, name string) (*emulator.Emulator, *CompileResult) {
	t.Helper()
	src, err := os.ReadFile(examplePath(name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return compileLoadForTest(t, string(src))
}

// hello.corelx: draws "HELLO NITRO" every frame.
func TestExampleHello(t *testing.T) {
	emu, _ := compileExample(t, "hello.corelx")
	emu.Start()
	emu.SetFrameLimit(false)
	if err := emu.RunFrame(); err != nil {
		t.Fatal(err)
	}
	// Text rendered to the framebuffer this frame.
	buf := emu.GetOutputBuffer()
	nz := 0
	for _, px := range buf {
		if px != 0 {
			nz++
		}
	}
	if nz == 0 {
		t.Error("hello.corelx produced an empty frame; expected visible text")
	}
}

// counter.corelx: A increments a displayed count; pressed fires once per press.
func TestExampleCounter(t *testing.T) {
	emu, result := compileExample(t, "counter.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)

	// Three clean taps of A: 3 frames down, 3 frames up, repeated. Because
	// the counter uses input.pressed (rising edge), each tap adds exactly one,
	// no matter how many frames A stays down.
	for frame := 0; frame < 18; frame++ {
		if (frame/3)%2 == 0 {
			emu.SetInputButtons(0x0010) // A down
		} else {
			emu.SetInputButtons(0x0000) // A up
		}
		emu.RunFrame()
	}
	if got := read16(emu, addrs["count"]); got != 3 {
		t.Errorf("counter after 3 taps of A: want count==3 (pressed = one per press), got %d", got)
	}
}

// floor.corelx: D-pad drives a pseudo-3D matrix floor camera.
func TestExampleFloor(t *testing.T) {
	emu, result := compileExample(t, "floor.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)
	emu.SetInputButtons(0x0001) // UP held
	for i := 0; i < 40; i++ {
		emu.RunFrame()
	}
	// Sustained UP drives the camera to the world's near edge (clamped to 0).
	if camY := read16(emu, addrs["cam_y"]); camY != 0 {
		t.Errorf("floor.corelx: cam_y under sustained UP should clamp to 0, got %d", camY)
	}
	// Projection is the perspective floor, synced to the plane.
	if emu.PPU.MatrixPlanes[0].ProjectionMode != 1 {
		t.Errorf("floor.corelx: ProjectionMode want 1, got %d", emu.PPU.MatrixPlanes[0].ProjectionMode)
	}
	if emu.PPU.MatrixPlanes[0].CameraY != 0 {
		t.Error("floor.corelx: plane CameraY not synced to clamped cam_y")
	}
}
