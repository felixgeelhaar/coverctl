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

	result := AggregateByDomain(files, domainDirs, exclude, moduleRoot)
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
