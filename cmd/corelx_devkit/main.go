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
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
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
	launchDir    string
	settingsPath string
	settings     devKitSettings

	currentPath  string
	lastROMPath  string
	autosavePath string
	dirty        bool

	diagnostics         []corelx.Diagnostic
	filteredDiagnostics []corelx.Diagnostic

	window          fyne.Window
	centerHost      *fyne.Container
	contentRoot     *fyne.Container
	currentView     viewMode
	statusLabel     *widget.Label
	buildStateLabel *widget.Label
	pathLabel       *widget.Label
	sourceEditor    *coreLXCodeEditor
	findBar         fyne.CanvasObject
	findEntry       *widget.Entry
	replaceEntry    *widget.Entry
	gotoEntry       *widget.Entry
	findMatchCase   *widget.Check
	buildOutput     *widget.Entry
	manifestOutput  *widget.Entry
	debuggerOutput  *widget.Entry

	diagnosticFilter  *widget.Select
	diagnosticSearch  *widget.Entry
	diagnosticsList   *widget.List
	diagnosticDetail  *widget.Entry
	diagnosticSummary *widget.Label
	diagnosticsToggle *widget.Button
	stepFrameEntry    *widget.Entry
	stepCPUEntry      *widget.Entry

	emuSurface       fyne.CanvasObject
	captureCheck     *widget.Check
	bottomLeftTabs   *container.AppTabs
	leftSplit        *container.Split
	mainSplit        *container.Split
	codeSplit        *container.Split
	editorPane       fyne.CanvasObject
	workbenchTabs    *container.AppTabs
	splitViewBtn     *widget.Button
	emulatorFocusBtn *widget.Button
	codeOnlyBtn      *widget.Button
	focusPanelBtn    *widget.Button
	runBtn           *widget.Button
	pauseBtn         *widget.Button
	stopBtn          *widget.Button

	emuScale       int
	emuImage       *canvas.Image
	emuKeys        *emulatorKeyOverlay
	emuLabel       *widget.Label
	emuWell        *canvas.Rectangle
	hardwareLamp   *statusLamp
	brandLamp      *statusLamp
	wordmark       *canvas.Text
	idleReadout    fyne.CanvasObject
	emuIdle        fyne.CanvasObject
	machinePhase   string
	panelFocused   bool
	applyingFocus  bool
	focusSavedView viewMode
	focusSavedMain float64
	focusSavedLeft float64
	focusSavedDiag bool
	audioDev       sdl.AudioDeviceID
	audioFrame     []byte

	frameImages [2]*image.RGBA
	frameIdx    int

	updateLoopStop chan struct{}
	updateLoopOnce sync.Once

	keyMu            sync.Mutex
	keyStates        map[fyne.KeyName]bool
	typedKeyUntil    map[fyne.KeyName]time.Time
	desktopKeyEvents bool
	captureGameInput bool
	spriteLabHotkey  func(fyne.KeyName) bool
	spriteLabUndo    func()
	spriteLabRedo    func()

	suppressSourceChange bool
	diagnosticsCollapsed bool
}

