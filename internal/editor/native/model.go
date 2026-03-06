package native

import "sort"

type Caret struct {
	Offset          int
	PreferredColumn int
}

type Selection struct {
	Anchor int
	Active int
}

func (s Selection) HasSelection() bool {
	return s.Anchor != s.Active
}

func (s Selection) Range() (int, int) {
	if s.Anchor <= s.Active {
		return s.Anchor, s.Active
	}
	return s.Active, s.Anchor
}

type Model struct {
	doc        *GapBuffer
	lineStarts []int
	Caret      Caret
	Sel        Selection
}

func NewModel(text string) *Model {
	doc := NewGapBuffer(text)
	m := &Model{doc: doc}
	m.rebuildLineStarts()
	m.Caret = Caret{Offset: 0, PreferredColumn: 0}
	m.Sel = Selection{Anchor: 0, Active: 0}
	return m
}

func (m *Model) Text() string {
	if m == nil || m.doc == nil {
		return ""
	}
	return m.doc.String()
}

func (m *Model) Len() int {
	if m == nil || m.doc == nil {
		return 0
	}
	return m.doc.Len()
}

func (m *Model) SetText(text string) {
	m.doc.SetText(text)
	m.rebuildLineStarts()
	m.SetCaretOffset(0, false)
}

func (m *Model) SetCaretOffset(offset int, keepSelection bool) {
	if offset < 0 {
		offset = 0
	}
	if offset > m.Len() {
		offset = m.Len()
	}
	m.Caret.Offset = offset
	_, col := m.OffsetToLineCol(offset)
	m.Caret.PreferredColumn = col
	if keepSelection {
		m.Sel.Active = offset
	} else {
		m.Sel.Anchor = offset
		m.Sel.Active = offset
	}
}

func (m *Model) SetCaretLineCol(line, col int, keepSelection bool) {
	off := m.LineColToOffset(line, col)
	m.SetCaretOffset(off, keepSelection)
}

func (m *Model) OffsetToLineCol(offset int) (line, col int) {
	if offset < 0 {
		offset = 0
	}
	if offset > m.Len() {
		offset = m.Len()
	}
	if len(m.lineStarts) == 0 {
		return 0, 0
	}
	line = sort.Search(len(m.lineStarts), func(i int) bool { return m.lineStarts[i] > offset }) - 1
	if line < 0 {
		line = 0
	}
	col = offset - m.lineStarts[line]
	return line, col
}

func (m *Model) LineColToOffset(line, col int) int {
	if len(m.lineStarts) == 0 {
		return 0
	}
	if line < 0 {
		line = 0
	}
	if line >= len(m.lineStarts) {
		line = len(m.lineStarts) - 1
	}
	start := m.lineStarts[line]
	end := m.Len()
	if line+1 < len(m.lineStarts) {
		end = m.lineStarts[line+1] - 1 // exclude newline
		if end < start {
			end = start
		}
	}
	lineLen := end - start
	if col < 0 {
		col = 0
	}
	if col > lineLen {
		col = lineLen
	}
	return start + col
}

func (m *Model) LineCount() int {
	if len(m.lineStarts) == 0 {
		return 1
	}
	return len(m.lineStarts)
}

func (m *Model) LineText(line int) string {
	if len(m.lineStarts) == 0 {
		return ""
	}
	if line < 0 || line >= len(m.lineStarts) {
		return ""
	}
	start := m.lineStarts[line]
	end := m.Len()
	if line+1 < len(m.lineStarts) {
		end = m.lineStarts[line+1] - 1 // drop trailing newline
	}
	if end < start {
		end = start
	}
	return m.doc.Slice(start, end)
}

func (m *Model) InsertRune(r rune) {
	if m.HasSelection() {
		m.DeleteSelection()
	}
	m.doc.Insert(m.Caret.Offset, []rune{r})
	m.Caret.Offset++
	m.rebuildLineStarts()
	_, col := m.OffsetToLineCol(m.Caret.Offset)
	m.Caret.PreferredColumn = col
	m.Sel.Anchor = m.Caret.Offset
	m.Sel.Active = m.Caret.Offset
}

