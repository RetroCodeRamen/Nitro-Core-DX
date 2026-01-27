package ppu

// ScanlineRenderer handles scanline-by-scanline and dot-by-dot rendering
// This replaces the frame-based RenderFrame() for clock-driven operation

// PPU timing constants (at 10 MHz CPU clock)
const (
	// Display resolution
	ScreenWidth  = 320
	ScreenHeight = 200

	// Timing (at 10 MHz)
	// Each scanline takes: 320 dots (visible) + 40 dots (hblank) = 360 dots
	// Each dot = 1 CPU cycle at 10 MHz
	DotsPerScanline = 360
	VisibleDots     = 320
	HBlankDots      = 40

	// Frame timing
	// Visible scanlines: 200
	// VBlank scanlines: 20 (total 220 scanlines per frame)
	VisibleScanlines = 200
	VBlankScanlines  = 20
	TotalScanlines   = 220
)

// StepPPU steps the PPU by a number of cycles (clock-driven)
// Returns error if any
func (p *PPU) StepPPU(cycles uint64) error {
	for i := uint64(0); i < cycles; i++ {
		if err := p.stepDot(); err != nil {
			return err
		}
	}
	return nil
}

// stepDot steps the PPU by one dot (one CPU cycle)
func (p *PPU) stepDot() error {
	// Initialize scanline/dot counters if needed
	if !p.scanlineInitialized {
		p.currentScanline = 0
		p.currentDot = 0
		p.scanlineInitialized = true
		p.frameStarted = false
	}

	// Check if we need to start a new frame
	if p.currentScanline == 0 && p.currentDot == 0 && !p.frameStarted {
		p.startFrame()
		p.frameStarted = true
		// Frame is starting, not complete yet
		p.FrameComplete = false
	}

	// Render current dot if in visible area
	if p.currentScanline < VisibleScanlines && p.currentDot < VisibleDots {
		// Render this pixel
		p.renderDot(p.currentScanline, p.currentDot)
	}

	// Handle HDMA on scanline start (if enabled)
	if p.currentDot == 0 && p.HDMAEnabled {
		p.updateHDMA(p.currentScanline)
	}

	// Advance dot counter
	p.currentDot++
	if p.currentDot >= DotsPerScanline {
		// End of scanline
		p.endScanline()
		p.currentDot = 0

		// Set VBlank flag at the END of scanline 199 (before incrementing to scanline 200)
		// This ensures the flag is set BEFORE scanline 200 starts, so ROM can read it
		// immediately when VBlank begins. This matches real hardware timing.
		if p.currentScanline == VisibleScanlines-1 {
			p.VBlankFlag = true
		}

		p.currentScanline++

		if p.currentScanline >= TotalScanlines {
			// End of frame
			p.endFrame()
			p.currentScanline = 0
			p.frameStarted = false
		}
	}

	return nil
}

// startFrame is called at the start of each frame
func (p *PPU) startFrame() {
	// Clear VBlank flag at start of frame
	// VBlank flag is set at end of visible period (scanline 200) and cleared here
	// This ensures sprite updates happen during VBlank, not during visible rendering
	p.VBlankFlag = false

	// Increment frame counter (hardware-accurate: simple counter)
	p.FrameCounter++

	// Clear output buffer
	for i := range p.OutputBuffer {
		p.OutputBuffer[i] = 0x000000 // Black
	}

	// Frame is not complete yet (we're starting to render)
	p.FrameComplete = false
}

// endScanline is called at the end of each scanline
func (p *PPU) endScanline() {
	// Handle end-of-scanline operations (if any)
	// For now, nothing special needed
}

// endFrame is called at the end of each frame
func (p *PPU) endFrame() {
	// Clear VBlank flag at end of frame
	// (It will be set again at start of next frame)
	// Actually, keep it set until read - it's cleared when read, not here

	// Mark frame as complete (buffer is safe to read)
	p.FrameComplete = true
}

