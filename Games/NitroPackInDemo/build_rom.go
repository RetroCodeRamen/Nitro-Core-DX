//go:build testrom_tools
// +build testrom_tools

package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"

	"nitro-core-dx/internal/emulator"
	ppucore "nitro-core-dx/internal/ppu"
	"nitro-core-dx/test/roms/romutil"
)

type asm struct{ *romutil.Asm }

func newASM(bank uint8) *asm { return &asm{Asm: romutil.NewASM(bank)} }

func (a *asm) pc() uint16                  { return a.PC() }
func (a *asm) mark(name string)            { a.Mark(name) }
func (a *asm) uniq(prefix string) string   { return a.Uniq(prefix) }
func (a *asm) movImm(reg uint8, v uint16)  { a.MovImm(reg, v) }
func (a *asm) movReg(dst, src uint8)       { a.MovReg(dst, src) }
func (a *asm) movLoad(dst, addrReg uint8)  { a.MovLoad(dst, addrReg) }
func (a *asm) movStore(addrReg, src uint8) { a.MovStore(addrReg, src) }
func (a *asm) setDBR(src uint8)            { a.SetDBR(src) }
func (a *asm) addImm(reg uint8, v uint16)  { a.AddImm(reg, v) }
func (a *asm) addReg(dst, src uint8)       { a.AddReg(dst, src) }
func (a *asm) subImm(reg uint8, v uint16)  { a.SubImm(reg, v) }
func (a *asm) subReg(dst, src uint8)       { a.SubReg(dst, src) }
func (a *asm) andImm(reg uint8, v uint16)  { a.AndImm(reg, v) }
func (a *asm) cmpImm(reg uint8, v uint16)  { a.CmpImm(reg, v) }
func (a *asm) cmpReg(r1, r2 uint8)         { a.CmpReg(r1, r2) }
func (a *asm) resolve() error              { return a.Resolve() }
func (a *asm) beq(label string)            { a.Beq(label) }
func (a *asm) bne(label string)            { a.Bne(label) }
func (a *asm) bgt(label string)            { a.Bgt(label) }
func (a *asm) blt(label string)            { a.Blt(label) }
func (a *asm) bge(label string)            { a.Bge(label) }
func (a *asm) ble(label string)            { a.Ble(label) }
func (a *asm) jmp(label string)            { a.Jmp(label) }

var (
	allocateROMData = romutil.AllocateROMData
	appendDataBlob  = romutil.AppendDataBlob
)

type billboardPlane struct {
	plane             uint8
	bgControlAddr     uint16
	bgControlValue    uint8
	transformBindAddr uint16
	matrixControlAddr uint16
	planeControl      uint8
	flags             uint8
	horizon           uint8
	baseDistance      uint16
	focalLength       uint16
	widthScale        uint16
	originX           int16
	originY           int16
	facingX           int16
	facingY           int16
	heightScale       uint16
}

func write8(a *asm, addr uint16, value uint8)   { romutil.Write8(a.Asm, addr, value) }
func write16(a *asm, addr uint16, value uint16) { romutil.Write16(a.Asm, addr, value) }
func write16s(a *asm, addr uint16, value int16) { romutil.Write16S(a.Asm, addr, value) }
func write16RegBytes(a *asm, addr uint16, reg uint8) {
	romutil.Write16RegBytes(a.Asm, addr, reg)
}
func (a *asm) sarImm(reg uint8, v uint16) { a.SarImm(reg, v) }
func write8Scratch(a *asm, addr uint16, value uint8, addrReg, valueReg uint8) {
	romutil.Write8Scratch(a.Asm, addr, value, addrReg, valueReg)
}
func emitText(a *asm, x uint16, y uint8, r, g, b uint8, text string) {
	romutil.EmitText(a.Asm, x, y, r, g, b, text)
}
func emitTextCentered(a *asm, y uint8, r, g, b uint8, text string) {
	emitText(a, uint16((320-len(text)*8)/2), y, r, g, b, text)
}
func setCGRAMColor(a *asm, colorIndex uint8, rgb555 uint16) {
	romutil.SetCGRAMColor(a.Asm, colorIndex, rgb555)
}
func emitWaitOneFrame(a *asm, wramLastFrame uint16) {
	romutil.EmitWaitOneFrame(a.Asm, wramLastFrame)
}
func emitMatrixBitmapDMAChunks(a *asm, channel uint8, ref romutil.DataRef) {
	romutil.EmitMatrixBitmapDMAChunks(a.Asm, channel, ref)
}

func loadWRAM(a *asm, dst uint8, addr uint16) {
	addrReg := uint8(7)
	if dst == addrReg {
		addrReg = 4
	}
	a.movImm(addrReg, addr)
	a.movLoad(dst, addrReg)
}

func storeWRAM(a *asm, addr uint16, src uint8) {
	addrReg := uint8(7)
	if src == addrReg {
		addrReg = 4
	}
	a.movImm(addrReg, addr)
	a.movStore(addrReg, src)
}

func emitInitHeadingTable(a *asm, tableBase uint16, steps int, moveSpeed float64) {
	for i := 0; i < steps; i++ {
		angle := (2.0 * math.Pi * float64(i)) / float64(steps)
		cosv := int16(math.Round(math.Cos(angle) * 256.0))
		sinv := int16(math.Round(math.Sin(angle) * 256.0))
		moveX := int16(math.Round(math.Cos(angle) * moveSpeed))
		moveY := int16(math.Round(math.Sin(angle) * moveSpeed))
		write16s(a, tableBase+uint16(i*8), cosv)
		write16s(a, tableBase+uint16(i*8)+2, sinv)
		write16s(a, tableBase+uint16(i*8)+4, moveX)
		write16s(a, tableBase+uint16(i*8)+6, moveY)
	}
}

func emitLoadHeadingEntry(a *asm, tableBase uint16, indexReg, headingXReg, headingYReg, moveXReg, moveYReg uint8) {
	a.movReg(4, indexReg)
	a.addReg(4, 4)
	a.addReg(4, 4)
	a.addReg(4, 4)
	a.addImm(4, tableBase)
	a.movLoad(headingXReg, 4)
	a.addImm(4, 2)
	a.movLoad(headingYReg, 4)
	a.addImm(4, 2)
	a.movLoad(moveXReg, 4)
	a.addImm(4, 2)
	a.movLoad(moveYReg, 4)
}

func setVRAMAddr(a *asm, addr uint16) {
	write8(a, 0x800E, uint8(addr&0xFF))
	write8(a, 0x800F, uint8((addr>>8)&0xFF))
}

func writeVRAMBlock(a *asm, addr uint16, data []uint8) {
	setVRAMAddr(a, addr)
	a.movImm(4, 0x8010)
	for _, b := range data {
		a.movImm(5, uint16(b))
		a.movStore(4, 5)
	}
}

func makePackedTile(size int, pixel func(x, y int) uint8) []uint8 {
	data := make([]uint8, size*size/2)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x += 2 {
			hi := pixel(x, y) & 0x0F
			lo := pixel(x+1, y) & 0x0F
			data[(y*size+x)/2] = (hi << 4) | lo
		}
	}
	return data
}

func buildPlayerTile() []uint8 {
	return makePackedTile(16, func(x, y int) uint8 {
		switch {
		case y <= 2 && x >= 5 && x <= 10:
			return 1
		case y >= 3 && y <= 11 && x >= 4 && x <= 11:
			if x == 4 || x == 11 || y == 3 || y == 11 {
				return 3
			}
			return 2
		case y >= 12 && y <= 15 && (x == 5 || x == 10):
			return 3
		default:
			return 0
		}
	})
}

