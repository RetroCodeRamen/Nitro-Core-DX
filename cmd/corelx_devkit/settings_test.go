package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeRecentFilesDedupAndLimit(t *testing.T) {
	input := []string{
		"/tmp/a.corelx",
		"/tmp/b.corelx",
		"/tmp/a.corelx",
		"",
	}
	for i := 0; i < maxRecentFiles+5; i++ {
		input = append(input, filepath.Join("/tmp", "f"+string(rune('a'+(i%26)))+".corelx"))
	}

	out := normalizeRecentFiles(input)
	if len(out) == 0 {
		t.Fatalf("expected normalized recent files")
	}
	if len(out) > maxRecentFiles {
		t.Fatalf("expected at most %d entries, got %d", maxRecentFiles, len(out))
	}
	seen := map[string]bool{}
	for _, p := range out {
		if p == "" {
			t.Fatalf("unexpected empty path in normalized list")
		}
		if seen[p] {
			t.Fatalf("expected no duplicates, got %q twice", p)
		}
		seen[p] = true
	}
}

func TestLoadDevKitSettingsMissingFileReturnsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing_settings.json")
	settings, err := loadDevKitSettings(path)
	if err != nil {
		t.Fatalf("load missing settings should not fail: %v", err)
	}
	if settings.ViewMode != string(viewModeFull) {
		t.Fatalf("expected default view mode %q, got %q", viewModeFull, settings.ViewMode)
	}
	if !settings.CaptureGameInput {
		t.Fatalf("expected CaptureGameInput default true")
	}
}

func TestSaveLoadDevKitSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	in := devKitSettings{
		LastSourceDir:    "/tmp/src",
		LastROMDir:       "/tmp/roms",
		LastOpenFile:     "/tmp/src/main.corelx",
		LastROMPath:      "/tmp/roms/demo.rom",
		ViewMode:         string(viewModeEmulatorOnly),
		CaptureGameInput: false,
		RecentFiles: []string{
			"/tmp/src/main.corelx",
			"/tmp/src/other.corelx",
		},
	}

	if err := saveDevKitSettings(path, in); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected saved settings file: %v", err)
	}

	out, err := loadDevKitSettings(path)
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if out.LastSourceDir != in.LastSourceDir {
		t.Fatalf("LastSourceDir mismatch: got %q want %q", out.LastSourceDir, in.LastSourceDir)
	}
	if out.LastROMDir != in.LastROMDir {
		t.Fatalf("LastROMDir mismatch: got %q want %q", out.LastROMDir, in.LastROMDir)
	}
	if out.LastOpenFile != in.LastOpenFile {
		t.Fatalf("LastOpenFile mismatch: got %q want %q", out.LastOpenFile, in.LastOpenFile)
	}
	if out.LastROMPath != in.LastROMPath {
		t.Fatalf("LastROMPath mismatch: got %q want %q", out.LastROMPath, in.LastROMPath)
	}
	if out.ViewMode != in.ViewMode {
		t.Fatalf("ViewMode mismatch: got %q want %q", out.ViewMode, in.ViewMode)
	}
	if out.CaptureGameInput != in.CaptureGameInput {
		t.Fatalf("CaptureGameInput mismatch: got %v want %v", out.CaptureGameInput, in.CaptureGameInput)
	}
	if len(out.RecentFiles) != len(in.RecentFiles) {
		t.Fatalf("RecentFiles length mismatch: got %d want %d", len(out.RecentFiles), len(in.RecentFiles))
	}
}

func TestLoadDevKitSettingsInvalidViewModeFallsBackToDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings_invalid.json")
	raw := []byte(`{"view_mode":"invalid_mode","capture_game_input":true}`)
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatalf("write settings fixture: %v", err)
	}

	out, err := loadDevKitSettings(path)
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if out.ViewMode != string(viewModeFull) {
		t.Fatalf("expected fallback view mode %q, got %q", viewModeFull, out.ViewMode)
	}
}

func TestResolveDefaultROMDirPrefersLastROMDir(t *testing.T) {
	root := t.TempDir()
	lastROM := filepath.Join(root, "last-rom")
	launch := filepath.Join(root, "launch")
	home := filepath.Join(root, "home")
	for _, p := range []string{lastROM, launch, home} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(launch, "roms"), 0755); err != nil {
		t.Fatalf("mkdir launch/roms: %v", err)
	}

	got := resolveDefaultROMDir(lastROM, launch, home)
	if got != filepath.Clean(lastROM) {
		t.Fatalf("got %q, want %q", got, filepath.Clean(lastROM))
	}
}

func TestResolveDefaultROMDirFallsBackToLaunchRomsThenHome(t *testing.T) {
	root := t.TempDir()
	launch := filepath.Join(root, "launch")
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(filepath.Join(launch, "roms"), 0755); err != nil {
		t.Fatalf("mkdir launch/roms: %v", err)
	}
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	got := resolveDefaultROMDir("", launch, home)
	if want := filepath.Join(launch, "roms"); got != filepath.Clean(want) {
		t.Fatalf("got %q, want %q", got, filepath.Clean(want))
	}

	if err := os.RemoveAll(filepath.Join(launch, "roms")); err != nil {
		t.Fatalf("remove launch/roms: %v", err)
	}
	got = resolveDefaultROMDir("", launch, home)
	if got != filepath.Clean(home) {
		t.Fatalf("got %q, want %q", got, filepath.Clean(home))
	}
}

func TestDefaultProjectOpenDirPrefersSourceDir(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	launch := filepath.Join(root, "launch")
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(launch, "roms"), 0755); err != nil {
		t.Fatalf("mkdir launch/roms: %v", err)
	}
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	// Resolve without relying on environment-dependent UserHomeDir.
	got := resolveDefaultROMDir("", launch, home)
	if want := filepath.Join(launch, "roms"); got != filepath.Clean(want) {
		t.Fatalf("got %q, want %q", got, filepath.Clean(want))
	}

	s := &devKitState{
		launchDir: launch,
		settings: devKitSettings{
			LastSourceDir: source,
		},
	}
	if gotSource := s.defaultProjectOpenDir(); gotSource != filepath.Clean(source) {
		t.Fatalf("project open dir got %q, want %q", gotSource, filepath.Clean(source))
	}
}
