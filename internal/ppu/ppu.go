package ppu

import (
	"nitro-core-dx/internal/debug"
)

type textCmd struct {
	x, y  int
	color uint32
	char  rune
}

// PPU represents the Picture Processing Unit
// It implements the memory.IOHandler interface
type PPU struct {
	// VRAM (64KB)
	VRAM [65536]uint8

	// CGRAM (512 bytes, 256 colors × 2 bytes)
	CGRAM [512]uint8
	// Cached RGB888 conversions of CGRAM entries (derived from CGRAM, software optimization only).
	cgramRGBCache   [256]uint32
	cgramCacheValid [256]bool

	// OAM (768 bytes, 128 sprites × 6 bytes)
	OAM [768]uint8

	// Background layers (each has its own matrix transformation)
	BG0, BG1, BG2, BG3 BackgroundLayer
	// Transform channels are the runtime matrix engines bound to visible layers.
	// Stage 2 keeps layer-owned matrix fields for compatibility, but rendering and
	// HDMA now resolve matrix state through these channels.
	TransformChannels [4]TransformChannel
	// Dedicated matrix-plane source backing. This is emulator-first but hardware-shaped:
	// each transform channel can source tilemap data from its own plane memory instead
	// of implicitly borrowing ordinary BG tilemap storage.
	MatrixPlanes [4]MatrixPlane
	// Per-plane row caches for live bitmap-floor sampling. These are derived state,
	// not programmer-visible memory, and avoid recomputing row origins/steps per pixel.
	MatrixFloorRows [4]matrixFloorRowCache

	// Windowing
	Window0, Window1 Window
	WindowControl    uint8
	WindowMainEnable uint8
	WindowSubEnable  uint8

	// HDMA
	HDMAEnabled    bool
	HDMATableBase  uint16
	HDMAControl    uint8 // Bit 0=enable, bits 1-4=layer enable (BG0-BG3), bit 5=rebind table, bit 6=priority table, bit 7=tilemap-base table
	HDMAExtControl uint8 // Bit 0=source-mode table present

	// Debug
	debugFrameCount int
	HDMAScrollX     [4][200]int16
	HDMAScrollY     [4][200]int16
	HDMAMatrixA     [4][200]int16 // Per-scanline matrix A updates
	HDMAMatrixB     [4][200]int16 // Per-scanline matrix B updates
	HDMAMatrixC     [4][200]int16 // Per-scanline matrix C updates
	HDMAMatrixD     [4][200]int16 // Per-scanline matrix D updates
	HDMAMatrixCX    [4][200]int16 // Per-scanline center X updates
	HDMAMatrixCY    [4][200]int16 // Per-scanline center Y updates

	// Frame counter (for ROM timing) - increments once per frame
	FrameCounter uint16

	// VBlank flag (hardware-accurate synchronization signal)
	// Set at start of VBlank period (scanline 200), cleared when read (one-shot)
	// This matches real hardware behavior (NES, SNES, etc.)
	// FPGA-implementable: Simple D flip-flop with read-clear logic
	VBlankFlag bool

	// Logger for centralized logging
	Logger *debug.Logger

	// Interrupt callback (called when VBlank occurs)
	// This allows PPU to trigger CPU interrupts
	InterruptCallback func(interruptType uint8)

	// Memory reader for DMA (reads from ROM/RAM)
	// Set by emulator to allow DMA transfers
	MemoryReader func(bank uint8, offset uint16) uint8

	// VRAM/CGRAM/OAM access registers
	VRAMAddr               uint16
	CGRAMAddr              uint8
	CGRAMWriteLatch        bool // For 16-bit RGB555 writes
	MatrixPlaneSelect      uint8
	MatrixPlaneAddr        uint16
	MatrixPlanePatternAddr uint16
	MatrixPlaneBitmapAddr  uint32

	// DMA (Direct Memory Access)
	DMAEnabled      bool
	DMASourceBank   uint8
	DMASourceOffset uint16
	DMADestType     uint8 // 0=VRAM, 1=CGRAM, 2=OAM, 3=matrix tilemap, 4=matrix pattern, 5=matrix bitmap
	DMADestAddr     uint16
	DMALength       uint16
	DMAMode         uint8  // 0=copy, 1=fill
	DMACycles       uint16 // Cycles remaining for DMA transfer (deprecated, use DMAProgress)
	// Cycle-accurate DMA state
	DMAProgress     uint16 // Current byte position in transfer (0 = start, DMALength = complete)
	DMACurrentSrc   uint16 // Current source offset
	DMACurrentDest  uint16 // Current destination address
	DMAFillValue    uint8  // Fill value for fill mode (read once at start)
	CGRAMWriteValue uint16
	OAMAddr         uint8
	OAMByteIndex    uint8 // Current byte index within sprite (0-5)

	// Text rendering state (MMIO 0x8070-0x8076)
	TextX     uint16 // cursor X (auto-advances by 8 after each char)
	TextY     uint8  // cursor Y
	TextR     uint8  // text color red
	TextG     uint8  // text color green
	TextB     uint8  // text color blue
	textCmds  [512]textCmd
	textCount int

	// Output buffer (320×200, RGB888) - back buffer, rendered to by PPU
	OutputBuffer [320 * 200]uint32
	// Display buffer - front buffer, presented after text overlay in endFrame
	DisplayBuffer [320 * 200]uint32

	// Scratch buffers used by dot renderer to avoid per-pixel allocations.
	// PPU rendering is single-threaded, so reusing these buffers is safe.
	spriteScratch        [128]spriteInfo
	renderElementScratch [132]renderElement
	// Hardware-like sprite evaluation cache: sprites active on the current scanline.
	// Evaluated once at scanline start, consumed by per-pixel rendering.
	activeScanlineSprites     [128]spriteInfo
	activeScanlineSpriteCount int
	activeScanlineY           int

	// Scanline/dot stepping state (for clock-driven operation)
	currentScanline     int
	currentDot          int
	scanlineInitialized bool
	frameStarted        bool
	FrameComplete       bool // Set to true when frame rendering is complete (safe to read buffer)
}

// GetScanline returns the current scanline (for debugging)
func (p *PPU) GetScanline() int {
	return p.currentScanline
}

// GetDot returns the current dot (for debugging)
func (p *PPU) GetDot() int {
	return p.currentDot
}

// GetOAMByteIndex returns the current OAM byte index (for debugging)
func (p *PPU) GetOAMByteIndex() uint8 {
	return p.OAMByteIndex
}

// GetTextCount returns the current number of buffered text commands (for debugging)
func (p *PPU) GetTextCount() int {
	return p.textCount
}

// BackgroundLayer represents a background layer
type BackgroundLayer struct {
	ScrollX     int16
	ScrollY     int16
	Enabled     bool
	Priority    uint8 // 0-3, higher renders on top
	TileSize    bool  // false = 8×8, true = 16×16
	TilemapSize uint8 // 0 = 32x32 tiles, 1 = 64x64 tiles, 2 = 128x128 tiles
	SourceMode  uint8 // 0=tilemap, 1=bitmap (reserved; tilemap is the only active runtime source today)
	TilemapBase uint16
	// TransformChannel selects the runtime transform channel bound to this layer.
	// Defaults to the layer index.
	TransformChannel uint8
	// Mosaic effect
	MosaicEnabled bool
	MosaicSize    uint8 // 1-15 (1 = no effect, 15 = max block size)
}

// TransformChannel represents a runtime affine transform engine that can be
// bound to a visible layer.
type TransformChannel struct {
	Enabled     bool
	A, B        int16 // 8.8 fixed point
	C, D        int16 // 8.8 fixed point
	CenterX     int16
	CenterY     int16
	MirrorH     bool
	MirrorV     bool
	OutsideMode uint8
	DirectColor bool
}

// MatrixPlane is the dedicated transform-source backing associated with a matrix
// channel. It stores both tilemap entries and dedicated pattern memory for a
// single tile-backed plane, decoupled from ordinary BG tilemap/tile storage.
type MatrixPlane struct {
	Enabled       bool
	Size          uint8 // 0 = 32x32, 1 = 64x64, 2 = 128x128
	SourceMode    uint8 // 0 = tilemap/pattern, 1 = bitmap
	BitmapPalette uint8 // palette bank used for bitmap-backed planes
	Transparent0  bool  // bitmap palette index 0 is transparent when set
	LiveFloorEnabled  bool
	LiveFloorHorizon  uint8
	LiveFloorCameraX  int16
	LiveFloorCameraY  int16
	LiveFloorHeadingX int16 // 8.8 fixed point
	LiveFloorHeadingY int16 // 8.8 fixed point
	Tilemap       [128 * 128 * 2]uint8
	Pattern       [32 * 1024]uint8
	Bitmap        [1024 * 1024 / 2]uint8 // 4bpp indexed bitmap backing, max 1024x1024
}

type matrixFloorRowCache struct {
	valid   bool
	scanline int
	startX  int32 // 16.16 fixed point
	startY  int32 // 16.16 fixed point
	stepX   int32 // 16.16 fixed point
	stepY   int32 // 16.16 fixed point
}

const (
	TilemapSize32x32   uint8 = 0
	TilemapSize64x64   uint8 = 1
	TilemapSize128x128 uint8 = 2

	MatrixPlaneSourceTilemap uint8 = 0
	MatrixPlaneSourceBitmap  uint8 = 1
)

// Window represents a window
type Window struct {
	Left, Right, Top, Bottom uint8
}

// NewPPU creates a new PPU instance
func NewPPU(logger *debug.Logger) *PPU {
	p := &PPU{
		BG0:             BackgroundLayer{},
		BG1:             BackgroundLayer{},
		BG2:             BackgroundLayer{},
		BG3:             BackgroundLayer{},
		Window0:         Window{},
		Window1:         Window{},
		Logger:          logger,
		activeScanlineY: -1,
	}
	p.initializeDefaultTransformBindings()
	return p
}

func (p *PPU) initializeDefaultTransformBindings() {
	p.BG0.Priority = 0
	p.BG1.Priority = 1
	p.BG2.Priority = 2
	p.BG3.Priority = 3
	p.BG0.TransformChannel = 0
	p.BG1.TransformChannel = 1
	p.BG2.TransformChannel = 2
	p.BG3.TransformChannel = 3
	for i := range p.MatrixPlanes {
		p.MatrixPlanes[i].Size = TilemapSize32x32
	}
}

