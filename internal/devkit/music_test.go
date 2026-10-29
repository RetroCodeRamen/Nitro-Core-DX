package devkit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nitro-core-dx/internal/apu"
	"nitro-core-dx/internal/emulator"
	"nitro-core-dx/internal/ymstream"
)

func testMusicBytes(t *testing.T) []byte {
	t.Helper()
	song := &ymstream.Song{
		Frames:       [][]ymstream.Write{{{Port: 0, Addr: 0x10, Data: 0xAA}}},
		FrameSamples: 735,
		WriteCount:   1,
	}
	data, err := ymstream.EncodeSong(song)
	if err != nil {
		t.Fatalf("EncodeSong: %v", err)
	}
	return data
}

func TestUpsertMusicAssetLine(t *testing.T) {
	src := "function Start()\n    return\n"
	next, state, err := UpsertMusicAssetLine(src, "Theme", "Theme.ncdxmusic")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if state != "new" {
		t.Fatalf("state = %q, want new", state)
	}
	if !strings.Contains(next, "asset Theme: music \"Theme.ncdxmusic\"") {
		t.Fatalf("missing declaration:\n%s", next)
	}
	if strings.Count(next, "asset Theme:") != 1 {
		t.Fatalf("declaration count = %d", strings.Count(next, "asset Theme:"))
	}

	repl, state, err := UpsertMusicAssetLine(next, "Theme", "Theme.ncdxmusic")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if state != "updated" {
		t.Fatalf("state = %q, want updated", state)
	}
	if strings.Count(repl, "asset Theme: music") != 1 {
		t.Fatalf("expected a single music declaration:\n%s", repl)
	}

	if _, _, err := UpsertMusicAssetLine("asset Theme: sprite hex\n    00\n", "Theme", "Theme.ncdxmusic"); err == nil {
		t.Fatal("expected a conflict with an existing non-music asset")
	}
}

func TestMusicSnippetsCompileShape(t *testing.T) {
	snippets := MusicSnippets("Theme")
	if len(snippets) != 6 {
		t.Fatalf("snippets = %d, want 6", len(snippets))
	}
	want := []string{
		"music.play(Theme)",
		"music.play_loop(Theme)",
		"music.play_jingle(Theme)",
		"music.stop()",
		"music.set_volume(255)",
		"music.fade_to(0, 60)",
	}
	for i, snip := range snippets {
		if snip.Code != want[i] {
			t.Fatalf("snippet %d = %q, want %q", i, snip.Code, want[i])
		}
	}
}

func TestApplyMusicAssetCompiles(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "main.corelx")
	original := "function Start()\n    while true\n        wait_vblank()\n"
	applied, err := ApplyMusicAsset(sourcePath, original, "My Song", testMusicBytes(t))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if applied.Name != "My_Song" || applied.FileName != "My_Song.ncdxmusic" {
		t.Fatalf("name/file = %s / %s", applied.Name, applied.FileName)
	}
	if applied.State != "new" {
		t.Fatalf("state = %q", applied.State)
	}
	onDisk, err := os.ReadFile(applied.FilePath)
	if err != nil {
		t.Fatalf("read music file: %v", err)
	}
	if len(onDisk) == 0 {
		t.Fatal("music file is empty")
	}

	svc := NewService(t.TempDir())
	build, err := svc.BuildSource(applied.Source, sourcePath)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if build.Result == nil || build.Result.Manifest == nil {
		t.Fatal("expected a manifest")
	}
	found := false
	for _, asset := range build.Result.Manifest.Assets {
		if asset.Name == "My_Song" && asset.Kind == "music" {
			found = true
		}
	}
	if !found {
		t.Fatalf("manifest assets = %+v", build.Result.Manifest.Assets)
	}
}

func TestApplyMusicAssetRequiresSavedSource(t *testing.T) {
	if _, err := ApplyMusicAsset("", "function Start()\n", "Theme", testMusicBytes(t)); err == nil {
		t.Fatal("expected an error when the source path is empty")
	}
}

func TestBuildMusicPreviewWritesYM(t *testing.T) {
	svc := NewService(t.TempDir())
	build, err := svc.BuildMusicPreview(testMusicBytes(t), "Theme")
	if err != nil {
		t.Fatalf("preview build: %v", err)
	}
	if build.Result == nil || len(build.Result.ROMBytes) == 0 {
		t.Fatal("expected preview ROM bytes")
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(build.Result.ROMBytes); err != nil {
		t.Fatalf("load: %v", err)
	}
	emu.Start()
	if err := emu.RunFrame(); err != nil {
		t.Fatalf("frame: %v", err)
	}
	if got := emu.APU.FM.Addr; got != 0x10 {
		t.Fatalf("FM addr = 0x%02X, want 0x10", got)
	}
	if got := emu.APU.FM.Read8(apu.FMRegData); got != 0xAA {
		t.Fatalf("FM data = 0x%02X, want 0xAA", got)
	}
}
