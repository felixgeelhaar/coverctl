package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/annotations"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/autodetect"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/badge"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/config"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/coverprofile"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/diff"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/gotool"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/history"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/report"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/watcher"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/wizard"
)

type Service interface {
	Check(ctx context.Context, opts application.CheckOptions) error
	RunOnly(ctx context.Context, opts application.RunOnlyOptions) error
	Detect(ctx context.Context, opts application.DetectOptions) (application.Config, error)
	Report(ctx context.Context, opts application.ReportOptions) error
	Ignore(ctx context.Context, opts application.IgnoreOptions) (application.Config, []domain.Domain, error)
	Badge(ctx context.Context, opts application.BadgeOptions) (application.BadgeResult, error)
	Trend(ctx context.Context, opts application.TrendOptions, store application.HistoryStore) (application.TrendResult, error)
	Record(ctx context.Context, opts application.RecordOptions, store application.HistoryStore) error
	Suggest(ctx context.Context, opts application.SuggestOptions) (application.SuggestResult, error)
	Watch(ctx context.Context, opts application.WatchOptions, watcher application.FileWatcher, callback application.WatchCallback) error
	Debt(ctx context.Context, opts application.DebtOptions) (application.DebtResult, error)
}

var initWizard = wizard.Run

func Run(args []string, stdout, stderr io.Writer, svc Service) int {
	if len(args) < 2 {
		usage(stderr)
		return 2
	}

	ctx := context.Background()

	switch args[1] {
	case "check":
		fs := flag.NewFlagSet("check", flag.ExitOnError)
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		output := outputFlags(fs)
		profile := fs.String("profile", "", "Coverage profile output path")
		historyPath := fs.String("history", "", "History file path for delta display")
		showDelta := fs.Bool("show-delta", false, "Show coverage change from previous run")
		var domains domainList
		fs.Var(&domains, "domain", "Filter to specific domain (repeatable)")
		_ = fs.Parse(args[2:])
		opts := application.CheckOptions{ConfigPath: *configPath, Output: *output, Profile: *profile, Domains: domains}
		if *showDelta {
			histPath := *historyPath
			if histPath == "" {
				histPath = ".cover/history.json"
			}
			opts.HistoryStore = &history.FileStore{Path: histPath}
		}
		err := svc.Check(ctx, opts)
		return exitCode(err, 1, stderr)
	case "run":
		fs := flag.NewFlagSet("run", flag.ExitOnError)
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		profile := fs.String("profile", "", "Coverage profile output path")
		watch := fs.Bool("watch", false, "Watch for file changes and re-run coverage")
		var domains domainList
		fs.Var(&domains, "domain", "Filter to specific domain (repeatable)")
		_ = fs.Parse(args[2:])

		if *watch {
			return runWatch(ctx, stdout, stderr, svc, *configPath, *profile, domains)
		}
		err := svc.RunOnly(ctx, application.RunOnlyOptions{ConfigPath: *configPath, Profile: *profile, Domains: domains})
		return exitCode(err, 3, stderr)
	case "detect":
		fs := flag.NewFlagSet("detect", flag.ExitOnError)
		writeConfig := fs.Bool("write-config", false, "Write detected config to .coverctl.yaml")
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		force := fs.Bool("force", false, "Overwrite config if it exists")
		_ = fs.Parse(args[2:])
		cfg, err := svc.Detect(ctx, application.DetectOptions{})
		if err != nil {
			return exitCode(err, 3, stderr)
		}
		if *writeConfig {
			if err := writeConfigFile(*configPath, cfg, stdout, *force); err != nil {
				return exitCode(err, 2, stderr)
			}
			return 0
		}
		if err := writeConfigFile("-", cfg, stdout, *force); err != nil {
			return exitCode(err, 2, stderr)
		}
		return 0
	case "report":
		fs := flag.NewFlagSet("report", flag.ExitOnError)
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		output := outputFlags(fs)
		profile := fs.String("profile", ".cover/coverage.out", "Coverage profile path")
		historyPath := fs.String("history", "", "History file path for delta display")
		showDelta := fs.Bool("show-delta", false, "Show coverage change from previous run")
		var domains domainList
		fs.Var(&domains, "domain", "Filter to specific domain (repeatable)")
		_ = fs.Parse(args[2:])
		opts := application.ReportOptions{ConfigPath: *configPath, Output: *output, Profile: *profile, Domains: domains}
		if *showDelta {
			histPath := *historyPath
			if histPath == "" {
				histPath = ".cover/history.json"
			}
			opts.HistoryStore = &history.FileStore{Path: histPath}
		}
		err := svc.Report(ctx, opts)
		return exitCode(err, 3, stderr)
	case "ignore":
		fs := flag.NewFlagSet("ignore", flag.ExitOnError)
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		_ = fs.Parse(args[2:])
		cfg, domains, err := svc.Ignore(ctx, application.IgnoreOptions{ConfigPath: *configPath})
		if err != nil {
			return exitCode(err, 4, stderr)
		}
		printIgnoreInfo(cfg, domains, stdout)
		return 0
	case "init":
		fs := flag.NewFlagSet("init", flag.ExitOnError)
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		force := fs.Bool("force", false, "Overwrite existing config file")
		noInteractive := fs.Bool("no-interactive", false, "Skip the interactive init wizard")
		_ = fs.Parse(args[2:])
		cfg, err := svc.Detect(ctx, application.DetectOptions{})
		if err != nil {
			return exitCode(err, 3, stderr)
		}
		if !*noInteractive {
			var confirmed bool
			cfg, confirmed, err = initWizard(cfg, stdout, os.Stdin)
			if err != nil {
				return exitCode(err, 5, stderr)
			}
			if !confirmed {
				fmt.Fprintln(stdout, "Init cancelled; no configuration written.")
				return 0
			}
		}
		if err := writeConfigFile(*configPath, cfg, stdout, *force); err != nil {
			return exitCode(err, 2, stderr)
		}
		return 0
	case "badge":
		fs := flag.NewFlagSet("badge", flag.ExitOnError)
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		profile := fs.String("profile", ".cover/coverage.out", "Coverage profile path")
		output := fs.String("output", "coverage.svg", "Output file path")
		label := fs.String("label", "coverage", "Badge label text")
		style := fs.String("style", "flat", "Badge style: flat|flat-square")
		_ = fs.Parse(args[2:])
		result, err := svc.Badge(ctx, application.BadgeOptions{
			ConfigPath:  *configPath,
			ProfilePath: *profile,
			Output:      *output,
			Label:       *label,
			Style:       *style,
		})
		if err != nil {
			return exitCode(err, 3, stderr)
		}
		if err := writeBadgeFile(*output, result.Percent, *label, *style); err != nil {
			return exitCode(err, 3, stderr)
		}
		fmt.Fprintf(stdout, "Badge written to %s (%.1f%%)\n", *output, result.Percent)
		return 0
	case "trend":
		fs := flag.NewFlagSet("trend", flag.ExitOnError)
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		profile := fs.String("profile", ".cover/coverage.out", "Coverage profile path")
		historyPath := fs.String("history", ".cover/history.json", "History file path")
		output := outputFlags(fs)
		_ = fs.Parse(args[2:])
		store := history.FileStore{Path: *historyPath}
		result, err := svc.Trend(ctx, application.TrendOptions{
			ConfigPath:  *configPath,
			ProfilePath: *profile,
			HistoryPath: *historyPath,
			Output:      *output,
		}, &store)
		if err != nil {
			return exitCode(err, 3, stderr)
		}
		printTrendResult(result, stdout)
		return 0
	case "record":
		fs := flag.NewFlagSet("record", flag.ExitOnError)
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		profile := fs.String("profile", ".cover/coverage.out", "Coverage profile path")
		historyPath := fs.String("history", ".cover/history.json", "History file path")
		commit := fs.String("commit", "", "Git commit SHA (optional)")
		branch := fs.String("branch", "", "Git branch name (optional)")
		_ = fs.Parse(args[2:])
		store := history.FileStore{Path: *historyPath}
		err := svc.Record(ctx, application.RecordOptions{
			ConfigPath:  *configPath,
			ProfilePath: *profile,
			HistoryPath: *historyPath,
			Commit:      *commit,
			Branch:      *branch,
		}, &store)
		if err != nil {
			return exitCode(err, 3, stderr)
		}
		fmt.Fprintln(stdout, "Coverage recorded to history")
		return 0
	case "suggest":
		fs := flag.NewFlagSet("suggest", flag.ExitOnError)
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		profile := fs.String("profile", ".cover/coverage.out", "Coverage profile path")
		strategy := fs.String("strategy", "current", "Suggestion strategy: current|aggressive|conservative")
		writeConfig := fs.Bool("write-config", false, "Update config with suggested thresholds")
		force := fs.Bool("force", false, "Overwrite config if it exists")
		_ = fs.Parse(args[2:])

		var strat application.SuggestStrategy
		switch *strategy {
		case "aggressive":
			strat = application.SuggestAggressive
		case "conservative":
			strat = application.SuggestConservative
		default:
			strat = application.SuggestCurrent
		}

		result, err := svc.Suggest(ctx, application.SuggestOptions{
			ConfigPath:  *configPath,
			ProfilePath: *profile,
			Strategy:    strat,
		})
		if err != nil {
			return exitCode(err, 3, stderr)
		}
		printSuggestResult(result, stdout)
		if *writeConfig {
			if err := writeConfigFile(*configPath, result.Config, stdout, *force); err != nil {
				return exitCode(err, 2, stderr)
			}
			fmt.Fprintf(stdout, "\nConfig updated: %s\n", *configPath)
		}
		return 0
	case "debt":
		fs := flag.NewFlagSet("debt", flag.ExitOnError)
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		profile := fs.String("profile", ".cover/coverage.out", "Coverage profile path")
		output := outputFlags(fs)
		_ = fs.Parse(args[2:])

		result, err := svc.Debt(ctx, application.DebtOptions{
			ConfigPath:  *configPath,
			ProfilePath: *profile,
			Output:      *output,
		})
		if err != nil {
			return exitCode(err, 3, stderr)
		}
		printDebtResult(result, stdout, *output)
		return 0
	default:
		usage(stderr)
		return 2
	}
}