func main() {
	openPath := flag.String("file", "", "CoreLX source file to open")
	audioBackend := flag.String("audio-backend", "", "Audio backend override: ymfm (default: ymfm)")
	flag.Parse()
	if err := applyAudioBackendSetting(*audioBackend); err != nil {
		fmt.Fprintf(os.Stderr, "invalid audio backend: %v\n", err)
		os.Exit(1)
	}

	tempDir, err := os.MkdirTemp("", "nitro-core-dx-devkit-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	a := app.New()
	if iconRes := loadDevKitIconResource(); iconRes != nil {
		a.SetIcon(iconRes)
	}

	settingsPath := devKitSettingsPath()
	settings, settingsErr := loadDevKitSettings(settingsPath)
	launchDir, _ := os.Getwd()

	a.Settings().SetTheme(newNitroTheme(settings.Theme, settings.UIDensity))

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
		tempDir:              tempDir,
		launchDir:            launchDir,
		settingsPath:         settingsPath,
		settings:             settings,
		autosavePath:         devKitAutosavePath(settingsPath),
		window:               w,
		currentView:          initialView,
		statusLabel:          widget.NewLabel("Ready"),
		buildStateLabel:      widget.NewLabel("Draft"),
		pathLabel:            widget.NewLabel("Untitled.corelx"),
		diagnostics:          make([]corelx.Diagnostic, 0),
		filteredDiagnostics:  make([]corelx.Diagnostic, 0),
		buildOutput:          newReadOnlyTextArea(),
		manifestOutput:       newReadOnlyTextArea(),
		diagnosticDetail:     newReadOnlyTextArea(),
		emuScale:             2,
		keyStates:            make(map[fyne.KeyName]bool),
		typedKeyUntil:        make(map[fyne.KeyName]time.Time),
		captureGameInput:     settings.CaptureGameInput,
		updateLoopStop:       make(chan struct{}),
		audioFrame:           make([]byte, 735*2*4),
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
	// Apply X11 maximize hint before first show so WM sees it when mapping (Linux/X11).
	_ = applyX11MaximizeHint(state.window)
	state.scheduleX11MaximizeHintRefresh()
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
		state.writeAutosaveSnapshot(state.sourceEditor.Text())
		state.stopEmulatorLoop()
		state.shutdownEmbeddedEmulator()
		state.shutdownAudio()
		state.persistSettings()
		w.Close()
	})

	w.ShowAndRun()
}

func applyAudioBackendSetting(flagValue string) error {
	mode := strings.ToLower(strings.TrimSpace(flagValue))
	if mode == "" {
		if strings.TrimSpace(os.Getenv("NCDX_YM_BACKEND")) == "" {
			return os.Setenv("NCDX_YM_BACKEND", "ymfm")
		}
		return nil
	}
	if mode != "ymfm" {
		return fmt.Errorf("%q (expected ymfm)", flagValue)
	}
	return os.Setenv("NCDX_YM_BACKEND", mode)
}

func (s *devKitState) scheduleX11MaximizeHintRefresh() {
	go func() {
		// Retry at several times so we apply as soon as the native handle is available
		// (often only after Show()). Early attempts catch WMs that read hints at map time.
		delays := []time.Duration{
			0,
			20 * time.Millisecond,
			60 * time.Millisecond,
			150 * time.Millisecond,
			400 * time.Millisecond,
			1000 * time.Millisecond,
			2000 * time.Millisecond,
		}
		var applied bool
		for _, delay := range delays {
			time.Sleep(delay)
			if err := applyX11MaximizeHint(s.window); err != nil {
				continue
			}
			applied = true
			appendWindowHintLog("window hint: X11 maximize hint applied")
			break
		}
		if !applied {
			appendWindowHintLog("window hint: X11 maximize hint not applied (Wayland or no native handle)")
		}
	}()
}

