// Package harness provides record/replay and comparison for deterministic
// emulator runs: record input + framebuffer (and optional audio) per frame,
// then replay with the same input and compare output to catch regressions.
package harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"nitro-core-dx/internal/emulator"
)

const (
	// Width and Height are the PPU display dimensions.
	Width  = 320
	Height = 200
)

// FrameRecord holds per-frame input and output hashes for one frame.
type FrameRecord struct {
	Input          uint16 `json:"input"`
	FramebufferHash string `json:"fb_hash"`
	AudioHash      string `json:"audio_hash,omitempty"`
}

// Recording is a full run: same-length input script and per-frame hashes.
// ROM is not stored; use ROMHash to ensure you replay with the same ROM.
type Recording struct {
	ROMHash   string        `json:"rom_hash"`   // SHA256 of ROM bytes (hex)
	Frames    []FrameRecord `json:"frames"`
	Width     int           `json:"width"`
	Height    int           `json:"height"`
	IncludeAudio bool       `json:"include_audio,omitempty"`
}

// fbHash computes SHA256 of framebuffer (320*200 uint32 as little-endian bytes).
func fbHash(buf []uint32) string {
	if len(buf) < Width*Height {
		return ""
	}
	b := make([]byte, Width*Height*4)
	for i := 0; i < Width*Height; i++ {
		p := buf[i]
		b[i*4] = byte(p)
		b[i*4+1] = byte(p >> 8)
		b[i*4+2] = byte(p >> 16)
		b[i*4+3] = byte(p >> 24)
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// audioHash computes SHA256 of int16 audio samples (little-endian).
func audioHash(samples []int16) string {
	if len(samples) == 0 {
		return ""
	}
	b := make([]byte, len(samples)*2)
	for i, s := range samples {
		b[i*2] = byte(s)
		b[i*2+1] = byte(s >> 8)
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// Record runs the emulator for len(inputScript) frames, applying inputScript[i]
// before frame i, and captures framebuffer (and optionally audio) hash each frame.
// The emulator must already have a ROM loaded and be started.
func Record(emu *emulator.Emulator, inputScript []uint16, includeAudio bool) (*Recording, error) {
	n := len(inputScript)
	rec := &Recording{
		Frames:        make([]FrameRecord, 0, n),
		Width:         Width,
		Height:        Height,
		IncludeAudio:  includeAudio,
	}
	// ROM hash from cartridge (header + data) for replay verification
	if emu.Cartridge != nil && (len(emu.Cartridge.ROMHeader) > 0 || len(emu.Cartridge.ROMData) > 0) {
		fullROM := append(append([]byte(nil), emu.Cartridge.ROMHeader[:]...), emu.Cartridge.ROMData...)
		h := sha256.Sum256(fullROM)
		rec.ROMHash = hex.EncodeToString(h[:])
	}

	for i := 0; i < n; i++ {
		emu.SetInputButtons(inputScript[i])
		if err := emu.RunFrame(); err != nil {
			return nil, fmt.Errorf("frame %d: %w", i, err)
		}
		fr := FrameRecord{
			Input:          inputScript[i],
			FramebufferHash: fbHash(emu.GetOutputBuffer()),
		}
		if includeAudio {
			fr.AudioHash = audioHash(emu.AudioSampleBuffer)
		}
		rec.Frames = append(rec.Frames, fr)
	}
	return rec, nil
}

// Save writes the recording to a JSON file.
func Save(rec *Recording, path string) error {
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Load reads a recording from a JSON file.
func Load(path string) (*Recording, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rec Recording
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// DiffResult describes the first frame where replay differed from the recording.
type DiffResult struct {
	Frame          int
	ExpectedFBHash string
	ActualFBHash   string
	ExpectedAudioHash string
	ActualAudioHash   string
	Input          uint16
}

// ReplayAndCompare loads the ROM, runs the same number of frames as the recording
// with the same per-frame input, and compares framebuffer (and audio if present)
// hashes. Returns nil if identical; otherwise returns the first DiffResult and
// optionally writes expected/actual PNGs to outDir (if non-empty).
func ReplayAndCompare(rom []byte, rec *Recording, outDir string) (*DiffResult, error) {
	emu := emulator.NewEmulator()
	if err := emu.LoadROM(rom); err != nil {
		return nil, fmt.Errorf("load ROM: %w", err)
	}
	emu.Start()
	emu.SetFrameLimit(false)

	// Optional: verify ROM hash
	if rec.ROMHash != "" && len(rom) > 0 {
		h := sha256.Sum256(rom)
		hexHash := hex.EncodeToString(h[:])
		if hexHash != rec.ROMHash {
			return nil, fmt.Errorf("ROM hash mismatch: recording was made with a different ROM")
		}
	}

	n := len(rec.Frames)
	for i := 0; i < n; i++ {
		input := rec.Frames[i].Input
		emu.SetInputButtons(input)
		if err := emu.RunFrame(); err != nil {
			return nil, fmt.Errorf("frame %d: %w", i, err)
		}
		actualFB := fbHash(emu.GetOutputBuffer())
		expectedFB := rec.Frames[i].FramebufferHash
		if actualFB != expectedFB {
			dr := &DiffResult{
				Frame:          i,
				ExpectedFBHash: expectedFB,
				ActualFBHash:   actualFB,
				Input:          input,
			}
			if outDir != "" {
				writeFramePNG(emu.GetOutputBuffer(), filepath.Join(outDir, fmt.Sprintf("actual_frame_%05d.png", i)))
			}
			return dr, nil
		}
		if rec.IncludeAudio && rec.Frames[i].AudioHash != "" {
			actualAudio := audioHash(emu.AudioSampleBuffer)
			if actualAudio != rec.Frames[i].AudioHash {
				dr := &DiffResult{
					Frame:            i,
					ExpectedAudioHash: rec.Frames[i].AudioHash,
					ActualAudioHash:   actualAudio,
					Input:            input,
				}
				return dr, nil
			}
		}
	}
	return nil, nil
}

func writeFramePNG(buf []uint32, path string) {
	img := image.NewRGBA(image.Rect(0, 0, Width, Height))
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			c := buf[y*Width+x]
			img.Set(x, y, color.RGBA{
				R: uint8((c >> 16) & 0xFF),
				G: uint8((c >> 8) & 0xFF),
				B: uint8(c & 0xFF),
				A: 255,
			})
		}
	}
	f, _ := os.Create(path)
	if f != nil {
		_ = png.Encode(f, img)
		_ = f.Close()
	}
}

// ExportReplayToFrames replays the recording with the given ROM and writes
// each frame as a PNG (frame_00000.png, ...) and a summary JSON (frames.json)
// listing frame index and input for human review.
func ExportReplayToFrames(rom []byte, rec *Recording, outDir string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	emu := emulator.NewEmulator()
	if err := emu.LoadROM(rom); err != nil {
		return err
	}
	emu.Start()
	emu.SetFrameLimit(false)

	type frameMeta struct {
		Frame int    `json:"frame"`
		Input uint16 `json:"input_hex"`
	}
	var meta []frameMeta

	for i := 0; i < len(rec.Frames); i++ {
		emu.SetInputButtons(rec.Frames[i].Input)
		if err := emu.RunFrame(); err != nil {
			return fmt.Errorf("frame %d: %w", i, err)
		}
		path := filepath.Join(outDir, fmt.Sprintf("frame_%05d.png", i))
		writeFramePNG(emu.GetOutputBuffer(), path)
		meta = append(meta, frameMeta{Frame: i, Input: rec.Frames[i].Input})
	}

	metaPath := filepath.Join(outDir, "frames.json")
	data, _ := json.MarshalIndent(meta, "", "  ")
	return os.WriteFile(metaPath, data, 0644)
}
