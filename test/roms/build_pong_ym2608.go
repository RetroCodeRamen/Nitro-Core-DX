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
func (a *asm) movReg(dst, src uint8)       { a.inst(rom.EncodeMOV(0, dst, src)) }
func (a *asm) movLoad(dst, addrReg uint8)  { a.inst(rom.EncodeMOV(2, dst, addrReg)) }
func (a *asm) movStore(addrReg, src uint8) { a.inst(rom.EncodeMOV(3, addrReg, src)) }
func (a *asm) addImm(reg uint8, v uint16)  { a.inst(rom.EncodeADD(1, reg, 0)); a.imm(v) }
func (a *asm) addReg(dst, src uint8)       { a.inst(rom.EncodeADD(0, dst, src)) }
func (a *asm) subImm(reg uint8, v uint16)  { a.inst(rom.EncodeSUB(1, reg, 0)); a.imm(v) }
func (a *asm) subReg(dst, src uint8)       { a.inst(rom.EncodeSUB(0, dst, src)) }
func (a *asm) andImm(reg uint8, v uint16)  { a.inst(rom.EncodeAND(1, reg, 0)); a.imm(v) }
func (a *asm) cmpImm(reg uint8, v uint16) {
	// CPU decode currently aliases CMP-immediate mode with BEQ encoding when reg1=0 and reg2=0.
	// Encode reg2 as non-zero for R0 compares so mode=1 is treated as CMP immediate, not BEQ.
	reg2 := uint8(0)
	if reg == 0 {
		reg2 = 1
	}
	a.inst(rom.EncodeCMP(1, reg, reg2))
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

func writeFMPort0(a *asm, addr, data uint8) {
	write8(a, 0x9100, addr)
	write8(a, 0x9101, data)
}

func writeFMPort1(a *asm, addr, data uint8) {
	write8(a, 0x9104, addr)
	write8(a, 0x9105, data)
}

func loadWRAM(a *asm, dst uint8, addr uint16) {
	a.movImm(4, addr)
	a.movLoad(dst, 4)
}

func storeWRAM(a *asm, addr uint16, src uint8) {
	a.movImm(4, addr)
	a.movStore(4, src)
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

func emitPongFrame(a *asm, frameWrites []ymWrite, playSong bool) {
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
	)

	const (
		leftX       = 12
		rightX      = 292
		paddleMinY  = 24
		paddleMaxY  = 168
		ballMinY    = 4
		ballMaxY    = 184
		leftColX    = 24
		rightColX   = 280
		rightScoreX = 304
		ctrl16On    = 0x03
	)

	emitWaitOneFrame(a, wramLastFrame)

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

	// Right paddle AI in R1, ball Y in R6
	loadWRAM(a, 1, wramRightY)
	loadWRAM(a, 6, wramBallY)

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

	// Left paddle collision (when vx < 0 and x <= leftColX)
	leftColDone := a.uniq("left_col_done")
	a.cmpImm(0, 0)
	a.bge(leftColDone)
	a.cmpImm(3, leftColX)
	a.bgt(leftColDone)
	loadWRAM(a, 7, wramLeftY)
	a.movReg(5, 7)
	a.subImm(5, 24)
	a.cmpReg(6, 5)
	a.blt(leftColDone)
	a.movReg(5, 7)
	a.addImm(5, 24)
	a.cmpReg(6, 5)
	a.bgt(leftColDone)
	a.movImm(3, leftColX)
	a.movImm(7, 0)
	a.subReg(7, 0)
	a.movReg(0, 7)
	a.mark(leftColDone)

	// Right paddle collision (when vx > 0 and x >= rightColX)
	rightColDone := a.uniq("right_col_done")
	a.cmpImm(0, 0)
	a.ble(rightColDone)
	a.cmpImm(3, rightColX)
	a.blt(rightColDone)
	loadWRAM(a, 7, wramRightY)
	a.movReg(5, 7)
	a.subImm(5, 24)
	a.cmpReg(6, 5)
	a.blt(rightColDone)
	a.movReg(5, 7)
	a.addImm(5, 24)
	a.cmpReg(6, 5)
	a.bgt(rightColDone)
	a.movImm(3, rightColX)
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
	// Each paddle uses 3 stacked 16x16 sprites to form a vertical line paddle.
	loadWRAM(a, 0, wramLeftY)
	loadWRAM(a, 1, wramRightY)
	loadWRAM(a, 3, wramBallX)
	loadWRAM(a, 6, wramBallY)
	a.movReg(5, 0)
	a.subImm(5, 16)
	writeSpriteImm(a, 0, leftX, 5, 1, 0x01, ctrl16On)
	writeSpriteImm(a, 1, leftX, 0, 1, 0x01, ctrl16On)
	a.movReg(5, 0)
	a.addImm(5, 16)
	writeSpriteImm(a, 2, leftX, 5, 1, 0x01, ctrl16On)

	a.movReg(5, 1)
	a.subImm(5, 16)
	writeSpriteImm(a, 3, rightX, 5, 1, 0x01, ctrl16On)
	writeSpriteImm(a, 4, rightX, 1, 1, 0x01, ctrl16On)
	a.movReg(5, 1)
	a.addImm(5, 16)
	writeSpriteImm(a, 5, rightX, 5, 1, 0x01, ctrl16On)

	writeSpriteFromRegs(a, 6, 3, 6, 2, 0x01, ctrl16On)

	// Left score pips (sprites 7..9)
	loadWRAM(a, 5, wramLeftScore)
	for i := 0; i < 3; i++ {
		draw := a.uniq("lp_draw")
		done := a.uniq("lp_done")
		a.cmpImm(5, uint16(i+1))
		a.bge(draw)
		clearSprite(a, uint8(7+i))
		a.jmp(done)
		a.mark(draw)
		a.movImm(7, 10)
		writeSpriteImm(a, uint8(7+i), uint16(100+(i*20)), 7, 3, 0x01, ctrl16On)
		a.mark(done)
	}

	// Right score pips (sprites 10..12)
	loadWRAM(a, 5, wramRightScore)
	for i := 0; i < 3; i++ {
		draw := a.uniq("rp_draw")
		done := a.uniq("rp_done")
		a.cmpImm(5, uint16(i+1))
		a.bge(draw)
		clearSprite(a, uint8(10+i))
		a.jmp(done)
		a.mark(draw)
		a.movImm(7, 10)
		writeSpriteImm(a, uint8(10+i), uint16(180+(i*20)), 7, 3, 0x01, ctrl16On)
		a.mark(done)
	}

	// Winner flash backdrop after game over
	loadWRAM(a, 7, wramGameOver)
	goDone := a.uniq("go_flash_done")
	a.cmpImm(7, 0)
	a.beq(goDone)
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

	if playSong {
		for _, w := range frameWrites {
			if w.port == 0 {
				writeFMPort0(a, w.addr, w.data)
			} else {
				writeFMPort1(a, w.addr, w.data)
			}
		}
	}
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

func initGraphicsAndGame(a *asm) {
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
	)

	// Basic palette/background
	write8(a, 0x8012, 0x00)
	write8(a, 0x8013, 0x00)
	write8(a, 0x8013, 0x00)
	write8(a, 0x8008, 0x00) // BG off

	// Sprite palette entries
	// P1 white
	write8(a, 0x8012, 0x11)
	write8(a, 0x8013, 0xFF)
	write8(a, 0x8013, 0x7F)
	// P2 cyan
	write8(a, 0x8012, 0x21)
	write8(a, 0x8013, 0xFF)
	write8(a, 0x8013, 0x03)
	// P3 yellow
	write8(a, 0x8012, 0x31)
	write8(a, 0x8013, 0xE0)
	write8(a, 0x8013, 0x7F)

	// Tile 1: solid 16x16 for paddles (0x11)
	write8(a, 0x800E, 0x20)
	write8(a, 0x800F, 0x00)
	a.movImm(6, 128)
	fill1 := a.uniq("fill_tile1")
	a.mark(fill1)
	write8(a, 0x8010, 0x11)
	a.subImm(6, 1)
	a.movImm(7, 0)
	a.cmpReg(6, 7)
	a.bne(fill1)

	// Tile 2: solid for ball (0x11 using palette 3)
	write8(a, 0x800E, 0x40)
	write8(a, 0x800F, 0x00)
	a.movImm(6, 128)
	fill2 := a.uniq("fill_tile2")
	a.mark(fill2)
	write8(a, 0x8010, 0x11)
	a.subImm(6, 1)
	a.movImm(7, 0)
	a.cmpReg(6, 7)
	a.bne(fill2)

	// Tile 3: score pip
	write8(a, 0x800E, 0x60)
	write8(a, 0x800F, 0x00)
	a.movImm(6, 128)
	fill3 := a.uniq("fill_tile3")
	a.mark(fill3)
	write8(a, 0x8010, 0x11)
	a.subImm(6, 1)
	a.movImm(7, 0)
	a.cmpReg(6, 7)
	a.bne(fill3)

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

	// Frame baseline
	a.movImm(4, 0x803F)
	a.movLoad(2, 4)
	a.movImm(4, wramLastFrame)
	a.movStore(4, 2)
}

func resetGameStateForMatchStart(a *asm) {
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
}

func buildPongROM(song *vgmSong, outPath string, framesPerBank int, songFrames int) error {
	if songFrames > 0 && songFrames < len(song.frames) {
		song.frames = song.frames[:songFrames]
	}
	if framesPerBank <= 0 {
		framesPerBank = 12
	}
	if len(song.frames) == 0 {
		return errors.New("no frames to emit")
	}

	banksNeeded := (len(song.frames) + framesPerBank - 1) / framesPerBank
	if banksNeeded > (rom.ROMMaxProgramBank - rom.ROMMinProgramBank + 1) {
		return fmt.Errorf("song requires %d banks with frames-per-bank=%d (max=%d)",
			banksNeeded, framesPerBank, rom.ROMMaxProgramBank-rom.ROMMinProgramBank+1)
	}

	banked := rom.NewBankedROMBuilder()
	loopBank := uint8(rom.ROMMinProgramBank)
	loopOffset := uint16(0x8002)

	for seg := 0; seg < banksNeeded; seg++ {
		start := seg * framesPerBank
		end := start + framesPerBank
		if end > len(song.frames) {
			end = len(song.frames)
		}
		bank := uint8(rom.ROMMinProgramBank + seg)
		isFirst := seg == 0
		isLastSongBank := end == len(song.frames)

		a := newASM()
		if isFirst {
			initGraphicsAndGame(a)

			// Title/start gate: no song writes until START is pressed.
			titleLoop := a.uniq("title_loop")
			startMatch := a.uniq("start_match")
			a.mark(titleLoop)
			emitWaitOneFrame(a, 0x0010)
			emitText(a, 88, 88, 255, 255, 255, "PRESS START")
			emitText(a, 72, 104, 120, 200, 255, "FIRST TO 3 WINS")
			// Latch input and read high byte for START (bit2).
			write8(a, 0xA001, 0x01)
			a.movImm(4, 0xA000)
			a.movLoad(2, 4)
			a.movImm(4, 0xA001)
			a.movLoad(3, 4)
			write8(a, 0xA001, 0x00)
			a.movReg(5, 3)
			a.andImm(5, 0x0004)
			a.cmpImm(5, 0x0004)
			a.beq(startMatch)
			a.jmp(titleLoop)

			a.mark(startMatch)
			resetGameStateForMatchStart(a)
			// Bank 1 has a RET trampoline inserted at 0x8000, so code starts at 0x8002.
			// Loop target is the first frame's game+music update code after start.
			loopOffset = 0x8002 + a.pc()
		}

		for _, fw := range song.frames[start:end] {
			emitPongFrame(a, fw, true)
		}

		if isLastSongBank {
			// Loop BGM segment during gameplay without reinitializing game state.
			emitFarJumpTo(a, loopBank, loopOffset)
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
			// IRQ/NMI trampoline
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
	outPath := flag.String("out", "roms/pong_ym2608_demo.rom", "Output ROM path")
	framesPerBank := flag.Int("frames-per-bank", 12, "Frames per program bank segment")
	songFrames := flag.Int("song-frames", 1500, "Song frame cap for Pong BGM (0 = full, may exceed bank budget)")
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

	if err := buildPongROM(song, *outPath, *framesPerBank, *songFrames); err != nil {
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
	fmt.Printf("Built %s\n", *outPath)
	fmt.Printf("Pong mode: first to 3 points (left paddle player, right paddle AI)\n")
	fmt.Printf("Frames: %d  YM writes emitted: %d  Approx music duration: %.2fs\n",
		effectiveFrames, selectedWrites, seconds)
	fmt.Printf("Frames per bank: %d  Estimated banks used: %d\n",
		*framesPerBank, (effectiveFrames+*framesPerBank-1)/(*framesPerBank))
}
