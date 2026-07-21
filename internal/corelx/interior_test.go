package corelx

import (
	"testing"

	"nitro-core-dx/internal/emulator"
)

// enterInterior walks to the overworld door and presses A, returning the
// emulator with scene == SCENE_INTERIOR (2) and the addrs map.
func enterInterior(t *testing.T) (*emulator.Emulator, map[string]uint16) {
	t.Helper()
	emu, result := compileProjectDirForTest(t, "Games/NitroPackInDemo/corelx/overworld.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)
	syncOverworldForTest(t, emu, addrs)

	emu.SetInputButtons(0x0001) // UP, walk to the facade
	for i := 0; i < 60; i++ {
		emu.RunFrame()
	}
	emu.SetInputButtons(0x0000)
	emu.RunFrame()
	emu.SetInputButtons(0x0010) // A
	emu.RunFrame()
	emu.SetInputButtons(0x0000)
	emu.RunFrame()

	if s := read16(emu, addrs["scene"]); s != 2 {
		t.Fatalf("scene after entering: want 2 (interior), got %d (test setup assumption broken)", s)
	}
	return emu, addrs
}

func TestOverworldNPCCollision(t *testing.T) {
	emu, addrs := enterInterior(t)

	// int_heading starts at 48 (North: move_x=0, move_y=-4) and int_x=512
	// sits inside the NPC's collision footprint (488..536), so walking UP
	// should stop the player at int_y=500 (npcAnchorY + 28), not reach the
	// NPC's own position (472).
	emu.SetInputButtons(0x0001) // UP
	for i := 0; i < 60; i++ {
		emu.RunFrame()
	}
	if y := read16(emu, addrs["int_y"]); y != 500 {
		t.Errorf("int_y after walking toward the guide: want 500 (stopped by collision), got %d", y)
	}
}

func TestOverworldTalkToNPC(t *testing.T) {
	emu, addrs := enterInterior(t)

	emu.SetInputButtons(0x0001) // UP, walk into the talk zone
	for i := 0; i < 60; i++ {
		emu.RunFrame()
	}
	emu.SetInputButtons(0x0000)
	emu.RunFrame()

	if s := read16(emu, addrs["scene"]); s != 2 {
		t.Fatalf("scene before talking: want 2 (interior), got %d", s)
	}
	emu.SetInputButtons(0x0010) // A
	emu.RunFrame()

	const sceneDialogue = 4
	if s := read16(emu, addrs["scene"]); s != sceneDialogue {
		t.Errorf("scene after pressing A at the guide: want %d (dialogue), got %d", sceneDialogue, s)
	}
}

func TestOverworldExitInterior(t *testing.T) {
	emu, addrs := enterInterior(t)

	// Entry position (512,616) already sits in the exit zone (X in
	// [472,552], Y>=608) -- press A immediately (edge-triggered, so release
	// first to guarantee a fresh press).
	emu.SetInputButtons(0x0000)
	emu.RunFrame()
	emu.SetInputButtons(0x0010) // A
	emu.RunFrame()

	if s := read16(emu, addrs["scene"]); s != 1 {
		t.Errorf("scene after pressing A at the exit: want 1 (overworld), got %d", s)
	}
}

func TestOverworldInteriorRoomBounds(t *testing.T) {
	emu, addrs := enterInterior(t)

	// Turn to face West (index 32: move_x=-4, move_y=0) and walk long enough
	// to hit the room's west wall (interiorMinX = 416).
	emu.SetInputButtons(0x0004) // LEFT, turn from North (48) toward West (32)
	for i := 0; i < 22; i++ {
		emu.RunFrame()
	}
	emu.SetInputButtons(0x0000)
	emu.RunFrame()
	emu.SetInputButtons(0x0001) // UP
	for i := 0; i < 60; i++ {
		emu.RunFrame()
	}
	if x := read16(emu, addrs["int_x"]); x < 416 {
		t.Errorf("int_x should clamp at the room's west wall (416), got %d", x)
	}
}
