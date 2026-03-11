package main

import (
	"flag"
	"fmt"
	"os"

	"nitro-core-dx/internal/ymstream"
)

func main() {
	inPath := flag.String("in", "Resources/Demo.vgz", "Input VGM/VGZ file")
	outPath := flag.String("out", "demo.ncdxmusic", "Output compact YM stream file")
	flag.Parse()

	raw, err := ymstream.ReadVGM(*inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", *inPath, err)
		os.Exit(1)
	}
	song, err := ymstream.ParseVGM(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse VGM: %v\n", err)
		os.Exit(1)
	}
	encoded, err := ymstream.EncodeSong(song)
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode stream: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outPath, encoded, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *outPath, err)
		os.Exit(1)
	}

	naiveBytes := len(song.Frames)*2 + song.WriteCount*3
	seconds := float64(song.TotalSamples) / 44100.0
	fmt.Printf("Wrote %s\n", *outPath)
	fmt.Printf("Frames: %d  YM writes: %d  Duration: %.2fs\n", len(song.Frames), song.WriteCount, seconds)
	fmt.Printf("Compact stream bytes: %d\n", len(encoded))
	fmt.Printf("Naive timed stream bytes (writes + per-frame waits): %d\n", naiveBytes)
	if naiveBytes > 0 {
		fmt.Printf("Reduction vs naive timed stream: %.2fx\n", float64(naiveBytes)/float64(len(encoded)))
	}
}