func (s *devKitState) initUI() {
	s.sourceEditor = newCoreLXCodeEditor()
	s.sourceEditor.SetText(defaultTemplate)
	s.sourceEditor.SetOnChanged(func(text string) {
		if s.suppressSourceChange {
			return
		}
		s.dirty = true
		s.refreshTitle()
		s.writeAutosaveSnapshot(text)
		s.setBuildState("Draft")
	})

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
	s.hardwareLamp = newStatusLamp(activePalette.disabled)
	s.machinePhase = "idle"
	s.emuLabel = widget.NewLabel("No program loaded. Build + Run or Load ROM.")
	s.emuWell = canvas.NewRectangle(activePalette.recessed)
	s.emuIdle = s.newEmulatorIdleCard()
	s.emuSurface = container.NewStack(s.emuImage, s.emuIdle, s.emuKeys)

	s.captureCheck = widget.NewCheck("Capture Input", func(v bool) {
		s.captureGameInput = v
		if !v {
			s.applyInputButtons(0)
		}
		s.settings.CaptureGameInput = v
		s.persistSettings()
	})
	s.captureCheck.SetChecked(s.captureGameInput)

	s.diagnosticSummary = widget.NewLabel("No build issues")
	s.diagnosticSummary.Importance = widget.SuccessImportance
	s.diagnosticsToggle = widget.NewButton("Collapse", func() {
		s.toggleDiagnosticsPanel()
	})
	searchBox := container.NewGridWrap(fyne.NewSize(220, 36), s.diagnosticSearch)
	diagToolbar := container.NewVBox(
		container.NewHBox(s.diagnosticSummary, s.diagnosticFilter, layout.NewSpacer(), s.diagnosticsToggle),
		searchBox,
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
		container.NewVBox(panelHeader("SOURCE"), s.buildEditorFindBar()),
		nil, nil, nil,
		s.sourceEditor,
	)
	spriteLabPane := s.buildSpriteLabPane()
	tilemapPane := s.buildTilemapPane()
	soundPane := s.buildSoundStudioPane()
	s.workbenchTabs = container.NewAppTabs(
		container.NewTabItem("Code", s.editorPane),
		container.NewTabItem("Sprite Lab", spriteLabPane),
		container.NewTabItem("Tilemap", tilemapPane),
		container.NewTabItem("Sound", soundPane),
	)

	s.centerHost = container.NewMax()
	s.contentRoot = container.NewBorder(
		container.NewVBox(s.buildToolbar(), newHazardStripe()),
		s.statusLabel, nil, nil, s.centerHost,
	)
	s.window.SetContent(s.contentRoot)
	s.window.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyZ, Modifier: fyne.KeyModifierControl}, func(fyne.Shortcut) {
		if !s.spriteLabHotkeysEnabled() || s.spriteLabUndo == nil {
			return
		}
		s.spriteLabUndo()
	})
	s.window.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyY, Modifier: fyne.KeyModifierControl}, func(fyne.Shortcut) {
		if !s.spriteLabHotkeysEnabled() || s.spriteLabRedo == nil {
			return
		}
		s.spriteLabRedo()
	})
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

	stepFrameBtn := widget.NewButton("Step Frame", func() { s.stepFrame() })
	stepCPUBtn := widget.NewButton("Step CPU", func() { s.stepCPU() })
	markFrameBtn := widget.NewButton("Mark Frame", func() { s.markCurrentFrame() })

	s.splitViewBtn = widget.NewButton("Split View", func() { s.setViewMode(viewModeFull) })
	s.emulatorFocusBtn = widget.NewButton("Emulator Focus", func() { s.setViewMode(viewModeEmulatorOnly) })
	s.codeOnlyBtn = widget.NewButton("Code Only", func() { s.setViewMode(viewModeCodeOnly) })
	s.focusPanelBtn = widget.NewButton("Focus Panel", func() { s.togglePanelFocus() })
	s.refreshViewToggleButtons()

	loadROMBtn := widget.NewButton("Load ROM", func() { s.openROMDialog() })

	s.brandLamp = newStatusLamp(activePalette.hazard)
	s.wordmark = canvas.NewText("NITRO CORE DX", activePalette.hazard)
	s.wordmark.TextStyle = fyne.TextStyle{Bold: true}
	s.wordmark.TextSize = 12
	identity := container.NewHBox(s.brandLamp, s.wordmark, layout.NewSpacer(), s.pathLabel, s.buildStateLabel)
	tools := container.NewHBox(
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
		widget.NewSeparator(),
		stepFrameBtn,
		stepCPUBtn,
		markFrameBtn,
		widget.NewSeparator(),
		s.codeOnlyBtn,
		s.splitViewBtn,
		s.emulatorFocusBtn,
		s.focusPanelBtn,
	)
	return container.NewVBox(identity, tools)
}

