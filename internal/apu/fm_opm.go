package apu

import (
	"math"

	"nitro-core-dx/internal/debug"
)

const (
	// FMExtensionOffsetBase is the APU-internal offset base for the FM extension.
	// CPU-visible address 0x9100 maps to APU offset 0x0100.
	FMExtensionOffsetBase = 0x0100

	// FM host interface registers (relative to FMExtensionOffsetBase).
	FMRegAddr    = 0x00 // OPM register address select
	FMRegData    = 0x01 // OPM register data port
	FMRegStatus  = 0x02 // Busy/timer/IRQ flags (stubbed in phase 1)
	FMRegControl = 0x03 // Enable/mute/reset
	FMRegMixL    = 0x04 // Left mix gain (future stereo path)
	FMRegMixR    = 0x05 // Right mix gain (future stereo path)
)

const (
	// Phase-1 FM_STATUS bits (host-visible).
	FMStatusTimerA = 1 << 0
	FMStatusTimerB = 1 << 1
	FMStatusBusy   = 1 << 6
	FMStatusIRQ    = 1 << 7

	// OPM/YM2151 timer-related register addresses (subset used in phase 1).
	fmOPMRegKeyOn     = 0x08
	fmOPMRegTimerAHi  = 0x10
	fmOPMRegTimerALo  = 0x11 // low 2 bits used for 10-bit timer A value
	fmOPMRegTimerB    = 0x12
	fmOPMRegTimerCtrl = 0x14
)

const (
	fmVoiceCount     = 8
	fmDefaultHz      = 44100
	fmSineTableSize  = 1024
	fmSineTableShift = 32 - 10 // use top 10 bits for lookup
)

var fmSineTable = func() [fmSineTableSize]int16 {
	var table [fmSineTableSize]int16
	for i := range table {
		phase := (2.0 * math.Pi * float64(i)) / float64(fmSineTableSize)
		table[i] = int16(math.Round(math.Sin(phase) * 32767.0))
	}
	return table
}()

type fmVoice struct {
	KeyOn bool
	PanL  bool
	PanR  bool

	Algorithm uint8
	Feedback  uint8
	PMS       uint8
	AMS       uint8

	KeyCode uint8
	KeyFrac uint8

	ModMul     uint8
	CarrierMul uint8
	ModTL      uint8
	CarrierTL  uint8

	baseInc    uint32
	modInc     uint32
	carrierInc uint32

	modPhase     uint32
	carrierPhase uint32
	lastMod      int16
	envLevel     uint16 // 0..256 simple attack envelope for pop reduction in phase-2
}

// FMOPM is a hardware-oriented YM2151/OPM-compatible extension skeleton.
// Phase 1 intentionally implements the MMIO contract first so ROMs/tools can
// target a stable interface before full FM synthesis is added.
type FMOPM struct {
	logger *debug.Logger

	// Host-visible control/status
	Addr    uint8
	Status  uint8
	Control uint8
	MixL    uint8
	MixR    uint8

	Enabled    bool
	Muted      bool
	SampleRate uint32

	// OPM register file shadow (0x00-0xFF) for software validation.
	Regs [256]uint8
	// Phase-2 audible subset (OPM-lite, hardware-oriented placeholder).
	Voices [fmVoiceCount]fmVoice

	// Phase-1 timer/status subset (hardware-friendly deterministic counters).
	timerARaw uint16 // 10-bit composite from regs 0x10/0x11
	timerBRaw uint8  // 8-bit from reg 0x12

	timerAStart     bool
	timerBStart     bool
	timerAIRQEnable bool
	timerBIRQEnable bool

	timerACounter uint64 // cycles until next expiry
	timerBCounter uint64 // cycles until next expiry
	busyCounter   uint64 // host-interface busy countdown in cycles
}

func NewFMOPM(logger *debug.Logger) *FMOPM {
	return &FMOPM{
		logger:     logger,
		MixL:       0xFF,
		MixR:       0xFF,
		SampleRate: fmDefaultHz,
	}
}

