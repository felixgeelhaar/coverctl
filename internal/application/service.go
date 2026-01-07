package application

import (
	"context"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
	ConfigPath   string
	Output       OutputFormat
	Profile      string
	Domains      []string     // Filter to specific domains (empty = all domains)
	HistoryStore HistoryStore // Optional: for delta calculation
	FailUnder    *float64     // Optional: fail if overall coverage is below this threshold
	Ratchet      bool         // Fail if coverage decreases from previous recorded value
	BuildFlags   BuildFlags   // Build and test flags
}

type RunOnlyOptions struct {
	ConfigPath string
	Profile    string
	Domains    []string   // Filter to specific domains (empty = all domains)
	BuildFlags BuildFlags // Build and test flags
}

type ReportOptions struct {
	ConfigPath    string
	Profile       string
	Output        OutputFormat
	Domains       []string     // Filter to specific domains (empty = all domains)
	HistoryStore  HistoryStore // Optional: for delta calculation
	ShowUncovered bool         // Show only files with 0% coverage
	DiffRef       string       // Git ref for diff-based filtering (overrides config)
	MergeProfiles []string     // Additional profile files to merge
}

type DetectOptions struct {
}

// CheckResult runs coverage tests and evaluates policy, returning the result.
// This is the pure function version that returns data instead of writing to output.
func (s *Service) CheckResult(ctx context.Context, opts CheckOptions) (domain.Result, error) {
	cfg, domains, err := s.loadOrDetect(opts.ConfigPath)
	if err != nil {
		return domain.Result{}, err
	}

	// Filter domains if specific ones are requested
	domains = filterDomainsByNames(domains, opts.Domains)
	if len(domains) == 0 {
		return domain.Result{}, fmt.Errorf("no matching domains found for: %v", opts.Domains)
	}

	profile, err := s.CoverageRunner.Run(ctx, RunOptions{Domains: domains, ProfilePath: opts.Profile, BuildFlags: opts.BuildFlags})
	if err != nil {
		return domain.Result{}, err
	}

	moduleRoot, err := s.DomainResolver.ModuleRoot(ctx)
	if err != nil {
		return domain.Result{}, err
	}

	modulePath, err := s.DomainResolver.ModulePath(ctx)
	if err != nil {
		return domain.Result{}, err
	}

	profiles := []string{profile}
	if cfg.Integration.Enabled {
		integrationProfile, err := s.CoverageRunner.RunIntegration(ctx, IntegrationOptions{
			Domains:    domains,
			Packages:   cfg.Integration.Packages,
			RunArgs:    cfg.Integration.RunArgs,
			CoverDir:   cfg.Integration.CoverDir,
			Profile:    cfg.Integration.Profile,
			BuildFlags: opts.BuildFlags,
		})
		if err != nil {
			return domain.Result{}, err
		}
		profiles = append(profiles, integrationProfile)
	}
	if len(cfg.Merge.Profiles) > 0 {
		profiles = append(profiles, cfg.Merge.Profiles...)
	}
	fileCoverage, err := s.ProfileParser.ParseAll(profiles)
	if err != nil {
		return domain.Result{}, err
	}

	normalizedCoverage := normalizeCoverageMap(fileCoverage, moduleRoot, modulePath)
	annotations, err := s.loadAnnotations(ctx, cfg, moduleRoot, normalizedCoverage)
	if err != nil {
		return domain.Result{}, err
	}
	changedFiles, err := s.diffFiles(ctx, cfg)
	if err != nil {
		return domain.Result{}, err
	}
	filteredCoverage := filterCoverageByFiles(normalizedCoverage, changedFiles)
	if cfg.Diff.Enabled && len(filteredCoverage) == 0 {
		result := domain.Result{Passed: true}
		result.Warnings = []string{"no files matched diff-based coverage check"}
		return result, nil
	}

	domainDirs, err := s.DomainResolver.Resolve(ctx, domains)
	if err != nil {
		return domain.Result{}, err
	}

	domainExcludes := buildDomainExcludes(domains)
	domainCoverage := AggregateByDomainWithExcludes(filteredCoverage, domainDirs, cfg.Exclude, domainExcludes, moduleRoot, modulePath, annotations)
	policy := cfg.Policy
	// Use filtered domains for policy evaluation
	policy.Domains = domains
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

	// Apply deltas from history if available
	if opts.HistoryStore != nil {
		history, err := opts.HistoryStore.Load()
		if err == nil {
			applyDeltas(&result, history)
		}
	}

	return result, nil
}

