package gotool

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
)

type Runner struct {
	Module     ModuleInfo
	Exec       func(ctx context.Context, dir string, args []string) error
	ExecOutput func(ctx context.Context, dir string, args []string) ([]byte, error)
	ExecEnv    func(ctx context.Context, dir string, env []string, cmd string, args []string) error
}

func (r Runner) Run(ctx context.Context, opts application.RunOptions) (string, error) {
	moduleRoot, err := r.Module.ModuleRoot(ctx)
	if err != nil {
		return "", err
	}

	profile := opts.ProfilePath
	if profile == "" {
		profile = filepath.Join(".cover", "coverage.out")
	}
	profilePath := profile
	if !filepath.IsAbs(profilePath) {
		profilePath = filepath.Join(moduleRoot, profilePath)
	}
	if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
		return "", err
	}

	coverpkg := buildCoverPkg(opts.Domains)
	args := []string{"test", "-covermode=atomic", "-coverprofile=" + profilePath}
	if coverpkg != "" {
		args = append(args, "-coverpkg="+coverpkg)
	}

	// Add build flags
	args = appendBuildFlags(args, opts.BuildFlags)
	args = append(args, "./...")

	execFn := r.Exec
	if execFn == nil {
		execFn = runCommand
	}
	fmt.Fprintf(os.Stderr, "running go args: %v\n", args)
	if err := execFn(ctx, moduleRoot, args); err != nil {
		return "", fmt.Errorf("go test failed: %w", err)
	}
	return profilePath, nil
}

func (r Runner) RunIntegration(ctx context.Context, opts application.IntegrationOptions) (string, error) {
	moduleRoot, err := r.Module.ModuleRoot(ctx)
	if err != nil {
		return "", err
	}

	coverDir := opts.CoverDir
	if coverDir == "" {
		coverDir = filepath.Join(".cover", "integration")
	}
	coverDirPath := coverDir
	if !filepath.IsAbs(coverDirPath) {
		coverDirPath = filepath.Join(moduleRoot, coverDirPath)
	}
	if err := os.RemoveAll(coverDirPath); err != nil {
		return "", err
	}
	if err := os.MkdirAll(coverDirPath, 0o755); err != nil {
		return "", err
	}

	profile := opts.Profile
	if profile == "" {
		profile = filepath.Join(".cover", "integration.out")
	}
	profilePath := profile
	if !filepath.IsAbs(profilePath) {
		profilePath = filepath.Join(moduleRoot, profilePath)
	}
	if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
		return "", err
	}

	packages, err := r.listPackages(ctx, moduleRoot, opts.Packages)
	if err != nil {
		return "", err
	}
	if len(packages) == 0 {
		return "", fmt.Errorf("no packages resolved for integration coverage")
	}

	coverpkg := buildCoverPkg(opts.Domains)
	execFn := r.Exec
	if execFn == nil {
		execFn = runCommand
	}
	execEnv := r.ExecEnv
	if execEnv == nil {
		execEnv = runCommandEnv
	}

	tmpDir, err := os.MkdirTemp("", "coverctl-integration-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	for _, pkg := range packages {
		binName := strings.ReplaceAll(pkg, "/", "_") + ".test"
		binPath := filepath.Join(tmpDir, binName)
		args := []string{"test", "-covermode=atomic", "-c", "-o", binPath}
		if coverpkg != "" {
			args = append(args, "-coverpkg="+coverpkg)
		}
		// Add build flags (only build-time flags for -c)
		if opts.BuildFlags.Tags != "" {
			args = append(args, "-tags="+opts.BuildFlags.Tags)
		}
		if opts.BuildFlags.Race {
			args = append(args, "-race")
		}
		args = append(args, pkg)
		if err := execFn(ctx, moduleRoot, args); err != nil {
			return "", fmt.Errorf("go test -c failed: %w", err)
		}
		env := append(os.Environ(), "GOCOVERDIR="+coverDirPath)
		if err := execEnv(ctx, moduleRoot, env, binPath, opts.RunArgs); err != nil {
			return "", fmt.Errorf("integration test failed: %w", err)
		}
	}

	if err := execFn(ctx, moduleRoot, []string{"tool", "covdata", "textfmt", "-i", coverDirPath, "-o", profilePath}); err != nil {
		return "", fmt.Errorf("covdata textfmt failed: %w", err)
	}
	return profilePath, nil
}

// appendBuildFlags adds build flags to the go test args slice
func appendBuildFlags(args []string, flags application.BuildFlags) []string {
	if flags.Tags != "" {
		args = append(args, "-tags="+flags.Tags)
	}
	if flags.Race {
		args = append(args, "-race")
	}
	if flags.Short {
		args = append(args, "-short")
	}
	if flags.Verbose {
		args = append(args, "-v")
	}
	if flags.Run != "" {
		args = append(args, "-run="+flags.Run)
	}
	if flags.Timeout != "" {
		args = append(args, "-timeout="+flags.Timeout)
	}
	// Add any additional test args
	args = append(args, flags.TestArgs...)
	return args
}

func (r Runner) listPackages(ctx context.Context, moduleRoot string, patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	execOut := r.ExecOutput
	if execOut == nil {
		execOut = runCommandOutput
	}
	args := append([]string{"list"}, patterns...)
	out, err := execOut(ctx, moduleRoot, args)
	if err != nil {
		return nil, fmt.Errorf("go list failed: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	pkgs := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pkgs = append(pkgs, line)
	}
	return pkgs, nil
}

func buildCoverPkg(domains []domain.Domain) string {
	if len(domains) == 0 {
		return "./..."
	}
	patterns := make([]string, 0)
	seen := make(map[string]struct{})
	for _, d := range domains {
		for _, match := range d.Match {
			if match == "" {
				continue
			}
			if _, ok := seen[match]; ok {
				continue
			}
			seen[match] = struct{}{}
			patterns = append(patterns, match)
		}
	}
	return strings.Join(patterns, ",")
}

func runCommand(ctx context.Context, dir string, args []string) error {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCommandOutput(ctx context.Context, dir string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

func runCommandEnv(ctx context.Context, dir string, env []string, cmdPath string, args []string) error {
	cmd := exec.CommandContext(ctx, cmdPath, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
