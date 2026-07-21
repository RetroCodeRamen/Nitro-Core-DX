package corelx

import "testing"

// TestOverworldDoorInteraction verifies walking up to the building's door
// footprint and pressing A transitions to the interior scene with the
// expected entry state, matching build_rom.go's doorMinX/doorMaxX/doorMaxY
// footprint and interiorEntryX/Y/Heading constants exactly.
func TestOverworldDoorInteraction(t *testing.T) {
	emu, result := compileProjectDirForTest(t, "Games/NitroPackInDemo/corelx/overworld.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)
	syncOverworldForTest(t, emu, addrs)

	// heading_index starts at 48 (North: move_x=0, move_y=-4) and cam_x=512
	// sits inside the door's X footprint (494..530) already, so walking UP
	// (no turning needed) approaches the door head-on. Building collision
	// stops cam_y at 600, well inside the door's Y footprint (<=696), long
	// before A is ever pressed.
	emu.SetInputButtons(0x0001) // UP
	for i := 0; i < 60; i++ {
		emu.RunFrame()
	}
	if y := read16(emu, addrs["cam_y"]); y != 600 {
		t.Fatalf("cam_y after walking to the door: want 600, got %d (test setup assumption broken)", y)
	}
	if s := read16(emu, addrs["scene"]); s != 1 {
		t.Fatalf("scene before pressing A: want 1 (overworld), got %d", s)
	}

	// Press A (edge-triggered: down this frame, was up last frame).
	emu.SetInputButtons(0x0000)
	emu.RunFrame()
	emu.SetInputButtons(0x0010) // A
	emu.RunFrame()

	if s := read16(emu, addrs["scene"]); s != 2 {
		t.Fatalf("scene after pressing A at the door: want 2 (interior), got %d", s)
	}
	if x := read16(emu, addrs["int_x"]); x != 512 {
		t.Errorf("int_x after entering: want 512, got %d", x)
	}
	if y := read16(emu, addrs["int_y"]); y != 616 {
		t.Errorf("int_y after entering: want 616, got %d", y)
	}
	if h := read16(emu, addrs["int_heading"]); h != 48 {
		t.Errorf("int_heading after entering: want 48 (North), got %d", h)
	}
}

// TestOverworldDoorRequiresProximity verifies pressing A far from the door
// does not trigger the scene transition.
func TestOverworldDoorRequiresProximity(t *testing.T) {
	emu, result := compileProjectDirForTest(t, "Games/NitroPackInDemo/corelx/overworld.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)
	syncOverworldForTest(t, emu, addrs)

	// Starting position (cam_x=512, cam_y=768) is within the door's X range
	// but Y=768 is outside the <=696 footprint -- press A immediately.
	emu.SetInputButtons(0x0010) // A
	emu.RunFrame()

	if s := read16(emu, addrs["scene"]); s != 1 {
		t.Errorf("scene after pressing A away from the door: want 1 (unchanged), got %d", s)
	}
}
