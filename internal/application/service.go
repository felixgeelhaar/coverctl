package application

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/felixgeelhaar/coverctl/internal/domain"
)

type Service struct {
	ConfigLoader   ConfigLoader
	Autodetector   Autodetector
	DomainResolver DomainResolver
	CoverageRunner CoverageRunner
	ProfileParser  ProfileParser
	Reporter       Reporter
	Out            io.Writer
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
	WriteConfig bool
	ConfigPath  string
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

	fileCoverage, err := s.ProfileParser.Parse(profile)
	if err != nil {
		return err
	}

	domainDirs, err := s.DomainResolver.Resolve(ctx, domains)
	if err != nil {
		return err
	}

	domainCoverage := AggregateByDomain(fileCoverage, domainDirs, cfg.Exclude, moduleRoot)
	result := domain.Evaluate(cfg.Policy, domainCoverage)
	result.Warnings = domainOverlapWarnings(domainDirs)

	if err := s.Reporter.Write(s.Out, result, opts.Output); err != nil {
		return err
	}
	if !result.Passed {
		return fmt.Errorf("policy violation")
	}
	return nil
}

func (s *Service) RunOnly(ctx context.Context, opts RunOnlyOptions) error {
	cfg, domains, err := s.loadOrDetect(opts.ConfigPath)
	if err != nil {
		return err
	}
	_, _ = cfg, domains
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

	fileCoverage, err := s.ProfileParser.Parse(opts.Profile)
	if err != nil {
		return err
	}

	domainDirs, err := s.DomainResolver.Resolve(ctx, domains)
	if err != nil {
		return err
	}

	domainCoverage := AggregateByDomain(fileCoverage, domainDirs, cfg.Exclude, moduleRoot)
	result := domain.Evaluate(cfg.Policy, domainCoverage)
	result.Warnings = domainOverlapWarnings(domainDirs)

	return s.Reporter.Write(s.Out, result, opts.Output)
}

func (s *Service) Detect(ctx context.Context, opts DetectOptions) (Config, error) {
	cfg, err := s.Autodetector.Detect()
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
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
func AggregateByDomain(files map[string]domain.CoverageStat, domainDirs map[string][]string, exclude []string, moduleRoot string) map[string]domain.CoverageStat {
	result := make(map[string]domain.CoverageStat, len(domainDirs))

	for file, stat := range files {
		if excluded(file, exclude) {
			continue
		}
		for domainName, dirs := range domainDirs {
			if matchesAnyDir(file, dirs, moduleRoot) {
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
