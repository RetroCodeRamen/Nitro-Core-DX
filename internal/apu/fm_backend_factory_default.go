//go:build !cgo

package apu

import "nitro-core-dx/internal/debug"

func newFMRuntimeBackend(_ *debug.Logger, _ uint32) fmRuntimeBackend {
	return nil
}
