package corelx

import "testing"

// TestDrawIntDigits verifies text.draw_int renders integers as the correct
// decimal digit characters at the text port: leading zeros suppressed, a
// leading '-' for negatives. Asserts the exact buffered characters against the
// real PPU.
func TestDrawIntDigits(t *testing.T) {
	source := `function Start()
    text.draw_int(0, 0, 255, 255, 255, 0)
    text.draw_int(0, 10, 255, 255, 255, 42)
    text.draw_int(0, 20, 255, 255, 255, 1234)
    text.draw_int(0, 30, 255, 255, 255, 0 - 7)
    while true
        wait_vblank()
`
	emu, _ := compileAndBoot(t, source, 3000)
	got := emu.PPU.GetBufferedText()
	want := "0" + "42" + "1234" + "-7"
	if got != want {
		t.Errorf("draw_int output: want %q, got %q", want, got)
	}
}

// TestDrawIntFromGlobal verifies a runtime-computed value (not a literal)
// renders correctly — the realistic case of a live score/counter.
func TestDrawIntFromGlobal(t *testing.T) {
	source := `var score: int = 0
function Start()
    score = 100
    score = score + 56
    text.draw_int(8, 8, 255, 255, 0, score)
    while true
        wait_vblank()
`
	emu, _ := compileAndBoot(t, source, 1000)
	if got := emu.PPU.GetBufferedText(); got != "156" {
		t.Errorf("draw_int(score=156): want \"156\", got %q", got)
	}
}
