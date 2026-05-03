package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/felixgeelhaar/coverctl/internal/mcp"
)

// runMCP implements `coverctl mcp <subcommand>`.
func runMCP(ctx context.Context, args []string, stdout, stderr io.Writer, svc Service, global GlobalOptions) int {
	_ = svc
	_ = global
	if len(args) < 1 {
		fmt.Fprintln(stderr, "Usage: coverctl mcp <subcommand>")
		fmt.Fprintln(stderr, "Subcommands: serve")
		return 2
	}
	switch args[0] {
	case "serve":
		fs := flag.NewFlagSet("mcp serve", flag.ContinueOnError)
		fs.Usage = func() { commandHelp("mcp", stderr) }
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		fs.StringVar(configPath, "c", ".coverctl.yaml", "Config file path (shorthand)")
		historyPath := fs.String("history", ".cover/history.json", "History file path")
		profilePath := fs.String("profile", ".cover/coverage.out", "Coverage profile path")
		fs.StringVar(profilePath, "p", ".cover/coverage.out", "Coverage profile path (shorthand)")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}

		// BuildService requires *os.File for the legacy reporter; CLI's
		// stdout is the canonical sink even when writing MCP frames over a
		// different pipe.
		mcpSvc := BuildService(os.Stdout)
		_ = stdout
		mcpServer := mcp.New(mcpSvc, mcp.Config{
			ConfigPath:  *configPath,
			HistoryPath: *historyPath,
			ProfilePath: *profilePath,
		}, Version)

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			cancel()
		}()

		if err := mcpServer.Run(ctx); err != nil {
			fmt.Fprintf(stderr, "MCP server error: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "Unknown mcp subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "Available subcommands: serve")
		return 2
	}
}
