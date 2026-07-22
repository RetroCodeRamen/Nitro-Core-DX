package ym2608

const (
	fmEGBits    = 16
	fmEGStepAdd = 3 << (11 + fmEGBits)
	fmPGBits    = 9
	fmRatioBits = 12
	fmLFOCBits  = 14
)

type egPhase uint8

const (
	egNext egPhase = iota
	egAttack
	egDecay
	egSustain
	egRelease
	egOff
)

var fmNoteTable = [128]uint8{
	0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 3, 3, 3, 3, 3, 3,
	4, 4, 4, 4, 4, 4, 4, 5, 6, 7, 7, 7, 7, 7, 7, 7,
	8, 8, 8, 8, 8, 8, 8, 9, 10, 11, 11, 11, 11, 11, 11, 11,
	12, 12, 12, 12, 12, 12, 12, 13, 14, 15, 15, 15, 15, 15, 15, 15,
	16, 16, 16, 16, 16, 16, 16, 17, 18, 19, 19, 19, 19, 19, 19, 19,
	20, 20, 20, 20, 20, 20, 20, 21, 22, 23, 23, 23, 23, 23, 23, 23,
	24, 24, 24, 24, 24, 24, 24, 25, 26, 27, 27, 27, 27, 27, 27, 27,
	28, 28, 28, 28, 28, 28, 28, 29, 30, 31, 31, 31, 31, 31, 31, 31,
}

var fmFeedbackTable = [8]uint32{31, 7, 6, 5, 4, 3, 2, 1}

var fmAMShiftTable = [4]uint8{29, 4, 2, 1}

var fmGainTable = [64]uint8{
	0xff, 0xea, 0xd7, 0xc5, 0xb5, 0xa6, 0x98, 0x8b, 0x80, 0x75, 0x6c, 0x63, 0x5a, 0x53, 0x4c, 0x46,
	0x40, 0x3b, 0x36, 0x31, 0x2d, 0x2a, 0x26, 0x23, 0x20, 0x1d, 0x1b, 0x19, 0x17, 0x15, 0x13, 0x12,
	0x10, 0x0f, 0x0e, 0x0c, 0x0b, 0x0a, 0x0a, 0x09, 0x08, 0x07, 0x07, 0x06, 0x06, 0x05, 0x05, 0x04,
	0x04, 0x04, 0x03, 0x03, 0x03, 0x03, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x01, 0x01, 0x01, 0x01,
}

var fmAlgorithmTable = [8][6]uint8{
	{0, 1, 1, 2, 2, 3}, {1, 0, 0, 1, 1, 2},
	{1, 1, 1, 0, 0, 2}, {0, 1, 2, 1, 1, 2},
	{0, 1, 2, 2, 2, 1}, {0, 1, 0, 1, 0, 1},
	{0, 1, 2, 1, 2, 1}, {1, 0, 1, 0, 1, 0},
}

var fmDetuneTable = [256]int32{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 2, 2, 2, 2, 2, 2, 2, 2, 4, 4, 4, 4,
	4, 6, 6, 6, 8, 8, 8, 10, 10, 12, 12, 14, 16, 16, 16, 16,
	2, 2, 2, 2, 4, 4, 4, 4, 4, 6, 6, 6, 8, 8, 8, 10,
	10, 12, 12, 14, 16, 16, 18, 20, 22, 24, 26, 28, 32, 32, 32, 32,
	4, 4, 4, 4, 4, 6, 6, 6, 8, 8, 8, 10, 10, 12, 12, 14,
	16, 16, 18, 20, 22, 24, 26, 28, 32, 34, 38, 40, 44, 44, 44, 44,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, -2, -2, -2, -2, -2, -2, -2, -2, -4, -4, -4, -4,
	-4, -6, -6, -6, -8, -8, -8, -10, -10, -12, -12, -14, -16, -16, -16, -16,
	-2, -2, -2, -2, -4, -4, -4, -4, -4, -6, -6, -6, -8, -8, -8, -10,
	-10, -12, -12, -14, -16, -16, -18, -20, -22, -24, -26, -28, -32, -32, -32, -32,
	-4, -4, -4, -4, -4, -6, -6, -6, -8, -8, -8, -10, -10, -12, -12, -14,
	-16, -16, -18, -20, -22, -24, -26, -28, -32, -34, -38, -40, -44, -44, -44, -44,
}

