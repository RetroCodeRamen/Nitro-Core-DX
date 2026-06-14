package corelx

import "testing"

// TestArrayInitializer verifies global arrays can be initialized with a value
// list (data tables), each element landing in its slot at power-on. This is the
// enabler for lookup tables like heading vectors.
func TestArrayInitializer(t *testing.T) {
	source := `var heading_x: int[8] = [0, 181, 256, 181, 0, 0 - 181, 0 - 256, 0 - 181]
var pick: int = 0

function Start()
    pick = heading_x[1] + heading_x[6]
    while true
        wait_vblank()
`
	emu, result := compileAndBoot(t, source, 800)
	addrs := map[string]uint16{}
	for _, e := range result.MemoryMap {
		addrs[e.Name] = e.Address
	}
	base := addrs["heading_x"]
	wantVals := []uint16{0, 181, 256, 181, 0, 0xFF4B, 0xFF00, 0xFF4B} // -181, -256, -181 in two's complement
	for i, want := range wantVals {
		if got := read16(emu, base+uint16(i*2)); got != want {
			t.Errorf("heading_x[%d]: want 0x%04X, got 0x%04X", i, want, got)
		}
	}
	// pick = heading_x[1] + heading_x[6] = 181 + (-256) = -75
	if got := int16(read16(emu, addrs["pick"])); got != -75 {
		t.Errorf("pick = heading_x[1]+heading_x[6]: want -75, got %d", got)
	}
}
