package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/autodetect"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/config"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/coverprofile"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/gotool"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/report"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/wizard"
)

func main() {
	code := run(os.Args, os.Stdout, os.Stderr, buildService(os.Stdout))
	os.Exit(code)
}

type service interface {
	Check(ctx context.Context, opts application.CheckOptions) error
	RunOnly(ctx context.Context, opts application.RunOnlyOptions) error
	Detect(ctx context.Context, opts application.DetectOptions) (application.Config, error)
	Report(ctx context.Context, opts application.ReportOptions) error
	Ignore(ctx context.Context, opts application.IgnoreOptions) (application.Config, []domain.Domain, error)
}

var initWizard = wizard.Run

func run(args []string, stdout, stderr io.Writer, svc service) int {
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
		_ = fs.Parse(args[2:])
		err := svc.Check(ctx, application.CheckOptions{ConfigPath: *configPath, Output: *output, Profile: *profile})
		return exitCode(err, 1, stderr)
	case "run":
		fs := flag.NewFlagSet("run", flag.ExitOnError)
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		profile := fs.String("profile", "", "Coverage profile output path")
		_ = fs.Parse(args[2:])
		err := svc.RunOnly(ctx, application.RunOnlyOptions{ConfigPath: *configPath, Profile: *profile})
		return exitCode(err, 3, stderr)
	case "detect":
		fs := flag.NewFlagSet("detect", flag.ExitOnError)
		writeConfig := fs.Bool("write-config", false, "Write detected config to .coverctl.yaml")
		configPath := fs.String("config", ".coverctl.yaml", "Config file path")
		force := fs.Bool("force", false, "Overwrite config if it exists")
		_ = fs.Parse(args[2:])
		cfg, err := svc.Detect(ctx, application.DetectOptions{WriteConfig: *writeConfig, ConfigPath: *configPath})
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
		_ = fs.Parse(args[2:])
		err := svc.Report(ctx, application.ReportOptions{ConfigPath: *configPath, Output: *output, Profile: *profile})
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
		cfg, err := svc.Detect(ctx, application.DetectOptions{WriteConfig: true, ConfigPath: *configPath})
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
	default:
		usage(stderr)
		return 2
	}
}

func buildService(out *os.File) *application.Service {
	module := gotool.ModuleResolver{}
	return &application.Service{
		ConfigLoader:   config.Loader{},
		Autodetector:   autodetect.Detector{Module: module},
		DomainResolver: gotool.DomainResolver{Module: module},
		CoverageRunner: gotool.Runner{Module: module},
		ProfileParser:  coverprofile.Parser{},
		Reporter:       report.Writer{},
		Out:            out,
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
  report  Analyze an existing profile`)
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
