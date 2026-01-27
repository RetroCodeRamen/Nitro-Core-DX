package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// SpriteEditor is a tool for creating and editing sprites for Nitro-Core-DX
func main() {
	tileSize := flag.Int("size", 16, "Tile size (8 or 16)")
	outputPath := flag.String("output", "", "Output path for sprite data")
	flag.Parse()

	if *tileSize != 8 && *tileSize != 16 {
		fmt.Fprintf(os.Stderr, "Error: tile size must be 8 or 16\n")
		os.Exit(1)
	}

	myApp := app.New()
	window := myApp.NewWindow("Nitro-Core-DX Sprite Editor")
	window.Resize(fyne.NewSize(800, 600))

	// Create pixel grid for editing
	pixelSize := 20
	gridWidth := *tileSize * pixelSize
	gridHeight := *tileSize * pixelSize

	// Create canvas for sprite editing
	spriteCanvas := canvas.NewRaster(func(w, h int) image.Image {
		img := image.NewRGBA(image.Rect(0, 0, gridWidth, gridHeight))
		// Draw grid
		for y := 0; y < gridHeight; y++ {
			for x := 0; x < gridWidth; x++ {
				// Grid pattern
				if (x/pixelSize+y/pixelSize)%2 == 0 {
					img.Set(x, y, color.RGBA{R: 240, G: 240, B: 240, A: 255})
				} else {
					img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
				}
			}
		}
		return img
	})
	spriteCanvas.Resize(fyne.NewSize(float32(gridWidth), float32(gridHeight)))

	// Palette selector (16 colors)
	paletteLabel := widget.NewLabel("Palette:")
	paletteButtons := make([]*widget.Button, 16)
	paletteContainer := container.NewHBox()
	for i := 0; i < 16; i++ {
		idx := i
		btn := widget.NewButton("", func() {
			// Select palette color
			fmt.Printf("Selected palette color %d\n", idx)
		})
		btn.Resize(fyne.NewSize(30, 30))
		paletteButtons[i] = btn
		paletteContainer.Add(btn)
	}

	// Toolbar
	toolbar := container.NewHBox(
		widget.NewButton("Clear", func() {
			// Clear sprite
		}),
		widget.NewButton("Export", func() {
			// Export sprite data
			if *outputPath != "" {
				fmt.Printf("Exporting to %s\n", *outputPath)
			}
		}),
		widget.NewButton("Import", func() {
			// Import sprite data
		}),
	)

	// Main content
	content := container.NewBorder(
		toolbar,      // Top
		paletteLabel, // Bottom
		nil,          // Left
		nil,          // Right
		container.NewVBox(
			spriteCanvas,
			paletteContainer,
		),
	)

	window.SetContent(content)
	window.ShowAndRun()
}
