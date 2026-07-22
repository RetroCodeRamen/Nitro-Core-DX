package ym2608

const (
	DefaultMasterClockHz = 8000000
	DefaultSampleRateHz  = 44100

	Port0Addr   uint16 = 0x00
	Port0Data   uint16 = 0x01
	PortStatus0 uint16 = 0x02
	PortControl uint16 = 0x03
	Port1Addr   uint16 = 0x04
	Port1Data   uint16 = 0x05

	StatusTimerA uint8 = 1 << 0
	StatusTimerB uint8 = 1 << 1
	StatusBusy   uint8 = 1 << 6
)

const (
	regTimerAHigh = 0x24
	regTimerALow  = 0x25
	regTimerB     = 0x26
	regTimerCtrl  = 0x27
)

// Config captures clock/output settings for the isolated Go chip.
type Config struct {
	MasterClockHz uint32
	SampleRateHz  uint32
	Interpolation bool
}

// Chip is a pure-Go YM2608/OPNA instance with the same dual-port host shape as
// the current cgo bridge. It currently implements the replacement boundary,
// register shadowing, reset, busy, and timer/status behavior.
type Chip struct {
	cfg Config

	addr [2]uint8
	regs [2][256]uint8

	psg       *PSG
	rhythm    *RhythmUnit
	fm        *FMCore
	prescaler uint8

	interpolation bool
	fmRate        uint32
	mpratio       int32
	mixdelta      int32
	mixl          int32
	mixl1         int32

	status     uint8
	busyClocks uint64

	timerA timer
	timerB timer

	timerStepFixed int64
	timerAFixed    int64
	timerAFixedCnt int64
	timerBFixed    int64
	timerBFixedCnt int64
}

func New(cfg Config) *Chip {
	if cfg.MasterClockHz == 0 {
		cfg.MasterClockHz = DefaultMasterClockHz
	}
	if cfg.SampleRateHz == 0 {
		cfg.SampleRateHz = DefaultSampleRateHz
	}
	c := &Chip{cfg: cfg}
	c.Reset()
	return c
}

func (c *Chip) Config() Config {
	return c.cfg
}

func (c *Chip) Reset() {
	c.addr = [2]uint8{}
	c.regs = [2][256]uint8{}
	c.psg = NewPSG(c.cfg.MasterClockHz/8, c.cfg.SampleRateHz)
	c.rhythm = NewRhythmUnit(c.cfg.SampleRateHz)
	c.prescaler = 0
	c.interpolation = c.cfg.Interpolation
	c.fmRate, c.mpratio = c.computeFMRate(c.cfg.SampleRateHz, c.interpolation, c.prescaler)
	c.mixdelta = 16383
	c.mixl = 0
	c.mixl1 = 0
	c.fm = NewFMCoreWithClock(c.cfg.MasterClockHz, c.fmRate)
	for i := uint8(0); i < 16; i++ {
		c.regs[0][i] = c.psg.Reg(i)
	}
	c.status = 0
	c.busyClocks = 0
	c.timerA = timer{}
	c.timerB = timer{}
	c.timerStepFixed = c.timerStepFixedPeriod()
	c.timerAFixed = c.timerAPeriodFixed()
	c.timerBFixed = c.timerBPeriodFixed()
	c.timerAFixedCnt = 0
	c.timerBFixedCnt = 0
}

func (c *Chip) ReadPort(offset uint16) uint8 {
	switch offset & 0x00ff {
	case Port0Addr:
		return c.addr[0]
	case Port0Data:
		return c.regs[0][c.addr[0]]
	case PortStatus0:
		return c.statusWithBusy()
	case PortControl:
		return 0
	case Port1Addr:
		return c.addr[1]
	case Port1Data:
		return c.regs[1][c.addr[1]]
	default:
		return 0
	}
}