type FMCore struct {
	Channels [6]FMChannel

	ratio     uint32
	rr        float32
	rateTable [64]int32
	lfoTable  [8]uint32

	fnum  [6]uint32
	fnum3 [3]uint32
	fnum2 [9]uint8
	regTC uint8
	reg29 uint8
	reg22 uint8

	lfoCount  uint32
	lfoDCount uint32
	aml       uint8
}

type FMChannel struct {
	Feedback  uint32
	Algorithm uint8
	Index     [6]uint8
	Operators [4]FMOperator
}

type FMOperator struct {
	KeyOn bool

	DP uint32
	BN uint32

	Detune   uint8
	Multiple uint8
	TL       uint8
	TLL      uint8
	KS       uint8
	AR       uint8
	DR       uint8
	AMOn     bool
	SR       uint8
	SL       uint8
	RR       uint8
	SSGType  uint8
	MS       uint8

	ParamChanged bool
	Mute         bool

	Phase       egPhase
	EGLevel     int32
	EGLevelNext int32
	EGStep      int32
	EGStepD     int32
	EGTransA    uint8
	EGTransD    int32
	KSR         uint32
	AMS         uint8
	EGOut       int32

	PGCount   uint32
	PGDCount  uint32
	PGDCountL uint32
	Out       int32
	Out2      int32
}

func NewFMCore() *FMCore {
	f := &FMCore{}
	f.SetClock(DefaultMasterClockHz, DefaultSampleRateHz)
	f.Reset()
	return f
}

func NewFMCoreWithClock(masterClockHz, sampleRateHz uint32) *FMCore {
	f := &FMCore{}
	f.SetClock(masterClockHz, sampleRateHz)
	f.Reset()
	return f
}

func (f *FMCore) SetClock(masterClockHz, sampleRateHz uint32) {
	f.SetPrescaler(masterClockHz, sampleRateHz, 0)
}

func (f *FMCore) SetPrescaler(masterClockHz, sampleRateHz uint32, prescaler uint8) {
	if masterClockHz == 0 {
		masterClockHz = DefaultMasterClockHz
	}
	if sampleRateHz == 0 {
		sampleRateHz = DefaultSampleRateHz
	}
	table := [3]uint32{6, 3, 2}
	if prescaler >= uint8(len(table)) {
		prescaler = 0
	}
	fmClock := (masterClockHz / 2) / table[prescaler] / 12
	if fmClock == 0 {
		fmClock = sampleRateHz
	}
	f.ratio = ((fmClock << fmRatioBits) + sampleRateHz/2) / sampleRateHz
	if f.ratio == 0 {
		f.ratio = 1
	}
	f.rr = float32(f.ratio) / float32(uint32(1)<<(2+fmRatioBits-fmPGBits))
	f.makeTimeTable()
	for i := range f.Channels {
		for j := range f.Channels[i].Operators {
			f.Channels[i].Operators[j].ParamChanged = true
		}
	}
}

func (f *FMCore) Reset() {
	f.fnum = [6]uint32{}
	f.fnum2 = [9]uint8{}
	f.fnum3 = [3]uint32{}
	f.regTC = 0
	f.reg29 = 0x1f
	f.reg22 = 0
	f.lfoCount = 0
	f.lfoDCount = 0
	f.aml = 0
	for i := range f.Channels {
		f.Channels[i].init()
	}
}

func (f *FMCore) Mix(dest []int32) {
	for i := range dest {
		dest[i] += f.MixSample()
	}
}

func (f *FMCore) MixSample() int32 {
	active := f.ActiveMask()
	if active&0x555 == 0 {
		return 0
	}
	return f.MixActiveSample(active)
}

func (f *FMCore) MixActiveSample(active int) int32 {
	var sample int32
	if active&0xaaa != 0 {
		f.stepLFO()
	}
	for ch := 0; ch < len(f.Channels); ch++ {
		if active&(1<<(ch*2)) != 0 {
			sample += f.Channels[ch].calc(f)
		}
	}
	return limit16ToInt32(sample >> 2)
}

func (f *FMCore) ActiveMask() int {
	f.applyChannel3Mode()
	active := 0
	limit := 3
	if f.reg29&0x80 != 0 {
		limit = len(f.Channels)
	}
	for ch := 0; ch < limit; ch++ {
		active |= f.Channels[ch].prepareState(f) << (ch * 2)
	}
	if f.reg22&0x08 == 0 {
		active &= 0x555
	}
	return active
}

