package corelx

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultProjectAssetManifest = "corelx.assets.json"

type ProjectAssetManifest struct {
	FormatVersion int                  `json:"format_version,omitempty"`
	Assets        []ProjectAssetRecord `json:"assets"`
}

type ProjectAssetRecord struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Encoding string `json:"encoding"`
	Path     string `json:"path,omitempty"`
	Data     string `json:"data,omitempty"`
}

func loadProjectAssets(sourcePath string, cfg CompileOptions) ([]*AssetDecl, []string, []Diagnostic) {
	manifestPath := strings.TrimSpace(cfg.AssetManifestPath)
	if manifestPath == "" {
		if sourcePath == "" {
			return nil, nil, nil
		}
		manifestPath = filepath.Join(filepath.Dir(sourcePath), defaultProjectAssetManifest)
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, []Diagnostic{{
			Category: CategoryIOError,
			Code:     "E_IO_READ_ASSET_MANIFEST",
			Message:  err.Error(),
			File:     manifestPath,
			Severity: SeverityError,
			Stage:    StageIO,
		}}
	}

	var m ProjectAssetManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, []string{manifestPath}, []Diagnostic{{
			Category: CategoryAssetParseError,
			Code:     "E_ASSET_MANIFEST_PARSE",
			Message:  err.Error(),
			File:     manifestPath,
			Severity: SeverityError,
			Stage:    StageAsset,
		}}
	}

	assets := make([]*AssetDecl, 0, len(m.Assets))
	sourceFiles := []string{manifestPath}
	for i, rec := range m.Assets {
		name := strings.TrimSpace(rec.Name)
		kind := strings.TrimSpace(rec.Type)
		enc := strings.TrimSpace(rec.Encoding)
		if name == "" || kind == "" || enc == "" {
			return nil, sourceFiles, []Diagnostic{{
				Category: CategoryAssetFormatError,
				Code:     "E_ASSET_MANIFEST_RECORD",
				Message:  fmt.Sprintf("asset record %d missing required name/type/encoding", i),
				File:     manifestPath,
				Line:     i + 1,
				Column:   1,
				Severity: SeverityError,
				Stage:    StageAsset,
			}}
		}
		if !isValidAssetType(kind) {
			return nil, sourceFiles, []Diagnostic{{
				Category: CategoryAssetFormatError,
				Code:     "E_ASSET_MANIFEST_TYPE",
				Message:  fmt.Sprintf("asset record %d has invalid type %q", i, kind),
				File:     manifestPath,
				Line:     i + 1,
				Column:   1,
				Severity: SeverityError,
				Stage:    StageAsset,
			}}
		}
		if !isValidAssetEncoding(enc) {
			return nil, sourceFiles, []Diagnostic{{
				Category: CategoryAssetFormatError,
				Code:     "E_ASSET_MANIFEST_ENCODING",
				Message:  fmt.Sprintf("asset record %d has invalid encoding %q", i, enc),
				File:     manifestPath,
				Line:     i + 1,
				Column:   1,
				Severity: SeverityError,
				Stage:    StageAsset,
			}}
		}

		payload := rec.Data
		if strings.TrimSpace(rec.Path) != "" {
			p := rec.Path
			if !filepath.IsAbs(p) {
				p = filepath.Join(filepath.Dir(manifestPath), p)
			}
			b, readErr := os.ReadFile(p)
			if readErr != nil {
				return nil, sourceFiles, []Diagnostic{{
					Category: CategoryIOError,
					Code:     "E_IO_READ_ASSET_FILE",
					Message:  readErr.Error(),
					File:     p,
					Severity: SeverityError,
					Stage:    StageIO,
				}}
			}
			payload = string(b)
			sourceFiles = append(sourceFiles, p)
		}

		assets = append(assets, &AssetDecl{
			Position: Position{Line: i + 1, Column: 1},
			Name:     name,
			Type:     kind,
			Encoding: enc,
			Data:     payload,
		})
	}
	return assets, uniqueStrings(sourceFiles), nil
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, v := range values {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
