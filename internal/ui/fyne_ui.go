package ui

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/emulator"
	"nitro-core-dx/internal/ui/panels"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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

	// Fyne widgets
	emulatorImage *canvas.Image
	statusLabel   *widget.Label

	// Debug panels
	showLogViewer   bool
	showRegisters   bool
	showMemory      bool
	showTiles       bool

	// Panel containers
	logViewerPanel   *fyne.Container
	registersPanel   *fyne.Container
	memoryPanel      *fyne.Container
	tilesPanel       *fyne.Container
	
	// Layout containers (for dynamic updates)
	splitContent      *container.Split
	rightPanels       *fyne.Container
	mainContent       *fyne.Container

	// Panel update functions
	updateRegisters func()
	updateMemory    func()
	updateTiles     func()
	updateLogs      func()
}

// NewFyneUI creates a new Fyne-based UI
func NewFyneUI(emu *emulator.Emulator, scale int) (*FyneUI, error) {
	// Initialize SDL2 for audio and rendering
	if err := sdl.Init(sdl.INIT_AUDIO); err != nil {
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

	// Create emulator image (will be updated with SDL2 rendering)
	emulatorImage := canvas.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 320*scale, 200*scale)))
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
		emulatorImage:   emulatorImage,
		statusLabel:     statusLabel,
		registersPanel:  registersPanel,
		memoryPanel:     memoryPanel,
		tilesPanel:      tilesPanel,
		logViewerPanel:  logViewerPanel,
		updateRegisters: updateRegistersFunc,
		updateMemory:    updateMemoryFunc,
		updateTiles:     updateTilesFunc,
		updateLogs:      updateLogsFunc,
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
		nil,           // Top (no toolbar - use menu instead)
		statusLabel,   // Bottom
		nil,           // Left
		nil,           // Right
		splitContent,  // Center (splitter with emulator and panels)
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

	return ui, nil
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


// createMenus creates the native Fyne menus
func createMenus(window fyne.Window, emu *emulator.Emulator, ui *FyneUI) {
	// File menu
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("Open ROM...", func() {
			// TODO: File dialog
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

	// Help menu
	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", func() {
			// TODO: About dialog
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
	// Get output buffer from emulator (make a copy to avoid race conditions)
	buffer := ui.emulator.GetOutputBuffer()

	if len(buffer) != 320*200 {
		return nil, fmt.Errorf("buffer size mismatch: expected %d, got %d", 320*200, len(buffer))
	}

	// Make a copy of the buffer to avoid race conditions with PPU rendering
	bufferCopy := make([]uint32, len(buffer))
	copy(bufferCopy, buffer)

	// Create scaled RGBA image
	scaledW := 320 * ui.scale
	scaledH := 200 * ui.scale
	img := image.NewRGBA(image.Rect(0, 0, scaledW, scaledH))

	// Scale pixels manually for perfect integer scaling
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			idx := y*320 + x
			colorValue := bufferCopy[idx]

			// Convert RGB888 to RGBA
			r := uint8((colorValue >> 16) & 0xFF)
			g := uint8((colorValue >> 8) & 0xFF)
			b := uint8(colorValue & 0xFF)
			c := color.RGBA{R: r, G: g, B: b, A: 255}

			// Scale pixel
			for sy := 0; sy < ui.scale; sy++ {
				for sx := 0; sx < ui.scale; sx++ {
					img.Set(x*ui.scale+sx, y*ui.scale+sy, c)
				}
			}
		}
	}

	return img, nil
}

// Run runs the Fyne UI main loop
func (ui *FyneUI) Run() error {
	defer ui.Cleanup()

	// Start emulator
	ui.emulator.Start()
	ui.running = true

	// Update loop
	go ui.updateLoop()

	// Show and run (blocks until window is closed)
	ui.window.ShowAndRun()
	ui.running = false

	return nil
}

// updateLoop updates the UI at 60 FPS
func (ui *FyneUI) updateLoop() {
	ticker := time.NewTicker(time.Second / 60) // 60 FPS
	defer ticker.Stop()

	for ui.running {
		<-ticker.C

		// Update emulator
		if !ui.emulator.Paused {
			if err := ui.emulator.RunFrame(); err != nil {
				if ui.emulator.Logger != nil {
					ui.emulator.Logger.LogUI(debug.LogLevelError, fmt.Sprintf("Emulation error: %v", err), nil)
				}
			}
		}

		// Render emulator screen
		// Note: RunFrame() completes a full PPU frame (127,820 cycles), so buffer is ready
		// FrameComplete flag ensures we don't read mid-frame, but RunFrame() guarantees completion
		img, err := ui.renderEmulatorScreen()
		if err == nil {
			// Update image - must be done on main thread
			fyne.Do(func() {
				ui.emulatorImage.Image = img
				ui.emulatorImage.Refresh()
			})
		}

		// Update status - must be done on main thread
		fps := ui.emulator.GetFPS()
		cycles := ui.emulator.GetCPUCyclesPerFrame()
		frameCount := ui.emulator.FrameCount
		fyne.Do(func() {
			ui.statusLabel.SetText(fmt.Sprintf("FPS: %.1f | CPU: %d cycles/frame | Frame: %d", fps, cycles, frameCount))
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
		})
	}
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
