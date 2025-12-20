package application

import (
	"context"
	"errors"
	"io"

	"github.com/felixgeelhaar/coverctl/internal/domain"
)

type OutputFormat string

const (
	OutputText OutputFormat = "text"
	OutputJSON OutputFormat = "json"
)

var ErrConfigNotFound = errors.New("config not found")

// Config represents validated, application-ready configuration.
type Config struct {
	Version int
	Policy  domain.Policy
	Exclude []string
}

type ConfigLoader interface {
	Load(path string) (Config, error)
	Exists(path string) (bool, error)
}

type Autodetector interface {
	Detect() (Config, error)
}

type DomainResolver interface {
	Resolve(ctx context.Context, domains []domain.Domain) (map[string][]string, error)
	ModuleRoot(ctx context.Context) (string, error)
	ModulePath(ctx context.Context) (string, error)
}

type CoverageRunner interface {
	Run(ctx context.Context, opts RunOptions) (string, error)
}

type ProfileParser interface {
	Parse(path string) (map[string]domain.CoverageStat, error)
}

type Reporter interface {
	Write(w io.Writer, result domain.Result, format OutputFormat) error
}

type RunOptions struct {
	Domains     []domain.Domain
	ProfilePath string
}

type IgnoreOptions struct {
	ConfigPath string
}
