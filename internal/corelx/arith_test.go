package corelx

import "testing"

func readGlobal(t *testing.T, source, name string) uint16 {
	t.Helper()
	emu, result := compileAndBoot(t, source, 8000)
	for _, e := range result.MemoryMap {
		if e.Name == name {
			return read16(emu, e.Address)
		}
	}
	t.Fatalf("memory map missing %s", name)
	return 0
}

func TestSignedDivAndMod(t *testing.T) {
	source := `var q: int = 0
var r: int = 0
function Start()
    q = -7 / 2
    r = -7 % 2
    while true
        wait_vblank()
`
	if got := int16(readGlobal(t, source, "q")); got != -3 {
		t.Fatalf("(-7)/2: want -3, got %d", got)
	}
	if got := int16(readGlobal(t, source, "r")); got != -1 {
		t.Fatalf("(-7)%%2: want -1, got %d", got)
	}

	source = `var q: int = 0
var r: int = 0
function Start()
    q = 7 / -2
    r = 7 % -2
    while true
        wait_vblank()
`
	if got := int16(readGlobal(t, source, "q")); got != -3 {
		t.Fatalf("7/(-2): want -3, got %d", got)
	}
	if got := int16(readGlobal(t, source, "r")); got != 1 {
		t.Fatalf("7%%(-2): want 1, got %d", got)
	}

	source = `var q: int = 0
function Start()
    q = -7 / -2
    while true
        wait_vblank()
`
	if got := int16(readGlobal(t, source, "q")); got != 3 {
		t.Fatalf("(-7)/(-2): want 3, got %d", got)
	}
}

func TestArithmeticRightShift(t *testing.T) {
	source := `var a: int = 0
var b: int = 0
var c: int = 0
function Start()
    a = -8 >> 1
    b = -1 >> 4
    c = 8 >> 1
    while true
        wait_vblank()
`
	if got := int16(readGlobal(t, source, "a")); got != -4 {
		t.Fatalf("-8 >> 1: want -4, got %d", got)
	}
	if got := int16(readGlobal(t, source, "b")); got != -1 {
		t.Fatalf("-1 >> 4: want -1, got %d", got)
	}
	if got := int16(readGlobal(t, source, "c")); got != 4 {
		t.Fatalf("8 >> 1: want 4, got %d", got)
	}
}

func TestConstFoldMatchesRuntimeFixdiv(t *testing.T) {
	cases := []struct {
		name string
		a, b string
	}{
		{"pos_pos", "1.5", "0.5"},
		{"neg_pos", "-1.5", "0.5"},
		{"pos_neg", "1.5", "-0.5"},
		{"neg_neg", "-1.5", "-0.5"},
		{"frac", "1.0", "4.0"},
		{"thirds", "1.0", "3.0"},
		{"mixed", "3.5", "1.5"},
		{"wide", "100.0", "0.25"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			source := `const FOLDED = ` + c.a + ` / ` + c.b + `
var rt_a: fixed = ` + c.a + `
var rt_b: fixed = ` + c.b + `
var folded: fixed = FOLDED
var runtime: fixed = 0.0
function Start()
    runtime = rt_a / rt_b
    while true
        wait_vblank()
`
			emu, result := compileAndBoot(t, source, 8000)
			addrs := map[string]uint16{}
			for _, e := range result.MemoryMap {
				addrs[e.Name] = e.Address
			}
			folded := read16(emu, addrs["folded"])
			runtime := read16(emu, addrs["runtime"])
			if folded != runtime {
				t.Errorf("%s: fold 0x%04X != runtime 0x%04X", c.name, folded, runtime)
			}
		})
	}
}

func TestUnsignedRangeQuotientViaSignedOps(t *testing.T) {
	// 36864 does not fit in a signed int. The overworld's screen-Y term is
	// floor(36864/d), built from 9216 so every intermediate fits.
	dots := []int{321, 400, 1940, 2176, 2816, 3456, 4736, 8000, 8192, 8193, 9216, 9217, 12288, 12289, 20000, 32767}
	for _, d := range dots {
		want := 36864 / d
		source := `var d: int = ` + itoa(d) + `
var q: int = 0
var r: int = 0
var out: int = 0
function Start()
    if d <= 8192
        q = 9216 / d
        r = 9216 % d
        r = r * 4
        r = r / d
        q = q * 4
        out = q + r
    else if d <= 9216
        out = 4
    else if d <= 12288
        out = 3
    else
        r = d - 16384
        if r > 2048
            out = 1
        else
            out = 2
    while true
        wait_vblank()
`
		if got := int16(readGlobal(t, source, "out")); int(got) != want {
			t.Errorf("d=%d: want %d, got %d", d, want, got)
		}
	}
}

func itoa(n int) string {
	return fmtInt(n)
}

func fmtInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [16]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func TestConstFoldMatchesRuntimeDivAndShift(t *testing.T) {
	source := `const FQ = -7 / 2
const FR = -7 % 2
const FS = -8 >> 1
var q: int = FQ
var r: int = FR
var s: int = FS
var rq: int = 0
var rr: int = 0
var rs: int = 0
function Start()
    rq = -7 / 2
    rr = -7 % 2
    rs = -8 >> 1
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 8000)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	for _, pair := range [][2]string{{"q", "rq"}, {"r", "rr"}, {"s", "rs"}} {
		if read16(emu, addrs[pair[0]]) != read16(emu, addrs[pair[1]]) {
			t.Errorf("%s fold 0x%04X != runtime 0x%04X", pair[0], read16(emu, addrs[pair[0]]), read16(emu, addrs[pair[1]]))
		}
	}
}
