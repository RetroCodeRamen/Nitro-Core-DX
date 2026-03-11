package emulator

import (
	"fmt"

	"nitro-core-dx/internal/ppu"
)

type matrixPlaneMMIOWriter interface {
	Write8(offset uint16, value uint8)
}

// MatrixPlaneProgram describes a dedicated matrix-plane upload through the PPU MMIO surface.
// The plane provides its own tilemap backing and dedicated pattern memory.
type MatrixPlaneProgram struct {
	Channel       uint8
	Enabled       bool
	Size          uint8
	SourceMode    uint8
	BitmapPalette uint8
	Transparent0  bool
	Tilemap       []byte
	Pattern       []byte
	Bitmap        []byte
}

// MatrixPlaneBuilder is the recommended software-side authoring path for
// dedicated matrix planes before CoreLX grows first-class support.
type MatrixPlaneBuilder struct {
	channel       uint8
	enabled       bool
	size          uint8
	width         int
	sourceMode    uint8
	bitmapPalette uint8
	transparent0  bool
	tilemap       []byte
	pattern       []byte
	bitmap        []byte
}

func matrixPlaneTilemapBytes(sizeMode uint8) (int, error) {
	width := 0
	switch sizeMode {
	case ppu.TilemapSize32x32:
		width = 32
	case ppu.TilemapSize64x64:
		width = 64
	case ppu.TilemapSize128x128:
		width = 128
	default:
		return 0, fmt.Errorf("matrix plane size mode %d unsupported", sizeMode)
	}
	return width * width * 2, nil
}

func matrixPlaneWidth(sizeMode uint8) (int, error) {
	switch sizeMode {
	case ppu.TilemapSize32x32:
		return 32, nil
	case ppu.TilemapSize64x64:
		return 64, nil
	case ppu.TilemapSize128x128:
		return 128, nil
	default:
		return 0, fmt.Errorf("matrix plane size mode %d unsupported", sizeMode)
	}
}

func NewMatrixPlaneBuilder(channel, sizeMode uint8) (*MatrixPlaneBuilder, error) {
	if channel > 3 {
		return nil, fmt.Errorf("matrix plane channel %d out of range", channel)
	}
	width, err := matrixPlaneWidth(sizeMode)
	if err != nil {
		return nil, err
	}
	return &MatrixPlaneBuilder{
		channel:    channel,
		enabled:    true,
		size:       sizeMode,
		width:      width,
		sourceMode: ppu.MatrixPlaneSourceTilemap,
		tilemap:    make([]byte, width*width*2),
		pattern:    make([]byte, 32*1024),
		bitmap:     make([]byte, (width*8*width*8)/2),
	}, nil
}

func (b *MatrixPlaneBuilder) SetEnabled(enabled bool) {
	b.enabled = enabled
}

func (b *MatrixPlaneBuilder) SetBitmapMode(palette uint8) {
	b.sourceMode = ppu.MatrixPlaneSourceBitmap
	b.bitmapPalette = palette & 0x0F
}

func (b *MatrixPlaneBuilder) SetBitmapTransparency(enabled bool) {
	b.transparent0 = enabled
}

func (b *MatrixPlaneBuilder) SetTilemapMode() {
	b.sourceMode = ppu.MatrixPlaneSourceTilemap
}

func (b *MatrixPlaneBuilder) SetTile(x, y int, tileIndex, attributes uint8) error {
	if x < 0 || y < 0 || x >= b.width || y >= b.width {
		return fmt.Errorf("matrix plane tile coordinate (%d,%d) out of range for %dx%d", x, y, b.width, b.width)
	}
	offset := (y*b.width + x) * 2
	b.tilemap[offset] = tileIndex
	b.tilemap[offset+1] = attributes
	return nil
}

func (b *MatrixPlaneBuilder) FillRect(x, y, w, h int, tileIndex, attributes uint8) error {
	if w < 0 || h < 0 {
		return fmt.Errorf("matrix plane fill rect width/height must be non-negative")
	}
	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			if err := b.SetTile(x+col, y+row, tileIndex, attributes); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *MatrixPlaneBuilder) SetPatternTile8x8(tileIndex int, packed4bpp []byte) error {
	if tileIndex < 0 {
		return fmt.Errorf("tile index %d out of range", tileIndex)
	}
	if len(packed4bpp) != 32 {
		return fmt.Errorf("8x8 tile pattern length %d, want 32", len(packed4bpp))
	}
	offset := tileIndex * 32
	if offset+32 > len(b.pattern) {
		return fmt.Errorf("tile index %d exceeds matrix plane pattern memory", tileIndex)
	}
	copy(b.pattern[offset:offset+32], packed4bpp)
	return nil
}

func (b *MatrixPlaneBuilder) SetPatternTile16x16(tileIndex int, packed4bpp []byte) error {
	if tileIndex < 0 {
		return fmt.Errorf("tile index %d out of range", tileIndex)
	}
	if len(packed4bpp) != 128 {
		return fmt.Errorf("16x16 tile pattern length %d, want 128", len(packed4bpp))
	}
	offset := tileIndex * 128
	if offset+128 > len(b.pattern) {
		return fmt.Errorf("tile index %d exceeds matrix plane pattern memory", tileIndex)
	}
	copy(b.pattern[offset:offset+128], packed4bpp)
	return nil
}

