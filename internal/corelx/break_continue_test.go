package corelx

import (
	"strings"
	"testing"
)

// TestForLoopBreakStopsEarly verifies `break` exits a for loop immediately,
// skipping remaining iterations.
func TestForLoopBreakStopsEarly(t *testing.T) {
	source := `var sum: int = 0
var iters: int = 0
function Start()
    for i = 0 to 9
        if i == 5
            break
        sum = sum + i
        iters = iters + 1
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 3000)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	// i = 0..4 run (5 iterations), i == 5 breaks before the body runs.
	if got := read16(emu, addrs["iters"]); got != 5 {
		t.Errorf("iters: want 5 (break at i==5), got %d", got)
	}
	if got := read16(emu, addrs["sum"]); got != 10 {
		t.Errorf("sum 0..4: want 10, got %d", got)
	}
}

// TestForLoopContinueSkipsIteration verifies `continue` skips the rest of
// the current iteration's body without stopping the loop, and that the loop
// variable still advances (continue must not skip the increment step).
func TestForLoopContinueSkipsIteration(t *testing.T) {
	source := `var sum: int = 0
var iters: int = 0
function Start()
    for i = 0 to 9
        if i == 5
            continue
        sum = sum + i
        iters = iters + 1
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 3000)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	// All 10 iterations (0..9) run; only i==5's body is skipped.
	if got := read16(emu, addrs["iters"]); got != 9 {
		t.Errorf("iters: want 9 (all but i==5), got %d", got)
	}
	// sum(0..9) = 45, minus the skipped 5 = 40.
	if got := read16(emu, addrs["sum"]); got != 40 {
		t.Errorf("sum 0..9 excluding 5: want 40, got %d", got)
	}
}

// TestWhileLoopBreakStopsEarly verifies `break` exits a while loop.
func TestWhileLoopBreakStopsEarly(t *testing.T) {
	source := `var count: int = 0
function Start()
    while true
        count = count + 1
        if count == 5
            break
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 3000)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	if got := read16(emu, addrs["count"]); got != 5 {
		t.Errorf("count: want 5 (break stops incrementing), got %d", got)
	}
}

// TestWhileLoopContinueSkipsRest verifies `continue` jumps back to the
// condition check, skipping the rest of the body for that iteration only.
func TestWhileLoopContinueSkipsRest(t *testing.T) {
	source := `var sum: int = 0
var count: int = 0
function Start()
    while count < 10
        count = count + 1
        if count == 5
            continue
        sum = sum + count
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 3000)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	if got := read16(emu, addrs["count"]); got != 10 {
		t.Errorf("count: want 10 (loop still runs to completion), got %d", got)
	}
	// sum(1..10) = 55, minus the skipped count==5 = 50.
	if got := read16(emu, addrs["sum"]); got != 50 {
		t.Errorf("sum 1..10 excluding 5: want 50, got %d", got)
	}
}

// TestNestedLoopBreakTargetsInnermost verifies break only exits the
// innermost enclosing loop, not any outer loop.
func TestNestedLoopBreakTargetsInnermost(t *testing.T) {
	source := `var outerIters: int = 0
var innerSum: int = 0
function Start()
    for i = 0 to 2
        outerIters = outerIters + 1
        for j = 0 to 9
            if j == 3
                break
            innerSum = innerSum + 1
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 4000)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	// Outer loop runs all 3 iterations (break only affects the inner loop).
	if got := read16(emu, addrs["outerIters"]); got != 3 {
		t.Errorf("outerIters: want 3 (inner break must not exit outer loop), got %d", got)
	}
	// Inner loop breaks at j==3 each outer pass: j=0,1,2 run (3 increments) x 3 outer passes.
	if got := read16(emu, addrs["innerSum"]); got != 9 {
		t.Errorf("innerSum: want 9 (3 inner iterations x 3 outer passes), got %d", got)
	}
}

// TestBreakOutsideLoopRejected verifies break is a compile error outside a loop.
func TestBreakOutsideLoopRejected(t *testing.T) {
	source := `function Start()
    break
    while true
        wait_vblank()
`
	err := compileExpectError(t, source)
	if !strings.Contains(err.Error(), "break used outside of a loop") {
		t.Errorf("expected 'break used outside of a loop' error, got: %v", err)
	}
}

// TestContinueOutsideLoopRejected verifies continue is a compile error
// outside a loop.
func TestContinueOutsideLoopRejected(t *testing.T) {
	source := `function Start()
    continue
    while true
        wait_vblank()
`
	err := compileExpectError(t, source)
	if !strings.Contains(err.Error(), "continue used outside of a loop") {
		t.Errorf("expected 'continue used outside of a loop' error, got: %v", err)
	}
}
