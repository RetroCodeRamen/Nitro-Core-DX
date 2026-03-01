//go:build linux && !wayland

package main

import (
	"encoding/binary"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

const (
	// ICCCM size hints flags
	sizeHintsPPosition   = 1 << 0
	sizeHintsPSize       = 1 << 1
	sizeHintsPMinSize    = 1 << 2
	sizeHintsPMaxSize    = 1 << 3
	sizeHintsPResizeInc  = 1 << 4
	sizeHintsPAspect    = 1 << 5
	sizeHintsPBaseSize   = 1 << 6
	sizeHintsPWinGravity = 1 << 7
)

// applyX11MaximizeHint sets WM_NORMAL_HINTS so the window manager allows
// maximize (and double-click title bar). Some WMs grey out Maximize when
// they see restrictive size hints; setting a large max size fixes that.
func applyX11MaximizeHint(w fyne.Window) error {
	nw, ok := w.(driver.NativeWindow)
	if !ok {
		return fmt.Errorf("window does not support RunNative")
	}
	var x11Window uintptr
	nw.RunNative(func(ctx any) {
		x11Ctx, ok := ctx.(driver.X11WindowContext)
		if ok && x11Ctx.WindowHandle != 0 {
			x11Window = x11Ctx.WindowHandle
		}
	})
	if x11Window == 0 {
		return fmt.Errorf("no X11 window handle")
	}

	conn, err := xgb.NewConn()
	if err != nil {
		return fmt.Errorf("xgb connect: %w", err)
	}
	defer conn.Close()

	// WM_SIZE_HINTS: 18 x 32-bit values (ICCCM). Set PMaxSize and large max dimensions
	// so the WM enables Maximize and double-click title bar.
	buf := make([]byte, 18*4)
	order := binary.LittleEndian
	// flags: set PMaxSize so max_width/max_height are used
	order.PutUint32(buf[0:], sizeHintsPMaxSize)
	// indices 7 and 8 are max_width, max_height (0-indexed)
	order.PutUint32(buf[7*4:], 65535)
	order.PutUint32(buf[8*4:], 65535)

	cookie := xproto.ChangePropertyChecked(
		conn,
		xproto.PropModeReplace,
		xproto.Window(x11Window),
		xproto.AtomWmNormalHints,
		xproto.AtomWmSizeHints,
		32,
		uint32(18),
		buf,
	)
	if err := cookie.Check(); err != nil {
		return fmt.Errorf("ChangeProperty WM_NORMAL_HINTS: %w", err)
	}
	return nil
}