// renderDot renders a single dot (pixel) at the given scanline and dot position
func (p *PPU) renderDot(scanline, dot int) {
	x := dot
	y := scanline

	// Render background layers (BG3 → BG0, back to front)
	if p.BG3.Enabled {
		p.renderDotBackgroundLayer(3, x, y)
	}
	if p.BG2.Enabled {
		p.renderDotBackgroundLayer(2, x, y)
	}
	if p.BG1.Enabled {
		p.renderDotBackgroundLayer(1, x, y)
	}
	if p.BG0.Enabled {
		if p.MatrixEnabled {
			p.renderDotMatrixMode(x, y)
		} else {
			p.renderDotBackgroundLayer(0, x, y)
		}
	}

	// Render sprites (for this scanline)
	p.renderDotSprites(x, y)
}

// renderDotBackgroundLayer renders a single dot for a background layer
func (p *PPU) renderDotBackgroundLayer(layerNum, x, y int) {
	// Get layer
	var layer *BackgroundLayer
	switch layerNum {
	case 0:
		layer = &p.BG0
	case 1:
		layer = &p.BG1
	case 2:
		layer = &p.BG2
	case 3:
		layer = &p.BG3
	default:
		return
	}

	if !layer.Enabled {
		return
	}

	// Check windowing
	if !p.isPixelInWindow(x, y, layerNum) {
		return
	}

	// Calculate tilemap coordinates with scroll
	worldX := int(x) + int(layer.ScrollX)
	worldY := int(y) + int(layer.ScrollY)

	// Tile size: 8x8 or 16x16
	tileSize := 8
	if layer.TileSize {
		tileSize = 16
	}

	// Wrap coordinates (tilemap repeats)
	tilemapWidth := 32
	tilemapPixelWidth := tilemapWidth * tileSize
	tilemapPixelHeight := tilemapWidth * tileSize
	worldX = worldX % tilemapPixelWidth
	if worldX < 0 {
		worldX += tilemapPixelWidth
	}
	worldY = worldY % tilemapPixelHeight
	if worldY < 0 {
		worldY += tilemapPixelHeight
	}

	// Calculate which tile this pixel is in
	tileX := worldX / tileSize
	tileY := worldY / tileSize

	// Calculate pixel position within tile
	pixelXInTile := worldX % tileSize
	pixelYInTile := worldY % tileSize

	// Read tilemap entry
	tilemapBase := uint16(0x4000)
	if layer.TilemapBase != 0 {
		tilemapBase = layer.TilemapBase
	}
	tilemapOffset := uint16((tileY*tilemapWidth + tileX) * 2)
	if uint32(tilemapBase)+uint32(tilemapOffset) >= 65536 {
		return
	}
	tilemapEntryAddr := tilemapBase + tilemapOffset

	// Read tile index and attributes
	tileIndex := uint8(p.VRAM[tilemapEntryAddr])
	attributes := uint8(p.VRAM[tilemapEntryAddr+1])
	paletteIndex := attributes & 0x0F
	flipX := (attributes & 0x10) != 0
	flipY := (attributes & 0x20) != 0

	// Apply flip
	if flipX {
		pixelXInTile = tileSize - 1 - pixelXInTile
	}
	if flipY {
		pixelYInTile = tileSize - 1 - pixelYInTile
	}

	// Read tile data
	tileDataOffset := uint16(tileIndex) * uint16(tileSize*tileSize/2)
	pixelOffsetInTile := pixelYInTile*tileSize + pixelXInTile
	byteOffsetInTile := pixelOffsetInTile / 2
	pixelInByte := pixelOffsetInTile % 2

	if uint32(tileDataOffset)+uint32(byteOffsetInTile) >= 65536 {
		return
	}
	tileDataAddr := tileDataOffset + uint16(byteOffsetInTile)

	// Read pixel color index
	tileByte := p.VRAM[tileDataAddr]
	var colorIndex uint8
	if pixelInByte == 0 {
		colorIndex = (tileByte >> 4) & 0x0F
	} else {
		colorIndex = tileByte & 0x0F
	}

	// Look up color in CGRAM
	color := p.getColorFromCGRAM(paletteIndex, colorIndex)
	p.OutputBuffer[y*320+x] = color
}

