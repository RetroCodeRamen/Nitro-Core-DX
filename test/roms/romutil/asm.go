//go:build testrom_tools
// +build testrom_tools

package romutil

import (
	"fmt"
	"math"

	"nitro-core-dx/internal/rom"
)

type patchRef struct {
	wordIndex int
	currentPC uint16
	target    string
}

type Asm struct {
	B       *rom.BankedROMBuilder
	bank    uint8
	labels  map[string]uint16
	patches []patchRef
	uniqID  int
}

func NewASM(bank uint8) *Asm {
	return &Asm{
		B:      rom.NewBankedROMBuilder(),
		bank:   bank,
		labels: make(map[string]uint16),
	}
}

func (a *Asm) PC() uint16                { return a.B.PC(a.bank) }
func (a *Asm) Mark(name string)          { a.labels[name] = a.PC() }
func (a *Asm) Inst(w uint16)             { a.B.AddInstruction(a.bank, w) }
func (a *Asm) Imm(v uint16)              { a.B.AddImmediate(a.bank, v) }
func (a *Asm) Uniq(prefix string) string { a.uniqID++; return fmt.Sprintf("%s_%d", prefix, a.uniqID) }

func (a *Asm) MovImm(reg uint8, v uint16)  { a.Inst(rom.EncodeMOV(1, reg, 0)); a.Imm(v) }
func (a *Asm) MovReg(dst, src uint8)       { a.Inst(rom.EncodeMOV(0, dst, src)) }
func (a *Asm) MovLoad(dst, addrReg uint8)  { a.Inst(rom.EncodeMOV(2, dst, addrReg)) }
func (a *Asm) MovLoad8(dst, addrReg uint8) { a.Inst(rom.EncodeMOV(6, dst, addrReg)) }
func (a *Asm) MovStore(addrReg, src uint8) { a.Inst(rom.EncodeMOV(3, addrReg, src)) }
func (a *Asm) SetDBR(src uint8)            { a.Inst(rom.EncodeMOV(8, src, 0)) }
func (a *Asm) AddImm(reg uint8, v uint16)  { a.Inst(rom.EncodeADD(1, reg, 0)); a.Imm(v) }
func (a *Asm) AddReg(dst, src uint8)       { a.Inst(rom.EncodeADD(0, dst, src)) }
func (a *Asm) SubImm(reg uint8, v uint16)  { a.Inst(rom.EncodeSUB(1, reg, 0)); a.Imm(v) }
func (a *Asm) SubReg(dst, src uint8)       { a.Inst(rom.EncodeSUB(0, dst, src)) }
func (a *Asm) AndImm(reg uint8, v uint16)  { a.Inst(rom.EncodeAND(1, reg, 0)); a.Imm(v) }
func (a *Asm) CmpImm(reg uint8, v uint16)  { a.Inst(rom.EncodeCMP(7, reg, 0)); a.Imm(v) }
func (a *Asm) CmpReg(r1, r2 uint8)         { a.Inst(rom.EncodeCMP(0, r1, r2)) }
func (a *Asm) ShrImm(reg uint8, v uint16)  { a.Inst(rom.EncodeSHR(1, reg, 0)); a.Imm(v) }

func (a *Asm) branch(op uint16, label string) {
	a.Inst(op)
	pc := a.PC()
	a.Imm(0)
	a.patches = append(a.patches, patchRef{
		wordIndex: a.B.GetCodeLength(a.bank) - 1,
		currentPC: pc,
		target:    label,
	})
}

func (a *Asm) Beq(label string)  { a.branch(rom.EncodeBEQ(), label) }
func (a *Asm) Bne(label string)  { a.branch(rom.EncodeBNE(), label) }
func (a *Asm) Jmp(label string)  { a.branch(rom.EncodeJMP(), label) }
func (a *Asm) Call(label string) { a.branch(rom.EncodeCALL(), label) }
func (a *Asm) Ret()              { a.Inst(rom.EncodeRET()) }

func (a *Asm) Resolve() error {
	for _, p := range a.patches {
		targetPC, ok := a.labels[p.target]
		if !ok {
			return fmt.Errorf("unknown label %q", p.target)
		}
		a.B.SetImmediateAt(a.bank, p.wordIndex, uint16(rom.CalculateBranchOffset(p.currentPC, targetPC)))
	}
	return nil
}

func Write8(a *Asm, addr uint16, value uint8) {
	a.MovImm(4, addr)
	a.MovImm(5, uint16(value))
	a.MovStore(4, 5)
}

func Write8Scratch(a *Asm, addr uint16, value uint8, addrReg, valueReg uint8) {
	a.MovImm(addrReg, addr)
	a.MovImm(valueReg, uint16(value))
	a.MovStore(addrReg, valueReg)
}

