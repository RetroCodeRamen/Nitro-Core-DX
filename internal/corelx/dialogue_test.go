package corelx

import (
	"testing"

	"nitro-core-dx/internal/emulator"
)

// enterDialogue walks from the interior entry point to the NPC's talk zone
// and presses A, returning the emulator with scene == SCENE_DIALOGUE (4).
func enterDialogue(t *testing.T) (*emulator.Emulator, map[string]uint16) {
	t.Helper()
	emu, addrs := enterInterior(t)

	emu.SetInputButtons(0x0001) // UP, walk into the talk zone
	for i := 0; i < 60; i++ {
		emu.RunFrame()
	}
	emu.SetInputButtons(0x0000)
	emu.RunFrame()
	emu.SetInputButtons(0x0010) // A
	emu.RunFrame()

	const sceneDialogue = 4
	if s := read16(emu, addrs["scene"]); s != sceneDialogue {
		t.Fatalf("scene after talking: want %d (dialogue), got %d (test setup assumption broken)", sceneDialogue, s)
	}
	return emu, addrs
}

func TestDialoguePaging(t *testing.T) {
	emu, addrs := enterDialogue(t)

	if p := read16(emu, addrs["dialog_page"]); p != 0 {
		t.Fatalf("dialog_page at dialogue start: want 0, got %d", p)
	}

	// Advance past page 0 (edge-triggered: release then press).
	emu.SetInputButtons(0x0000)
	emu.RunFrame()
	emu.SetInputButtons(0x0010) // A
	emu.RunFrame()

	const sceneDialogue = 4
	if p := read16(emu, addrs["dialog_page"]); p != 1 {
		t.Errorf("dialog_page after first A press: want 1, got %d", p)
	}
	if s := read16(emu, addrs["scene"]); s != sceneDialogue {
		t.Errorf("scene after first A press: want %d (still dialogue), got %d", sceneDialogue, s)
	}
}

func TestDialogueLastPageGoesToCredits(t *testing.T) {
	emu, addrs := enterDialogue(t)

	// Page 0 -> 1.
	emu.SetInputButtons(0x0000)
	emu.RunFrame()
	emu.SetInputButtons(0x0010)
	emu.RunFrame()

	// Page 1 (last) -> credits.
	emu.SetInputButtons(0x0000)
	emu.RunFrame()
	emu.SetInputButtons(0x0010)
	emu.RunFrame()

	const sceneCredits = 5
	if s := read16(emu, addrs["scene"]); s != sceneCredits {
		t.Errorf("scene after paging through dialogue: want %d (credits), got %d", sceneCredits, s)
	}
	if p := read16(emu, addrs["dialog_page"]); p != 0 {
		t.Errorf("dialog_page after reaching credits: want reset to 0, got %d", p)
	}
}

func TestCreditsResetsToOverworld(t *testing.T) {
	emu, addrs := enterDialogue(t)
	emu.SetInputButtons(0x0000)
	emu.RunFrame()
	emu.SetInputButtons(0x0010)
	emu.RunFrame()
	emu.SetInputButtons(0x0000)
	emu.RunFrame()
	emu.SetInputButtons(0x0010)
	emu.RunFrame()

	const sceneCredits = 5
	if s := read16(emu, addrs["scene"]); s != sceneCredits {
		t.Fatalf("scene: want %d (credits), got %d (test setup assumption broken)", sceneCredits, s)
	}

	emu.SetInputButtons(0x0000)
	emu.RunFrame()
	emu.SetInputButtons(0x0400) // START
	emu.RunFrame()

	const sceneOverworld = 1
	if s := read16(emu, addrs["scene"]); s != sceneOverworld {
		t.Fatalf("scene after START at credits: want %d (overworld), got %d", sceneOverworld, s)
	}
	if x := read16(emu, addrs["cam_x"]); x != 512 {
		t.Errorf("cam_x after reset: want 512, got %d", x)
	}
	if y := read16(emu, addrs["cam_y"]); y != 768 {
		t.Errorf("cam_y after reset: want 768, got %d", y)
	}
	if h := read16(emu, addrs["heading_index"]); h != 48 {
		t.Errorf("heading_index after reset: want 48, got %d", h)
	}

	// The loop restarts cleanly: overworld rendering should be live again
	// (bg 0/1 re-enabled by the reset).
	emu.SetInputButtons(0x0000)
	for i := 0; i < 5; i++ {
		emu.RunFrame()
	}
	if p := emu.PPU.MatrixPlanes[0]; !p.Enabled {
		t.Errorf("floor plane should be enabled again after returning to the overworld")
	}
}
