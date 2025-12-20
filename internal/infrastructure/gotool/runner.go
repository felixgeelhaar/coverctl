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
	Module ModuleResolver
	Exec   func(ctx context.Context, dir string, args []string) error
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