func Write8Reg(a *Asm, addr uint16, reg uint8) {
	addrReg := uint8(4)
	if reg == addrReg {
		addrReg = 7
	}
	a.MovImm(addrReg, addr)
	a.MovStore(addrReg, reg)
}

func Write16(a *Asm, addr uint16, value uint16) {
	Write8(a, addr, uint8(value&0xFF))
	Write8(a, addr+1, uint8(value>>8))
}

func Write16S(a *Asm, addr uint16, value int16) {
	Write16(a, addr, uint16(value))
}

func Write16Reg(a *Asm, addr uint16, reg uint8) {
	Write8Reg(a, addr, reg)
	a.MovReg(7, reg)
	a.ShrImm(7, 8)
	Write8Reg(a, addr+1, 7)
}

func Write16RegBytes(a *Asm, addr uint16, reg uint8) {
	a.MovReg(7, reg)
	a.AndImm(7, 0x00FF)
	Write8Reg(a, addr, 7)
	a.MovReg(7, reg)
	a.ShrImm(7, 8)
	a.AndImm(7, 0x00FF)
	Write8Reg(a, addr+1, 7)
}

func EmitText(a *Asm, x uint16, y uint8, r, g, b uint8, text string) {
	Write16(a, 0x8070, x)
	Write8(a, 0x8072, y)
	Write8(a, 0x8073, r)
	Write8(a, 0x8074, g)
	Write8(a, 0x8075, b)
	for i := 0; i < len(text); i++ {
		Write8(a, 0x8076, text[i])
	}
}

func SetCGRAMColor(a *Asm, colorIndex uint8, rgb555 uint16) {
	Write8(a, 0x8012, colorIndex)
	Write8(a, 0x8013, uint8(rgb555&0xFF))
	Write8(a, 0x8013, uint8(rgb555>>8))
}

func EmitWaitOneFrame(a *Asm, wramLastFrame uint16) {
	waitNotVBlank := a.Uniq("wait_not_vblank")
	waitFrameEdge := a.Uniq("wait_frame_edge")
	waitVBlank := a.Uniq("wait_vblank")

	a.Mark(waitNotVBlank)
	a.MovImm(4, 0x803E)
	a.MovLoad(2, 4)
	a.CmpImm(2, 0)
	a.Bne(waitNotVBlank)

	a.Mark(waitFrameEdge)
	a.MovImm(4, 0x803F)
	a.MovLoad(2, 4)
	a.MovImm(4, wramLastFrame)
	a.MovLoad(3, 4)
	a.CmpReg(2, 3)
	a.Beq(waitFrameEdge)
	a.MovImm(4, wramLastFrame)
	a.MovStore(4, 2)

	a.Mark(waitVBlank)
	a.MovImm(4, 0x803E)
	a.MovLoad(2, 4)
	a.CmpImm(2, 0)
	a.Beq(waitVBlank)
}

type DataRef struct {
	Bank   uint8
	Offset uint16
	Length int
}

type uploadChunk struct {
	bank   uint8
	offset uint16
	count  uint16
}

func AllocateROMData(cursor int, payload []byte) (DataRef, int) {
	return DataRef{
		Bank:   uint8(rom.ROMMinProgramBank + 1 + cursor/rom.ROMBankSizeBytes),
		Offset: uint16(rom.ROMBankOffsetBase + (cursor % rom.ROMBankSizeBytes)),
		Length: len(payload),
	}, cursor + len(payload)
}

func splitROMData(ref DataRef) []uploadChunk {
	remaining := ref.Length
	bank := ref.Bank
	offset := ref.Offset
	chunks := make([]uploadChunk, 0, (remaining/rom.ROMBankSizeBytes)+1)
	for remaining > 0 {
		avail := 0x10000 - int(offset)
		if avail > remaining {
			avail = remaining
		}
		chunks = append(chunks, uploadChunk{
			bank:   bank,
			offset: offset,
			count:  uint16(avail),
		})
		remaining -= avail
		bank++
		offset = rom.ROMBankOffsetBase
	}
	return chunks
}

func AppendDataBlob(b *rom.BankedROMBuilder, startBank uint8, payload []byte) error {
	for i := 0; i < len(payload); i += 2 {
		bank := uint8(int(startBank) + (i / rom.ROMBankSizeBytes))
		if bank > rom.ROMMaxProgramBank {
			return fmt.Errorf("data exceeds ROM bank budget at bank %d", bank)
		}
		word := uint16(payload[i])
		if i+1 < len(payload) {
			word |= uint16(payload[i+1]) << 8
		}
		b.AddInstruction(bank, word)
	}
	return nil
}

func PadPayloadToRef(payload []byte, startBank uint8, ref DataRef) []byte {
	target := (int(ref.Bank)-int(startBank))*rom.ROMBankSizeBytes + int(ref.Offset-rom.ROMBankOffsetBase)
	if target <= len(payload) {
		return payload
	}
	return append(payload, make([]byte, target-len(payload))...)
}

