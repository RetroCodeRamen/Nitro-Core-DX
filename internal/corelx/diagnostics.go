package corelx

import "fmt"

type DiagnosticSeverity string

const (
	SeverityError   DiagnosticSeverity = "error"
	SeverityWarning DiagnosticSeverity = "warning"
	SeverityInfo    DiagnosticSeverity = "info"
)

type DiagnosticStage string

const (
	StageIO       DiagnosticStage = "io"
	StageLexer    DiagnosticStage = "lexer"
	StageParser   DiagnosticStage = "parser"
	StageSemantic DiagnosticStage = "semantic"
	StageAsset    DiagnosticStage = "asset"
	StageCodegen  DiagnosticStage = "codegen"
	StagePack     DiagnosticStage = "pack"
)

type DiagnosticCategory string

const (
	CategoryLexError              DiagnosticCategory = "LexError"
	CategorySyntaxError           DiagnosticCategory = "SyntaxError"
	CategorySymbolError           DiagnosticCategory = "SymbolError"
	CategoryTypeError             DiagnosticCategory = "TypeError"
	CategoryValidationError       DiagnosticCategory = "ValidationError"
	CategoryAssetParseError       DiagnosticCategory = "AssetParseError"
	CategoryAssetFormatError      DiagnosticCategory = "AssetFormatError"
	CategoryAssetReferenceError   DiagnosticCategory = "AssetReferenceError"
	CategoryBackendCodegenError   DiagnosticCategory = "BackendCodegenError"
	CategoryLayoutError           DiagnosticCategory = "LayoutError"
	CategoryOverflowError         DiagnosticCategory = "OverflowError"
	CategoryInternalCompilerError DiagnosticCategory = "InternalCompilerError"
	CategoryIOError               DiagnosticCategory = "IOError"
)

type Diagnostic struct {
	Category  DiagnosticCategory
	Code      string
	Message   string
	File      string
	Line      int
	Column    int
	EndLine   int
	EndColumn int
	Severity  DiagnosticSeverity
	Stage     DiagnosticStage
	Notes     []string
	Related   []DiagnosticLocation
}

type DiagnosticLocation struct {
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
	Message string `json:"message,omitempty"`
}

func (d Diagnostic) Error() string {
	if d.File != "" && d.Line > 0 {
		return fmt.Sprintf("%s:%d:%d: %s", d.File, d.Line, d.Column, d.Message)
	}
	if d.Line > 0 {
		return fmt.Sprintf("line %d:%d: %s", d.Line, d.Column, d.Message)
	}
	return d.Message
}

type DiagnosticsError struct {
	Diagnostics []Diagnostic
}

func (e *DiagnosticsError) Error() string {
	if e == nil || len(e.Diagnostics) == 0 {
		return ""
	}
	return e.Diagnostics[0].Error()
}

func HasErrors(diags []Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}
