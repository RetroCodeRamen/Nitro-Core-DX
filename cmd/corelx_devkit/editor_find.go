package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// findEditorSpan searches text for query. Offsets are rune indexes, matching
// the native editor model. An empty query never matches. The search wraps once.
func findEditorSpan(text, query string, from int, forward, matchCase bool) (start, end int, wrapped, ok bool) {
	hay := []rune(text)
	q := []rune(query)
	if !matchCase {
		hay = lowerRunes(hay)
		q = lowerRunes(q)
	}
	if len(q) == 0 || len(hay) < len(q) {
		return 0, 0, false, false
	}
	if from < 0 {
		from = 0
	}
	if from > len(hay) {
		from = len(hay)
	}
	if forward {
		if i := indexRunesFrom(hay, q, from); i >= 0 {
			return i, i + len(q), false, true
		}
		if from > 0 {
			if i := indexRunesFrom(hay, q, 0); i >= 0 && i < from {
				return i, i + len(q), true, true
			}
		}
		return 0, 0, false, false
	}
	if i := lastIndexRunesBefore(hay, q, from); i >= 0 {
		return i, i + len(q), false, true
	}
	if from < len(hay) {
		if i := lastIndexRunesBefore(hay, q, len(hay)+1); i >= from {
			return i, i + len(q), true, true
		}
	}
	return 0, 0, false, false
}

// replaceAllEditorMatches replaces non-overlapping matches. Text inserted by
// the replacement is not searched again.
func replaceAllEditorMatches(text, query, replacement string, matchCase bool) (string, int) {
	src := []rune(text)
	hay := src
	q := []rune(query)
	if !matchCase {
		hay = lowerRunes(append([]rune(nil), src...))
		q = lowerRunes(q)
	}
	if len(q) == 0 || len(hay) < len(q) {
		return text, 0
	}
	repl := []rune(replacement)
	var b strings.Builder
	b.Grow(len(text))
	count := 0
	i := 0
	for i < len(hay) {
		if i+len(q) <= len(hay) && runesEqual(hay[i:i+len(q)], q) {
			b.WriteString(string(repl))
			i += len(q)
			count++
			continue
		}
		b.WriteRune(src[i])
		i++
	}
	return b.String(), count
}

// editorGotoRow converts a 1-based line number into a 0-based row. A number
// past the last line clamps to the last line and reports that clamp.
func editorGotoRow(lineCount, line1 int) (row int, clamped bool, ok bool) {
	if line1 < 1 || lineCount < 1 {
		return 0, false, false
	}
	row = line1 - 1
	if row >= lineCount {
		return lineCount - 1, true, true
	}
	return row, false, true
}

func lowerRunes(r []rune) []rune {
	out := make([]rune, len(r))
	for i, ch := range r {
		out[i] = unicode.ToLower(ch)
	}
	return out
}

func runesEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func indexRunesFrom(hay, q []rune, from int) int {
	if from < 0 {
		from = 0
	}
	last := len(hay) - len(q)
	for i := from; i <= last; i++ {
		if runesEqual(hay[i:i+len(q)], q) {
			return i
		}
	}
	return -1
}

func lastIndexRunesBefore(hay, q []rune, limit int) int {
	last := len(hay) - len(q)
	if limit-1 < last {
		last = limit - 1
	}
	for i := last; i >= 0; i-- {
		if runesEqual(hay[i:i+len(q)], q) {
			return i
		}
	}
	return -1
}

func (s *devKitState) buildEditorFindBar() fyne.CanvasObject {
	s.findEntry = widget.NewEntry()
	s.findEntry.SetPlaceHolder("Find")
	s.findEntry.OnSubmitted = func(string) { s.editorFind(true) }

	s.replaceEntry = widget.NewEntry()
	s.replaceEntry.SetPlaceHolder("Replace")
	s.replaceEntry.OnSubmitted = func(string) { s.editorReplace() }

	s.gotoEntry = widget.NewEntry()
	s.gotoEntry.SetPlaceHolder("Line")
	s.gotoEntry.OnSubmitted = func(string) { s.editorGotoLine() }

	s.findMatchCase = widget.NewCheck("Match case", nil)

	nextBtn := widget.NewButton("Next", func() { s.editorFind(true) })
	prevBtn := widget.NewButton("Prev", func() { s.editorFind(false) })
	replaceBtn := widget.NewButton("Replace", func() { s.editorReplace() })
	allBtn := widget.NewButton("All", func() { s.editorReplaceAll() })
	goBtn := widget.NewButton("Go", func() { s.editorGotoLine() })
	closeBtn := widget.NewButton("Close", func() { s.hideEditorFind() })
	for _, b := range []*widget.Button{nextBtn, prevBtn, replaceBtn, allBtn, goBtn, closeBtn} {
		b.Importance = widget.MediumImportance
	}

	bar := container.NewVBox(
		container.NewBorder(nil, nil, nil, container.NewHBox(nextBtn, prevBtn, s.findMatchCase), s.findEntry),
		container.NewBorder(nil, nil, nil, container.NewHBox(replaceBtn, allBtn, container.NewGridWrap(fyne.NewSize(72, 32), s.gotoEntry), goBtn, closeBtn), s.replaceEntry),
	)
	bar.Hide()
	s.findBar = bar
	return bar
}

func (s *devKitState) hideEditorFind() {
	if s.findBar != nil {
		s.findBar.Hide()
	}
	if s.editorPane != nil {
		s.editorPane.Refresh()
	}
	if s.window != nil && s.sourceEditor != nil {
		s.window.Canvas().Focus(s.sourceEditor)
	}
}

