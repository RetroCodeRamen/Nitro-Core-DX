package corelx

import (
	"testing"
)

// TestOverworldBuildingBillboard verifies the M8 demo rebuild's building
// billboard (matrix plane 1, vertical projection) matches the hand-built
// reference ROM's plane-1 configuration exactly (Games/NitroPackInDemo/
// build_rom.go's `objects` billboardPlane entry, and the register
// assertions in build_rom_test.go's TestNitroPackInDemoSceneFlow).
func TestOverworldBuildingBillboard(t *testing.T) {
	emu, result := compileProjectDirForTest(t, "Games/NitroPackInDemo/corelx/overworld.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)
	syncOverworldForTest(t, emu, addrs)

	floor := emu.PPU.MatrixPlanes[0]
	billboard := emu.PPU.MatrixPlanes[1]

	if !billboard.Enabled {
		t.Fatalf("billboard plane not enabled")
	}
	if billboard.ProjectionMode != 2 {
		t.Errorf("billboard ProjectionMode want 2 (vertical), got %d", billboard.ProjectionMode)
	}
	if !billboard.Transparent0 {
		t.Errorf("billboard Transparent0 want true (color-key 0), got false")
	}
	if !billboard.TwoSided {
		t.Errorf("billboard TwoSided want true (no backface cull), got false")
	}

	// Surface registers: world anchor (512,600), facing straight up-screen
	// (0, 0x0100), heightScale 0x4000 -- matches build_rom.go's `objects[0]`.
	if billboard.OriginX != 512 || billboard.OriginY != 600 {
		t.Errorf("billboard origin want (512,600), got (%d,%d)", billboard.OriginX, billboard.OriginY)
	}
	if billboard.FacingX != 0 || billboard.FacingY != 0x0100 {
		t.Errorf("billboard facing want (0,0x0100), got (0x%04X,0x%04X)", uint16(billboard.FacingX), uint16(billboard.FacingY))
	}
	if billboard.HeightScale != 0x4000 {
		t.Errorf("billboard HeightScale want 0x4000, got 0x%04X", billboard.HeightScale)
	}
	if billboard.WidthScale != 0x0070 {
		t.Errorf("billboard WidthScale want 0x0070, got 0x%04X", billboard.WidthScale)
	}

	// The billboard shares the floor's horizon/baseDistance/focalLength --
	// both planes render against the same camera projection depth.
	if billboard.Horizon != floor.Horizon {
		t.Errorf("billboard horizon should match floor: got %d want %d", billboard.Horizon, floor.Horizon)
	}
	if billboard.BaseDistance != floor.BaseDistance {
		t.Errorf("billboard base distance should match floor: got 0x%04X want 0x%04X", billboard.BaseDistance, floor.BaseDistance)
	}
	if billboard.FocalLength != floor.FocalLength {
		t.Errorf("billboard focal length should match floor: got 0x%04X want 0x%04X", billboard.FocalLength, floor.FocalLength)
	}

	// Camera sync: at boot (heading_index 48, North: headingX=0,
	// headingY=-256), the floor camera trails the player by heading>>2 (feet
	// pivot) -- 768 - (-256>>2) = 832 -- while the billboard tracks the raw
	// player position (768). Both share the same heading vector.
	if floor.CameraX != 512 || floor.CameraY != 832 {
		t.Errorf("floor camera want (512,832), got (%d,%d)", floor.CameraX, floor.CameraY)
	}
	if billboard.CameraX != 512 || billboard.CameraY != 768 {
		t.Errorf("billboard camera want (512,768) (raw player position), got (%d,%d)", billboard.CameraX, billboard.CameraY)
	}
	if floor.HeadingX != 0 || floor.HeadingY != -256 {
		t.Errorf("floor heading want (0,-256), got (%d,%d)", floor.HeadingX, floor.HeadingY)
	}
	if billboard.HeadingX != floor.HeadingX || billboard.HeadingY != floor.HeadingY {
		t.Errorf("billboard heading should match floor: got (%d,%d) want (%d,%d)",
			billboard.HeadingX, billboard.HeadingY, floor.HeadingX, floor.HeadingY)
	}
}

// TestOverworldBuildingCollision verifies the player is stopped at the
// building's front face (Y=600) rather than walking through it, matching
// build_rom.go's buildingCollisionMinX/MaxX/buildingFrontY clamp.
func TestOverworldBuildingCollision(t *testing.T) {
	emu, result := compileProjectDirForTest(t, "Games/NitroPackInDemo/corelx/overworld.corelx")
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu.Start()
	emu.SetFrameLimit(false)
	syncOverworldForTest(t, emu, addrs)

	// heading_index starts at 48 (North: move_x=0, move_y=-4), and cam_x=512
	// sits inside the building's collision footprint (472..552), so walking
	// UP for long enough should stop the player at cam_y=600, not below it.
	emu.SetInputButtons(0x0001) // UP held
	for i := 0; i < 60; i++ {
		emu.RunFrame()
	}
	emu.SetInputButtons(0x0000)
	emu.RunFrame()

	if y := read16(emu, addrs["cam_y"]); y != 600 {
		t.Errorf("cam_y after walking into the building: want 600 (stopped at the facade), got %d", y)
	}
}