func BuildService(out *os.File) *application.Service {
	module := gotool.ModuleResolver{}
	return &application.Service{
		ConfigLoader:      config.Loader{},
		Autodetector:      autodetect.Detector{Module: module},
		DomainResolver:    gotool.DomainResolver{Module: module},
		CoverageRunner:    gotool.Runner{Module: module},
		ProfileParser:     coverprofile.Parser{},
		DiffProvider:      diff.GitDiff{Module: module},
		AnnotationScanner: annotations.Scanner{},
		Reporter:          report.Writer{},
		Out:               out,
	}
}

func outputFlags(fs *flag.FlagSet) *application.OutputFormat {
	output := application.OutputText
	fs.Var((*outputValue)(&output), "output", "Output format: text|json")
	fs.Var((*outputValue)(&output), "o", "Output format: text|json")
	return &output
}

type outputValue application.OutputFormat

func (o *outputValue) String() string { return string(*o) }

func (o *outputValue) Set(value string) error {
	switch value {
	case string(application.OutputText), string(application.OutputJSON):
		*o = outputValue(value)
		return nil
	default:
		return fmt.Errorf("invalid output format: %s", value)
	}
}

// domainList implements flag.Value for repeatable --domain flags
type domainList []string

func (d *domainList) String() string { return strings.Join(*d, ",") }

