package ui

import (
	"fmt"
	"unsafe"

	"nitro-core-dx/internal/cpu"
	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/emulator"
	"nitro-core-dx/internal/ui/panels"

	"github.com/veandco/go-sdl2/sdl"
)

// UI represents the user interface
type UI struct {
	window          *sdl.Window
	renderer        *sdl.Renderer
	texture         *sdl.Texture
	emulator        *emulator.Emulator
	running         bool
	scale           int
	fullscreen      bool
	audioDev        sdl.AudioDeviceID
	debugFrameCount int // For debug logging

	// UI panels
	showLogViewer      bool
	showLogControls    bool
	showRegisterViewer bool
	showMemoryViewer   bool
	showTileViewer     bool

	// Panel instances (for interaction)
	logControlsPanel *panels.LogControls

	// Panel positions and sizes (for future docking system)
	logViewerRect      *sdl.Rect
	logControlsRect    *sdl.Rect
	registerViewerRect *sdl.Rect
	memoryViewerRect   *sdl.Rect
	tileViewerRect     *sdl.Rect

	// Menu bar and toolbar for click handling
	menuBar  *MenuBar
	toolbar  *Toolbar
	toolbarY int32

	// Modern text renderer (for UI elements)
	textRenderer TextRenderer
}

// NewUI creates a new UI instance (legacy SDL2-only UI)
// For new development, use NewFyneUI instead
func NewUI(emu *emulator.Emulator, scale int) (*UI, error) {
	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO); err != nil {
		return nil, fmt.Errorf("failed to initialize SDL: %w", err)
	}

	// Set render scale quality hint to nearest-neighbor for pixel-perfect scaling
	// This must be set before creating the renderer
	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "0")

	// Window size: menu bar + toolbar + emulator output + status bar
	menuBarHeight := int32(20 * scale)
	toolbarHeight := int32(30 * scale)
	emulatorHeight := int32(200 * scale)
	statusBarHeight := int32(20 * scale)

	width := int32(320 * scale)
	height := menuBarHeight + toolbarHeight + emulatorHeight + statusBarHeight

	window, err := sdl.CreateWindow(
		"Nitro-Core-DX Emulator",
		sdl.WINDOWPOS_CENTERED,
		sdl.WINDOWPOS_CENTERED,
		width,
		height,
		sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE, // Resizable for panels
	)
	if err != nil {
		sdl.Quit()
		return nil, fmt.Errorf("failed to create window: %w", err)
	}

	// Create renderer with software fallback if hardware doesn't support nearest-neighbor
	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		window.Destroy()
		sdl.Quit()
		return nil, fmt.Errorf("failed to create renderer: %w", err)
	}

	// Create texture - will be resized to scaled size in render()
	// Start with 320x200, will be recreated at scaled size
	texture, err := renderer.CreateTexture(
		sdl.PIXELFORMAT_RGB888,
		sdl.TEXTUREACCESS_STREAMING,
		int32(320*scale),
		int32(200*scale),
	)
	if err != nil {
		renderer.Destroy()
		window.Destroy()
		sdl.Quit()
		return nil, fmt.Errorf("failed to create texture: %w", err)
	}

	// Open audio device
	audioSpec := sdl.AudioSpec{
		Freq:     44100,
		Format:   sdl.AUDIO_F32,
		Channels: 2,   // Stereo
		Samples:  735, // Samples per frame (44100 Hz / 60 FPS)
	}
	audioDev, err := sdl.OpenAudioDevice("", false, &audioSpec, nil, 0)
	if err != nil {
		// Audio is optional, continue without it
		// Log using emulator's logger if available
		if emu.Logger != nil {
			emu.Logger.LogUI(debug.LogLevelWarning, fmt.Sprintf("Failed to open audio device: %v", err), nil)
		} else {
			fmt.Printf("Warning: Failed to open audio device: %v\n", err)
		}
		audioDev = 0
	} else {
		sdl.PauseAudioDevice(audioDev, false) // Start playback
	}

	// Create modern text renderer for UI elements
	// Try SDL_ttf first, fall back to simple bitmap font if not available
	textRenderer, err := NewTextRenderer(scale)
	if err != nil {
		// Fall back to simple bitmap font renderer
		if emu.Logger != nil {
			emu.Logger.LogUI(debug.LogLevelInfo, fmt.Sprintf("SDL_ttf not available, using simple bitmap font: %v", err), nil)
		}
		textRenderer = NewSimpleTextRenderer(scale)
	}

	ui := &UI{
		window:       window,
		renderer:     renderer,
		texture:      texture,
		emulator:     emu,
		running:      true,
		scale:        scale,
		audioDev:     audioDev,
		textRenderer: textRenderer,
		// Panels hidden by default
		showLogViewer:      false,
		showRegisterViewer: false,
		showMemoryViewer:   false,
		showTileViewer:     false,
	}

	return ui, nil
}

