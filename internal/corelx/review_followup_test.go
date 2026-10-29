package corelx

import (
	"fmt"
	"strings"
	"testing"
)

func mustCompile(t *testing.T, source string) *CompileResult {
	t.Helper()
	res, err := CompileSource(source, "t.corelx", nil)
	if err != nil {
		t.Fatalf("compile: %v\ndiagnostics: %+v", err, res.Diagnostics)
	}
	return res
}

func mustFailContaining(t *testing.T, source, want string) []Diagnostic {
	t.Helper()
	res, err := CompileSource(source, "t.corelx", nil)
	if err == nil {
		t.Fatalf("expected compile error containing %q", want)
	}
	var msgs []string
	for _, d := range res.Diagnostics {
		msgs = append(msgs, d.Message)
		if strings.Contains(d.Message, want) {
			return res.Diagnostics
		}
	}
	t.Fatalf("diagnostics missing %q: %s", want, strings.Join(msgs, "\n"))
	return nil
}

func TestU8GlobalsKeepTheirOwnByte(t *testing.T) {
	source := `var a: u8 = 1
var b: u8 = 2
var r: int = 0
function Start()
    r = a
    a = 9
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 8000)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	if got := int16(read16(emu, addrs["r"])); got != 1 {
		t.Fatalf("r = a: want 1, got %d", got)
	}
	if got := emu.CPU.Mem.Read8(0, addrs["a"]); got != 9 {
		t.Fatalf("a: want 9, got %d", got)
	}
	if got := emu.CPU.Mem.Read8(0, addrs["b"]); got != 2 {
		t.Fatalf("b: want 2, got %d", got)
	}
}

func TestGlobalAndLocalInitializersHonorDeclaredType(t *testing.T) {
	mustFailContaining(t, `var x: fixed = 64
function Start()
    while true
        wait_vblank()
`, "cannot initialize fixed")
	mustFailContaining(t, `var x: int = 1.5
function Start()
    while true
        wait_vblank()
`, "cannot initialize int")

	source := `var r: int = 0
function Start()
    speed: fixed = 0.0
    speed = speed + 0.5
    r = int(speed)
    while true
        wait_vblank()
`
	if got := int16(readGlobal(t, source, "r")); got != 0 {
		t.Fatalf("int(0.5): want 0, got %d", got)
	}
	mustFailContaining(t, `function Start()
    speed: fixed = 0
    while true
        wait_vblank()
`, "cannot initialize fixed")
}

func TestCallAndReturnChecks(t *testing.T) {
	diags := mustFailContaining(t, `function add(a: int, b: int) -> int
    return a + b
function Start()
    add(1)
    while true
        wait_vblank()
`, "expects 2 arguments")
	if diags[0].Line == 0 {
		t.Fatalf("arity error has no line: %+v", diags[0])
	}

	mustFailContaining(t, `type Player = struct
    hp: int
function dmg(p: Player)
    p.hp = 1
function Start()
    dmg(5)
    while true
        wait_vblank()
`, "expects Player")

	mustFailContaining(t, `function f() -> int
    return 1.5
function Start()
    while true
        wait_vblank()
`, "expects int")

	mustFailContaining(t, `function f()
    return 1
function Start()
    while true
        wait_vblank()
`, "does not return a value")

	mustFailContaining(t, `function f() -> int
    x := 1
function Start()
    while true
        wait_vblank()
`, "must return int")

	both := mustFailContaining(t, `function add(a: int, b: int) -> int
    return a + b
function Start()
    add(1)
    add(1, 2, 3)
    while true
        wait_vblank()
`, "expects 2 arguments")
	n := 0
	for _, d := range both {
		if strings.Contains(d.Message, "expects") {
			n++
		}
	}
	if n < 2 {
		t.Fatalf("want both arity errors, got %+v", both)
	}
}

func TestDiagnosticLineIsTheStatement(t *testing.T) {
	res, err := CompileSource(`function Start()
    x := 1
    x := 3
    while true
        wait_vblank()
`, "t.corelx", nil)
	if err == nil {
		t.Fatal("expected duplicate variable")
	}
	for _, d := range res.Diagnostics {
		if strings.Contains(d.Message, "already defined") {
			if d.Line != 3 {
				t.Fatalf("duplicate x reported at line %d, want 3", d.Line)
			}
			return
		}
	}
	t.Fatalf("missing duplicate diagnostic: %+v", res.Diagnostics)
}

func TestCodegenErrorCarriesLine(t *testing.T) {
	res, err := CompileSource(`function Start()
    break
    while true
        wait_vblank()
`, "t.corelx", nil)
	if err == nil {
		t.Fatal("expected break error")
	}
	for _, d := range res.Diagnostics {
		if strings.Contains(d.Message, "break") {
			if d.Line != 2 {
				t.Fatalf("break error at line %d, want 2: %+v", d.Line, d)
			}
			return
		}
	}
	t.Fatalf("missing break diagnostic: %+v", res.Diagnostics)
}

func TestNumericLiteralRules(t *testing.T) {
	source := `var r: int = 0
function Start()
    r = 010 + 0b101
    while true
        wait_vblank()
`
	if got := int16(readGlobal(t, source, "r")); got != 15 {
		t.Fatalf("010 + 0b101: want 15, got %d", got)
	}
	mustFailContaining(t, `var r: int = 70000
function Start()
    while true
        wait_vblank()
`, "does not fit")
	mustFailContaining(t, `var r: fixed = 200.0
function Start()
    while true
        wait_vblank()
`, "does not fit")
}

func TestConstArrayLengthAndColumnZeroComment(t *testing.T) {
	source := `const N = 4
var tbl: int[N] = [1, 2, 3, 4]
var r: int = 0
function Start()
-- this comment sits at column 0
    r = tbl[1]

    while true
        wait_vblank()
`
	if got := int16(readGlobal(t, source, "r")); got != 2 {
		t.Fatalf("tbl[1]: want 2, got %d", got)
	}
}

func TestCRLFBlankLineDoesNotCloseBlock(t *testing.T) {
	source := "function Start()\r\n    x := 1\r\n\r\n    x = x + 1\r\n    while true\r\n        wait_vblank()\r\n"
	mustCompile(t, source)
}

func TestUnknownBuiltinAndBareNamespace(t *testing.T) {
	mustFailContaining(t, `function Start()
    bg.enabel(0)
    while true
        wait_vblank()
`, "unknown function bg.enabel")
	mustFailContaining(t, `function Start()
    x := bg
    while true
        wait_vblank()
`, "namespace, not a value")
}

func TestManySmallFunctionsFit(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 40; i++ {
		b.WriteString("function f")
		b.WriteString(fmt.Sprint(i))
		b.WriteString("()\n    x := 1\n")
	}
	b.WriteString("function Start()\n    while true\n        wait_vblank()\n")
	mustCompile(t, b.String())
}

func TestOneHundredTwentyFunctionsFit(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 120; i++ {
		b.WriteString("function f")
		b.WriteString(fmt.Sprint(i))
		b.WriteString("()\n    x := 1\n")
	}
	b.WriteString("function Start()\n    while true\n        wait_vblank()\n")
	mustCompile(t, b.String())
}

func TestSpillSlotsSurviveASpillingCallee(t *testing.T) {
	source := `var a: int = 0
var b: int = 0
var c: int = 0
function sq(n: int) -> int
    return n * n
function inner(n: int) -> int
    return n + sq(n + 1)
function spills(n: int) -> int
    return n + (n * 2 + 1)
function add3(p: int, q: int, r: int) -> int
    return p + q + r
function Start()
    x := 100
    a = x + inner(2)
    b = add3(5, spills(1), 7)
    c = sq(x + inner(2)) - sq(x)
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 20000)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	if got := int16(read16(emu, addrs["a"])); got != 111 {
		t.Fatalf("x + inner(2): want 111, got %d", got)
	}
	if got := int16(read16(emu, addrs["b"])); got != 16 {
		t.Fatalf("add3(5, spills(1), 7): want 16, got %d", got)
	}
	if got := int16(read16(emu, addrs["c"])); got != 2321 {
		t.Fatalf("sq(x + inner(2)) - sq(x): want 2321, got %d", got)
	}
}

