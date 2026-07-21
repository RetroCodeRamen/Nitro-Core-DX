package corelx

import "testing"

// TestOverworldTurnLeftWraps verifies LEFT decrements heading_index and
// wraps 0 -> 63 (the mirror image of RIGHT's 63 -> 0 wrap, exercised by
// TestOverworldRebuild). 64 held-LEFT frames from the initial heading_index
// 48 apply exactly 48 turns (3 of every 4 frames, and 64 is a multiple of
// 4), landing exactly on 0; one more applied turn wraps to 63.
func TestOverworldTurnLeftWraps(t *testing.T) {
	emu, result := compileProjectDirForTest(t, "Games/NitroPackInDemo/corelx/overworld.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)
	syncOverworldForTest(t, emu, addrs)

	emu.SetInputButtons(0x0004) // LEFT
	for i := 0; i < 64; i++ {
		emu.RunFrame()
	}
	if h := read16(emu, addrs["heading_index"]); h != 0 {
		t.Fatalf("heading_index after 64 held-LEFT frames: want 0, got %d", h)
	}
	emu.RunFrame() // one more applied turn (65 is not a multiple of 4, but frame 65's tick is 0 -- see syncOverworldForTest)
	if h := read16(emu, addrs["heading_index"]); h != 63 {
		t.Fatalf("heading_index after one more LEFT turn past 0: want 63 (wrapped), got %d", h)
	}
}

// TestOverworldWalkBackward verifies DOWN moves the camera opposite to UP
// along the current heading (mirrors TestOverworldRebuild's UP-only check,
// which never exercised DOWN). At the initial heading (48, North: move_x=0,
// move_y=-4), holding DOWN should increase cam_y (walk backward = south)
// while leaving cam_x unchanged.
func TestOverworldWalkBackward(t *testing.T) {
	emu, result := compileProjectDirForTest(t, "Games/NitroPackInDemo/corelx/overworld.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)
	syncOverworldForTest(t, emu, addrs)

	emu.SetInputButtons(0x0002) // DOWN
	for i := 0; i < 5; i++ {
		emu.RunFrame()
	}
	const wantX, wantY = 512, 788 // 768 + 5*4
	if x := read16(emu, addrs["cam_x"]); x != wantX {
		t.Errorf("cam_x after walking backward at North: want %d (unchanged), got %d", wantX, x)
	}
	if y := read16(emu, addrs["cam_y"]); y != wantY {
		t.Errorf("cam_y after walking backward at North: want %d (moved south), got %d", wantY, y)
	}
}

// TestOverworldTurnAndWalkSimultaneously verifies turning and walking held
// together in the same frames: each frame's movement uses that same
// frame's just-updated heading (turn logic runs before movement in the
// loop body), matching build_rom.go's instruction order. Exact values
// verified against an independent frame-by-frame simulation of the turn
// rate limiter and heading table.
func TestOverworldTurnAndWalkSimultaneously(t *testing.T) {
	emu, result := compileProjectDirForTest(t, "Games/NitroPackInDemo/corelx/overworld.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)
	syncOverworldForTest(t, emu, addrs)

	emu.SetInputButtons(0x0005) // LEFT | UP
	for i := 0; i < 12; i++ {
		emu.RunFrame()
	}

	const wantHeading, wantX, wantY = 39, 491, 732
	if h := read16(emu, addrs["heading_index"]); h != wantHeading {
		t.Errorf("heading_index after 12 simultaneous LEFT+UP frames: want %d, got %d", wantHeading, h)
	}
	if x := read16(emu, addrs["cam_x"]); x != wantX {
		t.Errorf("cam_x after 12 simultaneous LEFT+UP frames: want %d, got %d", wantX, x)
	}
	if y := read16(emu, addrs["cam_y"]); y != wantY {
		t.Errorf("cam_y after 12 simultaneous LEFT+UP frames: want %d, got %d", wantY, y)
	}
}

// TestOverworldBuildingCollisionOffRange verifies the building collision
// clamp only applies when the player's X is within the facade's footprint
// (472..552) -- walking north with X well outside that range should pass
// straight through Y=600 uncontested, not get stopped.
func TestOverworldBuildingCollisionOffRange(t *testing.T) {
	emu, result := compileProjectDirForTest(t, "Games/NitroPackInDemo/corelx/overworld.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)
	syncOverworldForTest(t, emu, addrs)

	// Turn right 12 frames (heading_index -> 57, move_x=3/move_y=-2) then
	// walk that heading for 100 frames: cam_x drifts from 512 to 812 (well
	// outside the 472..552 collision footprint well before cam_y approaches
	// 600), and cam_y drifts from 768 to 568 -- below the facade line.
	emu.SetInputButtons(0x0008) // RIGHT
	for i := 0; i < 12; i++ {
		emu.RunFrame()
	}
	emu.SetInputButtons(0x0001) // UP
	for i := 0; i < 100; i++ {
		emu.RunFrame()
	}

	const wantX, wantY = 812, 568
	if x := read16(emu, addrs["cam_x"]); x != wantX {
		t.Fatalf("cam_x want %d, got %d (test setup assumption broken)", wantX, x)
	}
	if y := read16(emu, addrs["cam_y"]); y != wantY {
		t.Errorf("cam_y want %d (passed Y=600 uncontested, X out of collision range), got %d -- collision may be firing when it shouldn't", wantY, y)
	}
}
