package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/history"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// handleDebtResource returns coverage debt metrics.
func (s *Server) handleDebtResource(
	ctx context.Context,
	req *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
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

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

// handleTrendResource returns coverage trend analysis.
func (s *Server) handleTrendResource(
	ctx context.Context,
	req *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
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

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

// handleSuggestResource returns threshold recommendations.
func (s *Server) handleSuggestResource(
	ctx context.Context,
	req *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
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

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

// handleConfigResource returns the current or detected configuration.
func (s *Server) handleConfigResource(
	ctx context.Context,
	req *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	result, err := s.svc.Detect(ctx, application.DetectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to detect config: %w", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}