func (c *Chip) WritePort(offset uint16, data uint8) {
	switch offset & 0x00ff {
	case Port0Addr:
		c.addr[0] = data
		c.setBusy()
	case Port0Data:
		c.writeRegister(0, c.addr[0], data)
	case PortControl:
		if data&0x80 != 0 {
			c.Reset()
		}
	case Port1Addr:
		c.addr[1] = data
		c.setBusy()
	case Port1Data:
		c.writeRegister(1, c.addr[1], data)
	default:
	}
}

func (c *Chip) Register(port int, addr uint8) uint8 {
	if port < 0 || port >= len(c.regs) {
		return 0
	}
	return c.regs[port][addr]
}

func (c *Chip) OPNAReg(addr uint16) uint8 {
	if addr < 0x10 && c.psg != nil {
		return c.psg.Reg(uint8(addr))
	}
	if addr == 0xff {
		return 1
	}
	return 0
}

func (c *Chip) Address(port int) uint8 {
	if port < 0 || port >= len(c.addr) {
		return 0
	}
	return c.addr[port]
}

func (c *Chip) IRQPending() bool {
	return c.status&(StatusTimerA|StatusTimerB) != 0
}

func (c *Chip) Status() uint8 {
	return c.statusWithBusy()
}

// Step advances the isolated chip by YM2608 master-clock ticks.
func (c *Chip) Step(masterClocks uint64) {
	if masterClocks == 0 {
		return
	}
	if masterClocks >= c.busyClocks {
		c.busyClocks = 0
	} else {
		c.busyClocks -= masterClocks
	}
	if c.timerA.step(masterClocks) {
		if c.fm != nil {
			c.fm.TimerA()
		}
		if c.timerA.flagEnable {
			c.status |= StatusTimerA
		}
	}
	if c.timerB.step(masterClocks) {
		if c.timerB.flagEnable {
			c.status |= StatusTimerB
		}
	}
}

// TimerCount advances PMDWin-compatible timers by microseconds. The current C
// source exposes OPNATimerCount(OPNA*, us), and its timer periods are fixed-point
// microsecond counters derived from the selected prescaler.
func (c *Chip) TimerCount(us int32) bool {
	if us == 0 {
		return false
	}
	event := false
	if c.timerAFixedCnt != 0 {
		c.timerAFixedCnt -= int64(us) << 16
		if c.timerAFixedCnt <= 0 {
			event = true
			if c.fm != nil {
				c.fm.TimerA()
			}
			for c.timerAFixedCnt <= 0 {
				c.timerAFixedCnt += c.timerAFixed
			}
			if c.timerA.flagEnable {
				c.status |= StatusTimerA
			}
		}
	}
	if c.timerBFixedCnt != 0 {
		c.timerBFixedCnt -= int64(us) << 12
		if c.timerBFixedCnt <= 0 {
			event = true
			for c.timerBFixedCnt <= 0 {
				c.timerBFixedCnt += c.timerBFixed
			}
			if c.timerB.flagEnable {
				c.status |= StatusTimerB
			}
		}
	}
	return event
}

// GenerateSampleFixed mixes the PMDWin-compatible FM, PSG, and rhythm slices
// into a mono int16 sample.
func (c *Chip) GenerateSampleFixed() (left, right int16) {
	var mixed [1]int32
	if c.psg != nil {
		c.psg.Mix(mixed[:])
	}
	if c.fm != nil {
		if c.interpolation {
			mixed[0] += c.mixFMInterpolatedSample()
		} else {
			c.fm.Mix(mixed[:])
		}
	}
	if c.rhythm != nil {
		c.rhythm.Mix(mixed[:])
	}
	sample := int16(limit16ToInt32(mixed[0] >> 2))
	return sample, sample
}

func (c *Chip) SetRhythmSample(index int, sample []int8, sampleRate uint32) {
	if c.rhythm == nil {
		c.rhythm = NewRhythmUnit(c.cfg.SampleRateHz)
	}
	c.rhythm.SetSample(index, sample, sampleRate)
}

