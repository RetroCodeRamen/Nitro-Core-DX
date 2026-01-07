package panels

import (
	"github.com/veandco/go-sdl2/sdl"
	"nitro-core-dx/internal/debug"
)

// LogViewer represents the log viewer panel
type LogViewer struct {
	rect      *sdl.Rect
	scale     int
	logger    *debug.Logger
	
	// Filtering
	componentFilter map[debug.Component]bool
	levelFilter     debug.LogLevel
	autoScroll      bool
	scrollOffset    int // For virtual scrolling
	
	// Display
	visibleEntries int // Number of entries visible in panel
	entryHeight     int32
}

// NewLogViewer creates a new log viewer panel
func NewLogViewer(scale int, logger *debug.Logger) *LogViewer {
	// Default panel size (will be resizable)
	width := int32(400 * scale)
	height := int32(300 * scale)
	
	// Initialize component filter (all enabled by default)
	componentFilter := make(map[debug.Component]bool)
	componentFilter[debug.ComponentCPU] = true
	componentFilter[debug.ComponentPPU] = true
	componentFilter[debug.ComponentAPU] = true
	componentFilter[debug.ComponentMemory] = true
	componentFilter[debug.ComponentInput] = true
	componentFilter[debug.ComponentUI] = true
	componentFilter[debug.ComponentSystem] = true
	
	return &LogViewer{
		rect: &sdl.Rect{
			X: 0,
			Y: 0,
			W: width,
			H: height,
		},
		scale:           scale,
		logger:          logger,
		componentFilter: componentFilter,
		levelFilter:     debug.LogLevelInfo,
		autoScroll:      true,
		scrollOffset:    0,
		entryHeight:     int32(12 * scale),
	}
}

// SetRect sets the panel position and size
func (l *LogViewer) SetRect(x, y, w, h int32) {
	l.rect.X = x
	l.rect.Y = y
	l.rect.W = w
	l.rect.H = h
	l.visibleEntries = int(h / l.entryHeight)
}

// Render renders the log viewer panel
func (l *LogViewer) Render(renderer *sdl.Renderer) {
	// Draw panel background
	renderer.SetDrawColor(32, 32, 32, 255) // Dark gray
	renderer.FillRect(l.rect)
	
	// Draw panel border
	renderer.SetDrawColor(128, 128, 128, 255) // Gray
	renderer.DrawRect(l.rect)
	
	// Draw title bar
	titleBar := &sdl.Rect{
		X: l.rect.X,
		Y: l.rect.Y,
		W: l.rect.W,
		H: int32(20 * l.scale),
	}
	renderer.SetDrawColor(64, 64, 64, 255) // Darker gray
	renderer.FillRect(titleBar)
	
	// Draw filter controls area
	filterArea := &sdl.Rect{
		X: l.rect.X,
		Y: l.rect.Y + titleBar.H,
		W: l.rect.W,
		H: int32(30 * l.scale),
	}
	renderer.SetDrawColor(48, 48, 48, 255) // Very dark gray
	renderer.FillRect(filterArea)
	
	// Get filtered log entries
	entries := l.getFilteredEntries()
	
	// Calculate which entries to display (virtual scrolling)
	startIdx := 0
	if !l.autoScroll {
		startIdx = l.scrollOffset
	} else {
		// Auto-scroll: show most recent entries
		if len(entries) > l.visibleEntries {
			startIdx = len(entries) - l.visibleEntries
		}
	}
	
	// Draw log entries
	logAreaY := l.rect.Y + titleBar.H + filterArea.H
	logAreaHeight := l.rect.H - titleBar.H - filterArea.H
	
	entryY := logAreaY + 2
	for i := startIdx; i < len(entries) && i < startIdx+l.visibleEntries; i++ {
		entry := entries[i]
		
		// Color code by component
		color := l.getComponentColor(entry.Component)
		renderer.SetDrawColor(color.R, color.G, color.B, 255)
		
		// Draw entry background (alternating for readability)
		if i%2 == 0 {
			entryRect := &sdl.Rect{
				X: l.rect.X + 2,
				Y: entryY,
				W: l.rect.W - 4,
				H: l.entryHeight,
			}
			renderer.SetDrawColor(40, 40, 40, 255) // Slightly lighter for alternating rows
			renderer.FillRect(entryRect)
		}
		
		// Draw entry text (simplified - would need proper font rendering)
		// For now, we'll just draw a colored rectangle as placeholder
		textRect := &sdl.Rect{
			X: l.rect.X + 5,
			Y: entryY + 2,
			W: l.rect.W - 10,
			H: l.entryHeight - 4,
		}
		renderer.SetDrawColor(color.R, color.G, color.B, 128) // Semi-transparent
		renderer.FillRect(textRect)
		
		entryY += l.entryHeight
		if entryY >= logAreaY+logAreaHeight {
			break
		}
	}
}

// getFilteredEntries returns log entries that match the current filters
func (l *LogViewer) getFilteredEntries() []debug.LogEntry {
	allEntries := l.logger.GetEntries()
	filtered := make([]debug.LogEntry, 0, len(allEntries))
	
	for _, entry := range allEntries {
		// Check component filter
		if !l.componentFilter[entry.Component] {
			continue
		}
		
		// Check level filter
		if entry.Level < l.levelFilter {
			continue
		}
		
		filtered = append(filtered, entry)
	}
	
	return filtered
}

// getComponentColor returns the color for a component
func (l *LogViewer) getComponentColor(component debug.Component) sdl.Color {
	switch component {
	case debug.ComponentCPU:
		return sdl.Color{R: 100, G: 150, B: 255, A: 255} // Blue
	case debug.ComponentPPU:
		return sdl.Color{R: 100, G: 255, B: 100, A: 255} // Green
	case debug.ComponentAPU:
		return sdl.Color{R: 255, G: 200, B: 100, A: 255} // Yellow/Orange
	case debug.ComponentMemory:
		return sdl.Color{R: 255, G: 100, B: 255, A: 255} // Magenta
	case debug.ComponentInput:
		return sdl.Color{R: 255, G: 100, B: 100, A: 255} // Red
	case debug.ComponentUI:
		return sdl.Color{R: 200, G: 200, B: 255, A: 255} // Light blue
	case debug.ComponentSystem:
		return sdl.Color{R: 200, G: 200, B: 200, A: 255} // Gray
	default:
		return sdl.Color{R: 255, G: 255, B: 255, A: 255} // White
	}
}

// SetComponentFilter enables/disables a component in the filter
func (l *LogViewer) SetComponentFilter(component debug.Component, enabled bool) {
	l.componentFilter[component] = enabled
}

// SetLevelFilter sets the minimum log level to display
func (l *LogViewer) SetLevelFilter(level debug.LogLevel) {
	l.levelFilter = level
}

// SetAutoScroll enables/disables auto-scrolling
func (l *LogViewer) SetAutoScroll(enabled bool) {
	l.autoScroll = enabled
}

// Scroll scrolls the log view (for manual scrolling)
func (l *LogViewer) Scroll(delta int) {
	l.scrollOffset += delta
	if l.scrollOffset < 0 {
		l.scrollOffset = 0
	}
	// Max scroll will be handled by getFilteredEntries
}



