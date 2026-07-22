//go:build cgo

package ym2608

/*
#cgo CFLAGS: -I${SRCDIR}/../../Resources/PMDWinS036-master/fmgen
#cgo LDFLAGS: -lm
#include <stdint.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#define printf(...) ((int)0)
#include "../../Resources/PMDWinS036-master/fmgen/opna.c"
#undef printf
#include "../../Resources/PMDWinS036-master/fmgen/rhythmdata.c"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type opnaWrite struct {
	addr uint16
	data uint8
}

type opnaRhythmVolume struct {
	index int
	db    int
}

type opnaEvent struct {
	at             int
	writes         []opnaWrite
	rhythmVolumes  []opnaRhythmVolume
	channelMask    uint32
	setChannelMask bool
	sampleRate     uint32
	interpolation  bool
	setSampleRate  bool
}

type opnaTimerResult struct {
	event  bool
	status uint8
}

func renderCOPNAReference(writes []opnaWrite, samples int) ([]int16, error) {
	return renderCOPNAReferenceEvents([]opnaEvent{{writes: writes}}, samples)
}

func renderCOPNAReferenceEvents(events []opnaEvent, samples int) ([]int16, error) {
	return renderCOPNAReferenceEventsWithRate(events, samples, DefaultSampleRateHz, false)
}

func renderCOPNAReferenceWithRate(writes []opnaWrite, samples int, sampleRate uint32) ([]int16, error) {
	return renderCOPNAReferenceEventsWithRate([]opnaEvent{{writes: writes}}, samples, sampleRate, false)
}

func renderCOPNAReferenceWithRateAndInterpolation(writes []opnaWrite, samples int, sampleRate uint32, interpolation bool) ([]int16, error) {
	return renderCOPNAReferenceEventsWithRate([]opnaEvent{{writes: writes}}, samples, sampleRate, interpolation)
}

func renderCOPNAReferenceEventsWithRate(events []opnaEvent, samples int, sampleRate uint32, interpolation bool) ([]int16, error) {
	opnaPtr := C.malloc(C.size_t(unsafe.Sizeof(C.OPNA{})))
	if opnaPtr == nil {
		return nil, fmt.Errorf("malloc C OPNA")
	}
	defer C.free(opnaPtr)
	C.memset(opnaPtr, 0, C.size_t(unsafe.Sizeof(C.OPNA{})))

	opna := (*C.OPNA)(opnaPtr)
	if sampleRate == 0 {
		sampleRate = DefaultSampleRateHz
	}
	ipflag := C.uint8_t(0)
	if interpolation {
		ipflag = 1
	}
	if C.OPNAInit(opna, C.uint32_t(DefaultMasterClockHz), C.uint32_t(sampleRate), ipflag) == 0 {
		return nil, fmt.Errorf("OPNAInit failed")
	}

	bytes := C.size_t(samples) * C.size_t(unsafe.Sizeof(C.int16_t(0)))
	ptr := C.malloc(bytes)
	if ptr == nil {
		return nil, fmt.Errorf("malloc C OPNA output buffer")
	}
	defer C.free(ptr)
	C.memset(ptr, 0, bytes)

	cursor := 0
	for _, event := range events {
		if event.at < cursor || event.at > samples {
			return nil, fmt.Errorf("invalid OPNA event offset %d after cursor %d for %d samples", event.at, cursor, samples)
		}
		if event.at > cursor {
			renderCChunk(opna, ptr, cursor, event.at-cursor)
			cursor = event.at
		}
		applyCOPNAEvent(opna, event)
	}
	if cursor < samples {
		renderCChunk(opna, ptr, cursor, samples-cursor)
	}

	cSlice := unsafe.Slice((*C.int16_t)(ptr), samples)
	out := make([]int16, samples)
	for i, sample := range cSlice {
		out[i] = int16(sample)
	}
	return out, nil
}

func runCOPNATimerReference(writes []opnaWrite, steps []int32) ([]opnaTimerResult, error) {
	opnaPtr := C.malloc(C.size_t(unsafe.Sizeof(C.OPNA{})))
	if opnaPtr == nil {
		return nil, fmt.Errorf("malloc C OPNA")
	}
	defer C.free(opnaPtr)
	C.memset(opnaPtr, 0, C.size_t(unsafe.Sizeof(C.OPNA{})))

	opna := (*C.OPNA)(opnaPtr)
	if C.OPNAInit(opna, C.uint32_t(DefaultMasterClockHz), C.uint32_t(DefaultSampleRateHz), 0) == 0 {
		return nil, fmt.Errorf("OPNAInit failed")
	}
	for _, w := range writes {
		C.OPNASetReg(opna, C.uint32_t(w.addr), C.uint32_t(w.data))
	}

	out := make([]opnaTimerResult, len(steps))
	for i, us := range steps {
		event := C.OPNATimerCount(opna, C.int32_t(us))
		out[i] = opnaTimerResult{
			event:  event != 0,
			status: uint8(opna.status & 0x03),
		}
	}
	return out, nil
}

func applyCOPNAEvent(opna *C.OPNA, event opnaEvent) {
	if event.setChannelMask {
		C.OPNASetChannelMask(opna, C.uint32_t(event.channelMask))
	}
	if event.setSampleRate {
		ipflag := C.uint8_t(0)
		if event.interpolation {
			ipflag = 1
		}
		C.OPNASetRate(opna, C.uint32_t(event.sampleRate), ipflag)
	}
	for _, volume := range event.rhythmVolumes {
		C.SetVolumeRhythm(opna, C.uint(volume.index), C.int(volume.db))
	}
	for _, w := range event.writes {
		C.OPNASetReg(opna, C.uint32_t(w.addr), C.uint32_t(w.data))
	}
}

func renderCChunk(opna *C.OPNA, base unsafe.Pointer, offsetSamples, samples int) {
	offset := uintptr(offsetSamples) * unsafe.Sizeof(C.int16_t(0))
	C.OPNAMix(opna, (*C.int16_t)(unsafe.Add(base, offset)), C.uint32_t(samples))
}