func writeSpriteImm(a *asm, spriteID uint8, x uint16, yReg uint8, tile uint8, attr uint8, ctrl uint8) {
	write8(a, 0x8014, spriteID)
	write8(a, 0x8015, uint8(x&0xFF))
	write8(a, 0x8015, uint8((x>>8)&0x01))
	a.movImm(4, 0x8015)
	a.movStore(4, yReg)
	write8(a, 0x8015, tile)
	write8(a, 0x8015, attr)
	write8(a, 0x8015, ctrl)
}

func clearSprite(a *asm, spriteID uint8) {
	write8(a, 0x8014, spriteID)
	write8(a, 0x8015, 0)
	write8(a, 0x8015, 0)
	write8(a, 0x8015, 0)
	write8(a, 0x8015, 0)
	write8(a, 0x8015, 0)
	write8(a, 0x8015, 0)
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > img.Bounds().Dx() {
		x1 = img.Bounds().Dx()
	}
	if y1 > img.Bounds().Dy() {
		y1 = img.Bounds().Dy()
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func fillCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	r2 := r * r
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r2 && image.Pt(x, y).In(img.Bounds()) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func buildTreePlaceholderImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	fillRect(img, 29, 34, 35, 64, color.RGBA{R: 112, G: 70, B: 32, A: 255})
	fillRect(img, 27, 40, 37, 64, color.RGBA{R: 92, G: 52, B: 24, A: 255})
	fillCircle(img, 32, 20, 14, color.RGBA{R: 44, G: 144, B: 60, A: 255})
	fillCircle(img, 22, 28, 12, color.RGBA{R: 36, G: 120, B: 52, A: 255})
	fillCircle(img, 42, 28, 12, color.RGBA{R: 40, G: 128, B: 56, A: 255})
	fillCircle(img, 32, 32, 16, color.RGBA{R: 56, G: 164, B: 68, A: 255})
	fillCircle(img, 28, 18, 4, color.RGBA{R: 112, G: 208, B: 112, A: 255})
	fillCircle(img, 38, 24, 4, color.RGBA{R: 104, G: 196, B: 104, A: 255})
	return img
}

func buildAnnexPlaceholderImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	fillRect(img, 10, 12, 54, 58, color.RGBA{R: 110, G: 128, B: 146, A: 255})
	fillRect(img, 8, 10, 56, 14, color.RGBA{R: 86, G: 102, B: 120, A: 255})
	fillRect(img, 26, 34, 38, 58, color.RGBA{R: 72, G: 52, B: 36, A: 255})
	for row := 0; row < 2; row++ {
		for col := 0; col < 3; col++ {
			x := 14 + col*14
			y := 18 + row*12
			fillRect(img, x, y, x+8, y+6, color.RGBA{R: 154, G: 216, B: 244, A: 255})
		}
	}
	fillRect(img, 12, 58, 52, 62, color.RGBA{R: 42, G: 42, B: 42, A: 255})
	return img
}

// buildInteriorFloorImage draws the interior room floor for matrix plane 2.
// The 256x256 image maps onto the full 1024-unit plane world (4 world units
// per pixel): the walkable room is a checkered tile floor with a wall band,
// surrounded by dark void so the room edges read clearly in perspective.
func buildInteriorFloorImage(roomMinX, roomMinY, roomMaxX, roomMaxY int) image.Image {
	const worldPerPixel = 4
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))

	void := color.RGBA{R: 12, G: 10, B: 16, A: 255}
	wall := color.RGBA{R: 70, G: 78, B: 96, A: 255}
	tileA := color.RGBA{R: 150, G: 110, B: 70, A: 255}
	tileB := color.RGBA{R: 122, G: 86, B: 52, A: 255}
	rug := color.RGBA{R: 160, G: 48, B: 48, A: 255}

	fillRect(img, 0, 0, 256, 256, void)

	px0 := roomMinX / worldPerPixel
	py0 := roomMinY / worldPerPixel
	px1 := roomMaxX/worldPerPixel + 1
	py1 := roomMaxY/worldPerPixel + 1

	fillRect(img, px0-3, py0-3, px1+3, py1+3, wall)
	for y := py0; y < py1; y++ {
		for x := px0; x < px1; x++ {
			c := tileA
			if ((x/6)+(y/6))%2 == 1 {
				c = tileB
			}
			img.SetRGBA(x, y, c)
		}
	}

	// Rug in front of the NPC at the far (low-Y) end of the room.
	rugCX := (px0 + px1) / 2
	fillRect(img, rugCX-8, py0+6, rugCX+8, py0+16, rug)

	return img
}

// buildNPCImage draws the placeholder NPC character used as the interior
// vertical billboard (plane 3): bottom-anchored so the feet sit on the floor.
func buildNPCImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))

	skin := color.RGBA{R: 232, G: 188, B: 152, A: 255}
	hair := color.RGBA{R: 86, G: 56, B: 30, A: 255}
	tunic := color.RGBA{R: 56, G: 120, B: 190, A: 255}
	belt := color.RGBA{R: 40, G: 36, B: 32, A: 255}
	legs := color.RGBA{R: 62, G: 58, B: 70, A: 255}
	eye := color.RGBA{R: 24, G: 24, B: 30, A: 255}

	fillRect(img, 25, 46, 31, 62, legs)
	fillRect(img, 33, 46, 39, 62, legs)
	fillRect(img, 23, 60, 32, 63, belt)
	fillRect(img, 32, 60, 41, 63, belt)
	fillRect(img, 20, 26, 44, 47, tunic)
	fillRect(img, 20, 42, 44, 46, belt)
	fillRect(img, 15, 28, 20, 42, skin)
	fillRect(img, 44, 28, 49, 42, skin)
	fillCircle(img, 32, 16, 9, skin)
	fillRect(img, 23, 5, 42, 12, hair)
	fillRect(img, 22, 8, 26, 16, hair)
	fillRect(img, 38, 8, 43, 16, hair)
	fillRect(img, 28, 15, 30, 18, eye)
	fillRect(img, 35, 15, 37, 18, eye)

	return img
}

func normalizeBillboardImage(img image.Image) image.Image {
	if img == nil {
		return nil
	}

	bounds := img.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X-1, bounds.Min.Y-1
	foundOpaque := false

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if uint8(a>>8) < 128 {
				continue
			}
			if !foundOpaque {
				minX, minY = x, y
				maxX, maxY = x, y
				foundOpaque = true
				continue
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}

	if !foundOpaque {
		return img
	}

	cropW := maxX - minX + 1
	cropH := maxY - minY + 1
	side := cropW
	if cropH > side {
		side = cropH
	}

	dst := image.NewRGBA(image.Rect(0, 0, side, side))
	offsetX := (side - cropW) / 2
	offsetY := side - cropH

	for y := 0; y < cropH; y++ {
		for x := 0; x < cropW; x++ {
			dst.Set(offsetX+x, offsetY+y, img.At(minX+x, minY+y))
		}
	}

	return dst
}

func matrixPlaneBitmapControl64(paletteBank uint8) uint8 {
	return 0x01 | 0x02 | 0x08 | (paletteBank << 4)
}

func emitUploadPlaneBitmap(a *asm, plane uint8, ctl uint8, flags uint8, ref romutil.DataRef) {
	write8(a, 0x8080, plane)
	write8(a, 0x8081, ctl)
	write8(a, 0x808C, flags)
	write8(a, 0x8088, 0x00)
	write8(a, 0x8089, 0x00)
	write8(a, 0x808A, 0x00)
	emitMatrixBitmapDMAChunks(a, plane, ref)
}

