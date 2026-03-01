package ui

import (
	"encoding/binary"
	"fmt"
	"image"
	"io"
	"math"
	"sync"
	"time"

	"nitro-core-dx/internal/apu"
	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/emulator"
	"nitro-core-dx/internal/ui/panels"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/veandco/go-sdl2/sdl"
)

// FyneUI represents the Fyne-based UI with SDL2 for emulator rendering
type FyneUI struct {
	app      fyne.App
	window   fyne.Window
	emulator *emulator.Emulator
	scale    int
	running  bool
	paused   bool

	// SDL2 for emulator rendering
	sdlRenderer *sdl.Renderer
	sdlTexture  *sdl.Texture
	audioDev    sdl.AudioDeviceID
	audioFrame  []byte // Interleaved stereo float32 for one emulator frame (735 samples)

	// Fyne widgets
	emulatorImage *canvas.Image
	statusLabel   *widget.Label
	frameImages   [2]*image.RGBA
	frameImageIdx int

	// Debug panels
	showLogViewer bool
	showRegisters bool
	showMemory    bool
	showTiles     bool

	// Panel containers
	logViewerPanel *fyne.Container
	registersPanel *fyne.Container
	memoryPanel    *fyne.Container
	tilesPanel     *fyne.Container

	// Layout containers (for dynamic updates)
	splitContent *container.Split
	rightPanels  *fyne.Container
	mainContent  *fyne.Container

	// Panel update functions
	updateRegisters func()
	updateMemory    func()
	updateTiles     func()
	updateLogs      func()

	// Keyboard input state
	keyMu            sync.Mutex
	keyStates        map[fyne.KeyName]bool
	typedKeyUntil    map[fyne.KeyName]time.Time // fallback "held" lease for typed-only platforms
	desktopKeyEvents bool
}

