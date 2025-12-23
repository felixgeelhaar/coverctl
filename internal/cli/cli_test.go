package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
)

func TestOutputValueSet(t *testing.T) {
	val := outputValue(application.OutputText)
	if err := val.Set("json"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if string(val) != "json" {
		t.Fatalf("expected json")
	}
	if err := val.Set("bad"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestWriteConfigFile(t *testing.T) {
	min := 80.0
	cfg := application.Config{Version: 1, Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "core", Match: []string{"./internal/core/..."}, Min: &min}}}}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := writeConfigFile(path, cfg, os.Stdout, false); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file: %v", err)
	}
}

type fakeService struct {
	checkErr      error
	runErr        error
	detectErr     error
	detectCfg     application.Config
	reportErr     error
	ignoreErr     error
	ignoreCfg     application.Config
	ignoreDomains []domain.Domain
	badgeErr      error
	badgeResult   application.BadgeResult
	trendErr      error
	trendResult   application.TrendResult
	recordErr     error
	suggestErr    error
	suggestResult application.SuggestResult
}

func (f fakeService) Check(_ context.Context, _ application.CheckOptions) error { return f.checkErr }
func (f fakeService) RunOnly(_ context.Context, _ application.RunOnlyOptions) error {
	return f.runErr
}
func (f fakeService) Detect(_ context.Context, _ application.DetectOptions) (application.Config, error) {
	if f.detectErr != nil {
		return application.Config{}, f.detectErr
	}
	return f.detectCfg, nil
}
func (f fakeService) Report(_ context.Context, _ application.ReportOptions) error { return f.reportErr }
func (f fakeService) Ignore(_ context.Context, _ application.IgnoreOptions) (application.Config, []domain.Domain, error) {
	if f.ignoreErr != nil {
		return application.Config{}, nil, f.ignoreErr
	}
	return f.ignoreCfg, f.ignoreDomains, nil
}
func (f fakeService) Badge(_ context.Context, _ application.BadgeOptions) (application.BadgeResult, error) {
	if f.badgeErr != nil {
		return application.BadgeResult{}, f.badgeErr
	}
	return f.badgeResult, nil
}
func (f fakeService) Trend(_ context.Context, _ application.TrendOptions, _ application.HistoryStore) (application.TrendResult, error) {
	if f.trendErr != nil {
		return application.TrendResult{}, f.trendErr
	}
	return f.trendResult, nil
}
func (f fakeService) Record(_ context.Context, _ application.RecordOptions, _ application.HistoryStore) error {
	return f.recordErr
}
func (f fakeService) Suggest(_ context.Context, _ application.SuggestOptions) (application.SuggestResult, error) {
	if f.suggestErr != nil {
		return application.SuggestResult{}, f.suggestErr
	}
	return f.suggestResult, nil
}
func (f fakeService) Watch(_ context.Context, _ application.WatchOptions, _ application.FileWatcher, _ application.WatchCallback) error {
	return nil
}
func (f fakeService) Debt(_ context.Context, _ application.DebtOptions) (application.DebtResult, error) {
	return application.DebtResult{HealthScore: 100}, nil
}

func TestRunUsage(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"coverctl"}, &out, &out, fakeService{})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestRunUnknown(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"coverctl", "nope"}, &out, &out, fakeService{})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestRunCheck(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"coverctl", "check"}, &out, &out, fakeService{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestRunCheckError(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"coverctl", "check"}, &out, &out, fakeService{checkErr: errSentinel})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
}

func TestRunDetectWriteConfig(t *testing.T) {
	var out bytes.Buffer
	path := filepath.Join(t.TempDir(), ".coverctl.yaml")
	code := Run([]string{"coverctl", "detect", "--write-config", "--config", path}, &out, &out, fakeService{detectCfg: minimalConfig()})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file: %v", err)
	}
}

func TestRunReportError(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"coverctl", "report"}, &out, &out, fakeService{reportErr: errSentinel})
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
}

