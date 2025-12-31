package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/history"
	"github.com/felixgeelhaar/mcp-go"
)

// Version is set at build time.
var Version = "dev"

// Server wraps the application service with MCP protocol handling.
type Server struct {
	svc    Service
	config Config
	server *mcp.Server
}

// New creates a new MCP server wrapping the given service.
func New(svc Service, cfg Config) *Server {
	// Apply defaults
	if cfg.ConfigPath == "" {
		cfg.ConfigPath = DefaultConfig().ConfigPath
	}
	if cfg.HistoryPath == "" {
		cfg.HistoryPath = DefaultConfig().HistoryPath
	}
	if cfg.ProfilePath == "" {
		cfg.ProfilePath = DefaultConfig().ProfilePath
	}

	s := &Server{
		svc:    svc,
		config: cfg,
	}

	// Create MCP server with capabilities
	s.server = mcp.NewServer(mcp.ServerInfo{
		Name:    "coverctl",
		Version: Version,
		Capabilities: mcp.Capabilities{
			Tools:     true,
			Resources: true,
			Prompts:   true,
		},
	})

	// Register tools and resources
	s.registerTools()
	s.registerResources()

	return s
}

// Run starts the MCP server and blocks until the context is canceled.
func (s *Server) Run(ctx context.Context) error {
	return mcp.ServeStdio(ctx, s.server)
}

// registerTools adds all tool handlers to the server.
func (s *Server) registerTools() {
	// Check tool - runs coverage tests with policy enforcement
	s.server.Tool("check").
		Description("Run coverage tests and enforce policy thresholds. Executes 'go test -cover' and evaluates results against configured minimums.").
		Handler(s.handleCheck)

	// Report tool - analyzes existing coverage without running tests
	s.server.Tool("report").
		Description("Analyze an existing coverage profile without running tests. Use this when you already have a coverage.out file.").
		Handler(s.handleReport)

	// Record tool - saves coverage to history
	s.server.Tool("record").
		Description("Record current coverage to history for trend tracking. Call this after 'check' to save coverage data.").
		Handler(s.handleRecord)
}

// registerResources adds all resource handlers to the server.
func (s *Server) registerResources() {
	// Debt resource
	s.server.Resource("coverctl://debt").
		Name("Coverage Debt").
		Description("Shows coverage debt - gap between current and required coverage thresholds").
		MimeType("application/json").
		Handler(s.handleDebtResource)

	// Trend resource
	s.server.Resource("coverctl://trend").
		Name("Coverage Trend").
		Description("Shows coverage trends over time from recorded history").
		MimeType("application/json").
		Handler(s.handleTrendResource)

	// Suggest resource
	s.server.Resource("coverctl://suggest").
		Name("Threshold Suggestions").
		Description("Suggests optimal coverage thresholds based on current coverage").
		MimeType("application/json").
		Handler(s.handleSuggestResource)

	// Config resource
	s.server.Resource("coverctl://config").
		Name("Current Configuration").
		Description("Returns current or auto-detected coverctl configuration").
		MimeType("application/json").
		Handler(s.handleConfigResource)
}

// Tool handlers

func (s *Server) handleCheck(ctx context.Context, input CheckInput) (map[string]any, error) {
	opts := application.CheckOptions{
		ConfigPath: coalesce(input.ConfigPath, s.config.ConfigPath),
		Profile:    coalesce(input.Profile, s.config.ProfilePath),
		Output:     application.OutputJSON,
		Domains:    input.Domains,
		FailUnder:  input.FailUnder,
		Ratchet:    input.Ratchet,
	}

	// Add history store if ratchet is enabled
	if input.Ratchet {
		opts.HistoryStore = &history.FileStore{Path: s.config.HistoryPath}
	}

	result, err := s.svc.CheckResult(ctx, opts)

	output := map[string]any{
		"passed":   result.Passed,
		"summary":  generateSummary(result),
		"domains":  result.Domains,
		"files":    result.Files,
		"warnings": result.Warnings,
	}

	if err != nil {
		output["passed"] = false
		output["error"] = err.Error()
	}

	return output, nil
}

func (s *Server) handleReport(ctx context.Context, input ReportInput) (map[string]any, error) {
	opts := application.ReportOptions{
		ConfigPath:    coalesce(input.ConfigPath, s.config.ConfigPath),
		Profile:       coalesce(input.Profile, s.config.ProfilePath),
		Output:        application.OutputJSON,
		Domains:       input.Domains,
		ShowUncovered: input.ShowUncovered,
		DiffRef:       input.DiffRef,
	}

	result, err := s.svc.ReportResult(ctx, opts)

	output := map[string]any{
		"passed":   result.Passed,
		"summary":  generateSummary(result),
		"domains":  result.Domains,
		"files":    result.Files,
		"warnings": result.Warnings,
	}

	if err != nil {
		output["passed"] = false
		output["error"] = err.Error()
	}

	return output, nil
}

func (s *Server) handleRecord(ctx context.Context, input RecordInput) (map[string]any, error) {
	opts := application.RecordOptions{
		ConfigPath:  coalesce(input.ConfigPath, s.config.ConfigPath),
		ProfilePath: coalesce(input.Profile, s.config.ProfilePath),
		HistoryPath: coalesce(input.HistoryPath, s.config.HistoryPath),
		Commit:      input.Commit,
		Branch:      input.Branch,
	}

	store := &history.FileStore{Path: opts.HistoryPath}

	err := s.svc.Record(ctx, opts, store)

	output := map[string]any{
		"passed": err == nil,
	}

	if err != nil {
		output["error"] = err.Error()
		output["summary"] = "Failed to record coverage"
	} else {
		output["summary"] = "Coverage recorded to history"
	}

	return output, nil
}

// Resource handlers

func (s *Server) handleDebtResource(ctx context.Context, uri string, params map[string]string) (*mcp.ResourceContent, error) {
	result, err := s.svc.Debt(ctx, application.DebtOptions{
		ConfigPath:  s.config.ConfigPath,
		ProfilePath: s.config.ProfilePath,
		Output:      application.OutputJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to calculate debt: %w", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal debt result: %w", err)
	}

	return &mcp.ResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

func (s *Server) handleTrendResource(ctx context.Context, uri string, params map[string]string) (*mcp.ResourceContent, error) {
	store := &history.FileStore{Path: s.config.HistoryPath}

	result, err := s.svc.Trend(ctx, application.TrendOptions{
		ConfigPath:  s.config.ConfigPath,
		ProfilePath: s.config.ProfilePath,
		HistoryPath: s.config.HistoryPath,
		Output:      application.OutputJSON,
	}, store)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate trend: %w", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal trend result: %w", err)
	}

	return &mcp.ResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

func (s *Server) handleSuggestResource(ctx context.Context, uri string, params map[string]string) (*mcp.ResourceContent, error) {
	result, err := s.svc.Suggest(ctx, application.SuggestOptions{
		ConfigPath:  s.config.ConfigPath,
		ProfilePath: s.config.ProfilePath,
		Strategy:    application.SuggestCurrent,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate suggestions: %w", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal suggest result: %w", err)
	}

	return &mcp.ResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

func (s *Server) handleConfigResource(ctx context.Context, uri string, params map[string]string) (*mcp.ResourceContent, error) {
	result, err := s.svc.Detect(ctx, application.DetectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to detect config: %w", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	return &mcp.ResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}
