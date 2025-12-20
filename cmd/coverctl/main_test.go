package main

import (
	"bytes"
	"context"
	"errors"
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
	checkErr  error
	runErr    error
	detectErr error
	detectCfg application.Config
	reportErr error
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
