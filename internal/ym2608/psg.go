package ym2608

import "math"

const (
	toneShift = 24
	envShift  = 22
)

var (
	emitTable    [0x20]int32
	envelopTable [16][64]int32
)

func init() {
	makePSGTables()
}

type PSG struct {
	reg [16]uint8

	envelop []int32
	rng     uint32

	olevel  [3]int32
	scount  [3]uint32
	speriod [3]uint32

	ecount  uint32
	eperiod uint32
	ncount  uint32
	nperiod uint32

	tperiodbase uint32
	eperiodbase uint32
	mask        int
}

func NewPSG(clock, rate uint32) *PSG {
	p := &PSG{mask: 0x3f}
	p.SetClock(clock, rate)
	p.Reset()
	p.SetChannelMask(0x3f)
	return p
}

func (p *PSG) Reset() {
	for i := uint8(0); i < 14; i++ {
		p.SetReg(i, 0)
	}
	p.SetReg(7, 0xff)
	p.SetReg(14, 0xff)
	p.SetReg(15, 0xff)
	p.envelop = envelopTable[0][:]
	p.rng = 14231
	p.ncount = 0
	for i := range p.scount {
		p.scount[i] = 0
	}
}

func (p *PSG) Reg(regnum uint8) uint8 {
	return p.reg[regnum&0x0f]
}

func (p *PSG) SetClock(clock, rate uint32) {
	if rate == 0 {
		rate = DefaultSampleRateHz
	}
	p.tperiodbase = uint32((float32(uint32(1)<<toneShift) / 4.0) * float32(clock) / float32(rate))
	p.eperiodbase = uint32((float32(uint32(1)<<envShift) / 4.0) * float32(clock) / float32(rate))
	p.recomputeTonePeriod(0)
	p.recomputeTonePeriod(1)
	p.recomputeTonePeriod(2)
	p.nperiod = uint32(p.reg[6] & 0x1f)
	p.recomputeEnvelopePeriod()
}

func (p *PSG) SetChannelMask(mask int) {
	p.mask = mask
	for i := 0; i < 3; i++ {
		if p.mask&(1<<i) != 0 {
			p.olevel[i] = emitTable[(p.reg[8+i]&0x0f)*2+1]
		} else {
			p.olevel[i] = 0
		}
	}
}

func (p *PSG) SetReg(regnum, data uint8) {
	if regnum >= 0x10 {
		return
	}
	p.reg[regnum] = data
	switch regnum {
	case 0, 1:
		p.recomputeTonePeriod(0)
	case 2, 3:
		p.recomputeTonePeriod(1)
	case 4, 5:
		p.recomputeTonePeriod(2)
	case 6:
		p.nperiod = uint32(data & 0x1f)
	case 8:
		p.setOutputLevel(0, data)
	case 9:
		p.setOutputLevel(1, data)
	case 10:
		p.setOutputLevel(2, data)
	case 11, 12:
		p.recomputeEnvelopePeriod()
	case 13:
		p.ecount = 0
		p.envelop = envelopTable[data&0x0f][:]
	}
}

func (p *PSG) Mix(dest []int32) {
	r7 := ^p.reg[7]
	if (r7&0x3f)|((p.reg[8]|p.reg[9]|p.reg[10])&0x1f) == 0 {
		return
	}

	var chenable [3]uint8
	if r7&0x01 != 0 && p.speriod[0] <= 1<<toneShift {
		chenable[0] = 1
	}
	if r7&0x02 != 0 && p.speriod[1] <= 1<<toneShift {
		chenable[1] = 1
	}
	if r7&0x04 != 0 && p.speriod[2] <= 1<<toneShift {
		chenable[2] = 1
	}

	usesEnvelope := (p.mask&1 != 0 && p.reg[8]&0x10 != 0) ||
		(p.mask&2 != 0 && p.reg[9]&0x10 != 0) ||
		(p.mask&4 != 0 && p.reg[10]&0x10 != 0)
	if !usesEnvelope {
		p.mixFixed(dest, r7, chenable)
		return
	}
	p.mixEnvelope(dest, r7, chenable)
}

func (p *PSG) recomputeTonePeriod(ch int) {
	base := ch * 2
	tmp := (uint32(p.reg[base]) + uint32(p.reg[base+1])*256) & 0x0fff
	if tmp != 0 {
		p.speriod[ch] = p.tperiodbase / tmp
	} else {
		p.speriod[ch] = p.tperiodbase
	}
}

func (p *PSG) recomputeEnvelopePeriod() {
	tmp := (uint32(p.reg[11]) + uint32(p.reg[12])*256) & 0xffff
	if tmp != 0 {
		p.eperiod = p.eperiodbase / tmp
	} else {
		p.eperiod = p.eperiodbase * 2
	}
}

