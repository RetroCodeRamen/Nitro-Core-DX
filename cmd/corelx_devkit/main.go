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
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/veandco/go-sdl2/sdl"
	"nitro-core-dx/internal/apu"
	"nitro-core-dx/internal/corelx"
	"nitro-core-dx/internal/devkit"
)

const (
	devKitScreenW = 320
	devKitScreenH = 200
)

const defaultTemplate = `function Start()
    apu.enable()
`

type viewMode string

const (
	viewModeFull         viewMode = "full"
	viewModeEmulatorOnly viewMode = "emulator_only"
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
func (w *emulatorKeyOverlay) FocusGained()                  {}
func (w *emulatorKeyOverlay) FocusLost()                    {}
func (w *emulatorKeyOverlay) TypedRune(r rune)              {}

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

	tempDir string

	currentPath string
	lastROMPath string
	dirty       bool

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

	diagnosticFilter *widget.Select
	diagnosticSearch *widget.Entry
	diagnosticsList  *widget.List
	diagnosticDetail *widget.Entry

	fullLayout         fyne.CanvasObject
	emulatorOnlyLayout fyne.CanvasObject
	emulatorPane       fyne.CanvasObject
	bottomLeftTabs     *container.AppTabs
	editorPane         fyne.CanvasObject
	workbenchTabs      *container.AppTabs

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
	w := a.NewWindow("Nitro-Core-DX")
	w.Resize(fyne.NewSize(1500, 920))

	state := &devKitState{
		tempDir:             tempDir,
		window:              w,
		currentView:         viewModeFull,
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
		captureGameInput:    true,
		updateLoopStop:      make(chan struct{}),
		audioFrame:          make([]byte, 735*2*4),
	}
	state.backend = devkit.NewService(tempDir)
	if err := state.initAudio(); err != nil {
		state.appendBuildOutput("Audio init warning: " + err.Error())
		state.setStatus("Ready (audio unavailable)")
	}
	state.initUI()
	state.setupKeyboardInput()
	state.startEmulatorLoop()

	if *openPath != "" {
		if err := state.loadFile(*openPath); err != nil {
			state.appendBuildOutput(fmt.Sprintf("Open error: %v", err))
			state.setStatus("Open failed")
		}
	} else {
		state.dirty = false
		state.refreshTitle()
	}

	w.SetCloseIntercept(func() {
		state.stopEmulatorLoop()
		state.shutdownEmbeddedEmulator()
		state.shutdownAudio()
		w.Close()
	})

	w.ShowAndRun()
}

