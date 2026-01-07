//go:build no_sdl_ttf
// +build no_sdl_ttf

package ui

import (
	"fmt"
)

// newSDLTTFRenderer stub when SDL_ttf is not available
func newSDLTTFRenderer(scale int) (TextRenderer, error) {
	return nil, fmt.Errorf("SDL_ttf not available - install libsdl2-ttf-dev (Linux) or use 'go build -tags no_sdl_ttf' to use bitmap font")
}

