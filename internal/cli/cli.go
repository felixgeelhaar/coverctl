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
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/bitbucket"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/config"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/coverprofile"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/diff"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/github"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/gitlab"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/gotool"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/history"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/report"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/runners"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/watcher"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/wizard"
	"github.com/felixgeelhaar/coverctl/internal/mcp"
	"github.com/felixgeelhaar/coverctl/internal/pathutil"
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
	Compare(ctx context.Context, opts application.CompareOptions) (application.CompareResult, error)
	PRComment(ctx context.Context, opts application.PRCommentOptions) (application.PRCommentResult, error)
}

// GlobalOptions holds CLI-wide options that affect output behavior
type GlobalOptions struct {
	Quiet   bool // Suppress non-essential output
	NoColor bool // Disable colored output
	CI      bool // CI mode: quiet + no-color + GitHub Actions annotations
}

// IsQuiet returns true if output should be suppressed
func (g GlobalOptions) IsQuiet() bool {
	return g.Quiet || g.CI
}

// UseColor returns true if colored output should be used
func (g GlobalOptions) UseColor() bool {
	return !g.NoColor && !g.CI
}

var initWizard = wizard.Run

// parseGlobalFlags extracts global flags from args and returns:
// - GlobalOptions with parsed flags
// - command name (first non-flag argument)
// - remaining args after the command
func parseGlobalFlags(args []string) (GlobalOptions, string, []string) {
	var global GlobalOptions
	var cmd string
	var remaining []string

loop:
	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "-q", "--quiet":
			global.Quiet = true
		case "--no-color":
			global.NoColor = true
		case "--ci":
			global.CI = true
		default:
			// First non-global-flag is the command
			cmd = arg
			// Remaining args go to the command
			remaining = args[i+1:]
			break loop
		}
	}

	return global, cmd, remaining
}

