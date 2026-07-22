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
// m8-implementation-progress.md for what each of those changed. Updated
// again the same day for the hero sprite resize (8x8 -> a 2-tile 16x32
// composite, much closer to human-scaled next to the building's door), once
// more for the door-prompt heading-tolerance fix, again once the multi-bank
// compiler let overworld.corelx's world-projected trees/creature sprites
// actually ship, again for bump combat, and twice more on 2026-07-22: once
// for two real projection bugs (OAM anchor point; OBJ_FOV_K drastically
// undertuned, read as "everything bunches up around the player" -- see
// project_object/draw_object_sprite's comments), and once more for a
// distance-based near/far LOD swap (16x16 far, 32x32 4-tile composite near
// -- OAM sprites have no scale registers in this hardware, so a discrete
// size swap is the answer instead, AJ's call over adding real hardware
// scale registers) -- see draw_object_sprite_lod/OBJ_NEAR_LOD_FWD.
// Updated once more 2026-07-22 for a hardware-level sprite-size feature: OAM
// sprites now support 8 sizes (8x8 up to 128x128, see spriteSizeTable in
// internal/ppu/scanline.go) via a 3-bit code in the X-high byte, which
// required properly masking that byte's sign-bit computation everywhere it's
// written (oam.write_sprite_data, sprite.set_pos) instead of the previous
// raw/unmasked shift. That fix changes rendering for any sprite at a small
// negative screen X (previously got garbage in what are now the size-code
// bits, though harmless before since only bit 0 was ever consulted) --
// visually confirmed as more trees now correctly rendering near the screen
// edges rather than being silently corrupted.
// Updated once more 2026-07-22 for the tree/creature near-LOD simplification:
// once native OAM sprite sizes shipped, the software 4-tile 32x32 composite
// (4 OAM writes/tiles) was replaced with a single native SPR_SIZE_32X32()
// sprite (1 OAM write, one `tileset` asset holding the same 16 tiles laid
// out row-major). Visually identical (confirmed by PNG inspection -- same
// pixels, same positions); the hash changes are from OAM byte-layout
// differences leaking into checkpoints that still show a leftover
// tree/creature sprite from a previous scene (a pre-existing, harmless
// quirk -- see the multi-bank-era note above).
const (
	goldenOverworldSpawn    = "10ce19de8c2406e48b922574c22f0086068d79f93380b42fb6633db886cfc5af"
	goldenOverworldAtFacade = "776a4d9b6ed7ad09b573f5987c5f14c0da45ac5417628f5f22be4b6e29632e0a"
	goldenOverworldSideView = "1a97552a14f38b1d0b5a94a87fb962bc5711a4ba9c0c16e8712819547acdd908"
	goldenInteriorEntry     = "d1648573190f871354343cd84aaa4265c06926a438ac59d783fc8c02e84ba662"
	goldenDialoguePage0     = "cc8ff0c407c221488694e0f7e7d40804e9fd090457c7b818dbcf7b5c0ff0a893"
	goldenCredits           = "62838c47b803457083d7afc692853f0cdf4022ca669a8b667c103f82d0084a3f"
)
