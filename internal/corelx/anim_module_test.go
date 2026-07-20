package corelx

import (
	"os"
	"path/filepath"
	"testing"

	"nitro-core-dx/internal/emulator"
)

// realAnimModuleSource returns the actual shipped anim module source
// (modules/anim.corelx at the repo root), so these tests validate the real
// module rather than a duplicate copy that could drift out of sync.
func realAnimModuleSource(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "modules", "anim.corelx"))
	if err != nil {
		t.Fatalf("read modules/anim.corelx: %v", err)
	}
	return string(data)
}

// compileAndRunFramesWithModule mirrors compileAndBootWithModule but drives
// the emulator by real frames (emu.RunFrame(), which advances frame_counter())
// rather than raw CPU instruction stepping, since anim.frame_index depends on
// frame_counter().
func compileAndRunFramesWithModule(t *testing.T, moduleName, moduleSource, mainSource string, frames int) (*emulator.Emulator, *CompileResult) {
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
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	emu.Start()
	for i := 0; i < frames; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame %d: %v", i, err)
		}
	}
	return emu, result
}

// TestAnimFrameIndexAdvancesAndWraps verifies anim.frame_index advances one
// step every ticks_per_frame calls to wait_vblank() and wraps back to 0
// after frame_count steps, using the real shipped modules/anim.corelx.
func TestAnimFrameIndexAdvancesAndWraps(t *testing.T) {
	mainSource := `--! modules: anim

var observed: int = 0

function Start()
    while true
        wait_vblank()
        observed = anim.frame_index(4, 1)
`
	// After 4 frames with ticks_per_frame=1: idx = 4 % 4 = 0 (wrapped).
	emu, result := compileAndRunFramesWithModule(t, "anim", realAnimModuleSource(t), mainSource, 4)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	if got := read16(emu, addrs["observed"]); got != 0 {
		t.Errorf("frame_index after 4 frames (frame_count=4, ticks=1): want 0, got %d", got)
	}

	// After 6 frames: idx = 6 % 4 = 2.
	emu2, result2 := compileAndRunFramesWithModule(t, "anim", realAnimModuleSource(t), mainSource, 6)
	addrs2 := map[string]uint16{}
	for _, e := range result2.MemoryMap {
		addrs2[e.Name] = e.Address
	}
	if got := read16(emu2, addrs2["observed"]); got != 2 {
		t.Errorf("frame_index after 6 frames (frame_count=4, ticks=1): want 2, got %d", got)
	}
}

// TestAnimFrameIndexRespectsTicksPerFrame verifies frame_index only advances
// once every ticks_per_frame calls, not every call.
func TestAnimFrameIndexRespectsTicksPerFrame(t *testing.T) {
	mainSource := `--! modules: anim

var observed: int = 0

function Start()
    while true
        wait_vblank()
        observed = anim.frame_index(3, 2)
`
	// After 6 frames with ticks_per_frame=2: idx = (6/2) % 3 = 3 % 3 = 0.
	emu, result := compileAndRunFramesWithModule(t, "anim", realAnimModuleSource(t), mainSource, 6)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	if got := read16(emu, addrs["observed"]); got != 0 {
		t.Errorf("frame_index after 6 frames (frame_count=3, ticks=2): want 0, got %d", got)
	}
}

// TestAnimSetMirror verifies anim.set_mirror sets/clears exactly the
// horizontal-flip bit (SPR_HFLIP, 0x10) without disturbing other attr bits.
func TestAnimSetMirror(t *testing.T) {
	mainSource := `--! modules: anim

var observed_mirrored: int = 0
var observed_unmirrored: int = 0

function Start()
    hero := Sprite()
    hero.attr = SPR_PAL(3)
    anim.set_mirror(hero, 1)
    observed_mirrored = hero.attr
    anim.set_mirror(hero, 0)
    observed_unmirrored = hero.attr
    while true
        wait_vblank()
`
	emu, result := compileAndBootWithModule(t, "anim", realAnimModuleSource(t), mainSource, 600)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	// SPR_PAL(3) = 3; mirrored should OR in 0x10 -> 0x13.
	if got := read16(emu, addrs["observed_mirrored"]); got != 0x13 {
		t.Errorf("observed_mirrored: want 0x13 (SPR_PAL(3) | SPR_HFLIP), got 0x%02X", got)
	}
	// Clearing mirror should restore the base palette bits, dropping only 0x10.
	if got := read16(emu, addrs["observed_unmirrored"]); got != 0x03 {
		t.Errorf("observed_unmirrored: want 0x03 (SPR_PAL(3), flip cleared), got 0x%02X", got)
	}
}
