package corelx

import "testing"

// TestOverworldWorldBoundsClamp verifies the camera clamps to the world's
// left edge (X=0) after walking into it for a long time, and stays clamped
// under continued pressure -- mirrors build_rom_test.go's
// TestNitroPackInDemoWorldBoundsClamp intent (turn LEFT 16 frames, walk UP
// long enough to reach the clamp, then push 20 more). Frame count is tuned
// to the demo's own move_x/move_y (speed 2.0), not the reference ROM's.
func TestOverworldWorldBoundsClamp(t *testing.T) {
	emu, result := compileProjectDirForTest(t, "Games/NitroPackInDemo/corelx/overworld.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)
	syncOverworldForTest(t, emu, addrs)

	emu.SetInputButtons(0x0004) // LEFT
	for i := 0; i < 16; i++ {
		emu.RunFrame()
	}
	emu.SetInputButtons(0x0000)
	emu.RunFrame()
	emu.RunFrame()

	emu.SetInputButtons(0x0001) // UP
	for i := 0; i < 260; i++ {
		emu.RunFrame()
	}
	const worldMinX = 0
	if x := read16(emu, addrs["cam_x"]); x != worldMinX {
		t.Fatalf("cam_x after walking into the left edge: want %d, got %d", worldMinX, x)
	}

	// Keep pushing against the edge -- should stay clamped, not wrap or drift.
	for i := 0; i < 20; i++ {
		emu.RunFrame()
	}
	if x := read16(emu, addrs["cam_x"]); x != worldMinX {
		t.Errorf("cam_x after pushing against the left edge: want %d, got %d", worldMinX, x)
	}
}

// TestOverworldWorldBoundsClampOtherEdges rounds out coverage on the
// remaining three edges (the reference test only exercises the left one).
// Heading index 0 (East: move_x=4, move_y=0) and 16 (South: move_x=0,
// move_y=4) are pure-axis directions that don't cross the building's
// collision footprint, so the world edge is reached cleanly.
func TestOverworldWorldBoundsClampOtherEdges(t *testing.T) {
	t.Run("right_edge", func(t *testing.T) {
		emu, result := compileProjectDirForTest(t, "Games/NitroPackInDemo/corelx/overworld.corelx")
		addrs := map[string]uint16{}
		for _, e := range result.MemoryMap {
			addrs[e.Name] = e.Address
		}
		emu.Start()
		emu.SetFrameLimit(false)
		syncOverworldForTest(t, emu, addrs)

		// Turn from North (48) to East (0): RIGHT wraps 63->0, so this needs
		// 16 applied turns (48->63 is 15 steps, +1 more wraps to 0). Turning
		// is rate-limited to 3 of every 4 held frames starting from an
		// aligned turn_tick==0 (syncOverworldForTest), so applies(N) =
		// N - floor(N/4) exactly; N=21 gives exactly 16.
		emu.SetInputButtons(0x0008) // RIGHT
		for i := 0; i < 21; i++ {
			emu.RunFrame()
		}
		emu.SetInputButtons(0x0000)
		emu.RunFrame()
		if h := read16(emu, addrs["heading_index"]); h != 0 {
			t.Fatalf("heading_index after turning to East: want 0, got %d (test setup assumption broken)", h)
		}

		emu.SetInputButtons(0x0001) // UP
		for i := 0; i < 260; i++ {
			emu.RunFrame()
		}
		const worldMaxX = 1023
		if x := read16(emu, addrs["cam_x"]); x != worldMaxX {
			t.Errorf("cam_x after walking into the right edge: want %d, got %d", worldMaxX, x)
		}
		if y := read16(emu, addrs["cam_y"]); y != 768 {
			t.Errorf("cam_y should be unchanged walking due East: want 768, got %d", y)
		}
	})

	t.Run("bottom_edge", func(t *testing.T) {
		emu, result := compileProjectDirForTest(t, "Games/NitroPackInDemo/corelx/overworld.corelx")
		addrs := map[string]uint16{}
		for _, e := range result.MemoryMap {
			addrs[e.Name] = e.Address
		}
		emu.Start()
		emu.SetFrameLimit(false)
		syncOverworldForTest(t, emu, addrs)

		// Turn from North (48) to South (16): 32 applied turns needed
		// (48->63->0->16); N=42 gives exactly 32 (see the East case above
		// for the applies(N) = N - floor(N/4) derivation).
		emu.SetInputButtons(0x0008) // RIGHT
		for i := 0; i < 42; i++ {
			emu.RunFrame()
		}
		emu.SetInputButtons(0x0000)
		emu.RunFrame()
		if h := read16(emu, addrs["heading_index"]); h != 16 {
			t.Fatalf("heading_index after turning to South: want 16, got %d (test setup assumption broken)", h)
		}

		emu.SetInputButtons(0x0001) // UP
		for i := 0; i < 140; i++ {
			emu.RunFrame()
		}
		const worldMaxY = 1023
		if y := read16(emu, addrs["cam_y"]); y != worldMaxY {
			t.Errorf("cam_y after walking into the bottom edge: want %d, got %d", worldMaxY, y)
		}
		if x := read16(emu, addrs["cam_x"]); x != 512 {
			t.Errorf("cam_x should be unchanged walking due South: want 512, got %d", x)
		}
	})
}