// Run runs the UI main loop
func (u *UI) Run() error {
	defer u.Cleanup()

	// Start emulator
	u.emulator.Start()

	// Main event loop
	for u.running {
		// Handle events
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			if err := u.handleEvent(event); err != nil {
				return err
			}
		}

		// Update input
		u.updateInput()

		// Run emulator frame
		if err := u.emulator.RunFrame(); err != nil {
			return fmt.Errorf("emulation error: %w", err)
		}

		// Queue audio samples
		if u.audioDev != 0 {
			audioSamples := u.emulator.GetAudioSamples()
			if len(audioSamples) > 0 {
				// Check queue size to prevent backing up (limit to ~4 frames worth for smoother audio)
				queuedBytes := sdl.GetQueuedAudioSize(u.audioDev)
				maxQueuedBytes := uint32(len(audioSamples) * 2 * 4 * 4) // 4 frames worth

				// Only queue if queue isn't too full (prevent audio stuttering)
				if queuedBytes < maxQueuedBytes {
					// Convert float32 samples to bytes (stereo: interleaved L/R)
					// AUDIO_F32 format: native float32, little-endian
					audioBytes := make([]byte, len(audioSamples)*2*4) // 2 channels * 4 bytes per float32
					for i, sample := range audioSamples {
						// Convert float32 to bytes (little-endian, native format)
						sampleBytes := (*[4]byte)(unsafe.Pointer(&sample))
						// Left channel
						audioBytes[i*8] = sampleBytes[0]
						audioBytes[i*8+1] = sampleBytes[1]
						audioBytes[i*8+2] = sampleBytes[2]
						audioBytes[i*8+3] = sampleBytes[3]
						// Right channel (same sample)
						audioBytes[i*8+4] = sampleBytes[0]
						audioBytes[i*8+5] = sampleBytes[1]
						audioBytes[i*8+6] = sampleBytes[2]
						audioBytes[i*8+7] = sampleBytes[3]
					}
					err := sdl.QueueAudio(u.audioDev, audioBytes)
					if err != nil {
						// Only log if logging is enabled (avoid fmt.Printf overhead)
						if u.emulator.Logger != nil && u.emulator.Logger.IsComponentEnabled(debug.ComponentUI) {
							u.emulator.Logger.LogUIf(debug.LogLevelError, "Failed to queue audio: %v", err)
						}
					}
				}
				// If queue is full, just skip queuing this frame (audio will continue from buffer)
			}
		}

		// Render frame (using manual scaling for perfect pixel rendering)
		if err := u.renderFixed(); err != nil {
			return fmt.Errorf("render error: %w", err)
		}

		// Render UI elements (menu bar, toolbar, status bar, panels)
		u.renderUI()

		// Present the frame
		u.renderer.Present()

		// Small delay to prevent 100% CPU usage
		sdl.Delay(1)
	}

	return nil
}