// NewFyneUI creates a new Fyne-based UI
func NewFyneUI(emu *emulator.Emulator, scale int) (*FyneUI, error) {
	// Initialize SDL2 for audio, video, and events (needed for keyboard input)
	if err := sdl.Init(sdl.INIT_AUDIO | sdl.INIT_VIDEO | sdl.INIT_EVENTS); err != nil {
		return nil, fmt.Errorf("failed to initialize SDL: %w", err)
	}

	// Open audio device
	audioSpec := sdl.AudioSpec{
		Freq:     44100,
		Format:   sdl.AUDIO_F32,
		Channels: 2,
		Samples:  735,
	}
	audioDev, err := sdl.OpenAudioDevice("", false, &audioSpec, nil, 0)
	if err != nil {
		if emu.Logger != nil {
			emu.Logger.LogUI(debug.LogLevelWarning, fmt.Sprintf("Failed to open audio device: %v", err), nil)
		}
		audioDev = 0
	} else {
		sdl.PauseAudioDevice(audioDev, false)
	}

	// Create Fyne app
	fyneApp := app.NewWithID("com.nitro-core-dx.emulator")
	window := fyneApp.NewWindow("Nitro-Core-DX Emulator")

	// Create status label
	statusLabel := widget.NewLabel("FPS: 0.0 | CPU: 0 cycles/frame | Frame: 0")

	// Create reusable frame buffers for UI rendering (double-buffered to avoid UI thread races).
	frame0 := image.NewRGBA(image.Rect(0, 0, 320*scale, 200*scale))
	frame1 := image.NewRGBA(image.Rect(0, 0, 320*scale, 200*scale))
	// Create emulator image (will be updated with rendered frames)
	emulatorImage := canvas.NewImageFromImage(frame0)
	emulatorImage.FillMode = canvas.ImageFillContain

	// Create debug panels (initially hidden)
	registersPanel, updateRegistersFunc := panels.RegisterViewer(emu, window)
	registersPanel.Hide()

	memoryPanel, updateMemoryFunc := panels.MemoryViewer(emu)
	memoryPanel.Hide()

	tilesPanel, updateTilesFunc := panels.TileViewer(emu)
	tilesPanel.Hide()

	// Create log viewer panel (if logger is available)
	var logViewerPanel *fyne.Container
	var updateLogsFunc func()
	if emu.Logger != nil {
		logViewerPanel, updateLogsFunc = panels.LogViewerFyne(emu.Logger, window)
		logViewerPanel.Hide()
	}

	// Create UI instance first (needed for menu callbacks)
	ui := &FyneUI{
		app:             fyneApp,
		window:          window,
		emulator:        emu,
		scale:           scale,
		running:         false,
		paused:          false,
		audioDev:        audioDev,
		audioFrame:      make([]byte, 735*2*4),
		emulatorImage:   emulatorImage,
		statusLabel:     statusLabel,
		frameImages:     [2]*image.RGBA{frame0, frame1},
		registersPanel:  registersPanel,
		memoryPanel:     memoryPanel,
		tilesPanel:      tilesPanel,
		logViewerPanel:  logViewerPanel,
		updateRegisters: updateRegistersFunc,
		updateMemory:    updateMemoryFunc,
		updateTiles:     updateTilesFunc,
		updateLogs:      updateLogsFunc,
		keyStates:       make(map[fyne.KeyName]bool),
		typedKeyUntil:   make(map[fyne.KeyName]time.Time),
	}

	// Create right-side panel container (vertical stack for multiple panels)
	rightPanelsList := []fyne.CanvasObject{
		registersPanel,
		memoryPanel,
		tilesPanel,
	}
	if logViewerPanel != nil {
		rightPanelsList = append(rightPanelsList, logViewerPanel)
	}
	rightPanels := container.NewVBox(rightPanelsList...)

	// Create horizontal splitter for resizable panels
	// Left side: emulator screen, Right side: debug panels
	splitContent := container.NewHSplit(emulatorImage, rightPanels)
	// Initially hide panels by setting offset to 1.0 (fully to the right, panels hidden)
	splitContent.SetOffset(1.0)

	// Create main content with status bar at bottom
	// Always use splitter, but adjust offset to show/hide panels
	mainContent := container.NewBorder(
		nil,          // Top (no toolbar - use menu instead)
		statusLabel,  // Bottom
		nil,          // Left
		nil,          // Right
		splitContent, // Center (splitter with emulator and panels)
	)

	// Store references for dynamic layout updates
	ui.splitContent = splitContent
	ui.rightPanels = rightPanels
	ui.mainContent = mainContent

	window.SetContent(mainContent)
	// Set initial window size (emulator + panels + status bar)
	// Window is resizable, so this is just the initial size
	window.Resize(fyne.NewSize(float32(320*scale)+400, float32(200*scale)+50))
	window.CenterOnScreen()

	// Create menus (pass UI instance for panel toggling)
	createMenus(window, emu, ui)

	// Set up keyboard input handling
	setupKeyboardInput(window, ui)

	return ui, nil
}

// setupKeyboardInput sets up keyboard event handling to control the emulator
func setupKeyboardInput(window fyne.Window, ui *FyneUI) {
	// Button bit mappings (from HARDWARE_SPECIFICATION.md and test ROM):
	// Bit 0: UP
	// Bit 1: DOWN
	// Bit 2: LEFT
	// Bit 3: RIGHT
	// Bit 4: A
	// Bit 5: B
	// Bit 6: X
	// Bit 7: Y
	// Bit 8: L
	// Bit 9: R
	// Bit 10: START
	// Bit 11: Z (used as "Stop" in diagnostics)

	// Handle typed key events (fallback path).
	// On some platforms/toolkits, typed events may repeat while a key is held but there may be
	// no reliable key-up callback. We convert typed events into a short "held lease" that is
	// extended by repeats and expires when repeats stop.
	window.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		ui.keyMu.Lock()
		if !ui.desktopKeyEvents {
			ui.typedKeyUntil[key.Name] = time.Now().Add(450 * time.Millisecond)
		}
		ui.keyMu.Unlock()
		ui.updateInputFromKeys()
	})

	// Desktop platforms provide key down/up callbacks; use them for reliable key state tracking.
	if c, ok := window.Canvas().(desktop.Canvas); ok {
		ui.keyMu.Lock()
		ui.desktopKeyEvents = true
		ui.keyMu.Unlock()
		c.SetOnKeyDown(func(key *fyne.KeyEvent) {
			ui.keyMu.Lock()
			ui.keyStates[key.Name] = true
			ui.keyMu.Unlock()
			ui.updateInputFromKeys()
		})
		c.SetOnKeyUp(func(key *fyne.KeyEvent) {
			ui.keyMu.Lock()
			ui.keyStates[key.Name] = false
			delete(ui.typedKeyUntil, key.Name)
			ui.keyMu.Unlock()
			ui.updateInputFromKeys()
		})
	}
}

