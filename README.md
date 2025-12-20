# coverctl

**Domain-aware Go coverage enforcement via CLI.**

![Go](https://img.shields.io/badge/language-Go-00ADD8) ![coverage](https://img.shields.io/badge/coverage-80%25%2B-brightgreen) ![build](https://img.shields.io/github/actions/workflow/status/felixgeelhaar/coverctl/go.yml?branch=main&label=ci&logo=github)

## What it solves

Go’s native coverage reporting lacks policy enforcement and architectural context. `coverctl` addresses that by:

- modeling logical **domains** (packages or folder bundles) with per-domain minimums,
- running `go test` with `-covermode=atomic` and the configured `-coverpkg` set,
- aggregating coverage per domain, applying policy, and failing early when thresholds break,
- supporting both human-readable reports and machine-readable JSON output,
- providing `detect` to infer domains and reduce onboarding effort.

## Architecture & usage

The CLI sits in `cmd/coverctl`. Business rules live in `internal/domain`, orchestration lives in `internal/application`, and adapters (config, Go tooling, reporters, autodetect) live under `internal/infrastructure`. This strict DDD split keeps domain invariants central and makes TDD easy.

1. Build: `go build ./...` (CLI appears at `cmd/coverctl`).
2. Configure: `coverctl detect --write-config` or edit `.coverctl.yaml` manually (schema in `schemas/coverctl.schema.json`).
3. Enforce: `coverctl check --config .coverctl.yaml` (CI should fail when policy breaks; use `-o json` for automation).

## CLI reference

- `coverctl check`: run coverage, aggregate domains, evaluate policy; default output is text, `-o json` emits machine-readable payloads and warnings.
- `coverctl run`: rerun coverage only and produce the profile artifact (`--profile` overrides `.cover/coverage.out`).
- `coverctl detect`: infer domains from `cmd/`, `internal/`, `pkg/` and, optionally, persist `.coverctl.yaml`.
- `coverctl report --profile <path>`: evaluate an existing coverage profile without running tests (useful for CI artifacts).

## Configuration example

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

Use `schemas/coverctl.schema.json` to validate config files and ensure domain definitions match actual packages.

## Testing & TDD

- Unit tests rely on Go’s `testing` package with table-driven assertions for domain aggregation and policy evaluation.
- Keep coverage above 80%: `go test ./... -cover`.
- Prefer writing tests first (TDD) for new domain logic so invariants remain centered in the domain layer.

## Releases

- Releases use [Relicta](https://github.com/relicta-tech/relicta) with `relicta.config.yaml`, so tagging, changelog updates, and GitHub release publishing are automated.
- `.github/workflows/release.yml` builds CLI binaries (`dist/*.tar.gz`, `.zip`), runs `relicta release --yes`, and publishes the binaries via the GitHub plugin.
- The release workflow expects a `RELICTA_TOKEN` secret (fine-grained token with contents/write, workflows/write, packages/write; the default `GITHUB_TOKEN` works too) so Relicta can bump versions, push tags, update `CHANGELOG.md`, and attach assets.
