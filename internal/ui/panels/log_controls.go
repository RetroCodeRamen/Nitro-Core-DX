package panels

import (
	"github.com/veandco/go-sdl2/sdl"
	"nitro-core-dx/internal/debug"
	"nitro-core-dx/internal/cpu"
)

// LogControls represents the log control panel for toggling components and levels
type LogControls struct {
	rect      *sdl.Rect
	scale     int
	logger    *debug.Logger
	cpuLogger *cpu.CPULoggerAdapter
	
	// Component enable/disable states
	componentEnabled map[debug.Component]bool
	
	// CPU log level
	cpuLogLevel cpu.CPULogLevel
	
	// Checkbox positions for click detection
	checkboxRects map[debug.Component]*sdl.Rect
}

// NewLogControls creates a new log controls panel
func NewLogControls(scale int, logger *debug.Logger, cpuLogger *cpu.CPULoggerAdapter) *LogControls {
	width := int32(300 * scale)
	height := int32(200 * scale)
	
	// Initialize component states (all disabled by default - logging is opt-in)
	componentEnabled := make(map[debug.Component]bool)
	if logger != nil {
		// Sync with logger's current state
		componentEnabled[debug.ComponentCPU] = logger.IsComponentEnabled(debug.ComponentCPU)
		componentEnabled[debug.ComponentPPU] = logger.IsComponentEnabled(debug.ComponentPPU)
		componentEnabled[debug.ComponentAPU] = logger.IsComponentEnabled(debug.ComponentAPU)
		componentEnabled[debug.ComponentMemory] = logger.IsComponentEnabled(debug.ComponentMemory)
		componentEnabled[debug.ComponentInput] = logger.IsComponentEnabled(debug.ComponentInput)
		componentEnabled[debug.ComponentUI] = logger.IsComponentEnabled(debug.ComponentUI)
		componentEnabled[debug.ComponentSystem] = logger.IsComponentEnabled(debug.ComponentSystem)
	} else {
		// All disabled if no logger
		componentEnabled[debug.ComponentCPU] = false
		componentEnabled[debug.ComponentPPU] = false
		componentEnabled[debug.ComponentAPU] = false
		componentEnabled[debug.ComponentMemory] = false
		componentEnabled[debug.ComponentInput] = false
		componentEnabled[debug.ComponentUI] = false
		componentEnabled[debug.ComponentSystem] = false
	}
	
	return &LogControls{
		rect: &sdl.Rect{
			X: 0,
			Y: 0,
			W: width,
			H: height,
		},
		scale:            scale,
		logger:           logger,
		cpuLogger:        cpuLogger,
		componentEnabled: componentEnabled,
		cpuLogLevel:      cpu.CPULogNone, // Default level (disabled)
		checkboxRects:    make(map[debug.Component]*sdl.Rect),
	}
}

// SetRect sets the panel position and size
func (l *LogControls) SetRect(x, y, w, h int32) {
	l.rect.X = x
	l.rect.Y = y
	l.rect.W = w
	l.rect.H = h
}

