package mcp

import (
	"context"
	"fmt"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/history"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// handleCheck implements the check tool.
func (s *Server) handleCheck(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input CheckInput,
) (*mcp.CallToolResult, ToolOutput, error) {
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

	output := ToolOutput{
		Passed:   result.Passed,
		Domains:  result.Domains,
		Files:    result.Files,
		Warnings: result.Warnings,
	}

	if err != nil {
		output.Passed = false
		output.Error = err.Error()
	}

	// Generate summary
	output.Summary = generateSummary(result)

	return nil, output, nil
}

// handleReport implements the report tool.
func (s *Server) handleReport(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ReportInput,
) (*mcp.CallToolResult, ToolOutput, error) {
	opts := application.ReportOptions{
		ConfigPath:    coalesce(input.ConfigPath, s.config.ConfigPath),
		Profile:       coalesce(input.Profile, s.config.ProfilePath),
		Output:        application.OutputJSON,
		Domains:       input.Domains,
		ShowUncovered: input.ShowUncovered,
		DiffRef:       input.DiffRef,
	}

	result, err := s.svc.ReportResult(ctx, opts)

	output := ToolOutput{
		Passed:   result.Passed,
		Domains:  result.Domains,
		Files:    result.Files,
		Warnings: result.Warnings,
	}

	if err != nil {
		output.Passed = false
		output.Error = err.Error()
	}

	// Generate summary
	output.Summary = generateSummary(result)

	return nil, output, nil
}

// handleRecord implements the record tool.
func (s *Server) handleRecord(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input RecordInput,
) (*mcp.CallToolResult, ToolOutput, error) {
	opts := application.RecordOptions{
		ConfigPath:  coalesce(input.ConfigPath, s.config.ConfigPath),
		ProfilePath: coalesce(input.Profile, s.config.ProfilePath),
		HistoryPath: coalesce(input.HistoryPath, s.config.HistoryPath),
		Commit:      input.Commit,
		Branch:      input.Branch,
	}

	store := &history.FileStore{Path: opts.HistoryPath}

	err := s.svc.Record(ctx, opts, store)

	output := ToolOutput{
		Passed: err == nil,
	}

	if err != nil {
		output.Error = err.Error()
		output.Summary = "Failed to record coverage"
	} else {
		output.Summary = "Coverage recorded to history"
	}

	return nil, output, nil
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
