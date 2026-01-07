package ui

import (
	"fmt"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
	"nitro-core-dx/internal/emulator"
)

// UI represents the user interface
type UI struct {
	window     *sdl.Window
	renderer   *sdl.Renderer
	texture    *sdl.Texture
	emulator   *emulator.Emulator
	running    bool
	scale      int
	fullscreen bool
	audioDev   sdl.AudioDeviceID
	debugFrameCount int // For debug logging
}

// NewUI creates a new UI instance
func NewUI(emu *emulator.Emulator, scale int) (*UI, error) {
	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO); err != nil {
		return nil, fmt.Errorf("failed to initialize SDL: %w", err)
	}

	// Set render scale quality hint to nearest-neighbor for pixel-perfect scaling
	// This must be set before creating the renderer
	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "0")

	// Window size: emulator output + info bar at bottom
	// Info bar is 8 pixels tall (scaled)
	infoBarHeight := int32(8 * scale)
	width := int32(320 * scale)
	height := int32(200*scale) + infoBarHeight

	window, err := sdl.CreateWindow(
		"Nitro-Core-DX Emulator",
		sdl.WINDOWPOS_CENTERED,
		sdl.WINDOWPOS_CENTERED,
		width,
		height,
		sdl.WINDOW_SHOWN, // Not resizable to prevent scaling issues
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
		Channels: 2, // Stereo
		Samples:  735, // Samples per frame (44100 Hz / 60 FPS)
	}
	audioDev, err := sdl.OpenAudioDevice("", false, &audioSpec, nil, 0)
	if err != nil {
		// Audio is optional, continue without it
		fmt.Printf("Warning: Failed to open audio device: %v\n", err)
		audioDev = 0
	} else {
		sdl.PauseAudioDevice(audioDev, false) // Start playback
	}

	return &UI{
		window:   window,
		renderer: renderer,
		texture:  texture,
		emulator: emu,
		running:  true,
		scale:    scale,
		audioDev: audioDev,
	}, nil
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
				// Check queue size to prevent backing up (limit to ~2 frames worth)
				queuedBytes := sdl.GetQueuedAudioSize(u.audioDev)
				maxQueuedBytes := uint32(len(audioSamples) * 2 * 4 * 2) // 2 frames worth
				
				// Debug: Log first frame's audio samples
				if u.debugFrameCount < 1 {
					fmt.Printf("[UI] Queuing %d audio samples, queuedBytes=%d, maxQueuedBytes=%d\n", 
						len(audioSamples), queuedBytes, maxQueuedBytes)
					if len(audioSamples) > 0 {
						fmt.Printf("[UI] First 5 samples: %.6f, %.6f, %.6f, %.6f, %.6f\n",
							audioSamples[0], audioSamples[1], audioSamples[2], audioSamples[3], audioSamples[4])
					}
					u.debugFrameCount++
				}
				
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
						fmt.Printf("[UI ERROR] Failed to queue audio: %v\n", err)
					}
				} else {
					// Queue is full, skip this frame
					if u.debugFrameCount < 2 {
						fmt.Printf("[UI] Audio queue full (%d bytes), skipping frame\n", queuedBytes)
					}
				}
			}
		}

		// Render frame (using manual scaling for perfect pixel rendering)
		if err := u.renderFixed(); err != nil {
			return fmt.Errorf("render error: %w", err)
		}
		
		// Render debug overlay (FPS, CPU cycles)
		u.renderDebugOverlay()

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
	}

	return nil
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
		pixels[i*3] = byte((color >> 16) & 0xFF)     // R
		pixels[i*3+1] = byte((color >> 8) & 0xFF)    // G
		pixels[i*3+2] = byte(color & 0xFF)           // B
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
	fps := u.emulator.GetFPS()
	cycles := u.emulator.GetCPUCyclesPerFrame()
	
	// Format FPS string
	fpsStr := fmt.Sprintf("FPS: %.1f", fps)
	if fps < 1.0 {
		fpsStr = fmt.Sprintf("FPS: %.2f", fps)
	}
	
	// Format cycles string
	cyclesStr := fmt.Sprintf("CPU: %d cycles/frame", cycles)
	
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
	
	// Text position in info bar (centered vertically in the bar)
	textY := emulatorHeight + (infoBarHeight-int32(8*u.scale))/2
	textX1 := int32(4 * u.scale)   // Left side
	textX2 := int32(120 * u.scale) // Right side
	
	// White color for text
	white := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	
	// Draw text in info bar
	u.drawText(u.renderer, fpsStr, textX1, textY, u.scale, white)
	u.drawText(u.renderer, cyclesStr, textX2, textY, u.scale, white)
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

