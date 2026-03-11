package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"nitro-core-dx/internal/corelx"
)

const (
	tilemapLabFormatV1     = "clxtilemap-v1"
	tilemapLabDefaultSize  = 32
	tilemapLabMinSize      = 8
	tilemapLabMaxSize      = 32
	tilemapLabSizeStep     = 8
	tilemapLabHistoryLimit = 128
	tilemapLabEditorMaxPx  = 512
	tilemapLabPreviewMaxPx = 192
	tilemapLabTileIndexMax = 255
	tilemapLabPaletteMax   = 15
)

var (
	tilemapLabAssetHeaderRegex = regexp.MustCompile(`(?m)^asset\s+([A-Za-z_][A-Za-z0-9_]*)\s*:`)
)

type tilemapLabAsset struct {
	Format  string   `json:"format"`
	Name    string   `json:"name"`
	Width   int      `json:"width"`
	Height  int      `json:"height"`
	Entries []uint16 `json:"entries"`
}

type tilemapLabHistoryState struct {
	Width   int
	Height  int
	Entries []uint16
}

type tilemapLabTool string

const (
	tilemapLabToolPencil tilemapLabTool = "Pencil"
	tilemapLabToolErase  tilemapLabTool = "Erase"
)

type tilemapLabTileSet struct {
	Name  string
	Tiles [][]uint8 // each tile is 8x8, 64 palette indices (0-15)
}

