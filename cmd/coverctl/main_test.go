package main

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
	cfg := application.Config{Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "core", Match: []string{"./internal/core/..."}, Min: &min}}}}
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

func TestRunUsage(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"coverctl"}, &out, &out, fakeService{})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestRunUnknown(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"coverctl", "nope"}, &out, &out, fakeService{})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestRunCheck(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"coverctl", "check"}, &out, &out, fakeService{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestRunCheckError(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"coverctl", "check"}, &out, &out, fakeService{checkErr: errSentinel})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
}

func TestRunDetectWriteConfig(t *testing.T) {
	var out bytes.Buffer
	path := filepath.Join(t.TempDir(), ".coverctl.yaml")
	code := run([]string{"coverctl", "detect", "--write-config", "--config", path}, &out, &out, fakeService{detectCfg: minimalConfig()})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file: %v", err)
	}
}

func TestRunReportError(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"coverctl", "report"}, &out, &out, fakeService{reportErr: errSentinel})
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
}

func TestRunReportSuccess(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"coverctl", "report"}, &out, &out, fakeService{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestRunIgnore(t *testing.T) {
	var out bytes.Buffer
	cfg := application.Config{
		Exclude: []string{"internal/generated/proto/*"},
	}
	domains := []domain.Domain{{Name: "proto", Match: []string{"./internal/generated/proto/..."}}}
	code := run([]string{"coverctl", "ignore", "--config", "custom.yaml"}, &out, &out, fakeService{ignoreCfg: cfg, ignoreDomains: domains})
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
	code := run([]string{"coverctl", "ignore"}, &out, &out, fakeService{ignoreErr: errSentinel})
	if code != 4 {
		t.Fatalf("expected exit 4, got %d", code)
	}
}

func TestRunRunSuccess(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"coverctl", "run"}, &out, &out, fakeService{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestRunInitCreatesFile(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	path := filepath.Join(dir, ".coverctl.yaml")
	code := run([]string{"coverctl", "init", "--config", path, "--no-interactive"}, &out, &out, fakeService{detectCfg: minimalConfig()})
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
	code := run([]string{"coverctl", "init", "--config", path}, &out, &out, fakeService{detectCfg: minimalConfig()})
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
	code := run([]string{"coverctl", "init", "--config", path}, &out, &out, fakeService{detectCfg: minimalConfig()})
	if code != 0 {
		t.Fatalf("expected exit 0 when wizard cancels, got %d", code)
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("config should not exist when wizard cancels")
	}
	if !strings.Contains(out.String(), "Init cancelled") {
		t.Fatalf("expected cancellation message: %s", out.String())
	}
}

func TestWriteConfigFileStdout(t *testing.T) {
	min := 80.0
	cfg := application.Config{Policy: domain.Policy{DefaultMin: 80, Domains: []domain.Domain{{Name: "core", Match: []string{"./internal/core/..."}, Min: &min}}}}
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
	code := run([]string{"coverctl", "detect"}, &out, &out, fakeService{detectCfg: minimalConfig()})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out.String(), "policy:") {
		t.Fatalf("expected config output")
	}
}

func TestRunRunError(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"coverctl", "run"}, &out, &out, fakeService{runErr: errSentinel})
	if code != 3 {
		t.Fatalf("expected exit 3, got %d", code)
	}
}

var errSentinel = errors.New("sentinel")

func minimalConfig() application.Config {
	return application.Config{
		Policy: domain.Policy{
			DefaultMin: 80,
			Domains:    []domain.Domain{{Name: "module", Match: []string{"./..."}}},
		},
	}
}
