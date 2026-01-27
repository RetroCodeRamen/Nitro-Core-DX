package ui

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/veandco/go-sdl2/sdl"
	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/emulator"
)

// FyneUI represents the Fyne-based UI with SDL2 for emulator rendering
type FyneUI struct {
	app            fyne.App
	window         fyne.Window
	emulator       *emulator.Emulator
	scale          int
	running         bool
	paused          bool
	
	// SDL2 for emulator rendering
	sdlRenderer    *sdl.Renderer
	sdlTexture      *sdl.Texture
	audioDev        sdl.AudioDeviceID
	
	// Fyne widgets
	emulatorImage  *canvas.Image
	statusLabel    *widget.Label
	toolbar        *fyne.Container
	
	// Menu state
	showLogViewer   bool
	showLogControls bool
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
	
	// Create toolbar
	toolbar := createToolbar(emu)
	
	// Create emulator image (will be updated with SDL2 rendering)
	emulatorImage := canvas.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 320*scale, 200*scale)))
	emulatorImage.FillMode = canvas.ImageFillContain
	
	// Create main content
	content := container.NewBorder(
		toolbar,           // Top
		statusLabel,       // Bottom
		nil,               // Left
		nil,               // Right
		emulatorImage,     // Center
	)
	
	window.SetContent(content)
	window.Resize(fyne.NewSize(float32(320*scale), float32(200*scale)+100))
	
	// Create menus
	createMenus(window, emu)
	
	ui := &FyneUI{
		app:            fyneApp,
		window:         window,
		emulator:       emu,
		scale:          scale,
		running:         false,
		paused:          false,
		audioDev:        audioDev,
		emulatorImage: emulatorImage,
		statusLabel:    statusLabel,
		toolbar:        toolbar,
	}
	
	return ui, nil
}

// createToolbar creates the toolbar with emulator controls
func createToolbar(emu *emulator.Emulator) *fyne.Container {
	startBtn := widget.NewButton("Start", func() {
		emu.Start()
	})
	pauseBtn := widget.NewButton("Pause", func() {
		emu.Pause()
	})
	resumeBtn := widget.NewButton("Resume", func() {
		emu.Resume()
	})
	stopBtn := widget.NewButton("Stop", func() {
		emu.Stop()
	})
	resetBtn := widget.NewButton("Reset", func() {
		emu.Reset()
	})
	stepBtn := widget.NewButton("Step", func() {
		if emu.Paused {
			emu.RunFrame()
		}
	})
	
	return container.NewHBox(
		startBtn,
		pauseBtn,
		resumeBtn,
		stopBtn,
		resetBtn,
		stepBtn,
	)
}

// createMenus creates the native Fyne menus
func createMenus(window fyne.Window, emu *emulator.Emulator) {
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
			// TODO: Toggle log viewer
		}),
		fyne.NewMenuItem("Log Controls", func() {
			// TODO: Toggle log controls
		}),
	)
	
	// Debug menu
	debugMenu := fyne.NewMenu("Debug",
		fyne.NewMenuItem("Log Controls", func() {
			// TODO: Show log controls
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
		// Note: RunFrame() completes a full PPU frame (79,200 cycles), so buffer is ready
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

