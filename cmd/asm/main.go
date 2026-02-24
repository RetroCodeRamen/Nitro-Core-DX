package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	ncasm "nitro-core-dx/internal/asm"
)

func main() {
	entryBank := flag.Uint("entry-bank", 1, "entry bank")
	entryOffset := flag.Uint("entry-offset", 0x8000, "entry offset (hex or decimal)")
	flag.Parse()
	if flag.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s [--entry-bank N] [--entry-offset 0x8000] <input.asm> <output.rom>\n", os.Args[0])
		os.Exit(1)
	}
	in := flag.Arg(0)
	out := flag.Arg(1)
	res, err := ncasm.AssembleFile(in, &ncasm.Options{EntryBank: uint8(*entryBank), EntryOffset: uint16(*entryOffset), OutputPath: out})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Assembler error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Assembled %s -> %s\n", filepath.Base(in), filepath.Base(out))
	fmt.Printf("Entry: bank %d offset 0x%04X\n", res.EntryBank, res.EntryOffset)
	fmt.Printf("Code words: %d (%d bytes)\n", res.Words, res.Words*2)
}
