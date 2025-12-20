package application

import (
	"context"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/felixgeelhaar/coverctl/internal/domain"
)

type Service struct {
	ConfigLoader      ConfigLoader
	Autodetector      Autodetector
	DomainResolver    DomainResolver
	CoverageRunner    CoverageRunner
	ProfileParser     ProfileParser
	DiffProvider      DiffProvider
	AnnotationScanner AnnotationScanner
	Reporter          Reporter
	Out               io.Writer
}

type CheckOptions struct {
	ConfigPath string
	Output     OutputFormat
	Profile    string
}

type RunOnlyOptions struct {
	ConfigPath string
	Profile    string
}

type ReportOptions struct {
	ConfigPath string
	Profile    string
	Output     OutputFormat
}

type DetectOptions struct {
}

func (s *Service) Check(ctx context.Context, opts CheckOptions) error {
	cfg, domains, err := s.loadOrDetect(opts.ConfigPath)
	if err != nil {
		return err
	}

	profile, err := s.CoverageRunner.Run(ctx, RunOptions{Domains: domains, ProfilePath: opts.Profile})
	if err != nil {
		return err
	}

	moduleRoot, err := s.DomainResolver.ModuleRoot(ctx)
	if err != nil {
		return err
	}

	modulePath, err := s.DomainResolver.ModulePath(ctx)
	if err != nil {
		return err
	}

	profiles := []string{profile}
	if cfg.Integration.Enabled {
		integrationProfile, err := s.CoverageRunner.RunIntegration(ctx, IntegrationOptions{
			Domains:  domains,
			Packages: cfg.Integration.Packages,
			RunArgs:  cfg.Integration.RunArgs,
			CoverDir: cfg.Integration.CoverDir,
			Profile:  cfg.Integration.Profile,
		})
		if err != nil {
			return err
		}
		profiles = append(profiles, integrationProfile)
	}
	if len(cfg.Merge.Profiles) > 0 {
		profiles = append(profiles, cfg.Merge.Profiles...)
	}
	fileCoverage, err := s.ProfileParser.ParseAll(profiles)
	if err != nil {
		return err
	}

	normalizedCoverage := normalizeCoverageMap(fileCoverage, moduleRoot, modulePath)
	annotations, err := s.loadAnnotations(ctx, cfg, moduleRoot, normalizedCoverage)
	if err != nil {
		return err
	}
	changedFiles, err := s.diffFiles(ctx, cfg)
	if err != nil {
		return err
	}
	filteredCoverage := filterCoverageByFiles(normalizedCoverage, changedFiles)
	if cfg.Diff.Enabled && len(filteredCoverage) == 0 {
		result := domain.Result{Passed: true}
		result.Warnings = []string{"no files matched diff-based coverage check"}
		return s.Reporter.Write(s.Out, result, opts.Output)
	}

	domainDirs, err := s.DomainResolver.Resolve(ctx, domains)
	if err != nil {
		return err
	}

	domainCoverage := AggregateByDomain(filteredCoverage, domainDirs, cfg.Exclude, moduleRoot, modulePath, annotations)
	policy := cfg.Policy
	if cfg.Diff.Enabled {
		policy.Domains = filterPolicyDomains(policy.Domains, domainCoverage)
	}
	result := domain.Evaluate(policy, domainCoverage)
	result.Warnings = domainOverlapWarnings(domainDirs)
	fileResults, filesPassed := evaluateFileRules(filteredCoverage, cfg.Files, cfg.Exclude, annotations)
	result.Files = fileResults
	if !filesPassed {
		result.Passed = false
	}

	if err := s.Reporter.Write(s.Out, result, opts.Output); err != nil {
		return err
	}
	if !result.Passed {
		return fmt.Errorf("policy violation")
	}
	return nil
}

func (s *Service) RunOnly(ctx context.Context, opts RunOnlyOptions) error {
	_, domains, err := s.loadOrDetect(opts.ConfigPath)
	if err != nil {
		return err
	}
	_, err = s.CoverageRunner.Run(ctx, RunOptions{Domains: domains, ProfilePath: opts.Profile})
	return err
}

