package application

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/coverctl/internal/domain"
)

var errSentinel = errors.New("sentinel")

type fakeConfigLoader struct {
	exists    bool
	cfg       Config
	existsErr error
	loadErr   error
}

func (f fakeConfigLoader) Exists(path string) (bool, error) {
	return f.exists, f.existsErr
}

func (f fakeConfigLoader) Load(path string) (Config, error) {
	return f.cfg, f.loadErr
}

type fakeAutodetector struct {
	cfg Config
	err error
}

func (f fakeAutodetector) Detect() (Config, error) { return f.cfg, f.err }

type fakeResolver struct {
	dirs       map[string][]string
	moduleRoot string
	err        error
	modulePath string
}

func (f fakeResolver) Resolve(ctx context.Context, domains []domain.Domain) (map[string][]string, error) {
	return f.dirs, f.err
}

func (f fakeResolver) ModuleRoot(ctx context.Context) (string, error) { return f.moduleRoot, f.err }

func (f fakeResolver) ModulePath(ctx context.Context) (string, error) {
	return f.modulePath, f.err
}

type fakeRunner struct {
	profile string
	err     error
}

func (f fakeRunner) Run(ctx context.Context, opts RunOptions) (string, error) {
	return f.profile, f.err
}

func (f fakeRunner) RunIntegration(ctx context.Context, opts IntegrationOptions) (string, error) {
	return f.profile, f.err
}

func (f fakeRunner) Name() string {
	return "fake"
}

func (f fakeRunner) Language() Language {
	return LanguageGo
}

func (f fakeRunner) Detect(projectDir string) bool {
	return true
}

type fakeParser struct {
	stats map[string]domain.CoverageStat
	err   error
}

func (f fakeParser) Parse(path string) (map[string]domain.CoverageStat, error) { return f.stats, f.err }

func (f fakeParser) ParseAll(paths []string) (map[string]domain.CoverageStat, error) {
	return f.stats, f.err
}

func (f fakeParser) Format() Format { return FormatGo }

type fakeReporter struct {
	last domain.Result
	err  error
}

func (f *fakeReporter) Write(w io.Writer, result domain.Result, format OutputFormat) error {
	f.last = result
	return f.err
}

type fakeDiffProvider struct {
	files []string
	err   error
}

func (f fakeDiffProvider) ChangedFiles(ctx context.Context, base string) ([]string, error) {
	return f.files, f.err
}

func TestServiceCheckPass(t *testing.T) {
	min := 80.0
	cfg := Config{Version: 1, Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "core", Match: []string{"./internal/core/..."}, Min: &min}}}}
	reporter := &fakeReporter{}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: true, cfg: cfg},
		Autodetector:   fakeAutodetector{},
		DomainResolver: fakeResolver{dirs: map[string][]string{"core": {"/repo/internal/core"}}, moduleRoot: "/repo", modulePath: "github.com/felixgeelhaar/coverctl"},
		CoverageRunner: fakeRunner{profile: ".cover/coverage.out"},
		ProfileParser:  fakeParser{stats: map[string]domain.CoverageStat{"internal/core/a.go": {Covered: 8, Total: 10}}},
		Reporter:       reporter,
		Out:            io.Discard,
	}

	if err := svc.Check(context.Background(), CheckOptions{ConfigPath: ".coverctl.yaml", Output: OutputText}); err != nil {
		t.Fatalf("check: %v", err)
	}
	if !reporter.last.Passed {
		t.Fatalf("expected pass")
	}
}

func TestServiceCheckFail(t *testing.T) {
	min := 90.0
	cfg := Config{Version: 1, Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "core", Match: []string{"./internal/core/..."}, Min: &min}}}}
	reporter := &fakeReporter{}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: true, cfg: cfg},
		Autodetector:   fakeAutodetector{},
		DomainResolver: fakeResolver{dirs: map[string][]string{"core": {"/repo/internal/core"}}, moduleRoot: "/repo", modulePath: "github.com/felixgeelhaar/coverctl"},
		CoverageRunner: fakeRunner{profile: ".cover/coverage.out"},
		ProfileParser:  fakeParser{stats: map[string]domain.CoverageStat{"internal/core/a.go": {Covered: 8, Total: 10}}},
		Reporter:       reporter,
		Out:            io.Discard,
	}

	if err := svc.Check(context.Background(), CheckOptions{ConfigPath: ".coverctl.yaml", Output: OutputText}); err == nil {
		t.Fatalf("expected policy violation")
	}
	if reporter.last.Passed {
		t.Fatalf("expected fail")
	}
}

