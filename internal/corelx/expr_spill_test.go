package corelx

import "testing"

func TestNestedBinaryDoesNotClobberLeft(t *testing.T) {
	source := `var i: int = 2
var x: int = 5
var r1: int = 0
var r2: int = 0
var r3: int = 0
var r4: int = 0
var r5: int = 0
var r6: int = 0
function Start()
    r1 = 40 + i * 12
    r2 = 10 + (i + 1)
    r3 = 100 - i * 3
    r4 = 10 - (0 - x)
    r5 = (1 + 2) + (3 + 4)
    r6 = 1 + (5 ^ 1)
    while true
        wait_vblank()
`
	if got := int16(readGlobal(t, source, "r1")); got != 64 {
		t.Fatalf("40 + i*12: want 64, got %d", got)
	}
	if got := int16(readGlobal(t, source, "r2")); got != 13 {
		t.Fatalf("10 + (i+1): want 13, got %d", got)
	}
	if got := int16(readGlobal(t, source, "r3")); got != 94 {
		t.Fatalf("100 - i*3: want 94, got %d", got)
	}
	if got := int16(readGlobal(t, source, "r4")); got != 15 {
		t.Fatalf("10 - (0 - x): want 15, got %d", got)
	}
	if got := int16(readGlobal(t, source, "r5")); got != 10 {
		t.Fatalf("(1+2)+(3+4): want 10, got %d", got)
	}
	if got := int16(readGlobal(t, source, "r6")); got != 5 {
		t.Fatalf("1 + (5 ^ 1): want 5, got %d", got)
	}
}

func TestAndOrWithCompoundRight(t *testing.T) {
	source := `var x: int = 0
var y: int = 5
var a: int = 0
var b: int = 0
function Start()
    if x > 0 and y > 0
        a = 1
    if y > 0 and x > 0
        b = 1
    while true
        wait_vblank()
`
	if got := readGlobal(t, source, "a"); got != 0 {
		t.Fatalf("x > 0 and y > 0: want 0, got %d", got)
	}
	if got := readGlobal(t, source, "b"); got != 0 {
		t.Fatalf("y > 0 and x > 0: want 0, got %d", got)
	}
}

func TestCallAsOperandAndArgument(t *testing.T) {
	source := `var r1: int = 0
var r2: int = 0
var r3: int = 0
function double(n: int) -> int
    return n + n
function one() -> int
    return 7
function sub(a: int, b: int) -> int
    return a - b
function three(a: int, b: int, c: int) -> int
    return a * 100 + b * 10 + c
function Start()
    r1 = 100 + double(3)
    r2 = sub(50, one())
    r3 = three(1, 2, 1 + 2)
    while true
        wait_vblank()
`
	if got := int16(readGlobal(t, source, "r1")); got != 106 {
		t.Fatalf("100 + double(3): want 106, got %d", got)
	}
	if got := int16(readGlobal(t, source, "r2")); got != 43 {
		t.Fatalf("sub(50, one()): want 43, got %d", got)
	}
	if got := int16(readGlobal(t, source, "r3")); got != 123 {
		t.Fatalf("three(1, 2, 1+2): want 123, got %d", got)
	}
}

func TestIndexMultiplyUsesDistinctScratch(t *testing.T) {
	source := `var tbl: int[8] = [0, 0, 0, 0, 0, 0, 99, 0]
var i: int = 2
var r: int = 0
function Start()
    r = tbl[i * 3]
    while true
        wait_vblank()
`
	if got := int16(readGlobal(t, source, "r")); got != 99 {
		t.Fatalf("tbl[i*3]: want 99, got %d", got)
	}
}

func TestFieldPlusScaledField(t *testing.T) {
	source := `type P = struct
    x: int
    y: int
var r: int = 0
function Start()
    p := P()
    p.x = 3
    p.y = 4
    r = p.x + p.y * 2
    while true
        wait_vblank()
`
	if got := int16(readGlobal(t, source, "r")); got != 11 {
		t.Fatalf("p.x + p.y*2: want 11, got %d", got)
	}
}
