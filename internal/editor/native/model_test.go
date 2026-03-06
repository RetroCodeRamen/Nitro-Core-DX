package native

import "testing"

func TestModelInsertDelete(t *testing.T) {
	m := NewModel("")
	m.InsertRune('a')
	m.InsertRune('b')
	m.InsertRune('c')
	if got := m.Text(); got != "abc" {
		t.Fatalf("text = %q, want abc", got)
	}
	m.Backspace()
	if got := m.Text(); got != "ab" {
		t.Fatalf("text = %q, want ab", got)
	}
	m.MoveLeft(false)
	m.DeleteForward()
	if got := m.Text(); got != "a" {
		t.Fatalf("text = %q, want a", got)
	}
}

func TestModelSelectionReplace(t *testing.T) {
	m := NewModel("hello")
	m.SetCaretLineCol(0, 1, false)
	m.SetCaretLineCol(0, 4, true)
	if got := m.SelectedText(); got != "ell" {
		t.Fatalf("selected = %q, want ell", got)
	}
	m.InsertText("i")
	if got := m.Text(); got != "hio" {
		t.Fatalf("text = %q, want hio", got)
	}
}

func TestModelLineColRoundTrip(t *testing.T) {
	m := NewModel("ab\ncd\nxyz")
	for line := 0; line < m.LineCount(); line++ {
		for col := 0; col <= len([]rune(m.LineText(line))); col++ {
			off := m.LineColToOffset(line, col)
			l2, c2 := m.OffsetToLineCol(off)
			if l2 != line || c2 != col {
				t.Fatalf("roundtrip mismatch line=%d col=%d => line=%d col=%d", line, col, l2, c2)
			}
		}
	}
}
