package main

import (
	"image/color"
	"net/url"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type helpHeading struct {
	Level int
	Title string
	Start int
}

type helpSection struct {
	Title string
	Index int
}

func helpHeadings(src string) []helpHeading {
	lines := strings.Split(src, "\n")
	out := make([]helpHeading, 0)
	inCode := false
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") {
			inCode = !inCode
			continue
		}
		if inCode || !strings.HasPrefix(trim, "#") {
			continue
		}
		level := 0
		for level < len(trim) && trim[level] == '#' {
			level++
		}
		if level == 0 || level > 3 || level >= len(trim) || trim[level] != ' ' {
			continue
		}
		title := strings.TrimSpace(trim[level:])
		if title == "" {
			continue
		}
		out = append(out, helpHeading{Level: level, Title: title, Start: i})
	}
	return out
}

func helpSectionList(headings []helpHeading) []helpSection {
	sections := make([]helpSection, 0, len(headings)+1)
	sections = append(sections, helpSection{Title: "Full document", Index: -1})
	for i, heading := range headings {
		indent := ""
		switch heading.Level {
		case 2:
			indent = "  "
		case 3:
			indent = "    "
		}
		sections = append(sections, helpSection{Title: indent + heading.Title, Index: i})
	}
	return sections
}

func helpSectionMarkdown(src string, headings []helpHeading, index int) string {
	if index < 0 || index >= len(headings) {
		return src
	}
	lines := strings.Split(src, "\n")
	start := headings[index].Start
	if start < 0 {
		start = 0
	}
	if start > len(lines) {
		start = len(lines)
	}
	end := len(lines)
	for j := index + 1; j < len(headings); j++ {
		if headings[j].Level <= headings[index].Level {
			end = headings[j].Start
			break
		}
	}
	if end < start {
		end = start
	}
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n")
}

type helpBlock struct {
	Kind   string
	Level  int
	Marker string
	Text   string
}

// parseHelpBlocks splits Markdown into blocks without asking the rich-text
// widget to discover lists. CommonMark will not start an ordered list at a
// number other than 1 when it follows a paragraph, which was gluing the
// manual's table of contents into one horizontal run.
func parseHelpBlocks(src string) []helpBlock {
	lines := strings.Split(src, "\n")
	out := make([]helpBlock, 0, len(lines)/2)
	var para []string
	flush := func() {
		if len(para) == 0 {
			return
		}
		out = append(out, helpBlock{Kind: "p", Text: strings.Join(para, "\n")})
		para = nil
	}
	inCode := false
	var code []string
	var table []string
	flushTable := func() {
		if len(table) == 0 {
			return
		}
		out = append(out, helpBlock{Kind: "table", Text: strings.Join(table, "\n")})
		table = nil
	}
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if inCode {
			if strings.HasPrefix(trim, "```") {
				inCode = false
				out = append(out, helpBlock{Kind: "code", Text: strings.Join(code, "\n")})
				code = nil
				continue
			}
			code = append(code, line)
			continue
		}
		if strings.HasPrefix(trim, "```") {
			flush()
			flushTable()
			inCode = true
			continue
		}
		if strings.HasPrefix(trim, "|") {
			flush()
			table = append(table, line)
			continue
		}
		flushTable()
		if isMarkdownHeading(trim) {
			flush()
			level := 0
			for level < len(trim) && trim[level] == '#' {
				level++
			}
			out = append(out, helpBlock{Kind: "h", Level: level, Text: strings.TrimSpace(trim[level:])})
			continue
		}
		if trim == "---" || trim == "***" {
			flush()
			out = append(out, helpBlock{Kind: "rule"})
			continue
		}
		if strings.HasPrefix(trim, ">") {
			flush()
			out = append(out, helpBlock{Kind: "quote", Text: strings.TrimSpace(strings.TrimPrefix(trim, ">"))})
			continue
		}
		if indent, marker, body, ok := parseHelpListItem(line); ok {
			flush()
			out = append(out, helpBlock{Kind: "li", Level: indent, Marker: marker, Text: body})
			continue
		}
		if trim == "" {
			flush()
			continue
		}
		para = append(para, line)
	}
	flush()
	flushTable()
	if inCode && len(code) > 0 {
		out = append(out, helpBlock{Kind: "code", Text: strings.Join(code, "\n")})
	}
	return out
}

