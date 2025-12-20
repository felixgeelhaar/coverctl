package coverprofile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	content := "mode: atomic\n" +
		"internal/core/foo.go:1.2,3.4 2 1\n" +
		"internal/core/foo.go:5.6,7.8 3 0\n" +
		"internal/api/bar.go:1.2,3.4 1 1\n"

	tmp := t.TempDir()
	path := filepath.Join(tmp, "coverage.out")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	stats, err := (Parser{}).Parse(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := stats["internal/core/foo.go"]; got.Total != 5 || got.Covered != 2 {
		t.Fatalf("unexpected core stats: %+v", got)
	}
	if got := stats["internal/api/bar.go"]; got.Total != 1 || got.Covered != 1 {
		t.Fatalf("unexpected api stats: %+v", got)
	}
}

func TestParseInvalid(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "coverage.out")
	if err := os.WriteFile(path, []byte("oops\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := (Parser{}).Parse(path); err == nil {
		t.Fatalf("expected error")
	}
}
