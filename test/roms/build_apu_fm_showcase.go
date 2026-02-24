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
	currentPC uint16
	target    string
}

type asm struct {
	b       *rom.ROMBuilder
	labels  map[string]uint16
	patches []patchRef
	uniqID  int
}

func newASM() *asm {
	return &asm{
		b:      rom.NewROMBuilder(),
		labels: make(map[string]uint16),
	}
}

func (a *asm) pc() uint16                  { return uint16(a.b.GetCodeLength() * 2) }
func (a *asm) mark(name string)            { a.labels[name] = a.pc() }
func (a *asm) inst(w uint16)               { a.b.AddInstruction(w) }
func (a *asm) imm(v uint16)                { a.b.AddImmediate(v) }
func (a *asm) uniq(prefix string) string   { a.uniqID++; return fmt.Sprintf("%s_%d", prefix, a.uniqID) }
func (a *asm) movImm(reg uint8, v uint16)  { a.inst(rom.EncodeMOV(1, reg, 0)); a.imm(v) }
func (a *asm) movReg(dst, src uint8)       { a.inst(rom.EncodeMOV(0, dst, src)) }
func (a *asm) movLoad(dst, addrReg uint8)  { a.inst(rom.EncodeMOV(2, dst, addrReg)) }
func (a *asm) movStore(addrReg, src uint8) { a.inst(rom.EncodeMOV(3, addrReg, src)) }
func (a *asm) addImm(reg uint8, v uint16)  { a.inst(rom.EncodeADD(1, reg, 0)); a.imm(v) }
func (a *asm) subImm(reg uint8, v uint16)  { a.inst(rom.EncodeSUB(1, reg, 0)); a.imm(v) }
func (a *asm) andImm(reg uint8, v uint16)  { a.inst(rom.EncodeAND(1, reg, 0)); a.imm(v) }
func (a *asm) cmpReg(r1, r2 uint8)         { a.inst(rom.EncodeCMP(0, r1, r2)) }
func (a *asm) branch(op uint16, label string) {
	a.inst(op)
	pc := a.pc()
	a.imm(0)
	a.patches = append(a.patches, patchRef{wordIndex: a.b.GetCodeLength() - 1, currentPC: pc, target: label})
}
func (a *asm) beq(label string) { a.branch(rom.EncodeBEQ(), label) }
func (a *asm) bne(label string) { a.branch(rom.EncodeBNE(), label) }
func (a *asm) bgt(label string) { a.branch(rom.EncodeBGT(), label) }
func (a *asm) blt(label string) { a.branch(rom.EncodeBLT(), label) }
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

type noteStep struct {
	melodyHz  uint16
	harmonyHz uint16
	frames    uint16
}

func setVRAMAddr(a *asm, addr uint16) {
	a.movImm(4, 0x800E)
	a.movImm(5, addr&0xFF)
	a.movStore(4, 5)
	a.movImm(4, 0x800F)
	a.movImm(5, (addr>>8)&0xFF)
	a.movStore(4, 5)
}

func setCGRAMColor(a *asm, colorIndex uint8, rgb555 uint16) {
	a.movImm(4, 0x8012)
	a.movImm(5, uint16(colorIndex))
	a.movStore(4, 5)
	a.movImm(4, 0x8013)
	a.movImm(5, rgb555&0xFF)
	a.movStore(4, 5)
	a.movImm(5, (rgb555>>8)&0xFF)
	a.movStore(4, 5)
}

func setAPUReg8(a *asm, addr uint16, v uint8) {
	a.movImm(4, addr)
	a.movImm(5, uint16(v))
	a.movStore(4, 5)
}

func setAPUReg16(a *asm, addr uint16, v uint16) {
	setAPUReg8(a, addr, uint8(v&0xFF))
	setAPUReg8(a, addr+1, uint8(v>>8))
}

func apuChBase(ch int) uint16 { return 0x9000 + uint16(ch*8) }

func setAPUChannelFreq(a *asm, ch int, hz uint16) {
	base := apuChBase(ch)
	setAPUReg16(a, base+0, hz)
}

func setAPUChannelVol(a *asm, ch int, vol uint8) {
	setAPUReg8(a, apuChBase(ch)+2, vol)
}

func setAPUChannelCtrl(a *asm, ch int, ctrl uint8) {
	setAPUReg8(a, apuChBase(ch)+3, ctrl)
}

