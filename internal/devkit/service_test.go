package devkit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"nitro-core-dx/internal/emulator"
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

func TestServiceInstallRasterProgramSmoke(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)
	defer svc.Shutdown()

	src := `
function Start()
    while true
        wait_vblank()
`
	build, err := svc.BuildSource(src, "raster_smoke.corelx")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := svc.LoadROMBytes(build.Result.ROMBytes); err != nil {
		t.Fatalf("load rom: %v", err)
	}

	builder := emulator.NewRasterProgramBuilder(0x3000, emulator.RasterProgramLayout{
		Enabled:            true,
		LayerMask:          0x01,
		IncludeTilemapBase: true,
	})
	if err := builder.FillScanlineRange(0, 199, emulator.RasterScanlineProgram{
		Layers: [4]emulator.RasterLayerProgram{
			{
				TransformA:     0x0100,
				TransformD:     0x0100,
				HasTilemapBase: true,
				TilemapBase:    0x1000,
			},
		},
	}); err != nil {
		t.Fatalf("FillScanlineRange: %v", err)
	}

	if err := svc.InstallRasterProgram(builder.Build()); err != nil {
		t.Fatalf("InstallRasterProgram: %v", err)
	}
	if err := svc.RunFrame(); err != nil {
		t.Fatalf("RunFrame after raster install: %v", err)
	}
	if err := svc.ClearRasterProgram(); err != nil {
		t.Fatalf("ClearRasterProgram: %v", err)
	}
	if err := svc.RunFrame(); err != nil {
		t.Fatalf("RunFrame after raster clear: %v", err)
	}
}

func TestServiceInstallRasterDemoSplitTilemapRenders(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)
	defer svc.Shutdown()

	src := `
function Start()
    while true
        wait_vblank()
`
	build, err := svc.BuildSource(src, "raster_demo_split.corelx")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := svc.LoadROMBytes(build.Result.ROMBytes); err != nil {
		t.Fatalf("load rom: %v", err)
	}
	if err := svc.InstallRasterDemo(RasterDemoSplitTilemap); err != nil {
		t.Fatalf("InstallRasterDemo(split): %v", err)
	}
	if err := svc.RunFrame(); err != nil {
		t.Fatalf("RunFrame: %v", err)
	}

	fb := svc.FramebufferCopy()
	top := fb[10*320+10]
	bottom := fb[(200-10)*320+10]
	if top == 0 || bottom == 0 {
		t.Fatalf("expected visible split-tilemap colors, got top=0x%06X bottom=0x%06X", top, bottom)
	}
	if top == bottom {
		t.Fatalf("expected split-tilemap demo to render distinct halves, got 0x%06X", top)
	}
}

func TestServiceInstallRasterDemoRebindPriorityRenders(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)
	defer svc.Shutdown()

	src := `
function Start()
    while true
        wait_vblank()
`
	build, err := svc.BuildSource(src, "raster_demo_rebind.corelx")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := svc.LoadROMBytes(build.Result.ROMBytes); err != nil {
		t.Fatalf("load rom: %v", err)
	}
	if err := svc.InstallRasterDemo(RasterDemoRebindPriority); err != nil {
		t.Fatalf("InstallRasterDemo(rebind): %v", err)
	}
	if err := svc.RunFrame(); err != nil {
		t.Fatalf("RunFrame: %v", err)
	}

	fb := svc.FramebufferCopy()
	top := fb[10*320+3]
	bottom := fb[(200-10)*320+3]
	if top == 0 || bottom == 0 {
		t.Fatalf("expected visible rebind/priority colors, got top=0x%06X bottom=0x%06X", top, bottom)
	}
	if top == bottom {
		t.Fatalf("expected rebind/priority demo to change visible output, got 0x%06X", top)
	}
}

func TestServiceInstallRasterDemoScrollAffineRenders(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)
	defer svc.Shutdown()

	src := `
function Start()
    while true
        wait_vblank()
`
	build, err := svc.BuildSource(src, "raster_demo_scroll_affine.corelx")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := svc.LoadROMBytes(build.Result.ROMBytes); err != nil {
		t.Fatalf("load rom: %v", err)
	}
	if err := svc.InstallRasterDemo(RasterDemoScrollAffine); err != nil {
		t.Fatalf("InstallRasterDemo(scroll-affine): %v", err)
	}
	if err := svc.RunFrame(); err != nil {
		t.Fatalf("RunFrame: %v", err)
	}

	fb := svc.FramebufferCopy()
	top := fb[10*320+4]
	bottom := fb[(200-10)*320+4]
	if top == 0 || bottom == 0 {
		t.Fatalf("expected visible scroll/affine colors, got top=0x%06X bottom=0x%06X", top, bottom)
	}
	if top == bottom {
		t.Fatalf("expected scroll/affine demo to change visible output, got 0x%06X", top)
	}
}

