package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	fynetest "fyne.io/fyne/v2/test"
)

func TestHandleTypedRuneFallsBackToSourceEditorWhenNoFocus(t *testing.T) {
	a := fynetest.NewApp()
	defer a.Quit()

	editor := newCoreLXCodeEditor()
	w := fynetest.NewWindow(editor)
	defer w.Close()
	w.Resize(fyne.NewSize(640, 480))
	w.Show()

	state := &devKitState{
		window:           w,
		sourceEditor:     editor,
		captureGameInput: false,
	}

	// Ensure no focused widget to simulate canvas-level typed-rune path.
	w.Canvas().Focus(nil)
	state.handleTypedRune('x')

	if got := editor.Text(); got != "x" {
		t.Fatalf("expected fallback typed rune to edit source, got %q", got)
	}
}

func TestHandleKeyDownUpForwardedToSourceEditorForShiftSelection(t *testing.T) {
	a := fynetest.NewApp()
	defer a.Quit()

	editor := newCoreLXCodeEditor()
	w := fynetest.NewWindow(editor)
	defer w.Close()
	w.Resize(fyne.NewSize(640, 480))
	w.Show()

	state := &devKitState{
		window:           w,
		sourceEditor:     editor,
		captureGameInput: false,
	}

	editor.SetText("hello")
	editor.SetCursor(0, 0)
	w.Canvas().Focus(editor)

	state.handleKeyDown(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
	state.handleTypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	state.handleTypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	state.handleKeyUp(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})

	if got := editor.SelectedText(); got == "" {
		t.Fatalf("expected selection after forwarded Shift+Right input")
	}
}