func parseHelpListItem(line string) (indent int, marker, body string, ok bool) {
	spaces := 0
	for spaces < len(line) && line[spaces] == ' ' {
		spaces++
	}
	rest := line[spaces:]
	if rest == "" {
		return 0, "", "", false
	}
	i := 0
	for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
		i++
	}
	if i > 0 && i < len(rest) && (rest[i] == '.' || rest[i] == ')') && i+1 < len(rest) && rest[i+1] == ' ' {
		return spaces / 2, rest[:i+1], strings.TrimSpace(rest[i+2:]), true
	}
	if len(rest) >= 2 && (rest[0] == '-' || rest[0] == '*' || rest[0] == '+') && rest[1] == ' ' {
		return spaces / 2, string(rest[0]), strings.TrimSpace(rest[2:]), true
	}
	return 0, "", "", false
}

func renderHelpDocument(markdown string, onAnchor func(string)) fyne.CanvasObject {
	blocks := parseHelpBlocks(markdown)
	objs := make([]fyne.CanvasObject, 0, len(blocks))
	for _, block := range blocks {
		switch block.Kind {
		case "h":
			objs = append(objs, helpHeadingLabel(block.Level, block.Text))
		case "rule":
			objs = append(objs, widget.NewSeparator())
		case "code", "table":
			objs = append(objs, newBoundedCode(block.Text))
		case "quote":
			objs = append(objs, container.NewPadded(renderHelpInline(block.Text, onAnchor)))
		case "li":
			objs = append(objs, helpListRow(block.Level, block.Marker, block.Text, onAnchor))
		default:
			objs = append(objs, renderHelpInline(block.Text, onAnchor))
		}
	}
	if len(objs) == 0 {
		empty := widget.NewLabel("This section is empty.")
		empty.Wrapping = fyne.TextWrapWord
		objs = append(objs, empty)
	}
	return container.NewPadded(container.NewVBox(objs...))
}

func helpListRow(indent int, marker, body string, onAnchor func(string)) fyne.CanvasObject {
	mark := widget.NewLabel(marker)
	mark.TextStyle = fyne.TextStyle{Monospace: true}
	lead := []fyne.CanvasObject{mark}
	if indent > 0 {
		gap := canvas.NewRectangle(color.Transparent)
		gap.SetMinSize(fyne.NewSize(float32(indent)*14, 1))
		lead = []fyne.CanvasObject{gap, mark}
	}
	return container.NewBorder(nil, nil, container.NewHBox(lead...), nil, renderHelpInline(body, onAnchor))
}

func renderHelpInline(src string, onAnchor func(string)) fyne.CanvasObject {
	if !strings.Contains(src, "](") {
		rt := widget.NewRichTextFromMarkdown(src)
		rt.Wrapping = fyne.TextWrapWord
		return rt
	}
	parts := splitHelpInline(src)
	objs := make([]fyne.CanvasObject, 0, len(parts))
	var buf strings.Builder
	flush := func() {
		if buf.Len() == 0 {
			return
		}
		rt := widget.NewRichTextFromMarkdown(buf.String())
		rt.Wrapping = fyne.TextWrapWord
		objs = append(objs, rt)
		buf.Reset()
	}
	for _, part := range parts {
		if part.url == "" {
			buf.WriteString(part.raw)
			continue
		}
		flush()
		link, err := url.Parse(part.url)
		if err != nil {
			buf.WriteString(part.text)
			continue
		}
		hyper := widget.NewHyperlink(part.text, link)
		hyper.Wrapping = fyne.TextWrapWord
		if strings.HasPrefix(part.url, "#") && onAnchor != nil {
			frag := part.url
			hyper.OnTapped = func() { onAnchor(frag) }
		}
		objs = append(objs, hyper)
	}
	flush()
	if len(objs) == 0 {
		lbl := widget.NewLabel(src)
		lbl.Wrapping = fyne.TextWrapWord
		return lbl
	}
	if len(objs) == 1 {
		return objs[0]
	}
	return container.NewVBox(objs...)
}

