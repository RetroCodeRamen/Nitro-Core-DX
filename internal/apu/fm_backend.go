package apu

// fmRuntimeBackend is an optional runtime backend for the FM extension host
// interface. The in-tree OPM-lite model remains the default path; external
// backends (for example YMFM OPNA) can be selected without changing APU MMIO
// wiring.
type fmRuntimeBackend interface {
	Read8(offset uint16) uint8
	Write8(offset uint16, value uint8)
	GenerateSampleFixed() int16
	Step(cycles uint64)
	IRQPending() bool
	Reset()
	SetEnabledMuted(enabled, muted bool)
	SetSampleRate(sampleRate uint32)
}
