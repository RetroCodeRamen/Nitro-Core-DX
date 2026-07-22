package ym2608

import "math"

const rhythmVoiceCount = 6

var clipTable = makeClipTable()

type RhythmUnit struct {
	voices [rhythmVoiceCount]rhythmVoice

	totalLevel int
	keyMask    uint8
	outputRate uint32
}

type rhythmVoice struct {
	pan    uint8
	level  int
	volume int

	sample []int8
	pos    uint32
	size   uint32
	step   uint32
	rate   uint32
}

func NewRhythmUnit(outputRate uint32) *RhythmUnit {
	if outputRate == 0 {
		outputRate = DefaultSampleRateHz
	}
	r := &RhythmUnit{outputRate: outputRate}
	for i := range r.voices {
		r.voices[i].pos = ^uint32(0)
		r.voices[i].level = 0
		r.voices[i].pan = 0
	}
	r.LoadDefaultSamples()
	return r
}

func (r *RhythmUnit) LoadDefaultSamples() {
	for i := range defaultRhythmSamples {
		if len(defaultRhythmSamples[i]) == 0 {
			continue
		}
		r.SetSample(i, defaultRhythmSamples[i], DefaultSampleRateHz)
	}
}

func (r *RhythmUnit) Reset() {
	r.keyMask = 0
	r.totalLevel = 0
	for i := range r.voices {
		r.voices[i].pos = ^uint32(0)
	}
}

func (r *RhythmUnit) SetOutputRate(outputRate uint32) {
	if outputRate == 0 {
		return
	}
	r.outputRate = outputRate
	for i := range r.voices {
		r.recomputeStep(i)
	}
}

func (r *RhythmUnit) SetSample(index int, sample []int8, sampleRate uint32) {
	if index < 0 || index >= len(r.voices) {
		return
	}
	if sampleRate == 0 {
		sampleRate = DefaultSampleRateHz
	}
	v := &r.voices[index]
	v.sample = append(v.sample[:0], sample...)
	v.rate = sampleRate
	v.size = uint32(len(v.sample)) * 1024
	v.pos = v.size
	r.recomputeStep(index)
}

func (r *RhythmUnit) SetVolume(index int, db int) {
	if index < 0 || index >= len(r.voices) {
		return
	}
	if db > 20 {
		db = 20
	}
	r.voices[index].volume = 16 - (db * 2 / 3)
}

func (r *RhythmUnit) WriteReg(addr, data uint8) {
	switch addr {
	case 0x10:
		r.writeKey(data)
	case 0x11:
		r.totalLevel = int(^data & 0x3f)
	case 0x18, 0x19, 0x1b, 0x1c, 0x1d:
		v := &r.voices[addr&0x07]
		v.pan = (data >> 6) & 0x03
		v.level = int(^data & 0x1f)
	case 0x1a:
		// The C source leaves Top Cymbal parameter writes as a no-op.
	}
}

func (r *RhythmUnit) Mix(dest []int32) {
	if r == nil || r.keyMask&0x3f == 0 || len(r.voices[0].sample) == 0 {
		return
	}

	for i := range r.voices {
		v := &r.voices[i]
		if r.keyMask&(1<<i) == 0 || v.level < 0 {
			continue
		}
		db := clampInt(r.totalLevel+v.level+v.volume, 0, 127)
		vol := int32(clipTable[db])

		for j := 0; j < len(dest) && v.pos < v.size; j++ {
			raw := int32(v.sample[v.pos>>10]) << 8
			sample := limit16ToInt32((raw * vol) >> 10)
			v.pos += v.step
			dest[j] += sample
		}
	}
}

func (r *RhythmUnit) writeKey(data uint8) {
	if data&0x80 == 0 {
		mask := data & 0x3f
		r.keyMask |= mask
		for i := range r.voices {
			if mask&(1<<i) != 0 {
				r.voices[i].pos = 0
			}
		}
		return
	}
	r.keyMask &^= data & 0x3f
}

func (r *RhythmUnit) recomputeStep(index int) {
	v := &r.voices[index]
	if r.outputRate == 0 {
		r.outputRate = DefaultSampleRateHz
	}
	if v.rate == 0 {
		v.step = 0
		return
	}
	v.step = v.rate * 1024 / r.outputRate
}

func makeClipTable() [512]uint8 {
	var table [512]uint8
	for i := range table {
		f := float32(255.0) * float32(math.Exp(float64(float32(math.Ln2)*(-float32(i)/64.0))))
		table[i] = uint8(f)
	}
	return table
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
