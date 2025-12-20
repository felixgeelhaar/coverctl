package config

import (
	"errors"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
)

type Loader struct{}

type fileConfig struct {
	Policy  filePolicy `yaml:"policy"`
	Exclude []string   `yaml:"exclude"`
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

	return application.Config{
		Policy:  policy,
		Exclude: cfg.Exclude,
	}, nil
}

func Write(w io.Writer, cfg application.Config) error {
	out := fileConfig{
		Policy: filePolicy{
			Default: fileDefault{Min: cfg.Policy.DefaultMin},
			Domains: make([]fileDomain, 0, len(cfg.Policy.Domains)),
		},
		Exclude: cfg.Exclude,
	}
	for _, d := range cfg.Policy.Domains {
		out.Policy.Domains = append(out.Policy.Domains, fileDomain{
			Name:  d.Name,
			Match: d.Match,
			Min:   d.Min,
		})
	}
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc.Encode(out)
}
