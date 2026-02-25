package corelx

type CompileBundle struct {
	SchemaVersion int            `json:"schema_version"`
	Success       bool           `json:"success"`
	Summary       CompileSummary `json:"summary"`
	Diagnostics   []Diagnostic   `json:"diagnostics"`
	Manifest      *BuildManifest `json:"manifest,omitempty"`
}

type CompileSummary struct {
	ErrorCount   int `json:"error_count"`
	WarningCount int `json:"warning_count"`
	InfoCount    int `json:"info_count"`
}

func BuildCompileBundle(result *CompileResult) CompileBundle {
	b := CompileBundle{
		SchemaVersion: 1,
		Manifest:      nil,
	}
	if result == nil {
		return b
	}
	b.Diagnostics = result.Diagnostics
	b.Manifest = result.Manifest
	b.Success = !HasErrors(result.Diagnostics)
	for _, d := range result.Diagnostics {
		switch d.Severity {
		case SeverityError:
			b.Summary.ErrorCount++
		case SeverityWarning:
			b.Summary.WarningCount++
		case SeverityInfo:
			b.Summary.InfoCount++
		}
	}
	return b
}