func Run(args []string, stdout, stderr io.Writer, svc Service) int {
	if len(args) < 2 {
		usage(stderr)
		return 2
	}

	// Parse global flags and extract command
	global, cmd, cmdArgs := parseGlobalFlags(args[1:])

	// Handle global flags that exit early
	if cmd == "--version" || cmd == "-v" {
		printVersion(stdout)
		return 0
	}
	if cmd == "--help" || cmd == "-h" {
		usage(stdout)
		return 0
	}
	if cmd == "" {
		usage(stderr)
		return 2
	}

	ctx := context.Background()

	switch cmd {
	case "version":
		printVersion(stdout)
		return 0
	case "help":
		if len(cmdArgs) < 1 {
			usage(stdout)
			return 0
		}
		return commandHelp(cmdArgs[0], stdout)
	case "completion":
		return runCompletion(cmdArgs, stdout, stderr)
	case "check", "c":
		fs := flag.NewFlagSet("check", flag.ContinueOnError)
		fs.Usage = func() { commandHelp("check", stderr) }
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		fs.StringVar(configPath, "c", ".coverctl.yaml", "Config file path (shorthand)")
		output := outputFlags(fs)
		profile := fs.String("profile", ".cover/coverage.out", "Coverage profile output path")
		fs.StringVar(profile, "p", ".cover/coverage.out", "Coverage profile output path (shorthand)")
		historyPath := fs.String("history", "", "History file path for delta display")
		showDelta := fs.Bool("show-delta", false, "Show coverage change from previous run")
		failUnder := fs.Float64("fail-under", 0, "Fail if overall coverage is below this percentage")
		ratchet := fs.Bool("ratchet", false, "Fail if coverage decreases from previous recorded value")
		validate := fs.Bool("validate", false, "Validate config without running tests")
		// Build/test flags
		tags := fs.String("tags", "", "Build tags (e.g., integration,e2e)")
		race := fs.Bool("race", false, "Enable race detector")
		short := fs.Bool("short", false, "Skip long-running tests")
		verbose := fs.Bool("v", false, "Verbose test output")
		run := fs.String("run", "", "Run only tests matching pattern")
		timeout := fs.String("timeout", "", "Test timeout (e.g., 10m, 1h)")
		var testArgs testArgsList
		fs.Var(&testArgs, "test-arg", "Additional argument passed to go test (repeatable)")
		var domains domainList
		fs.Var(&domains, "domain", "Filter to specific domain (repeatable)")
		fs.Var(&domains, "d", "Filter to specific domain (shorthand)")
		// Incremental mode flags
		incremental := fs.Bool("incremental", false, "Only test packages with changed files")
		incrementalRef := fs.String("incremental-ref", "HEAD~1", "Git ref to compare against for incremental mode")
		if err := fs.Parse(cmdArgs); err != nil {
			return 2
		}
		// Validate-only mode: just check config file syntax
		if *validate {
			if err := validateConfig(*configPath); err != nil {
				return exitCodeWithCI(err, 1, stderr, global)
			}
			if !global.IsQuiet() {
				fmt.Fprintf(stdout, "Config %s is valid\n", *configPath)
			}
			return 0
		}
		opts := application.CheckOptions{
			ConfigPath:     *configPath,
			Output:         *output,
			Profile:        *profile,
			Domains:        domains,
			Incremental:    *incremental,
			IncrementalRef: *incrementalRef,
			BuildFlags: application.BuildFlags{
				Tags:     *tags,
				Race:     *race,
				Short:    *short,
				Verbose:  *verbose,
				Run:      *run,
				Timeout:  *timeout,
				TestArgs: testArgs,
			},
		}
		if *showDelta || *ratchet {
			histPath := *historyPath
			if histPath == "" {
				histPath = ".cover/history.json"
			}
			opts.HistoryStore = &history.FileStore{Path: histPath}
		}
		if *failUnder > 0 {
			opts.FailUnder = failUnder
		}
		opts.Ratchet = *ratchet
		err := svc.Check(ctx, opts)
		return exitCodeWithCI(err, 1, stderr, global)
	case "run", "r":
		fs := flag.NewFlagSet("run", flag.ContinueOnError)
		fs.Usage = func() { commandHelp("run", stderr) }
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		fs.StringVar(configPath, "c", ".coverctl.yaml", "Config file path (shorthand)")
		profile := fs.String("profile", ".cover/coverage.out", "Coverage profile output path")
		fs.StringVar(profile, "p", ".cover/coverage.out", "Coverage profile output path (shorthand)")
		// Build/test flags
		tags := fs.String("tags", "", "Build tags (e.g., integration,e2e)")
		race := fs.Bool("race", false, "Enable race detector")
		short := fs.Bool("short", false, "Skip long-running tests")
		verbose := fs.Bool("v", false, "Verbose test output")
		run := fs.String("run", "", "Run only tests matching pattern")
		timeout := fs.String("timeout", "", "Test timeout (e.g., 10m, 1h)")
		var testArgs testArgsList
		fs.Var(&testArgs, "test-arg", "Additional argument passed to go test (repeatable)")
		var domains domainList
		fs.Var(&domains, "domain", "Filter to specific domain (repeatable)")
		fs.Var(&domains, "d", "Filter to specific domain (shorthand)")
		if err := fs.Parse(cmdArgs); err != nil {
			return 2
		}

		err := svc.RunOnly(ctx, application.RunOnlyOptions{
			ConfigPath: *configPath,
			Profile:    *profile,
			Domains:    domains,
			BuildFlags: application.BuildFlags{
				Tags:     *tags,
				Race:     *race,
				Short:    *short,
				Verbose:  *verbose,
				Run:      *run,
				Timeout:  *timeout,
				TestArgs: testArgs,
			},
		})
		return exitCodeWithCI(err, 3, stderr, global)
	case "watch", "w":
		fs := flag.NewFlagSet("watch", flag.ContinueOnError)
		fs.Usage = func() { commandHelp("watch", stderr) }
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		fs.StringVar(configPath, "c", ".coverctl.yaml", "Config file path (shorthand)")
		profile := fs.String("profile", ".cover/coverage.out", "Coverage profile output path")
		fs.StringVar(profile, "p", ".cover/coverage.out", "Coverage profile output path (shorthand)")
		// Build/test flags
		tags := fs.String("tags", "", "Build tags (e.g., integration,e2e)")
		race := fs.Bool("race", false, "Enable race detector")
		short := fs.Bool("short", false, "Skip long-running tests")
		verbose := fs.Bool("v", false, "Verbose test output")
		run := fs.String("run", "", "Run only tests matching pattern")
		timeout := fs.String("timeout", "", "Test timeout (e.g., 10m, 1h)")
		var testArgs testArgsList
		fs.Var(&testArgs, "test-arg", "Additional argument passed to go test (repeatable)")
		var domains domainList
		fs.Var(&domains, "domain", "Filter to specific domain (repeatable)")
		fs.Var(&domains, "d", "Filter to specific domain (shorthand)")
		if err := fs.Parse(cmdArgs); err != nil {
			return 2
		}

		buildFlags := application.BuildFlags{
			Tags:     *tags,
			Race:     *race,
			Short:    *short,
			Verbose:  *verbose,
			Run:      *run,
			Timeout:  *timeout,
			TestArgs: testArgs,
		}
		return runWatch(ctx, stdout, stderr, svc, *configPath, *profile, domains, global, buildFlags)
	case "detect":
		fs := flag.NewFlagSet("detect", flag.ContinueOnError)
		fs.Usage = func() { commandHelp("detect", stderr) }
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		fs.StringVar(configPath, "c", ".coverctl.yaml", "Config file path (shorthand)")
		dryRun := fs.Bool("dry-run", false, "Preview config without writing")
		force := fs.Bool("force", false, "Overwrite config if it exists")
		fs.BoolVar(force, "f", false, "Overwrite config if it exists (shorthand)")
		if err := fs.Parse(cmdArgs); err != nil {
			return 2
		}
		cfg, err := svc.Detect(ctx, application.DetectOptions{})
		if err != nil {
			return exitCodeWithCI(err, 3, stderr, global)
		}
		if *dryRun {
			if err := writeConfigFile("-", cfg, stdout, false); err != nil {
				return exitCodeWithCI(err, 2, stderr, global)
			}
			return 0
		}
		if err := writeConfigFile(*configPath, cfg, stdout, *force); err != nil {
			return exitCodeWithCI(err, 2, stderr, global)
		}
		if !global.IsQuiet() {
			fmt.Fprintf(stdout, "Config written to %s\n", *configPath)
		}
		return 0
	case "report":
		fs := flag.NewFlagSet("report", flag.ContinueOnError)
		fs.Usage = func() { commandHelp("report", stderr) }
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		fs.StringVar(configPath, "c", ".coverctl.yaml", "Config file path (shorthand)")
		output := outputFlags(fs)
		profile := fs.String("profile", ".cover/coverage.out", "Coverage profile path")
		fs.StringVar(profile, "p", ".cover/coverage.out", "Coverage profile path (shorthand)")
		historyPath := fs.String("history", "", "History file path for delta display")
		showDelta := fs.Bool("show-delta", false, "Show coverage change from previous run")
		showUncovered := fs.Bool("uncovered", false, "Show only files with 0% coverage")
		diffRef := fs.String("diff", "", "Show coverage for files changed since git ref")
		var mergeProfiles profileList
		fs.Var(&mergeProfiles, "merge", "Merge additional coverage profile (repeatable)")
		var domains domainList
		fs.Var(&domains, "domain", "Filter to specific domain (repeatable)")
		fs.Var(&domains, "d", "Filter to specific domain (shorthand)")
		if err := fs.Parse(cmdArgs); err != nil {
			return 2
		}
		opts := application.ReportOptions{
			ConfigPath:    *configPath,
			Output:        *output,
			Profile:       *profile,
			Domains:       domains,
			ShowUncovered: *showUncovered,
			DiffRef:       *diffRef,
			MergeProfiles: mergeProfiles,
		}
		if *showDelta {
			histPath := *historyPath
			if histPath == "" {
				histPath = ".cover/history.json"
			}
			opts.HistoryStore = &history.FileStore{Path: histPath}
		}
		err := svc.Report(ctx, opts)
		return exitCodeWithCI(err, 3, stderr, global)
	case "ignore":
		fs := flag.NewFlagSet("ignore", flag.ContinueOnError)
		fs.Usage = func() { commandHelp("ignore", stderr) }
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		fs.StringVar(configPath, "c", ".coverctl.yaml", "Config file path (shorthand)")
		if err := fs.Parse(cmdArgs); err != nil {
			return 2
		}
		cfg, domains, err := svc.Ignore(ctx, application.IgnoreOptions{ConfigPath: *configPath})
		if err != nil {
			return exitCodeWithCI(err, 4, stderr, global)
		}
		printIgnoreInfo(cfg, domains, stdout)
		return 0
	case "init", "i":
		fs := flag.NewFlagSet("init", flag.ContinueOnError)
		fs.Usage = func() { commandHelp("init", stderr) }
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		fs.StringVar(configPath, "c", ".coverctl.yaml", "Config file path (shorthand)")
		force := fs.Bool("force", false, "Overwrite existing config file")
		fs.BoolVar(force, "f", false, "Overwrite existing config file (shorthand)")
		noInteractive := fs.Bool("no-interactive", false, "Skip the interactive init wizard")
		if err := fs.Parse(cmdArgs); err != nil {
			return 2
		}
		cfg, err := svc.Detect(ctx, application.DetectOptions{})
		if err != nil {
			return exitCodeWithCI(err, 3, stderr, global)
		}
		if !*noInteractive {
			var confirmed bool
			cfg, confirmed, err = initWizard(cfg, stdout, os.Stdin)
			if err != nil {
				return exitCodeWithCI(err, 5, stderr, global)
			}
			if !confirmed {
				if !global.IsQuiet() {
					fmt.Fprintln(stdout, "Init canceled; no configuration written.")
				}
				return 0
			}
		}
		if err := writeConfigFile(*configPath, cfg, stdout, *force); err != nil {
			return exitCodeWithCI(err, 2, stderr, global)
		}
		return 0
	case "badge":
		fs := flag.NewFlagSet("badge", flag.ContinueOnError)
		fs.Usage = func() { commandHelp("badge", stderr) }
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		fs.StringVar(configPath, "c", ".coverctl.yaml", "Config file path (shorthand)")
		profile := fs.String("profile", ".cover/coverage.out", "Coverage profile path")
		fs.StringVar(profile, "p", ".cover/coverage.out", "Coverage profile path (shorthand)")
		output := fs.String("output", "coverage.svg", "Output file path")
		fs.StringVar(output, "o", "coverage.svg", "Output file path (shorthand)")
		label := fs.String("label", "coverage", "Badge label text")
		style := fs.String("style", "flat", "Badge style: flat|flat-square")
		if err := fs.Parse(cmdArgs); err != nil {
			return 2
		}
		result, err := svc.Badge(ctx, application.BadgeOptions{
			ConfigPath:  *configPath,
			ProfilePath: *profile,
			Output:      *output,
			Label:       *label,
			Style:       *style,
		})
		if err != nil {
			return exitCodeWithCI(err, 3, stderr, global)
		}
		if err := writeBadgeFile(*output, result.Percent, *label, *style); err != nil {
			return exitCodeWithCI(err, 3, stderr, global)
		}
		if !global.IsQuiet() {
			fmt.Fprintf(stdout, "Badge written to %s (%.1f%%)\n", *output, result.Percent)
		}
		return 0
	case "trend":
		fs := flag.NewFlagSet("trend", flag.ContinueOnError)
		fs.Usage = func() { commandHelp("trend", stderr) }
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		fs.StringVar(configPath, "c", ".coverctl.yaml", "Config file path (shorthand)")
		profile := fs.String("profile", ".cover/coverage.out", "Coverage profile path")
		fs.StringVar(profile, "p", ".cover/coverage.out", "Coverage profile path (shorthand)")
		historyPath := fs.String("history", ".cover/history.json", "History file path")
		output := outputFlags(fs)
		if err := fs.Parse(cmdArgs); err != nil {
			return 2
		}
		store := history.FileStore{Path: *historyPath}
		result, err := svc.Trend(ctx, application.TrendOptions{
			ConfigPath:  *configPath,
			ProfilePath: *profile,
			HistoryPath: *historyPath,
			Output:      *output,
		}, &store)
		if err != nil {
			return exitCodeWithCI(err, 3, stderr, global)
		}
		printTrendResult(result, stdout)
		return 0
	case "record":
		fs := flag.NewFlagSet("record", flag.ContinueOnError)
		fs.Usage = func() { commandHelp("record", stderr) }
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		fs.StringVar(configPath, "c", ".coverctl.yaml", "Config file path (shorthand)")
		profile := fs.String("profile", ".cover/coverage.out", "Coverage profile path")
		fs.StringVar(profile, "p", ".cover/coverage.out", "Coverage profile path (shorthand)")
		historyPath := fs.String("history", ".cover/history.json", "History file path")
		commit := fs.String("commit", "", "Git commit SHA (optional)")
		branch := fs.String("branch", "", "Git branch name (optional)")
		if err := fs.Parse(cmdArgs); err != nil {
			return 2
		}
		store := history.FileStore{Path: *historyPath}
		err := svc.Record(ctx, application.RecordOptions{
			ConfigPath:  *configPath,
			ProfilePath: *profile,
			HistoryPath: *historyPath,
			Commit:      *commit,
			Branch:      *branch,
		}, &store)
		if err != nil {
			return exitCodeWithCI(err, 3, stderr, global)
		}
		if !global.IsQuiet() {
			fmt.Fprintln(stdout, "Coverage recorded to history")
		}
		return 0
	case "suggest":
		fs := flag.NewFlagSet("suggest", flag.ContinueOnError)
		fs.Usage = func() { commandHelp("suggest", stderr) }
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		fs.StringVar(configPath, "c", ".coverctl.yaml", "Config file path (shorthand)")
		profile := fs.String("profile", ".cover/coverage.out", "Coverage profile path")
		fs.StringVar(profile, "p", ".cover/coverage.out", "Coverage profile path (shorthand)")
		strategy := fs.String("strategy", "current", "Suggestion strategy: current|aggressive|conservative")
		apply := fs.Bool("apply", false, "Update config with suggested thresholds")
		force := fs.Bool("force", false, "Overwrite config if it exists")
		fs.BoolVar(force, "f", false, "Overwrite config if it exists (shorthand)")
		if err := fs.Parse(cmdArgs); err != nil {
			return 2
		}

		var suggestStrat application.SuggestStrategy
		switch *strategy {
		case "aggressive":
			suggestStrat = application.SuggestAggressive
		case "conservative":
			suggestStrat = application.SuggestConservative
		default:
			suggestStrat = application.SuggestCurrent
		}

		result, err := svc.Suggest(ctx, application.SuggestOptions{
			ConfigPath:  *configPath,
			ProfilePath: *profile,
			Strategy:    suggestStrat,
		})
		if err != nil {
			return exitCodeWithCI(err, 3, stderr, global)
		}
		printSuggestResult(result, stdout)
		if *apply {
			if err := writeConfigFile(*configPath, result.Config, stdout, *force); err != nil {
				return exitCodeWithCI(err, 2, stderr, global)
			}
			if !global.IsQuiet() {
				fmt.Fprintf(stdout, "\nConfig updated: %s\n", *configPath)
			}
		}
		return 0
	case "debt":
		fs := flag.NewFlagSet("debt", flag.ContinueOnError)
		fs.Usage = func() { commandHelp("debt", stderr) }
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		fs.StringVar(configPath, "c", ".coverctl.yaml", "Config file path (shorthand)")
		profile := fs.String("profile", ".cover/coverage.out", "Coverage profile path")
		fs.StringVar(profile, "p", ".cover/coverage.out", "Coverage profile path (shorthand)")
		output := outputFlags(fs)
		if err := fs.Parse(cmdArgs); err != nil {
			return 2
		}

		result, err := svc.Debt(ctx, application.DebtOptions{
			ConfigPath:  *configPath,
			ProfilePath: *profile,
			Output:      *output,
		})
		if err != nil {
			return exitCodeWithCI(err, 3, stderr, global)
		}
		printDebtResult(result, stdout, *output)
		return 0
	case "compare":
		fs := flag.NewFlagSet("compare", flag.ContinueOnError)
		fs.Usage = func() { commandHelp("compare", stderr) }
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		fs.StringVar(configPath, "c", ".coverctl.yaml", "Config file path (shorthand)")
		baseProfile := fs.String("base", "", "Base coverage profile (required)")
		fs.StringVar(baseProfile, "b", "", "Base coverage profile (shorthand)")
		headProfile := fs.String("head", ".cover/coverage.out", "Head coverage profile to compare against")
		fs.StringVar(headProfile, "H", ".cover/coverage.out", "Head coverage profile (shorthand)")
		output := outputFlags(fs)
		if err := fs.Parse(cmdArgs); err != nil {
			return 2
		}

		if *baseProfile == "" {
			fmt.Fprintln(stderr, "Error: --base flag is required")
			fs.Usage()
			return 2
		}

		result, err := svc.Compare(ctx, application.CompareOptions{
			ConfigPath:  *configPath,
			BaseProfile: *baseProfile,
			HeadProfile: *headProfile,
			Output:      *output,
		})
		if err != nil {
			return exitCodeWithCI(err, 3, stderr, global)
		}
		printCompareResult(result, stdout, *output)
		return 0
	case "pr-comment":
		fs := flag.NewFlagSet("pr-comment", flag.ContinueOnError)
		fs.Usage = func() { commandHelp("pr-comment", stderr) }
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		fs.StringVar(configPath, "c", ".coverctl.yaml", "Config file path (shorthand)")
		profilePath := fs.String("profile", ".cover/coverage.out", "Coverage profile path")
		fs.StringVar(profilePath, "p", ".cover/coverage.out", "Coverage profile path (shorthand)")
		baseProfile := fs.String("base", "", "Base coverage profile for comparison (optional)")
		prNumber := fs.Int("pr", 0, "Pull request/MR number (required)")
		owner := fs.String("owner", "", "Repository owner/namespace (auto-detected from env)")
		repo := fs.String("repo", "", "Repository name (auto-detected from env)")
		provider := fs.String("provider", "auto", "Git provider: github, gitlab, bitbucket, or auto")
		updateExisting := fs.Bool("update", true, "Update existing comment instead of creating new")
		dryRun := fs.Bool("dry-run", false, "Generate comment without posting")
		if err := fs.Parse(cmdArgs); err != nil {
			return 2
		}

		// Parse provider
		var prProvider application.PRProvider
		switch strings.ToLower(*provider) {
		case "github":
			prProvider = application.ProviderGitHub
		case "gitlab":
			prProvider = application.ProviderGitLab
		case "bitbucket":
			prProvider = application.ProviderBitbucket
		case "auto", "":
			prProvider = application.ProviderAuto
		default:
			fmt.Fprintf(stderr, "Error: unknown provider %q (use github, gitlab, bitbucket, or auto)\n", *provider)
			return 2
		}

		// Auto-detect owner/repo from environment based on provider
		ownerVal, repoVal, prNum := *owner, *repo, *prNumber
		ownerVal, repoVal, prNum = detectPRContext(prProvider, ownerVal, repoVal, prNum)

		if prNum == 0 {
			fmt.Fprintln(stderr, "Error: --pr flag is required")
			fs.Usage()
			return 2
		}
		if ownerVal == "" || repoVal == "" {
			fmt.Fprintln(stderr, "Error: --owner and --repo flags are required (or set provider-specific env vars)")
			fmt.Fprintln(stderr, "  GitHub: GITHUB_REPOSITORY=owner/repo")
			fmt.Fprintln(stderr, "  GitLab: CI_PROJECT_NAMESPACE and CI_PROJECT_NAME")
			fmt.Fprintln(stderr, "  Bitbucket: BITBUCKET_WORKSPACE and BITBUCKET_REPO_SLUG")
			return 2
		}

		result, err := svc.PRComment(ctx, application.PRCommentOptions{
			ConfigPath:     *configPath,
			ProfilePath:    *profilePath,
			BaseProfile:    *baseProfile,
			Provider:       prProvider,
			PRNumber:       prNum,
			Owner:          ownerVal,
			Repo:           repoVal,
			UpdateExisting: *updateExisting,
			DryRun:         *dryRun,
		})
		if err != nil {
			return exitCodeWithCI(err, 3, stderr, global)
		}

		if *dryRun {
			fmt.Fprintln(stdout, result.CommentBody)
		} else if result.Created {
			fmt.Fprintf(stdout, "Created comment: %s\n", result.CommentURL)
		} else {
			fmt.Fprintf(stdout, "Updated comment #%d\n", result.CommentID)
		}
		return 0
	case "mcp":
		if len(cmdArgs) < 1 {
			fmt.Fprintln(stderr, "Usage: coverctl mcp <subcommand>")
			fmt.Fprintln(stderr, "Subcommands: serve")
			return 2
		}
		switch cmdArgs[0] {
		case "serve":
			fs := flag.NewFlagSet("mcp serve", flag.ContinueOnError)
			fs.Usage = func() { commandHelp("mcp", stderr) }
			configPath := fs.String("config", ".coverctl.yaml", "Config file path")
			fs.StringVar(configPath, "c", ".coverctl.yaml", "Config file path (shorthand)")
			historyPath := fs.String("history", ".cover/history.json", "History file path")
			profilePath := fs.String("profile", ".cover/coverage.out", "Coverage profile path")
			fs.StringVar(profilePath, "p", ".cover/coverage.out", "Coverage profile path (shorthand)")
			if err := fs.Parse(cmdArgs[1:]); err != nil {
				return 2
			}

			// Build MCP server with application service and CLI version
			mcpSvc := BuildService(os.Stdout)
			mcpServer := mcp.New(mcpSvc, mcp.Config{
				ConfigPath:  *configPath,
				HistoryPath: *historyPath,
				ProfilePath: *profilePath,
			}, Version)

			// Handle signals for graceful shutdown
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			if err := mcpServer.Run(ctx); err != nil {
				fmt.Fprintf(stderr, "MCP server error: %v\n", err)
				return 1
			}
			return 0
		default:
			fmt.Fprintf(stderr, "Unknown mcp subcommand: %s\n", cmdArgs[0])
			fmt.Fprintln(stderr, "Available subcommands: serve")
			return 2
		}
	default:
		usage(stderr)
		return 2
	}
}

