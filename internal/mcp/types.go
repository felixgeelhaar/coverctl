// Package mcp provides Model Context Protocol server implementation for coverctl.
package mcp

import (
	"context"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
)

// Service defines the application operations needed by MCP.
// This interface allows for easy mocking in tests.
type Service interface {
	// Tools (actions that may have side effects)
	CheckResult(ctx context.Context, opts application.CheckOptions) (domain.Result, error)
	ReportResult(ctx context.Context, opts application.ReportOptions) (domain.Result, error)
	Record(ctx context.Context, opts application.RecordOptions, store application.HistoryStore) error

	// Resources (read-only queries)
	Debt(ctx context.Context, opts application.DebtOptions) (application.DebtResult, error)
	Trend(ctx context.Context, opts application.TrendOptions, store application.HistoryStore) (application.TrendResult, error)
	Suggest(ctx context.Context, opts application.SuggestOptions) (application.SuggestResult, error)
	Detect(ctx context.Context, opts application.DetectOptions) (application.Config, error)
}

// Config holds MCP server configuration.
type Config struct {
	ConfigPath  string // Path to .coverctl.yaml (default: ".coverctl.yaml")
	HistoryPath string // Path to history file (default: ".cover/history.json")
	ProfilePath string // Path to coverage profile (default: ".cover/coverage.out")
}

// DefaultConfig returns configuration with default values.
func DefaultConfig() Config {
	return Config{
		ConfigPath:  ".coverctl.yaml",
		HistoryPath: ".cover/history.json",
		ProfilePath: ".cover/coverage.out",
	}
}

// CheckInput defines the input parameters for the check tool.
type CheckInput struct {
	ConfigPath string   `json:"configPath,omitempty" jsonschema:"description=Path to .coverctl.yaml config file"`
	Profile    string   `json:"profile,omitempty" jsonschema:"description=Coverage profile output path"`
	Domains    []string `json:"domains,omitempty" jsonschema:"description=Filter to specific domains"`
	FailUnder  *float64 `json:"failUnder,omitempty" jsonschema:"description=Fail if coverage below threshold"`
	Ratchet    bool     `json:"ratchet,omitempty" jsonschema:"description=Fail if coverage decreases"`
}

// ReportInput defines the input parameters for the report tool.
type ReportInput struct {
	ConfigPath    string   `json:"configPath,omitempty" jsonschema:"description=Path to .coverctl.yaml config file"`
	Profile       string   `json:"profile,omitempty" jsonschema:"description=Path to existing coverage profile"`
	Domains       []string `json:"domains,omitempty" jsonschema:"description=Filter to specific domains"`
	ShowUncovered bool     `json:"showUncovered,omitempty" jsonschema:"description=Show only files with 0% coverage"`
	DiffRef       string   `json:"diffRef,omitempty" jsonschema:"description=Git ref for diff-based filtering"`
}

// RecordInput defines the input parameters for the record tool.
type RecordInput struct {
	ConfigPath  string `json:"configPath,omitempty" jsonschema:"description=Path to .coverctl.yaml config file"`
	Profile     string `json:"profile,omitempty" jsonschema:"description=Path to coverage profile"`
	HistoryPath string `json:"historyPath,omitempty" jsonschema:"description=Path to history file"`
	Commit      string `json:"commit,omitempty" jsonschema:"description=Git commit SHA"`
	Branch      string `json:"branch,omitempty" jsonschema:"description=Git branch name"`
}

// ToolOutput represents the common output structure for tools.
type ToolOutput struct {
	Passed   bool                  `json:"passed"`
	Summary  string                `json:"summary,omitempty"`
	Domains  []domain.DomainResult `json:"domains,omitempty"`
	Files    []domain.FileResult   `json:"files,omitempty"`
	Warnings []string              `json:"warnings,omitempty"`
	Error    string                `json:"error,omitempty"`
}

// coalesce returns value if non-empty, otherwise fallback.
func coalesce(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
