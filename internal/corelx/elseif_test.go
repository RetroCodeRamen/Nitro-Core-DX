package corelx

import (
	"fmt"
	"testing"
)

// elseIfProgram is a chain whose conditions overlap on purpose. n > 10 is
// also n > 0, so a second branch that runs unconditionally (the old `else if`
// misparse) or a taken branch that falls into else (the old elseif codegen)
// both produce the wrong got.
func elseIfProgram(keyword string, n int) string {
	return fmt.Sprintf(`var n: int = %d
var got: int = 0
function Start()
    if n > 30
        got = 3
    %s n > 20
        got = 2
    %s n > 10
        got = 1
    else
        got = 0
    while true
        wait_vblank()
`, n, keyword, keyword)
}

func TestElseIfChain(t *testing.T) {
	cases := []struct {
		keyword string
		n       int
		want    uint16
	}{
		{"else if", 40, 3},
		{"else if", 25, 2},
		{"else if", 15, 1},
		{"else if", 0, 0},
		{"elseif", 40, 3},
		{"elseif", 25, 2},
		{"elseif", 15, 1},
		{"elseif", 0, 0},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s_n%d", tc.keyword, tc.n), func(t *testing.T) {
			emu, result := compileAndBoot(t, elseIfProgram(tc.keyword, tc.n), 4000)
			var addr uint16
			found := false
			for _, e := range result.MemoryMap {
				if e.Name == "got" {
					addr = e.Address
					found = true
				}
			}
			if !found {
				t.Fatal("memory map missing got")
			}
			if got := read16(emu, addr); got != tc.want {
				t.Fatalf("got: want %d, got %d", tc.want, got)
			}
		})
	}
}

// TestNestedElseIfStillWorks keeps the indented else/if form, which is a
// real else body containing another if, not a chain clause.
func TestNestedElseIfStillWorks(t *testing.T) {
	source := `var n: int = 25
var got: int = 0
function Start()
    if n > 30
        got = 3
    else
        if n > 20
            got = 2
        else
            got = 0
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 4000)
	var addr uint16
	for _, e := range result.MemoryMap {
		if e.Name == "got" {
			addr = e.Address
		}
	}
	if got := read16(emu, addr); got != 2 {
		t.Fatalf("nested else/if: want 2, got %d", got)
	}
}