func (s *devKitState) initUI() {
	s.sourceEntry = widget.NewMultiLineEntry()
	s.sourceEntry.SetText(defaultTemplate)
	s.sourceEntry.Wrapping = fyne.TextWrapOff
	s.sourceEntry.OnChanged = func(string) {
		s.dirty = true
		s.refreshTitle()
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

	s.diagnosticsList = widget.NewList(
		func() int { return len(s.filteredDiagnostics) },
		func() fyne.CanvasObject { return widget.NewLabel("diagnostic") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id < 0 || id >= len(s.filteredDiagnostics) {
				lbl.SetText("")
				return
			}
			d := s.filteredDiagnostics[id]
			loc := ""
			if d.Line > 0 {
				loc = fmt.Sprintf(":%d:%d", d.Line, maxInt(1, d.Column))
			}
			lbl.SetText(fmt.Sprintf("[%s/%s] %s%s %s", d.Severity, d.Stage, baseNameOr(d.File, "<buffer>"), loc, d.Message))
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
	s.emuLabel = widget.NewLabel("Emulator: idle")
	emuSurface := container.NewStack(s.emuImage, s.emuKeys)

	emuToolbar := container.NewHBox(
		widget.NewButton("Reset", func() {
			if err := s.backend.ResetEmulator(); err != nil {
				s.setStatus("No ROM loaded")
				return
			}
			s.setStatus("Emulator reset")
		}),
		widget.NewButton("Pause/Resume", func() {
			paused, err := s.backend.TogglePause()
			if err != nil {
				s.setStatus("No ROM loaded")
				return
			}
			if paused {
				s.setStatus("Emulator paused")
			} else {
				s.setStatus("Emulator resumed")
			}
		}),
		widget.NewCheck("Capture Game Input", func(v bool) {
			s.captureGameInput = v
			if !v {
				s.applyInputButtons(0)
			}
		}),
	)
	// ensure initial check state shows current value
	if len(emuToolbar.Objects) >= 3 {
		if chk, ok := emuToolbar.Objects[2].(*widget.Check); ok {
			chk.SetChecked(s.captureGameInput)
		}
	}

	s.emulatorPane = container.NewBorder(
		container.NewVBox(widget.NewLabel("Emulator"), s.emuLabel, emuToolbar),
		nil, nil, nil,
		emuSurface,
	)

	diagToolbar := container.NewHBox(
		widget.NewLabel("Diagnostics"),
		s.diagnosticFilter,
		s.diagnosticSearch,
	)
	diagPane := container.NewBorder(
		diagToolbar,
		nil, nil, nil,
		container.NewVSplit(s.diagnosticsList, s.diagnosticDetail),
	)
	outputPane := container.NewBorder(widget.NewLabel("Build Output"), nil, nil, nil, s.buildOutput)
	manifestPane := container.NewBorder(widget.NewLabel("Manifest / Memory Summary"), nil, nil, nil, s.manifestOutput)
	s.bottomLeftTabs = container.NewAppTabs(
		container.NewTabItem("Diagnostics", diagPane),
		container.NewTabItem("Output", outputPane),
		container.NewTabItem("Manifest", manifestPane),
	)

	s.editorPane = container.NewBorder(
		container.NewVBox(widget.NewLabel("CoreLX Editor"), s.pathLabel),
		nil, nil, nil,
		s.sourceEntry,
	)
	spriteLabPlaceholder := widget.NewLabel("Sprite Lab (coming next)\n\nPlanned: palette-index sprite editor + .clxasset round-trip export/import.")
	tilemapPlaceholder := widget.NewLabel("Tilemap Editor (coming next)\n\nPlanned: grid placement + CoreLX asset export.")
	soundStudioPlaceholder := widget.NewLabel("Sound Studio (coming next)\n\nPlanned: music/ambience/SFX authoring integrated with packaging.")
	s.workbenchTabs = container.NewAppTabs(
		container.NewTabItem("Code", s.editorPane),
		container.NewTabItem("Sprite Lab", container.NewScroll(spriteLabPlaceholder)),
		container.NewTabItem("Tilemap", container.NewScroll(tilemapPlaceholder)),
		container.NewTabItem("Sound", container.NewScroll(soundStudioPlaceholder)),
	)

	leftSplit := container.NewVSplit(s.emulatorPane, s.bottomLeftTabs)
	leftSplit.Offset = 0.53
	fullSplit := container.NewHSplit(leftSplit, s.workbenchTabs)
	fullSplit.Offset = 0.43
	s.fullLayout = fullSplit

	s.emulatorOnlyLayout = container.NewBorder(
		container.NewVBox(widget.NewLabel("Emulator (Nitro-Core-DX Integrated View)"), s.emuLabel),
		nil, nil, nil,
		emuSurface,
	)

	s.centerHost = container.NewMax()
	s.contentRoot = container.NewBorder(s.buildToolbar(), s.statusLabel, nil, nil, s.centerHost)
	s.window.SetContent(s.contentRoot)
	s.setViewMode(viewModeFull)
}

func newReadOnlyTextArea() *widget.Entry {
	e := widget.NewMultiLineEntry()
	e.Wrapping = fyne.TextWrapOff
	e.Disable()
	return e
}

func (s *devKitState) buildToolbar() fyne.CanvasObject {
	openBtn := widget.NewButton("Open", func() { s.openDialog() })
	loadROMBtn := widget.NewButton("Load ROM", func() { s.openROMDialog() })
	saveBtn := widget.NewButton("Save", func() {
		if err := s.save(); err != nil {
			dialog.ShowError(err, s.window)
			s.setStatus("Save failed")
			return
		}
		s.setStatus("Saved")
	})
	saveAsBtn := widget.NewButton("Save As", func() { s.saveAsDialog() })
	buildBtn := widget.NewButton("Build", func() { s.runBuild(false) })
	buildRunBtn := widget.NewButton("Build + Run", func() { s.runBuild(true) })
	fullViewBtn := widget.NewButton("Full View", func() { s.setViewMode(viewModeFull) })
	emuOnlyBtn := widget.NewButton("Emulator Only", func() { s.setViewMode(viewModeEmulatorOnly) })

	return container.NewHBox(
		openBtn,
		loadROMBtn,
		saveBtn,
		saveAsBtn,
		widget.NewSeparator(),
		buildBtn,
		buildRunBtn,
		widget.NewSeparator(),
		fullViewBtn,
		emuOnlyBtn,
	)
}

func (s *devKitState) setViewMode(mode viewMode) {
	s.currentView = mode
	if mode == viewModeEmulatorOnly {
		s.centerHost.Objects = []fyne.CanvasObject{s.emulatorOnlyLayout}
		s.setStatus("View: Emulator Only")
		if s.captureGameInput {
			s.focusEmulatorInput()
		}
	} else {
		s.centerHost.Objects = []fyne.CanvasObject{s.fullLayout}
		s.setStatus("View: Full")
	}
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
	for _, d := range s.diagnostics {
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
		s.sourceEntry.SetText(string(data))
		s.currentPath = uriPath(rc.URI())
		s.dirty = false
		s.refreshTitle()
		s.pathLabel.SetText(displayPath(s.currentPath))
		s.setStatus("Opened " + baseNameOr(s.currentPath, "buffer"))
		s.appendBuildOutput("Opened " + s.currentPath)
	}, s.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".corelx", ".clx", ".txt"}))
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
			s.appendBuildOutput("Load ROM failed: " + loadErr.Error())
			s.setStatus("Load ROM failed")
			return
		}
		s.lastROMPath = uriPath(rc.URI())
		s.setViewMode(viewModeFull)
		s.appendBuildOutput("Loaded ROM into embedded emulator: " + s.lastROMPath)
		s.setStatus("ROM loaded in embedded emulator")
	}, s.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".rom"}))
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
		s.setStatus("Saved")
		s.appendBuildOutput("Saved " + s.currentPath)
	}, s.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".corelx", ".clx", ".txt"}))
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
	s.appendBuildOutput("Saved " + s.currentPath)
	return nil
}