func (c *Chip) SetRhythmVolume(index int, db int) {
	if c.rhythm == nil {
		return
	}
	c.rhythm.SetVolume(index, db)
}

func (c *Chip) SetChannelMask(mask uint32) {
	if c.fm != nil {
		c.fm.SetChannelMask(mask)
	}
	if c.psg != nil {
		c.psg.SetChannelMask(int(mask >> 6))
	}
}

func (c *Chip) SetSampleRate(sampleRate uint32) {
	c.SetRate(sampleRate, c.interpolation)
}

func (c *Chip) SetRate(sampleRate uint32, interpolation bool) {
	if sampleRate == 0 {
		return
	}
	c.cfg.SampleRateHz = sampleRate
	c.cfg.Interpolation = interpolation
	c.interpolation = interpolation
	c.fmRate, c.mpratio = c.computeFMRate(sampleRate, interpolation, c.prescaler)
	psgDiv := [3]uint32{8, 4, 2}
	p := c.prescaler
	if p >= uint8(len(psgDiv)) {
		p = 0
	}
	if c.psg != nil {
		c.psg.SetClock(c.cfg.MasterClockHz/psgDiv[p], sampleRate)
	}
	if c.rhythm != nil {
		c.rhythm.SetOutputRate(sampleRate)
	}
	if c.fm != nil {
		c.fm.SetPrescaler(c.cfg.MasterClockHz, c.fmRate, c.prescaler)
	}
}

func (c *Chip) FM() *FMCore {
	return c.fm
}

func (c *Chip) writeRegister(port int, addr, data uint8) {
	prev := c.regs[port][addr]
	c.regs[port][addr] = data
	c.setBusy()
	if c.fm != nil {
		fullAddr := uint16(addr)
		if port == 1 {
			fullAddr |= 0x100
		}
		c.fm.WriteReg(fullAddr, data)
	}
	if port != 0 {
		return
	}

	if addr < 0x10 && c.psg != nil {
		c.psg.SetReg(addr, data)
	}
	if c.rhythm != nil {
		c.rhythm.WriteReg(addr, data)
	}

	switch addr {
	case 0x2d, 0x2e, 0x2f:
		c.setPrescaler(addr - 0x2d)
	case regTimerAHigh, regTimerALow:
		c.configureTimerA()
	case regTimerB:
		c.configureTimerB()
	case regTimerCtrl:
		c.writeTimerControl(prev, data)
	}
}

func (c *Chip) writeTimerControl(prev, data uint8) {
	if data&0x10 != 0 {
		c.status &^= StatusTimerA
	}
	if data&0x20 != 0 {
		c.status &^= StatusTimerB
	}

	c.timerA.flagEnable = data&0x04 != 0
	c.timerB.flagEnable = data&0x08 != 0

	changed := prev ^ data
	if changed&0x01 != 0 {
		if data&0x01 != 0 {
			c.timerA.start(c.timerAPeriodClocks())
			c.timerAFixedCnt = c.timerAFixed
		} else {
			c.timerA.stop()
			c.timerAFixedCnt = 0
		}
	}
	if changed&0x02 != 0 {
		if data&0x02 != 0 {
			c.timerB.start(c.timerBPeriodClocks())
			c.timerBFixedCnt = c.timerBFixed
		} else {
			c.timerB.stop()
			c.timerBFixedCnt = 0
		}
	}
}

func (c *Chip) configureTimerA() {
	c.timerA.period = c.timerAPeriodClocks()
	c.timerAFixed = c.timerAPeriodFixed()
}

func (c *Chip) configureTimerB() {
	c.timerB.period = c.timerBPeriodClocks()
	c.timerBFixed = c.timerBPeriodFixed()
}

func (c *Chip) timerAPeriodClocks() uint64 {
	na := (uint16(c.regs[0][regTimerAHigh]) << 2) | uint16(c.regs[0][regTimerALow]&0x03)
	return 72 * uint64(0x400-na)
}

