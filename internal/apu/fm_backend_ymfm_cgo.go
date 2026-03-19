//go:build cgo

package apu

/*
#cgo CXXFLAGS: -std=c++17 -I${SRCDIR}/../../Resources/ymfm-main/src -I${SRCDIR}
#cgo CFLAGS: -I${SRCDIR}
#cgo LDFLAGS: -lstdc++
#include "fm_backend_ymfm_bridge.h"
*/
import "C"

import "runtime"

const ymfmMonoNormalize = 16384.0
const ymfmMasterClockHz = 8000000

type ymfmOPNABackend struct {
	handle *C.ncdx_ymfm_opna

	hostSampleRate uint32
	chipSampleRate uint32
	sampleRema     uint64
	busyCounter    uint64

	addr0 uint8
	addr1 uint8
	regs0 [256]uint8
	regs1 [256]uint8
}

func newYMFMOPNABackend(hostSampleRate uint32) *ymfmOPNABackend {
	if hostSampleRate == 0 {
		hostSampleRate = fmDefaultHz
	}
	h := C.ncdx_ymfm_opna_create(C.uint32_t(ymfmMasterClockHz))
	if h == nil {
		return nil
	}
	b := &ymfmOPNABackend{
		handle:         h,
		hostSampleRate: hostSampleRate,
		chipSampleRate: uint32(C.ncdx_ymfm_opna_sample_rate(h)),
	}
	if b.chipSampleRate == 0 {
		b.chipSampleRate = hostSampleRate
	}
	runtime.SetFinalizer(b, (*ymfmOPNABackend).finalize)
	return b
}

func (b *ymfmOPNABackend) finalize() {
	if b.handle != nil {
		C.ncdx_ymfm_opna_destroy(b.handle)
		b.handle = nil
	}
}

func (b *ymfmOPNABackend) Read8(offset uint16) uint8 {
	if b.handle == nil {
		return 0
	}
	switch offset {
	case FMRegAddr:
		return b.addr0
	case FMRegData:
		return b.regs0[b.addr0]
	case FMRegStatus:
		raw := uint8(C.ncdx_ymfm_opna_read_port(b.handle, C.uint16_t(offset)))
		status := uint8(0)
		if raw&0x01 != 0 {
			status |= FMStatusTimerA
		}
		if raw&0x02 != 0 {
			status |= FMStatusTimerB
		}
		if b.busyCounter > 0 {
			status |= FMStatusBusy
		}
		if b.IRQPending() {
			status |= FMStatusIRQ
		}
		return status
	case FMRegMixL:
		return b.addr1
	case FMRegMixR:
		return b.regs1[b.addr1]
	default:
		return uint8(C.ncdx_ymfm_opna_read_port(b.handle, C.uint16_t(offset)))
	}
}

func (b *ymfmOPNABackend) Write8(offset uint16, value uint8) {
	if b.handle == nil {
		return
	}
	switch offset {
	case FMRegAddr:
		b.addr0 = value
		b.setBusy()
	case FMRegData:
		b.regs0[b.addr0] = value
		b.setBusy()
		b.writePort0DataTranslated(value)
	case FMRegMixL:
		b.addr1 = value
		b.setBusy()
		C.ncdx_ymfm_opna_write_port(b.handle, C.uint16_t(FMRegMixL), C.uint8_t(value))
	case FMRegMixR:
		b.regs1[b.addr1] = value
		b.setBusy()
		C.ncdx_ymfm_opna_write_port(b.handle, C.uint16_t(FMRegMixR), C.uint8_t(value))
	default:
		C.ncdx_ymfm_opna_write_port(b.handle, C.uint16_t(offset), C.uint8_t(value))
	}
}

