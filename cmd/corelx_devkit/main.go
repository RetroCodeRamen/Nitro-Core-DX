package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/veandco/go-sdl2/sdl"
	"nitro-core-dx/internal/apu"
	"nitro-core-dx/internal/corelx"
	"nitro-core-dx/internal/devkit"
)

const (
	devKitScreenW          = 320
	devKitScreenH          = 200
	defaultWindowWidth     = 1280
	defaultWindowHeight    = 760
	defaultMainSplitOffset = 0.40
	defaultLeftSplitOffset = 0.60
)

const defaultTemplate = `function Start()
    apu.enable()
`

type viewMode string

const (
	viewModeFull         viewMode = "full"
	viewModeEmulatorOnly viewMode = "emulator_only"
	viewModeCodeOnly     viewMode = "code_only"
)

const (
	layoutPresetBalanced      = "balanced"
	layoutPresetCodeFocus     = "code_focus"
	layoutPresetArtMode       = "art_mode"
	layoutPresetDebugMode     = "debug_mode"
	layoutPresetEmulatorFocus = "emulator_focus"
)

type emulatorKeyOverlay struct {
	widget.BaseWidget
	onTap     func()
	onTyped   func(*fyne.KeyEvent)
	onKeyDown func(*fyne.KeyEvent)
	onKeyUp   func(*fyne.KeyEvent)
}