func (p *PSG) setOutputLevel(ch int, data uint8) {
	if p.mask&(1<<ch) != 0 {
		p.olevel[ch] = emitTable[(data&0x0f)*2+1]
	} else {
		p.olevel[ch] = 0
	}
}

func (p *PSG) mixFixed(dest []int32, r7 uint8, chenable [3]uint8) {
	for i := range dest {
		p.stepNoise()
		noise := uint8(p.rng & 1)

		sample := int32(0)
		x := int32((uint8(p.scount[0]>>toneShift) & chenable[0]) | ((r7 >> 3) & noise))
		x--
		sample += (p.olevel[0] + x) ^ x
		p.scount[0] += p.speriod[0]

		y := int32((uint8(p.scount[1]>>toneShift) & chenable[1]) | ((r7 >> 4) & noise))
		y--
		sample += (p.olevel[1] + y) ^ y
		p.scount[1] += p.speriod[1]

		z := int32((r7 >> 5) & noise)
		z--
		sample += (p.olevel[2] + z) ^ z
		p.scount[2] += p.speriod[2]

		dest[i] += limit16ToInt32(sample)
	}

	p.ecount = (p.ecount >> 8) + (p.eperiod>>8)*uint32(len(dest))
	if p.ecount >= 1<<(envShift+6-8) {
		if (p.reg[0x0d] & 0x0b) != 0x0a {
			p.ecount |= 1 << (envShift + 5 - 8)
		}
		p.ecount &= (1 << (envShift + 6 - 8)) - 1
	}
	p.ecount <<= 8
}

func (p *PSG) mixEnvelope(dest []int32, r7 uint8, chenable [3]uint8) {
	for i := range dest {
		p.stepNoise()
		noise := uint8(p.rng & 1)
		env := p.currentEnvelope()

		p1 := p.olevel[0]
		if p.mask&1 != 0 && p.reg[8]&0x10 != 0 {
			p1 = env
		}
		p2 := p.olevel[1]
		if p.mask&2 != 0 && p.reg[9]&0x10 != 0 {
			p2 = env
		}
		p3 := p.olevel[2]
		if p.mask&4 != 0 && p.reg[10]&0x10 != 0 {
			p3 = env
		}

		sample := int32(0)
		x := int32((uint8(p.scount[0]>>toneShift) & chenable[0]) | ((r7 >> 3) & noise))
		x--
		sample += (p1 + x) ^ x
		p.scount[0] += p.speriod[0]

		y := int32((uint8(p.scount[1]>>toneShift) & chenable[1]) | ((r7 >> 4) & noise))
		y--
		sample += (p2 + y) ^ y
		p.scount[1] += p.speriod[1]

		z := int32((r7 >> 5) & noise)
		z--
		sample += (p3 + z) ^ z
		p.scount[2] += p.speriod[2]

		dest[i] += limit16ToInt32(sample)
	}
}

func (p *PSG) stepNoise() {
	p.ncount++
	if p.ncount >= p.nperiod {
		if p.rng&1 != 0 {
			p.rng ^= 0x24000
		}
		p.rng >>= 1
		p.ncount = 0
	}
}

func (p *PSG) currentEnvelope() int32 {
	if p.envelop == nil {
		p.envelop = envelopTable[0][:]
	}
	env := p.envelop[p.ecount>>envShift]
	p.ecount += p.eperiod
	if p.ecount >= 1<<(envShift+6) {
		if (p.reg[0x0d] & 0x0b) != 0x0a {
			p.ecount |= 1 << (envShift + 5)
		}
		p.ecount &= (1 << (envShift + 6)) - 1
	}
	return env
}

func makePSGTables() {
	base := float64(0x4000) / 3.0
	for i := 31; i >= 2; i-- {
		emitTable[i] = int32(math.Round(base))
		base *= 0.840896415
	}
	emitTable[1] = 0
	emitTable[0] = 0

	table1 := [32]uint8{
		2, 0, 2, 0, 2, 0, 2, 0, 1, 0, 1, 0, 1, 0, 1, 0,
		2, 2, 2, 0, 2, 1, 2, 3, 1, 1, 1, 3, 1, 2, 1, 0,
	}
	table3 := [4]int8{0, 1, -1, 0}
	idx := 0
	for i := 0; i < len(table1); i++ {
		v := int16(0)
		if table1[i]&0x02 != 0 {
			v = 31
		}
		for j := 0; j < 32; j++ {
			envelopTable[idx/64][idx%64] = emitTable[uint8(v)]
			v += int16(table3[table1[i]])
			idx++
		}
	}
}

func limit16ToInt32(v int32) int32 {
	if v > 32767 {
		return 32767
	}
	if v < -32768 {
		return -32768
	}
	return v
}