// handleEvent handles SDL events
func (u *UI) handleEvent(event sdl.Event) error {
	switch e := event.(type) {
	case *sdl.QuitEvent:
		u.running = false
		return nil

	case *sdl.KeyboardEvent:
		if e.Type == sdl.KEYDOWN {
			return u.handleKeyDown(e.Keysym.Sym)
		} else if e.Type == sdl.KEYUP {
			return u.handleKeyUp(e.Keysym.Sym)
		}
	case *sdl.MouseButtonEvent:
		if e.Type == sdl.MOUSEBUTTONDOWN {
			return u.handleMouseClick(e.X, e.Y, e.Button)
		}
	}

	return nil
}

// handleKeyDown handles key press events
func (u *UI) handleKeyDown(key sdl.Keycode) error {
	switch key {
	case sdl.K_ESCAPE:
		u.running = false
	case sdl.K_SPACE:
		if u.emulator.Paused {
			u.emulator.Resume()
		} else {
			u.emulator.Pause()
		}
	case sdl.K_r:
		if sdl.GetModState()&sdl.KMOD_CTRL != 0 {
			u.emulator.Reset()
		}
	case sdl.K_f:
		if sdl.GetModState()&sdl.KMOD_ALT != 0 {
			u.toggleFullscreen()
		}
	case sdl.K_l:
		// Toggle log viewer (Ctrl+L)
		if sdl.GetModState()&sdl.KMOD_CTRL != 0 {
			u.showLogViewer = !u.showLogViewer
			if u.emulator.Logger != nil {
				u.emulator.Logger.LogUIf(debug.LogLevelInfo, "Log viewer %s",
					map[bool]string{true: "shown", false: "hidden"}[u.showLogViewer])
			}
		}
	case sdl.K_k:
		// Toggle log controls (Ctrl+K)
		if sdl.GetModState()&sdl.KMOD_CTRL != 0 {
			u.showLogControls = !u.showLogControls
			if u.emulator.Logger != nil {
				u.emulator.Logger.LogUIf(debug.LogLevelInfo, "Log controls %s",
					map[bool]string{true: "shown", false: "hidden"}[u.showLogControls])
			}
		}
	// CPU log level shortcuts (Ctrl+1-7)
	case sdl.K_1:
		if sdl.GetModState()&sdl.KMOD_CTRL != 0 && u.logControlsPanel != nil {
			u.logControlsPanel.SetCPULogLevel(cpu.CPULogNone)
		}
	case sdl.K_2:
		if sdl.GetModState()&sdl.KMOD_CTRL != 0 && u.logControlsPanel != nil {
			u.logControlsPanel.SetCPULogLevel(cpu.CPULogErrors)
		}
	case sdl.K_3:
		if sdl.GetModState()&sdl.KMOD_CTRL != 0 && u.logControlsPanel != nil {
			u.logControlsPanel.SetCPULogLevel(cpu.CPULogBranches)
		}
	case sdl.K_4:
		if sdl.GetModState()&sdl.KMOD_CTRL != 0 && u.logControlsPanel != nil {
			u.logControlsPanel.SetCPULogLevel(cpu.CPULogMemory)
		}
	case sdl.K_5:
		if sdl.GetModState()&sdl.KMOD_CTRL != 0 && u.logControlsPanel != nil {
			u.logControlsPanel.SetCPULogLevel(cpu.CPULogRegisters)
		}
	case sdl.K_6:
		if sdl.GetModState()&sdl.KMOD_CTRL != 0 && u.logControlsPanel != nil {
			u.logControlsPanel.SetCPULogLevel(cpu.CPULogInstructions)
		}
	case sdl.K_7:
		if sdl.GetModState()&sdl.KMOD_CTRL != 0 && u.logControlsPanel != nil {
			u.logControlsPanel.SetCPULogLevel(cpu.CPULogTrace)
		}
	}

	return nil
}