func (p *PPU) reconcileTransformBindings() {
	// Old save states decode missing uint8 fields as zero, so treat the all-zero
	// BG1/BG2/BG3 binding pattern as "uninitialized defaults".
	if p.BG1.TransformChannel == 0 && p.BG2.TransformChannel == 0 && p.BG3.TransformChannel == 0 {
		p.BG1.TransformChannel = 1
		p.BG2.TransformChannel = 2
		p.BG3.TransformChannel = 3
	}
	if p.BG0.TransformChannel > 3 {
		p.BG0.TransformChannel = 0
	}
	if p.BG1.TransformChannel > 3 {
		p.BG1.TransformChannel = 1
	}
	if p.BG2.TransformChannel > 3 {
		p.BG2.TransformChannel = 2
	}
	if p.BG3.TransformChannel > 3 {
		p.BG3.TransformChannel = 3
	}
}

// SyncTransformBindingsForStateLoad restores default bindings for older states.
func (p *PPU) SyncTransformBindingsForStateLoad() {
	p.reconcileTransformBindings()
}

func (p *PPU) getBackgroundLayer(layerNum int) *BackgroundLayer {
	switch layerNum {
	case 0:
		return &p.BG0
	case 1:
		return &p.BG1
	case 2:
		return &p.BG2
	case 3:
		return &p.BG3
	default:
		return nil
	}
}

func (p *PPU) applyLayerControlRegister(layerNum int, value uint8) {
	layer := p.getBackgroundLayer(layerNum)
	if layer == nil {
		return
	}
	layer.Enabled = (value & 0x01) != 0
	layer.TileSize = (value & 0x02) != 0
	layer.Priority = (value >> 2) & 0x03
	layer.TilemapSize = (value >> 4) & 0x03
	if layer.TilemapSize > TilemapSize128x128 {
		layer.TilemapSize = TilemapSize32x32
	}
}

func (p *PPU) readLayerControlRegister(layerNum int) uint8 {
	layer := p.getBackgroundLayer(layerNum)
	if layer == nil {
		return 0
	}
	var value uint8
	if layer.Enabled {
		value |= 0x01
	}
	if layer.TileSize {
		value |= 0x02
	}
	value |= (layer.Priority & 0x03) << 2
	value |= (layer.TilemapSize & 0x03) << 4
	return value
}

func (p *PPU) applyLayerSourceModeRegister(layerNum int, value uint8) {
	layer := p.getBackgroundLayer(layerNum)
	if layer == nil {
		return
	}
	layer.SourceMode = value & 0x01
}

func (p *PPU) readLayerSourceModeRegister(layerNum int) uint8 {
	layer := p.getBackgroundLayer(layerNum)
	if layer == nil {
		return 0
	}
	return layer.SourceMode & 0x01
}

func (p *PPU) applyLayerTransformBindRegister(layerNum int, value uint8) {
	layer := p.getBackgroundLayer(layerNum)
	if layer == nil {
		return
	}
	layer.TransformChannel = value & 0x03
}

func (p *PPU) readLayerTransformBindRegister(layerNum int) uint8 {
	layer := p.getBackgroundLayer(layerNum)
	if layer == nil {
		return 0
	}
	return layer.TransformChannel & 0x03
}

func (p *PPU) writeLayerTilemapBaseLow(layerNum int, value uint8) {
	layer := p.getBackgroundLayer(layerNum)
	if layer == nil {
		return
	}
	layer.TilemapBase = (layer.TilemapBase & 0xFF00) | uint16(value)
}

func (p *PPU) writeLayerTilemapBaseHigh(layerNum int, value uint8) {
	layer := p.getBackgroundLayer(layerNum)
	if layer == nil {
		return
	}
	layer.TilemapBase = (layer.TilemapBase & 0x00FF) | (uint16(value) << 8)
}

func (p *PPU) readLayerTilemapBaseLow(layerNum int) uint8 {
	layer := p.getBackgroundLayer(layerNum)
	if layer == nil {
		return 0
	}
	return uint8(layer.TilemapBase & 0xFF)
}

func (p *PPU) readLayerTilemapBaseHigh(layerNum int) uint8 {
	layer := p.getBackgroundLayer(layerNum)
	if layer == nil {
		return 0
	}
	return uint8(layer.TilemapBase >> 8)
}

func (p *PPU) getTransformChannel(index uint8) *TransformChannel {
	if index > 3 {
		return &p.TransformChannels[0]
	}
	return &p.TransformChannels[index]
}

func (p *PPU) getLayerBoundTransformChannel(layerNum int) *TransformChannel {
	layer := p.getBackgroundLayer(layerNum)
	if layer == nil {
		return &p.TransformChannels[0]
	}
	return p.getTransformChannel(layer.TransformChannel)
}

func (p *PPU) resolveLayerTransformChannel(layerNum int) (*BackgroundLayer, *TransformChannel) {
	p.reconcileTransformBindings()
	layer := p.getBackgroundLayer(layerNum)
	if layer == nil {
		return nil, nil
	}
	return layer, p.getTransformChannel(layer.TransformChannel)
}

func (p *PPU) getMatrixPlane(channelIndex uint8) *MatrixPlane {
	if channelIndex >= uint8(len(p.MatrixPlanes)) {
		return &p.MatrixPlanes[0]
	}
	return &p.MatrixPlanes[channelIndex]
}

func (p *PPU) getSelectedMatrixPlane() *MatrixPlane {
	return p.getMatrixPlane(p.MatrixPlaneSelect)
}

func tilemapWidthForSizeMode(mode uint8) int {
	switch mode {
	case TilemapSize64x64:
		return 64
	case TilemapSize128x128:
		return 128
	default:
		return 32
	}
}

func (p *PPU) applySelectedMatrixPlaneControl(value uint8) {
	plane := p.getSelectedMatrixPlane()
	plane.Enabled = (value & 0x01) != 0
	plane.Size = (value >> 1) & 0x03
	if plane.Size > TilemapSize128x128 {
		plane.Size = TilemapSize32x32
	}
	plane.SourceMode = (value >> 3) & 0x01
	plane.BitmapPalette = (value >> 4) & 0x0F
}

func (p *PPU) readSelectedMatrixPlaneControl() uint8 {
	plane := p.getSelectedMatrixPlane()
	var value uint8
	if plane.Enabled {
		value |= 0x01
	}
	value |= (plane.Size & 0x03) << 1
	value |= (plane.SourceMode & 0x01) << 3
	value |= (plane.BitmapPalette & 0x0F) << 4
	return value
}

func (p *PPU) applySelectedMatrixPlaneFlags(value uint8) {
	plane := p.getSelectedMatrixPlane()
	plane.Transparent0 = (value & 0x01) != 0
}

func (p *PPU) readSelectedMatrixPlaneFlags() uint8 {
	plane := p.getSelectedMatrixPlane()
	var value uint8
	if plane.Transparent0 {
		value |= 0x01
	}
	return value
}

func (p *PPU) invalidateSelectedMatrixFloorRowCache() {
	if int(p.MatrixPlaneSelect) < len(p.MatrixFloorRows) {
		p.MatrixFloorRows[p.MatrixPlaneSelect] = matrixFloorRowCache{}
	}
}

func (p *PPU) applySelectedMatrixPlaneLiveFloorControl(value uint8) {
	plane := p.getSelectedMatrixPlane()
	plane.LiveFloorEnabled = (value & 0x01) != 0
	p.invalidateSelectedMatrixFloorRowCache()
}

func (p *PPU) readSelectedMatrixPlaneLiveFloorControl() uint8 {
	plane := p.getSelectedMatrixPlane()
	var value uint8
	if plane.LiveFloorEnabled {
		value |= 0x01
	}
	return value
}

func (p *PPU) writeSelectedMatrixPlaneLiveFloorHorizon(value uint8) {
	plane := p.getSelectedMatrixPlane()
	plane.LiveFloorHorizon = value
	p.invalidateSelectedMatrixFloorRowCache()
}

func (p *PPU) readSelectedMatrixPlaneLiveFloorHorizon() uint8 {
	return p.getSelectedMatrixPlane().LiveFloorHorizon
}

func (p *PPU) writeSelectedMatrixPlaneLiveFloorCameraXLow(value uint8) {
	plane := p.getSelectedMatrixPlane()
	plane.LiveFloorCameraX = int16((uint16(plane.LiveFloorCameraX) & 0xFF00) | uint16(value))
	p.invalidateSelectedMatrixFloorRowCache()
}

func (p *PPU) writeSelectedMatrixPlaneLiveFloorCameraXHigh(value uint8) {
	plane := p.getSelectedMatrixPlane()
	plane.LiveFloorCameraX = int16((uint16(plane.LiveFloorCameraX) & 0x00FF) | (uint16(value) << 8))
	p.invalidateSelectedMatrixFloorRowCache()
}

func (p *PPU) readSelectedMatrixPlaneLiveFloorCameraXLow() uint8 {
	return uint8(uint16(p.getSelectedMatrixPlane().LiveFloorCameraX) & 0x00FF)
}

func (p *PPU) readSelectedMatrixPlaneLiveFloorCameraXHigh() uint8 {
	return uint8((uint16(p.getSelectedMatrixPlane().LiveFloorCameraX) >> 8) & 0x00FF)
}

func (p *PPU) writeSelectedMatrixPlaneLiveFloorCameraYLow(value uint8) {
	plane := p.getSelectedMatrixPlane()
	plane.LiveFloorCameraY = int16((uint16(plane.LiveFloorCameraY) & 0xFF00) | uint16(value))
	p.invalidateSelectedMatrixFloorRowCache()
}

func (p *PPU) writeSelectedMatrixPlaneLiveFloorCameraYHigh(value uint8) {
	plane := p.getSelectedMatrixPlane()
	plane.LiveFloorCameraY = int16((uint16(plane.LiveFloorCameraY) & 0x00FF) | (uint16(value) << 8))
	p.invalidateSelectedMatrixFloorRowCache()
}

func (p *PPU) readSelectedMatrixPlaneLiveFloorCameraYLow() uint8 {
	return uint8(uint16(p.getSelectedMatrixPlane().LiveFloorCameraY) & 0x00FF)
}

