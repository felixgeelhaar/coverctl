package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
)

func TestWriteText(t *testing.T) {
	buf := new(bytes.Buffer)
	res := domain.Result{
		Passed: true,
		Domains: []domain.DomainResult{{
			Domain:   "core",
			Percent:  83.2,
			Required: 80,
			Status:   domain.StatusPass,
		}},
	}
	if err := (Writer{}).Write(buf, res, application.OutputText); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.Contains(buf.String(), "core") {
		t.Fatalf("expected domain in output")
	}
}

func TestWriteJSON(t *testing.T) {
	buf := new(bytes.Buffer)
	res := domain.Result{Passed: false}
	if err := (Writer{}).Write(buf, res, application.OutputJSON); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.Contains(buf.String(), "\"pass\": false") {
		t.Fatalf("expected JSON summary")
	}
}

func TestWriteWarningsText(t *testing.T) {
	buf := new(bytes.Buffer)
	res := domain.Result{
		Passed: true,
		Domains: []domain.DomainResult{{
			Domain:   "core",
			Percent:  90,
			Required: 80,
			Status:   domain.StatusPass,
		}},
		Warnings: []string{"shared directory used by core and api"},
	}
	if err := (Writer{}).Write(buf, res, application.OutputText); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.Contains(buf.String(), "Warnings:") {
		t.Fatalf("expected warnings section")
	}
}

func TestWriteWarningsJSON(t *testing.T) {
	buf := new(bytes.Buffer)
	res := domain.Result{
		Passed:   true,
		Warnings: []string{"shared directory used by core and api"},
	}
	if err := (Writer{}).Write(buf, res, application.OutputJSON); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.Contains(buf.String(), "\"warnings\"") {
		t.Fatalf("expected warnings field")
	}
}

func TestWriteFileRulesText(t *testing.T) {
	buf := new(bytes.Buffer)
	res := domain.Result{
		Passed: true,
		Files: []domain.FileResult{{
			File:     "internal/core/a.go",
			Percent:  88.8,
			Required: 90,
			Status:   domain.StatusFail,
		}},
	}
	if err := (Writer{}).Write(buf, res, application.OutputText); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.Contains(buf.String(), "File rules:") {
		t.Fatalf("expected file rules section")
	}
}

func TestWriteFileRulesJSON(t *testing.T) {
	buf := new(bytes.Buffer)
	res := domain.Result{
		Passed: true,
		Files: []domain.FileResult{{
			File:     "internal/core/a.go",
			Percent:  88.8,
			Required: 90,
			Status:   domain.StatusFail,
		}},
	}
	if err := (Writer{}).Write(buf, res, application.OutputJSON); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.Contains(buf.String(), "\"files\"") {
		t.Fatalf("expected files field")
	}
}
