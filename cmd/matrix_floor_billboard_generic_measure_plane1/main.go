package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"

	"nitro-core-dx/internal/emulator"
	ppucore "nitro-core-dx/internal/ppu"
)

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			if i == 0 {
				return "/"
			}
			return path[:i]
		}
	}
	return "."
}

func writePNG(path string, fb []uint32) error {
	img := image.NewNRGBA(image.Rect(0, 0, 320, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			c := fb[y*320+x]
			i := img.PixOffset(x, y)
			img.Pix[i+0] = uint8((c >> 16) & 0xFF)
			img.Pix[i+1] = uint8((c >> 8) & 0xFF)
			img.Pix[i+2] = uint8(c & 0xFF)
			img.Pix[i+3] = 0xFF
		}
	}
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// billboardBottomDistFromBottom computes the max-y of "bright-green-ish" pixels.
// This is a heuristic for the `Resources/Test.png` billboard content.
func billboardBottomDistFromBottom(fb []uint32) (distFromBottom int, yMax int, count int) {
	h := 200
	const (
		w      = 320
		gMin   = 180
		rMax   = 70
		bMax   = 70
		gOverR = 20
	)

	yMax = -1
	count = 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := fb[y*w+x]
			r := int((c >> 16) & 0xFF)
			g := int((c >> 8) & 0xFF)
			b := int(c & 0xFF)
			if g >= gMin && r <= rMax && b <= bMax && (g-r) >= gOverR {
				count++
				if y > yMax {
					yMax = y
				}
			}
		}
	}
	if yMax < 0 {
		return 99999, yMax, 0
	}
	distFromBottom = (h - 1) - yMax
	return distFromBottom, yMax, count
}

const (
	measureScreenHeight = 200
)

func projectMatrixPlanePointYForPlane(plane *ppucore.MatrixPlane, worldX, worldY, worldZ int32) (screenY int32, ok bool) {
	camX := int32(plane.CameraX)
	camY := int32(plane.CameraY)
	forwardX := int32(plane.HeadingX)
	forwardY := int32(plane.HeadingY)
	if forwardX == 0 && forwardY == 0 {
		forwardY = -0x0100
	}
	rightX := -forwardY
	rightY := forwardX

	relX := (worldX << 8) - (camX << 8)
	relY := (worldY << 8) - (camY << 8)
	_ = ((relX * rightX) + (relY * rightY)) >> 8 // lateral; not needed for screenY
	depth := ((relX * forwardX) + (relY * forwardY)) >> 8
	if depth <= 0 {
		return 0, false
	}

	focal := int32(plane.FocalLength)
	if focal <= 0 {
		focal = 1
	}
	cameraHeight := int32(plane.BaseDistance)
	// screenY = Horizon + ((((cameraHeight-worldZ)*focal)/depth)>>8)
	screenY = int32(plane.Horizon) + ((((cameraHeight - worldZ) * focal) / depth) >> 8)
	return screenY, true
}

