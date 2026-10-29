package corelx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nitro-core-dx/internal/emulator"
)

// fillerStatements is enough `pad = pad + 1` lines to push one function past
// a single 32KB ROM bank. The exact word count per statement varies with
// codegen, so the tests also reject a ROM that still fits in one bank.
const fillerStatements = 3500

func largeFunctionSource(body string) string {
	var b strings.Builder
	b.WriteString("var pad: int = 0\nvar sentinel: int = 0\nvar n: int = 0\n")
	b.WriteString(body)
	return b.String()
}

func fillerLines(indent, n int) string {
	pad := strings.Repeat(" ", indent)
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "%spad = pad + 1\n", pad)
	}
	return b.String()
}

func compileLarge(t *testing.T, source string) (*emulator.Emulator, *CompileResult) {
	t.Helper()
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "large.corelx")
	romPath := filepath.Join(dir, "large.rom")
	if err := os.WriteFile(srcPath, []byte(source), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	result, err := CompileProject(srcPath, &CompileOptions{OutputPath: romPath})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if len(result.ROMBytes) <= 32+32768 {
		t.Fatalf("expected a function large enough to continue into a second ROM bank, got %d ROM bytes", len(result.ROMBytes))
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(result.ROMBytes); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	return emu, result
}

func globalAddr(t *testing.T, result *CompileResult, name string) uint16 {
	t.Helper()
	for _, e := range result.MemoryMap {
		if e.Name == name {
			return e.Address
		}
	}
	t.Fatalf("memory map missing %s", name)
	return 0
}

func runForSentinel(t *testing.T, emu *emulator.Emulator, addr, want uint16, frames int) {
	t.Helper()
	emu.Start()
	var got uint16
	for i := 0; i < frames; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		got = read16(emu, addr)
		if got == want {
			return
		}
	}
	t.Fatalf("sentinel: want %d, got %d after %d frames", want, got, frames)
}

// TestLargeFunctionCrossBankIfSkipped proves a forward branch can leave the
// bank a function started in. The taken path is the empty else, so the
// filler is never executed; it only exists to push the else clause out of
// bank 1.
func TestLargeFunctionCrossBankIfSkipped(t *testing.T) {
	body := "function Start()\n    if sentinel == 1\n" +
		fillerLines(8, fillerStatements) +
		"        sentinel = 42\n    else\n        sentinel = 7\n    while true\n        wait_vblank()\n"
	emu, result := compileLarge(t, largeFunctionSource(body))
	runForSentinel(t, emu, globalAddr(t, result, "sentinel"), 7, 5)
}

// TestLargeFunctionCrossBankBreak proves a break inside a loop can target
// an exit that lives in a later bank than the break site.
func TestLargeFunctionCrossBankBreak(t *testing.T) {
	body := "function Start()\n    while sentinel == 0\n        break\n" +
		fillerLines(8, fillerStatements) +
		"    sentinel = 4\n    while true\n        wait_vblank()\n"
	emu, result := compileLarge(t, largeFunctionSource(body))
	runForSentinel(t, emu, globalAddr(t, result, "sentinel"), 4, 5)
}

// TestLargeFunctionCrossBankLoop runs a loop body that itself crosses a
// bank, then takes the back edge once and exits. A bad continuation or a
// bad back edge leaves sentinel at 0.
func TestLargeFunctionCrossBankLoop(t *testing.T) {
	body := "function Start()\n    while n < 1\n" +
		fillerLines(8, fillerStatements) +
		"        n = n + 1\n    sentinel = n\n    while true\n        wait_vblank()\n"
	emu, result := compileLarge(t, largeFunctionSource(body))
	runForSentinel(t, emu, globalAddr(t, result, "sentinel"), 1, 40)
}
