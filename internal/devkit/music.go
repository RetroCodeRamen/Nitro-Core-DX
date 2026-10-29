package devkit

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"nitro-core-dx/internal/ymstream"
)

// MusicSnippet is one CoreLX playback call the Sound Studio can insert.
type MusicSnippet struct {
	Label string
	Code  string
}

// MusicApplyResult is a music file written beside a saved source file and the
// source text with a matching asset declaration.
type MusicApplyResult struct {
	Source   string
	FilePath string
	FileName string
	Name     string
	State    string
}

var musicAssetHeader = regexp.MustCompile(`^[ \t]*asset[ \t]+([A-Za-z_][A-Za-z0-9_]*)[ \t]*:[ \t]*([A-Za-z_][A-Za-z0-9_]*)\b`)

// SanitizeMusicAssetName turns a file stem or typed name into a CoreLX identifier.
func SanitizeMusicAssetName(raw string) (string, error) {
	var b strings.Builder
	raw = strings.TrimSpace(raw)
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			b.WriteByte(ch)
			continue
		}
		b.WriteByte('_')
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "", fmt.Errorf("music asset name is empty")
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = "A_" + out
	}
	return out, nil
}

// MusicFileName is the project-relative .ncdxmusic name for an asset.
func MusicFileName(name string) string {
	return name + ".ncdxmusic"
}

// MusicAssetLine is the one-line CoreLX declaration for an external music file.
func MusicAssetLine(name, fileName string) string {
	return fmt.Sprintf("asset %s: music \"%s\"", name, fileName)
}

// MusicSnippets returns the six playback calls for a declared music asset.
func MusicSnippets(name string) []MusicSnippet {
	return []MusicSnippet{
		{Label: "Play once", Code: fmt.Sprintf("music.play(%s)", name)},
		{Label: "Play loop", Code: fmt.Sprintf("music.play_loop(%s)", name)},
		{Label: "Play jingle", Code: fmt.Sprintf("music.play_jingle(%s)", name)},
		{Label: "Stop", Code: "music.stop()"},
		{Label: "Set volume", Code: "music.set_volume(255)"},
		{Label: "Fade out", Code: "music.fade_to(0, 60)"},
	}
}

// MusicPreviewSource is the throwaway program that loops a music asset.
func MusicPreviewSource(name, fileName string) string {
	return fmt.Sprintf("asset %s: music \"%s\"\n\nfunction Start()\n    music.play_loop(%s)\n    while true\n        wait_vblank()\n", name, fileName, name)
}

// UpsertMusicAssetLine inserts or replaces the music declaration for name.
// A same-named asset of another type is an error. State is "new" or "updated".
func UpsertMusicAssetLine(source, name, fileName string) (string, string, error) {
	sanitized, err := SanitizeMusicAssetName(name)
	if err != nil || sanitized != name {
		return "", "", fmt.Errorf("music asset name %q is not a CoreLX identifier", name)
	}
	line := MusicAssetLine(name, fileName)
	lines := strings.Split(source, "\n")
	found := -1
	lastAsset := -1
	for i, ln := range lines {
		m := musicAssetHeader.FindStringSubmatch(ln)
		if m == nil {
			continue
		}
		if m[1] == name {
			if m[2] != "music" {
				return "", "", fmt.Errorf("asset %s already exists as %s", name, m[2])
			}
			found = i
		}
		lastAsset = i
	}
	if found >= 0 {
		lines[found] = line
		return strings.Join(lines, "\n"), "updated", nil
	}
	if lastAsset >= 0 {
		out := make([]string, 0, len(lines)+1)
		out = append(out, lines[:lastAsset+1]...)
		out = append(out, line)
		out = append(out, lines[lastAsset+1:]...)
		return strings.Join(out, "\n"), "new", nil
	}
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return line + "\n", "new", nil
	}
	body := source
	if !strings.HasPrefix(body, "\n") {
		body = line + "\n" + body
	} else {
		body = line + body
	}
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return body, "new", nil
}

// ApplyMusicAsset writes a .ncdxmusic file next to a saved source file and
// returns the source with the asset declaration upserted. The file write and
// the declaration describe the same name, so the next build does not see an
// orphan music file.
func ApplyMusicAsset(sourcePath, source, rawName string, data []byte) (MusicApplyResult, error) {
	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		return MusicApplyResult{}, fmt.Errorf("save the project source file before adding a music asset")
	}
	if len(data) == 0 {
		return MusicApplyResult{}, fmt.Errorf("music asset is empty")
	}
	if _, err := ymstream.DecodeStream(data); err != nil {
		return MusicApplyResult{}, fmt.Errorf("music asset is not a .ncdxmusic stream: %w", err)
	}
	name, err := SanitizeMusicAssetName(rawName)
	if err != nil {
		return MusicApplyResult{}, err
	}
	fileName := MusicFileName(name)
	next, state, err := UpsertMusicAssetLine(source, name, fileName)
	if err != nil {
		return MusicApplyResult{}, err
	}
	dir := filepath.Dir(sourcePath)
	dest := filepath.Join(dir, fileName)
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return MusicApplyResult{}, err
	}
	return MusicApplyResult{
		Source:   next,
		FilePath: dest,
		FileName: fileName,
		Name:     name,
		State:    state,
	}, nil
}

// BuildMusicPreview compiles a temporary project that loops the given
// .ncdxmusic bytes. The editor's source is not used and is not modified.
func (s *Service) BuildMusicPreview(data []byte, rawName string) (*BuildResult, error) {
	if s == nil {
		return nil, fmt.Errorf("no dev kit service")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("music asset is empty")
	}
	if _, err := ymstream.DecodeStream(data); err != nil {
		return nil, fmt.Errorf("music asset is not a .ncdxmusic stream: %w", err)
	}
	name, err := SanitizeMusicAssetName(rawName)
	if err != nil {
		return nil, err
	}
	fileName := MusicFileName(name)
	dir := filepath.Join(s.tempDir, "sound-preview")
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, fileName), data, 0o644); err != nil {
		return nil, err
	}
	sourcePath := filepath.Join(dir, "preview.corelx")
	return s.BuildSource(MusicPreviewSource(name, fileName), sourcePath)
}
