package mcp

import (
	"log"
	"time"
)

// Telemetry records MCP tool usage metrics (opt-in only).
// Implementations must be non-blocking and safe for concurrent use.
type Telemetry interface {
	// RecordToolCall records a tool invocation with outcome.
	// duration: time from invocation to valid output.
	// err: non-nil if the tool returned an error.
	// rejected: true if the call was rejected by input sanitization.
	RecordToolCall(tool string, duration time.Duration, err error, rejected bool)

	// RecordRegressionCaught records a regression caught before commit.
	// tool: which tool caught it (check, compare).
	// domain: the domain name where regression was found.
	// shortfall: how many percentage points below threshold.
	RecordRegressionCaught(tool string, domain string, shortfall float64)
}

// NoopTelemetry is used when telemetry is disabled (default).
// It discards all events without side effects.
type NoopTelemetry struct{}

func (NoopTelemetry) RecordToolCall(_ string, _ time.Duration, _ error, _ bool) {}
func (NoopTelemetry) RecordRegressionCaught(_ string, _ string, _ float64)  {}

// MetricsTelemetry writes structured JSON logs to the provided writer.
// Format: {"tool":"check","duration_ms":1234,"outcome":"success","rejected":false}
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
	m.logger.Printf(`{"tool":%q,"duration_ms":%d,"outcome":%q,"rejected":%v}`,
		tool, duration.Milliseconds(), outcome, rejected)
}

func (m *MetricsTelemetry) RecordRegressionCaught(tool string, domain string, shortfall float64) {
	m.logger.Printf(`{"event":"regression_caught","tool":%q,"domain":%q,"shortfall":%.1f}`,
		tool, domain, shortfall)
}
