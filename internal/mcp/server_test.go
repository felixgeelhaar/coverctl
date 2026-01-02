package mcp

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
)

// mockService implements the Service interface for testing.
type mockService struct {
	checkResult   domain.Result
	checkErr      error
	reportResult  domain.Result
	reportErr     error
	recordErr     error
	debtResult    application.DebtResult
	debtErr       error
	trendResult   application.TrendResult
	trendErr      error
	suggestResult application.SuggestResult
	suggestErr    error
	detectResult  application.Config
	detectErr     error
}

func (m *mockService) CheckResult(ctx context.Context, opts application.CheckOptions) (domain.Result, error) {
	return m.checkResult, m.checkErr
}

func (m *mockService) ReportResult(ctx context.Context, opts application.ReportOptions) (domain.Result, error) {
	return m.reportResult, m.reportErr
}

func (m *mockService) Record(ctx context.Context, opts application.RecordOptions, store application.HistoryStore) error {
	return m.recordErr
}

func (m *mockService) Debt(ctx context.Context, opts application.DebtOptions) (application.DebtResult, error) {
	return m.debtResult, m.debtErr
}

func (m *mockService) Trend(ctx context.Context, opts application.TrendOptions, store application.HistoryStore) (application.TrendResult, error) {
	return m.trendResult, m.trendErr
}

func (m *mockService) Suggest(ctx context.Context, opts application.SuggestOptions) (application.SuggestResult, error) {
	return m.suggestResult, m.suggestErr
}

func (m *mockService) Detect(ctx context.Context, opts application.DetectOptions) (application.Config, error) {
	return m.detectResult, m.detectErr
}

func TestNew(t *testing.T) {
	svc := &mockService{}
	cfg := Config{
		ConfigPath:  "custom.yaml",
		HistoryPath: "custom/history.json",
		ProfilePath: "custom/coverage.out",
	}

	server := New(svc, cfg, "test")

	if server == nil {
		t.Fatal("expected non-nil server")
	}
	if server.config.ConfigPath != cfg.ConfigPath {
		t.Errorf("expected ConfigPath %q, got %q", cfg.ConfigPath, server.config.ConfigPath)
	}
	if server.config.HistoryPath != cfg.HistoryPath {
		t.Errorf("expected HistoryPath %q, got %q", cfg.HistoryPath, server.config.HistoryPath)
	}
	if server.config.ProfilePath != cfg.ProfilePath {
		t.Errorf("expected ProfilePath %q, got %q", cfg.ProfilePath, server.config.ProfilePath)
	}
	if server.server == nil {
		t.Error("expected internal MCP server to be initialized")
	}
}

