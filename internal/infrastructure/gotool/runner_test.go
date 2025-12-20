package gotool

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
)

func TestBuildCoverPkg(t *testing.T) {
	min := 80.0
	domains := []domain.Domain{
		{Name: "core", Match: []string{"./internal/core/..."}, Min: &min},
		{Name: "api", Match: []string{"./internal/api/...", "./internal/core/..."}},
	}
	got := buildCoverPkg(domains)
	if got == "" {
		t.Fatalf("expected coverpkg")
	}
	parts := strings.Split(got, ",")
	if len(parts) != 2 {
		t.Fatalf("expected 2 unique patterns, got %d", len(parts))
	}
}

func TestModuleRoot(t *testing.T) {
	root, err := (ModuleResolver{}).ModuleRoot(context.Background())
	if err != nil {
		t.Fatalf("module root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("expected go.mod in module root: %v", err)
	}
}

func TestResolveDomains(t *testing.T) {
	resolver := DomainResolver{Module: ModuleResolver{}}
	result, err := resolver.Resolve(context.Background(), []domain.Domain{{
		Name:  "domain",
		Match: []string{"./internal/domain"},
	}})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(result["domain"]) == 0 {
		t.Fatalf("expected domain directories")
	}
}

func TestBuildCoverPkgEmpty(t *testing.T) {
	if got := buildCoverPkg(nil); got != "./..." {
		t.Fatalf("expected default coverpkg, got %s", got)
	}
}

func TestResolveDomainsError(t *testing.T) {
	resolver := DomainResolver{Module: ModuleResolver{}}
	_, err := resolver.Resolve(context.Background(), []domain.Domain{{
		Name:  "bad",
		Match: []string{"./does-not-exist"},
	}})
	if err == nil {
		t.Fatalf("expected error for invalid pattern")
	}
}

func TestRunnerRun(t *testing.T) {
	tmp := t.TempDir()
	profile := filepath.Join(tmp, "coverage.out")
	runner := Runner{
		Module: ModuleResolver{},
		Exec: func(ctx context.Context, dir string, args []string) error {
			for _, arg := range args {
				if strings.HasPrefix(arg, "-coverprofile=") {
					path := strings.TrimPrefix(arg, "-coverprofile=")
					return os.WriteFile(path, []byte("mode: atomic\n"), 0o644)
				}
			}
			return nil
		},
	}
	out, err := runner.Run(context.Background(), application.RunOptions{ProfilePath: profile})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if out != profile {
		t.Fatalf("expected profile path %s, got %s", profile, out)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected coverage file: %v", err)
	}
}

func TestUnique(t *testing.T) {
	values := []string{"a", "b", "a"}
	out := unique(values)
	if len(out) != 2 {
		t.Fatalf("expected 2 unique values, got %d", len(out))
	}
}