func (s *Service) Report(ctx context.Context, opts ReportOptions) error {
	cfg, domains, err := s.loadOrDetect(opts.ConfigPath)
	if err != nil {
		return err
	}

	moduleRoot, err := s.DomainResolver.ModuleRoot(ctx)
	if err != nil {
		return err
	}

	modulePath, err := s.DomainResolver.ModulePath(ctx)
	if err != nil {
		return err
	}

	profiles := []string{opts.Profile}
	if len(cfg.Merge.Profiles) > 0 {
		profiles = append(profiles, cfg.Merge.Profiles...)
	}
	fileCoverage, err := s.ProfileParser.ParseAll(profiles)
	if err != nil {
		return err
	}

	normalizedCoverage := normalizeCoverageMap(fileCoverage, moduleRoot, modulePath)
	annotations, err := s.loadAnnotations(ctx, cfg, moduleRoot, normalizedCoverage)
	if err != nil {
		return err
	}
	changedFiles, err := s.diffFiles(ctx, cfg)
	if err != nil {
		return err
	}
	filteredCoverage := filterCoverageByFiles(normalizedCoverage, changedFiles)
	if cfg.Diff.Enabled && len(filteredCoverage) == 0 {
		result := domain.Result{Passed: true}
		result.Warnings = []string{"no files matched diff-based coverage check"}
		return s.Reporter.Write(s.Out, result, opts.Output)
	}

	domainDirs, err := s.DomainResolver.Resolve(ctx, domains)
	if err != nil {
		return err
	}

	domainCoverage := AggregateByDomain(filteredCoverage, domainDirs, cfg.Exclude, moduleRoot, modulePath, annotations)
	policy := cfg.Policy
	if cfg.Diff.Enabled {
		policy.Domains = filterPolicyDomains(policy.Domains, domainCoverage)
	}
	result := domain.Evaluate(policy, domainCoverage)
	result.Warnings = domainOverlapWarnings(domainDirs)
	fileResults, filesPassed := evaluateFileRules(filteredCoverage, cfg.Files, cfg.Exclude, annotations)
	result.Files = fileResults
	if !filesPassed {
		result.Passed = false
	}

	return s.Reporter.Write(s.Out, result, opts.Output)
}