func emitConfigureVerticalBillboardPlane(a *asm, cfg billboardPlane) {
	write8(a, cfg.transformBindAddr, cfg.plane)
	write8(a, cfg.bgControlAddr, cfg.bgControlValue)
	write8(a, cfg.matrixControlAddr, 0x01)
	write8(a, 0x8080, cfg.plane)
	write8(a, 0x8081, cfg.planeControl)
	write8(a, 0x808C, cfg.flags)
	write8(a, 0x8091, ppucore.MatrixPlaneProjectionVertical)
	write8(a, 0x8092, cfg.horizon)
	write16(a, 0x809B, cfg.baseDistance)
	write16(a, 0x809D, cfg.focalLength)
	write16(a, 0x809F, cfg.widthScale)
	write16s(a, 0x80A1, cfg.originX)
	write16s(a, 0x80A3, cfg.originY)
	write16s(a, 0x80A5, cfg.facingX)
	write16s(a, 0x80A7, cfg.facingY)
	write16(a, 0x80A9, cfg.heightScale)
}

func emitSyncPlaneCameraHeading(a *asm, plane uint8, cameraXReg, cameraYReg, headingXReg, headingYReg uint8) {
	write8Scratch(a, 0x8080, plane, 7, 5)
	write16RegBytes(a, 0x8093, cameraXReg)
	write16RegBytes(a, 0x8095, cameraYReg)
	write16RegBytes(a, 0x8097, headingXReg)
	write16RegBytes(a, 0x8099, headingYReg)
}

func emitDisableAllSceneLayers(a *asm) {
	write8(a, 0x8008, 0x00)
	write8(a, 0x8009, 0x00)
	write8(a, 0x8021, 0x00)
	write8(a, 0x8026, 0x00)
	write8(a, 0x8018, 0x00)
	write8(a, 0x802B, 0x00)
	write8(a, 0x8038, 0x00)
	write8(a, 0x8045, 0x00)
	write8(a, 0x8011, 0x01)
}

func emitEnableOverworldLayers(a *asm, objects []billboardPlane) {
	write8(a, 0x8008, 0x21)
	write8(a, 0x8018, 0x09)
	write8(a, 0x8021, 0x00)
	write8(a, 0x8026, 0x00)
	for _, obj := range objects {
		write8(a, obj.bgControlAddr, obj.bgControlValue)
		write8(a, obj.transformBindAddr, obj.plane)
		write8(a, obj.matrixControlAddr, 0x01)
	}
	write8(a, 0x8011, 0x01)
}

