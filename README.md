# coverctl

Declarative, domain-steered coverage validation for Go. The tool runs `go test` with `-covermode=atomic`, resolves coverage domains, and enforces policy thresholds per the PRD/TDD guidance.

## Getting started
1. `go build ./...` builds the CLI (`cmd/coverctl`).
2. Write a `.coverctl.yaml` or run `coverctl detect --write-config` to scaffold domains and policy.
3. Run `coverctl check --config .coverctl.yaml` in CI to enforce coverage policy.

## CLI commands
- `coverctl check` (default output: text; use `-o json` for machine-readable).
- `coverctl run` only produces coverage artifacts (`-profile` overrides `.cover/coverage.out`).
- `coverctl detect` infers domains from `cmd/`, `internal/`, `pkg/`; add `--write-config` to persist `.coverctl.yaml`.
- `coverctl report --profile .cover/coverage.out` evaluates an existing profile without rerunning tests.

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
Use `schemas/coverctl.schema.json` for validation.

## Modeling & development norms
- Follow strict DDD: `domain/` (entities/policy), `application/` (use cases), `infrastructure/` (config, CLI, Go tooling).
- Practice TDD: add or update tests before implementation; prefer table-driven coverage for policy/aggregation logic.
- Keep repo coverage > 80% on `go test ./... -cover`.

## Testing
- `go test ./...` (unit tests).
- `go test ./... -cover` confirms coverage budget.

## Contribution
Adhere to Conventional Commits (e.g., `feat: add autodetect report`). Provide PR descriptions, linked issues, and sample CLI output for changes affecting reports or policy behavior.
