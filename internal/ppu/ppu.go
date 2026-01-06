package ppu

import "fmt"

// PPU represents the Picture Processing Unit
// It implements the memory.IOHandler interface
type PPU struct {
	// VRAM (64KB)
	VRAM [65536]uint8

	// CGRAM (512 bytes, 256 colors × 2 bytes)
	CGRAM [512]uint8

	// OAM (768 bytes, 128 sprites × 6 bytes)
	OAM [768]uint8

	// Background layers
	BG0, BG1, BG2, BG3 BackgroundLayer

	// Matrix Mode
	MatrixEnabled      bool
	MatrixA, MatrixB   int16 // 8.8 fixed point
	MatrixC, MatrixD   int16 // 8.8 fixed point
	MatrixCenterX       int16
	MatrixCenterY       int16
	MatrixMirrorH       bool
	MatrixMirrorV       bool

	// Windowing
	Window0, Window1    Window
	WindowControl       uint8
	WindowMainEnable    uint8
	WindowSubEnable     uint8

	// HDMA
	HDMAEnabled         bool
	HDMATableBase       uint16

	// Debug
	debugFrameCount     int
	HDMAScrollX         [4][200]int16
	HDMAScrollY         [4][200]int16

	// VRAM/CGRAM/OAM access registers
	VRAMAddr            uint16
	CGRAMAddr           uint8
	CGRAMWriteLatch     bool // For 16-bit RGB555 writes
	CGRAMWriteValue     uint16
	OAMAddr             uint8

	// Output buffer (320×200, RGB888)
	OutputBuffer        [320 * 200]uint32
}

// BackgroundLayer represents a background layer
type BackgroundLayer struct {
	ScrollX     int16
	ScrollY     int16
	Enabled     bool
	TileSize    bool // false = 8×8, true = 16×16
	TilemapBase uint16
}

// Window represents a window
type Window struct {
	Left, Right, Top, Bottom uint8
}

// NewPPU creates a new PPU instance
func NewPPU() *PPU {
	return &PPU{
		BG0: BackgroundLayer{},
		BG1: BackgroundLayer{},
		BG2: BackgroundLayer{},
		BG3: BackgroundLayer{},
		Window0: Window{},
		Window1: Window{},
	}
}

// Read8 reads an 8-bit value from PPU registers
func (p *PPU) Read8(offset uint16) uint8 {
	switch offset {
	case 0x10: // VRAM_DATA
		value := p.VRAM[p.VRAMAddr]
		p.VRAMAddr++
		if p.VRAMAddr > 0xFFFF {
			p.VRAMAddr = 0
		}
		return value
	case 0x13: // CGRAM_DATA
		// CGRAM is write-only, return 0
		return 0
	case 0x15: // OAM_DATA
		if p.OAMAddr < 128 {
			return p.OAM[p.OAMAddr*6]
		}
		return 0
	default:
		return 0
	}
}