func (p *PPU) readSelectedMatrixPlaneLiveFloorCameraYHigh() uint8 {
	return uint8((uint16(p.getSelectedMatrixPlane().LiveFloorCameraY) >> 8) & 0x00FF)
}

func (p *PPU) writeSelectedMatrixPlaneLiveFloorHeadingXLow(value uint8) {
	plane := p.getSelectedMatrixPlane()
	plane.LiveFloorHeadingX = int16((uint16(plane.LiveFloorHeadingX) & 0xFF00) | uint16(value))
	p.invalidateSelectedMatrixFloorRowCache()
}

func (p *PPU) writeSelectedMatrixPlaneLiveFloorHeadingXHigh(value uint8) {
	plane := p.getSelectedMatrixPlane()
	plane.LiveFloorHeadingX = int16((uint16(plane.LiveFloorHeadingX) & 0x00FF) | (uint16(value) << 8))
	p.invalidateSelectedMatrixFloorRowCache()
}

func (p *PPU) readSelectedMatrixPlaneLiveFloorHeadingXLow() uint8 {
	return uint8(uint16(p.getSelectedMatrixPlane().LiveFloorHeadingX) & 0x00FF)
}

func (p *PPU) readSelectedMatrixPlaneLiveFloorHeadingXHigh() uint8 {
	return uint8((uint16(p.getSelectedMatrixPlane().LiveFloorHeadingX) >> 8) & 0x00FF)
}

func (p *PPU) writeSelectedMatrixPlaneLiveFloorHeadingYLow(value uint8) {
	plane := p.getSelectedMatrixPlane()
	plane.LiveFloorHeadingY = int16((uint16(plane.LiveFloorHeadingY) & 0xFF00) | uint16(value))
	p.invalidateSelectedMatrixFloorRowCache()
}

func (p *PPU) writeSelectedMatrixPlaneLiveFloorHeadingYHigh(value uint8) {
	plane := p.getSelectedMatrixPlane()
	plane.LiveFloorHeadingY = int16((uint16(plane.LiveFloorHeadingY) & 0x00FF) | (uint16(value) << 8))
	p.invalidateSelectedMatrixFloorRowCache()
}

func (p *PPU) readSelectedMatrixPlaneLiveFloorHeadingYLow() uint8 {
	return uint8(uint16(p.getSelectedMatrixPlane().LiveFloorHeadingY) & 0x00FF)
}

func (p *PPU) readSelectedMatrixPlaneLiveFloorHeadingYHigh() uint8 {
	return uint8((uint16(p.getSelectedMatrixPlane().LiveFloorHeadingY) >> 8) & 0x00FF)
}

func (p *PPU) readSelectedMatrixPlanePatternAddrLow() uint8 {
	return uint8(p.MatrixPlanePatternAddr & 0x00FF)
}

func (p *PPU) readSelectedMatrixPlanePatternAddrHigh() uint8 {
	return uint8((p.MatrixPlanePatternAddr >> 8) & 0x7F)
}

func (p *PPU) writeSelectedMatrixPlanePatternAddrLow(value uint8) {
	p.MatrixPlanePatternAddr = (p.MatrixPlanePatternAddr & 0xFF00) | uint16(value)
}

func (p *PPU) writeSelectedMatrixPlanePatternAddrHigh(value uint8) {
	p.MatrixPlanePatternAddr = (p.MatrixPlanePatternAddr & 0x00FF) | (uint16(value&0x7F) << 8)
}

func (p *PPU) readSelectedMatrixPlaneBitmapAddrLow() uint8 {
	return uint8(p.MatrixPlaneBitmapAddr & 0xFF)
}

func (p *PPU) readSelectedMatrixPlaneBitmapAddrMid() uint8 {
	return uint8((p.MatrixPlaneBitmapAddr >> 8) & 0xFF)
}

func (p *PPU) readSelectedMatrixPlaneBitmapAddrHigh() uint8 {
	return uint8((p.MatrixPlaneBitmapAddr >> 16) & 0x07)
}

func (p *PPU) writeSelectedMatrixPlaneBitmapAddrLow(value uint8) {
	p.MatrixPlaneBitmapAddr = (p.MatrixPlaneBitmapAddr & 0x7FFFF00) | uint32(value)
}

func (p *PPU) writeSelectedMatrixPlaneBitmapAddrMid(value uint8) {
	p.MatrixPlaneBitmapAddr = (p.MatrixPlaneBitmapAddr & 0x7FF00FF) | (uint32(value) << 8)
}

func (p *PPU) writeSelectedMatrixPlaneBitmapAddrHigh(value uint8) {
	p.MatrixPlaneBitmapAddr = (p.MatrixPlaneBitmapAddr & 0x0FFFF) | (uint32(value&0x07) << 16)
}

func (p *PPU) matrixTilemapEntry(layer *BackgroundLayer, tileX, tileY int) (tileIndex uint8, attributes uint8, ok bool) {
	if layer == nil {
		return 0, 0, false
	}
	plane := p.getMatrixPlane(layer.TransformChannel)
	if plane.Enabled {
		width := tilemapWidthForSizeMode(plane.Size)
		offset := (tileY*width + tileX) * 2
		if offset < 0 || offset+1 >= len(plane.Tilemap) {
			return 0, 0, false
		}
		return plane.Tilemap[offset], plane.Tilemap[offset+1], true
	}

	width := tilemapWidthForSizeMode(layer.TilemapSize)
	tilemapBase := uint16(0x4000)
	if layer.TilemapBase != 0 {
		tilemapBase = layer.TilemapBase
	}
	tilemapOffset := uint16((tileY*width + tileX) * 2)
	if uint32(tilemapBase)+uint32(tilemapOffset) >= 65536 {
		return 0, 0, false
	}
	addr := tilemapBase + tilemapOffset
	return p.VRAM[addr], p.VRAM[addr+1], true
}