// Render renders the log controls panel
func (l *LogControls) Render(renderer *sdl.Renderer) {
	// Draw panel background
	renderer.SetDrawColor(40, 40, 40, 255) // Dark gray
	renderer.FillRect(l.rect)
	
	// Draw panel border
	renderer.SetDrawColor(128, 128, 128, 255) // Gray
	renderer.DrawRect(l.rect)
	
	// Draw title bar
	titleBar := &sdl.Rect{
		X: l.rect.X,
		Y: l.rect.Y,
		W: l.rect.W,
		H: int32(25 * l.scale),
	}
	renderer.SetDrawColor(64, 64, 64, 255) // Darker gray
	renderer.FillRect(titleBar)
	
	// Draw component checkboxes (simplified - would need proper UI library for real checkboxes)
	// For now, just draw labels and indicate enabled state with color
	y := l.rect.Y + titleBar.H + 5
	components := []struct {
		comp debug.Component
		name string
	}{
		{debug.ComponentCPU, "CPU"},
		{debug.ComponentPPU, "PPU"},
		{debug.ComponentAPU, "APU"},
		{debug.ComponentMemory, "Memory"},
		{debug.ComponentInput, "Input"},
		{debug.ComponentUI, "UI"},
		{debug.ComponentSystem, "System"},
	}
	
	// Store checkbox positions for click detection
	l.checkboxRects = make(map[debug.Component]*sdl.Rect)
	
	for i, comp := range components {
		enabled := l.componentEnabled[comp.comp]
		
		// Draw checkbox (simplified - just a colored box)
		checkboxRect := &sdl.Rect{
			X: l.rect.X + 5,
			Y: y + int32(i*20*l.scale),
			W: int32(12 * l.scale),
			H: int32(12 * l.scale),
		}
		
		// Store for click detection
		l.checkboxRects[comp.comp] = checkboxRect
		
		if enabled {
			renderer.SetDrawColor(100, 200, 100, 255) // Green
		} else {
			renderer.SetDrawColor(200, 100, 100, 255) // Red
		}
		renderer.FillRect(checkboxRect)
		
		// Draw border
		renderer.SetDrawColor(200, 200, 200, 255)
		renderer.DrawRect(checkboxRect)
		
		// Note: Text rendering would go here (would need font rendering)
		// For now, the visual checkbox indicates state
	}
	
	// Draw CPU log level selector (simplified)
	// Would need dropdown or buttons in real implementation
	cpuLevelY := y + int32(len(components)*20*l.scale) + 10
	cpuLevelRect := &sdl.Rect{
		X: l.rect.X + 5,
		Y: cpuLevelY,
		W: l.rect.W - 10,
		H: int32(60 * l.scale),
	}
	renderer.SetDrawColor(50, 50, 50, 255)
	renderer.FillRect(cpuLevelRect)
	
	// Draw current CPU log level indicator
	levelNames := []string{
		"None", "Errors", "Branches", "Memory", "Registers", "Instructions", "Trace",
	}
	if int(l.cpuLogLevel) < len(levelNames) {
		// Visual indicator for current level
		indicatorRect := &sdl.Rect{
			X: cpuLevelRect.X + 5,
			Y: cpuLevelRect.Y + 5,
			W: cpuLevelRect.W - 10,
			H: int32(15 * l.scale),
		}
		renderer.SetDrawColor(150, 150, 255, 255) // Light blue
		renderer.FillRect(indicatorRect)
	}
}

// ToggleComponent toggles a component's logging
func (l *LogControls) ToggleComponent(component debug.Component) {
	enabled := l.componentEnabled[component]
	l.componentEnabled[component] = !enabled
	
	// Update logger
	if l.logger != nil {
		l.logger.SetComponentEnabled(component, !enabled)
	}
}

// SetCPULogLevel sets the CPU logging level
func (l *LogControls) SetCPULogLevel(level cpu.CPULogLevel) {
	l.cpuLogLevel = level
	if l.cpuLogger != nil {
		l.cpuLogger.SetLevel(level)
	}
}

// GetCPULogLevel returns the current CPU logging level
func (l *LogControls) GetCPULogLevel() cpu.CPULogLevel {
	return l.cpuLogLevel
}

// IsComponentEnabled returns whether a component is enabled
func (l *LogControls) IsComponentEnabled(component debug.Component) bool {
	return l.componentEnabled[component]
}

// HandleClick handles a mouse click at the given coordinates (relative to panel)
func (l *LogControls) HandleClick(x, y int32) {
	// Check if click is on any checkbox
	for comp, rect := range l.checkboxRects {
		if x >= rect.X-l.rect.X && x < rect.X-l.rect.X+rect.W &&
		   y >= rect.Y-l.rect.Y && y < rect.Y-l.rect.Y+rect.H {
			// Clicked on this checkbox - toggle it
			l.ToggleComponent(comp)
			return
		}
	}
	
	// Check if click is on CPU log level selector (simplified - just cycle through levels)
	cpuLevelY := l.rect.Y + int32(25*l.scale) + int32(7*20*l.scale) + 10
	cpuLevelRect := &sdl.Rect{
		X: l.rect.X + 5,
		Y: cpuLevelY,
		W: l.rect.W - 10,
		H: int32(60 * l.scale),
	}
	if x >= cpuLevelRect.X-l.rect.X && x < cpuLevelRect.X-l.rect.X+cpuLevelRect.W &&
	   y >= cpuLevelRect.Y-l.rect.Y && y < cpuLevelRect.Y-l.rect.Y+cpuLevelRect.H {
		// Clicked on CPU log level selector - cycle to next level
		nextLevel := (l.cpuLogLevel + 1) % (cpu.CPULogTrace + 1)
		l.SetCPULogLevel(nextLevel)
	}
}

