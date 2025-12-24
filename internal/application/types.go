package application

import (
	"context"
	"errors"
	"io"

	"github.com/felixgeelhaar/coverctl/internal/domain"
)

type OutputFormat string

const (
	OutputText  OutputFormat = "text"
	OutputJSON  OutputFormat = "json"
	OutputHTML  OutputFormat = "html"
	OutputBrief OutputFormat = "brief"
)

var ErrConfigNotFound = errors.New("config not found")

// Config represents validated, application-ready configuration.
type Config struct {
	Version     int
	Policy      domain.Policy
	Exclude     []string
	Files       []domain.FileRule
	Diff        DiffConfig
	Merge       MergeConfig
	Integration IntegrationConfig
	Annotations AnnotationsConfig
}

type FileRule = domain.FileRule

type DiffConfig struct {
	Enabled bool
	Base    string
}

type MergeConfig struct {
	Profiles []string
}

type IntegrationConfig struct {
	Enabled  bool
	Packages []string
	RunArgs  []string
	CoverDir string
	Profile  string
}

type AnnotationsConfig struct {
	Enabled bool
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
	RunIntegration(ctx context.Context, opts IntegrationOptions) (string, error)
}

type ProfileParser interface {
	Parse(path string) (map[string]domain.CoverageStat, error)
	ParseAll(paths []string) (map[string]domain.CoverageStat, error)
}

type DiffProvider interface {
	ChangedFiles(ctx context.Context, base string) ([]string, error)
}

type AnnotationScanner interface {
	Scan(ctx context.Context, moduleRoot string, files []string) (map[string]Annotation, error)
}

type Reporter interface {
	Write(w io.Writer, result domain.Result, format OutputFormat) error
}

type RunOptions struct {
	Domains     []domain.Domain
	ProfilePath string
	BuildFlags  BuildFlags // Build and test flags
}

// BuildFlags contains options passed to go test
type BuildFlags struct {
	Tags     string   // Build tags (e.g., "integration,e2e")
	Race     bool     // Enable race detector
	Short    bool     // Skip long-running tests
	Verbose  bool     // Verbose test output
	Run      string   // Run only tests matching pattern
	Timeout  string   // Test timeout (e.g., "10m", "1h")
	TestArgs []string // Additional arguments passed to go test
}

type IntegrationOptions struct {
	Domains    []domain.Domain
	Packages   []string
	RunArgs    []string
	CoverDir   string
	Profile    string
	BuildFlags BuildFlags // Build and test flags
}

type Annotation struct {
	Ignore bool
	Domain string
}

type IgnoreOptions struct {
	ConfigPath string
}

type BadgeOptions struct {
	ConfigPath  string
	ProfilePath string
	Output      string
	Label       string
	Style       string
}

type TrendOptions struct {
	ConfigPath  string
	ProfilePath string
	HistoryPath string
	Output      OutputFormat
	Days        int // Number of days to analyze (0 = all)
}

type RecordOptions struct {
	ConfigPath  string
	ProfilePath string
	HistoryPath string
	Commit      string
	Branch      string
}

type HistoryStore interface {
	Load() (domain.History, error)
	Save(h domain.History) error
	Append(entry domain.HistoryEntry) error
}

type SuggestOptions struct {
	ConfigPath  string
	ProfilePath string
	Strategy    SuggestStrategy
}

type SuggestStrategy string

const (
	// SuggestCurrent suggests thresholds slightly below current coverage
	SuggestCurrent SuggestStrategy = "current"
	// SuggestAggressive suggests higher thresholds to push for improvement
	SuggestAggressive SuggestStrategy = "aggressive"
	// SuggestConservative suggests lower thresholds for gradual improvement
	SuggestConservative SuggestStrategy = "conservative"
)

type Suggestion struct {
	Domain         string
	CurrentPercent float64
	CurrentMin     float64
	SuggestedMin   float64
	Reason         string
}

// FileWatcher provides file change notifications.
type FileWatcher interface {
	WatchDir(root string) error
	Events(ctx context.Context) <-chan struct{}
	Close() error
}

// WatchOptions configures watch mode behavior.
type WatchOptions struct {
	ConfigPath string
	Profile    string
	Domains    []string
	Clear      bool       // Clear terminal before each run
	BuildFlags BuildFlags // Build and test flags
}

// DebtOptions configures the coverage debt report.
type DebtOptions struct {
	ConfigPath  string
	ProfilePath string
	Output      OutputFormat
}

// DebtItem represents a single coverage debt item.
type DebtItem struct {
	Name      string  // Domain or file name
	Type      string  // "domain" or "file"
	Current   float64 // Current coverage percentage
	Required  float64 // Required minimum coverage
	Shortfall float64 // How much coverage is missing (required - current)
	Lines     int     // Estimated lines of code needing tests
}

// DebtResult contains the overall coverage debt analysis.
type DebtResult struct {
	Items       []DebtItem
	TotalDebt   float64 // Sum of all shortfalls
	TotalLines  int     // Total estimated lines needing tests
	HealthScore float64 // 0-100 score (higher is better)
}