func newEmulatorKeyOverlay(onTap func(), onTyped, onKeyDown, onKeyUp func(*fyne.KeyEvent)) *emulatorKeyOverlay {
	w := &emulatorKeyOverlay{
		onTap:     onTap,
		onTyped:   onTyped,
		onKeyDown: onKeyDown,
		onKeyUp:   onKeyUp,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *emulatorKeyOverlay) CreateRenderer() fyne.WidgetRenderer {
	rect := canvas.NewRectangle(color.Transparent)
	return widget.NewSimpleRenderer(rect)
}

func (w *emulatorKeyOverlay) Tapped(*fyne.PointEvent) {
	if w.onTap != nil {
		w.onTap()
	}
}

func (w *emulatorKeyOverlay) TappedSecondary(*fyne.PointEvent) {}
func (w *emulatorKeyOverlay) FocusGained()                     {}
func (w *emulatorKeyOverlay) FocusLost()                       {}
func (w *emulatorKeyOverlay) TypedRune(r rune)                 {}

func (w *emulatorKeyOverlay) TypedKey(ev *fyne.KeyEvent) {
	if w.onTyped != nil {
		w.onTyped(ev)
	}
}

func (w *emulatorKeyOverlay) KeyDown(ev *fyne.KeyEvent) {
	if w.onKeyDown != nil {
		w.onKeyDown(ev)
	}
}

func (w *emulatorKeyOverlay) KeyUp(ev *fyne.KeyEvent) {
	if w.onKeyUp != nil {
		w.onKeyUp(ev)
	}
}

type devKitState struct {
	backend *devkit.Service

	tempDir      string
	settingsPath string
	settings     devKitSettings

	currentPath  string
	lastROMPath  string
	autosavePath string
	dirty        bool

	diagnostics         []corelx.Diagnostic
	filteredDiagnostics []corelx.Diagnostic

	window         fyne.Window
	centerHost     *fyne.Container
	contentRoot    *fyne.Container
	currentView    viewMode
	statusLabel    *widget.Label
	pathLabel      *widget.Label
	sourceEntry    *widget.Entry
	buildOutput    *widget.Entry
	manifestOutput *widget.Entry
	debuggerOutput *widget.Entry

	diagnosticFilter *widget.Select
	diagnosticSearch *widget.Entry
	diagnosticsList  *widget.List
	diagnosticDetail *widget.Entry
	diagnosticSummary *widget.Label
	diagnosticsToggle *widget.Button
	stepFrameEntry   *widget.Entry
	stepCPUEntry     *widget.Entry

	emuSurface     fyne.CanvasObject
	captureCheck   *widget.Check
	bottomLeftTabs *container.AppTabs
	leftSplit      *container.Split
	mainSplit      *container.Split
	editorPane     fyne.CanvasObject
	workbenchTabs  *container.AppTabs
	splitViewBtn       *widget.Button
	emulatorFocusBtn   *widget.Button
	codeOnlyBtn        *widget.Button
	runBtn             *widget.Button
	pauseBtn           *widget.Button
	stopBtn            *widget.Button

	emuScale   int
	emuImage   *canvas.Image
	emuKeys    *emulatorKeyOverlay
	emuLabel   *widget.Label
	audioDev   sdl.AudioDeviceID
	audioFrame []byte

	frameImages [2]*image.RGBA
	frameIdx    int

	updateLoopStop chan struct{}
	updateLoopOnce sync.Once

	keyMu            sync.Mutex
	keyStates        map[fyne.KeyName]bool
	typedKeyUntil    map[fyne.KeyName]time.Time
	desktopKeyEvents bool
	captureGameInput bool

	suppressSourceChange bool
	diagnosticsCollapsed bool

	// programmatic maximize workaround when WM title-bar Maximize is greyed out
	savedRestoreSize fyne.Size
	windowMaximized  bool
}

func main() {
	openPath := flag.String("file", "", "CoreLX source file to open")
	flag.Parse()

	tempDir, err := os.MkdirTemp("", "nitro-core-dx-devkit-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	a := app.New()

	settingsPath := devKitSettingsPath()
	settings, settingsErr := loadDevKitSettings(settingsPath)

	if settings.UIDensity == "standard" {
		a.Settings().SetTheme(newStandardTheme())
	} else {
		a.Settings().SetTheme(newCompactTheme())
	}

	w := a.NewWindow("Nitro-Core-DX")
	w.SetFixedSize(false)
	w.Resize(fyne.NewSize(defaultWindowWidth, defaultWindowHeight))

	initialView := viewModeFull
	switch settings.ViewMode {
	case string(viewModeEmulatorOnly):
		initialView = viewModeEmulatorOnly
	case string(viewModeCodeOnly):
		initialView = viewModeCodeOnly
	}

	state := &devKitState{
		tempDir:             tempDir,
		settingsPath:        settingsPath,
		settings:            settings,
		autosavePath:        devKitAutosavePath(settingsPath),
		window:              w,
		currentView:         initialView,
		statusLabel:         widget.NewLabel("Ready"),
		pathLabel:           widget.NewLabel("Untitled.corelx"),
		diagnostics:         make([]corelx.Diagnostic, 0),
		filteredDiagnostics: make([]corelx.Diagnostic, 0),
		buildOutput:         newReadOnlyTextArea(),
		manifestOutput:      newReadOnlyTextArea(),
		diagnosticDetail:    newReadOnlyTextArea(),
		emuScale:            2,
		keyStates:           make(map[fyne.KeyName]bool),
		typedKeyUntil:       make(map[fyne.KeyName]time.Time),
		captureGameInput:    settings.CaptureGameInput,
		updateLoopStop:      make(chan struct{}),
		audioFrame:          make([]byte, 735*2*4),
		diagnosticsCollapsed: !settings.DiagnosticsPanel,
	}
	if settingsErr != nil {
		fmt.Fprintf(os.Stderr, "settings load warning: %v\n", settingsErr)
	}
	state.backend = devkit.NewService(tempDir)
	if err := state.initAudio(); err != nil {
		state.appendBuildOutput("Audio init warning: " + err.Error())
		state.setStatus("Ready (audio unavailable)")
	}
	state.initUI()
	state.window.SetMainMenu(state.buildMainMenu())
	state.setupKeyboardInput()
	state.startEmulatorLoop()

	if *openPath != "" {
		if err := state.loadFile(*openPath, true); err != nil {
			state.appendBuildOutput(fmt.Sprintf("Open error: %v", err))
			state.setStatus("Open failed")
		}
	} else if state.settings.LastOpenFile != "" {
		if err := state.loadFile(state.settings.LastOpenFile, false); err != nil {
			state.appendBuildOutput(fmt.Sprintf("Session restore warning: %v", err))
		}
	} else {
		state.dirty = false
		state.refreshTitle()
	}
	if *openPath == "" {
		state.tryRecoverAutosave()
	}
	if state.settings.LastROMPath != "" {
		state.lastROMPath = state.settings.LastROMPath
	}

	w.SetCloseIntercept(func() {
		state.captureLayoutState()
		state.writeAutosaveSnapshot(state.sourceEntry.Text)
		state.stopEmulatorLoop()
		state.shutdownEmbeddedEmulator()
		state.shutdownAudio()
		state.persistSettings()
		w.Close()
	})

	go func() {
		time.Sleep(300 * time.Millisecond)
		fyne.Do(func() {
			if w != nil {
				w.SetFixedSize(false)
				_ = applyX11MaximizeHint(w)
			}
		})
	}()

	w.ShowAndRun()
}

func (s *devKitState) initUI() {
	s.sourceEntry = widget.NewMultiLineEntry()
	s.sourceEntry.SetText(defaultTemplate)
	s.sourceEntry.Wrapping = fyne.TextWrapOff
	s.sourceEntry.OnChanged = func(text string) {
		if s.suppressSourceChange {
			return
		}
		s.dirty = true
		s.refreshTitle()
		s.writeAutosaveSnapshot(text)
	}

	s.diagnosticFilter = widget.NewSelect([]string{"All", "Errors", "Warnings", "Info"}, func(string) {
		s.applyDiagnosticFilter()
	})
	s.diagnosticSearch = widget.NewEntry()
	s.diagnosticSearch.SetPlaceHolder("Search diagnostics")
	s.diagnosticSearch.OnChanged = func(string) {
		s.applyDiagnosticFilter()
	}
	s.diagnosticFilter.SetSelected("All")
	s.stepFrameEntry = widget.NewEntry()
	s.stepFrameEntry.SetText("1")
	s.stepCPUEntry = widget.NewEntry()
	s.stepCPUEntry.SetText("1")
	s.debuggerOutput = newReadOnlyTextArea()

	s.diagnosticsList = widget.NewList(
		func() int { return len(s.filteredDiagnostics) },
		func() fyne.CanvasObject {
			return canvas.NewText("diagnostic", theme.ForegroundColor())
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			lbl := obj.(*canvas.Text)
			if id < 0 || id >= len(s.filteredDiagnostics) {
				lbl.Text = ""
				lbl.Refresh()
				return
			}
			d := s.filteredDiagnostics[id]
			loc := ""
			if d.Line > 0 {
				loc = fmt.Sprintf(":%d:%d", d.Line, maxInt(1, d.Column))
			}
			lbl.Text = fmt.Sprintf("[%s/%s] %s%s %s", d.Severity, d.Stage, baseNameOr(d.File, "<buffer>"), loc, d.Message)
			switch d.Severity {
			case corelx.SeverityError:
				lbl.Color = theme.ErrorColor()
			case corelx.SeverityWarning:
				lbl.Color = theme.WarningColor()
			default:
				lbl.Color = theme.ForegroundColor()
			}
			lbl.Refresh()
		},
	)
	s.diagnosticsList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(s.filteredDiagnostics) {
			return
		}
		d := s.filteredDiagnostics[id]
		s.showDiagnosticDetail(d)
		s.jumpToDiagnostic(d)
	}

	// Embedded emulator display (double-buffered RGBA image, same hot path strategy as FyneUI).
	frame0 := image.NewRGBA(image.Rect(0, 0, devKitScreenW*s.emuScale, devKitScreenH*s.emuScale))
	frame1 := image.NewRGBA(image.Rect(0, 0, devKitScreenW*s.emuScale, devKitScreenH*s.emuScale))
	s.frameImages = [2]*image.RGBA{frame0, frame1}
	s.emuImage = canvas.NewImageFromImage(frame0)
	s.emuImage.FillMode = canvas.ImageFillContain
	s.emuKeys = newEmulatorKeyOverlay(
		func() { s.focusEmulatorInput() },
		func(key *fyne.KeyEvent) { s.handleTypedKey(key) },
		func(key *fyne.KeyEvent) { s.handleKeyDown(key) },
		func(key *fyne.KeyEvent) { s.handleKeyUp(key) },
	)
	s.emuLabel = widget.NewLabel("Hardware: idle")
	s.emuSurface = container.NewStack(s.emuImage, s.emuKeys)

	s.captureCheck = widget.NewCheck("Capture Input", func(v bool) {
		s.captureGameInput = v
		if !v {
			s.applyInputButtons(0)
		}
		s.settings.CaptureGameInput = v
		s.persistSettings()
	})
	s.captureCheck.SetChecked(s.captureGameInput)

	s.diagnosticSummary = widget.NewLabel("Errors: 0 | Warnings: 0 | Info: 0")
	s.diagnosticsToggle = widget.NewButton("Collapse", func() {
		s.toggleDiagnosticsPanel()
	})
	diagToolbar := container.NewHBox(
		s.diagnosticSummary,
		s.diagnosticFilter,
		s.diagnosticSearch,
		s.diagnosticsToggle,
	)
	diagSplit := container.NewVSplit(s.diagnosticsList, s.diagnosticDetail)
	diagSplit.Offset = 0.62
	diagPane := container.NewBorder(
		diagToolbar,
		nil, nil, nil,
		diagSplit,
	)
	outputPane := s.buildOutput
	manifestPane := s.manifestOutput
	debugPane := s.debuggerOutput
	s.bottomLeftTabs = container.NewAppTabs(
		container.NewTabItem("Diagnostics", diagPane),
		container.NewTabItem("Output", outputPane),
		container.NewTabItem("Manifest", manifestPane),
		container.NewTabItem("Debugger", debugPane),
	)

	s.editorPane = container.NewBorder(
		s.pathLabel,
		nil, nil, nil,
		s.sourceEntry,
	)
	spriteLabPane := s.buildSpriteLabPane()
	tilemapPlaceholder := widget.NewLabel("Tilemap Editor (coming next)\n\nPlanned: grid placement + CoreLX asset export.")
	soundStudioPlaceholder := widget.NewLabel("Sound Studio (coming next)\n\nPlanned: music/ambience/SFX authoring integrated with packaging.")
	s.workbenchTabs = container.NewAppTabs(
		container.NewTabItem("Code", s.editorPane),
		container.NewTabItem("Sprite Lab", spriteLabPane),
		container.NewTabItem("Tilemap", container.NewScroll(tilemapPlaceholder)),
		container.NewTabItem("Sound", container.NewScroll(soundStudioPlaceholder)),
	)

	s.centerHost = container.NewMax()
	s.contentRoot = container.NewBorder(s.buildToolbar(), s.statusLabel, nil, nil, s.centerHost)
	s.window.SetContent(s.contentRoot)
	s.setViewMode(s.currentView)
	s.refreshDebuggerOutput()
}

func newReadOnlyTextArea() *widget.Entry {
	e := widget.NewMultiLineEntry()
	e.Wrapping = fyne.TextWrapOff
	e.Disable()
	return e
}

func (s *devKitState) buildToolbar() fyne.CanvasObject {
	newProjectBtn := widget.NewButton("New", func() { s.showTemplateDialog() })
	openProjectBtn := widget.NewButton("Open", func() { s.showOpenProjectDialog() })
	saveBtn := widget.NewButton("Save", func() {
		if err := s.save(); err != nil {
			dialog.ShowError(err, s.window)
			s.setStatus("Save failed")
			return
		}
		s.setStatus("Saved")
	})

	buildBtn := widget.NewButton("Build", func() { s.runBuild(false) })
	buildRunBtn := widget.NewButton("Build + Run", func() { s.runBuild(true) })
	buildRunBtn.Importance = widget.HighImportance

	s.runBtn = widget.NewButton("Run", func() { s.runEmulator() })
	s.pauseBtn = widget.NewButton("Pause", func() { s.pauseEmulator() })
	s.stopBtn = widget.NewButton("Stop", func() { s.stopEmulator() })
	s.stopBtn.Importance = widget.DangerImportance

	stepFrameBtn := widget.NewButton("Step F", func() { s.stepFrame() })
	stepCPUBtn := widget.NewButton("Step C", func() { s.stepCPU() })

	s.splitViewBtn = widget.NewButton("Split View", func() { s.setViewMode(viewModeFull) })
	s.emulatorFocusBtn = widget.NewButton("Emulator Focus", func() { s.setViewMode(viewModeEmulatorOnly) })
	s.codeOnlyBtn = widget.NewButton("Code Only", func() { s.setViewMode(viewModeCodeOnly) })
	s.refreshViewToggleButtons()

	loadROMBtn := widget.NewButton("Load ROM", func() { s.openROMDialog() })

	return container.NewHBox(
		newProjectBtn,
		openProjectBtn,
		saveBtn,
		loadROMBtn,
		widget.NewSeparator(),
		buildBtn,
		buildRunBtn,
		widget.NewSeparator(),
		s.runBtn,
		s.pauseBtn,
		s.stopBtn,
		stepFrameBtn,
		stepCPUBtn,
		widget.NewSeparator(),
		s.codeOnlyBtn,
		s.splitViewBtn,
		s.emulatorFocusBtn,
	)
}

func (s *devKitState) setViewMode(mode viewMode) {
	s.captureLayoutState()
	s.currentView = mode
	s.settings.ViewMode = string(mode)
	s.persistSettings()

	switch mode {
	case viewModeEmulatorOnly:
		emuLayout := container.NewBorder(
			container.NewHBox(s.emuLabel, layout.NewSpacer(), s.captureCheck),
			nil, nil, nil,
			s.emuSurface,
		)
		s.centerHost.Objects = []fyne.CanvasObject{emuLayout}
		s.setStatus("View: Emulator Focus")
		if s.captureGameInput {
			s.focusEmulatorInput()
		}
	case viewModeCodeOnly:
		codeLayout := container.NewVSplit(s.workbenchTabs, s.bottomLeftTabs)
		codeLayout.Offset = 0.72
		s.centerHost.Objects = []fyne.CanvasObject{codeLayout}
		s.setStatus("View: Code Only")
	default:
		emuPane := container.NewBorder(
			container.NewHBox(s.emuLabel, layout.NewSpacer(), s.captureCheck),
			nil, nil, nil,
			s.emuSurface,
		)
		s.leftSplit = container.NewVSplit(emuPane, s.bottomLeftTabs)
		s.leftSplit.Offset = clampOffset(s.settings.LeftSplitOffset, defaultLeftSplitOffset)
		if s.diagnosticsCollapsed {
			s.leftSplit.Offset = 1.0
			s.diagnosticsToggle.SetText("Expand")
		} else {
			s.diagnosticsToggle.SetText("Collapse")
		}
		s.mainSplit = container.NewHSplit(s.leftSplit, s.workbenchTabs)
		s.mainSplit.Offset = clampOffset(s.settings.MainSplitOffset, defaultMainSplitOffset)
		s.centerHost.Objects = []fyne.CanvasObject{s.mainSplit}
		s.setStatus("View: Split View")
	}
	s.refreshViewToggleButtons()
	s.centerHost.Refresh()
	s.refreshTitle()
}

func (s *devKitState) applyDiagnosticFilter() {
	needle := ""
	if s.diagnosticSearch != nil {
		needle = strings.ToLower(strings.TrimSpace(s.diagnosticSearch.Text))
	}
	mode := "All"
	if s.diagnosticFilter != nil && s.diagnosticFilter.Selected != "" {
		mode = s.diagnosticFilter.Selected
	}
	s.filteredDiagnostics = s.filteredDiagnostics[:0]
	errCount := 0
	warnCount := 0
	infoCount := 0
	for _, d := range s.diagnostics {
		switch d.Severity {
		case corelx.SeverityError:
			errCount++
		case corelx.SeverityWarning:
			warnCount++
		default:
			infoCount++
		}
		if !diagnosticMatchesFilter(d, mode) {
			continue
		}
		if needle != "" {
			haystack := strings.ToLower(d.Message + " " + d.Code + " " + d.File + " " + string(d.Stage) + " " + string(d.Category))
			if !strings.Contains(haystack, needle) {
				continue
			}
		}
		s.filteredDiagnostics = append(s.filteredDiagnostics, d)
	}
	if s.diagnosticsList != nil {
		s.diagnosticsList.UnselectAll()
		s.diagnosticsList.Refresh()
	}
	if s.diagnosticSummary != nil {
		s.diagnosticSummary.SetText(fmt.Sprintf("Errors: %d | Warnings: %d | Info: %d", errCount, warnCount, infoCount))
	}
	s.updateDiagnosticDetailSelection()
}

func (s *devKitState) openDialog() {
	fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, s.window)
			return
		}
		if rc == nil {
			return
		}
		defer rc.Close()
		data, readErr := io.ReadAll(rc)
		if readErr != nil {
			dialog.ShowError(readErr, s.window)
			return
		}
		s.setSourceContent(string(data), false, true)
		s.currentPath = uriPath(rc.URI())
		s.refreshTitle()
		s.pathLabel.SetText(displayPath(s.currentPath))
		s.rememberSourcePath(s.currentPath)
		s.setStatus("Opened project: " + baseNameOr(s.currentPath, "buffer"))
		s.appendBuildOutput("Opened " + s.currentPath)
	}, s.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".corelx", ".clx", ".txt"}))
	if loc := dialogListableForDir(s.settings.LastSourceDir); loc != nil {
		fd.SetLocation(loc)
	}
	fd.Show()
}

