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
   - `coverctl detect --write-config` writes `.coverctl.yaml` (autodetect inspects `cmd/`, `internal/`, `pkg/`).
   - Manually edit `.coverctl.yaml` and validate it with `schemas/coverctl.schema.json`.
3. Run `coverctl check --config .coverctl.yaml` in CI to enforce coverage policies. Use `-o json` for scripts and tooling integrations.

## Commands & expectations

| Command | Description | Notes |
| --- | --- | --- |
| `coverctl check` | Run coverage, aggregate by domain, evaluate policy | Exits `0` when all domains pass, `1` when policy fails; `-o json` emits structured output including warnings. |
| `coverctl run` | Run coverage only and emit profile | Useful for pairing with `report` or storing artifacts; use `--profile` to override `.cover/coverage.out`. |
| `coverctl detect [--write-config]` | Autodetect domains and optionally persist config | Does not modify source unless `--write-config` is provided (CLI implements `detect` rather than a dedicated `init`). |
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

- Releases are automated with [Relicta](https://github.com/relicta-tech/relicta) via `relicta.config.yaml`.
- `.github/workflows/release.yml` builds Linux/macOS/Windows binaries (`dist/coverctl-*-amd64.*`), runs `relicta release --yes`, and publishes the artifacts through the GitHub plugin.
- The workflow expects a `RELICTA_TOKEN` secret (contents: write, workflows: write, packages: write) or the default `GITHUB_TOKEN` so Relicta can tag, write `CHANGELOG.md`, and upload assets.
