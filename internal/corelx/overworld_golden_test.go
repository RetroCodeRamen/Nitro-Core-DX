package corelx

import (
	"testing"

	"nitro-core-dx/internal/emulator"
)

// turnTo holds LEFT or RIGHT and steps frame-by-frame until heading_index
// reaches target, rather than hand-deriving an exact frame count from the
// turn-rate limiter -- robust regardless of turn_tick's current phase
// (which the N - floor(N/4) shortcut requires starting aligned at 0 for,
// per syncOverworldForTest).
func turnTo(t *testing.T, emu *emulator.Emulator, addrs map[string]uint16, target uint16, left bool) {
	t.Helper()
	buttons := uint16(0x0008) // RIGHT
	if left {
		buttons = 0x0004 // LEFT
	}
	emu.SetInputButtons(buttons)
	for i := 0; i < 300; i++ {
		if read16(emu, addrs["heading_index"]) == target {
			break
		}
		emu.RunFrame()
	}
	emu.SetInputButtons(0x0000)
	emu.RunFrame()
	if h := read16(emu, addrs["heading_index"]); h != target {
		t.Fatalf("turnTo(%d): want heading %d, got %d (didn't converge)", target, target, h)
	}
}

// TestOverworldGoldenFrames is the M8 Phase 5 acceptance gate: a
// framebuffer-hash regression test over the compiled overworld.corelx demo,
// covering every major visual state established this phase (floor scale,
// sprite position/pivot, building placement, depth-sort, scene transitions).
// Unlike TestGraphicsPipelineShowcaseGoldenFrames (a passive, scripted
// animation with no input), these checkpoints are reached via real input
// sequences -- so a regression here can come from gameplay logic, not just
// rendering math.
//
// If a checkpoint's hash needs to change (an intentional visual change, not
// a regression), set NCDX_SHOWCASE_DUMP_DIR to inspect the new frame via
// maybeDumpShowcaseFrame, confirm the new frame looks right, then update the
// constant below.
func TestOverworldGoldenFrames(t *testing.T) {
	emu, result := compileProjectDirForTest(t, "Games/NitroPackInDemo/corelx/overworld.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)
	syncOverworldForTest(t, emu, addrs)

	check := func(name, wantHash string, fb []uint32) {
		t.Helper()
		got := framebufferHash(fb)
		maybeDumpShowcaseFrame(t, name, fb)
		if got != wantHash {
			t.Errorf("%s framebuffer hash mismatch: got=%s want=%s", name, got, wantHash)
		}
	}

	// 1. Spawn: floor scale, building placement, sprite position/pivot all
	// visible at once, no input yet.
	check("overworld_spawn", goldenOverworldSpawn, emu.PPU.OutputBuffer[:])

	// 2. Walk to the building's facade: exercises the AABB collision clamp
	// and the depth-sort (building should occlude the sprite here).
	emu.SetInputButtons(0x0001) // UP
	for i := 0; i < 100; i++ {
		emu.RunFrame()
	}
	emu.SetInputButtons(0x0000)
	emu.RunFrame()
	if y := read16(emu, addrs["cam_y"]); y != 600 {
		t.Fatalf("cam_y at facade: want 600, got %d (test setup assumption broken)", y)
	}
	if n := read16(emu, addrs["building_near"]); n != 1 {
		t.Fatalf("building_near at facade: want 1, got %d (test setup assumption broken)", n)
	}
	check("overworld_at_facade", goldenOverworldAtFacade, emu.PPU.OutputBuffer[:])

	// 3. Back off and turn to a side view: exercises the shared camera-eye
	// (floor/billboard alignment) from an angle other than straight-on.
	emu.SetInputButtons(0x0002) // DOWN, back away from the facade
	for i := 0; i < 40; i++ {
		emu.RunFrame()
	}
	emu.SetInputButtons(0x0000)
	emu.RunFrame()
	turnTo(t, emu, addrs, 32, true) // face West
	check("overworld_side_view", goldenOverworldSideView, emu.PPU.OutputBuffer[:])

	// 4. Interior scene entry.
	iemu, iaddrs := enterInterior(t)
	if s := read16(iemu, iaddrs["scene"]); s != 2 {
		t.Fatalf("scene after entering interior: want 2, got %d", s)
	}
	check("interior_entry", goldenInteriorEntry, iemu.PPU.OutputBuffer[:])

	// 5. Dialogue scene, first page. enterDialogue returns the frame
	// scene first flips to 4 -- the dialogue branch's own drawing (the
	// "GUIDE" text overlay) doesn't run until the following frame (the
	// interior branch that was active when this frame's dispatch ran
	// already did its own drawing before the scene-changing input check
	// at the bottom), so one settle frame is needed before the rendered
	// output actually reflects the dialogue scene.
	demu, daddrs := enterDialogue(t)
	if s := read16(demu, daddrs["scene"]); s != 4 {
		t.Fatalf("scene after entering dialogue: want 4, got %d", s)
	}
	demu.SetInputButtons(0x0000)
	demu.RunFrame()
	check("dialogue_page0", goldenDialoguePage0, demu.PPU.OutputBuffer[:])

	// 6. Credits scene, reached by paging through dialogue.
	demu.SetInputButtons(0x0000)
	demu.RunFrame()
	demu.SetInputButtons(0x0010) // A: page 0 -> 1
	demu.RunFrame()
	demu.SetInputButtons(0x0000)
	demu.RunFrame()
	demu.SetInputButtons(0x0010) // A: page 1 -> credits
	demu.RunFrame()
	demu.SetInputButtons(0x0000)
	demu.RunFrame()
	if s := read16(demu, daddrs["scene"]); s != 5 {
		t.Fatalf("scene after paging through dialogue: want 5 (credits), got %d", s)
	}
	check("credits", goldenCredits, demu.PPU.OutputBuffer[:])
}

// Golden hashes, captured 2026-07-21 right after the M8 Phase 5 post-review
// tuning pass (sprite position/pivot/speed, floor rescale, building
// collision/depth-sort, camera-eye alignment fix) -- see
// m8-implementation-progress.md for what each of those changed.
const (
	goldenOverworldSpawn    = "e5550e9ee2c46bc3583d027e158ce27dbf05e5ede4cedee26b3733f15042d6ce"
	goldenOverworldAtFacade = "6a16423056837ef62a83a2137f34772ce5e11c4d2f9f127b9658d92e73b772fa"
	goldenOverworldSideView = "283c19b257bdaf61c0860e0f18e43c940bec0b1e95c91823720899de3c2ec51b"
	goldenInteriorEntry     = "a5a063c9f64ea5218e110fc5972943537ee88777db6f2924ba7868a2a55b5574"
	goldenDialoguePage0     = "00f63faaa23c2a5ced8ff46f221e77517a314c410a4582b66a46c14c1197d3f0"
	goldenCredits           = "64c4fafbe7f877e6a0eebab7e89d281b2a0a93630903b6d6733defdae95ba2ce"
)
