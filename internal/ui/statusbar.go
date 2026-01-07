package ui

import (
	"github.com/veandco/go-sdl2/sdl"
)

// StatusBar represents the status bar at the bottom
type StatusBar struct {
	height int32
	scale  int
}

// NewStatusBar creates a new status bar
func NewStatusBar(scale int) *StatusBar {
	return &StatusBar{
		height: int32(20 * scale),
		scale:  scale,
	}
}

// Height returns the height of the status bar
func (s *StatusBar) Height() int32 {
	return s.height
}

// Render renders the status bar with FPS, cycles, and frame info
func (s *StatusBar) Render(renderer *sdl.Renderer, width int32, yOffset int32, fps float64, cycles uint32, frameCount uint64) {
	// Draw status bar background (dark gray)
	statusRect := &sdl.Rect{
		X: 0,
		Y: yOffset,
		W: width,
		H: s.height,
	}
	renderer.SetDrawColor(48, 48, 48, 255) // Dark gray
	renderer.FillRect(statusRect)

	// Status bar will be rendered by the main UI using drawText
	// This method just draws the background
	// Text rendering will be handled by the main UI's renderStatusBar method
}

