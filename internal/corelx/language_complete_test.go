package corelx

import (
	"strings"
	"testing"
)

func TestBlockScopeKeepsSiblingNames(t *testing.T) {
	source := `var r: int = 0
function Start()
    if 1
        tmp := 3
        r = tmp
    else
        tmp := 9
        r = tmp
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 8000)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	if got := int16(read16(emu, addrs["r"])); got != 3 {
		t.Fatalf("taken branch: want 3, got %d", got)
	}
}

func TestForVariableDoesNotEscapeTheLoop(t *testing.T) {
	mustFailContaining(t, `function Start()
    for i = 0 to 2
        x := i
    y := i
    while true
        wait_vblank()
`, "undefined identifier: i")
}

func TestSameBlockStillRejectsADuplicate(t *testing.T) {
	mustFailContaining(t, `function Start()
    if 1
        tmp := 1
        tmp := 2
    while true
        wait_vblank()
`, "already defined")
}

func TestCallMaySpanLinesInsideParentheses(t *testing.T) {
	mustCompile(t, `function Start()
    text.draw(
        1,
        2,
        3,
        4,
        5,
        "HI")
    while true
        wait_vblank()
`)
}

func TestParserReportsEveryStatementError(t *testing.T) {
	res, err := CompileSource(`function Start()
    x :=
    y :=
    while true
        wait_vblank()
`, "t.corelx", nil)
	if err == nil {
		t.Fatal("expected parse errors")
	}
	n := 0
	for _, d := range res.Diagnostics {
		if strings.Contains(d.Message, "expected expression") {
			n++
		}
	}
	if n < 2 {
		t.Fatalf("want both statement errors, got %+v", res.Diagnostics)
	}
}

func TestStructGlobalAndArrayAndNestedField(t *testing.T) {
	source := `type Vec = struct
    x: int
    y: int
type Body = struct
    pos: Vec
    hp: int
type Player = struct
    x: int
    y: int
var hero: Player
var enemies: Player[2]
var body: Body
var r0: int = 0
var r1: int = 0
var r2: int = 0
function Start()
    hero.x = 4
    hero.y = 5
    r0 = hero.x + hero.y
    enemies[1].x = 9
    r1 = enemies[1].x
    body.pos.x = 4
    body.hp = 7
    r2 = body.pos.x + body.hp
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 20000)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	if got := int16(read16(emu, addrs["r0"])); got != 9 {
		t.Fatalf("hero.x + hero.y: want 9, got %d", got)
	}
	if got := int16(read16(emu, addrs["r1"])); got != 9 {
		t.Fatalf("enemies[1].x: want 9, got %d", got)
	}
	if got := int16(read16(emu, addrs["r2"])); got != 11 {
		t.Fatalf("body.pos.x + body.hp: want 11, got %d", got)
	}
}

func TestLocalArrayAndArrayParameter(t *testing.T) {
	source := `var r: int = 0
function bump(xs: int[4])
    xs[1] = xs[1] + 5
function Start()
    xs: int[4]
    xs[2] = 7
    xs[1] = 3
    bump(xs)
    r = xs[2] + xs[1]
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 20000)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	if got := int16(read16(emu, addrs["r"])); got != 15 {
		t.Fatalf("xs[2] + bumped xs[1]: want 15, got %d", got)
	}
}

func TestSpriteGlobalStoresItsFields(t *testing.T) {
	source := `var s: Sprite
var r: int = 0
function Start()
    s.tile = 9
    r = s.tile
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 20000)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	if got := int16(read16(emu, addrs["r"])); got != 9 {
		t.Fatalf("s.tile: want 9, got %d", got)
	}
}

func TestPointerTypesAreNotInTheLanguage(t *testing.T) {
	mustFailContaining(t, `var x: *int = 0
function Start()
    while true
        wait_vblank()
`, "Expected type name")
}
