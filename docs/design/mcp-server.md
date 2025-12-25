# Technical Design: MCP Server for coverctl

## Overview

This document describes the design for adding Model Context Protocol (MCP) support to coverctl, enabling AI agents to interact with coverage tools programmatically.

**Goal:** Expose coverctl's coverage analysis capabilities via MCP while maintaining DDD architecture, testability, and idiomatic Go patterns.

---

## Architecture

### Layered Design (Following Existing Patterns)

```
┌─────────────────────────────────────────────────────────────┐
│                     Transport Layer                          │
│                    (STDIO / HTTP)                            │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    MCP Adapter Layer                         │
│              internal/mcp/                                   │
│   ┌─────────┐  ┌──────────┐  ┌───────────┐  ┌─────────┐    │
│   │ Server  │  │  Tools   │  │ Resources │  │  Types  │    │
│   └─────────┘  └──────────┘  └───────────┘  └─────────┘    │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   Application Layer                          │
│              internal/application/                           │
│              (Service - unchanged)                           │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                     Domain Layer                             │
│                internal/domain/                              │
│              (Pure domain logic)                             │
└─────────────────────────────────────────────────────────────┘
```

### Key Principles

1. **Thin Adapter Layer**: MCP layer only handles protocol translation
2. **Reuse Application Service**: All business logic stays in `application.Service`
3. **Dependency Injection**: MCP server receives service interface, not concrete type
4. **Testable Design**: All components can be tested with mocks

---

## Directory Structure

```
internal/
└── mcp/
    ├── server.go        # Server initialization, lifecycle
    ├── server_test.go   # Server unit tests
    ├── tools.go         # Tool definitions and handlers
    ├── tools_test.go    # Tool handler tests
    ├── resources.go     # Resource definitions and handlers
    ├── resources_test.go # Resource handler tests
    └── types.go         # MCP-specific types, schema builders
```

---

## Interface Design

### Service Interface (Port)

Define a focused interface for MCP needs (Interface Segregation):

```go
// internal/mcp/service.go

// Service defines the application operations needed by MCP.
// This interface allows for easy mocking in tests.
type Service interface {
    // Tools (actions that may have side effects)
    Check(ctx context.Context, opts application.CheckOptions) error
    Report(ctx context.Context, opts application.ReportOptions) error
    Record(ctx context.Context, opts application.RecordOptions, store application.HistoryStore) error

    // Resources (read-only queries)
    Debt(ctx context.Context, opts application.DebtOptions) (application.DebtResult, error)
    Trend(ctx context.Context, opts application.TrendOptions, store application.HistoryStore) (application.TrendResult, error)
    Suggest(ctx context.Context, opts application.SuggestOptions) (application.SuggestResult, error)
    Detect(ctx context.Context, opts application.DetectOptions) (application.Config, error)
}
```

### Server Configuration

```go
// internal/mcp/server.go

// Config holds MCP server configuration.
type Config struct {
    ConfigPath  string // Path to .coverctl.yaml (default: ".coverctl.yaml")
    HistoryPath string // Path to history file (default: ".cover/history.json")
    ProfilePath string // Path to coverage profile (default: ".cover/coverage.out")
}

// Server wraps the application service with MCP protocol handling.
type Server struct {
    svc    Service
    config Config
    output *bytes.Buffer // Captures service output for responses
}

// New creates a new MCP server wrapping the given service.
func New(svc Service, cfg Config) *Server {
    return &Server{
        svc:    svc,
        config: cfg,
        output: new(bytes.Buffer),
    }
}
```

---

## Tool Definitions

### Tool: `check`

**Purpose:** Run coverage tests and enforce policy.

