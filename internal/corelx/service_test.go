package corelx

import (
	"path/filepath"
	"testing"
)

func TestServiceCompileBundleSourceSuccess(t *testing.T) {
	svc := NewService()
	src := `
function Start()
    apu.enable()
`
	bundle, res, err := svc.CompileBundleSource(src, "svc_success.corelx", nil)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
	if res == nil {
		t.Fatalf("expected compile result")
	}
	if !bundle.Success {
		t.Fatalf("expected successful bundle: %+v", bundle)
	}
	if bundle.Manifest == nil {
		t.Fatalf("expected manifest in bundle")
	}
}

func TestServiceCompileBundleFileError(t *testing.T) {
	svc := NewService()
	path := filepath.Join(t.TempDir(), "svc_error.corelx")
	src := "function Nope()\n    apu.enable()\n"

	bundle, res, err := svc.CompileBundleSource(src, path, nil)
	if err == nil {
		t.Fatalf("expected compile error")
	}
	if res == nil {
		t.Fatalf("expected compile result")
	}
	if bundle.Success {
		t.Fatalf("expected failed bundle")
	}
	if bundle.Summary.ErrorCount == 0 {
		t.Fatalf("expected non-zero error count in bundle summary")
	}
}