// handleMouseClick handles mouse click events
func (u *UI) handleMouseClick(x, y int32, button uint8) error {
	if button != sdl.BUTTON_LEFT {
		return nil // Only handle left clicks
	}

	menuBarHeight := int32(20 * u.scale)

	// Check if click is in menu bar
	if y < menuBarHeight && u.menuBar != nil {
		item, clicked := u.menuBar.HandleClick(x, y)
		if clicked {
			u.handleMenuClick(item)
			return nil
		}
	}

	// Check if click is in toolbar
	if y >= menuBarHeight && y < menuBarHeight+u.toolbarY+int32(30*u.scale) && u.toolbar != nil {
		btn, clicked := u.toolbar.HandleClick(x, y-u.toolbarY)
		if clicked {
			u.handleToolbarClick(btn)
			return nil
		}
	}

	// Check if click is in log controls panel
	if u.showLogControls && u.logControlsPanel != nil && u.logControlsRect != nil {
		if x >= u.logControlsRect.X && x < u.logControlsRect.X+u.logControlsRect.W &&
			y >= u.logControlsRect.Y && y < u.logControlsRect.Y+u.logControlsRect.H {
			// Click is in log controls panel
			u.logControlsPanel.HandleClick(x-u.logControlsRect.X, y-u.logControlsRect.Y)
		}
	}

	return nil
}

// handleMenuClick handles menu bar item clicks
func (u *UI) handleMenuClick(item string) {
	switch item {
	case "File":
		// TODO: File menu (open ROM, exit, etc.)
		if u.emulator.Logger != nil {
			u.emulator.Logger.LogUI(debug.LogLevelInfo, "File menu clicked", nil)
		}
	case "Emulation":
		// TODO: Emulation menu (start, pause, reset, etc.)
		if u.emulator.Logger != nil {
			u.emulator.Logger.LogUI(debug.LogLevelInfo, "Emulation menu clicked", nil)
		}
	case "View":
		// Toggle log viewer
		u.showLogViewer = !u.showLogViewer
		if u.emulator.Logger != nil {
			u.emulator.Logger.LogUI(debug.LogLevelInfo, fmt.Sprintf("Log viewer %s", map[bool]string{true: "shown", false: "hidden"}[u.showLogViewer]), nil)
		}
	case "Debug":
		// Toggle log controls
		u.showLogControls = !u.showLogControls
		if u.emulator.Logger != nil {
			u.emulator.Logger.LogUI(debug.LogLevelInfo, fmt.Sprintf("Log controls %s", map[bool]string{true: "shown", false: "hidden"}[u.showLogControls]), nil)
		}
	case "Help":
		// TODO: Help menu
		if u.emulator.Logger != nil {
			u.emulator.Logger.LogUI(debug.LogLevelInfo, "Help menu clicked", nil)
		}
	}
}

// handleToolbarClick handles toolbar button clicks
func (u *UI) handleToolbarClick(btn string) {
	switch btn {
	case "Start":
		if !u.emulator.Running {
			u.emulator.Start()
		}
	case "Pause":
		if u.emulator.Running && !u.emulator.Paused {
			u.emulator.Pause()
		}
	case "Resume":
		if u.emulator.Running && u.emulator.Paused {
			u.emulator.Resume()
		}
	case "Stop":
		if u.emulator.Running {
			u.emulator.Stop()
		}
	case "Reset":
		u.emulator.Reset()
	case "Step":
		if u.emulator.Paused {
			// Step one instruction
			if err := u.emulator.RunFrame(); err != nil {
				if u.emulator.Logger != nil {
					u.emulator.Logger.LogUI(debug.LogLevelError, fmt.Sprintf("Step error: %v", err), nil)
				}
			}
		}
	}
}

// handleKeyUp handles key release events
func (u *UI) handleKeyUp(key sdl.Keycode) error {
	// Handle key releases if needed
	return nil
}