func (f *FMOPM) Read8(offset uint16) uint8 {
	switch offset {
	case FMRegAddr:
		return f.Addr
	case FMRegData:
		return f.Regs[f.Addr]
	case FMRegStatus:
		return f.Status
	case FMRegControl:
		return f.Control
	case FMRegMixL:
		return f.MixL
	case FMRegMixR:
		return f.MixR
	default:
		return 0
	}
}

func (f *FMOPM) Write8(offset uint16, value uint8) {
	switch offset {
	case FMRegAddr:
		f.Addr = value
		f.setBusy()
	case FMRegData:
		f.writeOPMData(value)
	case FMRegControl:
		f.writeControl(value)
	case FMRegMixL:
		f.MixL = value
	case FMRegMixR:
		f.MixR = value
	case FMRegStatus:
		// Status is read-only in the phase 1 interface.
	default:
		// Reserved for future FM host registers.
	}
}

func (f *FMOPM) Read16(offset uint16) uint16 {
	lo := f.Read8(offset)
	hi := f.Read8(offset + 1)
	return uint16(lo) | (uint16(hi) << 8)
}

func (f *FMOPM) Write16(offset uint16, value uint16) {
	f.Write8(offset, uint8(value&0xFF))
	f.Write8(offset+1, uint8(value>>8))
}

func (f *FMOPM) writeControl(value uint8) {
	// Bit 7 is a write-one-shot reset request; do not latch it.
	resetRequested := (value & 0x80) != 0

	f.Control = value &^ 0x80
	f.Enabled = (f.Control & 0x01) != 0
	f.Muted = (f.Control & 0x02) != 0

	if resetRequested {
		f.reset()
	}
}

func (f *FMOPM) reset() {
	f.Addr = 0
	f.Status = 0
	f.timerARaw = 0
	f.timerBRaw = 0
	f.timerAStart = false
	f.timerBStart = false
	f.timerAIRQEnable = false
	f.timerBIRQEnable = false
	f.timerACounter = 0
	f.timerBCounter = 0
	f.busyCounter = 0
	for i := range f.Regs {
		f.Regs[i] = 0
	}
	for i := range f.Voices {
		f.Voices[i] = fmVoice{}
	}
}

// GenerateSampleFloat returns the FM extension contribution for the current sample.
// Phase 2 implements an audible OPM-lite subset (8 voices, 2-op FM placeholder).
func (f *FMOPM) GenerateSampleFloat() float32 {
	return ConvertFixedToFloat(f.GenerateSampleFixed())
}

// GenerateSampleFixed returns the FM contribution as a 16-bit sample.
// Phase 2 implements an audible OPM-lite subset while keeping MMIO/timer behavior intact.
func (f *FMOPM) GenerateSampleFixed() int16 {
	if !f.Enabled || f.Muted {
		return 0
	}

	var sample int32
	activeVoices := 0
	for i := range f.Voices {
		v := &f.Voices[i]
		if !v.KeyOn {
			continue
		}
		activeVoices++

		// If neither pan bit is set, default to center for developer ergonomics in phase 2.
		panGain := int32(255)
		if v.PanL || v.PanR {
			switch {
			case v.PanL && v.PanR:
				panGain = 255
			default:
				panGain = 180 // mono fold-down attenuation for single-side pan
			}
		}

		modRaw := fmSineLookup(v.modPhase)
		modLevel := tlToLinear(v.ModTL)
		modScaled := int16((int32(modRaw) * modLevel) / 255)

		// Use PMS + feedback to control phase modulation depth. This is OPM-like but simplified.
		// Keep depth conservative in phase-2 to reduce harsh aliasy tones when layering voices.
		modDepth := int32(8 + (v.PMS * 6) + (v.Feedback * 4))
		if v.Algorithm&0x01 != 0 {
			// Alternate placeholder algorithm: more additive/less FM.
			modDepth /= 2
		}
		feedbackContribution := int32(v.lastMod) * int32(v.Feedback+1) / 24
		phaseOffset := (int32(modScaled)*modDepth + feedbackContribution) << 2

		carrierLevel := tlToLinear(v.CarrierTL)
		carrierPhase := v.carrierPhase + uint32(phaseOffset)
		carrierRaw := fmSineLookup(carrierPhase)
		voiceSample := (int32(carrierRaw) * carrierLevel) / 255

		// Add a little additive modulator content on alternate algorithms to broaden timbre.
		if v.Algorithm&0x01 != 0 {
			voiceSample += int32(modScaled) / 3
		}

		if v.envLevel < 256 {
			v.envLevel += 8
			if v.envLevel > 256 {
				v.envLevel = 256
			}
		}
		voiceSample = (voiceSample * int32(v.envLevel)) / 256

		voiceSample = (voiceSample * panGain) / 255
		sample += voiceSample

		v.lastMod = modScaled
		v.modPhase += v.modInc
		v.carrierPhase += v.carrierInc
	}

	// Add headroom when layering voices so one loud voice doesn't dominate via clipping.
	if activeVoices > 1 {
		sample = (sample * 2) / int32(activeVoices+1)
	}

	// Apply host FM mix gains (mono fold-down average).
	mix := (uint16(f.MixL) + uint16(f.MixR)) / 2
	sample = (sample * int32(mix)) / 255

	if sample > 32767 {
		sample = 32767
	} else if sample < -32768 {
		sample = -32768
	}
	return int16(sample)
}

