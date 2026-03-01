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