func (ui *FyneUI) applyFyneKeyStates(buttons uint16) uint16 {
	now := time.Now()

	ui.keyMu.Lock()
	defer ui.keyMu.Unlock()

	isPressed := func(key fyne.KeyName) bool {
		if ui.keyStates[key] {
			return true
		}
		if until, ok := ui.typedKeyUntil[key]; ok {
			if now.Before(until) {
				return true
			}
			delete(ui.typedKeyUntil, key)
		}
		return false
	}

	if isPressed(fyne.KeyW) || isPressed(fyne.KeyUp) {
		buttons |= 0x01 // UP
	}
	if isPressed(fyne.KeyS) || isPressed(fyne.KeyDown) {
		buttons |= 0x02 // DOWN
	}
	if isPressed(fyne.KeyA) || isPressed(fyne.KeyLeft) {
		buttons |= 0x04 // LEFT
	}
	if isPressed(fyne.KeyD) || isPressed(fyne.KeyRight) {
		buttons |= 0x08 // RIGHT
	}
	if isPressed(fyne.KeyZ) {
		buttons |= 0x10 // A
	}
	if isPressed(fyne.KeyX) {
		buttons |= 0x20 // B
	}
	if isPressed(fyne.KeyV) {
		buttons |= 0x40 // X
	}
	if isPressed(fyne.KeyC) {
		buttons |= 0x80 // Y
	}
	if isPressed(fyne.KeyQ) {
		buttons |= 0x100 // L
	}
	if isPressed(fyne.KeyE) {
		buttons |= 0x200 // R
	}
	if isPressed(fyne.KeyReturn) {
		buttons |= 0x400 // START
	}
	if isPressed(fyne.KeyBackspace) {
		buttons |= 0x800 // Z (used as STOP in diagnostics)
	}
	return buttons
}