// Step advances timers/state from the APU scheduler.
// Phase 1 is a no-op until OPM timers are implemented.
func (f *FMOPM) Step(cycles uint64) {
	if cycles == 0 || !f.Enabled {
		return
	}
	f.stepBusy(cycles)
	f.stepTimerA(cycles)
	f.stepTimerB(cycles)
	f.refreshIRQStatus()
}

func (f *FMOPM) IRQPending() bool {
	return (f.Status & FMStatusIRQ) != 0
}

func (f *FMOPM) writeOPMData(value uint8) {
	f.Regs[f.Addr] = value
	f.setBusy()

	switch f.Addr {
	case fmOPMRegKeyOn:
		f.writeOPMKeyOn(value)
	case fmOPMRegTimerAHi:
		f.timerARaw = (f.timerARaw & 0x0003) | (uint16(value) << 2)
		f.reloadTimerAIfRunning()
	case fmOPMRegTimerALo:
		f.timerARaw = (f.timerARaw &^ 0x0003) | uint16(value&0x03)
		f.reloadTimerAIfRunning()
	case fmOPMRegTimerB:
		f.timerBRaw = value
		f.reloadTimerBIfRunning()
	case fmOPMRegTimerCtrl:
		f.writeOPMTimerControl(value)
	default:
		f.writeOPMVoiceRegister(f.Addr, value)
	}
}

func (f *FMOPM) writeOPMKeyOn(value uint8) {
	ch := int(value & 0x07)
	if ch >= fmVoiceCount {
		return
	}
	opMask := (value >> 3) & 0x0F
	v := &f.Voices[ch]
	if opMask != 0 {
		if !v.KeyOn {
			v.modPhase = 0
			v.carrierPhase = 0
			v.lastMod = 0
			v.envLevel = 0
		} else {
			// Re-trigger while already active: avoid hard phase reset click, but dip envelope slightly for articulation.
			if v.envLevel > 160 {
				v.envLevel = 160
			}
		}
		v.KeyOn = true
	} else {
		v.KeyOn = false
		v.envLevel = 0
		v.lastMod = 0
	}
}