func (d *domainList) Set(value string) error {
	*d = append(*d, value)
	return nil
}

func writeConfigFile(path string, cfg application.Config, stdout io.Writer, force bool) error {
	if path == "-" {
		return config.Write(stdout, cfg)
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config %s already exists", path)
		}
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return config.Write(file, cfg)
}

func usage(w io.Writer) {
	fmt.Fprintln(w, `coverctl <command>

Commands:
  check   Run coverage and enforce policy
  run     Run coverage only, produce artifacts
  detect  Autodetect domains (use --write-config to save)
  ignore  Show configured excludes and ignore advice
  init    Run autodetect plus the interactive wizard
  report  Analyze an existing profile
  badge   Generate an SVG coverage badge
  trend   Show coverage trends over time
  record  Record current coverage to history
  suggest Suggest optimal coverage thresholds
  debt    Show coverage debt report`)
}

func writeBadgeFile(path string, percent float64, label, style string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	badgeStyle := badge.StyleFlat
	if style == "flat-square" {
		badgeStyle = badge.StyleFlatSquare
	}

	return badge.Generate(file, badge.Options{
		Label:   label,
		Percent: percent,
		Style:   badgeStyle,
	})
}

func exitCode(err error, code int, stderr io.Writer) int {
	if err == nil {
		return 0
	}
	fmt.Fprintln(stderr, err)
	return code
}

func printIgnoreInfo(cfg application.Config, domains []domain.Domain, w io.Writer) {
	fmt.Fprintln(w, "Configured exclude patterns:")
	if len(cfg.Exclude) == 0 {
		fmt.Fprintln(w, "  (none yet). Add patterns such as `internal/generated/*` to ignore generated proto domains.")
	} else {
		for _, pattern := range cfg.Exclude {
			fmt.Fprintf(w, "  - %s\n", pattern)
		}
	}
	fmt.Fprintln(w, "\nDomains tracked by the policy:")
	for _, d := range domains {
		fmt.Fprintf(w, "  - %s (matches: %s)\n", d.Name, strings.Join(d.Match, ", "))
	}
	fmt.Fprintln(w, "\nUse `exclude:` entries in `.coverctl.yaml` to skip generated folders (e.g., proto outputs) before running `coverctl check`.")
}