func (m *Model) InsertText(text string) {
	r := []rune(text)
	if len(r) == 0 {
		return
	}
	if m.HasSelection() {
		m.DeleteSelection()
	}
	m.doc.Insert(m.Caret.Offset, r)
	m.Caret.Offset += len(r)
	m.rebuildLineStarts()
	_, col := m.OffsetToLineCol(m.Caret.Offset)
	m.Caret.PreferredColumn = col
	m.Sel.Anchor = m.Caret.Offset
	m.Sel.Active = m.Caret.Offset
}

func (m *Model) Backspace() {
	if m.HasSelection() {
		m.DeleteSelection()
		return
	}
	if m.Caret.Offset <= 0 {
		return
	}
	m.doc.Delete(m.Caret.Offset-1, m.Caret.Offset)
	m.Caret.Offset--
	m.rebuildLineStarts()
	_, col := m.OffsetToLineCol(m.Caret.Offset)
	m.Caret.PreferredColumn = col
	m.Sel.Anchor = m.Caret.Offset
	m.Sel.Active = m.Caret.Offset
}

func (m *Model) DeleteForward() {
	if m.HasSelection() {
		m.DeleteSelection()
		return
	}
	if m.Caret.Offset >= m.Len() {
		return
	}
	m.doc.Delete(m.Caret.Offset, m.Caret.Offset+1)
	m.rebuildLineStarts()
	_, col := m.OffsetToLineCol(m.Caret.Offset)
	m.Caret.PreferredColumn = col
	m.Sel.Anchor = m.Caret.Offset
	m.Sel.Active = m.Caret.Offset
}

func (m *Model) DeleteSelection() {
	if !m.HasSelection() {
		return
	}
	start, end := m.Sel.Range()
	m.doc.Delete(start, end)
	m.rebuildLineStarts()
	m.Caret.Offset = start
	_, col := m.OffsetToLineCol(m.Caret.Offset)
	m.Caret.PreferredColumn = col
	m.Sel.Anchor = start
	m.Sel.Active = start
}

func (m *Model) MoveLeft(extend bool) {
	if m.Caret.Offset <= 0 {
		return
	}
	m.SetCaretOffset(m.Caret.Offset-1, extend)
}

func (m *Model) MoveRight(extend bool) {
	if m.Caret.Offset >= m.Len() {
		return
	}
	m.SetCaretOffset(m.Caret.Offset+1, extend)
}

func (m *Model) MoveUp(extend bool) {
	line, _ := m.OffsetToLineCol(m.Caret.Offset)
	if line <= 0 {
		m.SetCaretLineCol(0, 0, extend)
		return
	}
	m.SetCaretLineCol(line-1, m.Caret.PreferredColumn, extend)
}

func (m *Model) MoveDown(extend bool) {
	line, _ := m.OffsetToLineCol(m.Caret.Offset)
	if line >= m.LineCount()-1 {
		m.SetCaretOffset(m.Len(), extend)
		return
	}
	m.SetCaretLineCol(line+1, m.Caret.PreferredColumn, extend)
}

func (m *Model) MoveLineHome(extend bool) {
	line, _ := m.OffsetToLineCol(m.Caret.Offset)
	m.SetCaretLineCol(line, 0, extend)
}

func (m *Model) MoveLineEnd(extend bool) {
	line, _ := m.OffsetToLineCol(m.Caret.Offset)
	off := m.LineColToOffset(line, 1<<30)
	m.SetCaretOffset(off, extend)
}

func (m *Model) SelectAll() {
	m.Sel.Anchor = 0
	m.Sel.Active = m.Len()
	m.Caret.Offset = m.Len()
	_, col := m.OffsetToLineCol(m.Caret.Offset)
	m.Caret.PreferredColumn = col
}

func (m *Model) HasSelection() bool {
	return m.Sel.HasSelection()
}

func (m *Model) SelectedText() string {
	if !m.HasSelection() {
		return ""
	}
	start, end := m.Sel.Range()
	return m.doc.Slice(start, end)
}

func (m *Model) rebuildLineStarts() {
	text := []rune(m.doc.String())
	m.lineStarts = m.lineStarts[:0]
	m.lineStarts = append(m.lineStarts, 0)
	for i, r := range text {
		if r == '\n' {
			m.lineStarts = append(m.lineStarts, i+1)
		}
	}
	if len(m.lineStarts) == 0 {
		m.lineStarts = []int{0}
	}
}
