//go:build cgo

package apu

import "nitro-core-dx/internal/debug"

func newFMRuntimeBackend(logger *debug.Logger, sampleRate uint32) fmRuntimeBackend {
	backend := newYMFMOPNABackend(sampleRate)
	if backend == nil {
		if logger != nil {
			logger.LogAPUf(debug.LogLevelWarning, "YMFM backend init failed")
		}
		return nil
	}
	return backend
}