func TestServiceCheckWarnings(t *testing.T) {
	cfg := Config{Version: 1, Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{
		{Name: "core", Match: []string{"./internal/core/..."}},
		{Name: "api", Match: []string{"./internal/api/..."}}}}}
	reporter := &fakeReporter{}
	svc := &Service{
		ConfigLoader: fakeConfigLoader{exists: true, cfg: cfg},
		Autodetector: fakeAutodetector{},
		DomainResolver: fakeResolver{
			dirs: map[string][]string{
				"core": {"/repo/internal/core"},
				"api":  {"/repo/internal/core"},
			},
			moduleRoot: "/repo",
			modulePath: "github.com/felixgeelhaar/coverctl",
		},
		CoverageRunner: fakeRunner{profile: ".cover/coverage.out"},
		ProfileParser:  fakeParser{stats: map[string]domain.CoverageStat{"internal/core/a.go": {Covered: 8, Total: 10}}},
		Reporter:       reporter,
		Out:            io.Discard,
	}

	if err := svc.Check(context.Background(), CheckOptions{ConfigPath: ".coverctl.yaml", Output: OutputText}); err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(reporter.last.Warnings) == 0 {
		t.Fatalf("expected warnings for overlap")
	}
}

func TestServiceCheckFileRulesFail(t *testing.T) {
	cfg := Config{
		Version: 1,
		Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{
			{Name: "core", Match: []string{"./internal/core/..."}},
		}},
		Files: []domain.FileRule{{Match: []string{"internal/core/*.go"}, Min: 90}},
	}
	reporter := &fakeReporter{}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: true, cfg: cfg},
		DomainResolver: fakeResolver{dirs: map[string][]string{"core": {"/repo/internal/core"}}, moduleRoot: "/repo"},
		CoverageRunner: fakeRunner{profile: ".cover/coverage.out"},
		ProfileParser:  fakeParser{stats: map[string]domain.CoverageStat{"internal/core/a.go": {Covered: 8, Total: 10}}},
		Reporter:       reporter,
		Out:            io.Discard,
	}

	if err := svc.Check(context.Background(), CheckOptions{ConfigPath: ".coverctl.yaml", Output: OutputText}); err == nil {
		t.Fatalf("expected policy violation")
	}
	if reporter.last.Passed {
		t.Fatalf("expected file rule to fail")
	}
	if len(reporter.last.Files) == 0 {
		t.Fatalf("expected file results")
	}
}

func TestServiceCheckDiffFiltersDomains(t *testing.T) {
	cfg := Config{
		Version: 1,
		Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{
			{Name: "core", Match: []string{"./internal/core/..."}},
			{Name: "api", Match: []string{"./internal/api/..."}},
		}},
		Diff: DiffConfig{Enabled: true, Base: "main"},
	}
	reporter := &fakeReporter{}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: true, cfg: cfg},
		DomainResolver: fakeResolver{dirs: map[string][]string{"core": {"/repo/internal/core"}, "api": {"/repo/internal/api"}}, moduleRoot: "/repo"},
		CoverageRunner: fakeRunner{profile: ".cover/coverage.out"},
		ProfileParser: fakeParser{stats: map[string]domain.CoverageStat{
			"internal/core/a.go": {Covered: 8, Total: 10},
			"internal/api/b.go":  {Covered: 1, Total: 10},
		}},
		DiffProvider: fakeDiffProvider{files: []string{"internal/core/a.go"}},
		Reporter:     reporter,
		Out:          io.Discard,
	}

	if err := svc.Check(context.Background(), CheckOptions{ConfigPath: ".coverctl.yaml", Output: OutputText}); err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(reporter.last.Domains) != 1 || reporter.last.Domains[0].Domain != "core" {
		t.Fatalf("expected only core domain in diff mode")
	}
}

