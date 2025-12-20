package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
)

type Loader struct{}

type fileConfig struct {
	Version     int             `yaml:"version"`
	Policy      filePolicy      `yaml:"policy"`
	Exclude     []string        `yaml:"exclude,omitempty"`
	Files       []fileFileRule  `yaml:"files,omitempty"`
	Diff        fileDiff        `yaml:"diff,omitempty"`
	Merge       fileMerge       `yaml:"merge,omitempty"`
	Integration fileIntegration `yaml:"integration,omitempty"`
	Annotations fileAnnotations `yaml:"annotations,omitempty"`
}

type filePolicy struct {
	Default fileDefault  `yaml:"default"`
	Domains []fileDomain `yaml:"domains"`
}

type fileDefault struct {
	Min float64 `yaml:"min"`
}

type fileDomain struct {
	Name  string   `yaml:"name"`
	Match []string `yaml:"match"`
	Min   *float64 `yaml:"min"`
}

type fileFileRule struct {
	Match []string `yaml:"match"`
	Min   float64  `yaml:"min"`
}

type fileDiff struct {
	Enabled bool   `yaml:"enabled"`
	Base    string `yaml:"base,omitempty"`
}

type fileMerge struct {
	Profiles []string `yaml:"profiles,omitempty"`
}

type fileIntegration struct {
	Enabled  bool     `yaml:"enabled"`
	Packages []string `yaml:"packages,omitempty"`
	RunArgs  []string `yaml:"run_args,omitempty"`
	CoverDir string   `yaml:"cover_dir,omitempty"`
	Profile  string   `yaml:"profile,omitempty"`
}

type fileAnnotations struct {
	Enabled bool `yaml:"enabled"`
}

func (l Loader) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func (l Loader) Load(path string) (application.Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return application.Config{}, err
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return application.Config{}, err
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Version != 1 {
		return application.Config{}, fmt.Errorf("unsupported config version: %d", cfg.Version)
	}
	if cfg.Diff.Enabled && cfg.Diff.Base == "" {
		cfg.Diff.Base = "origin/main"
	}
	if cfg.Integration.Enabled {
		if cfg.Integration.CoverDir == "" {
			cfg.Integration.CoverDir = filepath.Join(".cover", "integration")
		}
		if cfg.Integration.Profile == "" {
			cfg.Integration.Profile = filepath.Join(".cover", "integration.out")
		}
	}

	policy := domain.Policy{
		DefaultMin: cfg.Policy.Default.Min,
		Domains:    make([]domain.Domain, 0, len(cfg.Policy.Domains)),
	}

	for _, d := range cfg.Policy.Domains {
		policy.Domains = append(policy.Domains, domain.Domain{
			Name:  d.Name,
			Match: d.Match,
			Min:   d.Min,
		})
	}

	fileRules := make([]domain.FileRule, 0, len(cfg.Files))
	for _, rule := range cfg.Files {
		fileRules = append(fileRules, domain.FileRule{
			Match: rule.Match,
			Min:   rule.Min,
		})
	}

	return application.Config{
		Version: cfg.Version,
		Policy:  policy,
		Exclude: cfg.Exclude,
		Files:   fileRules,
		Diff: application.DiffConfig{
			Enabled: cfg.Diff.Enabled,
			Base:    cfg.Diff.Base,
		},
		Merge: application.MergeConfig{
			Profiles: append([]string(nil), cfg.Merge.Profiles...),
		},
		Integration: application.IntegrationConfig{
			Enabled:  cfg.Integration.Enabled,
			Packages: append([]string(nil), cfg.Integration.Packages...),
			RunArgs:  append([]string(nil), cfg.Integration.RunArgs...),
			CoverDir: cfg.Integration.CoverDir,
			Profile:  cfg.Integration.Profile,
		},
		Annotations: application.AnnotationsConfig{
			Enabled: cfg.Annotations.Enabled,
		},
	}, nil
}

func Write(w io.Writer, cfg application.Config) error {
	version := cfg.Version
	if version == 0 {
		version = 1
	}
	out := fileConfig{
		Version: version,
		Policy: filePolicy{
			Default: fileDefault{Min: cfg.Policy.DefaultMin},
			Domains: make([]fileDomain, 0, len(cfg.Policy.Domains)),
		},
		Exclude: cfg.Exclude,
		Files:   make([]fileFileRule, 0, len(cfg.Files)),
		Diff: fileDiff{
			Enabled: cfg.Diff.Enabled,
			Base:    cfg.Diff.Base,
		},
		Merge: fileMerge{
			Profiles: append([]string(nil), cfg.Merge.Profiles...),
		},
		Integration: fileIntegration{
			Enabled:  cfg.Integration.Enabled,
			Packages: append([]string(nil), cfg.Integration.Packages...),
			RunArgs:  append([]string(nil), cfg.Integration.RunArgs...),
			CoverDir: cfg.Integration.CoverDir,
			Profile:  cfg.Integration.Profile,
		},
		Annotations: fileAnnotations{Enabled: cfg.Annotations.Enabled},
	}
	for _, d := range cfg.Policy.Domains {
		out.Policy.Domains = append(out.Policy.Domains, fileDomain{
			Name:  d.Name,
			Match: d.Match,
			Min:   d.Min,
		})
	}
	for _, rule := range cfg.Files {
		out.Files = append(out.Files, fileFileRule{
			Match: rule.Match,
			Min:   rule.Min,
		})
	}
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc.Encode(out)
}
