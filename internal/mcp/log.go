package mcp

import (
	"log/slog"
	"time"
)

// traceTool emits a debug event when an MCP tool handler enters and another
// when it returns. Use as: defer traceTool("check")() at the top of a handler.
//
// Logging is essential for debugging MCP-driven failures: when an agent
// reports "coverctl returned an error" the operator needs to see which tool
// was called, with what shape of input, how long it ran, and what came back.
// Without this, the only diagnostic surface is the agent's own redacted
// output. Defaults to slog.Default() so the CLI's --debug / --ci flags
// (which install JSON handlers) propagate through automatically.
func traceTool(name string) func() {
	start := time.Now()
	slog.Debug("mcp tool start", "tool", name)
	return func() {
		slog.Debug("mcp tool end", "tool", name, "duration_ms", time.Since(start).Milliseconds())
	}
}
