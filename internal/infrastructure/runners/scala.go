package runners

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/felixgeelhaar/coverctl/internal/application"
)

// ScalaRunner implements CoverageRunner for Scala projects.
// Supports sbt with sbt-scoverage and Mill build tools.
type ScalaRunner struct {
	// Exec overrides command execution (for testing).
	Exec func(ctx context.Context, dir string, cmd string, args []string) error
}

// NewScalaRunner creates a new Scala coverage runner.
func NewScalaRunner() *ScalaRunner {
	return &ScalaRunner{}
}

// Name returns the runner's identifier.
func (r *ScalaRunner) Name() string {
	return "scala"
}

// Language returns the language this runner supports.
func (r *ScalaRunner) Language() application.Language {
	return application.LanguageScala
}

// Detect checks if this runner can handle the current project.
func (r *ScalaRunner) Detect(projectDir string) bool {
	markers := []string{
		"build.sbt",
		"project/build.properties",
	}
	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(projectDir, marker)); err == nil {
			return true
		}
	}
	return false
}

// Run executes Scala coverage tools and returns the profile path.
func (r *ScalaRunner) Run(ctx context.Context, opts application.RunOptions) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Detect build tool
	tool := r.detectBuildTool(cwd)

	// Determine profile path
	profile := opts.ProfilePath
	if profile == "" {
		profile = r.getDefaultProfilePath(tool)
	}
	if !filepath.IsAbs(profile) {
		profile = filepath.Join(cwd, profile)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(profile), 0o750); err != nil {
		return "", fmt.Errorf("scala coverage failed: %w", err)
	}

	// Build command args
	args := r.buildArgs(tool, opts)

	execFn := r.Exec
	if execFn == nil {
		execFn = runScalaCommand
	}

	// Run coverage command
	if err := execFn(ctx, cwd, tool, args); err != nil {
		return "", fmt.Errorf("scala coverage failed: %w", err)
	}

	return profile, nil
}

// RunIntegration runs integration tests with coverage collection.
func (r *ScalaRunner) RunIntegration(ctx context.Context, opts application.IntegrationOptions) (string, error) {
	runOpts := application.RunOptions{
		ProfilePath: opts.Profile,
		BuildFlags:  opts.BuildFlags,
	}
	return r.Run(ctx, runOpts)
}

// detectBuildTool determines which Scala build tool is used.
func (r *ScalaRunner) detectBuildTool(projectDir string) string {
	// Check for sbt first
	if _, err := os.Stat(filepath.Join(projectDir, "build.sbt")); err == nil {
		return "sbt"
	}

	// Check for Mill
	if _, err := os.Stat(filepath.Join(projectDir, "build.sc")); err == nil {
		return "mill"
	}

	return "sbt" // default
}

// getDefaultProfilePath returns the default scoverage report path for the build tool.
func (r *ScalaRunner) getDefaultProfilePath(tool string) string {
	switch tool {
	case "mill":
		return filepath.Join("out", "__.test", "scoverage", "xmlReport.dest", "scoverage.xml")
	default: // sbt
		return filepath.Join("target", "scala-2.13", "scoverage-report", "scoverage.xml")
	}
}

// buildArgs builds command line arguments for the detected tool.
func (r *ScalaRunner) buildArgs(tool string, opts application.RunOptions) []string {
	switch tool {
	case "mill":
		return r.buildMillArgs(opts)
	default:
		return r.buildSbtArgs(opts)
	}
}

// buildSbtArgs builds command line arguments for sbt with sbt-scoverage.
func (r *ScalaRunner) buildSbtArgs(opts application.RunOptions) []string {
	args := []string{
		"clean",
		"coverage",
		"test",
		"coverageReport",
	}

	// Add test filter
	if opts.BuildFlags.Run != "" {
		args = append(args, fmt.Sprintf("testOnly *%s*", opts.BuildFlags.Run))
	}

	// Add additional args
	args = append(args, opts.BuildFlags.TestArgs...)

	return args
}

// buildMillArgs builds command line arguments for Mill.
func (r *ScalaRunner) buildMillArgs(opts application.RunOptions) []string {
	args := []string{
		"__.test",
	}

	// Add additional args
	args = append(args, opts.BuildFlags.TestArgs...)

	return args
}

// runScalaCommand executes a Scala build command.
func runScalaCommand(ctx context.Context, dir string, tool string, args []string) error {
	var cmd *exec.Cmd

	switch tool {
	case "mill":
		// Try local mill wrapper first, fall back to mill from PATH
		wrapperPath := filepath.Join(dir, "mill")
		if _, err := os.Stat(wrapperPath); err == nil {
			cmd = exec.CommandContext(ctx, "./mill", args...)
		} else {
			cmd = exec.CommandContext(ctx, "mill", args...)
		}
	default: // sbt
		// Try sbt wrapper if present, fall back to sbt from PATH
		wrapperPath := filepath.Join(dir, "sbt")
		if _, err := os.Stat(wrapperPath); err == nil {
			cmd = exec.CommandContext(ctx, "./sbt", args...)
		} else {
			cmd = exec.CommandContext(ctx, "sbt", args...)
		}
	}

	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