func (s *devKitState) showOpenProjectDialog() {
	type filterOpt struct {
		Label string
		Kind  string
	}
	options := []filterOpt{
		{Label: "CoreLX Project Source (*.corelx, *.clx, *.txt)", Kind: "source"},
		{Label: "Build Output (*.rom)", Kind: "rom"},
		{Label: "All Supported", Kind: "all"},
	}
	filterSel := widget.NewSelect([]string{options[0].Label, options[1].Label, options[2].Label}, nil)
	filterSel.SetSelected(options[0].Label)

	recentLabel := widget.NewLabel("Recent projects")
	recentLabel.TextStyle = fyne.TextStyle{Bold: true}
	preview := newReadOnlyTextArea()
	preview.SetPlaceHolder("Select a recent project to preview.")
	selectedRecent := ""

	recentOptions := append([]string{}, s.settings.RecentFiles...)
	recentSelect := widget.NewSelect(recentOptions, func(path string) {
		selectedRecent = path
		if path == "" {
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			preview.Enable()
			preview.SetText("Preview unavailable: " + err.Error())
			preview.Disable()
			return
		}
		text := string(data)
		if len(text) > 2000 {
			text = text[:2000] + "\n... (preview truncated)"
		}
		preview.Enable()
		preview.SetText(text)
		preview.Disable()
	})
	if len(recentOptions) == 0 {
		recentSelect.PlaceHolder = "No recent projects"
	}

	var d dialog.Dialog
	openRecentBtn := widget.NewButton("Open Selected Recent", func() {
		if selectedRecent == "" {
			s.setStatus("Select a recent project first")
			return
		}
		if strings.HasSuffix(strings.ToLower(selectedRecent), ".rom") {
			data, err := os.ReadFile(selectedRecent)
			if err != nil {
				dialog.ShowError(err, s.window)
				return
			}
			if err := s.loadROMIntoEmbedded(data); err != nil {
				dialog.ShowError(err, s.window)
				return
			}
			s.lastROMPath = selectedRecent
			s.rememberROMPath(selectedRecent)
			s.setStatus("Loaded project build artifact")
			d.Hide()
			return
		}
		if err := s.loadFile(selectedRecent, true); err != nil {
			dialog.ShowError(err, s.window)
			return
		}
		s.setStatus("Opened project")
		d.Hide()
	})
	browseBtn := widget.NewButton("Browse...", func() {
		switch filterSel.Selected {
		case options[1].Label:
			s.openROMDialog()
		case options[2].Label:
			s.openAnyProjectDialog()
		default:
			s.openDialog()
		}
		d.Hide()
	})
	openRecentBtn.Importance = widget.HighImportance
	buttons := container.NewHBox(openRecentBtn, browseBtn)

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Open Project"),
			filterSel,
		),
		buttons,
		nil, nil,
		container.NewVSplit(
			container.NewVBox(recentLabel, recentSelect),
			container.NewScroll(preview),
		),
	)

	d = dialog.NewCustom("Open Project", "Close", content, s.window)
	d.Resize(fyne.NewSize(860, 620))
	d.Show()
}