// Write8 writes an 8-bit value to PPU registers
func (p *PPU) Write8(offset uint16, value uint8) {
	switch offset {
	// BG0 scroll
	case 0x00: // BG0_SCROLLX_L
		p.BG0.ScrollX = int16((uint16(p.BG0.ScrollX) & 0xFF00) | uint16(value))
	case 0x01: // BG0_SCROLLX_H
		p.BG0.ScrollX = int16((uint16(p.BG0.ScrollX) & 0x00FF) | (uint16(value) << 8))
	case 0x02: // BG0_SCROLLY_L
		p.BG0.ScrollY = int16((uint16(p.BG0.ScrollY) & 0xFF00) | uint16(value))
	case 0x03: // BG0_SCROLLY_H
		p.BG0.ScrollY = int16((uint16(p.BG0.ScrollY) & 0x00FF) | (uint16(value) << 8))

	// BG1 scroll
	case 0x04: // BG1_SCROLLX_L
		p.BG1.ScrollX = int16((uint16(p.BG1.ScrollX) & 0xFF00) | uint16(value))
	case 0x05: // BG1_SCROLLX_H
		p.BG1.ScrollX = int16((uint16(p.BG1.ScrollX) & 0x00FF) | (uint16(value) << 8))
	case 0x06: // BG1_SCROLLY_L
		p.BG1.ScrollY = int16((uint16(p.BG1.ScrollY) & 0xFF00) | uint16(value))
	case 0x07: // BG1_SCROLLY_H
		p.BG1.ScrollY = int16((uint16(p.BG1.ScrollY) & 0x00FF) | (uint16(value) << 8))

	// BG0/BG1 control
	case 0x08: // BG0_CONTROL
		p.BG0.Enabled = (value & 0x01) != 0
		p.BG0.TileSize = (value & 0x02) != 0
	case 0x09: // BG1_CONTROL
		p.BG1.Enabled = (value & 0x01) != 0
		p.BG1.TileSize = (value & 0x02) != 0

	// BG2 scroll
	case 0x0A: // BG2_SCROLLX_L
		p.BG2.ScrollX = int16((uint16(p.BG2.ScrollX) & 0xFF00) | uint16(value))
	case 0x0B: // BG2_SCROLLX_H
		p.BG2.ScrollX = int16((uint16(p.BG2.ScrollX) & 0x00FF) | (uint16(value) << 8))
	case 0x0C: // BG2_SCROLLY_L
		p.BG2.ScrollY = int16((uint16(p.BG2.ScrollY) & 0xFF00) | uint16(value))
	case 0x0D: // BG2_SCROLLY_H
		p.BG2.ScrollY = int16((uint16(p.BG2.ScrollY) & 0x00FF) | (uint16(value) << 8))

	// VRAM access
	case 0x0E: // VRAM_ADDR_L
		p.VRAMAddr = (p.VRAMAddr & 0xFF00) | uint16(value)
	case 0x0F: // VRAM_ADDR_H
		p.VRAMAddr = (p.VRAMAddr & 0x00FF) | (uint16(value) << 8)
	case 0x10: // VRAM_DATA
		p.VRAM[p.VRAMAddr] = value
		p.VRAMAddr++
		if p.VRAMAddr > 0xFFFF {
			p.VRAMAddr = 0
		}

	// CGRAM access
	case 0x12: // CGRAM_ADDR
		p.CGRAMAddr = value
		p.CGRAMWriteLatch = false
	case 0x13: // CGRAM_DATA
		if !p.CGRAMWriteLatch {
			// First write: low byte
			p.CGRAMWriteValue = uint16(value)
			p.CGRAMWriteLatch = true
		} else {
			// Second write: high byte (RGB555 format)
			p.CGRAMWriteValue |= (uint16(value) << 8)
			// Write to CGRAM
			addr := uint16(p.CGRAMAddr) * 2
			if addr < 512 {
				// Store in little-endian order: low byte first, high byte second
				p.CGRAM[addr] = uint8(p.CGRAMWriteValue & 0xFF)      // Low byte
				p.CGRAM[addr+1] = uint8(p.CGRAMWriteValue >> 8)       // High byte
				p.CGRAMAddr++
				if p.CGRAMAddr > 255 {
					p.CGRAMAddr = 0
				}
			}
			p.CGRAMWriteLatch = false
		}

	// OAM access
	case 0x14: // OAM_ADDR
		p.OAMAddr = value
		if p.OAMAddr > 127 {
			p.OAMAddr = 127
		}
	case 0x15: // OAM_DATA
		addr := uint16(p.OAMAddr) * 6
		if addr < 768 {
			p.OAM[addr] = value
			p.OAMAddr++
			if p.OAMAddr > 127 {
				p.OAMAddr = 0
			}
		}

	// Matrix Mode
	case 0x18: // MATRIX_CONTROL
		p.MatrixEnabled = (value & 0x01) != 0
		p.MatrixMirrorH = (value & 0x02) != 0
		p.MatrixMirrorV = (value & 0x04) != 0
	case 0x19: // MATRIX_A_L
		p.MatrixA = int16((uint16(p.MatrixA) & 0xFF00) | uint16(value))
	case 0x1A: // MATRIX_A_H
		p.MatrixA = int16((uint16(p.MatrixA) & 0x00FF) | (uint16(value) << 8))
	case 0x1B: // MATRIX_B_L
		p.MatrixB = int16((uint16(p.MatrixB) & 0xFF00) | uint16(value))
	case 0x1C: // MATRIX_B_H
		p.MatrixB = int16((uint16(p.MatrixB) & 0x00FF) | (uint16(value) << 8))
	case 0x1D: // MATRIX_C_L
		p.MatrixC = int16((uint16(p.MatrixC) & 0xFF00) | uint16(value))
	case 0x1E: // MATRIX_C_H
		p.MatrixC = int16((uint16(p.MatrixC) & 0x00FF) | (uint16(value) << 8))
	case 0x1F: // MATRIX_D_L
		p.MatrixD = int16((uint16(p.MatrixD) & 0xFF00) | uint16(value))
	case 0x20: // MATRIX_D_H
		p.MatrixD = int16((uint16(p.MatrixD) & 0x00FF) | (uint16(value) << 8))

	// BG2/BG3 control
	case 0x21: // BG2_CONTROL
		p.BG2.Enabled = (value & 0x01) != 0
		p.BG2.TileSize = (value & 0x02) != 0
	case 0x22: // BG3_SCROLLX_L
		p.BG3.ScrollX = int16((uint16(p.BG3.ScrollX) & 0xFF00) | uint16(value))
	case 0x23: // BG3_SCROLLX_H
		p.BG3.ScrollX = int16((uint16(p.BG3.ScrollX) & 0x00FF) | (uint16(value) << 8))
	case 0x24: // BG3_SCROLLY_L
		p.BG3.ScrollY = int16((uint16(p.BG3.ScrollY) & 0xFF00) | uint16(value))
	case 0x25: // BG3_SCROLLY_H
		p.BG3.ScrollY = int16((uint16(p.BG3.ScrollY) & 0x00FF) | (uint16(value) << 8))
	case 0x26: // BG3_CONTROL
		p.BG3.Enabled = (value & 0x01) != 0
		p.BG3.TileSize = (value & 0x02) != 0

	// Matrix center
	case 0x27: // MATRIX_CENTER_X_L
		p.MatrixCenterX = int16((uint16(p.MatrixCenterX) & 0xFF00) | uint16(value))
	case 0x28: // MATRIX_CENTER_X_H
		p.MatrixCenterX = int16((uint16(p.MatrixCenterX) & 0x00FF) | (uint16(value) << 8))
	case 0x29: // MATRIX_CENTER_Y_L
		p.MatrixCenterY = int16((uint16(p.MatrixCenterY) & 0xFF00) | uint16(value))
	case 0x2A: // MATRIX_CENTER_Y_H
		p.MatrixCenterY = int16((uint16(p.MatrixCenterY) & 0x00FF) | (uint16(value) << 8))

	// Windowing
	case 0x2B: // WINDOW0_LEFT
		p.Window0.Left = value
	case 0x2C: // WINDOW0_RIGHT
		p.Window0.Right = value
	case 0x2D: // WINDOW0_TOP
		p.Window0.Top = value
	case 0x2E: // WINDOW0_BOTTOM
		p.Window0.Bottom = value
	case 0x2F: // WINDOW1_LEFT
		p.Window1.Left = value
	case 0x30: // WINDOW1_RIGHT
		p.Window1.Right = value
	case 0x31: // WINDOW1_TOP
		p.Window1.Top = value
	case 0x32: // WINDOW1_BOTTOM
		p.Window1.Bottom = value
	case 0x33: // WINDOW_CONTROL
		p.WindowControl = value
	case 0x34: // WINDOW_MAIN_ENABLE
		p.WindowMainEnable = value
	case 0x35: // WINDOW_SUB_ENABLE
		p.WindowSubEnable = value

	// HDMA
	case 0x36: // HDMA_CONTROL
		p.HDMAEnabled = (value & 0x01) != 0
	case 0x37: // HDMA_TABLE_BASE_L
		p.HDMATableBase = (p.HDMATableBase & 0xFF00) | uint16(value)
	case 0x38: // HDMA_TABLE_BASE_H
		p.HDMATableBase = (p.HDMATableBase & 0x00FF) | (uint16(value) << 8)
	}
}

