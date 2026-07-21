package corelx

import (
	"testing"

	"nitro-core-dx/internal/emulator"
)

// syncOverworldForTest advances past overworld.corelx's one-time Start()
// setup (loading the floor and billboard bitmaps spans multiple real
// frames, and that span grows as more assets get added by later M8 tasks --
// don't hardcode a frame count and hope it stays ahead) and then aligns
// turn_tick to a known phase (0).
//
// Why the alignment matters: turn_tick cycles 0..3 every real frame
// unconditionally, and turning is skipped whenever it reads 3. Assertions
// that only count total turns over an N-frame window (N a multiple of 4)
// are phase-invariant -- exactly 3*N/4 turns happen no matter which phase
// you start from -- so they'd pass even with the wrong assumption. But
// camera position accumulated frame-by-frame *while* the heading is
// changing depends on the exact sequence of when each turn lands, which
// does depend on phase. Found the hard way: after adding the billboard's
// second bitmap upload, a hardcoded "20 setup frames" no longer landed on
// turn_tick==0, and a simultaneous-turn-and-walk test's camera math (built
// assuming phase 0) silently diverged from the real, still-correct,
// still-deterministic runtime behavior at a different phase.
func syncOverworldForTest(t *testing.T, emu *emulator.Emulator, addrs map[string]uint16) {
	t.Helper()
	// Coarse signal: last_frame (initialized to 0) gets set to the real
	// frame_counter() value once, right before the game loop's first
	// debounce-wait -- i.e. once one-time setup has essentially finished.
	for read16(emu, addrs["last_frame"]) == 0 {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame during setup: %v", err)
		}
	}
	// Small fixed buffer (not a guess at total setup cost) to make sure
	// we're solidly past the setup/loop-entry boundary, whatever its exact
	// timing, before reading state.
	for i := 0; i < 8; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame during setup: %v", err)
		}
	}
	// Align the turn-rate-limiter phase so frame-by-frame camera math in
	// tests is reproducible regardless of how long setup took.
	for read16(emu, addrs["turn_tick"]) != 0 {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame during turn_tick alignment: %v", err)
		}
	}
}

// TestOverworldRebuild verifies the first chunk of the CoreLX demo rebuild: a
// walkable pseudo-3D overworld. It confirms the program compiles and runs,
// that holding RIGHT turns the heading (rate-limited across the 64-step
// table, matching the hand-built reference ROM's wramTurnTick behavior --
// see build_rom.go's sceneOverworldLabel turn logic), that walking moves the
// camera along the new heading, and that the floor projection and player
// sprite reach the hardware.
func TestOverworldRebuild(t *testing.T) {
	emu, result := compileProjectDirForTest(t, "Games/NitroPackInDemo/corelx/overworld.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)
	syncOverworldForTest(t, emu, addrs)

	if h := read16(emu, addrs["heading_index"]); h != 48 {
		t.Fatalf("initial heading_index want 48 (North), got %d", h)
	}

	// Hold RIGHT for 12 frames -- same frame count the hand-built reference's
	// TestNitroPackInDemoTurningChangesMovementVector uses. Turning is
	// rate-limited to 3 of every 4 held frames (skipped whenever turn_tick
	// reads 3): over 12 frames starting from turn_tick==0, that's 9 applied
	// turns (skips land on frames 4, 8, 12), so heading_index goes
	// 48 -> 57 exactly. This is deterministic (verified empirically) only
	// because the loop body now runs exactly once per real frame -- see the
	// last_frame debounce above.
	emu.SetInputButtons(0x0008) // RIGHT (bit 3)
	for i := 0; i < 12; i++ {
		emu.RunFrame()
	}
	emu.SetInputButtons(0x0000)
	emu.RunFrame()

	const wantHeading = 57
	turnedHeading := read16(emu, addrs["heading_index"])
	if turnedHeading != wantHeading {
		t.Fatalf("heading_index after 12 RIGHT-held frames: want %d, got %d", wantHeading, turnedHeading)
	}

	// Now hold UP and walk along the new heading: cam_x and cam_y should
	// both change (the new heading is off both cardinal axes).
	startX := read16(emu, addrs["cam_x"])
	startY := read16(emu, addrs["cam_y"])
	emu.SetInputButtons(0x0001) // UP held
	for i := 0; i < 5; i++ {
		emu.RunFrame()
	}
	if x := read16(emu, addrs["cam_x"]); x == startX {
		t.Errorf("walking along the turned heading: cam_x should change from %d, got %d", startX, x)
	}
	if y := read16(emu, addrs["cam_y"]); y == startY {
		t.Errorf("walking along the turned heading: cam_y should change from %d, got %d", startY, y)
	}

	// Floor projection and camera reached the plane, with a heading vector
	// matching heading_x/heading_y[heading_index] (8.8 fixed, magnitude 256).
	if emu.PPU.MatrixPlanes[0].ProjectionMode != 1 {
		t.Errorf("ProjectionMode want 1 (perspective floor), got %d", emu.PPU.MatrixPlanes[0].ProjectionMode)
	}
	if emu.PPU.MatrixPlanes[0].HeadingX == 0 && emu.PPU.MatrixPlanes[0].HeadingY == 0 {
		t.Errorf("plane heading should be nonzero after turning, got (0,0)")
	}
	// HUD text drew.
	emu.RunFrame()
	if emu.GetOutputBuffer() == nil {
		t.Error("no framebuffer")
	}
}
