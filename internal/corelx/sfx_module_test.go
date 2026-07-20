package corelx

import (
	"os"
	"path/filepath"
	"testing"

	"nitro-core-dx/internal/apu"
)

// realSfxModuleSource returns the actual shipped sfx module source
// (modules/sfx.corelx at the repo root), so these tests validate the real
// module rather than a duplicate copy that could drift out of sync.
func realSfxModuleSource(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "modules", "sfx.corelx"))
	if err != nil {
		t.Fatalf("read modules/sfx.corelx: %v", err)
	}
	return string(data)
}

// TestSfxPlayKeysOnChannel verifies sfx.play(channel) writes the YM2608
// key-on register (0x28) with all 4 operator bits set for the given channel,
// via the same host-interface path ym.write uses (port 0).
func TestSfxPlayKeysOnChannel(t *testing.T) {
	mainSource := `--! modules: sfx

function Start()
    sfx.play(2)
    while true
        wait_vblank()
`
	emu, _ := compileAndBootWithModule(t, "sfx", realSfxModuleSource(t), mainSource, 600)
	if got := emu.APU.FM.Addr; got != 0x28 {
		t.Errorf("key-on address-select latch: want 0x28, got 0x%02X", got)
	}
	// channel 2 | 0xF0 (all operators) = 0xF2.
	if got := emu.APU.FM.Read8(apu.FMRegData); got != 0xF2 {
		t.Errorf("key-on data for channel 2: want 0xF2, got 0x%02X", got)
	}
}

// TestSfxStopKeysOffChannel verifies sfx.stop(channel) writes the key-on
// register with no operator bits set (key-off) for the given channel.
func TestSfxStopKeysOffChannel(t *testing.T) {
	mainSource := `--! modules: sfx

function Start()
    sfx.play(2)
    sfx.stop(2)
    while true
        wait_vblank()
`
	emu, _ := compileAndBootWithModule(t, "sfx", realSfxModuleSource(t), mainSource, 600)
	if got := emu.APU.FM.Addr; got != 0x28 {
		t.Errorf("key-off address-select latch: want 0x28, got 0x%02X", got)
	}
	// channel 2, no operator bits set.
	if got := emu.APU.FM.Read8(apu.FMRegData); got != 0x02 {
		t.Errorf("key-off data for channel 2: want 0x02, got 0x%02X", got)
	}
}