func (s *devKitState) buildTilemapPane() fyne.CanvasObject {
	mapW := tilemapLabDefaultSize
	mapH := tilemapLabDefaultSize
	entries := make([]uint16, mapW*mapH)

	currentTool := tilemapLabToolPencil
	selectedTile := 1
	selectedPal := 0
	selectedFlipX := false
	selectedFlipY := false
	showGrid := true
	hoverX, hoverY := -1, -1
	strokeDirty := false
	suppressSizeSelect := false
	availableTileSets := make([]tilemapLabTileSet, 0)
	selectedTileSetIdx := -1

	history := make([]tilemapLabHistoryState, 0, 16)
	historyIndex := -1

	statusLabel := widget.NewLabel("Tilemap Lab ready")
	modeSummary := widget.NewLabel("")
	coordLabel := widget.NewLabel("Cursor: --")
	statsLabel := widget.NewLabel("")
	sizeLabel := widget.NewLabel("")
	historyLabel := widget.NewLabel("")

	nameEntry := widget.NewEntry()
	nameEntry.SetText("LevelMap")

	tileEntry := widget.NewEntry()
	tileEntry.SetText("1")
	palEntry := widget.NewEntry()
	palEntry.SetText("0")
	hexPreview := newReadOnlyTextArea()
	hexPreview.SetMinRowsVisible(8)
	tileSetSelect := widget.NewSelect([]string{"(No tileset parsed)"}, nil)
	tileAtlasImage := canvas.NewImageFromImage(image.NewNRGBA(image.Rect(0, 0, 1, 1)))
	tileAtlasImage.FillMode = canvas.ImageFillStretch
	tileAtlasImage.ScaleMode = canvas.ImageScalePixels
	tileAtlasImage.SetMinSize(fyne.NewSize(320, 192))
	var tileAtlasOverlay *spriteLabPaintOverlay
	atlasCols, atlasRows := 1, 1
	atlasCell := 16
	var refreshVisuals func()

	flipXCheck := widget.NewCheck("Flip X", func(v bool) {
		selectedFlipX = v
		refreshSummary := fmt.Sprintf("Brush attrs updated (tile=%d pal=%d fx=%t fy=%t)", selectedTile, selectedPal, selectedFlipX, selectedFlipY)
		statusLabel.SetText(refreshSummary)
		refreshVisuals()
	})
	flipYCheck := widget.NewCheck("Flip Y", func(v bool) {
		selectedFlipY = v
		refreshSummary := fmt.Sprintf("Brush attrs updated (tile=%d pal=%d fx=%t fy=%t)", selectedTile, selectedPal, selectedFlipX, selectedFlipY)
		statusLabel.SetText(refreshSummary)
		refreshVisuals()
	})

	editorImage := canvas.NewImageFromImage(image.NewNRGBA(image.Rect(0, 0, 1, 1)))
	editorImage.FillMode = canvas.ImageFillStretch
	editorImage.ScaleMode = canvas.ImageScalePixels
	previewImage := canvas.NewImageFromImage(image.NewNRGBA(image.Rect(0, 0, 1, 1)))
	previewImage.FillMode = canvas.ImageFillStretch
	previewImage.ScaleMode = canvas.ImageScalePixels

	undoButton := widget.NewButton("Undo", nil)
	redoButton := widget.NewButton("Redo", nil)
	sizeSelect := widget.NewSelect(nil, nil)
	var overlay *spriteLabPaintOverlay

	snapshot := func() tilemapLabHistoryState {
		cp := make([]uint16, len(entries))
		copy(cp, entries)
		return tilemapLabHistoryState{
			Width:   mapW,
			Height:  mapH,
			Entries: cp,
		}
	}

	commitHistory := func() {
		if historyIndex < len(history)-1 {
			history = history[:historyIndex+1]
		}
		if len(history) >= tilemapLabHistoryLimit {
			copy(history, history[1:])
			history = history[:len(history)-1]
			if historyIndex > 0 {
				historyIndex--
			}
		}
		history = append(history, snapshot())
		historyIndex = len(history) - 1
	}

	restoreHistory := func(idx int) bool {
		if idx < 0 || idx >= len(history) {
			return false
		}
		state := history[idx]
		mapW = state.Width
		mapH = state.Height
		entries = make([]uint16, len(state.Entries))
		copy(entries, state.Entries)
		historyIndex = idx
		setSizeSelection(sizeSelect, mapW, mapH, &suppressSizeSelect)
		updateCanvasSizes(editorImage, previewImage, overlay, mapW, mapH)
		return true
	}

	setEntry := func(x, y int, val uint16) bool {
		if x < 0 || x >= mapW || y < 0 || y >= mapH {
			return false
		}
		i := y*mapW + x
		if entries[i] == val {
			return false
		}
		entries[i] = val
		return true
	}

	brushEntry := func() uint16 {
		tile := uint16(clampInt(selectedTile, 0, tilemapLabTileIndexMax))
		attr := uint16(clampInt(selectedPal, 0, tilemapLabPaletteMax))
		if selectedFlipX {
			attr |= 0x10
		}
		if selectedFlipY {
			attr |= 0x20
		}
		return (attr << 8) | tile
	}

	applyFill := func(val uint16) bool {
		changed := false
		for i := range entries {
			if entries[i] != val {
				entries[i] = val
				changed = true
			}
		}
		return changed
	}

	parseBrushInputs := func() bool {
		t, err := strconv.Atoi(strings.TrimSpace(tileEntry.Text))
		if err != nil || t < 0 || t > tilemapLabTileIndexMax {
			statusLabel.SetText("Tile index must be 0-255")
			return false
		}
		p, err := strconv.Atoi(strings.TrimSpace(palEntry.Text))
		if err != nil || p < 0 || p > tilemapLabPaletteMax {
			statusLabel.SetText("Palette index must be 0-15")
			return false
		}
		selectedTile = t
		selectedPal = p
		return true
	}

	effectivePaletteData := func() []uint16 {
		out := defaultSpriteLabPaletteData()
		palColors, palSet := parsePaletteAssignments(s.sourceEditor.Text())
		for b := 0; b < spriteLabPaletteBanks; b++ {
			for i := 0; i < spriteLabColorsPerBank; i++ {
				if palSet[b][i] {
					out[b*spriteLabColorsPerBank+i] = nrgbaToRGB555(palColors[b][i])
				}
			}
		}
		return out
	}

	selectedTileSet := func() *tilemapLabTileSet {
		if selectedTileSetIdx < 0 || selectedTileSetIdx >= len(availableTileSets) {
			return nil
		}
		return &availableTileSets[selectedTileSetIdx]
	}

	refreshTileAtlas := func() {
		palData := effectivePaletteData()
		ts := selectedTileSet()
		img, cols, rows, cell := renderTileAtlasImage(ts, selectedPal, palData, selectedTile)
		atlasCols, atlasRows, atlasCell = cols, rows, cell
		tileAtlasImage.Image = img
		tileAtlasImage.Refresh()
		if tileAtlasOverlay != nil {
			tileAtlasOverlay.SetGrid(atlasCols, atlasRows, atlasCell)
		}
	}

	rebuildTileSetList := func() {
		sets, err := parseTileSetsFromSource(s.sourceEditor.Text())
		if err != nil {
			statusLabel.SetText("Tile parse warning: " + err.Error())
		}
		availableTileSets = sets
		if len(availableTileSets) == 0 {
			selectedTileSetIdx = -1
			tileSetSelect.Options = []string{"(No tileset parsed)"}
			tileSetSelect.SetSelected("(No tileset parsed)")
			refreshTileAtlas()
			return
		}
		opts := make([]string, 0, len(availableTileSets))
		for _, ts := range availableTileSets {
			opts = append(opts, fmt.Sprintf("%s (%d tiles)", ts.Name, len(ts.Tiles)))
		}
		tileSetSelect.Options = opts
		if selectedTileSetIdx < 0 || selectedTileSetIdx >= len(availableTileSets) {
			selectedTileSetIdx = 0
		}
		tileSetSelect.SetSelected(opts[selectedTileSetIdx])
		refreshTileAtlas()
	}

	countNonZero := func() int {
		n := 0
		for _, v := range entries {
			if v != 0 {
				n++
			}
		}
		return n
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
		cell := spriteLabCellPx(mapW, mapH, tilemapLabEditorMaxPx)
		editorImage.Image = renderTilemapLabImage(entries, mapW, mapH, cell, hoverX, hoverY, showGrid, selectedTileSet(), effectivePaletteData())
		editorImage.Refresh()
	}

	refreshVisuals = func() {
		refreshEditorOnly()
		cell := spriteLabCellPx(mapW, mapH, tilemapLabPreviewMaxPx)
		previewImage.Image = renderTilemapLabImage(entries, mapW, mapH, cell, -1, -1, false, selectedTileSet(), effectivePaletteData())
		previewImage.Refresh()
		refreshTilemapHexPreview(hexPreview, entries, mapW, mapH)
		modeSummary.SetText(fmt.Sprintf("Tool: %s | Brush tile=%d pal=%d fx=%t fy=%t | Size: %dx%d", currentTool, selectedTile, selectedPal, selectedFlipX, selectedFlipY, mapW, mapH))
		statsLabel.SetText(fmt.Sprintf("Non-zero cells: %d/%d", countNonZero(), mapW*mapH))
		sizeLabel.SetText(fmt.Sprintf("Size: %dx%d", mapW, mapH))
		refreshHistoryButtons()
		refreshTileAtlas()
	}

	paintAt := func(x, y int) {
		val := brushEntry()
		if currentTool == tilemapLabToolErase {
			val = 0
		}
		if setEntry(x, y, val) {
			strokeDirty = true
			refreshEditorOnly()
		}
	}

	overlay = newSpriteLabPaintOverlay(
		mapW, mapH, spriteLabCellPx(mapW, mapH, tilemapLabEditorMaxPx),
		func() { strokeDirty = false },
		func() {
			if strokeDirty {
				commitHistory()
				refreshVisuals()
				statusLabel.SetText("Tilemap stroke applied")
			}
		},
		func(x, y int) {
			if !parseBrushInputs() {
				return
			}
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

	tileAtlasOverlay = newSpriteLabPaintOverlay(
		atlasCols, atlasRows, atlasCell,
		nil, nil,
		func(x, y int) {
			idx := y*atlasCols + x
			ts := selectedTileSet()
			if ts == nil || idx < 0 || idx >= len(ts.Tiles) {
				return
			}
			selectedTile = idx
			tileEntry.SetText(strconv.Itoa(selectedTile))
			statusLabel.SetText(fmt.Sprintf("Selected tile %d from %s", selectedTile, ts.Name))
			refreshVisuals()
		},
		nil, nil,
	)

	sizeSelect.Options = make([]string, 0, ((tilemapLabMaxSize-tilemapLabMinSize)/tilemapLabSizeStep)+1)
	for v := tilemapLabMinSize; v <= tilemapLabMaxSize; v += tilemapLabSizeStep {
		sizeSelect.Options = append(sizeSelect.Options, fmt.Sprintf("%dx%d", v, v))
	}
	sizeSelect.OnChanged = func(v string) {
		if suppressSizeSelect {
			return
		}
		w, h, err := parseSpriteSizeSelection(v)
		if err != nil {
			statusLabel.SetText("Invalid tilemap size")
			setSizeSelection(sizeSelect, mapW, mapH, &suppressSizeSelect)
			return
		}
		if !resizeTilemap(&entries, mapW, mapH, w, h) {
			statusLabel.SetText("Tilemap size unchanged")
			return
		}
		mapW, mapH = w, h
		updateCanvasSizes(editorImage, previewImage, overlay, mapW, mapH)
		commitHistory()
		refreshVisuals()
		statusLabel.SetText(fmt.Sprintf("Resized tilemap to %dx%d", mapW, mapH))
	}
	setSizeSelection(sizeSelect, mapW, mapH, &suppressSizeSelect)
	updateCanvasSizes(editorImage, previewImage, overlay, mapW, mapH)

	toolGroup := widget.NewRadioGroup([]string{string(tilemapLabToolPencil), string(tilemapLabToolErase)}, func(v string) {
		if v == string(tilemapLabToolErase) {
			currentTool = tilemapLabToolErase
			statusLabel.SetText("Tool: Erase")
		} else {
			currentTool = tilemapLabToolPencil
			statusLabel.SetText("Tool: Pencil")
		}
		refreshVisuals()
	})
	toolGroup.Horizontal = true
	toolGroup.SetSelected(string(tilemapLabToolPencil))

	gridCheck := widget.NewCheck("Show Grid", func(v bool) {
		showGrid = v
		refreshEditorOnly()
	})
	gridCheck.SetChecked(true)

	tileSetSelect.OnChanged = func(v string) {
		for i, ts := range availableTileSets {
			label := fmt.Sprintf("%s (%d tiles)", ts.Name, len(ts.Tiles))
			if label == v {
				selectedTileSetIdx = i
				statusLabel.SetText("Tile source: " + ts.Name)
				refreshVisuals()
				return
			}
		}
	}

	refreshTilesBtn := widget.NewButton("Refresh Tiles From Code", func() {
		rebuildTileSetList()
		refreshVisuals()
		if selectedTileSet() == nil {
			statusLabel.SetText("No tile assets found in current source")
		} else {
			statusLabel.SetText("Tile assets refreshed from source")
		}
	})

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
			statusLabel.SetText("Tilemap already clear")
			return
		}
		commitHistory()
		refreshVisuals()
		statusLabel.SetText("Tilemap cleared")
	})
	fillButton := widget.NewButton("Fill", func() {
		if !parseBrushInputs() {
			return
		}
		if !applyFill(brushEntry()) {
			statusLabel.SetText("Tilemap already filled with brush value")
			return
		}
		commitHistory()
		refreshVisuals()
		statusLabel.SetText("Filled tilemap with brush value")
	})

	saveButton := widget.NewButton("Export .clxtilemap", func() {
		fd := dialog.NewFileSave(func(wc fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, s.window)
				return
			}
			if wc == nil {
				return
			}
			defer wc.Close()
			data, err := marshalTilemapLabAsset(nameEntry.Text, mapW, mapH, entries)
			if err != nil {
				dialog.ShowError(err, s.window)
				return
			}
			if _, err := wc.Write(data); err != nil {
				dialog.ShowError(err, s.window)
				return
			}
			if path := uriPath(wc.URI()); path != "" {
				s.settings.LastSourceDir = filepath.Dir(path)
				s.persistSettings()
			}
			statusLabel.SetText("Exported tilemap asset")
		}, s.window)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".clxtilemap", ".json"}))
		if loc := dialogListableForDir(s.settings.LastSourceDir); loc != nil {
			fd.SetLocation(loc)
		}
		fd.SetFileName(sanitizeSpriteLabName(nameEntry.Text) + ".clxtilemap")
		fd.Show()
	})

	loadButton := widget.NewButton("Import .clxtilemap", func() {
		fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, s.window)
				return
			}
			if rc == nil {
				return
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				dialog.ShowError(err, s.window)
				return
			}
			asset, err := unmarshalTilemapLabAsset(data)
			if err != nil {
				dialog.ShowError(err, s.window)
				return
			}
			mapW, mapH = asset.Width, asset.Height
			entries = make([]uint16, len(asset.Entries))
			copy(entries, asset.Entries)
			nameEntry.SetText(asset.Name)
			setSizeSelection(sizeSelect, mapW, mapH, &suppressSizeSelect)
			updateCanvasSizes(editorImage, previewImage, overlay, mapW, mapH)
			commitHistory()
			refreshVisuals()
			if path := uriPath(rc.URI()); path != "" {
				s.settings.LastSourceDir = filepath.Dir(path)
				s.persistSettings()
			}
			statusLabel.SetText("Imported tilemap asset")
		}, s.window)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".clxtilemap", ".json"}))
		if loc := dialogListableForDir(s.settings.LastSourceDir); loc != nil {
			fd.SetLocation(loc)
		}
		fd.Show()
	})

	insertButton := widget.NewButton("Insert CoreLX Asset", func() {
		name := sanitizeSpriteLabName(nameEntry.Text)
		snippet, err := tilemapLabCoreLXAssetSnippet(name, mapW, mapH, entries)
		if err != nil {
			dialog.ShowError(err, s.window)
			return
		}
		next := s.sourceEditor.Text()
		if strings.TrimSpace(next) != "" {
			next = strings.TrimRight(next, "\n") + "\n\n"
		}
		next += snippet + "\n"
		s.setSourceContent(next, true, false)
		if s.workbenchTabs != nil {
			s.workbenchTabs.SelectIndex(0)
		}
		s.setStatus("Inserted tilemap asset snippet")
		statusLabel.SetText("Inserted CoreLX tilemap snippet")
	})

	applyProjectButton := widget.NewButton("Apply To Project", func() {
		name := sanitizeSpriteLabName(nameEntry.Text)
		snippet, err := tilemapLabCoreLXAssetSnippet(name, mapW, mapH, entries)
		if err != nil {
			dialog.ShowError(err, s.window)
			return
		}
		next, summary := upsertTilemapLabBlockIntoSource(s.sourceEditor.Text(), name, snippet)
		s.setSourceContent(next, true, false)
		if s.workbenchTabs != nil {
			s.workbenchTabs.SelectIndex(0)
		}
		s.setStatus(summary)
		s.appendBuildOutput(summary)
		statusLabel.SetText(summary)
	})
	applyProjectButton.Importance = widget.HighImportance
	insertButton.Importance = widget.HighImportance

	applyManifestButton := widget.NewButton("Apply To Manifest", func() {
		name := sanitizeSpriteLabName(nameEntry.Text)
		hexData, err := tilemapLabAssetHexData(mapW, mapH, entries)
		if err != nil {
			dialog.ShowError(err, s.window)
			return
		}
		record := corelx.ProjectAssetRecord{
			Name:     name,
			Type:     "tilemap",
			Encoding: "hex",
			Data:     hexData,
		}
		manifestPath, state, err := upsertProjectAssetManifestRecord(s.currentPath, record)
		if err != nil {
			dialog.ShowError(err, s.window)
			return
		}
		s.rememberSourcePath(manifestPath)
		s.setStatus(fmt.Sprintf("Applied tilemap asset %s (%s) to %s", name, state, filepath.Base(manifestPath)))
		s.appendBuildOutput(fmt.Sprintf("Manifest upsert: %s (%s) -> %s", name, state, manifestPath))
		statusLabel.SetText(fmt.Sprintf("Manifest updated: %s (%s)", name, state))
	})
	applyManifestButton.Importance = widget.MediumImportance

	canvasPanel := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Tilemap Canvas"),
			container.NewHBox(widget.NewLabel("Map Size"), sizeSelect, sizeLabel, gridCheck),
			toolGroup,
			modeSummary,
		),
		container.NewHBox(coordLabel),
		nil, nil,
		container.NewCenter(container.NewStack(editorImage, overlay)),
	)

	paintTab := container.NewVBox(
		widget.NewLabel("Brush"),
		container.NewGridWithColumns(2, widget.NewLabel("Tile Index (0-255)"), tileEntry),
		container.NewGridWithColumns(2, widget.NewLabel("Palette (0-15)"), palEntry),
		container.NewGridWithColumns(2, flipXCheck, flipYCheck),
		container.NewGridWithColumns(2, undoButton, redoButton),
		container.NewGridWithColumns(2, clearButton, fillButton),
		historyLabel,
		widget.NewSeparator(),
		widget.NewLabel("Tile Source"),
		tileSetSelect,
		refreshTilesBtn,
		widget.NewLabel("Tile Atlas (click to pick tile index)"),
		container.NewStack(tileAtlasImage, tileAtlasOverlay),
	)
	assetTab := container.NewVBox(
		widget.NewLabel("Asset Name"),
		nameEntry,
		widget.NewLabel("Preview"),
		previewImage,
		statsLabel,
	)
	exportTab := container.NewVBox(
		container.NewGridWithColumns(2, loadButton, saveButton),
		applyManifestButton,
		applyProjectButton,
		insertButton,
	)
	inspectorTab := container.NewVBox(
		widget.NewLabel("Packed Tilemap Bytes (tile,attr pairs)"),
		hexPreview,
	)

	rightTabs := container.NewAppTabs(
		container.NewTabItem("Paint", container.NewScroll(paintTab)),
		container.NewTabItem("Asset", container.NewScroll(assetTab)),
		container.NewTabItem("Export/Code", container.NewScroll(exportTab)),
		container.NewTabItem("Inspector", container.NewScroll(inspectorTab)),
	)
	rightTabs.SetTabLocation(container.TabLocationTop)

	split := container.NewHSplit(canvasPanel, rightTabs)
	split.Offset = 0.60

	commitHistory()
	rebuildTileSetList()
	refreshVisuals()

	return container.NewBorder(nil, statusLabel, nil, nil, split)
}

