//go:build testrom_tools
// +build testrom_tools

package main

import (
	"compress/gzip"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"nitro-core-dx/internal/rom"
)

type ymWrite struct {
	port uint8 // 0 or 1
	addr uint8
	data uint8
}

type vgmSong struct {
	frames       [][]ymWrite
	frameSamples uint32
	totalSamples uint64
	writeCount   int
}

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

func (a *asm) pc() uint16                { return uint16(a.b.GetCodeLength() * 2) }
func (a *asm) mark(name string)          { a.labels[name] = a.pc() }
func (a *asm) inst(w uint16)             { a.b.AddInstruction(w) }
func (a *asm) imm(v uint16)              { a.b.AddImmediate(v) }
func (a *asm) uniq(prefix string) string { a.uniqID++; return fmt.Sprintf("%s_%d", prefix, a.uniqID) }

func (a *asm) movImm(reg uint8, v uint16)  { a.inst(rom.EncodeMOV(1, reg, 0)); a.imm(v) }
func (a *asm) movLoad(dst, addrReg uint8)  { a.inst(rom.EncodeMOV(2, dst, addrReg)) }
func (a *asm) movStore(addrReg, src uint8) { a.inst(rom.EncodeMOV(3, addrReg, src)) }
func (a *asm) subImm(reg uint8, v uint16)  { a.inst(rom.EncodeSUB(1, reg, 0)); a.imm(v) }
func (a *asm) cmpReg(r1, r2 uint8)         { a.inst(rom.EncodeCMP(0, r1, r2)) }
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

func writeAPU8(a *asm, addr uint16, value uint8) {
	a.movImm(4, addr)
	a.movImm(5, uint16(value))
	a.movStore(4, 5)
}

func writeFMPort0(a *asm, addr, data uint8) {
	writeAPU8(a, 0x9100, addr)
	writeAPU8(a, 0x9101, data)
}