func (s *devKitState) loadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s.sourceEntry.SetText(string(data))
	s.currentPath = path
	s.dirty = false
	s.refreshTitle()
	s.pathLabel.SetText(displayPath(s.currentPath))
	s.appendBuildOutput("Opened " + s.currentPath)
	return nil
}

func (s *devKitState) refreshTitle() {
	name := "Untitled.corelx"
	if s.currentPath != "" {
		name = baseNameOr(s.currentPath, name)
	}
	if s.dirty {
		name += " *"
	}
	viewLabel := "Full"
	if s.currentView == viewModeEmulatorOnly {
		viewLabel = "Emulator Only"
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
			s.appendBuildOutput("ROM loaded into embedded emulator")
			s.setStatus("Build + Run loaded in embedded emulator")
		} else {
			s.setStatus("Build succeeded (no ROM bytes emitted)")
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
		sb.WriteString(fmt.Sprintf("ROM bytes (emitted/planned): %d / %d\n",
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
		s.emuLabel.SetText("Emulator: running (embedded)")
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
					s.appendBuildOutput("Emulator frame error: " + err.Error())
					s.setStatus("Emulator error")
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
						s.emuLabel.SetText(fmt.Sprintf("Emulator: %s | FPS %.1f | CPU %d cycles/frame | Frame %d", state, fps, cycles, frameCount))
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
			s.diagnosticDetail.SetText("No diagnostics")
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
