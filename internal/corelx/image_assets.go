package corelx

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"nitro-core-dx/internal/rom"
	"nitro-core-dx/internal/ymstream"
)

// ImageAsset is a parsed external .cxasset bitmap image, placed in ROM.
type ImageAsset struct {
	Name        string
	PlaneSize   int      // 32, 64, or 128 (tiles per side)
	PaletteBank uint8    // CGRAM bank (0-15)
	Palette     []uint16 // RGB555 colors
	Bitmap      []byte   // 4bpp packed bitmap
	Bank        uint8    // ROM bank where Bitmap starts
	Offset      uint16   // ROM offset (0x8000-based) where Bitmap starts
}

// loadImageAssets reads and parses every `image` asset's external .cxasset
// file, lays the bitmaps out in the ROM data region starting at startBank
// (the first bank above the compiled code, known only once codegen's bank
// count is final -- see the compiler's pass 1/2/3 driver), and returns the
// assets (with bank/offset filled in) plus the concatenated data region
// bytes for ROMBuilder.SetDataRegion.
func loadImageAssets(program *Program, sourcePath string, startBank uint8) (map[string]*ImageAsset, []byte, error) {
	srcDir := filepath.Dir(sourcePath)
	assets := make(map[string]*ImageAsset)
	var region []byte
	cursor := 0 // byte offset within the data region (startBank, 0x8000 = cursor 0)

	for _, a := range program.Assets {
		if a.Type != "image" {
			continue
		}
		var rawText string
		if a.FilePath == bootLogoSentinelPath {
			rawText = embeddedBootLogoCxasset
		} else {
			path := a.FilePath
			if !filepath.IsAbs(path) {
				path = filepath.Join(srcDir, path)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return nil, nil, fmt.Errorf("image asset %s: %w", a.Name, err)
			}
			rawText = string(raw)
		}
		img, err := parseCxAsset(a.Name, rawText)
		if err != nil {
			return nil, nil, fmt.Errorf("image asset %s (%s): %w", a.Name, a.FilePath, err)
		}

		// Place the bitmap in the ROM data region.
		img.Bank = uint8(int(startBank) + cursor/rom.ROMBankSizeBytes)
		img.Offset = uint16(rom.ROMBankOffsetBase + (cursor % rom.ROMBankSizeBytes))
		region = append(region, img.Bitmap...)
		cursor += len(img.Bitmap)

		assets[a.Name] = img
	}

	// Orphan check: every .cxasset file in the project directory must be
	// referenced by an image asset declaration. A stray asset file (dead art,
	// or a typo'd reference that left the file behind) is a hard error.
	referenced := make(map[string]bool)
	for _, a := range program.Assets {
		if a.Type == "image" {
			referenced[filepath.Base(a.FilePath)] = true
		}
	}
	entries, _ := os.ReadDir(srcDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".cxasset") {
			continue
		}
		if !referenced[e.Name()] {
			return nil, nil, fmt.Errorf("asset file %q is in the project but not referenced by any code (orphan); remove it or add `asset <Name>: image %q`", e.Name(), e.Name())
		}
	}
	return assets, region, nil
}

// parseCxAsset parses the importer's .cxasset text format:
//
//	image Name:
//	    kind: bitmap_plane
//	    plane_size: 32
//	    palette_bank: 1
//	    palette: hex 0000 7fff ...
//	    data: hex
//	        a0 fa ...
func parseCxAsset(name, text string) (*ImageAsset, error) {
	img := &ImageAsset{Name: name}
	lines := strings.Split(text, "\n")
	inData := false
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "--") {
			continue
		}
		if inData {
			for _, tok := range strings.Fields(t) {
				v, err := strconv.ParseUint(tok, 16, 8)
				if err != nil {
					return nil, fmt.Errorf("bad data byte %q: %w", tok, err)
				}
				img.Bitmap = append(img.Bitmap, byte(v))
			}
			continue
		}
		switch {
		case strings.HasPrefix(t, "plane_size:"):
			n, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(t, "plane_size:")))
			img.PlaneSize = n
		case strings.HasPrefix(t, "palette_bank:"):
			n, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(t, "palette_bank:")))
			img.PaletteBank = uint8(n)
		case strings.HasPrefix(t, "palette: hex"):
			for _, tok := range strings.Fields(strings.TrimPrefix(t, "palette: hex")) {
				v, err := strconv.ParseUint(tok, 16, 16)
				if err != nil {
					return nil, fmt.Errorf("bad palette color %q: %w", tok, err)
				}
				img.Palette = append(img.Palette, uint16(v))
			}
		case strings.HasPrefix(t, "data: hex"):
			inData = true
		}
	}
	if img.PlaneSize != 32 && img.PlaneSize != 64 && img.PlaneSize != 128 {
		return nil, fmt.Errorf("plane_size must be 32, 64, or 128 (got %d)", img.PlaneSize)
	}
	if len(img.Bitmap) == 0 {
		return nil, fmt.Errorf("no bitmap data")
	}
	return img, nil
}