func TestInferredLocalKeepsFixedType(t *testing.T) {
	source := `var speed: fixed = 1.5
var tbl: fixed[2]
var r0: int = 0
var r1: int = 0
var r2: int = 0
var r3: int = 0
var r4: int = 0
var r5: int = 0
function half(v: fixed) -> fixed
    return v * 0.5
function Start()
    tbl[1] = 1.0
    x := speed
    x = x + 0.5
    r0 = int(x)
    y := -1.5
    y = y + 0.5
    r1 = int(y)
    z := speed * 2.0
    z = z + 0.5
    r2 = int(z)
    w := fixed(3)
    w = w + 0.5
    r3 = int(w)
    h := half(speed)
    h = h + 0.5
    r4 = int(h)
    t := tbl[1]
    t = t + 0.5
    r5 = int(t)
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 20000)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	want := []struct {
		name string
		val  int16
	}{
		{"r0", 2},
		{"r1", -1},
		{"r2", 3},
		{"r3", 3},
		{"r4", 1},
		{"r5", 1},
	}
	for _, tc := range want {
		if got := int16(read16(emu, addrs[tc.name])); got != tc.val {
			t.Errorf("%s: want %d, got %d", tc.name, tc.val, got)
		}
	}
}

func TestButtonNamesExplainTheCollision(t *testing.T) {
	for _, src := range []string{
		"const A = 1\nfunction Start()\n    while true\n        wait_vblank()\n",
		"var X: int = 0\nfunction Start()\n    while true\n        wait_vblank()\n",
		"function Start()\n    Z := 1\n    while true\n        wait_vblank()\n",
	} {
		mustFailContaining(t, src, "controller button")
	}
}

func TestRecursionIsRejected(t *testing.T) {
	mustFailContaining(t, `function fact(n: int) -> int
    if n <= 1
        return 1
    return n * fact(n - 1)
function Start()
    while true
        wait_vblank()
`, "recursion is not supported")
	mustFailContaining(t, `function a(n: int) -> int
    return b(n)
function b(n: int) -> int
    return a(n)
function Start()
    while true
        wait_vblank()
`, "a -> b -> a")
}
