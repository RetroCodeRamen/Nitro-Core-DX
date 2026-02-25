package corelx

type BuildManifest struct {
	FormatVersion       int                `json:"format_version"`
	SourceFiles         []string           `json:"source_files"`
	EntryBank           uint8              `json:"entry_bank"`
	EntryOffset         uint16             `json:"entry_offset"`
	ROMSizeBytes        uint32             `json:"rom_size_bytes"`         // emitted ROM size (compat field)
	EmittedROMSizeBytes uint32             `json:"emitted_rom_size_bytes"` // explicit emitted ROM size
	PlannedROMSizeBytes uint32             `json:"planned_rom_size_bytes"` // manifest/planned layout size incl. accounted sections
	Sections            []ManifestSection  `json:"sections"`
	Assets              []ManifestAssetRef `json:"assets"`
}

type ManifestSection struct {
	Name      string `json:"name"`
	Offset    uint32 `json:"offset"`
	SizeBytes uint32 `json:"size_bytes"`
	UsedBytes uint32 `json:"used_bytes"`
	Reserved  bool   `json:"reserved"`
}

type ManifestAssetRef struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Section    string `json:"section"`
	Offset     uint32 `json:"offset"`
	SizeBytes  uint32 `json:"size_bytes"`
	SourceFile string `json:"source_file,omitempty"`
	Line       int    `json:"line,omitempty"`
	Column     int    `json:"column,omitempty"`
}

func buildManifestFromCompileState(sourcePath string, entryBank uint8, entryOffset uint16, codeBytes, romBytes uint32, program *Program, assets []AssetIR) *BuildManifest {
	sectionOrder := []string{"gfx_tiles", "tilemaps", "palettes", "audio_seq", "audio_patch", "gamedata"}
	sectionSizes := make(map[string]uint32, len(sectionOrder))
	for _, a := range assets {
		sectionSizes[a.Section] += uint32(len(a.Data))
	}

	manifest := &BuildManifest{
		FormatVersion:       1,
		EntryBank:           entryBank,
		EntryOffset:         entryOffset,
		ROMSizeBytes:        romBytes,
		EmittedROMSizeBytes: romBytes,
		Sections: []ManifestSection{
			{Name: "header", Offset: 0, SizeBytes: 32, UsedBytes: 32},
			{Name: "code", Offset: 32, SizeBytes: codeBytes, UsedBytes: codeBytes},
		},
		Assets: make([]ManifestAssetRef, 0, len(program.Assets)),
	}
	if sourcePath != "" {
		manifest.SourceFiles = []string{sourcePath}
	}
	cursor := uint32(32 + codeBytes)
	sectionStart := make(map[string]uint32, len(sectionOrder))
	for _, name := range sectionOrder {
		size := sectionSizes[name]
		sectionStart[name] = cursor
		manifest.Sections = append(manifest.Sections, ManifestSection{
			Name:      name,
			Offset:    cursor,
			SizeBytes: size,
			UsedBytes: size,
			Reserved:  true,
		})
		cursor += size
	}

	assetOffsetByName := make(map[string]uint32, len(assets))
	sectionCursor := make(map[string]uint32, len(sectionOrder))
	for _, name := range sectionOrder {
		sectionCursor[name] = sectionStart[name]
	}
	for _, a := range assets {
		assetOffsetByName[a.Name] = sectionCursor[a.Section]
		sectionCursor[a.Section] += uint32(len(a.Data))
	}

	normalizedByName := make(map[string]AssetIR, len(assets))
	for _, a := range assets {
		normalizedByName[a.Name] = a
	}
	for _, a := range program.Assets {
		if norm, ok := normalizedByName[a.Name]; ok {
			offset := assetOffsetByName[norm.Name]
			manifest.Assets = append(manifest.Assets, ManifestAssetRef{
				Name:       norm.Name,
				Kind:       norm.Kind,
				Section:    norm.Section,
				Offset:     offset,
				SizeBytes:  uint32(len(norm.Data)),
				SourceFile: sourcePath,
				Line:       a.Position.Line,
				Column:     a.Position.Column,
			})
		} else {
			manifest.Assets = append(manifest.Assets, ManifestAssetRef{
				Name:       a.Name,
				Kind:       a.Type,
				Section:    "unpacked_asset_decl",
				Offset:     0,
				SizeBytes:  0,
				SourceFile: sourcePath,
				Line:       a.Position.Line,
				Column:     a.Position.Column,
			})
		}
	}

	plannedSize := uint32(0)
	for _, s := range manifest.Sections {
		end := s.Offset + s.SizeBytes
		if end > plannedSize {
			plannedSize = end
		}
	}
	if plannedSize < manifest.EmittedROMSizeBytes {
		plannedSize = manifest.EmittedROMSizeBytes
	}
	manifest.PlannedROMSizeBytes = plannedSize

	return manifest
}