// Read16 reads a 16-bit value from PPU registers
func (p *PPU) Read16(offset uint16) uint16 {
	low := p.Read8(offset)
	high := p.Read8(offset + 1)
	return uint16(low) | (uint16(high) << 8)
}

// Write16 writes a 16-bit value to PPU registers
func (p *PPU) Write16(offset uint16, value uint16) {
	p.Write8(offset, uint8(value&0xFF))
	p.Write8(offset+1, uint8(value>>8))
}

// RenderFrame renders a complete frame
func (p *PPU) RenderFrame() {
	// Clear output buffer
	for i := range p.OutputBuffer {
		p.OutputBuffer[i] = 0x000000 // Black
	}

	// Debug: Print CGRAM contents once per 60 frames
	p.debugFrameCount++
	if p.debugFrameCount == 60 {
		fmt.Printf("=== CGRAM Debug (palette 0, colors 0-3) ===\n")
		for i := 0; i < 4; i++ {
			addr := i * 2
			low := p.CGRAM[addr]
			high := p.CGRAM[addr+1]
			color := p.getColorFromCGRAM(0, uint8(i))
			r := (color >> 16) & 0xFF
			g := (color >> 8) & 0xFF
			b := color & 0xFF
			fmt.Printf("  Color %d: CGRAM[%d]=0x%02X, CGRAM[%d]=0x%02X -> RGB(%d,%d,%d) = 0x%06X\n", i, addr, low, addr+1, high, r, g, b, color)
		}
		fmt.Printf("=== First 10 pixels of output buffer ===\n")
		for i := 0; i < 10; i++ {
			color := p.OutputBuffer[i]
			r := (color >> 16) & 0xFF
			g := (color >> 8) & 0xFF
			b := color & 0xFF
			fmt.Printf("  Pixel %d: 0x%06X (RGB %d,%d,%d)\n", i, color, r, g, b)
		}
		fmt.Printf("=== BG0 state: Enabled=%v, ScrollX=%d, ScrollY=%d ===\n", p.BG0.Enabled, p.BG0.ScrollX, p.BG0.ScrollY)
		fmt.Printf("=== VRAM[0x4000-0x4003] (first tilemap entry): 0x%02X 0x%02X 0x%02X 0x%02X ===\n", p.VRAM[0x4000], p.VRAM[0x4001], p.VRAM[0x4002], p.VRAM[0x4003])
		fmt.Printf("=== VRAM[0x0000-0x0003] (first tile data): 0x%02X 0x%02X 0x%02X 0x%02X ===\n", p.VRAM[0x0000], p.VRAM[0x0001], p.VRAM[0x0002], p.VRAM[0x0003])
		p.debugFrameCount = 0
	}

	// Render background layers (BG3 → BG0, back to front)
	if p.BG3.Enabled {
		p.renderBackgroundLayer(3)
	}
	if p.BG2.Enabled {
		p.renderBackgroundLayer(2)
	}
	if p.BG1.Enabled {
		p.renderBackgroundLayer(1)
	}
	if p.BG0.Enabled {
		if p.MatrixEnabled {
			p.renderMatrixMode()
		} else {
			p.renderBackgroundLayer(0)
		}
	}

	// Render sprites
	p.renderSprites()
}