func TestServiceReportUsesAutodetect(t *testing.T) {
	cfg := Config{Version: 1, Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "module", Match: []string{"./..."}}}}}
	reporter := &fakeReporter{}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: false},
		Autodetector:   fakeAutodetector{cfg: cfg},
		DomainResolver: fakeResolver{dirs: map[string][]string{"module": {"/repo"}}, moduleRoot: "/repo", modulePath: "github.com/felixgeelhaar/coverctl"},
		ProfileParser:  fakeParser{stats: map[string]domain.CoverageStat{"main.go": {Covered: 1, Total: 1}}},
		Reporter:       reporter,
		Out:            io.Discard,
	}

	if err := svc.Report(context.Background(), ReportOptions{ConfigPath: ".coverctl.yaml", Output: OutputText, Profile: "coverage.out"}); err != nil {
		t.Fatalf("report: %v", err)
	}
	if !reporter.last.Passed {
		t.Fatalf("expected pass")
	}
}

func TestServiceReportError(t *testing.T) {
	cfg := Config{Version: 1, Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "module", Match: []string{"./..."}}}}}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: true, cfg: cfg},
		DomainResolver: fakeResolver{err: errors.New("resolver")},
		ProfileParser:  fakeParser{stats: map[string]domain.CoverageStat{}},
		Reporter:       &fakeReporter{},
		Out:            io.Discard,
	}

	if err := svc.Report(context.Background(), ReportOptions{ConfigPath: ".coverctl.yaml", Output: OutputText, Profile: "coverage.out"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestReporterWrites(t *testing.T) {
	var buf bytes.Buffer
	reporter := &fakeReporter{}
	_ = reporter.Write(&buf, domain.Result{Passed: true}, OutputText)
	if reporter.last.Passed != true {
		t.Fatalf("expected reporter to capture result")
	}
}

func TestServiceRunOnly(t *testing.T) {
	cfg := Config{Version: 1, Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "module", Match: []string{"./..."}}}}}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: true, cfg: cfg},
		Autodetector:   fakeAutodetector{},
		CoverageRunner: fakeRunner{profile: ".cover/coverage.out"},
	}
	if err := svc.RunOnly(context.Background(), RunOnlyOptions{ConfigPath: ".coverctl.yaml"}); err != nil {
		t.Fatalf("run only: %v", err)
	}
}

func TestServiceRunOnlyError(t *testing.T) {
	cfg := Config{Version: 1, Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "module", Match: []string{"./..."}}}}}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: true, cfg: cfg},
		Autodetector:   fakeAutodetector{},
		CoverageRunner: fakeRunner{err: errSentinel},
	}
	if err := svc.RunOnly(context.Background(), RunOnlyOptions{ConfigPath: ".coverctl.yaml"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestServiceCheckRunnerError(t *testing.T) {
	cfg := Config{Version: 1, Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "module", Match: []string{"./..."}}}}}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: true, cfg: cfg},
		Autodetector:   fakeAutodetector{},
		DomainResolver: fakeResolver{dirs: map[string][]string{"module": {"/repo"}}, moduleRoot: "/repo", modulePath: "github.com/felixgeelhaar/coverctl"},
		CoverageRunner: fakeRunner{err: errSentinel},
		ProfileParser:  fakeParser{stats: map[string]domain.CoverageStat{}},
		Reporter:       &fakeReporter{},
		Out:            io.Discard,
	}
	if err := svc.Check(context.Background(), CheckOptions{ConfigPath: ".coverctl.yaml"}); err == nil {
		t.Fatalf("expected runner error")
	}
}

