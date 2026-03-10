package corelx

import (
	"encoding/binary"
	"testing"
)

func decodeROMWords(t *testing.T, romBytes []byte) []uint16 {
	t.Helper()
	if len(romBytes) < 32 {
		t.Fatalf("ROM too small: %d bytes", len(romBytes))
	}
	payload := romBytes[32:]
	if len(payload)%2 != 0 {
		t.Fatalf("ROM payload must be word-aligned, got %d bytes", len(payload))
	}
	words := make([]uint16, len(payload)/2)
	for i := range words {
		words[i] = binary.LittleEndian.Uint16(payload[i*2 : i*2+2])
	}
	return words
}

func hasOpcodeMode(words []uint16, opcode, mode uint8) bool {
	for _, w := range words {
		op := uint8((w >> 12) & 0xF)
		md := uint8((w >> 8) & 0xF)
		if op == opcode && md == mode {
			return true
		}
	}
	return false
}

func TestCodegenAmpedVec2UsesIndexedWordMemberOps(t *testing.T) {
	src := `
type Vec2 = struct
    x: i16
    y: i16

function Start()
    pos := Vec2()
    pos.y = 0x1234
    out := pos.y
    apu.enable()
`

	res, err := CompileSource(src, "vec2_indexed.corelx", nil)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
	if res == nil || len(res.ROMBytes) == 0 {
		t.Fatalf("expected compiled ROM bytes")
	}
	words := decodeROMWords(t, res.ROMBytes)

	if !hasOpcodeMode(words, 0x1, 10) {
		t.Fatalf("expected MOV mode 10 in vec2 member store path")
	}
	if !hasOpcodeMode(words, 0x1, 9) {
		t.Fatalf("expected MOV mode 9 in vec2 member load path")
	}
	if hasOpcodeMode(words, 0x1, 7) {
		t.Fatalf("did not expect MOV mode 7 byte member path for Vec2.y")
	}
}

func TestCodegenAmpedSpriteStillUsesByteMemberStore(t *testing.T) {
	src := `
function Start()
    hero := Sprite()
    hero.tile = 7
    apu.enable()
`

	res, err := CompileSource(src, "sprite_byte_store.corelx", nil)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
	if res == nil || len(res.ROMBytes) == 0 {
		t.Fatalf("expected compiled ROM bytes")
	}
	words := decodeROMWords(t, res.ROMBytes)

	if !hasOpcodeMode(words, 0x1, 7) {
		t.Fatalf("expected MOV mode 7 in sprite member byte store path")
	}
}
