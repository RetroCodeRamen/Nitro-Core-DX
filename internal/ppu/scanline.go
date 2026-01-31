package ppu

// ScanlineRenderer handles scanline-by-scanline and dot-by-dot rendering
// This replaces the frame-based RenderFrame() for clock-driven operation

// PPU timing constants (at ~7.67 MHz CPU clock, Genesis-like)
const (
	// Display resolution
	ScreenWidth  = 320
	ScreenHeight = 200

	// Timing (at ~7.67 MHz, synchronized with CPU)
	// Each scanline takes: 320 dots (visible) + 261 dots (hblank) = 581 dots
	// Each dot = 1 CPU cycle at ~7.67 MHz
	// Total: 220 scanlines × 581 dots = 127,820 cycles per frame
	// At 60 FPS: 7,669,200 Hz ≈ 7.67 MHz (Genesis-like speed)
	DotsPerScanline = 581 // Changed from 360 to match Genesis-like CPU speed
	VisibleDots     = 320 // Keep same (visible pixels)
	HBlankDots      = 261 // Changed from 40 (581 - 320)

	// Frame timing
	// Visible scanlines: 200
	// VBlank scanlines: 20 (total 220 scanlines per frame)
	VisibleScanlines = 200
	VBlankScanlines  = 20
	TotalScanlines   = 220
)

