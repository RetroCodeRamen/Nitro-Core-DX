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
func write8Scratch(a *asm, addr uint16, value uint8, addrReg, valueReg uint8) {
	romutil.Write8Scratch(a.Asm, addr, value, addrReg, valueReg)
}
func emitText(a *asm, x uint16, y uint8, r, g, b uint8, text string) {
	romutil.EmitText(a.Asm, x, y, r, g, b, text)
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
	for _, obj := range objects {
		write8(a, obj.bgControlAddr, obj.bgControlValue)
		write8(a, obj.transformBindAddr, obj.plane)
		write8(a, obj.matrixControlAddr, 0x01)
	}
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

		headingTableBase = 0x0300
		headingSteps     = 64

		playerSpriteTile = 8
		playerSpriteX    = 152
		playerSpriteCtrl = 0x03
		playerSpriteAttr = 0x05

		floorPlaneCtl        = 0x1D
		floorPlaneFlags      = 0x00
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
	)

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

	floorRef, cursor := allocateROMData(0, floorAsset.Program.Bitmap)
	mainBuildingRef, cursor := allocateROMData(cursor, mainBuildingAsset.Program.Bitmap)
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

	a := newASM(codeBank)

	for i, c := range floorAsset.Palette {
		setCGRAMColor(a, uint8(1*16+i), c)
	}
	for i, c := range mainBuildingAsset.Palette {
		setCGRAMColor(a, uint8(2*16+i), c)
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
	emitInitHeadingTable(a, headingTableBase, headingSteps, 3.6)

	write8(a, 0x8008, 0x21)
	write8(a, 0x806C, 0x00)
	emitUploadPlaneBitmap(a, 0x00, floorPlaneCtl, floorPlaneFlags, floorRef)
	emitUploadPlaneBitmap(a, 0x01, objects[0].planeControl, objects[0].flags, mainBuildingRef)

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

	titlePressed := a.uniq("title_pressed")
	titleDone := a.uniq("title_done")
	pausePressed := a.uniq("pause_pressed")
	pauseDone := a.uniq("pause_done")
	interiorPausePressed := a.uniq("interior_pause_pressed")
	interiorPauseDone := a.uniq("interior_pause_done")
	interiorActionPressed := a.uniq("interior_action_pressed")
	interiorActionDone := a.uniq("interior_action_done")

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
	a.jmp(sceneOverworldLabel)

	a.mark(sceneTitleLabel)
	emitDisableAllSceneLayers(a)
	clearSprite(a, 0)
	clearSprite(a, 1)
	emitText(a, 88, 42, 0xFF, 0xFF, 0xFF, "NITRO PACK-IN DEMO")
	emitText(a, 72, 66, 0x90, 0xD8, 0xFF, "M2 OVERWORLD VERTICAL SLICE")
	emitText(a, 96, 98, 0xFF, 0xFF, 0xFF, "PRESS START")
	emitText(a, 44, 130, 0xC0, 0xE0, 0x90, "BUILDING ROW  TREES  DOOR INTERACT")
	emitText(a, 62, 154, 0xFF, 0xC0, 0x70, "INTERIOR ROOM PLACEHOLDER ACTIVE")
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

	a.mark(sceneInteriorLabel)
	emitDisableAllSceneLayers(a)
	clearSprite(a, 0)
	clearSprite(a, 1)
	emitText(a, 76, 44, 0xFF, 0xFF, 0xFF, "INTERIOR SHOWCASE STUB")
	emitText(a, 52, 72, 0x90, 0xD8, 0xFF, "NEXT: FLOOR CEILING WALLS NPC")
	emitText(a, 64, 104, 0xC0, 0xE0, 0x90, "A RETURNS TO THE OVERWORLD")
	emitText(a, 72, 128, 0xFF, 0xC0, 0x70, "START STILL OPENS PAUSE")
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
	a.movReg(6, 2)
	a.andImm(6, 0x0010)
	a.cmpImm(6, 0)
	a.bne(interiorActionPressed)
	a.movImm(7, 0)
	storeWRAM(a, wramActionHeld, 7)
	a.jmp(interiorActionDone)
	a.mark(interiorActionPressed)
	loadWRAM(a, 7, wramActionHeld)
	a.cmpImm(7, 0)
	a.bne(interiorActionDone)
	a.movImm(7, 1)
	storeWRAM(a, wramActionHeld, 7)
	a.movImm(7, sceneOverworld)
	storeWRAM(a, wramScene, 7)
	a.mark(interiorActionDone)
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

	storeWRAM(a, wramCameraX, 4)
	storeWRAM(a, wramCameraY, 0)

	emitEnableOverworldLayers(a, objects)
	loadWRAM(a, 4, wramCameraX)
	loadWRAM(a, 0, wramCameraY)
	a.movReg(3, 4)
	a.movReg(6, 0)
	emitSyncPlaneCameraHeading(a, 0x00, 3, 6, 1, 2)
	for _, obj := range objects {
		emitSyncPlaneCameraHeading(a, obj.plane, 3, 6, 1, 2)
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

	if err := a.resolve(); err != nil {
		return err
	}

	payload := append([]byte{}, floorAsset.Program.Bitmap...)
	payload = append(payload, mainBuildingAsset.Program.Bitmap...)
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
