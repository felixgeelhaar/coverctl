package cli

import (
	"io"
	"log/slog"
)

// setupLogger builds a slog.Logger from global flags and installs it as the
// default. CI mode emits JSON (machine-parseable in CI logs); --debug raises
// the level to Debug; otherwise warnings and above only.
//
// All log output goes to stderr to avoid contaminating stdout, which carries
// command results consumed by scripts and AI agents over MCP stdio.
func setupLogger(stderr io.Writer, global GlobalOptions) *slog.Logger {
	level := slog.LevelWarn
	if global.Debug {
		level = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if global.CI || global.Debug {
		// JSON for CI (parseable) and debug (machine-friendly diagnostics).
		handler = slog.NewJSONHandler(stderr, opts)
	} else {
		// Suppressed by default: WARN+ only, text format.
		handler = slog.NewTextHandler(stderr, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