// renderBackgroundLayer renders a background layer
func (p *PPU) renderBackgroundLayer(layerNum int) {
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

	// Tile size: 8x8 or 16x16
	tileSize := 8
	if layer.TileSize {
		tileSize = 16
	}

	// Tilemap is 32x32 tiles
	tilemapWidth := 32
	tilemapHeight := 32

	// Render each pixel
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			// Check windowing
			if !p.isPixelInWindow(x, y, layerNum) {
				continue
			}

			// Calculate tilemap coordinates with scroll
			// Screen pixel (x, y) -> world pixel (worldX, worldY)
			worldX := int(x) + int(layer.ScrollX)
			worldY := int(y) + int(layer.ScrollY)

			// Wrap coordinates (tilemap repeats)
			tilemapPixelWidth := tilemapWidth * tileSize
			tilemapPixelHeight := tilemapHeight * tileSize
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

			// Read tilemap entry (2 bytes per tile)
			// Tilemap entry at (tileX, tileY) = tilemapBase + (tileY * 32 + tileX) * 2
			// Default tilemap base: 0x4000 for BG0 (can be configured later)
			tilemapBase := uint16(0x4000) // Default tilemap base
			if layer.TilemapBase != 0 {
				tilemapBase = layer.TilemapBase
			}
			tilemapOffset := uint16((tileY*tilemapWidth+tileX) * 2)
			if uint32(tilemapBase)+uint32(tilemapOffset) >= 65536 {
				// Out of bounds, render black
				p.OutputBuffer[y*320+x] = 0x000000
				continue
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

			// Read tile data (4bpp = 2 pixels per byte)
			// Tile data starts at VRAM offset = tileIndex * (tileSize * tileSize / 2)
			tileDataOffset := uint16(tileIndex) * uint16(tileSize*tileSize/2)
			// Pixel position in tile: pixelYInTile * tileSize + pixelXInTile
			pixelOffsetInTile := pixelYInTile*tileSize + pixelXInTile
			// Byte offset in tile data
			byteOffsetInTile := pixelOffsetInTile / 2
			// Which pixel in the byte (0 = upper 4 bits, 1 = lower 4 bits)
			pixelInByte := pixelOffsetInTile % 2

			if uint32(tileDataOffset)+uint32(byteOffsetInTile) >= 65536 {
				// Out of bounds, render black
				p.OutputBuffer[y*320+x] = 0x000000
				continue
			}
			tileDataAddr := tileDataOffset + uint16(byteOffsetInTile)

			// Read pixel color index (4 bits = 0-15)
			tileByte := p.VRAM[tileDataAddr]
			var colorIndex uint8
			if pixelInByte == 0 {
				colorIndex = (tileByte >> 4) & 0x0F // Upper 4 bits
			} else {
				colorIndex = tileByte & 0x0F // Lower 4 bits
			}

			// Look up color in CGRAM
			// Note: Color index 0 is NOT transparent for backgrounds (only for sprites)
			// Backgrounds always render, even if color index is 0
			color := p.getColorFromCGRAM(paletteIndex, colorIndex)
			p.OutputBuffer[y*320+x] = color
		}
	}
}

