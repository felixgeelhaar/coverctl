# coverctl

**Domain-aware coverage enforcement for Go, built on strict DDD.**

![Go](https://img.shields.io/badge/language-Go-00ADD8) ![coverage](https://img.shields.io/badge/coverage-80%25%2B-brightgreen) ![build](https://img.shields.io/github/actions/workflow/status/felixgeelhaar/coverctl/go.yml?branch=main&label=ci&logo=github)

## Why coverctl exists

Go’s built-in coverage tooling reports global percentages but never enforces them, and it lacks architectural context. coverctl solves that by:

- defining **domains** (logical package groups) with configurable minimum coverage;
- running `go test -covermode=atomic` with the proper `-coverpkg` bindings;
- aggregating coverage per domain and failing CI if any domain or the default policy drops below its threshold;
- offering text and JSON reports plus safeguards (warnings when files belong to multiple domains).

## Project structure

- `cmd/coverctl`: CLI entry point.
- `internal/domain`: entities/value objects for policies, coverage stats, and evaluation results.
- `internal/application`: use cases (check, run-only, report, detect) orchestrating services.
- `internal/infrastructure`: adapters for config, Go tooling, coverage parsing, reporting, and autodetection.
- `schemas/coverctl.schema.json`: config schema referenced by docs and tooling.

## Getting started

1. `go build ./...` — produces the CLI under `cmd/coverctl`.
2. Generate or adjust config:
   - `coverctl init --config .coverctl.yaml` (default) autodetects domains and writes `.coverctl.yaml`; use `--force` to overwrite.
   - `coverctl detect` runs the same detection but only writes when `--write-config` is passed (prints to stdout otherwise); useful for inspection.
   - Manually edit `.coverctl.yaml` and validate it with `schemas/coverctl.schema.json`.
3. Run `coverctl check --config .coverctl.yaml` in CI to enforce coverage policies. Use `-o json` for scripts and tooling integrations.

## Commands & expectations

| Command | Description | Notes |
| --- | --- | --- |
| `coverctl check` | Run coverage, aggregate by domain, evaluate policy | Exits `0` when all domains pass, `1` when policy fails; `-o json` emits structured output including warnings. |
| `coverctl run` | Run coverage only and emit profile | Useful for pairing with `report` or storing artifacts; use `--profile` to override `.cover/coverage.out`. |
| `coverctl init` | Autodetect domains and write `.coverctl.yaml` (with `--config`/`--force`) | Recommended way to bootrap the repo; it always writes config, unlike `detect`. |
| `coverctl detect [--write-config]` | Inspect autodetected domains and optionally print/write config | Use `--write-config` (plus `--force`) to persist same result as `init`; omit it to preview the inferred policy. |
| `coverctl report --profile <path>` | Evaluate an existing coverage profile | Designed for CI job artifact analysis; reuses config and domain resolution without rerunning tests. |

### Outputs & warnings

- Text output prints a table of domain coverage plus optional warnings when directories belong to multiple domains.
- JSON output mirrors the text report and adds warning strings so automation can surface overlaps.
- Expect coverage artifacts in `.cover/coverage.out` (configurable via `--profile`).

## Sample config

```yaml
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

Autodetect writes a similar structure when you run `coverctl detect --write-config`. Use `schemas/coverctl.schema.json` for validation.

## Testing philosophy

- Follow TDD: write or update tests before implementation work.
- Use Go’s `testing` package with table-driven checks for policy and domain logic.
- Keep test coverage above 80%: `go test ./... -cover` is the gate for CI.

## Development & contributions

- The repo follows DDD: keep domain invariants inside `internal/domain`, orchestrate via `internal/application`, and wire up infrastructure without leaking dependencies upstream.
- Follow Conventional Commits (`feat:`, `fix:`, etc.) so Relicta can auto-bump versions.
- CI runs the Go test suite (`.github/workflows/go.yml`).
- Add PR descriptions that include CLI output samples when reports change.

## Releases

- Releases are automated with [Relicta (v2.6.1)](https://github.com/relicta-tech/relicta) via `relicta.config.yaml`.
- `.github/workflows/release.yml` sets up Go, runs Relicta, and `pre_release_hook` (`scripts/build-artifacts.sh`) compiles Linux/macOS/Windows tarballs/zips that the GitHub plugin attaches as release assets.
- The workflow prefers a `RELICTA_TOKEN` secret (contents: write, workflows: write, packages: write) but falls back to `GITHUB_TOKEN` so Relicta can tag, write `CHANGELOG.md`, and upload binaries.