func (f *FMCore) WriteReg(addr uint16, data uint8) {
	c := int(addr & 0x03)
	switch addr {
	case 0x27:
		f.regTC = data
	case 0x22:
		modified := f.reg22 ^ data
		f.reg22 = data
		if modified&0x08 != 0 {
			f.lfoCount = 0
		}
		if f.reg22&0x08 != 0 {
			f.lfoDCount = f.lfoTable[f.reg22&0x07]
		} else {
			f.lfoDCount = 0
		}
	case 0x28:
		f.writeKey(data)
	case 0x29:
		f.reg29 = data
	case 0xa0, 0xa1, 0xa2, 0x1a0, 0x1a1, 0x1a2:
		if addr&0x100 != 0 {
			c += 3
		}
		f.fnum[c] = uint32(data) + uint32(f.fnum2[c])*0x100
		f.applyFNum(c, f.fnum[c])
	case 0xa4, 0xa5, 0xa6, 0x1a4, 0x1a5, 0x1a6:
		if addr&0x100 != 0 {
			c += 3
		}
		f.fnum2[c] = data
	case 0xa8, 0xa9, 0xaa:
		f.fnum3[c] = uint32(data) + uint32(f.fnum2[c+6])*0x100
	case 0xac, 0xad, 0xae:
		f.fnum2[c+6] = data
	case 0xb0, 0xb1, 0xb2, 0x1b0, 0x1b1, 0x1b2:
		if addr&0x100 != 0 {
			c += 3
		}
		ch := &f.Channels[c]
		ch.Feedback = fmFeedbackTable[(data>>3)&0x07]
		ch.setAlgorithm(data & 0x07)
	case 0xb4, 0xb5, 0xb6, 0x1b4, 0x1b5, 0x1b6:
		if addr&0x100 != 0 {
			c += 3
		}
		for i := range f.Channels[c].Operators {
			op := &f.Channels[c].Operators[i]
			op.MS = data
			op.ParamChanged = true
		}
	default:
		f.writeOperatorReg(addr, data)
	}
}

func (f *FMCore) SetChannelMask(mask uint32) {
	for i := range f.Channels {
		muted := mask&(1<<i) == 0
		for j := range f.Channels[i].Operators {
			op := &f.Channels[i].Operators[j]
			op.Mute = muted
			op.ParamChanged = true
		}
	}
}

func (f *FMCore) FNum(ch int) uint32 {
	if ch < 0 || ch >= len(f.fnum) {
		return 0
	}
	return f.fnum[ch]
}

func (f *FMCore) FNum3(op int) uint32 {
	if op < 0 || op >= len(f.fnum3) {
		return 0
	}
	return f.fnum3[op]
}

func (f *FMCore) TimerA() {
	if f.regTC&0x80 == 0 {
		return
	}
	for i := range f.Channels[2].Operators {
		f.Channels[2].Operators[i].keyOn()
	}
	for i := range f.Channels[2].Operators {
		f.Channels[2].Operators[i].keyOff()
	}
}

func (f *FMCore) writeKey(data uint8) {
	if data&0x03 >= 3 {
		return
	}
	c := int(data & 0x03)
	if data&0x04 != 0 {
		c += 3
	}
	key := data >> 4
	for i := 0; i < 4; i++ {
		op := &f.Channels[c].Operators[i]
		if key&(1<<i) != 0 {
			op.keyOn()
		} else {
			op.keyOff()
		}
	}
}

func (f *FMCore) applyFNum(ch int, value uint32) {
	dp := (value & 2047) << ((value >> 11) & 7)
	bn := uint32(fmNoteTable[(value>>7)&127])
	for i := range f.Channels[ch].Operators {
		op := &f.Channels[ch].Operators[i]
		op.DP = dp
		op.BN = bn
		op.ParamChanged = true
	}
}

func (f *FMCore) makeTimeTable() {
	for h := 1; h < 16; h++ {
		for l := 0; l < 4; l++ {
			m := l + 4
			if h == 15 {
				m = 8
			}
			v := ((f.ratio << (fmEGBits - 3 - fmRatioBits)) << minInt(h, 11)) * uint32(m)
			f.rateTable[h*4+l] = int32(v)
		}
	}
	f.rateTable[0], f.rateTable[1], f.rateTable[2], f.rateTable[3] = 0, 0, 0, 0
	f.rateTable[5] = f.rateTable[4]
	f.rateTable[7] = f.rateTable[6]

	table2 := [8]uint32{109, 78, 72, 68, 63, 45, 9, 6}
	for i := range f.lfoTable {
		f.lfoTable[i] = (f.ratio << (1 + fmLFOCBits - fmRatioBits)) / table2[i]
	}
}

