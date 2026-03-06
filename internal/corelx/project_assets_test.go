package corelx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileSourceLoadsExternalAssetManifest(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.corelx")
	src := `
function Start()
    t := gfx.load_tiles(ASSET_Ship, 16)
    if t == 0
        apu.enable()
    else
        apu.enable()
`
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	assetHex := filepath.Join(dir, "ship.hex")
	if err := os.WriteFile(assetHex, []byte("00 11 22 33"), 0644); err != nil {
		t.Fatalf("write asset file: %v", err)
	}

	manifest := `{
  "format_version": 1,
  "assets": [
    { "name": "Ship", "type": "tiles8", "encoding": "hex", "path": "ship.hex" }
  ]
}`
	manifestPath := filepath.Join(dir, defaultProjectAssetManifest)
	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	res, err := CompileProject(srcPath, nil)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
	if len(res.NormalizedAssets) != 1 {
		t.Fatalf("expected 1 normalized asset, got %d", len(res.NormalizedAssets))
	}
	if res.NormalizedAssets[0].Name != "Ship" {
		t.Fatalf("expected Ship asset, got %q", res.NormalizedAssets[0].Name)
	}
	if res.Manifest == nil || len(res.Manifest.SourceFiles) < 2 {
		t.Fatalf("expected source + manifest files in manifest.SourceFiles, got %+v", res.Manifest)
	}
}

func TestCompileSourceExternalAssetDuplicateDiagnostic(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.corelx")
	src := `
asset Ship: tiles8 hex
    00 11 22 33

function Start()
    apu.enable()
`
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	manifest := `{
  "assets": [
    { "name": "Ship", "type": "tiles8", "encoding": "hex", "data": "00 00" }
  ]
}`
	manifestPath := filepath.Join(dir, defaultProjectAssetManifest)
	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	res, err := CompileProject(srcPath, nil)
	if err == nil {
		t.Fatalf("expected duplicate asset error")
	}
	if res == nil {
		t.Fatalf("expected result with diagnostics")
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "E_ASSET_DUPLICATE" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected E_ASSET_DUPLICATE diagnostics, got %+v", res.Diagnostics)
	}
}

func TestLoadProjectAssetsBadRecord(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.corelx")
	if err := os.WriteFile(srcPath, []byte("function Start()\n    apu.enable()\n"), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	manifestPath := filepath.Join(dir, defaultProjectAssetManifest)
	if err := os.WriteFile(manifestPath, []byte(`{"assets":[{"name":"A","type":"","encoding":"hex","data":"00"}]}`), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	_, _, diags := loadProjectAssets(srcPath, CompileOptions{})
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics for bad manifest record")
	}
	if !strings.HasPrefix(diags[0].Code, "E_ASSET_MANIFEST_") {
		t.Fatalf("unexpected diagnostic code: %s", diags[0].Code)
	}
}