func TestServiceCheckProfileParserError(t *testing.T) {
	cfg := Config{Version: 1, Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "module", Match: []string{"./..."}}}}}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: true, cfg: cfg},
		DomainResolver: fakeResolver{dirs: map[string][]string{"module": {"/repo"}}, moduleRoot: "/repo", modulePath: "github.com/felixgeelhaar/coverctl"},
		CoverageRunner: fakeRunner{profile: ".cover/coverage.out"},
		ProfileParser:  fakeParser{err: errSentinel},
		Reporter:       &fakeReporter{},
		Out:            io.Discard,
	}
	if err := svc.Check(context.Background(), CheckOptions{ConfigPath: ".coverctl.yaml"}); err == nil {
		t.Fatalf("expected parser error")
	}
}

func TestServiceCheckResolveError(t *testing.T) {
	cfg := Config{Version: 1, Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "module", Match: []string{"./..."}}}}}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: true, cfg: cfg},
		DomainResolver: fakeResolver{err: errSentinel},
		CoverageRunner: fakeRunner{profile: ".cover/coverage.out"},
		ProfileParser:  fakeParser{stats: map[string]domain.CoverageStat{}},
		Reporter:       &fakeReporter{},
		Out:            io.Discard,
	}
	if err := svc.Check(context.Background(), CheckOptions{ConfigPath: ".coverctl.yaml"}); err == nil {
		t.Fatalf("expected resolve error")
	}
}

func TestServiceReportParserError(t *testing.T) {
	cfg := Config{Version: 1, Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "module", Match: []string{"./..."}}}}}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: true, cfg: cfg},
		DomainResolver: fakeResolver{dirs: map[string][]string{"module": {"/repo"}}, moduleRoot: "/repo", modulePath: "github.com/felixgeelhaar/coverctl"},
		ProfileParser:  fakeParser{err: errSentinel},
		Reporter:       &fakeReporter{},
		Out:            io.Discard,
	}
	if err := svc.Report(context.Background(), ReportOptions{ConfigPath: ".coverctl.yaml", Profile: "coverage.out"}); err == nil {
		t.Fatalf("expected parser error")
	}
}

func TestServiceDetectError(t *testing.T) {
	svc := &Service{
		Autodetector: fakeAutodetector{err: errSentinel},
	}
	if _, err := svc.Detect(context.Background(), DetectOptions{}); err == nil {
		t.Fatalf("expected detect error")
	}
}

func TestLoadOrDetectExistsError(t *testing.T) {
	svc := &Service{
		ConfigLoader: fakeConfigLoader{existsErr: errSentinel},
		Autodetector: fakeAutodetector{},
	}
	if _, _, err := svc.loadOrDetect(".coverctl.yaml"); err == nil {
		t.Fatalf("expected exists error")
	}
}

func TestServiceDetectUsesAutodetector(t *testing.T) {
	cfg := Config{Version: 1, Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "module", Match: []string{"./..."}}}}}
	svc := &Service{
		Autodetector: fakeAutodetector{cfg: cfg},
	}
	got, err := svc.Detect(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if got.Policy.DefaultMin != cfg.Policy.DefaultMin {
		t.Fatalf("unexpected config: %+v", got)
	}
}

func TestLoadOrDetectFailsWithoutDomains(t *testing.T) {
	svc := &Service{
		ConfigLoader: fakeConfigLoader{exists: true, cfg: Config{Version: 1, Policy: domain.Policy{Domains: nil}}},
		Autodetector: fakeAutodetector{cfg: Config{Version: 1}},
	}
	if _, _, err := svc.loadOrDetect(".coverctl.yaml"); err == nil {
		t.Fatalf("expected error when no domains configured")
	}
}

func TestServiceDetect(t *testing.T) {
	cfg := Config{Version: 1, Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "module", Match: []string{"./..."}}}}}
	svc := &Service{Autodetector: fakeAutodetector{cfg: cfg}}
	got, err := svc.Detect(context.Background(), DetectOptions{})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(got.Policy.Domains) != 1 {
		t.Fatalf("expected domains")
	}
}

