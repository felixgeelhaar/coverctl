package gotool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"

	"github.com/felixgeelhaar/coverctl/internal/domain"
)

type DomainResolver struct {
	Module ModuleResolver
}

type goPackage struct {
	Dir string `json:"Dir"`
}

func (r DomainResolver) Resolve(ctx context.Context, domains []domain.Domain) (map[string][]string, error) {
	moduleRoot, err := r.Module.ModuleRoot(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]string, len(domains))
	for _, d := range domains {
		dirs := make([]string, 0)
		for _, match := range d.Match {
			pkgs, err := goList(ctx, moduleRoot, match)
			if err != nil {
				return nil, fmt.Errorf("go list %s: %w", match, err)
			}
			for _, pkg := range pkgs {
				dirs = append(dirs, pkg.Dir)
			}
		}
		result[d.Name] = unique(dirs)
	}
	return result, nil
}

func (r DomainResolver) ModuleRoot(ctx context.Context) (string, error) {
	return r.Module.ModuleRoot(ctx)
}

func (r DomainResolver) ModulePath(ctx context.Context) (string, error) {
	return r.Module.ModulePath(ctx)
}

func goList(ctx context.Context, dir string, pattern string) ([]goPackage, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-json", pattern)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytesReader(out))
	pkgs := []goPackage{}
	for {
		var pkg goPackage
		if err := dec.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

func unique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func bytesReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