func TestRunReportSuccess(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"coverctl", "report"}, &out, &out, fakeService{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestRunIgnore(t *testing.T) {
	var out bytes.Buffer
	cfg := application.Config{
		Version: 1,
		Exclude: []string{"internal/generated/proto/*"},
	}
	domains := []domain.Domain{{Name: "proto", Match: []string{"./internal/generated/proto/..."}}}
	code := Run([]string{"coverctl", "ignore", "--config", "custom.yaml"}, &out, &out, fakeService{ignoreCfg: cfg, ignoreDomains: domains})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	got := out.String()
	if !strings.Contains(got, "internal/generated/proto/*") || !strings.Contains(got, "proto (matches: ./internal/generated/proto/...)") {
		t.Fatalf("unexpected output: %s", got)
	}
}

func TestRunIgnoreError(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"coverctl", "ignore"}, &out, &out, fakeService{ignoreErr: errSentinel})
	if code != 4 {
		t.Fatalf("expected exit 4, got %d", code)
	}
}

func TestRunRunSuccess(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"coverctl", "run"}, &out, &out, fakeService{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestRunInitCreatesFile(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	path := filepath.Join(dir, ".coverctl.yaml")
	code := Run([]string{"coverctl", "init", "--config", path, "--no-interactive"}, &out, &out, fakeService{detectCfg: minimalConfig()})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file: %v", err)
	}
}

func TestRunInitInteractiveBranch(t *testing.T) {
	old := initWizard
	defer func() { initWizard = old }()
	called := false
	initWizard = func(cfg application.Config, stdout io.Writer, stdin io.Reader) (application.Config, bool, error) {
		called = true
		return cfg, true, nil
	}
	dir := t.TempDir()
	var out bytes.Buffer
	path := filepath.Join(dir, ".coverctl.yaml")
	code := Run([]string{"coverctl", "init", "--config", path}, &out, &out, fakeService{detectCfg: minimalConfig()})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !called {
		t.Fatalf("expected interactive wizard to run")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file: %v", err)
	}
}

func TestRunInitInteractiveCancelled(t *testing.T) {
	old := initWizard
	defer func() { initWizard = old }()
	initWizard = func(cfg application.Config, stdout io.Writer, stdin io.Reader) (application.Config, bool, error) {
		return cfg, false, nil
	}
	dir := t.TempDir()
	var out bytes.Buffer
	path := filepath.Join(dir, ".coverctl.yaml")
	code := Run([]string{"coverctl", "init", "--config", path}, &out, &out, fakeService{detectCfg: minimalConfig()})
	if code != 0 {
		t.Fatalf("expected exit 0 when wizard cancels, got %d", code)
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("config should not exist when wizard cancels")
	}
	if !strings.Contains(out.String(), "Init canceled") {
		t.Fatalf("expected cancellation message: %s", out.String())
	}
}

func TestRunInitWizardError(t *testing.T) {
	old := initWizard
	defer func() { initWizard = old }()
	initWizard = func(cfg application.Config, stdout io.Writer, stdin io.Reader) (application.Config, bool, error) {
		return cfg, false, errors.New("wizard failed")
	}
	dir := t.TempDir()
	var out bytes.Buffer
	path := filepath.Join(dir, ".coverctl.yaml")
	code := Run([]string{"coverctl", "init", "--config", path}, &out, &out, fakeService{detectCfg: minimalConfig()})
	if code != 5 {
		t.Fatalf("expected exit 5, got %d", code)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no config file when wizard errors")
	}
	if !strings.Contains(out.String(), "wizard failed") {
		t.Fatalf("expected wizard error printed")
	}
}

func TestWriteConfigFileStdout(t *testing.T) {
	min := 80.0
	cfg := application.Config{Version: 1, Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "core", Match: []string{"./internal/core/..."}, Min: &min}}}}
	var out bytes.Buffer
	if err := writeConfigFile("-", cfg, &out, true); err != nil {
		t.Fatalf("write to stdout: %v", err)
	}
	if !strings.Contains(out.String(), "policy:") {
		t.Fatalf("expected config output")
	}
}

func TestOutputValueString(t *testing.T) {
	val := outputValue("text")
	if val.String() != "text" {
		t.Fatalf("expected string value")
	}
}

func TestRunDetectStdout(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"coverctl", "detect"}, &out, &out, fakeService{detectCfg: minimalConfig()})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out.String(), "policy:") {
		t.Fatalf("expected config output")
	}
}

func TestRunRunError(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"coverctl", "run"}, &out, &out, fakeService{runErr: errSentinel})
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
}

var errSentinel = errors.New("sentinel")