// StepPPU steps the PPU by a number of cycles (clock-driven)
// Optimized version: processes scanlines in batches for better performance
// Returns error if any
func (p *PPU) StepPPU(cycles uint64) error {
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
		p.FrameComplete = false
	}

	cyclesRemaining := cycles
	for cyclesRemaining > 0 {
		// Execute DMA (cycle-accurate: one byte per cycle)
		if p.DMAEnabled {
			p.stepDMA()
		}

		// Handle HDMA on scanline start (before processing dots)
		if p.currentDot == 0 && p.HDMAEnabled && p.currentScanline < VisibleScanlines {
			p.updateHDMA(p.currentScanline)
		}

		// Calculate cycles until end of current scanline
		cyclesUntilScanlineEnd := uint64(DotsPerScanline - p.currentDot)
		if cyclesUntilScanlineEnd > cyclesRemaining {
			cyclesUntilScanlineEnd = cyclesRemaining
		}

		// Process visible dots in current scanline
		if p.currentScanline < VisibleScanlines {
			// Render all visible dots in this batch
			for p.currentDot < VisibleDots && cyclesUntilScanlineEnd > 0 {
				p.renderDot(p.currentScanline, p.currentDot)
				p.currentDot++
				cyclesUntilScanlineEnd--
				cyclesRemaining--
			}

			// Skip HBlank dots (just advance counter)
			if p.currentDot >= VisibleDots && cyclesUntilScanlineEnd > 0 {
				// We're in HBlank, just advance
				advance := int(cyclesUntilScanlineEnd)
				if advance > DotsPerScanline-p.currentDot {
					advance = DotsPerScanline - p.currentDot
				}
				p.currentDot += advance
				cyclesRemaining -= uint64(advance)
			}
		} else {
			// VBlank scanline - just advance counters
			advance := int(cyclesUntilScanlineEnd)
			if advance > DotsPerScanline-p.currentDot {
				advance = DotsPerScanline - p.currentDot
			}
			p.currentDot += advance
			cyclesRemaining -= uint64(advance)
		}

		// Check if we've reached end of scanline
		if p.currentDot >= DotsPerScanline {
			p.endScanline()

			// Set VBlank flag at end of last visible scanline (before incrementing)
			if p.currentScanline == VisibleScanlines-1 {
				p.VBlankFlag = true
				// Trigger VBlank interrupt (IRQ) if callback is set
				if p.InterruptCallback != nil {
					p.InterruptCallback(1) // INT_VBLANK = 1
				}
			}

			p.currentDot = 0
			p.currentScanline++

			// Check if frame is complete
			if p.currentScanline >= TotalScanlines {
				p.endFrame()
				p.currentScanline = 0
				p.frameStarted = false
			}
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
			// Trigger VBlank interrupt (IRQ) if callback is set
			if p.InterruptCallback != nil {
				p.InterruptCallback(1) // INT_VBLANK = 1
			}
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
// Implements proper priority handling: backgrounds and sprites are rendered together based on priority
// Priority system: BG3=3, BG2=2, BG1=1, BG0=0, Sprites=0-3 (from attributes)
// Lower priority value = renders first (so higher priority renders on top)
func (p *PPU) renderDot(scanline, dot int) {
	x := dot
	y := scanline

	// Collect all renderable elements (backgrounds and sprites) with their priorities
	type renderElement struct {
		priority    uint8
		elementType int         // 0=background, 1=sprite
		layerNum    int         // for backgrounds
		spriteInfo  *spriteInfo // for sprites
	}

	var elements []renderElement

	// Add background layers with their implicit priority (BG3=3, BG2=2, BG1=1, BG0=0)
	if p.BG3.Enabled {
		elements = append(elements, renderElement{
			priority:    3,
			elementType: 0,
			layerNum:    3,
		})
	}
	if p.BG2.Enabled {
		elements = append(elements, renderElement{
			priority:    2,
			elementType: 0,
			layerNum:    2,
		})
	}
	if p.BG1.Enabled {
		elements = append(elements, renderElement{
			priority:    1,
			elementType: 0,
			layerNum:    1,
		})
	}
	if p.BG0.Enabled {
		elements = append(elements, renderElement{
			priority:    0,
			elementType: 0,
			layerNum:    0,
		})
	}

	// Collect sprites that overlap this pixel
	sprites := p.collectSpritesAtPixel(x, y)
	for _, sprite := range sprites {
		elements = append(elements, renderElement{
			priority:    sprite.priority,
			elementType: 1,
			spriteInfo:  &sprite,
		})
	}

	// Sort by priority (lower priority = render first, so higher priority renders on top)
	// Within same priority, backgrounds render before sprites, and lower sprite index renders first
	for i := 0; i < len(elements); i++ {
		for j := i + 1; j < len(elements); j++ {
			swap := false
			if elements[i].priority > elements[j].priority {
				swap = true
			} else if elements[i].priority == elements[j].priority {
				// Same priority: backgrounds before sprites, lower sprite index first
				if elements[i].elementType == 1 && elements[j].elementType == 0 {
					swap = true // sprite should render after background
				} else if elements[i].elementType == 1 && elements[j].elementType == 1 {
					if elements[i].spriteInfo.index > elements[j].spriteInfo.index {
						swap = true
					}
				}
			}
			if swap {
				elements[i], elements[j] = elements[j], elements[i]
			}
		}
	}

	// Render elements in sorted order
	for _, elem := range elements {
		if elem.elementType == 0 {
			// Render background layer
			layerNum := elem.layerNum
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
			}

			if layer.MatrixEnabled || (layerNum == 0 && p.MatrixEnabled) {
				p.renderDotMatrixMode(layerNum, x, y)
			} else {
				p.renderDotBackgroundLayer(layerNum, x, y)
			}
		} else {
			// Render sprite
			p.renderDotSpritePixel(x, y, elem.spriteInfo)
		}
	}
}

// collectSpritesAtPixel collects all sprites that overlap the given pixel
func (p *PPU) collectSpritesAtPixel(x, y int) []spriteInfo {
	var sprites []spriteInfo

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

		// Extract priority (bits [7:6] of attributes)
		priority := (attributes >> 6) & 0x3

		// Add to sprite list
		sprites = append(sprites, spriteInfo{
			index:      spriteIndex,
			priority:   priority,
			x:          spriteX,
			y:          spriteY,
			size:       spriteSize,
			tileIndex:  tileIndex,
			attributes: attributes,
			control:    control,
		})
	}

	return sprites
}

