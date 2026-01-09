package domain

import (
	"testing"
)

// testPathNormalizer is a simple path normalizer for testing.
type testPathNormalizer struct {
	moduleRoot string
}

func (n *testPathNormalizer) NormalizePath(file string) string {
	return file
}

func (n *testPathNormalizer) ToRelativePath(normalized string) string {
	return normalized
}

func TestCoverageAggregator(t *testing.T) {
	t.Run("NewCoverageAggregator creates aggregator", func(t *testing.T) {
		normalizer := &testPathNormalizer{}
		agg := NewCoverageAggregator(normalizer)

		if agg == nil {
			t.Fatal("Expected non-nil aggregator")
		}
		if agg.PathNormalizer != normalizer {
			t.Error("PathNormalizer not set correctly")
		}
	})

	t.Run("Aggregate aggregates by domain directories", func(t *testing.T) {
		normalizer := &testPathNormalizer{}
		agg := NewCoverageAggregator(normalizer)

		input := AggregationInput{
			FileCoverage: map[string]CoverageStat{
				"internal/core/service.go": {Covered: 80, Total: 100},
				"internal/core/repo.go":    {Covered: 60, Total: 100},
				"internal/api/handler.go":  {Covered: 50, Total: 100},
			},
			DomainDirs: map[string][]string{
				"core": {"internal/core"},
				"api":  {"internal/api"},
			},
		}

		result := agg.Aggregate(input)

		if len(result) != 2 {
			t.Fatalf("Expected 2 domains, got %d", len(result))
		}

		coreStat := result["core"]
		if coreStat.Covered != 140 {
			t.Errorf("Expected core covered 140, got %d", coreStat.Covered)
		}
		if coreStat.Total != 200 {
			t.Errorf("Expected core total 200, got %d", coreStat.Total)
		}

		apiStat := result["api"]
		if apiStat.Covered != 50 {
			t.Errorf("Expected api covered 50, got %d", apiStat.Covered)
		}
	})

	t.Run("Aggregate respects global excludes", func(t *testing.T) {
		normalizer := &testPathNormalizer{}
		agg := NewCoverageAggregator(normalizer)

		// Using exact filename match for exclusion pattern
		// Note: filepath.Match checks the full relPath against the pattern
		input := AggregationInput{
			FileCoverage: map[string]CoverageStat{
				"core/service.go":      {Covered: 80, Total: 100},
				"core/service_test.go": {Covered: 60, Total: 100},
			},
			DomainDirs: map[string][]string{
				"core": {"core"},
			},
			// Use exact filename patterns that match the relative paths
			GlobalExcludes: []string{"core/service_test.go"},
		}

		result := agg.Aggregate(input)

		coreStat := result["core"]
		if coreStat.Total != 100 {
			t.Errorf("Expected core total 100 (excluding test), got %d", coreStat.Total)
		}
	})

	t.Run("Aggregate respects domain-specific excludes", func(t *testing.T) {
		normalizer := &testPathNormalizer{}
		agg := NewCoverageAggregator(normalizer)

		// Use paths with directory structure for proper matching
		input := AggregationInput{
			FileCoverage: map[string]CoverageStat{
				"core/service.go": {Covered: 80, Total: 100},
				"core/mock.go":    {Covered: 0, Total: 50},
			},
			DomainDirs: map[string][]string{
				"core": {"core"},
			},
			DomainExcludes: map[string][]string{
				"core": {"core/mock.go"},
			},
		}

		result := agg.Aggregate(input)

		coreStat := result["core"]
		if coreStat.Total != 100 {
			t.Errorf("Expected core total 100 (excluding mock), got %d", coreStat.Total)
		}
	})

	t.Run("Aggregate respects annotations", func(t *testing.T) {
		normalizer := &testPathNormalizer{}
		agg := NewCoverageAggregator(normalizer)

		input := AggregationInput{
			FileCoverage: map[string]CoverageStat{
				"internal/core/service.go": {Covered: 80, Total: 100},
				"internal/core/legacy.go":  {Covered: 10, Total: 100},
				"internal/util/helper.go":  {Covered: 50, Total: 50},
			},
			DomainDirs: map[string][]string{
				"core": {"internal/core"},
				"util": {"internal/util"},
			},
			Annotations: map[string]FileAnnotation{
				"internal/core/legacy.go": {Ignore: true},
				"internal/util/helper.go": {Domain: "core"}, // Override to core
			},
		}

		result := agg.Aggregate(input)

		coreStat := result["core"]
		// Should include service.go (80/100) + helper.go (50/50) = 130/150
		if coreStat.Covered != 130 {
			t.Errorf("Expected core covered 130, got %d", coreStat.Covered)
		}
		if coreStat.Total != 150 {
			t.Errorf("Expected core total 150, got %d", coreStat.Total)
		}

		// util should be empty since helper.go was reassigned
		utilStat := result["util"]
		if utilStat.Total != 0 {
			t.Errorf("Expected util total 0, got %d", utilStat.Total)
		}
	})
}