func (f *FMCore) setEGRate(op *FMOperator, r uint32) {
	if r > 63 {
		r = 63
	}
	op.EGStepD = f.rateTable[r]
	op.EGTransA = uint8(clampInt(15-int(r>>2), 1, 4))
	op.EGTransD = int32(16 >> op.EGTransA)
}

func (f *FMCore) applyChannel3Mode() {
	if f.regTC&0xc0 == 0 {
		f.applyFNum(2, f.fnum[2])
		return
	}
	f.applyFNumToOperator(2, 0, f.fnum3[1])
	f.applyFNumToOperator(2, 1, f.fnum3[2])
	f.applyFNumToOperator(2, 2, f.fnum3[0])
	f.applyFNumToOperator(2, 3, f.fnum[2])
}

func (f *FMCore) applyFNumToOperator(ch, opIndex int, value uint32) {
	if ch < 0 || ch >= len(f.Channels) || opIndex < 0 || opIndex >= len(f.Channels[ch].Operators) {
		return
	}
	op := &f.Channels[ch].Operators[opIndex]
	op.DP = (value & 2047) << ((value >> 11) & 7)
	op.BN = uint32(fmNoteTable[(value>>7)&127])
	op.ParamChanged = true
}

func (f *FMCore) anyLFOActive(limit int) bool {
	for ch := 0; ch < limit; ch++ {
		if f.Channels[ch].lfoActive() {
			return true
		}
	}
	return false
}

func (f *FMCore) stepLFO() {
	c := uint8((f.lfoCount >> fmLFOCBits) & 0xff)
	f.lfoCount += f.lfoDCount
	if c < 0x80 {
		f.aml = c << 1
	} else {
		f.aml = ^(c << 1)
	}
}

func (f *FMCore) prepareOperator(op *FMOperator) {
	if !op.ParamChanged {
		return
	}
	mul := uint8(1)
	if op.Multiple != 0 {
		mul = 2 * op.Multiple
	}
	op.ParamChanged = false

	detuneIndex := int(op.Detune) + int(op.BN)
	detune := int32(0)
	if detuneIndex >= 0 && detuneIndex < len(fmDetuneTable) {
		detune = fmDetuneTable[detuneIndex]
	}
	base := int32(op.DP) + detune
	if base < 0 {
		base = 0
	}
	op.PGDCount = uint32(base) * uint32(float32(mul)*f.rr)
	op.PGDCountL = op.PGDCount >> 11
	op.KSR = op.BN >> (3 - op.KS)

	switch op.Phase {
	case egAttack:
		if op.AR != 0 {
			f.setEGRate(op, minUint32(63, uint32(op.AR)+op.KSR))
		} else {
			f.setEGRate(op, 0)
		}
	case egDecay:
		if op.DR != 0 {
			f.setEGRate(op, minUint32(63, uint32(op.DR)+op.KSR))
		} else {
			f.setEGRate(op, 0)
		}
		op.EGLevelNext = int32(op.SL) * 8
	case egSustain:
		if op.SR != 0 {
			f.setEGRate(op, minUint32(63, uint32(op.SR)+op.KSR))
		} else {
			f.setEGRate(op, 0)
		}
	case egRelease:
		f.setEGRate(op, minUint32(63, uint32(op.RR)+op.KSR))
	}

	if op.AMOn {
		op.AMS = (op.MS >> 4) & 0x03
	} else {
		op.AMS = 0
	}
}

