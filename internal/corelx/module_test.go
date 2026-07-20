package corelx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nitro-core-dx/internal/emulator"
)

// compileAndBootWithModule writes a single module file into a "modules"
// directory next to the main source (the default module search path — see
// resolveModulesDir), then compiles and boots the main source exactly like
// compileAndBoot.
func compileAndBootWithModule(t *testing.T, moduleName, moduleSource, mainSource string, maxSteps int) (*emulator.Emulator, *CompileResult) {
	t.Helper()
	dir := t.TempDir()
	modulesDir := filepath.Join(dir, "modules")
	if err := os.MkdirAll(modulesDir, 0755); err != nil {
		t.Fatalf("mkdir modules dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modulesDir, moduleName+".corelx"), []byte(moduleSource), 0644); err != nil {
		t.Fatalf("write module source: %v", err)
	}

	srcPath := filepath.Join(dir, "main.corelx")
	romPath := filepath.Join(dir, "main.rom")
	if err := os.WriteFile(srcPath, []byte(mainSource), 0644); err != nil {
		t.Fatalf("write main source: %v", err)
	}
	result, err := CompileProject(srcPath, &CompileOptions{OutputPath: romPath})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	romData, err := os.ReadFile(romPath)
	if err != nil {
		t.Fatalf("read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	for i := 0; i < maxSteps; i++ {
		if err := emu.CPU.ExecuteInstruction(); err != nil {
			t.Fatalf("CPU step %d: %v", i, err)
		}
	}
	return emu, result
}

// TestModuleFunctionCallable verifies a `--! modules:` request resolves a
// plain .corelx file from the modules directory, and its functions are
// callable with the same dotted-namespace convention as builtins
// (walker.update(...)).
func TestModuleFunctionCallable(t *testing.T) {
	moduleSource := `function update(amount: int) -> int
    return amount + 1
`
	mainSource := `--! modules: walker

var observed: int = 0

function Start()
    observed = walker.update(5)
    while true
        wait_vblank()
`
	emu, result := compileAndBootWithModule(t, "walker", moduleSource, mainSource, 600)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	if got := read16(emu, addrs["observed"]); got != 6 {
		t.Errorf("observed: want 6 (walker.update(5)), got %d", got)
	}
}

// TestModuleGlobalsAndConstsMerge verifies a module's own const/global
// declarations are merged and usable by its functions, and that state
// persists across multiple calls into the module (the module's global is
// real WRAM, not per-call scratch).
func TestModuleGlobalsAndConstsMerge(t *testing.T) {
	moduleSource := `const STEP = 10
var total: int = 0

function bump() -> int
    total = total + STEP
    return total
`
	mainSource := `--! modules: counter

var observed: int = 0

function Start()
    observed = counter.bump()
    observed = counter.bump()
    while true
        wait_vblank()
`
	emu, result := compileAndBootWithModule(t, "counter", moduleSource, mainSource, 600)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	if got := read16(emu, addrs["observed"]); got != 20 {
		t.Errorf("observed: want 20 (two bumps of STEP=10), got %d", got)
	}
}

// TestModuleNotInstalledRejected verifies requesting a module with no
// corresponding file in the modules directory is a compile error, matching
// the design record's "module `X` not installed" wording (as opposed to a
// generic "unknown function" error at the call site).
func TestModuleNotInstalledRejected(t *testing.T) {
	source := `--! modules: nonexistent_module
function Start()
    while true
        wait_vblank()
`
	err := compileExpectError(t, source)
	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("expected 'not installed' error, got: %v", err)
	}
}

// TestModuleParseErrorReported verifies a syntax error inside a module file
// is reported (mentioning the module name), not silently ignored.
func TestModuleParseErrorReported(t *testing.T) {
	moduleSource := `function broken(
`
	mainSource := `--! modules: broken_module
function Start()
    while true
        wait_vblank()
`
	dir := t.TempDir()
	modulesDir := filepath.Join(dir, "modules")
	if err := os.MkdirAll(modulesDir, 0755); err != nil {
		t.Fatalf("mkdir modules dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modulesDir, "broken_module.corelx"), []byte(moduleSource), 0644); err != nil {
		t.Fatalf("write module source: %v", err)
	}
	srcPath := filepath.Join(dir, "main.corelx")
	if err := os.WriteFile(srcPath, []byte(mainSource), 0644); err != nil {
		t.Fatalf("write main source: %v", err)
	}
	_, err := CompileProject(srcPath, &CompileOptions{OutputPath: filepath.Join(dir, "main.rom")})
	if err == nil {
		t.Fatal("expected compile error, got success")
	}
	if !strings.Contains(err.Error(), "broken_module") {
		t.Errorf("expected error mentioning module name 'broken_module', got: %v", err)
	}
}