func BuildService(out *os.File) *application.Service {
	module := gotool.NewCachedModuleResolver()
	// Use the runner registry for language auto-detection.
	// The registry will detect the project type and delegate to the appropriate runner.
	registry := runners.NewRegistry(module)
	return &application.Service{
		ConfigLoader:      config.Loader{},
		Autodetector:      autodetect.Detector{Module: module},
		DomainResolver:    gotool.DomainResolver{Module: module},
		CoverageRunner:    registry,
		ProfileParser:     coverprofile.Parser{},
		DiffProvider:      diff.GitDiff{Module: module},
		AnnotationScanner: annotations.Scanner{},
		Reporter:          report.Writer{},
		PRClients:         buildPRClients(),
		CommentFormatter:  commentFormatter{},
		Out:               out,
	}
}

// buildPRClients creates clients for all supported PR providers.
func buildPRClients() map[application.PRProvider]application.PRClient {
	return map[application.PRProvider]application.PRClient{
		application.ProviderGitHub:    github.NewClient(""),
		application.ProviderGitLab:    gitlab.NewClient(""),
		application.ProviderBitbucket: bitbucket.NewClient("", ""),
	}
}

// detectPRContext auto-detects owner, repo, and PR number from environment variables.
// Returns the provided values if already set, otherwise tries to detect from env.
func detectPRContext(provider application.PRProvider, owner, repo string, prNumber int) (string, string, int) {
	// If all values are already provided, return them
	if owner != "" && repo != "" && prNumber != 0 {
		return owner, repo, prNumber
	}

	// GitHub: GITHUB_REPOSITORY=owner/repo
	if (provider == application.ProviderGitHub || provider == application.ProviderAuto) && (owner == "" || repo == "") {
		if ghRepo := os.Getenv("GITHUB_REPOSITORY"); ghRepo != "" {
			parts := strings.SplitN(ghRepo, "/", 2)
			if len(parts) == 2 {
				if owner == "" {
					owner = parts[0]
				}
				if repo == "" {
					repo = parts[1]
				}
			}
		}
	}

	// GitLab: CI_PROJECT_NAMESPACE and CI_PROJECT_NAME
	if (provider == application.ProviderGitLab || provider == application.ProviderAuto) && (owner == "" || repo == "") {
		if ns := os.Getenv("CI_PROJECT_NAMESPACE"); ns != "" && owner == "" {
			owner = ns
		}
		if name := os.Getenv("CI_PROJECT_NAME"); name != "" && repo == "" {
			repo = name
		}
		// GitLab can also auto-detect MR number
		if prNumber == 0 {
			if mrIID := os.Getenv("CI_MERGE_REQUEST_IID"); mrIID != "" {
				if n, err := parseInt(mrIID); err == nil {
					prNumber = n
				}
			}
		}
	}

	// Bitbucket: BITBUCKET_WORKSPACE and BITBUCKET_REPO_SLUG
	if (provider == application.ProviderBitbucket || provider == application.ProviderAuto) && (owner == "" || repo == "") {
		if ws := os.Getenv("BITBUCKET_WORKSPACE"); ws != "" && owner == "" {
			owner = ws
		}
		if slug := os.Getenv("BITBUCKET_REPO_SLUG"); slug != "" && repo == "" {
			repo = slug
		}
		// Bitbucket can also auto-detect PR number
		if prNumber == 0 {
			if prID := os.Getenv("BITBUCKET_PR_ID"); prID != "" {
				if n, err := parseInt(prID); err == nil {
					prNumber = n
				}
			}
		}
	}

	return owner, repo, prNumber
}

