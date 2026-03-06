package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nitro-core-dx/internal/corelx"
)

const devKitProjectAssetManifestName = "corelx.assets.json"

func projectAssetManifestPathForSource(sourcePath string) (string, error) {
	p := strings.TrimSpace(sourcePath)
	if p == "" {
		return "", fmt.Errorf("save the project source file before applying assets to manifest")
	}
	return filepath.Join(filepath.Dir(p), devKitProjectAssetManifestName), nil
}

func upsertProjectAssetManifestRecord(sourcePath string, rec corelx.ProjectAssetRecord) (string, string, error) {
	manifestPath, err := projectAssetManifestPathForSource(sourcePath)
	if err != nil {
		return "", "", err
	}

	manifest, err := loadOrInitProjectAssetManifest(manifestPath)
	if err != nil {
		return "", "", err
	}

	name := strings.TrimSpace(rec.Name)
	kind := strings.TrimSpace(rec.Type)
	enc := strings.TrimSpace(rec.Encoding)
	if name == "" || kind == "" || enc == "" {
		return "", "", fmt.Errorf("asset record requires name/type/encoding")
	}

	rec.Name = name
	rec.Type = kind
	rec.Encoding = enc
	rec.Path = strings.TrimSpace(rec.Path)
	rec.Data = strings.TrimSpace(rec.Data)

	state := "new"
	updated := false
	for i := range manifest.Assets {
		if strings.EqualFold(strings.TrimSpace(manifest.Assets[i].Name), name) {
			manifest.Assets[i] = rec
			state = "updated"
			updated = true
			break
		}
	}
	if !updated {
		manifest.Assets = append(manifest.Assets, rec)
	}
	if manifest.FormatVersion == 0 {
		manifest.FormatVersion = 1
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", "", err
	}
	data = append(data, '\n')
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return "", "", err
	}

	return manifestPath, state, nil
}

func loadOrInitProjectAssetManifest(path string) (corelx.ProjectAssetManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return corelx.ProjectAssetManifest{
				FormatVersion: 1,
				Assets:        make([]corelx.ProjectAssetRecord, 0, 8),
			}, nil
		}
		return corelx.ProjectAssetManifest{}, err
	}

	var m corelx.ProjectAssetManifest
	if len(strings.TrimSpace(string(data))) == 0 {
		m.FormatVersion = 1
		m.Assets = make([]corelx.ProjectAssetRecord, 0, 8)
		return m, nil
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return corelx.ProjectAssetManifest{}, fmt.Errorf("invalid %s: %w", devKitProjectAssetManifestName, err)
	}
	if m.FormatVersion == 0 {
		m.FormatVersion = 1
	}
	if m.Assets == nil {
		m.Assets = make([]corelx.ProjectAssetRecord, 0, 8)
	}
	return m, nil
}

func bytesToHexFields(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, b := range data {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(fmt.Sprintf("%02X", b))
	}
	return sb.String()
}
