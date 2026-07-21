package corelx

import (
	"os"
	"path/filepath"
	"testing"

	"nitro-core-dx/internal/apu"
	"nitro-core-dx/internal/emulator"
	"nitro-core-dx/internal/ymstream"
)

// twoFrameNcdxmusicSong returns a 2-frame .ncdxmusic stream with the given
// distinguishable per-frame writes.
func twoFrameNcdxmusicSong(t *testing.T, f0addr, f0data, f1addr, f1data uint8) []byte {
	t.Helper()
	song := &ymstream.Song{
		Frames: [][]ymstream.Write{
			{{Port: 0, Addr: f0addr, Data: f0data}},
			{{Port: 0, Addr: f1addr, Data: f1data}},
		},
		FrameSamples: 735,
		WriteCount:   2,
	}
	data, err := ymstream.EncodeSong(song)
	if err != nil {
		t.Fatalf("EncodeSong: %v", err)
	}
	return data
}

// compileAndRunFramesWithTwoMusicAssets compiles a project with two music
// asset files and boots it, running the given number of real PPU frames.
func compileAndRunFramesWithTwoMusicAssets(t *testing.T, bgmData, jingleData []byte, mainSource string, frames int) *emulator.Emulator {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bgm.ncdxmusic"), bgmData, 0644); err != nil {
		t.Fatalf("write bgm asset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "jingle.ncdxmusic"), jingleData, 0644); err != nil {
		t.Fatalf("write jingle asset: %v", err)
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

const jingleMainSource = `asset Bgm: music "bgm.ncdxmusic"
asset Jingle: music "jingle.ncdxmusic"

function Start()
    music.play_loop(Bgm)
    wait_vblank()
    music.play_jingle(Jingle)
    while true
        wait_vblank()
`

// TestMusicPlayJingleThenResumesBGM verifies play_jingle plays its own song
// to completion, then resumes the BGM exactly where it left off (not
// restarted), and that the BGM keeps looping normally afterward.
func TestMusicPlayJingleThenResumesBGM(t *testing.T) {
	bgm := twoFrameNcdxmusicSong(t, 0x20, 0xAA, 0x21, 0xBB)
	jingle := twoFrameNcdxmusicSong(t, 0x30, 0xCC, 0x31, 0xDD)

	cases := []struct {
		frames   int
		wantAddr uint8
		wantData uint8
		desc     string
	}{
		{1, 0x20, 0xAA, "BGM frame 0 before the jingle starts"},
		{2, 0x30, 0xCC, "jingle frame 0"},
		{3, 0x31, 0xDD, "jingle frame 1 (its last)"},
		{4, 0x21, 0xBB, "BGM resumes at its frame 1, not restarted at frame 0"},
		{5, 0x20, 0xAA, "BGM loops back to frame 0 normally"},
	}
	for _, c := range cases {
		emu := compileAndRunFramesWithTwoMusicAssets(t, bgm, jingle, jingleMainSource, c.frames)
		if got := emu.APU.FM.Addr; got != c.wantAddr {
			t.Errorf("%s (after %d frames): FM addr want 0x%02X, got 0x%02X", c.desc, c.frames, c.wantAddr, got)
		}
		if got := emu.APU.FM.Read8(apu.FMRegData); got != c.wantData {
			t.Errorf("%s (after %d frames): FM data want 0x%02X, got 0x%02X", c.desc, c.frames, c.wantData, got)
		}
	}
}

// TestMusicPlayJingleWithNoBGMStopsAfter verifies play_jingle with nothing
// playing beforehand just plays the jingle, then silences and stops
// (nothing to resume) rather than erroring or looping.
func TestMusicPlayJingleWithNoBGMStopsAfter(t *testing.T) {
	jingle := twoFrameNcdxmusicSong(t, 0x30, 0xCC, 0x31, 0xDD)
	mainSource := `asset Jingle: music "jingle.ncdxmusic"

function Start()
    music.play_jingle(Jingle)
    while true
        wait_vblank()
`
	// Frame 1: jingle frame 0. Frame 2: jingle frame 1 (last) -> reaches end,
	// restores "nothing playing" (saved mode 0), silences.
	emu := compileAndRunFramesWithMusic(t, "jingle.ncdxmusic", jingle, mainSource, 2)
	if got := emu.APU.FM.MixL; got != 0x00 {
		t.Errorf("jingle end with no BGM: port-1 address want 0x00 (ADPCM-B reset from silence), got 0x%02X", got)
	}
	if got := emu.APU.FM.MixR; got != 0x01 {
		t.Errorf("jingle end with no BGM: port-1 data want 0x01, got 0x%02X", got)
	}
}