// parseInt is a helper to parse integers from strings.
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// commentFormatter wraps github.FormatCoverageComment to implement CommentFormatter interface.
type commentFormatter struct{}

func (commentFormatter) FormatCoverageComment(result domain.Result, comparison *application.CompareResult) string {
	return github.FormatCoverageComment(result, comparison)
}

func outputFlags(fs *flag.FlagSet) *application.OutputFormat {
	output := application.OutputText
	fs.Var((*outputValue)(&output), "output", "Output format: text|json|html|brief")
	fs.Var((*outputValue)(&output), "o", "Output format: text|json|html|brief")
	return &output
}

type outputValue application.OutputFormat

func (o *outputValue) String() string { return string(*o) }

func (o *outputValue) Set(value string) error {
	switch value {
	case string(application.OutputText), string(application.OutputJSON), string(application.OutputHTML), string(application.OutputBrief):
		*o = outputValue(value)
		return nil
	default:
		return fmt.Errorf("invalid output format: %s (valid: text, json, html, brief)", value)
	}
}

// domainList implements flag.Value for repeatable --domain flags
type domainList []string

func (d *domainList) String() string { return strings.Join(*d, ",") }

func (d *domainList) Set(value string) error {
	*d = append(*d, value)
	return nil
}

// profileList implements flag.Value for repeatable --merge flags
type profileList []string