func (s *Service) Detect(ctx context.Context, opts DetectOptions) (Config, error) {
	cfg, err := s.Autodetector.Detect()
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (s *Service) Ignore(ctx context.Context, opts IgnoreOptions) (Config, []domain.Domain, error) {
	cfg, domains, err := s.loadOrDetect(opts.ConfigPath)
	if err != nil {
		return Config{}, nil, err
	}
	return cfg, domains, nil
}

func (s *Service) loadOrDetect(configPath string) (Config, []domain.Domain, error) {
	exists, err := s.ConfigLoader.Exists(configPath)
	if err != nil {
		return Config{}, nil, err
	}

	var cfg Config
	if !exists {
		cfg, err = s.Autodetector.Detect()
		if err != nil {
			return Config{}, nil, err
		}
	} else {
		cfg, err = s.ConfigLoader.Load(configPath)
		if err != nil {
			return Config{}, nil, err
		}
	}

	if len(cfg.Policy.Domains) == 0 {
		return Config{}, nil, fmt.Errorf("no domains configured")
	}

	return cfg, cfg.Policy.Domains, nil
}

// AggregateByDomain matches files to domain directories and aggregates coverage.
func AggregateByDomain(files map[string]domain.CoverageStat, domainDirs map[string][]string, exclude []string, moduleRoot, modulePath string, annotations map[string]Annotation) map[string]domain.CoverageStat {
	result := make(map[string]domain.CoverageStat, len(domainDirs))

	for file, stat := range files {
		normalized := normalizeCoverageFile(file, modulePath, moduleRoot)
		relPath := moduleRelativePath(normalized, moduleRoot)
		if excluded(relPath, exclude) {
			continue
		}
		if ann, ok := annotations[filepath.ToSlash(relPath)]; ok {
			if ann.Ignore {
				continue
			}
			if ann.Domain != "" {
				agg := result[ann.Domain]
				agg.Covered += stat.Covered
				agg.Total += stat.Total
				result[ann.Domain] = agg
				continue
			}
		}
		for domainName, dirs := range domainDirs {
			if matchesAnyDir(normalized, dirs, moduleRoot) {
				agg := result[domainName]
				agg.Covered += stat.Covered
				agg.Total += stat.Total
				result[domainName] = agg
			}
		}
	}
	return result
}

func excluded(file string, patterns []string) bool {
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

func matchesAnyDir(file string, dirs []string, moduleRoot string) bool {
	cleanFile := filepath.Clean(file)
	for _, dir := range dirs {
		cleanDir := filepath.Clean(dir)
		if strings.HasPrefix(cleanFile, cleanDir+string(filepath.Separator)) || cleanFile == cleanDir {
			return true
		}
		if moduleRoot != "" {
			relDir, err := filepath.Rel(moduleRoot, cleanDir)
			if err == nil {
				relDir = filepath.Clean(relDir)
				if relDir == "." {
					return true
				}
				if strings.HasPrefix(cleanFile, relDir+string(filepath.Separator)) || cleanFile == relDir {
					return true
				}
			}
		}
	}
	return false
}

func normalizeCoverageFile(file, modulePath, moduleRoot string) string {
	clean := filepath.Clean(file)
	if filepath.IsAbs(clean) {
		return clean
	}
	if modulePath != "" {
		if file == modulePath {
			return filepath.Clean(moduleRoot)
		}
		if strings.HasPrefix(file, modulePath+"/") {
			rel := strings.TrimPrefix(file, modulePath+"/")
			rel = filepath.FromSlash(rel)
			return filepath.Join(moduleRoot, rel)
		}
	}
	if moduleRoot != "" {
		return filepath.Join(moduleRoot, filepath.FromSlash(clean))
	}
	return clean
}

func moduleRelativePath(path, moduleRoot string) string {
	if moduleRoot == "" {
		return filepath.Clean(path)
	}
	rel, err := filepath.Rel(moduleRoot, path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(rel)
}

func normalizeCoverageMap(files map[string]domain.CoverageStat, moduleRoot, modulePath string) map[string]domain.CoverageStat {
	result := make(map[string]domain.CoverageStat, len(files))
	for file, stat := range files {
		normalized := normalizeCoverageFile(file, modulePath, moduleRoot)
		rel := filepath.ToSlash(moduleRelativePath(normalized, moduleRoot))
		agg := result[rel]
		agg.Covered += stat.Covered
		agg.Total += stat.Total
		result[rel] = agg
	}
	return result
}

func filterCoverageByFiles(files map[string]domain.CoverageStat, allow map[string]struct{}) map[string]domain.CoverageStat {
	if allow == nil {
		return files
	}
	filtered := make(map[string]domain.CoverageStat)
	for file, stat := range files {
		if _, ok := allow[file]; ok {
			filtered[file] = stat
		}
	}
	return filtered
}

func evaluateFileRules(files map[string]domain.CoverageStat, rules []domain.FileRule, exclude []string, annotations map[string]Annotation) ([]domain.FileResult, bool) {
	if len(rules) == 0 {
		return nil, true
	}
	minByFile := make(map[string]float64)
	for file := range files {
		if excluded(file, exclude) {
			continue
		}
		if ann, ok := annotations[file]; ok && ann.Ignore {
			continue
		}
		for _, rule := range rules {
			if matchAnyPattern(file, rule.Match) {
				if minByFile[file] < rule.Min {
					minByFile[file] = rule.Min
				}
			}
		}
	}
	results := make([]domain.FileResult, 0, len(minByFile))
	passed := true
	for file, min := range minByFile {
		stat := files[file]
		percent := round1(stat.Percent())
		status := domain.StatusPass
		if percent < min {
			status = domain.StatusFail
			passed = false
		}
		results = append(results, domain.FileResult{
			File:     file,
			Covered:  stat.Covered,
			Total:    stat.Total,
			Percent:  percent,
			Required: min,
			Status:   status,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].File < results[j].File
	})
	return results, passed
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func matchAnyPattern(file string, patterns []string) bool {
	for _, pattern := range patterns {
		if ok, _ := filepath.Match(pattern, file); ok {
			return true
		}
	}
	return false
}

func filterPolicyDomains(domains []domain.Domain, coverage map[string]domain.CoverageStat) []domain.Domain {
	filtered := make([]domain.Domain, 0, len(domains))
	for _, d := range domains {
		if stat, ok := coverage[d.Name]; ok && stat.Total > 0 {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

func (s *Service) diffFiles(ctx context.Context, cfg Config) (map[string]struct{}, error) {
	if !cfg.Diff.Enabled || s.DiffProvider == nil {
		return nil, nil
	}
	files, err := s.DiffProvider.ChangedFiles(ctx, cfg.Diff.Base)
	if err != nil {
		return nil, err
	}
	allow := make(map[string]struct{}, len(files))
	for _, file := range files {
		allow[filepath.ToSlash(filepath.Clean(file))] = struct{}{}
	}
	return allow, nil
}

func (s *Service) loadAnnotations(ctx context.Context, cfg Config, moduleRoot string, files map[string]domain.CoverageStat) (map[string]Annotation, error) {
	if !cfg.Annotations.Enabled || s.AnnotationScanner == nil {
		return nil, nil
	}
	paths := make([]string, 0, len(files))
	for file := range files {
		paths = append(paths, file)
	}
	return s.AnnotationScanner.Scan(ctx, moduleRoot, paths)
}

func domainOverlapWarnings(domainDirs map[string][]string) []string {
	dirOwners := make(map[string][]string, len(domainDirs))
	for name, dirs := range domainDirs {
		for _, dir := range dirs {
			cleanDir := filepath.Clean(dir)
			dirOwners[cleanDir] = append(dirOwners[cleanDir], name)
		}
	}
	var warnings []string
	for dir, owners := range dirOwners {
		if len(owners) <= 1 {
			continue
		}
		sort.Strings(owners)
		warnings = append(warnings, fmt.Sprintf("directory %s belongs to %s domains", dir, strings.Join(owners, ", ")))
	}
	sort.Strings(warnings)
	return warnings
}
