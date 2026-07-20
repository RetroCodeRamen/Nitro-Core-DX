package corelx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A user-defined struct beyond the two compiler-intrinsic/legacy shapes
// (Sprite, Vec2) must work end to end: instantiate, assign every field
// (including non-zero byte offsets), and read them back correctly. Before
// struct codegen was generalized, only "Sprite" and "Vec2" were recognized by
// name in three separate hardcoded maps, so a third struct type like this one
// could not even compile.
func TestUserDefinedStructBeyondSpriteVec2(t *testing.T) {
	source := `type Player = struct
    x: fixed
    y: fixed
    lives: int

var observed_x: fixed = 0.0
var observed_y: fixed = 0.0
var observed_lives: int = 0

function Start()
    p := Player()
    p.x = 1.5
    p.y = 2.25
    p.lives = 42
    observed_x = p.x
    observed_y = p.y
    observed_lives = p.lives
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 600)

	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}

	// fixed 8.8: 1.5 -> 0x0180, 2.25 -> 0x0240.
	if got := read16(emu, addrs["observed_x"]); got != 0x0180 {
		t.Errorf("observed_x: want 0x0180 (1.5 fixed), got 0x%04X", got)
	}
	if got := read16(emu, addrs["observed_y"]); got != 0x0240 {
		t.Errorf("observed_y: want 0x0240 (2.25 fixed), got 0x%04X", got)
	}
	if got := read16(emu, addrs["observed_lives"]); got != 42 {
		t.Errorf("observed_lives (3rd field, byte offset 4): want 42, got %d", got)
	}
}

// Assigning to a field that doesn't exist on a struct must be a compile
// error, not a silent no-op (the previous codegen fallback discarded the
// assignment and compiled clean).
func TestStructUnknownFieldAssignmentRejected(t *testing.T) {
	source := `type Player = struct
    x: fixed
    y: fixed
    lives: int

function Start()
    p := Player()
    p.score = 10
    while true
        wait_vblank()
`
	err := compileExpectError(t, source)
	if !strings.Contains(err.Error(), "no field") {
		t.Errorf("expected 'no field' error, got: %v", err)
	}
}

// Reading a field that doesn't exist on a struct must be a compile error, not
// a silent zero (the previous codegen fallback loaded 0 for any unmatched
// member on a recognized struct variable).
func TestStructUnknownFieldReadRejected(t *testing.T) {
	source := `type Player = struct
    x: fixed
    y: fixed
    lives: int

var out: int = 0

function Start()
    p := Player()
    out = p.score
    while true
        wait_vblank()
`
	err := compileExpectError(t, source)
	if !strings.Contains(err.Error(), "no field") {
		t.Errorf("expected 'no field' error, got: %v", err)
	}
}

// Member access on a variable that isn't a struct at all must be a compile
// error.
func TestMemberAccessOnNonStructVariableRejected(t *testing.T) {
	source := `function Start()
    x := 5
    x.member = 1
    while true
        wait_vblank()
`
	err := compileExpectError(t, source)
	if !strings.Contains(err.Error(), "not a struct variable") {
		t.Errorf("expected 'not a struct variable' error, got: %v", err)
	}
}

// TestTopLevelStructDeclarationCompiles verifies the frozen v1 syntax
// spec's top-level `struct Name:` form (D10) — as opposed to the older
// `type Name = struct` form — compiles and round-trips field values
// correctly. This is the exact shape of the spec's own Player exemplar.
func TestTopLevelStructDeclarationCompiles(t *testing.T) {
	source := `struct Player:
    x: fixed
    y: fixed
    lives: int

var observed_lives: int = 0

function Start()
    player := Player()
    player.lives = 3
    observed_lives = player.lives
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 600)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	if got := read16(emu, addrs["observed_lives"]); got != 3 {
		t.Errorf("observed_lives: want 3, got %d", got)
	}
}

// TestTopLevelStructAndLegacyFormCoexist verifies both struct declaration
// forms can be used for different types in the same program.
func TestTopLevelStructAndLegacyFormCoexist(t *testing.T) {
	source := `type Vec2Like = struct
    x: i16
    y: i16

struct Player:
    pos: int
    lives: int

function Start()
    p := Player()
    p.lives = 5
    v := Vec2Like()
    v.x = 10
    while true
        wait_vblank()
`
	if _, _, err := compileOnly(t, source); err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
}

// TestTopLevelStructDuplicateNameRejected verifies declaring the same
// struct name twice (across either or both declaration forms) is a compile
// error, not silently accepted with one definition winning.
func TestTopLevelStructDuplicateNameRejected(t *testing.T) {
	source := `type Player = struct
    x: int

struct Player:
    y: int

function Start()
    while true
        wait_vblank()
`
	err := compileExpectError(t, source)
	if !strings.Contains(err.Error(), "already defined") {
		t.Errorf("expected 'already defined' duplicate-type error, got: %v", err)
	}
}

// compileOnly compiles source and returns the source/output paths and any
// compile error, without asserting success.
func compileOnly(t *testing.T, source string) (string, string, error) {
	t.Helper()
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.corelx")
	outPath := filepath.Join(dir, "main.rom")
	if err := os.WriteFile(srcPath, []byte(source), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	_, err := CompileProject(srcPath, &CompileOptions{OutputPath: outPath})
	return srcPath, outPath, err
}

// compileExpectError compiles source and fails the test if compilation
// succeeds; returns the compile error.
func compileExpectError(t *testing.T, source string) error {
	t.Helper()
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.corelx")
	if err := os.WriteFile(srcPath, []byte(source), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	_, err := CompileProject(srcPath, &CompileOptions{OutputPath: filepath.Join(dir, "main.rom")})
	if err == nil {
		t.Fatal("expected compile error, got success")
	}
	return err
}
