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
	"math"
	"os"
	"path/filepath"
	"strings"

	"nitro-core-dx/internal/rom"
	"nitro-core-dx/internal/ymstream"
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
func (a *asm) movReg(dst, src uint8)       { a.inst(rom.EncodeMOV(0, dst, src)) }
func (a *asm) movLoad(dst, addrReg uint8)  { a.inst(rom.EncodeMOV(2, dst, addrReg)) }
func (a *asm) movLoad8(dst, addrReg uint8) { a.inst(rom.EncodeMOV(6, dst, addrReg)) }
func (a *asm) movStore(addrReg, src uint8) { a.inst(rom.EncodeMOV(3, addrReg, src)) }
func (a *asm) setDBR(src uint8)            { a.inst(rom.EncodeMOV(8, src, 0)) }
func (a *asm) addImm(reg uint8, v uint16)  { a.inst(rom.EncodeADD(1, reg, 0)); a.imm(v) }
func (a *asm) addReg(dst, src uint8)       { a.inst(rom.EncodeADD(0, dst, src)) }
func (a *asm) subImm(reg uint8, v uint16)  { a.inst(rom.EncodeSUB(1, reg, 0)); a.imm(v) }
func (a *asm) subReg(dst, src uint8)       { a.inst(rom.EncodeSUB(0, dst, src)) }
func (a *asm) andImm(reg uint8, v uint16)  { a.inst(rom.EncodeAND(1, reg, 0)); a.imm(v) }
func (a *asm) cmpImm(reg uint8, v uint16) {
	a.inst(rom.EncodeCMP(7, reg, 0))
	a.imm(v)
}
func (a *asm) cmpReg(r1, r2 uint8)        { a.inst(rom.EncodeCMP(0, r1, r2)) }
func (a *asm) shrImm(reg uint8, v uint16) { a.inst(rom.EncodeSHR(1, reg, 0)); a.imm(v) }

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

	frameSamples := uint32(735)
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
			current = append(current, ymWrite{port: cmd - 0x56, addr: data[p], data: data[p+1]})
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
			if p+7 > len(data) {
				return nil, fmt.Errorf("truncated data block at 0x%X", p-1)
			}
			if data[p] != 0x66 {
				return nil, fmt.Errorf("invalid data block marker at 0x%X", p)
			}
			p++
			p++
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

func encodeCompactSong(song *vgmSong) ([]byte, error) {
	frames := make([][]ymstream.Write, len(song.frames))
	for i, fw := range song.frames {
		frames[i] = make([]ymstream.Write, len(fw))
		for j, w := range fw {
			frames[i][j] = ymstream.Write{Port: w.port, Addr: w.addr, Data: w.data}
		}
	}
	return ymstream.EncodeSong(&ymstream.Song{
		Frames:       frames,
		FrameSamples: song.frameSamples,
		TotalSamples: song.totalSamples,
		WriteCount:   song.writeCount,
	})
}

func encodeFrameTable(song *vgmSong) ([]byte, []byte, int) {
	counts := make([]byte, len(song.frames)*2)
	writeBytes := make([]byte, 0, song.writeCount*3)
	totalWrites := 0
	for i, fw := range song.frames {
		binary.LittleEndian.PutUint16(counts[i*2:i*2+2], uint16(len(fw)))
		totalWrites += len(fw)
		for _, w := range fw {
			writeBytes = append(writeBytes, w.port, w.addr, w.data)
		}
	}
	return counts, writeBytes, totalWrites
}

