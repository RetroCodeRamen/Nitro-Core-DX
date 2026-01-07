package ui

import (
	"github.com/veandco/go-sdl2/sdl"
)

// Toolbar represents the toolbar with quick action buttons
type Toolbar struct {
	height     int32
	scale      int
	buttonRects []sdl.Rect // Store button positions for click detection
	buttonLabels []string  // Store button labels for click detection
}

// NewToolbar creates a new toolbar
func NewToolbar(scale int) *Toolbar {
	return &Toolbar{
		height: int32(30 * scale),
		scale:  scale,
	}
}

// Height returns the height of the toolbar
func (t *Toolbar) Height() int32 {
	return t.height
}

// Render renders the toolbar with proper text labels
func (t *Toolbar) Render(renderer *sdl.Renderer, width int32, yOffset int32, emuRunning bool, emuPaused bool, textRenderer TextRenderer) {
	// Draw toolbar background (lighter gray)
	toolbarRect := &sdl.Rect{
		X: 0,
		Y: yOffset,
		W: width,
		H: t.height,
	}
	renderer.SetDrawColor(96, 96, 96, 255) // Light gray
	renderer.FillRect(toolbarRect)

	buttonWidth := int32(50 * t.scale)
	buttonHeight := t.height - 4
	
	// Draw buttons: [Start] [Pause] [Resume] [Stop] [Reset] [Step]
	buttons := []struct {
		label   string
		enabled bool
	}{
		{"Start", !emuRunning},
		{"Pause", emuRunning && !emuPaused},
		{"Resume", emuRunning && emuPaused},
		{"Stop", emuRunning},
		{"Reset", true},
		{"Step", emuPaused},
	}
	
	// Clear previous button rects
	t.buttonRects = t.buttonRects[:0]
	t.buttonLabels = t.buttonLabels[:0]
	
	x := int32(5 * t.scale)
	for i, btn := range buttons {
		if !btn.enabled {
			continue
		}
		
		btnRect := &sdl.Rect{
			X: x,
			Y: yOffset + 2,
			W: buttonWidth,
			H: buttonHeight,
		}
		
		// Store for click detection
		t.buttonRects = append(t.buttonRects, *btnRect)
		t.buttonLabels = append(t.buttonLabels, btn.label)
		
		// Draw button background
		renderer.SetDrawColor(128, 128, 128, 255)
		renderer.FillRect(btnRect)
		
		// Draw button border
		renderer.SetDrawColor(64, 64, 64, 255)
		renderer.DrawRect(btnRect)
		
		// Draw button text (centered) with modern font
		if textRenderer != nil {
			textColor := sdl.Color{R: 255, G: 255, B: 255, A: 255} // White
			// Center text in button (approximate - will be refined with actual text size)
			textY := yOffset + (t.height-int32(12*t.scale))/2
			textX := x + (buttonWidth-int32(len(btn.label)*6*t.scale))/2
			if err := textRenderer.DrawText(renderer, btn.label, textX, textY, textColor); err != nil {
				// If text rendering fails, skip text but still draw the button
			}
		}
		
		x += buttonWidth + int32(5*t.scale)
		
		// Draw separator after each button except the last
		if i < len(buttons)-1 {
			renderer.SetDrawColor(64, 64, 64, 255)
			renderer.DrawLine(x-int32(2*t.scale), yOffset+int32(5*t.scale), x-int32(2*t.scale), yOffset+int32(t.height)-int32(5*t.scale))
		}
	}
}

// HandleClick handles a mouse click on the toolbar
func (t *Toolbar) HandleClick(x, y int32) (string, bool) {
	for i, rect := range t.buttonRects {
		if x >= rect.X && x < rect.X+rect.W && y >= rect.Y && y < rect.Y+rect.H {
			return t.buttonLabels[i], true
		}
	}
	return "", false
}

