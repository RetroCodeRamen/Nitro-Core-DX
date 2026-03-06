package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nitro-core-dx/internal/corelx"
)

func TestUpsertProjectAssetManifestRecord(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "main.corelx")
	if err := os.WriteFile(src, []byte("function Start()\n    wait_vblank()\n"), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	rec := corelx.ProjectAssetRecord{
		Name:     "Ship",
		Type:     "tiles16",
		Encoding: "hex",
		Data:     "00 11 22 33",
	}
	manifestPath, state, err := upsertProjectAssetManifestRecord(src, rec)
	if err != nil {
		t.Fatalf("upsert first: %v", err)
	}
	if state != "new" {
		t.Fatalf("expected new state, got %q", state)
	}
	if filepath.Base(manifestPath) != devKitProjectAssetManifestName {
		t.Fatalf("unexpected manifest path: %s", manifestPath)
	}

	rec.Data = "AA BB"
	_, state, err = upsertProjectAssetManifestRecord(src, rec)
	if err != nil {
		t.Fatalf("upsert second: %v", err)
	}
	if state != "updated" {
		t.Fatalf("expected updated state, got %q", state)
	}

	m, err := loadOrInitProjectAssetManifest(manifestPath)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if len(m.Assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(m.Assets))
	}
	if strings.TrimSpace(m.Assets[0].Data) != "AA BB" {
		t.Fatalf("unexpected data: %q", m.Assets[0].Data)
	}
}

func TestProjectAssetManifestPathForSourceRequiresPath(t *testing.T) {
	if _, err := projectAssetManifestPathForSource(""); err == nil {
		t.Fatalf("expected error for empty source path")
	}
}

func TestBytesToHexFields(t *testing.T) {
	got := bytesToHexFields([]byte{0x00, 0x1A, 0xFF})
	if got != "00 1A FF" {
		t.Fatalf("unexpected hex fields: %q", got)
	}
}
