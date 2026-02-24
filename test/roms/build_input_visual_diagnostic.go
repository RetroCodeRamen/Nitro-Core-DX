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

func (a *asm) pc() uint16 { return uint16(a.b.GetCodeLength() * 2) }
func (a *asm) mark(name string) { a.labels[name] = a.pc() }
func (a *asm) inst(w uint16) { a.b.AddInstruction(w) }
func (a *asm) imm(v uint16)  { a.b.AddImmediate(v) }

func (a *asm) movImm(reg uint8, v uint16) { a.inst(rom.EncodeMOV(1, reg, 0)); a.imm(v) }
func (a *asm) movReg(dst, src uint8)      { a.inst(rom.EncodeMOV(0, dst, src)) }
func (a *asm) movLoad(dst, addrReg uint8) { a.inst(rom.EncodeMOV(2, dst, addrReg)) }
func (a *asm) movStore(addrReg, src uint8){ a.inst(rom.EncodeMOV(3, addrReg, src)) }
func (a *asm) addImm(reg uint8, v uint16) { a.inst(rom.EncodeADD(1, reg, 0)); a.imm(v) }
func (a *asm) subImm(reg uint8, v uint16) { a.inst(rom.EncodeSUB(1, reg, 0)); a.imm(v) }
func (a *asm) andImm(reg uint8, v uint16) { a.inst(rom.EncodeAND(1, reg, 0)); a.imm(v) }
func (a *asm) cmpImm(reg uint8, v uint16) { a.inst(rom.EncodeCMP(1, reg, 0)); a.imm(v) }
func (a *asm) cmpReg(r1, r2 uint8)        { a.inst(rom.EncodeCMP(0, r1, r2)) }

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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run -tags testrom_tools ./test/roms/build_input_visual_diagnostic.go <output.rom>")
		os.Exit(1)
	}

	out := os.Args[1]
	a := newASM()

	fmt.Println("Building input visual diagnostic ROM (v2)...")
	fmt.Println("Controls:")
	fmt.Println("  Arrows/WASD: move sprite (1 pixel per frame)")
	fmt.Println("  Z (A): change background color")
	fmt.Println("  X (B): change sprite color")
	fmt.Println("  C (Y): reset sprite position")
	fmt.Println("  Q (L) / E (R): select note (low/high, default middle)")
	fmt.Println("  Enter (START): start note")
	fmt.Println("  Backspace (Z button): stop note")

	// Registers:
	// R0 = sprite X (0..248)
	// R1 = sprite Y (0..192)
	// R2 = input low byte  (buttons 0..7)
	// R3 = input high byte (buttons 8..15)
	// R4 = IO address scratch
	// R5 = data scratch
	// R6 = last frame counter low byte (for one-update-per-frame sync)
	// R7 = compare/mask scratch

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
	a.movImm(6, 800)      // 32 * 25 tile entries
	a.movImm(4, 0x8010)   // VRAM_DATA
	a.mark("fill_tilemap")
	a.movImm(5, 0x00)     // tile index 0
	a.movStore(4, 5)
	a.movImm(5, 0x00)     // attrs palette 0
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

	// ---- Initial sprite position/state ----
	a.movImm(0, 120) // X
	a.movImm(1, 96)  // Y
	// Capture current frame counter low byte as sync baseline
	a.movImm(4, 0x803F) // FRAME_COUNTER_LOW
	a.movLoad(6, 4)

	// ---- Main loop ----
	a.mark("main_loop")

	// Wait for next frame counter tick (one logic update per emulated frame)
	a.mark("wait_next_frame")
	a.movImm(4, 0x803F) // FRAME_COUNTER_LOW
	a.movLoad(5, 4)
	a.cmpReg(5, 6)
	a.bne("frame_start")
	a.jmp("wait_next_frame")

	a.mark("frame_start")
	a.movReg(6, 5) // remember current frame counter low byte

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
	a.movImm(7, 0x0080)
	a.cmpReg(5, 7)
	a.bne("skip_reset")
	a.movImm(0, 120)
	a.movImm(1, 96)
	a.mark("skip_reset")

	// UP (bit 0)
	a.movReg(5, 2)
	a.andImm(5, 0x0001)
	a.movImm(7, 0x0001)
	a.cmpReg(5, 7)
	a.bne("skip_up")
	a.movImm(7, 0)
	a.cmpReg(1, 7)
	a.beq("skip_up")
	a.subImm(1, 1)
	a.mark("skip_up")

	// DOWN (bit 1)
	a.movReg(5, 2)
	a.andImm(5, 0x0002)
	a.movImm(7, 0x0002)
	a.cmpReg(5, 7)
	a.bne("skip_down")
	a.cmpImm(1, 192)
	a.beq("skip_down")
	a.addImm(1, 1)
	a.mark("skip_down")

	// LEFT (bit 2)
	a.movReg(5, 2)
	a.andImm(5, 0x0004)
	a.movImm(7, 0x0004)
	a.cmpReg(5, 7)
	a.bne("skip_left")
	a.movImm(7, 0)
	a.cmpReg(0, 7)
	a.beq("skip_left")
	a.subImm(0, 1)
	a.mark("skip_left")

	// RIGHT (bit 3)
	a.movReg(5, 2)
	a.andImm(5, 0x0008)
	a.movImm(7, 0x0008)
	a.cmpReg(5, 7)
	a.bne("skip_right")
	// Avoid cmpImm on R0: encoding collides with BEQ when reg1=0 in current CPU decode.
	a.movImm(7, 248) // keep xHigh=0 to avoid sign-extension edge cases
	a.cmpReg(0, 7)
	a.beq("skip_right")
	a.addImm(0, 1)
	a.mark("skip_right")

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

	// Audio note selection from high-byte buttons:
	// default=440 Hz, L(bit0)=330 Hz, R(bit1)=523 Hz (R overrides L if both held)
	setAPUCh0Freq(a, 440)
	a.movReg(5, 3)
	a.andImm(5, 0x0001) // L
	a.movImm(7, 0x0001)
	a.cmpReg(5, 7)
	a.bne("skip_note_l")
	setAPUCh0Freq(a, 330)
	a.mark("skip_note_l")
	a.movReg(5, 3)
	a.andImm(5, 0x0002) // R
	a.movImm(7, 0x0002)
	a.cmpReg(5, 7)
	a.bne("skip_note_r")
	setAPUCh0Freq(a, 523)
	a.mark("skip_note_r")

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

	// Z button / STOP (high-byte bit3, Backspace in UI mapping) = stop note
	a.movReg(5, 3)
	a.andImm(5, 0x0008)
	a.movImm(7, 0x0008)
	a.cmpReg(5, 7)
	a.bne("skip_note_stop")
	a.movImm(4, 0x9003)
	a.movImm(5, 0x02) // square waveform bits set, disabled
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
	a.movStore(4, 0)    // X low
	a.movImm(5, 0x0000) // X high = 0 (avoid sign-extension behavior)
	a.movStore(4, 5)
	a.movStore(4, 1)    // Y
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
