package panels

import (
	"fmt"
	"os"
	"time"

	"nitro-core-dx/internal/debug"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// LogViewerFyne creates a Fyne panel showing log entries
// Returns both the container and an update function that should be called periodically
func LogViewerFyne(logger *debug.Logger, window fyne.Window) (*fyne.Container, func()) {
	// Log display text (scrollable, selectable for copy/paste)
	logText := widget.NewMultiLineEntry()
	logText.Wrapping = fyne.TextWrapOff
	// Disable editing but allows text selection and copy (Ctrl+C works)
	logText.Disable()
	logScroll := container.NewScroll(logText)
	logScroll.SetMinSize(fyne.NewSize(600, 400))

	// Component filter checkboxes
	cpuCheck := widget.NewCheck("CPU", nil)
	ppuCheck := widget.NewCheck("PPU", nil)
	apuCheck := widget.NewCheck("APU", nil)
	memCheck := widget.NewCheck("Memory", nil)
	inputCheck := widget.NewCheck("Input", nil)
	uiCheck := widget.NewCheck("UI", nil)
	sysCheck := widget.NewCheck("System", nil)

	// Set all enabled by default
	cpuCheck.SetChecked(true)
	ppuCheck.SetChecked(true)
	apuCheck.SetChecked(true)
	memCheck.SetChecked(true)
	inputCheck.SetChecked(true)
	uiCheck.SetChecked(true)
	sysCheck.SetChecked(true)

	// Level filter dropdown
	levelSelect := widget.NewSelect([]string{"None", "Error", "Warning", "Info", "Debug", "Trace"}, nil)
	levelSelect.SetSelected("Info")

	// Auto-scroll checkbox
	autoScrollCheck := widget.NewCheck("Auto-scroll", nil)
	autoScrollCheck.SetChecked(true)

	// Copy button - copy all visible text to clipboard
	copyBtn := widget.NewButton("Copy All", func() {
		text := logText.Text
		if text != "" && window != nil {
			window.Clipboard().SetContent(text)
		}
	})

	// Save to file button
	saveBtn := widget.NewButton("Save Logs", func() {
		// Generate filename with timestamp
		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("logs_%s.txt", timestamp)

		// Get current log text
		logContent := logText.Text
		if logContent == "" {
			logContent = "No log entries"
		}
		
		// Add header
		logContent = fmt.Sprintf("Nitro Core DX Logs\nGenerated: %s\n\n%s",
			time.Now().Format("2006-01-02 15:04:05"), logContent)

		// Write to file
		err := os.WriteFile(filename, []byte(logContent), 0644)
		if err != nil {
			fmt.Printf("Error saving logs: %v\n", err)
		} else {
			fmt.Printf("Logs saved to: %s\n", filename)
		}
	})

	// Filter container
	filterContainer := container.NewVBox(
		// First row: component filters
		container.NewHBox(
			widget.NewLabel("Components:"),
			cpuCheck,
			ppuCheck,
			apuCheck,
			memCheck,
			inputCheck,
			uiCheck,
			sysCheck,
		),
		// Second row: level filter, auto-scroll, and action buttons
		container.NewHBox(
			widget.NewLabel("Level:"),
			levelSelect,
			autoScrollCheck,
			widget.NewSeparator(),
			copyBtn,
			saveBtn,
		),
	)

	// Update function
	updateLogs := func() {
		if logger == nil {
			logText.SetText("Logger not available")
			return
		}

		// Get component filter state
		componentFilter := make(map[debug.Component]bool)
		componentFilter[debug.ComponentCPU] = cpuCheck.Checked
		componentFilter[debug.ComponentPPU] = ppuCheck.Checked
		componentFilter[debug.ComponentAPU] = apuCheck.Checked
		componentFilter[debug.ComponentMemory] = memCheck.Checked
		componentFilter[debug.ComponentInput] = inputCheck.Checked
		componentFilter[debug.ComponentUI] = uiCheck.Checked
		componentFilter[debug.ComponentSystem] = sysCheck.Checked

		// Get level filter
		var levelFilter debug.LogLevel
		switch levelSelect.Selected {
		case "None":
			levelFilter = debug.LogLevelNone
		case "Error":
			levelFilter = debug.LogLevelError
		case "Warning":
			levelFilter = debug.LogLevelWarning
		case "Info":
			levelFilter = debug.LogLevelInfo
		case "Debug":
			levelFilter = debug.LogLevelDebug
		case "Trace":
			levelFilter = debug.LogLevelTrace
		default:
			levelFilter = debug.LogLevelInfo
		}

		// Get all entries
		allEntries := logger.GetEntries()

		// Filter entries
		filtered := make([]debug.LogEntry, 0, len(allEntries))
		for _, entry := range allEntries {
			// Check component filter
			if !componentFilter[entry.Component] {
				continue
			}

			// Check level filter
			if entry.Level < levelFilter {
				continue
			}

			filtered = append(filtered, entry)
		}

		// Format entries as text
		var text string
		if len(filtered) == 0 {
			text = "No log entries (filters may be too restrictive)"
		} else {
			// Show most recent entries if auto-scroll
			startIdx := 0
			maxEntries := 1000 // Limit to prevent UI lag
			if autoScrollCheck.Checked && len(filtered) > maxEntries {
				startIdx = len(filtered) - maxEntries
			}

			for i := startIdx; i < len(filtered); i++ {
				entry := filtered[i]
				timestamp := entry.Timestamp.Format("15:04:05.000")
				text += fmt.Sprintf("[%s] [%s] %s: %s\n",
					timestamp, entry.Component, entry.Level, entry.Message)
			}
		}

		logText.SetText(text)

		// Auto-scroll to bottom if enabled
		if autoScrollCheck.Checked {
			logScroll.ScrollToBottom()
		}
	}

	// Main container
	mainContainer := container.NewBorder(
		filterContainer, // Top
		nil,             // Bottom
		nil,             // Left
		nil,             // Right
		logScroll,       // Center
	)

	return mainContainer, updateLogs
}