func (s *devKitState) openAnyProjectDialog() {
	fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, s.window)
			return
		}
		if rc == nil {
			return
		}
		defer rc.Close()
		path := uriPath(rc.URI())
		data, readErr := io.ReadAll(rc)
		if readErr != nil {
			dialog.ShowError(readErr, s.window)
			return
		}
		if strings.HasSuffix(strings.ToLower(path), ".rom") {
			if loadErr := s.loadROMIntoEmbedded(data); loadErr != nil {
				dialog.ShowError(loadErr, s.window)
				return
			}
			s.lastROMPath = path
			s.rememberROMPath(path)
			s.setStatus("Loaded project build artifact")
			return
		}
		s.setSourceContent(string(data), false, true)
		s.currentPath = path
		s.refreshTitle()
		s.pathLabel.SetText(displayPath(s.currentPath))
		s.rememberSourcePath(s.currentPath)
		s.setStatus("Opened project")
		s.appendBuildOutput("Opened " + s.currentPath)
	}, s.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".corelx", ".clx", ".txt", ".rom"}))
	fd.Show()
}

func (s *devKitState) openROMDialog() {
	fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, s.window)
			return
		}
		if rc == nil {
			return
		}
		defer rc.Close()
		data, readErr := io.ReadAll(rc)
		if readErr != nil {
			dialog.ShowError(readErr, s.window)
			return
		}
		if loadErr := s.loadROMIntoEmbedded(data); loadErr != nil {
			dialog.ShowError(loadErr, s.window)
			s.appendBuildOutput("Load build artifact failed: " + loadErr.Error())
			s.setStatus("Load project build failed")
			return
		}
		s.lastROMPath = uriPath(rc.URI())
		s.rememberROMPath(s.lastROMPath)
		s.setViewMode(viewModeFull)
		s.appendBuildOutput("Loaded build artifact into emulator subsystem: " + s.lastROMPath)
		s.setStatus("Project build loaded")
	}, s.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".rom"}))
	if loc := dialogListableForDir(s.settings.LastROMDir); loc != nil {
		fd.SetLocation(loc)
	}
	fd.Show()
}

func (s *devKitState) saveAsDialog() {
	fd := dialog.NewFileSave(func(wc fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, s.window)
			return
		}
		if wc == nil {
			return
		}
		defer wc.Close()
		if _, writeErr := wc.Write([]byte(s.sourceEntry.Text)); writeErr != nil {
			dialog.ShowError(writeErr, s.window)
			return
		}
		s.currentPath = uriPath(wc.URI())
		s.dirty = false
		s.refreshTitle()
		s.pathLabel.SetText(displayPath(s.currentPath))
		s.rememberSourcePath(s.currentPath)
		s.clearAutosaveJournal()
		s.setStatus("Saved")
		s.appendBuildOutput("Saved " + s.currentPath)
	}, s.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".corelx", ".clx", ".txt"}))
	if loc := dialogListableForDir(s.settings.LastSourceDir); loc != nil {
		fd.SetLocation(loc)
	}
	if s.currentPath != "" {
		fd.SetFileName(baseNameOr(s.currentPath, "main.corelx"))
	} else {
		fd.SetFileName("main.corelx")
	}
	fd.Show()
}

func (s *devKitState) save() error {
	if s.currentPath == "" {
		s.saveAsDialog()
		return nil
	}
	if err := os.WriteFile(s.currentPath, []byte(s.sourceEntry.Text), 0644); err != nil {
		return err
	}
	s.dirty = false
	s.refreshTitle()
	s.pathLabel.SetText(displayPath(s.currentPath))
	s.rememberSourcePath(s.currentPath)
	s.clearAutosaveJournal()
	s.appendBuildOutput("Saved " + s.currentPath)
	return nil
}

func (s *devKitState) loadFile(path string, clearAutosave bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s.setSourceContent(string(data), false, clearAutosave)
	s.currentPath = path
	s.refreshTitle()
	s.pathLabel.SetText(displayPath(s.currentPath))
	s.rememberSourcePath(s.currentPath)
	s.appendBuildOutput("Opened " + s.currentPath)
	return nil
}

