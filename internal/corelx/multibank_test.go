package corelx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nitro-core-dx/internal/emulator"
)

// TestMultiBankCrossBankCallSmoke is the end-to-end proof that multi-bank
// ROM code support actually works, not just compiles: a synthetic program
// with enough padding functions to force pass 1 past the 32KB single-bank
// ceiling, where the last function -- guaranteed by packFunctionBanks'
// greedy in-emission-order packer to land in bank 2+ -- writes a sentinel
// value to WRAM, called from Start() via a far call. If far-call encoding,
// the callPatches bank-explicit patch resolution, or RET's PBR unwind were
// wrong, this would either crash, hang, or resume execution in the wrong
// bank and never reach the sentinel write.
//
// Padding uses a handful of large functions (many statements each) rather
// than many small ones: generateFunction reserves a 256-byte stack window
// per function regardless of body size, and the stack region only has
// budget for roughly two dozen functions total before "stack allocation
// exhausted" -- so the byte count has to come from statement volume within
// a few functions, not function count.
func TestMultiBankCrossBankCallSmoke(t *testing.T) {
	var src strings.Builder
	src.WriteString("var padscratch: int = 0\n")
	src.WriteString("var sentinel: int = 0\n\n")
	src.WriteString("function Start()\n")
	src.WriteString("    write_sentinel()\n")
	src.WriteString("    while true\n")
	src.WriteString("        wait_vblank()\n\n")

	const numPadFuncs = 15
	const stmtsPerFunc = 600
	for f := 0; f < numPadFuncs; f++ {
		fmt.Fprintf(&src, "function pad%d()\n", f)
		for i := 0; i < stmtsPerFunc; i++ {
			fmt.Fprintf(&src, "    padscratch = padscratch + %d\n", i%97+1)
		}
		src.WriteString("\n")
	}

	// Declared last, so packFunctionBanks' in-emission-order packer places
	// it after every padding function -- guaranteed to land in bank 2+
	// given the padding above pushes well past one bank's 16384 words.
	src.WriteString("function write_sentinel()\n")
	src.WriteString("    sentinel = 12345\n")

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "multibank_smoke.corelx")
	romPath := filepath.Join(dir, "multibank_smoke.rom")
	if err := os.WriteFile(srcPath, []byte(src.String()), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	result, err := CompileProject(srcPath, &CompileOptions{OutputPath: romPath})
	if err != nil {
		t.Fatalf("compile: %v (diagnostics: %+v)", err, result.Diagnostics)
	}
	if HasErrors(result.Diagnostics) {
		t.Fatalf("compile produced error diagnostics: %+v", result.Diagnostics)
	}
	if len(result.ROMBytes) <= 32+32768 {
		t.Fatalf("expected a multi-bank ROM (>32800 bytes), got %d bytes -- padding wasn't enough to force pass 1 to overflow", len(result.ROMBytes))
	}

	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(result.ROMBytes); err != nil {
		t.Fatalf("LoadROM failed: %v", err)
	}
	emu.Start()

	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	sentinelAddr, ok := addrs["sentinel"]
	if !ok {
		t.Fatalf("memory map missing 'sentinel' global: %+v", result.MemoryMap)
	}

	for frame := 0; frame < 10; frame++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed at frame %d: %v", frame, err)
		}
	}

	if got := read16(emu, sentinelAddr); got != 12345 {
		t.Fatalf("sentinel: want 12345, got %d -- far call across ROM banks did not execute correctly", got)
	}
}
