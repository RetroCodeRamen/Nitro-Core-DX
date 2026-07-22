//go:build cgo

package ym2608

/*
#cgo CFLAGS: -I${SRCDIR}/../../Resources/PMDWinS036-master/fmgen
#cgo LDFLAGS: -lm
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include "../../Resources/PMDWinS036-master/fmgen/psg.c"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func renderCPSGReference(writes []psgWrite, samples int) ([]int32, error) {
	var psg C.PSG
	C.PSGInit(&psg)
	C.PSGSetClock(&psg, C.uint32_t(DefaultMasterClockHz/8), C.uint32_t(DefaultSampleRateHz))
	C.PSGSetChannelMask(&psg, C.int(0x3f))
	for _, w := range writes {
		C.PSGSetReg(&psg, C.uint8_t(w.reg), C.uint8_t(w.data))
	}

	bytes := C.size_t(samples) * C.size_t(unsafe.Sizeof(C.int32_t(0)))
	ptr := C.malloc(bytes)
	if ptr == nil {
		return nil, fmt.Errorf("malloc C PSG output buffer")
	}
	defer C.free(ptr)
	C.memset(ptr, 0, bytes)

	C.PSGMix(&psg, (*C.int32_t)(ptr), C.uint32_t(samples))
	cSlice := unsafe.Slice((*C.int32_t)(ptr), samples)
	out := make([]int32, samples)
	for i, sample := range cSlice {
		out[i] = int32(sample)
	}
	return out, nil
}
