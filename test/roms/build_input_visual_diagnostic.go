//go:build testrom_tools
// +build testrom_tools

package main

import (
	"fmt"
	"os"

	"nitro-core-dx/internal/rom"
)

type patchRef struct {
	wordIndex int
	currentPC uint16 // address of branch/jump offset word
	target    string
}

type asm struct {
	b       *rom.ROMBuilder
	labels  map[string]uint16
	patches []patchRef
}

func newASM() *asm {
	return &asm{
		b:      rom.NewROMBuilder(),
		labels: make(map[string]uint16),
	}
}

func (a *asm) pc() uint16       { return uint16(a.b.GetCodeLength() * 2) }
func (a *asm) mark(name string) { a.labels[name] = a.pc() }
func (a *asm) inst(w uint16)    { a.b.AddInstruction(w) }
func (a *asm) imm(v uint16)     { a.b.AddImmediate(v) }

func (a *asm) movImm(reg uint8, v uint16)  { a.inst(rom.EncodeMOV(1, reg, 0)); a.imm(v) }
func (a *asm) movReg(dst, src uint8)       { a.inst(rom.EncodeMOV(0, dst, src)) }
func (a *asm) movLoad(dst, addrReg uint8)  { a.inst(rom.EncodeMOV(2, dst, addrReg)) }
func (a *asm) movStore(addrReg, src uint8) { a.inst(rom.EncodeMOV(3, addrReg, src)) }
func (a *asm) addReg(dst, src uint8)       { a.inst(rom.EncodeADD(0, dst, src)) }
func (a *asm) addImm(reg uint8, v uint16)  { a.inst(rom.EncodeADD(1, reg, 0)); a.imm(v) }
func (a *asm) subImm(reg uint8, v uint16)  { a.inst(rom.EncodeSUB(1, reg, 0)); a.imm(v) }
func (a *asm) andImm(reg uint8, v uint16)  { a.inst(rom.EncodeAND(1, reg, 0)); a.imm(v) }
func (a *asm) cmpImm(reg uint8, v uint16)  { a.inst(rom.EncodeCMP(1, reg, 0)); a.imm(v) }
func (a *asm) cmpReg(r1, r2 uint8)         { a.inst(rom.EncodeCMP(0, r1, r2)) }
func (a *asm) shrImm(reg uint8, v uint16)  { a.inst(rom.EncodeSHR(1, reg, 0)); a.imm(v) }

func (a *asm) branch(op uint16, label string) {
	a.inst(op)
	pc := a.pc()
	a.imm(0)
	a.patches = append(a.patches, patchRef{
		wordIndex: a.b.GetCodeLength() - 1,
		currentPC: pc,
		target:    label,
	})
}
func (a *asm) beq(label string) { a.branch(rom.EncodeBEQ(), label) }
func (a *asm) bne(label string) { a.branch(rom.EncodeBNE(), label) }
func (a *asm) bgt(label string) { a.branch(rom.EncodeBGT(), label) }
func (a *asm) blt(label string) { a.branch(rom.EncodeBLT(), label) }
func (a *asm) bge(label string) { a.branch(rom.EncodeBGE(), label) }
func (a *asm) ble(label string) { a.branch(rom.EncodeBLE(), label) }
func (a *asm) jmp(label string) { a.branch(rom.EncodeJMP(), label) }

func (a *asm) resolve() error {
	for _, p := range a.patches {
		targetPC, ok := a.labels[p.target]
		if !ok {
			return fmt.Errorf("unknown label %q", p.target)
		}
		a.b.SetImmediateAt(p.wordIndex, uint16(rom.CalculateBranchOffset(p.currentPC, targetPC)))
	}
	return nil
}

func setVRAMAddr(a *asm, addr uint16) {
	a.movImm(4, 0x800E) // VRAM_ADDR_L
	a.movImm(5, addr&0xFF)
	a.movStore(4, 5)
	a.movImm(4, 0x800F) // VRAM_ADDR_H
	a.movImm(5, (addr>>8)&0xFF)
	a.movStore(4, 5)
}

func setCGRAMColor(a *asm, colorIndex uint8, rgb555 uint16) {
	a.movImm(4, 0x8012) // CGRAM_ADDR
	a.movImm(5, uint16(colorIndex))
	a.movStore(4, 5)
	a.movImm(4, 0x8013) // CGRAM_DATA
	a.movImm(5, rgb555&0xFF)
	a.movStore(4, 5)
	a.movImm(5, (rgb555>>8)&0xFF)
	a.movStore(4, 5)
}

func setAPUCh0Freq(a *asm, freq uint16) {
	a.movImm(4, 0x9000) // CH0 FREQ_LOW
	a.movImm(5, freq&0xFF)
	a.movStore(4, 5)
	a.movImm(4, 0x9001) // CH0 FREQ_HIGH
	a.movImm(5, (freq>>8)&0xFF)
	a.movStore(4, 5)
}

func setAPUCh2Freq(a *asm, freq uint16) {
	a.movImm(4, 0x9010) // CH2 FREQ_LOW
	a.movImm(5, freq&0xFF)
	a.movStore(4, 5)
	a.movImm(4, 0x9011) // CH2 FREQ_HIGH
	a.movImm(5, (freq>>8)&0xFF)
	a.movStore(4, 5)
}

