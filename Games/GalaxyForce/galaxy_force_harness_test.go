package main

import (
	"os"
	"path/filepath"
	"testing"

	"nitro-core-dx/internal/harness"
	"nitro-core-dx/internal/emulator"
)

const (
	// Button bits (match game/input)
	ButtonStart = 0x400
	ButtonZ     = 0x10
)

// galaxyForceInputScript returns per-frame input for a deterministic run:
// - Frames 0-59: title (no input)
// - Frame 60: Start pressed (start game)
// - Frames 61-119: gameplay, no fire
// - Frame 120: Z pressed (one frame for edge-trigger shoot)
// - Frames 121-299: continue
func galaxyForceInputScript(numFrames int) []uint16 {
	script := make([]uint16, numFrames)
	for i := range script {
		script[i] = 0
	}
	if numFrames > 60 {
		script[60] = ButtonStart
	}
	if numFrames > 120 {
		script[120] = ButtonZ
	}
	return script
}

// TestGalaxyForceRecordReplay runs a deterministic script and either records a new
// golden file (RECORD_GALAXY_FORCE=1) or replays and compares to the golden.
func TestGalaxyForceRecordReplay(t *testing.T) {
	const numFrames = 300
	script := galaxyForceInputScript(numFrames)

	srcPath := filepath.Join("main.corelx")
	romData := compileROM(t, srcPath)

	if os.Getenv("RECORD_GALAXY_FORCE") == "1" {
		// Record: run and save golden recording
		emu := emulator.NewEmulator()
		if err := emu.LoadROM(romData); err != nil {
			t.Fatalf("load ROM: %v", err)
		}
		emu.Start()
		emu.SetFrameLimit(false)
		rec, err := harness.Record(emu, script, true)
		if err != nil {
			t.Fatalf("record: %v", err)
		}
		testdata := filepath.Join("testdata")
		if err := os.MkdirAll(testdata, 0755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		goldenPath := filepath.Join(testdata, "galaxy_force_golden.json")
		if err := harness.Save(rec, goldenPath); err != nil {
			t.Fatalf("save recording: %v", err)
		}
		t.Logf("Recorded %d frames to %s", numFrames, goldenPath)
		return
	}

	// Replay: load golden and compare
	goldenPath := filepath.Join("testdata", "galaxy_force_golden.json")
	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		t.Skipf("golden recording not found at %s; run with RECORD_GALAXY_FORCE=1 to create", goldenPath)
		return
	}
	rec, err := harness.Load(goldenPath)
	if err != nil {
		t.Fatalf("load recording: %v", err)
	}
	diffDir := filepath.Join("testdata", "diff")
	dr, err := harness.ReplayAndCompare(romData, rec, diffDir)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if dr != nil {
		t.Fatalf("frame %d: framebuffer mismatch (input=0x%04X); expected hash %s, got %s; see %s for actual frame",
			dr.Frame, dr.Input, dr.ExpectedFBHash[:16], dr.ActualFBHash[:16], diffDir)
	}
}

// TestGalaxyForceExportReplay exports the golden replay as PNGs and frames.json
// for visual review. Run after recording a golden; output in testdata/replay_frames.
func TestGalaxyForceExportReplay(t *testing.T) {
	goldenPath := filepath.Join("testdata", "galaxy_force_golden.json")
	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		t.Skipf("golden recording not found; run RECORD_GALAXY_FORCE=1 first")
		return
	}
	rec, err := harness.Load(goldenPath)
	if err != nil {
		t.Fatalf("load recording: %v", err)
	}
	romData := compileROM(t, filepath.Join("main.corelx"))
	outDir := filepath.Join("testdata", "replay_frames")
	if err := harness.ExportReplayToFrames(romData, rec, outDir); err != nil {
		t.Fatalf("export replay: %v", err)
	}
	t.Logf("Exported %d frames to %s (frame_00000.png, ..., frames.json)", len(rec.Frames), outDir)
}