// blendColor blends two colors based on blend mode and alpha
func (p *PPU) blendColor(foreground, background uint32, blendMode uint8, alpha uint8) uint32 {
	// Extract RGB components (RGB555 format: 0xRRRRRGGGGGBBBBB)
	fgR := uint8((foreground >> 10) & 0x1F)
	fgG := uint8((foreground >> 5) & 0x1F)
	fgB := uint8(foreground & 0x1F)

	bgR := uint8((background >> 10) & 0x1F)
	bgG := uint8((background >> 5) & 0x1F)
	bgB := uint8(background & 0x1F)

	var outR, outG, outB uint8

	// Normalize alpha from 0-15 to 0-31 (for better precision)
	alphaNorm := uint16(alpha) * 2 // 0-30

	switch blendMode {
	case 0: // Normal (opaque)
		outR = fgR
		outG = fgG
		outB = fgB

	case 1: // Alpha blending
		// Alpha blend: out = fg * alpha + bg * (1 - alpha)
		// alphaNorm is 0-30, so we use it as numerator with denominator 31
		alphaNum := uint16(alphaNorm)
		alphaDenom := uint16(31)
		invAlphaNum := alphaDenom - alphaNum

		outR = uint8((uint16(fgR)*alphaNum + uint16(bgR)*invAlphaNum) / alphaDenom)
		outG = uint8((uint16(fgG)*alphaNum + uint16(bgG)*invAlphaNum) / alphaDenom)
		outB = uint8((uint16(fgB)*alphaNum + uint16(bgB)*invAlphaNum) / alphaDenom)

	case 2: // Additive
		// Additive: out = bg + fg * alpha
		// Clamp to max (31 for 5-bit)
		fgRAdd := uint16(fgR) * uint16(alphaNorm) / 31
		fgGAdd := uint16(fgG) * uint16(alphaNorm) / 31
		fgBAdd := uint16(fgB) * uint16(alphaNorm) / 31

		outR = uint8(min(31, uint16(bgR)+fgRAdd))
		outG = uint8(min(31, uint16(bgG)+fgGAdd))
		outB = uint8(min(31, uint16(bgB)+fgBAdd))

	case 3: // Subtractive
		// Subtractive: out = bg - fg * alpha
		// Clamp to min (0)
		fgRSub := uint16(fgR) * uint16(alphaNorm) / 31
		fgGSub := uint16(fgG) * uint16(alphaNorm) / 31
		fgBSub := uint16(fgB) * uint16(alphaNorm) / 31

		outR = uint8(max(0, int16(bgR)-int16(fgRSub)))
		outG = uint8(max(0, int16(bgG)-int16(fgGSub)))
		outB = uint8(max(0, int16(bgB)-int16(fgBSub)))
	}

	// Reconstruct RGB555 color
	return uint32(outR)<<10 | uint32(outG)<<5 | uint32(outB)
}