func setAPUCh1Freq(a *asm, freq uint16) {
	a.movImm(4, 0x9008) // CH1 FREQ_LOW
	a.movImm(5, freq&0xFF)
	a.movStore(4, 5)
	a.movImm(4, 0x9009) // CH1 FREQ_HIGH
	a.movImm(5, (freq>>8)&0xFF)
	a.movStore(4, 5)
}

func setAPUCh3Freq(a *asm, freq uint16) {
	a.movImm(4, 0x9018) // CH3 FREQ_LOW
	a.movImm(5, freq&0xFF)
	a.movStore(4, 5)
	a.movImm(4, 0x9019) // CH3 FREQ_HIGH
	a.movImm(5, (freq>>8)&0xFF)
	a.movStore(4, 5)
}

func writeFMHost(a *asm, hostReg uint16, v uint8) {
	a.movImm(4, hostReg)
	a.movImm(5, uint16(v))
	a.movStore(4, 5)
}

func writeFMOPMReg(a *asm, regAddr, value uint8) {
	writeFMHost(a, 0x9100, regAddr) // FM_ADDR
	writeFMHost(a, 0x9101, value)   // FM_DATA
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run -tags testrom_tools ./test/roms/build_input_visual_diagnostic.go <output.rom>")
		os.Exit(1)
	}

	out := os.Args[1]
	a := newASM()

	fmt.Println("Building input visual diagnostic ROM (v2)...")
	fmt.Println("Controls:")
	fmt.Println("  Arrows/WASD: move sprite (acceleration/friction)")
	fmt.Println("  Z (A): change background color")
	fmt.Println("  X (B): change sprite color")
	fmt.Println("  C (Y): reset sprite position")
	fmt.Println("  Q (L): trigger \"bewp\" jump SFX (pitch sweep)")
	fmt.Println("  E (R): toggle Hall-of-the-Mountain-King-style layered loop")
	fmt.Println("  Enter (START): legacy tone on CH0")
	fmt.Println("  Backspace (Z button): stop all tones + music")

	const (
		wramLastFrame = 0x0010
		wramVelX      = 0x0012
		wramVelY      = 0x0014
		wramPrevHigh  = 0x0016
		wramMusicOn   = 0x0018
		wramMusicStep = 0x001A
		wramMusicTick = 0x001C
		wramSfxTick   = 0x001E
		accelStep     = 0x0001 // 1 px/frame²
		frictionStep  = 0x0001 // 1 px/frame²
		maxVel        = 0x0003 // 3 px/frame
		negMaxVel     = 0xFFFD // -3 px/frame (two's complement)
		xMinPos       = 0x0000
		xMaxPos       = 248
		yMinPos       = 0x0000
		yMaxPos       = 192
	)

	// Registers:
	// R0 = sprite X (integer pixels)
	// R1 = sprite Y (integer pixels)
	// R2 = input low byte  (buttons 0..7)
	// R3 = input high byte (buttons 8..15)
	// R4 = IO address scratch
	// R5 = data scratch
	// R6 = velocity X (integer px/frame, signed)
	// R7 = velocity Y (integer px/frame, signed)

	// ---- Palette init ----
	// BG palette0 color0 (gray default)
	setCGRAMColor(a, 0x00, 0x4210)
	// Sprite palette1 color1 (white default)
	setCGRAMColor(a, 0x11, 0x7FFF)

	// ---- Tile data init ----
	// Tile 0 (background): solid color index 0
	setVRAMAddr(a, 0x0000)
	a.movImm(6, 32)
	a.movImm(4, 0x8010) // VRAM_DATA
	a.movImm(5, 0x00)
	a.mark("fill_tile0")
	a.movStore(4, 5)
	a.subImm(6, 1)
	a.movImm(7, 0)
	a.cmpReg(6, 7)
	a.bne("fill_tile0")

	// Tile 1 (sprite): solid color index 1
	setVRAMAddr(a, 0x0020)
	a.movImm(6, 32)
	a.movImm(4, 0x8010)
	a.movImm(5, 0x11)
	a.mark("fill_tile1")
	a.movStore(4, 5)
	a.subImm(6, 1)
	a.movImm(7, 0)
	a.cmpReg(6, 7)
	a.bne("fill_tile1")

	// ---- BG0 tilemap init (fill visible 32x25 area with tile 0) ----
	setVRAMAddr(a, 0x4000)
	a.movImm(6, 800)    // 32 * 25 tile entries
	a.movImm(4, 0x8010) // VRAM_DATA
	a.mark("fill_tilemap")
	a.movImm(5, 0x00) // tile index 0
	a.movStore(4, 5)
	a.movImm(5, 0x00) // attrs palette 0
	a.movStore(4, 5)
	a.subImm(6, 1)
	a.movImm(7, 0)
	a.cmpReg(6, 7)
	a.bne("fill_tilemap")

	// Enable BG0
	a.movImm(4, 0x8008)
	a.movImm(5, 0x01)
	a.movStore(4, 5)

	// ---- Audio init (channel 0 square, disabled) ----
	a.movImm(4, 0x9020) // MASTER_VOLUME
	a.movImm(5, 0xC0)
	a.movStore(4, 5)
	a.movImm(4, 0x9002) // CH0 VOLUME
	a.movImm(5, 0x80)
	a.movStore(4, 5)
	a.movImm(4, 0x9003) // CH0 CONTROL (disabled, square waveform bits set)
	a.movImm(5, 0x02)
	a.movStore(4, 5)
	setAPUCh0Freq(a, 440)
	// CH2 bass/music channel (square, disabled)
	a.movImm(4, 0x9012) // CH2 VOLUME
	a.movImm(5, 0x70)
	a.movStore(4, 5)
	a.movImm(4, 0x9013) // CH2 CONTROL (disabled, square waveform bits set)
	a.movImm(5, 0x02)
	a.movStore(4, 5)
	setAPUCh2Freq(a, 523)
	// CH1 harmony channel (sine, disabled)
	a.movImm(4, 0x900A) // CH1 VOLUME
	a.movImm(5, 0x60)
	a.movStore(4, 5)
	a.movImm(4, 0x900B) // CH1 CONTROL (disabled, sine waveform)
	a.movImm(5, 0x00)
	a.movStore(4, 5)
	setAPUCh1Freq(a, 330)
	// CH3 SFX/percussion (noise capable, disabled)
	a.movImm(4, 0x901A) // CH3 VOLUME
	a.movImm(5, 0x70)
	a.movStore(4, 5)
	a.movImm(4, 0x901B) // CH3 CONTROL (disabled, square/noise config bit retained as noise)
	a.movImm(5, 0x02)
	a.movStore(4, 5)
	setAPUCh3Freq(a, 1400)
	// FM extension host block enabled (no IRQ enable in this ROM path)
	writeFMHost(a, 0x9103, 0x81) // reset + enable (reset bit is one-shot)
	writeFMHost(a, 0x9103, 0x01) // enable

	// ---- Initial sprite position/state ----
	a.movImm(0, 120) // X
	a.movImm(1, 96)  // Y
	a.movImm(6, 0)   // VX
	a.movImm(7, 0)   // VY
	a.movImm(4, wramVelX)
	a.movStore(4, 6)
	a.movImm(4, wramVelY)
	a.movStore(4, 7)
	a.movImm(4, wramPrevHigh)
	a.movImm(5, 0)
	a.movStore(4, 5)
	a.movImm(4, wramMusicOn)
	a.movStore(4, 5)
	a.movImm(4, wramMusicStep)
	a.movStore(4, 5)
	a.movImm(4, wramMusicTick)
	a.movStore(4, 5)
	a.movImm(4, wramSfxTick)
	a.movStore(4, 5)
	// Capture current frame counter low byte as sync baseline (stored in WRAM)
	a.movImm(4, 0x803F) // FRAME_COUNTER_LOW
	a.movLoad(2, 4)
	a.movImm(4, wramLastFrame)
	a.movStore(4, 2)

	// ---- Main loop ----
	a.mark("main_loop")

	// Wait for next frame counter tick (one logic update per emulated frame)
	a.mark("wait_next_frame")
	a.movImm(4, 0x803F) // FRAME_COUNTER_LOW
	a.movLoad(2, 4)     // current frame counter
	a.movImm(4, wramLastFrame)
	a.movLoad(3, 4) // previous frame counter
	a.cmpReg(2, 3)
	a.bne("frame_start")
	a.jmp("wait_next_frame")

	a.mark("frame_start")
	a.movImm(4, wramLastFrame)
	a.movStore(4, 2) // remember current frame counter low byte
	// Load persisted velocities (R7 gets reused later in the frame for scratch constants)
	a.movImm(4, wramVelX)
	a.movLoad(6, 4)
	a.movImm(4, wramVelY)
	a.movLoad(7, 4)

	// Latch input, read low/high bytes, release latch
	a.movImm(4, 0xA001)
	a.movImm(5, 0x01)
	a.movStore(4, 5)
	a.movImm(4, 0xA000)
	a.movLoad(2, 4) // low byte
	a.movImm(4, 0xA001)
	a.movLoad(3, 4) // high byte (read)
	a.movImm(5, 0x00)
	a.movStore(4, 5) // release latch

	// Reset position on Y button (bit 7 in low byte) - key C in UI mapping
	a.movReg(5, 2)
	a.andImm(5, 0x0080)
	a.movImm(4, 0x0080)
	a.cmpReg(5, 4)
	a.bne("skip_reset")
	a.movImm(0, 120)
	a.movImm(1, 96)
	a.movImm(6, 0)
	a.movImm(7, 0)
	a.mark("skip_reset")

	// ---- Horizontal velocity: acceleration / friction ----
	// LEFT (bit 2)
	a.movReg(5, 2)
	a.andImm(5, 0x0004)
	a.movImm(4, 0x0004)
	a.cmpReg(5, 4)
	a.beq("h_accel_left")
	// RIGHT (bit 3)
	a.movReg(5, 2)
	a.andImm(5, 0x0008)
	a.movImm(4, 0x0008)
	a.cmpReg(5, 4)
	a.beq("h_accel_right")
	a.jmp("h_friction")

	a.mark("h_accel_left")
	a.subImm(6, accelStep)
	a.jmp("h_clamp")

	a.mark("h_accel_right")
	a.addImm(6, accelStep)
	a.jmp("h_clamp")

	a.mark("h_friction")
	a.movImm(5, 0)
	a.cmpReg(6, 5)
	a.beq("h_clamp")
	a.bgt("h_friction_pos")
	// vx < 0
	a.addImm(6, frictionStep)
	a.movImm(5, 0)
	a.cmpReg(6, 5)
	a.ble("h_clamp")
	a.movImm(6, 0)
	a.jmp("h_clamp")
	a.mark("h_friction_pos")
	a.subImm(6, frictionStep)
	a.movImm(5, 0)
	a.cmpReg(6, 5)
	a.bge("h_clamp")
	a.movImm(6, 0)

	a.mark("h_clamp")
	a.movImm(5, maxVel)
	a.cmpReg(6, 5)
	a.ble("h_clamp_neg")
	a.movImm(6, maxVel)
	a.mark("h_clamp_neg")
	a.movImm(5, negMaxVel)
	a.cmpReg(6, 5)
	a.bge("h_done")
	a.movImm(6, negMaxVel)
	a.mark("h_done")

	// ---- Vertical velocity: acceleration / friction ----
	// UP (bit 0)
	a.movReg(5, 2)
	a.andImm(5, 0x0001)
	a.movImm(4, 0x0001)
	a.cmpReg(5, 4)
	a.beq("v_accel_up")
	// DOWN (bit 1)
	a.movReg(5, 2)
	a.andImm(5, 0x0002)
	a.movImm(4, 0x0002)
	a.cmpReg(5, 4)
	a.beq("v_accel_down")
	a.jmp("v_friction")

	a.mark("v_accel_up")
	a.subImm(7, accelStep)
	a.jmp("v_clamp")

	a.mark("v_accel_down")
	a.addImm(7, accelStep)
	a.jmp("v_clamp")

	a.mark("v_friction")
	a.movImm(5, 0)
	a.cmpReg(7, 5)
	a.beq("v_clamp")
	a.bgt("v_friction_pos")
	// vy < 0
	a.addImm(7, frictionStep)
	a.movImm(5, 0)
	a.cmpReg(7, 5)
	a.ble("v_clamp")
	a.movImm(7, 0)
	a.jmp("v_clamp")
	a.mark("v_friction_pos")
	a.subImm(7, frictionStep)
	a.movImm(5, 0)
	a.cmpReg(7, 5)
	a.bge("v_clamp")
	a.movImm(7, 0)

	a.mark("v_clamp")
	a.movImm(5, maxVel)
	a.cmpReg(7, 5)
	a.ble("v_clamp_neg")
	a.movImm(7, maxVel)
	a.mark("v_clamp_neg")
	a.movImm(5, negMaxVel)
	a.cmpReg(7, 5)
	a.bge("v_done")
	a.movImm(7, negMaxVel)
	a.mark("v_done")

	// Apply velocities (integer positions)
	a.addReg(0, 6)
	a.addReg(1, 7)

	// Clamp X to [0, 248]
	a.movImm(5, xMinPos)
	a.cmpReg(0, 5)
	a.blt("clamp_x_min")
	a.movImm(4, xMaxPos)
	a.cmpReg(0, 4)
	a.bgt("clamp_x_max")
	a.jmp("x_clamped")
	a.mark("clamp_x_min")
	a.movImm(0, xMinPos)
	a.movImm(6, 0)
	a.jmp("x_clamped")
	a.mark("clamp_x_max")
	a.movImm(0, xMaxPos)
	a.movImm(6, 0)
	a.mark("x_clamped")

	// Clamp Y to [0, 192]
	a.movImm(5, yMinPos)
	a.cmpReg(1, 5)
	a.blt("clamp_y_min")
	a.movImm(4, yMaxPos)
	a.cmpReg(1, 4)
	a.bgt("clamp_y_max")
	a.jmp("y_clamped")
	a.mark("clamp_y_min")
	a.movImm(1, yMinPos)
	a.movImm(7, 0)
	a.jmp("y_clamped")
	a.mark("clamp_y_max")
	a.movImm(1, yMaxPos)
	a.movImm(7, 0)
	a.mark("y_clamped")

	// Persist velocities before reusing R7 as a general-purpose scratch register below.
	a.movImm(4, wramVelX)
	a.movStore(4, 6)
	a.movImm(4, wramVelY)
	a.movStore(4, 7)

	// Background color: A button (bit 4, key Z) toggles cyan while held, else gray
	setCGRAMColor(a, 0x00, 0x4210) // default gray
	a.movReg(5, 2)
	a.andImm(5, 0x0010)
	a.movImm(7, 0x0010)
	a.cmpReg(5, 7)
	a.bne("skip_bg_color")
	setCGRAMColor(a, 0x00, 0x03FF) // cyan
	a.mark("skip_bg_color")

	// Sprite color: B button (bit 5, key X) toggles green while held, else white
	setCGRAMColor(a, 0x11, 0x7FFF) // default white
	a.movReg(5, 2)
	a.andImm(5, 0x0020)
	a.movImm(7, 0x0020)
	a.cmpReg(5, 7)
	a.bne("skip_sprite_color")
	setCGRAMColor(a, 0x11, 0x03E0) // green
	a.mark("skip_sprite_color")

	// Default state for manual CH0 tone every frame (START can re-enable below).
	a.movImm(4, 0x9003) // CH0 CONTROL
	a.movImm(5, 0x02)   // disabled, square waveform config retained
	a.movStore(4, 5)

	// ---- Edge-detected audio controls on high-byte buttons ----
	// Load previous high-byte input snapshot from WRAM into R6.
	a.movImm(4, wramPrevHigh)
	a.movLoad(6, 4)

	// L (bit0) rising edge => trigger short "bewp" SFX state on CH0 (pitch sweep)
	a.movReg(5, 3)
	a.andImm(5, 0x0001)
	a.movImm(7, 0x0001)
	a.cmpReg(5, 7)
	a.bne("skip_l_edge")
	a.movReg(5, 6)
	a.andImm(5, 0x0001)
	a.movImm(7, 0x0000)
	a.cmpReg(5, 7)
	a.bne("skip_l_edge")
	// 4-frame sweep envelope: low -> high -> peak -> drop
	a.movImm(4, wramSfxTick)
	a.movImm(5, 0x0004)
	a.movStore(4, 5)
	a.mark("skip_l_edge")

	// R (bit1) rising edge => toggle layered music loop on/off
	a.movReg(5, 3)
	a.andImm(5, 0x0002)
	a.movImm(7, 0x0002)
	a.cmpReg(5, 7)
	a.bne("skip_r_edge")
	a.movReg(5, 6)
	a.andImm(5, 0x0002)
	a.movImm(7, 0x0000)
	a.cmpReg(5, 7)
	a.bne("skip_r_edge")
	// Toggle wramMusicOn
	a.movImm(4, wramMusicOn)
	a.movLoad(5, 4)
	a.movImm(7, 0x0000)
	a.cmpReg(5, 7)
	a.bne("music_toggle_off")
	// Turn music on and force immediate step update
	a.movImm(5, 0x0001)
	a.movStore(4, 5)
	a.movImm(4, wramMusicStep)
	a.movImm(5, 0x0000)
	a.movStore(4, 5)
	a.movImm(4, wramMusicTick)
	a.movStore(4, 5)
	a.jmp("skip_r_edge")
	a.mark("music_toggle_off")
	a.movImm(5, 0x0000)
	a.movStore(4, 5) // wramMusicOn = 0
	// Disable music channels CH1/CH2 (CH0 left to manual START path)
	a.movImm(4, 0x900B)
	a.movImm(5, 0x00)
	a.movStore(4, 5)
	a.movImm(4, 0x9013)
	a.movImm(5, 0x02)
	a.movStore(4, 5)
	a.mark("skip_r_edge")

	// Store current high-byte input snapshot for next frame edge detection
	a.movImm(4, wramPrevHigh)
	a.movStore(4, 3)

	// ---- One-shot SFX engine (non-blocking, runs while moving/music) ----
	// Uses CH0 square waveform and a short 4-frame pitch sweep for a "bewp" jump sound.
	a.movImm(4, wramSfxTick)
	a.movLoad(5, 4)
	a.movImm(7, 0x0000)
	a.cmpReg(5, 7)
	a.beq("skip_sfx_engine")

	// Select frequency based on remaining sweep step count (R5 = 4..1)
	a.movImm(7, 0x0004)
	a.cmpReg(5, 7)
	a.beq("sfx_step4")
	a.movImm(7, 0x0003)
	a.cmpReg(5, 7)
	a.beq("sfx_step3")
	a.movImm(7, 0x0002)
	a.cmpReg(5, 7)
	a.beq("sfx_step2")
	a.jmp("sfx_step1")

	a.mark("sfx_step4")
	setAPUCh0Freq(a, 220) // low start
	a.jmp("sfx_apply")
	a.mark("sfx_step3")
	setAPUCh0Freq(a, 392) // rise
	a.jmp("sfx_apply")
	a.mark("sfx_step2")
	setAPUCh0Freq(a, 587) // quick peak
	a.jmp("sfx_apply")
	a.mark("sfx_step1")
	setAPUCh0Freq(a, 294) // drop

	a.mark("sfx_apply")
	a.movImm(4, 0x9002) // CH0 VOLUME
	a.movImm(5, 0xC0)
	a.movStore(4, 5)
	a.movImm(4, 0x9003) // CH0 CONTROL = enable + square
	a.movImm(5, 0x03)
	a.movStore(4, 5)
	// Decrement sweep tick
	a.movImm(4, wramSfxTick)
	a.movLoad(5, 4)
	a.subImm(5, 1)
	a.movStore(4, 5)
	a.mark("skip_sfx_engine")

	// ---- Music loop engine (non-blocking, runs while moving) ----
	a.movImm(4, wramMusicOn)
	a.movLoad(5, 4)
	a.movImm(7, 0x0000)
	a.cmpReg(5, 7)
	a.beq("skip_music_engine")

	// FM MMIO/timer activity while music is enabled (diagnostic traffic)
	writeFMOPMReg(a, 0x10, 0xFF) // Timer A high
	writeFMOPMReg(a, 0x11, 0x03) // Timer A low
	writeFMOPMReg(a, 0x28, 0x60) // arbitrary note-ish shadow
	writeFMOPMReg(a, 0x14, 0x05) // start A + clear A flag

	// Countdown ticks until next music step
	a.movImm(4, wramMusicTick)
	a.movLoad(5, 4)
	a.movImm(7, 0x0000)
	a.cmpReg(5, 7)
	a.beq("music_step_update")
	a.subImm(5, 1)
	a.movStore(4, 5)
	a.jmp("skip_music_engine")

	a.mark("music_step_update")
	// Load current music step index into R5
	a.movImm(4, wramMusicStep)
	a.movLoad(5, 4)
	a.movReg(6, 5) // preserve step index for percussion trigger checks
	a.movImm(7, 0x0000)
	a.cmpReg(5, 7)
	a.beq("music_step0")
	a.movImm(7, 0x0001)
	a.cmpReg(5, 7)
	a.beq("music_step1")
	a.movImm(7, 0x0002)
	a.cmpReg(5, 7)
	a.beq("music_step2")
	a.movImm(7, 0x0003)
	a.cmpReg(5, 7)
	a.beq("music_step3")
	a.movImm(7, 0x0004)
	a.cmpReg(5, 7)
	a.beq("music_step4")
	a.movImm(7, 0x0005)
	a.cmpReg(5, 7)
	a.beq("music_step5")
	a.movImm(7, 0x0006)
	a.cmpReg(5, 7)
	a.beq("music_step6")
	a.movImm(7, 0x0007)
	a.cmpReg(5, 7)
	a.beq("music_step7")
	a.movImm(7, 0x0008)
	a.cmpReg(5, 7)
	a.beq("music_step8")
	a.movImm(7, 0x0009)
	a.cmpReg(5, 7)
	a.beq("music_step9")
	a.movImm(7, 0x000A)
	a.cmpReg(5, 7)
	a.beq("music_step10")
	a.movImm(7, 0x000B)
	a.cmpReg(5, 7)
	a.beq("music_step11")
	a.movImm(7, 0x000C)
	a.cmpReg(5, 7)
	a.beq("music_step12")
	a.movImm(7, 0x000D)
	a.cmpReg(5, 7)
	a.beq("music_step13")
	a.movImm(7, 0x000E)
	a.cmpReg(5, 7)
	a.beq("music_step14")
	a.movImm(7, 0x000F)
	a.cmpReg(5, 7)
	a.beq("music_step15")
	a.movImm(7, 0x0010)
	a.cmpReg(5, 7)
	a.beq("music_step16")
	a.movImm(7, 0x0011)
	a.cmpReg(5, 7)
	a.beq("music_step17")
	a.movImm(7, 0x0012)
	a.cmpReg(5, 7)
	a.beq("music_step18")
	a.movImm(7, 0x0013)
	a.cmpReg(5, 7)
	a.beq("music_step19")
	a.movImm(7, 0x0014)
	a.cmpReg(5, 7)
	a.beq("music_step20")
	a.movImm(7, 0x0015)
	a.cmpReg(5, 7)
	a.beq("music_step21")
	a.movImm(7, 0x0016)
	a.cmpReg(5, 7)
	a.beq("music_step22")
	a.movImm(7, 0x0017)
	a.cmpReg(5, 7)
	a.beq("music_step23")
	a.movImm(7, 0x0018)
	a.cmpReg(5, 7)
	a.beq("music_step24")
	a.movImm(7, 0x0019)
	a.cmpReg(5, 7)
	a.beq("music_step25")
	a.movImm(7, 0x001A)
	a.cmpReg(5, 7)
	a.beq("music_step26")
	a.movImm(7, 0x001B)
	a.cmpReg(5, 7)
	a.beq("music_step27")
	a.movImm(7, 0x001C)
	a.cmpReg(5, 7)
	a.beq("music_step28")
	a.movImm(7, 0x001D)
	a.cmpReg(5, 7)
	a.beq("music_step29")
	a.movImm(7, 0x001E)
	a.cmpReg(5, 7)
	a.beq("music_step30")
	a.jmp("music_step31")

	// "In the Hall of the Mountain King" opening phrase (32-step loop).
	// Baked from a MIDI/sheet-derived note sequence: CH1 melody, CH2 bass ostinato.
	// Each step is an eighth-note-style trigger in a simplified grid.
	a.mark("music_step0")
	setAPUCh1Freq(a, 247) // B3
	setAPUCh2Freq(a, 123) // B2
	a.jmp("music_apply_common")
	a.mark("music_step1")
	setAPUCh1Freq(a, 277) // C#4
	setAPUCh2Freq(a, 123) // B2
	a.jmp("music_apply_common")
	a.mark("music_step2")
	setAPUCh1Freq(a, 294) // D4
	setAPUCh2Freq(a, 185) // F#3
	a.jmp("music_apply_common")
	a.mark("music_step3")
	setAPUCh1Freq(a, 330) // E4
	setAPUCh2Freq(a, 185) // F#3
	a.jmp("music_apply_common")
	a.mark("music_step4")
	setAPUCh1Freq(a, 370) // F#4
	setAPUCh2Freq(a, 123) // B2
	a.jmp("music_apply_common")
	a.mark("music_step5")
	setAPUCh1Freq(a, 294) // D4
	setAPUCh2Freq(a, 123) // B2
	a.jmp("music_apply_common")
	a.mark("music_step6")
	setAPUCh1Freq(a, 370) // F#4
	setAPUCh2Freq(a, 185) // F#3
	a.jmp("music_apply_common")
	a.mark("music_step7")
	setAPUCh1Freq(a, 370) // F#4 (held)
	setAPUCh2Freq(a, 185) // F#3
	a.jmp("music_apply_common")
	a.mark("music_step8")
	setAPUCh1Freq(a, 349) // F4 (E#4)
	setAPUCh2Freq(a, 123) // B2
	a.jmp("music_apply_common")
	a.mark("music_step9")
	setAPUCh1Freq(a, 277) // C#4
	setAPUCh2Freq(a, 123) // B2
	a.jmp("music_apply_common")
	a.mark("music_step10")
	setAPUCh1Freq(a, 349) // F4
	setAPUCh2Freq(a, 185) // F#3
	a.jmp("music_apply_common")
	a.mark("music_step11")
	setAPUCh1Freq(a, 349) // F4 (held)
	setAPUCh2Freq(a, 185) // F#3
	a.jmp("music_apply_common")
	a.mark("music_step12")
	setAPUCh1Freq(a, 330) // E4
	setAPUCh2Freq(a, 123) // B2
	a.jmp("music_apply_common")
	a.mark("music_step13")
	setAPUCh1Freq(a, 262) // C4
	setAPUCh2Freq(a, 123) // B2
	a.jmp("music_apply_common")
	a.mark("music_step14")
	setAPUCh1Freq(a, 330) // E4
	setAPUCh2Freq(a, 185) // F#3
	a.jmp("music_apply_common")
	a.mark("music_step15")
	setAPUCh1Freq(a, 330) // E4 (held)
	setAPUCh2Freq(a, 185) // F#3
	a.jmp("music_apply_common")
	a.mark("music_step16")
	setAPUCh1Freq(a, 247) // B3
	setAPUCh2Freq(a, 123) // B2
	a.jmp("music_apply_common")
	a.mark("music_step17")
	setAPUCh1Freq(a, 277) // C#4
	setAPUCh2Freq(a, 123) // B2
	a.jmp("music_apply_common")
	a.mark("music_step18")
	setAPUCh1Freq(a, 294) // D4
	setAPUCh2Freq(a, 185) // F#3
	a.jmp("music_apply_common")
	a.mark("music_step19")
	setAPUCh1Freq(a, 330) // E4
	setAPUCh2Freq(a, 185) // F#3
	a.jmp("music_apply_common")
	a.mark("music_step20")
	setAPUCh1Freq(a, 370) // F#4
	setAPUCh2Freq(a, 123) // B2
	a.jmp("music_apply_common")
	a.mark("music_step21")
	setAPUCh1Freq(a, 294) // D4
	setAPUCh2Freq(a, 123) // B2
	a.jmp("music_apply_common")
	a.mark("music_step22")
	setAPUCh1Freq(a, 370) // F#4
	setAPUCh2Freq(a, 185) // F#3
	a.jmp("music_apply_common")
	a.mark("music_step23")
	setAPUCh1Freq(a, 494) // B4
	setAPUCh2Freq(a, 185) // F#3
	a.jmp("music_apply_common")
	a.mark("music_step24")
	setAPUCh1Freq(a, 440) // A4
	setAPUCh2Freq(a, 147) // D3
	a.jmp("music_apply_common")
	a.mark("music_step25")
	setAPUCh1Freq(a, 370) // F#4
	setAPUCh2Freq(a, 147) // D3
	a.jmp("music_apply_common")
	a.mark("music_step26")
	setAPUCh1Freq(a, 294) // D4
	setAPUCh2Freq(a, 220) // A3
	a.jmp("music_apply_common")
	a.mark("music_step27")
	setAPUCh1Freq(a, 370) // F#4
	setAPUCh2Freq(a, 220) // A3
	a.jmp("music_apply_common")
	a.mark("music_step28")
	setAPUCh1Freq(a, 440) // A4
	setAPUCh2Freq(a, 147) // D3
	a.jmp("music_apply_common")
	a.mark("music_step29")
	setAPUCh1Freq(a, 440) // A4 (held)
	setAPUCh2Freq(a, 147) // D3
	a.jmp("music_apply_common")
	a.mark("music_step30")
	setAPUCh1Freq(a, 370) // F#4
	setAPUCh2Freq(a, 220) // A3
	a.jmp("music_apply_common")
	a.mark("music_step31")
	setAPUCh1Freq(a, 294) // D4
	setAPUCh2Freq(a, 220) // A3

	a.mark("music_apply_common")
	// CH1 sine melody / CH2 square bass, timed to re-articulate cleanly.
	a.movImm(4, 0x900A) // CH1 VOLUME
	a.movImm(5, 0x82)   // +~30% melody volume (0x64 -> 0x82)
	a.movStore(4, 5)
	a.movImm(4, 0x900C) // CH1 DUR low
	a.movImm(5, 0x06)
	a.movStore(4, 5)
	a.movImm(4, 0x900D) // CH1 DUR high
	a.movImm(5, 0x00)
	a.movStore(4, 5)
	a.movImm(4, 0x900E) // CH1 DUR mode
	a.movImm(5, 0x00)
	a.movStore(4, 5)
	a.movImm(4, 0x900B) // CH1 CONTROL = enable + sine
	a.movImm(5, 0x01)
	a.movStore(4, 5)

	a.movImm(4, 0x9012) // CH2 VOLUME
	a.movImm(5, 0x4E)   // half bass volume (0x9C -> 0x4E)
	a.movStore(4, 5)
	a.movImm(4, 0x9014) // CH2 DUR low
	a.movImm(5, 0x08)
	a.movStore(4, 5)
	a.movImm(4, 0x9015) // CH2 DUR high
	a.movImm(5, 0x00)
	a.movStore(4, 5)
	a.movImm(4, 0x9016) // CH2 DUR mode
	a.movImm(5, 0x00)
	a.movStore(4, 5)
	a.movImm(4, 0x9013) // CH2 CONTROL = enable + square
	a.movImm(5, 0x03)
	a.movStore(4, 5)

	// Add a simple drum layer (CH3 noise) on quarter-note pulses.
	a.movImm(7, 0x0000)
	a.cmpReg(6, 7)
	a.beq("drum_hit")
	a.movImm(7, 0x0004)
	a.cmpReg(6, 7)
	a.beq("drum_hit")
	a.movImm(7, 0x0008)
	a.cmpReg(6, 7)
	a.beq("drum_hit")
	a.movImm(7, 0x000C)
	a.cmpReg(6, 7)
	a.beq("drum_hit")
	a.movImm(7, 0x0010)
	a.cmpReg(6, 7)
	a.beq("drum_hit")
	a.movImm(7, 0x0014)
	a.cmpReg(6, 7)
	a.beq("drum_hit")
	a.movImm(7, 0x0018)
	a.cmpReg(6, 7)
	a.beq("drum_hit")
	a.movImm(7, 0x001C)
	a.cmpReg(6, 7)
	a.beq("drum_hit")
	a.jmp("drum_done")
	a.mark("drum_hit")
	setAPUCh3Freq(a, 900)
	a.movImm(4, 0x901A) // CH3 VOLUME
	a.movImm(5, 0x40)
	a.movStore(4, 5)
	a.movImm(4, 0x901C) // CH3 DUR low
	a.movImm(5, 0x02)
	a.movStore(4, 5)
	a.movImm(4, 0x901D) // CH3 DUR high
	a.movImm(5, 0x00)
	a.movStore(4, 5)
	a.movImm(4, 0x901E) // CH3 DUR mode
	a.movImm(5, 0x00)
	a.movStore(4, 5)
	a.movImm(4, 0x901B) // CH3 CONTROL = enable + noise
	a.movImm(5, 0x03)
	a.movStore(4, 5)
	a.mark("drum_done")

	// Advance step index (0..31) and reset tick countdown.
	a.movImm(4, wramMusicStep)
	a.movLoad(5, 4)
	a.addImm(5, 1)
	a.movImm(7, 0x0020)
	a.cmpReg(5, 7)
	a.blt("music_store_step")
	a.movImm(5, 0x0000)
	a.mark("music_store_step")
	a.movStore(4, 5)
	a.movImm(4, wramMusicTick)
	a.movImm(5, 0x0008)
	a.movStore(4, 5)
	a.mark("skip_music_engine")

	// START (high-byte bit2) = start note (enable CH0 square)
	a.movReg(5, 3)
	a.andImm(5, 0x0004)
	a.movImm(7, 0x0004)
	a.cmpReg(5, 7)
	a.bne("skip_note_start")
	a.movImm(4, 0x9003)
	a.movImm(5, 0x03) // enable + square waveform
	a.movStore(4, 5)
	a.mark("skip_note_start")

	// Z button / STOP (high-byte bit3, Backspace in UI mapping) = stop all tones + music
	a.movReg(5, 3)
	a.andImm(5, 0x0008)
	a.movImm(7, 0x0008)
	a.cmpReg(5, 7)
	a.bne("skip_note_stop")
	// Music off
	a.movImm(4, wramMusicOn)
	a.movImm(5, 0x0000)
	a.movStore(4, 5)
	a.movImm(4, 0x9003)
	a.movImm(5, 0x02) // square waveform bits set, disabled
	a.movStore(4, 5)
	a.movImm(4, 0x900B)
	a.movImm(5, 0x00) // CH1 sine disabled
	a.movStore(4, 5)
	a.movImm(4, 0x9013)
	a.movImm(5, 0x02) // square waveform bits set, disabled
	a.movStore(4, 5)
	a.movImm(4, 0x901B)
	a.movImm(5, 0x02) // CH3 noise disabled
	a.movStore(4, 5)
	a.mark("skip_note_stop")

	// Wait for VBlank before writing OAM (writes during visible scanlines are ignored)
	a.mark("wait_vblank_for_oam")
	a.movImm(4, 0x803E) // VBLANK_FLAG
	a.movLoad(5, 4)
	a.movImm(7, 0)
	a.cmpReg(5, 7)
	a.bne("do_oam_write")
	a.jmp("wait_vblank_for_oam")

	a.mark("do_oam_write")
	// Update sprite 0 OAM during VBlank
	a.movImm(4, 0x8014) // OAM_ADDR
	a.movImm(5, 0x0000)
	a.movStore(4, 5)

	a.movImm(4, 0x8015) // OAM_DATA
	a.movReg(5, 0)
	a.movStore(4, 5)    // X low
	a.movImm(5, 0x0000) // X high = 0 (avoid sign-extension behavior)
	a.movStore(4, 5)
	a.movReg(5, 1)
	a.movStore(4, 5)    // Y
	a.movImm(5, 0x0001) // Tile index = 1
	a.movStore(4, 5)
	a.movImm(5, 0x0001) // Attributes = palette 1
	a.movStore(4, 5)
	a.movImm(5, 0x0001) // Control = enabled, 8x8
	a.movStore(4, 5)

	a.jmp("main_loop")

	if err := a.resolve(); err != nil {
		fmt.Fprintf(os.Stderr, "Patch/label error: %v\n", err)
		os.Exit(1)
	}

	if err := a.b.BuildROM(1, 0x8000, out); err != nil {
		fmt.Fprintf(os.Stderr, "Build ROM error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Built %s\n", out)
}