func updateCanvasSizes(editorImage, previewImage *canvas.Image, overlay *spriteLabPaintOverlay, w, h int) {
	editorImage.SetMinSize(fyne.NewSize(float32(w*spriteLabCellPx(w, h, tilemapLabEditorMaxPx)), float32(h*spriteLabCellPx(w, h, tilemapLabEditorMaxPx))))
	previewImage.SetMinSize(fyne.NewSize(float32(w*spriteLabCellPx(w, h, tilemapLabPreviewMaxPx)), float32(h*spriteLabCellPx(w, h, tilemapLabPreviewMaxPx))))
	if overlay != nil {
		overlay.SetGrid(w, h, spriteLabCellPx(w, h, tilemapLabEditorMaxPx))
	}
}

func setSizeSelection(sel *widget.Select, w, h int, suppress *bool) {
	*suppress = true
	sel.SetSelected(fmt.Sprintf("%dx%d", w, h))
	*suppress = false
}

func resizeTilemap(entries *[]uint16, oldW, oldH, newW, newH int) bool {
	if oldW == newW && oldH == newH {
		return false
	}
	if !isValidSpriteDimension(newW) || !isValidSpriteDimension(newH) {
		return false
	}
	next := make([]uint16, newW*newH)
	copyW := minInt(oldW, newW)
	copyH := minInt(oldH, newH)
	for y := 0; y < copyH; y++ {
		for x := 0; x < copyW; x++ {
			next[y*newW+x] = (*entries)[y*oldW+x]
		}
	}
	*entries = next
	return true
}

