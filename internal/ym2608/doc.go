// Package ym2608 contains the isolated pure-Go YM2608/OPNA port.
//
// This package is intentionally not wired into the main APU build path yet.
// It gives the C-to-Go audio port a focused module and test surface so the
// implementation can mature independently, then be selected later with a small
// build-tagged adapter in internal/apu.
package ym2608