// min returns the minimum of two uint16 values
func min(a, b uint16) uint16 {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two int16 values (clamped to 0)
func max(a, b int16) int16 {
	if a > b {
		return a
	}
	if b < 0 {
		return 0
	}
	return b
}

// renderDotSpritePixel renders a single sprite pixel with blending support
func (p *PPU) renderDotSpritePixel(x, y int, sprite *spriteInfo) {
	// Calculate tile coordinates
	px := x - sprite.x
	py := y - sprite.y

	// Apply flip
	flipX := (sprite.attributes & 0x10) != 0
	flipY := (sprite.attributes & 0x20) != 0
	tileX := px
	tileY := py
	if flipX {
		tileX = sprite.size - 1 - tileX
	}
	if flipY {
		tileY = sprite.size - 1 - tileY
	}

	// Read tile data
	tileDataOffset := uint16(sprite.tileIndex) * uint16(sprite.size*sprite.size/2)
	pixelOffsetInTile := tileY*sprite.size + tileX
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

	// Color index 0 is transparent for sprites
	if colorIndex == 0 {
		return
	}

	// Look up sprite color
	paletteIndex := sprite.attributes & 0x0F
	spriteColor := p.getColorFromCGRAM(paletteIndex, colorIndex)

	// Extract blend mode and alpha from control byte
	blendMode := (sprite.control >> 2) & 0x3 // Bits [3:2]
	alpha := (sprite.control >> 4) & 0xF     // Bits [7:4]

	// Apply blending
	if blendMode == 0 {
		// Normal mode (opaque) - ignore alpha, just write sprite color
		// This maintains backward compatibility with ROMs that use control byte 0x03
		p.OutputBuffer[y*320+x] = spriteColor
	} else {
		// Blending modes (alpha, additive, subtractive) - need background color
		backgroundColor := p.OutputBuffer[y*320+x]
		blendedColor := p.blendColor(spriteColor, backgroundColor, blendMode, alpha)
		p.OutputBuffer[y*320+x] = blendedColor
	}
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

	// Apply mosaic effect if enabled
	if layer.MosaicEnabled && layer.MosaicSize > 1 {
		// Calculate top-left pixel of mosaic block
		mosaicSize := int(layer.MosaicSize)
		mosaicBlockX := (x / mosaicSize) * mosaicSize
		mosaicBlockY := (y / mosaicSize) * mosaicSize

		// If this is not the top-left pixel, use the color from top-left
		if x != mosaicBlockX || y != mosaicBlockY {
			// Use color from top-left pixel of mosaic block
			if mosaicBlockY*320+mosaicBlockX < len(p.OutputBuffer) {
				color = p.OutputBuffer[mosaicBlockY*320+mosaicBlockX]
			}
		}
	}

	p.OutputBuffer[y*320+x] = color
}

// renderDotMatrixMode renders a single dot for Matrix Mode on a specific layer
// Implements Mode 7-style affine transformation
// layerNum: 0=BG0, 1=BG1, 2=BG2, 3=BG3
func (p *PPU) renderDotMatrixMode(layerNum, x, y int) {
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

	if !layer.Enabled || !layer.MatrixEnabled {
		return
	}

	// Check windowing
	if !p.isPixelInWindow(x, y, layerNum) {
		return
	}

	// Apply mirroring if enabled
	screenX := int16(x)
	screenY := int16(y)
	if layer.MatrixMirrorH {
		screenX = int16(ScreenWidth - 1 - x)
	}
	if layer.MatrixMirrorV {
		screenY = int16(ScreenHeight - 1 - y)
	}

	// Transform screen coordinates to tilemap coordinates
	// Formula: [x'] = [A B] × [x - CX]
	//          [y']   [C D]   [y - CY]
	// Where A, B, C, D are 8.8 fixed point (1.0 = 0x0100)

	// Calculate relative to center
	relX := int32(screenX) - int32(layer.MatrixCenterX)
	relY := int32(screenY) - int32(layer.MatrixCenterY)

	// Apply transformation matrix (8.8 fixed point multiplication)
	// Matrix values are int16 (8.8 fixed point)
	// Result needs to be divided by 256 (>> 8) to get integer result
	worldX := (int32(layer.MatrixA)*relX + int32(layer.MatrixB)*relY) >> 8
	worldY := (int32(layer.MatrixC)*relX + int32(layer.MatrixD)*relY) >> 8

	// Add layer scroll offset
	worldX += int32(layer.ScrollX)
	worldY += int32(layer.ScrollY)

	// Tile size: 8x8 or 16x16
	tileSize := 8
	if layer.TileSize {
		tileSize = 16
	}

	// Handle outside-screen coordinates based on MatrixOutsideMode
	tilemapWidth := 32
	tilemapPixelWidth := tilemapWidth * tileSize
	tilemapPixelHeight := tilemapWidth * tileSize

	// Check if coordinates are outside tilemap bounds
	worldXValid := worldX >= 0 && worldX < int32(tilemapPixelWidth)
	worldYValid := worldY >= 0 && worldY < int32(tilemapPixelHeight)

	if !worldXValid || !worldYValid {
		// Outside screen bounds - handle based on mode
		switch layer.MatrixOutsideMode {
		case 0: // Repeat/wrap mode (default)
			worldX = worldX % int32(tilemapPixelWidth)
			if worldX < 0 {
				worldX += int32(tilemapPixelWidth)
			}
			worldY = worldY % int32(tilemapPixelHeight)
			if worldY < 0 {
				worldY += int32(tilemapPixelHeight)
			}
		case 1: // Backdrop mode - render backdrop color
			// Use backdrop color (CGRAM palette 0, color 0)
			backdropColor := p.getColorFromCGRAM(0, 0)
			p.OutputBuffer[y*320+x] = backdropColor
			return
		case 2: // Character #0 mode - render tile 0
			// Force tile index to 0
			worldX = 0
			worldY = 0
		default:
			// Default to wrap
			worldX = worldX % int32(tilemapPixelWidth)
			if worldX < 0 {
				worldX += int32(tilemapPixelWidth)
			}
			worldY = worldY % int32(tilemapPixelHeight)
			if worldY < 0 {
				worldY += int32(tilemapPixelHeight)
			}
		}
	}

	// Calculate which tile this pixel is in
	tileX := int(worldX) / tileSize
	tileY := int(worldY) / tileSize

	// Calculate pixel position within tile
	pixelXInTile := int(worldX) % tileSize
	pixelYInTile := int(worldY) % tileSize

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

	// Apply mosaic effect if enabled
	if layer.MosaicEnabled && layer.MosaicSize > 1 {
		// Calculate top-left pixel of mosaic block
		mosaicSize := int(layer.MosaicSize)
		mosaicBlockX := (x / mosaicSize) * mosaicSize
		mosaicBlockY := (y / mosaicSize) * mosaicSize

		// If this is not the top-left pixel, use the color from top-left
		if x != mosaicBlockX || y != mosaicBlockY {
			// Use color from top-left pixel of mosaic block
			if mosaicBlockY*320+mosaicBlockX < len(p.OutputBuffer) {
				color = p.OutputBuffer[mosaicBlockY*320+mosaicBlockX]
			}
		}
	}

	p.OutputBuffer[y*320+x] = color
}

// spriteInfo holds sprite data for priority sorting
type spriteInfo struct {
	index      int
	priority   uint8
	x, y       int
	size       int
	tileIndex  uint8
	attributes uint8
	control    uint8
}

// renderDotSprites renders sprites for a single dot with proper priority handling
func (p *PPU) renderDotSprites(x, y int) {
	// Collect all sprites that overlap this pixel
	var sprites []spriteInfo

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

		// Extract priority (bits [7:6] of attributes)
		priority := (attributes >> 6) & 0x3

		// Add to sprite list
		sprites = append(sprites, spriteInfo{
			index:      spriteIndex,
			priority:   priority,
			x:          spriteX,
			y:          spriteY,
			size:       spriteSize,
			tileIndex:  tileIndex,
			attributes: attributes,
			control:    control,
		})
	}

	// Sort sprites by priority (lower priority value = render first, so higher priority sprites render on top)
	// Within same priority, lower sprite index = render first (so lower index renders on top)
	for i := 0; i < len(sprites); i++ {
		for j := i + 1; j < len(sprites); j++ {
			if sprites[i].priority > sprites[j].priority ||
				(sprites[i].priority == sprites[j].priority && sprites[i].index > sprites[j].index) {
				sprites[i], sprites[j] = sprites[j], sprites[i]
			}
		}
	}

	// Render sprites in sorted order (lowest priority first, so they get covered by higher priority sprites)
	for _, sprite := range sprites {
		// Calculate tile coordinates
		px := x - sprite.x
		py := y - sprite.y

		// Apply flip
		flipX := (sprite.attributes & 0x10) != 0
		flipY := (sprite.attributes & 0x20) != 0
		tileX := px
		tileY := py
		if flipX {
			tileX = sprite.size - 1 - tileX
		}
		if flipY {
			tileY = sprite.size - 1 - tileY
		}

		// Read tile data
		tileDataOffset := uint16(sprite.tileIndex) * uint16(sprite.size*sprite.size/2)
		pixelOffsetInTile := tileY*sprite.size + tileX
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

		// Look up color and render
		paletteIndex := sprite.attributes & 0x0F
		color := p.getColorFromCGRAM(paletteIndex, colorIndex)
		p.OutputBuffer[y*320+x] = color
		// Don't break - continue to render higher priority sprites on top
	}
}

