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

	raw, err := os.ReadFile(*inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", *inPath, err)
		os.Exit(1)
	}
	encoded, report, err := ymstream.ConvertVGM(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "convert VGM: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outPath, encoded, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *outPath, err)
		os.Exit(1)
	}

	naiveBytes := report.FrameCount*2 + report.WriteCount*3
	fmt.Printf("Wrote %s\n", *outPath)
	fmt.Print(report.String())
	fmt.Printf("Naive timed stream bytes (writes + per-frame waits): %d\n", naiveBytes)
	if naiveBytes > 0 {
		fmt.Printf("Reduction vs naive timed stream: %.2fx\n", float64(naiveBytes)/float64(len(encoded)))
	}
}
