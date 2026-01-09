package application

import (
	"context"
	"fmt"
)

// PRCommentHandler handles PR comment operations.
type PRCommentHandler struct {
	ConfigLoader      ConfigLoader
	Autodetector      Autodetector
	DomainResolver    DomainResolver
	ProfileParser     ProfileParser
	DiffProvider      DiffProvider
	AnnotationScanner AnnotationScanner
	PRClients         map[PRProvider]PRClient
	CommentFormatter  CommentFormatter
}

// PRComment posts a coverage report as a comment on a PR/MR.
func (h *PRCommentHandler) PRComment(ctx context.Context, opts PRCommentOptions) (PRCommentResult, error) {
	if h.CommentFormatter == nil {
		return PRCommentResult{}, fmt.Errorf("comment formatter not configured")
	}

	provider := opts.Provider
	if provider == "" || provider == ProviderAuto {
		provider = detectProvider()
	}

	client, ok := h.PRClients[provider]
	if !ok || client == nil {
		return PRCommentResult{}, fmt.Errorf("%s client not configured", provider)
	}

	profilePath := opts.ProfilePath
	if profilePath == "" {
		profilePath = "coverage.out"
	}

	// Create a report handler to get coverage result
	reportHandler := &ReportHandler{
		ConfigLoader:      h.ConfigLoader,
		Autodetector:      h.Autodetector,
		DomainResolver:    h.DomainResolver,
		ProfileParser:     h.ProfileParser,
		DiffProvider:      h.DiffProvider,
		AnnotationScanner: h.AnnotationScanner,
	}

	result, err := reportHandler.ReportResult(ctx, ReportOptions{
		ConfigPath: opts.ConfigPath,
		Profile:    profilePath,
	})
	if err != nil {
		return PRCommentResult{}, fmt.Errorf("generate coverage report: %w", err)
	}

	var comparison *CompareResult
	if opts.BaseProfile != "" {
		analyticsHandler := &AnalyticsHandler{
			ConfigLoader:      h.ConfigLoader,
			Autodetector:      h.Autodetector,
			DomainResolver:    h.DomainResolver,
			ProfileParser:     h.ProfileParser,
			AnnotationScanner: h.AnnotationScanner,
		}

		comp, err := analyticsHandler.Compare(ctx, CompareOptions{
			ConfigPath:  opts.ConfigPath,
			BaseProfile: opts.BaseProfile,
			HeadProfile: profilePath,
		})
		if err != nil {
			return PRCommentResult{}, fmt.Errorf("compare coverage: %w", err)
		}
		comparison = &comp
	}

	commentBody := h.CommentFormatter.FormatCoverageComment(result, comparison)

	if opts.DryRun {
		return PRCommentResult{
			CommentBody: commentBody,
		}, nil
	}

	if opts.UpdateExisting {
		existingID, err := client.FindCoverageComment(ctx, opts.Owner, opts.Repo, opts.PRNumber)
		if err != nil {
			return PRCommentResult{}, fmt.Errorf("find existing comment: %w", err)
		}

		if existingID != 0 {
			if err := client.UpdateComment(ctx, opts.Owner, opts.Repo, existingID, commentBody); err != nil {
				return PRCommentResult{}, fmt.Errorf("update comment: %w", err)
			}
			return PRCommentResult{
				CommentID:   existingID,
				CommentBody: commentBody,
				Created:     false,
			}, nil
		}
	}

	commentID, commentURL, err := client.CreateComment(ctx, opts.Owner, opts.Repo, opts.PRNumber, commentBody)
	if err != nil {
		return PRCommentResult{}, fmt.Errorf("create comment: %w", err)
	}

	return PRCommentResult{
		CommentID:   commentID,
		CommentURL:  commentURL,
		CommentBody: commentBody,
		Created:     true,
	}, nil
}

// Note: CommentFormatter interface is defined in types.go
// Note: detectProvider function is defined in service.go
