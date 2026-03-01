package devkit

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"nitro-core-dx/internal/corelx"
	"nitro-core-dx/internal/emulator"
)

type BuildArtifacts struct {
	ROMPath         string `json:"rom_path"`
	ManifestPath    string `json:"manifest_path"`
	DiagnosticsPath string `json:"diagnostics_path"`
	BundlePath      string `json:"bundle_path"`
}

type BuildResult struct {
	Bundle     corelx.CompileBundle  `json:"bundle"`
	Result     *corelx.CompileResult `json:"-"`
	Artifacts  BuildArtifacts        `json:"artifacts"`
	Elapsed    time.Duration         `json:"-"`
	SourcePath string                `json:"source_path"`
}

type EmulatorSnapshot struct {
	Loaded            bool
	Running           bool
	Paused            bool
	FPS               float64
	CPUCyclesPerFrame uint32
	FrameCount        uint64
}

type TickResult struct {
	Snapshot      EmulatorSnapshot `json:"snapshot"`
	FramesStepped int              `json:"frames_stepped"`
	PresentFrame  bool             `json:"present_frame"`
	Framebuffer   []uint32         `json:"-"`
	AudioFrames   [][]int16        `json:"-"`
}

type CPURegistersSnapshot struct {
	Loaded bool
	R0     uint16
	R1     uint16
	R2     uint16
	R3     uint16
	R4     uint16
	R5     uint16
	R6     uint16
	R7     uint16
}

type PCStateSnapshot struct {
	Loaded   bool
	PCBank   uint8
	PCOffset uint16
	PBR      uint8
	DBR      uint8
	SP       uint16
	Flags    uint8
	Cycles   uint32
}

// Backend defines the UI-agnostic Dev Kit contract intended for frontend wrappers.
// Frontends may be rewritten freely as long as they target this contract (or a compatible superset)
// and preserve emulator input/output semantics.
type Backend interface {
	TempDir() string
	BuildSource(source, sourcePath string) (*BuildResult, error)
	LoadROMBytes(romBytes []byte) error
	Shutdown()
	Snapshot() EmulatorSnapshot
	ResetEmulator() error
	TogglePause() (bool, error)
	SetInputButtons(buttons uint16)
	RunFrame() error
	StepFrame(frames int) error
	StepCPU(steps int) error
	Tick(delta time.Duration) (TickResult, error)
	FramebufferCopy() []uint32
	AudioSamplesFixedCopy() []int16
	GetRegisters() CPURegistersSnapshot
	GetPCState() PCStateSnapshot
}

// Service is the UI-agnostic Dev Kit backend wrapper.
// It owns the compiler service and an embedded emulator session while keeping
// emulator semantics unchanged.
type Service struct {
	tempDir string

	compiler *corelx.Service

	mu              sync.RWMutex
	emu             *emulator.Emulator
	tickAccumulator time.Duration
}

var _ Backend = (*Service)(nil)

func NewService(tempDir string) *Service {
	return &Service{
		tempDir:  tempDir,
		compiler: corelx.NewService(),
	}
}

func (s *Service) TempDir() string {
	return s.tempDir
}

func (s *Service) BuildSource(source, sourcePath string) (*BuildResult, error) {
	if sourcePath == "" {
		sourcePath = "untitled.corelx"
	}
	artifactBase := strings.TrimSuffix(baseNameOr(sourcePath, "untitled.corelx"), filepath.Ext(sourcePath))
	if artifactBase == "" {
		artifactBase = "untitled"
	}

	artifacts := BuildArtifacts{
		ROMPath:         filepath.Join(s.tempDir, artifactBase+".rom"),
		ManifestPath:    filepath.Join(s.tempDir, artifactBase+".manifest.json"),
		DiagnosticsPath: filepath.Join(s.tempDir, artifactBase+".diagnostics.json"),
		BundlePath:      filepath.Join(s.tempDir, artifactBase+".bundle.json"),
	}

	start := time.Now()
	opts := &corelx.CompileOptions{
		OutputPath:            artifacts.ROMPath,
		ManifestOutputPath:    artifacts.ManifestPath,
		DiagnosticsOutputPath: artifacts.DiagnosticsPath,
		BundleOutputPath:      artifacts.BundlePath,
		EmitROMBytes:          true,
		EmitManifestJSON:      true,
		EmitDiagnosticsJSON:   true,
		EmitBundleJSON:        true,
	}
	bundle, res, err := s.compiler.CompileBundleSource(source, sourcePath, opts)
	return &BuildResult{
		Bundle:     bundle,
		Result:     res,
		Artifacts:  artifacts,
		Elapsed:    time.Since(start),
		SourcePath: sourcePath,
	}, err
}

func (s *Service) LoadROMBytes(romBytes []byte) error {
	if len(romBytes) == 0 {
		return fmt.Errorf("empty ROM bytes")
	}

	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romBytes); err != nil {
		if emu.Logger != nil {
			emu.Logger.Shutdown()
		}
		return err
	}
	emu.Start()
	emu.SetInputButtons(0)

	s.mu.Lock()
	old := s.emu
	s.emu = emu
	s.tickAccumulator = 0
	s.mu.Unlock()

	if old != nil {
		old.Stop()
		if old.Logger != nil {
			old.Logger.Shutdown()
		}
	}

	return nil
}

func (s *Service) Shutdown() {
	s.mu.Lock()
	emu := s.emu
	s.emu = nil
	s.tickAccumulator = 0
	s.mu.Unlock()
	if emu != nil {
		emu.Stop()
		if emu.Logger != nil {
			emu.Logger.Shutdown()
		}
	}
}