func (c *Chip) timerBPeriodClocks() uint64 {
	nb := c.regs[0][regTimerB]
	return 1152 * uint64(0x100-uint16(nb))
}

func (c *Chip) timerAPeriodFixed() int64 {
	na := (uint16(c.regs[0][regTimerAHigh]) << 2) | uint16(c.regs[0][regTimerALow]&0x03)
	return int64(0x400-uint32(na)) * c.timerStepFixed
}

func (c *Chip) timerBPeriodFixed() int64 {
	nb := c.regs[0][regTimerB]
	return int64(0x100-uint32(nb)) * c.timerStepFixed
}

func (c *Chip) timerStepFixedPeriod() int64 {
	table := [3]uint32{6, 3, 2}
	p := c.prescaler
	if p >= uint8(len(table)) {
		p = 0
	}
	fmClock := ((c.cfg.MasterClockHz / 2) / table[p]) / 12
	if fmClock == 0 {
		fmClock = 1
	}
	return int64(int32(float32(1000000.0) * float32(65536.0) / float32(fmClock)))
}

func (c *Chip) computeFMRate(sampleRate uint32, interpolation bool, prescaler uint8) (uint32, int32) {
	if sampleRate == 0 {
		sampleRate = DefaultSampleRateHz
	}
	if !interpolation {
		return sampleRate, 0
	}
	table := [3]uint32{6, 3, 2}
	if prescaler >= uint8(len(table)) {
		prescaler = 0
	}
	fmClock := ((c.cfg.MasterClockHz / 2) / table[prescaler]) / 12
	if fmClock == 0 {
		return sampleRate, 0
	}
	rate := fmClock * 2
	var mpratio uint32
	for {
		rate >>= 1
		if rate == 0 {
			return sampleRate, 0
		}
		mpratio = sampleRate * 16384 / rate
		if mpratio > 8192 {
			break
		}
	}
	return rate, int32(mpratio)
}

func (c *Chip) mixFMInterpolatedSample() int32 {
	active := c.fm.ActiveMask()
	if active&0x555 == 0 {
		c.mixl = 0
		c.mixdelta = 16383
		return 0
	}
	if c.mpratio == 0 {
		return c.fm.MixSample()
	}
	if c.mpratio < 16384 {
		var l, d int32
		for c.mixdelta > 0 {
			l = c.fm.MixActiveSample(active)
			d = minInt32(c.mpratio, c.mixdelta)
			c.mixl += l * d
			c.mixdelta -= c.mpratio
		}
		out := c.mixl >> 14
		c.mixl = l * (16384 - d)
		c.mixdelta += 16384
		return out
	}

	impr := int32(16384 * 16384 / c.mpratio)
	if c.mixdelta < 0 {
		c.mixdelta += 16384
		c.mixl = c.mixl1
		c.mixl1 = c.fm.MixActiveSample(active)
	}
	out := (c.mixdelta*c.mixl + (16384-c.mixdelta)*c.mixl1) / 16384
	c.mixdelta -= impr
	return out
}

func (c *Chip) statusWithBusy() uint8 {
	status := c.status
	if c.busyClocks != 0 {
		status |= StatusBusy
	}
	return status
}

func (c *Chip) setBusy() {
	c.busyClocks = 32
}

func (c *Chip) setPrescaler(p uint8) {
	if p > 2 {
		p = 0
	}
	c.prescaler = p
	c.timerStepFixed = c.timerStepFixedPeriod()
	c.fmRate, c.mpratio = c.computeFMRate(c.cfg.SampleRateHz, c.interpolation, p)
	psgDiv := [3]uint32{8, 4, 2}
	if c.psg != nil {
		c.psg.SetClock(c.cfg.MasterClockHz/psgDiv[p], c.cfg.SampleRateHz)
	}
	if c.fm != nil {
		c.fm.SetPrescaler(c.cfg.MasterClockHz, c.fmRate, p)
	}
}

func minInt32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}
