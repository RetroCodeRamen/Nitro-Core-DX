package main

import (
	"strings"
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

func TestInferSpriteLabAssetPaletteHints(t *testing.T) {
	source := `-- Sprite Lab asset (24x24, 4bpp)
-- Packed byte format: byte = (right_pixel<<4) | left_pixel
asset Slime: tileset hex
    00 00

-- Sprite Lab palette bank 3
gfx.set_palette(3, 0, 0x0000)
`
	hints := inferSpriteLabAssetPaletteHints(strings.Split(source, "\n"))
	if got, ok := hints["Slime"]; !ok || got != 3 {
		t.Fatalf("expected Slime->3 hint, got %v (ok=%v)", got, ok)
	}
}
