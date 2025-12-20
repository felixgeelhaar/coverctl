package diff

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/coverctl/internal/infrastructure/gotool"
)

func TestGitDiffChangedFiles(t *testing.T) {
	diff := GitDiff{
		Module: gotool.ModuleResolver{},
		Exec: func(ctx context.Context, dir string, args []string) ([]byte, error) {
			return []byte("internal/core/a.go\ninternal/api/b.go\n"), nil
		},
	}
	files, err := diff.ChangedFiles(context.Background(), "main")
	if err != nil {
		t.Fatalf("changed files: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0] != "internal/core/a.go" {
		t.Fatalf("unexpected first file: %s", files[0])
	}
}
