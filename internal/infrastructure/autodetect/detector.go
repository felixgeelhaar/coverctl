package autodetect

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/gotool"
)

type Detector struct {
	Module gotool.ModuleInfo
}

func (d Detector) Detect() (application.Config, error) {
	root, err := d.Module.ModuleRoot(contextBackground())
	if err != nil {
		return application.Config{}, err
	}

	domains := detectDomains(root)
	policy := domain.Policy{DefaultMin: 80, Domains: domains}
	return application.Config{Version: 1, Policy: policy}, nil
}

func detectDomains(root string) []domain.Domain {
	var domains []domain.Domain
	top := []string{"cmd", "internal", "pkg"}
	for _, dir := range top {
		full := filepath.Join(root, dir)
		info, err := os.Stat(full)
		if err != nil || !info.IsDir() {
			continue
		}
		if dir == "internal" {
			domains = append(domains, subdomains(full)...)
			continue
		}
		domains = append(domains, domain.Domain{
			Name:  dir,
			Match: []string{"./" + dir + "/..."},
		})
	}
	if len(domains) == 0 {
		domains = append(domains, domain.Domain{Name: "module", Match: []string{"./..."}})
	}
	return domains
}

func subdomains(internalPath string) []domain.Domain {
	entries, err := os.ReadDir(internalPath)
	if err != nil {
		return []domain.Domain{{Name: "internal", Match: []string{"./internal/..."}}}
	}
	ignore := map[string]struct{}{"mocks": {}, "mock": {}, "generated": {}, "testdata": {}}
	out := make([]domain.Domain, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, ok := ignore[name]; ok {
			continue
		}
		out = append(out, domain.Domain{
			Name:  name,
			Match: []string{"./internal/" + name + "/..."},
		})
	}
	if len(out) == 0 {
		out = append(out, domain.Domain{Name: "internal", Match: []string{"./internal/..."}})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func contextBackground() context.Context {
	return context.Background()
}
