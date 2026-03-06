package native

// GapBuffer stores text as runes with a movable gap for efficient local edits.
type GapBuffer struct {
	data     []rune
	gapStart int
	gapEnd   int
}

const defaultGapSize = 256

func NewGapBuffer(text string) *GapBuffer {
	r := []rune(text)
	gap := defaultGapSize
	data := make([]rune, len(r)+gap)
	copy(data, r)
	return &GapBuffer{data: data, gapStart: len(r), gapEnd: len(r) + gap}
}

func (g *GapBuffer) Len() int {
	if g == nil {
		return 0
	}
	return len(g.data) - (g.gapEnd - g.gapStart)
}

func (g *GapBuffer) String() string {
	if g == nil {
		return ""
	}
	out := make([]rune, 0, g.Len())
	out = append(out, g.data[:g.gapStart]...)
	out = append(out, g.data[g.gapEnd:]...)
	return string(out)
}

func (g *GapBuffer) SetText(text string) {
	r := []rune(text)
	gap := defaultGapSize
	g.data = make([]rune, len(r)+gap)
	copy(g.data, r)
	g.gapStart = len(r)
	g.gapEnd = len(r) + gap
}

func (g *GapBuffer) moveGap(offset int) {
	if offset < 0 {
		offset = 0
	}
	if offset > g.Len() {
		offset = g.Len()
	}
	if offset == g.gapStart {
		return
	}
	if offset < g.gapStart {
		shift := g.gapStart - offset
		copy(g.data[g.gapEnd-shift:g.gapEnd], g.data[offset:g.gapStart])
		g.gapStart -= shift
		g.gapEnd -= shift
		return
	}
	shift := offset - g.gapStart
	copy(g.data[g.gapStart:g.gapStart+shift], g.data[g.gapEnd:g.gapEnd+shift])
	g.gapStart += shift
	g.gapEnd += shift
}

func (g *GapBuffer) ensureGap(n int) {
	if n <= (g.gapEnd - g.gapStart) {
		return
	}
	need := n - (g.gapEnd - g.gapStart)
	grow := need + defaultGapSize
	newData := make([]rune, len(g.data)+grow)
	copy(newData, g.data[:g.gapStart])
	newGapEnd := g.gapEnd + grow
	copy(newData[newGapEnd:], g.data[g.gapEnd:])
	g.data = newData
	g.gapEnd = newGapEnd
}

func (g *GapBuffer) Insert(offset int, r []rune) {
	if len(r) == 0 {
		return
	}
	g.moveGap(offset)
	g.ensureGap(len(r))
	copy(g.data[g.gapStart:g.gapStart+len(r)], r)
	g.gapStart += len(r)
}

func (g *GapBuffer) Delete(start, end int) {
	if start < 0 {
		start = 0
	}
	if end > g.Len() {
		end = g.Len()
	}
	if end <= start {
		return
	}
	g.moveGap(start)
	g.gapEnd += end - start
}

func (g *GapBuffer) Slice(start, end int) string {
	if start < 0 {
		start = 0
	}
	if end > g.Len() {
		end = g.Len()
	}
	if end <= start {
		return ""
	}
	if end <= g.gapStart {
		return string(g.data[start:end])
	}
	if start >= g.gapStart {
		shift := g.gapEnd - g.gapStart
		return string(g.data[start+shift : end+shift])
	}
	left := g.data[start:g.gapStart]
	shift := g.gapEnd - g.gapStart
	right := g.data[g.gapEnd : end+shift]
	out := make([]rune, 0, end-start)
	out = append(out, left...)
	out = append(out, right...)
	return string(out)
}