func renderTilemapLabImage(entries []uint16, w, h, cell, hoverX, hoverY int, drawGrid bool, tileSet *tilemapLabTileSet, paletteData []uint16) image.Image {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if cell < 1 {
		cell = 1
	}
	img := image.NewNRGBA(image.Rect(0, 0, w*cell, h*cell))
	gridColor := color.NRGBA{R: 0x2E, G: 0x33, B: 0x3E, A: 0xFF}
	hoverColor := color.NRGBA{R: 0xF8, G: 0xFA, B: 0xFF, A: 0xFF}
	gridThick := 0
	if drawGrid {
		if cell >= 16 {
			gridThick = 2
		} else if cell >= 4 {
			gridThick = 1
		}
	}
	for py := 0; py < h*cell; py++ {
		y := py / cell
		by := py % cell
		for px := 0; px < w*cell; px++ {
			x := px / cell
			bx := px % cell
			i := y*w + x
			val := uint16(0)
			if i >= 0 && i < len(entries) {
				val = entries[i]
			}
			tile := uint8(val & 0xFF)
			attr := uint8((val >> 8) & 0xFF)
			pal := attr & 0x0F
			flipX := (attr & 0x10) != 0
			flipY := (attr & 0x20) != 0
			clr := tilemapCellColor(tile, pal)
			if tileSet != nil && int(tile) < len(tileSet.Tiles) && len(paletteData) == spriteLabPaletteCount {
				srcX := (bx * 8) / maxInt(1, cell)
				srcY := (by * 8) / maxInt(1, cell)
				if srcX > 7 {
					srcX = 7
				}
				if srcY > 7 {
					srcY = 7
				}
				if flipX {
					srcX = 7 - srcX
				}
				if flipY {
					srcY = 7 - srcY
				}
				px := tileSet.Tiles[int(tile)][srcY*8+srcX] & 0x0F
				clr = rgb555ToNRGBA(paletteData[int(pal)*spriteLabColorsPerBank+int(px)])
			}
			if gridThick > 0 && (bx < gridThick || by < gridThick) {
				clr = gridColor
			}
			if x == hoverX && y == hoverY {
				if bx < 2 || by < 2 || bx >= cell-2 || by >= cell-2 {
					clr = hoverColor
				}
			}
			off := py*img.Stride + px*4
			img.Pix[off+0] = clr.R
			img.Pix[off+1] = clr.G
			img.Pix[off+2] = clr.B
			img.Pix[off+3] = clr.A
		}
	}
	return img
}