func (s *Service) Snapshot() EmulatorSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.emu == nil {
		return EmulatorSnapshot{}
	}
	return EmulatorSnapshot{
		Loaded:            true,
		Running:           s.emu.Running,
		Paused:            s.emu.Paused,
		FPS:               s.emu.GetFPS(),
		CPUCyclesPerFrame: s.emu.GetCPUCyclesPerFrame(),
		FrameCount:        s.emu.FrameCount,
	}
}

func (s *Service) ResetEmulator() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.emu == nil {
		return fmt.Errorf("no ROM loaded")
	}
	s.emu.Reset()
	return nil
}

func (s *Service) TogglePause() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.emu == nil {
		return false, fmt.Errorf("no ROM loaded")
	}
	if s.emu.Paused {
		s.emu.Resume()
		return false, nil
	}
	s.emu.Pause()
	return true, nil
}

func (s *Service) SetInputButtons(buttons uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.emu == nil {
		return
	}
	s.emu.SetInputButtons(buttons)
}

func (s *Service) RunFrame() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.emu == nil {
		return nil
	}
	return s.emu.RunFrame()
}

func (s *Service) StepFrame(frames int) error {
	if frames <= 0 {
		return fmt.Errorf("frames must be > 0")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.emu == nil {
		return fmt.Errorf("no ROM loaded")
	}

	wasPaused := s.emu.Paused
	if wasPaused {
		s.emu.Paused = false
		defer func() {
			s.emu.Paused = true
		}()
	}

	for i := 0; i < frames; i++ {
		if err := s.emu.RunFrame(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) StepCPU(steps int) error {
	if steps <= 0 {
		return fmt.Errorf("steps must be > 0")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.emu == nil {
		return fmt.Errorf("no ROM loaded")
	}

	for i := 0; i < steps; i++ {
		if err := s.emu.CPU.ExecuteInstruction(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Tick(delta time.Duration) (TickResult, error) {
	const (
		emuHz            = 60
		maxCatchUpFrames = 4
		maxDelta         = 250 * time.Millisecond
	)
	frameStep := time.Second / emuHz

	if delta < 0 {
		delta = 0
	}
	if delta > maxDelta {
		delta = maxDelta
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var out TickResult
	if s.emu == nil {
		return out, nil
	}

	if s.emu.Paused {
		s.tickAccumulator = 0
		out.Snapshot = EmulatorSnapshot{
			Loaded:            true,
			Running:           s.emu.Running,
			Paused:            s.emu.Paused,
			FPS:               s.emu.GetFPS(),
			CPUCyclesPerFrame: s.emu.GetCPUCyclesPerFrame(),
			FrameCount:        s.emu.FrameCount,
		}
		if s.emu.FrameCount%8 == 0 {
			out.PresentFrame = true
			out.Framebuffer = copyFramebufferLocked(s.emu)
		}
		return out, nil
	}

	s.tickAccumulator += delta
	maxAccum := frameStep * maxCatchUpFrames
	if s.tickAccumulator > maxAccum {
		s.tickAccumulator = maxAccum
	}

	audioFrames := make([][]int16, 0, maxCatchUpFrames)
	for s.tickAccumulator >= frameStep && out.FramesStepped < maxCatchUpFrames {
		if err := s.emu.RunFrame(); err != nil {
			return out, err
		}
		audioFrames = append(audioFrames, copyAudioLocked(s.emu))
		s.tickAccumulator -= frameStep
		out.FramesStepped++
	}

	out.Snapshot = EmulatorSnapshot{
		Loaded:            true,
		Running:           s.emu.Running,
		Paused:            s.emu.Paused,
		FPS:               s.emu.GetFPS(),
		CPUCyclesPerFrame: s.emu.GetCPUCyclesPerFrame(),
		FrameCount:        s.emu.FrameCount,
	}
	out.AudioFrames = audioFrames
	if out.FramesStepped > 0 {
		out.PresentFrame = true
		out.Framebuffer = copyFramebufferLocked(s.emu)
	}
	return out, nil
}

func (s *Service) FramebufferCopy() []uint32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.emu == nil {
		return nil
	}
	return copyFramebufferLocked(s.emu)
}

func (s *Service) AudioSamplesFixedCopy() []int16 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.emu == nil {
		return nil
	}
	return copyAudioLocked(s.emu)
}

func (s *Service) GetRegisters() CPURegistersSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.emu == nil {
		return CPURegistersSnapshot{}
	}
	st := s.emu.CPU.State
	return CPURegistersSnapshot{
		Loaded: true,
		R0:     st.R0,
		R1:     st.R1,
		R2:     st.R2,
		R3:     st.R3,
		R4:     st.R4,
		R5:     st.R5,
		R6:     st.R6,
		R7:     st.R7,
	}
}

func (s *Service) GetPCState() PCStateSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.emu == nil {
		return PCStateSnapshot{}
	}
	st := s.emu.CPU.State
	return PCStateSnapshot{
		Loaded:   true,
		PCBank:   st.PCBank,
		PCOffset: st.PCOffset,
		PBR:      st.PBR,
		DBR:      st.DBR,
		SP:       st.SP,
		Flags:    st.Flags,
		Cycles:   st.Cycles,
	}
}

func baseNameOr(path, fallback string) string {
	if path == "" {
		return fallback
	}
	b := filepath.Base(path)
	if b == "." || b == string(filepath.Separator) || b == "" {
		return fallback
	}
	return b
}

func copyFramebufferLocked(emu *emulator.Emulator) []uint32 {
	src := emu.GetOutputBuffer()
	dst := make([]uint32, len(src))
	copy(dst, src)
	return dst
}

func copyAudioLocked(emu *emulator.Emulator) []int16 {
	src := emu.AudioSampleBuffer
	dst := make([]int16, len(src))
	copy(dst, src)
	return dst
}
