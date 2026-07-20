package corelx

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nitro-core-dx/internal/ymstream"
)

// tinyNcdxmusic returns a minimal valid .ncdxmusic stream (one frame, one YM
// write) so tests don't need a committed binary fixture.
func tinyNcdxmusic(t *testing.T) []byte {
	t.Helper()
	song := &ymstream.Song{
		Frames:       [][]ymstream.Write{{{Port: 0, Addr: 0x28, Data: 0xF0}}},
		FrameSamples: 735,
		WriteCount:   1,
	}
	data, err := ymstream.EncodeSong(song)
	if err != nil {
		t.Fatalf("EncodeSong: %v", err)
	}
	return data
}

func writeProject(t *testing.T, files map[string][]byte) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, "main.corelx")
}

// A valid folder project with a referenced .ncdxmusic compiles, and the music
// bytes land in the ROM (the .cart grows by at least the stream size).
func TestMusicAssetCompiles(t *testing.T) {
	music := tinyNcdxmusic(t)
	main := `asset Theme: music "theme.ncdxmusic"
function Start()
    while true
        wait_vblank()
`
	mainPath := writeProject(t, map[string][]byte{
		"main.corelx":     []byte(main),
		"theme.ncdxmusic": music,
	})
	out := filepath.Join(filepath.Dir(mainPath), "out.cart")
	if _, err := CompileProject(mainPath, &CompileOptions{OutputPath: out}); err != nil {
		t.Fatalf("compile with music asset: %v", err)
	}
	info, err := os.Stat(out)
	if err != nil || info.Size() < int64(len(music)) {
		t.Fatalf("expected .cart at least as large as the music stream (%d bytes), got %v (%v)", len(music), info, err)
	}
}

// A reference to a missing .ncdxmusic is a blocking error.
func TestMusicAssetMissingFails(t *testing.T) {
	mainPath := writeProject(t, map[string][]byte{
		"main.corelx": []byte("asset Theme: music \"nope.ncdxmusic\"\nfunction Start()\n    while true\n        wait_vblank()\n"),
	})
	_, err := CompileProject(mainPath, &CompileOptions{OutputPath: filepath.Join(filepath.Dir(mainPath), "o.cart")})
	if err == nil {
		t.Fatal("expected error for missing music file")
	}
}

// A file with an invalid YM stream header is a blocking error.
func TestMusicAssetInvalidFails(t *testing.T) {
	mainPath := writeProject(t, map[string][]byte{
		"main.corelx":     []byte("asset Theme: music \"theme.ncdxmusic\"\nfunction Start()\n    while true\n        wait_vblank()\n"),
		"theme.ncdxmusic": []byte("NOT A YM STREAM"),
	})
	_, err := CompileProject(mainPath, &CompileOptions{OutputPath: filepath.Join(filepath.Dir(mainPath), "o.cart")})
	if err == nil || !strings.Contains(err.Error(), "invalid .ncdxmusic") {
		t.Fatalf("expected invalid-stream error, got: %v", err)
	}
}

// An unreferenced .ncdxmusic in the project is an orphan error.
func TestMusicAssetOrphanFails(t *testing.T) {
	mainPath := writeProject(t, map[string][]byte{
		"main.corelx":     []byte("function Start()\n    while true\n        wait_vblank()\n"),
		"theme.ncdxmusic": tinyNcdxmusic(t),
	})
	_, err := CompileProject(mainPath, &CompileOptions{OutputPath: filepath.Join(filepath.Dir(mainPath), "o.cart")})
	if err == nil || !strings.Contains(err.Error(), "orphan") {
		t.Fatalf("expected orphan error, got: %v", err)
	}
}

// A .ncdx container with a music asset compiles (container handling is generic).
func TestNcdxContainerWithMusic(t *testing.T) {
	dir := t.TempDir()
	ncdx := filepath.Join(dir, "Game.ncdx")
	zf, _ := os.Create(ncdx)
	zw := zip.NewWriter(zf)
	files := map[string][]byte{
		"main.corelx":     []byte("asset Theme: music \"theme.ncdxmusic\"\nfunction Start()\n    while true\n        wait_vblank()\n"),
		"theme.ncdxmusic": tinyNcdxmusic(t),
		"project.toml":    []byte("title = \"Game\"\n"),
	}
	for name, content := range files {
		w, _ := zw.Create(name)
		w.Write(content)
	}
	zw.Close()
	zf.Close()
	if _, err := CompileProject(ncdx, &CompileOptions{OutputPath: filepath.Join(dir, "out.cart")}); err != nil {
		t.Fatalf("compile .ncdx with music: %v", err)
	}
}