func (s *devKitState) setViewMode(mode viewMode) {
	if !s.applyingFocus {
		s.panelFocused = false
		if s.focusPanelBtn != nil {
			s.focusPanelBtn.SetText("Focus Panel")
		}
	}
	s.captureLayoutState()
	s.currentView = mode
	s.settings.ViewMode = string(mode)
	s.persistSettings()

	switch mode {
	case viewModeEmulatorOnly:
		s.centerHost.Objects = []fyne.CanvasObject{s.emulatorInstrument()}
		s.setStatus("View: Emulator Focus")
		if s.captureGameInput {
			s.focusEmulatorInput()
		}
	case viewModeCodeOnly:
		codeLayout := container.NewVSplit(s.workbenchTabs, s.bottomLeftTabs)
		codeLayout.Offset = 0.72
		if len(s.diagnostics) == 0 {
			codeLayout.Offset = 0.86
		}
		s.codeSplit = codeLayout
		s.centerHost.Objects = []fyne.CanvasObject{codeLayout}
		s.setStatus("View: Code Only")
	default:
		s.leftSplit = container.NewVSplit(s.emulatorInstrument(), s.bottomLeftTabs)
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

func (s *devKitState) emulatorInstrument() fyne.CanvasObject {
	title := widget.NewLabel("DISPLAY")
	title.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	if s.emuWell == nil {
		s.emuWell = canvas.NewRectangle(activePalette.recessed)
	}
	s.emuWell.FillColor = activePalette.recessed
	header := container.NewBorder(nil, nil, container.NewHBox(s.hardwareLamp, title), s.captureCheck, s.emuLabel)
	view := container.NewStack(s.emuWell, container.NewPadded(s.emuSurface))
	return container.NewBorder(header, nil, nil, nil, view)
}

func (s *devKitState) setHardwareReadout(phase, detail string) {
	s.machinePhase = phase
	if s.hardwareLamp != nil {
		switch phase {
		case "running":
			s.hardwareLamp.SetFill(activePalette.ok)
		case "paused":
			s.hardwareLamp.SetFill(activePalette.warning)
		case "stopped":
			s.hardwareLamp.SetFill(activePalette.err)
		default:
			s.hardwareLamp.SetFill(activePalette.disabled)
		}
	}
	if s.emuLabel != nil && detail != "" {
		s.emuLabel.SetText(detail)
	}
	if s.emuIdle != nil {
		if phase == "idle" {
			s.emuIdle.Show()
		} else {
			s.emuIdle.Hide()
		}
	}
}

// displayIdle is the empty-display readout. It sits in the top-left of the
// well as instrument status, and it disappears once a program is running.
type displayIdle struct {
	widget.BaseWidget
}

func newDisplayIdle() *displayIdle {
	d := &displayIdle{}
	d.ExtendBaseWidget(d)
	return d
}

func (d *displayIdle) CreateRenderer() fyne.WidgetRenderer {
	line := func(text string) *canvas.Text {
		t := canvas.NewText(text, activePalette.secondary)
		t.TextStyle = fyne.TextStyle{Monospace: true}
		t.TextSize = 11
		return t
	}
	state := line("STATE     IDLE")
	program := line("PROGRAM   NONE")
	return &displayIdleRenderer{state: state, program: program, objs: []fyne.CanvasObject{state, program}}
}

type displayIdleRenderer struct {
	state   *canvas.Text
	program *canvas.Text
	objs    []fyne.CanvasObject
}

func (r *displayIdleRenderer) Layout(fyne.Size) {
	r.state.Resize(r.state.MinSize())
	r.program.Resize(r.program.MinSize())
	r.state.Move(fyne.NewPos(10, 8))
	r.program.Move(fyne.NewPos(10, 8+r.state.MinSize().Height+2))
}

func (r *displayIdleRenderer) MinSize() fyne.Size {
	return fyne.NewSize(160, 36)
}

func (r *displayIdleRenderer) Refresh() {
	r.state.Color = activePalette.secondary
	r.program.Color = activePalette.secondary
	r.state.Refresh()
	r.program.Refresh()
}

func (r *displayIdleRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *displayIdleRenderer) Destroy()                     {}

func (s *devKitState) newEmulatorIdleCard() fyne.CanvasObject {
	readout := newDisplayIdle()
	s.idleReadout = readout
	return readout
}

func (s *devKitState) togglePanelFocus() {
	if s.panelFocused {
		s.panelFocused = false
		s.applyingFocus = true
		s.diagnosticsCollapsed = s.focusSavedDiag
		s.settings.MainSplitOffset = s.focusSavedMain
		s.settings.LeftSplitOffset = s.focusSavedLeft
		s.setViewMode(s.focusSavedView)
		s.applyingFocus = false
		if s.focusPanelBtn != nil {
			s.focusPanelBtn.SetText("Focus Panel")
		}
		s.setStatus("Layout restored")
		return
	}
	s.focusSavedView = s.currentView
	s.focusSavedMain = s.settings.MainSplitOffset
	s.focusSavedLeft = s.settings.LeftSplitOffset
	s.focusSavedDiag = s.diagnosticsCollapsed
	s.panelFocused = true
	s.applyingFocus = true
	s.setViewMode(viewModeCodeOnly)
	s.applyingFocus = false
	if s.codeSplit != nil {
		s.codeSplit.Offset = 0.94
		s.codeSplit.Refresh()
	}
	if s.focusPanelBtn != nil {
		s.focusPanelBtn.SetText("Restore Layout")
	}
	s.setStatus("Active panel focused")
}

func (s *devKitState) applyAppearance() {
	choice := normalizeThemeName(s.settings.Theme)
	density := s.settings.UIDensity
	if density != "standard" {
		density = "compact"
	}
	fyne.CurrentApp().Settings().SetTheme(newNitroTheme(choice, density))
	if s.wordmark != nil {
		s.wordmark.Color = activePalette.hazard
		s.wordmark.Refresh()
	}
	if s.brandLamp != nil {
		s.brandLamp.SetFill(activePalette.hazard)
	}
	if s.idleReadout != nil {
		s.idleReadout.Refresh()
	}
	if s.emuWell != nil {
		s.emuWell.FillColor = activePalette.recessed
		s.emuWell.Refresh()
	}
	if s.hardwareLamp != nil {
		s.setHardwareReadout(s.machinePhase, "")
	}
	if s.sourceEditor != nil {
		s.sourceEditor.scheduleRefresh()
	}
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
		if errCount == 0 && warnCount == 0 && infoCount == 0 {
			s.diagnosticSummary.Importance = widget.SuccessImportance
			s.diagnosticSummary.SetText("No build issues")
		} else {
			if errCount > 0 {
				s.diagnosticSummary.Importance = widget.DangerImportance
			} else if warnCount > 0 {
				s.diagnosticSummary.Importance = widget.WarningImportance
			} else {
				s.diagnosticSummary.Importance = widget.MediumImportance
			}
			s.diagnosticSummary.SetText(fmt.Sprintf("Errors: %d  Warnings: %d  Info: %d", errCount, warnCount, infoCount))
		}
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
	preview.SetMinRowsVisible(6)
	preview.SetPlaceHolder("Select a recent project to preview its source.")
	selectedRecent := ""

	recentOptions := append([]string{}, s.settings.RecentFiles...)
	applyRecent := func(path string) {
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
	}
	recentList := widget.NewList(
		func() int { return len(recentOptions) },
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("project/path.corelx")
			lbl.Wrapping = fyne.TextWrapWord
			return lbl
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id < 0 || id >= len(recentOptions) {
				lbl.SetText("")
				return
			}
			lbl.SetText(recentOptions[id])
		},
	)
	recentList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(recentOptions) {
			return
		}
		applyRecent(recentOptions[id])
	}
	emptyRecent := widget.NewLabel("No recent projects. Browse to open a CoreLX source file or a ROM.")
	emptyRecent.Wrapping = fyne.TextWrapWord

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

	recentBody := fyne.CanvasObject(recentList)
	if len(recentOptions) == 0 {
		recentBody = emptyRecent
	}
	// The list owns a split pane, so its rows scroll inside that pane and
	// cannot draw over the preview or the action buttons.
	recentSplit := container.NewVSplit(recentBody, preview)
	recentSplit.Offset = 0.42
	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Open Project"),
			filterSel,
			recentLabel,
		),
		buttons,
		nil, nil,
		recentSplit,
	)

	d = dialog.NewCustom("Open Project", "Close", content, s.window)
	d.Resize(fyne.NewSize(720, 420))
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
	if loc := dialogListableForDir(s.defaultProjectOpenDir()); loc != nil {
		fd.SetLocation(loc)
	}
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
	if loc := dialogListableForDir(s.defaultROMDialogDir()); loc != nil {
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
		if _, writeErr := wc.Write([]byte(s.sourceEditor.Text())); writeErr != nil {
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
	if err := os.WriteFile(s.currentPath, []byte(s.sourceEditor.Text()), 0644); err != nil {
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
	s.sourceEditor.SetText(text)
	s.suppressSourceChange = false
	s.setBuildState("Draft")
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
	s.setBuildState("Validating...")
	s.appendBuildOutput(fmt.Sprintf("Build started (%s)", sourcePath))

	buildResult, err := s.backend.BuildSource(s.sourceEditor.Text(), sourcePath)
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
		s.setBuildState("Error")
		s.setStatus(fmt.Sprintf("Build failed (%d errors)", bundle.Summary.ErrorCount))
		return
	}

	s.setBuildState("Validated")
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

func (s *devKitState) setBuildState(state string) {
	if s.buildStateLabel == nil {
		return
	}
	s.buildStateLabel.SetText(state)
	switch state {
	case "Validated":
		s.buildStateLabel.Importance = widget.SuccessImportance
	case "Error":
		s.buildStateLabel.Importance = widget.DangerImportance
	case "Validating...":
		s.buildStateLabel.Importance = widget.HighImportance
	default:
		s.buildStateLabel.Importance = widget.MediumImportance
	}
	s.buildStateLabel.Refresh()
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
		s.setHardwareReadout("running", "Hardware running")
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
						phase := "running"
						if paused {
							phase = "paused"
						}
						if s.machinePhase == "stopped" {
							phase = "stopped"
						}
						s.setHardwareReadout(phase, fmt.Sprintf(
							"%s  |  FPS %.1f  |  CPU %d cyc/frame  |  Frame %d  |  %s",
							phase,
							fps,
							cycles,
							frameCount,
							formatFrameClock(frameCount),
						))
						s.refreshDebuggerOutput()
					})
				}
			}
		}
	}()
}

