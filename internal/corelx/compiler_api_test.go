package corelx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCompileSourceStructuredDiagnosticsMissingStart(t *testing.T) {
	src := `
function Nope()
    apu.enable()
`

	res, err := CompileSource(src, "missing_start.corelx", nil)
	if err == nil {
		t.Fatalf("expected compile error, got nil")
	}
	if res == nil {
		t.Fatalf("expected compile result with diagnostics")
	}
	if len(res.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics, got none")
	}

	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "E_MISSING_ENTRYPOINT" {
			found = true
			if d.Stage != StageSemantic {
				t.Fatalf("expected semantic stage, got %s", d.Stage)
			}
			if d.File != "missing_start.corelx" {
				t.Fatalf("expected file stamp, got %q", d.File)
			}
		}
	}
	if !found {
		t.Fatalf("missing E_MISSING_ENTRYPOINT diagnostic: %+v", res.Diagnostics)
	}
}

func TestCompileProjectParserDiagnosticIncludesLocation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.corelx")
	// Missing function name after 'function'
	src := "function (\n"
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	res, err := CompileProject(path, nil)
	if err == nil {
		t.Fatalf("expected parse error")
	}
	if res == nil || len(res.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics")
	}
	if res.Diagnostics[0].Stage != StageParser {
		t.Fatalf("expected parser stage, got %s", res.Diagnostics[0].Stage)
	}
	if res.Diagnostics[0].Line == 0 {
		t.Fatalf("expected parser line/column, got %+v", res.Diagnostics[0])
	}
}

func TestCompileSourceReturnsManifestSkeleton(t *testing.T) {
	src := `
function Start()
    apu.enable()
`
	res, err := CompileSource(src, "manifest_test.corelx", nil)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
	if res == nil {
		t.Fatalf("expected compile result")
	}
	if res.Manifest == nil {
		t.Fatalf("expected build manifest")
	}
	if res.Manifest.ROMSizeBytes == 0 {
		t.Fatalf("expected non-zero rom size in manifest")
	}
	if len(res.Manifest.Sections) < 2 {
		t.Fatalf("expected section skeleton, got %+v", res.Manifest.Sections)
	}
	if res.Manifest.Sections[0].Name != "header" || res.Manifest.Sections[0].SizeBytes != 32 {
		t.Fatalf("unexpected header section: %+v", res.Manifest.Sections[0])
	}
	if res.Manifest.Sections[1].Name != "code" || res.Manifest.Sections[1].UsedBytes == 0 {
		t.Fatalf("unexpected code section: %+v", res.Manifest.Sections[1])
	}
	if res.Manifest.PlannedROMSizeBytes < res.Manifest.EmittedROMSizeBytes {
		t.Fatalf("expected planned ROM size >= emitted size, got planned=%d emitted=%d", res.Manifest.PlannedROMSizeBytes, res.Manifest.EmittedROMSizeBytes)
	}
	foundReservedAudio := false
	for _, s := range res.Manifest.Sections {
		if s.Name == "audio_seq" && s.Reserved {
			foundReservedAudio = true
			break
		}
	}
	if !foundReservedAudio {
		t.Fatalf("expected reserved audio_seq section placeholder")
	}
	if len(res.Manifest.SourceFiles) != 1 || res.Manifest.SourceFiles[0] != "manifest_test.corelx" {
		t.Fatalf("unexpected source files in manifest: %+v", res.Manifest.SourceFiles)
	}
}

