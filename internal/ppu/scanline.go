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

const (
	hdmaLayerPayloadBytes      = 16
	hdmaBaseScanlineBytes      = 4 * hdmaLayerPayloadBytes
	hdmaRebindTablePresent     = 0x20
	hdmaPriorityTablePresent   = 0x40
	hdmaTilemapTablePresent    = 0x80
	hdmaExtSourceModePresent   = 0x01
	hdmaRebindSentinelKeep     = 0xFF
	hdmaPrioritySentinelKeep   = 0xFF
	hdmaTilemapSentinelKeep    = 0xFFFF
	hdmaSourceModeSentinelKeep = 0xFF
)

type scanlineLayerCommand struct {
	LayerEnabled     bool
	ApplyScroll      bool
	ScrollX          int16
	ScrollY          int16
	ApplyTransform   bool
	TransformA       int16
	TransformB       int16
	TransformC       int16
	TransformD       int16
	CenterX          int16
	CenterY          int16
	ApplyRebind      bool
	TransformBinding uint8
	ApplyPriority    bool
	Priority         uint8
	ApplyTilemapBase bool
	TilemapBase      uint16
	ApplySourceMode  bool
	SourceMode       uint8
}

type scanlineCommandSet struct {
	Layers [4]scanlineLayerCommand
}

func (p *PPU) prepareLiveFloorRow(channelIndex uint8, scanline int) bool {
	if int(channelIndex) >= len(p.MatrixPlanes) || scanline < 0 || scanline >= VisibleScanlines {
		return false
	}
	plane := &p.MatrixPlanes[channelIndex]
	if !plane.Enabled || plane.SourceMode != MatrixPlaneSourceBitmap || !plane.LiveFloorEnabled {
		p.MatrixFloorRows[channelIndex] = matrixFloorRowCache{}
		return false
	}
	if scanline < int(plane.LiveFloorHorizon) {
		p.MatrixFloorRows[channelIndex] = matrixFloorRowCache{}
		return false
	}
	cache := &p.MatrixFloorRows[channelIndex]
	if cache.valid && cache.scanline == scanline {
		return true
	}

	row := int64(scanline-int(plane.LiveFloorHorizon)) + 1
	headingX := int64(plane.LiveFloorHeadingX)
	headingY := int64(plane.LiveFloorHeadingY)
	if headingX == 0 && headingY == 0 {
		headingY = -0x0100
	}

	cameraX16 := int64(plane.LiveFloorCameraX) << 16
	cameraY16 := int64(plane.LiveFloorCameraY) << 16
	headingX16 := headingX << 8
	headingY16 := headingY << 8
	rightX16 := -headingY16
	rightY16 := headingX16

	// These constants intentionally mirror the earlier ROM-side floor model, but
	// move the work into the PPU so the ROM only updates camera state each frame.
	forward16 := (int64(3072) << 16) / (row + 6)
	step16 := (int64(104858) * 18) / (row + 18) // about 1.6 px at the near rows
	if step16 < 5243 {
		step16 = 5243 // about 0.08 px
	}

	rowCenterX16 := cameraX16 + int64((headingX16*forward16)>>16)
	rowCenterY16 := cameraY16 + int64((headingY16*forward16)>>16)
	stepX16 := int32((rightX16 * step16) >> 16)
	stepY16 := int32((rightY16 * step16) >> 16)
	startX16 := int32(rowCenterX16 - int64(stepX16*(ScreenWidth/2)))
	startY16 := int32(rowCenterY16 - int64(stepY16*(ScreenWidth/2)))

	*cache = matrixFloorRowCache{
		valid:    true,
		scanline: scanline,
		startX:   startX16,
		startY:   startY16,
		stepX:    stepX16,
		stepY:    stepY16,
	}
	return true
}

func (p *PPU) prepareLiveFloorRowsForScanline(scanline int) {
	for channelIndex := range p.MatrixPlanes {
		_ = p.prepareLiveFloorRow(uint8(channelIndex), scanline)
	}
}

