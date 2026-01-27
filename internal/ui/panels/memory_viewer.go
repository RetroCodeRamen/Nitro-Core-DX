package panels

import (
	"fmt"

	"nitro-core-dx/internal/emulator"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// MemoryViewer creates a panel showing memory contents in hex dump format
// Returns both the container and an update function that should be called periodically
func MemoryViewer(emu *emulator.Emulator) (*fyne.Container, func()) {
	// Bank selector (0-255)
	bankEntry := widget.NewEntry()
	bankEntry.SetText("0")
	bankLabel := widget.NewLabel("Bank:")

	// Offset selector (0x0000-0xFFFF)
	offsetEntry := widget.NewEntry()
	offsetEntry.SetText("0x0000")
	offsetLabel := widget.NewLabel("Offset:")

	// Memory display (hex dump) - use label with monospace text
	memoryText := widget.NewLabel("")
	memoryText.Wrapping = fyne.TextWrapOff
	memoryScroll := container.NewScroll(memoryText)
	memoryScroll.SetMinSize(fyne.NewSize(400, 400))

	// Current bank and offset
	currentBank := uint8(0)
	currentOffset := uint16(0)

	// Update function (called periodically)
	updateFunc := func() {
		if emu == nil || emu.Bus == nil {
			return
		}

		// Parse bank and offset from entries
		var bank uint8
		var offset uint16
		fmt.Sscanf(bankEntry.Text, "%d", &bank)
		fmt.Sscanf(offsetEntry.Text, "0x%X", &offset)

		// Only update if bank/offset changed
		if bank != currentBank || offset != currentOffset {
			currentBank = bank
			currentOffset = offset
		}

		// Build hex dump (16 bytes per line)
		var dumpText string
		dumpText += fmt.Sprintf("Memory Dump - Bank %d, Offset 0x%04X\n\n", currentBank, currentOffset)

		lines := 16 // Show 16 lines (256 bytes)
		for line := 0; line < lines; line++ {
			lineOffset := offset + uint16(line*16)
			
			// Address
			dumpText += fmt.Sprintf("%02X:%04X  ", currentBank, lineOffset)

			// Hex bytes
			for i := 0; i < 16; i++ {
				byteOffset := lineOffset + uint16(i)
				value := emu.Bus.Read8(currentBank, byteOffset)
				dumpText += fmt.Sprintf("%02X ", value)
			}

			// ASCII representation
			dumpText += " |"
			for i := 0; i < 16; i++ {
				byteOffset := lineOffset + uint16(i)
				value := emu.Bus.Read8(currentBank, byteOffset)
				if value >= 32 && value < 127 {
					dumpText += string(rune(value))
				} else {
					dumpText += "."
				}
			}
			dumpText += "|\n"
		}

		// Update display
		memoryText.SetText(dumpText)
	}

	// Bank/offset input handlers
	bankEntry.OnChanged = func(text string) {
		updateFunc()
	}
	offsetEntry.OnChanged = func(text string) {
		updateFunc()
	}

	// Initial update
	updateFunc()

	controls := container.NewHBox(
		bankLabel,
		bankEntry,
		offsetLabel,
		offsetEntry,
	)

	container := container.NewVBox(
		widget.NewLabel("Memory Viewer"),
		controls,
		memoryScroll,
	)

	return container, updateFunc
}