// Read8 reads an 8-bit value from PPU registers
func (p *PPU) Read8(offset uint16) uint8 {
	switch offset {
	case 0x08: // BG0_CONTROL
		return p.readLayerControlRegister(0)
	case 0x09: // BG1_CONTROL
		return p.readLayerControlRegister(1)
	case 0x21: // BG2_CONTROL
		return p.readLayerControlRegister(2)
	case 0x26: // BG3_CONTROL
		return p.readLayerControlRegister(3)
	case 0x68: // BG0_SOURCE_MODE
		return p.readLayerSourceModeRegister(0)
	case 0x69: // BG1_SOURCE_MODE
		return p.readLayerSourceModeRegister(1)
	case 0x6A: // BG2_SOURCE_MODE
		return p.readLayerSourceModeRegister(2)
	case 0x6B: // BG3_SOURCE_MODE
		return p.readLayerSourceModeRegister(3)
	case 0x6C: // BG0_TRANSFORM_BIND
		return p.readLayerTransformBindRegister(0)
	case 0x6D: // BG1_TRANSFORM_BIND
		return p.readLayerTransformBindRegister(1)
	case 0x6E: // BG2_TRANSFORM_BIND
		return p.readLayerTransformBindRegister(2)
	case 0x6F: // BG3_TRANSFORM_BIND
		return p.readLayerTransformBindRegister(3)
	case 0x77: // BG0_TILEMAP_BASE_L
		return p.readLayerTilemapBaseLow(0)
	case 0x78: // BG0_TILEMAP_BASE_H
		return p.readLayerTilemapBaseHigh(0)
	case 0x79: // BG1_TILEMAP_BASE_L
		return p.readLayerTilemapBaseLow(1)
	case 0x7A: // BG1_TILEMAP_BASE_H
		return p.readLayerTilemapBaseHigh(1)
	case 0x7B: // BG2_TILEMAP_BASE_L
		return p.readLayerTilemapBaseLow(2)
	case 0x7C: // BG2_TILEMAP_BASE_H
		return p.readLayerTilemapBaseHigh(2)
	case 0x7D: // BG3_TILEMAP_BASE_L
		return p.readLayerTilemapBaseLow(3)
	case 0x7E: // BG3_TILEMAP_BASE_H
		return p.readLayerTilemapBaseHigh(3)
	case 0x5D: // HDMA_CONTROL
		return p.HDMAControl
	case 0x5E: // HDMA_TABLE_BASE_L
		return uint8(p.HDMATableBase & 0xFF)
	case 0x5F: // HDMA_TABLE_BASE_H
		return uint8(p.HDMATableBase >> 8)
	case 0x7F: // HDMA_EXTENSION_CONTROL
		return p.HDMAExtControl
	case 0x80: // MATRIX_PLANE_SELECT
		return p.MatrixPlaneSelect & 0x03
	case 0x81: // MATRIX_PLANE_CONTROL
		return p.readSelectedMatrixPlaneControl()
	case 0x82: // MATRIX_PLANE_ADDR_L
		return uint8(p.MatrixPlaneAddr & 0xFF)
	case 0x83: // MATRIX_PLANE_ADDR_H
		return uint8((p.MatrixPlaneAddr >> 8) & 0x7F)
	case 0x84: // MATRIX_PLANE_DATA
		plane := p.getSelectedMatrixPlane()
		addr := int(p.MatrixPlaneAddr) & (len(plane.Tilemap) - 1)
		value := plane.Tilemap[addr]
		p.MatrixPlaneAddr = uint16((addr + 1) & (len(plane.Tilemap) - 1))
		return value
	case 0x85: // MATRIX_PLANE_PATTERN_ADDR_L
		return p.readSelectedMatrixPlanePatternAddrLow()
	case 0x86: // MATRIX_PLANE_PATTERN_ADDR_H
		return p.readSelectedMatrixPlanePatternAddrHigh()
	case 0x87: // MATRIX_PLANE_PATTERN_DATA
		plane := p.getSelectedMatrixPlane()
		addr := int(p.MatrixPlanePatternAddr) & (len(plane.Pattern) - 1)
		value := plane.Pattern[addr]
		p.MatrixPlanePatternAddr = uint16((addr + 1) & (len(plane.Pattern) - 1))
		return value
	case 0x88: // MATRIX_PLANE_BITMAP_ADDR_L
		return p.readSelectedMatrixPlaneBitmapAddrLow()
	case 0x89: // MATRIX_PLANE_BITMAP_ADDR_M
		return p.readSelectedMatrixPlaneBitmapAddrMid()
	case 0x8A: // MATRIX_PLANE_BITMAP_ADDR_H
		return p.readSelectedMatrixPlaneBitmapAddrHigh()
	case 0x8B: // MATRIX_PLANE_BITMAP_DATA
		plane := p.getSelectedMatrixPlane()
		addr := int(p.MatrixPlaneBitmapAddr) & (len(plane.Bitmap) - 1)
		value := plane.Bitmap[addr]
		p.MatrixPlaneBitmapAddr = uint32((addr + 1) & (len(plane.Bitmap) - 1))
		return value
	case 0x8C: // MATRIX_PLANE_FLAGS
		return p.readSelectedMatrixPlaneFlags()
	case 0x8D: // MATRIX_PLANE_LIVE_FLOOR_CONTROL
		return p.readSelectedMatrixPlaneLiveFloorControl()
	case 0x8E: // MATRIX_PLANE_LIVE_FLOOR_HORIZON
		return p.readSelectedMatrixPlaneLiveFloorHorizon()
	case 0x8F: // MATRIX_PLANE_LIVE_FLOOR_CAMERA_X_L
		return p.readSelectedMatrixPlaneLiveFloorCameraXLow()
	case 0x90: // MATRIX_PLANE_LIVE_FLOOR_CAMERA_X_H
		return p.readSelectedMatrixPlaneLiveFloorCameraXHigh()
	case 0x91: // MATRIX_PLANE_LIVE_FLOOR_CAMERA_Y_L
		return p.readSelectedMatrixPlaneLiveFloorCameraYLow()
	case 0x92: // MATRIX_PLANE_LIVE_FLOOR_CAMERA_Y_H
		return p.readSelectedMatrixPlaneLiveFloorCameraYHigh()
	case 0x93: // MATRIX_PLANE_LIVE_FLOOR_HEADING_X_L
		return p.readSelectedMatrixPlaneLiveFloorHeadingXLow()
	case 0x94: // MATRIX_PLANE_LIVE_FLOOR_HEADING_X_H
		return p.readSelectedMatrixPlaneLiveFloorHeadingXHigh()
	case 0x95: // MATRIX_PLANE_LIVE_FLOOR_HEADING_Y_L
		return p.readSelectedMatrixPlaneLiveFloorHeadingYLow()
	case 0x96: // MATRIX_PLANE_LIVE_FLOOR_HEADING_Y_H
		return p.readSelectedMatrixPlaneLiveFloorHeadingYHigh()
	case 0x18: // MATRIX_CONTROL (BG0)
		channel := p.getLayerBoundTransformChannel(0)
		var value uint8
		if channel.Enabled {
			value |= 0x01
		}
		if channel.MirrorH {
			value |= 0x02
		}
		if channel.MirrorV {
			value |= 0x04
		}
		value |= (channel.OutsideMode & 0x03) << 3
		if channel.DirectColor {
			value |= 0x20
		}
		return value
	case 0x2B: // BG1_MATRIX_CONTROL
		channel := p.getLayerBoundTransformChannel(1)
		var value uint8
		if channel.Enabled {
			value |= 0x01
		}
		if channel.MirrorH {
			value |= 0x02
		}
		if channel.MirrorV {
			value |= 0x04
		}
		value |= (channel.OutsideMode & 0x03) << 3
		if channel.DirectColor {
			value |= 0x20
		}
		return value
	case 0x38: // BG2_MATRIX_CONTROL
		channel := p.getLayerBoundTransformChannel(2)
		var value uint8
		if channel.Enabled {
			value |= 0x01
		}
		if channel.MirrorH {
			value |= 0x02
		}
		if channel.MirrorV {
			value |= 0x04
		}
		value |= (channel.OutsideMode & 0x03) << 3
		if channel.DirectColor {
			value |= 0x20
		}
		return value
	case 0x45: // BG3_MATRIX_CONTROL
		channel := p.getLayerBoundTransformChannel(3)
		var value uint8
		if channel.Enabled {
			value |= 0x01
		}
		if channel.MirrorH {
			value |= 0x02
		}
		if channel.MirrorV {
			value |= 0x04
		}
		value |= (channel.OutsideMode & 0x03) << 3
		if channel.DirectColor {
			value |= 0x20
		}
		return value
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
			addr := uint16(p.OAMAddr)*6 + uint16(p.OAMByteIndex)
			if addr < 768 {
				value := p.OAM[addr]
				// Increment byte index (like write does)
				p.OAMByteIndex++
				if p.OAMByteIndex >= 6 {
					// Move to next sprite after reading 6 bytes
					p.OAMByteIndex = 0
					p.OAMAddr++
					if p.OAMAddr > 127 {
						p.OAMAddr = 0
					}
				}
				return value
			}
		}
		return 0
	case 0x3E: // VBLANK_FLAG (one-shot: cleared when read)
		// VBlank flag: hardware-accurate synchronization signal
		// Set at start of VBlank period (scanline 200), cleared when read (one-shot)
		// Bit 0 = VBlank active (1 = VBlank period, 0 = not VBlank)
		// This matches real hardware behavior (NES, SNES, etc.)
		//
		// IMPORTANT: The flag persists through the entire VBlank period (scanlines 200-219).
		// If ROM reads it during VBlank and clears it, we re-set it so it's available
		// for the rest of VBlank. This allows ROM to read the flag multiple times during
		// VBlank if needed (though typically only once).
		//
		// CRITICAL FIX: Check if we're in VBlank BEFORE reading the flag value.
		// This ensures the flag is set correctly even if it was cleared by a previous read.
		inVBlank := p.currentScanline >= VisibleScanlines && p.currentScanline < TotalScanlines

		flag := p.VBlankFlag

		// If we're in VBlank period, the flag should always be true
		// This fixes the issue where ROM reads flag multiple times during VBlank
		if inVBlank {
			flag = true
		}

		// Clear flag after read (one-shot behavior)
		// But immediately re-set if still in VBlank period
		p.VBlankFlag = false
		if inVBlank {
			p.VBlankFlag = true
		}

		if p.Logger != nil {
			p.Logger.LogPPUf(debug.LogLevelDebug,
				"VBlank flag read: scanline=%d, dot=%d, inVBlank=%v, flag=%v, returning=0x%02X",
				p.currentScanline, p.currentDot, inVBlank, flag, map[bool]uint8{true: 0x01, false: 0x00}[flag])
		}
		if flag {
			return 0x01
		}
		return 0x00
	case 0x3F: // FRAME_COUNTER_LOW
		return uint8(p.FrameCounter & 0xFF)
	case 0x40: // FRAME_COUNTER_HIGH
		return uint8(p.FrameCounter >> 8)
	case 0x60: // DMA_STATUS
		// Bit 0: DMA active (1=transferring, 0=idle)
		if p.DMAEnabled && p.DMAProgress < p.DMALength {
			return 0x01
		}
		return 0x00
	case 0x61: // DMA_SOURCE_BANK
		return p.DMASourceBank
	case 0x62: // DMA_SOURCE_OFFSET_L
		return uint8(p.DMASourceOffset & 0xFF)
	case 0x63: // DMA_SOURCE_OFFSET_H
		return uint8(p.DMASourceOffset >> 8)
	case 0x64: // DMA_DEST_ADDR_L
		return uint8(p.DMADestAddr & 0xFF)
	case 0x65: // DMA_DEST_ADDR_H
		return uint8(p.DMADestAddr >> 8)
	case 0x66: // DMA_LENGTH_L
		return uint8(p.DMALength & 0xFF)
	case 0x67: // DMA_LENGTH_H
		return uint8(p.DMALength >> 8)
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
		p.applyLayerControlRegister(0, value)
	case 0x09: // BG1_CONTROL
		p.applyLayerControlRegister(1, value)

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
		// Only log VRAM writes during initialization (first frame) and only first 32 bytes
		if p.Logger != nil && p.FrameCounter == 0 && p.VRAMAddr < 32 {
			p.Logger.LogPPUf(debug.LogLevelDebug, "VRAM_DATA write: addr=0x%04X, value=0x%02X", p.VRAMAddr, value)
		}
		p.VRAM[p.VRAMAddr] = value
		p.VRAMAddr++
		if p.VRAMAddr > 0xFFFF {
			p.VRAMAddr = 0
		}

	// CGRAM access
	case 0x12: // CGRAM_ADDR
		// Only log CGRAM_ADDR during initialization (first frame)
		if p.Logger != nil && p.FrameCounter == 0 && value < 64 {
			paletteIndex := value / 32
			colorIndex := (value / 2) % 16
			p.Logger.LogPPUf(debug.LogLevelDebug, "CGRAM_ADDR write: 0x%02X (palette %d, color %d)", value, paletteIndex, colorIndex)
		}
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
				cgramIndex := p.CGRAMAddr
				// Only log CGRAM_DATA during initialization (first frame) and only first 20 colors
				if p.Logger != nil && p.FrameCounter == 0 && addr < 40 {
					paletteIndex := cgramIndex / 32
					colorIndex := (cgramIndex / 2) % 16
					p.Logger.LogPPUf(debug.LogLevelDebug, "CGRAM_DATA write complete: addr=0x%02X (palette %d, color %d), RGB555=0x%04X",
						cgramIndex, paletteIndex, colorIndex, p.CGRAMWriteValue)
				}
				// Store in little-endian order: low byte first, high byte second
				p.CGRAM[addr] = uint8(p.CGRAMWriteValue & 0xFF) // Low byte
				p.CGRAM[addr+1] = uint8(p.CGRAMWriteValue >> 8) // High byte
				p.updateCGRAMCacheEntry(cgramIndex)
				p.CGRAMAddr++
				if p.CGRAMAddr > 255 {
					p.CGRAMAddr = 0
				}
			}
			p.CGRAMWriteLatch = false
		}

	// OAM access
	case 0x14: // OAM_ADDR
		// Only log OAM_ADDR writes occasionally (every 60 frames) to reduce performance impact
		if p.Logger != nil && p.FrameCounter%60 == 0 && value < 4 {
			p.Logger.LogPPUf(debug.LogLevelDebug, "OAM_ADDR write: 0x%02X (sprite %d), byte index was %d, resetting to 0",
				value, value, p.OAMByteIndex)
		}
		// OAM writes are only allowed during VBlank period (hardware-accurate)
		// During visible rendering (scanlines 0-199), OAM is locked
		// Allow writes if: VBlank period (scanline >= 200) OR frame hasn't started yet OR first frame (initialization)
		// Note: ROM should wait for VBlank before updating sprites to avoid wavy artifacts
		if p.currentScanline < 200 && p.frameStarted && p.FrameCounter > 1 {
			if p.Logger != nil {
				p.Logger.LogPPUf(debug.LogLevelWarning, "OAM_ADDR write ignored during visible rendering (scanline %d)", p.currentScanline)
			}
			return
		}
		p.OAMAddr = value
		if p.OAMAddr > 127 {
			p.OAMAddr = 127
		}
		p.OAMByteIndex = 0 // Reset byte index when setting sprite address
		// Removed frequent logging - only log occasionally above
	case 0x15: // OAM_DATA
		// OAM writes are only allowed during VBlank period (hardware-accurate)
		// During visible rendering (scanlines 0-199), OAM is locked
		// Allow writes if: VBlank period (scanline >= 200) OR frame hasn't started yet OR first frame (initialization)
		// Note: ROM should wait for VBlank before updating sprites to avoid wavy artifacts
		if p.currentScanline < 200 && p.frameStarted && p.FrameCounter > 1 {
			if p.Logger != nil {
				p.Logger.LogPPUf(debug.LogLevelWarning, "OAM_DATA write ignored during visible rendering (scanline %d)", p.currentScanline)
			}
			return
		}
		// Only log OAM_DATA writes occasionally (every 60 frames) and only for first few sprites
		// Log only when completing a sprite (byte 5 = Ctrl) to reduce verbosity
		if p.Logger != nil && p.FrameCounter%60 == 0 && p.OAMByteIndex == 5 && p.OAMAddr < 4 {
			spriteID := p.OAMAddr
			p.Logger.LogPPUf(debug.LogLevelDebug, "OAM_DATA: sprite=%d complete (Ctrl=0x%02X), addr=%d",
				spriteID, value, uint16(p.OAMAddr)*6+uint16(p.OAMByteIndex))
		}
		addr := uint16(p.OAMAddr)*6 + uint16(p.OAMByteIndex)
		if addr < 768 {
			p.OAM[addr] = value
			// Removed frequent logging - only log occasionally above
			p.OAMByteIndex++
			if p.OAMByteIndex >= 6 {
				// Move to next sprite after writing 6 bytes
				p.OAMByteIndex = 0
				p.OAMAddr++
				if p.OAMAddr > 127 {
					p.OAMAddr = 0
				}
			}
			// Debug logging removed for performance - use -log flag to enable PPU logging if needed
		} else {
			if p.Logger != nil {
				p.Logger.LogPPUf(debug.LogLevelWarning, "OAM_DATA write out of bounds: addr=%d (max 767)", addr)
			}
		}

	// Matrix Mode (Legacy - maps to BG0 for backward compatibility)
	case 0x18: // MATRIX_CONTROL (BG0)
		channel := p.getLayerBoundTransformChannel(0)
		channel.Enabled = (value & 0x01) != 0
		channel.MirrorH = (value & 0x02) != 0
		channel.MirrorV = (value & 0x04) != 0
		channel.OutsideMode = (value >> 3) & 0x3
		channel.DirectColor = (value & 0x20) != 0
	case 0x19: // MATRIX_A_L (BG0)
		channel := p.getLayerBoundTransformChannel(0)
		channel.A = int16((uint16(channel.A) & 0xFF00) | uint16(value))
	case 0x1A: // MATRIX_A_H (BG0)
		channel := p.getLayerBoundTransformChannel(0)
		channel.A = int16((uint16(channel.A) & 0x00FF) | (uint16(value) << 8))
	case 0x1B: // MATRIX_B_L (BG0)
		channel := p.getLayerBoundTransformChannel(0)
		channel.B = int16((uint16(channel.B) & 0xFF00) | uint16(value))
	case 0x1C: // MATRIX_B_H (BG0)
		channel := p.getLayerBoundTransformChannel(0)
		channel.B = int16((uint16(channel.B) & 0x00FF) | (uint16(value) << 8))
	case 0x1D: // MATRIX_C_L (BG0)
		channel := p.getLayerBoundTransformChannel(0)
		channel.C = int16((uint16(channel.C) & 0xFF00) | uint16(value))
	case 0x1E: // MATRIX_C_H (BG0)
		channel := p.getLayerBoundTransformChannel(0)
		channel.C = int16((uint16(channel.C) & 0x00FF) | (uint16(value) << 8))
	case 0x1F: // MATRIX_D_L (BG0)
		channel := p.getLayerBoundTransformChannel(0)
		channel.D = int16((uint16(channel.D) & 0xFF00) | uint16(value))
	case 0x20: // MATRIX_D_H (BG0)
		channel := p.getLayerBoundTransformChannel(0)
		channel.D = int16((uint16(channel.D) & 0x00FF) | (uint16(value) << 8))

	// BG2/BG3 control
	case 0x21: // BG2_CONTROL
		p.applyLayerControlRegister(2, value)
	case 0x22: // BG3_SCROLLX_L
		p.BG3.ScrollX = int16((uint16(p.BG3.ScrollX) & 0xFF00) | uint16(value))
	case 0x23: // BG3_SCROLLX_H
		p.BG3.ScrollX = int16((uint16(p.BG3.ScrollX) & 0x00FF) | (uint16(value) << 8))
	case 0x24: // BG3_SCROLLY_L
		p.BG3.ScrollY = int16((uint16(p.BG3.ScrollY) & 0xFF00) | uint16(value))
	case 0x25: // BG3_SCROLLY_H
		p.BG3.ScrollY = int16((uint16(p.BG3.ScrollY) & 0x00FF) | (uint16(value) << 8))
	case 0x26: // BG3_CONTROL
		p.applyLayerControlRegister(3, value)

	// Matrix center (BG0)
	case 0x27: // MATRIX_CENTER_X_L (BG0)
		channel := p.getLayerBoundTransformChannel(0)
		channel.CenterX = int16((uint16(channel.CenterX) & 0xFF00) | uint16(value))
	case 0x28: // MATRIX_CENTER_X_H (BG0)
		channel := p.getLayerBoundTransformChannel(0)
		channel.CenterX = int16((uint16(channel.CenterX) & 0x00FF) | (uint16(value) << 8))
	case 0x29: // MATRIX_CENTER_Y_L (BG0)
		channel := p.getLayerBoundTransformChannel(0)
		channel.CenterY = int16((uint16(channel.CenterY) & 0xFF00) | uint16(value))
	case 0x2A: // MATRIX_CENTER_Y_H (BG0)
		channel := p.getLayerBoundTransformChannel(0)
		channel.CenterY = int16((uint16(channel.CenterY) & 0x00FF) | (uint16(value) << 8))

	// BG1 Matrix Mode (per-layer transformation)
	case 0x2B: // BG1_MATRIX_CONTROL
		channel := p.getLayerBoundTransformChannel(1)
		channel.Enabled = (value & 0x01) != 0
		channel.MirrorH = (value & 0x02) != 0
		channel.MirrorV = (value & 0x04) != 0
		channel.OutsideMode = (value >> 3) & 0x3
		channel.DirectColor = (value & 0x20) != 0
	case 0x2C: // BG1_MATRIX_A_L
		channel := p.getLayerBoundTransformChannel(1)
		channel.A = int16((uint16(channel.A) & 0xFF00) | uint16(value))
	case 0x2D: // BG1_MATRIX_A_H
		channel := p.getLayerBoundTransformChannel(1)
		channel.A = int16((uint16(channel.A) & 0x00FF) | (uint16(value) << 8))
	case 0x2E: // BG1_MATRIX_B_L
		channel := p.getLayerBoundTransformChannel(1)
		channel.B = int16((uint16(channel.B) & 0xFF00) | uint16(value))
	case 0x2F: // BG1_MATRIX_B_H
		channel := p.getLayerBoundTransformChannel(1)
		channel.B = int16((uint16(channel.B) & 0x00FF) | (uint16(value) << 8))
	case 0x30: // BG1_MATRIX_C_L
		channel := p.getLayerBoundTransformChannel(1)
		channel.C = int16((uint16(channel.C) & 0xFF00) | uint16(value))
	case 0x31: // BG1_MATRIX_C_H
		channel := p.getLayerBoundTransformChannel(1)
		channel.C = int16((uint16(channel.C) & 0x00FF) | (uint16(value) << 8))
	case 0x32: // BG1_MATRIX_D_L
		channel := p.getLayerBoundTransformChannel(1)
		channel.D = int16((uint16(channel.D) & 0xFF00) | uint16(value))
	case 0x33: // BG1_MATRIX_D_H
		channel := p.getLayerBoundTransformChannel(1)
		channel.D = int16((uint16(channel.D) & 0x00FF) | (uint16(value) << 8))
	case 0x34: // BG1_MATRIX_CENTER_X_L
		channel := p.getLayerBoundTransformChannel(1)
		channel.CenterX = int16((uint16(channel.CenterX) & 0xFF00) | uint16(value))
	case 0x35: // BG1_MATRIX_CENTER_X_H
		channel := p.getLayerBoundTransformChannel(1)
		channel.CenterX = int16((uint16(channel.CenterX) & 0x00FF) | (uint16(value) << 8))
	case 0x36: // BG1_MATRIX_CENTER_Y_L
		channel := p.getLayerBoundTransformChannel(1)
		channel.CenterY = int16((uint16(channel.CenterY) & 0xFF00) | uint16(value))
	case 0x37: // BG1_MATRIX_CENTER_Y_H
		channel := p.getLayerBoundTransformChannel(1)
		channel.CenterY = int16((uint16(channel.CenterY) & 0x00FF) | (uint16(value) << 8))

	// BG2 Matrix Mode
	case 0x38: // BG2_MATRIX_CONTROL
		channel := p.getLayerBoundTransformChannel(2)
		channel.Enabled = (value & 0x01) != 0
		channel.MirrorH = (value & 0x02) != 0
		channel.MirrorV = (value & 0x04) != 0
		channel.OutsideMode = (value >> 3) & 0x3
		channel.DirectColor = (value & 0x20) != 0
	case 0x39: // BG2_MATRIX_A_L
		channel := p.getLayerBoundTransformChannel(2)
		channel.A = int16((uint16(channel.A) & 0xFF00) | uint16(value))
	case 0x3A: // BG2_MATRIX_A_H
		channel := p.getLayerBoundTransformChannel(2)
		channel.A = int16((uint16(channel.A) & 0x00FF) | (uint16(value) << 8))
	case 0x3B: // BG2_MATRIX_B_L
		channel := p.getLayerBoundTransformChannel(2)
		channel.B = int16((uint16(channel.B) & 0xFF00) | uint16(value))
	case 0x3C: // BG2_MATRIX_B_H
		channel := p.getLayerBoundTransformChannel(2)
		channel.B = int16((uint16(channel.B) & 0x00FF) | (uint16(value) << 8))
	case 0x3D: // BG2_MATRIX_C_L
		channel := p.getLayerBoundTransformChannel(2)
		channel.C = int16((uint16(channel.C) & 0xFF00) | uint16(value))
	case 0x3E: // BG2_MATRIX_C_H
		channel := p.getLayerBoundTransformChannel(2)
		channel.C = int16((uint16(channel.C) & 0x00FF) | (uint16(value) << 8))
	case 0x3F: // BG2_MATRIX_D_L
		channel := p.getLayerBoundTransformChannel(2)
		channel.D = int16((uint16(channel.D) & 0xFF00) | uint16(value))
	case 0x40: // BG2_MATRIX_D_H
		channel := p.getLayerBoundTransformChannel(2)
		channel.D = int16((uint16(channel.D) & 0x00FF) | (uint16(value) << 8))
	case 0x41: // BG2_MATRIX_CENTER_X_L
		channel := p.getLayerBoundTransformChannel(2)
		channel.CenterX = int16((uint16(channel.CenterX) & 0xFF00) | uint16(value))
	case 0x42: // BG2_MATRIX_CENTER_X_H
		channel := p.getLayerBoundTransformChannel(2)
		channel.CenterX = int16((uint16(channel.CenterX) & 0x00FF) | (uint16(value) << 8))
	case 0x43: // BG2_MATRIX_CENTER_Y_L
		channel := p.getLayerBoundTransformChannel(2)
		channel.CenterY = int16((uint16(channel.CenterY) & 0xFF00) | uint16(value))
	case 0x44: // BG2_MATRIX_CENTER_Y_H
		channel := p.getLayerBoundTransformChannel(2)
		channel.CenterY = int16((uint16(channel.CenterY) & 0x00FF) | (uint16(value) << 8))

	// BG3 Matrix Mode
	case 0x45: // BG3_MATRIX_CONTROL
		channel := p.getLayerBoundTransformChannel(3)
		channel.Enabled = (value & 0x01) != 0
		channel.MirrorH = (value & 0x02) != 0
		channel.MirrorV = (value & 0x04) != 0
		channel.OutsideMode = (value >> 3) & 0x3
		channel.DirectColor = (value & 0x20) != 0
	case 0x46: // BG3_MATRIX_A_L
		channel := p.getLayerBoundTransformChannel(3)
		channel.A = int16((uint16(channel.A) & 0xFF00) | uint16(value))
	case 0x47: // BG3_MATRIX_A_H
		channel := p.getLayerBoundTransformChannel(3)
		channel.A = int16((uint16(channel.A) & 0x00FF) | (uint16(value) << 8))
	case 0x48: // BG3_MATRIX_B_L
		channel := p.getLayerBoundTransformChannel(3)
		channel.B = int16((uint16(channel.B) & 0xFF00) | uint16(value))
	case 0x49: // BG3_MATRIX_B_H
		channel := p.getLayerBoundTransformChannel(3)
		channel.B = int16((uint16(channel.B) & 0x00FF) | (uint16(value) << 8))
	case 0x4A: // BG3_MATRIX_C_L
		channel := p.getLayerBoundTransformChannel(3)
		channel.C = int16((uint16(channel.C) & 0xFF00) | uint16(value))
	case 0x4B: // BG3_MATRIX_C_H
		channel := p.getLayerBoundTransformChannel(3)
		channel.C = int16((uint16(channel.C) & 0x00FF) | (uint16(value) << 8))
	case 0x4C: // BG3_MATRIX_D_L
		channel := p.getLayerBoundTransformChannel(3)
		channel.D = int16((uint16(channel.D) & 0xFF00) | uint16(value))
	case 0x4D: // BG3_MATRIX_D_H
		channel := p.getLayerBoundTransformChannel(3)
		channel.D = int16((uint16(channel.D) & 0x00FF) | (uint16(value) << 8))
	case 0x4E: // BG3_MATRIX_CENTER_X_L
		channel := p.getLayerBoundTransformChannel(3)
		channel.CenterX = int16((uint16(channel.CenterX) & 0xFF00) | uint16(value))
	case 0x4F: // BG3_MATRIX_CENTER_X_H
		channel := p.getLayerBoundTransformChannel(3)
		channel.CenterX = int16((uint16(channel.CenterX) & 0x00FF) | (uint16(value) << 8))
	case 0x50: // BG3_MATRIX_CENTER_Y_L
		channel := p.getLayerBoundTransformChannel(3)
		channel.CenterY = int16((uint16(channel.CenterY) & 0xFF00) | uint16(value))
	case 0x51: // BG3_MATRIX_CENTER_Y_H
		channel := p.getLayerBoundTransformChannel(3)
		channel.CenterY = int16((uint16(channel.CenterY) & 0x00FF) | (uint16(value) << 8))

	// Windowing (0x52-0x5C)
	case 0x52: // WINDOW0_LEFT
		p.Window0.Left = value
	case 0x53: // WINDOW0_RIGHT
		p.Window0.Right = value
	case 0x54: // WINDOW0_TOP
		p.Window0.Top = value
	case 0x55: // WINDOW0_BOTTOM
		p.Window0.Bottom = value
	case 0x56: // WINDOW1_LEFT
		p.Window1.Left = value
	case 0x57: // WINDOW1_RIGHT
		p.Window1.Right = value
	case 0x58: // WINDOW1_TOP
		p.Window1.Top = value
	case 0x59: // WINDOW1_BOTTOM
		p.Window1.Bottom = value
	case 0x5A: // WINDOW_CONTROL
		p.WindowControl = value
	case 0x5B: // WINDOW_MAIN_ENABLE
		p.WindowMainEnable = value
	case 0x5C: // WINDOW_SUB_ENABLE
		p.WindowSubEnable = value

	// HDMA (0x5D-0x5F)
	case 0x5D: // HDMA_CONTROL
		// Bit 0: HDMA enable
		// Bits 1-4: Layer enable (BG0-BG3)
		// Bit 5: Rebind table present
		// Bit 6: Priority table present
		// Bit 7: Tilemap-base table present
		p.HDMAEnabled = (value & 0x01) != 0
		p.HDMAControl = value
	case 0x5E: // HDMA_TABLE_BASE_L
		p.HDMATableBase = (p.HDMATableBase & 0xFF00) | uint16(value)
	case 0x5F: // HDMA_TABLE_BASE_H
		p.HDMATableBase = (p.HDMATableBase & 0x00FF) | (uint16(value) << 8)

	// DMA registers (0x8060-0x8067, but offset is relative to 0x8000, so 0x60-0x67)
	case 0x60: // DMA_CONTROL
		// Bit 0: Enable DMA (1=start transfer, 0=disable)
		// Bit 1: Mode (0=copy, 1=fill)
		// Bits [4:2]: Destination type
		//   0=VRAM, 1=CGRAM, 2=OAM, 3=matrix tilemap, 4=matrix pattern, 5=matrix bitmap
		if (value & 0x01) != 0 {
			// Start DMA transfer
			p.DMAEnabled = true
			p.DMAMode = (value >> 1) & 0x01
			p.DMADestType = (value >> 2) & 0x7
			// Initialize cycle-accurate DMA state
			p.DMAProgress = 0
			p.DMACurrentSrc = p.DMASourceOffset
			p.DMACurrentDest = p.DMADestAddr
			// For fill mode, read fill value once at start
			if p.DMAMode == 1 && p.MemoryReader != nil {
				p.DMAFillValue = p.MemoryReader(p.DMASourceBank, p.DMASourceOffset)
			}
			// Note: DMA will execute incrementally during StepPPU (cycle-accurate)
		} else {
			// Disable DMA (abort current transfer)
			p.DMAEnabled = false
			p.DMAProgress = 0
		}
	case 0x61: // DMA_SOURCE_BANK
		p.DMASourceBank = value
	case 0x62: // DMA_SOURCE_OFFSET_L
		p.DMASourceOffset = (p.DMASourceOffset & 0xFF00) | uint16(value)
	case 0x63: // DMA_SOURCE_OFFSET_H
		p.DMASourceOffset = (p.DMASourceOffset & 0x00FF) | (uint16(value) << 8)
	case 0x64: // DMA_DEST_ADDR_L
		p.DMADestAddr = (p.DMADestAddr & 0xFF00) | uint16(value)
	case 0x65: // DMA_DEST_ADDR_H
		p.DMADestAddr = (p.DMADestAddr & 0x00FF) | (uint16(value) << 8)
	case 0x66: // DMA_LENGTH_L
		p.DMALength = (p.DMALength & 0xFF00) | uint16(value)
	case 0x67: // DMA_LENGTH_H
		p.DMALength = (p.DMALength & 0x00FF) | (uint16(value) << 8)

	// Layer source-mode registers (0x68-0x6B)
	case 0x68: // BG0_SOURCE_MODE
		p.applyLayerSourceModeRegister(0, value)
	case 0x69: // BG1_SOURCE_MODE
		p.applyLayerSourceModeRegister(1, value)
	case 0x6A: // BG2_SOURCE_MODE
		p.applyLayerSourceModeRegister(2, value)
	case 0x6B: // BG3_SOURCE_MODE
		p.applyLayerSourceModeRegister(3, value)
	case 0x6C: // BG0_TRANSFORM_BIND
		p.applyLayerTransformBindRegister(0, value)
	case 0x6D: // BG1_TRANSFORM_BIND
		p.applyLayerTransformBindRegister(1, value)
	case 0x6E: // BG2_TRANSFORM_BIND
		p.applyLayerTransformBindRegister(2, value)
	case 0x6F: // BG3_TRANSFORM_BIND
		p.applyLayerTransformBindRegister(3, value)
	case 0x77: // BG0_TILEMAP_BASE_L
		p.writeLayerTilemapBaseLow(0, value)
	case 0x78: // BG0_TILEMAP_BASE_H
		p.writeLayerTilemapBaseHigh(0, value)
	case 0x79: // BG1_TILEMAP_BASE_L
		p.writeLayerTilemapBaseLow(1, value)
	case 0x7A: // BG1_TILEMAP_BASE_H
		p.writeLayerTilemapBaseHigh(1, value)
	case 0x7B: // BG2_TILEMAP_BASE_L
		p.writeLayerTilemapBaseLow(2, value)
	case 0x7C: // BG2_TILEMAP_BASE_H
		p.writeLayerTilemapBaseHigh(2, value)
	case 0x7D: // BG3_TILEMAP_BASE_L
		p.writeLayerTilemapBaseLow(3, value)
	case 0x7E: // BG3_TILEMAP_BASE_H
		p.writeLayerTilemapBaseHigh(3, value)
	case 0x7F: // HDMA_EXTENSION_CONTROL
		// Bit 0: Source-mode table present
		p.HDMAExtControl = value
	case 0x80: // MATRIX_PLANE_SELECT
		p.MatrixPlaneSelect = value & 0x03
	case 0x81: // MATRIX_PLANE_CONTROL
		// Bit 0: enable, Bits [2:1]: size, Bit 3: source mode (0=tilemap, 1=bitmap), Bits [7:4]: bitmap palette
		p.applySelectedMatrixPlaneControl(value)
	case 0x82: // MATRIX_PLANE_ADDR_L
		p.MatrixPlaneAddr = (p.MatrixPlaneAddr & 0xFF00) | uint16(value)
	case 0x83: // MATRIX_PLANE_ADDR_H
		p.MatrixPlaneAddr = (p.MatrixPlaneAddr & 0x00FF) | (uint16(value&0x7F) << 8)
	case 0x84: // MATRIX_PLANE_DATA
		plane := p.getSelectedMatrixPlane()
		addr := int(p.MatrixPlaneAddr) & (len(plane.Tilemap) - 1)
		plane.Tilemap[addr] = value
		p.MatrixPlaneAddr = uint16((addr + 1) & (len(plane.Tilemap) - 1))
	case 0x85: // MATRIX_PLANE_PATTERN_ADDR_L
		p.writeSelectedMatrixPlanePatternAddrLow(value)
	case 0x86: // MATRIX_PLANE_PATTERN_ADDR_H
		p.writeSelectedMatrixPlanePatternAddrHigh(value)
	case 0x87: // MATRIX_PLANE_PATTERN_DATA
		plane := p.getSelectedMatrixPlane()
		addr := int(p.MatrixPlanePatternAddr) & (len(plane.Pattern) - 1)
		plane.Pattern[addr] = value
		p.MatrixPlanePatternAddr = uint16((addr + 1) & (len(plane.Pattern) - 1))
	case 0x88: // MATRIX_PLANE_BITMAP_ADDR_L
		p.writeSelectedMatrixPlaneBitmapAddrLow(value)
	case 0x89: // MATRIX_PLANE_BITMAP_ADDR_M
		p.writeSelectedMatrixPlaneBitmapAddrMid(value)
	case 0x8A: // MATRIX_PLANE_BITMAP_ADDR_H
		p.writeSelectedMatrixPlaneBitmapAddrHigh(value)
	case 0x8B: // MATRIX_PLANE_BITMAP_DATA
		plane := p.getSelectedMatrixPlane()
		addr := int(p.MatrixPlaneBitmapAddr) & (len(plane.Bitmap) - 1)
		plane.Bitmap[addr] = value
		p.MatrixPlaneBitmapAddr = uint32((addr + 1) & (len(plane.Bitmap) - 1))
	case 0x8C: // MATRIX_PLANE_FLAGS
		// Bit 0: bitmap palette index 0 is transparent
		p.applySelectedMatrixPlaneFlags(value)
	case 0x8D: // MATRIX_PLANE_LIVE_FLOOR_CONTROL
		p.applySelectedMatrixPlaneLiveFloorControl(value)
	case 0x8E: // MATRIX_PLANE_LIVE_FLOOR_HORIZON
		p.writeSelectedMatrixPlaneLiveFloorHorizon(value)
	case 0x8F: // MATRIX_PLANE_LIVE_FLOOR_CAMERA_X_L
		p.writeSelectedMatrixPlaneLiveFloorCameraXLow(value)
	case 0x90: // MATRIX_PLANE_LIVE_FLOOR_CAMERA_X_H
		p.writeSelectedMatrixPlaneLiveFloorCameraXHigh(value)
	case 0x91: // MATRIX_PLANE_LIVE_FLOOR_CAMERA_Y_L
		p.writeSelectedMatrixPlaneLiveFloorCameraYLow(value)
	case 0x92: // MATRIX_PLANE_LIVE_FLOOR_CAMERA_Y_H
		p.writeSelectedMatrixPlaneLiveFloorCameraYHigh(value)
	case 0x93: // MATRIX_PLANE_LIVE_FLOOR_HEADING_X_L
		p.writeSelectedMatrixPlaneLiveFloorHeadingXLow(value)
	case 0x94: // MATRIX_PLANE_LIVE_FLOOR_HEADING_X_H
		p.writeSelectedMatrixPlaneLiveFloorHeadingXHigh(value)
	case 0x95: // MATRIX_PLANE_LIVE_FLOOR_HEADING_Y_L
		p.writeSelectedMatrixPlaneLiveFloorHeadingYLow(value)
	case 0x96: // MATRIX_PLANE_LIVE_FLOOR_HEADING_Y_H
		p.writeSelectedMatrixPlaneLiveFloorHeadingYHigh(value)

	// Text rendering registers (0x70-0x76)
	case 0x70: // TEXT_X_L
		p.TextX = (p.TextX & 0xFF00) | uint16(value)
	case 0x71: // TEXT_X_H
		p.TextX = (p.TextX & 0x00FF) | (uint16(value) << 8)
	case 0x72: // TEXT_Y
		p.TextY = value
	case 0x73: // TEXT_COLOR_R
		p.TextR = value
	case 0x74: // TEXT_COLOR_G
		p.TextG = value
	case 0x75: // TEXT_COLOR_B
		p.TextB = value
	case 0x76: // TEXT_CHAR - buffers character for end-of-frame rendering, advances X by 8
		if p.textCount < len(p.textCmds) {
			p.textCmds[p.textCount] = textCmd{
				x:     int(p.TextX),
				y:     int(p.TextY),
				color: (uint32(p.TextR) << 16) | (uint32(p.TextG) << 8) | uint32(p.TextB),
				char:  rune(value),
			}
			p.textCount++
		}
		p.TextX += 8

	default:
		// Unknown register, ignore
	}

}