func (s *Service) Check(ctx context.Context, opts CheckOptions) error {
	result, err := s.CheckResult(ctx, opts)
	if err != nil {
		return err
	}

	if err := s.Reporter.Write(s.Out, result, opts.Output); err != nil {
		return err
	}

	// Check fail-under threshold if specified
	if opts.FailUnder != nil {
		overallPercent := calculateOverallPercent(result)
		if overallPercent < *opts.FailUnder {
			return fmt.Errorf("coverage %.1f%% is below --fail-under threshold of %.1f%%", overallPercent, *opts.FailUnder)
		}
	}

	// Check ratchet: coverage must not decrease from previous value
	if opts.Ratchet && opts.HistoryStore != nil {
		hist, err := opts.HistoryStore.Load()
		if err == nil && len(hist.Entries) > 0 {
			previousPercent := hist.Entries[len(hist.Entries)-1].Overall
			currentPercent := calculateOverallPercent(result)
			if currentPercent < previousPercent {
				return fmt.Errorf("coverage decreased from %.1f%% to %.1f%% (--ratchet prevents regression)", previousPercent, currentPercent)
			}
		}
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

	// Filter domains if specific ones are requested
	domains = filterDomainsByNames(domains, opts.Domains)
	if len(domains) == 0 {
		return fmt.Errorf("no matching domains found for: %v", opts.Domains)
	}

	_, err = s.CoverageRunner.Run(ctx, RunOptions{Domains: domains, ProfilePath: opts.Profile, BuildFlags: opts.BuildFlags})
	return err
}

// ReportResult analyzes an existing coverage profile and returns the result.
// This is the pure function version that returns data instead of writing to output.
func (s *Service) ReportResult(ctx context.Context, opts ReportOptions) (domain.Result, error) {
	cfg, domains, err := s.loadOrDetect(opts.ConfigPath)
	if err != nil {
		return domain.Result{}, err
	}

	// Filter domains if specific ones are requested
	domains = filterDomainsByNames(domains, opts.Domains)
	if len(domains) == 0 {
		return domain.Result{}, fmt.Errorf("no matching domains found for: %v", opts.Domains)
	}

	moduleRoot, err := s.DomainResolver.ModuleRoot(ctx)
	if err != nil {
		return domain.Result{}, err
	}

	modulePath, err := s.DomainResolver.ModulePath(ctx)
	if err != nil {
		return domain.Result{}, err
	}

	profiles := []string{opts.Profile}
	if len(cfg.Merge.Profiles) > 0 {
		profiles = append(profiles, cfg.Merge.Profiles...)
	}
	// Add CLI-specified merge profiles
	if len(opts.MergeProfiles) > 0 {
		profiles = append(profiles, opts.MergeProfiles...)
	}
	fileCoverage, err := s.ProfileParser.ParseAll(profiles)
	if err != nil {
		return domain.Result{}, err
	}

	normalizedCoverage := normalizeCoverageMap(fileCoverage, moduleRoot, modulePath)
	annotations, err := s.loadAnnotations(ctx, cfg, moduleRoot, normalizedCoverage)
	if err != nil {
		return domain.Result{}, err
	}

	// Handle --uncovered flag: show only files with 0% coverage
	if opts.ShowUncovered {
		return s.reportUncoveredResult(normalizedCoverage, cfg.Exclude, annotations)
	}

	// Handle --diff flag: override config diff setting
	diffCfg := cfg.Diff
	if opts.DiffRef != "" {
		diffCfg.Enabled = true
		diffCfg.Base = opts.DiffRef
	}
	changedFiles, err := s.diffFilesWithConfig(ctx, diffCfg)
	if err != nil {
		return domain.Result{}, err
	}
	filteredCoverage := filterCoverageByFiles(normalizedCoverage, changedFiles)
	if diffCfg.Enabled && len(filteredCoverage) == 0 {
		result := domain.Result{Passed: true}
		result.Warnings = []string{"no files matched diff-based coverage check"}
		return result, nil
	}

	domainDirs, err := s.DomainResolver.Resolve(ctx, domains)
	if err != nil {
		return domain.Result{}, err
	}

	domainExcludes := buildDomainExcludes(domains)
	domainCoverage := AggregateByDomainWithExcludes(filteredCoverage, domainDirs, cfg.Exclude, domainExcludes, moduleRoot, modulePath, annotations)
	policy := cfg.Policy
	// Use filtered domains for policy evaluation
	policy.Domains = domains
	if diffCfg.Enabled {
		policy.Domains = filterPolicyDomains(policy.Domains, domainCoverage)
	}
	result := domain.Evaluate(policy, domainCoverage)
	result.Warnings = domainOverlapWarnings(domainDirs)
	fileResults, filesPassed := evaluateFileRules(filteredCoverage, cfg.Files, cfg.Exclude, annotations)
	result.Files = fileResults
	if !filesPassed {
		result.Passed = false
	}

	// Apply deltas from history if available
	if opts.HistoryStore != nil {
		history, err := opts.HistoryStore.Load()
		if err == nil {
			applyDeltas(&result, history)
		}
	}

	return result, nil
}

func (s *Service) Report(ctx context.Context, opts ReportOptions) error {
	result, err := s.ReportResult(ctx, opts)
	if err != nil {
		return err
	}
	return s.Reporter.Write(s.Out, result, opts.Output)
}

// reportUncoveredResult returns a result of files with 0% coverage.
func (s *Service) reportUncoveredResult(files map[string]domain.CoverageStat, exclude []string, annotations map[string]Annotation) (domain.Result, error) {
	var uncoveredFiles []domain.FileResult
	for file, stat := range files {
		if excluded(file, exclude) {
			continue
		}
		if ann, ok := annotations[file]; ok && ann.Ignore {
			continue
		}
		percent := 0.0
		if stat.Total > 0 {
			percent = round1((float64(stat.Covered) / float64(stat.Total)) * 100)
		}
		if percent == 0 && stat.Total > 0 {
			uncoveredFiles = append(uncoveredFiles, domain.FileResult{
				File:     file,
				Covered:  stat.Covered,
				Total:    stat.Total,
				Percent:  0,
				Required: 0,
				Status:   domain.StatusFail,
			})
		}
	}
	sort.Slice(uncoveredFiles, func(i, j int) bool {
		return uncoveredFiles[i].File < uncoveredFiles[j].File
	})

	result := domain.Result{
		Passed: len(uncoveredFiles) == 0,
		Files:  uncoveredFiles,
	}
	if len(uncoveredFiles) > 0 {
		result.Warnings = []string{fmt.Sprintf("%d files have 0%% coverage", len(uncoveredFiles))}
	}
	return result, nil
}

// diffFilesWithConfig gets changed files using the given diff configuration.
func (s *Service) diffFilesWithConfig(ctx context.Context, cfg DiffConfig) (map[string]struct{}, error) {
	if !cfg.Enabled || s.DiffProvider == nil {
		return nil, nil
	}
	files, err := s.DiffProvider.ChangedFiles(ctx, cfg.Base)
	if err != nil {
		return nil, err
	}
	allow := make(map[string]struct{}, len(files))
	for _, file := range files {
		allow[filepath.ToSlash(filepath.Clean(file))] = struct{}{}
	}
	return allow, nil
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
	return AggregateByDomainWithExcludes(files, domainDirs, exclude, nil, moduleRoot, modulePath, annotations)
}

// AggregateByDomainWithExcludes matches files to domain directories and aggregates coverage,
// supporting both global excludes and per-domain excludes.
func AggregateByDomainWithExcludes(files map[string]domain.CoverageStat, domainDirs map[string][]string, exclude []string, domainExcludes map[string][]string, moduleRoot, modulePath string, annotations map[string]Annotation) map[string]domain.CoverageStat {
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
				// Check domain-specific excludes
				if domainExcludes != nil {
					if excludePatterns, ok := domainExcludes[domainName]; ok && excluded(relPath, excludePatterns) {
						continue
					}
				}
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

// filterDomainsByNames filters domains to only those whose names match the given list.
// If names is empty, all domains are returned unchanged.
func filterDomainsByNames(domains []domain.Domain, names []string) []domain.Domain {
	if len(names) == 0 {
		return domains
	}
	nameSet := make(map[string]struct{}, len(names))
	for _, name := range names {
		nameSet[name] = struct{}{}
	}
	filtered := make([]domain.Domain, 0, len(names))
	for _, d := range domains {
		if _, ok := nameSet[d.Name]; ok {
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

// buildDomainExcludes creates a map of domain name to exclude patterns from domain configs.
func buildDomainExcludes(domains []domain.Domain) map[string][]string {
	result := make(map[string][]string)
	for _, d := range domains {
		if len(d.Exclude) > 0 {
			result[d.Name] = d.Exclude
		}
	}
	return result
}

// calculateOverallPercent computes the weighted average coverage from domain results
func calculateOverallPercent(result domain.Result) float64 {
	var totalCovered, totalLines int
	for _, dr := range result.Domains {
		totalCovered += dr.Covered
		totalLines += dr.Total
	}
	if totalLines == 0 {
		return 0
	}
	return float64(totalCovered) / float64(totalLines) * 100
}

// BadgeResult contains the data needed to generate a coverage badge.
type BadgeResult struct {
	Percent float64
}

// Badge calculates overall coverage for badge generation.
func (s *Service) Badge(ctx context.Context, opts BadgeOptions) (BadgeResult, error) {
	cfg, domains, err := s.loadOrDetect(opts.ConfigPath)
	if err != nil {
		return BadgeResult{}, err
	}

	moduleRoot, err := s.DomainResolver.ModuleRoot(ctx)
	if err != nil {
		return BadgeResult{}, err
	}

	modulePath, err := s.DomainResolver.ModulePath(ctx)
	if err != nil {
		return BadgeResult{}, err
	}

	profiles := []string{opts.ProfilePath}
	if len(cfg.Merge.Profiles) > 0 {
		profiles = append(profiles, cfg.Merge.Profiles...)
	}
	fileCoverage, err := s.ProfileParser.ParseAll(profiles)
	if err != nil {
		return BadgeResult{}, err
	}

	normalizedCoverage := normalizeCoverageMap(fileCoverage, moduleRoot, modulePath)
	annotations, err := s.loadAnnotations(ctx, cfg, moduleRoot, normalizedCoverage)
	if err != nil {
		return BadgeResult{}, err
	}

	domainDirs, err := s.DomainResolver.Resolve(ctx, domains)
	if err != nil {
		return BadgeResult{}, err
	}

	domainExcludes := buildDomainExcludes(domains)
	domainCoverage := AggregateByDomainWithExcludes(normalizedCoverage, domainDirs, cfg.Exclude, domainExcludes, moduleRoot, modulePath, annotations)

	// Calculate overall coverage across all domains
	var totalCovered, totalStatements int
	for _, stat := range domainCoverage {
		totalCovered += stat.Covered
		totalStatements += stat.Total
	}

	percent := 0.0
	if totalStatements > 0 {
		percent = round1((float64(totalCovered) / float64(totalStatements)) * 100)
	}

	return BadgeResult{Percent: percent}, nil
}

// TrendResult contains trend analysis data.
type TrendResult struct {
	Current  float64
	Previous float64
	Trend    domain.Trend
	Entries  []domain.HistoryEntry
	ByDomain map[string]domain.Trend
}

// Trend analyzes coverage trends over time.
func (s *Service) Trend(ctx context.Context, opts TrendOptions, store HistoryStore) (TrendResult, error) {
	history, err := store.Load()
	if err != nil {
		return TrendResult{}, err
	}

	if len(history.Entries) == 0 {
		return TrendResult{}, fmt.Errorf("no history data available; run 'coverctl record' after coverage runs")
	}

	// Get current coverage
	cfg, domains, err := s.loadOrDetect(opts.ConfigPath)
	if err != nil {
		return TrendResult{}, err
	}

	moduleRoot, err := s.DomainResolver.ModuleRoot(ctx)
	if err != nil {
		return TrendResult{}, err
	}

	modulePath, err := s.DomainResolver.ModulePath(ctx)
	if err != nil {
		return TrendResult{}, err
	}

	profiles := []string{opts.ProfilePath}
	if len(cfg.Merge.Profiles) > 0 {
		profiles = append(profiles, cfg.Merge.Profiles...)
	}
	fileCoverage, err := s.ProfileParser.ParseAll(profiles)
	if err != nil {
		return TrendResult{}, err
	}

	normalizedCoverage := normalizeCoverageMap(fileCoverage, moduleRoot, modulePath)
	annotations, err := s.loadAnnotations(ctx, cfg, moduleRoot, normalizedCoverage)
	if err != nil {
		return TrendResult{}, err
	}

	domainDirs, err := s.DomainResolver.Resolve(ctx, domains)
	if err != nil {
		return TrendResult{}, err
	}

	domainExcludes := buildDomainExcludes(domains)
	domainCoverage := AggregateByDomainWithExcludes(normalizedCoverage, domainDirs, cfg.Exclude, domainExcludes, moduleRoot, modulePath, annotations)

	// Calculate current overall coverage
	var totalCovered, totalStatements int
	for _, stat := range domainCoverage {
		totalCovered += stat.Covered
		totalStatements += stat.Total
	}
	currentPercent := 0.0
	if totalStatements > 0 {
		currentPercent = round1((float64(totalCovered) / float64(totalStatements)) * 100)
	}

	// Get previous entry for trend calculation
	latest := history.LatestEntry()
	previousPercent := latest.Overall
	trend := domain.CalculateTrend(previousPercent, currentPercent)

	// Calculate per-domain trends
	byDomain := make(map[string]domain.Trend)
	for domainName, stat := range domainCoverage {
		currentDomainPercent := 0.0
		if stat.Total > 0 {
			currentDomainPercent = round1((float64(stat.Covered) / float64(stat.Total)) * 100)
		}
		if prevEntry, ok := latest.Domains[domainName]; ok {
			byDomain[domainName] = domain.CalculateTrend(prevEntry.Percent, currentDomainPercent)
		} else {
			byDomain[domainName] = domain.Trend{Direction: domain.TrendStable, Delta: 0}
		}
	}

	return TrendResult{
		Current:  currentPercent,
		Previous: previousPercent,
		Trend:    trend,
		Entries:  history.Entries,
		ByDomain: byDomain,
	}, nil
}

// Record saves current coverage to history.
func (s *Service) Record(ctx context.Context, opts RecordOptions, store HistoryStore) error {
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

	profiles := []string{opts.ProfilePath}
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

	domainDirs, err := s.DomainResolver.Resolve(ctx, domains)
	if err != nil {
		return err
	}

	domainExcludes := buildDomainExcludes(domains)
	domainCoverage := AggregateByDomainWithExcludes(normalizedCoverage, domainDirs, cfg.Exclude, domainExcludes, moduleRoot, modulePath, annotations)

	// Calculate overall coverage
	var totalCovered, totalStatements int
	domainEntries := make(map[string]domain.DomainEntry)
	for domainName, stat := range domainCoverage {
		totalCovered += stat.Covered
		totalStatements += stat.Total

		percent := 0.0
		if stat.Total > 0 {
			percent = round1((float64(stat.Covered) / float64(stat.Total)) * 100)
		}

		// Find the min threshold for this domain
		var min float64
		for _, d := range domains {
			if d.Name == domainName && d.Min != nil {
				min = *d.Min
				break
			}
		}
		if min == 0 {
			min = cfg.Policy.DefaultMin
		}

		status := domain.StatusPass
		if percent < min {
			status = domain.StatusFail
		}

		domainEntries[domainName] = domain.DomainEntry{
			Name:    domainName,
			Percent: percent,
			Min:     min,
			Status:  status,
		}
	}

	overallPercent := 0.0
	if totalStatements > 0 {
		overallPercent = round1((float64(totalCovered) / float64(totalStatements)) * 100)
	}

	entry := domain.HistoryEntry{
		Timestamp: timeNow(),
		Commit:    opts.Commit,
		Branch:    opts.Branch,
		Overall:   overallPercent,
		Domains:   domainEntries,
	}

	return store.Append(entry)
}

// timeNow is a variable to allow test injection
var timeNow = func() time.Time {
	return time.Now()
}

// applyDeltas calculates and applies coverage deltas from history to the result.
func applyDeltas(result *domain.Result, history domain.History) {
	if len(history.Entries) == 0 {
		return
	}
	latest := history.LatestEntry()
	if latest == nil {
		return
	}

	for i := range result.Domains {
		domainName := result.Domains[i].Domain
		if prevEntry, ok := latest.Domains[domainName]; ok {
			delta := round1(result.Domains[i].Percent - prevEntry.Percent)
			result.Domains[i].Delta = &delta
		}
	}
}

// SuggestResult contains threshold suggestions for all domains.
type SuggestResult struct {
	Suggestions []Suggestion
	Config      Config
}

// Suggest analyzes current coverage and suggests optimal thresholds.
func (s *Service) Suggest(ctx context.Context, opts SuggestOptions) (SuggestResult, error) {
	cfg, domains, err := s.loadOrDetect(opts.ConfigPath)
	if err != nil {
		return SuggestResult{}, err
	}

	moduleRoot, err := s.DomainResolver.ModuleRoot(ctx)
	if err != nil {
		return SuggestResult{}, err
	}

	modulePath, err := s.DomainResolver.ModulePath(ctx)
	if err != nil {
		return SuggestResult{}, err
	}

	profiles := []string{opts.ProfilePath}
	if len(cfg.Merge.Profiles) > 0 {
		profiles = append(profiles, cfg.Merge.Profiles...)
	}
	fileCoverage, err := s.ProfileParser.ParseAll(profiles)
	if err != nil {
		return SuggestResult{}, err
	}

	normalizedCoverage := normalizeCoverageMap(fileCoverage, moduleRoot, modulePath)
	annotations, err := s.loadAnnotations(ctx, cfg, moduleRoot, normalizedCoverage)
	if err != nil {
		return SuggestResult{}, err
	}

	domainDirs, err := s.DomainResolver.Resolve(ctx, domains)
	if err != nil {
		return SuggestResult{}, err
	}

	domainExcludes := buildDomainExcludes(domains)
	domainCoverage := AggregateByDomainWithExcludes(normalizedCoverage, domainDirs, cfg.Exclude, domainExcludes, moduleRoot, modulePath, annotations)

	// Generate suggestions for each domain
	suggestions := make([]Suggestion, 0, len(domains))
	for i, d := range domains {
		stat := domainCoverage[d.Name]
		currentPercent := 0.0
		if stat.Total > 0 {
			currentPercent = round1((float64(stat.Covered) / float64(stat.Total)) * 100)
		}

		currentMin := cfg.Policy.DefaultMin
		if d.Min != nil {
			currentMin = *d.Min
		}

		suggestedMin, reason := calculateSuggestion(currentPercent, currentMin, opts.Strategy)

		suggestions = append(suggestions, Suggestion{
			Domain:         d.Name,
			CurrentPercent: currentPercent,
			CurrentMin:     currentMin,
			SuggestedMin:   suggestedMin,
			Reason:         reason,
		})

		// Update config with suggested values
		suggestedMinPtr := suggestedMin
		cfg.Policy.Domains[i].Min = &suggestedMinPtr
	}

	return SuggestResult{
		Suggestions: suggestions,
		Config:      cfg,
	}, nil
}

func calculateSuggestion(current, currentMin float64, strategy SuggestStrategy) (float64, string) {
	switch strategy {
	case SuggestAggressive:
		// Suggest 5% above current, capped at 95%
		suggested := math.Min(current+5, 95)
		if suggested > currentMin {
			return round1(suggested), "push for improvement (+5%)"
		}
		return currentMin, "already at or above aggressive target"

	case SuggestConservative:
		// Suggest 5% below current, but at least current min
		suggested := math.Max(current-5, currentMin)
		suggested = math.Max(suggested, 50) // Never suggest below 50%
		return round1(suggested), "gradual improvement target"

	default: // SuggestCurrent
		// Suggest 2% below current to allow some variance
		suggested := current - 2
		if suggested < currentMin {
			return currentMin, "keep current threshold (coverage near minimum)"
		}
		if suggested < 50 {
			suggested = 50
		}
		return round1(suggested), "based on current coverage (-2% buffer)"
	}
}

// Debt calculates coverage debt - the gap between current and required coverage.
func (s *Service) Debt(ctx context.Context, opts DebtOptions) (DebtResult, error) {
	cfg, domains, err := s.loadOrDetect(opts.ConfigPath)
	if err != nil {
		return DebtResult{}, err
	}

	moduleRoot, err := s.DomainResolver.ModuleRoot(ctx)
	if err != nil {
		return DebtResult{}, err
	}

	modulePath, err := s.DomainResolver.ModulePath(ctx)
	if err != nil {
		return DebtResult{}, err
	}

	profiles := []string{opts.ProfilePath}
	if len(cfg.Merge.Profiles) > 0 {
		profiles = append(profiles, cfg.Merge.Profiles...)
	}
	fileCoverage, err := s.ProfileParser.ParseAll(profiles)
	if err != nil {
		return DebtResult{}, err
	}

	normalizedCoverage := normalizeCoverageMap(fileCoverage, moduleRoot, modulePath)
	annotations, err := s.loadAnnotations(ctx, cfg, moduleRoot, normalizedCoverage)
	if err != nil {
		return DebtResult{}, err
	}

	domainDirs, err := s.DomainResolver.Resolve(ctx, domains)
	if err != nil {
		return DebtResult{}, err
	}

	domainExcludes := buildDomainExcludes(domains)
	domainCoverage := AggregateByDomainWithExcludes(normalizedCoverage, domainDirs, cfg.Exclude, domainExcludes, moduleRoot, modulePath, annotations)

	var items []DebtItem
	var totalDebt float64
	var totalLines int
	var passCount, failCount int

	// Calculate domain debt
	for _, d := range domains {
		stat := domainCoverage[d.Name]
		currentPercent := 0.0
		if stat.Total > 0 {
			currentPercent = round1((float64(stat.Covered) / float64(stat.Total)) * 100)
		}

		required := cfg.Policy.DefaultMin
		if d.Min != nil {
			required = *d.Min
		}

		if currentPercent < required {
			shortfall := round1(required - currentPercent)
			// Estimate lines needing tests: (shortfall% * total statements) / 100
			linesNeeded := int(float64(stat.Total-stat.Covered) * (shortfall / (required - currentPercent + 0.01)))
			if linesNeeded < 0 {
				linesNeeded = stat.Total - stat.Covered
			}

			items = append(items, DebtItem{
				Name:      d.Name,
				Type:      "domain",
				Current:   currentPercent,
				Required:  required,
				Shortfall: shortfall,
				Lines:     linesNeeded,
			})
			totalDebt += shortfall
			totalLines += linesNeeded
			failCount++
		} else {
			passCount++
		}
	}

	// Calculate file rule debt
	for _, rule := range cfg.Files {
		for file, stat := range normalizedCoverage {
			if excluded(file, cfg.Exclude) {
				continue
			}
			if ann, ok := annotations[file]; ok && ann.Ignore {
				continue
			}
			if matchAnyPattern(file, rule.Match) {
				currentPercent := 0.0
				if stat.Total > 0 {
					currentPercent = round1((float64(stat.Covered) / float64(stat.Total)) * 100)
				}

				if currentPercent < rule.Min {
					shortfall := round1(rule.Min - currentPercent)
					linesNeeded := stat.Total - stat.Covered

					items = append(items, DebtItem{
						Name:      file,
						Type:      "file",
						Current:   currentPercent,
						Required:  rule.Min,
						Shortfall: shortfall,
						Lines:     linesNeeded,
					})
					totalDebt += shortfall
					totalLines += linesNeeded
					failCount++
				} else {
					passCount++
				}
			}
		}
	}

	// Sort by shortfall (highest first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Shortfall > items[j].Shortfall
	})

	// Calculate health score (0-100, higher is better)
	healthScore := 100.0
	if passCount+failCount > 0 {
		healthScore = round1((float64(passCount) / float64(passCount+failCount)) * 100)
	}

	return DebtResult{
		Items:       items,
		TotalDebt:   round1(totalDebt),
		TotalLines:  totalLines,
		HealthScore: healthScore,
	}, nil
}

// WatchCallback is called after each coverage run in watch mode.
type WatchCallback func(runNumber int, err error)

// Watch runs coverage tests in a loop, re-running when source files change.
func (s *Service) Watch(ctx context.Context, opts WatchOptions, watcher FileWatcher, callback WatchCallback) error {
	moduleRoot, err := s.DomainResolver.ModuleRoot(ctx)
	if err != nil {
		return err
	}

	if err := watcher.WatchDir(moduleRoot); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	// Prepare run options
	runOpts := RunOnlyOptions{
		ConfigPath: opts.ConfigPath,
		Profile:    opts.Profile,
		Domains:    opts.Domains,
		BuildFlags: opts.BuildFlags,
	}

	// Run immediately on start
	runNumber := 1
	runErr := s.RunOnly(ctx, runOpts)
	if callback != nil {
		callback(runNumber, runErr)
	}

	// Watch for changes
	events := watcher.Events(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, ok := <-events:
			if !ok {
				return nil
			}
			runNumber++
			runErr := s.RunOnly(ctx, runOpts)
			if callback != nil {
				callback(runNumber, runErr)
			}
		}
	}
}