```go
type CheckInput struct {
    ConfigPath string   `json:"configPath,omitempty" jsonschema:"description=Path to .coverctl.yaml config file"`
    Profile    string   `json:"profile,omitempty" jsonschema:"description=Coverage profile output path"`
    Domains    []string `json:"domains,omitempty" jsonschema:"description=Filter to specific domains"`
    FailUnder  *float64 `json:"failUnder,omitempty" jsonschema:"description=Fail if coverage below threshold"`
    Ratchet    bool     `json:"ratchet,omitempty" jsonschema:"description=Fail if coverage decreases"`
}

type CheckOutput struct {
    Passed   bool     `json:"passed"`
    Summary  string   `json:"summary"`
    Domains  []Domain `json:"domains,omitempty"`
    Warnings []string `json:"warnings,omitempty"`
    Error    string   `json:"error,omitempty"`
}
```

### Tool: `report`

**Purpose:** Analyze existing coverage profile (no test execution).

```go
type ReportInput struct {
    ConfigPath    string   `json:"configPath,omitempty"`
    Profile       string   `json:"profile,omitempty"`
    Domains       []string `json:"domains,omitempty"`
    ShowUncovered bool     `json:"showUncovered,omitempty" jsonschema:"description=Show only 0% coverage files"`
    DiffRef       string   `json:"diffRef,omitempty" jsonschema:"description=Git ref for diff-based filtering"`
}
```

### Tool: `record`

**Purpose:** Record current coverage to history for trend tracking.

```go
type RecordInput struct {
    ConfigPath  string `json:"configPath,omitempty"`
    Profile     string `json:"profile,omitempty"`
    HistoryPath string `json:"historyPath,omitempty"`
    Commit      string `json:"commit,omitempty" jsonschema:"description=Git commit SHA"`
    Branch      string `json:"branch,omitempty" jsonschema:"description=Git branch name"`
}
```

---

## Resource Definitions

### Resource: `coverctl://debt`

**Purpose:** Coverage debt metrics (gap between current and required).

```go
// Returns application.DebtResult as JSON
{
    "items": [
        {
            "name": "core",
            "type": "domain",
            "current": 65.5,
            "required": 80.0,
            "shortfall": 14.5,
            "lines": 150
        }
    ],
    "totalDebt": 14.5,
    "totalLines": 150,
    "healthScore": 75.0
}
```

### Resource: `coverctl://trend`

**Purpose:** Coverage trends over time.

### Resource: `coverctl://suggest`

**Purpose:** Threshold recommendations.

### Resource: `coverctl://config`

**Purpose:** Current/detected configuration.

---

## Implementation Details

### Server Implementation

```go
// internal/mcp/server.go

func (s *Server) Run(ctx context.Context) error {
    server := mcp.NewServer(&mcp.Implementation{
        Name:    "coverctl",
        Version: version.Version,
    }, nil)

    // Register tools
    s.registerTools(server)

    // Register resources
    s.registerResources(server)

    // Run with STDIO transport
    return server.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) registerTools(server *mcp.Server) {
    mcp.AddTool(server, &mcp.Tool{
        Name:        "check",
        Description: "Run coverage tests and enforce policy thresholds",
    }, s.handleCheck)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "report",
        Description: "Analyze existing coverage profile without running tests",
    }, s.handleReport)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "record",
        Description: "Record current coverage to history for trend tracking",
    }, s.handleRecord)
}

func (s *Server) registerResources(server *mcp.Server) {
    // Resources are read via ReadResource callbacks
    server.HandleReadResource(s.handleReadResource)
}
```

### Tool Handler Pattern

```go
// internal/mcp/tools.go

func (s *Server) handleCheck(
    ctx context.Context,
    req *mcp.CallToolRequest,
    input CheckInput,
) (*mcp.CallToolResult, CheckOutput, error) {
    // Apply defaults
    opts := application.CheckOptions{
        ConfigPath: coalesce(input.ConfigPath, s.config.ConfigPath),
        Profile:    coalesce(input.Profile, s.config.ProfilePath),
        Output:     application.OutputJSON,
        Domains:    input.Domains,
        FailUnder:  input.FailUnder,
        Ratchet:    input.Ratchet,
    }

    // Capture output
    s.output.Reset()
    // Note: Service.Out would need to be settable, or we wrap differently

    err := s.svc.Check(ctx, opts)

    output := CheckOutput{
        Passed: err == nil,
    }

    if err != nil {
        output.Error = err.Error()
        // Parse captured JSON output for details
    }

    return nil, output, nil
}

func coalesce(value, fallback string) string {
    if value == "" {
        return fallback
    }
    return value
}
```

