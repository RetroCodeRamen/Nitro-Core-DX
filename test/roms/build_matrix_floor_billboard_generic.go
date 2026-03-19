//go:build testrom_tools
// +build testrom_tools

package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"

	"nitro-core-dx/internal/emulator"
	ppucore "nitro-core-dx/internal/ppu"
	"nitro-core-dx/test/roms/romutil"
)

// This ROM is the generic-projection baseline for a floor + billboard scene.
// Both BG0 (floor) and BG1 (billboard) are driven from the same live
// camera/heading in WRAM:
//   - LEFT/RIGHT: rotate heading
//   - UP/DOWN   : move camera in world space
//
// CoreLX should eventually emit this pattern for 3D-ish matrix floors.

type asm struct{ *romutil.Asm }

func newASM(bank uint8) *asm { return &asm{Asm: romutil.NewASM(bank)} }

// Thin wrappers mirroring the reference builder so we can use the same style of code.
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

func (a *asm) beq(label string) { a.Beq(label) }
func (a *asm) bne(label string) { a.Bne(label) }
func (a *asm) jmp(label string) { a.Jmp(label) }

var (
	allocateROMData = romutil.AllocateROMData
	appendDataBlob  = romutil.AppendDataBlob
)

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

func buildMatrixFloorBillboardGenericROM(floorImg, billboardImg image.Image, outPath string) error {
	const (
		codeBank         = 1
		dataStartBank    = 2
		wramLastFrame    = 0x0200
		wramHeadingIndex = 0x0202
		wramCameraX      = 0x0204
		wramCameraY      = 0x0206
		wramTurnTick     = 0x0208 // turn-rate throttling counter (0..3)
		headingTableBase = 0x0300
		headingSteps     = 64

		matrixPlaneFloorCtl = 0x1D // enabled, 128x128, bitmap, palette bank 1
		// Billboard asset is built as 64x64, so the matrix plane control must request 64x64 too.
		// Control bits (see PPU decode):
		//   bit0 enable
		//   bits[2:1] size (0=32,1=64,2=128)
		//   bit3 source mode (1=bitmap)
		//   bits[7:4] bitmap palette bank
		// For palette bank 2 and size=64:
		//   enable(1)=0x01 + size(1<<1)=0x02 + bitmap(1<<3)=0x08 + palette(2<<4)=0x20 => 0x2B
		matrixPlaneBillboardCtl = 0x2B // enabled, 64x64, bitmap, palette bank 2
		matrixPlaneFloorFlags   = 0x00 // opaque floor
		matrixPlaneBillFlags    = 0x03 // transparent index 0, two-sided vertical quad
	)

	floorAsset, err := emulator.BuildBitmapMatrixPlaneAssetFromImage(floorImg, 0, ppucore.TilemapSize128x128, 1)
	if err != nil {
		return err
	}
	billboardAsset, err := emulator.BuildBitmapMatrixPlaneAssetFromImage(billboardImg, 1, ppucore.TilemapSize64x64, 2)
	if err != nil {
		return err
	}

	floorRef, cursor := allocateROMData(0, floorAsset.Program.Bitmap)
	billboardRef, cursor := allocateROMData(cursor, billboardAsset.Program.Bitmap)
	_ = cursor

	a := newASM(codeBank)

	// Palette.
	for i, c := range floorAsset.Palette {
		setCGRAMColor(a, uint8(1*16+i), c)
	}
	for i, c := range billboardAsset.Palette {
		setCGRAMColor(a, uint8(2*16+i), c)
	}
	setCGRAMColor(a, 0, 0)

	// Initial camera.
	write16(a, wramHeadingIndex, 48) // roughly "up" in source space
	write16(a, wramCameraX, 512)
	write16(a, wramCameraY, 768)
	write16(a, wramTurnTick, 0)
	// Tune movement speed (UP/DOWN). Higher = faster forward/back motion.
	// Tune forward/back speed.
	emitInitHeadingTable(a, headingTableBase, headingSteps, 3.6)

	// BG config: BG0=floor, BG1=billboard.
	write8(a, 0x8008, 0x21)
	write8(a, 0x8009, 0x25)
	write8(a, 0x806C, 0x00) // BG0 -> channel 0
	write8(a, 0x806D, 0x01) // BG1 -> channel 1

	// Upload floor bitmap into matrix plane 0.
	write8(a, 0x8080, 0x00)
	write8(a, 0x8081, matrixPlaneFloorCtl)
	write8(a, 0x808C, matrixPlaneFloorFlags)
	write8(a, 0x8088, 0x00)
	write8(a, 0x8089, 0x00)
	write8(a, 0x808A, 0x00)
	emitMatrixBitmapDMAChunks(a, 0x00, floorRef)

	// Upload billboard bitmap into matrix plane 1.
	write8(a, 0x8080, 0x01)
	write8(a, 0x8081, matrixPlaneBillboardCtl)
	write8(a, 0x808C, matrixPlaneBillFlags)
	write8(a, 0x8088, 0x00)
	write8(a, 0x8089, 0x00)
	write8(a, 0x808A, 0x00)
	emitMatrixBitmapDMAChunks(a, 0x01, billboardRef)

	// Ensure DBR=0 for MMIO and latch frame counter baseline.
	a.movImm(0, 0x0000)
	a.setDBR(0)
	a.movImm(4, 0x803F)
	a.movLoad(2, 4)
	a.movImm(4, wramLastFrame)
	a.movStore(4, 2)

	// Configure floor plane (plane 0) generic perspective.
	// Tuning knobs:
	// - BaseDistance: acts like camera height above the ground (bigger = less "near-ground" warp)
	// - FocalLength: bigger = narrower FOV (less warping)
	// - Horizon: shifts the horizon line on screen
	write8(a, 0x8080, 0x00)
	write8(a, 0x8081, matrixPlaneFloorCtl)
	write8(a, 0x808C, matrixPlaneFloorFlags)
	write8(a, 0x8091, 0x01) // generic perspective projection
	// Aim for a horizon around ~1/3 from bottom of a 200px screen (~133).
	// Also reduce mid-ground warp by further raising camera and narrowing FOV.
	// Raise the board by about ~15px in screen space.
	// In this PPU projection, decreasing Horizon moves the rendered floor upward.
	write8(a, 0x8092, 113) // horizon
	// Lower view further: reduce base distance so the camera sits even closer
	// to the ground. Keep FOV reasonably narrow to limit curvature.
	write16(a, 0x809B, 0x0C00)
	// Keep FOV narrow-ish to reduce curvature (fisheye) while lowering view.
	write16(a, 0x809D, 0xC000)
	write16(a, 0x809F, 0x00C0) // width scale

	// Seed floor camera/heading from initial WRAM state.
	a.movImm(0, 48)
	emitLoadHeadingEntry(a, headingTableBase, 0, 1, 2, 3, 6)
	write16(a, 0x8093, 512)
	write16(a, 0x8095, 768)
	write16RegBytes(a, 0x8097, 1)
	write16RegBytes(a, 0x8099, 2)

	// Configure billboard (plane 1) vertical quad, sharing camera.
	// Make it less extreme when you get close: push it farther, reduce width, increase height scale.
	write8(a, 0x8080, 0x01)
	write8(a, 0x8091, ppucore.MatrixPlaneProjectionVertical)
	// Billboard vertical projection horizon. Lowering this moves the quad
	// upward on screen (increasing distance-from-bottom) to match the target
	// overlap with the ground.
	write8(a, 0x8092, 63)
	// Start from the known-good vertical-quad config used by the showcase ROM.
	// This ensures the billboard projects as a proper textured quad instead of collapsing.
	write16(a, 0x809B, 0x01C0) // billboardBaseDist
	write16(a, 0x809D, 0x3A00) // billboardFocal
	write16(a, 0x809F, 0x00B8) // billboardWidthScale
	write16(a, 0x80A1, 512)
	// Lower the billboard in world space (it was floating too high).
	// Push it further down so it fills the view when you're close.
	// Move the quad toward the camera / ground depth so the bottom edge
	// overlaps the floor instead of floating above it.
	write16(a, 0x80A3, 686) // billboardOriginY
	write16s(a, 0x80A5, 0)
	write16s(a, 0x80A7, 0x0100) // billboard faces +Y in source space
	// Increase billboard quad vertical size so that when you're close it
	// occupies a much larger portion of the screen (not just ~10px tall).
	// HeightScale controls how tall the vertical-quad projects in screen space.
	// Current tuning was slightly better; user wants ~5x taller.
	// Stretch out a bit more than the previous doubling.
	write16(a, 0x80A9, 0x9A00)

	// Enable matrix backgrounds and display.
	write8(a, 0x8018, 0x01) // BG0 matrix mode on
	write8(a, 0x802B, 0x01) // BG1 matrix mode on
	write8(a, 0x8011, 0x01) // display on

	a.mark("main_loop")
	a.movImm(0, 0x0000)
	a.setDBR(0)
	emitWaitOneFrame(a, wramLastFrame)

	// Read controller.
	write8(a, 0xA001, 0x01)
	a.movImm(4, 0xA000)
	a.movLoad(2, 4)
	write8(a, 0xA001, 0x00)
	a.movReg(5, 2)

	// Heading: LEFT/RIGHT.
	a.movImm(4, wramHeadingIndex)
	a.movLoad(0, 4)

	noTurnLeft := a.uniq("no_turn_left")
	afterTurnLeft := a.uniq("after_turn_left")
	lookLeftWrap := a.uniq("look_left_wrap")

	a.movReg(4, 5)
	a.andImm(4, 0x0004)
	a.cmpImm(4, 0)
	a.beq(noTurnLeft)
	// Throttle turning to ~75% speed: allow turn on ticks 0..2, skip on tick 3.
	a.movImm(4, wramTurnTick)
	a.movLoad(6, 4)
	a.cmpImm(6, 3)
	a.beq(afterTurnLeft)
	a.cmpImm(0, 0)
	a.beq(lookLeftWrap)
	a.subImm(0, 1)
	a.jmp(afterTurnLeft)
	a.mark(lookLeftWrap)
	a.movImm(0, headingSteps-1)
	a.mark(afterTurnLeft)
	a.mark(noTurnLeft)

	noTurnRight := a.uniq("no_turn_right")
	a.movReg(4, 5)
	a.andImm(4, 0x0008)
	a.cmpImm(4, 0)
	a.beq(noTurnRight)
	afterTurnRight := a.uniq("after_turn_right")
	// Same throttling as LEFT.
	a.movImm(4, wramTurnTick)
	a.movLoad(6, 4)
	a.cmpImm(6, 3)
	a.beq(afterTurnRight)
	lookRightWrap := a.uniq("look_right_wrap")
	a.cmpImm(0, headingSteps-1)
	a.beq(lookRightWrap)
	a.addImm(0, 1)
	a.jmp(afterTurnRight)
	a.mark(lookRightWrap)
	a.movImm(0, 0)
	a.mark(afterTurnRight)
	a.mark(noTurnRight)

	// tick = (tick + 1) & 3
	a.movImm(4, wramTurnTick)
	a.movLoad(6, 4)
	a.addImm(6, 1)
	a.andImm(6, 3)
	a.movStore(4, 6)

	a.movImm(4, wramHeadingIndex)
	a.movStore(4, 0)

	// Resolve heading vectors + move deltas.
	emitLoadHeadingEntry(a, headingTableBase, 0, 1, 2, 3, 6)

	// Load camera from WRAM.
	a.movImm(7, wramCameraX)
	a.movLoad(4, 7)
	a.movImm(7, wramCameraY)
	a.movLoad(0, 7)

	// UP: move forward.
	noMoveForward := a.uniq("no_move_forward")
	a.movReg(7, 5)
	a.andImm(7, 0x0001)
	a.cmpImm(7, 0)
	a.beq(noMoveForward)
	a.addReg(4, 3)
	a.addReg(0, 6)
	a.mark(noMoveForward)

	// DOWN: move backward.
	noMoveBackward := a.uniq("no_move_backward")
	a.movReg(7, 5)
	a.andImm(7, 0x0002)
	a.cmpImm(7, 0)
	a.beq(noMoveBackward)
	a.subReg(4, 3)
	a.subReg(0, 6)
	a.mark(noMoveBackward)

	// Write back camera to WRAM.
	a.movImm(7, wramCameraX)
	a.movStore(7, 4)
	a.movImm(7, wramCameraY)
	a.movStore(7, 0)

	// Write camera/heading into both planes.
	// Plane 0 (floor).
	write8Scratch(a, 0x8080, 0x00, 7, 5)
	a.movReg(3, 4)
	a.movReg(6, 0)
	write16RegBytes(a, 0x8093, 3)
	write16RegBytes(a, 0x8095, 6)
	write16RegBytes(a, 0x8097, 1)
	write16RegBytes(a, 0x8099, 2)

	// Plane 1 (billboard).
	write8Scratch(a, 0x8080, 0x01, 7, 5)
	write16RegBytes(a, 0x8093, 3)
	write16RegBytes(a, 0x8095, 6)
	write16RegBytes(a, 0x8097, 1)
	write16RegBytes(a, 0x8099, 2)

	emitText(a, 8, 8, 0xF8, 0xF8, 0xF8, "GENERIC FLOOR+BILLBOARD REFERENCE")
	emitText(a, 8, 20, 0xB0, 0xE0, 0xFF, "BG0 FLOOR (GENERIC PROJECTION)")
	emitText(a, 8, 32, 0xB0, 0xFF, 0xB0, "BG1 BILLBOARD (VERTICAL QUAD)")
	emitText(a, 8, 44, 0xFF, 0xD0, 0x70, "UP/DOWN MOVE  LEFT/RIGHT TURN")

	a.jmp("main_loop")

	if err := a.resolve(); err != nil {
		return err
	}

	payload := append([]byte{}, floorAsset.Program.Bitmap...)
	payload = append(payload, billboardAsset.Program.Bitmap...)
	if err := appendDataBlob(a.B, dataStartBank, payload); err != nil {
		return err
	}

	return a.B.BuildROM(codeBank, 0x8000, outPath)
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func main() {
	inPath := flag.String("in", "Resources/kart.png", "floor PNG image")
	billboardPath := flag.String("billboard", "Resources/Test.png", "billboard PNG image")
	outPath := flag.String("out", "roms/matrix_floor_billboard_generic.rom", "output ROM path")
	flag.Parse()

	floorImg, err := loadPNG(*inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load %s: %v\n", *inPath, err)
		os.Exit(1)
	}
	billboardImg, err := loadPNG(*billboardPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load %s: %v\n", *billboardPath, err)
		os.Exit(1)
	}
	if err := buildMatrixFloorBillboardGenericROM(floorImg, billboardImg, *outPath); err != nil {
		fmt.Fprintf(os.Stderr, "build ROM: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Built %s using %s floor and %s billboard (generic projection)\n", *outPath, *inPath, *billboardPath)
}