// updateInputFromKeys updates the emulator's input state based on current SDL keyboard state
func (ui *FyneUI) updateInputFromKeys() {
	// Always start with 0 - only set bits if keys are actually pressed
	var buttons uint16 = 0

	// Use SDL2 keyboard state for accurate key tracking
	// Note: GetKeyboardState() may return nil if SDL isn't fully initialized
	// In that case, buttons stays 0 (no input)
	keyboardState := sdl.GetKeyboardState()
	if keyboardState != nil {
		// Map SDL2 scancodes to controller buttons
		// Check each key explicitly and only set if actually pressed (value != 0)
		// Note: SDL keyboard state uses 1 for pressed, 0 for released
		// UP: W or Up Arrow
		if keyboardState[sdl.SCANCODE_W] != 0 || keyboardState[sdl.SCANCODE_UP] != 0 {
			buttons |= 0x01 // UP
		}
		// DOWN: S or Down Arrow
		if keyboardState[sdl.SCANCODE_S] != 0 || keyboardState[sdl.SCANCODE_DOWN] != 0 {
			buttons |= 0x02 // DOWN
		}
		// LEFT: A or Left Arrow
		// DEBUG: Check if A or Left Arrow is being detected incorrectly
		aPressed := keyboardState[sdl.SCANCODE_A] != 0
		leftPressed := keyboardState[sdl.SCANCODE_LEFT] != 0
		if aPressed || leftPressed {
			buttons |= 0x04 // LEFT
		}
		// RIGHT: D or Right Arrow
		if keyboardState[sdl.SCANCODE_D] != 0 || keyboardState[sdl.SCANCODE_RIGHT] != 0 {
			buttons |= 0x08 // RIGHT
		}
		// A button: Z
		if keyboardState[sdl.SCANCODE_Z] != 0 {
			buttons |= 0x10 // A
		}
		// B button: X
		if keyboardState[sdl.SCANCODE_X] != 0 {
			buttons |= 0x20 // B
		}
		// X button: V
		if keyboardState[sdl.SCANCODE_V] != 0 {
			buttons |= 0x40 // X
		}
		// Y button: C
		if keyboardState[sdl.SCANCODE_C] != 0 {
			buttons |= 0x80 // Y
		}
		// L button: Q
		if keyboardState[sdl.SCANCODE_Q] != 0 {
			buttons |= 0x100 // L
		}
		// R button: E
		if keyboardState[sdl.SCANCODE_E] != 0 {
			buttons |= 0x200 // R
		}
		// START: Enter/Return
		if keyboardState[sdl.SCANCODE_RETURN] != 0 {
			buttons |= 0x400 // START
		}
		// Z button (used as STOP in diagnostics): Backspace
		if keyboardState[sdl.SCANCODE_BACKSPACE] != 0 {
			buttons |= 0x800 // Z
		}
	}

	// Merge Fyne key state tracking. This is the primary path for the Fyne window and
	// also acts as a fallback when SDL keyboard state does not reflect Fyne focus/input.
	buttons = ui.applyFyneKeyStates(buttons)

	// Always set input, even if 0 (this ensures input is cleared when no keys are pressed)
	// This also ensures the latched state will be 0 when the ROM next latches
	ui.emulator.SetInputButtons(buttons)
}

// updateLayout updates the main layout based on which panels are visible
// If any panels are visible, show the splitter with panels. Otherwise, hide panels by setting offset to 1.0.
func (ui *FyneUI) updateLayout() {
	// Check if any panels are visible
	anyVisible := ui.showLogViewer || ui.showRegisters || ui.showMemory || ui.showTiles

	if anyVisible {
		// At least one panel is visible - show splitter with panels (70% emulator, 30% panels)
		ui.splitContent.SetOffset(0.7)
	} else {
		// No panels visible - hide panels by setting offset to 1.0 (fully to the right)
		ui.splitContent.SetOffset(1.0)
	}
}

func (ui *FyneUI) loadROMBytes(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("ROM file is empty")
	}

	ui.emulator.Stop()
	if err := ui.emulator.LoadROM(data); err != nil {
		return err
	}
	ui.emulator.Reset()
	ui.emulator.SetInputButtons(0)
	if ui.audioDev != 0 {
		sdl.ClearQueuedAudio(ui.audioDev)
	}
	ui.emulator.Start()
	return nil
}

