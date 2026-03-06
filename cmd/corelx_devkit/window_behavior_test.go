package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Guardrail test: window hint helper should only be called from main startup.
func TestX11MaximizeHintCallsitesRestricted(t *testing.T) {
	files, err := filepath.Glob("./*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, path := range files {
		base := filepath.Base(path)
		if strings.HasPrefix(base, "window_x11_maximize") {
			continue
		}
		if base == "window_behavior_test.go" {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", base, err)
		}
		if strings.Contains(string(b), "applyX11MaximizeHint(") {
			if base != "main.go" {
				t.Fatalf("unexpected applyX11MaximizeHint callsite in %s", base)
			}
		}
	}
}