func TestServiceInstallRasterDemoMatrixPlaneRenders(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)
	defer svc.Shutdown()

	src := `
function Start()
    while true
        wait_vblank()
`
	build, err := svc.BuildSource(src, "raster_demo_matrix_plane.corelx")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := svc.LoadROMBytes(build.Result.ROMBytes); err != nil {
		t.Fatalf("load rom: %v", err)
	}
	if err := svc.InstallRasterDemo(RasterDemoMatrixPlane); err != nil {
		t.Fatalf("InstallRasterDemo(matrix-plane): %v", err)
	}
	if err := svc.RunFrame(); err != nil {
		t.Fatalf("RunFrame: %v", err)
	}

	fb := svc.FramebufferCopy()
	a := fb[30*320+30]
	b := fb[120*320+160]
	c := fb[180*320+280]
	if a == 0 || b == 0 || c == 0 {
		t.Fatalf("expected visible matrix-plane colors, got a=0x%06X b=0x%06X c=0x%06X", a, b, c)
	}
	if a == b && b == c {
		t.Fatalf("expected matrix-plane demo to produce spatially varying output, got a=b=c=0x%06X", a)
	}
}

func TestServiceInstallMatrixPlaneProgramWithBuilder(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)
	defer svc.Shutdown()

	src := `
function Start()
    while true
        wait_vblank()
`
	build, err := svc.BuildSource(src, "matrix_plane_builder.corelx")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := svc.LoadROMBytes(build.Result.ROMBytes); err != nil {
		t.Fatalf("load rom: %v", err)
	}

	emu := svc.emu
	emu.PPU.BG0.Enabled = true
	emu.PPU.BG0.TileSize = false
	emu.PPU.BG0.TilemapSize = 2
	emu.PPU.BG0.TransformChannel = 0
	emu.PPU.TransformChannels[0].Enabled = true
	emu.PPU.TransformChannels[0].A = 0x0100
	emu.PPU.TransformChannels[0].D = 0x0100
	emu.PPU.TransformChannels[0].CenterX = 0
	emu.PPU.TransformChannels[0].CenterY = 0
	emu.PPU.CGRAM[0x01*2] = 0x1F
	emu.PPU.CGRAM[0x01*2+1] = 0x00
	emu.PPU.CGRAM[0x02*2] = 0xE0
	emu.PPU.CGRAM[0x02*2+1] = 0x03

	builder, err := emulator.NewMatrixPlaneBuilder(0, 2)
	if err != nil {
		t.Fatalf("NewMatrixPlaneBuilder: %v", err)
	}
	if err := builder.SetPatternTile8x8(0, bytesRepeat(0x11, 32)); err != nil {
		t.Fatalf("SetPatternTile8x8(0): %v", err)
	}
	if err := builder.SetPatternTile8x8(1, bytesRepeat(0x22, 32)); err != nil {
		t.Fatalf("SetPatternTile8x8(1): %v", err)
	}
	if err := builder.FillRect(0, 0, 20, 128, 0x00, 0x00); err != nil {
		t.Fatalf("FillRect left: %v", err)
	}
	if err := builder.FillRect(20, 0, 108, 128, 0x01, 0x00); err != nil {
		t.Fatalf("FillRect right: %v", err)
	}
	if err := svc.InstallMatrixPlaneProgram(builder.Build()); err != nil {
		t.Fatalf("InstallMatrixPlaneProgram: %v", err)
	}
	if err := svc.RunFrame(); err != nil {
		t.Fatalf("RunFrame: %v", err)
	}

	fb := svc.FramebufferCopy()
	left := fb[20*320+20]
	right := fb[20*320+280]
	if left == 0 || right == 0 {
		t.Fatalf("expected visible matrix plane halves, got left=0x%06X right=0x%06X", left, right)
	}
	if left == right {
		t.Fatalf("expected matrix plane builder program to produce distinct halves, got 0x%06X", left)
	}
}

func bytesRepeat(value byte, count int) []byte {
	out := make([]byte, count)
	for i := range out {
		out[i] = value
	}
	return out
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