// updateHDMA updates HDMA scroll and matrix values for the current scanline
// HDMA table format (per scanline, per layer):
//   - If layer has Matrix Mode enabled: 14 bytes per layer
//     [ScrollX_L, ScrollX_H, ScrollY_L, ScrollY_H,
//     MatrixA_L, MatrixA_H, MatrixB_L, MatrixB_H,
//     MatrixC_L, MatrixC_H, MatrixD_L, MatrixD_H,
//     CenterX_L, CenterX_H, CenterY_L, CenterY_H]
//   - If layer has normal mode: 4 bytes per layer
//     [ScrollX_L, ScrollX_H, ScrollY_L, ScrollY_H]
//
// Table is organized by layer: BG0, BG1, BG2, BG3
func (p *PPU) updateHDMA(scanline int) {
	if !p.HDMAEnabled || scanline >= VisibleScanlines {
		return
	}

	// Calculate base address for this scanline
	// Each scanline entry contains data for all enabled layers
	tableAddr := uint32(p.HDMATableBase) + uint32(scanline*64) // Max 64 bytes per scanline (4 layers × 16 bytes)

	// Update each layer based on HDMA control bits
	for layerNum := 0; layerNum < 4; layerNum++ {
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
			continue
		}

		// Check if this layer is enabled for HDMA (bits 1-4 of HDMAControl)
		layerHDMAEnabled := (p.HDMAControl & (1 << (layerNum + 1))) != 0
		if !layerHDMAEnabled {
			continue
		}

		// Calculate offset for this layer (16 bytes per layer)
		layerOffset := uint32(layerNum * 16)
		addr := tableAddr + layerOffset

		if addr+15 < 65536 {
			addrU16 := uint16(addr)

			// Always read scroll values (first 4 bytes)
			scrollX := int16(uint16(p.VRAM[addrU16]) | (uint16(p.VRAM[addrU16+1]) << 8))
			scrollY := int16(uint16(p.VRAM[addrU16+2]) | (uint16(p.VRAM[addrU16+3]) << 8))
			layer.ScrollX = scrollX
			layer.ScrollY = scrollY

			// If layer has Matrix Mode enabled, read matrix parameters (next 12 bytes)
			if layer.MatrixEnabled {
				// Matrix A (8.8 fixed point)
				matrixA := int16(uint16(p.VRAM[addrU16+4]) | (uint16(p.VRAM[addrU16+5]) << 8))
				// Matrix B (8.8 fixed point)
				matrixB := int16(uint16(p.VRAM[addrU16+6]) | (uint16(p.VRAM[addrU16+7]) << 8))
				// Matrix C (8.8 fixed point)
				matrixC := int16(uint16(p.VRAM[addrU16+8]) | (uint16(p.VRAM[addrU16+9]) << 8))
				// Matrix D (8.8 fixed point)
				matrixD := int16(uint16(p.VRAM[addrU16+10]) | (uint16(p.VRAM[addrU16+11]) << 8))
				// Center X
				centerX := int16(uint16(p.VRAM[addrU16+12]) | (uint16(p.VRAM[addrU16+13]) << 8))
				// Center Y
				centerY := int16(uint16(p.VRAM[addrU16+14]) | (uint16(p.VRAM[addrU16+15]) << 8))

				// Update matrix parameters
				layer.MatrixA = matrixA
				layer.MatrixB = matrixB
				layer.MatrixC = matrixC
				layer.MatrixD = matrixD
				layer.MatrixCenterX = centerX
				layer.MatrixCenterY = centerY

				// Store for debug
				p.HDMAMatrixA[layerNum][scanline] = matrixA
				p.HDMAMatrixB[layerNum][scanline] = matrixB
				p.HDMAMatrixC[layerNum][scanline] = matrixC
				p.HDMAMatrixD[layerNum][scanline] = matrixD
				p.HDMAMatrixCX[layerNum][scanline] = centerX
				p.HDMAMatrixCY[layerNum][scanline] = centerY
			}

			// Store scroll for debug
			p.HDMAScrollX[layerNum][scanline] = scrollX
			p.HDMAScrollY[layerNum][scanline] = scrollY
		}
	}
}
