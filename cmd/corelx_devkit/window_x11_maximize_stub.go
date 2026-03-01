//go:build !linux || wayland

package main

import "fyne.io/fyne/v2"

func applyX11MaximizeHint(w fyne.Window) error {
	return nil // no-op when not on X11
}