func TestCoverageAggregator_isExcluded(t *testing.T) {
	agg := NewCoverageAggregator(&testPathNormalizer{})

	tests := []struct {
		file     string
		patterns []string
		expected bool
	}{
		{"service_test.go", []string{"*_test.go"}, true},
		{"service.go", []string{"*_test.go"}, false},
		{"mock.go", []string{"mock.go", "stub.go"}, true},
		{"service.go", []string{}, false},
		// Note: filepath.Match doesn't support multi-level matching
		// vendor/* matches vendor/foo but not vendor/pkg/lib.go
		{"vendor/foo", []string{"vendor/*"}, true},
		{"vendor/pkg/lib.go", []string{"vendor/*"}, false}, // Limited by filepath.Match
	}

	for _, tc := range tests {
		result := agg.isExcluded(tc.file, tc.patterns)
		if result != tc.expected {
			t.Errorf("isExcluded(%q, %v) = %v, expected %v", tc.file, tc.patterns, result, tc.expected)
		}
	}
}

func TestCoverageAggregator_matchesAnyDir(t *testing.T) {
	agg := NewCoverageAggregator(&testPathNormalizer{})

	tests := []struct {
		file     string
		dirs     []string
		expected bool
	}{
		{"internal/core/service.go", []string{"internal/core"}, true},
		{"internal/core/sub/deep.go", []string{"internal/core"}, true},
		{"internal/api/handler.go", []string{"internal/core"}, false},
		{"internal/core/service.go", []string{"internal/core", "internal/api"}, true},
		{"external/lib.go", []string{"internal/core", "internal/api"}, false},
	}

	for _, tc := range tests {
		result := agg.matchesAnyDir(tc.file, tc.dirs)
		if result != tc.expected {
			t.Errorf("matchesAnyDir(%q, %v) = %v, expected %v", tc.file, tc.dirs, result, tc.expected)
		}
	}
}

func TestClassifyFiles(t *testing.T) {
	normalizer := &testPathNormalizer{}
	agg := NewCoverageAggregator(normalizer)

	// Using base filenames for pattern matching to work correctly
	input := AggregationInput{
		FileCoverage: map[string]CoverageStat{
			"service.go":      {Covered: 80, Total: 100},
			"service_test.go": {Covered: 0, Total: 50},
			"handler.go":      {Covered: 50, Total: 100},
			"external.go":     {Covered: 0, Total: 20},
		},
		DomainDirs: map[string][]string{
			"core": {"service.go", "service_test.go"},
			"api":  {"handler.go"},
		},
		GlobalExcludes: []string{"*_test.go"},
	}

	classifications := agg.ClassifyFiles(input)

	if len(classifications) != 4 {
		t.Fatalf("Expected 4 classifications, got %d", len(classifications))
	}

	// Check for excluded test file
	foundExcluded := false
	foundNoMatch := false
	for _, c := range classifications {
		if c.File == "service_test.go" && c.Excluded {
			foundExcluded = true
		}
		if c.File == "external.go" && c.Reason == "no domain match" {
			foundNoMatch = true
		}
	}

	if !foundExcluded {
		t.Error("Expected test file to be classified as excluded")
	}
	if !foundNoMatch {
		t.Error("Expected external file to be classified as no domain match")
	}
}

func TestDefaultPathNormalizer(t *testing.T) {
	t.Run("NormalizePath handles absolute paths", func(t *testing.T) {
		n := &DefaultPathNormalizer{
			ModuleRoot: "/project",
			ModulePath: "github.com/example/project",
		}

		result := n.NormalizePath("/absolute/path/file.go")
		if result != "/absolute/path/file.go" {
			t.Errorf("Expected absolute path unchanged, got %s", result)
		}
	})

	t.Run("NormalizePath handles module paths", func(t *testing.T) {
		n := &DefaultPathNormalizer{
			ModuleRoot: "/project",
			ModulePath: "github.com/example/project",
		}

		result := n.NormalizePath("github.com/example/project/internal/pkg.go")
		expected := "/project/internal/pkg.go"
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})

	t.Run("NormalizePath handles exact module path", func(t *testing.T) {
		n := &DefaultPathNormalizer{
			ModuleRoot: "/project",
			ModulePath: "github.com/example/project",
		}

		result := n.NormalizePath("github.com/example/project")
		if result != "/project" {
			t.Errorf("Expected /project, got %s", result)
		}
	})

	t.Run("NormalizePath handles relative paths with module root", func(t *testing.T) {
		n := &DefaultPathNormalizer{
			ModuleRoot: "/project",
			ModulePath: "",
		}

		result := n.NormalizePath("internal/pkg.go")
		expected := "/project/internal/pkg.go"
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})

	t.Run("ToRelativePath converts to relative", func(t *testing.T) {
		n := &DefaultPathNormalizer{
			ModuleRoot: "/project",
			ModulePath: "github.com/example/project",
		}

		result := n.ToRelativePath("/project/internal/pkg.go")
		if result != "internal/pkg.go" {
			t.Errorf("Expected internal/pkg.go, got %s", result)
		}
	})

	t.Run("ToRelativePath handles empty module root", func(t *testing.T) {
		n := &DefaultPathNormalizer{
			ModuleRoot: "",
			ModulePath: "",
		}

		result := n.ToRelativePath("/some/path.go")
		if result != "/some/path.go" {
			t.Errorf("Expected unchanged path, got %s", result)
		}
	})
}
