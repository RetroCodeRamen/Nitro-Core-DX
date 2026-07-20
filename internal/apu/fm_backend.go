package apu

// fmRuntimeBackend is the runtime backend for the YM2608 audio subsystem host
// interface. The active runtime uses the YMFM OPNA backend (cgo builds); builds
// without it use the in-tree FM model. This backend choice is an internal
// implementation detail (not a user-facing audio mode) and does not change APU
// MMIO wiring.
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