func (p *PPU) hdmaScanlineStride() uint32 {
	stride := uint32(hdmaBaseScanlineBytes)
	if (p.HDMAControl & hdmaRebindTablePresent) != 0 {
		stride += 4 // One binding byte per layer
	}
	if (p.HDMAControl & hdmaPriorityTablePresent) != 0 {
		stride += 4 // One priority byte per layer
	}
	if (p.HDMAControl & hdmaTilemapTablePresent) != 0 {
		stride += 8 // One uint16 tilemap base per layer
	}
	if (p.HDMAExtControl & hdmaExtSourceModePresent) != 0 {
		stride += 4 // One source-mode byte per layer
	}
	return stride
}

func (p *PPU) decodeScanlineCommands(scanline int) scanlineCommandSet {
	var commands scanlineCommandSet
	tableAddr := uint32(p.HDMATableBase) + uint32(scanline)*p.hdmaScanlineStride()
	rebindBase := tableAddr + hdmaBaseScanlineBytes
	priorityBase := rebindBase
	if (p.HDMAControl & hdmaRebindTablePresent) != 0 {
		priorityBase += 4
	}
	tilemapBase := priorityBase
	if (p.HDMAControl & hdmaPriorityTablePresent) != 0 {
		tilemapBase += 4
	}
	sourceModeBase := tilemapBase
	if (p.HDMAControl & hdmaTilemapTablePresent) != 0 {
		sourceModeBase += 8
	}

	for layerNum := 0; layerNum < 4; layerNum++ {
		layerEnabled := (p.HDMAControl & (1 << (layerNum + 1))) != 0
		command := &commands.Layers[layerNum]
		command.LayerEnabled = layerEnabled
		if !layerEnabled {
			continue
		}

		addr := tableAddr + uint32(layerNum*hdmaLayerPayloadBytes)
		if addr+15 >= 65536 {
			continue
		}
		addrU16 := uint16(addr)

		command.ApplyScroll = true
		command.ScrollX = int16(uint16(p.VRAM[addrU16]) | (uint16(p.VRAM[addrU16+1]) << 8))
		command.ScrollY = int16(uint16(p.VRAM[addrU16+2]) | (uint16(p.VRAM[addrU16+3]) << 8))
		command.ApplyTransform = true
		command.TransformA = int16(uint16(p.VRAM[addrU16+4]) | (uint16(p.VRAM[addrU16+5]) << 8))
		command.TransformB = int16(uint16(p.VRAM[addrU16+6]) | (uint16(p.VRAM[addrU16+7]) << 8))
		command.TransformC = int16(uint16(p.VRAM[addrU16+8]) | (uint16(p.VRAM[addrU16+9]) << 8))
		command.TransformD = int16(uint16(p.VRAM[addrU16+10]) | (uint16(p.VRAM[addrU16+11]) << 8))
		command.CenterX = int16(uint16(p.VRAM[addrU16+12]) | (uint16(p.VRAM[addrU16+13]) << 8))
		command.CenterY = int16(uint16(p.VRAM[addrU16+14]) | (uint16(p.VRAM[addrU16+15]) << 8))

		if (p.HDMAControl & hdmaRebindTablePresent) != 0 {
			rebindAddr := rebindBase + uint32(layerNum)
			if rebindAddr < 65536 {
				binding := p.VRAM[uint16(rebindAddr)]
				if binding != hdmaRebindSentinelKeep {
					command.ApplyRebind = true
					command.TransformBinding = binding & 0x03
				}
			}
		}
		if (p.HDMAControl & hdmaPriorityTablePresent) != 0 {
			priorityAddr := priorityBase + uint32(layerNum)
			if priorityAddr < 65536 {
				priority := p.VRAM[uint16(priorityAddr)]
				if priority != hdmaPrioritySentinelKeep {
					command.ApplyPriority = true
					command.Priority = priority & 0x03
				}
			}
		}
		if (p.HDMAControl & hdmaTilemapTablePresent) != 0 {
			layerTilemapAddr := tilemapBase + uint32(layerNum*2)
			if layerTilemapAddr+1 < 65536 {
				tilemapBaseValue := uint16(p.VRAM[uint16(layerTilemapAddr)]) | (uint16(p.VRAM[uint16(layerTilemapAddr+1)]) << 8)
				if tilemapBaseValue != hdmaTilemapSentinelKeep {
					command.ApplyTilemapBase = true
					command.TilemapBase = tilemapBaseValue
				}
			}
		}
		if (p.HDMAExtControl & hdmaExtSourceModePresent) != 0 {
			sourceModeAddr := sourceModeBase + uint32(layerNum)
			if sourceModeAddr < 65536 {
				sourceMode := p.VRAM[uint16(sourceModeAddr)]
				if sourceMode != hdmaSourceModeSentinelKeep {
					command.ApplySourceMode = true
					command.SourceMode = sourceMode & 0x01
				}
			}
		}
	}

	return commands
}

