package devkit

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestServiceBuildSourceSuccessArtifacts(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	src := `
function Start()
    apu.enable()
`
	build, err := svc.BuildSource(src, "main.corelx")
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}
	if build == nil {
		t.Fatalf("expected build result")
	}
	if !build.Bundle.Success {
		t.Fatalf("expected successful bundle: %+v", build.Bundle)
	}
	if build.Result == nil || len(build.Result.ROMBytes) == 0 {
		t.Fatalf("expected ROM bytes in build result")
	}
	if build.Result.Manifest == nil {
		t.Fatalf("expected manifest in build result")
	}
	for _, p := range []string{build.Artifacts.ROMPath, build.Artifacts.ManifestPath, build.Artifacts.DiagnosticsPath, build.Artifacts.BundlePath} {
		if p == "" {
			t.Fatalf("expected artifact path")
		}
		if filepath.Dir(p) != tmpDir {
			t.Fatalf("expected artifact under temp dir %q, got %q", tmpDir, p)
		}
		if _, statErr := os.Stat(p); statErr != nil {
			t.Fatalf("expected artifact file %q: %v", p, statErr)
		}
	}
}

func TestServiceBuildSourceErrorDiagnostics(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	src := "function Nope()\n    apu.enable()\n"
	build, err := svc.BuildSource(src, "bad.corelx")
	if err == nil {
		t.Fatalf("expected build error")
	}
	if build == nil {
		t.Fatalf("expected build result with diagnostics")
	}
	if build.Bundle.Success {
		t.Fatalf("expected failed bundle")
	}
	if build.Bundle.Summary.ErrorCount == 0 {
		t.Fatalf("expected error count > 0")
	}
	if len(build.Bundle.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics in bundle")
	}
	if _, statErr := os.Stat(build.Artifacts.DiagnosticsPath); statErr != nil {
		t.Fatalf("expected diagnostics artifact file: %v", statErr)
	}
}

func TestServiceEmulatorSessionSmoke(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)
	defer svc.Shutdown()

	// Keep ROM execution stable by idling in a vblank loop after init.
	src := `
function Start()
    apu.enable()
    while true
        wait_vblank()
`
	build, err := svc.BuildSource(src, "session.corelx")
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}
	if build == nil || build.Result == nil || len(build.Result.ROMBytes) == 0 {
		t.Fatalf("expected compiled ROM bytes")
	}

	if err := svc.LoadROMBytes(build.Result.ROMBytes); err != nil {
		t.Fatalf("load rom bytes: %v", err)
	}

	snap := svc.Snapshot()
	if !snap.Loaded || !snap.Running {
		t.Fatalf("expected loaded/running snapshot, got %+v", snap)
	}
	if snap.Paused {
		t.Fatalf("expected not paused initially")
	}

	svc.SetInputButtons(0x123)
	if err := svc.RunFrame(); err != nil {
		t.Fatalf("run frame: %v", err)
	}
	snap = svc.Snapshot()
	if snap.FrameCount == 0 {
		t.Fatalf("expected frame count > 0 after RunFrame")
	}

	fb := svc.FramebufferCopy()
	if len(fb) != 320*200 {
		t.Fatalf("unexpected framebuffer length: %d", len(fb))
	}
	audio := svc.AudioSamplesFixedCopy()
	if len(audio) != 735 {
		t.Fatalf("unexpected audio buffer length: %d", len(audio))
	}

	paused, err := svc.TogglePause()
	if err != nil {
		t.Fatalf("toggle pause: %v", err)
	}
	if !paused {
		t.Fatalf("expected paused=true on first toggle")
	}
	paused, err = svc.TogglePause()
	if err != nil {
		t.Fatalf("toggle pause (resume): %v", err)
	}
	if paused {
		t.Fatalf("expected paused=false on second toggle")
	}
	if err := svc.ResetEmulator(); err != nil {
		t.Fatalf("reset emulator: %v", err)
	}

	svc.Shutdown()
	snap = svc.Snapshot()
	if snap.Loaded {
		t.Fatalf("expected unloaded snapshot after shutdown, got %+v", snap)
	}
}