func (b *ymfmOPNABackend) GenerateSampleFixed() int16 {
	if b.handle == nil {
		return 0
	}

	steps := uint64(1)
	if b.chipSampleRate > b.hostSampleRate && b.hostSampleRate != 0 {
		b.sampleRema += uint64(b.chipSampleRate)
		steps = b.sampleRema / uint64(b.hostSampleRate)
		if steps == 0 {
			steps = 1
		}
		b.sampleRema -= steps * uint64(b.hostSampleRate)
		if steps > 64 {
			steps = 64
			b.sampleRema = 0
		}
	}

	var monoSum int64
	for i := uint64(0); i < steps; i++ {
		var l, r C.int32_t
		C.ncdx_ymfm_opna_generate_sample(b.handle, &l, &r)
		monoSum += int64(l+r) / 2
	}

	s := float32(monoSum) / float32(steps) / ymfmMonoNormalize
	if s > 1.0 {
		s = 1.0
	} else if s < -1.0 {
		s = -1.0
	}
	return int16(s * 32767.0)
}

func (b *ymfmOPNABackend) Step(cycles uint64) {
	if b.handle == nil || cycles == 0 {
		return
	}
	if b.busyCounter > 0 {
		if cycles >= b.busyCounter {
			b.busyCounter = 0
		} else {
			b.busyCounter -= cycles
		}
	}
	C.ncdx_ymfm_opna_step_clocks(b.handle, C.uint64_t(cycles))
}

func (b *ymfmOPNABackend) IRQPending() bool {
	if b.handle == nil {
		return false
	}
	return C.ncdx_ymfm_opna_irq_pending(b.handle) != 0
}

func (b *ymfmOPNABackend) Reset() {
	if b.handle == nil {
		return
	}
	C.ncdx_ymfm_opna_reset(b.handle)
	b.sampleRema = 0
	b.busyCounter = 0
	b.addr0 = 0
	b.addr1 = 0
	for i := range b.regs0 {
		b.regs0[i] = 0
		b.regs1[i] = 0
	}
}

func (b *ymfmOPNABackend) SetEnabledMuted(_ bool, _ bool) {
	// Host-side FM control gates output in FMOPM before backend sampling.
}

func (b *ymfmOPNABackend) SetSampleRate(sampleRate uint32) {
	if sampleRate == 0 {
		return
	}
	b.hostSampleRate = sampleRate
}

func (b *ymfmOPNABackend) setBusy() {
	b.busyCounter = 32
}

func (b *ymfmOPNABackend) writeYM(addr, data uint8) {
	C.ncdx_ymfm_opna_write_port(b.handle, C.uint16_t(FMRegAddr), C.uint8_t(addr))
	C.ncdx_ymfm_opna_write_port(b.handle, C.uint16_t(FMRegData), C.uint8_t(data))
}

func (b *ymfmOPNABackend) writePort0DataTranslated(value uint8) {
	switch b.addr0 {
	case fmOPMRegTimerAHi:
		// OPM 0x10 -> YM2608 Timer A high 0x24
		b.writeYM(0x24, value)
	case fmOPMRegTimerALo:
		// OPM 0x11 -> YM2608 Timer A low (2 bits) 0x25
		b.writeYM(0x25, value&0x03)
	case fmOPMRegTimerB:
		// OPM 0x12 -> YM2608 Timer B 0x26
		b.writeYM(0x26, value)
	case fmOPMRegTimerCtrl:
		// Translate OPM-ish timer control into OPNA timer + IRQ mask controls.
		ym27 := uint8(0)
		if value&0x01 != 0 {
			ym27 |= 0x01 // load/start A
		}
		if value&0x02 != 0 {
			ym27 |= 0x02 // load/start B
		}
		if value&0x10 != 0 {
			ym27 |= 0x04 // enable A flag/IRQ source
		}
		if value&0x20 != 0 {
			ym27 |= 0x08 // enable B flag/IRQ source
		}
		if value&0x04 != 0 {
			ym27 |= 0x10 // clear A flag
		}
		if value&0x08 != 0 {
			ym27 |= 0x20 // clear B flag
		}
		b.writeYM(0x27, ym27)

		irqMask := uint8(0)
		if value&0x10 != 0 {
			irqMask |= 0x01
		}
		if value&0x20 != 0 {
			irqMask |= 0x02
		}
		b.writeYM(0x29, irqMask)
	default:
		b.writeYM(b.addr0, value)
	}
}
