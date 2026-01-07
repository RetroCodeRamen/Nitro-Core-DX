//go:build !no_sdl_ttf
// +build !no_sdl_ttf

package ui

import (
	"fmt"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

// SDLTTFRenderer uses SDL_ttf for system font rendering
type SDLTTFRenderer struct {
	font     *ttf.Font
	fontSize int
}

// newSDLTTFRenderer creates a new SDL_ttf renderer
func newSDLTTFRenderer(scale int) (TextRenderer, error) {
	// Initialize TTF
	if err := ttf.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize TTF: %w", err)
	}

	// Try to load a system font
	// Common font paths on Linux
	fontPaths := []string{
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
		"/usr/share/fonts/TTF/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf",
		"/System/Library/Fonts/Helvetica.ttc", // macOS
		"C:/Windows/Fonts/arial.ttf",          // Windows
	}

	var font *ttf.Font
	var err error
	fontSize := 12 + (scale-1)*2 // Scale font size with UI scale

	for _, path := range fontPaths {
		font, err = ttf.OpenFont(path, fontSize)
		if err == nil {
			break
		}
	}

	// If no system font found, return error
	if font == nil {
		return nil, fmt.Errorf("no system font found, tried: %v (last error: %v)", fontPaths, err)
	}

	return &SDLTTFRenderer{
		font:     font,
		fontSize: fontSize,
	}, nil
}

// DrawText draws text using SDL_ttf
func (tr *SDLTTFRenderer) DrawText(renderer *sdl.Renderer, text string, x, y int32, color sdl.Color) error {
	// Create surface from text
	surface, err := tr.font.RenderUTF8Solid(text, color)
	if err != nil {
		return fmt.Errorf("failed to render text: %w", err)
	}
	defer surface.Free()

	// Create texture from surface
	texture, err := renderer.CreateTextureFromSurface(surface)
	if err != nil {
		return fmt.Errorf("failed to create texture: %w", err)
	}
	defer texture.Destroy()

	dstRect := &sdl.Rect{X: x, Y: y, W: surface.W, H: surface.H}
	return renderer.Copy(texture, nil, dstRect)
}

// Close closes the text renderer
func (tr *SDLTTFRenderer) Close() {
	if tr.font != nil {
		tr.font.Close()
	}
	ttf.Quit()
}