// stepDMA executes one cycle of DMA transfer (transfers one byte per cycle)
// This is called from StepPPU to make DMA cycle-accurate
func (p *PPU) stepDMA() {
	if !p.DMAEnabled || p.DMAProgress >= p.DMALength {
		// DMA not active or already complete
		if p.DMAProgress >= p.DMALength {
			// DMA just completed
			p.DMAEnabled = false
			p.DMAProgress = 0
		}
		return
	}

	if p.MemoryReader == nil {
		// No memory reader, abort DMA
		p.DMAEnabled = false
		p.DMAProgress = 0
		return
	}

	// Transfer one byte
	var data uint8
	if p.DMAMode == 1 {
		// Fill mode: use fill value for all bytes
		data = p.DMAFillValue
	} else {
		// Copy mode: read from source
		if p.MemoryReader == nil {
			// No memory reader, abort DMA
			p.DMAEnabled = false
			p.DMAProgress = 0
			return
		}
		data = p.MemoryReader(p.DMASourceBank, p.DMACurrentSrc)
		p.DMACurrentSrc++
	}

	// Write to destination
	switch p.DMADestType {
	case 0: // VRAM
		destAddr := uint32(p.DMACurrentDest)
		if destAddr < 65536 {
			p.VRAM[destAddr] = data
		}
		p.DMACurrentDest++
	case 1: // CGRAM
		// CGRAM is 16-bit (RGB555), so we need to handle it specially
		// For simplicity, write as 8-bit (low byte only)
		addr := p.DMACurrentDest & 0x1FF // Wrap at 512 bytes
		p.CGRAM[addr] = data
		p.invalidateCGRAMCacheByByteAddr(addr)
		p.DMACurrentDest++
	case 2: // OAM
		addr := p.DMACurrentDest & 0x2FF // Wrap at 768 bytes
		p.OAM[addr] = data
		p.DMACurrentDest++
	case 3: // Dedicated matrix-plane tilemap
		plane := p.getSelectedMatrixPlane()
		addr := int(p.MatrixPlaneAddr) & (len(plane.Tilemap) - 1)
		plane.Tilemap[addr] = data
		p.MatrixPlaneAddr = uint16((addr + 1) & (len(plane.Tilemap) - 1))
	case 4: // Dedicated matrix-plane pattern memory
		plane := p.getSelectedMatrixPlane()
		addr := int(p.MatrixPlanePatternAddr) & (len(plane.Pattern) - 1)
		plane.Pattern[addr] = data
		p.MatrixPlanePatternAddr = uint16((addr + 1) & (len(plane.Pattern) - 1))
	case 5: // Dedicated matrix-plane bitmap memory
		plane := p.getSelectedMatrixPlane()
		addr := int(p.MatrixPlaneBitmapAddr) & (len(plane.Bitmap) - 1)
		plane.Bitmap[addr] = data
		p.MatrixPlaneBitmapAddr = uint32((addr + 1) & (len(plane.Bitmap) - 1))
	}

	// Advance progress
	p.DMAProgress++

	// Check if DMA is complete
	if p.DMAProgress >= p.DMALength {
		p.DMAEnabled = false
		p.DMAProgress = 0
	}
}

