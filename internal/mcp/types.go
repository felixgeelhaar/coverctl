// Package mcp provides Model Context Protocol server implementation for coverctl.
package mcp

import (
	"context"
	"fmt"

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
	PRComment(ctx context.Context, opts application.PRCommentOptions) (application.PRCommentResult, error)

	// Query tools (read-only but exposed as tools for better discoverability)
	Debt(ctx context.Context, opts application.DebtOptions) (application.DebtResult, error)
	Trend(ctx context.Context, opts application.TrendOptions, store application.HistoryStore) (application.TrendResult, error)
	Suggest(ctx context.Context, opts application.SuggestOptions) (application.SuggestResult, error)
	Badge(ctx context.Context, opts application.BadgeOptions) (application.BadgeResult, error)
	Compare(ctx context.Context, opts application.CompareOptions) (application.CompareResult, error)
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
	ConfigPath  string   `json:"configPath,omitempty" jsonschema:"description=Path to .coverctl.yaml config file"`
	Profile     string   `json:"profile,omitempty" jsonschema:"description=Coverage profile output path"`
	FromProfile bool     `json:"fromProfile,omitempty" jsonschema:"description=Use existing coverage profile instead of running tests"`
	Domains     []string `json:"domains,omitempty" jsonschema:"description=Filter to specific domains"`
	FailUnder   *float64 `json:"failUnder,omitempty" jsonschema:"description=Fail if coverage below threshold"`
	Ratchet     bool     `json:"ratchet,omitempty" jsonschema:"description=Fail if coverage decreases"`
	// Build flags for go test
	Tags     string   `json:"tags,omitempty" jsonschema:"description=Build tags (e.g. 'integration,e2e')"`
	Race     bool     `json:"race,omitempty" jsonschema:"description=Enable race detector"`
	Short    bool     `json:"short,omitempty" jsonschema:"description=Skip long-running tests (-short flag)"`
	Verbose  bool     `json:"verbose,omitempty" jsonschema:"description=Verbose test output"`
	Run      string   `json:"run,omitempty" jsonschema:"description=Run only tests matching pattern"`
	Timeout  string   `json:"timeout,omitempty" jsonschema:"description=Test timeout (e.g. '10m', '1h')"`
	TestArgs []string `json:"testArgs,omitempty" jsonschema:"description=Additional arguments passed to go test"`
	// Incremental mode
	Incremental    bool   `json:"incremental,omitempty" jsonschema:"description=Only test packages with changed files"`
	IncrementalRef string `json:"incrementalRef,omitempty" jsonschema:"description=Git ref to compare against for incremental mode (default: HEAD~1)"`
}

// ReportInput defines the input parameters for the report tool.
type ReportInput struct {
	ConfigPath    string   `json:"configPath,omitempty" jsonschema:"description=Path to .coverctl.yaml config file"`
	Profile       string   `json:"profile,omitempty" jsonschema:"description=Path to existing coverage profile"`
	Domains       []string `json:"domains,omitempty" jsonschema:"description=Filter to specific domains"`
	ShowUncovered bool     `json:"showUncovered,omitempty" jsonschema:"description=Show only files with 0%% coverage"`
	DiffRef       string   `json:"diffRef,omitempty" jsonschema:"description=Git ref for diff-based filtering"`
}

// RecordInput defines the input parameters for the record tool.
type RecordInput struct {
	ConfigPath  string   `json:"configPath,omitempty" jsonschema:"description=Path to .coverctl.yaml config file"`
	Profile     string   `json:"profile,omitempty" jsonschema:"description=Path to coverage profile"`
	HistoryPath string   `json:"historyPath,omitempty" jsonschema:"description=Path to history file"`
	Commit      string   `json:"commit,omitempty" jsonschema:"description=Git commit SHA"`
	Branch      string   `json:"branch,omitempty" jsonschema:"description=Git branch name"`
	Run         bool     `json:"run,omitempty" jsonschema:"description=Run coverage before recording history"`
	Domains     []string `json:"domains,omitempty" jsonschema:"description=Filter to specific domains"`
	Language    string   `json:"language,omitempty" jsonschema:"description=Override language detection (go, python, nodejs, rust, java)"`
	Tags        string   `json:"tags,omitempty" jsonschema:"description=Build tags (e.g. 'integration,e2e')"`
	Race        bool     `json:"race,omitempty" jsonschema:"description=Enable race detector"`
	Short       bool     `json:"short,omitempty" jsonschema:"description=Skip long-running tests (-short flag)"`
	Verbose     bool     `json:"verbose,omitempty" jsonschema:"description=Verbose test output"`
	TestRun     string   `json:"testRun,omitempty" jsonschema:"description=Run only tests matching pattern"`
	Timeout     string   `json:"timeout,omitempty" jsonschema:"description=Test timeout (e.g. '10m', '1h')"`
	TestArgs    []string `json:"testArgs,omitempty" jsonschema:"description=Additional arguments passed to go test"`
}