func (s *devKitState) setSourceContent(text string, dirty bool, clearAutosave bool) {
	s.suppressSourceChange = true
	s.sourceEntry.SetText(text)
	s.suppressSourceChange = false
	s.dirty = dirty
	if dirty {
		s.writeAutosaveSnapshot(text)
	} else if clearAutosave {
		s.clearAutosaveJournal()
	}
	s.refreshTitle()
}

func (s *devKitState) refreshTitle() {
	name := "Untitled.corelx"
	if s.currentPath != "" {
		name = baseNameOr(s.currentPath, name)
	}
	if s.dirty {
		name += " *"
	}
	var viewLabel string
	switch s.currentView {
	case viewModeEmulatorOnly:
		viewLabel = "Emulator Focus"
	case viewModeCodeOnly:
		viewLabel = "Code Only"
	default:
		viewLabel = "Split View"
	}
	s.window.SetTitle("Nitro-Core-DX - " + name + " [" + viewLabel + "]")
}

func (s *devKitState) runBuild(runAfter bool) {
	sourcePath := s.currentPath
	if sourcePath == "" {
		sourcePath = "untitled.corelx"
	}
	artifactBase := strings.TrimSuffix(baseNameOr(sourcePath, "untitled.corelx"), filepathExtOrEmpty(sourcePath))
	if artifactBase == "" {
		artifactBase = "untitled"
	}
	romOut := pathJoin(s.tempDir, artifactBase+".rom")

	start := time.Now()
	s.setStatus("Building...")
	s.appendBuildOutput(fmt.Sprintf("Build started (%s)", sourcePath))

	buildResult, err := s.backend.BuildSource(s.sourceEntry.Text, sourcePath)
	elapsed := time.Since(start)
	var bundle corelx.CompileBundle
	var res *corelx.CompileResult
	if buildResult != nil {
		bundle = buildResult.Bundle
		res = buildResult.Result
		s.lastROMPath = buildResult.Artifacts.ROMPath
		elapsed = buildResult.Elapsed
	} else {
		bundle = corelx.CompileBundle{}
		res = nil
		s.lastROMPath = romOut
	}
	s.diagnostics = bundle.Diagnostics
	s.applyDiagnosticFilter()

	s.updateManifestPane(bundle, res)
	s.appendBuildSummary(bundle, res, err, elapsed)

	if err != nil {
		s.setStatus(fmt.Sprintf("Build failed (%d errors)", bundle.Summary.ErrorCount))
		return
	}

	s.setStatus("Build succeeded")
	if runAfter {
		if res != nil && len(res.ROMBytes) > 0 {
			if loadErr := s.loadROMIntoEmbedded(res.ROMBytes); loadErr != nil {
				dialog.ShowError(loadErr, s.window)
				s.appendBuildOutput("Run failed: " + loadErr.Error())
				s.setStatus("Build succeeded; run failed")
				return
			}
			s.setViewMode(viewModeFull)
			s.appendBuildOutput("Project build loaded into emulator subsystem")
			s.setStatus("Build + Run completed")
		} else {
			s.setStatus("Build succeeded (no executable artifact emitted)")
		}
	}
}

func (s *devKitState) updateManifestPane(bundle corelx.CompileBundle, res *corelx.CompileResult) {
	var text string
	if res != nil && len(res.ManifestJSON) > 0 {
		text = string(res.ManifestJSON)
	} else if bundle.Manifest != nil {
		if b, err := json.MarshalIndent(bundle.Manifest, "", "  "); err == nil {
			text = string(b)
		}
	}
	s.manifestOutput.Enable()
	s.manifestOutput.SetText(text)
	s.manifestOutput.Disable()
}

func (s *devKitState) appendBuildSummary(bundle corelx.CompileBundle, res *corelx.CompileResult, buildErr error, elapsed time.Duration) {
	var sb strings.Builder
	if buildErr != nil {
		sb.WriteString("Build failed\n")
	} else {
		sb.WriteString("Build succeeded\n")
	}
	sb.WriteString(fmt.Sprintf("Time: %s\n", elapsed.Round(time.Millisecond)))
	sb.WriteString(fmt.Sprintf("Errors: %d  Warnings: %d  Info: %d\n",
		bundle.Summary.ErrorCount, bundle.Summary.WarningCount, bundle.Summary.InfoCount))
	if res != nil && res.Manifest != nil {
		sb.WriteString(fmt.Sprintf("Build output bytes (emitted/planned): %d / %d\n",
			res.Manifest.EmittedROMSizeBytes, res.Manifest.PlannedROMSizeBytes))
	}
	sb.WriteString(fmt.Sprintf("Artifacts: %s\n", s.tempDir))
	if buildErr != nil && len(bundle.Diagnostics) > 0 {
		sb.WriteString("\nFirst error:\n")
		sb.WriteString(formatDiagnostic(bundle.Diagnostics[0]))
		sb.WriteString("\n")
	}
	s.appendBuildOutput(sb.String())
}

func (s *devKitState) loadROMIntoEmbedded(romBytes []byte) error {
	if err := s.backend.LoadROMBytes(romBytes); err != nil {
		return err
	}
	if s.audioDev != 0 {
		sdl.ClearQueuedAudio(s.audioDev)
	}

	fyne.Do(func() {
		s.emuLabel.SetText("Hardware: running")
		if s.captureGameInput {
			s.focusEmulatorInput()
		}
	})
	return nil
}

func (s *devKitState) shutdownEmbeddedEmulator() {
	s.backend.Shutdown()
}

func (s *devKitState) startEmulatorLoop() {
	go func() {
		const uiTickHz = 120
		ticker := time.NewTicker(time.Second / uiTickHz)
		defer ticker.Stop()

		lastTick := time.Now()
		for {
			select {
			case <-s.updateLoopStop:
				return
			case <-ticker.C:
			}

			now := time.Now()
			delta := now.Sub(lastTick)
			lastTick = now
			if delta > 250*time.Millisecond {
				delta = 250 * time.Millisecond
			}

			s.routeInputToEmulator()
			tick, err := s.backend.Tick(delta)
			if err != nil {
				fyne.Do(func() {
					s.appendBuildOutput("Hardware frame error: " + err.Error())
					s.setStatus("Hardware error")
				})
				continue
			}
			if !tick.Snapshot.Loaded {
				continue
			}
			for _, samples := range tick.AudioFrames {
				s.queueFrameAudio(samples)
			}
			if tick.PresentFrame {
				img, err := s.renderEmbeddedFrame(tick.Framebuffer)
				if err == nil {
					fps := tick.Snapshot.FPS
					cycles := tick.Snapshot.CPUCyclesPerFrame
					frameCount := tick.Snapshot.FrameCount
					paused := tick.Snapshot.Paused
					fyne.Do(func() {
						s.emuImage.Image = img
						s.emuImage.Refresh()
						state := "running"
						if paused {
							state = "paused"
						}
						s.emuLabel.SetText(fmt.Sprintf("Hardware: %s | FPS %.1f | CPU %d cycles/frame | Frame %d", state, fps, cycles, frameCount))
						s.refreshDebuggerOutput()
					})
				}
			}
		}
	}()
}

