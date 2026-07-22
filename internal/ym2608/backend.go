package ym2608

// Backend exposes the same method shape as the APU runtime backend without
// importing internal/apu. A later build-tagged factory can return this type as
// the pure-Go YM2608 replacement while the current main build remains untouched.
type Backend struct {
	chip    *Chip
	enabled bool
	muted   bool
}

func NewBackend(cfg Config) *Backend {
	return &Backend{chip: New(cfg)}
}

func (b *Backend) Chip() *Chip {
	return b.chip
}

func (b *Backend) Read8(offset uint16) uint8 {
	if b == nil || b.chip == nil {
		return 0
	}
	return b.chip.ReadPort(offset)
}

func (b *Backend) Write8(offset uint16, value uint8) {
	if b == nil || b.chip == nil {
		return
	}
	b.chip.WritePort(offset, value)
}

func (b *Backend) GenerateSampleFixed() int16 {
	if b == nil || b.chip == nil || !b.enabled || b.muted {
		return 0
	}
	left, right := b.chip.GenerateSampleFixed()
	return int16((int32(left) + int32(right)) / 2)
}

func (b *Backend) Step(cycles uint64) {
	if b == nil || b.chip == nil {
		return
	}
	b.chip.Step(cycles)
}

func (b *Backend) IRQPending() bool {
	if b == nil || b.chip == nil {
		return false
	}
	return b.chip.IRQPending()
}

func (b *Backend) Reset() {
	if b == nil || b.chip == nil {
		return
	}
	b.chip.Reset()
}

func (b *Backend) SetEnabledMuted(enabled, muted bool) {
	if b == nil {
		return
	}
	b.enabled = enabled
	b.muted = muted
}

func (b *Backend) SetSampleRate(sampleRate uint32) {
	if b == nil || b.chip == nil || sampleRate == 0 {
		return
	}
	b.chip.SetSampleRate(sampleRate)
}