// InitInput defines the input parameters for the init tool.
type InitInput struct {
	ConfigPath string `json:"configPath,omitempty" jsonschema:"description=Path to write .coverctl.yaml config file"`
	Force      bool   `json:"force,omitempty" jsonschema:"description=Overwrite existing config file if it exists"`
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

// SuggestInput defines the input parameters for the suggest tool.
type SuggestInput struct {
	ConfigPath  string `json:"configPath,omitempty" jsonschema:"description=Path to .coverctl.yaml config file"`
	Profile     string `json:"profile,omitempty" jsonschema:"description=Path to coverage profile"`
	Strategy    string `json:"strategy,omitempty" jsonschema:"description=Suggestion strategy: current (default)|aggressive|conservative"`
	WriteConfig bool   `json:"writeConfig,omitempty" jsonschema:"description=Write suggested thresholds to config file (creates backup if file exists)"`
}

// DebtInput defines the input parameters for the debt tool.
type DebtInput struct {
	ConfigPath string `json:"configPath,omitempty" jsonschema:"description=Path to .coverctl.yaml config file"`
	Profile    string `json:"profile,omitempty" jsonschema:"description=Path to coverage profile"`
}

// BadgeInput defines the input parameters for the badge tool.
type BadgeInput struct {
	ConfigPath string `json:"configPath,omitempty" jsonschema:"description=Path to .coverctl.yaml config file"`
	Profile    string `json:"profile,omitempty" jsonschema:"description=Path to coverage profile"`
	Output     string `json:"output,omitempty" jsonschema:"description=Output file path for SVG badge"`
	Label      string `json:"label,omitempty" jsonschema:"description=Badge label text (default: coverage)"`
	Style      string `json:"style,omitempty" jsonschema:"description=Badge style: flat (default)|flat-square"`
}

// CompareInput defines the input parameters for the compare tool.
type CompareInput struct {
	ConfigPath  string `json:"configPath,omitempty" jsonschema:"description=Path to .coverctl.yaml config file"`
	BaseProfile string `json:"baseProfile" jsonschema:"description=Path to the base coverage profile (required)"`
	HeadProfile string `json:"headProfile,omitempty" jsonschema:"description=Path to the head coverage profile to compare against"`
}

// PRCommentInput defines the input parameters for the pr-comment tool.
type PRCommentInput struct {
	ConfigPath     string `json:"configPath,omitempty" jsonschema:"description=Path to .coverctl.yaml config file"`
	Profile        string `json:"profile,omitempty" jsonschema:"description=Path to coverage profile"`
	BaseProfile    string `json:"baseProfile,omitempty" jsonschema:"description=Base coverage profile for comparison (optional)"`
	Provider       string `json:"provider,omitempty" jsonschema:"description=Git provider: github gitlab bitbucket or auto (default: auto)"`
	PRNumber       int    `json:"prNumber" jsonschema:"description=Pull request/MR number (required for GitHub auto-detected for GitLab/Bitbucket CI)"`
	Owner          string `json:"owner,omitempty" jsonschema:"description=Repository owner/namespace (auto-detected from env)"`
	Repo           string `json:"repo,omitempty" jsonschema:"description=Repository name (auto-detected from env)"`
	UpdateExisting *bool  `json:"updateExisting,omitempty" jsonschema:"description=Update existing comment instead of creating new (default: true)"`
	DryRun         bool   `json:"dryRun,omitempty" jsonschema:"description=Generate comment without posting"`
}

// generateSummary creates a human-readable summary from the result.
func generateSummary(result domain.Result) string {
	if len(result.Domains) == 0 {
		return "No domains found"
	}

	var totalCovered, totalStatements int
	var passing int

	for _, d := range result.Domains {
		totalCovered += d.Covered
		totalStatements += d.Total
		if d.Status == domain.StatusPass {
			passing++
		}
	}

	overallPercent := 0.0
	if totalStatements > 0 {
		overallPercent = float64(totalCovered) / float64(totalStatements) * 100
	}

	total := len(result.Domains)
	if result.Passed {
		return fmt.Sprintf("PASS | %.1f%% overall | %d/%d domains passing", overallPercent, passing, total)
	}
	return fmt.Sprintf("FAIL | %.1f%% overall | %d/%d domains passing", overallPercent, passing, total)
}
