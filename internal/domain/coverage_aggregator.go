package domain

import (
	"path/filepath"
	"strings"
)

// FileAnnotation represents coverage annotation for a file.
type FileAnnotation struct {
	Ignore bool
	Domain string
}

// AggregationInput contains the input data for coverage aggregation.
type AggregationInput struct {
	FileCoverage   map[string]CoverageStat
	DomainDirs     map[string][]string
	GlobalExcludes []string
	DomainExcludes map[string][]string
	Annotations    map[string]FileAnnotation
}

// CoverageAggregator is a domain service that aggregates file-level coverage
// into domain-level coverage based on directory mappings and exclusion rules.
type CoverageAggregator struct {
	// PathNormalizer normalizes file paths for comparison.
	// This is injected to allow infrastructure-specific path handling.
	PathNormalizer PathNormalizer
}

// PathNormalizer is a port that normalizes file paths.
// The actual implementation lives in the infrastructure layer.
type PathNormalizer interface {
	// NormalizePath converts a coverage file path to a normalized form.
	NormalizePath(file string) string
	// ToRelativePath converts a normalized path to a relative path.
	ToRelativePath(normalized string) string
}

// NewCoverageAggregator creates a new CoverageAggregator with the given path normalizer.
func NewCoverageAggregator(normalizer PathNormalizer) *CoverageAggregator {
	return &CoverageAggregator{
		PathNormalizer: normalizer,
	}
}

// Aggregate aggregates file-level coverage into domain-level coverage.
func (a *CoverageAggregator) Aggregate(input AggregationInput) map[string]CoverageStat {
	result := make(map[string]CoverageStat, len(input.DomainDirs))

	for file, stat := range input.FileCoverage {
		normalized := a.PathNormalizer.NormalizePath(file)
		relPath := a.PathNormalizer.ToRelativePath(normalized)

		// Check global exclusions
		if a.isExcluded(relPath, input.GlobalExcludes) {
			continue
		}

		// Check annotations
		if ann, ok := input.Annotations[filepath.ToSlash(relPath)]; ok {
			if ann.Ignore {
				continue
			}
			if ann.Domain != "" {
				a.addCoverage(result, ann.Domain, stat)
				continue
			}
		}

		// Match against domain directories
		for domainName, dirs := range input.DomainDirs {
			if a.matchesAnyDir(normalized, dirs) {
				// Check domain-specific excludes
				if input.DomainExcludes != nil {
					if patterns, ok := input.DomainExcludes[domainName]; ok {
						if a.isExcluded(relPath, patterns) {
							continue
						}
					}
				}
				a.addCoverage(result, domainName, stat)
			}
		}
	}

	return result
}

// addCoverage adds coverage statistics to a domain.
func (a *CoverageAggregator) addCoverage(result map[string]CoverageStat, domain string, stat CoverageStat) {
	agg := result[domain]
	agg.Covered += stat.Covered
	agg.Total += stat.Total
	result[domain] = agg
}

// isExcluded checks if a file matches any exclusion pattern.
func (a *CoverageAggregator) isExcluded(file string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	for _, pattern := range patterns {
		if ok, _ := filepath.Match(pattern, file); ok {
			return true
		}
	}
	return false
}

// matchesAnyDir checks if a file belongs to any of the given directories.
func (a *CoverageAggregator) matchesAnyDir(file string, dirs []string) bool {
	cleanFile := filepath.Clean(file)
	for _, dir := range dirs {
		cleanDir := filepath.Clean(dir)
		if strings.HasPrefix(cleanFile, cleanDir+string(filepath.Separator)) || cleanFile == cleanDir {
			return true
		}
	}
	return false
}

// DefaultPathNormalizer provides a simple path normalizer for testing and simple cases.
type DefaultPathNormalizer struct {
	ModuleRoot string
	ModulePath string
}

// NormalizePath normalizes a coverage file path.
func (n *DefaultPathNormalizer) NormalizePath(file string) string {
	clean := filepath.Clean(file)
	if filepath.IsAbs(clean) {
		return clean
	}
	if n.ModulePath != "" {
		if file == n.ModulePath {
			return filepath.Clean(n.ModuleRoot)
		}
		if strings.HasPrefix(file, n.ModulePath+"/") {
			rel := strings.TrimPrefix(file, n.ModulePath+"/")
			rel = filepath.FromSlash(rel)
			return filepath.Join(n.ModuleRoot, rel)
		}
	}
	if n.ModuleRoot != "" {
		return filepath.Join(n.ModuleRoot, filepath.FromSlash(clean))
	}
	return clean
}

// ToRelativePath converts a normalized path to a relative path.
func (n *DefaultPathNormalizer) ToRelativePath(normalized string) string {
	if n.ModuleRoot == "" {
		return normalized
	}
	rel, err := filepath.Rel(n.ModuleRoot, normalized)
	if err != nil {
		return normalized
	}
	return rel
}

// CoverageClassification represents how a file's coverage is classified.
type CoverageClassification struct {
	File      string
	Domain    string
	Excluded  bool
	Annotated bool
	Reason    string
}

// ClassifyFiles classifies files according to domain rules without aggregating coverage.
// This is useful for debugging and reporting.
func (a *CoverageAggregator) ClassifyFiles(input AggregationInput) []CoverageClassification {
	var classifications []CoverageClassification

	for file := range input.FileCoverage {
		normalized := a.PathNormalizer.NormalizePath(file)
		relPath := a.PathNormalizer.ToRelativePath(normalized)

		// Check global exclusions
		if a.isExcluded(relPath, input.GlobalExcludes) {
			classifications = append(classifications, CoverageClassification{
				File:     file,
				Excluded: true,
				Reason:   "matches global exclude pattern",
			})
			continue
		}

		// Check annotations
		if ann, ok := input.Annotations[filepath.ToSlash(relPath)]; ok {
			if ann.Ignore {
				classifications = append(classifications, CoverageClassification{
					File:      file,
					Excluded:  true,
					Annotated: true,
					Reason:    "ignored by annotation",
				})
				continue
			}
			if ann.Domain != "" {
				classifications = append(classifications, CoverageClassification{
					File:      file,
					Domain:    ann.Domain,
					Annotated: true,
					Reason:    "assigned by annotation",
				})
				continue
			}
		}

		// Match against domain directories
		matched := false
		for domainName, dirs := range input.DomainDirs {
			if a.matchesAnyDir(normalized, dirs) {
				// Check domain-specific excludes
				if input.DomainExcludes != nil {
					if patterns, ok := input.DomainExcludes[domainName]; ok {
						if a.isExcluded(relPath, patterns) {
							classifications = append(classifications, CoverageClassification{
								File:     file,
								Domain:   domainName,
								Excluded: true,
								Reason:   "matches domain-specific exclude pattern",
							})
							matched = true
							continue
						}
					}
				}
				classifications = append(classifications, CoverageClassification{
					File:   file,
					Domain: domainName,
					Reason: "matches domain directory",
				})
				matched = true
			}
		}

		if !matched {
			classifications = append(classifications, CoverageClassification{
				File:   file,
				Reason: "no domain match",
			})
		}
	}

	return classifications
}
