package application

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/felixgeelhaar/coverctl/internal/domain"
)

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
}

func (f fakeResolver) Resolve(ctx context.Context, domains []domain.Domain) (map[string][]string, error) {
	return f.dirs, f.err
}

func (f fakeResolver) ModuleRoot(ctx context.Context) (string, error) { return f.moduleRoot, f.err }

type fakeRunner struct {
	profile string
	err     error
}

func (f fakeRunner) Run(ctx context.Context, opts RunOptions) (string, error) {
	return f.profile, f.err
}

type fakeParser struct {
	stats map[string]domain.CoverageStat
	err   error
}

func (f fakeParser) Parse(path string) (map[string]domain.CoverageStat, error) { return f.stats, f.err }

type fakeReporter struct {
	last domain.Result
	err  error
}

func (f *fakeReporter) Write(w io.Writer, result domain.Result, format OutputFormat) error {
	f.last = result
	return f.err
}

func TestServiceCheckPass(t *testing.T) {
	min := 80.0
	cfg := Config{Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "core", Match: []string{"./internal/core/..."}, Min: &min}}}}
	reporter := &fakeReporter{}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: true, cfg: cfg},
		Autodetector:   fakeAutodetector{},
		DomainResolver: fakeResolver{dirs: map[string][]string{"core": {"/repo/internal/core"}}, moduleRoot: "/repo"},
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
	cfg := Config{Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "core", Match: []string{"./internal/core/..."}, Min: &min}}}}
	reporter := &fakeReporter{}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: true, cfg: cfg},
		Autodetector:   fakeAutodetector{},
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
		t.Fatalf("expected fail")
	}
}

func TestServiceCheckWarnings(t *testing.T) {
	cfg := Config{Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{
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

func TestServiceReportUsesAutodetect(t *testing.T) {
	cfg := Config{Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "module", Match: []string{"./..."}}}}}
	reporter := &fakeReporter{}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: false},
		Autodetector:   fakeAutodetector{cfg: cfg},
		DomainResolver: fakeResolver{dirs: map[string][]string{"module": {"/repo"}}, moduleRoot: "/repo"},
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
	cfg := Config{Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "module", Match: []string{"./..."}}}}}
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
	cfg := Config{Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "module", Match: []string{"./..."}}}}}
	svc := &Service{
		ConfigLoader:   fakeConfigLoader{exists: true, cfg: cfg},
		Autodetector:   fakeAutodetector{},
		CoverageRunner: fakeRunner{profile: ".cover/coverage.out"},
	}
	if err := svc.RunOnly(context.Background(), RunOnlyOptions{ConfigPath: ".coverctl.yaml"}); err != nil {
		t.Fatalf("run only: %v", err)
	}
}

func TestServiceDetect(t *testing.T) {
	cfg := Config{Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "module", Match: []string{"./..."}}}}}
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
	cfg := Config{Policy: domain.Policy{DefaultMin: 80}}
	svc := &Service{
		ConfigLoader: fakeConfigLoader{exists: true, cfg: cfg},
	}
	if err := svc.Check(context.Background(), CheckOptions{ConfigPath: ".coverctl.yaml"}); err == nil {
		t.Fatalf("expected error for empty domains")
	}
}