func TestServiceTickReturnsFrameAndAudio(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)
	defer svc.Shutdown()

	src := `
function Start()
    apu.enable()
    while true
        wait_vblank()
`
	build, err := svc.BuildSource(src, "tick.corelx")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := svc.LoadROMBytes(build.Result.ROMBytes); err != nil {
		t.Fatalf("load rom: %v", err)
	}

	tick, err := svc.Tick(time.Second / 60)
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	if !tick.Snapshot.Loaded {
		t.Fatalf("expected loaded snapshot")
	}
	if tick.FramesStepped == 0 {
		t.Fatalf("expected at least one stepped frame")
	}
	if !tick.PresentFrame {
		t.Fatalf("expected present frame")
	}
	if len(tick.Framebuffer) != 320*200 {
		t.Fatalf("unexpected framebuffer length: %d", len(tick.Framebuffer))
	}
	if len(tick.AudioFrames) != tick.FramesStepped {
		t.Fatalf("expected audio frames == frames stepped, got %d vs %d", len(tick.AudioFrames), tick.FramesStepped)
	}
	for i, af := range tick.AudioFrames {
		if len(af) != 735 {
			t.Fatalf("unexpected audio frame length at %d: %d", i, len(af))
		}
	}
}

func TestServiceTickPausedPresentsWithoutStepping(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)
	defer svc.Shutdown()

	src := `
function Start()
    apu.enable()
    while true
        wait_vblank()
`
	build, err := svc.BuildSource(src, "tick_pause.corelx")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := svc.LoadROMBytes(build.Result.ROMBytes); err != nil {
		t.Fatalf("load rom: %v", err)
	}

	if _, err := svc.TogglePause(); err != nil {
		t.Fatalf("pause: %v", err)
	}
	tick, err := svc.Tick(time.Second / 60)
	if err != nil {
		t.Fatalf("tick paused: %v", err)
	}
	if tick.FramesStepped != 0 {
		t.Fatalf("expected no stepped frames while paused, got %d", tick.FramesStepped)
	}
	if !tick.Snapshot.Paused {
		t.Fatalf("expected paused snapshot")
	}
	// On initial framecount 0, paused tick should still request a present refresh.
	if !tick.PresentFrame {
		t.Fatalf("expected present frame on paused refresh")
	}
}

func TestServiceStepFrameWhilePaused(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)
	defer svc.Shutdown()

	src := `
function Start()
    apu.enable()
    while true
        wait_vblank()
`
	build, err := svc.BuildSource(src, "step_frame.corelx")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := svc.LoadROMBytes(build.Result.ROMBytes); err != nil {
		t.Fatalf("load rom: %v", err)
	}
	if _, err := svc.TogglePause(); err != nil {
		t.Fatalf("pause: %v", err)
	}

	before := svc.Snapshot()
	if !before.Paused {
		t.Fatalf("expected paused snapshot before StepFrame")
	}
	if err := svc.StepFrame(1); err != nil {
		t.Fatalf("step frame: %v", err)
	}
	after := svc.Snapshot()
	if !after.Paused {
		t.Fatalf("expected paused=true after StepFrame")
	}
	if after.FrameCount <= before.FrameCount {
		t.Fatalf("expected frame count to increase, before=%d after=%d", before.FrameCount, after.FrameCount)
	}
}

func TestServiceStepCPUAndSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)
	defer svc.Shutdown()

	src := `
function Start()
    apu.enable()
    while true
        wait_vblank()
`
	build, err := svc.BuildSource(src, "step_cpu.corelx")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := svc.LoadROMBytes(build.Result.ROMBytes); err != nil {
		t.Fatalf("load rom: %v", err)
	}
	if _, err := svc.TogglePause(); err != nil {
		t.Fatalf("pause: %v", err)
	}

	beforePC := svc.GetPCState()
	beforeRegs := svc.GetRegisters()
	if !beforePC.Loaded || !beforeRegs.Loaded {
		t.Fatalf("expected loaded snapshots")
	}

	if err := svc.StepCPU(1); err != nil {
		t.Fatalf("step cpu: %v", err)
	}

	afterPC := svc.GetPCState()
	afterRegs := svc.GetRegisters()
	if !afterPC.Loaded || !afterRegs.Loaded {
		t.Fatalf("expected loaded snapshots after step")
	}
	if afterPC.Cycles <= beforePC.Cycles {
		t.Fatalf("expected cycle count to increase, before=%d after=%d", beforePC.Cycles, afterPC.Cycles)
	}
	if afterPC.PCBank == beforePC.PCBank && afterPC.PCOffset == beforePC.PCOffset {
		t.Fatalf("expected PC to change after CPU step (%02X:%04X)", afterPC.PCBank, afterPC.PCOffset)
	}
}