// renderMatrixMode renders Matrix Mode (Mode 7-style)
func (p *PPU) renderMatrixMode() {
	// TODO: Implement Matrix Mode transformation
	// For now, just render BG0 normally
	p.renderBackgroundLayer(0)
}

// renderSprites renders all sprites
func (p *PPU) renderSprites() {
	// Render sprites (128 max)
	for spriteIndex := 0; spriteIndex < 128; spriteIndex++ {
		// OAM entry is 6 bytes per sprite
		oamAddr := spriteIndex * 6
		
		// Read sprite data
		// Byte 0: X position (low byte, signed)
		xLow := int8(p.OAM[oamAddr])
		// Byte 1: X position (high byte, bit 0 only, sign extends)
		xHigh := int8(p.OAM[oamAddr+1])
		// Combine X position (sign extend)
		spriteX := int(xLow) | (int(xHigh) << 8)
		if (xHigh & 0x01) != 0 {
			// Sign extend
			spriteX |= 0xFFFFFF00
		}
		
		// Byte 2: Y position (8-bit, 0-255)
		spriteY := int(p.OAM[oamAddr+2])
		
		// Byte 3: Tile index
		tileIndex := uint8(p.OAM[oamAddr+3])
		
		// Byte 4: Attributes
		attributes := uint8(p.OAM[oamAddr+4])
		paletteIndex := attributes & 0x0F
		flipX := (attributes & 0x10) != 0
		flipY := (attributes & 0x20) != 0
		_ = (attributes >> 6) & 0x3 // priority (not used yet)
		
		// Byte 5: Control
		control := uint8(p.OAM[oamAddr+5])
		enabled := (control & 0x01) != 0
		tileSize16 := (control & 0x02) != 0
		
		if !enabled {
			continue
		}
		
		// Sprite size
		spriteSize := 8
		if tileSize16 {
			spriteSize = 16
		}
		
		// Render sprite pixels
		for py := 0; py < spriteSize; py++ {
			for px := 0; px < spriteSize; px++ {
				// Calculate screen position
				screenX := spriteX + px
				screenY := spriteY + py
				
				// Check bounds
				if screenX < 0 || screenX >= 320 || screenY < 0 || screenY >= 200 {
					continue
				}
				
				// Apply flip
				tileX := px
				tileY := py
				if flipX {
					tileX = spriteSize - 1 - tileX
				}
				if flipY {
					tileY = spriteSize - 1 - tileY
				}
				
				// Read tile data (4bpp = 2 pixels per byte)
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
				
				// Look up color and render
				color := p.getColorFromCGRAM(paletteIndex, colorIndex)
				p.OutputBuffer[screenY*320+screenX] = color
			}
		}
	}
}

