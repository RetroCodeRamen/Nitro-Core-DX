package corelx

import "testing"

// TestMusicSetVolumeSetsRegister verifies music.set_volume writes FMRegVolume
// immediately.
func TestMusicSetVolumeSetsRegister(t *testing.T) {
	mainSource := `asset Theme: music "theme.ncdxmusic"

function Start()
    music.set_volume(128)
    while true
        wait_vblank()
`
	emu := compileAndRunFramesWithMusic(t, "theme.ncdxmusic", threeFrameNcdxmusic(t), mainSource, 1)
	if got := emu.APU.FM.Volume; got != 128 {
		t.Errorf("FM.Volume: want 128, got %d", got)
	}
}

// TestMusicSetVolumeCancelsFade verifies an explicit music.set_volume call
// stops an in-progress fade rather than letting it keep overwriting the
// volume on later frames.
func TestMusicSetVolumeCancelsFade(t *testing.T) {
	mainSource := `asset Theme: music "theme.ncdxmusic"

function Start()
    music.fade_to(0, 4)
    wait_vblank()
    music.set_volume(200)
    while true
        wait_vblank()
`
	// Frame 1: one fade step runs (before set_volume executes, since
	// set_volume is called after the first wait_vblank()). Frame 2:
	// set_volume(200) has now run; if the fade were still active it would
	// immediately overwrite this on frame 2's own advance.
	emu := compileAndRunFramesWithMusic(t, "theme.ncdxmusic", threeFrameNcdxmusic(t), mainSource, 2)
	if got := emu.APU.FM.Volume; got != 200 {
		t.Errorf("FM.Volume after set_volume cancels fade: want 200, got %d", got)
	}
}

// TestMusicFadeToInterpolatesAndLandsOnTarget verifies fade_to steps the
// volume down over the requested number of frames and lands exactly on the
// target on the last step, regardless of integer-division rounding along
// the way.
func TestMusicFadeToInterpolatesAndLandsOnTarget(t *testing.T) {
	mainSource := `asset Theme: music "theme.ncdxmusic"

function Start()
    music.fade_to(0, 4)
    while true
        wait_vblank()
`
	// Starting volume is 0xFF (255, the FMOPM default). Fading to 0 over 4
	// frames: 255 -> 192 -> 128 -> 64 -> 0 (three partial steps, then an
	// exact snap to target on the last one).
	want := []uint8{192, 128, 64, 0}
	for i, w := range want {
		frames := i + 1
		emu := compileAndRunFramesWithMusic(t, "theme.ncdxmusic", threeFrameNcdxmusic(t), mainSource, frames)
		if got := emu.APU.FM.Volume; got != w {
			t.Errorf("after %d frame(s): FM.Volume want %d, got %d", frames, w, got)
		}
	}
}

// TestMusicFadeToUpwardInterpolates verifies fade_to also works when fading
// up (target > current), exercising the sign-handling path.
func TestMusicFadeToUpwardInterpolates(t *testing.T) {
	mainSource := `asset Theme: music "theme.ncdxmusic"

function Start()
    music.set_volume(0)
    music.fade_to(100, 2)
    while true
        wait_vblank()
`
	// 0 -> 50 -> 100 (2 steps: partial then exact snap).
	emu := compileAndRunFramesWithMusic(t, "theme.ncdxmusic", threeFrameNcdxmusic(t), mainSource, 1)
	if got := emu.APU.FM.Volume; got != 50 {
		t.Errorf("after 1 frame: FM.Volume want 50, got %d", got)
	}
	emu = compileAndRunFramesWithMusic(t, "theme.ncdxmusic", threeFrameNcdxmusic(t), mainSource, 2)
	if got := emu.APU.FM.Volume; got != 100 {
		t.Errorf("after 2 frames: FM.Volume want 100, got %d", got)
	}
}

// TestMusicFadeToWithoutMusicAssetRejected verifies fade_to is a compile
// error when the program declares no music asset, since there would be no
// per-frame hook to advance it.
func TestMusicFadeToWithoutMusicAssetRejected(t *testing.T) {
	source := `function Start()
    music.fade_to(0, 10)
    while true
        wait_vblank()
`
	err := compileExpectError(t, source)
	if err == nil {
		t.Fatal("expected compile error")
	}
}