func (p *PPU) invalidateCGRAMCacheByByteAddr(addr uint16) {
	p.cgramCacheValid[(addr>>1)&0xFF] = false
}

func (p *PPU) updateCGRAMCacheEntry(cgramIndex uint8) {
	p.cgramRGBCache[cgramIndex] = p.decodeCGRAMColor(cgramIndex)
	p.cgramCacheValid[cgramIndex] = true
}

func (p *PPU) decodeCGRAMColor(cgramIndex uint8) uint32 {
	addr := uint16(cgramIndex) * 2
	if addr >= 512 {
		return 0x000000
	}
	low := p.CGRAM[addr]
	high := p.CGRAM[addr+1]

	r := uint32((high & 0x7C) >> 2)
	g := uint32(((high & 0x03) << 3) | ((low & 0xE0) >> 5))
	b := uint32(low & 0x1F)

	r = (r * 255) / 31
	g = (g * 255) / 31
	b = (b * 255) / 31

	return (r << 16) | (g << 8) | b
}

// executeDMA executes a DMA transfer immediately.
// Deprecated: this is a test-only compatibility shim. Clock-driven runtime code
// should step DMA incrementally via StepPPU/stepDMA.
func (p *PPU) executeDMA() {
	if !p.DMAEnabled || p.MemoryReader == nil {
		return
	}

	// Initialize DMA state if not already initialized (check if we need to reset)
	// If DMAProgress is 0 and we haven't started, initialize
	if p.DMAProgress == 0 {
		p.DMACurrentSrc = p.DMASourceOffset
		p.DMACurrentDest = p.DMADestAddr
		// For fill mode, read fill value once at start
		if p.DMAMode == 1 {
			p.DMAFillValue = p.MemoryReader(p.DMASourceBank, p.DMASourceOffset)
		}
	}

	// Execute until DMA disables itself on completion/abort.
	// stepDMA() resets DMAProgress to 0 when complete, so looping on
	// DMAProgress < DMALength can spin forever.
	for p.DMAEnabled {
		p.stepDMA()
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

// RenderFrame renders a complete frame.
// Deprecated: this is a compatibility entry point for older frame-based callers.
// Runtime code should use StepPPU for clock-driven operation.
func (p *PPU) RenderFrame() {
	// DEPRECATED: This is the old frame-based rendering function
	// In clock-driven mode, PPU rendering happens via StepPPU() -> stepDot() -> renderDot()
	// This function should not be called in clock-driven mode

	// Set VBlank flag at start of frame (hardware-accurate synchronization)
	// This signal indicates the start of vertical blanking period
	// ROMs can wait for this signal to synchronize with frame boundaries
	p.VBlankFlag = true

	// Increment frame counter at start of frame (for ROM timing)
	p.FrameCounter++

	// Clear output buffer
	for i := range p.OutputBuffer {
		p.OutputBuffer[i] = 0x000000 // Black
	}

	// Debug: Print CGRAM contents once per 60 frames
	p.debugFrameCount++
	if p.debugFrameCount == 60 && p.Logger != nil {
		// Log CGRAM debug info
		for i := 0; i < 4; i++ {
			addr := i * 2
			low := p.CGRAM[addr]
			high := p.CGRAM[addr+1]
			color := p.getColorFromCGRAM(0, uint8(i))
			r := (color >> 16) & 0xFF
			g := (color >> 8) & 0xFF
			b := color & 0xFF
			p.Logger.LogPPUf(debug.LogLevelDebug,
				"CGRAM palette 0, color %d: CGRAM[%d]=0x%02X, CGRAM[%d]=0x%02X -> RGB(%d,%d,%d) = 0x%06X",
				i, addr, low, addr+1, high, r, g, b, color)
		}
		// Log first 10 pixels
		for i := 0; i < 10; i++ {
			color := p.OutputBuffer[i]
			r := (color >> 16) & 0xFF
			g := (color >> 8) & 0xFF
			b := color & 0xFF
			p.Logger.LogPPUf(debug.LogLevelDebug,
				"Output buffer pixel %d: 0x%06X (RGB %d,%d,%d)",
				i, color, r, g, b)
		}
		// Log BG0 state
		p.Logger.LogPPUf(debug.LogLevelDebug,
			"BG0 state: Enabled=%v, ScrollX=%d, ScrollY=%d",
			p.BG0.Enabled, p.BG0.ScrollX, p.BG0.ScrollY)
		// Log VRAM entries
		p.Logger.LogPPUf(debug.LogLevelDebug,
			"VRAM[0x4000-0x4003] (first tilemap entry): 0x%02X 0x%02X 0x%02X 0x%02X",
			p.VRAM[0x4000], p.VRAM[0x4001], p.VRAM[0x4002], p.VRAM[0x4003])
		p.Logger.LogPPUf(debug.LogLevelDebug,
			"VRAM[0x0000-0x0003] (first tile data): 0x%02X 0x%02X 0x%02X 0x%02X",
			p.VRAM[0x0000], p.VRAM[0x0001], p.VRAM[0x0002], p.VRAM[0x0003])
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
		if channel := p.getTransformChannel(p.BG0.TransformChannel); channel.Enabled {
			p.renderMatrixMode()
		} else {
			p.renderBackgroundLayer(0)
		}
	}

	// Render sprites
	p.renderSprites()

	// Debug: Log sprite 0 OAM data
	if p.debugFrameCount%60 == 0 && p.Logger != nil {
		p.Logger.LogPPUf(debug.LogLevelDebug,
			"Sprite 0 OAM: OAM[0-5]=0x%02X 0x%02X 0x%02X 0x%02X 0x%02X 0x%02X",
			p.OAM[0], p.OAM[1], p.OAM[2], p.OAM[3], p.OAM[4], p.OAM[5])
		spriteX := int(p.OAM[0])
		if (p.OAM[1] & 0x01) != 0 {
			spriteX |= 0xFFFFFF00
		}
		spriteY := int(p.OAM[2])
		tileIndex := uint8(p.OAM[3])
		attributes := uint8(p.OAM[4])
		control := uint8(p.OAM[5])
		paletteIndex := attributes & 0x0F
		enabled := (control & 0x01) != 0
		p.Logger.LogPPUf(debug.LogLevelDebug,
			"Sprite 0: X=%d, Y=%d, Tile=%d, Palette=%d, Enabled=%v",
			spriteX, spriteY, tileIndex, paletteIndex, enabled)
		tileAddr := uint16(tileIndex) * 32
		p.Logger.LogPPUf(debug.LogLevelDebug,
			"Sprite 0 tile data at VRAM[%d]: 0x%02X 0x%02X 0x%02X 0x%02X",
			tileAddr, p.VRAM[tileAddr], p.VRAM[tileAddr+1],
			p.VRAM[tileAddr+2], p.VRAM[tileAddr+3])
		if uint16(paletteIndex)*16+1 < 256 {
			addr := (uint16(paletteIndex)*16 + 1) * 2
			if addr < 512 {
				low := p.CGRAM[addr]
				high := p.CGRAM[addr+1]
				color := p.getColorFromCGRAM(paletteIndex, 1)
				p.Logger.LogPPUf(debug.LogLevelDebug,
					"Sprite 0 CGRAM palette %d, color 1: CGRAM[%d]=0x%02X, CGRAM[%d]=0x%02X -> RGB(0x%06X)",
					paletteIndex, addr, low, addr+1, high, color)
			}
		}
	}
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

	// Tilemap is 32x32, 64x64, or 128x128 tiles depending on layer control.
	tilemapWidth := 32
	tilemapHeight := 32
	switch layer.TilemapSize {
	case TilemapSize64x64:
		tilemapWidth = 64
		tilemapHeight = 64
	case TilemapSize128x128:
		tilemapWidth = 128
		tilemapHeight = 128
	}

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
			tilemapOffset := uint16((tileY*tilemapWidth + tileX) * 2)
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
// NOTE: This is the old frame-based rendering function (deprecated)
// Clock-driven mode uses renderDotMatrixMode() instead
func (p *PPU) renderMatrixMode() {
	// Clock-driven mode uses renderDotMatrixMode() per-pixel
	// This function is kept for compatibility but should not be called in clock-driven mode
	p.renderBackgroundLayer(0)
}

// renderSprites renders all sprites
func (p *PPU) renderSprites() {
	// Render sprites (128 max)
	for spriteIndex := 0; spriteIndex < 128; spriteIndex++ {
		// OAM entry is 6 bytes per sprite
		oamAddr := spriteIndex * 6

		// Read sprite data
		// Byte 0: X position (low byte, unsigned)
		xLow := uint8(p.OAM[oamAddr])
		// Byte 1: X position (high byte, bit 0 only, sign extends)
		xHigh := uint8(p.OAM[oamAddr+1])
		// Combine X position: 9-bit signed value
		// Low 8 bits from byte 0, sign bit from bit 0 of byte 1
		spriteX := int(xLow)
		if (xHigh & 0x01) != 0 {
			// Sign extend (negative value)
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

		// Log sprite 0 rendering state (for debugging blinking)
		if spriteIndex == 0 && p.Logger != nil && p.currentScanline == 0 && p.currentDot == 0 {
			p.Logger.LogPPUf(debug.LogLevelDebug,
				"SPRITE0_RENDER: Enabled=%v X=%d Y=%d Tile=%d Palette=%d Control=0x%02X",
				enabled, spriteX, spriteY, tileIndex, paletteIndex, control)
		}

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
	if p.Window0.Right > 0 || p.Window0.Left > 0 {
		// Window is configured, check bounds
		win0Inside = x >= int(p.Window0.Left) && x <= int(p.Window0.Right) &&
			y >= int(p.Window0.Top) && y <= int(p.Window0.Bottom)
	}

	win1Inside := true // Default to inside if window not configured
	if p.Window1.Right > 0 || p.Window1.Left > 0 {
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
	fullIndex := uint16(paletteIndex)*16 + uint16(colorIndex)
	if fullIndex >= 256 {
		return 0x000000
	}
	cgramIndex := uint8(fullIndex)
	if !p.cgramCacheValid[cgramIndex] {
		p.updateCGRAMCacheEntry(cgramIndex)
	}
	return p.cgramRGBCache[cgramIndex]
}