func writeFMPort1(a *asm, addr, data uint8) {
	writeAPU8(a, 0x9104, addr)
	writeAPU8(a, 0x9105, data)
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

func disableLegacyAPUChannels(a *asm) {
	for ch := 0; ch < 4; ch++ {
		base := 0x9000 + (ch * 8)
		writeAPU8(a, uint16(base+3), 0x00) // CONTROL
	}
}

func readVGM(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if strings.EqualFold(filepath.Ext(path), ".vgz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		return io.ReadAll(gz)
	}
	return io.ReadAll(f)
}

func parseVGM(data []byte) (*vgmSong, error) {
	if len(data) < 0x100 || string(data[0:4]) != "Vgm " {
		return nil, errors.New("input is not a valid VGM stream")
	}

	version := binary.LittleEndian.Uint32(data[0x08:0x0C])
	rate := binary.LittleEndian.Uint32(data[0x24:0x28])
	dataOff := uint32(0x40)
	if version >= 0x150 {
		rel := binary.LittleEndian.Uint32(data[0x34:0x38])
		if rel != 0 {
			dataOff = 0x34 + rel
		}
	}

	frameSamples := uint32(735) // 60 Hz default
	if rate == 50 {
		frameSamples = 882
	}

	frames := make([][]ymWrite, 0, 5000)
	current := make([]ymWrite, 0, 64)
	var sampleAcc uint64
	var totalSamples uint64
	var writeCount int

	flushFrame := func() {
		cloned := make([]ymWrite, len(current))
		copy(cloned, current)
		frames = append(frames, cloned)
		current = current[:0]
	}

	addWait := func(n uint64) {
		sampleAcc += n
		totalSamples += n
		for sampleAcc >= uint64(frameSamples) {
			flushFrame()
			sampleAcc -= uint64(frameSamples)
		}
	}

	for p := int(dataOff); p < len(data); {
		cmd := data[p]
		p++

		switch {
		case cmd == 0x66:
			if len(current) > 0 {
				flushFrame()
			}
			return &vgmSong{
				frames:       frames,
				frameSamples: frameSamples,
				totalSamples: totalSamples,
				writeCount:   writeCount,
			}, nil

		case cmd == 0x56 || cmd == 0x57:
			if p+2 > len(data) {
				return nil, fmt.Errorf("truncated YM2608 write at 0x%X", p-1)
			}
			current = append(current, ymWrite{
				port: cmd - 0x56, // 0x56->0, 0x57->1
				addr: data[p],
				data: data[p+1],
			})
			writeCount++
			p += 2

		case cmd == 0x61:
			if p+2 > len(data) {
				return nil, fmt.Errorf("truncated wait(0x61) at 0x%X", p-1)
			}
			n := binary.LittleEndian.Uint16(data[p : p+2])
			addWait(uint64(n))
			p += 2

		case cmd == 0x62:
			addWait(735)

		case cmd == 0x63:
			addWait(882)

		case cmd >= 0x70 && cmd <= 0x7F:
			addWait(uint64((cmd & 0x0F) + 1))

		case cmd == 0x67:
			// Data block: 0x67 0x66 tt ss ss ss ss [data...]
			if p+7 > len(data) {
				return nil, fmt.Errorf("truncated data block at 0x%X", p-1)
			}
			if data[p] != 0x66 {
				return nil, fmt.Errorf("invalid data block marker at 0x%X", p)
			}
			p++ // skip 0x66 marker
			p++ // skip block type
			sz := binary.LittleEndian.Uint32(data[p : p+4])
			p += 4
			if p+int(sz) > len(data) {
				return nil, fmt.Errorf("truncated data block payload at 0x%X", p)
			}
			p += int(sz)

		default:
			return nil, fmt.Errorf("unsupported VGM command 0x%02X at 0x%X", cmd, p-1)
		}
	}

	return nil, errors.New("VGM stream ended without 0x66 end marker")
}

func extractBuilderWords(b *rom.ROMBuilder) ([]uint16, error) {
	img, err := b.BuildROMBytes(1, 0x8000)
	if err != nil {
		return nil, err
	}
	if len(img) < 32 {
		return nil, errors.New("invalid ROM image")
	}
	payload := img[32:]
	if len(payload)%2 != 0 {
		return nil, errors.New("invalid ROM payload size")
	}
	words := make([]uint16, len(payload)/2)
	for i := range words {
		words[i] = binary.LittleEndian.Uint16(payload[i*2 : i*2+2])
	}
	return words, nil
}

func emitFarJumpToBank(a *asm, bank uint8) {
	// Prepare a synthetic RET frame: [Flags, PCOffset, PBR] on stack.
	// RET pops Flags first, then PCOffset, then PBR.
	a.movImm(5, uint16(bank))
	a.inst(rom.EncodeMOV(4, 5, 0)) // PUSH R5 (PBR)
	a.movImm(5, 0x8000)
	a.inst(rom.EncodeMOV(4, 5, 0)) // PUSH R5 (PCOffset)
	a.movImm(5, 0x0000)
	a.inst(rom.EncodeMOV(4, 5, 0)) // PUSH R5 (Flags)
	a.inst(rom.EncodeRET())
}

func buildSongROM(song *vgmSong, outPath string, framesPerBank int) error {
	if framesPerBank <= 0 {
		framesPerBank = len(song.frames)
	}
	if len(song.frames) == 0 {
		return errors.New("no frames to emit")
	}

	banksNeeded := (len(song.frames) + framesPerBank - 1) / framesPerBank
	if banksNeeded > (rom.ROMMaxProgramBank - rom.ROMMinProgramBank + 1) {
		return fmt.Errorf("song requires %d banks with frames-per-bank=%d (max supported banks=%d)",
			banksNeeded, framesPerBank, rom.ROMMaxProgramBank-rom.ROMMinProgramBank+1)
	}

	banked := rom.NewBankedROMBuilder()
	const wramLastFrame = 0x0020

	for seg := 0; seg < banksNeeded; seg++ {
		start := seg * framesPerBank
		end := start + framesPerBank
		if end > len(song.frames) {
			end = len(song.frames)
		}
		bank := uint8(rom.ROMMinProgramBank + seg)
		isFirst := seg == 0
		isLast := end == len(song.frames)

		a := newASM()

		if isFirst {
			// Basic visual + audio init.
			writeAPU8(a, 0x8012, 0x00) // CGRAM index
			writeAPU8(a, 0x8013, 0x10) // dark gray low
			writeAPU8(a, 0x8013, 0x42) // dark gray high
			writeAPU8(a, 0x9020, 0xC0) // master volume
			disableLegacyAPUChannels(a)

			// FM host reset + enable.
			writeAPU8(a, 0x9103, 0x81)
			writeAPU8(a, 0x9103, 0x01)

			// Capture frame baseline.
			a.movImm(4, 0x803F)
			a.movLoad(2, 4)
			a.movImm(4, wramLastFrame)
			a.movStore(4, 2)
		}

		for _, frameWrites := range song.frames[start:end] {
			for _, w := range frameWrites {
				if w.port == 0 {
					writeFMPort0(a, w.addr, w.data)
				} else {
					writeFMPort1(a, w.addr, w.data)
				}
			}
			emitWaitOneFrame(a, wramLastFrame)
		}

		if isLast {
			// Force-safe tail: key-off + host mute so clipped segments do not leave a latched drone.
			for _, chSel := range []uint8{0x00, 0x01, 0x02, 0x04, 0x05, 0x06} {
				writeFMPort0(a, 0x28, chSel)
			}
			writeAPU8(a, 0x9103, 0x03) // mute host gate

			// Stay alive after playback tail.
			a.mark("done")
			emitWaitOneFrame(a, wramLastFrame)
			a.jmp("done")
		} else {
			emitFarJumpToBank(a, bank+1)
		}

		if err := a.resolve(); err != nil {
			return fmt.Errorf("segment %d resolve: %w", seg, err)
		}

		words, err := extractBuilderWords(a.b)
		if err != nil {
			return fmt.Errorf("segment %d extract: %w", seg, err)
		}
		if len(words) > rom.ROMBankSizeWords {
			return fmt.Errorf("segment %d overflow: %d words > %d (reduce frames-per-bank)",
				seg, len(words), rom.ROMBankSizeWords)
		}
		if bank == rom.ROMMinProgramBank {
			// Bank 1 offset 0x8000 is the default IRQ/NMI vector target.
			// Install a RET trampoline there so interrupts return cleanly
			// instead of restarting song code at ROM entry.
			banked.AddInstruction(bank, rom.EncodeRET())
		}
		for _, w := range words {
			banked.AddInstruction(bank, w)
		}
	}

	return banked.BuildROM(1, 0x8002, outPath)
}

func main() {
	inPath := flag.String("in", "Resources/Demo.vgz", "Input VGM/VGZ file")
	outPath := flag.String("out", "roms/ym2608_demo_song.rom", "Output ROM path")
	maxFrames := flag.Int("max-frames", 0, "Frame cap for generated ROM playback (0 = full song)")
	framesPerBank := flag.Int("frames-per-bank", 70, "Max emitted frames per ROM bank segment")
	flag.Parse()

	raw, err := readVGM(*inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", *inPath, err)
		os.Exit(1)
	}

	song, err := parseVGM(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse VGM: %v\n", err)
		os.Exit(1)
	}

	if *maxFrames > 0 && *maxFrames < len(song.frames) {
		song.frames = song.frames[:*maxFrames]
	}
	if err := buildSongROM(song, *outPath, *framesPerBank); err != nil {
		fmt.Fprintf(os.Stderr, "build ROM: %v\n", err)
		os.Exit(1)
	}

	selectedWrites := 0
	for _, fw := range song.frames {
		selectedWrites += len(fw)
	}
	seconds := float64(len(song.frames)*int(song.frameSamples)) / 44100.0
	fmt.Printf("Built %s\n", *outPath)
	fmt.Printf("Frames: %d  YM writes in ROM: %d (source total: %d)  Approx duration: %.2fs\n",
		len(song.frames), selectedWrites, song.writeCount, seconds)
	fmt.Printf("Frame quantum: %d samples/frame\n", song.frameSamples)
	fmt.Printf("Frames per bank: %d  Estimated banks used: %d\n",
		*framesPerBank, (len(song.frames)+*framesPerBank-1)/(*framesPerBank))
}
