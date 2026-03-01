package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2/dialog"
)

type autosaveJournal struct {
	SourcePath string `json:"source_path"`
	SavedAt    string `json:"saved_at"`
	Content    string `json:"content"`
}

func devKitAutosavePath(settingsPath string) string {
	if settingsPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(settingsPath), "devkit_autosave.json")
}

func (s *devKitState) writeAutosaveSnapshot(content string) {
	if s.autosavePath == "" || !s.dirty {
		return
	}
	if strings.TrimSpace(content) == "" {
		return
	}

	journal := autosaveJournal{
		SourcePath: s.currentPath,
		SavedAt:    time.Now().UTC().Format(time.RFC3339),
		Content:    content,
	}
	data, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.autosavePath), 0755); err != nil {
		return
	}
	_ = os.WriteFile(s.autosavePath, data, 0644)
}

func (s *devKitState) clearAutosaveJournal() {
	if s.autosavePath == "" {
		return
	}
	_ = os.Remove(s.autosavePath)
}

func (s *devKitState) tryRecoverAutosave() {
	if s.autosavePath == "" {
		return
	}
	data, err := os.ReadFile(s.autosavePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		s.appendBuildOutput("Autosave recover warning: " + err.Error())
		return
	}
	if len(data) == 0 {
		s.clearAutosaveJournal()
		return
	}

	var journal autosaveJournal
	if err := json.Unmarshal(data, &journal); err != nil {
		s.appendBuildOutput("Autosave recover warning: invalid journal format")
		s.clearAutosaveJournal()
		return
	}
	if strings.TrimSpace(journal.Content) == "" {
		s.clearAutosaveJournal()
		return
	}
	if journal.Content == s.sourceEntry.Text {
		s.clearAutosaveJournal()
		return
	}

	savedAt := journal.SavedAt
	if savedAt == "" {
		savedAt = "unknown time"
	}
	sourceRef := journal.SourcePath
	if sourceRef == "" {
		sourceRef = "unsaved buffer"
	}
	msg := fmt.Sprintf("An autosave journal from %s was found for %s.\n\nRecover it now?", savedAt, sourceRef)
	dialog.NewConfirm("Recover Autosave", msg, func(recover bool) {
		if !recover {
			s.clearAutosaveJournal()
			return
		}
		s.setSourceContent(journal.Content, true, false)
		if journal.SourcePath != "" {
			s.currentPath = journal.SourcePath
			s.pathLabel.SetText(displayPath(s.currentPath))
		}
		s.refreshTitle()
		s.setStatus("Recovered autosave journal")
		s.appendBuildOutput("Recovered autosave journal")
	}, s.window).Show()
}