func (p *profileList) String() string { return strings.Join(*p, ",") }

func (p *profileList) Set(value string) error {
	*p = append(*p, value)
	return nil
}

// testArgsList implements flag.Value for repeatable --test-arg flags
type testArgsList []string

func (t *testArgsList) String() string { return strings.Join(*t, " ") }

func (t *testArgsList) Set(value string) error {
	*t = append(*t, value)
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
	cleanPath, err := pathutil.ValidatePath(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	file, err := os.Create(cleanPath) // #nosec G304 - path is validated above
	if err != nil {
		return err
	}
	defer file.Close()
	return config.Write(file, cfg)
}

// validateConfig checks if the config file is valid without running tests
func validateConfig(path string) error {
	_, err := config.Loader{}.Load(path)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	return nil
}

func usage(w io.Writer) {
	fmt.Fprintf(w, `coverctl - Domain-driven coverage enforcement for Go

Usage:
  coverctl [global-flags] <command> [flags]
  coverctl [--version | --help]

Global Flags:
  -q, --quiet     Suppress non-essential output
      --no-color  Disable colored output
      --ci        CI mode: quiet + GitHub Actions annotations

Commands:
  check, c    Run coverage and enforce policy
  run, r      Run coverage only, produce artifacts
  watch, w    Watch for file changes and re-run coverage
  init, i     Interactive setup wizard
  detect      Autodetect domains and write config
  report      Analyze an existing profile
  badge       Generate an SVG coverage badge
  trend       Show coverage trends over time
  record      Record current coverage to history
  suggest     Suggest optimal coverage thresholds
  debt        Show coverage debt report
  compare     Compare coverage between two profiles
  ignore      Show configured excludes and ignore advice
  pr-comment  Post coverage report as PR/MR comment (GitHub, GitLab, Bitbucket)
  mcp         MCP (Model Context Protocol) server for AI agents

Other:
  help        Show help for a command
  version     Show version information
  completion  Generate shell completion scripts

Version: %s

Run 'coverctl help <command>' for more information on a command.
`, Version)
}

func writeBadgeFile(path string, percent float64, label, style string) error {
	cleanPath, err := pathutil.ValidatePath(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	file, err := os.Create(cleanPath) // #nosec G304 - path is validated above
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

// exitCodeWithCI outputs errors in GitHub Actions annotation format when CI mode is enabled
func exitCodeWithCI(err error, code int, stderr io.Writer, global GlobalOptions) int {
	if err == nil {
		return 0
	}
	if global.CI {
		// GitHub Actions annotation format
		fmt.Fprintf(stderr, "::error::%s\n", err)
	} else {
		fmt.Fprintln(stderr, err)
	}
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

func printCompareResult(result application.CompareResult, w io.Writer, format application.OutputFormat) {
	if format == application.OutputJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
		return
	}

	// Text output
	fmt.Fprintln(w, "Coverage Comparison")
	fmt.Fprintln(w, "===================")
	fmt.Fprintln(w, "")

	// Overall summary
	deltaSign := "+"
	if result.Delta < 0 {
		deltaSign = ""
	}
	fmt.Fprintf(w, "Overall: %.1f%% → %.1f%% (%s%.1f%%)\n", result.BaseOverall, result.HeadOverall, deltaSign, result.Delta)
	fmt.Fprintln(w, "")

	// Domain deltas if available
	if len(result.DomainDeltas) > 0 {
		fmt.Fprintln(w, "Domain Changes:")
		for domain, delta := range result.DomainDeltas {
			sign := "+"
			if delta < 0 {
				sign = ""
			}
			if delta > 0.1 || delta < -0.1 {
				fmt.Fprintf(w, "  %-20s %s%.1f%%\n", domain, sign, delta)
			}
		}
		fmt.Fprintln(w, "")
	}

	// Improved files
	if len(result.Improved) > 0 {
		fmt.Fprintf(w, "Improved Files (%d):\n", len(result.Improved))
		for i, f := range result.Improved {
			if i >= 10 {
				fmt.Fprintf(w, "  ... and %d more\n", len(result.Improved)-10)
				break
			}
			fmt.Fprintf(w, "  %-50s %.1f%% → %.1f%% (+%.1f%%)\n", truncateLeft(f.File, 50), f.BasePct, f.HeadPct, f.Delta)
		}
		fmt.Fprintln(w, "")
	}

	// Regressed files
	if len(result.Regressed) > 0 {
		fmt.Fprintf(w, "Regressed Files (%d):\n", len(result.Regressed))
		for i, f := range result.Regressed {
			if i >= 10 {
				fmt.Fprintf(w, "  ... and %d more\n", len(result.Regressed)-10)
				break
			}
			fmt.Fprintf(w, "  %-50s %.1f%% → %.1f%% (%.1f%%)\n", truncateLeft(f.File, 50), f.BasePct, f.HeadPct, f.Delta)
		}
		fmt.Fprintln(w, "")
	}

	// Summary
	fmt.Fprintf(w, "Summary: %d improved, %d regressed, %d unchanged\n", len(result.Improved), len(result.Regressed), result.Unchanged)
}

func truncateLeft(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return "..." + s[len(s)-(maxLen-3):]
}

func runWatch(ctx context.Context, stdout, stderr io.Writer, svc Service, configPath, profile string, domains []string, global GlobalOptions, buildFlags application.BuildFlags) int {
	// Create watcher
	w, err := watcher.New(watcher.WithDebounce(500 * time.Millisecond))
	if err != nil {
		if global.CI {
			fmt.Fprintf(stderr, "::error::failed to create watcher: %v\n", err)
		} else {
			fmt.Fprintf(stderr, "failed to create watcher: %v\n", err)
		}
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
		if !global.IsQuiet() {
			fmt.Fprintln(stdout, "\nStopping watch mode...")
		}
		cancel()
	}()

	if !global.IsQuiet() {
		fmt.Fprintln(stdout, "Watching for file changes... (Ctrl+C to stop)")
		fmt.Fprintln(stdout, "")
	}

	callback := func(runNumber int, runErr error) {
		if !global.IsQuiet() {
			fmt.Fprintf(stdout, "\n--- Run #%d at %s ---\n", runNumber, time.Now().Format("15:04:05"))
		}
		if runErr != nil {
			if global.CI {
				fmt.Fprintf(stderr, "::error::Coverage run failed: %v\n", runErr)
			} else {
				fmt.Fprintf(stderr, "Coverage run failed: %v\n", runErr)
			}
		} else if !global.IsQuiet() {
			fmt.Fprintln(stdout, "Coverage run completed successfully")
		}
	}

	opts := application.WatchOptions{
		ConfigPath: configPath,
		Profile:    profile,
		Domains:    domains,
		BuildFlags: buildFlags,
	}

	if err := svc.Watch(ctx, opts, w, callback); err != nil {
		if ctx.Err() == context.Canceled {
			return 0 // Normal exit on Ctrl+C
		}
		if global.CI {
			fmt.Fprintf(stderr, "::error::watch error: %v\n", err)
		} else {
			fmt.Fprintf(stderr, "watch error: %v\n", err)
		}
		return 3
	}
	return 0
}

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "coverctl version %s\n", Version)
	if Commit != "unknown" {
		fmt.Fprintf(w, "  commit: %s\n", Commit)
	}
	if Date != "unknown" {
		fmt.Fprintf(w, "  built:  %s\n", Date)
	}
}

var commandHelpText = map[string]string{
	"check": `coverctl check - Run coverage and enforce policy

Usage:
  coverctl check [flags]

Aliases:
  c

Flags:
  -c, --config string    Config file path (default ".coverctl.yaml")
  -p, --profile string   Coverage profile output path (default ".cover/coverage.out")
  -d, --domain string    Filter to specific domain (repeatable)
  -o, --output string    Output format: text|json|html|brief (default "text")
                         Use 'brief' for single-line LLM/agent-optimized output
      --show-delta       Show coverage change from previous run
      --history string   History file path for delta display
      --fail-under N     Fail if overall coverage is below N percent
      --ratchet          Fail if coverage decreases from previous recorded value
      --validate         Validate config file without running tests

Build/Test Flags:
      --tags string      Build tags (e.g., integration,e2e)
      --race             Enable race detector
      --short            Skip long-running tests
  -v                     Verbose test output
      --run string       Run only tests matching pattern
      --timeout string   Test timeout (e.g., 10m, 1h)
      --test-arg string  Additional argument passed to go test (repeatable)

Examples:
  coverctl check
  coverctl check -c custom.yaml
  coverctl check --fail-under 80
  coverctl check --ratchet
  coverctl check --validate
  coverctl check --tags integration
  coverctl check --race --timeout 30m
  coverctl c -d core -d api`,

	"run": `coverctl run - Run coverage only, produce artifacts

Usage:
  coverctl run [flags]

Aliases:
  r

Flags:
  -c, --config string    Config file path (default ".coverctl.yaml")
  -p, --profile string   Coverage profile output path (default ".cover/coverage.out")
  -d, --domain string    Filter to specific domain (repeatable)

Build/Test Flags:
      --tags string      Build tags (e.g., integration,e2e)
      --race             Enable race detector
      --short            Skip long-running tests
  -v                     Verbose test output
      --run string       Run only tests matching pattern
      --timeout string   Test timeout (e.g., 10m, 1h)
      --test-arg string  Additional argument passed to go test (repeatable)

Examples:
  coverctl run
  coverctl run --tags integration
  coverctl run --race -v
  coverctl r -p coverage.out`,

	"watch": `coverctl watch - Watch for file changes and re-run coverage

Usage:
  coverctl watch [flags]

Aliases:
  w

Flags:
  -c, --config string    Config file path (default ".coverctl.yaml")
  -p, --profile string   Coverage profile output path (default ".cover/coverage.out")
  -d, --domain string    Filter to specific domain (repeatable)

Build/Test Flags:
      --tags string      Build tags (e.g., integration,e2e)
      --race             Enable race detector
      --short            Skip long-running tests
  -v                     Verbose test output
      --run string       Run only tests matching pattern
      --timeout string   Test timeout (e.g., 10m, 1h)
      --test-arg string  Additional argument passed to go test (repeatable)

Examples:
  coverctl watch
  coverctl watch --tags integration
  coverctl w -d core`,

	"init": `coverctl init - Interactive setup wizard

Usage:
  coverctl init [flags]

Aliases:
  i

Flags:
  -c, --config string    Config file path (default ".coverctl.yaml")
  -f, --force            Overwrite existing config file
      --no-interactive   Skip the interactive init wizard

Examples:
  coverctl init
  coverctl i -f`,

	"detect": `coverctl detect - Autodetect domains and write config

Usage:
  coverctl detect [flags]

Flags:
  -c, --config string    Config file path (default ".coverctl.yaml")
  -f, --force            Overwrite config if it exists
      --dry-run          Preview config without writing

Examples:
  coverctl detect
  coverctl detect --dry-run
  coverctl detect -f`,

	"report": `coverctl report - Analyze an existing profile

Usage:
  coverctl report [flags]

Flags:
  -c, --config string    Config file path (default ".coverctl.yaml")
  -p, --profile string   Coverage profile path (default ".cover/coverage.out")
  -d, --domain string    Filter to specific domain (repeatable)
  -o, --output string    Output format: text|json|html|brief (default "text")
                         Use 'brief' for single-line LLM/agent-optimized output
      --show-delta       Show coverage change from previous run
      --history string   History file path for delta display
      --uncovered        Show only files with 0% coverage
      --diff <ref>       Show coverage for files changed since git ref
      --merge <file>     Merge additional coverage profile (repeatable)

Examples:
  coverctl report
  coverctl report -p custom.out -o json
  coverctl report -o html > coverage.html
  coverctl report --uncovered
  coverctl report --diff main
  coverctl report --merge integration.out --merge e2e.out`,

	"badge": `coverctl badge - Generate an SVG coverage badge

Usage:
  coverctl badge [flags]

Flags:
  -c, --config string    Config file path (default ".coverctl.yaml")
  -p, --profile string   Coverage profile path (default ".cover/coverage.out")
  -o, --output string    Output file path (default "coverage.svg")
      --label string     Badge label text (default "coverage")
      --style string     Badge style: flat|flat-square (default "flat")

Examples:
  coverctl badge
  coverctl badge -o badge.svg --style flat-square`,

	"trend": `coverctl trend - Show coverage trends over time

Usage:
  coverctl trend [flags]

Flags:
  -c, --config string    Config file path (default ".coverctl.yaml")
  -p, --profile string   Coverage profile path (default ".cover/coverage.out")
      --history string   History file path (default ".cover/history.json")
  -o, --output string    Output format: text|json|html|brief (default "text")

Examples:
  coverctl trend
  coverctl trend -o json`,

	"record": `coverctl record - Record current coverage to history

Usage:
  coverctl record [flags]

Flags:
  -c, --config string    Config file path (default ".coverctl.yaml")
  -p, --profile string   Coverage profile path (default ".cover/coverage.out")
      --history string   History file path (default ".cover/history.json")
      --commit string    Git commit SHA (optional)
      --branch string    Git branch name (optional)

Examples:
  coverctl record
  coverctl record --commit abc123 --branch main`,

	"suggest": `coverctl suggest - Suggest optimal coverage thresholds

Usage:
  coverctl suggest [flags]

Flags:
  -c, --config string    Config file path (default ".coverctl.yaml")
  -p, --profile string   Coverage profile path (default ".cover/coverage.out")
      --strategy string  Suggestion strategy: current|aggressive|conservative (default "current")
      --apply            Update config with suggested thresholds
  -f, --force            Overwrite config if it exists

Examples:
  coverctl suggest
  coverctl suggest --strategy aggressive --apply`,

	"debt": `coverctl debt - Show coverage debt report

Usage:
  coverctl debt [flags]

Flags:
  -c, --config string    Config file path (default ".coverctl.yaml")
  -p, --profile string   Coverage profile path (default ".cover/coverage.out")
  -o, --output string    Output format: text|json|brief (default "text")

Examples:
  coverctl debt
  coverctl debt -o json`,

	"ignore": `coverctl ignore - Show configured excludes and ignore advice

Usage:
  coverctl ignore [flags]

Flags:
  -c, --config string    Config file path (default ".coverctl.yaml")

Examples:
  coverctl ignore`,

	"compare": `coverctl compare - Compare coverage between two profiles

Usage:
  coverctl compare [flags]

Flags:
  -c, --config string    Config file path (default ".coverctl.yaml")
  -b, --base string      Base coverage profile (required)
  -H, --head string      Head coverage profile (default ".cover/coverage.out")
  -o, --output string    Output format: text|json|brief (default "text")

Examples:
  coverctl compare --base main.out --head feature.out
  coverctl compare -b main.out -o json`,

	"pr-comment": `coverctl pr-comment - Post coverage report as PR/MR comment

Supports GitHub, GitLab, and Bitbucket. Provider is auto-detected from
environment variables or can be specified with --provider.

Usage:
  coverctl pr-comment [flags]

Flags:
  -c, --config string    Config file path (default ".coverctl.yaml")
  -p, --profile string   Coverage profile path (default ".cover/coverage.out")
      --base string      Base coverage profile for comparison (optional)
      --pr int           Pull request/MR number (required, auto-detected on GitLab/Bitbucket)
      --owner string     Repository owner/namespace (auto-detected from env)
      --repo string      Repository name (auto-detected from env)
      --provider string  Git provider: github, gitlab, bitbucket, or auto (default "auto")
      --update           Update existing comment instead of creating new (default true)
      --dry-run          Generate comment without posting

Environment Variables:
  GitHub:
    GITHUB_TOKEN           API token for authentication
    GITHUB_REPOSITORY      Repository in owner/repo format

  GitLab:
    GITLAB_TOKEN           API token (or CI_JOB_TOKEN in GitLab CI)
    CI_PROJECT_NAMESPACE   Project namespace (auto-set in GitLab CI)
    CI_PROJECT_NAME        Project name (auto-set in GitLab CI)
    CI_MERGE_REQUEST_IID   MR number (auto-set in GitLab CI)

  Bitbucket:
    BITBUCKET_USERNAME     Username for basic auth
    BITBUCKET_APP_PASSWORD App password for authentication
    BITBUCKET_WORKSPACE    Workspace name
    BITBUCKET_REPO_SLUG    Repository slug
    BITBUCKET_PR_ID        PR number (auto-set in Bitbucket Pipelines)

Examples:
  # GitHub (auto-detected)
  coverctl pr-comment --pr 123

  # GitLab (in CI, auto-detects everything)
  coverctl pr-comment --provider gitlab

  # Bitbucket with explicit values
  coverctl pr-comment --provider bitbucket --owner myworkspace --repo myrepo --pr 45

  # Dry run to preview comment
  coverctl pr-comment --pr 123 --dry-run`,

	"mcp": `coverctl mcp - MCP (Model Context Protocol) server for AI agents

Usage:
  coverctl mcp <subcommand> [flags]

Subcommands:
  serve       Start the MCP server (STDIO transport)

Flags for 'serve':
  -c, --config string    Config file path (default ".coverctl.yaml")
  -p, --profile string   Coverage profile path (default ".cover/coverage.out")
      --history string   History file path (default ".cover/history.json")

Description:
  The MCP server enables AI agents (like Claude) to interact with coverctl
  programmatically. It exposes coverage tools and resources via the Model
  Context Protocol using STDIO transport.

Tools (actions):
  check     Run coverage tests and enforce policy thresholds
  report    Analyze an existing coverage profile
  record    Record current coverage to history

Resources (read-only queries):
  coverctl://debt      Coverage debt metrics
  coverctl://trend     Coverage trends over time
  coverctl://suggest   Threshold recommendations
  coverctl://config    Current configuration

Claude Desktop Configuration:
  Add to ~/.config/claude/claude_desktop_config.json:

  {
    "mcpServers": {
      "coverctl": {
        "command": "coverctl",
        "args": ["mcp", "serve"],
        "cwd": "/path/to/your/go/project"
      }
    }
  }

Examples:
  coverctl mcp serve
  coverctl mcp serve -c custom.yaml
  coverctl mcp serve --history .cover/history.json`,
}

func commandHelp(cmd string, w io.Writer) int {
	if help, ok := commandHelpText[cmd]; ok {
		fmt.Fprintln(w, help)
		return 0
	}
	fmt.Fprintf(w, "Unknown command: %s\n\n", cmd)
	usage(w)
	return 2
}

func runCompletion(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "Usage: coverctl completion <bash|zsh|fish>")
		return 2
	}

	switch args[0] {
	case "bash":
		fmt.Fprintln(stdout, bashCompletion)
	case "zsh":
		fmt.Fprintln(stdout, zshCompletion)
	case "fish":
		fmt.Fprintln(stdout, fishCompletion)
	default:
		fmt.Fprintf(stderr, "Unknown shell: %s\nSupported: bash, zsh, fish\n", args[0])
		return 2
	}
	return 0
}

