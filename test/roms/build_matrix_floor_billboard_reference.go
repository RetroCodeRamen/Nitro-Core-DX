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

// This builder intentionally does NOT reuse the earlier showcase ROMs. It is
// a minimal reference that exercises the cleaned generic matrix-plane
// contract:
//   - Plane 0: bitmap-backed floor from Resources/kart.png using ROW MODE
//     (per-scanline StartX/Y, StepX/Y) to match SNES Mode 7-style perspective.
//   - Plane 1: bitmap-backed vertical projected billboard from Resources/Test.png
//   - Controller: LEFT/RIGHT turn, UP/DOWN move forward/backward in world space.
//
// Floor uses precomputed row tables (64 headings); billboard uses generic
// vertical-quad projection. No built-in perspective mode for the floor.

type asm struct{ *romutil.Asm }

func newASM(bank uint8) *asm { return &asm{Asm: romutil.NewASM(bank)} }

func (a *asm) pc() uint16                  { return a.PC() }
func (a *asm) mark(name string)            { a.Mark(name) }
func (a *asm) uniq(prefix string) string   { return a.Uniq(prefix) }
func (a *asm) inst(w uint16)               { a.Inst(w) }
func (a *asm) imm(v uint16)                { a.Imm(v) }
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

// Thin wrappers to keep the builder code concise.
func (a *asm) beq(label string) { a.Beq(label) }
func (a *asm) bne(label string) { a.Bne(label) }
func (a *asm) jmp(label string) { a.Jmp(label) }

