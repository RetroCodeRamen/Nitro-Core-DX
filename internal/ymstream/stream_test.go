package ymstream

import (
	"bytes"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	song := &Song{
		FrameSamples: 735,
		Frames: [][]Write{
			{
				{Port: 0, Addr: 0x20, Data: 0x11},
				{Port: 0, Addr: 0x21, Data: 0x22},
				{Port: 0, Addr: 0x22, Data: 0x33},
			},
			nil,
			{
				{Port: 1, Addr: 0x10, Data: 0x44},
			},
			nil,
			nil,
		},
		WriteCount: 4,
	}

	encoded, err := EncodeSong(song)
	if err != nil {
		t.Fatalf("EncodeSong failed: %v", err)
	}
	if !bytes.Equal(encoded[:8], []byte(Magic)) {
		t.Fatalf("bad magic: %q", encoded[:8])
	}
	decoded, err := DecodeStream(encoded)
	if err != nil {
		t.Fatalf("DecodeStream failed: %v", err)
	}

	if decoded.FrameSamples != song.FrameSamples {
		t.Fatalf("frame samples mismatch: got %d want %d", decoded.FrameSamples, song.FrameSamples)
	}
	if len(decoded.Frames) != len(song.Frames) {
		t.Fatalf("frame count mismatch: got %d want %d", len(decoded.Frames), len(song.Frames))
	}
	for i := range song.Frames {
		got := decoded.Frames[i]
		want := song.Frames[i]
		if len(got) != len(want) {
			t.Fatalf("frame %d write count mismatch: got %d want %d", i, len(got), len(want))
		}
		for j := range want {
			if got[j] != want[j] {
				t.Fatalf("frame %d write %d mismatch: got %+v want %+v", i, j, got[j], want[j])
			}
		}
	}
}

func TestBurstEncodingCompressesSequentialWrites(t *testing.T) {
	song := &Song{
		FrameSamples: 735,
		Frames: [][]Write{
			{
				{Port: 0, Addr: 0x20, Data: 0xAA},
				{Port: 0, Addr: 0x21, Data: 0xBB},
				{Port: 0, Addr: 0x22, Data: 0xCC},
				{Port: 0, Addr: 0x23, Data: 0xDD},
			},
		},
		WriteCount: 4,
	}

	encoded, err := EncodeSong(song)
	if err != nil {
		t.Fatalf("EncodeSong failed: %v", err)
	}

	// Header(16) + burst(1+1+1+4) + wait8(2) + end(1) = 26 bytes.
	if got, want := len(encoded), 26; got != want {
		t.Fatalf("encoded size mismatch: got %d want %d", got, want)
	}
}

func TestDecodeRejectsInvalidMagic(t *testing.T) {
	if _, err := DecodeStream([]byte("bad")); err == nil {
		t.Fatal("DecodeStream should reject invalid magic")
	}
}
