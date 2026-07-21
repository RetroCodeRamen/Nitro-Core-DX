package corelx

import (
	"os"
	"path/filepath"
	"testing"

	"nitro-core-dx/internal/apu"
	"nitro-core-dx/internal/emulator"
	"nitro-core-dx/internal/ymstream"
)

// threeFrameNcdxmusic returns a 3-frame .ncdxmusic stream, one distinguishable
// YM2608 write per frame, so frame-by-frame advancement and looping can be
// observed precisely.
func threeFrameNcdxmusic(t *testing.T) []byte {
	t.Helper()
	song := &ymstream.Song{
		Frames: [][]ymstream.Write{
			{{Port: 0, Addr: 0x10, Data: 0xAA}},
			{{Port: 0, Addr: 0x11, Data: 0xBB}},
			{{Port: 0, Addr: 0x12, Data: 0xCC}},
		},
		FrameSamples: 735,
		WriteCount:   3,
	}
	data, err := ymstream.EncodeSong(song)
	if err != nil {
		t.Fatalf("EncodeSong: %v", err)
	}
	return data
}

// compileAndRunFramesWithMusic compiles a project with the given music asset
// file and main source, boots it, and runs it for the given number of real
// PPU frames (needed since __musicadvance is driven by wait_vblank(), which
// blocks on the real VBlank flag).
func compileAndRunFramesWithMusic(t *testing.T, musicFile string, musicData []byte, mainSource string, frames int) *emulator.Emulator {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, musicFile), musicData, 0644); err != nil {
		t.Fatalf("write music asset: %v", err)
	}
	srcPath := filepath.Join(dir, "main.corelx")
	romPath := filepath.Join(dir, "main.rom")
	if err := os.WriteFile(srcPath, []byte(mainSource), 0644); err != nil {
		t.Fatalf("write main source: %v", err)
	}
	if _, err := CompileProject(srcPath, &CompileOptions{OutputPath: romPath}); err != nil {
		t.Fatalf("compile: %v", err)
	}
	romData, err := os.ReadFile(romPath)
	if err != nil {
		t.Fatalf("read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	emu.Start()
	for i := 0; i < frames; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame %d: %v", i, err)
		}
	}
	return emu
}

const musicMainSource = `asset Theme: music "theme.ncdxmusic"

function Start()
    music.play_loop(Theme)
    while true
        wait_vblank()
`

// TestMusicPlayLoopAdvancesFrames verifies music.play_loop streams each
// song frame's YM2608 writes through the burst streamer on successive
// wait_vblank() calls, in order.
func TestMusicPlayLoopAdvancesFrames(t *testing.T) {
	cases := []struct {
		frames   int
		wantAddr uint8
		wantData uint8
	}{
		{1, 0x10, 0xAA}, // frame 0
		{2, 0x11, 0xBB}, // frame 1
		{3, 0x12, 0xCC}, // frame 2 (last frame)
	}
	for _, c := range cases {
		emu := compileAndRunFramesWithMusic(t, "theme.ncdxmusic", threeFrameNcdxmusic(t), musicMainSource, c.frames)
		if got := emu.APU.FM.Addr; got != c.wantAddr {
			t.Errorf("after %d frame(s): FM addr want 0x%02X, got 0x%02X", c.frames, c.wantAddr, got)
		}
		if got := emu.APU.FM.Read8(apu.FMRegData); got != c.wantData {
			t.Errorf("after %d frame(s): FM data want 0x%02X, got 0x%02X", c.frames, c.wantData, got)
		}
	}
}

// TestMusicPlayLoopWrapsToStart verifies play_loop wraps back to frame 0
// after the last frame instead of stopping.
func TestMusicPlayLoopWrapsToStart(t *testing.T) {
	// 3 frames, then a 4th wait_vblank() should replay frame 0.
	emu := compileAndRunFramesWithMusic(t, "theme.ncdxmusic", threeFrameNcdxmusic(t), musicMainSource, 4)
	if got := emu.APU.FM.Addr; got != 0x10 {
		t.Errorf("after wrap: FM addr want 0x10 (frame 0 replayed), got 0x%02X", got)
	}
	if got := emu.APU.FM.Read8(apu.FMRegData); got != 0xAA {
		t.Errorf("after wrap: FM data want 0xAA, got 0x%02X", got)
	}
}

// TestMusicPlayOneShotStopsAtEnd verifies music.play (one-shot) silences the
// chip and stops advancing once the last frame has played, rather than
// looping like play_loop.
func TestMusicPlayOneShotStopsAtEnd(t *testing.T) {
	mainSource := `asset Theme: music "theme.ncdxmusic"

function Start()
    music.play(Theme)
    while true
        wait_vblank()
`
	// Run past the end (3 frames) by one more tick.
	emu := compileAndRunFramesWithMusic(t, "theme.ncdxmusic", threeFrameNcdxmusic(t), mainSource, 4)
	// The silence sequence's last operation is the ADPCM-B reset via port 1
	// (addr 0x00, data 0x01) — confirms music.stop's equivalent sequence ran
	// at the natural end of a one-shot song, not just on an explicit
	// music.stop() call.
	if got := emu.APU.FM.MixL; got != 0x00 {
		t.Errorf("one-shot end: port-1 address want 0x00 (ADPCM-B reset), got 0x%02X", got)
	}
	if got := emu.APU.FM.MixR; got != 0x01 {
		t.Errorf("one-shot end: port-1 data want 0x01, got 0x%02X", got)
	}
}

// TestMusicStopSilencesImmediately verifies an explicit music.stop() call
// runs the silence sequence right away, not just at a song's natural end.
func TestMusicStopSilencesImmediately(t *testing.T) {
	mainSource := `asset Theme: music "theme.ncdxmusic"

function Start()
    music.play_loop(Theme)
    wait_vblank()
    music.stop()
    while true
        wait_vblank()
`
	emu := compileAndRunFramesWithMusic(t, "theme.ncdxmusic", threeFrameNcdxmusic(t), mainSource, 2)
	if got := emu.APU.FM.MixL; got != 0x00 {
		t.Errorf("music.stop: port-1 address want 0x00 (ADPCM-B reset), got 0x%02X", got)
	}
	if got := emu.APU.FM.MixR; got != 0x01 {
		t.Errorf("music.stop: port-1 data want 0x01, got 0x%02X", got)
	}
}

// TestMusicPlayUnknownAssetRejected verifies referencing an undeclared music
// asset name is a compile error, not silently compiling to nothing.
func TestMusicPlayUnknownAssetRejected(t *testing.T) {
	source := `function Start()
    music.play(NotDeclared)
    while true
        wait_vblank()
`
	err := compileExpectError(t, source)
	if err == nil {
		t.Fatal("expected compile error")
	}
}
