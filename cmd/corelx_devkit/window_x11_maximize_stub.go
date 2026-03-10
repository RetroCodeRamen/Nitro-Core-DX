//go:build !linux || wayland

package main

import "fyne.io/fyne/v2"

// applyX11MaximizeHint is a no-op when not on X11 (e.g. Wayland or non-Linux).
// On Wayland, maximize is controlled by the compositor; if Maximize is missing,
// try running under X11 (e.g. X11 session or WAYLAND_DISPLAY= when launching).
func applyX11MaximizeHint(w fyne.Window) error {
	return nil
}

func appendWindowHintLog(string) {}