func TestServiceLoadOrDetectNoDomains(t *testing.T) {
	cfg := Config{Version: 1, Policy: domain.Policy{DefaultMin: 80}}
	svc := &Service{
		ConfigLoader: fakeConfigLoader{exists: true, cfg: cfg},
	}
	if err := svc.Check(context.Background(), CheckOptions{ConfigPath: ".coverctl.yaml"}); err == nil {
		t.Fatalf("expected error for empty domains")
	}
}

func TestAggregateByDomainNormalizesAndExcludes(t *testing.T) {
	moduleRoot := filepath.Join(t.TempDir(), "repo")
	modulePath := "github.com/felixgeelhaar/coverctl"
	files := map[string]domain.CoverageStat{
		modulePath + "/internal/core/a.go":                       {Covered: 2, Total: 3},
		filepath.Join(moduleRoot, "internal", "ignored", "b.go"): {Covered: 1, Total: 1},
	}
	domainDirs := map[string][]string{
		"core": {filepath.Join(moduleRoot, "internal", "core")},
	}
	result := AggregateByDomain(files, domainDirs, []string{"internal/ignored/*"}, moduleRoot, modulePath, nil)
	stat, ok := result["core"]
	if !ok {
		t.Fatalf("expected core domain coverage")
	}
	if stat.Covered != 2 || stat.Total != 3 {
		t.Fatalf("unexpected core stats: %+v", stat)
	}
	if _, ok := result["ignored"]; ok {
		t.Fatalf("excluded file should not contribute")
	}
}

func TestExcludedPatterns(t *testing.T) {
	if !excluded("internal/core/a.go", []string{"internal/core/*"}) {
		t.Fatalf("expected match for pattern")
	}
	if excluded("internal/core/a.go", []string{"pkg/*"}) {
		t.Fatalf("expected no match for unrelated pattern")
	}
}

func TestMatchesAnyDirModuleRootAndRelatives(t *testing.T) {
	moduleRoot := filepath.Join(t.TempDir(), "repo")
	file := filepath.Join(moduleRoot, "internal", "core", "a.go")
	dirs := []string{
		filepath.Join(moduleRoot, "internal", "core"),
		moduleRoot,
	}
	if !matchesAnyDir(file, dirs, moduleRoot) {
		t.Fatalf("expected file to match directory list")
	}
}

func TestNormalizeCoverageFileVariousCases(t *testing.T) {
	moduleRoot := filepath.Join(t.TempDir(), "repo")
	modulePath := "github.com/felixgeelhaar/coverctl"

	if got := normalizeCoverageFile(modulePath, modulePath, moduleRoot); got != filepath.Clean(moduleRoot) {
		t.Fatalf("expected module path to map to module root, got %s", got)
	}
	relFile := modulePath + "/internal/core/a.go"
	expected := filepath.Join(moduleRoot, "internal", "core", "a.go")
	if got := normalizeCoverageFile(relFile, modulePath, moduleRoot); got != expected {
		t.Fatalf("expected normalized path %s, got %s", expected, got)
	}
	absFile := filepath.Join(moduleRoot, "internal", "pkg", "b.go")
	if got := normalizeCoverageFile(absFile, "", moduleRoot); got != absFile {
		t.Fatalf("expected absolute path to remain, got %s", got)
	}
}

func TestModuleRelativePath(t *testing.T) {
	moduleRoot := filepath.Join(t.TempDir(), "repo")
	path := filepath.Join(moduleRoot, "internal", "core", "a.go")
	if got := moduleRelativePath(path, moduleRoot); got != filepath.Join("internal", "core", "a.go") {
		t.Fatalf("expected relative path")
	}
	outside := filepath.Join(filepath.Dir(moduleRoot), "other.go")
	if got := moduleRelativePath(outside, moduleRoot); got != filepath.Clean(filepath.Join("..", "other.go")) {
		t.Fatalf("expected clean outside path")
	}
}

