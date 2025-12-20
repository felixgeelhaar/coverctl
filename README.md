# coverctl

**Declarative, domain-aware coverage enforcement for Go teams.**

![Go](https://img.shields.io/badge/language-Go-00ADD8) ![coverage](https://img.shields.io/badge/coverage-80%25%2B-brightgreen) ![build](https://img.shields.io/github/actions/workflow/status/felixgeelhaar/coverctl/go.yml?branch=main&label=ci&logo=github)

coverctl wraps `go test` with `-covermode=atomic`, groups packages into configurable domains, and fails CI when a domain’s coverage drops below policy. It ships with strict DDD layers, TDD guidance, JSON/text output, and an autodetect flow so teams can guard architectural boundaries.

## Quick start
1. `go build ./...` to build `cmd/coverctl`.
2. Run `coverctl detect --write-config` to scaffold `.coverctl.yaml` (or hand-edit per `schemas/coverctl.schema.json`).
3. Run `coverctl check --config .coverctl.yaml` in CI; use `-o json` when you need machine-readable reports.

## CLI commands
- `coverctl check` (defaults to text, `-o json` for structured output).
- `coverctl run` only generates coverage artifacts (`--profile` overrides `.cover/coverage.out`).
- `coverctl detect` infers domains (`cmd/`, `internal/`, `pkg/`, with autodetected subdomains under `internal/`).
- `coverctl report --profile .cover/coverage.out` evaluates an existing profile without rerunning tests.

## Configuration sample
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

Use `schemas/coverctl.schema.json` to validate authoring. Autodetect writes a similar policy with defaults tuned to directories that exist.

## Repository conventions
- **Modeling**: strict DDD split between `internal/domain`, `internal/application`, `internal/infrastructure`.
- **Development**: TDD-first. Add tests before behaviors, keep coverage > 80% (`go test ./... -cover`).
- **Review**: Conventional commits (e.g., `feat: add autodetect report`); PRs should describe behavior changes and include CLI output samples for coverage-reporting features.
- **Support**: Issue templates live under `.github/ISSUE_TEMPLATE`; CI runs via `.github/workflows/go.yml`.

## Testing
- `go test ./...`
- `go test ./... -cover`

## Tags
Suggested repository topics: `go`, `coverage`, `domain-driven-design`, `tdd`, `cli`.
