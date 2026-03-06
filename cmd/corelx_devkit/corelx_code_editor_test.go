package main

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	fynetest "fyne.io/fyne/v2/test"
)

func TestCoreLXCodeEditorTypingAfterTap(t *testing.T) {
	a := fynetest.NewApp()
	defer a.Quit()

	editor := newCoreLXCodeEditor()
	w := fynetest.NewWindow(editor)
	defer w.Close()
	w.Resize(fyne.NewSize(640, 480))
	w.Show()

	fynetest.Tap(editor)
	fynetest.Type(editor, "abc")

	if got := editor.Text(); got != "abc" {
		t.Fatalf("expected typed text \"abc\", got %q", got)
	}
}

func TestCoreLXCodeEditorShiftSelection(t *testing.T) {
	a := fynetest.NewApp()
	defer a.Quit()

	editor := newCoreLXCodeEditor()
	w := fynetest.NewWindow(editor)
	defer w.Close()
	w.Resize(fyne.NewSize(640, 480))
	w.Show()

	editor.SetText("hello")
	editor.SetCursor(0, 0)
	w.Canvas().Focus(editor)

	editor.KeyDown(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
	editor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	editor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	editor.KeyUp(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})

	if got := editor.SelectedText(); got == "" {
		t.Fatalf("expected non-empty selected text after Shift+Right")
	}
}

func TestCoreLXCodeEditorTypingBurstIsResponsive(t *testing.T) {
	a := fynetest.NewApp()
	defer a.Quit()

	editor := newCoreLXCodeEditor()
	w := fynetest.NewWindow(editor)
	defer w.Close()
	w.Resize(fyne.NewSize(640, 480))
	w.Show()
	fynetest.Tap(editor)

	start := time.Now()
	for i := 0; i < 400; i++ {
		editor.TypedRune('a')
	}
	elapsed := time.Since(start)

	if len(editor.Text()) != 400 {
		t.Fatalf("expected 400 chars typed, got %d", len(editor.Text()))
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("typing burst too slow: %s", elapsed)
	}
}
