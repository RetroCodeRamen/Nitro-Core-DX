package panels

import (
	"fmt"
	"image"
	"image/color"

	"nitro-core-dx/internal/emulator"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// TileViewer creates a panel showing tiles from VRAM in a visual grid or single tile view
// Returns both the container and an update function that should be called periodically
func TileViewer(emu *emulator.Emulator) (*fyne.Container, func()) {
	// View mode selector (Grid or Single Tile)
	viewModeSelect := widget.NewSelect([]string{"Grid", "Single Tile"}, func(value string) {})
	viewModeSelect.SetSelected("Grid")
	viewModeLabel := widget.NewLabel("View Mode:")

	// Palette selector (0-15)
	paletteEntry := widget.NewEntry()
	paletteEntry.SetText("0")
	paletteLabel := widget.NewLabel("Palette:")

	// Tile size selector (8x8 or 16x16)
	tileSizeSelect := widget.NewSelect([]string{"8x8", "16x16"}, func(value string) {})
	tileSizeSelect.SetSelected("8x8")
	tileSizeLabel := widget.NewLabel("Tile Size:")

	// Tile selector (which tile to view/edit)
	tileSelectEntry := widget.NewEntry()
	tileSelectEntry.SetText("0")
	tileSelectLabel := widget.NewLabel("Tile #:")

	// Tile offset selector (which tile to start from in grid view)
	tileOffsetEntry := widget.NewEntry()
	tileOffsetEntry.SetText("0")
	tileOffsetLabel := widget.NewLabel("Start Tile:")

	// Grid size selector (how many tiles per row)
	gridSizeSelect := widget.NewSelect([]string{"8", "16", "32"}, func(value string) {})
	gridSizeSelect.SetSelected("16")
	gridSizeLabel := widget.NewLabel("Tiles Per Row:")

	// Color selector for editing (0-15)
	colorSelectEntry := widget.NewEntry()
	colorSelectEntry.SetText("1")
	colorSelectLabel := widget.NewLabel("Color Index:")

	// Current settings
	viewMode := "Grid"
	currentPalette := uint8(0)
	currentTileSize := 8
	currentTileIndex := uint8(0)
	currentTileOffset := uint8(0)
	currentGridSize := 16

	// Create raster for tile display
	tileRaster := canvas.NewRaster(func(w, h int) image.Image {
		if emu == nil || emu.PPU == nil {
			// Return blank image if PPU not available
			img := image.NewRGBA(image.Rect(0, 0, w, h))
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					img.Set(x, y, color.RGBA{0, 0, 0, 255})
				}
			}
			return img
		}

		ppu := emu.PPU
		img := image.NewRGBA(image.Rect(0, 0, w, h))

		if viewMode == "Single Tile" {
			// Single tile view - show one tile large
			tileSize := currentTileSize
			scale := 8 // Scale factor for visibility
			displaySize := tileSize * scale

			// Center the tile
			startX := (w - displaySize) / 2
			if startX < 0 {
				startX = 0
			}
			startY := (h - displaySize) / 2
			if startY < 0 {
				startY = 0
			}

			// Fill background
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					img.Set(x, y, color.RGBA{32, 32, 32, 255}) // Dark gray background
				}
			}

			// Render single tile
			tileDataOffset := uint16(currentTileIndex) * uint16(tileSize*tileSize/2) // 4bpp = 2 pixels per byte

			for py := 0; py < tileSize; py++ {
				for px := 0; px < tileSize; px++ {
					// Calculate pixel offset in tile
					pixelOffsetInTile := py*tileSize + px
					byteOffsetInTile := pixelOffsetInTile / 2
					pixelInByte := pixelOffsetInTile % 2

					// Check bounds
					if uint32(tileDataOffset)+uint32(byteOffsetInTile) >= uint32(len(ppu.VRAM)) {
						continue
					}

					tileDataAddr := tileDataOffset + uint16(byteOffsetInTile)

					// Read pixel color index (4 bits)
					tileByte := ppu.VRAM[tileDataAddr]
					var colorIndex uint8
					if pixelInByte == 0 {
						colorIndex = (tileByte >> 4) & 0x0F // Upper 4 bits
					} else {
						colorIndex = tileByte & 0x0F // Lower 4 bits
					}

					// Convert CGRAM color to RGB
					cgramAddr := (uint16(currentPalette)*16 + uint16(colorIndex)) * 2
					var r, g, b uint8
					if cgramAddr < 512 {
						low := ppu.CGRAM[cgramAddr]
						high := ppu.CGRAM[cgramAddr+1]

						// Extract RGB555 components
						r555 := uint32((high & 0x7C) >> 2)
						g555 := uint32(((high & 0x03) << 3) | ((low & 0xE0) >> 5))
						b555 := uint32(low & 0x1F)

						// Scale to 8 bits
						r = uint8((r555 * 255) / 31)
						g = uint8((g555 * 255) / 31)
						b = uint8((b555 * 255) / 31)
					} else {
						r, g, b = 0, 0, 0
					}

					// Draw scaled pixel
					for sy := 0; sy < scale; sy++ {
						for sx := 0; sx < scale; sx++ {
							imgX := startX + px*scale + sx
							imgY := startY + py*scale + sy
							if imgX < w && imgY < h {
								img.Set(imgX, imgY, color.RGBA{r, g, b, 255})
							}
						}
					}
				}
			}

			// Draw grid lines
			for i := 0; i <= tileSize; i++ {
				// Vertical lines
				x := startX + i*scale
				if x < w {
					for y := startY; y < startY+displaySize && y < h; y++ {
						img.Set(x, y, color.RGBA{128, 128, 128, 255})
					}
				}
				// Horizontal lines
				y := startY + i*scale
				if y < h {
					for x := startX; x < startX+displaySize && x < w; x++ {
						img.Set(x, y, color.RGBA{128, 128, 128, 255})
					}
				}
			}
		} else {
			// Grid view (original code)
			tilesPerRow := currentGridSize
			tileSize := currentTileSize
			tilePixelSize := tileSize + 1 // +1 for grid line
			gridWidth := tilesPerRow * tilePixelSize
			gridHeight := ((256 / tilesPerRow) + 1) * tilePixelSize // Show up to 256 tiles

			// Limit to actual widget size
			if gridWidth > w {
				gridWidth = w
			}
			if gridHeight > h {
				gridHeight = h
			}

			// Fill background
			for y := 0; y < gridHeight; y++ {
				for x := 0; x < gridWidth; x++ {
					img.Set(x, y, color.RGBA{32, 32, 32, 255}) // Dark gray background
				}
			}

			// Render tiles
			tilesToShow := 256 // Show up to 256 tiles
			for tileIndex := 0; tileIndex < tilesToShow && uint8(tileIndex) >= currentTileOffset; tileIndex++ {
				actualTileIndex := uint8(tileIndex) - currentTileOffset
				if int(actualTileIndex) >= tilesToShow {
					break
				}

				// Calculate grid position
				gridX := (tileIndex - int(currentTileOffset)) % tilesPerRow
				gridY := (tileIndex - int(currentTileOffset)) / tilesPerRow

				// Calculate pixel position
				pixelX := gridX * tilePixelSize
				pixelY := gridY * tilePixelSize

				// Render tile
				tileDataOffset := uint16(actualTileIndex) * uint16(tileSize*tileSize/2) // 4bpp = 2 pixels per byte

				for py := 0; py < tileSize; py++ {
					for px := 0; px < tileSize; px++ {
						// Calculate pixel offset in tile
						pixelOffsetInTile := py*tileSize + px
						byteOffsetInTile := pixelOffsetInTile / 2
						pixelInByte := pixelOffsetInTile % 2

					// Check bounds
					if uint32(tileDataOffset)+uint32(byteOffsetInTile) >= uint32(len(ppu.VRAM)) {
						continue
					}

						tileDataAddr := tileDataOffset + uint16(byteOffsetInTile)

						// Read pixel color index (4 bits)
						tileByte := ppu.VRAM[tileDataAddr]
						var colorIndex uint8
						if pixelInByte == 0 {
							colorIndex = (tileByte >> 4) & 0x0F // Upper 4 bits
						} else {
							colorIndex = tileByte & 0x0F // Lower 4 bits
						}

						// Convert CGRAM color to RGB
						cgramAddr := (uint16(currentPalette)*16 + uint16(colorIndex)) * 2
						var r, g, b uint8
						if cgramAddr < 512 {
							low := ppu.CGRAM[cgramAddr]
							high := ppu.CGRAM[cgramAddr+1]

							// Extract RGB555 components
							r555 := uint32((high & 0x7C) >> 2)
							g555 := uint32(((high & 0x03) << 3) | ((low & 0xE0) >> 5))
							b555 := uint32(low & 0x1F)

							// Scale to 8 bits
							r = uint8((r555 * 255) / 31)
							g = uint8((g555 * 255) / 31)
							b = uint8((b555 * 255) / 31)
						} else {
							r, g, b = 0, 0, 0
						}

						// Set pixel color
						imgX := pixelX + px
						imgY := pixelY + py
						if imgX < gridWidth && imgY < gridHeight {
							img.Set(imgX, imgY, color.RGBA{r, g, b, 255})
						}
					}
				}

				// Draw grid line (right edge)
				for py := 0; py < tileSize; py++ {
					imgX := pixelX + tileSize
					imgY := pixelY + py
					if imgX < gridWidth && imgY < gridHeight {
						img.Set(imgX, imgY, color.RGBA{64, 64, 64, 255})
					}
				}

				// Draw grid line (bottom edge)
				for px := 0; px < tileSize; px++ {
					imgX := pixelX + px
					imgY := pixelY + tileSize
					if imgX < gridWidth && imgY < gridHeight {
						img.Set(imgX, imgY, color.RGBA{64, 64, 64, 255})
					}
				}
			}
		}

		return img
	})

	tileRaster.SetMinSize(fyne.NewSize(400, 400))
	tileScroll := container.NewScroll(tileRaster)
	tileScroll.SetMinSize(fyne.NewSize(400, 400))

	// Info label
	infoLabel := widget.NewLabel("")

	// Hex dump for selected tile (single tile view)
	hexDumpLabel := widget.NewLabel("")
	hexDumpLabel.Wrapping = fyne.TextWrapOff

	// Byte editor for manual editing (single tile view)
	byteAddrEntry := widget.NewEntry()
	byteAddrEntry.SetText("0x0000")
	byteAddrLabel := widget.NewLabel("VRAM Addr:")

	byteValueEntry := widget.NewEntry()
	byteValueEntry.SetText("00")
	byteValueLabel := widget.NewLabel("Value:")

	// Will be set after updateFunc is defined
	var writeByteBtn *widget.Button

	// Update function
	updateFunc := func() {
		if emu == nil || emu.PPU == nil {
			infoLabel.SetText("PPU not available")
			hexDumpLabel.SetText("")
			return
		}

		// Parse view mode
		viewMode = viewModeSelect.Selected

		// Parse palette
		var palette uint8
		if _, err := fmt.Sscanf(paletteEntry.Text, "%d", &palette); err == nil {
			if palette > 15 {
				palette = 15
			}
			currentPalette = palette
		}

		// Parse tile size
		if tileSizeSelect.Selected == "16x16" {
			currentTileSize = 16
		} else {
			currentTileSize = 8
		}

		// Parse tile index (for single tile view)
		var tileIndex uint8
		if _, err := fmt.Sscanf(tileSelectEntry.Text, "%d", &tileIndex); err == nil {
			currentTileIndex = tileIndex
		}

		// Parse tile offset (for grid view)
		var tileOffset uint8
		if _, err := fmt.Sscanf(tileOffsetEntry.Text, "%d", &tileOffset); err == nil {
			currentTileOffset = tileOffset
		}

		// Parse grid size
		var gridSize int
		if _, err := fmt.Sscanf(gridSizeSelect.Selected, "%d", &gridSize); err == nil {
			currentGridSize = gridSize
		}

		// Parse color index (for future click-to-edit feature)
		// var colorIndex uint8
		// if _, err := fmt.Sscanf(colorSelectEntry.Text, "%d", &colorIndex); err == nil {
		// 	if colorIndex > 15 {
		// 		colorIndex = 15
		// 	}
		// }

		// Update info
		if viewMode == "Single Tile" {
			infoLabel.SetText(
				"Palette: " + paletteEntry.Text + " | " +
					"Tile Size: " + tileSizeSelect.Selected + " | " +
					"Tile #: " + tileSelectEntry.Text + " | " +
					"Color: " + colorSelectEntry.Text + " | " +
					"Click pixel to edit")
		} else {
			infoLabel.SetText(
				"Palette: " + paletteEntry.Text + " | " +
					"Tile Size: " + tileSizeSelect.Selected + " | " +
					"Start Tile: " + tileOffsetEntry.Text + " | " +
					"Grid: " + gridSizeSelect.Selected + " tiles/row")
		}

		// Update hex dump for single tile view
		if viewMode == "Single Tile" {
			tileSize := currentTileSize
			bytesPerTile := tileSize * tileSize / 2
			tileDataOffset := uint16(currentTileIndex) * uint16(bytesPerTile)

			var hexText string
			hexText = fmt.Sprintf("Tile #%d (%dx%d) - VRAM 0x%04X-0x%04X:\n\n", currentTileIndex, tileSize, tileSize, tileDataOffset, tileDataOffset+uint16(bytesPerTile)-1)

			// Show hex dump (8 bytes per row)
			for row := 0; row < (bytesPerTile+7)/8; row++ {
				hexText += fmt.Sprintf("0x%04X: ", tileDataOffset+uint16(row*8))
				for col := 0; col < 8 && (row*8+col) < bytesPerTile; col++ {
					addr := tileDataOffset + uint16(row*8+col)
					if uint32(addr) < uint32(len(emu.PPU.VRAM)) {
						hexText += fmt.Sprintf("%02X ", emu.PPU.VRAM[addr])
					}
				}
				hexText += "\n"
			}

			hexDumpLabel.SetText(hexText)
		} else {
			hexDumpLabel.SetText("")
		}

		// Refresh raster
		tileRaster.Refresh()
	}

	// Input handlers
	viewModeSelect.OnChanged = func(value string) {
		updateFunc()
	}
	paletteEntry.OnChanged = func(text string) {
		updateFunc()
	}
	tileSizeSelect.OnChanged = func(value string) {
		updateFunc()
	}
	tileSelectEntry.OnChanged = func(text string) {
		updateFunc()
	}
	tileOffsetEntry.OnChanged = func(text string) {
		updateFunc()
	}
	gridSizeSelect.OnChanged = func(value string) {
		updateFunc()
	}
	colorSelectEntry.OnChanged = func(text string) {
		updateFunc()
	}

	// Create write byte button now that updateFunc is defined
	writeByteBtn = widget.NewButton("Write Byte", func() {
		if emu != nil && emu.PPU != nil {
			var addr uint16
			var value uint8
			if _, err := fmt.Sscanf(byteAddrEntry.Text, "0x%X", &addr); err == nil {
				if _, err := fmt.Sscanf(byteValueEntry.Text, "%X", &value); err == nil {
					if uint32(addr) < uint32(len(emu.PPU.VRAM)) {
						emu.PPU.VRAM[addr] = value
						tileRaster.Refresh()
						updateFunc() // Refresh hex dump
					}
				}
			}
		}
	})

	// Initial update
	updateFunc()

	// Controls layout
	controls := container.NewVBox(
		container.NewHBox(
			viewModeLabel,
			viewModeSelect,
			paletteLabel,
			paletteEntry,
			tileSizeLabel,
			tileSizeSelect,
		),
		container.NewHBox(
			tileSelectLabel,
			tileSelectEntry,
			colorSelectLabel,
			colorSelectEntry,
		),
		container.NewHBox(
			tileOffsetLabel,
			tileOffsetEntry,
			gridSizeLabel,
			gridSizeSelect,
		),
		infoLabel,
	)

	// Create main container with dynamic content
	mainContainer := container.NewVBox(
		widget.NewLabel("Tile Viewer"),
		controls,
		tileScroll,
	)

	// Update container when view mode changes
	viewModeSelect.OnChanged = func(value string) {
		updateFunc()
		// Rebuild container structure
		if value == "Single Tile" {
			hexScroll := container.NewScroll(hexDumpLabel)
			hexScroll.SetMinSize(fyne.NewSize(400, 100))
			byteEditor := container.NewHBox(
				byteAddrLabel,
				byteAddrEntry,
				byteValueLabel,
				byteValueEntry,
				writeByteBtn,
			)
			mainContainer.Objects = []fyne.CanvasObject{
				widget.NewLabel("Tile Viewer"),
				controls,
				tileScroll,
				widget.NewLabel("Hex Dump:"),
				hexScroll,
				widget.NewLabel("Manual Edit:"),
				byteEditor,
			}
		} else {
			mainContainer.Objects = []fyne.CanvasObject{
				widget.NewLabel("Tile Viewer"),
				controls,
				tileScroll,
			}
		}
		mainContainer.Refresh()
	}

	return mainContainer, updateFunc
}
