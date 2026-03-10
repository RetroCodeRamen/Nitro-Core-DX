package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
)

type wavPCM struct {
	sampleRate uint32
	channels   uint16
	samples    []float64 // mono normalized [-1,1]
}

func loadWAV(path string) (*wavPCM, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < 44 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return nil, errors.New("not a RIFF/WAVE file")
	}

	var (
		audioFmt      uint16
		channels      uint16
		sampleRate    uint32
		bitsPerSample uint16
		dataChunk     []byte
	)

	for p := 12; p+8 <= len(b); {
		id := string(b[p : p+4])
		sz := int(binary.LittleEndian.Uint32(b[p+4 : p+8]))
		p += 8
		if p+sz > len(b) {
			return nil, errors.New("invalid chunk size")
		}
		switch id {
		case "fmt ":
			if sz < 16 {
				return nil, errors.New("invalid fmt chunk")
			}
			audioFmt = binary.LittleEndian.Uint16(b[p : p+2])
			channels = binary.LittleEndian.Uint16(b[p+2 : p+4])
			sampleRate = binary.LittleEndian.Uint32(b[p+4 : p+8])
			bitsPerSample = binary.LittleEndian.Uint16(b[p+14 : p+16])
		case "data":
			dataChunk = b[p : p+sz]
		}
		p += sz
		if sz%2 == 1 {
			p++
		}
	}

	if audioFmt != 1 {
		return nil, fmt.Errorf("unsupported format %d (PCM=1 required)", audioFmt)
	}
	if bitsPerSample != 16 {
		return nil, fmt.Errorf("unsupported bits/sample %d (16 required)", bitsPerSample)
	}
	if channels == 0 {
		return nil, errors.New("invalid channel count")
	}
	if len(dataChunk) == 0 {
		return nil, errors.New("missing data chunk")
	}

	frameBytes := int(channels) * 2
	nFrames := len(dataChunk) / frameBytes
	mono := make([]float64, nFrames)
	for i := 0; i < nFrames; i++ {
		base := i * frameBytes
		sum := 0.0
		for c := 0; c < int(channels); c++ {
			s := int16(binary.LittleEndian.Uint16(dataChunk[base+c*2 : base+c*2+2]))
			sum += float64(s) / 32768.0
		}
		mono[i] = sum / float64(channels)
	}

	return &wavPCM{
		sampleRate: sampleRate,
		channels:   channels,
		samples:    mono,
	}, nil
}

func rms(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	sum := 0.0
	for _, x := range v {
		sum += x * x
	}
	return math.Sqrt(sum / float64(len(v)))
}

func main() {
	refPath := flag.String("ref", "Resources/Demo.wav", "Reference WAV path")
	gotPath := flag.String("got", "", "Captured WAV path")
	seconds := flag.Float64("seconds", 30, "Compare this many leading seconds (0 = full overlap)")
	flag.Parse()

	if *gotPath == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/wav_compare -ref Resources/Demo.wav -got <capture.wav> [-seconds N]")
		os.Exit(2)
	}

	ref, err := loadWAV(*refPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load ref: %v\n", err)
		os.Exit(1)
	}
	got, err := loadWAV(*gotPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load got: %v\n", err)
		os.Exit(1)
	}
	if ref.sampleRate != got.sampleRate {
		fmt.Fprintf(os.Stderr, "sample-rate mismatch: ref=%d got=%d\n", ref.sampleRate, got.sampleRate)
		os.Exit(1)
	}

	n := len(ref.samples)
	if len(got.samples) < n {
		n = len(got.samples)
	}
	if *seconds > 0 {
		limit := int(*seconds * float64(ref.sampleRate))
		if limit < n {
			n = limit
		}
	}
	if n <= 0 {
		fmt.Fprintln(os.Stderr, "no overlap to compare")
		os.Exit(1)
	}

	refSlice := append([]float64(nil), ref.samples[:n]...)
	gotSlice := append([]float64(nil), got.samples[:n]...)

	// Remove DC.
	refMean, gotMean := 0.0, 0.0
	for i := 0; i < n; i++ {
		refMean += refSlice[i]
		gotMean += gotSlice[i]
	}
	refMean /= float64(n)
	gotMean /= float64(n)
	for i := 0; i < n; i++ {
		refSlice[i] -= refMean
		gotSlice[i] -= gotMean
	}

	refRMS := rms(refSlice)
	gotRMS := rms(gotSlice)
	if refRMS == 0 || gotRMS == 0 {
		fmt.Fprintln(os.Stderr, "zero-energy signal encountered")
		os.Exit(1)
	}

	// Normalize energy for shape comparison.
	for i := 0; i < n; i++ {
		refSlice[i] /= refRMS
		gotSlice[i] /= gotRMS
	}

	// Zero-lag Pearson correlation + normalized MSE.
	dot, refPow, gotPow, mse := 0.0, 0.0, 0.0, 0.0
	for i := 0; i < n; i++ {
		r := refSlice[i]
		g := gotSlice[i]
		dot += r * g
		refPow += r * r
		gotPow += g * g
		d := r - g
		mse += d * d
	}
	corr := dot / math.Sqrt(refPow*gotPow)
	mse /= float64(n)

	fmt.Printf("Reference: %s (rate=%d, channels=%d)\n", *refPath, ref.sampleRate, ref.channels)
	fmt.Printf("Captured : %s (rate=%d, channels=%d)\n", *gotPath, got.sampleRate, got.channels)
	fmt.Printf("Compared : %d samples (%.2fs)\n", n, float64(n)/float64(ref.sampleRate))
	fmt.Printf("Metrics  : corr=%.6f  norm_mse=%.6f  rms_ref=%.6f  rms_got=%.6f\n", corr, mse, refRMS, gotRMS)
}
