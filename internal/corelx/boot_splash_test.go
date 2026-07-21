package corelx

import (
	"testing"

	"nitro-core-dx/internal/emulator"
)

// TestBootSplashDefaultTimeline verifies the real default boot sequence
// (skipped under go test unless CompileOptions.ForceBootSplash is set --
// see injectBootEntry): the embedded logo slides in and holds for roughly
// three seconds before Start() ever runs.
func TestBootSplashDefaultTimeline(t *testing.T) {
	src := `
var marker: int = 0

function Start()
    marker = 42
    while true
        wait_vblank()
`
	res, err := CompileSource(src, "boot_timeline.corelx", &CompileOptions{ForceBootSplash: true})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	addrs := map[string]uint16{}
	for _, e := range res.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(res.ROMBytes); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	emu.Start()

	// Still mid-splash at frame 100 -- Start() hasn't run yet.
	for i := 0; i < 100; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame: %v", err)
		}
	}
	if m := read16(emu, addrs["marker"]); m != 0 {
		t.Errorf("marker at frame 100 (still sliding/holding): want 0, got %d", m)
	}
	if p := emu.PPU.MatrixPlanes[0]; !p.Enabled {
		t.Errorf("splash plane should be enabled and rendering the logo at frame 100")
	}

	// By frame 250 (slide ~40 frames + hold 150 frames), Start() should have
	// run and the splash's own plane/BG/matrix should be torn down.
	for i := 100; i < 250; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame: %v", err)
		}
	}
	if m := read16(emu, addrs["marker"]); m != 42 {
		t.Errorf("marker at frame 250: want 42 (Start() should have run), got %d", m)
	}
}

// TestBootCustomOverridesDefault verifies a program-defined __Boot() takes
// over the entry point entirely instead of the automatic default splash --
// Generate()/semantic.go's existing entry-point selection (also used by
// test/roms/pellet_game.corelx), not new behavior. An empty-of-splash
// __Boot() that just calls Start() should reach it immediately, with no
// slide/hold delay at all.
func TestBootCustomOverridesDefault(t *testing.T) {
	src := `
var marker: int = 0

function __Boot()
    marker = 1
    Start()

function Start()
    marker = 42
    while true
        wait_vblank()
`
	res, err := CompileSource(src, "boot_custom.corelx", &CompileOptions{ForceBootSplash: true})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	addrs := map[string]uint16{}
	for _, e := range res.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(res.ROMBytes); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	emu.Start()
	for i := 0; i < 3; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame: %v", err)
		}
	}
	if m := read16(emu, addrs["marker"]); m != 42 {
		t.Errorf("marker at frame 3 (custom __Boot(), no splash): want 42, got %d", m)
	}
}

// TestBootCustomCanStillShowDefault verifies a program-defined __Boot() can
// still play the exact stock slide+hold sequence via boot.show_default()
// before doing its own thing and calling Start() -- the override replaces
// automatic invocation, not the sequence's availability.
func TestBootCustomCanStillShowDefault(t *testing.T) {
	src := `
var marker: int = 0

function __Boot()
    boot.show_default()
    marker = 7
    Start()

function Start()
    marker = 42
    while true
        wait_vblank()
`
	res, err := CompileSource(src, "boot_custom_default.corelx", nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	addrs := map[string]uint16{}
	for _, e := range res.MemoryMap {
		addrs[e.Name] = e.Address
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(res.ROMBytes); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	emu.Start()
	for i := 0; i < 100; i++ {
		emu.RunFrame()
	}
	if m := read16(emu, addrs["marker"]); m != 0 {
		t.Errorf("marker at frame 100 (still in boot.show_default()): want 0, got %d", m)
	}
	for i := 100; i < 250; i++ {
		emu.RunFrame()
	}
	if m := read16(emu, addrs["marker"]); m != 42 {
		t.Errorf("marker at frame 250: want 42, got %d", m)
	}
}

// TestBootCanReadInputBeforeStart verifies a program-defined __Boot() can
// read controller state and set ordinary global state before Start() ever
// runs -- plain input.poll()/input.held() work identically at this point in
// the boot sequence as they do anywhere else.
func TestBootCanReadInputBeforeStart(t *testing.T) {
	src := `
var secret_mode: int = 0

function __Boot()
    input.poll()
    if input.held(A)
        secret_mode = 1
    Start()

function Start()
    while true
        wait_vblank()
`
	res, err := CompileSource(src, "boot_input.corelx", nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	addrs := map[string]uint16{}
	for _, e := range res.MemoryMap {
		addrs[e.Name] = e.Address
	}

	held := emulator.NewEmulator()
	held.SetFrameLimit(false)
	if err := held.LoadROM(res.ROMBytes); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	held.SetInputButtons(0x0010) // A
	held.Start()
	for i := 0; i < 3; i++ {
		held.RunFrame()
	}
	if m := read16(held, addrs["secret_mode"]); m != 1 {
		t.Errorf("secret_mode with A held at boot: want 1, got %d", m)
	}

	notHeld := emulator.NewEmulator()
	notHeld.SetFrameLimit(false)
	if err := notHeld.LoadROM(res.ROMBytes); err != nil {
		t.Fatalf("load ROM: %v", err)
	}
	notHeld.Start()
	for i := 0; i < 3; i++ {
		notHeld.RunFrame()
	}
	if m := read16(notHeld, addrs["secret_mode"]); m != 0 {
		t.Errorf("secret_mode with nothing held at boot: want 0, got %d", m)
	}
}
