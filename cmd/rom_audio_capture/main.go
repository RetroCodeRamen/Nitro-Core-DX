package main

import (
	"crypto/sha256"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"strings"

	"nitro-core-dx/internal/emulator"
)

func writeWAVMono16(path string, sampleRate uint32, samples []int16) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dataBytes := uint32(len(samples) * 2)
	byteRate := sampleRate * 2
	blockAlign := uint16(2)
	bitsPerSample := uint16(16)
	riffSize := 4 + (8 + 16) + (8 + dataBytes)

	if _, err := f.Write([]byte("RIFF")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, riffSize); err != nil {
		return err
	}
	if _, err := f.Write([]byte("WAVE")); err != nil {
		return err
	}
	if _, err := f.Write([]byte("fmt ")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(16)); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(1)); err != nil { // PCM
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint16(1)); err != nil { // mono
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, sampleRate); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, byteRate); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, blockAlign); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, bitsPerSample); err != nil {
		return err
	}
	if _, err := f.Write([]byte("data")); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, dataBytes); err != nil {
		return err
	}
	return binary.Write(f, binary.LittleEndian, samples)
}

func main() {
	romPath := flag.String("rom", "", "Input ROM path")
	outPath := flag.String("out", "rom_capture.wav", "Output WAV path")
	frames := flag.Int("frames", 0, "Number of frames to run (0 = 4300)")
	audioBackend := flag.String("audio-backend", "ymfm", "Audio backend: ymfm")
	flag.Parse()

	if *romPath == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/rom_audio_capture -rom <path.rom> [-out file.wav] [-frames N] [-audio-backend ymfm]")
		os.Exit(2)
	}

	mode := strings.ToLower(strings.TrimSpace(*audioBackend))
	if mode == "" {
		mode = "ymfm"
	}
	if mode != "ymfm" {
		fmt.Fprintf(os.Stderr, "invalid -audio-backend %q (expected ymfm)\n", *audioBackend)
		os.Exit(2)
	}
	if err := os.Setenv("NCDX_YM_BACKEND", mode); err != nil {
		fmt.Fprintf(os.Stderr, "set NCDX_YM_BACKEND: %v\n", err)
		os.Exit(1)
	}

	nFrames := *frames
	if nFrames <= 0 {
		nFrames = 4300 // ~71.6s at 60Hz
	}

	romData, err := os.ReadFile(*romPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read ROM: %v\n", err)
		os.Exit(1)
	}

	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		fmt.Fprintf(os.Stderr, "load ROM: %v\n", err)
		os.Exit(1)
	}
	emu.SetFrameLimit(false)
	emu.Start()

	all := make([]int16, 0, nFrames*735)
	for i := 0; i < nFrames; i++ {
		emu.SetInputButtons(0)
		if err := emu.RunFrame(); err != nil {
			fmt.Fprintf(os.Stderr, "run frame %d: %v\n", i, err)
			os.Exit(1)
		}
		all = append(all, emu.AudioSampleBuffer...)
	}

	if err := writeWAVMono16(*outPath, 44100, all); err != nil {
		fmt.Fprintf(os.Stderr, "write WAV: %v\n", err)
		os.Exit(1)
	}

	// Print deterministic content hash for quick comparisons.
	raw := make([]byte, len(all)*2)
	for i, s := range all {
		binary.LittleEndian.PutUint16(raw[i*2:i*2+2], uint16(s))
	}
	h := sha256.Sum256(raw)

	fmt.Printf("Captured %d frames (%d samples) to %s\n", nFrames, len(all), *outPath)
	fmt.Printf("Backend: %s  Sample SHA256: %x\n", mode, h)
}