func (f *FMOPM) writeOPMVoiceRegister(addr, value uint8) {
	switch {
	case addr >= 0x20 && addr <= 0x27:
		ch := int(addr - 0x20)
		v := &f.Voices[ch]
		v.PanL = (value & 0x40) != 0
		v.PanR = (value & 0x80) != 0
		v.Feedback = (value >> 3) & 0x07
		v.Algorithm = value & 0x07
	case addr >= 0x28 && addr <= 0x2F:
		ch := int(addr - 0x28)
		f.Voices[ch].KeyCode = value
		f.recomputeVoiceIncrements(ch)
	case addr >= 0x30 && addr <= 0x37:
		ch := int(addr - 0x30)
		f.Voices[ch].KeyFrac = value
		f.recomputeVoiceIncrements(ch)
	case addr >= 0x38 && addr <= 0x3F:
		ch := int(addr - 0x38)
		v := &f.Voices[ch]
		v.PMS = (value >> 4) & 0x07
		v.AMS = value & 0x03
	case addr >= 0x40 && addr <= 0x47:
		ch := int(addr - 0x40) // Operator slot 0 MUL (used as modulator MUL in phase 2)
		f.Voices[ch].ModMul = value & 0x0F
		f.recomputeVoiceIncrements(ch)
	case addr >= 0x58 && addr <= 0x5F:
		ch := int(addr - 0x58) // Operator slot 3 MUL (used as carrier MUL in phase 2)
		f.Voices[ch].CarrierMul = value & 0x0F
		f.recomputeVoiceIncrements(ch)
	case addr >= 0x60 && addr <= 0x67:
		ch := int(addr - 0x60) // Operator slot 0 TL -> modulator level
		f.Voices[ch].ModTL = value & 0x7F
	case addr >= 0x78 && addr <= 0x7F:
		ch := int(addr - 0x78) // Operator slot 3 TL -> carrier level
		f.Voices[ch].CarrierTL = value & 0x7F
	}
}

func (f *FMOPM) recomputeVoiceIncrements(ch int) {
	if ch < 0 || ch >= len(f.Voices) {
		return
	}
	v := &f.Voices[ch]
	if f.SampleRate == 0 {
		f.SampleRate = fmDefaultHz
	}

	baseHz := fmKeyToHz(v.KeyCode, v.KeyFrac)
	v.baseInc = hzToPhaseInc(baseHz, f.SampleRate)
	v.modInc = scalePhaseInc(v.baseInc, fmMulToRatio(v.ModMul))
	v.carrierInc = scalePhaseInc(v.baseInc, fmMulToRatio(v.CarrierMul))
}

func (f *FMOPM) writeOPMTimerControl(value uint8) {
	// Preserve raw register shadow first (already written by writeOPMData).
	prevA := f.timerAStart
	prevB := f.timerBStart

	// Phase-1 OPM-like control subset:
	// bit0=start Timer A, bit1=start Timer B
	// bit2=clear Timer A flag (one-shot action)
	// bit3=clear Timer B flag (one-shot action)
	// bit4=Timer A IRQ enable, bit5=Timer B IRQ enable
	f.timerAStart = (value & 0x01) != 0
	f.timerBStart = (value & 0x02) != 0
	if (value & 0x04) != 0 {
		f.Status &^= FMStatusTimerA
	}
	if (value & 0x08) != 0 {
		f.Status &^= FMStatusTimerB
	}
	f.timerAIRQEnable = (value & 0x10) != 0
	f.timerBIRQEnable = (value & 0x20) != 0

	// (Re)load counters on rising edge start or if unset while running.
	if f.timerAStart && (!prevA || f.timerACounter == 0) {
		f.timerACounter = f.timerAPeriodCycles()
	}
	if !f.timerAStart {
		f.timerACounter = 0
	}
	if f.timerBStart && (!prevB || f.timerBCounter == 0) {
		f.timerBCounter = f.timerBPeriodCycles()
	}
	if !f.timerBStart {
		f.timerBCounter = 0
	}

	f.refreshIRQStatus()
}

func (f *FMOPM) reloadTimerAIfRunning() {
	if f.timerAStart {
		f.timerACounter = f.timerAPeriodCycles()
	}
}

func (f *FMOPM) reloadTimerBIfRunning() {
	if f.timerBStart {
		f.timerBCounter = f.timerBPeriodCycles()
	}
}

func (f *FMOPM) timerAPeriodCycles() uint64 {
	// Phase-1 deterministic placeholder timing.
	// Uses an OPM-like 10-bit timer value where larger raw values expire sooner.
	raw := uint64(f.timerARaw & 0x03FF)
	periodUnits := uint64(0x400) - raw
	if periodUnits == 0 {
		periodUnits = 1
	}
	return periodUnits * 64
}