func (s *devKitState) stopEmulatorLoop() {
	s.updateLoopOnce.Do(func() {
		close(s.updateLoopStop)
	})
}

func (s *devKitState) initAudio() error {
	if err := sdl.InitSubSystem(sdl.INIT_AUDIO); err != nil {
		return fmt.Errorf("SDL audio init failed: %w", err)
	}

	spec := sdl.AudioSpec{
		Freq:     44100,
		Format:   sdl.AUDIO_F32,
		Channels: 2,
		Samples:  735,
	}
	dev, err := sdl.OpenAudioDevice("", false, &spec, nil, 0)
	if err != nil {
		sdl.QuitSubSystem(sdl.INIT_AUDIO)
		return fmt.Errorf("SDL open audio failed: %w", err)
	}
	s.audioDev = dev
	sdl.PauseAudioDevice(s.audioDev, false)
	return nil
}

func (s *devKitState) shutdownAudio() {
	if s.audioDev != 0 {
		sdl.CloseAudioDevice(s.audioDev)
		s.audioDev = 0
	}
	sdl.QuitSubSystem(sdl.INIT_AUDIO)
}

func (s *devKitState) queueFrameAudio(samples []int16) {
	if s.audioDev == 0 {
		return
	}
	// Keep queue bounded to reduce audio latency growth during UI stalls.
	if sdl.GetQueuedAudioSize(s.audioDev) > uint32(len(s.audioFrame))*4 {
		return
	}
	if len(samples) == 0 {
		return
	}
	j := 0
	for _, sample := range samples {
		f := apu.ConvertFixedToFloat(sample)
		bits := math.Float32bits(f)
		binary.LittleEndian.PutUint32(s.audioFrame[j:j+4], bits)
		binary.LittleEndian.PutUint32(s.audioFrame[j+4:j+8], bits)
		j += 8
	}
	_ = sdl.QueueAudio(s.audioDev, s.audioFrame)
}

func (s *devKitState) renderEmbeddedFrame(buf []uint32) (image.Image, error) {
	if len(buf) != devKitScreenW*devKitScreenH {
		return nil, fmt.Errorf("buffer size mismatch: expected %d got %d", devKitScreenW*devKitScreenH, len(buf))
	}
	img := s.frameImages[s.frameIdx]
	s.frameIdx ^= 1
	pix := img.Pix
	stride := img.Stride
	scale := s.emuScale
	for y := 0; y < devKitScreenH; y++ {
		for x := 0; x < devKitScreenW; x++ {
			c := buf[y*devKitScreenW+x]
			r := uint8((c >> 16) & 0xFF)
			g := uint8((c >> 8) & 0xFF)
			b := uint8(c & 0xFF)
			bx := x * scale
			by := y * scale
			for sy := 0; sy < scale; sy++ {
				row := (by + sy) * stride
				for sx := 0; sx < scale; sx++ {
					off := row + (bx+sx)*4
					pix[off+0] = r
					pix[off+1] = g
					pix[off+2] = b
					pix[off+3] = 0xFF
				}
			}
		}
	}
	return img, nil
}

func (s *devKitState) setupKeyboardInput() {
	s.window.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		s.handleTypedKey(key)
	})

	if c, ok := s.window.Canvas().(desktop.Canvas); ok {
		s.keyMu.Lock()
		s.desktopKeyEvents = true
		s.keyMu.Unlock()
		c.SetOnKeyDown(func(key *fyne.KeyEvent) {
			s.handleKeyDown(key)
		})
		c.SetOnKeyUp(func(key *fyne.KeyEvent) {
			s.handleKeyUp(key)
		})
	}
}

func (s *devKitState) handleTypedKey(key *fyne.KeyEvent) {
	if key == nil {
		return
	}
	if s.shouldHandleTypedWindowKey(key.Name) {
		if s.handleWindowHotkey(key.Name) {
			return
		}
	}
	s.keyMu.Lock()
	if !s.desktopKeyEvents {
		s.typedKeyUntil[key.Name] = time.Now().Add(450 * time.Millisecond)
	}
	s.keyMu.Unlock()
	s.routeInputToEmulator()
}

func (s *devKitState) handleKeyDown(key *fyne.KeyEvent) {
	if key == nil {
		return
	}
	if s.handleWindowHotkey(key.Name) {
		return
	}
	s.keyMu.Lock()
	s.keyStates[key.Name] = true
	s.keyMu.Unlock()
	s.routeInputToEmulator()
}

func (s *devKitState) handleKeyUp(key *fyne.KeyEvent) {
	if key == nil {
		return
	}
	s.keyMu.Lock()
	s.keyStates[key.Name] = false
	delete(s.typedKeyUntil, key.Name)
	s.keyMu.Unlock()
	s.routeInputToEmulator()
}

func (s *devKitState) focusEmulatorInput() {
	if s.window == nil || s.window.Canvas() == nil || s.emuKeys == nil {
		return
	}
	s.window.Canvas().Focus(s.emuKeys)
}

func (s *devKitState) shouldHandleTypedWindowKey(name fyne.KeyName) bool {
	switch name {
	case fyne.KeyF11:
		s.keyMu.Lock()
		desktop := s.desktopKeyEvents
		s.keyMu.Unlock()
		return !desktop
	default:
		return false
	}
}

func (s *devKitState) handleWindowHotkey(name fyne.KeyName) bool {
	switch name {
	case fyne.KeyF11:
		s.maximizeWindow()
		return true
	}
	return false
}

// maximizeWindow resizes the window to fill the screen.
func (s *devKitState) maximizeWindow() {
	if s.window == nil {
		return
	}
	if s.windowMaximized {
		s.restoreWindow()
		return
	}
	s.savedRestoreSize = s.window.Canvas().Size()
	s.windowMaximized = true
	s.window.Resize(fyne.NewSize(9999, 9999))
	s.setStatus("Window: Maximized (F11 or View > Restore Window to restore)")
}

// restoreWindow restores the window size after a programmatic maximize.
func (s *devKitState) restoreWindow() {
	if s.window == nil || !s.windowMaximized {
		return
	}
	s.windowMaximized = false
	restore := s.savedRestoreSize
	if restore.Width <= 0 || restore.Height <= 0 {
		restore = fyne.NewSize(defaultWindowWidth, defaultWindowHeight)
	}
	s.window.Resize(restore)
	s.setStatus("Window: Restored")
	s.reapplyWindowHints()
}

// reapplyWindowHints ensures native WM maximize stays enabled after state changes.
func (s *devKitState) reapplyWindowHints() {
	if s.window == nil {
		return
	}
	go func() {
		time.Sleep(200 * time.Millisecond)
		fyne.Do(func() {
			if s.window != nil {
				s.window.SetFixedSize(false)
				_ = applyX11MaximizeHint(s.window)
			}
		})
	}()
}

func (s *devKitState) shouldCaptureGameInput() bool {
	if !s.captureGameInput {
		return false
	}
	// "Capture Game Input" is an explicit override. When enabled, route keys to the
	// embedded emulator regardless of which editor widget currently has focus.
	// This matches user expectation for Build+Run testing inside the Dev Kit.
	return true
}