const bashCompletion = `# coverctl bash completion
_coverctl() {
    local cur prev commands global_flags
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    commands="check run watch init detect report badge trend record suggest debt ignore mcp help version completion c r w i"
    global_flags="-q --quiet --no-color --ci"

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands} ${global_flags}" -- ${cur}) )
        return 0
    fi

    case "${prev}" in
        -c|--config)
            COMPREPLY=( $(compgen -f -X '!*.yaml' -- ${cur}) )
            return 0
            ;;
        -p|--profile)
            COMPREPLY=( $(compgen -f -X '!*.out' -- ${cur}) )
            return 0
            ;;
        -o|--output)
            COMPREPLY=( $(compgen -W "text json html" -- ${cur}) )
            return 0
            ;;
        --strategy)
            COMPREPLY=( $(compgen -W "current aggressive conservative" -- ${cur}) )
            return 0
            ;;
        --style)
            COMPREPLY=( $(compgen -W "flat flat-square" -- ${cur}) )
            return 0
            ;;
        completion)
            COMPREPLY=( $(compgen -W "bash zsh fish" -- ${cur}) )
            return 0
            ;;
        mcp)
            COMPREPLY=( $(compgen -W "serve" -- ${cur}) )
            return 0
            ;;
    esac

    COMPREPLY=( $(compgen -W "-c --config -p --profile -d --domain -o --output -f --force -h --help -q --quiet --no-color --ci --uncovered --diff --merge --show-delta --history --fail-under --ratchet --validate --tags --race --short -v --run --timeout --test-arg" -- ${cur}) )
}
complete -F _coverctl coverctl`