func (b *MatrixPlaneBuilder) SetBitmapPixel(x, y int, colorIndex uint8) error {
	widthPixels := b.width * 8
	if x < 0 || y < 0 || x >= widthPixels || y >= widthPixels {
		return fmt.Errorf("matrix plane bitmap coordinate (%d,%d) out of range for %dx%d", x, y, widthPixels, widthPixels)
	}
	offset := y*widthPixels + x
	byteOffset := offset / 2
	if byteOffset < 0 || byteOffset >= len(b.bitmap) {
		return fmt.Errorf("matrix plane bitmap offset %d out of range", byteOffset)
	}
	if offset%2 == 0 {
		b.bitmap[byteOffset] = (b.bitmap[byteOffset] & 0x0F) | ((colorIndex & 0x0F) << 4)
	} else {
		b.bitmap[byteOffset] = (b.bitmap[byteOffset] & 0xF0) | (colorIndex & 0x0F)
	}
	return nil
}

func (b *MatrixPlaneBuilder) SetBitmapPacked4bpp(packed []byte, palette uint8) error {
	if len(packed) != len(b.bitmap) {
		return fmt.Errorf("bitmap length %d, want %d", len(packed), len(b.bitmap))
	}
	copy(b.bitmap, packed)
	b.SetBitmapMode(palette)
	return nil
}

func (b *MatrixPlaneBuilder) Build() MatrixPlaneProgram {
	// Preserve explicit zero-initialized pattern bytes; callers can choose how
	// much of the 32KB region they want emitted. We trim trailing zeros to keep
	// uploads smaller while leaving live contents deterministic.
	patternLen := len(b.pattern)
	for patternLen > 0 && b.pattern[patternLen-1] == 0 {
		patternLen--
	}
	pattern := make([]byte, patternLen)
	copy(pattern, b.pattern[:patternLen])

	bitmapLen := len(b.bitmap)
	for bitmapLen > 0 && b.bitmap[bitmapLen-1] == 0 {
		bitmapLen--
	}
	bitmap := make([]byte, bitmapLen)
	copy(bitmap, b.bitmap[:bitmapLen])

	tilemap := make([]byte, len(b.tilemap))
	copy(tilemap, b.tilemap)

	return MatrixPlaneProgram{
		Channel:       b.channel,
		Enabled:       b.enabled,
		Size:          b.size,
		SourceMode:    b.sourceMode,
		BitmapPalette: b.bitmapPalette,
		Transparent0:  b.transparent0,
		Tilemap:       tilemap,
		Pattern:       pattern,
		Bitmap:        bitmap,
	}
}

func ProgramMatrixPlaneThroughMMIO(w matrixPlaneMMIOWriter, program MatrixPlaneProgram) error {
	if w == nil {
		return fmt.Errorf("matrix plane MMIO target unavailable")
	}
	if program.Channel > 3 {
		return fmt.Errorf("matrix plane channel %d out of range", program.Channel)
	}
	tilemapBytes, err := matrixPlaneTilemapBytes(program.Size)
	if err != nil {
		return err
	}
	if len(program.Tilemap) != tilemapBytes {
		return fmt.Errorf("matrix plane tilemap length %d, want %d", len(program.Tilemap), tilemapBytes)
	}
	if len(program.Pattern) > 32*1024 {
		return fmt.Errorf("matrix plane pattern length %d exceeds %d", len(program.Pattern), 32*1024)
	}
	if program.SourceMode > ppu.MatrixPlaneSourceBitmap {
		return fmt.Errorf("matrix plane source mode %d unsupported", program.SourceMode)
	}
	if len(program.Bitmap) > (1024*1024)/2 {
		return fmt.Errorf("matrix plane bitmap length %d exceeds %d", len(program.Bitmap), (1024*1024)/2)
	}

	w.Write8(0x80, program.Channel&0x03)
	control := (program.Size & 0x03) << 1
	if program.Enabled {
		control |= 0x01
	}
	control |= (program.SourceMode & 0x01) << 3
	control |= (program.BitmapPalette & 0x0F) << 4
	w.Write8(0x81, control)
	var flags uint8
	if program.Transparent0 {
		flags |= 0x01
	}
	w.Write8(0x8C, flags)
	w.Write8(0x82, 0x00)
	w.Write8(0x83, 0x00)
	for _, b := range program.Tilemap {
		w.Write8(0x84, b)
	}
	w.Write8(0x85, 0x00)
	w.Write8(0x86, 0x00)
	for _, b := range program.Pattern {
		w.Write8(0x87, b)
	}
	w.Write8(0x88, 0x00)
	w.Write8(0x89, 0x00)
	w.Write8(0x8A, 0x00)
	for _, b := range program.Bitmap {
		w.Write8(0x8B, b)
	}
	return nil
}

// InstallMatrixPlaneProgram uploads one dedicated matrix plane through the public PPU MMIO path.
func (e *Emulator) InstallMatrixPlaneProgram(program MatrixPlaneProgram) error {
	if e == nil || e.PPU == nil {
		return fmt.Errorf("emulator PPU unavailable")
	}
	return ProgramMatrixPlaneThroughMMIO(e.PPU, program)
}