// createMenus creates the native Fyne menus
func createMenus(window fyne.Window, emu *emulator.Emulator, ui *FyneUI) {
	// File menu
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("Open ROM...", func() {
			openDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
				if err != nil {
					dialog.ShowError(fmt.Errorf("failed to open ROM: %w", err), window)
					return
				}
				if reader == nil {
					return
				}
				defer reader.Close()

				data, readErr := io.ReadAll(reader)
				if readErr != nil {
					dialog.ShowError(fmt.Errorf("failed to read ROM: %w", readErr), window)
					return
				}

				if loadErr := ui.loadROMBytes(data); loadErr != nil {
					dialog.ShowError(fmt.Errorf("failed to load ROM: %w", loadErr), window)
					return
				}

				romName := reader.URI().Name()
				ui.statusLabel.SetText(fmt.Sprintf("Loaded ROM: %s", romName))
			}, window)
			openDialog.SetFilter(storage.NewExtensionFileFilter([]string{".rom"}))
			openDialog.Show()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Exit", func() {
			window.Close()
		}),
	)

	// Emulation menu
	emulationMenu := fyne.NewMenu("Emulation",
		fyne.NewMenuItem("Start", func() {
			emu.Start()
		}),
		fyne.NewMenuItem("Pause", func() {
			emu.Pause()
		}),
		fyne.NewMenuItem("Resume", func() {
			emu.Resume()
		}),
		fyne.NewMenuItem("Stop", func() {
			emu.Stop()
		}),
		fyne.NewMenuItem("Reset", func() {
			emu.Reset()
		}),
		fyne.NewMenuItem("Step Frame", func() {
			if emu.Paused {
				emu.RunFrame()
			}
		}),
	)

	// View menu
	viewMenu := fyne.NewMenu("View",
		fyne.NewMenuItem("Log Viewer", func() {
			ui.showLogViewer = !ui.showLogViewer
			if ui.logViewerPanel != nil {
				if ui.showLogViewer {
					ui.logViewerPanel.Show()
				} else {
					ui.logViewerPanel.Hide()
				}
			}
			ui.updateLayout()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Registers", func() {
			ui.showRegisters = !ui.showRegisters
			if ui.registersPanel != nil {
				if ui.showRegisters {
					ui.registersPanel.Show()
				} else {
					ui.registersPanel.Hide()
				}
			}
			ui.updateLayout()
		}),
		fyne.NewMenuItem("Memory Viewer", func() {
			ui.showMemory = !ui.showMemory
			if ui.memoryPanel != nil {
				if ui.showMemory {
					ui.memoryPanel.Show()
				} else {
					ui.memoryPanel.Hide()
				}
			}
			ui.updateLayout()
		}),
		fyne.NewMenuItem("Tile Viewer", func() {
			ui.showTiles = !ui.showTiles
			if ui.tilesPanel != nil {
				if ui.showTiles {
					ui.tilesPanel.Show()
				} else {
					ui.tilesPanel.Hide()
				}
			}
			ui.updateLayout()
		}),
	)

	// Debug menu
	debugMenu := fyne.NewMenu("Debug",
		fyne.NewMenuItem("Registers", func() {
			ui.showRegisters = !ui.showRegisters
			if ui.registersPanel != nil {
				if ui.showRegisters {
					ui.registersPanel.Show()
				} else {
					ui.registersPanel.Hide()
				}
			}
			ui.updateLayout()
		}),
		fyne.NewMenuItem("Memory Viewer", func() {
			ui.showMemory = !ui.showMemory
			if ui.memoryPanel != nil {
				if ui.showMemory {
					ui.memoryPanel.Show()
				} else {
					ui.memoryPanel.Hide()
				}
			}
			ui.updateLayout()
		}),
		fyne.NewMenuItem("Tile Viewer", func() {
			ui.showTiles = !ui.showTiles
			if ui.tilesPanel != nil {
				if ui.showTiles {
					ui.tilesPanel.Show()
				} else {
					ui.tilesPanel.Hide()
				}
			}
			ui.updateLayout()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Toggle Cycle Logging", func() {
			if emu.CycleLogger != nil {
				emu.CycleLogger.Toggle()
				// Only show cycle logging status if logging is enabled
				if emu.Logger != nil && emu.Logger.IsComponentEnabled(debug.ComponentSystem) {
					enabled, current, total, max := emu.CycleLogger.GetStatus()
					if enabled {
						fmt.Printf("Cycle logging ENABLED (logged: %d/%d cycles, total: %d)\n", current, max, total)
					} else {
						fmt.Printf("Cycle logging DISABLED (logged: %d/%d cycles, total: %d)\n", current, max, total)
					}
				}
			} else {
				// Only show message if logging is enabled
				if emu.Logger != nil && emu.Logger.IsComponentEnabled(debug.ComponentSystem) {
					fmt.Println("Cycle logging not initialized (use -cyclelog flag to enable)")
				}
			}
		}),
	)

	// Create logging submenu
	if emu.Logger != nil {
		loggingSubmenu := fyne.NewMenu("Logging",
			fyne.NewMenuItem("Enable All Logging", func() {
				emu.Logger.SetComponentEnabled(debug.ComponentCPU, true)
				emu.Logger.SetComponentEnabled(debug.ComponentPPU, true)
				emu.Logger.SetComponentEnabled(debug.ComponentAPU, true)
				emu.Logger.SetComponentEnabled(debug.ComponentMemory, true)
				emu.Logger.SetComponentEnabled(debug.ComponentInput, true)
				emu.Logger.SetComponentEnabled(debug.ComponentUI, true)
				emu.Logger.SetComponentEnabled(debug.ComponentSystem, true)
			}),
			fyne.NewMenuItem("Disable All Logging", func() {
				emu.Logger.SetComponentEnabled(debug.ComponentCPU, false)
				emu.Logger.SetComponentEnabled(debug.ComponentPPU, false)
				emu.Logger.SetComponentEnabled(debug.ComponentAPU, false)
				emu.Logger.SetComponentEnabled(debug.ComponentMemory, false)
				emu.Logger.SetComponentEnabled(debug.ComponentInput, false)
				emu.Logger.SetComponentEnabled(debug.ComponentUI, false)
				emu.Logger.SetComponentEnabled(debug.ComponentSystem, false)
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Toggle CPU Logging", func() {
				enabled := emu.Logger.IsComponentEnabled(debug.ComponentCPU)
				emu.Logger.SetComponentEnabled(debug.ComponentCPU, !enabled)
			}),
			fyne.NewMenuItem("Toggle PPU Logging", func() {
				enabled := emu.Logger.IsComponentEnabled(debug.ComponentPPU)
				emu.Logger.SetComponentEnabled(debug.ComponentPPU, !enabled)
			}),
			fyne.NewMenuItem("Toggle APU Logging", func() {
				enabled := emu.Logger.IsComponentEnabled(debug.ComponentAPU)
				emu.Logger.SetComponentEnabled(debug.ComponentAPU, !enabled)
			}),
			fyne.NewMenuItem("Toggle Memory Logging", func() {
				enabled := emu.Logger.IsComponentEnabled(debug.ComponentMemory)
				emu.Logger.SetComponentEnabled(debug.ComponentMemory, !enabled)
			}),
			fyne.NewMenuItem("Toggle Input Logging", func() {
				enabled := emu.Logger.IsComponentEnabled(debug.ComponentInput)
				emu.Logger.SetComponentEnabled(debug.ComponentInput, !enabled)
			}),
			fyne.NewMenuItem("Toggle UI Logging", func() {
				enabled := emu.Logger.IsComponentEnabled(debug.ComponentUI)
				emu.Logger.SetComponentEnabled(debug.ComponentUI, !enabled)
			}),
			fyne.NewMenuItem("Toggle System Logging", func() {
				enabled := emu.Logger.IsComponentEnabled(debug.ComponentSystem)
				emu.Logger.SetComponentEnabled(debug.ComponentSystem, !enabled)
			}),
		)
		// Insert the logging submenu before the separator and cycle logging
		// Find the separator before "Toggle Cycle Logging"
		for i, item := range debugMenu.Items {
			if item.Label == "Toggle Cycle Logging" {
				// Insert logging submenu before this item
				newItems := make([]*fyne.MenuItem, 0, len(debugMenu.Items)+1)
				newItems = append(newItems, debugMenu.Items[:i-1]...) // Items before separator
				// Create menu item with submenu
				loggingMenuItem := fyne.NewMenuItem("Logging", nil)
				loggingMenuItem.ChildMenu = loggingSubmenu
				newItems = append(newItems, loggingMenuItem)
				newItems = append(newItems, debugMenu.Items[i-1:]...) // Separator and rest
				debugMenu.Items = newItems
				break
			}
		}
	}

	// Help menu
	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", func() {
			dialog.ShowInformation(
				"About Nitro-Core-DX Emulator",
				"Nitro-Core-DX Emulator is the standalone emulator UI for Nitro-Core-DX.\n\nUse File > Open ROM... to load a .rom file, then use Emulation controls to start/pause/reset.",
				window,
			)
		}),
	)

	mainMenu := fyne.NewMainMenu(
		fileMenu,
		emulationMenu,
		viewMenu,
		debugMenu,
		helpMenu,
	)

	window.SetMainMenu(mainMenu)
}