// emitEnableInteriorLayers swaps the scene to the interior set: BG2 carries
// the interior floor (plane 2), BG3 carries the NPC billboard (plane 3), and
// the overworld layers (BG0/BG1) are switched off.
func emitEnableInteriorLayers(a *asm) {
	write8(a, 0x8008, 0x00)
	write8(a, 0x8009, 0x00)
	write8(a, 0x8021, 0x21)
	write8(a, 0x806E, 0x02)
	write8(a, 0x8038, 0x01)
	write8(a, 0x8026, 0x15)
	write8(a, 0x806F, 0x03)
	write8(a, 0x8045, 0x01)
	write8(a, 0x8011, 0x01)
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func buildNitroPackInDemoROM(floorImg, billboardImg image.Image, outPath string) error {
	const (
		codeBank      = 1
		dataStartBank = 2

		sceneTitle     = 0
		sceneOverworld = 1
		sceneInterior  = 2
		scenePause     = 3
		sceneDialogue  = 4
		sceneCredits   = 5

		wramLastFrame    = 0x0200
		wramScene        = 0x0202
		wramSceneReturn  = 0x0204
		wramStartHeld    = 0x0206
		wramActionHeld   = 0x0208
		wramHeadingIndex = 0x020A
		wramCameraX      = 0x020C
		wramCameraY      = 0x020E
		wramTurnTick     = 0x0210
		wramJumpTimer    = 0x0212
		wramInputLow     = 0x0214
		wramInputHigh    = 0x0216
		wramIntX         = 0x0218
		wramIntY         = 0x021A
		wramIntHeading   = 0x021C
		wramDialogPage   = 0x021E
		wramDialogReveal = 0x0220

		headingTableBase = 0x0300
		headingSteps     = 64

		// Dialogue text lives in WRAM as one 16-bit word per character so the
		// per-frame typewriter loop can stream revealed characters to the text
		// port with plain indexed loads.
		dialogPage0Base = 0x0500
		dialogPage1Base = 0x0560
		dialogPageCount = 2

		playerSpriteTile = 8
		playerSpriteX    = 152
		playerSpriteCtrl = 0x03
		// Priority bits [7:6] = 0b01 = priority 1. BG1 (building billboard) also has
		// priority 1, but same-priority BGs render before sprites, so the player sprite
		// renders on top of the building when they overlap.
		playerSpriteAttr = 0x45

		floorPlaneCtl        = 0x1D
		floorPlaneFlags      = 0x00
		interiorFloorCtl     = 0x3D // floorPlaneCtl pattern with palette bank 3
		interiorFloorFlags   = 0x00
		overworldHorizon     = 113
		overworldBaseDist    = 0x0C00
		overworldFocalLength = 0xC000

		worldMinX = 0
		worldMaxX = 1023
		worldMinY = 0
		worldMaxY = 1023

		buildingAnchorX       = 512
		buildingAnchorY       = 600
		buildingDoorHalfWidth = 18
		buildingDoorFrontY    = 96

		doorMinX = buildingAnchorX - buildingDoorHalfWidth
		doorMaxX = buildingAnchorX + buildingDoorHalfWidth
		doorMaxY = buildingAnchorY + buildingDoorFrontY

		// Collision footprint for the building facade. The player is stopped at
		// buildingFrontY when their X is within the building's half-width.
		buildingCollisionHalfWidth = 40
		buildingCollisionMinX      = buildingAnchorX - buildingCollisionHalfWidth // 472
		buildingCollisionMaxX      = buildingAnchorX + buildingCollisionHalfWidth // 552
		buildingFrontY             = buildingAnchorY                              // 600

		// Interior room. The player enters at the south (high-Y) end facing
		// north toward the NPC, and the door back out is the entry zone.
		interiorMinX         = 416
		interiorMaxX         = 608
		interiorMinY         = 432
		interiorMaxY         = 632
		interiorEntryX       = 512
		interiorEntryY       = 616
		interiorEntryHeading = 48 // heading table index: facing -Y (north)

		npcAnchorX   = 512
		npcAnchorY   = 472
		npcStopY     = npcAnchorY + 28 // collision: player stops in front of the NPC
		npcBlockMinX = npcAnchorX - 24
		npcBlockMaxX = npcAnchorX + 24
		npcTalkMinX  = npcAnchorX - 28
		npcTalkMaxX  = npcAnchorX + 28
		npcTalkMaxY  = npcAnchorY + 96

		exitZoneMinX = interiorEntryX - 40
		exitZoneMaxX = interiorEntryX + 40
		exitZoneMinY = 608
	)

	dialogPages := []struct {
		base uint16
		text string
	}{
		{dialogPage0Base, "WELCOME TO THE NITRO SHOWCASE"},
		{dialogPage1Base, "THANKS FOR TRYING THE DEMO"},
	}

	if floorImg == nil {
		return fmt.Errorf("floor image is required")
	}
	if billboardImg == nil {
		return fmt.Errorf("billboard image is required")
	}

	floorAsset, err := emulator.BuildBitmapMatrixPlaneAssetFromImage(floorImg, 0, ppucore.TilemapSize128x128, 1)
	if err != nil {
		return err
	}
	normalizedBillboardImg := normalizeBillboardImage(billboardImg)
	mainBuildingAsset, err := emulator.BuildBitmapMatrixPlaneAssetFromImage(normalizedBillboardImg, 1, ppucore.TilemapSize64x64, 2)
	if err != nil {
		return err
	}
	interiorFloorImg := buildInteriorFloorImage(interiorMinX, interiorMinY, interiorMaxX, interiorMaxY)
	interiorFloorAsset, err := emulator.BuildBitmapMatrixPlaneAssetFromImage(interiorFloorImg, 2, ppucore.TilemapSize128x128, 3)
	if err != nil {
		return err
	}
	npcImg := normalizeBillboardImage(buildNPCImage())
	npcAsset, err := emulator.BuildBitmapMatrixPlaneAssetFromImage(npcImg, 3, ppucore.TilemapSize64x64, 4)
	if err != nil {
		return err
	}

	floorRef, cursor := allocateROMData(0, floorAsset.Program.Bitmap)
	mainBuildingRef, cursor := allocateROMData(cursor, mainBuildingAsset.Program.Bitmap)
	interiorFloorRef, cursor := allocateROMData(cursor, interiorFloorAsset.Program.Bitmap)
	npcRef, cursor := allocateROMData(cursor, npcAsset.Program.Bitmap)
	_ = cursor

	objects := []billboardPlane{
		{
			plane:             1,
			bgControlAddr:     0x8009,
			bgControlValue:    0x15,
			transformBindAddr: 0x806D,
			matrixControlAddr: 0x802B,
			planeControl:      matrixPlaneBitmapControl64(2),
			flags:             0x03,
			horizon:           overworldHorizon,
			baseDistance:      overworldBaseDist,
			focalLength:       overworldFocalLength,
			widthScale:        0x0070,
			originX:           buildingAnchorX,
			originY:           buildingAnchorY,
			facingX:           0,
			facingY:           0x0100,
			heightScale:       0x4000,
		},
	}

	// The interior NPC shares the building's projection model so the room and
	// its occupant track the same camera, just person-sized.
	npcObject := billboardPlane{
		plane:             3,
		bgControlAddr:     0x8026,
		bgControlValue:    0x15,
		transformBindAddr: 0x806F,
		matrixControlAddr: 0x8045,
		planeControl:      matrixPlaneBitmapControl64(4),
		flags:             0x03,
		horizon:           overworldHorizon,
		baseDistance:      overworldBaseDist,
		focalLength:       overworldFocalLength,
		widthScale:        0x0024,
		originX:           npcAnchorX,
		originY:           npcAnchorY,
		facingX:           0,
		facingY:           0x0100,
		heightScale:       0x1C00,
	}

	a := newASM(codeBank)

	for i, c := range floorAsset.Palette {
		setCGRAMColor(a, uint8(1*16+i), c)
	}
	for i, c := range mainBuildingAsset.Palette {
		setCGRAMColor(a, uint8(2*16+i), c)
	}
	for i, c := range interiorFloorAsset.Palette {
		setCGRAMColor(a, uint8(3*16+i), c)
	}
	for i, c := range npcAsset.Palette {
		setCGRAMColor(a, uint8(4*16+i), c)
	}
	setCGRAMColor(a, 0x00, 0x0000)
	setCGRAMColor(a, 0x51, 0x7FFF)
	setCGRAMColor(a, 0x52, 0x03FF)
	setCGRAMColor(a, 0x53, 0x7FE0)

	writeVRAMBlock(a, uint16(playerSpriteTile)*128, buildPlayerTile())

	write16(a, wramScene, sceneTitle)
	write16(a, wramSceneReturn, sceneOverworld)
	write16(a, wramStartHeld, 0)
	write16(a, wramActionHeld, 0)
	write16(a, wramHeadingIndex, 48)
	write16(a, wramCameraX, 512)
	write16(a, wramCameraY, 768)
	write16(a, wramTurnTick, 0)
	write16(a, wramJumpTimer, 0)
	write16(a, wramInputLow, 0)
	write16(a, wramInputHigh, 0)
	write16(a, wramIntX, interiorEntryX)
	write16(a, wramIntY, interiorEntryY)
	write16(a, wramIntHeading, interiorEntryHeading)
	write16(a, wramDialogPage, 0)
	write16(a, wramDialogReveal, 0)
	emitInitHeadingTable(a, headingTableBase, headingSteps, 3.6)
	for _, page := range dialogPages {
		for i, ch := range page.text {
			write16(a, page.base+uint16(i*2), uint16(ch))
		}
	}

	write8(a, 0x8008, 0x21)
	write8(a, 0x806C, 0x00)
	emitUploadPlaneBitmap(a, 0x00, floorPlaneCtl, floorPlaneFlags, floorRef)
	emitUploadPlaneBitmap(a, 0x01, objects[0].planeControl, objects[0].flags, mainBuildingRef)
	emitUploadPlaneBitmap(a, 0x02, interiorFloorCtl, interiorFloorFlags, interiorFloorRef)
	emitUploadPlaneBitmap(a, 0x03, npcObject.planeControl, npcObject.flags, npcRef)

	a.movImm(0, 0x0000)
	a.setDBR(0)
	a.movImm(4, 0x803F)
	a.movLoad(2, 4)
	a.movImm(4, wramLastFrame)
	a.movStore(4, 2)

	write8(a, 0x8080, 0x00)
	write8(a, 0x8081, floorPlaneCtl)
	write8(a, 0x808C, floorPlaneFlags)
	write8(a, 0x8091, 0x01)
	write8(a, 0x8092, overworldHorizon)
	write16(a, 0x809B, overworldBaseDist)
	write16(a, 0x809D, overworldFocalLength)
	write16(a, 0x809F, 0x00C0)
	a.movImm(0, 48)
	emitLoadHeadingEntry(a, headingTableBase, 0, 1, 2, 3, 6)
	a.movImm(3, 512)
	a.movImm(6, 768)
	emitSyncPlaneCameraHeading(a, 0x00, 3, 6, 1, 2)

	for _, obj := range objects {
		emitConfigureVerticalBillboardPlane(a, obj)
	}
	for _, obj := range objects {
		emitSyncPlaneCameraHeading(a, obj.plane, 3, 6, 1, 2)
	}

	// Interior planes are projected at boot too; their layers stay disabled
	// until the interior scene enables them.
	write8(a, 0x8080, 0x02)
	write8(a, 0x8081, interiorFloorCtl)
	write8(a, 0x808C, interiorFloorFlags)
	write8(a, 0x8091, 0x01)
	write8(a, 0x8092, overworldHorizon)
	write16(a, 0x809B, overworldBaseDist)
	write16(a, 0x809D, overworldFocalLength)
	write16(a, 0x809F, 0x00C0)
	emitConfigureVerticalBillboardPlane(a, npcObject)

	write8(a, 0x806C, 0x00)
	for _, obj := range objects {
		write8(a, obj.transformBindAddr, obj.plane)
	}
	write8(a, 0x8011, 0x01)

	clearSprite(a, 0)
	clearSprite(a, 1)

	sceneTitleLabel := a.uniq("scene_title")
	sceneOverworldLabel := a.uniq("scene_overworld")
	sceneInteriorLabel := a.uniq("scene_interior")
	scenePauseLabel := a.uniq("scene_pause")
	sceneDialogueLabel := a.uniq("scene_dialogue")
	sceneCreditsLabel := a.uniq("scene_credits")

	titlePressed := a.uniq("title_pressed")
	titleDone := a.uniq("title_done")
	pausePressed := a.uniq("pause_pressed")
	pauseDone := a.uniq("pause_done")
	interiorPausePressed := a.uniq("interior_pause_pressed")
	interiorPauseDone := a.uniq("interior_pause_done")

	noTurnLeft := a.uniq("no_turn_left")
	afterTurnLeft := a.uniq("after_turn_left")
	lookLeftWrap := a.uniq("look_left_wrap")
	noTurnRight := a.uniq("no_turn_right")
	afterTurnRight := a.uniq("after_turn_right")
	lookRightWrap := a.uniq("look_right_wrap")
	noMoveForward := a.uniq("no_move_forward")
	noMoveBackward := a.uniq("no_move_backward")
	clampXMinDone := a.uniq("clamp_x_min_done")
	clampXMaxDone := a.uniq("clamp_x_max_done")
	clampYMinDone := a.uniq("clamp_y_min_done")
	clampYMaxDone := a.uniq("clamp_y_max_done")
	buildingCollisionSkip := a.uniq("building_collision_skip")
	doorInteractOutOfRange := a.uniq("door_interact_out_of_range")
	doorPromptOutOfRange := a.uniq("door_prompt_out_of_range")
	doorPromptDone := a.uniq("door_prompt_done")
	doorActionPressed := a.uniq("door_action_pressed")
	doorActionEdgeDone := a.uniq("door_action_edge_done")
	outsideActionPressed := a.uniq("outside_action_pressed")
	outsideActionDone := a.uniq("outside_action_done")

	a.mark("main_loop")
	a.movImm(0, 0x0000)
	a.setDBR(0)
	emitWaitOneFrame(a, wramLastFrame)

	write8(a, 0xA001, 0x01)
	a.movImm(4, 0xA000)
	a.movLoad(2, 4)
	a.movImm(4, 0xA001)
	a.movLoad(3, 4)
	write8(a, 0xA001, 0x00)
	storeWRAM(a, wramInputLow, 2)
	storeWRAM(a, wramInputHigh, 3)

	loadWRAM(a, 0, wramScene)
	a.cmpImm(0, sceneTitle)
	a.beq(sceneTitleLabel)
	a.cmpImm(0, sceneInterior)
	a.beq(sceneInteriorLabel)
	a.cmpImm(0, scenePause)
	a.beq(scenePauseLabel)
	a.cmpImm(0, sceneDialogue)
	a.beq(sceneDialogueLabel)
	a.cmpImm(0, sceneCredits)
	a.beq(sceneCreditsLabel)
	a.jmp(sceneOverworldLabel)

	a.mark(sceneTitleLabel)
	emitDisableAllSceneLayers(a)
	clearSprite(a, 0)
	clearSprite(a, 1)
	emitTextCentered(a, 42, 0xFF, 0xFF, 0xFF, "NITRO PACK-IN DEMO")
	emitTextCentered(a, 66, 0x90, 0xD8, 0xFF, "OVERWORLD  INTERIOR  DIALOGUE  CREDITS")
	emitTextCentered(a, 98, 0xFF, 0xFF, 0xFF, "PRESS START")
	emitTextCentered(a, 130, 0xC0, 0xE0, 0x90, "WALK TO THE BUILDING AND PRESS A")
	emitTextCentered(a, 154, 0xFF, 0xC0, 0x70, "TALK TO THE GUIDE INSIDE")
	a.movReg(6, 3)
	a.andImm(6, 0x0004)
	a.cmpImm(6, 0)
	a.bne(titlePressed)
	a.movImm(7, 0)
	storeWRAM(a, wramStartHeld, 7)
	a.jmp(titleDone)
	a.mark(titlePressed)
	loadWRAM(a, 7, wramStartHeld)
	a.cmpImm(7, 0)
	a.bne(titleDone)
	a.movImm(7, 1)
	storeWRAM(a, wramStartHeld, 7)
	a.movImm(7, sceneOverworld)
	storeWRAM(a, wramScene, 7)
	storeWRAM(a, wramSceneReturn, 7)
	a.mark(titleDone)
	a.jmp("main_loop")

	a.mark(scenePauseLabel)
	emitDisableAllSceneLayers(a)
	clearSprite(a, 0)
	clearSprite(a, 1)
	emitText(a, 140, 88, 0xFF, 0xFF, 0xFF, "PAUSE")
	emitText(a, 64, 120, 0xA0, 0xD8, 0xFF, "PRESS START TO RESUME")
	a.movReg(6, 3)
	a.andImm(6, 0x0004)
	a.cmpImm(6, 0)
	a.bne(pausePressed)
	a.movImm(7, 0)
	storeWRAM(a, wramStartHeld, 7)
	a.jmp(pauseDone)
	a.mark(pausePressed)
	loadWRAM(a, 7, wramStartHeld)
	a.cmpImm(7, 0)
	a.bne(pauseDone)
	a.movImm(7, 1)
	storeWRAM(a, wramStartHeld, 7)
	loadWRAM(a, 7, wramSceneReturn)
	storeWRAM(a, wramScene, 7)
	a.mark(pauseDone)
	a.jmp("main_loop")

	intNoTurnLeft := a.uniq("int_no_turn_left")
	intAfterTurnLeft := a.uniq("int_after_turn_left")
	intLookLeftWrap := a.uniq("int_look_left_wrap")
	intNoTurnRight := a.uniq("int_no_turn_right")
	intAfterTurnRight := a.uniq("int_after_turn_right")
	intLookRightWrap := a.uniq("int_look_right_wrap")
	intNoMoveForward := a.uniq("int_no_move_forward")
	intNoMoveBackward := a.uniq("int_no_move_backward")
	intClampXMinDone := a.uniq("int_clamp_x_min_done")
	intClampXMaxDone := a.uniq("int_clamp_x_max_done")
	intClampYMinDone := a.uniq("int_clamp_y_min_done")
	intClampYMaxDone := a.uniq("int_clamp_y_max_done")
	intNpcBlockSkip := a.uniq("int_npc_block_skip")
	intTalkOut := a.uniq("int_talk_out")
	intTalkPressed := a.uniq("int_talk_pressed")
	intTalkEdgeDone := a.uniq("int_talk_edge_done")
	intExitOut := a.uniq("int_exit_out")
	intExitPressed := a.uniq("int_exit_pressed")
	intExitEdgeDone := a.uniq("int_exit_edge_done")
	intOutsidePressed := a.uniq("int_outside_pressed")
	intZoneDone := a.uniq("int_zone_done")

	a.mark(sceneInteriorLabel)
	a.movReg(6, 3)
	a.andImm(6, 0x0004)
	a.cmpImm(6, 0)
	a.bne(interiorPausePressed)
	a.movImm(7, 0)
	storeWRAM(a, wramStartHeld, 7)
	a.jmp(interiorPauseDone)
	a.mark(interiorPausePressed)
	loadWRAM(a, 7, wramStartHeld)
	a.cmpImm(7, 0)
	a.bne(interiorPauseDone)
	a.movImm(7, 1)
	storeWRAM(a, wramStartHeld, 7)
	a.movImm(7, sceneInterior)
	storeWRAM(a, wramSceneReturn, 7)
	a.movImm(7, scenePause)
	storeWRAM(a, wramScene, 7)
	a.jmp("main_loop")
	a.mark(interiorPauseDone)

	loadWRAM(a, 5, wramInputLow)
	loadWRAM(a, 0, wramIntHeading)

	a.movReg(4, 5)
	a.andImm(4, 0x0004)
	a.cmpImm(4, 0)
	a.beq(intNoTurnLeft)
	loadWRAM(a, 7, wramTurnTick)
	a.cmpImm(7, 3)
	a.beq(intAfterTurnLeft)
	a.cmpImm(0, 0)
	a.beq(intLookLeftWrap)
	a.subImm(0, 1)
	a.jmp(intAfterTurnLeft)
	a.mark(intLookLeftWrap)
	a.movImm(0, headingSteps-1)
	a.mark(intAfterTurnLeft)
	a.mark(intNoTurnLeft)

	a.movReg(4, 5)
	a.andImm(4, 0x0008)
	a.cmpImm(4, 0)
	a.beq(intNoTurnRight)
	loadWRAM(a, 7, wramTurnTick)
	a.cmpImm(7, 3)
	a.beq(intAfterTurnRight)
	a.cmpImm(0, headingSteps-1)
	a.beq(intLookRightWrap)
	a.addImm(0, 1)
	a.jmp(intAfterTurnRight)
	a.mark(intLookRightWrap)
	a.movImm(0, 0)
	a.mark(intAfterTurnRight)
	a.mark(intNoTurnRight)

	loadWRAM(a, 7, wramTurnTick)
	a.addImm(7, 1)
	a.andImm(7, 3)
	storeWRAM(a, wramTurnTick, 7)
	storeWRAM(a, wramIntHeading, 0)

	emitLoadHeadingEntry(a, headingTableBase, 0, 1, 2, 3, 6)

	loadWRAM(a, 4, wramIntX)
	loadWRAM(a, 0, wramIntY)

	a.movReg(7, 5)
	a.andImm(7, 0x0001)
	a.cmpImm(7, 0)
	a.beq(intNoMoveForward)
	a.addReg(4, 3)
	a.addReg(0, 6)
	a.mark(intNoMoveForward)

	a.movReg(7, 5)
	a.andImm(7, 0x0002)
	a.cmpImm(7, 0)
	a.beq(intNoMoveBackward)
	a.subReg(4, 3)
	a.subReg(0, 6)
	a.mark(intNoMoveBackward)

	a.cmpImm(4, interiorMinX)
	a.bge(intClampXMinDone)
	a.movImm(4, interiorMinX)
	a.mark(intClampXMinDone)

	a.cmpImm(4, interiorMaxX)
	a.ble(intClampXMaxDone)
	a.movImm(4, interiorMaxX)
	a.mark(intClampXMaxDone)

	a.cmpImm(0, interiorMinY)
	a.bge(intClampYMinDone)
	a.movImm(0, interiorMinY)
	a.mark(intClampYMinDone)

	a.cmpImm(0, interiorMaxY)
	a.ble(intClampYMaxDone)
	a.movImm(0, interiorMaxY)
	a.mark(intClampYMaxDone)

	// NPC collision: stop the player in front of the guide.
	a.cmpImm(4, npcBlockMinX)
	a.blt(intNpcBlockSkip)
	a.cmpImm(4, npcBlockMaxX)
	a.bgt(intNpcBlockSkip)
	a.cmpImm(0, npcStopY)
	a.bge(intNpcBlockSkip)
	a.movImm(0, npcStopY)
	a.mark(intNpcBlockSkip)

	storeWRAM(a, wramIntX, 4)
	storeWRAM(a, wramIntY, 0)

	emitEnableInteriorLayers(a)
	loadWRAM(a, 4, wramIntX)
	loadWRAM(a, 0, wramIntY)
	a.movReg(3, 4)
	a.movReg(6, 0)

	// Same feet-pivot split as the overworld: the floor camera trails the
	// player by heading>>2 so turning pivots at the feet, while the NPC
	// billboard tracks the raw player position so it scales correctly.
	a.movReg(7, 1)
	a.sarImm(7, 2)
	a.subReg(3, 7)
	a.movReg(7, 2)
	a.sarImm(7, 2)
	a.subReg(6, 7)
	emitSyncPlaneCameraHeading(a, 0x02, 3, 6, 1, 2)

	loadWRAM(a, 3, wramIntX)
	a.movReg(6, 0)
	emitSyncPlaneCameraHeading(a, 0x03, 3, 6, 1, 2)

	a.movImm(1, 136)
	writeSpriteImm(a, 0, playerSpriteX, 1, playerSpriteTile, playerSpriteAttr, playerSpriteCtrl)
	clearSprite(a, 1)

	emitText(a, 8, 8, 0xF8, 0xF8, 0xF8, "INTERIOR SHOWCASE")
	emitText(a, 8, 20, 0xB0, 0xE0, 0xFF, "TALK TO THE GUIDE  EXIT AT THE DOOR")

	loadWRAM(a, 4, wramIntX)
	loadWRAM(a, 0, wramIntY)
	loadWRAM(a, 5, wramInputLow)

	a.cmpImm(4, npcTalkMinX)
	a.blt(intTalkOut)
	a.cmpImm(4, npcTalkMaxX)
	a.bgt(intTalkOut)
	a.cmpImm(0, npcTalkMaxY)
	a.bgt(intTalkOut)
	a.movReg(6, 5)
	a.andImm(6, 0x0010)
	a.cmpImm(6, 0)
	a.bne(intTalkPressed)
	a.movImm(7, 0)
	storeWRAM(a, wramActionHeld, 7)
	a.jmp(intTalkEdgeDone)
	a.mark(intTalkPressed)
	loadWRAM(a, 7, wramActionHeld)
	a.cmpImm(7, 0)
	a.bne(intTalkEdgeDone)
	a.movImm(7, 1)
	storeWRAM(a, wramActionHeld, 7)
	a.movImm(7, 0)
	storeWRAM(a, wramDialogPage, 7)
	storeWRAM(a, wramDialogReveal, 7)
	a.movImm(7, sceneDialogue)
	storeWRAM(a, wramScene, 7)
	a.jmp("main_loop")
	a.mark(intTalkEdgeDone)
	emitText(a, 104, 176, 0xFF, 0xFF, 0xFF, "PRESS A TO TALK")
	a.jmp(intZoneDone)

	a.mark(intTalkOut)
	a.cmpImm(4, exitZoneMinX)
	a.blt(intExitOut)
	a.cmpImm(4, exitZoneMaxX)
	a.bgt(intExitOut)
	a.cmpImm(0, exitZoneMinY)
	a.blt(intExitOut)
	a.movReg(6, 5)
	a.andImm(6, 0x0010)
	a.cmpImm(6, 0)
	a.bne(intExitPressed)
	a.movImm(7, 0)
	storeWRAM(a, wramActionHeld, 7)
	a.jmp(intExitEdgeDone)
	a.mark(intExitPressed)
	loadWRAM(a, 7, wramActionHeld)
	a.cmpImm(7, 0)
	a.bne(intExitEdgeDone)
	a.movImm(7, 1)
	storeWRAM(a, wramActionHeld, 7)
	a.movImm(7, sceneOverworld)
	storeWRAM(a, wramScene, 7)
	storeWRAM(a, wramSceneReturn, 7)
	a.jmp("main_loop")
	a.mark(intExitEdgeDone)
	emitText(a, 104, 176, 0xFF, 0xFF, 0xFF, "PRESS A TO EXIT")
	a.jmp(intZoneDone)

	a.mark(intExitOut)
	a.movReg(6, 5)
	a.andImm(6, 0x0010)
	a.cmpImm(6, 0)
	a.bne(intOutsidePressed)
	a.movImm(7, 0)
	storeWRAM(a, wramActionHeld, 7)
	a.jmp(intZoneDone)
	a.mark(intOutsidePressed)
	loadWRAM(a, 7, wramActionHeld)
	a.cmpImm(7, 0)
	a.bne(intZoneDone)
	a.movImm(7, 1)
	storeWRAM(a, wramActionHeld, 7)
	a.mark(intZoneDone)
	a.jmp("main_loop")

	a.mark(sceneOverworldLabel)
	loadWRAM(a, 5, wramInputLow)
	loadWRAM(a, 0, wramHeadingIndex)

	a.movReg(4, 5)
	a.andImm(4, 0x0004)
	a.cmpImm(4, 0)
	a.beq(noTurnLeft)
	loadWRAM(a, 7, wramTurnTick)
	a.cmpImm(7, 3)
	a.beq(afterTurnLeft)
	a.cmpImm(0, 0)
	a.beq(lookLeftWrap)
	a.subImm(0, 1)
	a.jmp(afterTurnLeft)
	a.mark(lookLeftWrap)
	a.movImm(0, headingSteps-1)
	a.mark(afterTurnLeft)
	a.mark(noTurnLeft)

	a.movReg(4, 5)
	a.andImm(4, 0x0008)
	a.cmpImm(4, 0)
	a.beq(noTurnRight)
	loadWRAM(a, 7, wramTurnTick)
	a.cmpImm(7, 3)
	a.beq(afterTurnRight)
	a.cmpImm(0, headingSteps-1)
	a.beq(lookRightWrap)
	a.addImm(0, 1)
	a.jmp(afterTurnRight)
	a.mark(lookRightWrap)
	a.movImm(0, 0)
	a.mark(afterTurnRight)
	a.mark(noTurnRight)

	loadWRAM(a, 7, wramTurnTick)
	a.addImm(7, 1)
	a.andImm(7, 3)
	storeWRAM(a, wramTurnTick, 7)
	storeWRAM(a, wramHeadingIndex, 0)

	emitLoadHeadingEntry(a, headingTableBase, 0, 1, 2, 3, 6)

	loadWRAM(a, 4, wramCameraX)
	loadWRAM(a, 0, wramCameraY)

	a.movReg(7, 5)
	a.andImm(7, 0x0001)
	a.cmpImm(7, 0)
	a.beq(noMoveForward)
	a.addReg(4, 3)
	a.addReg(0, 6)
	a.mark(noMoveForward)

	a.movReg(7, 5)
	a.andImm(7, 0x0002)
	a.cmpImm(7, 0)
	a.beq(noMoveBackward)
	a.subReg(4, 3)
	a.subReg(0, 6)
	a.mark(noMoveBackward)

	a.cmpImm(4, worldMinX)
	a.bge(clampXMinDone)
	a.movImm(4, worldMinX)
	a.mark(clampXMinDone)

	a.cmpImm(4, worldMaxX)
	a.ble(clampXMaxDone)
	a.movImm(4, worldMaxX)
	a.mark(clampXMaxDone)

	a.cmpImm(0, worldMinY)
	a.bge(clampYMinDone)
	a.movImm(0, worldMinY)
	a.mark(clampYMinDone)

	a.cmpImm(0, worldMaxY)
	a.ble(clampYMaxDone)
	a.movImm(0, worldMaxY)
	a.mark(clampYMaxDone)

	// Building collision: prevent the player from walking through the building facade.
	// R4=playerX, R0=playerY. If X is within the building footprint and Y would cross
	// the front face, clamp Y to the face so the player stops at the wall.
	a.cmpImm(4, buildingCollisionMinX)
	a.blt(buildingCollisionSkip)
	a.cmpImm(4, buildingCollisionMaxX)
	a.bgt(buildingCollisionSkip)
	a.cmpImm(0, buildingFrontY)
	a.bge(buildingCollisionSkip)
	a.movImm(0, buildingFrontY)
	a.mark(buildingCollisionSkip)

	storeWRAM(a, wramCameraX, 4)
	storeWRAM(a, wramCameraY, 0)

	emitEnableOverworldLayers(a, objects)
	loadWRAM(a, 4, wramCameraX)
	loadWRAM(a, 0, wramCameraY)
	a.movReg(3, 4)
	a.movReg(6, 0)

	// Shift the floor camera backward so rotation pivots at the character's feet
	// rather than the abstract camera origin. feetForward ≈ 64 world units
	// (headingX/Y SAR 2), which maps to screen Y≈148 — the lower half of the
	// player sprite at Y=136 with height 16.
	// Floor uses the offset camera; billboard planes use the player position so
	// the building scales correctly relative to the character.
	a.movReg(7, 1)
	a.sarImm(7, 2)
	a.subReg(3, 7)
	a.movReg(7, 2)
	a.sarImm(7, 2)
	a.subReg(6, 7)
	emitSyncPlaneCameraHeading(a, 0x00, 3, 6, 1, 2) // floor: feet-pivot camera

	loadWRAM(a, 3, wramCameraX) // reload player position for billboard planes
	a.movReg(6, 0)              // R0 still holds playerY
	for _, obj := range objects {
		emitSyncPlaneCameraHeading(a, obj.plane, 3, 6, 1, 2) // billboard: player position
	}

	a.movImm(1, 136)
	writeSpriteImm(a, 0, playerSpriteX, 1, playerSpriteTile, playerSpriteAttr, playerSpriteCtrl)
	clearSprite(a, 1)

	emitText(a, 8, 8, 0xF8, 0xF8, 0xF8, "NITRO PACK-IN DEMO")
	emitText(a, 8, 20, 0xB0, 0xE0, 0xFF, "OVERWORLD BASED ON GENERIC FLOOR+BILLBOARD")
	emitText(a, 8, 32, 0xB0, 0xFF, 0xB0, "UP/DOWN MOVE  LEFT/RIGHT TURN")
	emitText(a, 8, 44, 0xFF, 0xD0, 0x70, "A INTERACTS AT THE BUILDING DOOR")

	loadWRAM(a, 4, wramCameraX)
	loadWRAM(a, 0, wramCameraY)
	loadWRAM(a, 5, wramInputLow)

	a.cmpImm(4, doorMinX)
	a.blt(doorInteractOutOfRange)
	a.cmpImm(4, doorMaxX)
	a.bgt(doorInteractOutOfRange)
	a.cmpImm(0, doorMaxY)
	a.bgt(doorInteractOutOfRange)
	a.movReg(6, 5)
	a.andImm(6, 0x0010)
	a.cmpImm(6, 0)
	a.bne(doorActionPressed)
	a.movImm(7, 0)
	storeWRAM(a, wramActionHeld, 7)
	a.jmp(doorActionEdgeDone)
	a.mark(doorActionPressed)
	loadWRAM(a, 7, wramActionHeld)
	a.cmpImm(7, 0)
	a.bne(doorActionEdgeDone)
	a.movImm(7, sceneInterior)
	storeWRAM(a, wramScene, 7)
	a.movImm(7, 1)
	storeWRAM(a, wramActionHeld, 7)
	a.movImm(7, interiorEntryX)
	storeWRAM(a, wramIntX, 7)
	a.movImm(7, interiorEntryY)
	storeWRAM(a, wramIntY, 7)
	a.movImm(7, interiorEntryHeading)
	storeWRAM(a, wramIntHeading, 7)
	a.jmp("main_loop")
	a.mark(doorActionEdgeDone)
	a.jmp(outsideActionDone)

	a.mark(doorInteractOutOfRange)
	a.movReg(6, 5)
	a.andImm(6, 0x0010)
	a.cmpImm(6, 0)
	a.bne(outsideActionPressed)
	a.movImm(7, 0)
	storeWRAM(a, wramActionHeld, 7)
	a.jmp(outsideActionDone)
	a.mark(outsideActionPressed)
	loadWRAM(a, 7, wramActionHeld)
	a.cmpImm(7, 0)
	a.bne(outsideActionDone)
	a.movImm(7, 1)
	storeWRAM(a, wramActionHeld, 7)
	a.mark(outsideActionDone)

	a.cmpImm(4, doorMinX)
	a.blt(doorPromptOutOfRange)
	a.cmpImm(4, doorMaxX)
	a.bgt(doorPromptOutOfRange)
	a.cmpImm(0, doorMaxY)
	a.bgt(doorPromptOutOfRange)
	emitText(a, 104, 176, 0xFF, 0xFF, 0xFF, "PRESS A TO ENTER")
	a.jmp(doorPromptDone)

	a.mark(doorPromptOutOfRange)

	a.mark(doorPromptDone)
	a.jmp("main_loop")

	// Dialogue scene: the interior render state (layers, planes, sprites)
	// persists from the frame that opened the dialogue, so each frame only
	// redraws the text overlay. One character is revealed per frame; A skips
	// to the full line, then advances pages, then hands off to the credits.
	dialogPageLabels := make([]string, len(dialogPages))
	for i := range dialogPages {
		dialogPageLabels[i] = a.uniq(fmt.Sprintf("dialog_page_%d", i))
	}
	dialogInputLabel := a.uniq("dialog_input")
	dialogPressed := a.uniq("dialog_pressed")
	dialogAdvance := a.uniq("dialog_advance")
	dialogToCredits := a.uniq("dialog_to_credits")
	dialogEdgeDone := a.uniq("dialog_edge_done")

	a.mark(sceneDialogueLabel)
	loadWRAM(a, 0, wramDialogPage)
	for i := 0; i < len(dialogPages)-1; i++ {
		a.cmpImm(0, uint16(i))
		a.beq(dialogPageLabels[i])
	}
	a.jmp(dialogPageLabels[len(dialogPages)-1])

	for i, page := range dialogPages {
		pageLen := uint16(len(page.text))
		revealClamped := a.uniq("dialog_reveal_clamped")
		typeLoop := a.uniq("dialog_type_loop")
		typeDone := a.uniq("dialog_type_done")
		noPrompt := a.uniq("dialog_no_prompt")

		a.mark(dialogPageLabels[i])
		loadWRAM(a, 2, wramDialogReveal)
		a.cmpImm(2, pageLen)
		a.bge(revealClamped)
		a.addImm(2, 1)
		storeWRAM(a, wramDialogReveal, 2)
		a.mark(revealClamped)

		emitTextCentered(a, 152, 0xFF, 0xD0, 0x70, "GUIDE")

		// Stream the revealed prefix of the page text from its WRAM table.
		write16(a, 0x8070, uint16((320-len(page.text)*8)/2))
		write8(a, 0x8072, 172)
		write8(a, 0x8073, 0xFF)
		write8(a, 0x8074, 0xFF)
		write8(a, 0x8075, 0xFF)
		a.movImm(1, page.base)
		loadWRAM(a, 2, wramDialogReveal)
		a.mark(typeLoop)
		a.cmpImm(2, 0)
		a.beq(typeDone)
		a.movLoad(3, 1)
		a.movImm(4, 0x8076)
		a.movStore(4, 3)
		a.addImm(1, 2)
		a.subImm(2, 1)
		a.jmp(typeLoop)
		a.mark(typeDone)

		loadWRAM(a, 2, wramDialogReveal)
		a.cmpImm(2, pageLen)
		a.blt(noPrompt)
		emitTextCentered(a, 188, 0xA0, 0xD8, 0xFF, "PRESS A")
		a.mark(noPrompt)

		loadWRAM(a, 2, wramDialogReveal)
		a.movImm(3, pageLen)
		a.jmp(dialogInputLabel)
	}

	// Shared dialogue input handling. R2=reveal, R3=page length.
	a.mark(dialogInputLabel)
	loadWRAM(a, 5, wramInputLow)
	a.movReg(6, 5)
	a.andImm(6, 0x0010)
	a.cmpImm(6, 0)
	a.bne(dialogPressed)
	a.movImm(7, 0)
	storeWRAM(a, wramActionHeld, 7)
	a.jmp(dialogEdgeDone)
	a.mark(dialogPressed)
	loadWRAM(a, 7, wramActionHeld)
	a.cmpImm(7, 0)
	a.bne(dialogEdgeDone)
	a.movImm(7, 1)
	storeWRAM(a, wramActionHeld, 7)
	a.cmpReg(2, 3)
	a.bge(dialogAdvance)
	storeWRAM(a, wramDialogReveal, 3)
	a.jmp(dialogEdgeDone)
	a.mark(dialogAdvance)
	loadWRAM(a, 0, wramDialogPage)
	a.addImm(0, 1)
	a.cmpImm(0, dialogPageCount)
	a.bge(dialogToCredits)
	storeWRAM(a, wramDialogPage, 0)
	a.movImm(7, 0)
	storeWRAM(a, wramDialogReveal, 7)
	a.jmp(dialogEdgeDone)
	a.mark(dialogToCredits)
	a.movImm(7, 0)
	storeWRAM(a, wramDialogPage, 7)
	storeWRAM(a, wramDialogReveal, 7)
	a.movImm(7, sceneCredits)
	storeWRAM(a, wramScene, 7)
	a.mark(dialogEdgeDone)
	a.jmp("main_loop")

	// Credits scene: START performs a full state reset back to the title.
	creditsPressed := a.uniq("credits_pressed")
	creditsDone := a.uniq("credits_done")

	a.mark(sceneCreditsLabel)
	emitDisableAllSceneLayers(a)
	clearSprite(a, 0)
	clearSprite(a, 1)
	emitTextCentered(a, 48, 0xFF, 0xFF, 0xFF, "NITRO PACK-IN DEMO")
	emitTextCentered(a, 76, 0x90, 0xD8, 0xFF, "A NITRO-CORE-DX SHOWCASE")
	emitTextCentered(a, 104, 0xC0, 0xE0, 0x90, "ENGINE  TOOLS  ROM  BY RETROCODERAMEN")
	emitTextCentered(a, 128, 0xB0, 0xFF, 0xB0, "THANKS FOR PLAYING")
	emitTextCentered(a, 160, 0xFF, 0xC0, 0x70, "PRESS START FOR TITLE")
	a.movReg(6, 3)
	a.andImm(6, 0x0004)
	a.cmpImm(6, 0)
	a.bne(creditsPressed)
	a.movImm(7, 0)
	storeWRAM(a, wramStartHeld, 7)
	a.jmp(creditsDone)
	a.mark(creditsPressed)
	loadWRAM(a, 7, wramStartHeld)
	a.cmpImm(7, 0)
	a.bne(creditsDone)
	a.movImm(7, 1)
	storeWRAM(a, wramStartHeld, 7)
	a.movImm(7, 512)
	storeWRAM(a, wramCameraX, 7)
	a.movImm(7, 768)
	storeWRAM(a, wramCameraY, 7)
	a.movImm(7, 48)
	storeWRAM(a, wramHeadingIndex, 7)
	a.movImm(7, 0)
	storeWRAM(a, wramTurnTick, 7)
	storeWRAM(a, wramDialogPage, 7)
	storeWRAM(a, wramDialogReveal, 7)
	a.movImm(7, sceneOverworld)
	storeWRAM(a, wramSceneReturn, 7)
	a.movImm(7, sceneTitle)
	storeWRAM(a, wramScene, 7)
	a.mark(creditsDone)
	a.jmp("main_loop")

	if err := a.resolve(); err != nil {
		return err
	}

	payload := append([]byte{}, floorAsset.Program.Bitmap...)
	payload = append(payload, mainBuildingAsset.Program.Bitmap...)
	payload = append(payload, interiorFloorAsset.Program.Bitmap...)
	payload = append(payload, npcAsset.Program.Bitmap...)
	if err := appendDataBlob(a.B, dataStartBank, payload); err != nil {
		return err
	}

	return a.B.BuildROM(codeBank, 0x8000, outPath)
}

func main() {
	floorPath := flag.String("floor", "Games/NitroPackInDemo/park.png", "floor PNG image")
	billboardPath := flag.String("billboard", "Games/NitroPackInDemo/building.png", "main building PNG image")
	outPath := flag.String("out", "roms/nitro_pack_in_demo.rom", "output ROM path")
	flag.Parse()

	floorImg, err := loadPNG(*floorPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load %s: %v\n", *floorPath, err)
		os.Exit(1)
	}
	billboardImg, err := loadPNG(*billboardPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load %s: %v\n", *billboardPath, err)
		os.Exit(1)
	}
	if err := buildNitroPackInDemoROM(floorImg, billboardImg, *outPath); err != nil {
		fmt.Fprintf(os.Stderr, "build ROM: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Built %s using %s and %s\n", *outPath, *floorPath, *billboardPath)
}
