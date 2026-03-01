package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

const (
	spriteLabDefaultSize   = 24
	spriteLabMinSize       = 8
	spriteLabMaxSize       = 64
	spriteLabSizeStep      = 8
	spriteLabFormatV1      = "clxsprite-v1"
	spriteLabHistoryLimit  = 128
	spriteLabPaletteBanks  = 16
	spriteLabColorsPerBank = 16
	spriteLabPaletteCount  = spriteLabPaletteBanks * spriteLabColorsPerBank
	spriteLabTransparentIx = 0
)

type spriteLabAsset struct {
	Format      string   `json:"format"`
	Name        string   `json:"name"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	Pixels      []uint8  `json:"pixels"`
	PaletteBank int      `json:"palette_bank,omitempty"`
	Palettes    []uint16 `json:"palettes,omitempty"`
}

type spriteLabHistoryState struct {
	Width    int
	Height   int
	Pixels   []uint8
	Palettes []uint16
}

type spriteLabTool string

const (
	spriteLabToolPencil spriteLabTool = "Pencil"
	spriteLabToolErase  spriteLabTool = "Erase"
)

type spriteLabPaintOverlay struct {
	widget.BaseWidget
	onStrokeStart func()
	onStrokeEnd   func()
	onPaint       func(x, y int)
	onHover       func(x, y int)
	onHoverOut    func()
	strokeActive  bool
	gridW         int
	gridH         int
	cellPx        int
}

var spriteLabBasePalette = [16]color.NRGBA{
	{R: 0x00, G: 0x00, B: 0x00, A: 0xFF},
	{R: 0x20, G: 0x20, B: 0x20, A: 0xFF},
	{R: 0x3A, G: 0x6E, B: 0xA5, A: 0xFF},
	{R: 0x58, G: 0xB3, B: 0x72, A: 0xFF},
	{R: 0xD2, G: 0x53, B: 0x49, A: 0xFF},
	{R: 0xD8, G: 0x9F, B: 0x3A, A: 0xFF},
	{R: 0x6D, G: 0x56, B: 0xA5, A: 0xFF},
	{R: 0x8E, G: 0x8E, B: 0x8E, A: 0xFF},
	{R: 0xC7, G: 0xC7, B: 0xC7, A: 0xFF},
	{R: 0x4B, G: 0x86, B: 0xC5, A: 0xFF},
	{R: 0x71, G: 0xCF, B: 0x8D, A: 0xFF},
	{R: 0xF1, G: 0x74, B: 0x68, A: 0xFF},
	{R: 0xF4, G: 0xBE, B: 0x56, A: 0xFF},
	{R: 0x8A, G: 0x72, B: 0xCB, A: 0xFF},
	{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
	{R: 0x9A, G: 0xD7, B: 0xEA, A: 0xFF},
}

func (s *devKitState) buildSpriteLabPane() fyne.CanvasObject {
	spriteW := spriteLabDefaultSize
	spriteH := spriteLabDefaultSize
	pixels := make([]uint8, spriteW*spriteH)
	palettes := defaultSpriteLabPaletteData()

	selectedColor := 1
	selectedBank := 0
	currentTool := spriteLabToolPencil
	showGrid := true
	mirrorX := false
	transparentZero := true
	hoverX, hoverY := -1, -1

	history := make([]spriteLabHistoryState, 0, 16)
	historyIndex := -1
	strokeDirty := false
	suppressSizeSelect := false

	snapshot := func() spriteLabHistoryState {
		pixelCopy := make([]uint8, len(pixels))
		copy(pixelCopy, pixels)
		paletteCopy := make([]uint16, len(palettes))
		copy(paletteCopy, palettes)
		return spriteLabHistoryState{
			Width:    spriteW,
			Height:   spriteH,
			Pixels:   pixelCopy,
			Palettes: paletteCopy,
		}
	}

	commitHistory := func() {
		if historyIndex < len(history)-1 {
			history = history[:historyIndex+1]
		}
		if len(history) >= spriteLabHistoryLimit {
			copy(history, history[1:])
			history = history[:len(history)-1]
			if historyIndex > 0 {
				historyIndex--
			}
		}
		history = append(history, snapshot())
		historyIndex = len(history) - 1
	}

	selectedPaletteOffset := func() int {
		return selectedBank*spriteLabColorsPerBank + selectedColor
	}

	setPixel := func(x, y int, value uint8) bool {
		if x < 0 || x >= spriteW || y < 0 || y >= spriteH {
			return false
		}
		idx := y*spriteW + x
		value &= 0x0F
		if pixels[idx] == value {
			return false
		}
		pixels[idx] = value
		return true
	}

	nonZeroPixels := func() int {
		count := 0
		for _, px := range pixels {
			if px != 0 {
				count++
			}
		}
		return count
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetText("SpriteTile")

	statusLabel := widget.NewLabel("Sprite Lab ready")
	coordLabel := widget.NewLabel("Cursor: --")
	selectedLabel := widget.NewLabel("Color: 1")
	selectedValueLabel := widget.NewLabel("RGB555: 0x0000")
	statsLabel := widget.NewLabel("Active pixels: 0/0")
	historyLabel := widget.NewLabel("History: 1/1")
	sizeLabel := widget.NewLabel("Size: 24x24")

	rEntry := widget.NewEntry()
	gEntry := widget.NewEntry()
	bEntry := widget.NewEntry()
	hexColorEntry := widget.NewEntry()

	hexPreview := newReadOnlyTextArea()
	hexPreview.SetMinRowsVisible(8)

	editorImage := canvas.NewImageFromImage(image.NewNRGBA(image.Rect(0, 0, 1, 1)))
	editorImage.FillMode = canvas.ImageFillStretch
	editorImage.ScaleMode = canvas.ImageScalePixels
	editorImage.SetMinSize(spriteLabEditorDisplaySize(spriteW, spriteH))

	previewImage := canvas.NewImageFromImage(image.NewNRGBA(image.Rect(0, 0, 1, 1)))
	previewImage.FillMode = canvas.ImageFillStretch
	previewImage.ScaleMode = canvas.ImageScalePixels
	previewImage.SetMinSize(spriteLabPreviewDisplaySize(spriteW, spriteH))

	paletteButtons := make([]*widget.Button, spriteLabColorsPerBank)
	paletteChips := make([]*canvas.Image, spriteLabColorsPerBank)
	selectedColorChip := canvas.NewImageFromImage(renderSpriteLabPaletteChipImage(color.NRGBA{A: 0xFF}, false))
	selectedColorChip.SetMinSize(fyne.NewSize(30, 30))
	selectedColorChip.FillMode = canvas.ImageFillStretch
	selectedColorChip.ScaleMode = canvas.ImageScalePixels

	undoButton := widget.NewButton("Undo", nil)
	redoButton := widget.NewButton("Redo", nil)
	resizeSelect := widget.NewSelect(nil, nil)
	var canvasOverlay *spriteLabPaintOverlay

	setSizeSelection := func() {
		label := formatSpriteSizeLabel(spriteW, spriteH)
		suppressSizeSelect = true
		resizeSelect.SetSelected(label)
		suppressSizeSelect = false
	}

	updateCanvasSizes := func() {
		cellPx := spriteLabCellPx(spriteW, spriteH, spriteLabEditorMaxPx)
		editorImage.SetMinSize(spriteLabEditorDisplaySize(spriteW, spriteH))
		previewImage.SetMinSize(spriteLabPreviewDisplaySize(spriteW, spriteH))
		sizeLabel.SetText("Size: " + formatSpriteSizeLabel(spriteW, spriteH))
		if canvasOverlay != nil {
			canvasOverlay.SetGrid(spriteW, spriteH, cellPx)
		}
	}

	restoreHistory := func(index int) bool {
		if index < 0 || index >= len(history) {
			return false
		}
		state := history[index]
		spriteW = state.Width
		spriteH = state.Height
		pixels = make([]uint8, len(state.Pixels))
		copy(pixels, state.Pixels)
		copy(palettes, state.Palettes)
		historyIndex = index
		setSizeSelection()
		updateCanvasSizes()
		return true
	}

	resizeSprite := func(newW, newH int) bool {
		if !isValidSpriteDimension(newW) || !isValidSpriteDimension(newH) {
			return false
		}
		if newW == spriteW && newH == spriteH {
			return false
		}
		next := make([]uint8, newW*newH)
		copyW := minInt(spriteW, newW)
		copyH := minInt(spriteH, newH)
		for y := 0; y < copyH; y++ {
			for x := 0; x < copyW; x++ {
				next[y*newW+x] = pixels[y*spriteW+x]
			}
		}
		spriteW = newW
		spriteH = newH
		pixels = next
		updateCanvasSizes()
		setSizeSelection()
		return true
	}

	setColorEditors := func() {
		idx := selectedPaletteOffset()
		val := palettes[idx]
		r, g, b := decodeRGB555(val)
		rEntry.SetText(strconv.Itoa(int(r)))
		gEntry.SetText(strconv.Itoa(int(g)))
		bEntry.SetText(strconv.Itoa(int(b)))
		hexColorEntry.SetText(fmt.Sprintf("%04X", val))
		if transparentZero && selectedColor == spriteLabTransparentIx {
			selectedValueLabel.SetText(fmt.Sprintf("RGB555: 0x%04X (transparent index)", val))
		} else {
			selectedValueLabel.SetText(fmt.Sprintf("RGB555: 0x%04X", val))
		}
		selectedColorChip.Image = renderSpriteLabPaletteChipImage(
			rgb555ToNRGBA(val),
			transparentZero && selectedColor == spriteLabTransparentIx,
		)
		selectedColorChip.Refresh()
	}

	refreshPaletteButtons := func() {
		for i := 0; i < spriteLabColorsPerBank; i++ {
			btn := paletteButtons[i]
			chip := paletteChips[i]
			if btn == nil || chip == nil {
				continue
			}
			idx := selectedBank*spriteLabColorsPerBank + i
			chip.Image = renderSpriteLabPaletteChipImage(
				rgb555ToNRGBA(palettes[idx]),
				transparentZero && i == spriteLabTransparentIx,
			)
			chip.Refresh()
			if i == selectedColor {
				btn.Importance = widget.HighImportance
				btn.SetText(fmt.Sprintf("%X*", i))
			} else {
				btn.Importance = widget.MediumImportance
				btn.SetText(fmt.Sprintf("%X", i))
			}
			btn.Refresh()
		}
	}

	refreshHistoryButtons := func() {
		historyLabel.SetText(fmt.Sprintf("History: %d/%d", historyIndex+1, len(history)))
		if historyIndex > 0 {
			undoButton.Enable()
		} else {
			undoButton.Disable()
		}
		if historyIndex >= 0 && historyIndex < len(history)-1 {
			redoButton.Enable()
		} else {
			redoButton.Disable()
		}
	}

	refreshEditorOnly := func() {
		cellPx := spriteLabCellPx(spriteW, spriteH, spriteLabEditorMaxPx)
		editorImage.Image = renderSpriteLabImage(
			pixels, palettes, selectedBank, spriteW, spriteH, cellPx, hoverX, hoverY, showGrid, transparentZero,
		)
		editorImage.Refresh()
	}

	refreshVisuals := func() {
		refreshEditorOnly()
		previewCellPx := spriteLabCellPx(spriteW, spriteH, spriteLabPreviewMaxPx)
		previewImage.Image = renderSpriteLabImage(
			pixels, palettes, selectedBank, spriteW, spriteH, previewCellPx, -1, -1, false, transparentZero,
		)
		previewImage.Refresh()
		refreshSpriteLabHexPreview(hexPreview, pixels, spriteW, spriteH)
		selectedLabel.SetText(fmt.Sprintf("Color: %X (Bank %d)", selectedColor, selectedBank))
		statsLabel.SetText(fmt.Sprintf("Active pixels: %d/%d", nonZeroPixels(), spriteW*spriteH))
		refreshPaletteButtons()
		refreshHistoryButtons()
		setColorEditors()
	}

	applyFill := func(value uint8) bool {
		changed := false
		for i := range pixels {
			if pixels[i] != (value & 0x0F) {
				pixels[i] = value & 0x0F
				changed = true
			}
		}
		return changed
	}

	setSelectedPaletteColor := func(newColor uint16, note string) {
		idx := selectedPaletteOffset()
		if palettes[idx] == newColor {
			statusLabel.SetText("Selected palette color unchanged")
			return
		}
		palettes[idx] = newColor
		commitHistory()
		refreshVisuals()
		statusLabel.SetText(note)
	}

	paintAt := func(x, y int) {
		value := uint8(selectedColor)
		if currentTool == spriteLabToolErase {
			value = 0
		}
		changed := setPixel(x, y, value)
		if mirrorX {
			mx := (spriteW - 1) - x
			if mx != x {
				if setPixel(mx, y, value) {
					changed = true
				}
			}
		}
		if changed {
			strokeDirty = true
			refreshEditorOnly()
		}
	}

	canvasOverlay = newSpriteLabPaintOverlay(
		spriteW,
		spriteH,
		spriteLabCellPx(spriteW, spriteH, spriteLabEditorMaxPx),
		func() {
			strokeDirty = false
		},
		func() {
			if strokeDirty {
				commitHistory()
				refreshVisuals()
				statusLabel.SetText("Paint stroke applied")
			}
		},
		func(x, y int) {
			paintAt(x, y)
			coordLabel.SetText(fmt.Sprintf("Cursor: (%d,%d)", x, y))
		},
		func(x, y int) {
			hoverX, hoverY = x, y
			coordLabel.SetText(fmt.Sprintf("Cursor: (%d,%d)", x, y))
			refreshEditorOnly()
		},
		func() {
			hoverX, hoverY = -1, -1
			coordLabel.SetText("Cursor: --")
			refreshEditorOnly()
		},
	)

	sizeOptions := spriteLabSizeOptions()
	resizeSelect.Options = make([]string, 0, len(sizeOptions))
	for _, size := range sizeOptions {
		resizeSelect.Options = append(resizeSelect.Options, formatSpriteSizeLabel(size, size))
	}
	resizeSelect.OnChanged = func(v string) {
		if suppressSizeSelect {
			return
		}
		w, h, err := parseSpriteSizeSelection(v)
		if err != nil {
			statusLabel.SetText("Invalid size selection")
			setSizeSelection()
			return
		}
		if !resizeSprite(w, h) {
			statusLabel.SetText("Sprite size unchanged")
			return
		}
		commitHistory()
		refreshVisuals()
		statusLabel.SetText("Resized sprite canvas to " + formatSpriteSizeLabel(spriteW, spriteH))
	}
	setSizeSelection()

	toolGroup := widget.NewRadioGroup([]string{string(spriteLabToolPencil), string(spriteLabToolErase)}, func(v string) {
		if v == string(spriteLabToolErase) {
			currentTool = spriteLabToolErase
			statusLabel.SetText("Tool: Erase")
			return
		}
		currentTool = spriteLabToolPencil
		statusLabel.SetText("Tool: Pencil")
	})
	toolGroup.Horizontal = true
	toolGroup.SetSelected(string(spriteLabToolPencil))

	mirrorCheck := widget.NewCheck("Mirror X", func(v bool) {
		mirrorX = v
		if v {
			statusLabel.SetText("Mirror painting enabled")
		} else {
			statusLabel.SetText("Mirror painting disabled")
		}
	})

	gridCheck := widget.NewCheck("Show Grid", func(v bool) {
		showGrid = v
		refreshEditorOnly()
	})
	gridCheck.SetChecked(true)

	transparencyCheck := widget.NewCheck("Index 0 Transparent", func(v bool) {
		transparentZero = v
		if v {
			statusLabel.SetText("Transparency enabled for color index 0")
		} else {
			statusLabel.SetText("Transparency disabled in Sprite Lab preview")
		}
		refreshVisuals()
	})
	transparencyCheck.SetChecked(true)

	paletteBankOptions := make([]string, spriteLabPaletteBanks)
	for i := 0; i < spriteLabPaletteBanks; i++ {
		paletteBankOptions[i] = fmt.Sprintf("Bank %d", i)
	}
	paletteBankSelect := widget.NewSelect(paletteBankOptions, func(v string) {
		idx, err := parsePaletteBankSelection(v)
		if err != nil {
			statusLabel.SetText("Invalid palette bank selection")
			return
		}
		selectedBank = idx
		refreshVisuals()
		statusLabel.SetText(fmt.Sprintf("Palette bank set to %d", selectedBank))
	})
	paletteBankSelect.SetSelected(paletteBankOptions[selectedBank])

	paletteGrid := container.NewGridWithColumns(4)
	for i := 0; i < spriteLabColorsPerBank; i++ {
		idx := i
		chip := canvas.NewImageFromImage(renderSpriteLabPaletteChipImage(color.NRGBA{A: 0xFF}, false))
		chip.SetMinSize(fyne.NewSize(16, 16))
		chip.FillMode = canvas.ImageFillStretch
		chip.ScaleMode = canvas.ImageScalePixels
		btn := widget.NewButton("", func() {
			selectedColor = idx
			refreshVisuals()
			statusLabel.SetText(fmt.Sprintf("Selected color %X in bank %d", selectedColor, selectedBank))
		})
		paletteButtons[i] = btn
		paletteChips[i] = chip
		paletteGrid.Add(container.NewBorder(nil, nil, chip, nil, btn))
	}

	undoButton.OnTapped = func() {
		if historyIndex <= 0 {
			return
		}
		if restoreHistory(historyIndex - 1) {
			statusLabel.SetText("Undo")
			refreshVisuals()
		}
	}

	redoButton.OnTapped = func() {
		if historyIndex < 0 || historyIndex >= len(history)-1 {
			return
		}
		if restoreHistory(historyIndex + 1) {
			statusLabel.SetText("Redo")
			refreshVisuals()
		}
	}

	clearButton := widget.NewButton("Clear", func() {
		if !applyFill(0) {
			statusLabel.SetText("Sprite already clear")
			return
		}
		commitHistory()
		refreshVisuals()
		statusLabel.SetText("Sprite cleared")
	})

	fillButton := widget.NewButton("Fill", func() {
		if !applyFill(uint8(selectedColor)) {
			statusLabel.SetText("Sprite already filled with selected color")
			return
		}
		commitHistory()
		refreshVisuals()
		statusLabel.SetText(fmt.Sprintf("Filled sprite with color %X", selectedColor))
	})

	applyRGBButton := widget.NewButton("Apply RGB", func() {
		r, err := parseSpriteLabChannel(rEntry.Text)
		if err != nil {
			statusLabel.SetText("Invalid R value (0-31)")
			return
		}
		g, err := parseSpriteLabChannel(gEntry.Text)
		if err != nil {
			statusLabel.SetText("Invalid G value (0-31)")
			return
		}
		b, err := parseSpriteLabChannel(bEntry.Text)
		if err != nil {
			statusLabel.SetText("Invalid B value (0-31)")
			return
		}
		setSelectedPaletteColor(encodeRGB555(r, g, b), fmt.Sprintf("Updated bank %d color %X from RGB", selectedBank, selectedColor))
	})

	applyHexButton := widget.NewButton("Apply Hex", func() {
		v, err := parseSpriteLabRGB555Hex(hexColorEntry.Text)
		if err != nil {
			statusLabel.SetText("Invalid RGB555 hex (use 0x1234 or 1234)")
			return
		}
		setSelectedPaletteColor(v, fmt.Sprintf("Updated bank %d color %X from hex", selectedBank, selectedColor))
	})

	copyHexButton := widget.NewButton("Copy Tile Hex", func() {
		if s.window != nil && s.window.Clipboard() != nil {
			s.window.Clipboard().SetContent(hexPreview.Text)
			statusLabel.SetText("Packed tile hex copied")
		}
	})

	saveButton := widget.NewButton("Export .clxsprite", func() {
		fd := dialog.NewFileSave(func(wc fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, s.window)
				return
			}
			if wc == nil {
				return
			}
			defer wc.Close()

			data, marshalErr := marshalSpriteLabAsset(nameEntry.Text, pixels, spriteW, spriteH, selectedBank, palettes)
			if marshalErr != nil {
				dialog.ShowError(marshalErr, s.window)
				return
			}
			if _, writeErr := wc.Write(data); writeErr != nil {
				dialog.ShowError(writeErr, s.window)
				return
			}

			path := uriPath(wc.URI())
			if path != "" {
				s.settings.LastSourceDir = filepath.Dir(path)
				s.persistSettings()
			}
			statusLabel.SetText("Exported sprite asset with palette banks")
		}, s.window)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".clxsprite", ".json"}))
		if loc := dialogListableForDir(s.settings.LastSourceDir); loc != nil {
			fd.SetLocation(loc)
		}
		fd.SetFileName(sanitizeSpriteLabName(nameEntry.Text) + ".clxsprite")
		fd.Show()
	})

	loadButton := widget.NewButton("Import .clxsprite", func() {
		fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, s.window)
				return
			}
			if rc == nil {
				return
			}
			defer rc.Close()

			data, readErr := io.ReadAll(rc)
			if readErr != nil {
				dialog.ShowError(readErr, s.window)
				return
			}

			asset, parseErr := unmarshalSpriteLabAsset(data)
			if parseErr != nil {
				dialog.ShowError(parseErr, s.window)
				return
			}

			spriteW = asset.Width
			spriteH = asset.Height
			pixels = make([]uint8, len(asset.Pixels))
			copy(pixels, asset.Pixels)
			copy(palettes, asset.Palettes)
			selectedBank = asset.PaletteBank
			if selectedBank < 0 || selectedBank >= spriteLabPaletteBanks {
				selectedBank = 0
			}
			paletteBankSelect.SetSelected(fmt.Sprintf("Bank %d", selectedBank))
			setSizeSelection()
			updateCanvasSizes()
			nameEntry.SetText(asset.Name)
			commitHistory()
			refreshVisuals()

			path := uriPath(rc.URI())
			if path != "" {
				s.settings.LastSourceDir = filepath.Dir(path)
				s.persistSettings()
			}
			statusLabel.SetText("Imported sprite asset + palette banks")
		}, s.window)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".clxsprite", ".json"}))
		if loc := dialogListableForDir(s.settings.LastSourceDir); loc != nil {
			fd.SetLocation(loc)
		}
		fd.Show()
	})

	includePaletteCheck := widget.NewCheck("Include palette setup in code snippet", nil)
	includePaletteCheck.SetChecked(true)

	insertButton := widget.NewButton("Insert CoreLX Asset", func() {
		name := sanitizeSpriteLabName(nameEntry.Text)
		assetSnippet, err := spriteLabCoreLXAssetSnippet(name, pixels, spriteW, spriteH)
		if err != nil {
			dialog.ShowError(err, s.window)
			return
		}
		next := s.sourceEntry.Text
		if strings.TrimSpace(next) != "" {
			next = strings.TrimRight(next, "\n") + "\n\n"
		}
		next += assetSnippet
		if includePaletteCheck.Checked {
			paletteSnippet, pErr := spriteLabPaletteInitSnippet(selectedBank, palettes)
			if pErr != nil {
				dialog.ShowError(pErr, s.window)
				return
			}
			next += "\n\n" + paletteSnippet
		}
		next += "\n"
		s.setSourceContent(next, true, false)
		if s.workbenchTabs != nil {
			s.workbenchTabs.SelectIndex(0)
		}
		s.setStatus("Inserted sprite + palette snippet")
		s.appendBuildOutput("Sprite Lab inserted asset: " + name)
		statusLabel.SetText("Inserted CoreLX asset snippet")
	})

	canvasPanel := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Sprite Canvas"),
			container.NewHBox(widget.NewLabel("Canvas Size"), resizeSelect, sizeLabel),
			toolGroup,
			container.NewHBox(mirrorCheck, gridCheck, transparencyCheck),
		),
		container.NewHBox(coordLabel, widget.NewSeparator(), selectedLabel),
		nil,
		nil,
		container.NewCenter(container.NewStack(editorImage, canvasOverlay)),
	)

	colorEditorPanel := container.NewVBox(
		widget.NewLabel("Selected Color Editor (0-31 channels)"),
		container.NewHBox(
			widget.NewLabel("R"),
			rEntry,
			widget.NewLabel("G"),
			gEntry,
			widget.NewLabel("B"),
			bEntry,
		),
		container.NewHBox(
			widget.NewLabel("Hex"),
			hexColorEntry,
			applyHexButton,
		),
		container.NewHBox(applyRGBButton, selectedColorChip),
		selectedValueLabel,
	)

	rightPanel := container.NewVBox(
		widget.NewLabel("Asset Name"),
		nameEntry,
		widget.NewSeparator(),
		widget.NewLabel("Palette Bank"),
		paletteBankSelect,
		widget.NewLabel("Palette Colors (16 colors in current bank)"),
		paletteGrid,
		colorEditorPanel,
		container.NewGridWithColumns(2, undoButton, redoButton),
		historyLabel,
		container.NewGridWithColumns(2, clearButton, fillButton),
		container.NewGridWithColumns(2, loadButton, saveButton),
		includePaletteCheck,
		insertButton,
		widget.NewSeparator(),
		widget.NewLabel("Preview"),
		previewImage,
		statsLabel,
		widget.NewSeparator(),
		widget.NewLabel("Packed 4bpp bytes"),
		hexPreview,
		copyHexButton,
	)

	split := container.NewHSplit(canvasPanel, container.NewScroll(rightPanel))
	split.Offset = 0.58

	commitHistory()
	refreshVisuals()

	return container.NewBorder(nil, statusLabel, nil, nil, split)
}

func newSpriteLabPaintOverlay(
	gridW, gridH, cellPx int,
	onStrokeStart func(),
	onStrokeEnd func(),
	onPaint func(x, y int),
	onHover func(x, y int),
	onHoverOut func(),
) *spriteLabPaintOverlay {
	o := &spriteLabPaintOverlay{
		onStrokeStart: onStrokeStart,
		onStrokeEnd:   onStrokeEnd,
		onPaint:       onPaint,
		onHover:       onHover,
		onHoverOut:    onHoverOut,
		gridW:         gridW,
		gridH:         gridH,
		cellPx:        cellPx,
	}
	o.ExtendBaseWidget(o)
	return o
}

func (o *spriteLabPaintOverlay) SetGrid(w, h, cellPx int) {
	o.gridW = w
	o.gridH = h
	o.cellPx = cellPx
}

func (o *spriteLabPaintOverlay) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (o *spriteLabPaintOverlay) beginStroke() {
	if o.strokeActive {
		return
	}
	o.strokeActive = true
	if o.onStrokeStart != nil {
		o.onStrokeStart()
	}
}

func (o *spriteLabPaintOverlay) endStroke() {
	if !o.strokeActive {
		return
	}
	o.strokeActive = false
	if o.onStrokeEnd != nil {
		o.onStrokeEnd()
	}
}

func (o *spriteLabPaintOverlay) cellAt(pos fyne.Position) (int, int, bool) {
	sz := o.Size()
	if sz.Width <= 0 || sz.Height <= 0 || o.gridW <= 0 || o.gridH <= 0 {
		return 0, 0, false
	}
	if pos.X < 0 || pos.Y < 0 || pos.X >= sz.Width || pos.Y >= sz.Height {
		return 0, 0, false
	}
	x := int((pos.X * float32(o.gridW)) / sz.Width)
	y := int((pos.Y * float32(o.gridH)) / sz.Height)
	if x < 0 || x >= o.gridW || y < 0 || y >= o.gridH {
		return 0, 0, false
	}
	return x, y, true
}

func (o *spriteLabPaintOverlay) paintAt(pos fyne.Position) {
	x, y, ok := o.cellAt(pos)
	if !ok {
		return
	}
	if o.onPaint != nil {
		o.onPaint(x, y)
	}
}

func (o *spriteLabPaintOverlay) hoverAt(pos fyne.Position) {
	x, y, ok := o.cellAt(pos)
	if !ok {
		if o.onHoverOut != nil {
			o.onHoverOut()
		}
		return
	}
	if o.onHover != nil {
		o.onHover(x, y)
	}
}

func (o *spriteLabPaintOverlay) Tapped(ev *fyne.PointEvent) {
	o.beginStroke()
	o.paintAt(ev.Position)
	o.endStroke()
}

func (o *spriteLabPaintOverlay) TappedSecondary(*fyne.PointEvent) {}

func (o *spriteLabPaintOverlay) Dragged(ev *fyne.DragEvent) {
	o.beginStroke()
	o.paintAt(ev.Position)
}

func (o *spriteLabPaintOverlay) DragEnd() {
	o.endStroke()
}

func (o *spriteLabPaintOverlay) MouseIn(ev *desktop.MouseEvent) {
	o.hoverAt(ev.Position)
}

func (o *spriteLabPaintOverlay) MouseMoved(ev *desktop.MouseEvent) {
	o.hoverAt(ev.Position)
}

func (o *spriteLabPaintOverlay) MouseOut() {
	if o.onHoverOut != nil {
		o.onHoverOut()
	}
}

func renderSpriteLabImage(
	pixels []uint8,
	palettes []uint16,
	paletteBank int,
	spriteW, spriteH int,
	cellPx int,
	hoverX, hoverY int,
	drawGrid bool,
	transparentZero bool,
) image.Image {
	if cellPx < 1 {
		cellPx = 1
	}
	if spriteW < 1 {
		spriteW = 1
	}
	if spriteH < 1 {
		spriteH = 1
	}
	w := spriteW * cellPx
	h := spriteH * cellPx
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	gridColor := color.NRGBA{R: 0x2E, G: 0x33, B: 0x3E, A: 0xFF}
	hoverColor := color.NRGBA{R: 0xF9, G: 0xFB, B: 0xFF, A: 0xFF}

	gridThick := 0
	if drawGrid {
		switch {
		case cellPx >= 16:
			gridThick = 2
		case cellPx >= 4:
			gridThick = 1
		}
	}

	hoverBorder := gridThick + 1
	if hoverBorder < 2 {
		hoverBorder = 2
	}

	if paletteBank < 0 || paletteBank >= spriteLabPaletteBanks {
		paletteBank = 0
	}
	bankBase := paletteBank * spriteLabColorsPerBank

	for py := 0; py < h; py++ {
		sy := py / cellPx
		by := py % cellPx
		for px := 0; px < w; px++ {
			sx := px / cellPx
			bx := px % cellPx
			pixelIdx := sy*spriteW + sx
			if pixelIdx < 0 || pixelIdx >= len(pixels) {
				continue
			}
			colorIdx := int(pixels[pixelIdx] & 0x0F)
			paletteIdx := bankBase + colorIdx
			clr := color.NRGBA{A: 0xFF}
			if transparentZero && colorIdx == spriteLabTransparentIx {
				block := cellPx / 4
				if block < 2 {
					block = 2
				}
				clr = spriteLabCheckerColor(px, py, block)
			} else if paletteIdx >= 0 && paletteIdx < len(palettes) {
				clr = rgb555ToNRGBA(palettes[paletteIdx])
			}

			if gridThick > 0 && (bx < gridThick || by < gridThick) {
				clr = gridColor
			}

			if sx == hoverX && sy == hoverY {
				if bx < hoverBorder || by < hoverBorder || bx >= cellPx-hoverBorder || by >= cellPx-hoverBorder {
					clr = hoverColor
				}
			}

			off := py*img.Stride + px*4
			img.Pix[off] = clr.R
			img.Pix[off+1] = clr.G
			img.Pix[off+2] = clr.B
			img.Pix[off+3] = clr.A
		}
	}

	return img
}

func renderSpriteLabPaletteChipImage(base color.NRGBA, transparent bool) image.Image {
	const chipSize = 18
	img := image.NewNRGBA(image.Rect(0, 0, chipSize, chipSize))
	for y := 0; y < chipSize; y++ {
		for x := 0; x < chipSize; x++ {
			c := base
			if transparent {
				c = spriteLabCheckerColor(x, y, 4)
			}
			off := y*img.Stride + x*4
			img.Pix[off] = c.R
			img.Pix[off+1] = c.G
			img.Pix[off+2] = c.B
			img.Pix[off+3] = c.A
		}
	}
	return img
}

func spriteLabCheckerColor(x, y, block int) color.NRGBA {
	if block < 1 {
		block = 1
	}
	a := color.NRGBA{R: 0xCB, G: 0xCF, B: 0xD8, A: 0xFF}
	b := color.NRGBA{R: 0x8D, G: 0x95, B: 0xA5, A: 0xFF}
	if ((x/block)+(y/block))%2 == 0 {
		return a
	}
	return b
}

func refreshSpriteLabHexPreview(entry *widget.Entry, pixels []uint8, width, height int) {
	packed, err := packSpriteLabPixels(pixels, width, height)
	if err != nil {
		entry.Enable()
		entry.SetText("error: " + err.Error())
		entry.Disable()
		return
	}
	var sb strings.Builder
	rowBytes := width / 2
	for row := 0; row < height; row++ {
		if row > 0 {
			sb.WriteByte('\n')
		}
		base := row * rowBytes
		for col := 0; col < rowBytes; col++ {
			if col > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(fmt.Sprintf("%02X", packed[base+col]))
		}
	}
	entry.Enable()
	entry.SetText(sb.String())
	entry.Disable()
}

func marshalSpriteLabAsset(name string, pixels []uint8, width, height int, paletteBank int, palettes []uint16) ([]byte, error) {
	a := spriteLabAsset{
		Format:      spriteLabFormatV1,
		Name:        sanitizeSpriteLabName(name),
		Width:       width,
		Height:      height,
		Pixels:      append([]uint8(nil), pixels...),
		PaletteBank: paletteBank,
		Palettes:    append([]uint16(nil), palettes...),
	}
	if err := validateSpriteLabAsset(a); err != nil {
		return nil, err
	}
	return json.MarshalIndent(a, "", "  ")
}

func unmarshalSpriteLabAsset(data []byte) (spriteLabAsset, error) {
	var a spriteLabAsset
	if err := json.Unmarshal(data, &a); err != nil {
		return spriteLabAsset{}, fmt.Errorf("invalid sprite asset JSON: %w", err)
	}
	if a.Format == "" {
		a.Format = spriteLabFormatV1
	}
	if a.Name == "" {
		a.Name = "SpriteTile"
	}
	a.Name = sanitizeSpriteLabName(a.Name)

	if a.Width == 0 {
		a.Width = spriteLabDefaultSize
	}
	if a.Height == 0 {
		a.Height = spriteLabDefaultSize
	}
	if len(a.Palettes) == 0 {
		a.Palettes = defaultSpriteLabPaletteData()
	}
	if a.PaletteBank < 0 || a.PaletteBank >= spriteLabPaletteBanks {
		a.PaletteBank = 0
	}

	if err := validateSpriteLabAsset(a); err != nil {
		return spriteLabAsset{}, err
	}
	return a, nil
}

func validateSpriteLabAsset(a spriteLabAsset) error {
	if a.Format != spriteLabFormatV1 {
		return fmt.Errorf("unsupported sprite asset format: %q", a.Format)
	}
	if !isValidSpriteDimension(a.Width) || !isValidSpriteDimension(a.Height) {
		return fmt.Errorf("sprite asset dimensions must be %d-%d in steps of %d", spriteLabMinSize, spriteLabMaxSize, spriteLabSizeStep)
	}
	expectedPixels := a.Width * a.Height
	if len(a.Pixels) != expectedPixels {
		return fmt.Errorf("sprite asset pixel length must be %d", expectedPixels)
	}
	for i, px := range a.Pixels {
		if px > 0x0F {
			return fmt.Errorf("sprite pixel %d out of range: %d", i, px)
		}
	}
	if a.PaletteBank < 0 || a.PaletteBank >= spriteLabPaletteBanks {
		return fmt.Errorf("palette bank out of range: %d", a.PaletteBank)
	}
	if len(a.Palettes) != spriteLabPaletteCount {
		return fmt.Errorf("palette data length must be %d", spriteLabPaletteCount)
	}
	return nil
}

func spriteLabCoreLXAssetSnippet(name string, pixels []uint8, width, height int) (string, error) {
	packed, err := packSpriteLabPixels(pixels, width, height)
	if err != nil {
		return "", err
	}
	assetType := spriteLabAssetTypeForDimensions(width, height)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("// Sprite Lab asset (%dx%d, 4bpp)\n", width, height))
	sb.WriteString("asset ")
	sb.WriteString(sanitizeSpriteLabName(name))
	sb.WriteString(": ")
	sb.WriteString(assetType)
	sb.WriteString(" hex\n")

	rowBytes := width / 2
	for row := 0; row < height; row++ {
		sb.WriteString("    ")
		base := row * rowBytes
		for col := 0; col < rowBytes; col++ {
			if col > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(fmt.Sprintf("%02X", packed[base+col]))
		}
		if row < height-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String(), nil
}

func spriteLabAssetTypeForDimensions(width, height int) string {
	if width == 8 && height == 8 {
		return "tiles8"
	}
	if width == 16 && height == 16 {
		return "tiles16"
	}
	return "tileset"
}

func spriteLabPaletteInitSnippet(bank int, palettes []uint16) (string, error) {
	if bank < 0 || bank >= spriteLabPaletteBanks {
		return "", fmt.Errorf("palette bank out of range: %d", bank)
	}
	if len(palettes) != spriteLabPaletteCount {
		return "", fmt.Errorf("palette data length must be %d", spriteLabPaletteCount)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("// Sprite Lab palette bank %d\n", bank))
	base := bank * spriteLabColorsPerBank
	for i := 0; i < spriteLabColorsPerBank; i++ {
		sb.WriteString(fmt.Sprintf("gfx.set_palette(%d, %d, 0x%04X)", bank, i, palettes[base+i]))
		if i < spriteLabColorsPerBank-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String(), nil
}

func packSpriteLabPixels(pixels []uint8, width, height int) ([]byte, error) {
	if !isValidSpriteDimension(width) || !isValidSpriteDimension(height) {
		return nil, fmt.Errorf("invalid dimensions %dx%d", width, height)
	}
	expected := width * height
	if len(pixels) != expected {
		return nil, fmt.Errorf("expected %d pixels, got %d", expected, len(pixels))
	}
	if width%2 != 0 {
		return nil, fmt.Errorf("width must be even for 4bpp packing")
	}
	out := make([]byte, 0, expected/2)
	for i := 0; i < len(pixels); i += 2 {
		lo := pixels[i]
		hi := pixels[i+1]
		if lo > 0x0F || hi > 0x0F {
			return nil, fmt.Errorf("pixel value out of range at pair %d", i/2)
		}
		out = append(out, (hi<<4)|lo)
	}
	return out, nil
}

func sanitizeSpriteLabName(raw string) string {
	var sb strings.Builder
	raw = strings.TrimSpace(raw)
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			sb.WriteByte(ch)
			continue
		}
		sb.WriteByte('_')
	}
	out := sb.String()
	if out == "" {
		out = "SpriteTile"
	}
	first := out[0]
	if first >= '0' && first <= '9' {
		out = "A_" + out
	}
	return out
}

func defaultSpriteLabPaletteData() []uint16 {
	out := make([]uint16, spriteLabPaletteCount)
	for bank := 0; bank < spriteLabPaletteBanks; bank++ {
		rBoost := uint8(bank % 4)
		gBoost := uint8((bank / 4) % 4)
		bBoost := uint8((bank / 8) * 2)
		for i := 0; i < spriteLabColorsPerBank; i++ {
			base := nrgbaToRGB555(spriteLabBasePalette[i])
			r, g, b := decodeRGB555(base)
			r = clamp5(r + rBoost)
			g = clamp5(g + gBoost)
			b = clamp5(b + bBoost)
			out[bank*spriteLabColorsPerBank+i] = encodeRGB555(r, g, b)
		}
	}
	return out
}

func encodeRGB555(r, g, b uint8) uint16 {
	r &= 0x1F
	g &= 0x1F
	b &= 0x1F
	low := (b & 0x1F) | ((g & 0x07) << 5)
	high := ((r & 0x1F) << 2) | ((g >> 3) & 0x03)
	return uint16(low) | (uint16(high) << 8)
}

func decodeRGB555(v uint16) (uint8, uint8, uint8) {
	low := uint8(v & 0xFF)
	high := uint8((v >> 8) & 0xFF)
	r := (high & 0x7C) >> 2
	g := ((high & 0x03) << 3) | ((low & 0xE0) >> 5)
	b := low & 0x1F
	return r, g, b
}

func rgb555ToNRGBA(v uint16) color.NRGBA {
	r, g, b := decodeRGB555(v)
	return color.NRGBA{
		R: uint8((uint32(r) * 255) / 31),
		G: uint8((uint32(g) * 255) / 31),
		B: uint8((uint32(b) * 255) / 31),
		A: 0xFF,
	}
}

func nrgbaToRGB555(c color.NRGBA) uint16 {
	r := uint8((uint32(c.R)*31 + 127) / 255)
	g := uint8((uint32(c.G)*31 + 127) / 255)
	b := uint8((uint32(c.B)*31 + 127) / 255)
	return encodeRGB555(r, g, b)
}

func clamp5(v uint8) uint8 {
	if v > 31 {
		return 31
	}
	return v
}

func parsePaletteBankSelection(text string) (int, error) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "Bank"))
	if trimmed == "" {
		return 0, fmt.Errorf("empty bank")
	}
	v, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, err
	}
	if v < 0 || v >= spriteLabPaletteBanks {
		return 0, fmt.Errorf("bank out of range")
	}
	return v, nil
}

func parseSpriteLabChannel(text string) (uint8, error) {
	v, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		return 0, err
	}
	if v < 0 || v > 31 {
		return 0, fmt.Errorf("channel out of range")
	}
	return uint8(v), nil
}

func parseSpriteLabRGB555Hex(text string) (uint16, error) {
	t := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(text), "0x"), "0X"))
	if t == "" {
		return 0, fmt.Errorf("empty hex")
	}
	v, err := strconv.ParseUint(t, 16, 16)
	if err != nil {
		return 0, err
	}
	return uint16(v), nil
}

func spriteLabSizeOptions() []int {
	out := make([]int, 0, ((spriteLabMaxSize-spriteLabMinSize)/spriteLabSizeStep)+1)
	for v := spriteLabMinSize; v <= spriteLabMaxSize; v += spriteLabSizeStep {
		out = append(out, v)
	}
	return out
}

func isValidSpriteDimension(v int) bool {
	if v < spriteLabMinSize || v > spriteLabMaxSize {
		return false
	}
	return v%spriteLabSizeStep == 0
}

func formatSpriteSizeLabel(w, h int) string {
	return fmt.Sprintf("%dx%d", w, h)
}

func parseSpriteSizeSelection(text string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(strings.ToLower(text)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid size")
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}
	if !isValidSpriteDimension(w) || !isValidSpriteDimension(h) {
		return 0, 0, fmt.Errorf("size out of supported range")
	}
	return w, h, nil
}

const (
	spriteLabEditorMaxPx  = 384
	spriteLabPreviewMaxPx = 192
)

// spriteLabCellPx computes an integer cell size so every sprite pixel maps to
// exactly cellPx*cellPx screen pixels with no fractional scaling.
func spriteLabCellPx(spriteW, spriteH, maxPx int) int {
	if spriteW < 1 {
		spriteW = 1
	}
	if spriteH < 1 {
		spriteH = 1
	}
	largest := spriteW
	if spriteH > largest {
		largest = spriteH
	}
	cell := maxPx / largest
	if cell < 1 {
		cell = 1
	}
	return cell
}

func spriteLabEditorDisplaySize(w, h int) fyne.Size {
	cell := spriteLabCellPx(w, h, spriteLabEditorMaxPx)
	return fyne.NewSize(float32(w*cell), float32(h*cell))
}

func spriteLabPreviewDisplaySize(w, h int) fyne.Size {
	cell := spriteLabCellPx(w, h, spriteLabPreviewMaxPx)
	return fyne.NewSize(float32(w*cell), float32(h*cell))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
