# MCP Tool-Call Success Metrics Spec

## Overview

Define instrumented metrics to evaluate which MCP tools drive retained value and where agents fail/hallucinate in real usage. These metrics enable data-driven decisions on tool prioritization.

## Metrics Definitions

### 1. Tool-Call Success Rate

**Definition:** `(successful calls / total calls) × 100`

**Success criteria per tool:**
- `check`: `passed: true` in output, no error
- `report`: non-empty `domains` array in output, no error
- `suggest`: non-empty suggestions or thresholds in output
- `debt`: non-empty `items` array in output
- `compare`: `delta` computed, no error
- `record`: history entry created, no error
- `badge`: SVG generated, no error
- `pr-comment`: comment created/updated, no error

**Target:** >85% success rate across all tools.

### 2. Rejection Rate (Security Sanitization)

**Definition:** `(rejected calls / total calls) × 100`

**Rejection reasons (from `internal/mcp/sanitize.go`):**
- Shell metacharacter in args
- Dangerous long flag (`--rootdir`, `--init-script`, etc.)
- Dangerous short flag prefix (`-D`, `-I`)
- Invalid tags pattern
- Invalid timeout format
- Control characters in input

**Target:** <5% rejection rate (high rate suggests AI agent is being fed malicious prompts or has broken output format).

### 3. Time-to-Success

**Definition:** `timestamp(output contains meaningful result) - timestamp(tool called)`

**Measurement:** Each tool handler records duration from invocation to valid output.

**Target:** 
- `check`: <30s (coverage run + policy evaluation)
- `report`: <5s (profile parse + analysis)
- `suggest`/`debt`: <3s (analysis only)
- `compare`: <10s (two profile parses + diff)

### 4. Pre-Commit Regression Catch Rate (North Star)

**Definition:** `(regressions caught by coverctl before commit) / (total regressions introduced)`

**Measurement:** 
- Instrument `coverctl check --from-profile` in agent loop
- Track: agent edits code → runs check → regression caught (pass) vs regression missed (fail)
- Source: agent session logs or opt-in telemetry

**Target:** >80% regression catch rate. This is the core value proposition: "coverage feedback in the agent edit loop."

## Telemetry Abstraction

### Interface (for opt-in data collection)

```go
// internal/mcp/telemetry.go

// Telemetry records MCP tool usage metrics (opt-in only).
type Telemetry interface {
    // RecordToolCall records a tool invocation with outcome.
    RecordToolCall(tool string, duration time.Duration, err error, rejected bool)
    
    // RecordRegressionCaught records a regression caught before commit.
    RecordRegressionCaught(tool string, domain string, shortfall float64)
}

// NoopTelemetry is used when telemetry is disabled (default).
type NoopTelemetry struct{}

func (NoopTelemetry) RecordToolCall(_ string, _ time.Duration, _ error, _ bool) {}
func (NoopTelemetry) RecordRegressionCaught(_ string, _ string, _ float64) {}

// MetricsTelemetry writes to structured log (opt-in).
type MetricsTelemetry struct {
    logger *log.Logger
}

func (m *MetricsTelemetry) RecordToolCall(tool string, duration time.Duration, err error, rejected bool) {
    outcome := "success"
    if err != nil {
        outcome = "error"
    }
    if rejected {
        outcome = "rejected"
    }
    m.logger.Printf(`{"tool":%q,"duration_ms":%d,"outcome":%q}`, 
        tool, duration.Milliseconds(), outcome)
}
```

### Integration Point

Add `Telemetry` field to `mcp.Server` struct:

```go
// internal/mcp/server.go
type Server struct {
    svc       Service
    config    Config
    server    *mcp.Server
    prCommentLimit *rateLimiter
    Telemetry Telemetry // nil = NoopTelemetry (opt-in via config)
}
```

Default: `Telemetry: NoopTelemetry{}` (no data leaves the system).

Opt-in: Set `Telemetry: MetricsTelemetry{logger: log.New(os.Stderr, "mcp-telemetry: ", 0)}` when user configures `telemetry: true` in `.coverctl.yaml` or passes `--mcp-telemetry` flag.

## Success Metric: "Which 3 MCP tools drive retained value?"

**Hypothesis:** `check`, `suggest`, and `debt` are the primary value drivers.

**Validation:** After 30 days of opt-in telemetry:
1. Rank tools by `(successful calls / total calls) × (regressions caught)`
2. Double-down on top 3: improve error messages, add examples, reduce friction
3. Deprioritize bottom tools: document as "stable" and maintain only

## Next Steps

- [ ] Add `Telemetry` interface and `NoopTelemetry` implementation
- [ ] Instrument all tool handlers in `server.go` with `RecordToolCall`
- [ ] Add `telemetry: true` config option to `.coverctl.yaml` schema
- [ ] Document opt-in telemetry in README ("Help improve coverctl by sharing anonymous usage metrics")
- [ ] After 30 days: analyze data and publish "which tools drive value" blog post