func computePlane1VerticalQuadBottomDistFromBottom(emu *emulator.Emulator, debug bool) (distFromBottom int, bottomMaxY int32, visible bool) {
	plane := &emu.PPU.MatrixPlanes[1]

	facingX := int32(plane.FacingX)
	facingY := int32(plane.FacingY)
	if facingX == 0 && facingY == 0 {
		facingY = -0x0100
	}
	tangentX := -facingY
	tangentY := facingX
	widthHalf := int32(plane.WidthScale) / 2
	heightScale := int32(plane.HeightScale)
	originX := int32(plane.OriginX)
	originY := int32(plane.OriginY)

	leftBottomX := originX - ((tangentX * widthHalf) >> 8)
	leftBottomY := originY - ((tangentY * widthHalf) >> 8)
	rightBottomX := originX + ((tangentX * widthHalf) >> 8)
	rightBottomY := originY + ((tangentY * widthHalf) >> 8)

	ly0, leftOK := projectMatrixPlanePointYForPlane(plane, leftBottomX, leftBottomY, 0)
	ry0, rightOK := projectMatrixPlanePointYForPlane(plane, rightBottomX, rightBottomY, 0)
	if !leftOK && !rightOK {
		return 99999, 0, false
	}

	// Render code treats depth<=0 as invalid for OK, but still allows
	// rendering if TwoSided=true (using 0 screenY from the projector).
	ly1, _ := projectMatrixPlanePointYForPlane(plane, leftBottomX, leftBottomY, heightScale)
	ry1, _ := projectMatrixPlanePointYForPlane(plane, rightBottomX, rightBottomY, heightScale)

	topMin := ly0
	if ly1 < topMin {
		topMin = ly1
	}
	if ry0 < topMin {
		topMin = ry0
	}
	if ry1 < topMin {
		topMin = ry1
	}

	_ = topMin // not needed; bottomMax is what we want

	bottomMax := ly0
	if ly1 > bottomMax {
		bottomMax = ly1
	}
	if ry0 > bottomMax {
		bottomMax = ry0
	}
	if ry1 > bottomMax {
		bottomMax = ry1
	}

	// Convert to dist-from-bottom in pixels (same coord system as PNG)
	distFromBottom = (measureScreenHeight - 1) - int(bottomMax)

	if debug {
		fmt.Printf("[geo debug] cam=(%d,%d) heading=(%d,%d)\n", plane.CameraX, plane.CameraY, plane.HeadingX, plane.HeadingY)
		fmt.Printf("[geo debug] plane origin=(%d,%d) facing=(0x%04X,0x%04X) widthHalf=%d heightScale=%d baseDist=%d focal=%d horizon=%d\n",
			plane.OriginX, plane.OriginY, uint16(plane.FacingX), uint16(plane.FacingY),
			widthHalf, heightScale, plane.BaseDistance, plane.FocalLength, plane.Horizon)
		fmt.Printf("[geo debug] bottom L=(%d,%d) R=(%d,%d)\n", leftBottomX, leftBottomY, rightBottomX, rightBottomY)
		fmt.Printf("[geo debug] proj worldZ=0  ly0=%d ok=%v ry0=%d ok=%v\n", ly0, leftOK, ry0, rightOK)
		fmt.Printf("[geo debug] proj worldZ=HS ly1=%d ry1=%d\n", ly1, ry1)
		fmt.Printf("[geo debug] bottomMaxY=%d => dist=%dpx\n", bottomMax, distFromBottom)
	}
	return distFromBottom, bottomMax, true
}

func runWithPlaneMask(romData []byte, warmupFrames, baselineFrames, inputFrames int, buttons uint16, plane0Enabled, plane1Enabled bool) (*emulator.Emulator, []uint32) {
	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		panic(err)
	}
	emu.Running = true
	emu.SetFrameLimit(false)

	for i := 0; i < warmupFrames; i++ {
		emu.SetInputButtons(0)
		// ROM code may reconfigure plane control registers; enforce plane enables
		// right before the frame starts so they can't be overwritten mid-run.
		emu.PPU.MatrixPlanes[0].Enabled = plane0Enabled
		emu.PPU.MatrixPlanes[1].Enabled = plane1Enabled
		if err := emu.RunFrame(); err != nil {
			panic(err)
		}
	}
	for i := 0; i < baselineFrames; i++ {
		emu.SetInputButtons(0)
		emu.PPU.MatrixPlanes[0].Enabled = plane0Enabled
		emu.PPU.MatrixPlanes[1].Enabled = plane1Enabled
		if err := emu.RunFrame(); err != nil {
			panic(err)
		}
	}

	for i := 0; i < inputFrames; i++ {
		emu.SetInputButtons(buttons)
		emu.PPU.MatrixPlanes[0].Enabled = plane0Enabled
		emu.PPU.MatrixPlanes[1].Enabled = plane1Enabled
		if err := emu.RunFrame(); err != nil {
			panic(err)
		}
	}
	return emu, emu.GetOutputBuffer()
}