func printTrendResult(result application.TrendResult, w io.Writer) {
	trendSymbol := "→"
	switch result.Trend.Direction {
	case domain.TrendUp:
		trendSymbol = "↑"
	case domain.TrendDown:
		trendSymbol = "↓"
	}

	fmt.Fprintf(w, "Coverage Trend: %.1f%% %s %.1f%% (%+.1f%%)\n",
		result.Previous, trendSymbol, result.Current, result.Trend.Delta)
	fmt.Fprintln(w, "\nDomain Trends:")
	for name, trend := range result.ByDomain {
		symbol := "→"
		switch trend.Direction {
		case domain.TrendUp:
			symbol = "↑"
		case domain.TrendDown:
			symbol = "↓"
		}
		fmt.Fprintf(w, "  %s: %s %+.1f%%\n", name, symbol, trend.Delta)
	}
	fmt.Fprintf(w, "\nHistory: %d entries\n", len(result.Entries))
}

func printSuggestResult(result application.SuggestResult, w io.Writer) {
	fmt.Fprintln(w, "Threshold Suggestions:")
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "%-20s %10s %10s %12s  %s\n", "DOMAIN", "CURRENT", "MIN", "SUGGESTED", "REASON")
	fmt.Fprintf(w, "%-20s %10s %10s %12s  %s\n", "------", "-------", "---", "---------", "------")
	for _, s := range result.Suggestions {
		change := ""
		if s.SuggestedMin > s.CurrentMin {
			change = "↑"
		} else if s.SuggestedMin < s.CurrentMin {
			change = "↓"
		}
		fmt.Fprintf(w, "%-20s %9.1f%% %9.1f%% %10.1f%% %s  %s\n",
			s.Domain, s.CurrentPercent, s.CurrentMin, s.SuggestedMin, change, s.Reason)
	}
}

func printDebtResult(result application.DebtResult, w io.Writer, format application.OutputFormat) {
	if format == application.OutputJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
		return
	}

	// Text output
	if len(result.Items) == 0 {
		fmt.Fprintln(w, "No coverage debt found - all targets are met!")
		fmt.Fprintf(w, "Health Score: %.1f%%\n", result.HealthScore)
		return
	}

	fmt.Fprintln(w, "Coverage Debt Report")
	fmt.Fprintln(w, "====================")
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "%-8s %-30s %10s %10s %10s %8s\n", "TYPE", "NAME", "CURRENT", "REQUIRED", "SHORTFALL", "LINES")
	fmt.Fprintf(w, "%-8s %-30s %10s %10s %10s %8s\n", "----", "----", "-------", "--------", "---------", "-----")

	for _, item := range result.Items {
		name := item.Name
		if len(name) > 30 {
			name = "..." + name[len(name)-27:]
		}
		fmt.Fprintf(w, "%-8s %-30s %9.1f%% %9.1f%% %9.1f%% %8d\n",
			item.Type, name, item.Current, item.Required, item.Shortfall, item.Lines)
	}

	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "Total Debt: %.1f%% shortfall across %d items\n", result.TotalDebt, len(result.Items))
	fmt.Fprintf(w, "Estimated Lines Needing Tests: %d\n", result.TotalLines)
	fmt.Fprintf(w, "Health Score: %.1f%%\n", result.HealthScore)
}

func runWatch(ctx context.Context, stdout, stderr io.Writer, svc Service, configPath, profile string, domains []string) int {
	// Create watcher
	w, err := watcher.New(watcher.WithDebounce(500 * time.Millisecond))
	if err != nil {
		fmt.Fprintf(stderr, "failed to create watcher: %v\n", err)
		return 3
	}
	defer w.Close()

	// Handle Ctrl+C gracefully
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(stdout, "\nStopping watch mode...")
		cancel()
	}()

	fmt.Fprintln(stdout, "Watching for file changes... (Ctrl+C to stop)")
	fmt.Fprintln(stdout, "")

	callback := func(runNumber int, runErr error) {
		fmt.Fprintf(stdout, "\n--- Run #%d at %s ---\n", runNumber, time.Now().Format("15:04:05"))
		if runErr != nil {
			fmt.Fprintf(stderr, "Coverage run failed: %v\n", runErr)
		} else {
			fmt.Fprintln(stdout, "Coverage run completed successfully")
		}
	}

	opts := application.WatchOptions{
		ConfigPath: configPath,
		Profile:    profile,
		Domains:    domains,
	}

	if err := svc.Watch(ctx, opts, w, callback); err != nil {
		if ctx.Err() == context.Canceled {
			return 0 // Normal exit on Ctrl+C
		}
		fmt.Fprintf(stderr, "watch error: %v\n", err)
		return 3
	}
	return 0
}