// updateInput updates the emulator input state
func (u *UI) updateInput() {
	keys := sdl.GetKeyboardState()

	// Map keyboard to controller
	buttons := uint16(0)

	// Arrow keys
	if keys[sdl.SCANCODE_UP] != 0 {
		buttons |= 0x01 // UP
	}
	if keys[sdl.SCANCODE_DOWN] != 0 {
		buttons |= 0x02 // DOWN
	}
	if keys[sdl.SCANCODE_LEFT] != 0 {
		buttons |= 0x04 // LEFT
	}
	if keys[sdl.SCANCODE_RIGHT] != 0 {
		buttons |= 0x08 // RIGHT
	}

	// Action buttons (WASD + ZX)
	if keys[sdl.SCANCODE_W] != 0 || keys[sdl.SCANCODE_Z] != 0 {
		buttons |= 0x10 // A
	}
	if keys[sdl.SCANCODE_X] != 0 {
		buttons |= 0x20 // B
	}
	if keys[sdl.SCANCODE_A] != 0 {
		buttons |= 0x40 // X
	}
	if keys[sdl.SCANCODE_S] != 0 {
		buttons |= 0x80 // Y
	}

	// Shoulder buttons
	if keys[sdl.SCANCODE_Q] != 0 {
		buttons |= 0x100 // L
	}
	if keys[sdl.SCANCODE_E] != 0 {
		buttons |= 0x200 // R
	}

	// Start/Select
	if keys[sdl.SCANCODE_RETURN] != 0 {
		buttons |= 0x400 // START
	}
	if keys[sdl.SCANCODE_RSHIFT] != 0 || keys[sdl.SCANCODE_LSHIFT] != 0 {
		buttons |= 0x800 // SELECT
	}

	// Update emulator input (need to access through emulator)
	// The Input field is not exported, so we need a method
	u.emulator.SetInputButtons(buttons)
}

// render renders a frame
func (u *UI) render() error {
	// Get output buffer from emulator
	buffer := u.emulator.GetOutputBuffer()

	// Convert RGB888 to bytes (RGB888 format: R, G, B order)
	// Ensure we have exactly 320*200 pixels
	if len(buffer) != 320*200 {
		return fmt.Errorf("buffer size mismatch: expected %d, got %d", 320*200, len(buffer))
	}

	pixels := make([]byte, 320*200*3)
	for i := 0; i < 320*200; i++ {
		color := buffer[i]
		// RGB888 format: R, G, B order
		pixels[i*3] = byte((color >> 16) & 0xFF)  // R
		pixels[i*3+1] = byte((color >> 8) & 0xFF) // G
		pixels[i*3+2] = byte(color & 0xFF)        // B
	}

	// Update texture with proper pitch (bytes per row)
	// Pitch must be exactly 320*3 = 960 bytes per row
	pitch := 320 * 3
	if err := u.texture.Update(nil, unsafe.Pointer(&pixels[0]), pitch); err != nil {
		return fmt.Errorf("failed to update texture: %w", err)
	}

	// Clear renderer
	u.renderer.Clear()

	// Copy texture to renderer with exact integer scaling
	// Get renderer output size
	outputW, outputH, _ := u.renderer.GetOutputSize()

	// Calculate expected sizes
	infoBarHeight := int32(8 * u.scale)
	expectedW := int32(320 * u.scale)
	expectedH := int32(200*u.scale) + infoBarHeight

	// Verify window size matches expected (should be exact)
	if int32(outputW) != expectedW || int32(outputH) != expectedH {
		u.window.SetSize(expectedW, expectedH)
		outputW, outputH, _ = u.renderer.GetOutputSize()
	}

	// Emulator output: exact 320x200 scaled, at top-left
	dstW := int32(320 * u.scale)
	dstH := int32(200 * u.scale)

	// Ensure exact integer scaling - don't allow any clamping that could cause issues
	if dstW != int32(outputW) || dstH != int32(outputH)-infoBarHeight {
		// Window size mismatch - this shouldn't happen but handle it
		if dstW > int32(outputW) {
			dstW = int32(outputW)
		}
		if dstH > int32(outputH)-infoBarHeight {
			dstH = int32(outputH) - infoBarHeight
		}
	}

	// Source: full texture (320x200) - exact source rectangle
	srcRect := &sdl.Rect{X: 0, Y: 0, W: 320, H: 200}
	// Destination: exact scaled size at top-left (no centering, no offset)
	dstRect := &sdl.Rect{X: 0, Y: 0, W: dstW, H: dstH}

	// Reset renderer scale to 1:1 (critical for pixel-perfect rendering)
	u.renderer.SetScale(1.0, 1.0)

	// Copy texture with exact source and destination rectangles
	// The hint set earlier should ensure nearest-neighbor scaling
	if err := u.renderer.Copy(u.texture, srcRect, dstRect); err != nil {
		return fmt.Errorf("failed to copy texture: %w", err)
	}

	// Render debug overlay (FPS, CPU cycles) on top
	u.renderDebugOverlay()

	// Present
	u.renderer.Present()

	return nil
}