func (f *FMOPM) timerBPeriodCycles() uint64 {
	// Phase-1 deterministic placeholder timing.
	// Uses an OPM-like 8-bit timer value where larger raw values expire sooner.
	raw := uint64(f.timerBRaw)
	periodUnits := uint64(0x100) - raw
	if periodUnits == 0 {
		periodUnits = 1
	}
	return periodUnits * 1024
}

func (f *FMOPM) stepTimerA(cycles uint64) {
	if !f.timerAStart || f.timerACounter == 0 {
		return
	}

	remaining := cycles
	for remaining > 0 {
		if remaining < f.timerACounter {
			f.timerACounter -= remaining
			return
		}
		remaining -= f.timerACounter
		f.Status |= FMStatusTimerA
		f.timerACounter = f.timerAPeriodCycles()
		if f.timerACounter == 0 {
			f.timerACounter = 1
		}
	}
}

func (f *FMOPM) stepTimerB(cycles uint64) {
	if !f.timerBStart || f.timerBCounter == 0 {
		return
	}

	remaining := cycles
	for remaining > 0 {
		if remaining < f.timerBCounter {
			f.timerBCounter -= remaining
			return
		}
		remaining -= f.timerBCounter
		f.Status |= FMStatusTimerB
		f.timerBCounter = f.timerBPeriodCycles()
		if f.timerBCounter == 0 {
			f.timerBCounter = 1
		}
	}
}

func (f *FMOPM) refreshIRQStatus() {
	irqPending := (f.timerAIRQEnable && (f.Status&FMStatusTimerA) != 0) ||
		(f.timerBIRQEnable && (f.Status&FMStatusTimerB) != 0)
	if irqPending {
		f.Status |= FMStatusIRQ
	} else {
		f.Status &^= FMStatusIRQ
	}
}

func (f *FMOPM) setBusy() {
	// Phase-1 host interface busy timing placeholder: 32 CPU/APU cycles.
	// Deterministic and hardware-friendly; can be replaced with more accurate OPM bus timing.
	f.busyCounter = 32
	f.Status |= FMStatusBusy
}

func (f *FMOPM) stepBusy(cycles uint64) {
	if f.busyCounter == 0 {
		f.Status &^= FMStatusBusy
		return
	}
	if cycles >= f.busyCounter {
		f.busyCounter = 0
		f.Status &^= FMStatusBusy
		return
	}
	f.busyCounter -= cycles
	f.Status |= FMStatusBusy
}

func fmSineLookup(phase uint32) int16 {
	idx := (phase >> fmSineTableShift) & (fmSineTableSize - 1)
	return fmSineTable[idx]
}

func tlToLinear(tl uint8) int32 {
	// TL 0 = loudest, 127 = quietest; phase-2 placeholder uses a linear attenuation.
	if tl >= 127 {
		return 0
	}
	return int32(127-tl) * 2 // 0..254
}

func fmMulToRatio(mul uint8) float64 {
	// YM-style MUL uses 0 as a special low ratio; phase-2 maps it to 1x for simplicity.
	if mul == 0 {
		return 1.0
	}
	return float64(mul)
}

func fmKeyToHz(kc, kf uint8) float64 {
	// Phase-2 OPM-lite pitch mapping:
	// Treat KeyCode as a semitone index relative to MIDI note 24 (C1),
	// with KeyFrac providing fractional semitone precision.
	semi := float64(int(kc) - 24)
	semi += float64(kf) / 256.0
	return 32.70319566257483 * math.Pow(2.0, semi/12.0) // C1 base
}

func hzToPhaseInc(hz float64, sampleRate uint32) uint32 {
	if hz <= 0 || sampleRate == 0 {
		return 0
	}
	inc := (hz * 4294967296.0) / float64(sampleRate)
	if inc <= 0 {
		return 0
	}
	if inc >= 4294967295.0 {
		return 0xFFFFFFFF
	}
	return uint32(inc)
}

func scalePhaseInc(base uint32, ratio float64) uint32 {
	if base == 0 || ratio <= 0 {
		return 0
	}
	v := float64(base) * ratio
	if v >= 4294967295.0 {
		return 0xFFFFFFFF
	}
	return uint32(v)
}