func tilemapCellColor(tile uint8, pal uint8) color.NRGBA {
	r := uint8((int(tile)*53 + int(pal)*29) % 256)
	g := uint8((int(tile)*97 + int(pal)*41) % 256)
	b := uint8((int(tile)*31 + int(pal)*113) % 256)
	return color.NRGBA{R: r, G: g, B: b, A: 0xFF}
}

func renderTileAtlasImage(tileSet *tilemapLabTileSet, paletteBank int, paletteData []uint16, selectedTile int) (image.Image, int, int, int) {
	if tileSet == nil || len(tileSet.Tiles) == 0 {
		return image.NewNRGBA(image.Rect(0, 0, 1, 1)), 1, 1, 1
	}
	cols := 16
	rows := int(math.Ceil(float64(len(tileSet.Tiles)) / float64(cols)))
	cell := 16
	w := cols * cell
	h := rows * cell
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for ty := 0; ty < rows; ty++ {
		for tx := 0; tx < cols; tx++ {
			idx := ty*cols + tx
			if idx >= len(tileSet.Tiles) {
				continue
			}
			tile := tileSet.Tiles[idx]
			for py := 0; py < cell; py++ {
				for px := 0; px < cell; px++ {
					sx := (px * 8) / cell
					sy := (py * 8) / cell
					pix := tile[sy*8+sx] & 0x0F
					c := tilemapCellColor(uint8(idx), uint8(paletteBank))
					if len(paletteData) == spriteLabPaletteCount {
						base := clampInt(paletteBank, 0, spriteLabPaletteBanks-1) * spriteLabColorsPerBank
						c = rgb555ToNRGBA(paletteData[base+int(pix)])
					}
					x := tx*cell + px
					y := ty*cell + py
					off := img.PixOffset(x, y)
					img.Pix[off+0] = c.R
					img.Pix[off+1] = c.G
					img.Pix[off+2] = c.B
					img.Pix[off+3] = 0xFF
				}
			}
			if idx == selectedTile {
				drawTileAtlasBorder(img, tx*cell, ty*cell, cell, color.NRGBA{R: 0xF8, G: 0xFA, B: 0xFF, A: 0xFF})
			} else {
				drawTileAtlasBorder(img, tx*cell, ty*cell, cell, color.NRGBA{R: 0x2E, G: 0x33, B: 0x3E, A: 0xFF})
			}
		}
	}
	return img, cols, rows, cell
}