// renderDebugOverlay renders the info bar below the emulator output
func (u *UI) renderDebugOverlay() {
	// Get renderer output size
	outputW, _, _ := u.renderer.GetOutputSize()

	// Info bar is below the emulator output
	infoBarHeight := int32(8 * u.scale)
	emulatorHeight := int32(200 * u.scale)

	// Draw black info bar background
	infoBarRect := &sdl.Rect{
		X: 0,
		Y: emulatorHeight,
		W: int32(outputW),
		H: infoBarHeight,
	}
	u.renderer.SetDrawColor(0, 0, 0, 255) // Black
	u.renderer.FillRect(infoBarRect)

	// Draw text in info bar with modern font
	if u.textRenderer != nil {
		fps := u.emulator.GetFPS()
		cycles := u.emulator.GetCPUCyclesPerFrame()

		// Format FPS string
		fpsStr := fmt.Sprintf("FPS: %.1f", fps)
		if fps < 1.0 {
			fpsStr = fmt.Sprintf("FPS: %.2f", fps)
		}

		// Format cycles string
		cyclesStr := fmt.Sprintf("CPU: %d cycles/frame", cycles)

		// White color for text
		white := sdl.Color{R: 255, G: 255, B: 255, A: 255}

		// Text position in info bar (centered vertically in the bar)
		textY := emulatorHeight + (infoBarHeight-int32(12*u.scale))/2
		textX1 := int32(4 * u.scale)   // Left side
		textX2 := int32(120 * u.scale) // Right side

		u.textRenderer.DrawText(u.renderer, fpsStr, textX1, textY, white)
		u.textRenderer.DrawText(u.renderer, cyclesStr, textX2, textY, white)
	}
}

// toggleFullscreen toggles fullscreen mode
func (u *UI) toggleFullscreen() {
	if u.fullscreen {
		u.window.SetFullscreen(0)
		u.fullscreen = false
	} else {
		u.window.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
		u.fullscreen = true
	}
}

// Cleanup cleans up SDL resources
func (u *UI) Cleanup() {
	// Shutdown logger to clean up goroutine (prevents goroutine leak)
	if u.emulator != nil && u.emulator.Logger != nil {
		u.emulator.Logger.Shutdown()
	}

	if u.textRenderer != nil {
		u.textRenderer.Close()
	}
	if u.audioDev != 0 {
		sdl.CloseAudioDevice(u.audioDev)
	}
	if u.texture != nil {
		u.texture.Destroy()
	}
	if u.renderer != nil {
		u.renderer.Destroy()
	}
	if u.window != nil {
		u.window.Destroy()
	}
	sdl.Quit()
}

// SetScale sets the display scale
func (u *UI) SetScale(scale int) {
	u.scale = scale
	// Window size: emulator output + info bar
	infoBarHeight := int32(8 * scale)
	u.window.SetSize(int32(320*scale), int32(200*scale)+infoBarHeight)
}
