# coverctl

**Domain-aware test coverage enforcement for Go teams**  
Built with strict Domain-Driven Design layers, TDD-first validation, and automated releases powered by Relicta v2.6.1.

![Go](https://img.shields.io/badge/language-Go-00ADD8) ![coverage](https://img.shields.io/badge/coverage-80%25%2B-brightgreen) ![releases](https://img.shields.io/github/v/release/felixgeelhaar/coverctl?label=releases)

## Overview

coverctl wraps `go test` with the right `-covermode`, `-coverpkg`, and domain policy definitions so coverage policy failures surface at the slice level, not just the module level. It autodetects domains, emits human-readable and JSON reports, surfaces warnings when files overlap multiple domains, and keeps builds consistently above 80% coverage.

## Getting started

```bash
git clone git@github.com:felixgeelhaar/coverctl.git
cd coverctl
go build ./...
coverctl init --config .coverctl.yaml       # creates policy, use --force to overwrite
covercil detect --write-config              # inspect/domain output without writing
coverctl check --config .coverctl.yaml      # enforce policy, add -o json for automation
```

## CLI reference

| Command | What it does | Notes |
| --- | --- | --- |
| `coverctl init` | Autodetect domains, write `.coverctl.yaml`, and ask for overrides | Always persists config; use `--force` to overwrite safely. |
| `coverctl detect` | Preview domain policy and optionally write config | Pass `--write-config`/`--force` to persist identical config; omit to see the policy before writing. |
| `coverctl check` | Run coverage, aggregate domains, enforce policy | `-o json` emits machine-readable results; exit code `1` signals policy violations. |
| `coverctl run` | Produce coverage artifacts without evaluating policy | Use `--profile` to customize output path. |
| `coverctl report` | Evaluate an already generated profile | Consumes the same config + domains; ideal for CI artifacts or debugging. |

Text output shows domain coverage, required thresholds, and statuses. JSON adds warnings for overlap detection and is suitable for dashboards.

## Configuration

The schema lives in `schemas/coverctl.schema.json`. A policy looks like:

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

The autodetect command covers `cmd/`, `pkg/`, and directories inside `internal/`, skipping `generated`/`mocks`.

## Architecture

- `internal/domain`: coverage stats, policy evaluation, warning aggregation.
- `internal/application`: services (`check`, `run`, `report`, `detect`) orchestrate config loading, domain resolution, coverage runs, and reporting.
- `internal/infrastructure`: adapters for config files, Go tooling (`go test`), profile parsing, reporters, and autodetection.
- `cmd/coverctl`: CLI glue keeping dependencies flowing inward (DDD) and enabling TDD-friendly testing.

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