### Resource Handler Pattern

```go
// internal/mcp/resources.go

func (s *Server) handleReadResource(
    ctx context.Context,
    req *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
    switch req.Params.URI {
    case "coverctl://debt":
        return s.readDebt(ctx)
    case "coverctl://trend":
        return s.readTrend(ctx)
    case "coverctl://suggest":
        return s.readSuggest(ctx)
    case "coverctl://config":
        return s.readConfig(ctx)
    default:
        return nil, fmt.Errorf("unknown resource: %s", req.Params.URI)
    }
}

func (s *Server) readDebt(ctx context.Context) (*mcp.ReadResourceResult, error) {
    result, err := s.svc.Debt(ctx, application.DebtOptions{
        ConfigPath:  s.config.ConfigPath,
        ProfilePath: s.config.ProfilePath,
        Output:      application.OutputJSON,
    })
    if err != nil {
        return nil, err
    }

    data, err := json.Marshal(result)
    if err != nil {
        return nil, err
    }

    return &mcp.ReadResourceResult{
        Contents: []mcp.ResourceContent{{
            URI:      "coverctl://debt",
            MimeType: "application/json",
            Text:     string(data),
        }},
    }, nil
}
```

---

## CLI Integration

### Subcommand: `coverctl mcp serve`

```go
// internal/cli/cli.go (modification)

case "mcp":
    return c.handleMCP(ctx, args[1:])

func (c *CLI) handleMCP(ctx context.Context, args []string) error {
    if len(args) == 0 || args[0] != "serve" {
        return fmt.Errorf("usage: coverctl mcp serve [flags]")
    }

    fs := flag.NewFlagSet("mcp serve", flag.ContinueOnError)
    configPath := fs.String("config", ".coverctl.yaml", "Config file path")
    historyPath := fs.String("history", ".cover/history.json", "History file path")
    profilePath := fs.String("profile", ".cover/coverage.out", "Coverage profile path")

    if err := fs.Parse(args[1:]); err != nil {
        return err
    }

    svc := BuildService(os.Stdout)
    server := mcp.New(svc, mcp.Config{
        ConfigPath:  *configPath,
        HistoryPath: *historyPath,
        ProfilePath: *profilePath,
    })

    return server.Run(ctx)
}
```

---

## Testing Strategy

### Unit Tests (TDD Approach)

Write tests first for each component:

```go
// internal/mcp/server_test.go

func TestServer_New(t *testing.T) {
    svc := &mockService{}
    cfg := Config{ConfigPath: ".coverctl.yaml"}

    server := New(svc, cfg)

    assert.NotNil(t, server)
    assert.Equal(t, cfg.ConfigPath, server.config.ConfigPath)
}
```

```go
// internal/mcp/tools_test.go

func TestHandleCheck_Success(t *testing.T) {
    svc := &mockService{
        checkFn: func(ctx context.Context, opts application.CheckOptions) error {
            return nil
        },
    }
    server := New(svc, Config{})

    input := CheckInput{Domains: []string{"core"}}
    _, output, err := server.handleCheck(context.Background(), nil, input)

    assert.NoError(t, err)
    assert.True(t, output.Passed)
}

func TestHandleCheck_PolicyViolation(t *testing.T) {
    svc := &mockService{
        checkFn: func(ctx context.Context, opts application.CheckOptions) error {
            return fmt.Errorf("policy violation")
        },
    }
    server := New(svc, Config{})

    input := CheckInput{}
    _, output, err := server.handleCheck(context.Background(), nil, input)

    assert.NoError(t, err) // Handler doesn't error, captures in output
    assert.False(t, output.Passed)
    assert.Contains(t, output.Error, "policy violation")
}
```