func minimalConfig() application.Config {
	return application.Config{
		Version: 1,
		Policy: domain.Policy{
			DefaultMin: 80,
			Domains:    []domain.Domain{{Name: "module", Match: []string{"./..."}}},
		},
	}
}

func TestDomainListFlag(t *testing.T) {
	var dl domainList

	t.Run("empty string", func(t *testing.T) {
		if dl.String() != "" {
			t.Fatalf("expected empty string, got %s", dl.String())
		}
	})

	t.Run("append single", func(t *testing.T) {
		if err := dl.Set("core"); err != nil {
			t.Fatalf("set: %v", err)
		}
		if len(dl) != 1 || dl[0] != "core" {
			t.Fatalf("expected [core], got %v", dl)
		}
	})

	t.Run("append multiple", func(t *testing.T) {
		if err := dl.Set("api"); err != nil {
			t.Fatalf("set: %v", err)
		}
		if len(dl) != 2 {
			t.Fatalf("expected 2 domains, got %d", len(dl))
		}
		if dl.String() != "core,api" {
			t.Fatalf("expected 'core,api', got %s", dl.String())
		}
	})
}

func TestRunCheckWithDomainFlag(t *testing.T) {
	var out bytes.Buffer
	// The domain flag should be parsed without error
	code := Run([]string{"coverctl", "check", "--domain", "core", "--domain", "api"}, &out, &out, fakeService{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestRunBadge(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "coverage.svg")
	var out bytes.Buffer
	code := Run([]string{"coverctl", "badge", "--output", outputPath}, &out, &out, fakeService{badgeResult: application.BadgeResult{Percent: 85.5}})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected badge file: %v", err)
	}
	if !strings.Contains(out.String(), "Badge written") {
		t.Fatalf("expected success message")
	}
}

func TestRunBadgeError(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "coverage.svg")
	var out bytes.Buffer
	code := Run([]string{"coverctl", "badge", "--output", outputPath}, &out, &out, fakeService{badgeErr: errSentinel})
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
}

func TestRunTrend(t *testing.T) {
	var out bytes.Buffer
	trendResult := application.TrendResult{
		Current:  85.0,
		Previous: 80.0,
		Trend:    domain.Trend{Direction: domain.TrendUp, Delta: 5.0},
		Entries:  []domain.HistoryEntry{{Overall: 80.0}},
		ByDomain: map[string]domain.Trend{
			"core": {Direction: domain.TrendUp, Delta: 3.0},
		},
	}
	code := Run([]string{"coverctl", "trend"}, &out, &out, fakeService{trendResult: trendResult})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out.String(), "Coverage Trend") {
		t.Fatalf("expected trend output, got: %s", out.String())
	}
}

func TestRunTrendError(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"coverctl", "trend"}, &out, &out, fakeService{trendErr: errSentinel})
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
}

func TestRunRecord(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"coverctl", "record"}, &out, &out, fakeService{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out.String(), "Coverage recorded") {
		t.Fatalf("expected record success message, got: %s", out.String())
	}
}

func TestRunRecordError(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"coverctl", "record"}, &out, &out, fakeService{recordErr: errSentinel})
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
}

func TestRunSuggest(t *testing.T) {
	var out bytes.Buffer
	suggestResult := application.SuggestResult{
		Suggestions: []application.Suggestion{
			{Domain: "core", CurrentPercent: 85.0, CurrentMin: 80.0, SuggestedMin: 83.0, Reason: "based on current coverage"},
		},
		Config: minimalConfig(),
	}
	code := Run([]string{"coverctl", "suggest"}, &out, &out, fakeService{suggestResult: suggestResult})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out.String(), "Threshold Suggestions") {
		t.Fatalf("expected suggestion output, got: %s", out.String())
	}
}

func TestRunSuggestError(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"coverctl", "suggest"}, &out, &out, fakeService{suggestErr: errSentinel})
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
}

func TestRunSuggestWithStrategy(t *testing.T) {
	var out bytes.Buffer
	suggestResult := application.SuggestResult{
		Suggestions: []application.Suggestion{
			{Domain: "core", CurrentPercent: 85.0, CurrentMin: 80.0, SuggestedMin: 90.0, Reason: "aggressive target"},
		},
		Config: minimalConfig(),
	}
	code := Run([]string{"coverctl", "suggest", "--strategy", "aggressive"}, &out, &out, fakeService{suggestResult: suggestResult})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}