func drawTileAtlasBorder(img *image.NRGBA, x, y, size int, c color.NRGBA) {
	if size <= 1 {
		return
	}
	maxX := x + size - 1
	maxY := y + size - 1
	for px := x; px <= maxX; px++ {
		setNRGBAPixel(img, px, y, c)
		setNRGBAPixel(img, px, maxY, c)
	}
	for py := y; py <= maxY; py++ {
		setNRGBAPixel(img, x, py, c)
		setNRGBAPixel(img, maxX, py, c)
	}
}

func setNRGBAPixel(img *image.NRGBA, x, y int, c color.NRGBA) {
	if !image.Pt(x, y).In(img.Rect) {
		return
	}
	off := img.PixOffset(x, y)
	img.Pix[off+0] = c.R
	img.Pix[off+1] = c.G
	img.Pix[off+2] = c.B
	img.Pix[off+3] = c.A
}

func refreshTilemapHexPreview(entry *widget.Entry, entries []uint16, w, h int) {
	data := tilemapLabPackedBytes(entries)
	var sb strings.Builder
	rowBytes := w * 2
	for y := 0; y < h; y++ {
		if y > 0 {
			sb.WriteByte('\n')
		}
		base := y * rowBytes
		for x := 0; x < rowBytes; x++ {
			if x > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(fmt.Sprintf("%02X", data[base+x]))
		}
	}
	entry.Enable()
	entry.SetText(sb.String())
	entry.Disable()
}