func setAPUChannelDuration(a *asm, ch int, frames uint16, loop bool) {
	base := apuChBase(ch)
	setAPUReg16(a, base+4, frames)
	if loop {
		setAPUReg8(a, base+6, 0x01)
	} else {
		setAPUReg8(a, base+6, 0x00)
	}
}

func disableAllAPUChannels(a *asm) {
	for ch := 0; ch < 4; ch++ {
		setAPUChannelCtrl(a, ch, 0x00)
	}
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

func emitWaitOneFrame(a *asm, wramLastFrame uint16) {
	waitLabel := a.uniq("wait_frame")
	doneLabel := a.uniq("frame_advance")
	a.mark(waitLabel)
	a.movImm(4, 0x803F) // FRAME_COUNTER_LOW
	a.movLoad(2, 4)
	a.movImm(4, wramLastFrame)
	a.movLoad(3, 4)
	a.cmpReg(2, 3)
	a.bne(doneLabel)
	a.jmp(waitLabel)
	a.mark(doneLabel)
	a.movImm(4, wramLastFrame)
	a.movStore(4, 2)
}

func emitWaitFrames(a *asm, wramLastFrame uint16, frames uint16) {
	for i := uint16(0); i < frames; i++ {
		emitWaitOneFrame(a, wramLastFrame)
	}
}

func emitWaitAPUCompletion(a *asm, channel int) {
	mask := uint16(1 << channel)
	loop := a.uniq("wait_apu_done")
	done := a.uniq("apu_done")
	a.mark(loop)
	a.movImm(4, 0x9021) // CHANNEL_COMPLETION_STATUS (one-shot)
	a.movLoad(5, 4)
	a.andImm(5, mask)
	a.movImm(6, mask)
	a.cmpReg(5, 6)
	a.beq(done)
	a.jmp(loop)
	a.mark(done)
}

func emitWaitFMTimerAFlag(a *asm) {
	loop := a.uniq("wait_fm_ta")
	done := a.uniq("fm_ta_done")
	a.mark(loop)
	a.movImm(4, 0x9102) // FM_STATUS
	a.movLoad(5, 4)
	a.andImm(5, 0x0001) // Timer A flag
	a.movImm(6, 0x0001)
	a.cmpReg(5, 6)
	a.beq(done)
	a.jmp(loop)
	a.mark(done)
}

func emitLegacyNote(a *asm, channel int, hz uint16, frames uint16, volume uint8, waveform uint8, wramLastFrame uint16) {
	if hz == 0 {
		setAPUChannelCtrl(a, channel, 0x00)
		emitWaitFrames(a, wramLastFrame, frames)
		return
	}
	setAPUChannelFreq(a, channel, hz)
	setAPUChannelVol(a, channel, volume)
	setAPUChannelDuration(a, channel, frames, false)
	ctrl := uint8(0x01) | ((waveform & 0x03) << 1)
	if channel == 3 {
		// Channel 3 treats bit1 as noise mode selector in current APU.
		if waveform == 3 {
			ctrl = 0x03
		} else {
			ctrl = 0x01
		}
	}
	setAPUChannelCtrl(a, channel, ctrl)
	emitWaitAPUCompletion(a, channel)
}

func emitDualChannelNote(a *asm, melodyHz, harmonyHz, frames uint16, wramLastFrame uint16) {
	// Melody on CH0 (square), harmony on CH1 (sine).
	if harmonyHz != 0 {
		setAPUChannelFreq(a, 1, harmonyHz)
		setAPUChannelVol(a, 1, 96)
		setAPUChannelDuration(a, 1, frames, false)
		setAPUChannelCtrl(a, 1, 0x01|(0x00<<1)) // enable + sine
	} else {
		setAPUChannelCtrl(a, 1, 0x00)
	}

	// Melody note drives blocking completion wait to showcase duration/completion.
	emitLegacyNote(a, 0, melodyHz, frames, 170, 1, wramLastFrame) // square

	// Add a tiny noise click on CH3 every note as a percussive marker (optional complexity).
	setAPUChannelFreq(a, 3, 1000)
	setAPUChannelVol(a, 3, 48)
	setAPUChannelDuration(a, 3, 2, false)
	setAPUChannelCtrl(a, 3, 0x03) // enable + noise mode
	emitWaitFrames(a, wramLastFrame, 1)
}

func emitLegacyScale(a *asm, channel int, waveform uint8, volume uint8, wramLastFrame uint16) {
	scale := []uint16{262, 294, 330, 349, 392, 440, 494, 523}
	for _, hz := range scale {
		emitLegacyNote(a, channel, hz, 8, volume, waveform, wramLastFrame)
	}
	setAPUChannelCtrl(a, channel, 0x00)
}

func emitFMProxyScaleDemo(a *asm, wramLastFrame uint16) {
	// Ensure FM extension is enabled (host control).
	writeFMHost(a, 0x9103, 0x01)
	// Configure Timer A for shortest deterministic phase-1 expiry.
	writeFMOPMReg(a, 0x10, 0xFF)
	writeFMOPMReg(a, 0x11, 0x03)

	scale := []uint16{262, 294, 330, 349, 392, 440, 494, 523}
	for i, hz := range scale {
		// Write a future-facing FM "note-ish" register pattern (currently shadowed by FM stub).
		writeFMOPMReg(a, 0x28, uint8(0x30+i)) // arbitrary note code shadow for diagnostics
		writeFMOPMReg(a, 0x14, 0x01)          // start Timer A (no IRQ enable to avoid CPU IRQ during this ROM)
		emitWaitFMTimerAFlag(a)

		// Flash cyan while Timer A flag is observed.
		setCGRAMColor(a, 0x00, 0x03FF)

		// Clear Timer A flag while keeping Timer A running.
		writeFMOPMReg(a, 0x14, 0x05) // start A + clear A flag

		// Audible proxy on legacy CH2 so the user hears the scale until FM synthesis exists.
		emitLegacyNote(a, 2, hz, 7, 120, 2, wramLastFrame) // saw waveform proxy

		// Back to blue base after each step.
		setCGRAMColor(a, 0x00, 0x001F)
	}

	setAPUChannelCtrl(a, 2, 0x00)
}

func emitBachExcerpt(a *asm, wramLastFrame uint16) {
	// Simplified short excerpt inspired by "Jesu, Joy of Man's Desiring" (public-domain composition).
	phrase := []noteStep{
		{392, 262, 12}, // G4 / C4
		{440, 294, 12}, // A4 / D4
		{494, 330, 12}, // B4 / E4
		{523, 349, 12}, // C5 / F4
		{587, 392, 16}, // D5 / G4
		{523, 349, 12}, // C5 / F4
		{494, 330, 12}, // B4 / E4
		{440, 294, 12}, // A4 / D4
		{392, 262, 16}, // G4 / C4
		{440, 294, 12}, // A4 / D4
		{494, 330, 12}, // B4 / E4
		{523, 349, 12}, // C5 / F4
		{494, 330, 16}, // B4 / E4
		{440, 294, 12}, // A4 / D4
		{392, 262, 12}, // G4 / C4
		{330, 247, 20}, // E4 / B3
	}

	// FM extension writes are future-facing only (register shadow + status path today).
	writeFMHost(a, 0x9103, 0x01)
	writeFMOPMReg(a, 0x10, 0xF0)
	writeFMOPMReg(a, 0x11, 0x03)

	for i, st := range phrase {
		// Mirror a "note code" into FM register shadow for future synthesis diagnostics.
		writeFMOPMReg(a, 0x29, uint8(0x40+(i&0x1F)))
		writeFMOPMReg(a, 0x14, 0x01)
		emitWaitFMTimerAFlag(a)
		writeFMOPMReg(a, 0x14, 0x05)

		emitDualChannelNote(a, st.melodyHz, st.harmonyHz, st.frames, wramLastFrame)
	}

	disableAllAPUChannels(a)
}

func fillSolidBG(a *asm) {
	// Tile 0: solid color index 0
	setVRAMAddr(a, 0x0000)
	a.movImm(6, 32)
	a.movImm(4, 0x8010)
	a.movImm(5, 0x00)
	loop0 := a.uniq("fill_tile0")
	done0 := a.uniq("fill_tile0_done")
	a.mark(loop0)
	a.movStore(4, 5)
	a.subImm(6, 1)
	a.movImm(7, 0)
	a.cmpReg(6, 7)
	a.beq(done0)
	a.jmp(loop0)
	a.mark(done0)

	// Fill visible 32x25 tilemap with tile 0
	setVRAMAddr(a, 0x4000)
	a.movImm(6, 800)
	a.movImm(4, 0x8010)
	loop1 := a.uniq("fill_map")
	done1 := a.uniq("fill_map_done")
	a.mark(loop1)
	a.movImm(5, 0x00) // tile index
	a.movStore(4, 5)
	a.movImm(5, 0x00) // attrs palette 0
	a.movStore(4, 5)
	a.subImm(6, 1)
	a.movImm(7, 0)
	a.cmpReg(6, 7)
	a.beq(done1)
	a.jmp(loop1)
	a.mark(done1)

	// Enable BG0
	a.movImm(4, 0x8008)
	a.movImm(5, 0x01)
	a.movStore(4, 5)
}

func emitReadInput(a *asm) {
	// Latch input and read low/high bytes into R2/R3
	a.movImm(4, 0xA001)
	a.movImm(5, 0x01)
	a.movStore(4, 5)
	a.movImm(4, 0xA000)
	a.movLoad(2, 4)
	a.movImm(4, 0xA001)
	a.movLoad(3, 4)
	a.movImm(5, 0x00)
	a.movStore(4, 5)
}

func emitWaitRelease(a *asm) {
	loop := a.uniq("wait_release")
	clear := a.uniq("released")
	a.mark(loop)
	emitReadInput(a)
	a.movReg(5, 2)
	a.andImm(5, 0x00B0) // A/B/Y (keyboard Z/X/C in current UI mapping)
	a.movImm(6, 0x0000)
	a.cmpReg(5, 6)
	a.beq(clear)
	a.jmp(loop)
	a.mark(clear)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run -tags testrom_tools ./test/roms/build_apu_fm_showcase.go <output.rom>")
		os.Exit(1)
	}

	out := os.Args[1]
	a := newASM()

	fmt.Println("Building APU + FM showcase diagnostic ROM...")
	fmt.Println("Controls (current default keyboard mapping):")
	fmt.Println("  Z (A): Legacy APU scale")
	fmt.Println("  X (B): FM extension MMIO/timer demo + audible legacy proxy scale")
	fmt.Println("  C (Y): Simplified Bach excerpt using multi-channel legacy APU + duration/completion")
	fmt.Println("Note: FM extension audio synthesis is not implemented yet; B/C mirror writes to FM MMIO for future validation.")

	const wramLastFrame = 0x0020

	// Basic visual setup (full-screen solid tile; color 0 is the status/background color).
	setCGRAMColor(a, 0x00, 0x4210) // idle gray
	fillSolidBG(a)

	// APU init
	setAPUReg8(a, 0x9020, 0xC0) // master volume
	disableAllAPUChannels(a)

	// FM extension init (enable host block, clear status via reset pulse then enable)
	writeFMHost(a, 0x9103, 0x81) // reset request + enable (reset bit is one-shot)
	writeFMHost(a, 0x9103, 0x01) // enable

	// Capture frame baseline
	a.movImm(4, 0x803F)
	a.movLoad(2, 4)
	a.movImm(4, wramLastFrame)
	a.movStore(4, 2)

	// Main loop
	a.mark("main_loop")
	emitWaitOneFrame(a, wramLastFrame)
	emitReadInput(a)

	// Idle color
	setCGRAMColor(a, 0x00, 0x4210) // gray

	// If A (bit4), run legacy scale
	a.movReg(5, 2)
	a.andImm(5, 0x0010)
	a.movImm(6, 0x0010)
	a.cmpReg(5, 6)
	a.beq("handle_a")

	// If B (bit5), run FM demo/proxy scale
	a.movReg(5, 2)
	a.andImm(5, 0x0020)
	a.movImm(6, 0x0020)
	a.cmpReg(5, 6)
	a.beq("handle_b")

	// If Y (bit7, keyboard C), run Bach excerpt
	a.movReg(5, 2)
	a.andImm(5, 0x0080)
	a.movImm(6, 0x0080)
	a.cmpReg(5, 6)
	a.beq("handle_c")

	a.jmp("main_loop")

	a.mark("handle_a")
	setCGRAMColor(a, 0x00, 0x03E0)               // green
	emitLegacyScale(a, 0, 1, 180, wramLastFrame) // CH0 square
	setCGRAMColor(a, 0x00, 0x4210)
	emitWaitRelease(a)
	a.jmp("main_loop")

	a.mark("handle_b")
	setCGRAMColor(a, 0x00, 0x001F) // blue
	emitFMProxyScaleDemo(a, wramLastFrame)
	setCGRAMColor(a, 0x00, 0x4210)
	emitWaitRelease(a)
	a.jmp("main_loop")

	a.mark("handle_c")
	setCGRAMColor(a, 0x00, 0x7FE0) // yellow
	emitBachExcerpt(a, wramLastFrame)
	setCGRAMColor(a, 0x00, 0x4210)
	emitWaitRelease(a)
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
