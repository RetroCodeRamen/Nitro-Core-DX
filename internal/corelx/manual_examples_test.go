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
	// The floor actually renders pixels (not a blank screen).
	buf := emu.GetOutputBuffer()
	floorPx := 0
	for y := 120; y < 180; y++ {
		for x := 0; x < 320; x++ {
			if buf[y*320+x] != 0 {
				floorPx++
			}
		}
	}
	if floorPx < 1000 {
		t.Errorf("floor.corelx: floor not rendering (%d pixels in floor region)", floorPx)
	}
}

// sprite.corelx: Sprite()/OAM workflow, D-pad moves an 8x8 box, clamped to
// stay fully on screen.
func TestExampleSprite(t *testing.T) {
	emu, result := compileExample(t, "sprite.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)

	emu.SetInputButtons(0x0008) // RIGHT held
	for i := 0; i < 200; i++ {
		emu.RunFrame()
	}
	if x := read16(emu, addrs["box_x"]); x != 312 {
		t.Errorf("sprite.corelx: box_x under sustained RIGHT should clamp to 312, got %d", x)
	}
	oamCtrl := emu.PPU.OAM[5]
	if oamCtrl&0x01 == 0 {
		t.Error("sprite.corelx: sprite 0 should be enabled in OAM after oam.write/flush")
	}
}

// structs.corelx: a user-defined struct (not Sprite/Vec2) is a reference
// type -- a function's edits to a passed-in struct are visible to the caller.
func TestExampleStructs(t *testing.T) {
	emu, result := compileExample(t, "structs.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)
	emu.RunFrame()

	// hero.lives starts at 3; two damage(hero) calls before the main loop
	// should leave 1 -- and because Player is a reference type, those calls
	// mutated the same hero the main loop reads back into hero_lives.
	if lives := read16(emu, addrs["hero_lives"]); lives != 1 {
		t.Errorf("structs.corelx: hero_lives after two damage() calls: want 1, got %d", lives)
	}
}

// break_continue.corelx: continue skips just one iteration (loop variable
// still advances), break stops the loop outright.
func TestExampleBreakContinue(t *testing.T) {
	emu, result := compileExample(t, "break_continue.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)
	emu.RunFrame()

	if total := read16(emu, addrs["total"]); total != 23 {
		t.Errorf("break_continue.corelx: total want 23 (0+1+2+3+4+6+7, skipping 5, stopping at 8), got %d", total)
	}
}

// anim_module.corelx: the anim module's frame_index() advances one frame
// every N ticks and wraps. Verified with an explicit ModulesPath pointing at
// the real modules/ folder (docs/manual_examples/ has no modules/ sibling of
// its own -- module resolution otherwise expects one next to the source).
func TestExampleAnimModule(t *testing.T) {
	src, err := os.ReadFile(examplePath("anim_module.corelx"))
	if err != nil {
		t.Fatalf("read anim_module.corelx: %v", err)
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.corelx")
	if err := os.WriteFile(srcPath, src, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	romPath := filepath.Join(dir, "main.rom")
	modulesDir := filepath.Join("..", "..", "modules")
	result, err := CompileProject(srcPath, &CompileOptions{OutputPath: romPath, ModulesPath: modulesDir})
	if err != nil {
		t.Fatalf("compile anim_module.corelx: %v", err)
	}
	romData, err := os.ReadFile(romPath)
	if err != nil {
		t.Fatalf("read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)

	// frame_index(4, 8) should still be 0 just before the 8th tick, then 1
	// starting on the 8th.
	for i := 0; i < 7; i++ {
		emu.RunFrame()
	}
	oamTileBefore := emu.PPU.OAM[3]
	emu.RunFrame() // 8th tick
	oamTileAfter := emu.PPU.OAM[3]
	if oamTileBefore == oamTileAfter {
		t.Errorf("anim_module.corelx: expected the sprite's tile to change on the 8th tick (frame_index advancing), stayed at %d", oamTileBefore)
	}
	_ = addrs
}