const zshCompletion = `#compdef coverctl

_coverctl() {
    local -a commands
    commands=(
        'check:Run coverage and enforce policy'
        'c:Run coverage and enforce policy (alias)'
        'run:Run coverage only, produce artifacts'
        'r:Run coverage only (alias)'
        'watch:Watch for file changes and re-run coverage'
        'w:Watch for file changes (alias)'
        'init:Interactive setup wizard'
        'i:Interactive setup wizard (alias)'
        'detect:Autodetect domains and write config'
        'report:Analyze an existing profile'
        'badge:Generate an SVG coverage badge'
        'trend:Show coverage trends over time'
        'record:Record current coverage to history'
        'suggest:Suggest optimal coverage thresholds'
        'debt:Show coverage debt report'
        'ignore:Show configured excludes and ignore advice'
        'mcp:MCP server for AI agents'
        'help:Show help for a command'
        'version:Show version information'
        'completion:Generate shell completion scripts'
    )

    _arguments -C \
        '-q[Suppress non-essential output]' \
        '--quiet[Suppress non-essential output]' \
        '--no-color[Disable colored output]' \
        '--ci[CI mode: quiet + GitHub Actions annotations]' \
        '1: :->command' \
        '*: :->args'

    case $state in
        command)
            _describe 'command' commands
            ;;
        args)
            case $words[2] in
                check|c|run|r|watch|w|report|badge|trend|record|suggest|debt|ignore|init|i|detect)
                    _arguments \
                        '-c[Config file path]:file:_files -g "*.yaml"' \
                        '--config[Config file path]:file:_files -g "*.yaml"' \
                        '-p[Coverage profile path]:file:_files -g "*.out"' \
                        '--profile[Coverage profile path]:file:_files -g "*.out"' \
                        '-d[Filter to domain]:domain:' \
                        '--domain[Filter to domain]:domain:' \
                        '-o[Output format]:format:(text json html)' \
                        '--output[Output format]:format:(text json html)' \
                        '-f[Force overwrite]' \
                        '--force[Force overwrite]' \
                        '--uncovered[Show only files with 0% coverage]' \
                        '--diff[Show coverage for changed files]:ref:' \
                        '--merge[Merge additional profile]:file:_files -g "*.out"' \
                        '--show-delta[Show coverage change from previous run]' \
                        '--history[History file path]:file:_files -g "*.json"' \
                        '--fail-under[Fail if coverage below threshold]:percent:' \
                        '--ratchet[Fail if coverage decreases]' \
                        '--validate[Validate config without running tests]' \
                        '--tags[Build tags]:tags:' \
                        '--race[Enable race detector]' \
                        '--short[Skip long-running tests]' \
                        '-v[Verbose test output]' \
                        '--run[Run tests matching pattern]:pattern:' \
                        '--timeout[Test timeout]:duration:' \
                        '--test-arg[Additional test argument]:arg:'
                    ;;
                completion)
                    _arguments '1:shell:(bash zsh fish)'
                    ;;
                mcp)
                    _arguments '1:subcommand:(serve)'
                    ;;
            esac
            ;;
    esac
}

_coverctl "$@"`

