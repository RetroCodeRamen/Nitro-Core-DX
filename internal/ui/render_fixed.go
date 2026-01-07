package ui

import (
	"fmt"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

// Static debug counter to limit debug output
var staticDebugCounter int

// renderFixed renders with manual pixel scaling for perfect integer scaling
func (u *UI) renderFixed() error {
	// Get output buffer from emulator
	buffer := u.emulator.GetOutputBuffer()

	// Ensure we have exactly 320*200 pixels
	if len(buffer) != 320*200 {
		return fmt.Errorf("buffer size mismatch: expected %d, got %d", 320*200, len(buffer))
	}
	
	// Get renderer output size first
	outputW, outputH, _ := u.renderer.GetOutputSize()
	
	// Calculate UI element heights
	menuBarHeight := int32(20 * u.scale)
	toolbarHeight := int32(30 * u.scale)
	statusBarHeight := int32(20 * u.scale)
	emulatorHeight := int32(200 * u.scale)
	
	expectedW := int32(320 * u.scale)
	expectedH := menuBarHeight + toolbarHeight + emulatorHeight + statusBarHeight
	
	// Verify window size matches expected
	if int32(outputW) != expectedW || int32(outputH) != expectedH {
		u.window.SetSize(expectedW, expectedH)
		outputW, outputH, _ = u.renderer.GetOutputSize()
	}
	
	// Create scaled pixel buffer manually for perfect integer scaling
	// This avoids any SDL scaling interpolation issues
	scaledW := 320 * u.scale
	scaledH := 200 * u.scale
	// Check if texture uses ARGB8888 (4 bytes) or RGB888 (3 bytes)
	var textureFormat uint32 = sdl.PIXELFORMAT_RGB888
	bytesPerPixel := 3
	if u.texture != nil {
		format, _, _, _, _ := u.texture.Query()
		textureFormat = format
		if format == sdl.PIXELFORMAT_ARGB8888 {
			bytesPerPixel = 4
		}
	}
	scaledPixels := make([]byte, scaledW*scaledH*bytesPerPixel)
	
	// Initialize to black first (ensures no garbage data)
	for i := range scaledPixels {
		scaledPixels[i] = 0
	}
	
	// Manually scale each pixel (nearest-neighbor - perfect integer scaling)
	// Each original pixel becomes a scale×scale block
	// Process row by row for better cache locality
	for y := 0; y < 200; y++ {
		baseY := y * u.scale
		for x := 0; x < 320; x++ {
			// Get color from buffer (RGB888 format: 0xRRGGBB)
			color := buffer[y*320+x]
			// Extract RGB components
			r := byte((color >> 16) & 0xFF) // Red in upper byte
			g := byte((color >> 8) & 0xFF)  // Green in middle byte
			b := byte(color & 0xFF)          // Blue in lower byte
			
			// Calculate base position in scaled buffer
			baseX := x * u.scale
			
			// Scale this pixel to scale×scale block
			// Write all pixels in the block
			for sy := 0; sy < u.scale; sy++ {
				scaledY := baseY + sy
				if scaledY >= scaledH {
					break
				}
				rowStart := scaledY * scaledW * bytesPerPixel
				for sx := 0; sx < u.scale; sx++ {
					scaledX := baseX + sx
					if scaledX >= scaledW {
						break
					}
					// Calculate index: rowStart + (x * bytesPerPixel)
					idx := rowStart + (scaledX * bytesPerPixel)
					if textureFormat == sdl.PIXELFORMAT_ARGB8888 {
						// ARGB8888 format: The format code 0x16362004 suggests it might be BGRA or RGBA
						// Based on seeing cyan/magenta, try BGRA order (Blue, Green, Red, Alpha)
						if idx+3 < len(scaledPixels) {
							scaledPixels[idx] = b      // Blue first (BGRA)
							scaledPixels[idx+1] = g    // Green
							scaledPixels[idx+2] = r    // Red
							scaledPixels[idx+3] = 0xFF // Alpha (fully opaque)
						}
					} else {
						// RGB888 format: Try BGR order since we're seeing channel swaps
						if idx+2 < len(scaledPixels) {
							scaledPixels[idx] = b     // Blue first (BGR)
							scaledPixels[idx+1] = g   // Green
							scaledPixels[idx+2] = r   // Red last
						}
					}
				}
			}
		}
	}
	
	// Always recreate texture to ensure correct size (textures can't be resized)
	// Destroy old texture if it exists
	if u.texture != nil {
		w, h, _, _, _ := u.texture.Query()
		if int(w) != scaledW || int(h) != scaledH {
			u.texture.Destroy()
			u.texture = nil
		}
	}
	
	// Create texture at scaled size if it doesn't exist or was destroyed
	if u.texture == nil {
		var err error
		// Use ARGB8888 format (with alpha) - SDL2 renderer may convert RGB888 incorrectly
		// ARGB8888 is more commonly supported and less likely to have channel swapping issues
		u.texture, err = u.renderer.CreateTexture(
			sdl.PIXELFORMAT_ARGB8888,
			sdl.TEXTUREACCESS_STREAMING,
			int32(scaledW),
			int32(scaledH),
		)
		if err != nil {
			// Fallback to RGB888
			u.texture, err = u.renderer.CreateTexture(
				sdl.PIXELFORMAT_RGB888,
				sdl.TEXTUREACCESS_STREAMING,
				int32(scaledW),
				int32(scaledH),
			)
			if err != nil {
				return fmt.Errorf("failed to create scaled texture: %w", err)
			}
		}
	}
	
	// Update texture with scaled pixels
	// Pitch is bytes per row: width * bytesPerPixel
	// Update the entire texture region
	pitch := scaledW * bytesPerPixel
	rect := &sdl.Rect{X: 0, Y: 0, W: int32(scaledW), H: int32(scaledH)}
	if err := u.texture.Update(rect, unsafe.Pointer(&scaledPixels[0]), pitch); err != nil {
		return fmt.Errorf("failed to update texture: %w", err)
	}
	
	// DEBUG: Print first row of pixels to diagnose color issues
	// Only print once to avoid spam - use a static counter
	staticDebugCounter++
	if staticDebugCounter <= 3 { // Print first 3 frames
		u.debugPixelOutput(scaledPixels, scaledW, bytesPerPixel, textureFormat)
	}
	
	// Set texture blend mode to none (no alpha blending)
	u.texture.SetBlendMode(sdl.BLENDMODE_NONE)
	
	// Ensure no color modulation is applied
	u.texture.SetColorMod(255, 255, 255)

	// Don't clear renderer here - let renderUI() handle the full screen clear
	// We'll just draw the emulator screen texture at the correct position
	// Position emulator screen below menu bar and toolbar
	// (menuBarHeight and toolbarHeight already calculated above)
	emulatorY := menuBarHeight + toolbarHeight
	dstRect := &sdl.Rect{
		X: 0, 
		Y: emulatorY, 
		W: int32(scaledW), 
		H: int32(scaledH),
	}
	srcRect := &sdl.Rect{X: 0, Y: 0, W: int32(scaledW), H: int32(scaledH)}
	
	u.renderer.SetScale(1.0, 1.0)
	
	if err := u.renderer.Copy(u.texture, srcRect, dstRect); err != nil {
		return fmt.Errorf("failed to copy texture: %w", err)
	}
	
	// Present is called in the main loop, not here
	return nil
}

// debugPixelOutput prints the first row of pixels to help diagnose color issues
func (u *UI) debugPixelOutput(scaledPixels []byte, scaledW, bytesPerPixel int, textureFormat uint32) {
	fmt.Printf("\n=== DEBUG: First row of scaled pixels (width=%d, bytesPerPixel=%d, format=0x%08X) ===\n", scaledW, bytesPerPixel, textureFormat)
	
	// Print first row (y=0)
	rowStart := 0
	fmt.Printf("Expected pattern: 20px RED, 20px GREEN, 20px BLUE (repeating)\n")
	fmt.Printf("Actual pixels (first 100 pixels):\n")
	
	for x := 0; x < 100 && x < scaledW; x++ {
		idx := rowStart + (x * bytesPerPixel)
		if idx+2 >= len(scaledPixels) {
			break
		}
		
		var r, g, b byte
		var rawBytes string
		if textureFormat == sdl.PIXELFORMAT_ARGB8888 {
			if idx+3 < len(scaledPixels) {
				r = scaledPixels[idx+1] // ARGB: A, R, G, B
				g = scaledPixels[idx+2]
				b = scaledPixels[idx+3]
				rawBytes = fmt.Sprintf("[%02X %02X %02X %02X]", scaledPixels[idx], scaledPixels[idx+1], scaledPixels[idx+2], scaledPixels[idx+3])
			}
		} else {
			r = scaledPixels[idx]     // RGB: R, G, B
			g = scaledPixels[idx+1]
			b = scaledPixels[idx+2]
			rawBytes = fmt.Sprintf("[%02X %02X %02X]", scaledPixels[idx], scaledPixels[idx+1], scaledPixels[idx+2])
		}
		
		// Determine expected color based on x position
		// Pattern: 20px red, 20px green, 20px blue (repeating)
		// At scale 3, 20px becomes 60px
		barIndex := x / (20 * u.scale)
		pixelPattern := barIndex % 3
		
		var expectedR, expectedG, expectedB byte
		var expectedColor string
		switch pixelPattern {
		case 0:
			expectedR, expectedG, expectedB = 0xFF, 0x00, 0x00
			expectedColor = "RED"
		case 1:
			expectedR, expectedG, expectedB = 0x00, 0xFF, 0x00
			expectedColor = "GREEN"
		case 2:
			expectedR, expectedG, expectedB = 0x00, 0x00, 0xFF
			expectedColor = "BLUE"
		}
		
		// Check if colors match
		match := (r == expectedR && g == expectedG && b == expectedB)
		matchStr := "✓"
		if !match {
			matchStr = "✗"
		}
		
		if x%10 == 0 || !match {
			fmt.Printf("  x=%3d: RGB(%02X,%02X,%02X) %s [%s] expected %s RGB(%02X,%02X,%02X)\n",
				x, r, g, b, rawBytes, matchStr, expectedColor, expectedR, expectedG, expectedB)
		}
	}
	
	// Also check the source buffer
	fmt.Printf("\n=== DEBUG: First row of source buffer (320 pixels) ===\n")
	buffer := u.emulator.GetOutputBuffer()
	for x := 0; x < 100 && x < 320; x++ {
		color := buffer[x]
		r := byte((color >> 16) & 0xFF)
		g := byte((color >> 8) & 0xFF)
		b := byte(color & 0xFF)
		
		barIndex := x / 20
		pixelPattern := barIndex % 3
		var expectedR, expectedG, expectedB byte
		var expectedColor string
		switch pixelPattern {
		case 0:
			expectedR, expectedG, expectedB = 0xFF, 0x00, 0x00
			expectedColor = "RED"
		case 1:
			expectedR, expectedG, expectedB = 0x00, 0xFF, 0x00
			expectedColor = "GREEN"
		case 2:
			expectedR, expectedG, expectedB = 0x00, 0x00, 0xFF
			expectedColor = "BLUE"
		}
		
		match := (r == expectedR && g == expectedG && b == expectedB)
		matchStr := "✓"
		if !match {
			matchStr = "✗"
		}
		
		if x%10 == 0 || !match {
			fmt.Printf("  x=%3d: RGB(%02X,%02X,%02X) [%s] expected %s RGB(%02X,%02X,%02X)\n",
				x, r, g, b, matchStr, expectedColor, expectedR, expectedG, expectedB)
		}
	}
	fmt.Printf("=== END DEBUG ===\n\n")
}