func TestDomainOverlapWarnings(t *testing.T) {
	domainDirs := map[string][]string{
		"core": {"/repo/internal/core"},
		"api":  {"/repo/internal/core"},
	}
	warnings := domainOverlapWarnings(domainDirs)
	if len(warnings) != 1 {
		t.Fatalf("expected one warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "api, core") {
		t.Fatalf("unexpected warning message: %s", warnings[0])
	}
}

func TestFilterDomainsByNames(t *testing.T) {
	domains := []domain.Domain{
		{Name: "core", Match: []string{"./internal/core/..."}},
		{Name: "api", Match: []string{"./internal/api/..."}},
		{Name: "cli", Match: []string{"./cmd/..."}},
	}

	t.Run("empty filter returns all", func(t *testing.T) {
		result := filterDomainsByNames(domains, nil)
		if len(result) != 3 {
			t.Fatalf("expected 3 domains, got %d", len(result))
		}
	})

	t.Run("filter single domain", func(t *testing.T) {
		result := filterDomainsByNames(domains, []string{"core"})
		if len(result) != 1 {
			t.Fatalf("expected 1 domain, got %d", len(result))
		}
		if result[0].Name != "core" {
			t.Fatalf("expected core, got %s", result[0].Name)
		}
	})

	t.Run("filter multiple domains", func(t *testing.T) {
		result := filterDomainsByNames(domains, []string{"core", "cli"})
		if len(result) != 2 {
			t.Fatalf("expected 2 domains, got %d", len(result))
		}
	})

	t.Run("filter non-existent domain", func(t *testing.T) {
		result := filterDomainsByNames(domains, []string{"nonexistent"})
		if len(result) != 0 {
			t.Fatalf("expected 0 domains, got %d", len(result))
		}
	})
}

func TestServiceCheckWithDomainFilter(t *testing.T) {
	min := 80.0
	cfg := Config{
		Version: 1,
		Policy: domain.Policy{
			DefaultMin: 80,
			Domains: []domain.Domain{
				{Name: "core", Match: []string{"./internal/core/..."}, Min: &min},
				{Name: "api", Match: []string{"./internal/api/..."}, Min: &min},
			},
		},
	}
	reporter := &fakeReporter{}
	svc := &Service{
		ConfigLoader: fakeConfigLoader{exists: true, cfg: cfg},
		Autodetector: fakeAutodetector{},
		DomainResolver: fakeResolver{
			dirs: map[string][]string{
				"core": {"/repo/internal/core"},
				"api":  {"/repo/internal/api"},
			},
			moduleRoot: "/repo",
			modulePath: "github.com/felixgeelhaar/coverctl",
		},
		CoverageRunner: fakeRunner{profile: ".cover/coverage.out"},
		ProfileParser: fakeParser{stats: map[string]domain.CoverageStat{
			"internal/core/a.go": {Covered: 8, Total: 10},
			"internal/api/b.go":  {Covered: 5, Total: 10},
		}},
		Reporter: reporter,
		Out:      io.Discard,
	}

	// Filter to only core domain
	err := svc.Check(context.Background(), CheckOptions{
		ConfigPath: ".coverctl.yaml",
		Output:     OutputText,
		Domains:    []string{"core"},
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(reporter.last.Domains) != 1 {
		t.Fatalf("expected 1 domain in result, got %d", len(reporter.last.Domains))
	}
	if reporter.last.Domains[0].Domain != "core" {
		t.Fatalf("expected core domain, got %s", reporter.last.Domains[0].Domain)
	}
}

func TestServiceCheckDomainFilterNoMatch(t *testing.T) {
	cfg := Config{
		Version: 1,
		Policy: domain.Policy{
			DefaultMin: 80,
			Domains: []domain.Domain{
				{Name: "core", Match: []string{"./internal/core/..."}},
			},
		},
	}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: true, cfg: cfg},
		CoverageRunner: fakeRunner{profile: "test.out"},
		Out:            io.Discard,
	}

	err := svc.Check(context.Background(), CheckOptions{
		ConfigPath: ".coverctl.yaml",
		Domains:    []string{"nonexistent"},
	})
	if err == nil {
		t.Fatalf("expected error for non-matching domain filter")
	}
	if !strings.Contains(err.Error(), "no matching domains") {
		t.Fatalf("unexpected error: %v", err)
	}
}
