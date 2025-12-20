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
coverctl init --config .coverctl.yaml       # runs an interactive Bubble Tea wizard (use --no-interactive for automation)
coverctl detect --write-config              # inspect/domain output without writing
coverctl check --config .coverctl.yaml      # enforce policy, add -o json for automation
```

## CLI reference

| Command | What it does | Notes |
| --- | --- | --- |
| `coverctl init` | Autodetect domains and launch the Bubble Tea wizard before writing `.coverctl.yaml` | Navigate with ↑/↓, adjust thresholds with ←/→ or +/-, and confirm to persist. Pass `--no-interactive` to skip the UI in scripts. |
| `coverctl detect` | Preview domain policy and optionally write config | Pass `--write-config`/`--force` to persist identical config; omit to see the policy before writing. |
| `coverctl check` | Run coverage, aggregate domains, enforce policy | `-o json` emits machine-readable results; exit code `1` signals policy violations. |
| `coverctl run` | Produce coverage artifacts without evaluating policy | Use `--profile` to customize output path. |
| `coverctl report` | Evaluate an already generated profile | Consumes the same config + domains; ideal for CI artifacts or debugging. |
| `coverctl ignore` | Show configured `exclude` patterns and the tracked domains | Use this to document generated folders (e.g., `internal/generated/proto/...`) that you wish to skip. |

Text output (the default) shows domain coverage, required thresholds, and statuses. JSON adds warnings for overlap detection and is suitable for dashboards when you pass `-o json`. Use `coverctl ignore` to review the `exclude` list, which is how generated folders such as proto artifacts can be omitted before running `coverctl check`.

## Init wizard

`coverctl init` now launches a short Bubble Tea wizard that reviews the detected domains, lets you adjust coverage minima with arrow keys or +/- shortcuts, and confirms the policy before persisting `.coverctl.yaml`. Use `--no-interactive` when you need to run the command in CI or scripted workflows and you just want to write the autodetected configuration.

## Coverage policy

`coverctl check` parses the profile produced by `go test -coverpkg=…` and assigns statements to the domains defined in `.coverctl.yaml`. Because the raw Go output aggregates every instrumented package—including helpers, generated files, and adapters you may already exclude—the percentage it reports (often ~18.5%) is not used directly. The policy enforces >80% coverage only within the scoped domains (`cmd/coverctl`, `internal/application`, etc.), so staying focused there while keeping generated folders listed in `exclude` prevents quality from falling through the cracks.

## Configuration

The schema lives in `schemas/coverctl.schema.json`. A policy looks like:

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

## Docs & governance

- Product/architecture docs live under `docs/` (TDD/PRD). `AGENTS.md` covers contributor expectations.
- The `scripts/build-artifacts.sh` helper compiles cross-platform binaries for release artifacts.
- Keep `CHANGELOG.md` tracked so Relicta can automatically append release notes during `relicta release`.