// renderEmulatorScreen renders the emulator screen and converts to scaled Fyne image
func (ui *FyneUI) renderEmulatorScreen() (image.Image, error) {
	// Get output buffer from emulator. The UI update loop calls RunFrame() and then renders
	// on the same goroutine, so reading the buffer here is safe.
	buffer := ui.emulator.GetOutputBuffer()

	if len(buffer) != 320*200 {
		return nil, fmt.Errorf("buffer size mismatch: expected %d, got %d", 320*200, len(buffer))
	}

	// Reuse double-buffered RGBA images to avoid per-frame allocations.
	img := ui.frameImages[ui.frameImageIdx]
	ui.frameImageIdx ^= 1

	// Scale pixels manually for perfect integer scaling (direct Pix writes are much
	// faster than img.Set in the hot path).
	pix := img.Pix
	stride := img.Stride
	scale := ui.scale
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			idx := y*320 + x
			colorValue := buffer[idx]

			// Convert RGB888 to RGBA
			r := uint8((colorValue >> 16) & 0xFF)
			g := uint8((colorValue >> 8) & 0xFF)
			b := uint8(colorValue & 0xFF)

			// Scale pixel
			baseX := x * scale
			baseY := y * scale
			for sy := 0; sy < scale; sy++ {
				row := (baseY + sy) * stride
				for sx := 0; sx < scale; sx++ {
					off := row + (baseX+sx)*4
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

// Run runs the Fyne UI main loop
func (ui *FyneUI) Run() error {
	defer ui.Cleanup()

	// UI updateLoop provides the pacing target (60Hz ticker), so disable emulator-internal
	// frame sleeping here to avoid double-throttling/jitter.
	ui.emulator.SetFrameLimit(false)

	// Start emulator
	ui.emulator.Start()
	ui.running = true

	// Initialize input to 0 (no buttons pressed)
	ui.emulator.SetInputButtons(0)

	// Update loop
	go ui.updateLoop()

	// Show and run (blocks until window is closed)
	ui.window.ShowAndRun()
	ui.running = false

	return nil
}

// updateLoop updates the UI at 60 FPS
func (ui *FyneUI) updateLoop() {
	// Run a higher-rate UI tick and advance emulation using a fixed 60Hz timestep accumulator.
	// This keeps gameplay speed stable even when UI rendering occasionally stutters.
	const emuHz = 60
	const uiTickHz = 120
	const maxCatchUpFrames = 4
	frameStep := time.Second / emuHz

	ticker := time.NewTicker(time.Second / uiTickHz)
	defer ticker.Stop()
	uiTickCount := 0
	lastTick := time.Now()
	accumulator := time.Duration(0)

	for ui.running {
		<-ticker.C
		uiTickCount++
		now := time.Now()
		delta := now.Sub(lastTick)
		lastTick = now
		// Clamp long stalls (window drag/breakpoint/suspend) to avoid huge catch-up bursts.
		if delta > 250*time.Millisecond {
			delta = 250 * time.Millisecond
		}

		// Pump SDL events to update keyboard state
		sdl.PumpEvents()

		// Update input from SDL keyboard state BEFORE running the frame
		// This ensures the ROM reads the correct input state
		ui.updateInputFromKeys()

		framesStepped := 0
		if ui.emulator.Paused {
			// Do not accumulate emulation debt while paused.
			accumulator = 0
		} else {
			accumulator += delta
			maxAccum := frameStep * maxCatchUpFrames
			if accumulator > maxAccum {
				accumulator = maxAccum
			}

			// Fixed-timestep emulation: advance 1 frame per 16.67ms of accumulated time.
			for accumulator >= frameStep && framesStepped < maxCatchUpFrames {
				if err := ui.emulator.RunFrame(); err != nil {
					if ui.emulator.Logger != nil {
						ui.emulator.Logger.LogUI(debug.LogLevelError, fmt.Sprintf("Emulation error: %v", err), nil)
					}
					break
				}
				ui.queueFrameAudio()
				accumulator -= frameStep
				framesStepped++
			}
		}

		// Render emulator screen
		// Note: RunFrame() completes a full PPU frame (127,820 cycles), so buffer is ready
		// FrameComplete flag ensures we don't read mid-frame, but RunFrame() guarantees completion
		var img image.Image
		var imgErr error
		if framesStepped > 0 || (ui.emulator.Paused && uiTickCount%8 == 0) {
			img, imgErr = ui.renderEmulatorScreen()
		}

		// Update UI on main thread. Throttle status/panel refreshes slightly to reduce
		// Fyne/UI work in the hot path while keeping the emulator display at full rate.
		fps := ui.emulator.GetFPS()
		cycles := ui.emulator.GetCPUCyclesPerFrame()
		frameCount := ui.emulator.FrameCount
		refreshAuxPanels := (uiTickCount%4 == 0) // ~15 Hz at 60 FPS target
		fyne.Do(func() {
			if img != nil && imgErr == nil {
				ui.emulatorImage.Image = img
				ui.emulatorImage.Refresh()
			}
			ui.statusLabel.SetText(fmt.Sprintf("FPS: %.1f | CPU: %d cycles/frame | Frame: %d", fps, cycles, frameCount))
			if refreshAuxPanels {
				// Update register viewer if visible
				if ui.showRegisters && ui.updateRegisters != nil {
					ui.updateRegisters()
				}
				// Update memory viewer if visible
				if ui.showMemory && ui.updateMemory != nil {
					ui.updateMemory()
				}
				// Update tile viewer if visible
				if ui.showTiles && ui.updateTiles != nil {
					ui.updateTiles()
				}
				// Update log viewer if visible
				if ui.showLogViewer && ui.updateLogs != nil {
					ui.updateLogs()
				}
			}
		})
	}
}

func (ui *FyneUI) queueFrameAudio() {
	if ui.audioDev == 0 || ui.emulator == nil {
		return
	}

	// Prevent runaway queue growth if rendering stalls. Keep roughly <= 4 frames queued.
	if sdl.GetQueuedAudioSize(ui.audioDev) > uint32(len(ui.audioFrame))*4 {
		return
	}

	samples := ui.emulator.AudioSampleBuffer // 735 mono int16 samples for last frame
	if len(samples) == 0 {
		return
	}

	// Convert mono int16 fixed-point samples to interleaved stereo float32 (little-endian).
	// Duplicating channels keeps behavior simple until a stereo mixer path is introduced.
	j := 0
	for _, s := range samples {
		f := apu.ConvertFixedToFloat(s)
		bits := math.Float32bits(f)
		binary.LittleEndian.PutUint32(ui.audioFrame[j:j+4], bits)
		binary.LittleEndian.PutUint32(ui.audioFrame[j+4:j+8], bits)
		j += 8
	}

	_ = sdl.QueueAudio(ui.audioDev, ui.audioFrame)
}

// Cleanup cleans up resources
func (ui *FyneUI) Cleanup() {
	// Shutdown logger to clean up goroutine (prevents goroutine leak)
	if ui.emulator != nil && ui.emulator.Logger != nil {
		ui.emulator.Logger.Shutdown()
	}

	if ui.audioDev != 0 {
		sdl.CloseAudioDevice(ui.audioDev)
	}
	if ui.sdlTexture != nil {
		ui.sdlTexture.Destroy()
	}
	if ui.sdlRenderer != nil {
		ui.sdlRenderer.Destroy()
	}
	sdl.Quit()
}