func tilemapLabPackedBytes(entries []uint16) []byte {
	out := make([]byte, 0, len(entries)*2)
	for _, e := range entries {
		out = append(out, byte(e&0xFF), byte((e>>8)&0xFF))
	}
	return out
}

func marshalTilemapLabAsset(name string, w, h int, entries []uint16) ([]byte, error) {
	a := tilemapLabAsset{
		Format:  tilemapLabFormatV1,
		Name:    sanitizeSpriteLabName(name),
		Width:   w,
		Height:  h,
		Entries: append([]uint16(nil), entries...),
	}
	if err := validateTilemapLabAsset(a); err != nil {
		return nil, err
	}
	return json.MarshalIndent(a, "", "  ")
}

func unmarshalTilemapLabAsset(data []byte) (tilemapLabAsset, error) {
	var a tilemapLabAsset
	if err := json.Unmarshal(data, &a); err != nil {
		return tilemapLabAsset{}, fmt.Errorf("invalid tilemap asset JSON: %w", err)
	}
	if a.Format == "" {
		a.Format = tilemapLabFormatV1
	}
	if a.Name == "" {
		a.Name = "LevelMap"
	}
	a.Name = sanitizeSpriteLabName(a.Name)
	if a.Width == 0 {
		a.Width = tilemapLabDefaultSize
	}
	if a.Height == 0 {
		a.Height = tilemapLabDefaultSize
	}
	if err := validateTilemapLabAsset(a); err != nil {
		return tilemapLabAsset{}, err
	}
	return a, nil
}

func validateTilemapLabAsset(a tilemapLabAsset) error {
	if a.Format != tilemapLabFormatV1 {
		return fmt.Errorf("unsupported tilemap format: %q", a.Format)
	}
	if !isValidSpriteDimension(a.Width) || !isValidSpriteDimension(a.Height) {
		return fmt.Errorf("tilemap dimensions must be %d-%d in steps of %d", tilemapLabMinSize, tilemapLabMaxSize, tilemapLabSizeStep)
	}
	if len(a.Entries) != a.Width*a.Height {
		return fmt.Errorf("tilemap entries length must be %d", a.Width*a.Height)
	}
	return nil
}