// renderDotMatrixMode renders a single dot for Matrix Mode
func (p *PPU) renderDotMatrixMode(x, y int) {
	// TODO: Implement Matrix Mode transformation
	// For now, just render BG0 normally
	p.renderDotBackgroundLayer(0, x, y)
}

// renderDotSprites renders sprites for a single dot
func (p *PPU) renderDotSprites(x, y int) {
	// Render sprites (128 max)
	// For now, render all sprites that overlap this pixel
	// In a more optimized version, we could pre-calculate sprite positions per scanline
	for spriteIndex := 0; spriteIndex < 128; spriteIndex++ {
		oamAddr := spriteIndex * 6

		// Read sprite data
		xLow := uint8(p.OAM[oamAddr])
		xHigh := uint8(p.OAM[oamAddr+1])
		spriteX := int(xLow)
		if (xHigh & 0x01) != 0 {
			spriteX |= 0xFFFFFF00
		}

		spriteY := int(p.OAM[oamAddr+2])
		tileIndex := uint8(p.OAM[oamAddr+3])
		attributes := uint8(p.OAM[oamAddr+4])
		paletteIndex := attributes & 0x0F
		flipX := (attributes & 0x10) != 0
		flipY := (attributes & 0x20) != 0

		control := uint8(p.OAM[oamAddr+5])
		enabled := (control & 0x01) != 0
		tileSize16 := (control & 0x02) != 0

		if !enabled {
			continue
		}

		spriteSize := 8
		if tileSize16 {
			spriteSize = 16
		}

		// Check if this pixel is within sprite bounds
		if x < spriteX || x >= spriteX+spriteSize || y < spriteY || y >= spriteY+spriteSize {
			continue
		}

		// Calculate tile coordinates
		px := x - spriteX
		py := y - spriteY

		// Apply flip
		tileX := px
		tileY := py
		if flipX {
			tileX = spriteSize - 1 - tileX
		}
		if flipY {
			tileY = spriteSize - 1 - tileY
		}

		// Read tile data
		tileDataOffset := uint16(tileIndex) * uint16(spriteSize*spriteSize/2)
		pixelOffsetInTile := tileY*spriteSize + tileX
		byteOffsetInTile := pixelOffsetInTile / 2
		pixelInByte := pixelOffsetInTile % 2

		if uint32(tileDataOffset)+uint32(byteOffsetInTile) >= 65536 {
			continue
		}
		tileDataAddr := tileDataOffset + uint16(byteOffsetInTile)

		// Read pixel color index
		tileByte := p.VRAM[tileDataAddr]
		var colorIndex uint8
		if pixelInByte == 0 {
			colorIndex = (tileByte >> 4) & 0x0F
		} else {
			colorIndex = tileByte & 0x0F
		}

		// Color index 0 is transparent for sprites
		if colorIndex == 0 {
			continue
		}

		// Look up color and render (sprites render on top)
		color := p.getColorFromCGRAM(paletteIndex, colorIndex)
		p.OutputBuffer[y*320+x] = color
		break // Only render first sprite (priority handling can be added later)
	}
}

// updateHDMA updates HDMA scroll values for the current scanline
func (p *PPU) updateHDMA(scanline int) {
	if !p.HDMAEnabled {
		return
	}

	// Read HDMA table entry for this scanline
	// HDMA table format: each entry is 2 bytes (scrollX, scrollY) per scanline
	tableAddr := uint32(p.HDMATableBase) + uint32(scanline*2)
	if tableAddr+1 < 65536 {
		tableAddrU16 := uint16(tableAddr)
		scrollX := int16(uint16(p.VRAM[tableAddrU16]) | (uint16(p.VRAM[tableAddrU16+1]) << 8))
		// Apply HDMA scroll to BG0 (or configured layer)
		p.BG0.ScrollX = scrollX
		// Can extend to other layers if needed
	}
}
