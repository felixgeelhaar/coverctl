package annotations

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScannerDetectsAnnotations(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "main.go")
	content := `package main

// coverctl:domain=core
// coverctl:ignore
func main() {}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, err := (Scanner{}).Scan(context.Background(), tmp, []string{"main.go"})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	ann, ok := out["main.go"]
	if !ok {
		t.Fatalf("expected annotation entry")
	}
	if !ann.Ignore {
		t.Fatalf("expected ignore true")
	}
	if ann.Domain != "core" {
		t.Fatalf("expected domain core, got %s", ann.Domain)
	}
}

func TestScannerIgnoresMissingFile(t *testing.T) {
	tmp := t.TempDir()
	if _, err := (Scanner{}).Scan(context.Background(), tmp, []string{"missing.go"}); err != nil {
		t.Fatalf("expected missing file to be ignored: %v", err)
	}
}