func (s *devKitState) routeInputToEmulator() {
	if !s.shouldCaptureGameInput() {
		s.applyInputButtons(0)
		return
	}
	s.applyInputButtons(s.computeButtonMask())
}

func (s *devKitState) applyInputButtons(buttons uint16) {
	s.backend.SetInputButtons(buttons)
}

func (s *devKitState) computeButtonMask() uint16 {
	now := time.Now()
	s.keyMu.Lock()
	defer s.keyMu.Unlock()
	isPressed := func(key fyne.KeyName) bool {
		if s.keyStates[key] {
			return true
		}
		if until, ok := s.typedKeyUntil[key]; ok {
			if now.Before(until) {
				return true
			}
			delete(s.typedKeyUntil, key)
		}
		return false
	}

	var buttons uint16
	if isPressed(fyne.KeyW) || isPressed(fyne.KeyUp) {
		buttons |= 0x01
	}
	if isPressed(fyne.KeyS) || isPressed(fyne.KeyDown) {
		buttons |= 0x02
	}
	if isPressed(fyne.KeyA) || isPressed(fyne.KeyLeft) {
		buttons |= 0x04
	}
	if isPressed(fyne.KeyD) || isPressed(fyne.KeyRight) {
		buttons |= 0x08
	}
	if isPressed(fyne.KeyZ) {
		buttons |= 0x10
	}
	if isPressed(fyne.KeyX) {
		buttons |= 0x20
	}
	if isPressed(fyne.KeyV) {
		buttons |= 0x40
	}
	if isPressed(fyne.KeyC) {
		buttons |= 0x80
	}
	if isPressed(fyne.KeyQ) {
		buttons |= 0x100
	}
	if isPressed(fyne.KeyE) {
		buttons |= 0x200
	}
	if isPressed(fyne.KeyReturn) {
		buttons |= 0x400
	}
	if isPressed(fyne.KeyBackspace) {
		buttons |= 0x800
	}
	return buttons
}

func (s *devKitState) jumpToDiagnostic(d corelx.Diagnostic) {
	if d.Line <= 0 {
		return
	}
	row := d.Line - 1
	col := 0
	if d.Column > 0 {
		col = d.Column - 1
	}
	s.window.Canvas().Focus(s.sourceEntry)
	s.sourceEntry.CursorRow = maxInt(0, row)
	s.sourceEntry.CursorColumn = maxInt(0, col)
	s.sourceEntry.Refresh()
}

func (s *devKitState) showDiagnosticDetail(d corelx.Diagnostic) {
	var sb strings.Builder
	sb.WriteString(formatDiagnostic(d))
	if len(d.Notes) > 0 {
		sb.WriteString("\n\nNotes:\n")
		for _, n := range d.Notes {
			sb.WriteString("- ")
			sb.WriteString(n)
			sb.WriteString("\n")
		}
	}
	if len(d.Related) > 0 {
		sb.WriteString("\nRelated:\n")
		for _, r := range d.Related {
			sb.WriteString(fmt.Sprintf("- %s:%d:%d %s\n", r.File, r.Line, r.Column, r.Message))
		}
	}
	s.diagnosticDetail.Enable()
	s.diagnosticDetail.SetText(strings.TrimSpace(sb.String()))
	s.diagnosticDetail.Disable()
}

func (s *devKitState) updateDiagnosticDetailSelection() {
	if len(s.filteredDiagnostics) == 0 {
		s.diagnosticDetail.Enable()
		if len(s.diagnostics) == 0 {
			s.diagnosticDetail.SetText("No build issues")
		} else {
			s.diagnosticDetail.SetText("No diagnostics match current filter")
		}
		s.diagnosticDetail.Disable()
		return
	}
	s.showDiagnosticDetail(s.filteredDiagnostics[0])
}

func (s *devKitState) appendBuildOutput(msg string) {
	ts := time.Now().Format("15:04:05")
	prev := s.buildOutput.Text
	if prev != "" && !strings.HasSuffix(prev, "\n") {
		prev += "\n"
	}
	line := fmt.Sprintf("[%s] %s", ts, msg)
	s.buildOutput.Enable()
	s.buildOutput.SetText(prev + line + "\n")
	s.buildOutput.Disable()
}

func (s *devKitState) setStatus(msg string) {
	s.statusLabel.SetText(msg)
}

func (s *devKitState) refreshViewToggleButtons() {
	if s.splitViewBtn == nil || s.emulatorFocusBtn == nil || s.codeOnlyBtn == nil {
		return
	}
	s.splitViewBtn.Importance = widget.MediumImportance
	s.emulatorFocusBtn.Importance = widget.MediumImportance
	s.codeOnlyBtn.Importance = widget.MediumImportance
	switch s.currentView {
	case viewModeEmulatorOnly:
		s.emulatorFocusBtn.Importance = widget.HighImportance
	case viewModeCodeOnly:
		s.codeOnlyBtn.Importance = widget.HighImportance
	default:
		s.splitViewBtn.Importance = widget.HighImportance
	}
	s.splitViewBtn.Refresh()
	s.emulatorFocusBtn.Refresh()
	s.codeOnlyBtn.Refresh()
}

func (s *devKitState) runEmulator() {
	snap := s.backend.Snapshot()
	if !snap.Loaded {
		s.runBuild(true)
		return
	}
	if snap.Paused {
		if _, err := s.backend.TogglePause(); err != nil {
			s.appendBuildOutput("Run failed: " + err.Error())
			s.setStatus("Run failed")
			return
		}
	}
	s.setStatus("Running")
}

func (s *devKitState) pauseEmulator() {
	snap := s.backend.Snapshot()
	if !snap.Loaded {
		s.setStatus("No active project build")
		return
	}
	if snap.Paused {
		s.setStatus("Already paused")
		return
	}
	if _, err := s.backend.TogglePause(); err != nil {
		s.appendBuildOutput("Pause failed: " + err.Error())
		s.setStatus("Pause failed")
		return
	}
	s.setStatus("Paused")
}

func (s *devKitState) stopEmulator() {
	snap := s.backend.Snapshot()
	if !snap.Loaded {
		s.setStatus("No active project build")
		return
	}
	if !snap.Paused {
		if _, err := s.backend.TogglePause(); err != nil {
			s.appendBuildOutput("Stop failed: " + err.Error())
			s.setStatus("Stop failed")
			return
		}
	}
	s.applyInputButtons(0)
	s.setStatus("Stopped")
}

func (s *devKitState) hardwareReset() {
	if err := s.backend.ResetEmulator(); err != nil {
		s.setStatus("No active project build")
		return
	}
	s.setStatus("Hardware reset complete")
}

func (s *devKitState) stepFrame() {
	snap := s.backend.Snapshot()
	if !snap.Loaded {
		s.setStatus("No active project build")
		return
	}
	if !snap.Paused {
		s.setStatus("Pause before stepping frames")
		return
	}
	frames := s.parseStepCount(s.stepFrameEntry.Text, 1)
	if err := s.backend.StepFrame(frames); err != nil {
		s.setStatus("Step frame failed")
		s.appendBuildOutput("Step frame failed: " + err.Error())
		return
	}
	s.refreshDebuggerOutput()
	s.setStatus(fmt.Sprintf("Stepped %d frame(s)", frames))
}

