package ui

import (
	"github.com/veandco/go-sdl2/sdl"
)

// MenuBar represents the menu bar
type MenuBar struct {
	height    int32
	menuItems []string
	itemRects []sdl.Rect // Store positions for click detection
}

// NewMenuBar creates a new menu bar
func NewMenuBar(scale int) *MenuBar {
	return &MenuBar{
		height:    int32(20 * scale),
		menuItems: []string{"File", "Emulation", "View", "Debug", "Help"},
		itemRects: make([]sdl.Rect, 0),
	}
}

// Height returns the height of the menu bar
func (m *MenuBar) Height() int32 {
	return m.height
}

// Render renders the menu bar with proper text labels
func (m *MenuBar) Render(renderer *sdl.Renderer, width int32, textRenderer TextRenderer, scale int) {
	// Draw menu bar background (dark gray)
	menuRect := &sdl.Rect{
		X: 0,
		Y: 0,
		W: width,
		H: m.height,
	}
	renderer.SetDrawColor(64, 64, 64, 255) // Dark gray
	renderer.FillRect(menuRect)

	// Draw menu items with proper text labels
	// Clear previous rects
	m.itemRects = m.itemRects[:0]
	
	textColor := sdl.Color{R: 240, G: 240, B: 240, A: 255} // Light gray/white
	
	x := int32(10 * scale)
	for i, item := range m.menuItems {
		// Estimate text width (will be refined with actual rendering)
		itemWidth := int32(len(item) * 7 * scale) + int32(20*scale) // Padding
		
		// Store rect for click detection
		m.itemRects = append(m.itemRects, sdl.Rect{
			X: x,
			Y: 0,
			W: itemWidth,
			H: m.height,
		})
		
		// Draw menu item text
		if textRenderer != nil {
			textY := (m.height - int32(12*scale)) / 2 // Center vertically (approximate)
			if err := textRenderer.DrawText(renderer, item, x+int32(5*scale), textY, textColor); err != nil {
				// If text rendering fails, skip text but still draw the box
			}
		}
		
		x += itemWidth
		
		// Draw separator (except after last item)
		if i < len(m.menuItems)-1 {
			renderer.SetDrawColor(128, 128, 128, 255)
			renderer.DrawLine(x-int32(5*scale), int32(2*scale), x-int32(5*scale), m.height-int32(2*scale))
		}
	}
}

// HandleClick handles a mouse click on the menu bar
func (m *MenuBar) HandleClick(x, y int32) (string, bool) {
	for i, rect := range m.itemRects {
		if x >= rect.X && x < rect.X+rect.W && y >= rect.Y && y < rect.Y+rect.H {
			return m.menuItems[i], true
		}
	}
	return "", false
}

