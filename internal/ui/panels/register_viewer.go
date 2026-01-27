package panels

import (
	"fmt"
	"os"
	"time"

	"nitro-core-dx/internal/emulator"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// RegisterViewer creates a panel showing CPU registers in real-time
// Returns both the container and an update function that should be called periodically
// window is needed for clipboard access
func RegisterViewer(emu *emulator.Emulator, window fyne.Window) (*fyne.Container, func()) {
	// Register display text (scrollable, selectable for copy/paste)
	registerText := widget.NewMultiLineEntry()
	registerText.Wrapping = fyne.TextWrapOff
	registerText.Disable() // Disable editing but allows selection/copy
	registerScroll := container.NewScroll(registerText)
	registerScroll.SetMinSize(fyne.NewSize(300, 300))

	// Function to format register state as text
	formatRegisterState := func() string {
		if emu == nil || emu.CPU == nil {
			return "CPU not available\n"
		}

		state := emu.CPU.State
		var text string

		text += "=== CPU Registers ===\n\n"

		// General purpose registers
		text += "General Purpose Registers:\n"
		regs := []uint16{state.R0, state.R1, state.R2, state.R3, state.R4, state.R5, state.R6, state.R7}
		for i := 0; i < 8; i++ {
			text += fmt.Sprintf("  R%d: 0x%04X (%5d)  %016b\n", i, regs[i], regs[i], regs[i])
		}

		text += "\nSpecial Registers:\n"
		text += fmt.Sprintf("  PC:  %02X:%04X  (Bank: %02X, Offset: %04X)\n", state.PCBank, state.PCOffset, state.PCBank, state.PCOffset)
		text += fmt.Sprintf("  SP:  0x%04X  (%5d)\n", state.SP, state.SP)
		text += fmt.Sprintf("  PBR: 0x%02X  (%3d)\n", state.PBR, state.PBR)
		text += fmt.Sprintf("  DBR: 0x%02X  (%3d)\n", state.DBR, state.DBR)

		text += fmt.Sprintf("\nFlags Register (0x%02X):\n", state.Flags)
		text += fmt.Sprintf("  Z (Zero):        %d\n", map[bool]int{true: 1, false: 0}[state.Flags&0x01 != 0])
		text += fmt.Sprintf("  N (Negative):    %d\n", map[bool]int{true: 1, false: 0}[state.Flags&0x02 != 0])
		text += fmt.Sprintf("  C (Carry):       %d\n", map[bool]int{true: 1, false: 0}[state.Flags&0x04 != 0])
		text += fmt.Sprintf("  V (Overflow):    %d\n", map[bool]int{true: 1, false: 0}[state.Flags&0x08 != 0])
		text += fmt.Sprintf("  I (Interrupt):   %d\n", map[bool]int{true: 1, false: 0}[state.Flags&0x10 != 0])
		text += fmt.Sprintf("  D (Div by Zero): %d\n", map[bool]int{true: 1, false: 0}[state.Flags&0x20 != 0])

		text += "\nCPU State:\n"
		text += fmt.Sprintf("  Cycles: %d\n", state.Cycles)
		text += fmt.Sprintf("  Running: %v\n", emu.Running)
		text += fmt.Sprintf("  Paused: %v\n", emu.Paused)

		return text
	}

	// Update function (called periodically)
	updateFunc := func() {
		registerText.SetText(formatRegisterState())
	}

	// Copy button - copy all text to clipboard
	copyBtn := widget.NewButton("Copy All", func() {
		text := registerText.Text
		if text != "" && window != nil {
			// Copy to clipboard
			window.Clipboard().SetContent(text)
		}
	})

	// Save to file button
	saveBtn := widget.NewButton("Save State", func() {
		// Generate filename with timestamp
		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("register_state_%s.txt", timestamp)

		// Get current register state
		stateText := formatRegisterState()
		stateText = fmt.Sprintf("Register State Dump\nGenerated: %s\n\n%s", 
			time.Now().Format("2006-01-02 15:04:05"), stateText)

		// Write to file
		err := os.WriteFile(filename, []byte(stateText), 0644)
		if err != nil {
			// Show error dialog (would need window reference)
			fmt.Printf("Error saving register state: %v\n", err)
		} else {
			fmt.Printf("Register state saved to: %s\n", filename)
		}
	})

	// Button container
	buttons := container.NewHBox(copyBtn, saveBtn)

	// Initial update
	updateFunc()

	container := container.NewVBox(
		widget.NewLabel("CPU Registers"),
		buttons,
		registerScroll,
	)

	return container, updateFunc
}