func tilemapLabCoreLXAssetSnippet(name string, w, h int, entries []uint16) (string, error) {
	if w < 1 || h < 1 || len(entries) != w*h {
		return "", fmt.Errorf("invalid tilemap shape (%dx%d, entries=%d)", w, h, len(entries))
	}
	data := tilemapLabPackedBytes(entries)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("-- Tilemap Lab asset (%dx%d, entry=tile+attr)\n", w, h))
	sb.WriteString("asset ")
	sb.WriteString(sanitizeSpriteLabName(name))
	sb.WriteString(": tilemap hex\n")
	rowBytes := w * 2
	for y := 0; y < h; y++ {
		sb.WriteString("    ")
		base := y * rowBytes
		for x := 0; x < rowBytes; x++ {
			if x > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(fmt.Sprintf("%02X", data[base+x]))
		}
		if y < h-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String(), nil
}

func tilemapLabAssetHexData(w, h int, entries []uint16) (string, error) {
	if w < 1 || h < 1 || len(entries) != w*h {
		return "", fmt.Errorf("invalid tilemap shape (%dx%d, entries=%d)", w, h, len(entries))
	}
	data := tilemapLabPackedBytes(entries)
	return bytesToHexFields(data), nil
}

func upsertTilemapLabBlockIntoSource(source, assetName, snippet string) (string, string) {
	updated := strings.TrimRight(source, "\n")
	if updated == "" {
		return snippet + "\n", fmt.Sprintf("Applied tilemap asset %s (new)", assetName)
	}
	lines := strings.Split(updated, "\n")
	if start, end, ok := findTilemapAssetBlock(lines, assetName); ok {
		repl := strings.Split(snippet, "\n")
		next := make([]string, 0, len(lines)-((end-start)+1)+len(repl))
		next = append(next, lines[:start]...)
		next = append(next, repl...)
		next = append(next, lines[end+1:]...)
		return strings.Join(next, "\n") + "\n", fmt.Sprintf("Applied tilemap asset %s (updated)", assetName)
	}
	lines = append(lines, "")
	lines = append(lines, strings.Split(snippet, "\n")...)
	return strings.Join(lines, "\n") + "\n", fmt.Sprintf("Applied tilemap asset %s (new)", assetName)
}

func findTilemapAssetBlock(lines []string, assetName string) (int, int, bool) {
	for i := 0; i < len(lines); i++ {
		m := tilemapLabAssetHeaderRegex.FindStringSubmatch(strings.TrimSpace(lines[i]))
		if len(m) != 2 || m[1] != assetName {
			continue
		}
		end := i
		for j := i + 1; j < len(lines); j++ {
			trimmed := strings.TrimSpace(lines[j])
			if trimmed == "" || strings.HasPrefix(trimmed, "--") {
				end = j
				continue
			}
			if !isIndentedLine(lines[j]) {
				break
			}
			end = j
		}
		return i, end, true
	}
	return 0, 0, false
}

func parseTileSetsFromSource(source string) ([]tilemapLabTileSet, error) {
	lexer := corelx.NewLexer(source)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, err
	}
	parser := corelx.NewParser(tokens)
	prog, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	out := make([]tilemapLabTileSet, 0, len(prog.Assets))
	for _, a := range prog.Assets {
		if a == nil {
			continue
		}
		if a.Encoding != "hex" {
			continue
		}
		switch a.Type {
		case "tiles8", "tileset", "sprite":
		default:
			continue
		}
		data, err := decodeHexBytes(a.Data)
		if err != nil {
			continue
		}
		tiles := decode4BPPTiles(data)
		if len(tiles) == 0 {
			continue
		}
		out = append(out, tilemapLabTileSet{Name: a.Name, Tiles: tiles})
	}
	return out, nil
}

func decodeHexBytes(s string) ([]byte, error) {
	fields := strings.Fields(s)
	out := make([]byte, 0, len(fields))
	for _, tok := range fields {
		t := strings.TrimSpace(tok)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "0x") || strings.HasPrefix(t, "0X") {
			t = t[2:]
		}
		if len(t)%2 != 0 {
			return nil, fmt.Errorf("invalid hex byte token %q", tok)
		}
		for i := 0; i < len(t); i += 2 {
			part := t[i : i+2]
			v, err := strconv.ParseUint(part, 16, 8)
			if err != nil {
				return nil, err
			}
			out = append(out, byte(v))
		}
	}
	return out, nil
}

func decode4BPPTiles(data []byte) [][]uint8 {
	const tileBytes = 32
	tileCount := len(data) / tileBytes
	if tileCount <= 0 {
		return nil
	}
	out := make([][]uint8, 0, tileCount)
	for i := 0; i < tileCount; i++ {
		chunk := data[i*tileBytes : (i+1)*tileBytes]
		tile := make([]uint8, 64)
		p := 0
		for _, b := range chunk {
			hi := (b >> 4) & 0x0F
			lo := b & 0x0F
			if p < 64 {
				tile[p] = hi
				p++
			}
			if p < 64 {
				tile[p] = lo
				p++
			}
		}
		out = append(out, tile)
	}
	return out
}
