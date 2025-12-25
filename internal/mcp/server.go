package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version is set at build time.
var Version = "dev"

// Server wraps the application service with MCP protocol handling.
type Server struct {
	svc    Service
	config Config
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

	return &Server{
		svc:    svc,
		config: cfg,
	}
}

// Run starts the MCP server and blocks until the context is canceled.
func (s *Server) Run(ctx context.Context) error {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "coverctl",
			Version: Version,
		},
		&mcp.ServerOptions{
			Capabilities: &mcp.ServerCapabilities{
				Tools:     &mcp.ToolCapabilities{},
				Resources: &mcp.ResourceCapabilities{},
			},
		},
	)

	// Register tools
	s.registerTools(server)

	// Register resources
	s.registerResources(server)

	// Run with STDIO transport
	transport := &mcp.StdioTransport{}
	if err := server.Run(ctx, transport); err != nil {
		// EOF is a normal shutdown condition when stdin closes
		// This happens when the client disconnects gracefully
		if isGracefulShutdown(err) {
			return nil
		}
		return fmt.Errorf("mcp server error: %w", err)
	}

	return nil
}

// isGracefulShutdown checks if the error indicates a normal client disconnection.
// EOF and "server is closing" errors are expected when the client closes the connection.
func isGracefulShutdown(err error) bool {
	if err == nil {
		return true
	}
	// Direct EOF check
	if errors.Is(err, io.EOF) {
		return true
	}
	// The SDK wraps EOF in "server is closing: EOF"
	errMsg := err.Error()
	if strings.Contains(errMsg, "EOF") || strings.Contains(errMsg, "server is closing") {
		return true
	}
	return false
}

// registerTools adds all tool handlers to the server.
func (s *Server) registerTools(server *mcp.Server) {
	// Check tool - runs coverage tests with policy enforcement
	mcp.AddTool(server, &mcp.Tool{
		Name:        "check",
		Description: "Run coverage tests and enforce policy thresholds. Executes 'go test -cover' and evaluates results against configured minimums.",
	}, s.handleCheck)

	// Report tool - analyzes existing coverage without running tests
	mcp.AddTool(server, &mcp.Tool{
		Name:        "report",
		Description: "Analyze an existing coverage profile without running tests. Use this when you already have a coverage.out file.",
	}, s.handleReport)

	// Record tool - saves coverage to history
	mcp.AddTool(server, &mcp.Tool{
		Name:        "record",
		Description: "Record current coverage to history for trend tracking. Call this after 'check' to save coverage data.",
	}, s.handleRecord)
}

// registerResources adds all resource handlers to the server.
func (s *Server) registerResources(server *mcp.Server) {
	// Debt resource
	server.AddResource(&mcp.Resource{
		URI:         "coverctl://debt",
		Name:        "Coverage Debt",
		Description: "Shows coverage debt - gap between current and required coverage thresholds",
		MIMEType:    "application/json",
	}, s.handleDebtResource)

	// Trend resource
	server.AddResource(&mcp.Resource{
		URI:         "coverctl://trend",
		Name:        "Coverage Trend",
		Description: "Shows coverage trends over time from recorded history",
		MIMEType:    "application/json",
	}, s.handleTrendResource)

	// Suggest resource
	server.AddResource(&mcp.Resource{
		URI:         "coverctl://suggest",
		Name:        "Threshold Suggestions",
		Description: "Suggests optimal coverage thresholds based on current coverage",
		MIMEType:    "application/json",
	}, s.handleSuggestResource)

	// Config resource
	server.AddResource(&mcp.Resource{
		URI:         "coverctl://config",
		Name:        "Current Configuration",
		Description: "Returns current or auto-detected coverctl configuration",
		MIMEType:    "application/json",
	}, s.handleConfigResource)
}