func encodeFramePointers(song *vgmSong, writeStartBank uint8, writeStartOffset uint16) []byte {
	ptrs := make([]byte, len(song.frames)*4)
	byteOffset := 0
	for i, fw := range song.frames {
		abs := int(writeStartOffset) + byteOffset
		bank := writeStartBank + uint8((abs-0x8000)/rom.ROMBankSizeBytes)
		off := uint16(0x8000 + ((abs - 0x8000) % rom.ROMBankSizeBytes))
		base := i * 4
		ptrs[base] = bank
		binary.LittleEndian.PutUint16(ptrs[base+1:base+3], off)
		ptrs[base+3] = 0
		byteOffset += len(fw) * 3
	}
	return ptrs
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

func write8(a *asm, addr uint16, value uint8) {
	a.movImm(4, addr)
	a.movImm(5, uint16(value))
	a.movStore(4, 5)
}

func write16(a *asm, addr uint16, value uint16) {
	write8(a, addr, uint8(value&0xFF))
	write8(a, addr+1, uint8((value>>8)&0xFF))
}

func write16s(a *asm, addr uint16, value int16) {
	write16(a, addr, uint16(value))
}

func setVRAMAddr(a *asm, addr uint16) {
	write8(a, 0x800E, uint8(addr&0xFF))
	write8(a, 0x800F, uint8((addr>>8)&0xFF))
}

func setCGRAMColor(a *asm, colorIndex uint8, rgb555 uint16) {
	write8(a, 0x8012, colorIndex)
	write8(a, 0x8013, uint8(rgb555&0xFF))
	write8(a, 0x8013, uint8((rgb555>>8)&0xFF))
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

func buildSolidTile(size int, color uint8) []uint8 {
	return makePackedTile(size, func(x, y int) uint8 {
		return color
	})
}

func buildPaddleTile() []uint8 {
	return makePackedTile(16, func(x, y int) uint8 {
		if x >= 6 && x <= 9 {
			return 1
		}
		return 0
	})
}

func buildBallTile() []uint8 {
	return makePackedTile(16, func(x, y int) uint8 {
		dx := x - 7
		dy := y - 7
		d2 := dx*dx + dy*dy
		switch {
		case d2 <= 16:
			return 2
		case d2 <= 28:
			return 1
		default:
			return 0
		}
	})
}

func buildPipTile() []uint8 {
	return makePackedTile(16, func(x, y int) uint8 {
		if x >= 4 && x <= 11 && y >= 4 && y <= 11 {
			return 1
		}
		return 0
	})
}

func buildCheckerTilemap(baseA, baseB, marker uint8, width int) []uint8 {
	data := make([]uint8, width*width*2)
	for y := 0; y < width; y++ {
		for x := 0; x < width; x++ {
			mx := x / 2
			my := y / 2
			tile := baseA
			if (mx+my)%2 != 0 {
				tile = baseB
			}

			// Large-scale asymmetry so the rotated field does not read like a
			// perfectly repeating square.
			if mx <= 1 || my <= 1 {
				tile = baseA
			}
			if mx >= 14 && my >= 10 {
				tile = baseB
			}

			// Add lane/marker features across the court.
			if (mx >= 4 && mx <= 12 && (my == 7 || my == 8)) ||
				(my >= 3 && my <= 11 && (mx == 5 || mx == 6)) ||
				(mx >= 10 && mx <= 13 && (my == 3 || my == 4)) {
				tile = marker
			}
			idx := (y*width + x) * 2
			data[idx] = tile
			data[idx+1] = 0x00
		}
	}
	return data
}

func fillTilemapConstant(a *asm, base uint16, entries uint16, tileIndex uint8, attr uint8) {
	loop := a.uniq("fill_tilemap")
	setVRAMAddr(a, base)
	a.movImm(6, entries)
	a.movImm(4, 0x8010)
	a.mark(loop)
	a.movImm(5, uint16(tileIndex))
	a.movStore(4, 5)
	a.movImm(5, uint16(attr))
	a.movStore(4, 5)
	a.subImm(6, 1)
	a.movImm(7, 0)
	a.cmpReg(6, 7)
	a.bne(loop)
}

func emitCheckerTilemapRuntime(a *asm, base uint16, width uint16, groupShift uint16, baseA, baseB uint8) {
	setVRAMAddr(a, base)
	a.movImm(4, 0x8010) // VRAM data port
	a.movImm(0, 0)      // y

	rowLoop := a.uniq("checker_row")
	colLoop := a.uniq("checker_col")
	useA := a.uniq("checker_use_a")
	nextCol := a.uniq("checker_next_col")
	nextRow := a.uniq("checker_next_row")

	a.mark(rowLoop)
	a.movImm(1, 0) // x
	a.mark(colLoop)

	// parity = ((x >> groupShift) + (y >> groupShift)) & 1
	a.movReg(2, 1)
	a.shrImm(2, groupShift)
	a.movReg(3, 0)
	a.shrImm(3, groupShift)
	a.addReg(2, 3)
	a.andImm(2, 1)
	a.cmpImm(2, 0)
	a.beq(useA)

	a.movImm(5, uint16(baseB))
	a.movStore(4, 5)
	a.movImm(5, 0)
	a.movStore(4, 5)
	a.jmp(nextCol)

	a.mark(useA)
	a.movImm(5, uint16(baseA))
	a.movStore(4, 5)
	a.movImm(5, 0)
	a.movStore(4, 5)

	a.mark(nextCol)
	a.addImm(1, 1)
	a.cmpImm(1, width)
	a.blt(colLoop)

	a.mark(nextRow)
	a.addImm(0, 1)
	a.cmpImm(0, width)
	a.blt(rowLoop)
}

func emitMatrixBackgroundFrame(a *asm, animFrame int) {
	angle := float64(animFrame) * (math.Pi / 2700.0)
	scale := 1.15
	cosv := int16(math.Round(math.Cos(angle) * scale * 256.0))
	sinv := int16(math.Round(math.Sin(angle) * scale * 256.0))

	write16s(a, 0x8019, cosv)
	write16s(a, 0x801B, -sinv)
	write16s(a, 0x801D, sinv)
	write16s(a, 0x801F, cosv)
}

func emitText(a *asm, x uint16, y uint8, r, g, b uint8, s string) {
	write8(a, 0x8070, uint8(x&0xFF))
	write8(a, 0x8071, uint8((x>>8)&0xFF))
	write8(a, 0x8072, y)
	write8(a, 0x8073, r)
	write8(a, 0x8074, g)
	write8(a, 0x8075, b)
	for i := 0; i < len(s); i++ {
		write8(a, 0x8076, s[i])
	}
}

func emitDigit(a *asm, x uint16, y uint8, r, g, b uint8, reg uint8) {
	write8(a, 0x8070, uint8(x&0xFF))
	write8(a, 0x8071, uint8((x>>8)&0xFF))
	write8(a, 0x8072, y)
	write8(a, 0x8073, r)
	write8(a, 0x8074, g)
	write8(a, 0x8075, b)
	a.movReg(6, reg)
	a.addImm(6, uint16('0'))
	a.movImm(4, 0x8076)
	a.movStore(4, 6)
}

func writeFMPort0(a *asm, addr, data uint8) {
	write8(a, 0x9100, addr)
	write8(a, 0x9101, data)
}

func writeFMPort1(a *asm, addr, data uint8) {
	write8(a, 0x9104, addr)
	write8(a, 0x9105, data)
}

func writeFMPort0FromRegs(a *asm, addrReg, dataReg uint8) {
	a.movImm(4, 0x9100)
	a.movStore(4, addrReg)
	a.movImm(4, 0x9101)
	a.movStore(4, dataReg)
}

func writeFMPort1FromRegs(a *asm, addrReg, dataReg uint8) {
	a.movImm(4, 0x9104)
	a.movStore(4, addrReg)
	a.movImm(4, 0x9105)
	a.movStore(4, dataReg)
}

func emitSilenceYM2608(a *asm) {
	// FM key-off for all 6 melodic channels.
	for _, ch := range []uint8{0x00, 0x01, 0x02, 0x04, 0x05, 0x06} {
		writeFMPort0(a, 0x28, ch)
	}

	// Silence SSG mixer and channel levels.
	writeFMPort0(a, 0x07, 0x3F)
	writeFMPort0(a, 0x08, 0x00)
	writeFMPort0(a, 0x09, 0x00)
	writeFMPort0(a, 0x0A, 0x00)

	// Disable rhythm and ADPCM playback if any state was left active.
	writeFMPort0(a, 0x10, 0x00)
	writeFMPort1(a, 0x00, 0x01) // ADPCM-B reset/stop
}

func loadWRAM(a *asm, dst uint8, addr uint16) {
	a.movImm(4, addr)
	a.movLoad(dst, 4)
}

func storeWRAM(a *asm, addr uint16, src uint8) {
	a.movImm(4, addr)
	a.movStore(4, src)
}

func emitInitSongState(a *asm, writeStartBank uint8, writeStartOffset uint16) {
	const (
		wramSongFrameIndex = 0x0040
	)
	a.movImm(7, 0)
	storeWRAM(a, wramSongFrameIndex, 7)
}

func emitSongPlayerFrame(a *asm, frameCount uint16, countsBank uint8, countsOffset uint16, ptrBank uint8, ptrOffset uint16) {
	const (
		wramSongFrameIndex = 0x0040
	)
	writeDone := a.uniq("song_write_done")
	nextFrameDone := a.uniq("song_next_frame_done")

	// Load current frame index into R7.
	loadWRAM(a, 7, wramSongFrameIndex)

	// Look up frame write count from count table.
	a.movReg(3, 7) // R3 = frame index
	a.inst(rom.EncodeSHL(1, 3, 0))
	a.imm(1) // *2
	a.addImm(3, countsOffset)
	a.movImm(6, uint16(countsBank))
	a.setDBR(6)
	a.movLoad8(4, 3) // count lo
	a.addImm(3, 1)
	a.movLoad8(5, 3) // count hi
	a.inst(rom.EncodeSHL(1, 5, 0))
	a.imm(8)
	a.addReg(5, 4) // R5 = write count

	// Look up raw write stream bank/offset for this frame from pointer table.
	a.movReg(3, 7)
	a.inst(rom.EncodeSHL(1, 3, 0))
	a.imm(2) // *4
	a.addImm(3, ptrOffset)
	a.movImm(6, uint16(ptrBank))
	a.setDBR(6)
	a.movLoad8(6, 3) // data bank
	a.addImm(3, 1)
	a.movLoad8(0, 3) // data off lo
	a.addImm(3, 1)
	a.movLoad8(1, 3) // data off hi
	a.inst(rom.EncodeSHL(1, 1, 0))
	a.imm(8)
	a.addReg(0, 1) // R0 = data offset
	a.movImm(1, 0)
	a.setDBR(1)
	a.cmpImm(5, 0)
	a.beq(writeDone)

	// Program bus-side YM burst streamer:
	// 0x9110/11 count, 0x9112 bank, 0x9113/14 offset, 0x9115 trigger.
	a.movImm(4, 0x9110)
	a.movStore(4, 4) // count low
	a.addImm(4, 1)
	a.movStore(4, 1) // count high
	a.addImm(4, 1)
	a.movStore(4, 6) // source bank
	a.addImm(4, 1)
	a.movStore(4, 0) // source offset low
	a.addImm(4, 1)
	a.movReg(1, 0)
	a.shrImm(1, 8)
	a.movStore(4, 1) // source offset high
	a.addImm(4, 1)
	a.movImm(1, 1)
	a.movStore(4, 1) // trigger burst

	a.mark(writeDone)

	loadWRAM(a, 7, wramSongFrameIndex)
	a.addImm(7, 1)
	a.cmpImm(7, frameCount)
	a.blt(nextFrameDone)
	a.movImm(7, 0)
	a.mark(nextFrameDone)
	storeWRAM(a, wramSongFrameIndex, 7)
}

func emitInitMatrixTable(a *asm) {
	const wramMatrixTableBase = 0x0600
	for i := 0; i < 256; i++ {
		angle := float64(i) * (2.0 * math.Pi / 256.0)
		scale := 1.15
		cosv := int16(math.Round(math.Cos(angle) * scale * 256.0))
		sinv := int16(math.Round(math.Sin(angle) * scale * 256.0))
		write16s(a, wramMatrixTableBase+uint16(i*4), cosv)
		write16s(a, wramMatrixTableBase+uint16(i*4)+2, sinv)
	}
}

func emitMatrixBackgroundFrameRuntime(a *asm) {
	const wramMatrixTableBase = 0x0600
	// Use the full 16-bit frame counter and a 256-entry table.
	// Shift by 3 to slow the motion down substantially without reintroducing the earlier coarse 64-step jitter.
	a.movImm(4, 0x803F)
	a.movLoad(2, 4)
	a.shrImm(2, 3)
	a.andImm(2, 0x00FF)
	a.movReg(3, 2)
	a.inst(rom.EncodeSHL(1, 3, 0))
	a.imm(2)
	a.addImm(3, wramMatrixTableBase)
	a.movLoad(0, 3) // cos
	a.inst(rom.EncodeMOV(9, 1, 3))
	a.imm(2) // sin
	a.movImm(5, 0)
	a.subReg(5, 1)
	// Write A=cos, B=-sin, C=sin, D=cos
	a.movImm(4, 0x8019)
	a.movStore(4, 0)
	a.addImm(4, 1)
	a.movReg(6, 0)
	a.shrImm(6, 8)
	a.movStore(4, 6)
	a.addImm(4, 1)
	a.movStore(4, 5)
	a.addImm(4, 1)
	a.movReg(6, 5)
	a.shrImm(6, 8)
	a.movStore(4, 6)
	a.addImm(4, 1)
	a.movStore(4, 1)
	a.addImm(4, 1)
	a.movReg(6, 1)
	a.shrImm(6, 8)
	a.movStore(4, 6)
	a.addImm(4, 1)
	a.movStore(4, 0)
	a.addImm(4, 1)
	a.movReg(6, 0)
	a.shrImm(6, 8)
	a.movStore(4, 6)
}

func emitWaitOneFrame(a *asm, wramLastFrame uint16) {
	waitNotVBlank := a.uniq("wait_not_vblank")
	waitFrameEdge := a.uniq("wait_frame_edge")
	waitVBlank := a.uniq("wait_vblank")

	// 1) Leave current VBlank first so we can never process multiple times in one blanking window.
	a.mark(waitNotVBlank)
	a.movImm(4, 0x803E) // VBLANK_FLAG
	a.movLoad(2, 4)
	a.cmpImm(2, 0)
	a.bne(waitNotVBlank)

	// 2) Wait for frame counter LOW byte to change (hard frame-edge gate).
	a.mark(waitFrameEdge)
	a.movImm(4, 0x803F) // FRAME_COUNTER_LOW
	a.movLoad(2, 4)
	a.movImm(4, wramLastFrame)
	a.movLoad(3, 4)
	a.cmpReg(2, 3)
	a.beq(waitFrameEdge)
	a.movImm(4, wramLastFrame)
	a.movStore(4, 2)

	// 3) Wait until VBlank is asserted so OAM writes are accepted.
	a.mark(waitVBlank)
	a.movImm(4, 0x803E) // VBLANK_FLAG
	a.movLoad(2, 4)
	a.cmpImm(2, 0)
	a.beq(waitVBlank)
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

func writeSpriteFromRegs(a *asm, spriteID uint8, xReg uint8, yReg uint8, tile uint8, attr uint8, ctrl uint8) {
	write8(a, 0x8014, spriteID)
	a.movImm(4, 0x8015)
	a.movStore(4, xReg)
	a.movReg(7, xReg)
	a.shrImm(7, 8)
	a.movStore(4, 7)
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

func emitPongFrame(a *asm, titleOffset uint16, songFrameCount uint16, songCountsBank uint8, songCountsOffset uint16, songWriteStartBank uint8, songWriteStartOffset uint16) {
	const (
		wramLastFrame  = 0x0010
		wramLeftY      = 0x0020
		wramRightY     = 0x0022
		wramBallX      = 0x0024
		wramBallY      = 0x0026
		wramBallVX     = 0x0028
		wramBallVY     = 0x002A
		wramLeftScore  = 0x002C
		wramRightScore = 0x002E
		wramGameOver   = 0x0030
		wramWinner     = 0x0032
		wramTitleArmed = 0x0034
		wramAIMiss     = 0x0036
		wramAIDecided  = 0x0038
	)

	const (
		leftX            = 12
		rightX           = 244
		paddleMinY       = 24
		paddleMaxY       = 168
		ballMinY         = 4
		ballMaxY         = 184
		leftPaddleRight  = leftX + 9
		rightPaddleLeft  = rightX + 6
		rightBallTrigger = rightPaddleLeft - 15
		rightBallResetX  = rightPaddleLeft - 16
		rightScoreX      = 304
		ctrl16On         = 0x03
		paddleTile       = 8
		ballTile         = 9
	)

	emitWaitOneFrame(a, wramLastFrame)
	emitMatrixBackgroundFrameRuntime(a)

	// Read controller low byte (up/down for left paddle)
	write8(a, 0xA001, 0x01)
	a.movImm(4, 0xA000)
	a.movLoad(2, 4)
	write8(a, 0xA001, 0x00)

	// Left paddle in R0
	loadWRAM(a, 0, wramLeftY)

	checkUp := a.uniq("check_up")
	moveDone := a.uniq("move_done")
	downSkip := a.uniq("down_skip")
	upSkip := a.uniq("up_skip")

	// DOWN has priority over UP when both bits are set.
	// Decode DOWN via shift-and-test to avoid reliance on 0x0002 compare behavior.
	a.movReg(7, 2)
	a.shrImm(7, 1)
	a.andImm(7, 0x0001)
	a.cmpImm(7, 0)
	a.beq(checkUp)
	a.cmpImm(0, paddleMaxY)
	a.bge(downSkip)
	a.addImm(0, 2)
	a.mark(downSkip)
	a.jmp(moveDone)

	a.mark(checkUp)
	a.movReg(7, 2)
	a.andImm(7, 0x0001)
	a.cmpImm(7, 0)
	a.beq(upSkip)
	a.cmpImm(0, paddleMinY)
	a.ble(upSkip)
	a.subImm(0, 2)
	a.mark(upSkip)

	a.mark(moveDone)
	storeWRAM(a, wramLeftY, 0)

	// Right paddle AI in R1, ball Y in R6.
	// When a new approach starts (ball moving right), decide once whether the AI
	// intentionally misses this return, with the miss chance decreasing as the
	// match progresses.
	loadWRAM(a, 1, wramRightY)
	loadWRAM(a, 6, wramBallY)
	loadWRAM(a, 5, wramBallVX)

	aiDecisionReady := a.uniq("ai_decision_ready")
	aiDecisionDone := a.uniq("ai_decision_done")
	aiThreshold25 := a.uniq("ai_threshold_25")
	aiThreshold40 := a.uniq("ai_threshold_40")
	aiSetMiss := a.uniq("ai_set_miss")
	aiSetTrack := a.uniq("ai_set_track")
	aiStoreRightY := a.uniq("ai_store_right_y")

	a.cmpImm(5, 0)
	a.bgt(aiDecisionReady)
	a.movImm(7, 0)
	storeWRAM(a, wramAIMiss, 7)
	storeWRAM(a, wramAIDecided, 7)
	a.jmp(aiDecisionDone)

	a.mark(aiDecisionReady)
	loadWRAM(a, 7, wramAIDecided)
	a.cmpImm(7, 0)
	a.bne(aiDecisionDone)

	a.movImm(4, 0x803F)
	a.movLoad(7, 4)
	a.addReg(7, 6)
	loadWRAM(a, 3, wramLeftScore)
	loadWRAM(a, 4, wramRightScore)
	a.addReg(7, 3)
	a.addReg(7, 4)
	a.andImm(7, 0x00FF)
	a.addReg(4, 3) // total points so far
	a.cmpImm(4, 0)
	a.beq(aiSetMiss) // fall through to 1-in-10 check using threshold below
	a.cmpImm(4, 1)
	a.beq(aiThreshold25)
	a.jmp(aiThreshold40)

	a.mark(aiSetMiss) // about 1 in 10
	a.cmpImm(7, 26)
	a.blt(aiSetMiss + "_store")
	a.jmp(aiSetTrack)

	a.mark(aiThreshold25) // about 1 in 25
	a.cmpImm(7, 10)
	a.blt(aiSetMiss + "_store")
	a.jmp(aiSetTrack)

	a.mark(aiThreshold40) // about 1 in 40
	a.cmpImm(7, 6)
	a.blt(aiSetMiss + "_store")
	a.jmp(aiSetTrack)

	a.mark(aiSetMiss + "_store")
	a.movImm(7, 1)
	storeWRAM(a, wramAIMiss, 7)
	a.movImm(7, 1)
	storeWRAM(a, wramAIDecided, 7)
	a.jmp(aiDecisionDone)

	a.mark(aiSetTrack)
	a.movImm(7, 0)
	storeWRAM(a, wramAIMiss, 7)
	a.movImm(7, 1)
	storeWRAM(a, wramAIDecided, 7)
	a.mark(aiDecisionDone)

	loadWRAM(a, 7, wramAIMiss)
	aiTrackNormally := a.uniq("ai_track_normally")
	a.cmpImm(7, 0)
	a.beq(aiTrackNormally)

	missDownSkip := a.uniq("ai_miss_down_skip")
	a.movReg(7, 1)
	a.addImm(7, 8)
	a.cmpReg(7, 6)
	a.ble(missDownSkip)
	a.cmpImm(1, paddleMaxY)
	a.bge(missDownSkip)
	a.addImm(1, 1)
	a.mark(missDownSkip)

	missUpSkip := a.uniq("ai_miss_up_skip")
	a.movReg(7, 1)
	a.addImm(7, 8)
	a.cmpReg(7, 6)
	a.bge(missUpSkip)
	a.cmpImm(1, paddleMinY)
	a.ble(missUpSkip)
	a.subImm(1, 1)
	a.mark(missUpSkip)
	a.jmp(aiStoreRightY)

	a.mark(aiTrackNormally)
	aiDownSkip := a.uniq("ai_down_skip")
	a.movReg(7, 1)
	a.addImm(7, 8)
	a.cmpReg(7, 6)
	a.bge(aiDownSkip)
	a.cmpImm(1, paddleMaxY)
	a.bge(aiDownSkip)
	a.addImm(1, 1)
	a.mark(aiDownSkip)

	aiUpSkip := a.uniq("ai_up_skip")
	a.movReg(7, 6)
	a.addImm(7, 8)
	a.cmpReg(1, 7)
	a.ble(aiUpSkip)
	a.cmpImm(1, paddleMinY)
	a.ble(aiUpSkip)
	a.subImm(1, 1)
	a.mark(aiUpSkip)

	a.mark(aiStoreRightY)
	storeWRAM(a, wramRightY, 1)

	// If game over, skip movement and collisions.
	movementDone := a.uniq("movement_done")
	loadWRAM(a, 7, wramGameOver)
	a.cmpImm(7, 0)
	a.bne(movementDone)

	// Ball position and velocity
	loadWRAM(a, 3, wramBallX)
	loadWRAM(a, 6, wramBallY)
	loadWRAM(a, 0, wramBallVX)
	loadWRAM(a, 1, wramBallVY)
	a.addReg(3, 0)
	a.addReg(6, 1)

	// Y bounce top
	topDone := a.uniq("top_done")
	a.cmpImm(6, ballMinY)
	a.bge(topDone)
	a.movImm(6, ballMinY)
	a.movImm(7, 0)
	a.subReg(7, 1)
	a.movReg(1, 7)
	a.mark(topDone)

	// Y bounce bottom
	bottomDone := a.uniq("bottom_done")
	a.cmpImm(6, ballMaxY)
	a.ble(bottomDone)
	a.movImm(6, ballMaxY)
	a.movImm(7, 0)
	a.subReg(7, 1)
	a.movReg(1, 7)
	a.mark(bottomDone)

	// Left paddle collision against the visible 4-pixel paddle strip inside the 16x16 sprite.
	leftColDone := a.uniq("left_col_done")
	a.cmpImm(0, 0)
	a.bge(leftColDone)
	a.cmpImm(3, leftPaddleRight)
	a.bgt(leftColDone)
	loadWRAM(a, 7, wramLeftY)
	a.movReg(5, 7)
	a.addImm(5, 15)
	a.cmpReg(6, 5)
	a.bgt(leftColDone)
	a.movReg(5, 6)
	a.addImm(5, 15)
	a.cmpReg(5, 7)
	a.blt(leftColDone)
	a.movImm(3, leftPaddleRight+1)
	a.movImm(7, 0)
	a.subReg(7, 0)
	a.movReg(0, 7)
	a.mark(leftColDone)

	// Right paddle collision against the visible 4-pixel paddle strip inside the 16x16 sprite.
	rightColDone := a.uniq("right_col_done")
	a.cmpImm(0, 0)
	a.ble(rightColDone)
	a.cmpImm(3, rightBallTrigger)
	a.blt(rightColDone)
	loadWRAM(a, 7, wramRightY)
	a.movReg(5, 7)
	a.addImm(5, 15)
	a.cmpReg(6, 5)
	a.bgt(rightColDone)
	a.movReg(5, 6)
	a.addImm(5, 15)
	a.cmpReg(5, 7)
	a.blt(rightColDone)
	a.movImm(3, rightBallResetX)
	a.movImm(7, 0)
	a.subReg(7, 0)
	a.movReg(0, 7)
	a.mark(rightColDone)

	// Left out => right scores
	leftOutDone := a.uniq("left_out_done")
	a.cmpImm(3, 0)
	a.bge(leftOutDone)
	loadWRAM(a, 5, wramRightScore)
	a.addImm(5, 1)
	storeWRAM(a, wramRightScore, 5)
	a.movImm(3, 152)
	a.movImm(6, 92)
	a.movImm(0, 2)
	a.movImm(1, 1)
	a.cmpImm(5, 3)
	blow := a.uniq("right_not_win")
	a.blt(blow)
	a.movImm(7, 1)
	storeWRAM(a, wramGameOver, 7)
	a.movImm(7, 2)
	storeWRAM(a, wramWinner, 7)
	a.mark(blow)
	a.mark(leftOutDone)

	// Right out => left scores
	rightOutDone := a.uniq("right_out_done")
	a.cmpImm(3, rightScoreX)
	a.ble(rightOutDone)
	loadWRAM(a, 5, wramLeftScore)
	a.addImm(5, 1)
	storeWRAM(a, wramLeftScore, 5)
	a.movImm(3, 152)
	a.movImm(6, 92)
	a.movImm(0, 0xFFFE)
	a.movImm(1, 0xFFFF)
	alow := a.uniq("left_not_win")
	a.cmpImm(5, 3)
	a.blt(alow)
	a.movImm(7, 1)
	storeWRAM(a, wramGameOver, 7)
	a.movImm(7, 1)
	storeWRAM(a, wramWinner, 7)
	a.mark(alow)
	a.mark(rightOutDone)

	storeWRAM(a, wramBallX, 3)
	storeWRAM(a, wramBallY, 6)
	storeWRAM(a, wramBallVX, 0)
	storeWRAM(a, wramBallVY, 1)

	a.mark(movementDone)

	// Draw sprites.
	loadWRAM(a, 0, wramLeftY)
	loadWRAM(a, 1, wramRightY)
	loadWRAM(a, 3, wramBallX)
	loadWRAM(a, 6, wramBallY)
	writeSpriteImm(a, 0, leftX, 0, paddleTile, 0x01, ctrl16On)
	writeSpriteImm(a, 1, rightX, 1, paddleTile, 0x02, ctrl16On)
	writeSpriteFromRegs(a, 2, 3, 6, ballTile, 0x03, ctrl16On)

	// Disable all remaining sprite slots to keep the scene unambiguous.
	for i := 3; i <= 12; i++ {
		clearSprite(a, uint8(i))
	}

	loadWRAM(a, 6, wramLeftScore)
	emitDigit(a, 136, 20, 255, 255, 255, 6)
	loadWRAM(a, 6, wramRightScore)
	emitDigit(a, 176, 20, 120, 220, 255, 6)
	emitText(a, 156, 20, 200, 200, 200, "-")

	// Winner flash backdrop after game over
	loadWRAM(a, 7, wramGameOver)
	goDone := a.uniq("go_flash_done")
	a.cmpImm(7, 0)
	a.beq(goDone)
	emitText(a, 116, 90, 255, 255, 255, "GAME OVER")
	emitText(a, 88, 110, 255, 220, 120, "PRESS START")
	loadWRAM(a, 7, wramWinner)
	leftWin := a.uniq("left_win")
	a.cmpImm(7, 1)
	a.beq(leftWin)
	// Right winner backdrop tint
	write8(a, 0x8012, 0x00)
	write8(a, 0x8013, 0x1F)
	write8(a, 0x8013, 0x00)
	a.jmp(goDone)
	a.mark(leftWin)
	write8(a, 0x8012, 0x00)
	write8(a, 0x8013, 0x00)
	write8(a, 0x8013, 0x7C)
	a.mark(goDone)

	// Return to title from game over on START or A.
	restartDone := a.uniq("restart_done")
	loadWRAM(a, 7, wramGameOver)
	a.cmpImm(7, 0)
	a.beq(restartDone)
	write8(a, 0xA001, 0x01)
	a.movImm(4, 0xA000)
	a.movLoad(2, 4)
	a.movImm(4, 0xA001)
	a.movLoad(3, 4)
	write8(a, 0xA001, 0x00)
	a.movReg(6, 2)
	a.andImm(6, 0x0010)
	a.cmpImm(6, 0x0010)
	restartTitle := a.uniq("restart_title")
	a.beq(restartTitle)
	a.movReg(5, 3)
	a.andImm(5, 0x0004)
	a.cmpImm(5, 0x0004)
	a.beq(restartTitle)
	a.jmp(restartDone)
	a.mark(restartTitle)
	a.movImm(7, 0)
	storeWRAM(a, wramTitleArmed, 7)
	emitFarJumpTo(a, uint8(rom.ROMMinProgramBank), titleOffset)
	a.mark(restartDone)

	emitSongPlayerFrame(a, songFrameCount, songCountsBank, songCountsOffset, songWriteStartBank, songWriteStartOffset)
}

func emitFarJumpTo(a *asm, bank uint8, offset uint16) {
	a.movImm(5, uint16(bank))
	a.inst(rom.EncodeMOV(4, 5, 0))
	a.movImm(5, offset)
	a.inst(rom.EncodeMOV(4, 5, 0))
	a.movImm(5, 0x0000)
	a.inst(rom.EncodeMOV(4, 5, 0))
	a.inst(rom.EncodeRET())
}

func emitFarJumpToBank(a *asm, bank uint8) {
	emitFarJumpTo(a, bank, 0x8000)
}

func initGraphicsAndGame(a *asm, songWriteStartBank uint8, songWriteStartOffset uint16) {
	const (
		wramLastFrame  = 0x0010
		wramLeftY      = 0x0020
		wramRightY     = 0x0022
		wramBallX      = 0x0024
		wramBallY      = 0x0026
		wramBallVX     = 0x0028
		wramBallVY     = 0x002A
		wramLeftScore  = 0x002C
		wramRightScore = 0x002E
		wramGameOver   = 0x0030
		wramWinner     = 0x0032
		wramTitleArmed = 0x0034
		wramAIMiss     = 0x0036
		wramAIDecided  = 0x0038
	)

	const (
		bgCourtTile   = 0
		paddleTile    = 8
		ballTile      = 9
		tilemapBase   = 0x4000
	)

	// Backdrop + BG palette.
	setCGRAMColor(a, 0x00, 0x0000)
	setCGRAMColor(a, 0x01, 0x0820) // dark navy
	setCGRAMColor(a, 0x02, 0x14A8) // bright blue
	setCGRAMColor(a, 0x03, 0x7FFF) // white

	// Sprite palettes.
	setCGRAMColor(a, 0x11, 0x7FFF) // P1 paddle white
	setCGRAMColor(a, 0x21, 0x03FF) // P2 paddle cyan
	setCGRAMColor(a, 0x31, 0x7FFF) // Ball outline
	setCGRAMColor(a, 0x32, 0x7FE0) // Ball fill

	// BG + sprite art.
	writeVRAMBlock(a, uint16(bgCourtTile)*32, buildSolidTile(8, 1))
	writeVRAMBlock(a, uint16(bgCourtTile+1)*32, buildSolidTile(8, 2))
	writeVRAMBlock(a, uint16(paddleTile)*128, buildPaddleTile())
	writeVRAMBlock(a, uint16(ballTile)*128, buildBallTile())

	// 128x128 tilemap arranged as 4x4 tile blocks, yielding 32x32 pixel checker cells.
	emitCheckerTilemapRuntime(a, tilemapBase, 128, 2, bgCourtTile, bgCourtTile+1)

	// Bind BG0 to an 8x8 matrix-backed playfield with a 128x128 tile source.
	write16(a, 0x8077, tilemapBase)
	write8(a, 0x8008, 0x21) // BG0 enable + 8x8 tiles + 128x128 tilemap
	write8(a, 0x806C, 0x00) // BG0 -> transform channel 0
	write8(a, 0x8068, 0x00) // tilemap source
	write8(a, 0x8018, 0x01) // matrix mode enabled on BG0
	write16s(a, 0x8000, 256)
	write16s(a, 0x8002, 256)
	write16s(a, 0x8019, 0x0100)
	write16s(a, 0x801B, 0)
	write16s(a, 0x801D, 0)
	write16s(a, 0x801F, 0x0100)
	write16s(a, 0x8027, 160)
	write16s(a, 0x8029, 100)
	emitInitMatrixTable(a)

	// YM/FM host setup
	write8(a, 0x9020, 0xC0)
	write8(a, 0x9103, 0x81)
	write8(a, 0x9103, 0x01)

	// Game state
	// Keep these values centralized so title/start can reset cleanly before match.
	a.movImm(7, 92)
	storeWRAM(a, wramLeftY, 7)
	storeWRAM(a, wramRightY, 7)
	a.movImm(7, 152)
	storeWRAM(a, wramBallX, 7)
	a.movImm(7, 92)
	storeWRAM(a, wramBallY, 7)
	a.movImm(7, 2)
	storeWRAM(a, wramBallVX, 7)
	a.movImm(7, 1)
	storeWRAM(a, wramBallVY, 7)
	a.movImm(7, 0)
	storeWRAM(a, wramLeftScore, 7)
	storeWRAM(a, wramRightScore, 7)
	storeWRAM(a, wramGameOver, 7)
	storeWRAM(a, wramWinner, 7)
	a.movImm(7, 1)
	storeWRAM(a, wramTitleArmed, 7)
	a.movImm(7, 0)
	storeWRAM(a, wramAIMiss, 7)
	storeWRAM(a, wramAIDecided, 7)
	emitInitSongState(a, songWriteStartBank, songWriteStartOffset)

	// Frame baseline
	a.movImm(4, 0x803F)
	a.movLoad(2, 4)
	a.movImm(4, wramLastFrame)
	a.movStore(4, 2)
}

func resetGameStateForMatchStart(a *asm, songWriteStartBank uint8, songWriteStartOffset uint16) {
	const (
		wramLeftY      = 0x0020
		wramRightY     = 0x0022
		wramBallX      = 0x0024
		wramBallY      = 0x0026
		wramBallVX     = 0x0028
		wramBallVY     = 0x002A
		wramLeftScore  = 0x002C
		wramRightScore = 0x002E
		wramGameOver   = 0x0030
		wramWinner     = 0x0032
		wramTitleArmed = 0x0034
		wramAIMiss     = 0x0036
		wramAIDecided  = 0x0038
	)
	a.movImm(7, 92)
	storeWRAM(a, wramLeftY, 7)
	storeWRAM(a, wramRightY, 7)
	a.movImm(7, 152)
	storeWRAM(a, wramBallX, 7)
	a.movImm(7, 92)
	storeWRAM(a, wramBallY, 7)
	a.movImm(7, 2)
	storeWRAM(a, wramBallVX, 7)
	a.movImm(7, 1)
	storeWRAM(a, wramBallVY, 7)
	a.movImm(7, 0)
	storeWRAM(a, wramLeftScore, 7)
	storeWRAM(a, wramRightScore, 7)
	storeWRAM(a, wramGameOver, 7)
	storeWRAM(a, wramWinner, 7)
	a.movImm(7, 1)
	storeWRAM(a, wramTitleArmed, 7)
	a.movImm(7, 0)
	storeWRAM(a, wramAIMiss, 7)
	storeWRAM(a, wramAIDecided, 7)
	emitInitSongState(a, songWriteStartBank, songWriteStartOffset)
}

func buildPongROM(song *vgmSong, outPath string, songFrames int) error {
	if songFrames > 0 && songFrames < len(song.frames) {
		song.frames = song.frames[:songFrames]
	}
	if len(song.frames) == 0 {
		return errors.New("no frames to emit")
	}

	countBytes, writeBytes, _ := encodeFrameTable(song)
	const songCountsBank = uint8(2)
	const songCountsOffset = uint16(0x8000)
	const songPtrBank = uint8(2)
	ptrOffset := uint16(int(songCountsOffset) + len(countBytes))
	writeStartAbs := int(ptrOffset) + len(song.frames)*4
	songWriteStartBank := songCountsBank + uint8((writeStartAbs-0x8000)/rom.ROMBankSizeBytes)
	songWriteStartOffset := uint16(0x8000 + ((writeStartAbs - 0x8000) % rom.ROMBankSizeBytes))
	ptrBytes := encodeFramePointers(song, songWriteStartBank, songWriteStartOffset)

	banked := rom.NewBankedROMBuilder()
	a := newASM()
	initGraphicsAndGame(a, songWriteStartBank, songWriteStartOffset)

	titleLoop := a.uniq("title_loop")
	startMatch := a.uniq("start_match")
	gameLoop := a.uniq("game_loop")
	titleOffset := uint16(0x8002)

	a.mark(titleLoop)
	titleOffset = 0x8002 + a.pc()
	emitWaitOneFrame(a, 0x0010)
	emitSilenceYM2608(a)
	setCGRAMColor(a, 0x00, 0x0000)
	emitText(a, 78, 58, 255, 255, 255, "NITRO PONG DX")
	emitText(a, 64, 78, 120, 220, 255, "YM2608 SHOWCASE MATCH")
	emitText(a, 88, 104, 255, 255, 255, "PRESS START")
	emitText(a, 56, 122, 180, 220, 255, "UP / DOWN TO MOVE PADDLE")
	emitText(a, 76, 140, 180, 220, 255, "FIRST TO 3 POINTS")
	emitText(a, 64, 176, 255, 220, 120, "ROTATING MATRIX BACKDROP")
	for i := 0; i <= 12; i++ {
		clearSprite(a, uint8(i))
	}
	write8(a, 0xA001, 0x01)
	a.movImm(4, 0xA000)
	a.movLoad(2, 4)
	a.movImm(4, 0xA001)
	a.movLoad(3, 4)
	write8(a, 0xA001, 0x00)
	loadWRAM(a, 7, 0x0034)
	titleReady := a.uniq("title_ready")
	titleKeepWaiting := a.uniq("title_keep_waiting")
	a.cmpImm(7, 0)
	a.bne(titleReady)
	a.movReg(6, 2)
	a.andImm(6, 0x0010)
	a.cmpImm(6, 0)
	a.bne(titleKeepWaiting)
	a.movReg(5, 3)
	a.andImm(5, 0x0004)
	a.cmpImm(5, 0)
	a.bne(titleKeepWaiting)
	a.movImm(7, 1)
	storeWRAM(a, 0x0034, 7)
	a.jmp(titleLoop)
	a.mark(titleKeepWaiting)
	a.jmp(titleLoop)
	a.mark(titleReady)
	a.movReg(6, 2)
	a.andImm(6, 0x0010)
	a.cmpImm(6, 0x0010)
	a.beq(startMatch)
	a.movReg(5, 3)
	a.andImm(5, 0x0004)
	a.cmpImm(5, 0x0004)
	a.beq(startMatch)
	a.jmp(titleLoop)

	a.mark(startMatch)
	resetGameStateForMatchStart(a, songWriteStartBank, songWriteStartOffset)
	a.mark(gameLoop)
	emitPongFrame(a, titleOffset, uint16(len(song.frames)), songCountsBank, songCountsOffset, songPtrBank, ptrOffset)
	a.jmp(gameLoop)

	if err := a.resolve(); err != nil {
		return fmt.Errorf("resolve code bank: %w", err)
	}
	words, err := extractBuilderWords(a.b)
	if err != nil {
		return fmt.Errorf("extract code bank: %w", err)
	}
	if len(words) > rom.ROMBankSizeWords {
		return fmt.Errorf("code bank overflow: %d words > %d", len(words), rom.ROMBankSizeWords)
	}
	banked.AddInstruction(1, rom.EncodeRET()) // IRQ/NMI trampoline
	for _, w := range words {
		banked.AddInstruction(1, w)
	}

	dataBlob := append(countBytes, ptrBytes...)
	dataBlob = append(dataBlob, writeBytes...)
	for i := 0; i < len(dataBlob); i += 2 {
		bank := uint8(int(songCountsBank) + (i / rom.ROMBankSizeBytes))
		if bank > rom.ROMMaxProgramBank {
			return fmt.Errorf("song data exceeds ROM bank budget at bank %d", bank)
		}
		word := uint16(dataBlob[i])
		if i+1 < len(dataBlob) {
			word |= uint16(dataBlob[i+1]) << 8
		}
		banked.AddInstruction(bank, word)
	}

	return banked.BuildROM(1, 0x8002, outPath)
}

func main() {
	inPath := flag.String("in", "Resources/Demo.vgz", "Input VGM/VGZ file")
	outPath := flag.String("out", "roms/pong_ym2608_demo.rom", "Output ROM path")
	songFrames := flag.Int("song-frames", 0, "Song frame cap for Pong BGM (0 = full song)")
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

	if err := buildPongROM(song, *outPath, *songFrames); err != nil {
		fmt.Fprintf(os.Stderr, "build ROM: %v\n", err)
		os.Exit(1)
	}

	effectiveFrames := len(song.frames)
	if *songFrames > 0 && *songFrames < effectiveFrames {
		effectiveFrames = *songFrames
	}
	selectedWrites := 0
	for i, fw := range song.frames {
		if i >= effectiveFrames {
			break
		}
		selectedWrites += len(fw)
	}
	seconds := float64(effectiveFrames*int(song.frameSamples)) / 44100.0
	countBytes, writeBytes, totalWrites := encodeFrameTable(song)
	ptrBytes := make([]byte, len(song.frames)*4)
	dataBytes := len(countBytes) + len(ptrBytes) + len(writeBytes)
	dataBanks := (dataBytes + rom.ROMBankSizeBytes - 1) / rom.ROMBankSizeBytes
	fmt.Printf("Built %s\n", *outPath)
	fmt.Printf("Pong mode: first to 3 points (left paddle player, right paddle AI)\n")
	fmt.Printf("Frames: %d  YM writes source: %d  Approx music duration: %.2fs\n",
		effectiveFrames, totalWrites, seconds)
	fmt.Printf("Frame table bytes: %d (counts=%d ptrs=%d writes=%d)  Estimated data banks used: %d\n",
		dataBytes, len(countBytes), len(ptrBytes), len(writeBytes), dataBanks)
}