type helpInlinePart struct {
	raw  string
	text string
	url  string
}

func splitHelpInline(src string) []helpInlinePart {
	var parts []helpInlinePart
	for len(src) > 0 {
		open := strings.Index(src, "[")
		if open < 0 {
			parts = append(parts, helpInlinePart{raw: src})
			break
		}
		close := strings.Index(src[open:], "](")
		if close < 0 {
			parts = append(parts, helpInlinePart{raw: src})
			break
		}
		close += open
		end := strings.Index(src[close+2:], ")")
		if end < 0 {
			parts = append(parts, helpInlinePart{raw: src})
			break
		}
		end += close + 2
		if open > 0 {
			parts = append(parts, helpInlinePart{raw: src[:open]})
		}
		parts = append(parts, helpInlinePart{
			text: src[open+1 : close],
			url:  src[close+2 : end],
		})
		src = src[end+1:]
	}
	return parts
}

func headingMatchesAnchor(title, anchor string) bool {
	anchor = strings.TrimPrefix(strings.ToLower(anchor), "#")
	slug := helpSlug(title)
	if anchor == "" || slug == "" {
		return false
	}
	return anchor == slug || strings.Contains(anchor, slug)
}

func helpSlug(title string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(title) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func isMarkdownHeading(trim string) bool {
	if !strings.HasPrefix(trim, "#") {
		return false
	}
	level := 0
	for level < len(trim) && trim[level] == '#' {
		level++
	}
	return level > 0 && level <= 3 && level < len(trim) && trim[level] == ' '
}

func helpHeadingLabel(level int, title string) fyne.CanvasObject {
	lbl := widget.NewLabel(title)
	lbl.Wrapping = fyne.TextWrapWord
	lbl.TextStyle = fyne.TextStyle{Bold: true}
	if level == 1 {
		lbl.TextStyle.Monospace = true
	}
	return lbl
}

// boundedCode scrolls a monospace block horizontally without widening the page.
type boundedCode struct {
	widget.BaseWidget
	text  string
	lines int
}

func newBoundedCode(text string) *boundedCode {
	lines := 1
	if text != "" {
		lines = strings.Count(text, "\n") + 1
	}
	b := &boundedCode{text: text, lines: lines}
	b.ExtendBaseWidget(b)
	return b
}

func (b *boundedCode) CreateRenderer() fyne.WidgetRenderer {
	lbl := widget.NewLabel(b.text)
	lbl.TextStyle = fyne.TextStyle{Monospace: true}
	lbl.Wrapping = fyne.TextWrapOff
	scroll := container.NewHScroll(lbl)
	return &boundedCodeRenderer{scroll: scroll, lines: b.lines, objs: []fyne.CanvasObject{scroll}}
}

type boundedCodeRenderer struct {
	scroll fyne.CanvasObject
	lines  int
	objs   []fyne.CanvasObject
}

func (r *boundedCodeRenderer) Layout(size fyne.Size) { r.scroll.Resize(size) }
func (r *boundedCodeRenderer) MinSize() fyne.Size {
	line := theme.TextSize() + 6
	h := float32(r.lines)*line + 8
	if h < line+8 {
		h = line + 8
	}
	return fyne.NewSize(80, h)
}
func (r *boundedCodeRenderer) Refresh()                     {}
func (r *boundedCodeRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *boundedCodeRenderer) Destroy()                     {}