func formatFrameClock(frameCount uint64) string {
	totalCentiseconds := (frameCount * 100) / 60
	minutes := totalCentiseconds / 6000
	seconds := (totalCentiseconds / 100) % 60
	centiseconds := totalCentiseconds % 100
	return fmt.Sprintf("%02d:%02d.%02d", minutes, seconds, centiseconds)
}

func formatFrameMark(snap devkit.EmulatorSnapshot) string {
	state := "stopped"
	if snap.Running {
		state = "running"
	}
	if snap.Paused {
		state = "paused"
	}
	return fmt.Sprintf(
		"Frame mark: frame=%d time=%s fps=%.1f cpu=%d state=%s",
		snap.FrameCount,
		formatFrameClock(snap.FrameCount),
		snap.FPS,
		snap.CPUCyclesPerFrame,
		state,
	)
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
	s.window.Canvas().SetOnTypedRune(func(r rune) {
		s.handleTypedRune(r)
	})
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

func (s *devKitState) handleTypedRune(r rune) {
	if s.spriteLabHotkeysEnabled() {
		switch r {
		case 'b', 'B':
			if s.spriteLabHotkey(fyne.KeyB) {
				return
			}
		case 'e', 'E':
			if s.spriteLabHotkey(fyne.KeyE) {
				return
			}
		case 'x', 'X':
			if s.spriteLabHotkey(fyne.KeyX) {
				return
			}
		case 'g', 'G':
			if s.spriteLabHotkey(fyne.KeyG) {
				return
			}
		case 't', 'T':
			if s.spriteLabHotkey(fyne.KeyT) {
				return
			}
		}
	}
	if s.isSourceEditorFocused() || !s.shouldCaptureGameInput() {
		if !s.dispatchTypedRuneToFocused(r) {
			s.dispatchTypedRuneToSourceEditor(r)
		}
		return
	}

	// Preserve emulator input fallback behavior for typed-rune-only backends.
	s.keyMu.Lock()
	if !s.desktopKeyEvents {
		switch r {
		case 'w', 'W':
			s.typedKeyUntil[fyne.KeyW] = time.Now().Add(450 * time.Millisecond)
		case 'a', 'A':
			s.typedKeyUntil[fyne.KeyA] = time.Now().Add(450 * time.Millisecond)
		case 's', 'S':
			s.typedKeyUntil[fyne.KeyS] = time.Now().Add(450 * time.Millisecond)
		case 'd', 'D':
			s.typedKeyUntil[fyne.KeyD] = time.Now().Add(450 * time.Millisecond)
		case 'z', 'Z':
			s.typedKeyUntil[fyne.KeyZ] = time.Now().Add(450 * time.Millisecond)
		case 'x', 'X':
			s.typedKeyUntil[fyne.KeyX] = time.Now().Add(450 * time.Millisecond)
		case 'c', 'C':
			s.typedKeyUntil[fyne.KeyC] = time.Now().Add(450 * time.Millisecond)
		case 'v', 'V':
			s.typedKeyUntil[fyne.KeyV] = time.Now().Add(450 * time.Millisecond)
		case 'q', 'Q':
			s.typedKeyUntil[fyne.KeyQ] = time.Now().Add(450 * time.Millisecond)
		case 'e', 'E':
			s.typedKeyUntil[fyne.KeyE] = time.Now().Add(450 * time.Millisecond)
		}
	}
	s.keyMu.Unlock()
	s.routeInputToEmulator()
}

func (s *devKitState) handleTypedKey(key *fyne.KeyEvent) {
	if key == nil {
		return
	}
	if s.spriteLabHotkeysEnabled() {
		if s.spriteLabHotkey(key.Name) {
			return
		}
	}
	if s.isSourceEditorFocused() || !s.shouldCaptureGameInput() {
		if !s.dispatchTypedKeyToFocused(key) {
			s.dispatchTypedKeyToSourceEditor(key)
		}
		return
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
	if s.spriteLabHotkeysEnabled() {
		if s.spriteLabHotkey(key.Name) {
			return
		}
	}
	if s.isSourceEditorFocused() || !s.shouldCaptureGameInput() {
		s.dispatchKeyDownToFocused(key)
		return
	}
	s.keyMu.Lock()
	s.keyStates[key.Name] = true
	s.keyMu.Unlock()
	s.routeInputToEmulator()
}

func (s *devKitState) spriteLabHotkeysEnabled() bool {
	if s.spriteLabHotkey == nil || s.workbenchTabs == nil {
		return false
	}
	tab := s.workbenchTabs.CurrentTab()
	if tab == nil {
		return false
	}
	return tab.Text == "Sprite Lab"
}

func (s *devKitState) handleKeyUp(key *fyne.KeyEvent) {
	if key == nil {
		return
	}
	if s.isSourceEditorFocused() || !s.shouldCaptureGameInput() {
		s.dispatchKeyUpToFocused(key)
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

func (s *devKitState) shouldCaptureGameInput() bool {
	if !s.captureGameInput {
		return false
	}
	// "Capture Game Input" is an explicit override. When enabled, route keys to the
	// embedded emulator regardless of which editor widget currently has focus.
	// This matches user expectation for Build+Run testing inside the Dev Kit.
	return true
}

func (s *devKitState) isSourceEditorFocused() bool {
	if s.window == nil || s.window.Canvas() == nil || s.sourceEditor == nil {
		return false
	}
	return s.window.Canvas().Focused() == s.sourceEditor
}

func (s *devKitState) dispatchTypedRuneToFocused(r rune) bool {
	if s.window == nil || s.window.Canvas() == nil {
		return false
	}
	if focused := s.window.Canvas().Focused(); focused != nil {
		if s.emuKeys != nil && focused == s.emuKeys {
			return false
		}
		focused.TypedRune(r)
		return true
	}
	return false
}

func (s *devKitState) dispatchTypedKeyToFocused(key *fyne.KeyEvent) bool {
	if key == nil || s.window == nil || s.window.Canvas() == nil {
		return false
	}
	if focused := s.window.Canvas().Focused(); focused != nil {
		if s.emuKeys != nil && focused == s.emuKeys {
			return false
		}
		focused.TypedKey(key)
		return true
	}
	return false
}

func (s *devKitState) dispatchTypedRuneToSourceEditor(r rune) {
	if s.window == nil || s.window.Canvas() == nil || s.sourceEditor == nil {
		return
	}
	s.window.Canvas().Focus(s.sourceEditor)
	s.sourceEditor.TypedRune(r)
}

func (s *devKitState) dispatchTypedKeyToSourceEditor(key *fyne.KeyEvent) {
	if key == nil || s.window == nil || s.window.Canvas() == nil || s.sourceEditor == nil {
		return
	}
	s.window.Canvas().Focus(s.sourceEditor)
	s.sourceEditor.TypedKey(key)
}

func (s *devKitState) dispatchKeyDownToFocused(key *fyne.KeyEvent) {
	if key == nil || s.window == nil || s.window.Canvas() == nil {
		return
	}
	if focused := s.window.Canvas().Focused(); focused != nil {
		if s.emuKeys != nil && focused == s.emuKeys {
			return
		}
		if k, ok := focused.(desktop.Keyable); ok {
			k.KeyDown(key)
		}
	}
}

func (s *devKitState) dispatchKeyUpToFocused(key *fyne.KeyEvent) {
	if key == nil || s.window == nil || s.window.Canvas() == nil {
		return
	}
	if focused := s.window.Canvas().Focused(); focused != nil {
		if s.emuKeys != nil && focused == s.emuKeys {
			return
		}
		if k, ok := focused.(desktop.Keyable); ok {
			k.KeyUp(key)
		}
	}
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
	s.window.Canvas().Focus(s.sourceEditor)
	s.sourceEditor.SetCursor(maxInt(0, row), maxInt(0, col))
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
	mark := func(btn *widget.Button, label string, on bool) {
		if btn == nil {
			return
		}
		btn.Importance = widget.MediumImportance
		if on {
			btn.SetText("▸ " + label)
		} else {
			btn.SetText(label)
		}
		btn.Refresh()
	}
	mark(s.codeOnlyBtn, "Code Only", s.currentView == viewModeCodeOnly)
	mark(s.splitViewBtn, "Split View", s.currentView == viewModeFull)
	mark(s.emulatorFocusBtn, "Emulator Focus", s.currentView == viewModeEmulatorOnly)
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
	s.setHardwareReadout("running", "Hardware running")
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
	s.setHardwareReadout("paused", "Hardware paused")
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
	s.setHardwareReadout("stopped", "Hardware stopped")
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

func (s *devKitState) markCurrentFrame() {
	snap := s.backend.Snapshot()
	if !snap.Loaded {
		s.setStatus("No active project build")
		return
	}
	line := formatFrameMark(snap)
	s.appendBuildOutput(line)
	s.setStatus(line)
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
	if density != "standard" {
		density = "compact"
	}
	s.settings.UIDensity = density
	s.persistSettings()
	s.applyAppearance()
	s.setStatus("UI density: " + density)
}

func (s *devKitState) setThemeChoice(choice string) {
	s.settings.Theme = normalizeThemeName(choice)
	s.persistSettings()
	s.applyAppearance()
	if s.settings.Theme == themeClassic {
		s.setStatus("Theme: Nitro Classic")
		return
	}
	s.setStatus("Theme: Nitro Industrial")
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

func loadDevKitIconResource() fyne.Resource {
	candidates := []string{
		filepath.Join("Images", "ncdxiconbg.png"),
		filepath.Join("..", "..", "Images", "ncdxiconbg.png"),
	}
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(base, "Images", "ncdxiconbg.png"),
			filepath.Join(base, "..", "Images", "ncdxiconbg.png"),
		)
	}
	for _, p := range candidates {
		res, err := fyne.LoadResourceFromPath(p)
		if err == nil && res != nil {
			return res
		}
	}
	return nil
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