func (f *FMCore) writeOperatorReg(addr uint16, data uint8) {
	c := int(addr & 0x03)
	if c >= 3 {
		return
	}
	if addr&0x100 != 0 {
		c += 3
	}

	group := (addr >> 4) & 0x0f
	if group < 3 || group > 9 {
		return
	}
	slot := fmSlotForAddr(addr)
	op := &f.Channels[c].Operators[slot]

	switch group {
	case 3:
		op.Detune = ((data >> 4) & 0x07) * 0x20
		op.Multiple = data & 0x0f
	case 4:
		if !((f.regTC&0x80 != 0) && c == 2) {
			op.TL = data & 0x7f
			op.ParamChanged = true
		}
		op.TLL = data & 0x7f
	case 5:
		op.KS = (data >> 6) & 0x03
		op.AR = (data & 0x1f) * 2
		op.ParamChanged = true
	case 6:
		op.DR = (data & 0x1f) * 2
		op.AMOn = data&0x80 != 0
		op.ParamChanged = true
	case 7:
		op.SR = (data & 0x1f) * 2
		op.ParamChanged = true
	case 8:
		op.SL = ((data >> 4) & 0x0f) * 4
		op.RR = (data&0x0f)*4 + 2
		op.ParamChanged = true
	case 9:
		op.SSGType = data & 0x0f
	}
}

func (ch *FMChannel) init() {
	for i := range ch.Operators {
		ch.Operators[i] = newFMOperator()
	}
	ch.setAlgorithm(0)
}

func (ch *FMChannel) setAlgorithm(algo uint8) {
	ch.Algorithm = algo & 0x07
	table := fmAlgorithmTable[ch.Algorithm]
	ch.Index[0] = table[0]
	ch.Index[1] = table[2]
	ch.Index[2] = table[4]
	ch.Index[3] = table[1]
	ch.Index[4] = table[3]
	ch.Index[5] = table[5]
}

func (ch *FMChannel) calc(core *FMCore) int32 {
	for i := range ch.Operators {
		core.prepareOperator(&ch.Operators[i])
	}
	if !ch.active() {
		return 0
	}

	ch.Operators[0].calcEnvelope(core)
	ch.Operators[1].calcEnvelope(core)
	ch.Operators[2].calcEnvelope(core)
	ch.Operators[3].calcEnvelope(core)

	var buf [4]int32
	buf[0] = ch.Operators[0].Out
	ch.Operators[0].calcFeedback(ch.Feedback)
	if ch.Index[0]|ch.Index[2]|ch.Index[4] == 0 {
		out := ch.Operators[1].calc(buf[0])
		out += ch.Operators[2].calc(buf[0])
		out += ch.Operators[3].calc(buf[0])
		return out >> 8
	}

	buf[ch.Index[3]] += ch.Operators[1].calc(buf[ch.Index[0]])
	buf[ch.Index[4]] += ch.Operators[2].calc(buf[ch.Index[1]])
	prev := ch.Operators[3].Out
	ch.Operators[3].calc(buf[ch.Index[2]])
	return (buf[ch.Index[5]] + prev) >> 8
}

func (ch *FMChannel) active() bool {
	muted := true
	key := false
	for i := range ch.Operators {
		if !ch.Operators[i].Mute {
			muted = false
		}
		if ch.Operators[i].Phase != egOff {
			key = true
		}
	}
	return !muted && key
}

func (ch *FMChannel) prepareState(core *FMCore) int {
	for i := range ch.Operators {
		core.prepareOperator(&ch.Operators[i])
	}
	if ch.Operators[0].Mute && ch.Operators[1].Mute && ch.Operators[2].Mute && ch.Operators[3].Mute {
		return 0
	}
	key := 0
	for i := range ch.Operators {
		if ch.Operators[i].Phase != egOff {
			key = 1
			break
		}
	}
	amMask := uint8(0x07)
	if ch.Operators[0].AMOn || ch.Operators[1].AMOn || ch.Operators[2].AMOn || ch.Operators[3].AMOn {
		amMask = 0x37
	}
	lfo := 0
	if ch.Operators[0].MS&amMask != 0 {
		lfo = 2
	}
	return key | lfo
}

func (ch *FMChannel) lfoActive() bool {
	anyAM := false
	for i := range ch.Operators {
		anyAM = anyAM || ch.Operators[i].AMOn
	}
	mask := uint8(0x07)
	if anyAM {
		mask = 0x37
	}
	return ch.Operators[0].MS&mask != 0 && ch.active()
}

func newFMOperator() FMOperator {
	op := FMOperator{
		TL:           127,
		TLL:          127,
		EGLevel:      0xff,
		EGLevelNext:  0x100,
		Phase:        egOff,
		ParamChanged: true,
	}
	op.EGTransA = 4
	op.EGTransD = 1
	return op
}