func EmitUploadRoutine(a *Asm, label string) {
	a.Mark(label)
	loop := a.Uniq("upload_loop")
	done := a.Uniq("upload_done")
	a.SetDBR(0)
	a.Mark(loop)
	a.CmpImm(2, 0)
	a.Beq(done)
	a.MovLoad8(3, 1)
	a.MovStore(4, 3)
	a.AddImm(1, 1)
	a.SubImm(2, 1)
	a.Jmp(loop)
	a.Mark(done)
	a.Ret()
}

func EmitUploadChunks(a *Asm, routine string, targetPort uint16, ref DataRef) {
	for _, chunk := range splitROMData(ref) {
		a.MovImm(0, uint16(chunk.bank))
		a.MovImm(1, chunk.offset)
		a.MovImm(2, chunk.count)
		a.MovImm(4, targetPort)
		a.Call(routine)
	}
}

func EmitWaitForDMAIdle(a *Asm) {
	loop := a.Uniq("wait_dma_idle")
	a.Mark(loop)
	a.MovImm(4, 0x8060)
	a.MovLoad(2, 4)
	a.CmpImm(2, 0)
	a.Bne(loop)
}

func EmitMatrixBitmapDMAChunks(a *Asm, channel uint8, ref DataRef) {
	var destOffset uint32
	for _, chunk := range splitROMData(ref) {
		Write8(a, 0x8080, channel)
		Write8(a, 0x8088, uint8(destOffset&0xFF))
		Write8(a, 0x8089, uint8((destOffset>>8)&0xFF))
		Write8(a, 0x808A, uint8((destOffset>>16)&0x07))
		Write8(a, 0x8061, chunk.bank)
		Write16(a, 0x8062, chunk.offset)
		Write16(a, 0x8064, 0x0000)
		Write16(a, 0x8066, chunk.count)
		Write8(a, 0x8060, 0x15)
		EmitWaitForDMAIdle(a)
		destOffset += uint32(chunk.count)
	}
}

func EmitMatrixRowDMAChunks(a *Asm, channel uint8, ref DataRef) {
	var destOffset uint16
	for _, chunk := range splitROMData(ref) {
		Write8(a, 0x8080, channel)
		Write8(a, 0x808E, uint8(destOffset&0xFF))
		Write8(a, 0x808F, uint8(destOffset>>8))
		Write8(a, 0x8061, chunk.bank)
		Write16(a, 0x8062, chunk.offset)
		Write16(a, 0x8064, 0x0000)
		Write16(a, 0x8066, chunk.count)
		Write8(a, 0x8060, 0x19)
		EmitWaitForDMAIdle(a)
		destOffset += chunk.count
	}
}

func EmitVRAMDMAChunks(a *Asm, destAddr uint16, ref DataRef) {
	var offset uint16
	for _, chunk := range splitROMData(ref) {
		Write8(a, 0x8061, chunk.bank)
		Write16(a, 0x8062, chunk.offset)
		Write16(a, 0x8064, destAddr+offset)
		Write16(a, 0x8066, chunk.count)
		Write8(a, 0x8060, 0x01)
		EmitWaitForDMAIdle(a)
		offset += chunk.count
	}
}

func EmitInitTrigTable(a *Asm, tableBase uint16, steps int) {
	for i := 0; i < steps; i++ {
		angle := (2.0 * math.Pi * float64(i)) / float64(steps)
		cosv := int16(math.Round(math.Cos(angle) * 256.0))
		sinv := int16(math.Round(math.Sin(angle) * 256.0))
		Write16S(a, tableBase+uint16(i*4), cosv)
		Write16S(a, tableBase+uint16(i*4)+2, sinv)
	}
}

func EmitLoadTrigPair(a *Asm, tableBase uint16, indexReg, cosReg, sinReg uint8) {
	a.AddReg(indexReg, indexReg)
	a.AddReg(indexReg, indexReg)
	a.AddImm(indexReg, tableBase)
	a.MovLoad(cosReg, indexReg)
	a.AddImm(indexReg, 2)
	a.MovLoad(sinReg, indexReg)
}

func EmitWriteMatrixRegs(a *Asm, controlAddr, aAddr, bAddr, cAddr, dAddr, cxAddr, cyAddr uint16, controlValue uint8, aReg, bReg, cReg, dReg uint8, centerX, centerY int16) {
	Write8(a, controlAddr, controlValue)
	Write16Reg(a, aAddr, aReg)
	Write16Reg(a, bAddr, bReg)
	Write16Reg(a, cAddr, cReg)
	Write16Reg(a, dAddr, dReg)
	Write16S(a, cxAddr, centerX)
	Write16S(a, cyAddr, centerY)
}