func write8(a *asm, addr uint16, value uint8) { romutil.Write8(a.Asm, addr, value) }
func write8Scratch(a *asm, addr uint16, value uint8, addrReg, valueReg uint8) {
	romutil.Write8Scratch(a.Asm, addr, value, addrReg, valueReg)
}
func write16(a *asm, addr uint16, value uint16) { romutil.Write16(a.Asm, addr, value) }
func write16s(a *asm, addr uint16, value int16) { romutil.Write16S(a.Asm, addr, value) }
func write16RegBytes(a *asm, addr uint16, reg uint8) {
	romutil.Write16RegBytes(a.Asm, addr, reg)
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

func emitWriteMatrixRegs(a *asm, controlAddr, aAddr, bAddr, cAddr, dAddr, cxAddr, cyAddr uint16, controlValue uint8, aReg, bReg, cReg, dReg uint8, centerX, centerY int16) {
	romutil.EmitWriteMatrixRegs(a.Asm, controlAddr, aAddr, bAddr, cAddr, dAddr, cxAddr, cyAddr, controlValue, aReg, bReg, cReg, dReg, centerX, centerY)
}
func emitMatrixRowDMAChunks(a *asm, channel uint8, ref romutil.DataRef) {
	romutil.EmitMatrixRowDMAChunks(a.Asm, channel, ref)
}

// emitInitHeadingTable writes a table of [cos, sin, moveX, moveY] pairs into WRAM.
// cos/sin are 8.8 fixed-point; moveX/moveY are per-step pixel deltas in world space.
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

// emitLoadHeadingEntry loads one [cos, sin, moveX, moveY] entry from the heading table.
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

// put32LE writes a int32 in little-endian into b[0:4].
func put32LE(b []byte, v int32) {
	u := uint32(v)
	b[0] = uint8(u)
	b[1] = uint8(u >> 8)
	b[2] = uint8(u >> 16)
	b[3] = uint8(u >> 24)
}

// buildFloorRowTables builds 64 row tables (one per heading) for a fixed camera.
//
// Uses the same row-table formula as the pre-cleanup floor demos (build_matrix_floor_only,
// build_matrix_rowmode_showcase) so the look matches "before we separated demo code from
// the PPU". The PPU stays generic (row mode = Start + Step*x per scanline); only the
// ROM-supplied table changes.
//
// Formula (float, then *65536 for 16.16):
//
//	line = (y - horizon) + 1
//	step = 1.6 / (1 + line/18), clamp [0.08, 1.6]  — independent horizontal scale curve
//	forward = 3072 / (line + 6)                     — center depth, "+6" softens
//	du,dv = right*step; center = camera + forward*heading; start = center - 160*du
func buildFloorRowTables(camX, camY int, horizon uint8, _, _, _ uint16) [][]byte {
	const (
		screenCX = 160.0
		stride   = 16
		stepMax  = 1.6
		stepMin  = 0.08
		lineDiv  = 18.0
		forwardK = 3072.0
		forwardB = 6.0
	)
	tables := make([][]byte, 64)
	for idx := 0; idx < 64; idx++ {
		angle := (2.0 * math.Pi * float64(idx)) / 64.0
		headingX := math.Cos(angle)
		headingY := math.Sin(angle)
		rightX := headingY
		rightY := -headingX
		cameraX := float64(camX)
		cameraY := float64(camY)
		table := make([]byte, ppucore.VisibleScanlines*stride)
		for y := 0; y < ppucore.VisibleScanlines; y++ {
			base := y * stride
			startX := int32(0)
			startY := int32(0)
			stepX := int32(0)
			stepY := int32(0)
			if y >= int(horizon) {
				line := float64(y-int(horizon)) + 1.0
				step := stepMax / (1.0 + line/lineDiv)
				if step < stepMin {
					step = stepMin
				}
				if step > stepMax {
					step = stepMax
				}
				forward := forwardK / (line + forwardB)
				du := rightX * step
				dv := rightY * step
				rowCenterX := cameraX + headingX*forward
				rowCenterY := cameraY + headingY*forward
				rowStartX := rowCenterX - screenCX*du
				rowStartY := rowCenterY - screenCX*dv
				startX = int32(math.Round(rowStartX * 65536.0))
				startY = int32(math.Round(rowStartY * 65536.0))
				stepX = int32(math.Round(du * 65536.0))
				stepY = int32(math.Round(dv * 65536.0))
			}
			put32LE(table[base+0:base+4], startX)
			put32LE(table[base+4:base+8], startY)
			put32LE(table[base+8:base+12], stepX)
			put32LE(table[base+12:base+16], stepY)
		}
		tables[idx] = table
	}
	return tables
}

func buildMatrixFloorBillboardReferenceROM(floorImg, billboardImg image.Image, outPath string) error {
	const (
		codeBank      = 1
		dataStartBank = 2

		wramLastFrame        = 0x0200
		wramHeadingIndex     = 0x0202
		wramCameraX          = 0x0204
		wramCameraY          = 0x0206
		wramLastFloorHeading = 0x0208 // only DMA row table when heading changes (saves FPS)
		trigTableBase        = 0x0300
		trigSteps            = 64

		matrixPlane0 = 0 // floor
		matrixPlane1 = 1 // billboard

		// Slightly slower turn, but much more obvious forward/back motion in the real-time emulator.
		headingTurnStep = 2 // ~11° per left/right frame while held
		moveSpeedPixels = 80

		// Floor: same formula as pre-cleanup demos (step = 1.6/(1+line/18), forward = 3072/(line+6)).
		floorHorizon    = 92
		floorBaseDist   = 0
		floorFocal      = 0
		floorWidthScale = 0
		floorCamX       = 512
		floorCamY       = 768

		// Billboard projection. Match floor feel; slightly tighter horizontal.
		billboardHorizon     = 72
		billboardBaseDist    = 0x0C00
		billboardFocal       = 0x4000
		billboardWidthScale  = 0x0070
		billboardHeightScale = 0x0280
		billboardWorldX      = 512
		billboardWorldY      = 640
	)

	if floorImg == nil {
		return fmt.Errorf("floor image is required")
	}
	if billboardImg == nil {
		return fmt.Errorf("billboard image is required")
	}

	// Each plane is a 1024x1024 bitmap-backed matrix source.
	floorAsset, err := emulator.BuildBitmapMatrixPlaneAssetFromImage(
		floorImg, matrixPlane0, ppucore.TilemapSize128x128, 1,
	)
	if err != nil {
		return err
	}
	billboardAsset, err := emulator.BuildBitmapMatrixPlaneAssetFromImage(
		billboardImg, matrixPlane1, ppucore.TilemapSize64x64, 2,
	)
	if err != nil {
		return err
	}

	// Build 64 floor row tables (one per heading) at fixed camera for row-mode floor.
	floorRowTables := buildFloorRowTables(floorCamX, floorCamY, floorHorizon, floorBaseDist, floorFocal, floorWidthScale)
	// Place bitmaps and row tables in ROM data space.
	floorRef, cursor := romutil.AllocateROMData(0, floorAsset.Program.Bitmap)
	billboardRef, cursor := romutil.AllocateROMData(cursor, billboardAsset.Program.Bitmap)
	rowTableRefs := make([]romutil.DataRef, 64)
	for i := 0; i < 64; i++ {
		rowTableRefs[i], cursor = romutil.AllocateROMData(cursor, floorRowTables[i])
	}
	_ = billboardRef

	a := newASM(codeBank)

	// Palette: floor in bank 1, billboard in bank 2, backdrop = 0.
	for i, c := range floorAsset.Palette {
		setCGRAMColor(a, uint8(1*16+i), c)
	}
	for i, c := range billboardAsset.Palette {
		setCGRAMColor(a, uint8(2*16+i), c)
	}
	setCGRAMColor(a, 0, 0)

	// Initial camera state: slightly "behind" kart image, facing up (-Y).
	write16(a, wramHeadingIndex, 48)         // 48/64 ≈ 270 degrees => roughly -Y.
	write16(a, wramLastFloorHeading, 0x00FF) // invalid so first frame always uploads row table
	write16(a, wramCameraX, 512)
	write16(a, wramCameraY, 768)
	emitInitHeadingTable(a, trigTableBase, trigSteps, float64(moveSpeedPixels))

	// BG0 = floor, BG1 = billboard (higher priority).
	write8(a, 0x8008, 0x21) // BG0 enabled, 128x128
	write8(a, 0x8009, 0x25) // BG1 enabled, 128x128, higher priority
	write8(a, 0x806C, 0x00) // BG0 -> transform channel 0
	write8(a, 0x806D, 0x01) // BG1 -> transform channel 1

	// Upload floor bitmap into matrix plane 0.
	write8(a, 0x8080, matrixPlane0)
	write8(a, 0x8081, 0x1D) // enabled, 128x128, bitmap, palette bank 1
	write8(a, 0x808C, 0x00) // opaque bitmap (index 0 not transparent)
	write8(a, 0x8088, 0x00)
	write8(a, 0x8089, 0x00)
	write8(a, 0x808A, 0x00)
	romutil.EmitMatrixBitmapDMAChunks(a.Asm, matrixPlane0, floorRef)

	// Upload billboard bitmap into matrix plane 1.
	write8(a, 0x8080, matrixPlane1)
	write8(a, 0x8081, 0x2D) // enabled, 128x128, bitmap, palette bank 2
	write8(a, 0x808C, 0x03) // transparent index 0, two-sided vertical quad
	write8(a, 0x8088, 0x00)
	write8(a, 0x8089, 0x00)
	write8(a, 0x808A, 0x00)
	romutil.EmitMatrixBitmapDMAChunks(a.Asm, matrixPlane1, billboardRef)

	// Ensure DBR=0 for MMIO.
	a.movImm(0, 0x0000)
	a.setDBR(0)

	// Latch current frame counter as baseline for EmitWaitOneFrame.
	a.movImm(4, 0x803F)
	a.movLoad(2, 4)
	a.movImm(4, wramLastFrame)
	a.movStore(4, 2)

	// Configure floor matrix plane for row mode (plane 0): no built-in projection.
	write8(a, 0x8080, matrixPlane0)
	write8(a, 0x8091, ppucore.MatrixPlaneProjectionNone)
	write8(a, 0x808D, 0x01) // RowModeEnabled

	// Configure billboard matrix plane projection parameters (plane 1).
	write8(a, 0x8080, matrixPlane1)
	write8(a, 0x8091, ppucore.MatrixPlaneProjectionVertical)
	write8(a, 0x8092, billboardHorizon)
	write16(a, 0x809B, billboardBaseDist)
	write16(a, 0x809D, billboardFocal)
	write16(a, 0x809F, billboardWidthScale)
	write16(a, 0x80A1, billboardWorldX)
	write16(a, 0x80A3, billboardWorldY)
	// Billboard faces against +Y in source space (toward the camera start).
	write16s(a, 0x80A5, 0)       // FacingX
	write16s(a, 0x80A7, -0x0100) // FacingY
	write16(a, 0x80A9, billboardHeightScale)

	// Enable matrix-mode sampling for BG0/BG1 and turn on display.
	write8(a, 0x8018, 0x01) // BG0 matrix mode on, wrap outside
	write8(a, 0x802B, 0x01) // BG1 matrix mode on, wrap outside
	write8(a, 0x8011, 0x01) // DISPLAY_CONTROL: enable display

	// Main loop: controller, heading/camera update, then matrix plane updates.
	a.mark("main_loop")
	// Ensure DBR=0 for entire loop: wait (0x803E/0x803F), controller (0xA000), and PPU writes (0x8080+).
	a.movImm(0, 0x0000)
	a.setDBR(0)
	emitWaitOneFrame(a, wramLastFrame)

	// Read controller: latch then read. DBR already 0 from loop head.
	a.movImm(0, 0x0000)
	a.setDBR(0)
	write8(a, 0xA001, 0x01)
	a.movImm(4, 0xA000)
	a.movLoad(2, 4)
	write8(a, 0xA001, 0x00)
	a.movReg(5, 2) // keep buttons in R5

	// Load heading index.
	a.movImm(4, wramHeadingIndex)
	a.movLoad(0, 4)

	// LEFT: increase heading with wrap (camera turns left when you press LEFT).
	noTurnLeft := a.uniq("no_turn_left")
	afterTurnLeft := a.uniq("after_turn_left")
	lookLeftWrapHi := a.uniq("look_left_wrap_hi")
	lookLeftWrapMax := a.uniq("look_left_wrap_max")

	a.movReg(4, 5)
	a.andImm(4, 0x0004)
	a.cmpImm(4, 0)
	a.beq(noTurnLeft)

	a.cmpImm(0, trigSteps-headingTurnStep)
	a.beq(lookLeftWrapHi)
	a.cmpImm(0, trigSteps-1)
	a.beq(lookLeftWrapMax)
	a.addImm(0, headingTurnStep)
	a.jmp(afterTurnLeft)

	a.mark(lookLeftWrapHi)
	a.movImm(0, 0)
	a.jmp(afterTurnLeft)

	a.mark(lookLeftWrapMax)
	a.movImm(0, 1)

	a.mark(afterTurnLeft)
	a.mark(noTurnLeft)

	// RIGHT: decrease heading with wrap (camera turns right when you press RIGHT).
	noTurnRight := a.uniq("no_turn_right")
	afterTurnRight := a.uniq("after_turn_right")
	lookRightWrap0 := a.uniq("look_right_wrap0")
	lookRightWrapSmall := a.uniq("look_right_wrap_small")

	a.movReg(4, 5)
	a.andImm(4, 0x0008)
	a.cmpImm(4, 0)
	a.beq(noTurnRight)

	a.cmpImm(0, 0)
	a.beq(lookRightWrap0)
	a.cmpImm(0, headingTurnStep-1)
	a.beq(lookRightWrapSmall)
	a.subImm(0, headingTurnStep)
	a.jmp(afterTurnRight)

	a.mark(lookRightWrap0)
	a.movImm(0, trigSteps-headingTurnStep)
	a.jmp(afterTurnRight)

	a.mark(lookRightWrapSmall)
	a.movImm(0, trigSteps-1)

	a.mark(afterTurnRight)
	a.mark(noTurnRight)

	// Store updated heading index.
	a.movImm(4, wramHeadingIndex)
	a.movStore(4, 0)

	// Upload floor row table only when heading changed (avoids 3200-byte DMA every frame → FPS).
	a.movImm(4, wramLastFloorHeading)
	a.movLoad(2, 4)
	a.CmpReg(0, 2)
	labelAfterRowTable := a.uniq("after_row_table")
	a.beq(labelAfterRowTable)

	// Heading changed: DMA the row table for current heading (R0 = 0..63).
	rowPhaseLabels := make([]string, 64)
	for i := 0; i < 64; i++ {
		rowPhaseLabels[i] = a.uniq(fmt.Sprintf("row_%02d", i))
	}
	for i := 0; i < 64; i++ {
		a.cmpImm(0, uint16(i))
		a.beq(rowPhaseLabels[i])
	}
	emitMatrixRowDMAChunks(a, matrixPlane0, rowTableRefs[63])
	a.jmp(labelAfterRowTable)
	for i := 0; i < 64; i++ {
		a.mark(rowPhaseLabels[i])
		emitMatrixRowDMAChunks(a, matrixPlane0, rowTableRefs[i])
		if i != 63 {
			a.jmp(labelAfterRowTable)
		}
	}
	a.mark(labelAfterRowTable)
	a.movImm(4, wramLastFloorHeading)
	a.movStore(4, 0)
	write8(a, 0x8080, matrixPlane0)
	write8(a, 0x808D, 0x01) // RowModeEnabled

	// Resolve heading forward vector and per-step deltas. HeadingX/HeadingY
	// go into R1/R2; per-step moveX/moveY go into R3/R6.
	emitLoadHeadingEntry(a, trigTableBase, 0, 1, 2, 3, 6)

	// Load camera in source space; WRAM camera is 16-bit.
	a.movImm(5, wramCameraX)
	a.movLoad(4, 5) // cameraX in R4
	a.movImm(5, wramCameraY)
	a.movLoad(7, 5) // cameraY in R7

	// UP: move forward along heading vector.
	noMoveForward := a.uniq("no_move_forward")
	a.movReg(4, 5)
	a.andImm(4, 0x0001)
	a.cmpImm(4, 0)
	a.beq(noMoveForward)
	// camera += precomputed move delta for this heading.
	a.addReg(4, 3)
	a.addReg(7, 6)
	a.mark(noMoveForward)

	// DOWN: move backward along heading vector.
	noMoveBackward := a.uniq("no_move_backward")
	a.movReg(4, 5)
	a.andImm(4, 0x0002)
	a.cmpImm(4, 0)
	a.beq(noMoveBackward)
	a.subReg(4, 3)
	a.subReg(7, 6)
	a.mark(noMoveBackward)

	// Store updated camera position.
	a.movImm(5, wramCameraX)
	a.movStore(5, 4)
	a.movImm(5, wramCameraY)
	a.movStore(5, 7)

	// Program billboard matrix plane (plane 1) from camera and heading. DBR=0 so 0x8080+ hit PPU.
	// Use scratch regs (6,5) for plane select so R4/R7 (camera) and R1/R2 (heading) are preserved.
	a.movImm(0, 0x0000)
	a.setDBR(0)
	write8Scratch(a, 0x8080, matrixPlane1, 5, 6)
	a.movReg(6, 7)                // Copy camera Y to R6; Write16RegBytes uses R4 or R7 as addr so we'll clobber R7 next
	write16RegBytes(a, 0x8093, 4) // CameraX (clobbers R7)
	write16RegBytes(a, 0x8095, 6) // CameraY
	write16RegBytes(a, 0x8097, 1) // HeadingX
	write16RegBytes(a, 0x8099, 2) // HeadingY

	emitText(a, 8, 8, 0xF8, 0xF8, 0xF8, "MATRIX FLOOR+OBJECT REFERENCE")
	emitText(a, 8, 20, 0xB0, 0xE0, 0xFF, "BG0 FLOOR (ROW MODE)")
	emitText(a, 8, 32, 0xB0, 0xFF, 0xB0, "BG1 BILLBOARD (VERTICAL QUAD)")
	emitText(a, 8, 44, 0xFF, 0xD0, 0x70, "UP/DOWN MOVE  LEFT/RIGHT TURN")

	a.jmp("main_loop")

	if err := a.resolve(); err != nil {
		return err
	}

	// Emit ROM payload: floor bitmap, billboard bitmap, then 64 row tables.
	payload := append([]byte{}, floorAsset.Program.Bitmap...)
	payload = append(payload, billboardAsset.Program.Bitmap...)
	for i := 0; i < 64; i++ {
		payload = append(payload, floorRowTables[i]...)
	}
	if err := romutil.AppendDataBlob(a.B, dataStartBank, payload); err != nil {
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
	floorPath := flag.String("floor", "Resources/kart.png", "floor PNG image")
	billboardPath := flag.String("billboard", "Resources/Test.png", "billboard PNG image")
	outPath := flag.String("out", "roms/matrix_floor_billboard_reference.rom", "output ROM path")
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
	if err := buildMatrixFloorBillboardReferenceROM(floorImg, billboardImg, *outPath); err != nil {
		fmt.Fprintf(os.Stderr, "build ROM: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Built %s using floor=%s billboard=%s\n", *outPath, *floorPath, *billboardPath)
}
