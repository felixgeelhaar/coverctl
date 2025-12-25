# coverctl

**Domain-aware test coverage enforcement for Go teams**  
Built with strict Domain-Driven Design layers, TDD-first validation, and automated releases powered by Relicta v2.6.1.

![Go](https://img.shields.io/badge/language-Go-00ADD8) ![coverage](https://img.shields.io/badge/coverage-80%25%2B-brightgreen) ![releases](https://img.shields.io/github/v/release/felixgeelhaar/coverctl?label=releases)

## Overview

coverctl wraps `go test` with the right `-covermode`, `-coverpkg`, and domain policy definitions so coverage policy failures surface at the slice level, not just the module level. It autodetects domains, emits human-readable and JSON reports, surfaces warnings when files overlap multiple domains, and keeps builds consistently above 80% coverage.

## Getting started

```bash
go install github.com/felixgeelhaar/coverctl@latest
git clone git@github.com:felixgeelhaar/coverctl.git
cd coverctl
go build ./...
coverctl init                               # runs an interactive Bubble Tea wizard (use --no-interactive for automation)
coverctl detect --dry-run                   # preview detected domains without writing config
coverctl check                              # enforce policy, add -o json for automation
coverctl watch                              # continuous coverage feedback during development
```

## CLI reference

| Command | What it does | Notes |
| --- | --- | --- |
| `coverctl init` | Autodetect domains and launch the Bubble Tea wizard before writing `.coverctl.yaml` | Navigate with ↑/↓, adjust thresholds with ←/→ or +/-, and confirm to persist. Pass `--no-interactive` to skip the UI in scripts. |
| `coverctl detect` | Autodetect domains and write config | Writes config by default; use `--dry-run` to preview without writing. Pass `--force` to overwrite existing config. |
| `coverctl check` | Run coverage, aggregate domains, enforce policy | `-o json` emits machine-readable results; exit code `1` signals policy violations. Use `--show-delta` to display coverage changes. Supports `--fail-under N` and `--ratchet`. |
| `coverctl run` | Produce coverage artifacts without evaluating policy | Use `--profile` to customize output path. |
| `coverctl watch` | Watch for file changes and re-run coverage | Continuous coverage feedback during development. |
| `coverctl report` | Evaluate an already generated profile | Consumes the same config + domains; ideal for CI artifacts or debugging. Supports `-o html`, `--uncovered`, `--diff <ref>`, and `--merge <profile>`. |
| `coverctl ignore` | Show configured `exclude` patterns and the tracked domains | Use this to document generated folders (e.g., `internal/generated/proto/...`) that you wish to skip. |
| `coverctl badge` | Generate an SVG coverage badge | Use `--style flat-square` for a different style. Output to `coverage.svg` by default. |
| `coverctl trend` | Show coverage trends over time | Requires history data recorded via `coverctl record`. |
| `coverctl record` | Record current coverage to history | Use with `--commit` and `--branch` for CI integration. |
| `coverctl suggest` | Suggest optimal coverage thresholds | Strategies: `current`, `aggressive`, `conservative`. Use `--write-config` to apply. |
| `coverctl debt` | Show coverage debt report | Identifies domains/files below target and estimates remediation effort. |
| `coverctl mcp serve` | Start MCP server for AI agents | Enables Claude and other AI agents to interact with coverage tools programmatically via STDIO. |

Text output (the default) shows domain coverage, required thresholds, and statuses. JSON adds warnings for overlap detection and is suitable for dashboards when you pass `-o json`. HTML output (`-o html`) generates a visual report with coverage percentages and status indicators. Use `coverctl ignore` to review the `exclude` list, which is how generated folders such as proto artifacts can be omitted before running `coverctl check`.

## Init wizard

`coverctl init` now launches a short Bubble Tea wizard that reviews the detected domains, lets you adjust coverage minima with arrow keys or +/- shortcuts, and confirms the policy before persisting `.coverctl.yaml`. Use `--no-interactive` when you need to run the command in CI or scripted workflows and you just want to write the autodetected configuration.

## Build/test flags

The `check`, `run`, and `watch` commands support common Go test flags for customizing test execution:

| Flag | Description | Example |
| --- | --- | --- |
| `--tags` | Build tags | `--tags integration,e2e` |
| `--race` | Enable race detector | `--race` |
| `--short` | Skip long-running tests | `--short` |
| `-v` | Verbose test output | `-v` |
| `--run` | Run only tests matching pattern | `--run TestFoo` |
| `--timeout` | Test timeout | `--timeout 30m` |
| `--test-arg` | Additional go test argument (repeatable) | `--test-arg=-count=1` |

Examples:

```bash
# Run integration tests with build tag
coverctl check --tags integration

# Run with race detector and extended timeout
coverctl check --race --timeout 30m

# Run specific tests with verbose output
coverctl run --run TestMyFunction -v

# Pass multiple extra arguments to go test
coverctl check --test-arg=-count=1 --test-arg=-parallel=4
```

## Coverage policy

`coverctl check` parses the profile produced by `go test -coverpkg=…` and assigns statements to the domains defined in `.coverctl.yaml`. Because the raw Go output aggregates every instrumented package—including helpers, generated files, and adapters you may already exclude—the percentage it reports (often ~18.5%) is not used directly. The policy enforces >80% coverage only within the scoped domains (`cmd/coverctl`, `internal/application`, etc.), so staying focused there while keeping generated folders listed in `exclude` prevents quality from falling through the cracks.

## Configuration

The schema lives in `schemas/coverctl.schema.json`. Configs are versioned; set `version: 1` today and keep it in place so future schema upgrades can be detected safely. A policy looks like:

```yaml
version: 1
policy:
  default:
    min: 75
  domains:
    - name: core
      match: ["./internal/core/..."]
      min: 85
    - name: api
      match: ["./internal/api/..."]
exclude:
  - internal/generated/*
```

The autodetect command covers `cmd/`, `pkg/`, and directories inside `internal/`, skipping `generated`/`mocks`.

Advanced options you can enable as needed:

```yaml
version: 1
files:
  - match: ["internal/core/*.go"]
    min: 90
diff:
  enabled: true
  base: origin/main
integration:
  enabled: true
  packages: ["./internal/integration/..."]
  run_args: ["-test.run", "TestIntegration"]
  cover_dir: ".cover/integration"
  profile: ".cover/integration.out"
merge:
  profiles: [".cover/unit.out", ".cover/integration.out"]
annotations:
  enabled: true
```

- `files` enforces per-file minima for any matching paths.
- `diff` enforces coverage only on files changed since the base ref.
- `integration` builds `go test -c` binaries and runs them with `GOCOVERDIR` (Go 1.20+).
- `merge` combines multiple coverprofiles into a single policy evaluation.
- `annotations` enables `// coverctl:ignore` and `// coverctl:domain=NAME` pragmas.

Need a starting point? Copy `templates/coverctl.yaml` and adjust domains and thresholds to fit your repo.

## MCP Server (AI Agent Integration)

coverctl includes a Model Context Protocol (MCP) server that enables AI agents like Claude to interact with coverage tools programmatically. Start it with:

```bash
coverctl mcp serve
```

### Available Tools

| Tool | Description |
| --- | --- |
| `check` | Run coverage tests and enforce policy thresholds |
| `report` | Analyze an existing coverage profile |
| `record` | Record current coverage to history |

### Available Resources

| URI | Description |
| --- | --- |
| `coverctl://debt` | Coverage debt metrics |
| `coverctl://trend` | Coverage trends over time |
| `coverctl://suggest` | Threshold recommendations |
| `coverctl://config` | Current configuration |

### Claude Desktop Configuration

Add to `~/.config/claude/claude_desktop_config.json` (macOS/Linux) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "coverctl": {
      "command": "coverctl",
      "args": ["mcp", "serve"],
      "cwd": "/path/to/your/go/project"
    }
  }
}
```

Once configured, Claude can run coverage checks, analyze reports, and provide recommendations based on your project's coverage data.

## Architecture

- `internal/domain`: coverage stats, policy evaluation, warning aggregation.
- `internal/application`: services (`check`, `run`, `report`, `detect`) orchestrate config loading, domain resolution, coverage runs, and reporting.
- `internal/infrastructure`: adapters for config files, Go tooling (`go test`), profile parsing, reporters, and autodetection.
- `internal/cli`: CLI parsing, output, and wiring for the root command entrypoint.
- `main.go`: root entrypoint so `go install github.com/felixgeelhaar/coverctl@latest` works. (The `cmd/coverctl` wrapper remains for compatibility.)

## Testing & contribution guidelines

- Practice TDD: add or update tests before implementing behavior changes.
- Keep test coverage ≥80% using `go test ./... -cover`.
- Follow Conventional Commits (`feat:`, `fix:`, etc.) for Relicta’s version bump logic.
- `main` is protected: merge via PRs after CI passes (see `.github/workflows/go.yml`). This keeps the release workflow deterministic and reviewable.
- Pull requests that touch reporting should document CLI output or sample JSON to keep churn visible.

## Releases

- Relicta v2.6.1 (`relicta.config.yaml`) drives releases: it creates semver tags, updates `CHANGELOG.md`, and publishes GitHub releases.
- `.github/workflows/release.yml` triggers `relicta release --yes` on protected `main` (after each merge). The Relicta `pre_release_hook` runs `scripts/build-artifacts.sh` to compile Linux/macOS/Windows CLI tarballs/zips that the GitHub plugin attaches as release assets.
- Configure a secret named `RELICTA_TOKEN` (or rely on `${{ secrets.GITHUB_TOKEN }}`) with contents/workflows/packages permissions so Relicta can push tags, update the changelog, and attach artifacts.
- **Do not manually push `v*` tags**; let Relicta own the tag lifecycle to avoid conflicts.

## GitHub Action

Use the built-in composite action to run coverctl in CI:

```yaml
jobs:
  coverage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with:
          go-version: "1.24"
      - uses: ./.github/actions/coverctl
        with:
          command: check
          config: .coverctl.yaml
          output: text
```

## Docs & governance

- Product/architecture docs live under `docs/` (TDD/PRD). `AGENTS.md` covers contributor expectations.
- The `scripts/build-artifacts.sh` helper compiles cross-platform binaries for release artifacts.
- Keep `CHANGELOG.md` tracked so Relicta can automatically append release notes during `relicta release`.