func (p *PPU) applyScanlineCommands(scanline int, commands scanlineCommandSet) {
	for layerNum := 0; layerNum < 4; layerNum++ {
		command := commands.Layers[layerNum]
		if !command.LayerEnabled {
			continue
		}

		layer := p.getBackgroundLayer(layerNum)
		if layer == nil {
			continue
		}

		if command.ApplyRebind {
			layer.TransformChannel = command.TransformBinding & 0x03
		}
		if command.ApplyPriority {
			layer.Priority = command.Priority & 0x03
		}
		if command.ApplyTilemapBase {
			layer.TilemapBase = command.TilemapBase
		}
		if command.ApplySourceMode {
			layer.SourceMode = command.SourceMode & 0x01
		}
		layer, channel := p.resolveLayerTransformChannel(layerNum)
		if layer == nil || channel == nil {
			continue
		}

		if command.ApplyScroll {
			layer.ScrollX = command.ScrollX
			layer.ScrollY = command.ScrollY
			p.HDMAScrollX[layerNum][scanline] = command.ScrollX
			p.HDMAScrollY[layerNum][scanline] = command.ScrollY
		}

		if command.ApplyTransform && channel.Enabled {
			channel.A = command.TransformA
			channel.B = command.TransformB
			channel.C = command.TransformC
			channel.D = command.TransformD
			channel.CenterX = command.CenterX
			channel.CenterY = command.CenterY
			p.HDMAMatrixA[layerNum][scanline] = command.TransformA
			p.HDMAMatrixB[layerNum][scanline] = command.TransformB
			p.HDMAMatrixC[layerNum][scanline] = command.TransformC
			p.HDMAMatrixD[layerNum][scanline] = command.TransformD
			p.HDMAMatrixCX[layerNum][scanline] = command.CenterX
			p.HDMAMatrixCY[layerNum][scanline] = command.CenterY
		}
	}
}

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
		// Handle HDMA on scanline start (before processing dots)
		if p.currentDot == 0 && p.HDMAEnabled && p.currentScanline < VisibleScanlines {
			p.updateHDMA(p.currentScanline)
		}
		// Hardware-like sprite evaluation stage at scanline start.
		if p.currentDot == 0 && p.currentScanline < VisibleScanlines {
			p.evaluateSpritesForScanline(p.currentScanline)
			p.prepareLiveFloorRowsForScanline(p.currentScanline)
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
				// Execute DMA (cycle-accurate: one byte per cycle)
				if p.DMAEnabled {
					p.stepDMA()
				}
				p.renderDot(p.currentScanline, p.currentDot)
				p.currentDot++
				cyclesUntilScanlineEnd--
				cyclesRemaining--
			}

			// Skip HBlank dots (just advance counter)
			if p.currentDot >= VisibleDots && cyclesUntilScanlineEnd > 0 {
				// We're in HBlank, just advance
				// Execute DMA for each HBlank cycle
				for cyclesUntilScanlineEnd > 0 {
					if p.DMAEnabled {
						p.stepDMA()
					}
					p.currentDot++
					cyclesUntilScanlineEnd--
					cyclesRemaining--
				}
			}
		} else {
			// VBlank scanline - just advance counters
			// Execute DMA for each VBlank cycle
			for cyclesUntilScanlineEnd > 0 {
				if p.DMAEnabled {
					p.stepDMA()
				}
				p.currentDot++
				cyclesUntilScanlineEnd--
				cyclesRemaining--
			}
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
	// Hardware-like sprite evaluation stage at scanline start.
	if p.currentDot == 0 && p.currentScanline < VisibleScanlines {
		p.evaluateSpritesForScanline(p.currentScanline)
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
	// Render buffered text commands on top of the finished frame
	for i := 0; i < p.textCount; i++ {
		cmd := &p.textCmds[i]
		p.drawChar(cmd.char, cmd.x, cmd.y, cmd.color)
	}
	p.textCount = 0

	// Copy to display buffer (front buffer) so startFrame can safely clear the back buffer
	copy(p.DisplayBuffer[:], p.OutputBuffer[:])

	p.FrameComplete = true
}

// renderDot renders a single dot (pixel) at the given scanline and dot position
// Implements proper priority handling: backgrounds and sprites are rendered together based on priority
// Priority system: BG3=3, BG2=2, BG1=1, BG0=0, Sprites=0-3 (from attributes)
// Lower priority value = renders first (so higher priority renders on top)
func (p *PPU) renderDot(scanline, dot int) {
	x := dot
	y := scanline

	// Collect all renderable elements (backgrounds and sprites) with their priorities.
	// Reuse scratch storage to avoid per-pixel allocations.
	elements := p.renderElementScratch[:0]

	// Add background layers with their explicit layer priority.
	if p.BG3.Enabled {
		elements = append(elements, renderElement{
			priority:    p.BG3.Priority,
			elementType: 0,
			layerNum:    3,
		})
	}
	if p.BG2.Enabled {
		elements = append(elements, renderElement{
			priority:    p.BG2.Priority,
			elementType: 0,
			layerNum:    2,
		})
	}
	if p.BG1.Enabled {
		elements = append(elements, renderElement{
			priority:    p.BG1.Priority,
			elementType: 0,
			layerNum:    1,
		})
	}
	if p.BG0.Enabled {
		elements = append(elements, renderElement{
			priority:    p.BG0.Priority,
			elementType: 0,
			layerNum:    0,
		})
	}

	// Collect sprites that overlap this pixel
	spriteCount := p.collectSpritesAtPixel(x, y, p.spriteScratch[:])
	for i := 0; i < spriteCount; i++ {
		elements = append(elements, renderElement{
			priority:    p.spriteScratch[i].priority,
			elementType: 1,
			spriteIndex: i,
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
					if p.spriteScratch[elements[i].spriteIndex].index > p.spriteScratch[elements[j].spriteIndex].index {
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
			layer, channel := p.resolveLayerTransformChannel(layerNum)
			if layer == nil || channel == nil {
				continue
			}

			if channel.Enabled {
				p.renderDotMatrixMode(layerNum, x, y)
			} else {
				p.renderDotBackgroundLayer(layerNum, x, y)
			}
		} else {
			// Render sprite
			p.renderDotSpritePixel(x, y, &p.spriteScratch[elem.spriteIndex])
		}
	}
}

// collectSpritesAtPixel collects all sprites that overlap the given pixel
func (p *PPU) collectSpritesAtPixel(x, y int, sprites []spriteInfo) int {
	count := 0
	active := p.activeScanlineSprites[:p.activeScanlineSpriteCount]
	// Fallback for non-sequential callers (tests/debug helpers) that invoke renderDot
	// without stepping through the scanline pipeline first.
	if p.activeScanlineY != y {
		p.evaluateSpritesForScanline(y)
		active = p.activeScanlineSprites[:p.activeScanlineSpriteCount]
	}

	for i := range active {
		s := active[i]

		// Scanline membership is already pre-evaluated. Only X bounds need per-pixel check.
		if x < s.x || x >= s.x+s.size {
			continue
		}

		// Add to sprite list
		if count >= len(sprites) {
			break
		}
		sprites[count] = s
		count++
	}
	return count
}

// evaluateSpritesForScanline builds the list of enabled sprites overlapping a scanline.
// This mirrors a hardware sprite evaluation stage and avoids per-pixel full OAM scans.
func (p *PPU) evaluateSpritesForScanline(y int) {
	count := 0
	for spriteIndex := 0; spriteIndex < 128; spriteIndex++ {
		oamAddr := spriteIndex * 6

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
		if !enabled {
			continue
		}

		spriteSize := 8
		if (control & 0x02) != 0 {
			spriteSize = 16
		}
		if y < spriteY || y >= spriteY+spriteSize {
			continue
		}
		if count >= len(p.activeScanlineSprites) {
			break
		}

		p.activeScanlineSprites[count] = spriteInfo{
			index:      spriteIndex,
			priority:   (attributes >> 6) & 0x3,
			x:          spriteX,
			y:          spriteY,
			size:       spriteSize,
			tileIndex:  tileIndex,
			attributes: attributes,
			control:    control,
		}
		count++
	}
	p.activeScanlineSpriteCount = count
	p.activeScanlineY = y
}

// blendColor blends two colors based on blend mode and alpha
func (p *PPU) blendColor(foreground, background uint32, blendMode uint8, alpha uint8) uint32 {
	// OutputBuffer stores RGB888 (0xRRGGBB), so blending must operate in 8-bit channels.
	fgR := uint8((foreground >> 16) & 0xFF)
	fgG := uint8((foreground >> 8) & 0xFF)
	fgB := uint8(foreground & 0xFF)

	bgR := uint8((background >> 16) & 0xFF)
	bgG := uint8((background >> 8) & 0xFF)
	bgB := uint8(background & 0xFF)

	var outR, outG, outB uint8

	// Normalize alpha from 0-15 to 0-255 for RGB888 blending.
	alphaNum := uint16(alpha) * 17 // 0..255
	alphaDenom := uint16(255)

	switch blendMode {
	case 0: // Normal (opaque)
		outR = fgR
		outG = fgG
		outB = fgB

	case 1: // Alpha blending
		// Alpha blend: out = fg * alpha + bg * (1 - alpha)
		invAlphaNum := alphaDenom - alphaNum

		outR = uint8((uint16(fgR)*alphaNum + uint16(bgR)*invAlphaNum) / alphaDenom)
		outG = uint8((uint16(fgG)*alphaNum + uint16(bgG)*invAlphaNum) / alphaDenom)
		outB = uint8((uint16(fgB)*alphaNum + uint16(bgB)*invAlphaNum) / alphaDenom)

	case 2: // Additive
		// Additive: out = bg + fg * alpha
		// Clamp to max 255 for 8-bit channels.
		fgRAdd := uint16(fgR) * alphaNum / alphaDenom
		fgGAdd := uint16(fgG) * alphaNum / alphaDenom
		fgBAdd := uint16(fgB) * alphaNum / alphaDenom

		outR = uint8(min(255, uint16(bgR)+fgRAdd))
		outG = uint8(min(255, uint16(bgG)+fgGAdd))
		outB = uint8(min(255, uint16(bgB)+fgBAdd))

	case 3: // Subtractive
		// Subtractive: out = bg - fg * alpha
		// Clamp to min (0)
		fgRSub := uint16(fgR) * alphaNum / alphaDenom
		fgGSub := uint16(fgG) * alphaNum / alphaDenom
		fgBSub := uint16(fgB) * alphaNum / alphaDenom

		outR = uint8(max(0, int16(bgR)-int16(fgRSub)))
		outG = uint8(max(0, int16(bgG)-int16(fgGSub)))
		outB = uint8(max(0, int16(bgB)-int16(fgBSub)))
	}

	// Reconstruct RGB888 color
	return uint32(outR)<<16 | uint32(outG)<<8 | uint32(outB)
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

	// Tile size and tilemap dimensions are powers of two, so wrapping/division can use
	// masks/shifts (hardware-friendly and faster than modulo/division).
	tileSize := 8
	tileShift := 3              // 8x8 tiles
	tileMask := 0x07            // pixel within tile
	tileBytesShift := uint16(5) // 8*8/2 = 32 bytes
	if layer.TileSize {
		tileSize = 16
		tileShift = 4 // 16x16 tiles
		tileMask = 0x0F
		tileBytesShift = 7 // 16*16/2 = 128 bytes
	}
	tilemapWidthTiles := 32
	switch layer.TilemapSize {
	case TilemapSize64x64:
		tilemapWidthTiles = 64
	case TilemapSize128x128:
		tilemapWidthTiles = 128
	}
	tilemapMask := (tilemapWidthTiles << tileShift) - 1

	// Wrap coordinates (tilemap repeats)
	worldX &= tilemapMask
	worldY &= tilemapMask

	// Calculate which tile this pixel is in
	tileX := worldX >> tileShift
	tileY := worldY >> tileShift

	// Calculate pixel position within tile
	pixelXInTile := worldX & tileMask
	pixelYInTile := worldY & tileMask

	// Read tilemap entry
	tilemapBase := uint16(0x4000)
	if layer.TilemapBase != 0 {
		tilemapBase = layer.TilemapBase
	}
	tilemapOffset := uint16((tileY*tilemapWidthTiles + tileX) * 2)
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
	tileDataOffset := uint16(tileIndex) << tileBytesShift
	pixelOffsetInTile := (pixelYInTile << tileShift) | pixelXInTile
	byteOffsetInTile := pixelOffsetInTile >> 1
	pixelInByte := pixelOffsetInTile & 0x01

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

	// Look up color in CGRAM, or synthesize direct color if enabled.
	color := p.getColorFromCGRAM(paletteIndex, colorIndex)
	if _, channel := p.resolveLayerTransformChannel(layerNum); channel != nil && channel.DirectColor {
		// Synthesize RGB from palette+pixel index to bypass CGRAM.
		combined := (uint16(paletteIndex&0x0F) << 4) | uint16(colorIndex&0x0F)
		r5 := uint32(combined&0x07) * 31 / 7
		g5 := uint32((combined>>3)&0x07) * 31 / 7
		b5 := uint32((combined>>6)&0x03) * 31 / 3
		color = ((r5 * 255 / 31) << 16) | ((g5 * 255 / 31) << 8) | (b5 * 255 / 31)
	}

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
	layer, channel := p.resolveLayerTransformChannel(layerNum)
	if layer == nil || channel == nil {
		return
	}

	if !layer.Enabled || !channel.Enabled {
		return
	}

	// Check windowing
	if !p.isPixelInWindow(x, y, layerNum) {
		return
	}

	// Apply mirroring if enabled
	screenX := int16(x)
	screenY := int16(y)
	if channel.MirrorH {
		screenX = int16(ScreenWidth - 1 - x)
	}
	if channel.MirrorV {
		screenY = int16(ScreenHeight - 1 - y)
	}

	// Transform screen coordinates to tilemap coordinates
	// Formula: [x'] = [A B] × [x - CX]
	//          [y']   [C D]   [y - CY]
	// Where A, B, C, D are 8.8 fixed point (1.0 = 0x0100)

	// Calculate relative to center
	relX := int32(screenX) - int32(channel.CenterX)
	relY := int32(screenY) - int32(channel.CenterY)

	// Apply transformation matrix using SNES-style origin semantics:
	//   world = origin + M * (screen - origin) + scroll
	// This preserves the source-space pivot instead of treating CenterX/Y as
	// only a screen-space subtraction term.
	worldX := int32(channel.CenterX) + ((int32(channel.A)*relX + int32(channel.B)*relY) >> 8)
	worldY := int32(channel.CenterY) + ((int32(channel.C)*relX + int32(channel.D)*relY) >> 8)

	// Add layer scroll offset after the pivot-preserving transform.
	worldX += int32(layer.ScrollX)
	worldY += int32(layer.ScrollY)

	plane := p.getMatrixPlane(layer.TransformChannel)

	if plane.Enabled && plane.SourceMode == MatrixPlaneSourceBitmap && plane.LiveFloorEnabled {
		if !p.prepareLiveFloorRow(layer.TransformChannel, y) {
			return
		}
		cache := &p.MatrixFloorRows[layer.TransformChannel]
		worldX = int32((int64(cache.startX) + int64(cache.stepX)*int64(x)) >> 16)
		worldY = int32((int64(cache.startY) + int64(cache.stepY)*int64(x)) >> 16)
	}

	// Handle outside-screen coordinates based on MatrixOutsideMode.
	tileSize := 8
	if layer.TileSize {
		tileSize = 16
	}
	sourcePixelWidth := tilemapWidthForSizeMode(layer.TilemapSize) * tileSize
	sourcePixelHeight := sourcePixelWidth
	if plane.Enabled {
		if plane.SourceMode == MatrixPlaneSourceBitmap {
			sourcePixelWidth = tilemapWidthForSizeMode(plane.Size) * 8
			sourcePixelHeight = sourcePixelWidth
		} else {
			sourcePixelWidth = tilemapWidthForSizeMode(plane.Size) * tileSize
			sourcePixelHeight = sourcePixelWidth
		}
	}

	// Check if coordinates are outside tilemap bounds
	worldXValid := worldX >= 0 && worldX < int32(sourcePixelWidth)
	worldYValid := worldY >= 0 && worldY < int32(sourcePixelHeight)

	if !worldXValid || !worldYValid {
		// Outside screen bounds - handle based on mode
		switch channel.OutsideMode {
		case 0: // Repeat/wrap mode (default)
			worldX = worldX % int32(sourcePixelWidth)
			if worldX < 0 {
				worldX += int32(sourcePixelWidth)
			}
			worldY = worldY % int32(sourcePixelHeight)
			if worldY < 0 {
				worldY += int32(sourcePixelHeight)
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
		case 3: // Clamp mode - clamp to nearest edge pixel
			if worldX < 0 {
				worldX = 0
			} else if worldX >= int32(sourcePixelWidth) {
				worldX = int32(sourcePixelWidth - 1)
			}
			if worldY < 0 {
				worldY = 0
			} else if worldY >= int32(sourcePixelHeight) {
				worldY = int32(sourcePixelHeight - 1)
			}
		default:
			// Default to wrap
			worldX = worldX % int32(sourcePixelWidth)
			if worldX < 0 {
				worldX += int32(sourcePixelWidth)
			}
			worldY = worldY % int32(sourcePixelHeight)
			if worldY < 0 {
				worldY += int32(sourcePixelHeight)
			}
		}
	}

	if plane.Enabled && plane.SourceMode == MatrixPlaneSourceBitmap {
		pixelOffset := int(worldY)*sourcePixelWidth + int(worldX)
		byteOffset := pixelOffset / 2
		pixelInByte := pixelOffset % 2
		if byteOffset < 0 || byteOffset >= len(plane.Bitmap) {
			return
		}
		pixelByte := plane.Bitmap[byteOffset]
		var colorIndex uint8
		if pixelInByte == 0 {
			colorIndex = (pixelByte >> 4) & 0x0F
		} else {
			colorIndex = pixelByte & 0x0F
		}
		if plane.Transparent0 && colorIndex == 0 {
			return
		}
		color := p.getColorFromCGRAM(plane.BitmapPalette&0x0F, colorIndex)
		if channel.DirectColor {
			combined := (uint16(plane.BitmapPalette&0x0F) << 4) | uint16(colorIndex&0x0F)
			r5 := uint32(combined&0x07) * 31 / 7
			g5 := uint32((combined>>3)&0x07) * 31 / 7
			b5 := uint32((combined>>6)&0x03) * 31 / 3
			color = ((r5 * 255 / 31) << 16) | ((g5 * 255 / 31) << 8) | (b5 * 255 / 31)
		}
		p.OutputBuffer[y*320+x] = color
		return
	}

	// Calculate which tile this pixel is in
	tileX := int(worldX) / tileSize
	tileY := int(worldY) / tileSize

	// Calculate pixel position within tile
	pixelXInTile := int(worldX) % tileSize
	pixelYInTile := int(worldY) % tileSize

	// Read tilemap entry
	tileIndex, attributes, ok := p.matrixTilemapEntry(layer, tileX, tileY)
	if !ok {
		return
	}
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

	pixelOffsetInTile := pixelYInTile*tileSize + pixelXInTile
	byteOffsetInTile := pixelOffsetInTile / 2
	pixelInByte := pixelOffsetInTile % 2
	tileByte := uint8(0)
	if plane.Enabled {
		tileDataOffset := uint32(tileIndex)*uint32(tileSize*tileSize/2) + uint32(byteOffsetInTile)
		if tileDataOffset >= uint32(len(plane.Pattern)) {
			return
		}
		tileByte = plane.Pattern[tileDataOffset]
	} else {
		tileDataOffset := uint16(tileIndex) * uint16(tileSize*tileSize/2)
		if uint32(tileDataOffset)+uint32(byteOffsetInTile) >= 65536 {
			return
		}
		tileDataAddr := tileDataOffset + uint16(byteOffsetInTile)
		tileByte = p.VRAM[tileDataAddr]
	}

	// Read pixel color index
	var colorIndex uint8
	if pixelInByte == 0 {
		colorIndex = (tileByte >> 4) & 0x0F
	} else {
		colorIndex = tileByte & 0x0F
	}

	// Look up color in CGRAM, or synthesize direct color if enabled.
	color := p.getColorFromCGRAM(paletteIndex, colorIndex)
	if channel.DirectColor {
		// Synthesize RGB from palette+pixel index to bypass CGRAM.
		combined := (uint16(paletteIndex&0x0F) << 4) | uint16(colorIndex&0x0F)
		r5 := uint32(combined&0x07) * 31 / 7
		g5 := uint32((combined>>3)&0x07) * 31 / 7
		b5 := uint32((combined>>6)&0x03) * 31 / 3
		color = ((r5 * 255 / 31) << 16) | ((g5 * 255 / 31) << 8) | (b5 * 255 / 31)
	}

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

// renderElement is a sortable render item (background layer or sprite).
// spriteIndex indexes into PPU.spriteScratch when elementType == 1.
type renderElement struct {
	priority    uint8
	elementType int // 0=background, 1=sprite
	layerNum    int // for backgrounds
	spriteIndex int // for sprites (index into PPU scratch)
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

// updateHDMA applies per-scanline raster commands.
// Base table format is 64 bytes per scanline: 16 bytes per layer
// [ScrollX_L, ScrollX_H, ScrollY_L, ScrollY_H,
//
//	MatrixA_L, MatrixA_H, MatrixB_L, MatrixB_H,
//	MatrixC_L, MatrixC_H, MatrixD_L, MatrixD_H,
//	CenterX_L, CenterX_H, CenterY_L, CenterY_H]
//
// Optional extension when HDMAControl bit 5 is set:
//
//	4 rebind bytes follow the 64-byte block, one per layer.
//	0xFF = keep current binding, 0x00-0x03 = bind layer to transform channel.
//
// Rebinding is applied before scroll/transform parameters so the decoded
// transform payload targets the newly bound channel for that scanline.
func (p *PPU) updateHDMA(scanline int) {
	if !p.HDMAEnabled || scanline >= VisibleScanlines {
		return
	}
	commands := p.decodeScanlineCommands(scanline)
	p.applyScanlineCommands(scanline, commands)
}
