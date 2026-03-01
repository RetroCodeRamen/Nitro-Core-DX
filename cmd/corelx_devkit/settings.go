package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

const maxRecentFiles = 15

type devKitSettings struct {
	LastSourceDir    string   `json:"last_source_dir"`
	LastROMDir       string   `json:"last_rom_dir"`
	LastOpenFile     string   `json:"last_open_file"`
	LastROMPath      string   `json:"last_rom_path"`
	ViewMode         string   `json:"view_mode"`
	LayoutPreset     string   `json:"layout_preset"`
	MainSplitOffset  float64  `json:"main_split_offset"`
	LeftSplitOffset  float64  `json:"left_split_offset"`
	DiagnosticsPanel bool     `json:"diagnostics_panel"`
	CaptureGameInput bool     `json:"capture_game_input"`
	RecentFiles      []string `json:"recent_files"`
	UIDensity        string   `json:"ui_density"`
}

func defaultDevKitSettings() devKitSettings {
	return devKitSettings{
		ViewMode:         string(viewModeFull),
		LayoutPreset:     layoutPresetBalanced,
		MainSplitOffset:  defaultMainSplitOffset,
		LeftSplitOffset:  defaultLeftSplitOffset,
		DiagnosticsPanel: true,
		CaptureGameInput: true,
		RecentFiles:      []string{},
		UIDensity:        "compact",
	}
}

func devKitSettingsPath() string {
	cfgDir, err := os.UserConfigDir()
	if err != nil || cfgDir == "" {
		return ""
	}
	return filepath.Join(cfgDir, "nitro-core-dx", "devkit_settings.json")
}

func loadDevKitSettings(path string) (devKitSettings, error) {
	settings := defaultDevKitSettings()
	if path == "" {
		return settings, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return settings, nil
		}
		return settings, err
	}
	if len(data) == 0 {
		return settings, nil
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return defaultDevKitSettings(), err
	}

	switch settings.ViewMode {
	case string(viewModeFull), string(viewModeEmulatorOnly), string(viewModeCodeOnly):
	default:
		settings.ViewMode = string(viewModeFull)
	}
	if settings.LayoutPreset == "" {
		settings.LayoutPreset = layoutPresetBalanced
	}
	if settings.MainSplitOffset <= 0 || settings.MainSplitOffset >= 1 {
		settings.MainSplitOffset = defaultMainSplitOffset
	}
	if settings.LeftSplitOffset <= 0 || settings.LeftSplitOffset >= 1 {
		settings.LeftSplitOffset = defaultLeftSplitOffset
	}
	if settings.UIDensity == "" {
		settings.UIDensity = "compact"
	}
	settings.RecentFiles = normalizeRecentFiles(settings.RecentFiles)
	return settings, nil
}

func saveDevKitSettings(path string, settings devKitSettings) error {
	if path == "" {
		return nil
	}

	settings.RecentFiles = normalizeRecentFiles(settings.RecentFiles)
	switch settings.ViewMode {
	case string(viewModeFull), string(viewModeEmulatorOnly), string(viewModeCodeOnly):
	default:
		settings.ViewMode = string(viewModeFull)
	}
	if settings.LayoutPreset == "" {
		settings.LayoutPreset = layoutPresetBalanced
	}
	if settings.MainSplitOffset <= 0 || settings.MainSplitOffset >= 1 {
		settings.MainSplitOffset = defaultMainSplitOffset
	}
	if settings.LeftSplitOffset <= 0 || settings.LeftSplitOffset >= 1 {
		settings.LeftSplitOffset = defaultLeftSplitOffset
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func normalizeRecentFiles(paths []string) []string {
	if len(paths) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(paths))
	seen := make(map[string]bool, len(paths))
	for _, p := range paths {
		if p == "" {
			continue
		}
		clean := filepath.Clean(p)
		if seen[clean] {
			continue
		}
		seen[clean] = true
		out = append(out, clean)
		if len(out) >= maxRecentFiles {
			break
		}
	}
	return out
}

func dialogListableForDir(dir string) fyne.ListableURI {
	if dir == "" {
		return nil
	}

	clean := filepath.Clean(dir)
	if st, err := os.Stat(clean); err == nil && !st.IsDir() {
		clean = filepath.Dir(clean)
	}

	listable, err := storage.ListerForURI(storage.NewFileURI(clean))
	if err != nil {
		return nil
	}
	return listable
}

func (s *devKitState) persistSettings() {
	if err := saveDevKitSettings(s.settingsPath, s.settings); err != nil {
		if s.buildOutput != nil {
			s.appendBuildOutput("Settings save warning: " + err.Error())
		} else {
			fmt.Fprintf(os.Stderr, "settings save warning: %v\n", err)
		}
	}
}

func (s *devKitState) rememberSourcePath(path string) {
	if path == "" {
		return
	}
	clean := filepath.Clean(path)
	s.settings.LastOpenFile = clean
	s.settings.LastSourceDir = filepath.Dir(clean)
	s.pushRecentFile(clean)
	s.persistSettings()
	s.refreshMainMenu()
}

func (s *devKitState) rememberROMPath(path string) {
	if path == "" {
		return
	}
	clean := filepath.Clean(path)
	s.settings.LastROMPath = clean
	s.settings.LastROMDir = filepath.Dir(clean)
	s.persistSettings()
}

func (s *devKitState) pushRecentFile(path string) {
	if path == "" {
		return
	}
	clean := filepath.Clean(path)
	next := make([]string, 0, len(s.settings.RecentFiles)+1)
	next = append(next, clean)
	for _, existing := range s.settings.RecentFiles {
		if existing == "" {
			continue
		}
		if filepath.Clean(existing) == clean {
			continue
		}
		next = append(next, existing)
		if len(next) >= maxRecentFiles {
			break
		}
	}
	s.settings.RecentFiles = next
}

func (s *devKitState) refreshMainMenu() {
	if s.window == nil {
		return
	}
	s.window.SetMainMenu(s.buildMainMenu())
}
