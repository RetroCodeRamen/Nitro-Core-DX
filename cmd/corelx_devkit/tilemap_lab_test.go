package main

import (
	"strings"
	"testing"
)

func TestTilemapLabPackedBytesOrder(t *testing.T) {
	entries := []uint16{
		0x1201, // tile=0x01 attr=0x12
		0x34AB, // tile=0xAB attr=0x34
	}
	got := tilemapLabPackedBytes(entries)
	want := []byte{0x01, 0x12, 0xAB, 0x34}
	if len(got) != len(want) {
		t.Fatalf("packed length mismatch: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("packed[%d] = 0x%02X want 0x%02X", i, got[i], want[i])
		}
	}
}

func TestTilemapLabAssetRoundTrip(t *testing.T) {
	w, h := 32, 32
	entries := make([]uint16, w*h)
	for i := range entries {
		entries[i] = uint16((i % 256) | ((i % 16) << 8))
	}
	data, err := marshalTilemapLabAsset("Level-1", w, h, entries)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	a, err := unmarshalTilemapLabAsset(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if a.Name != "Level_1" {
		t.Fatalf("expected sanitized name Level_1, got %q", a.Name)
	}
	if a.Width != w || a.Height != h {
		t.Fatalf("unexpected size: %dx%d", a.Width, a.Height)
	}
	if len(a.Entries) != len(entries) {
		t.Fatalf("entries length mismatch: got %d want %d", len(a.Entries), len(entries))
	}
	if a.Entries[17] != entries[17] {
		t.Fatalf("entry mismatch at 17: got 0x%04X want 0x%04X", a.Entries[17], entries[17])
	}
}

func TestTilemapLabCoreLXSnippetShape(t *testing.T) {
	w, h := 8, 8
	entries := make([]uint16, w*h)
	for i := range entries {
		entries[i] = 0x2301
	}
	snippet, err := tilemapLabCoreLXAssetSnippet("MapA", w, h, entries)
	if err != nil {
		t.Fatalf("snippet error: %v", err)
	}
	if !strings.Contains(snippet, "asset MapA: tilemap hex") {
		t.Fatalf("missing tilemap header: %q", snippet)
	}
	lines := strings.Split(snippet, "\n")
	if len(lines) != 10 {
		t.Fatalf("expected 10 lines, got %d", len(lines))
	}
	fields := strings.Fields(lines[2])
	if len(fields) != 16 {
		t.Fatalf("expected 16 bytes per row for 8-wide tilemap row, got %d", len(fields))
	}
	if fields[0] != "01" || fields[1] != "23" {
		t.Fatalf("unexpected first entry bytes: %v", fields[:2])
	}
}

func TestTilemapLabAssetHexData(t *testing.T) {
	entries := []uint16{0x1201, 0x34AB}
	hexData, err := tilemapLabAssetHexData(2, 1, entries)
	if err != nil {
		t.Fatalf("asset hex data: %v", err)
	}
	if hexData != "01 12 AB 34" {
		t.Fatalf("unexpected hex data: %q", hexData)
	}
}

func TestUpsertTilemapLabBlockIntoSource(t *testing.T) {
	src := "function Start()\n    return\n"
	snippet := "-- Tilemap Lab asset (8x8, entry=tile+attr)\nasset MapA: tilemap hex\n    00 00\n"
	next, msg := upsertTilemapLabBlockIntoSource(src, "MapA", snippet)
	if !strings.Contains(next, "asset MapA: tilemap hex") {
		t.Fatalf("missing inserted asset")
	}
	if !strings.Contains(msg, "(new)") {
		t.Fatalf("expected new status msg, got %q", msg)
	}

	repl := "-- Tilemap Lab asset (8x8, entry=tile+attr)\nasset MapA: tilemap hex\n    01 00\n"
	next2, msg2 := upsertTilemapLabBlockIntoSource(next, "MapA", repl)
	if strings.Count(next2, "asset MapA: tilemap hex") != 1 {
		t.Fatalf("expected single asset block after update")
	}
	if !strings.Contains(next2, "01 00") {
		t.Fatalf("expected updated content")
	}
	if !strings.Contains(msg2, "(updated)") {
		t.Fatalf("expected updated status msg, got %q", msg2)
	}
}