```go
// internal/mcp/resources_test.go

func TestReadDebt_ReturnsJSON(t *testing.T) {
    svc := &mockService{
        debtFn: func(ctx context.Context, opts application.DebtOptions) (application.DebtResult, error) {
            return application.DebtResult{
                HealthScore: 85.0,
                TotalDebt:   10.5,
            }, nil
        },
    }
    server := New(svc, Config{})

    result, err := server.readDebt(context.Background())

    assert.NoError(t, err)
    assert.Contains(t, result.Contents[0].Text, `"healthScore":85`)
}
```

### Mock Service

```go
// internal/mcp/mock_test.go

type mockService struct {
    checkFn   func(context.Context, application.CheckOptions) error
    reportFn  func(context.Context, application.ReportOptions) error
    debtFn    func(context.Context, application.DebtOptions) (application.DebtResult, error)
    trendFn   func(context.Context, application.TrendOptions, application.HistoryStore) (application.TrendResult, error)
    suggestFn func(context.Context, application.SuggestOptions) (application.SuggestResult, error)
    detectFn  func(context.Context, application.DetectOptions) (application.Config, error)
    recordFn  func(context.Context, application.RecordOptions, application.HistoryStore) error
}

func (m *mockService) Check(ctx context.Context, opts application.CheckOptions) error {
    if m.checkFn != nil {
        return m.checkFn(ctx, opts)
    }
    return nil
}

// ... implement other methods
```

### Integration Tests

```go
// internal/mcp/integration_test.go

func TestMCPServer_E2E(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Create temp directory with test fixtures
    // Start MCP server in subprocess
    // Connect as MCP client
    // Call tools and verify responses
}
```

---

## Error Handling

### MCP Error Format

```go
func toolError(msg string) CheckOutput {
    return CheckOutput{
        Passed: false,
        Error:  msg,
    }
}

// In handler:
if err != nil {
    return nil, toolError(err.Error()), nil
}
```

### Graceful Degradation

- If config not found: return error with helpful message
- If profile missing: suggest running `coverctl check` first
- If history empty: suggest running `coverctl record`

---

## Configuration

### Claude Desktop

```json
{
  "mcpServers": {
    "coverctl": {
      "command": "coverctl",
      "args": ["mcp", "serve"],
      "env": {}
    }
  }
}
```

### With Custom Paths

```json
{
  "mcpServers": {
    "coverctl": {
      "command": "coverctl",
      "args": [
        "mcp", "serve",
        "--config", ".coverctl.yaml",
        "--history", ".cover/history.json"
      ],
      "cwd": "/path/to/project"
    }
  }
}
```

---

## Implementation Phases

### Phase 1: MVP (Tools Only)
1. Add `github.com/modelcontextprotocol/go-sdk` dependency
2. Create `internal/mcp/types.go` with input/output structs
3. Create `internal/mcp/server.go` with basic server
4. Create `internal/mcp/tools.go` with `check` and `report`
5. Add `mcp serve` subcommand
6. Write unit tests for all handlers

### Phase 2: Complete Tools
1. Add `record` tool
2. Add proper error formatting
3. Add integration tests

### Phase 3: Resources
1. Create `internal/mcp/resources.go`
2. Implement all 4 resources
3. Add resource tests

### Phase 4: Polish
1. Add MCP Inspector testing guide
2. Update README with Claude Desktop config
3. Add help text for `mcp serve`

---

## Dependencies

```go
// go.mod additions
require (
    github.com/modelcontextprotocol/go-sdk v0.x.x
)
```

---

## Open Questions

1. **Output Capture**: The current `Service.Out` is set once. Should we:
   - Make it settable per-call?
   - Create a wrapper that captures output?
   - Modify service methods to return structured results?

2. **History Store**: Should MCP create its own `FileStore` or accept one?
   - Recommendation: Create internally using config paths

3. **Version**: Should MCP server version match coverctl version?
   - Recommendation: Yes, use shared `version.Version` constant
