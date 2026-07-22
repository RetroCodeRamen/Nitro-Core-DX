package corelx

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

type AssetIR struct {
	Name       string
	Kind       string
	Section    string
	Encoding   string
	Data       []byte
	SourceFile string
	Position   Position
}

func NormalizeAssets(program *Program, sourcePath string) ([]AssetIR, []Diagnostic) {
	assets := make([]AssetIR, 0, len(program.Assets))
	diags := make([]Diagnostic, 0)

	for _, a := range program.Assets {
		// Image and music assets are external files (.cxasset / .ncdxmusic)
		// handled by loadImageAssets / loadMusicAssets, not inline-normalized here.
		if a.Type == "image" || a.Type == "music" {
			continue
		}
		ir, errDiag := normalizeAssetDecl(a, sourcePath)
		if errDiag != nil {
			diags = append(diags, *errDiag)
			continue
		}
		assets = append(assets, ir)
	}

	return assets, diags
}

func normalizeAssetDecl(a *AssetDecl, sourcePath string) (AssetIR, *Diagnostic) {
	ir := AssetIR{
		Name:       a.Name,
		Kind:       a.Type,
		Section:    sectionForAssetType(a.Type),
		Encoding:   a.Encoding,
		SourceFile: sourcePath,
		Position:   a.Position,
	}

	if sectionForAssetType(a.Type) == "unknown" {
		d := assetDiagnostic(a, sourcePath, CategoryAssetFormatError, "E_ASSET_TYPE_UNSUPPORTED", fmt.Sprintf("unsupported asset type for normalization: %s", a.Type))
		return AssetIR{}, &d
	}

	switch a.Encoding {
	case "hex":
		data, err := decodeHexAssetData(a.Data)
		if err != nil {
			d := assetDiagnostic(a, sourcePath, CategoryAssetParseError, "E_ASSET_HEX_PARSE", err.Error())
			return AssetIR{}, &d
		}
		ir.Data = data
	case "b64":
		data, err := decodeBase64AssetData(a.Data)
		if err != nil {
			d := assetDiagnostic(a, sourcePath, CategoryAssetParseError, "E_ASSET_B64_PARSE", err.Error())
			return AssetIR{}, &d
		}
		ir.Data = data
	case "text":
		ir.Data = []byte(a.Data)
	default:
		d := assetDiagnostic(a, sourcePath, CategoryAssetFormatError, "E_ASSET_ENCODING_UNSUPPORTED", fmt.Sprintf("unsupported asset encoding: %s", a.Encoding))
		return AssetIR{}, &d
	}
	ir.Data = normalizeLegacyTilePayload(a.Type, ir.Data)

	return ir, nil
}

func normalizeLegacyTilePayload(assetType string, data []byte) []byte {
	switch assetType {
	case "tiles8":
		// Accept historical "one byte per pixel index" authoring for 8x8 tiles
		// and pack it into the runtime 4bpp format.
		if len(data) == 64 {
			return packExpanded4bppPixels(data)
		}
	case "tiles16":
		// Accept historical "one byte per pixel index" authoring for 16x16 tiles
		// and pack it into the runtime 4bpp format.
		if len(data) == 256 {
			return packExpanded4bppPixels(data)
		}
	case "tileset", "sprite":
		// Unlike tiles8/tiles16 (always exactly one tile, so raw-vs-packed can
		// be told apart by a fixed length), a tileset/sprite asset can hold any
		// number of tiles, so length alone is ambiguous -- 512 bytes is valid
		// both as 16 already-packed tiles and as 8 raw one-byte-per-pixel
		// tiles. Instead, detect raw "one byte per pixel index" authoring by
		// value range: every nibble-packed byte with a nonzero high nibble
		// (i.e. >0x0F) can only occur in already-packed data, since a raw
		// index-per-byte tile (this project's convention for every other hand
		// -authored tile asset, e.g. TreeA/Creature/PlayerTop/PlayerBottom)
		// never uses palette indices above 15 as a bare byte. If every byte
		// is <=0x0F, treat it as raw and pack it; otherwise assume it's
		// already in packed 4bpp form (e.g. Games/SpriteProbe's FourTiles,
		// authored directly as packed bytes like 0x11/0x22/0x33/0x44).
		if len(data) > 0 && len(data)%2 == 0 {
			allLowNibble := true
			for _, b := range data {
				if b > 0x0F {
					allLowNibble = false
					break
				}
			}
			if allLowNibble {
				return packExpanded4bppPixels(data)
			}
		}
	}
	return data
}

func packExpanded4bppPixels(data []byte) []byte {
	if len(data)%2 != 0 {
		return data
	}
	out := make([]byte, 0, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		hi := data[i] & 0x0F
		lo := data[i+1] & 0x0F
		out = append(out, (hi<<4)|lo)
	}
	return out
}

func decodeHexAssetData(s string) ([]byte, error) {
	fields := strings.Fields(s)
	out := make([]byte, 0, len(fields))
	for _, tok := range fields {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if strings.HasPrefix(tok, "0x") || strings.HasPrefix(tok, "0X") {
			tok = tok[2:]
		}
		if len(tok)%2 != 0 {
			return nil, fmt.Errorf("invalid hex byte %q", tok)
		}
		for i := 0; i < len(tok); i += 2 {
			part := tok[i : i+2]
			v, err := strconv.ParseUint(part, 16, 8)
			if err != nil {
				return nil, fmt.Errorf("invalid hex byte %q", part)
			}
			out = append(out, byte(v))
		}
	}
	return out, nil
}

func decodeBase64AssetData(s string) ([]byte, error) {
	compact := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r':
			return -1
		default:
			return r
		}
	}, s)

	data, err := base64.StdEncoding.DecodeString(compact)
	if err == nil {
		return data, nil
	}
	// Allow raw base64 without padding for convenience.
	data, rawErr := base64.RawStdEncoding.DecodeString(compact)
	if rawErr == nil {
		return data, nil
	}
	return nil, err
}

func sectionForAssetType(assetType string) string {
	switch assetType {
	case "tiles8", "tiles16", "sprite", "tileset":
		return "gfx_tiles"
	case "tilemap":
		return "tilemaps"
	case "palette":
		return "palettes"
	case "music", "sfx", "ambience":
		return "audio_seq"
	case "gamedata", "blob":
		return "gamedata"
	default:
		return "unknown"
	}
}

func assetDiagnostic(a *AssetDecl, sourcePath string, cat DiagnosticCategory, code, msg string) Diagnostic {
	return Diagnostic{
		Category: cat,
		Code:     code,
		Message:  msg,
		File:     sourcePath,
		Line:     a.Position.Line,
		Column:   a.Position.Column,
		Severity: SeverityError,
		Stage:    StageAsset,
	}
}