func TestNew_DefaultConfig(t *testing.T) {
	svc := &mockService{}
	cfg := Config{} // Empty config should get defaults

	server := New(svc, cfg, "test")

	defaults := DefaultConfig()
	if server.config.ConfigPath != defaults.ConfigPath {
		t.Errorf("expected default ConfigPath %q, got %q", defaults.ConfigPath, server.config.ConfigPath)
	}
	if server.config.HistoryPath != defaults.HistoryPath {
		t.Errorf("expected default HistoryPath %q, got %q", defaults.HistoryPath, server.config.HistoryPath)
	}
	if server.config.ProfilePath != defaults.ProfilePath {
		t.Errorf("expected default ProfilePath %q, got %q", defaults.ProfilePath, server.config.ProfilePath)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ConfigPath != ".coverctl.yaml" {
		t.Errorf("expected ConfigPath '.coverctl.yaml', got %q", cfg.ConfigPath)
	}
	if cfg.HistoryPath != ".cover/history.json" {
		t.Errorf("expected HistoryPath '.cover/history.json', got %q", cfg.HistoryPath)
	}
	if cfg.ProfilePath != ".cover/coverage.out" {
		t.Errorf("expected ProfilePath '.cover/coverage.out', got %q", cfg.ProfilePath)
	}
}

func TestCoalesce(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback string
		expected string
	}{
		{
			name:     "returns value when non-empty",
			value:    "custom",
			fallback: "default",
			expected: "custom",
		},
		{
			name:     "returns fallback when value is empty",
			value:    "",
			fallback: "default",
			expected: "default",
		},
		{
			name:     "returns empty fallback when both empty",
			value:    "",
			fallback: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := coalesce(tt.value, tt.fallback)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenerateSummary(t *testing.T) {
	tests := []struct {
		name     string
		result   domain.Result
		contains string
	}{
		{
			name:     "no domains returns no domains message",
			result:   domain.Result{Domains: []domain.DomainResult{}},
			contains: "No domains found",
		},
		{
			name: "passing result shows PASS",
			result: domain.Result{
				Passed: true,
				Domains: []domain.DomainResult{
					{Domain: "core", Status: domain.StatusPass, Covered: 80, Total: 100},
				},
			},
			contains: "PASS",
		},
		{
			name: "failing result shows FAIL",
			result: domain.Result{
				Passed: false,
				Domains: []domain.DomainResult{
					{Domain: "core", Status: domain.StatusFail, Covered: 50, Total: 100},
				},
			},
			contains: "FAIL",
		},
		{
			name: "includes percentage",
			result: domain.Result{
				Passed: true,
				Domains: []domain.DomainResult{
					{Domain: "core", Status: domain.StatusPass, Covered: 75, Total: 100},
				},
			},
			contains: "75.0%",
		},
		{
			name: "includes domain count",
			result: domain.Result{
				Passed: true,
				Domains: []domain.DomainResult{
					{Domain: "core", Status: domain.StatusPass, Covered: 80, Total: 100},
					{Domain: "api", Status: domain.StatusPass, Covered: 90, Total: 100},
				},
			},
			contains: "2/2 domains passing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := generateSummary(tt.result)
			if !containsString(summary, tt.contains) {
				t.Errorf("expected summary to contain %q, got %q", tt.contains, summary)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestHandleCheck(t *testing.T) {
	svc := &mockService{
		checkResult: domain.Result{
			Passed: true,
			Domains: []domain.DomainResult{
				{Domain: "core", Status: domain.StatusPass, Covered: 80, Total: 100},
			},
		},
	}
	server := New(svc, DefaultConfig(), "test")

	output, err := server.handleCheck(context.Background(), CheckInput{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed, ok := output["passed"].(bool); !ok || !passed {
		t.Error("expected output['passed'] to be true")
	}
	if summary, ok := output["summary"].(string); !ok || summary == "" {
		t.Error("expected non-empty summary")
	}
}

func TestHandleReport(t *testing.T) {
	svc := &mockService{
		reportResult: domain.Result{
			Passed: true,
			Domains: []domain.DomainResult{
				{Domain: "core", Status: domain.StatusPass, Covered: 75, Total: 100},
			},
		},
	}
	server := New(svc, DefaultConfig(), "test")

	output, err := server.handleReport(context.Background(), ReportInput{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed, ok := output["passed"].(bool); !ok || !passed {
		t.Error("expected output['passed'] to be true")
	}
}

func TestHandleRecord(t *testing.T) {
	svc := &mockService{}
	server := New(svc, DefaultConfig(), "test")

	output, err := server.handleRecord(context.Background(), RecordInput{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed, ok := output["passed"].(bool); !ok || !passed {
		t.Error("expected output['passed'] to be true")
	}
	if summary, ok := output["summary"].(string); !ok || summary != "Coverage recorded to history" {
		t.Errorf("expected success summary, got %q", summary)
	}
}

func TestHandleDebtResource(t *testing.T) {
	svc := &mockService{
		debtResult: application.DebtResult{
			TotalDebt: 10.5,
		},
	}
	server := New(svc, DefaultConfig(), "test")

	content, err := server.handleDebtResource(context.Background(), "coverctl://debt", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content == nil {
		t.Fatal("expected non-nil content")
	}
	if content.URI != "coverctl://debt" {
		t.Errorf("expected URI 'coverctl://debt', got %q", content.URI)
	}
	if content.MimeType != "application/json" {
		t.Errorf("expected MIME type 'application/json', got %q", content.MimeType)
	}
}

func TestHandleTrendResource(t *testing.T) {
	svc := &mockService{
		trendResult: application.TrendResult{},
	}
	server := New(svc, DefaultConfig(), "test")

	content, err := server.handleTrendResource(context.Background(), "coverctl://trend", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content == nil {
		t.Fatal("expected non-nil content")
	}
}

func TestHandleSuggestResource(t *testing.T) {
	svc := &mockService{
		suggestResult: application.SuggestResult{},
	}
	server := New(svc, DefaultConfig(), "test")

	content, err := server.handleSuggestResource(context.Background(), "coverctl://suggest", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content == nil {
		t.Fatal("expected non-nil content")
	}
}

func TestHandleConfigResource(t *testing.T) {
	svc := &mockService{
		detectResult: application.Config{},
	}
	server := New(svc, DefaultConfig(), "test")

	content, err := server.handleConfigResource(context.Background(), "coverctl://config", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content == nil {
		t.Fatal("expected non-nil content")
	}
}