func (op *FMOperator) shiftPhase(core *FMCore, next egPhase) {
	switch next {
	case egAttack:
		op.TL = op.TLL
		if uint32(op.AR)+op.KSR < 62 {
			if op.AR != 0 {
				core.setEGRate(op, minUint32(63, uint32(op.AR)+op.KSR))
			} else {
				core.setEGRate(op, 0)
			}
			op.Phase = egAttack
			return
		}
		fallthrough
	case egDecay:
		if op.SL != 0 {
			op.EGLevel = 0
			op.EGLevelNext = int32(op.SL) * 8
			if op.DR != 0 {
				core.setEGRate(op, minUint32(63, uint32(op.DR)+op.KSR))
			} else {
				core.setEGRate(op, 0)
			}
			op.Phase = egDecay
			return
		}
		fallthrough
	case egSustain:
		op.EGLevel = int32(op.SL) * 8
		op.EGLevelNext = 0x100
		if op.SR != 0 {
			core.setEGRate(op, minUint32(63, uint32(op.SR)+op.KSR))
		} else {
			core.setEGRate(op, 0)
		}
		op.Phase = egSustain
		return
	case egRelease:
		if op.Phase == egAttack || op.EGLevel < 0x100 {
			op.EGLevelNext = 0x100
			core.setEGRate(op, minUint32(63, uint32(op.RR)+op.KSR))
			op.Phase = egRelease
			return
		}
		fallthrough
	case egOff:
		fallthrough
	default:
		op.EGLevel = 0xff
		op.EGLevelNext = 0x100
		core.setEGRate(op, 0)
		op.Phase = egOff
	}
}

func (op *FMOperator) calcEnvelope(core *FMCore) {
	op.EGStep -= op.EGStepD
	if op.EGStep < 0 {
		op.EGStep += fmEGStepAdd
		if op.Phase == egAttack {
			op.EGLevel -= 1 + (op.EGLevel >> op.EGTransA)
			if op.EGLevel <= 0 {
				op.shiftPhase(core, egDecay)
			}
		} else {
			op.EGLevel += op.EGTransD
			if op.EGLevel >= op.EGLevelNext {
				op.shiftPhase(core, op.Phase+1)
			}
		}
	}
	level := uint32(op.EGLevel)
	if level >= 0xff {
		op.EGOut = 0
		return
	}
	level += uint32(core.amlShift(op))
	if level >= 0xff {
		op.EGOut = 0
		return
	}
	op.EGOut = int32(clipTable[level]) * int32(fmGainCompatTable[op.TL&0x7f])
}

func (f *FMCore) amlShift(op *FMOperator) uint8 {
	return f.aml >> fmAMShiftTable[op.AMS]
}

func (op *FMOperator) keyOn() {
	if op.KeyOn {
		return
	}
	op.KeyOn = true
	if op.SL == 0 {
		op.Phase = egSustain
		op.EGLevel = 0
		op.EGLevelNext = 0x100
		op.Out, op.Out2 = 0, 0
		op.PGCount = 0
		op.ParamChanged = true
		return
	}
	if op.Phase == egOff || op.Phase == egRelease {
		op.Phase = egAttack
		op.Out, op.Out2 = 0, 0
		op.PGCount = 0
		op.ParamChanged = true
	}
}

func (op *FMOperator) keyOff() {
	if !op.KeyOn {
		return
	}
	op.KeyOn = false
	if op.Phase == egAttack || op.EGLevel < 0x100 {
		op.Phase = egRelease
		op.EGLevelNext = 0x100
		op.ParamChanged = true
	}
}

func (op *FMOperator) calc(in int32) int32 {
	tmp := fmSine(op.PGCount + uint32(in<<7))
	op.Out = op.EGOut * int32(tmp)
	op.PGCount += op.PGDCount
	return op.Out
}

func (op *FMOperator) calcFeedback(fb uint32) {
	in := op.Out + op.Out2
	op.Out2 = op.Out
	phase := op.PGCount
	if fb != 31 {
		phase += uint32((in << 6) >> fb)
	}
	tmp := fmSine(phase)
	op.Out = op.EGOut * int32(tmp)
	op.PGCount += op.PGDCount
}

func fmSine(phase uint32) int16 {
	return fmSineTable[(phase>>(20+fmPGBits-10))&1023]
}

func fmSlotForAddr(addr uint16) int {
	return int(([4]uint8{0, 2, 1, 3})[(addr>>2)&0x03])
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minUint32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
