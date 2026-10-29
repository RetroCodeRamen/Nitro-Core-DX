package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"nitro-core-dx/internal/devkit"
	"nitro-core-dx/internal/ymstream"
)

func (s *devKitState) buildSoundStudioPane() fyne.CanvasObject {
	var song []byte

	statusLabel := widget.NewLabel("Import a .vgm/.vgz file or open a .ncdxmusic file.")
	reportLabel := widget.NewLabel("No song loaded.")
	reportLabel.Wrapping = fyne.TextWrapWord

	nameEntry := widget.NewEntry()
	nameEntry.SetText("Theme")
	nameEntry.SetPlaceHolder("Asset name")

	showErr := func(err error) {
		if err == nil {
			return
		}
		dialog.ShowError(err, s.window)
		statusLabel.SetText(err.Error())
	}

	setSong := func(data []byte, report ymstream.Report, suggestedName string) {
		song = append([]byte(nil), data...)
		if strings.TrimSpace(suggestedName) != "" {
			nameEntry.SetText(suggestedName)
		}
		reportLabel.SetText(report.String())
		statusLabel.SetText(fmt.Sprintf("Loaded %s (%d frames)", nameEntry.Text, report.FrameCount))
	}

	importButton := widget.NewButton("Import VGM", func() {
		fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
			if err != nil {
				showErr(err)
				return
			}
			if rc == nil {
				return
			}
			defer rc.Close()
			data, readErr := io.ReadAll(rc)
			if readErr != nil {
				showErr(readErr)
				return
			}
			encoded, report, convErr := ymstream.ConvertVGM(data)
			if convErr != nil {
				showErr(convErr)
				return
			}
			setSong(encoded, report, musicNameFromFile(uriPath(rc.URI())))
			path := uriPath(rc.URI())
			if path != "" {
				s.settings.LastSourceDir = filepath.Dir(path)
				s.persistSettings()
			}
		}, s.window)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".vgm", ".vgz"}))
		if loc := dialogListableForDir(s.settings.LastSourceDir); loc != nil {
			fd.SetLocation(loc)
		}
		fd.Show()
	})

	openButton := widget.NewButton("Open .ncdxmusic", func() {
		fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
			if err != nil {
				showErr(err)
				return
			}
			if rc == nil {
				return
			}
			defer rc.Close()
			data, readErr := io.ReadAll(rc)
			if readErr != nil {
				showErr(readErr)
				return
			}
			decoded, decErr := ymstream.DecodeStream(data)
			if decErr != nil {
				showErr(decErr)
				return
			}
			report, inspectErr := ymstream.Inspect(decoded, len(data))
			if inspectErr != nil {
				showErr(inspectErr)
				return
			}
			setSong(data, report, musicNameFromFile(uriPath(rc.URI())))
			path := uriPath(rc.URI())
			if path != "" {
				s.settings.LastSourceDir = filepath.Dir(path)
				s.persistSettings()
			}
		}, s.window)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".ncdxmusic"}))
		if loc := dialogListableForDir(s.settings.LastSourceDir); loc != nil {
			fd.SetLocation(loc)
		}
		fd.Show()
	})

	addButton := widget.NewButton("Add to project", func() {
		if len(song) == 0 {
			showErr(fmt.Errorf("import or open a song first"))
			return
		}
		if strings.TrimSpace(s.currentPath) == "" {
			showErr(fmt.Errorf("save the project source file before adding a music asset"))
			return
		}
		applied, err := devkit.ApplyMusicAsset(s.currentPath, s.sourceEditor.Text(), nameEntry.Text, song)
		if err != nil {
			showErr(err)
			return
		}
		nameEntry.SetText(applied.Name)
		s.setSourceContent(applied.Source, true, false)
		if s.workbenchTabs != nil {
			s.workbenchTabs.SelectIndex(0)
		}
		s.setStatus(fmt.Sprintf("Music asset %s (%s)", applied.Name, applied.State))
		s.appendBuildOutput(fmt.Sprintf("Sound Studio %s %s -> %s", applied.State, applied.Name, applied.FilePath))
		statusLabel.SetText(fmt.Sprintf("Added %s (%s). Save the source to keep the asset line.", applied.FileName, applied.State))
	})
	addButton.Importance = widget.MediumImportance

	previewButton := widget.NewButton("Preview", func() {
		if len(song) == 0 {
			showErr(fmt.Errorf("import or open a song first"))
			return
		}
		build, err := s.backend.BuildMusicPreview(song, nameEntry.Text)
		if err != nil {
			if build != nil && len(build.Bundle.Diagnostics) > 0 {
				showErr(fmt.Errorf("%s", build.Bundle.Diagnostics[0].Message))
				return
			}
			showErr(err)
			return
		}
		if build == nil || build.Result == nil || len(build.Result.ROMBytes) == 0 {
			showErr(fmt.Errorf("preview produced no ROM"))
			return
		}
		if loadErr := s.loadROMIntoEmbedded(build.Result.ROMBytes); loadErr != nil {
			showErr(loadErr)
			return
		}
		s.setViewMode(viewModeFull)
		s.setStatus("Preview playing in the emulator")
		s.appendBuildOutput("Sound Studio preview loaded into the emulator")
		statusLabel.SetText("Preview replaced the emulator ROM. Build + Run returns to the project.")
	})
	previewButton.Importance = widget.MediumImportance

	pauseButton := widget.NewButton("Pause preview", func() {
		snap := s.backend.Snapshot()
		s.pauseEmulator()
		if snap.Loaded && !snap.Paused {
			statusLabel.SetText("Preview paused")
		}
	})

	snippetButtons := make([]fyne.CanvasObject, 0, 6)
	for _, snip := range devkit.MusicSnippets("Theme") {
		label := snip.Label
		snippetButtons = append(snippetButtons, widget.NewButton(label, func() {
			name := strings.TrimSpace(nameEntry.Text)
			sanitized, err := devkit.SanitizeMusicAssetName(name)
			if err != nil || sanitized != name {
				showErr(fmt.Errorf("asset name must be a CoreLX identifier"))
				return
			}
			code := ""
			for _, item := range devkit.MusicSnippets(sanitized) {
				if item.Label == label {
					code = item.Code
					break
				}
			}
			if code == "" {
				return
			}
			s.sourceEditor.InsertAtCursor(code + "\n")
			if s.workbenchTabs != nil {
				s.workbenchTabs.SelectIndex(0)
			}
			s.setStatus("Inserted " + code)
			statusLabel.SetText("Inserted " + code)
		}))
	}

	nameRow := container.NewBorder(nil, nil, widget.NewLabel("Asset name"), nil, nameEntry)
	form := container.NewVBox(
		panelHeader("SOUND"),
		statusLabel,
		widget.NewLabelWithStyle("ASSET", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nameRow,
		container.NewGridWrap(fyne.NewSize(160, 36), importButton, openButton, addButton),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("PREVIEW", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		reportLabel,
		container.NewHBox(previewButton, pauseButton),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("INSERT", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWrap(fyne.NewSize(168, 36), snippetButtons...),
	)
	return container.NewVScroll(container.NewPadded(form))
}

func musicNameFromFile(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	name, err := devkit.SanitizeMusicAssetName(base)
	if err != nil {
		return "Theme"
	}
	return name
}