func (s *devKitState) showEditorFind(replace bool) {
	s.revealEditor()
	if s.findBar != nil {
		s.findBar.Show()
	}
	if s.editorPane != nil {
		s.editorPane.Refresh()
	}
	if s.sourceEditor != nil && s.findEntry != nil {
		sel := s.sourceEditor.SelectedText()
		if sel != "" && !strings.Contains(sel, "\n") {
			s.findEntry.SetText(sel)
		}
	}
	if s.window == nil {
		return
	}
	if replace && s.replaceEntry != nil {
		s.window.Canvas().Focus(s.replaceEntry)
		return
	}
	if s.findEntry != nil {
		s.window.Canvas().Focus(s.findEntry)
	}
}

func (s *devKitState) showEditorGoto() {
	s.revealEditor()
	if s.findBar != nil {
		s.findBar.Show()
	}
	if s.editorPane != nil {
		s.editorPane.Refresh()
	}
	if s.window != nil && s.gotoEntry != nil {
		s.window.Canvas().Focus(s.gotoEntry)
	}
}

func (s *devKitState) revealEditor() {
	if s.currentView == viewModeEmulatorOnly {
		s.setViewMode(viewModeFull)
	}
	if s.workbenchTabs != nil {
		s.workbenchTabs.SelectIndex(0)
	}
}

func (s *devKitState) editorFind(forward bool) bool {
	if s.sourceEditor == nil || s.findEntry == nil {
		return false
	}
	s.revealEditor()
	query := s.findEntry.Text
	if query == "" {
		s.setStatus("Find text is empty")
		return false
	}
	from := s.sourceEditor.model.Caret.Offset
	if s.sourceEditor.model.HasSelection() {
		start, end := s.sourceEditor.model.Sel.Range()
		if spansEqualQuery(s.sourceEditor.SelectedText(), query, s.matchCase()) {
			if forward {
				from = end
			} else {
				from = start
			}
		} else if forward {
			from = start
		}
	}
	start, end, wrapped, ok := findEditorSpan(s.sourceEditor.Text(), query, from, forward, s.matchCase())
	if !ok {
		s.setStatus("Not found")
		return false
	}
	s.sourceEditor.selectSpan(start, end)
	row, _ := s.sourceEditor.model.OffsetToLineCol(start)
	if wrapped {
		s.setStatus(fmt.Sprintf("Search wrapped — line %d", row+1))
		return true
	}
	s.setStatus(fmt.Sprintf("Line %d", row+1))
	return true
}

func (s *devKitState) editorReplace() {
	if s.sourceEditor == nil || s.findEntry == nil {
		return
	}
	query := s.findEntry.Text
	if query == "" {
		s.setStatus("Find text is empty")
		return
	}
	repl := ""
	if s.replaceEntry != nil {
		repl = s.replaceEntry.Text
	}
	if s.sourceEditor.model.HasSelection() && spansEqualQuery(s.sourceEditor.SelectedText(), query, s.matchCase()) {
		s.sourceEditor.model.DeleteSelection()
		s.sourceEditor.model.InsertText(repl)
		s.sourceEditor.notifyChanged()
		s.sourceEditor.scheduleRefresh()
		if !s.editorFind(true) {
			s.setStatus("Replaced 1")
		}
		return
	}
	s.editorFind(true)
}

func (s *devKitState) editorReplaceAll() {
	if s.sourceEditor == nil || s.findEntry == nil {
		return
	}
	query := s.findEntry.Text
	if query == "" {
		s.setStatus("Find text is empty")
		return
	}
	repl := ""
	if s.replaceEntry != nil {
		repl = s.replaceEntry.Text
	}
	next, count := replaceAllEditorMatches(s.sourceEditor.Text(), query, repl, s.matchCase())
	if count == 0 {
		s.setStatus("Not found")
		return
	}
	s.sourceEditor.SetText(next)
	s.setStatus(fmt.Sprintf("Replaced %d", count))
}

func (s *devKitState) editorGotoLine() {
	if s.sourceEditor == nil || s.gotoEntry == nil {
		return
	}
	s.revealEditor()
	n, err := strconv.Atoi(strings.TrimSpace(s.gotoEntry.Text))
	if err != nil {
		s.setStatus("Line number is not valid")
		return
	}
	row, clamped, ok := editorGotoRow(s.sourceEditor.model.LineCount(), n)
	if !ok {
		s.setStatus("Line number is not valid")
		return
	}
	s.sourceEditor.SetCursor(row, 0)
	s.sourceEditor.revealLine(row)
	if clamped {
		s.setStatus(fmt.Sprintf("Line %d — moved to line %d", n, row+1))
		return
	}
	s.setStatus(fmt.Sprintf("Line %d", row+1))
}

func (s *devKitState) matchCase() bool {
	return s.findMatchCase != nil && s.findMatchCase.Checked
}

func spansEqualQuery(selected, query string, matchCase bool) bool {
	if matchCase {
		return selected == query
	}
	return strings.EqualFold(selected, query)
}

func (e *coreLXCodeEditor) selectSpan(start, end int) {
	if e == nil || e.model == nil {
		return
	}
	if end < start {
		start, end = end, start
	}
	e.model.SetCaretOffset(start, false)
	e.model.SetCaretOffset(end, true)
	row, _ := e.model.OffsetToLineCol(start)
	e.revealLine(row)
	e.scheduleRefresh()
}

func (e *coreLXCodeEditor) revealLine(row int) {
	if e == nil || e.scroll == nil {
		return
	}
	lineHeight, _, _ := e.metrics()
	e.scroll.ScrollToOffset(fyne.NewPos(0, float32(row)*lineHeight))
}
