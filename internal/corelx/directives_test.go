package corelx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDirectivesParsed verifies `--! corelx <version>` and
// `--! modules: name, name, ...` at the top of the file are recognized,
// recorded on the parsed Program (charter D1), and resolve through the real
// module loader (using placeholder module names, decoupled from whatever
// real modules — anim, sfx — end up shipping).
func TestDirectivesParsed(t *testing.T) {
	dir := t.TempDir()
	modulesDir := filepath.Join(dir, "modules")
	if err := os.MkdirAll(modulesDir, 0755); err != nil {
		t.Fatalf("mkdir modules dir: %v", err)
	}
	for _, name := range []string{"modone", "modtwo"} {
		if err := os.WriteFile(filepath.Join(modulesDir, name+".corelx"), []byte("function noop()\n    return\n"), 0644); err != nil {
			t.Fatalf("write module %s: %v", name, err)
		}
	}

	source := `--! corelx 1.0
--! modules: modone, modtwo

function Start()
    while true
        wait_vblank()
`
	srcPath := filepath.Join(dir, "main.corelx")
	if err := os.WriteFile(srcPath, []byte(source), 0644); err != nil {
		t.Fatalf("write main source: %v", err)
	}
	result, err := CompileProject(srcPath, &CompileOptions{OutputPath: filepath.Join(dir, "main.rom")})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	if result.Program.CoreLXVersion != "1.0" {
		t.Errorf("CoreLXVersion: want %q, got %q", "1.0", result.Program.CoreLXVersion)
	}
	want := []string{"modone", "modtwo"}
	if len(result.Program.Modules) != len(want) {
		t.Fatalf("Modules: want %v, got %v", want, result.Program.Modules)
	}
	for i, name := range want {
		if result.Program.Modules[i] != name {
			t.Errorf("Modules[%d]: want %q, got %q", i, name, result.Program.Modules[i])
		}
	}
}

// TestDirectivesOptionalNoDirectives verifies a file with no directives at
// all still compiles (directives are optional).
func TestDirectivesOptionalNoDirectives(t *testing.T) {
	source := `function Start()
    while true
        wait_vblank()
`
	_, result := compileAndBoot(t, source, 600)
	if result.Program.CoreLXVersion != "" {
		t.Errorf("CoreLXVersion: want empty, got %q", result.Program.CoreLXVersion)
	}
	if len(result.Program.Modules) != 0 {
		t.Errorf("Modules: want empty, got %v", result.Program.Modules)
	}
}

// TestUnknownDirectiveRejected verifies an unrecognized directive keyword is
// a compile error, not silently ignored (this reserves the namespace for
// additive growth, per the cartridge format spec).
func TestUnknownDirectiveRejected(t *testing.T) {
	source := `--! bogus_directive
function Start()
    while true
        wait_vblank()
`
	err := compileExpectError(t, source)
	if !strings.Contains(err.Error(), "unknown directive") {
		t.Errorf("expected 'unknown directive' error, got: %v", err)
	}
}

// TestDirectiveAfterCodeRejected verifies directives are only legal at the
// top of the file, before any code (charter D1).
func TestDirectiveAfterCodeRejected(t *testing.T) {
	source := `function Start()
    while true
        wait_vblank()

--! corelx 1.0
`
	err := compileExpectError(t, source)
	if !strings.Contains(err.Error(), "only legal at the top of the file") {
		t.Errorf("expected 'only legal at the top of the file' error, got: %v", err)
	}
}

// TestDirectiveMissingVersionRejected verifies a `corelx` directive with no
// version is a compile error.
func TestDirectiveMissingVersionRejected(t *testing.T) {
	source := `--! corelx
function Start()
    while true
        wait_vblank()
`
	err := compileExpectError(t, source)
	if !strings.Contains(err.Error(), "expected a version") {
		t.Errorf("expected 'expected a version' error, got: %v", err)
	}
}

// TestDirectiveEmptyModulesListRejected verifies a `modules:` directive with
// no names is a compile error.
func TestDirectiveEmptyModulesListRejected(t *testing.T) {
	source := `--! modules:
function Start()
    while true
        wait_vblank()
`
	err := compileExpectError(t, source)
	if !strings.Contains(err.Error(), "expected at least one module name") {
		t.Errorf("expected 'expected at least one module name' error, got: %v", err)
	}
}