func TestCompileSourceDuplicateAssetDiagnostic(t *testing.T) {
	src := `
asset Tiles: tiles8 hex
    00 00 00 00

asset Tiles: tiles8 hex
    00 00 00 00

function Start()
    apu.enable()
`
	res, err := CompileSource(src, "dup_asset.corelx", nil)
	if err == nil {
		t.Fatalf("expected compile error")
	}
	if res == nil {
		t.Fatalf("expected compile result")
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "E_ASSET_DUPLICATE" {
			found = true
			if d.Stage != StageSemantic {
				t.Fatalf("expected semantic stage, got %s", d.Stage)
			}
			if len(d.Related) == 0 || d.Related[0].Line == 0 {
				t.Fatalf("expected related previous declaration location, got %+v", d.Related)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected duplicate asset diagnostic, got %+v", res.Diagnostics)
	}
}

func TestCompileSourceAssetNormalizationUpdatesManifest(t *testing.T) {
	src := `
asset TileA: tiles8 hex
    00 11 22 33 44 55 66 77 88 99 AA BB CC DD EE FF
    00 11 22 33 44 55 66 77 88 99 AA BB CC DD EE FF

function Start()
    apu.enable()
`
	res, err := CompileSource(src, "asset_manifest.corelx", nil)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
	if len(res.NormalizedAssets) != 1 {
		t.Fatalf("expected 1 normalized asset, got %d", len(res.NormalizedAssets))
	}
	if got := len(res.NormalizedAssets[0].Data); got != 32 {
		t.Fatalf("expected 32-byte tiles8 asset, got %d", got)
	}

	if res.Manifest == nil {
		t.Fatalf("expected manifest")
	}
	var gfxSection *ManifestSection
	for i := range res.Manifest.Sections {
		if res.Manifest.Sections[i].Name == "gfx_tiles" {
			gfxSection = &res.Manifest.Sections[i]
			break
		}
	}
	if gfxSection == nil {
		t.Fatalf("missing gfx_tiles section")
	}
	if gfxSection.UsedBytes != 32 {
		t.Fatalf("expected gfx_tiles used bytes 32, got %d", gfxSection.UsedBytes)
	}
	if len(res.Manifest.Assets) != 1 {
		t.Fatalf("expected 1 manifest asset, got %d", len(res.Manifest.Assets))
	}
	if res.Manifest.Assets[0].Section != "gfx_tiles" || res.Manifest.Assets[0].SizeBytes != 32 {
		t.Fatalf("unexpected manifest asset ref: %+v", res.Manifest.Assets[0])
	}
	if res.Manifest.PlannedROMSizeBytes <= res.Manifest.EmittedROMSizeBytes {
		t.Fatalf("expected planned ROM size to include accounted asset section bytes (planned=%d emitted=%d)", res.Manifest.PlannedROMSizeBytes, res.Manifest.EmittedROMSizeBytes)
	}
}

func TestCompileSourceInvalidHexAssetDiagnostic(t *testing.T) {
	src := `
asset BadTile: tiles8 hex
    00 GG

function Start()
    apu.enable()
`
	res, err := CompileSource(src, "bad_asset.corelx", nil)
	if err == nil {
		t.Fatalf("expected asset parse error")
	}
	if res == nil {
		t.Fatalf("expected compile result")
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "E_ASSET_HEX_PARSE" {
			found = true
			if d.Stage != StageAsset {
				t.Fatalf("expected asset stage, got %s", d.Stage)
			}
			if d.Line == 0 {
				t.Fatalf("expected line/column on asset diagnostic, got %+v", d)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected E_ASSET_HEX_PARSE diagnostic, got %+v", res.Diagnostics)
	}
}

func TestCompileSourceManifestJSONOutput(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "build_manifest.json")
	bundlePath := filepath.Join(dir, "build_bundle.json")
	src := `
function Start()
    apu.enable()
`
	res, err := CompileSource(src, "manifest_json_test.corelx", &CompileOptions{
		ManifestOutputPath: manifestPath,
		BundleOutputPath:   bundlePath,
	})
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
	if res == nil || res.Manifest == nil {
		t.Fatalf("expected compile result with manifest")
	}
	if len(res.ManifestJSON) == 0 {
		t.Fatalf("expected manifest JSON bytes in result")
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest file: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("manifest JSON invalid: %v", err)
	}
	if _, ok := parsed["sections"]; !ok {
		t.Fatalf("manifest JSON missing sections: %v", parsed)
	}
	if len(res.DiagnosticsJSON) == 0 {
		t.Fatalf("expected diagnostics JSON bytes in result")
	}
	if len(res.BundleJSON) == 0 {
		t.Fatalf("expected bundle JSON bytes in result")
	}
	bundleData, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle file: %v", err)
	}
	var bundle map[string]any
	if err := json.Unmarshal(bundleData, &bundle); err != nil {
		t.Fatalf("bundle JSON invalid: %v", err)
	}
	if success, ok := bundle["success"].(bool); !ok || !success {
		t.Fatalf("expected success=true in bundle, got %v", bundle["success"])
	}
}

func TestCompileSourceInlineTileLoadUsesNormalizedB64Asset(t *testing.T) {
	// 32 bytes of tile data (tiles8) encoded as base64
	src := `
asset T: tiles8 b64
    "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="

function Start()
    base := gfx.load_tiles(ASSET_T, 0)
    ppu.enable_display()
`
	res, err := CompileSource(src, "b64_tiles.corelx", nil)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
	if res == nil || len(res.ROMBytes) == 0 {
		t.Fatalf("expected compiled ROM bytes")
	}
	if len(res.NormalizedAssets) != 1 || len(res.NormalizedAssets[0].Data) != 32 {
		t.Fatalf("expected normalized 32-byte b64 tile asset, got %+v", res.NormalizedAssets)
	}
}

func TestCompileSourceDiagnosticsJSONOutputOnError(t *testing.T) {
	dir := t.TempDir()
	diagPath := filepath.Join(dir, "diagnostics.json")
	bundlePath := filepath.Join(dir, "bundle.json")
	src := `
function Nope()
    apu.enable()
`
	res, err := CompileSource(src, "diag_error.corelx", &CompileOptions{
		DiagnosticsOutputPath: diagPath,
		BundleOutputPath:      bundlePath,
	})
	if err == nil {
		t.Fatalf("expected compile error")
	}
	if res == nil || len(res.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics in result")
	}
	data, readErr := os.ReadFile(diagPath)
	if readErr != nil {
		t.Fatalf("read diagnostics file: %v", readErr)
	}
	var parsed []map[string]any
	if uErr := json.Unmarshal(data, &parsed); uErr != nil {
		t.Fatalf("invalid diagnostics JSON: %v", uErr)
	}
	if len(parsed) == 0 {
		t.Fatalf("expected at least one diagnostic in JSON")
	}
	bundleData, bErr := os.ReadFile(bundlePath)
	if bErr != nil {
		t.Fatalf("read bundle file: %v", bErr)
	}
	var bundle map[string]any
	if err := json.Unmarshal(bundleData, &bundle); err != nil {
		t.Fatalf("invalid bundle JSON: %v", err)
	}
	if success, ok := bundle["success"].(bool); !ok || success {
		t.Fatalf("expected success=false in bundle, got %v", bundle["success"])
	}
}

func TestCompileSourceMultiSectionAssetManifestAccounting(t *testing.T) {
	src := `
asset Pal: palette text
    "001F 03E0 7C00"

asset Map: tilemap text
    "0,1,2,3"

asset Music: music text
    "tempo=120; note C4 4"

asset Data: gamedata text
    "lives=3"

function Start()
    apu.enable()
`
	res, err := CompileSource(src, "multi_sections.corelx", nil)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
	if res == nil || res.Manifest == nil {
		t.Fatalf("expected manifest")
	}
	used := map[string]uint32{}
	for _, s := range res.Manifest.Sections {
		used[s.Name] = s.UsedBytes
	}
	if used["palettes"] == 0 {
		t.Fatalf("expected palette section usage > 0")
	}
	if used["tilemaps"] == 0 {
		t.Fatalf("expected tilemap section usage > 0")
	}
	if used["audio_seq"] == 0 {
		t.Fatalf("expected audio_seq section usage > 0")
	}
	if used["gamedata"] == 0 {
		t.Fatalf("expected gamedata section usage > 0")
	}
	if len(res.NormalizedAssets) != 4 {
		t.Fatalf("expected 4 normalized assets, got %d", len(res.NormalizedAssets))
	}
}

func TestCompileSourceSectionBudgetOverflowDiagnostic(t *testing.T) {
	src := `
asset TileA: tiles8 hex
    00 11 22 33 44 55 66 77 88 99 AA BB CC DD EE FF
    00 11 22 33 44 55 66 77 88 99 AA BB CC DD EE FF

function Start()
    apu.enable()
`
	res, err := CompileSource(src, "budget_section.corelx", &CompileOptions{
		SectionBudgets: map[string]uint32{
			"gfx_tiles": 16,
		},
	})
	if err == nil {
		t.Fatalf("expected section overflow error")
	}
	if res == nil {
		t.Fatalf("expected compile result")
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "E_OVERFLOW_SECTION" {
			found = true
			if d.Stage != StagePack || d.Category != CategoryOverflowError {
				t.Fatalf("unexpected overflow diagnostic: %+v", d)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected E_OVERFLOW_SECTION diagnostic, got %+v", res.Diagnostics)
	}
}

func TestCompileSourceROMBudgetOverflowDiagnostic(t *testing.T) {
	src := `
function Start()
    apu.enable()
`
	res, err := CompileSource(src, "budget_rom.corelx", &CompileOptions{
		MaxROMBytes: 16,
	})
	if err == nil {
		t.Fatalf("expected ROM overflow error")
	}
	if res == nil {
		t.Fatalf("expected compile result")
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "E_OVERFLOW_ROM" {
			found = true
			if d.Stage != StagePack || d.Category != CategoryOverflowError {
				t.Fatalf("unexpected ROM overflow diagnostic: %+v", d)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected E_OVERFLOW_ROM diagnostic, got %+v", res.Diagnostics)
	}
}
