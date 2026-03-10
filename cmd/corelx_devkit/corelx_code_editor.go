package main

import (
	"image/color"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nitro-core-dx/internal/corelx"
	nativeed "nitro-core-dx/internal/editor/native"
)

var (
	reAssetDecl      = regexp.MustCompile(`^\s*asset\s+([A-Za-z_][A-Za-z0-9_]*)\s*:`)
	reHexDirective   = regexp.MustCompile(`^\s*hex\s*$`)
	reLoadTiles      = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\s*:=\s*gfx\.load_tiles\(\s*ASSET_([A-Za-z_][A-Za-z0-9_]*)\s*,`)
	reSpriteTileAttr = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\.tile\s*=\s*([A-Za-z_][A-Za-z0-9_]*)`)
	reSpritePalAttr  = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\.attr\s*=.*\bSPR_PAL\(\s*(\d+)\s*\)`)
	reSetPalette     = regexp.MustCompile(`\bgfx\.set_palette\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(0x[0-9A-Fa-f]+|\d+)\s*\)`)
	reSpriteLabBank  = regexp.MustCompile(`^\s*--\s*Sprite Lab palette bank\s+(\d+)\s*$`)
	reFunctionDecl   = regexp.MustCompile(`\bfunction\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	reFunctionCall   = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	reVarAssign      = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\s*:=`)
	reConstSymbol    = regexp.MustCompile(`\b(ASSET_[A-Za-z0-9_]+|SPR_[A-Za-z0-9_]+|SYS_[A-Za-z0-9_]+)\b`)
	reAPINamespace   = regexp.MustCompile(`\b(gfx|ppu|apu|oam|sprite|memory|dma|input|math)\b`)
	reTypeCall       = regexp.MustCompile(`\b([A-Z][A-Za-z0-9_]*)\s*\(`)
)

type coreLXCodeEditor struct {
	widget.BaseWidget

	model  *nativeed.Model
	grid   *widget.TextGrid
	scroll *container.Scroll

	onChanged func(string)

	refreshMu      sync.Mutex
	refreshPending bool

	shiftHeld bool
	dragSel   bool
	popup     *widget.PopUpMenu

	lastRenderedText string
	lastTokenSource  string
	lastTokens       []corelx.Token
}

func newCoreLXCodeEditor() *coreLXCodeEditor {
	grid := widget.NewTextGrid()
	grid.ShowLineNumbers = true
	grid.ShowWhitespace = false
	grid.Scroll = fyne.ScrollNone

	e := &coreLXCodeEditor{
		model: nativeed.NewModel(""),
		grid:  grid,
	}
	e.scroll = container.NewScroll(grid)
	e.scroll.Direction = container.ScrollBoth
	e.ExtendBaseWidget(e)
	e.refreshGrid("")
	return e
}

func (e *coreLXCodeEditor) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.NRGBA{R: 12, G: 14, B: 18, A: 255})
	return widget.NewSimpleRenderer(container.NewMax(bg, e.scroll))
}

// MinSize is intentionally clamped so large buffers/long lines in TextGrid do
// not force oversized window minimum limits, which can disable native maximize
// controls on some window managers.
func (e *coreLXCodeEditor) MinSize() fyne.Size {
	return fyne.NewSize(420, 260)
}

func (e *coreLXCodeEditor) FocusGained() { e.scheduleRefresh() }
func (e *coreLXCodeEditor) FocusLost()   { e.dragSel = false; e.scheduleRefresh() }

func (e *coreLXCodeEditor) Tapped(ev *fyne.PointEvent) {
	e.focusSelf()
	e.dragSel = false
	e.setCaretFromPoint(ev.Position, e.shiftHeld)
}

func (e *coreLXCodeEditor) DoubleTapped(ev *fyne.PointEvent) {
	e.focusSelf()
	e.setCaretFromPoint(ev.Position, false)
	if !e.model.HasSelection() {
		start := e.model.Caret.Offset
		text := []rune(e.model.Text())
		for start > 0 && isWordRune(text[start-1]) {
			start--
		}
		end := e.model.Caret.Offset
		for end < len(text) && isWordRune(text[end]) {
			end++
		}
		e.model.SetCaretOffset(start, false)
		e.model.SetCaretOffset(end, true)
		e.scheduleRefresh()
	}
}

func (e *coreLXCodeEditor) TappedSecondary(ev *fyne.PointEvent) {
	e.focusSelf()
	e.showContextMenu(ev)
}

func (e *coreLXCodeEditor) MouseDown(ev *desktop.MouseEvent) {
	if ev == nil {
		return
	}
	if ev.Button == desktop.MouseButtonPrimary {
		e.focusSelf()
		e.dragSel = true
		e.setCaretFromPoint(ev.Position, e.shiftHeld)
	}
}

func (e *coreLXCodeEditor) MouseUp(ev *desktop.MouseEvent) {
	e.dragSel = false
}

func (e *coreLXCodeEditor) Dragged(ev *fyne.DragEvent) {
	if ev == nil || !e.dragSel {
		return
	}
	e.setCaretFromPoint(ev.Position, true)
}

func (e *coreLXCodeEditor) DragEnd() { e.dragSel = false }

func (e *coreLXCodeEditor) TypedRune(r rune) {
	if r == '\r' {
		return
	}
	e.model.InsertRune(r)
	e.notifyChanged()
	e.scheduleRefresh()
}

func (e *coreLXCodeEditor) TypedKey(ev *fyne.KeyEvent) {
	if ev == nil {
		return
	}
	extend := e.shiftHeld
	switch ev.Name {
	case fyne.KeyReturn, fyne.KeyEnter:
		e.model.InsertRune('\n')
		e.notifyChanged()
	case fyne.KeyTab:
		e.model.InsertRune('\t')
		e.notifyChanged()
	case fyne.KeyBackspace:
		e.model.Backspace()
		e.notifyChanged()
	case fyne.KeyDelete:
		e.model.DeleteForward()
		e.notifyChanged()
	case fyne.KeyLeft:
		e.model.MoveLeft(extend)
	case fyne.KeyRight:
		e.model.MoveRight(extend)
	case fyne.KeyUp:
		e.model.MoveUp(extend)
	case fyne.KeyDown:
		e.model.MoveDown(extend)
	case fyne.KeyHome:
		e.model.MoveLineHome(extend)
	case fyne.KeyEnd:
		e.model.MoveLineEnd(extend)
	}
	e.scheduleRefresh()
}

func (e *coreLXCodeEditor) TypedShortcut(shortcut fyne.Shortcut) {
	switch s := shortcut.(type) {
	case *fyne.ShortcutCopy:
		if s != nil && s.Clipboard != nil {
			s.Clipboard.SetContent(e.model.SelectedText())
		}
	case *fyne.ShortcutCut:
		if s != nil && s.Clipboard != nil {
			s.Clipboard.SetContent(e.model.SelectedText())
		}
		e.model.DeleteSelection()
		e.notifyChanged()
		e.scheduleRefresh()
	case *fyne.ShortcutPaste:
		if s != nil && s.Clipboard != nil {
			e.model.InsertText(s.Clipboard.Content())
			e.notifyChanged()
			e.scheduleRefresh()
		}
	case *fyne.ShortcutSelectAll:
		e.model.SelectAll()
		e.scheduleRefresh()
	}
}

func (e *coreLXCodeEditor) KeyDown(key *fyne.KeyEvent) {
	if key == nil {
		return
	}
	if key.Name == desktop.KeyShiftLeft || key.Name == desktop.KeyShiftRight {
		e.shiftHeld = true
	}
}

func (e *coreLXCodeEditor) KeyUp(key *fyne.KeyEvent) {
	if key == nil {
		return
	}
	if key.Name == desktop.KeyShiftLeft || key.Name == desktop.KeyShiftRight {
		e.shiftHeld = false
	}
}

func (e *coreLXCodeEditor) SetOnChanged(cb func(string)) { e.onChanged = cb }

func (e *coreLXCodeEditor) SetText(text string) {
	e.model.SetText(text)
	e.invalidateTokenCache()
	e.notifyChanged()
	e.scheduleRefresh()
}

func (e *coreLXCodeEditor) Text() string { return e.model.Text() }

func (e *coreLXCodeEditor) Cursor() (row, col int) {
	return e.model.OffsetToLineCol(e.model.Caret.Offset)
}

func (e *coreLXCodeEditor) SetCursor(row, col int) {
	e.model.SetCaretLineCol(row, col, false)
	e.scheduleRefresh()
}

func (e *coreLXCodeEditor) SelectedText() string {
	return e.model.SelectedText()
}

func (e *coreLXCodeEditor) focusSelf() {
	if fyne.CurrentApp() == nil || fyne.CurrentApp().Driver() == nil {
		return
	}
	if c := fyne.CurrentApp().Driver().CanvasForObject(e); c != nil {
		c.Focus(e)
	}
}

func (e *coreLXCodeEditor) notifyChanged() {
	e.invalidateTokenCache()
	if e.onChanged != nil {
		e.onChanged(e.model.Text())
	}
}

func (e *coreLXCodeEditor) invalidateTokenCache() {
	e.lastTokenSource = ""
	e.lastTokens = nil
}

func (e *coreLXCodeEditor) setCaretFromPoint(pos fyne.Position, keepSelection bool) {
	lineHeight, colWidth, gutter := e.metrics()
	x := float32(pos.X) + e.scroll.Offset.X - gutter
	y := float32(pos.Y) + e.scroll.Offset.Y
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	row := int(y / lineHeight)
	col := int(x / colWidth)
	e.model.SetCaretLineCol(row, col, keepSelection)
	e.scheduleRefresh()
}

func (e *coreLXCodeEditor) metrics() (lineHeight float32, colWidth float32, gutter float32) {
	sz := theme.TextSize()
	lineHeight = fyne.MeasureText("Mg", sz, fyne.TextStyle{Monospace: true}).Height
	if lineHeight < 1 {
		lineHeight = 1
	}
	colWidth = fyne.MeasureText("M", sz, fyne.TextStyle{Monospace: true}).Width
	if colWidth < 1 {
		colWidth = 1
	}
	digits := len(strconv.Itoa(maxInt(1, e.model.LineCount())))
	gutter = colWidth * float32(digits+2)
	return lineHeight, colWidth, gutter
}

func (e *coreLXCodeEditor) showContextMenu(ev *fyne.PointEvent) {
	if ev == nil || fyne.CurrentApp() == nil {
		return
	}
	if e.popup != nil {
		e.popup.Hide()
		e.popup = nil
	}
	clipboard := fyne.CurrentApp().Clipboard()
	menu := fyne.NewMenu("",
		fyne.NewMenuItem("Cut", func() {
			clipboard.SetContent(e.model.SelectedText())
			e.model.DeleteSelection()
			e.notifyChanged()
			e.scheduleRefresh()
		}),
		fyne.NewMenuItem("Copy", func() {
			clipboard.SetContent(e.model.SelectedText())
		}),
		fyne.NewMenuItem("Paste", func() {
			e.model.InsertText(clipboard.Content())
			e.notifyChanged()
			e.scheduleRefresh()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Select All", func() {
			e.model.SelectAll()
			e.scheduleRefresh()
		}),
	)
	if c := fyne.CurrentApp().Driver().CanvasForObject(e); c != nil {
		entryPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(e)
		e.popup = widget.NewPopUpMenu(menu, c)
		e.popup.ShowAtPosition(entryPos.Add(ev.Position))
	}
}

func (e *coreLXCodeEditor) refreshGrid(text string) {
	displayText := text
	if displayText == "" {
		displayText = " "
	}
	if displayText != e.lastRenderedText {
		e.grid.SetText(displayText)
		e.lastRenderedText = displayText
	}
	e.applySyntaxHighlight(text)
}

func (e *coreLXCodeEditor) scheduleRefresh() {
	e.refreshMu.Lock()
	if e.refreshPending {
		e.refreshMu.Unlock()
		return
	}
	e.refreshPending = true
	e.refreshMu.Unlock()

	time.AfterFunc(16*time.Millisecond, func() {
		fyne.Do(func() {
			e.refreshMu.Lock()
			e.refreshPending = false
			e.refreshMu.Unlock()
			e.refreshGrid(e.model.Text())
		})
	})
}

func (e *coreLXCodeEditor) applySyntaxHighlight(source string) {
	if len(source) > 200000 {
		e.grid.Refresh()
		return
	}

	lines := strings.Split(source, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}

	activeRow, activeCol := e.model.OffsetToLineCol(e.model.Caret.Offset)
	if activeRow < 0 {
		activeRow = 0
	}
	if activeRow >= len(lines) {
		activeRow = len(lines) - 1
	}

	lineStyle := &widget.CustomTextGridStyle{FGColor: theme.ForegroundColor(), BGColor: color.NRGBA{R: 23, G: 26, B: 33, A: 255}}
	e.grid.SetRowStyle(activeRow, lineStyle)

	tokens := e.lastTokens
	if source != e.lastTokenSource {
		lexer := corelx.NewLexer(source)
		parsed, err := lexer.Tokenize()
		if err == nil {
			tokens = parsed
			e.lastTokens = parsed
			e.lastTokenSource = source
		} else {
			tokens = nil
			e.lastTokens = nil
			e.lastTokenSource = source
		}
	}
	if len(tokens) > 0 {
		for _, tok := range tokens {
			if tok.Type == corelx.TOKEN_EOF || tok.Line <= 0 || tok.Column <= 0 {
				continue
			}
			style := tokenGridStyle(tok.Type, tok.Literal)
			length := utf8.RuneCountInString(tok.Literal)
			if length <= 0 {
				length = utf8.RuneCountInString(tokenFallbackLiteral(tok.Type))
			}
			if length <= 0 {
				continue
			}
			row := tok.Line - 1
			startCol := tok.Column - 1
			if row < 0 || row >= len(lines) {
				continue
			}
			lineLen := utf8.RuneCountInString(lines[row])
			if startCol < 0 || startCol >= lineLen {
				continue
			}
			endCol := startCol + length - 1
			if endCol >= lineLen {
				endCol = lineLen - 1
			}
			if endCol >= startCol {
				e.grid.SetStyleRange(row, startCol, row, endCol, style)
			}
		}
	}

	e.applyLuaStyleSemanticHighlight(lines, tokens, activeRow, activeCol)
	e.applySpritePaletteHighlight(source, lines)
	e.applySelectionHighlight(lines)

	lineLen := utf8.RuneCountInString(lines[activeRow])
	if lineLen > 0 {
		if activeCol >= lineLen {
			activeCol = lineLen - 1
		}
		if activeCol < 0 {
			activeCol = 0
		}
		cursorStyle := &widget.CustomTextGridStyle{FGColor: theme.ForegroundColor(), BGColor: color.NRGBA{R: 60, G: 72, B: 95, A: 255}}
		e.grid.SetStyle(activeRow, activeCol, cursorStyle)
	}

	e.grid.Refresh()
}

func (e *coreLXCodeEditor) applySelectionHighlight(lines []string) {
	if !e.model.HasSelection() {
		return
	}
	start, end := e.model.Sel.Range()
	if end <= start {
		return
	}
	startRow, startCol := e.model.OffsetToLineCol(start)
	endRow, endCol := e.model.OffsetToLineCol(end)
	selStyle := &widget.CustomTextGridStyle{FGColor: theme.ForegroundColor(), BGColor: color.NRGBA{R: 54, G: 72, B: 103, A: 255}}
	for row := startRow; row <= endRow && row < len(lines); row++ {
		lineLen := utf8.RuneCountInString(lines[row])
		if lineLen <= 0 {
			continue
		}
		left := 0
		right := lineLen - 1
		if row == startRow {
			left = startCol
		}
		if row == endRow {
			right = endCol - 1
		}
		if right >= lineLen {
			right = lineLen - 1
		}
		if left < 0 {
			left = 0
		}
		if right >= left {
			e.grid.SetStyleRange(row, left, row, right, selStyle)
		}
	}
}

func isWordRune(r rune) bool {
	if r >= 'a' && r <= 'z' {
		return true
	}
	if r >= 'A' && r <= 'Z' {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	return r == '_'
}

func (e *coreLXCodeEditor) applyLuaStyleSemanticHighlight(lines []string, tokens []corelx.Token, cursorRow, cursorCol int) {
	functionStyle := &widget.CustomTextGridStyle{TextStyle: fyne.TextStyle{Bold: true}, FGColor: color.NRGBA{R: 0x9A, G: 0xD7, B: 0xEA, A: 0xFF}, BGColor: color.Transparent}
	localVarStyle := &widget.CustomTextGridStyle{FGColor: color.NRGBA{R: 0x8D, G: 0xE0, B: 0xC1, A: 0xFF}, BGColor: color.Transparent}
	globalVarStyle := &widget.CustomTextGridStyle{FGColor: color.NRGBA{R: 0x7B, G: 0xC6, B: 0xDE, A: 0xFF}, BGColor: color.Transparent}
	constantStyle := &widget.CustomTextGridStyle{TextStyle: fyne.TextStyle{Bold: true}, FGColor: color.NRGBA{R: 0xF4, G: 0xBE, B: 0x56, A: 0xFF}, BGColor: color.Transparent}
	namespaceStyle := &widget.CustomTextGridStyle{TextStyle: fyne.TextStyle{Bold: true}, FGColor: color.NRGBA{R: 0x6D, G: 0xB6, B: 0xFF, A: 0xFF}, BGColor: color.Transparent}
	typeStyle := &widget.CustomTextGridStyle{FGColor: color.NRGBA{R: 0xCF, G: 0xA8, B: 0xFF, A: 0xFF}, BGColor: color.Transparent}

	for row, line := range lines {
		code := line
		if idx := strings.Index(code, "--"); idx >= 0 {
			code = code[:idx]
		}
		if strings.TrimSpace(code) == "" {
			continue
		}

		applyGroupMatches(e.grid, row, code, reConstSymbol, 1, constantStyle)
		applyAPINamespaceMatches(e.grid, row, code, namespaceStyle)
		applyGroupMatches(e.grid, row, code, reFunctionDecl, 1, functionStyle)
		applyFilteredGroupMatches(e.grid, row, code, reFunctionCall, 1, functionStyle, func(name string) bool {
			return !isKeywordIdent(name) && !isLikelyTypeName(name)
		})
		applyFilteredGroupMatches(e.grid, row, code, reTypeCall, 1, typeStyle, func(name string) bool {
			return !isKeywordIdent(strings.ToLower(name))
		})
	}

	if len(tokens) == 0 {
		return
	}

	functionNames, functionLocals, tokenFunctionCtx := analyzeIdentifierSemantics(tokens)
	cursorSymbol, cursorSymbolFunc := symbolAtCursor(tokens, tokenFunctionCtx, cursorRow, cursorCol)
	cursorSymbolStyle := &widget.CustomTextGridStyle{TextStyle: fyne.TextStyle{Bold: true}, FGColor: color.NRGBA{R: 0xFF, G: 0xF2, B: 0xB3, A: 0xFF}, BGColor: color.NRGBA{R: 0x3A, G: 0x3F, B: 0x22, A: 0xFF}}

	for i, tok := range tokens {
		if tok.Type != corelx.TOKEN_IDENTIFIER || tok.Line <= 0 || tok.Column <= 0 || tok.Literal == "" {
			continue
		}
		if strings.HasPrefix(tok.Literal, "ASSET_") || strings.HasPrefix(tok.Literal, "SPR_") || strings.HasPrefix(tok.Literal, "SYS_") {
			continue
		}
		prevType := tokenTypeAt(tokens, i-1)
		nextType := tokenTypeAt(tokens, i+1)
		if prevType == corelx.TOKEN_DOT || nextType == corelx.TOKEN_DOT {
			continue
		}

		row := tok.Line - 1
		startCol := tok.Column - 1
		endCol := startCol + utf8.RuneCountInString(tok.Literal) - 1
		if endCol < startCol || row < 0 || row >= len(lines) {
			continue
		}

		if functionNames[tok.Literal] && (prevType == corelx.TOKEN_FUNCTION || nextType == corelx.TOKEN_LPAREN) {
			e.grid.SetStyleRange(row, startCol, row, endCol, functionStyle)
		} else {
			fn := tokenFunctionCtx[i]
			if fn != "" && functionLocals[fn][tok.Literal] {
				e.grid.SetStyleRange(row, startCol, row, endCol, localVarStyle)
			} else {
				e.grid.SetStyleRange(row, startCol, row, endCol, globalVarStyle)
			}
		}

		if cursorSymbol != "" && tok.Literal == cursorSymbol {
			if cursorSymbolFunc == "" || tokenFunctionCtx[i] == cursorSymbolFunc {
				e.grid.SetStyleRange(row, startCol, row, endCol, cursorSymbolStyle)
			}
		}
	}
}

func analyzeIdentifierSemantics(tokens []corelx.Token) (map[string]bool, map[string]map[string]bool, map[int]string) {
	functionNames := make(map[string]bool)
	functionLocals := make(map[string]map[string]bool)
	tokenFunctionCtx := make(map[int]string)

	for i := 0; i < len(tokens)-1; i++ {
		if tokens[i].Type == corelx.TOKEN_FUNCTION && tokens[i+1].Type == corelx.TOKEN_IDENTIFIER {
			fn := tokens[i+1].Literal
			functionNames[fn] = true
			if _, ok := functionLocals[fn]; !ok {
				functionLocals[fn] = make(map[string]bool)
			}
			j := i + 2
			if tokenTypeAt(tokens, j) == corelx.TOKEN_LPAREN {
				j++
				for j < len(tokens) && tokenTypeAt(tokens, j) != corelx.TOKEN_RPAREN {
					if tokenTypeAt(tokens, j) == corelx.TOKEN_IDENTIFIER {
						functionLocals[fn][tokens[j].Literal] = true
					}
					j++
				}
			}
		}
	}

	pendingFunc := ""
	currentFunc := ""
	funcStack := make([]string, 0, 8)
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		switch tok.Type {
		case corelx.TOKEN_FUNCTION:
			if tokenTypeAt(tokens, i+1) == corelx.TOKEN_IDENTIFIER {
				pendingFunc = tokens[i+1].Literal
			}
		case corelx.TOKEN_INDENT:
			if pendingFunc != "" {
				currentFunc = pendingFunc
				funcStack = append(funcStack, currentFunc)
				pendingFunc = ""
			} else {
				funcStack = append(funcStack, currentFunc)
			}
		case corelx.TOKEN_DEDENT:
			if len(funcStack) > 0 {
				funcStack = funcStack[:len(funcStack)-1]
			}
			currentFunc = ""
			for j := len(funcStack) - 1; j >= 0; j-- {
				if funcStack[j] != "" {
					currentFunc = funcStack[j]
					break
				}
			}
		}
		tokenFunctionCtx[i] = currentFunc
		if currentFunc != "" && tok.Type == corelx.TOKEN_IDENTIFIER && tokenTypeAt(tokens, i+1) == corelx.TOKEN_ASSIGN {
			if _, ok := functionLocals[currentFunc]; !ok {
				functionLocals[currentFunc] = make(map[string]bool)
			}
			functionLocals[currentFunc][tok.Literal] = true
		}
	}

	return functionNames, functionLocals, tokenFunctionCtx
}

func symbolAtCursor(tokens []corelx.Token, tokenFunctionCtx map[int]string, cursorRow, cursorCol int) (string, string) {
	for i, tok := range tokens {
		if tok.Type != corelx.TOKEN_IDENTIFIER || tok.Line <= 0 || tok.Column <= 0 || tok.Literal == "" {
			continue
		}
		row := tok.Line - 1
		startCol := tok.Column - 1
		endCol := startCol + utf8.RuneCountInString(tok.Literal)
		if row == cursorRow && cursorCol >= startCol && cursorCol < endCol {
			return tok.Literal, tokenFunctionCtx[i]
		}
	}
	return "", ""
}

func tokenTypeAt(tokens []corelx.Token, idx int) corelx.TokenType {
	if idx < 0 || idx >= len(tokens) {
		return corelx.TOKEN_EOF
	}
	return tokens[idx].Type
}

func applyGroupMatches(grid *widget.TextGrid, row int, line string, rx *regexp.Regexp, group int, style widget.TextGridStyle) {
	matches := rx.FindAllStringSubmatchIndex(line, -1)
	for _, m := range matches {
		if group*2+1 >= len(m) {
			continue
		}
		startByte, endByte := m[group*2], m[group*2+1]
		if startByte < 0 || endByte <= startByte {
			continue
		}
		startCol := byteToRuneCol(line, startByte)
		endCol := byteToRuneCol(line, endByte) - 1
		if endCol >= startCol {
			grid.SetStyleRange(row, startCol, row, endCol, style)
		}
	}
}

func applyFilteredGroupMatches(grid *widget.TextGrid, row int, line string, rx *regexp.Regexp, group int, style widget.TextGridStyle, keep func(string) bool) {
	matches := rx.FindAllStringSubmatchIndex(line, -1)
	for _, m := range matches {
		if group*2+1 >= len(m) {
			continue
		}
		startByte, endByte := m[group*2], m[group*2+1]
		if startByte < 0 || endByte <= startByte {
			continue
		}
		name := line[startByte:endByte]
		if keep != nil && !keep(name) {
			continue
		}
		startCol := byteToRuneCol(line, startByte)
		endCol := byteToRuneCol(line, endByte) - 1
		if endCol >= startCol {
			grid.SetStyleRange(row, startCol, row, endCol, style)
		}
	}
}

func applyAPINamespaceMatches(grid *widget.TextGrid, row int, line string, style widget.TextGridStyle) {
	matches := reAPINamespace.FindAllStringSubmatchIndex(line, -1)
	for _, m := range matches {
		if len(m) < 4 {
			continue
		}
		startByte, endByte := m[2], m[3]
		if startByte < 0 || endByte <= startByte {
			continue
		}
		if endByte >= len(line) || line[endByte] != '.' {
			continue
		}
		startCol := byteToRuneCol(line, startByte)
		endCol := byteToRuneCol(line, endByte) - 1
		if endCol >= startCol {
			grid.SetStyleRange(row, startCol, row, endCol, style)
		}
	}
}

func byteToRuneCol(s string, byteIndex int) int {
	if byteIndex <= 0 {
		return 0
	}
	if byteIndex > len(s) {
		byteIndex = len(s)
	}
	return utf8.RuneCountInString(s[:byteIndex])
}

func isKeywordIdent(name string) bool {
	switch strings.ToLower(name) {
	case "function", "if", "elseif", "else", "while", "for", "return", "type", "struct", "asset", "true", "false", "and", "or", "not":
		return true
	default:
		return false
	}
}

func isLikelyTypeName(name string) bool {
	if name == "" {
		return false
	}
	first := rune(name[0])
	return first >= 'A' && first <= 'Z'
}

func (e *coreLXCodeEditor) applySpritePaletteHighlight(source string, lines []string) {
	paletteColors, paletteSet := parsePaletteAssignments(source)
	assetPalette := inferAssetPaletteBanks(source)
	spriteLabHints := inferSpriteLabAssetPaletteHints(lines)
	for asset, bank := range spriteLabHints {
		if _, exists := assetPalette[asset]; !exists {
			assetPalette[asset] = bank
		}
	}

	currentAsset := ""
	inHexBlock := false
	for row, line := range lines {
		trimmed := strings.TrimSpace(line)

		if m := reAssetDecl.FindStringSubmatch(line); len(m) == 2 {
			currentAsset = m[1]
			inHexBlock = strings.Contains(strings.ToLower(line), " hex")
			continue
		}

		if strings.TrimSpace(line) == "" {
			continue
		}

		if currentAsset != "" && !isIndentedLine(line) && !strings.HasPrefix(trimmed, "--") {
			currentAsset = ""
			inHexBlock = false
		}
		if currentAsset == "" {
			continue
		}

		if !inHexBlock && reHexDirective.MatchString(line) {
			inHexBlock = true
			continue
		}
		if !inHexBlock {
			continue
		}
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		if !looksLikeHexDataLine(trimmed) {
			continue
		}

		bank := 0
		if b, ok := assetPalette[currentAsset]; ok {
			bank = b
		}
		if bank < 0 || bank >= spriteLabPaletteBanks {
			bank = 0
		}

		content := line
		if cmt := strings.Index(content, "--"); cmt >= 0 {
			content = content[:cmt]
		}
		runes := []rune(content)
		for col, r := range runes {
			nib, ok := hexNibbleValue(r)
			if !ok {
				continue
			}
			fg := theme.ForegroundColor()
			if nib == 0 && !paletteSet[bank][nib] {
				fg = theme.DisabledColor()
			} else if paletteSet[bank][nib] {
				fg = paletteColors[bank][nib]
			}
			e.grid.SetStyle(row, col, &widget.CustomTextGridStyle{FGColor: fg, BGColor: color.Transparent})
		}
	}
}

func parsePaletteAssignments(source string) ([spriteLabPaletteBanks][spriteLabColorsPerBank]color.NRGBA, [spriteLabPaletteBanks][spriteLabColorsPerBank]bool) {
	var colors [spriteLabPaletteBanks][spriteLabColorsPerBank]color.NRGBA
	var set [spriteLabPaletteBanks][spriteLabColorsPerBank]bool
	matches := reSetPalette.FindAllStringSubmatch(source, -1)
	for _, m := range matches {
		if len(m) != 4 {
			continue
		}
		bank, errBank := strconv.Atoi(m[1])
		idx, errIdx := strconv.Atoi(m[2])
		val, errVal := strconv.ParseUint(m[3], 0, 16)
		if errBank != nil || errIdx != nil || errVal != nil {
			continue
		}
		if bank < 0 || bank >= spriteLabPaletteBanks || idx < 0 || idx >= spriteLabColorsPerBank {
			continue
		}
		colors[bank][idx] = rgb555ToNRGBA(uint16(val))
		set[bank][idx] = true
	}
	return colors, set
}

func inferAssetPaletteBanks(source string) map[string]int {
	tileVarToAsset := make(map[string]string)
	for _, m := range reLoadTiles.FindAllStringSubmatch(source, -1) {
		if len(m) == 3 {
			tileVarToAsset[m[1]] = m[2]
		}
	}

	spriteVarToTileVar := make(map[string]string)
	for _, m := range reSpriteTileAttr.FindAllStringSubmatch(source, -1) {
		if len(m) == 3 {
			spriteVarToTileVar[m[1]] = m[2]
		}
	}

	spriteVarToBank := make(map[string]int)
	for _, m := range reSpritePalAttr.FindAllStringSubmatch(source, -1) {
		if len(m) != 3 {
			continue
		}
		bank, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		spriteVarToBank[m[1]] = bank
	}

	assetToBank := make(map[string]int)
	for spriteVar, bank := range spriteVarToBank {
		tileVar, ok := spriteVarToTileVar[spriteVar]
		if !ok {
			continue
		}
		assetName, ok := tileVarToAsset[tileVar]
		if !ok {
			continue
		}
		assetToBank[assetName] = bank
	}
	return assetToBank
}

func inferSpriteLabAssetPaletteHints(lines []string) map[string]int {
	assetToBank := make(map[string]int)
	pendingSpriteLabAsset := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-- Sprite Lab asset") {
			pendingSpriteLabAsset = ""
			continue
		}

		if m := reAssetDecl.FindStringSubmatch(line); len(m) == 2 {
			pendingSpriteLabAsset = m[1]
			continue
		}

		if pendingSpriteLabAsset == "" {
			continue
		}

		if m := reSpriteLabBank.FindStringSubmatch(line); len(m) == 2 {
			bank, err := strconv.Atoi(m[1])
			if err == nil && bank >= 0 && bank < spriteLabPaletteBanks {
				assetToBank[pendingSpriteLabAsset] = bank
			}
			pendingSpriteLabAsset = ""
		}
	}

	return assetToBank
}

func isIndentedLine(line string) bool {
	return strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")
}

func looksLikeHexDataLine(line string) bool {
	if line == "" {
		return false
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return false
	}
	hexPairs := 0
	for _, field := range fields {
		if field == "--" || strings.HasPrefix(field, "--") {
			break
		}
		if len(field) != 2 {
			return false
		}
		if _, err := strconv.ParseUint(field, 16, 8); err != nil {
			return false
		}
		hexPairs++
	}
	return hexPairs > 0
}

func hexNibbleValue(r rune) (int, bool) {
	switch {
	case r >= '0' && r <= '9':
		return int(r - '0'), true
	case r >= 'a' && r <= 'f':
		return int(r-'a') + 10, true
	case r >= 'A' && r <= 'F':
		return int(r-'A') + 10, true
	default:
		return 0, false
	}
}

func tokenGridStyle(tt corelx.TokenType, literal string) widget.TextGridStyle {
	fg := theme.ForegroundColor()
	style := fyne.TextStyle{}
	switch {
	case isKeyword(tt):
		fg = theme.PrimaryColor()
	case tt == corelx.TOKEN_STRING:
		fg = color.NRGBA{R: 0xE6, G: 0xA6, B: 0x5E, A: 0xFF}
	case tt == corelx.TOKEN_NUMBER:
		fg = color.NRGBA{R: 0xE3, G: 0xD5, B: 0x8B, A: 0xFF}
	case tt == corelx.TOKEN_COMMENT:
		fg = theme.DisabledColor()
		style.Italic = true
	case tt == corelx.TOKEN_IDENTIFIER && strings.HasPrefix(literal, "SYS_"):
		fg = theme.Color(theme.ColorNameFocus)
		style.Bold = true
	case tt == corelx.TOKEN_IDENTIFIER:
		fg = color.NRGBA{R: 0x8D, G: 0xE0, B: 0xC1, A: 0xFF}
	}
	return &widget.CustomTextGridStyle{TextStyle: style, FGColor: fg, BGColor: color.Transparent}
}
