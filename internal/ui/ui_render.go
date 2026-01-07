package ui

import (
	"fmt"
	"nitro-core-dx/internal/cpu"
	"nitro-core-dx/internal/ui/panels"

	"github.com/veandco/go-sdl2/sdl"
)

// renderUI renders all UI elements (menu bar, toolbar, status bar, panels)
func (u *UI) renderUI() {
	outputW, _, _ := u.renderer.GetOutputSize()

	// Calculate UI element heights
	menuBarHeight := int32(20 * u.scale)
	toolbarHeight := int32(30 * u.scale)
	emulatorHeight := int32(200 * u.scale)

	// Render menu bar
	menuBar := NewMenuBar(u.scale)
	menuBar.Render(u.renderer, int32(outputW), u.textRenderer, u.scale)
	// Store menu bar for click handling
	u.menuBar = menuBar

	// Render toolbar
	toolbar := NewToolbar(u.scale)
	toolbarY := menuBarHeight
	toolbar.Render(u.renderer, int32(outputW), toolbarY, u.emulator.Running, u.emulator.Paused, u.textRenderer)
	// Store toolbar for click handling
	u.toolbar = toolbar
	u.toolbarY = toolbarY

	// Emulator screen is already rendered by renderFixed() at correct position
	// (below menu bar and toolbar)

	// Render status bar
	statusBar := NewStatusBar(u.scale)
	statusBarY := menuBarHeight + toolbarHeight + emulatorHeight
	statusBar.Render(u.renderer, int32(outputW), statusBarY,
		u.emulator.GetFPS(), u.emulator.GetCPUCyclesPerFrame(), u.emulator.FrameCount)

	// Status bar text removed - status bar is just a clean bar for now
	// In a full implementation, you'd use SDL_ttf or another text rendering solution

	// Render debug panels if visible
	if u.showLogViewer && u.emulator.Logger != nil {
		u.renderLogViewer()
	}
	if u.showLogControls && u.emulator.Logger != nil {
		u.renderLogControls()
	}
}

// renderStatusBarText renders the text in the status bar
func (u *UI) renderStatusBarText(yOffset, height int32) {
	if u.textRenderer == nil {
		return
	}

	fps := u.emulator.GetFPS()
	cycles := u.emulator.GetCPUCyclesPerFrame()
	frameCount := u.emulator.FrameCount

	// Format status text
	fpsStr := fmt.Sprintf("FPS: %.1f", fps)
	if fps < 1.0 {
		fpsStr = fmt.Sprintf("FPS: %.2f", fps)
	}
	cyclesStr := fmt.Sprintf("CPU: %d cycles/frame", cycles)
	frameStr := fmt.Sprintf("Frame: %d", frameCount)

	// White color for text
	white := sdl.Color{R: 255, G: 255, B: 255, A: 255}

	// Draw text
	textY := yOffset + (height-int32(12*u.scale))/2
	textX1 := int32(5 * u.scale)
	textX2 := int32(120 * u.scale)
	textX3 := int32(250 * u.scale)

	u.textRenderer.DrawText(u.renderer, fpsStr, textX1, textY, white)
	u.textRenderer.DrawText(u.renderer, cyclesStr, textX2, textY, white)
	u.textRenderer.DrawText(u.renderer, frameStr, textX3, textY, white)
}

// renderLogViewer renders the log viewer panel
func (u *UI) renderLogViewer() {
	if u.emulator.Logger == nil {
		return
	}

	// Calculate panel position (right side of window for now)
	outputW, _, _ := u.renderer.GetOutputSize()
	menuBarHeight := int32(20 * u.scale)
	toolbarHeight := int32(30 * u.scale)

	panelWidth := int32(400 * u.scale)
	panelHeight := int32(300 * u.scale)
	panelX := int32(outputW) - panelWidth - 10
	panelY := menuBarHeight + toolbarHeight + 10

	// Create or update log viewer
	if u.logViewerRect == nil {
		u.logViewerRect = &sdl.Rect{
			X: panelX,
			Y: panelY,
			W: panelWidth,
			H: panelHeight,
		}
	} else {
		u.logViewerRect.X = panelX
		u.logViewerRect.Y = panelY
		u.logViewerRect.W = panelWidth
		u.logViewerRect.H = panelHeight
	}

	// Create log viewer panel and render
	logViewer := panels.NewLogViewer(u.scale, u.emulator.Logger)
	logViewer.SetRect(panelX, panelY, panelWidth, panelHeight)
	logViewer.Render(u.renderer)
}

// renderLogControls renders the log controls panel
func (u *UI) renderLogControls() {
	if u.emulator.Logger == nil {
		return
	}

	// Calculate panel position (left side of window)
	menuBarHeight := int32(20 * u.scale)
	toolbarHeight := int32(30 * u.scale)

	panelWidth := int32(300 * u.scale)
	panelHeight := int32(200 * u.scale)
	panelX := int32(10)
	panelY := menuBarHeight + toolbarHeight + 10

	// Get CPU logger adapter from emulator
	// We need to access it through the CPU - for now, create a new one
	// TODO: Store CPU logger adapter in emulator for easy access
	var cpuLogger *cpu.CPULoggerAdapter
	if u.emulator.CPU != nil && u.emulator.CPU.Log != nil {
		// Try to cast to CPULoggerAdapter
		if adapter, ok := u.emulator.CPU.Log.(*cpu.CPULoggerAdapter); ok {
			cpuLogger = adapter
		}
	}

	// Create or update log controls
	if u.logControlsRect == nil {
		u.logControlsRect = &sdl.Rect{
			X: panelX,
			Y: panelY,
			W: panelWidth,
			H: panelHeight,
		}
	} else {
		u.logControlsRect.X = panelX
		u.logControlsRect.Y = panelY
		u.logControlsRect.W = panelWidth
		u.logControlsRect.H = panelHeight
	}

	// Create or reuse log controls panel
	if u.logControlsPanel == nil {
		u.logControlsPanel = panels.NewLogControls(u.scale, u.emulator.Logger, cpuLogger)
	}
	u.logControlsPanel.SetRect(panelX, panelY, panelWidth, panelHeight)
	u.logControlsPanel.Render(u.renderer)
}
