//go:build cgo

package ym2608

import (
	"reflect"
	"testing"
)

type psgWrite struct {
	reg  uint8
	data uint8
}

func TestPSGMatchesPMDWinCFixedTone(t *testing.T) {
	writes := []psgWrite{
		{0x00, 0x20},
		{0x01, 0x00},
		{0x02, 0x35},
		{0x03, 0x00},
		{0x07, 0x38},
		{0x08, 0x0f},
		{0x09, 0x0b},
	}

	goBuf := renderGoPSG(writes, 256)
	cBuf := renderCPSG(t, writes, 256)
	if !reflect.DeepEqual(goBuf, cBuf) {
		t.Fatalf("Go PSG fixed-tone output diverged from PMDWin C\nfirst mismatch: %s", firstMismatch(goBuf, cBuf))
	}
}

func TestPSGMatchesPMDWinCEnvelopeTone(t *testing.T) {
	writes := []psgWrite{
		{0x00, 0x10},
		{0x01, 0x00},
		{0x07, 0x3e},
		{0x08, 0x10},
		{0x0b, 0x01},
		{0x0c, 0x00},
		{0x0d, 0x0a},
	}

	goBuf := renderGoPSG(writes, 512)
	cBuf := renderCPSG(t, writes, 512)
	if !reflect.DeepEqual(goBuf, cBuf) {
		t.Fatalf("Go PSG envelope output diverged from PMDWin C\nfirst mismatch: %s", firstMismatch(goBuf, cBuf))
	}
}

func TestPSGMatchesPMDWinCNoise(t *testing.T) {
	writes := []psgWrite{
		{0x06, 0x03},
		{0x07, 0x07},
		{0x0a, 0x0f},
	}

	goBuf := renderGoPSG(writes, 256)
	cBuf := renderCPSG(t, writes, 256)
	if !reflect.DeepEqual(goBuf, cBuf) {
		t.Fatalf("Go PSG noise output diverged from PMDWin C\nfirst mismatch: %s", firstMismatch(goBuf, cBuf))
	}
}

func renderGoPSG(writes []psgWrite, samples int) []int32 {
	psg := NewPSG(DefaultMasterClockHz/8, DefaultSampleRateHz)
	for _, w := range writes {
		psg.SetReg(w.reg, w.data)
	}
	buf := make([]int32, samples)
	psg.Mix(buf)
	return buf
}

func renderCPSG(t *testing.T, writes []psgWrite, samples int) []int32 {
	t.Helper()
	out, err := renderCPSGReference(writes, samples)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func firstMismatch(a, b []int32) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return "index " + itoa(i) + ": go=" + itoa32(a[i]) + " c=" + itoa32(b[i])
		}
	}
	if len(a) != len(b) {
		return "length: go=" + itoa(len(a)) + " c=" + itoa(len(b))
	}
	return "none"
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func itoa32(v int32) string {
	return itoa(int(v))
}