func main() {
	romPath := flag.String("rom", "roms/matrix_floor_billboard_generic.rom", "ROM to run")
	warmupFrames := flag.Int("warmup_frames", 5, "frames to run before baseline capture")
	baselineFrames := flag.Int("baseline_frames", 10, "extra frames with no input before second capture")
	inputFrames := flag.Int("input_frames", 20, "frames to hold input before second capture")
	buttons := flag.Uint("buttons", 0x0001, "controller buttons mask for second phase (default: UP)")

	outPlane1OnlyPNG := flag.String("plane1_only_after", ".tmp_matrixfloor_generic_measure/plane1_only_after.png", "optional PNG output (plane0 disabled, plane1 enabled)")
	measureBaseline := flag.Bool("measure_baseline", false, "also measure baseline (inputFrames=0 run)")
	debugGeo := flag.Bool("debug_geo", false, "print geometric projection debug")

	flag.Parse()

	romData, err := os.ReadFile(*romPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read ROM: %v\n", err)
		os.Exit(1)
	}

	// We care about the "after input" state where user is actively driving.
	emuAfter, fbAfter := runWithPlaneMask(romData, *warmupFrames, *baselineFrames, *inputFrames, uint16(*buttons), false, true)
	dist, yMax, cnt := billboardBottomDistFromBottom(fbAfter)
	geoDist, geoBottomY, geoVisible := computePlane1VerticalQuadBottomDistFromBottom(emuAfter, *debugGeo)

	if err := writePNG(*outPlane1OnlyPNG, fbAfter); err != nil {
		fmt.Fprintf(os.Stderr, "write plane1-only PNG: %v\n", err)
	}

	fmt.Printf("Plane1-only billboard bottom (after input)\n")
	fmt.Printf("  WRAM camY=%d camX=%d heading=%d\n",
		int16(emuAfter.Bus.Read16(0, 0x0206)), int16(emuAfter.Bus.Read16(0, 0x0204)), emuAfter.Bus.Read16(0, 0x0202))
	fmt.Printf("  dist_from_bottom=%dpx (yMax=%d, count=%d) png=%s\n\n", dist, yMax, cnt, *outPlane1OnlyPNG)
	fmt.Printf("Plane1 vertical-quad geometric bottom (after input)\n")
	fmt.Printf("  visible=%v geo_bottomMaxY=%d => geo_dist_from_bottom=%dpx\n\n", geoVisible, geoBottomY, geoDist)

	if !*measureBaseline {
		return
	}

	// Baseline: run again but with inputFrames=0.
	emuBase, fbBase := runWithPlaneMask(romData, *warmupFrames, *baselineFrames, 0, 0, false, true)
	distB, yMaxB, cntB := billboardBottomDistFromBottom(fbBase)
	geoDistB, geoBottomYB, geoVisibleB := computePlane1VerticalQuadBottomDistFromBottom(emuBase, *debugGeo)
	basePNG := ".tmp_matrixfloor_generic_measure/plane1_only_before.png"
	_ = writePNG(basePNG, fbBase)

	fmt.Printf("Plane1-only billboard bottom (baseline before input)\n")
	fmt.Printf("  WRAM camY=%d camX=%d heading=%d\n",
		int16(emuBase.Bus.Read16(0, 0x0206)), int16(emuBase.Bus.Read16(0, 0x0204)), emuBase.Bus.Read16(0, 0x0202))
	fmt.Printf("  dist_from_bottom=%dpx (yMax=%d, count=%d) png=%s\n", distB, yMaxB, cntB, basePNG)

	fmt.Printf("Plane1 vertical-quad geometric bottom (baseline before input)\n")
	fmt.Printf("  visible=%v geo_bottomMaxY=%d => geo_dist_from_bottom=%dpx\n", geoVisibleB, geoBottomYB, geoDistB)
}
