package application

import (
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/coverctl/internal/domain"
)

func TestAggregateByDomain(t *testing.T) {
	files := map[string]domain.CoverageStat{
		"internal/core/a.go": {Covered: 1, Total: 2},
		"internal/api/b.go":  {Covered: 2, Total: 2},
		"internal/gen/c.go":  {Covered: 1, Total: 1},
	}
	moduleRoot := "/repo"
	domainDirs := map[string][]string{
		"core": {filepath.Join(moduleRoot, "internal/core")},
		"api":  {filepath.Join(moduleRoot, "internal/api")},
	}
	exclude := []string{"internal/gen/*"}

	modulePath := "github.com/felixgeelhaar/coverctl"
	result := AggregateByDomain(files, domainDirs, exclude, moduleRoot, modulePath, nil)
	if got := result["core"]; got.Covered != 1 || got.Total != 2 {
		t.Fatalf("unexpected core coverage: %+v", got)
	}
	if got := result["api"]; got.Covered != 2 || got.Total != 2 {
		t.Fatalf("unexpected api coverage: %+v", got)
	}
}

func TestExcludeNoMatch(t *testing.T) {
	if excluded("internal/core/a.go", []string{"internal/gen/*"}) {
		t.Fatalf("did not expect exclusion")
	}
}

func TestMatchesAnyDirFalse(t *testing.T) {
	if matchesAnyDir("internal/core/a.go", []string{"/repo/internal/api"}, "/repo") {
		t.Fatalf("expected no match")
	}
}

func TestMatchesAnyDirModuleRoot(t *testing.T) {
	if !matchesAnyDir("/repo/main.go", []string{"/repo"}, "/repo") {
		t.Fatalf("expected match for module root")
	}
}

func TestNormalizeCoverageFileNoModulePath(t *testing.T) {
	path := normalizeCoverageFile("internal/api/handler.go", "", "/repo")
	if path != filepath.Join("/repo", "internal", "api", "handler.go") {
		t.Fatalf("unexpected normalized path: %s", path)
	}
}

func TestModuleRelativePathNoRoot(t *testing.T) {
	if got := moduleRelativePath("/repo/main.go", ""); got != filepath.Clean("/repo/main.go") {
		t.Fatalf("expected clean path, got %s", got)
	}
}

func TestAggregateWithModulePath(t *testing.T) {
	moduleRoot := "/repo"
	modulePath := "github.com/felixgeelhaar/coverctl"
	files := map[string]domain.CoverageStat{
		"github.com/felixgeelhaar/coverctl/cmd/coverctl/main.go": {Covered: 8, Total: 10},
	}
	domainDirs := map[string][]string{
		"cmd": {filepath.Join(moduleRoot, "cmd/coverctl")},
	}
	result := AggregateByDomain(files, domainDirs, nil, moduleRoot, modulePath, nil)
	if got := result["cmd"]; got.Total != 10 || got.Covered != 8 {
		t.Fatalf("expected cmd to aggregate coverage, got %+v", got)
	}
}

func TestAggregateByDomainAnnotations(t *testing.T) {
	files := map[string]domain.CoverageStat{
		"internal/core/a.go": {Covered: 1, Total: 2},
		"internal/skip/b.go": {Covered: 2, Total: 2},
	}
	moduleRoot := "/repo"
	domainDirs := map[string][]string{
		"core": {filepath.Join(moduleRoot, "internal/core")},
	}
	annotations := map[string]Annotation{
		"internal/core/a.go": {Domain: "core"},
		"internal/skip/b.go": {Ignore: true},
	}
	result := AggregateByDomain(files, domainDirs, nil, moduleRoot, "", annotations)
	if got := result["core"]; got.Covered != 1 || got.Total != 2 {
		t.Fatalf("unexpected core coverage: %+v", got)
	}
	if _, ok := result["skip"]; ok {
		t.Fatalf("expected ignored file to be skipped")
	}
}