const fishCompletion = `# coverctl fish completion
complete -c coverctl -f

# Global flags
complete -c coverctl -s q -l quiet -d "Suppress non-essential output"
complete -c coverctl -l no-color -d "Disable colored output"
complete -c coverctl -l ci -d "CI mode: quiet + GitHub Actions annotations"

# Commands
complete -c coverctl -n "__fish_use_subcommand" -a "check" -d "Run coverage and enforce policy"
complete -c coverctl -n "__fish_use_subcommand" -a "c" -d "Run coverage and enforce policy (alias)"
complete -c coverctl -n "__fish_use_subcommand" -a "run" -d "Run coverage only, produce artifacts"
complete -c coverctl -n "__fish_use_subcommand" -a "r" -d "Run coverage only (alias)"
complete -c coverctl -n "__fish_use_subcommand" -a "watch" -d "Watch for file changes and re-run coverage"
complete -c coverctl -n "__fish_use_subcommand" -a "w" -d "Watch for file changes (alias)"
complete -c coverctl -n "__fish_use_subcommand" -a "init" -d "Interactive setup wizard"
complete -c coverctl -n "__fish_use_subcommand" -a "i" -d "Interactive setup wizard (alias)"
complete -c coverctl -n "__fish_use_subcommand" -a "detect" -d "Autodetect domains and write config"
complete -c coverctl -n "__fish_use_subcommand" -a "report" -d "Analyze an existing profile"
complete -c coverctl -n "__fish_use_subcommand" -a "badge" -d "Generate an SVG coverage badge"
complete -c coverctl -n "__fish_use_subcommand" -a "trend" -d "Show coverage trends over time"
complete -c coverctl -n "__fish_use_subcommand" -a "record" -d "Record current coverage to history"
complete -c coverctl -n "__fish_use_subcommand" -a "suggest" -d "Suggest optimal coverage thresholds"
complete -c coverctl -n "__fish_use_subcommand" -a "debt" -d "Show coverage debt report"
complete -c coverctl -n "__fish_use_subcommand" -a "ignore" -d "Show configured excludes"
complete -c coverctl -n "__fish_use_subcommand" -a "mcp" -d "MCP server for AI agents"
complete -c coverctl -n "__fish_use_subcommand" -a "help" -d "Show help for a command"
complete -c coverctl -n "__fish_use_subcommand" -a "version" -d "Show version information"
complete -c coverctl -n "__fish_use_subcommand" -a "completion" -d "Generate shell completion"

# Flags for all commands
complete -c coverctl -s c -l config -d "Config file path" -r -F
complete -c coverctl -s p -l profile -d "Coverage profile path" -r -F
complete -c coverctl -s d -l domain -d "Filter to specific domain" -r
complete -c coverctl -s o -l output -d "Output format" -r -a "text json html"
complete -c coverctl -s f -l force -d "Force overwrite"
complete -c coverctl -s h -l help -d "Show help"
complete -c coverctl -l uncovered -d "Show only files with 0% coverage"
complete -c coverctl -l diff -d "Show coverage for changed files" -r
complete -c coverctl -l merge -d "Merge additional coverage profile" -r -F
complete -c coverctl -l show-delta -d "Show coverage change from previous run"
complete -c coverctl -l history -d "History file path" -r -F
complete -c coverctl -l fail-under -d "Fail if coverage below threshold" -r
complete -c coverctl -l ratchet -d "Fail if coverage decreases"
complete -c coverctl -l validate -d "Validate config without running tests"
complete -c coverctl -l tags -d "Build tags (e.g., integration,e2e)" -r
complete -c coverctl -l race -d "Enable race detector"
complete -c coverctl -l short -d "Skip long-running tests"
complete -c coverctl -s v -d "Verbose test output"
complete -c coverctl -l run -d "Run tests matching pattern" -r
complete -c coverctl -l timeout -d "Test timeout (e.g., 10m, 1h)" -r
complete -c coverctl -l test-arg -d "Additional argument passed to go test" -r

# Completion subcommand
complete -c coverctl -n "__fish_seen_subcommand_from completion" -a "bash zsh fish"

# MCP subcommand
complete -c coverctl -n "__fish_seen_subcommand_from mcp" -a "serve" -d "Start the MCP server"`