func (s *devKitState) stepCPU() {
	snap := s.backend.Snapshot()
	if !snap.Loaded {
		s.setStatus("No active project build")
		return
	}
	if !snap.Paused {
		s.setStatus("Pause before stepping CPU")
		return
	}
	steps := s.parseStepCount(s.stepCPUEntry.Text, 1)
	if err := s.backend.StepCPU(steps); err != nil {
		s.setStatus("Step CPU failed")
		s.appendBuildOutput("Step CPU failed: " + err.Error())
		return
	}
	s.refreshDebuggerOutput()
	s.setStatus(fmt.Sprintf("Stepped %d CPU instruction(s)", steps))
}

func (s *devKitState) toggleDiagnosticsPanel() {
	if s.leftSplit == nil || s.diagnosticsToggle == nil {
		return
	}
	s.diagnosticsCollapsed = !s.diagnosticsCollapsed
	if s.diagnosticsCollapsed {
		s.leftSplit.Offset = 1.0
		s.diagnosticsToggle.SetText("Expand")
		s.settings.DiagnosticsPanel = false
	} else {
		s.leftSplit.Offset = clampOffset(s.settings.LeftSplitOffset, defaultLeftSplitOffset)
		s.diagnosticsToggle.SetText("Collapse")
		s.settings.DiagnosticsPanel = true
	}
	s.persistSettings()
	s.leftSplit.Refresh()
}

func (s *devKitState) captureLayoutState() {
	if s.currentView == viewModeFull && s.mainSplit != nil {
		s.settings.MainSplitOffset = clampOffset(s.mainSplit.Offset, defaultMainSplitOffset)
		if s.leftSplit != nil && !s.diagnosticsCollapsed {
			s.settings.LeftSplitOffset = clampOffset(s.leftSplit.Offset, defaultLeftSplitOffset)
		}
	}
	s.settings.DiagnosticsPanel = !s.diagnosticsCollapsed
}

func (s *devKitState) setUIDensity(density string) {
	s.settings.UIDensity = density
	s.persistSettings()
	if density == "standard" {
		fyne.CurrentApp().Settings().SetTheme(newStandardTheme())
	} else {
		fyne.CurrentApp().Settings().SetTheme(newCompactTheme())
	}
	s.setStatus("UI density: " + density + " (applied)")
}

func (s *devKitState) applyLayoutPreset(preset string) {
	switch preset {
	case layoutPresetCodeFocus:
		s.setViewMode(viewModeCodeOnly)
		s.diagnosticsCollapsed = false
	case layoutPresetArtMode:
		s.settings.MainSplitOffset = 0.55
		s.settings.LeftSplitOffset = 0.72
		s.setViewMode(viewModeFull)
		s.diagnosticsCollapsed = false
	case layoutPresetDebugMode:
		s.settings.MainSplitOffset = 0.50
		s.settings.LeftSplitOffset = 0.42
		s.setViewMode(viewModeFull)
		s.diagnosticsCollapsed = false
	case layoutPresetEmulatorFocus:
		s.setViewMode(viewModeEmulatorOnly)
		s.diagnosticsCollapsed = true
	default:
		preset = layoutPresetBalanced
		s.settings.MainSplitOffset = defaultMainSplitOffset
		s.settings.LeftSplitOffset = defaultLeftSplitOffset
		s.setViewMode(viewModeFull)
		s.diagnosticsCollapsed = false
	}
	if s.diagnosticsToggle != nil {
		if s.diagnosticsCollapsed {
			if s.leftSplit != nil {
				s.leftSplit.Offset = 1.0
			}
			s.diagnosticsToggle.SetText("Expand")
		} else {
			s.diagnosticsToggle.SetText("Collapse")
		}
	}
	s.settings.LayoutPreset = preset
	s.captureLayoutState()
	s.persistSettings()
}

func clampOffset(v, fallback float64) float64 {
	if v <= 0 || v >= 1 {
		return fallback
	}
	return v
}

func (s *devKitState) parseStepCount(text string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || n <= 0 {
		return fallback
	}
	if n > 10000 {
		return 10000
	}
	return n
}

func (s *devKitState) refreshDebuggerOutput() {
	if s.debuggerOutput == nil {
		return
	}

	snap := s.backend.Snapshot()
	pc := s.backend.GetPCState()
	regs := s.backend.GetRegisters()

	var sb strings.Builder
	if !snap.Loaded || !pc.Loaded || !regs.Loaded {
		sb.WriteString("No build loaded\n")
		sb.WriteString("Use Build + Run to load a project and inspect CPU state.")
	} else {
		sb.WriteString(fmt.Sprintf("Running: %v\nPaused: %v\n", snap.Running, snap.Paused))
		sb.WriteString(fmt.Sprintf("PC: %02X:%04X  PBR:%02X DBR:%02X SP:%04X\n", pc.PCBank, pc.PCOffset, pc.PBR, pc.DBR, pc.SP))
		sb.WriteString(fmt.Sprintf("Flags: 0x%02X  Cycles: %d  Frame: %d\n", pc.Flags, pc.Cycles, snap.FrameCount))
		sb.WriteString("\nRegisters:\n")
		sb.WriteString(fmt.Sprintf("R0:%04X  R1:%04X  R2:%04X  R3:%04X\n", regs.R0, regs.R1, regs.R2, regs.R3))
		sb.WriteString(fmt.Sprintf("R4:%04X  R5:%04X  R6:%04X  R7:%04X\n", regs.R4, regs.R5, regs.R6, regs.R7))
	}

	s.debuggerOutput.Enable()
	s.debuggerOutput.SetText(sb.String())
	s.debuggerOutput.Disable()
}

func formatDiagnostic(d corelx.Diagnostic) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s %s (%s/%s)", strings.ToUpper(string(d.Severity)), d.Code, d.Stage, d.Category))
	if d.File != "" {
		sb.WriteString("\n")
		sb.WriteString(d.File)
		if d.Line > 0 {
			sb.WriteString(fmt.Sprintf(":%d:%d", d.Line, maxInt(1, d.Column)))
		}
		if d.EndLine > 0 && d.EndColumn > 0 {
			sb.WriteString(fmt.Sprintf("-%d:%d", d.EndLine, d.EndColumn))
		}
	}
	sb.WriteString("\n")
	sb.WriteString(d.Message)
	return sb.String()
}

func displayPath(path string) string {
	if path == "" {
		return "Untitled.corelx"
	}
	return path
}

func uriPath(u fyne.URI) string {
	if u == nil {
		return ""
	}
	return u.Path()
}

func baseNameOr(path, fallback string) string {
	if path == "" {
		return fallback
	}
	b := filepath.Base(path)
	if b == "." || b == string(os.PathSeparator) || b == "" {
		return fallback
	}
	return b
}

func filepathExtOrEmpty(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Ext(path)
}

func pathJoin(elem ...string) string {
	return filepath.Join(elem...)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func diagnosticMatchesFilter(d corelx.Diagnostic, mode string) bool {
	switch mode {
	case "Errors":
		return d.Severity == corelx.SeverityError
	case "Warnings":
		return d.Severity == corelx.SeverityWarning
	case "Info":
		return d.Severity == corelx.SeverityInfo
	default:
		return true
	}
}