// MusicAsset is a parsed external .ncdxmusic YM2608 stream, decoded and laid
// out in ROM for frame-by-frame playback. The minimal player (music.play) walks
// the song one frame per VBlank: each frame it reads the write count from the
// counts table and the (bank, offset) of that frame's writes from the pointer
// table, then hands the run of (port, addr, data) triplets to the bus-side YM
// burst streamer (0x9110-0x9115). The three sections are laid out contiguously:
//
//	counts table : 2 bytes/frame (uint16 LE write count)
//	write stream : 3 bytes/write (port, addr, data), all frames concatenated
//	pointer table: 4 bytes/frame (bank, offset_lo, offset_hi, 0) into the stream
//
// This is the same in-ROM shape the YM2608 demo ROM builders use. The
// `.ncdxmusic` file format itself is unchanged; this is only its ROM image.
type MusicAsset struct {
	Name       string
	FrameCount int    // number of song frames (one advance per VBlank)
	CountsBank uint8  // ROM bank of the counts table
	CountsOff  uint16 // ROM offset (0x8000-based) of the counts table
	PtrBank    uint8  // ROM bank of the pointer table
	PtrOff     uint16 // ROM offset (0x8000-based) of the pointer table
}

// dataAddr maps a flat byte cursor in the shared ROM data region (cursor 0 =
// bank startBank, offset 0x8000) to its (bank, 0x8000-based offset).
func dataAddr(startBank uint8, cursor int) (uint8, uint16) {
	return uint8(int(startBank) + cursor/rom.ROMBankSizeBytes),
		uint16(rom.ROMBankOffsetBase + (cursor % rom.ROMBankSizeBytes))
}

// loadMusicAssets reads and validates every `music` asset's external
// .ncdxmusic file, decodes it, and lays out the playback tables (counts, write
// stream, pointers) in the shared ROM data region starting at startBank,
// continuing from baseCursor (the byte length already used by image assets)
// so music and image data never overlap. Returns the assets (with table
// bank/offset filled in) and the bytes to append to the data region.
func loadMusicAssets(program *Program, sourcePath string, startBank uint8, baseCursor int) (map[string]*MusicAsset, []byte, error) {
	srcDir := filepath.Dir(sourcePath)
	assets := make(map[string]*MusicAsset)
	var region []byte
	cursor := baseCursor

	for _, a := range program.Assets {
		if a.Type != "music" {
			continue
		}
		path := a.FilePath
		if !filepath.IsAbs(path) {
			path = filepath.Join(srcDir, path)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("music asset %s: %w", a.Name, err)
		}
		// Validate and decode the YM2608 stream at compile time.
		song, err := ymstream.DecodeStream(raw)
		if err != nil {
			return nil, nil, fmt.Errorf("music asset %s (%s): invalid .ncdxmusic stream: %w", a.Name, path, err)
		}

		nf := len(song.Frames)
		// The minimal player indexes the counts/pointer tables with the DBR
		// pinned to a single bank, so each table must stay within its starting
		// bank. (A long song still fits: ~16K frames for the counts table.)
		countsBank, countsOff := dataAddr(startBank, cursor)
		if int(countsOff)+nf*2 > 0x10000 {
			return nil, nil, fmt.Errorf("music asset %s: too large for the minimal player (counts table crosses a ROM bank)", a.Name)
		}
		counts := make([]byte, nf*2)
		totalWrites := 0
		for i, fr := range song.Frames {
			binary.LittleEndian.PutUint16(counts[i*2:i*2+2], uint16(len(fr)))
			totalWrites += len(fr)
		}

		// Write stream (port, addr, data triplets) plus a pointer table whose
		// entries hold the absolute (bank, offset) of each frame's triplets.
		writesStart := cursor + nf*2
		writes := make([]byte, 0, totalWrites*3)
		ptrs := make([]byte, nf*4)
		byteOffset := 0
		for i, fr := range song.Frames {
			b, o := dataAddr(startBank, writesStart+byteOffset)
			ptrs[i*4] = b
			binary.LittleEndian.PutUint16(ptrs[i*4+1:i*4+3], o)
			ptrs[i*4+3] = 0
			for _, w := range fr {
				writes = append(writes, w.Port, w.Addr, w.Data)
			}
			byteOffset += len(fr) * 3
		}
		ptrStart := writesStart + len(writes)
		ptrBank, ptrOff := dataAddr(startBank, ptrStart)
		if int(ptrOff)+nf*4 > 0x10000 {
			return nil, nil, fmt.Errorf("music asset %s: too large for the minimal player (pointer table crosses a ROM bank)", a.Name)
		}

		region = append(region, counts...)
		region = append(region, writes...)
		region = append(region, ptrs...)
		cursor = ptrStart + len(ptrs)

		assets[a.Name] = &MusicAsset{
			Name:       a.Name,
			FrameCount: nf,
			CountsBank: countsBank,
			CountsOff:  countsOff,
			PtrBank:    ptrBank,
			PtrOff:     ptrOff,
		}
	}

	// Orphan check: every .ncdxmusic file in the project must be referenced by a
	// music asset declaration (same policy as .cxasset image assets).
	referenced := make(map[string]bool)
	for _, a := range program.Assets {
		if a.Type == "music" {
			referenced[filepath.Base(a.FilePath)] = true
		}
	}
	entries, _ := os.ReadDir(srcDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".ncdxmusic") {
			continue
		}
		if !referenced[e.Name()] {
			return nil, nil, fmt.Errorf("music file %q is in the project but not referenced by any code (orphan); remove it or add `asset <Name>: music %q`", e.Name(), e.Name())
		}
	}
	return assets, region, nil
}