// isPixelInWindow checks if a pixel is inside the window
func (p *PPU) isPixelInWindow(x, y, layerNum int) bool {
	// Check if windowing is enabled for this layer
	if (p.WindowMainEnable & (1 << layerNum)) == 0 {
		return true // No windowing
	}

	// Check window logic
	// Window bounds: Left/Right/Top/Bottom are 8-bit values (0-255)
	// If windowing is enabled, check if pixel is inside window bounds
	// If Right is 0 and Left is 0, assume window is not active
	win0Inside := true // Default to inside if window not configured
	if (p.Window0.Right > 0 || p.Window0.Left > 0) {
		// Window is configured, check bounds
		win0Inside = x >= int(p.Window0.Left) && x <= int(p.Window0.Right) &&
			y >= int(p.Window0.Top) && y <= int(p.Window0.Bottom)
	}
	
	win1Inside := true // Default to inside if window not configured
	if (p.Window1.Right > 0 || p.Window1.Left > 0) {
		// Window is configured, check bounds
		win1Inside = x >= int(p.Window1.Left) && x <= int(p.Window1.Right) &&
			y >= int(p.Window1.Top) && y <= int(p.Window1.Bottom)
	}

	logic := (p.WindowControl >> 2) & 0x3
	switch logic {
	case 0: // OR
		return win0Inside || win1Inside
	case 1: // AND
		return win0Inside && win1Inside
	case 2: // XOR
		return win0Inside != win1Inside
	case 3: // XNOR
		return win0Inside == win1Inside
	}

	return true
}

// getColorFromCGRAM gets a color from CGRAM
func (p *PPU) getColorFromCGRAM(paletteIndex, colorIndex uint8) uint32 {
	addr := (uint16(paletteIndex)*16 + uint16(colorIndex)) * 2
	if addr >= 512 {
		return 0x000000
	}

	// Read RGB555 color
	// CGRAM stores colors in little-endian order: low byte first, high byte second
	low := p.CGRAM[addr]    // Low byte is stored first
	high := p.CGRAM[addr+1] // High byte is stored second

	// Convert RGB555 to RGB888
	// RGB555 format: Low byte = GGGGG BBBBB, High byte = 0 RRRRR GG
	// Extract components
	// R: bits 10-14 from high byte (bits 2-6)
	r := uint32((high & 0x7C) >> 2)
	// G: bits 5-9, split between high (bits 0-1) and low (bits 5-7)
	g := uint32(((high & 0x03) << 3) | ((low & 0xE0) >> 5))
	// B: bits 0-4 from low byte
	b := uint32(low & 0x1F)

	// Scale to 8 bits
	r = (r * 255) / 31
	g = (g * 255) / 31
	b = (b * 255) / 31

	return (r << 16) | (g << 8) | b
}

